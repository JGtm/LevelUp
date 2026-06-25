# Doppler (not used)

French version: [FR/DOPPLER_SETUP.md](FR/DOPPLER_SETUP.md)

LevelUp does **not** integrate Doppler. The old Python secrets loader
(`src/utils/secrets.py`, `load_doppler_secrets_to_env()`) was removed with the
rest of the Python/Streamlit stack, and the Go backend (`apps/go-api`) has no
Doppler wiring. The `"doppler_enabled": false` key left in the demo config seed
is inert and read by nothing.

## How secrets are handled instead

- **Auth tokens** (Microsoft OAuth refresh token + MSAL cache) live in the
  `MultiUserTokenStore`, one JSON file per user — see
  [adr/0023-auth-tokens-single-source.md](adr/0023-auth-tokens-single-source.md).
- **App secrets and env vars** (OAuth client id/secret, Discord webhook, etc.)
  are read from the process environment / `.env.local`.

Full reference: [CONFIGURATION.md](CONFIGURATION.md) (Token Storage & Onboarding,
environment variables).
