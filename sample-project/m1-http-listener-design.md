# M1: HTTP Listener — Design Spec

## Goal

Set up a minimal HTTP server that listens on a configurable address, logs startup, and supports graceful shutdown on SIGTERM.

## Architecture

```
main.go
  ├── flag.Parse()          → get -addr (default :8080)
  ├── http.Handle("/", mux)
  └── http.ListenAndServe(addr, nil)
      └── signal.Notify(SIGTERM) → server.Shutdown(ctx)
```

The listener is the entry point. All requests enter via this listener. It must be configurable and shut down cleanly.

## Components

### Flag parsing (m1-s1-t1)

- Use `flag` package to define `-addr` with default `:8080`
- Store parsed address in a package-level or `main()` scoped variable
- Call `flag.Parse()` before any server startup logic

### Server startup (m1-s1-t2)

- Call `http.ListenAndServe(addr, nil)` with the parsed address
- Log startup: `log.Printf("listening on %s", addr)`
- Spawn a goroutine for ListenAndServe to allow interrupt handling
- Listen on `os.Signal` channel for `SIGTERM` / `SIGINT`
- On signal, call `server.Shutdown(context.Background())`

## Data Flow

```
[Client] → HTTP request → :8080 → mux.Handler → [to be built in M2]
```

No application state is maintained. The listener simply accepts and dispatches.

## Error Handling

- If `-addr` is invalid or port is unavailable, `ListenAndServe` returns error → `log.Fatal(err)`
- If `Shutdown` times out or fails, log the error but exit anyway
- Server should not hang on shutdown — use context timeout (default 5s)

## Testing

- `go build ./...` verifies compilation
- Manual test: start server, `curl localhost:8080`, observe 404 (no handler yet)
- SIGTERM test: kill the process, verify clean exit

## Verify

```yaml
verify:
  - go build ./...
```
