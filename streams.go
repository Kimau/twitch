package twitch

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetStreamByUser     | Get Stream by User      | Gets stream information (the stream object) for a specified user.
// GetLiveStreams      | Get Live Streams        | Gets a list of live streams.
// GetStreamsSummary   | Get Streams Summary     | Gets a summary of live streams.
// GetFeaturedStreams  | Get Featured Streams    | Gets a list of all featured live streams.
// GetFollowedStreams  | Get Followed Streams    | Gets a list of online streams a user is following, based on a specified OAuth token.

// StreamPreview - Container of Thumbnail links
type StreamPreview struct {
	Small    string `json:"small"`    // "https://static-cdn.jtvnw.net/previews-ttv/live_user_dansgaming-80x45.jpg",
	Medium   string `json:"medium"`   // "https://static-cdn.jtvnw.net/previews-ttv/live_user_dansgaming-320x180.jpg",
	Large    string `json:"large"`    // "https://static-cdn.jtvnw.net/previews-ttv/live_user_dansgaming-640x360.jpg",
	Template string `json:"template"` // "https://static-cdn.jtvnw.net/previews-ttv/live_user_dansgaming-{width}x{height}.jpg"
}

// StreamBody - Stream Data
type StreamBody struct {
	ResponseID  int     `json:"_id"`          //  23932774784,
	Game        string  `json:"game"`         //  "BATMAN - The Telltale Series",
	Community   string  `json:"community_id"` //  "Community Name"
	Viewers     int     `json:"viewers"`      //  7254,
	VideoHeight int     `json:"video_height"` //  720,
	AverageFPS  float64 `json:"average_fps"`  //  60.00,
	Delay       int     `json:"delay"`        //  0,
	IsPlaylist  bool    `json:"is_playlist"`  //  false,

	Channel Channel       `json:"channel"`
	Preview StreamPreview `json:"preview"`

	CreatedAtString string `json:"created_at"` // 2013-06-03T19:12:02Z
}

// HostData - Host Instance
type HostData struct {
	HostID          int    `json:"host_id"`           //  649402
	HostLogin       string `json:"host_login"`        //  groundz3r0
	HostDisplayName string `json:"host_display_name"` //  GROUNDZ3R0

	TargetID          int    `json:"target_id"`           //  259266
	TargetLogin       string `json:"target_login"`        //  epicdarksparky
	TargetDisplayName string `json:"target_display_name"` //  EpicDarkspark
}

// StreamsMethod - The functions for Streams
type StreamsMethod struct {
	client *Client
	au     *UserAuth
}

// GetStreamByUser - Get Active Stream of User
func (c *StreamsMethod) GetStreamByUser(id ID) (*StreamBody, error) {

	resp := struct {
		Body StreamBody `json:"stream"`
	}{}

	_, err := c.client.Get(c.au, fmt.Sprintf("streams/%s", id), &resp)
	if err != nil {
		return nil, err
	}

	if resp.Body.ResponseID == 0 {
		return nil, fmt.Errorf("Not Streaming %s", id)
	}

	return &resp.Body, nil
}

// GetHostsByUser - Undocumented api to get hosts
// WARNING :: Undocumented API
func (c *StreamsMethod) GetHostsByUser(id ID) ([]HostData, error) {
	hosts := struct {
		HostList []HostData `json:"hosts"`
	}{}

	resp, err := http.Get(fmt.Sprintf("https://tmi.twitch.tv/hosts?include_logins=1&target=%s", id))
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&hosts)
	if err != nil {
		return nil, err
	}

	return hosts.HostList, nil
}

// func (c *StreamsMethod) GetLiveStreams() ([]*StreamBody, int, error) {}
// func (c *StreamsMethod) GetStreamsSummary() ([]*StreamBody, int, error) {}
// func (c *StreamsMethod) GetFeaturedStreams() ([]*StreamBody, int, error) {}
// func (c *StreamsMethod) GetFollowedStreams() ([]*StreamBody, int, error) {}
