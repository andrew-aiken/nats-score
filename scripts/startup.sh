#!/bin/bash

export NATS_ADDRESS="nats://nats:4222"
export NATS_CREDS="/server.creds"
export SCORE_CONFIG="/opt/config.json"

/go/bin/score server nats init

/go/bin/score server user add --username admin --team admin --password admin --force

/go/bin/score server user add --username team0 --team 0 --password team0 --force

/go/bin/score server user add --username team1 --team 1 --password team1 --force

/go/bin/score server user add --username observer --team observer --password observer --force

/go/bin/score server checks import --directory /checks/

/go/bin/score server nats auth --standalone > /shared/agent.creds
