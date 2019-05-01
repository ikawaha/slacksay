package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/ikawaha/slacksay"
)

const (
	commandName  = "slacksay"
	usageMessage = "%s -t <slack_token> [-d (<json data>|@<file_name>|@-)]"
	maxRetry     = 3
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
	if err := backoff.RetryNotify(
		func() error { return loop(opt.token, config) },
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetry),
		func(err error, d time.Duration) { log.Printf("backoff after %v, %v", d, err) },
	); err != nil {
		log.Printf("%v", err)
		return
	}
	return
}

func loop(token string, config *slacksay.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bot, err := slacksay.NewBot(ctx, token, config)
	if err != nil {
		return fmt.Errorf("configureation error, %v, %v", err, time.Now())
	}
	defer bot.Close()
	log.Printf("%+v\n", config.String())
	log.Printf("^C exits")
	for {
		msg, err := bot.GetMessage(ctx)
		if err != nil {
			return fmt.Errorf("receive error, %v", err)
		}
		log.Printf("bot_id: %v, msg_user_id: %v, msg:%+v\n", bot.ID, msg.UserID, msg)
		go bot.Response(&msg)
	}
}
