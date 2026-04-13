# CryptoTax Frontend Agent Notes

## Mission
Build a demo React UI for CryptoTax that reflects real backend workflows and contracts.
This is a desktop-first product demo, not a production web app.

## App location
- Frontend app should live in `/frontend`.
- If no React app exists yet, scaffold it in `/frontend` with React + TypeScript (prefer Vite).
- Keep this file in `/frontend/AGENTS.md` and treat it as implementation guardrails.

## Contract sources (priority order)
1. `/docs/api/frontend.openapi.yaml` (source of truth for UI-facing REST)
2. `/docs/api/internal.openapi.yaml` (for internal-only routes and gap analysis)
3. Service code when contracts are ambiguous:
   - `/services/tax-svc`
   - `/services/aggregation-svc`
   - `/Users/evgeniybukharev/Desktop/CryptoTax/auth-svc`
   - `/Users/evgeniybukharev/Desktop/CryptoTax/ledger-svc`

## Confirmed frontend endpoints
- Auth (`auth-svc`): `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/verify`
- Ledger (`ledger-svc`): `/exchanges/supported`, `/mexc/csv`
- Aggregation (`aggregation-svc`): `/fiat-currencies`, `/imports/{import_id}/transactions`, `/settings` (GET/PUT)
- Tax (`tax-svc`): `/tax/profile` (GET/PUT), `/tax/reports:start`, `/tax/reports`, `/tax/reports/{report_id}`

## Known contract gaps (do not hide these)
- No current-user endpoint (`/auth/me`-style route is absent).
- `/mexc/csv` returns only status; it does not return `import_id`.
- Aggregation range/filter endpoint (`/v1/tenants/{tenant_id}/transactions`) exists in internal API, not frontend API.

## Integration rules
- Do not invent endpoints or DTO fields.
- Use typed API clients per service/domain (`auth`, `ledger`, `aggregation`, `tax`).
- Send `Authorization: Bearer <access_token>` where required.
- Do not send client-controlled identity headers or `tenant_id` path parameters for user-facing routes; tenant context is resolved on the backend.
- Handle and render both error formats:
  - grpc-gateway: `{ code, message, details[] }`
  - rust services: `{ code, message }` or `{ error }`

## UX/style constraints
- Formal legal/financial look: white background, restrained palette, readable typography.
- Keep layout simple, clean, and table/form oriented.
- No marketing sections, gimmicks, or heavy decoration.
- Always include loading, empty, error, and success states.
- Desktop-first; ensure usable behavior on narrow widths.

## Implementation bias
- Prefer straightforward components over over-abstraction.
- Keep state/API logic explicit and debuggable.
- Add lightweight runtime validation at request boundaries where useful.
- Surface backend constraints in UI copy when contracts are incomplete.

## Done criteria
- Frontend builds and runs locally.
- Auth/session flow is functional with token refresh and logout.
- API calls match OpenAPI contracts.
- Core demo pages are usable and readable.
