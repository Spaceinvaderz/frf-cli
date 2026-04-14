# Agent: add-cli-command

Scaffold and implement a new CLI command for the `frf` binary.

## Instructions

You are adding a new cobra command to the `frf` CLI tool.

### Input
The user will specify:
- Command path (e.g. `frf post create`)
- What client method(s) it calls
- Required and optional arguments/flags

### Steps

1. Read `cmd/frf/main.go` and `internal/cli/root.go` to understand the command tree
2. Read the client method this command will call in `internal/client/client.go`
3. Create or update the command file in `internal/cli/` (e.g. `post.go` for post subcommands)
4. Implement the command:
   - Parse args and flags
   - Create client, authenticate
   - Call the appropriate client method
   - Output result via `internal/cli/output.go`
5. Register the command in the parent command
6. Run `go build ./cmd/frf` to verify

### Command conventions
- Use `cobra.Command` with `Use`, `Short`, `Args`, `RunE`
- Required positional args via `cobra.ExactArgs(N)` or `cobra.MinimumNArgs(N)`
- Optional params as flags: `--limit`, `--offset`, `--format`
- Auth: create client from config, call `Authenticate()` first
- Output: call `outputJSON(result)` or `outputText(result)` based on `--format` flag
- Errors: return `fmt.Errorf(...)` — cobra handles exit code

### Template
```go
var postCreateCmd = &cobra.Command{
    Use:   "create <body>",
    Short: "Create a new post",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := app.LoadConfig()
        if err != nil {
            return err
        }
        c := client.New(cfg.BaseURL, cfg.Username, cfg.Password)
        if err := c.Authenticate(); err != nil {
            return err
        }
        // ... call client method, output result
        return outputResult(cmd, result)
    },
}
```

### Do NOT
- Modify the client — use the expand-client agent for that
- Add TUI code
- Skip compilation check
