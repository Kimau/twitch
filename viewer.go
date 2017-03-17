package twitch

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"
)

// Viewer is basic Viewer
type Viewer struct {
	TwitchID ID
	Coins    Currency

	User     *User
	Auth     *UserAuth
	Chatter  *Chatter
	Follower *ChannelFollow

	client *Client
}

// CreateViewer - Create Bare Viewer
func (ah *Client) CreateViewer(id ID) *Viewer {
	v, ok := ah.Viewers[id]

	if ok {
		log.Printf("Attempting to Create user that already exsists: %s", id)
	} else {
		v = &Viewer{
			TwitchID: id,
			client:   ah,
		}
		ah.Viewers[id] = v
	}

	v.UpdateUser()
	if v.TwitchID != id || v.User.ID != id {
		debug.PrintStack()
		log.Fatalf("Twitch ID doesn't match %s %s %s", id, v.TwitchID, v.User.ID)
	}

	return v
}

// CreateViewerFromUser - Create Viewer from User
func (ah *Client) CreateViewerFromUser(usr User) *Viewer {
	v, ok := ah.Viewers[usr.ID]

	if ok {
		log.Printf("Attempting to Create %s that already exsists: %s", usr.DisplayName, usr.ID)
	} else {
		v = &Viewer{
			TwitchID: usr.ID,
			client:   ah,
			User:     &usr,
		}

		ah.Viewers[usr.ID] = v
	}

	return v
}

// GetViewerFromChatter - Get Viewer from Chatter
func (ah *Client) GetViewerFromChatter(cu *Chatter) *Viewer {
	if cu.id != "" {
		v := ah.GetViewer(cu.id)
		v.Chatter = cu
		return v
	} else if cu.Nick != "" {
		v, err := ah.FindViewer(cu.Nick)
		if err != nil {
			log.Printf("GetViewerFromChatter - unable to get from nick [%s] \n%s", cu.Nick, err)
			return nil
		}
		v.Chatter = cu
		return v
	} else if cu.DisplayName != "" {
		v, err := ah.FindViewer(IrcNick(cu.DisplayName))
		if err != nil {
			log.Printf("GetViewerFromChatter - unable to get from display name [%s] \n%s", cu.DisplayName, err)
			return nil
		}
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

		v = ah.CreateViewerFromUser(*u)
	}

	return v
}

func (ah *Client) findViewerByName(nick IrcNick) *Viewer {
	nick = IrcNick(strings.ToLower(string(nick)))
	for _, v := range ah.Viewers {
		if v.User.Name == nick {
			return v
		}
	}
	return nil
}

// FindViewer -
func (ah *Client) FindViewer(nick IrcNick) (*Viewer, error) {
	v := ah.findViewerByName(nick)
	if v != nil {
		return v, nil
	}

	userList, err := ah.User.GetByName([]IrcNick{nick})
	if err != nil {
		return nil, err
	}

	if len(userList) == 0 {
		return nil, fmt.Errorf("No user found called: %s", nick)
	}

	return ah.CreateViewerFromUser(userList[0]), nil
}

// UpdateViewers - Update Viewers from list of Names
func (ah *Client) UpdateViewers(nickList []IrcNick) []*Viewer {
	vList := []*Viewer{}

	unkownNicks := []IrcNick{}
	// Check if Anyone Unknown
	for _, nick := range nickList {
		ov := ah.findViewerByName(nick)
		if ov != nil {
			vList = append(vList, ov)
		} else {
			unkownNicks = append(unkownNicks, nick)
		}
	}

	if len(unkownNicks) == 0 {
		return vList
	}

	// Get Full List by Name
	userList, err := ah.User.GetByName(unkownNicks)
	if err != nil {
		log.Printf("Error in userList \n---\n%s\n---\n%s",
			JoinNicks(unkownNicks, 4, 18),
			err.Error())
		return nil
	}

	// Get Viewer
	for _, usr := range userList {
		vList = append(vList, ah.CreateViewerFromUser(usr))
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

// UpdateFollowStatus - Update Follower Status
func (vw *Viewer) UpdateFollowStatus() (bool, error) {

	cFollow, err := vw.client.User.IsFollowing(vw.TwitchID, vw.client.MyID)
	if err != nil {
		return false, err
	}

	vw.Follower = cFollow
	return (vw.Follower != nil), nil
}

// UpdateFollowers - Update all the channels followers
func (ah *Client) UpdateFollowers() (int, error) {
	hostChan, err := ah.FindViewer(IrcNick(ah.Chat.Room))

	followers, numFollowers, err := ah.Channel.GetFollowers(hostChan.TwitchID, -1, true)

	for _, f := range followers {
		v, ok := ah.Viewers[f.User.ID]
		if !ok {
			v = ah.CreateViewerFromUser(*f.User)
		}

		v.User = f.User
		v.Follower = &f
	}

	return numFollowers, err
}
