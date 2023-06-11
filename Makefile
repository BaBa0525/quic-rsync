.PHONY: server
server:
	go run ./cmd/rsyncd

.PHONY: client
client:
	go run ./cmd/rsync
