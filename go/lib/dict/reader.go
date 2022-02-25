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

type Entry interface {
	ReplaceSynonymAt(synonym string, index int)
	GetSynonyms() []string
	GetIdentifiers() map[string]interface{}
	GetMetadata() map[string]interface{}
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

func (ne *NerEntry) ReplaceSynonymAt(synonym string, index int) {
	ne.Synonyms[index] = synonym
}

func (ne NerEntry) GetSynonyms() []string {
	return ne.Synonyms
}

func (ne NerEntry) GetIdentifiers() map[string]interface{} {
	res := make(map[string]interface{}, len(ne.Identifiers))
	for k, v := range ne.Identifiers {
		var identiferValue interface{} = v
		res[k] = identiferValue
	}
	return res

}

func (ne NerEntry) GetMetadata() map[string]interface{} {
	return ne.Metadata
}

func (spe *SwissProtEntry) ReplaceSynonymAt(synonym string, index int) {
	spe.Synonyms[index] = synonym
}

func (spe SwissProtEntry) GetSynonyms() []string {
	return spe.Synonyms
}

func (spe SwissProtEntry) GetIdentifiers() map[string]interface{} {
	res := make(map[string]interface{}, len(spe.Identifiers))
	for k, v := range spe.Identifiers {
		var identiferValue interface{} = v
		res[k] = identiferValue
	}
	return res
}

func (spe SwissProtEntry) GetMetadata() map[string]interface{} {
	res := make(map[string]interface{}, len(spe.Metadata))
	for k, v := range spe.Metadata {
		var metadataValue interface{} = v
		res[k] = metadataValue
	}
	return res
}

type Format string

const (
	PubchemDictionaryFormat   Format = "pubchem"
	LeadmineDictionaryFormat  Format = "leadmine"
	NativeDictionaryFormat    Format = "native"
	SwissProtDictionaryFormat Format = "swissprot"
)

type Reader interface {
	Read(file *os.File) (chan Entry, chan error)
}

func Read(format Format, file *os.File) (chan Entry, chan error, error) {
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
func ReadWithCallback(file *os.File, format Format, onEntry func(entry Entry) error, onEOF func() error) error {
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
