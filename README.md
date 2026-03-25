# consul-review

> Orchestrate multiple AI agents to review GitHub Pull Requests in parallel.

---

## What is consul-review?

`consul-review` is a CLI tool that fans out a PR review request to multiple AI agents — **Consuls** — concurrently and prints each agent's full review to stdout when all are done.

```
consul-review review --pr 42 --repo github.com/owner/repo
        │
        ├─► gemini  ─────► (review output) ─┐
        ├─► copilot ─────► (review output) ─┼─► stdout (each review printed with header)
        ├─► oz ──────────► (review output) ─┤
        ├─► claude ──────► (review output) ─┤
        └─► codex ───────► (review output) ─┘
```

**How PR retrieval works:** The tool does _not_ fetch the PR itself. Instead it passes the skill file, repository, and PR number directly to each agent binary. The **skill file instructs each agent** on how to fetch the PR details — this could be via the `gh` CLI, an MCP server, a browser tool, or any other mechanism the agent supports. This keeps `consul-review` as a pure orchestration layer with no hard dependency on `gh` or any aggregation tool.

**Why multiple agents?** Different models surface different issues. Running them in parallel costs no extra wall-clock time — you wait as long as the slowest consul.

---

## Prerequisites

| Dependency | Purpose | Install |
|-----------|---------|---------|
| `gemini` | AI consul _(if enabled)_ | [AI Studio CLI](https://developers.google.com/gemini-api/docs/gemini-cli) |
| `copilot` | AI consul _(if enabled)_ | [GitHub Copilot CLI](https://docs.github.com/en/copilot/github-copilot-in-the-cli) |
| `oz` | AI consul _(if enabled)_ | Internal / your own install |
| `claude` | AI consul _(if enabled)_ | [Claude Code](https://docs.anthropic.com/en/docs/claude-code) |
| `codex` | AI consul _(if enabled)_ | [OpenAI Codex CLI](https://github.com/openai/codex) |

> **Authentication is your responsibility.** Run each agent's auth command before using `consul-review`. No API keys or tokens are stored in the config file.
>
> ```bash
> gemini auth login              # Gemini
> gh auth login                  # GitHub Copilot
> oz auth login                  # Oz
> claude auth login              # Claude Code
> codex auth login               # Codex CLI
> ```

---

## Installation

### Homebrew

```bash
brew install binsabbar/tap/consul-review
```

### Download a release binary

Download the latest binary from the [Releases page](https://github.com/binsabbar/consul-review/releases):

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

# 2. Edit ~/.consul-review/config.yaml
#    Set your repo and enable the agents you have installed

# 3. Authenticate each enabled agent
gemini auth login

# 4. Review a PR
consul-review review --pr 42
```

Each agent's full review prints to stdout when all consuls finish.

---

## Configuration

Generate the default config at `~/.consul-review/config.yaml`:

```bash
consul-review config init          # create
consul-review config init --force  # overwrite existing
```

### Full config reference

```yaml
# Full GitHub repository path including hostname (required).
# Can also be overridden per-run with: --repo github.com/owner/repo
repo: "github.com/owner/repo"

# Optional: path to a custom skill/prompt file.
# Omit → the bundled go-code-review skill is used automatically.
# Override per-run with: --skill /path/to/SKILL.md
# code_review_skill: "~/.agents/skills/my-review/SKILL.md"

# ─── Consuls ──────────────────────────────────────────────────────────────────
# Enable only the agents you have installed and authenticated.
#
# Built-in non-interactive defaults (used when extra_args is omitted):
#   gemini:  [--yolo]
#   copilot: [--allow-all-tools]
#   oz:      [--no-interactive]
#   claude:  [--dangerously-skip-permissions]
#   codex:   [-q]
#
# extra_args REPLACES the built-in flags when set.

gemini:
  enabled: true
  model: "gemini-2.5-pro"
  # extra_args: ["--yolo", "--sandbox"]

copilot:
  enabled: false
  model: "gpt-5.2"

oz:
  enabled: false
  model: "claude-4-sonnet"

claude:
  enabled: false
  model: "claude-sonnet-4-6"

codex:
  enabled: false
  model: "gpt-5.3-codex"
```

---

## CLI Reference

### `consul-review review`

Review a GitHub PR using all enabled consuls.

```
consul-review review --pr <PR_NUMBER> [--repo <HOST/OWNER/REPO>] [--skill <PATH>] [--config <PATH>]
```

| Flag | Description |
|------|-------------|
| `--pr` | **(Required)** GitHub PR number |
| `--repo` | Full GitHub path including hostname, e.g. `github.com/owner/repo` (overrides config) |
| `--skill` | Path to a skill/prompt file — overrides `code_review_skill` in config and the bundled default |
| `--config` | Path to config file (default: `~/.consul-review/config.yaml`) |
| `--debug` | Enable verbose debug logging |

**Examples:**

```bash
# Review PR #42 (repo set in config)
consul-review review --pr 42

# Override repo for this run
consul-review review --pr 42 --repo github.com/my-org/my-service

# Use a custom skill file for this run only
consul-review review --pr 42 --skill ./my-project-review.md

# Use a non-standard config (e.g. in CI)
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

A **skill file** is a Markdown document that:
1. Defines the review standards and focus areas for each agent
2. **Instructs each agent how to fetch PR details** (e.g. via `gh pr view`, an MCP tool, web browsing — whatever the agent supports)

`consul-review` ships with a **bundled default skill** (`go-code-review`) focused on Go best practices, embedded directly in the binary — no extra files required.

**Priority order** (highest wins):

1. `--skill <PATH>` CLI flag (per-run override)
2. `code_review_skill` in your config file (project-level default)
3. Bundled `go-code-review` skill (zero-config fallback)

---

## How It Works

```
consul-review review --pr 42 --repo github.com/owner/repo
         │
         ▼
  Load config + validate
  Check enabled consul binaries exist in PATH
         │
         ▼
  Build prompt = skill content + repo + PR number
  (skill instructs each agent on HOW to fetch the PR)
         │
  ┌──────┴──────┐
  │ Goroutine 1 │  gemini  -p "<prompt>" --yolo --model gemini-2.5-pro
  │ Goroutine 2 │  copilot -p "<prompt>" --allow-all-tools
  │ Goroutine 3 │  oz agent run --prompt "<prompt>" --no-interactive
  │ Goroutine 4 │  claude -p "<prompt>" --dangerously-skip-permissions --model claude-sonnet-4-6
  │ Goroutine 5 │  codex -q "<prompt>" --model gpt-5.3-codex
  └──────┬──────┘
         │ all run concurrently (sync.WaitGroup)
         ▼
  Print each agent's full review to stdout
```

Each agent binary reads the skill + repo + PR number and decides how to fetch the PR data (gh CLI, MCP, API, etc.). The orchestrator only cares about the final review text.

Partial failures are tolerated — if one consul crashes, the others continue and successful reviews are still printed.

---

## Development

```bash
make go-build          # build binary → ./bin/consul-review
make go-test           # run tests with race detector
make go-lint           # run golangci-lint
make go-vulncheck      # run govulncheck
make release-snapshot  # local GoReleaser snapshot (no tag needed)
```

### Adding a changelog entry

```bash
changie new        # select kind, enter description
git add .changes/
git commit -m "chore(changes): add changelog entry"
```

### Cutting a release

1. Run the **prepare-release** GitHub Actions workflow (select version)
   — it batches the changelog and opens a PR automatically
2. Merge the PR
3. Run the **release** GitHub Actions workflow
   — it creates the git tag and publishes binaries via GoReleaser

---

## Roadmap

### v0.x — Current (CLI-based)
- ✅ Parallel multi-agent PR review (pure orchestration, no gh dependency)
- ✅ Each agent uses its own skill-instructed method to fetch PRs
- ✅ Configurable non-interactive flags (`extra_args`)
- ✅ Custom skill files with bundled fallback
- ✅ `config init` subcommand
- ✅ `--skill` and `--repo` runtime overrides
- ✅ Clean `Agent` interface — binary and API backends are plug-and-play

### Future milestone — Direct API Integration
> **No binary dependencies.** The next major milestone is direct API integration with each agent's cloud (Gemini API, OpenAI API, Anthropic API). This will eliminate the requirement to have `gemini`, `copilot`, and `oz` CLI binaries installed locally, making `consul-review` truly self-contained and easy to run in CI/CD pipelines. The `Agent` interface is already designed for this — adding a new backend is a single new implementation file with zero changes to the orchestrator or CLI.

---

## License

MIT — see [LICENSE](./LICENSE).
