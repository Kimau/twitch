package twitch

import (
	"math/rand"
	"testing"
)

func TestBufferRandom(t *testing.T) {
	clb := makeCircLineBuffer(40)

	testBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9} //, 10, 11, 12, 13, 14, 15}
	for i := 0; i < 1000; i++ {
		clb.Write(testBytes[:rand.Intn(9)])
	}

	output := clb.Bytes()
	p := output[0]
	for i := 1; i < len(output); i++ {
		c := output[i]
		if (c == 0) || (c == (p + 1)) {
			p = c
		} else {
			t.Logf("Corrupted Buffer at %d \n%v", i, output)
			t.Fail()
			return
		}
	}

	t.Logf("Output %d %d %d\n%v\n%v", len(output), clb.readOff, clb.writeOff, clb.buf, output)

}
