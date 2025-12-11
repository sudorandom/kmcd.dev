run:
  ./ensure-build-server.sh
  hugo server --buildDrafts --buildFuture --minify --disableFastRender

build:
  ./ensure-build-server.sh
  hugo --gc --minify --cleanDestinationDir --baseURL "https://kmcd.dev/" --destination=live
  npx -y pagefind --site live

build-future:
  ./ensure-build-server.sh
  hugo --gc --minify --cleanDestinationDir --baseURL "https://shhh.kmcd.dev/" --buildFuture  --buildDrafts --destination=future
  npx -y pagefind --site future

# Usage: just cover content/posts/2025/my-post
cover path:
  go run ./cmd/cover-art-generator \
    -o {{path}} \
    --style=random \
    --network \
    --seed=$(awk 'BEGIN{srand(); print int(rand()*9999)}') -n 100 -m $(awk 'BEGIN{srand(); print int(rand()*9)}')

cover-debug path:
  go run ./cmd/cover-art-generator \
    -o {{path}} \
    -png {{path}}/cover.png \
    --style=random \
    --network \
    --seed=$(awk 'BEGIN{srand(); print int(rand()*9999)}') -n 100 -m $(awk 'BEGIN{srand(); print int(rand()*9)}')
