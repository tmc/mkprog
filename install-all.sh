#!/bin/bash
# install-all.sh - Install all tools in the mkprog toolkit
# This script installs the main mkprog program and all tools in the tools/ directory

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

echo -e "${GREEN}Installing mkprog toolkit...${NC}"
echo

# Install main program
echo -e "${GREEN}Installing main program...${NC}"
go install
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Successfully installed main program${NC}"
    ((successful_tools++))
else
    echo -e "${RED}✗ Failed to install main program${NC}"
    ((failed_tools++))
fi
((total_tools++))
echo

# Check if tools directory exists
if [ ! -d "tools" ]; then
    echo -e "${RED}Error: tools directory not found${NC}"
    exit 1
fi

# Install all tools
echo -e "${GREEN}Installing tools...${NC}"

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
        echo -e "${YELLOW}Installing $tool_name...${NC}"
        
        # Navigate to tool directory and install
        (cd "$dir" && go install)
        
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ Successfully installed $tool_name${NC}"
            ((successful_tools++))
        else
            echo -e "${RED}✗ Failed to install $tool_name${NC}"
            ((failed_tools++))
        fi
        ((total_tools++))
        echo
    else
        echo -e "${YELLOW}Skipping $tool_name (no go.mod file)${NC}"
    fi
done

# Print summary
echo -e "${GREEN}Installation Summary:${NC}"
echo -e "Total tools: $total_tools"
echo -e "${GREEN}Successful: $successful_tools${NC}"
if [ $failed_tools -gt 0 ]; then
    echo -e "${RED}Failed: $failed_tools${NC}"
    exit 1
else
    echo -e "Failed: $failed_tools"
    echo -e "${GREEN}All tools installed successfully!${NC}"
fi

# Print usage information
echo
echo -e "${GREEN}Usage Information:${NC}"
echo -e "The following tools have been installed and should be available in your PATH:"
echo -e "  - mkprog: Main program for generating Go code"

for dir in $tool_dirs; do
    tool_name=$(basename "$dir")
    # Skip hidden directories and directories without go.mod
    if [[ $tool_name == .* ]] || [ ! -f "$dir/go.mod" ]; then
        continue
    fi
    echo -e "  - $tool_name: $(grep -m 1 -A 1 "^# $tool_name" "$dir/README.md" | tail -1 | sed 's/^[#\- ]*//')"
done

echo
echo -e "For a complete list of tools with descriptions, run:"
echo -e "  list-tools"
echo
echo -e "For help with a specific tool, run:"
echo -e "  <tool-name> -h"