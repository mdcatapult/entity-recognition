package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/dict"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/text"
)

// config structure
type dictionaryImporterConfig struct {
	lib.BaseConfig
	Dictionary   dict.DictConfig
	PipelineSize int `mapstructure:"pipeline_size"`
	Redis        remote.RedisConfig
}

var defaultConfig = map[string]interface{}{
	"log_level":     "info",
	"pipeline_size": 10000,
	"dictionary": map[string]interface{}{
		"name":   "pubchem_synonyms",
		"path":   "./dictionaries/pubchem.tsv",
		"format": dict.PubchemDictionaryFormat,
	},
	"redis": map[string]interface{}{
		"host": "localhost",
		"port": 6379,
	},
}

var config dictionaryImporterConfig

func main() {

	// initialise config with defaults.
	if err := lib.InitializeConfig("./config/dictionary-importer.yml", defaultConfig, &config); err != nil {
		log.Fatal().Err(err).Send()
	}

	for _, arg := range os.Args {

		if strings.Contains(arg, "=") {
			k := strings.Split(arg, "=")[0]
			v := strings.Split(arg, "=")[1]
			switch k {
			case "dictionaryPath":
				config.Dictionary.Path = v
			case "dictionaryFormat":
				config.Dictionary.Format = dict.Format(v)
			}

		}
	}

	// Get a redis client
	var redisClient = remote.NewRedisClient(config.Redis)
	var err error

	dictFile, err := os.Open(config.Dictionary.Path)
	if err != nil {
		log.Fatal().Str("path", config.Dictionary.Path).Err(err).Send()
	}

	entries := 0
	pipeline := redisClient.NewSetPipeline(config.PipelineSize)
	onEntry := func(entry dict.Entry) error {
		entries++

		if entries%50000 == 0 {
			log.Info().Int("entries", entries).Msg("importing")
		}

		for i, synonym := range entry.GetSynonyms() {
			tokens := strings.Fields(synonym)
			normalizedTokens := make([]string, 0, len(tokens))
			for _, token := range tokens {
				normalizedToken, _ := text.NormalizeAndLowercaseString(token)
				if len(normalizedToken) > 0 {
					normalizedTokens = append(normalizedTokens, normalizedToken)
				}
			}
			entry.ReplaceSynonymAt(strings.Join(normalizedTokens, " "), i)
		}

		if err := addToPipe(entry, pipeline); err != nil {
			return err
		}

		if pipeline.Size() > config.PipelineSize {
			awaitDB(redisClient)
			if err := pipeline.ExecSet(); err != nil {
				return err
			}

			pipeline = redisClient.NewSetPipeline(config.PipelineSize)
		}

		return nil
	}

	onEOF := func() error {
		if pipeline.Size() > 0 {
			return pipeline.ExecSet()
		}

		return nil
	}

	if err := dict.ReadWithCallback(dictFile, config.Dictionary.Format, onEntry, onEOF); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func addToPipe(entry dict.Entry, pipe remote.SetPipeline) error {
	// Mid process, some stuff to do
	for _, synonym := range entry.GetSynonyms() {

		metadata, err := json.Marshal(entry.GetMetadata())
		if err != nil {
			return err
		}

		bytes, err := json.Marshal(cache.Lookup{
			Dictionary:  config.Dictionary.Name,
			Identifiers: entry.GetIdentifiers(),
			Metadata:    metadata,
		})
		if err != nil {
			return err
		}
		pipe.Set(synonym, bytes)
	}
	return nil
}

func awaitDB(dbClient remote.Client) {
	for !dbClient.Ready() {
		log.Info().Msg("database is not ready, waiting...")
		time.Sleep(5 * time.Second)
	}
}
