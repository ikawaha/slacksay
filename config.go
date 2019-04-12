package slacksay

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Config struct {
	Command    string    `json:"command"`
	Channel    Condition `json:"channel"`
	User       Condition `json:"user"`
	Keyword    Condition `json:"keyword"`
	BotMessage bool      `json:"bot_message"`
	Timeout    string    `json:"timeout"`
}

type Condition struct {
	Yomi     []string          `json:"yomi"`
	Includes []string          `json:"includes"`
	Excludes []string          `json:"excludes"`
	replacer *strings.Replacer `json:-`
}

func (c Condition) NewYomiReplacer() (*strings.Replacer, error) {
	if len(c.Yomi)%2 == 1 {
		return nil, fmt.Errorf("invalid yomi format")
	}
	return strings.NewReplacer(c.Yomi...), nil
}

func (c Condition) IsNotified(item string) bool {
	for _, v := range c.Includes {
		if v == item {
			return true
		}
	}
	return false
}

func (c Condition) IsMute(item string) bool {
	for _, v := range c.Excludes {
		if v == item {
			return true
		}
	}
	return false
}

func NewConfigReader(r io.Reader) (*Config, error) {
	dec := json.NewDecoder(r)
	var c Config
	err := dec.Decode(&c)
	return &c, err
}

func (c Config) String() string {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err.Error()
	}
	return string(b)
}
