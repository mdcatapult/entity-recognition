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

## Deployment overview
The recognition API doesn't do anything on its own. You need to configure it with some `recognisers` (either `http` or `grpc`).
To configure the recognisers, you need to mount [config map](https://kubernetes.io/docs/concepts/configuration/configmap/) in `/app/config` with the name `recognition-api.yml`.
(When containerised, all apps in this repo will look for a config file in the `/app/config` folder).
Currently, there are only two types of recogniser: `grpc` and `http` (of which leadmine is a subtype, see the [example config file](./config/recognition-api.example.yml)).

To summarise:
1. Deploy a grpc or http recogniser. You may need to create additional resources for these recognisers such as configmaps, secrets, or even other deployments such as redis.
2. Ensure the recogniser is accessible over the network.
3. Create a key in a configmap with the `recognition-api.yml` config.
4. Deploy the recognition api with the config map key mounted in `/app/config` with the path `recognition-api.yml`.
