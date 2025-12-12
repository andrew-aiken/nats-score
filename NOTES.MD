alias nats="nats --user admin --password adminpass"

nats kv put settings settings '{"checks":{"noop":{"type":"noop","description":"No operation check","score_weight":1},"icmp":{"type":"icmp","description":"ICMP check","score_weight":1,"definition":{"host":"1.1.1.1"}},"http":{"type":"http","description":"HTTP Check","score_weight":2,"definition":{"host":"infrasec.sh"}}}}'

nats kv put settings 1.settings '{"icmp":{"host":"10.9.9.9","username":"foobar"}}'

nats pub events.score.check1 '{"pass":true}'
