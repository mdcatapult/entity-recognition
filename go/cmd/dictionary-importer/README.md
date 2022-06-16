# Dictionary Importer

This program takes a "dictionary" - a file mapping tokens to recognised entities - and imports it into redis. Once this has been used to populate redis, the NER system is ready to recognise tokens!

The dictionary file can be one of several formats, depending on the data it represents. The supported datasets and their corresponding formats are:

- pubchem: .tsv. Each key-value pair *must* be on a new line.
- leadmine: .tsv. Each key-value pair *must* be on a new line.
- swissprot: .jsonl.

## Running

The dictionary filepath and format can be given as program arguments, for example to import `./my-dictionary` which contains Pubchem data:

`go run main.go dictionaryPath=dictionaries/pubchem.tsv dictionaryFormat=pubchem`

Other config e.g. redi port is located in `./config/dictionary.yml`, relative from the NER project root. See the existing config for examples. 