[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

# Basic usage example
mod? basic 'basic/.justfile'

# Stick template engine example
mod? stick 'stick/.justfile'

vet:
    #!/usr/bin/env bash
    for dir in */; do
        if [ -f "$dir/go.mod" ]; then
            echo "Running go vet in $dir"
            (cd "$dir" && just vet)
        fi
    done
