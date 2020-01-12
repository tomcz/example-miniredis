.PHONY: all clean format lint compile run

all: clean format lint compile

target:
	mkdir target

clean:
	rm -rf target

format:
	goimports -w -local github.com/tomcz/example-miniredis .

lint:
	go vet ./...
	golint -set_exit_status ./...

compile: target
	go build -o target/example ./cmd/example/...

run: compile
	./target/example
