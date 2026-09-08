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
| Q1 | Ménage des branches : suppression de `feat/outcome-cle-canonique`, `feat/v2-audit-vies` (locales + origin) et de leurs worktrees ; liste des ~95 branches fusionnées présentée puis `git branch -d` | §1.1 | Superviseur | `git branch --no-merged feat/v75` ne perd rien ; `git worktree prune` | [ ] |
| Q2 | Fusion `wt/blob-304-retry` → `feat/v75` : worktree dédié, `git merge`, deux conflits doc résolus « les deux côtés », `go test ./internal/sync/haloclient/ ./internal/sync/ -run 'Blob\|Pooled\|TextPredicate'`, lint new-from-merge-base, push, CI | §1.1 | Superviseur (accord utilisateur avant commit) | tests haloclient verts, lint ; entrée thought_log (CI : fin de vague) | [ ] |
| Q3 | Mojibake : réencoder les 45 fichiers (16 littéraux d'abord, vérif à l'octet), garde-rail `archlint/no_mojibake_test.go` (allowlist vide), cause écrite dans le test | Libellés L0 | Sonnet, effort moyen | `go build ./...`, `go vet`, tests des paquets touchés, garde vert ; `git diff --stat` = uniquement des octets `C3 83…` → caractères | [x] fait — 62 fichiers re-vérifiés sur pièces (45 estimés hors tests + tests inclus par D9), garde-rail posé et mutation-testé ; branche `feat/mojibake-garde-rail`, non fusionnée (voir `.ai/thought_log.md` 2026-09-07 « lot Q3 ») |
| Q4 | Issue de match, fin de L1 **par la CLÉ CANONIQUE (décision D5, 2026-09-07)** : chaque DTO qui portait `outcome_label` porte `outcome` = clé `win|loss|tie|dnf` (`mappings.Canonical(rawCode)`) ; les 8 appelants du repli et `resolveOutcomeLabel` disparaissent avec la map FR et son kill-switch ; `HomeMatchRow.OutcomeLabel` + maps de `home_locale.go` supprimés (0 lecteur, D4) ; `duelOutcomeLabel` (Explorer) : clé `won|lost` + libellé web, même mécanique ; côté web les lecteurs de `outcome_label` (Match View `MatchHeader.card.tsx:362`, historique, Explorer, export CSV, Carrière) passent par `useOutcomeLabel`/`useOutcomeMapping` ; l'export CSV, qui a besoin d'un texte, le rend côté web ; `openapi.yaml` + `make generate-types` ; ratchet FINAL `no_french_label_literal_test.go` posé avec le compte du jour | Libellés L1 + garde-rail final (option 1) | Sonnet, effort élevé (contrat + ~20 lecteurs web) | `go test ./internal/service/... ./internal/analysis/...`, `openapi-gen -check`, `make generate-types` + diff vide ou commité, vitest des consommateurs web, parité FR/EN à l'écran (Match View, Explorer, Carrière) par l'utilisateur | [ ] |
| Q5 | Flakes CI : `TestStartImport_HappyPathReturns202WithJobID` (attendre la fin du job / fermer le store) ; `TestWorker_Run_PersistsAndACKs` (remplacer l'attente implicite par une synchronisation) | v2 R8 | Sonnet, effort bas | `go test -count=20 -p 8 -run <Nom>` vert, deux fois | [ ] |
| Q6 | Dette mécanique du rejeu : scissions `document.go` / `usage_summary.go` / `replaybuild.go` / `flag_carries_test.go`, commentaire faux `equipment_episodes_test.go:371-372` ; 0 bump | v2 R0 | Sonnet, effort bas | goldens INCHANGÉS (`git diff --exit-code testdata/`), preuve de déplacement pur (concaténation triée avant/après identique), lint 0 | [x] fait — 4 scissions + 1 commentaire corrigé, preuves de déplacement pur collées, gates verts (build/vet/test/lint) ; `document.go`/`usage_summary.go`/`replaybuild.go` restent > 500 L après l'extraction unique prescrite (statué `[!]`, registre) ; branche `feat/v2-restes-r0`, non fusionnée (voir `.ai/thought_log.md` 2026-09-07 « lot Q6 = R0 ») |
| Q7 | Un seul peintre de chaleur : noyau `buildHeatmap`/`drawHeatmapLayer` déplacé de `features/match-replay/layers/heatmapLayer.ts` vers `lib/replay/heatPaint.ts` (entrée « cellules pré-agrégées » ajoutée LÀ), `features/tactical/heatPaint.ts` supprimé, garde-rail grep (aucune seconde implémentation hors `lib/replay/`) | Tactique S.2 (prémisse corrigée §1.3) | Sonnet, effort moyen | `tacticalGridFromRaster` + snapshot léger identiques, typecheck, vitest complet, `crossFeatureBoundary.guard` vert, garde-rail vert | [ ] |
| Q8 | Finition Tactique (clôture réelle de la phase 8) : (a) les trois constats de la revue (§1.2) : réserve `echantillon_faible` rendue sur les tuiles KPI Tactique et sur le nuage Escouade (avec test), quatre clés i18n mortes retirées ; (b) `EtatParti` mort retiré ; (c) registre Tactique, entrées « petites » : regex `no_raw_rating_reads_test.go:44` (3 tables), WARN + compteur sur le refus de positions (`positions.go:103`), WARN sur `settingsStore.Load()` en erreur (cron de purge silencieux — 2 appelants), compteur dédié pont non publiable (`isolation_facts.go:176`), test d'orthogonalité non tautologique (`lives_export_test.go:180-193`), test film réel réparé (roster depuis `ScanDeaths`, `t.Fatal` si 0 vie) ; entrée « LEFT JOIN sans test » fermée comme caduque ; (d) cases 8.1-8.4 du plan Tactique statuées ; (e) `matchs_sans_rayon` : note de couverture sur la tuile « Morts en isolement » (contrat déjà publié, 0 SQL) | Découvertes Tactique + plan Tactique phase 8 | Sonnet, effort moyen ; le test film réel se joue SUR LE POSTE PRINCIPAL (`KILLSOURCE_FIXTURES`) par le superviseur | `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...`, `no_art_patterns_test`, typecheck + vitest, `make gate-push` | [ ] |

Non retenus en vague 1 (consignés, pas oubliés) : canvas tactique à `k = 1` sans `devicePixelRatio`
(choix assumé) ; double lecture des artefacts par cycle dans `projeterRastersTactiques`
(optimisation, à regrouper avec « le document qui circule entre projections » — vague 2) ;
test `:memory:` dédié à `QTacticalIsolement` (entre dans Q8 si la revue le juge P1, sinon vague 2).

## 3. Vague 2 — finitions moyennes (après vague 1 ; un lot par domaine en vol)

| # | Lot | Source | Exécutant | Gate |
|---|---|---|---|---|
| M1 | Lien « voir dans le rejeu » depuis une cellule : contrat `POST …/tactical/{map_id}/cellule` (contributions `{match_id, instant_ms, xuid}` filtrées par ownership ADR 0029 + `matchs_non_ouvrables`), service, handler, `TacticalCellCard` avec lien `?frame=` (`playbackStore`), instant → frame en logique pure | Tactique S.1 (5.5 + 5.6 complet) | Opus effort bas pour le contrat + Sonnet pour le web | test « un match d'un autre joueur n'apparaît pas mais compte », `?frame=` positionne le rejeu, contrat régénéré, typecheck, vitest |
| M2 | Constats P2 de l'audit des vies (5) + `drawnSwapAt` borné à la vie courante côté web | v2 R6 | Sonnet, effort moyen | mutation par constat, vitest, gates v2 — **[x] FAIT le 2026-09-07** (branche `feat/v2-restes-r6`, non fusionnée) : P2-1/P2-2 confirmés, correctif hors périmètre (racine `filmdec`, décodeur gelé, diagnostic + registre) ; P2-3 non-lieu (déjà corrigé par `f1b4f4ee5`) ; P2-4/P2-5 corrigés avec mutation ; `drawnSwapAt` (en réalité `features/match-replay/model/equippedLogic.ts`, pas `lib/replay/`) borné via `currentLifeOf`/`trackWindow`, mutation jouée. 0 bump (aucun correctif ne change le contenu cuit). Détail : `.ai/V7.5/v2/RESTES_R6_2026-09-07.md` |
| M3 | Budget de candidats du calage (adaptatif ou dédoublonnage par fin de vie) ; test de documentation RETOURNÉ ; bump 49 partagé avec le lot suivant qui bumpe | v2 R7 | Opus, effort moyen | `d9781168` / `51ebbc0f` identiques hors numéro — [x] 2026-09-07 : `feat/v2-restes-r7` (worktree `LevelUp-wt-m3-calage`, base R6, poussée) ; **dédoublonnage par FIN DE VIE** retenu (le vote comptait les morts appariables quand l'affinage apparie 1:1 : un amas de 20 morts déposait 20 voix sur chaque fin isolée pour 1 paire réalisable) — la voix d'un panier devient `min(morts, fins)`, borne de Hall/König, **aucun seuil nouveau** ; budget adaptatif écarté (classement resté faux + 15 affinages inutiles). Fixture adversariale `off=216350 n=2` → `off=200000 n=15 second=2`, alarme éteinte ; test retourné en mutation `TestUnAmasPlusGrosQueLeVraiCalageNEmportePasLeBudget` (teste `voteDeathOffsets` seul), baseline inchangée (le test date du 07/09, la baseline du 26/06). **AUCUN golden ne bouge → PAS de bump, `SchemaVersion` reste 48**, le 49 reste pour P2. ~~`make replay-corpus-gate` et les témoins `d9781168`/`51ebbc0f` restent À JOUER PAR LE SUPERVISEUR (worktree sans film)~~ **JOUÉS le 2026-09-08 dans le lot M4** : gate mode base `0 perte sur 7/7`, `d9781168` et `51ebbc0f` identiques À L'OCTET entre base et HEAD ; unique effet du correctif sur tout le corpus : `bf15f7ab` `deathOffsetRunnerUp` 12 → 13 (calage retenu inchangé). Réserve : leurs marges absolues ne valent plus 157:15 et 71:8 mais 143:18 et 71:10 — les FILMS ont été re-téléchargés, pas le code (attribution par une cuisson à `ee4084c14`) |
| M4 | Diagnostics : écart K/D/A sur `51ebbc0f` (R4) ; re-vérification CTF multi-manche + VIP/crâne (R5) | v2 R4, R5 | Opus, effort moyen (diagnostic = jugement) | journal chiffré par joueur et par manche ; verdict source vs lecteur ; registre fermé ou rouvert avec chiffres — **[x] FAIT le 2026-09-08** (worktree `LevelUp-wt-m4-diag`, branche `feat/v2-restes-r4r5-diag`, aucun fichier de code modifié). **VERDICT R4 : LECTEUR** — l'écart vaut **9** (et non 69 : les films témoins ont été re-téléchargés le 08/09, attribution faite par une cuisson à `ee4084c14` identique à l'octet à celle du HEAD), 7 joueurs sur 8 EXACTS, tout l'écart sur un joueur dont la manche 0 n'est pas publiée parce qu'il n'y meurt **pas une seule fois** : `bestDeathClaim` exige 3 morts coïncidentes (`deathInstantMin`) et `CompletedByLines`, le rattrapage écrit pour ce trou, est gardé MONO-MANCHE. Généralité : 8 couples (xuid, manche) perdus sur 3 films, toujours avec 0 ou 1 mort — aucun contre-exemple ; le même défaut fait perdre une capture de drapeau sur `64e8adfa`. Correctif **REPORTÉ** au lot suivant (forme, tests par mutation, témoins et portée chiffrés au journal §4 ; bump requis, le 49 est pris). **R5** : `--reference=parc` DÉGÉNÉRÉ (7/7 témoins absents du parc) → mode d'autorité joué, `--base feat/v2-restes-r6` **0 perte sur 7/7** (1 gain `bf15f7ab` = le seul effet du correctif R7, ce qui CLÔT son item superviseur) ; crâne 36 et 19 portages, 0 anonyme, 0 identité inventée, mais 6/36 portages fantômes toujours ouverts ; **aucun film VIP au parc** sur 466 recensés. Détail : `.ai/V7.5/v2/RESTES_R4_R5_2026-09-08.md` |
| M5 | Libellés L2 (accueil), L3 (modes/playlists → `assets.toml`), L4 (armes), L5 (rangs → `mappings/ranks.go`) | Libellés | Sonnet, effort moyen, un lot par famille | parité FR/EN à l'écran, ratchet qui baisse, `no_slug_comparison_test` |
| M6 | Le document de rejeu circule entre projections : `ProjeterRasterTactique` réutilise `a.doc` au lieu de relire le fichier | Découvertes Tactique (fusion v75) | Sonnet, effort bas | durée d'un cycle `Deriver` mesurée avant/après, goldens inchangés |

## 4. Vague 3 — le paradigme (lot P du plan v2) et ce qu'il débloque

- **P1 — Inventaire** (journal seul, AUCUN code) : Opus, effort élevé. Ajouter aux axes (a)-(h)
  du plan v2 : (i) les lecteurs HORS rejeu à migrer — `killcollector/positions.go`,
  `isolation_facts.go`, tables `match_lives` / `match_death_context` (§1.2) ; (j) l'emplacement
  du registre = fonction pure de `analysis/replay` (§1.2). P1 rend un VERDICT écrit pour S.3
  (le film porte-t-il les entrées/sorties ? sinon S.3 se fait par calage `real_start_time` /
  `t0_quality`, comme prévu, mais APRÈS P2).
- **P2 → P5** : Opus, une phase = un lot clos (gate + journal), UNE revue adversariale à la
  clôture de P5 sur le diff cumulé (règle §0.5), bump 49 unique.
  R1, R2, R3 ne s'exécutent pas séparément.
- **S.3** (Tactique) : après P1 selon son verdict, avec le calage prescrit par P1.
- **L6, L7, L8** (narratif, erreurs API, Discord) : après décisions §5.
- **R9** : release, séquence Notion, main de l'utilisateur.

## 5. Décisions qui appartiennent à l'utilisateur (avec recommandation ; à trancher avant le lot concerné)

| # | Question | Bloque | Recommandation |
|---|---|---|---|
| D1 | Supprimer `feat/outcome-cle-canonique` et `feat/v2-audit-vies` (et origin) | Q1 | Oui : rien d'unique dedans (§1.1). |
| D2 | Fusionner `wt/blob-304-retry` dans `feat/v75` maintenant | Q2 | Oui : complet, revu, sans conflit de code ; la CI ne l'a jamais vu, c'est la seule raison de le faire vite. |
| D3 | Ménage des ~95 branches fusionnées et de leurs worktrees | Q1 | Oui, liste soumise avant suppression ; `-d` seulement. |
| D4 | `HomeMatchRow.OutcomeLabel` et autres `*_label` sans lecteur web : supprimer ou garder pour des clients tiers | Q4 | Supprimer (règle 7 « 0 code mort ») ; git garde l'historique. |
| D5 | Cible par famille de libellés : clé canonique (option 1) ou libellé serveur localisé (option 2) | M5 | **TRANCHÉ 2026-09-07 : option 1 partout** (utilisateur : « la clé canonique est ce qu'il y a de plus solide, propre et pérenne »). Le Go sert des clés, le web localise depuis les TOML du titre. L'option 2 transitoire de l'issue est abandonnée (Q4 reformulé). |
| D6 | Erreurs API (L7) : localiser côté web par `code` ; descriptions OpenAPI en FR acceptables ? | L7 | Reco ferme (à confirmer avant L7) : le Go ne sert QUE `code` (+ `detail` technique en anglais pour les logs, jamais affiché) ; table `code → texte` FR/EN côté web ; garde-rail « une erreur sans code est refusée ». Descriptions OpenAPI : gardées en FR (documentation, jamais vue par un utilisateur final). Cohérent avec D5 : une erreur est une clé, comme une issue. |
| D7 | Narratif / synthèse / citations (L6) : libellés ou contenu ? | L6 | Reco ferme (à confirmer avant L6) : ce sont des PHRASES (contenu), pas des libellés → ADR 0028 : gabarits FR/EN par titre dans `config/titles/{slug}/templates/`, le Go ne choisit qu'une clé de gabarit et remplit les variables. Même principe que D5 (clé côté Go, texte côté données), ce qui permet à un autre titre d'apporter ses propres phrases. Hors plan libellés, sauf le mojibake (Q3). |
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
- 2026-09-07 ; `.ai/PLAN_TACTIQUE_2026-09-06.md` phase 8 ; cases 8.1-8.4 vides, 8.4 (revue du
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

## 7. Reprise de session

Lire ce plan, puis le document source du lot en cours ; `git branch --show-current` ;
`git log --oneline -5` ; reprendre à la première case non statuée de la vague en cours.
Un lot n'ouvre pas tant que le précédent DU MÊME DOMAINE n'est pas fusionné dans `feat/v75`.
Worktree d'orchestration : `LevelUp-wt-orchestration` (branche `wt/orchestration-0907`,
documents seulement).
