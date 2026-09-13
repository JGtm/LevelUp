# RAPPORT F.0 — Le type 103 `EquipmentSpawnedObject`, références RÉSOLUES

Date : 2026-09-13. Item F.0 du `.ai/PLAN_FINITIONS_2026-09-13.md`. **Instruction pure, lecture
seule** : aucun fichier de production touché, aucune base DuckDB ouverte, aucune écriture sous
`data/`. Worktree `LevelUp-wt-finitions-equipement`, branche `feat/finitions-equipement`.

---

## Verdict en cinq phrases

**Les références du type 103 se résolvent : sa deuxième référence (`ref1`) désigne l'OBJET
ENGENDRÉ, index sur 13 bits à base 512 plus 2 bits de génération — 93,6 % des 925 références
présentes du parc tombent sur une vie d'entité réellement créée dans le film, contre 2,2 % pour
le témoin de hasard tiré dans le même domaine (facteur 42).** **Le 103 NE TIRE PAS À LA MORT :
sur 4 853 poses classées `dropped`, quatre sont désignées par un 103 (0,1 %), et 216 des 217
poses désignées du parc sont des PANNEAUX DE MUR.** **Mais il ne dit « déployé » que de la
famille qui ENGENDRE UNE PIÈCE : les 216 poses de panneau publiées sont désignées à 100 %, et
les 526 poses `deployed` de toutes les autres familles à 0 % — capteur 0/45, appareil de mur
0/31, écran 0/4, traqueur 0/3, champ de réparation 0/2.** **Les cas de D12 ne se tranchent donc
PAS sur le 103 : aucune des 15 poses classées `deployed` à l'image exacte de la fin de vie de
leur poseur n'est désignée par un 103, et aucune ne pouvait l'être — le film n'émet cet
événement pour aucun appareil porté, ni déployé ni lâché.** **Et la pièce engendrée cherchée
par D13 n'existe pas : sur le témoin POSITIF (le mur, `0x528fce46` à ×20,3 d'enrichissement
autour de ses consommations de charge), la méthode la trouve ; sur le capteur, le traqueur et
l'écran, elle ne trouve que l'objet PORTÉ lui-même — et le champ de réparation ne porte QU'UNE
consommation exploitable dans les 76 artefacts du parc, ce qui le rend non mesurable.**

---

## 0. Méthode, instrument, commandes

### 0.1 Ce qui est mesuré, et contre quoi

| Grandeur | Source | Nature |
|---|---|---|
| Occurrences du type 103 | marche de la **LISTE COMPLÈTE** (marcheur R7), tous les paquets delta | film |
| Références du 103 | `[1 porte][R(13) index][R(2) génération]`, domaines `{0,0,7}` de la table R7 | film |
| Vies d'objet d'équipement | `filmdec.ScanEquipmentCreations` — balayage **BRUT** `ti=37`, toutes catégories | film |
| Vies de projectile | `filmdec.ScanFilmProjectiles` — `ti=41` | film |
| Archétype d'un slot | recensement des **images-clés** (`WalkKeyframeWorld`, couple slot/ti) | film |
| Origine d'une pose (`deployed`/`dropped`/`unknown`) | **l'artefact déjà cuit** (schéma 54) | production |
| Consommations de charge (`spent`) | `equipmentChanges` de l'artefact, `gap = 0`, rang `from` nommé | production |

L'origine n'est **pas** recalculée ici : c'est une décision de production
(`replay.equipmentOrigin`), et la recalculer en ferait une seconde écriture qui divergerait au
premier correctif — celui que ce lot instruit.

### 0.2 Les témoins, écrits AVANT la mesure

1. **Témoin négatif de résolution** : les mêmes résolutions sur des références **tirées au
   hasard** dans le domaine de 13 bits, en même nombre et sous la même base, graine fixe
   `20260913`. Sans lui, un taux élevé pourrait n'être que la densité d'occupation du domaine.
2. **Deux bases mesurées côte à côte** (0 et 512). La base 512 n'est pas un réglage : R1
   l'avait établie sur le type 117, et c'est la mesure qui la retient ici.
3. **Témoin positif obligatoire de la question 4** : le **MUR**, dont la pièce engendrée est
   connue (`0x528fce46`, `0x686b40c9`, `kind = "deployed"` au manifeste). Si la méthode ne
   retrouve pas celle-là, elle ne prouve rien sur les autres et le dit.
4. **Témoin de hasard de la question 4** : le même nombre de fenêtres, posées à des instants
   tirés au hasard dans le film.

### 0.3 La carte de chaque film, et pourquoi elle n'est pas un point faible

La marche a besoin des étendues de la carte (les largeurs d'axe d'un vecteur quantifié en
dépendent). Elles sont obtenues en deux temps, **sans jamais consulter de base** :

1. `DetectI0Layout` **lit le découpage d'i0 dans le film lui-même** (profil de bascule par
   position de bit) — 25 films sur 25, sans erreur ;
2. parmi les seules entrées du catalogue de production dont `axisWidths` s'y accorde, l'**oracle
   de trame** (témoin 3 de R7) retient celle qui décode le plus loin.

Et pour tout ce que ce rapport juge — `(slot, génération, GlobalID, instant)` —, le balayage est
**invariant d'échelle** : la position déquantifiée vaut `min + (q+0,5)·étendue/2^w` et le rayon
d'accord de l'oracle vaut `mppCalibPosEps·étendue` (`EquipmentPosEps`), tous deux proportionnels
à l'étendue, `min` se simplifiant dans la différence. Deux cartes de mêmes largeurs d'axe
rendent donc le même jeu de records ; seules les coordonnées en mètres changent, et ce rapport
n'en lit aucune.

**Contrôle mesuré** (`TestF0InvarianceEchelle`) : chaque film est rebalayé avec les bornes d'une
carte JUMELLE — mêmes largeurs d'axe, bornes différentes — et les signatures
`(slot, génération, GlobalID, instant)` sont comparées.

| Film | Bornes de référence | Jumelle d'échelle | Signatures | Écarts |
|---|---|---|---|---|
| `1cd3848a` | Behemoth | Fragmentation | 394 / 394 | **0** |
| `46c3f91d` | Catalyst | Deadlock | 335 / 335 | **0** |
| `d1dfbc02` | Prism | Scarr | 320 / 320 | **0** |

(`9e8fb31b` skippe : aucune carte du catalogue ne partage les largeurs `[13 13 14]` de
Cliffhanger sans partager ses bornes — le témoin n'existe pas, et le test le dit.)

### 0.4 Instrument et commandes rejouables

Package `apps/go-api/internal/games/halo_infinite/film/filmdec/`, six fichiers `_test.go`,
skip par défaut (gardes d'environnement), `CGO_ENABLED=0` :

| Fichier | Rôle |
|---|---|
| `f0_103_contexte_research_test.go` | calibration de carte, balayages, marche, invariance d'échelle |
| `f0_103_artefact_research_test.go` | lecture de l'artefact et **jointure** artefact ↔ film |
| `f0_103_verdicts_research_test.go` | questions 1 à 3 |
| `f0_103_pieces_research_test.go` | question 4 |
| `f0_103_corpus_research_test.go` | recensement du parc d'artefacts (choix des films) |
| `f0_103_sonde_research_test.go` | la SONDE : valeurs brutes, refs et créations contemporaines |

Un changement ADDITIF au marcheur de R7 : `r7Ev.Refs` porte désormais les **trois** références
avec leur génération (`r7Refs3`). La marche est inchangée bit pour bit — la lecture consommait
déjà les trois références, elle n'en rendait qu'une.

```bash
cd apps/go-api
export REPO=<dépôt>            # LevelUp-go-migration (lecture seule)
export WT=<worktree du lot>    # LevelUp-wt-finitions-equipement
export CGO_ENABLED=0
export F0_ROOT="$REPO/data/cache/film_chunks"
export F0_ARTS="$REPO/data/cache/replays/halo_infinite"
export F0_CAT="$WT/data/titles/halo_infinite/reference/map_quant_bounds.json"
export F0_LABELS="$WT/config/titles/halo_infinite/mappings/replay_labels.toml"

# 0. quels films portent des consommations exploitables, et lesquelles (artefacts seuls, 1 s)
go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0CorpusSpent$' -count=1 -v

# 1. la carte de chaque film (R7_CHUNKS borne le balayage ; 8 suffisent)
R7_CHUNKS=8 F0_IDS=<liste> \
  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0CalibreCarte$' -count=1 -v
# -> recopier la ligne `F0_MAPS=...` de la sortie

# 2. questions 1 à 3 (25 films : ~2 min)
F0_IDS=<liste> F0_MAPS=<sortie de l'étape 1> \
  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0Deploiement103$' \
  -count=1 -timeout 90m -v

# 3. question 4
F0_IDS=<liste> F0_MAPS=<idem> \
  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0PieceEngendree$' \
  -count=1 -timeout 90m -v

# 4. contrôle d'invariance d'échelle
F0_IDS=<un film> F0_MAPS=<idem> \
  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0InvarianceEchelle$' -count=1 -v

# 5. la sonde (valeurs brutes, pour re-instruire une conclusion)
F0_SONDE_N=8 F0_IDS=<un film> F0_MAPS=<idem> \
  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0Sonde103$' -count=1 -v
```

### 0.5 Le corpus

Les **10 films imposés par le plan** — les 3 à oracle Theater (`000d5950`, `1cd3848a`,
`215e7022`) et les 8 films des cas de D12 (`5dfdc63b`, `4f77afc1`, `1cd3848a`, `46c3f91d`,
`8a485699`, `9e8fb31b`, `bfcd1175`, `fccc61cd`) — plus **15 films choisis par le recensement**
du parc (§4.1), qui portent les consommations de charge des familles que D13 vise. **25 films,
5 761 poses publiées appariées, 931 occurrences du type 103.**

`000d5950` et `215e7022` n'ont **pas** d'artefact dans le cache local : ils entrent dans la
question 1 (qui n'en a pas besoin) et sortent des questions 2 à 4. C'est dit, pas masqué.

---

## 1. Question 1 — le 103 en liste complète, et ses références résolues

### 1.1 Ce que la marche voit

Sur les 10 films imposés :

| Film | Carte | Listes non vides | Fermées proprement | Occurrences 103 | dont en tête |
|---|---|---|---|---|---|
| `000d5950` | Cliffhanger | 3 508 | 97,8 % | 46 | 46 |
| `1cd3848a` | Behemoth | 5 571 | 95,5 % | 40 | 40 |
| `215e7022` | (canevas Forge) | 5 937 | 97,5 % | **0** | 0 |
| `5dfdc63b` | Forest | 3 554 | 98,2 % | 36 | 36 |
| `4f77afc1` | (canevas Forge, BTB) | 17 441 | 98,4 % | 33 | 30 |
| `46c3f91d` | Catalyst | 3 180 | 98,6 % | 39 | 39 |
| `8a485699` | Behemoth | 5 108 | 95,1 % | 40 | 40 |
| `9e8fb31b` | Cliffhanger | 3 372 | 98,5 % | 25 | 25 |
| `bfcd1175` | Recharge | 3 855 | 98,4 % | 63 | 62 |
| `fccc61cd` | Behemoth | 4 839 | 95,3 % | 49 | 49 |

**La LISTE COMPLÈTE n'ajoute presque rien au recensement des TÊTES pour ce type** : 927 des 931
occurrences du parc sont en position 1. R5 mesurait 46 têtes sur `000d5950` ; la marche complète
en trouve 46. **Le gain de la marche n'est donc pas le NOMBRE d'événements — c'est le CADRAGE
des références**, que la lecture de tête seule ne permettait pas de décoder (R5 §3.1 en était
resté à « ref0 ≈ objet 13 bits, ref1 ≈ objet 13 bits », jamais résolues).

### 1.2 Les trois références, base par base (parc, 25 films)

| Base | Réf | Présente | Vie `ti=37` | Vie `ti=41` | **TOTAL RÉSOLUE** | dt médian | Archétype dominant |
|---|---|---|---|---|---|---|---|
| 0 | ref0 | 929/931 | — | — | 1,9 % | — | ti=35 (bipède) |
| 0 | ref1 | 925/931 | 3,6 % | 3,1 % | **6,7 %** | +229 961 ms | ti=35 (bipède), 292 |
| **512** | ref0 | 929/931 | 2,8 % | 0,0 % | 2,8 % | +31 399 ms | **ti=37, 737 sur 739 recensés** |
| **512** | **ref1** | 925/931 | **231 (25,0 %)** | **635 (68,6 %)** | **93,6 %** | **+49 ms** | ti=41 226 · ti=37 196 |
| 512 | ref2 | **3/931** | — | — | — | — | — |
| *témoin* | ref0 | 929/929 | 0,8 % | 0,5 % | **1,3 %** | — | dispersé |
| *témoin* | **ref1** | 925/925 | 0,8 % | 1,4 % | **2,2 %** | — | dispersé |

**Verdict Q1 : OUI pour `ref1`, qui désigne l'OBJET ENGENDRÉ.**

- **93,6 % contre 2,2 % au témoin** : facteur 42. Sous l'hypothèse nulle « la référence ne
  désigne rien », observer ce taux est hors de portée du hasard.
- **La base 512 est la bonne, et la mesure la départage seule** : 93,6 % contre 6,7 % à base 0,
  et à base 0 l'archétype dominant est le **bipède** (le décalage fait retomber dans la bande
  des slots de joueur) alors qu'à base 512 il est l'objet du monde.
- **La datation est chirurgicale** : dt médian **+49 ms** (bornes +32 à +19 386 ms, la queue
  venant des rares collisions de clé `(slot, génération)` rebouclée). L'événement suit la
  création d'une à deux images.
- **L'objet désigné est nommé** : `0x528fce46` 227 fois, `0x686b40c9` 3 fois — **les deux
  panneaux de mur du manifeste, les deux seuls objets `kind = "deployed"`** — et `0x412000aa`
  une fois. Les 68,6 % restants sont des **projectiles** (`ti=41`) : le 103 est donc bien
  « un objet a été engendré », pas « un équipement a été déployé ».

**Verdict Q1 pour `ref0` : NON, ce n'est pas l'équipement source au sens d'une vie créée.** Elle
est présente 929 fois sur 931, elle désigne un slot que les images-clés voient porter
l'archétype `ti=37` **737 fois sur 739 recensées** (témoin : 17 sur 95) — donc bien un objet
d'équipement —, mais elle ne se résout à **aucune création delta** (2,8 %). Lecture mesurée :
`ref0` désigne un objet `ti=37` **présent aux images-clés et jamais créé dans un paquet delta**,
c'est-à-dire une entité de longue durée ; ce n'est **pas** l'appareil porté par le joueur, dont
les créations delta existent et sont dans la table. Ce que c'est exactement n'est pas établi par
ce lot, et n'est pas nécessaire aux questions 2 à 4.

**`ref2` est absente** : 3 portes posées sur 931. C'est la confirmation, sur 931 occurrences, du
décodage manuel de deux têtes fait par R5 §3.1 (« ref2 absente »).

### 1.3 Un film à ZÉRO occurrence

`215e7022` (Argyle, le film à relevé Theater du répulseur) porte **0 occurrence du type 103**
sur 5 937 listes non vides. Cohérent : c'est un film sans mur déployé. Rappel de la règle du
dossier — **le répulseur est un négatif MESURÉ, il ne se recherche pas à nouveau** ; ce film
n'est ici que comme témoin de marche.

---

## 2. Question 2 — le 103 tire-t-il à la MORT ?

Pour chaque pose publiée (jointure artefact ↔ film, §2.1), une référence d'un 103 du film
désigne-t-elle sa vie ?

### 2.1 La jointure, et son contrôle

Clé `(t0, GlobalID)`, les deux côtés venant du même balayage. Sur les 10 films imposés :
**2 747 poses appariées, 8 poses d'artefact sans correspondance, 8 poses de film non publiées**
— 99,7 % de jointure. Le reste est le filtre d'axe de `buildEquipmentPlacements` (une pose hors
de l'axe de frames publié n'est pas publiée). La mesure repose donc sur une jointure quasi
totale, et le chiffre est publié plutôt que supposé.

### 2.2 Par ORIGINE (parc, 25 films, 5 761 poses)

| Origine | Désignées par un 103 | Poses | Taux |
|---|---|---|---|
| `deployed` | 209 | 526 | **39,7 %** |
| `dropped` | **4** | **4 853** | **0,1 %** |
| `unknown` | 4 | 382 | 1,0 % |

### 2.3 Par OBJET — et c'est là que tout se joue

| Objet (manifeste) | `deployed` | `dropped` | `unknown` |
|---|---|---|---|
| **`wall/0x528fce46`** (PANNEAU, `kind=deployed`) | **206 / 206 = 100 %** | **3 / 3 = 100 %** | **4 / 4 = 100 %** |
| **`wall/0x686b40c9`** (PANNEAU, `kind=deployed`) | **3 / 3 = 100 %** | — | — |
| `wall/0x8e2dc574` (appareil porté) | 0 / 31 | 0 / 145 | 0 / 6 |
| `wall/0x2974c233` (appareil porté) | 0 / 3 | 0 / 2 | 0 / 1 |
| `sensor/0x72199cba` | 0 / 45 | 0 / 276 | 0 / 12 |
| `sensor/0x72b63d69` | 0 / 3 | 0 / 15 | — |
| `threat_seeker/0x4744d742` | 0 / 3 | 0 / 16 | — |
| `repair_field/0x32d97758` | 0 / 2 | 0 / 9 | — |
| `shroud_screen/0x4396db42` | 0 / 4 | 0 / 32 | 0 / 4 |
| `translocator_beacon`, `repulsor`, `grapple`, `thruster`, les 4 grenades, les 2 bonus | 0 partout | 0 partout (sauf 1 grenade frag sur 1 626) | 0 partout |

**Verdict Q2, en trois temps :**

1. **Le 103 ne tire PAS à la mort.** 4 poses désignées sur 4 853 `dropped` — et **3 de ces 4
   sont elles-mêmes des panneaux de mur** (un panneau classé `dropped` par `equipmentOrigin`,
   ce qui est un défaut d'origine, pas un tir à la mort). La quatrième est une grenade sur
   1 626 : le bruit d'une collision de clé `(slot, génération)`, la génération ne faisant que
   2 bits. **L'affirmation de R5 §3.2 — « 90 têtes vers un dropped », donc « le 103 tire aussi
   à la mort » — est RÉFUTÉE.** Elle reposait sur un appariement en TEMPS SEUL à ±1,2 s, sans
   référence résolue ; avec la référence, la coïncidence temporelle disparaît.
2. **Le 103 EST le fait « une pièce a été engendrée ».** Les 216 poses de panneau publiées du
   parc sont désignées **à 100 %**, dans les trois origines ; 216 des 217 poses désignées sont
   des panneaux.
3. **Mais il n'existe que pour la famille qui engendre une pièce.** 0 sur les 317 poses
   `deployed` de toutes les autres familles — dont **0 sur les 91 poses `deployed` d'un
   DÉPLOYABLE PORTÉ** : appareil de mur 34, capteur 48, écran occultant 4, traqueur 3, champ de
   réparation 2. **Le film ne porte aucun signal d'événement pour le déploiement d'un capteur,
   d'un traqueur, d'un écran ou d'un champ de réparation.**

C'est l'exacte confirmation, par un canal INDÉPENDANT (les événements nommés), de ce que la
mesure E0 du 2026-09-10 avait établi par les consommations de charge : *le canal des poses ne
voit le déploiement que d'UNE famille, celle qui engendre une pièce distincte.* Les deux chaînes
n'ont aucune étape commune.

---

## 3. Question 3 — les cas de D12

### 3.1 Ce que le parc porte aujourd'hui

Critère appliqué : pose classée `deployed` dont la frame `t0` est **exactement** la dernière
frame d'une vie de son poseur (`tracks[].endFrame` de l'artefact). Sur les 25 films : **15 cas**,
dont **8 sont l'appareil de mur `0x8e2dc574`**.

| Film | t0 | Objet | Famille | Poseur | Un 103 désigne sa vie ? |
|---|---|---|---|---|---|
| `1cd3848a` | 4 081 | `0x8e2dc574` | wall | 581 | **non** |
| `1cd3848a` | 4 356 | `0x8c77ffe7` | grapple | 584 | non |
| `5dfdc63b` | 909 | `0xeef5d48d` | thruster | 518 | non |
| `5dfdc63b` | 1 253 | `0x8c77ffe7` | grapple | 530 | non |
| `4f77afc1` | 831 (×2) | `0xbcabbe43` | grenade_frag | 525 | non |
| `4f77afc1` | 2 454 (×2) | `0xbcabbe43` | grenade_frag | 584 | non |
| `46c3f91d` | 1 467 | `0x8e2dc574` | wall | 538 | **non** |
| `8a485699` | 6 191 | `0x8e2dc574` | wall | 603 | **non** |
| `9e8fb31b` | 4 088 | `0x8e2dc574` | wall | 598 | **non** |
| `bfcd1175` | 5 277 | `0x8e2dc574` | wall | 607 | **non** |
| `fccc61cd` | 3 624 | `0x8e2dc574` | wall | 572 | **non** |
| `0797ce72` | 1 195 | `0x8e2dc574` | wall | 539 | **non** |
| `bc60b4d9` | 1 924 | `0x4744d742` | threat_seeker | 545 | non |

**Écart avec la liste de E0, et il est dit.** E0 (2026-09-10) nommait huit films pour
`0x8e2dc574` : `5dfdc63b`, `4f77afc1`, `1cd3848a`, `46c3f91d`, `8a485699`, `9e8fb31b`,
`bfcd1175`, `fccc61cd`. Six s'y retrouvent ; `5dfdc63b` et `4f77afc1` n'y sont plus, et
`0797ce72` (hors du corpus de E0) s'y ajoute. Le parc a été **recuit depuis** (schéma 50 → 54) :
les poses et les bornes de vie ont bougé. Le compte 8/295 de E0 n'est donc pas reproductible tel
quel ; l'ORDRE DE GRANDEUR l'est (8 appareils de mur promus `deployed` sur 5 761 poses).

### 3.2 Verdict Q3 : le 103 ne tranche PAS ces cas, et il ne le pouvait pas

**Aucun des 15 cas n'est désigné par un 103.** Mais ce n'est pas un verdict « lâché » : c'est
un **silence**, et le silence est total pour cette famille d'objet — **0 sur 31 appareils de mur
classés `deployed`** et **0 sur 145 classés `dropped`**. Le 103 n'émet rien pour un appareil
porté, quel que soit son sort. Comparer les 15 cas à l'oracle Theater n'aurait rien ajouté : il
n'y a pas de signal à comparer.

### 3.3 La lecture INDIRECTE, celle qui intéresse F.1

Le film porte quand même le fait, mais par la PIÈCE et non par l'appareil : un mur réellement
déployé fait naître un panneau, et ce panneau est désigné par un 103 à 100 %. D'où la mesure :
une pose d'appareil porté est-elle accompagnée, dans les 5 s, d'une pose de PIÈCE de la même
famille (définie par la mesure — « un identifiant qu'un 103 désigne » — et non par une liste
écrite) ?

| Famille / origine de la pose d'APPAREIL | Une pièce de la même famille à ±5 s | Poses | Taux |
|---|---|---|---|
| `wall` / `deployed` | 5 | 34 | **14,7 %** |
| `wall` / `dropped` | 32 | 147 | **21,8 %** |
| `wall` / `unknown` | 1 | 7 | 14,3 % |
| toutes les autres familles, toutes origines | **0** | 2 067 | 0 % |

**La lecture indirecte ne tranche PAS non plus, et c'est mesuré.** Une pose d'appareil de mur
classée `deployed` n'a pas plus de panneau voisin qu'une pose classée `dropped` — elle en a
MOINS (14,7 % contre 21,8 %). Les deux chiffres sont le bruit de fond d'un match où des murs
s'ouvrent ailleurs sur la carte pendant que celui-ci tombe : la fenêtre de 5 s n'a aucune
contrainte spatiale, et lui en donner une reviendrait à réintroduire la clause de distance que
F.1 doit supprimer. Pour toutes les autres familles, la question ne se pose même pas : elles
n'ont **aucune** pièce (0 sur 2 067 poses).

**Conclusion de la question 3 : le film ne porte, par aucune des deux voies mesurées, de signal
permettant de séparer le déploiement du lâcher pour un équipement PORTÉ.** Le 103 est le fait
« une pièce a été engendrée » — rien de plus, et pour une seule famille.

---

## 4. Question 4 — la pièce engendrée du champ de réparation (D13)

### 4.1 Le recensement du parc : le corpus dit d'abord ce qu'il ne peut pas mesurer

`TestF0CorpusSpent` lit les **76 artefacts** exploitables du cache (un seul sans table de
palette) et compte les consommations de charge exploitables (`spent`, `gap = 0`, rang `from`
nommé) :

| Famille | Consommations exploitables du parc |
|---|---|
| mur | 125 |
| camouflage | 96 |
| surbouclier | 47 |
| **capteur** | **39** |
| propulseur | 27 |
| **écran occultant** | **22** |
| grappin | 18 |
| balise du translocateur | 14 |
| répulseur | 7 |
| **traqueur de menaces** | **6** |
| **champ de réparation** | **1** |

**Le premier résultat de la question 4 est là : le champ de réparation ne porte QU'UNE
consommation de charge exploitable dans tout le parc local** (film `5676a9ba`). Aucune mesure
d'enrichissement ne se conclut sur n = 1. Les dix films imposés par le plan n'en portaient
AUCUNE : c'est ce recensement qui a fait ajouter 15 films au corpus, et qui dit pourquoi la
réponse ne peut pas venir de ce parc.

### 4.2 La mesure d'enrichissement (15 films choisis, fenêtre ±2 s)

Fond du parc : **6 143 créations `ti=37` acceptées, 1 985 identifiants distincts**.

| Famille | Consommations | Créations `ti=37` dans les fenêtres | Témoin hasard | Identifiant le plus fréquent dans les fenêtres |
|---|---|---|---|---|
| **mur** *(témoin POSITIF)* | 65 | 184 | 128 | **`0x528fce46` (wall) n = 76, ×20,3** |
| capteur | 28 | 56 | 73 | `0xbcabbe43` (grenade frag) n = 10, ×0,82 |
| écran occultant | 8 | 35 | 15 | `0xbcabbe43` n = 10 ×1,31 ; puis `0x4396db42` (**l'écran lui-même**) n = 4, ×14,0 |
| traqueur | 6 | 17 | 19 | `0xbcabbe43` n = 6 ×1,62 ; puis `0x4744d742` (**le traqueur lui-même**) n = 2, ×17,2 |
| propulseur | 13 | 40 | 46 | `0xaada07f3` (grenade dynamo) n = 9, ×3,55 |
| grappin | 4 | 20 | 10 | grenades |
| balise du translocateur | 3 | 8 | 11 | grenades |
| camouflage | 12 | 77 | 78 | grenades |
| **champ de réparation** | **1** | 10 | 12 | que des singletons hors manifeste (n = 1) |

Lecture du classement : il est fait sur le COMPTE, pas sur l'enrichissement. Le balayage brut
accepte 1 985 identifiants distincts pour 6 143 records — un identifiant vu **une seule fois**
dans tout le film rend mécaniquement l'enrichissement maximal (1/part) dès qu'il tombe dans une
fenêtre. Classer dessus ferait remonter le bruit du balayage ; l'enrichissement reste la colonne
qui dit si un compte vaut mieux que le hasard.

### 4.3 Verdict Q4 : NON pour le champ de réparation, et le témoin positif l'ancre

1. **Le témoin POSITIF passe.** Autour des 65 consommations de charge de mur, `0x528fce46`
   apparaît 76 fois à un enrichissement de **×20,3** : la méthode retrouve la pièce engendrée
   qu'on sait exister. Elle est donc capable d'en trouver une.
2. **Aucune pièce distincte pour le capteur, le traqueur, l'écran.** Ce qui remonte autour de
   leurs consommations est, soit une grenade (l'objet le plus fréquent du film, à enrichissement
   ~1 — donc rien), soit **l'objet PORTÉ LUI-MÊME** (`0x4396db42` pour l'écran à ×14,0,
   `0x4744d742` pour le traqueur à ×17,2). Ce n'est pas une pièce engendrée : c'est le même
   GlobalID que celui qui est porté et lâché. **C'est précisément pourquoi `origin` ne peut pas
   séparer leur déploiement de leur lâcher — un seul identifiant sert les deux formes.**
3. **Le champ de réparation n'est pas mesurable sur ce parc** : n = 1.
4. **Le `tag group` demandé par l'item n'est PAS lisible ici** : le record de création ne porte
   qu'un mot de 32 bits (le GlobalID). Résoudre son groupe de tags exigerait de lire
   l'installation du jeu (`internal/himap`), hors du périmètre et hors des gates de ce lot.

### 4.4 « Le film voit 6 651 apparitions pour 295 poses publiées sur `000d5950` : dis ce que sont les non publiées »

La cascade, mesurée film par film :

| Film | Ancres `ti=37` | Records acceptés | dont identité au manifeste | Confirmés par l'oracle de vie | Poses publiées |
|---|---|---|---|---|---|
| `000d5950` | **6 651** | **401** | — | **295** | **295** |
| `1cd3848a` | 6 953 | 394 | 293 | 291 | 291 |
| `9e8fb31b` | 4 881 | 359 | — | 267 | 267 |
| `bfcd1175` | 6 227 | 387 | 275 | 265 | 265 |
| `4f77afc1` (BTB) | 100 117 | 3 185 | 922 | 658 | 658 |
| `5676a9ba` (BTB) | 52 515 | 1 605 | 678 | 490 | 490 |

**Les 6 356 « apparitions » non publiées de `000d5950` ne sont PAS des objets d'équipement que
l'artefact cacherait.** Ce sont, dans l'ordre :

1. **6 250 positions de bit qui passent un en-tête NON SÉLECTIF** et rien de plus. L'en-tête NEW
   `ti=37` fait 24 bits (3 de type + 13 de slot + 2 de génération + 6 de `typeIndex`) plus un
   test de bande ; il est balayé bit à bit. Le chiffre « ancres » est donc un compte de
   CANDIDATS, pas d'apparitions — c'est déjà ce que disait la mesure du 2026-08-17 (20 657
   ancres pour ~20 créations réelles sur un film BTB). Le facteur se lit au mieux sur les deux
   films BTB : 100 117 et 52 515 ancres, pour 3 185 et 1 605 corps qui se déroulent.
2. **~100 records dont le corps se déroule mais dont le mot de 32 bits ne se résout dans aucune
   entrée du manifeste** : 101 sur 394 (`1cd3848a`), 112 sur 387 (`bfcd1175`), et jusqu'à 2 263
   sur 3 185 sur le film BTB. Ils portent presque tous un identifiant **unique dans le film**
   (98 identifiants distincts pour 101 records) — la signature d'un mot lu au mauvais endroit,
   pas d'une famille d'objet qu'on aurait oubliée. C'est le même diagnostic que le témoin
   fantôme du lot des armes au sol (1 785 retenues contre 13 fantômes, facteur 137).
3. **~100 records d'identité valide que l'oracle de vie n'a pas confirmés** (394 → 291 sur
   `1cd3848a`) : la position du record ne retombe pas sur le premier point d'une vie décodée des
   paquets delta. Ce sont les objets qui **ne bougent pas** — les objets de socle, écartés par
   construction, comme la phase 3 du lot des power-ups l'a établi.

**Aucune des trois catégories ne cache une pièce engendrée nommable.** La question 4 se ferme
donc sur un négatif documenté, et non sur un « on n'a pas trouvé ».

---

## 5. Ce que cela commande pour F.1 et F.2

Ce paragraphe est une LECTURE du mesuré, pas une décision : les items F.1 et F.2 attendent le
feu vert du pilote.

- **F.1 (D12) — la branche « si F.0 dit oui » du plan est FERMÉE.** Le 103 ne peut pas porter
  l'origine d'une pose : il n'existe que pour les panneaux de mur, et les poses dont l'origine
  est en cause sont celles des appareils PORTÉS, pour lesquels il est muet à 0 sur 176.
  Supprimer `originDropWindowUS` et `originDropMaxDist` au profit d'une lecture du 103 rendrait
  l'origine indéterminée pour 5 545 poses sur 5 761. **La branche applicable est donc la
  seconde : retirer la seule clause de DISTANCE et garder le fait temporel**, avec la mesure
  avant/après par famille — sous réserve du §3.3.
- **F.2 (D13) — `[!]`, avec la mesure.** Aucune pièce engendrée n'est identifiable pour le champ
  de réparation (n = 1 consommation dans tout le parc), ni pour le capteur, le traqueur ou
  l'écran (leurs fenêtres ne montrent que l'objet porté lui-même). Rien à ajouter au manifeste
  ni à `usageFamiliesWithSpawnedPiece` : y ajouter une famille sans pièce ferait lire son
  « utilisé » sur `DeployedByFamily`, c'est-à-dire exactement le défaut que `us6` a corrigé le
  2026-09-10.
- **Le fichier de référence de l'équipement est à corriger sur un point**, et c'est un GAIN :
  `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` n'a pas de ligne fausse ici, mais le rapport
  R5 §3.2 en a une (« les 90 têtes vers un dropped montrent que les objets lâchés à la mort SONT
  aussi des apparitions d'objets d'équipement », lue depuis comme « le 103 tire aussi à la
  mort »). Elle est **réfutée** par le §2.2.

---

## 6. Réserves, dites

1. **Le corpus n'est pas le parc.** 25 films sur les 1 351 du cache de chunks, 76 artefacts
   exploitables sur 140 fichiers. Les taux sont ceux de ce corpus.
2. **La carte de 3 films ne se distingue pas dans sa classe d'étendue** (`1cd3848a`, `8a485699`,
   `fccc61cd` : 3 candidats à égalité de profondeur d'oracle). L'invariance d'échelle (§0.3) rend
   ce choix sans effet sur ce que ce rapport juge ; elle ne le rendrait pas sans effet sur des
   COORDONNÉES, et ce rapport n'en publie aucune.
3. **La clé `(slot, génération)` n'est pas unique dans un film** : le pool de slots reboucle et
   la génération ne fait que 2 bits. À clé égale, la création la plus ANCIENNE est retenue, et
   l'écart de temps est publié — c'est lui qui montre les collisions (la queue à +19 s du dt de
   `ref1`). Les 4 poses `dropped` désignées du §2.2 sont, pour 3 d'entre elles, des panneaux ;
   la quatrième est très probablement une collision de ce type.
4. **`ref0` n'est pas identifiée.** On sait ce qu'elle n'est pas (l'appareil porté) et ce que
   les images-clés en disent (un `ti=37` de longue durée). Ce lot ne l'établit pas, et ne
   prétend pas le contraire.
5. **Les largeurs de référence du type 103 restent celles de la table R7**, dérivées des thunks
   `vtable+0x58` de l'exécutable. Elles sont ici corroborées d'une manière nouvelle — un
   décalage d'un bit ne rendrait pas 93,6 % de clés d'entité valides — mais elles ne sont pas
   re-sourcées.
6. **La question 4 mesure un ENRICHISSEMENT sur un balayage brut bruité** (1 985 identifiants
   pour 6 143 records). C'est le témoin positif du mur qui la rend concluante ; sans lui elle ne
   vaudrait rien, et c'est pour cela qu'il est là.

---

## 7. Découvertes hors périmètre — consignées, NON traitées

- **`0x412000aa` est désigné une fois par un `ref1` de 103** sur `9e8fb31b`, avec un dt de
  +19 386 ms — c'est-à-dire très probablement une collision de clé, mais l'identifiant est hors
  manifeste et n'a pas été instruit.
- **Trois poses de panneau de mur du parc sont classées `dropped` et quatre `unknown`** alors
  qu'un panneau, par construction, ne peut pas être lâché à la mort (il n'existe qu'une fois
  déployé). Ce sont **7 défauts d'origine sur 216 panneaux**, du même ordre que les 8 appareils
  promus `deployed` de D12, mais dans l'autre sens. À traiter avec D12, pas à part.
- **`ref0` du 103 désigne un `ti=37` que les images-clés voient et que les paquets delta ne
  créent jamais** (737 sur 739). Si cette entité est le « spawner » d'équipement de la carte,
  elle ouvrirait une voie d'identification de l'équipement SOURCE d'un déploiement — sujet du
  lot I (refonte du décodeur), pas de celui-ci.
- **Le balayage brut `ti=37` accepte 2 263 records hors manifeste sur 3 185 sur `4f77afc1`**
  (71 %) contre 25 % sur un film d'arène. La sélectivité de l'en-tête NEW s'effondre sur les
  films BTB, et aucune mesure du dépôt ne la borne aujourd'hui pour l'équipement.
