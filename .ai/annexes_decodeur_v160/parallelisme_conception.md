# Parallélisme du décodeur de film : conception en items

Tête lue : `785c38e` (branche `feat/v1.6.0`). Tous les chemins sont relatifs au dépôt du décodeur. Aucun code n'a été écrit.
Films : `film01` à `film13`, comme dans `bench/films.tsv`.

Marques : **vérifié** = ligne relue ou graphe d'appels statique (voir § 8). **déduit** = raisonné sans preuve, avec ce qui le vérifierait.

## 0. Résumé

1. Le chantier 1 (pré-décodage des images-clés) est faisable tel que décidé. Les quatre résultats sont purs. Deux corrections au plan de `bench/recents/recents.md` : seuls deux résultats sur quatre ont déjà un mémo, et la refonte « largeurs MPP en paramètre » n'est **pas** nécessaire pour ce chantier.
2. Le chantier 2 (balayages d'après la marche) est faisable, mais plus lourd que prévu. La plus petite transformation n'est pas « largeurs en paramètre ». C'est un **contexte de voie** : une copie privée du profil de balayage, de l'observation, du rapport de replis et des diagnostics, fusionnée au retour.
3. Un sous-chantier du chantier 2 est presque gratuit : les lectures qui ne dépendent que du film (temps forts, index des joueurs, origine d'horloge, table des joueurs aux listages, scripts). Elles ne touchent pas le contexte et peuvent tourner **pendant la marche**.
4. Entrée publique retenue : `func (c *FilmContext) PoserFils(n int)`. Elle est nourrie par `replay.Options.Vitesse` (le porteur `OptionsVitesse` existe déjà).
5. 21 items, en 5 vagues (§ 6). Les trois risques principaux sont en § 7.

## 1. Chantier 1 — pré-décodage des images-clés

### 1.1 Les quatre résultats, un par un

| # | Fonction qui fait le travail | Appelant (ligne) | Entrées réelles | Lit / écrit dans `FilmContext` | Mémo existant |
|---|---|---|---|---|---|
| 1 | `marcherLaTable` (`internal/grammar/keyframe_world_marche.go:146`), via `MarcheDImageCle.Marcher` (`:84`) | `bipedSlotBand` (`offline_biped_band.go:248`), appelée par `FilmContext.BipedSlots` (`film_context.go:383`) ; aussi 14 autres parcours | octets du payload + **preuve** `PreuveDImageCle` = registre + contexte de lecture invariant dont le MPP vient de `EnTete().MPP` (`keyframe_world_preuve.go:79-93`) | rien pendant le calcul. Écrit le mémo `memoireDesMarches.parPayload` | oui : clé `(adresse du 1er octet, longueur)`, **avec** mutex (`keyframe_world_marche.go:55-59, 91-105`) |
| 2 | `TableDeDatums` → `EntetesDeTable` (`keyframe_datums.go:224-239`) | `declarerLImageCle` (`keyframe_liaison.go:99-102`), via `liaisonDesImagesCles.recevoir` (`:115-117`) et `preliminairesDesTrames.recevoir` (`marche_trames_preliminaires.go:38-44`), dans `distribuerLesImagesCles` (`distribuer.go:297`) | octets du payload seuls (constantes de paquet seulement) | rien. Écrit `recup.tables` | oui : `map[*byte]tableDeDatumsMemo`, **sans verrou** (`keyframe_datums.go:253-266`, `recuperations.go:24`) |
| 3 | `motsParRecord` (`keyframe_loadout.go:164-215`) | `canalDeLEtatCompletBipede.fenetresDerriereLaLecture` (`keyframe_etats_fenetre.go:71`), appelée par `ImageCle` (`keyframe_etats_scan.go:148-157`) | octets + records du paquet (ancre et type seulement) + jeux de mots : `known` (donné par `replay`, `loadoutFamilies()`, `replay/loadouts.go:51`) et `carrierMarkViews` | rien | **aucun** |
| 4 | `slotsDEntetesExacts` (`player_entities_entetes.go:39-56`) | `canalDesEquipes.ImageCle` (`player_teams.go:233`), appelée par `ScanPlayerTeams` (`:142`) | octets du payload seuls (`kfTableCap` et `managedPlayerTypeIndex` sont des constantes : `keyframe_world.go:28`, `player_teams.go:65`) | rien | **aucun** |

Pureté, sans détour :

- **Ancres (1)** : pure, mais pas « payload seul ». Elle dépend de la preuve, figée au **premier** appel de `FilmContext.MarcheDImageCle()` (`keyframe_world_preuve.go:101-103`). La preuve lit le registre, le MPP du format et la grammaire du film (`controle_corruption_du_film.go:40-51`). Pour un film sans section d'identification, cette grammaire retombe sur `c.bal` à l'instant du premier appel (`:44`, vérifié). Conséquence : le pré-passage doit partir **dans** ce premier appel, pas avant.
- **Datums (2)** et **entêtes exacts (4)** : purs, payload seul. Graphe d'appels statique depuis `TableDeDatums` et `slotsDEntetesExacts` : zéro accès à `FilmContext`, zéro écriture de variable de paquet, zéro écriture dans un type partagé (vérifié).
- **Mots (3)** : pure, mais à trois entrées : payload, records (ancre et type, donc le résultat des ancres), jeux de mots. Le contexte ne connaît pas `known` : le pré-passage doit se faire côté `ScanEtatsDesImagesCles`, qui le reçoit (`keyframe_etats_scan.go:64`).
- Les ancres, `WalkKeyframeFullState` (appelée par la preuve) et les quatre fonctions n'écrivent aucune variable de paquet et ne lisent aucun accesseur du contexte (graphe d'appels, vérifié). Les seuls `sync` du module de décodage sont le mutex de `memoireDesMarches`, `registryWarned` (`sync.Map`, `registry_fingerprint.go:119`) et les `sync.OnceValues` de `internal/source/zstd.go:67,72` ; `fallback.Compteur` a aussi un mutex (`internal/facts/fallback/compteur.go:43-48`). Le film (`internal/source/film.go:42-72`) est immuable après chargement.

**Lemme des records muets (mots).** Le parcours d'aujourd'hui passe à `motsParRecord` une copie des records où les records *admis* ont le type « non résolu » (`keyframe_etats_fenetre.go:48-56`). Dans `motsParRecord`, le rattachement d'un mot à un record se fait par `recordContaining(starts, …)` (`keyframe_loadout.go:198`), qui ne regarde que les débuts. Le type ne sert qu'à filtrer (`:199`). Donc le résultat d'un record non admis est identique que les autres soient muets ou non (vérifié par lecture). Le pré-passage peut calculer tous les records bipèdes ; le consommateur ne lit que les non-admis. Seul `Rec.Debut` et `Families` sont lus (`keyframe_etats_fenetre.go:73-80`). Preuve à écrire : item I4.

**Gaspillage connu** : aujourd'hui `motsParRecord` ne tourne que si un record de la payload est non admis (`:58-60`). Le pré-passage la calcule pour tous les payloads à bipèdes. Part des payloads à zéro non-admis : non établie (voir § 7).

### 1.2 L'état muté qui gêne (`PoserMPP`)

Sites d'écriture : `marche_images_cles.go:112-115`, `equipment_placements.go:155,176,180`, `equipment_creation_width.go:262`, `build_profile.go:115`, `contexte_de_carte.go:41`, `replay/build_ground_weapons.go:164`, `internal/facts/killsource/decode.go:401` (vérifié par recherche).

Qui lit `bal.MPP` : jamais en direct. Les lecteurs prennent une **copie** : `ProfilDeBalayage()` (`film_context.go:174-181`), `ContexteDeLecture()` (`:217-222`), `CadreDeBalayage()` (`:230-234`), `NouveauLecteur` (`:239-243`), et `profilDeLaVueA` (`controle_corruption_du_film.go:90-98`, copie directe de `c.bal`). Le lecteur de bits lit `b.p.MPP` (`default_state.go:459`) sur sa copie.

Réponse à la question du plan :

- **Chantier 1 : aucun des quatre résultats ne lit `bal.MPP`.** Les ancres utilisent le MPP du format via la preuve (`keyframe_world_preuve.go:89-91`), pas `c.bal`. Le pré-passage n'appelle jamais `nouvelleMarcheDesImagesCles`, donc jamais `PoserMPP`. **La refonte « largeurs en paramètre » n'est pas un prérequis du chantier 1.** Cela corrige `recents.md` § 1.5 (« points durs »).
- Les parcours qui lisent le corps des records (états complets, équipes, statborg, propriétés ti=13) posent le MPP du format le temps de leur boucle, puis le restaurent (`marche_images_cles.go:112-115`). Ils restent dans le chemin chaud. Ils comptent pour le chantier 2.

### 1.3 Où placer le pré-passage

Deux étages, un seul code de fan-out :

- **Étage A — premier accès aux images-clés** : dans `FilmContext.MarcheDImageCle()` à la création de `c.marches` (`keyframe_world_preuve.go:101-103`), quand `c.fils > 1`. Tâches : ancres, datums, entêtes exacts, pour chaque payload non vide de chaque chunk de `ChunkNumbers()`. Jointure. Fusion depuis le fil appelant : `c.marches.parPayload`, `c.recup.tables`, un nouveau `c.recup.entetes`. Le premier accès est le même instant qu'aujourd'hui, donc la preuve est figée dans le même état (vérifié par construction). Ce premier accès peut être `installWorldObjectPrecision` (formats 20 à 25, `mpp_declare.go:49-68` lit les 4 premières images-clés) : `PoserFils` doit donc être posé avant `poserProfilPuisCarte` (`replay/build_from_film.go:101`).
- **Étage B — début de `ScanEtatsDesImagesCles`** (`keyframe_etats_scan.go:64-69`) : tâches « mots par record » sur les ancres déjà chaudes, avec le `known` reçu. Résultat rangé **dans le canal**, pas dans le contexte : pas d'identité de carte à comparer, pas de mémo global.

**Forme préférée du plan : confirmée avec deux amendements.** (1) Seuls ancres et datums ont un mémo ; entêtes et mots en demandent un neuf. (2) Les mots ne peuvent pas entrer dans le même passage que les trois autres sans passer `known` au contexte ; le passage en deux étages est plus simple.

**killsource.** Le contexte est partagé quand l'hôte passe `killsource.Options.Contexte` et `replay.Options.Contexte` (`killsource/options.go:20-26`, `replay/options.go:441-443`, `replay/build_from_film.go:97-100`, `killsource/decode.go:359-362`). Alors killsource touche les images-clés le premier ; l'étage A y part, et la cuisson trouve tout chaud. Sans contexte partagé, chacun ouvre le sien et les ancres sont calculées deux fois (aujourd'hui aussi). L'hôte qui ouvre le contexte partagé est dans LevelUp, pas ici : non vérifié.

### 1.4 Entrée publique : une forme

`func (c *FilmContext) PoserFils(n int)` dans `internal/grammar`, fichier neuf `film_context_fils.go`. Elle est joignable par l'hôte sans cycle : `decfilm.FilmContext` est déjà un alias (`decfilm/decfilm.go:145`) et l'hôte importe `replay` pour `OptionsVitesse`.

- `PoserFils` range `max(1, n)`. Le plafond (`MaxFils`, `GOMAXPROCS`) reste dans `OptionsVitesse.FilsEffectifs()` (`replay/options_vitesse.go:28-46`). `grammar` n'importe pas `replay`.
- Le porteur côté hôte est `replay.Options.Vitesse OptionsVitesse` (ajout d'un champ). `BuildFromFilmAvecFaits` appelle `fc.PoserFils(opt.Vitesse.FilsEffectifs())` quand `Vitesse.Fils > 0`. Avec 0 il ne touche pas un contexte partagé déjà réglé par l'hôte.
- `killsource` n'a pas besoin d'un champ si l'hôte règle le contexte partagé. Pour un décodage killsource sans contexte partagé, un `Fils int` dans `killsource.Options` serait nécessaire : décision en § 7.
- Contrainte de taille (vérifié) : `internal/grammar/film_context.go` fait exactement 500 lignes, plafond de `internal/depottest/taille_test.go`. Le champ `fils` s'y ajoute en compactant un commentaire de la même taille. `replay/options.go` fait 494 lignes : le champ tient en 6 lignes.
- Compatibilité : ajouts seulement (une méthode, un champ). À vérifier à la fin contre `v1.5.0` avec `apidiff`.

### 1.5 Mémoire

Déduit des tailles de structures lues ; à mesurer avec `film-bench -profil-tas` sur `film10`.

- Nombre de payloads : 23 à 31 par film (414 à 558 paquets parcourus pour 18 parcours, `recents.md` § 1.3).
- Ancres : `KeyframeRec` = 4 entiers + 1 booléen, 40 octets (`keyframe_world.go:47-50`) ; environ 500 records par payload, soit 20 Kio par payload, 0,6 Mio au plus. Datums : 8 octets par entrée, environ 8 Kio par payload, 0,25 Mio. Ces deux mémos existent déjà aujourd'hui, au même volume : le pré-passage les remplit plus tôt, il ne les agrandit pas.
- Nouveau et retenu : entêtes exacts (au plus quelques dizaines de slots, environ 2 Kio par payload) et mots (au plus 4 Kio par payload). Moins de 0,2 Mio en tout.
- Transitoire : tampon de chaque fil (maps, tranches de records, filtre de mots de 8 Kio). Environ 0,1 à 0,3 Mio par fil, 2,4 Mio à 8 fils, plus le travail du ramasse-miettes.
- **Borne : +5 Mio au pic pour le plus gros film.** Le pic actuel est atteint en publication, après tous les balayages (`recents.md` § 1.4) ; le pré-passage ne l'élève pas.

### 1.6 Gain attendu (déduit)

Part du temps mural parallélisée : ancres 12,3 % + datums et liaison 14,5 % + mots 7,5 % + entêtes 6 % = environ 40 % (`recents.md` tables A et § 1.5). Aux rendements 90/80/70 % à 2/4/8 fils : environ -17 / -26 / -31 %, contre -18 / -28 / -33 % annoncés. L écart vient des entêtes mesurés sur deux films seulement et du gaspillage des mots. À remesurer en item I5.

## 2. Chantier 2 — balayages d'après la marche

### 2.1 Liste fermée et dépendances (vérifiées sur `replay/film_scan*.go`)

Appelés dans cet ordre par `scanFilmInputs` (`replay/build_from_film.go:258-317`). Parts de temps : `recents.md` § 3.1 (moyenne des cinq films récents).

| # | Balayage (appel) | Entrées autres que le film | Étapes observées | Part |
|---|---|---|---|---|
| 0 | marche des trames `ScanMarcheDesTramesAvec` (`film_scan_mouvement.go:112`) | — | (publiées en 8) | hors périmètre |
| 1 | pont d'identité `ScanPontDIdentite` (`film_scan_pont.go:45`) : téléportations, créations, positions, temps forts, index, origine | options de balayage, `RosterXUIDs` | `translocations`, puis `positions`, `bipedCreations`, `deaths`, `playerIndices`, `clockOrigin` | 10 % (index 6,6) |
| 2 | `ScanKnockbacks` (`film_scan.go:53`) | — | `knockbacks` | < 1 % |
| 3 | `ScanFireEvents` (`:67`) | — | `fire` | < 1 % |
| 4 | `ScanEtatsDesImagesCles` (`film_scan_images_cles.go:28`) | — | `loadouts` | 7,9 % |
| 5 | `ScanBirthLoadouts` (`film_scan_naissances.go:26`) | `BipedCreations` (de 1) | `birthLoadouts` | 0,3 % |
| 6 | `ScanZoomEvents` (`film_scan.go:351`) | — | `zoomEvents` | 0,6 % |
| 7 | `decodeFilmPlacements` (`:355`) | `&s.world` | `placements` | 4,2 % |
| 8 | `decodeFilmSpawnEvents` (`:362`) | — | `spawnEvents` | 0,6 % |
| 9 | `decodeFilmPadScans` (`:367`) | largeurs MPP **de 7** (`PlacementStats.Calibration.Widths`, `:366`) | `pads` | 0,4 % |
| 10 | `decodeFilmVehicleScan` (`:372`) | largeurs MPP de 7 | `vehicles` | faible |
| 11 | marque de portage `decodeFilmCarrierMarks` (`:383`) | `s.etats.Marques` (de 4) | `carrierMarks` | faible |
| 12 | propriétés ti=13 : zones, jauge (`film_scan_ti13.go:22-35`) | marche (`ProprietesGerees`) | `zoneReads`, `zoneKeyReads`, `flagGauge`, `flagGaugeWalk` | faible |
| 13 | `decodeFilmBombReads` (`film_scan.go:397`) | — | `bombReads` | faible |
| 14 | publication de la marche (`film_scan_mouvement.go:37-96`) : `ScanScriptsDuFilm`, `mortsDeVehicule` | `s.in.Vehicles.Scanned` (de 10) | 12 étapes | 0,2 % |
| 15 | `ScanHeldWeaponChanges` (`film_scan.go:139`) | `Loadouts` (4), `BirthLoadouts` (5), `BipedCreations` (1) | `heldWeaponChanges` | 0,6 % (avec 16-17) |
| 16 | `ScanBipedPickups` (`:156`) | — | `pickups` | |
| 17 | `ScanInventoryDeltas` (`:207`) | — | `inventory`, `inventoryDeltas` | |
| 18 | `ScanAbilityRanks` (`:232`) | — | `abilityRanks` | 1,6 % (avec 19-23) |
| 19 | `ScanEquipmentChanges` (`:249`) | `birthOfLives(Positions)` (de 1) | `equipmentChanges` | |
| 20 | `ScanEquipementPorte` (`film_scan_equipement_porte.go:14`) | `&s.world` | `carriedEquipment` | |
| 21 | `ScanCamoStates`, `ScanGrappleReads` (`film_scan.go:273,287`) | — | `camoStates`, `grappleReads` | |
| 22 | `ScanAbilityImpulses`, `ScanAbilityCharges` (`:309,325`) | — | `abilityImpulses`, `abilityCharges` | |
| 23 | `ScanGrenadeThrows`, `ScanProjectiles`, `ScanProjectileDetonations` (`film_scan_pont.go:99,107,116`) | `&s.world` | `grenades`, `projectiles`, `detonations` | 1 % |
| 24 | `poserLeFilDesMorts` (`:120`) | `pont.Morts` (de 1) | `deaths` | |
| 25 | `lireSiegesEtOccupants` (`:81-92`) : `ScanFilmPlayerTableAuxListages(film)`, `ScanPlayerTeams`, `ScanStatborgEntries` | — | `filmTable`, `playerTeams`, `rosterListings`, `statborgEntries` | 9,4 % |
| 26 | index des joueurs et origine (`:150-175`) | `pont.Index`, `pont.OrigineUS` | `playerIndices`, `clockOrigin` | |

**Graphe de `recents.md` § 3.2, vérifié :**

- positions → capacités : confirmé (19).
- états → portage et naissances → portage : confirmés (15).
- poses → socles et véhicules → publication de la marche : confirmés (7 → 9, 10 → 14).
- temps forts → index des joueurs : confirmé, **à l'intérieur** du pont (`pont_identite.go:116-124`).
- **états → naissances : non fondé.** `ScanBirthLoadouts` prend les créations de bipède, pas les loadouts (`film_scan_naissances.go:26`). Seul l'ordre du source les lie. La dépendance réelle est créations → naissances.
- Dépendances manquées : marque de portage ← états (11) ; publication de la marche ← véhicules (14) ; `playerIndices` ← `deaths` (26) ; ti=13 ← résultat de la marche (12).
- « Sans dépendance entre elles » (tables de joueurs et d'équipes, lunette, apparitions, grenades, détonations) : confirmé pour les arguments. Mais 7, 9, 10, 23 partagent les mémos `recup.pistes` et `recup.creations` du contexte (§ 2.2).

### 2.2 État partagé que ces balayages touchent

Écrivains paresseux du contexte (vérifié, graphe d'appels + lecture) :

| Champ | Écrit à | Chaud après une marche réussie ? |
|---|---|---|
| `chunks` | `film_chunks.go:129-135` | oui |
| `reg` | `film_context.go:431-442` (écrit aussi `diag`, `replis`) | oui |
| `prof`, `duFilm` | `film_context.go:354-356`, `controle_corruption_du_film.go:40-51` | oui |
| `slots` | `film_context.go:387-391` | oui (`canal_des_lectures_bipedes.go:229`) |
| `lay` | `film_context.go:408-411` | oui (catalogue) |
| `vies.*` | `generations_vivantes.go:256-268, 278-287, 313-319` | oui : `Clore` du canal des lectures bipèdes les force (`canal_des_lectures_bipedes.go:327-341`) |
| `recup.lectures`, `.ancres` | `canal_des_lectures_bipedes.go:339`, `ancres_bipedes.go:59-82` | oui |
| `recup.objets`, `.equipementPorte` | `objets_du_monde_lus.go:196`, `equipement_porte.go:168` | oui |
| `marches` | `keyframe_world_preuve.go:101-103` | oui |
| `recup.tables` | `keyframe_datums.go:260-264` | oui après I2 |
| `recup.pistes` | `pistes_du_monde.go:82-86` : le **premier** appel relève toutes les bandes | **non** : écrit par 7, 9, 10, 23 |
| `recup.creations` | `creations_du_monde.go:123-161` | **non** : écrit par 7, 9, 10 |
| `obs` (créé à la demande) | `film_context.go:164-168` | à créer avant les voies |
| `replis.comptes`, `.i0Note`, `.regNote` | `replis_du_film.go:145-180`, appelés partout | **non** : écrits par presque tous |
| `diag` | `film_context.go:299,441`, `world.go:148` | **non** |
| `bal` | `PoserMPP` et autres (§ 1.2) | **non** : écrit par 7, 9, 10 (poses, `defer` de restauration) et par chaque parcours de corps d'images-clés (4, 12, 25) |

Conséquences (vérifiées sauf mention) :

- Si la marche réussit, tous les mémos « de contexte » sont chauds avant le premier balayage. Si elle échoue, des balayages relancent une marche de repli (`lectures_bipedes.go:170-175`, `objets_du_monde_lus.go:202-216`, `equipement_porte.go:305-310`) : le chemin parallèle ne doit alors **pas** s'activer.
- Tous les lecteurs de bits construits depuis le contexte portent la **même** `*Observation` (`film_context.go:221`). Elle contient des compteurs entiers écrits par la lecture (`observateur.go:346-420`) : une course de données dès que deux voies lisent. Aucun compteur de cette observation n'est lu en production (commentaire `localisateur.go:98` ; recherche des lecteurs : seule la marche lit les siens, `movement_states.go:383`). **Déduit** : à confirmer en J4.
- Les balayages qui posent des crochets d'observation après la marche : le seul trouvé est `canalDesProprietesGerees` (`zone_state_scan_images_cles.go:57-74`), sur une observation propre au canal. Le graphe d'appels prouve l'absence de pose de crochet pour 2, 3, 6, 8, 23 (grenades, détonations) ; pour les autres il l'atteint seulement par des repli de marche (mémo nul). **Déduit** : garde dynamique en J4.
- `replay` : `fallback.Compteur` est déjà protégé par un mutex et ses sommes sont commutatives. `s.in`, `s.opt.observe`, les journaux et l'horloge d'étapes (`observe.go:99-125`) ne sont pas sûrs en parallèle.

### 2.3 Plus petite suite de transformations

1. **Chauffer** : en un fil, après la marche, forcer dans un ordre fixe les mémos paresseux restants (`Observation()` en premier). Un contrôle `EstChaud()` décide si les voies peuvent partir ; sinon, chemin d'aujourd'hui.
2. **Contexte de voie** : `Deriver()` copie le contexte (mémos chauds partagés en lecture) et lui donne un `bal` copié, une `obs` neuve, un `replis` et un `diag` vides. `Fusionner(voie)` ajoute les comptes (`ComptesDesReplis.Plus`, commutatif), fait un OU des drapeaux « une fois par film » (un test vérifie qu'aucun n'est compté deux fois), verse `diag` dans l'ordre des voies, et reprend `recup.pistes` et `recup.creations` de la voie qui les a écrits. Chaque mémo écrit pendant les balayages a **un seul** propriétaire de voie.
   *Pourquoi pas « largeurs en paramètre »* : il faudrait changer chaque site qui prend une copie de `bal` (voir § 1.2), plus les canaux qui lisent `fc.ContexteDeLecture()` au milieu d'une phase (`keyframe_etats_scan.go:163`, `zone_state_scan_images_cles.go:89`) mais d'autres avant la phase (`player_teams.go:166`). La copie privée par voie garde chacun exactement où il en est.
3. **Gel des mémos de lecture** : `tableDeDatumsDe` ne mémorise plus quand le contexte est gelé (une absence se recalcule sans écrire). Plus de course sans verrou sur le chemin de lecture.
4. **Scinder lire et publier** dans `replay/film_scan*.go` : `lireX()` (calcul, peut partir en voie) et `publierX(résultat)` (`s.in`, journaux, compteur de replis, `observe`), toujours sur le fil appelant **dans l'ordre d'aujourd'hui**. L'ordre des étapes publiées ne change donc pas. Le garde `replay/observe_test.go` (marcheur d'AST de `scanFilmInputs`) reste vert si les fichiers neufs entrent dans `fichiersDuBalayage`. Les lignes de journal émises *dans* un calcul (par exemple `build_ground_weapons.go:103,109`) peuvent s'entremêler : décision en § 7.
5. **Voies** (après les transformations 1 à 4) :
   - voie **monde** : 6, 7, 8, 9, 10, 23 (sur un contexte dérivé ; 7 → 9 → 10 en série) ;
   - voie **sièges et occupants** : `ScanPlayerTeams` + `ScanStatborgEntries` (contexte dérivé) ;
   - fil appelant : 1 → 2 → 3 → 4 → 5 → 15 … 22 dans l'ordre. Les balayages 16 à 22 sont des rejeux de mémos chauds, à 1,6 % : pas de voie.
6. **Démarrer avant la marche** (sans contexte) : `scanTempsForts(film)`, `scanPlayerIndices(film, roster)`, `ScanClockOrigin(film)`, `ScanFilmPlayerTableAuxListages(film)`, `ScanScriptsDuFilm(film)`. Graphe d'appels depuis chacune : aucun accès au contexte, aucune écriture de variable de paquet ni de type partagé (vérifié). Elles prennent le `*source.Film` immuable. Le pont les consomme via un champ optionnel `OptionsDuPont` ; les compteurs de replis qu'elles produisent (`pont_identite.go:118,123`) restent versés par le fil appelant, au même endroit. **Le roster vient d'une fonction de l'hôte** (`rosterOf(…, s.opt.RosterXUIDs)`, `film_scan_pont.go:47`) : elle doit être pure (déduit).
   Les lectures pures du payload d'image-clé (chantier 1) sont, elles, déjà « avant la marche ».

Gain (déduit, `recents.md` § 3.3 et § 1.4) : le sous-chantier « film seul » cache derrière la marche environ 8 à 10 % du temps total à 2 fils et plus (index 6,6 %, temps forts 1,0 %, listages, scripts), sans partager d'état. Les voies monde et sièges, après le chantier 1 (sièges 9,4 % → environ 4 % ; états 7,9 % → environ 2 %), ramènent le chemin critique des balayages à environ 10 % : gain d'environ 10 % de plus, pour un risque nettement plus haut. **À remesurer après les items I5 et J2 avant de décider J3 à J10.**

### 2.4 Ce qui reste risqué, et comment on le voit

| Risque | Détection |
|---|---|
| Course de données sur un mémo non listé | `-race` sur les tests de voies (job CI `film-race`, filtre à élargir) ; `TestChaqueMemoEstChaud` (J3) ; `TestAucunDistribuerApresLaMarche` (J4, compteur de test) |
| Un balayage laisse `bal` modifié | `TestChaqueBalayageRestitueLeProfil` (J4) : profil avant et après chaque balayage, à 1 fil |
| Ordre des étapes ou des compteurs de replis | marcheur `observe_test.go` ; film-bench `-etapes` (F4) ; colonne `empreinte_document` (le document porte `coverage.fallbacks`) |
| Panique dans un fil | l'assistant `executerEnParallele` relaie la panique sur le fil appelant. Aucun `recover()` dans ce module, et aucune goroutine aujourd'hui (vérifié par recherche) |
| Résultat différent selon l'ordonnancement | 3 passes à 8 fils par film, empreintes identiques ; `marche_determinisme_test.go` étendu à 1, 2, 4, 8 fils |

## 3. Corrections à `bench/recents/recents.md`

1. § 1.5 : « `bal.MPP` à passer en paramètre » n'est pas nécessaire au chantier 1.
2. § 1.5 : seuls deux des quatre résultats ont un mémo.
3. § 3.2 : l'arc « états des images-clés vers naissances » n'est pas une dépendance de données.
4. § 3.2 : « un seul primitif de synchronisation dans `internal/grammar` » est exact pour `grammar` (`keyframe_world_marche.go:57`). `source` (`OnceValues`) et `fallback` (mutex) en ont aussi.
5. § 3.2 : les crochets posés par les balayages d'après la marche se limitent à un seul canal, sur sa propre observation (à confirmer par garde dynamique).
6. § 3.2 : l'observation **partagée** du contexte (compteurs entiers) est une cause de course non citée.

## 4. Preuves communes à tous les items

- **PS1** : `GOWORK=off go vet ./...` puis `go test ./internal/profile/... ./internal/grammar/... ./internal/facts/... ./replay/... -count=1` (comme `.github/workflows/ci.yml`). Le test de plafond de lignes (`internal/depottest/taille_test.go`) doit rester vert.
- **PS2** : `CGO_ENABLED=1 go test -race -count=1 -run '<tests de l'item>' ./internal/grammar/` (et `./replay/`). Le job CI `film-race` filtre aujourd'hui `TestDeuxFilmsEnParallele` : l'élargir (item I5) aux préfixes `TestFils` et `TestVoie`.
- **PS3** : `film-bench` sur les 13 films à 1, 2, 4 et 8 fils. Les colonnes `empreinte_document` et `empreinte_faits` doivent être identiques entre elles **et** à la référence R0. R0 = `film-bench` de la tête `785c38e` à 1 fil, rangée avant le premier item. Trois passes à 8 fils pour attraper une course rare. Pour les tests de réplay, régler `runtime.GOMAXPROCS(8)` : `FilsEffectifs()` plafonne à `GOMAXPROCS`.
- **PS4** (mesure) : médiane, `pic_mio`, `-profil-tas`.
- **À 1 fil** : chaque item garde la forme `if fc.Fils() <= 1 { chemin d'aujourd'hui }`, et un test affirme qu'aucune goroutine n'est lancée. Le test `TestUneSeuleSourceDeGoroutines` (item F1) interdit toute instruction `go` hors des fichiers autorisés (`internal/grammar/fils.go`, et le fichier de fond de `replay` pour J2).
- Dans tous les items : aucune variable de paquet neuve ; pas de nouveau verrou sur un chemin de lecture ; pas de nouveau symbole exporté hors ceux nommés ici.

## 5. Les items

Taille : S = quelques heures, M = une séance, L = une séance chargée (à re-scinder si le premier essai dépasse).

### Fondations (sans comportement)

**F1 — assistant de fils** (S). Fichiers : `internal/grammar/fils.go` + test. Ajoute `executerEnParallele(n, nbTâches, fn)` : `n <= 1` ou une tâche : exécute dans l'appelant, sans goroutine ; sinon `min(n, nbTâches)` fils qui tirent l'indice suivant par un compteur atomique ; capture la première panique (par indice) et la rejoue sur l'appelant après jointure. Ne doit pas changer : tout (rien ne l'appelle). Preuve : `TestExecuterEnParallele_AUnFilAucuneGoroutine`, `_ChaqueTacheUneFois`, `_PaniqueRejoueeSurLAppelant`, `_ResultatsParIndice`, `TestUneSeuleSourceDeGoroutines`, `-race`. Dépend de : rien. Parallèle avec F2, J1, J4.

**F2 — porteur sur le contexte** (S). Fichiers : `internal/grammar/film_context_fils.go`, `film_context.go` (1 champ, commentaire compacté pour rester à 500 lignes). Ajoute `PoserFils(n)`, `Fils()`, défaut 1 dans les deux constructeurs. Ne doit pas changer : tout. Preuve : `TestFilsParDefautVautUn`, `TestPoserFilsBorneAuMoinsUn`, test de plafond de lignes. Dépend de : rien. Parallèle avec F1.

**F3 — entrée `replay.Options.Vitesse`** (S). Fichiers : `replay/options.go` (+ au plus 6 lignes), `replay/build_from_film.go` (après la ligne 99, **avant** `poserProfilPuisCarte`). `fc.PoserFils(opt.Vitesse.FilsEffectifs())` quand `Fils > 0`. Preuve : `TestVitesseArriveAuContexte`, `TestVitesseZeroNeTouchePasUnContexteDejaRegle`, `GOMAXPROCS` réglé dans le test. Dépend de : F2.

**F4 — `film-bench -fils` et `-etapes`** (M). Fichiers : `cmd/film-bench/main.go`, `mesure.go:170-171` (l'emplacement prévu), `releve.go`, tests (`main_test.go:23` traite `-fils` comme drapeau inconnu : à changer). `-fils N` pose `opt.Vitesse` et la colonne `fils`. `-etapes` écrit, hors chronomètre, la suite ordonnée `(étape, SHA-256 de la valeur)` de `Options.Observe` ; c'est la preuve de l'ordre des étapes. Ne doit pas changer : les colonnes existantes, le chronométrage. Preuve : tests de la commande ; deux passes `-fils 1` identiques. Dépend de : F3. Livre R0 et le gabarit de PS3.

### Chantier 1

**I1 — cadre du pré-passage et ancres** (M). Fichiers : `internal/grammar/keyframe_predecodage.go` (neuf), `keyframe_world_marche.go` (extraire `calculerLaMarche(pay)` = lignes 99-100, ajouter `poserSiAbsent` sous le mutex existant), `keyframe_world_preuve.go:101-103` (appel). Quand `c.fils > 1` : lister les payloads non vides, calculer en parallèle, joindre, ranger depuis le fil appelant. Ne doit pas changer : `Marcher` rend toujours une copie, mêmes `Stats`, même mémo à 1 fil, payloads vides jamais mémoïsés. Preuve : `TestFilsAncresIdentiquesATousLesFils` (bobines `ks_000d5950`, `ks_e5adf7b2` de `marche_determinisme_test.go` : toutes les `Marcher` égales à 1, 2, 4, 8, `Stats` compris), `TestFilsAUnFilAucuneGoroutine`, `TestFilsLaPreuveEstFigeeAuPremierAcces`, `-race`, PS3, PS4. Dépend de : F1, F2 (preuve de bout en bout : F3, F4).

**I2 — datums** (S). Fichiers : `keyframe_datums.go` (remplissage), `keyframe_predecodage.go` (une tâche de plus). Remplit `recup.tables` (clé `&pay[0]`, `bits = len(pay)`) depuis le fil appelant. Ne doit pas changer : `tableDeDatumsDe` à 1 fil. Preuve : `TestFilsDatumsIdentiques`, `-race`, PS3. Dépend de : I1. Parallèle avec I3, I4 (conflit attendu : une ligne dans la liste des tâches).

**I3 — entêtes exacts** (S). Fichiers : `recuperations.go` (champ `entetes map[clePayload]map[int]bool`), `player_teams.go:233` (consulter le mémo, sinon appel direct, **sans** mémoriser), `keyframe_predecodage.go`. Ne doit pas changer : `slotsDEntetesExacts` (clé de `balayagesNommes`, `internal/depottest/balayages_test.go:87` : ne pas la déplacer ni la renommer), `douterDe` (ne modifie pas la table : `player_entities.go:236-246`). Preuve : `TestFilsEntetesExactsIdentiques`, test de `ScanPlayerTeams` complet, `-race`, PS3. Dépend de : I1. Parallèle avec I2, I4.

**I4 — mots par record** (M). Fichiers : `internal/grammar/keyframe_etats_predecodage.go` (neuf), `keyframe_etats_scan.go:88-100`, `keyframe_etats_fenetre.go:71`. Étage B : calcul parallèle de `motsParRecord` sur les records bipèdes **non muets** de chaque payload, avec les jeux de l'appel (un seul jeu sous `armesSeules`), rangés dans le canal. Ne doit pas changer : les compteurs `FenetresArmes/Inventaire/Marque` (comptés sur `nonAdmis`, `keyframe_etats_fenetre.go:58-67`), `RecordRecupere`. Preuve : `TestMotsParRecordIndependantDuMuet` (lemme du § 1.1 : mêmes familles pour un record non admis, que les autres soient muets ou non, sur les deux bobines), `TestFilsMotsIdentiques`, `-race`, PS3. Dépend de : F1, F2. Parallèle avec I2, I3 (sait lire les ancres chaudes ou non).

**I5 — mesure du chantier 1 et CI** (S). Fichiers : `bench/recents/` (nouveau relevé), `.github/workflows/ci.yml` (filtre `film-race`). Contenu : temps médians, pic, empreintes à 1/2/4/8 fils pour les 13 films ; part des payloads à zéro non-admis ; gain réel. Décide si la suite (J3 à J10) vaut son risque. Dépend de : I1 à I4, F4.

### Chantier 2a — lectures qui ne dépendent que du film

**J1 — extraire la lecture « film seul » du pont** (M). Fichiers : `internal/grammar/pont_identite.go:101-127`, + type `FaitsDuFilmSeul` et champ optionnel `OptionsDuPont.FaitsDuFilm`. Une fonction exportée côté `grammar` `LireLesFaitsDuFilmSeul(film, roster)` rend temps forts, médailles, index, origine, et les deux comptes de replis (jamais versés par elle). `lire` l'appelle à la même place quand le champ est nul. Ne doit pas changer : ordre et valeur des versements de replis, erreurs. Preuve : `TestPontAvecFaitsPrecalculesEgalAuPontSeul` (bobines), PS1. Dépend de : rien. Parallèle avec F1, F2, J4.

**J2 — tâche de fond de `replay`** (M). Fichiers : `replay/film_scan_fond.go` (neuf, `go` autorisé), `replay/film_scan.go` (départ avant `lireLaMarcheDesTrames`, ligne 307), `film_scan_pont.go` (consommation), `film_scan_mouvement.go:70` (scripts), `replay/observe_test.go` (`fichiersDuBalayage`). À `Fils > 1` : une voie sans contexte calcule `FaitsDuFilmSeul`, la table aux listages, les scripts ; chaque consommateur attend sa valeur à l'endroit où il la lit aujourd'hui. Ne doit pas changer : ordre des étapes, valeurs, API de `PortagesAuSync` qui appelle `lireSiegesEtOccupants` (`film_scan_pont.go:81`). Preuve : `TestVoieFondIdentiqueATousLesFils`, `TestObserveEtapesBuildFromFilm`, `-race`, PS3, PS4 (gain attendu : -8 à -10 %). Dépend de : F1, F3, J1.

### Chantier 2b — voies sur contexte dérivé (à décider après I5 et J2)

**J3 — contexte de voie** (M). Fichiers : `internal/grammar/film_context_voie.go` (neuf). `Chauffer()`, `EstChaud()`, `Deriver()`, `Fusionner()`. Rien ne les appelle encore. Preuve : `TestDeriverNePartagePasLeProfil`, `TestFusionnerSommeLesReplis`, `TestFusionnerNeCompteJamaisDeuxFoisUnReplisParFilm`, `TestChaufferRendTousLesMemosLus` (réflexion sur les drapeaux de § 2.2), `TestVoieNEcritPasLeParent` (deux voies qui appellent `PoserMPP`, `-race`). Dépend de : F2. Parallèle avec J4, I1.

**J4 — gardes de caractérisation, à 1 fil** (M). Fichiers : tests seulement (corpus local ; saute sans données). `TestChaqueBalayageRestitueLeProfil`, `TestAucunDistribuerApresLaMarche`, `TestAucunCrochetNEstPoseSurLContexteApresLaMarche`, `TestObservationDuContexteNEstLueParPersonne`. Ils transforment les « déduit » de § 2.2 en « vérifié ». Dépend de : rien. À faire tôt : s'ils échouent, ils changent le plan.

**J5 — gel des mémos de lecture** (S). Fichiers : `keyframe_datums.go`, `recuperations.go`. Un drapeau `figee` posé par `Chauffer()` : `tableDeDatumsDe` recalcule sans écrire en cas d'absence. Ne doit pas changer : comportement quand le contexte n'est pas gelé. Preuve : `TestTableDeDatumsGeleNeEcritPas`, `-race`. Dépend de : I2, J3.

**J6a / J6b / J6c — scinder lire et publier** (M chacun, **en série à 1 fil, sorties identiques**). J6a : monde (6 à 10, 23). J6b : sièges et occupants (25) et pont (23 à 26). J6c : chaîne des positions (1 à 5, 15 à 22). Fichiers : `replay/film_scan*.go`, `observe_test.go`. Aucune goroutine. Ne doit pas changer : ordre des étapes, textes et ordre des journaux (comparés avant/après). Preuve : `TestObserveEtapesBuildFromFilm`, PS3 à 1 fil identique à R0, `-etapes` identique. Dépend de : rien d'autre que F4 pour la preuve. J6a, J6b, J6c touchent des fonctions différentes mais le même `film_scan.go` : les coder dans des worktrees séparés, les fusionner dans l'ordre a-b-c.

**J7 — orchestrateur de voies** (L). Fichiers : `replay/film_scan_voies.go` (neuf), `film_scan.go`. À `Fils > 1` et `EstChaud()` et marche sans erreur : lance les voies, exécute la chaîne principale, joint chaque résultat là où `publier` en a besoin. Sinon : chemin d'aujourd'hui. Relaye les paniques. Ne doit pas changer : ordre publié, valeurs. Preuve : `TestVoieOrdreDesEtapesIdentique`, `TestVoieRepliAUnFilSiMarcheEnErreur`, `TestVoieAUnFilAucuneGoroutine`, `-race`, PS3. Dépend de : F1, J3, J5, J6a à J6c.

**J8 — voie monde** (M). Contexte dérivé ; 6 → 7 → 8 → 9 → 10 → 23 ; la voie est propriétaire de `recup.pistes` et `recup.creations`. Preuve : `TestVoieMondeIdentique`, `TestVoieMondeCompteLesReplisUneSeuleFois` (`PistesDuMondeApresLaMarche`, `CreationsDuMondeApresLaMarche`, `LargeursMPPCalibrees`), `-race`, PS3, PS4. Dépend de : J7. Parallèle avec J9 (fichiers de voie distincts ; le registre des voies dans `film_scan_voies.go` change d'une ligne chacun).

**J9 — voie sièges et occupants** (M). Contexte dérivé ; `ScanPlayerTeams` puis `ScanStatborgEntries`. Preuve : `TestVoieSiegesIdentique`, `-race`, PS3, PS4. Dépend de : J7, I3. Parallèle avec J8.

**J10 — clôture** (S). Relevé complet 1/2/4/8 fils, pic mémoire, retrait des essais, documentation (`bench/README.md`, README), `apidiff` contre `v1.5.0` (ajouts seulement). Dépend de : tout le reste.

## 6. Ordre et parallélisme

```
Vague 0 : F1 || F2 || J1 || J4
Vague 1 : F3 -> F4 ; I1 ; J3 (parallèles entre eux, F3/F4 en série)
Vague 2 : I2 || I3 || I4 ; J2
Vague 3 : I5 (mesure, décision) ; J5 ; J6a -> J6b -> J6c
Vague 4 : J7 ; puis J8 || J9
Vague 5 : J10
```

Fichiers partagés à surveiller : `keyframe_predecodage.go` (I1, I2, I3 : une ligne), `film_scan.go` (J2, J6a-c, J7), `observe_test.go` (J2, J6, J7), `film_context.go` (F2 seul). Un worktree par item qui écrit.

Chemin court : F1, F2, F3, F4, I1 à I5 donnent le chantier 1 complet (environ -17 à -31 %). J1 et J2 ajoutent -8 à -10 % pour un risque faible.

## 7. Ce que je n'ai pas pu établir, et ce qui ferait déconseiller

Non établi (rien n'a été décodé, par consigne) :

1. L'ensemble exact des écritures du contexte par balayage, **à chaud**. Le graphe d'appels sur-approxime : il inclut les replis de marche. Les gardes de J4 le mesureront.
2. La part des payloads sans non-admis (gaspillage des mots), la part de `lireSiegesEtOccupants` entre listages, équipes et statborg, et tous les gains. Seul le banc le dit.
3. Si l'hôte partage le contexte entre killsource et la cuisson, et s'il a un `recover()` autour d'un film (le module n'en a aucun). Si oui, le relais de panique de F1 est indispensable.
4. La sûreté de `ReplisALaConsultation` (`internal/facts/objectives`) et celle de `rosterOf` en parallèle (J2, J8).
5. La mémoire de J7 à J9 : `recents.md` annonce +50 à +150 Mio à 4 fils, non mesuré.

Décisions à prendre :

- Ordre des lignes de journal dans les voies : ordre libre, ou tampon et réémission dans l'ordre (plus de code).
- `killsource.Options.Fils` : ajouter, ou régler le contexte partagé côté hôte seulement.
- Durées par étape de `film-bench` : non significatives à plus d'un fil. À écrire dans `bench/README.md`.

Ce qui ferait déconseiller :

- **Chantier 1** : rien de sérieux. À abandonner seulement si I1 montre que la preuve n'est pas figée au même instant (le test `TestFilsLaPreuveEstFigeeAuPremierAcces` échoue sur un film réel).
- **Chantier 2b (J3 à J10)** : à déconseiller si, après I5 et J2, les balayages restent sous environ 20 % du temps ; si J4 trouve un mémo écrit à chaud hors liste ; ou si les journaux ordonnés sont exigés (coût du tampon). Il pèse 8 items, dont un L, pour environ 10 %.
- Le recouvrement des tâches pures du chantier 1 avec la marche (cacher entêtes et mots derrière la marche) demanderait une attente sur le chemin de lecture : non retenu.

## 8. Méthode

Lecture des fichiers cités. En complément, un outil jetable hors dépôt (module `golang.org/x/tools`, analyse de types et graphe d'appels RTA, une racine par balayage) a servi à : confirmer l'absence d'écriture de variable de paquet et d'accès au contexte pour les fonctions pures du § 1.1 et 2.3.6 ; lister les écrivains de champs de `FilmContext` ; chercher les crochets posés. Il sur-approxime (il suit les chemins de repli) : toute conclusion « non atteint » est solide, toute conclusion « atteint » est à confirmer par lecture ou par les gardes de J4.
