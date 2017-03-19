package twitch

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

const ()

var ()

// DumpState - Dump the Internal State to File
func (ah *Client) DumpState() error {
	f, err := os.Create(fmt.Sprintf("dump_%s_%d.bin", ah.RoomName, time.Now().Unix()))
	if err != nil {
		return err
	}

	enc := gob.NewEncoder(f)
	for _, v := range ah.Viewers {
		err = enc.Encode(v)
		if err != nil {
			f.Close()
			return err
		}
	}

	fmt.Printf("Dumped data to file: %s", f.Name())
	f.Close()
	return nil
}
