package twitch

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// UserAuthSessionCookieName is the session cookie name
	UserAuthSessionCookieName = "session"
)

type authToken struct {
	AuthToken struct {
		CreatedAtString string   `json:"created_at"` // 2013-06-03T19:12:02Z
		UpdatedAtStr    string   `json:"updated_at"` // 2016-12-14T01:01:44Z
		ScopeList       []string `json:"scopes"`
	} `json:"authorization"`
	ClientID string  `json:"client_id"` // "uo6dggojyb8d6soh92zknwmi5ej1q2"
	UserID   ID      `json:"user_id"`   // "44322889"
	Username IrcNick `json:"user_name"` // "dallas"
	IsValid  bool    `json:"valid"`     // true
}

// UserAuth - Used to manage OAuth for Logins
type UserAuth struct {
	AuthCode      string
	IrcCode       string
	Scopes        map[string]bool
	SessionCookie *http.Cookie

	oauthState authInternalState
	token      *authToken
}

// GetAuth - checks if auth and if auth returns auth code
func (ua *UserAuth) GetAuth() (bool, string) {
	if ua.token == nil {
		return false, ""
	}

	return ua.token.IsValid, ua.AuthCode
}

// GetIrcAuth - returns the stuff needed for IRC
func (ua *UserAuth) GetIrcAuth() (hasauth bool, name string, pass string) {
	isAuth, _ := ua.GetAuth()
	if !isAuth {
		return false, "", ""
	}

	return true, string(ua.token.Username), "oauth:" + ua.AuthCode
}

func mergeScopeString(scopeList []string) string {
	return strings.Join(scopeList, "+")
}

func splitScopeString(scopeString string) []string {
	return strings.Split(scopeString, "+")
}

func (ua *UserAuth) getScopeString() string {
	if ua.Scopes == nil {
		return ""
	}

	s := ""
	for k, v := range ua.Scopes {
		if v {
			s += k + "+"
		}
	}

	s = strings.TrimRight(s, "+")

	return s
}

func (ua *UserAuth) updateScope(scopeList []string) {
	for k := range ua.Scopes {
		ua.Scopes[k] = false
	}
	for _, k := range scopeList {
		ua.Scopes[k] = true
	}
}

func (ua *UserAuth) checkScope(reqScopes ...string) error {
	for _, v := range reqScopes {
		if ua.Scopes[v] == false {
			return fmt.Errorf("Scope Required: %s", v)
		}
	}

	return nil
}

func (ua *UserAuth) checkCookie(c *http.Cookie) bool {
	return (ua.SessionCookie != nil && c != nil && ua.SessionCookie.Value == c.Value)
}

func (ua *UserAuth) createSessionCookie(domain string) *http.Cookie {
	expiration := time.Now().Add(365 * 24 * time.Hour)
	ua.SessionCookie = &http.Cookie{
		Name:    UserAuthSessionCookieName,
		Value:   fmt.Sprintf("%s:%s", ua.token.UserID, GenerateRandomString(16)),
		Domain:  domain, // Wont work for local host because not valid domain
		Path:    "/twitch",
		Expires: expiration,
	}

	return ua.SessionCookie
}
