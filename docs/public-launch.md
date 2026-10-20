# Public Website, Invitations, and Launch Plan

This is a planning document for a later phase. The immediate implementation is limited to server-enforced registration modes and administrator-created test accounts.

## Registration modes (implemented)

The protected web administration page controls a persisted server setting:

- **Invite-only** — default. Public self-registration is blocked until the invitation system is implemented. An administrator can create trusted tester accounts.
- **Open** — public self-registration is enabled.
- **Closed** — public self-registration is blocked.

All modes preserve login for existing users. Administrator-created accounts are allowed in every mode. The server-side API is authoritative; client-side UI changes are not security controls.

## Invitation system (planned)

- Generate cryptographically random, unique, single-use invitation keys.
- Store only a hash of each key, never the raw key.
- Expire keys using a configurable lifetime and support revocation.
- A key authorizes the invited person to download the client and create one account.
- Consume an invitation atomically only after successful account creation.
- Manage invitations through the existing `otterlink-admin` CLI and authenticated admin API; do not create a second database or separate admin website.
- Add tests for expired, revoked, invalid, reused, and concurrent redemption attempts.
- Keep the public API from bypassing invitation checks.

## Public-facing website (planned)

Host a separate static, retro-inspired OtterLink showcase at `https://amigaz.org/otterlink/`, served by Nginx on `ottertak` and logically separate from the embedded development web client. Proposed pages include overview, features, screenshots, news, invitation entry, registration, and client downloads.

The public website must not be the security boundary. Download and account-creation permissions must be enforced by the server/API. Confirm Nginx location routing, TLS, upstream access, and firewall exposure before going live; those live-host settings have not yet been audited.

## Abuse prevention and launch checklist (planned)

- Add application-level rate limits for registration attempts, invalid invitation keys, and login attempts; consider Nginx edge limits as an additional layer.
- Use generic responses for invalid/expired/revoked invitations to avoid leaking key status.
- Log administrative mode changes, invitation creation/revocation/redemption, and account creation without logging passwords, raw invite keys, or session tokens.
- Test direct API calls, not only the browser UI.
- Verify unauthenticated and regular-user sessions cannot access `/admin`, `/admin/content`, `/admin/news/publishing`, or their direct HTML/JavaScript assets; verify an admin can access them.
- Verify admin session cookies use the expected HttpOnly/SameSite settings and Secure when served over HTTPS, and are cleared on logout or when a non-admin logs in.
- Verify open, invite-only, and closed behavior after server restart.
- Review public listener/firewall configuration and Nginx routing.
- Keep existing-user login and all existing services working through policy changes.
- Before open registration, decide any additional abuse controls, moderation process, and support contact.
