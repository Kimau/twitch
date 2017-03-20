package twitch

import "log"

// Viewer is basic Viewer
type Viewer struct {
	TwitchID ID       `json:"id"`
	Coins    Currency `json:"coins"`

	User     *User          `json:"user"`
	Auth     *UserAuth      `json:"auth"`
	Chatter  *Chatter       `json:"chatter"`
	Follower *ChannelFollow `json:"follow"`

	client *Client
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

	if vw.Auth != nil && vw.Auth.token != nil {
		return vw.Auth.token.Username
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

// UpdateFollowStatus - Update Follower Status
func (vw *Viewer) UpdateFollowStatus() (bool, error) {

	cFollow, err := vw.client.User.IsFollowing(vw.TwitchID, vw.client.RoomID)
	if err != nil {
		return false, err
	}

	vw.Follower = cFollow
	return (vw.Follower != nil), nil
}
