# Recognition API

Recognition API provides a HTTP server to handle requests relating to entity recognition.
Requires redis to be running, as well as the dictionary program.

## Documentation 

To see documentation around endpoints and types, `make docs` from project root. This requires go-swagger which can be installed from source:

```
dir=$(mktemp -d) 
git clone https://github.com/go-swagger/go-swagger "$dir" 
cd "$dir"
go install ./cmd/swagger
```

## Running

`go build -o . ./... && ./recognition-api`

This service can be configured using yml. The yml must be located in `./config/dictionary.yml`, relative from the NER project root. See the existing config for examples. 

## Testing

