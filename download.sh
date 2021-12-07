#!/usr/bin/env bash
set -eux
search_words="Stralsund Altefähr"
for i in $search_words
do
  wget -c -o "$i.json" "https://vvr.verbindungssuche.de/fpl/suhast.php?&query=$i"
done

