# P1 — Inventaire des entités du film et de leurs liens (avant tout code)

> Lot **P1** du plan `.ai/PLAN_V2_RESTES_2026-09-07.md` (§2, lot P), amendé par
> `.ai/PLAN_ORCHESTRATION_2026-09-07.md` §4 (axes (i) et (j)) et §1.2 (conditions 1 à 3).
> Contrat d'exécution : skill `plan-execution`. **Aucun code produit** : ce document est le
> contrat des phases P2 à P5.
>
> Base : `feat/v75` = `22d76a738` (schéma 48, fusion Tactique `f2c8ddce1` incluse), branche
> `feat/p1-inventaire`, worktree dédié `LevelUp-wt-p1-inventaire`.
>
> **Règle de rédaction tenue** : chaque affirmation porte un `fichier:ligne` rouvert dans la
> session du 2026-09-07 sur ce worktree. Ce qui n'a pas été relu dans cette session n'entre pas.
> Les mesures citées viennent de rapports VERSIONNÉS, nommés à chaque fois.

---

## 0. VERDICT ROSTER (axe d) — il débloque S.3, il est en tête pour cette raison

**NON. Le film ne porte PAS les entrées et sorties de joueurs de façon exploitable.**

Les deux composants qui les porteraient existent, sont décodés, et n'ont aucun débit :

| Fait | Pièce |
|---|---|
| `player-active-in-game-component` (ti=5 i18, `R(1)`) est décodé et publié par hook | `apps/go-api/internal/analysis/filmdec/components_player.go:33`, `:53`, `:182-185` |
| `player-pending-join-in-progress-spawn-component` (ti=5 i19, `R(1)`) idem | `components_player.go:34`, `:54`, `:187-190` |
| Le hook (`SetPlayerStateHook`) n'a **aucun consommateur de production** : seuls des `_test.go` l'installent | `components_player.go:94` ; appelants : `filmdec/game_entities_walk_test.go:84`, `filmdec/game_entities_chain_test.go:129` (grep sur tout `apps/go-api` : 0 hors tests) |
| Débit mesuré : **163 lectures d'i18 et 105 d'i19 sur 22 films** (≈7 et ≈5 par film, pour 8 joueurs), valeurs 0/1 seulement | `.ai/V7.5/replay2d/registre_film/LOTBP_PHASE0.md` §P.0.1 (tableau des annonces par composant) et §P.0.5 (ligne « i18 / i19 arrivées-départs », verdict **NON TENU (dénominateur)**) |

Conséquence, conforme à la décision **D10** (`PLAN_ORCHESTRATION` §5) : **S.3 se fera par calage
`real_start_time` / `t0_quality`, comme son plan le prescrit, et APRÈS P2.** Rien dans le film ne
permet de faire autrement aujourd'hui ; passer par ti=5 fabriquerait une couverture de 5 % que le
paradigme condamnerait aussitôt.

**Ce que la source directe reste, faute de film** : la base. Les arrivées en cours de partie sont
déjà lues de `match_participants` (`JoinedInProgress` + `JoinMatchMS`) et consommées par les
relais de bots — `apps/go-api/internal/replaybuild/replaybuild.go:507-521`, contrat
`apps/go-api/internal/analysis/replay/successions.go:31-43`. C'est la même donnée que S.3 doit
recaler ; elle est déjà dans le pipeline de cuisson, S.3 n'a qu'à la porter au sync.

**Report inscrit** : le branchement de ti=5 i18/i19 (et de tout ti=5) appartient au **plan
décodeur d'après v7.5.0** (§0.6 du plan v2), en tant qu'item additif mesuré par la part de liens
directs. Il ne peut pas remonter la couverture seul : le canal parle 5 à 7 fois par film.

---

## 1. Table des entités que le décodeur expose déjà

Colonnes : **Identifiant** (type d'enregistrement + champ + `fichier:ligne` du lecteur) ·
**Durée de vie** · **Liens DIRECTS du film** · **Liens par REPLI** (calque + ligne exacte) ·
**Dispo décodeur actuel** / **Exige travail décodeur**.

Le détail de chaque ligne suit la table, avec les mesures.

| # | Entité | Identifiant dans le film (lecteur) | Durée de vie | Liens DIRECTS | Liens par REPLI (ligne du repli) | Dispo | Exige décodeur |
|---|---|---|---|---|---|---|---|
| E1 | **Index de joueur du film** | 5 bits précédant le xuid 64 bits dans les chunks de réplication — `replay/player_index.go:64-106` (via `weaponv3.ResolveXuidToPI`) | tout le film ; table exigée CONCORDANTE sur tous les chunks de réplication, sinon non publiée (`player_index.go:96-104`) | index ↔ xuid (26 lectures concordantes) ; index ↔ auteur d'un tir (`FireEvent.FilmIndex`, cf. `replay/owners.go:24-28`) ; index ↔ gamertag (fil des morts, `replay/identity.go:67-78`) | — (aucun : c'est LA lecture) | oui | non |
| E2 | **Slot de bipède** (trajectoires, ti=35) | `filmdec.BipedPosition.Slot` ; découpe en vies par trou > `lifeGapUS` — `replay/lives.go:117-119` (`buildLifeSpans`) | une vie = un slot entre deux trous de 5 s ; slot RÉATTRIBUÉ aux réapparitions et entre manches | position, vitalité, visée, marqueur de portage (image-clé) | **slot ↔ index/xuid : POINT AVEUGLE.** Pont par morts `replay/owners.go:188-224` (`bestDeathOffset` + `nameLivesByDeaths` `:196-199`), puis fermetures (`closures.go`), sièges de bot (`identity.go:207`), relais (`successions.go:61`), nommage final par occupation (`unnamed_lives.go:86`) | oui (pont) | **oui** pour un lien direct |
| E3 | **Slot d'entité statborg — joueur** (10..24 pairs) | `objectiveevents.StatRecord{TimeMS, Slot, Round, Comps}` — `objectiveevents/statborg.go:153-165`, datation `:199` | réattribué **d'une manche à l'autre** (mesuré : `d9781168` slot 22 = deux joueurs) — `objectiveevents/slotidentity_rounds.go:9-16` | son instant (horloge manifeste), sa **manche déclarée** (2 en-têtes de 5 bits), ses compteurs | slot ↔ xuid par **triplet K/D/A contre la feuille** (`slotidentity.go:82-113`) ou par **instants de mort** (`slotidentity_deaths.go:141-152`, seuils `:41-50` dont `deathInstantMin = 3`), résolu **par manche** (`slotidentity_rounds.go:39-58`) | oui | **oui** pour un lien direct |
| E4 | **Slot d'entité statborg — équipe** (6 et 8) | même enregistrement, `IsTeamSlot` — `objectiveevents/statborg.go:167-168` | tout le match | compteurs d'équipe (score, somme des frags en `comp 2 A`) | slot ↔ camp `team_0`/`team_1` par **score final** puis **somme des frags** — `replay/score_team_identity.go:34-45` ; refus explicite sinon (`ScoreIdentityUnresolved`) | oui | **oui** (le film ne nomme pas le camp) |
| E5 | **Objet d'objectif** (drapeau, crâne, bombe) — archétype `ti=42` | clé de vie `EquipmentLifeKey{Slot, Gen}` (`filmdec/equipment_creation_width.go:43`) + **mot MPP 32 bits** du record de création = identité de type (`filmdec/equipment_creation.go:99-118`, lecture `replay/ground_weapon_rules.go:388-393`) | création → piste delta → **silence dès qu'il est porté ou immobile** ; fin de vie = création suivante de la même clé (`replay/flag_objects.go:112-116`, boucle `:132-140`). Frontière de manche : **hypothèse §0.4 non vérifiée** | type de l'objet nommé par le TOML du titre (`config/titles/halo_infinite/mappings/replay_labels.toml:683-717`, table `LabelCatalog.ObjectiveObjects`, `replay/catalog.go:105-127` — champ `:127`) ; piste de l'objet LIBRE ; marqueur « ce slot porte quelque chose » (image-clé, `filmdec/keyframe_carrier_mark.go:5-31`, vues `:35-41`) | **portage ↔ objet** : géométrie socle/rayon + invariant « jamais son propre drapeau » — `replay/flag_assign.go:242` et l'en-tête `:5-58` · **objet ↔ équipe propriétaire** : `flag_spawn.team_index` du catalogue de carte — `replaybuild/flagspawns.go:5-10`, `:50` · **porteur** : compteur de statistique statborg (donc E3) | oui | **oui** pour l'attachement porteur↔objet |
| E6 | **Zone / colline** — archétype `ti=13` | slot `ti=13` + valeur de canal (tag 4 = index d'équipe) — `replay/zone_states.go:9-18` ; formes et identité du catalogue de carte (`InstanceID`, `SpatialRank`) — `replay/mapvar/objectives.go:215-222`, `replay/objectives_catalog.go:121`, `:133` | intervalles d'occupation par slot, sur l'horloge MOTEUR (division sans recalage — `zone_states.go:17-18`) | valeur du canal ↔ équipe (mesuré 100 % / 91,1 %) ; forme et `team_index` de la zone via `map_id` (asset UGC) | **slot ↔ zone** : **vote modal** sur la coïncidence sommet-de-jauge / capture attribuée géométriquement — `replay/zone_states_owner.go:191`, `modalZone` `:240-241` ; tolérance `zoneCaptureDistanceM = 5.0` `zone_states.go:34-37` · **action ↔ zone** : géométrie `replay/zone_attribution.go` | oui | **oui** pour l'identité de zone |
| E7 | **Véhicule** — archétype `ti=40` | slot `ti=40` (recensement d'images-clés) ; **réf 1 (domaine 1) de l'événement `unit_exit_vehicle`** nomme le véhicule — `filmdec/event_list.go:313-345` (`decodeExitRefs` `:321`) | vie bornée par le recensement d'images-clés (~20 s de pas) ; épisode d'occupation borné par les événements | **occupant ↔ événement** : réf 0 en bande bipède (sortie 95,5 %, embarquement 100 %) — `event_list.go:287-292` · **véhicule ↔ sortie** : 105/105 en bande `ti=40`, zéro bipède — `event_list.go:317-320` · siège `R(6)` `event_list.go:276-278` | **véhicule d'un épisode sans sortie** : géométrie 3 m — `replay/vehicle_rides_events.go:248-252`, rayon `:77` ; provenance interne `vehicleResolvedBy` `:282-301` · **occupant ↔ xuid** : `own.xuidAt(slot, instant)` `vehicle_rides_events.go:275`, `vehicle_rides.go:287` (donc E2) | oui | **partiel** : l'embarquement ne nomme pas le véhicule (0/15 sur 12 films, rapport V8 §2 cité `vehicle_rides_events.go:92-96`) |
| E8 | **Arme au sol / équipement / socle** — `ti=42` et `ti=37` | mêmes clés que E5 : `EquipmentLifeKey{Slot, Gen}` + mot MPP ; balayage commun `replay/ground_weapon_objects.go:25-45` | création → piste → **mise au repos, pas disparition** (`filmdec/equipment_creation_width.go:49-52`) ; disparition bornée par le recensement d'images-clés | identité de famille (mot MPP → `weaponv3.WeaponName`, `ground_weapon_rules.go:398-403`) · **ramasseur ↔ objet de catalogue** : événement natif `biped_pickup` (type 9) — `filmdec/biped_pickups.go:5-13, 27-40` | **`biped_pickup` ne donne PAS l'instance monde, donc pas le socle** (`biped_pickups.go:42-46`) · **socle du match ↔ socle du catalogue** : géométrie 1 m — `replay/map_weapon_pads.go:37-40`, `confirmePar` `:99-100` · **origine d'un ramassage** (socle/sol) : géométrie — `replay/pickup_origin.go:16-38` · **poseur d'un équipement** : `deployed`/`dropped`/`unknown` mesurés — `replay/equipment_placements.go:210-215` | oui | **oui** pour instance ↔ socle |
| E9 | **Bot** (paquet type 12, BOT_METADATA) | `bot{Slot, BotID, Name}` — `games/halo_infinite/film/killsource/botmeta.go:32-37`, lecture `:101` ; exposé `killsource/roster.go:39-44` (`BotEntry`), rempli `:128` | déclaré une fois ; **deux bots peuvent partager un siège** (remplaçants successifs) — `replay/identity.go:165-170` | **BotID ↔ slot de roster ↔ nom, LU** ; `BotID` **est** le N de `bid(N.0)` de la base (attesté `botmeta.go:22-25` : « `343 Aloysius`/bid(39.0) et `343 PardonMy`/bid(7.0) ») | **le `BotID` n'est PAS publié dans l'artefact** : `RosterEntry` ne porte que `Name` et `Bot` (`replay/document.go:1363-1381`) → **la jointure web se fait sur le NOM NU** — `apps/web/src/lib/replay/rosterLogic.ts:100`, `:110`, `:122-128` | oui (côté Go) | non — c'est un défaut de **publication**, pas de décodage |

### 1.1 Ce que les colonnes de faisabilité veulent dire

- **« Dispo décodeur actuel »** = P2 à P5 peuvent s'en servir SANS toucher `filmdec` / `himap`
  (doctrine §0.6 du plan v2 : le décodeur ne bouge pas avant v7.5.0).
- **« Exige travail décodeur »** = un lien direct EXISTE peut-être, mais il faudrait lire un flux
  non décodé ou non branché. Ces cas sont **classés, pas planifiés** : ils appartiennent au plan
  décodeur d'après v7.5.0, un item par flux, additif.

Récapitulatif des cases « exige décodeur » : **slot de bipède ↔ index de joueur** (E2, le trou
central), **slot statborg ↔ joueur** (E3), **équipe** (E4), **attachement porteur ↔ objet** (E5),
**identité de zone** (E6), **véhicule de l'embarquement** (E7, partiel), **instance ↔ socle** (E8).

### 1.2 Décompte des liens, par entité

Un **lien** = une relation entre deux clés (pas un attribut). C'est ce décompte que la provenance
de l'axe (g) doit rendre mesurable, et que le gate corpus surveillera.

| Entité | Liens DIRECTS (lus dans le film) | Liens par REPLI (déduits, catalogue, ou base) |
|---|---|---|
| E1 index de joueur | **3** — index ↔ xuid ; index ↔ auteur d'un tir ; xuid ↔ gamertag | **0** |
| E2 slot de bipède | **0** — aucune identité dans `BipedPosition` | **5** — pont par morts ; fermetures A/B ; siège de bot ; relais (base) ; nommage final par occupation |
| E3 slot statborg joueur | **2** — slot ↔ manche déclarée ; slot ↔ instant (horloge manifeste) | **2** — slot ↔ xuid par triplet K/D/A (feuille) ; slot ↔ xuid par instants de mort (par manche) |
| E4 slot statborg équipe | **1** — slot ↔ ses compteurs | **2** — slot ↔ camp par score final ; par somme des frags |
| E5 objet d'objectif | **3** — objet ↔ type de catalogue (mot MPP) ; objet ↔ sa vie `(slot, gen)` ; slot bipède ↔ « porte quelque chose » (marqueur d'image-clé, **CTF seulement**) | **3** — portage ↔ drapeau (géométrie + invariant) ; objet ↔ équipe propriétaire (catalogue de carte) ; porteur ↔ portage (via E3) |
| E6 zone | **2** — valeur de canal ↔ index d'équipe ; lecture ↔ instant | **2** — slot `ti=13` ↔ zone (vote modal) ; action ↔ zone (géométrie 5 m) |
| E7 véhicule | **3** — occupant ↔ sortie ; occupant ↔ embarquement ; véhicule ↔ sortie (+ siège) | **2** — véhicule d'un épisode sans sortie (géométrie 3 m) ; occupant ↔ xuid (via E2) |
| E8 arme au sol / équipement | **2** — objet ↔ famille (mot MPP) ; ramasseur ↔ objet de catalogue (`biped_pickup`) | **3** — socle du match ↔ socle du catalogue (1 m) ; ramassage ↔ origine socle/sol ; pose ↔ poseur (`deployed`/`dropped`) |
| E9 bot | **1** — BotID ↔ slot de roster ↔ nom | **1** — bot ↔ ligne de feuille, par **nom nu**, côté web |
| **transverses** | **1** — chunk ↔ `start_ms` du manifeste (le seul zéro commun) | **3** — origine de la frise ; T0 (premier mouvement) ; bornes de manche (consensus) ; **plus** l'équipe d'un joueur et les arrivées/départs, qui ne viennent **pas du film** mais de la base |
| **TOTAL** | **18** | **23** (+ 2 liens de source externe : équipe, arrivées) |

Lecture de ce tableau : **le trou est E2**. Cinq replis empilés y répondent à une seule question —
« quel joueur occupe ce slot de bipède, à cet instant » — et c'est de ce trou que dépendent E5, E7
et E8 pour nommer leur porteur, leur occupant et leur ramasseur. C'est pour cela que **P2 vient
avant P3, P4 et P5**, et pas l'inverse.

---

## 2. Les dix axes

### (a) TEMPS — quelle horloge DIRECTE le film porte

**État actuel.** La frise n'a pas d'horloge : elle est ancrée sur le **premier paquet de position**
(`origin := sorted[0].TimestampUS`, `replay/build.go:49`) et sa longueur est le **dernier paquet de
position** (`frameSpan`, `build.go:530-533`). L'origine publiée est une DIFFÉRENCE de deux lectures
d'en-têtes — premier paquet de position moins premier paquet du chunk 1 — contrôlée par le calage
du fil des morts et **refusée** si les deux divergent de plus d'une seconde
(`replay/origin.go:31-37`, `:108-124`, tolérance `:68`). Sans origine, `originResolved` vaut faux et
les calques datés sur l'horloge du film ne sont pas recalés (`origin.go:126-147`).

**Ce que le film porte de DIRECT :**

1. **L'horodatage moteur de chaque paquet** (`filmsource.Packet.TS`, µs) —
   `filmsource/film.go:53-58`. C'est une horloge monotone du moteur, PAS un temps depuis le début
   du film : 4 521 / 3 896 / 2 259 / 8 583 s sur quatre témoins (`origin.go:25-28`).
2. **Le `start_ms` de chaque chunk, porté par le MANIFESTE du film** (`filmsource.ChunkMeta.StartMS`,
   `film.go:38-43`, alimenté `:171-190`). C'est le seul zéro commun, et il est déjà employé :
   `objectiveevents/statborg.go:199` date chaque enregistrement de statborg par
   `c.meta.StartMS + (f.TS - base)/1000`. Un chunk hors manifeste est écarté plutôt que daté à zéro
   (`objectiveevents/film.go:79-97`).
3. **Le coup d'envoi (T0)**, dérivé du premier mouvement des pistes (`replay/t0_film.go:9-42`) —
   c'est une mesure, pas une lecture, mais elle est stable (écart-type 299 ms, CV 0,013 sur
   83 matchs, `t0_film.go:32-34`).

**Ce que le film NE porte PAS.** L'horloge officielle de manche —
`game-engine-round-timer-component` (ti=0 i5), décodée en `RoundTimer` par
`filmdec/vitality.go:175-185` via la couche de capture `filmdec/capture.go:39-40` — **n'a aucun
porteur** : l'entité moteur de partie (ti=0) n'est pas répliquée. Mesure :
**1 record de ti=0 sur 22 films et 1 269 000 records certains**
(`.ai/V7.5/replay2d/registre_film/LOTBP_PHASE0.md` §0 et §B.0.1), corroborée par un oracle extérieur
(captures Cheat Engine : ti=0 absent des deux ventilations). Verdict écrit du lot :
**B.0.2 « NON MESURABLE, 0 lecture sur 22 films »**. `RoundTimerOf` n'a d'ailleurs aucun appelant de
production (grep : `filmdec/capture.go:65` et `game_entities_hunt_test.go` seulement).

**VERDICT (a).** *Le film ne porte aucune horloge de partie directe.* La seule horloge directe
disponible est le couple **(horodatage moteur du paquet, `start_ms` du manifeste)**, déjà exact et
déjà utilisé par le statborg. **Recommandation pour P2** : la grille de frames doit se déclarer sur
cette horloge-là — origine et durée depuis le manifeste, non depuis le premier et le dernier paquet
de position — au lieu d'être bornée par ce que les bipèdes ont émis ; l'origine publiée devient une
lecture et non une différence contrôlée, et `resolveOriginMs` garde son rôle de **contrôle**.
Le calage du fil des morts (`DeathOffsetMS`) reste nécessaire tant que E3 et le fil des morts vivent
sur l'horloge du match. **Aucun changement de `filmdec` n'est requis** : `ChunkMeta.StartMS` est déjà
chargé et déjà lu ailleurs.

### (b) MANCHES — quel flux d'état de mode donne les frontières

**État actuel.** Les bornes sont un **consensus statistique** sur les trains de score :
`objectiveevents.ResolveRoundBounds` (`round_bounds.go:139-162`) prend la **médiane sur les slots**
du premier instant où chaque slot déclare la manche, exige qu'une **majorité de slots** parle,
vérifie que la borne **sépare vraiment** les deux populations, et garde une **exemption par slot**
pour ne jamais jeter un bloc entier (`round_bounds.go:26-72`, `KeptSegment` `:112-124`). Trois films
du parc sortent intacts parce que leur étiquetage de manche est faux (`round_bounds.go:74-78`).

**Le semi-direct qui existe déjà.** Chaque `StatRecord` **déclare sa manche**, lue dans deux en-têtes
de 5 bits (`objectiveevents/statborg.go:159-162`). Ce n'est pas un consensus : c'est un champ du film.
Le consensus n'existe que pour **arbitrer les faux positifs d'en-tête** — la mesure fondatrice est
citée sur pièces (`round_bounds.go:16-23` : `51ebbc0f`, un enregistrement à 316 777 ms déclarant la
manche 0 et coûtant 58 assistances au document).

**Le flux d'état de mode.** `game-engine-current-round-component` (ti=0 i4, `R(1)` porte inversée puis
`R(5)`) et `game-engine-current-state-component` (ti=0 i2, `R(3)`) sont décodés et publiés par hook
(`filmdec/components_game_engine.go:51-55`, `:102-113`), **sans consommateur de production** (grep
`SetGameEngineHook` : `components_game_engine.go:91` + deux `_test.go`). Ils n'ont **aucun flux** :
`LOTBP_PHASE0.md` §B.0.5 — « i2 (état), i4 (manche), i8 (conditions de fin), i7 (grâce) sont des
composants de ti=0 : **0 lecture confirmée sur 22 films** », et les 47 « lectures d'i2 » vues avant
le durcissement étaient des artefacts de liaison désalignée.

**VERDICT (b).** *Aucun flux d'état de mode ne donne les frontières de manche.* La seule source
directe disponible est le **numéro de manche déclaré par chaque enregistrement du statborg**, et le
consensus reste indispensable comme **filtre de faux positifs**, non comme source. **Pour P3** : le
registre doit exposer les bornes de manche comme une donnée UNIQUE (`RoundBounds` reste la source,
`objectiveevents.RoundStartsMS` en est déjà le point d'entrée partagé — défini
`round_bounds.go:394`, consommé `slotidentity_rounds.go:155`), et la provenance de la borne doit dire `deduit` (consensus), jamais
`direct`. La question « un objet est-il replacé à la frontière de manche » (§0.4 du plan v2) reste
une **mesure à faire en P3**, elle n'est tranchée par aucune lecture.

### (c) ÉQUIPES — l'équipe par index de joueur telle que le film la porte

**État actuel.** `Track.Team` vaut **-1 en permanence** : « L'ÉQUIPE N'EST PAS DANS LE FILM. Elle vit
dans la base, avec le gamertag, et le client la joint par XUID »
(`replay/document.go:1474-1479`). Les tables d'équipe des calques arrivent de l'appelant :
`FlagInput.TeamOf` (`replaybuild/matchfacts.go:223`) et `ZoneInput.TeamByXUID`
(`replaybuild/replaybuild.go:262`), tous deux construits depuis `port.MatchFacts` — c'est-à-dire la
base. Le calque du drapeau le dit explicitement : « L'EQUIPE DU PORTEUR NE VIENT PAS DU FILM »
(`replay/flag_assign.go:26-29`), table vide = l'invariant se tait.

**Ce que le film porte.** Deux choses seulement, et aucune n'est une équipe par joueur :

1. `game-engine-team-mapping-component` (ti=0 i0) est **traversé mais ses valeurs sont jetées** —
   `filmdec/components_team_mapping.go:33-45` (aucun hook, aucun retour). Et il appartient à ti=0,
   donc **0 record sur 22 films** (§(b)).
2. Les **entités d'équipe du statborg** (slots 6 et 8), qui portent le score et la somme des frags —
   mais **rien ne dit laquelle est `team_0`** : le camp est déduit par le score final puis par la
   somme des frags, avec refus explicite en cas d'égalité
   (`replay/score_team_identity.go:5-27`, `:34-45`).
3. Pour les zones, la **valeur du tag 4 EST l'index d'équipe**, mesurée à 100 % et 91,1 % hors
   émissions neutres (`replay/zone_states.go:13-15`) — c'est un lien direct **valeur → camp**, pas
   **joueur → camp**.

**VERDICT (c).** *Le film ne porte l'équipe d'aucun joueur.* Le lien joueur ↔ camp reste
**exclusivement la base**, et la feuille de match ne peut donc pas être une « vérification » : elle
est la source unique. **Pour P2** : le registre doit porter le camp comme un lien de provenance
`externe` (voir (g)) — ni `direct`, ni `deduit`, parce qu'il ne vient pas du film ; et le lien
**valeur de canal ↔ camp** des zones (E6) reste `direct` et se publie comme tel. Les joueurs entrés
en cours et les changements d'équipe ne sont donc **pas observables dans le film** — même verdict
que (d).

### (d) ROSTER — voir §0

Verdict rendu en tête. Compléments de pièces :

- Le roster du rejeu est aujourd'hui construit de **deux sources** : les humains du fil des morts
  (via `PlayerIndexTable`) et les bots déclarés (`replay/identity.go:171-198`). Un joueur qui ne
  meurt jamais n'apparaît dans aucune des deux — d'où `Options.RosterXUIDs`, un complément fourni
  par l'appelant depuis la base (`replay/player_index.go:114-142`, mesure citée `:120` :
  `3372e7eb`, 6 joueurs publiés sur 8).
- Le seul « départ » que le pipeline connaisse est celui que la base date
  (`successions.go:5-27`) ; le film ne date que le **silence d'un slot**, qui est ambigu (véhicule,
  mort, départ — `replay/lives.go:90-97`).

### (e) BOTS — l'identifiant stable `bid(N.0)`

**Il est LU, il n'est pas PUBLIÉ.** Le paquet type 12 déclare `{Slot, BotID, Name}`
(`killsource/botmeta.go:32-37`, `:101`) et `BotID` est le N de `bid(N.0)` — attesté par le critère
pré-enregistré du lot (`botmeta.go:22-25`). `replaybuild` s'en sert déjà comme **clé exacte** pour
les relais : `byID[b.BotID]` puis `fmt.Sscanf(p.XUID, "bid(%d.0)", &id)`
(`replaybuild/replaybuild.go:500-508`).

Mais `botIdentities` ne transporte que `{FilmIndex, Name + " [bot]"}`
(`replaybuild/replaybuild.go:477-483`), et `RosterEntry` n'a pas de champ d'identifiant de bot
(`replay/document.go:1363-1381`). **Le web joint donc par le NOM NU** :
`apps/web/src/lib/replay/rosterLogic.ts:100` (`botKey(entry.name)`), `:110`, et surtout `:122-128` —
« LA JOINTURE D'UN BOT SE FAIT PAR GAMERTAG, faute de xuid commun […] la clé de jointure est le nom
NU des deux côtés ». Le commentaire le dit lui-même : deux bots homonymes fusionnent
(`rosterLogic.ts:97-98`).

**VERDICT (e).** *Le lien direct existe et n'est pas publié.* C'est le **gain le moins cher de tout
le lot P** : ni décodage neuf, ni mesure. **Pour P2** : `RosterEntry` et `Track` portent le `bid`
(forme `bid(N.0)`, la même que la base) ; le registre publie le lien `bid ↔ index de roster ↔ nom`
avec provenance `direct` ; `rosterLogic.ts` joint sur `bid` et garde le nom en repli **compté**.
Garde-rail : un test qui interdit une jointure de bot par égalité de nom hors du repli déclaré.

### (f) CATALOGUES DE CARTE — référencés par identifiant d'asset, pas par géométrie

**Ce qui est déjà par identifiant d'asset :**

- Les catalogues sont indexés par **`map_id` (asset UGC)** et la jointure se fait par `map_id` SEUL :
  `replay/objectives_catalog.go:44-47` (`Maps map[string]MapObjectivesEntry`, « clé = map_id »), et
  la raison est mesurée — `public_name` est vide sur la quasi-totalité des entrées et le module ne
  porte pas le même nom des deux côtés (`replaybuild/flagspawns.go:12-18`).
- Le catalogue des socles suit le même patron (`replay/map_weapon_pads_catalog.go:23-24`).
- Une zone porte son `TypeID` brut du fichier de variante (`replay/mapvar/objectives.go:217`) et son
  `TeamIndex` (`:220`), et la colline de KOTH est reconnue **par hash de label, jamais par nom**
  (`mapvar/objectives.go:144-148`, `:180`).

**Ce qui reste géométrique :**

- **socle du match ↔ emplacement du catalogue** : rayon de 1 m (`replay/map_weapon_pads.go:37-40`,
  `confirmePar` `:99-100`). Ce n'est pas une paresse : le catalogue **pose** les socles, le mode les
  **allume**, et rien dans le `.mvar` n'explique l'extinction (`mapvar/socles.go:14-20` :
  Cliffhanger, 17 socles au fichier, 10 en CTF, 0 en Super Fiesta).
- **ramassage ↔ point d'apparition** : rayon `PickupOriginMatchM` (`replay/pickup_origin.go:53-64`).
- **objet d'objectif ↔ socle propriétaire** : le socle le plus proche (`replay/flag_assign.go:31-38`).
- **zone du film ↔ zone du catalogue** : vote modal + 5 m (§(a) de E6).

**VERDICT (f).** *Le lien carte ↔ catalogue est déjà par identifiant d'asset et n'a rien à changer.
Le lien objet-du-match ↔ entrée-de-catalogue est GÉOMÉTRIQUE partout, et il le restera* : rien dans le
film ne porte l'identifiant de l'instance de socle ou de zone du fichier de carte
(`biped_pickups.go:42-46` le dit pour les ramassages ; `mapvar/socles.go:14-16` pour les socles).
**Pour P3/P4** : le registre porte l'entrée de catalogue **par son identifiant** (`map_id` +
`type_id` + rang spatial, jamais un nom) et **publie la distance** qui a servi à la confirmer, avec
provenance `catalogue`. C'est ce qui permettra au gate de refuser une régression `catalogue → deduit`.

### (g) PROVENANCE — la forme exacte du champ

**L'art existant, à ne pas ré-inventer :**

| Prior art | Forme | Publié ? |
|---|---|---|
| `NomParMort` / `NomParFermeture` — comment on sait à qui une vie appartient | deux constantes chaîne | oui, via `LifeInsert.NamedBy` (`replay/lives_export.go:30-37`, `lives.go:104-115`) |
| `BridgeHealth` — sur quoi repose le pont, par voie et par compte | 11 compteurs JSON | oui (`replay/coverage_bridge.go:24-50`) |
| `vehicleResolvedBy` — event / event-nearest / geometry | énumération `int` | **non**, interne (`replay/vehicle_rides_events.go:280-301`) |
| `EquipmentPlacement.Origin` — deployed / dropped / unknown | constantes chaîne | oui (`replay/equipment_placements.go:210-215`) |
| `Pickup.Origin` — spawner / ground / absent | constantes chaîne | oui (`replay/document_pickups.go:149`, `pickup_origin.go:60-63`) |
| `ScoreIdentityFinal` / `Frags` / `Unresolved` — quelle preuve a nommé le camp | constantes chaîne (`"a"`, `"b"`, `"unresolved"`) | oui (`replay/document_score.go:65`, `:69`, `:71` ; posées par `replay/score_team_identity.go:34-45`) |

**Forme proposée pour le registre** — une seule structure, réutilisée par TOUS les liens :

```go
// LinkSource dit D'OU vient un lien du registre. Valeurs FERMEES ; toute autre valeur est un
// defaut de producteur, pas une categorie.
type LinkSource string

const (
    LinkDirect    LinkSource = "direct"     // lu dans le film : un champ, un identifiant, une reference
    LinkCatalog   LinkSource = "catalogue"  // fige par le catalogue du titre ou de la carte (map_id, type_id, TOML)
    LinkExternal  LinkSource = "externe"    // vient de la base / de la feuille : equipe, arrivee, bid
    LinkInferred  LinkSource = "deduit"     // pont par morts, consensus, vote, geometrie, elimination
    LinkUnresolved LinkSource = "non_resolu" // rien n'a nomme ce lien — PUBLIE, jamais jete
)

// Link est UN lien du registre : de quoi il part, vers quoi il va, d'ou il vient, et ce qui
// permet de le juger.
type Link struct {
    Source LinkSource `json:"source"`
    // Method nomme la voie EXACTE a l'interieur de la source. Il n'y a pas deux facons de
    // deduire qui se valent : `deathBridge`, `closure`, `elimination`, `succession`,
    // `previousLife`, `nextLife`, `slotBridge`, `nearestPad`, `nearestVehicle`, `modalVote`,
    // `roundConsensus`, `finalScore`, `fragSum`. Vide pour `direct` et `catalogue`.
    Method string `json:"method,omitempty"`
    // Confidence porte le DENOMINATEUR de la voie quand elle en a un — lectures concordantes
    // pour `direct`, distance en metres pour la geometrie, votes pour un modal, morts
    // appariees pour le pont. Nul quand la voie n'en produit pas.
    Readings int      `json:"readings,omitempty"`
    Metric   *float64 `json:"metric,omitempty"`
    // From / To bornent la validite du lien sur l'axe du rejeu (index de frame). Un lien
    // valable tout le match porte [0, frameCount-1]. C'EST CE CHAMP QUI INTERDIT DE REJOUER
    // LE DEFAUT P0-2 : un slot recycle porte DEUX liens bornes, jamais un lien aplati.
    From, To int `json:"from"`
}
```

Publication : une section `identity` de l'artefact, avec **par entité** la liste de ses liens, et un
récapitulatif `identity.coverage` = `{direct, catalogue, externe, deduit, non_resolu}` par TYPE de
lien. C'est ce récapitulatif que le **gate corpus** compare : `make replay-corpus-gate`
(`Makefile:231-232`) échoue si un compte `direct` **baisse** ou si un compte `deduit` **monte** à
`direct` constant, exactement comme il échoue aujourd'hui sur une durée qui baisse.

**VERDICT (g).** *La provenance existe déjà en cinq exemplaires incompatibles ; P2 la centralise.*
La règle « ≤ 2 copies » impose de migrer `vehicleResolvedBy`, `NomPar*`, `Origin` et
`ScoreIdentity*` vers `LinkSource`/`Link` **et** de poser le garde-rail (test grep interdisant une
seconde énumération de provenance hors du registre) — sans quoi la dette re-croît (anti-pattern 8 du
CLAUDE.md, leçon du prédicat bot passé de 8 à 36 copies).

### (h) MULTI-TITRE — types d'entités canoniques

**État actuel.** `internal/games/canonical/` ne porte **aucun type d'entité de film** : ni slot de
bipède, ni objet d'objectif, ni véhicule. Le seul type d'identité est `PlayerIdentity`
(`canonical/identity.go:6-16`), qui porte déjà `XUID` en **chaîne** (« pour préserver les
bit-patterns Halo qui débordent en int64 signé », `identity.go:3-5`) et `IsBot`. Les seuls types
proches sont `HighlightEventType` (`canonical/enums.go:113-128`) et `KillKind`
(`canonical/events.go:69+`). Le décodeur, lui, est déjà rangé par titre
(`internal/games/halo_infinite/film/killsource/`).

**Le portail est la capability, et il est déjà fin.** `film.replay_artifact` gouverne la PRODUCTION
de l'artefact et l'affichage suit (`config/titles/halo_infinite/mappings/capabilities.toml:150-164`) ;
Halo 5 la déclare `not_exposed` avec la raison écrite — « autre format de film, aucun décodeur, donc
aucun artefact possible » (`config/titles/halo_5/mappings/capabilities.toml:58`, `:65`).

**VERDICT (h).** *Le registre doit naître canonique, parce qu'aucun type ne préexiste et qu'il n'y a
donc rien à casser.* **Pour P2** : les types du registre (`EntityKind`, `EntityRef`, `Link`,
`LinkSource`) vivent dans `internal/games/canonical/` — ce sont des types de DONNÉE inter-titres, pas
des adapters ; `internal/analysis/replay` les remplit pour Halo Infinite ; un autre titre les
remplirait par son propre adapter sans toucher au consommateur. `EntityKind` est une liste FERMÉE
(`player_index`, `biped_slot`, `statborg_slot`, `team_slot`, `objective_object`, `zone`, `vehicle`,
`world_item`, `bot`) — la même discipline que `AllOutcomes()` / `IsKnownOutcome`
(`canonical/enums.go:17-29`). **Aucun `slug ==` nulle part** : la porte reste la capability
(ratchet `no_slug_comparison_test.go`).

### (i) LECTEURS HORS REJEU À MIGRER — et ce qu'un bump impose

**Les trois pièces, sur pièces :**

1. **`internal/sync/killcollector/positions.go`** — la passe de positions du sync appelle
   `replay.ScanPlayerIndices` (`positions.go:258`) puis `replay.ResolveSlotXUID`
   (`positions.go:266`), c'est-à-dire **le pont par morts du paquet `replay`**, pas un pont maison.
   C'est la raison même de l'export (`replay/killpos_bridge.go:41-53`, et
   `positions.go:43` : « c'est pourquoi `replay.ResolveSlotXUID` a été exportée »).
   Les désaccords d'index sont comptés (`positions.go:263-265`,
   `metricPositionsAmbiguous`) et un pont vide fait échouer la passe (`positions.go:267-272`).
2. **`internal/sync/killcollector/isolation_facts.go`** — le matériau remonte le `OwnerReport`
   entier (`materiauDIsolement`, `isolation_facts.go:118-129`) ; la projection consomme
   `replay.ContextesDesMorts`, qui attribue les positions **par la vie qui couvre l'instant** via
   `occupantA` (`replay/death_context.go:278`, définition `:306-315`) — c'est la correction P0-2, et
   c'est exactement le comportement que le registre doit reprendre.
   La garde de publication est `IndexDisagreements == 0 && len(SlotXUID) > 0`
   (`replay/death_context.go:133-141`).
3. **Les deux tables** `match_lives` et `match_death_context`
   (`internal/migration/steps_shared_match_lives.go:82-104` et `:143-168`) sont **append-only avec
   vue `_latest` par passe** (`:110-116`, `:171-177`) et portent `decode_pass` + `decoder_rev`.
   La révision est `IsolationDecoderRev = "isolement-2026-09-07"`
   (`killcollector/isolation_facts.go:116`), **distincte** de `KillSourceDecoderRev`
   (`killcollector/collector.go:72`) pour que les deux passes évoluent séparément
   (`isolation_facts.go:111-115`).

**Réponse explicite à la question du plan d'orchestration.** **OUI, un bump d'`IsolationDecoderRev`
à P2 impose une réécriture par `backfill-killsource`, et voici laquelle.** La sélection est
`matchsAJour` (`apps/go-api/cmd/levelup/cmd_backfill_killsource_selection.go:89-101`) : un match est
« à jour » si son journal porte `KillSourceDecoderRev` **ET** que l'une des trois conditions tient —
aucune position, `match_lives_latest` **à la révision d'isolement courante**, ou aucune équipe en
base. Changer `IsolationDecoderRev` invalide la deuxième condition pour **tous les matchs qui ont
des positions ET des équipes** : ils sortent de `matchsAJour`, sont re-décodés, et écrivent une
**nouvelle passe** dans les deux tables ; les vues `_latest` basculent d'un bloc
(`QUALIFY … FIRST_VALUE(decode_pass) OVER (PARTITION BY match_id ORDER BY written_at DESC, id DESC)`).
Ce n'est pas une réécriture en place : c'est un ajout append-only, conforme à l'ADR 0026.
**Le journal des morts, lui, n'est PAS re-écrit** tant que `KillSourceDecoderRev` ne bouge pas — c'est
tout l'objet des deux révisions séparées. La commande est donc `levelup backfill-killsource`, et le
critère de convergence est écrit dans le même fichier
(`cmd_backfill_killsource_selection.go:86-88` : sans la troisième condition, un match à positions mais
sans équipe serait re-décodé indéfiniment).

**Liste complète des lecteurs du pont, à migrer en P2** (grep `SlotXUID|NamingBridge()|xuidAt(|own.Owner`
sur `apps/go-api/internal`, hors `_test.go`, HEAD `22d76a738`) :

| Fichier:ligne | Ce qu'il lit |
|---|---|
| `replay/build.go:70`, `:92`, `:105` | `own.Owner` — nommage des bots, tirs, lancers |
| `replay/build.go:80` | `own.SlotXUID` + `own.SlotAmbiguous` — nommage final |
| `replay/build.go:130` | `own.SlotXUID` — frags sous équipement actif |
| `replay/build.go:197` | `own.SlotXUID` — ramassages (`pickupInputs.slotXUID`) |
| `replay/build.go:294` | `own.NamingBridge()` — morts neutres |
| `replay/build_objectives_live.go:195` | `own.NamingBridge()` + `own.SlotAmbiguous` — drapeau |
| `replay/build_zones.go:70` | `own.NamingBridge()` — zones |
| `replay/objectives.go:160` | `own.NamingBridge()` — actions d'objectif |
| `replay/bomb_carries.go:137`, `:142` | `own.SlotXUID` — portage de bombe |
| `replay/inventory_dead_readings.go:33`, `:42` | `own.SlotXUID` — lectures d'inventaire post-mortem |
| `replay/vehicle_rides.go:287`, `vehicle_rides_events.go:275` | `own.xuidAt(slot, instant)` — occupant |
| `replay/vehicle_shots.go:73` | `own.Owner` — tirs de véhicule |
| `replay/death_context.go:296`, `:338` | `r.lives` / `r.SlotXUID` — contexte de mort |
| `replay/coverage.go:327` | `own.Owner` / `own.FromDeaths` — santé publiée |
| `replay/killpos_bridge.go:51-53` | la porte exportée elle-même |
| **`sync/killcollector/positions.go:266`** | **hors rejeu** |
| **`sync/killcollector/isolation_facts.go:118-129`** | **hors rejeu** |

### (j) EMPLACEMENT DU REGISTRE — amendement exact du §0.7 (décision D11, TRANCHÉE)

Le §0.7 du plan v2 dit aujourd'hui :

> « UNE table d'identité par film, **calculée une fois dans `replaybuild`**, publiée dans l'artefact
> (section `identity` : index ↔ xuid, slot de statborg ↔ index, slot de bipède ↔ index dans le temps,
> avec la source et la couverture de chaque lien) ; tous les calques la consomment ; aucun calque ne
> reconstruit son propre pont (garde-rail). »

**Phrase de remplacement (amendement à porter dans `.ai/PLAN_V2_RESTES_2026-09-07.md` §0.7 au premier
commit de P2) :**

> « UNE table d'identité par film, **calculée par une FONCTION PURE de
> `internal/analysis/replay`**, appelée **par `replaybuild` ET par le collecteur de sync** —
> parce que les données d'un match sont complètes au sync, seul le rejeu attend la cuisson —,
> publiée dans l'artefact (section `identity` : index ↔ xuid, slot de statborg ↔ index, slot de
> bipède ↔ index dans le temps, avec la source et la couverture de chaque lien) ; tous les calques
> et tous les lecteurs hors rejeu la consomment ; aucun calque ni aucun collecteur ne reconstruit
> son propre pont (garde-rail : allowlist datée des seuls producteurs). »

**Contrainte que cela impose à la signature — écrite pour l'exécuteur de P2 :**

1. **Aucune dépendance à l'artefact cuit.** L'entrée est ce que le décodage rend, jamais un
   `ReplayDocument` ni un fichier d'artefact : positions, fil des morts, table d'index, records de
   statborg, événements, roster/bots/relais/équipes fournis par l'appelant. C'est déjà le contrat
   de `BuildFromPositions` (« PUR (aucune I/O) : c'est le cœur testable de l'assemblage »,
   `replay/build.go:24-25`) et de `ResolveSlotXUID` (`replay/killpos_bridge.go:41-53`).
2. **Aucune I/O, aucune base, aucun `filmsource.Film`** dans la fonction pure. Le collecteur charge
   déjà le film lui-même et détient `LockProcessDecode` (`killcollector/positions.go:235-236`) ; une
   variante `Scan*` de commodité peut exister À CÔTÉ, comme `ScanFilmPlayerIndices` existe à côté de
   `ScanPlayerIndices` (`replay/player_index.go:55-61`), mais elle est marquée hors production.
3. **≤ 5 paramètres** (seuil du dépôt) : une structure d'entrée unique, sur le modèle de
   `EntreeContexteMorts` (`replay/death_context.go`) et de `vehicleRideInputs`.
4. **Sortie sérialisable telle quelle** : la section `identity` de l'artefact EST le type rendu, pas
   une projection — sinon deux écritures divergeront (leçon `flag_assign` vs `assembleFlagLives`,
   plan v2 R3).
5. **Le collecteur n'écrit pas l'artefact** : il consomme le registre pour `match_lives` /
   `match_death_context` et pour les positions ; il ne publie pas `identity`. La table est la même,
   les deux sorties diffèrent.
6. Cette fonction devient le **seul** producteur autorisé : `buildOwners` et `ResolveSlotXUID`
   deviennent internes au registre, et l'allowlist du garde-rail ne contient que le fichier du
   registre.

---

## 3. Séquençage proposé pour P2 à P5 (une page)

**P2 — Registre des joueurs** (= R1 + R2 du plan v2). Producteur unique : nouveau fichier pur de
`internal/analysis/replay` ; types dans `internal/games/canonical`. Publie `identity` + provenance.
Lecteurs à migrer, fichier par fichier : `build.go:70,80,92,105,130,197,294` ·
`build_objectives_live.go:195` · `build_zones.go:70` · `objectives.go:160` · `bomb_carries.go:137` ·
`inventory_dead_readings.go:33` · `vehicle_rides.go:287` · `vehicle_rides_events.go:275` ·
`vehicle_shots.go:73` · `death_context.go:296` · `coverage.go:327` · `unnamed_lives.go:86` ·
`identity.go:207` · `successions.go:61` · `zone_attribution.go` · `lives_export.go` ·
**hors rejeu** `sync/killcollector/positions.go:266` et `isolation_facts.go:118`.
Traite au passage R1 (actions jamais jetées, identité par élimination sur le roster) et R2 (vies
qu'aucune mort ne termine). **Garde-rail** : test qui interdit tout appel à `buildOwners` /
`ResolveSlotXUID` et toute lecture de `OwnerReport.SlotXUID` / `.Owner` hors du fichier du registre
(allowlist datée, une entrée). **Bump** : 49 (unique pour tout P). **Réécriture** : bump
`IsolationDecoderRev` + `levelup backfill-killsource` (§(i)).

**P3 — Registre des objets d'objectif** (= R3). Une seule machine à états d'objet, partagée par
drapeau / crâne / bombe / zone. Lecteurs : `flag_assign.go:242` · `flag_carries.go:165` ·
`flag_carries_lives.go:84` · `flag_carries_marker.go:30` · `flag_objects.go:116` · `flag_neutral.go` ·
`skull_carries.go:102` · `vip_crown.go:153` · `bomb_carries.go` · `held_object_carry.go` ·
`build_objective_objects.go` · `zone_states_owner.go:191` · `zone_states_hill.go:294` ·
`hill_hold_ticks.go` · bornes `objectiveevents/round_bounds.go:139`.
**Garde-rail** : (1) test grep interdisant une seconde machine à états d'objet (aucun autre fichier
que le registre ne déclare de transition `enJeu`/`sol`/`porte`) ; (2) test qui échoue si un calque
retient « le socle le plus proche » alors que le registre nomme l'objet.

**P4 — Registre des véhicules et assets.** Lecteurs : `vehicle_rides.go:189-230` ·
`vehicle_rides_events.go:243` · `vehicle_tracks.go` · `vehicle_relays.go` · `vehicle_shots.go:73` ·
`ground_weapon_objects.go` · `ground_weapon_pads.go` · `ground_weapon_rules.go:388` ·
`powerup_pads.go` · `equipment_placements.go` · `equipment_episodes.go:329` ·
`equipment_episode_kills.go` · `pickup_origin.go` · `document_pickups.go:183` ·
`pad_pickup_dating.go` · `map_weapon_pads.go:99`.
**Garde-rail** : test qui interdit le rattachement géométrique (`vehicleNearestTo`, `confirmePar`,
rayon de socle) quand le registre porte un lien `direct` pour la même entité au même instant —
c'est-à-dire que la géométrie ne peut être qu'un `Method` de la source `deduit`, jamais un
remplaçant silencieux d'un `direct`.

**P5 — Clôture.** `make replay-corpus-gate` à 0 perte ; balayage informatif du parc ; oracle feuille
de match sur tous les témoins ; chronique 49 ; garde-rail global « aucun calque ne construit de lien
d'identité hors du registre » (allowlist datée) ; doc `docs/` FR **et** EN de la section `identity` ;
**une seule** revue adversariale sur le diff cumulé P2→P5 (règle §0.5 du plan d'orchestration §4).

---

## 4. Questions ouvertes — avec le témoin qui les tranche

Aucune de ces questions n'est devinée ici. Chacune porte la commande exacte qui la ferme.

| # | Question | Ce qui la tranche |
|---|---|---|
| Q1 | Le débit de ti=5 i18/i19 mesuré en août (163 et 105 lectures sur 22 films) tient-il au HEAD, après le durcissement du décodeur du 2026-09-05 ? Un `bid` de bot arrivant en cours pourrait-il être daté par i19 ? | L'instrument existe et n'a pas de tag `gamefiles` : `CGO_ENABLED=0 GAME_FILM=<...>/data/cache/film_chunks/000d5950 GAME_OUT=<scratchpad> go test ./internal/analysis/filmdec/ -run '^TestGameEntitiesPhase0$' -timeout 30m -v` (usage écrit `filmdec/game_state_measure_test.go:26-29`) ; colonnes `active`/`joining` du TSV `<short8>_ti5.tsv` (`filmdec/game_state_dump_test.go:31-33`). **UN film par processus** (D17). |
| Q2 | ti=0 est-il toujours absent au HEAD (0 record / 22 films) ? Si un build récent le répliquait, (a) et (b) changeraient de verdict. | Même commande que Q1 ; la sortie `<short8>_ti0.tsv` doit rester vide. Corroboration : `.ai/V7.5/replay2d/registre_film/lotBP/<short8>.GAME_FILM.log`. |
| Q3 | La grille de frames est-elle réellement plus courte que ce que les enregistrements couvrent, et de combien ? Combien d'actions d'objectif tombent en `OutOfWindow` de ce fait ? | Le compteur existe déjà : `LayerCoverage.OutOfWindow` peuplé en `replay/objectives.go:98-101`. Relever `coverage.objectives.outOfWindow` sur les 7 témoins de `config/replay_corpus.toml` : `cd apps/go-api && go run ./cmd/replay-build --facts <film>` puis lire `facts/<short8>.facts.json`. Comparer `frameCount * frameIntervalMS` au dernier `TimeMS` des `StatRecord`. **Prérequis P2/P3 de l'axe (a).** |
| Q4 | Un objet d'objectif est-il replacé à la frontière de manche SANS événement daté (hypothèse §0.4 du plan v2) ? | Positions de l'objet drapeau/crâne à la première frame de chaque manche sur `64e8adfa`, `fb1a1a72`, `51ebbc0f`, `d9781168` : `cd apps/go-api && go run ./cmd/replay-build --facts <film>` puis confronter aux bornes rendues par `MANCHES_CACHE=<...>/data/cache MANCHES_FILMS=64e8adfa,fb1a1a72,51ebbc0f,d9781168 go test ./internal/analysis/replay/ -run '^TestManchesBornesReleve$' -v` (garde décrite `replay/manches_bornes_research_test.go:31-34`). **C'est le premier item de P3, pas de P1.** |
| Q5 | Le marqueur de portage (image-clé) peut-il devenir un lien `direct` porteur ↔ objet, ou reste-t-il un contrôle ? | Il est mesuré à 37/38 fenêtres de portage en CTF et **totalement absent en Oddball** (`filmdec/keyframe_carrier_mark.go:17-31`). Il ne nomme pas l'objet (`:11-13`). Reprise éventuelle : recompter `coverage.flagCarries.markerObserved` / `markerConfirmed` sur les 3 films CTF du corpus après P3 (champs `replay/document_objectives_live.go:243-244`, peuplement `replay/flag_carries_marker.go:68-89`). **Verdict provisoire : contrôle, pas source** — il dit « ce joueur porte quelque chose », pas « quoi ». |
| Q6 | L'embarquement (`biped_board_vehicle`) peut-il nommer son véhicule par une autre référence que la 1 ? | Réfuté sur 12 films (0/15) selon le rapport V8 §2 cité en `replay/vehicle_rides_events.go:94-96` ; les trois domaines (2/3/7) sont lus dans l'exécutable (`filmdec/event_list.go:347-366`, aiguillage cité `:355-358`). Rouvrir exigerait de relire le descripteur — **plan décodeur d'après v7.5.0**, pas P4. |
| Q7 | Le `bid` publié au roster suffit-il à fermer l'écart de jointure des bots relevé par `scratchpad/review/BOT_SANS_EQUIPE.md` (mémoire, R9) ? | Après P2 : recuire un film à bots (`cmd/replay-build`) et vérifier que chaque entrée `roster[].bot` porte un `bid` qui trouve sa ligne dans `match_participants` ; le repli par nom doit tomber à 0 et être compté. |

---

## 5. Ce que ce lot n'a PAS fait, et pourquoi

- **Aucun code, aucun test, aucun fichier `.go`/`.ts`/`.toml` touché** : c'est le contrat de P1
  (plan v2 §2, lot P : « Inventaire (journal, avant tout code) »).
- **Aucun film décodé, aucune cuisson, aucune écriture sous `data/`** : la mémoire du dépôt interdit
  toute cuisson d'artefacts en lot sans accord préalable ; les questions ouvertes portent les
  commandes, elles ne les ont pas jouées.
- **Aucun test `gamefiles` ni `go test ./...`** lancé : hors périmètre d'un lot de journal.
- **Les découvertes de la lecture ne sont pas traitées** : elles partent au registre
  (`.ai/V7.5/REGISTRE_REPORTS.md`), avec leur condition de reprise.

## 6. Découvertes de cette lecture (consignées, NON traitées)

1. **Commentaire périmé (doc inversée) dans `replay/vehicle_rides.go:29-31`** : « L EVENEMENT NE
   NOMME PAS LE VEHICULE […] Le vehicule d un episode vient donc de la GEOMETRIE […], jamais de
   l evenement. » C'est faux depuis le lot V8 : `filmdec/event_list.go:318-324` publie la réf 1 de
   la SORTIE (105/105 en bande `ti=40`) et `replay/vehicle_rides_events.go:186-187` la consomme, la
   géométrie n'étant plus que le repli (`:248-252`). L'anti-pattern « doc inversée » du CLAUDE.md.
   *Condition de reprise : P4, au moment où le calque véhicule passe au registre.*
2. **`RoundTimerOf` et le hook `SetGameEngineHook` sont du code sans consommateur de production**
   (`filmdec/capture.go:64-68`, `filmdec/components_game_engine.go:91-93`) : le flux n'existe pas
   (0 record de ti=0 / 22 films). Ce n'est PAS du code mort au sens de la règle 7 — c'est de la
   plomberie de décodage validée par garde-rail —, mais le lot 0 du registre-film l'avait
   explicitement rangé comme « correct et inutilisable, faute de flux »
   (`LOTBP_PHASE0.md` §B.0.1). *Condition de reprise : plan décodeur d'après v7.5.0, à l'ouverture
   du dossier « états de partie » ; ne rien supprimer avant.*
3. **`ti=8`, `ti=32`, `ti=17` ne sont inventoriés nulle part** (volumes non négligeables mesurés :
   2 102 / 70 / 135 records certains sur `000d5950`) — report n°7 de `LOTBP_PHASE0.md` §6, jamais
   repris. *Condition de reprise : plan décodeur d'après v7.5.0, phase de recensement.*

---

*Fin de P1. Statut du lot : `[x]` — inventaire rendu, verdict ROSTER rendu, axes (a) à (j) statués,
séquençage P2-P5 posé, questions ouvertes assorties de leur témoin. Aucun item différé.*
