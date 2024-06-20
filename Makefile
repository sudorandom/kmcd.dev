.PHONY: rss mastodon run update-socialstore-me

rss:
	curl -k -u kevin https://rss.local.squirrels.dk/v1/export > data/links.xml

run:
	hugo server --buildDrafts --buildFuture --minify

socialstore-me:
	git clone --depth=1 git@github.com:sudorandom/socialstore-me.git

update-socialstore-me: socialstore-me
	cd socialstore-me; git fetch origin; git reset --hard origin/main

mastodon: update-socialstore-me
	@mkdir -p assets/mastodon data/mastodon
	cp -r socialstore-me/media/* assets/mastodon/
	cp -r socialstore-me/statuses/* data/mastodon/

build:
	hugo --gc --minify --cleanDestinationDir --baseURL "https://kmcd.dev/" --destination=live
	npx -y pagefind --site live

build-future:
	hugo --gc --minify --cleanDestinationDir --baseURL "https://shhh.kmcd.dev/" --buildFuture  --buildDrafts --destination=future
	npx -y pagefind --site future
