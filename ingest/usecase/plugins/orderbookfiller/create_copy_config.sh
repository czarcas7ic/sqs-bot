#!/bin/bash

# Define the input and output file paths
ORIGINAL_APP_TOML_NAME="$HOME/.osmosisd/config/app.toml"   # Replace with the actual file path
BACKUP_APP_TOML_NAME="$HOME/.osmosisd/config/app-backup.toml"

mv $ORIGINAL_APP_TOML_NAME $BACKUP_APP_TOML_NAME

# Use sed to modify the TOML and create a new file
sed -e 's/^service-name = "osmosis-dev"/service-name = "osmosis-fill-bot"/' \
    -e 's/^grpc-ingest-address = "localhost:50051"/grpc-ingest-address = "osmosis-sqs:50051"/' \
    -e '/^\[osmosis-sqs\]/,/^is-enabled = ".*"/s/^is-enabled = ".*"/is-enabled = "true"/' \
    "$BACKUP_APP_TOML_NAME" > "$ORIGINAL_APP_TOML_NAME"

echo "Modified configuration saved to $ORIGINAL_APP_TOML_NAME, backup made in $BACKUP_APP_TOML_NAME"
