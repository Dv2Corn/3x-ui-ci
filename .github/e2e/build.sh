#!/bin/bash
# Builds the base (upstream main) and head panels plus the e2e helper.
set -eux
mkdir -p /tmp/bin
git worktree add -q /tmp/base "$BASE_SHA"
for d in . /tmp/base; do mkdir -p $d/internal/web/dist && touch $d/internal/web/dist/.gitkeep; done
(cd /tmp/base && go build -o /tmp/bin/x-ui-before .)
go build -o /tmp/bin/x-ui-after .
mkdir -p internal/zz_e2etool && cp .github/e2e/e2etool.go internal/zz_e2etool/main.go
go build -o /tmp/bin/e2etool ./internal/zz_e2etool
