# Misc constants
UOSMO = "uosmo"
USDC = 'ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4'
USDC_PRECISION = 6

UOSMO_USDC_POOL_ID =  1464

# Defaults

## Liquidity
MIN_LIQ_FILTER_DEFAULT = 5000
MAX_VAL_LOW_LIQ_FILTER_DEFAULT = 10000

## Volume
MIN_VOL_FILTER_DEFAULT = 5000
MAX_VAL_LOW_VOL_FILTER_DEFAULT = 10000
MAX_VAL_MID_VOL_FILTER_DEFAULT = 15000

## Default no. of tokens returned by helper functions
NUM_TOKENS_DEFAULT = 5

## Acceptable price differences with coingecko
HIGH_PRICE_DIFF = 0.08 ## 8%
MID_PRICE_DIFF = 0.05 ## 5%
LOW_PRICE_DIFF = 0.02 ## 2%

## Response time threshold, be more tolerant and set to 1 second
## to allow not breaking e2e tests in CI
RT_THRESHOLD = 5

## Unsupported token count threshold
UNSUPPORTED_TOKEN_COUNT_THRESHOLD = 10

# Min liquidity in USD of each token in the transmuter pool
# required to run the transmuter test. This is to avoid the flakiness
# stemming from transmuter pool imbalance.
TRANSMUTER_MIN_TOKEN_LIQ_USD = 15000
