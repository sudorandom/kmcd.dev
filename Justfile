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

cover:
  python3 scripts/generate_cover_art.py /tmp/cover-art-raw.png --style=random --network
  go tool primitive -i /tmp/cover-art-raw.png -o cover.svg -n 100 -m $(awk 'BEGIN{srand(); print int(rand()*9)}')
