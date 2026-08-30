# PLAN — Note de performance : scission ranked par famille, métrique objectif, hygiène des non-terminés (+ volet Escouade)

> Rédigé le 2026-08-27 après diagnostic sur pièces + analyse corpus (session pilote).
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, gate
> passé avant l'étape suivante, statuts [x]/[~]/[!] obligatoires, zéro fix hors périmètre,
> découvertes consignées ici, jamais traitées à la volée).
> Validation utilisateur : diagnostic + recommandations validés le 2026-08-27 (« go »),
> y compris purge sèche des notes orphelines (pas de fallback famille sociale) et
> exigence explicite : recompute inclus ET les futurs matchs adoptent le nouveau régime
> sans intervention.

## Objectif et critère de succès

- La note de perf des modes objectifs est calibrée : un match ranked objectif est comparé
  à l'historique ranked objectif (idem slayer), et la participation à l'objectif compte
  dans la note des chaînes objectif sans récompenser l'absence de combat.
- Plus aucune note stockée sur un match non terminé (outcome=4), exclu, ou sous seuil de
  chaîne. Recompute force exécuté pour les 4 joueurs suivis.
- Un match futur (post-sync) reçoit chaîne + note selon le nouveau régime automatiquement
  (chemin `engine_postsync_scoring.go:83` vérifié par test d'intégration).
- Volet A : la case « Composition stricte » de la page Escouade est cochée par défaut.

## Branches / worktrees

| Volet | Branche | Worktree | Base |
|---|---|---|---|
| A (Escouade) | `wt/squad-compo` | `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-squad-compo` | feat/v75 @ 823cf96d5 |
| B (perf note) | `feat/perf-note-objectifs` | `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-perfnote` | feat/v75 @ 823cf96d5 |

Volet A se replie dans `feat/v75` à la clôture (mode branche unique v7.5). Volet B reste
sur sa branche (chantier isolé) ; merge décidé avec l'utilisateur après la v7.5.0.
Commits : TOUJOURS demander l'autorisation utilisateur (un commit par lot, fichiers
stagés explicitement, jamais `git add -A`).

## Décisions verrouillées (ne pas rouvrir en exécution)

- D-A **Scission ranked en 2 chaînes** : `ranked_slayer` / `ranked_objectif` (remplacent
  `ranked`). Classification par le sous-mode du pair_name via la MÊME liste objectif que
  le social (`classify.go:75-85`), factorisée en helper partagé unique + garde-rail grep.
  Fallback (pair_name NULL/inconnu, classifier absent) : `ranked_slayer`.
- D-B **Pas de chaînes par mode ni par « dominante »** (défensive/agressive/collaborative).
  Corpus : écarts intra-famille (pspm ±14 %) moitié moindres que l'écart de famille
  (+28 %), volumes morts (Oddball 19, KOTH 68). La nuance passe par les poids, pas par la
  population de référence.
- D-C **Nouvelle métrique `objective_participation` (ospm)** = points d'awards catégorie
  `objective` par minute (source `personal_score_awards`, DB joueur, lecture
  tombstone-aware), active UNIQUEMENT sur les chaînes de famille objectif
  (`arena_objectif`, `ranked_objectif`). Poids de départ (à confirmer lot 0) :
  ospm 0.12 ; kpm 0.14→0.10 ; kda 0.11→0.09 ; accuracy 0.04→0.03 ; pspm 0.10→0.08 ;
  TOUTES les métriques morts/dégâts/attendus inchangées (dpm_deaths 0.10,
  deaths_vs_expected 0.07, defensive_resistance 0.05, kills_vs_expected 0.09,
  dpm_damage 0.06, apm 0.06, rank_perf 0.04, medal_exploit 0.06,
  offensive_conversion 0.09). Somme ≈ 1.04, renormalisée comme aujourd'hui.
  Chargement DANS `batchComputePerformanceScores` depuis playerDB (aucun changement de
  signature → couvre les 5 call-sites : engine_postsync_scoring.go:83,
  engine_backfills.go:352, enrichment_backfill.go:115, match_recomputer.go:102,
  openspartan_post_import_service.go:250). Table absente (autre titre) → map vide +
  log debug, dégradation gracieuse (pas de capability nouvelle, pas de `slug ==`).
- D-D **Purge sèche des notes orphelines** (validée contre l'alternative fallback famille
  sociale) : toute ligne enrichment avec `performance_score` non NULL dont le match est
  hors univers de calcul (outcome=4, is_excluded, sous seuil de chaîne) est remise à
  NULL (score + chain). Conséquence assumée et annoncée : les 8 notes ranked de JGtm et
  les 8 de Chocoboflor disparaissent (elles dataient de l'ère pré-chaînes, référence
  globale abandonnée) ; Madina garde ~14 notes ranked recalculées par famille.
- D-E **Le batch devient auto-nettoyant** : à chaque run (force ou non), les scores
  stockés qui ne qualifient plus sont NULLés (pas seulement à la purge one-shot).
  L'exclusion manuelle d'un match est donc rattrapée au batch suivant (pas de nettoyage
  synchrone dans match_exclusion_service — hors périmètre).
- D-F **BTB non scindé** (écarts modestes, volumes faibles hors Madina) — consigné au
  registre des reports avec condition de reprise (volumes BTB en hausse ou plainte
  calibration BTB).
- D-G **`buildFormTab`/`analysis.ComputePerformanceSeries` (onglet API `form` sans UI)** :
  NON touché par ce chantier. Candidat code mort à statuer séparément avec l'utilisateur.
- D-H **Seuils inchangés** : fenêtre 50, `MinMatchesPerChainForRelative` 10, pas de
  fallback global.
- D-I **Lacunes de la liste des sous-modes objectif — CORRIGÉES dans ce chantier**
  (décision utilisateur du 2026-08-27, remplace le report initial). Constat (annexe du
  rapport lot 0) : 26 matchs objectifs classés slayer — Assaut `neutral bomb` /
  `one bomb` / `neutral bomb squad`, `vip`, `ctf 3 captures`, et 14 pair_names INVERSÉS
  type `Strongholds:Arena on Behemoth` (mode à GAUCHE du deux-points, catégorie Other).
  Traitement en DEUX temps assumés : le lot 1 pose le helper avec la liste VERBATIM
  (diff byte-identique hors ranked, régression impossible à confondre), puis le
  **lot 1bis** corrige la classification dans son propre diff. Conséquence LUSR assumée :
  ~25 matchs sociaux changent de chaîne LUSR → recompute LUSR complet des 4 joueurs au
  lot 4 via `RecomputeLUSRCanonicalForPlayer` (sync/lusr_full_recompute.go:26, replay
  complet watermark-safe, garde append_only_state_guard_test). CSR non concerné (l'API
  est la source). Le recompute force de perf du lot 4 absorbe l'effet fenêtres.
- D-J **Poids FIGÉS au gate 0 (2026-08-27)** : ospm = 0.12, profil objectif D-C
  confirmé sans modification. Lecture PSA du lot 3 : vue
  `personal_score_awards_latest` (dédup génération max + non-tombstone, ADR 0026),
  après réparation d'index B2.4. Présence d'ospm : métrique présente ssi le match a une
  couverture PSA (0 point objectif couvert = ospm 0 légitime ; aucune ligne PSA =
  métrique absente, poids redistribué).

## Volet A — « Composition stricte » cochée par défaut (Escouade)

Périmètre fermé (aucun changement backend — `filter_exact_composition` reste optionnel
défaut off côté API, le front l'envoie toujours) :

- [x] A1 Helper pur `exactCompositionDefault(stored: string | null): boolean` (nouveau
      petit module à côté de `squadPending.ts`) : `null → true`, `'true' → true`,
      `'false' → false`. Utilisé par l'init de `SquadLayout.tsx:169-172` (le choix
      stocké reste respecté ; pas de clé versionnée).
- [x] A2 i18n FR+EN réécrits dans le sens du nouveau défaut :
      FR (`features/squad/i18n.ts:372-374`) : label « Composition stricte » inchangé ;
      title = « Cochée par défaut : seuls les matchs joués avec exactement cette
      composition sont comptés. Décochez pour inclure tous les matchs commencés
      ensemble, même si un autre joueur connu vous accompagnait. »
      EN (`:716-718`) : « Ticked by default: only matches played with exactly this
      line-up are counted. Untick to include every match started together, even if
      another known player was with you. »
      Doc commentaire du type (`:36`) : « (activée par défaut) ».
- [x] A3 `features/squad/queries.ts:26` : fallback `?? false` → `?? true`. Étendu en
      exécution (découverte exécuteur vérifiée pilote) : défaut de paramètre
      `exactComposition = true` dans `lib/query/keys.ts:164` — les deux moitiés de la
      chaîne de fallback alignées.
- [x] A4 Commentaire `SquadLayout.tsx:671-672` (« option OFF par défaut ») réécrit.
- [x] A5 Tests : 4 cas du helper (A1, dont valeur illisible → défaut) ; aucun test
      existant ne supposait le défaut off (vérifié par grep exécuteur).

Gate A (commandes exactes, avant-plan uniquement) :
```
cd apps/web
Remove-Item -Recurse -Force node_modules\.tmp   # purge tsbuildinfo (faux vert)
npm run typecheck
npm run lint
npx vitest run src/features/squad
```
Puis revue navigateur par le pilote (population intersection par défaut, état vide propre,
ré-ancrage session composition) et gate visuel utilisateur.

**Statut Volet A : LIVRÉ le 2026-08-27** — commit `6b7c5402b` (wt/squad-compo) mergé
fast-forward dans feat/v75 et poussé sur origin (autorisation utilisateur explicite).
Gates rejoués par le pilote en première main : typecheck exit 0 (cache purgé), lint
exit 0 (22 warnings pré-existants), vitest squad+lib 1321/1321 exit 0. CI de branche
verte au niveau JOB (CI 25m48s + Deploy Pre-Check + Secrets, run 33093141505).
**Gate visuel utilisateur VALIDÉ le 2026-08-27 — VOLET A CLOS.**

## Volet B — chantier note de perf (lots strictement séquentiels)

### Lot 0 — Simulation offline (AUCUN code produit)

- [x] B0.1 Outil jetable `apps/go-api/cmd/diag_perfsim/` (lecture seule,
      `?access_mode=read_only`, style diag_q — fmt autorisé dans cmd/diag_*) qui rejoue
      le pipeline complet pour les 4 joueurs suivis : SQL de `loadHistoryForPerf`
      (performance_helpers.go:153-218, exclusion outcome=4 comprise) + exclusions
      is_excluded (reprendre la table lue par `loadExcludedMatchIDs`) + chaînes NOUVEAU
      régime (réutiliser `skillchain.ClassifyLUSRChain` /
      `halo_infinite.InferModeCategoryFromPairName` / `analysis.NormalizeModeLabel` ;
      liste objectif copiée localement dans l'outil, la factorisation prod = lot 1) +
      fenêtre 50 / seuil 10 + percentiles pondérés (répliquer performance.go).
- [x] B0.2 ospm : somme `award_score` catégorie `objective` par match depuis
      `personal_score_awards` (NOT is_tombstone) ; INVESTIGUER les doublons
      génération/written_at et documenter la règle de dédup retenue dans le rapport.
- [x] B0.3 Comparaison : profil de poids ACTUEL partout vs NOUVEAU régime (scission +
      profil objectif D-C) + 2 variantes de sensibilité (ospm 0.08 et 0.16).
- [x] B0.4 Rapport `<worktree>\.ai\RAPPORT_SIM_PERF_NOTE_2026-08.md` : par joueur ×
      chaîne — n matchs, n scorés avant/après, médiane/p10/p90 des notes avant/après ;
      comptes de notes purgées par joueur (DNF / sous-seuil / exclus) ; top-5 matchs
      « écrasé mais actif à l'objectif » (percentile combat bas, percentile ospm haut)
      avec note avant/après ; recommandation de poids finale argumentée.

Gate 0 : outil build + run verts sur les 4 joueurs ; rapport complet ; critères :
médiane des notes par chaîne scorée ≈ 50 ± 5, zéro note simulée sur outcome=4, tableau de
sensibilité présent. LES POIDS SE FIGENT ICI (décision pilote+utilisateur consignée dans
ce fichier avant lot 3).

**Gate 0 PASSÉ le 2026-08-27.** Outil vert (build/vet/gofmt/lint/run exit 0). Rapport :
`LevelUp-wt-perfnote\.ai\RAPPORT_SIM_PERF_NOTE_2026-08.md`. Réplique du régime actuel
exacte à 0.00 pt chez JGtm (1033 matchs) et Daemon (12) ; écarts Choco/Madina (1.3-2.2 pt
moyens) dans la plage de la sonde « métrique 0.06 absente » (medal_exploit). 10/12
chaînes dans [45,55] ; les 2 hors fenêtre (chaos Madina 41.0 inchangée par la réforme ;
ranked_objectif Madina 38.8 → 44.4, AMÉLIORÉE par la scission) sont pré-existantes et
documentées. Zéro note simulée sur outcome=4 (116 DNF). Purge prévue : JGtm 78 /
Chocoboflor 41 / Madina 118 / Daemon 10 — recoupe le diagnostic à l'unité près.
**POIDS FIGÉS : ospm = 0.12**, profil D-C confirmé tel quel (gain des « écrasés mais
actifs » = 2.0× le plancher de bruit ; à 0.16 les porteurs de combat perdraient
jusqu'à 12 pts et la marge de non-inversion tomberait au niveau du bruit ; profil
symétrique +4.7/−4.4, médianes stables).

### Lot 1 — Scission ranked par famille

- [x] B1.1 Helper partagé de famille objectif dans `games/halo_infinite/skillchain`
      (ex. `IsObjectiveSubMode(pairName)`) — SEULE occurrence de la liste ;
      `lusrChainForAssassin` migre dessus.
- [x] B1.2 Garde-rail grep : test qui interdit une 2e occurrence du littéral de liste
      (même esprit que les ratchets existants).
- [x] B1.3 `GetPerformanceChain` (skill_config.go:185-196) : `isRanked` →
      `ranked_slayer`/`ranked_objectif` via un seam title-aware de famille (miroir de
      `SetLUSRChainClassifier` ; halo_5 : vérifier son wiring, fallback `ranked_slayer`
      documenté). Constantes `PerfChainRankedSlayer`/`PerfChainRankedObjectif` ;
      l'ancienne valeur `ranked` ne survit que comme donnée stockée à recalculer
      (mécanisme skip existant performance.go:372-379).
- [x] B1.4 Tests : extension `perf_chain_test.go` (Ranked:CTF/Oddball/Strongholds/KotH →
      ranked_objectif ; Ranked:Slayer → ranked_slayer ; pair NULL ranked →
      ranked_slayer), cross-check sync_test, cas H5.

Gate 1 :
```
# PowerShell natif, recette CGO msys64, séquentiel
go test ./internal/sync/... ./internal/games/...
go vet ./...
go test -tags=integration -count=1 -p 1 ./...   # sync touché → OBLIGATOIRE, exit code vérifié
```

**Gate 1 PASSÉ le 2026-08-27.** Unit scope / vet / build / gofmt exit 0 ; intégration
complète `-p 1` en 24,8 min — périmètre vert (sync 162 s, persist 22 s, duckdb 117 s,
migration 23 s ok), 2 échecs HORS périmètre : (a) ratchet `start_time` déclenché par
`cmd/diag_perfsim/load.go` (outil lot 0) — soldé par le pilote : entrée allowlist datée
avec justification « oracle-réplique de performance_helpers.go, à migrer ensemble »,
ratchet rejoué vert ; (b) timeout local `himap` (>10 min), lenteur locale connue — la CI
Linux fait foi. `PerfChainRanked` conservée (2 usages vivants hors classification :
playlist_group CSR, parsing de nom de playlist) avec commentaire 3-statuts. Migration
des notes stockées : GRATUITE par le skip de chaîne (performance.go:372-379, vérifié sur
pièces — `ranked` n'étant plus jamais recalculée, tout match classé noté repasse au
calcul dès le premier batch, sans force). Câblage seam : serveur en fail-fast
(`ValidateObjectiveFamilyClassifierWired`), binaires LUSR/h5 par symétrie, fallback
ranked_slayer documenté pour les binaires non câblés.

### Lot 1bis — Correction de la classification objectif (D-I, décision user 2026-08-27)

Pré-requis : lot 1 clos (le helper unique existe). Périmètre : la classification de
FAMILLE uniquement — `NormalizeModeLabel` et `InferModeCategoryFromPairName` (catégories
UI) ne changent PAS.

- [x] B1b.1 Le helper unique devient `IsObjectiveMode(pairName)` (ou équivalent) avec
      deux règles : (1) sous-mode normalisé (partie droite) ∈ liste objectif ; (2) sinon,
      si le PRÉFIXE (partie gauche du `:`, normalisée) ∈ liste objectif → objectif
      (couvre les pair_names inversés type `Strongholds:Arena`). Ajouts à la liste :
      `vip`, `neutral bomb`, `one bomb`, `neutral bomb squad`, `ctf 3 captures`.
      `arena` n'entre dans AUCUNE liste.
- [x] B1b.2 Brancher les trois consommateurs sur le helper : `lusrChainForAssassin`
      (déjà fait au lot 1), `lusrChainForOther` (les inversés sont en catégorie Other —
      test IsObjectiveMode APRÈS les règles chaos, avant le fallback arena_slayer), et
      le classifieur de famille ranked (lot 1).
- [x] B1b.3 Tests : les 26 cas du corpus (annexe rapport lot 0) deviennent des fixtures —
      `Assault:Neutral Bomb on Origin` → arena_objectif, `Arena:VIP on Catalyst` →
      arena_objectif, `Strongholds:Arena on Behemoth` → arena_objectif,
      `Ranked:CTF 3 Captures on Argyle` → ranked_objectif (remplace le cas « limitation
      connue » du lot 1), etc. + cas de non-régression (Fiesta/chaos prioritaires sur la
      règle préfixe ; `Arena:Slayer` intact). Garde-rail B1.2 mis à jour si nécessaire.
- [x] B1b.4 Rejouer `cmd/diag_perfsim` (lecture seule, serveur arrêté) et vérifier le
      delta : exactement les 26 matchs changent de famille sur le corpus des 4 joueurs
      (14 `arena` + 4 `neutral bomb` + 3 `vip` + 3 `one bomb` + 1 `neutral bomb squad` +
      1 `ctf 3 captures`) — toute autre bascule = STOP et analyse. Consigner les comptes
      LUSR impactés (~25 sociaux) pour le recompute du lot 4.

Gate 1bis : gates go du lot 1 rejoués (unit + vet + build + intégration -p 1) + le delta
de B1b.4 conforme (26 matchs, pas un de plus).

**Gate 1bis PARTIEL le 2026-08-27** — code livré et vérifié pilote (helper 17 entrées +
règle du préfixe via `isObjectiveModeLabel` interne, nom `IsObjectiveSubMode` conservé —
justifié : symétrie du seam avec halo_5, 20 call-sites, le sous-mode d'un pair inversé
EST la partie gauche). Gates : unit/vet/build/gofmt exit 0 ; intégration `-p 1` en 6
tranches couvrant 304/304 packages, ZÉRO `--- FAIL:` (seul himap sort en timeout local
connu). RESTE `[!]` : le rejeu corpus B1b.4 — bloqué par le serveur dev actif sur les
DBs réelles (session utilisateur) ; condition de reprise : port 8000 libre.
**Grille de lecture du rejeu (consignée d'avance)** : (a) l'annexe des sous-modes est
bâtie sur l'UNIVERS complet (DNF/exclus inclus), les tables par chaîne sur le SCORABLE —
le delta lu dans les tables sera ≤ 26 sans anomalie ; (b) un pair de catégorie Other
portant un mode objectif (ex. `Rumble Pit:Oddball`) peut bouger EN PLUS des 26 de
l'annexe (qui ne couvrait qu'Assassin+Ranked) — valider ces movers À LA PIÈCE (le pair
porte réellement un mode objectif) au lieu de crier au 27e ; (c) le rapport de l'outil
imprime désormais `missClassified` avec attendu 0 (témoin automatique).

**B1b.4 REJOUÉ ET VALIDÉ le 2026-08-27** (serveur arrêté avec autorisation utilisateur,
RUN_EXIT=0, comptes globaux inchangés). Delta validé À LA PIÈCE par requêtes DB
directes : **26 movers exactement** — 6 `CTF:Arena on Opulence` + 6 `CTF:Arena Neutral
Flag` (Live Fire ×3, Prism ×3) + 2 `Strongholds:Arena` + 4 `Assault:Neutral Bomb` +
3 `Arena:VIP` + 3 `Assault:One Bomb` + 1 `Assault:Neutral Bomb Squad` (= 25 sociaux →
arena_objectif) + 1 `Ranked:CTF 3 Captures` (→ ranked_objectif). Tables par chaîne
réconciliées : JGtm arena_objectif 221→235 (+14), Choco 152→156 (+4), Madina 192→199
(+7) et ranked_objectif 21→22 (+1). **Correction du lot 0** : sur les « 14 pair_names
inversés », 6 étaient des modes SLAYER inversés (`Team Slayer:Arena` ×5, `Slayer:Arena`
×1) — ils restent à bon droit en famille slayer ; l'annexe lot 0 les comptait à tort
comme objectifs. La composition des 26 diffère donc de la prédiction (6 slayer-inversés
sortent, 6 `CTF:Arena Neutral Flag` entrent), le TOTAL de 26 est identique. Témoin
`missClassified` REDÉFINI (mesure les deux moitiés du pair_name, croise miroir et
classifieur de production) : **0 mal classé — attendu 0**, run vert, comptes inchangés,
aucun 27e mover. **GATE 1BIS INTÉGRALEMENT PASSÉ le 2026-08-27.**
Raffinement final (exécuteur, vérifié cohérent avec les requêtes pilote) : la
répartition des movers inversés est 8 par la branche Assassin (`CTF:Arena` ×6,
`Strongholds:Arena` ×2) et **6 par la branche Other** (`CTF:Arena Neutral Flag` —
les DEUX côtés inconnus du classifieur de catégories) : le branchement
`lusrChainForOther` (B1b.2) est un chemin ACTIF portant 6/26 movers, pas une simple
robustesse.

### Lot 2 — Hygiène non-terminés / orphelins (batch auto-nettoyant)

- [x] B2.1 Dans `batchComputePerformanceScores` : après la boucle, NULLer
      (score + chain) toute ligne `player_match_enrichment_latest` scorée dont le
      match_id n'est PAS dans l'ensemble qualifié du run (couvre DNF, is_excluded,
      sous-seuil). Écriture via `PostSyncEnrichmentPersister.BatchUpdateMulti`
      (patterns persist existants, jamais d'UPSERT concurrent), observabilité slog
      (compte NULLés par cause).
- [x] B2.2 Tests d'intégration (tags=integration) : DNF scoré → NULLé ; sous-seuil
      scoré → NULLé ; qualifié → conservé ; is_excluded scoré → NULLé ; idempotence
      (2e run = 0 changement).
- [x] B2.3 Garde-rail pérenne : assertion d'intégration « aucun performance_score sur
      outcome=4 » après un run de batch sur fixture.
- [x] B2.4 Réparation d'index PSA (découverte lot 0, CONFIRMÉE sur pièces par le pilote
      le 2026-08-27) : sur la DB `XxDaemonGamerxX`, `idx_psa_match` est incohérent —
      `WHERE match_id='05fffb2a-...'` rend 2 lignes, le scan forcé (`match_id||''=`)
      en rend 4. Item : balayer les 4 DBs joueur (comparaison lookup indexé vs scan
      forcé sur l'échantillon des match_ids multi-lignes), DROP INDEX + CREATE INDEX
      sur toute DB en écart, re-vérifier, consigner les comptes. Serveur arrêté,
      écriture one-shot de réparation d'intégrité (hors chemin batch). PRÉ-REQUIS du
      lot 3 : le loader ospm lit par prédicat indexé via `personal_score_awards_latest`.

Gate 2 : mêmes commandes que gate 1 (integration -p 1 incluse).

**Gate 2 PASSÉ le 2026-08-27.** Batch auto-nettoyant livré (extraction
`runPerfCleanupPass`, chargement des notes stockées dans les DEUX modes, ensembles
qualified/below, NULLage via `PostSyncEnrichmentPersister.BatchUpdateMulti` — NULL SQL
natif, INSERT-only stage perf, vecteur ART éliminé). DEUX resserrements nécessaires
tranchés en exécution et testés : le SEUIL prime sur le skip (un match sous-seuil
portant une note dans la bonne chaîne est nettoyé — la population a pu rétrécir), et
les retours anticipés (univers vide / tout exclu) nettoient aussi
(`cleanupAllScoredAsOrphans`). 6 tests d'intégration + garde-rail pérenne
`TestBatchPerformance_NoStoredScoreSurvivesOnUnfinishedMatch` (modes normal ET force).
Gates verts : unit/vet/build/gofmt exit 0, intégration -p 1 en tranches 305/305
packages zéro `--- FAIL:` (himap timeout local toléré).
**B2.4 exécuté sur les 4 DBs réelles** (port 8000 vérifié libre, dry-run → repair →
re-vérif par process neuf + contre-sondes pilote) : la corruption touchait LES QUATRE
DBs sur `idx_psa_match` ET `idx_psa_category` (JGtm : 220/3830 lignes invisibles à
l'index, 5,7 % ; Choco 37 ; Madina 32 ; Daemon 7) ; `idx_psa_gen` sain partout.
DROP+CREATE+CHECKPOINT, row counts identiques avant/après, zéro divergence après
(témoin 05fffb2a : 2/4 → 4/4). Cause racine de la désync NON élucidée (hors périmètre,
famille bug ART) — consignée en Découvertes.

### Lot 3 — Métrique ospm + profils de poids par chaîne

- [x] B3.1 Loader interne au batch : `match_id → somme award_score objective` depuis
      playerDB (dédup selon règle B0.2 ; table absente → map vide + slog debug).
- [x] B3.2 `MetricKeyObjectiveParticipation = "objective_participation"` ; ospm câblé
      dans historyRow/matchMetrics/prepareHistoryMetrics/getMetricValue
      (performance.go + performance_helpers.go).
- [x] B3.3 `RelativeWeights` devient profil par chaîne : `WeightsForChain(chain)` —
      défaut = profil actuel ; chaînes `arena_objectif`/`ranked_objectif` = profil
      objectif aux poids FIGÉS au gate 0. ospm absent des profils slayer/btb/chaos/
      firefight/ranked_slayer.
- [x] B3.4 Tests unitaires : profils (somme, renormalisation, ospm absent → poids
      redistribués), métrique ospm, match sans awards.
- [x] B3.5 Test d'intégration « futur match » : pipeline post-sync sur fixture (nouveau
      match objectif ranked) → chaîne `ranked_objectif` + note calculée avec ospm sans
      intervention (exigence utilisateur du 2026-08-27).

Gate 3 : gates go complets (dont integration -p 1) + rejouer `diag_perfsim` : les notes
simulées et les notes produites par le code réel sur les mêmes données doivent concorder
(écart ≤ 0.1 attendu ; divergence = bug à élucider avant clôture).

**Gate 3 PASSÉ (partie code) le 2026-08-27.** ospm + profils livrés : loader
`skill.LoadObjectiveParticipation` (vue `_latest`, xuid strict aligné sur le lecteur
canonique, cast ::DOUBLE — SUM(INTEGER) rend un HUGEINT), pointeur = couverture (D-J),
`WeightsForChain` avec profil objectif FIGÉ documenté, filtrage par profil AVANT
renormalisation, signature du batch inchangée (les 5 call-sites héritent). 12 tests
neufs verts dont les jumeaux « futur match » (actif objectif note plus haut que
l'inactif ; non-couvert = poids redistribué). Gates : unit/vet/build/gofmt exit 0,
intégration -p 1 tranches 303/303 zéro `--- FAIL:` (himap timeout local toléré).
DEUX arbitrages d'exécution VALIDÉS pilote : (a) loader dans
`internal/sync/skill/objective_participation.go` (sync gelé à 80 fichiers racine +
précédents LoadExcludedMatchIDs/isSchemaMissingErr dans skill) ; (b) `diag_perfsim`
aligné sur `skill.WeightsForChain` (source unique, seul ospm parametrable pour ses
variantes) mais MAINTENU en scan complet sans prédicat — sentinelle volontaire : une
divergence outil/produit au lot 4 signalerait une re-corruption d'index.
L'oracle « notes réelles vs simulées » sur données réelles est REPORTÉ AU LOT 4
(B4.2) : ce lot n'a aucun droit d'écriture réel. Tolérance à prévoir au lot 4 : le
recompute réel passe par le chemin backfill AVEC medal_exploit, la sim est sans —
écart moyen attendu ~1,2-2,2 pt (sonde du rapport lot 0), médianes ±1,5 pt.
Budget fichiers : performance.go 475 L / helpers 487 L — la prochaine évolution du
batch passera par un sous-package.

### Lot 4 — Recompute réel + gate visuel + clôture

- [x] B4.1 Serveur :8000 arrêté vérifié ; recompute force des 4 joueurs via le point
      d'entrée existant (CLI levelup backfill / enrichment_backfill — identifier la
      commande exacte, sinon mini-cmd diag) sur les DONNÉES RÉELLES depuis le worktree
      (binaire de la branche, LEVELUP_REPO_ROOT → checkout principal).
      PIÈGE (découverte lot 1) : `cmd/levelup` ne câble PAS les classifiers (LUSR ni
      famille objectif) — un recompute depuis un binaire non câblé classerait tout le
      ranked en ranked_slayer EN SILENCE. Exigence : le binaire du recompute pose les
      seams (Set*Classifier) et le vérifie (Validate*Wired) AVANT tout calcul.
- [x] B4.2 Contrôles diag_q post-recompute : 0 score sur outcome=4 ; 0 chaîne `ranked`
      restante ; distribution des chaînes conforme au rapport lot 0 ; comptes purgés =
      comptes prédits.
- [x] B4.3 Gate visuel utilisateur VALIDÉ le 2026-08-28 (témoins présentés : CTF
      drapeau neutre 2/13 11→21, contre-témoin VIP 11/5 90→77, DNF sans note, Ranked
      Oddball 28,7→42,4, vignettes kill feed).
- [x] B4.4 Recompute LUSR complet des 4 joueurs (`RecomputeLUSRCanonicalForPlayer`,
      chemin canonique v2 — réutiliser l'orchestration existante type
      `recompute_after_art_rebuild.go` si adaptée) APRÈS la reclassification 1bis ;
      contrôles : comptes `match_skill_rank_latest` par chaîne avant/après (~25 sociaux
      déplacés arena_slayer → arena_objectif), aucune perte de lignes, watermarks sains.
- [~] B4.5 Balayage PSA des DBs de PROD (VPS) — REQUALIFIÉ par décision utilisateur du
      2026-08-28 : « on ne refait pas tout sur le VPS, je migrerai les BDD » — les DBs
      locales (index réparés, notes et LUSR recalculés, 80 scores corrigés, killsource
      100 %) deviennent la source de vérité et seront copiées vers la prod par
      l'utilisateur. Aucune opération VPS par le pilote. L'outil `repair_psa_index
      -dry-run` reste le détecteur si une vérification prod ponctuelle est voulue un
      jour.
- [x] B4.6 (2026-08-28) Registre des reports FAIT : entrées killsource (:43) et jour-J
      80 scores (:418) SOLDÉES avec résultats ; 4 entrées ajoutées (BTB D-F,
      buildFormTab D-G, cause racine h5_arena, cause racine index PSA — les deux
      dernières avec « prod couverte par la migration des DBs »). C4/D4 couverts par
      ces soldes + thought_log du 2026-08-28. MERGE dans feat/v75 autorisé par
      l'utilisateur SANS attendre le tag (décision 2026-08-28) — exécuté par le pilote
      avec delivery-checklist.

**B4.1/B4.2/B4.4 EXÉCUTÉS ET VALIDÉS le 2026-08-28** (backup 429,6 Mo vérifié — dont la
shared : `RecomputeLUSRCanonicalForPlayer` écrit AUSSI en shared (sentinelle +
player_skill_state_v2, lusr_full_recompute.go:44) — écart de périmètre assumé et
documenté ; le mode perf garde la shared en read_only). Point d'entrée :
`cmd/recompute_perfnote` (jetable) + wrapper `RecomputePerformanceScoresWithMedals`
(performance.go, chemin backfill AVEC médailles), seams câblés + validés fail-fast +
3 gardes anti-no-op LUSR (SlugHasLUSR/IsLUSRV2Enabled/IsLUSRV2Canonical). RÉSULTATS,
contre-sondés par le pilote : purge = prédiction À L'UNITÉ 4/4 (78/41/118/10) ; zéro
note sur outcome=4 ; zéro chaîne `ranked` ; aucune perte de ligne (arithmétique
append-only exacte) ; 26 movers confirmés perf ET LUSR (concordance
performance_chain↔playlist_group : 0 divergence) ; médianes dans [45,55] hors 2 cas
pré-existants documentés (chaos Madina 41,8 amélioré ; ranked_slayer n=2) ;
ranked_objectif Madina 38,8 → 45,25 ; oracle sentinelle diag_perfsim : purge=0,
1041/502/1128/12 notes = stockées, écarts médians ±1,2 pt (effet medal_exploit
attendu). Témoins gate visuel prêts (section 6 du CR lot 4) — B4.3 après volets D/C,
serveur relancé.

## Volet C — backfill-killsource (demande utilisateur 2026-08-27, solde le registre lot 6 item 6.5)

Entrée du registre `.ai/V7.5/REGISTRE_REPORTS.md:43` : re-run `backfill-killsource`
COMPLET (couverture arme-du-kill 0-5 % sur les matchs récents avril-juillet 2026).
Condition de reprise remplie : serveur arrêté avec accord utilisateur (fenêtre du
lot 4). SÉQUENCEMENT STRICT : APRÈS la fin des recomputes du lot 4 (le backfill tient
la shared en ÉCRITURE ; aucune concurrence de process sur les DBs ni d'invocations go
parallèles).

- [x] C1 (2026-08-28, pilote) Dry-run exécuté : 2 films en cache à décoder (33 chunks
      chacun) + 1948 matchs à examiner en passe crédit — charge légère, --limit inutile.
- [x] C2 Passe complète : EXIT 0 en 7 min 54 — crédit 416 écrits + 949 enrichis par un
      film (1365 matchs), 136 900 morts créditées, 583 sans événement, 0 erreur. Les 2
      films du cache étaient INCOMPLETS (compteur films_absents=2, 0 mort issue du
      décodage film — sans impact, le crédit SQL a porté le rattrapage). Compteurs de
      santé tous à zéro d'erreur ; assist_extra et orphelins humain-contre-humain dans
      leurs seuils documentés.
- [x] C3 Contrôle de couverture par mois (read-only) : 2026-03 99,5 % (1 match sans
      événement), 2026-04→08 **100 %** — contre 0-5 % avant sur avril-juillet. Le volet
      visuel (vignettes du kill feed sur un match récent) rejoint le gate visuel
      utilisateur groupé.
- [ ] C4 Solder l'entrée du registre (:43, date + résultat) ; thought_log — au lot de
      clôture B4.6.

## Volet D — backfill des 80 scores d'équipe faux (demande utilisateur 2026-08-28, IMPÉRATIF avant le tag v7.5.0)

Mode d'emploi source : `.ai/V7.5/replay2d/RAPPORT_QUALITE_SCORE_EQUIPE.md` (CLI
`cmd/backfill-team-scores`, JAMAIS exécutée à ce jour — ni dry-run ni apply). Résidu
API 343 pré-mai 2026 : 80 matchs à scores d'équipe faux (participants corrects), dont
7 inversions exactes des deux camps (6 Oddball + 1 BTB:Sentry Defense) et 1 cas
inversion+ticks. Le dry-run fait 80 APPELS API (tokens valides requis) et lit la
shared ; l'apply ouvre la shared en RW sous lease → serveur arrêté OBLIGATOIRE (piège
connu registre :418 : un --apply avec serveur allumé sort en fatal AVANT le résumé et
brûle les 80 appels). SÉQUENCEMENT : après le lot 4, AVANT le volet C (court et
impératif d'abord), même fenêtre serveur-arrêté, jamais en concurrence d'accès DB.

- [x] D1 (2026-08-28, pilote) Dry-run : EXIT 0, `lus=80 identiques=0 planifiees=80
      skippes=0 echecs=0` — décompte de référence exact, journal capturé.
- [x] D2 CONTRÔLE DE FORME VALIDÉ : les 7 inversions du journal sont des permutations
      EXACTES et correspondent une à une aux 7 attendues du TSV cause=autre (6 Oddball
      + 1 BTB:Sentry Defense, mêmes api_t0/api_t1) ; les non-permutations du journal
      sont précisément les cas documentés hors des 7 (Attrition 2/1950→2/0 ticks,
      Strongholds à ticks, 2 One Flag). Pas de STOP.
- [x] D3 Apply : EXIT 0, `planifiees=80 corriges=80 skippes=0 echecs=0` ; contrôle
      final (dry-run rejoué) : `identiques=80 planifiees=0`, plus d'avertissement de
      répétition à blanc. **Pré-requis « avant tag v7.5.0 » LEVÉ.**
- [ ] D4 Thought_log + entrée du registre (:418) soldée — au lot de clôture B4.6.

## Hors périmètre / interdits (exécuteurs)

- Aucun commit/push/stage par un exécuteur ; le pilote commit après revue, avec
  autorisation utilisateur.
- Aucune écriture dans le checkout principal ni dans `.ai/` partagé (thought_log
  INTERDIT aux exécuteurs) ; rapport de lot dans le worktree.
- Zéro fix opportuniste (découverte → section Découvertes du CR, pas de correctif).
- Pas de nouveau Python, pas de SQLite, pas d'emoji, seuils 500 L/80 L respectés.
- DBs réelles : lecture seule exclusivement, serveur :8000 arrêté vérifié avant.
- LUSR/CSR intouchés (la scission ne concerne QUE la chaîne de perf).

## Découvertes (à remplir en exécution — ne pas traiter)

- (diagnostic 2026-08-27) `buildFormTab`/`ComputePerformanceSeries` : chemin API `form`
  sans consommateur UI (confirmé utilisateur) — candidat suppression, décision D-G.
- (diagnostic) 87 matchs pair_name NULL dans match_registry → fallback arena_slayer.
- (diagnostic) `match_exclusion_service` ne nettoie pas la note (couvert par D-E au
  batch suivant).
- (lot 0, 2026-08-27, VÉRIFIÉE pilote) Index ART `idx_psa_match` incohérent sur la DB
  XxDaemonGamerxX (2 lignes indexées vs 4 au scan) → item B2.4. Le lecteur de prod
  `PersonalScoreAwardsRepo` (colonne Score personnel) est exposé dès AUJOURD'HUI sur
  cette DB.
- (lot 0) 26 matchs objectifs classés slayer par la liste actuelle (affecte aussi les
  chaînes LUSR depuis toujours) → décision D-I RÉVISÉE par l'utilisateur le 2026-08-27 :
  correction dans ce chantier (lot 1bis) + recompute LUSR au lot 4.
- (lot 0) DB JGtm : 15 lignes `personal_score_awards` avec xuid vide sur 6 matchs
  (filtrées par la sélection sur xuid — sans effet sur les calculs).
- (lot 0) L'annexe corpus du plan disait Madina 22 objectif / 12 slayer en ranked ; la
  classification fidèle au code donne 21/13 (l'écart = `Ranked:CTF 3 Captures`, cf.
  D-I). Totaux ranked 50 et comptes JGtm 8 / Choco 8 / Daemon 0 confirmés.
- (lot 2, 2026-08-27) La corruption d'index PSA touchait LES 4 DBs locales (pas
  seulement Daemon), sur idx_psa_match ET idx_psa_category — réparée (B2.4), mais la
  CAUSE RACINE de la désynchronisation n'est pas élucidée (famille bug ART, hors
  périmètre). Piste d'investigation future : reproduire sur une DB fixture avec le
  pattern d'écriture INSERT+tombstone de personal_score_awards.
- (lot 2) Ratchet `TestSyncRootPackageFrozen` : `internal/sync/` gelé à 80 fichiers
  racine non-test — tout fichier neuf y est refusé. Impact lots 3-4 : étendre les
  fichiers existants (performance*.go) plutôt que d'en créer.
- (lot 2) `batchComputePerformanceScores` : 189 → 180 L (dette funlen réduite, pas
  accrue). Commentaire orphelin préexistant en fin de performance_helpers.go (doc
  détachée de BatchComputePerformanceScores) — non traité.
- (lot 4, 2026-08-28) **MAJEURE — corruption LUSR `h5_arena`** : 2 461 lignes
  match_skill_rank_latest des 4 DBs halo_infinite (jusqu'à 80 % de l'historique LUSR,
  JGtm 895 / Madina 1064 / Choco 471 / Daemon 31) portaient la chaîne Halo 5
  `h5_arena`, écrites dans la fenêtre 26-28 juin 2026 — signature d'un binaire h5 (ou
  classifier h5 par défaut) exécuté sur les données Infinite. RÉPARÉE à 99,9 % par le
  replay canonique du lot 4 (résidu : 2 lignes Madina, matchs à équipes non binaires,
  non rejouables). Cause racine + exposition PROD à investiguer (registre B4.6 ; le
  balayage VPS B4.5 vérifiera aussi `playlist_group='h5_arena'` en read-only).
- (lot 4) `RecomputeLUSRCanonicalForPlayer` écrit dans la SHARED (sentinelle + états
  skill), pas seulement la player DB — à retenir pour tout futur périmètre.
- (lot 4) Replay LUSR : WARN « EP n'a pas convergé en 500 itérations » sur 11 matchs
  JGtm (équipes déséquilibrées 4v8), sautés — comportement pré-existant identique au
  post-sync.
- (lot 4) Texte narratif de la section concordance de diag_perfsim figé au lot 0
  (« écart 0.00 » alors que ses tableaux recalculés disent 0,71) — cosmétique,
  trompeur si l'outil resert.

## Protocole de reprise

1. Lire ce fichier (statuts des cases) + le dernier CR d'exécuteur dans la conversation
   pilote + `git -C <worktree> status`.
2. Reprendre AU PREMIER item non coché du lot courant ; ne jamais ouvrir le lot N+1 si
   le gate N n'est pas consigné ici comme passé.
3. Poids figés au gate 0 : consignés dans D-C (mise à jour datée si le lot 0 les change).

## Annexe — chiffres corpus (2026-08-27, 4 joueurs, 2 948 participations)

- Ranked : 50 participations (Madina 34 = 22 obj/12 slayer ; JGtm 8 ; Choco 8 ; Daemon 0).
- Arena sociale (médianes/min) : pspm 200 (obj) vs 156 (slayer), kpm 1.30 vs 1.25,
  dpm 1.45 vs 1.30 ; ranked : pspm 206 vs 161, kpm 1.36 vs 1.17.
- Intra-famille objectif : pspm CTF 190 / SH 201 / KOTH 233 / Oddball 244 ;
  volumes 355 / 154 / 68 / 19.
- personal_score JGtm ≈ 95 % combat (kills+assists 1,52 M pts vs objective 85 k).
- `personal_score_awards` : 1 101/1 120 matchs JGtm ; catégorie `objective` explicite.
- Fuites : 33 notes sur DNF + 45 orphelines sous-seuil (JGtm) ; outcome=4 ⇔ quitté
  (116 + 1 cas limite) ; enrichment JGtm : arena_slayer 414, chaos 408,
  arena_objectif 211, chain NULL 87 (78 scorées).
