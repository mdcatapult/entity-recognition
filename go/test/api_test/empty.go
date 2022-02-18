package apitest

/**
	This is an empty file which is needed to get coverpkg to work in the CI.
	It needs at least one file in the package which is not a "test" file as api_test.go is.
	Without this, the command

	`go test ./go/... -coverpkg=./... -coverprofile=cover.out`

	fails with

	go build gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/test/api_test: no non-test Go files in /builds/informatics/software-engineering/entity-recognition/go/test/api_test

	See https://github.com/golang/go/issues/27333.
**/
