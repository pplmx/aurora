# Show help
default:
    @just --list

init:
    uv tool install prek
    uv tool install rumdl
    uv tool install ruff
    uv tool install rust-just
    prek install --hook-type commit-msg --hook-type pre-push
