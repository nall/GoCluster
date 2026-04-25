# Download GoCluster

Compiled Windows packages are published on the GitHub Releases page when a
release has been created:

https://github.com/N2WQ/GoCluster/releases

Download this asset:

```text
gocluster-windows-amd64.zip
```

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

The source repository does not commit generated binaries or zip files. The
release script attaches build artifacts to GitHub Releases so the Git history
stays source-only.
