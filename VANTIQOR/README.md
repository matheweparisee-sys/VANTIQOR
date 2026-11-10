# VANTIQOR

**AI-assisted multi-asset trading workspace** for crypto, forex and US-market instruments.

VANTIQOR is a rebranded, self-hostable derivative of the upstream Clawtrade project. It retains the upstream trading engine and dashboard capabilities while adding VANTIQOR product branding and a single-container deployment path.

> **Important — this repository is not a ready-to-launch public investment business.**
> It does not provide customer onboarding/KYC, investor account segregation, regulated custody, payment processing, customer deposits/withdrawals, or a compliant pooled-funds investment service. Do not accept public deposits or enable live trading for customers based on this code alone. Use paper trading until broker permissions, security, legal, and regulatory requirements have been independently reviewed.

## Included capabilities

- React + TypeScript trading dashboard
- Go API and trading engine
- AI agent integrations (requires a configured LLM provider)
- Exchange/broker adapters included by upstream
- Portfolio, positions, charts, analytics and alerts
- Backtesting and risk controls
- Paper-trading configuration by default
- Docker deployment for a single web/API service

## Branding and upstream attribution

This repository changes the customer-facing dashboard identity to **VANTIQOR**. Internal Go package paths and the upstream CLI implementation retain some `clawtrade` identifiers to avoid risky, broad refactors of the trading engine.

Upstream project: https://github.com/Paparusi/Clawtrade

Review the included `LICENSE` before redistributing or operating a modified copy. Preserve applicable copyright and license notices.

## Run locally

Prerequisites:
- Go 1.25+
- Node.js 22+
- npm

Build the dashboard:

```bash
cd web
npm install
npm run build
cd ..
```

Build and run the Go service:

```bash
go mod download
go build -o vantiqor ./cmd/clawtrade
./vantiqor serve
```

Open `http://localhost:8899`.

The API serves the built dashboard from `web/dist`. For frontend development, run `npm run dev` inside `web`; the Vite API proxy targets the local Go API.

## Docker

```bash
docker build -t vantiqor .
docker run --rm -p 8080:8080 -e PORT=8080 -v vantiqor-data:/app/data vantiqor
```

Open `http://localhost:8080`.

Persistent storage is mounted at `/app/data`. Back up this volume securely. Do not commit database files, vault files, API keys, broker credentials, or `.env` files.

## Deploy to Render

1. Push this repository to a new GitHub repository.
2. In Render, create a new **Blueprint** and select that repository.
3. Render reads `render.yaml` and builds the Docker image.
4. Keep the persistent disk attached at `/app/data`.
5. Configure required secrets only through the host's private environment-variable settings or the application's encrypted credential vault.
6. Verify `/api/v1/system/health` and confirm the dashboard loads.

The service must use persistent storage for its SQLite database and encrypted credential vault. Do not scale this SQLite deployment horizontally without first migrating to a database and architecture designed for multi-instance operation.

## Safe initial configuration

`config/default.yaml` is configured for paper trading with automated trading disabled. Keep those settings while testing:

- `risk.default_mode: paper`
- `agent.auto_trade: false`
- `agent.confirmation: true`

Never place exchange API secrets in frontend code or `VITE_*` variables. Use restricted API keys; disable withdrawals on exchange keys. Live trading should remain disabled until thoroughly tested and legally authorized.

## US-market and regulatory note

The included adapters do not by themselves establish that a broker, instrument, trading activity, or service is available to US customers. Availability and permissions depend on the broker, account type, product, and applicable law. Operating a service that manages or advises on customer investments may trigger SEC/state investment-adviser, broker-dealer, CFTC/NFA, custody, money-transmission, AML/KYC, sanctions, and consumer-protection obligations. Obtain qualified US legal and compliance advice before handling customer assets or offering managed trading.

## Tests

```bash
go test ./...
cd web && npm run build
```

The build and tests validate software behavior only; they are not a security audit, broker approval, regulatory approval, or proof of profitability.
