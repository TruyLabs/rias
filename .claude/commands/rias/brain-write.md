Write or update a brain file.

Parse `$ARGUMENTS` for the required fields:
- `path` — relative path (e.g. "opinions/testing.md")
- `content` — the markdown content
- `tags` — comma-separated tags
- `confidence` — high, medium, or low (optional, default: medium)

Call the `mcp__rias__brain_write` tool with those fields.

Show confirmation of what was saved.
