package twitch

import (
	"encoding/json"
	"testing"
)

var (
	PubSubMsgExamples = []string{
		`{"type":"MESSAGE","data":{"topic":"whispers.144091363","message":"{\"type\":\"whisper_received\",\"data\":\"{\\\"message_id\\\":\\\"520beeb5-b169-40a8-8446-d4a4f5508733\\\",\\\"id\\\":3,\\\"thread_id\\\":\\\"24181541_144091363\\\",\\\"body\\\":\\\"Pickle\\\",\\\"sent_ts\\\":1494172399,\\\"from_id\\\":24181541,\\\"tags\\\":{\\\"login\\\":\\\"kimau\\\",\\\"display_name\\\":\\\"Kimau\\\",\\\"color\\\":\\\"#C705C0\\\",\\\"user_type\\\":\\\"\\\",\\\"turbo\\\":true,\\\"emotes\\\":[],\\\"badges\\\":[{\\\"id\\\":\\\"premium\\\",\\\"version\\\":\\\"1\\\"}]},\\\"recipient\\\":{\\\"id\\\":144091363,\\\"username\\\":\\\"kimaubot\\\",\\\"display_name\\\":\\\"KimauBot\\\",\\\"color\\\":\\\"\\\",\\\"user_type\\\":\\\"\\\",\\\"turbo\\\":false,\\\"badges\\\":[],\\\"profile_image\\\":null},\\\"nonce\\\":\\\"MgwgoqHm3RLd8vPoVN1V1z9NSEofu9\\\"}\",\"data_object\":{\"message_id\":\"520beeb5-b169-40a8-8446-d4a4f5508733\",\"id\":3,\"thread_id\":\"24181541_144091363\",\"body\":\"Pickle\",\"sent_ts\":1494172399,\"from_id\":24181541,\"tags\":{\"login\":\"kimau\",\"display_name\":\"Kimau\",\"color\":\"#C705C0\",\"user_type\":\"\",\"turbo\":true,\"emotes\":[],\"badges\":[{\"id\":\"premium\",\"version\":\"1\"}]},\"recipient\":{\"id\":144091363,\"username\":\"kimaubot\",\"display_name\":\"KimauBot\",\"color\":\"\",\"user_type\":\"\",\"turbo\":false,\"badges\":[],\"profile_image\":null},\"nonce\":\"MgwgoqHm3RLd8vPoVN1V1z9NSEofu9\"}}"}}`,
	}
)

func ParseMsg(raw string) error {
	msg := PubSubBase{}
	err := json.Unmarshal([]byte(raw), &msg)
	if err != nil {
		return err
	}

	if msg.Type == "MESSAGE" {

		switch msg.Data.Topic.Subject {
		case psChanBits:
			bitData := psBitsMsgData{}
			err = json.Unmarshal([]byte(msg.Data.DataStr), &bitData)

		case psChanSubs:
			subData := psSubMsgData{}
			err = json.Unmarshal([]byte(msg.Data.DataStr), &subData)

		case psVideoPlayback:
			panic("TODO")

		case psChatModActions:
			panic("TODO")

		case psUserWhispers:
			wrapper := psWrapper{}
			err = json.Unmarshal([]byte(msg.Data.DataStr), &wrapper)
			if err != nil {
				return err
			}

			whispData := psWhispMsgData{}
			err = json.Unmarshal([]byte(wrapper.DataStr), &whispData)

		default:
			panic("UNKNOWN")
		}
	}

	return err

}

func TestPubSubMsg(t *testing.T) {
	for i, v := range PubSubMsgExamples {
		err := ParseMsg(v)
		if err != nil {
			t.Logf("Failed PubSubMsg fail %d: %s", i, err.Error())
			t.Fail()
		}

	}
}
