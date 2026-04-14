# Agent: expand-client

Add a new API method to the FreeFeed Go client.

## Instructions

You are adding a new method to `internal/client/client.go`. 

### Input
The user will specify:
- Method name (e.g. `CreatePost`)
- HTTP method and endpoint (e.g. `POST /v4/posts`)
- Request payload shape
- Expected response shape

### Steps

1. Read `internal/client/client.go` to understand existing patterns
2. Read the equivalent Python implementation in `../freefeed-mcp-server/freefeed_mcp_server/client.py` for reference
3. Add the method to the `Client` struct following these conventions:
   - Use `context.WithTimeout` (20s default)
   - Use `c.doJSON()` for JSON requests
   - Return parsed Go types, not raw JSON
   - Handle error cases (404, auth errors)
4. Add any new types needed (request/response structs)
5. Run `go build ./...` to verify compilation

### Conventions
- Method signature: `func (c *Client) MethodName(args) (ReturnType, error)`
- Use `url.PathEscape` for path parameters
- Use `url.Values` for query parameters
- Auth requests: pass `true` to `doJSON`'s `auth` parameter
- Non-auth requests (only session creation): pass `false`

### Do NOT
- Change existing methods
- Add tests (use the test-writer agent for that)
- Modify the TUI code
