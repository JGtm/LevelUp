# RAPPORT — lot V1 (suite) : CONDUCTEUR & VISEE i21 (vehicules)

> Execute le 2026-09-01 dans le worktree `LevelUp-wt-vehicules`, HEAD de depart `7011bdaed`.
> Aucun commit, aucun `git add`, aucune ecriture DuckDB. Toutes les mesures ont tourne en
> avant-plan, `CGO_ENABLED=0`, GOCACHE isole (scratchpad), sur les donnees reelles du checkout
> principal (`C:/Users/Guillaume/Projects/LevelUp/data`, lecture seule). Perimetre : les deux
> restes OUVERTS de la ligne V1 (V1a/V1b deja livres). Coordination : trois autres agents
> tournent ; ce lot n'ecrit QUE deux fichiers d'instrument neufs, sous garde d'environnement,
> et ne touche aucun fichier de production.

## 0. LE RESULTAT EN CINQ LIGNES

1. **ITEM 1 (CONDUCTEUR) — SIGNAL REEL, PARTIELLEMENT EXPLOITABLE.** La primitive « DEBUT DE
   TROU pres d'un vehicule » de V1a.4 est productionisee et rejoue a l'identique. Temoin fantome
   **NUL** (0 contre 12 et 14), signal a **x20 a x30 le hasard**. Attribution **15,6 % / 21,1 %**
   des vies, ambiguite **28,6 % / 8,3 %** (publiee, pas cachee). Elle NE nomme pas tous les
   vehicules, mais elle donne la **couleur d'equipe** des vehicules attribues.
2. **ITEM 2 (VISEE i21) — ABSENT, NON EXPLOITABLE.** i21 (`unit-desired-aiming-vector`) est a
   **0,00 % du masque** sur 81 540 records `ti=40` (0/32246, 0/5431, 0/36126, 0/7737), tres
   au-dessous du plancher de faux positifs de 0,17 %. Le meme deserialiseur, sur le bipede,
   capture i21 dans **48 a 55 %** des records : l'absence est reelle, pas un echec de decodage.
3. **POURQUOI c'est un negatif propre** : le vehicle-entity ne « vise » pas — son OCCUPANT le
   fait (i21 vit sur `ti=35`). L'histogramme complet du masque `ti=40` reproduit le cadrage
   § 1.4 (i0/i1/i2/i3/i25 presents, i34/i37 intermittents) et i21 n'y figure a aucun film.
4. **CONSEQUENCE « ou porte le regard » en 2D** : ni i2 (refute V1a.3) ni i21 (absent) ne donne
   la visee/l'orientation propre du vehicule. La seule orientation fiable d'un vehicule EN
   MOUVEMENT reste la direction de velocite `i1` (validee V1a.3, ecart median 1,7-2,1 deg). Un
   vehicule a l'arret n'a aucune orientation lisible dans `ti=40`.
5. **DEUX INSTRUMENTS NEUFS**, gardes par environnement, lecture seule, aucun code de production
   modifie. Ils dependent des instruments V0/V1a (corpus, bornes, oracle geometrique) et vivent
   avec eux.

---

## 1. ITEM 1 — ATTRIBUTION DU CONDUCTEUR

### 1.1 Ce qui a ete productionise, et la primitive

V1a.4 (§ 4.5) avait tranche : la bonne primitive n'est pas la coincidence prolongee (peu de
periodes) mais le **DEBUT DE TROU** du flux de position d'un bipede pres d'un vehicule —
l'occupant attache cesse de repliquer sa position monde, donc l'embarquement se voit comme le
dernier point d'un flux bipede pres d'un vehicule, suivi d'un silence.

Instrument : `apps/go-api/internal/analysis/replay/vehicules_v1_conducteur_test.go`,
`TestV1ConducteurAttribution`. Pour chaque VIE de vehicule (recensement images-cles `(slot,
gen)`, fenetre bornee `[premier, dernier] +/- 20 s`), on releve les debuts de trou (>= 3 s) dont
le dernier point est a moins de **1,5 m** du vehicule pendant la fenetre. Le(s) bipede(s) ainsi
retenu(s) sont les **occupants candidats**. Predicat de proximite : `attVehiculeLePlusProche`,
exactement celui de l'oracle geometrique du 18/08 — le signal reste comparable a V1a.4.

Reglages repris tels quels (chiffres incomparables sinon) : nuage vehicule par la grammaire
bipede (`v1aOptions(&wr, true)`, filtres de production), nuage bipede par `DefaultScanFilmOptions`
(`RequireTag1` arme), oracle et seuils (`attTrouMS = 3 s`, `attBordRayonM = 1,5 m`), temoin
fantome (`attBandeFantome`), denominateur de presence de fond (`v1aPresenceDeFond`).

### 1.2 Les seuils, ecrits avant la mesure

- **TEMOIN FANTOME** : une bande de slots jamais vus porter le moindre archetype rend, par le
  meme chemin, un nombre de debuts-de-trou pres d'un « vehicule » fantome **< 5 % du signal
  reel**. Un temoin au-dessus arrete l'item.
- **AMBIGUITE** : publiee, chiffree, jamais cachee (>= 2 occupants candidats distincts sur une
  vie).
- **CHANCE** : la presence de fond est le denominateur ; le signal est publie en rapport au
  hasard.

### 1.3 Les mesures (gate `0d76e8f1` + `4898d586`)

| film | vies recensees | dont vues >= 2x | pistes vehicule | trous >= 3 s | debuts de trou < 1,5 m | presence de fond | rapport / hasard |
|---|---|---|---|---|---|---|---|
| `0d76e8f1` Behemoth SF 8 j. | 45 | 28 | 19 | 26 | **12** | 2,3 % | **x20,3** |
| `4898d586` Behemoth SF 3 j. | 57 | 42 | 23 | 31 | **14** | 1,5 % | **x30,5** |

| film | attribution (vies avec >= 1 candidat) | ambiguite (>= 2 candidats) | TEMOIN FANTOME |
|---|---|---|---|
| `0d76e8f1` | **7 / 45 (15,6 %)** | 2 / 7 (28,6 %) | **0 / 12 — PASSE** |
| `4898d586` | **12 / 57 (21,1 %)** | 1 / 12 (8,3 %) | **0 / 14 — PASSE** |

**Reproduction de V1a.4** : sur `0d76e8f1`, 26 trous >= 3 s dont 12 pres d'un vehicule — exactement
les chiffres du rapport V1a (§ 4.4 : 26 trous, 12 a moins de 1,5 m). Le signal est donc le meme,
desormais rattache aux VIES.

Extrait de la table vie -> conducteur(s) candidat(s) (instants en horloge film) :

```
0d76e8f1  vie slot=768 gen=1 [2129..2249 s] : candidat [522]
0d76e8f1  vie slot=771 gen=1 [2129..2309 s] : candidats [512 514 515]  [AMBIGU]
0d76e8f1  vie slot=773 gen=1 [2129..2489 s] : candidats [551 554]      [AMBIGU]
0d76e8f1  vie slot=775 gen=1 [2129..2489 s] : candidat [555]
4898d586  vie slot=790 gen=1 [3842..4082 s] : candidat [579]
4898d586  vie slot=814 gen=1 [4122..4202 s] : candidats [601 606]      [AMBIGU]
```

### 1.4 Limite structurelle, ecrite car elle borne le livrable

`BipedPosition` **ne porte pas de generation** (rappel V1a.4 § 4.6) : les vies d'un vehicule sont
fusionnees par SLOT dans le flux de position. Le recensement (`SeenUS`, cle `(slot, gen)`) les
separe ; l'attribution rattache donc un evenement a la vie dont la fenetre bornee contient son
instant, mais elle designe fondamentalement un **SLOT a une fenetre de temps** — pas une vie au
sens strict. Le bornage du recensement est a **+/- une image-cle (~20 s)** : `SeenUS` BORNE, ne
DATE pas.

### 1.5 VERDICT ITEM 1 : **A INSTRUIRE — exploitable pour la couleur, pas pour l'exhaustivite**

- **Ce n'est pas refute** : temoin fantome nul, signal x20 a x30 le hasard, primitive datee du
  cote de l'embarquement.
- **Ce n'est pas exhaustif** : 15,6 a 21,1 % des vies attribuees. La primitive ne voit que les
  embarquements ou l'occupant repliquait sa position monde jusqu'au bord du vehicule ; un joueur
  deja a bord (flux deja eteint) ou qui monte hors d'un pas de replication est muet. Beaucoup de
  vies sont courtes (vue une seule fois : 17 sur 45, 15 sur 57).
- **L'ambiguite est le fait attendu d'un vehicule a plusieurs places** : conducteur + canonnier +
  passagers cessent TOUS de repliquer en embarquant. Les 8 a 29 % de vies a >= 2 candidats ne sont
  donc pas du bruit, ce sont des vehicules a plusieurs occupants.
- **POUR LA COULEUR D'EQUIPE (le but produit), c'est exploitable des maintenant** : l'occupant
  candidat est un SLOT de bipede, que le rejeu sait deja traduire en joueur -> equipe -> couleur.
  Plusieurs candidats de la MEME equipe ne creent aucune ambiguite de couleur ; seul un
  detournement (hijack cross-equipe, rare) le ferait, et il serait borne dans le temps.

### 1.6 Point d'entree de production a ecrire (sans l'ecrire ici)

Un service offline, tout en trame ECS (aucune RE), enchainant :
1. `filmdec.ScanFilmWorldObjectKeyframes(dir, 40)` -> vies bornees `(slot, gen)` + bande.
2. `filmdec.ScanFilmBipedPositionsForBand(dir, bande, opts vehicule)` -> pistes vehicule.
3. `filmdec.ScanFilmBipedPositions(dir, opts production)` -> flux bipede.
4. l'attribution « debut de trou < 1,5 m dans la fenetre de vie » (le corps de l'instrument).
5. la jointure **slot bipede -> equipe** deja detenue par le rejeu (roster / indices joueur).

Le livrable produit doit porter, par vehicule, la couleur d'equipe et **le taux d'ambiguite/de
non-attribution du match**, pour que l'UI degrade proprement (vehicule « equipe inconnue » quand
aucun candidat).

---

## 2. ITEM 2 — LA VISEE i21 SUR LE VEHICULE

### 2.1 Le piege de mesure, traite de front

`scanRecordDirs` (offline_aim.go) s'arrete au premier index de masque non modelise ; seuls
`i1/i2/i3/i4/i5/i21` sont geres. Or `i18/i19/i20` (unit-control/actor) sont < 21 et NON modelises :
un record qui les porte au masque AVANT i21 n'amene jamais le curseur jusqu'a i21. La **capture de
valeur** (`HasYaw`) SOUS-COMPTE donc la **presence au masque**. On mesure les deux :

- presence au masque via `filmdec.SetRecordMaskHook` (histogramme complet des index) ;
- capture de valeur via `HasYaw` sur les positions rendues.

Le decoupage i0 est FORCE (`opt.Layout`) pour que le seul balayage declenchant le hook soit le
notre (flux brut, hook 1:1 positions). Instrument :
`apps/go-api/internal/analysis/replay/vehicules_v1_visee_test.go`, `TestV1ViseeI21`.

### 2.2 Les seuils, ecrits avant la mesure

- **PRESENCE** : i21 « present dans le flux » si sa presence au masque depasse **1 %** des records
  `ti=40` acceptes (plancher de faux positifs 0,17 %, cadrage § 1.4).
- **CONCENTRATION** : une vraie visee a un tangage concentre pres de l'horizontale ; le bruit est
  uniforme. Part des tangages dans +/-45 deg : >= 60 % (visee reelle), ~25 % (90/360, champ
  uniforme). Reference : le bipede, valide.
- **CONTINUITE** : un cap de visee reel varie lentement ; mediane du pas |dyaw| (pas <= 2 s)
  < 45 deg ; **temoin par melange deterministe** > 70 deg (un decalage ne fait pas un temoin sur
  serie autocorrelee — lecon V1a.3.4).
- **MOUVEMENT** (informatif) : le cap de visee suit-il le deplacement (conducteur) ou en est-il
  independant (canonnier) ? Oracle de V1a.3, temoin par melange.

### 2.3 La mesure qui tranche : i21 ABSENT du flux `ti=40`

| film | records `ti=40` acceptes | i21 AU MASQUE | capture VALEUR | reference bipede (valeur) |
|---|---|---|---|---|
| `0d76e8f1` Behemoth | 32 246 | **0 (0,00 %)** | 0 | 101 103 / 207 720 (**48,7 %**) |
| `fccc61cd` Launch Site | 5 431 | **0 (0,00 %)** | 0 | 93 559 / 193 750 (**48,3 %**) |
| `4898d586` Behemoth | 36 126 | **0 (0,00 %)** | 0 | 117 405 / 213 930 (**54,9 %**) |
| `8a049c50` Behemoth | 7 737 | **0 (0,00 %)** | 0 | 84 881 / 155 866 (**54,5 %**) |

**Le hook lit de VRAIS masques** — l'histogramme complet du masque `ti=40` reproduit le cadrage
§ 1.4 et montre i21 absent au milieu de composants bien presents :

```
0d76e8f1 : i0=100,0% i1=98,5% i2=95,5% i3=95,9% i4=3,9% i25=99,9% i34=31,2% i37=2,7%   (i21 absent)
fccc61cd : i0=100,0% i1=97,9% i2=92,3% i3=93,3% i4=3,7% i25=99,9% i34=18,2% i37=4,1%   (i21 absent)
```

**Le meme deserialiseur, sur le bipede, capture i21 dans 48 a 55 % des records** (memes chemin,
meme code) : l'absence sur le vehicule n'est donc pas un echec de decodage, c'est une absence du
flux. Concentration, continuite et mouvement n'ont **aucun echantillon** a juger — ce vide EST la
reponse.

### 2.4 Pourquoi c'est coherent, et ce que ca implique

Le registre attribue bien i21 a l'archetype `ti=40` (cadrage § 1.2), mais le flux delta ne
l'emet jamais : le **vehicle-entity ne vise pas**, c'est son OCCUPANT (un bipede `ti=35`) qui
porte `unit-desired-aiming-vector`. La visee propre d'une TOURELLE de vehicule vivrait dans les
composants `i38 vehicle-weapon-set`, `i39 vehicle-auto-turret`, `i41/i42 seats-override-pitch/yaw`
(cadrage § 1.3) — tous **NON PORTES**.

### 2.5 VERDICT ITEM 2 : **ABSENT / NON CONCLUANT — non exploitable en l'etat**

« Ou porte le regard » en 2D **ne se lit pas** dans `ti=40` : ni i2 (refute V1a.3), ni i21
(absent). La direction fiable d'un vehicule **en mouvement** reste la velocite `i1` (validee
V1a.3 : ecart median 1,7-2,1 deg au deplacement) ; un vehicule a l'arret reste sans orientation.

Deux voies possibles pour une VRAIE visee de vehicule/canonnier, **toutes deux en RE, hors
perimetre V1** (a trancher par le superviseur) :

- **(a)** porter i38-i42 (weapon-set / auto-turret / seats-override) — un decodeur nouveau, pas un
  cablage ; a instruire seulement si un lot ulterieur reclame l'orientation de tourelle.
- **(b)** deriver la visee du CONDUCTEUR attribue (item 1) via son i21 de bipede — mais le bipede
  est muet tant qu'il est a bord (c'est la primitive meme du trou), donc cette voie ne donnerait
  la visee qu'avant l'embarquement et apres la descente, pas pendant la conduite.

---

## 3. FICHIERS

### Crees (instruments, gardes par environnement, lecture seule)

| fichier | contenu |
|---|---|
| `apps/go-api/internal/analysis/replay/vehicules_v1_conducteur_test.go` | `TestV1ConducteurAttribution` (item 1) : primitive debut-de-trou, attribution par vie bornee, temoin fantome, presence de fond, table vie -> candidat(s). |
| `apps/go-api/internal/analysis/replay/vehicules_v1_visee_test.go` | `TestV1ViseeI21` (item 2) : presence au masque (hook + histogramme) + capture de valeur + concentration + continuite + mouvement, reference bipede. |
| `.ai/V7.5/film_re/V1_CONDUCTEUR_VISEE_2026-09-01.md` | ce rapport. |

Aucun fichier de production modifie. Les deux instruments reutilisent `v0Corpus`/`v0Bornes`
(instrument V0), `v1aBandeVehicule`/`v1aOptions`/`v1aPistes`/`v1aPermutation`/`v1aEcarts`
(instrument V1a) et l'oracle geometrique `attVehiculeLePlusProche`/`attBandeFantome`
(`attachement_phase0_bord_test.go`, 18/08) : ils **vivent et meurent avec les instruments V0/V1a**.

`go vet` propre, `gofmt` propre sur les deux fichiers.

### Commandes de rejeu

```
# Item 1 — conducteur (~68 s pour 2 films)
CGO_ENABLED=0 GOCACHE=<isole> ATT_FILM=<depot>/data/cache \
  V0_FILMS="0d76e8f1:behemoth,4898d586:behemoth" \
  go test -C apps/go-api ./internal/analysis/replay/ -run TestV1Conducteur -v -timeout 30m

# Item 2 — visee i21 (~35 s pour 2 films)
CGO_ENABLED=0 GOCACHE=<isole> ATT_FILM=<depot>/data/cache \
  V0_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
  go test -C apps/go-api ./internal/analysis/replay/ -run TestV1Visee -v -timeout 30m
```

---

## 4. STATUT DES ITEMS

| item | statut | justification |
|---|---|---|
| 1 — attribution du conducteur (primitive debut-de-trou) | `[x]` mesure | Signal reel (x20-30 hasard, temoin fantome nul), gate PASSE sur les deux films. Attribution 15,6 / 21,1 %, ambiguite 28,6 / 8,3 % publiees. |
| 1 — verdict exploitabilite | `[x]` | A INSTRUIRE : exploitable pour la COULEUR d'equipe des vehicules attribues, pas pour l'exhaustivite. Point d'entree prod decrit (§ 1.6). |
| 2 — presence de i21 dans le flux `ti=40` | `[x]` mesure, **ABSENT** | 0,00 % au masque sur 81 540 records ; histogramme complet reproduit le cadrage ; reference bipede 48-55 %. |
| 2 — concentration / continuite / mouvement | `[~]` | Sans objet : 0 echantillon i21 sur le vehicule. Le vide est la reponse (couvert par la presence). |
| 2 — verdict exploitabilite | `[x]` | NON EXPLOITABLE en l'etat. « Ou porte le regard » = i1 (mouvement) seul ; visee de tourelle = i38-i42, RE hors perimetre. |

CR remis au superviseur. Aucun commit, aucun `git add`, aucun `thought_log`, aucune entree au
registre des reports — a la main du superviseur, comme pour V1a.
