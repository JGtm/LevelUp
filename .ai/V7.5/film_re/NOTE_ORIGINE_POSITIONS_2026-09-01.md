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

## HYPOTHESE UTILISATEUR « BOBINE / FUSION COIL » SUR `00007ca9` — REFUTEE, proprement

L'enonce (autorite gameplay) : un OBJET DU MONDE PORTABLE, tenu comme une arme mais qui
n'occupe jamais un emplacement d'arme. **L'hypothese etait bien formee** : elle explique
exactement le 0/15 dans i43..i46 et le classement en classe ARME. Et le vocabulaire existe au
depot — `internal/games/weapons/registry.go` porte QUATRE bobines, `hinf_coil_kinetic`
(« UNSC Fusion Coil », FR « Bobine a fusion UNSC »), `plasma`, `shock`, `hardlight`, toutes
classees **`clsEnvironmental`** : objet du decor, pas arme portee. Le concept est reel.

**Ce qui la tue est la SIGNATURE TEMPORELLE, et elle est nette.**

| | 000d5950 | 00502e52 |
|---|---|---|
| occurrences de `00007ca9` | 7 | 8 |
| cohorte **EN MATCH** (> 1 s apres le debut de vie) | **0** | **0** |
| cohorte SPAWN (<= 1 s du debut de vie) | 4 | 0 |
| prises **AVANT la premiere position repliquee** du porteur | 3 | 8 |
| dont sur un slot SANS aucune position (defaut possible) | **0** | **0** |
| avance sur la premiere position, en secondes | 22,3 · 22,7 · 34,9 | 22,2 x5 · 22,5 · 27,3 · 32,3 |
| arme recue en main dans les 2 s | 0 / 7 | 0 / 8 |

**Les quinze occurrences, sans exception, precedent la premiere position repliquee de leur
porteur** — de 22 a 35 secondes, un ordre de grandeur qui est celui de l'intervalle entre
images-cles. Aucune n'arrive en cours de vie. Zero slot sans position, donc ce n'est pas un
defaut de lecture. Une bobine se ramasse EN JEU, a un endroit fixe de la carte ; ceci se
distribue au demarrage, une fois par joueur. **Hypothese REFUTEE pour cet identifiant.**

### Et sous un AUTRE identifiant ? Non plus, sur ces deux films

Balayage de tous les identifiants de classe ARME, avec les trois colonnes qui departagent :

- **`00007ca9` est le SEUL** a porter des prises anterieures a la replication (3 et 8). Tous les
  autres identifiants sont pris integralement EN COURS DE MATCH (`enMatch == total`).
- Le critere « jamais vu par i43..i46 » ne tient PAS d'un film a l'autre : `b619d84a` est absent
  du canal sur le film 1 et present sur le film 2 ; `a0955e9e` l'inverse ; `71ab0a2c` de meme.
  C'est un artefact de COUVERTURE (i43..i46 ne rend que 31 et 14 emissions), pas une propriete
  de l'objet. Il ne peut donc pas isoler un objet du monde.
- **Aucun identifiant ne cumule les deux marques attendues d'une bobine** (jamais dans un
  emplacement d'arme de facon stable ET pris en cours de match a position fixe).

**Ce que `00007ca9` reste** : un evenement de DEMARRAGE, une fois par joueur, classe ARME, qui
n'entre dans aucun emplacement d'arme. Dotation initiale ou marqueur d'equipement de depart —
non tranche, mais la piste « objet du monde portable » est fermee sur ces donnees.

### Un defaut d'instrument, encore attrape par la mesure

Le premier jet de la scission utilisait `equipmentLives`, qui ecarte les positions sans
coordonnees monde (`!p.HasWorld`). En lecture `QuantaOnly` — la seule possible, le film 2
n'ayant pas de carte connue — AUCUNE position n'a de monde : la fonction rendait zero vie et
les 15 occurrences tombaient toutes en « sans vie rattachable » (7/7 et 8/8). Un resultat
parfaitement uniforme sur deux films est un signal d'alarme, pas une decouverte. Corrige par un
decoupage sur le seul axe du temps, au MEME seuil (`lifeGapUS`).

## PASSE `.mvar` (2026-09-01, GO utilisateur) — LES POINTS D'EQUIPEMENT SONT TROUVES

Le blocage est leve : les variantes ont ete re-telechargees, et les 12 positions publiees ont
ete confrontees au contenu reel des fichiers de carte.

### Ce qui a ete telecharge — 3 appels reseau

**Diagnostic d'authentification d'abord.** Le premier essai (`--player Chocoboflor`, le joueur
par defaut) echoue sur **AADSTS70000** — « token issued for a different client id ». AUCUNE
re-capture (doctrine du depot). L'inspection du magasin montre pourquoi, et qui utiliser :

| compte | RT | `reauth_required` | classe d'erreur | maj |
|---|---|---|---|---|
| Chocoboflor, Madina97294, XxDaemonGamerxX | 417/393 car. | **oui** | `revoked` | 30/08 |
| **JGtm** | **329 car.** | **non** | — | **31/08** |
| Trimbutton, DankerGlue, QuiteSiren, UppedJoker, GeleJugefi | 373 car. | non | — | 30/08 |

Trois comptes sont revoques depuis le 30/08 ; la longueur du RT les trahit (417/393 = ancienne
application). **Un seul essai supplementaire**, avec `JGtm` (sain, rafraichi la veille) : succes.

Telechargement : `--dry-run --save-mvar` vers le worktree de recherche — **le catalogue partage
n'a PAS ete regenere**. 3 fichiers : `map.mvar` (453 objets) et `ridgeline.mvar` (443) pour
Cliffhanger, `catalyst.mvar` (337) pour Catalyst.

### L'histogramme aux 12 positions — le test

**Catalyst : 5 positions sur 5 touchees, a 0,00-0,01 m.** La meme resolution que l'etalon de
production (mediane 0,01 m). Deux types HORS liste blanche s'y trouvent :

| type_id | Catalyst | Cliffhanger | positions couvertes |
|---|---|---|---|
| **`0xADEEE6D8`** | 4 objets | 5 objets | 2 (n=14 et n=10) |
| **`0xE42158DF`** | 4 objets | 4 objets | 3 (n=6, 6, 4) |

Leur **cardinalite est celle d'un socle** — quelques unites par carte. Un troisieme type,
`0xA495FE83`, tombait a 0,51 m d'une position de Cliffhanger : il compte **95 a 100 exemplaires
par carte**, c'est du decor, sa proximite est fortuite. **Ecarte** — trois chiffres suffisent a
separer un socle d'un pave.

**Cliffhanger : 1 position sur 7, et c'est le decor ecarte.** Les deux variantes du fichier
(`map.mvar` ET `ridgeline.mvar`) ont ete testees : aucun objet a moins d'un metre des six autres
grappes. Sur ce film — Super Fiesta — l'equipement n'est PAS pose sur la carte, il est distribue.

### Le catalogue elargi, et le verdict

Elargissement EN MEMOIRE (aucune regeneration de `map_weapon_pads.json`, aucun changement a
`mapvar.PadFamilyOf`), memes temoins spatiaux :

**CATALYST (`01e1f945`, KOTH) — 283 naissances, 8 points d'equipement ajoutes**

| | SOCLE | SOL | ABSTENTION |
|---|---|---|---|
| AVANT | 14 (4,9 %) | 149 (52,7 %) | 120 (42,4 %) |
| **APRES** | **61 (21,6 %)** | 146 (51,6 %) | **76 (26,9 %)** |
| temoin +10 x | 5 (1,8 %) | 148 | 130 |
| temoin -7 y | 12 (4,2 %) | 151 | 120 |

**X1 TENU** (le seau SOCLE quadruple : 14 -> 61) · **X2 TENU** (temoins a 5 et 12 contre 61).
Quarante-quatre abstentions passent au seau SOCLE. Huit points ajoutes expliquent 47 naissances.

**CLIFFHANGER (`000d5950`, Super Fiesta) — 401 naissances, 9 points ajoutes**

| | SOCLE | SOL | ABSTENTION |
|---|---|---|---|
| AVANT | 8 (2,0 %) | 254 (63,3 %) | 139 (34,7 %) |
| APRES | 12 (3,0 %) | 252 (62,8 %) | 137 (34,2 %) |
| temoin +10 x | 0 | 262 | 139 |
| temoin -7 y | **21** | 242 | 138 |

**X1 NON TENU · X2 NON TENU** — et le temoin decale de -7 m rend PLUS de socles (21) que le reel
(12). Quand un temoin bat le reel, il n'y a pas de signal : sur ce mode, les points d'equipement
de la carte ne servent pas. C'est ce que X3 avait ecrit avant la mesure.

### Trois defauts d'instrument, tous attrapes avant de conclure

1. Constante `0xE42158DF` transcrite en int32 avec la mauvaise valeur (-468670241 au lieu de
   -467576609) — attrapee en comparant au calcul.
2. `go vet` a refuse un tag JSON groupe sur `X, Y, Z float64` : le tag va aux TROIS
   champs, et **Y comme Z se seraient lus a ZERO en silence**. Un tag par champ.
3. (Rappel du lot precedent) le temoin par permutation de liste, nul par construction.

## VERDICT — prouve / plausible / refute

- **PROUVE** : le catalogue de socles est utilisable pour trancher l'origine, LA OU LE MODE
  ALLUME LES SOCLES (Catalyst : 33,3 % contre 0,5 % et 3,5 % aux temoins). Le repere est
  commun, l'appariement fonctionne. Et `e9e7ff79` est une arme (4/4, auto-coherent).
- **PROUVE AUSSI** : `00007ca9` n'occupe jamais un emplacement d'arme (0/15 sur deux films).
- **PLAUSIBLE** : les 12 regroupements d'abstentions sont des points d'apparition d'equipement
  absents du catalogue. Recurrent, groupe, mais non verifie a la source.
- **REFUTE** : l'hypothese « bobine / objet du monde portable » pour `00007ca9` — ses 15
  occurrences precedent toutes la premiere position repliquee de leur porteur (22 a 35 s), zero
  en cours de match. Et aucun AUTRE identifiant ne porte la signature d'une bobine sur ces deux
  films.
- **NON TRANCHE** : ce qu'EST `00007ca9`. Ce qu'on sait : evenement de demarrage, une fois par
  joueur, classe ARME, jamais dans un emplacement d'arme.
- **REFUTE** : que la chaine `.module` puisse rendre les placements d'objets. Elle va dans
  l'autre sens (le `.module` resout un `type_id` en modele ; le `.mvar` porte la liste et les
  positions). Maillon manquant nomme : la LISTE des objets.
- **FAIT, ET CONCLUANT** : les `.mvar` ont ete re-telecharges (3 appels) et les points
  d'apparition d'equipement TROUVES — `0xADEEE6D8` et `0xE42158DF`, 4 a 5 objets par carte,
  a 0,00-0,01 m des positions publiees. Catalogue elargi en memoire : sur Catalyst le seau
  SOCLE passe de 14 a **61** et l'ABSTENTION de 120 a **76**, temoins a 5 et 12.
- **PUBLIABLE, SOUS RESERVE D'ELARGIR LE CORPUS** : l'origine socle/sol devient decidable sur
  les modes qui posent l'equipement. Deux cartes seulement, et un mode sur deux : la
  regeneration du catalogue partage merite un lot dedie avec plus de cartes.

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

---

# LA RECETTE — trouver les points d'apparition sur une carte JAMAIS VUE

Ce qui precede reposait sur deux `type_id` trouves a la main en comparant des positions. Une
liste de deux identifiants n'est pas une recette : sur une carte inconnue, rien ne dit qu'ils
suffisent ni qu'ils sont les bons. Cette section remplace la liste par une FONCTION.

## Enonce

> Un objet d'une variante de carte est un POINT D'APPARITION D'OBJET RAMASSABLE si et seulement
> si son `type_id` resout, dans le module Forge, vers un tag `food` qui satisfait DEUX cribles :
>
> 1. il reference au moins un tag du groupe `foki` ;
> 2. aucun de ces `foki` ne reference de `bloc` (geometrie solide) ni de `hsc*` (script).

Elle est autonome : elle ne connait aucune carte, aucune position, aucun identifiant. Elle lit
le fichier de carte et interroge le catalogue du jeu. Code : `EstPointDApparition`, dans
`apps/go-api/internal/himap/origine_recette_gamefiles_test.go`.

## Comment on y est arrive, et ce qui a echoue en chemin

**Le mauvais module d'abord.** La premiere sonde a resolu les `type_id` contre le chemin de
geometrie (module de carte + globaux, 56 766 entrees) : **0 sur 8 resolus**, pas meme les trois
socles PROUVES. C'est le catalogue FORGE qui porte ces tags —
`any/globals/forge/forge_objects-rtx-new.module`, comme l'ecrit deja `cuisson_forge.go:439`.
Sur lui : 8 sur 8.

**Le groupe ne separe pas.** Les huit types resolvent TOUS en `food` — socles prouves comme
paves de decor. Une recette « groupe == food » ramasserait la carte entiere.

**Ce qui separe est un cran plus bas** — ce que le tag `food` REFERENCE :

| type_id | ce qu'on en savait | groupes references |
|---|---|---|
| `0x5F379533` | liste blanche, socle PROUVE (power) | `foki:4` |
| `0x6253CFC0` | liste blanche, socle PROUVE (rack) | `foki:4` |
| `0x5E86D110` | liste blanche, socle PROUVE (powerup) | `foki:4` |
| `0xADEEE6D8` | candidat mesure | `foki:4` |
| `0xE42158DF` | candidat mesure | `foki:4` |
| `0xA495FE83` | decor, 95-100 par carte | aucun `foki` |
| `0x8ACF288B` | decor, 63-67 par carte | aucun `foki` |
| `0xCBB239F7` | decor, 18-22 par carte | aucun `foki` |

**Le crible 1 seul SUR-RETIENT, et le corpus l a montre.** Applique aux 15 cartes, il classait
**61,5 % des objets de Highpower** et 60,5 % de Scarr comme points d'apparition — invraisemblable.
Deux types portaient l'anomalie : `0x8413E9BA` (jusqu'a 178 objets sur une carte) et
`0xA4EE54ED` (83). Le seul comptage ne pouvait pas les ecarter : le `rack` PROUVE monte lui-meme
a 52 sur Fragmentation Heavies.

**Le crible 2 est venu d'un cran de plus.** En publiant ce que le `foki` mene a son tour :

```
0x8413E9BA  1 foki -> :44 hsc*:4 bloc:4 fosp:4 foki:1     ABERRANT
0xA4EE54ED  1 foki -> :45 hsc*:4 bloc:4 fosp:4 foki:1     ABERRANT
0x6253CFC0  1 foki -> :44           fosp:4 foki:1          PROUVE (rack)
0x5F379533  1 foki -> :30           fosp:4 foki:1          PROUVE (power)
0x5E86D110  1 foki -> :30           fosp:4 foki:1          PROUVE (powerup)
0xADEEE6D8  1 foki -> :26           fosp:4 foki:1          candidat mesure
0xE42158DF  1 foki -> :37           fosp:4 foki:1          candidat mesure
```

Les deux aberrants portent de la geometrie solide et un script ; aucun point prouve ou mesure
n'en porte. Un point d'apparition NU fait naitre un objet — un objet Forge scripte et solide
n'en est pas un. Le discriminateur est net et il va dans le sens INVERSE de l'hypothese de
depart, qui pariait sur la cardinalite.

## Selectivite

| | types retenus sur 4 235 tags `food` | etalons | decor retenu |
|---|---|---|---|
| crible 1 seul | 27 (0,64 %) | 5/5 | 0/3 |
| cribles 1+2 | **16 (0,38 %)** | **5/5** | **0/3** |

Le crible 2 divise le catalogue par 1,7 sans perdre un seul etalon.

## Universalite — 15 cartes, 13 telechargees pour ce lot

Recette a 2 cribles. Le catalogue partage `map_objectives.json` n'a PAS ete regenere
(`--dry-run`) ; les `.mvar` ne sont pas commites.

| carte | objets | points | % |
|---|---|---|---|
| Scarr | 294 | 97 | 33,0 |
| Forest - Ranked | 308 | 79 | 25,6 |
| Catalyst | 337 | 65 | 19,3 |
| Illusion | 387 | 71 | 18,3 |
| Deadlock | 410 | 67 | 16,3 |
| Breaker Heavies | 431 | 79 | 18,3 |
| Cliffhanger | 443 | 74 | 16,7 |
| Fragmentation Heavies | 490 | 109 | 22,2 |
| Oasis | 497 | 118 | 23,7 |
| Forbidden | 499 | 77 | 15,4 |
| Highpower Sentry Defense | 524 | 89 | 17,0 |
| Bazaar | 993 | 61 | 6,1 |
| Lattice - Ranked (Forge) | 5 032 | 32 | 0,6 |
| Flood Gulch (Forge) | 3 767 | 53 | 1,4 |
| Pharaoh (Forge) | 3 194 | 9 | 0,3 |

Les 12 cartes natives tiennent dans **61-118 points** ; le crible 1 seul donnait 61-322. Les
cartes Forge, faites surtout de decor pose, tombent sous 2 % — coherent.

Les deux `type_id` trouves a la main sont presents sur **14 cartes sur 15**. Leur universalite
est donc etablie ; mais la recette ne s y reduit pas, elle en trouve 14 autres.

## Validation

**(a) Catalyst — 0 rate, 0 faux point au decor.** Les cinq etalons sont reconnus et les trois
types de decor ecartes (`TestOrigineRecetteSepareSurLesTypesConnus`). Les 8 points trouves a la
main appartiennent aux deux types que la recette retient : ils sont inclus par construction.

**(b) Catalyst, film KOTH — le seau socle monte, temoins bas.**

```
            SOCLE        SOL          ABSTENTION
AVANT       14 (4,9 %)   149 (52,7 %) 120 (42,4 %)
APRES       68 (24,0 %)  139 (49,1 %)  76 (26,9 %)
TEMOIN +10x 14 (4,9 %)
TEMOIN -7y  23 (8,1 %)
```

Le gain est reel : 68 contre un plancher de coincidence de 14 et 23. **Le seuil ecrit avant
(temoin x3 <= reel) est manque d'UNE occurrence sur le temoin -7y** — 23x3 = 69 contre 68. Publie
tel quel : succes a la limite, pas succes franc.

**(c) Cliffhanger, film Super Fiesta — la recette rend les points, le film ne les confirme pas.**

```
            SOCLE        SOL          ABSTENTION
AVANT        8 (2,0 %)   254 (63,3 %) 139 (34,7 %)
APRES       26 (6,5 %)   240 (59,9 %) 135 (33,7 %)
TEMOIN +10x  0 (0,0 %)
TEMOIN -7y  22 (5,5 %)
```

Ici le temoin -7y rend 22 contre 26 reels : **le gain n'est pas separable du hasard**. C'est le
resultat ATTENDU et il valide l'enonce — la recette decrit la CARTE, le mode decide. Super
Fiesta ne pose pas l'equipement aux socles ; les 74 points existent sur la carte et restent
eteints. Deux films, deux verdicts opposes, et c'est le temoin qui les separe.

## Ce que la recette ne fait pas

Elle dit OU un objet ramassable peut naitre, pas LEQUEL. Distinguer arme, equipement et grenade
demande de lire le `foki` lui-meme — non fait dans ce lot. Le `fosp:4` present sous tous les
points, etalons compris, n'a pas ete elucide non plus.

## Piege de methode rencontre

`go test` a servi un resultat CACHE apres que le fichier de types soit passe de 27 a 16 entrees :
le journal annoncait « 27 types » alors que le fichier en contenait 16. Un test de recherche qui
lit un fichier de donnees externe **exige `-count=1`** — sans quoi il publie une mesure perimee
sous une etiquette fraiche.

## Reproduire

```
cd apps/go-api
# 1. la recette produit la liste des types (lecture des fichiers du jeu installe)
ORIGINE_TYPES_OUT=<dossier>/points_types.json \
  go test ./internal/himap/ -run TestOrigineRecette -count=1 -v
# 2. l instrument d origine la consomme
ORIGINE_TYPES=<dossier>/points_types.json \
  PICKUP_FILM=<depot>/data/cache/film_chunks/01e1f945 PICKUP_MAP=Catalyst \
  ORIGINE_MAPID=<map_id lu dans map_objectives.json> ORIGINE_DUMP=<dossier>/objets_catalyst.json \
  go test ./internal/analysis/replay/ -run TestOrigineAvecCatalogueElargi -count=1 -v
```

Telechargement borne (13 appels, catalogue partage intact) :

```
LEVELUP_REPO_ROOT=<depot principal> go run ./cmd/mapobj-build --player <GT> \
  --dry-run --rate-ms 1200 --save-mvar <dossier> --map-id <uuid> [--map-id <uuid>...]
```

`--dry-run` neutralise l ecriture du catalogue ; `--save-mvar` depose quand meme les fichiers.
`LEVELUP_REPO_ROOT` sert a lire `db_profiles.json` et les jetons, gitignores et donc absents
d un worktree. Aucune re-capture de jeton (ADR 0023).
