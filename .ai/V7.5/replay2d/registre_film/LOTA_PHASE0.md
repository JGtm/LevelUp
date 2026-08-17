# Lot A phase 0 — MESURER le score dans le temps (statborg ti=6), tous modes

> Plan : `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`, lot A, items A.0.1 a A.0.5.
> Executeur : worktree `../LevelUp-wt-score-film`, branche `wt/score-film`, base `3fdbb8030`.
> Date : 2026-08-17. Phase de MESURE : lecture seule, aucune publication, aucun code de
> production modifie, aucune ecriture DuckDB.
>
> **VERDICT GATE 0 : NEGATIF.** Les trois seuils ecrits avant la mesure sont manques :
> A.0.1 = 58,3 % (seuil 90 %), 1 mode tenu sur 5 (seuil 4), A.0.2 = 83,3 % (seuil 90 %).
> Aucun seuil n'a ete rebaisse. Les causes sont mesurees et nommees ci-dessous ; deux d'entre
> elles sont des ecarts d'UNITE de l'oracle, pas des defauts du decodeur.

## 0. Instrument, et ce qui le borne

| quoi | ou |
|---|---|
| Mesure par film (A.0.1 a A.0.4) | `apps/go-api/internal/analysis/objectiveevents/score_measure_test.go` (+ `score_measure_oracle_test.go` lecture de l'oracle, `score_measure_probe_test.go` sondes de mode) |
| Controle D1 (voie chaine) | `apps/go-api/internal/analysis/filmdec/statborg_chain_count_test.go` |
| Oracle, exporte UNE fois en lecture seule | `registre_film/oracle_lotA.tsv` (12 lignes de `match_registry`), `registre_film/oracle_lotA_participants.tsv` (117 lignes de `match_participants`), requete : `registre_film/oracle_export.sql` |
| Resultats par film | `registre_film/lotA/<short8>.tsv` (12 fichiers) |
| Resultat du controle D1 | `registre_film/lotA_d1_chaine.tsv` |
| Verdicts des commandes | `registre_film/LOTA_gates.log` |

Gardes d'environnement (aucun film, aucune variable = SKIP propre) :

```
$env:SCORE_FILM   = "C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/<short8>"
$env:SCORE_ORACLE = "<worktree>/.ai/V7.5/replay2d/registre_film/oracle_lotA.tsv"
$env:D1_FILM      = "C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/<short8>"
$env:D1_OUT       = "<worktree>/.ai/V7.5/replay2d/registre_film/lotA_d1_chaine.tsv"
```

Le binaire de test est compile une fois (`go test -c`) puis lance UN FILM PAR PROCESSUS via
`Start-Process -PassThru`, en avant-plan, sous plafond de 3 Go surveille (`PeakWorkingSet64`,
kill au-dela) — regle D17, memoire `reference_statrecords_corpus_sweep_ram_bomb`. Lancer le
binaire plutot que `go test` est ce qui rend la surveillance juste : le pic du pilote `go`
n'est pas celui du decodage.

## 1. Le corpus, et un ecart avec le plan

Le mode de chaque film a ete LU dans `match_registry`, pas suppose. **Le plan §1.2 classe
`06dfe6d9` en « Slayer/temoins » : la base dit `BTB:Fiesta CTF` sur Threshold, 26
participants.** Le corpus reel est donc 4 CTF, 4 KOTH, 2 Strongholds, 1 Oddball, 1 Slayer —
et **Slayer n'a qu'UN film** (`000d5950`), ce qui rend son taux de mode non significatif
quel qu'il soit.

| film | mode (base) | carte | oracle t0-t1 | duree | participants | couverture du film |
|---|---|---|---|---|---|---|
| `64e8adfa` | CTF:Arena | Catalyst | 2-3 | 839 s | 8 | 0,97 |
| `530820e5` | CTF:Arena | Catalyst | 3-0 | 475 s | 8 | 1,00 |
| `53ce4390` | CTF:Arena | Behemoth | 1-2 | 753 s | 8 | 0,99 |
| `06dfe6d9` | BTB:Fiesta CTF | Threshold | 1-3 | 859 s | 26 | 0,99 |
| `0a247154` | Ranked:King of the Hill | Solitude | 4-2 | 787 s | 8 | 1,00 |
| `01e1f945` | KOTH:Arena | Catalyst | 3-2 | 540 s | 8 | 1,00 |
| `606d9844` | KOTH:Arena | Chasm | 105-8 | 235 s | 9 | 0,99 |
| `8076f97f` | KOTH:Arena | Shogun | 78-105 | 349 s | 10 | 1,01 |
| `7344d24f` | Strongholds:Arena | Vagabond | 193-112 | 596 s | 8 | 1,00 |
| `696a9d7c` | Strongholds:Arena | Vagabond | 200-94 | 561 s | 8 | 1,00 |
| `24dbb67d` | Ranked:Oddball | Recharge | 200-121 | 519 s | 8 | 0,98 |
| `000d5950` | Slayer:Arena Super Fiesta | Cliffhanger | 43-50 | 496 s | 8 | 1,00 |

« couverture » = dernier instant d'un enregistrement d'entite / duree du match. Elle importe :
un film qui s'arrete avant la fin ne peut pas porter le score final.

## 2. A.0.1 — score final des slots 6 et 8 contre `team_0_score`/`team_1_score`

Seuil ecrit avant la mesure : accord exact sur l'ENSEMBLE {a,b} pour >= 90 % des films, et
au moins 4 modes sur 5 tenus.

**Convention publiee, et ce n'est pas une devinette** : un composant n'est reemis que
lorsqu'il CHANGE (`statborg.go`, en-tete). Une equipe qui ne marque jamais n'emet donc jamais
le score de mode : son absence EST son zero. Verifie sur `530820e5` (oracle 3-0) : le slot 6
n'a AUCUNE emission, le slot 8 en a exactement trois, une par capture. Le nombre de slots
emetteurs est publie a cote de chaque verdict.

| film | mode | film 6/8 | oracle t0/t1 | accord | emetteurs | pts 6 | pts 8 | monotone |
|---|---|---|---|---|---|---|---|---|
| `530820e5` | CTF | 0 / 3 | 3 / 0 | **exact** | 1 | 0 | 3 | oui |
| `53ce4390` | CTF | 1 / 2 | 1 / 2 | **exact** | 2 | 1 | 2 | oui |
| `06dfe6d9` | CTF (BTB) | 3 / 1 | 1 / 3 | **exact** (croise) | 2 | 3 | 1 | oui |
| `64e8adfa` | CTF | 2 / 2 | 2 / 3 | ecart | 2 | 2 | 2 | oui |
| `0a247154` | KOTH | 4 / 2 | 4 / 2 | **exact** | 2 | 4 | 2 | oui |
| `01e1f945` | KOTH | 3 / 2 | 3 / 2 | **exact** | 2 | 3 | 2 | oui |
| `606d9844` | KOTH | 0 / 3 | 105 / 8 | ecart | 1 | 0 | 3 | oui |
| `8076f97f` | KOTH | 3 / 0 | 78 / 105 | ecart | 1 | 3 | 0 | oui |
| `696a9d7c` | Strongholds | 200 / 94 | 200 / 94 | **exact** | 2 | 190 | 93 | oui |
| `7344d24f` | Strongholds | 200 / 126 | 193 / 112 | ecart | 2 | 194 | 112 | oui |
| `24dbb67d` | Oddball | 100 / 78 | 200 / 121 | ecart | 2 | 100 | 78 | oui |
| `000d5950` | Slayer | 43 / 50 | 43 / 50 | **exact** | 2 | 42 | 48 | oui |

**Taux global : 7 / 12 = 58,3 %** (seuil 90 % : NON TENU).

| mode | denominateur | exacts | taux | tenu (>= 90 %) |
|---|---|---|---|---|
| Slayer | 1 film | 1 | 100 % | oui (1 seul film : non significatif) |
| CTF | 4 films | 3 | 75 % | non |
| KOTH | 4 films | 2 | 50 % | non |
| Strongholds | 2 films | 1 | 50 % | non |
| Oddball | 1 film | 0 | 0 % | non — **negatif de mode** |

**Modes tenus : 1 sur 5** (seuil 4 : NON TENU). Toutes les courbes retenues sont strictement
monotones (par construction de `keepMonotoneBySlot`), sur 3 a 306 points selon le film.

Note de lecture, pour qu'un ecart apparent ne passe pas pour une incoherence : les colonnes
« pts » ci-dessus viennent de `ScoreCurve` (filtre `keepMonotoneBySlot` sur tous les slots a la
fois), tandis que la colonne « pts equipes » du tableau des volumes (§5) vient d'un filtre par
slot suivi d'une deduplication aux changements. Les deux comptes coincident partout sauf sur
`06dfe6d9` (4 contre 5) et `0a247154` (6 contre 7), ou le filtre par slot retient une emission
de plus.

### 2.1 Les cinq ecarts, un par un

1. **`64e8adfa` (CTF, 2/2 vs 2/3)** — le film s'arrete **24,6 s avant la fin du match**
   (couverture 0,97). Les compteurs de joueurs y sont TOUS en sous-comptage coherent (-1 a -4
   frags, -1 a -3 morts, cf. §4.1). Cause : couverture temporelle du film, pas decodage.
2. **`606d9844` et `8076f97f` (KOTH)** — le film porte 0/3 et 3/0, l'oracle 105/8 et 78/105 :
   **deux unites differentes**. Le film compte des MANCHES gagnees, l'oracle des secondes de
   colline cumulees. Le meme mode a l'oracle en manches sur `0a247154` (4-2) et `01e1f945`
   (3-2), et le film y est exact. L'oracle KOTH n'est donc pas homogene, et l'ecart n'est pas
   imputable au decodeur.
3. **`7344d24f` (Strongholds, 200/126 vs 193/112)** — le film DEPASSE l'oracle (200 = plafond
   de score du mode). Non explique : ni troncature (couverture 1,00) ni unite. A reprendre.
4. **`24dbb67d` (Oddball, 100/78 vs 200/121)** — ecart d'un facteur ~2 sur un slot mais pas sur
   l'autre (100 -> 200, 78 -> 121). Les deux sondes lancees sur ce film sont NEGATIVES (§4.2 et
   §2.2). Seul mode a 0/N : **negatif de mode ecrit**.

### 2.2 Sonde d'explication REFUTEE : « current round »

Le composant s'appelle `statborg-current-round-value-stat-component`. Hypothese testee : il
repart de zero a chaque MANCHE et l'oracle porte le cumul, ce que le filtre de monotonie
masquerait. Mesure (`writeRounds`, decoupe de la suite d'emissions a chaque chute de valeur,
segments >= 3 emissions, somme des maxima) :

| film | somme des manches 6/8 | oracle | verdict |
|---|---|---|---|
| `24dbb67d` | 100 / 78 | 200 / 121 | inchange — 1 seul segment par slot |
| `696a9d7c` | 273 / 94 | 200 / 94 | DEGRADE (200 -> 273) |
| `53ce4390`, `64e8adfa` | 0 / 0 | 1/2, 2/3 | ecrases (aucun segment assez long) |

**Hypothese refutee** : aucune remise a zero de manche n'est visible dans le composant 0 A, et
la sonde degrade les films deja exacts. Limite de la sonde a inscrire : son seuil de 3
emissions par segment la rend inapplicable aux modes a faible score (CTF), donc elle ne peut
pas non plus prouver l'absence de manches sur ces films.

## 3. A.0.2 — identite des equipes (cascade D3)

Seuil ecrit : >= 90 % de films resolus.

| film | methode | slot 6 | slot 8 | appui |
|---|---|---|---|---|
| `530820e5` | **(a)** | team 1 | team 0 | scores finaux distincts |
| `53ce4390` | **(a)** | team 0 | team 1 | scores finaux distincts |
| `06dfe6d9` | **(a)** | team 1 | team 0 | scores finaux distincts |
| `0a247154` | **(a)** | team 0 | team 1 | scores finaux distincts |
| `01e1f945` | **(a)** | team 0 | team 1 | scores finaux distincts |
| `696a9d7c` | **(a)** | team 0 | team 1 | scores finaux distincts |
| `000d5950` | **(a)** | team 0 | team 1 | scores finaux distincts |
| `606d9844` | **(b)** | team 1 | team 0 | comp 2 A equipes 8/27 = sommes joueurs 27/8 |
| `8076f97f` | **(b)** | team 1 | team 0 | comp 2 A equipes 38/31 = sommes joueurs 31/38 |
| `7344d24f` | **(b)** | team 0 | team 1 | comp 2 A equipes 58/59 = sommes joueurs 58/59 |
| `64e8adfa` | (c) | — | — | aucun slot joueur apparie (sommes 0/0) |
| `24dbb67d` | (c) | — | — | comp 2 A equipes 24/24 (identiques), sommes 0/0 |

**Resolus : 10 / 12 = 83,3 %** — (a) 7 films (58,3 %), (b) 3 films (25,0 %), (c) 2 films
(16,7 %). Seuil 90 % : **NON TENU**, manque a 1 film pres.

Acquis mesure et reutilisable : **le slot d'equipe porte, en `comp 2 A`, la somme des frags de
son camp.** Verifie sur les 3 films ou (b) a servi, exactement. C'est ce qui rend (b)
operante ; le plan la donnait comme hypothese, elle est desormais mesuree. Les deux films en
(c) sont exactement les deux ou les compteurs de joueurs sont faux (§4) : la cascade D3 ne
degrade donc rien qu'elle n'ait deja perdu ailleurs.

## 4. A.0.3 — compteurs des joueurs (comp 2 A / 2 B / 3 A)

Seuil ecrit : >= 90 % des joueurs exacts, par mode. Denominateur = **96 slots de joueur**
(8 par film x 12 films) ; `match_participants` donne 117 lignes, dont 26 sur le seul BTB
`06dfe6d9` — le statborg n'a que 8 slots de joueur, donc un BTB a 26 participants est
STRUCTURELLEMENT hors de portee au-dela de 8.

**La circularite est levee, et voici comment.** `named.go` documente que le nommage de
`comp 2 A` (frags) a ete etabli en s'en servant d'ancre d'identite : verifier « 2 A == kills »
sur un appariement qui exige cette egalite ne prouve rien. L'instrument apparie donc AUSSI par
le seul couple (`2 B` = morts, `3 A` = assistances), frags LIBRES, puis controle `2 A`. Les
deux appariements sont publies cote a cote.

| mode | slots | compteurs justes | taux | identite + exact, sans circularite | taux |
|---|---|---|---|---|---|
| KOTH | 32 | 32 | **100 %** | 30 | 93,8 % |
| Strongholds | 16 | 16 | **100 %** | 16 | 100 % |
| Slayer | 8 | 7 | 87,5 % | 5 | 62,5 % |
| CTF | 32 | 24 | 75 % | 23 | 71,9 % |
| Oddball | 8 | 0 | **0 %** | 0 | 0 % |
| **total** | **96** | **79** | **82,3 %** | **74** | **77,1 %** |

« compteurs justes » = le triplet lu dans le film existe dans `match_participants` (au moins
une ligne) : c'est la justesse des COMPTEURS, independamment de l'unicite de l'identite.
« identite + exact » = un seul joueur porte ce triplet ET les trois valeurs concordent.
L'appariement par triplet complet donne 77/96 (80,2 %) — l'ecart de 3 slots avec la version
non circulaire est le prix de laisser les frags libres, et il est faible.

**Modes tenus (>= 90 % de compteurs justes) : KOTH et Strongholds (2 sur 5).**

### 4.1 Ce qui plombe CTF et Slayer, mesure slot par slot

- `64e8adfa` (CTF) : **0/8**, et c'est le film tronque de 24,6 s. Les valeurs sont proches mais
  toutes en dessous — slot 16 : 14/12/10 contre 17/12/10 pour `2535449963449748` (-3 frags) ;
  slot 12 : 11/14/9 contre 15/15/9 (-4/-1). Le decodage est bon, la fin du match manque.
- `000d5950` (Slayer) : 7/8 justes mais 5/8 attribues. Les slots 14 et 24 lisent tous deux
  (8,14,1) et DEUX joueurs de l'oracle ont ce triplet : la regle de prudence de
  `slotidentity.go` refuse d'attribuer — refus correct, pas erreur. Le slot 12 lit (13,12,1)
  quand l'oracle a (14,13,1) : -1 frag, -1 mort.
- `8076f97f` (KOTH, 10 participants) : 8/8 justes, 6/8 attribues — deux slots ambigus.
- `06dfe6d9` (BTB, 26 participants) : 8/8 justes, 7/8 attribues sans circularite, 8/8 par
  triplet. Le statborg ne couvre que 8 des 26 joueurs (limite structurelle).

### 4.2 Oddball : negatif de mode, sonde comprise

`24dbb67d` : 0/8. Les frags lus (8,8,8,4,3,3,5,9 ; somme 48) sont sans rapport avec l'oracle
(9,7,15,14,5,20,10,8 ; somme 88), et la couverture du film est de 0,98 — la troncature
n'explique rien. La sonde de mode (`writeProbe`) a balaye **les 58 composants de l'archetype x
les 2 cotes A/B** en confrontant le multi-ensemble des valeurs finales des 8 slots aux
multi-ensembles des frags, morts et assistances de l'oracle : **0 candidat**.

Conclusion ecrite : en Oddball, ni le score de mode ni les trois compteurs de base ne sont
lisibles aux emplacements etablis pour flag et zone. Limite de la sonde a inscrire : elle
exige que les HUIT slots emettent l'emplacement, donc une replication partielle lui
echapperait — c'est la condition de reprise.

## 5. A.0.4 — volume de la publication

Seuil ecrit : <= 60 Ko de plus par artefact en mediane. La charge utile pesee est celle
decrite en A.1.1 du plan (equipes + joueurs, points aux CHANGEMENTS seulement), sérialisée en
JSON avec des instants en millisecondes — en frames du document les entiers seraient plus
courts, la mesure **majore** donc la taille reelle.

| film | pts equipes | pts score perso | pts frags | pts morts | pts assist. | octets equipes | octets joueurs | **total** |
|---|---|---|---|---|---|---|---|---|
| `7344d24f` | 306 | 249 | 121 | 121 | 58 | 6 255 | 11 696 | **17 951** |
| `0a247154` | 7 | 353 | 162 | 161 | 94 | 182 | 16 305 | **16 487** |
| `696a9d7c` | 283 | 225 | 107 | 104 | 45 | 5 759 | 10 294 | **16 053** |
| `01e1f945` | 5 | 216 | 108 | 106 | 48 | 144 | 10 239 | **10 383** |
| `53ce4390` | 3 | 184 | 118 | 121 | 35 | 106 | 9 787 | **9 893** |
| `06dfe6d9` | 5 | 182 | 142 | 88 | 30 | 144 | 9 537 | **9 681** |
| `530820e5` | 3 | 161 | 95 | 97 | 35 | 107 | 8 342 | **8 449** |
| `8076f97f` | 3 | 157 | 74 | 68 | 41 | 107 | 7 355 | **7 462** |
| `000d5950` | 90 | 79 | 63 | 55 | 14 | 1 823 | 4 572 | **6 395** |
| `606d9844` | 3 | 89 | 39 | 39 | 10 | 107 | 4 069 | **4 176** |
| `24dbb67d` | 178 | 0 | 0 | 0 | 0 | 3 563 | 25 | **3 588** |
| `64e8adfa` | 4 | 0 | 0 | 0 | 0 | 125 | 25 | **150** |

**Mediane : 9 065 octets (8,9 Ko). Maximum : 17 951 octets (17,5 Ko). Seuil 60 Ko : TENU**,
avec une marge de 6,6x sur la mediane et 3,3x sur le pire cas. La decimation de repli prevue
par le plan (suppression du score personnel) n'est pas necessaire : le score personnel pese
la moitie des points joueurs et le total reste tres en dessous du seuil.

Les deux plus petits totaux (`64e8adfa`, `24dbb67d`) le sont parce qu'AUCUN joueur n'y est
apparie : leur volume n'est pas une economie, c'est l'absence de donnee.

## 6. Controle D1 — la voie « ancrage » contre la voie « chaine »

Demande du superviseur : compter les records statborg vus par `StatRecords` (ancrage) et par
la boucle de records de `filmdec` (chaine, `DecodeFrameRecords` + `consumeByName`), sur trois
films, et le signaler si la chaine en voit plus de 5 % de plus.

| film | mode | ancrage | chaine (total) | chaine (paquets propres) | chaine (paquets desync) | paquets marches | propres | desync |
|---|---|---|---|---|---|---|---|---|
| `530820e5` | CTF | 712 | 255 | **135** | 120 | 29 148 | 9 385 (32 %) | 19 763 |
| `696a9d7c` | Strongholds | 1 081 | 638 | **248** | 390 | 34 276 | 13 501 (39 %) | 20 775 |
| `000d5950` | Slayer | 578 | 1 530 | **620** | 910 | 30 418 | 11 384 (37 %) | 19 034 |

Le compte « paquets desync » est du BRUIT et doit etre ecarte : passe une desynchronisation, la
boucle lit des positions qui ne sont plus des en-tetes de record, et un slot tombe par hasard
sur une entite liee au statborg. La colonne qui compte est « paquets propres ».

**Ce qu'il faut dire, sans l'arrondir dans un sens ni dans l'autre** :

- sur `530820e5` et `696a9d7c`, la chaine voit **19,0 %** et **22,9 %** de ce que voit
  l'ancrage — quatre a cinq fois MOINS. D1 est confirme sur ces deux films.
- sur `000d5950` (Slayer), la chaine voit **620 contre 578, soit +7,3 %** : au-dessus du seuil
  de 5 % fixe par le superviseur. **Signale.**
- nuance de comparabilite, qui ne doit pas etre passee sous silence : les deux comptes n'ont
  pas exactement le meme denominateur. La chaine compte les RECORDS d'entite d'archetype
  statborg atteints ; l'ancrage ne retient que les enregistrements dont AU MOINS UN composant
  a ete decode (`statborg.go`, `decodeComponents` rend nil sinon). Un +7,3 % de records
  atteints ne signifie donc pas +7,3 % de valeurs de score lisibles.
- dans tous les cas, la chaine desynchronise sur **61 a 68 % des paquets** : elle ne peut pas
  etre la source du score dans le temps sans une refonte du decodage de la chaine, qui n'est ni
  dans ce lot ni dans ce plan.

Aucune ligne de code de production n'a ete touchee pour ce controle ; `filmdec/statborg.go`
(la 2e voie, 0 appelant) n'a pas ete supprime — c'est l'item A.1.3, phase 1.

## 7. Cout machine (D17)

| film | wall (s) | decodage (ms) | Sys (Mo) | Heap apres (Mo) | alloc cumulee (Mo) | pic working set (Mo) |
|---|---|---|---|---|---|---|
| `06dfe6d9` | 1,9 | 1 646 | 17 | 5 | 114 | 16,5 |
| `0a247154` | 1,2 | 977 | 22 | 3 | 87 | 15,4 |
| `64e8adfa` | 1,5 | 897 | 17 | 1 | 90 | 16,2 |
| `53ce4390` | 1,1 | 890 | 17 | 3 | 84 | 15,6 |
| `7344d24f` | 1,0 | 709 | 17 | 2 | 72 | 16,4 |
| `696a9d7c` | 1,3 | 684 | 17 | 0 | 59 | 14,2 |
| `01e1f945` | 1,0 | 666 | 17 | 3 | 64 | 16,5 |
| `000d5950` | 1,0 | 611 | 17 | 2 | 60 | 15,7 |
| `24dbb67d` | 1,0 | 560 | 17 | 3 | 62 | 15,4 |
| `530820e5` | 0,8 | 524 | 17 | 0 | 48 | 14,1 |
| `8076f97f` | 0,6 | 394 | 22 | 2 | 46 | 16,1 |
| `606d9844` | 0,4 | 268 | 17 | 2 | 34 | 15,6 |

Cout mesure sur 3 films (`530820e5`, `696a9d7c`, `000d5950`) AVANT d'enchainer les 12, comme
prescrit. Extrapolation du pire cas : **12 films = ~14 s de decodage cumule, jamais plus de
25 Mo par processus**. Aucun processus tue, aucun approche du plafond de 3 Go — l'ordre de
grandeur est 120 fois sous le plafond. Le controle D1 est plus lourd (4,6 a 9,6 s par film,
pic 52,3 Mo au premier passage) et reste tres loin du plafond.

Lecon a garder : le danger de `StatRecords` n'est pas un film, c'est le CORPUS dans un meme
processus (951 films x ~60 Mo d'allocations cumulees). Un film par processus suffit a le
neutraliser.

## 8. Verdict du GATE 0

| condition du plan | seuil | mesure | verdict |
|---|---|---|---|
| A.0.1 accord global | >= 90 % | 58,3 % (7/12) | **NON TENU** |
| A.0.1 modes tenus | >= 4 sur 5 | 1 sur 5 | **NON TENU** |
| A.0.2 identite resolue | >= 90 % | 83,3 % (10/12) | **NON TENU** |
| A.0.3 joueurs exacts par mode | >= 90 % | 82,3 % global ; 2 modes sur 5 | non tenu (informatif au gate) |
| A.0.4 volume median | <= 60 Ko | 8,9 Ko | tenu |

**GATE 0 NEGATIF.** Selon le plan (« sinon le lot s'arrete sur un negatif ecrit (registre) et
le lot B commence »), la phase 1 du lot A n'est PAS ouverte par cette phase 0. La decision
d'ouvrir malgre tout un perimetre reduit appartient au superviseur et a l'utilisateur, sur les
elements suivants — tous mesures :

1. **Deux modes sont solides sur les compteurs de joueurs** : Strongholds 16/16 et KOTH 32/32
   de compteurs justes. Le score d'equipe y est exact sur 3 films sur 6.
2. **Trois des cinq ecarts d'A.0.1 ne sont pas des defauts de decodage** : `606d9844` et
   `8076f97f` sont des ecarts d'UNITE de l'oracle KOTH (manches contre secondes de colline),
   `64e8adfa` est un film tronque de 24,6 s. Restent deux ecarts vraiment ouverts :
   `7344d24f` (le film depasse l'oracle) et `24dbb67d` (Oddball).
3. **Oddball est un negatif de mode ecrit** (0/1 film, 0/8 slots, sonde a 0 candidat sur
   58 composants x 2 cotes).
4. **Le volume ne sera jamais le probleme** (8,9 Ko median contre 60 Ko autorises).
5. **L'acquis reutilisable** : le slot d'equipe porte la somme des frags de son camp en
   `comp 2 A` (mesure exacte sur 3 films) — c'est la methode (b) de D3, desormais etablie.

## 9. Conditions de reprise (a porter au registre par le superviseur)

| sujet | condition de reprise |
|---|---|
| A.0.1 KOTH | comparer le score de mode a une source de manches (et non a `team_0_score`) ; ou identifier ou le film porte les secondes de colline. Ne pas conclure a un echec du decodeur : sur les 2 films dont l'oracle est en manches, il est exact. |
| A.0.1 `7344d24f` | seul ecart sans cause identifiee : le film donne 200/126, l'oracle 193/112. Verifier si l'oracle est un score tronque a la sortie du dernier joueur. |
| A.0.1 / A.0.3 `64e8adfa` | film tronque de 24,6 s. Reprise = mesurer sur un film dont la couverture est >= 0,99, ou tolerer explicitement un ecart de fin de match. |
| A.0.3 Oddball | la sonde exige que les 8 slots emettent l'emplacement ; refaire avec une correspondance PARTIELLE (>= 6 slots) avant de fermer definitivement. |
| A.0.3 BTB | le statborg a 8 slots de joueur, un BTB en a 26 : plafond structurel, a ecrire dans tout `coverage` publie. |
| D1 | l'ecart +7,3 % sur `000d5950` porte sur des RECORDS ATTEINTS, pas sur des valeurs lisibles. Le trancher demanderait de compter les COMPOSANTS 0 A/1 B/2 A/3 A effectivement decodes des deux cotes — mesure non faite ici (hors perimetre A.0). |
| Sonde « current round » | refutee, mais inapplicable aux modes a faible score (seuil de 3 emissions par segment). |

## 10. Decouvertes (hors perimetre — notees, NON traitees)

1. **Le plan §1.2 se trompe sur `06dfe6d9`** : la base dit `BTB:Fiesta CTF`, pas Slayer. Le
   corpus n'a donc qu'UN film Slayer, ce qui rend le taux de ce mode non significatif. A
   corriger dans le plan par le superviseur (l'executeur ne modifie que les cases A.0.x).
2. **L'oracle KOTH n'est pas homogene** : `team_0_score`/`team_1_score` valent des manches
   (3-2, 4-2) sur deux films et des secondes de colline (105-8, 78-105) sur deux autres, dans
   le meme mode. Tout consommateur de `team_*_score` en KOTH herite de cette ambiguite —
   elle depasse largement ce plan.
3. **Le slot d'equipe replique la somme des frags du camp** (`comp 2 A`) : mesure exacte sur 3
   films. Utilisable comme controle croise, et pas seulement pour l'etiquetage D3.
4. **`match_registry.player_count` est incoherent** avec le nombre de participants : 0 pour
   `530820e5` et `696a9d7c` (8 participants chacun), 3 pour `24dbb67d`. Non utilise par cette
   mesure, mais un lecteur qui s'y fierait se tromperait.
5. **La chaine `filmdec` desynchronise sur 61 a 68 % des paquets delta** de ces trois films
   (mesure de section 6). C'est un chiffre neuf, utile aux lots qui posent des hooks dans
   `consumeByName` (lots B, C, D, P) : un hook dans la chaine ne verra qu'un tiers des paquets.

## 11. Statut des items

- [x] **A.0.1** — mesure sur les 12 films, taux global et par mode, points/instants/monotonie
  publies ; negatif de mode Oddball ecrit. Resultat : 58,3 %, sous le seuil.
- [x] **A.0.2** — cascade D3 mesuree : (a) 7, (b) 3, (c) 2. Resultat : 83,3 %, sous le seuil.
- [x] **A.0.3** — compteurs mesures par mode, circularite de l'ancre `2 A` levee par un
  appariement sur (`2 B`, `3 A`). Resultat : 82,3 % de compteurs justes.
- [x] **A.0.4** — volume mesure sur la charge utile reelle : mediane 8,9 Ko, seuil tenu.
- [x] **A.0.5** — ce journal.
- [x] **Controle D1** — trois films, deux comptes, ecart publie et signale.

Ce qui n'est PAS fait, et ne devait pas l'etre en phase 0 : aucune publication (A.1.x), aucune
suppression de `filmdec/statborg.go`, aucun champ de document, aucun changement de
`traverse.go`, aucune ecriture en base.
