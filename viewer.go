package twitch

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

const (
	debugViewerLock = false
)

// Viewer is basic Viewer
type Viewer struct {
	TwitchID ID `json:"id"`

	User     *User          `json:"user"`    // Read Only - not enforced for perf reasons
	Auth     *UserAuth      `json:"auth"`    // Read Only - not enforced for perf reasons
	Chatter  *Chatter       `json:"chatter"` // Read Only - not enforced for perf reasons
	Follower *ChannelFollow `json:"follow"`  // Read Only - not enforced for perf reasons

	mylock sync.Mutex
	client *Client
}

// CopyTo - Deep copies in threadsafe fashion
func (vw *Viewer) CopyTo(copyV *Viewer) {
	vw.Lockme()
	defer vw.Unlockme()

	copyV.TwitchID = vw.TwitchID

	if vw.User != nil {
		u := *vw.User
		copyV.User = &u
	} else {
		copyV.User = nil
	}

	if vw.Auth != nil {
		a := *vw.Auth
		copyV.Auth = &a
	} else {
		copyV.Auth = nil
	}

	if vw.Chatter != nil {
		c := *vw.Chatter
		copyV.Chatter = &c
	} else {
		copyV.Chatter = nil
	}

	if vw.Follower != nil {
		f := *vw.Follower
		copyV.Follower = &f
	} else {
		copyV.Follower = nil
	}
}

// Lockme - Lock The Viewer
func (vw *Viewer) Lockme() {
	vw.mylock.Lock()
	if debugViewerLock {
		fmt.Println("- LOCK -", vw.TwitchID)
		debug.PrintStack()
	}
}

// Unlockme - Unlock the Viewer
func (vw *Viewer) Unlockme() {
	vw.mylock.Unlock()
	if debugViewerLock {
		fmt.Println("- UNLOCK -", vw.TwitchID)
		debug.PrintStack()
	}
}

//SetUser - Sets the new value in with lock
func (vw *Viewer) SetUser(newVal User) {
	vw.Lockme()
	vw.User = &newVal
	vw.Unlockme()
}

//SetAuth - Sets the new value in with lock
func (vw *Viewer) SetAuth(newVal UserAuth) {
	vw.Lockme()
	vw.Auth = &newVal
	vw.Unlockme()
}

//ClearAuth - Clear the Value with lock
func (vw *Viewer) ClearAuth() {
	vw.Lockme()
	vw.Auth = nil
	vw.Unlockme()
}

//SetChatter - Sets the new value in with lock
func (vw *Viewer) SetChatter(newVal Chatter) {
	vw.Lockme()
	vw.Chatter = &newVal
	vw.Unlockme()
}

//CreateChatter - Creates Blank Chatter
func (vw *Viewer) CreateChatter() {
	vw.Lockme()
	if vw.Chatter == nil {
		log.Printf("Chat: ++Created++ %s", vw.User.Name)
		vw.Chatter = &Chatter{
			Nick:        vw.User.Name,
			DisplayName: vw.User.DisplayName,
			Bits:        0,

			Mod:      false,
			Sub:      0,
			UserType: TwitchTypeEmpty,
			Color:    "#000000",

			TimeInChannel: 0,
			LastActive:    time.Now(),
		}
	}
	vw.Unlockme()
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
