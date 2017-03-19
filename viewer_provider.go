package twitch

import (
	"fmt"
	"log"
	"strings"
)

type viewerProvider interface {
	GetAuthViewer() *Viewer
	GetRoom() *Viewer

	GetViewer(ID) *Viewer
	GetViewerFromChatter(*Chatter) *Viewer
	GetViewerFromUser(*User) *Viewer

	FindViewer(IrcNick) (*Viewer, error)
	UpdateViewers([]IrcNick) []*Viewer
}

// GetAuthViewer - Returns the account running the bot
func (ah *Client) GetAuthViewer() *Viewer {
	if ah.AdminAuth != nil && ah.AdminAuth.token != nil {
		return ah.Viewers[ah.AdminAuth.token.UserID]
	}
	return nil
}

// GetRoom - Returns the account we are watching
func (ah *Client) GetRoom() *Viewer {
	return ah.Viewers[ah.RoomID]
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

		v = ah.GetViewerFromUser(u)
	}

	return v
}

// GetViewerFromUser - Get Viewer from User
func (ah *Client) GetViewerFromUser(usr *User) *Viewer {
	v, ok := ah.Viewers[usr.ID]

	if ok {
		log.Printf("Attempting to Create %s that already exsists: %s", usr.DisplayName, usr.ID)
	} else {
		v = &Viewer{
			TwitchID: usr.ID,
			client:   ah,
			User:     usr,
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

	return ah.GetViewerFromUser(&userList[0]), nil
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
	for i := range userList {
		vList = append(vList, ah.GetViewerFromUser(&userList[i]))
	}

	return vList
}

// UpdateFollowers - Update all the channels followers
func (ah *Client) UpdateFollowers() (int, error) {
	followers, numFollowers, err := ah.Channel.GetFollowers(ah.RoomID, -1, true)

	for _, f := range followers {
		v, ok := ah.Viewers[f.User.ID]
		if !ok {
			v = ah.GetViewerFromUser(f.User)
		}

		v.User = f.User
		v.Follower = &f
	}

	return numFollowers, err
}
