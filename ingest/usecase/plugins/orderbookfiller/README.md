# Order Book Filler Plugin

The Order Book Filler plugin is a plugin that fills the order book orders.

It scans all active orders in the order book, comparing their tick/price to the current market.

If ask orders are below the current market, the plugin will attempt to fill them by constructing
a cyclic arbitrage route.

If bid orders are above the current market, the plugin will attempt to fill them by constructing
a cyclic arbitrage route.

All order cyclic arb are simulated and batched in a single transaction within a block.
The simulation is done to estimate the gas cost and also to avoid executing a transaction that
would be unprofitable. The batching is done to save on transaction fees but also to maximize the
possibility of dust orders being filled (i.e. orders that are too small to be profitable on their own).

## Criteria

- For an order book to be processed, it must be canonical as per SQS definition and have at least $10 USDC of liquidity
- User/bot address must have at least $10 USDC in each token to attempt to process an orderbook and a sufficient
amount to execute it, including gas fees.

## Configuration

### Node

1. Initialize a fresh node with the `osmosisd` binary.
```bash
osmosisd init fill-bot --chain-id osmosis-1
```

2. Get latest snapshot from [here](https://snapshots.osmosis.zone/index.html)

3. Go to `$HOME/.osmosisd/config/app.toml` and set `osmosis-sqs.is-enabled` to true

4. Optionally, turn off any services from `app.toml` and `config.toml` that you don't need

### SQS

In `config.json`, set the plugin to enabled:

```json
"grpc-ingester":{
    ...
    "plugins": [
        {
            "name": "orderbook",
            "enabled": true
        }
    ]
},
```

Configure the key on a test keyring, and set the following environment variables:
```bash
OSMOSIS_KEYRING_PATH=/root/.osmosisd/keyring-test
OSMOSIS_KEYRING_PASSWORD=test
OSMOSIS_KEYRING_KEY_NAME=local.info
```
- Here, the key is named `local` and the keyring path is in the default `osmosisd` home directory.

To create your key:
```bash
osmosisd keys add local --kerying-backend test --recover

# Enter your mnemonic

# Confirm the key is created
osmosisd keys list --keyring-backend test
```

Note that the test keyring is not a secure approach but we opted-in for simplicity and speed
of POC implementation. In the future, this can be improved to support multiple backends.

## Starting (via docker compose)

1. Ensure that the "Configuration" section is complete.
2. From project root, `cd` into `ingest/usecase/plugins/orderbookfiller`
3. Update `.env` with your environment variables.
4. Run `make orderbook-filler-start`
5. Run `osmosisd status` to check that the node is running and caught up to tip.
6. Curl `/healthcheck` to check that SQS is running `curl http://localhost:9092/healthcheck`
