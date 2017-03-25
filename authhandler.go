package twitch

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	authSignRegEx = regexp.MustCompile("/twitch/signin/([\\w]+)/*")
)

func (ah *Client) getRootToken(ua *UserAuth) error {

	ua.Token = &authToken{}

	tokenContain := &struct {
		Token *authToken `json:"token"`
	}{ua.Token}

	_, err := ah.Get(ua, "", tokenContain)
	if err != nil {
		ua.Token = nil
		return err
	}

	if ua.Token.IsValid == false {
		err = fmt.Errorf("Root Response is Invalid: %v ", ua.Token)
		ua.Token = nil
		return err
	}

	if ah.ClientID != ua.Token.ClientID {
		ua.Token = nil
		return fmt.Errorf("Client ID doesn't match [%s:%s]", ah.ClientID, ua.Token.ClientID)
	}

	return nil
}

func (ah *Client) handleOAuthAdminStart(w http.ResponseWriter, req *http.Request) {

	fullRedirStr := fmt.Sprintf(twitchAuthURL,
		ah.ClientID,
		fmt.Sprintf(redirStringURL, ah.domain),
		mergeScopeString(DefaultStreamerScope),
		ah.AdminAuth.InteralState)

	http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
}

func (ah *Client) handleOAuthStart(w http.ResponseWriter, req *http.Request) {
	myState := GenerateRandomString(16)
	ah.PendingLogins[authInternalState(myState)] = time.Now()

	fullRedirStr := fmt.Sprintf(twitchAuthURL,
		ah.ClientID,
		fmt.Sprintf(redirStringURL, ah.domain),
		mergeScopeString(DefaultViewerScope),
		myState)
	http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
}

func (ah *Client) handlePublicOAuthResult(w http.ResponseWriter, req *http.Request) {
	qList := req.URL.Query()

	c, ok := qList["code"]
	if !ok {
		s := fmt.Sprintf("Hello your auth was cancelled")

		for k, v := range qList {
			s += fmt.Sprintf("%s:%s\n", k, v)
		}
		http.Error(w, s, 400)
		return
	}

	stateList, ok := qList["state"]
	if !ok {
		http.Error(w, "Invalid State", 400)
		return
	}

	var authU *UserAuth
	isAdmin := false
	stateVal := authInternalState(stateList[0])
	// Check if Admin Login
	if (ah.AdminAuth.Token == nil) && stateVal == ah.AdminAuth.InteralState {
		authU = ah.AdminAuth
		isAdmin = true
	} else { // Normal user do logic
		authU = &UserAuth{
			InteralState: stateVal,
			Scopes:       make(map[string]bool),
		}

		delete(ah.PendingLogins, stateVal)
	}

	if authU == nil {
		http.Error(w, "Invalid Auth State", 400)
		return
	}

	scopeList, ok := qList["scope"]
	if !ok {
		http.Error(w, "Invalid Scope", 400)
		return
	}
	scopeList = strings.Split(scopeList[0], " ")
	log.Println(strings.Split(scopeList[0], "\n\t"))

	// Save State
	authU.Token = nil
	authU.updateScope(scopeList)
	authU.IrcCode = c[0]

	err := ah.handleOAuthResult(authU)
	if err != nil {
		log.Println(err, authU)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tID := authU.Token.UserID
	if isAdmin {
		if ah.AdminAuth.Token != nil {
			err := ah.saveToken()
			if err != nil {
				log.Fatalf("Unable to save token: %s", err)
			}

			ah.adminHasAuthed()

			fmt.Fprintf(w, "Admin logged in %s #%s\n---Scope---\n\t%s\n---------\n",
				authU.Token.Username, tID,
				strings.Join(scopeList, "\n\t"))
		} else {
			http.Error(w, "Admin Auth has no token", 400)
		}
	} else {
		v := ah.GetViewer(tID)
		v.Auth = authU

		http.SetCookie(w, v.Auth.createSessionCookie(ah.domain))
		fmt.Fprintf(w, "Logged in %s #%s", v.GetNick(), tID)
	}
}

func (ah *Client) handleOAuthResult(authU *UserAuth) error {

	// Setup Payload
	data := url.Values{}
	data.Set("client_id", ah.ClientID)
	data.Set("client_secret", ah.ClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", fmt.Sprintf(redirStringURL, ah.domain))
	data.Set("code", authU.IrcCode)
	data.Set("state", string(authU.InteralState))
	payload := strings.NewReader(data.Encode())

	// Server get Auth Code
	req, err := http.NewRequest("POST", "https://api.twitch.tv/kraken/oauth2/token", payload)
	if err != nil {
		log.Println("Failed to Build Request")
		return err
	}

	req.Header.Add("accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("client-id", ah.ClientID)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("content-length", strconv.Itoa(len(data.Encode())))
	req.Header.Add("cache-control", "no-cache")

	resp, err := ah.httpClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		log.Printf("---\n %#v \n---\n %#v \n---\n %#v \n---\n", req, resp, payload)
		if err != nil {
			return err
		}
		return fmt.Errorf("Failed to auth follow through %d - %s", resp.StatusCode, resp.Body)
	}

	// Decode JSON
	defer resp.Body.Close()
	tokenStruct := struct {
		Token   string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&tokenStruct)
	if err != nil {
		return err
	}
	authU.AuthCode = tokenStruct.Token

	err = ah.getRootToken(authU)
	if err != nil {
		return err
	}

	// Output Result
	return nil
}
