package twitch

import (
	"fmt"
)

/*  v5 User Calls
GetMe       | Get User                           | Gets a user object based on the OAuth token provided.
Get         | Get User by ID                     | Gets a specified user object.
EmoteList   | Get User Emotes                    | Gets a list of the emojis and emoticons that the specified user can use in chat
IsSubTo     | Check User Subscription by Channel | Checks if a specified user is subscribed to a specified channel. Intended for viewers.
FollowList  | Get User Follows                   | Gets a list of all channels followed by a specified user, sorted by the date when they started following each channel.
IsFollowing | Check User Follows by Channel      | Checks if a specified user follows a specified channel. If the user is following the channel, a follow object is returned.
Follow      | Follow Channel                     | Adds a specified user to the followers of a specified channel.
Unfollow    | Unfollow Channel                   | Deletes a specified user from the followers of a specified channel.
BlockList   | Get User Block List                | Gets a user’s block list.
Block       | Block User                         | Blocks the target user.
Unblock     | Unblock User                       | Unblocks the target user.
*/

// User - Twitch User
type User struct {
	ID          string `json:"_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Bio         string `json:"bio,omitempty"` // "Just a gamer playing games and chatting. :)"

	Logo     string `json:"logo,omitempty"` // "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",
	UserType string `json:"type,omitempty"` // staff

	CreatedAtString string `json:"created_at"` // 2013-06-03T19:12:02Z
	UpdatedAtStr    string `json:"updated_at"` // 2016-12-14T01:01:44Z
}

type UserNotification struct {
	Email bool `json:"email,omitempty"`
	Push  bool `json:"push,omitempty"`
}

// UserFull can only be fetched for OAuth User
type UserFull struct {
	*User

	Email            string           `json:"email,omitempty"`
	EmailIsVerified  bool             `json:"email_verified,omitempty"`
	Partnered        bool             `json:"partnered,omitempty"`
	TwitterConnected bool             `json:"twitter_connected,omitempty"`
	Notification     UserNotification `json:"notifications,omitempty"`
}

type Emote struct {
	Code string `json:"code,omitempty"`
	ID   int    `json:"id,omitempty"`
}

type EmoteList []Emote
type EmoteSets struct {
	SetMap map[string]EmoteList `json:"emoticon_sets,omitempty"`
}

type UsersMethod struct {
	client *Client
}

// GetMe - Get OAuth User Details
func (u *UsersMethod) GetMe() (*UserFull, error) {
	if u.client.scopes[scopeUserRead] == false {
		return nil, fmt.Errorf("Scope Required: %s", scopeUserRead)
	}

	var user UserFull
	_, err := u.client.Get("user", &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Get - Get User by ID
func (u *UsersMethod) Get(id string) (*User, error) {
	var user User
	_, err := u.client.Get("users/"+id, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByName - Get User by v3 Name
func (u *UsersMethod) GetByName(name string) (*User, error) {

	uList := struct {
		Total    int    `json:"_total"`
		UserList []User `json:"users"`
	}{}

	_, err := u.client.Get("users?login="+name, &uList)
	if err != nil {
		return nil, err
	}

	if uList.Total != 1 {
		return nil, fmt.Errorf("Total Number of Users was: %d", uList.Total)
	}

	return &uList.UserList[0], nil
}

// EmoteList - Get User Emotes
func (u *UsersMethod) EmoteList(id string) (*EmoteList, error) {
	if u.client.scopes[scopeUserSubscriptions] == false {
		return nil, fmt.Errorf("Scope Required: %s", scopeUserSubscriptions)
	}

	var eList EmoteList
	_, err := u.client.Get("users/"+id+"/emotes", &eList)
	if err != nil {
		return nil, err
	}

	return &eList, nil
}

/*
// IsSubTo - Check User Subscription by Channel
func (u *UsersMethod) IsSubTo(id string) error {}

// FollowList - Get User Follows
func (u *UsersMethod) FollowList(id string) error {}

// IsFollowing - Check User Follows by Channel
func (u *UsersMethod) IsFollowing(id string) error {}

// Follow - Follow Channel
func (u *UsersMethod) Follow(id string) error {}

// Unfollow - Unfollow Channel
func (u *UsersMethod) Unfollow(id string) error {}

// BlockList - Get User Block List
func (u *UsersMethod) BlockList(id string) error {}

// Block - Block User
func (u *UsersMethod) Block(id string) error {}

// Unblock - Unblock User
func (u *UsersMethod) Unblock(id string) error {}
*/
