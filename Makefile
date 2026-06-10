.PHONY: build clean test

BINARY=gdc-sentinel

build:
	PATH=/usr/local/go/bin:$$PATH go build -o $(BINARY) .

clean:
	rm -f $(BINARY)

test:
	PATH=/usr/local/go/bin:$$PATH go test ./...
