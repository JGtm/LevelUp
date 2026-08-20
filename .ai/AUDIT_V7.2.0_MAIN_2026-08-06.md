# Audit du diff v7.2.0 -> main (634e079f2) — 2026-08-06

## Cadrage

**Objet** : audit rétrospectif de tout ce qui a été livré entre le tag `v7.2.0`
(d4851af83, 2026-07-25) et `634e079f2` (2026-08-05, HEAD de `main` au moment de l'audit).
274 commits (241 hors merges), 1155 fichiers de code applicatif, ~129k insertions.

**Chantiers couverts par la plage** (chronologie des merges) :
- v7.2.1 (chantier Notion V721-01..15) + v7.2.5 (2026-07-26) ;
- quick wins post-v7.2.1, batch v7.3 Notion + lot 2 (prolongations, graphe premier
  frag/première mort, rotation des logs, 8 anomalies), CI hardening, échantillons démo
  Prestige, bump echarts 6.1.0 (CVE-2026-45249), palette escouade ;
- vagues post-v7.3.0 (2026-08-04/05) : autorité de schéma player DB, suppression des
  index ART résiduels `idx_psa_*`, correctifs Explorer (placement, badge unranked_N) ;
- merge `feat/replay2d-prod` (2026-08-05) : killfeed visible + re-mode-score (C-bis) +
  lot d'hygiène H1-H6.

**Périmètre** : fichiers touchés par la plage sous `apps/go-api/`, `apps/web/src/`,
`tools/`, `config/`. Exclus : dumps RE (`.ai/V7.5/`, `tools/ce/`), `static/`, artefacts
générés. Règle d'attribution : un défaut dont les lignes préexistent à v7.2.0 est hors
périmètre (dette antérieure, écartée en le notant).

**Axes passés** : A1 (code mort), A2 (invariants ART, 2 auditeurs), A3 (multi-titre),
A4 (frontières de couches), A6 (erreurs avalées), A8 (couverture de tests),
A9 (correction des données), A10 (front).
**Axes NON passés** : A5 (duplication/factorisations) et A7 (flags/compatibilité) —
non couverts par un auditeur dédié, partiellement balayés par A4/A10 (ratchets,
kill-switch MULTI_TITLE_API_ENABLED vérifié retiré proprement). À passer dans une
campagne ultérieure si besoin.

**Méthode** : 9 auditeurs parallèles aveugles (un axe chacun, doctrine collée, filtre de
recevabilité fichier:ligne + règle citée + conséquence + repro), puis vérification
adverse de chaque P0/P1 par 7 réfutateurs frais à consigne inversée (« établis que le
constat est FAUX ; en cas de doute, réfuté »). Les P2 ne passent pas la vérification
adverse (grille du skill). Limites : les tests garde-rails n'ont pas été exécutés
(compilation > 5 min dans l'environnement d'audit) — vérifications sur pièces ;
quelques points marqués « non vérifié » dans les rapports sont listés en Suite.

**Doctrine de référence** : CLAUDE.md (règles 1-16, règles critiques DuckDB,
anti-patterns 1-10), ADR 0006/0011/0012/0013/0016/0019/0022/0023/0026/0030/0031,
skills `arch-rules`, `db-schema`, `color-tokens`, `frontend-patterns`.

---

## Constats retenus

### [P0] Régression : les modes à objectifs ne reçoivent plus jamais de flag de dominance
- Où : `apps/go-api/internal/sync/comeback.go:75-91` (routage par type d'objectif) ;
  commit `4665794f8`.
- Règle : résultat faux servi à l'UI (grille P0). Avant la plage, tout match (CTF inclus)
  passait par le contrôle Steaktacular -> `DominanceFlagDomination`/`Humiliation` ;
  après, CTF est routé vers `computeCTFDominanceFlag` qui lit `match_objective_events`
  — table dont l'UNIQUE écrivain est `cmd/diag_weapons_v3 -write` (CLI manuel, « HORS
  chemin live » écrit dans l'en-tête de sa migration). Aucun backfill, aucune étape de
  sync ne la peuple. Courbe vide -> flag 0 sec, sans fallback.
- Conséquence : tout match CTF nouvellement syncé reçoit `dominance_flag = 0` définitif
  (état terminal exclu du re-calcul, `engine_postsync_csr.go:245-247`). Le court-circuit
  `case zone/hill/skull` (comeback.go:83-90) fige AUSSI Strongholds/KOTH/Oddball à 0 —
  la classe de régression dépasse le CTF. Le flag est servi sur au moins 6 surfaces UI
  (Explorer, Match View header, Sessions, highlights Carrière, OutcomeSequenceTape,
  home) via `lib/narrative/dominance.ts`. Les matchs flaggés avant la plage gardent
  leur valeur — la perte porte sur les nouveaux syncs et le backlog non flaggé.
- Aggravant : la régression est verrouillée par un test —
  `comeback_objective_test.go:82-97` sème une médaille Steaktacular et EXIGE flag=0
  (« le chemin CTF ne doit PAS la consulter »).
- Repro : `grep -rn "INSERT INTO match_objective_events" apps/go-api --include='*.go'`
  (un seul écrivain, cmd/) ; `git show v7.2.0:apps/go-api/internal/sync/comeback.go`
  (absence de routage par mode avant la plage).
- Vérification adverse : TIENT (recherche exhaustive des écrivains, comportement
  pré-plage reconstitué, consommateurs UI listés).
- Traitement proposé : décision produit à trancher AVANT tout code — soit fallback vers
  l'ancien chemin (Steaktacular/marge) quand la courbe est vide, soit peuplement de
  `match_objective_events` par le pipeline live, soit assumer et documenter la perte.
  | Décision : **escalade utilisateur** (contrat produit + le test verrouille le
  comportement inverse).

### [P1] `computeCTFDominanceFlag` avale l'erreur de lecture et persiste un 0 indiscernable
- Où : `apps/go-api/internal/sync/comeback.go:170-171` (`return 0, nil`).
- Règle : CLAUDE.md règle 3 (« jamais d'erreur avalée en silence : logger AVANT toute
  dégradation best-effort ») + anti-pattern n°10.
- Conséquence : l'erreur de `loadCaptureEvents` (table absente — `EnsureSharedSchema` ne
  crée PAS cette table, seule la migration `shared_objective_events_v1` le fait —,
  verrou, invalidation) est effacée ; le chemin de sécurité
  `comeback_postsync_persist.go:40-43` (WarnContext + continue, qui aurait retiré le
  match du lot et permis un retry) n'est jamais atteint ; le 0 est écrit
  (`BatchUpdateColumn`, stage `dominance`) sans filtre, indiscernable d'un calcul
  légitime, sans trace ni retry.
- Repro : `grep -n -B3 "non critique : pas de courbe" apps/go-api/internal/sync/comeback.go`
- Vérification adverse : TIENT (aucun log amont, aucun garde, état terminal confirmé ;
  nuance : le motif `return 0, nil` préexiste sur des chemins secondaires — ici il
  couvre l'unique source de courbe CTF).
- Traitement proposé : logger l'erreur puis la propager (le chemin de sécurité existe
  déjà en aval). | Décision : à planifier (avec le P0 ci-dessus, même fonction).

### [P1] Quatre tables shared à index ART écrites en DELETE-then-INSERT, hors persist et hors garde-rail, justification pointant vers un document inexistant
- Où : `apps/go-api/internal/platform/duckdb/objective_events_repo.go:90,94`,
  `objective_score_repo.go:75`, `weapon_kills_v3_repo.go:92` ; DDL
  `internal/migration/steps_shared_objective_events.go:35,43` (PK `match_id VARCHAR` en
  tête), `steps_shared_objective_score.go:37`, `steps_shared_weapon_kills_v3.go:47`
  (CREATE INDEX).
- Règle : règle critique DuckDB n°1 (écriture per-match sur DB partagée via
  `persist.BatchBuilder`, INSERT-only) et n°3 (garde-rail `tablesProtegees`).
- Conséquence : DELETE sur index ART = la configuration qui a déjà FATAL-invalidé des
  DBs (bug #23046). L'allowlist du garde-rail exige « PK BIGINT, pas VARCHAR » pour
  tolérer un DELETE brut, et le crash du 2026-06-20 a eu lieu MALGRÉ mono-writer + PK
  BIGINT — les 4 tables (VARCHAR en tête) sont dans une configuration strictement plus
  faible. `internal/platform/duckdb/` EST scanné par `TestNoRawDeleteOnAppendOnlyTables`
  (seuls `_test.go`, `/migration(s)/`, `/cmd/`, `/scripts/` sont exclus) : le trou vient
  uniquement de `tablesProtegees` où aucune des 4 ne figure (`weapon_kills_v3` échappe
  même à `criticalMatchTables` par l'ancrage `\b` de la regex `weapon_kills`). La
  justification des 5 en-têtes cite `.ai/PLAN_WEAPON_ATTRIBUTION_V3.md §10` — document
  ABSENT du repo (fichier utilisateur non tracké) — et ne couvre que la pression
  concurrente, pas la suppression d'entrées d'index (le déclencheur documenté).
- Ce qui borne : le port `ObjectiveEventsRepository` n'expose que `LoadMatch` — le
  serveur ne peut pas appeler `WriteMatch` ; seul `cmd/diag_weapons_v3` écrit, et
  `cmd/` est légitimement hors garde-rail. Risque LATENT : il se matérialise le jour où
  le port est élargi ou le backfill branché in-process, et ce jour-là aucun test ne mord.
- Aggravation (vérification adverse) : 5e table au même pattern,
  `match_player_positions` (`player_positions_repo.go:54+`), même justification vers le
  même document inexistant — sans index (pas de surface ART) mais même dette.
- Repro : `grep -n "DELETE FROM" apps/go-api/internal/platform/duckdb/objective_events_repo.go apps/go-api/internal/platform/duckdb/objective_score_repo.go apps/go-api/internal/platform/duckdb/weapon_kills_v3_repo.go`
- Vérification adverse : TIENT AVEC NUANCE (toutes les pistes de réfutation échouées ;
  latence confirmée).
- Traitement proposé : ajouter les 4 tables à `tablesProtegees` (avec allowlist datée
  pour le chemin cmd/ si nécessaire) ; recopier la justification §10 dans le repo ou la
  réécrire ; à terme basculer les writers sous persist. | Décision : à planifier
  (P0 différé : le jour du câblage live, c'est une corruption).

### [P1] Décodeur de score de mode : validé uniquement par des fixtures construites avec ses propres constantes
- Où : `apps/go-api/internal/analysis/objectivescore/decode_test.go:80-84,185-193` ;
  constantes cibles `decode.go:56-66`.
- Règle : A8 doctrine 4 (fixtures auto-validantes) + 2 (test vert avec et sans le code).
- Conséquence : la substance du décodeur est la position de bit (`shOffTeam0=24`,
  `tokenWinLo=835`...). L'écrivain de fixture consomme les mêmes constantes que le
  lecteur : les décaler laisse la suite verte. Un offset faux produit une courbe
  entièrement fausse écrite dans `shared.match_objective_score_timeline`, plausible
  (calibrée sur le final DB : la fin est juste, le milieu est du bruit).
- Repro : `cd apps/go-api && sed -i 's/shOffTeam0   = 24/shOffTeam0   = 23/' internal/analysis/objectivescore/decode.go && go test ./internal/analysis/objectivescore/ ; git checkout internal/analysis/objectivescore/decode.go` -> suite VERTE.
- Vérification adverse : TIENT (aucun testdata/, aucun golden ; seule la valeur du token
  est contrainte par 2 octets figés, aucun offset de champ).
- Traitement proposé : versionner un chunk réel (le patron minibobine existe) et un
  golden. | Décision : à planifier (avec le constat suivant, même remède).

### [P1] Le seul test du score de mode sur film réel est éteint par un chemin absolu mort
- Où : `apps/go-api/internal/analysis/objectivescore/cache_backed_test.go:20` (const
  `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`), skip `:88`.
- Règle : A8 doctrine 5 (garde-rail qui ne tourne pas = décoratif) + 1.
- Conséquence : répertoire inexistant (les trois niveaux), const sans repli env
  (contrairement à `FILM_CACHE_ROOT`/`KILLSOURCE_FIXTURES`/`REPLAY_FILM_DIR` ailleurs) ;
  `TestDecodeStrongholds_CacheBacked_7344d24f` — les seules assertions sur données
  réelles (finals 193-112) — skippe sur 100 % des machines et en CI. Combiné au constat
  précédent : AUCUN octet de film réel n'est exécuté contre ce décodeur.
- Repro : `cd apps/go-api && go test -v -run CacheBacked ./internal/analysis/objectivescore/ 2>&1 | grep -i skip`
- Vérification adverse : TIENT.
- Traitement proposé : repli sur variable d'env comme les autres tests film, ou fixture
  versionnée. | Décision : à planifier.

### [P1] Le « verrou d'exhaustivité » des voies de lecture film est tautologique
- Où : `apps/go-api/internal/sync/killcollector/film_read_paths_test.go:52,58`.
- Règle : A8 doctrine 2 (test vert avec et sans le code).
- Conséquence : le test compare `killscope.FilmReadPaths()` (littéral) à `duDecodeur`
  (littéral) — aucun des deux ne dérive du décodeur. Ajouter une 3e voie (`PathHybrid`)
  dans `killsource/kill.go` laisse tout vert (cardinaux 2==2) ; le ratchet archlint
  voisin a une liste fermée sans « hybride » ET exempte `killsource/`
  (`killScopeOwners`). Or le fichier documente lui-même la conséquence d'une voie non
  enregistrée : elle « passe pour du crédit et se fait écraser par le producteur
  suivant » — perte de source de dégât fatal dans `shared.match_kill_events`, invisible
  par comptage. Ironie : le test se présente comme la réparation du constat J4R-3 et
  reproduit le défaut qu'il prétend fermer (le verrou d'égalité tient, celui
  d'exhaustivité non).
- Repro : `cd apps/go-api && printf '\nconst PathHybrid Path = "hybride"\n' >> internal/games/halo_infinite/film/killsource/kill.go && go test ./internal/sync/killcollector/ ./internal/archlint/ ; git checkout internal/games/halo_infinite/film/killsource/kill.go` -> VERT.
- Vérification adverse : TIENT (pas de réflexion/AST ; `killscope_test` n'épingle que
  les voies crédit).
- Traitement proposé : énumération par réflexion ou ratchet regexp sur les
  déclarations `Path = ` du paquet. | Décision : à planifier.

### [P1] Overlay « captures CTF » sur l'horloge film, courbe K/D sur l'horloge gameplay — deux référentiels sur le même axe
- Où : écriture `analysis/objectiveevents/extract.go:182` (TimeMS = horloge film,
  countdown inclus) ; lecture sans recalage `platform/duckdb/objective_events_repo.go:165-188`,
  service `match_view_service.go:401-406`, handler (recopie verbatim) ; front
  `_objectiveCaptures.ts:67`, `MatchKDCumulChart.tsx:120,262-263,278`,
  `MatchTugOfWarChart.tsx:415-418`. À comparer : `match_view_data_loaders.go:253-264`
  recale les kill events (`correctMatchViewEventsT0`), golden lock
  `match_view_t0_golden_test.go:104-126`.
- Règle : A9 doctrine 7 (cohérence des unités/référentiels de la chaîne) — le commentaire
  de `correctMatchViewEventsT0` énonce lui-même le risque d'« incohérence visuelle sur
  la même page ».
- Conséquence : les verticales de capture sont décalées du countdown (~10-30 s, plafond
  120 s) vers la droite par rapport à la courbe qu'elles annotent ; le tooltip
  « Capture à M:SS » affiche l'instant film, différent de tous les autres marqueurs du
  même graphe. T0 est exactement le countdown mesuré sur l'horloge film
  (`t0_ms = epoch(real_start_time) − epoch(start_time)`), donc l'écart est structurel.
- Ce qui borne : visible uniquement sur les matchs backfillés via
  `diag_weapons_v3 -write` (sinon 503 -> overlay absent) ; cas `t0_ms` NULL sain.
- Repro : voir chaîne dans le rapport A9 ; vérification front : grep `t0` sur
  `apps/web/src/features/match-view` -> zéro occurrence (recalage impossible côté client,
  le DTO ne porte pas T0).
- Vérification adverse : TIENT (chaîne suivie de bout en bout, aucun recalage
  compensatoire nulle part).
- Traitement proposé : recaler côté service (comme les kill events) ou exposer
  `t0_ms` dans le DTO et recaler au tracé. | Décision : à planifier.

### [P1] Décodeurs Halo-Infinite-only sous `internal/analysis/` (contre ADR 0012), import title-specific dans le port cross-titre
- Où : `internal/analysis/filmdec/` (créé par la plage), `internal/analysis/positions/`,
  `internal/analysis/replay/` ; import `analysis/positions` dans
  `internal/port/services.go:152`, `internal/service/match_view_service.go:26`,
  `internal/api/handlers/match_view_positions.go:18,35`.
- Règle : ADR 0012 (« Extraire toute la logique Halo-only de internal/analysis/ vers
  internal/games/halo_infinite/ ») dont la lecture large est confirmée par ADR 0031
  (« Halo-only code lives under internal/games/<slug>/ », appliqué à un non-adaptateur
  avant la plage).
- Conséquence : incohérence de frontière — le même chantier a placé `killsource` et
  `damagetag` au bon endroit (`games/halo_infinite/film/`) ; `filmdec`/`positions`/
  `replay` non, sans aucune justification écrite (grep « ADR 0012 » sur `.ai/V7.5/` :
  les seuls hits pointent vers le placement correct). Le garde-rail archlint
  `no_temporal_title_import_test.go` n'a pas été étendu aux paquets neufs.
- Ce qui borne (vérification adverse) : `positions.PlayerPosition` est
  `{TimeMS; X,Y,Z; Team}` — 100 % générique, canonique de fait : le préjudice est
  l'EMPLACEMENT DE PAQUET, pas un couplage de titre dans la signature. Atténuation :
  transplantation verbatim d'une branche antérieure — la décision de placement a été
  prise hors plage et héritée sans réexamen.
- Repro : `grep -n "analysis/positions" apps/go-api/internal/port/services.go` ;
  `head -3 apps/go-api/internal/analysis/filmdec/doc.go`
- Vérification adverse : TIENT AVEC NUANCE (sous-affirmation « fuite de type » réfutée
  en gravité).
- Traitement proposé : déplacer les 3 paquets sous `games/halo_infinite/film/` (ou ADR
  d'exemption écrite) + étendre le ratchet. | Décision : **escalade utilisateur**
  (décision d'architecture — un déplacement de paquets se planifie).

### [P1] Quatre fichiers neufs > 500 lignes sans exemption commentée
- Où : `internal/analysis/filmdec/traverse.go` (1251 L), `unit_weaponstate.go` (791 L),
  `frame_records.go` (784 L), `components_biped_ability.go` (634 L) — tous créés par la
  plage.
- Règle : CLAUDE.md règle 5 (fichier <= 500 L, exemption commentée sinon ; « ne pas
  accroître » la dette) + anti-pattern n°3.
- Conséquence : ~3460 lignes hors seuil en un lot, aucun en-tête ne porte de
  justification de seuil même reformulée (vérifié mot à mot par la vérification
  adverse) ; aucun linter de longueur de FICHIER n'existe dans `.golangci.yml` (le
  contrôle a été perdu dans la traduction `enforce_size_limits.py` -> funlen+lll) —
  seule la règle CLAUDE.md le tient, et le repo sait écrire l'exemption canonique
  (cf. `.golangci.yml:230-238`, « exemption justifiée » K3f) : il n'y en a pas ici.
- Repro : `git diff --name-status v7.2.0..HEAD -- apps/go-api/internal/analysis | awk '$1=="A" && $2 ~ /\.go$/ && $2 !~ /_test/{print $2}' | xargs wc -l | sort -rn | head -6`
- Vérification adverse : TIENT.
- Traitement proposé : découper (la structure 1 fichier = 1 plage de composants s'y
  prête) ou poser l'exemption justifiée datée. | Décision : à planifier (bas risque,
  peut accompagner le déplacement ADR 0012).

### [P1] String UI française en dur dans un composant partagé du shell (ligne réécrite par la plage)
- Où : `apps/web/src/components/shell/_filter_pills/CheckboxGroup.tsx:144`.
- Règle : règle 1 + ADR 0003 (toute string UI en FR ET EN via manifeste).
- Conséquence : tooltip « 0 match si cette option est cochée » servi en français aux
  anglophones, sur le chemin nominal du progressive disclosure (options `count=0`
  dépliées). La plage a réécrit ce littéral sans le porter au manifeste ; `OptionRow`
  n'a pas la locale en props alors que le fichier a déjà `t()` (l.41, utilisé l.116).
  Aucune clé existante ne couvre ce texte (vérifié).
- Repro : `git diff v7.2.0..HEAD -- apps/web/src/components/shell/_filter_pills/CheckboxGroup.tsx`
- Vérification adverse : TIENT.
- Traitement proposé : clé `common.filters.*` + passer la locale à OptionRow.
  | Décision : à planifier (quick win).

### [P1] Dictionnaire i18n inline dans un composant neuf, hors garde-rail anti-anglicismes
- Où : `apps/web/src/features/match-view/MatchPositionsHeatmap.tsx:40-57`
  (`const TEXT = { fr, en } as const`).
- Règle : règle 1 + ADR 0003 (strings dans `i18n.ts`, parité `Record<Locale, T>`).
- Conséquence : (a) le garde-rail `no-anglicisms.guard.test.ts` est une liste FERMÉE
  (aucun glob — vérifié sur les 239 lignes) qui scanne `MATCH_VIEW_TEXT.fr` mais pas ce
  composant : ses FR ne sont jamais scannés ; (b) parité non contractualisée (une clé
  inutilisée d'un seul côté passe) ; (c) `features/match-view/i18n.ts` était modifié
  par la même plage (+94 L) — le domicile naturel était ouvert. Seul `const TEXT = {`
  de tout `apps/web/src` : divergence isolée, pas une convention.
- Repro : `grep -rn "const TEXT = {" apps/web/src/features --include=*.tsx`
- Vérification adverse : TIENT (nuance : les clés LUES cassent la compilation si une
  branche manque — parité partiellement structurelle).
- Traitement proposé : déplacer les 6 clés dans `MATCH_VIEW_TEXT`. | Décision : à
  planifier (quick win).

### [P1] Notification `media_liked` : erreur de factory et erreur d'émission avalées sans log
- Où : `apps/go-api/internal/api/handlers/media_likes.go:157-160` (return nu) et `:167`
  (`_ = em.Emit(...)`).
- Règle : règle 3 + anti-pattern n°10.
- Conséquence : 200 OK, like persisté, notification perdue sans trace. Vérifié
  implémentation en main : `notifierFor` ne loggue RIEN en interne (chaîne complète
  factory -> resolve -> `fmt.Errorf` sans slog) ; `Emit` ne loggue qu'un seul de ses
  chemins d'erreur et son contrat (service.go:124) délègue explicitement le log à
  l'appelant — validation, encodage, catégorie, lease : aucune trace. Les voisins
  `sync_handler.go:267,279` et `settings.go:363,376` logguent ces deux points.
- Repro : `grep -n -A3 "em, err := h.notifierFor" apps/go-api/internal/api/handlers/media_likes.go`
- Vérification adverse : TIENT (nuance de cadrage : `media.go:271-273`, préexistant,
  fait pareil — le code neuf a copié son voisin ; la « convention » du package est
  2 sites sur 4).
- Traitement proposé : 2 `slog.WarnContext` alignés sur sync_handler ; traiter
  `media.go:271-273` dans la foulée (même correctif, hors plage). | Décision : à
  planifier (quick win).

### [P1] `filmdec/probe_export.go` : 11 exports publics dont les consommateurs déclarés n'existent plus
- Où : `internal/analysis/filmdec/probe_export.go:1-111` (fichier entier, 11 exports +
  la const `VariantBitOffsetInWST`).
- Règle : règle 7 (0 code mort) + anti-pattern n°1 (dead code museum).
- Conséquence : wrappers publics sur des consumers privés — toute refactorisation les
  casse pour rien ; la surface publique fait croire qu'un harness de validation existe.
- Vérification adverse : TIENT, et le constat était SOUS-évalué — le fichier nomme SIX
  consommateurs disparus (`cmd/wf_c_traverse`, `cmd/tmp_widthcmp`,
  `cmd/tmp_defstate_measure`, `cmd/tmp_residual`, `cmd/tmp_deadstate`,
  `cmd/tmp_bipedscan`), aucun n'existe, y compris dans les zones gitignorées (vérifié
  `git ls-files --others` : zéro fichier Go non suivi dans le repo). Zéro test (le seul
  candidat appelle la version productisée `ScanFilmKeyframeLoadouts`).
- Repro : `rg -n "ConsumeWeaponStateTypeInfoVariantAt|TraverseKeyframeBipedAt" apps/go-api --glob '!probe_export.go'`
- Traitement proposé : suppression. | Décision : à planifier.

### [P1] `filmdec/frame_debug.go` : surface de diagnostic complète, zéro appelant — mais un usage futur nommé dans un handoff
- Où : `internal/analysis/filmdec/frame_debug.go:1-159` (`ScanFrameBipeds`,
  `DebugDecodeFrame`...).
- Règle : règle 7 + anti-pattern n°1.
- Conséquence : 159 L qui dupliquent la traversée de frame — à la première évolution du
  format, le fichier diverge silencieusement du décodeur et devient une référence
  périmée.
- Vérification adverse : TIENT sur le fait (zéro appelant, zéro test, l'outil
  `cmd/tmp_framedump` cité par le journal n'existe plus). Nuance : le handoff
  `.ai/V7.5/film_re/HANDOFF_KEYFRAME_LIVE_CAPTURE.md:419` nomme `DebugDecodeFrame`
  comme critère d'acceptation FUTUR (« une fois le deser porté ») — un usage en
  attente, pas un appelant.
- Repro : `rg -n "ScanFrameBipeds|DebugDecodeFrame" apps/go-api --glob '!frame_debug.go'`
- Traitement proposé : supprimer (git garde l'historique, le handoff peut le
  récupérer) OU le garder en le référençant explicitement depuis le handoff avec date
  de retrait. | Décision : à trancher (arbitrage avec le handoff keyframe).

### [P1] `himap.DescribeRootBlocks` + `RootBlockInfo` : inventaire RE sans appelant ni référence documentaire
- Où : `internal/himap/instances.go:296-372`.
- Règle : règle 7 + anti-pattern n°1.
- Conséquence : ~75 L de parsing sbsp non exercé qui doublent la surface à relire dans
  le paquet le plus fragile (parsing binaire de fichiers du jeu).
- Vérification adverse : TIENT — le plus solide des constats code mort : zéro appelant,
  zéro test, et zéro référence dans `.ai/`, `docs/` ou les plans (contrairement à
  `frame_debug` et `LocalToWorld`). Son commentaire l'annonce « outil de diagnostic »
  mais aucun diagnostic écrit ne s'en sert.
- Repro : `rg -ni --no-ignore "DescribeRootBlocks|RootBlockInfo" .ai docs apps/go-api`
- Traitement proposé : suppression. | Décision : à planifier.

### [P2] La passe kill-events recomposée hérite du `publishable` du film et déclasse la base crédit
- Où : `internal/persist/kill_events_merge.go:185` + miroir SQL
  `internal/migration/steps_shared_kill_events_credit_base.go:136,250,266`.
- Règle : garanties écrites du périmètre (`steps_shared_kill_events.go:204-206`,
  `kill_events_credit.go:113-116`) — les lignes crédit valent `killer_victim_pairs` ;
  les déclarer non publiables retire une capacité existante.
- Conséquence : sur un match à passe film `publishable=FALSE` (BTB, marge de bijection
  nulle), les morts issues de la seule base crédit sortent non publiables -> un lecteur
  conforme afficherait zéro mort nommée là où `killer_victim_pairs` sert tout. LATENT
  (les lecteurs de prod lisent encore `killer_victim_pairs`), mais la colonne est déjà
  écrite fausse, y compris rétroactivement (migration de boot).
- Repro : `grep -n "Publishable: *film.Publishable" apps/go-api/internal/persist/kill_events_merge.go`
- Traitement proposé : `publishable` par population (film vs crédit) dans la passe
  recomposée. | Décision : backlog (à traiter avant la bascule des lecteurs).

### [P2] Axe radar « Objectifs » : erreur de chargement indiscernable de « aucun match à objectif »
- Où : `internal/progression/profile/queries.go:280-281` (`return 0, 0` sur err).
- Règle : règle 3. Conséquence : l'axe disparaît du radar sans log — même sortie que le
  cas légitime. Repro : rapport A6. | Décision : backlog (quick win slog).

### [P2] Suppression de média : échec de `Load()` des réglages dégradé en « base de captures vide », cause perdue
- Où : `internal/api/wire/registry_media.go:91-92`. Règle : règle 3 / anti-pattern 10.
- Conséquence : fichiers laissés sur disque servis par URL directe, message identique au
  cas « jamais configuré ». | Décision : backlog (quick win slog).

### [P2] Harnais de régression visuelle hors de toute CI
- Où : `apps/web/playwright.config.ts:41,45` vs `.github/workflows/ci.yml` (e2e-react =
  `--project=chromium` seul) ; 14 baselines `-win32` versionnées inutilisables sur
  runner ubuntu. Règle : A8 doctrine 5 (garde-rail décoratif).
- Conséquence : régression de rendu ECharts livrable sans signal ; seuils revendiqués
  jamais évalués. | Décision : backlog (job CI dédié ou baselines linux).

### [P2] Le « GATE 1 » du branchement killfeed skippe partout alors qu'une mini-bobine versionnée existe
- Où : `internal/sync/killcollector/collector_test.go:116,150` (`KILLSOURCE_FIXTURES`
  défini nulle part). Règle : A8 doctrines 1 et 6.
- Conséquence : la chaîne décodeur->persister->vue `_latest` (celle qui écrit
  `shared.match_kill_events`) n'a aucun bout-en-bout automatique ; la
  `minibobine_000d5950` (3,8 Mo) versionnée pour `TestGoldenMiniBobine` rendrait le
  gate exécutable. | Décision : backlog.

### [P2] `consumeByName` : fonction neuve de 757 lignes
- Où : `internal/analysis/filmdec/traverse.go:165-921`. Règle : règle 5 (fonction
  <= 80 L). Caveat : l'exclusion `funlen` de `.golangci.yml` couvre `internal/analysis/`
  (antérieure, justifiée « fidélité portage Python » — qui ne décrit pas un portage
  Ghidra). | Décision : backlog (à trancher avec le déplacement ADR 0012).

### [P2] Binning 2D dans le fichier composant
- Où : `apps/web/src/features/match-view/MatchPositionsHeatmap.tsx:68-116`. Règle :
  anti-pattern n°7 (extraire vers `*_logic.ts` — tout le reste de la plage le fait).
  Atténuation : fonction exportée et testée (3 cas). | Décision : backlog.

### [P2] Jeton technique serveur (`coverage.verdict`) rendu brut dans un `title`
- Où : `apps/web/src/features/match-replay/ReplayCoverage.tsx:53` — contredit
  `coverageLogic.ts:20` (« Interne : personne d'autre ne le lit »). Règle : règle 1.
  | Décision : backlog (mapper vers `REPLAY_TEXT` ou retirer le title).

### [P2] `cmd_backfill_killsource` recompose `data/cache` à la main
- Où : `apps/go-api/cmd/levelup/cmd_backfill_killsource.go:415` — seconde définition du
  sous-chemin que `PathResolver.CacheRootDir()` déclare détenir seul. Règle : règle
  PathResolver. Conséquence : si le cache se scope par titre, le backfill lit un
  répertoire vide et termine en SUCCÈS — le mode de panne que le commit introducteur
  dit avoir payé. Les ratchets ne couvrent pas `cmd/`. | Décision : backlog (1 ligne).

### [P2] `replay-build` : entrée film title-aveugle pour une sortie title-scopée
- Où : `apps/go-api/cmd/replay-build/main.go:67` (ignore `--title` en entrée alors que
  la sortie `ReplayArtifactPath(*titleFlag, matchID)` est scopée ; le commentaire du
  même fichier à 4 lignes de là énonce la règle). Conséquence latente : deux `--title`
  sur le même matchID liraient le même film et écriraient deux artefacts de titres
  distincts. | Décision : backlog.

### [P2] `cmd/frontb_coverage` + `cmd/probe_pi_reconcile` : sondes d'une enquête tranchée
- Où : en-têtes = mesure unique sur le film-oracle `000d5950` ; l'hypothèse est fermée
  par `weaponv3/pi_resolver.go` en prod ; zéro référence externe (6 emplacements
  vérifiés). Règle : anti-pattern n°1. Contre-vérification `.ai/` (l'angle mort qui a
  réfuté `cmd/killsource`) faite : seuls hits = journaux historiques (thought_log,
  RE_LOG), aucun guide/plan/handoff — pas d'usage vivant. | Décision : backlog
  (suppression).

### [P2] `himap.Instance.MeshKey` : méthode sans appelant, sans test, sans défense documentaire
- Où : `internal/himap/instances.go:62`. Règle : règle 7.
- Vérification adverse : TIENT sur le fait. Atténuation : seul accesseur exporté de
  `MeshRef` (champ `json:"-"`). Son jumeau `LocalToWorld` est ÉCARTÉ, lui (plan actif
  ouvert le désigne — voir écartés). | Décision : backlog (supprimer ou attendre
  l'étape 3 du plan triangles qui réutilisera le voisin).

### [P2] En-tête périmé de `cmd/killsource` (doc inversée), dans 2 fichiers
- Où : `apps/go-api/cmd/killsource/main.go:3-8` et
  `internal/games/halo_infinite/film/killsource/doc.go:92` — le paragraphe décrit un
  état du monde disparu (« le paquet n'est pas branché », « le pont killsource_bridge.go
  est sans appelant ») alors que le paquet est branché en prod par `backfill-killsource`
  et que le pont a été DÉPLACÉ en `killcollector/bridge.go` (commit `36fc76835`).
- Règle : anti-pattern n°9 (doc inversée). Issu de la réfutation du constat « outil
  mort » (voir écartés) : l'outil vit, son en-tête ment. | Décision : backlog
  (rafraîchir 2 paragraphes).

---

## Constats écartés

| Constat | Axe | Motif d'écart |
|---|---|---|
| « Fuite de type title-specific » via `positions.PlayerPosition` dans le port | A4 | Réfuté en gravité par la vérification adverse : struct 100 % générique (TimeMS/X/Y/Z/Team), canonique de fait — reste un problème d'emplacement de paquet (constat ADR 0012 retenu) |
| `loadGameVariant` err -> `return 0, nil` (comeback.go:75-76) | A6 | Préexistant à v7.2.0, lignes déplacées à l'identique |
| `windowMatchIDs` return nil nu (profile/queries.go:170,178) | A6 | Préexistant (extraction verbatim) |
| Erreurs 500 sans log dans les handlers admin/groups/settings/... | A6 | Lignes hors plage |
| `media.go:271-277` mêmes erreurs avalées que media_likes | A6 | Hors plage (préexistant) — cité comme contexte du P1, à corriger avec lui |
| SQL brut dans `internal/api/wire/*.go` | A4 | Lignes antérieures, la plage n'a touché que des constantes |
| `legacymatch.StatsMatchRow` traversant 3 services | A4 | Paquet transitionnel documenté (ADR 0011 P4.3) + dette Escouade assumée |
| `replay.ReplayDocument` servi tel quel en corps d'API | A4 | Exemption écrite et versionnée (`SchemaVersion=2`, doc en tête de document.go) |
| Fonctions filmdec à 6-7 paramètres | A4 | Seuil effectif du repo = 7 (revive.argument-limit, daté, antérieur) |
| `sync_handler.go`/`settings.go` > 500 L | A4 | Dette antérieure gelée (+21/+22 L sans responsabilité nouvelle) |
| Modes EN en dur `objectivescore.ClassifyMode` | A3 | Seul appelant réel = cmd/diag (hors surface title-agnostic) |
| `wire.haloInfiniteModeTaxonomy()` | A3 | Racine de composition DI autorisée (ADR 0012) + note FOLLOW-UP |
| Libellés FR `ops/seed_demo_prestige_plan.go` / `seed_citation_data.go` | A3 | Données de seed/démo, couche allowlistée |
| `cmd/server/main.go:1320` join `data/cache` à la main | A3 | Ligne du 2026-06-27, antérieure |
| Portée internal/-only des ratchets slug/data-path (cmd/ non gardé) | A3 | La portée est antérieure à la plage ; seuls les 2 sites neufs sont remontés (P2) |
| DELETE-then-INSERT `killer_victim_pairs`/`weapon_kills` | A2 | Préexistant, tables sans index |
| `match_player_positions` DELETE-then-INSERT (volet ART) | A2 | Sans PK ni index = pas de surface ART (le volet « justification vers doc inexistant » est intégré au P1 des 4 tables) |
| Lecture brute dans `steps_shared_kill_events_credit_base.go` | A2 | Migrations exclues par construction ; lecture brute requise (idempotence toutes passes) |
| Absence des ratchets ADR 0030 D-3/D-4 | A2 | ADR antérieur qui déclare lui-même ne changer aucun code |
| `diag_weapons_v3 -write` objective-events sans sonde de verrou | A2b | Conséquence = message d'erreur moins clair, pas de corruption (irrecevable) |
| Snapshots antérieurs au 26/07 illisibles (snapshot_export) | A2b | Dégradation documentée et gracieuse (ErrSnapshotIncomplete -> repli live) |
| ~40 `t.Skip` fixture-dépendants filmdec/weaponv3 | A8 | Motif écrit + commande de rejeu + doublés par tests synthétiques |
| Pas de tests `replayDraw`/`ReplayCanvas` | A8 | Canvas impératif, endpoint loopback-only, logique extraite testée |
| KDA divisé `LeaderboardBlock.tsx:212,389` | A9 | Fichier non touché par la plage (dette antérieure réelle — voir Suite) |
| Accuracy 0..100 `timeseries_service_correlation.go:47-52` | A9 | Repris verbatim du god-file split v7.2.0 (antérieur) |
| Décodeur objectivescore « pas un vrai score » | A9 | Documenté dans le paquet, aucun consommateur UI (le manque de test est le P1 A8) |
| Coeur U+2665 dans home.ts | A10 | Glyphe réel du bouton like, convention préexistante |
| `isFr ?` inline AddFriendFlow / MatchEncountersTable | A10 | Structure préexistante, la plage ne retouche que des libellés à parité conservée |
| « Heatmap » en FR | A10 | Convention établie antérieure (explorer.toml, timeseries.toml) |
| formatSeconds point vs virgule ; ammoNoneHint sans accents ; aria InfoTooltip | A10 | Aucune règle écrite ne les couvre (opinion / axe accessibilité hors doctrine) |
| `Footprint()`/`QuickDeleted()` himap | A1 | Appelants réels trouvés (mapstruct-build) |
| `diag_weapons_v3` comme code mort | A1 | Mode -write opérationnel + convention diag_* préexistante (usage ops plausible) |
| Chaîne ooz->himodule->himap->map*-build | A1 | Vivante : artefacts consommés au runtime via PathResolver |
| `cmd/killsource` (6 fichiers) comme outil mort | A1 | RÉFUTÉ en vérification adverse : documentation vivante versionnée dans `.ai/` (GUIDE_KILLSOURCE.md 1005 L avec section d'usage ; PLAN_BRANCHEMENT_KILLSOURCE.md:26 l'inventorie « intact » = survie délibérée au branchement, + GATE 0 et oracle de non-régression ; HANDOFF avec consigne utilisateur assignée) ; la sous-commande `comparer` n'a aucun équivalent dans le chemin de prod. L'auditeur avait cherché dans docs/ mais pas .ai/. Résidu réel : en-tête périmé (P2 dédié ci-dessus) |
| `himap.Instance.LocalToWorld` comme code mort | A1 | RÉFUTÉ sur l'action : zéro appelant aujourd'hui, mais `.ai/PLAN_BELLE_CARTE_TRIANGLES.md:208-210` (étape 3 OUVERTE) et `HANDOFF_GEOMETRIE_TRIANGLES.md:151` le désignent nommément « à réutiliser » — usage planifié documenté |

## Axes sans constat

- **A2b (sync/service/ops/cmd — invariants ART)** : aucun constat recevable. 10
  conditions vérifiées qui tiennent, dont : garde-rails uniquement DURCIS sur la plage
  (+2 tables protégées, 2 nouveaux tests de morsure, allowlists vides), persisters
  INSERT-only, lectures via `_latest`, écritures sous lease avec releases sous defer,
  CHECKPOINT sur toutes les écritures shared_social nouvelles, split COLLECT/FLUSH suivi
  d'un re-check anti-TOCTOU sous lease. L'axe le plus critique du projet ressort solide.
- Conditions notables qui tiennent ailleurs : zéro SQL et zéro logique métier dans les
  6 handlers neufs (A4) ; ratchet slug non élargi, capabilities fines validées au
  chargement, dégradations 503 propres (A3) ; zéro hex/Tailwind couleur, 100 % des
  query keys centralisées, canvas rejeu conforme aux tokens (A10) ; KDA/TZ/_latest/LUSR
  v2/unités tous conformes, la plage REMPLACE des recalculs ad hoc par les noyaux
  partagés (A9) ; aucun `fmt.Print*` hors cmd/ sur ~103k lignes ajoutées (A6).

## Escalades utilisateur (décisions à trancher, aucune action entreprise)

1. **Dominance des modes à objectifs (P0)** : le comportement actuel est verrouillé par
   un test qui exige la régression. Trois options (fallback ancien chemin quand la
   courbe est vide ; peuplement live de `match_objective_events` ; assumer la perte en
   attendant le pipeline film complet) — choix produit.
2. **Placement `filmdec`/`positions`/`replay`** vs ADR 0012 : déplacement de paquets ou
   ADR d'exemption — décision d'architecture.
3. **Doctrine des écritures « hors chemin live »** (5 tables diag) : officialiser une
   allowlist datée + garde-rail, ou exiger persist partout — touche la doctrine ART.

## Suite

- **Chantier immédiat proposé** (si le P0 est arbitré) : dominance objectifs (P0 + P1
  erreur avalée, même fonction).
- **Lot durcissement tests** : golden réel objectivescore + réactivation cache-backed +
  verrou d'exhaustivité non tautologique + GATE 1 sur minibobine (4 constats, même
  thème : le nouveau décodé n'est contraint par aucun octet réel).
- **Lot hygiène** : i18n (2 quick wins), slog (3 quick wins), code mort confirmé
  (`probe_export`, `DescribeRootBlocks`, 2 sondes cmd/ ; `frame_debug` à arbitrer avec
  le handoff keyframe ; en-têtes killsource à rafraîchir), 2 chemins cmd/ via
  PathResolver, `tablesProtegees` +4.
- **Découvertes hors plage consignées** (dette antérieure réelle, non traitée ici) :
  KDA divisé dans `LeaderboardBlock.tsx:212,389` ; `media.go:271-277` erreurs avalées ;
  Accuracy 0..100 dans `timeseries_service_correlation.go:47-52` ; FR en dur préexistant
  dans CheckboxGroup (l.85-86, 107).
- **Non vérifié / à instruire plus tard** : setters de `traverse.go` (flags à un côté
  mort ?) ; couverture `schemadrift`/`convergence.go`/`seed_demo_prestige*` ; CTAS démo
  de `match_kill_events` sans PK/DEFAULT ; tri lexicographique `SessionFdaGapCumulative`
  si offsets hétérogènes ; proportion réelle de films `publishable=FALSE` en prod.
- Statuts des items : aucun constat n'est corrigé par cet audit (contrat du skill :
  l'audit ne corrige rien). Chaque « à planifier » attend un cadrage sous plan-review.
