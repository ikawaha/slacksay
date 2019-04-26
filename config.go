package slacksay

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Config represents the setting for the slacksay bot.
type Config struct {
	Command    string    `json:"command"`
	Channel    Condition `json:"channel"`
	User       Condition `json:"user"`
	Keyword    Condition `json:"keyword"`
	BotMessage bool      `json:"bot_message"`
	Timeout    string    `json:"timeout"`
}

// Condition represents message filters and pronunciations for some keywords.
type Condition struct {
	Yomi     []string          `json:"yomi"`
	Includes []string          `json:"includes"`
	Excludes []string          `json:"excludes"`
	replacer *strings.Replacer `json:"-"`
}

func (c Condition) newYomiReplacer() (*strings.Replacer, error) {
	if len(c.Yomi)%2 == 1 {
		return nil, fmt.Errorf("invalid yomi format")
	}
	return strings.NewReplacer(c.Yomi...), nil
}

func (c Condition) isNotified(item string) bool {
	for _, v := range c.Includes {
		if v == item {
			return true
		}
	}
	return false
}

func (c Condition) isMute(item string) bool {
	for _, v := range c.Excludes {
		if v == item {
			return true
		}
	}
	return false
}

// NewConfigReader creates a config of slacksay bot from io reader.
func NewConfigReader(r io.Reader) (*Config, error) {
	dec := json.NewDecoder(r)
	var c Config
	err := dec.Decode(&c)
	return &c, err
}

// String returns json representation of the config.
func (c Config) String() string {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err.Error()
	}
	return string(b)
}
