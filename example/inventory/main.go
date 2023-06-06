package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/vuquang23/go-steam/tradeoffer"
)

var (
	apiKey           string
	sessionID        string
	steamLogin       string
	steamLoginSecure string
)

func init() {
	apiKey = os.Getenv("API_KEY")
	steamLogin = os.Getenv("STEAM_LOGIN")
	steamLoginSecure = os.Getenv("STEAM_LOGIN_SECURE")
	sessionID = os.Getenv("SESSION_ID")
}

func main() {
	client := tradeoffer.NewClient(tradeoffer.APIKey(apiKey), sessionID, steamLogin, steamLoginSecure)

	data, err := client.GetOwnInventory(2, 730, true)
	if err != nil {
		log.Fatal(err)
	}

	m, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(m))
}
