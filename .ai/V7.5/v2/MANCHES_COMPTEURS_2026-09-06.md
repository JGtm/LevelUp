# La decoupe par manche des compteurs par joueur — 2026-09-06

Branche `feat/v2-manches`, worktree `LevelUp-wt-v2-manches`, base `a059caefc` (= `feat/v75`,
schema 43). Instruction : le fait n° 1 des « decouvertes non traitees » de
`.ai/V7.5/v2/INSTRUCTION_RESIDUS_2026-09-06.md` — `51ebbc0f` publie **63 assistances** pour un
joueur qui en a **5** a la feuille de match.

---

## Verdict en une page

| | |
|---|---|
| Diagnostic du relecteur | **CONFIRME sur pieces**, a la milliseconde et a la valeur pres |
| Cause racine | la decoupe par manche croyait sur parole le numero de manche lu dans le film ; ce numero est faux sur un residu d'enregistrements mal alignes, et la sous-suite non decroissante ne peut pas les ecarter |
| Correctif | la manche DECLAREE est confrontee au TEMPS : les manches se jouent dans l'ordre, un enregistrement date hors de l'intervalle de la manche qu'il declare n'alimente aucune serie |
| Filet aval | une serie cumulee qui RECULE dans le temps n'est plus publiee, et l'ecart est journalise |
| Temoins | 15 films re-cuits : **11 identiques hors numero de schema**, 4 corriges, **zero perte reelle, zero disparition** |
| Schema | **43 -> 44**, chronique ecrite dans `document.go` et dans le ratchet `structure_test.go` |
| Regression trouvee en cours de route | une premiere version du correctif cassait `a4083bd2` (24 compteurs de joueur en baisse) ; corrigee, prouvee par mutation, mesuree |

---

## 1. Reproduction, et le diagnostic chiffre

Cuisson de `51ebbc0f` au HEAD de `feat/v75` (`cmd/replay-build --map "Banished Narrows"
--facts <faits> 51ebbc0f-...`, un film par processus, verrou `filmproc.AcquireSolo`, racine de
travail dediee — technique du balayage, §2 de `BALAYAGE_PARC_2026-09-06.md`).

Serie d'assistances publiee pour `2535439712156981` (feuille : **5**) :

```
rounds[0] = [{t:1412, v:1}, {t:3167, v:60}]
rounds[1] = [{t:2592, v:0}, {t:3057, v:1}, {t:4225, v:2}, {t:4452, v:3}]
total     = [{t:1412,1}, {t:3167,60}, {t:3057,61}, {t:4225,62}, {t:4452,63}]
```

Les trois affirmations du relecteur, verifiees une a une :

| affirmation | mesure |
|---|---|
| la manche 0 porte un echantillon date **t = 3167** (frame), soit **316 777 ms** | VRAI |
| la manche 1 demarre a **t = 2592** (259 240 ms) | VRAI — les huit slots de joueur l'ouvrent tous a 259 240 ms, les deux slots d'equipe a 262 339 ms |
| les sept autres joueurs s'arretent avant **t = 2285** | VRAI (derniers points de manche 0 : 742, 1306, 1421, 1563, 1977-2081, 2118, 2248, 2285) |
| le `total` concatene sans controle de monotonie : `{3167,60}` puis `{3057,61}` | VRAI — les instants **reculent** |
| le 60 devient la base de la manche 1 | VRAI — `cumulateXUIDRounds` prend le dernier point de la manche comme decalage |

### D'ou vient l'echantillon egare : ni par instant, ni par slot, ni par identite

Il vient du **numero de manche lu dans le film**. La manche d'un enregistrement est lue dans
DEUX en-tetes de 5 bits du premier composant (`objectiveevents/statborg.go`,
`decodeComponents`) ; l'assertion d'en-tete ayant ete relachee le 2026-08-18 (sans quoi tout ce
qui suit la premiere manche etait jete), un residu de faux positifs franchit la porte et porte
une manche quelconque.

L'enregistrement fautif, lu tel quel :

```
51ebbc0f  manche 0  slot 12  t=316777  ncomp=8
  c2(A=0,B=0)  c3(A=60,B=7)  c5(A=4164778782,B=3315695892)  c20(A=4152,B=20582)
  c25(A=0,B=3) c26(A=-59,B=306)  c28(A=2915764224,B=88)  c31(A=0,B=1)
```

`c5 = 4 164 778 782` le denonce : cet enregistrement n'est aligne sur rien. Mais son `c3 A = 60`
(assistances) est PLAUSIBLE en valeur, et la plus longue sous-suite non decroissante
(`longestRun`, non stricte pour les compteurs de recompense) le GARDE — une valeur plus GRANDE
prolonge la suite au lieu de la rompre. Aucun filtre existant ne pouvait l'atteindre :

- le rejet des valeurs negatives : 60 est positif ;
- la garde de domaine `modeScoreInDomain` : elle ne s'applique qu'au composant 0 ;
- les bornes de deroulage (`maxUnrollPerStep = 100 000`) : 60 est loin dessous.

Son `c2 (A=0, B=0)` coutait en plus **le seul frag de manche 0** du joueur : sur la suite
`(0,0,0,0,1,0)` la plus longue sous-suite non decroissante retenue devenait `(0,0,0,0,0)` — le
`1` reel etait remplace par le `0` du faux positif. C'est pourquoi le document publiait `kills = 0`
pour un joueur qui en a 2.

**Ce n'est pas un cas isole.** Le MEME motif (`slot 12`, `c3 A = 60, B = 7`, `c5` aberrant)
existe sur `d9781168` a `t = 345 931 ms`, ou il portait les assistances d'un joueur a **69** pour
**11** a la feuille.

---

## 2. Inventaire : quels compteurs, quels consommateurs

La decoupe par manche est faite par DEUX marches de `objectiveevents/named_series.go`, et les
deux prennent le numero de manche pour argent comptant :

| marche | emplacements groupes | consommateurs |
|---|---|---|
| `rawSeriesByRound` | un emplacement a la fois | `SeriesByRound` (courbes par manche) et `SeriesTotal` (cumul), `seriesBySlot` |
| `rawSeriesByKey` | tous les emplacements d'une table, en une passe | `NamedEventsFrom` — les **actions d'objectif** |

Les emplacements publies, et ce qu'ils alimentent :

| emplacement | calque | ou |
|---|---|---|
| `comp 1 B` score personnel | `scoreTimeline.players[].score` | `analysis/replay/score_timeline.go` |
| `comp 2 A` frags · `comp 2 B` morts | `.kills` / `.deaths` | idem |
| `comp 3 A` assistances | `.assists` | idem |
| `comp 0 A` score de mode (slots d'equipe) | `scoreTimeline.teams[]` + `coverage.score` | idem |
| `comp 0 A` (slots joueur) = tics de possession | `skullCarries` (Oddball) | `skull_carries.go` |
| `comp 21 B` prises du crane | `skullCarries` | idem |
| `comp 23 A` tics de colline | `scoreTimeline.holdTicks` | `hill_hold_ticks.go` |
| table `namedStatSlots` (flag, zone, vip, bomb, koth) | `objectives[]`, `coverage.flagCarries` | `build_objectives_live.go` |

**Consommateurs du `total`.** Deux cumuls concatenent les manches DANS L'ORDRE DES MANCHES en
supposant que c'est l'ordre du TEMPS :

- `objectiveevents.cumulateRounds` — sert `SeriesTotal` (courbes d'equipe, tics de colline) et
  `seriesBySlot` (actions nommees) ;
- `analysis/replay.cumulateXUIDRounds` — sert le total par JOUEUR (`seriesOfRounds`).

Aucun des deux ne verifiait cette supposition. C'est ce que le relecteur a vu.

---

## 3. Le correctif, a la source

### 3.1 Les bornes de manche (`objectiveevents/round_windows.go`, nouveau)

**La regle** : les manches se jouent dans l'ordre ; une manche occupe un intervalle de temps ; un
enregistrement date hors de l'intervalle de la manche qu'il declare n'en est pas.

Trois mesures, **aucune constante ajustee** — que des medianes et une majorite :

| | |
|---|---|
| DEBUT d'une manche | la mediane, sur les slots, du premier instant ou le slot declare cette manche |
| MANCHE UTILISABLE | une manche dont moins de la moitie des slots parlent ne fixe aucune borne |
| BORNE CREDIBLE | mediane des instants de la manche precedente STRICTEMENT avant la borne, mediane de la suivante a la borne ou apres |

Plus une garde d'ensemble : **les debuts doivent croitre** avec le numero de manche ; sinon
aucune borne n'est posee du tout.

**Pourquoi la mediane et pas le minimum.** Releve du 2026-09-06, colonne « debuts par slot » :

```
24dbb67d manche 1 : debuts = [85193, 116358, 298909 x7, 301983]   mediane = 298909
51ebbc0f manche 1 : debuts = [259240 x8, 262339 x2]               mediane = 259240
43716616 manche 1 : debuts = [168746 x8, 171837 x2]               mediane = 168746
d9781168 manche 1 : debuts = [231896 x8, 234987 x2]               mediane = 231896
```

Le minimum aurait pris **85 193** sur `24dbb67d` et jete **3 612** enregistrements legitimes de la
manche 0 ; la mediane en jette **16**. Le decalage regulier de ~3 s entre les huit slots de
joueur et les deux slots d'equipe est le nominal, et la mediane l'absorbe.

**Mediane BASSE**, et c'est un choix de surete : sur une longueur paire, la mediane haute tombe
APRES le premier slot qui a ouvert la manche, et ce slot perd son ouverture (montre a nu par le
corpus a deux slots de `named_onepass_test.go`). Sur les films reels (8 a 10 slots) les deux
medianes donnent la meme valeur.

### 3.2 Le filet aval (`objectiveevents.ChronologicalTotal`)

Une serie cumulee qui recule dans le temps n'est plus publiee : le point tardif-dans-la-liste
mais precoce-dans-le-temps est ecarte, et l'ecart est journalise
(`slog.Warn "serie cumulee NON CHRONOLOGIQUE"`, avec le slot, le compte et le premier recul).
Le controle vit dans UNE fonction et sert les DEUX cumuls — pas deux copies de la meme decision.

Ce n'est pas ce qui repare `51ebbc0f` : la cause est traitee en amont. C'est ce qui interdit a une
courbe non chronologique de sortir en silence si une autre cause apparait — et il MORD encore
aujourd'hui, sur les trois films dont l'etiquetage de manche ne suit pas l'horloge et ou aucune
borne n'est posable.

### 3.3 Observabilite

`attachScoreTimeline` publie, une fois par match a plusieurs manches :

- `rejeu : enregistrements hors de la fenetre de leur manche declaree, ecartes` — le NOMINAL
  (5 a 27 sur les temoins) ;
- ou, si le compte est nul, `rejeu : AUCUNE borne de manche posee sur un film a plusieurs
  manches` en WARN — le silence est un signal, il dit que le numero de manche du film ne suit
  pas l'horloge.

Rien n'est publie sur un film mono-manche : il n'a pas de borne.

---

## 4. La regression que la premiere version du correctif introduisait

**Elle a ete trouvee par la mesure, pas par relecture.** La premiere version ne portait que la
mediane et la credibilite. Sur `a4083bd2` (Team Slayer — un mode SANS manche, dont `RealRounds`
retient pourtant trois manches) :

```
manche 0 : 10 slots, debut 4203,   milieu 334763
manche 1 :  1 slot,  debut 485722, milieu 485722   <- UN enregistrement
manche 2 : 10 slots, debut 187378, milieu 298155
```

La « manche 1 » d'UN SEUL enregistrement fixait une borne a 485 722 ms, **153 enregistrements sur
719 etaient jetes**, 24 compteurs de joueur baissaient (frags 16 -> 11, morts 15 -> 11, score
1 950 -> 1 450), le score d'equipe passait de 50/45 a 36/35 et l'identite des camps de `a` a
`unresolved`.

Deux gardes ont ete ajoutees, chacune prouvee par mutation :

- **majorite de slots** — un debut de manche est un consensus, un slot n'en est pas un ;
- **croissance des debuts** — si le numero de manche ne suit pas l'horloge, rien n'est coupe.

Apres correction, `a4083bd2` est **identique a l'octet**.

---

## 5. Temoins : 15 films re-cuits, base `feat/v75` schema 43 contre schema 44

Douze films multi-manche (LA TOTALITE de ceux du parc de 119) + trois mono-manche de controle.
Comparateur `cmd/replay-diff`.

| film | mode / manches | ecart |
|---|---|---|
| **`51ebbc0f`** | Oddball, 2 | **assists 63 -> 4**, **kills 0 -> 1**, `flagCarries.steals` 58 -> 0 |
| **`d9781168`** | Oddball, 3 | **assists 69 -> 11** (= la feuille), `score.points` 2137 -> 2135 |
| **`24dbb67d`** | Ranked:Oddball, 2 | `flagCarries.steals` **994 -> 0** |
| **`c75f33b8`** | Assault:One Bomb, 3 | `score.points` 400 -> 396 |
| `43716616` | Oddball, 2 | — |
| `9f57c612` | Assault:One Bomb, 4 | — |
| `cde26226` `7fce3219` `64e8adfa` | CTF, 2 | — |
| `fb1a1a72` `72b0a25e` `a4083bd2` | etiquetage faux, 3 | — (aucune borne posee) |
| `000d5950` `c0a82e88` `53ce4390` | mono-manche | — |

« — » = **identique a l'octet hors la ligne `schemaVersion 43 -> 44`**. Onze sur quinze.

**Zero disparition, zero perte reelle.** Les quatre « pertes » que le comparateur signale sont
toutes des compteurs FANTOMES qui tombent a leur vraie valeur :

- `flagCarries.steals` 58 et 994 sur **deux films d'Oddball**, donc sans drapeau
  (`coverage.flagCarries.flagFilm = false`, `carries = 0`) : le `comp 24 A` d'un enregistrement
  egare (`A = 58`, `A = 994`) etait deroule en autant de vols de drapeau nommes ;
- `score.points` -2 et -4 : les points de courbe que les echantillons egares ajoutaient.

### L'oracle independant : la feuille de match

Le film reconstruit desormais, par slot, des totaux qui rejoignent la feuille :

| slot | frags / morts / assists (film) | joueur de la feuille | feuille |
|---|---|---|---|
| 10 | 11 / 6 / 4 | `2535458847428879` | 11 / 6 / 4 **exact** |
| 12 | **2 / 10 / 5** | `2535439712156981` | **2 / 10 / 5 exact** |
| 14 | 11 / 12 / 7 | `2535469190789936` | 11 / 12 / 7 **exact** |
| 20 | 15 / 4 / 2 | `2535469889270266` | 15 / 4 / 2 **exact** |
| 16 · 18 · 22 · 24 | 10/11/6 · 7/9/3 · 7/8/5 · 7/11/7 | — | a +/- 1 pres |

Le joueur en cause a donc bien **5 assistances et 2 frags** dans ce que le film reconstruit,
exactement la feuille. Le DOCUMENT en publie 4 et 1 — pour une raison etrangere a ce lot, mesuree
et consignee en decouverte n° 1 ci-dessous.

---

## 6. Tests par mutation

`analysis/replay/manches_compteurs_test.go` — enregistrements synthetiques, aucune lecture de
film, joues par la CI. Dix mutations passees sur le correctif, chacune tue au moins un test :

| mutation | tests tues |
|---|---|
| M1 `rawSeriesByRound` sans le filtre de fenetre | 3 |
| M1b `rawSeriesByKey` sans le filtre de fenetre | 1 |
| M2a `cumulateRounds` sans le controle de chronologie | 1 |
| M2b `ChronologicalTotal` neutralisee | 2 |
| M4 sans la majorite de slots | 1 |
| M5 sans la croissance des debuts | 1 |
| M6 mediane haute au lieu de basse | 1 |
| M7 sans le test de credibilite | 1 |
| M8 sans le bord gauche de la fenetre | 1 |
| M9 sans le bord droit de la fenetre | 3 |

Le film mono-manche fait exception et c'est ecrit dans le test : sa neutralite tient a ce qu'une
SEULE manche n'offre aucune PAIRE de manches, donc aucune borne — aucune retouche d'une seule
ligne ne la casse. Sa preuve est la mesure : `000d5950`, `c0a82e88` et `53ce4390` re-cuits sont
identiques a l'octet.

Instrument du releve conserve : `analysis/replay/manches_bornes_research_test.go` (gardes
`MANCHES_CACHE` + `MANCHES_FILMS`), qui imprime la table des debuts par slot dont sortent les
trois gardes.

---

## 7. Gates joues

| gate | resultat |
|---|---|
| `go build ./...` | OK |
| `go test -count=1 ./internal/analysis/replay/... ./internal/replaybuild/... ./internal/archlint/... ./contracttest/...` | ok (5 paquets) |
| `go test -tags=integration -p 1 -count=1 ./internal/api/wire/...` | ok (29,9 s) |
| `golangci-lint run --new-from-merge-base=origin/main ./...` | **0 issues** |
| golden d'assemblage regenere | UNE ligne changee (`schema 43` -> `schema 44`), sur 605 |

---

## 8. Decouvertes, notees et NON traitees (regle 7)

1. **`51ebbc0f` publie 4 assistances sur les 5 que le film reconstruit — et la cause n'est pas la
   decoupe par manche.** Sa grille de frames s'arrete a **451 400 ms** (`frameCount = 4514`,
   `frameIntervalMs = 100`) alors que les enregistrements de statistique vont jusqu'a
   **498 941 ms** : **47,5 s de match tombent hors fenetre**, et avec elles la derniere emission
   de plusieurs compteurs. La cause probable est nommee par le document lui-meme :
   `coverage.originResolved = false` — l'origine du fil n'est pas resolue, donc `originMS = 0`
   alors que les joueurs rejoignent le match a `joinMatchMs = 51 364`. **Ceci explique la
   question restee ouverte au registre** (« aucune des 8 courbes de `51ebbc0f` ne colle a la
   feuille, contre 4 sur 8 pour `43716616`, l'ecart n'est pas explique ») : sur les cinq films
   mesures, `51ebbc0f` et `fb1a1a72` ont `originResolved = false`, `43716616`, `d9781168` et
   `000d5950` l'ont a `true`. A instruire pour lui-meme.
2. **`RealRounds` retient des manches sur des modes qui n'en ont pas.** `a4083bd2` et `72b0a25e`
   sont du Team Slayer / Slayer et rendent 3 manches ; `fb1a1a72` (CTF) en rend 3 dont une sans
   aucun enregistrement et une repandue de 66 s a 814 s. Le present lot s'en protege (il ne pose
   alors aucune borne) mais ne corrige pas la cause. Trois films sur 119.
3. **`coverage.flagCarries.captures` reste a 13 sur `51ebbc0f`**, un Oddball sans drapeau. Le
   `comp 21 A` y sert a autre chose (le `comp 21 B` est le compteur de prises de crane). Bruit
   pre-existant, hors perimetre.
4. **Le journal du filet de chronologie est repetitif sur les trois films a etiquetage faux**
   (25 a 63 lignes WARN par cuisson) : la meme serie est re-derivee par chaque appel de
   `SeriesTotal` / `seriesBySlot`. Le compte de faits distincts, lui, est de 5 a 8. Reduire la
   repetition demanderait un point d'agregation par match que le paquet n'a pas ; consigne ici
   plutot que traite.

---

## 9. Reproduire

```bash
# 1. les faits (une fois, serveur arrete)
levelup replay-facts-export --out <travail>/facts 51ebbc0f d9781168 ...

# 2. une cuisson = un processus borne, un film a la fois
LEVELUP_REPO_ROOT=<racine de travail> replay-build \
  --map "<carte>" --facts <travail>/facts/<short8>.facts.json <matchId complet>

# 3. la comparaison, axe par axe
replay-diff -ancien <base43>.json -nouveau <schema44>.json

# 4. le releve des bornes de manche, film par film
MANCHES_CACHE=<depot>/data/cache MANCHES_FILMS=51ebbc0f,d9781168,a4083bd2 \
  go test ./internal/analysis/replay/ -run ManchesBornes -v -timeout 60m
```

Racine de travail : `data/cache/film_chunks`, `data/cache/film_manifests`, `data/cache/mvar`,
`data/titles` et `config` en JONCTIONS de lecture ; `data/cache/replays` un VRAI dossier. Temoins
avant/apres sur le checkout principal : `data/cache/replays` (128 fichiers), `data/titles/.../reference`
(230 fichiers) et `data/cache/film_chunks` (1 380 entrees) INCHANGES ; `git status` du principal
vierge sous `data/`.
