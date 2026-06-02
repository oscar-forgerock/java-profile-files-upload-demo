# Java Profiler with CLI Management

> **Version 2.0 - Sidecarless Architecture**
> This branch implements a simplified architecture without the Go sidecar container. JFR recordings are managed via an external CLI tool using `kubectl exec`.

An automated JVM profiling and observability system that enables on-demand profiling of Java applications running in Kubernetes with automatic profile upload to Google Cloud Storage.

## 🎯 Overview

This system consists of three main components:

1. **Java Application**: A minimal "Hello World" web service with JFR profiling capabilities
2. **JFR CLI Tool**: External Go CLI tool that manages JFR recordings via `kubectl exec`
3. **Go DaemonSet**: File scanner that automatically uploads profile files to GCS
4. **PreStop Hook**: Shell script that stops JFR recordings and copies GC logs during pod termination

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Developer Workstation                                       │
│                                                             │
│  ┌────────────┐                                             │
│  │ JFR CLI    │──kubectl exec──┐                            │
│  └────────────┘                │                            │
└────────────────────────────────┼────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│ Kubernetes Node                                             │
│                                                             │
│  ┌──────────────────┐         ┌──────────────────┐         │
│  │ Java Pod         │         │ Go DaemonSet     │         │
│  │                  │         │                  │         │
│  │  ┌────────────┐  │         │  ┌────────────┐  │         │
│  │  │ Java App   │◄─┼─jcmd─   │  │ Scanner    │  │         │
│  │  │  +JFR      │  │         │  └─────┬──────┘  │         │
│  │  └──────┬─────┘  │         │        │         │         │
│  │         │        │         │        ▼         │         │
│  │  ┌──────▼─────┐  │         │  ┌────────────┐  │         │
│  │  │PreStop Hook│  │         │  │ GCS Upload │  │         │
│  │  │(Cleanup)   │  │         │  └────────────┘  │         │
│  │  └──────┬─────┘  │         │                  │         │
│  └─────────┼────────┘         └─────────────────┘         │
│            │                                                │
│            ▼                                                │
│  ┌─────────────────────────────────────┐                   │
│  │ HostPath: /tmp/jfr/{POD_NAME}/      │                   │
│  │  - jfr_*.jfr (profiles)             │                   │
│  │  - gc_*.log* (GC logs)              │◄──────────────────┤
│  └─────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
                ┌──────────────────────┐
                │ Google Cloud Storage │
                │  gs://bucket/pod/    │
                └──────────────────────┘
```

### Data Flow

1. **Trigger**: User runs `jfr-cli create` which executes `jcmd` in the pod via `kubectl exec`
2. **Profile**: Java JVM generates JFR file in `/tmp/jfr/{POD_NAME}/`
3. **Scan**: Go DaemonSet detects new `.jfr` or `.log` files via fsnotify
4. **Upload**: Files are streamed to GCS at `gs://{BUCKET}/{POD_NAME}/{FILE}`
5. **Cleanup**: Local files are deleted after successful upload
6. **PreStop**: When pod terminates, preStop hook stops active JFR recordings and copies GC logs

## 📂 Repository Structure

```
.
├── java-app/                 # Java "Hello World" application
│   ├── src/                  # Java source code
│   ├── prestop.sh            # PreStop hook script (JFR cleanup + GC log copy)
│   ├── Dockerfile            # Multi-stage build (Gradle → JDK)
│   └── build.gradle          # Gradle build configuration
│
├── jfr-cli/                  # CLI tool for JFR management
│   ├── main.go               # Go CLI application
│   ├── Makefile              # Build commands
│   └── README.md             # CLI documentation
│
├── go-sidecar/               # Go DaemonSet (file uploader)
│   ├── cmd/                  # Main entry point
│   ├── internal/
│   │   ├── daemon/           # File scanner
│   │   ├── logger/           # Structured logging (logrus)
│   │   └── uploader/         # GCS upload client
│   └── Dockerfile            # Multi-stage build (Go → Alpine)
│
├── infra/                    # Kubernetes manifests
│   ├── java/
│   │   └── statefulSet.yaml  # Java app deployment (no sidecar)
│   └── go/
│       └── daemonset.yaml    # Go daemon deployment
│
└── Makefile                  # Build and deployment commands
```

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (v1.20+)
- Docker
- `kubectl` configured
- GCP project with GCS bucket (for DaemonSet mode)

### 1. Build Docker Images

```bash
# Build Java application
make build-java

# Build Go sidecar
make build-go
```

### 2. Configure GCS Bucket

Edit `infra/go/daemonset.yaml` and update the ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: profiler-config
data:
  gcs-bucket: "your-gcs-bucket-name"  # Replace with your bucket
```

### 3. Deploy to Kubernetes

```bash
# Deploy Java application with sidecar
kubectl apply -f infra/java/statefulSet.yaml

# Deploy Go DaemonSet for file uploads
kubectl apply -f infra/go/daemonset.yaml
```

### 4. Verify Deployment

```bash
# Check Java pods
kubectl get pods -l app=java-profiler

# Check DaemonSet
kubectl get pods -l app=profiler-daemon

# View logs
kubectl logs -l app=profiler-daemon -f
```

## 📡 API Usage

The Go Sidecar exposes a REST API on port `8081` for controlling JFR profiling.

### Create Profile (Auto-named)

```bash
curl -X POST http://localhost:8081/create \
  -H "Content-Type: application/json" \
  -d '{"duration": "60s"}'
```

**Response:**
```json
{
  "success": true,
  "message": "Profiling started successfully",
  "data": {
    "pid": "14",
    "name": "jfr_2026-01-10T08-30-15+11-00",
    "duration": "60s",
    "filename": "jfr_2026-01-10T08-30-15+11-00.jfr"
  }
}
```

### Create Profile (Custom Name)

```bash
curl -X POST http://localhost:8081/create \
  -H "Content-Type: application/json" \
  -d '{"duration": "30s", "name": "my-custom-profile"}'
```

### List Running JFR Sessions

```bash
curl http://localhost:8081/running
```

### Stop JFR Profile

```bash
curl -X POST http://localhost:8081/stop \
  -H "Content-Type: application/json" \
  -d '{"name": "jfr_2026-01-10T08-30-15+11-00"}'
```

### List Profile Files

```bash
curl http://localhost:8081/list
```

### Health Check

```bash
curl http://localhost:8081/health
```

## 🔄 Graceful Shutdown

The Go Sidecar implements graceful shutdown to ensure JFR recordings are properly stopped when the pod is terminated (e.g., during rollout restarts).

### How It Works

1. **Signal Handling**: The sidecar listens for `SIGTERM` and `SIGINT` signals
2. **Automatic Cleanup**: When a shutdown signal is received:
   - All running JFR recordings are identified using `jcmd JFR.check`
   - Each recording is stopped via `jcmd JFR.stop`
   - Profile data is saved to the configured output files
3. **HTTP Server Shutdown**: The API server gracefully shuts down with a 30-second timeout
4. **Kubernetes Integration**: Works with the `preStop` lifecycle hook and `terminationGracePeriodSeconds` (60s)

### Pod Lifecycle Configuration

The StatefulSet includes a `preStop` hook that delays pod termination by 5 seconds:

```yaml
lifecycle:
  preStop:
    exec:
      command:
        - sh
        - -c
        - sleep 5
```

This ensures the sidecar has time to receive and process the SIGTERM signal before being forcefully terminated.

### Testing Graceful Shutdown

Use the provided test script to verify graceful shutdown behavior:

```bash
# Run the graceful shutdown test
./infra/scripts/test-graceful-shutdown.sh
```

The test will:
1. Create a long-running JFR profile (5 minutes)
2. Verify the profile is running
3. Send SIGTERM to the sidecar process
4. Check logs to confirm JFR recordings were stopped

### Monitoring Shutdown

During pod termination, you'll see the following in the sidecar logs:

```json
{"level":"info","msg":"Shutdown signal received, stopping all JFR recordings..."}
{"level":"info","count":2,"msg":"Stopping active JFR recordings"}
{"level":"info","name":"jfr_2026-01-15T10-30-00+00-00","msg":"Successfully stopped JFR recording"}
{"level":"info","msg":"API server stopped gracefully"}
```

### Rollout Restart Example

When performing a rollout restart, JFR recordings are automatically stopped:

```bash
# Trigger a rollout restart
kubectl rollout restart statefulset/java-jfr-with-sidecar

# Watch the logs to see graceful shutdown
kubectl logs -f java-jfr-with-sidecar-0 -c go-sidecar
```

## ⚙️ Configuration

### Java Application

| Environment Variable | Description | Default | Required |
|---------------------|-------------|---------|----------|
| `POD_NAME` | Pod identifier (from DownwardAPI) | - | Yes |

### Go Sidecar (API Mode)

| Environment Variable | Description | Default | Required |
|---------------------|-------------|---------|----------|
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` | No |

### Go DaemonSet (Scanner Mode)

| Environment Variable | Description | Default | Required |
|---------------------|-------------|---------|----------|
| `GCS_BUCKET` | GCS bucket name for uploads | - | **Yes** |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` | No |
| `NODE_NAME` | Node identifier (from DownwardAPI) | - | No |

## 🔍 JFR Recording Naming Convention

- **Format**: `jfr_<RFC3339-timestamp>`
- **Example**: `jfr_2026-01-10T08-30-15+11-00`
- **Filename**: Same as recording name with `.jfr` extension
- **Note**: Colons are replaced with hyphens for filesystem compatibility

## 🛠 Development

### Local Testing (Sidecar Mode)

```bash
# Start Java application
cd java-app
./gradlew bootRun

# Start Go sidecar
cd go-sidecar
go run cmd/main.go sidecar
```

### Local Testing (Daemon Mode)

```bash
# Set required environment variable
export GCS_BUCKET=your-test-bucket
export LOG_LEVEL=debug

# Start Go daemon
cd go-sidecar
go run cmd/main.go daemon
```

### Run Tests

```bash
# Test API endpoints
make test-api

# Manual profiling test
make test-profile
```

## 📊 Monitoring

### View Logs

```bash
# Sidecar logs (API)
kubectl logs -l component=api -f

# DaemonSet logs (Scanner)
kubectl logs -l app=profiler-daemon -f

# Java application logs
kubectl logs -l app=java-profiler -c java-app -f
```

### Check GCS Bucket

The DaemonSet logs the GCS bucket name at startup:

```bash
kubectl logs -l app=profiler-daemon | grep "GCS bucket"
```

Or query the ConfigMap:

```bash
kubectl get configmap profiler-config -o jsonpath='{.data.gcs-bucket}'
```

## 🔐 Security

### GCP Authentication

The DaemonSet uses **Workload Identity** or **Service Account Keys** for GCS access:

**Option 1: Workload Identity (Recommended)**
```bash
# Bind Kubernetes SA to GCP SA
gcloud iam service-accounts add-iam-policy-binding \
  your-gsa@project.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:project.svc.id.goog[default/profiler-daemon-sa]"

# Annotate Kubernetes SA
kubectl annotate serviceaccount profiler-daemon-sa \
  iam.gke.io/gcp-service-account=your-gsa@project.iam.gserviceaccount.com
```

**Option 2: Service Account Key**
- Create a GCP service account key
- Store in Kubernetes secret
- Mount in DaemonSet pod

### Required GCS Permissions

The service account needs:
- `storage.objects.create`
- `storage.objects.delete` (optional)

## 🐛 Troubleshooting

### DaemonSet fails to start

**Error**: `GCS_BUCKET environment variable is required`

**Solution**: Ensure the ConfigMap is created and referenced in the DaemonSet.

### Files not uploading

1. Check DaemonSet logs: `kubectl logs -l app=profiler-daemon -f`
2. Verify GCS bucket exists and is accessible
3. Check Workload Identity or service account permissions
4. Ensure files exist in `/tmp/jfr/{POD_NAME}/` on the node

### JFR profiling fails

1. Verify Java process is running: `kubectl exec -it <pod> -- ps aux`
2. Check sidecar logs: `kubectl logs <pod> -c profiler-sidecar`
3. Ensure `jcmd` is available in the Java container

## 📝 License

[Add your license here]

## 🤝 Contributing

[Add contribution guidelines here]

## 📚 Additional Resources

- [Java Flight Recorder Documentation](https://docs.oracle.com/javacomponents/jmc-5-4/jfr-runtime-guide/about.htm)
- [GCS Client Library for Go](https://cloud.google.com/storage/docs/reference/libraries#client-libraries-install-go)
- [Kubernetes DownwardAPI](https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/)
