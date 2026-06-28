

alias score="go run cmd/cli/main.go"

score server nats init

score server checks import --directory ../checks/

score server start
