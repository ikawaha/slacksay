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
	sayCommand        = "say"
	sayCommandTimeout = 1 * time.Minute
	messageQueueSize  = 128
	speakerQueueSize  = 128
	botMessageSubType = "bot_message"
)

// Bot represents slack bot client.
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

// NewBot returns a client.
func NewBot(ctx context.Context, token string, cfg *Config) (*Bot, error) {
	ret := Bot{
		config: cfg,
		queue:  make(chan *slackbot.Message, messageQueueSize),
	}
	c, err := slackbot.New(token)
	if err != nil {
		return nil, err
	}
	ret.Client = c
	ret.command = sayCommand
	if cfg.Command != "" {
		ret.command = cfg.Command
	}
	ret.timeout = sayCommandTimeout
	if cfg.Timeout != "" {
		d, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format, %v", err)
		}
		ret.timeout = d
	}
	if ret.channelYomi, err = cfg.Channel.newYomiReplacer(); err != nil {
		return nil, fmt.Errorf("invalid channel yomi format")
	}
	if ret.userYomi, err = cfg.User.newYomiReplacer(); err != nil {
		return nil, fmt.Errorf("invalid user yomi format")
	}
	if ret.keywordYomi, err = cfg.Keyword.newYomiReplacer(); err != nil {
		return nil, fmt.Errorf("invalid keyword yomi format")
	}

	go ret.workerListener(ctx)

	return &ret, nil
}

// Close closes client connections.
func (bot Bot) Close() {
	close(bot.queue)
	if err := bot.Client.Close(); err != nil {
		log.Printf("bot close, %v", err)
	}
}

func (bot Bot) filter(msg *slackbot.Message) (ok bool) {
	if bot.config.Channel.isNotified(bot.Channels[msg.Channel]) {
		return true
	}
	if bot.config.User.isNotified(bot.Users[msg.UserID]) {
		return true
	}
	if bot.config.Keyword.isNotified(msg.Text) {
		return true
	}
	if !bot.config.BotMessage && msg.SubType == botMessageSubType {
		return false
	}
	if bot.config.Channel.isMute(bot.Channels[msg.Channel]) {
		return false
	}
	if bot.config.User.isMute(bot.Users[msg.UserID]) {
		return false
	}
	if bot.config.Keyword.isMute(msg.Text) {
		return false
	}
	return true
}

// Response processes a slack message.
func (bot Bot) Response(msg *slackbot.Message) {
	if msg.Text == "" {
		return
	}
	msg.Text = bot.ToPlainMessageText(msg.Text)
	msg.Text = strings.ToLower(msg.Text)
	if !bot.filter(msg) {
		return
	}
	bot.queue <- msg
}

func (bot Bot) workerListener(ctx context.Context) {
	q := make(chan string, speakerQueueSize)
	go bot.workerSpeaker(ctx, q)

	for {
		select {
		case msg, ok := <-bot.queue:
			if !ok {
				return
			}
			txt := bot.keywordYomi.Replace(msg.Text)
			slackChannel, ok := bot.Channels[msg.Channel]
			if !ok {
				slackChannel = bot.Groups[msg.Channel]
			}
			slackChannel = bot.channelYomi.Replace(slackChannel)
			if slackChannel == "" {
				slackChannel = "不明"
			}
			user := bot.userYomi.Replace(bot.Users[msg.UserID])
			if user == "" {
				user = "不明"
			}
			q <- fmt.Sprintf("%v。発言者%v。チャンネル%v", txt, user, slackChannel)
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
			var m sync.Mutex
			m.Lock()
			if err := bot.say(ctx, &m, msg); err != nil {
				log.Println("worker speaker, ", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (bot Bot) say(ctx context.Context, m *sync.Mutex, s string) error {
	defer m.Unlock()
	if _, err := exec.LookPath(bot.command); err != nil {
		return fmt.Errorf("command %v is not installed in your $PATH", bot.command)
	}
	r0, w0 := io.Pipe()
	ctx, cancel := context.WithTimeout(ctx, bot.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bot.command)
	cmd.Stdin = r0
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := io.Copy(w0, bytes.NewBufferString(s)); err != nil {
		return err
	}
	if err := w0.Close(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("process done with error, %v", err)
	}
	return nil
}
