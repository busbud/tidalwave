VERSION := 1.1.0

# Creates binary
build:
	go build -x -ldflags="-X github.com/dustinblackman/tidalwave/cmd.version=$(VERSION)" -o tidalwave tidalwave.go

cli-deps:
	@which gobin &> /dev/null || GO111MODULE=off go get -u github.com/myitcv/gobin

# Creates bash autocomplete file
bashautocomplete:
	go run ./tools/bash-autocomplete/bash.go
	gofmt -s -w ./cmd/autocomplete.go

# Creates easyjson file for parser/parser.go
easyjson: cli-deps
	gobin -m -run github.com/mailru/easyjson/easyjson parser/parser.go

# Builds and installs binary. Mainly used from people wanting to install from source.
install:
	go install -ldflags="-X github.com/dustinblackman/tidalwave/cmd.version $(VERSION)" *.go

# Runs tests
lint: cli-deps
	gobin -m -run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...

lint-fix: cli-deps
	gobin -m -run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix ./...

test:
	make lint
