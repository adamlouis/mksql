
make build:
	go build -o mksql main.go

test:
	go test ./..
	