package twitch

import "log"

// Viewer is basic Viewer
type Viewer struct {
	TwitchID TwitchID

	User    *User
	Auth    *UserAuth
	Chatter *chatter

	client *Client
}

// GetViewer - Retrieves the viewer or adds a dummy viewer and requests an update
func (client *Client) GetViewer(id TwitchID) *Viewer {
	v, ok := client.Viewers[id]

	if ok {
		return v
	}

	v = &Viewer{
		TwitchID: id,
		client:   client,
	}
	go v.UpdateUser()

	client.Viewers[id] = v
	return v
}

// FindViewerIdByName - Attempts to find viewer by ID
func (client *Client) FindViewerIdByName(name ircNick) *Viewer {

	for _, v := range client.Viewers {
		if v.User != nil && v.User.Nick == name {
			return v
		}
		if v.Chat != nil && v.Chat.Nick == name {
			return v
		}
		if v.Auth != nil && v.Auth.token != nil && v.Auth.token.Username == name {
			return v
		}
	}

	return nil
}

// Get will make Twitch API request with correct headers then attempt to decode JSON into jsonStruct
func (vw *Viewer) Get(path string, jsonStruct interface{}) (string, error) {
	return vw.client.Get(vw.Auth, path, jsonStruct)
}

// Nick - Returns short username of current UserAuth
func (vw *Viewer) Nick() IrcNick {
	if vw.User != nil {
		return vw.User.Name
	}

	if vw.Auth != nil && vw.Auth.token != nil {
		return vw.Auth.token.Username
	}

	return vw.TwitchID
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
