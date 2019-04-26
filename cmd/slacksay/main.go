package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/cenkalti/backoff"
	"github.com/ikawaha/slacksay"
)

const (
	commandName  = "slacksay"
	usageMessage = "%s -t <slack_token> [-d (<json data>|@<file_name>|@-)]"
)

// Usage provides information on the use of the server
func Usage() {
	log.Printf(usageMessage, commandName)
}

func main() {
	if len(os.Args) < 2 {
		Usage()
		printOptionDefaults(flag.ExitOnError)
		return
	}
	opt := newOption(flag.ExitOnError)
	if err := opt.parse(os.Args[1:]); err != nil {
		Usage()
		printOptionDefaults(flag.ExitOnError)
		log.Printf("%v, %v", commandName, err)
		return
	}
	config, err := opt.newConfig()
	if err != nil {
		log.Printf("configureation error, %v", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	bot, err := slacksay.NewBot(ctx, opt.token, config)
	if err != nil {
		log.Printf("configureation error, %v", err)
		return
	}
	defer bot.Close()
	log.Printf("%+v\n", config.String())
	log.Printf("^C exits")

	for {
		msg, err := bot.GetMessage(ctx)
		if err != nil {
			log.Printf("receive error, %v", err)
			cancel()
			bot.Close()
			if err := backoff.Retry(func() error {
				ctx, cancel = context.WithCancel(context.Background())
				if bot, err = slacksay.NewBot(ctx, opt.token, config); err != nil { // reboot
					return err
				}
				log.Printf("reboot")
				return nil
			}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3)); err != nil {
				log.Fatalf("backoff failed, %v", err)
			}
			continue
		}
		log.Printf("bot_id: %v, msg_user_id: %v, msg:%+v\n", bot.ID, msg.UserID, msg)
		if msg.Type != "message" && len(msg.Text) == 0 {
			continue
		}
		go bot.Response(&msg)
	}
}
