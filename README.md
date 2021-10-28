# Entity recognition

This is intended to be a monorepo [*monorepo*](https://www.atlassian.com/git/tutorials/monorepos) containing all MDC entity recognition software, configuration, documentation, and so on.

## Overview
There are currently 4 primary applications:
1. **The recognition API**. This HTTP REST API calls out to the recognizer gRPC recognizer services. (See the [overview diagram](#diagrams)).
2. **The regexer recognizer**. This simple gRPC recognizer service receives a stream of tokens and returns a stream of entities based on a regex match.
3. **The dictionary recognizer**. This gRPC recognizer service recieves a stream of tokens and looks them up in a backend database, returning a stream of entities based on the result. (This can be complicated by a number of things, see the [diagram](#diagrams))
4. **The dictionary importer**. This app reads a file line by line, parses it, and upserts it to a backend database that the dictionary recognizer is compatible with.
## Diagrams
* [Overview of the architecture](https://lucid.app/lucidchart/1598c66b-ddb5-486c-a706-5d8a44f07220/edit?page=0_0#).
* [Dictionary recognizer workflow](https://lucid.app/lucidchart/899a175a-a933-4f8d-9b4f-ff6d93f72896/edit?beaconFlowId=CD8D681A5455AD49&page=0_0#)

## Development
### Run
```bash
make config
make build
docker-compose up -d redis
bin/dictionary-importer
make run
```
**IMPORTANT:** The make run command runs processes in the background using `&`. There is a bash trap which executes a function to foreground those processes on interrupt. In case this doesn't work, you might have some hanging processes on your machine. Use `ps` or `pgrep` to find and kill them.


You can also just press the play button next to a main function in intellij :smiley:.
### Test
Grab some html from a website (ctrl+U in chrome). Make a post request to `localhost:8080/html/text`, `localhost:8080/html/tokens`, or `localhost:8080/html/entities` with the html in the body of the request.

For example:
```bash
curl -L https://en.wikipedia.org/wiki/Acetylcarnitine > /tmp/acetylcarnitine.html
curl -XPOST --data-binary "@/tmp/acetylcarnitine.html" 'http://localhost:8080/html/entities?recogniser=dictionary'
```

### Code Generation
Some code in this repo is generated. The generated code is committed, so you don't need to regenerate it yourself. See the [Makefile](Makefile) for more info.

