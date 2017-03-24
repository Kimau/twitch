package twitch

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type tokenData struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	IrcServerAddr string `json:"irc_server"`
}

func (ah *Client) loadSecrets() {
	fileData, err := ioutil.ReadFile("./data/twitch_secret.json")
	if err != nil {
		log.Fatalf("Dailed to load token data from ./data/secret.json: \n%s", err)
	}

	sd := tokenData{}
	json.Unmarshal(fileData, &sd)

	// fmt.Printf("ClientID: %s\n ClientSecret: %s\n IrcServerAddr: %s",sd.ClientID,sd.ClientSecret,sd.IrcServerAddr)

	ah.tokenData = &sd
}
