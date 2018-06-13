VERSION := 0.1.0
LINTER_TAG := v1.0.3

# Creates binary
build:
	go build -ldflags="-X github.com/dustinblackman/tidalwave/cmd.version=$(VERSION)" -o tidalwave *.go

# Creates bash autocomplete file
bashautocomplete:
	go run ./tools/bash-autocomplete/bash.go
	gofmt -s -w ./cmd/autocomplete.go

deps:
	which dep && echo "" || go get -u github.com/golang/dep/cmd/dep
	dep ensure
	rm -rf vendor/github.com/lfittl/pg_query_go
	go get github.com/lfittl/pg_query_go
	cd $$GOPATH/src/github.com/lfittl/pg_query_go && make build

dev:
	which reflex && echo "" || go get github.com/cespare/reflex
	reflex -R '^vendor/' -r '\.go$\' -s -- sh -c 'go run tidalwave.go -server -client'

dev-build:
	which reflex && echo "" || go get github.com/cespare/reflex
	reflex -R '^vendor/' -r '\.go$\' -s -- sh -c 'go build -o tidalwave tidalwave.go && ./tidalwave -server'

# Creates binarys for all available systems in gox and then zips/tars for distribution.
dist:
	which gox && echo "" || go get github.com/mitchellh/gox
	rm -rf tmp dist
	gox -os="linux windows freebsd" -osarch="darwin/amd64" -output='tmp/{{.OS}}-{{.Arch}}-$(VERSION)/{{.Dir}}' -ldflags="-X github.com/dustinblackman/tidalwave/cmd.version=$(VERSION)"
	mkdir dist

	# Build for Windows
	@for i in $$(find ./tmp -type f -name "tidalwave.exe" | awk -F'/' '{print $$3}'); \
	do \
	  zip -j "dist/tidalwave-$$i.zip" "./tmp/$$i/tidalwave.exe"; \
	done

	# Build for everything else
	@for i in $$(find ./tmp -type f -not -name "tidalwave.exe" | awk -F'/' '{print $$3}'); \
	do \
	  chmod +x "./tmp/$$i/tidalwave"; \
	  tar -zcvf "dist/tidalwave-$$i.tar.gz" --directory="./tmp/$$i" "./tidalwave"; \
	done

	rm -rf tmp

# Creates easyjson file for parser/parser.go
easyjson:
	easyjson parser/parser.go

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
