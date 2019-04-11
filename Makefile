VERSION := 1.0.0
LINTER_TAG := v1.0.3

# Creates binary
build:
	go build -x -ldflags="-X github.com/dustinblackman/tidalwave/cmd.version=$(VERSION)" -o tidalwave *.go

# Creates bash autocomplete file
bashautocomplete:
	go run ./tools/bash-autocomplete/bash.go
	gofmt -s -w ./cmd/autocomplete.go

deps:
	which dep && echo "" || go get -u github.com/golang/dep/cmd/dep
	dep ensure
	rm -rf vendor/github.com/lfittl/pg_query_go
	go get -u github.com/lfittl/pg_query_go
	cd $$GOPATH/src/github.com/lfittl/pg_query_go && make build

# Creates easyjson file for parser/parser.go
easyjson:
	runvendor github.com/mailru/easyjson/easyjson parser/parser.go

# Builds and installs binary. Mainly used from people wanting to install from source.
install:
	go install -ldflags="-X github.com/dustinblackman/tidalwave/cmd.version $(VERSION)" *.go

# Setups linter configuration for tests
setup-linter:
	@if [ "$$(which gometalinter)" = "" ]; then \
		go get -u -v github.com/alecthomas/gometalinter; \
		cd $$GOPATH/src/github.com/alecthomas/gometalinter;\
		git checkout tags/$(LINTER_TAG);\
		go install;\
		gometalinter --install;\
	fi

# Runs tests
test: setup-linter
	gometalinter --vendor --fast --dupl-threshold=100 --cyclo-over=25 --min-occurrences=5 --disable=gas --disable=gotype ./...
