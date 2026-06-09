VERSION ?= $(shell sed -n '1p' VERSION)

all: deps check build
format:
	find . -iname "*.go" -exec gofmt -s -l -w {} \;
check:
	go vet ./...
	go test ./...
run:
	go run ./cmd/HellPot
deps:
	go mod tidy -v
build:
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o HellPot ./cmd/HellPot
