#!/bin/bash
# This script organizes the changes into clean, focused commits

set -e  # Exit on error

# Make sure we're in the root directory
cd "$(dirname "$0")"

# First, check if we have pending changes
if [[ -z "$(git status --porcelain)" ]]; then
    echo "No changes to commit."
    exit 0
fi

# Commit 1: Add tool discovery capability
echo "Creating commit 1: Add tool discovery capability..."
git add main.go
git commit -m "feat: Add tool discovery capability to mkprog

This commit adds the ability for mkprog to discover other tools in its ecosystem.
It adds a discoverTools() function that searches common installation paths to find
related tools, and implements a -list-tools flag to display available tools.
This creates a more integrated experience and helps users discover the full toolkit."

# Commit 2: Add program metadata storage
echo "Creating commit 2: Add program metadata storage..."
git add main.go
git commit -m "feat: Add program metadata storage

This commit adds comprehensive metadata storage for all generated programs, saved
in .mkprog.json files. The metadata includes program name, description, generation
details, system prompt used, and available tools. Generated programs can access this
metadata for introspection purposes. The feature can be toggled with the -embed-metadata flag."

# Commit 3: Enhance system prompt
echo "Creating commit 3: Enhance system prompt..."
git add system-prompt.txt main.go
git commit -m "feat: Enhance system prompt for program self-awareness

This commit updates the system prompt to guide the AI in making generated programs
self-aware. The updated prompt includes instructions and example code for implementing
introspection features like showing prompts, displaying source code, and accessing
metadata. This ensures all generated programs have consistent self-examination capabilities."

# Commit 4: Add self-introspection flags
echo "Creating commit 4: Add self-introspection flags..."
git add main.go
git commit -m "feat: Add self-introspection flags to generated programs

This commit adds standardized self-introspection capabilities to all generated programs.
Programs now include flags for --show-prompt, --show-source, and --mkprog-info, allowing
them to display their generation details, source code, and other metadata. These
introspection features help users understand and share generated code more easily."

# Commit 5: Update documentation
echo "Creating commit 5: Update documentation..."
git add README.md
git commit -m "docs: Update documentation with new features

This commit updates the documentation to explain the new features: tool discovery,
metadata storage, and self-introspection capabilities. It includes examples showing
how to use the introspection flags and access program metadata. This ensures users
can fully leverage the enhanced functionality."

# Commit 6: Add mkprog-mkprog tool
echo "Creating commit 6: Add mkprog-mkprog tool..."
git add tools/mkprog-mkprog
git commit -m "feat: Add mkprog-mkprog tool for generating system prompts

This commit adds a new tool that generates system prompts for mkprog.
The tool takes a description of a program type and creates a specialized
system prompt that can be used with mkprog to generate programs with
specific functionality. This enables more customized program generation."

# Commit 7: Add mkprog-quine tool
echo "Creating commit 7: Add mkprog-quine tool..."
git add tools/mkprog-quine
git commit -m "feat: Add mkprog-quine tool for self-reproduction

This commit adds an 'AI quine' tool that can generate the source code of
the original mkprog program. This meta-tool demonstrates the self-descriptive
capabilities of the system and provides a way to bootstrap the mkprog
ecosystem from a single tool."

echo "All changes committed successfully!"
echo "Review the commits with: git log --oneline -n 7"