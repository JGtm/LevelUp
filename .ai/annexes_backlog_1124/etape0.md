# Étape 0 — instruction sur pièces du plan du backlog du 24 novembre

Annexe de `.ai/PLAN_BACKLOG_2026-11-24.md`, écrite le 2026-11-24 en lecture seule sur la tête de
`release/v8.3.0` (commit `e884790c6b`, avant le report de `main`).

## Comment lire

- **vérifié** : relu dans le fichier cité ou mesuré par une requête rejouée ce jour (base du poste, lecture
  seule, outil `cmd/diag_q`).
- **déduit** : raisonné sans preuve directe ; ce qui le vérifierait est écrit à côté.
- Les mesures viennent des bases du poste (`data/titles/<titre>/warehouse/shared_matches_v2.duckdb`). La
  production n'a pas été regardée.
- Les fichiers `sync/engine.go`, `sync/writes.go`, `games/halo_5/capture.go` et `livesync/runner.go` sont
  touchés par le report de `main` en cours : les items A.1, A.2 et D.4 sont à relire après lui (numéros de
  ligne surtout).
- Chemins Go relatifs à `apps/go-api/`, chemins web relatifs à `apps/web/src/`, sauf mention.
- Un match est désigné par les 8 premiers caractères de son identifiant.

Ce qui contredit une hypothèse du plan est rassemblé en fin de document (section « Écarts avec le plan »).

---

## Piste A — données et fil

### A.1 — Éliminations en double dans `highlight_events`

Entrée : « [données] Éliminations en double dans `highlight_events` ».

**Verdict : à faire.** Le chiffre de l'entrée est confirmé, et la cause est identifiée.

**Preuve.**
- Mesure (vérifié) : 172 matchs sur 1 462 portent chaque élimination deux fois ; 30 372 lignes dans des
  groupes en double, 15 186 lignes en trop. Les 172 matchs sont doublés en entier (aucun double partiel,
  jamais plus de deux copies). Les morts y sont doublées aussi (172 matchs), les médailles dans 166.
- Deux épisodes distincts (vérifié par les plages d'identifiants de ligne et les dates de première
  synchronisation) :
  1. **145 matchs** synchronisés les 25 et 26 mai 2026 : une première écriture sans code (`type_hint` NULL,
     `raw_json` NULL), puis une seconde avec code. Ce sont deux écrivains différents du même fait.
  2. **27 matchs** (synchronisés de février à mai 2026) : deux copies identiques avec code, écrites en deux
     blocs contigus décalés de 6 303 lignes. C'est un rejeu en lot de matchs qui avaient déjà leurs
     événements (déduit ; le journal de l'époque le confirmerait).
- Aucun double n'est né depuis : 321 matchs synchronisés depuis le 27 mai 2026, 0 doublé (vérifié).
- L'écrivain qui peut encore doubler aujourd'hui (vérifié à la lecture) :
  `internal/sync/events_replay.go:127-184`, `ReplayHighlightEventsForMatches`. Elle remet
  `events_loaded` à faux (`clearEventsLoaded`, ligne 228) puis rappelle `ProcessHighlightEvents`, qui insère
  sans rien retirer. Son commentaire dit « Idempotent : peut être ré-exécuté sans dommage » (ligne 124) :
  c'est faux pour `highlight_events`. Son second cas d'usage (ligne 10 : « événements présents mais paires
  vides ») double à coup sûr. Appelants : `levelup replay-events` et la reprise forcée des événements
  (`BackfillEventsForMatches`, ligne 250).
- La protection annoncée n'en est pas une : `internal/persist/events_completion_persister.go:230-241` écrit
  en `INSERT OR IGNORE`, et son propre commentaire dit qu'avec la clé `id` auto-incrémentée ce « OR IGNORE »
  ne dédoublonne rien. La non-duplication repose seulement sur le drapeau `events_loaded` (lignes 36-40).
- Le flux courant est sain (vérifié à la lecture) : la convergence ne sélectionne que les matchs
  `events_loaded = FALSE` (`internal/sync/convergence.go:80`) et interdit de passer par le rejeu (ligne 389).

**Fichiers à toucher.**
- `internal/persist/events_completion_persister.go` (écriture qui ne double plus)
- `internal/sync/events_replay.go` (commentaire, et ne plus rejouer un match qui a déjà ses événements)
- `internal/migration/` : une étape neuve de reconstruction de `highlight_events` sans doubles, plus son
  inscription dans `internal/migration/order.go`
- tests : `internal/persist/events_completion_persister_test.go`, `internal/sync/events_replay*_test.go`,
  test de migration
- Couches : `persist` (écriture), `migration`, `sync`. Aucun handler, aucun service.

**Approche.**
1. Écrivain : insérer une ligne seulement si la même (match, type, instant, joueur) n'existe pas déjà
   (`INSERT ... SELECT ... WHERE NOT EXISTS`, insertion pure, pas de `ON CONFLICT`). Le rejeu devient alors
   réellement sans dommage.
2. Dédoublonnage : pas de `DELETE` sur cette table (index ART, règle anti-corruption n°1). Reconstruire la
   table par la recette de `internal/migration/append_only_rebuild.go` ; il y a un précédent sur cette même
   table (`internal/migration/steps_shared.go:141` et `:193`, sauvegarde puis réinsertion). Garder, par
   groupe (match, type, instant, joueur), la ligne la plus riche (code et `raw_json` présents). Garde sur le
   nombre de lignes attendu, étape idempotente.
3. Production : compter d'abord par la requête de `temoins_avant.md` (lecture seule, le user prévenu).
4. Règles : écriture par `persist` seulement ; la reconstruction tourne au démarrage, sous l'écrivain unique.

**Tests et gate.** Test d'intégration « deux passages de la complétion sur le même match = mêmes lignes » ;
test du rejeu sur un match déjà rempli ; test de la migration sur une base avec doubles des deux formes.
`go test -tags=integration ./internal/persist/... ./internal/sync/... ./internal/migration/...`,
`go test ./internal/sync/ -run 'ART|Allowlist'`, puis la requête de comptage : 0 attendu.

**Effort : moyen.**

**Conflits.** A.2 : même table, même migration, à faire dans le même lot. A.5 : même dossier `migration/`
(fichiers différents, `order.go` commun).

---

### A.2 — Temps forts « mode » en double et sans code

Entrées : « [base/temps forts] Temps forts « mode » en double, sans code, sur 76 matchs » et « [décodeur]
Temps forts de type « mode » du film : ce qu'ils recouvrent ».

**Verdict : à faire, avec A.1 (même cause, même migration).**

**Preuve.**
- Mesure (vérifié) : 2 330 événements « mode » sans code sur 76 matchs ; 33 matchs ont aussi la forme à
  code 10, et leurs 1 284 événements sans code sont tous le double d'un événement à code 10 ; 43 matchs
  n'ont que la forme sans code. Chiffres de l'entrée confirmés à l'unité.
- Les 33 matchs sont un sous-ensemble des 145 matchs de A.1 (synchronisés les 25 et 26 mai). Les 43 autres
  ont été synchronisés du 8 juin au 1er septembre 2026 (vérifié).
- L'écrivain de la forme sans code est le flux principal d'avant le 2 septembre 2026 (déduit, deux indices
  concordants) : toutes les lignes sans code (éliminations, morts, médailles, mode) appartiennent à des
  matchs synchronisés entre le 25 mai et le 1er septembre ; et `internal/persist/shared_persister.go:451-456`
  écrit qu'avant le 2026-09-02 le flux principal ne posait pas le code. `valeurTypeHint`
  (`shared_persister.go:439-447`) rend NULL quand la ligne ne porte ni code ni détail. Ce n'est donc pas
  « une ancienne passe » : c'est l'ancien état de l'écrivain courant, corrigé depuis.
- La forme sans code ne touche pas que le mode : 346 matchs ont des éliminations sans code, dont 201 qui
  n'ont que cette forme (vérifié).
- Lecteur touché (vérifié à la lecture) : `internal/games/halo_infinite/citations_custom.go:86-103`,
  `computeAnnexionForcee`, compte chaque événement « mode » comme une capture de zone, sans vérifier le
  mode du match et sans écarter les doubles. L'entrée du décodeur dit que cette lecture est fausse hors
  Bases.
- Côté web, aucun composant de la page du match ni du rejeu ne lit un événement de type « mode » (vérifié
  par recherche du littéral dans `features/match-replay`, `features/match-view`, `lib/replay`). La partie
  « ne les afficher qu'en Bases » de l'entrée n'a donc pas d'écran à corriger ; il reste la citation.

**Fichiers à toucher.**
- la migration de A.1 (retirer le double sans code quand un jumeau à code existe)
- `internal/games/halo_infinite/citations_custom.go` (compter une capture par (joueur, instant), et
  seulement en Bases)
- son test `citations_custom_test.go`

**Approche.**
1. Dans la reconstruction de A.1, le double sans code disparaît (la ligne à code est « la plus riche »).
2. Les 43 matchs qui n'ont que la forme sans code : pas de repassage par l'API. Tous les événements
   « mode » de la base portent le code 10 ou rien (vérifié) : la migration peut poser 10 sur ces lignes.
   À décider au lot ; sans cela, les lecteurs doivent accepter le code absent.
3. La citation : la garder aux matchs de Bases (`isStrongholdsMode` existe,
   `citations_custom_modes.go:47`). Recalcul des citations ensuite (commande existante), le user prévenu :
   c'est un changement de valeur affichée.

**Tests et gate.** Test de la citation hors Bases (0 attendu) et sur un match doublé ; gate de A.1.

**Effort : faible une fois A.1 fait.**

**Conflits.** A.1 (même migration).

---

### A.3 — Match dit « film absent » alors que son film est archivé ; coups d'envoi après la fin du match

Entrées (section du 2026-11-23) : « Match posé « film absent » alors que son film est archivé » et « Rejeux
dont le coup d'envoi tombe après la fin du match ».

**Verdict : à faire. Les deux constats sont plus larges que l'entrée ne le dit.**

**Preuve, film absent.**
- 616 matchs portent la marque « film absent » (bit 22 du registre). **36 d'entre eux ont un dossier de
  film à l'archive du poste, dont 34 avec des morceaux** (14 à 63 morceaux) ; 33 ont même un rejeu généré
  (vérifié, croisement du registre avec `data/cache/film_chunks/` et `data/cache/replays/`). L'entrée n'en
  citait qu'un.
- Le match cité (`58e6f72a`) : marque posée, 33 morceaux et un manifeste à l'archive, pas de rejeu. Son
  film est aussi écarté pour profil de build inconnu (autre entrée du backlog, hors plan).
- La marque est terminale et rien ne la retire (vérifié à la lecture de
  `internal/sync/killcollector/registry_flags.go:66-98` et `film_absent_mark.go`). Le code courant ne la
  pose plus que sur un film réellement introuvable ; elle date donc d'un état antérieur, ou d'un film
  arrivé à l'archive par une autre voie après la marque (déduit ; l'historique des journaux le dirait).
- Effets de la marque : le match est dit « analyse terminée, film perdu »
  (`internal/platform/duckdb/analysis_probe_repo.go:91-92`) et sort de la file des rejeux
  (`internal/sync/replayartifacts/backlog.go:90`).

**Preuve, coup d'envoi après la fin.**
- Les deux matchs cités (`ed3b6497`, `dd29957c`) ont une durée rangée de 21 s et 24 s, mais leur dernière
  mort lue est à 460 s et 674 s, et le plus long temps de jeu d'un joueur est de 462 s et 677 s (vérifié).
  **Ce n'est pas le coup d'envoi qui est faux, c'est la durée du match au registre.** La résolution refuse
  donc un coup d'envoi juste (`internal/analysis/timeline/kickoff.go:80-92`, contrôle `FilmWithinMatch`).
- Portée : 23 matchs ont une mort lue plus de 5 s après la fin déclarée ; au moins 30 ont un temps de jeu
  qui dépasse la durée de plus d'une minute (vérifié). Ce constat rejoint l'entrée « Durée de l'API
  différente de fin moins début : 437 matchs », non instruite.

**Fichiers à toucher.**
- `internal/persist/` : une écriture qui retire la marque d'un match (même fichier que
  `EventsCompletionPersister` ou persisteur voisin)
- `internal/sync/killcollector/` : au début de la passe, réconcilier la marque avec l'archive
  (`filmcache.FilmComplet` existe)
- `internal/analysis/timeline/kickoff.go` et `internal/sync/kickoff/kickoff.go` : la fenêtre du match
- tests des trois paquets

**Approche.**
1. Marque : une passe de réconciliation retire la marque quand le film est complet à l'archive ; elle
   compte et journalise. Écriture par `persist`, sous l'écrivain partagé.
2. Fenêtre : contrôler le coup d'envoi du rejeu contre la plus grande des deux valeurs « durée du registre »
   et « plus long temps de jeu d'un joueur ». Décision à faire valider : elle change la source retenue pour
   les 2 matchs (et la borne pour les autres). `timeline.KickoffRev` est à monter (ADR 0050), donc le
   rattrapage se rejoue une fois au démarrage.
3. Ne pas corriger la durée du registre dans ce lot (hors périmètre, à noter en découverte).

**Tests et gate.** Test « marque retirée quand le film est complet, gardée sinon » ; test de la résolution
avec une durée plus courte que le temps de jeu. `go test -tags=integration ./internal/sync/... ./internal/persist/...`,
`go test ./internal/archlint/ -run Kickoff`.

**Effort : moyen.**

**Conflits.** A.4 (mêmes fichiers du coup d'envoi) : à faire par le même agent, dans l'ordre A.4 puis A.3.

---

### A.4 — Coup d'envoi, trois points

Entrées (section du 2026-11-23) : « Course étroite entre le rattrapage du coup d'envoi et une publication
de rejeu », « Coup d'envoi « film » gardé non contrôlé contre la fenêtre du match », « Compteur des coups
d'envoi écrits incrémenté avant l'insertion ».

**Verdict : à faire, les trois existent encore.**

**Preuve (vérifié à la lecture).**
1. Course : à la publication d'un rejeu, l'état du match se lit dans un segment de lecture
   (`internal/sync/replayartifacts/coupdenvoi.go:70-73`, `kickoff.Prepare`) et s'écrit plus tard dans le
   segment d'écriture (`:103-131`, `persisterCoupsDEnvoi`). Entre les deux, le rattrapage peut écrire. Le
   rattrapage, lui, lit et écrit sous le même segment (`internal/sync/kickoff/backfill.go:108-141`).
2. Contrôle : `internal/sync/kickoff/kickoff.go:120-123` et `:153-159` (`keptFilm`) gardent tel quel un
   coup d'envoi de source « film » quand le rejeu est absent ou illisible, sans le repasser par
   `timeline.FilmWithinMatch`.
3. Compteur : `kickoff.Observe` (`internal/sync/kickoff/write.go:47-53`) dit compter « un coup d'envoi
   ÉCRIT », mais ses trois appelants l'appellent avant l'insertion : `internal/sync/engine_fetch.go:95`,
   `internal/sync/engine_v2bridge.go:104`, `internal/service/openspartan_import_service.go:231`. Une
   insertion qui échoue est comptée.

**Fichiers à toucher.** `internal/sync/replayartifacts/coupdenvoi.go`, `internal/sync/kickoff/kickoff.go`,
`internal/sync/kickoff/write.go`, les trois appelants, les tests de `sync/kickoff` et de
`sync/replayartifacts`.

**Approche.**
1. Course : toujours résoudre sous le segment d'écriture (le chemin `surWriter` existe déjà,
   `coupdenvoi.go:119-131`) ; la lecture préalable ne sert plus qu'à décider s'il faut prendre l'écrivain.
2. Contrôle : dans `Resolve`, un coup d'envoi « film » gardé hors de la fenêtre retombe sur la résolution
   normale, compté par `CompteurRejeuHorsBornes`.
3. Compteur : compter après l'insertion réussie (rendre le coup d'envoi à l'appelant, qui compte une fois
   le lot rangé), ou renommer le compteur en « résolu ». La première option garde le sens du compteur.
4. Un seul point d'écriture hors insertion : `persist.KickoffPersister` (garde
   `archlint/kickoff_write_guard_test.go`), à ne pas contourner.

**Tests et gate.** Test de course (une écriture intercalée entre lecture et écriture) ; test du coup d'envoi
gardé hors fenêtre ; test du compteur sur insertion en échec.
`go test -tags=integration ./internal/sync/kickoff/... ./internal/sync/replayartifacts/...`.

**Effort : faible.**

**Conflits.** A.3 (fenêtre du match, mêmes fichiers). L'appelant `engine_fetch.go` est touché par le report
de `main`.

---

### A.5 — Médailles du bouclier, trois défauts

Entrée : « [médailles du bouclier] Trois défauts P2 de la revue adversariale, non corrigés ».

**Verdict : à faire, les trois existent encore.**

**Preuve (vérifié à la lecture).**
- (a) `internal/sync/replayartifacts/usage_frags.go:34-37` : `ORDER BY match_id, time_ms`, sans départage.
  La vue `match_kill_events_latest` porte bien une colonne `id` (vérifié par `DESCRIBE`).
- (b) `internal/sync/replayartifacts/usage_levelup.go:63` : `Rev: UsageLevelUpRev` est posé sans regarder
  `in.Frags.Read` ni `in.Equipes`. Le rattrapage saute ensuite le match
  (`cmd/levelup/cmd_backfill_usage_summary.go:362-368`, comparaison de `levelUpRev`).
- (c) `internal/migration/steps_shared_usage_summary_medals_view.go:57-77` : la branche « dérivé » existe
  seulement `WHERE u.revenge_kills IS NOT NULL` ; une passe sans ligne de joueur ne produit aucune ligne
  « dérivé », donc la branche « hérité » revient (`NOT bool_or(t.kind = 'derived')`).

**Fichiers à toucher.** Les trois fichiers ci-dessus pour (a) et (b) ; pour (c), une étape de migration
NEUVE qui repose la vue (ne pas modifier l'étape déjà jouée en production) et son inscription dans
`internal/migration/order.go` ; tests de `sync/replayartifacts` et de `migration`.

**Approche.**
- (a) ajouter `, id` au tri.
- (b) ne poser la révision que si le fil des morts et les équipes ont été lus ; sinon laisser la révision
  vide, le rattrapage reprendra le match.
- (c) décider « dérivé » sur l'existence d'une passe de LevelUp pour le match (`match_usage_films_latest`
  avec `levelup_rev` non vide), pas sur une ligne de joueur. Lecture par les vues `_latest` seulement.
- `UsageLevelUpRev` passe à `lu2` (le résultat change à temps égal), donc `levelup backfill-usage-summary`
  est à relancer : geste de production à écrire, le user prévenu.

**Tests et gate.** Deux frags au même instant dans les deux ordres d'insertion (même résultat) ; passe sans
fil des morts (révision vide) ; passe sans ligne de joueur (pas de Vengeur hérité).
`go test -tags=integration ./internal/sync/replayartifacts/... ./internal/migration/...`.

**Effort : faible.**

**Conflits.** A.1 (`migration/order.go`).

---

### A.6 — Écriture concurrente du catalogue de cartes perdue

Entrée (section du 2026-11-23) : « Écriture concurrente du catalogue de cartes perdue ».

**Verdict : à faire.**

**Preuve (vérifié à la lecture).** `internal/mapcatalog/store.go:101-131`, `prendreVerrou` : toute erreur
d'ouverture du fichier de verrou autre que « existe déjà » fait passer en force tout de suite (lignes
117-122), et le délai dépassé aussi (lignes 123-128). Sous Windows, créer un fichier que l'autre écrivain
est en train de supprimer rend « accès refusé », pas « existe déjà » : c'est la cause probable de l'échec
1 fois sur 10 (déduit ; à vérifier en journalisant l'erreur pendant 50 passages du test
`TestAddOverlayEntryConcurrentNePerdPasDEntree`).

**Fichiers à toucher.** `internal/mapcatalog/store.go`, `internal/mapcatalog/store_test.go` (ou le fichier
du test cité).

**Approche.** Traiter « accès refusé » et « partage refusé » comme un verrou tenu (réessayer jusqu'au
délai) ; ne passer en force immédiatement que pour un dossier absent. Garder « jamais d'erreur rendue ».
Au passage, la journalisation de ce fichier utilise `slog.Warn` sans contexte : à noter en découverte, pas à
corriger ici.

**Tests et gate.** `go test ./internal/mapcatalog/ -run Concurrent -count=50` vert 50 fois sur 50.

**Effort : faible.**

**Conflits.** Aucun.

---

## Piste B — page du match

### B.1 — Cartes Némésis : victime ambiguë

Entrée : « [page du match] Cartes Némésis : compte possiblement inférieur de 1 quand le fil ne nomme pas la
victime ».

**Verdict : à écarter (volume).** Le défaut existe dans le code, mais il ne touche presque rien.

**Preuve.**
- Code (vérifié) : `features/match-view/MatchNemesisCards.logic.ts:40` ne compte qu'une élimination dont la
  victime est nommée ; `internal/service/match_view_killfeed_weapon.go:97-119` ne nomme aucune victime quand
  un tueur en a deux au même instant ; `internal/service/match_view_builders_combat.go:105` retire du fil
  une élimination d'avant le coup d'envoi.
- Mesure (vérifié, requête dans `temoins_avant.md`) : Halo Infinite, **0** élimination à victime ambiguë sur
  le poste aujourd'hui (l'entrée en comptait 4 dans 1 match, avant la régénération du parc du 24 novembre) ;
  Halo 5, 6 éliminations dans 3 matchs.
- Les éliminations d'avant le coup d'envoi n'ont pas été comptées.

**Si le user le garde.** Fichiers : `internal/service/match_view_killfeed_weapon.go` et son test. Approche :
quand une clé (tueur, instant) porte deux victimes, répartir les victimes sur les événements de cette clé
dans l'ordre des paires, au lieu de n'en nommer aucune. Effort faible.

**Conflits.** B.5 (même dossier `service/match_view_*`).

---

### B.2 — Ordre déterministe des sections « rencontres » et « némésis »

Entrée (section du 2026-11-23) : « Sections « rencontres » et « némésis » de l'onglet d'équipe ».

**Verdict : à faire pour « rencontres » ; « némésis » n'existe plus côté serveur.**

**Preuve (vérifié à la lecture).**
- `internal/platform/duckdb/queries_match_detail.go:38-64`, `Q23MatchEncounters` : `ORDER BY count_together
  DESC`, sans départage. Deux joueurs rencontrés le même nombre de fois sortent dans un ordre libre.
- `internal/service/match_view_builders_team.go:216-262` garde l'ordre de la requête.
- L'onglet d'équipe ne sert plus de section « némésis » : `domain.MatchTeamTab`
  (`internal/domain/match_view.go:763-768`) n'a que `roster`, `scoreboard`, `encounters`. Le bloc
  `team_tab.nemesis` ne vit plus que dans le fichier golden (voir B.5).

**Fichiers à toucher.** `internal/platform/duckdb/queries_match_detail.go`, un test de
`internal/platform/duckdb/match_view_repo_*_test.go`.

**Approche.** `ORDER BY count_together DESC, tm.xuid` (le SQL reste sous `platform/duckdb`). Vérifier au lot
les autres lectures de l'onglet (statistiques de rencontre) pour la même absence de départage.

**Tests et gate.** Deux joueurs à égalité, ordre fixe sur 20 exécutions.
`go test ./internal/platform/duckdb/ -run Encounter`.

**Effort : faible.**

**Conflits.** Aucun.

---

### B.3 — Éliminations liées à l'objectif sur la fiche de match

Entrée : « [match/objectifs] Éliminations liées à l'objectif sur la fiche de match ».

**Verdict : à faire. Le serveur sert déjà tout ; c'est un travail web.**

**Preuve (vérifié à la lecture).**
- `internal/domain/match_view.go:638-698`, `MatchScoreboardObjective`, sert déjà par joueur :
  `flag_carriers_killed`, `flag_returners_killed`, `kills_as_flag_carrier`, `kills_as_flag_returner`,
  `zone_offensive_kills`, `zone_defensive_kills`, `skull_carriers_killed`, `kills_as_skull_carrier`,
  `vip_assists`.
- `features/match-view/MatchScoreboard.logic.ts:184-242` : les colonnes affichées par mode ne prennent
  aucun de ces champs (drapeau : captures, retours, vols, temps de porteur ; zones : captures,
  sécurisations, temps ; crâne : prises et temps ; VIP : cinq colonnes sans `vip_assists`).
- Aucun fichier de `features/match-view` ne lit ces champs (vérifié par recherche).

**Fichiers à toucher.** `features/match-view/MatchScoreboard.logic.ts`, `features/match-view/i18n/fr.ts`,
`en.ts`, `contract.ts`, `features/match-view/objectivesChart.test.ts`,
`features/match-view/MatchObjectivesSection.test.tsx`. Aucun fichier Go pour la fiche de match.

**Approche.**
1. Ajouter par mode les colonnes « éliminations liées à l'objectif » ; la grille et le face-à-face
   s'adaptent seuls (`objectivesChart.ts` ne code rien en dur par mode).
2. Libellés en français et en anglais, sans anglicisme (« Porteurs éliminés », « Éliminations en défense
   de zone »). Une valeur absente reste « non mesurée », jamais zéro (règle du fichier).
3. Séries temporelles : `features/timeseries/TimeseriesObjectiveCard.tsx` existe ; y ajouter ces grandeurs
   demande un champ serveur neuf par mode. À proposer en second temps : sur la fiche de match le gain est
   immédiat, dans les séries il faut d'abord dire quelle grandeur a un sens d'un match à l'autre.
4. La règle « Sessions, séries temporelles, Escouade » : l'Escouade les exploite déjà (onglet Objectif).

**Tests et gate.** Tests du modèle par mode (colonnes présentes, total d'équipe).
`npx vitest run src/features/match-view --maxWorkers=2`, `make check-types`.

**Effort : faible pour la fiche ; moyen si les séries temporelles entrent.**

**Conflits.** B.6 (dictionnaires de `features/match-view/i18n`).

---

### B.4 — Tableau « Impact des coéquipiers » par session

Entrée (découvertes du 2026-11-22) : « Tableau « Impact des coéquipiers » par session (revue du
2026-11-22, P2) ».

**Verdict : à faire, les trois points existent encore.**

**Preuve (vérifié à la lecture).**
- (a) `internal/analysis/squadimpact/sessions.go:131-134` : un match compte comme joué si le joueur est dans
  l'équipe alliée lue, ou s'il a un rôle. Quand l'équipe alliée n'est pas lue
  (`internal/service/teammates/teammates_squad_impact_sessions.go:28-45`, table vide sans xuid du joueur
  principal), « joué » retombe sur « a un rôle » et le net par match est faux, sans signal.
- (b) `features/squad/impact-sessions/SessionMatrix.tsx:109-118` : table en `table-fixed`, les colonnes de
  match n'ont aucune largeur ; sur une longue session elles s'écrasent.
- (c) `lib/api/types.ts:1071-1077` : sept ré-exports `SquadImpactSessions*` sans aucun lecteur ; seul
  `SquadImpactSessions` (ligne 1070) est lu, par 4 fichiers (vérifié par recherche de chaque nom).

**Fichiers à toucher.** `internal/analysis/squadimpact/sessions.go`,
`internal/service/teammates/teammates_squad_impact_sessions.go`, leurs tests ;
`features/squad/impact-sessions/SessionMatrix.tsx` et son test ; `lib/api/types.ts`.

**Approche.**
- (a) ne pas publier le bloc quand l'équipe alliée n'a pas été lue, avec la raison journalisée
  (`slog.InfoContext`) ; le calcul reste dans `analysis`, la décision de publier dans `service`.
- (b) largeur minimale par colonne de match et défilement horizontal du tableau ; à juger à l'écran (ligne à
  ajouter à l'artefact des vérifications).
- (c) retirer les sept ré-exports et compléter le commentaire de `MatchesPlayed`.

**Tests et gate.** Go : bloc nul sans équipe lue. Web : session de 20 matchs, colonnes à largeur minimale.
`go test ./internal/analysis/squadimpact/ ./internal/service/teammates/`,
`npx vitest run src/features/squad/impact-sessions --maxWorkers=2`, `npx knip`.

**Effort : faible.**

**Conflits.** B.5 (`lib/api/types.ts`).

---

### B.5 — Champs servis sans lecteur et fonctions sans appelant

Entrées : « [contrat] `combat_tab.nemesis_duels` servi vide et sans lecteur », « [code mort] Fonctions
d'analyse sans appelant de production », « `medal_name` des événements de médaille du contexte du match »
(section du 2026-11-23).

**Verdict : à faire, sauf un point déjà fait.**

**Preuve (vérifié par recherche des noms, hors fichiers de test).**
- `nemesis_duels` : toujours servi vide (`internal/service/match_view_builders_combat.go:92`), type
  `MatchNemesisRow` (`internal/domain/match_view.go:459` et `:699-720`) ; aucun lecteur web hors
  `lib/api/generated.ts`.
- Golden : `tests/fixtures/golden_values/match_view_slayer.json:261` porte encore `team_tab.nemesis`. Le
  test qui lit ce fichier ne contrôle que la présence de `header` et `summary_tab`
  (`tests/golden/golden_test.go:133-137`).
- `medal_name` : champ de `MatchHighlightEvent` (`internal/domain/match_view.go:394-398`), rempli par
  `match_view_builders_combat.go:116-118` depuis `internal/platform/duckdb/match_view_repo_noms.go:94`.
  Aucun composant web ne lit `.medal_name` d'un événement (les seuls usages web du mot sont une clé de tri
  de la page Médailles et un commentaire).
- `ComputeAntagonistCounts` (`internal/analysis/killer_victim.go:178`) : aucun appelant.
- `ComputeModeCategoryAverages` (`internal/analysis/match_history_avg.go:50`) : aucun appelant.
- `TitleDataAdapter.LoadMatchSummaries` (`internal/games/adapter.go:382`) : trois implémentations
  (`halo_infinite/adapter_data.go:264`, `halo_5/adapter_data_loaders.go:177`,
  `synthetic_title_b/adapter.go:66`) et aucun service appelant.
- **Déjà fait** : `internal/analysis/spawn_detection.go` n'existe plus (supprimé par le commit `87910a4602`).

**Fichiers à toucher.** `internal/domain/match_view.go`, `internal/service/match_view_builders_combat.go`,
`internal/platform/duckdb/match_view_repo_noms.go` (lecture du nom de médaille, si elle n'a plus d'autre
usage), `internal/analysis/killer_victim.go`, `internal/analysis/match_history_avg.go`,
`internal/games/adapter.go` et les trois adaptateurs, les doublures de test qui implémentent l'interface
(`internal/games/resolver_test.go`, `internal/service/fake_detail_adapter_test.go`), leurs tests,
`tests/fixtures/golden_values/match_view_slayer.json`, `api/openapi.yaml` et `lib/api/generated.ts`
(régénérés), `docs/adr/0011-*.md` (mention de `MedalImageURL`, ADR en anglais seulement).

**Approche.**
1. Vérifier chaque nom une dernière fois avant suppression (règle 7 : supprimer avec tests et imports).
2. `medal_name` : avant de retirer la lecture SQL, vérifier que le fil du rejeu ne passe plus par elle (le
   rejeu lit ses médailles dans le document de rejeu, dit l'entrée ; vérifié seulement côté web).
3. `LoadMatchSummaries` touche l'interface multi-titre : retirer la méthode de l'interface et des trois
   titres dans le même commit (skill `canonical-types`).
4. `make generate-types`, puis `openapi-gen -check`.

**Tests et gate.** `go build ./... && go vet ./...`, `go test ./internal/analysis/... ./internal/games/...
./internal/service/... ./tests/golden/...`, `go test ./internal/archlint/...`, `make check-types`,
`npx knip`.

**Effort : faible à moyen (l'interface multi-titre fait le « moyen »).**

**Conflits.** B.1 et A.2 (fichiers `service/match_view_*`), B.4 (`lib/api/types.ts`), tout item qui
régénère le contrat (B.6, C.6).

---

### B.6 — « Mètres »

Voir la section « 0.6 » plus bas : le constat, la conversion, la liste des distances affichées et
l'approche y sont. **Verdict : à faire, mais l'item est plus large que « convertir l'affichage » et demande
deux décisions du user. Effort : moyen à lourd.**

---

## Piste C — rejeu

### C.1 — Zone du tueur à chaque ligne du fil

Entrée : « [rejeu/killfeed] Zone de callout du tueur dans le killfeed ».

**Verdict : déjà fait.** Voir « 0.2 » : livrée en 8.2.0, puis le lieu affiché est devenu celui de la victime
par la décision D-5 de la refonte de la page du rejeu.

---

### C.2 — Suite des collines jouées

Entrée : « [décodeur/objectifs] Ordre de rotation des collines du Roi de la colline ».

**Verdict : à faire.**

**Preuve (vérifié par recherche).** Le document de rejeu porte `zoneStates[].spans` ; il est lu par les
calques de la carte (`features/match-replay/layers/useZoneStates.ts`, `zoneStatesLayer.ts`,
`zoneStatesPaint.ts`). Aucun composant de `features/match-replay/ui/` n'affiche la suite des collines : le
seul fichier de ce dossier qui cite `zoneStates` est un test du fil.

**Fichiers à toucher.** Un modèle pur neuf sous `features/match-replay/model/` (suite des collines à
l'instant lu), un composant sous `features/match-replay/ui/`, les dictionnaires `features/match-replay/i18n/`
(`fr.ts`, `en.ts`, `i18nContract.ts`), tests. Aucun fichier Go.

**Approche.**
1. Modèle pur : des `spans` de chaque zone, tirer la liste ordonnée des collines déjà jouées et la colline
   active à l'instant lu.
2. « La prochaine quand elle est connue » : l'ordre est une donnée de la variante de mode, pas du film
   (dit l'entrée). Elle n'est donc connue que si la suite jouée montre un cycle complet déjà répété ; sinon
   ne rien afficher (rien d'inventé).
3. Emplacement : à décider avec la piste M (réglages et cadres du rejeu) ; ne pas ajouter d'interrupteur.
4. Lettres ou noms des zones : par les zones nommées de la carte quand elles existent, sinon un rang.

**Tests et gate.** Modèle sur un rejeu de Roi de la colline réel aminci ; état vide hors de ce mode.
`npx vitest run src/features/match-replay --maxWorkers=2`, vérification en lecture.

**Effort : moyen.**

**Conflits.** M (emplacement), C.3 et C.4 (dictionnaires du rejeu).

---

### C.3 — Gestes du joueur dans la fiche (section « En passant »)

Entrée : « [décodeur] Commandes et gestes du joueur : notes de bas de page de la fiche du rejeu ».

**Verdict : à faire.**

**Preuve (vérifié).** Le décodeur monté (`apps/go-api/go.mod:14`, v1.6.1) publie `gestures[]` et sa
couverture (`replay` : `Gestures []Gesture`, `GestureCoverage`, commandes `crouchCommand` et
`jumpCommand`). Le schéma web du document ne déclare que `reloads`
(`lib/replay/replayDocumentSchema.ts:156`) : `gestures` n'y est pas, et aucun fichier de
`features/match-replay` ne porte de section « En passant ».

**Fichiers à toucher.** `lib/replay/replayDocumentSchema.ts`, `lib/replay/replayNormalize.ts`,
`lib/replay/replayReadyTypes.ts`, `features/match-replay/model/playerInstantSources.ts`,
`playerInstantView.ts`, un modèle neuf pour les gestes, `features/match-replay/ui/fiche/FicheGroupes.tsx`,
`features/match-replay/i18n/playerInstant.i18n.ts`, tests. Vérifier côté Go que le service du rejeu ne
filtre pas le champ (`internal/service/replay_service.go`, désérialisation tolérante).

**Approche.**
1. Déclarer `gestures` au schéma et à la normalisation, comme `reloads`.
2. Compter par joueur à l'instant lu : coups de mêlée, lancers amorcés, rechargements, accroupis et sauts
   (commandes tenues). Non publié = non affiché : sur les builds 1.9 à 1.11 aucun geste n'est publié, la
   section doit alors être absente ou dire « non mesuré », jamais zéro.
3. Section « En passant » sous « Totaux du match » (décision D-24 de la refonte).
4. Règle 18 : sans objet ici, le champ vient du décodeur.

**Tests et gate.** Normalisation avec et sans `gestures` ; fiche sur un rejeu récent et sur un rejeu d'un
ancien build. `npx vitest run src/lib/replay src/features/match-replay --maxWorkers=2`, vérification en
lecture sur un rejeu à 60 Hz.

**Effort : moyen.**

**Conflits.** C.4 (mêmes fichiers de schéma et de normalisation), C.5 (`rosterLogic.ts` si la fiche y
touche), M.

---

### C.4 — Éliminations et actions de score d'un bot désignées par place

Entrée : « [décodeur/rejeu] Éliminations dont un bot est l'auteur : à rouvrir côté LevelUp à la montée du
décodeur v1.6.0 ».

**Verdict : à faire.**

**Preuve (vérifié).** Le décodeur v1.6.1 publie des champs `place` et `victimPlace` (paquet `replay` du
module). `lib/replay/replayDocumentSchema.ts` ne déclare aucun champ `place` (recherche vide). Les
antagonistes du rejeu désignent un bot par `bot:<nom>` (`features/match-replay/model/antagonistNames.ts:7`)
et les bots en sont exclus par décision du 2026-11-22, « tant que ces éliminations ne sont pas lues ».

**Fichiers à toucher.** `lib/replay/replayDocumentSchema.ts`, `replayNormalize.ts`, `replayReadyTypes.ts`,
`features/match-replay/model/killFeedLogic.ts`, `replayAntagonists.ts`, `antagonistNames.ts`,
`features/match-replay/hooks/useReplayAntagonists.ts`, le modèle des actions de score, tests.

**Approche.**
1. Accepter une élimination ou une action de score désignée par `place` sans identifiant ; la rattacher au
   joueur par la table des places du document (une seule table d'identité, pas de seconde clé).
2. Rouvrir les bots dans les antagonistes une fois ces éliminations lues.
3. Règles du user : aucune vie anonyme, aucune section « sans équipe ».

**Tests et gate.** Document avec éliminations par place ; antagonistes avec un bot.
`npx vitest run src/lib/replay src/features/match-replay --maxWorkers=2`, vérification en lecture sur un
match avec bots.

**Effort : moyen.**

**Conflits.** C.3 (schéma et normalisation) : les deux par le même agent, l'un après l'autre.

---

### C.5 — Fichiers au-dessus de 500 lignes ; quatre mesures de largeur

Entrées (section du 2026-11-23) : « `ReplayPage.row.test.tsx` : 508 lignes », « `rosterLogic.ts` au-delà du
seuil », « Quatre mesures de largeur ».

**Verdict : à faire.**

**Preuve (vérifié, `wc -l`).** `lib/replay/rosterLogic.ts` : 781 lignes ;
`features/match-replay/ReplayPage.row.test.tsx` : 508 lignes. Mesures de largeur : `useBodyWidth`
(`features/match-replay/hooks/useCardRowLayout.ts:40`), `useReplayViewport`
(`features/match-replay/hooks/useReplayViewport.ts:137`), `useContentWidth`
(`lib/replay/useContentWidth.ts:20`), et l'effet de taille de « Occupation du terrain »
(`features/match-view/MatchPositionsHeatmap.tsx:202`). Hors de l'entrée, deux autres existent ailleurs :
`features/_shared/assists/useElementWidth.ts:8` et `useMeasuredWidth`
(`features/synthesis/WeaponRecordsRuler.tsx:60`) ; le dépôt compte 15 `new ResizeObserver` hors tests.

**Fichiers à toucher.** `lib/replay/rosterLogic.ts` (à découper en sous-fichiers par responsabilité),
`features/match-replay/ReplayPage.row.test.tsx` (à couper en deux), les quatre fichiers de mesure, un hook
commun (par exemple `lib/layout/useElementWidth.ts`, hors de toute feature), un garde-rail
`*.guard.test.ts`.

**Approche.**
1. Un seul hook de largeur dans `lib/` (une feature n'importe pas une autre : lint inter-features).
   `useReplayViewport` mesure aussi la hauteur et règle un cas propre : ne réunir que la partie commune.
2. Garde-rail : `new ResizeObserver` interdit hors du hook et d'une liste datée (règle des trois copies).
3. `rosterLogic.ts` : découpe sans changement de comportement, témoins = la suite de tests existante.
4. Ne rien changer au rendu : aucun contrôle à l'écran attendu.

**Tests et gate.** `npx vitest run src/lib/replay src/features/match-replay src/features/match-view
--maxWorkers=2`, `node tools/lint-cross-feature-imports.mjs`, `make check-types`.

**Effort : moyen.**

**Conflits.** C.3 et C.4 si elles touchent `rosterLogic.ts` : faire C.5 en dernier dans la piste, ou en
premier et avant tout autre agent.

---

### C.6 — Compte des rejeux à l'ancien schéma dans la vue d'administration

Entrée : « [déploiement] Rejeux du site hébergé à l'ancien schéma, sans film ».

**Verdict : à faire.**

**Preuve (vérifié).** Aucun compte n'existe : le seul signe est le badge par rejeu
(`features/match-replay/ui/ReplaySchemaBadge.tsx`). La lecture de l'en-tête d'un rejeu existe
(`internal/replaybuild/artifact_meta.go:104`, `ArtifactHeader`, fichier `.meta` qui porte
`schemaVersion`). Sur le poste : 1 455 rejeux, tous au schéma 127.

**Fichiers à toucher.** Un port dans `internal/port/`, une lecture de fichiers hors `service` (paquet
voisin de `internal/replaybuild` ou `internal/replayartifact`), un service et un handler d'administration,
`api/openapi.yaml` et `lib/api/generated.ts` (régénérés), une section de `features/admin/` et ses
dictionnaires, `lib/query/keys.ts`, tests.

**Approche.**
1. Compter par schéma en lisant les en-têtes (0,25 s pour 1 455 fichiers, mesure du plan du 18 novembre) :
   « à jour », « à l'ancien schéma », « illisibles ».
2. Handler sans logique ; service derrière un port ; pas de SQL.
3. Règle 18 : le champ a son lecteur, la section d'administration, dans le même lot.
4. Par titre (capacité des rejeux), pas de comparaison de slug.

**Tests et gate.** Dossier de rejeux de test à trois schémas ; section d'administration.
`go test ./internal/service/... ./internal/api/...`, `go test ./internal/archlint/...`,
`npx vitest run src/features/admin --maxWorkers=2`, `make generate-types`.

**Effort : faible.**

**Conflits.** B.5 et B.6 (contrat régénéré).

---

## Piste D — serveur et production

### D.1 — Archivage des films par le calcul des frags

Entrée : « [films/prod] Le calcul des frags archive chaque film téléchargé sans lire le réglage ».

**Verdict : à faire.**

**Preuve (vérifié à la lecture).**
- `internal/sync/killcollector/remote_films.go:124-145` : tout film lu est écrit à l'archive
  (`filmarchive.Write`) dès que la racine d'archive est renseignée ; le réglage n'est pas lu.
- Le réglage est lu ailleurs : `internal/sync/replayartifacts/artifacts.go:93-95` et `:129-131`,
  `internal/api/wire/registry_replay_generation_film.go:265`.
- Défaut « archiver » pour tous les profils : `internal/platform/settings/defaults.go:102-103` et `:164`.
- Capacités du profil de déploiement : `internal/config/deployment.go:68-71` et `:294-333`
  (`capabilitiesOf`).

**Fichiers à toucher.** `internal/sync/killcollector/remote_films.go` et son câblage,
`internal/config/deployment.go` (capacité neuve, par exemple « archive des films par défaut »),
`internal/platform/settings/defaults.go` (défaut dérivé de la capacité), `/bootstrap` si l'écran des
réglages affiche le défaut, tests de `killcollector`, `config`, `settings` ; `.ai/gestes_prod_8_3_0.md`
(retrait de la tâche planifiée du serveur).

**Approche.**
1. Le calcul des frags reçoit une fonction « archiver ? » relue à chaque passe, comme `replayartifacts`.
2. Le défaut vient d'une capacité du profil : oui sur le bureau et en développement, non sur le site
   hébergé (ADR 0044 : jamais de lecture du profil hors du résolveur ; ratchet sur `.DemoMode` et
   `LEVELUP_ENV`).
3. Point à vérifier avant de coder : la passe des rejeux relit-elle le film depuis l'archive juste après la
   passe des frags ? Si oui, « ne pas archiver » dans la première impose un second téléchargement dans la
   seconde. Il faut alors écrire puis retirer après usage, pas « ne jamais écrire ».
4. Un réglage déjà enregistré par un utilisateur garde sa valeur : seul le défaut change.

**Tests et gate.** Passe des frags avec archive coupée (aucun fichier écrit) et active ; défaut par profil.
`go test -tags=integration ./internal/sync/...`, `go test ./internal/config/... ./internal/platform/settings/...`,
`go test ./internal/archlint/...`.

**Effort : moyen.**

**Conflits.** D.3 (dossier `sync/`, fichiers différents).

---

### D.2 — Démarrage : pool de jetons créé vide

Entrée : « [boot/auth] Le démarrage attend la résolution du pool de jetons avant d'ouvrir le port ».

**Verdict : à faire.**

**Preuve (vérifié à la lecture).**
- `cmd/server/main.go:75` appelle `initAccountsPool` avant `buildRouter` et `openRouterAndHeartbeat`.
- `cmd/server/boot_tasks.go:43-73` puis `:255-281` : `pool.NewPool` est appelé de façon synchrone.
- `internal/platform/auth/pool/pool.go:173` : `resolveBootSources` résout tous les comptes avant de rendre
  le pool ; `pool_boot_resolve.go:15` : 6 résolutions à la fois.
- Le pool vide est déjà permis (`AllowEmpty`, `boot_tasks.go:272`, `pool.go:152` et `:195`) et le
  remplissage à chaud existe (re-scan, connexion).

**Fichiers à toucher.** `internal/platform/auth/pool/pool.go`, `pool_boot_resolve.go`, `types.go` (option
de résolution différée), `cmd/server/boot_tasks.go`, `cmd/server/boot_gate.go` si l'étape de démarrage
change de nom, tests du paquet `pool` et de `cmd/server`.

**Approche.**
1. Créer le pool vide, lancer la résolution des comptes en tâche de fond, ajouter chaque compte résolu par
   la voie du remplissage à chaud.
2. Garder la règle « deux comptes au même jeton de rafraîchissement ne se résolvent jamais en même temps »
   (`pool_boot_resolve.go`, rotation du jeton).
3. Mise en commun (ADR 0038) : un compte résolu en tâche de fond entre au round-robin public par la même
   porte (`Pool`, `IsWithheld`), pas par un chemin neuf.
4. L'auto-sync doit tolérer un pool vide les premières secondes sans journaliser une erreur par cycle.
5. Revue adversariale dédiée (authentification), comme le plan le prévoit.

**Tests et gate.** Résolveur lent : le pool est rendu tout de suite, se remplit ensuite ; résolveur en
panne : pool vide, pas d'erreur fatale ; arrêt du serveur pendant la résolution.
`go test ./internal/platform/auth/... ./cmd/server/...`, `go test ./internal/auth/ -run Sentinel`, mesure du
temps jusqu'à « serveur prêt » avant et après.

**Effort : moyen.**

**Conflits.** D.3 (auto-sync et pool vide).

---

### D.3 — Alerte « zero-insert » ; avertissement de mise en file sans compte consentant

Entrées : « [sync/journal] Alerte « zero-insert prolongé » qui sonne à chaque cycle en prod » et
« Avertissement journalisé à chaque mise en file » (section du 2026-11-23).

**Verdict : à faire, les deux existent encore.**

**Preuve (vérifié à la lecture).**
- `internal/scheduler/auto_sync_run.go:373-388` : le compteur ne retombe à zéro que si CE cycle a inséré un
  match ; au-delà du seuil, l'avertissement part à chaque cycle. Les matchs rangés par le déclencheur de
  fin de partie ne le remettent jamais à zéro.
- `internal/api/wire/registry_catalog_drain.go:158-170` : avertissement à chaque appel quand aucun compte
  n'a consenti à la mise en commun.

**Fichiers à toucher.** `internal/scheduler/auto_sync_run.go` (et son état des joueurs),
`internal/api/wire/registry_catalog_drain.go`, tests des deux paquets ; un port si le planificateur doit
lire « dernier match connu du joueur » (`internal/port/`, lecture sous `internal/cronstore`).

**Approche.**
1. Zero-insert : remettre le compteur à zéro quand le dernier match connu du joueur a changé depuis le
   cycle précédent, quelle que soit la voie ; et n'avertir qu'au franchissement du seuil puis à intervalle
   long, pas à chaque cycle (modèle : « une fois par épisode », lot des journaux du 2026-11-15).
2. Le planificateur n'importe pas `platform/duckdb` (garde `TestSchedulerSansStockageDuckDB`) : la lecture
   passe par un port servi par `cronstore`.
3. Mise en file : avertir une fois par épisode (passage de « au moins un compte » à « aucun »), niveau
   débogage ensuite.

**Tests et gate.** Joueur dont les matchs arrivent par une autre voie : aucun avertissement ; vraie panne
simulée : un avertissement au seuil. `go test -tags=integration ./internal/scheduler/...`,
`go test ./internal/api/wire/...`, `go test ./internal/archlint/...`.

**Effort : faible.**

**Conflits.** D.2.

---

### D.4 — Halo 5 incomplet

Entrée : « [sync/h5] Match Halo 5 rangé sans participants ni chronologie, jamais repris ».

**Verdict : à écarter (volume faible, titre inactif, reprise manuelle existante).** Détail en « 0.4 ».

---

## Piste E — hygiène

### E.1 — Ménage du disque

Entrée : « [outillage] disk-hygiene supprime les worktrees enregistrés sous C:/t et les clones git à
commits non poussés ».

**Verdict : déjà fait.** Détail en « 0.2 ».

---

### E.2 — Commentaires périmés, libellé de garde-rail faux, doc d'une commande

Entrées : « [commentaires] Commentaires du code périmés (regroupés) », « Base par défaut périmée d'un
outil, dans la documentation des commandes », « En-tête faux d'un outil », « Libellé d'un garde-rail qui
décrit l'ancien comportement ».

**Verdict : à faire, sauf un point non localisé.**

**Preuve (vérifié).**
- 38 occurrences de « CLAUDE.md §20 » sous `apps/web/src` (l'entrée en comptait 33).
- `lib/accessibility/semantic-tokens.ts:92` parle encore de tons en opacités.
- `internal/games/canonical/fields.go:10` : « ne dépend de rien hors stdlib », alors que
  `canonical/assets.go:7` et `canonical/match.go:6` importent `internal/i18n`.
- `internal/analysis/modelabel/category.go:35` renvoie à une référence git qui n'existe plus.
- `docs/COMMANDS.md:1737` et `docs/FR/COMMANDS.md:1806` citent `origin/feat/v75` comme base par défaut.
- `cmd/killsource/main.go` (en-tête, lignes 1-8) : formulation sur l'usage du décodeur par l'application, à
  relire.
- **Non localisé** : « libellé d'un garde-rail qui décrit l'ancien comportement » (P3 de la relecture finale
  du 2026-11-23). Ni le backlog ni le plan du 18 novembre ne nomment le fichier. Le pilote doit le fournir,
  sinon ce point est à statuer « non traité, non localisé ».

**Fichiers à toucher.** Les fichiers cités, plus les 38 fichiers web portant le marqueur (remplacement
mécanique : agent Haiku).

**Approche.** Une passe unique sans effet à l'écran. Le marqueur `color-allow` est lu par le lint des
couleurs : ne changer que le texte de la raison, pas le marqueur. Doc de commande : anglais et français dans
le même commit (règle 15). Vérifier la vraie base par défaut de `replay-corpus-gate --reference=base` dans
son code avant d'écrire la doc.

**Tests et gate.** `make check-types`, lint des couleurs, `npx vitest run src/lib/accessibility
--maxWorkers=2`, `go build ./...`.

**Effort : faible.**

**Conflits.** Les 38 fichiers web peuvent croiser B, C et M : faire E.2 après la fusion des pistes web, ou
en tout premier.

---

### E.3 — Garde-rail des jetons figé sur 2026 ; garde « processus enfants » ; test au nom de fichier invalide

Entrées : « Garde-rail des jetons à préfixe de date figé sur l'année 2026 », « [archlint] Garde « tout
enfant naît par childproc » sans objet sur les briques », « Test qui écrit un nom de fichier avec un octet
nul ».

**Verdict : à faire pour les deux premiers ; le troisième est à localiser au lot.**

**Preuve.**
- (vérifié) `internal/archlint/wire_jeton_par_profil_test.go:32` : `prefixeDateExemptions = "2026-"` ; une
  exemption datée de 2027 serait refusée (message ligne 130).
- (vérifié) `internal/archlint/childproc_single_spawn_test.go:27` : racines `libs/film`, `libs/gamefiles`,
  `libs/haloapi`, `libs/himap`, alors que `libs/` n'existe plus ; les deux exemptions des lignes 31-33
  visent des fichiers disparus.
- (non localisé) le test de `sync/replayartifacts` qui écrit un nom de fichier à octet nul : aucun littéral
  d'octet nul trouvé dans les tests du paquet. Le nom vient sans doute d'un identifiant de match fabriqué
  par un test (déduit). Pour le trouver : lancer les tests du paquet et lister ce qui est écrit sous
  `data/cache/replays/halo_5/` de la racine de test.

**Fichiers à toucher.** Les deux tests d'archlint ; le test fautif de `sync/replayartifacts` une fois
trouvé.

**Approche.**
- Date : accepter toute date `AAAA-MM-JJ` par une expression régulière, comme les autres listes datées du
  dépôt.
- Processus enfants : retirer les racines `libs/` et leurs exemptions (les briques ont leurs propres
  dépôts). `pkg/duckdbbackup` lance un outil externe hors garde : décider d'étendre la garde à `pkg/` ou
  de l'écrire comme exemption datée.
- Octet nul : corriger l'identifiant du test, pas le code de production.

**Tests et gate.** `go test ./internal/archlint/...`, `go test ./internal/sync/replayartifacts/...`.

**Effort : faible.**

**Conflits.** Aucun.

---

### E.4 — Pseudos réels : noms de tests, surnoms, fichiers hors `apps/`

Entrées : « [hygiène/tests] Trois noms de tests portent encore un pseudo de joueur », « [licences/tests]
Surnoms dérivés de pseudos réels dans des identifiants de tests », « [licences] Pseudos réels hors de
`apps/` ».

**Verdict : à faire pour les trois noms de tests ; le reste à borner avant de lancer.**

**Preuve (vérifié par recherche).** Trois fonctions de test portent un pseudo de joueur (hors pseudo du
mainteneur, conservé par décision) : `internal/platform/duckdb/career_repo_q7_monotone_test.go:199`,
`internal/service/career_helpers_test.go:383`, `internal/sync/enrichments_test.go:453`. Deux autres portent
le pseudo du mainteneur (`internal/persist/batch_roundtrip_test.go:74`,
`internal/testfixtures/paths_test.go:42`) : conservées.

**Fichiers à toucher.** Les trois fichiers de test, et la référence des tests lue par la CI
(`apps/go-api/baseline_current.jsonl` et la référence sous `.ai/baselines/`) : un test renommé sans sa
ligne fait rougir la CI.

**Approche.**
1. Renommer les trois tests avec un libellé neutre, et leur ligne de la référence dans le même commit.
2. Surnoms et fichiers hors `apps/` : les entrées disent « plusieurs centaines d'occurrences » et renvoient
   à des tables hors dépôt. Sans ces tables, un agent Haiku ne peut pas borner le travail. Proposition :
   limiter E.4 aux trois noms de tests, et garder les deux autres entrées au backlog avec leur condition.
3. Ne jamais écrire un pseudo dans un message de commit.

**Tests et gate.** `go test ./...` complet (référence des tests), gate de baseline de la CI.

**Effort : faible pour les trois noms ; moyen à lourd pour le reste.**

**Conflits.** Aucun sur les trois fichiers.

---

### E.5 — `seed-demo --emit-manifest` et l'identifiant du joueur source

Entrée (section « Confidentialité », 2026-11-23) : « `seed-demo --emit-manifest` écrit dans le manifeste
versionné l'identifiant réel du joueur source ».

**Verdict : à faire.**

**Preuve (vérifié à la lecture).** `internal/ops/seed_demo_manifest.go:32` déclare `source_xuid` ;
`internal/ops/seed_demo_emit.go:24` l'écrit ; aucun code ne relit ce champ du manifeste (les lectures de
`SourceXUID` portent sur les options de la commande ou sur les entrées de l'effectif, pas sur le
manifeste). Les deux manifestes versionnés sous `config/demo/` portent aujourd'hui une valeur ; le commit
`a4ca9a33db` dit l'avoir remplacée (non vérifié ici : la valeur a la forme d'un identifiant, rien ne dit
qu'elle est fictive).

**Fichiers à toucher.** `internal/ops/seed_demo_manifest.go`, `internal/ops/seed_demo_emit.go`, leurs
tests, les deux manifestes sous `config/demo/` (retrait du champ).

**Approche.** Retirer le champ de la structure et de l'émission ; la lecture reste tolérante (un champ
inconnu est ignoré). Vérifier que `source_gamertag` suit la même règle (le pseudo du mainteneur est
conservé par décision).

**Tests et gate.** Test « l'émission n'écrit aucun identifiant de joueur » (garde-rail sur le manifeste
émis). `go test ./internal/ops/...`.

**Effort : faible.**

**Conflits.** Aucun.

---

## Points demandés par le plan

### 0.2 — Entrées peut-être déjà closes

| Entrée | Verdict | Preuve |
|---|---|---|
| Zone du tueur dans le fil du rejeu | **déjà fait** | Livrée en 8.2.0 (plan `.ai/V8.3/PLAN_V8_1_2_2026-11-09.md`, item 1.5 coché ; ligne « Récemment complété » du 2026-11-14). Le fil affiche aujourd'hui le lieu de la **victime** : `features/match-replay/hooks/useFeedZones.ts`, `model/feedZone.ts`, colonne `zoneOf` de `ui/ReplayKillFeed.tsx:78-82`. Le passage du tueur à la victime est la décision D-5 de `.ai/V8.3/PLAN_REFONTE_UI_REJEU_2026-11-15.md:70` (« lieu = celui de la victime, dans les trois cadres »), confirmée ligne 311. Vérifié. |
| `scripts/disk-hygiene.ps1` | **déjà fait** | Commit `84ec5ed295` du 2026-11-22. Règle 5 du script (lignes 26-31, 138-161, 196-201) : un dossier qui est, ou qui contient jusqu'à 3 niveaux plus bas, un `.git` (dossier ou fichier, donc dépôt comme worktree) n'est jamais supprimé, et la ligne « CONSERVE » est journalisée. C'est plus large que la demande : tout dépôt est épargné, poussé ou non. Limite : un dépôt à plus de 3 niveaux de profondeur n'est pas vu (`$GitMarkerMaxDepth = 3`, ligne 95). Vérifié à la lecture, non rejoué. |
| Deux films avec des faits d'un ancien schéma | **déjà fait (sans objet)** | Les deux matchs cités (`edc5daf6`, `e96070e2`) n'ont plus rien sur le poste : ni rejeu, ni morceau de film, ni manifeste, ni fichier de faits. Les 1 455 rejeux du poste sont tous au schéma 127. Vérifié. |
| Arme d'une touche lue sur l'index brut | **déjà fait** | `.ai/V8.3/annexes_decodeur_v160/l7_mesures_apres_regeneration.md`, piste D : touches avec arme 1 197 454 (86,6 %) avant, 1 296 016 (89,5 %) après ; touches sans tireur 79 457 avant, 1 419 après. L'entrée demandait de relire le parc après la régénération : c'est fait par cette pièce. Relu dans l'annexe, non remesuré. |

Conséquences : C.1 et E.1 sortent du plan ; les deux entrées du décodeur passent en « Récemment complété ».

---

### 0.4 — Halo 5 : les 303 matchs sans ligne dans `highlight_events`

L'API Halo n'a pas été appelée (consigne). L'item 0.4 du plan, « dire si l'API sert encore la chronologie de
trois matchs », n'est donc pas exécuté tel qu'écrit ; la caractérisation ci-dessous suffit à conclure.

**Mesures (vérifié, base Halo 5 du poste).**
- 3 032 matchs, 0 sans participants, 303 sans ligne dans `highlight_events`. Tous les matchs de la base ont
  été synchronisés le même jour (23 juin 2026).
- Sur Halo 5, `highlight_events` ne porte que des médailles, des assistances et des événements de mode ; les
  éliminations vivent ailleurs. Un match sans ligne n'est donc pas forcément un match sans chronologie.
- Les 303 matchs :
  - années de jeu : 2015 (94), 2016 (144), 2017 (16), 2018 (35), 2019 à 2023 (14) ;
  - quatre joueurs à l'origine de la synchronisation : 133, 131, 32 et 7 matchs ;
  - **244 sont des parties à un seul participant** (parties personnalisées ou hors matchmaking) ;
  - aucun n'a de playlist ni de mode au registre, mais c'est vrai des 3 032 matchs de cette base ;
  - **132 ont d'autres tables d'événements remplies** (précision par arme pour 130, paires tueur-victime
    pour 42) : leur chronologie a bien été lue, elle ne contenait ni médaille ni assistance ni événement de
    mode ;
  - **171 n'ont aucune table dérivée** : 105 sans une seule élimination au tableau (rien à raconter), 66
    avec des éliminations, dont 65 de plus de 2 minutes et 41 avec des assistances au tableau.
- Le défaut réel touche donc au plus **66 matchs sur 3 032 (2,2 %)**, joués surtout en 2015 et 2016.

**Le code peut-il produire cet état ?** Oui (vérifié à la lecture).
- `internal/games/halo_5/capture.go:314-321` : une chronologie indisponible rend une chronologie vide et
  incrémente `EventsFailed` ; le match est rangé quand même. Idem pour le tableau, lignes 343-349.
- Le commentaire promet « un futur passage le re-tentera » (lignes 311-313 et 339-342). C'est faux : un
  match connu est sauté (`capture.go:261-268`), en synchronisation comme en rattrapage.
- `EventsFailed` et `CarnageFailed` sont journalisés mais ne comptent pas comme erreurs du cycle
  (`internal/games/halo_5/livesync/runner.go:161-164` et `:224-225`).

**Une reprise existe-t-elle ?** Oui, manuelle : `internal/games/halo_5/livesync/events_backfill_surfaces.go:39-45`
(`RunEventsBackfill`, commande `cmd/h5-events-backfill`) reprend les matchs sans précision par arme,
serveur arrêté, en rappelant la chronologie par match. Elle vise exactement les 173 matchs sans table
dérivée. Elle ne tourne pas seule.

**Conclusion.** Le volume ne justifie pas un correctif : au plus 66 matchs anciens, surtout des parties en
solo, sur un jeu qui n'est plus joué, avec une commande manuelle qui existe. **Recommandation : abandonner
D.4** (décision D-5 du plan confirmée). Deux restes à noter en découverte, sans les traiter :
- le commentaire faux de `capture.go` (« un futur passage le re-tentera ») ;
- `events_backfill_surfaces.go:104-109` fait un `DELETE` sur `highlight_events` hors de `persist` (table à
  index ART) ; l'outil tourne serveur arrêté, mais la règle anti-corruption n°1 ne le prévoit pas.

---

### 0.5 — Trois écarts de maquette à juger à l'écran

Source : entrée « [rejeu/maquette] Écarts avec les maquettes à juger à l'écran après livraison ». Aucun n'est
un défaut constaté ; ce sont trois points sans dessin, à trancher par le user. À ajouter à l'artefact des
vérifications (collection `verifs`) :

1. **Icônes de cadre.** Où : page du rejeu, les boutons qui choisissent le cadre (Standard, Cinéma, Plein
   cadre). À juger : les pictogrammes conviennent-ils, et le cadre actif se reconnaît-il du premier coup
   d'œil ? Attendu : un seul bouton marqué actif, lisible en thème clair et sombre.
2. **Curseur de volume dans la barre d'une ligne.** Où : barre de lecture du rejeu quand elle tient sur une
   ligne. À juger : garder le curseur de volume visible, ou ne garder qu'une icône comme sur la maquette ?
   Attendu : la barre ne passe pas sur deux lignes à cause du curseur, à la largeur habituelle de la
   fenêtre.
3. **Bouton caméra quand un joueur est suivi.** Où : page du rejeu, bouton caméra, après avoir choisi
   « Suivre » sur un joueur. À juger : le fond de couleur d'accent du bouton est-il le bon signal ? Attendu :
   on voit que le suivi est actif, et le bouton reprend son aspect normal dès que le suivi s'arrête.

---

### 0.6 — « Mètres » : conversion, distances affichées

#### Ce que vaut une unité du jeu

Les positions du rejeu et de la base sont en unités du jeu. Trois mesures indépendantes, toutes tirées du
dépôt ou des bases du poste, disent qu'une unité vaut environ 3 m, pas 1 m.

| Mesure | Valeur en unités | Si 1 u = 1 m | Si 1 u = 3,048 m | Source |
|---|---|---|---|---|
| Vitesse de course au sol (médiane) | 2,2 à 2,4 u/s | 8 km/h, un pas rapide | 24 à 26 km/h, une course | `.ai/V8.3/annexes_fonds_relief/conception.md:9-11`, 1,5 million de points (relu) |
| Sprint | 3,05 u/s | 11 km/h | 33 km/h | même pièce (relu) |
| Sommet d'un saut ordinaire | 0,82 à 0,89 u | 0,85 m | 2,6 m | même pièce, lignes 673-679 (relu) |
| Hauteur d'un étage | 1,8 à 2,2 u | 2 m : un Spartan n'y tient pas debout | 5,5 à 6,7 m | même pièce, ligne 11 (relu) |
| Distance tueur-victime d'une élimination par derrière au corps à corps (2 833 cas) | médiane 0,34 u (quartiles 0,27 et 0,46) | 34 cm entre deux centres de corps : impossible | 1,04 m (0,82 à 1,40 m) | mesuré ce jour, requête dans `temoins_avant.md` (vérifié) |
| Distance d'un tir à la tête (35 986 cas) | médiane 7,7 u, 95e centile 27,7 u | 7,7 m et 28 m | 23,6 m et 84 m | même requête (vérifié) |

**Degré de certitude.**
- « Une unité vaut environ 3 m et non 1 m » : **établi** (trois mesures qui ne dépendent pas l'une de
  l'autre, dont une faite aujourd'hui ; l'hypothèse 1 m donne deux résultats impossibles).
- « Une unité vaut exactement 10 pieds, soit 3,048 m » : **déduit, non mesuré**. Les mesures donnent 3 m à
  10 ou 15 % près ; elles ne séparent pas 3,0 de 3,048. La valeur 3,048 est la convention des moteurs Halo
  précédents, citée par l'entrée du backlog et par un commentaire du dépôt
  (`features/match-replay/model/threatSensor.ts:24`). Ce qui la vérifierait : une distance que le jeu
  affiche lui-même en mètres (marqueur d'objectif) relevée par le user sur un match, comparée à la même
  distance mesurée dans le rejeu.
- Le facteur est donc à faire trancher par le user, comme le prévoit la décision D-11 : 3,048 (convention)
  ou 3 (mesure arrondie). L'écart entre les deux est de 1,6 %, sans effet visible sur un affichage arrondi.

#### Le dépôt se contredit

- `features/match-replay/model/threatSensor.ts:21-26` écrit que l'unité est SUPPOSÉE valoir 1 m, et écarte
  3,048 parce qu'un rayon de 13 m serait « invraisemblable sur une carte de 50 m ». L'argument est
  circulaire : les 50 « m » de la carte sont eux-mêmes des unités.
- `.ai/V8.3/annexes_fonds_relief/conception.md` et l'ADR 0049 disent « unités d'environ 3 m ».

#### Trois sortes de valeurs, à ne pas traiter pareil

Aucune conversion n'existe dans le code (recherche de `3.048`, `0.3048` : rien).

1. **Distances mesurées dans le film et affichées « m »** : elles sont en unités, présentées comme des
   mètres. L'affichage est faux d'un facteur 3. C'est toute la liste ci-dessous.
2. **Seuils donnés en unités par l'éditeur ou mesurés dans le film** (rayon du capteur 4,25 « wu », rayon de
   zone 2, marge de tranche 0,25) : la valeur est juste, seul le nom « mètres » est faux. Rien à convertir.
3. **Seuils donnés en vrais mètres et comparés à des unités** : le calcul lui-même est faux.
   - **Portée du radar** : `config/titles/halo_infinite/mappings/regulation.toml:380` et suivantes, 18 m en
     arène et 24 m en Grande bataille (valeurs du user, celles que le jeu annonce). Elles sont comparées
     telles quelles à des distances en unités (`internal/analysis/coordination/vies_pres_ou_seul.go:145-147`,
     `APortee`, appelée par `internal/analysis/tactical/equipe.go:49`). Le rayon appliqué est donc de 18
     unités, soit environ 55 m : trois fois trop large. La lecture « où je meurs isolé » de la Tactique et
     les comptes « dans le rayon » sous-comptent l'isolement (vérifié à la lecture du code ; l'effet chiffré
     reste à mesurer).
   - À classer au lot, un par un : `DistanceMaxM = 40`, `DistanceMaxMortM = 60`, `RayonEnnemisProchesM = 20`
     (`internal/analysis/tactical/reapparition.go:73-77`), `RayonEvaluationM = 8`
     (`reapparition_evaluation.go:25`), `WeaponRangeLevelBandM = 1` (`internal/analysis/weapon_range.go:71`),
     et côté web `MELEE_LINK_MAX_M = 8`, `REPAIR_FIELD_RADIUS_M = 3`, `SHROUD_RADIUS_M = 6`,
     `WALL_RADIUS_M = 1.6`. Pour chacun : la valeur a-t-elle été choisie sur des données en unités (elle
     reste, on renomme) ou sur une distance réelle (elle se convertit) ?

#### Distances affichées à l'utilisateur

Toutes viennent des positions du film (table `kill_positions_latest` et pistes du rejeu), donc en unités,
sauf mention. « Faux ×3 » = unités du jeu présentées comme des mètres.

**Web (`apps/web/src`)**

| Fichier et ligne | Ce qui est affiché | Source | État |
|---|---|---|---|
| `features/match-view/i18n/fr.ts:100`, `en.ts:105` | distance moyenne d'élimination par arme, fiche de match | `avg_distance_m` (`internal/domain/match_view_kill_distance.go:34-36`) | faux ×3 |
| `features/compare/compareWeaponsShared.ts:52` | distance des armes, page Comparer | profils de portée (`median_m`) | faux ×3 |
| `features/explorer/ExplorerTargetFragRange.tsx:86` | portée des frags d'un joueur exploré | profils de portée | faux ×3 |
| `features/timeseries/weaponRangeText.ts:45-46` | distance et écart de distance, Séries temporelles | `internal/domain/synthesis_weapon_range.go:47-48`, `:110-111`, `:138`, `:163` | faux ×3 |
| `lib/i18n/manifests/timeseries.toml:870-871` | « médiane {m} m » | même source | faux ×3 |
| `features/synthesis/weaponRecords_logic.ts:54`, `WeaponRecordsRuler.tsx:198` | records et médianes de distance par arme | `median_m`, `record_m` (`internal/domain/synthesis_weapon_records.go:43-45`) | faux ×3 |
| `features/squad/charts/squadRangeRolesChart.ts:346` | axe « {value} m » des rôles de portée, Escouade | `role_low_m`, `role_high_m`, `median_m` (`internal/domain/match_range_profile.go:36-108`) | faux ×3 |
| `lib/i18n/manifests/squad.toml:615-616` | « écart au lobby {delta} m » | `lobby_delta_m` | faux ×3 |
| `features/session-detail/coordinationI18n.ts:103-104`, `:144-145` | « Médiane {med} m », « Écart au lobby {ecart} m » | `median_m`, `lobby_delta_m` | faux ×3 |
| `features/tactical/TacticalMortDetail.tsx:50` | distance d'un coéquipier au mort et au tueur | `distance_mort_m`, `distance_tueur_m` (`internal/domain/tactical_equipe.go:47-48`) | faux ×3 |
| `lib/i18n/manifests/tactical.toml:429-442` | « seul · {d} m », « près · {d} m », « à {d} m » | distances de coordination | faux ×3 |
| `lib/i18n/manifests/tactical.toml:496-497`, `:536-537` | « médiane {d} m », « {d} m » | `mediane_tueur_m`, `mediane_plus_proche_m` (`tactical_equipe.go:77-78`), `distance_m` (`tactical_zone.go:24`) | faux ×3 |
| `lib/i18n/manifests/tactical.toml:225-226`, `:592-593`, `:616-617`, `:640-641` | « Rayon du radar ({r} m) », « dans le rayon de {r} m » | `rayon_m`, `rayons_radar_m` (`tactical_equipe.go:56`, `:178`, `tactical_page.go:198`) | le nombre affiché (18, 24) est le vrai rayon du jeu ; c'est le calcul qui l'applique en unités (sorte 3) |
| `lib/i18n/manifests/tactical.toml:197-198` | « grille {pas} m » | `pas_m` (`tactical_page.go:71`, `:140`, `tactical_raster.go:113`) | faux ×3 |
| `lib/i18n/manifests/tactical.toml:352-353` | « x {x0}…{x1} m · y {y0}…{y1} m » | coordonnées d'une cellule | faux ×3 (ce sont des coordonnées, pas une distance : l'unité est à retirer plutôt qu'à convertir) |
| `lib/i18n/manifests/tactical.toml:724` | « < 1 m » | distance arrondie | faux ×3 |
| `features/match-replay/i18n/fr.ts:36-37`, `en.ts:36-37` | marqueur hors cadre du rejeu : « nom · N m », « N joueurs · N m » | positions des pistes | faux ×3 |

**Serveur (`apps/go-api`)**

- `internal/i18n/messages/*.toml` : aucun message ne porte de distance (vérifié par recherche).
- Champs d'API de distance (`api/openapi.yaml`, tous relus dans `internal/domain`) : `avg_distance_m`,
  `min_distance_m`, `max_distance_m`, `median_m`, `lobby_delta_m`, `lobby_median_m`, `role_low_m`,
  `role_high_m`, `period_median_delta_m`, `median_kills_m`, `median_deaths_m`, `min_m`, `max_m`,
  `record_m`, `distance_mort_m`, `distance_tueur_m`, `mediane_tueur_m`, `mediane_plus_proche_m`,
  `distance_m`, `rayon_m`, `rayons_radar_m`, `pas_m`, et `radiusM` du document de rejeu servi
  (`internal/domain/replaydoc/objectives.go:95`, rayon d'une zone d'objectif, dessiné et non affiché en
  chiffres). Tous en unités, sauf `rayon_m` et `rayons_radar_m` (vrais mètres du jeu).
- Hors périmètre de B.6 d'après le plan : `metersPerPixel` et `anchorMedianGapM` des fonds de carte (noms
  internes, aucun affichage en chiffres trouvé).

#### Approche recommandée pour B.6

1. **Une constante, deux helpers.** Côté Go : une constante « mètres par unité » et une fonction de
   conversion dans `internal/analysis` (sans stockage, sans SQL). Côté web : la même constante et un
   formateur de distance dans `lib/formatters/`. La valeur est écrite deux fois : un test de parité lit les
   deux (même patron que le jeu de cas partagé des zones).
2. **Où convertir : à l'affichage, côté web.** Les champs `_m` de l'API restent en unités ; les renommer
   demande un changement de contrat sur une vingtaine de champs. Proposition : ne pas renommer dans ce
   lot, écrire dans la description OpenAPI de chaque champ qu'il est en unités du jeu, noter le renommage en
   découverte. Le user tranche si le renommage entre.
3. **Le radar est un correctif de calcul, pas d'affichage.** Convertir le rayon (18 m → 5,9 unités) avant
   `APortee`. Tous les comptes « dans le rayon » et « isolé » changent : à mesurer avant et après sur trois
   matchs, et à annoncer dans les notes de version. Décision du user requise.
4. **Garde-rails.** Web : aucun gabarit finissant par « m » hors du formateur (liste datée pour les
   durées). Go : aucun littéral de facteur hors de la constante.
5. Libellés en français et en anglais ; les coordonnées de cellule perdent leur unité.

**Fichiers à toucher (B.6).** Les 17 lignes du tableau web, `lib/formatters/` (helper neuf),
`internal/analysis/` (constante et conversion), `internal/analysis/tactical/equipe.go` ou le point où le
rayon est résolu (`internal/service/tactical*.go`, `rayonsParMatch`), `threatSensor.ts` (commentaire faux),
les tests de chaque écran, deux garde-rails. Si le renommage des champs entre : `internal/domain/*` (22
champs), `api/openapi.yaml`, `lib/api/generated.ts`.

**Effort : moyen sans le radar ni le renommage ; lourd avec.**

**Conflits.** B.3 (dictionnaires de la fiche de match), B.5 et C.6 (contrat régénéré), M (dictionnaires du
rejeu pour le marqueur hors cadre).

---

## Écarts avec le plan

1. **A.1 et A.2 sont un seul défaut.** Le plan les sépare et cherche deux écrivains. Les mesures montrent
   une seule famille : l'ancien état du flux principal (sans code, jusqu'au 1er septembre 2026), une double
   écriture les 25 et 26 mai, et un rejeu en lot sur 27 matchs. Une seule migration les règle.
2. **A.1 : l'écrivain qui double est le rejeu des événements**, dont le commentaire affirme le contraire.
   Le flux courant ne double plus.
3. **A.3 : 36 matchs, pas un.** Et les deux « coups d'envoi après la fin » viennent d'une durée de match
   fausse au registre (21 s et 24 s pour des matchs de 7 et 11 minutes), pas du coup d'envoi.
4. **B.1 : volume nul sur Halo Infinite aujourd'hui** (0 élimination à victime ambiguë, contre 4 à
   l'écriture de l'entrée).
5. **B.2 : la section « némésis » de l'onglet d'équipe n'existe plus** côté serveur.
6. **B.5 : `spawn_detection.go` est déjà supprimé.**
7. **B.6 n'est pas qu'un affichage** : la portée du radar (vrais mètres) est comparée à des unités, donc la
   lecture « isolé » de la Tactique est calculée avec un rayon trois fois trop large. Et un commentaire du
   dépôt soutient l'hypothèse inverse (1 unité = 1 m).
8. **C.1 et E.1 sont déjà faits**, ainsi que les deux entrées du décodeur de 0.2.
9. **0.4 n'a pas appelé l'API** (consigne de l'étape 0) : la conclusion repose sur la base et le code.
10. **0.3 : le fichier golden de la vue du match ne protège rien** (le test ne contrôle que deux clés) ; les
    témoins utiles sont les tests de service et les comptes SQL (voir `temoins_avant.md`).
11. **E.2 et E.3 : deux points ne sont pas localisables** depuis le dépôt (le « libellé de garde-rail » et le
    test à octet nul).

## Tableau de synthèse

| Item | Verdict | Effort | Fichiers principaux |
|---|---|---|---|
| A.1 | à faire | moyen | `persist/events_completion_persister.go`, `sync/events_replay.go`, migration neuve |
| A.2 | à faire (avec A.1) | faible | migration de A.1, `games/halo_infinite/citations_custom.go` |
| A.3 | à faire (36 matchs ; durée du registre fausse) | moyen | `sync/killcollector/`, `persist/`, `analysis/timeline/kickoff.go` |
| A.4 | à faire | faible | `sync/replayartifacts/coupdenvoi.go`, `sync/kickoff/kickoff.go`, `write.go` |
| A.5 | à faire | faible | `sync/replayartifacts/usage_frags.go`, `usage_levelup.go`, étape de migration neuve |
| A.6 | à faire | faible | `mapcatalog/store.go` |
| B.1 | à écarter (volume) | faible si gardé | `service/match_view_killfeed_weapon.go` |
| B.2 | à faire (rencontres seules) | faible | `platform/duckdb/queries_match_detail.go` |
| B.3 | à faire (web) | faible ; moyen avec les séries | `features/match-view/MatchScoreboard.logic.ts`, dictionnaires |
| B.4 | à faire | faible | `analysis/squadimpact/sessions.go`, `SessionMatrix.tsx`, `lib/api/types.ts` |
| B.5 | à faire (un point déjà fait) | faible à moyen | `domain/match_view.go`, `analysis/*`, `games/adapter.go`, golden, contrat |
| B.6 | à faire, deux décisions du user | moyen à lourd | 17 points d'affichage web, helper Go et web, radar de la Tactique |
| C.1 | déjà fait | — | — |
| C.2 | à faire | moyen | `features/match-replay/model/`, `ui/`, dictionnaires |
| C.3 | à faire | moyen | `lib/replay/replayDocumentSchema.ts`, fiche du rejeu |
| C.4 | à faire | moyen | `lib/replay/replayDocumentSchema.ts`, fil, antagonistes |
| C.5 | à faire | moyen | `lib/replay/rosterLogic.ts`, test de la page, hook de largeur |
| C.6 | à faire | faible | port, service et handler d'administration, `features/admin/` |
| D.1 | à faire | moyen | `sync/killcollector/remote_films.go`, `config/deployment.go`, `platform/settings/defaults.go` |
| D.2 | à faire | moyen | `platform/auth/pool/pool.go`, `cmd/server/boot_tasks.go` |
| D.3 | à faire | faible | `scheduler/auto_sync_run.go`, `api/wire/registry_catalog_drain.go` |
| D.4 | à écarter (volume) | — | — |
| E.1 | déjà fait | — | — |
| E.2 | à faire (un point non localisé) | faible | 38 fichiers web, 4 fichiers Go, `docs/COMMANDS.md` et sa version française |
| E.3 | à faire (un point à localiser) | faible | deux tests d'`archlint` |
| E.4 | à faire pour trois noms de tests ; reste à borner | faible ; moyen à lourd pour le reste | trois fichiers de test, référence des tests |
| E.5 | à faire | faible | `ops/seed_demo_manifest.go`, `ops/seed_demo_emit.go`, deux manifestes |

## Items recommandés à l'écart

- **B.1** — aucune élimination touchée sur Halo Infinite aujourd'hui, 6 sur Halo 5 (jeu inactif) ; l'entrée
  disait effort moyen, priorité basse.
- **C.1** — déjà livré ; le lieu affiché est celui de la victime par décision du user (D-5 de la refonte).
- **D.4** — au plus 66 matchs anciens sur 3 032, surtout des parties en solo ; une commande de reprise
  manuelle existe.
- **E.1** — déjà corrigé le 2026-11-22.
- **E.4, parties « surnoms » et « hors `apps/` »** — non bornables sans les tables hors dépôt ; à garder au
  backlog. Les trois noms de tests restent à faire.
- **B.3, partie « séries temporelles »** — à sortir de ce lot si le user veut un effort faible : la fiche de
  match seule ne demande aucun code serveur.

Décisions attendues du user avant les pistes : le facteur de conversion (3,048 ou 3), la correction du
radar de la Tactique (elle change des comptes affichés), le renommage ou non des champs d'API en `_m`, la
fenêtre de contrôle du coup d'envoi (A.3), et l'écart de B.1 et D.4.
