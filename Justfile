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

# Converts and scales a .mov file to an animated .webp file
# Usage: just mov-to-webp video.mov animated.webp
mov-to-webp input output width="500" fps="15" quality="100":
    ffmpeg -i {{input}} -vcodec libwebp -filter:v "fps={{fps}},scale={{width}}:-1" -lossless 0 -compression_level 4 -q:v {{quality}} -loop 0 {{output}}
