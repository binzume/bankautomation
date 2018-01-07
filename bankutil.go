package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/binzume/gobanking/common"
	"github.com/binzume/gobanking/mizuho"
	"github.com/binzume/gobanking/rakuten"
	"github.com/binzume/gobanking/sbi"
	"github.com/binzume/gobanking/shinsei"
	"github.com/binzume/gobanking/stub"
)

type AccountConfig struct {
	Bank     string                 `json:"bank"`
	Id       string                 `json:"id"`
	Password string                 `json:"password"`
	Options  map[string]interface{} `json:"options"`
}

type Account common.Account

func login(path string) (common.Account, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c AccountConfig
	err = json.Unmarshal(raw, &c)
	if err != nil {
		return nil, err
	}
	switch c.Bank {
	case "mizuho":
		words := map[string]string{}
		for k, v := range c.Options {
			words[k] = v.(string)
		}
		return mizuho.Login(c.Id, c.Password, words)
	case "rakuten":
		words := map[string]string{}
		for k, v := range c.Options {
			words[k] = v.(string)
		}
		return rakuten.Login(c.Id, c.Password, words)
	case "shinsei":
		grid := []string{}
		for _, f := range c.Options["grid"].([]interface{}) {
			grid = append(grid, f.(string))
		}
		return shinsei.Login(c.Id, c.Password, c.Options["numId"].(string), grid)
	case "sbi":
		return sbi.Login(c.Id, c.Password)
	case "stub":
		return stub.Login(c.Id, c.Password, c.Options)
	default:
		return nil, errors.New("unknown:" + c.Bank)
	}
}
