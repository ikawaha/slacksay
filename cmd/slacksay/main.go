package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ikawaha/slacksay"
)

const (
	CommandName  = "slacksay"
	usageMessage = "%s -t <slack_token> [-d (<json data>|@<file_name>|@-)]"
)

// Usage provides information on the use of the server
func Usage() {
	log.Printf(usageMessage, CommandName)
}

func main() {
	if len(os.Args) < 2 {
		Usage()
		PrintOptionDefaults(flag.ExitOnError)
		return
	}
	opt := newOption(flag.ExitOnError)
	if err := opt.parse(os.Args[1:]); err != nil {
		Usage()
		PrintOptionDefaults(flag.ExitOnError)
		fmt.Fprintf(os.Stderr, "%v, %v", CommandName, err)
		return
	}
	config, err := opt.NewConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configureation error, %v", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	bot, err := slacksay.NewBot(ctx, opt.token, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configureation error, %v", err)
		return
	}
	defer bot.Close()
	fmt.Fprintf(os.Stderr, "%+v\n", config.String())
	fmt.Fprintln(os.Stderr, "^C exits")

	for {
		msg, err := bot.GetMessage()
		if err != nil {
			log.Printf("receive error, %v", err)
			cancel()
			bot.Close()
			ctx, cancel = context.WithCancel(context.Background())
			time.Sleep(3 * time.Second)
			if bot, err = slacksay.NewBot(ctx, opt.token, config); err != nil { // reboot
				log.Fatalf("reboot failed, %v", err)
			}
			log.Printf("reboot")
			continue
		}
		log.Printf("bot_id: %v, msg_user_id: %v, msg:%+v\n", bot.ID, msg.UserID, msg)
		if msg.Type != "message" && len(msg.TextBody()) == 0 {
			continue
		}
		go bot.Response(&msg)
	}
}
