# Go RPC Chat Room

A concurrent chat application in Go using `net/rpc`. Clients talk to a central server using **polling** (each client requests pending messages every 500 ms).

## Features
- Unique usernames (duplicates rejected)
- Join/leave notifications
- Broadcast to all other clients (sender excluded)
- `/list` shows connected users, `/quit` disconnects
- Handles unexpected disconnects and Ctrl+C

## Requirements
- Go 1.20+

## Run

Server (terminal 1):
```
go run ./server -addr 127.0.0.1:1234
```

Clients (one per terminal):
```
go run ./client -addr 127.0.0.1:1234 -name ziad
go run ./client -addr 127.0.0.1:1234 -name ali
```

## Structure
```
common/   shared RPC types
server/   chat server
client/   chat client
```
