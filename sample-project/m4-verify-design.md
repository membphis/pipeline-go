# M4: Verification — Design Spec

## Goal

Ensure the final project compiles and passes static analysis with no warnings or errors.

## Architecture

```
go build -o /dev/null ./...     → exit 0 = compilation OK
go vet ./...                    → exit 0 = no suspicious constructs
```

These are the two gates before the project is considered complete.

## Components

### Build smoke test (m4-s1-t1)

- Run `go build -o /dev/null ./...` from the project root
- This compiles all packages without producing a binary (output to /dev/null)
- Exit code 0 means the project compiles cleanly

### Static analysis (m4-s1-t2)

- Run `go vet ./...` on all packages
- `go vet` reports suspicious constructs: unreachable code, misused reflect.DeepEqual, unsafe pointer arithmetic, etc.
- Exit code 0 means no issues found

## Data Flow

```
[Source code] ──► go build ──► exit 0? ──► go vet ──► exit 0? ──► PASS
                     │                        │
                  exit ≠0                  exit ≠0
                     │                        │
                     ▼                        ▼
                   FAIL                      FAIL
```

No runtime component — these are build-time checks.

## Error Handling

- Build failure: `go build` prints compiler errors to stderr
- Vet failure: `go vet` prints issues to stderr with file:line positions
- Both return non-zero exit code on failure

## Testing

- These checks ARE the testing — they verify the project itself
- If `go build` passes, the project compiles
- If `go vet` passes, code quality baseline is met

## Verify

```yaml
verify:
  - go build -o /dev/null ./...
  - go vet ./...
```
