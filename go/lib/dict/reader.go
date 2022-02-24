package dict

import (
	"fmt"
	"os"
)

type DictConfig struct {
	Name   string
	Path   string
	Format Format
}

type NerEntry struct {
	Synonyms    []string
	Identifiers map[string]string
	Metadata    map[string]interface{}
}

type SwissProtEntry struct {
	Synonyms    []string
	Identifiers map[string]map[string]string
	Metadata    map[string]map[string]string
}

type Format string

const (
	PubchemDictionaryFormat   Format = "pubchem"
	LeadmineDictionaryFormat  Format = "leadmine"
	NativeDictionaryFormat    Format = "native"
	SwissProtDictionaryFormat Format = "swissprot"
)

type Reader interface {
	Read(file *os.File) (chan NerEntry, chan error)
}

func Read(format Format, file *os.File) (chan NerEntry, chan error, error) {
	switch format {
	case PubchemDictionaryFormat:
		entries, errors := NewPubchemReader().Read(file)
		return entries, errors, nil
	case LeadmineDictionaryFormat:
		entries, errors := NewLeadmineReader().Read(file)
		return entries, errors, nil
	case NativeDictionaryFormat:
		entries, errors := NewNativeReader().Read(file)
		return entries, errors, nil
	case SwissProtDictionaryFormat:
		entries, errors := NewSwissProtReader().Read(file)
		return entries, errors, nil
	default:
		return nil, nil, fmt.Errorf("unsupported dictionary format %v", format)
	}
}

// ReadWithCallback reads the dictionary file according to its format and executes the onEntry callback for each NerEntry.
// The onEOF callback is executed when there are no more entries in the file.
func ReadWithCallback(file *os.File, format Format, onEntry func(entry NerEntry) error, onEOF func() error) error {
	entries, errors, err := Read(format, file)
	if err != nil {
		return err
	}

Listen:
	for {
		select {
		case err := <-errors:
			if err != nil {
				return err
			}
			break Listen
		case entry := <-entries:
			if err := onEntry(entry); err != nil {
				return err
			}
		}
	}

	if onEOF != nil {
		return onEOF()
	}

	return nil
}
