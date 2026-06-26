# Exploration Halo 5 — Match Events / Impulses / Personnalisation

Samples live + analyses produits les 2026-06-26 (sonde JGtm, SpartanToken v4 +
API Metadata officielle). Dossier d'exploration, NON commité par défaut.

## Analyses
- `FINDINGS_events_impulses.md` — Match Events + Impulses + carnage : ce qu'on a / n'a pas,
  verdict MMR (inexistant), champs droppés récupérables.
- `FINDINGS_personnalisation.md` — Emblème/bannière, cryptum vs officiel, appearance + inventory.

## Samples bruts
| Fichier | Source | Contenu |
|---|---|---|
| `01_match_events_SAMPLE.json` | `/h5/matches/{id}/events` | 983 events (Death/Medal/Impulse/Weapon…) |
| `02_impulses_catalogue.json` | `/metadata/h5/metadata/impulses` (officiel) | 66 définitions d'impulses |
| `03_carnage_ranked_SAMPLE.json` | `/h5/arena/matches/{id}` | carnage Arena CLASSÉ (CSR réel, ratings null) |
| `04_carnage_social_SAMPLE.json` | `/h5/arena/matches/{id}` | carnage social (CTF) |
| `05_appearance_cryptum_SAMPLE.json` | `/h5/profiles/{gt}/appearance` (cryptum) | appearance RICHE (729 o) |
| `06_appearance_official_SAMPLE.json` | `/profile/h5/profiles/{gt}/appearance` (officiel) | appearance pauvre (297 o) |
| `07_inventory_cryptum_SAMPLE.json` | `/h5/profiles/{gt}/inventory` (cryptum) | cosmétiques débloqués (67 Ko) |
| `08_preferences_cryptum_SAMPLE.json` | `/h5/profiles/{gt}/preferences` (cryptum) | réglages joueur |

## Reproduire un sample
- Events/carnage : `go run ./cmd/probe-h5 JGtm` (dump dans `%TEMP%/h5_events.json`, `h5_carnage.json`).
- Metadata (impulses/medals/weapons…) : `LEVELUP_HALOAPI_KEY=<clé> go run ./cmd/h5-metadata-fetch`.
- Toolchain : `PATH` msys64 ucrt64 + `CGO_ENABLED=1` (driver DuckDB).
- Auth : SpartanToken v4 du store (`RefreshHaloTokensViaStoreFirst`), owner JGtm.
