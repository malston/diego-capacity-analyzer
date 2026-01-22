#!/usr/bin/env bash

cd ~/.claude/plugins/cache/claude-code-plugins/hookify/0.1.0/ || exit 1

[[ -L hookify ]] || ln -s . hookify

cd - > /dev/null || exit 1
