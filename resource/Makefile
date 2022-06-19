.PHONY: build format test clean run docker-build

all: format test build run

build:
	go build -o server ../server.go

clean:
	rm ../server

test:
	go test -race $(go list ../... | grep -v /vendor/)

run:
	../server

format:
	go fmt $(go list ../... | grep -v /vendor/)
	go vet $(go list ../... | grep -v /vendor/)

docker-build:
	docker build -t DockerImage .

