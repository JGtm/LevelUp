# Plan R7-c — L'encodage des valeurs en image-cle : un DRAPEAU DE CONTEXTE dans les lecteurs ?

> Ecrit le 2026-08-17. Suite de `PLAN_R7A_IMAGE_CLE_BIPEDE_ETAT_COMPLET.md` (la FORME) et de
> `PLAN_R7B_BIPEDE_IMAGE_CLE_BIT_EXACT.md` (les DESERIALISEURS). R7-b a corrige trois desers
> sur piece, divise l'ecart median par deux — et laisse le taux d'atterrissage bit-exact a
> **0,54 %**, avec un residu qui est une DERIVE DISPERSEE (p10 46-105 bits, p50 ~500, p90 a
> quatre ordres de grandeur, aucun palier) : chaque champ devie un peu, aucun n'est casse.
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kfflag`, branche `wt/kf-encodage-drapeau`,
> base `wt/kf-biped-bit-exact` = `9b269d7f7` (R3+R5+R6+R7-a+R7-b).
> Execution sous le contrat du skill `plan-execution`.

---

## 1. L'hypothese — et pourquoi elle est la suite logique de R7-b

Un residu qui n'est ni un bloc manquant ni un deser casse, mais « un peu partout », a une
cause de forme connue : **les MEMES deserialiseurs, lus sous un ENCODAGE DIFFERENT**. Le RE
externe l'ecrit noir sur blanc (`.ai/V7.5/film_re/GITHUB_RE_FINDINGS_EN.md`, section
statlines, « Caveat / what's left ») :

> *two encodings coexist — the keyframe (TYPE_2) uses a continuation varint, while the
> per-frame deltas use the 2-bit-selector encoding above.*

Et sa section positions ajoute que l'image-cle porte la position en **float32 brut** la ou le
delta la porte quantifiee — ce que R7-b a mesure independamment (`i0` = 117 bits de mediane,
la MEME sur trois cartes aux decoupages differents, donc un chemin BRUT 96 bits).

**H — Il existe un DRAPEAU DE CONTEXTE, lu par les lecteurs de valeurs eux-memes (pas un bit
du flux), qui bascule l'encodage delta vers absolu/brut.** Si oui, la grammaire d'image-cle
n'est pas une grammaire neuve : ce sont les 64 memes deser, joues avec le drapeau leve.

## 2. Le critere — C1, avec son denominateur

**C1 : un drapeau de contexte identifie dans au moins UN lecteur de valeur (adresse, offset
ou global, valeurs, encodage de chaque cote), ET le port Go du mode « baseline » fait passer
l'atterrissage bit-exact des 591 records `ti=35` bornes des 3 films oracles de 0,54 % a
>= 50 %** (palier intermediaire honnete ; cible 95 %).

Denominateurs R7-a/R7-b, inchanges : 184 (`000d5950`) + 209 (`00502e52`) + 198 (`07aa428d`)
= **591**. Oracle de frontiere : `WalkKeyframeWorld`. Variante de reference : v4 « etat
complet », trous neutralises, corruption-check du mode film ETEINT (R7-b : ALLUME est pire),
largeurs d'axe de la CARTE installees.

**C2** : un negatif chiffre est un livrable. Si aucun drapeau n'existe, ou s'il existe sans
effet mesurable, le dire avec les chiffres et nommer ou l'hypothese casse.

## 3. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17

Canal Ghidra : instance PID 10104, **LECTURE SEULE**, pont MCP HS (« unknown ») ; API HTTP du
plugin `http://127.0.0.1:8089`, endpoint `GET /decompile_function?address=0x...` (le POST JSON
et le parametre `name` sont refuses — verifie ce jour). Un lot jumeau (R7-d) lit la meme
instance : aucun rename, aucun script, aucune analyse relancee.

| lecteur | ce que la decompile montre CE JOUR | drapeau de contexte ? |
|---|---|---|
| `FUN_1406cf008` (R(1)) | ne lit que `+0x2c` (compteur de bits), `+0x30` (registre), `+0x38` (bits en cache) | **non** |
| `FUN_140c18a1c` (int signe a selecteur 2 bits) | idem + `+0x10` (fin) / `+0x40` (curseur octet) | **non** |
| `FUN_14076e524` (vec3 quantifie) | globaux `DAT_144632be0` (largeur d'index) et tables `DAT_1445cc9e0` / `DAT_14462cbe0` — des TABLES, pas une bascule d'encodage | **non** |
| `FUN_1406cfe44` (`i0` position) | `cVar6 = FUN_14076f91c()` — **predicat SANS argument** : `DAT_144e61ea0 != 0` OU `DAT_145121140 == 1`. VRAI vers `FUN_1411b259c` = `FUN_1406d676c(...,0x60)` = **R(96) brut** ; FAUX vers `FUN_14076e524` = **vec3 quantifie** | **OUI** |

**Le port Go n'honore ce drapeau que sur UNE de ses deux branches.** `components_movement.go`
consulte `PositionFullPrecision` dans la branche PREDICTED-DELTA (`:197`) mais **pas** dans la
branche ABSOLUE (`:180-183`), qui appelle `consumeAbsoluteWithGate` sans condition. L'EXE, lui,
teste `FUN_14076f91c()` dans les DEUX. C'est un ecart de port etabli, pas une conjecture — et
c'est le premier fil de la phase 1.

Autres gates de contexte deja modelises cote Go (donc deja connus, a re-verifier et a
completer) : `PositionDeltaHasHandleTail` (descripteur runtime `precIndex != -1`),
`PositionFullPrecision`, `filmComponentCorruptionCheck` (mode film), `simStateComplete`.

## 4. Phases

### Phase 0 — Plan (ce fichier)

- [x] 0.1 Hypothese, critere C1 avec denominateur, etat des lieux sur pieces, phases, gates,
      contrat.

### Phase 1 — LIRE : la table des branches par CONTEXTE

- [x] 1.1 Extraire des sources `filmdec` la liste des adresses EXE des deser du bipede et de
      leurs lecteurs de valeurs, decompiler chacune (Ghidra lecture seule, cache local).
- [x] 1.2 Pour chacune, relever toute branche conditionnee par un OCTET ou CHAMP DE CONTEXTE
      (global `DAT_`, predicat sans argument, champ d'un parametre qui n'est pas le reader) —
      PAS par un bit du flux : adresse, offset/global, valeurs, encodage de chaque cote.
      Cible prioritaire : position, vitals, munitions, statlines, et la famille `FUN_1406cf008`.
- [x] 1.3 Consigner la table dans `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`, section neuve
      « image-cle — drapeau d'encodage ».

### Phase 2 — PORTER et MESURER

- [x] 2.1 Ajouter au contexte Go un mode « baseline » (variable de paquet + setter, comme
      `SetFilmComponentCorruptionCheck`), **sans dupliquer les 64 deser** : le drapeau se lit
      la ou le jeu le lit.
- [x] 2.2 Rejouer l'instrument R7-b en A/B baseline OFF/ON : pourcentage bit-exact, dispersion
      (parts a 8 / 16 / 64 / 256 bits), profil de decrochage par composant.
- [x] 2.3 **Non-regression delta OBLIGATOIRE** : baseline OFF par defaut = comportement
      IDENTIQUE. Suites `filmdec`, `replay`, `killsource` vertes ; gradient
      `cmd/tmp_cleanframe` si le binaire existe dans cette branche.

### Phase 3 — Verdict

- [x] 3.1 POSITIF (chiffres) : ce que ca ouvre. NEGATIF : ou l'hypothese casse, chiffre.
- [x] 3.2 Entree thought_log redigee (NON ecrite) + lignes de registre, dont l'amendement a
      R7-b.

## 5. Gates — commandes exactes, a chaque cloture de phase

```
CGO_ENABLED=0 go build ./internal/analysis/...
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/
gofmt -l internal/analysis/filmdec/                          (doit etre vide)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
CGO_ENABLED=0 KF35_ROOT=<...> go test ./internal/analysis/filmdec/ -run '^TestKF35C' -timeout 60m -v
```

Toute modification d'un fichier PARTAGE du decodeur ajoute :
`CGO_ENABLED=0 go test ./internal/analysis/... ./cmd/killsource/... -count=1`.
**Jamais de verdict de gate lu a travers un tube** (piege R7-a) : le log va dans `.gocache/`,
chemin PERSISTANT, et le verdict se lit sur une ligne `EXIT_*=0`.

## 6. Contrat — NON NEGOCIABLE

1. Fichiers PARTAGES du decodeur modifies UNIQUEMENT pour poser le drapeau la ou le jeu le
   lit, chacun avec sa preuve (adresse decompilee) ET sa non-regression delta.
2. Le mode baseline est **OFF par defaut** : aucun changement de comportement en production
   sans preuve. Aucune bosse de `SchemaVersion`, aucune ecriture DuckDB, aucun rendu, aucune
   string UI, **aucune publication a l'artefact**, aucun balayage de masse (3 films).
3. Lecture seule stricte hors du worktree (le corpus `data/cache/film_chunks` du worktree
   principal est LU, jamais ecrit). Ghidra : LECTURE SEULE, aucun rename, aucun script.
4. `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfflag/.gocache`, **une seule commande `go`
   a la fois**, `CGO_ENABLED=0`.
5. Bascules globales restaurees en `defer` ; `LockProcessDecode` tenu pour tout le test.
6. Pas de Python, pas d'emoji, seuils 500 L / 80 L / 5 parametres respectes.
7. Zero fix opportuniste hors perimetre : toute decouverte va a la section 7, non traitee.
8. Jamais `--no-verify`, jamais `git stash`, jamais `main`, jamais de merge. Commit par phase,
   `git push -u origin wt/kf-encodage-drapeau` apres chaque phase close.
9. Pas d'ecriture dans `.ai/thought_log.md` (le superviseur s'en charge) ; l'entree est
   REDIGEE dans ce plan.

## 7. Decouvertes — consignees, NON traitees

(a remplir en cours d'execution)

## 8. Journal d'execution

(a remplir en cours d'execution)

**2026-08-17 — Phase 0 CLOSE.** Reconnaissance de pre-plan faite AVANT d ecrire le plan : canal
Ghidra retabli (le POST JSON du plugin refuse tout, seul `GET /decompile_function?address=0x...`
repond), quatre lecteurs de valeurs relus ligne a ligne, et UN drapeau de contexte deja trouve —
`FUN_14076f91c`, predicat sans argument sur `DAT_144e61ea0` / `DAT_145121140`, qui bascule le
lecteur de position entre vec3 QUANTIFIE et vec3 BRUT 96 bits. Le port Go ne le consulte que sur
sa branche delta. La phase 1 part de la.

**2026-08-17 — Phase 1 CLOSE. Le drapeau existe, il y en a DEUX, et le port en confond un.**

407 fonctions decompilees et balayees. Resultats, table complete dans
`.ai/V7.5/killweapon/WALK_PORT_NOTES.md` section « IMAGE-CLE — LE DRAPEAU D'ENCODAGE » :

1. **`DAT_144e61ea0` est une PORTEE, pas un reglage** : les huit lecteurs d'etat complet du
   groupe `142e2*`/`142e3*` le levent a 1 juste avant l'appel `vtable[0x60]` (etat par
   defaut) et le remettent a 0 juste apres. Pendant cette portee, `FUN_14076f91c()` est VRAI
   et **six lecteurs de position passent du quantifie au BRUT 96 bits**.
2. **`DAT_145121140` est un SECOND drapeau, distinct**, de configuration process. Il garde
   `i49` (R(2) -> R(4)), `i2 forward-and-up` (autre lecteur) et le bloc MPP. Le port Go les
   confond tous les deux sous `PositionFullPrecision` — c'est une faute de modele : sous la
   portee baseline seule, `i49` NE bouge PAS.
3. **Aucun drapeau dans les primitives** : `FUN_1406cf008` et `FUN_140c18a1c` ne lisent que
   l'etat du lecteur. Le « varint a continuation » annonce par le RE externe n'a aucune trace
   dans le selecteur 2 bits, qui est inconditionnel.
4. **Un ECART DE PORT etabli** : `FUN_1411b259c` = `FUN_1406d676c(...,0x60)` = **R(96)**, et
   TROIS sites Go le traitent comme 0 bit (`consumeAbsoluteWithGate`, `consumeFlockPosition`,
   `consumeE494Position`). Seule la queue d'`i60` portee en R7-b lit les 96 bits.

**2026-08-17 — Phase 2. Le port, et la fidelite du temoin.**

Le modele Go portait les DEUX globaux sous une seule variable. Corrige : `PositionFullPrecision`
ne porte plus que `DAT_145121140` ; `keyframeBaselineScope` porte `DAT_144e61ea0` ; et
`fullPrecisionGate()` porte `FUN_14076f91c` = leur OU. Cinq sites de lecture branchent
desormais sur le predicat, et **quatre d'entre eux lisaient ZERO bit la ou l'EXE lit 96** —
`consumeAbsoluteWithGate`, `consumeFlockPosition`, `consumeE494Position`, et la branche
`predFlag == 1` d'`i0`, qui ignorait la garde tout court. `i49` reste sur `PositionFullPrecision`
SEUL, conformement a sa decompile.

**FIDELITE DU TEMOIN, verifiee.** `TestKF35CDispersion` portee ETEINTE reproduit la table de
dispersion de R7-b **chiffre pour chiffre** (p10 48/105/46, p25 149/285/146, p50 507/636/424,
p75 929/1069/944, p90 69 341/68 131/20 065 ; parts <= 256 bits 31,0/23,9/35,4 %). Le temoin
n'est donc pas une approximation de R7-b : c'est R7-b.

**Non-regression delta : VERTE.** `go test ./internal/analysis/... ./internal/games/halo_infinite/film/... ./cmd/killsource/... -count=1`
-> `EXIT_NONREG=0`, 20 paquets `ok`, 0 `FAIL`. Le defaut reste OFF : aucun bit ne change en
production. `cmd/tmp_cleanframe` n'existe pas dans cette branche — item de gradient sans objet.

## 9. VERDICT — NEGATIF, ET IL EST NET

**C1 ECHOUE, et pas de peu : la portee baseline ne fait pas monter l'atterrissage, elle
l'ANNULE.** 591 records `ti=35` bornes, 3 films, 4 lectures, largeurs de carte installees,
corruption-check eteint, `simStateComplete` allume — une seule variable change entre les deux
colonnes.

| lecture | atterrissage, portee ETEINTE | atterrissage, portee ALLUMEE |
|---|---|---|
| v0b temoin « record NEW » | 0 / 0 / 0 | 0 / 0 / 0 |
| v4 etat complet (64 leaf nus) | 0 / 0 / 0 | 0 / 0 / 0 |
| **v2 etat par defaut + leaf** | **1 / 0 / 1** | 0 / 0 / 0 |
| v3 etat par defaut + porte + leaf | 0 / 0 / 1 | 0 / 0 / 0 |
| **TOTAL sur 591** | **3 = 0,51 %** | **0 = 0,00 %** |

Ecart absolu MEDIAN de la meilleure lecture (v4), en bits :

| film | ETEINTE | ALLUMEE | ecart |
|---|---|---|---|
| `000d5950` | **511** | 543 | +6 % |
| `00502e52` | **636** | 660 | +4 % |
| `07aa428d` | **448** | 526 | +17 % |

Part des records a moins de 256 bits (v4) : 31,0 / 23,9 / 35,4 % eteinte contre
29,9 / 25,4 / 32,8 % allumee — a l'iso, voire en retrait.

**OU L'HYPOTHESE CASSE, ET LA PIECE QUI LE DIT.** Le profil par composant donne la largeur
MEDIANE consommee par `i0` :

| film (decoupage lu dans le film) | `i0`, portee ETEINTE | `i0`, portee ALLUMEE |
|---|---|---|
| `000d5950` (13/13/14, i0 = 45 bits) | 102 | 102 (inchange) |
| `00502e52` (17/17/16, i0 = 55 bits) | **57** | 99 |
| `07aa428d` (18/18/17, i0 = 58 bits) | **62** | 99 |

Eteinte, `i0` mesure **trois largeurs DIFFERENTES**, et chacune suit le decoupage de SA carte
(57 pour un i0 de 55 bits, 62 pour un i0 de 58). Allumee, les trois convergent vers 99 = le
chemin brut 96 bits plus son en-tete — et la mesure se degrade. **Donc le corps d'image-cle
porte une position QUANTIFIEE aux largeurs de la carte, pas un float32 brut : la portee
baseline n'est PAS levee sur le payload que le film stocke.** Le drapeau existe, il est
correctement identifie et desormais correctement porte ; il ne s'applique simplement pas ici.

**CE QUE CA FERME.** L'hypothese « l'image-cle, c'est les memes deser sous un autre encodage »
est REFUTEE pour le seul encodage alternatif que le binaire propose. Les primitives
(`FUN_1406cf008`, `FUN_140c18a1c`) n'ont AUCUN drapeau : le selecteur 2 bits est
inconditionnel, et le « varint a continuation » du RE externe n'a pas de trace dans le
deserialiseur de valeurs. Il n'y a pas de troisieme encodage a chercher sous cette forme.

**CE QUE CA OUVRE, MALGRE TOUT.** Trois acquis qui survivent au negatif :

1. Un ecart de port de PRODUCTION corrige : `FUN_1411b259c` lit 96 bits, quatre sites Go
   lisaient zero. Inerte tant que le drapeau est bas, mais faux des qu'il se leve.
2. Le modele des deux globaux est separe, avec sa preuve — `i49` ne suit pas le meme drapeau
   que la position, ce que le port confondait.
3. **Le lecteur d'etat complet du jeu est NOMME** : `FUN_1428e2a04` -> `FUN_1428e2a9c` ->
   `FUN_142e2bfd0`, qui construit son BitReader (`FUN_1424c7b4c`) sur un paquet lu par
   `FUN_142988338(..., 0x10, 0)` puis, pour chaque entite, leve `DAT_144e61ea0` et appelle
   `vtable[0x60]`. R5 ecrivait « le CONSOMMATEUR du payload type-2 n'est identifie nulle
   part » : une chaine candidate l'est maintenant, et c'est elle qu'il faut confronter au
   demultiplexeur de paquets du film (lot R7-d, l'ecrivain par les vtables).

## 10. Entree thought_log (redigee, NON ecrite par ce lot)

```
### [2026-08-17] Lot R7-c — Le drapeau d encodage de l image-cle existe, et il n est pas leve

Statut : Complete (C1 echoue, negatif chiffre ; un bug de port corrige, un modele separe)

Decision technique. R7-b laissait un residu DISPERSE (0,54 % bit-exact, p10 46 bits, p90 a
quatre ordres de grandeur) : ni bloc manquant ni deser casse. Hypothese testee : les memes
deserialiseurs lus sous un AUTRE encodage, commande par un drapeau de CONTEXTE du lecteur.
407 fonctions du paquet filmdec decompilees (Ghidra LECTURE SEULE, API HTTP du plugin, le
pont MCP restant HS). Le drapeau EXISTE, et il y en a deux : DAT_144e61ea0 est une PORTEE
que les huit lecteurs d etat complet du groupe 142e2*/142e3* levent juste avant l appel
vtable[0x60] et abaissent juste apres ; DAT_145121140 est un reglage de process. Leur OU est
FUN_14076f91c, et il fait passer six lecteurs de position du quantifie au BRUT 96 bits. Le
port Go confondait les deux sous PositionFullPrecision et traitait FUN_1411b259c comme zero
bit alors qu il lit 96 : quatre sites corriges.

Resultats observes. A/B a une seule variable, 591 records ti=35 bornes, 3 films, 4 lectures,
largeurs de carte installees. Portee ETEINTE (temoin, qui reproduit la table de dispersion de
R7-b chiffre pour chiffre) : 3 atterrissages bit-exact sur 591. Portee ALLUMEE : ZERO, et l
ecart absolu median monte de 511/636/448 a 543/660/526. La piece qui tranche est le profil d
i0 : eteinte, sa largeur mediane vaut 102/57/62 bits sur les trois films et suit le decoupage
de CHAQUE carte ; allumee, les trois convergent vers 99 (le brut 96 plus en-tete) et la
mesure se degrade. Le corps d image-cle porte donc une position QUANTIFIEE aux largeurs de
la carte.

Conclusion / prochaine etape. L hypothese est REFUTEE pour le seul encodage alternatif que le
binaire propose, et les primitives n ont aucun drapeau (le selecteur 2 bits est
inconditionnel) : il n y a pas de troisieme encodage a chercher sous cette forme. Ce qui
reste, et qui est neuf : le lecteur d etat complet du jeu est NOMME (FUN_1428e2a04 ->
FUN_1428e2a9c -> FUN_142e2bfd0), la ou R5 ecrivait que le consommateur du payload type-2 n
etait identifie nulle part. C est cette chaine qu il faut confronter au demultiplexeur de
paquets du film. Et une conclusion de R7-b est amendee : i0 ne prend PAS le chemin brut en
image-cle, sa largeur suit la carte.
```

## 11. Lignes de registre (redigees, NON ecrites par ce lot)

```
| 2026-08-17 | R7-c drapeau d'encodage de l'image-cle | MESURE, C1 ECHOUE et le negatif est
NET : la portee baseline `DAT_144e61ea0` existe bien (huit ecrivains, levee autour de
`vtable[0x60]`, lue par `FUN_14076f91c`, six lecteurs de position basculent en brut 96 bits),
mais l'allumer fait tomber l'atterrissage de 3/591 a 0/591 et monter l'ecart median de
511/636/448 a 543/660/526. Piece decisive : la largeur mediane d'`i0` vaut 102/57/62 bits
eteinte — trois valeurs qui suivent les trois decoupages de carte — contre 99/99/99 allumee.
Le corps d'image-cle porte une position QUANTIFIEE. Condition de reprise : AUCUNE sous cette
forme (les primitives n'ont pas de drapeau, le selecteur 2 bits est inconditionnel) ; la piste
vivante est la chaine d'etat complet nommee ci-dessous. |
| 2026-08-17 | AMENDEMENT a R7-b, decouverte n. 2 de sa section « CE QUE CA FERME » | « En
image-cle, `i0` prend le chemin BRUT : 117 bits de mediane, identiques sur les trois cartes »
est FAUX. Re-mesure avec les largeurs de carte installees (meme instrument, meme corpus, et un
temoin qui reproduit la dispersion de R7-b chiffre pour chiffre) : la largeur mediane d'`i0`
vaut **102 / 57 / 62 bits**, trois valeurs distinctes qui suivent les decoupages 13/13/14,
17/17/16 et 18/18/17. La conclusion pratique de R7-b (« les bornes de carte ne sont pas
necessaires pour lire la position a une image-cle ») tombe avec elle : elles le SONT sur deux
films sur trois. |
| 2026-08-17 | REPORT : confronter la chaine d'etat complet `FUN_1428e2a04` -> `FUN_1428e2a9c`
-> `FUN_142e2bfd0` au demultiplexeur de paquets du film | R5 a clos sa phase 2 sur « le
CONSOMMATEUR du payload type-2 n'est identifie nulle part, et c'est ELLE qu'il faut
decompiler ». Ce lot en a trouve une CANDIDATE en cherchant autre chose : `FUN_142e2bfd0` lit
une suite de couples (id 32 bits, type 32 bits) puis, par entite, leve `DAT_144e61ea0` et
appelle `vtable[0x60]` (etat par defaut) — c'est la forme meme d'un payload d'etat complet.
Son BitReader est construit par `FUN_1424c7b4c` sur un paquet obtenu par
`FUN_142988338(..., 0x10, 0)`. Condition de reprise : identifier ce `0x10` dans le
demultiplexeur du film (`.ai/V7.5/killweapon/WALK_PORT_NOTES.md`, section « le demultiplexeur
de paquets du film — et le sort du type-2 ») et verifier si le film emprunte cette chaine. |
```
