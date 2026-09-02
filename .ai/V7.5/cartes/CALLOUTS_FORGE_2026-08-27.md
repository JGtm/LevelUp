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

> **MAJ 2026-09-02** — le rejeu 2D affiche desormais les zones des cartes Forge (section 7),
> et TOUTES portent leur nom depuis le decodage du lexique `uslg` (section 8).

- ~~**Le texte joueur manque pour 63 % des zones**~~ **LIVRE le 2026-09-02** (section 8) :
  l'extraction `uslg` est faite, le lexique versionne resout **266/266** StringId de la
  rotation et **2 536 zones sur 2 536** portent un libelle FR/EN.
- ~~**Le rejeu 2D n'affiche toujours pas les callouts des cartes Forge**~~ **LIVRE le
  2026-09-02** (section 7).
- **31 zones sur 4 161 ont des formes aberrantes** (30 sur `argyle`, 1 sur `detachment`) :
  l'emplacement 8 vaut `0xFFC80000` = **-56,00 m** en virgule fixe 16.16, l'emplacement 7 est
  parfois absent. L'anomalie est dans la donnee, pas dans le lecteur. **Traitement 2026-09-02 :
  `himap.prismeLisible` REFUSE ces enregistrements** (87 zones sur 2 566 de la rotation,
  3 cartes) plutot que de les redresser — les emplacements 5 a 8 se lisant a la file, une
  valeur impossible en 8 met en doute la largeur et la profondeur, donc le polygone.
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

## 7. Alimentation du catalogue — LIVRE le 2026-09-02

### Ce qui bloquait, exactement

Le catalogue `map_callouts.json` n'avait qu'UN espace de cles, le module installe. Une carte
Forge n'a pas de module — le jeu ne range sous `levels/multi/` que les cartes integrees et les
canevas. `replayService.MapCallouts` s'arretait donc apres l'essai par module, et
`/matches/{id}/replay/callouts` repondait 404. Cote web, `ReplayCanvas` calcule
`zones: calloutZones.length > 0` : sans zones servies, la bascule n'existe pas. **Rien a
corriger cote web** — le manque etait entierement backend + donnees.

L'extraction, elle, existait deja en production (`himap.ZonesNommeesForge`,
`mapvar.Object.LocationID`) : seule l'ALIMENTATION manquait.

### Ce qui a ete livre

| brique | fichier |
|---|---|
| section `maps_by_id` (cle = map_id) + `LookupByID` + provenance `mvar` | `internal/analysis/replay/callouts_catalog.go` |
| essai module PUIS map_id au service | `internal/service/replay_map_callouts.go` |
| index des libelles par `string_id` (seule cle possible pour Forge) | `cmd/mapcallouts-build/csv.go` |
| passe Forge (mvar -> zones -> libelles -> classement) | `cmd/mapcallouts-build/forge.go` |
| telechargement anonyme des `map.mvar` depuis l'inventaire versionne | `cmd/mapcallouts-build/forge_fetch.go`, `mapcatalog.Client.FetchFileAt` |
| refus des prismes retournes | `himap.prismeLisible` |

Commande : `CGO_ENABLED=1 go run ./cmd/mapcallouts-build --forge-only --forge-fetch`.
Elle N'EXIGE PAS le jeu installe (contrairement a la passe native) : sa source est
l'inventaire versionne `inventaire_rotation_ugc_2026-08-27.json` + le stockage blob UGC, qui
repond **sans jeton**. Le rejeu, lui, reste 100 % hors ligne a la lecture.

### La mesure (2026-09-02, rotation du 2026-08-27)

| grandeur | valeur |
|---|---|
| cartes Forge visees (inventaire) | 83 |
| variantes `map.mvar` telechargees | 82 nouvelles, **0 echec** |
| **cartes publiees au catalogue** | **61** |
| **zones publiees** | **2 392** |
| zones portant un libelle FR/EN | **869 (36 %)** |
| StringId de lieu distincts employes | 266 |
| StringId resolus par `callouts_i18n.csv` | **66 (25 %)** |
| cartes sans aucune zone nommee (donc non publiees) | 19 |
| cartes a zones mais aucune nommable (non publiees) | 3 |
| zones ecartees pour prisme retourne | 87 |

Les 19 cartes « sans zone » sont pour l'essentiel des cartes NATIVES republiees en asset UGC
(Aquarius, Catalyst, Cliffhanger, Deadlock, Fragmentation, Highpower, Live Fire, Recharge,
Streets, Oasis, Breaker...) : leurs callouts vivent dans le tag `levl`, elles sont deja
couvertes par la passe native, sous leur module. `Forbidden` est le RATELIER documente en
section 3 — le filtre a fait son travail.

### La regle tranchee sur les libelles manquants

**Une zone sans libelle est PUBLIEE, sans texte.** Ni omise, ni affublee d'un nom de repli.
- sa geometrie est MESUREE ; la cacher parce que son nom manque perd plus qu'elle ne protege ;
- un repli invente (« Zone 7 », le hash) afficherait un nom que le jeu ne prononce pas — la
  regle du chantier est « aucun nom devine » ;
- le rendu saute deja les libelles vides (`calloutsLayer.ts`, `drawLabels`) : rien a changer.

**Seuil unique** : une carte n'entre au catalogue que si AU MOINS UNE de ses zones porte un
libelle. Sous une bascule qui s'appelle « Zones nommees », un calque entierement muet serait
du bruit, pas une information. 3 cartes sont ecartees par cette regle.

### Ce qui reste vraiment

1. ~~**L'extraction `uslg` des 200 StringId sans texte**~~ **LIVRE le 2026-09-02** — section 8.
2. **Les cartes hors rotation du 2026-08-27** : l'inventaire est date. Une carte jouee depuis
   n'y est pas — il faudra le regenerer (balayage Discovery authentifie, une fois).
3. **Gate visuel** : aucune planche n'a ete faite. Le rendu d'une carte Forge n'a jamais ete
   regarde a l'ecran — maintenant que toutes ses zones portent un nom, c'est le vrai reste.

## 8. Le lexique des noms de lieu — LIVRE le 2026-09-02

### Le format, decode

Le blob de ressources d'un tag `uslg` n'est pas une soupe d'octets : c'est **18 sous-fichiers
`ucsh` CONCATENES**, un par langue, dans l'ordre du bloc des 18 du tag. Chaque sous-fichier est
un tag complet, et le suivant commence a `headerSize + dataSize`.

| element | ou |
|---|---|
| longueurs | `+0x38` headerSize, `+0x3C` dataSize |
| comptes | `+0x18` deps, `+0x1C` blocs, `+0x20` structs, `+0x24` dataRefs |
| table des blocs | `0x50 + deps*0x18`, 0x10/entree (taille, section, offset) |
| table des structs | `+ blocs*0x10`, 0x20/entree (type, cible, proprietaire, champ) |
| table des dataRefs | `+ structs*0x20`, 0x14/entree (bloc cible a `+0x08`) |
| **racine** | la struct SANS proprietaire (`owner == -1`) |
| **table d'index** | le TagBlock au champ 0 de la racine : N x `{ u32 string_id, u32 offset }` |
| **texte** | le bloc de la 1re dataRef : chaines UTF-8 terminees par NUL |

Deux pieges leves :

- **`0x150` n'etait pas une adresse, c'etait une coincidence.** La table d'index se trouve par
  la struct-table du sous-fichier ; sur le plus gros `uslg` elle tombe a `0x154`, ailleurs a
  `0x144`. Un offset en dur ratait 451 des 488 tags (ceux qui ont 3 blocs et non 4).
- **La langue ne se devine pas.** L'ordre des 18 slots n'est PAS croissant :
  `0..9, 11, 12, 15, 17, 10, 13, 14, 16`. Il se lit dans le tag. `0 = anglais`, `3 = francais`,
  verifies par la traduction d'une chaine connue sur les 18 slots.

### La preuve que le hash EST le string_id

Le lexique decode reproduit **les 463 string_id de `callouts_i18n.csv` avec un texte EN et FR
identique au caractere pres — 0 absent, 0 divergence** — et resout **777/777** entrees du
vocabulaire global `locs`. Ce n'est pas un echantillon : c'est la table figee entiere, validee
zone par zone en aout sur les 22 cartes integrees.

### Ce qui a ete livre

| brique | fichier |
|---|---|
| decodeur `uslg` + vocabulaire `locs` | `internal/himap/uslg.go` |
| lexique versionne `string_id;en;fr` (810 entrees) | `data/titles/halo_infinite/reference/callouts_lexique.csv` |
| ecriture / lecture / fusion du lexique | `cmd/mapcallouts-build/lexique.go` |
| bascule `--lexique` + jointure avant les passes | `cmd/mapcallouts-build/main.go` |
| temoins hors ligne + garde-rail contre le CSV fige | `cmd/mapcallouts-build/lexique_test.go` |
| anti-regression du decodeur sur les fichiers du jeu | `internal/himap/uslg_gamefiles_test.go` |

Commande : `CGO_ENABLED=1 go run ./cmd/mapcallouts-build --lexique --forge-only`.
`--lexique` EXIGE le jeu installe ; il ne se rejoue qu'a une mise a jour du jeu. La LECTURE du
lexique n'exige rien : c'est de la donnee versionnee, comme `map_structure`.

### La mesure — avant / apres

| grandeur | avant | apres |
|---|---|---|
| StringId de lieu resolus | 66 / 266 (25 %) | **266 / 266 (100 %)** |
| zones Forge portant un libelle | 869 / 2 392 (36 %) | **2 536 / 2 536 (100 %)** |
| cartes Forge publiees | 61 | **64** |
| cartes ecartees faute de zone nommable | 3 (Vallaheim x3) | **0** |

Les 3 cartes gagnees sont les Vallaheim (48 zones chacune) : leur vocabulaire etait
entierement hors du CSV natif. **Toutes les cartes passent a 100 %** — les plus fournies :
Kaiketsu 41 -> 105, Solution 41 -> 100, Interference 39 -> 84, Last Broadcast 34 -> 81,
Starboard 29 -> 95, Refuge 20 -> 83, Opulence 16 -> 64.

La section native est INCHANGEE (22 cartes, 816 zones, octet pour octet) : seule `maps_by_id`
bouge.

### Ce qui n'a PAS ete fait, et pourquoi

- **Les 16 autres langues** ne sont pas extraites. Le decodeur les lit toutes ; le lexique n'en
  ecrit que deux, parce que le depot est bilingue par typage (`Record<Locale, T>`). Ajouter une
  langue = ajouter une colonne, pas re-decoder.
- **Le seuil « au moins une zone nommee » est conserve** meme s'il ne mord plus : il
  redeviendrait la bonne reponse le jour ou le jeu introduirait un nom absent du lexique
  versionne.
- **Aucun nom n'est invente.** La regle de la section precedente tient telle quelle ; elle n'a
  simplement plus d'occasion de s'appliquer sur cette rotation.
