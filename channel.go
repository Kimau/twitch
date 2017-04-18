package twitch

import (
	"fmt"
	"time"
)

const (
	// SoftCapOnFollowGrab - Implicit Soft Cap on Follower requests can be overridden by specificy a limit but in the case of a -1 it will be set to this
	SoftCapOnFollowGrab = 2000
)

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
	ID          int    `json:"_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`

	BroadcasterLanguage string `json:"broadcaster_language,omitempty"` // : "en"

	Game               string `json:"game,omitempty"`                            // : "Final Fantasy XV"
	Language           string `json:"language,omitempty"`                        // : "en"
	Mature             bool   `json:"mature,omitempty"`                          // : true
	Partner            bool   `json:"partner,omitempty"`                         // : false
	ProfileBanner      string `json:"profile_banner,omitempty"`                  // : null
	ProfileBannerColor string `json:"profile_banner_background_color,omitempty"` // : null
	Status             string `json:"status,omitempty"`                          // : "The Finalest of Fantasies"
	URL                string `json:"url,omitempty"`                             // : "https://www.twitch.tv/dallas"
	VideoBanner        string `json:"video_banner,omitempty"`                    // : null

	Followers int `json:"followers,omitempty"` // : 40
	Views     int `json:"views,omitempty"`     // : 232

	Logo string `json:"logo,omitempty"` // "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",

	CreatedAtString string `json:"created_at"` // 2013-06-03T19:12:02Z
	UpdatedAtString string `json:"updated_at"` // 2016-12-14T01:01:44Z
}

// ChannelFull - Full Channel details can only get self
type ChannelFull struct {
	*Channel

	Email     string `json:"email,omitempty"`      // "email-address@provider.com",
	StreamKey string `json:"stream_key,omitempty"` // "live_44322889_nCGwsCl38pt21oj4UJJZbFQ9nrVIU5",
}

// ChannelRelationship - Useful for Queries
type ChannelRelationship struct {
	ID              string   `json:"_id"`
	CreatedAtString string   `json:"created_at"` // 2013-06-03T19:12:02Z
	Notifications   bool     `json:"notifications"`
	User            *User    `json:"user,omitempty"`
	Channel         *Channel `json:"channel,omitempty"`
}

// ChannelFollow - Follow Relationship
type ChannelFollow ChannelRelationship

// ChannelSub - Sub Relationship
type ChannelSub ChannelRelationship

// ChannelsMethod - The functions for Channels
type ChannelsMethod struct {
	client *Client
	au     *UserAuth
}

// GetMe - Get Channel with Full Auth
func (c *ChannelsMethod) GetMe() (*ChannelFull, error) {
	err := c.au.checkScope(scopeChannelRead)
	if err != nil {
		return nil, err
	}

	var channel ChannelFull
	_, err = c.client.Get(c.au, "channels", &channel)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

// Get - Get Channel by ID
func (c *ChannelsMethod) Get(id ID) (*Channel, error) {
	var channel Channel
	_, err := c.client.Get(c.au, fmt.Sprintf("channels/%s", id), &channel)
	if err != nil {
		return nil, err
	}

	return &channel, nil
}

// GetEditors - Return list of users allow to edit the channel
func (c *ChannelsMethod) GetEditors(id ID) ([]User, error) {
	err := c.au.checkScope(scopeChannelRead)
	if err != nil {
		return nil, err
	}

	uList := struct {
		Total    int    `json:"_total"`
		UserList []User `json:"users"`
	}{}

	_, err = c.client.Get(c.au, fmt.Sprintf("channels/%s/editors", id), &uList)
	if err != nil {
		return nil, err
	}

	if uList.Total != 1 {
		return nil, fmt.Errorf("Total Number of Users was: %d", uList.Total)
	}

	return uList.UserList, nil
}

// GetFollowers - Returns the Followers for a Channel
func (c *ChannelsMethod) GetFollowers(id ID, limit int, newestFirst bool) ([]ChannelFollow, int, error) {

	// Can be overridden by passing explicit limit
	if limit < 0 {
		limit = SoftCapOnFollowGrab
	}

	// Page Limit
	reqPageLimit := limit
	if limit < 0 {
		reqPageLimit = pageLimit
	} else if reqPageLimit > pageLimit { // Only support up to pageLimit
		reqPageLimit = pageLimit
	}

	compiledList := []ChannelFollow{}
	cursor := ""
	followersTotal := 0

RequestLoop:
	for len(compiledList) < limit {
		followList, err := c.getFollowersWithCursor(id, reqPageLimit, newestFirst, cursor)
		if err != nil {
			return compiledList, followersTotal, err
		}

		cursor = followList.Cursor
		followersTotal = followList.Total
		compiledList = append(compiledList, followList.Follows...)

		// Do we have all followers
		if len(compiledList) >= followersTotal {
			break RequestLoop
		}
	}

	return compiledList, followersTotal, nil
}

// GetAllFollowersSlow - Returns a Channel which Squirts followers
func (c *ChannelsMethod) GetAllFollowersSlow(id ID, delay time.Duration, newestFirst bool) chan []ChannelFollow {

	chanToSquirtFollowersOut := make(chan []ChannelFollow, 10)

	squirtTicker := time.NewTicker(delay)
	go func() {
		cursor := ""
		for range squirtTicker.C {
			followList, err := c.getFollowersWithCursor(id, pageLimit, newestFirst, cursor)
			if err != nil || cursor == followList.Cursor {
				close(chanToSquirtFollowersOut)
				return
			}

			cursor = followList.Cursor
			chanToSquirtFollowersOut <- followList.Follows
		}
	}()

	return chanToSquirtFollowersOut
}

type followResponse struct {
	Total   int             `json:"_total,omitempty"`
	Cursor  string          `json:"_cursor,omitempty"`
	Follows []ChannelFollow `json:"follows"`
}

// getFollowersWithCursor - Let's client space out requests for doing things like renewing all followers
func (c *ChannelsMethod) getFollowersWithCursor(id ID, limit int, newestFirst bool, cursor string) (followResponse, error) {
	followList := followResponse{
		Cursor: cursor,
	}

	// Check Limit Range
	if limit < 0 {
		limit = pageLimit
	} else if limit > pageLimit {
		limit = pageLimit
	}

	// Request Order
	reqOrder := "asc"
	if newestFirst {
		reqOrder = "desc"
	}

	// Actual Request
	reqURL := fmt.Sprintf("channels/%s/follows?limit=%d&direction=%s",
		id, limit, reqOrder)
	if len(followList.Cursor) > 0 {
		reqURL += fmt.Sprintf("&cursor=%s", followList.Cursor)
	}

	_, err := c.client.Get(c.au, reqURL, &followList)
	return followList, err
}

// getFollowersWithOffset - Let's client space out requests for doing things like renewing all followers
func (c *ChannelsMethod) getFollowersWithOffset(id ID, limit int, newestFirst bool, offset int) (followResponse, error) {
	followList := followResponse{}

	// Check Limit Range
	if limit < 0 {
		limit = pageLimit
	} else if limit > pageLimit {
		limit = pageLimit
	}

	// Request Order
	reqOrder := "asc"
	if newestFirst {
		reqOrder = "desc"
	}

	// Actual Request
	reqURL := fmt.Sprintf("channels/%s/follows?limit=%d&direction=%s",
		id, limit, reqOrder)
	if offset > 0 {
		reqURL += fmt.Sprintf("&offset=%d", offset)
	}

	_, err := c.client.Get(c.au, reqURL, &followList)
	return followList, err
}

// GetSubscribers - Get all subs to channel id
// limit - negative limit will get all subs
func (c *ChannelsMethod) GetSubscribers(id string, limit int, newestFirst bool) ([]ChannelSub, int, error) {
	err := c.au.checkScope(scopeChannelRead)
	if err != nil {
		return nil, 0, err
	}

	reqPageLimit := limit
	if limit < 0 {
		reqPageLimit = 100
	} else if reqPageLimit > pageLimit {
		// Only support up to pageLimit
		reqPageLimit = pageLimit
	}

	reqOrder := "asc"
	if newestFirst {
		reqOrder = "desc"
	}

	subList := struct {
		Total int          `json:"_total,omitempty"`
		Subs  []ChannelSub `json:"subscriptions"`
	}{}

	compiledList := []ChannelSub{}

	offset := 0
	for offset < limit {

		_, err := c.client.Get(c.au,
			fmt.Sprintf("channels/%s/subscriptions?limit=%d&offset=%ds&direction=%s",
				id, reqPageLimit, offset, reqOrder), &subList)
		if err != nil {
			return nil, 0, err
		}

		compiledList = append(compiledList, subList.Subs...)

		if len(subList.Subs) < reqPageLimit {
			return compiledList, subList.Total, err
		}

		offset += reqPageLimit
	}

	return compiledList, subList.Total, nil
}

// func (c *ChannelsMethod) GetTeams(id string) (*Channel , error)            {}
// func (c *ChannelsMethod) IsUserGetSubscribed(id string) (*Channel , error) {}
// func (c *ChannelsMethod) GetVideos(id string) (*Channel , error)           {}
// func (c *ChannelsMethod) StartCommercial(id string) (*Channel , error)     {}
// func (c *ChannelsMethod) ResetStreamKey(id string) (*Channel , error)      {}
// func (c *ChannelsMethod) GetCommunity(id string) (*Channel , error)        {}
// func (c *ChannelsMethod) SetCommunity(id string) (*Channel , error)        {}
// func (c *ChannelsMethod) DeleteFromCommunity(id string) (*Channel , error) {}

/******************************************************************************
			Channel
******************************************************************************/

// CreatedAt - Parses the internal String
func (c Channel) CreatedAt() time.Time {
	t, err := time.Parse(time.RFC3339, c.CreatedAtString)
	if err != nil {
		panic(err)
	}

	return t
}

// UpdatedAt - Parses the internal String
func (c Channel) UpdatedAt() time.Time {
	t, err := time.Parse(time.RFC3339, c.UpdatedAtString)
	if err != nil {
		panic(err)
	}

	return t
}

/******************************************************************************
			Channel Relationship
******************************************************************************/

// CreatedAt - Parses the internal String
func (cr ChannelRelationship) CreatedAt() time.Time {
	t, err := time.Parse(time.RFC3339, cr.CreatedAtString)
	if err != nil {
		panic(err)
	}

	return t
}
