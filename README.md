# goscoop

Native Go CLI that replaces Scoop's PowerShell backend. Compatible with existing Scoop buckets and manifests. Single binary, zero runtime dependencies, faster installs.

## Install

```
go build -o goscoop.exe .
```

Put `goscoop.exe` somewhere in your `PATH`.

## Commands

| Command | Action |
|---|---|
| `install <app>` | Install an app (multi-threaded download, auto-extract, persist, shim, LZX compress) |
| `update [app]` | Update all buckets / a specific app |
| `uninstall <app> [apps...]` | Remove app(s) (`-p` to purge persist) |
| `list` | Show installed apps |
| `search <query>` | Search across all buckets |
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
| Single binary |  |  |
| Zero runtime dependencies |  |  |
| Multi-threaded download |  |  (4 parts per file) |
| Animated ASCII progress |  |  |
| Auto LZX compression on install |  |  |
| `optimize` command |  |  |
| Start Menu shortcuts |  |  |
| Manifest `depends` |  |  |
| Persist (dirs + files) |  |  |
| Nested archive extraction |  |  |
| Innosetup / MSI / 7z / tar |  |  |
| Pre/post install scripts |  |  |
| Bucket management |  |  |
| Hold / unhold |  |  |
| `list` / `search` / `status` |  |  |
| `cache` management |  |  |
| `--global` support |  |  |

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
