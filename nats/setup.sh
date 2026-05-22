#!/bin/bash

KEY_DIR=~/ccdc/nats-score/nats/nsc

nsc --all-dirs $KEY_DIR add operator -n score --sys --generate-signing-key

OPERATOR_SIGNING_KEY=$(nsc --all-dirs $KEY_DIR list keys --operator --json --show-seeds | jq -r '.[] | select(.signing == true).seed')

nsc --all-dirs $KEY_DIR edit operator --account-jwt-server-url nats://localhost:4222 -K ${OPERATOR_SIGNING_KEY}

SYS_ID=$(nsc --all-dirs $KEY_DIR list keys --accounts --account SYS --json | jq -r '.[] | select(.signing == false).pub')


nsc --all-dirs $KEY_DIR add account --name score -K ${OPERATOR_SIGNING_KEY}
nsc --all-dirs $KEY_DIR edit account score --sk generate -K ${OPERATOR_SIGNING_KEY}

ACCOUNT_PUBLIC_KEY=$(nsc --all-dirs $KEY_DIR list keys --accounts --account score --json --show-seeds | jq -r '.[] | select(.signing == false).pub')
ACCOUNT_SIGNING_KEY=$(nsc --all-dirs $KEY_DIR list keys --accounts --account score --json --show-seeds | jq -r '.[] | select(.signing == true).seed')

nsc edit account score \
    --all-dirs $KEY_DIR \
    --js-mem-storage 256Mb \
    --js-disk-storage 1Gb \
    --js-streams -1 \
    --js-consumer -1 \
    -K ${OPERATOR_SIGNING_KEY}


nsc --all-dirs $KEY_DIR add user --account score --name admin -K ${ACCOUNT_SIGNING_KEY}

nsc --all-dirs $KEY_DIR add user --account score --name server -K ${ACCOUNT_SIGNING_KEY}


echo System account id $SYS_ID

echo Account public key $ACCOUNT_PUBLIC_KEY
echo Account signing key $ACCOUNT_SIGNING_KEY
