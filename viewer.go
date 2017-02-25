package twitch

import "log"

// Viewer is basic Viewer
type Viewer struct {
	TwitchID string

	User   *User
	Auth   *UserAuth
	client *Client
}

// Get will make Twitch API request with correct headers then attempt to decode JSON into jsonStruct
func (vw *Viewer) Get(path string, jsonStruct interface{}) (string, error) {
	return vw.client.Get(vw.Auth, path, jsonStruct)
}

// Nick - Returns short username of current UserAuth
func (vw *Viewer) Nick() string {
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
