[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

mod? basic 'basic/.justfile'
mod? basic_image 'basic_image/.justfile'
mod? openai_and_gemini 'openai_and_gemini/.justfile'
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
mod? custom_config 'custom_config/.justfile'
mod? temporal_demo 'temporal_demo/.justfile'

