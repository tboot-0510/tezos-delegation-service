build:
	go build -o bin/xtz

run: build
	./bin/xtz

test:
	go test -v ./...