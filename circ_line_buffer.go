package twitch

import "errors"

// Optimised for Writing and frequent small reads
// Reading the entire buffer is a bugger for perf

type circLineBuffer struct {
	size        int
	bufHalfSize int
	readOff     int // Read Offset aligned to nearest end of line
	writeOff    int // Write Offset aligned to nearest end of line
	buf         []byte
}

var (
	// ErrEmptyWrite - Empty write to buffer
	ErrEmptyWrite = errors.New("Cannot write empty slice")

	// ErrWriteTooBig - Can only write half the size of CLB
	ErrWriteTooBig = errors.New("Cannot write more than half the size")
)

func makeCircLineBuffer(bufsize int) *circLineBuffer {

	clb := &circLineBuffer{
		size:        bufsize,
		bufHalfSize: bufsize / 2,
		buf:         make([]byte, bufsize, bufsize),
		readOff:     0,
		writeOff:    0,
	}

	return clb
}

func (clb *circLineBuffer) inc(pos *int) {
	*pos++
	if *pos >= clb.size {
		*pos -= clb.size
	}
}

func (clb *circLineBuffer) Size() int {
	return clb.size
}

func (clb *circLineBuffer) Len() int {
	l := clb.writeOff - clb.readOff
	if l < 0 {
		l += clb.size
	}
	return l
}

func (clb *circLineBuffer) readInteral(pos int, endpos int, dst []byte) {
	wrapCopy := (endpos < pos)

	// Copy Accross
	if wrapCopy {
		mp := clb.size - pos
		copy(dst[0:mp], clb.buf[pos:])
		copy(dst[mp:], clb.buf[0:endpos])
	} else {
		copy(dst, clb.buf[pos:endpos])
	}
}

func (clb *circLineBuffer) writeInternal(pos int, endpos int, src []byte) {
	wrapCopy := (endpos < pos)

	// Copy Accross
	if wrapCopy {
		mp := clb.size - pos
		copy(clb.buf[pos:], src[0:mp])
		copy(clb.buf[0:endpos], src[mp:])
	} else {
		copy(clb.buf[pos:endpos], src)
	}
}

func (clb *circLineBuffer) Bytes() []byte {
	bLen := clb.writeOff - clb.readOff
	if bLen < 0 {
		bLen += clb.size
	}
	outBuf := make([]byte, bLen, bLen)
	clb.readInteral(clb.readOff, clb.writeOff, outBuf)

	return outBuf
}

func (clb *circLineBuffer) String() string {
	return string(clb.Bytes())
}

func (clb *circLineBuffer) Reset() {
	clb.writeOff = 0
	clb.readOff = 0
}

func (clb *circLineBuffer) Write(p []byte) (n int, err error) {
	wl := len(p)
	if wl == 0 {
		return 0, ErrEmptyWrite
	}
	if wl > clb.bufHalfSize {
		return 0, ErrWriteTooBig
	}

	advanceRead := false

	// Update Cursors
	pos := clb.writeOff
	endPos := clb.writeOff + wl
	if endPos >= clb.size {
		endPos -= clb.size
		advanceRead = clb.readOff < endPos
	}

	// Do Actual Write
	clb.writeInternal(pos, endPos, p)

	// If we dont have a zero at the end
	if p[wl-1] != 0 {
		clb.buf[endPos] = 0
		clb.inc(&endPos)
	}
	clb.writeOff = endPos

	// Did we cross read head
	if advanceRead || ((pos-clb.readOff) >= 0) != ((endPos-clb.readOff) > 0) {
		clb.inc(&endPos)

		// Advance 0 to avoid partial string
		for clb.buf[endPos] != 0 {
			clb.buf[endPos] = 0 // Slight perf cost but saves a lot of headaches
			clb.inc(&endPos)
		}
		clb.readOff = endPos
	}

	// fmt.Printf("%2d %2d - %v\n", clb.readOff, clb.writeOff, clb.buf)
	return wl, nil
}

func (clb *circLineBuffer) Read(p []byte) (n int, err error) {
	maxLen := len(p)

	startPos := clb.readOff
	endPos := clb.writeOff

	// Figure out Actual Read Length
	readLen := endPos - startPos
	if readLen < 0 {
		readLen += clb.size
	}
	if maxLen < readLen {
		endPos = startPos + maxLen
		if endPos >= clb.size {
			endPos -= clb.size
		}
	}

	// Update Cursor then Read
	clb.readInteral(startPos, endPos, p)

	return readLen, nil
}
