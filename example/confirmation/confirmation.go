package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/vuquang23/go-steam/confirmation"
)

var (
	accountName      string
	password         string
	identitySecret   string
	steamLogin       string
	steamLoginSecure string
	sessionID        string
	steamID          string
)

func init() {
	accountName = os.Getenv("ACCOUNT_NAME")
	password = os.Getenv("PASSWORD")
	identitySecret = os.Getenv("IDENTITY_SECRET")
	steamLogin = os.Getenv("STEAM_LOGIN")
	steamLoginSecure = os.Getenv("STEAM_LOGIN_SECURE")
	sessionID = os.Getenv("SESSION_ID")
	steamID = os.Getenv("STEAM_ID")
}

func main() {
	c := confirmation.NewClient(sessionID, steamLogin, steamLoginSecure, identitySecret, accountName, password, steamID)

	confs, err := c.GetConfirmations()
	if err != nil {
		log.Fatal(err)
	}
	if len(confs) == 0 {
		return
	}

	m, _ := json.Marshal(confs)
	fmt.Println(string(m))

	err = c.AcceptConfirmation(confs[0])
	if err != nil {
		log.Fatal(err)
	}
}
