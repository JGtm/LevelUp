# RAPPORT — lot V12 : LA VISEE DES OCCUPANTS DE VEHICULE, PUBLIEE ET BRANCHEE

> Execute le 2026-09-03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`. GOCACHE isole (`scratchpad/gocache_v12`), mesures en AVANT-PLAN,
> `CGO_ENABLED=0` pour l'analyse, films du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks`, LECTURE SEULE).
>
> Ce lot EXECUTE le report n° 1 du lot V11 (§ 8.1 de `V11_ORIENTATION_TOURELLE_2026-09-03.md`) :
> publier la visee d'occupant sur l'EPISODE D'OCCUPATION, et remplacer le cone au cap du chassis.

---

## 0. LE RESULTAT EN CINQ LIGNES

1. **LE DOCUMENT PORTE LA VISEE, SUR L'EPISODE.** `vehicles[].rides[].aim = [{t, h, p}]`, echantillonne
   sur la grille du document (un point par frame, le premier observe gagne), alimente par
   `filmdec.ScanFilmBipedAimOnly` filtre au slot de l'occupant et a la fenetre de l'episode.
   `SchemaVersion` 30 -> **31**.
2. **COUVERTURE MESUREE SUR LES DEUX ARTEFACTS RECONSTRUITS : 11 / 11 EPISODES (100 %) PORTENT UNE
   VISEE.** 1 537 points publies pour 2 980 frames d'episode (51,6 %), sur 27 795 lectures brutes
   decodees. Le detail par episode est au § 3 — et il porte UN cas a 0 % qui explique a lui seul la
   moitie du manque (§ 6.1).
3. **LE CONE NE MENT PLUS, ET IL Y EN A UN PAR OCCUPANT.** Le calque web dessine desormais un cone
   PAR occupant actif, a SA couleur, oriente par SA visee ; le cap du chassis n'est plus que le
   REPLI. Mesure sur les artefacts publies : l'ecart entre la visee et le cap du chassis vaut
   **52,0 deg de mediane** (q1 22,0 · q3 88,7 · 68,2 % au-dela de 30 deg) sur `0d76e8f1` et
   **6,6 deg** (q1 4,5 · q3 23,2 · 21,4 %) sur `fccc61cd`.
4. **QUATRE COMPTEURS DE COUVERTURE**, dont le denominateur : `aimReads` (brut du film),
   `ridesWithAim`, `aimSamples`, `aimRideFrames`.
5. **RIEN DE LA TOURELLE N'EST PUBLIE**, et c'est la refutation mesuree du lot V11 reportee dans la
   chronique du schema 31 : l'entite tourelle ne replique RIEN, `i31`/`i41`/`i42` ne sont jamais
   emis. Le cone de l'artilleur vient de l'HOMME.

---

## 1. CE QUI EST LIVRE

### 1.1 Go — production

| fichier | etat | contenu |
|---|---|---|
| `replay/vehicle_rides_aim.go` | **NEUF** (127 L) | `vehicleAimBySlot` (index par slot d'occupant, trie) · `vehicleRideAimOf` (projection sur la grille de frames, « le premier observe gagne », bornee a l'episode) · `clampVehicleRideAim` (la serie suit son episode clampe). PUR. Toute la chaine de preuve V11 en en-tete. |
| `replay/document_vehicles.go` | MODIFIE (356 -> 435 L) | type `VehicleAim {t,h,p}` + champ `VehicleRide.Aim` (`omitempty`) + 4 compteurs de couverture + comptage dans `tallyVehicleRides` + journal et warn de silence. |
| `replay/build_vehicles.go` | MODIFIE (186 -> 214 L) | `VehicleScan.Aims` + `decodeFilmOccupantAims` (5e lecture du film, ADDITIVE ET NON FATALE). |
| `replay/vehicle_tracks.go` | MODIFIE (533 -> 538 L) | `AimReads` a l'initialisation de la couverture, `aimBySlot` passe aux entrees, clamp de la serie. |
| `replay/vehicle_rides.go` | MODIFIE (444 -> 449 L) | `vehicleRideInputs.aimBySlot` + remplissage sur la voie du TROU. |
| `replay/vehicle_rides_events.go` | MODIFIE (433 -> 437 L) | remplissage sur la voie de l'EVENEMENT (la voie nominale). |
| `replay/document.go` | MODIFIE (751 -> 780 L) | `SchemaVersion = 31` + chronique (ce que la version porte, son niveau de preuve, ce qu'elle refuse). |
| `replay/structure_test.go` | MODIFIE | ratchet 30 -> 31 + chronique v31. |
| `replay/vehicle_rides_aim_test.go` | **NEUF** (180 L) | 7 tests : echantillonnage deterministe, bornes de l'episode, serie vide, index par slot, clamp, **serialisation** (piege `omitempty` des deux angles), compteurs. |
| `api/openapi.yaml` | REGENERE | `VehicleAim` + `VehicleRide.aim` + 4 champs de `VehicleCoverage`. Diff STRICTEMENT additif. |
| `replay/testdata/assembly_000d5950.golden` | REGENERE | **UNE SEULE LIGNE** : `schema 30` -> `schema 31`. Rien d'autre n'a derive (verifie par `git diff` : 1 insertion, 1 suppression). |

`filmdec` : **AUCUN FICHIER TOUCHE**. Le decodeur `offline_aim_only.go` du lot V11 est consomme tel
quel.

### 1.2 Web

| fichier | etat | contenu |
|---|---|---|
| `features/match-replay/vehiclesAim.ts` | **NEUF** (110 L) | `VEHICLE_AIM_HOLD_FRAMES` (10 frames = 1 s, justifie par la densite mesuree) · `vehicleRideAimReading` (derniere lecture en vigueur, aucune interpolation) · `vehicleOccupantAimAt` (mesure d'abord, cap du chassis en repli). |
| `features/match-replay/vehiclesPaint.ts` | MODIFIE (379 -> 390 L) | `drawVehicleAimCone` (conducteur seul) devient `drawVehicleAimCones` : **une boucle sur les occupants actifs**, un cone chacun a sa couleur, longueur portee par l'elevation (`aimLengthScale`). `vehicleActiveRides` n'est plus appele deux fois par vehicule et par image. |
| `features/match-replay/vehiclesLayer.ts` | MODIFIE (489 -> 496 L) | en-tete (7e responsabilite, deportee) ; `vehicleDriverAt` n'est plus la porte du cone et le dit. |
| `features/match-replay/replayAimCone.ts` | MODIFIE (+7 L) | `pitchScale` devient `aimLengthScale`, EXPORTE : le cone d'un occupant traduit son elevation avec le MEME bareme qu'un pion (regle « <= 2 copies »). |
| `features/match-replay/replayLogic.ts` | MODIFIE (+7 L) | `lastIndexAt` elargi a `readonly {t:number}[]` — evite une seconde dichotomie copiee. |
| `features/match-replay/replayNormalize.ts` | MODIFIE | `ReplayVehicleRideReady` : la serie est COMBLEE au 3e niveau (premier tableau nullable imbrique dans un tableau imbrique du document). |
| `lib/api/types.ts` | MODIFIE | `ReplayVehicleAim`. |
| `lib/api/generated.ts` | REGENERE | +17 L, strictement additif. |
| `features/match-replay/replaySchemaLogic.ts` | MODIFIE | copie locale 30 -> 31 (garde-rail de parite avec le Go). |
| `features/match-replay/replayContract.test.ts` | MODIFIE | `vehicles[].rides[].aim` ajoute a la carte exhaustive des tableaux nullables. |
| `features/match-replay/vehiclesAim.test.ts` | **NEUF** (94 L) | 8 cas : lecture en vigueur, maintien, absence, point sans cap, mesure vs repli, **artilleur et passager ont chacun LEUR angle**, elevation absente = a plat. |
| `features/match-replay/vehiclesPaint.test.ts` | MODIFIE | le bloc « cone » reecrit : 1 cone par occupant (3 occupants = 3 cones), chaque cone a SON angle, repli cap-chassis sans visee ET sur lecture perimee, elevation qui allonge/raccourcit, aucun cone pour un occupant non resolu, aucun cone sur vehicule vide. |

---

## 2. LA FORME PUBLIEE, ET POURQUOI CELLE-LA

```json
"rides": [{"t0": 120, "t1": 271, "slot": 554, "xuid": "...", "seat": 0, "src": "mixed",
           "aim": [{"t": 120, "h": 287.4, "p": -3.2}, {"t": 121, "h": 291.1}, ...]}]
```

- **SUR L'EPISODE, pas ailleurs.** `Point.H` ne convient pas (l'occupant n'a PAS de position
  pendant l'episode : il n'y a aucun point ou accrocher l'angle) ; `VehicleTrack.Samples` non plus
  (la visee est celle de l'OCCUPANT — un vehicule en porte plusieurs, simultanees et distinctes).
- **MEMES CONVENTIONS QUE LE PION, au bit pres** : `h` et `p` sortent du meme composant `i21` et du
  meme accesseur (`aimHeadingDegFromRaw` / `aimPitchDegFromRaw`, detenteur unique depuis V11).
  `headingForJSON` (un cap qui s'arrondit a 0 est publie 360) et `pitchForJSON` (une elevation a
  plat est OMISE — son absence VEUT DIRE « a plat ») sont reutilises tels quels.
- **UN POINT PAR FRAME AU PLUS, le premier observe gagne** — la regle de `vehicleSamplesOf` et de
  `decimateTracks`. Le film replique a 5-46 lectures/s pour 10 frames/s : publier le brut
  multiplierait le poids du calque pour un rendu que le client n'affiche pas plus finement.
- **AUCUNE INTERPOLATION**, ni au serveur ni au client : interpoler deux caps ferait tourner le cone
  par le chemin le plus court a travers 0/360 deg, un artefact que le film ne montre pas.

---

## 3. CHIFFRES AVANT / APRES — LES DEUX ARTEFACTS DE DEMONSTRATION

Reconstruits par `cmd/replay-build` (`LEVELUP_REPO_ROOT` = ce worktree), puis copies vers
`C:/Users/Guillaume/Projects/LevelUp-wt-capture-rejeu/data/cache/replays/halo_infinite/`.

| | `0d76e8f1` (Behemoth) | `fccc61cd` (Launch Site) |
|---|---|---|
| schema | 30 -> **31** | 30 -> **31** |
| octets | 2 487 918 | 2 026 137 |
| vies de vehicule publiees | 20 | 8 |
| episodes d'occupation | 9 | 2 |
| **lectures de visee BRUTES (`aimReads`)** | **22 963** | **4 832** |
| **episodes avec visee (`ridesWithAim`)** | **9 / 9 (100 %)** | **2 / 2 (100 %)** |
| **points publies (`aimSamples`)** | **1 419** | **118** |
| **frames d'episode (`aimRideFrames`)** | 2 821 | 159 |
| part des frames d'episode couvertes | **50,3 %** | **74,2 %** |
| cones dessines AVANT (schema 30) | 1 par vehicule au plus (siege 0), au cap du chassis | idem |
| cones dessines APRES | **1 par occupant actif resolu**, a SA visee mesuree | idem |

### 3.1 Detail episode par episode (`0d76e8f1`)

```
vie slot=768 warthog  · ride slot=522 siege=0    src=mixed frames=  27 visees= 11 ( 41 %)
vie slot=773 warthog  · ride slot=554 siege=0    src=mixed frames= 152 visees=124 ( 82 %)
vie slot=773 warthog  · ride slot=551 siege=nil  src=gap   frames=  51 visees= 20 ( 39 %)
vie slot=773 warthog  · ride slot=561 siege=0    src=mixed frames= 907 visees=  2 (  0 %)   <-- § 6.1
vie slot=773 warthog  · ride slot=551 siege=0    src=mixed frames= 212 visees=124 ( 58 %)
vie slot=775 warthog  · ride slot=555 siege=nil  src=gap   frames= 243 visees=202 ( 83 %)
vie slot=776 mongoose · ride slot=531 siege=0    src=mixed frames=  59 visees= 37 ( 63 %)
vie slot=777 ghost    · ride slot=514 siege=0    src=mixed frames=  83 visees= 67 ( 81 %)
vie slot=786 wasp     · ride slot=559 siege=0    src=mixed frames=1087 visees=832 ( 77 %)
```

`fccc61cd` :

```
vie slot=772 mongoose · ride slot=515 siege=0 src=mixed frames=132 visees=100 ( 76 %)
vie slot=776 warthog  · ride slot=558 siege=0 src=mixed frames= 27 visees= 18 ( 67 %)
```

**HORS l'episode `slot=561`** (§ 6.1), la couverture de `0d76e8f1` passe de 50,3 % a **1 417 / 1 914
= 74,0 %** — c'est-a-dire le meme ordre que `fccc61cd`.

### 3.2 L'ERREUR QUE LE CONE FAISAIT — mesuree SUR L'ARTEFACT PUBLIE

Chaque point de visee publie est compare au cap du chassis EN VIGUEUR a la meme frame (le dernier
`VehicleSample.h` connu, exactement ce que `vehicleHeadingAt` rend au client) :

| film | paires | mediane | q1 | q3 | part > 30 deg |
|---|---|---|---|---|---|
| `0d76e8f1` | 1 018 | **52,0 deg** | 22,0 | 88,7 | **68,2 %** |
| `fccc61cd` | 98 | **6,6 deg** | 4,5 | 23,2 | 21,4 % |

**CE QUE JE NE MAQUILLE PAS.** Ces chiffres NE SONT PAS ceux du lot V11 (15,7-21,8 deg de mediane),
et l'ecart est explicable, pas suspect : V11 comparait a la direction de la VELOCITE `i1` sur une
fenetre de 20 s AVANT une sortie, donc sur du vehicule EN MOUVEMENT ; ce tableau-ci compare a ce que
le CLIENT dessine, c'est-a-dire au dernier cap connu, **REPORTE tant que le vehicule est sous
5 m/s** (regle `vehicleMinSpeedMPS`, cf. `VehicleSample.H`). Un vehicule a l'arret garde donc un cap
fige pendant que son occupant balaye la carte du regard — et c'est precisement la situation ou
l'ancien cone se trompait le plus. Les deux mesures disent la meme chose et la seconde est la plus
severe parce qu'elle est la plus proche du rendu reel. Sur `fccc61cd`, deux episodes courts et
roulants : 6,6 deg, l'ancien cone y etait presque juste.

---

## 4. LE CONE, COTE WEB

**AVANT (schema 30).** `drawVehicleAimCone` : `vehicleDriverAt` (siege 0 STRICT), un seul cone,
angle = `vehicleAimAngle(vehicleHeadingAt(...))` — le cap du chassis. Un artilleur, un passager, un
siege NON LU : aucun cone.

**APRES (schema 31).** `drawVehicleAimCones` boucle sur `vehicleActiveRides` :

1. l'encre est celle de l'occupant (`vehicleRideColor` : xuid du document d'abord, pont slot
   ensuite) — **un occupant non resolu ne recoit aucun cone**, regle inchangee, deliberement ;
2. l'angle est `vehicleOccupantAimAt` : la derniere lecture de SA serie en vigueur a l'image (dans
   `VEHICLE_AIM_HOLD_FRAMES` = 10 frames), sinon le cap du chassis ;
3. la LONGUEUR porte l'elevation par le meme bareme que le pion (`aimLengthScale`, exporte plutot
   que recopie).

**LE MAINTIEN A UNE SECONDE, ET IL EST MESURE, pas choisi.** Le film replique la visee d'un occupant
a 5-46 lectures/s pour 10 frames/s : une seconde sans la moindre lecture n'est pas un defaut
d'echantillonnage, c'est une interruption reelle. Ce n'est PAS `TIMING_MS.aimHold` (5 s, le maintien
du regard d'un PION, justifie par une couverture de ~44 % des points) : maintenir 5 s ici afficherait
une direction perimee la ou le chassis, lui, est connu a l'image pres.

**CE QUI RESTE UNE APPROXIMATION ASSUMEE.** Un occupant SANS visee en vigueur recoit le cap du
chassis, y compris s'il est artilleur ou passager — ou cette approximation est plus fausse que pour
un conducteur. C'est la consigne du mandat (« le cap du vehicule reste le repli »), et le champ
`measured` du resultat porte la distinction pour qui voudra la rendre visible ; aucun rendu ne
l'exploite aujourd'hui.

---

## 5. GATES D'EXECUTION — TOUS PASSES DANS CETTE SESSION

```
Go (GOCACHE isole, avant-plan)
  gofmt -l internal/                                                    -> sortie VIDE
  CGO_ENABLED=0 go vet ./internal/analysis/filmdec/ ./internal/analysis/replay/   -> exit 0
  CGO_ENABLED=0 go test ./internal/analysis/replay/ ./internal/analysis/filmdec/ -count=1
      ok  levelup/go-api/internal/analysis/replay    122,8 s
      ok  levelup/go-api/internal/analysis/filmdec     2,9 s
      grep '^--- FAIL:'  =  VIDE          (aucune variable d'environnement posee)
  CGO_ENABLED=1 go build ./...                                          -> exit 0 (service)
  CGO_ENABLED=1 go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate  -> PASS (contrat a jour)
  go run ./cmd/openapi-gen                                              -> 652 576 octets

Web (node_modules/.tmp purge avant)
  npm run typecheck   -> exit 0
  npm run lint        -> 0 error, 25 warnings (TOUTES preexistantes : react-hooks/incompatible-library
                         sur TanStack Table, directives eslint-disable inutiles — aucune dans ce lot)
  npm run test        -> 546 fichiers, 5 725 tests PASSES, 14 skippes, 0 echec
```

Seuils du depot : fonctions neuves, la plus longue fait **28 L** (`vehicleRideAimOf`), seuil 80.
Fichiers : tous les fichiers NEUFS sont <= 180 L. **Deux fichiers deja au-dessus du seuil de 500 L
s'accroissent, et il faut le dire** : `vehicle_tracks.go` 533 -> **538** (+5, dont 3 de code : une
affectation, un appel de clamp, un commentaire de deux lignes) et `document.go` 751 -> **780** (+29,
**uniquement du commentaire** : la chronique du schema, que la convention du fichier exige a chaque
montee de version). `vehiclesLayer.ts` serait passe a 581 L : la 7e responsabilite a donc ete
EXTRAITE dans `vehiclesAim.ts` plutot qu'exemptee (496 L apres extraction), meme geste que la
separation `vehiclesPaint.ts` du lot precedent.

---

## 6. DECOUVERTES — NOTEES, NON TRAITEES (regle « zero fix hors perimetre »)

### 6.1 UN EPISODE DE 90,7 s AVEC 2 LECTURES DE VISEE

`0d76e8f1`, vie `slot=773` (warthog), episode `slot=561` siege 0, `src=mixed`, 907 frames, **2 visees
(0,2 %)**. Il pese a lui seul **905 des 1 402 frames non couvertes** du film : sans lui la couverture
passe de 50,3 % a 74,0 %.

Ce N'EST PAS un defaut du decodeur de visee : le meme slot bipede emet a 5-46 lectures/s partout
ailleurs. La forme du cas — un episode a fin OUVERTE dont `endUS` a ete etendu a `life.hiUS` (cf.
`vehicleRideFromEpisode`, `ep.openEnd`) — designe la BORNE DE FIN de l'episode, pas la visee : le
joueur a tres probablement quitte le vehicule bien avant la fin publiee, et la tolerance de
recensement (~20 s) plus la fenetre de vie ont etire l'episode. La visee, en refusant de suivre, est
le premier instrument qui le RENDE VISIBLE — c'est un temoin gratuit de la qualite des bornes.
**Non traite** : corriger la borne relevait de `vehicle_rides_events.go` et changerait les episodes
publies, hors perimetre de ce lot. Conditions de reprise au § 7.

### 6.2 `TestReplayDocumentFieldCountIsFrozen` (paquet `contracttest`) ETAIT DEJA ROUGE

`46 champ(s) publie(s) par l artefact, attendu 44`. **Ce lot n'ajoute AUCUN champ de premier niveau**
(`git diff` sur `document.go` : zero balise `json:` ajoutee), et le fichier de test n'est pas
modifie localement : les deux champs excedentaires sont `vehicles` et `vehicleLabels`, du schema 29,
livres sans mise a jour du compteur gele. Le test echoue donc AVANT ce lot comme apres. Non traite,
signale.

### 6.3 LA VISEE COUTE UNE 5e LECTURE COMPLETE DU FILM

`ScanFilmBipedAimOnly` reparcourt tous les chunks. Cout mesure de bout en bout :
`replay-build` sur `0d76e8f1` a pris ~80 s de plus qu'un build sans ce balayage (~4 min au total).
Les deux balayages ne peuvent PAS etre fusionnes sans relacher la porte du nuage de positions
(celle sous laquelle tout le calque a ete mesure) ; une fusion propre supposerait un balayage unique
rendant DEUX flux. Non instruit.

### 6.4 L'ORACLE DU TIR RESTE NON POSE

Report n° 4 du lot V11 : croiser la visee lue a l'instant d'un tir en vehicule avec la direction
tireur -> victime validerait la visee A BORD par une source totalement independante. Les 23 tirs en
vehicule de `0d76e8f1` (tous poses, cf. journal) en sont le corpus. Non fait ici : ce lot est une
PUBLICATION, pas une mesure ; l'oracle deja obtenu au lot V11 (0,2-0,5 deg contre `Point.H`) est du
meme ordre de force.

### 6.5 AUCUN EPISODE DU CORPUS NE PORTE UN SIEGE > 0

Sur les 11 episodes des deux artefacts, **7 portent le siege 0, 2 aucun siege lu, 0 un siege > 0**.
Le cas ARTILLEUR / PASSAGER est donc etabli PAR CONSTRUCTION (chaque occupant a son slot bipede,
donc sa visee — et le code, comme les tests web, le traite explicitement) mais il n'est **PAS
DEMONTRE SUR PIECES dans un artefact reconstruit** : aucune image de ces deux films ne montre deux
cones distincts sur le meme vehicule. C'est la meme limite qu'au lot V11 (§ 8.5), inchangee. Un film
dont la sortie nomme la tourelle serait le corpus qu'il faut.

---

## 7. CONDITIONS DE REPRISE

1. **LA BORNE DE FIN DES EPISODES A FIN OUVERTE** (§ 6.1). La serie de visee est desormais un
   instrument disponible : un episode dont la visee cesse 80 s avant `t1` est un episode dont la fin
   est fausse. Une regle candidate — fermer l'episode a la derniere visee + le maintien — est
   TESTABLE contre les sorties attestees, et elle ne demande aucun decodage nouveau.
2. **DEUX CONES SUR UN MEME VEHICULE, SUR PIECES** (§ 6.5). Reconstruire un artefact d'un film ou un
   evenement de sortie porte un siege > 0 (les 6 desaccords du lot V8 sont la piste).
3. **L'ORACLE DU TIR** (§ 6.4), inchange depuis V11.
4. **`i2` DU CHASSIS RESTE OUVERT** (V11 § 2), inchange : c'est la seule orientation qui vaudrait
   AUSSI A L'ARRET — et le § 3.2 de ce rapport montre que c'est exactement la ou le repli est le
   plus faux.
5. **LE COMPTEUR GELE DU CONTRAT** (§ 6.2) : a corriger par le detenteur du lot schema 29.

---

## 8. STATUT DES ITEMS DU MANDAT

| item | statut | justification |
|---|---|---|
| 1. publier la visee sur `VehicleRide`, forme `aim: [{t,h,p}]`, grille du document | `[x]` | § 1.1, § 2 — `omitempty`, arrondis `headingForJSON`/`pitchForJSON` (les memes que `Point.H`/`Point.P`), commentaires FR documentant source et mesure |
| 1bis. bump 30 -> 31 + chronique `document.go` ET ratchet `structure_test.go` | `[x]` | § 1.1 |
| 1ter. `openapi-gen` + golden regenere | `[x]` | golden : **UNE LIGNE** (`schema 30` -> `schema 31`), verifie par `git diff` (1 insertion / 1 suppression). openapi : diff strictement additif, `TestOpenAPIYAMLIsUpToDate` PASS |
| 2. compteurs de couverture | `[x]` | `aimReads` · `ridesWithAim` · `aimSamples` · `aimRideFrames` (le denominateur), sur le modele des compteurs existants + journal + warn de silence |
| 3. cone par occupant, repli cap-vehicule, couleur par occupant | `[x]` | § 4 — `drawAimSector` et le resolveur par xuid REUTILISES, aucune geometrie reecrite |
| 3bis. `npm run generate-types` | `[x]` | +17 L, additif |
| 4. gates Go (gofmt, vet, tests sans env, service CGO=1) | `[x]` | § 5 — `grep '^--- FAIL:'` VIDE |
| 4bis. gates Web (purge `.tmp`, typecheck, lint, test) | `[x]` | § 5 — 5 725 tests verts, 0 erreur de lint |
| 4ter. tests attendus (serialisation, echantillonnage, cone par occupant, repli, aucun cone sans occupant) | `[x]` | 7 tests Go (`vehicle_rides_aim_test.go`) + 8 cas web purs (`vehiclesAim.test.ts`) + 10 cas de trace (`vehiclesPaint.test.ts`, bloc du cone reecrit) |
| 5. reconstruire les 2 demos et les copier | `[x]` | § 3 — 2 487 918 et 2 026 137 octets, copies verifiees dans `LevelUp-wt-capture-rejeu` |
| 5bis. donner les compteurs | `[x]` | § 3 et § 3.1 |
| rapport + entree de thought_log EN TETE | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md` |
| verification VISUELLE du rendu (deux cones distincts a l'ecran) | `[!]` **non faite** | § 6.5 — aucun episode du corpus ne porte un siege > 0, donc aucune image des deux artefacts ne peut la montrer. Le cas est couvert par les tests, pas par l'oeil |

---

## 9. CR HONNETE

- **Ce que le lot livre** : la visee de chaque occupant est dans le document, sur le seul objet qui
  puisse la porter, avec les memes conventions d'angle que le pion ; le calque en fait un cone par
  occupant. 11 episodes sur 11 en portent une, 27 795 lectures brutes decodees sur deux films.
- **Ce que le lot corrige, chiffres a l'appui** : le cone du conducteur se trompait de 52,0 deg en
  mediane sur `0d76e8f1` (68 % des instants au-dela de 30 deg) — davantage que ce que V11 laissait
  attendre, parce que le cap du chassis est FIGE a l'arret alors que le regard ne l'est pas.
- **Ce que le lot ne prouve pas** : qu'un artilleur et un conducteur affichent deux cones distincts
  DANS UN ARTEFACT REEL. Le corpus des deux demos ne porte aucun siege > 0. C'est etabli par
  construction et par les tests, pas par l'image.
- **Ce que la visee a revele en passant, et qui ne lui etait pas demande** : un episode publie de
  90,7 s dont la visee s'arrete au bout de 2 lectures. La borne de fin des episodes a fin ouverte
  est suspecte, et la serie de visee est le premier instrument capable de la contredire. Non traite,
  inscrit au § 7.1.
- **Ce qui a failli passer inapercu** : `vehiclesLayer.ts` serait sorti a 581 L, et il aurait ete
  tentant de l'exempter « pour une seule fonction ». La couture (`vehiclesAim.ts`) tombe en fait sur
  une frontiere nette — un seul fichier lit `ride.aim` — et elle rend la regle testable a part.
  Deuxieme quasi-oubli : la serie est un tableau nullable au TROISIEME niveau du document, le
  premier du contrat ; sans la garde `replayContract.test.ts`, elle serait arrivee au rendu en
  `null`.
