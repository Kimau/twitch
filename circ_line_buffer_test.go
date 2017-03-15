package twitch

import (
	"math/rand"
	"testing"
)

func createSpamBuffer(t *testing.T) (*circLineBuffer, int) {
	clb := makeCircLineBuffer(40)
	testBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9} //, 10, 11, 12, 13, 14, 15}
	total := 0
	for i := 0; i < 1000; i++ {
		bList := testBytes[:rand.Intn(8)+1]
		w, err := clb.Write(bList)
		if err != nil {
			t.Logf("Failed Write: %s", err)
			t.FailNow()
			return nil, 0
		}

		total += w
	}

	return clb, total
}

func TestBufferRandom(t *testing.T) {
	clb, _ := createSpamBuffer(t)

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

func TestBufferLimit(t *testing.T) {
	clb, total := createSpamBuffer(t)

	t.Logf("Buffer \n%v\n", clb.Bytes())

	line := make([]byte, 15, 15)
	r, err := clb.Read(line)
	for r > 0 {
		t.Log(r)
		t.Logf("READ  %2d:%v", r, line[:r])
		if err != nil {
			t.Logf("Failed Write: %v", err)
			t.Fail()
			return
		}

		total -= r
		r, err = clb.Read(line)
	}
	t.Logf("END    %2d:%v", r, err)
}
