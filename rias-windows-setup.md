# Rias — Windows Setup Guide

> Recommended approach: WSL2 (Windows Subsystem for Linux). All commands are identical to macOS/Linux once inside Ubuntu.

---

## Step 1: Install WSL2 + Ubuntu

Open **PowerShell as Administrator** and run:

```powershell
wsl --install
```

This installs WSL2 and Ubuntu automatically. When it finishes, **reboot your PC**.

After rebooting, Ubuntu will launch and ask you to create a Linux username and password. Set those up.

> If WSL is already installed, update it first:
> ```powershell
> wsl --update
> wsl --set-default-version 2
> ```

---

## How to Open Ubuntu

After installation, open Ubuntu using any of these:

| Method | How |
|---|---|
| Start menu | Search "Ubuntu" → open the Ubuntu app |
| Windows Terminal | Click the ▼ dropdown next to `+` → select "Ubuntu" |
| PowerShell / CMD | Type `wsl` or `wsl -d Ubuntu` |
| Run dialog | Press `Win + R` → type `wsl` → Enter |

You'll see a prompt like `username@DESKTOP-XXXXX:~$` — that's your Linux shell.

---

## Step 2: Install Go and Dependencies

Open Ubuntu and run:

```bash
# Install build tools and git
sudo apt update && sudo apt install -y git curl build-essential

# Install Go 1.24
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz
rm go1.24.2.linux-amd64.tar.gz

# Add Go to PATH permanently
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc && source ~/.bashrc
```

Verify:

```bash
go version
```

Expected: `go version go1.24.2 linux/amd64`

---

## Step 3: Install GitHub CLI

```bash
# Install gh CLI
(type -p wget >/dev/null || (sudo apt update && sudo apt-get install wget -y)) \
&& sudo mkdir -p -m 755 /etc/apt/keyrings \
&& out=$(mktemp) && wget -nv -O$out https://cli.github.com/packages/githubcli-archive-keyring.gpg \
&& cat $out | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null \
&& sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
&& echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
&& sudo apt update \
&& sudo apt install gh -y
```

Authenticate:

```bash
gh auth login
```

Follow the prompts — select GitHub.com → HTTPS → authenticate via browser.

---

## Step 4: Set Up SSH Key for GitHub (optional but recommended)

```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "your-email@example.com"

# Add to SSH agent
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519

# Copy public key
cat ~/.ssh/id_ed25519.pub
```

Go to GitHub → Settings → SSH Keys → New SSH Key → paste the output above.

Test:

```bash
ssh -T git@github.com
```

Expected: `Hi TruyLabs! You've authenticated...`

---

## Step 5: Clone and Build Rias

```bash
# Create project directory
mkdir -p ~/code/personal && cd ~/code/personal

# Clone the fork
git clone git@github.com:TruyLabs/rias.git
cd rias

# Build
make build

# Verify binary
./rias version
```

Expected: `rias v1.2.0-... (commit: ..., built: ...)`

---

## Step 6: Install Globally

```bash
make install
```

Verify:

```bash
rias version
```

If `rias: command not found`, run:

```bash
source ~/.bashrc
```

---

## Step 7: Run Setup

```bash
rias setup --local
```

Expected output:

```
  Brain:    /home/<you>/.rias/brain
  Sessions: /home/<you>/.rias/sessions
  Config:   /home/<you>/.rias/config.yaml
  Mode: local binary (...)
  MCP config: /home/<you>/.mcp.json
  Commands: /home/<you>/.claude/commands/rias (10 slash commands)

Setup complete! rias is ready as an MCP server for Claude Code.
```

Verify brain directories:

```bash
ls ~/.rias/brain/
```

Expected: `decisions  expertise  goals  identity  knowledge  opinions  style  tasks`

---

## Step 8: Configure Your API Key

```bash
rias auth set-key --provider claude
```

Paste your Anthropic API key when prompted.

To use other providers:

```bash
rias auth set-key --provider openai   # OpenAI / GPT-4o
rias auth set-key --provider gemini   # Google Gemini
```

---

## Step 9: Teach Rias About Yourself

```bash
rias teach
```

Type something about yourself when prompted, for example:

```
I am a software developer. I prefer Go for backend and React for frontend.
I value clean architecture and prefer integration tests over unit tests.
```

Press Enter twice or Ctrl+D to finish.

---

## Step 10: Verify It Works

```bash
rias ask "What do you know about me?"
```

Expected: rias answers based on what you taught it in Step 9.

---

## Step 11: MCP Integration (Claude Code / Cursor on Windows)

The MCP config file is at `~/.mcp.json` inside WSL — but Claude Code and Cursor run on Windows, not inside WSL. You need to register the MCP server on the Windows side.

### For Claude Code (Windows)

Add to `%USERPROFILE%\.mcp.json` (create it if it doesn't exist):

```json
{
  "mcpServers": {
    "rias": {
      "type": "stdio",
      "command": "wsl",
      "args": ["rias", "mcp", "--config", "/home/<your-ubuntu-username>/.rias/config.yaml"],
      "env": {}
    }
  }
}
```

Replace `<your-ubuntu-username>` with your Ubuntu username (run `whoami` in Ubuntu to check).

### For Cursor (Windows)

Add to `%USERPROFILE%\.cursor\mcp.json`:

```json
{
  "mcpServers": {
    "rias": {
      "type": "stdio",
      "command": "wsl",
      "args": ["rias", "mcp", "--config", "/home/<your-ubuntu-username>/.rias/config.yaml"],
      "env": {}
    }
  }
}
```

Restart Claude Code / Cursor after saving. The `/rias:ask`, `/rias:teach`, and `/rias:brain-search` slash commands will be available.

---

## Daily Usage

All commands run inside your Ubuntu terminal:

```bash
rias                          # start interactive chat
rias ask "your question"      # one-shot question
rias teach                    # teach rias something new
rias brain search "topic"     # search your brain
rias dashboard                # open brain visualizer in browser
rias reflect                  # (Plan 3) analyze sessions → update brain/style
```

---

## Notes

- **Brain location:** `~/.rias/brain/` inside WSL (`\\wsl$\Ubuntu\home\<you>\.rias\brain\` from Windows Explorer)
- **Config location:** `~/.rias/config.yaml`
- **Sessions:** `~/.rias/sessions/`
- **Provider switching:** Edit `~/.rias/config.yaml` → change `provider: claude` to `provider: openai`, `gemini`, or `ollama`
- **Ollama (local models):** Install Ollama for Windows from https://ollama.com, then set `provider: openai` with `base_url: http://localhost:11434/v1` and `model: llama3` in config.yaml
