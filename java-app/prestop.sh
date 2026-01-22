#!/bin/sh
set -e

echo "=== Starting preStop cleanup ==="

# Detect Java process PID using pgrep
JAVA_PID=$(pgrep -f "java.*app.jar" || echo "")

if [ -z "$JAVA_PID" ]; then
  echo "ERROR: Could not find Java process"
  exit 1
fi

echo "Found Java process with PID: $JAVA_PID"

# Stop all JFR recordings
echo "Checking for active JFR recordings..."
jcmd "$JAVA_PID" JFR.check > /tmp/jfr_check.txt 2>&1 || true

# Parse JFR.check output to find recording IDs
# Format: Recording <id>: name=<name> (running)
RECORDING_IDS=$(grep -oP 'Recording \K\d+(?=:)' /tmp/jfr_check.txt 2>/dev/null || true)

if [ -n "$RECORDING_IDS" ]; then
  echo "Found active JFR recordings, stopping them..."
  for RECORDING_ID in $RECORDING_IDS; do
    echo "Stopping JFR recording ID: $RECORDING_ID"
    jcmd "$JAVA_PID" JFR.stop "recording=$RECORDING_ID" || echo "Warning: Failed to stop recording $RECORDING_ID"
  done
  echo "All JFR recordings stopped"
else
  echo "No active JFR recordings found"
fi

# Copy GC logs from /opt/gc to /tmp/jfr
echo "Copying GC logs..."
GC_LOG_DIR="/opt/gc"
DEST_DIR="/tmp/jfr"

if [ ! -d "$GC_LOG_DIR" ]; then
  echo "GC log directory $GC_LOG_DIR does not exist, skipping GC log copy"
else
  # Find and copy all gc.log* files
  GC_LOG_COUNT=0
  for gc_file in "$GC_LOG_DIR"/gc_*.log*; do
    if [ -f "$gc_file" ]; then
      filename=$(basename "$gc_file")
      dest_path="$DEST_DIR/$filename"

      echo "Copying $gc_file to $dest_path"
      cp "$gc_file" "$dest_path" || echo "Warning: Failed to copy $gc_file"

      # Set permissions
      chmod 644 "$dest_path" 2>/dev/null || true

      GC_LOG_COUNT=$((GC_LOG_COUNT + 1))
    fi
  done

  if [ "$GC_LOG_COUNT" -eq 0 ]; then
    echo "No GC log files found in $GC_LOG_DIR"
  else
    echo "Successfully copied $GC_LOG_COUNT GC log file(s)"
  fi
fi

echo "=== preStop cleanup completed ==="
