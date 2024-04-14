#!/usr/bin/env bash

curl 'https://infosec.exchange/api/v1/accounts/lookup?acct=@sudorandom' > data/mastodon-account.json
ACCOUNT_ID=$(jq -r '.id' data/mastodon-account.json)
curl "https://infosec.exchange/api/v1/accounts/${ACCOUNT_ID}/statuses" \
    | jq 'del(.[] | select(.in_reply_to_id != null))' \
    > data/mastodon.json
rm -rf ./content/updates/imported
mkdir -p ./content/updates/imported

jq -cr 'keys[] as $k | "\($k)\n\({date: .[$k].created_at, permalink: .[$k].url, params: .[$k]})"' data/mastodon.json | while read -r key; do
  fname=$(jq --raw-output ".[$key].id" data/mastodon.json)
  read -r item
  printf '%s\n' "$item" > "./content/updates/imported/$fname.html"

  echo $item | echo "$(jq --raw-output .params.content)" >> "./content/updates/imported/$fname.html"
done
