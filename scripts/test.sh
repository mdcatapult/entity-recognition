#! /bin/bash

echo "starting!!"
go run go/cmd/dictionary-importer/main.go & # TODO: configure importer to use test.csv rather than hardcoding it

bin/dictionary &
bin/regexer &
bin/recognition-api > /dev/null 2>&1 & disown

export NER_API_TEST=yes
go test -v -run=TestAPI ./...
unset NER_API_TEST
echo "finished!!"