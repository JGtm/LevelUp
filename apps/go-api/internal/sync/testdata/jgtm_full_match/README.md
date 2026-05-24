# JGtm Full Match Fixture — `b71d39db-e3af-40e4-b7f9-e7c34c367981`

Match Halo Infinite complet capturé le **2026-05-24** depuis l'API live, utilisé comme fixture E2E pour `internal/sync` + `internal/persist` + `internal/analysis`.

**Owner** : JGtm (xuid `2533274823110022`)
**Mode** : Arena (LifecycleMode=3), 8 participants
**Date du match** : 2026-05-19 17:35:20 UTC
**Durée** : 532s (~9 min)

## Contenu

| Fichier | Source API | Taille | Usage test |
|---|---|---|---|
| `manifest_raw.json` | `GET /hi/films/matches/{id}/spectate` | 4.8 KB | Test parser manifest + URLs blob |
| `chunks/filmChunk0` | Blob CDN (chunk_type=1 header) | 435 KB | Test parser header film |
| `chunks/filmChunk1..28` | Blob CDN (chunk_type=2 replication × 28) | ~5.5 MB | Test parser replication chunks |
| `chunks/filmChunk29` | Blob CDN (chunk_type=3 highlight events) | 140 KB | Test `ParseHighlightEvents` |
| `api_match_stats.json` | `GET /hi/matches/{id}/stats` (Accept: JSON) | 16.6 KB | Reconstruction MatchBatch côté `engine.go` |
| `api_skill.json` | `GET /hi/matches/{id}/skill?players=xuid(X)` | 2.8 KB | Test skill enrichment côté participants |
| `api_match_history_page0.json` | `GET /hi/players/xuid(X)/matches?start=0&count=5` | 21.5 KB | Test pagination loop côté `engine.go` |

## Vérification d'intégrité

```
30 chunks téléchargés, 0 mismatch vs manifest.ChunkSize
chunk types présents : [1, 2, 3]
replication chunks (type=2) : 28
```

## Usage côté tests

```go
// internal/sync/testdata/fixtures.go
func LoadJGtmFullMatch(t *testing.T) JGtmFixture {
    return JGtmFixture{
        Manifest:      loadJSON[FilmManifestRaw](t, "jgtm_full_match/manifest_raw.json"),
        ChunksDir:     "jgtm_full_match/chunks",
        MatchStats:    loadJSON[MatchStatsAPI](t, "jgtm_full_match/api_match_stats.json"),
        Skill:         loadJSON[SkillAPI](t, "jgtm_full_match/api_skill.json"),
        History:       loadJSON[MatchHistoryAPI](t, "jgtm_full_match/api_match_history_page0.json"),
    }
}
```

## Aspects PII

- Le xuid de JGtm (`2533274823110022`) est conservé : c'est l'owner du compte connecté pour le test
- Les xuids des 7 autres participants sont publics côté API Halo et ne sont pas considérés PII strict
- Aucun token (SpartanToken, ClearanceToken, OAuth) n'est embarqué — uniquement les réponses des endpoints

## Reproduction

Pour re-télécharger ce match (tokens requis via `apps/go-api/cmd/get-token`) :

```bash
SPARTAN=$(go run ./cmd/get-token | grep SPARTAN= | cut -d= -f2-)
CLEARANCE=$(go run ./cmd/get-token | grep CLEARANCE= | cut -d= -f2-)
# Manifest
curl -H "X-343-Authorization-Spartan: ${SPARTAN}" -H "343-Clearance: ${CLEARANCE}" \
     -H "User-Agent: SHIVA-2043073184/06.10122.05904.0 (release; PC)" \
     "https://discovery-infiniteugc.svc.halowaypoint.com/hi/films/matches/{match_id}/spectate" > manifest_raw.json
# Chunks via BlobStoragePathPrefix du manifest
# Stats avec Accept: application/json
```

## Limitations connues

- Ne couvre pas Firefight (PvE) — match Arena uniquement
- Ne couvre pas FFA (Rumble) — match en équipes
- Ne couvre pas Ranked (CSR) — match Social Arena
- Pour ces edge cases, voir les autres fixtures dans `testdata/sync_fixtures/`
