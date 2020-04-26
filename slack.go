package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/binzume/gobanking/utils"
)

func SendSlackMessage(url, text string) error {
	data := &struct {
		Text string `json:"text"`
	}{text}
	b, err := json.Marshal(data)
	client, _ := utils.NewHttpClient()

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()

	return err
}
