# M3: Observability — Design Spec

## Goal

Add request-level logging and proper graceful shutdown handling via OS signals.

## Architecture

```
Request ──► Logging Middleware ──► ReverseProxy ──► Target
                │
            log.Printf("%s %s → %d", method, path, status)

Server loop ──► signal.Notify(ch, SIGINT, SIGTERM)
                    │
                    ▼
               server.Shutdown(ctx)
```

The logging middleware wraps the existing proxy handler. Signal handler wraps the existing server startup goroutine.

## Components

### Logging middleware (m3-s1-t1)

- Wrap proxy handler with a closure that:
  - Records start time
  - Calls `proxy.ServeHTTP(w, r)`
  - Logs: `log.Printf("%s %s → %d (%s)", r.Method, r.URL.Path, statusCode, duration)`
- Since `http.ResponseWriter` doesn't expose status code directly, use `httputil.ResponseRecorder` or a custom wrapper that captures `WriteHeader(code)`

### Graceful shutdown (m3-s1-t2)

- Create `signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)` channel
- Block on `<-ch` in a goroutine
- On signal: `log.Print("shutting down...")` then `server.Shutdown(ctx)`
- Use context with 5-second timeout for pending requests
- After shutdown completes, `os.Exit(0)`

## Data Flow

```
Client ──► Logging Middleware (log request start)
             │
             ▼
          ReverseProxy ──► Target
             │
             ▼
          Logging Middleware (log response + status)
             │
             ▼
          Client receives response

Signal ──► signal.Notify ──► Shutdown ──► drain connections → exit
```

## Error Handling

- `Shutdown()` returns error if connections don't drain within timeout → log it
- Already-in-flight requests complete normally during shutdown
- Second SIGTERM/SIGINT during shutdown: exit immediately with `os.Exit(1)`
- Logging must not panic if handler panics — use `defer` recovery if needed

## Testing

- `go build ./...` verifies compilation
- Manual: start server, make requests, check log output
- Manual: SIGTERM the process, verify "shutting down..." appears and server exits cleanly

## Verify

```yaml
verify:
  - go build ./...
```
