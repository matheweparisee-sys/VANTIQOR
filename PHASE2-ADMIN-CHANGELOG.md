# VANTIQOR Phase 3 admin changes

- Added web/src/AdminPortal.tsx: admin key access, deposit queue, request status updates, support inbox and replies.
- /admin now renders the admin interface; / renders the investor portal.
- Fixed unrelated signup validation in deposit request submission.

## Still pending
- Individual admin identities, MFA, role permissions and audit logging.
- Real payment verification, reconciliation, ledger, withdrawals, customer management and production database.
- Supabase and Vercel deployment configuration.

## Security
The existing X-Admin-Key authentication is for local development only. Admin status approval does not credit customer funds.

## Tests
Frontend build not verified; dependency installation was unavailable in this environment.
