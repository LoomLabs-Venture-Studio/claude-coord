# claude-coord

Lightweight coordination for multiple Claude Code agents working in the same codebase.

## The Problem

Running multiple Claude Code instances in parallel (different terminals, worktrees, or sessions) is powerful — but agents have no awareness of each other. Agent A might modify a database schema while Agent B is querying it. This leads to conflicts, broken migrations, and wasted work.

## The Solution

`claude-coord` adds a simple file-based locking mechanism. Before modifying protected files, agents acquire a lock. If another agent holds the lock, they stop and tell you.

```
Terminal 1                              Terminal 2
──────────                              ──────────
> "Add email verification"              > "Add OAuth support"
                                    
✓ Acquired lock: db/schema/*            Checking locks...
  Editing db/schema/users.sql...   
                                        ✗ BLOCKED: db/schema/*
                                          Locked by: Terminal 1  
                                          Task: "Add email verification"
                                       
                                        "I can't modify the schema right now.
                                         Want me to wait or work on something else?"
                                    
✓ Done, released lock                   
                                        ✓ Lock free, proceeding...
```

Works seamlessly with git worktrees — locks are stored in `.git/claude-coord/` which is shared across all worktrees.

---

## Installation

### Option 1: Go Install

```bash
go install github.com/LoomLabs-Venture-Studio/claude-coord@latest
```

### Option 2: Homebrew (macOS/Linux)

```bash
brew install LoomLabs-Venture-Studio/tap/claude-coord
```

### Option 3: Download Binary

```bash
curl -fsSL https://raw.githubusercontent.com/claude-coord/claude-coord/main/install.sh | bash
```

### Option 4: Build from Source

```bash
git clone https://github.com/LoomLabs-Venture-Studio/claude-coord.git
cd claude-coord
go build -o claude-coord ./cmd/claude-coord
sudo mv claude-coord /usr/local/bin/
```

---

## Quick Start

### 1. Initialize in Your Project

```bash
cd your-project
claude-coord init
```

This creates:
- `.git/claude-coord/config.yaml` — Define what files need protection
- Updates `CLAUDE.md` with coordination instructions

### 2. Configure Protected Files

Edit `.git/claude-coord/config.yaml`:

```yaml
version: 1

protected:
  # Add patterns for files that shouldn't be edited concurrently
  - pattern: "db/schema/*"
  - pattern: "db/migrations/*"
  - pattern: "package.json"
  - pattern: "package-lock.json"
  - pattern: ".env*"
  # Add your own patterns...

settings:
  default_ttl: 300  # Lock timeout in seconds
```

### 3. Add Automatic Enforcement

Create `.claude/settings.json` in your project:

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Edit|Write|MultiEdit|CreateFile",
      "hooks": [{
        "type": "command",
        "command": "claude-coord check --acquire \"$CLAUDE_FILE_PATHS\" --agent \"$CLAUDE_SESSION_ID\""
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "claude-coord unlock --all --agent \"$CLAUDE_SESSION_ID\""
      }]
    }]
  }
}
```

### 4. Done

Now when Claude tries to edit a protected file:
1. Hook fires automatically
2. `claude-coord` checks if file matches a protected pattern
3. If protected, checks for existing locks
4. If locked by another agent → **blocks the edit**
5. If free → acquires lock and allows edit
6. When Claude's session ends → releases locks

---

## Commands

```bash
# Initialize in current project
claude-coord init

# Show all locks and agents
claude-coord status

# Manually lock a resource
claude-coord lock "db/schema/*" --op "Adding new column"

# Release a lock
claude-coord unlock "db/schema/*"

# Check if a file is protected/locked
claude-coord check path/to/file.sql

# Wait for a resource to become available
claude-coord wait "db/schema/*" --timeout 60

# Clean up stale locks
claude-coord gc
```

---

## Configuration

### Protected Patterns

Use glob patterns to define what needs coordination:

```yaml
protected:
  # Exact file
  - pattern: "package.json"
  
  # Directory contents
  - pattern: "db/schema/*"
  
  # Recursive
  - pattern: "src/core/**/*"
  
  # Multiple extensions
  - pattern: "**/*.{sql,prisma}"
  
  # With metadata
  - pattern: "terraform/**/*"
    name: "Infrastructure"
    description: "Terraform state and configs"
```

### Example Configs

<details>
<summary><b>Node.js / TypeScript</b></summary>

```yaml
version: 1
protected:
  - pattern: "package.json"
  - pattern: "package-lock.json"
  - pattern: "tsconfig*.json"
  - pattern: "prisma/schema.prisma"
  - pattern: "prisma/migrations/*"
  - pattern: ".env*"
  - pattern: "**/*.config.{js,ts,mjs}"
```
</details>

<details>
<summary><b>Python</b></summary>

```yaml
version: 1
protected:
  - pattern: "pyproject.toml"
  - pattern: "poetry.lock"
  - pattern: "requirements*.txt"
  - pattern: "alembic/versions/*"
  - pattern: "migrations/*"
  - pattern: ".env*"
```
</details>

<details>
<summary><b>Go</b></summary>

```yaml
version: 1
protected:
  - pattern: "go.mod"
  - pattern: "go.sum"
  - pattern: "db/migrations/*"
  - pattern: "*.proto"
  - pattern: ".env*"
```
</details>

<details>
<summary><b>Monorepo</b></summary>

```yaml
version: 1
protected:
  - pattern: "pnpm-lock.yaml"
  - pattern: "packages/*/package.json"
  - pattern: "packages/shared/**/*"
  - pattern: ".github/workflows/*"
```
</details>

<details>
<summary><b>Infrastructure</b></summary>

```yaml
version: 1
protected:
  - pattern: "terraform/**/*.tf"
  - pattern: "terraform/**/*.tfvars"
  - pattern: "kubernetes/**/*.yaml"
  - pattern: "docker-compose*.yml"
  - pattern: "ansible/**/*"
```
</details>

---

## Git Worktrees

`claude-coord` is designed for worktree workflows. Locks are stored in `.git/claude-coord/` which is automatically shared across all worktrees:

```
repo/                           # Main checkout
├── .git/
│   └── claude-coord/           # ← Shared coordination
│       └── locks/
│           └── db-schema.lock  # Visible to ALL worktrees
└── src/

../repo-feature-a/              # Worktree (Agent A)
├── .git                        # File pointing to repo/.git
└── src/

../repo-feature-b/              # Worktree (Agent B)  
├── .git                        # File pointing to repo/.git
└── src/
                                # Both agents see the same locks!
```

The tool uses `git rev-parse --git-common-dir` to always find the shared location, regardless of whether you're in the main repo or a worktree.

---

## How It Works

```
Claude tries to edit db/schema/users.sql
                │
                ▼
    ┌───────────────────────┐
    │  PreToolUse Hook      │  ← Fires before any edit
    │  runs shell command   │
    └───────────┬───────────┘
                │
                ▼
    claude-coord check --acquire "db/schema/users.sql"
                │
                ├── 1. Load config.yaml
                ├── 2. Match "db/schema/*" pattern ✓
                ├── 3. Check locks/ directory
                │
        ┌───────┴───────┐
        │               │
        ▼               ▼
    Locked by        No lock
    other agent
        │               │
        ▼               ▼
    Exit 1          Create lock
    (blocks         (atomic via O_EXCL)
     edit)          Exit 0
                    (allows edit)
```

Locks are created atomically using `O_EXCL` flag — if two agents try to acquire simultaneously, exactly one succeeds.

---

## What to Commit

```bash
# Commit these (configuration)
git add .git/claude-coord/config.yaml
git add .claude/settings.json
git add CLAUDE.md

# These are gitignored automatically (runtime state)
# .git/claude-coord/locks/
# .git/claude-coord/agents/
```

---

## FAQ

**Does this work with git worktrees?**  
Yes, this is the primary use case. Locks are stored in the shared `.git/` directory.

**What if Claude ignores the instructions?**  
Use the hooks in `.claude/settings.json` — they enforce locking at the Claude Code level, before edits can happen.

**What if an agent crashes?**  
Locks have a TTL (default 5 minutes). Stale locks are automatically skipped. Run `claude-coord gc` to clean them manually.

**What if I'm not using git?**  
Run `claude-coord init --local` to use `.claude-coord/` in the current directory instead.

**Does this work on Windows?**  
Yes. Uses platform-appropriate atomic file operations.

**Can I use this without the CLI?**  
Yes, Claude can manage locks with shell commands. See the `CLAUDE.md` generated by `init`.

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

```bash
git clone https://github.com/LoomLabs-Venture-Studio/claude-coord.git
cd claude-coord
make build   # Build binary
make test    # Run tests
make demo    # Run interactive demo
```

---

## License

MIT — see [LICENSE](LICENSE)
