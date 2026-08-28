# Protocole — re-cuisson Oddball d9781168 (schema 23, porteur du crane)

> Ecrit et commite AVANT le fix. Branche `wt/oddball-recuisson`, base 53db169ce.
> Cible : re-cuire `data/cache/replays/halo_infinite/d9781168.json` du depot principal
> (via `LEVELUP_REPO_ROOT`) avec le PORTEUR DU CRANE visible, le score a equipes resolues
> et les dots de manche.

## Diagnostic (etabli sur pieces, avant tout fix)

- `match_registry` de d9781168 : `map_name = "Dredge"`, `map_id = e4bb06db-...`,
  `game_variant_name = "Oddball:Arena"`, 8 joueurs / 2 equipes, scores 191 / 196.
- `map_quant_bounds.json` (`data/titles/halo_infinite/reference/`) PORTE la cle `"dredge"`.
  `NormalizeMapName("Dredge") == "dredge"` => `Lookup` REUSSIT.
- CAUSE de « carte hors catalogue ([]) » : la forme DIRECTE `backfill-replay --one <id>`
  (documentee comme licite a la main) n'appelle PAS `registreParShort` — la resolution de
  carte vit UNIQUEMENT dans le PARENT, qui passe les candidats a l'enfant via `--map-name`.
  Tapee a la main sans `--map-name`, `o.mapNames == []` => `ResolveMapEntry([])` =>
  `ErrMapNotInCatalog (candidats: [])`. Le `[]` du message EST la liste vide, PAS une
  resolution DB qui rend vide. Ce n'est donc PAS un defaut de donnee (« Dredge » est deja au
  registre ET au catalogue) : c'est un TROU de code dans l'auto-suffisance de l'enfant.
- CONSEQUENCE sur l'artefact servi (schema 23, issu du repli `replay-build --map Dredge`
  SANS facts) : `objectiveObjects` present (crane libre — ne depend pas des facts),
  mais `skullCarries` ABSENT et `scoreTimeline.players` ABSENT. Le porteur exige
  `facts.GameVariantName` (garde de mode `isSkullVariant`) ET le roster ; le chemin
  `replay-build --map` ne charge aucun fait. D'ou : crane invisible.

## Fix (minimal, cible, title-agnostic)

Faire que la forme `--one` resolve elle-meme les identites de carte depuis le registre quand
`--map-name` est absent, par la MEME logique que le parent (`map_id` -> asset EN + `map_name`
brut), via PathResolver. La masse (parent -> enfant) passe toujours `--map-name` : la
resolution registre ne se declenche QUE pour le `--one` tape a la main (aucun surcout de masse).
Centraliser l'ordre des candidats dans un seul helper (`candidatsCarte`) partage parent/enfant
(regle ≤2 copies).

## Etapes

1. [ ] Fix code : `candidatsCarte` partage ; `mapNamesForOne` (RO relache avant decodage) ;
       branchement dans `runBackfillReplayUn` (explicit `--map-name` gagne, sinon registre).
2. [ ] Gates : `go build ./...`, `go vet ./...`, `go test ./cmd/levelup/... ./internal/replaybuild/...`.
3. [ ] Re-cuire d9781168 par `backfill-replay --one d9781168 --force` (LEVELUP_REPO_ROOT = principal).
4. [ ] Verdict sur pieces de l'artefact ecrit : schema 23, `skullCarries` present,
       `objectiveObjects` present, `scoreTimeline.players` > 0, identite des camps `resolved`,
       `rounds[]` par equipe.
5. [ ] Investigation SANS fix du « 2 joueurs / respawn 20 s » de fin de manche.
6. [ ] Impact re-cuisson de masse : combien de films au meme cas.

## Invariants

- DBs en LECTURE SEULE (serveur tient RW) : `OpenReadForQuery` / `diag_q`, jamais RW.
- Seul l'artefact d9781168 du principal est reecrit. Pas de push. thought_log/REGISTRE = CR.
