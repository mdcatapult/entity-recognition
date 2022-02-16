#! /bin/bash

go run go/cmd/dictionary-importer/main.go dictionaryPath=./go/cmd/dictionary-importer/dictionaries/test.tsv &

bin/dictionary &
bin/regexer &
bin/recognition-api > /dev/null 2>&1 & disown

export NER_API_TEST=yes
go test -v -run=TestAPI ./...
unset NER_API_TEST