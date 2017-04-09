package twitch

import (
	"log"
	"sync"
)

// Viewer is basic Viewer
type Viewer struct {
	TwitchID ID `json:"id"`

	User     *User          `json:"user"`    // Read Only - not enforced for perf reasons
	Auth     *UserAuth      `json:"auth"`    // Read Only - not enforced for perf reasons
	Chatter  *Chatter       `json:"chatter"` // Read Only - not enforced for perf reasons
	Follower *ChannelFollow `json:"follow"`  // Read Only - not enforced for perf reasons

	m      sync.Mutex
	client *Client
}

//SetUser - Sets the new value in with lock
func (vw *Viewer) SetUser(newVal User) {
	vw.m.Lock()
	vw.User = &newVal
	vw.m.Unlock()
}

//SetAuth - Sets the new value in with lock
func (vw *Viewer) SetAuth(newVal UserAuth) {
	vw.m.Lock()
	vw.Auth = &newVal
	vw.m.Unlock()
}

//ClearAuth - Clear the Value with lock
func (vw *Viewer) ClearAuth() {
	vw.m.Lock()
	vw.Auth = nil
	vw.m.Unlock()
}

//SetChatter - Sets the new value in with lock
func (vw *Viewer) SetChatter(newVal Chatter) {
	vw.m.Lock()
	vw.Chatter = &newVal
	vw.m.Unlock()
}

// Get will make Twitch API request with correct headers then attempt to decode JSON into jsonStruct
func (vw *Viewer) Get(path string, jsonStruct interface{}) (string, error) {
	return vw.client.Get(vw.Auth, path, jsonStruct)
}

// GetNick - Returns short username of current UserAuth
func (vw *Viewer) GetNick() IrcNick {
	if vw.User != nil {
		return vw.User.Name
	}

	if vw.Auth != nil && vw.Auth.Token != nil {
		return vw.Auth.Token.Username
	}

	return IrcNick("0x" + vw.TwitchID)
}

// UpdateUser - Calls API to update User Data
func (vw *Viewer) UpdateUser() error {
	var err error

	vw.User, err = vw.client.User.Get(vw.TwitchID)
	if err != nil {
		log.Printf("Failed to Get User Data for %s - %s", vw.TwitchID, err)
	}

	return nil
}
