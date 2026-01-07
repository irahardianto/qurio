The technical architecture is standard Go backend (ServeMux), Python worker, Vue frontend.
Configuration is hybrid: env vars + defaults.
Security policies strictly enforce "No silent failures" and "Input validation".
We are currently fixing critical security (XSS, Permissions) and reliability (Uploads) issues.
Current plan: docs/plans/2026-01-07-security-and-bug-fixes-1.md.