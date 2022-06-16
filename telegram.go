package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

var (
	botToken = os.Getenv("TELEGRAM_TOKEN")
	chat     = os.Getenv("TELEGRAM_CHAT")
)

func telegramNotify(str string) {
	if botToken == "" || chat == "" {
		panic("Missing [TELEGRAM_TOKEN] or [TELEGRAM_CHAT], can't notify.")
	}

	json, err := json.Marshal(map[string]string{"chat_id": chat, "text": str})
	if err != nil {
		panic(err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	_, err = http.Post(url, "application/json", bytes.NewBuffer(json))
	if err != nil {
		panic(err)
	}
}
