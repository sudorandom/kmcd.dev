#!/usr/bin/env bash

curl 'https://infosec.exchange/api/v1/accounts/lookup?acct=@sudorandom' > data/mastodon-account.json
ACCOUNT_ID=$(jq -r '.id' data/mastodon-account.json)
curl "https://infosec.exchange/api/v1/accounts/${ACCOUNT_ID}/statuses" \
    | jq 'del(.[] | select(.in_reply_to_id != null))' \
    > data/mastodon.json

jq -cr 'keys[] as $k | "\($k)\n\({title: ("Mastodon Post: " + (.[$k].created_at | sub("\\.[[:digit:]]+"; "") | fromdate | strftime("%Y-%m-%d@%H:%M"))), date: .[$k].created_at, devtoPublished: false, devtoSkip: true, sitemap: {disable: true}, categories: ["short-form"], permalink: .[$k].url, params: .[$k]})"' data/mastodon.json | while read -r key; do
  fname=$(jq --raw-output ".[$key].id" data/mastodon.json)
  read -r item
  echo $item | jq 'del(.params.tags)' > "./content/updates/imported/$fname.html"
  echo $item | echo "$(jq --raw-output .params.content)" >> "./content/updates/imported/$fname.html"
done

rm -rf ./data/mastodon-inform
