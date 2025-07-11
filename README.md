# stampede

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

## Example

```bash
$ stampede '[google] ping -c 4 8.8.8.8' '[cloudflare] ping -c 4 1.1.1.1'
cloudflare | Running: ping -c 4 1.1.1.1
google     | Running: ping -c 4 8.8.8.8
google     | PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
google     | 64 bytes from 8.8.8.8: icmp_seq=1 ttl=118 time=12.8 ms
cloudflare | PING 1.1.1.1 (1.1.1.1) 56(84) bytes of data.
cloudflare | 64 bytes from 1.1.1.1: icmp_seq=1 ttl=56 time=13.6 ms
google     | 64 bytes from 8.8.8.8: icmp_seq=2 ttl=118 time=14.0 ms
cloudflare | 64 bytes from 1.1.1.1: icmp_seq=2 ttl=56 time=14.4 ms
google     | 64 bytes from 8.8.8.8: icmp_seq=3 ttl=118 time=14.0 ms
cloudflare | 64 bytes from 1.1.1.1: icmp_seq=3 ttl=56 time=14.3 ms
google     | 64 bytes from 8.8.8.8: icmp_seq=4 ttl=118 time=14.1 ms
google     |
google     | --- 8.8.8.8 ping statistics ---
google     | 4 packets transmitted, 4 received, 0% packet loss, time 2997ms
google     | rtt min/avg/max/mdev = 12.823/13.737/14.120/0.531 ms
cloudflare | 64 bytes from 1.1.1.1: icmp_seq=4 ttl=56 time=14.4 ms
cloudflare |
cloudflare | --- 1.1.1.1 ping statistics ---
cloudflare | 4 packets transmitted, 4 received, 0% packet loss, time 2997ms
cloudflare | rtt min/avg/max/mdev = 13.575/14.169/14.408/0.343 ms

Tasks finished: 2 / 2 succeeded, 0 failed
All tasks completed successfully!
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
