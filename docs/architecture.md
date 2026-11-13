# Architecture and next steps

Implemented: Go CLI/API -> mainnet Intents token discovery and validated preview. Local CLI -> prepared quote -> manual wallet deposit -> status lookup.

The embedded dashboard currently exposes an empty live-position boundary. No fake position or fallback rescue path exists. A future lending adapter must provide authenticated reads and a user-authorized rescue executor.
Future execution: owner-approved policy -> funded swap -> confirmed settlement -> lending adapter -> verified health factor.

The Rust NEAR contract currently stores owner-controlled health thresholds and a pause flag. It is not connected to a lending adapter. Budget enforcement, reserves and execution permissions remain unimplemented.

## Demo semantics

`GET /api/positions` currently returns an empty response with `connected:false`. The dashboard refuses to show balances or actions until a real lending adapter is selected and configured.

## Implementation order

1. Choose a real lending protocol and implement a read-only position source with freshness checks.
2. Add wallet/account connection and explicit rescue authorization.
3. Verify the implemented Intents reserve swap with a funded user wallet and confirmed settlement.
4. Add keeper scheduling, duplicate prevention and persisted operation states.
5. Implement on-chain budgets, keeper permissions and an allowlisted lending adapter.
6. Extend the dashboard with authenticated live positions and durable per-user storage.
7. Verify rescue transactions and failure/refund behavior before enabling execution.

Cross-chain settlement is asynchronous. Funds arriving in a wallet do not automatically top up a lending position. Keep emergency reserves local and treat remote replenishment as a separate operation.

## References

- https://nearhackathon.com/HACKATHON.md
- https://docs.near.org/smart-contracts/quickstart
- https://docs.near.org/api/rpc/introduction
- https://docs.near-intents.org/

Intents is mainnet-only. This scaffold never signs or sends a blockchain transaction.
