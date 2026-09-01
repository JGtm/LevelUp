# Origine socle vs sol par les POSITIONS DE CARTE — l'idee marche la ou le mode allume les socles

Date : 2026-09-01. Recherche PURE (aucune publication, aucun fichier de production modifie,
aucune cuisson). Worktree `wt/origine-equipement`, base `05cb536d6`. Instruments :
`apps/go-api/internal/analysis/replay/origine_{positions,ids_muets}_research_test.go`, sous
gardes `PICKUP_FILM` + `PICKUP_MAP` + `ORIGINE_MAPID` et `ORIGINE_FILM`.

## L'INVENTAIRE A CHANGE LE PLAN — ce qui existait deja

1. **Le catalogue des socles EXISTE et vient bien du fichier de carte.**
   `data/titles/halo_infinite/reference/map_weapon_pads.json`, 72 cartes, 1 454 emplacements,
   construit par `cmd/mapopads-build --from <dossier de .mvar>` a partir des variantes UGC
   (Bond CompactBinary v2). **Positions en repere monde, metres, NON transformees — le meme
   repere que les positions joueur du rejeu.** C'est exactement l'entree que l'idee demande :
   il n'y avait pas de catalogue a construire, il y en avait un a EXPLOITER.
2. **L'etalon etait deja mesure**, et il est excellent : le croisement catalogue <-> socles du
   match (`map_weapon_pads.go`) rapporte **32 positions d'oracle sur trois cartes, 32
   appariees, mediane 0,01 m**. Le seuil de production est 1,0 m, « pas une tolerance, une
   marge ». Il n'y avait donc pas d'epsilon a deriver.
3. **Le catalogue ne connait que TROIS types d'objet** : `0x5F379533` (power), `0x6253CFC0`
   (rack), `0x5E86D110` (powerup) — une liste blanche (`mapvar.PadFamilyOf`). Le parseur, lui,
   lit TOUS les objets (443 sur Cliffhanger, 337 sur Catalyst ; 18 et 11 retenus). Les points
   d'apparition d'EQUIPEMENT et de GRENADES, s'ils existent, portent d'autres types et sont
   **ecartes a la construction, pas absents du fichier**.
4. **Le catalogue ne dit pas si un socle est ALLUME.** Note du fichier : « le fichier de carte
   POSE, le mode ALLUME : Cliffhanger porte 17 socles, en rend 10 en CTF et ZERO en Super
   Fiesta ». Ce fait explique tout le resultat ci-dessous.

## ETAPE 2 — BLOQUEE, et le negatif est net

**Les `.mvar` ne sont plus au depot.** Zero fichier trouve (arborescence `Scripts`, `Downloads`,
`Documents`, dossier d'installation du jeu). Le catalogue a ete genere le 2026-08-19 depuis un
dossier de dump qui n'existe plus. **On ne peut donc pas elargir la liste blanche dans ce lot** :
la source manque SUR DISQUE. Ce n'est pas « les .mvar ne declarent pas l'equipement » — c'est
« on ne peut pas le verifier hors ligne aujourd'hui ». La distinction compte, et la suite le
montre : la source est re-telechargeable (voir « VERIFICATION COMPLEMENTAIRE » plus bas).

## ETAPE 3 — L'ETALON, ET CE QU'IL REVELE

Naissances `ti=42` (armes au sol) **restreintes aux APPARUES** — les lachees sont ecartees par
la regle de production (`gwPadsClass`, memes constantes) : melanger les deux mesurerait surtout
des cadavres.

| | Catalyst (`01e1f945`, KOTH) | Cliffhanger (`000d5950`, Super Fiesta) |
|---|---|---|
| naissances totales · lachees · APPARUES | 405 · 204 · **201** | 317 · 181 · **136** |
| **REEL** part <= 1 m d'un socle catalogue | **33,3 %** | **0,7 %** |
| TEMOIN decale +10 m en x | 0,5 % | 0,0 % |
| TEMOIN decale -7 m en y | 3,5 % | 0,7 % |
| **VERDICT E1/E2** | **TENU** (>= 9x les deux temoins) | **NON TENU** |

**Le verdict depend du MODE, pas de l'instrument.** Super Fiesta n'allume AUCUN socle sur
Cliffhanger — c'est ecrit dans le catalogue lui-meme. Il n'y a donc aucune arme a apparaitre sur
un socle, et l'appariement n'a rien a trouver. Sur Catalyst en KOTH, ou dix socles sont allumes,
la separation est franche.

## ETAPE 3 (suite) — L'ORIGINE DES NAISSANCES `ti=37`

| | Catalyst | Cliffhanger |
|---|---|---|
| naissances `ti=37` · fins de vie de bipede | 283 · 162 | 401 · 138 |
| distance mediane au socle catalogue le plus proche | 6,17 m | 5,60 m |
| **SOCLE** (<= 1 m d'un emplacement catalogue) | 14 (**4,9 %**) | 8 (**2,0 %**) |
| temoins sur ce seau (+10 x / -7 y) | 0,4 % / 0,0 % | 0,0 % / 1,0 % |
| **SOL** (regle de production `gwPadsClass`) | 149 (**52,7 %**) | 254 (**63,3 %**) |
| **ABSTENTION** | 120 (**42,4 %**) | 139 (**34,7 %**) |

Le seau SOCLE est REEL (4,9 % contre 0,4 % au temoin) mais **minuscule** — et c'est attendu : le
catalogue ne connait qu'**UN** emplacement `powerup` par carte. Le seau SOL domine, ce qui
confirme le canal deja mesure : l'equipement tombe surtout des corps.

## LE RESULTAT LE PLUS UTILE — les abstentions se REGROUPENT

Si les abstentions etaient du sol quelconque, elles seraient eparpillees. Elles ne le sont pas.

| carte | abstentions | regroupements <= 1,5 m | dont **>= 3 naissances** |
|---|---|---|---|
| Catalyst | 120 | 82 | **5** |
| Cliffhanger | 139 | 109 | **7** |

**Catalyst — les cinq points presumes** (14, 10, 6, 6 et 4 naissances) :
`x=15,24 y=-0,00 z=25,25` · `x=0,00 y=-3,41 z=23,30` · `x=-17,12 y=0,01 z=22,42` ·
`x=15,67 y=-0,00 z=22,86` · `x=-7,95 y=-7,71 z=24,33`.

**Cliffhanger — les sept** (13, 5, 5, 3, 3, 3, 3) : `x=16,63 y=8,62 z=-2,41` ·
`x=-40,89 y=48,59 z=-49,93` · `x=18,77 y=3,46 z=-2,49` · `x=5,02 y=-10,62 z=-1,27` ·
`x=24,40 y=6,78 z=-1,18` · `x=-6,05 y=-26,74 z=38,77` · `x=-6,05 y=-30,30 z=38,74`.

**Ce sont tres probablement les points d'apparition d'equipement que le catalogue ne connait
pas** — la liste blanche a trois types les a ecartes a la construction. Douze positions, sur
deux cartes, ou 4 a 14 objets naissent chacun au cours d'un seul match : un sol quelconque ne
fait pas ca.

**Ce n'est pas une preuve**, et il faut le dire : la recurrence est compatible avec un point
d'apparition, elle ne l'etablit pas (une zone de combat tres frequentee produirait aussi des
morts groupees — mais celles-la sont deja dans le seau SOL). La preuve demande de rouvrir un
`.mvar` et de regarder ce qui s'y declare a ces coordonnees.

## UN DEFAUT D'INSTRUMENT, ATTRAPE PAR LA MESURE ELLE-MEME

Le premier jet utilisait comme temoin une **permutation de la liste** des socles. Les chiffres
sont sortis **rigoureusement identiques au reel** (18,5 % et 5,72 m des deux cotes) : permuter
une LISTE ne change pas l'ENSEMBLE des positions, donc la distance au plus proche est la meme.
Controle nul. Remplace par deux translations independantes (+10 m en x, -7 m en y). Un temoin
qui egale le reel au chiffre pres n'est pas un resultat surprenant, c'est un temoin casse.

## ADDENDUM — les deux identifiants muets sont TRANCHES

Methode : pour chaque ramassage portant l'identifiant, quelles armes le RAMASSEUR recoit-il en
main dans les 2 s (canal i43..i46) ?

| identifiant | occurrences | classe | arme recue en main | est-il une famille vue par i43..i46 ? |
|---|---|---|---|---|
| **`e9e7ff79`** | 4 (film 1 ; absent du film 2) | 0 | **`e9e7ff79` elle-meme, 4/4** | **OUI**, 4 emissions |
| **`00007ca9`** | 7 + 8 = **15** (les deux films) | 0 | **AUCUNE, 0/15** | **NON**, 0 emission |

- **`e9e7ff79` EST UNE ARME**, et une arme ordinaire : le ramassage natif et le canal porteur
  disent la meme chose, 4 fois sur 4. Elle manque simplement a la table de libelles
  (`weapon_names.toml`). **A verser au chantier catalogue d'armes** — ce n'est pas un mystere,
  c'est un trou de nommage.
- **`00007ca9` N'EST PAS UNE ARME DU CATALOGUE.** Ramasse 15 fois, toujours en classe ARME, il
  **n'entre JAMAIS dans un emplacement d'arme** et n'apparait dans AUCUNE emission i43..i46 des
  deux films. Sa forme le disait deja : 31 913, une valeur basse, pas l'allure des autres
  identifiants (2 a 4 milliards). Rappel de mesure anterieure (lot 1) : ses 7 occurrences du
  film 1 tombent **dans les 180 premieres millisecondes du match, une par joueur** — la
  signature d'une DOTATION a l'apparition, pas d'un ramassage. Hypothese la plus economique :
  un objet tenu de dotation, ou un marqueur d'equipement de depart. **Non tranche.**

Recoupement consigne sans conclusion, comme demande : `0xE9E7FF79` a deja ete releve par le
chantier Assaut comme mot MPP recurrent (4 occurrences). Deux apparitions d'un meme motif ne
font pas un lien ; c'est note pour qui reprendra.

## VERDICT — prouve / plausible / refute

- **PROUVE** : le catalogue de socles est utilisable pour trancher l'origine, LA OU LE MODE
  ALLUME LES SOCLES (Catalyst : 33,3 % contre 0,5 % et 3,5 % aux temoins). Le repere est
  commun, l'appariement fonctionne. Et `e9e7ff79` est une arme (4/4, auto-coherent).
- **PROUVE AUSSI** : `00007ca9` n'occupe jamais un emplacement d'arme (0/15 sur deux films).
- **PLAUSIBLE** : les 12 regroupements d'abstentions sont des points d'apparition d'equipement
  absents du catalogue. Recurrent, groupe, mais non verifie a la source.
- **NON TRANCHE** : ce qu'EST `00007ca9`.
- **REFUTE** : que la chaine `.module` puisse rendre les placements d'objets. Elle va dans
  l'autre sens (le `.module` resout un `type_id` en modele ; le `.mvar` porte la liste et les
  positions). Maillon manquant nomme : la LISTE des objets.
- **PLUS BLOQUE, mais NON EXECUTE** : l'elargissement du catalogue. Les `.mvar` sont
  re-telechargeables par `cmd/mapobj-build --save-mvar` (API UGC, auth en place). C'est une
  commande reseau sur le compte de l'utilisateur — hors perimetre d'un lot de recherche hors
  ligne. Sequence prete a lancer ci-dessus.

## VERIFICATION COMPLEMENTAIRE — la chaine `.module` rend-elle les `.mvar` ? NON, mais il y a mieux

Question posee : le chantier cartes ayant resolu l'extraction de la geometrie 2D depuis les
`.module` du jeu installe, cette meme chaine permettrait-elle de re-extraire les placements
d'objets — et donc de lever le blocage sans dossier de dump ?

**REPONSE : NON, et le depot le documente noir sur blanc.** La chaine Forge etablie le
2026-08-10 (`internal/himap/cuisson_forge.go`, sondes `sonde_forge_gamefiles_test.go`) va dans
l'AUTRE SENS :

```
objet .mvar --type_id--> tag `food` (GlobalID, forge_objects-rtx-new.module)
           --refs inline--> tags `rtgo` (maillages)
           --Pos/Up/Forward--> repere monde
```

Le `.mvar` est la SOURCE des objets — leur liste, leurs positions, leur `type_id`. Le `.module`
ne fait que RESOUDRE un `type_id` en modele 3D. C'est exactement le piege « canevas + rack »
du lot 5 cartes, et le depot le chiffre : « Vagabond n'est pas cuite — **788 instances dans son
canevas contre 4 709 objets dans son `.mvar`**. Sa carte est le RACK D'OBJETS, pas le module. »

**LE MAILLON MANQUANT EST DONC PRECIS** : la LISTE des objets et leurs positions. Seul le
`.mvar` la porte ; aucun `.module` ne la contient. `himap` CONSOMME des `.mvar`
(`cartes_forge.go`, `callouts.go`), il n'en PRODUIT pas.

### MAIS LE BLOCAGE TOMBE QUAND MEME — par une autre voie, deja au depot

En cherchant, j'ai trouve mieux que la voie `.module` : **`cmd/mapobj-build` TELECHARGE les
`.mvar`** depuis l'API UGC Discovery de Halo (authentifiee, jeton Spartan de l'auth existante,
ADR 0023, aucune re-capture). Il porte exactement les trois drapeaux qu'il faut :

| drapeau | ce qu'il fait |
|---|---|
| `--save-mvar <dossier>` | depose chaque `.mvar` telecharge — **la source qui manquait** |
| `--from-file <x.mvar> --dump-objects <out.json>` | ecrit **TOUS** les objets de la variante (diagnostic) — **l'histogramme des `type_id` demande** |
| `--refresh-from <dossier>` | regenere tout le catalogue HORS LIGNE depuis des `.mvar` locaux |

La condition de reprise n'est donc PAS « attendre un dump perdu » : c'est **une commande
reseau**, sur le compte Halo de l'utilisateur. Les jetons sont en place
(`data/auth/watcher_tokens/` porte un xuid reel).

**JE NE L'AI PAS EXECUTEE.** Ce lot est une recherche hors ligne en lecture seule ; declencher
des appels authentifies sur le compte Halo de l'utilisateur, ecrire des `.mvar` et regenerer un
catalogue partage sortent de son perimetre. La decision revient a l'utilisateur, et elle est a
une commande pres.

### La sequence exacte, prete a lancer

```
# 1. Recuperer les deux variantes (un appel reseau par carte, politesse 1 s)
go run ./cmd/mapobj-build --player <Gamertag> --save-mvar <dossier>     --map-id <map_id de Catalyst> --map-id <map_id de Cliffhanger>

# 2. Histogramme de TOUS les objets d'une variante — le test des 12 coordonnees
go run ./cmd/mapobj-build --from-file <dossier>/catalyst_catalyst.mvar     --map-id <map_id> --dump-objects /tmp/objets_catalyst.json

# 3. Chercher, dans ce dump, les objets situes aux 12 positions publiees plus haut :
#    le(s) type_id qui s'y trouve(nt) est le type d'apparition d'equipement cherche.

# 4. Elargir `mapvar.PadFamilyOf` a ce(s) type_id, puis regenerer le catalogue des socles
go run ./cmd/mapopads-build --from <dossier>

# 5. Rejouer l'instrument : le seau ABSTENTION doit s'effondrer au profit du seau SOCLE.
```

## Condition de reprise

1. **Re-dumper les `.mvar`** des cartes du corpus (la voie d'origine du 2026-08-19).
2. Histogrammer TOUS les `type_id` du fichier — le parseur les lit deja — et regarder ceux qui
   tombent aux **12 coordonnees publiees ci-dessus**. Un type qui s'y trouve est le type
   d'apparition d'equipement cherche.
3. Elargir `mapvar.PadFamilyOf` a ce(s) type(s), regenerer le catalogue, rejouer cet instrument :
   le seau ABSTENTION doit s'effondrer au profit du seau SOCLE. C'est le test qui tranchera.
4. Verser `e9e7ff79` au catalogue d'armes ; laisser `00007ca9` en suspens avec sa signature
   (classe arme, jamais en main, groupe au debut de match).

## A ne pas confondre

`origine_poses_research_test.go` (deja au depot, autre session) repond a une AUTRE question :
dotation au spawn vs objet deploye, pour les poses `equipmentPlacements`. Celui-ci porte sur
l'ORIGINE SPATIALE des naissances `ti=37`. Les deux ne se recouvrent pas.

## Reproduire

```
cd apps/go-api
PICKUP_FILM=<depot>/data/cache/film_chunks/01e1f945 PICKUP_MAP=Catalyst \
  ORIGINE_MAPID=<le map_id de Catalyst, lu dans map_weapon_pads.json> \
  go test ./internal/analysis/replay/ -run TestOrigine -v
ORIGINE_FILM=<depot>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/replay/ -run TestOrigineIdentifiantsMuets -v
```

Le `map_id` ne figure pas en clair dans cette note : `gitleaks` le prend pour un secret (faux
positif — c'est un identifiant public, deja present dans `map_weapon_pads.json` et
`map_objectives.json`). Le lire dans le catalogue evite d'ouvrir une exception de configuration
pour un lot de recherche.

Un film par process, lecture seule, aucune cuisson.
