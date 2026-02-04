#!/bin/bash
set -e

DOCS_DIR="docs/decisions"

# - Determine Default ID
# List files, filter for those starting with digits, extract the number, sort numerically descending, take the top one.
LAST_ID=$(ls "$DOCS_DIR" | grep -E '^[0-9]+-' | cut -d'-' -f1 | sort -rn | head -n1)

if [ -z "$LAST_ID" ]; then
    NEXT_ID_NUM=1
else
    # Force base 10 arithmetic to avoid octal confusion with leading zeros
    NEXT_ID_NUM=$((10#$LAST_ID + 1))
fi

DEFAULT_ID=$(printf "%03d" "$NEXT_ID_NUM")

# - Prompt for inputs
# usage of /dev/tty allows interaction even if run via 'make' which might redirect stdin
exec < /dev/tty

echo "-------------------------------------------------"
echo "  Create New ADR (Architecture Decision Record)"
echo "-------------------------------------------------"

read -p "ADR ID [$DEFAULT_ID]: " USER_ID
ID="${USER_ID:-$DEFAULT_ID}"

DEFAULT_TITLE="New Feature"
read -p "Title [$DEFAULT_TITLE]: " USER_TITLE
TITLE="${USER_TITLE:-$DEFAULT_TITLE}"

# - Slugify Title for Filename
# Convert to lowercase, replace non-alphanumeric chars with dashes, remove leading/trailing dashes
SLUG=$(echo "$TITLE" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g' | sed -E 's/^-|-$//g')
FILENAME="${DOCS_DIR}/${ID}-${SLUG}.md"

if [ -f "$FILENAME" ]; then
    echo "Error: File $FILENAME already exists."
    exit 1
fi

DATE=$(date +%Y-%m-%d)

# - Generate File Content
cat <<EOF > "$FILENAME"
# ADR [${ID}]: ${TITLE}

- **Status:** Proposed | Accepted | Superseded
- **Date:** ${DATE}
- **Author:** Victoria Cheng

## Context and Problem Statement

What specific issue triggered this change?

## Decision Outcome

What was the chosen architectural path?

## Consequences

- **Positive:** (e.g., Faster development, resolved dependency drift).
- **Negative/Trade-offs:** (e.g., Added complexity to the CI/CD pipeline).

## Verification

- [ ] **Manual Check:** (e.g., Verified logs/UI locally).
- [ ] **Automated Tests:** (e.g., \`make nix-go-test\` passed).

EOF


echo "✅ Created: $FILENAME"
