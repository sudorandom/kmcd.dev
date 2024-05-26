.PHONY: rss mastodon run

rss:
	curl -k -u kevin https://rss.local.squirrels.dk/v1/export > data/links.xml

run:
	hugo server --buildDrafts --buildFuture --minify

socialstore-me:
	git clone --depth=1 https://github.com/sudorandom/socialstore-me.git

mastodon:
	@mkdir -p assets/mastodon data/mastodon
	cp -r socialstore-me/media/* assets/mastodon/
	cp -r socialstore-me/statuses/* data/mastodon/

build:
	hugo --gc --minify --cleanDestinationDir --baseURL "https://kmcd.dev/"
	npx -y pagefind --site public

build-future:
	hugo --gc --minify --cleanDestinationDir --baseURL "https://shhh.kmcd.dev/" --buildFuture
	npx -y pagefind --site public
