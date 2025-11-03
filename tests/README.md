# Go Test Suite (E2E-style)

These Go tests build the `templr` binary and exercise core behaviors end-to-end:

- Single-file rendering with values
- Guard overwrite behavior (`--guard`)
- Walk mode with pruning of empty results
- `.Files.Get` API in `--dir` mode
- `--ext` support in walk mode

## Running

```bash
go test ./tests/...
```

The tests will:
1. Run `go build` to produce a temporary test binary.
2. Create temporary fixtures.
3. Execute the CLI and assert outputs.
