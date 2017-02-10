package twitch

import "fmt"

// GetMe               | Get Channel                      | Gets a channel object based on the OAuth token provided.
// Get                 | Get Channel by ID                | Gets a specified channel object.
// Update              | Update Channel                   | Updates specified properties of a specified channel.
// GetEditors          | Get Channel Editors              | Gets a list of users who are editors for a specified channel.
// GetFollowers        | Get Channel Followers            | Gets a list of users who follow a specified channel, sorted by the date when they started following the channel (newest first, unless specified otherwise).
// GetTeams            | Get Channel Teams                | Gets a list of teams to which a specified channel belongs.
// GetSubscribers      | Get Channel Subscribers          | Gets a list of users subscribed to a specified channel, sorted by the date when they subscribed.
// IsUserGetSubscribed | Check Channel Subscription User  | Checks if a specified channel has a specified user subscribed to it. Intended for use by channel owners.
// GetVideos           | Get Channel Videos               | Gets a list of VODs (Video on Demand) from a specified channel.
// StartCommercial     | Start Channel Commercial         | Starts a commercial (advertisement) on a specified channel. This is valid only for channels that are Twitch partners. You cannot start a commercial more often than once every 8 minutes.
// ResetStreamKey      | Reset Channel Stream Key         | Deletes the stream key for a specified channel. Once it is deleted, the stream key is automatically reset.
// GetCommunity        | Get Channel Community            | Gets the community for a specified channel.
// SetCommunity        | Set Channel Community            | Sets a specified channel to be in a specified community.
// DeleteFromCommunity | Delete Channel from Community    | Deletes a specified channel from its community.

// Channel - Channel Data
type Channel struct {
	ID          string `json:"_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`

	BroadcasterLanguage string `json:"broadcaster_language,omitempty"` // : "en"

	Game               string `json:"game,omitempty"`                            // : "Final Fantasy XV"
	Language           string `json:"language,omitempty"`                        // : "en"
	Mature             string `json:"mature,omitempty"`                          // : true
	Partner            string `json:"partner,omitempty"`                         // : false
	ProfileBanner      string `json:"profile_banner,omitempty"`                  // : null
	ProfileBannerColor string `json:"profile_banner_background_color,omitempty"` // : null
	Status             string `json:"status,omitempty"`                          // : "The Finalest of Fantasies"
	Url                string `json:"url,omitempty"`                             // : "https://www.twitch.tv/dallas"
	VideoBanner        string `json:"video_banner,omitempty"`                    // : null

	Followers int `json:"followers,omitempty"` // : 40
	Views     int `json:"views,omitempty"`     // : 232

	Logo string `json:"logo,omitempty"` // "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",

	CreatedAtString string `json:"created_at"` // 2013-06-03T19:12:02Z
	UpdatedAtStr    string `json:"updated_at"` // 2016-12-14T01:01:44Z
}

// ChannelFull - Full Channel details can only get self
type ChannelFull struct {
	*Channel

	Email     string `json:"email,omitempty"`      // "email-address@provider.com",
	StreamKey string `json:"stream_key,omitempty"` // "live_44322889_nCGwsCl38pt21oj4UJJZbFQ9nrVIU5",
}

// ChannelSub - Subscription to Channel
type ChannelSub struct {
    ID          string `json:"_id"`
    CreatedAtString string `json:"created_at"` // 2013-06-03T19:12:02Z
User *User `json:"user"`
}

type ChannelsMethod struct {
	client *Client
}

// GetMe - Get Channel with Full Auth
func (c *ChannelsMethod) GetMe() (*ChannelFull, error) {
	if c.client.scopes[scopeUserRead] == false {
		return nil, fmt.Errorf("Scope Required: %s", scopeChannelRead)
	}

	var channel ChannelFull
	_, err := c.client.Get("channels", &channel)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

// Get - Get Channel by ID
func (c *ChannelsMethod) Get(id string) (*Channel, error) {
	var channel Channel
	_, err := c.client.Get("channels/"+id, &channel)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

// func (c *ChannelsMethod) Update(id string) (*Channel , error)              {}

func (c *ChannelsMethod) GetEditors(id string) ([]User, error) {
	if c.client.scopes[scopeUserRead] == false {
		return nil, fmt.Errorf("Scope Required: %s", scopeChannelRead)
	}

	editorList := struct {
		Total    int    `json:"_total,omitempty"`
		UserList []User `json:"users"`
	}{}

	_, err := c.client.Get("channels/"+id+"/editors", &editorList)
	if err != nil {
		return nil, err
	}

	return editorList.UserList, nil
}

func (c *ChannelsMethod) GetFollowers(id string) (*Channel, error)   ([]User, error) {
	if c.client.scopes[scopeUserRead] == false {
		return nil, fmt.Errorf("Scope Required: %s", scopeChannelRead)
	}

	editorList := struct {
		Total    int    `json:"_total,omitempty"`
		UserList []User `json:"users"`
	}{}

	_, err := c.client.Get("channels/"+id+"/editors", &editorList)
	if err != nil {
		return nil, err
	}

	return editorList.UserList, nil
}

// GetSubscribers - Get all subs to channel id 
// limit - negative limit will get all subs
func (c *ChannelsMethod) GetSubscribers(id string, limit int) (*Channel, error)  ([]User, error) {
	if c.client.scopes[scopeUserRead] == false {
		return nil, fmt.Errorf("Scope Required: %s", scopeChannelRead)
	}

    if(limit < 0) { 
        reqPageLimit = 100
    } else {

    // Only support up to pageLimit
    reqPageLimit := limit
    if reqPageLimit > pageLimit {
        reqPageLimit = pageLimit
    }
    }

	userList := struct {
		Total    int    `json:"_total,omitempty"`
		UserList []User `json:"subscriptions"`
	}{}

	_, err := c.client.Get("channels/"+id+"/editors", &editorList)
	if err != nil {
		return nil, err
	}

	return editorList.UserList, nil
}

// func (c *ChannelsMethod) GetTeams(id string) (*Channel , error)            {}
// func (c *ChannelsMethod) IsUserGetSubscribed(id string) (*Channel , error) {}
// func (c *ChannelsMethod) GetVideos(id string) (*Channel , error)           {}
// func (c *ChannelsMethod) StartCommercial(id string) (*Channel , error)     {}
// func (c *ChannelsMethod) ResetStreamKey(id string) (*Channel , error)      {}
// func (c *ChannelsMethod) GetCommunity(id string) (*Channel , error)        {}
// func (c *ChannelsMethod) SetCommunity(id string) (*Channel , error)        {}
// func (c *ChannelsMethod) DeleteFromCommunity(id string) (*Channel , error) {}
