#!/bin/bash
# build-all.sh - Build all tools in the mkprog toolkit
# This script builds the main mkprog program and all tools in the tools/ directory

set -e

# Color codes for output
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Count stats
total_tools=0
successful_tools=0
failed_tools=0

echo -e "${GREEN}Building mkprog toolkit...${NC}"
echo

# Build main program
echo -e "${GREEN}Building main program...${NC}"
go build
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully built main program${NC}"
    ((successful_tools++))
else
    echo -e "${RED}✗ Failed to build main program${NC}"
    ((failed_tools++))
fi
((total_tools++))
echo

# Check if tools directory exists
if [ ! -d "tools" ]; then
    echo -e "${RED}Error: tools directory not found${NC}"
    exit 1
fi

# Build all tools
echo -e "${GREEN}Building tools...${NC}"

# Get a list of all tool directories
tool_dirs=$(find tools -maxdepth 1 -mindepth 1 -type d | sort)

for dir in $tool_dirs; do
    tool_name=$(basename "$dir")
    # Skip hidden directories
    if [[ $tool_name == .* ]]; then
        continue
    fi

    # Check if the directory has a go.mod file
    if [ -f "$dir/go.mod" ]; then
        echo -e "${YELLOW}Building $tool_name...${NC}"
        
        # Navigate to tool directory and build
        (cd "$dir" && go build -v)
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ Successfully built $tool_name${NC}"
            ((successful_tools++))
        else
            echo -e "${RED}✗ Failed to build $tool_name${NC}"
            ((failed_tools++))
        fi
        ((total_tools++))
        echo
    else
        echo -e "${YELLOW}Skipping $tool_name (no go.mod file)${NC}"
    fi
done

# Print summary
echo -e "${GREEN}Build Summary:${NC}"
echo -e "Total tools: $total_tools"
echo -e "${GREEN}Successful: $successful_tools${NC}"
if [ $failed_tools -gt 0 ]; then
    echo -e "${RED}Failed: $failed_tools${NC}"
    exit 1
else
    echo -e "Failed: $failed_tools"
    echo -e "${GREEN}All tools built successfully!${NC}"
fi