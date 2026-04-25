# External Authorities

Use official upstream documentation for general Go, GitHub, Linux/systemd, and
PowerShell questions. Link to these sources instead of copying their content
into this repository.

## Go

- Go documentation: https://go.dev/doc/
- Install Go: https://go.dev/doc/install
- Go standard library and packages: https://pkg.go.dev/
- Go command docs, modules, fuzzing, diagnostics, memory model, race detector,
  and profiling should route through https://go.dev/doc/.

## GitHub

- GitHub Docs: https://docs.github.com/en
- GitHub Actions: https://docs.github.com/en/actions
- Releases: https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases
- Pull requests: https://docs.github.com/en/pull-requests
- Git basics through GitHub: https://docs.github.com/en/get-started/using-git/about-git

## Linux And systemd

- systemd unit docs: https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html
- systemd service docs: https://man7.org/linux/man-pages/man5/systemd.service.5.html
- journalctl docs: https://man7.org/linux/man-pages/man1/journalctl.1%40%40systemd.html

For machine-specific Linux answers, prefer distro-local manuals when available:

- `man systemd.service`
- `man systemd.unit`
- `man systemctl`
- `man journalctl`

## PowerShell

- PowerShell docs: https://learn.microsoft.com/en-us/powershell/
- PowerShell overview: https://learn.microsoft.com/en-us/powershell/scripting/overview
- Install PowerShell on Windows: https://learn.microsoft.com/en-us/powershell/scripting/install/install-powershell-on-windows
- PowerShell 101: https://learn.microsoft.com/en-us/powershell/scripting/learn/ps101/00-introduction

## Use Rules

- For GoCluster-specific behavior, prefer repo docs over external docs.
- For tool behavior outside this repository, prefer the official upstream docs.
- If upstream docs are versioned or change over time, tell the user to verify
  against the current upstream page.
