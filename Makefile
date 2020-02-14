VERSION := 1.3.0

# Creates binary
build:
	go build -x -ldflags="-X github.com/busbud/tidalwave/cmd.version=$(VERSION)" -o tidalwave tidalwave.go
# Creates bash autocomplete file
bashautocomplete:
	go run ./tools/bash-autocomplete/bash.go
	gofmt -s -w ./cmd/autocomplete.go

# Creates easyjson file for parser/parser.go
easyjson:
	gomodrun easyjson parser/parser.go

# Builds and installs binary. Mainly used from people wanting to install from source.
install:
	go install -ldflags="-X github.com/busbud/tidalwave/cmd.version $(VERSION)"

# Runs tests
lint: cli-deps
	gomodrun golangci-lint run ./...

lint-fix: cli-deps
	gomodrun golangci-lint run --fix ./...

test:
	make lint
