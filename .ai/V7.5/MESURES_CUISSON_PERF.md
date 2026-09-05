# Mesures — cuisson des artefacts de rejeu (PLAN_CUISSON_PERF)

> Tenu par le pilote du chantier. Chaque tableau porte la date, la machine, le commit et la
> commande EXACTE. Une mesure sans ces quatre colonnes n'est pas une mesure.
> Machine de reference : poste de travail de l'utilisateur (Windows 11), serveur de dev local
> actif sur :8000 pendant les mesures sauf mention contraire.

## 1. Reference HEAD (item 0.8) — avant tout refacto

2026-09-02, 18:28-18:37, commit de base `0c70e3bbc` + lot 0 (observateur/instrumentation — aucun
changement de decodage hors 0.4bis), binaire `tmp/replay-build.exe`, un decodage a la fois
(verrou solo), faits par `--facts`, `LEVELUP_LOG_LEVEL=debug`, profils CPU+heap dans `tmp/`.
Machine : poste de travail (serveur de dev actif sur :8000, session en cours — bruit accepte,
meme conditions pour les lots suivants).

| film | variante | chunks | duree totale | stats | killsource | decodage | marshal | pic |
|---|---|---|---|---|---|---|---|---|
| `01e1f945` | KOTH:Arena | 30 | **2 min 24 s** | 0,65 s | 7,9 s | 2 min 15 | 8 ms | 0,20 Gio |
| `7344d24f` | Strongholds | 33 | **2 min 49 s** | 0,75 s | 9,4 s | 2 min 38 | 8 ms | 0,18 Gio |
| `696a9d7c` | Strongholds | 31 | **2 min 43 s** | 0,72 s | 9,4 s | 2 min 33 | 9 ms | 0,20 Gio |
| `1c4c63c2` | BTB One Flag | 69 | TUE au plafond (4,09 Gio) | — | — | — | — | bombe (cf. §2 du plan) |

Durees par balayage (ms, Debug `replay: balayage`) — les trois temoins racontent la meme chose :

| balayage | 01e1f945 | 7344d24f | 696a9d7c | lecture |
|---|---|---|---|---|
| playerIndices | 35 301 | 40 015 | 37 328 | LE poste n°1 (~25 % du decodage) pour une simple table identite->index |
| pads | 12 742 | 15 248 | 14 679 | 6 passes completes (2 archetypes x 3) |
| placements | 11 771 | 14 016 | 13 514 | 4 passes + calibration MPP |
| 6 scanners delta (chacun) | 6 564-6 650 | 7 670-7 875 | 7 633-7 752 | ~40-46 s cumules : bande de slots + layout recalcules par chacun (lot 2) |
| positions / loadouts / inventory / pickups / projectiles / zoneReads | 5 355-6 183 | 6 420-7 196 | 6 192-6 979 | une a deux passes chacun |
| grenades | 763 | 871 | 891 | |
| zoomEvents / deaths / fire / clockOrigin / carrierMarks / bombReads | < 60 | < 60 | < 50 | negligeables |

Enseignements pour les lots : (1) l'assemblage + serialisation sont NEGLIGEABLES (8 ms) — tout le
temps est dans les balayages ; (2) `playerIndices` est le poste n°1 inattendu (35-40 s pour lire
une table de 8 entrees) — candidat prioritaire du lot 1/4 ; (3) les six scanners delta pesent
~45 s cumules, exactement ce que les lots 1 (decompression unique) et 2 (bande/layout partages)
suppriment ; (4) killsource ne pese que 8-9 s : D14 (hors perimetre) est conforte ; (5) les pics
memoire des films sains restent sous 0,21 Gio.

Profils : `tmp/01e1f945.cpu.prof`, `tmp/01e1f945.heap.prof` (+ 7344d24f, 696a9d7c) — a lire par
`go tool pprof` au moment d'attaquer les boucles chaudes (lot 4).

## 2. Divergence des marcheurs de paquets (item 0.7, decision D3)

> **LE MODE `-walkers` N'EXISTE PLUS** (retire au lot 6, 2026-09-03) : il portait EN COPIE les
> trois marcheurs historiques, dont les originaux ont disparu aux items 1.4/1.5 — il ne comparait
> donc plus qu'a lui-meme. Cette section FAIT FOI et reste la reference de la mesure ; la grammaire
> retenue, elle, est rejouee a chaque CI par le test de la mini-bobine de
> `internal/analysis/filmsource` (comparaison a `filmdec.WalkPackets`, paquet par paquet, sur un
> vrai chunk). Pour rejouer la mesure complete sur le cache : commit `aa694442f` ou anterieur.

### 2a. Premiere mesure (grammaire CANDIDATE « arret sur taille 0 », ABANDONNEE)

2026-09-02, 13:41-13:48 (6 min 55), 1 380 films, un enfant borne par film. Resultat : 0 flux
tronque ; `killsource` = candidate partout ; `filmdec` divergent sur TOUS les chunks de
1 378 films (axe `taille_nulle`) ; `objectiveevents` divergent sur exactement 1 chunk par film.
2 repertoires sans aucun `chunk_NN.bin` (`33b9fbe9`, `f8c067d7`) comptes en echec (devenus des
ecartes depuis N7). Le diagnostic paquet par paquet (plus bas) a montre que le paquet supprime
par la candidate etait le TERMINATEUR (type 7, taille 0, dernier, rien apres) : la candidate a
ete abandonnee et D3 REVISEE (le terminateur est EMIS, arret apres). Sortie brute :
`tmp/walkers.tsv`.

### 2b. Mesure de la grammaire RETENUE (D3 revisee) — fait foi

2026-09-02, 15:52-15:59 (6 min 40), binaires corriges (revue ronde 1 integree) :

```
LEVELUP_REPO_ROOT=<worktree> tmp/replay-equiv.exe -walkers -walkers-out tmp/walkers2.tsv
```

| grandeur | valeur |
|---|---|
| films demandes / mesures | 1 380 / 1 378 (2 repertoires sans chunk, ECARTES) |
| films a flux zlib tronque | **0** |
| divergents filmdec vs retenue | 1 378 films — **exactement 1 chunk par film : `chunk_00`** (registre, en-tetes degeneres, jamais consomme comme flux de paquets) ; **0 divergence sur les chunks de DONNEES** |
| divergents killsource vs retenue | 1 378 films — les chunks de donnees, du SEUL terminateur emis (paquet type 7, que killsource n'emet pas et que ses consommateurs filtrent par type) |
| divergents objectiveevents vs retenue (type 0) | 1 378 films — exactement 1 chunk par film : `chunk_00` |

Verdict D3 : sur les chunks de donnees, la grammaire retenue reproduit la vue de `filmdec`
BIT-IDENTIQUE sur tout le cache ; l'ecart killsource est un paquet de type 7 par chunk, filtre
par type en aval (sorties inchangees — a prouver par les empreintes du lot 1) ; `chunk_00` sera
lu comme REGISTRE par `filmsource` (jamais marche en paquets). Le prerequis « lot 1 = refacto
pur » est ACQUIS. Aucun film du corpus ni hors corpus n'est un cas particulier — l'escalade
R-11 (films hors corpus a sortie changee) est SANS OBJET : la grammaire retenue ne change la
vue d'aucun consommateur sur les chunks de donnees. Sortie brute : `tmp/walkers2.tsv`.

Diagnostic paquet par paquet (agent du harnais, programme temporaire supprime, 3 films, TOUS les
chunks) — les paquets que `filmdec.WalkPackets` emet et que la grammaire « arret sur taille 0 »
n'emet pas :

| film | chunks de donnees divergents | paquets en plus par chunk | type | taille | position | octets apres |
|---|---|---|---|---|---|---|
| `000d5950` | 27/27 | 1 | 7 (CHUNK_END) | 0 | dernier | 0 |
| `7344d24f` | 32/32 | 1 | 7 (CHUNK_END) | 0 | dernier | 0 |
| `60ae07c4` | 43/43 | 1 | 7 (CHUNK_END) | 0 | dernier | 0 |

`chunk_00` (registre ECS, pas un flux de paquets) : la grammaire unifiee emet 1 en-tete puis
s'arrete ; `filmdec` en emet 14 de plus, tous de taille 0, aux offsets 43..251 (pas de 16), au
MILIEU du chunk (types 0 ; un type 29806 a l'offset 40 sur `60ae07c4` ; le 14e porte
`ts = 2^40`). C'est le SEUL chunk ou `objectiveevents` diverge (il emet ces memes en-tetes vides de
type 0). Aucun consommateur legitime ne marche `chunk_00` comme des paquets (`ParseRegistryChunk`
le lit comme registre). Conclusion factuelle : sur les chunks de donnees, « taille 0 » et
« CHUNK_END » sont le MEME paquet ; la grammaire retenue (D3 revisee) l'emet puis s'arrete.

## 3. Digests d'equivalence figes (item 0.5)

Grammaire de digest : version 2 (marqueur `# digest-grammar: 2` en tete de chaque TSV).
Corpus ACTIF : 9 films (les deux bombes `1c4c63c2` et `60ae07c4` sont au regime D11,
empreintes au lot du plafond memoire). Chaque verification = une cuisson complete par film,
un processus borne par film.

| passe | date | resultat |
|---|---|---|
| creation (`-update`, 11 films tentes) | 2026-09-02 16:29-17:20 | 9 figes ; 2 tues au plafond (bombes revelees) |
| verification 1 (binaire pre-corrections assemblage) | 17:20-18:05 | 8 identiques, 1 different : `084a804d`, etape `artifact` seule (44 balayages identiques) — non-determinisme d'ASSEMBLAGE revele |
| verification 2 (idem) | 18:05-18:28 | 8 identiques, meme ecart `084a804d` (3e sha distinct : vivant) |
| correction : ordre total sur 6 sites d'assemblage (5 de l'inventaire + `equipmentOwner` manque), 6 tests d'ordre discriminants | 18:15-18:45 | gate vert |
| re-figeage `084a804d` (binaire corrige) | 18:47-19:05 | fige |
| verification finale 1 (9 films) | 19:05-19:45 | **9/9 identiques** |
| verification finale 2 (9 films) | 19:45-20:25 | **9/9 identiques** |

Le determinisme de la cuisson est PROUVE sur le corpus actif : deux verifications completes
rendent les memes 45 empreintes par film, artefact compris. Les 8 films stables l'etaient deja
avant les corrections d'assemblage (aucun ex aequo actif) : leurs references n'ont pas bouge.

## 4. Mesures par lot (1, 2, 4, 6) — meme protocole que §1

_a remplir a chaque cloture de lot_
### Lot 1, apres items 1.1-1.3 (source unique du film) — 2026-09-02, 22:20-23:12

Equivalence : **9/9 identiques** contre les references du lot 0 (45 empreintes par film) — les
items 1.1-1.3 sont un refacto pur, prouve.

| film | L0 (reference) | L1 (items 1-3) | delta |
|---|---|---|---|
| `01e1f945` | 2 min 24 (decodage 2 min 15) | 2 min 35 (decodage 2 min 27) | +8 % — bruit |
| `7344d24f` | 2 min 49 (2 min 38) | 2 min 53 (2 min 39) | +2 % — bruit |
| `696a9d7c` | 2 min 43 (2 min 33) | 3 min 03 (2 min 44) | +12 % — bruit + soiree chargee |

VERDICT HONNETE : aucun gain de vitesse mesurable a ce stade, pics memoire +~0,05 Gio (le film
decompresse reste resident — attendu et borne). L'hypothese « l'inflate x40 pese lourd » est
INFIRMEE pour les films de 20-30 Mo : une passe d'inflate y vaut ~0,3 s (mesure walkers :
33 Go du cache en ~7 min), soit ~12 s sur 144 — noyees dans le cout des boucles de balayage
bit a bit, qui relisent desormais la RAM au lieu du cache de pages de l'OS. Les gains attendus
restent devant : lot 2 (bande/layout/registre partages — les 6 scanners delta pesent 40-46 s
cumules), lot 4 (boucles chaudes — `playerIndices` 35-40 s, lecteurs de bits). La valeur des
items 1.1-1.3 est structurelle : une source unique, la grammaire mesuree, les enveloppes hors
production, les garde-rails — le socle sans lequel les lots suivants ne se prouvent pas.
### Lot 2 (contexte du film partage) — 2026-09-03, 01:05-01:52

Equivalence : **9/9 identiques**. Mesures temoins (meme protocole §6) :

| film | L0 (reference) | L1 | L2 | gain L0->L2 |
|---|---|---|---|---|
| `01e1f945` | 2 min 24 | 2 min 35 | **1 min 36** | **-33 %** — DANS LA CIBLE 60-100 s |
| `7344d24f` | 2 min 49 | 2 min 53 | **1 min 55** | **-32 %** |
| `696a9d7c` | 2 min 43 | 3 min 03 | **1 min 50** | **-33 %** |

Enfants du harnais (vs lot 1) : `000d5950` 1 min 52 -> 1 min 13 ; `084a804d` 19 min 56 ->
**12 min 06** (-39 %) ; `53ce4390` 4 min 07 -> 2 min 41 ; `d9781168` 2 min 55 -> 1 min 58 ;
`9f57c612` 2 min 07 -> 1 min 29. Pics memoire inchanges (0,17-0,45 Gio).

Lecture : mutualiser bande de slots (8->2), detection de layout (6->1) et registre (10-12->1)
vaut ~50-55 s par film moyen — bien plus que l'estimation (~15-20 s), parce que la detection et
la bande sont des passes de resynchronisation bit a bit (le poste chaud du profil) et que le
registre allouait ~1 067 chaines par lecture. Le solde vers <= 100 s partout est au lot 4
(`playerIndices` 35-40 s, lecteurs de bits, consultation de map par bit candidat).
### Lot 4 (boucles chaudes, guide par le profil) — 2026-09-03, 02:30-02:59

Equivalence : **9/9 identiques**. Le profil frais (152 s de CPU) contredisait le plan : les
primitives visees par 4.2 pesaient 0,6 % ; le cout etait `kfReadBits` (58 %) et le lecteur de
`playerIndices` (26 % cumule). Corrections livrees la ou le profil pointait : lecture par mots
de 64 bits (4 primitives, ancienne implementation gardee en oracle dans les tests), recherche
de motif xuid par mot (x97 en micro-mesure), bande de slots dense (struct, 30 sites trouves par
le compilateur), 22 re-balayages de NamedEventsFrom -> 1, cle de tri completee (R-8).

| film | L0 (reference) | L2 | L4 | gain L0->L4 |
|---|---|---|---|---|
| `01e1f945` | 2 min 24 | 1 min 36 | **15,7 s** | **-89 %** |
| `7344d24f` | 2 min 49 | 1 min 55 | **18,6 s** | **-89 %** |
| `696a9d7c` | 2 min 43 | 1 min 50 | **18,2 s** | **-89 %** |

Enfants du harnais : `000d5950` 14,0 s · `64e8adfa` 26,2 s · `53ce4390` 27,5 s · `d9781168`
24,4 s · `9f57c612` 14,3 s · `084a804d` (BTB 26 joueurs) 19 min 54 -> **1 min 40** (x12).
Pics memoire inchanges (0,17-0,43 Gio).

**LA CIBLE 60-100 s EST DEPASSEE PAR LE BAS : tous les temoins cuisent en 14-28 s** (seul le
BTB 57 chunks reste au-dela, a 1 min 40 — contre 19 min 54). Restent : 4b (plafond anti-bombe,
debloque `51101d1d`/`a349fea8`/`1c4c63c2`/`60ae07c4` et la re-cuisson de masse), lot 3 (Live
Fire, apres 4b), lot 5 (orchestration), lot 6 (cloture, revue de branche, gate-push).
### Lot 4b (bornes anti-bombe, les quatre films-bombes cuits) — 2026-09-03, 08:26-08:45

Bornes posees apres l'arbitrage utilisateur : deroulage **<= 100 000 par pas** (premier terme
compris) et **<= 1 000 000 evenements par film**, le SOLDE descendant dans `incrementTimes`
(note N-AV). Cuisson par `tmp/replay-equiv.exe` (binaire rebati), un film par enfant borne a
3 Gio.

| film | variante | avant la borne | duree | pic | deroulages REJETES | evenements emis |
|---|---|---|---|---|---|---|
| `51101d1d` | CTF:Arena Neutral Flag | mort OOM (~26 Gio demandes) | **4,9 s** | **0,08 Gio** | 1 (flag) | 67 |
| `a349fea8` | BTB Heavies:Total Control | mort a 3,17 Gio en 3,6 s | **1 min 48** | **0,48 Gio** | **9 (flag) + 3 (zone)** | 231 589 / 162 175 |
| `1c4c63c2` | BTB:One Flag CTF | mort a 3,96 Gio en 4,5 s | **1 min 54** | **0,68 Gio** | 1 (flag) | 26 413 |
| `60ae07c4` | Ranked:Oddball | mort a 4,02 Gio en 2 min 12 | **25 s** | **0,34 Gio** | 2 (flag) | 3 086 |

Les quatre passent donc de « tues au plafond » a **0,08-0,68 Gio**, soit 4,4x a 37x de marge sous
le plafond de 3 Gio. **Aucun n'atteint le plafond TOTAL** (`tronque=false` partout, maximum emis
231 589 sur 1 000 000) : c'est la borne PAR PAS qui fait tout le travail, le plafond total restant
le second rempart. Les plus gros deroulages refuses : 2 163 333 610 (`51101d1d`, comp 20 B,
slot 24, t = 136 636 ms), 2 148 206 590 et 1 745 602 538 (`60ae07c4`, comps 21 A et 23 A — MEME
slot 12, MEME instant 570 965 ms, signature N-AT de l'enregistrement mal aligne), 1 107 820 492
(`a349fea8`, 21 B), 537 698 416 (`1c4c63c2`, 22 A).

Verification, en trois passes sans `-update` : les 4 bombes **4/4 identiques** (figeage
deterministe) ; les 9 films sains **9/9 identiques** (13,9 s a 1 min 39, pics 0,18-0,42 Gio) ;
et **zero ligne de journal de borne sur les films sains** — la borne de 100 000 ne touche aucun
film sain, verifie et non deduit (pire pas sain mesure : 17 306).

Le corpus d'equivalence actif passe de 9 a **13 films**.

### Verdict final du chantier (item 6.1) — 2026-09-03

| critere du plan (§1) | verdict |
|---|---|
| 1. Equivalence | **TENU** — chaque lot prouve par le harnais (45 empreintes x 9 films contre les references du lot 0) : lots 1, 2, 4, 5 = 9/9 identiques a chaque passage. Les seuls ecarts jamais admis sont les corrections declarees 0.4bis (ordre des ex aequo, qui etait aleatoire). |
| 2. Vitesse | **DEPASSE** — cible 60-100 s ; obtenu **15,7 / 18,6 / 18,2 s** sur les trois temoins (references 144 / 169 / 163 s, soit -89 %). Le pire film sain du corpus (BTB 26 joueurs, 57 chunks) : 19 min 54 -> ~1 min 40. Harnais complet (9 cuissons) : ~5 min contre ~40 min au lot 0. |
| 3. Protections | **TENUES ET ETENDUES** — plafonds, priorite basse, enfant borne, verrou filmdec intacts ; le verrou solo machine couvre desormais les 4 chemins (1 sur 4 avant) ; deadline par cuisson post-sync ; le backfill herite de la priorite basse. |

Reste HORS de ce verdict, en attente de la decision D13 (borne anti-bombe, proposition
100 000) : lot 4b (plafond + figeage des 4 bombes) puis lot 3 (justesse Live Fire). La
re-cuisson de masse (1 380 films) attend le schema vehicules — au tarif mesure, elle vaut
desormais ~1 h 30 - 2 h de machine au lieu des ~55 h projetees a l'audit.

**MISE A JOUR DU 2026-09-03 (lot 4b clos).** La decision D13 est rendue (borne 100 000), les
bornes sont posees et les quatre films-bombes cuisent — cf. la section « Lot 4b » ci-dessus. Le
critere 1 (equivalence) reste TENU : 9/9 sains identiques apres la pose des bornes, aucune borne
activee sur un film sain. Le corpus d'equivalence passe a 13 films. **Reste ouvert : le lot 3**
(layout du catalogue pour les six balayages delta), dont le temoin `60ae07c4` est desormais
cuisable et fige — il n'est plus bloque.
