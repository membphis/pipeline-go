# M2: Calculator API

## Goal

Add an API endpoint that computes the sum or product of all query parameters.

## Tasks

### m2-s1-t1: Sum Operation

- Add `GET /calc` endpoint
- Accept query parameter `op=sum` (or omitted, default to `sum`)
- Accept any number of numeric query parameters (e.g. `a=1&b=2&c=3`)
- Return JSON: `{"op":"sum","result":6}`

### m2-s1-t2: Mul Operation & Error Handling

- Support `op=mul` — compute product of all numeric parameters
- Return HTTP 400 for non-numeric values with JSON error body `{"error":"invalid input: <param_name>"}`
- Return HTTP 200 with JSON result otherwise

### Examples

```
GET /calc?op=sum&a=1&b=2&c=3     → {"op":"sum","result":6}
GET /calc?op=mul&a=2&b=3&c=4     → {"op":"mul","result":24}
GET /calc?a=10&b=20               → {"op":"sum","result":30}
GET /calc?op=mul&x=foo            → 400 Bad Request
```

## Verify

```
go build ./...
curl -s http://localhost:8080/calc?op=sum&a=1&b=2
```
