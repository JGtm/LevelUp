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
| `cmd/mapstruct-build` | production de l'asset fige — **c'est lui qui doit evoluer** (§6 du handoff) |
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
- [!] **T4 — dequantifier.** Sommets `u16` x4, 4e composante nulle. **`u16` BRUT, jamais
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
- [ ] **T5 — appariement sommets/indices.** *Temoin* **T1 du handoff** (le seul non
      tautologique) : l'indice maximal du tampon d'INDICES doit etre strictement inferieur
      au nombre de sommets du tampon apparie. Rend 100 % pour le bon appariement et **5,1 %**
      pour un LOD voisin — c'est ce contraste qui fait le test.
- [ ] **T6 — assembler en monde** par `LocalToWorld` de l'instance (deja en Go), et
      produire le champ d'altitude / la surface.
      *Temoin de non-regression, a refaire PROPREMENT* : le handoff donne 82,0 % de
      positions de joueur a moins de **25 cm** du sol, quand le fond en boites etait valide
      a 80,6 % sous **5 cm**. **Ces deux nombres ne se comparent pas** (seuil cinq fois plus
      large). Remesurer les deux au MEME seuil avant toute conclusion.

**Gate T** : les six temoins verts sur ridgeline, ET l'image produite comparee cote a cote
avec `carte_validee_v1.png`.

## 3. Cuire les cartes

- [ ] **C1** Cliffhanger (ridgeline) — c'est la carte de reference, la seule dont l'image
      soit validee. Rien ne se publie tant qu'elle ne se reproduit pas.
- [ ] **C2** Catalyst. Sa structure instanciee est aussi complete que celle de Cliffhanger
      (79,0 % contre 83,1 % a perimetre egal, remesure du 2026-08-08), l'etape 1 doit donc
      la rendre.
- [ ] **C3** Les autres cartes cuites dans un module, une par une, avec leur couverture.
- [ ] **C4** GATE VISUEL par carte : artefact de revue, temoins donnes par l'UTILISATEUR
      (jamais par la session), validation ecrite. Voir le §6 du rapport du lot 5.

## 4. Etape 2 — les cartes NON cuites (Vagabond)

La difference n'est pas « Forge ou pas » : Cliffhanger aussi a ete concue dans Forge, mais
343 l'a **cuite** dans un module dedie. Mesure du 2026-08-08 :

| module | instances de geometrie | objets Forge du `.mvar` |
|---|---:|---:|
| ridgeline (Cliffhanger) | 10 223 | 443 |
| catalyst | 11 178 | 357 |
| fo08_wetland (Vagabond) | 788 | 4 709 |

- [ ] **F1** Resoudre `type_id -> tag de modele` (question Q1 de la piste palette Forge,
      non couverte a ce jour).
- [ ] **F2** Poser les triangles du modele par la transformation de chaque objet
      (`Pos`/`Up`/`Forward` + echelle), au-dessus de la toile rendue par l'etape 1.
- [ ] **F3** Gate visuel Vagabond.

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
`internal/himap` (a cote de `instances.go`), l'orchestration reste dans
`cmd/mapstruct-build`, la sortie passe par `PathResolver.MapStructurePath`, jamais par un
`filepath.Join` sur `data/`. `internal/ooz` etant en GPLv3, **rien de cette chaine ne doit
etre linke par le serveur** : l'app lit l'asset fige, elle ne le fabrique pas.

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

## 8. Journal

- **2026-08-08** — plan ouvert. Contexte : le lot 5 a livre les zones de Catalyst et
  Vagabond mais a montre des fonds en boites englobantes ; l'utilisateur a rappele que la
  belle carte de Cliffhanger vient des triangles et que ce chemin est le seul valable en
  production. Deux erreurs du lot 5 corrigees au passage : la couverture comparee entre
  cartes (denominateurs incomparables) et « Vagabond restera a sa carte d'altitude »
  (l'etape 2 existe, cf. §4).
