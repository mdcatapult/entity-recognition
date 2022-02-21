#! /bin/bash
bin/dictionary-importer dictionaryPath=./go/cmd/dictionary-importer/dictionaries/test.tsv dictionaryFormat=leadmine &
bin/regexer & 
bin/dictionary &
sleep 2 && # services need time to spin up
bin/recognition-api & 


export NER_API_TEST=yes
go test -v -run=TestAPI ./...
unset NER_API_TEST