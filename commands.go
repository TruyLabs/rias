package kai

import "strings"

// commandTemplates defines slash commands with {{NAME}} as a placeholder
// for the configured agent name.
var commandTemplates = map[string]string{
	"ask": `Ask {{NAME}} a question using brain context.

Call the ` + "`mcp__{{NAME}}__ask`" + ` tool with:
- ` + "`question`" + `: ` + "`$ARGUMENTS`" + `

Show the response directly.
`,
	"teach": `Teach {{NAME}} something new.

If ` + "`$ARGUMENTS`" + ` contains structured fields (category, topic, content, tags), call ` + "`mcp__{{NAME}}__teach`" + ` in direct mode with those fields.

Otherwise, call ` + "`mcp__{{NAME}}__teach`" + ` with:
- ` + "`input`" + `: ` + "`$ARGUMENTS`" + `

Show what was saved.
`,
	"brain-list": `List all brain knowledge files.

Call the ` + "`mcp__{{NAME}}__brain_list`" + ` tool (no parameters needed).

Show the results as a table with path, tags, and confidence.
`,
	"brain-read": `Read a brain file's content.

Call the ` + "`mcp__{{NAME}}__brain_read`" + ` tool with:
- ` + "`path`" + `: ` + "`$ARGUMENTS`" + `

Show the file content with its tags and confidence level.
`,
	"brain-search": `Search brain knowledge by keywords.

Call the ` + "`mcp__{{NAME}}__brain_search`" + ` tool with:
- ` + "`query`" + `: ` + "`$ARGUMENTS`" + `

Show the results ranked by score.
`,
	"brain-write": `Write or update a brain file.

Parse ` + "`$ARGUMENTS`" + ` for the required fields:
- ` + "`path`" + ` — relative path (e.g. "opinions/testing.md")
- ` + "`content`" + ` — the markdown content
- ` + "`tags`" + ` — comma-separated tags
- ` + "`confidence`" + ` — high, medium, or low (optional, default: medium)

Call the ` + "`mcp__{{NAME}}__brain_write`" + ` tool with those fields.

Show confirmation of what was saved.
`,
	"brain-reorganize": `Analyze brain files for reorganization opportunities.

Call the ` + "`mcp__{{NAME}}__brain_reorganize`" + ` tool with:
- ` + "`mode`" + `: from ` + "`$ARGUMENTS`" + ` if specified (all, dedup, recategorize, consolidate), default: all
- ` + "`apply`" + `: false (dry-run by default; pass "apply" in arguments to execute)

Show the suggested actions. Ask for confirmation before applying.
`,
	"module-list": `List all available {{NAME}} plugins/modules.

Call the ` + "`mcp__{{NAME}}__module_list`" + ` tool (no parameters needed).

Show each module with its name, description, and enabled/disabled status.
`,
	"module-run": `Run a {{NAME}} plugin/module to fetch external data into the brain.

Call the ` + "`mcp__{{NAME}}__module_run`" + ` tool with:
- ` + "`name`" + `: ` + "`$ARGUMENTS`" + ` (if provided; omit to run all enabled modules)

Show the import results.
`,
	"setup-commands": `Install {{NAME}} slash commands for Claude Code.

Call the ` + "`mcp__{{NAME}}__setup_commands`" + ` tool to get all command files.

For each command in the result, write the file to the path specified in the "install_dir" field.

After writing all files, tell the user to restart Claude Code to pick up the new commands.
`,
}

// ClaudeCommands returns the slash command files with the agent name substituted.
// The returned map keys are filenames (without .md), values are file contents.
func ClaudeCommands(agentName string) map[string]string {
	cmds := make(map[string]string, len(commandTemplates))
	for name, tmpl := range commandTemplates {
		cmds[name] = strings.ReplaceAll(tmpl, "{{NAME}}", agentName)
	}
	return cmds
}
