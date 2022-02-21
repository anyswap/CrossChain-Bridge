package data

import (
	"io"
)

type Reader interface {
	io.ByteScanner
	io.Reader
	Len() int
}

type LimitByteReader struct {
	R Reader // underlying reader
	N int64  // max bytes remaining
}

func LimitedByteReader(r Reader, n int64) *LimitByteReader {
	return &LimitByteReader{r, n}
}

func (l *LimitByteReader) Len() int {
	return int(l.N)
}

func NewVariableByteReader(r Reader) (Reader, error) {
	if length, err := readVariableLength(r); err != nil {
		return nil, err
	} else {
		return LimitedByteReader(r, int64(length)), nil
	}
}

func (l *LimitByteReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

func (l *LimitByteReader) ReadByte() (c byte, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	l.N--
	return l.R.ReadByte()
}

func (l *LimitByteReader) UnreadByte() error {
	if err := l.UnreadByte(); err != nil {
		return err
	}
	l.N++
	return nil
}
