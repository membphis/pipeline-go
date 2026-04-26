# M1: HTTP Listener

## Goal

Set up a minimal HTTP server that listens on `:8080` and responds with a simple health check.

## Tasks

### m1-s1-t1: Server Scaffolding

- Use `net/http` from the standard library
- Listen on address `:8080`
- Log `"listening on :8080"` on startup
- Handle SIGINT/SIGTERM for graceful shutdown (server.Shutdown)

### m1-s1-t2: Health Endpoint

- Register a handler for `GET /health`
- Return HTTP 200 with body `"ok"`

## Verify

```
go build ./...
curl -s http://localhost:8080/health
```
