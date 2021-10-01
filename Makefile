
run:
	go run main.go $(ARGS)

build:
	go build -o mksql main.go

test:
	go test ./...

lint:
	golangci-lint run
	