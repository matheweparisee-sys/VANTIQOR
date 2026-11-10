# VANTIQOR rebrand notes

- Customer-facing web title and header updated to VANTIQOR.
- PWA manifest updated to VANTIQOR.
- npm package metadata updated.
- Default SQLite database renamed to `data/vantiqor.db`.
- Go API configured to serve `web/dist`.
- `PORT` environment variable supported for cloud deployment.
- Dockerfile and Render Blueprint added.
- Trading remains paper-mode by default; live execution is not enabled.

The upstream Go module/package identifiers and some CLI strings intentionally remain unchanged to reduce risk in a large trading-engine refactor.

- Deposit application form and database-backed customer support threads.
- Admin inbox for reviewing deposit applications and replying to customers (requires `ADMIN_API_KEY`).
- Manual application workflow only: no payment processor, customer balance credit, or automatic money movement.
