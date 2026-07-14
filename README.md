# goscoop

Native Go CLI that replaces Scoop's PowerShell backend. Compatible with existing Scoop buckets and manifests. Single binary, zero runtime dependencies, faster installs.

## Install

### One-liner (PowerShell)

**If you already have Scoop** (put in Scoop's shims directory, already on PATH):
```powershell
iwr -Uri https://github.com/lque36708-pixel/goscoop/releases/latest/download/goscoop.exe -OutFile "$env:USERPROFILE\scoop\shims\goscoop.exe"
```

**Standalone install** (creates `~\goscoop\` and adds to PATH):
```powershell
md "$env:USERPROFILE\goscoop" -Force; iwr -Uri https://github.com/lque36708-pixel/goscoop/releases/latest/download/goscoop.exe -OutFile "$env:USERPROFILE\goscoop\goscoop.exe"; setx PATH "$env:PATH;$env:USERPROFILE\goscoop"
```

No admin required. Restart your terminal after `setx`.

### Via `go install`

```bash
go install github.com/lque36708-pixel/goscoop@latest
```

### From source

```bash
git clone https://github.com/lque36708-pixel/goscoop.git
cd goscoop
go build -o goscoop.exe .
```

Put `goscoop.exe` somewhere in your PATH (e.g. `~\scoop\shims\` or `~\goscoop\`).

## Usage

```
goscoop search chrome
goscoop install googlechrome
goscoop list
goscoop update
goscoop uninstall googlechrome
```

## Commands

| Command | Action |
|---|---|
| `install <app>` | Install an app (multi-threaded download, auto-extract, persist, shim, LZX compress) |
| `update [app]` | Update all buckets / a specific app |
| `uninstall <app> [apps...]` | Remove app(s) (`-p` to purge persist) |
| `list` | Show installed apps |
| `search <query>` | Search across all buckets (auto-cached after first run) |
| `status` | Show outdated apps (respects `.hold`) |
| `info <app>` | Show manifest details |
| `bucket list\|add\|rm` | Manage buckets |
| `cache list\|rm [app]` | Manage download cache |
| `hold/unhold <app>` | Pin an app to prevent updates |
| `reset <app>` | Reinstall shims |
| `optimize` | Compact all apps with LZX compression |
| `--global`/`-g` | Install to `%ProgramData%\scoop` |

## Feature comparison

| Feature | Scoop (PS) | goscoop |
|---|---|---|
| Single binary | | ✓ |
| Zero runtime dependencies | | ✓ |
| Multi-threaded download | | ✓ (4 parts per file) |
| Animated ASCII progress | | ✓ |
| Auto LZX compression on install | | ✓ |
| `optimize` command | | ✓ |
| Start Menu shortcuts | ✓ | ✓ |
| Manifest `depends` | ✓ | ✓ |
| Persist (dirs + files) | ✓ | ✓ |
| Nested archive extraction | ✓ | ✓ |
| Innosetup / MSI / 7z / tar | ✓ | ✓ |
| Pre/post install scripts | ✓ | ✓ |
| Bucket management | ✓ | ✓ |
| Hold / unhold | ✓ | ✓ |
| Search cache / index | | ✓ |
| `list` / `search` / `status` | ✓ | ✓ |
| `cache` management | ✓ | ✓ |
| `--global` support | ✓ | ✓ |
| Suggest similar name on typo | | ✓ |
| Warn on leftover installation | | ✓ |

## v0.1.1 — UX improvements

- All commands now handle missing scoop directories gracefully (friendly messages instead of raw Go errors)
- `findInnounp` respects `SCOOP` env var instead of hard-coded paths
- Non-fatal search index rebuild after bucket update
- Typo detection in `uninstall` + leftover binary warnings

## Not yet implemented

| Command | Original Scoop |
|---|---|
| `cleanup` | Remove old versions |
| `home <app>` | Open homepage |
| `which <cmd>` | Find app that owns a shim |
| `prefix` | Show scoop directory |
| `virustotal` | Check hashes on VirusTotal |
| `cat <app>` | Show manifest content |
| `config` | Manage settings |
| `checkup` | Check for issues |
| 32-bit architecture | `--arch 32bit` |
