# Clean Git State Instructions

Follow these steps to clean up the repository and create tidy commits:

## Step 1: Reset any staged changes

```bash
git reset
```

## Step 2: Add and commit mkprog-mkprog tool

```bash
git add tools/mkprog-mkprog/
git commit -m "feat: Add mkprog-mkprog tool for generating system prompts

This tool generates system prompts for mkprog based on user descriptions.
It enables more specialized program generation by creating custom system
prompts tailored to specific types of programs or use cases."
```

## Step 3: Add and commit mkprog-quine tool

```bash
git add tools/mkprog-quine/
git commit -m "feat: Add mkprog-quine AI quine tool for self-reproduction

This 'AI quine' tool generates the source code of the original mkprog program.
It demonstrates self-reference capabilities of the system and provides a way
to bootstrap the entire mkprog ecosystem from a single tool."
```

## Step 4: Commit tool discovery functionality

```bash
git add main.go
git commit -m "feat: Add tool discovery capability to mkprog

Add functionality to discover other tools in the mkprog ecosystem.
Implements discoverTools() function and -list-tools flag to help users
find and utilize related tools in the ecosystem."
```

## Step 5: Commit metadata system

```bash
git add main.go
git commit -m "feat: Add program metadata storage via .mkprog.json

Add persistent metadata storage for all generated programs in .mkprog.json files.
This includes program name, description, generation details, and available tools.
The feature can be enabled/disabled with the -embed-metadata flag."
```

## Step 6: Commit self-introspection enhancements

```bash
git add main.go system-prompt.txt
git commit -m "feat: Enable self-introspection in generated programs

Update system prompt and code generation to make programs self-aware.
Generated programs now include flags to show their source code, 
display the prompt used to generate them, and list available tools."
```

## Step 7: Commit documentation updates

```bash
git add README.md
git commit -m "docs: Update documentation with new self-aware features

Document the new tool discovery capability, metadata storage, and
self-introspection flags. Add examples showing how to use these features
to help users understand and leverage the enhanced functionality."
```

## Step 8: Commit Go 1.24 tools system integration

```bash
git add tools.go sdk/ templates/
git commit -m "feat: Add Go 1.24 tools system integration

Integrate with Go 1.24 tools system for better developer experience.
Add SDK package for introspection features and templates for generated code.
This improves code reuse and maintainability across the ecosystem."
```

## Step 9: Clean up temporary and build files

```bash
# Remove build artifacts
find . -type f -name "*.exe" -delete
find . -type f -name "*.test" -delete
find . -type f -name "*.out" -delete

# Remove any temporary files
find . -type f -name "*~" -delete
find . -type f -name "*.bak" -delete
find . -type f -name "*.tmp" -delete

# Remove execution scripts 
git rm clean-commits.sh cleanup.sh CLEAN_GIT_STATE.md
git commit -m "chore: Remove temporary build and script files"
```

## Step 10: Verify clean state

```bash
git status
```

Follow these steps in order to create clean, well-organized commits that represent each logical change separately.