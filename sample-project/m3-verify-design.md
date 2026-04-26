# M3: Final Verification

## Goal

Ensure the project compiles and passes static analysis.

## Tasks

### m3-s1-t1: Build Check

- Run `go build ./...` — exit 0 means compilation OK

### m3-s1-t2: Static Analysis

- Run `go vet ./...` — exit 0 means no suspicious code

## Verify

```
go build -o /dev/null ./...
go vet ./...
```
