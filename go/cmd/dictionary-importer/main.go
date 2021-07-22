package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
)

type DictionaryFormat string

const (
	PubchemDictionaryFormat  DictionaryFormat = "pubchem"
	LeadmineDictionaryFormat DictionaryFormat = "leadmine"
)

// config structure
type dictionaryImporterConfig struct {
	lib.BaseConfig
	Dictionary struct {
		Name   string
		Path   string
		Format DictionaryFormat
	}
	BackendDatabase db.DictionaryBackend `mapstructure:"dictionary_backend"`
	PipelineSize    int                  `mapstructure:"pipeline_size"`
	Redis           db.RedisConfig
	Elasticsearch   db.ElasticsearchConfig
}

var config dictionaryImporterConfig

func initConfig() {
	// initialise config with defaults.
	err := lib.InitializeConfig("./config/dictionary-importer.yml", map[string]interface{}{
		"log_level":          "info",
		"dictionary_backend": db.RedisDictionaryBackend,
		"pipeline_size":      10000,
		"dictionary": map[string]interface{}{
			"name":   "pubchem_synonyms",
			"path":   "./dictionaries/pubchem.tsv",
			"format": PubchemDictionaryFormat,
		},
		"redis": map[string]interface{}{
			"host": "localhost",
			"port": 6379,
		},
		"elasticsearch": map[string]interface{}{
			"host": "localhost",
			"port": 9200,
			"index": "pubchem",
		},
	})
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	// unmarshal the viper contents into our config struct
	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func main() {
	initConfig()
	// Get a redis client
	var dbClient db.Client
	var err error
	switch config.BackendDatabase {
	case db.RedisDictionaryBackend:
		dbClient = db.NewRedisClient(config.Redis)
	case db.ElasticsearchDictionaryBackend:
		dbClient, err = db.NewElasticsearchClient(config.Elasticsearch)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	default:
		log.Fatal().Msg("invalid backend database type")
	}

	dict, err := os.Open(config.Dictionary.Path)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	for !dbClient.Ready() {
		log.Info().Msg("database is not ready, waiting...")
		time.Sleep(10 * time.Second)
	}

	switch config.Dictionary.Format {
	case PubchemDictionaryFormat:
		err = new(pubchemUploader).uploadDictionary(dict, dbClient)
	case LeadmineDictionaryFormat:
		err = new(leadmineUploader).uploadDictionary(dict, dbClient)
	}
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

type leadmineUploader struct {}

func (l leadmineUploader) uploadDictionary(dict *os.File, dbClient db.Client) error {

	// Instantiate variables we need to keep track of across lines.
	pipe := dbClient.NewSetPipeline(config.PipelineSize)
	scn := bufio.NewScanner(dict)

	for scn.Scan() {
		line := scn.Text()

		// skip empty lines and commented out lines.
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		entries := strings.Split(line, "\t")

		// The identifier is the last entry, other entries are synonyms.
		identifier := entries[len(entries)-1]
		synonyms := entries[:len(entries)-1]

		// Create a redis lookup for each synonym.
		for _, synonym := range synonyms {
			b, err := json.Marshal(&db.Lookup{
				Dictionary:       config.Dictionary.Name,
				ResolvedEntities: []string{identifier},
			})
			if err != nil {
				return err
			}
			pipe.Set(synonym, b)
		}

		// Check if the pipe size has exceeded the config limit.
		// If it has, execute it and reset the pipe.
		if pipe.Size() > config.PipelineSize {
			if err := pipe.ExecSet(); err != nil {
				return err
			}
			pipe = dbClient.NewSetPipeline(config.PipelineSize)
		}
	}

	// Execute the pipe for any remaining synonyms.
	if pipe.Size() > 0 {
		return pipe.ExecSet()
	}
	return nil
}

type pubchemUploader struct {}

func (p pubchemUploader) uploadDictionary(dict *os.File, dbClient db.Client) error {

	// Instantiate variables we need to keep track of across lines.
	pipe := dbClient.NewSetPipeline(config.PipelineSize)
	scn := bufio.NewScanner(dict)
	currentId := -1
	row := 0
	dbEntries := 0
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
			if err := p.addToPipe(synonyms, identifiers, pipe, &dbEntries); err != nil {
				return err
			}

			// If pipe size is big enough execute it.
			if pipe.Size() > config.PipelineSize {
				log.Info().Int("row", row).Int("keys", dbEntries).Msgf("Upserting dictionary to %s...", config.BackendDatabase)
				if err := pipe.ExecSet(); err != nil {
					return err
				}
				pipe = dbClient.NewSetPipeline(config.PipelineSize)
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

	// Execute pipe for any remaining stuff.
	if pipe.Size() > 0 {
		if err := pipe.ExecSet(); err != nil {
			return err
		}
	}

	return nil
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

func (p pubchemUploader) addToPipe(synonyms, identifiers []string, pipe db.SetPipeline, dbEntries *int) error {
	// Mid process, some stuff to do
	switch config.BackendDatabase {
	case db.RedisDictionaryBackend:
		for _, s := range synonyms {
			b, err := json.Marshal(db.Lookup{
				Dictionary:       config.Dictionary.Name,
				ResolvedEntities: identifiers,
			})
			if err != nil {
				return err
			}
			pipe.Set(s, b)
			*dbEntries++
		}
	case db.ElasticsearchDictionaryBackend:
		b, err := json.Marshal(db.EsLookup{
			Dictionary:  config.Dictionary.Name,
			Synonyms:    synonyms,
			Identifiers: identifiers,
		})
		if err != nil {
			return err
		}
		pipe.Set("", b)
		*dbEntries++
	}
	return nil
}