# CryptoTax Frontend (Demo)

React + TypeScript + Vite demo UI for CryptoTax backend workflows.

## Run

From repository root:

```bash
cd frontend
npm install
npm run dev
```

Production build check:

```bash
npm run build
npm run preview
```

## Environment variables

Create `frontend/.env.local` (recommended) or `frontend/.env`:

```bash
VITE_API_BASE_URL=http://localhost:8080
```

- `VITE_API_BASE_URL`: Istio ingress base URL for all frontend routes (`auth`, `ledger`, `aggregation`, `tax`).

If omitted, the app uses the same defaults shown above.

Notes:
- In Vite, only variables with `VITE_` prefix are exposed to browser code.
- `.env.local` is local-only and should not be committed.
- Restart `npm run dev` after changing env values.

## Backend dependencies

The UI depends on these services and contracts:

- `auth-svc`
  - `/auth/register`
  - `/auth/login`
  - `/auth/refresh`
  - `/auth/logout`
- `ledger-svc`
  - `/exchanges/supported`
  - `/mexc/csv`
- `aggregation-svc` (via frontend gateway)
  - `/fiat-currencies`
  - `/imports/{import_id}/transactions`
  - `/settings`
- `tax-svc` (via frontend gateway)
  - `/tax/profile`
  - `/tax/reports:start`
  - `/tax/reports`
  - `/tax/reports/{report_id}`

Source of truth for REST contracts (frontend + internal): `docs/api/openapi.yaml`.
