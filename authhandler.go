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

type authToken struct {
	AuthToken struct {
		CreatedAtString string   `json:"created_at"` // 2013-06-03T19:12:02Z
		UpdatedAtStr    string   `json:"updated_at"` // 2016-12-14T01:01:44Z
		ScopeList       []string `json:"scopes"`
	} `json:"authorization"`
	ClientID string `json:"client_id"` // "uo6dggojyb8d6soh92zknwmi5ej1q2"
	UserID   string `json:"user_id"`   // "44322889"
	Username string `json:"user_name"` // "dallas"
	IsValid  bool   `json:"valid"`     // true
}

// UserAuth - Used to manage OAuth for Logins
type UserAuth struct {
	TwitchID string
	User     *User

	authcode   string
	oauthState string
	token      *authToken

	scopes map[string]bool

	client *Client
}

func makeUserAuth(id string, twitchClient *Client, reqScopes []string) *UserAuth {
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
		token:    nil,

		oauthState: GenerateRandomString(16),
		scopes:     newScope,
		client:     twitchClient,
	}

	return &au
}

func (ua *UserAuth) getRootToken() error {

	rootResponse := authToken{}

	_, err := ua.client.Get(ua, "", &rootResponse)
	if err != nil {
		return err
	}

	if rootResponse.IsValid == false {
		return fmt.Errorf("Root Response is Invalid: %v", rootResponse)
	}

	if clientID != rootResponse.ClientID {
		return fmt.Errorf("Client ID doesn't match [%s:%s]", clientID, rootResponse.ClientID)
	}

	ua.token = &rootResponse
	return nil
}

// GetAuth - checks if auth and if auth returns auth code
func (ua *UserAuth) GetAuth() (bool, string) {
	if ua.token == nil {
		return false, ""
	}
	return true, ua.authcode
}

// GetIrcAuth - returns the stuff needed for IRC
func (ua *UserAuth) GetIrcAuth() (hasauth bool, name string, pass string, addr string) {
	isAuth, c := ua.GetAuth()
	if !isAuth {
		return false, "", "", ircServerAddr
	}

	return true, ua.token.Username, "oauth:" + c, ircServerAddr
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
	if ua.token != nil {
		http.Error(w, "Already logged in admin", http.StatusConflict)
		return
	}

	fullRedirStr := fmt.Sprintf(baseURL, rootURL, clientID, redirURL, ua.getScopeString(), ua.oauthState)
	http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
}

func (ah *Client) findUserByState(state string) *UserAuth {
	for _, v := range ah.AuthUsers {
		if v.oauthState == state {
			return v
		}
	}

	return nil
}

func (ah *Client) handleAdminOAuthResult(w http.ResponseWriter, req *http.Request) {
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
	if !ok || ah.AdminAuth.oauthState != stateList[0] {
		http.Error(w, "Invalid State", 400)
		return
	}

	scopeList, ok := qList["scope"]
	if !ok {
		http.Error(w, "Invalid Scope", 400)
		return
	}
	scopeList = strings.Split(scopeList[0], " ")

	// Save State
	ah.AdminAuth.updateScope(scopeList)
	ah.AdminAuth.authcode = c[0]

	ah.handleOAuthResult(w, ah.AdminAuth)

	if ah.AdminAuth.token != nil {
		var err error
		ah.AdminAuth.User, err = ah.User.Get(ah.AdminAuth.token.ClientID)

		if err != nil {
			http.Error(w, "Unable to get Auth Token: "+err.Error(), 400)
		}

		ah.AdminChannel <- 1
	}
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
	authU.token = nil
	authU.updateScope(scopeList)
	authU.authcode = c[0]

	ah.handleOAuthResult(w, authU)
}

func (ah *Client) handleOAuthResult(w http.ResponseWriter, authU *UserAuth) {

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

	err = authU.getRootToken()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Output Result
	fmt.Fprint(w, "You are logged in")
}
