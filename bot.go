package slacksay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ikawaha/slackbot"
)

var replacer *strings.Replacer

const (
	SayCommand               = "say"
	DefaultSayCommandTimeout = 1 * time.Minute
	ListenWait               = 1 * time.Second
	MessageQueueSize         = 128
	SpeakerQueueSize         = 128
	BotMessageSubType        = "bot_message"
)

type Bot struct {
	*slackbot.Client
	config  *Config
	command string
	timeout time.Duration

	channelYomi *strings.Replacer
	userYomi    *strings.Replacer
	keywordYomi *strings.Replacer

	queue chan *slackbot.Message
}

func NewBot(ctx context.Context, token string, cfg *Config) (*Bot, error) {
	ret := Bot{
		config: cfg,
		queue:  make(chan *slackbot.Message, MessageQueueSize),
	}
	c, err := slackbot.New(token)
	if err != nil {
		return nil, err
	}
	ret.Client = c
	ret.command = SayCommand
	if cfg.Command != "" {
		ret.command = cfg.Command
	}
	ret.timeout = DefaultSayCommandTimeout
	if cfg.Timeout != "" {
		d, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format, %v", err)
		}
		ret.timeout = d
	}
	if ret.channelYomi, err = cfg.Channel.NewYomiReplacer(); err != nil {
		return nil, fmt.Errorf("invalid channel yomi format")
	}
	if ret.userYomi, err = cfg.User.NewYomiReplacer(); err != nil {
		return nil, fmt.Errorf("invalid user yomi format")
	}
	if ret.keywordYomi, err = cfg.Keyword.NewYomiReplacer(); err != nil {
		return nil, fmt.Errorf("invalid keyword yomi format")
	}

	go ret.workerListener(ctx)

	return &ret, nil
}

func (bot Bot) Close() {
	if err := bot.Client.Close(); err != nil {
		log.Printf("bot close, %v", err)
	}
	close(bot.queue)
}

func (bot Bot) Filter(msg *slackbot.Message) (ok bool) {
	if bot.config.Channel.IsNotified(bot.Channels[msg.Channel]) {
		return true
	}
	if bot.config.User.IsNotified(bot.Users[msg.UserID]) {
		return true
	}
	if bot.config.Keyword.IsNotified(bot.Users[msg.UserID]) {
		return true
	}
	if !bot.config.BotMessage && msg.SubType == BotMessageSubType {
		return false
	}
	if bot.config.Channel.IsMute(bot.Channels[msg.Channel]) {
		return false
	}
	if bot.config.User.IsMute(bot.Users[msg.UserID]) {
		return false
	}
	if bot.config.Keyword.IsMute(bot.Users[msg.UserID]) {
		return false
	}
	return true
}

func (bot Bot) Response(msg *slackbot.Message) {
	if !bot.Filter(msg) {
		return
	}
	bot.queue <- msg
}

func (bot Bot) workerListener(ctx context.Context) {
	q := make(chan string, SpeakerQueueSize)
	go bot.workerSpeaker(ctx, q)

	talks := map[string][]string{}
	for {
		select {
		case msg, ok := <-bot.queue:
			if !ok {
				return
			}
			txt := strings.ToLower(msg.Text)
			if len(txt) == 0 {
				continue
			}
			txt = bot.keywordYomi.Replace(txt)
			slackChannel := bot.channelYomi.Replace(bot.Channels[msg.Channel])
			if slackChannel == "" {
				slackChannel = "不明"
			}
			user := bot.userYomi.Replace(bot.Users[msg.UserID])
			if user == "" {
				user = "不明"
			}
			txt = fmt.Sprintf("%v。発言者%v。チャンネル%v", txt, user, slackChannel)
			talks[msg.Channel] = append(talks[msg.Channel], txt)
			log.Printf("%v>>> %v\n", bot.command, txt)
		case <-time.After(ListenWait):
			for _, v := range talks {
				q <- strings.Join(v, "。")
			}
			talks = map[string][]string{}
		case <-ctx.Done():
			return
		}
	}

}

func (bot Bot) workerSpeaker(ctx context.Context, q <-chan string) {
	for {
		select {
		case msg, ok := <-q:
			if !ok {
				return
			}
			var wg sync.WaitGroup
			wg.Add(1)
			if err := bot.say(ctx, &wg, msg); err != nil {
				log.Println("worker speaker, ", err)
			}
			wg.Wait()
		case <-ctx.Done():
			return
		}
	}
}

func (bot Bot) say(ctx context.Context, wg *sync.WaitGroup, s string) error {
	defer wg.Done()
	if _, err := exec.LookPath(bot.command); err != nil {
		return fmt.Errorf("command %v is not installed in your $PATH", bot.command)
	}
	cmd := exec.Command(bot.command)
	r0, w0 := io.Pipe()
	cmd.Stdin = r0
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("process done with error = %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	if _, err := io.Copy(w0, bytes.NewBufferString(s)); err != nil {
		done <- err
	}
	if err := w0.Close(); err != nil {
		done <- err
	}

	select {
	case <-time.After(bot.timeout):
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill, %v", err)
		}
		<-done
		return fmt.Errorf("%v command timeout", bot.command)
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill, %v", err)
		}
		<-done
		return fmt.Errorf("%v context done", bot.command)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("process done with error, %v", err)
		}
	}
	return nil
}
