.PHONY: all clean format lint compile run

all: clean format lint compile

target:
	mkdir target

clean:
	rm -rf target

format:
ifeq (, $(shell which goimports))
	go install golang.org/x/tools/cmd/goimports
endif
	goimports -w -local github.com/tomcz/example-miniredis .

lint:
ifeq (, $(shell which staticcheck))
	go install honnef.co/go/tools/cmd/staticcheck@2021.1
endif
	staticcheck ./...

compile: target
	go build -o target/example ./cmd/example/...

run: compile
	ENV="dev" ./target/example

HELLO ?= Bob

enqueue:
	curl -s -X POST 'http://localhost:3000/enqueue?key=${HELLO}'

dequeue:
	curl -s 'http://localhost:3000/dequeue?key=${HELLO}'

show-stats:
	curl -s 'http://localhost:3000/workers/stats'

show-retries:
	curl -s 'http://localhost:3000/workers/retries'
