package twitch

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserFull(t *testing.T) {

	xStr := `{
    "_id": "44322889",
    "bio": "Just a gamer playing games and chatting. :)",
    "created_at": "2013-06-03T19:12:02Z",
    "display_name": "dallas",
    "email": "email-address@provider.com",
    "email_verified": true,
    "logo": "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",
    "name": "dallas",
    "notifications": {
        "email": false,
        "push": true
    },
    "partnered": false,
    "twitter_connected": false,
    "type": "staff",
    "updated_at": "2016-12-14T01:01:44Z"
}`

	testUser := UserFull{
		User: &User{
			ID:              "44322889",
			Name:            "dallas",
			DisplayName:     "dallas",
			Bio:             "Just a gamer playing games and chatting. :)",
			Logo:            "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",
			UserType:        "staff",
			CreatedAtString: "2013-06-03T19:12:02Z",
			UpdatedAtStr:    "2016-12-14T01:01:44Z",
		},

		UserPersonal: &UserPersonal{
			Email:            "email-address@provider.com",
			EmailIsVerified:  true,
			Partnered:        false,
			TwitterConnected: false,
			Notification: UserNotification{
				Email: false,
				Push:  true,
			},
		},
	}

	dec := json.NewDecoder(strings.NewReader(xStr))
	var user UserFull
	err := dec.Decode(&user)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if testUser.ID != user.ID {
		t.Fail()
		t.Logf("User [%s:%s] not equal", testUser.ID, user.ID)
	}

	if testUser.Name != user.Name {
		t.Fail()
		t.Logf("Name [%s:%s] not equal", testUser.Name, user.Name)
	}

	if testUser.DisplayName != user.DisplayName {
		t.Fail()
		t.Logf("DisplayName [%s:%s] not equal", testUser.DisplayName, user.DisplayName)
	}

	if testUser.Bio != user.Bio {
		t.Fail()
		t.Logf("Bio [%s:%s] not equal", testUser.Bio, user.Bio)
	}

	if testUser.Logo != user.Logo {
		t.Fail()
		t.Logf("Logo [%s:%s] not equal", testUser.Logo, user.Logo)
	}

	if testUser.UserType != user.UserType {
		t.Fail()
		t.Logf("UserType [%s:%s] not equal", testUser.UserType, user.UserType)
	}

	if testUser.CreatedAtString != user.CreatedAtString {
		t.Fail()
		t.Logf("CreatedAtString [%s:%s] not equal", testUser.CreatedAtString, user.CreatedAtString)
	}

	if testUser.UpdatedAtStr != user.UpdatedAtStr {
		t.Fail()
		t.Logf("UpdatedAtStr [%s:%s] not equal", testUser.UpdatedAtStr, user.UpdatedAtStr)
	}
	if testUser.Email != user.Email {
		t.Fail()
		t.Logf("Email [%s:%s] not equal", testUser.Email, user.Email)
	}
	if testUser.EmailIsVerified != user.EmailIsVerified {
		t.Fail()
		t.Logf("EmailIsVerified [%s:%s] not equal", testUser.EmailIsVerified, user.EmailIsVerified)
	}
	if testUser.Partnered != user.Partnered {
		t.Fail()
		t.Logf("Partnered [%s:%s] not equal", testUser.Partnered, user.Partnered)
	}
	if testUser.TwitterConnected != user.TwitterConnected {
		t.Fail()
		t.Logf("TwitterConnected [%s:%s] not equal", testUser.TwitterConnected, user.TwitterConnected)
	}

	if testUser.Notification.Email != user.Notification.Email {
		t.Fail()
		t.Logf("Notification.Email [%s:%s] not equal", testUser.Notification.Email, user.Notification.Email)
	}
	if testUser.Notification.Push != user.Notification.Push {
		t.Fail()
		t.Logf("Notification.Push [%s:%s] not equal", testUser.Notification.Push, user.Notification.Push)
	}
}
