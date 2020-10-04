package server

import (
	"io"
	"sync"
)

type syncReader struct {
	mux    sync.Mutex
	reader io.Reader
}

func (sr *syncReader) Read(p []byte) (n int, err error) {
	sr.mux.Lock()
	defer sr.mux.Unlock()

	return sr.reader.Read(p)
}
