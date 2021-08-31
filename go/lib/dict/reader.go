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
	Read(file *os.File) (chan *Entry, chan error)
}

func Read(format Format, file *os.File) (chan *Entry, chan error, error) {
	switch format {
	case PubchemDictionaryFormat:
		lCh, eCh :=  NewPubchemReader().Read(file)
		return lCh, eCh, nil
	case LeadmineDictionaryFormat:
		lCh, eCh := NewLeadmineReader().Read(file)
		return lCh, eCh, nil
	default:
		return nil, nil, fmt.Errorf("unsupported dictionary format %v", format)
	}
}

func ReadWithCallback(format Format, callback func(entry *Entry) error, file *os.File) error {
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
			if err := callback(entry); err != nil {
				return err
			}
		}
	}

	return callback(nil)
}