package dict

import (
	"fmt"
	"os"
)

type Entry struct {
	Synonyms []string
	Identifiers []string
}


type Format string

const (
	PubchemDictionaryFormat  Format = "pubchem"
	LeadmineDictionaryFormat Format = "leadmine"
)

type DictReader interface {
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