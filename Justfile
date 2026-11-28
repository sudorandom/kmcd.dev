run:
  ./ensure-build-server.sh
  hugo server -w --buildDrafts --buildFuture --minify --noHTTPCache

build:
  ./ensure-build-server.sh
  hugo --gc --minify --cleanDestinationDir --baseURL "https://kmcd.dev/" --destination=live
  npx -y pagefind --site live

build-future:
  ./ensure-build-server.sh
  hugo --gc --minify --cleanDestinationDir --baseURL "https://shhh.kmcd.dev/" --buildFuture  --buildDrafts --destination=future
  npx -y pagefind --site future

# Usage: just new-post "My Post Title" [date]
# Examples:
#   just new-post "How DNS Works"            # schedules 1 week from today at 10:00 UTC
#   just new-post my-slug                    # title-cases slug, schedules 1 week from today
#   just new-post "How DNS Works" 2026-12-22 # schedules on 2026-12-22T10:00:00Z
#   just new-post "How DNS Works" today      # schedules for today at 10:00 UTC
new-post title_or_slug date="":
  #!/usr/bin/env python3
  import datetime, os, re, subprocess, sys

  raw_input = {{quote(title_or_slug)}}.strip()
  raw_date = {{quote(date)}}.strip()

  if not raw_input:
      sys.exit("Usage: just new-post <title-or-slug> [YYYY-MM-DD|today]")

  # Derive slug and detect whether caller passed an explicit title vs a slug
  slug = re.sub(r"[^a-z0-9]+", "-", raw_input.lower()).strip("-")
  if not slug:
      sys.exit(f"Could not derive a valid slug from: {raw_input!r}")
  has_explicit_title = raw_input != slug

  # Resolve publish date (defaults to 1 week from today at 10:00:00Z)
  now = datetime.datetime.now(datetime.timezone.utc)
  if not raw_date:
      publish_date = f"{(now + datetime.timedelta(days=7)).date().isoformat()}T10:00:00Z"
  elif raw_date.lower() in ("today", "now"):
      publish_date = f"{now.date().isoformat()}T10:00:00Z"
  elif re.fullmatch(r"\d{4}-\d{2}-\d{2}", raw_date):
      publish_date = f"{raw_date}T10:00:00Z"
  else:
      publish_date = raw_date

  year = publish_date[:4]
  rel_content_path = f"posts/{year}/{slug}/index.md"
  post_dir = f"content/posts/{year}/{slug}"
  index_path = f"{post_dir}/index.md"

  if os.path.exists(index_path):
      sys.exit(f"Post already exists: {index_path}")

  subprocess.run(
      ["hugo", "new", "content", "--kind", "posts", "--clock", publish_date, rel_content_path],
      check=True,
  )

  # Preserve exact capitalization/punctuation when caller passed a title
  if has_explicit_title:
      with open(index_path, encoding="utf-8") as f:
          text = f.read()
      escaped_title = raw_input.replace("\\", "\\\\").replace('"', '\\"')
      text = re.sub(r'^title:.*$', f'title: "{escaped_title}"', text, count=1, flags=re.M)
      with open(index_path, "w", encoding="utf-8") as f:
          f.write(text)

  subprocess.run(["just", "cover", post_dir], check=True)

  dow = datetime.date.fromisoformat(publish_date[:10]).strftime("%a")
  print(f"\nCreated:   {index_path}")
  print(f"Cover:     {post_dir}/cover.svg")
  print(f"Scheduled: {publish_date} ({dow})")
  print(f"Preview:   /posts/{slug}/")

# Usage: just cover content/posts/2025/my-post
cover path:
  go run ./cmd/cover-art-generator \
    -o {{path}} \
    --style=auto \
    --network \
    --seed=$(awk 'BEGIN{srand(); print int(rand()*999999)}') -m $(awk 'BEGIN{srand(); print int(rand()*5)}')

cover-debug path:
  go run ./cmd/cover-art-generator \
    -o {{path}} \
    --style=auto \
    --network \
    --seed=$(awk 'BEGIN{srand(); print int(rand()*999999)}') -m $(awk 'BEGIN{srand(); print int(rand()*5)}')

# Converts and scales a .mov file to an animated .webp file
# Usage: just mov-to-webp video.mov animated.webp
mov-to-webp input output width="500" fps="15" quality="100":
    ffmpeg -i {{input}} -vcodec libwebp -filter:v "fps={{fps}},scale={{width}}:-1" -lossless 0 -compression_level 4 -q:v {{quality}} -loop 0 {{output}}
