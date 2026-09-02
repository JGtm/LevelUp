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
