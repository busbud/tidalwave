VERSION := 1.3.1

# Creates binary
build:
	go build -x -ldflags="-X github.com/busbud/tidalwave/cmd.version=$(VERSION)" -o tidalwave tidalwave.go

# Creates easyjson file for parser/parser.go
easyjson:
	gomodrun easyjson parser/parser.go

# Builds and installs binary. Mainly used from people wanting to install from source.
install:
	go install -ldflags="-X github.com/busbud/tidalwave/cmd.version $(VERSION)"

# Runs tests
lint:
	gomodrun golangci-lint run

lint-fix:
	gomodrun golangci-lint run --fix

test:
	make lint
