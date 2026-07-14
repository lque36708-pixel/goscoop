# goscoop

Native Go CLI that replaces Scoop's PowerShell backend. Compatible with existing Scoop buckets and manifests. Single binary, zero runtime dependencies, faster installs.

## Install

### One-liner (cmd)

**If you already have Scoop** (put in Scoop's shims directory, already on PATH):
```cmd
curl -Lo "%USERPROFILE%\scoop\shims\goscoop.exe" https://github.com/lque36708-pixel/goscoop/releases/latest/download/goscoop.exe
```

**Standalone install** (creates `~\goscoop\` and adds to PATH):
```cmd
md "%USERPROFILE%\goscoop" && curl -Lo "%USERPROFILE%\goscoop\goscoop.exe" https://github.com/lque36708-pixel/goscoop/releases/latest/download/goscoop.exe && setx PATH "%PATH%;%USERPROFILE%\goscoop"
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
| `uninstall <app> [apps...]` | Remove app(s) (`-p` to purge persist; `--self` to remove goscoop entirely) |
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

[Đọc bằng tiếng Việt](README_vie.md)
