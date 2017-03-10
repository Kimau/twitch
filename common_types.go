package twitch

import "net/http"

// ID - Numberic Identifier of Twitch Identity
type ID string

// IrcNick - Irc Nick all lowercase identifier
type IrcNick string

// WebClient - Provides basic Request Poster
type WebClient interface {
	Do(req *http.Request) (*http.Response, error)
}
