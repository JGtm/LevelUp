# HANDOFF — score par mode, evenements d'objectif : le deroule d'un match est lisible

> Reecrit le 2026-08-01 en fin de 3e passe. Branche `feat/re-mode-score` (worktree dedie).
> Etat de l'art complet : `.ai/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md`, **section 15** pour
> cette passe.
>
> **L'OBJECTIF** : rejouer le deroule d'un match — la progression du score et qui prend une
> base / attrape un drapeau, a l'instant pres. **Il est atteint.** Ce document dit ce qui est
> livrable, ce qui reste, et par ou reprendre.

---

## 1. CE QUI EST LIVRABLE AUJOURD'HUI, SANS AUCUN DUMP

| ce qu'on peut dessiner | statut | precision |
|---|---|---|
| **La courbe de score d'equipe, mode a objectifs** | **ETABLI** | a la ms, reemise a chaque changement |
| **Strongholds : qui prend/securise une base, avec le gamertag** | ETABLI | a la ms |
| **CTF : l'instant de chaque capture** | ETABLI | a la ms, 0 manque / 0 faux positif sur 4 matchs |
| **CTF : le compteur de captures PAR JOUEUR** | ETABLI | a la ms (le composant de score des 8 entites de joueur) |
| Score personnel et frags/morts par joueur, dans le temps | mesure, PAS livre | composants 1 et 2, meme ancrage — voir §6 |
| **CTF : qui a pris / rendu / vole (hors capture)** | **ETABLI** *(2026-08-02)* | par la lecture par COMPOSANT : `comp 22 A` = `flag_grabs`, `23 A` = `flag_returns`, `24 A` = `flag_steals`, dates a la ms et attribues par xuid — **48/48 exacts** sur `1bc77d2e` contre l'oracle a 8 joueurs (etat de l'art §22.1) |
| **Quelle** zone (A/B/C) | **NON** | resultat negatif inchange |

> **Mise a jour 2026-08-02 — la ligne « qui a pris / rendu » a change de verdict.** Le
> **NON** portait sur les evenements de MODE du footer (etat de l'art 4.2 : le compte du film
> et celui de l'API divergent, et l'ecart change de signe d'un film a l'autre). Cette
> refutation tient toujours **pour ce canal-la**. Les emplacements de statistique du statborg
> sont un canal different, et lui ferme la question : detail dans
> `HANDOFF_EVENEMENTS_NOMMES_2026-08-01.md` §2 et etat de l'art §§17-18, 22.

**Outil** : `cmd/tmp_timeline <cacheDir> <filmID> <gameVariantName>` — ligne de temps fusionnee
« evenement d'objectif + progression du score ». `COMPACT=1` masque les lignes par joueur.

---

## 2. POURQUOI C'EST CREDIBLE — les chiffres, pas les impressions

**Confrontation position de bit par position de bit** contre la capture Cheat Engine
(`cmd/tmp_scoreverify`) :

| film | verite terrain | ancrages | retrouves (position ET valeur) | faux positifs |
|---|---|---|---|---|
| `696a9d7c` Strongholds | 284 | 285 | **283 (99,6 %)** | **2 (0,7 %)** |
| `530820e5` CTF | 6 | 6 | **6 (100 %)** | **0** |

Zero valeur fausse a bonne position. Les 2 residus tombent sur la seule monotonie du score.

**Deux decodeurs independants, la meme milliseconde.** En CTF le score n'avance que sur une
capture ; le detecteur de captures (bursts a 6 tiers du footer) et le decodeur de score
(composant 0 des paquets delta) n'ont rien en commun :

```
140.072 OBJ capture   ->  140.073 SCORE slot 8 = 1
186.218 OBJ capture   ->  186.219 SCORE slot 8 = 2
473.211 OBJ capture   ->  473.212 SCORE slot 8 = 3
```

**Le releve terrain Strongholds, ecrit a l'oeil AVANT tout decodage : 4 ancres sur 4.**
capture a 48,901 s · 21 pour l'equipe plafonnee a t=90 s · **69-30 exactement a t=190 s** ·
trois bases tenues a 5:34. Score final 200-94, conforme a l'API.

---

## 3. LA GRAMMAIRE, TELLE QU'ELLE EST MESUREE

```
[1 bit = 1][14 bits slot][2 bits = 10][1 bit forme = 0][3 bits N][N x 6 bits index croissants]
puis, par index : [5 bits = 0][5 bits = 0][valeur A][valeur B][2 drapeaux][conditionnelles]
```

- slots : **6 et 8** = les deux equipes ; **10, 12, 14, 16, 18, 20, 22, 24** = les 8 joueurs ;
- composant **0** = score de mode, **1** = score personnel, **2** = frags/morts ;
- le composant n'est reemis **que lorsqu'il change** — bien plus fin que le tick de 5 s du
  footer ;
- selecteur de largeur de la valeur : 0 ou 1, **jamais 2** sur les lectures reelles.

**Ce que la mesure a corrige par rapport au handoff precedent** : les 2 bits qui precedent le
bit de presence ne sont PAS un champ de type — ils sont statistiquement independants (22
co-occurrences pour 21,6 attendues sous independance). C'est la queue de l'enregistrement
precedent. Ne pas les contraindre.

---

## 4. L'HORLOGE — une seule ligne de temps, et c'est mesure

`cmd/tmp_filmclock` : le `TimestampUS` du premier paquet de chaque chunk reproduit le
`start_ms` du manifeste a **-4 a 0 ms pres sur 573 s** (30 chunks). Les evenements du footer
sont sur la meme base. Tout est superposable sans recalage.

**Le piege, paye** : prendre pour origine le premier paquet *ou l'on trouve quelque chose*
decale toute la courbe — 36 s sur le Strongholds, 140 s sur le CTF. **L'origine se prend par
chunk, sur le manifeste.**

---

## 5. LES OUTILS (tous en `CGO_ENABLED=0`, non suivis par git)

| outil | role |
|---|---|
| **`cmd/tmp_timeline`** | **le livrable** — ligne de temps fusionnee objectifs + score |
| `cmd/tmp_scoreverify` | la mise a l'epreuve : ancrage hors ligne confronte a la capture CE |
| `cmd/tmp_filmclock` | l'alignement des horloges manifeste / paquet |
| `cmd/tmp_chainhdr` | grammaire d'en-tete ; `FRAMEBITS=1` = le cadrage mesure, `MAP=1` = eid -> slot |
| `cmd/tmp_statborgfilm` | le pont capture CE -> film (verite terrain) |
| `cmd/tmp_modeticks` | evenements de mode du footer ; `VERBOSE=1` = timeline horodatee |
| autres `tmp_*` | mesures des passes precedentes (cartographies, refutations) |

Captures CE : `E:/LevelUp_rejeu2D/captures_2026-07-31/`. Films : `data/cache/film_chunks/`,
manifestes : `data/cache/film_manifests/`.

---

## 6. LE POINT DE REPRISE — ce n'est plus de la retro-ingenierie

Le volet retro-ingenierie est **clos**. Ce qui reste est de l'industrialisation, et le plan
existe : `.ai/PLAN_OBJECTIFS_TEMPS_REEL.md`, etape 1.

1. **[FAIT dans cette passe]** Le decodeur est promu dans le depot :
   `internal/analysis/objectiveevents/score.go` — `ScoreCurve(FilmSource) []ScorePoint`,
   avec deux tests de verite terrain (`score_test.go`, skip propre si le cache est absent).
   Il y vit plutot que dans `objectivescore` parce qu'il reutilise `readBitsBE`,
   `decompressChunk` et `walkFrames` : une troisieme copie du lecteur de bits aurait viole
   la regle des 2 copies.
2. **[STATUE — SUPPRIME le 2026-08-08, v7.5 lot 3]** Le sort de
   `internal/analysis/objectivescore` est tranche : **suppression**, decidee par
   l'utilisateur sur le verdict des octets reels du lot 2. Le paquet, son repo
   (`platform/duckdb/objective_score_repo.go`), le mode `-score` de `cmd/diag_weapons_v3`
   et la table `match_objective_score_timeline` (step de DROP au nom neuf
   `shared_objective_score_v1_drop`) sont partis dans le meme commit. Ce qui SURVIT : le
   corpus d'octets reels, re-domicilie sous
   `internal/games/halo_infinite/film/testdata/corpus_objectifs/` avec son README — il ne
   decrit pas un decodeur, il decrit un format, et il contraindra le futur peuplement
   evenementiel des modes a zones / KOTH.

   Ce que la proposition d'origine disait, et qui reste vrai : la condition de conservation
   (« en attente d'une RE plus poussee ») avait expire, cette RE ayant eu lieu et son
   resultat etant meilleur sur tous les axes (per-joueur, a la ms, sans calibration).
   Le lot 2 y a ajoute la preuve directe : sur films reels, la brute Strongholds plafonne
   quel que soit le final, la courbe n'etait pas un score per-equipe.
3. **Le faire produire par la synchronisation**, pas par un CLI de diagnostic : aujourd'hui les
   8 140 evenements d'objectif en base existent parce que quelqu'un a lance un outil a la main.

   **Re-scopage du 2026-08-08 (le `[!]` laisse ouvert par le lot 1).** Le lot 1 avait laisse
   « le peuplement live des events objectif » comme seul moyen de rendre la courbe CTF
   disponible ET « de brancher un jour le decodeur zones/KOTH ». Cette seconde branche
   n'existe plus : le decodeur en question a ete supprime. Le peuplement live signifie
   desormais **la source evenementielle et elle seule** (`analysis/objectiveevents` ->
   `match_objective_events`), et **le pont vers les modes a zones / KOTH passe par elle** —
   il n'y a plus d'autre chemin, ni de score per-equipe a aller relire ailleurs. La table
   `match_objective_events` reste donc pleinement vivante : elle a deux lecteurs applicatifs
   (la dominance CTF de `sync/comeback.go` et `MatchViewService`), et n'a rien a voir avec la
   table supprimee au point 2.
4. **Persister la courbe de score** (append-only, regle ART n°2 : ecriture INSERT, lecture par
   vue `_latest`).
5. **Exposer et consommer** : le front n'affiche que 7 % des evenements deja en base.
6. **Etendre aux composants 1 et 2 si on veut le detail par joueur dans le temps** (score
   personnel, frags/morts). Le meme ancrage les atteint, et le rendement est **mesure** :
   composant 1 = 374/381 retrouves pour 3 faux positifs, composant 2 = 385/397 pour 12.
   Non livre volontairement : atteindre l'index 1 ou 2 oblige a decoder les composants qui
   precedent dans la liste creuse, donc a chainer des largeurs — c'est un chemin moins sur
   que l'index 0, qui est toujours le premier de sa liste. A ne brancher qu'avec ses
   propres tests de verite terrain.

**Nota, mesure** : en Strongholds le composant de score n'est emis QUE par les 2 entites
d'equipe (284 lectures, toutes sur les slots 6 et 8) ; en CTF il l'est aussi par les
joueurs, ou il vaut leur compte de captures. Le detail par joueur depend donc du mode.

**Generalite mesuree — le balayage des 951 films du cache est FAIT** : **944 films decodes
proprement (99,3 %)**, 5 a valeur aberrante, 2 sans aucune emission. Et la distribution des
scores maximaux reproduit les plafonds de Halo sans qu'on les ait dits au decodeur :
**50 (576 films, Slayer) · 3 et 2 (194, CTF) · 200 (48, Strongholds) · 100 (44) · 5 (18,
KOTH classe)**. Deux contraintes sont nees de ce balayage : selecteur de largeur != 2
(24 aberrants -> 10) puis plus longue suite croissante au lieu d'un filtre glouton
(10 -> 5). Detail : etat de l'art §15.8.

---

## 7. LES REGLES QUE CE CHANTIER A PAYEES

1. **Une valeur finale juste ne prouve rien.** Un candidat Strongholds parfait sur le papier
   etait faux : sa courbe restait a zero jusqu'a 400 s quand le releve terrain atteste 21
   points a 1:30.
2. **Publier le compte de faux positifs, toujours.** 151 faux positifs par film avant la
   contrainte sur les en-tetes de 5 bits, 2 apres : c'est ce chiffre qui separe une mesure
   d'une impression.
3. **Un cadrage se mesure, il ne se derive pas du binaire.** Ghidra donnait
   `[presence][2 bits type][slot]` ; la mesure sur 1 078 en-tetes dit `[presence][14 bits slot]`
   et rien de plus. Les deux bits « de type » etaient du bruit du record precedent.
4. **Une source externe credible se traite comme une hypothese** (guilty-spark PR 752-757 :
   deux revendications sur trois refutees par controle negatif).
5. **Verifier l'origine d'une horloge avant de comparer deux series.** Un decalage de 140 s
   passe totalement inapercu quand les deux series sont monotones.

---

## 8. NOMMER LES EVENEMENTS (2e volet du 2026-08-01) — detail : etat de l'art §16

**Le bareme des actions, mesure et non suppose** : `+100` = un frag · `+50` = une
assistance · `+25` = une action d'objectif · `+300` = une capture de drapeau. Confronte a
deux decodeurs independants (`killsource` pour les morts, `objectiveevents` pour les
objectifs). Cela referme `ObjectivePointsPerCapture = 25.0` d'`engagement_score.go`, pose
« a calibrer ». **Limite** : les increments ne sont pas atomiques (125 = 100 + 25 observe).

**L'identite des entites est resolue** : chaque slot s'attribue a UN joueur par ses
increments de +100 (95/95 expliques) et a UN xuid par ses evenements d'objectif. Le pont
gamertag <-> xuid est referme **sans jointure en base**, et concorde avec les identites
etablies ailleurs. Toutes les courbes par joueur deviennent nominatives.

**88 medailles nommees** sur 95 couples `(type_hint, medal_type)`, sur 948 films. Controle :
tables ajustees separement sur les films pairs et impairs (aucun film partage), 72 couples
communs, **zero desaccord**. Table : `.ai/refs/TABLE_MEDAILLES_FILM.tsv`.

**Ce qui plafonne, et c'est mesure** : le scan d'evenements du footer n'est exact sur un
film entier que dans ~37 % des cas ; applique film par film, le nommage rend 27,3 % de
triplets exacts (11 440 sur-comptes, 3 730 sous-comptes). **C'est le scan qu'il faut
durcir**, pas le nommage.

**Garde a conserver** : tout evenement de score n'est PAS une medaille. Les actions
d'objectif pures (prise, retour, capture) ne sont pas des medailles ; les medailles liees
aux objectifs sont des faits d'armes. Deux canaux complementaires, jamais a confondre.

**Autres modes** : 46 scores finaux exacts sur 61 films KOTH / Total Control / Oddball
(75 %, aucun 0-0 trivial). Les 15 ecarts sont structures : en Total Control le film compte
les points fins et le registre les SETS — facteur 32, sur deux films, vainqueur preserve
(`a521164d` 96-0 / 3-0 · `a349fea8` 32-64 / 1-2). La semantique du composant 0 depend donc
du mode.

**KOTH : « le film compte les collines » est une INTERPRETATION, pas un fait etabli.** Elle
repose sur **deux films**, et sur les **deux** le vainqueur est INVERSE entre le film et
l'API : `606d9844` film 0-3 / API 105-8 ; `8076f97f` film 3-0 / API 78-105. L'appariement des
equipes n'est donc meme pas assure, et aucun autre canal ne vient l'appuyer — le statborg ne
replique **aucun** compteur de colline (etat de l'art §21, mesure sur deux films KOTH) et le
binaire ne declare **aucune** famille de stats KOTH (§19.3). A mettre a l'epreuve avant
d'exposer ce mode ; en l'etat, ne rien en deduire.

### Le point de reprise de ce volet

1. **Durcir le scan de footer** — c'est lui qui plafonne les medailles horodatees.
2. **Le bareme complet des actions d'objectif** se resout par regression entiere
   `personal_score = 100*frags + 50*assistances + somme(bareme x statistique)` sur
   `match_objective_stats_latest`, tous matchs : mesure purement en base, sans film.
3. **Nommer la quantite du composant 0 mode par mode** (sets en Total Control, collines en
   KOTH, manches en Oddball) avant d'exposer ces modes.
4. **7 couples de medaille restent ambigus**, tous vus sur un seul film (vehicules) : ils se
   leveront avec de nouveaux films.

### Outils de ce volet (tous `CGO_ENABLED=0`, non suivis par git)

| outil | role |
|---|---|
| `cmd/tmp_statdump` | recensement des composants d'un film ; `HISTO=1` = histogramme des increments |
| `cmd/tmp_scorenames` | **le nommage** : increments confrontes aux morts et aux objectifs, + identite des slots |
| `cmd/tmp_medalid` | un film : le bloc porte-t-il l'identifiant de medaille ? (non) et nommage par les comptes |
| `cmd/tmp_medalmap` | **le solveur multi-films** ; `HOLDOUT=1` + `PARITY=0/1` = le controle sur moities disjointes |

---

## 9. L'ETIQUETAGE (3e volet du 2026-08-01) — la ligne de temps NOMMEE

**La bibliotheque des evenements de score existait deja dans le depot** : la table
`personal_score_awards` des bases joueur porte `award_name`, `award_category` et
`award_score`, donc le bareme NOMME (`killed_player` 100 · `kill_assist` 50 ·
`flag_captured` 300 · `flag_returned`/`flag_stolen`/`zone_secured`/`runner_stopped` 25 ·
`flag_taken`/`sensor_assist` 10 · `self_destruction` **-100**). Les valeurs mesurees sur le
film en §8 sont donc confirmees par une source qui ne doit rien au film.

**Defaut corrige** : le score personnel N'EST PAS monotone (-100 existe). `PersonalScoreCurve`
n'applique plus aucun filtre de monotonie ; celui de `ScoreCurve` ne vaut que pour le score
de MODE.

**Livre** : `objectiveevents.LabelPersonalScore(points, quotas)` +
`SummarizeLabels` + `LabelledEvent.Describe()`, avec 4 tests.
Bout en bout (`cmd/tmp_awards`, film `1bc77d2e`) : **100 evenements etiquetes, 56 nommes
sans ambiguite, 0 sans nom ni candidate**.

**La regle qui gouverne** : on nomme a la VALEUR, et on se tait quand elle ne discrimine
pas. Un increment compose n'est decompose que si la decomposition est UNIQUE a nombre de
parts minimal — `zone_captured` valant 50 ou 75, un +125 admet `25+100` et `50+75`, il reste
donc brut.

**Prochain lot** : lever les cas restants par **coincidence temporelle** (`runner_stopped`
sur un frag du joueur, `flag_capture_assist` sur une capture de son equipe). Ce sont des
HYPOTHESES : les mettre a l'epreuve avec un controle negatif avant de les coder.
