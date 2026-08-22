#!/bin/sh

KEY_DIR=/nsc

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

cp /scripts/nats.conf.tmpl /nsc/nats.conf

SYS_JWT=$(nsc --all-dirs $KEY_DIR describe account SYS --raw | tr -d '\n')

SCORE_JWT=$(nsc --all-dirs $KEY_DIR describe account score --raw | tr -d '\n')

# Fill out the nats server configuration
sed -i "s/SYS_ACCOUNT_ID/$SYS_ID/" /nsc/nats.conf
sed -i "s/SYS_JWT/$SYS_JWT/" /nsc/nats.conf

sed -i "s/SCORE_ACCOUNT_ID/$ACCOUNT_PUBLIC_KEY/" /nsc/nats.conf
sed -i "s/SCORE_JWT/$SCORE_JWT/" /nsc/nats.conf

# Fill out the score config
cp /scripts/config.json.tmpl /nsc/config.json

sed -i "s/ACCOUNT_PUBLIC_KEY/$ACCOUNT_PUBLIC_KEY/" /nsc/config.json
sed -i "s/ACCOUNT_SIGNING_SEED/$ACCOUNT_SIGNING_KEY/" /nsc/config.json
