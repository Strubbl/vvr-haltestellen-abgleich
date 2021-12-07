#!/usr/bin/env bash
set -eux
cache_dir="cache/osm/version-20211207_234400"

OVERPASS_QUERY="http://overpass-api.de/api/interpreter?data=[timeout:600];area[boundary=administrative][admin_level=6][name~'(Altefähr|Parow|Prohn|Stralsund)'];(rel(area)[~'route'~'(bus|tram|train|subway|light_rail|trolleybus|ferry|monorail|aerialway|share_taxi|funicular)'];rel(br);rel[type='route'](r);)->.routes;(.routes;<<;rel(r.routes);way(r);node(w);way(r.routes);node(w);node(r.routes););out;"

if [ ! -d "$cache_dir" ]
then
  mkdir -p "$cache_dir"
fi

for i in $search_words
do
#  wget -O "${cache_dir}/${i}.json" "OVERPASS_QUERY"
done

