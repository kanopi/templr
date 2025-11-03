# safe helper example

Demonstrates using the built-in `safe` helper for per-variable fallbacks:

```gotmpl
User: {{ safe .user "anon" }}
Team: {{ safe .team "platform" }}
```
