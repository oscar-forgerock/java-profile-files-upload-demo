# JFR CLI - JFR Recording Management Tool

A command-line tool to manage Java Flight Recorder (JFR) recordings in Kubernetes pods.

## Features

- **Create** on-demand JFR recordings with custom duration and name
- **Stop** active JFR recordings by name
- **List** all active JFR recordings in a pod

## Installation

### Build from source

```bash
cd jfr-cli
make build
```

The binary will be available at `bin/jfr-cli`.

### Install to system

```bash
make install
```

This installs the binary to `$GOPATH/bin/jfr-cli`.

## Usage

### Prerequisites

- `kubectl` must be installed and configured
- Access to the Kubernetes cluster where the Java pods are running

### Basic Commands

```bash
# List all active JFR recordings
jfr-cli -p java-jfr-with-sidecar-0 list

# Create a 60-second recording with auto-generated name
jfr-cli -p java-jfr-with-sidecar-0 create 60s

# Create a 30-second recording with custom name
jfr-cli -p java-jfr-with-sidecar-0 create 30s my-profile

# Stop a recording by name
jfr-cli -p java-jfr-with-sidecar-0 stop my-profile
```

### Using Pod Selector

Instead of specifying the pod name, you can use a label selector:

```bash
# Use label selector to find the pod
jfr-cli -l app=java-jfr-with-sidecar create 60s

# Specify namespace and selector
jfr-cli -n production -l app=java-jfr-with-sidecar list
```

### Flags

- `-n, --namespace <ns>` - Kubernetes namespace (default: `default`)
- `-p, --pod <name>` - Pod name
- `-c, --container <name>` - Container name (default: `java-app`)
- `-l, --selector <selector>` - Pod label selector (default: `app=java-jfr-with-sidecar`)

## Examples

### Create a recording

```bash
# Auto-generated name, 60s duration
$ jfr-cli -p java-jfr-with-sidecar-0 create 60s
Using pod: java-jfr-with-sidecar-0
Creating JFR recording...
✓ JFR recording started successfully
  Pod:      java-jfr-with-sidecar-0
  Name:     jfr_2026-01-22T08-30-15+00-00
  Duration: 60s
  Filename: jfr_2026-01-22T08-30-15+00-00.jfr
  Path:     /tmp/jfr/jfr_2026-01-22T08-30-15+00-00.jfr

# Custom name, 30s duration
$ jfr-cli -p java-jfr-with-sidecar-0 create 30s my-profile
```

### List active recordings

```bash
$ jfr-cli -p java-jfr-with-sidecar-0 list
Using pod: java-jfr-with-sidecar-0
Listing JFR recordings...

Pod: java-jfr-with-sidecar-0
Active JFR recordings (2):

  ID:    1
  Name:  jfr_2026-01-22T08-30-15+00-00
  State: running

  ID:    2
  Name:  my-profile
  State: running
```

### Stop a recording

```bash
$ jfr-cli -p java-jfr-with-sidecar-0 stop my-profile
Using pod: java-jfr-with-sidecar-0
Stopping JFR recording...
✓ JFR recording stopped successfully
  Pod:  java-jfr-with-sidecar-0
  Name: my-profile
```

## How It Works

The CLI tool uses `kubectl exec` to run `jcmd` commands inside the Java container:

1. **CREATE**: Executes `jcmd 1 JFR.start duration=<duration>,name=<name>,filename=/tmp/jfr/<name>.jfr`
2. **STOP**: Executes `jcmd 1 JFR.stop name=<name>`
3. **LIST**: Executes `jcmd 1 JFR.check` and parses the output

All JFR files are saved to `/tmp/jfr/` in the pod, which is mapped to a HostPath volume for persistence.

## Recording Name Format

Auto-generated recording names follow RFC3339 timestamp format with colons replaced by hyphens:

```
jfr_2026-01-22T08-30-15+00-00
```

The corresponding JFR file is saved as:

```
jfr_2026-01-22T08-30-15+00-00.jfr
```

## Development

### Build for multiple platforms

```bash
make build-all
```

This creates binaries for:
- macOS (AMD64 and ARM64)
- Linux (AMD64 and ARM64)

### Clean build artifacts

```bash
make clean
```