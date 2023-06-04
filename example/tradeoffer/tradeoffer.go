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

	var (
		getSent              = true
		getReceived          = true
		getDescriptions      = false
		activeOnly           = true
		historicalOnly       = false
		timeHistoricalCutoff = uint32(0) // https://dev.doctormckay.com/topic/4013-ieconservicegettradeoffersv1-time_historical_cutoff-on-sent-offers/
	)

	offers, err := client.GetOffers(getSent, getReceived, getDescriptions, activeOnly, historicalOnly, &timeHistoricalCutoff)
	if err != nil {
		log.Fatal(err)
	}

	bytes, err := json.Marshal(offers)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(bytes))
}
