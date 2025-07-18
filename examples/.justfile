[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

mod? openai_and_gemini 'openai_and_gemini/.justfile'

mod? basic 'basic/.justfile'
mod? stick 'stick/.justfile'
mod? complex 'complex/.justfile'
mod? plan 'plan/.justfile'
mod? vision 'vision/.justfile'
mod? stats_demo 'stats_demo/.justfile'
mod? assets 'assets/.justfile'
mod? openai 'openai/.justfile'
mod? vertexai 'vertexai/.justfile'
mod? groups 'groups/.justfile'
mod? explain 'explain/.justfile'
