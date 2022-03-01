#! /bin/bash
testSuite=$1
dict=$2
format=$3

bin/dictionary-importer dictionaryPath=$dict dictionaryFormat=$format &&
bin/regexer & 
bin/dictionary &
sleep 2 && # services need time to spin up
bin/recognition-api & 

export NER_API_TEST=yes
go test -v -run=$1 ./...
result=$?
unset NER_API_TEST
pkill -P $$
exit $result