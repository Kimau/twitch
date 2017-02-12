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
)

var (
	authSignRegEx = regexp.MustCompile("/twitch/signin/([\\w]+)/*")
)

// UserAuth - Used to manage OAuth for Logins
type UserAuth struct {
	TwitchID string
	User     *User

	hasAuth    bool
	oauthState string
	authcode   string
	scopes     map[string]bool
}

func makeUserAuth(id string, reqScopes []string) *UserAuth {
	newScope := make(map[string]bool)
	for _, ov := range ValidScopes {
		newScope[ov] = false
		for _, v := range reqScopes {
			if v == ov {
				newScope[ov] = true
			}
		}
	}

	au := UserAuth{
		TwitchID: id,
		User:     nil,

		hasAuth:    false,
		oauthState: GenerateRandomString(16),
		scopes:     newScope,
	}

	return &au
}

func (ua *UserAuth) getScopeString() string {
	if ua.scopes == nil {
		return ""
	}

	s := ""
	for k, v := range ua.scopes {
		if v {
			s += k + "+"
		}
	}

	s = strings.TrimRight(s, "+")

	return s
}

func (ua *UserAuth) updateScope(scopeList []string) {
	for k := range ua.scopes {
		ua.scopes[k] = false
	}
	for _, k := range scopeList {
		ua.scopes[k] = true
	}
}

func (ua *UserAuth) checkScope(reqScopes ...string) error {
	for _, v := range reqScopes {
		if ua.scopes[v] == false {
			return fmt.Errorf("Scope Required: %s", v)
		}
	}

	return nil
}

func (ua *UserAuth) handleOAuthStart(w http.ResponseWriter, req *http.Request) {
	if ua.hasAuth {
		http.Error(w, "Already logged in admin", http.StatusConflict)
		return
	}

	fullRedirStr := fmt.Sprintf(baseURL, rootURL, clientID, redirURL, ua.getScopeString(), ua.oauthState)
	http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
}

func (ah *Client) findUserByState(state string) *UserAuth {
	if ah.AdminAuth.oauthState == state {
		return ah.AdminAuth
	}

	for _, v := range ah.AuthUsers {
		if v.oauthState == state {
			return v
		}
	}

	return nil
}

func (ah *Client) handleOAuthResult(w http.ResponseWriter, req *http.Request) {
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

	authU := ah.findUserByState(stateList[0])
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

	// Save State
	authU.hasAuth = false
	authU.updateScope(scopeList)
	authU.authcode = c[0]

	// Setup Payload
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirURL)
	data.Set("code", authU.authcode)
	data.Set("state", authU.oauthState)
	payload := strings.NewReader(data.Encode())

	// Server get Auth Code
	req, err := http.NewRequest("POST", "https://api.twitch.tv/kraken/oauth2/token", payload)
	if err != nil {
		log.Println("Failed to Build Request")
		http.Error(w, err.Error(), 500)
		return
	}

	req.Header.Add("accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("client-id", clientID)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("content-length", strconv.Itoa(len(data.Encode())))
	req.Header.Add("cache-control", "no-cache")

	resp, err := ah.httpClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		if err != nil {
			http.Error(w, fmt.Sprintf("---\n %#v \n---\n %#v \n---\n %#v \n---\n %s", req, resp, payload, err.Error()), 500)
		} else {
			http.Error(w, fmt.Sprintf("---\n %#v \n---\n %#v \n---\n %#v \n---\n", req, resp, payload), resp.StatusCode)
		}
		return
	}

	// Decode JSON
	defer resp.Body.Close()
	tokenStruct := struct {
		Token   string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&tokenStruct)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	authU.authcode = tokenStruct.Token
	authU.hasAuth = true

	// Output Result
	fmt.Fprint(w, "You are logged in")
}
