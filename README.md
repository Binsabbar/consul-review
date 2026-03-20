# consul-review

> Orchestrate multiple AI agents to review GitHub Pull Requests in parallel, then aggregate their findings into a single consolidated review.

---

## What is consul-review?

`consul-review` is a CLI tool for engineering teams who want richer, more diverse automated PR feedback. Instead of relying on a single AI model, it fans out the same review request to multiple AI agents — **Consuls** — concurrently, then uses Claude to synthesise their outputs into one actionable summary.

```
PR #42
  │
  ├─► gemini ──────► review_gemini.md ─┐
  ├─► copilot ─────► review_copilot.md ─┼─► claude (aggregation) ──► stdout
  └─► oz ──────────► review_oz.md ─────┘
```

**Why multiple agents?**
Different models have different strengths, blind spots, and coding knowledge. Running them in parallel and aggregating their output surfaces issues that a single model might miss, reduces false negatives, and gives you a second opinion from another model family — without waiting any longer than the slowest consul.

---

## Prerequisites

| Dependency | Purpose | Install |
|-----------|---------|---------|
| `gh` CLI | Fetch PR diffs and metadata from GitHub | [cli.github.com](https://cli.github.com) |
| `claude` | Final review aggregation | [claude.ai/code](https://claude.ai/code) |
| `gemini` | AI consul _(if enabled)_ | [AI Studio CLI](https://developers.google.com/gemini-api/docs/gemini-cli) |
| `copilot` | AI consul _(if enabled)_ | [GitHub Copilot CLI](https://docs.github.com/en/copilot/github-copilot-in-the-cli) |
| `oz` | AI consul _(if enabled)_ | Internal / your own install |

> **Authentication is your responsibility.** Run each agent's auth command before using `consul-review`. No API keys or tokens are stored in the config file.
>
> ```bash
> gh auth login                  # GitHub (also covers copilot)
> gemini auth login              # Gemini
> oz auth login                  # Oz / Claude
> ```

---

## Installation

### Homebrew _(coming soon)_

```bash
brew install binsabbar/tap/consul-review
```

### Download a release binary

Download the latest binary for your platform from the [Releases page](https://github.com/binsabbar/consul-review/releases), then move it to a directory in your `$PATH`:

```bash
# macOS (Apple Silicon)
curl -sSL https://github.com/binsabbar/consul-review/releases/latest/download/consul-review_darwin_arm64.tar.gz | tar xz
mv consul-review /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/binsabbar/consul-review.git
cd consul-review
make go-build
# Binary is at ./bin/consul-review
```

---

## Quick Start

```bash
# 1. Initialise your config file
consul-review config init

# 2. Edit ~/.consul-review/config.yaml — enable the agents you have installed
#    (see Configuration section below)

# 3. Authenticate each agent you enabled
gh auth login
gemini auth login

# 4. Review a PR
consul-review review --pr 42
```

That's it. The combined review prints to your terminal when all consuls finish.

---

## Configuration

The default config file lives at `~/.consul-review/config.yaml`. Generate it with:

```bash
consul-review config init          # create
consul-review config init --force  # overwrite existing
```

### Full config reference

```yaml
# Optional: path to a custom skill/prompt file.
# Omit → bundled go-code-review skill is used automatically.
# Override per-run with: consul-review review --skill /path/to/SKILL.md
# code_review_skill: "~/.agents/skills/my-review/SKILL.md"

# ─── Consuls ──────────────────────────────────────────────────────────────────
# Enable only the agents you have installed and authenticated.
#
# Built-in non-interactive defaults (used when extra_args is omitted):
#   gemini:  [--yolo]
#   copilot: [--allow-all-tools]
#   oz:      [--no-interactive]
#
# extra_args REPLACES the built-in flags when set — use to add model-specific
# options or experiment with different flags.

gemini:
  enabled: true
  model: "gemini-2.5-pro"
  # extra_args: ["--yolo", "--sandbox"]

copilot:
  enabled: false
  model: "gpt-4"

oz:
  enabled: false
  model: "claude-3-5-sonnet"
```

> **Note:** `extra_args` replaces the non-interactive flags entirely for that consul. The prompt is always appended last.

---

## CLI Reference

### `consul-review review`

Review a GitHub PR using all enabled consuls.

```
consul-review review --pr <PR_NUMBER> [--skill <PATH>] [--config <PATH>]
```

| Flag | Description |
|------|-------------|
| `--pr` | **(Required)** GitHub PR number |
| `--skill` | Path to a skill/prompt file — overrides `code_review_skill` in config and the bundled default |
| `--config` | Path to config file (default: `~/.consul-review/config.yaml`) |
| `--debug` | Enable verbose debug logging |

**Examples:**

```bash
# Review PR #42 with defaults
consul-review review --pr 42

# Use a custom skill file for this run only
consul-review review --pr 42 --skill ./my-project-review.md

# Use a non-standard config
consul-review review --pr 42 --config ./ci-config.yaml
```

### `consul-review config init`

Initialise the default config file.

```
consul-review config init [--force]
```

| Flag | Description |
|------|-------------|
| `--force` | Overwrite an existing config file |

### `consul-review --version`

Print version, commit, build date, and builder.

---

## Skill Files

A **skill file** is a Markdown document that instructs the AI consuls how to review code. It is prepended to the PR diff and title before being sent to each agent.

`consul-review` ships with a **bundled default skill** (`go-code-review`) focused on Go best practices. It is embedded directly in the binary — no extra files required.

**Priority order** (highest wins):

1. `--skill <PATH>` CLI flag (per-run override)
2. `code_review_skill` in your config file (project-level default)
3. Bundled `go-code-review` skill (zero-config fallback)

You can write your own skill for any language or review style and point to it via either mechanism.

---

## How It Works

```
consul-review review --pr 42
         │
         ▼
  Load config + validate
  Check required binaries exist (gh, claude, enabled consuls)
         │
         ▼
  gh pr view  ──► PR title + body
  gh pr diff  ──► unified diff
         │
  Build prompt = skill + PR title + body + diff
         │
  ┌──────┴──────┐
  │ Goroutine 1 │ gemini  -p "<prompt>" --yolo --model gemini-2.5-pro
  │ Goroutine 2 │ copilot -p "<prompt>" --allow-all-tools
  │ Goroutine 3 │ oz agent run --prompt "<prompt>" --no-interactive
  └──────┬──────┘
         │ all run concurrently (WaitGroup)
         ▼
  Collect output files + errors
         │
         ▼
  claude "<summarise review_gemini.md review_copilot.md ...>"
         │
         ▼
  Consolidated review ──► stdout
```

Partial failures are tolerated — if one consul fails, the others continue and the aggregation still runs over the successful outputs.

---

## Development

```bash
# Build
make go-build

# Test (with race detector)
make go-test

# Lint
make go-lint

# Vulnerability scan
make go-vulncheck

# Snapshot release (local, no git tag required)
make release-snapshot
```

### Adding a changelog entry

```bash
changie new        # select kind, enter description
git add .changes/
git commit -m "chore(changes): add changelog entry"
```

### Cutting a release

1. Run the **prepare-release** GitHub Actions workflow (select version + type)
   — it batches the changelog and opens a PR automatically
2. Merge the PR
3. Run the **release** GitHub Actions workflow (select version + type)
   — it creates the git tag and publishes binaries via GoReleaser

---

## Roadmap

### v0.x — Current (CLI-based)
- ✅ Parallel multi-agent PR review
- ✅ Claude aggregation
- ✅ Configurable non-interactive flags (`extra_args`)
- ✅ Custom skill files with bundled fallback
- ✅ `config init` subcommand
- ✅ `--skill` runtime override

### Future milestone — Direct API Integration
> **No binary dependencies.** The next major milestone is direct API integration with each agent's cloud API (Gemini API, OpenAI API, Anthropic API). This will eliminate the requirement to have `gemini`, `copilot`, and `oz` CLI binaries installed locally, making `consul-review` truly self-contained and easier to run in CI/CD pipelines.

---

## License

MIT — see [LICENSE](./LICENSE).
