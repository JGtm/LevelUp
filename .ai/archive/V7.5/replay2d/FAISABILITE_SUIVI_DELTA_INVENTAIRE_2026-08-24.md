# FAISABILITE — SUIVI DELTA DE L'INVENTAIRE (grenades, munitions, selecteurs)

> Etude du 2026-08-24. Branche `feat/v75`, aucun commit. Film de reference
> `data/cache/film_chunks/000d5950` (Cliffhanger, 49 chunks).
> Perimetre : ETUDE, pas d'implementation prod. Les deux sondes ecrites sont des tests de
> recherche gates par variable d'environnement, sautes en CI.

## VERDICT

**FAISABLE AVEC CE QU'ON SAIT DEJA — aucune retro-ingenierie nouvelle n'est requise pour les
grenades (i22), la selection de grenade (i47), le selecteur d'arme (i42), le chargeur (i30/i33)
et la reserve (i31/i34).**

La question qui fondait le doute — « le curseur moteur est-il reconstructible hors capture
Cheat Engine ? » — est **tranchee par la mesure : OUI, a 100 %**. Les identites du handoff
(`position = paquet.Start*8 + curseur_moteur`, `largeur = curseur(suivant) - curseur(courant)`)
ne sont PAS la seule voie d'acces : le depot possede depuis le lot i48 une **ancre offline**
(`matchBipedHeader`) et une **marche de composants par desers de production**
(`walkRecordTo`) qui reconstruisent le curseur sans aucune capture. La capture CE reste le
JUGE qui a servi a calibrer les largeurs ; elle n'est plus le CHEMIN de lecture.

Ce qui reste non resolu apres cette etude est **hors du chemin critique du suivi delta** : ce
sont les memes trous de NOMMAGE que le handoff du 2026-07-27 listait deja (quel compteur i22
= quel type de grenade ; quel rang i48 = quelle capacite). Le suivi delta n'en depend pas — il
suit des grandeurs deja nommees par le canal des images-cles, qu'il se contente de rafraichir.

---

## 1. ETAT DES PRIMITIVES — ce que filmdec sait DEJA faire sur les paquets delta

### 1.1 Le precedent qui marche (i48) n'est pas un motif, c'est une marche ECS complete

`filmdec/ability_rank.go: ScanFilmAbilityRanks` ne cherche AUCUN motif d'octets. Il fait,
pour chaque paquet de type `PacketTypeDelta` :

1. **Ancrage bit a bit** — `matchBipedHeader` (`offline_biped.go:305`) teste a chaque position
   la grammaire d'en-tete de record : prefixe `1`, slot de 13 bits appartenant a la bande des
   slots bipedes lus dans les images-cles (`bipedSlotBand`), tag `= 1`, deux bits nuls,
   compteur de masque dans `[4..7]`, indices de composants **strictement croissants a partir
   de 0**, puis porte d'i0 nulle. C'est un test de plusieurs dizaines de bits contraints : il
   ne demande rien au monde exterieur.
2. **Marche du curseur** — `walkRecordTo` (`ability_rank.go:188`) part de
   `i0 + lay.TotalBits() + i0TailBits` et consomme, dans l'ordre du masque, chaque composant
   via `consumeByName` (`traverse.go:194`), c'est-a-dire **les desers de production eux-memes**
   — 189 etiquettes `case`, largeurs portees depuis Ghidra et corrigees par la vraie
   verite-terrain (i0 a 47 bits, i25/i26/i27 rebattus, polarite de porte d'i30/i33 inversee).
   `paramForComponent` fournit le `param_4` mesure par composant.
3. Le deser lui-meme **publie** par un hook global (`SetAbilitySetHook`) : on ne relit jamais
   les bits a cote du deser, ce qui interdit qu'un second lecteur diverge.

**Il n'y a donc ni ancre par motif, ni table ECS separee dans ce chemin.** La « table ECS »
(`.ai/V7.5/film_re/PLAN_TABLE_ECS.md`, garde-rail `ecs_table_guard_test.go`) est un registre
DOCUMENTAIRE (1 067 lignes archetype x composant, 49 archetypes porteurs) : elle dit qui existe
et quel est son statut de portage. La grammaire bit-exacte reste dans `consumeByName`.

Le meme chemin sert deja i28 (camouflage), i54/i57 (capacites), i59 (grappin) et l'archetype
equipement ti=37 (`equipmentWalk.walk`).

### 1.2 Le curseur moteur EST reconstructible offline — recensement mesure

Sonde `filmdec/i22_delta_research_test.go: TestInventoryComponentsDeltaCensus`, film
`000d5950`, **49 chunks, 171 851 records biped delta ancres** :

| composant | nom du registre | annonce au masque | atteint par la marche |
|---|---|---|---|
| i22 | `unit-grenade-counts-component` | 120 | **120 (100,0 %)** |
| i25 | `unit-command-tick-component` | 171 842 | 171 842 (100,0 %) |
| i28 | `unit-active-camo-state-component` | 1 280 | 1 280 (100,0 %) |
| i30 | `weapon-state-ammo` (emplacement 0) | 740 | **740 (100,0 %)** |
| i31 | `weapon-state-rounds-inventory` (0) | 56 | **56 (100,0 %)** |
| i32 | `weapon-state-overheated` (0) | 2 875 | 2 875 (100,0 %) |
| i33 | `weapon-state-ammo` (emplacement 1) | 773 | **773 (100,0 %)** |
| i34 | `weapon-state-rounds-inventory` (1) | 43 | **43 (100,0 %)** |
| i42 | `biped-desired-weapon-set` | 245 | **245 (100,0 %)** |
| i43 | `weapon-state-type-info` (arme 0) | 14 | 14 (100,0 %) |
| i44 | `weapon-state-type-info` (arme 1) | 9 | 9 (100,0 %) |
| i47 | `biped-desired-grenade-set-component` | 64 | **64 (100,0 %)** |
| i48 | `biped-desired-ability-set-component` | 92 | 92 (100,0 %) |
| i54 | `biped-mobility-action-component` | 2 819 | 2 819 (100,0 %) |
| i57 | `biped-spartan-ability-component` | 1 286 | 1 253 (97,4 %) |

**Tout composant d'inventaire annonce au masque est atteint.** Le seul taux inferieur a 100 %
est i57, deja connu pour ses desyncs data-dependants.

### 1.3 La note « 92,46 % de comptes de grenades impossibles » est PERIMEE

`frame_records.go:81` et `grenade_events.go:7` disent qu'i22 lit 91-92 % de comptes impossibles
et concluent qu'il faut ancrer sur une constante plutot que derouler la chaine. **Cette mesure
appartient a l'autre chemin** (`DecodeFrameRecords`, marche non ancree sur la bande de slots)
et elle est ANTERIEURE aux correctifs d'i0 (47 bits), d'i25/i26/i27 et de la polarite d'i30/i33.
Sur le chemin ancre, mesure ce jour :

    compteur R(3) d'i22 : 4 dans 120 lectures sur 120 (100,00 %)
    valeurs R(8)        : {0, 1, 2} exclusivement — 3 valeurs distinctes sur 480 lectures

Le test refutable est celui que `unit_weaponstate.go` enonce lui-meme (« une lecture qui rend
count != 4 est la signature d'un curseur mal place ») : **il ne refute pas**. Un curseur au
hasard rendrait `count == 4` une fois sur huit et des octets uniformes sur 0..255.

Les deux commentaires perimes doivent etre corriges lors du portage (cf. lot 0 du plan).

---

## 2. LE PROTOTYPE — i22 dans les deltas, confronte aux images-cles

### 2.1 Ce que les deltas rendent : des EVENEMENTS de changement, coherents

Les 120 lectures se repartissent sur **56 slots** (des VIES, pas des joueurs) et sur
4 554,9 s .. 5 016,6 s — la meme horloge que les images-cles. Extrait, slot 516 :

    t+0,0 s   [0 1 0 0]
    t+27,5 s  [0 1 1 0]   <- ramassage rang 2
    t+27,6 s  [0 1 2 0]   <- ramassage rang 2
    t+40,4 s  [0 2 2 0]   <- ramassage rang 1
    t+59,5 s  [0 1 2 0]   <- lancer rang 1
    t+60,6 s  [0 0 2 0]   <- lancer rang 1
    t+72,8 s  [0 0 1 0]   <- lancer rang 2

Monotonie par pas de 1, jamais de saut : c'est un inventaire vivant, pas du bruit.

### 2.2 La confrontation aux images-cles : 97,2 %

Sonde `replay/i22_confrontation_research_test.go: TestI22Confrontation`. Regle refutable :
pour chaque image-cle dont les grenades sont lues, **la derniere lecture delta anterieure du
meme slot doit donner le meme quadruplet**.

    images-cles avec grenades lues                    120
    dont un delta anterieur existe sur le meme slot     36
    CONCORDANCE                                        35 / 36 = 97,2 %

L'unique divergence est instructive et non contradictoire :

    slot 569  image-cle t=4878,0 s -> [1 0 0 0]   delta t=4874,7 s -> [0 0 0 0]

Un ramassage de 3,3 s survenu entre les deux, dont le delta n'a pas ete lu : c'est un defaut de
RAPPEL de l'ancre (un record non apparie), pas un defaut de JUSTESSE. Les deux canaux, decodes
par des chemins sans etape commune (motif d'ancrage dans les keyframes contre marche ECS dans
les deltas), disent la meme chose.

### 2.3 Le gain de fraicheur, mesure

Age median de la derniere lecture connue, echantillonne a 1 Hz sur la duree de vie de chaque
slot :

    images-cles seules  11,0 s
    deux canaux fusionnes 7,0 s      (-36 %)
    lectures delta STRICTEMENT entre deux images-cles du meme slot : 44 sur 120

Le gain sur les grenades est **reel mais modeste** : i22 est rare (120 transmissions pour
171 851 records, 0,07 %). Le film ne transmet i22 qu'au CHANGEMENT — ce qui est exactement ce
qu'il faut pour un suivi, mais borne le gain a la frequence reelle des changements.

### 2.4 Le vrai gisement est ailleurs : les MUNITIONS

Sonde `filmdec/i22_delta_research_test.go: TestInventoryValuesDeltaProbe` (relecture des bits a
la position que la marche etablit, sans toucher aux desers) :

| grandeur | grammaire du deser de production | n | distribution mesuree |
|---|---|---|---|
| i30 chargeur | porte active-bas puis `R(8)` | 563 | min 1 · p50 30 · p90 72 · **max 80** |
| i33 chargeur | idem | 593 | min 1 · p50 24 · p90 70 · **max 80** |
| i31 reserve | `R(11)` | 56 | min 0 · p50 4 · p90 25 · max 80 |
| i34 reserve | `R(11)` | 43 | min 0 · p50 6 · p90 50 · max 240 |

**Ces valeurs sont calibrees de fait.** Le champ chargeur est un `R(8)` (0..255) : les
1 156 lectures tiennent toutes dans 1..80, sans une exception. Un curseur mal place produirait
une loi uniforme sur 0..255 ; la probabilite d'observer 1 156 valeurs consecutives sous 81 par
hasard est de l'ordre de `(81/256)^1156`, soit rigoureusement nulle. Meme argument pour la
reserve, `R(11)` (0..2047) dont 99 lectures tiennent dans 0..240. Le handoff classait ces
composants « largeurs mesurees, valeurs NON calibrees » : cette etude les calibre.

**Volume : 1 156 lectures de chargeur + 99 de reserve contre 120 lectures d'inventaire aux
images-cles.** C'est un facteur 10 sur la grandeur la plus visible d'une fiche joueur.

### 2.5 i47 et i42 confirment le handoff, et corrigent un detail

i47, `R(6)` masque puis `R(3)` selection (`consumeBipedDesiredGrenadeSet`, 9 bits a plat) —
64 lectures :

    selection = 0 : 20 lectures     -> « aucune selection », le codage est 1-base
    selection 1..4 : 44 lectures    -> appartient au masque dans 44 cas sur 44 (100,0 %)

Le test refutable du handoff (« la selection appartient toujours au masque ») **passe a 100 %
sur les selections non nulles**. La valeur 0 n'est pas une violation : c'est l'absence.

i42, 7 premiers bits, 245 lectures :

    99:92  97:55  17:34  19:32  49:11  51:7  29:3  81:3  83:3  117:3  37:1  61:1

Les deux valeurs dominantes separees d'un bit (99 et 97) que le handoff decrit sont
reproduites. Le SENS du bit reste non tranche — mais le canal des images-cles publie deja
`DrawnSlot` (0/1/2) et sert d'oracle gratuit pour le trancher.

### 2.6 Ce que le prototype NE dit pas

- **i43/i44 (identite de l'arme en main) sont quasi absents des deltas** : 14 et 9
  annonces sur 171 851 records. L'arme en main **ne peut pas** etre suivie par ce canal ; elle
  reste une lecture d'image-cle. Le suivi delta rafraichit les MUNITIONS d'une arme dont
  l'identite vient d'ailleurs.
- **Le rappel de l'ancre n'est pas mesure.** 171 851 records apparies, mais on ignore combien
  de records biped delta existent reellement. La divergence du slot 569 en prouve au moins un
  manquant. C'est le seul vrai risque residuel (cf. lot 3).
- **Un seul film.** Toutes les mesures portent sur `000d5950`.

---

## 3. VERDICT DETAILLE, PAR GRANDEUR

| grandeur | canal delta | statut | ce qui manque |
|---|---|---|---|
| Comptes de grenades (i22) | 120 lectures, 100 % plausibles, 97,2 % concordants | **FAISABLE** | rien |
| Selection de grenade (i47) | 64 lectures, 100 % coherentes avec le masque | **FAISABLE** | rien |
| Chargeur (i30/i33) | 1 156 lectures, bornees a 80/255 | **FAISABLE** | rien (calibre par cette etude) |
| Reserve (i31/i34) | 99 lectures, bornees a 240/2047 | **FAISABLE** | confrontation aux couples de la verite terrain (2/2, 6/6, 8/16, 12/12, 25/75) |
| Selecteur d'arme (i42) | 245 lectures, histogramme conforme | **FAISABLE, sens a trancher** | RE nulle : l'oracle `DrawnSlot` des images-cles suffit |
| Capacite portee (i48) | 92 lectures | **DEJA EN PROD** | le NOMMAGE des rangs (hors perimetre) |
| Arme en main (i43/i44) | 14 + 9 lectures | **NON FAISABLE par ce canal** | rien a faire : garder les images-cles |
| Nom du type de grenade par rang | — | inchange depuis le 2026-07-27 | verite terrain a l'image 250 (§8.1 du handoff) |

**Aucun blocage.** Aucune retro-ingenierie nouvelle sur le format n'est necessaire : les
desers sont deja portes, la marche les atteint, et les valeurs sont dans des bornes de jeu.

---

## 4. PLAN D'IMPLEMENTATION EN LOTS COURTS

Chaque lot se termine par un gate mesurable. Contrat : skill `plan-execution`.

### Lot 0 — Assainir la doctrine perimee (aucun code fonctionnel)

- 0.1 Corriger le commentaire de `filmdec/frame_records.go:81` : dire que les 92,46 % portent
  sur `DecodeFrameRecords` et qu'ils sont ANTERIEURS aux correctifs d'i0/i25/i30 ; citer la
  mesure du chemin ancre (120/120).
- 0.2 Idem pour `filmdec/grenade_events.go:7`, dont la justification (« la chaine de composants
  ne marche pas ») ne tient plus. Le decodeur d'evenements de lancer, lui, RESTE : il donne le
  TYPE lance et son auteur, qu'i22 ne donne pas.
- **Gate** : aucune ligne de code executable modifiee ; `go build ./...` vert.

### Lot 1 — Le scanner delta d'inventaire (grenades)

- 1.1 `filmdec/inventory_delta.go` : `ScanFilmInventoryDeltas(dir) ([]InventoryDelta, Stats, error)`,
  calque de `ScanFilmAbilityRanks` — meme ancre, meme marche, **un seul exemplaire** de la
  marche biped (regle des <= 2 copies : ne pas dupliquer `walkRecordTo`).
- 1.2 Publier i22 par le hook EXISTANT `SetGrenadeCountsHook` ; publier i47 par un nouveau hook
  dans `consumeBipedDesiredGrenadeSet` (le deser publie, on ne relit pas a cote).
- 1.3 Garde-rail : un test golden fige `count == 4` a 100 % et `valeur <= 2` a 100 % sur le film
  temoin ; une derive de largeur en amont le fait rougir.
- **Gate** : 120 lectures sur `000d5950`, 100 % plausibles.

### Lot 2 — Les munitions par delta

- 2.1 Hooks de publication dans `consumeWeaponStateAmmo` (chargeur R(8) + fraction R(12)) et
  `consumeWeaponStateRoundsInventory` (reserve R(11)), avec l'index d'emplacement.
- 2.2 Confronter aux `SlotAmmo` des images-cles (meme regle que §2.2) ET aux couples de
  `VERITE_TERRAIN_INVENTAIRE_2026-07-27.md`.
- 2.3 Garde-rail : chargeur borne a 255 par construction, **assertion a 100 % sous 120** sur le
  film temoin, dite comme une mesure et non comme une loi du format.
- **Gate** : concordance >= 95 % avec les images-cles sur les couples comparables.

### Lot 3 — Le rappel de l'ancre (le seul risque residuel)

- 3.1 Mesurer combien de records biped delta l'ancre MANQUE : comparer le nombre de records
  apparies au nombre rendu par `DecodeFrameRecords` sur les memes paquets, et diagnostiquer la
  divergence du slot 569 (record non apparie ou i22 non transmis ?).
- 3.2 Si le rappel est bas, la piste connue est l'avance du curseur apres appariement
  (`p = i0 + lay.TotalBits()`), qui saute le corps du record et peut manquer un record imbrique.
- **Gate** : un chiffre de rappel publie, pas une impression.

### Lot 4 — La fusion dans le document de rejeu

- 4.1 `replay/inventory.go` : fusionner les deux canaux sur la meme grandeur en publiant la
  SOURCE de chaque lecture — exactement le patron d'`abilities.go` (`AbilitySrcI48` /
  `AbilitySrcKeyframe`), qui est deja le remede eprouve contre « deux canaux, une seule
  etiquette ».
- 4.2 Bump de `SchemaVersion`, regeneration des types web, i18n FR+EN de tout libelle nouveau.
- 4.3 Le client doit continuer d'afficher l'AGE de la lecture : le passer de 11,0 s a 7,0 s ne
  le supprime pas.
- **Gate** : `make check-types`, `make test-web`, goldens du rejeu regeneres avec l'explication
  de chaque delta.

### Hors perimetre (a ne pas melanger)

Le NOMMAGE — quel compteur i22 est quel type, quel rang i48 est quelle capacite — reste le
chantier du handoff du 2026-07-27 §8.1 (confrontation a l'image 250). Le suivi delta n'en
depend pas.

---

## 5. SONDES ECRITES (jetables)

    apps/go-api/internal/analysis/filmdec/i22_delta_research_test.go
      TestI22DeltaResearch              lecture d'i22 dans les deltas + export JSON (I22_OUT)
      TestInventoryComponentsDeltaCensus recensement masque/atteint des 15 composants
      TestInventoryValuesDeltaProbe      valeurs d'i30/i31/i33/i34/i42/i47

    apps/go-api/internal/analysis/replay/i22_confrontation_research_test.go
      TestI22Confrontation               deltas contre images-cles + gain de fraicheur

Gates : `I22_FILM` (et `I22_DELTA_JSON` pour la confrontation). Sautees sans elles, CI comprise.
Lecture seule. Reproduction :

    cd apps/go-api
    CGO_ENABLED=0 I22_FILM=<repo>/data/cache/film_chunks/000d5950 I22_OUT=<tmp>/i22_delta.json \
      go test ./internal/analysis/filmdec/ -run '^TestI22DeltaResearch$' -v
    CGO_ENABLED=0 I22_FILM=<repo>/data/cache/film_chunks/000d5950 \
      go test ./internal/analysis/filmdec/ -run '^TestInventoryComponentsDeltaCensus$' -v
    CGO_ENABLED=0 I22_FILM=<repo>/data/cache/film_chunks/000d5950 \
      go test ./internal/analysis/filmdec/ -run '^TestInventoryValuesDeltaProbe$' -v
    CGO_ENABLED=0 I22_FILM=<repo>/data/cache/film_chunks/000d5950 I22_DELTA_JSON=<tmp>/i22_delta.json \
      go test ./internal/analysis/replay/ -run '^TestI22Confrontation$' -v

Si le plan du §4 est retenu, ces quatre sondes sont a SUPPRIMER au lot 4 : leurs mesures
deviennent des garde-rails de production (regle « 0 code mort »).
