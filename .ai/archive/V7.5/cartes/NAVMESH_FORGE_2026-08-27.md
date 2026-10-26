# Le maillage de navigation des cartes Forge — format, decodage, usage

> Ecrit le 2026-08-27, le jour ou il a rendu Isolation lisible apres deux jours de
> soustractions infructueuses. Code : `apps/go-api/internal/hinavmesh` (decodage),
> `internal/himap/cuisson_navmesh.go` et `reference_navmesh.go` (usage).

## 1. Pourquoi il existe pour nous

Une carte Forge « organique » est un EMPILEMENT DE COQUES au-dessus de son arene. Sur Isolation :
un type d'objet a 32 exemplaires, pose entre Z 136 et 160 quand le sol joue est a Z 117, peint
**82,7 % de l'image** ; le retirer decouvre une deuxieme couche (3 pieces de 65 m, 46 %), puis
une troisieme (117 pieces). Le pelage lui-meme s'arrete : **des le premier retrait il coute une
ancre d'objectif**, c'est-a-dire du sol.

Toutes les soustractions ont ete essayees et mesurees inoperantes : ecretage des toits a 4, 2 et
1 m ; tranche de rendu plafonnee a +3, +6 et +12 m ; bornage a l'emprise des volumes de mort ;
substitution par surface de reference ; six habillages ; trois echelles de pixel ; exclusion des
objets par emprise du modele, par aire de maillage, par couverture au sol, par drapeau, par
altitude de pose.

**La sortie n'etait pas de mieux soustraire mais de changer de source.**

## 2. Ou le trouver

Chaque carte Forge publie son `navmesh.blob` a cote de sa variante, dans l'asset UGC :

```
https://www.halowaypoint.com/halo-infinite/ugc/maps/{assetId}
  -> bloc <script id="__NEXT_DATA__">
     -> Files.Prefix + "navmesh.blob"
```

**Le blob se telecharge SANS AUCUN JETON.** Deux requetes, aucune authentification.

Depot local hors ligne : `.ai/re_dump/navmesh/{map_id}.blob` (constante `himap.DepotNavmesh`).

**COUVERTURE, MESUREE** : le navmesh n'existe QUE pour les cartes Forge, et seulement au-dela
d'environ 1 000 objets — present sur 10 cartes testees sur 10 dans cette bande, absent (HTTP 404,
`BlobNotFound`) sur 13 sur 13 en dessous. Sur les 101 cartes du referentiel, **66 sont dans la
bande favorable**. Les cartes natives n'en ont pas — et n'ont jamais eu ce probleme.

## 3. Le format, couche par couche

| couche | contenu |
|---|---|
| octets 0-11 | en-tete GROS-BOUTISTE : `[0..3]` version = 2 ; `[4..7]` = taille du fichier moins 12 ; `[8..11]` = `0x001FFFFF`, **constante** (identique sur un fichier quatre fois plus petit) |
| a partir de 12 | **le MEME CONTENEUR QUE LES `.mvar`** — Bond CompactBinary v2, decode par `internal/analysis/replay/mapvar/cb2.go` **sans une ligne nouvelle** |
| champ 0 | i32 = 1 |
| champ 1 | liste de 363 827 int8 : **la charge utile compressee** |
| champ 2 | u32 = taille decompressee (1 128 372 pour Isolation) |

**LA COMPRESSION, ET LE PIEGE QU'ELLE POSE.** C'est du **zlib ordinaire**, mais son en-tete est
`58 09` et non `78 9c` : CINFO=5, soit une fenetre de **8 Ko** au lieu de 32. C'est pourquoi les
recherches de signature usuelle l'avaient manque. Entropie du blob 7,92 bits/octet contre 6,60
pour le `.mvar` de la meme carte.

**La charge decompressee** commence par un preambule gros-boutiste de 28 octets, puis CINQ
regions dont les tailles y sont declarees :

| region | contenu |
|---|---|
| 1 | tagfile Havok 2022.1.0, classe racine **`hkaiNavMesh`** — c'est elle qu'on lit |
| 2 | `hkaiClusterGraph` |
| 3 | `hkaiTraversalAnnotationLibrary` |
| 4 | `hkcdStaticAabbTree` |
| 5 | non-Havok : 1 154 blocs `{hkAabb, n, n x 44 octets}`, 13 879 enregistrements portant position, indice de face et angle |

**La topologie du `hkaiNavMesh`** (Isolation) : 2 348 faces, 8 218 aretes, 3 350 sommets.

- sommets : `hkArray<hkVector4>`, pas de 16 octets (x, y, z, w) ;
- faces : 12 octets `{i32 startEdgeIndex, i32 startUserEdgeIndex, i16 numEdges, i16 numUserEdges}` ;
- aretes : 20 octets `{i32 a, i32 b, u32 oppositeEdge, u32 oppositeFace, u8 flags, u8 pad, u16 cost}`.

Repartition des faces : 1 731 triangles, 477 quadrilateres, 99 pentagones, 31 hexagones, 8
heptagones, 1 octogone, 1 decagone — profil classique de polygones convexes de navmesh.

## 4. L'oracle qui valide le decodage

**Une ancre d'objectif est du terrain joue par definition** : elle doit tomber dans un polygone
du maillage, a une altitude proche.

| carte | ancres dans un polygone | ecart d'altitude median | emprise |
|---|---|---|---|
| Isolation | **24 / 25** | **7,4 cm** | X [-53,42 ; 6,58] Y [-52,75 ; -5,57] Z [112,54 ; 124,08] |
| Kiken'na | **13 / 13** | **9,1 cm** | X [-187,21 ; -155,60] Y [-27,73 ; 6,57] Z [172,18 ; 179,34] |

La seule exception, `assault_bomb` d'Isolation, est a **2,03 m du bord** : le navmesh se retire
le long des murs. Elle est pinnee PAR ROLE dans le test, pas par compte — si une autre ancre
sortait pendant que celle-ci rentre, un simple compte laisserait passer la regression.

Kiken'na vit dans un repere completement different, ce qui **exclut tout codage en dur**.

Garde-rails : `TestOracleAncresDansLeMaillage`, `TestOracleEmpriseEnveloppeLesAncres`,
`TestOracleMaillageSousLesCoques` (`internal/hinavmesh`).

## 5. Les deux usages, et lequel choisir

### `sourceNavmesh` — dessiner le maillage lui-meme

Le fond EST le maillage. Resultat sur Isolation : lisible immediatement, **mais nu** — ni murs
ni structures, puisque le navmesh ne decrit que le sol marchable. 3 160 triangles, 24/25 ancres.

### `navmeshReference` — le maillage sert de REFERENCE (recommande)

La geometrie ordinaire reste la source du dessin ; on ne remplace que ce a quoi les surfaces
sont comparees. **C'est un changement d'entree, pas un nouveau moteur** : toute la machinerie de
substitution lit `r.ref[k]` et ne change pas d'une ligne.

Ce que ca remplace : la reference etait INTERPOLEE depuis une vingtaine d'ancres, ponderee par
l'inverse du carre de la distance — approximation grossiere qui ignore les etages, les rampes et
les creux, et qui extrapole au-dela de `PorteeAncre`. Le maillage donne l'altitude reelle en
chaque point : **845 552 cellules** de la grille d'Isolation en recoivent une.

**Le dome disparait sans qu'on le touche** : il vit onze metres au-dessus du sol, donc compare a
une reference qui est le sol reel, il n'est plus jamais la surface retenue. Ce n'est pas une
soustraction, c'est une comparaison enfin juste. **25 ancres sur 25** au sol.

### `rogneAuNavmesh` — le pendant Forge du masque de callouts

La ou le maillage ne dit rien, on n'affiche rien : **2 175 499 cellules effacees** sur Isolation,
decor du canevas et dalles de ciel partis. Marge de 3 m (`MargeNavmesh`) pour garder les murs qui
bordent le sol.

Aucun polygone a dessiner a la main : **le maillage EST la zone jouable**.

## 6. Pieges payes, a ne pas repayer

1. **Le tampon de reference est LIBERE** par `AppliqueReference` en sortant (memes buffers pour
   les deux voies). Un rognage place apres lui n'y trouve plus rien et efface zero cellule. La
   couverture est donc MEMORISEE (`Rendu.couvertureNavmesh`) au moment de l'armement.
2. **Deux `.mvar` par asset** : `map.mvar` est la carte (300 Ko a 1,6 Mo), `<canevas>.mvar` un
   fichier-lien de ~17 Ko qui nomme le canevas. Prendre le premier de la liste rapporte le lien.
3. **La signature zlib n'est pas `78 9c`** (voir §3) : chercher `78 ..` ne trouve rien.

## 7. Ce qui reste ouvert

- Un **second ilot** de maillage apparait au sud-est d'Isolation : appartient-il a la carte ?
  A trancher a l'oeil, puis eventuellement ne garder que la composante connexe qui porte les
  ancres.
- Les **regions 2 a 5** du blob ne sont pas exploitees. La region 5, dont la structure est
  elucidee (13 879 enregistrements position + face + angle), ressemble a des points
  d'accroche — a instruire si on veut un jour les couverts ou les sauts.
- La **couverture** : 35 cartes du referentiel sont sous le seuil des 1 000 objets et n'ont donc
  pas de navmesh. Elles gardent la chaine ordinaire, qui leur suffit.

## 8. Deux ecritures de la table de chaines (2026-08-30)

Trois cartes restaient en bouillie faute de maillage lisible : **Absolution, Insolence, Insolence
Heavies**. Le decodeur les refusait sur « fichier-tag sans section TST1 ».

**LA CAUSE : un nom de section, pas un format.** Leur section `TYPE` porte exactement les memes
voisins que celle d'Isolation — `TPTR`, `TNA1`, `TBDY`, `THSH`, `TPAD` — mais nomme ses deux tables
de chaines **`TSTR` et `FSTR`** au lieu de `TST1` et `FST1`. Deux generations de l'outil
d'export. `internal/hinavmesh/typetable.go` accepte desormais les deux (`sectionsChaines`).

**CE QUI RESTE, ET LES MESURES QUI LE CERNENT.** Le decodage va bien plus loin puis bute : un
membre de `TBDY` demande l'indice 98 d'une table qui en porte 98.

| | Isolation (TST1/FST1) | Absolution (TSTR/FSTR) |
|---|---|---|
| table des types | 1 044 o, 66 morceaux, 0 vide, queue `\xff\xff\xff` | 1 452 o, 99 morceaux, **3 vides**, queue `\x00\x00` |
| table des champs | 704 o, 61 morceaux, 0 vide, queue `\xff\xff\xff` | 1 048 o, 98 morceaux, **1 vide** |
| TBDY | 808 o | 1 472 o |

Deux differences de forme sautent aux yeux : la generation TST1 **remplit sa fin avec `0xFF`**
(le dernier morceau du decoupage n'est donc pas vide), la generation TSTR **termine par des
nuls** (dernier morceau vide) et porte des **morceaux vides intercales**, ce qu'Isolation n'a pas.

**DEUX PISTES ESSAYEES ET ECARTEES** :
1. *Indexation a partir de 1* (prepender la chaine vide) : decale tout et corrompt les noms de
   types — `hkPropertyId` devient `tITEM`. L'origine n'est donc pas en cause.
2. *Table tronquee par le decoupage* : non, `chaines()` compte deja l'entree vide finale.

**PISTE SUIVANTE, non engagee** : les morceaux vides intercales suggerent que cette generation
n'encode pas ses chaines a la queue leu leu. Verifier si `TSTR`/`FSTR` sont a longueur prefixee,
portent un en-tete de section, ou si les deux tables partagent un espace d'indices commun — dans
ce dernier cas l'indice 98 d'un champ pointerait dans la table des TYPES.

**Garde-rail** : `TestLesDeuxEcrituresDeTableDeChaines` exige qu'Isolation se decode entierement
et qu'Absolution echoue PLUS LOIN que la section manquante. Retirer la reconnaissance de
`TSTR/FSTR` le fait tomber.

**Hors de portee du decodeur** : Thunderhead et Thunderhead Heavies ne publient AUCUN maillage.
Rien a decoder, aucun travail sur le decodeur ne les debloquera.

**TROISIEME PISTE, ESSAYEE ET ECARTEE le 2026-08-30** : un espace d'indices COMMUN aux deux
tables, les champs etant indexes dans la concatenation types + champs. L'indice demande passe
alors a **1 064 pour 197 chaines** — un ecart bien pire que celui qu'on cherchait a expliquer.
Les deux tables ont donc chacune leur espace.

**CE QU'IL RESTE A INSTRUIRE** : les morceaux vides INTERCALES sont le seul indice de forme
restant, et Isolation n'en a aucun. Verifier si `TSTR`/`FSTR` sont a longueur prefixee ou portent
un en-tete de section — dans les deux cas, un decoupage naif aux octets nuls produirait exactement
ce qu'on observe : des morceaux vides la ou se trouvent des octets de longueur ou de remplissage,
et un compte trop court.
