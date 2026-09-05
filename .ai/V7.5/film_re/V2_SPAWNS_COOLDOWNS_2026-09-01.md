# RAPPORT — lot V2 (vehicules : emplacements, spawns, cooldowns, etat detruit)

> Execute le 2026-09-01 dans le worktree `LevelUp-wt-vehicules`. Aucun commit, aucun `git add`,
> aucune ecriture DuckDB. Toutes les mesures en AVANT-PLAN, `CGO_ENABLED=0`, GOCACHE isole
> (`scratchpad/gocache_v2`), sur les donnees reelles du checkout principal
> (`C:/Users/Guillaume/Projects/LevelUp/data`, lecture seule).
>
> Corpus mesure : **18 films Behemoth Super Fiesta + 8 films Launch Site Super Fiesta** (la liste
> V1a des films retenus). Prerequis V1b utilises tels quels, jamais modifies :
> `ScanFilmVehicleCreations` (naissance + chassis `MPPWord32`), `ScanFilmWorldObjectKeyframes`
> (recensement / bornage des vies), et le gate durci de V1.5 (nuage + calibration MPP).

## 0. LE RESULTAT EN SEPT LIGNES

1. **Les vehicules naissent a des EMPLACEMENTS FIXES EXACTS** (rayon d'amas **0,00 m**) : **6 pads
   sur Behemoth** (grille 2x3), **4-5 pads sur Launch Site**. 441 naissances Behemoth, 135 Launch
   Site, **100 % de chassis lus** (`MPPWord32`), 11 chassis distincts par carte.
2. **Le seuil « <= 4 amas par chassis » est REFUTE sur Behemoth (6 pads), PASSE sur Launch Site
   (<= 5)** — il est map-dependant. Le fait robuste (positions fixes, rayon 0) tient sur les deux.
   En Super Fiesta, **le CHASSIS est tire au hasard par pad** : d'ou des amas/chassis = sous-ensemble
   des pads, et une stabilite film-a-film bruitee pour les chassis frequents.
3. **CONFRONTATION .mvar — le piege Forge est confirme SUR PIECES** : le catalogue `map_weapon_pads.json`
   couvre les deux cartes mais ne porte que les familles **rack / power / powerup — AUCUNE famille
   vehicule**. Les repères sont ALIGNES (bbox naissances ~ bbox pads), et les emplacements vehicule
   sont a **3-10 m** des pads d'arme. Gate NON APPLICABLE (rien a confronter). REPORTE.
4. **COOLDOWNS non resolus a la resolution du recensement** : median ~35 s, IQR/mediane 0,87-0,98
   (seuil 0,25) : ECHEC. Les images-cles sont a ~20 s ; un cooldown de cet ordre n'est pas separable
   du bruit de bornage.
5. **ETAT DETRUIT — i14 object-dissolver INUTILISABLE** : 0,0019-0,0024 % des records delta, tres
   sous le plancher de faux positifs (0,17 %). NON CONCLUANT (comme i11 au cadrage).
6. **DATER LA DESTRUCTION PAR LA MORT — signal AU HASARD sur le corpus** : la coincidence temporelle
   (+/-2 s) entre fin de vie vehicule et une mort vaut **52 % vs 48 % au temoin (Behemoth)**, **47 %
   vs 43 % (Launch Site)** — a peine au-dessus du hasard, car les morts sont denses (~95/match). Le
   film pilote (0d76e8f1 : 38 % vs 18 %) NE GENERALISE PAS.
7. **CLASSER detruit-avec-mort / ejection / despawn est BLOQUE** sur deux prerequis absents :
   (A) l'attribution du CONDUCTEUR (modele « trou de position » de V1a.4, non porte) et (B) un signal
   de destruction date (i14 sous plancher ; sons non decodes). Le critere spatial le confirme : les
   victimes coincidentes situables sont des BADAUDS (mediane 17-25 m ; **0 a 4 % a < 5 m**), et le
   conducteur probable tombe dans le trou de position embarque (V1a.4), donc introuvable sur sa propre
   trace.

---

## 1. ITEM 1 — EMPLACEMENTS DE SPAWN

### 1.1 Methode, et le passage unite -> metres

Les naissances passent par le **gate durci de V1.5** (reutilise sans copie : `vehProbe`,
`vehicleBuildCloud`, `vehicleCalibrateMPP` de `vehicle_creation_test.go`) : default-state porte
(`consumeDefaultStateTI40`) + gate i0 dyn.-prec. + coincidence avec le NUAGE des positions reelles +
calibration MPP via `ti=37`. La coordonnee est decodee en cube unite `[0,1]` (comme V1.5, pour NE PAS
bouger le gate valide) puis **projetee en metres** par les bornes de la carte
(`map_quant_bounds.json`) : `metres[ax] = min[ax] + unite[ax] * (max[ax] - min[ax])` — lineaire, exact.
Une naissance par vie `(slot, gen)` (la plus precoce). Regroupement en amas : algorithme du meneur,
seuil 2 m, DETERMINISTE (points tries).

### 1.2 Behemoth — 6 pads fixes (grille 2x3), rayon 0,00 m

441 naissances, **0 sans chassis lu**, 11 chassis distincts. bbox naissances (m) :
x `-146,9..-100,7` · y `26,3..80,3` · z `8,0..10,4`. Le regroupement tous-chassis (amas de support
`n >= 3`) rend **six emplacements**, tous de **rayon 0,00 m** :

| emplacement (m) | naissances |
|---|---|
| (-100,7 , 53,7 , 10,3) | 98 |
| (-146,9 , 53,7 , 10,3) | 75 |
| (-101,6 , 27,6 ,  8,8) | 73 |
| (-101,7 , 80,2 ,  8,3) | 67 |
| (-146,1 , 80,3 ,  8,3) | 66 |
| (-146,0 , 27,4 ,  8,5) | 61 |

C'est une **grille 2x3** : x dans `{-146, -101}`, y dans `{27, 54, 80}`. Deux naissances isolees
supplementaires (chassis vus une fois) au bord de la zone.

**Gate « <= 4 amas de rayon <= 2 m par chassis » : ECHEC.** Le chassis le plus frequent
(`0x00fe32c0f4`, 143 naissances) occupe **5** des 6 pads ; plusieurs chassis frequents en occupent 5.
Le rayon est nul partout (naissances a la coordonnee exacte). **La cause de l'echec n'est pas le
bruit : c'est que Behemoth a 6 pads, pas <= 4**, et qu'en Super Fiesta chaque pad tire un chassis au
hasard, donc un chassis frequent apparait sur presque tous les pads. Le bon objet n'est pas
« amas par chassis » mais **l'ensemble mutualise des pads**, qui est net (6, rayon 0).

**Stabilite film-a-film** : les chassis peu frequents sont stables (amplitude 0-1 : PASS) ; les
chassis frequents varient de 2 a 5 amas/film (amplitude 2-3 : ECHEC) — consequence directe du tirage
aleatoire par pad en Super Fiesta, pas d'une instabilite des pads.

### 1.3 Launch Site — 4-5 pads fixes, rayon 0,00 m

135 naissances, **0 sans chassis lu**, 11 chassis distincts. bbox naissances (m) :
x `-14,2..27,9` · y `-35,2..8,9` · z `-3,8..-0,5`. Emplacements principaux (amas `n >= 3`, rayon
0,00 m) : (20,5 , 2,7 , -3,8) ; (-3,9 , 8,9 , -3,2) ; (-14,2 , -35,2 , -0,6) ; (1,9 , 7,1 , -3,8) ;
plus quelques naissances isolees. **Gate « <= 4 amas par chassis » : PASS** — Launch Site a moins de
pads (4-5), donc aucun chassis n'en occupe plus de 4.

### 1.4 Verdict item 1

**Le fait est acquis et fort : les vehicules naissent a un ensemble FIXE et EXACT de pads
(rayon 0,00 m), 6 sur Behemoth, 4-5 sur Launch Site.** Le seuil « <= 4 amas/chassis » du plan est
map-dependant (echoue sur Behemoth par exces de pads) ; il est a remplacer par la mesure de
l'ENSEMBLE MUTUALISE des pads, qui est le livrable exploitable.

---

## 2. ITEM 2 — CONFRONTATION .mvar (le piege Forge, sur pieces)

Catalogue statique employe : `data/titles/halo_infinite/reference/map_weapon_pads.json` (schema 1).
Il couvre les deux cartes :

| carte | module (.mvar) | pads | familles | famille vehicule |
|---|---|---|---|---|
| Behemoth | `behemoth_va_behemoth.mvar` | 21 | rack 14 · power 6 · powerup 1 | **AUCUNE** |
| Launch Site | `launch_site_va_launchsite.mvar` | 19 | rack 16 · power 2 · powerup 1 | **AUCUNE** |

**Les repères sont ALIGNES** (le doute etait reel) : sur Behemoth, bbox pads x `-143,7..-104,0` /
y `25,0..82,6` / z `3,8..11,3` recouvre la bbox des naissances x `-146,9..-100,7` / y `26,3..80,3` /
z `8,0..10,4`. Les naissances vehicule tombent donc dans la MEME zone de jeu Forge que les pads
d'arme, dans le MEME repere BSP — la projection unite->metres est donc bien calee.

**Distance des emplacements vehicule au pad d'arme le plus proche** : 3,2-3,9 m (Behemoth,
6 emplacements) ; 5,8-10,5 m (Launch Site, 4 emplacements). **0/6 et 0/4 a moins de 1 m.** Les
vehicules ne naissent PAS aux racks d'armes : c'est un sous-systeme distinct.

**Verdict : gate « >= 80 % des amas a < 1 m d'un emplacement declare » NON APPLICABLE.** Le catalogue
`.mvar` — canevas + rack d'une carte Forge, exactement le piege documente — ne declare AUCUN
emplacement vehicule (l'extracteur n'a retenu que 3 `type_id` d'arme). **REPORTE** : une vraie
confrontation exige de re-extraire le `.mvar` sur le `type_id` du spawner de vehicule, ce qui suppose
(a) l'acces au parseur `.mvar` (dans `himap` — collision agent, non touche) et (b) les fichiers
`.mvar` eux-memes (ABSENTS du dossier `data`). Condition de reprise inscrite ci-dessous (section 6).

---

## 3. ITEM 3 — COOLDOWNS

Par carte, les emplacements de reference sont les amas mutualises `n >= 3` (item 1). Par film, par
emplacement, les vies sont ordonnees par instant de naissance ; le cooldown est l'ecart entre la
**fin BORNEE** d'une vie (premiere image-cle apres son dernier recensement) et la naissance suivante
au meme emplacement. Les couples a ecart negatif (deux vies simultanees au meme pad) sont ecartes.

| carte | emplacements | cooldowns | chevauchements ecartes | mediane | IQR | IQR/mediane (seuil 0,25) |
|---|---|---|---|---|---|---|
| Behemoth | 6 | 132 | 185 | 36 s | 31 s | **0,87 : ECHEC** |
| Launch Site | 4 | 42 | 54 | 34 s | 33 s | **0,98 : ECHEC** |

**Verdict : distribution NON resserree.** Deux causes, ecrites AVANT la mesure :
1. **Limite structurelle de resolution** : les images-cles sont espacees de ~20 s (mesure du
   recensement), donc chaque fin de vie est bornee a **+/-20 s** — jamais datee. Un cooldown de
   l'ordre de 20-35 s n'est pas separable de cette incertitude, et un cooldown < ~20 s n'est pas
   mesurable du tout.
2. **Densite Super Fiesta** : le grand nombre de chevauchements ecartes (185 / 54) montre que
   plusieurs vies coexistent souvent au meme pad — le regime n'est pas « une vie, puis un cooldown,
   puis la suivante ».

**Le cooldown vehicule N'EST PAS mesurable a la resolution du recensement.** Il le deviendra avec un
signal de destruction DATE dans le flux (item 4) — qui, aujourd'hui, n'existe pas de facon fiable.

---

## 4. ITEM 4 — ETAT DETRUIT

### 4.1 i14 object-dissolver dans le flux delta — INUTILISABLE

Frequence de i14 (`object-dissolver`, porte, ecs_table) dans les records delta `ti=40` acceptes
(meme acceptation que le cadrage V0) :

| carte | i14 / records delta | % | plancher faux positifs |
|---|---|---|---|
| Behemoth (18 films) | 9 / 368 847 | **0,0024 %** | 0,17 % |
| Launch Site (8 films) | 2 / 103 305 | **0,0019 %** | 0,17 % |

**Tres sous le plancher de 0,17 % : NON CONCLUANT** (regle du cadrage : ne pas interpreter sous le
plancher). i14 ne date pas la destruction dans le flux delta, exactement comme i11 (dead-state) au
cadrage.

### 4.2 Dater la destruction par la MORT de l'occupant — signal AU HASARD sur le corpus

Instrument dedie dans le package `replay` (qui accede a `filmdec` ET a `replay`). Il reutilise, sans
copie : positions joueur (`filmdec.ScanFilmBipedPositions`), fil des morts (`ScanFilmDeaths`), index
joueur (`ScanFilmPlayerIndices`), et surtout le **PONT slot->xuid + le CALAGE d'horloge PROUVE** du
pont de production (`buildOwners` -> `own.DeathOffsetMS`, `own.SlotXUID` ;
`horlogeFilm_ms = death.TimeMS + DeathOffsetMS`). Fin de vie SERREE = dernier echantillon de
trajectoire du slot vehicule dans la fenetre du recensement (resolution ~0,5 s), et sa position.

Seuils ecrits AVANT mesure : coincidence temporelle **+/-2 s** ; TEMOIN = le meme test au calage
**decale de 37 s** (decorrelation ; un decalage constant, pas d'un cran, car les fins de vie sont
autocorrelees — lecon V1a) ; validation = la part reelle doit depasser le temoin de plus de 10 pts.

| carte | vies vehicule | DETRUIT temporel | TEMOIN (decale) | verdict |
|---|---|---|---|---|
| Behemoth (18 films) | 654 | 339 (**52 %**) | 313 (**48 %**) | **NON VALIDE (au hasard)** |
| Launch Site (8 films) | 206 | 97 (**47 %**) | 89 (**43 %**) | **NON VALIDE (au hasard)** |
| corpus (26 films) | 860 | 436 (51 %) | 402 (47 %) | au hasard |

**Le signal est au niveau du hasard.** Cause mesuree : les morts sont DENSES (~85-101 par match) ;
une fenetre de +/-2 s dans un match de ~600 s en capture ~45-50 % par pur hasard. Le film pilote
`0d76e8f1` (38 % vs 18 %) est un cas favorable qui NE GENERALISE PAS — plusieurs films decents
rendent meme un temoin SUPERIEUR au reel (ex. `21468645` : 15 vs 22).

### 4.3 Critere spatial (victime a < 5 m) — les coincidences sont des BADAUDS

Comme `Death` ne porte pas de position, la victime est situee par sa propre trace joueur a l'instant
de sa mort (via `slotTrack.at` + le pont `SlotXUID`).

| carte | victimes situables | mediane distance | min | a < 5 m | sans echantillon (trou embarque) |
|---|---|---|---|---|---|
| Behemoth | 251 | 17-25 s/lot (17 a 25 m) | 1,4 m | **11 (~4 %)** | 12 |
| Launch Site | 76 | 21 m | 6,5 m | **0** | 10 |

**Lecture, et c'est un recoupement fort avec V1a.4** : les victimes coincidentes que l'on PEUT situer
sont a 17-25 m en mediane — ce sont des BADAUDS morts ailleurs au meme instant, pas le conducteur.
Le conducteur, LUI, ne replique plus sa position tant qu'il est EMBARQUE (« l'enfant attache ne
replique plus », V1a.4) : quand il meurt dans le vehicule, il est dans son TROU de position et reste
introuvable sur sa propre trace (les 10-12 cas « sans echantillon »). Aucun bug d'echelle : les
distances 1,4-30 m sont physiquement plausibles en arene, ce qui confirme au passage que positions
joueur et vehicule partagent le meme repere monde.

### 4.4 Classer detruit-avec-mort / ejection / despawn — BLOQUE, avec la cause exacte

La demande (verifier que l'occupant MEURT VRAIMENT et PASSE EN ATTENTE DE RESPAWN, et distinguer
ejection / despawn) se heurte a deux prerequis absents :

- **(A) Attribution du CONDUCTEUR.** Le fil des morts donne bien de VRAIES morts (evenement
  `EventTypeDeath` = le joueur passe en attente de respawn — les deux conditions de la demande sont
  reunies par une entree du fil). Mais **on ne sait pas de QUI attribuer la mort au vehicule** : le
  conducteur n'est pas identifie (le modele « debut de trou » de V1a.4 n'est pas porte). Sans lui, on
  ne peut ni confirmer que l'occupant est mort, ni le distinguer d'une ejection.
- **(B) Signal de DESTRUCTION date.** Separer EJECTION (le vehicule explose, l'occupant a saute)
  de DESPAWN (le vehicule disparait sans exploser) exige de savoir si le vehicule a EXPLOSE. i14 est
  sous le plancher (4.1) et les sons ne sont pas decodes. Sans lui, ejection et despawn sont
  indistinguables.

**Ce qui EST mesurable aujourd'hui** : DESPAWN-ou-EJECTION (aucune mort coincidente) ~48-53 % des
vies ; DESTRUCTION-POSSIBLE (mort coincidente, joueur quelconque) ~47-52 % — mais au hasard, donc non
attribuable. **La classification en trois etats attend l'attribution du conducteur (A) puis un signal
de destruction (B).**

### 4.5 BONUS a exploiter plus tard (signale, non traite)

6 banques Wwise `sb_008_exp_vehicle_{large,med,small}_{covenant,unsc}` = explosions de vehicule par
taille/faction. C'est le SON de l'etat detruit (lot sons) — et potentiellement le signal de
destruction (B) manquant, si le fil des sons est un jour date et decode.

---

## 5. INSTRUMENTS ECRITS (neufs, gardes, aucun code de production modifie)

| fichier | package | contenu |
|---|---|---|
| `apps/go-api/internal/analysis/filmdec/vehicules_v2_test.go` | filmdec | `TestV2SpawnsCooldowns` : items 1-3 + i14 (item 4a). Gardes `V2_FILMS`, `V2_FILM_ROOT`, `V2_BOUNDS`, `V2_PADS`. |
| `apps/go-api/internal/analysis/filmdec/vehicules_v2_items_test.go` | filmdec | les 4 items, le regroupement en amas, les chargeurs (corpus, bornes, pads). |
| `apps/go-api/internal/analysis/replay/vehicules_v2_deaths_test.go` | replay | `TestV2VehicleDeathDating` : items 4b/4c (mort-coincidente temporelle + spatiale + temoin). Gardes `V2D_FILMS`, `V2D_FILM_ROOT`, `V2D_BOUNDS`. |

Reutilisation stricte (regle des <= 2 copies) : les instruments s'appuient sur `vehProbe`,
`vehicleBuildCloud`, `vehicleCalibrateMPP` (V1b) et sur `indexBySlot`, `buildOwners`,
`bestDeathOffset`, `ScanFilmDeaths`, `ScanFilmPlayerIndices` (replay) — aucune grammaire ni aucun
calage recopie. `go vet` propre sur les deux paquets ; tous les runs PASS, EXIT=0.

Commandes de rejeu (avant-plan, GOCACHE isole) :

```
# Items 1-3 + i14, par carte (agregation par carte dans le test)
CGO_ENABLED=0 V2_FILM_ROOT=<repo>/data/cache V2_FILMS="<short8>:behemoth,..." \
  go test -C apps/go-api ./internal/analysis/filmdec/ -run '^TestV2SpawnsCooldowns$' -v -timeout 60m

# Item 4b/4c mort-coincidente (couteux : ~60-110 s/film ; decouper en lots de 6)
CGO_ENABLED=0 V2D_FILM_ROOT=<repo>/data/cache V2D_FILMS="<short8>:behemoth,..." \
  go test -C apps/go-api ./internal/analysis/replay/ -run '^TestV2VehicleDeathDating$' -v -timeout 30m
```

---

## 6. EXPLOITABLE EN PRODUCTION, ET LE POINT D'ENTREE A CREER (non ecrit)

### 6.1 Exploitable MAINTENANT

- **Catalogue des EMPLACEMENTS de spawn vehicule par carte** — fixes, exacts (rayon 0,00 m) :
  Behemoth 6 pads (grille 2x3), Launch Site 4-5 pads. Petit, stable, directement utile au rejeu 2D
  (dessiner un vehicule qui apparait a un pad).
- **Par vie de vehicule** : position de naissance (metres monde) + identite du chassis (`MPPWord32`,
  100 % lus) + intervalle de vie BORNE par le recensement (`[dernier vu, premiere image-cle apres]`,
  +/-20 s). Suffisant pour faire apparaitre/disparaitre un vehicule dans le calque 2D, SANS pretendre
  dater sa destruction.

### 6.2 Point d'entree de prod a creer (modele : `equipment_placements.go`)

Un fichier de production neuf `vehicle_placements.go` avec :

```
type VehiclePlacement struct {
    Life               EquipmentLifeKey // (slot, gen)
    T0US               uint64           // naissance (record de creation)
    CensusFirstUS/LastUS/GoneByUS uint64 // intervalle de vie BORNE (recensement) — jamais « detruit a T »
    X, Y, Z            float32          // position de naissance, MONDE (metres)
    ChassisGlobalID    uint32           // MPPWord32 -> tag `vehi` (nom via himap + vocabulaire damagetag, V1.5)
    TrajectoryPoints   int
}
func ScanFilmVehiclePlacements(dir string, wr *Vec3Range) ([]VehiclePlacement, VehiclePlacementStats, error)
```

- Reutiliser le gate durci de V1.5 (nuage + calibration MPP) et le recensement pour le bornage.
- Decoder les positions avec les BORNES REELLES de la carte (`wr`) pour un rendu monde natif (ou
  unite + projection, comme l'instrument).
- **Une couche analyse au-dessus** (hors `filmdec`) agrege les naissances en EMPLACEMENTS (leader
  clustering) : c'est le catalogue de pads par carte, le produit exploitable.

### 6.3 NON exploitable (et pourquoi)

- **Datation de la destruction** : i14 sous plancher ; mort-coincidente au hasard ; conducteur non
  attribue. Ne jamais ecrire « detruit a T ».
- **Confrontation .mvar** : le catalogue Forge ne porte pas de famille vehicule.

---

## 7. STATUT DES ITEMS

| item | statut | resultat |
|---|---|---|
| 1 emplacements par chassis | `[x]` mesure | 6 pads Behemoth / 4-5 Launch Site, rayon 0,00 m. Gate « <=4/chassis » : ECHEC Behemoth (6 pads), PASS Launch Site. Fait fort : pads fixes exacts. |
| 2 confrontation .mvar | `[~]` REPORTE | Repères alignes ; catalogue sans famille vehicule (piege Forge sur pieces). Gate NON APPLICABLE. Reprise = re-extraction .mvar (himap + fichiers .mvar). |
| 3 cooldowns | `[x]` mesure, **ECHEC** | Median ~35 s, IQR/mediane 0,87-0,98. Non resolu a +/-20 s. |
| 4a i14 detruit | `[x]` mesure, non concluant | 0,0019-0,0024 % (< plancher 0,17 %). |
| 4b/4c mort-coincidente | `[x]` mesure, **au hasard** | 52 % vs 48 % (Beh.) / 47 % vs 43 % (LS). Spatial : badauds (0-4 % a < 5 m) ; conducteur dans le trou embarque (V1a.4). |
| 4d classement detruit/ejection/despawn | `[!]` BLOQUE | Prerequis : attribution conducteur (V1a.4, non porte) + signal de destruction date (i14/sons). |
| BONUS sons explosion | `[~]` signale | 6 banques `sb_008_exp_vehicle_*`. |

## 8. CONDITIONS DE REPRISE (registre des reports)

- **Item 2 (.mvar)** : rouvrir quand une fenetre sans l'agent `himap` est disponible ET que les
  fichiers `.mvar` des cartes Forge sont accessibles — re-extraire le `type_id` du spawner de
  vehicule pour confronter aux 6 / 4-5 pads mesures.
- **Item 4 (destruction datee)** : depend de l'attribution du CONDUCTEUR (porter le modele
  « debut de trou » de V1a.4) puis d'un signal de destruction date. La mort-coincidente seule est au
  hasard et ne doit pas etre exploitee comme datation.
