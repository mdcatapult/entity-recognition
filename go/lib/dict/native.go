package dict

import (
	"bufio"
	"encoding/json"
	"os"
)

func NewNativeReader() Reader {
	return nativeReader{}
}

type nativeReader struct{}

func (p nativeReader) Read(file *os.File) (chan Entry, chan error) {
	entries := make(chan Entry)
	errors := make(chan error)
	go p.read(file, entries, errors)
	return entries, errors
}

func (p nativeReader) read(dict *os.File, entries chan Entry, errors chan error) {
	scn := bufio.NewScanner(dict)
	for scn.Scan() {
		var e Entry
		if err := json.Unmarshal(scn.Bytes(), &e); err != nil {
			errors <- err
			return
		}
		entries <- e
	}
}
