[![Deploy Hugo site to Pages](https://github.com/sudorandom/kmcd.dev/actions/workflows/hugo.yml/badge.svg)](https://github.com/sudorandom/kmcd.dev/actions/workflows/hugo.yml)

# Building my site
I'm not sure why you're reading this if you're anyone other than me. But if you are me, here's how to run the site locally. This re-builds when a file is changed:

Run server:
```
hugo server -w
```

Just build the site:
```
hugo --minify
```

To show drafts/future posts:
```
hugo server --buildDrafts --buildFuture --minify
```

### PDF Export
If you want to export a page as a PDF, this is the process:
```
cat content/posts/2023-07-31_power-plant-02/index.md | python3 strip_frontmatter.py | pandoc -o output.pdf --pdf-engine=xelatex
```
This is useful when sharing a page that isn't yet ready to publish.
