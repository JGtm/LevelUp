# Plan — porter la chaine des TRIANGLES en Go (fonds de carte de production)

> Ouvert le 2026-08-08 sur decision utilisateur : « la recette des triangles est la seule
> voie possible de rendu des maps en production, le faire en Go est une priorite ».
> Branche `feat/v75` (mode branche unique). S'execute sous le contrat `plan-execution`.
>
> Source de verite de la recette : `HANDOFF_GEOMETRIE_TRIANGLES.md` (§1 la chaine, §2 les
> trois corrections, §4 les portes fermees, §6 ou sont les choses). Ce plan ne re-decouvre
> rien : il PORTE, avec des garde-fous que le prototype Python n'avait pas.

## 0. Ce qui existe deja en Go, et qu'il ne faut pas refaire

| brique | etat |
|---|---|
| `internal/ooz` | decompression Kraken, cgo, hors ligne |
| `internal/himodule` | lecteur `.module` — `dataOffset` 48 bits et drapeau `UseHd1` PORTES |
| `internal/himap/instances.go` | bloc `instanced geometry instances`, `LocalToWorld` (convention vecteur-ligne) |
| `cmd/mapstruct-build` | production de l'asset AABB — **n'a PAS evolue**, cf. l'amendement du §6 : le fond de carte a son propre binaire, `cmd/mapfond-build` |
| `internal/analysis/replay/structure.go` | rendu de la surface cote document |

Ce qui manque, et c'est tout le sujet : `rtgo` (Per Mesh Data, sections, descripteurs de
tampons), l'appariement maillage -> tampon de sommets / tampon d'indices, la
dequantification, l'assemblage des triangles en monde.

## 1. Condition d'entree — le test qui manque (exigence du plan maitre)

`internal/himodule` **n'a aucun test** et le decoupage de `loadHd1` a ete verifie par
lecture, pas par execution. Le gate des artefacts ne le couvre pas : `replay-build` lit des
fichiers FIGES, il ne repasse pas par `himodule`.

- [x] **E1** Regenerer `ridgeline.json` et `sgh_streets.json` depuis les modules et les
      comparer aux versions figees du depot. Egalite attendue au centieme (le document est
      arrondi au centimetre). Un ecart = un bug de lecture a corriger AVANT de porter quoi
      que ce soit.
- [x] **E2** Figer cette comparaison en test (`-tags=gamefiles` comme les tests existants
      de `himap`, qui se declarent absents quand le jeu n'est pas installe).

**Gate E** : PASSE le 2026-08-08. E1 rend l'egalite EXACTE sur les deux cartes — 10 223 et
10 908 emprises, zero ecart, ecart max 0,000 m, tous les champs identiques hors
`generatedAt`. E2 est fige dans `cmd/mapstruct-build/equivalence_gamefiles_test.go` et sa
mutation a ete jouee : `insOffAABB` decale de 0x7C a 0x80 rend les deux cartes ROUGES.

**Decouverte traitee en passant, parce qu'elle bloquait le gate** : le chemin de
l'installation etait ecrit en dur a TROIS endroits, tous pointant vers `D:/SteamLibrary`,
un disque qui n'est plus monte. Les tests `gamefiles` de `himap` ne le disaient pas — ils
se declaraient « module absent » et passaient au VERT. Centralise dans
`himap.DeployRoot()` / `himap.LevelsDir()` (variable `LEVELUP_HALO_DEPLOY`, puis
emplacements connus), avec un garde-rail grep qui interdit le litteral ailleurs. Les deux
tests `gamefiles` TOURNENT desormais au lieu de se declarer absents.

## 2. Le portage, etape par etape

Chaque etape porte son temoin FALSIFIABLE. Les temoins tautologiques sont interdits : le
§2.3 du handoff en a deja retire un (« les bornes locales transformees reproduisent l'AABB
monde » est vrai par construction, ecart median 0,0000 m).

- [x] **T1 — lire `rtgo`.** Depuis `RuntimeGeoMeshReference` de l'instance : `PerMeshData`
      @16 (pas de 144), `Sections` @64, `BoundingBoxes` @104, `TotalVertexBufferCount` @190,
      `MeshResourceGroups` @196. *Temoin* : le bloc `PerMeshData` doit mesurer un multiple
      ENTIER de 144 (mesure sur nos tags : 864 = 6 x 144, 1 296 = 9 x 144, 720 = 5 x 144).
      **FAIT le 2026-08-08.** `himap.ModuleIndex` (index GlobalID -> module, carte puis
      globaux) + `himap.ReadRuntimeGeoTag`. Mesure sur ridgeline : 11 modules, 57 251
      entrees indexees, **9 832/10 357 instances resolues (94,9 %)**, **525 tags rtgo
      ouverts, 1 195 maillages**, pas de 144 respecte partout, `MeshIndex` dans la borne
      pour 100 % des instances.
      **PIEGE RENCONTRE, a ne pas perdre** : les offsets 8 ET 12 de la reference resolvent
      tous deux vers des `rtgo` valides — 98,6 % des instances y portent la MEME valeur.
      Ni le taux de resolution, ni la borne de `MeshIndex` (satisfaite a 100 % par les
      deux) ne les departagent : le premier temoin ecrit passait avec l'offset 12, donc il
      ne testait rien. Ce qui departage : a l'offset 8 la resolution est un SUR-ENSEMBLE
      STRICT (147 instances de plus, et jamais l'inverse). Le temoin exige desormais que
      `refOffGlobalID` soit l'ARGMAX STRICT sur tous les offsets de la reference — et la
      mutation 8 -> 12 le fait rougir.
- [x] **T2 — atteindre le maillage.** Champ racine `meshes`, TagBlock enfant a
      `foff = meshIndex * 60`, records de LOD render data de 148 octets, `u16` @0x64 =
      index du tampon de SOMMETS, `u16` @0x8A = index du tampon d'INDICES.
      *Temoin* : couverture des couples `(globalID, meshIndex)` resolus — le prototype rend
      **1 247/1 247**, on exige le meme 100 % ou on s'arrete.
      **FAIT** : `Sections` @64 EST le tableau des maillages (420 = 7 x 60, 360 = 6 x 60,
      120 = 2 x 60, le compte colle a `PerMeshData`). Les 525 tags livrent **1 195 blocs de
      LOD render data, tous multiples de 148**, un par maillage a `foff = meshIndex x 60`.
      **Zero maillage nul** sur les 9 832 instances rendues.
- [x] **T3 — les descripteurs de tampons.** Tables 0x50 (sommets) et 0x48 (indices) ;
      `off` est un **offset d'octets dans la CONCATENATION des entrees-ressource du tag**,
      ce n'est PAS un identifiant a resoudre contre le manifeste.
      *Temoin* : **0 descripteur hors bornes** sur les 29 683 du prototype.
      **FAIT** : les tables ne sont PAS designees par un chemin de champs — on les reconnait
      a leur invariant interne `size == count * stride` sur TOUS leurs enregistrements.
      Le blob de ressources manquait entierement a `himodule` : `ResourceBlob` le
      reconstitue (table de slots avant la table des blocs). **0 descripteur hors du blob**
      sur les 525 tags. Resultat : **46,6 M de triangles** assembles en repere monde.
- [x] **T4 — dequantifier.** Sommets `u16` x4, 4e composante nulle. **`u16` BRUT, jamais
      `i16 + 32768`** (§2.1 : 5,8 mm d'ecart aux bornes contre 84 mm avec la mauvaise
      lecture — tout resultat geometrique anterieur au 2026-07-26 est entache de 8,4 cm).
      *Temoin* : ecart median aux bornes < 1 cm.
      **Point ouvert a trancher ici, pas plus tard** (§3.2 du handoff) : Reclaimer place les
      bornes dans un `BoundingBoxes` UNIQUE par tag (84 octets, trois paires symetriques),
      designe par `BoundsIndex` (+118 de l'instance) ; le prototype lit des bornes PAR
      MAILLAGE. Les deux produisent des chiffres, une seule est juste — les departager par
      T4 sur les deux lectures.
      **NON TRANCHE, et c'est le resultat.** Les bornes sont PAR MAILLAGE (0x44/0x50 du
      record de 144) — ce point-la est acquis. Mais AUCUNE statistique de maillage ne
      departage le `u16` brut du `i16 + 32768`, trois essais a l'appui sur les MEMES
      octets : ecart aux bornes 16,9 mm contre 2,1 mm (donne raison a la FAUSSE, metrique
      biaisee — une rotation disperse les valeurs vers les extremes donc epouse mieux les
      bornes) ; longueur mediane d'arete 0,0189 contre 0,0196 ; part d'aretes longues
      0,0821 contre 0,0897. **L'echec est instructif** : la rotation ne DECHIRE pas les
      maillages, elle les DECALE chacun d'une demi-boite — la forme de chaque maillage
      reste intacte, c'est leur REGISTRE MUTUEL qui casse, invisible a toute statistique
      interne. Le juge est donc un oracle EXTERNE : les positions de joueurs du film
      (temoin de T6). Le `u16` brut est retenu en attendant, sur la foi du handoff §2.1
      et du rendu compare, PAS d'une mesure de ce chantier.
      **TRANCHE LE 2026-08-08 par l'oracle externe, et le statut passe a `[x]`** : les
      29 221 positions de joueur de `000d5950.json` donnent **63,6 % des positions a moins
      de 25 cm du sol pour le `u16` BRUT contre 34,3 % pour `i16 + 32768`**, a reglage egal
      (`carte_oracle_gamefiles_test.go`, handoff §1 bis). L'oracle a tranche du meme coup le
      filtre de module. La statistique INTERNE ne pouvait pas separer — c'est le temoin
      venu de DEHORS qui l'a fait.
- [!] **T5 — appariement sommets/indices.** *Temoin* **T1 du handoff** (le seul non
      tautologique) : l'indice maximal du tampon d'INDICES doit etre strictement inferieur
      au nombre de sommets du tampon apparie. Rend 100 % pour le bon appariement et **5,1 %**
      pour un LOD voisin — c'est ce contraste qui fait le test.
      **NON TRAITE, et il faut le dire (statue le 2026-08-10, lot A).** L'INVARIANT est bien
      applique en production — `RuntimeGeoAsset.triangles` ECARTE toute face dont un indice
      depasse le tampon de sommets, avec la bonne raison ecrite (« la rendre inventerait de la
      geometrie »). Ce qui manque, c'est la MESURE qui departage : personne n'a jamais compare
      le taux du bon appariement a celui d'un LOD voisin. La chaine est donc sure par
      construction, mais on ignore combien de faces elle jette. Hors perimetre du lot A (qui
      produit les assets) ; porte au registre des reports avec sa condition de reprise.
- [~] **T6 — assembler en monde** par `LocalToWorld` de l'instance (deja en Go), et
      produire le champ d'altitude / la surface.
      *Temoin de non-regression, a refaire PROPREMENT* : le handoff donne 82,0 % de
      positions de joueur a moins de **25 cm** du sol, quand le fond en boites etait valide
      a 80,6 % sous **5 cm**. **Ces deux nombres ne se comparent pas** (seuil cinq fois plus
      large). Remesurer les deux au MEME seuil avant toute conclusion.
      **MECANISME FAIT, CHOIX DE L'ETAGE NON RESOLU (2026-08-08).** `himap.HeightField`
      rasterise par TRIANGLE (pas par sommet : un sol est fait de grandes faces peu denses
      en sommets, les compter par sommet le laisse troue) et ne garde que les faces dont la
      normale s'ecarte de moins de 45 degres de la verticale. Trois tests unitaires purs,
      sans fichiers de jeu : filtre des murs, rasterisation par surface, choix de l'etage
      par le plafond — chacun avec sa mutation.
      **CE QUI RESTE, et c'est le vrai sujet** : « la surface marchable la plus haute » est
      le SOMMET DES FALAISES sur une carte encaissee. Mesure sur Cliffhanger : le marchable
      s'etale de -107 a +60 m **sans bande dominante** (aucune tranche de 2 m ne depasse
      2,7 % des cellules), donc aucun plafond fixe n'isole l'arene — essais a 2, 8 et 20 m,
      tous illisibles. Il ne s'agit pas de regler un seuil : il faut savoir OU LES JOUEURS
      MARCHENT. C'est l'oracle des positions du film, qui tranchera aussi T4.
      **CORRECTION UTILISATEUR DU 2026-08-08, et elle a debloque l'etape.** « Tu ne te
      compliques pas la vie ? Le choix de l'etage est venu APRES le dessin de la carte, on
      est en 2D, qu'il y ait plusieurs niveaux nous importe peu tant qu'on sait qu'ils
      existent. » C'etait juste. Le defaut n'etait ni l'etage ni la geometrie : c'etait le
      RENDU. Deux corrections, et la carte apparait :
      (1) echelle de couleur ROBUSTE, centiles 2 et 98 au lieu de min/max — une seule
      cellule a -131 m ecrasait toute la carte dans deux nuances de blanc ;
      (2) OMBRAGE DE RELIEF depuis la pente. Un degrade d'altitude seul est plat a l'oeil ;
      c'est l'ombre qui fait apparaitre les aretes, les plateformes et le bati.
      Resultat sur Cliffhanger : la roche se distingue des plateformes construites, le
      batiment a arcades, la plateforme circulaire ajouree et les passerelles se lisent.
      **Aucun plafond, aucun choix d'etage, aucun reglage sur le resultat.**
      **Lecon** : avant de soupconner la donnee, verifier son AFFICHAGE. J'ai passe trois
      essais de plafond a chercher un probleme de geometrie qui etait un probleme de
      dessin.

**Gate T** : les six temoins verts sur ridgeline, ET l'image produite comparee cote a cote
avec `carte_validee_v1.png`.

## 3. Cuire les cartes

**FAIT LE 2026-08-10 (lot A).** La chaine est sortie des tests (`internal/himap/cuisson.go`,
`cuisson_forge.go`, `fond_png.go`), la sortie passe par `PathResolver.MapBackgroundPath`, et
`cmd/mapfond-build` cuit les cartes. Rapport chiffre :
`.ai/V7.5/cartes/RAPPORT_CUISSON_FONDS_2026-08-10.md`.

- [x] **C1** Cliffhanger (ridgeline) — **5 102 instances dessinees, 859 ecartees comme decor,
      14/14 ancres avec sol, ecart median -0,32 m, 1 633 x 1 627 px, 681 529 octets.** Le banc
      ne bouge pas : accord 66,7 %, positions jouees 93,82 %, eau 233 volumes -> 5 467 cellules,
      terrain compare a l'octet.
- [x] **C2** Catalyst — **7 796 instances, 162 ecartees, 24/24 ancres, ecart median -0,29 m,
      1 406 x 1 553 px.** Le meilleur ecart median des 17, avec Cliffhanger.
- [x] **C3** Les autres cartes cuites dans un module : **16 cartes natives publiees au total**,
      plus Vagabond (Forge). Les 25 « cartes » du balayage sont des ENTREES DE CATALOGUE, pas
      des cartes : `aquarius_map` et `aquarius_-_ranked_map` designent le meme dossier
      d'installation. Regroupees par module installe, avec l'UNION dedupliquee de leurs ancres,
      elles font 16 cartes distinctes. **278/344 ancres ont du sol dessine, 4,92 Mo au total.**
      Une carte NON CUISINABLE, connue et instruite : `sgh_interlock` (live_fire) ne porte aucun
      tag sbsp (§1 ter) — signalee comme telle, elle ne fait pas echouer la cuisson.
      **DEUX CARTES SONT AMPUTEES et il faut le dire** : `chasm` (5/17 ancres) et
      `btb_highpower` (14/51). Cause MESUREE, cf. D3 au §7 — leur oracle est publie dans leur
      sidecar, elles ne passent pas pour completes.
- [x] **C4** GATE VISUEL — **TENU le 2026-08-10.** Quatre cartes tirees AU HASARD a la demande
      de l'utilisateur (hors Catalyst / Cliffhanger / Vagabond), puis verdict de sa part sur
      l'ensemble des 17. Le style `jeu` est valide de fait (aucune remarque de couleur ; les
      cartes lisibles sont jugees « nickel »).

      | verdict utilisateur | cartes |
      |---|---|
      | nickel | Launch Site, Breaker, Behemoth, Fragmentation, Catalyst, Cliffhanger, Forest |
      | « un peu rudimentaire » | Streets, Bazaar (cause non identifiee) |
      | « on ne voit que les toits », forme globale correcte | Illusion, Prism, Aquarius |
      | idem mais partiellement legitime (une partie a ciel ouvert) | Forbidden |
      | rien d'exploitable | **Chasm, Highpower** -> CORRIGE, cf. D3 |
      | non reconnues (jamais jouees / autre nom) | Recharge, Vagabond -> les assets portent desormais leur nom affiche |

      **DEUX DEFAUTS OUVERTS PAR CE GATE, au registre** : les TOITS des cartes couvertes (le
      z-buffer garde la surface la plus haute), et le « rudimentaire » de Streets/Bazaar. Ni
      l'un ni l'autre n'etait visible dans un chiffre — c'est l'oeil qui les a trouves.

## 4. Etape 2 — les cartes NON cuites (Vagabond)

La difference n'est pas « Forge ou pas » : Cliffhanger aussi a ete concue dans Forge, mais
343 l'a **cuite** dans un module dedie. Mesure du 2026-08-08 :

| module | instances de geometrie | objets Forge du `.mvar` |
|---|---:|---:|
| ridgeline (Cliffhanger) | 10 223 | 443 |
| catalyst | 11 178 | 357 |
| fo08_wetland (Vagabond) | 788 | 4 709 |

- [x] **F1** Resoudre `type_id -> tag de modele` — FAIT au lot 2 (2026-08-10) : le lien est
      INLINE dans les octets du tag `food`, 374 type_id portent au moins une ref `rtgo`.
- [x] **F2** Poser les triangles du modele par la transformation de chaque objet
      (`Pos`/`Up`/`Forward` + echelle) — FAIT, et en PRODUCTION depuis le lot A
      (`internal/himap/cuisson_forge.go`, publie sous la cle `fo08_wetland`). Mesure de la
      cuisson : **3 558 objets dessines sur 4 709 (75,6 %), 1 113 sans modele `rtgo` direct,
      38 volumes de mort exclus, 4/4 ancres avec sol, 1 332 x 1 287 px.**
      L'echelle : AUCUNE dans le `.mvar` — mesuree absente, pas supposee.
      **NON FAIT, et c'est le lot B** : les 1 113 objets sans modele direct passent par
      `bloc`/`scen`/`mach` -> `hlmt`. La toile du canevas `fo08_wetland` n'est toujours pas
      rendue SOUS les objets (registre).
- [!] **F3** Gate visuel Vagabond — meme statut que C4 : l'asset est produit, le juge est
      l'utilisateur.

## 5. Portes fermees — ne pas les rouvrir (§4 du handoff)

| piste | pourquoi |
|---|---|
| compagnon `.module_hd1` | preuve d'ABSENCE : 59 tags `bitm`, les 1 784 ressources `rtgo` sont toutes dans le module principal |
| « int16 x4 a 4e composante nulle » comme detecteur de sommets | tautologique : un bloc DXT1 plat rend `w == 0`, une TEXTURE atteint 0,915 |
| saturation de la plage comme critere de maillage normalise | tautologique : 0,888 pour les vrais tampons contre 0,890 au hasard |
| collision `scgt` lue en float32 brut | faux positif : 95 a 97 % des points a moins d'un metre de l'origine — une croix |
| les boites, alignees ou orientees | 0,00 m² de vide dans le fer a cheval dans les DEUX cas : l'anneau vit dans les triangles |

**Regle de methode qui vient de ces echecs : toujours DESSINER un resultat, jamais seulement
le compter.**

## 6. Contrat d'execution

**Critere de succes** : l'image de ridgeline produite par le code Go se lit comme
`carte_validee_v1.png` (architecture reelle, roche distincte des plateformes, aucun
rectangle) ET les six temoins du §2 sont verts. Rien d'autre ne vaut « fait ».

**Effort** : lourd, plusieurs seances. Les etapes E et T sont sequentielles et
indissociables ; C et F sont incrementales carte par carte.

**Ou vit le code** : production d'asset HORS LIGNE, donc aucune couche service / handler /
adapter n'est concernee — c'est deliberé, pas un oubli. Le decodage va dans
`internal/himap` (a cote de `instances.go`), la sortie passe par `PathResolver`, jamais par un
`filepath.Join` sur `data/`. `internal/ooz` etant en GPLv3, **rien de cette chaine ne doit
etre linke par le serveur** : l'app lit l'asset fige, elle ne le fabrique pas.

**AMENDE LE 2026-08-10 (lot A) sur un point : l'orchestration ne reste PAS dans
`cmd/mapstruct-build`.** Ce plan l'annoncait ; a l'execution les deux chaines n'ont ni la meme
entree, ni la meme sortie, ni le meme consommateur :

    mapstruct-build   map_quant_bounds.json -> map_structure/{module}.json   (AABB, aucune forme)
    mapfond-build     map_objectives.json   -> map_backgrounds/{module}.png  (triangles, image)

Le fond de carte a besoin des ANCRES d'objectifs — pour cadrer, pour deduire le niveau joue et
pour se juger — que le catalogue de bornes ne porte pas ; et `map_structure/` a des lecteurs
VIVANTS (`cmd/replay-build`) qu'on ne casse pas. Fondre les deux orchestrations melangerait deux
chaines sans rien mutualiser. D'ou `cmd/mapfond-build`, et
`PathResolver.MapBackgroundDir/Path/MetaPath` a cote de `MapStructurePath`.

**Le type du sidecar de calage vit du cote CONSOMMATEUR** (`internal/analysis/replay/
map_background.go`) et non dans `himap` : lire un calage ne doit jamais faire linker
`himap -> himodule -> ooz`. Garde-rail : `cmd/server/licence_gplv3_test.go` parcourt le graphe
des imports depuis `cmd/server` et refuse tout paquet interdit — mutation jouee.

**Regles d'execution** (le skill `plan-execution` fait foi, resume ici) :

1. Ordre strict : une etape N n'est close que gate passe ET tous ses items statues.
2. Statuts : `[x]` fait et verifie · `[~]` couvert ailleurs (dire ou) · `[!]` non traite
   (justification ecrite). **Aucune case vide a la cloture.**
3. Zero fix hors perimetre : toute decouverte va au §7 « Decouvertes », elle n'est pas
   traitee — sauf si elle bloque le gate courant.
4. Aucune decision produit n'est laissee a l'execution. La seule qui restait — bornes par
   maillage contre `BoundingBoxes` unique par tag — est **tranchee par la mesure T4**, pas
   par un arbitrage : celle des deux lectures qui rend un ecart median < 1 cm gagne. Si les
   deux passent ou aucune, s'arreter et remonter.
5. Reprise de session : ce fichier est la source de verite de l'avancement. Relire le §7
   puis reprendre a la premiere case non statuee.
6. **La doc suit la cadence des etapes, pas celle du chantier** (regle utilisateur du
   2026-08-08). A CHAQUE etape close : la case est statuee ici avec le CHIFFRE mesure, la
   decouverte eventuelle va au §7, et l'entree `thought_log` est ecrite — le tout dans le
   commit de l'etape, jamais reporte a la fin. Une doc redigee d'un bloc a la cloture est
   deja une doc reconstruite de memoire.

## 7. Decouvertes (en cours d'execution)

**D1 — « une carte = un module » est FAUX pour la geometrie de rendu, pas seulement pour la
collision** (mesure du 2026-08-08, ridgeline, 10 357 instances).

Le champ `RuntimeGeoMeshReference` (28 octets a +0x3C) est structure ainsi : les offsets
0, 4, 20 et 24 sont CONSTANTS sur toutes les instances (une seule valeur distincte
chacun) ; les offsets 8, 12 et 16 portent l'identite, avec 548 valeurs distinctes chacun.
**L'offset 8 est le `GlobalID` du tag `rtgo`** — 525 des 548 references (95,8 %) y
resolvent contre les identifiants globaux des modules, ce qui n'arrive pas par hasard.

Mais elles resolvent dans QUATRE modules differents :

| module | references resolues | instances |
|---|---:|---:|
| `levels/multi/ridgeline` | 324 | 2 730 |
| `globals/common` | 70 | 4 198 |
| `globals/multiplayer` | 102 | 2 519 |
| `globals/multiplayer_r3` | 29 | 385 |
| **non resolu** apres balayage des **44 modules de `pc/`** | **23** | **525 (5,1 %)** |

Le module de la carte ne couvre donc que **26 % des instances**. Le §5 du handoff ne
signalait ce piege que pour la COLLISION (195 des 552 modeles dans des modules globaux) ;
il vaut aussi pour le rendu, et plus fortement.

**Consequence sur le plan** : T1 doit resoudre les references contre un ENSEMBLE de modules,
pas contre celui de la carte. `himodule` ouvre un fichier ; il faut un index
`GlobalID -> (module, entree)` construit sur le module de la carte plus les modules globaux.
A traiter DANS T1, pas apres — sans lui, 74 % des instances n'ont pas de maillage.

**Reste ouvert** : les 23 references (5,1 % des instances) que les 44 modules de `pc/` ne
portent pas. Pistes non explorees : les variantes `ds/` et `any/`, ou un contenu non
installe localement. A quantifier au moment du rendu — 5 % d'instances manquantes se voient
sur une image, ou ne se voient pas.

**D2 — l'echelle d'instance n'etait pas appliquee, et Reclaimer n'avait jamais ete lu**
(2026-08-09, seconde session ; detail chiffre au §10 du handoff `HANDOFF_PORT_TRIANGLES_2026-08-08.md`).

La lecture croisee de `ScenarioStructureBspTag.cs` / `RuntimeGeoTag.cs` et du plugin
`sbsp.xml` — qui concordent a l'octet sur les 320 d'une instance — a rendu trois resultats :

1. **`TransformScale` @0x00 doit etre applique.** 12 009 des 14 328 instances portent une
   echelle differente de 1 (de -38,86 a +116,33). Temoin qui separe : ecart a la boite monde
   declaree de l'instance, source independante du maillage, median **0,2238 sans l'echelle
   contre 0,0665 avec**. `LocalToWorld` l'applique desormais.
2. **`mesh flags override` @0x110 n'etait pas lu.** 4 343 instances portent
   `mesh is custom shadow caster`, que Reclaimer ecarte. Les retirer coute 47 % des instances
   rendues pour 0,1 point de couverture : c'etaient bien des proxys redondants.
3. **La geometrie NON INSTANCIEE du sbsp n'existe pas** sur ces cartes : blocs `meshes`,
   `compression info`, `mesh resource groups` et `raw_resources` tous a **count=0**. Sixieme
   porte fermee, par la mesure.

**Consequence sur le plan** : tout resultat geometrique produit avant le 2026-08-09 l'a ete
avec 84 % des instances a la mauvaise taille. Les mesures du handoff §9.2 (les cinq portes)
ont ete rendues sur cette geometrie — elles restent valides comme constats de l'epoque, elles
ne valent plus comme refutations definitives.

**Reste non porte** : `MeshResourcePackingPolicy` @186 du `rtgo` (au registre des reports).

**D3 — la tranche de jeu est ABSOLUE, et c'est un accident propre a Cliffhanger**
(mesure du 2026-08-10, lot A ; c'est la cuisson qui l'a revele).

`TrancheDeJeuMin/Max` = `[-12 ; +28]` sont des altitudes ABSOLUES, mesurees sur Cliffhanger dont
le sol joue est a -2,2 m. Le niveau joue des 17 cartes cuites s'etale de **-136,7 m (`chasm`) a
+52,3 m (Vagabond)**. La tranche decapite donc les cartes qui ne jouent pas vers zero — et
personne ne pouvait le voir avant, parce que le balayage compare les ancres AVANT et APRES la
coquille, jamais leur total.

Ce que la cuisson publie, carte par carte (`anchorsWithGround/anchorsInFrame` du sidecar) :

| carte | niveau joue | ancres avec sol |
|---|---:|---|
| `chasm` | -136,7 m | **5/17** |
| `btb_highpower` | +44,5 m | **14/51** |
| `btb_fragmentation` | +6,2 m | 36/46 |
| `forest` | +1,9 m | 14/18 |
| `catalyst` | +24,4 m | 24/24 (a 3,6 m du plafond) |

**La lecture concurrente a ete MESUREE, pas supposee**, en translatant la tranche au sol des
ancres — ce que la chaine Forge fait deja :

| carte | tranche absolue | tranche relative |
|---|---|---|
| `chasm` | 5/17 | **17/17** |
| `btb_highpower` | 14/51 | **38/51** |
| `catalyst` (temoin) | 24/24 | 24/24 |
| `ridgeline` (temoin) | 14/14 | 14/14 |
| **`ridgeline`, ORACLE FORT** | **accord 66,7 %** · positions 93,82 % | accord **64,7 %** · positions 93,95 % |

La relative repare deux cartes et en degrade UNE : celle qui possede la seule reference validee.
L'exces passe de 33,8 a 39,1 % — c'est la VALLEE sous Cliffhanger qui entre dans le cadre. Le
banc passe au rouge, et le plan dit que toute modification qui le fait bouger est un echec.

**TRANCHE PAR L'UTILISATEUR LE 2026-08-10, SUR PIECES** : images des quatre cartes tirees au
sort soumises au gate, verdict « Chasm n'affiche rien d'exploitable et highpower pareil ». La
tranche est desormais RELATIVE (`himap.TrancheDeJeu`), et le banc est RE-BASE par ecrit —
accord 64,7 %, positions 93,95 %. Ce n'est pas une regression toleree : c'est un arbitrage entre
deux cartes inexploitables et deux points d'accord sur une carte deja lisible.

**Une seule expression pour la translation**, partagee par la chaine native, la chaine Forge et
le banc. Elle etait ecrite a six endroits avant ce lot ; six copies d'une regle finissent par
diverger, et le banc aurait cesse de garder ce qui est reellement cuit.

**D4 — trois cartes etaient declarees « non installees » alors que leur dossier etait la**
(2026-08-10, lot A ; souleve par l'utilisateur : « y en a bien plus dans le jeu non ? »).

L'installation porte **31 dossiers de carte**, dont 2 tutoriels, 1 PvE et 8 canevas Forge —
soit **20 cartes multijoueur natives**. La cuisson n'en rendait que 16 :

| dossier | pourquoi absent |
|---|---|
| `sgh_interlock` (Live Fire) | aucun tag sbsp — exception instruite §1 ter |
| `btb_drydock`, `btb_engine`, `btb_exiled` | **l'appariement par jetons de nom echouait EN SILENCE** |

Le catalogue les nomme `deadlock_map`, `scarr_map`, `oasis_map` ; aucun de ces jetons ne colle a
`drydock`, `engine`, `exiled`. La commande les rangeait sous « non installees localement » —
une explication FAUSSE fabriquee par un heuristique qui echoue sans le dire.

**UNE PISTE REFUTEE AVANT LA BONNE, et elle merite d'etre gardee.** Remplacer les noms par une
MESURE — « le module dont un bsp contient les ancres » — semblait evident : `ChoisitBSP` fait
deja exactement ca un cran plus bas. Son temoin l'a tuee deux fois. D'abord « ne tranche pas »
partout, avec des scores PLEINS : chaque carte declare un bsp d'HORIZON de plusieurs kilometres
qui contient les ancres de n'importe qui. Puis, departage par le plus petit volume, elle
contredisait le nom sur ~20 cartes connues (Catalyst -> `ctf_breaker`, Oasis -> `ridgeline`).
**Chaque carte Halo est batie autour de SA propre origine : une position monde ne designe aucune
carte.** Code supprime, entree au registre.

**LA REGLE QUI TIENT EST UNE DONNEE.** Le depot de variantes porte, a cote de `deadlock_map.mvar`,
un `deadlock_btb_drydock.mvar` : le lien est ecrit noir sur blanc. Convention **validee sur les
18 appariements deja connus, 0 manquant** — calibrer sur les cas connus puis appliquer aux
inconnus, la methode du chantier.

Ce qu'elle rend, avec l'oracle de chaque carte :

| carte | dossier | ancres | niveau joue |
|---|---|---|---|
| Deadlock | `btb_drydock` | **46/46** | +77,3 m |
| Oasis | `btb_exiled` | 30/35 | +24,9 m |
| Scarr | `btb_engine` | **19/19** | +0,4 m |
| Corpo | `fo11_blank` | **4/4** | chaine FORGE |

**Deadlock joue a +77,3 m** : avec la tranche absolue elle aurait ete aussi vide que Chasm. Les
deux corrections de ce lot se tenaient l'une l'autre.

**ET CORPO A APPRIS COMMENT ON RECONNAIT UNE CARTE FORGE.** Son module `fo11_blank` se resout
tres bien — mais la chaine native y a echoue avec « aucune instance dessinee sur 0 du bsp
retenu ». Un canevas VIERGE est vide par construction : la carte EST son rack d'objets. Le
critere n'est donc pas un prefixe de nom (`fo*`), c'est l'absence de geometrie, et c'est la
chaine elle-meme qui le dit. Corpo passe desormais par `CartesForge` : 1 725 objets sur 1 988,
4/4 ancres.

## 8. Journal

- **2026-08-10 (lot A — LES ASSETS)** — etape 3 close, `C1`/`C2`/`C3` faites, `C4` en attente de
  l'utilisateur. La chaine est sortie des tests et produit **17 assets** (16 cartes natives +
  Vagabond), 4,92 Mo, style `jeu`, 0,0920 m/px, fond transparent, calage publie dans un sidecar
  par carte. Trois garde-rails poses, chacun avec sa MUTATION JOUEE : licence GPLv3
  (`cmd/server` ne linke pas `ooz` — 108 paquets parcourus, la mutation rougit), calage publie
  identique a l'image produite (l'inversion du bord Y rougit avec 3,68 m d'ecart), echelle de
  production egale a celle du banc. **Determinisme verifie** : deux cuissons independantes
  rendent des PNG identiques au bit.
  Le banc ne bouge pas — accord 66,7 %, positions 93,82 % — alors meme que la boucle de rendu du
  banc est desormais CELLE de la production (`PeupleRendu`), et non plus sa jumelle.
  **Ce que la cuisson a appris et que les tests ne pouvaient pas dire** : voir D3. L'oracle des
  ancres, publie carte par carte, a revele que la tranche absolue ampute `chasm` et
  `btb_highpower`. Un asset produit dit des choses qu'un test qui n'ecrit rien ne dira jamais.
  **Trouve en voulant passer le gate : `TestBalayageCoquille` n'assertait RIEN** — il
  journalisait « 0 perdent des ancres » et rendait `ok` quoi qu'il arrive. Le gate du plan
  reposait sur un humain lisant une sortie `-v`. Assertion posee.
- **2026-08-09** — dette de lecture Reclaimer soldee (§9.4 point 1 du handoff). Voir D2
  ci-dessus. Le gate visuel du point 3 reste du : trois artefacts A/B/C produits, chaque
  correction isolable, juge = l'utilisateur.
- **2026-08-08** — plan ouvert. Contexte : le lot 5 a livre les zones de Catalyst et
  Vagabond mais a montre des fonds en boites englobantes ; l'utilisateur a rappele que la
  belle carte de Cliffhanger vient des triangles et que ce chemin est le seul valable en
  production. Deux erreurs du lot 5 corrigees au passage : la couverture comparee entre
  cartes (denominateurs incomparables) et « Vagabond restera a sa carte d'altitude »
  (l'etape 2 existe, cf. §4).
