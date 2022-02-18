#! /bin/bash

echo "LS'ing!"
ls

bin/dictionary-importer dictionaryPath=./go/cmd/dictionary-importer/dictionaries/test.tsv &

bin/dictionary &
bin/recognition-api > /dev/null 2>&1 & disown

export NER_API_TEST=yes
go test -v -run=TestAPI ./...
unset NER_API_TEST