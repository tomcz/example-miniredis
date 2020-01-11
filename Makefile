.PHONY: run clean compile

run: clean compile
	./target/example

target:
	mkdir target

clean:
	rm -rf target

compile: target
	go build -o target/example ./cmd/example/...
