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

// Emote - Emote match string and internal ID
type Emote struct {
	Code string `json:"code,omitempty"`
	ID   int    `json:"id,omitempty"`
}

// EmoteSets - Group of Emotes
type EmoteSets struct {
	SetMap map[string][]Emote `json:"emoticon_sets,omitempty"`
}
