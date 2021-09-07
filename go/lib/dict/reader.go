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

type Entry struct {
	Synonyms []string
	Identifiers []string
}


type Format string

const (
	PubchemDictionaryFormat  Format = "pubchem"
	LeadmineDictionaryFormat Format = "leadmine"
)

type Reader interface {
	Read(file *os.File) (chan Entry, chan error)
}

func Read(format Format, file *os.File) (chan Entry, chan error, error) {
	switch format {
	case PubchemDictionaryFormat:
		entries, errors :=  NewPubchemReader().Read(file)
		return entries, errors, nil
	case LeadmineDictionaryFormat:
		entries, errors := NewLeadmineReader().Read(file)
		return entries, errors, nil
	default:
		return nil, nil, fmt.Errorf("unsupported dictionary format %v", format)
	}
}

// ReadWithCallback reads the dictionary file according to its format and executes the onEntry callback for each Entry.
// The onEOF callback is executed when there are no more entries in the file.
func ReadWithCallback(file *os.File, format Format, onEntry func(entry Entry) error, onEOF func() error) error {
	entries, errors, err := Read(format, file)
	if err != nil {
		return err
	}

	Listen: for {
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