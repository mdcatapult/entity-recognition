/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package swissprot

/**
	This is an empty file which is needed to get coverpkg to work in the CI.
	It needs at least one file in the package which is not a "test" file as api_test.go is.
	Without this, the command

	`go test ./go/... -coverpkg=./... -coverprofile=cover.out`

	fails with

	go build gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/test/api_test: no non-test Go files in /builds/informatics/software-engineering/entity-recognition/go/test/api_test

	See https://github.com/golang/go/issues/27333.
**/
