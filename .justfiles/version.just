set unstable
# scalver.org 
# MAJOR.DATE.PATCH → MAJOR.YYYY[MM[DD]].PATCH
MAJOR := shell("cat ../MAJOR 2>/dev/null || echo 0")
DATE_FORMAT := shell("cat FORMAT 2>/dev/null || echo \"%Y%m\"")
DATE := datetime_utc(DATE_FORMAT)
PATCH := shell("cat ../PATCH 2>/dev/null || echo 0")

VERSION := MAJOR + "." + DATE + "." + PATCH

# Colors and formatting
GREEN         := shell("printf '%b' '\\033[32m'")
RED           := shell("printf '%b' '\\033[31m'")
ORANGE        := shell("printf '%b' '\\033[38;5;208m'")
PINK          := shell("printf '%b' '\\033[38;5;213m'")
GRAY          := shell("printf '%b' '\\033[38;5;244m'")
RESET         := shell("printf '%b' '\\033[0m'")
BOLD          := shell("printf '%b' '\\033[1m'")
OK            := GREEN + "✓" + RESET
NOPE          := RED + "⛌" + RESET
INFO          := GRAY + "⚙︎" + RESET

[private]
default:
    @echo "{{BOLD}}Format{{RESET}}: {{ORANGE}}MAJOR{{RESET}}.{{PINK}}YYYY[MM[DD]]{{RESET}}.{{GREEN}}PATCH{{RESET}}\n"
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

# Get latest tag name
[private]
get-latest-tag-name:
    @git describe --tags --abbrev=0 || echo "v0.0.0"

# Parse version components from a git tag (format: vMAJOR.DATE.PATCH)
[private]
parse-tag-version tag:
    #!/bin/bash
    if [[ "{{tag}}" =~ v([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
        echo "${BASH_REMATCH[1]} ${BASH_REMATCH[2]} ${BASH_REMATCH[3]}"
    else
        echo "0 0 0"
    fi

# Get the latest tag
[private]
get-latest-tag:
    @git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"

# Show the latest git tag
[group("tag")]
latest:
    @just -f {{source_file()}} get-latest-tag

# Delete all git tags locally and remotely (requires confirmation)
[confirm]
[group("tag")]
purge: 
    #!/bin/bash
    echo "{{NOPE}} WARNING: This will delete ALL tags locally and remotely!"
    read -p "Type '{{PINK}}DELETE ALL TAGS{{RESET}}' to confirm: " confirmation
    if [ "$confirmation" != "DELETE ALL TAGS" ]; then
        echo "{{INFO}} Operation cancelled"
        exit 0
    fi
    
    echo "{{INFO}} Deleting all local tags..."
    git tag -l | xargs -r git tag -d
    
    echo "{{INFO}} Deleting all remote tags..."
    git tag -l | xargs -r -I {} git push origin --delete {}
    
    echo "{{OK}} All tags deleted"

# List all git tags with details (date, commit)
[group("tag")]
list:
    @echo "{{INFO}} All git tags with details:"
    @git for-each-ref --format='%(refname:short) - %(creatordate:short) - %(subject)' refs/tags --sort=-creatordate || echo "No tags found"

# Validate version progression rules
[private]
validate-version prev_major prev_date prev_patch:
    #!/bin/bash
    # Validate: new major should be >= previous major
    if [ "{{MAJOR}}" -lt "{{prev_major}}" ]; then
        echo "{{NOPE}} New major version {{MAJOR}} is less than previous major version {{prev_major}}"
        exit 1
    fi
    
    # Validate: if previous tag date is different, patch must be 0
    if [ "{{prev_date}}" != "{{DATE}}" ] && [ "{{PATCH}}" != "0" ]; then
        echo "{{NOPE}} When date changes from {{prev_date}} to {{DATE}}, patch must be 0 but got {{PATCH}}"
        exit 1
    fi

    echo "{{INFO}} {{ORANGE}}{{prev_major}}.{{prev_date}}.{{prev_patch}}{{RESET}} → {{PINK}}{{MAJOR}}.{{DATE}}.{{PATCH}}{{RESET}}"
    

# Check if current date matches expected release date
[private]
validate-release-date:
    #!/bin/bash
    current_date=$(date +"{{DATE_FORMAT}}")
    if [ "$current_date" != "{{DATE}}" ]; then
        echo "Error: Current date $current_date does not match expected release date {{DATE}}"
        exit 1
    fi
    echo "{{INFO}} Release date validation passed"

# Check if a git tag already exists
[private]
check-tag-exists tag:
    #!/bin/bash
    if git rev-parse "{{tag}}" >/dev/null 2>&1; then
        echo "Error: Tag {{tag}} already exists"
        exit 1
    fi
    echo "{{INFO}} Tag {{tag}} is available"

# Create and push git tag and GitHub release
[private]
create-release tag:
    echo "Creating release {{tag}}..."
    git tag -a "{{tag}}" -m "Release {{tag}}"
    git push origin "{{tag}}"
    gh release create "{{tag}}" --generate-notes
    echo "{{INFO}} Release {{tag}} created successfully"


# Release a new version
[group("version")]
release:
    #!/bin/bash
    set -e 

    echo "→ Starting release process for {{PINK}}v{{VERSION}}{{RESET}}..."
    
    just -f {{source_file()}} validate-release-date
    
    LATEST_TAG=$(just -f {{source_file()}} get-latest-tag)
    echo "{{INFO}} Latest tag: ${LATEST_TAG}"
    
    VERSION_COMPONENTS=$(just -f {{source_file()}} parse-tag-version "$LATEST_TAG")
    PREV_MAJOR=$(echo $VERSION_COMPONENTS | cut -d' ' -f1)
    PREV_DATE=$(echo $VERSION_COMPONENTS | cut -d' ' -f2)
    PREV_PATCH=$(echo $VERSION_COMPONENTS | cut -d' ' -f3)
    
    just -f {{source_file()}} validate-version "$PREV_MAJOR" "$PREV_DATE" "$PREV_PATCH"
    just -f {{source_file()}} check-tag-exists "v{{VERSION}}"
    just -f {{source_file()}} create-release "v{{VERSION}}"

    echo "{{OK}} Release {{PINK}}v{{VERSION}}{{RESET}} completed successfully! 🎉"
