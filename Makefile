all: server client linux

release: linux darwin windows shasum

client:
	go build -o bin/client client.go
server:
	go build -o bin/server server.go
linux:
	CGO_ENABLED=0 GOOS=linux go build -o bin/server_linux server.go
	CGO_ENABLED=0 GOOS=linux go build -o bin/client_linux  client.go
darwin:
	CGO_ENABLED=0 GOOS=darwin go build -o bin/server_darwin server.go
	CGO_ENABLED=0 GOOS=darwin go build -o bin/client_darwin client.go
windows:
	CGO_ENABLED=0 GOOS=windows go build -o bin/server_windows server.go
	CGO_ENABLED=0 GOOS=windows go build -o bin/client_windows  client.go
shasum:
	shasum -a 256 bin/*_*
