package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/dict"
)

// config structure
type dictionaryImporterConfig struct {
	lib.BaseConfig
	Dictionary      dict.DictConfig
	BackendDatabase cache.Type `mapstructure:"dictionary_backend"`
	PipelineSize    int        `mapstructure:"pipeline_size"`
	Redis           remote.RedisConfig
	Elasticsearch   remote.ElasticsearchConfig
}

var defaultConfig = map[string]interface{}{
	"log_level":          "info",
	"dictionary_backend": cache.Redis,
	"pipeline_size":      10000,
	"dictionary": map[string]interface{}{
		"name":   "pubchem_synonyms",
		"path":   "./dictionaries/pubchem.tsv",
		"format": dict.PubchemDictionaryFormat,
	},
	"redis": map[string]interface{}{
		"host": "localhost",
		"port": 6379,
	},
	"elasticsearch": map[string]interface{}{
		"host":  "localhost",
		"port":  9200,
		"index": "pubchem",
	},
}

var config dictionaryImporterConfig

func main() {

	// initialise config with defaults.
	if err := lib.InitializeConfig("./config/dictionary-importer.yml", defaultConfig, &config); err != nil {
		log.Fatal().Err(err).Send()
	}

	// Get a redis client
	var dbClient remote.Client
	var err error
	switch config.BackendDatabase {
	case cache.Redis:
		dbClient = remote.NewRedisClient(config.Redis)
	case cache.Elasticsearch:
		dbClient, err = remote.NewElasticsearchClient(config.Elasticsearch)
		if err != nil {
			log.Fatal().Err(err).Send()
		}
	default:
		log.Fatal().Str("backend database", string(config.BackendDatabase)).Msg("invalid backend database type")
	}

	dictFile, err := os.Open(config.Dictionary.Path)
	if err != nil {
		log.Fatal().Str("path", config.Dictionary.Path).Err(err).Send()
	}

	for !dbClient.Ready() {
		log.Info().Msg("database is not ready, waiting...")
		time.Sleep(10 * time.Second)
	}

	pipe := dbClient.NewSetPipeline(config.PipelineSize)
	nInsertions := 0
	onEntry := func(entry dict.Entry) error {
		if err := addToPipe(entry, pipe, &nInsertions); err != nil {
			return err
		}

		if pipe.Size() > config.PipelineSize {
			if err := pipe.ExecSet(); err != nil {
				return err
			}

			log.Info().Str("backend", string(config.BackendDatabase)).Int("insertions", nInsertions).Msg("uploading data")

			pipe = dbClient.NewSetPipeline(config.PipelineSize)
		}

		return nil
	}

	onEOF := func() error {
		if pipe.Size() > 0 {
			return pipe.ExecSet()
		}

		return nil
	}

	if err := dict.ReadWithCallback(dictFile, config.Dictionary.Format, onEntry, onEOF); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func addToPipe(entry dict.Entry, pipe remote.SetPipeline, nInsertions *int) error {
	// Mid process, some stuff to do
	switch config.BackendDatabase {
	case cache.Redis:
		for _, s := range entry.Synonyms {
			b, err := json.Marshal(cache.Lookup{
				Dictionary:  config.Dictionary.Name,
				Identifiers: entry.Identifiers,
			})
			if err != nil {
				return err
			}
			pipe.Set(s, b)
			*nInsertions++
		}
	case cache.Elasticsearch:
		b, err := json.Marshal(remote.EsLookup{
			Dictionary:  config.Dictionary.Name,
			Synonyms:    entry.Synonyms,
			Identifiers: entry.Identifiers,
		})
		if err != nil {
			return err
		}
		pipe.Set("", b)
		*nInsertions++
	}
	return nil
}
