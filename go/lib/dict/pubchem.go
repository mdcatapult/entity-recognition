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

func NewPubchemReader() Reader {
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
	row := 1
	var synonyms []string
	var identifiers []string

	scn.Scan()
	currentId, firstValue := parseLine(scn.Text(), row)
	identifiers = []string{fmt.Sprintf("PUBCHEM:%d", currentId)}
	if isIdentifier(firstValue) {
		identifiers = append(identifiers, firstValue)
	} else {
		synonyms = append(synonyms, firstValue)
	}

	for scn.Scan() {
		row++
		line := scn.Text()
		id, value := parseLine(line, row)

		if id != currentId {
			lookupChan <- &Entry{
				Synonyms:    synonyms,
				Identifiers: identifiers,
			}
			synonyms = []string{}
			identifiers = []string{fmt.Sprintf("PUBCHEM:%d", id)}
			currentId = id
		}

		if isIdentifier(value) {
			identifiers = append(identifiers, value)
		} else {
			synonyms = append(synonyms, value)
		}
	}

	errChan <- nil
}

func parseLine(line string, row int) (id int, value string) {
	// Split by tab to get a slice of length 2.
	entries := strings.Split(line, "\t")
	if len(entries) != 2 {
		log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid row in dictionary tsv")
	}

	// Ensure the pubchem id is an int.
	pubchemId, err := strconv.Atoi(entries[0])
	if err != nil {
		log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid pubchem id")
	}

	return pubchemId, entries[1]
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
