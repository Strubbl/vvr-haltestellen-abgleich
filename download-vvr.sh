#!/usr/bin/env bash
set -eux
cache_dir="cache/vvr/version-20211207_234400"
search_words="Altefähr Parow Prohn Stralsund"

if [ ! -d "$cache_dir" ]
then
  mkdir -p "$cache_dir"
fi

for i in $search_words
do
  wget -O "${cache_dir}/${i}.json" "https://vvr.verbindungssuche.de/fpl/suhast.php?&query=$i"
done

