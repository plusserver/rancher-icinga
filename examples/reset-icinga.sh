#!/bin/sh

for t in hostgroups hosts services; do
  for hg in $(curl -s -k -u $ICINGA_USER:$ICINGA_PASSWORD $ICINGA_URL/v1/objects/${t}/ \
     | jq -r '.results[].attrs.name'); do
     curl -k -H 'Accept: application/json' -X DELETE -u $ICINGA_USER:$ICINGA_PASSWORD "$ICINGA_URL/v1/objects/${t}/${hg}?cascade=1" >/dev/null 2>&1
  done
done
