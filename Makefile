.PHONY: rss mastodon run

rss:
	curl -k -u kevin https://rss.local.squirrels.dk/v1/export > data/links.xml

mastodon:
	./import_mastodon.sh

run:
	hugo server --buildDrafts --buildFuture --minify

build:
	hugo --gc --minify --cleanDestinationDir --baseURL "https://kmcd.dev/"
	npx -y pagefind --site public

build-future:
	hugo --gc --minify --cleanDestinationDir --baseURL "https://shhh.kmcd.dev/" --buildFuture
	npx -y pagefind --site public
