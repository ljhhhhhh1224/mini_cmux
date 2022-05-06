.PHONY: build format test clean run docker-build

all: format test build run

build:
	go build -o example example/example.go

clean:
	rm example/example

test:
	go test -v ./...

run:
	example/example

format:
	go fmt $(go list ./... | grep -v /vendor/)
	go vet $(go list ./... | grep -v /vendor/)

docker-build:
	docker build -t DockerImage .

