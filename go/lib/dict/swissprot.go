package dict

import (
	"bufio"
	"encoding/json"
	"os"
)

func NewSwissProtReader() Reader {
	return swissProtReader{}
}

type swissProtReader struct{}

func (p swissProtReader) Read(file *os.File) (chan Entry, chan error) {
	entries := make(chan Entry)
	errors := make(chan error)
	go p.read(file, entries, errors)
	return entries, errors
}

func (p swissProtReader) read(dict *os.File, entries chan Entry, errors chan error) {
	scn := bufio.NewScanner(dict)
	for scn.Scan() {
		var e SwissProtEntry
		if err := json.Unmarshal(scn.Bytes(), &e); err != nil {
			errors <- err
			return
		}
		entries <- &e
	}
}