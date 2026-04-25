# M2: Forwarding Handler — Design Spec

## Goal

Implement an HTTP forward proxy that intercepts all incoming requests and forwards them to a configurable target server.

## Architecture

```
[Client] → HTTP /any → :8080 → ReverseProxy → httpbin.org
                                    │
                                proxy.ServeHTTP rewrites:
                                  - Scheme → target.Scheme
                                  - Host  → target.Host
                                  - Path  → same
```

Uses `net/http/httputil.ReverseProxy` from the standard library — no external dependencies.

## Components

### Target flag parsing (m2-s1-t1)

- Add `-target` flag with default `http://httpbin.org`
- Parse alongside existing `-addr` in `init()` or before `main()` usage
- Store target URL as `*url.URL` for direct use with ReverseProxy

### Reverse proxy creation (m2-s1-t2)

- Call `httputil.NewSingleHostReverseProxy(targetURL)` to create proxy
- This automatically handles:
  - Scheme/host rewriting in outgoing requests
  - Response copying back to original client
  - X-Forwarded-For header injection

### Handler wiring (m2-s1-t3)

- Replace default `http.DefaultServeMux` with proxy as handler
- Use `http.Handle("/", proxy)` to catch all routes
- The previous mux (nil) is replaced — all traffic routes through proxy

## Data Flow

```
Request in  ──► ReverseProxy ──► Target Server
                 │                     │
                 │  ┌──────────────────┘
                 │  ▼
Response out  ◄──┘
```

- Request: Client → `:8080` → ReverseProxy rewrites host → Target
- Response: Target → ReverseProxy copies headers/body → Client

## Error Handling

- If `-target` is an invalid URL, `url.Parse()` returns error → `log.Fatal(err)`
- ReverseProxy has built-in error handling: logs failed upstream requests, returns 502
- Network errors proxying to target: ReverseProxy writes `502 Bad Gateway` response

## Testing

- `go build ./...` verifies compilation
- Manual: start server, `curl http://localhost:8080/get` → forwards to httpbin.org/get
- Invalid target: verify `log.Fatal` on bad URL

## Verify

```yaml
verify:
  - go build ./...
```
