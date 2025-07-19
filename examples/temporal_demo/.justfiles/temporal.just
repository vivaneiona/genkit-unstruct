TEMPORAL_IP := env("TEMPORAL_IP", "127.0.0.1")

[private]
default:
    @just -f {{source_file()}} --unsorted --list --list-prefix '{{BOLD}}➤ {{NORMAL}}' --list-heading $'' | sed 's/^   //g'

# Run temporal dev server
dev-up:
    #!/bin/bash
    temporal server start-dev --dynamic-config-value frontend.enableUpdateWorkflowExecution=true --ip {{TEMPORAL_IP}} > .tmp.log 2>&1 &
    echo $! > .tmp.pid
    echo "UI: http://{{TEMPORAL_IP}}:8233"

# Stop temporal dev server
dev-down:
    @test -f .tmp.pid && kill -TERM $$(cat .tmp.pid) && rm .tmp.pid || echo "no server"
