package twitch

import "testing"
import "encoding/json"

const (
	testDeadStreamData = `{"stream": null}`

	testStreamData = `{
	"stream": {
		"_id": 24810420256,
		"game": "Horizon Zero Dawn",
		"community_id": "",
		"viewers": 11219,
		"video_height": 1080,
		"average_fps": 60.936497755,
		"delay": 0,
		"created_at": "2017-03-17T11:55:50Z",
		"is_playlist": false,
		"preview": {
			"small": "https://static-cdn.jtvnw.net/previews-ttv/live_user_cohhcarnage-80x45.jpg",
			"medium": "https://static-cdn.jtvnw.net/previews-ttv/live_user_cohhcarnage-320x180.jpg",
			"large": "https://static-cdn.jtvnw.net/previews-ttv/live_user_cohhcarnage-640x360.jpg",
			"template": "https://static-cdn.jtvnw.net/previews-ttv/live_user_cohhcarnage-{width}x{height}.jpg"
		},
		"channel": {
			"mature": false,
			"status": "Horizon: Zero Dawn! \\o/ - Mass Effect: Andromeda continues at 2pm EDT! - @CohhCarnage - !Achievements - !4Year",
			"broadcaster_language": "en",
			"display_name": "CohhCarnage",
			"game": "Horizon Zero Dawn",
			"language": "en",
			"_id": 26610234,
			"name": "cohhcarnage",
			"created_at": "2011-12-06T18:20:34Z",
			"updated_at": "2017-03-17T15:35:07Z",
			"partner": true,
			"logo": "https://static-cdn.jtvnw.net/jtv_user_pictures/cohhcarnage-profile_image-92dc409e41560047-300x300.png",
			"video_banner": "https://static-cdn.jtvnw.net/jtv_user_pictures/cohhcarnage-channel_offline_image-6007ac3e62b7357a-1920x1080.png",
			"profile_banner": "https://static-cdn.jtvnw.net/jtv_user_pictures/cohhcarnage-profile_banner-bcb1b1b8e6194799-480.png",
			"profile_banner_background_color": null,
			"url": "https://www.twitch.tv/cohhcarnage",
			"views": 54100214,
			"followers": 743113
		}
	}
}`
)

func TestStreamParse(t *testing.T) {
	resp := struct {
		Body StreamBody `json:"stream"`
	}{}

	err := json.Unmarshal([]byte(testStreamData), &resp)
	if err != nil {
		t.Log("LIVE", err)
		t.Fail()
	}

	err = json.Unmarshal([]byte(testDeadStreamData), &resp)
	if err != nil {
		t.Log("DEAD", err)
		t.Fail()
	}
}
