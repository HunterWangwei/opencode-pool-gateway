# Security Policy

## Sensitive data

`data/accounts.json` may contain OpenCode API keys and website authentication cookies. Never attach this file to an issue, commit it to Git, or include it in a release archive.

If a credential is exposed:

1. Revoke or rotate the API key.
2. Sign out of OpenCode sessions and sign in again to invalidate the old cookie.
3. Remove the leaked data from every published artifact and Git history.

## Deployment

GoQuota is intended for localhost or a trusted private network. Do not expose port 8787 directly to the public internet. Use HTTPS and an additional authentication layer when remote access is required.

## Reporting

Please report security issues privately through the repository owner's GitHub contact channel. Do not publish credentials or working exploits in a public issue.
