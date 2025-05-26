#!/bin/bash
# This script cleans up any temporary files or work in progress

set -e  # Exit on error

# Make sure we're in the root directory
cd "$(dirname "$0")"

# Remove any backup files
find . -name "*~" -delete
find . -name "*.bak" -delete

# Remove any temporary build artifacts
find . -name "*.exe" -delete
find . -name "*.test" -delete
find . -name "*.out" -delete

# Make scripts executable
chmod +x clean-commits.sh
chmod +x cleanup.sh
chmod +x build-all.sh
chmod +x install-all.sh

# Remove any binary files in tools directories
find ./tools -type f -executable -delete

echo "Cleanup complete!"