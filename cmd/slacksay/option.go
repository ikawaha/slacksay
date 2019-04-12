package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ikawaha/slacksay"
)

// options
type option struct {
	data    string
	token   string
	flagSet *flag.FlagSet
}

// ContinueOnError ErrorHandling // Return a descriptive error.
// ExitOnError                   // Call os.Exit(2).
// PanicOnError                  // Call panic with a descriptive error.flag.ContinueOnError
func newOption(eh flag.ErrorHandling) *option {
	ret := &option{
		flagSet: flag.NewFlagSet("main", eh),
	}
	// option settings
	ret.flagSet.StringVar(&ret.data, "d", "", "json data. If you start the data with the letter @, the rest should be a file name to read the data from, or -  if you  want to read the data from stdin.")
	ret.flagSet.StringVar(&ret.token, "t", "", "slack token")
	return ret
}

func (o *option) parse(args []string) error {
	if err := o.flagSet.Parse(args); err != nil {
		return err
	}
	// validations
	if nonFlag := o.flagSet.Args(); len(nonFlag) != 0 {
		return fmt.Errorf("invalid arguments, %+v\n", nonFlag)
	}
	if o.token == "" {
		return fmt.Errorf("token is required\n")
	}
	return nil
}

// PrintOptionDefaults prints out the default flags
func PrintOptionDefaults(eh flag.ErrorHandling) {
	o := newOption(eh)
	o.flagSet.PrintDefaults()
}

var defaultConfig = slacksay.Config{
	Timeout:    "1m",
	BotMessage: false,
}

func (o option) NewConfig() (*slacksay.Config, error) {
	var ret slacksay.Config
	if o.data == "" {
		ret = defaultConfig
		return &ret, nil
	}
	var r io.Reader
	if !strings.HasPrefix(o.data, "@") {
		r = bytes.NewBufferString(o.data)
	} else {
		if o.data[1:] == "-" {
			r = os.Stdin
		} else {
			fp, err := os.Open(o.data[1:])
			if err != nil {
				return nil, err
			}
			r = fp
		}
	}
	return slacksay.NewConfigReader(r)
}
