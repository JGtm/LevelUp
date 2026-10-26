# Vérification adverse V-GO-A2

Dépôt lu à `081871f09` (descendant de `736ccf3c3`, aucun des fichiers cités n'a bougé entre les
deux). Lecture seule, chemins absolus sous
`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/apps/go-api/`.

## Constat 1 — pipeline `source_tag` écrit cinq fois, populations divergentes : RÉFUTÉ

- **Ce que j'ai vérifié**
  - Comptage propre :
    `git grep -n "GROUP BY k.source_tag\|GROUP BY k.feed_killer_xuid, k.source_tag\|GROUP BY source_tag" -- 'apps/go-api/internal/**/*.go' | grep -v _test`
    → 5 sorties : `internal/api/wire/registry_weapon_coverage.go:183`,
    `internal/platform/duckdb/killsource_weapon_kills_repo.go:242`,
    `internal/platform/duckdb/killsource_weapon_scope.go:149`,
    `internal/platform/duckdb/match_view_repo_weapons_source.go:44`,
    `internal/sync/citations_weapons_source.go:88`. Le chiffre 5 est exact.
  - `internal/analysis/campaign_exclusion.go:34-39` : `campaignExcludedVariantIDs` ne porte
    **que** `"halo_5"` (2 GUID). `:98-102` : `sqlExcludeByMatchIDSubquery` rend `""` quand la
    liste est vide. Doc du fichier, `:29-33` : « un titre absent de la map (ex. halo_infinite …)
    → aucune clause émise (no-op) ».
  - `config/titles/halo_infinite/mappings/capabilities.toml:100` :
    `"film.kill_source" = "supported"`. `grep -n "film\." config/titles/halo_5/mappings/capabilities.toml`
    → **aucune sortie**. `internal/sync/killcollector/classifier.go:33-44` :
    `ClassifierPourTitre` rend `nil` sans `CapFilmKillSource`.
- **Ce que l'auditeur n'a pas vu**
  1. **La « divergence de population mesurable » vaut mesurablement ZÉRO.** Les cinq requêtes
     sont toutes conditionnées à un `port.KillSourceClassifier` non nil, qui n'existe que sur
     `halo_infinite` — et sur `halo_infinite` `excludeCampaignByMatchID` émet la **chaîne vide**.
     Les 2/5 qui l'appellent et les 3/5 qui ne l'appellent pas lisent donc, dans toute exécution
     atteignable, exactement la même population. L'auditeur a lu la présence/absence d'un appel
     sans vérifier que le helper est un no-op documenté sur le seul titre concerné.
  2. **Deux des trois étapes du « pipeline » sont déjà centralisées.** « Traduire par le
     classificateur » = `port.KillSourceClassifier.KillSourceRegistryKey`, une seule
     implémentation appelée par les 5. « Résoudre le registre » =
     `internal/platform/duckdb/weapon_resolver.go:284` (`resolveWeaponKeyDimensions`), appelé
     par les 3 qui résolvent le registre (`killsource_weapon_scope.go:66` et `:223`,
     `killsource_weapon_kills_repo.go:285`, `match_view_repo_weapons_source.go:77`). Ce qui est
     écrit 5 fois est un squelette SQL de 4 lignes, avec **5 portées et 3 clés de GROUP BY
     différentes**.
  3. **Deux des cinq ne font pas le pipeline décrit.** `citations_weapons_source.go:78-108`
     résout vers `name_en` par sa propre table (`src.keyNames`), contrat différent du registre.
     `registry_weapon_coverage.go:172-198` ne résout **rien** : il compte
     `ResolvedRegistry` / `Unresolved` / `TopUnresolved` pour la page admin de couverture —
     y exclure la campagne sous-estimerait la couverture du classificateur.
  4. **La doctrine du dépôt justifie explicitement l'absence de token sur une requête
     mono-match.** `internal/platform/duckdb/campaign_exclusion_guard_test.go`, allowlist :
     `"Q17PlayerMatchStats": "requête MONO-MATCH (WHERE match_id = ? AND xuid = ?) : aucune
     agrégation d'historique → la Campagne ne peut pas polluer un affichage de liste."` C'est
     exactement le cas de `match_view_repo_weapons_source.go:44` (`WHERE k.match_id = ?`) et de
     `citations_weapons_source.go:88` (`WHERE k.match_id = ? AND k.feed_killer_xuid = ?`).
  5. **Les cinq sites sont le bras Infinite de cinq lecteurs bi-branches préexistants**, pas
     cinq copies d'une fonction unique : l'allowlist de
     `internal/archlint/no_weapon_kills_sql_test.go` les nomme un à un comme « repli … branche
     sans classifier » (`explorer_repo.go`, `queries_home_citations.go`, `sync/citations.go`,
     `registry_weapon_coverage.go`, `weapon_kills_repo.go`). Les unifier suppose d'abord
     d'unifier les cinq lecteurs legacy — bien au-delà de ce que le constat énonce.
- **Ce qui survit** : une seule phrase. L'en-tête `killsource_weapon_scope.go:7-8` nomme « la
  statistique d'arme du moteur de citations » parmi ses trois surfaces, alors que ses seuls
  appelants sont `explorer_repo.go:570` et `killsource_weapon_scope.go:214` (HomeRepo). Le
  moteur de citations a reçu sa propre copie plus tard (`67303538a`, A2.6, postérieur à
  `d71d135f9` qui a écrit l'en-tête). Défaut de commentaire sur une ligne, pas de divergence de
  données.
- **Conséquence réelle** : les cinq lectures rendent le même corpus ; seul l'en-tête d'un
  fichier annonce une surface qu'il ne sert pas.

## Constat 2 — refus d'ambiguïté écrit six fois sur deux clés : RÉFUTÉ

- **Ce que j'ai vérifié** — comptage propre
  (`git grep -n "count(DISTINCT\|appariementUnique\|compterInstantsAmbigus\|conflict" -- persist migration platform/duckdb service | grep -v _test`) :
  6 points de décision confirmés (`persist/kill_events_merge.go:237`,
  `migration/steps_shared_kill_events_credit_base.go:128-133`,
  `platform/duckdb/queries_match.go:505-506`, `:544-550`,
  `platform/duckdb/kill_distance_repo.go:133`,
  `service/match_view_killfeed_weapon.go:97-112`). Le chiffre est exact ; la lecture ne l'est pas.
- **Ce que l'auditeur n'a pas vu** — les six écritures relèvent de **trois faits distincts**,
  chacun avec sa raison écrite et mesurée :
  1. **Clé producteur `(match_id, time_ms)` — l'identité est l'INCONNUE en cours de
     résolution.** `kill_events_merge.go:31-35` : « L IDENTITE EST UN CONTROLE, PAS UN CRITERE
     DE JOINTURE. Sur les 73 589 lignes appariees : 0 divergence … Le seul ecart est une
     ABSENCE (631 victimes et 754 tueurs pour lesquels le film n a pas resolu de xuid) — c est
     exactement la population qu une clef a quatre colonnes perdrait. » Clefer la fusion sur le
     tueur **jetterait 754 lignes**. Les deux clés ne sont pas deux copies d'une règle, ce sont
     deux règles qui ne peuvent pas fusionner.
  2. **Le remède proposé par le constat démontre lui-même la non-duplication.** `(match_id,
     time_ms)` est **strictement plus grossière** que `(feed_killer_xuid, time_ms)` : deux
     tueurs différents qui tuent au même millième sont ambigus pour le producteur et univoques
     pour chaque lecteur. Un booléen `attributable` unique écrit par le producteur
     supprimerait arme et assistant sur **tout double kill simultané par deux joueurs
     distincts** — le cas courant en BTB.
  3. **Les trois HAVING lecteurs portent sur trois jeux de colonnes différents parce qu'ils
     publient trois choses différentes**, et chacun le dit : `queries_match.go:483-487`
     (« un double kill … peut porter la MÊME arme mais des CATÉGORIES différentes — la HAVING
     sur `source_tag` seule ne le protégerait pas ») ; `:510-514` (« l'assistance est lue même
     quand la source de dégât ne l'est pas … La conditionner à `source_tag IS NOT NULL`
     perdrait des assistants mesurés ») ; `kill_distance_repo.go:107-114` (une seule colonne
     publiée). Une clause partagée sur-refuserait ou sous-refuserait.
  4. **`memeVictime` NE PEUT PAS être un `count(DISTINCT)`**, et c'est documenté :
     `match_view_killfeed_weapon.go:83-91` — « Comparer un xuid vide à un xuid vide déclarerait
     « même victime » pour deux bots différents, et le feed nommerait le mauvais. » Les
     victimes bot arrivent avec un xuid NULL/vide ; c'est précisément le fantôme d'acteur que
     garde `kill_events_source_guard_test.go / TestPasDeXuidNormaliseEnChaineVide`. Règle
     différente pour une forme de donnée différente, pas une copie.
  5. **La conséquence décrite est fausse.** « Un double kill au même millième porte donc une
     arme en base » — non : `appariementUnique` (`:237`) refuse d'enrichir dès que l'instant
     porte deux morts, donc **aucun `source_tag` n'est écrit** ; Q21b ne rend alors rien
     (`source_tag IS NOT NULL`). Les trois couches dégradent de façon cohérente : la base
     crédit nomme la victime (elle fait autorité sur la LISTE des morts), le film refuse
     l'arme.
- **Conséquence réelle** : trois décisions sur trois clés que le schéma rend non fusionnables,
  chacune motivée sur mesures — pas six copies d'une même règle.

## Constat 3 — la Match View lit trois fois la MÊME ligne : RÉFUTÉ

- **Ce que j'ai vérifié**
  - `platform/duckdb/queries_match.go:424-434` (Q20), `:493-507` (Q21b), `:530-551` (Q21c) ;
    appels en `match_view_repo_extras.go:94`, `:141`, `:197`.
  - `analysis/timeline/correct_events.go` — `grep -n "^func "` → `CorrectKillSourceRaws:83`,
    `CorrectKVPairRaws:101`, `CorrectKillAssistRaws:118`. Les trois lignes citées sont exactes.
- **Ce que l'auditeur n'a pas vu**
  1. **Q20 n'est pas une troisième lecture pour le kill feed : c'est une lecture préexistante à
     cinq autres consommateurs.** `service/match_view_builders_combat.go` :
     `applyKVSynthesisIfNeeded` (`:56`), `buildTugEvents` (`:82`), `buildKDEvents` (`:96`),
     `buildKillerVictimPairs` / antagonistes (`:116`) ; plus `buildNemesisMap`
     (`match_view_builders_team.go:9`). Le feed consomme une **COPIE corrigée T0**,
     `kvPairsFeed` (`match_view_data_loaders.go:59-62`, affectée `:564`) — la doc de
     `CorrectKVPairRaws` le dit : « les consommateurs historiques des paires (tug-of-war, KD
     timeline) restent sur l'horloge brute ». Q20 s'exécuterait même sans kill feed.
  2. **Les trois ne lisent pas « la même ligne ».** Q20 projette **une ligne PAR MORT**, sans
     `GROUP BY`, sans filtre, et sert délibérément les tueurs bot à `feed_killer_xuid` NULL.
     Q21b et Q21c sont des **AGRÉGATS** sur `(feed_killer_xuid, time_ms)` qui excluent
     explicitement `feed_killer_xuid IS NULL` et **suppriment** les groupes non unanimes. Une
     projection unique ne peut pas être les deux à la fois.
  3. **« Trois allers-retours DuckDB » : ils sont CONCURRENTS.** `match_view_data_loaders.go`
     les lance dans le même errgroup — `goLoad(gctx, g, matchID, "kill_sources", …)` (`:157`),
     `"kill_assists"` (`:162`), `"kv_pairs"` (`:171`) — parmi une quinzaine d'autres. Le coût
     marginal n'est pas trois allers-retours séquentiels.
  4. **« Q20 sans `publishable` » est la doctrine, pas un oubli.** `kill_events_source.go:44-50` :
     « `publishable` NE FILTRE PAS LES LIGNES … Filtrer dessus coûterait **47 037 morts** et
     viderait 27 % des matchs à l'écran … La colonne se lira le jour où une surface affichera
     l'ARME d'une mort. » Q20 n'affiche aucune arme. « Poser une victime sur une clé que le code
     refuse pour l'arme » est la dégradation VOULUE (nommer la mort, ne pas inventer l'arme).
  5. **Les « trois corps identiques » ne sont pas factorisables en Go.** Les paramètres de type
     Go ne permettent ni de lire ni d'affecter un **champ** de struct sur un type paramétré :
     factoriser exige d'ajouter des méthodes accesseur à trois types `domain`. Ce n'est pas une
     copie évitable. Et la fonction voisine `CorrectEventRaws:54` n'est **pas** identique (elle
     réalloue un `*int64` pour ne pas muter l'entrée partagée).
- **Conséquence réelle** : deux requêtes spécifiques au feed s'ajoutent, en parallèle, à une
  lecture qui servait déjà cinq surfaces — et leurs filtres divergent parce que les faits qu'elles
  publient divergent.

## Constat 4 — `BuildKillPositions` reçoit l'origine d'horloge là où la doc nomme `bestDeathOffset` : RÉFUTÉ

- **Ce que j'ai vérifié** — `analysis/replay/killpos.go:121-137`,
  `analysis/replay/origin.go:76-124`, `sync/killcollector/positions.go:183-210` et `:262-288`,
  `platform/duckdb/kill_distance_repo.go:115-133`.
- **`originUS` et `DeathOffsetMS` sont la MÊME grandeur, et le dépôt le démontre.**
  - `ScanClockOrigin` (`origin.go:89-99`) rend l'horodatage moteur du **premier paquet du
    chunk 1**, documenté `:76-77` comme « **le zero de l'horloge sur laquelle les highlight
    events sont dates** ».
  - `KillRef.TimeMS` vient de `killRefsFromDeaths` (`positions.go:270`) →
    `persist.KillEventInsert.TimeMS` → `killsource.Kill.TimeMS`, c'est-à-dire des ms sur cette
    même horloge highlight (`killpos.go:59-61` établit l'identité du champ). Donc
    `tUS = k.TimeMS*1000 + originUS` est une conversion **EXACTE** vers les µs moteur, l'horloge
    de `filmdec.BipedPosition.TimestampUS`.
  - Identité algébrique lue dans `resolveOriginMs` (`origin.go:108-124`) : le contrôle affirme
    `read = (firstPosUS − filmClockUS)/1000 ≈ control = firstPosUS/1000 − deathOffsetMS`, d'où
    **`deathOffsetMS ≈ filmClockUS/1000 = originUS/1000`**. Écart mesuré 16 à 81 ms sur les
    quatre films témoins (`origin.go:41-45`), gardé à 1 000 ms (`originControlToleranceMS`).
- **Ce que l'auditeur n'a pas vu**
  1. **Le code passe la lecture EXACTE là où `bestDeathOffset` est l'ESTIMATEUR de la même
     grandeur** — quantifié au pas de 10 ms et apparié dans une fenêtre de 150 ms
     (`origin.go:64-65`). Appliquer le remède proposé (passer `owners.DeathOffsetMS`)
     **dégraderait** la précision. La doc `killpos.go:123-125` décrit correctement la grandeur
     (« le décalage entre l'horloge du fil des morts et celle du film ») ; « cf.
     bestDeathOffset » renvoie au témoin de cette grandeur, ce n'est pas une déclaration que le
     paramètre en est la sortie.
  2. **`owners` n'est pas jeté** : `positions.go:204-208` utilise `owners.LivesTotal`,
     `owners.DeathsNamed`, `owners.IndexReadings` dans le message d'erreur. Seul le champ
     `DeathOffsetMS` est inutilisé — parce que l'équivalent exact a déjà été lu ligne 183.
  3. **`offsetUS` n'entre jamais dans le `time_ms` persisté.** `toKillPositionRows`
     (`positions.go:277`) écrit `TimeMS: int(p.TimeMS)`, soit les mêmes ms d'horloge highlight
     que `match_kill_events.time_ms` — et c'est ce qui rend la jointure de
     `kill_distance_repo.go:118-121` (`kp.time_ms = e.time_ms`) exacte. `offsetUS` ne choisit
     que **l'échantillon de position lu**.
  4. **Une divergence grossière n'est pas silencieuse** : `positionOf` (`killpos.go:178-191`)
     rend `nil` au-delà de `killPosToleranceUS` (120 ms). Les écarts de référentiel que
     `origin.go:20-21` mesure (3,7 s à 39,8 s) **videraient** la table via
     `KillPosReport.Dropped`, qui est rendu à l'appelant et journalisé — pas un décalage muet.
- **`kill_positions` est bien servie** : `kill_distance_repo.go:115-133` lit
  `kill_positions_latest` pour la distance de kill du produit. Le constat a raison sur ce point,
  et c'est le seul.
- **Conséquence réelle** : la conversion est exacte, la doc décrit la bonne grandeur, et le
  remède proposé introduirait l'imprécision qu'il prétend corriger.

## Constat 5 — tolérance 5 ms ×4 et conversion `HighlightEvent → RawEvent` ×3 : TIENT (gravité → P2)

- **Ce que j'ai vérifié** — comptage propre
  (`git grep -n "toleranceMS\|killPairToleranceMs\|creditToleranceMS" -- apps/go-api | grep -v _test`).
- **Ce qui confirme (moitié tolérance)** : **4 littéraux indépendants**, confirmés à la ligne —
  `games/halo_infinite/events.go:31` (`const killPairToleranceMs = 5`), `sync/collect.go:146`
  et `sync/engine_highlight_events.go:383` (`const toleranceMS = int64(5)` chacun),
  `sync/killcollector/credit.go:83` (`const creditToleranceMS = int64(5)`). Le propriétaire
  `analysis/killer_victim.go:45` n'exporte **aucune** constante (« défaut recommandé : 5 » en
  prose). Aucun test ne relie les quatre (`grep` sur les trois identifiants dans les `_test.go`
  → une seule sortie, sans rapport : `objectiveevents/score_test.go`). L'invariant est écrit
  noir sur blanc en `credit.go:79-82` (« c est EXACTEMENT la valeur qu emploie la completion …
  Une autre valeur produirait un jeu de couples different ») et rien ne le tient. 4 > 2 :
  CLAUDE.md n° 6 enfreint. **Cette moitié tient.**
- **Ce que l'auditeur n'a pas vu (moitié conversion — RÉFUTÉE)** : il n'y a que **2** copies
  identiques, pas 3. `sync/collect.go:135-145` et `sync/engine_highlight_events.go:371-382` sont
  textuellement identiques (même filtre kill/death, même `strconv.FormatUint`, mêmes 4 champs).
  La troisième, `games/halo_infinite/events.go:101-111` (`toRawEvents`), opère sur un **type
  d'entrée différent** (`canonical.HighlightEvent`, XUID déjà `string`), recopie **3 champs**
  (pas de `Gamertag`) et n'applique **aucun** filtre de type d'event. Ce n'est pas une copie des
  deux autres. 2 copies = au seuil, pas au-dessus.
- **Conséquence réelle** : un vrai littéral magique en 4 exemplaires portant un invariant écrit
  et non testé (à traiter) ; la duplication de conversion annoncée n'existe pas.

## Constat 6 — conversion film↔match réécrite neuf fois : RÉFUTÉ

- **Ce que j'ai vérifié** — comptage propre :
  `grep -rn "deathOffsetMS\|DeathOffsetMS" apps/go-api/internal/analysis/replay/*.go | grep -v _test | grep -E "/ ?1000|\* ?1000"`
  → **8 lignes de code** portent l'arithmétique, pas 9.
- **Ce que l'auditeur n'a pas vu**
  1. **La 9ᵉ citée ne porte pas la formule.** `killpos.go:137` fait
     `k.TimeMS*1000 + offsetUS` — `offsetUS` est un horodatage de paquet en **µs**
     (`originUS`), jamais `deathOffsetMS`. Ni le même opérande, ni la même unité.
  2. **Deux des neuf sont le helper canonique lui-même.** `match_clock.go:38` (`frameOfMatchMS`)
     et `:53` (`matchMSOfFrame`) sont la source unique — les compter comme deux de leurs propres
     copies gonfle le total de 25 %.
  3. **`origin.go:116` est un contrôle DÉLIBÉRÉMENT indépendant.** `resolveOriginMs` existe pour
     confronter deux expressions de la même origine (`origin.go:31-51` : « Les deux chemins ne
     partagent aucune piece … Leur accord a moins de 100 ms est ce qui fonde la publication »).
     Le factoriser avec ce qu'il contrôle **détruirait** le contrôle.
  4. **`bomb_stats_document.go:90` calcule une autre grandeur** : `FilmClockOriginUS/1000 −
     DeathOffsetMS` est le pont **MANIFESTE↔match**, une constante de film, pas la conversion
     d'un instant — et `killpos.go:66-69` interdit explicitement de l'appliquer aux kills.
  5. **La déclaration « écrite une seule fois » porte sur une AUTRE conversion.**
     `match_clock.go:3` : « L'HORLOGE MATCH **<-> FRAMES**, ÉCRITE UNE SEULE FOIS ». Et
     `match_clock_guard_test.go` interdit précisément toute autre déclaration de
     `frameOfMatchMS` / `matchMSOfFrame` — **le garde-rail couvre exactement ce que l'en-tête
     annonce**. Il n'y a ni doc inversée ni garde-rail en défaut.
  6. **La règle invoquée est fausse.** « Factorisation abandonnée — le helper existe, les copies
     n'ont pas été migrées » : `matchClock` (`match_clock.go:24-30`) n'expose **aucune**
     conversion `paquetUS ↔ matchMS`. Il n'y a rien vers quoi migrer.
- **Ce qui survit** : 4 lignes dans **3 fonctions** portent `paquetUS/1000 ∓ deathOffsetMS` —
  `bomb_carries.go:56` (`bombHeldEventsOf`), `flag_carries_marker.go:41` et `:45`
  (`markFlagCarries`, même fonction, 4 lignes d'écart), `successions.go:68`
  (`attributeSuccessions`, sens inverse). 3 sites atteignent le seuil « à la 3ᵉ on centralise » ;
  c'est un constat P2 d'une tout autre taille que celui écrit.
- **Conséquence réelle** : 3 fonctions à corriger si le calage change, pas neuf ; et le fichier
  qui se dit unique l'est réellement pour ce qu'il revendique.

## Bilan : 1 tient, 5 réfutés, 1 requalifié

- **Réfutés** : constats 1, 2, 3, 4, 6.
- **Tient, gravité abaissée P1 → P2** : constat 5, sur sa seule moitié « tolérance » (4 littéraux
  `5`, invariant écrit, aucun garde-rail) ; sa moitié « conversion ×3 » est réfutée (2 copies).
- **Résidus non nuls dégagés des constats réfutés**, tous d'un rang inférieur à celui annoncé :
  - constat 1 → une phrase d'en-tête (`killsource_weapon_scope.go:7-8`) nomme le moteur de
    citations parmi ses surfaces alors qu'il ne le sert pas ;
  - constat 6 → 3 fonctions portent `paquetUS/1000 ∓ deathOffsetMS` sans helper ni garde-rail.
