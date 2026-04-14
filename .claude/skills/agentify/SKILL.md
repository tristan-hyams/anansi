---
name: agentify
description: Bootstrap a repo with ML context files (CLAUDE.md, copilot-instructions.md, AGENTS.md) as identical thin shims pointing to partitioned context docs, then review for accuracy.
user-invocable: true
---

# Agentify a Repository

Bootstrap three identical ML context shims and ensure the central context docs exist.

The shim files (`.claude/CLAUDE.md`, `.github/copilot-instructions.md`, `AGENTS.md`) are **byte-for-byte identical in content** except for the file header and relative paths. They are dumb anchor points — redirecting any AI tool (Claude, Copilot, Codex) to the same partitioned context files. **No content divergence between shims.** Identity, philosophy, commands, workflow conventions — all live in the shared context files, never in a shim.

This pattern maximises context window efficacy by letting agents load only the partitions they need, and streamlines ML onboarding to any repo regardless of which tool the engineer uses.

## Step 1 — Discover repo layout

- Determine the repo root (look for `.git/`, `go.mod`, `package.json`, `*.sln`, etc.)
- **Detect context directory**: check for existing `.context/`, `context/`, `docs/`, or `doc/` directories containing `RULES.md` or `STRUCTURE.md`. If none found, ask the user which directory to use. Default suggestion: `.context/`.
- Check for existing shim files: `.claude/CLAUDE.md`, `.github/copilot-instructions.md`, `AGENTS.md`
- Read any existing versions of all files to understand current state
- Identify the primary language, build system, and package structure

### Framework detection

Detect the primary language and framework to seed RULES.md with language-appropriate sections:

- **Go**: look for `go.mod`, `.golangci.yml`, `Makefile`
- **Python**: look for `pyproject.toml`, `setup.py`, `requirements.txt`, `.flake8`, `ruff.toml`
- **TypeScript/JavaScript**: look for `package.json`, `tsconfig.json`, `.eslintrc.*`, `biome.json`
- **Rust**: look for `Cargo.toml`, `clippy.toml`
- **C# / .NET**: look for `*.sln`, `*.csproj`, `Directory.Build.props`, `.editorconfig`, `global.json`
- **Java/Kotlin**: look for `build.gradle`, `pom.xml`

Use detected language to inform idiomatic patterns, naming conventions, and toolchain sections in RULES.md.

### Existing docs discovery

Scan for existing documentation and config that agents should know about without reading in full:

- `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`
- `Makefile`, `Taskfile.yml`, `justfile`
- `docker-compose.yml`, `Dockerfile`
- Linter configs: `.golangci.yml`, `.eslintrc.*`, `ruff.toml`, `biome.json`, `.prettierrc`
- CI configs: `.github/workflows/`, `.gitlab-ci.yml`, `Jenkinsfile`
- `.env.example`, `.env.test`
- `openapi.yaml`, `swagger.json`, `proto/` directories

Reference discovered files from RULES.md so agents know they exist without loading them all.

### Gitignore check

Verify `.gitignore` exists and includes relevant entries. Suggest additions if missing:
- Build artifacts for the detected language
- IDE directories (`.idea/`, `.vscode/`)
- OS files (`.DS_Store`, `Thumbs.db`)

## Step 2 — Bootstrap {context_dir}/RULES.md (if it doesn't exist)

`{context_dir}` is the directory detected or chosen in Step 1.

Create `{context_dir}/RULES.md` by analyzing the codebase. This is the **single source of truth** for how to write code in this repo. Include:

- **Identity** — what this project is, one-line description, key technology choices
- **Design principles** — the *why* behind patterns, so agents can reason about novel situations
- **Style & formatting** — linters, formatters, naming conventions observed in the code
- **Language-idiomatic patterns** — error handling, dependency injection, concurrency, etc.
- **Rejected patterns** — table of patterns explicitly not used and why
- **Commands** — build, test, lint, run, deploy. Copy-paste runnable.
- **Testing conventions** — test runner, patterns, env/config approach
- **Observability** — logging, tracing, metrics conventions
- **References** — pointers to discovered config files (linters, CI, Docker, API specs). Don't duplicate their content, just note they exist and their purpose.

If it already exists, note it for the review step.

## Step 3 — Bootstrap {context_dir}/STRUCTURE.md (if it doesn't exist)

Create `{context_dir}/STRUCTURE.md` by analyzing the codebase. Include:

- Project overview (what is this repo, what does it do)
- Module/dependency direction (strict layering rules if applicable)
- Directory tree (annotated with purpose per directory)
- Package/module table with responsibility descriptions and key types
- Integration points table (need → package mapping)

If it already exists, note it for the review step.

## Step 4 — Evaluate graduated context files

Beyond RULES.md and STRUCTURE.md, check if the repo warrants additional focused context docs. Create them only if there is enough substance; otherwise fold the content into RULES.md.

| Doc | Create when... |
|-----|---------------|
| `{context_dir}/ARCH.md` | Non-trivial architecture decisions, trade-offs, concurrency models, or scope boundaries that don't fit in RULES or STRUCTURE |
| `{context_dir}/TESTING.md` | Multiple test patterns, integration tests, fixtures, or notable env/config conventions |
| `{context_dir}/DEPLOYMENT.md` | CI/CD pipelines, release scripts, or multi-environment deploy config |
| `{context_dir}/API.md` | OpenAPI specs, proto files, or significant API surface |

Only create additional docs if they would meaningfully reduce RULES.md size. Don't split for the sake of splitting.

## Step 5 — Bootstrap journal directory

Create `{context_dir}/journal/` if it doesn't exist. This is where session-to-session work history and decision rationale is recorded.

Add a **Workflow** section to RULES.md (if not already present) documenting journal conventions:

```markdown
## Agent Workflow

- **Journal:** Read the latest entry in `{context_dir}/journal/` at session start for continuity.
- **Naming:** `YYYY-MM-DD_NN.md` (date + sequence number per day).
- **Content:** Context, decisions made, work done, next steps.
```

## Step 6 — Create the three identical thin shim files

All three shims have **identical content structure**. The only differences are:
- The `# Header` line (tool-specific name)
- Relative paths (adjusted for file location)

No Identity, Philosophy, Commands, or Workflow sections in shims. All of that lives in the context files.

### Template

```markdown
# {HEADER}
<!-- last reviewed: {YYYY-MM-DD} -->

## Context Sources

All coding rules, conventions, commands, and standards:
**[{context_dir}/RULES.md]({relative_path}/RULES.md)**

Package architecture, dependencies, and directory layout:
**[{context_dir}/STRUCTURE.md]({relative_path}/STRUCTURE.md)**
```

If graduated context files were created (ARCH.md, TESTING.md, etc.), add them:

```markdown
Architecture decisions, trade-offs, and rationale:
**[{context_dir}/ARCH.md]({relative_path}/ARCH.md)**
```

### File locations and headers

| File | Header | Path base |
|------|--------|-----------|
| `.claude/CLAUDE.md` | `# CLAUDE.md` | `../{context_dir}/` |
| `.github/copilot-instructions.md` | `# Copilot Instructions` | `../{context_dir}/` |
| `AGENTS.md` (repo root) | `# AGENTS.md` | `{context_dir}/` |

Replace `{YYYY-MM-DD}` with today's date. Replace `{context_dir}` with the actual directory name.

### Path validation

After writing all shim files, verify that every relative path in each shim resolves to an actual file. Report any broken links.

## Step 7 — Review for accuracy

Compare all context files against the actual codebase:

- Are all packages/modules listed in STRUCTURE.md?
- Is the dependency tree accurate?
- Are the coding conventions in RULES.md actually followed in the code?
- Are the commands correct and runnable?
- Are there undocumented patterns that should be captured?
- Are discovered config files (linters, CI, Docker) referenced?
- Do all relative paths in shim files resolve correctly?
- **Are the three shims identical in content** (modulo header and paths)? Flag any divergence.
- Flag any discrepancies or recommendations to the user

### Staleness check

If docs already existed, check the `<!-- last reviewed: -->` comment. If older than 90 days or missing, flag it and update the date after review.

Present a summary of what was created, what was found, and any recommendations.
