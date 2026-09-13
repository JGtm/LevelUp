# Plan d'orchestration — finitions Tactique, restes v2, libellés en dur, fusions — 2026-09-07

> Rédigé par le superviseur (Fable 5.1) après vérification sur pièces le 2026-09-07 au soir,
> HEAD `feat/v75` = `f2c8ddce1` (fusion du chantier Tactique). Ce plan ORDONNE quatre documents
> existants (`PLAN_TACTIQUE_SUITE`, `PLAN_V2_RESTES`, `PLAN_LIBELLES_EN_DUR_GO`,
> `DECOUVERTES_TACTIQUE`) et tranche le sort de trois branches. Il ne réécrit pas leurs
> items : chaque lot ci-dessous RENVOIE au document source, qui reste la liste cochable.
> Contrat d'exécution : skill `plan-execution`. Doctrine RE-VÉRIFIER : les `fichier:ligne`
> sont une carte datée du 07/09.

## 0. Principes (sobriété des quotas, zéro dérive)

1. **Priorités** : (a) finir ce qui existe et n'est pas fini ; (b) quick wins ; (c) seulement
   ensuite le changement de paradigme (lot P du plan v2), qui est gros.
2. **Un lot = une session d'agent = un worktree dédié = une branche `feat/<slug>`** (préfixe
   `feat/` obligatoire : la CI ne se déclenche pas sur `wt/`). Base : `feat/v75`. Jamais le
   worktree partagé `LevelUp-go-migration`.
3. **Brief fermé** : l'agent reçoit la liste des items du document source (numéros), le gate
   exact, l'interdiction d'étendre le périmètre, et le fichier où consigner. Il ne lance pas
   de sous-agent (sauf la revue adversariale quand le lot la prescrit).
4. **Découvertes** : consignées, jamais traitées dans le lot — SAUF P0 (donnée fausse servie,
   corruption, sécurité), traité immédiatement avec entrée au registre. Registres :
   Tactique → `.ai/DECOUVERTES_TACTIQUE_2026-09-07.md` ; rejeu/v2 → `.ai/V7.5/REGISTRE_REPORTS.md` ;
   libellés → §3 du plan libellés ; tout le reste → `.ai/V7.5/REGISTRE_REPORTS.md`.
5. **Modèle et effort** par lot (colonne « Exécutant ») : Sonnet pour le mécanique, le web
   d'interface et les tests ; Opus pour les diagnostics, les contrats et le paradigme.
   **Revue adversariale : UNE SEULE PAR VAGUE** (décision utilisateur 2026-09-07 : pas de
   revue par lot, trop coûteux), sur le diff cumulé des lots de la vague, juste avant leur
   fusion dans `feat/v75`. Elle REMPLACE les revues par lot prescrites par les plans source
   (y compris les deux rondes du plan v2). Une seule ronde de corrections ensuite.
6. **Un seul lot en vol par domaine** (Go sync / web / rejeu) pour éviter les fusions croisées.
   Deux lots de domaines différents peuvent tourner en parallèle.
7. **Gate minimal commun** : tests des paquets touchés (jamais `go test ./...` global sur un
   poste avec le jeu installé), `golangci-lint run --new-from-merge-base=origin/main`,
   typecheck + vitest si web, `openapi-gen -check` + `make generate-types` si contrat, entrée
   `.ai/thought_log.md`, push. **La CI n'est consultée qu'UNE fois par vague**, après le
   dernier push de la vague (décision utilisateur 2026-09-07 : pas de contrôle par lot) ;
   `gh run view`, jamais `watch`.
8. Le superviseur vérifie SUR PIÈCES avant de cocher, fusionne dans `feat/v75` seulement avec
   l'accord de l'utilisateur (règle 16 du dépôt). Push sur `main` = prod : jamais sans lui.

## 1. Verdicts établis sur pièces (2026-09-07)

### 1.1 Les trois branches

| Branche | Constat vérifié | Décision |
|---|---|---|
| `feat/outcome-cle-canonique` | 1 commit d'avance (`56e5d5ba0`), 213 de retard. Son unique changement (quatre littéraux du repli FR en constantes `goconst`) est DÉJÀ sur `feat/v75` par `b11bc6872`, mêmes valeurs, noms de constantes différents (`outcomeLabelTie` vs `outcomeFallbackTie`). Fusion = conflit sur `outcome_label.go` pour rien. | **SUPPRIMER** (locale + `origin`), retirer le worktree `LevelUp-wt-outcome-cle`. Rien à récupérer. Le lot L1 se poursuit sur une branche neuve (§2, Q4). |
| `feat/v2-audit-vies` | 1 commit (`370955a35`) : brouillon de `.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md`. `feat/v75` porte la version STATUÉE (41 lignes de plus, fusion `eb7a3dfbd` de `feat/v2-vies-anonymes`). Fusion = conflit add/add. | **SUPPRIMER** (locale + `origin`). Rien à récupérer. |
| `wt/blob-304-retry` | 6 commits, 406 de retard. Code : `sync/haloclient` (retry d'un blob CDN, 304 d'edge non traité en échec, corps coupé retenté, garde-rail `no_text_predicate_test`), 2 rondes de revue adversariale P0+P1 = 0, un seul item ouvert = observation J+7 après déploiement (report valide). **Aucun conflit de code** : `haloclient/` et `pooled_client_test.go` n'ont pas bougé sur `feat/v75` depuis la base `081871f09`. Conflits uniquement sur `.ai/thought_log.md` et `.ai/V7.5/REGISTRE_REPORTS.md` (garder les deux côtés). Branche `wt/` = jamais passée en CI. | **FUSIONNER dans `feat/v75`** (Q2). Puis supprimer la branche et les deux worktrees `LevelUp-wt-blob-304` / `LevelUp-wt-film-blobs`. |

Quick win annexe (Q1) : `git branch --merged feat/v75` liste ~95 branches locales déjà
fusionnées (`feat/duels`, `feat/v2-*`, `wt/*`…) et une soixantaine de worktrees
`LevelUp-wt-*` qui pointent dessus. Ménage = `git worktree remove` + `git branch -d`
(refuse ce qui n'est pas fusionné : sans risque). À faire par le superviseur, liste
présentée à l'utilisateur avant suppression.

### 1.2 Le chantier Tactique fusionné — était-ce justifié, est-ce correct, est-ce compatible avec P ?

**Justifié.** Les phases exécutées par Haiku 4.5 (7C.9, 5, 7B, 8) étaient toutes DANS le plan
d'origine (`PLAN_TACTIQUE_2026-09-06.md`) avec maquettes validées par l'utilisateur ; aucune
n'est une extension de périmètre. La dérive « usine à gaz » est ailleurs : dans les phases
6.5, 7.8, 7.10, 7C.8 (revues en cascade, ~40 constats corrigés dans le lot) — déjà revues
par Opus, CI verte, on ne les rouvre pas.

**Correct — vérifié par le superviseur :**
- 7C.9 (`0df2aa226`, correctif du « fait faux possible » horloge API vs film) : retrait propre
  de `ArriveeMS`/`DepartMS`, de la requête `match_registry` et du helper, 172 lignes en
  moins, aucun code mort ajouté. Conforme à l'option « V1 sans participation » du registre.
  Biais résiduel documenté : un coéquipier arrivé en cours de partie compte « hors de vue »
  (vivant) avant son arrivée → penche vers « accompagnée », jamais vers « isolée » (biais
  conservateur, acceptable en V1). Deux reliquats à traiter en finition (Q8) :
  `replay.EtatParti` (`death_context.go:73`) et son `case` (`:185`) ne sont plus jamais
  produits — exactement l'énumération morte que 7C.1 avait combattue ; l'entrée du
  registre « `LEFT JOIN match_registry` sans test » est caduque (le JOIN a disparu).
- Phase 8 : **NON CLOSE** malgré le titre du commit `05548d4e4`. Cases 8.1 à 8.4 vides dans le
  plan ; 8.4 (revue adversariale du diff intégral avant merge) jamais faite ; la CI du merge
  `f2c8ddce1` était encore `in_progress` à la rédaction. Le commit `6c960861c` (phase 5
  posée par erreur directement sur `feat/v75`, ESLint rouge — registre l.578) est réputé
  corrigé par la « passe 2 » `bf5d465bd` : à confirmer par la CI, pas par le journal.
- Phases 5 et 7B (~3 100 lignes web + Go, Haiku) : relues le 07/09 au soir (Sonnet, contexte
  frais, lecture seule, tests ciblés verts). **P0 = 0, P1 = 2, P2 = 1.** Vérifiés SANS défaut :
  plancher 3 aligné serveur (`merge.go:16`), bandeaux attente/indisponible, bulle 6-30 px,
  session < 5 morts exclue, clic/cellule cohérent. Constats (au HEAD `f2c8ddce1`) :
  P1-1 `features/tactical/TacticalAnalysisView.tsx:184-199` — tuiles KPI « Échange » et
  « Isolement » n'affichent jamais la réserve `echantillon_faible` publiée par le contrat
  (doctrine : le drapeau interdit de comparer, il ne cache pas) ; P1-2
  `features/squad/.../SquadIsolementNuageCard.tsx:210-223` + `squadIsolementStrings.ts:34` —
  chaîne « échantillon faible » définie, jamais rendue, seul signal = opacité 0,35, non
  testé ; P2-1 `features/tactical/i18n.ts:73-76` — quatre clés (`planOf`, `footerHeatmap`,
  `footerRoutes`, `footerEmpty`) jamais consommées. Correctif d'affichage sans contrat →
  Q8 (a). Rapport : scratchpad `REVUE_TACTIQUE_PHASES_5_7B.md` (copie à ranger sous
  `.ai/V7.5/` par Q8).

**Compatible avec le lot P — à trois conditions, écrites ici pour P1 :**
1. Les faits d'isolement au sync (7C) consomment `replay.ScanPlayerIndices` +
   `replay.ResolveSlotXUID` (`killcollector/positions.go:258-266`) : le pont par morts du
   paquet `replay`, PAS un pont maison. Acceptable aujourd'hui ; mais quand P2 remplacera ce
   pont par le registre d'identité, `positions.go`, `isolation_facts.go` (`occupantA`, garde
   `IndexDisagreements`) sont des LECTEURS À MIGRER, au même titre que les calques du rejeu.
   Ils portent `decoder_rev` (`IsolationDecoderRev`) : un bump à P2 fait réécrire les deux
   tables par `backfill-killsource`. À inscrire dans l'inventaire P1.
2. Le registre d'identité de P doit vivre dans `analysis/replay` (PUR), pas seulement dans
   `replaybuild` : le principe de 7C (« les données d'un match sont complètes au sync, seul le
   rejeu attend la cuisson ») exige que le collecteur puisse le calculer sans cuire. Le plan v2
   §0.7 dit « calculée une fois dans `replaybuild` » : à amender en « calculée par une fonction
   pure de `analysis/replay`, appelée par `replaybuild` ET par le collecteur ».
3. **S.3 (arrivées/départs par l'horloge API) est GELÉ derrière P1.** Tel qu'écrit, S.3
   construirait un calage déduit (horloge API → film) alors que l'axe (d) ROSTER de P1 doit
   d'abord dire si le film porte directement les entrées/sorties de joueurs (et l'axe (a) TEMPS
   quelle horloge directe il porte). Faire S.3 avant P1 = fabriquer une déduction que P
   condamne. Le plan de suite Tactique est amendé en ce sens (§4).

Point de vigilance PROD (registre Tactique, non traité, à surveiller au déploiement de v75) :
la capture des positions est désormais branchée au sync de prod (scan des bipèdes par match
synchronisé) — coût inconnu, `PostSyncBudget` à observer sur les premiers cycles.

### 1.3 Les plans

- `PLAN_TACTIQUE_SUITE` : le pré-requis P.1 est FAIT (fusion `f2c8ddce1`). S.2 est un quick win,
  mais sa prémisse est fausse : le peintre partagé n'est PAS dans `lib/replay/` mais dans
  `features/match-replay/layers/heatmapLayer.ts` (`buildHeatmap`, `drawHeatmapLayer`) ; S.2
  doit donc DÉPLACER le noyau dans `lib/replay/` (garde `crossFeatureBoundary.guard.test.ts`)
  puis supprimer `features/tactical/heatPaint.ts`. S.1 exige un changement de contrat Go
  (identifiants de match par cellule + filtrage ownership ADR 0029) : lot moyen. S.3 gelé (§1.2).
- `PLAN_V2_RESTES` : base atteinte (`SchemaVersion = 48` sur `feat/v75`, `feat/v2-integ` à 0
  d'avance). R0, R8 = quick wins mécaniques. R6, R7 = petits. R4, R5 = diagnostics. P =
  gros, phasé. R9 = release, main de l'utilisateur.
- `PLAN_LIBELLES` : L0 mesuré au HEAD : 45 fichiers Go hors tests avec mojibake, 16 littéraux
  (le plan disait 13). L1 : **8** appelants du repli `outcomeLabel(` restent (le plan en
  comptait 5) : `career_service_encounters.go:341`, `explorer_service_convert.go:85`,
  `match_history_explorer_options.go:94`, `match_history_service_enrich.go:134`,
  `match_view_builders_header.go:194`, `match_view_builders_summary.go:88`,
  `match_view_builders_team.go:95`, plus `outcome_label.go:82` (le repli lui-même).
  `HomeMatchRow.OutcomeLabel` : 0 lecteur dans `features/home` (confirmé). L2 à L8 attendent
  les décisions §7 du plan (reprises en §5 ici avec une recommandation).

## 2. Vague 1 — finitions et quick wins (dans cet ordre ; Q1-Q2 par le superviseur)

| # | Lot | Source | Exécutant | Gate de clôture | Statut |
|---|---|---|---|---|---|
| Q1 | Ménage des branches : suppression de `feat/outcome-cle-canonique`, `feat/v2-audit-vies` (locales + origin) et de leurs worktrees ; liste des ~95 branches fusionnées présentée puis `git branch -d` | §1.1 | Superviseur | `git branch --no-merged feat/v75` ne perd rien ; `git worktree prune` | [x] 2026-09-07 : 2 branches absorbées supprimées (locales + origin) ; 73 worktrees propres retirés, 97 branches locales `-d` ; 11 worktrees SALES laissés (liste : scratchpad `menage_worktrees.log`) ; 2 dossiers verrouillés par un autre processus (`.claude/worktrees/t0-film`, `LevelUp-wt-frise-pov/apps/web`) — à retirer à froid ; `wt/fonds-manquants` non fusionnée, gardée |
| Q2 | Fusion `wt/blob-304-retry` → `feat/v75` : worktree dédié, `git merge`, deux conflits doc résolus « les deux côtés », `go test ./internal/sync/haloclient/ ./internal/sync/ -run 'Blob\|Pooled\|TextPredicate'`, lint new-from-merge-base, push, CI | §1.1 | Superviseur (accord utilisateur avant commit) | tests haloclient verts, lint ; entrée thought_log (CI : fin de vague) | [x] 2026-09-07 : `666b02d17` sur `feat/v75` (poussé) ; branches `wt/blob-304-retry` et `wt/film-blobs-expires` supprimées (locales + origin) |
| Q3 | Mojibake : réencoder les 45 fichiers (16 littéraux d'abord, vérif à l'octet), garde-rail `archlint/no_mojibake_test.go` (allowlist vide), cause écrite dans le test | Libellés L0 | Sonnet, effort moyen | `go build ./...`, `go vet`, tests des paquets touchés, garde vert ; `git diff --stat` = uniquement des octets `C3 83…` → caractères | [x] 2026-09-07 : `feat/mojibake-garde-rail` (`63241db93` + `058e632e6` + `d435d1611`, poussée) ; 62 fichiers Go réencodés, 0 TOML touché, 62 faux positifs `Â` exclus ; garde-rail `archlint/no_mojibake_test.go` allowlist vide ; vérifié sur pièces par le superviseur : 1 résidu (`home_service.go:251`, U+00C3 en fin de ligne) corrigé et règle durcie (U+00C3 seul = défaut), mutation rejouée. Découvertes au §9 du plan libellés (5 `fmt.Errorf` FR, corruption secondaire) |
| Q4 | Issue de match, fin de L1 **par la CLÉ CANONIQUE (décision D5, 2026-09-07)** : chaque DTO qui portait `outcome_label` porte `outcome` = clé `win|loss|tie|dnf` (`mappings.Canonical(rawCode)`) ; les 8 appelants du repli et `resolveOutcomeLabel` disparaissent avec la map FR et son kill-switch ; `HomeMatchRow.OutcomeLabel` + maps de `home_locale.go` supprimés (0 lecteur, D4) ; `duelOutcomeLabel` (Explorer) : clé `won|lost` + libellé web, même mécanique ; côté web les lecteurs de `outcome_label` (Match View `MatchHeader.card.tsx:362`, historique, Explorer, export CSV, Carrière) passent par `useOutcomeLabel`/`useOutcomeMapping` ; l'export CSV, qui a besoin d'un texte, le rend côté web ; `openapi.yaml` + `make generate-types` ; ratchet FINAL `no_french_label_literal_test.go` posé avec le compte du jour | Libellés L1 + garde-rail final (option 1) | Sonnet, effort élevé (contrat + ~20 lecteurs web) — [x] 2026-09-07 : `feat/issue-cle-canonique` (`898bb3084` Go+contrat, `a354815f0` web, `18c7f1c50` journal, base mojibake, poussée) ; 7 DTO `OutcomeLabel` → `Outcome` (clé, enum), `RecentMatchItem.OutcomeLabel` supprimé, maps FR de `home_locale.go` et kill-switch supprimés ; CSV rendu serveur → texte via l'adapter sémantique du titre (`OutcomeText`), jamais une map ; `duelOutcomeLabel` déjà conforme (clés) ; ratchet `no_french_label_literal_test.go` posé : 132 fichiers / 538 littéraux (ampleur réelle, famille erreurs API en tête) ; vérifié sur pièces par le superviseur : 0 littéral d'issue FR dans service/analysis/domain, `MatchHeader.card.tsx` sur `useOutcomeLabel`, `generated.ts` régénéré ; parité FR/EN à l'écran À VÉRIFIER par l'utilisateur (Match View, Explorer + CSV, historique, Carrière, accueil) | `go test ./internal/service/... ./internal/analysis/...`, `openapi-gen -check`, `make generate-types` + diff vide ou commité, vitest des consommateurs web, parité FR/EN à l'écran (Match View, Explorer, Carrière) par l'utilisateur | [ ] |
| Q5 | Flakes CI : `TestStartImport_HappyPathReturns202WithJobID` (attendre la fin du job / fermer le store) ; `TestWorker_Run_PersistsAndACKs` (remplacer l'attente implicite par une synchronisation) | v2 R8 | Sonnet, effort bas | `go test -count=20 -p 8 -run <Nom>` vert, deux fois | [x] 2026-09-07 : `feat/ci-flakes-import-worker` (`2a20af702` + journal `8fc6f42f7`, poussée) ; vérifié sur pièces par le superviseur : attente de l'état terminal du job via `pollJobUntilDone` existant ; synchro sur `OnPersistOK` (après ACK) ; 0/20 x 2 ; intégration persist `-p 1` verte. Découverte consignée : `TestWorker_Run_PersistFailure_NoACK` a un `time.Sleep(200ms)` de la même famille |
| Q6 | Dette mécanique du rejeu : scissions `document.go` / `usage_summary.go` / `replaybuild.go` / `flag_carries_test.go`, commentaire faux `equipment_episodes_test.go:371-372` ; 0 bump | v2 R0 | Sonnet, effort bas | goldens INCHANGÉS (`git diff --exit-code testdata/`), preuve de déplacement pur (concaténation triée avant/après identique), lint 0 | [x] 2026-09-07 : `feat/v2-restes-r0` (6 commits `a0574cc46`..`676596a54`, poussée, base mojibake) ; vérifié sur pièces par le superviseur : 0 fichier `testdata` touché, `SchemaVersion` 48, `flag_carries_identity_test.go` intact (incident d'écrasement intercepté par l'exécutant avant commit) ; preuves de déplacement pur vides sauf la glue de fonction d'`options.go`. RESTE au-dessus de 500 L après l'extraction prescrite : `document.go` 553, `usage_summary.go` 513, `replaybuild.go` 545, et le nouveau `document_chronicle.go` 959 (chronique = commentaires) — consigné, pas de découpe au-delà du plan |
| Q7 | Un seul peintre de chaleur : noyau `buildHeatmap`/`drawHeatmapLayer` déplacé de `features/match-replay/layers/heatmapLayer.ts` vers `lib/replay/heatPaint.ts` (entrée « cellules pré-agrégées » ajoutée LÀ), `features/tactical/heatPaint.ts` supprimé, garde-rail grep (aucune seconde implémentation hors `lib/replay/`) | Tactique S.2 (prémisse corrigée §1.3) | Sonnet, effort moyen | `tacticalGridFromRaster` + snapshot léger identiques, typecheck, vitest complet, `crossFeatureBoundary.guard` vert, garde-rail vert | [x] 2026-09-07 : `feat/peintre-chaleur-unique` (`d6ebb9e9d`, poussée) ; noyau `lib/replay/heatPaint.ts` (498 L) à deux entrées, `heatmapLayer.ts` et `features/tactical/heatPaint.ts` supprimés, garde-rail des cinq noms + mutation ; vérifié sur pièces par le superviseur (0 copie hors `lib/replay/`, 0 import de feature) ; vitest 653 fichiers / 6 979 tests, lint 29 warnings inchangés |
| Q8 | Finition Tactique (clôture réelle de la phase 8) : (a) les trois constats de la revue (§1.2) : réserve `echantillon_faible` rendue sur les tuiles KPI Tactique et sur le nuage Escouade (avec test), quatre clés i18n mortes retirées ; (b) `EtatParti` mort retiré ; (c) registre Tactique, entrées « petites » : regex `no_raw_rating_reads_test.go:44` (3 tables), WARN + compteur sur le refus de positions (`positions.go:103`), WARN sur `settingsStore.Load()` en erreur (cron de purge silencieux — 2 appelants), compteur dédié pont non publiable (`isolation_facts.go:176`), test d'orthogonalité non tautologique (`lives_export_test.go:180-193`), test film réel réparé (roster depuis `ScanDeaths`, `t.Fatal` si 0 vie) ; entrée « LEFT JOIN sans test » fermée comme caduque ; (d) cases 8.1-8.4 du plan Tactique statuées ; (e) `matchs_sans_rayon` : note de couverture sur la tuile « Morts en isolement » (contrat déjà publié, 0 SQL) | Découvertes Tactique + plan Tactique phase 8 | Sonnet, effort moyen ; le test film réel se joue SUR LE POSTE PRINCIPAL (`KILLSOURCE_FIXTURES`) par le superviseur | `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...`, `no_art_patterns_test`, typecheck + vitest (gate-push remplacé par la CI de fin de vague) | [x] 2026-09-07 : `feat/tactique-cloture` (3 commits `eb8e79834`, `afa0a9cfc`, `fed64de2f`, base peintre + mojibake fusionnée sans conflit, poussée) ; vérifié sur pièces par le superviseur : réserve « échantillon faible » rendue et testée sur les tuiles Tactique et le nuage Escouade, note `matchs_sans_rayon`, 4 clés i18n mortes retirées, `EtatParti`/`Partis` supprimés (ne subsistent qu'en commentaire d'historique), regex de lecture brute élargie aux 3 tables, WARN + compteur sur le refus de positions, WARN sur `settingsStore.Load()` (cron de purge et verrou d'instance), compteur dédié « pont non publiable », test film réel réparé (à jouer sur le poste principal avec `KILLSOURCE_FIXTURES`), phase 8 du plan Tactique statuée |

Non retenus en vague 1 (consignés, pas oubliés) : canvas tactique à `k = 1` sans `devicePixelRatio`
(choix assumé) ; double lecture des artefacts par cycle dans `projeterRastersTactiques`
(optimisation, à regrouper avec « le document qui circule entre projections » — vague 2) ;
test `:memory:` dédié à `QTacticalIsolement` (entre dans Q8 si la revue le juge P1, sinon vague 2).

**VAGUE 1 CLOSE le 2026-09-07** : fusionnee dans `feat/v75` (`6a1496e30`), revue unique P0 0 / P1 1 corrige, CI verte (run 34157192772, 8 jobs verts + E2E skip). Deux rouges de CI corriges au passage, hors lots : baseline des tests Go (8 tests `TestOutcomeLabel*` retires avec les maps FR) et `MatchStatCards.test` (mock du store sans `availableTitles` apres le commit parallele `9af8aea41`).

## 3. Vague 2 — finitions moyennes (après vague 1 ; un lot par domaine en vol)

| # | Lot | Source | Exécutant | Gate |
|---|---|---|---|---|
| M1 | Lien « voir dans le rejeu » depuis une cellule : contrat `POST …/tactical/{map_id}/cellule` (contributions `{match_id, instant_ms, xuid}` filtrées par ownership ADR 0029 + `matchs_non_ouvrables`), service, handler, `TacticalCellCard` avec lien `?frame=` (`playbackStore`), instant → frame en logique pure | Tactique S.1 (5.5 + 5.6 complet) | Opus effort bas pour le contrat + Sonnet pour le web | test « un match d'un autre joueur n'apparaît pas mais compte », `?frame=` positionne le rejeu, contrat régénéré, typecheck, vitest — [x] 2026-09-08 : `feat/tactique-lien-rejeu` (`bd2e20726`, `3d70de951`, `ca33684ba`, base v75, poussée ; exécuté en deux passes Sonnet, la première coupée par le quota) ; contrat `POST …/tactical/{map_id}/cellule` → `contributions[{match_id, instant_ms, xuid, match_started_at}]` + `matchs_non_ouvrables`, ownership ADR 0029 en défense en profondeur ; web `TacticalCellCard` + `instantToFrame` + 16 tests ; DÉCOUVERTE À TRAITER AVANT FUSION (§6) : `instant_ms` sur l'horloge du MATCH pour 4 questions sur 6 → `?frame=` décalé de 3,6 à 50,8 s ; l'appareil T0 existe (`match_events_source.go`, `real_start_time`) |
| M2 | Constats P2 de l'audit des vies (5) + `drawnSwapAt` borné à la vie courante côté web | v2 R6 | Sonnet, effort moyen | mutation par constat, vitest, gates v2 — [x] 2026-09-07 : `feat/v2-restes-r6` (`a77f16ab7`, `0a3c4bc42`, base R0, poussée) ; P2-3 non-lieu (déjà corrigé par `f1b4f4ee5`), P2-4 et P2-5 corrigés côté web avec mutations (`buildSlotOwnership`, `rosterEntryKey`), `drawnSwapAt` borné à la vie courante (`equippedLogic.ts`, pas `lib/replay/`) ; P2-1 et P2-2 REPORTÉS : racine dans `filmdec` (décodeur gelé §0.6) → plan décodeur après v7.5.0 ; aucun bump (48), décodeur intact, vérifié sur pièces par le superviseur ; gate corpus sans objet (contenu cuit inchangé) |
| M3 | Budget de candidats du calage (adaptatif ou dédoublonnage par fin de vie) ; test de documentation RETOURNÉ ; bump 49 partagé avec le lot suivant qui bumpe | v2 R7 | Opus, effort moyen | `d9781168` / `51ebbc0f` identiques hors numéro — [x] 2026-09-07 : `feat/v2-restes-r7` (`0a6fb7ddf`, `52adb276a`, base R6, poussée) ; cause : le vote comptait des morts distinctes, l'affinage un appariement 1:1 — un amas votait M fois sur une fin de vie isolée ; correctif : voix d'un panier = min(morts distinctes, fins distinctes) (borne de l'appariement), aucun seuil nouveau ; test retourné (`…NEmportePasLeBudget`, hors baseline), mutation rouge→vert ; goldens inchangés, PAS de bump (48) ; gate corpus joué par le superviseur le 2026-09-08 sur les 7 témoins (films re-téléchargés) : 0 perte, 1 gain (`bf15f7ab`), 48/48 partout — PROUVÉ ; découverte : `lives.go` 509 → 543 L (dette préexistante, registre R0) |
| M4 | Diagnostics : écart K/D/A sur `51ebbc0f` (R4) ; re-vérification CTF multi-manche + VIP/crâne (R5) | v2 R4, R5 | Opus, effort moyen (diagnostic = jugement) | journal chiffré par joueur et par manche ; verdict source vs lecteur ; registre fermé ou rouvert avec chiffres — [x] 2026-09-08 : `feat/v2-restes-r4r5-diag` (`d205625da`, `ba47093b6`, docs seuls, poussée) ; R4 : écart cumulé 9 (pas 69 : films re-téléchargés, artefacts identiques à l'octet), UN joueur perd sa manche 0 (7/15 frags, 1/2 assists) ; VERDICT LECTEUR = nommage par manche : `bestDeathClaim` exige 3 morts (`deathInstantMin`), le rattrapage `CompletedByLines` est mono-manche, `buildPlayerScores` ne passe pas `Lines` en multi-manche ; même racine sur `64e8adfa` (16, une capture perdue) et `d9781168` (10) ; correctif proposé `CompletedByElimination` par manche = PÉRIMÈTRE DE P2 (élimination sur le roster, plan v2 §0.7) → pas de lot séparé ; R5 : 7/7 témoins 0 perte, captures = score sauf `64e8adfa` (2/2 vs 2/3) et `fb1a1a72` (0/0 vs 0/1) ; crâne 36 portages sur `d9781168` mais 6/36 et 1/19 portages FANTÔMES (hors vie bipède) rouverts au registre → P3 ; VIP : aucun film au parc (466 recensés) `[~]` ; hypothèse « CTF multi-manche » réfutée (pont parfait, `objectives` ne nomme que 3 actions sur 637 → P2/P3) |
| M5 | Libellés L2 (accueil), L3 (modes/playlists → `assets.toml`), L4 (armes), L5 (rangs → `mappings/ranks.go`) | Libellés | Sonnet, effort moyen, un lot par famille | parité FR/EN à l'écran, ratchet qui baisse, `no_slug_comparison_test` — L2+L5 [x] 2026-09-07 : `feat/libelles-accueil-rangs` (`ee65ec07f`, `7161d18fe`, `92ae3fe3d`, base v75, poussée) ; accueil : composite `Title` et `OutcomeText` supprimés (le web compose le repli), les paires `labelForLocale` de l'accueil sont des noms d'ASSET (map/mode/playlist) → L3 ; rangs : `csrUnrankedLabel` → clé `unranked`, web `lib/skillTiers.ts` ; ratchet 538 → 537 ; découvertes §11 du plan libellés : tier CSR formaté en clair 3 fois dont une PERSISTÉE dans `match_skill_rank.tier_label` (migration de données) ; L3 [x] 2026-09-08 : `feat/libelles-modes-playlists` (`6eb62c893`, `2f190c543`, base L2+L5, poussée, deux passes Sonnet) ; playlists classées : noms sortis du Go vers un TOML EMBARQUÉ dans le paquet `games/halo_infinite/rankedplaylists` (title-scoped par construction ; le loader par titre est inapplicable aux appelants purs des migrations et jobs de sync — condition de reprise §12 du plan libellés) ; `mode_category.go` et les paires de l'accueil déjà conformes (clés + données en base) ; ratchet 537 → 522 ; nouveau garde `TestNoNewModePlaylistLabelLiteral` ; NON TRAITÉ : `match_history_service.go` portées/expériences (VALUE FR = contrat testé avec ~80 fichiers web, lot dédié avec décision produit) ; L4 [x] 2026-09-08 : `feat/libelles-armes` (`eb350a44a`, `67afc4e78`, base L3, poussée) ; `weapon_families.name_en/name_fr` = DONNÉE MORTE (aucun lecteur Go ni web ; le nom par arme vient de `weapon_name_labels`/`weapon_names.toml`) → supprimée du registre Go ET des colonnes de `metadata.duckdb` par migration CTAS-swap (précédent `purge_weapons_name_fr_column`) — POINT PROD : migration au boot du prochain déploiement ; `labels.go` (12 littéraux) = repli seedé VIVANT de `weapon_labels`, `[!]` hors périmètre ; ratchet 522 → 500, garde `TestNoNewWeaponFamilyLabelLiteral` ; découvertes §12 : `WeaponLabel`/`TopWeaponLabel` de la fiche match encore en option 2 (chantier séparé). Famille L3 : portées Explorer restent [!] (lot dédié) |
| M6 | Le document de rejeu circule entre projections : `ProjeterRasterTactique` réutilise `a.doc` au lieu de relire le fichier | Découvertes Tactique (fusion v75) | Sonnet, effort bas | durée d'un cycle `Deriver` mesurée avant/après, goldens inchangés — [x] 2026-09-07 : `feat/raster-document-unique` (`25a9eb8a9`, base mojibake) ; `projeterRasterDepuisDocument(doc)` extrait, `projeterRastersTactiques` réutilise `r.doc` ; test compteur d'ouvertures rouge (1) → vert (0) ; vérifié sur pièces par le superviseur ; intégration `-p 1` replayartifacts + persist verte |

**VAGUE 2 CLOSE le 2026-09-08** : fusionnee dans `feat/v75` (`66e90be64`), schema de rejeu 49, gate corpus 7/7 (0 perte), revue unique P0 0 / P1 0 / P2 1, CI verte (run 34201813049, 8 jobs verts + E2E skip). Restes consignes : portees Explorer (lot dedie a decider), correctif R4 = P2, portages fantomes du crane = P3, flake temporel `LoaderRunsInParallel`, `labels.go` armes, tier CSR persiste en clair.

## 4. Vague 3 — le paradigme (lot P du plan v2) et ce qu'il débloque

- **P1 — Inventaire** (journal seul, AUCUN code) : Opus, effort élevé. Ajouter aux axes (a)-(h)
  du plan v2 : (i) les lecteurs HORS rejeu à migrer — `killcollector/positions.go`,
  `isolation_facts.go`, tables `match_lives` / `match_death_context` (§1.2) ; (j) l'emplacement
  du registre = fonction pure de `analysis/replay` (§1.2). P1 rend un VERDICT écrit pour S.3
  (le film porte-t-il les entrées/sorties ? sinon S.3 se fait par calage `real_start_time` /
  `t0_quality`, comme prévu, mais APRÈS P2).
- **P2 → P5** : Opus, une phase = un lot clos (gate + journal), UNE revue adversariale à la
  clôture de P5 sur le diff cumulé (règle §0.5), bump 50 unique (49 pris par le lien exact le
  2026-09-08).
- **Statut** : P1 [x] 2026-09-07 par une autre session (`77200cb7a`,
  `.ai/V7.5/v2/RESTES_P1_INVENTAIRE_2026-09-07.md`), vérifié par le superviseur le 08/09 : axes
  (i) et (j) couverts, verdict roster = le film ne porte pas les entrées/sorties → S.3 par calage
  `real_start_time` après P2 ; séquençage P2-P5 §3 repris tel quel. P2 lancé le 2026-09-08 (Opus,
  `feat/v2-p2-registre-joueurs`, worktree `LevelUp-wt-p2`, commits fréquents) — go utilisateur
  « ok go pour la vague 3 ».
- **P2 [x] 2026-09-08** (`feat/v2-p2-registre-joueurs`, 6 commits Opus + polarité du gate `632bd20c8` + P2-bis
  `1496b386c`..`fd07570bc`, poussée, NON fusionnée) : registre pur `BuildIdentityRegistry` (liens directs
  d abord, pont par morts en repli et vérification, élimination sur le roster, exclusion temporelle),
  section `identity` publiée (schéma 50), 18 lecteurs du rejeu + collecteur migrés, garde-rail à
  allowlist datée. MESURÉ par le superviseur (base 49 / HEAD 50, 10 témoins) : 0 perte de calque ;
  `3372e7eb` unpublished 35 → 0 ; écart K/D/A `51ebbc0f` 9 → 0, `d9781168` 10 → 0 ; `unnamedLives`
  `d9781168` 19 → 15, `51ebbc0f` 8 → 3, `64e8adfa` 11 → 7, `bf15f7ab` 1 → 0 ; portage du crâne inchangé ;
  tirs rattachés +187/+149/+147. Résidu 13 pistes expliqué (grappes de fin de manche, attend le lien
  décodeur E2). Deux compteurs d échec MONTENT honnêtement (`slotCollisions` 0 → 1, `unnamedLivesContested`
  0 → 2 : collision jusque-là servie en silence) — à accepter au gate corpus (registre). `filmIndex`
  8/8 direct partout ; `bipedSlot.direct` 0 partout (E2).
  R1, R2, R3 ne s'exécutent pas séparément.
- **S.3** (Tactique) : après P1 selon son verdict, avec le calage prescrit par P1.
- **L6, L7, L8** (narratif, erreurs API, Discord) : après décisions §5.
- **R9** : release, séquence Notion, main de l'utilisateur.

### 4.1 Suivi d'exécution de la vague 3 (ouverte le 2026-09-07 à la demande de l'utilisateur)

| # | Lot | Source | Exécutant | Gate de clôture | Statut |
|---|---|---|---|---|---|
| P1 | Inventaire : table des entités (identifiant dans le film, durée de vie, liens DIRECTS, replis avec `fichier:ligne` du calque), axes (a)-(h) du plan v2 + (i) lecteurs hors rejeu + (j) emplacement du registre ; VERDICT écrit ROSTER pour S.3 ; séquençage P2-P5 ; questions ouvertes avec leur témoin | v2 lot P item P1 ; §1.2 et §4 ici | Opus, effort élevé | `.ai/V7.5/v2/RESTES_P1_INVENTAIRE_2026-09-07.md` complet (aucun axe vide), registre, thought_log, commit `docs(restes/p1):` poussé ; ZÉRO code | [x] 2026-09-07 : `7cbec66f4` sur `feat/p1-inventaire`, poussé (`origin` confirmé) ; livrable 569 L, 9 entités, axes (a) à (j) tous statués, 18 liens directs contre 23 replis, 3 découvertes au registre ; **0 fichier de code touché** (vérifié sur pièces par le superviseur) ; verdicts en §4.2 |
| P2 | Registre des joueurs (= R1 + R2) : table d'identité unique publiée, lien direct à 100 %, pont par morts en repli et vérification, migration des lecteurs joueurs, garde-rail anti-pont maison | v2 lot P | Opus | ouvre après P1 statué ET après fusion des lots rejeu en vol (règle §0.6) | [x] 2026-09-08 : `feat/v2-p2-registre-joueurs` (`333c7f3e9`, `d9548a5dc`, `58da800a1`, `871cfaa51`, base `feat/v75` `7254b3853`, poussée, worktree `LevelUp-wt-p2`) ; `BuildIdentityRegistry` PURE dans `analysis/replay`, appelée par `replaybuild` ET par `sync/killcollector` (amendement §0.7 / D11 porté au 1er commit) ; types canoniques `games/canonical/film_identity.go` (enums fermées, aucun slug) ; section `identity` publiée jusqu'au contrat (+ `roster[].bid`, E9 fermé côté Go) ; bump **49 → 50** (le 49 était pris par M1b) + chronique + cliquet 56 → 57 + goldens (un seul écart : le numéro de schéma — témoin de neutralité) ; R1 (actions jamais jetées), R2 (élimination sur le roster) et le correctif R4 reporté par M4 livrés ; 18 lecteurs migrés + les 2 hors rejeu, `killpos_bridge.go` supprimé ; garde-rail `archlint` allowlist datée à UNE entrée, 7 mutations jouées ROUGE ; `IsolationDecoderRev` bumpée → `levelup backfill-killsource` ; gates verts (vet, 53 paquets `ok`, intégration `-p 1` 11 paquets, parité, `openapi-gen -check`, lint 0 issue, web typecheck + vitest 7 024 tests). **Témoins chiffrés et `replay-corpus-gate` `[!]` : à jouer par le SUPERVISEUR** (aucun film dans le worktree), commandes et chiffres attendus au §7 de `.ai/V7.5/v2/RESTES_P2_2026-09-08.md` |
| P2-bis | Résidu de R2 : les 19 vies sans nom de `d9781168` (vague 3, §4) | v2 lot P item R2 | Opus | ouvre après le relevé des témoins de P2 | [x] 2026-09-08 : `feat/v2-p2-registre-joueurs` (`1496b386c` journal, `fb953ee94` code), même worktree, **même bump 50** (P non fusionné : le contenu cuit change sous le même numéro). **La prémisse de R2 était FAUSSE** — aucun joueur de `d9781168` n'a zéro mort (13 au minimum à la feuille), les 19 vies sont sur 18 slots distincts et 18 slots muets subsistent, d'où `parElimination = 0` : le code de P2 n'était pas en panne, le cas qu'il ferme n'existe pas sur ce film. Règle ajoutée : **exclusion temporelle** (l'élimination du roster portée à l'intervalle d'une vie ; un joueur n'occupe qu'un slot à la fois), itérée jusqu'au point fixe, abstention à deux candidats ET à zéro (contradiction alarmée), trois garde-fous (roster de la feuille, aucun bot, occupation ≤ roster), voie canonique `exclusion_temporelle`. Re-cuisson locale des 10 témoins : `d9781168` `unnamedLives` **19 → 15** et `bipedSlot` `142/34 → 152/24`, portage du crâne **inchangé (172,5 / 158,8 s)** ; `51ebbc0f` **8 → 3**, `64e8adfa` **11 → 7**, `fb1a1a72` **3 → 2**, `bf15f7ab` **1 → 0**, `c0a82e88` `non_resolu` **1 → 0** ; 4 films inchangés ; tirs rattachés +187/+149/+147/+28/+12, aucun calque en perte ; `scoreTimeline` et `statborgSlots` identiques. 10 mutations ROUGE, goldens inchangés, lint 0 issue. **Résidu de 13 pistes expliqué vie par vie** (§B.2/B.5 du journal) et inscrit au registre ; `slotCollisions` +1 et `unnamedLivesContested` +2 **à accepter au gate corpus** (§B.8) |
| P-décodeur E2 | Le lien DIRECT corps ↔ joueur : le record de création du bipède (`ti=35`) porte l'index de participant de son propriétaire (sondage E2, décisions D12/D13) | sondage `.ai/V7.5/film_re/SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md` §4 | Opus | re-cuisson des 5 films : `bipedSlot.direct` ≥ 95 %, `non_resolu` ventilé par cause, 0 vie nommée par le pont | [x] 2026-09-08 : `feat/v2-decodeur-e2` (`44bed2167` I0, `67fbbac0c` I1, `57196477d` I1b, `b6b0ee4c5` I2-I4, `079318959` + `84445fede` I5), worktree `LevelUp-wt-e2`, base `feat/v2-p2-registre-joueurs` + docs du sondage, **même bump 50** (P non fusionné). **BRIEF AMENDÉ EN COURS DE LOT** (« je ne vois pas le besoin d'un repli ; il faut chercher à comprendre les 5 à 13 % restants ») : le pont par morts devient VÉRIFICATION SEULEMENT, et le résidu est instruit par CAUSE au lieu d'être replié. **I0** : l'espace d'index est PARTAGÉ humains/bots, l'index 8 de `c75f33b8` reste `non_resolu` cause `index_hors_table`. **I1** : `filmdec.ScanBipedCreations`, gate par SIGNATURE (43 bits déterminés — l'ancre i0 de `runCreationWalk` est inatteignable sur le bipède), 10 tests bit-exacts dont une mutation d'un bit et un témoin fantôme à 1 000 ancres parfaites. **I1b** : le résidu instruit SANS relire un film, sur les TSV du sondage — **un seul record de création par slot, génération invariante, bijection exacte slots-avec-vies ↔ slots-avec-lecture** sur les cinq films ; donc dans un film un slot de bipède est UN CORPS. Les 53 vies du résidu : 51 propagation, 2 `index_hors_table`, **0 « non lu », 0 « sans record »**. **I2-I4** : lien direct posé d'abord, propagation par corps (`creation_bipede` / `creation_bipede_propagee`, toutes deux `direct`), `nameLivesByDeaths` SUPPRIMÉ, `coverage.bridge.{concordant,discordant,bridgeNamedLives}`, `bipedSlot.{direct_propage, non_resolu_par_cause}`, collecteur de sync branché, `IsolationDecoderRev` → `isolement-2026-09-08-creation-bipede`. 8 mutations ROUGE. **I5, re-cuisson des 5 films** : `bipedSlot.direct` **0 % → 100/100/100/100/97,7 %** ; `unnamedLives` **15/0/7/8/14 → 0/0/0/0/1** ; `shots.noSlot` **513/12/213/289/450 → 64/12/13/10/119** ; **27 discordances du pont corrigées** ; `slotCollisions` et `unnamedLivesContested` de `d9781168` **redescendus à 0** (les deux hausses que P2-bis faisait accepter) ; `scoreTimeline` et `statborgSlots` IDENTIQUES octet pour octet (écart K/D/A inchangé : 0/0/16/0/21) ; **zéro compteur d'échec ne monte** ; goldens **inchangés** (témoin de neutralité). Seule baisse : le portage du crâne de `d9781168` **172,5 → 172,4 s**, UNE frame sur UN portage — le portage ne survit plus d'une frame à son porteur, dont la vie est passée de `déduite` à LECTURE (gain d'exactitude, instruit §I5.5). Gates : gofmt/build/vet EXIT 0, batterie de paquets EXIT 0, intégration `-p 1` EXIT 0, `openapi-gen -check` EXIT 0, `golangci-lint --new-from-merge-base` **0 issues**, web typecheck EXIT 0, vitest **658 fichiers / 7 024 tests** EXIT 0. **`replay-corpus-gate` complet `[!]`** : à jouer par le SUPERVISEUR (§I7 de `.ai/V7.5/v2/RESTES_E2_2026-09-08.md`), une seule ligne à accepter (`flagCarries.homeByObject`) |
| P-décodeur E2-bis | Instruire et corriger les pertes que le gate corpus révèle sur l'intégration du lien direct (vague 3, §4) | gate corpus du 2026-09-08, 10 témoins, 8 en PERTE | Opus | re-cuisson locale des 10 films : aucune perte hors les acceptées et les réattributions à total constant | [x] 2026-09-08 : `feat/v2-decodeur-e2` (`453b719fc`, `45c850637`, `f7709ae01` + journal), même worktree, **même bump 50**. **Une affirmation du lot E2 RÉFUTÉE** : le pool de handles REBOUCLE — `084a804d` porte 379 records pour 256 slots, 123 slots à deux corps (`gen=1` puis `gen=2`, index différents), et le refus `lectures_divergentes` y jetait 59 vies en bloc. Le corps est la paire `(slot, génération)` ; une vie revient au corps que le slot portait à son début. Deux P0 de plus, tous deux causés par le GAIN d'identité : la borne d'un épisode d'équipement lisait « une mort ferme cette vie » dans « la vie porte un nom » (camo du slot 620 : 568 → 16 frames), et l'invariant « jamais son propre drapeau » se repliait sur « l'autre drapeau s'il est unique » — faux sur une carte à 6 socles (`unresolved` 0 → 4). Mesure : `084a804d` `non_resolu` **59 → 1** / 353 vies, `unnamedLives` **80 → 0**, `equipmentEpisodes` durée **rétablie à 3676**, `flagCarries.spans` **26 → 29**, véhicules **74 trajets / 28 995 frames conservés** avec `ridesNamed` 52 → 74. Les 13 lignes de perte du gate sont instruites une par une (§E2-bis.1) : 7 réattributions à total constant, 2 lectures fausses corrigées (des bots posés sur des corps d'humains), 3 acceptées confirmées, 1 famille de compteurs d'échec mal classés. 5 mutations jouées rouges, goldens inchangés, tous les gates EXIT 0 (`golangci-lint` 0 issue). **`replay-corpus-gate` complet `[!]`** : à jouer par le SUPERVISEUR (§E2-bis.4 de `.ai/V7.5/v2/RESTES_E2_2026-09-08.md`) |
| P3 | Registre des objets d'objectif (= R3) | v2 lot P | Opus | ouvre après P2 | [ ] |
| P4 | Registre des véhicules et assets | v2 lot P | Opus | ouvre après P3 | [ ] |
| P5 | Clôture du paradigme (gate corpus 0 perte, chronique 49, garde-rail identité, doc FR/EN) + UNE revue adversariale sur le diff cumulé P2-P5 (règle §0.5) | v2 lot P | Opus | `make replay-corpus-gate` | [ ] |
| S.3 | Arrivées/départs (Tactique) — verdict ROSTER de P1 rendu : **NON**, le film ne porte pas les entrées/sorties. S.3 se fera donc par calage `real_start_time` / `t0_quality`, APRÈS P2 | Tactique S.3, D10 | Sonnet, effort moyen | à ouvrir après la clôture de P2 | [ ] OUVRABLE depuis le 2026-09-08 (P2 clos, branche non fusionnée : attendre la fusion dans `feat/v75`) |
| L6 / L7 / L8 | Narratif (gabarits par titre, D7), erreurs API (`code` seul, D6), Discord (langue du compte, D8) | Libellés | Sonnet | — | [ ] |
| R9 | Release : séquence Notion, tag `v7.5.0`, push `main` | v2 R9 | Utilisateur | — | [ ] |

Ordonnancement (2026-09-07, vérifié sur pièces) : la vague 3 s'ouvre alors que Q4 et Q8
(vague 1) et la vague 2 ne sont pas clos — leurs branches et worktrees existent
(`feat/issue-cle-canonique`, `feat/tactique-cloture`, `feat/v2-restes-r6`,
`feat/raster-document-unique`) sans commit de lot à ce jour. **P1 ne produit AUCUN code**,
il ne peut donc entrer en conflit avec aucun lot en vol. P2, qui touche `analysis/replay`
et le collecteur, n'ouvre qu'après la fusion des lots du domaine rejeu (règle §0.6 : un
seul lot en vol par domaine).

### 4.2 Ce que P1 a tranché (2026-09-07)

- **ROSTER (axe d) : NON.** `PlayerActiveInGame` (ti=5 i18) et `PlayerPendingJoinInProgress`
  (ti=5 i19) sont décodés et publiés par la sonde `SetPlayerStateHook`
  (`internal/analysis/filmdec/components_player.go:94,182-190`) mais n'ont **aucun consommateur
  de production**, et leur débit mesuré (163 et 105 lectures sur 22 films, soit ~7 et ~5 par film
  pour 8 joueurs) ne tient pas le dénominateur. Conséquence : **S.3 par calage `real_start_time` /
  `t0_quality`, après P2** — D10 confirmée, pas infirmée.
- **Le trou central est E2, le slot de bipède** : zéro lien direct, cinq replis empilés (pont par
  morts, fermetures, siège de bot, relais, nommage final) ; E5 (objets d'objectif), E7 (véhicules)
  et E8 (armes au sol, équipements) en dépendent pour nommer porteur, occupant et ramasseur.
  C'est ce qui ORDONNE P2 avant P3, P4 et P5.
- **TEMPS (a) et MANCHES (b) : verdicts négatifs.** ti=0 n'est pas répliqué (1 enregistrement sur
  22 films). La seule horloge directe est le couple `Packet.TS` + `start_ms` du manifeste, déjà lu
  par le statborg (`internal/analysis/objectiveevents/statborg.go:199`) et JAMAIS par la grille de
  frames, qui reste ancrée sur le premier paquet de position (`internal/analysis/replay/build.go:49`).
- **(g) PROVENANCE** : elle existe déjà en cinq exemplaires incompatibles ; forme unique proposée
  `Link{Source, Method, Readings, Metric, From, To}`, bornes temporelles obligatoires.
- **(i)** un bump d'`IsolationDecoderRev` à P2 impose la réécriture de `match_lives` /
  `match_death_context` par `levelup backfill-killsource` (le journal des morts n'est pas touché) :
  à budgéter DANS P2, pas après.
- **(j)** l'amendement du §0.7 du plan v2 (registre = fonction pure de `internal/analysis/replay`,
  appelée par `replaybuild` ET par le collecteur) est rédigé mot pour mot dans le livrable :
  à porter au premier commit de P2.

Découvertes consignées au registre, NON traitées : (1) doc inversée
`internal/analysis/replay/vehicle_rides.go:29-31` — l'événement NOMME le véhicule depuis le lot V8
(`internal/analysis/filmdec/event_list.go:317-320`), à corriger en P4 ; (2) plomberie ti=0
(`SetGameEngineHook`, `RoundTimerOf`) sans consommateur ni porteur — ne rien supprimer avant le plan
décodeur ; (3) `bid(N.0)` lu mais non publié, d'où la jointure web des bots par nom nu
(`rosterLogic.ts:122-128`) — gain le moins cher du lot P, à fermer en P2.

Contrôle du superviseur : vérifiés SUR PIÈCES le 2026-09-07 — les deux composants de roster sans
consommateur de production, l'ancrage de la grille de frames sur le premier paquet, l'horloge du
statborg, la contradiction de `vehicle_rides.go`, et l'absence totale de fichier de code dans le
diff (`git diff --name-only feat/v75..feat/p1-inventaire` : 4 fichiers, tous sous `.ai/`). Les
débits de lecture et les décomptes de liens viennent du balayage du lot, non re-mesurés.

- **D12 (2026-09-08, utilisateur)** : le lien slot de bipède ↔ index de joueur est un BLOQUANT DE LA
  RELEASE v7.5.0 (« sinon on a un truc bricolé et pas fiable/stable »). Le gel du décodeur (§0.6 du
  plan v2) reste, ce seul item en est exempté : sondage borné de rétro-ingénierie (Opus, diagnostic
  seul, 2-3 films) puis branchement additif sous corpus gate, à lancer dès que le quota le permet
  (après P3-P4, ou en parallèle si budget). Entrée de registre datée ; critère de succès
  `identity.coverage.bipedSlot.direct`. Vague 3 devient : P3, P4, **P-décodeur (E2)**, P5.
- **D13 (2026-09-08, utilisateur, « go avec ta reco »)** : RÉORDONNANCEMENT — P2-bis (résidu) → **sondage
  décodeur E2 AVANT P3** (Opus, diagnostic seul, borné : le film porte-t-il le propriétaire d un corps
  et d un objet, et où ; critère `bipedSlot.direct`) → P3 et P4 ALLÉGÉS (registre, machine à états
  unique, migration des lecteurs ; ZÉRO heuristique nouvelle : un lien que le film ne donne pas est
  publié `non_resolu` et compté, jamais deviné par géométrie) → P5. Rien n est abandonné : les
  heuristiques de déduction prévues par P3/P4 sont retirées de leur périmètre, pas le reste.
- **Sondage E2 [x] 2026-09-08** (`feat/v2-sondage-e2`, `eaa35ba06`, Opus, diagnostic seul) : le record de
  CRÉATION d une entité bipède `ti=35` porte l index de participant du propriétaire (`+67` R(5),
  primitive `ECS_ReadEntityRefIndex5`, déjà consommée puis jetée par le décodeur). Couverture directe
  mesurée : `d9781168` 90,3 %, `bf15f7ab` 86,8 %, `64e8adfa` 95,0 % (372/408 vies), 0 fantôme ;
  contre le pont par morts : 329 concordants, 18 discordants dont 7 paires ÉCHANGÉES (le pont nomme
  à tort), 39 vies neuves. Spécification et plan d intégration additif I1-I5 (~420 L) dans
  `.ai/V7.5/film_re/SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md`. Préalable : trancher le cas des bots
  (index lu 8 hors table sur `c75f33b8`) — doctrine : index hors table = `non_resolu` + compteur,
  jamais inventé ; instruction dans le lot d intégration. Lot suivant : **P-décodeur E2 (I1-I5)**.
- **D14 (2026-09-08, utilisateur)** : PAS DE REPLI par les morts pour les vies sans record de création lu.
  Le pont par morts devient vérification seule ; les 5-13 % restants sont INSTRUITS vie par vie et
  classés par cause : (a) vie découpée par un trou de réplication → lien PROPAGÉ par (slot, génération)
  (même record, source directe) ; (b) record présent mais non lu → améliorer le lecteur ; (c) bot / index
  hors table → `non_resolu` ; (d) record réellement absent (prouvé) → `non_resolu` compté, classe qui
  doit rester petite. Objectif : `bipedSlot.direct` ≥ 95 %, `non_resolu` entièrement ventilé, 0 vie
  nommée par le pont. Exécutant relancé avec ce brief (I0 bots déjà tranché : espace d index partagé).

## 5. Décisions qui appartiennent à l'utilisateur (avec recommandation ; à trancher avant le lot concerné)

| # | Question | Bloque | Recommandation |
|---|---|---|---|
| D1 | Supprimer `feat/outcome-cle-canonique` et `feat/v2-audit-vies` (et origin) | Q1 | Oui : rien d'unique dedans (§1.1). |
| D2 | Fusionner `wt/blob-304-retry` dans `feat/v75` maintenant | Q2 | Oui : complet, revu, sans conflit de code ; la CI ne l'a jamais vu, c'est la seule raison de le faire vite. |
| D3 | Ménage des ~95 branches fusionnées et de leurs worktrees | Q1 | Oui, liste soumise avant suppression ; `-d` seulement. |
| D4 | `HomeMatchRow.OutcomeLabel` et autres `*_label` sans lecteur web : supprimer ou garder pour des clients tiers | Q4 | Supprimer (règle 7 « 0 code mort ») ; git garde l'historique. |
| D5 | Cible par famille de libellés : clé canonique (option 1) ou libellé serveur localisé (option 2) | M5 | **TRANCHÉ 2026-09-07 : option 1 partout** (utilisateur : « la clé canonique est ce qu'il y a de plus solide, propre et pérenne »). Le Go sert des clés, le web localise depuis les TOML du titre. L'option 2 transitoire de l'issue est abandonnée (Q4 reformulé). |
| D6 | Erreurs API (L7) : localiser côté web par `code` ; descriptions OpenAPI en FR acceptables ? | L7 | **TRANCHÉ 2026-09-07 (ok utilisateur)** : le Go ne sert QUE `code` (+ `detail` technique en anglais pour les logs, jamais affiché) ; table `code → texte` FR/EN côté web ; garde-rail « une erreur sans code est refusée ». Descriptions OpenAPI : gardées en FR (documentation, jamais vue par un utilisateur final). Cohérent avec D5 : une erreur est une clé, comme une issue. |
| D7 | Narratif / synthèse / citations (L6) : libellés ou contenu ? | L6 | **TRANCHÉ 2026-09-07 (ok utilisateur)** : ce sont des PHRASES (contenu), pas des libellés → ADR 0028 : gabarits FR/EN par titre dans `config/titles/{slug}/templates/`, le Go ne choisit qu'une clé de gabarit et remplit les variables. Même principe que D5 (clé côté Go, texte côté données), ce qui permet à un autre titre d'apporter ses propres phrases. Hors plan libellés, sauf le mojibake (Q3). |
| D8 | Discord (L8) : langue du destinataire, du compte ou de l'instance ? | L8 | **TRANCHÉ 2026-09-07** : langue du compte propriétaire de la notification ; repli langue de l'instance. |
| D9 | Périmètre du mojibake : Go seul ou aussi `docs/`, `.ai/`, TOML ? | Q3 | **TRANCHÉ 2026-09-07** : Go + TOML en Q3 ; `docs/` et `.ai/` en lot séparé bas effort si souhaité. |
| D10 | S.3 gelé derrière P1 (§1.2) | S.3 | **TRANCHÉ 2026-09-07 : oui.** |
| D11 | Amendement du plan v2 §0.7 : registre calculé dans `analysis/replay` (pur), consommé par `replaybuild` ET le collecteur | P1 | **TRANCHÉ 2026-09-07 : oui.** |

D1 à D4 tranchées le 2026-09-07 (oui ; D3 : uniquement les worktrees PROPRES des branches fusionnées,
dans `Downloads/Scripts` ET sous `.claude/worktrees/` ; `LevelUp-go-migration` porte les données non
suivies et n'est jamais touché ; suppression locale seulement, `origin` gardé sauf pour D1).

## 6. Découvertes de cette planification (consignées, non traitées)

- 2026-09-07 ; `service/*` ; 8 appelants du repli `outcomeLabel(` au lieu des 5 du plan
  libellés (`match_view_builders_header.go:194` et `match_history_service_enrich.go:134` en
  plus). Reprise : Q4.
- 2026-09-07 ; `apps/web/src/features/match-replay/layers/heatmapLayer.ts` ; le peintre de
  chaleur du lot D n'est pas dans `lib/replay/` contrairement à la prémisse de S.2. Reprise : Q7.
- 2026-09-07 ; `analysis/replay/death_context.go:73,185` ; `EtatParti` n'est plus produit
  depuis 7C.9 (énumération morte). Reprise : Q8.
- 2026-09-07 ; `.ai/V7.5/PLAN_TACTIQUE_2026-09-06.md` phase 8 ; cases 8.1-8.4 vides, 8.4 (revue du
  diff intégral) jamais faite bien que le commit dise « clôture ». Reprise : Q8.
- 2026-09-07 ; `DECOUVERTES_TACTIQUE`, entrée « `LEFT JOIN match_registry` sans test » ; caduque
  depuis 7C.9 (le JOIN a disparu avec la lecture des instants). Reprise : Q8 (fermer).
- 2026-09-07 ; `.ai/V7.5/REGISTRE_REPORTS.md:578` ; le commit `6c960861c` (phase 5 sur
  `feat/v75`, ESLint rouge) est réputé corrigé par `bf5d465bd` : seule la CI de `f2c8ddce1`
  le prouvera. Reprise : Q8 (d).
- 2026-09-07 ; `killcollector/positions.go:258-266` ; lecteur du pont par morts hors rejeu, à
  migrer au registre P2. Reprise : P1 axe (i).
- 2026-09-07 ; `.ai/PLAN_V2_RESTES_2026-09-07.md` §0.7 ; « calculée une fois dans `replaybuild` »
  incompatible avec le principe 7C (faits complets au sync). Reprise : D11 / P1.
- 2026-09-07 ; `.ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md` §1 ; 16 littéraux mojibake au HEAD
  (13 annoncés). Reprise : Q3.

- 2026-09-08 ; lot M1 (`feat/tactique-lien-rejeu`) ; `instant_ms` des questions `morts`/`kills`/
  `gagne`/`isole` est sur l'horloge du MATCH (`match_kill_events.time_ms`), le lecteur du rejeu
  consomme l'horloge du FILM (`horlogeFilm = horlogeMatch + DeathOffsetMS`, décalage par match de
  3,6 à 50,8 s, mesuré à la cuisson mais NON publié : `coverage.bridge` ne porte que la marge) ;
  seules `temps`/`routes` (sidecar) donnent une frame exacte. P1 de fonctionnalité (lien qui
  ouvre le rejeu au mauvais instant). Reprise = lot **M1b** : publier `deathOffsetMs` dans
  `coverage.bridge` du document de rejeu (additif, bump 49 + chronique + goldens + recuisson du
  parc), le web convertit `?t=` (horloge match) en frame avec le document chargé ; en l'absence
  d'offset (artefact ancien) le lien ne s'affiche pas pour ces quatre questions. À faire AVANT
  la fusion de la vague 2 ou consigné comme réserve visible à l'écran.

- 2026-09-08 ; lot P2 (`feat/v2-p2-registre-joueurs`) ; `roster[].bid` est publié côté Go et au
  contrat, mais `apps/web/src/lib/replay/rosterLogic.ts:100,122-128` joint toujours les bots par le
  NOM NU (`botKey(entry.name)`) — deux bots homonymes fusionnent toujours. Le brief de P2 fige le
  web. Reprise : lot web dédié, APRÈS la re-cuisson du parc au schéma 50 (R9), pour que les
  artefacts servis portent le champ.
- 2026-09-08 ; lot P2 ; les vies nommées par FERMETURE (`closures.go`) ne sont pas marquées
  « déduites » dans `unnamed.deduced` — une fermeture est pourtant une déduction, et les lecteurs
  qui prouvent une ABSENCE (`carrierPresenceOf`) ne s'en abstiennent pas. Préexistant, non aggravé.
  Reprise : P5, avec le garde-rail global d'identité.

## 7. Reprise de session

Lire ce plan, puis le document source du lot en cours ; `git branch --show-current` ;
`git log --oneline -5` ; reprendre à la première case non statuée de la vague en cours.
Un lot n'ouvre pas tant que le précédent DU MÊME DOMAINE n'est pas fusionné dans `feat/v75`.
Worktree d'orchestration : `LevelUp-wt-orchestration` (branche `wt/orchestration-0907`,
documents seulement).
