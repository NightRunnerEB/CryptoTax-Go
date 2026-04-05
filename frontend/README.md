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
  - `/v1/exchanges/supported`
  - `/mexc/csv`
- `aggregation-svc` (via frontend gateway)
  - `/v1/fiat-currencies`
  - `/v1/tenants/{tenant_id}/imports/{import_id}/transactions`
  - `/v1/tenants/{tenant_id}/settings`
- `tax-svc` (via frontend gateway)
  - `/v1/tenants/{tenant_id}/tax/profile`
  - `/v1/tenants/{tenant_id}/tax/reports:start`
  - `/v1/tenants/{tenant_id}/tax/reports`
  - `/v1/tenants/{tenant_id}/tax/reports/{report_id}`

Source of truth for frontend contracts: `docs/api/frontend.openapi.yaml`.
