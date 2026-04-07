# JIT CLI

`jit` is a non-interactive CLI for JIT, designed for AI agents and scripts.
Phase 1 uses PAT (`jit_pat_*`) as the default auth mechanism and reuses existing JIT backend APIs through `Authorization: Bearer <token>`.

## Install

### 1. Download binary from Release

Expected release assets (from GoReleaser):

- `jit-darwin-amd64.tar.gz`
- `jit-darwin-arm64.tar.gz`
- `jit-linux-amd64.tar.gz`
- `jit-linux-arm64.tar.gz`
- `jit-windows-amd64.zip`
- `jit-windows-arm64.zip`

After extracting:

```bash
chmod +x jit
mv jit ~/.local/bin/jit
```

### 2. Install via script

Linux/macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/${OWNER}/${REPO}/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/${OWNER}/${REPO}/main/scripts/install.ps1 | iex
```

Optional install env vars:

- `JIT_CLI_REPO` (default: `wanyun/JitCli`)
- `JIT_CLI_VERSION` (default: `latest`)
- `JIT_CLI_INSTALL_DIR` (default: `~/.local/bin` on Unix, `~/.local/bin` on Windows PowerShell)

## Authentication (PAT + Profile)

Create a PAT in JIT Web personal center, then login:

```bash
jit auth login --server https://demo.jit.cn --app wanyun/JitAi --token jit_pat_xxx_yyy
```

Or pipe token via stdin:

```bash
printf '%s' 'jit_pat_xxx_yyy' | jit auth login --server https://demo.jit.cn --app wanyun/JitAi
```

Check current identity:

```bash
jit auth status
jit whoami
```

Profile operations:

```bash
jit auth list
jit auth use demo
jit auth logout --profile demo
```

## API Usage

Raw API gateway (default: current profile + `default_app`):

```bash
jit api services/JitAISvc/sendMessage --data '{"assistantId":"a","chatId":"c","message":"hello"}'
```

Specify app explicitly:

```bash
jit api auths/loginTypes/services/AuthSvc/listCliTokens --app wanyun/JitAuth
```

Model examples:

```bash
jit model list
jit model meta
jit model info wanyun.crm.Customer
jit model query wanyun.crm.Customer --filter '{}' --page 1 --size 10 --app erp_demo/ErpApp
```

Global flags:

- `--profile <name>`
- `--app <org/app>`
- `--jq <expr>`
- `--format json` (Phase 1 only)
- `--dry-run`

## Exit Codes

- `0`: request success and backend `errcode == 0`
- `1`: request success but backend `errcode != 0` (business error)
- `2`: CLI-side error (network/auth/invalid args/profile not found, etc.)

`jit api` prints backend raw response to stdout.
CLI-side errors are printed as JSON to stderr.

## Build and Release

Local build:

```bash
make build
```

Run tests:

```bash
make test
```

GoReleaser snapshot:

```bash
make snapshot
```

## Notes

- Phase 1 is JSON-first and script-friendly.
- `--format table` is intentionally not included in Phase 1.
- `jit api` does not wrap backend payload into a custom CLI envelope.
