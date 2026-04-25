# Download GoCluster

Compiled ready-to-run packages are published on the GitHub Releases page. Use
the latest release:

https://github.com/N2WQ/GoCluster/releases/latest

Download this release asset:

```text
gocluster-windows-amd64.zip
```

Do not use GitHub's automatic `Source code (zip)` or `Source code (tar.gz)`
downloads unless you want the developer source tree. They are not ready-to-run
packages.

Extract the zip and open:

```text
ready_to_run/
```

The extracted directory contains:

```text
ready_to_run/
  gocluster.exe
  README.md
  data/
  docs/
```

Start with `ready_to_run/README.md`. It is generated from the packaged config
for runtime values such as the telnet port.

For a real node, copy `ready_to_run/data/config` to a private complete config
directory such as `ready_to_run/data/config.local`, edit that private copy, and
run with `DXC_CONFIG_PATH` pointing at the private directory.

The source repository does not commit generated binaries or zip files. The
release script attaches build artifacts to GitHub Releases so the Git history
stays source-only.
