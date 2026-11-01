# Handoff — Publication des briques : clôture (2026-11-01)

Plan : `.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md` (archive) (lots 0 à 4, tous `[x]`, aucun `[!]`).
Relevé : `.ai/V8/RELEVE_PUBLICATION_BRIQUES_LOT0_2026-11-01.md`. Décisions : ADR 0043
(`docs/adr/0043-publishing-the-bricks.md`), ADR 0041 et 0042 amendés.

## 1. État

| Étape | État |
|---|---|
| Lots 0 à 4 (coupes, module `libs/haloapi`, README, ADR) | fait, PR #118 squashée dans `release/v8.1.0` (`e2c449b38c`) |
| `himap` requiert `gamefiles v0.1.0` | fait, PR #119 (`5c37ca56a3`) |
| Tags `libs/gamefiles/v0.1.0`, `libs/haloapi/v0.1.0`, `libs/himap/v0.1.0` | posés et poussés, `go get ...@v0.1.0` vérifié hors dépôt pour les trois |
| Worktree `../LevelUp-wt-publication`, branche, caches `-pub` | supprimés |
| Checkout principal | `release/v8.1.0` = `origin`, arbre propre |
| Passage `release/v8.1.0` → `main` | **à faire, après le chantier desktop** (push `main` = déploiement prod, prévenir) |
| Dépôts GitHub dédiés (les trois, ensemble) | **validé, à faire après le desktop**, recette ADR 0043 D-5 |

Les trois briques sont publiées : la forme d'installation est `go get github.com/JGtm/LevelUp/libs/<brique>@v0.1.0`.
Les tags pointent sur des commits de la release, pas de `main` ; le squash vers `main` créera un autre commit, le
tag garde le sien. Un `v0.1.1` sur `main` est possible plus tard si l'alignement importe.

## 2. Ordre des suites

1. Finir le chantier application de bureau (`.ai/V8/PLAN_DESKTOP_2026-11-01.md`, worktree
   `../LevelUp-wt-desktop`, branche `feat/desktop-p0`). Décision du user du 2026-11-01.
2. PR `release/v8.1.0` → `main` (squash). Prévenir avant : déploiement prod automatique.
3. Avant toute publicité des briques ou le passage en dépôts dédiés, trancher deux découvertes :
   - PUB-2.5-e : la redirection 302 des rendus Halo 5 réémet le jeton Spartan vers `image.halocdn.com`
     (préexistant, figé par le témoin T2) ; décider si le `Doer` retire l'en-tête sur changement d'hôte.
   - PUB-4-a : origine des fixtures `libs/gamefiles/{mapvar,hinavmesh}/testdata` à documenter (ADR 0041 interdit
     tout fichier extrait du jeu dans une brique).
4. Dépôts dédiés `JGtm/gamefiles`, `JGtm/himap`, `JGtm/haloapi` : `git subtree split`, chemin de module conservé,
   les trois en une fois, jamais un seul (ADR 0043 D-5 ; décision user 2026-11-01).

## 3. Dettes laissées au registre

`.ai/REGISTRE_REPORTS.md`, section « Reports du plan Publication des briques (2026-11-01) » : `assets`
(`TokenProvider` avalé, `http.DefaultClient` sans délai), GameCMS `Progression/file` en 403, neuf enregistreurs
de test hors `haloapitest`, ratchets `netguard_coverage_test` et archlint bornés à go-api, flake Windows
`mapcatalog`, `t.Skipf` de `leaderboard_scraper_test`, baseline `tests_libs_haloapi.jsonl` vérifiée en CI seulement.
Hors registre : `docs/CONFIGURATION.md:150` cite `halo.InvalidateCachedPlayerTokens` (la fonction est
`tokencache.InvalidateCachedPlayerTokens`) ; la ligne 4.2 du plan dit « tags par l'utilisateur » alors qu'ils ont été
posés par le pilote sur validation du user.

## 4. Repères pour reprendre

- Gates d'une brique : `make libs-test` ; lint = `golangci-lint run --config ../../apps/go-api/.golangci.yml ./...`
  depuis `libs/<brique>` (la config par défaut donne 3 faux errcheck).
- CGO sous PowerShell : `C:\msys64\ucrt64\bin` en tête du PATH + `CGO_ENABLED=1`, sinon le lien DuckDB échoue.
- Vérification d'un tag hors dépôt : `GOWORK=off GOPROXY=direct GONOSUMDB=github.com/JGtm/LevelUp go get <module>@vX.Y.Z`.
- Témoins : URL figées (`*_url_figees_test.go`, 19 tests), lectures live `cmd/diag_api_lecture` (11 surfaces, SHA-256),
  cinq surfaces netguard. Toute reprise d'une coupe se prouve par ces trois témoins inchangés.
