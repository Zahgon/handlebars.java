#!/usr/bin/env sh
# Pre-commit hook: formats the staged Go files with gofmt and stages the result.
# Install it with: ln -s ../../src/etc/formatter.sh .git/hooks/pre-commit
stagedFiles=$(git diff --staged --diff-filter=d --name-only | grep '\.go$')
if [ -n "$stagedFiles" ]; then
  echo "formatting $(echo "$stagedFiles" | wc -l | tr -d ' ') file(s)"
  gofmt -w $stagedFiles || exit 1
  git add $stagedFiles
fi
