[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

mod? basic 'basic/.justfile'
mod? stick 'stick/.justfile'
mod? complex 'complex/.justfile'
mod? plan 'plan/.justfile'
mod? vision 'vision/.justfile'

vet:
    #!/usr/bin/env bash
    for dir in */; do
        if [ -f "$dir/go.mod" ]; then
            echo "Running go vet in $dir"
            (cd "$dir" && just vet)
        fi
    done

some:
    just --chooser zfz 