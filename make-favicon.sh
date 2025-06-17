#!/usr/bin/env bash

mogrify -background none -format ico -path static -density 600 -define icon:auto-resize=256,128,64,48,40,32,24,16 static/favicon.svg
convert -background none -format png -resize 192x192 static/favicon.svg static/android-chrome-192x192.png
convert -background none -format png -resize 512x512 static/favicon.svg static/android-chrome-512x512.png
convert -background none -format png -resize 16x16 static/favicon.svg static/favicon-16x16.png
convert -background none -format png -resize 32x32 static/favicon.svg static/favicon-32x32.png
convert -background none -format png -resize 180x180 static/favicon.svg static/apple-touch-icon.png
