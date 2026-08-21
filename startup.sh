#!/bin/bash

/go/bin/score server --config /opt/config.json nats init

/go/bin/score server --config /opt/config.json user add --username admin --team admin --password admin --force

/go/bin/score server --config /opt/config.json user add --username team0 --team 0 --password team0 --force

/go/bin/score server --config /opt/config.json user add --username team1 --team 1 --password team1 --force

/go/bin/score server --config /opt/config.json user add --username observer --team observer --password observer --force

/go/bin/score server --config /opt/config.json checks import --directory /checks/
