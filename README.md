# portkill

A cross-platform CLI tool written in Go to find and kill processes listening on a given port. Works on Windows, macOS, and Linux.

## Install

```bash
go install ://github.com
```

## Usage

Inspect and confirm before killing the process on port `3000`:

```bash
portkill 3000
```

Force kill immediately without a confirmation prompt:

```bash
portkill -f 8080
```
