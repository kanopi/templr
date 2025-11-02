# Contributing to templr

Thank you for your interest in contributing to **templr**! 🎉  
We welcome pull requests, bug reports, feature ideas, documentation improvements, and general feedback.

---

## 🧰 Project Setup

### Prerequisites
- Go **1.22+**
- `make` (for common build/test commands)
- Docker (optional, for integration testing)

Clone the repository:

```bash
git clone https://github.com/kanopi/templr.git
cd templr
```

Build the binary:

```bash
make build
```

Run all tests (including example integration tests):

```bash
make test
```

Run example test suite with golden file comparison:

```bash
tests/run_examples.sh
```

If golden files need to be updated:

```bash
UPDATE_GOLDEN=1 tests/run_examples.sh
```

---

## 🧩 Development Workflow

1. **Create a new branch** for your feature or fix:
   ```bash
   git checkout -b feature/my-new-feature
   ```
2. **Write or modify code** in `main.go` or the relevant submodules.
3. **Add or update tests** in the `tests/` and `playground/` directories.
4. **Run tests** locally to ensure everything passes.
5. **Commit** your changes with a clear, conventional message:
   ```
   feat: add --foo flag for new templating behavior
   fix: handle invalid YAML input gracefully
   docs: update usage examples
   ```
6. **Open a Pull Request** against the `main` branch.

---

## ✅ Code Guidelines

- Use `go fmt` before committing.
- Keep PRs focused—avoid combining unrelated changes.
- Document new flags, features, or behaviors in:
  - `README.md`
  - `docs.md` (if they add new templating capabilities)
- Include meaningful **error messages** and **dry-run output** for user clarity.

---

## 🧪 Testing

templr uses both **unit** and **integration tests**:

- **Unit tests:** Validate helpers and logic using Go’s `testing` package.
- **Integration tests:** Located in `/tests` and `/playground`, verifying real templating scenarios.

Run tests:
```bash
go test ./...
```

Update test outputs:
```bash
make golden
```

---

## 🧱 Release Process

Releases are built and published automatically via CircleCI when tags are pushed.

To create a new release:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The version is embedded via build flags and used by the `-version` CLI flag.

---

## 🪪 License

By contributing, you agree that your contributions will be licensed under the **MIT License**, as specified in the [LICENSE](./LICENSE) file.
