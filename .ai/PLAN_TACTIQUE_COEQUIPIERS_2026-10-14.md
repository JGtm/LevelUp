# Plan — Tactique : coéquipiers à la mort (2026-10-14)

Statut : VALIDÉ par le user le 2026-10-14 (« Ok avec toi [...] Je valide »). D1 = option A. D2 = même garde que le rejeu ; le user activera le rejeu en production à la prochaine release : la route de séquence DOIT donc suivre exactement le même interrupteur que le rejeu (même groupe de routes / même garde), pour qu'activer le rejeu active la vue D sans autre changement.

- **A6 (base)** : pour ne pas attendre la fusion, la branche part de `feat/tactique-tueurs` (c5debdf5f). L0.1 devient : vérifier `AutrePoint` présent ; `instant_affiche_ms` et le correctif du lien arrivent par une refusion de `feat/tactique-tueurs` AVANT L2.5 (bloquant pour L2.5 et L3, pas pour L1).
Maquette validée : https://claude.ai/artifact/UYTwfMq8w9CbN2sQo1psYg (A instantané, B échange, C zone, D séquence).
Décisions user du 14/10 : délai d'échange 3 s ; terme « tueur abattu » ; rejeu ouvert 5 s avant ; miroir frags « abattu en retour en 3 s » ; horloge affichée = celle du rejeu (coup d'envoi, `t0FilmMs` puis T0 API, plancher 0:00).

## Amendements du superviseur au plan du planificateur (prévalent sur le corps)

- **A1 (corrige la décision 12 du planificateur)** : la vue d'une mort (A+B+D) se peint SUR LE PLAN PRINCIPAL de la carte (fond de carte, même projection et même cadrage que la chaleur, toile superposée comme la couche tueurs/victimes), pas dans un encadré SVG autonome. C'est ce que montre la maquette validée. L'encadré « Instant de la mort » de la colonne droite porte le texte (tueur, coéquipiers et distances, issue de l'échange, étapes D, bouton de rejeu).
- **A2 (D3 tranchée par la maquette)** : avant la mort le cercle suit la victime ; à partir de l'instant 0 il reste au point de mort. Conforme au corps.
- **A3** : L0.1 attend la fusion de `feat/tactique-tueurs` (avec le correctif du lien et `instant_affiche_ms`) dans `feat/v75` ; le contrat réel est relu sur pièces en L0.3, y compris si `instant_affiche_ms` a pris une autre forme.
- **A4** : L3.9 « enquête sur le lien de rejeu » est COUVERT par le lot tueurs (`[~]` vers son commit) : cause = conversion par `deathOffsetMs` au lieu de `t − originMs` ; corrigé avec l'ouverture 5 s avant et l'instant affiché sur l'horloge du rejeu.
- **A5** : exécuteur Sonnet par défaut, worktree dédié `LevelUp-wt-tactique-coequipiers`, branche `feat/tactique-coequipiers` ; commit et push de branche par le superviseur à chaque lot ; fusion dans `feat/v75` et backfill sur accord explicite du user.

## Décisions user ouvertes

- **D1 bots** : le substrat positionnel (`kill_positions`, contexte de mort, équipes indexées par xuid) n'a pas les bots. Recommandation : option A (ce lot garde le substrat ; les bots comptent là où la source les a — « tueur abattu » / « abattu en retour » par le journal des kills ; report daté d'un lot « bots dans les positions »).
- **D2 vue D en production** : la séquence lit les trajectoires du rejeu, qui n'est servi qu'en local en production (décision du 2026-07-28, réexamen au plus tard le 2026-11-08). Recommandation : même garde, la vue D n'apparaît que là où le rejeu est servi.

## Corps du plan (planificateur, Sonnet)

### Constat sur pièces (planificateur)
- `match_death_context` porte `nearest_teammate_m`, `teammates_visible/waiting/out_of_sight/left(=0)/total` ; aucune position de coéquipier persistée. `replay.ContextesDesMorts` calcule état + position par coéquipier puis les jette dans `contexteDUneMort` ; itération sur map (`e.Equipes`) → trier avant de persister.
- Bots : `Equipes` indexé par xuid décimal (`equipesNumeriques`) ; `journalDesMorts` écarte les victimes sans xuid ; persister `kill_positions` refuse un tueur sans xuid.
- Angles : Morts=`morts`, Morts seul=`isole`, Frags=`kills`, Solde=`solde`. Rayon par match `mappings.PorteesDuRadarParMatch`, borne inclusive `APortee` (ratchet `archlint/no_local_portee_comparison_test.go`). Appariement mort/contexte ±1,5 s (`tactical.ContexteLePlusProche`) : lire les coéquipiers sur la clé exacte de la ligne appariée.
- Aucun algo d'échange existant → `analysis/coordination/`. `match_kill_events_latest` : `victim_xuid` NULL pour un bot ; tueur bot = `feed_killer_gamertag` sans xuid.
- Rejeu servi en local seulement (`LocalOnlyReplay`, réexamen ≤ 2026-11-08). Artefact JSON par match, pas 100 ms, `Track`/`Point{t,x,y,g}`/`ReplayDocument` déjà cités hors `film/`.
- `backfill-killsource` accepte `--match` ; un bump de `IsolationDecoderRev` rend le parc éligible via `matchsAJour`.

### Décisions tranchées (planificateur)
1. Nouvelle table append-only `match_death_mates` (une ligne par mort et par coéquipier), même transaction / `decode_pass` que `match_lives` et `match_death_context`.
2. Bump `IsolationDecoderRev` (pas de révision séparée).
3. Aucune capability neuve : `film.kill_positions` pour coéquipiers ; vue D sous `film.replay_artifact` + capability `replay`.
4. `FenetreRiposteMs = 3000`, bornes [0, 3000] incluses ; plusieurs vengeurs : le plus tôt puis gamertag.
5. « Tueur abattu » : un coéquipier de la victime tue le tueur ; identité du tueur par xuid sinon gamertag.
6. « Abattu en retour » (face frag) : joueur tué dans les 3 s après son frag, par n'importe qui (`feed_killer_gamertag`).
7. Règles de portée côté Go ; le web lit `statut` / `dans_le_rayon`. Comptes N sur M et médianes, jamais de taux nu.
8. Rose : 8 secteurs de 45°, repère plan (haut = +Y, sens horaire), intervalles `[ , )`.
9. Vue D : lecture à la demande d'une piste, décodage partiel (`tracks`, `frameIntervalMs`, `originMs`, `t0FilmMs`), fenêtre [−8 s, +3 s] ré-échantillonnée 200 ms, sans cache ; mesure sur le plus gros artefact consignée ; > 500 ms = `[!]` au user. Acteurs depuis la base (`match_death_mates`, journal), jamais du `team` du film.
10. Une seule conversion match → film → affiché (celle du lot tueurs), extraite si privée.
11. Jetons : `team-ally` coéquipiers, `stat-deaths` tueur, `stat-kills` victime.
12. REMPLACÉE par A1.

### L0 Prérequis (bloquant)
- [x] L0.1 (réduit par A6) `AutrePoint` présent dans `domain/tactical_cellule.go` l.132 (vérifié le 2026-10-14) ; `instant_affiche_ms` et le correctif du lien restent à refusionner avant L2.5. Texte d origine : fusionnée dans `feat/v75` (grep `AutrePoint` / `instant_affiche_ms` dans `domain/tactical_cellule.go`) ; sinon `[!]` et arrêt.
- [x] L0.2 Branche `feat/tactique-coequipiers` depuis `feat/v75`, worktree dédié. — fait par le superviseur (HEAD 95dee81c4).
- [x] L0.3 Rouvrir : `death_context.go` (l.175-215), `isolation_facts.go`, `lives_persister.go`, `tactical_service_cellule_enrichir.go` (`enrichir`, `poserPlacements`), conversion `instant_affiche_ms`, ADR 0026 / 0034 / 0036 (I2, I4, I6). — relu le 2026-10-14 : death_context.go, isolation_facts.go, lives_persister.go, order.go, compaction_registry.go, garde-rails ; `tactical_service_cellule_enrichir.go` et la conversion `instant_affiche_ms` relèvent de L2.5, non touchés par L0-L1.
- [x] L0.4 Plan dans `.ai/`, entrée thought_log « En cours ». — entrée thought_log « En cours » ajoutée (L0-L1).

### L1 Stockage, collecteur, migration (Go)
- [x] L1.1 `film/replay/death_context.go` : `ContexteMort.Coequipiers []CoequipierDeMort{XUID uint64; Etat string; X, Y, DistanceM *float64}`, tri par xuid, arrondis existants (`arrondiMetres`, `distanceHorizontale`), `PlusProcheM` dérivé de la même liste. — `ContexteMort.Coequipiers`, `CoequipierDeMort` (X, Y, DistanceM nil hors visible), tri par xuid, `PlusProcheM` dérivé de la liste (`plusProcheDesVisibles`).
- [x] L1.2 `migration/steps_shared_match_lives_mates.go` : step `shared_match_death_mates_v1` (id PK séquence, match_id, decode_pass, decoder_rev, written_at, victim_xuid, time_ms, mate_xuid, state, x, y, distance_m NULL ; index `(match_id, victim_xuid, time_ms, written_at)`) ; vue `match_death_mates_latest` (QUALIFY dernière passe entière) ; entrée dans `migration/order.go` entre `..._lives` et `..._lives_placement`. — step `shared_match_death_mates_v1`, vue `match_death_mates_latest`, entrée dans `order.go` après `shared_match_lives_v1`.
- [x] L1.3 `migration/compaction_registry.go` `dernierePasse("match_death_mates","decode_pass")` + tests de compaction. — registre + oracle de `compaction_e2e_test.go` ; tests de compaction verts.
- [x] L1.4 `persist/lives_persister.go` (extraire `lives_mates_persister.go` si besoin) : `DeathMateInsert`, `LivesBatch.Mates` ; constantes d'état + extension `archlint/no_life_cause_divergence_test.go` ; `validateLivesBatch` : somme des lignes = `teammates_total`, comptes par état = colonnes, état visible ⇔ x/y/distance, `nearest_teammate_m` = min des distances visibles. — extraction faite : `persist/lives_mates_persister.go` (`DeathMateInsert`, `EtatCoequipier*`, `validateMates`) ; `LivesBatch.Mates` ; ratchet d états dans `archlint/no_life_cause_divergence_test.go` (`sourcesDesEtats`).
- [x] L1.5 `sync/killcollector/isolation_facts.go` : lignes de coéquipiers (`matesDe`), compteur `killsource_isolement_coequipiers_ecrits`, bump `IsolationDecoderRev` → `isolement-2026-10-14-coequipiers`. — `toDeathContextRows` rend aussi les lignes ; traduction inline (pas de fonction `matesDe` : citer `replay.ContexteMort` / `CoequipierDeMort` hors de `film/` aurait fait monter le plafond de `film_facade_surface_test.go`, 299).
- [x] L1.6 Garde-rails : `sync/no_art_patterns_test.go`, `sync/append_only_state_guard_test.go`, `platform/duckdb/no_raw_rating_reads_test.go`. — `match_death_mates` ajoutée aux trois listes (sans allowlist de motif risqué).
- [x] L1.7 En-têtes `cmd_backfill_killsource.go`.
- Tests : death_context (états, ordre, sommes), persister `integration`, isolation_facts, sélection backfill.
- Gate : `go test` des paquets touchés + `go test -tags=integration ./internal/persist/... ./internal/sync/... ./internal/migration/... ./internal/games/halo_infinite/migrations/...` + `make go-api-lint`.

### L2 Lectures, API, contrat
- [!] L2.0 Sondes (`*_research_test.go` skippé sans parc) : bots dans `match_participants`, unicité gamertag bot, doublons `(match_id, victim, time_ms)`, part des morts par bot. Fixe l'attribution d'un vengeur bot. — PARC OCCUPE (backfill killsource en cours sur les bases reelles, consigne du superviseur du 14/10) : sondes ECRITES (`platform/duckdb/tactical_repo_coequipiers_research_test.go`, tag `research`, `SONDE_SHARED=<copie>`), NON JOUEES ; a lancer par le superviseur sur une copie. Decision provisoire, a confirmer par les sondes : un vengeur doit avoir un xuid de l equipe de la victime (un bot vengeur n est pas attribue) ; un tueur ou un auteur de frag bot se designe par son gamertag.
- [x] L2.1 `domain/tactical_equipe.go` : `TacticalCoequipier`, `TacticalEquipeALaMort`, `TacticalRiposte`, `TacticalBilanZone`, `TacticalSequence*` ; `TacticalContribution.equipe` / `.riposte` ; `TacticalCelluleReponse.bilan` {morts : 4 statuts, examinées, rose[8], médiane tueur, médiane plus proche, tueur abattu N/M ; frags : abattu en retour N/M}. Aucun champ « non mesuré ». — `domain/tactical_equipe.go` ; `ContexteDeMort` / `MortContexte` gagnent `EnAttente` et `Total` ; quatre statuts de mort : accompagnee, visible_hors_rayon, aucun_visible, equipe_a_terre.
- [x] L2.2 `analysis/coordination/` : `riposte.go` (`TueurAbattu`, `AbattuEnRetour`, `FenetreRiposteMs`), `rose.go` (`Rose8`), `statut_plus_proche.go` (via `APortee`) ; garde-rail contre un second littéral 3000. — `riposte.go`, `rose.go` (secteur k = [45k, 45(k+1)) degres depuis le haut, sans trigonometrie), `statut_plus_proche.go` ; distances et medianes dans `analysis/tactical/mesures_equipe.go` (le ratchet `no_naked_rate_test` interdit un float64 exporte dans `coordination`) ; `domain.TacticalRiposte` ajoute a sa liste blanche (justification datee) ; garde-rail `archlint/no_second_riposte_window_test.go`.
- [x] L2.3 `platform/duckdb/tactical_repo_coequipiers.go` `MatesDesMorts` (`_latest`, `IN` bornés, liste blanche `exigerLaListe`). — `tactical_repo_coequipiers.go` (LEFT JOIN participants pour le nom, cle exacte filtree cote Go).
- [x] L2.4 `tactical_repo_ripostes.go` (journal publiable de la zone + positions du coup via `positionsAtKill`, `HAVING count(*) = 1`, LEFT JOIN justifié) ; `port/tactical.go` + mocks. — `tactical_repo_ripostes.go` (`JournalDesKills` ; le GROUP BY sur l identite du kill remplace le HAVING count(*) = 1, la position du TUEUR etant la meme pour deux victimes d un double kill) ; port + mocks.
- [x] L2.5 `service/tactical_service_equipe.go` : `poserEquipe`, `poserRiposte`, `bilanDeZone` depuis `enrichir` ; sources best-effort `sourceCoequipiers`, `sourceJournal` ; sections `timing` ; rayon `rayonsParMatch` ; distance coéquipier→tueur sur `AutrePoint` ; bilan morts pour `morts`/`isole`/`solde`, frags pour `kills`/`solde`. — `tactical_service_equipe.go` + `tactical_service_bilan.go` ; `enrichir` rend aussi le bilan ; les contextes ne sont pas relus pour la lecture « isole » (`MortContexte` porte `Total`/`EnAttente`) ; mort sans rayon : liste sans statut ; lignes != `teammates_total` : pas d equipe.
- [x] L2.6 Vue D serveur : `analysis/tactical/sequence.go` (étapes −4..+3 s au pas de 1 s, traînes 4 s, compte dans le rayon via `APortee`, cercle sur la victime puis au point de mort, pas d'interpolation à travers `G > 0`) ; store `service/tactical_sequence_store.go` (lecture fichier, décodage partiel, `timing`) ; `MatchsOuvrables` ; conversion partagée ; sans `originMs` = indisponible sans message ; ratchet de façade confirmé. — `analysis/tactical/sequence.go`, `service/tactical_sequence_store.go`, `service/tactical_service_sequence.go` ; aucun nouveau symbole `replay.X` hors de `film/` (pistes decodees dans `domain.TacticalPistes`) : plafond de facade intact ; horloge : `matchMs - originMs`, affichage par `instantAffiche`.
- [x] L2.7 Handler `api/handlers/tactical_sequence.go`, `POST /players/{p}/tactical/{map}/sequence` `{match_id, victim_xuid, time_ms}`, sous `RequireCapability(CapReplay)` + `LocalOnlyReplay` (selon D2), `archlint/replay_routes_capability_gate_test.go` étendu, 404 `replay_not_available`, `middleware.readOnlyPostPrefixes`. — monte dans le groupe `NewReplayHandler` ; ratchet `archlint/tactical_sequence_gate_test.go` (porte de titre ET garde local, memes extracteurs) ; `readOnlyPostPrefixes` couvre deja `/tactical/` [~] (cas de test ajoute).
- [x] L2.8 `make openapi-gen`, `make generate-types`, `make openapi-check`. — `openapi.yaml` regenere et `-check` vert ; `generated.ts` regenere par le CLI openapi-typescript du checkout principal (lecture seule, 162 ajouts, 0 suppression) faute de `node_modules` dans ce worktree ; `tools/check-generated-types-fresh.mjs` NON JOUE (exige `apps/web/node_modules`) : a rejouer par le superviseur.
- [x] L2.9 Mesure de lecture du plus gros artefact, consignée. — lecture des pistes : 123-139 ms sur le plus gros artefact (13,4 Mo, 338 pistes, 157 000 points), 81-100 ms sur 8 Mo ; < 500 ms. Poids d une zone de 100 morts : 99 Ko (Arene) / 231 Ko (BTB) contre 30 Ko sans equipe (`TestCellule_PoidsDeLaReponseDUneZoneChargee`, plafonds 130 / 300 Ko).
- Tests : purs (bornes 3 s et rayon, délai 0, vengeurs multiples, tueur bot, 8 frontières de secteurs) ; repo `:memory:` + `exigerFenetresBornees` + EXPLAIN + refus sans liste ; service avec mocks (sources absentes, titre sans capability, match sans rayon) ; handler httptest ; contracttest ; Halo 5 partiel sans 5xx.
- Gate : `go test` analysis/service/platform/duckdb/api/archlint/contracttest + `make openapi-check` + `make go-api-lint`.

### L3 Web
- [x] L3.1 `lib/query/keys.ts`, `queries.ts` `useTacticalSequence` (404 silencieux, sans retry). — `queryKeys.tacticalSequence`, `useTacticalSequence` (retry: false, 404 sans bruit ; entree datee dans la liste du garde `queriesPlaceholder`).
- [x] L3.2 i18n FR+EN (`tactical.toml`, générateur), titres factuels sans possessif, test « aucune chaîne non mesuré ». — `tactical.mort.*` (FR+EN, générateur rejoué), `mortText.ts` ; test « instant, jamais heure, aucun non mesuré ».
- [x] L3.3 Vue C : `TacticalZoneBilan.tsx`, `TacticalRose.tsx`, `TacticalPlanCoequipiers.tsx`, `bilan.logic.ts` (barre 4 statuts, rose + médiane tueur, tueur abattu N sur M, médiane coéquipier, nuage `team-ally`) pour `morts` et `isole`. — `TacticalZoneBilan.tsx`, `TacticalRose.tsx`, `mort.logic.ts` ; le nuage est peint par `TacticalPlanCoequipiers.tsx` (pas de fichier `TacticalPlanCoequipiers` séparé du reste de la couche) ; vue C pour `morts` et `isole`, miroir des frags pour `kills` et `solde`.
- [x] L3.4 Parcours : clic sur une mort → `TacticalMortDetail.tsx` + retour à la zone ; encadré « Instant de la mort · date · horloge » (`instant_affiche_ms`) ; bouton `MatchReplayLink`. — `TacticalMortDetail.tsx` (en bas de la colonne), ouverte par la sélection de tuile ; horloge du rejeu (`instant_affiche_ms`).
- [x] L3.5 Vue A+B (sur le plan principal, cf. A1) : mort, tueur + trait, coéquipiers pleins/creux, cercle, hors de vue / en attente listés sans point ; « Tueur abattu N,N s après la mort, par X », trait coéquipier→tueur pour ce frag seulement ; distances à la mort et au tueur. — scène `sceneDeMort`, bulle d échange sur le plan, trait du frag du coéquipier ; tueur peint par la couche des tueurs (non modifiée).
- [x] L3.6 Vue D : `TacticalSequence.tsx`, étapes −4..+3 s, traînes 4 s, compte dans le rayon ; rendue seulement si la requête réussit. — barre `Instantané` + 8 étapes, `sceneDeSequence`, fiche d étape ; rendue seulement si la requête réussit.
- [~] L3.7 Miroir frags : « abattu en retour en 3 s » et par qui ; `solde` = deux faces. — carte « Abattu en retour en N s : X sur Y » en vue de zone (kills, solde) et échange dans la fiche d un frag fait ; le BADGE sur la tuile est reporté après la refusion de `feat/tactique-tueurs` (`TacticalRejeuTile.tsx` appartient à l autre exécuteur).
- [x] L3.8 Non-régression : nouveaux boutons de rejeu par le même chemin (−5 s, calage). — le bouton de rejeu de l encadré passe par `modeleDeTuile` (mêmes `search`, même libellé, même recul de 5 s).
- [~] L3.9 `[~]` cf. A4. — couvert par le lot tueurs (A4).
- [x] L3.10 Garde-rails : couleurs, TanStack Table, seuils (ne pas grossir `TacticalPlanCard.tsx`). — `lint:colors`, `lint:fields`, eslint, knip verts ; `TacticalPlanCard.tsx` +17 lignes.
- Gate : `make check-types`, vitest features/tactical lib/replay lib/match-nav, lint couleurs et champs ; test discriminant d'horloge (countdown non nul).

### L4 Backfill (BLOQUANT, accord user, superviseur seul)
- [ ] L4.1 L1-L3 verts et fusionnés ; serveur local ARRÊTÉ ; sauvegarde à jour.
- [ ] L4.2 Accord écrit du user (≈ 1 600 films, 3 ouvriers, ≈ 54 min de référence du 22/09, reprise par `decoder_rev`).
- [ ] L4.3 `go run ./cmd/levelup backfill-killsource --dry-run` puis `--workers 3` (+ `--status`) ; pas de `--force` ; pas de cuisson d'artefacts.
- [ ] L4.4 Vérifications : somme des lignes = `teammates_total`, une `decode_pass` par match dans `_latest`, compteurs cohérents.
- [ ] L4.5 Production : prévenir avant toute opération VPS ; push `main` = déploiement.

### L5 Clôture
- [ ] L5.1 `docs/COMMANDS.md` + `docs/FR/COMMANDS.md` (tables écrites par la passe), CHANGELOG EN+FR.
- [ ] L5.2 Skill `db-schema` (`match_death_mates`), `.ai/project_map.md` si besoin.
- [ ] L5.3 thought_log (une entrée par lot, mesures L2.0 / L2.9).
- [ ] L5.4 `.ai/REGISTRE_REPORTS.md` : bots dans les positions (D1), garde local de D (D2), écart artefact / persisté.
- [ ] L5.5 Revue adversariale L1 (persist/sync) et L2 (algo), P0/P1 traités.
- [ ] L5.6 `delivery-checklist`, `make gate-push`, `disk-hygiene -Lot coequipiers -Apply`.

### Découvertes (non traitées)
- `match_death_context.teammates_left` est une constante morte.
- La fonction d'état d'un coéquipier ignore les bots.
- Revue adversariale L1+L2 (P2-3) : un refus de `validateMates` fait perdre `match_lives` et `match_death_context` du match pour la passe — choix assumé (un producteur incohérent ne publie rien) ; à écrire dans l ADR de clôture. Dédoublonnage des coéquipiers par `mate_xuid` à la lecture, écart restant journalisé et compté (`tactique_equipe_lignes_incoherentes`).

### Résumé des lots : L0 (prérequis), L1 (stockage `match_death_mates` append-only + vue `_latest`, collecteur, bump `IsolationDecoderRev`, garde-rails ART/compaction/divergence d'états), L2 (sondes L2.0, domaine, `analysis/coordination` riposte/rose/statut, lectures repo bornées, service, vue D côté serveur avec lecture partielle d'artefact, handler `POST /tactical/{map}/sequence` sous `CapReplay` + `LocalOnlyReplay`, OpenAPI), L3 (web : vues C, A+B, D, miroir frags, i18n, garde-rails), L4 (backfill `backfill-killsource --workers 3`, BLOQUANT, accord user, serveur arrêté), L5 (clôture : docs bilingues COMMANDS + CHANGELOG, db-schema, thought_log, registre des reports, revue adversariale L1/L2, delivery-checklist, gate-push).
 L0 (prérequis), L1 (stockage `match_death_mates` append-only + vue `_latest`, collecteur, bump `IsolationDecoderRev`, garde-rails ART/compaction/divergence d'états), L2 (sondes L2.0, domaine, `analysis/coordination` riposte/rose/statut, lectures repo bornées, service, vue D côté serveur avec lecture partielle d'artefact, handler `POST /tactical/{map}/sequence` sous `CapReplay` + `LocalOnlyReplay`, OpenAPI), L3 (web : vues C, A+B, D, miroir frags, i18n, garde-rails), L4 (backfill `backfill-killsource --workers 3`, BLOQUANT, accord user, serveur arrêté), L5 (clôture : docs bilingues COMMANDS + CHANGELOG, db-schema, thought_log, registre des reports, revue adversariale L1/L2, delivery-checklist, gate-push).
