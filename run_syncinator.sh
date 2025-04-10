#!/bin/bash
# filepath: run_syncinator.sh

# Check if destination argument is provided
if [ -z "$1" ]; then
    echo "Error: Destination argument required"
    echo "Usage: $0 <destination>"
    exit 1
fi

# Store the destination argument
DEST=$1

echo "Starting Syncinator with destination: $DEST"
echo "Press Ctrl+C to stop execution"

# Loop indefinitely
while true; do
    # Run the command with the provided destination
    go run cmd/SyncinatorClientExec/main.go -f config.json "$DEST" 4096
    
    # Sleep for 100ms
    sleep 0.1
done
