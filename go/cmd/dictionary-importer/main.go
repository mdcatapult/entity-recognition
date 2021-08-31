package main

import (
	"encoding/json"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache/remote"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/dict"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib"
)


// config structure
type dictionaryImporterConfig struct {
	lib.BaseConfig
	dict.DictConfig
	BackendDatabase cache.Type `mapstructure:"dictionary_backend"`
	PipelineSize    int        `mapstructure:"pipeline_size"`
	Redis           remote.RedisConfig
	Elasticsearch   remote.ElasticsearchConfig
}

var config dictionaryImporterConfig

func initConfig() {
	// initialise config with defaults.
	err := lib.InitializeConfig("./config/dictionary-importer.yml", map[string]interface{}{
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
		log.Fatal().Msg("invalid backend database type")
	}

	dictFile, err := os.Open(config.DictConfig.Path)
	if err != nil {
		log.Fatal().Err(err).Send()
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

			pipe = dbClient.NewSetPipeline(config.PipelineSize)
		}

		return nil
	}

	onExit := func() error {
		if pipe.Size() > 0 {
			return pipe.ExecSet()
		}

		return nil
	}

	if err := dict.ReadWithCallback(config.DictConfig.Format, onEntry, onExit, dictFile); err != nil {
		log.Fatal().Err(err).Send()
	}
}

func addToPipe(entry dict.Entry, pipe remote.SetPipeline, nInsertions *int) error {
	// Mid process, some stuff to do
	switch config.BackendDatabase {
	case cache.Redis:
		for _, s := range entry.Synonyms {
			b, err := json.Marshal(cache.Lookup{
				Dictionary:       config.DictConfig.Name,
				ResolvedEntities: entry.Identifiers,
			})
			if err != nil {
				return err
			}
			pipe.Set(s, b)
			*nInsertions++
		}
	case cache.Elasticsearch:
		b, err := json.Marshal(remote.EsLookup{
			Dictionary:  config.DictConfig.Name,
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