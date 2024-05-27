#!/usr/bin/env bash

make mastodon

# Normal site
make build
if diff -q -r live live.old; then
    echo "no changes detected, do nothing"
else
    echo "changes detected!"
    # npx wrangler --exclude=pagefind pages deploy --project-name=kmcd-dev ./live
fi
rm -rf live.old; cp -r live live.old

# Future site
make build-future
if diff -q -r --exclude=pagefind future future.old; then
    echo "no changes detected, do nothing"
else
    echo "changes detected!"
    # npx wrangler pages deploy --project-name=kmcd-dev-future ./future
fi
rm -rf future.old; cp -r future future.old
