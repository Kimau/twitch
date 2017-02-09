package twitch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func (ah *Client) handleOAuthStart(w http.ResponseWriter, req *http.Request) {
	match := authSignRegEx.FindStringSubmatch(req.URL.Path)
	if match != nil {
		fullRedirStr := fmt.Sprintf(baseURL, rootURL, clientID, redirURL, ah.getScopeString(), match[1])
		http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
		return
	}
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

	nameList, ok := qList["state"]
	if !ok {
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
	ah.hasAuth = false
	ah.username = nameList[0]
	ah.authcode = c[0]
	for k := range ah.scopes {
		ah.scopes[k] = false
	}
	for _, k := range scopeList {
		ah.scopes[k] = true
	}

	// Setup Payload
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirURL)
	data.Set("code", ah.authcode)
	data.Set("state", ah.username)
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

	// Dump Output
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	fmt.Fprintf(w, "Output: %s", string(body))
	return

	// Decode JSON
	tokenStruct := struct {
		Token   string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&tokenStruct)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	ah.authcode = tokenStruct.Token
	ah.hasAuth = true

	// Output Result
	fmt.Fprintf(w,
		"Hello %s your code is [%s] and you have been allowed these scopes\n",
		ah.username, ah.authcode)

	for k, v := range ah.scopes {
		fmt.Fprintf(w, "* %s: %v \n", k, v)
	}

}
