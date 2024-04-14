.PHONY: rss mastodon run

rss:
	curl -k -u kevin https://rss.local.squirrels.dk/v1/export > data/links.xml

mastodon:
	./import_mastodon.sh

run:
	hugo server --buildDrafts --buildFuture --minify

build:
	hugo
	npx -y pagefind --site public
