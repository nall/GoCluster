# H3 Table Generation (Linux/Ubuntu)

This directory holds the precomputed H3 resolution tables used by the Windows build.
Files:
- res1.bin (842 entries)
- res2.bin (5882 entries)

Each file is a flat list of uint64 H3 cell IDs in little-endian order, sorted
ascending. The implicit ID mapping is index+1 (1-based). Entry 0 is reserved
for InvalidCell.

Below are step-by-step instructions to generate these files on Linux using the
cgo H3 generator.

## Prerequisites
- Linux host with internet access (Ubuntu recommended)
- Go 1.25 installed
- build-essential and git

## Step-by-step

1) Update packages
   sudo apt-get update

2) Install Go, build tools, git
   sudo apt-get install -y golang-go build-essential git

3) Install Go 1.25 (required by this repo)
   curl -fsSL https://go.dev/dl/go1.25.0.linux-amd64.tar.gz -o /tmp/go1.25.0.tar.gz
   sudo rm -rf /usr/local/go
   sudo tar -C /usr/local -xzf /tmp/go1.25.0.tar.gz
   export PATH=/usr/local/go/bin:$PATH
   go version

4) Clone the repo
   git clone https://github.com/N2WQ/gocluster gocluster
   cd gocluster

5) Download deps
   go mod download

6) Run the generator
   go run ./cmd/h3gen --out data/h3

7) Verify output
   ls -l data/h3
   # Expect:
   # res1.bin (6736 bytes)
   # res2.bin (47056 bytes)

8) Copy to Windows host
   scp -r data/h3 <windows-user>@<windows-host>:'C:\src\gocluster\data\h3'

9) Confirm Windows config
   Ensure data/config/data.yaml has:
     h3_table_path: "data/h3"

## Troubleshooting
- If go mod download tries to fetch a newer toolchain and fails, verify Go 1.25
  is installed and on PATH.
- If res1.bin/res2.bin sizes differ, rerun the generator and ensure no old
  files remain in data/h3.

