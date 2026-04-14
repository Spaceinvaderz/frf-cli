# Agent: test-writer

Write unit tests for client methods and CLI commands.

## Instructions

### Input
The user will specify what to test:
- A client method (e.g. `CreatePost`)
- A CLI command (e.g. `frf post create`)
- Or "all untested" to scan for coverage gaps

### Steps

1. Read the code to test
2. Create test file next to the source (e.g. `client_test.go`, `post_test.go`)
3. Write tests following the patterns below
4. Run `go test ./...` to verify

### Client method tests
Use recorded JSON fixtures. Test both success and error cases.

```go
func TestCreatePost(t *testing.T) {
    // Set up httptest.Server that returns fixture JSON
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify method, path, headers, body
        // Return fixture response
    }))
    defer server.Close()

    c := client.New(server.URL, "testuser", "testpass")
    c.SetToken("test-token") // skip auth for unit tests

    result, err := c.CreatePost("Hello world", nil)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // Assert result fields
}
```

### Fixture format
Store JSON fixtures in `testdata/` directories:
- `internal/client/testdata/create_post_response.json`
- `internal/client/testdata/timeline_home_response.json`

### What to test
- Happy path with realistic API response
- Error responses (401, 404, 500)
- Edge cases from FreeFeed API (array vs map responses, missing fields)
- Verify request method, path, auth header, content-type

### Do NOT
- Mock the entire HTTP client — use httptest.Server
- Write integration tests (those go behind build tags)
- Test TUI rendering
