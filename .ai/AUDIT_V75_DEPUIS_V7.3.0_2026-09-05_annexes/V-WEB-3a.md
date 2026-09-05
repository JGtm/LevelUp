# Vérification adverse V-WEB-3a

Dépôt `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, branche `feat/v75`, HEAD `736ccf3c3`.
Lecture seule. Toutes les lignes citées ont été rouvertes et recomptées.

## Constat 1 — « le web ne PEUT PAS gater sur le film » : RÉFUTÉ

- Ce que j'ai vérifié :
  - `apps/web/src/lib/capabilities/capabilities.ts:14-35` → **20 clés**, aucune `film.*`. Exact.
  - `apps/go-api/internal/service/bootstrap_service.go:586-600` → `buildAvailableTitlesFrom` ne
    projette bien que `t.Capabilities` (les `Cap*`). Exact.
  - `apps/go-api/internal/domain/title/capabilities_ts_mirror_test.go:14-30` → le ratchet compare
    `registry.go` (`Cap*`) et `TITLE_CAPABILITIES`, rien d'autre. Exact.
  - Recomptage des chiffres du constat : `features/match-replay/` hors tests et hors `test/` =
    **203 fichiers**, **38** `.tsx` de premier niveau, **1 seul** `useCapability`
    (`useReplayTimeline.ts:24,117`, capability `media`). Exact.
  - `grep -rn "film\." apps/web/src` (hors tests) = **40 occurrences, toutes en commentaire**,
    zéro comme clé de capability. Exact.
- Ce que l'auditeur n'a pas vu :
  1. **Les cinq clés fines SONT exposées au web par un endpoint existant.**
     `apps/go-api/internal/api/handlers/capabilities.go:55` monte
     `GET /api/v1/titles/{slug}/capabilities`, dont le corps est
     `Capabilities: set.All()` (`:97-101`) — c'est-à-dire la **totalité** du
     `capabilities.toml`, sans filtrage : `internal/games/mappings/capabilities.go:63-73`
     recopie la map entière, et `loader_capabilities.go:32-64` ne valide QUE les statuts,
     jamais les clés (« Les CLÉS de capability NE sont PAS validées ici »). Pour
     `halo_infinite` la réponse porte donc `film.kill_source`, `film.weapon_shots`,
     `film.kill_positions`, `film.usage_summary`, `film.bomb_stats`.
     L'affirmation du constat — les clés fines « ne sont ni mirrorées **ni exposées** » — est
     fausse sur son second terme.
  2. **Cet endpoint est monté au même endroit et dans les mêmes conditions que
     `/field-mappings`, que le front consomme déjà au boot.**
     `apps/go-api/internal/api/server_apiv1.go:235` (field-mappings) et `:238-239`
     (capabilities) : même routeur `r`, même `apiOpt`, **aucun middleware d'auth**
     (contrairement aux handlers voisins l.185-213 qui, eux, sont enveloppés dans
     `RequireAuth`). Le front lit déjà `/field-mappings` par ce chemin
     (`apps/web/src/lib/i18n/fieldMappings.ts:1-19`, TanStack Query, une requête par
     (slug, locale), cache infini) — le patron de consommation existe, testé, en place.
  3. **L'endpoint est dans le contrat OpenAPI et déjà typé côté web** :
     `apps/web/src/lib/api/generated.ts:3785-3793` et `:19977-20013` (`getTitleCapabilities`),
     plus `/titles/{slug}/feature-matrix` (`:3853-3861`, `:20118`).
  4. **Un second canal sert déjà ces clés fines à du code React** : la page admin des titres
     reçoit `declared_capabilities` (`apps/web/src/features/admin/titles/AdminTitlesPage.test.tsx:45`
     — `{ 'match.history': 'supported' }`), preuve que le registre fin traverse déjà la
     frontière HTTP jusqu'au front.
  5. Mesure annexe, du même paragraphe : le préambule de l'audit écrit « halo_infinite
     déclare 20/20 capabilities produit ». Recompté sur pièces
     (`apps/go-api/internal/domain/title/registry.go:298-323` vs `:31-134`) : le dépôt
     déclare **21** constantes `Cap*`, et `halo_infinite` en accorde **18** — il ne déclare
     ni `weapon_accuracy` (retirée le 2026-09-01, commentaire `:314-322`), ni
     `spartan_customizer`, ni `native_kill_mechanics`.
- Conséquence réelle reformulée en une phrase : le contrat serveur publie déjà les cinq clés
  `film.*` sur un endpoint public, inconditionnel, versionné ETag et typé côté web — ce qui
  manque n'est pas un moyen de gater sur le film mais **un client pour un canal existant**,
  donc ni un P1 ni une « cause racine », et le traitement v2 proposé (inventer un champ
  `data_capabilities` dans `TitleSummary`) redouble un mécanisme déjà livré.

## Constat 2 — route de rejeu gatée par `matchmaking` seule : TIENT (gravité → P2)

- Ce que j'ai vérifié :
  - `apps/web/src/routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay.tsx:52-60`
    → `<RouteCapabilityGate capability="matchmaking">`. Exact.
  - `config/titles/halo_5/title.toml:22` `status = "active"` et `:48-49` `capabilities = [ "matchmaking", …]`
    → la porte est ouverte pour halo_5, et le titre est bien SERVI (pas `coming_soon`).
  - Les 3 requêtes film : `replay.tsx:72` `useMatchReplay(playerSlug, matchId)` sans 3e argument
    (`queries.ts:29,41` : `enabled = true` par défaut), `:82` `useReplayMapBackground`
    (`queries.ts:54-68`, `enabled: !!playerSlug && !!matchId`), `:90` `useReplayMapCallouts`
    (`queries.ts:113-127`, idem). `useReplayMapImage` est bien gaté par `!!background`
    (`replay.tsx:83`) donc ne part pas. **3 requêtes, pas 4** : le compte de l'auditeur est juste.
  - Aucune porte en amont : le layout parent `t/$titleSlug.tsx:30-47` ne fait que
    `resolveTitleGate` (titre connu/actif), aucune capability.
- Ce qui confirme, et ce qui la nuance :
  - **La « décision » citée par l'auditeur n'en est pas une réglée** : l'en-tête de
    `replay.tsx:5-9` affirme conserver la garde `matchmaking`, mais sans date ni condition de
    reprise. Le vrai enregistrement est `.ai/PLAN_FINALISATION_REJEU_2D.md:414-417`, item
    **3.5 « Poser la capability »**, coché `[ ]` (non fait) : « Déclarer `film.replay2d` et y
    brancher la route et le lien. Aujourd'hui la seule porte est un 404 sur un fichier absent :
    un titre qui ne sait pas produire de rejeu n'a aucun moyen de le dire. » Le dépôt dit donc
    lui-même que le constat est vrai et non traité. `.ai/V7.5/REGISTRE_REPORTS.md:493` ne
    l'énonce qu'en incise, à l'intérieur d'une entrée **SOLDÉE** sur les médias : ce n'est pas
    un report avec condition de reprise.
  - **Ce que l'auditeur n'a pas vu, et qui abaisse la gravité : aucun chemin produit ne mène à
    cette route sur halo_5.** Le bouton de la fiche (`MatchHeader.replayLink.tsx:36`
    `if (!available) return null`), la colonne d'Explorer et celle des Synergies
    (`lib/match-nav/MatchReplayLink.tsx:36`, même garde sur `has_replay`) ne rendent rien ;
    et la bascule de titre navigue vers `…/home`, jamais vers le chemin courant
    (`components/shell/TitleSwitcher.tsx:55-58`). L'exposition se réduit à une URL tapée ou
    mise en favori.
- Conséquence réelle reformulée en une phrase : la porte capability manque bien (le plan du
  dépôt la réclame explicitement et l'item est ouvert), mais sur halo_5 la page n'est
  atteignable que par URL directe et rend un état vide honnête — défaut réel, exposition
  marginale : **P2**.

## Constat 3 — `MatchKillDistanceSection` rendue sans porte, motif faux sur halo_5 : TIENT (P1)

- Ce que j'ai vérifié :
  - `apps/web/src/features/match-view/MatchKillDistanceSection.tsx` : **0 `return null`** dans
    tout le fichier (relu de bout en bout, 120 lignes) ; le `SectionCard` est ouvert
    inconditionnellement (`:87-98`) et la branche vide imprime `t.killDistanceEmpty` (`:99-100`).
  - Montage sans wrapper : `MatchViewPage.tsx:355-359`, dans `activeTab === 'summary'` (`:315`).
    J'ai relu `:290-359` : aucun `FeatureGate` ni condition en amont (le seul `FeatureGate` de
    l'onglet est celui des médias, `:381`).
  - Doc inversée confirmée mot pour mot : `MatchViewPage.tsx:352-354` « Rend null sans donnée
    mesurée … : pas de wrapper ici. »
  - Le message : `match-view/i18n.ts:398-399` — « Distances non mesurées sur ce match — elles
    demandent le décodage du film (positions du tueur et de la victime), qui n'a pas encore été
    joué ici. »
  - Côté Go, le champ est vide **par construction** pour halo_5 :
    `internal/api/wire/registry_pages.go:482-489` — `!data.Capabilities().Has(games.CapFilmKillSource)`
    ⇒ repo nil ⇒ `kill_distance_by_weapon` jamais peuplé.
- Ce qui confirme :
  - La décision utilisateur datée existe bien (en-tête `:11-14`, « 2026-09-02, retour user
    “je ne vois rien du tout” ») mais elle porte sur les matchs **Halo Infinite** non encore
    backfillés — « tant que le backfill de masse n'a pas tourné ». Elle ne dit rien d'un titre
    sans décodeur, où le motif affiché (« pas encore joué ici ») est simplement faux. Elle
    n'abaisse donc pas la gravité : elle justifie l'état vide, pas son affichage hors film.
  - halo_5 est `status = "active"` : la carte paraît sur l'onglet Général de **chaque** match.
- Conséquence réelle reformulée en une phrase : sur un titre actif sans décodeur de film,
  chaque fiche de match affiche une carte qui promet une mesure qui n'arrivera jamais, et le
  commentaire de montage décrit l'inverse de ce que le composant fait.

## Constat 4 — `SessionUsageSection` : `unsupported` traité comme `empty` : TIENT (P1)

- Ce que j'ai vérifié :
  - `apps/web/src/features/session-detail/usageLogic.ts:472-485` : `usage == null` ⇒ `hidden` ;
    `!usage.available` ⇒ `{ kind: 'empty' }` **quelle que soit** la raison, `unsupported`
    comprise (`:477-483`). Exact.
  - `SessionUsageSection.tsx:111-130` : `hidden` ⇒ `null` ; `empty` ⇒ `SectionCard` titrée
    `t.blockUnavailableTitle`, soit « Usages d'équipement, socles et objectifs »
    (`usageI18n.ts:105`), avec « Ce titre ne publie pas de résumé d'usage des films. »
    (`usageI18n.ts:108`). Exact.
  - Côté Go : `internal/service/session_page_usage.go:53-61` — ≥ 1 match et repo nil (cas halo_5,
    DI gatée `film.usage_summary`, cf. `:33-36`) ⇒ `UnavailableReason: SessionUsageUnsupported`,
    `Available` laissé à faux. Appelé sans condition : `session_page_service.go:287`.
- Ce que j'ai cherché en réfutation, et qui n'existe pas :
  - **Aucune prescription de plan ni de handoff n'impose d'afficher la raison.**
    `.ai/PLAN_SESSION_USAGE_BDD_EXECUTION.md:220` ne parle que du serveur (« degradation
    unsupported/load_failed jamais 500 ») ; la section front, lot S3 (`:119-157`), ne dit rien
    de l'état indisponible ; et `.ai/HANDOFF_SESSION_USAGE_BDD_2026-09-04.md:159-161` dit
    l'inverse du comportement livré : « Un titre sans film ne produit rien, **proprement**. »
    La seule justification est l'en-tête du composant lui-même (`:23-26`) — auto-documentation,
    pas une décision datée.
- Conséquence réelle reformulée en une phrase : sur halo_5, toute session portant au moins un
  match affiche une carte dont le seul contenu est l'explication de sa propre inutilité — le
  bloc mort que le mécanisme de capability existe précisément pour éviter.

## Constat 5 — filtre « Avec rejeu / Sans rejeu » sans porte : TIENT (P1)

- Ce que j'ai vérifié :
  - `apps/web/src/features/explorer/ExplorerPage.matchesMode.tsx:228-245` : le `select`
    `replayScope` est rendu nu, dans la ligne 1 des filtres, sans condition.
  - Recherche exhaustive des portes dans ce fichier (`grep -n "useCapability|replayScope|
    ReplayScope|columnVisibility|replayAvailable|FeatureGate"`) : 8 lignes seulement — l'import
    `:15`, le type `:21`, les props `:33,39,114,120`, l'unique `useCapability` `:156` (sur
    `ranked`, pour les paliers de skill), et le `select` `:229-230`. **Aucune** porte sur le
    rejeu : ni `useCapability`, ni `columnVisibility`, ni `replayAvailable`.
  - L'incohérence interne alléguée est réelle : `ExplorerMatchesTable.tsx:352`
    `useCapability('waypoint_match_url')` puis `:946` `{ waypoint: waypointCapability && … }`
    injecté dans `columnVisibility` (`:958`), alors que la colonne `id: 'replay'` (`:447`)
    ne figure dans aucune carte de visibilité.
- Nuance mesurée (qui n'abaisse pas la gravité) : « Avec rejeu ramène toujours 0 » n'est
  peut-être pas propre à halo_5 — `MatchHeader.replayLink.tsx:26-28` affirme « En production
  aucun artefact n'est produit aujourd'hui ». Si cette assertion du dépôt est à jour, le filtre
  est inopérant sur les deux titres en prod, ce qui élargit le constat au lieu de le réduire.
- Conséquence réelle reformulée en une phrase : la barre de filtres de l'Explorer expose sur
  halo_5 une commande à trois états dont l'un ne peut structurellement rien ramener, là où sa
  voisine Waypoint, dans le même tableau, est correctement gatée par capability.

## Constat 6 — colonnes d'Assaut gatées par `objective_stats` : RÉFUTÉ

- Ce que j'ai vérifié :
  - Le fait littéral est exact : `MatchObjectivesSection.tsx:71` `useCapability('objective_stats')`
    et `:90` `if (!hasObjectiveCap || mode == null || withObjective.length === 0) return null` ;
    `BOMB_COLS` vit en `MatchScoreboard.logic.ts:237-243`. Aucun `film.bomb_stats` côté web.
  - **Mais la conséquence énoncée est impossible.** Les champs `bomb_*` n'existent dans la
    charge utile que si le titre déclare `film.bomb_stats` :
    `internal/api/wire/registry.go:490-493` →
    `WithBombStats(r.capabilitiesForPDB(pdb).Has(games.CapFilmBombStats))`, puis
    `internal/platform/duckdb/match_view_repo_scoreboard.go:61-73` →
    `if r.bombStats { bombByXUID := loadMatchBombStats(...) }` : sans la capability, **la
    requête n'est même pas exécutée** (« Un titre qui ne la déclare pas ne paie même pas la
    requête »).
  - Côté web, `BOMB_COLS` ne sort QUE de `objectiveColsFor('bomb')`
    (`MatchScoreboard.logic.ts:288-289`), et `'bomb'` ne sort QUE de `detectObjectiveMode`
    quand une ligne porte `bomb_detonations | bomb_arms | bomb_grabs` (`:268`). Pas de champs
    ⇒ pas de mode ⇒ pas de colonnes. La chaîne est fermée.
  - Le dépôt épingle exactement ce raisonnement dans un test :
    `apps/web/src/features/match-view/objectivesBomb.test.ts:7-11` — « c'est ce qui distingue
    “un titre sans la capability `film.bomb_stats`” (aucune colonne servie -> section masquée)
    d'un match d'Assaut mesuré ».
  - Doctrine cohérente : `config/titles/halo_infinite/mappings/capabilities.toml:140-141` —
    « elle gouverne la PRODUCTION et l'EXPOSITION de ces cinq statistiques ». La porte fine est
    posée au site qui fait autorité (l'exposition), pas dupliquée sur le client.
- Ce que l'auditeur n'a pas vu : `objective_stats` n'est pas « la porte des colonnes bomb_* »
  mais un pré-filtre grossier de la section entière — dont les colonnes non-Assaut, elles,
  viennent bien de l'API. Le scénario redouté (« un titre qui déclarerait `objective_stats`
  sans décodeur de film ouvrirait la porte aux colonnes `bomb_*` ») ne peut pas se produire.
  Le seul accouplement résiduel est l'INVERSE — un titre déclarant `film.bomb_stats` sans
  `objective_stats` verrait ses cinq colonnes servies puis masquées par le web — et il est
  vide aujourd'hui (halo_infinite déclare les deux, `registry.go:313` et
  `capabilities.toml:142` ; halo_5 aucune des deux).
- Conséquence réelle reformulée en une phrase : la double porte existe et fonctionne — fine
  côté serveur (production/exposition), grossière côté web (section) — donc la conséquence P1
  annoncée est nulle, et non « latente ».

## Bilan : 3 tiennent, 2 réfutés, 1 requalifié

| Constat | Verdict |
|---|---|
| 1 — aucune `film.*` servie au web | **RÉFUTÉ** (`GET /titles/{slug}/capabilities` sert les 5 clés, sans auth, dans le contrat et déjà typé `generated.ts`) |
| 2 — route rejeu gatée `matchmaking` | **TIENT — gravité P1 → P2** (aucun chemin produit n'y mène sur halo_5 ; item de plan ouvert `PLAN_FINALISATION_REJEU_2D.md:414`) |
| 3 — `MatchKillDistanceSection` sans porte + doc inversée | **TIENT (P1)** |
| 4 — `SessionUsageSection` : `unsupported` traité en `empty` | **TIENT (P1)** |
| 5 — filtre rejeu Explorer sans porte | **TIENT (P1)** |
| 6 — colonnes d'Assaut gatées `objective_stats` | **RÉFUTÉ** (porte fine `film.bomb_stats` appliquée côté Go, chaîne fermée, test dédié) |
