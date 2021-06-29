package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/db"
)

// This number of operations to pipeline to redis (to save on round trip time).
var pipelineSize = 10000

type DictionaryFormat string

const (
	PubchemDictionaryFormat  DictionaryFormat = "pubchem"
	LeadmineDictionaryFormat DictionaryFormat = "leadmine"
)

// config structure
type conf struct {
	LogLevel       string           `mapstructure:"log_level"`
	Dictionary struct {
		Name string
		Path string
		Format DictionaryFormat
	}
	Redis          db.RedisConfig
}

var config conf

func init() {
	// initialise config with defaults.
	err := lib.InitializeConfig(map[string]interface{}{
		"log_level":       "info",
		"dictionary": map[string]interface{}{
			"name": "pubchem_synonyms",
			"path": "./dictionaries/pubchem.tsv",
			"format": PubchemDictionaryFormat,
		},
		"redis": map[string]interface{}{
			"host": "localhost",
			"port": 6379,
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

	// Get a redis client
	dbClient := db.NewRedisClient(config.Redis)

	absPath := config.Dictionary.Path
	if !filepath.IsAbs(absPath) {
		_, thisFile, _, _ := runtime.Caller(0)
		thisDirectory := path.Dir(thisFile)
		absPath = filepath.Join(thisDirectory, config.Dictionary.Path)
	}

	dict, err := os.Open(absPath)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	switch config.Dictionary.Format {
	case PubchemDictionaryFormat:
		err = uploadPubchemDictionary(config.Dictionary.Name, dict, dbClient)
	case LeadmineDictionaryFormat:
		err = uploadLeadmineDictionary(config.Dictionary.Name, dict, dbClient)
	}
	if err != nil {
		log.Fatal().Err(err).Send()
	}
}

func uploadLeadmineDictionary(name string, dict *os.File, dbClient db.Client) error {

	pipe := dbClient.NewSetPipeline(pipelineSize)
	scn := bufio.NewScanner(dict)
	for scn.Scan() {
		line := scn.Text()
		uncommented := strings.Split(line, "#")
		if len(uncommented[0]) > 0 {
			record := strings.Split(uncommented[0], "\t")
			resolvedEntity := strings.TrimSpace(record[len(record)-1])
			if resolvedEntity == "" {
				continue
			}
			if len(record) == 1 {
				b, err := json.Marshal(&db.Lookup{
					Dictionary: name,
				})
				if err != nil {
					return err
				}

				pipe.Set(strings.TrimSpace(record[0]), b)
				continue
			}
			for _, key := range record[:len(record)-1] {
				if key == "" {
					continue
				}
				b ,err := json.Marshal(&db.Lookup{
					Dictionary:     name,
					ResolvedEntities: []string{resolvedEntity},
				})
				if err != nil {
					return err
				}
				pipe.Set(strings.TrimSpace(key), b)
			}
		}
		if pipe.Size() > pipelineSize {
			if err := pipe.ExecSet(); err != nil {
				return err
			}
			pipe = dbClient.NewSetPipeline(pipelineSize)
		}
	}
	if pipe.Size() > 0 {
		return pipe.ExecSet()
	}
	return nil
}

func uploadPubchemDictionary(name string, dict *os.File, dbClient db.Client) error {
	pipe := dbClient.NewSetPipeline(pipelineSize)

	scn := bufio.NewScanner(dict)
	currentId := -1
	row := 0
	redisKeys := 0
	var synonyms []string
	var identifiers []string
	for scn.Scan() {
		row++
		line := scn.Text()
		entries := strings.Split(line, "\t")
		if len(entries) != 2 {
			log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid row in dictionary tsv")
			continue
		}

		pubchemId, err := strconv.Atoi(entries[0])
		if err != nil {
			log.Warn().Int("row", row).Strs("entries", entries).Msg("invalid pubchem id")
			continue
		}

		var synonym string
		var identifier string
		if isIdentifier(entries[1]) {
			identifier = entries[1]
		} else {
			synonym = entries[1]
		}

		if pubchemId != currentId {
			if currentId != -1 {
				// Mid process, some stuff to do
				for _, s := range synonyms {
					b, err := json.Marshal(db.Lookup{
						Dictionary:       name,
						ResolvedEntities: identifiers,
					})
					if err != nil {
						return err
					}
					pipe.Set(s, b)
					redisKeys++
				}

				if pipe.Size() > pipelineSize {
					log.Info().Int("row", row).Int("keys", redisKeys).Msg("Upserting dictionary to redis...")
					if err := pipe.ExecSet(); err != nil {
						return err
					}
					pipe = dbClient.NewSetPipeline(pipelineSize)
				}

				synonyms = []string{}
				identifiers = []string{}
			}

			// Set new current id
			currentId = pubchemId
			if synonym != "" {
				synonyms = append(synonyms, synonym)
			} else {
				identifiers = append(identifiers, fmt.Sprintf("PUBCHEM:%d", pubchemId))
				identifiers = append(identifiers, identifier)
			}
		} else {
			if synonym != "" {
				synonyms = append(synonyms, synonym)
			} else {
				identifiers = append(identifiers, identifier)
			}
		}
	}

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
