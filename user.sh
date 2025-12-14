#!/bin/bash

nsc delete user dummy

nsc add user --name dummy --account score \
  --allow-sub "results.1.>,_INBOX.>" \
  --allow-pub '_INBOX.>,$JS.API.STREAM.NAMES,$JS.API.STREAM.INFO.results' \
  --allow-pub '$JS.API.CONSUMER.CREATE.results.*.results.1.>' \
  --allow-pub '$JS.API.CONSUMER.MSG.NEXT.results.*' \
  --allow-pub '$JS.API.CONSUMER.DELETE.results.*' \
  --allow-pub '$JS.ACK.results.>' \
  -K SAAD5LLN5HXXCNJATN3EU7XMOP7W2B6T7GEFG434IU54AVYZBK6ZZUV5MU
