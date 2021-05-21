.PHONY: all clean format lint compile run

all: clean format lint compile

PACKAGES = $(shell go list ./... | grep -v /vendor/)

target:
	mkdir target

clean:
	rm -rf target

format:
	goimports -w -local github.com/tomcz/example-miniredis .

lint:
	go vet ${PACKAGES}
	golint -set_exit_status ${PACKAGES}

compile: target
	go build -o target/example ./cmd/example/...

run: compile
	ENV="dev" ./target/example

HELLO ?= Bob

enqueue:
	curl -s -X POST 'http://localhost:3000/enqueue?key=${HELLO}'

dequeue:
	curl -s 'http://localhost:3000/dequeue?key=${HELLO}'
