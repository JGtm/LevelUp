# PLAN — Positions de force par carte (calque dérivé, distinct des zones nommées)

> **CLOS le 2026-09-20 — ARRÊT décidé par l'utilisateur après la recherche hybride v2.**
> Aucune lignée (empirique v1, empirique v2, géométrie, fusion) ne tient le critère écrit ;
> la géométrie retrouve 10/13 positions « hauteur / lignes de vue » avec des empreintes serrées
> mais rate par nature « arme / objectif » ; la fusion sans borne de taille colorie des salles ;
> la précision est plafonnée par la finesse de l'oracle. Les étapes 3 à 7 (production) sont
> statuées `[!]` : non ouvertes, arrêt utilisateur. Ce qui est conservé : les documents de
> recherche sous `.ai/V7.5/positions_de_force/`, l'oracle pro v2 (28 positions fortes, 8 cartes),
> les outils de mesure (`cmd/mappower-build`, `cmd/mapgeo-build`), les paquets purs
> (`internal/analysis/powerpos`, `/geo`, `/fusion`) et les harnais de verdict (tag `research`).
> Voie de reprise recommandée par le pilote si le sujet revient : géométrie comme moteur de
> candidats + validation humaine par carte + oracle témoin (voir le bilan dans le thought_log).

> Date : 2026-09-20. Branche `wt/power-positions`, worktree dédié `LevelUp-wt-power-positions`
> (base `feat/v75` à `f2f4f4bdd`). Chantier de RECHERCHE puis de production.
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report, tout item statué).
> Pilotage : la session pilote écrit et arbitre ; des agents Opus exécutent chaque étape sur
> brief, et rendent un rapport avec chiffres. Revue adversariale UNE fois, sur le diff cumulé,
> avant fusion (règle de sobriété du 2026-09-07).
> Le dépôt de travail n'a PAS de `data/` : les bases et les caches se lisent dans
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data` (lecture SEULE, via
> `OpenReadForQuery` — le serveur `air` de l'utilisateur peut tenir la base partagée en RW).

## Objectif et critère de succès

Demande utilisateur du 2026-09-20 : « pour chaque map on puisse colorer ou dessiner un truc
pour indiquer que ce lieu est une power position, comme un calque (distinct des zones de
callout) ; la page replay pourra l'afficher, dans match view on l'affiche d'office, et sur la
page tactique d'office aussi ».

Une **position de force** est un lieu qu'une équipe cherche à tenir : hauteur, lignes de vue
sur plusieurs voies, peu d'accès, proximité d'une arme forte ou d'un objectif. Aucune source du
jeu ne la déclare : elle se DÉRIVE (de nos matchs, et si besoin de la géométrie) ou se saisit à
la main. Ce plan la dérive, et se sert des guides pro comme ORACLE de validation, jamais comme
source.

**Critère de succès** :

1. Un catalogue de référence par carte (`map_power_positions.json`) porte, pour chaque carte
   dont le corpus suffit, des POLYGONES MONDE de positions de force avec score, nombre de
   matchs et nom de la zone nommée dominante quand il existe.
2. Sur au moins **4 cartes du circuit HCS** présentes dans le corpus, l'algorithme retrouve
   les positions décrites par les guides pro (oracle transcrit à l'étape 0) avec un rappel
   ≥ 0,7 et une précision ≥ 0,6 par carte — chiffres ÉCRITS AVANT la mesure, verdict écrit
   après. Un échec n'est pas un ajustement de seuil a posteriori : c'est le NO-GO de l'étape 2
   et le passage à la voie géométrique (étape 2bis).
3. Le calque est affiché : rejeu (bascule « Positions de force », défaut ACTIVÉ, groupe
   Terrain), vue de match (d'office sur le bloc « Où ça se joue »), page Tactique (d'office sur
   le plan de la carte). Une carte sans catalogue n'affiche RIEN et ne propose aucune commande
   qui ne commande rien.
4. Title-agnostique : capability `map.power_positions` ; un titre sans catalogue dégrade
   proprement (absent / 404 typé), jamais de panic.

## Ce que le dépôt fournit déjà (inventaire du 2026-09-20, vérifié sur pièces)

| Brique | Où | Ce qu'elle donne |
|---|---|---|
| Positions de kills | vue `kill_positions_latest` (shared), colonnes `match_id, killer_xuid, time_ms, killer_x/y/z, victim_x/y/z, decode_pass` ; jointure `match_kill_events_latest` sur `(match_id, killer_xuid, time_ms)` | où l'on tue, où l'on meurt, distance et dénivelé d'engagement ; 99,5 % des morts de 951 films au 2026-08-08 |
| Trajectoires + équipe | artefacts `data/cache/replays/halo_infinite/*.json` (`replay.ReplayDocument`, `tracks[].team`, `tracks[].xuid`, `points[].t/x/y/z/h`) | occupation par équipe, cap de visée (~44 % des points), 173 artefacts en cache |
| Grille tactique | `internal/analysis/tactical/` (pas 0,5 m ancré sur l'origine monde, plancher en matchs distincts, `domain.PositionSample`) | l'unité d'analyse et le rasterisage, déjà éprouvés |
| Zones nommées | `reference/map_callouts.json` (`replay.LoadMapCallouts`, deux espaces de clés module / map_id), `zone_attribution.go`, `calloutsLayer.ts` | polygones pour NOMMER une position ; gabarit complet d'un catalogue de référence |
| Objectifs, socles d'armes | `reference/map_objectives.json`, `reference/map_weapon_pads.json` (rempli À LA REQUÊTE dans le document : `replay_service.go:132-137`, `replay_map_objectives.go`) | gabarit du transport « embarqué au document » |
| Fond de carte, calage | endpoints `replay/background(.png)` et `tactical/{map_id}/background(.png)`, `layers/mapBackground.ts` | repère monde → pixels sur les trois surfaces |
| Sol praticable | `internal/hinavmesh` (polygones convexes en monde) | voie géométrique (étape 2bis) |
| Registre des calques du rejeu | `layers/replayCompose.ts` (`LAYER_ORDER`, `SceneToggles`, `sceneLayers`), cuisson statique `useReplayStaticLayers.ts`, préférences `useReplaySettings.ts`, tiroir `ReplaySettingsLayers.tsx` | où se branche un calque statique |
| Vue de match | `features/match-view/MatchPositionsHeatmap.tsx` (mêmes endpoints de fond que le rejeu, projection `{topLeftWorld, scale}`) | où dessiner d'office |
| Tactique | `features/tactical/TacticalPlanCard.tsx` (un seul calque de chaleur, repère `repereDuPlan`, aucune bascule) ; clés `tacticalMapBackground*` SANS joueur (`lib/query/keys.ts:192-206`) | où dessiner d'office ; patron de clé de query par carte |
| Couleurs | `lib/accessibility/semantic-tokens.ts` (`zone-neutral` = personne ne tient ; `warning` = arme de puissance) | aucun token « position de force » : à créer |

Guides pro : pas de jeu de données structuré public. Vues de dessus HCS (Arturo Pérez, 343),
guides texte par carte, carte interactive communautaire (armes, pas positions). Ce sont des
textes à TRANSCRIRE, carte par carte, en vocabulaire de zones nommées.

## Décisions tranchées AVANT exécution

**D1 — L'unité d'analyse est la CELLULE de la grille tactique (0,5 m), pas la zone nommée.**
Une position de force est souvent une partie d'une zone (le haut d'une tour, pas la tour). Les
zones nommées servent à NOMMER le résultat (zone dominante par recouvrement), jamais à le
délimiter. Une position sans zone nommée dominante est publiée avec un nom vide (règle « aucun
nom deviné », comme les callouts Forge).

**D2 — Le signal EMPIRIQUE d'abord ; la géométrie ensuite, et seulement si l'oracle l'exige.**
Signaux par cellule, calculés sur tout le corpus de la carte (tous joueurs, tous matchs, modes
d'arène et BTB séparés par la même clé que la tactique) :
- `kills_depuis` : morts dont le TUEUR était dans la cellule ;
- `morts_dedans` : morts dont la VICTIME était dans la cellule ;
- `portee_mediane` : distance tueur-victime des kills depuis la cellule ;
- `denivele_median` : `killer_z - victim_z` des kills depuis la cellule ;
- `matchs_distincts` : plancher de rareté (même règle que `mappos-build`) ;
- `occupation_gagnants / occupation_perdants` : temps de présence par équipe, depuis les
  artefacts de rejeu joints à l'issue du match (quand l'artefact existe).
Toute lecture et toute cuisson loggent en `slog` structuré (aucune erreur avalée, compteurs
de lignes lues / écartées). Le SCORE combine ces signaux (formule à écrire à l'étape 1 et à
FIGER avant l'étape 2). La
voie géométrique (navmesh, visibilité par lancer de rayons) n'est ouverte que par un NO-GO de
l'étape 2, ou plus tard pour les cartes sans corpus ; elle n'est pas dans le périmètre initial.

**D3 — La sortie est un catalogue de référence versionné, produit par un outil, jamais à la main.**
`data/titles/halo_infinite/reference/map_power_positions.json`, `SchemaVersion = 1`, deux
espaces de clés `maps` (module installé) et `mapsById` (map_id UGC), `PathResolver.MapPowerPositionsPath`,
lecteur `replay.LoadMapPowerPositions` qui REFUSE une autre version, producteur
`cmd/mappower-build` avec invariants mesurés qui font échouer la passe. Une entrée =
`{ id, polygon [[x,y]...], zMin, zMax, score, kills, deaths, matches, calloutStringId, provenance }`.
Les polygones sont l'enveloppe des composantes connexes de cellules retenues (4-connexité),
dilatée d'une demi-cellule, dans le repère monde partagé.

**D4 — Une seule résolution Go, deux transports.** Fonction partagée
`positionsDeForcePourIdentites(nomCarte, mapID)` (patron `zonesPourIdentites`, cascade module
puis map_id). Exposée (a) EMBARQUÉE au document de rejeu, remplie À LA REQUÊTE (`MapPowerPositions`,
patron `MapObjectives` : jamais écrite dans l'artefact) — sert le rejeu ET la vue de match ;
(b) par `GET /players/{slug}/tactical/{map_id}/power-positions` — sert la page Tactique, qui
n'a pas de match. Contrat OpenAPI régénéré (`make openapi-gen`, `make generate-types`).

**D5 — Affichage.** Rejeu : calque `positions-de-force` dans `LAYER_ORDER` juste après
`zones-nommees`, calque STATIQUE cuit hors écran, bascule `replay-show-power-positions` dans
le groupe Terrain, défaut ACTIVÉ (commentaire daté), `available = liste non vide`. Vue de
match : dessiné d'office sur le bloc « Où ça se joue », dans sa propre projection. Tactique :
dessiné d'office sur le plan de la carte, sous la chaleur, dans le repère `repereDuPlan` ; PAS
sur les vignettes de la grille (trop petites pour se lire). Une fonction de tracé unique
`drawPowerPositions(ctx, zones, projection, style)` qui prend la projection en paramètre,
partagée par les trois surfaces via `lib/replay/`.

**D6 — Couleur : nouveau token sémantique `zone-power`** dans `semantic-tokens.ts`, avec sa
justification en commentaire (précédent : `zone-neutral`). Ni `warning` (déjà l'arme de
puissance et le mur), ni `zone-neutral` (dit « personne ne tient »). Remplissage à faible alpha
+ contour ; libellé optionnel (nom de la zone dominante) dans le style des callouts.

**D7 — Title-agnostique.** Capability `map.power_positions` dans `capabilities.toml` de
Halo Infinite ; le service rend « absent » sur un titre sans catalogue (`ErrCapabilityNotSupported`
→ 404 typé côté tactique, champ omis côté document). Aucun `slug == ...`.

**D8 — Pas de calque sans preuve.** Une carte n'entre au catalogue que si (a) son corpus
atteint le plancher de matchs distincts fixé à l'étape 1 et (b) l'étape 2 a rendu GO. Une carte
en dessous du plancher est ABSENTE, pas « vide ». La provenance de chaque entrée dit le corpus
(`empirique`, nombre de matchs, date de cuisson).

## Étapes

> Ordre : les étapes 0 et 1 sont INDÉPENDANTES jusqu'à l'item 1.4 (qui prend la liste des
> cartes de l'oracle) ; sur décision du pilote elles s'exécutent en parallèle par deux agents.
> Le recensement du corpus (0.1) est fait par l'agent de l'étape 1, qui construit la lecture
> de la base : l'agent de l'oracle couvre TOUT le circuit HCS et le recensement filtre ensuite.
> Toutes les autres étapes suivent l'ordre strict, gate passé avant d'ouvrir la suivante.
> Le serveur `air` de l'utilisateur tient la base partagée en RW (constaté le 2026-09-20) :
> toute lecture passe par `OpenReadForQuery`, jamais par le CLI duckdb ni un attach RO.

### Étape 0 — Oracle : transcrire les positions de force des cartes HCS depuis les guides pro

Exécutant : agent Opus avec recherche web. Livrable :
`.ai/V7.5/positions_de_force/ORACLE_PRO_2026-09-20.md`.

- [x] 0.1 Recenser le corpus par carte : nombre de matchs avec positions de kills
      (`kill_positions_latest` joint à `match_registry`, par nom de carte et par catégorie
      arène / BTB) et nombre d'artefacts de rejeu en cache par carte. Tableau trié.
      → **fait par l'agent de l'étape 1** (décision du pilote : l'oracle couvre tout le
      circuit HCS sans filtrage préalable, le recensement filtrera ensuite).
      Livré : `recensement_2026-09-20.md` (89 cartes) + section « Corpus » de
      `MESURE_EMPIRIQUE_2026-09-20.md`, produits par `mappower-build --recensement`.
      Clé d'agrégation = `decfilm.NormalizeMapName` (rabote « - Ranked » / « Heavies » :
      sans quoi Live Fire compterait 50 + 30 matchs au lieu de 80) ; axe arène / BTB par
      `halo_infinite.InferModeCategoryFromPairName` — la colonne `mode_category` du registre
      est inutilisable (« Other » sur 126 matchs de « Ranked:Strongholds on Live Fire »).
      9 110 matchs retenus, 34 sans nom de carte, 0 artefact orphelin. Dix cartes HCS au
      corpus (live fire 80, recharge 78, streets 75, aquarius 57, forbidden 44, origin 37,
      lattice 28, solitude 26, fortress 18, empyrean 17) ; **Argyle absente du corpus**
      (aucun match). 92 artefacts de rejeu pour tout le titre, 0 à 6 par carte.
- [~] 0.2 Choisir les cartes de l'oracle : les cartes du circuit HCS (Aquarius, Live Fire,
      Recharge, Streets, Forbidden, Solitude, Empyrean, Argyle, Fortress, Origin, Lattice…)
      présentes dans le corpus avec le plus de matchs ; au moins 4, au plus 6.
      → **filtrage par le recensement de l'étape 1**. Couvert par l'oracle : 16 cartes
      recherchées, **12 avec au moins une position** (Aquarius `ctf_aquarius`, Live Fire
      `sgh_interlock`, Recharge `sgh_blueprint`, Streets `sgh_streets`, Bazaar `ctf_bazaar`,
      Catalyst `catalyst`, Forbidden `ctf_forbidden`, Argyle `dd600260-…`, Empyrean
      `d035fc3e-…`, Solitude `f1cc3b4e-…`, Interference `654dff62-…`, Banished Narrows
      `9ad226d8-…`) ; 3 sans aucune source de position (Fortress, Origin, Lattice) ;
      Illusion écartée (vocabulaire du dépôt inrattachable aux guides).
- [x] 0.3 Pour chaque carte retenue, transcrire les positions de force décrites par au moins
      DEUX sources indépendantes (guides écrits, vidéos d'analyse HCS, discussions de joueurs
      pro), dans le VOCABULAIRE des zones nommées de la carte (`reference/callouts_i18n.csv`,
      colonne EN ; si le guide emploie un nom absent, le noter tel quel avec la zone la plus
      proche). Par position : nom, pourquoi (hauteur / lignes de vue / arme / objectif),
      sources citées, confiance (forte = ≥ 2 sources concordantes ; faible = 1 source).
      → fait, section 2 de l'oracle. Réserve écrite : sur 6 cartes seulement la confiance
      `forte` est atteinte ; Reddit / YouTube / Liquipedia sont inaccessibles à l'outillage,
      donc l'analyse proprement HCS manque pour les cartes postérieures à 2023.
- [x] 0.4 Écrire l'oracle sous forme de tableau exploitable par un test : une ligne par
      (carte, zone nommée EN, confiance). Les positions de confiance faible sont dans
      l'oracle mais ne comptent pas dans le rappel.
      → fait, section 3 : 84 lignes à 6 colonnes `| carte_cle | zone_en | nom_guide |
      confiance | raison | sources |`, plus une table de 26 contre-exemples. Règle ajoutée
      et à respecter par le test : une ligne `zone_en = ?` (15 lignes) n'est exploitable ni
      au rappel ni à la précision.

Gate 0 : le document existe, ≥ 4 cartes, chaque position a ≥ 1 source citée par URL, le
tableau final est parsable (une ligne par position). Le pilote relit et valide.

### Étape 1 — Mesure empirique par cellule, et figer le score

Exécutant : agent Opus (Go). Livrables : paquet `internal/analysis/powerpos/` (pur, testé),
outil `cmd/mappower-build` en mode `--mesure`, dossier
`.ai/V7.5/positions_de_force/mesures_2026-09-20/` (CSV par carte + PNG de contrôle),
document `MESURE_EMPIRIQUE_2026-09-20.md`.

- [x] 1.1 `internal/analysis/powerpos/` : types d'entrée (`KillSample{MatchID, KillerX/Y/Z,
      VictimX/Y/Z}`, `PresenceSample{MatchID, Team, Won bool, X, Y, DurMS}`), grille réutilisant
      `internal/analysis/tactical` (même pas, même ancrage), accumulateurs par cellule
      (`kills_depuis, morts_dedans, portee_mediane, denivele_median, matchs_distincts,
      occupation_gagnants, occupation_perdants`). Tests unitaires purs (grille, plancher,
      médianes, composantes connexes, enveloppe).
      → `doc.go`, `types.go`, `accumulateur.go`, `mediane.go`, `composantes.go`,
      `enveloppe.go`, `score.go`, `selection.go` (8 fichiers, 1 220 L, aucun > 500 L) ;
      tests `accumulateur_test.go`, `mediane_test.go`, `composantes_test.go`,
      `enveloppe_test.go`, `score_test.go`, `selection_test.go`. Le paquet IMPORTE
      `tactical.Grille` / `tactical.Cellule` (aucune duplication de grille). Écarts assumés
      et documentés : `matchs_distincts` est dédoublé en `MatchsKills` / `MatchsPresence`
      (les deux sources n'ont pas la même densité) ; l'enveloppe v1 est convexe et dilatée
      d'une demi-cellule ; les composantes sont en 4-connexité (le flood-fill de
      `tactical.GrappesDeSpawn` est privé ET en 8-connexité — 2e copie, cf. Découvertes).
- [x] 1.2 Lecture des données dans `cmd/mappower-build` : kills par carte via
      `OpenReadForQuery` sur la base partagée (vues `_latest` UNIQUEMENT), présence par équipe
      depuis les artefacts de rejeu (`tracks[].team` + issue du match par xuid depuis la base).
      Aucune écriture en base. Chemin des données par drapeau `--data-root`.
      → `corpus.go`, `mesure.go`, `presence.go`, `controle_variantes.go`. Une seule passe sur
      `kill_positions_latest`, dispatch en mémoire. Issue par (match, xuid) depuis
      `match_participants`. Durée d'occupation = écart au point suivant × `frameIntervalMs`
      (les points d'une piste ne sont pas équidistants). Compteurs slog :
      kills lus / ignorés, artefacts lus / en échec, pistes sans issue, points lus,
      présences ignorées, cellules scorables, positions retenues.
- [x] 1.3 Rendu PNG de contrôle par carte : fond de carte (`reference/map_backgrounds`) +
      cellules colorées par `kills_depuis / (kills_depuis + morts_dedans)` et par occupation
      gagnants − perdants, zones nommées en contour pour se repérer. Une image par signal.
      → `png.go`, TROIS planches par carte (`_duel`, `_occupation`, `_score`) ; la troisième
      porte le score figé et le contour des positions retenues. Calage lu dans le sidecar
      publié (`MapBackgroundCalibration`), zones résolues par la même cascade que le service
      (module puis map_id). 33 planches pour 11 cartes ; `lattice` n'a pas de fond publié,
      ses CSV existent, ses PNG non (journalisé, non fatal).
- [x] 1.4 Mesurer sur les cartes de l'oracle ET sur deux cartes hors oracle (témoins). Tableau
      par carte : cellules peuplées, plancher de matchs distincts retenu (mesuré comme dans
      `mappos-build` : le rayon du nuage en fonction du plancher), distribution des signaux.
      → **12 cartes** : les 10 cartes HCS du corpus + 2 témoins hors HCS (illusion 55 matchs,
      bazaar 52). Tables dans `mesures_2026-09-20/_rapport.md` et reprises dans
      `MESURE_EMPIRIQUE_2026-09-20.md`. Mesure marquante : sur les positions de KILL le nuage
      est DÉJÀ compact (rayon p99 24,3 → 23,4 m de 1 à 3 matchs sur live fire), à l'inverse
      des positions de passage de `mappos-build` (268 → 19,4 m) — le plancher ne sert donc
      pas à rogner des bras hors de l'arène mais à écarter l'anecdote d'un seul match.
- [x] 1.5 Figer la FORMULE de score et la règle de sélection (seuils, plancher, taille
      minimale d'une composante) dans `MESURE_EMPIRIQUE_2026-09-20.md`, AVANT l'étape 2, avec
      une justification par signal. Écrire aussi ce qui a été essayé et écarté.
      → section « Score figé — 2026-09-20 », implémentée par `powerpos.ReglageV1()`.
      **Le score se calcule sur un DISQUE DE 2 m, pas sur la cellule** : le rapport de duel
      par cellule de 0,5 m rend la MÊME dispersion sur les 12 cartes au centième
      (0,29 / 0,38 / 0,50 / 0,62 / 0,71) — c'est du bruit binomial, pas du terrain.
      `SCORE = 0,50·avantage + 0,25·intensité + 0,15·hauteur + 0,10·portée`, l'avantage étant
      le rapport de duel du disque rétréci vers 0,5 (force 40), les normalisations étant PAR
      CARTE. Sélection : ≥ 3 matchs distincts sur la cellule centrale, ≥ 40 engagements dans
      le disque, score ≥ max(p90 de la carte, 0,65), composantes 4-connexes ≥ 12 cellules,
      ≤ 8 positions par carte. **Occupation par équipe écartée (poids 0)**, motifs mesurés
      écrits. Résultat : 0 à 5 positions par carte, de 6 à 31 m² ; fortress et empyrean
      rendent ZÉRO position (3 % de cellules scorables) — comportement voulu par D8.

Gate 1 : `go test ./internal/analysis/powerpos/` vert ; `go vet` propre ; CSV et PNG présents
pour ≥ 6 cartes ; la formule est écrite et datée. Aucun seuil ne sera retouché à l'étape 2.

**Gate 1 PASSÉ le 2026-09-20** : `go build ./...` OK ;
`go vet ./internal/analysis/powerpos/ ./cmd/mappower-build/` OK ;
`go test ./internal/analysis/powerpos/` OK ; `gofmt -l` vide ;
`golangci-lint run` sur les deux paquets : 0 issue ; `go test ./internal/archlint/
./internal/analysis/tactical/` OK (aucun ratchet cassé). **12 cartes** avec CSV,
**11 cartes** avec les 3 PNG (lattice n'a pas de fond publié). Formule écrite et datée.

### Étape 2 — Verdict contre l'oracle (GO / NO-GO)

Exécutant : agent Opus. Livrable : `.ai/V7.5/positions_de_force/VERDICT_ORACLE_2026-09-20.md`.

- [x] 2.1 Test Go de recherche (tag `research`, lit les CSV de l'étape 1 et l'oracle de
      l'étape 0) : pour chaque carte, une position calculée est « retrouvée » si son polygone
      recouvre ≥ 30 % de la zone nommée de l'oracle OU si son centroïde tombe dedans ; rappel =
      positions fortes de l'oracle retrouvées / positions fortes ; précision = positions
      calculées qui touchent une zone de l'oracle (forte ou faible) / positions calculées.
      → `cmd/mappower-build/verdict_{oracle,sources,geometrie,rapport,diagnostic}_research_test.go`
      (`//go:build research`, `TestVerdictOracle`, 5 fichiers, aucun > 500 L). **Prérequis
      livré au passage** : les polygones des positions SÉLECTIONNÉES n'étaient écrits nulle
      part (CSV = cellules, rapport = centres seulement) → sortie `positions.json` ajoutée au
      mode `--mesure` (`positions_json.go`), pure SÉRIALISATION de `Cible.Positions`, aucun
      seuil ni calcul touché ; passe regénérée avec la commande de l'étape 1, chiffres
      identiques (3/4/4/5/4/2/1/3/4/1/0/0 positions). Recouvrements mesurés par
      échantillonnage au pas de 0,2 m (les zones du catalogue sont concaves, trouées et en
      plusieurs morceaux). Lignes `zone_en = ?` ignorées, zones `forte` absentes du catalogue
      sorties du dénominateur et signalées.
- [x] 2.2 Tableau par carte : rappel, précision, faux positifs nommés, faux négatifs nommés.
      → `VERDICT_ORACLE_2026-09-20.md` §1 (rappel / précision / témoin / rappel hors « arme »
      seule), §3 (contre-exemples), §4 (faux positifs et faux négatifs nommés), §5 (chaque
      position calculée, sa zone nommée dominante, sa part et son aire — la table du pilote).
- [x] 2.3 Verdict : GO si ≥ 4 cartes tiennent rappel ≥ 0,7 ET précision ≥ 0,6 ; sinon NO-GO
      avec le diagnostic (quel signal manque, quelles positions échappent et pourquoi).
      → **NO-GO** : 1 carte sur 4 (bazaar, 1 seule zone `forte`) hors Live Fire. Diagnostic
      §6 du verdict, zone par zone, en rejouant le score figé sur les cellules de la zone :
      sur 13 zones `forte` manquées, **6 composante trop petite** (des cellules passent le
      seuil mais l'amas 4-connexe fait 2 à 10 cellules pour 12 exigées), **6 score sous le
      seuil** (dont Orange Pipes à 0,678 contre 0,680 et Platform à 0,661 contre 0,667),
      **1 non scorable**. Aucun seuil retouché.
- [x] 2.4 Témoin négatif : décaler l'oracle d'une zone (permutation circulaire des zones) et
      vérifier que rappel et précision s'effondrent. Sans témoin, un GO ne vaut rien.
      → §2 du verdict, avec la colonne du décalage appliqué. **Le témoin ne s'effondre pas :
      rappel réel 0,25 en moyenne contre 0,22 au témoin.** Deux lectures, toutes deux écrites :
      (a) l'appariement ne porte quasiment aucun signal — c'est le constat du NO-GO ; (b) le
      témoin lui-même est FAIBLE parce que la liste des zones est triée par nom et que le
      voisin alphabétique est souvent le voisin géographique (`Dried Rat Hole` →
      `Dried Rat Tunnel`, `Whirlpool Dam` → `Whirlpool Ledge`, `Subway Balcony` →
      `Subway Bend`). Il aurait suffi à interdire un GO ; il n'avait pas à le faire.

Gate 2 : verdict écrit avec les chiffres et le témoin. Le PILOTE lit, tranche, et informe
l'utilisateur AVANT d'ouvrir l'étape 3 (GO) ou 2bis (NO-GO).

**Gate 2 PASSÉ le 2026-09-20** : `go vet -tags research ./...` sur tout le module : 0 issue ;
`MAPPOWER_DATA_ROOT=... go test -tags research ./cmd/mappower-build/ -run Verdict` : PASS ;
`go build ./...`, `go vet ./cmd/mappower-build/ ./internal/analysis/powerpos/`, `gofmt -l` vide,
`go test ./internal/analysis/powerpos/ ./internal/archlint/` : verts (aucun ratchet cassé).
`VERDICT_ORACLE_2026-09-20.md` porte le verdict en une ligne en tête, les 7 sections, le
témoin et le diagnostic. **Verdict : NO-GO** — l'étape 2bis (voie géométrique) est la suite
prévue par le plan, la décision revient au pilote.

### Étape 2bis — Recherche hybride v2 (OUVERTE par le NO-GO du 2026-09-20, décision utilisateur)

Le verdict v1 (`VERDICT_ORACLE_2026-09-20.md`) est NO-GO : le score « avantage au duel »
retrouve les bases et les points de défense, pas les positions que les pros tiennent (Top Mid,
Hydro, Whirlpool sont détectées mais fragmentées ou juste sous le seuil). Décision utilisateur
du 2026-09-20, en réponse au questionnaire : **poursuivre l'empirique ET la géométrie, en
combinaison** (« les power positions sont une combinaison de tous ces angles plutôt
qu'exclusifs »), **vérifier le catalogue pro** (« voir si on ne rate pas un truc »), et tenir
compte de ce que **nos données ne viennent pas de joueurs pro** (« à part Nuzzle les autres
sont plutôt mauvais »). Référence fournie par l'utilisateur : la pré-analyse de sa page Notion
« Backlog LevelUp » (modèle Gemini léger) — cinq variables sur le maillage de déplacement :
altitude relative H, visibilité sortante V (lancer de rayons), exposition E (angle sous lequel
on peut être touché), ressources proches R (distance de déplacement aux armes / objectifs),
échappatoire M (distance au couvert) ; `Score = w1·H + w2·V − w3·E + w4·R + w5·M` ; grille
50 cm, matrice d'intervisibilité, extraction des maxima locaux. C'est la voie géométrique
telle qu'elle se pratique (EQS d'Unreal) ; elle est adoptée comme formulation de 2bis.C.

Décisions supplémentaires (D9-D12) :

**D9 — Trois angles, un seul score final.** Empirique (ce que nos matchs montrent),
géométrique (ce que la carte permet), oracle (ce que les pros disent). Le score final est
une combinaison des deux premiers ; l'oracle reste le juge, jamais une entrée du score.

**D10 — L'empirique v2 doit être ROBUSTE au niveau des joueurs.** Deux parades, mesurées
séparément : (a) pondérer chaque kill par le rang du TUEUR (CSR / LUSR du match, vues
`_latest`) ou restreindre aux tueurs au-dessus d'un rang, et (b) préférer des signaux
GÉOMÉTRIQUES tirés des données plutôt que des ratios de duel : dispersion angulaire des
positions des tueurs pour les morts subies dans une cellule (= exposition mesurée),
dispersion angulaire des victimes pour les kills depuis la cellule (= couverture de voies
mesurée), portée. Le ratio de duel v1 reste au CSV, à poids réévalué.

**D11 — Verdict v2 avec cartes tenues à l'écart.** Les poids de fusion se choisissent sur
TROIS cartes (calibrage) et se jugent sur les autres (validation), jamais l'inverse. Les
seuils de succès du plan (rappel 0,7 / précision 0,6 sur ≥ 4 cartes) restent, la précision
devient « touche une zone FORTE ou une zone faible confirmée par l'oracle v2 » (découverte 9 :
les 67 zones faibles v1 pavent les cartes), et le témoin négatif devient un décalage
GÉOGRAPHIQUE (zone la plus éloignée de la carte), pas alphabétique (découverte du verdict).

**D12 — Le verdict v1 n'est pas rouvert.** `ReglageV1` reste dans le dépôt tel quel (preuve
datée) ; la v2 est un réglage NEUF (`ReglageV2`), figé à son tour avant son verdict.

- [x] 2bis.A **Oracle v2 — audit et densification.** Relire l'oracle v1 zone par zone (les
      rattachements lexicaux, les « ? »), puis densifier par le NAVIGATEUR (MCP chrome) là où
      WebFetch était refusé : r/CompetitiveHalo, descriptions et chapitres de VOD HCS, guides
      de coaching, liquipedia. Cible : ≥ 3 positions fortes sur ≥ 6 cartes, contre-exemples
      confirmés. Livrable `ORACLE_PRO_V2_2026-09-20.md`, même table parsable, colonne
      « v1 / v2 » par ligne, et une section « ce qu'on ratait » (positions absentes de v1).
      → fait, `ORACLE_PRO_V2_2026-09-20.md` (§1-§9). Navigateur réel (MCP `chrome-devtools`) :
      Reddit accessible ET exploitable (l'endpoint `.json` d'un fil rend l'anglais brut, non
      traduit — bien plus fiable que la page rendue) ; liquipedia et halo.fandom.com restent
      bloqués (défi Cloudflare / interstitiel qui ne se résout pas même en navigateur réel,
      `wait_for` en échec après 8 s) ; x.com est un mur de connexion (pas un blocage anti-bot) ;
      YouTube sert descriptions/chapitres mais PAS les sous-titres (`timedtext` répond HTTP 200
      corps vide, vérifié sur pièces). **~35 fils Reddit lus, 28 positions `forte`** (16 en v1
      post-relecture pilote → +12, dont 2 par correction d'audit sur Bazaar `Cafe`/`Den` — 2
      sources déjà présentes en v1 mais mal comptées `faible`) réparties sur **8 cartes avec
      au moins 1 forte**. **Gate cible (≥ 3 fortes sur ≥ 6 cartes) NON ATTEINT au sens strict :
      4 cartes** (Recharge 9, Live Fire 3, Streets 3, Bazaar 5) ; les 8 autres sont couvertes
      par la clause alternative du gate (démonstration chiffrée, §8 et §2.12 du document) —
      3 cartes sans AUCUNE zone officielle au dépôt (Fortress, Origin, Banished Narrows),
      3 cartes en dessous du seuil malgré recherche dédiée (Aquarius 2, Forbidden 2, Catalyst
      0), 3 cartes où la connaissance pro existe mais ne rattache à aucun nom du catalogue
      (Lattice, Argyle, Interference — angle mort de VOCABULAIRE, pas de source ; le cas le
      plus net : Lattice, sortie le 5 août 2026, a déjà un vocabulaire communautaire vivant et
      une partie pro 8s documentée, mais 0 zone nommée). 11 rattachements lexicaux de v1 §5.2
      audités : 0 invalidé, 4 promus forte, 7 non renforcés mais non contredits. Les 4 zones
      « arme seule » (Live Fire `Hallway`, Recharge `Pit`, Streets `Main Street`, Forbidden
      `Center Bridge`) restent non-tenues après audit — Streets `Main Street` doublement
      confirmé comme zone qu'on ne tient PAS. 1 contradiction non tranchée signalée au pilote
      (Empyrean : v1 dit l'épée remplacée par le Heatwave, des fils 2024-2025 emploient
      « sword » activement). Aucun fichier de code touché, aucun commit.
- [x] 2bis.B **Empirique v2.** Dans `internal/analysis/powerpos/` : pondération par rang du
      tueur (D10a), exposition et couverture angulaires (D10b), sortie CSV par cellule avec
      les nouveaux signaux ; PNG de contrôle ; `ReglageV2` figé sur les distributions des 3
      cartes de calibrage SEULEMENT, écrit et daté avant 2bis.D. Vérifier d'abord où vit le
      rang des joueurs par match (skill rank / CSR, vues `_latest`, base partagée ou base
      joueur) et sa couverture sur le corpus.
      → **Rang** : une seule source par match ET par joueur, `match_csrs_latest` (base
      partagée, clé `(match_id, xuid)`) — `match_skill_rank_latest` est sans `xuid` (rang
      du propriétaire de la base), les snapshots et `player_skill_state_v2` sont des états
      sans `match_id`. Couverture : **28 % des kills** (13 057 / 46 795) ont un tueur au
      rang connu, de 0 % (illusion, bazaar, forbidden, fortress, empyrean : jouées en
      social) à 97 % (lattice) ; recharge 47 %, streets 38 %, aquarius 19 %. Rampe de poids
      0,5 (p10 = 1 251) à 1,5 (p90 = 1 590) mesurée sur les 28 937 rangs du titre (pas sur
      les cartes demandées : sinon le poids d'un kill dépendrait de `--cartes`), rang
      inconnu = 1,0 ; variante « tueurs au-dessus de la médiane » comptée au CSV.
      **Signaux angulaires** (`angles.go`, sommes vectorielles additives, dispersion corrigée
      du biais de Rayleigh, 7 tests purs) : couverture et abri **anticorrélés à −0,54 /
      −0,70 / −0,73** sur les 3 cartes de calibrage (un lieu ouvert tue et meurt de partout)
      → poids ÉGAUX pour n'en garder que l'asymétrie (étalement 0,14 contre 0,40 par axe),
      indépendante de l'avantage (corr −0,21 à −0,32). **`ReglageV2` figé le 2026-09-20
      14:06** (`reglage_v2.go`, `MESURE_EMPIRIQUE_V2_2026-09-20.md`) : poids par la règle
      « part voulue / étalement mesuré » (0,26 couverture + 0,26 abri + 0,21 avantage
      pondéré + 0,16 hauteur + 0,09 intensité + 0,02 portée), hystérésis amorce p95 /
      croissance p90 (`selection.go`), fermeture morphologique de rayon 1 (`morphologie.go`),
      8-connexité après fermeture, 10 cellules minimum, plancher absolu 0,57. Passe sur les
      12 cartes : **35 positions** (live fire 5, recharge 4, streets 4, aquarius 4, illusion
      3, bazaar 3, forbidden 2, origin 4, lattice 3, solitude 3, fortress 0, empyrean 0) de
      5,0 à 58,6 m² — `mesures_v2_2026-09-20/` (CSV à 25 colonnes, 5 planches par carte,
      `positions_v2.json`, `_rapport.md` avec 4 sections neuves). Live Fire mesurée SANS sa
      variante classée (`--exclure-variantes`, 3 854 kills écartés, motif daté). Travail de
      l'agent coupé repris et relu : un défaut corrigé (`construis` comptait dans les
      moyennes une cellule de fermeture scorable sous le seuil — test
      `TestFermetureNEntrePasDansLesMoyennes`), `Diagnostique` ajouté (tailles des
      composantes avant / après fermeture, ce qui manquait au verdict v1). V1 rejouée à
      l'identique (`TestSelectionV1Inchangee`, mêmes 4 / 4 / 5 positions sur les cartes de
      calibrage).
- [x] 2bis.C **Géométrie.** Inventaire des sources d'occlusion disponibles (navmesh
      `internal/hinavmesh`, structure `reference/map_structure/`, triangles des `.module` via
      `internal/himap` — jeu installé requis, outil de cuisson hors ligne comme
      `mapcallouts-build` natif) ; prototype H / V / E / R / M sur UNE carte de calibrage
      (Recharge) : grille 50 cm sur le navmesh, rayons 2,5D à hauteur d'yeux, distance de
      déplacement aux socles (`map_weapon_pads.json`) et aux objectifs
      (`map_objectives.json`) ; PNG de contrôle ; coût mesuré par carte ; puis les 5 autres
      cartes de l'oracle si le coût le permet. Score géométrique figé sur les 3 cartes de
      calibrage.
      → **fait (2026-09-20)**, `GEOMETRIE_2026-09-20.md`. **Inventaire** : aucune carte native
      n'a de navmesh (blob UGC Forge seulement, dépôt `.ai/re_dump/navmesh` absent du poste),
      `map_structure` = 2 modules de boîtes sans forme ; la SEULE source d'occlusion est le
      maillage de RENDU des `.module` (3,3 M triangles sur Recharge, 26 M sur Forbidden) +
      la coquille de mort `sddt` (5 cartes sur 6). **« Grille sur le navmesh » est donc devenu
      « sol DÉRIVÉ »** : surfaces montantes + hauteur libre dans une grille de voxels 0,25 m,
      graphe de déplacement ORIENTÉ (marche 0,75 / saut 1,6 / chute 5 m, rayons de passage
      poitrine et saut, verticale libre), atteignabilité depuis ancres + socles, élagage des
      nœuds sans retour vers un objectif. Paquet pur `internal/analysis/powerpos/geo/`
      (14 fichiers, 14 tests, aucun > 300 L) + `cmd/mapgeo-build/` (+ test `gamefiles`, 5 s,
      cible Makefile ajoutée). **Six cartes cuites en 62 s** (3,4 s Recharge, 23 s Forbidden ;
      les rayons coûtent 0,1 s, la voxelisation 60-80 %) : 2 901 à 5 914 nœuds, 25/25 ancres
      posées sur Recharge. Recharge : H p5..p95 −1,5..+1,6 m, V p50 0,10, E p50 0,50,
      **V et E corrélés à 0,82-0,89** sur les trois cartes de calibrage. **Réglage figé
      `ReglageGeoV1`** sans oracle : `0,30·H + 0,20·V − 0,15·E + 0,20·R + 0,15·M`, p90,
      maximum local 3 m, croissance 4 m, ≥ 12 nœuds, ≤ 8. 5 à 8 positions par carte, 5 à
      36 m², toutes en hauteur sur Recharge. `positions_geo.json` aux clés du fichier
      empirique (`axe`, `map_id_dominant`, `polygone`…) pour le harnais de 2bis.D. Limite
      structurelle mesurée : le rendu n'est pas la collision (artefacts de Recharge : joueurs
      à 0,5-1,5 m dans l'escalier de la fosse que des corniches de rendu rejetaient) → hauteur
      libre 1,5 m ; bloc de collision du sbsp non décodé (découverte 17).
- [x] 2bis.D **Fusion et verdict v2.** Poids empirique / géométrique choisis sur les 3 cartes
      de calibrage, verdict sur les cartes de validation, témoin géographique, contre-exemples
      à zéro. GO → étape 3 avec le score hybride comme producteur ; NO-GO → rapport au
      user avec les deux options restantes (catalogue depuis l'oracle, arrêt).
      → **PREMIÈRE MOITIÉ FAITE (2026-09-20)** — harnais de verdict v2
      (`cmd/mappower-build/verdict_v2_{oracle,diagnostic,rapport,rapport_tables}_research_test.go`,
      tag `research`, fichier de positions par `MAPPOWER_POSITIONS`, document par
      `MAPPOWER_VERDICT_SORTIE`) et **verdict de l'EMPIRIQUE V2 SEUL : NO-GO**
      (`VERDICT_V2_2026-09-20.md`) : 0 carte de validation sur 4 tient rappel ≥ 0,7 ET
      précision ≥ 0,6 — bazaar 0,20 / 0,33, live fire 0,33 / 0,40 (mesurée sans la variante
      classée), forbidden 0,50 / 0,50 et solitude 0,50 / 0,33 (indéterminées, 2 fortes),
      empyrean 0 position. Zéro piège pur sur les cartes de validation (1 sur aquarius,
      calibrage : Blue Courtyard). Témoin géographique concluant en moyenne (rappel 0,37 → 0,06,
      précision 0,35 → 0,03), une exception documentée (Live Fire, découverte 15). Rappel HV
      (hauteur / lignes de vue) 3/9 sur les cartes de validation contre AO (arme / objectif)
      1/5 ; toutes cartes : HV 7/17, AO 3/11. Progrès v1 → v2 à règles égales : aucune carte
      de validation ne tient dans un cas comme dans l'autre ; le rappel monte sur aquarius
      (0 → 1,00), forbidden et solitude (0 → 0,50), recharge (0,33 → 0,44), stagne sur bazaar
      et live fire, RECULE sur streets (0,33 → 0) ; pièges purs sur validation 1 → 0. Les deux
      couloirs : `streets__arene__2` (88 cellules, 58,6 m²) = SALLE (Old Town couverte à 60 %),
      `recharge__arene__4` (85 cellules) = à cheval Whirlpool Dam 49 % / Elevator 36 %. Diagnostic
      (rejeu v2 prouvé fidèle sur les 8 cartes) : 18 fortes manquées = 9 « score sous la
      croissance », 5 « retenue ailleurs » (une composante touche la zone sans la couvrir),
      2 « sans amorce » (Control Room à 0,001 du p95, Orange Pipes), 2 « non scorable ».
      Précision v2 = « touche une forte » (brief du pilote), et non « forte ou faible
      confirmée » (formulation initiale de D11) : les faibles pavent les cartes.
      → **SECONDE MOITIÉ FAITE (2026-09-20)** — `FUSION_2026-09-20.md` (méthode, balayage,
      lecture), `VERDICT_GEO_2026-09-20.md` et `VERDICT_FUSION_2026-09-20.md` (générés).
      **Alignement** des deux grilles vérifié sur pièces (centre = (col + 0,5) × 0,5 des deux
      côtés, écart 0 sur Recharge ; contrôle rejoué à chaque lecture par `fusion_lecture.go`),
      aucun décalage à corriger. **Géométrie seule : NO-GO** strict (1/4 : Bazaar 1,00 / 0,88 ;
      Live Fire 0,67 / 0,71 + 1 piège pur Canal ; Forbidden 0,50 / 0,12) et élargi (1/6) ;
      témoin 0,68 → 0,07 sauf Recharge (0,22 réel, 0,40 témoin : 5 fortes sur 9 remplacées par
      Hydro, 488 m²) ; HV 10/13 contre AO 4/11 ; diagnostic par zone rejoué sur le CSV par
      nœud : 0 sol absent, 5 « retenue ailleurs », 3 sous le seuil, 1 sans maximum local, 1
      trop petite / hors plafond. **Fusion** : paquet pur `powerpos/fusion` (agrégation des
      étages au maximum, normalisation p5..p95 réutilisée, absences explicites, sélection v2
      réutilisée), mode `--fusion` de `mappower-build` SANS base, balayage de 112 candidats sur
      les 3 cartes de calibrage (`fusion_2026-09-20/_calibrage_fusion.md`, critère écrit :
      rappel puis précision puis pièges purs) → **`ReglageFusionV1` figé** : 0,6·geo + 0,4·emp,
      emp absent 0,25, geo absent 0, amorce p90 / croissance p80, plancher 0,70 posé sous
      l'amorce de calibrage la plus basse (test `TestFusionCalibrage` échoue si le figé n'est
      pas le gagnant). **Verdict FUSION : NO-GO strict (1/4) et élargi (1/6)** — Bazaar
      1,00 / 1,00 mais par UNE position de 884 cellules / 500 m² (Market 77 %, Tower 99 %, Den
      100 %) ; Live Fire 1,00 / 0,25 + 1 piège pur (Canal sous Nest, découverte 8) ; Forbidden
      0,50 / 0,25 ; 17 positions sur 32 > 60 cellules, 8 « salles entières ». Toutes les fortes
      HV retrouvées, 4 faux négatifs AO (Overgrown Rat Hole, Hydro, Orange Pipes, Pit). Le
      critère de balayage sans borne de taille a acheté du rappel avec de la surface
      (découverte 22). Empyrean et Solitude : Forge, module = canevas `fo11_blank`, NON
      cuisables → « empirique seul », hors verdict de fusion. Quatre lignées à règles égales
      (§8 des documents) : v1 0/0, v2 0/0, géo 1/1, fusion 1/1 (strict / élargi). Harnais :
      lignée déduite du nom de fichier, verdict strict + élargi, planches `*_verdict.png`
      (positions en vert, fortes en orange) pour les trois lignées. Aucun réglage figé touché.

Gate 2bis : documents d'oracle v2, de mesure v2, de géométrie et de verdict v2 écrits avec
chiffres ; tests Go des paquets touchés verts ; réglages v2 datés AVANT le verdict ; le
pilote lit et rapporte à l'utilisateur.

### Étape 3 — Catalogue de référence

- [!] 3.1 `PathResolver.MapPowerPositionsPath` (+ test), `replay.LoadMapPowerPositions`
      (`SchemaVersion = 1`, refus d'une autre version, deux espaces de clés), types Go.
- [!] 3.2 `cmd/mappower-build` mode production : score figé, sélection, composantes,
      enveloppes, nommage par zone dominante, provenance ; invariants qui font échouer la passe
      (nombre de cartes ≥ 4, aucune entrée sans polygone, aucune carte sous le plancher) ;
      garde anti-perte contre le catalogue existant (patron `mapcallouts-build/garde_perte.go`).
- [!] 3.3 Cuire le catalogue sur tout le corpus ; le versionner ; noter la commande exacte et
      la date dans le document de mesure.
- [!] 3.4 Capability `map.power_positions` (`capabilities.toml` Halo Infinite, absent Halo 5).

Gate 3 : `go test ./internal/domain/title/ ./internal/games/halo_infinite/film/replay/ -run 'PowerPositions'`
vert ; le fichier de référence est produit et relu (nombre de cartes, tailles de polygones
plausibles : aucune entrée < 1 m² ni > 30 % de la carte).

### Étape 4 — Service et API

- [!] 4.1 `positionsDeForcePourIdentites` dans `internal/service/` (cascade module → map_id),
      port dans `internal/port/services.go`, DTO `replaydoc.PowerPosition` + projection
      `replayview`.
- [!] 4.2 Document de rejeu : champ `MapPowerPositions` rempli à la requête (`replay_service.go`,
      patron `MapObjectives`), jamais écrit dans l'artefact ; commentaire de champ dans
      `document.go` sur le modèle des voisins.
- [!] 4.3 Endpoint `GET /players/{slug}/tactical/{map_id}/power-positions` (handler sans
      logique métier, 404 typé `map_power_positions_not_available`).
- [!] 4.4 Dégradation : titre sans capability → champ omis / 404 typé ; test.
- [!] 4.5 `make openapi-gen`, `make generate-types` ; tests service (mock) et handler (httptest).

Gate 4 : `go test ./internal/service/ ./internal/api/... -run 'PowerPosition'` vert ;
`openapi-gen -check` propre ; types web frais (diff vide après `generate-types`).

### Étape 5 — Web, page rejeu

- [!] 5.1 `lib/replay/powerPositionsPaint.ts` : `drawPowerPositions(ctx, zones, projection, style)`
      (remplissage faible alpha + contour + libellé optionnel), testé.
- [!] 5.2 `features/match-replay/layers/powerPositionsLayer.ts` + quatrième calque statique dans
      `useReplayStaticLayers.ts`.
- [!] 5.3 `LAYER_ORDER` (`positions-de-force` après `zones-nommees`), `SceneToggles`,
      `SceneMatter`, `sceneLayers`, `buildScene` ; `sceneBinding.guard.test.ts` mis à jour.
- [!] 5.4 Préférence `replay-show-power-positions` (défaut ACTIVÉ, commentaire daté),
      `available` dans `useReplayDrawer.ts` et `ReplayCanvas.tsx`, bascule dans le groupe
      Terrain de `ReplaySettingsLayers.tsx`.
- [!] 5.5 Token `zone-power` dans `semantic-tokens.ts` (toutes palettes, justification datée) ;
      encre résolue dans `useReplayInks.ts`.
- [!] 5.6 i18n FR + EN (`i18nContract.ts`, `i18n.ts`) : libellé de la bascule, aide, légende.

Gate 5 : `make check-types` ; `npx vitest run src/features/match-replay src/lib/replay` vert ;
lint inter-features et lint couleurs propres ; capture d'écran du rejeu sur une carte du
catalogue et sur une carte absente (bascule indisponible).

### Étape 6 — Web, vue de match et page Tactique

- [!] 6.1 Vue de match : `MatchPositionsHeatmap.tsx` dessine le calque d'office, dans sa
      projection, sous la chaleur ; source = `doc.mapPowerPositions` du document de rejeu déjà
      chargé (ou la query de rejeu si le bloc ne la tient pas) ; rien si absent.
- [!] 6.2 Tactique : query `useTacticalMapPowerPositions(titleSlug, mapId)` + clé
      `tacticalMapPowerPositions` SANS joueur dans `lib/query/keys.ts` ; `TacticalPlanCard.tsx`
      peint le calque sous la chaleur dans `repereDuPlan`.
- [!] 6.3 Tests des deux surfaces (rendu conditionnel, projection).

Gate 6 : `make check-types` ; vitest des deux features vert ; captures d'écran des deux pages.

### Étape 7 — Clôture

- [!] 7.1 Gates complets : `cd apps/go-api && go test ./...` (timeout 30 min), `make go-api-lint`,
      `make check-types`, `make test-web`, `openapi-gen -check`, ratchet couleurs.
- [!] 7.2 Revue adversariale UNIQUE sur le diff cumulé (skill `adversarial-review`), une ronde
      de corrections.
- [!] 7.3 Docs : entrée `.ai/thought_log.md` ; `docs/adr/0036-power-positions-derived.md`
      (EN, court : dérivées, jamais devinées ; oracle ; plancher ; pas de calque sans preuve) ;
      ligne dans la table des ADR de `CLAUDE.md` ; table des données de référence du skill
      `db-schema` si elle liste les fichiers `reference/` ; `.ai/BACKLOG.md` (voie géométrique
      pour les cartes sans corpus, vignettes tactiques, Halo 5 depuis ses positions API).
- [!] 7.4 Mémoire agent : fait durable sur le chantier (worktree, verdict, ce qui a été écarté).
- [!] 7.5 Demander à l'utilisateur avant tout commit et avant la fusion dans `feat/v75`.

## Découvertes hors périmètre (à consigner, ne pas traiter)

1. **[2026-09-20, étape 1] Les positions de kill de « Live Fire - Ranked » sont décalées et
   étirées en X.** Barycentre du nuage distant de **9,88 m** de celui de « Live Fire », alors
   que les dix autres cartes à deux variantes tiennent sous 2,6 m (huit sous 1,2 m). Détail :
   x médian 21,3 m contre 8,2 m, y médian identique (32,7 m des deux côtés), x maximal 36,9 m
   contre 27,3 m ; **13 des 30 matchs « - Ranked » débordent** de l'emprise de la carte de
   base. Les ARTEFACTS DE REJEU des deux variantes s'accordent (x max 27,3 et 27,4 m) et les
   deux partagent un seul fond publié (`sgh_interlock.json`, qui déclare les deux noms) :
   c'est donc un défaut de décodage des positions de kill (piste : la loi des largeurs d'axe
   de la déquantification), pas une différence de géométrie. **Conséquence retenue pour
   l'étape 2 : les chiffres de Live Fire ne comptent pas comme preuve.** Un contrôle générique
   a été posé dans l'outil (`controle_variantes.go`, seuil 2 m) ; rien n'est corrigé ni écarté.
2. **[2026-09-20, étape 1] `match_registry.mode_category` est périmée.** Elle vaut « Other »
   sur 126 matchs de « Ranked:Strongholds on Live Fire », 116 de « Ranked:Oddball on
   Recharge », etc. La catégorie se recalcule à la lecture par
   `halo_infinite.InferModeCategoryFromPairName` — ce que fait déjà l'app —, mais la colonne
   stockée reste une source fausse offerte à quiconque la lira.
3. **[2026-09-20, étape 1] Le flood-fill de composantes connexes existe en deux exemplaires.**
   `tactical.composantes` (privé, 8-connexité) et `powerpos.Composantes` (public,
   4-connexité). Les deux voisinages sont volontairement différents et justifiés dans leurs
   en-têtes. C'est la 2e copie : à la 3e, la règle du dépôt impose de centraliser avec un
   garde-rail.
4. **[2026-09-20, étape 1] Argyle est absente du corpus** (aucun match), alors qu'elle est au
   circuit HCS et dans l'oracle. Aucune mesure empirique n'est possible sur cette carte tant
   qu'elle n'est pas jouée.
5. **[2026-09-20, étape 1] `lattice` n'a pas de fond de carte publié** (`map_backgrounds`),
   alors qu'elle compte 28 matchs avec positions et 3 artefacts. Toute surface qui dessine un
   calque sur un fond sera muette sur cette carte. Cuisson à demander à `cmd/mapfond-build`
   (exige le jeu installé).
6. **[2026-09-20, étape 1] `origin` et `fortress` n'ont aucune zone nommée** au catalogue, ni
   par module ni par `map_id` — les planches de contrôle sortent sans repères. Extraction
   Forge à compléter (`cmd/mapcallouts-build`).
7. **[2026-09-20, étape 2] Le « module » d'une carte Forge est celui de son CANEVAS, pas de
   la carte.** `map_quant_bounds.json` rend `fo11_blank` pour Solitude ET pour Empyrean,
   `fo09_academy` pour Fortress. La cascade du service s'en sort par accident (le catalogue de
   zones n'attribue aucune zone à un canevas, donc le lookup par module échoue et le `map_id`
   tranche), mais quiconque lira ce champ comme une identité de carte se trompera. Rien n'est
   corrigé ; `positions.json` publie le module ET tous les `map_id` du corpus pour que le
   rattachement ne se devine pas.
8. **[2026-09-20, étape 2] Les polygones des zones nommées sont des emprises 2D, et les
   étages se superposent en vue de dessus.** Sur Recharge, une position dont la zone dominante
   par recouvrement est `Batteries` (rez-de-chaussée) est appariée à `Attic` (au-dessus) parce
   que le barycentre tombe dans les deux. Le catalogue porte pourtant `z_bottom` / `z_top` par
   zone, et `powerpos.Position` ne porte AUCUN Z (la sélection est purement XY). Tout
   appariement ou nommage par zone — y compris le nommage du catalogue de l'étape 3 — est donc
   ambigu sur les cartes à étages tant que le Z n'entre pas dans la comparaison.
9. **[2026-09-20, étape 2] La précision contre l'oracle est presque toujours 1,00, et ne
   discrimine rien.** Les 67 lignes `faible` de l'oracle pavent les cartes (bases, couloirs,
   places) : une position tombe forcément dans l'une d'elles. Sur 6 cartes jugées, 4 rendent
   une précision de 1,00 avec un rappel de 0,00 à 0,33. La précision telle que le plan la
   définit n'est pas un garde-fou ; seuls le rappel et les contre-exemples le sont.
10. **[2026-09-20, 2bis.B] Le rang par match n'existe que pour les matchs classés, et cinq
    des douze cartes mesurées n'en ont AUCUN** (illusion, bazaar, forbidden, fortress,
    empyrean : 0 % des kills avec un tueur au rang connu ; 28 % sur l'ensemble). Toute
    parade « robuste au niveau » fondée sur le rang est muette sur ces cartes — le poids
    neutre du rang inconnu y rend le score v2 égal à un score sans rang. Une source de
    niveau pour les matchs sociaux (MMR d'équipe de `match_participants`, à défaut de mieux)
    serait à évaluer à part.
11. **[2026-09-20, 2bis.B] La formule v1 ne pesait pas ce qu'elle disait.** L'axe portée,
    normalisé entre p50 et p90 de la carte, SATURE à 0 ou 1 sur la moitié des disques
    (étalement p90 − p10 : 0,92), l'avantage rétréci ne s'étale que de 0,17 : à poids 0,10
    la portée contribuait autant au score v1 que l'avantage à 0,50. `ReglageV1` reste tel
    quel (preuve datée) ; la v2 déduit ses poids de l'étalement mesuré.
12. **[2026-09-20, 2bis.B] Couverture et abri angulaires sont anticorrélés** (−0,54 à −0,73
    sur les cartes de calibrage, −0,34 à −0,73 sur les douze) : pris séparément, chacun
    mesure surtout l'OUVERTURE du lieu. Toute pondération inégale des deux réintroduit
    cette composante ; seule leur somme à poids égaux isole l'asymétrie.
13. **[2026-09-20, 2bis.B] `cmd/mapgeo-build` (agent 2bis.C, en vol) ne compile pas** au
    moment du gate de 2bis.B (`function main is undeclared`) : `go build ./...` est rouge
    pour cette seule raison, le gate a été rendu sur `./internal/... ./cmd/mappower-build/`.
    À revérifier à la clôture de 2bis.C. → **Revérifié à la clôture de 2bis.C (2026-09-20) :
    `go build ./...` vert.**
14. **[2026-09-20, 2bis.D] Le verdict v1 n'était plus rejouable depuis 2bis.B.** Le lecteur
    CSV du harnais (`litCSVCellules`) exigeait exactement `len(colonnesCSV)` colonnes ; la
    passe v2 a porté l'en-tête à 25, et le diagnostic v1 (§6 du verdict) ne relisait plus ses
    propres CSV à 13 colonnes (« en-tête inattendu »). Corrigé dans l'extraction du commun
    (13 ou 25 colonnes, en-tête vérifié nom à nom) ; verdict v1 rejoué octet pour octet
    identique au document du matin.
15. **[2026-09-20, 2bis.D] Le témoin géographique tombe sur des zones hors aire jouable.**
    Sur Live Fire, `Landing Pad` est à 82-93 m des fortes (la carte fait ~40 m) : le catalogue
    porte des zones décoratives, et « la plus éloignée » les choisit. Et `Hallway → Nest`
    (50 m) remplace une forte par une zone que la v2 colore réellement (`live_fire__arene__1`,
    nest 100 %) : témoin 0,50 au-dessus du réel 0,33 sur cette carte. Le témoin reste
    concluant en moyenne (0,37 → 0,06) ; une exclusion des zones hors emprise praticable le
    renforcerait — non fait.
16. **[2026-09-20, 2bis.D] L'ambiguïté d'étage (découverte 8) porte le rappel d'Aquarius.**
    `aquarius__arene__1` a pour zone dominante `hydro` à 100 % et retrouve À LA FOIS Hydro et
    Top Mid (barycentre dans les deux emprises superposées) : le rappel 1,00 tient à une seule
    position de 14 m². Même mécanique sur Recharge (`recharge__arene__1`, long hall 100 %,
    apparié à Attic ; `recharge__arene__2`, batteries 53 %, apparié à Attic). Tant que le Z
    n'entre pas dans l'appariement, un rappel sur une carte à étages se lit avec cette réserve.
17. **[2026-09-20, 2bis.C] Le maillage de rendu n'est pas le maillage de collision, et le
    bloc de collision n'est pas décodé.** Les artefacts de rejeu de Recharge (`3923bede`,
    `e85d7bad`, retrouvés par leurs bornes) portent des joueurs à 0,5 / 1,0 / 1,5 m dans
    l'escalier de la fosse (x 13–16,5, y 0,5–3,5) là où des corniches de rendu à 1,6–2,0 m
    au-dessus des marches rejetaient le sol à 1,9 m de hauteur libre. Le bloc `instanced
    physics instances` du sbsp (handoff §9.2, 1,4 Mo sur catalyst) est la source juste pour
    tout ce qui touche à la praticabilité ; personne ne l'a décodé. Palliatif retenu : hauteur
    libre 1,5 m, voisinage à 2 cellules, germes = ancres + socles, élagage « sans retour ».
    Piège résiduel : le niveau −3 m de la halle de Recharge (1 147 nœuds) se rejoint en
    tombant et ne se quitte pas dans le graphe.
18. **[2026-09-20, 2bis.C] Le dépôt hors ligne `.ai/re_dump/navmesh` n'existe pas sur ce
    poste** (gitignoré, jamais rapatrié ici) : même les cartes Forge n'ont pas de maillage de
    navigation local. Sans effet sur ce chantier (cartes natives), à savoir avant toute cuisson
    Forge par `mapfond-build`.
19. **[2026-09-20, 2bis.C] Le `sddt` de `sgh_interlock` (Live Fire, variante `any/`) porte
    0 plan de frontière** : la coquille de mort ne s'applique pas, l'arène n'est pas bornée
    (4 932 candidats rejetés par la seule hauteur libre). Le fond de carte a le même trou de
    règle sur cette carte.
20. **[2026-09-20, 2bis.C] `reference/map_positions_jouees.json` ne couvre que des cartes
    Forge** (keyé par map_id, 0 entrée Recharge / Aquarius / Streets…) : aucune position jouée
    native n'est disponible comme oracle du sol, et les artefacts de rejeu ne portent pas le
    nom de la carte (il faut passer par `match_registry` ou par les bornes).
21. **[2026-09-20, 2bis.C] Le peintre de planches PNG (calage sidecar, cellules à leur
    emprise, contours de zones, Bresenham) est en DEUXIÈME copie** : `cmd/mappower-build/png.go`
    et `cmd/mapgeo-build/png.go`. À la troisième, la règle du dépôt impose un helper partagé
    avec garde-rail (candidat : un paquet `internal/analysis/planche` ou voisin de `himap.FondPNG`).
22. **[2026-09-20, 2bis.D] Le critère de verdict n'a pas de borne de taille, et un balayage
    qui l'optimise achète du rappel avec de la surface.** Le gagnant du balayage de fusion
    (croissance p80) produit sur Bazaar une position de 884 cellules / 500 m² qui « retrouve »
    5 fortes sur 5 ; 17 positions sur 32 dépassent 60 cellules. La sélection v2 (réutilisée par
    la fusion) ne borne pas la croissance ; la sélection géométrique la borne à 4 m de marche.
    Tout nouveau balayage doit mettre l'aire maximale (ou le nombre de couloirs) dans son
    critère, et toute règle de verdict devrait plafonner l'aire d'une position (D1 : un lieu de
    quelques m²). Non corrigé : réglage neuf après verdict (D12).
23. **[2026-09-20, 2bis.D] Le témoin géographique dégénère sur Recharge** : 5 des 9 fortes sont
    remplacées par la même zone (Hydro, la plus éloignée ET la plus grande, 488 m²), et le
    témoin (0,40) dépasse le réel (0,22) en géométrie seule. Complète la découverte 15 : un
    témoin qui impose une bijection ou exclut les zones de plus de N m² le corrigerait.
24. **[2026-09-20, 2bis.D] La règle « 30 % de l'aire de la zone » est inatteignable sur les
    grandes zones** (Hydro 488 m², Market 436 m², Blue Base 218 m²) : une position de 7 à 30 m²
    posée DANS la zone ne l'apparie que par son barycentre. Cinq « retenue ailleurs » du
    diagnostic géométrique sont des appariements manqués par la règle, pas par l'algorithme.
25. **[2026-09-20, 2bis.D] Empyrean et Solitude ne sont pas cuisables par `mapgeo-build`** :
    cartes Forge dont le « module » est le canevas vide `fo11_blank` (découverte 7) ; les
    objets Forge vivent dans l'asset UGC, que la chaîne des `.module` ne lit pas. Toute voie
    géométrique sur une carte Forge exige une autre source (navmesh UGC, découverte 18).
26. **[2026-09-20, 2bis.D] La zone `Canal` de Live Fire se superpose au `Nest` en vue de
    dessus** : une position de 23 m² à 45 % dans Nest a son barycentre dans Canal (piège pur
    de la fusion). Même mécanique que la découverte 8 ; tant que le Z n'entre pas dans
    l'appariement, un piège « chute » sous un balcon compte contre le balcon.
27. **[2026-09-20, 2bis.D] Le plancher absolu de la fusion (0,70) mord sur deux cartes de
    validation** (Forbidden 0,691, Bazaar 0,689 → amorce relevée à 0,70) sans changer le
    nombre de positions. Posé avant tout regard sur la validation, selon la recette v2 ; noté
    parce qu'un score normalisé par carte rend le plancher plus sensible qu'un score brut.

## Protocole de reprise

Lire ce fichier (statuts des items), puis le dernier document produit sous
`.ai/V7.5/positions_de_force/`, puis `git log --oneline -10` sur `wt/power-positions`.
Reprendre au premier item non statué de la première étape non close. Ne jamais rouvrir une
étape close pour retoucher un seuil : une retouche de seuil après l'étape 2 invalide le verdict
et se traite comme une nouvelle étape 2.

## Journal d'avancement

| Date | Étape | Statut | Note |
|---|---|---|---|
| 2026-09-20 | Plan | écrit | worktree créé, inventaire du dépôt fait, décisions D1-D8 tranchées |
| 2026-09-20 | Étape 0 | CLOSE — gate 0 validé par le pilote (réserves en §7 de l oracle : Center Bridge rétrogradé, contre-exemples ajoutés au verdict, raison « arme » rapportée à part) | `ORACLE_PRO_2026-09-20.md` écrit : 12 cartes, 84 positions (17 `forte` sur 6 cartes, 67 `faible`, 15 sans zone officielle), 26 contre-exemples, ~70 URLs. 0.1 et 0.2 en `[~]` (corpus = étape 1). Angle mort assumé : Reddit / YouTube / Liquipedia inaccessibles → l'oracle repose sur des guides grand public, et les 4 cartes de remake (Empyrean, Solitude, Interference, Banished Narrows) n'ont que des positions transférées depuis leur carte d'origine, donc `faible`. Relecture pilote requise avant l'étape 2 |
| 2026-09-20 | Étape 1 | CLOSE — gate 1 passé (build, vet, test, gofmt, golangci-lint 0 issue, archlint vert) | Paquet pur `internal/analysis/powerpos` (8 fichiers + 6 fichiers de test, aucun > 500 L) et outil `cmd/mappower-build` (`--recensement` / `--mesure`). Item 0.1 livré au passage : 9 110 matchs, 89 cartes recensées, Argyle absente, 92 artefacts de rejeu pour tout le titre. **12 cartes mesurées** (10 HCS + 2 témoins), CSV pour 12, planches de contrôle pour 11. Mesure structurante : le rapport de duel par cellule de 0,5 m est du BRUIT BINOMIAL (même dispersion au centième sur les 12 cartes) → le score se calcule sur un disque de 2 m. Formule figée : `0,50·avantage + 0,25·intensité + 0,15·hauteur + 0,10·portée`, occupation par équipe écartée (poids 0, motifs mesurés). Rend 0 à 5 positions par carte de 6 à 31 m² ; fortress et empyrean rendent zéro, comme prévu par D8. **Réserve : Live Fire contaminée** (décalage de 9,88 m entre ses deux variantes, découverte n°1) — ses chiffres ne comptent pas comme preuve à l'étape 2. 6 découvertes consignées. Aucun commit (demande utilisateur requise) |
| 2026-09-20 | Étape 2 | CLOSE — gate 2 passé (`go vet -tags research ./...` 0 issue, `go test -tags research -run Verdict` PASS, build/vet/gofmt/powerpos/archlint verts) — **VERDICT : NO-GO** | Test de recherche `TestVerdictOracle` (5 fichiers sous `cmd/mappower-build/`, tag `research`) + sortie `positions.json` ajoutée au mode `--mesure` (sérialisation pure, aucun seuil touché ; passe regénérée, chiffres identiques à l'étape 1). Document : `VERDICT_ORACLE_2026-09-20.md`, 7 sections, verdict en une ligne en tête. **1 carte sur les 4 exigées tient rappel ≥ 0,70 ET précision ≥ 0,60** (bazaar, et elle n'a qu'UNE zone `forte`) ; recharge 0,17, live fire 0,33 (hors preuve, contaminée), aquarius / streets / forbidden 0,00. **Témoin négatif NON CONCLUANT dans le bon sens** : rappel réel 0,25 contre 0,22 au témoin — l'appariement ne porte quasiment aucun signal (et le témoin lui-même est faible : liste triée par nom, le voisin alphabétique est souvent le voisin géographique — colonne ajoutée au document). **Contre-exemples : 1 piège PUR coloré hors Live Fire** (`recharge__arene__4` dans `Storage`) ; les 7 autres marques portent sur des zones que l'oracle décrit à la fois comme position et comme piège (§4 de l'oracle), comptées à part. Diagnostic zone par zone (§6) : sur 13 zones `forte` manquées, 6 « composante trop petite » (l'amas 4-connexe fait 2 à 10 cellules pour 12 exigées), 6 « score sous le seuil » (2 à moins de 0,01 du p90), 1 « non scorable ». Découverte structurante : la précision telle que définie ne discrimine rien (les 67 lignes `faible` pavent les cartes). 3 découvertes ajoutées (7 à 9). Aucun seuil modifié, aucun commit |
| 2026-09-20 | Étape 2bis.B | CLOSE — gate passé (gofmt vide, `go build ./internal/... ./cmd/mappower-build/`, `go vet` avec et sans `-tags research`, `go test ./internal/analysis/powerpos/` 40 tests verts, `golangci-lint` 0 issue, `./internal/archlint/` vert ; `go build ./...` rouge à cause du seul `cmd/mapgeo-build` de l'agent 2bis.C, découverte 13) | Reprise du travail d'un agent coupé en vol (signaux angulaires, pondération, fermeture, hystérésis écrits mais jamais compilés côté outil ni testés) : relu, un défaut corrigé (`construis`), 5 fichiers de tests neufs (angles, pondération, morphologie, hystérésis + fermeture + diagnostic, accumulateur v2), `Diagnostique` ajouté, outil branché (`--reglage v1/v2`, `--exclure-variantes`, `killer_xuid` joint en mémoire, CSV 25 colonnes, planches exposition / couverture, score étalé p10-p99 à l'affichage, `positions_v2.json`, 4 sections de rapport). **Rang** : `match_csrs_latest` seule source par (match, joueur) ; 28 % des kills couverts, 0 % sur 5 cartes. **Angles** : couverture / abri anticorrélés → poids égaux (asymétrie). **`ReglageV2` figé 14:06** sur recharge / aquarius / streets, règle « part / étalement », hystérésis p95 / p90 + fermeture r = 1 + 8-connexité + 10 cellules. 12 cartes : 35 positions (5/4/4/4/3/3/2/4/3/3/0/0), 5,0 à 58,6 m². Live Fire sans sa variante classée. `ReglageV1` intact, v1 rejouée à l'identique. 4 découvertes (10 à 13). Aucun commit |
| 2026-09-20 | Étape 2bis.A | CLOSE — gate accepté par le pilote avec réserves (§10 de l'oracle v2) : 28 fortes sur 8 cartes à ≥ 2 fortes ; calibrage Recharge / Aquarius / Streets, validation Live Fire / Bazaar / Forbidden / Empyrean / Solitude ; rappel rapporté à part pour les positions hauteur / lignes de vue |
| 2026-09-20 | Étape 2bis.D | CLOSE — bilan pilote : NO-GO des quatre lignées à règles égales (v1 0, v2 0, géo 1, fusion 1 carte) ; la géométrie retrouve 10/13 positions hauteur / vue avec des empreintes serrées, la fusion sans borne de taille colorie des salles (500 m²), la précision est plafonnée par la finesse de l'oracle. Aucune lignée en production telle quelle ; questionnaire user : géométrie comme moteur de candidats + validation humaine par carte (reco), fusion bornée, oracle seul, arrêt |
| 2026-09-20 | Étape 2bis.C | CLOSE — accepté par le pilote : tests `geo` rejoués verts, planches Recharge relues (sol dérivé couvre l'arène jouée, 7 positions en hauteur : halle nord-ouest, étage est, blocs sud) ; réserves : aucun navmesh natif (sol DÉRIVÉ du rendu, hauteur libre 1,5 m), V et E corrélés à 0,82-0,89 (rayons symétriques — E n'apporte presque rien de plus que V), niveau −3 m de la halle piégé dans le graphe ; réglage géométrique figé sans oracle |
| 2026-09-20 | Étape 2bis.B | CLOSE — accepté par le pilote : tests rejoués verts, planches relues (Recharge : 4 positions dont Platform / Control Room côté nord-ouest et deux blocs sud), réglage v2 figé 14:06 avec amendement transparent 14:20 AVANT tout verdict (admis : antérieur au jugement, écrit) ; réserve : rang muet sur 5 cartes, deux positions « couloir entier » (85-88 cellules) à trancher au verdict |
| 2026-09-20 | Étape 2bis.A | FAIT — pas de gate dédié à cet item seul (gate 2bis global reste ouvert, items B-D restants) | `ORACLE_PRO_V2_2026-09-20.md` écrit via navigateur réel (MCP `chrome-devtools`, ~35 fils Reddit lus par l'endpoint `.json` — bien plus fiable que la page traduite ; YouTube limité aux descriptions, sous-titres confirmés inaccessibles sur pièces ; liquipedia et halo.fandom.com confirmés bloqués même en navigateur réel ; x.com confirmé mur de connexion). **28 positions `forte`** (16 en v1 sur pièces → +12, dont 2 par correction d'audit Bazaar `Cafe`/`Den`) sur **8 cartes**. **Gate cible du plan (≥ 3 fortes sur ≥ 6 cartes) NON ATTEINT au sens strict : 4 cartes** (Recharge 9, Live Fire 3, Streets 3, Bazaar 5) ; clause alternative servie pour les 8 autres avec chiffrage (3 sans zone officielle au dépôt, 3 sous le seuil malgré recherche dédiée, 3 où le vocabulaire communautaire ne rattache à aucun nom du catalogue — Lattice en particulier, sortie le 5 août 2026, tactique pro documentée mais 0 zone nommée). 11 rattachements lexicaux de v1 audités (0 invalidé, 4 promus), 4 positions « arme seule » réauditées (aucune promue en position tenue), 1 contradiction Empyrean (« sword ») signalée non tranchée. Aucun fichier de code touché, aucun commit — décision de poursuivre 2bis.B/C/D ou de prescrire une passe Reddit supplémentaire (Aquarius/Forbidden les plus proches du seuil) revient au pilote |
| 2026-09-20 | Étape 2bis.D (première moitié) | FAIT — gate passé (`gofmt -l` vide, `go vet -tags research ./cmd/mappower-build/`, `go test -tags research -run VerdictV2` PASS, `golangci-lint run --build-tags research ./cmd/mappower-build/` 0 issue, build/vet sans tag OK, tous fichiers ≤ 500 L) — **VERDICT EMPIRIQUE V2 SEUL : NO-GO** ; l'item reste OUVERT pour la géométrie et la fusion | Harnais v2 en 4 fichiers `verdict_v2_*_research_test.go` (jugement, diagnostic/rejeu, rapport ×2), commun EXTRAIT du harnais v1 sans le dupliquer (lecteur d'oracle paramétré par ses marqueurs de sections et 6 ou 7 colonnes, remplaçant de zone en fonction, rappel et comptes par prédicat, lecteur CSV 13/25 colonnes, index des zones déplacé vers le fichier de géométrie) ; verdict v1 rejoué identique. Document `VERDICT_V2_2026-09-20.md` (10 sections) : 0/4 cartes de validation tiennent ; témoin géographique 0,37 → 0,06 ; 0 piège pur sur validation ; 18 fortes manquées diagnostiquées (9 sous la croissance, 5 retenues ailleurs, 2 sans amorce, 2 non scorables), rejeu prouvé fidèle 8/8 ; couloirs tranchés (streets = salle, recharge = à cheval sur deux zones) ; comparaison v1 → v2 à règles égales ; section « Géométrie et fusion » vide avec la commande. Découvertes 14-16. Aucun réglage touché, aucun commit |
| 2026-09-20 | Étape 2bis.C | FAIT — gate passé (`gofmt -l` vide ; `go vet` avec et sans `-tags gamefiles` ; `go test ./internal/analysis/powerpos/geo/` 14 tests verts ; `go test -tags gamefiles ./cmd/mapgeo-build/ -run TestCuissonRecharge` PASS en 4 s ; `golangci-lint run` 0 issue avec et sans tag ; `go test ./internal/archlint/` vert après ajout de `./cmd/mapgeo-build/` à `go-api-test-gamefiles` et passage du test à `testutil.RepoRoot` ; `go build ./...` vert — découverte 13 levée ; tous fichiers ≤ 300 L) | Reprise d'un agent coupé en vol (`cibles.go`, `redirections.go`, `triangles.go`, `cadre.go`, `doc.go` compilaient, rien n'était testé). **Inventaire** (`GEOMETRIE_2026-09-20.md` §1) : aucune carte native n'a de navmesh, `map_structure` = boîtes, seule source d'occlusion = triangles de rendu des `.module` (3,3 M Recharge … 26 M Forbidden) + coquille `sddt` (5/6). Paquet pur `powerpos/geo` (voxels 0,25 m Akenine-Möller, sol dérivé, graphe orienté marche/saut/chute, rayons Amanatides-Woo en parallèle, Dijkstra R et M, normalisation p5..p95, maxima locaux + croissance bornée, enveloppe v1) et outil `mapgeo-build` (CSV nœuds / rejets / coutures, 8 planches par carte, `positions_geo.json` aux clés du fichier empirique, `_rapport.md`). **Six cartes en 62 s**, 2 901–5 914 nœuds, rayons 0,1 s, voxelisation 60-80 % du coût. Recharge : H −1,5..+1,6 m, V p50 0,10, E p50 0,50, V/E corrélés 0,82-0,89 sur le calibrage. **`ReglageGeoV1` figé sans oracle** : 0,30·H + 0,20·V − 0,15·E + 0,20·R + 0,15·M, p90, max local 3 m, croissance 4 m, ≥ 12, ≤ 8 → 5 à 8 positions par carte (5–36 m²), toutes en hauteur sur Recharge. Neuf essais écartés consignés (§8). Découvertes 17-21 (rendu ≠ collision et bloc physique non décodé, dépôt navmesh absent, sddt Live Fire vide, positions jouées Forge seulement, peintre PNG 2e copie). Aucun commit |
| 2026-09-20 | Étape 2bis.D (seconde moitié) | FAIT — gate passé (`go build ./...` vert ; `go vet` avec et sans `-tags research` ; `go test ./internal/analysis/powerpos/...` 3 paquets verts dont `fusion` 5 tests ; `go test -tags research -run 'VerdictV2\|Fusion'` PASS ; `gofmt -l` vide ; `golangci-lint run` 0 issue avec et sans tag ; archlint vert ; fichiers ≤ 472 L) — **GÉOMÉTRIE SEULE : NO-GO ; FUSION : NO-GO** (strict et élargi) ; item 2bis.D CLOS | Alignement des grilles vérifié (écart 0) et gardé par le lecteur ; paquet pur `powerpos/fusion` ; mode `--fusion` sans base ; lecteur CSV extrait du harnais (une copie) ; balayage de 112 candidats sur le calibrage seul → `ReglageFusionV1` figé (0,6 / 0,4, absents 0,25 / 0, p90 / p80, plancher 0,70) ; harnais : lignée par nom de fichier, strict + élargi, quatre lignées côte à côte, planches de verdict, diagnostic géométrique par zone. Géo seule : Bazaar tient (1,00 / 0,88), Live Fire 0,67 / 0,71 (1 piège pur), Recharge 0,22 avec témoin 0,40 ; HV 10/13, AO 4/11. Fusion : Bazaar 1,00 / 1,00 par une position de 500 m², Live Fire 1,00 / 0,25 (1 piège pur), Forbidden 0,50 / 0,25 ; toutes les HV retrouvées, 4 AO manquées ; 17 positions sur 32 > 60 cellules. Empyrean / Solitude non cuisables (Forge). 6 découvertes (22-27). Aucun réglage figé touché, aucun commit |

## Second temps envisagé (demande utilisateur du 2026-09-20, HORS périmètre de ce chantier)

« Après, si on a un truc solide, ça nous fera une stat de match et de session sur le contrôle
des positions de force ; le nombre de positions par carte étant limité, réfléchir ensuite à un
rendu graphique. »

Ce que le chantier actuel doit LAISSER en place pour le rendre possible, sans le construire :
- des polygones monde par carte avec un identifiant STABLE par position (clé du catalogue), pour
  que la présence d'un joueur dans une position se calcule exactement comme la présence dans une
  colline de Roi de la colline (`hill_hold_ticks.go`, `zone_states_hill.go`, `TeamHold`) —
  même mécanique, autre forme ;
- la provenance et le nombre de matchs de chaque position, pour que la stat dise sur quoi elle
  repose.

Ce que la stat demandera (à planifier à part, une fois le catalogue livré et validé) :
- par match : temps de tenue par équipe et par position, temps contesté, temps vide ; par
  joueur : temps passé en position de force, frags faits depuis, morts subies dedans ; source =
  trajectoires du film (donc seulement les matchs avec artefact de rejeu) ;
- par session : agrégat des matchs à artefact (part de contrôle moyenne, écart entre victoires
  et défaites) ; la page Tactique peut la porter par carte ;
- rendu : bande de tenue par position sur la frise du match (patron de la tenue des collines),
  carte de tenue par joueur sur la fiche match, un chiffre de contrôle en carte de session ;
  à dessiner APRÈS avoir vu combien de positions par carte le catalogue produit (2 à 6 attendues).
