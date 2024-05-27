#!/usr/bin/env bash

source ./env

git clone --depth=1 git@github.com:sudorandom/socialstore-me.git
pushd socialstore-me
    # update socialstore
    # go run main.go

    # if there are updates, push new change to github
    if git status -s; then
        echo "no new changes from mastodon"
    else
        git add .
        git commit -m "Update database"
        git push origin main  
    fi
popd
