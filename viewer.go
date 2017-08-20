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

// ViewerData - Viewer Data
type ViewerData struct {
	TwitchID ID             `json:"id"`
	User     *User          `json:"user"`    // Read Only - not enforced for perf reasons
	Auth     *UserAuth      `json:"auth"`    // Read Only - not enforced for perf reasons
	Chatter  *Chatter       `json:"chatter"` // Read Only - not enforced for perf reasons
	Follower *ChannelFollow `json:"follow"`  // Read Only - not enforced for perf reasons
}

// Viewer is basic Viewer
type Viewer struct {
	data   ViewerData
	mylock sync.Mutex
	client *Client
}

// GetData - Returns the Viewer Data
func (vw *Viewer) GetData() ViewerData {
	vw.Lockme()
	defer vw.Unlockme()
	return vw.data
}

// SetData - Returns the Viewer Data
func (vw *Viewer) SetData(vd ViewerData) {
	vw.Lockme()
	defer vw.Unlockme()
	vw.data = vd
}

// Lockme - Lock The Viewer
func (vw *Viewer) Lockme() {
	vw.mylock.Lock()
	if debugViewerLock {
		fmt.Println("- LOCK -", vw.data.TwitchID)
		debug.PrintStack()
	}
}

// Unlockme - Unlock the Viewer
func (vw *Viewer) Unlockme() {
	vw.mylock.Unlock()
	if debugViewerLock {
		fmt.Println("- UNLOCK -", vw.data.TwitchID)
		debug.PrintStack()
	}
}

//SetUser - Sets the new value in with lock
func (vw *Viewer) SetUser(newVal User) {
	vw.Lockme()
	vw.data.User = &newVal
	vw.Unlockme()
}

//SetAuth - Sets the new value in with lock
func (vw *Viewer) SetAuth(newVal UserAuth) {
	vw.Lockme()
	vw.data.Auth = &newVal
	vw.Unlockme()
}

//ClearAuth - Clear the Value with lock
func (vw *Viewer) ClearAuth() {
	vw.Lockme()
	vw.data.Auth = nil
	vw.Unlockme()
}

//SetChatter - Sets the new value in with lock
func (vw *Viewer) SetChatter(newVal Chatter) {
	vw.Lockme()
	vw.data.Chatter = &newVal
	vw.Unlockme()
}

//CreateChatter - Creates Blank Chatter
func (vw *Viewer) CreateChatter() Chatter {
	vw.Lockme()
	defer vw.Unlockme()

	if vw.data.Chatter == nil {
		log.Printf("Chat: ++Created++ %s", vw.data.User.Name)
		vw.data.Chatter = &Chatter{
			Nick:        vw.data.User.Name,
			DisplayName: vw.data.User.DisplayName,
			Bits:        0,

			Mod:      false,
			Sub:      0,
			UserType: TwitchTypeEmpty,
			Color:    "#000000",

			TimeInChannel: 0,
			LastActive:    time.Now(),
		}
	}

	return *vw.data.Chatter
}

// Get will make Twitch API request with correct headers then attempt to decode JSON into jsonStruct
func (vw *Viewer) Get(path string, jsonStruct interface{}) (string, error) {
	return vw.client.Get(vw.data.Auth, path, jsonStruct)
}

// GetNick - Returns short username of current UserAuth
func (vData *ViewerData) GetNick() IrcNick {
	if vData.User != nil {
		return vData.User.Name
	}

	if vData.Auth != nil && vData.Auth.Token != nil {
		return vData.Auth.Token.Username
	}

	return IrcNick("0x" + vData.TwitchID)
}

// UpdateUser - Calls API to update User Data
func (vw *Viewer) UpdateUser() error {
	var err error
	vw.Lockme()

	vw.data.User, err = vw.client.User.Get(vw.data.TwitchID)
	if err != nil {
		log.Printf("Failed to Get User Data for %s - %s", vw.data.TwitchID, err)
	}

	vw.Unlockme()

	return nil
}
