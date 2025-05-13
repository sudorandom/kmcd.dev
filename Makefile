.PHONY: rss run

rss:
	curl -k -u kevin https://rss.local.squirrels.dk/v1/export > data/links.xml

run:
	./ensure-build-server.sh
	hugo server --buildDrafts --buildFuture --minify

build:
	./ensure-build-server.sh
	hugo --gc --minify --cleanDestinationDir --baseURL "https://kmcd.dev/" --destination=live
	npx -y pagefind --site live

build-future:
	./ensure-build-server.sh
	hugo --gc --minify --cleanDestinationDir --baseURL "https://shhh.kmcd.dev/" --buildFuture  --buildDrafts --destination=future
	npx -y pagefind --site future
