TARGET ?= ./...

.PHONY: build run fix test clean

build:
	go build -o ./bin/braider ./cmd/braider

run: build
	./bin/braider -test=false -json $(TARGET)

fix: build
	./bin/braider -fix -test=false $(TARGET)

test:
	go test -v ./...

clean:
	rm -f ./bin/braider
