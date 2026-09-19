#!/bin/bash

NATS_CONF="--nats-address nats://nats:4222 --nats-creds /server.creds"

/go/bin/score server $NATS_CONF nats init

/go/bin/score server $NATS_CONF user add --username admin --team admin --password admin --force

/go/bin/score server $NATS_CONF user add --username team0 --team 0 --password team0 --force

/go/bin/score server $NATS_CONF user add --username team1 --team 1 --password team1 --force

/go/bin/score server $NATS_CONF user add --username observer --team observer --password observer --force

/go/bin/score server $NATS_CONF checks import --directory /checks/

/go/bin/score server $NATS_CONF  nats auth --config /opt/config.json --standalone > /shared/agent.creds
