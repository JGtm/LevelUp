# Les callouts des cartes Forge — ou ils vivent, comment les lire

> Ecrit le 2026-08-27, le jour ou l'utilisateur a refuse une affirmation que nous avions
> ecrite partout : « une carte Forge n'a pas de zones de callout ». Sa question tenait en une
> phrase — « En jeu je les vois, c'est peut-etre un fichier non telecharge ? » — et elle a
> renverse la conclusion. Code : `internal/analysis/replay/mapvar` (lecture),
> `internal/himap/zones_forge.go` (geometrie et tri).

## 1. La reponse

**Les callouts d'une carte Forge sont dans son `map.mvar`, et nous l'avions deja.** Aucun
fichier ne manquait : le `sha256` du `map.mvar` telecharge depuis le stockage UGC d'Isolation
est celui de `.ai/re_dump/mapvar/isolation_map.mvar`, en depot depuis le 31/07.

Chaque zone nommee est un OBJET de la variante :

| ou | quoi |
|---|---|
| `type_id` | **-696190206** (`himap.TypeIDZoneNommee`) — 4 161 objets sur 4 161 |
| `#8/4[]/0/0` | le **StringId du lieu** — chemin UNIQUE, mesure sur les 257 variantes |
| `#8/0[]/0[]` | la forme Forge ordinaire (famille, dimensions, hauteurs) |
| `#3`, `#5` | position et vecteur avant de l'objet |

Le StringId se resout contre le tableau de **778 entrees** du tag global `locs` (bloc a
`locs+0x120`, pas de 4 octets) — **le meme vocabulaire que les cartes natives** : 439 des 463
chaines de `callouts_i18n.csv` y figurent.

**Isolation : 18/18 StringId resolus** — *bottom mid*, *cave* (x4), *top mid*, *north base*,
*south base*, *pipes* (x2), plus 8 entrees de `locs` que notre CSV ne nomme pas encore.

## 2. Ce qui etait faux, et pourquoi personne ne l'avait vu

L'erreur n'etait pas une mesure fausse mais une GENERALISATION fausse. Le zero mesure portait
sur les **canevas** (8 canevas installes, 12 volumes anonymes chacun, tous `kind=1`, aux bornes
de la boite ±212,5/250 m : ce sont les barrieres). De la, nos en-tetes ont conclu au zero des
**cartes** Forge. Les deux affirmations n'ont rien a voir.

**Un balayage d'entiers LE32 du `.mvar` rend ZERO occurrence du vocabulaire.** Les entiers Bond
sont des varint zigzag : il faut decoder l'arbre. C'est pour cela que la piste avait ete
fermee — la recherche naive confirmait l'absence.

Trois en-tetes affirmaient le contraire de la mesure et ont ete corriges le meme jour :
`replay/callouts_catalog.go`, `service/replay_map_callouts.go`, `himap/callouts.go`.

## 3. Le piege : le RATELIER

Deux variantes du dump portent des objets de zone qui ne sont PAS des zones :

- `illusion_map.mvar` : **57 boites identiques de 0,5 m**, alignees a `x = -13,660`, pas de
  0,6 m en y ;
- `forbidden_map.mvar` : **33 boites de 1,0 m** sur une grille a `z = 8,5` constant, portant des
  noms venus de toute la franchise (*ridgeline icicle*, *oscars house*).

Ce sont des PALETTES d'objets non poses. Corroboration croisee sur `illusion` : les 43 string_id
du `levl` et les 57 du `.mvar` sont **disjoints**, et l'emprise du `.mvar` est une DROITE.

`himap.ratelier` les ecarte : quand 90 % des zones partagent un gabarit identique, c'est une
palette. Une carte donne a chaque lieu la taille de son lieu.

Decompte honnete : **4 161 objets bruts -> 3 971 zones reellement posees**, sur 96 fichiers de
variante = 75 cartes distinctes.

## 4. Premier usage : rogner le fond de carte (Isolation)

`OptionsCuissonForge.RogneAuxZones` — jumeau exact du levier natif, a ceci pres que les zones ne
sont pas fournies mais LUES DANS LES OBJETS deja charges par la cuisson. La mesure est
inconditionnelle, le rognage se decide carte par carte.

Mesure sur Isolation (1 162 199 cellules de matiere, apres rognage au maillage) :

| marge | matiere hors zones | part | ancres au sol |
|---|---|---|---|
| 1 m | 306 128 | 26,3 % | 24/25 |
| 4 m (defaut) | 198 738 | 17,1 % | 24/25 |
| 8 m | 117 889 | 10,1 % | 24/25 |

**L'ORACLE MORD, ET IL FAUT L'ECOUTER** : la recette au maillage seul tient **25/25** ancres au
sol ; le rognage aux zones en coute une, et la marge de 8 m ne la recupere pas. Une ancre
d'objectif est du terrain joue par definition — les zones de callout d'Isolation ne couvrent
donc PAS tout le terrain joue. Le rognage aux zones est un levier de plus, pas un remplacant du
maillage.

**VERDICT UTILISATEUR, 2026-08-27** : « valide avec + zones, marge 1 m ». La marge la plus
SERREE des trois — celle qui laisse le moins de gribouillis autour des zones — et l'ecart d'une
ancre est accepte en connaissance de cause. La recette d'Isolation cumule donc les deux
sources : maillage pour la reference et le gros du nettoyage, zones de callout pour la coupe
fine. `map_fond_reglages.json`, cle `01af558d`.

## 5. Ce qui reste

- **Le texte joueur manque pour 63 % des zones** : 434 StringId distincts employes, 432 dans
  `locs`, mais **160 seulement** portent un libelle dans `callouts_i18n.csv`. Les 274 autres
  demandent une extraction `uslg` — la meme chaine qui a produit le CSV. **C'est le vrai reste
  de travail, plus gros que le decodage.** La geometrie, elle, n'a pas besoin des noms.
- **Le rejeu 2D n'affiche toujours pas les callouts des cartes Forge** : il manque le catalogue
  cote donnees (cle `map_id` a cote des cles-module) et l'essai `map_id` au service. Le chemin
  est trace dans les deux en-tetes corriges.
- **31 zones sur 4 161 ont des formes aberrantes** (30 sur `argyle`, 1 sur `detachment`) :
  l'emplacement 8 vaut `0xFFC80000` = **-56,00 m** en virgule fixe 16.16, l'emplacement 7 est
  parfois absent. L'anomalie est dans la donnee, pas dans le lecteur.
- **10 zones portent un StringId hors de `locs`**, dont 2 sur `live_fire` a `sid = 0`.
- **La concordance de repere ne tient qu'a `chasm`** (2 zones `.mvar` contre 23 zones `levl`,
  meme repere monde, pas de decalage). C'est un oracle, pas une preuve statistique.

## 6. Ce que le navmesh, lui, ne porte pas

La troisieme region du `navmesh.blob` (`hkaiTraversalAnnotationLibrary`) a ete ouverte et
entierement decodee dans la meme passe : c'est une table de **liens de saut** (190 sur Isolation,
36 sur Kiken'na) — franchir un vide, descendre d'une plateforme, grimper. Le signe du denivele
etablit la semantique : type 1 toujours positif (+0,64 a +2,93 m), type 4 toujours negatif aux
memes bornes (les liens miroirs), type 0 median nul (a plat).

**Elle ne porte AUCUNE chaine**, et ce n'est pas un sondage mais une preuve : un fichier-tag
Havok est integralement auto-descriptif, et la table des types des 4 regions ne declare aucun
type de chaine. Temoin : `hinavmesh.TestNavmeshNePorteAucuneChaine`.
