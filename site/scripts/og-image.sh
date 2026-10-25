#!/bin/sh
# Renders the social cards in public/og/ from og-image.html with headless Chrome. Run from anywhere after editing the template: sh site/scripts/og-image.sh
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
OUT="$HERE/../public/og"
CHROME=${CHROME:-"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
PROFILE=$(mktemp -d)
trap 'rm -rf "$PROFILE"' EXIT INT TERM

mkdir -p "$OUT"
for page in home docs; do
	for lang in zh en; do
		# --virtual-time-budget lets the web fonts load before the capture; Chrome exits on its own once the screenshot is written.
		"$CHROME" --headless=new --disable-gpu --no-first-run --hide-scrollbars --force-device-scale-factor=1 \
			--user-data-dir="$PROFILE" --window-size=1200,630 --virtual-time-budget=8000 \
			--screenshot="$OUT/$page-$lang.png" "file://$HERE/og-image.html?page=$page&lang=$lang" >/dev/null 2>&1
		echo "wrote public/og/$page-$lang.png"
	done
done
