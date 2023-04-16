# Building my site
I'm not sure why you're reading this if you're anyone other than me. But if you are me, here's how to run the site locally. This re-builds when a file is changed, most of the time:

```
npm run watch:site
```

Here's how to generate the output, which I check in because I had issues with GH actions doing this and I didn't spend a lot of time figuring it out:
```
npm run build:site
```