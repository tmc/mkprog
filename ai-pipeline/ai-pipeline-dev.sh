#!/bin/bash

LOG_FILE="ai-pipeline-dev.log"

echo "=== AI Pipeline Development Session Started at $(date) ===" >> "$LOG_FILE"

while true; do
    echo -n "> "
    read -r cmd
    echo "$(date '+%Y-%m-%d %H:%M:%S') Command: $cmd" >> "$LOG_FILE"
    eval "$cmd" 2>&1 | tee -a "$LOG_FILE"
    echo "" >> "$LOG_FILE"  # Add a blank line for readability
done
