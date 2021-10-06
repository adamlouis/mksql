
benchmark:
	go run cmd/benchmark/mattn/main.go data/dbs/hackernews.db
	go run cmd/benchmark/crawshaw/main.go data/dbs/hackernews.db
	go run cmd/benchmark/bvinc/main.go data/dbs/hackernews.db

run:
	go run main.go $(ARGS)

build:
	go build -o mksql main.go

test:
	go test ./...

lint:
	golangci-lint run
	