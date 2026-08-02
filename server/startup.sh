#!/bin/bash

go run cmd/cli/main.go server nats init

go run cmd/cli/main.go server user add --username admin --team admin --password admin --force

go run cmd/cli/main.go server user add --username team0 --team 0 --password team0 --force

go run cmd/cli/main.go server user add --username team1 --team 1 --password team1 --force

go run cmd/cli/main.go server user add --username observer --team observer --password observer --force

go run cmd/cli/main.go server nats auth --standalone > agent.creds

go run cmd/cli/main.go server checks import --directory ../checks/

go run cmd/cli/main.go server start
