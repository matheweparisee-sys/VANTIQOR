# VANTIQOR — Development-to-launch roadmap

Legend: [x] implemented in source; [~] partially implemented; [ ] pending. A checked item does not imply production verification.

## Phase 1 — Foundation
- [x] Inspect existing application and choose root project as canonical source
- [x] Signup, login, logout, session lookup implemented in development backend
- [x] Password hashing and session-cookie implementation
- [~] Signup IP capture (currently direct peer; trusted Vercel proxy handling pending)
- [~] Customer dashboard (placeholder balances/performance)
- [~] Privacy notice (development disclosure only; final legal notice pending)
- [ ] Successful clean frontend build and backend tests
- [ ] Email verification, password reset, rate limits and session controls

## Phase 2 — Customer/admin workflows
- [x] Authenticated manual deposit request and request history
- [x] Legacy admin deposit endpoints exist (shared admin API key)
- [ ] Individual admin accounts, MFA and role-based access
- [ ] Dedicated protected admin dashboard
- [ ] Immutable customer ownership of deposit requests
- [ ] Verified funding reconciliation and audit logs
- [ ] Withdrawal request, review and payout reconciliation

## Phase 3 — Investment accounting
- [ ] Double-entry immutable ledger, currency precision and idempotency
- [ ] Customer-specific portfolio holdings and verified performance
- [ ] Investment products, mandate consent and risk controls
- [ ] Statements, fees and financial reporting

## Phase 4 — Quality and compliance
- [ ] Mobile and accessibility QA across pages
- [ ] International signup and eligibility policy enforcement
- [ ] Jurisdiction-specific regulatory, KYC/AML, custody and privacy review
- [ ] Notifications, support workflows and email deliverability
- [ ] Security tests, backups, incident monitoring and recovery drills

## Phase 5 — Infrastructure and launch
- [ ] PostgreSQL schema and versioned migrations
- [ ] Supabase Auth/Postgres/Storage integration (deferred by project decision)
- [ ] GitHub CI, Vercel preview and production configuration
- [ ] Trusted-proxy IP extraction with allowlist
- [ ] End-to-end signup/deposit/admin/withdrawal tests
- [ ] Authorized real-money operational launch (only after applicable approvals)

## Current iteration
- Added signup confirm-password validation.
- Added registration privacy notice view and required acknowledgment.
- Added deposit-history refresh button.

## Known constraints
- No production database is configured; Vercel's ephemeral filesystem must not hold customer records.
- Signup IP capture behind Vercel requires trusted proxy configuration.
- The existing shared-admin-key workflow must not be treated as production-ready.
- No real customer balances or investment returns should be displayed until reconciled accounting is implemented.
