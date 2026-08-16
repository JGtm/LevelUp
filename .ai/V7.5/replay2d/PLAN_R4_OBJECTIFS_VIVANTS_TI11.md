# Plan R4 — Les objectifs vivants (`ti=11`) : le crane, le drapeau, le noyau

> Ecrit le 2026-08-17. Lot R4 du `PLAN_RETOURS_PLANCHE_2026-08-16.md` (item F1 du bilan
> utilisateur : « Les objectifs le but c'est de les avoir aussi et de les symboliser aussi,
> surtout le crane d'oddball et le drapeau de CTF, ou le noyau de stockpile »).
> Worktree `LevelUp-wt-ti11`, branche `wt/ti11-objectifs`, base `feat/v75` = `3058afbba`.
> Contrat d'execution : skill `plan-execution`. Lot jumeau parallele : R3 (`ti=37`), voir
> le contrat d'interface en fin de document.

---

## 1. Objectif et critere de succes

> **AVERTISSEMENT DE LECTURE, ajoute le 2026-08-17 a la cloture.** Cette section a ete ecrite
> AVANT la mesure et sa premisse est FAUSSE : `ti=11` ne porte aucun composant de position, donc
> ni « la position monde par image » ni les controles C1/C3 n'ont d'objet sur cet archetype.
> Elle est conservee telle quelle, sans retouche, parce qu'un plan qui reecrit ses hypotheses
> apres coup ne s'audite plus — et parce que la comparer a la phase 1 montre exactement ce que
> la mesure a corrige. **Pour l'etat reel : §4 phase 1 (« le resultat de fond ») et §3 revise.**
> Ce qui SURVIT de cette section : la regle de non-publication (appliquee : rien n'a ete publie)
> et le controle fantome (applique : c'est lui qui a refute la voie delta).

**Objectif.** Publier dans l'artefact de rejeu, pour un match a objectif portable, **l'OBJET**
(drapeau, crane, noyau) — son type et sa position monde par image — et **son PORTEUR** (xuid,
avec la fenetre de portage). L'UI est HORS PERIMETRE : ce lot publie la donnee, il ne dessine
rien (`objectivesLayer.ts` sera etendu plus tard, par un lot web).

**Critere de succes, mesurable, avec denominateurs publies.** Un temps est reussi si, ET
seulement si, les trois controles ci-dessous sont rendus chiffres sur au moins deux films de
modes differents :

| # | controle | seuil de reussite | temoin independant |
|---|---|---|---|
| C1 | **EMPRISE** — les positions de l'objet tombent dans l'AABB du nuage de positions biped du meme film | >= 95 % | l'AABB vient des trajectoires joueurs, decodees par une autre chaine |
| C2 | **COINCIDENCE PORTEUR** — a l'instant d'un `flag_grabs` / `flag_steals` attribue au xuid X, la position de l'objet est plus proche de la Track de X que de celle d'un joueur tire au hasard | mediane portage < mediane temoin, ecart d'un facteur >= 3 | `objectiveevents.IdentifiedEvent` (statborg), chaine TOTALEMENT disjointe du decodage d'entite |
| C3 | **DISCRIMINANT DE MOBILITE** — un objet d'objectif porte BOUGE ; une bande de slots FANTOME de meme cardinalite (slots jamais vus porter `ti=11`) doit rendre nettement moins de signal | vraie bande > fantome sur (slots peuplés, echantillons) ET dispersion coherente | le fantome passe par le MEME code (regle du lot armes au sol) |

**Ce qui compte comme echec acceptable** : un negatif MESURE, ecrit au registre avec ses
chiffres et sa condition de reprise. Ce qui ne compte pas : un decodage « plausible » non
confronte a C1-C3.

**Regle de non-publication.** Aucun champ n'entre dans l'artefact tant que C1 et C2 ne sont pas
passes. Le precedent qui fonde cette regle est dans le depot : `GroundWeaponPositions`
(`filmdec/keyframe_ground_weapons.go:118-141`) rend des positions **non publiables**, et le dit
dans son propre en-tete, parce que le temoin fantome les a reduites a du bruit.

---

## 2. Etat des lieux — verifie sur pieces le 2026-08-17

### 2.1 Ce que le depot SAIT deja de `ti=11`

| fait | piece | portee |
|---|---|---|
| `ti=11` = archetype OBJECTIFS, 34 composants, couverture de dispatch **0/34** | `.ai/V7.5/PLAN_DETTE_AVANT_MERGE.md:413` | aucun composant d'objectif n'est lu aujourd'hui |
| Le traverseur s'arrete au **premier composant PRESENT non porte**, et un composant ABSENT du masque ne consomme **aucun bit** | `filmdec/traverse.go:1179-1225` (`if t.Mask&(1<<i) == 0 { continue }`, puis `t.DesyncAt = i; return`) | le « mur » depend du MASQUE REEL, pas de la liste de l'archetype |
| Le mur de `ti=11` est annonce **a i0**, pas a i4 | `filmdec/components_batch3.go:12-13` (« le traverseur s'arrete a i0 (traverse.go:1187) ») et `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md:25-27` (« Le SUIVI accusait `interaction-filter` en `i4` [...] **C'est faux.** [...] la traversee bute **des `i0`** ») | **le brief du lot cite le mur i4 : c'est la version PERIMEE.** A re-mesurer, c'est la phase 1 |
| Un lecteur de composant `ti=11` existe deja, garde sans appelant | `filmdec/components_batch3.go:19-42` (`consumeObjectiveFormattedText`, `//nolint:unused`, ti=11 i2 et i9) | condition de retrait explicite : « branchee ou retirée quand ti=11 sera decode » — ce lot est cette condition |
| `ti=11` est dispatche **162 fois en CTF, 0 fois en Strongholds** (sur 1 205 704 records) | `.ai/V7.5/film_re/RELEVE_TERRAIN_CAPTURES_2026-07-31.md:100-112`, `.ai/thought_log.md:10154` | le corpus utile est CTF/Oddball, pas les modes a zones |
| Films temoins : `64e8adfa` (Catalyst CTF) = **5 entites `ti=11` par image-cle**, `000d5950` (Slayer) = **0** | `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md` (tableau du temoin) | temoin negatif natif disponible |

### 2.2 L'ASYMETRIE DECISIVE avec les armes au sol (`ti=42`)

Le lot « armes au sol » a ete **refute** le 2026-08-12 (`REGISTRE_REPORTS.md:103`) : les
positions etaient du bruit. Deux causes structurelles y sont nommees. **La premiere ne
s'applique PAS a `ti=11`, et c'est verifie sur pieces :**

| cause de la refutation ti=42 | statut pour ti=11 |
|---|---|
| « au KEYFRAME, la position suit le default-state de l'archetype, dont la largeur n'est PAS resolue pour ti=42 (`defaultStateDeserByTI` n'a pas d'entree 42) » | **RESOLU pour ti=11** : `filmdec/default_state_arch.go:48` porte `11: consumeVersionPrefix` (FUN_14110d4d8, « V seul ») — l'offset d'i0 dans le record de keyframe est donc connu |
| « en DELTA, le record ne porte aucun typeIndex et la bande de slots comblee est contaminee par les archetypes voisins » | **s'applique toujours** — c'est le risque n°1 du lot, et c'est ce que le temoin fantome C3 doit trancher |

La condition de reprise ecrite au registre pour `ti=42` etait : « default-state de `ti=42`
resolu ». Pour `ti=11`, **cette condition est deja remplie a la base.** C'est la raison
technique pour laquelle ce lot n'est pas une redite du lot refute.

**Deuxieme asymetrie, semantique et au moins aussi forte** : le discriminant spatial qui a tue
`ti=42` etait « une arme posee NE BOUGE PAS » — donc une position qui bouge = du bruit. Pour un
objectif porte, **c'est l'inverse** : l'objet DOIT bouger, et il doit bouger **avec un joueur
identifie**. Le bruit et le signal ne se ressemblent donc pas, alors qu'ils se ressemblaient
pour `ti=42`. C'est ce qui rend C2 discriminant.

### 2.3 L'oracle, et le piege a eviter dans l'oracle

Les evenements d'objectif nommes et attribues existent deja et sont **decodes par une chaine
disjointe** (statborg, pas le walker d'entite) : `objectiveevents.IdentifiedEvent`
(`internal/analysis/objectiveevents/slotidentity.go:112`), avec `TimeMS` + `XUID` + `Stat`.
Statistiques utiles ici (`objectiveevents/named.go:125-134`) : `flag_grabs`, `flag_steals`,
`flag_captures`, `flag_returns`, `flag_carriers_killed`.

**PIEGE, deja mesure et NON CORRIGE dans le depot** (`REGISTRE_REPORTS.md:123`) : le calque
d'objectifs SERVI est decale de `originMs` (3,6 s a 50,8 s selon le match), parce que
`buildObjectiveActions` (`replay/objectives.go`) pose `t = TimeMS / interval` alors que `TimeMS`
compte depuis le PREMIER PAQUET DU FILM et la grille de frames depuis le PREMIER PAQUET DE
POSITION. **Consequence pour ce lot : l'oracle se prend sur `IdentifiedEvent.TimeMS` BRUT, et
l'origine s'applique explicitement dans mon instrument.** Ne jamais joindre sur
`doc.Objectives[].T`, qui porte le decalage. Corriger ce defaut est HORS PERIMETRE (regle 7 du
contrat) — il est deja au registre avec sa recette.

### 2.4 La machinerie reutilisable, et ce qu'il ne faut PAS refaire

| existe deja | fichier | usage ici |
|---|---|---|
| Reconstruction offline du World (slot -> typeIndex) depuis les keyframes | `filmdec/keyframe_world.go` | enumerer les entites `ti=11` sans Cheat Engine |
| Bande de slots d'un archetype, comblee puis nettoyee des slots vus porter un AUTRE archetype | `filmdec/keyframe_ground_weapons.go:114` -> `worldObjectSlotBand(dir, n, typeIndex)` — **deja parametree par archetype** | bande `ti=11`, sans nouvelle fonction |
| Positions monde des paquets delta pour une bande de slots donnee | `filmdec/keyframe_ground_weapons.go:157` `WorldObjectPositionsForBand(dir, wr, band)` — la bande est un parametre **precisement pour que le temoin fantome passe par le meme code** | C3 sans dupliquer le decodeur |
| Echantillon + recherche du plus proche dans le temps | `WorldObjectSample`, `NearestWorldObjectSample` (meme fichier) | appariement objet <-> instant d'evenement |
| Calque STATIQUE des objectifs (socles, zones) deja publie | `replay/map_objectives.go`, `replay/objectives_catalog.go` | **NE PAS REFAIRE** — ce lot ajoute le VIVANT, pas le statique |
| Actions d'objectif deja publiees (pulses) | `replay/objectives.go`, `document.go:293` | **NE PAS REFAIRE** |

`internal/analysis/` a ete verifie (skill `go-features`) : aucun algorithme d'objet d'objectif
vivant n'y existe. Le seul code `ti=11` du depot est le lecteur garde cite en 2.1.

---

## 3. Livrable EN DEUX TEMPS (exigence du brief, Notion 15.2)

> **REVISION DU 2026-08-17, apres le gate de la phase 1.** Le decoupage initial est conserve
> ci-dessous en tete, BARRE, parce qu'un plan qui reecrit son passe ne s'audite plus. La mesure
> qui l'a invalide est en §4, phase 1 (« le resultat de fond »).

### ~~Decoupage initial (INVALIDE)~~

> ~~TEMPS 1 : l'objet (position par image) via `WorldObjectPositionsForBand`, et le porteur par
> coincidence spatiale. TEMPS 2 : le porteur declare, lu dans la chaine de composants.~~
>
> **Pourquoi il tombe** : `ti=11` **ne porte aucun composant de position** (34/34 composants
> mesures, 0 positionnel). Le temps 1 n'avait pas d'objet. Et son chemin technique
> (`matchWorldObjectRecord`) ne reconnait pas les records `ti=11` (rapport reel/fantome 0,73x et
> 0,37x sur deux films). Les deux moities de la premisse etaient fausses, independamment.

### Le decoupage REEL, tel que la mesure l'impose

La frontiere n'est plus « avec ou sans position » : `ti=11` est le **descripteur d'objectif**
(le suivi affiche au joueur), et l'objet physique est une entite AUTRE qu'il designe par `i3`.
La frontiere est donc **ce que le descripteur dit de lui-meme** contre **ce vers quoi il pointe**.

#### TEMPS 1 — L'OBJECTIF : son type, son etat, sa progression

Tout est dans `ti=11` et n'exige aucune autre entite : `i5 type`, `i14 state`, `i12 progress` /
`i13 required-progress`, `i6 enabled`, `i7 priority`. C'est la matiere de « ou en est une
capture » (Notion 15.2), et c'est **la moitie de la demande utilisateur qui reste atteignable**.

**Mais elle exige de franchir le mur** — ce qui n'etait pas vrai dans le decoupage initial, et
c'est le vrai cout de la revision : avec 0/34 composants portes, meme `i5` est inaccessible.
Le temps 1 passe donc par la phase 4.

#### TEMPS 2 — L'OBJET PHYSIQUE : suivre `i3` vers l'entite referee

`i3 managed-objective-object-reference-component` designe l'entite du drapeau / crane / noyau.
La position, la trajectoire et le porteur vivent **la-bas**, dans un archetype a identifier.
Trois inconnues en chaine, chacune pouvant echouer : lire `i3` (mur), resoudre la reference en
slot d'entite, puis decoder la position de CET archetype-la.

**L'ordre est impose et le mur est commun** : les deux temps commencent par la phase 4. Un
negatif en phase 4 les ferme tous les deux — et c'est alors le resultat du lot.

---

## 4. Phases

> Regle d'ordre : une phase ne commence QUE si la precedente est CLOSE (gate passe + tous les
> items statues + plan a jour + commit). Statuts : `[x]` fait et verifie · `[~]` couvert
> ailleurs (avec la reference) · `[!]` non traite (avec justification ecrite). Aucune case vide
> a la cloture. Zero fix hors perimetre : tout ce qui est decouvert va en section 7.

### Phase 1 — MESURER le terrain de `ti=11` — **CLOSE le 2026-08-17**

Objet : remplacer les affirmations contradictoires du corpus documentaire (mur a i0 ? a i4 ?)
par une mesure datee, et etablir le corpus de travail.

- [x] 1.1 **Corpus** — mesure sur 3 films (`sonde_ti11_objectifs_test.go`, `TestTI11Corpus`) :

      | film | mode | chunks | keyframes | records de keyframe | records `ti=11` | slots `ti=11` |
      |---|---|---|---|---|---|---|
      | `64e8adfa` | CTF Catalyst | 44 | 43 | 13 654 | **201** | **10** |
      | `530820e5` | CTF Catalyst | 26 | 25 | 7 750 | **115** | **5** |
      | `000d5950` | Slayer | 27 | 26 | 7 825 | **0** | **0** |

      **Le temoin negatif natif passe** : le Slayer porte 0 entite d'objectif, sur 7 825 records
      de keyframe et 28 archetypes distincts. La detection d'archetype est donc saine.
      `ti=11` est un archetype RARE : 201 records sur 13 654 (1,5 %), loin derriere `ti=6`
      (2 049), `ti=17` (1 419), `ti=5` (1 376).
      **DECOUVERTE (voir §7)** : les slots sont **STABLES d'un film a l'autre** —
      `[1383 1399 1400 1415 1416]` sur les DEUX films CTF, `64e8adfa` en portant 5 de plus
      (`[3059 3075 3076 3091 3092]`, translation de +1 676). Tous < 8 192.
- [x] 1.2 **Les 34 composants de `ti=11`** lus dans le registre (`TestTI11Grammaire`, 118
      archetypes). **La couverture de dispatch mesuree est 0/34** — confirmee en interrogeant
      `consumeByName` lui-meme, pas une liste ecrite a la main. Les composants qui portent
      l'enjeu du lot :

      | i | composant | ce qu'il promet |
      |---|---|---|
      | i0 | `managed-objective-timers-component` | — |
      | i1 | `managed-objective-color-component` | camp / couleur |
      | i2 | `managed-objective-formatted-text-component` | **lecteur DEJA ECRIT** (`components_batch3.go:19`) |
      | **i3** | `managed-objective-object-reference-component` | **l'OBJET vise — le drapeau/crane** |
      | i4 | `managed-objective-interaction-filter-component` | le « mur » suppose |
      | **i5** | `managed-objective-type-component` | **le TYPE d'objectif** |
      | i12 / i13 | `progress` / `required-progress` | **la fraction de capture** |
      | i14 | `managed-objective-state-component` | **l'etat vivant** |
      | i16..i31 | `sub-objective-entities` ×16 | sous-objectifs |
      | i9 | `secondary-formatted-text` | jumeau d'i2 |

      **La contradiction du brief est TRANCHEE** : les NOMS du brief sont exacts (i3 =
      `object-reference`, i4 = `interaction-filter`), mais l'affirmation « i3 est avant le mur,
      donc accessible » est **FAUSSE**. Avec 0/34 composants portes, le premier composant
      PRESENT d'un record est deja non porte : le mur est au premier present, pas a i4. Le
      depot (`components_batch3.go:12`, `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md:25`) avait raison
      contre `PLAN_RETOURS_PLANCHE` §R4 et `SUIVI_REPLAY_2D.md:320-321`.
- [x] 1.3 **LE MASQUE REEL — REFUTATION MESUREE de la voie « objet du monde »**
      (`TestTI11MasqueEtBande`). Le reconnaisseur d'en-tete des objets du monde
      (`matchWorldObjectRecord`, `projectiles.go:283` — celui qui sert aux projectiles et a
      l'equipement) **ne reconnait PAS les records `ti=11`**. Trois controles concordants :

      | controle | `64e8adfa` | `530820e5` | lecture |
      |---|---|---|---|
      | records sur la bande OBSERVEE | 4 680 | 744 | — |
      | records sur le FANTOME de meme taille et **de meme voisinage numerique** | 6 421 | 2 037 | — |
      | **rapport reel / fantome** | **0,73x** | **0,37x** | **< 1 : la vraie bande rend MOINS que le hasard** |
      | records portant un index **hors grammaire** (> i33, impossible pour `ti=11`) | **45,9 %** | **36,4 %** | contre 38,4 % et 11,3 % au fantome |

      Le taux d'indices hors grammaire est le controle de purete le plus direct qui soit : un
      record d'objectif ne PEUT PAS porter i40. Pres d'un « record » reconnu sur deux en porte
      un. **Ce que le balayage trouve sur la bande `ti=11` est du bruit**, et pas meme autant
      de bruit qu'un tirage de controle.
      **Cause structurelle ecartee** : ce n'est PAS une limite de largeur de champ — les slots
      `ti=11` valent 1 383 a 3 092, tous representables sur les 13 bits de l'en-tete (mesure
      1.1). C'est la FORME de l'en-tete qui ne correspond pas.
- [x] 1.4 **Bande de slots** — et la regle de comblement s'avere INADAPTEE a `ti=11`, ce que la
      mesure montre : sur `64e8adfa`, 10 slots observes deviennent **891** apres comblement
      (**facteur 89,1x**), contre 5 -> 5 (facteur 1,0x) sur `530820e5`. Le comblement
      (`projectiles.go:365`) existe parce qu'un projectile vit moins d'une seconde et echappe
      aux keyframes espaces de ~20 s. **Un objectif est l'exact contraire : il vit toute la
      partie, il est present a CHAQUE keyframe, et l'observe est deja complet.** Combler ne
      recupere donc aucune couverture — ca n'avale que les voisins. L'instrument mesure
      desormais les DEUX bandes ; les deux sont refutees (0,73x et 1,37x contre leurs fantomes),
      l'observee etant la moins mauvaise.
- [x] 1.5 Instrument `filmdec/sonde_ti11_objectifs_test.go` (3 tests), garde `TI11_FILM`,
      saute partout ailleurs (verifie : 3 SKIP sans la variable), lecture seule, aucune
      ecriture disque. Le fantome est tire dans le MEME voisinage numerique que les slots reels
      — precaution ajoutee apres avoir constate qu'un fantome tire a partir du slot 1 gonflait
      le temoin (22 176 records au lieu de 6 421) et faussait la comparaison en defaveur du
      signal.

**Gate 1 : PASSE.** Commandes executees le 2026-08-17 (depuis `apps/go-api`, `GOCACHE` isole) :
```
CGO_ENABLED=0 go build ./internal/analysis/filmdec/ ./internal/analysis/replay/ \
  ./internal/analysis/objectiveevents/                          -> BUILD_EXIT=0
CGO_ENABLED=0 TI11_FILM=<...>/64e8adfa go test ./internal/analysis/filmdec/ -run TI11 -v
                                                                -> PASS (3 tests)
CGO_ENABLED=0 TI11_FILM=<...>/530820e5 go test ... -run TI11 -v  -> PASS
CGO_ENABLED=0 TI11_FILM=<...>/000d5950 go test ... -run TI11Corpus -v -> PASS (temoin 0)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -run TI11 -v  -> 3 SKIP (garde OK)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -count=1      -> ok (suite complete)
```

---

#### LE RESULTAT DE FOND DE LA PHASE 1 — `ti=11` NE PORTE AUCUNE POSITION, PAR CONSTRUCTION

C'est le fait qui reorganise tout le lot, et il ne demande aucune interpretation : sur les **34
composants** de l'archetype lus dans le registre, **34 commencent par `managed-objective-`, et
AUCUN ne porte de position** — pas un `position`, pas un `transform`, pas un `origin`, pas un
`location`, pas un `coord` (grep sur la sortie de `TestTI11Grammaire` : 0 occurrence).

Ce que `ti=11` porte, ce sont : des minuteurs, une couleur, deux textes formates, une reference
d'objet, un filtre d'interaction, un type, un booleen d'activation, une priorite, un type de
message, « nouveau et pas encore vu », « un seul element deverrouille », une progression et sa
cible, un etat, un objectif parent, seize entites de sous-objectif, une duree de phase de sortie
et un drapeau de mise a jour forcee.

**C'est le descripteur d'objectif du HUD — le suivi d'objectif affiche au joueur — et non
l'objet physique.** Le drapeau, le crane et le noyau sont des entites AUTRES ; `ti=11` ne fait
que **les designer**, par son `i3 managed-objective-object-reference-component`.

Trois consequences immediates, toutes structurelles :

1. **La demande « la position de l'objet d'objectif par image » ne peut PAS etre satisfaite par
   `ti=11`, quel que soit l'effort de reverse.** Ce n'est pas un mur qu'on franchit, c'est un
   champ qui n'existe pas dans cet archetype. Le brief du lot (et le critere de succes que
   j'avais ecrit en §1) reposaient sur une premisse fausse.
2. **La position devra venir de l'entite REFEREE**, donc d'un autre archetype — a identifier en
   suivant `i3`. Ce qui exige de franchir le mur : `i3` n'est lisible que si tous les composants
   presents qui le precedent le sont. On revient donc au temps 2, et le temps 1 tel que decrit
   n'a pas d'objet.
3. **En revanche, ce que `ti=11` porte a une valeur propre et elle est elevee** : `i5 type`,
   `i12/i13 progress / required-progress`, `i14 state` — c'est-a-dire **quel objectif est actif,
   ou en est sa progression, et dans quel etat il se trouve**. C'est exactement la matiere de
   « ou en est une capture » (Notion 15.2), et c'est independant de toute position.

**CONSEQUENCE SUR LE PLAN.**

- **La phase 2 est statuee `[!]`, non executee**, avec deux justifications independantes, l'une
  et l'autre suffisantes : (a) son entree technique est refutee par le gate 1 — elle passe par
  `matchWorldObjectRecord`, qui ne reconnait pas `ti=11` (rapport reel/fantome < 1) ; (b) son
  OBJET est inexistant — `ti=11` ne porte pas de position. Mesurer la dispersion de positions
  d'une bande dont les records ne sont pas reconnus, pour un archetype qui n'a pas de champ de
  position, serait exactement l'erreur que le lot des armes au sol a payee.
- **La phase 3 est statuee `[!]`** : elle depend de la phase 2 (le plan le declarait deja).
- **Le decoupage en deux temps est REECRIT** (voir §3, revision du 2026-08-17) : le temps 1 n'est
  plus « l'objet et sa position », il devient **« l'objectif : son type, son etat, sa
  progression »** — ce que `ti=11` porte reellement. Le temps 2 reste le mur, dont depend a la
  fois `i3` (la reference vers l'objet physique) et donc toute position future.
- La suite du lot bascule sur la **phase 4**, qui devient la voie unique : franchir le mur est
  desormais le prealable des deux temps, et non plus du seul temps 2.

### Phase 2 — TEMPS 1 : l'OBJET (position par image) — **`[!]` NON EXECUTEE, ENTREE REFUTEE**

Justification du `[!]`, deux motifs independants et chacun suffisant, tous deux etablis par le
gate 1 : (a) le chemin technique de la phase (`WorldObjectPositionsForBand` ->
`scanProjectileRecords` -> `matchWorldObjectRecord`) **ne reconnait pas les records `ti=11`**
(rapport reel/fantome 0,73x et 0,37x ; 45,9 % et 36,4 % d'indices hors grammaire) ; (b) son
OBJET n'existe pas — **`ti=11` ne porte aucun composant de position** (0 sur 34). Executer la
phase reviendrait a mesurer la dispersion du bruit pour un champ absent.

- [!] 2.1 Positions de la bande / de la bande fantome — non executee (motifs a et b).
- [!] 2.2 Controle C1 (emprise) — sans objet : pas de positions a encadrer.
- [!] 2.3 Controle C3 (mobilite) — sans objet.
- [!] 2.4 Verdict de phase — **rendu par anticipation au gate 1**, avec ses chiffres.

**Gate 2 : SANS OBJET.** La phase n'a pas ete ouverte.

### Phase 3 — TEMPS 1 : le PORTEUR par coincidence — **`[!]` NON EXECUTEE, DEPENDANCE MORTE**

Le plan la declarait ouvrable « seulement si le gate 2 est favorable ». Il ne l'est pas.

- [!] 3.1 Jointure objet <-> Tracks — sans entree (aucune position d'objet).
- [!] 3.2 Controle C2 contre l'oracle statborg — non executee. **L'oracle, lui, reste
      disponible et intact** (`objectiveevents.IdentifiedEvent`) : c'est l'objet a confronter
      qui manque, pas le temoin. C'est ce qui rendra la reprise peu couteuse.
- [!] 3.3 Fenetres de portage — sans entree.
- [!] 3.4 Verdict de phase — celui du gate 1.

**Gate 3 : SANS OBJET.**

### Phase 4 — TEMPS 2 : le mur de chainage — **CLOSE le 2026-08-17, NEGATIF MESURE**

Voie choisie apres la refutation de la voie delta : les records d'IMAGE-CLE sont **localises
exactement** par `WalkKeyframeWorld` (qui rend le bit de debut de chacun), donc sans aucune
reconnaissance de motif et sans faux positif possible — 201 records `ti=11` sur `64e8adfa`.
L'alignement teste est structurel et non devine : l'en-tete d'un record d'image-cle est
`[id:32][field:26][ti:6]` (`keyframe_world.go:19-22`) et `TraverseEntity` commence par lire
`R(6) typeIndex` (`traverse.go:1010`) — le champ `ti` de l'en-tete EST ce typeIndex, donc le
lecteur se pose a `Bit + 58`.

- [x] 4.1 Tentative de traversee des composants `ti=11` **REFUTEE PAR SON PROPRE TEMOIN**.
      Instrument `filmdec/sonde_ti11_mur_test.go`, temoin obligatoire = `ti=37` (equipement,
      **30/31 composants portes**, archetype deja decode en production) passe par le MEME code :

      | grandeur | TEMOIN `ti=37` (deja decode) | CIBLE `ti=11` |
      |---|---|---|
      | records d'image-cle | 797 | 201 |
      | `ti` relu != attendu | 0,0 % | 0,0 % |
      | **masque VIDE** | **65,7 %** | **73,6 %** |
      | **masque HORS GRAMMAIRE** | **33,2 %** | 1,5 % |
      | traversees ABOUTIES | 34,3 % | 1,5 % |

      **Le temoin echoue autant que la cible.** Sur un archetype dont 30 composants sur 31 sont
      portes, la traversee n'aboutit que dans 34,3 % des cas et un tiers des masques porte un
      index impossible. **Ce n'est donc PAS `ti=11` qui resiste : c'est la METHODE qui est
      fausse.** Conclusion refusee a l'intuition mais imposee par le temoin : le corps d'un
      record d'IMAGE-CLE ne se lit pas comme un record NEW pose a `Bit + 58`.
      Signature confirmant la mauvaise lecture cote `ti=11` : les seuls index « presents »
      trouves sont i0, i16, i32, i48 — des multiples de 16, et i48 est hors des 34 composants.
      C'est un motif d'alignement, pas de la donnee.
- [x] 4.2 **Preuve de marche : NON ATTEIGNABLE**, et pour une raison etablie sur pieces — la
      grammaire du CORPS d'un record d'image-cle **n'est resolue nulle part dans le depot**.
      Verification : les 6 appelants de `TraverseEntity` (`frame_records.go` x3,
      `frame_chain_infer.go` x2, `frame_debug.go`) sont **tous** sur le chemin FRAME/DELTA,
      **aucun** sur un payload d'image-cle. Et les deux lecteurs d'image-cle existants
      (`keyframe_loadout.go`, `keyframe_ground_weapons.go`) **evitent deliberement la grammaire**
      : ils balaient des identifiants de famille de 32 bits a l'interieur de l'emprise d'un
      record. C'est une confirmation independante — si la grammaire etait connue, ils
      l'utiliseraient.
- [~] 4.3 Confrontation porteur lu / porteur infere — **couvert par 4.1** : aucun porteur n'est
      lu, et la phase 3 n'en a infere aucun. Rien a confronter.
- [x] 4.4 **NEGATIF MESURE, ecrit.** Le mur de `ti=11` n'est pas un composant : c'est **deux
      murs superposes**, et aucun des deux n'est celui que le brief annoncait.

      1. **Aucune voie d'ACCES aux records `ti=11` n'est fiable aujourd'hui.** En DELTA,
         l'en-tete des objets du monde ne les reconnait pas (gate 1). En IMAGE-CLE, ils sont
         parfaitement localises mais la grammaire de leur corps est inconnue (4.1-4.2).
      2. **Meme l'acces resolu, les 34 composants restent a porter** (0/34), a commencer par le
         premier PRESENT — que cette mesure ne peut pas encore nommer avec certitude, puisque
         les masques lus sont faux.

      **Ce qui manque, nomme** : le deserialiseur du corps d'un record d'image-cle (la fonction
      qui suit l'en-tete 64 bits). C'est un travail de reverse (Ghidra), du meme ordre que celui
      qui a resolu `FUN_142f25e90` pour l'ancre du grappin — et il **beneficierait a tous les
      archetypes**, pas seulement aux objectifs : il debloquerait du meme coup la position des
      armes au sol (`ti=42`), reportee au registre le 2026-08-12 pour une cause voisine.

**Gate 4 : PASSE PAR LE NEGATIF** (le plan prevoyait explicitement les deux issues). Commandes :
```
CGO_ENABLED=0 go vet ./internal/analysis/filmdec/                     -> exit 0
CGO_ENABLED=0 TI11_FILM=<...>/64e8adfa go test ./internal/analysis/filmdec/ \
  -run TI11Mur -v                                                     -> PASS
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -run TI11 -v       -> 4 SKIP (garde OK)
```
Le negatif est chiffre, son temoin de controle est publie, et sa cause est nommee.

### Phase 5 — PUBLIER dans l'artefact — **`[!]` RIEN A PUBLIER**

La regle de non-publication du plan (§1) est appliquee telle qu'elle a ete ecrite : aucun champ
n'entre dans l'artefact tant que ses controles ne sont pas passes. Aucun ne l'est.

- [!] 5.1 `replay/objective_object_carriers.go` — **non cree**. Un fichier vide de sens serait
      du code mort (anti-pattern n°1 du depot).
- [!] 5.2 Champ optionnel sur `ReplayDocument` — **non ajoute**. `SchemaVersion` reste a **8**,
      conformement au contrat R3/R4 ; et ce lot **ne demande donc AUCUNE bosse** au superviseur
      a la fusion (information utile pour R3, qui peut en demander une pour son compte).
- [!] 5.3 `LayerCoverage` — sans objet.
- [!] 5.4 `go test ./contracttest/` — **non requis** : le compte gele des champs de
      `ReplayDocument` est INCHANGE, donc rien a mettre a jour. Verifie : aucun fichier de
      `internal/analysis/replay/` n'est touche par ce lot.
- [!] 5.5 Tests unitaires du nouvel assemblage — sans objet.

**Gate 5 : SANS OBJET.** Aucune modification de l'artefact ni du document.

### Phase 6 — Cloture — **CLOSE le 2026-08-17**

- [x] 6.1 Toutes les cases des phases 1-5 sont statuees, aucune vide.
- [x] 6.2 Lignes de registre redigees (§10), a verser en une seule fois.
- [x] 6.3 Entree `thought_log.md` redigee et remise au superviseur — ce lot n'ecrit PAS dans
      `thought_log.md`.
- [x] 6.4 Rapport final rendu, avec denominateurs, gates executes et ce qui reste.
## 5. Decisions produit — TRANCHEES avant execution (aucune a prendre en cours de route)

1. **L'UI n'est pas dans ce lot.** On publie la donnee. `objectivesLayer.ts` sera etendu par un
   lot web ulterieur. Aucun fichier `apps/web/` n'est touche.
2. **Un porteur INFERE se declare comme tel** dans l'artefact (methode + ecart), jamais comme une
   lecture. Le depot a deja paye ce principe (v6 de `SchemaVersion` : republier une autre
   grandeur sous la meme cle a coute un chantier entier — `document.go:56-66`).
3. **Rien n'est publie sans son temoin de controle.** Le fantome est obligatoire, patron du lot
   armes au sol.
4. **Aucun mode n'est etendu par analogie.** Ce qui est mesure en CTF vaut pour la CTF. L'Oddball
   et le Stockpile exigent leur propre mesure, ou sont declares non couverts. (Regle deja posee :
   `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md` etape 2.3.)
5. **Le decalage `originMs` du calque servi n'est PAS corrige ici** (hors perimetre, deja au
   registre) — mais il est CONTOURNE dans l'instrument, qui travaille sur `TimeMS` brut.
6. **Pas de CGO.** Les paquets `filmdec` / `replay` sont construits `CGO_ENABLED=0`. Si un gate
   exige CGO (DuckDB), il est signale au rapport et non force.

---

## 5 bis. Conformite architecture (passage de la grille `plan-review`, 2026-08-17)

### Couches (skill `arch-rules`)

| couche | ce que ce lot y met |
|---|---|
| `internal/analysis/filmdec/` | le decodage et les instruments de mesure (`sonde_ti11_*`) — algorithme pur, aucun acces DB/HTTP |
| `internal/analysis/replay/` | l'assemblage du calque (`objective_object_carriers.go`) — pur, sans I/O |
| `internal/service/`, `internal/port/`, `internal/api/handlers/` | **RIEN.** Le champ voyage sur `ReplayDocument`, deja construit et servi. Aucune interface nouvelle, aucun handler touche |
| `platform/duckdb/`, `persist/` | **RIEN.** Aucune lecture ni ecriture DuckDB (contrat R3/R4) |
| `apps/web/` | **RIEN** (decision 1) |

Seuils respectes : fichier <= 500 L, fonction <= 80 L, <= 5 parametres. `WorldObjectPositionsForBand`
prend deja 3 parametres ; si l'appariement objet/Track demande plus, les grandeurs sont
regroupees en struct plutot qu'ajoutees en parametres.

### Multi-titre — title-agnostic (regle transverse, ADR 0011/0025)

C'est le point que la premiere redaction de ce plan avait manque. La frontiere est **deja posee
dans le depot** et ce lot s'y range :

- Le **vocabulaire des roles d'objectif est un savoir DU TITRE**, pas du decodeur : il vit dans
  `config/titles/halo_infinite/mappings/objective_roles.toml`, dont l'en-tete l'ecrit
  explicitement (« la meme frontiere que replay_labels (ADR 0011). Un second titre apportera sa
  propre table, ou aucune »). Roles admis : `flag_spawn`, `flag_delivery`, `stockpile_socket`,
  `stockpile_navpoint`, `strongholds_zone`, `extraction_zone`, `oddball_spawn`, `assault_bomb`.
- `internal/analysis/replay/` **recoit** ses roles en parametre (`BuildMapObjectives(e, specs
  []ObjectiveRoleSpec)`, `map_objectives.go:84`) et ne lit jamais le slug. **Le type de l'objet
  vivant publie par ce lot suit la MEME regle** : il est nomme par le vocabulaire du titre, injecte,
  jamais un litteral « drapeau »/« crane » code en Go.
- **Aucune comparaison `slug == "..."`** (ratchet `no_slug_comparison_test.go`). Aucun libellé FR/EN
  en dur cote Go : l'artefact publie une CLE, les libelles suivent le patron `Label{En, Fr}` deja
  en place.
- Degradation gracieuse : un titre sans table de roles = aucun calque vivant, silence propre,
  jamais les objets d'un autre titre.
- **Reserve honnete** : `filmdec` est aujourd'hui du RE Halo Infinite (le walker keyframe le dit
  lui-meme, `keyframe_world.go:19-23` : « walker keyframe = RE Halo-Infinite-specifique pour
  l'instant »), et sa migration vers `internal/games/halo_infinite/film/` est un chantier deja
  identifie (plan audits 2026-07, item F12). Ce lot **n'aggrave pas** cette dette et ne la traite
  pas : il n'ajoute aucun nouveau litteral de titre.

### Logging (regle 3 du depot)

- Les instruments de mesure sont des TESTS : ils rendent par `t.Logf`, pas par `slog`. C'est le
  patron du depot (`ground_weapon_research_test.go`).
- Le code d'assemblage de `replay/` est PUR et sans contexte : il ne journalise pas, il rend sa
  couverture (`LayerCoverage`) — c'est la forme sous laquelle ce paquet rend deja compte de ses
  rejets, et elle est superieure a un log pour cet usage.
- **Aucun `fmt.Println` / `log.Printf`.** Si un chemin de degradation devait avaler une erreur, il
  la compte dans la couverture ; aucune erreur n'est ignoree en silence.

### Effort estime

| phase | effort | livrable independant ? |
|---|---|---|
| 1 mesure du terrain | moyen | oui — la mesure vaut meme si tout le reste echoue |
| 2 objet + fantome | moyen | oui |
| 3 porteur infere | moyen | non (depend de 2) |
| 4 mur de chainage | **lourd, risque eleve** — c'est le reverse | oui (un negatif mesure est un livrable) |
| 5 publication | rapide | non (depend de 2, et de 3 pour le porteur) |

Phases 1, 2 et 4 sont livrables independamment ; 3 et 5 portent leur dependance.

---

## 6. Contrat d'interface avec le lot jumeau R3 (`ti=37`, autre worktree, en parallele)

- Je ne CREE que mes fichiers : `*_ti11_*` / `*_objective_object_*` (deserialiseurs, tests,
  instruments), ce plan, et mes lignes de registre (a la fin, en une seule fois).
- Fichiers PARTAGES du decodeur — table de deserialisation par `ti`, registre de hooks,
  `build.go`, types du document, `SchemaVersion` : **ajout d'UNE ligne d'enregistrement chacun au
  maximum**, jamais de reecriture.
- **AUCUNE bosse de `SchemaVersion`.** La bosse est unique et faite a la fusion par le
  superviseur.
  **Champ ajoute par ce lot et pourquoi il exigera une bosse** : le champ vivant des objectifs
  est optionnel (`omitempty`), mais il suit la doctrine etablie aux versions v3 a v8
  (`document.go:24-78`) — un rendu client qui N'EXISTE que si l'artefact porte le champ, et une
  reprise de backfill qui se fait PAR `SchemaVersion`. Un artefact v8 doit donc se voir « a
  re-cuire », pas « a jour ». La bosse attendue est **v9** (a coordonner avec R3, qui peut
  demander la meme).
- Aucun run de masse, aucune re-cuisson d'artefacts publies (`data/`), aucune ecriture DuckDB.
  Lecture des films locaux seulement (`data/cache/film_chunks/`, worktree principal, LECTURE
  SEULE).
- Ordre de fusion prevu : **R3 puis R4**. Je me realigne sur R3 si les fichiers partages bougent.
- Cache Go isole : `GOCACHE=C:\Users\Guillaume\Projects\LevelUp-wt-ti11\.gocache` sur TOUTE
  commande `go`, une seule a la fois (le cache partage se corrompt sous deux `go` concurrents).

---

## 7. Decouvertes (a consigner, NE PAS traiter)

> Regle 7 du contrat d'execution. Toute decouverte hors perimetre entre ici et n'est pas corrigee.

- (2026-08-17, phase 1) **Les slots des entites `ti=11` sont STABLES d un film a l autre** :
  `[1383 1399 1400 1415 1416]` sur les deux films CTF mesures, `64e8adfa` en portant cinq de
  plus (`[3059 3075 3076 3091 3092]`, exactement +1 676). Un slot stable est un point d ancrage
  d identite potentiel (et une piste pour distinguer les deux groupes : deux equipes ? deux
  manches ?). NON EXPLOITE par ce lot — a verifier sur un troisieme mode avant d en faire quoi
  que ce soit, une stabilite vue sur deux films de la MEME carte peut n etre qu une propriete de
  la carte.

- (2026-08-17, phase 1) **La regle de comblement de bande de `projectiles.go:365` est inadaptee
  aux entites de longue duree de vie** : elle multiplie par 89 la bande de `ti=11` sur
  `64e8adfa`. Elle est JUSTE pour ce pour quoi elle a ete ecrite (les projectiles, invisibles aux
  keyframes) et fausse au-dela. Aucun appelant existant n est en cause — `GroundWeaponSlotBand`
  et les projectiles sont dans son domaine de validite. Rien a corriger, mais tout futur usage
  sur un archetype persistant doit s en tenir a l observe.

- (2026-08-17, avant execution) **Le brief du lot R4 cite le mur `interaction-filter` i4 ; le
  depot dit i0** (`components_batch3.go:12`, `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md:25`). La source i4
  est `SUIVI_REPLAY_2D.md:320-321`, explicitement refutee depuis. `PLAN_RETOURS_PLANCHE` §R4 a
  recopie la version perimee. A trancher par la mesure 1.3 ; le document fautif n'est pas corrige
  par ce lot.

---

## 8. Protocole de reprise de session

1. Relire le contrat `plan-execution` (skill), puis ce plan.
2. L'avancement fait foi ICI : cases de la section 4 + journal de la section 9.
3. Reprendre a la **premiere case non statuee de la phase courante**. Ne pas rouvrir les
   decisions de la section 5 : elles sont fermes.
4. Verifier sur pieces avant de coder ET avant de cocher (le code bouge — le lot jumeau R3
   travaille en parallele sur des fichiers partages).
5. Etat git : `git -C C:\Users\Guillaume\Projects\LevelUp-wt-ti11 log --oneline -10`.

---

## 9. Journal d'execution

_(rempli a la cloture de chaque phase : date, gate execute, sorties, commit)_

### 2026-08-17 — Phase 1 CLOSE (commit `af1acb125`)

Instrument `filmdec/sonde_ti11_objectifs_test.go` (3 tests, garde `TI11_FILM`). Gate 1 passe :
build exit 0, 3 films mesures, temoin negatif Slayer a 0 entite, 3 SKIP sans la variable, suite
`filmdec` complete verte. Deux resultats : la voie « objet du monde » en DELTA est REFUTEE
(rapport reel/fantome 0,73x et 0,37x, 45,9 % et 36,4 % d'indices hors grammaire), et surtout
**`ti=11` ne porte aucune position sur ses 34 composants** — c'est le descripteur d'objectif du
HUD, pas le drapeau. Phases 2 et 3 statuees `[!]`, decoupage en deux temps reecrit (§3).
Point d'etape rendu.

### 2026-08-17 — Phase 4 CLOSE par le NEGATIF (commit `6300e24e1`)

Instrument `filmdec/sonde_ti11_mur_test.go` (1 test, meme garde). Voie image-cle : les records
sont localises exactement, l'alignement `Bit + 58` est structurel. **Le temoin de controle
`ti=37` (30/31 composants portes) echoue autant que la cible** — 33,2 % de masques hors
grammaire et 34,3 % de traversees abouties seulement. Le temoin a donc fait son travail : ce
n'est pas `ti=11` qui resiste, c'est la grammaire du CORPS d'un record d'image-cle qui n'est
resolue nulle part dans le depot (verifie : les 6 appelants de `TraverseEntity` sont tous sur le
chemin delta ; les deux lecteurs d'image-cle existants evitent la grammaire et balaient des
motifs). Phase 5 statuee `[!]` — rien a publier, `SchemaVersion` reste a 8, aucune bosse
demandee au superviseur. Phase 6 close.

---

## 10. Lignes de registre proposees (a verser en une seule fois — §4 item 6.2)

A ajouter a `.ai/V7.5/REGISTRE_REPORTS.md`. Redigees ici pour eviter tout conflit d'edition avec
le lot jumeau R3, qui ecrit dans le meme fichier.

| sujet | lot / date | ce qui a ete mesure | condition de reprise |
|---|---|---|---|
| **[POST-v7.5] Objectifs vivants du rejeu 2D (`ti=11`) : objet porte, porteur, progression** | lot R4, phase 1 et 4, 2026-08-17 | **`ti=11` N'EST PAS L'OBJET, c'est le DESCRIPTEUR d'objectif du HUD** : ses 34 composants (lus au registre) sont tous `managed-objective-*` et **AUCUN ne porte de position** (0 occurrence de position/transform/origin/location/coord). Le drapeau/crane/noyau est une entite AUTRE, designee par `i3 managed-objective-object-reference-component`. La demande « position de l'objet par image » ne peut donc pas etre satisfaite par `ti=11`, quel que soit l'effort de reverse. Corpus mesure : `64e8adfa` 201 records / 10 slots, `530820e5` 115 / 5, `000d5950` (Slayer) 0 / 0 — temoin negatif natif OK. **Voie DELTA refutee** : `matchWorldObjectRecord` ne reconnait pas ces records — bande observee 4 680 contre 6 421 pour un fantome de meme taille ET de meme voisinage numerique (0,73x ; 0,37x sur le second film), 45,9 % et 36,4 % des « records » portant un index hors grammaire (> i33). Ce n'est pas une limite de largeur : les slots valent 1 383 a 3 092, tous representables sur 13 bits. **Voie IMAGE-CLE refutee PAR SON TEMOIN** : les records sont localises exactement (`WalkKeyframeWorld`) et l'alignement `Bit + 58` est structurel, mais le temoin `ti=37` — archetype couvert 30/31 — rend 33,2 % de masques hors grammaire et 34,3 % de traversees abouties. La methode est fautive, pas la cible. Couverture de dispatch `ti=11` confirmee **0/34** en interrogeant `consumeByName`. Rien n'a ete publie ; `SchemaVersion` reste a 8 | condition de reprise : **la grammaire du CORPS d'un record d'IMAGE-CLE** (le deserialiseur qui suit l'en-tete 64 bits `[id:32][field:26][ti:6]`), aujourd'hui resolue NULLE PART — les 6 appelants de `TraverseEntity` sont tous sur le chemin delta, et les deux lecteurs d'image-cle existants (`keyframe_loadout.go`, `keyframe_ground_weapons.go`) balaient des motifs plutot que de parser. **Ce meme deblocage sert `ti=42`** (position des armes au sol, reportee le 2026-08-12 pour une cause voisine) : un seul reverse, deux reports leves. ENSUITE seulement : porter les 34 composants a partir du premier PRESENT, en visant `i5 type`, `i12/i13 progress`, `i14 state` (l'objectif lui-meme) puis `i3 object-reference` (le chemin vers l'objet physique). Refutations reproductibles : `TI11_FILM=<film> go test ./internal/analysis/filmdec/ -run TI11 -v` |
| **Ce que `ti=11` PROMET une fois le mur tombe (a ne pas re-decouvrir)** | lot R4, phase 1, 2026-08-17 | Les 34 composants sont NOMMES et indexes dans `sonde_ti11_objectifs_test.go` / le plan R4. Les porteurs d'enjeu : `i5 type` (quel objectif), `i12 progress` + `i13 required-progress` (**la fraction de capture**, exactement « ou en est une capture » de Notion 15.2), `i14 state`, `i1 color` (camp), `i3 object-reference` (**le pont vers l'objet physique**), `i2`/`i9` textes formates — dont le lecteur `consumeObjectiveFormattedText` **est deja ecrit** (`components_batch3.go:19`, `//nolint:unused`, condition de retrait « quand ti=11 sera decode »). La progression d'objectif est donc atteignable SANS aucune position, et c'est la moitie de la demande utilisateur | meme condition que la ligne ci-dessus (grammaire du corps d'image-cle). Noter que `consumeObjectiveFormattedText` garde sa justification `//nolint` intacte : ce lot n'a PAS rempli sa condition de retrait |
| **Les slots des entites `ti=11` sont STABLES d'un film a l'autre** | lot R4, decouverte phase 1, 2026-08-17 | `[1383 1399 1400 1415 1416]` sur les DEUX films CTF mesures ; `64e8adfa` en porte cinq de plus, `[3059 3075 3076 3091 3092]`, exactement +1 676. NON EXPLOITE | verifier sur un troisieme mode (Oddball / Stockpile) et une AUTRE carte avant d'en faire un ancrage d'identite : une stabilite vue sur deux films de la meme carte peut n'etre qu'une propriete de la carte |
