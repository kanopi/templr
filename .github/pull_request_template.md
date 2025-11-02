# Pull Request Template

## 🧩 Summary
Provide a clear and concise summary of what this PR does and why it’s needed.

---

## 🧠 Context
Explain the background of this change. Reference any related issues, discussions, or design documents.

- Related Issue(s): Closes # (issue number)
- Related PR(s): (if any)

---

## ✅ Changes
List key changes introduced in this PR:
- [ ] Added / Updated functionality
- [ ] Fixed bug(s)
- [ ] Updated documentation
- [ ] Added / Updated tests

---

## 🧪 Testing
Describe how you tested your changes:
- [ ] Unit tests
- [ ] Integration tests
- [ ] Manual testing steps (include details)
- [ ] CI pipeline run successful

Include any test output or screenshots if applicable.

---

## ⚙️ How to Review
Provide steps or commands for reviewers to validate this PR locally.

Example:
```bash
make build
./templr --walk --src ./templates --dst ./out
```

---

## 🧱 Checklist
Before requesting a review, ensure the following are complete:

- [ ] Code follows the project’s Go style guide and passes linting (`golangci-lint run`)
- [ ] All tests pass locally
- [ ] Documentation updated (README / docs.md / contributing.md)
- [ ] No secrets, credentials, or sensitive data included
- [ ] Commits are squashed and messages follow the convention (`feat:`, `fix:`, `docs:`, etc.)

---

## 🪪 License
By submitting this PR, you agree that your contribution will be licensed under the [MIT License](../LICENSE).

---