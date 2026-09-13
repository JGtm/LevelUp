# Revue adversariale bornée — lots G (fiabilité) et H (rejeu) des finitions v7.5

> 2026-09-13 — contexte frais, worktree de LECTURE `LevelUp-wt-finitions-review2`.
> Contrat `adversarial-review` : UNE ronde, constats reproductibles, aucune correction appliquée.
> **Bornée** : seuls P0 et P1 sont à corriger. Les P2 sont consignés en une ligne, sans développement.
> Diffs relus : `4acf56aae..origin/feat/finitions-fiabilite` (6 commits) et
> `4acf56aae..origin/feat/finitions-rejeu` (5 commits). Le reste de `feat/v75` est hors périmètre.

## 1. Constats

| id | sévérité | fichier:ligne | reproduction | impact | correction (une phrase) |
|---|---|---|---|---|---|
| G2-A | **P1** | `apps/go-api/internal/sync/no_art_patterns_test.go:686` (helper) vs `:269-278`, `:347-352`, `:503-508` (copies inline) | `grep -n "dansLePerimetreART" no_art_patterns_test.go` → 1 seule utilisation (`:721`) ; les trois autres scans gardent leur chaîne de `strings.Contains(path, "/migration/") \|\| …` recopiée. Modifier le périmètre (ajouter/retirer un segment exclu) ne touche qu'un scan sur quatre. | CLAUDE.md règle 6 + anti-pattern n°8 (« factorisation abandonnée ») : le helper canonique est créé et les copies ne sont pas migrées, donc les quatre scans anti-ART divergeront en silence à la première évolution du périmètre. | Remplacer les trois blocs inline par `!dansLePerimetreART(path)`. |
| G4-A | **P1** | `apps/go-api/internal/service/openspartan_post_import_title_test.go:140-141` | Injecté `ctx = ctxkeys.WithTitleSlug(ctx, opts.TitleSlug)` avant `s.recomputePerfScores(` dans `openspartan_post_import_service.go`, puis `go test ./internal/service/ -run TestRunStampeLeTitreAvantLaPremiereEtape` → **PASS, exit 0**. Cause : la seconde condition `!strings.Contains(texte, "func postImportCtx(")` est toujours FAUSSE, `postImportCtx` étant défini dans le fichier lu (`openspartan_post_import_service.go:129`). | L'assertion « stamp de titre en ligne dans Run » ne peut jamais échouer : garde-rail vert à vide (anti-pattern n°1). Le volet ORDRE du test, lui, tient et mord. | Supprimer `&& !strings.Contains(texte, "func postImportCtx(")` — le littéral recherché n'existe plus nulle part dans la source (`grep -c` = 0). |

Aucun P0. Aucun défaut de correction des données, aucune régression d'invariant ART, aucune
fuite inter-titre trouvée.

### P2 — consignés, NON traités (une ligne chacun)

- `indexcheck.MatchSkillRankAxes()` (`axes_match_skill_rank.go:24`) rend une copie SUPERFICIELLE : `KeyExprs`/`Indexes` restent partagés, alors que le godoc et `TestMatchSkillRankAxesEstUneCopieDefensive` affirment la copie défensive (aucun appelant ne mute aujourd'hui).
- `indexcheck.scanReference` (`indexcheck.go:~160`) : en mode `Sample`, `Report.Truncated` reste toujours `false` alors que l'échantillonnage a bien eu lieu (le SQL borne déjà à `MaxKeys`) — sans conséquence, la sonde ne lit pas ce champ.
- `families_parite_ts_test.go:38` (`reChaineTS = '([a-z_]+)'`) : le ratchet de parité est aveugle à un nom de famille portant un chiffre ou une majuscule ajouté côté TS seul — reproduit (`'assaut_2'` → VERT ; `'assaut'` → ROUGE).
- `skillchain/chains.go:16-19` : l'exhaustivité repose sur un corpus MANUEL (`corpusToutesBranches`) ; une 5e chaîne ajoutée à `ClassifyLUSRChain` sans entrée de corpus ni entrée dans `Chains()` laisse les deux tests verts, contrairement à ce qu'affirme l'en-tête.
- `equipment_placements.go` passe de 628 à 639 lignes : la dette de taille consignée en R11 s'accroît (CLAUDE.md règle 5, seuil 500 L).
- `no_art_patterns_test.go` atteint 823 lignes (seuil 500 L).
- D-H2 : aucun garde-rail **Go** n'apparie `usageWallPanelIDs` (`usage_summary_families.go:82`) au manifeste ; la contrainte n'existe que côté web (`placementPanels.guard.test.ts`).
- `docs/adr/0017-rebuild-art-corruption-pattern.md:4` : la ligne « Status » réécrite par G.1 conserve l'emoji `✅` (CLAUDE.md règle 4).
- `flag_spawn_neutral_label_test.go` (`TestCatalogueRecensementDesSoclesNeutres`, dernière assertion) : le contrôle `avant != 63 + 8` est tautologique compte tenu des deux assertions qui le précèdent.

### Verdict demandé sur D-H2 (P1 ou P2 ?) → **P2**

Le manifeste (`config/titles/halo_infinite/mappings/replay_labels.toml`) porte aujourd'hui
exactement deux `kind = "deployed"` (lignes 569 et 576). Un TROISIÈME panneau promu au
manifeste ferait **échouer** `apps/web/src/features/match-replay/layers/placementPanels.guard.test.ts`
(`expect(deployes).toEqual([...WALL_PANEL_IDS].sort())`) : la promotion n'est donc PAS
silencieuse, la CI rougit. Ce qui manque est seulement que ce rouge nomme la table **web** et
non la table **Go** — le développeur peut mettre à jour `WALL_PANEL_IDS` sans toucher
`usageWallPanelIDs`, et le panneau resterait alors non compté comme mur et non promu
`deployed`. Réel, mais alarmé par ailleurs et hors périmètre H (fermé) → P2.

## 2. Ce qui a été vérifié sur pièces et qui TIENT

**G.1 — suppression de `RebuildMatchSkillRankART` / `cmd/force_rebuild_art`.** Zéro référence
résiduelle dans du code Go (`git grep -l` sur `*.go` : seuls `.ai/migrations/squashed/…` — source
archivée — et `cmd/rebuild_pme_art/main.go`, qui ne fait que la mentionner dans un commentaire
corrigé). Les mentions restantes sont dans `.ai/archive/*` et le plan : historique, pas du code.
Le fichier supprimé ne déclarait AUCUN step (`git show 4acf56aae:…steps_player_rebuild_match_skill_rank.go` :
util runtime exporté uniquement), donc `order.go` est intact et la chaîne de migrations n'est pas
touchée. Les trois helpers privés qu'il consommait (`tableExists`, `loadTableColumns`, `firstWords`)
ont d'autres appelants : aucun code mort laissé. Baseline de tests : exactement les 3 lignes
package-level de `cmd/force_rebuild_art` retirées, sans champ `Test` (le contrôle de présence ne
les voyait pas), retrait justifié en tête de `scripts/check_test_baseline.sh`. ADR 0017 et les
commentaires de `cmd/server`, `steps_shared.go`, `steps_player_repairs.go` réorientés vers
`rebuild_mp` / `rebuild_pme_art`.

**G.2 — ratchet anti-ART sur nom de table interpolé.** MORDANT VÉRIFIÉ SUR LA FORME HISTORIQUE
RÉELLE : `git show 044751026^:apps/go-api/internal/ops/seed_demo_corpus.go` déposé à un chemin de
production → `TestNoInterpolatedWriteOnProtectedTables` **FAIL** (`UPDATE <table interpolee> sans
valeur liee (set-based)`), exit 1 ; témoin retiré → PASS, exit 0. Vert sur la forme actuelle
(`WHERE %s = ?`). **Aucune allowlist agrandie** : le diff du fichier est purement additif à partir
de la ligne 594. **Pas de faux positif** : balayage indépendant du dépôt (`grep` des verbes
interpolés hors test/cmd/migration/scripts) → 4 fichiers seulement ; `medal_feed_backfill.go:378`
est un `fmt.Errorf` (pas de `SET` dans le littéral), `seed_demo.go:966` et
`seed_demo_corpus.go:397` sont à valeurs liées, `restore.go:223` est la limite D-G1 déjà consignée
(vérifié : ce fichier ne nomme AUCUNE table protégée ni critique, `grep -c` = 0, donc la
corrélation file-level ne peut pas le juger — limite assumée et documentée, pas un oubli).

**G.3 — sonde `data_health_msr_index` + paquet `indexcheck`.** `OpenReadForQuery` (jamais
`OpenReadOnly`) ; alerte seule, aucune écriture, aucune réparation ; chemins par `PathResolver`
(`pr.PlayersRootDir` / `pr.PlayerDBPath`), aucun `filepath.Join(..., "data", ...)` ajouté ;
title-agnostic (boucle sur le registre + sonde de PRÉSENCE de la table, aucun `slug == "…"`) ;
`MSRIndexDesyncKeys` entre bien dans `WarningsTotal` et `MSRIndexPlayersUnmeasured` dans
`unmeasured` (`data_health_check.go:220`, `:243`) ; jauge GELÉE si non mesuré
(`publishMSRIndexGaugeIfComplete`, testé dans les deux sens) ; échantillon borné à 200 clés par
axe avec tirage réservoir sur les CLÉS (pas les lignes), mêmes paramètres que la sonde PSA, cycle
24 h ; coût mesuré sans seuil (`TestScanMSRIndexDesyncCout`) sur fixture bâtie par
`migration.RunForDB` (migrations réelles, pas de DDL recopiée). **`cmd/repair_msr_index/diag.go`
réutilise effectivement `indexcheck`** — la carte d'axes, la règle de comparaison et les types de
rapport sont supprimés du CLI et importés (`-207` lignes) ; la sémantique d'origine est préservée
(scan forcé par expression, lookup par colonnes nues, clés NULL écartées, ordre déterministe en
mode exhaustif). **Ratchet `no_local_msr_axes_test.go` MORDANT vérifié** : un fichier planté avec
`[]string{"idx_msr_playlist"}` → FAIL nommant le fichier, exit 1 ; retiré → PASS. Il porte en
plus son propre contrôle d'auto-annulation (`trouveChezLeProprietaire`).

**G.4 — stamp du titre à l'entrée du post-import.** `ctx = postImportCtx(ctx, opts.TitleSlug)` est
posé **après** `applyPostImportDefaults` (qui garantit un slug non vide, `titlePkg.DefaultSlug` en
repli) et **avant** toute étape et toute ouverture de base. Le stamp par étape a bien été retiré
(`grep -c "ctxkeys.WithTitleSlug(ctx, opts.TitleSlug)"` = 0 dans la source). Le ratchet C.1
supprimé est remplacé par : (a) `TestPostImportCtxImposeLeTitreDeLaBase` (3 cas, dont ctx vide et
sens inverse) ; (b) `TestChainePerformanceSuitLeTitreDeLaBase`, qui mesure la CONSÉQUENCE sur
`GetPerformanceChain` avec un classifier de titre SYNTHÉTIQUE enregistré puis restauré par
`t.Cleanup` (pas d'empoisonnement des tests voisins) et un contrôle positif préalable ; (c) le
volet ORDRE de `TestRunStampeLeTitreAvantLaPremiereEtape`. La garantie est strictement plus large
que C.1 (toutes les étapes au lieu du seul `recomputeLUSR`) — **sauf** l'assertion morte G4-A
ci-dessus. Le test supprimé n'était pas dans la baseline (`grep` = 0) : rien à y retirer.

**G.5 — parités.** (a) Go↔TS : `TestFamillesObjectifPariteGoTS` LIT réellement
`apps/web/src/features/match-replay/model/objectiveFamilies.ts` (chemin résolu depuis
`runtime.Caller`, `t.Fatalf` si le fichier a bougé ou si la forme change) et compare le tableau ET
le type union, ORDRE COMPRIS. **Morsure vérifiée** : une 7e famille `'assaut'` ajoutée au tableau
TS → FAIL sur les deux tests, exit 1 ; fichier restauré → PASS. Le TSDoc trompeur est corrigé dans
le même commit. (b) `infiniteLUSRChains` du gate I14 **dérive** désormais de `skillchain.Chains()`
(plus de recopie) ; `TestChainsEqualsSyncConstants` verrouille l'égalité avec les constantes
`sync.LUSRChain*` ordre compris (ce que `TestSkillChainLiterals_NoDrift` seul ne faisait pas) ;
`TestChainsEstExhaustive` contrôle les DEUX sens (toute chaîne rendue est déclarée, toute chaîne
déclarée est rendue) sur un corpus dont chaque entrée nomme la branche qu'elle exerce.

**H.1 — neutralité d'un socle.** `FlagSpawn.Neutral` est posé depuis `PointObjective.Neutral`,
lui-même dérivé du LABEL (`mapvar.Objective.IsCTFNeutral` → `objectives_catalog.go:209`), jamais
du `team_index`. `flag_neutral.go:72` trie bien sur `s.Neutral`. `flagSpawnTeam` est inchangée
(`p.Neutral` → `TeamNeutral`, sinon `p.TeamIndex`) et documentée pour ce qu'elle ne dit PAS.
**Aucun autre lecteur oublié** : `git grep TeamNeutral` hors tests rend 4 sites fonctionnels
(`flag_assign.go:239-242`, `flag_carries_handoff.go:60-65`, `flag_carries_home.go:67`,
`flag_carries_lives.go:346-348`) — tous comparent le socle à l'équipe du PORTEUR, pas à une
variante ; leur comportement est strictement inchangé, y compris sur les 8 socles à
`team_index = -1`, qui gardent `Team == -1`. Le seul comportement modifié est la composition du
panier neutre. Le recensement (`TestCatalogueRecensementDesSoclesNeutres`) lit le catalogue
VERSIONNÉ (`data/titles/halo_infinite/reference/map_objectives.json`, suivi par git) et fige
63 / 8 ; son `switch` teste `p.Neutral` AVANT `p.TeamIndex == TeamNeutral`, donc les deux comptes
sont disjoints et leur somme est exactement l'ancien panier (71).

**H.2 — pièce engendrée = déployée.** `equipmentIsSpawnedPiece` vit avec sa table
`usageWallPanelIDs`, aucune 3e copie créée. `buildEquipmentPlacements` force `OriginDeployed`
APRÈS `equipmentOrigin` et hors de son `if ok`, ce qui couvre bien le cas sans poseur
(`unknown`) ; `tallyEquipmentPlacements` est appelé APRÈS, donc la couverture reflète la
réécriture (assertion `cov.Unknown == 0` dans le test). Deux témoins NÉGATIFS sur l'appareil porté
`0x8e2dc574`, qui reste `dropped`/`deployed` selon le temps — la règle ne déborde pas. Godoc de
`EquipmentPlacement.Origin` mis à jour dans le même commit. La mesure « 10 poses basculent, toutes
des panneaux » est PLAUSIBLE et son instrument est légitime : la règle étant une réécriture pure
du champ `origin` décidée sur le seul `id`, l'appliquer aux poses déjà publiées d'un artefact rend
bien ce que le constructeur rendrait ; le test est en LECTURE SEULE, borné par `H2_ARTS`, se skippe
en CI et ne cuit rien.

**H.3 — « Live Fire, pas Aquarius ».** Le verdict TIENT, et il tient même sans la mesure.
`config/replay_corpus.toml:133-138` déclare `0797ce72` comme témoin `region_index_2_bits`,
`carte = "Live Fire"`, « seule carte du catalogue a index de region sur 2 bits » — et ce fichier
est **présent à l'identique dans la base `4acf56aae`** et **non touché par le lot H**
(`git diff … -- config/` vide) : corroboration documentaire authentiquement indépendante,
antérieure au lot. Le raisonnement sur `DetectI0Layout` est cohérent avec le code : la détection
lit un profil de bascule bit à bit et imputerait le bit d'en-tête supplémentaire à X. **La
production est effectivement indemne** : `FilmContext.I0Layout()`
(`filmdec/film_context.go:216-230`) rend `*c.impose` dès qu'un découpage est imposé et ne
consulte `DetectI0LayoutOf` qu'en repli ; `resolveI0Layout` impose le découpage du CATALOGUE dès
que `entry.Layout().Valid()`, et `replay.BuildFromFilm` passe par `NewFilmContextForMap`. Le
défaut est donc bien dans l'INSTRUMENT DE RECHERCHE, pas en production, et `[!]` avec D-H1
consigné est le bon statut.

**Transverse.** Aucun `fmt.Println` / `log.Printf` ajouté sous `internal/` ; les logs ajoutés sont
des `slog.{Warn,Error,Debug}Context(ctx, …)` structurés avec `"err", err` ; aucune erreur avalée
dans les chemins ajoutés (chaque `err` de la sonde loggue ET incrémente `ProbeErrors` /
`MSRIndexPlayersUnmeasured`). Aucun `filepath.Join(..., "data", ...)` ajouté. Aucune fonction
ajoutée > 80 L (funlen vert). Une entrée `.ai/thought_log.md` par lot, toutes deux datées
`[2026-09-13]` et statuées « Complete ». Plan entièrement statué : G.1-G.6 `[x]`, H.1/H.2/H.4
`[x]`, H.3 `[!]` avec justification écrite et découverte consignée.

## 3. Gates rejoués dans ce worktree (codes de sortie)

| gate | commande | sortie |
|---|---|---|
| build | `go build ./...` | **exit 0** |
| vet | `go vet` sur les 10 paquets touchés | **exit 0** |
| gofmt | `gofmt -l` sur les 10 paquets touchés | **aucune sortie**, exit 0 |
| tests lot G | `go test ./internal/platform/duckdb/indexcheck/ ./internal/scheduler/ ./internal/service/ ./internal/analysis/objectiveevents/ ./internal/games/halo_infinite/skillchain/ ./internal/replaybuild/ ./cmd/repair_msr_index/ -count=1` | **7 × `ok`**, exit 0 (scheduler 179,0 s · service 78,0 s · repair_msr_index 27,6 s) |
| tests lot H + ratchets | `go test ./internal/games/halo_infinite/film/replay/... ./internal/sync/ ./internal/archlint/ ./internal/migration/ -count=1` | **5 × `ok`**, exit 0 (sync 468,1 s · archlint 212,4 s · replay 26,1 s) |
| morsure G.2 (témoin réel) | fichier historique `044751026^` planté en production | **FAIL exit 1** puis PASS exit 0 après retrait |
| morsure G.3 (ratchet axes) | littéral `"idx_msr_playlist"` planté hors `indexcheck/` | **FAIL exit 1** puis PASS exit 0 après retrait |
| morsure G.5a (parité TS) | 7e famille `'assaut'` ajoutée au tableau TS | **FAIL exit 1** puis PASS exit 0 après restauration |
| non-morsure G4-A | re-stamp en ligne injecté dans `Run` | **PASS exit 0** — constat P1 |
| golangci-lint | `golangci-lint run --new-from-merge-base=4acf56aae` | **NON REJOUÉ** — `Error: parallel golangci-lint is running` (verrou global, une autre session lint en parallèle), exit 3 |

Worktree laissé **propre** (`git status --short` vide) : tous les témoins plantés ont été retirés
et tous les fichiers modifiés pour reproduction restaurés.
