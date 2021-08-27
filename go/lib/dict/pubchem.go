package dict

import (
	"bufio"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func NewPubchemReader() DictReader {
	return pubchemReader{}
}

type pubchemReader struct {}

func (p pubchemReader) Read(file *os.File) (chan *Entry, chan error) {
	lookupChan := make(chan *Entry)
	errChan := make(chan error)
	go p.read(file, lookupChan, errChan)
	return lookupChan, errChan
}

func (p pubchemReader) read(dict *os.File, lookupChan chan *Entry, errChan chan error) {

	// Instantiate variables we need to keep track of across lines.
	scn := bufio.NewScanner(dict)
	currentId := -1
	row := 0
	var synonyms []string
	var identifiers []string

	for scn.Scan() {
		row++
		line := scn.Text()

		// Split by tab to get a slice of length 2.
		entries := strings.Split(line, "\t")
		if len(entries) != 2 {
			log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid row in dictionary tsv")
			continue
		}

		// Ensure the pubchem id is an int.
		pubchemId, err := strconv.Atoi(entries[0])
		if err != nil {
			log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid pubchem id")
			continue
		}

		if pubchemId == currentId && isIdentifier(entries[1]) {
			// Same id and value is an identifier.
			identifiers = append(identifiers, entries[1])
		} else if pubchemId == currentId {
			// Same id and value is not an identifier.
			synonyms = append(synonyms, entries[1])
		} else if row != 1 {
			// Different id, add synonyms & identifiers to pipeline.
			lookupChan <- &Entry{
				Synonyms:    synonyms,
				Identifiers: identifiers,
			}

			// Reset synonyms and identifiers.
			synonyms = []string{}
			identifiers = []string{}
		}

		if pubchemId != currentId {
			// Different id but only on first line, so nothing to add to the pipeline.
			currentId = pubchemId
			identifiers = append(identifiers, fmt.Sprintf("PUBCHEM:%d", pubchemId))
		}
	}

	errChan <- nil
}

func isIdentifier(thing string) bool {
	for _, re := range chemicalIdentifiers {
		if re.MatchString(thing) {
			return true
		}
	}
	return false
}

var chemicalIdentifiers = []*regexp.Regexp{
	regexp.MustCompile(`^SCHEMBL\d+$`),
	regexp.MustCompile(`^DTXSID\d{8}$`),
	regexp.MustCompile(`^CHEMBL\d+$`),
	regexp.MustCompile(`^CHEBI:\d+$`),
	regexp.MustCompile(`^LMFA\d{8}$`),
	regexp.MustCompile(`^HY-\d+?[A-Z]?$`),
	regexp.MustCompile(`^CS-.*$`),
	regexp.MustCompile(`^FT-\d{7}$`),
	regexp.MustCompile(`^Q\d+$`),
	regexp.MustCompile(`^ACMC-\w+$`),
	regexp.MustCompile(`^ALBB-\d{6}$`),
	regexp.MustCompile(`^AKOS\d{9}$`),
	regexp.MustCompile(`^\d+-\d+-\d+$`),
	regexp.MustCompile(`^EINCES\s\d+-\d+-\d+$`),
	regexp.MustCompile(`^EC\s\d+-\d+-\d+$`),
}
