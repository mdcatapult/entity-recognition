package dict

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func NewLeadmineReader() Reader {
	return leadmineReader{}
}

type leadmineReader struct{}

func (l leadmineReader) Read(dict *os.File) (chan Entry, chan error) {
	log.Info().Msg("leadminer reader active!")
	entries := make(chan Entry)
	errors := make(chan error)
	go l.read(dict, entries, errors)
	return entries, errors
}

func (l leadmineReader) read(dict *os.File, entries chan Entry, errors chan error) {

	// Instantiate variables we need to keep track of across lines.
	scn := bufio.NewScanner(dict)

	for scn.Scan() {
		line := scn.Text()

		log.Info().Msg(fmt.Sprint("reading", line))
		// skip empty lines and commented out lines.
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		row := strings.Split(line, "\t")

		// The identifier is the last entry, other entries are synonyms.
		identifier := row[len(row)-1]
		synonyms := row[:len(row)-1]

		// Create a redis lookup for each synonym.
		entries <- Entry{
			Synonyms:    synonyms,
			Identifiers: map[string]string{identifier: ""},
		}
	}
	errors <- nil
}
