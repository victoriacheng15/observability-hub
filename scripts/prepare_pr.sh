#!/bin/bash

# Utility to extract PR body from commit.md and create/update a GitHub PR
COMMIT_FILE="commit.md"

if [ ! -f "$COMMIT_FILE" ]; then
    echo "Error: $COMMIT_FILE not found."
    exit 1
fi

# Create a temporary file for the PR body and ensure it is cleaned up on exit
BODY_FILE=$(mktemp)
trap 'rm -f "$BODY_FILE"' EXIT

# Extract content from '### Summary' to just before '## Execution Commands'
awk '/### Summary/,/## Execution Commands/ { if (!/## Execution Commands/) print }' "$COMMIT_FILE" > "$BODY_FILE"

# Attempt to extract the commit message for the PR title
TITLE=$(grep "git commit -m" "$COMMIT_FILE" | sed 's/.*-m "\(.*\)".*/\1/')

DRY_RUN=false
if [[ "$1" == "--dry-run" ]]; then
    DRY_RUN=true
fi

if [ "$DRY_RUN" = true ]; then
    echo "---------------------------------------------------"
    echo "📄 Extracted PR Body (Dry Run):"
    echo "---------------------------------------------------"
    cat "$BODY_FILE"
    echo "---------------------------------------------------"
    echo "🔍 DRY RUN: Command that would be executed:"
    if gh pr view >/dev/null 2>&1; then
        echo "   gh pr edit --body-file <temp_file>"
    else
        echo "   gh pr create --title \"$TITLE\" --body-file <temp_file>"
    fi
    echo "---------------------------------------------------"
else
    # Check if a PR already exists for the current branch
    if gh pr view >/dev/null 2>&1; then
        echo "🔄 Existing PR found. Updating body content..."
        gh pr edit --body-file "$BODY_FILE"
    else
        echo "🚀 No existing PR found. Executing PR creation..."
        gh pr create --title "$TITLE" --body-file "$BODY_FILE"
    fi
fi
