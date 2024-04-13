.PHONY: rss mastodon run

rss:
	curl -k -u kevin https://rss.local.squirrels.dk/v1/export > data/links.xml

mastodon:
	curl 'https://infosec.exchange/api/v1/accounts/lookup?acct=@sudorandom' > data/mastodon-account.json
	curl 'https://infosec.exchange/api/v1/accounts/109300069582362316/statuses' \
		| jq 'del(.[] | select(.in_reply_to_id != null))' \
		> data/mastodon.json

run:
	hugo server --buildDrafts --buildFuture --minify

build:
	hugo
	npx -y pagefind --site public
