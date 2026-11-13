# Mainnet reserve swap milestone

This release prepares a real NEAR Intents swap and tracks its status. The user funds it in their own wallet. It does not sign transactions, custody funds, replenish a contract automatically, or protect a live lending position.

Only NEAR-to-NEAR asset conversions are enabled. Both asset IDs must come from the current token list. Cross-chain routes, destination memos, direct contract calls and signed Intents are outside this milestone.

## 1. Discover and configure

From the repository root:

```sh
make intents ARGS='tokens'
mkdir -p local
cp examples/quote.json local/request.json
```

Edit `local/request.json`. Replace all placeholders with two different NEAR-chain asset IDs, your mainnet recipient/refund account, and the integer input amount. Asset decimals determine the conversion: for a token with 6 decimals, 1 token is `1000000` units. Never use floating-point amounts. Keep enough native NEAR for gas.

Use a fresh dedicated wallet with a small balance for the first swap. The recipient must be able to receive the selected NEP-141 token; check token storage registration before funding. A NEAR account ID passing syntax validation is not proof that the account exists or has registered token storage.

The CLI fills an empty request deadline with five minutes from now. This request deadline starts refunds for an unfinished swap; the response quote deadline describes deposit-address lifetime. Fund only before the earlier deadline. The API requires an explicit RFC3339 deadline. Supported slippage is 0–100 basis points; 50 means 0.5%. Route minimum amounts and provider fees still apply; very small swaps may be rejected.

Optionally export `INTENTS_JWT` in the shell. Do not paste a wallet private key into the API or configuration. `.env` is not loaded automatically.

## 2. Preview

```sh
make intents ARGS='quote --request ../local/request.json'
```

The CLI runs inside `backend`, so file paths in `ARGS` are relative to that directory. Preview always sends `dry: true`, regardless of the file's `dry` value. A preview never starts a swap.

Review `quote.amountIn`, `quote.amountOut`, `quote.minAmountOut` and the echoed request. The client checks exact input, output bounds and recipient/refund routing. Token prices are not used as a risk oracle.

## 3. Prepare an address

After reviewing the preview, run the following with a real maximum integer amount in place of `MAX_INPUT_UNITS`:

```sh
make intents ARGS='quote --request ../local/request.json --prepare --confirm-mainnet --max-input MAX_INPUT_UNITS --out ../local/prepared-quote.json'
```

This requests `dry: false`. It still sends no funds. The entire response, including the provider signature, is saved with the request in a new file with owner-only permissions. Existing files are never overwritten. A failed preparation can leave an empty file; it is not a valid quote. Use a new filename for another attempt.

The maximum is denominated in the source asset's smallest units, not USD. It limits the requested quote, not what a user can manually send from a wallet. Prepared quote expiry, account syntax, request echo, input and output amounts are checked. Service signatures are preserved but not cryptographically verified in this release; the client trusts HTTPS to the fixed official API endpoint.

## 4. Fund from your wallet

Review the saved response and the actual wallet transaction together. Confirm the source asset, exact input amount, chain, deposit address, recipient, refund account, deadline and any deposit memo. Transfer only the quoted source asset and amount to the newly returned deposit address before it expires. For the specific `nep141:wrap.near` origin-chain route, NEAR's official 1Click example funds the quote address with a native NEAR transfer. A zero wNEAR token balance therefore does not itself block that route. This exception must not be generalized to other token IDs: USDC and other NEP-141 assets require their own token transfer path. See the official funding example linked below.

Never fund a preview, stale quote, failed/empty record or an address copied from an example. Do not repeat a transfer because status polling is slow. The swap begins when the provider receives the deposit. Save the wallet's transaction hash for reconciliation.

## 5. Track and verify

```sh
make intents ARGS='status --address DEPOSIT_ADDRESS'
# Include --memo if the quote returned a depositMemo.
```

Known statuses include `PENDING_DEPOSIT`, `KNOWN_DEPOSIT_TX`, `PROCESSING`, `SUCCESS`, `INCOMPLETE_DEPOSIT`, `REFUNDED` and `FAILED`. Preserve unknown statuses and investigate; never interpret an unrecognized state as success. `FAILED` does not prove a refund arrived. Check the destination wallet balance and on-chain transaction receipt after `SUCCESS`; check the refund wallet after `REFUNDED`.

A successful swap means the reserve asset reached the wallet. It does not mean collateral was deposited or a position rescued.

## Local HTTP API

Run `make run`. It binds only to `127.0.0.1:8080`:

- `GET /api/intents/tokens`: NEAR-chain assets supported by the live provider.
- `POST /api/intents/quote`: a validated preview request; always dry.
- `GET /api/intents/status?depositAddress=...&depositMemo=...`: provider status.
- `GET /api/demo`: simulated lending calculation, independent of the swap.
- `GET /healthz`: local process status, not an upstream availability check.

There is no HTTP prepare/fund endpoint. Before exposing the API publicly, add authentication and rate limiting at the deployment boundary. No automated retry is performed for quote creation.

## Validation and remaining work

Run `make test vet`. Tests exercise forced preview mode, explicit preparation limits, unsupported routes, malformed/mismatched quotes, expiry, response limits, HTTP errors, memo encoding and method restrictions. Mock fixtures do not establish live liquidity or settlement.

Real funding and settlement must be verified with the user's selected route before calling this milestone end-to-end tested. Automatic lending rescue remains a separate integration requiring a real protocol adapter, integer risk calculations, on-chain budget enforcement and settlement verification. The Rust policy scaffold is not a production reserve vault and should not receive user funds.

References:

- https://docs.near-intents.org/integration/distribution-channels/1click-api/quickstart/making-a-request
- https://1click.chaindefuser.com/docs/v0/openapi.yaml

- https://github.com/near-examples/near-intents-examples/blob/main/1click-example/3-send-deposit.ts
