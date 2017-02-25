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

	ua.token = &authToken{}

	tokenContain := &struct {
		Token *authToken `json:"token"`
	}{ua.token}

	_, err := ah.Get(ua, "", tokenContain)
	if err != nil {
		ua.token = nil
		return err
	}

	if ua.token.IsValid == false {
		err = fmt.Errorf("Root Response is Invalid: %v ", ua.token)
		ua.token = nil
		return err
	}

	if clientID != ua.token.ClientID {
		ua.token = nil
		return fmt.Errorf("Client ID doesn't match [%s:%s]", clientID, ua.token.ClientID)
	}

	return nil
}

func (ah *Client) handleOAuthAdminStart(w http.ResponseWriter, req *http.Request) {
	fullRedirStr := fmt.Sprintf(baseURL,
		rootURL,
		clientID,
		redirURL,
		mergeScopeString(DefaultStreamerScope),
		ah.AdminAuth.oauthState)
	http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
}

func (ah *Client) handleOAuthStart(w http.ResponseWriter, req *http.Request) {
	myState := GenerateRandomString(16)
	ah.PendingLogins[myState] = time.Now()

	fullRedirStr := fmt.Sprintf(baseURL,
		rootURL,
		clientID,
		redirURL,
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
	stateVal := stateList[0]
	// Check if Admin Login
	if (ah.AdminAuth.token == nil) && stateVal == ah.AdminAuth.oauthState {
		authU = ah.AdminAuth
		isAdmin = true
	} else { // Normal user do logic
		authU = &UserAuth{
			oauthState: stateVal,
			scopes:     make(map[string]bool),
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
	authU.token = nil
	authU.updateScope(scopeList)
	authU.ircCode = c[0]

	err := ah.handleOAuthResult(authU)
	if err != nil {
		log.Println(err, authU)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tID := authU.token.UserID
	if isAdmin {
		if ah.AdminAuth.token != nil {
			ah.AdminChannel <- 1
			fmt.Fprintf(w, "Admin logged in %s #%s\n---Scope---\n\t%s\n---------\n",
				authU.token.Username, tID,
				strings.Join(scopeList, "\n\t"))

			if ah.AdminAuth.scopes[scopeChatLogin] {
				ah.Chat, err = createIrcClient(ah.AdminAuth)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				go func() {
					err := ah.Chat.StartRunLoop()
					if err != nil {
						log.Fatal(err)
					}
				}()
			}

		} else {
			http.Error(w, "Admin Auth has no token", 400)
		}

		go func() {
			ah.AdminUser, err = ah.User.Get(tID)
			if err != nil {
				log.Println("Failed to Get Admin User Data", err)
			}
		}()
	} else {

		v := &Viewer{
			User:   nil,
			Auth:   authU,
			client: ah,
		}

		ah.Viewers[tID] = v
		http.SetCookie(w, v.Auth.createSessionCookie())
		fmt.Fprintf(w, "Logged in %s #%s", v.Nick(), tID)

		go v.UpdateUser()
	}
}

func (ah *Client) handleOAuthResult(authU *UserAuth) error {

	// Setup Payload
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirURL)
	data.Set("code", authU.ircCode)
	data.Set("state", authU.oauthState)
	payload := strings.NewReader(data.Encode())

	// Server get Auth Code
	req, err := http.NewRequest("POST", "https://api.twitch.tv/kraken/oauth2/token", payload)
	if err != nil {
		log.Println("Failed to Build Request")
		return err
	}

	req.Header.Add("accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("client-id", clientID)
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
	authU.authcode = tokenStruct.Token

	err = ah.getRootToken(authU)
	if err != nil {
		return err
	}

	// Output Result
	return nil
}
