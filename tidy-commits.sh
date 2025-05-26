#!/bin/bash
# Script to create tidy commits and clean up git state

# Exit on any error
set -e

# Function to print section headers
header() {
  echo
  echo "==== $1 ===="
  echo
}

# Reset any existing staged changes
header "Resetting staged changes"
git reset

# Add and commit the two new tools first
header "Committing mkprog-mkprog tool"
git add tools/mkprog-mkprog/
git commit -m "feat: Add mkprog-mkprog tool for generating system prompts

This tool generates system prompts for mkprog based on user descriptions.
It enables more specialized program generation by creating custom system
prompts tailored to specific types of programs or use cases."

header "Committing mkprog-quine tool"
git add tools/mkprog-quine/
git commit -m "feat: Add mkprog-quine AI quine tool for self-reproduction

This 'AI quine' tool generates the source code of the original mkprog program.
It demonstrates self-reference capabilities of the system and provides a way
to bootstrap the entire mkprog ecosystem from a single tool."

# Now commit the main.go changes in logical chunks
header "Committing tool discovery functionality"
git add -p main.go # This will prompt you to select chunks about tool discovery
# Accept chunks related to discoverTools function and -list-tools flag
git commit -m "feat: Add tool discovery capability to mkprog

Add functionality to discover other tools in the mkprog ecosystem.
Implements discoverTools() function and -list-tools flag to help users
find and utilize related tools in the ecosystem."

header "Committing metadata system"
git add -p main.go # This will prompt you to select chunks about metadata
# Accept chunks related to ProgramMetadata struct and -embed-metadata flag
git commit -m "feat: Add program metadata storage via .mkprog.json

Add persistent metadata storage for all generated programs in .mkprog.json files.
This includes program name, description, generation details, and available tools.
The feature can be enabled/disabled with the -embed-metadata flag."

header "Committing self-introspection enhancements"
git add -p main.go system-prompt.txt # This will prompt you to select chunks about self-introspection
# Accept chunks related to enhancedSystemPrompt and self-introspection flags
git commit -m "feat: Enable self-introspection in generated programs

Update system prompt and code generation to make programs self-aware.
Generated programs now include flags to show their source code, 
display the prompt used to generate them, and list available tools."

header "Committing documentation updates"
git add README.md
git commit -m "docs: Update documentation with new self-aware features

Document the new tool discovery capability, metadata storage, and
self-introspection flags. Add examples showing how to use these features
to help users understand and leverage the enhanced functionality."

# Add Go 1.24 tools system integration if created
if [ -f "tools.go" ] || [ -d "sdk" ] || [ -d "templates" ]; then
  header "Committing Go 1.24 tools system integration"
  git add tools.go sdk/ templates/ 2>/dev/null || true
  git commit -m "feat: Add Go 1.24 tools system integration

Integrate with Go 1.24 tools system for better developer experience.
Add SDK package for introspection features and templates for generated code.
This improves code reuse and maintainability across the ecosystem."
fi

# Clean up temporary and build files
header "Cleaning up temporary files"
find . -type f -name "*.exe" -delete 2>/dev/null || true
find . -type f -name "*.test" -delete 2>/dev/null || true
find . -type f -name "*.out" -delete 2>/dev/null || true
find . -type f -name "*~" -delete 2>/dev/null || true
find . -type f -name "*.bak" -delete 2>/dev/null || true
find . -type f -name "*.tmp" -delete 2>/dev/null || true

# Remove the cleanup scripts themselves
header "Removing temporary scripts"
git rm -f clean-commits.sh cleanup.sh CLEAN_GIT_STATE.md tidy-commits.sh setup-tools-system.sh 2>/dev/null || true
git commit -m "chore: Remove temporary build and script files" || echo "No cleanup files to commit"

# Final verification
header "Verifying clean state"
git status

header "All done! Your repository now has clean, focused commits."