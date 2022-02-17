#!/usr/bin/env bash

git checkout --orphan gh-pages
git --work-tree site add --all
git --work-tree site commit -m 'gh-pages'
git push origin HEAD:gh-pages --force
git checkout -f main
