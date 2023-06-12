.PHONY: build
build:
	go build -o ./bin/rsyncd ./cmd/rsyncd
	go build -o ./bin/rsync ./cmd/rsync

.PHONY: server
server:
	go run ./cmd/rsyncd

.PHONY: client
client:
	go run ./cmd/rsync

.PHONY: clean
clean:
	@rm -vrf ./bin
