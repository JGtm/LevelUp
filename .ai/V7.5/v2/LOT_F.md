# LOT F — Tests, garde-rails et CI (Go + CI)

> Journal d'exécution du lot F du plan `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`.
> Worktree `LevelUp-wt-v2-tests-ci`, branche `feat/v2-tests-ci`, base `a21fd77f4`.
> Contrat : skill `plan-execution`. Statuts : `[x]` fait et vérifié · `[~]` couvert
> ailleurs (référence) · `[!]` non traité (justification).

## Environnement

Toutes les commandes `go` en série, depuis `apps/go-api`, cache dédié :

```
GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-tests-ci CGO_ENABLED=1
```

Aucun test sur film réel (aucune variable `REPLAY_FILM_DIR`, `KILLSOURCE_FIXTURES`,
`ECS_TABLE_FILM`, `DELTA_WITNESS_FILM`), jamais le tag `gamefiles`.

## Items

### [x] F.1 (G1) — Assertions de VALEUR sur le document cuit par l'ouvrier

Fichier neuf : `apps/go-api/internal/api/wire/build_queue_worker_valeurs_integration_test.go`
(le fichier existant est à 435 L, seuil 500 — un fichier séparé, mêmes tags
`integration && cgo`, appelé depuis `assertArtefactLivreEtComplet`).

**Provenance et oracle de la fixture, vérifiés sur pièces.** `internal/api/wire/testdata/
film_e2e/c0a82e88/` = le plus petit film du corpus à joueurs (Husky Raid:CTF, 8 morceaux,
~1,6 Mio zlib), versionné le 2026-08-25 (commit `6d8c5e921`). Son `fixture.json` porte DEUX
choses de natures différentes : les `chunks` (les octets du film, entrée du décodage) et les
`facts` (la feuille de match du service Halo : frags/morts/assistances par xuid, scores de
camp). Les `facts` ne traversent le décodeur QUE comme pont d'identité — les compteurs publiés
dans `ScoreTimeline.Players` sont décodés des octets (`document_score.go`). Les deux chaînes
sont donc indépendantes, et rien ne les confrontait (PLAN_MASTER §6.5).

**Mesure d'abord, gel ensuite** (exigence du rapport G7). Relevé du 2026-09-05, schéma 39 :

| Grandeur | Mesure |
|---|---|
| `frameCount` / `frameIntervalMs` / `durationMs` | 781 / 100 / 78 100 |
| `originMs` / `t0FilmMs` | 34 870 / 35 170 (300 ms = 3 frames d'écart) |
| trajectoires | 22 (20 nommées + 2 anonymes) |
| joueurs publiés dans la courbe de score | 5 sur 8 |
| courbe d'équipe | 1 seule, `teamId` absent (`unresolved`), points `195:1 485:2 706:3` |
| actions d'objectif | 12 — 8 `kills`, 4 `assists`, AUCUNE de la famille drapeau |
| tirs / loadouts / objets d'objectif / portages de drapeau | 265 / 20 / 0 / 0 |

**Différentiel film ↔ API : 15 compteurs sur 15 EXACTS.** Pour les 5 joueurs que le pont
d'identité apparie (`2533275001554469`, `2535429692041611`, `2535432531943478`,
`2535463878425995`, `2535465632069522`), les frags, morts et assistances décodés du film
égalent exactement ceux de l'API. Aucune tolérance n'est donc nommée : l'égalité est la règle.
Les 3 joueurs non appariés sont figés en liste (un 4e refus doit se voir).

**Un second fait des deux chaînes** : le roster NOMMÉ du film (7 xuids) est exactement
l'ensemble des joueurs que l'API donne à au moins une mort — le seul joueur à 0 mort
(`2535458702376288`) est le seul absent. Une vie n'est nommée que par la mort qui la ferme
(`lives.go`). En revanche le NOMBRE de vies nommées n'est PAS le nombre de morts (4 joueurs
sur 7 coïncident, 3 non, dans les deux sens) : la mesure est figée telle quelle, sans théorie.

**Anti-auto-validation** : l'oracle de l'API est RECOPIÉ À LA MAIN dans le test, et
`assertOracleFideleAuFixture` confronte la recopie au fichier à chaque exécution. Lire l'oracle
du fichier au moment du test aurait rendu l'assertion auto-validante côté oracle ; ne pas le
confronter aurait laissé une retouche du fixture déplacer l'attendu en silence.

**Preuve par mutation (3 mutations, toutes rattrapées, toutes annulées ensuite)** :

| Mutation | Effet observé |
|---|---|
| `replay/origin.go:114` `read := ... + 100` | `originMs = 34970 ms, attendu 34870 ms` ET `t0FilmMs = 35270 ms, attendu 35170 ms` |
| `replay/score_timeline.go:136` `ScoreTick{T: t, V: v + 1}` | 4 lignes `DIFFÉRENTIEL film ↔ API` (frags, morts ET assistances décalés) + `courbe d'équipe : 2 points, attendu 3 ([{194 2} {705 4}])` |
| `fixture.json` `"kills": 7` → `6` | `oracle recopié périmé pour 2535463878425995 : fixture.json dit 6/2/1 camp 0, la recopie dit 7/2/1 camp 0` |

Gate F.1 (avant-plan) :

```
GOCACHE=... CGO_ENABLED=1 go test -tags=integration -p 1 -count=1 \
  -run '^TestOuvrierReel_ConstruitEtLivre$' ./internal/api/wire/
→ ok  	levelup/go-api/internal/api/wire	9.584s
GOCACHE=... CGO_ENABLED=1 golangci-lint run --build-tags=integration \
  --new-from-merge-base=origin/main ./internal/api/wire/...
→ 0 issues.
```

## Découvertes (consignées, NON traitées — hors périmètre du lot F)

1. **Le calque des actions d'objectif ne rend plus rien de la famille drapeau sur un film
   CTF.** L'en-tête de `build_queue_worker_binary_integration_test.go` annonçait (écrit le
   2026-08-25, schéma 37) « 92 actions d'objectif nommées (famille flag) ». Mesure du
   2026-09-05 au schéma 39 sur le MÊME fixture : 12 actions, 8 `kills` + 4 `assists`, zéro
   drapeau ; `flagCarries` = 0 et `objectiveObjects` = 0 alors que la variante est
   `Husky Raid:CTF` et que l'API donne 3 captures. La ligne d'en-tête a été corrigée sur la
   mesure (doc inversée, anti-pattern 9) et la mesure figée par `assertObjectifs`, mais la
   CAUSE n'est pas traitée ici : le calque des objectifs relève des lots A et E. À vérifier :
   la garde de mode sur `Husky Raid:CTF` (`objectiveevents.ObjectiveTypeOf`,
   `replaybuild/matchfacts.go:102` → `identifiedEvents`).
