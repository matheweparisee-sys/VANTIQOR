# VANTIQOR Phase 1 implementation — 2026-11-20

## Implemented in this build
- Root VANTIQOR-main is the canonical source; the nested copy is preserved for comparison.
- Customer signup, login, logout, and session lookup endpoints (`/api/v1/auth/*`).
- Passwords hashed with PBKDF2-SHA256, per-user random salt, 310,000 iterations.
- Session cookies are HttpOnly, SameSite Strict, and Secure on HTTPS (set `VANTIQOR_HTTPS_ONLY=true` when behind a TLS-terminating reverse proxy). Tokens are stored as SHA-256 hashes in the database, expire after 24 hours, and can be revoked at logout.
- Registration IP address recorded in `investor_users.signup_ip` from the direct TCP peer address, not user-controlled forwarding headers. **When deployed behind a proxy this will record the proxy IP, not the visitor IP.** Configure an explicit trusted-proxy policy before relying on visitor IPs. Protect this personal data, document retention and access controls in the privacy notice.
- Investor account and session tables created on API startup when a database is configured.
- New responsive customer signup/login portal, signed-in customer overview, manual deposit request form and customer deposit request history.
- Deposit requests require a session and server-side customer identity replaces any name/email in the submitted payload.
- Origin and Fetch Metadata checks for state-changing requests.
- Shared trading, portfolio, positions, balances, backtest, agent and WebSocket endpoints disabled until customer-level isolation exists. Market-data endpoints remain public.
- No automatic deposits, payments, credits or simulated investment returns were introduced.

## Remaining before production
- Trusted-proxy IP extraction with a controlled proxy allowlist and IPv4/IPv6 validation.
- Email verification, password reset, login throttling, MFA for privileged users, account recovery and session management UI.
- Admin identity/RBAC to replace shared ADMIN_API_KEY; finance approvals with separation of duties.
- Deposit request ownership via immutable user ID rather than legacy support email join; migrate old requests carefully.
- Proper customer financial ledger, reconciled funding confirmations, custody and withdrawal workflow.
- Jurisdiction eligibility checks, KYC/AML and privacy policy/retention requirements.
- Database migrations versioning, PostgreSQL deployment and backups.
- Re-enable only those trading APIs redesigned to enforce tenant isolation.
- Full integration/security test suite and successful production build.

## Test status
- Go test blocked: container Go 1.23.2, repository requires Go >= 1.25.
- npm build blocked: dependencies not cached and package registry inaccessible.
- Code formatted with gofmt. No claim of successful compilation or runtime testing.

## Important note
This is a development implementation. Do not accept customer funds or publish the app as an operational investment service until authorization, custody, security, accounting and operational controls have been validated.
