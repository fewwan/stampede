# 🐐 Stampede

`stampede` is a simple CLI tool to run multiple commands concurrently, each with optional labels and colored output.

## Usage

```
Usage:
  stampede [options] '[label] command' '[label] command' ...
  stampede --from tasks.txt

Examples:
  # Run commands with optional labels (labels in square brackets)
  stampede "[Google] ping -c 3 8.8.8.8" "[Cloudflare] ping -c 3 1.1.1.1"

  # Run commands without labels; labels will be inferred from executable names
  stampede "ping -c 3 8.8.8.8" "ping -c 3 1.1.1.1"

  # Load commands from file (one per line, optional labels allowed)
  stampede --from commands.txt

Flags:
  -a, --abort-on-fail   Stop all tasks if any fail
  -f, --from string     Load tasks from file
      --max int         Maximum concurrent tasks (0 = unlimited)
      --no-color        Disable color output
  -q, --quiet           Suppress command run messages and summary
  -r, --raw             Disable output labels
```

## Installation

1. Install from source:
```bash
go install github.com/fewwan/stampede@latest
```

2. Or clone and build:
```bash
git clone https://github.com/fewwan/stampede.git
cd stampede
go build
```
## Example
<img width="1920" height="1080" alt="image" src="https://github.com/user-attachments/assets/0c003842-286c-434d-9d19-d8b4a6f4498b" />

