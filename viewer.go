package twitch

import (
	"fmt"
	"log"
)

// Viewer is basic Viewer
type Viewer struct {
	TwitchID ID

	User    *User
	Auth    *UserAuth
	Chatter *chatter

	client *Client
}

// GetViewerFromChatter - Get Viewer from Chatter
func (ah *Client) GetViewerFromChatter(cu *chatter) *Viewer {
	if cu.id != "" {
		v := ah.GetViewer(cu.id)
		v.Chatter = cu
		return v
	} else if cu.nick != "" {
		v := ah.FindViewer(cu.nick)
		v.Chatter = cu
		return v
	} else if cu.displayName != "" {
		v := ah.FindViewer(IrcNick(cu.displayName))
		v.Chatter = cu
		return v
	}

	fmt.Printf("GetViewerFromChatter ERROR \n %#v", cu)
	return nil
}

// GetViewer - Get Viewer by ID
func (ah *Client) GetViewer(twitchID ID) *Viewer {
	v, ok := ah.Viewers[twitchID]
	if !ok {
		u, err := ah.User.Get(twitchID)
		if err != nil {
			log.Printf("Unable to get User %s\n%s", twitchID, err.Error())
			return nil
		}

		ah.Viewers[twitchID] = &Viewer{
			TwitchID: twitchID,
			User:     u,
		}
	}

	return v
}

// FindViewer -
func (ah *Client) FindViewer(nick IrcNick) *Viewer {
	for _, v := range ah.Viewers {
		if v.User.Name == nick {
			return v
		}
	}

	userList, err := ah.User.GetByName([]IrcNick{nick})
	if err != nil {
		log.Printf("Error in finding %s\n%s", nick, err.Error())
		return nil
	}

	return &Viewer{
		TwitchID: userList[0].ID,
		User:     &userList[0],
	}
}

// UpdateViewers - Update Viewers from list of Names
func (ah *Client) UpdateViewers(nickList []IrcNick) []*Viewer {
	userList, err := ah.User.GetByName(nickList)
	if err != nil {
		log.Printf("Error in userList \n---\n%s\n---\n%s",
			JoinNicks(nickList, 4, 18),
			err.Error())
		return nil
	}

	vList := []*Viewer{}
	for _, u := range userList {
		v, ok := ah.Viewers[u.ID]
		if !ok {
			v = &Viewer{
				TwitchID: u.ID,
				User:     &u,
			}
			ah.Viewers[u.ID] = v
		} else {
			// Update User Data
			v.User = &u
		}
		vList = append(vList, v)
	}

	return vList
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
