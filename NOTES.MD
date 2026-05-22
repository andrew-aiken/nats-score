## NATS JWK Auth

nsc init --dir ./nats/nsc --name admin

nsc generate config --sys-account SYS --nats-resolver-cache


### Export JWTS
```bash
nsc --keystore-dir ./nats/nsc describe operator admin --raw > nats/jwts/operator.jwt

nsc --keystore-dir ./nats/nsc describe account admin --raw > nats/jwts/account.jwt

nsc --keystore-dir ./nats/nsc describe user admin --raw > nats/jwts/admin.jwt
```



nsc add operator -n score

nsc generate nkey --operator
<!-- SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U
OABJQGO3EOW44N7IT4X32KDXM3LX3IKH6QIR4SYO2QBYF2L43RJON3HO -->

// nsc edit operator --account-jwt-server-url nats://localhost:4222 -K SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U

nsc edit operator --sk SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U
nsc edit operator --require-signing-keys


<!-- nsc add account --name SYS -->
nsc add account --name SYS -K SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U


nsc generate nkey --account
<!-- SAAFUIO7RQ4HJMDTGRHX7EHRZYREYURTXWUOVLHHG7XZMX4S37JJ7ZVGTM
ABFG45VZS6CNMFUIBDTMNZKB7ZJB2SMUCCWJ474HLK64NYSBKCVRN5OB -->

nsc edit account SYS --sk ABFG45VZS6CNMFUIBDTMNZKB7ZJB2SMUCCWJ474HLK64NYSBKCVRN5OB -K SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U





nsc add account --name score -K SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U

nsc generate nkey --account
<!-- SAAD5LLN5HXXCNJATN3EU7XMOP7W2B6T7GEFG434IU54AVYZBK6ZZUV5MU
ACBT3TI7EW7V6LO4AMSL5HWPHQONAXUX5X76P3PEXAKH2FWEVCF6VS3E -->

nsc edit account score --sk ACBT3TI7EW7V6LO4AMSL5HWPHQONAXUX5X76P3PEXAKH2FWEVCF6VS3E -K SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U

## Enable Jetstream on the account
nsc edit account score \
    --js-mem-storage 256Mb \
    --js-disk-storage 1Gb \
    --js-streams -1 \
    --js-consumer -1 \
    -K SOAKXZSOOR3MR4FY25DXGWH3RAFG7ANUVBKVOIPLG37MWHUB27TXZO5T5U

(nsc list keys --accounts --account SYS --show-seeds)
nsc add user --name admin -K SAAD5LLN5HXXCNJATN3EU7XMOP7W2B6T7GEFG434IU54AVYZBK6ZZUV5MU




# System Account ID & Resolver Key
nsc describe account SYS --json | jq -r '.sub'

# Resolver Value
nsc describe account SYS --raw

## Get user seed key
nsc list keys --users --show-seeds --account SYS

## Get Account key
nsc list keys --accounts --account SYS --show-seeds


# Setup Context
Context data written to `~/.config/nats/context/<name>.json`
Set context is stored in `~/.config/nats/context.txt`

```bash
nats context add score --description="Score Server NATS" --creds $HOME/.local/share/nats/nsc/keys/creds/score/score/admin.creds --server nats://localhost:4222
```

Edit shows all possible options
```bash
EDITOR=vim nats context edit score
```
