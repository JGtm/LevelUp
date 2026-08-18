# Lot E — phases 1 et 2 (le deuxieme axe de la visee) : journal d'execution

> Executeur, 2026-08-18, worktree `../LevelUp-wt-visee-pub`, branche `wt/visee-pub`, base
> `fed987953`. Perimetre ferme : items E.1 et E.2 du plan
> `PLAN_EXPLOITATION_REGISTRE_FILM.md`, decision D7. La phase 0.1 est CLOSE (gate tenu,
> `LOTEF_PHASE0.md` §1) : la convention est MESUREE et l'accesseur
> `filmdec.BipedPosition.AimPitchDeg()` existe en production. Ce lot-ci ne mesure pas la
> convention, il la PUBLIE et la DESSINE.
>
> Aucune ecriture DuckDB, aucun push, aucun autre worktree. Les films sont lus par chemin
> ABSOLU depuis le cache du principal, en lecture seule ; les artefacts temoins sont ecrits
> SOUS ce worktree (`LEVELUP_REPO_ROOT` pointe le worktree).

---

## E.1 — publier `Point.p` (schema 13)

### Ce qui est livre, fichier par fichier

| Fichier | Ce qu'il porte |
|---|---|
| `internal/analysis/replay/document_aim.go` (NEUF, 116 L) | la CHRONIQUE du schema 13 (pourquoi la version monte pour un champ de sous-objet), la convention mesuree, la reserve +/- 180 / +/- 90 — et le type `Point` lui-meme, DESCENDU de `document.go` avec son nouveau champ `P float32` (cle JSON `p`, `omitempty`) |
| `internal/analysis/replay/build_aim.go` (NEUF, 46 L) | `headingForJSON` (deplacee) et `pitchForJSON` (neuve), cote a cote : elles font le meme geste — arrondir au dixieme de degre — en SENS CONTRAIRE face a `omitempty`, et c'est le seul endroit ou l'opposition se lit d'un coup d'oeil |
| `internal/analysis/replay/build.go` | le cablage dans `decimateTracks` : `pt.P = pitchForJSON(pitch)` sous `if pitch, ok := p.AimPitchDeg(); ok`, du MEME record que la position et que le cap |
| `internal/analysis/replay/document.go` | `SchemaVersion` 12 -> **13** + sa ligne de chronique (renvoi vers `document_aim.go`) |
| `internal/analysis/replay/build_aim_test.go` (NEUF, 6 tests) | l'arrondi, l'omission a plat, la serialisation, le SIGNE, le CABLAGE et son temoin negatif |
| `internal/analysis/replay/golden_inputs_test.go` | fixture `REPLAYINPUTS7` -> **`REPLAYINPUTS8`** : la position serialise AUSSI `PitchRaw` |
| `internal/analysis/replay/golden_assembly_test.go` | la sortie figee compte desormais les points qui portent une ELEVATION, a cote de ceux qui portent un cap |
| `internal/analysis/replay/structure_test.go` | le garde de version : `SchemaVersion != 13` + la raison ecrite (v12 -> v13) |
| `contracttest/replay_contract_test.go` | la ligne `34 -> 34` de la chronique du compte : elle dit POURQUOI le compte ne bouge pas (`Point` n'est pas un champ racine) |
| `api/openapi.yaml`, `apps/web/src/lib/api/generated.ts` | regeneres (`go run ./cmd/openapi-gen`, `npm run generate-types`) : `Point.p`, `format: float` |
| `internal/analysis/filmdec/testdata/ecs_table.tsv` | ligne 710 (ti=35 i21 `unit-desired-aiming-vector-component`) : `doc_field` passe de `Point.H` a `Point.H;Point.P`, et la colonne `exploitable_fr` — qui disait que l'elevation est decodee et JETEE, et qu'aucun champ du document ne la porte — devenait FAUSSE sans correction |
| `internal/analysis/filmdec/ecs_table_guard_test.go` | garde G3 : elle lisait `../replay/document.go` SEUL ; elle lit desormais tous les fichiers non-test du paquet |

### Pourquoi `Point` change de fichier, et pourquoi la garde G3 change avec

`document.go` pesait **539 lignes** (seuil projet : 500). Y ajouter le champ et sa chronique
aurait accru une dette gelee. `Point` (50 L) en descend donc, avec le champ neuf et la
chronique : `document.go` retombe a **488 L**, sous le seuil. Meme geste sur `build.go`
(585 L) : les deux helpers d'angle en sortent, il finit a **581 L** cablage compris. C'est le
geste que la ronde 1 du lot A avait deja fait (R1-5).

**Consequence NON PREVUE, et c'est le seul fix hors du champ initial :** la garde G3
(`filmdec/ecs_table_guard_test.go`) verifie que chaque `doc_field` de la table ECS existe dans
le document — en parsant **un seul fichier**, `../replay/document.go`. Deplacer `Point` l'a
rendue ROUGE sur six references (`Point.X`, `Point.Y`, `Point.Z`, `Point.H`, `Point.Sh`,
`Point.Hp`) alors que les champs existaient toujours. Elle bloquait le gate de l'etape, donc
elle est TRAITEE (regle 7, seule exception admise) : elle lit maintenant tous les fichiers
non-test du paquet. Le defaut etait dormant et double — elle etait **deja aveugle** a tout
champ vivant dans `document_score.go` ou `document_ground_weapons.go`. Le correctif ferme les
deux sens : plus de faux rouge, plus de faux vert.

### Le contrat du champ, ecrit avant la mesure et tenu

`Point.p` est en DEGRES, positif = vers le haut, arrondi au dixieme, publie quand
|p| >= 0,05 deg — **absent = a plat, jamais « inconnu »**. C'est le PIEGE `omitempty` documente
sur `H`, et ici il est ASSUME au lieu d'etre contourne : contrairement au cap, aucun repechage
n'est possible (0 et 360 sont le meme cap ; 0 et 180 ne sont pas la meme elevation).

**CE QUE LA MESURE AJOUTE, ET QUI N'ETAIT PAS PREVU : la bande d'omission est VIDE.** La
formule vaut 360 x (raw + 0,5) / 2048 - 180, et le demi-pas empeche tout `raw` entier de rendre
exactement 0 : la plus petite valeur absolue possible est **0,0879 deg**, soit 1,76 fois le
seuil d'omission. Sur les 2 048 valeurs que le champ R(11) peut prendre, **AUCUNE** ne tombe
dans la bande omise. `p` accompagne donc TOUJOURS `h` dans un artefact cuit depuis un film, et
le cout du champ est plein (cf. tailles). La regle n'est pas retiree pour autant — elle protege
le client d'un producteur futur et donne son sens a l'absence — mais elle est desormais ECRITE
et VERROUILLEE par un test (`TestPitchPublieEstToujoursHorsDeLaBandeDOmission`), pour que le
cout ne soit pas sous-estime a la lecture du code.

### Les tests ajoutes, et ce qu'ils attrapent

| Test | Ce qui le fait rougir |
|---|---|
| `TestPitchForJSONArrondiAuDixieme` | un arrondi au degre, ou a la seconde decimale |
| `TestPitchForJSONOmetLaViseeAPlat` | inverser les deux regles : il verifie DANS LE MEME TEST que `pitchForJSON(0)` vaut 0 et que `headingForJSON(0)` vaut 360 |
| `TestPointOmetPQuandLaViseeEstAPlat` | un tag JSON sans `omitempty` (la fonction seule ne le dirait pas), et la perte du cap sur un point a plat |
| `TestPitchPublieEstToujoursHorsDeLaBandeDOmission` | un changement de convention de decodage : il balaye les 2 048 valeurs du champ et epingle le demi-pas a 0,0879 deg |
| `TestDocumentPortePEtSonSigne` | le CABLAGE retire (tout le reste resterait vert et l'artefact serait a plat), et le signe inverse |
| `TestDocumentSansViseeNePubliePasDElevation` | le temoin NEGATIF : un `pt.P = ...` pose hors du `if` publierait -180 deg partout (`PitchRaw` valant 0 par defaut) |

### Contrat : le compte de champs ne bouge pas, et c'est ecrit

`wantReplayDocumentFields` reste a **34** — verifie sur pieces : ce compte ne compte que les
champs RACINE de `ReplayDocument`, et `Point` vit sous `tracks[].points[]`. Une ligne
`34 -> 34` est quand meme ajoutee a la chronique : l'absence de ligne pour une montee de schema
se lirait comme un oubli. Le champ est verrouille par
`TestReplayContractDescribesEveryPublishedField`, qui compare le type Go `Point` au schema
`Point` du contrat **dans les deux sens**.

`NULLABLE_ARRAYS` : **aucune entree a ajouter, verifie et non suppose** — `p` est un SCALAIRE,
et cette liste n'enumere que les tableaux nullables. La liste exhaustive de
`replayContract.test.ts` ne concerne donc pas ce champ.

### Golden : deux lignes changent, et c'est la justification

    -schema 12 · titre halo_infinite
    +schema 13 · titre halo_infinite
    -15315 point(s) portent un cap de visee · 4620 un bouclier · 163 une fraction de vie
    +15315 point(s) portent un cap de visee · 15315 une elevation de visee · 4620 un bouclier · 163 une fraction de vie

Le fixture d'entrees est regenere (la seule porte d'ecriture : `-update` + `REPLAY_FILM_DIR`)
parce que la magie passe a `REPLAYINPUTS8` — sans `PitchRaw` serialise, le golden
verrouillerait un document dont TOUTES les visees sont a plat, c'est-a-dire pas celui que la
production sert. Cout : **789 960 -> 882 458 octets** compresses (+11,7 %), 128 s de decodage.

### Temoins re-cuits : couverture, distribution, tailles

Trois films, trois modes, trois cartes — les memes que la mesure E.0.1. Un film par processus,
avant-plan (D17) ; artefacts ecrits sous le worktree.

| Temoin | mode / carte | points avec `h` | points avec `p` | couverture |
|---|---|---|---|---|
| `000d5950` | Slayer / Cliffhanger | 15 315 | **15 315** | **100 %** |
| `530820e5` | CTF / Catalyst | 13 338 | **13 338** | **100 %** |
| `7344d24f` | Strongholds / Vagabond | 18 963 | **18 963** | **100 %** |

Le denominateur est le bon, et il a fallu le verifier : les occurrences de la cle `h` dans
l'artefact sont 15 659 / 13 788 / 19 538, mais l'ecart (344 / 450 / 575) ce sont `Shot.h` et
`EquipmentPlacement.h` — deux AUTRES schemas qui portent aussi un cap, hors perimetre. Controle
sur `000d5950` : 344 = 90 tirs avec cap + 254 poses avec cap, les deux comptes etant lus dans
le golden. Sur les POINTS, la couverture de `p` est celle de `h`, exactement — consequence
directe du fait que les deux angles viennent du meme composant et partagent leur validite.

Distribution de `p`, en degres, sur les artefacts publies :

| Temoin | min | p05 | mediane | p95 | max | moyenne | vers le BAS | hors +/-90 |
|---|---|---|---|---|---|---|---|---|
| `000d5950` | -76,6 | -21,9 | **-4,7** | 8,2 | 49,7 | -5,52 | 77,2 % | **0** |
| `530820e5` | -85,5 | -25,6 | **-3,4** | 13,4 | 65,0 | -4,48 | 67,2 % | **0** |
| `7344d24f` | -76,2 | -20,7 | **-3,6** | 9,8 | 82,0 | -4,28 | 73,0 % | **0** |

Ce que j'observe, et ce que je n'observe pas :

- **Le mode tombe pres de 0, quelques degres en dessous** (valeurs les plus frequentes sur
  `000d5950` : -3,1 / -4,0 / -3,4 / +0,1 deg ; mediane -4,7). C'est exactement ce que la phase
  0.1 avait mesure sur le champ BRUT (mode 1006 sur ce film, soit -3,1 deg) : on vise des corps
  depuis une hauteur d'oeil, donc quelques degres sous l'horizontale. Le calque publie
  reproduit la mesure, il ne la deplace pas.
- **77,2 % des visees pointent vers le bas sur `000d5950`**, contre 77,4 % de valeurs sous 1024
  mesurees en phase 0.1 sur les positions NON decimees. La decimation ne biaise pas le signe.
- **Aucune valeur ne sort de +/- 90 deg** sur les trois films (extremes reels -85,5 et +82,0).
  La reserve ecrite en phase 0.1 reste donc entiere et NON TRANCHEE : ce corpus ne pouvait pas
  distinguer « le champ couvre +/- 180 et le jeu borne le tangage » de « le champ ne code que
  +/- 90 sur sa moitie », et il ne le distingue toujours pas.
- **10,1 % des points sont a moins d'un degre de l'horizontale** sur `000d5950` : « a plat » est
  frequent sans etre dominant — 89,9 % des visees publiees ont une elevation que le rendu peut
  montrer.

Tailles (octets) — la mesure est EXACTE, pas estimee. Entre le schema 12 et le schema 13, la
seule difference dans le JSON est l'ajout des jetons de la cle `p` (le numero de schema occupe
le meme nombre d'octets). La somme des longueurs de ces jetons EST l'ecart :

| Temoin | avant (schema 12) | apres (schema 13) | octets de `p` | delta |
|---|---|---|---|---|
| `000d5950` | 2 266 456 | **2 401 937** | 135 481 | **+5,98 %** |
| `530820e5` | 1 494 172 | **1 611 598** | 117 426 | **+7,86 %** |
| `7344d24f` | 1 993 725 | **2 160 362** | 166 637 | **+8,36 %** |

Controle de la methode : pour `000d5950`, la taille « avant » calculee par soustraction
(2 401 937 - 135 481 = 2 266 456) tombe **a l'octet** sur la taille de l'artefact publie par le
lot A apres sa ronde 1. La soustraction n'est pas une approximation ici.

Le cout est reel et il est PLEIN, parce que la bande d'omission est vide : le champ pese 4 a
9 octets par point porteur de visee. C'est un fait pose devant le superviseur, pas un arbitrage
d'executeur — D7 tranche l'arrondi (0,1 deg) et le seuil (0,05 deg), et ils sont appliques tels
quels. Un arrondi au demi-degre ferait tomber ce cout d'environ 40 %, mais ce serait rouvrir D7.

### Decouvertes E.1 (hors perimetre — NOTEES, NON TRAITEES)

1. **La garde G3 etait aveugle** aux champs vivant hors de `document.go`. Le correctif est fait
   parce qu'il bloquait le gate ; le fait qu'elle ait pu rester aveugle depuis le lot A
   (schema 12, `document_score.go`) est le vrai enseignement : une garde qui nomme un FICHIER
   se perime des que le code se reorganise.
2. **`7344d24f` n'a pas de fichier de faits** (le lot A n'en a produit que pour les cinq films
   de son corpus). Il est cuit SANS faits — donc sans `scoreTimeline` ni actions d'objectif.
   Cela n'affecte pas la mesure de `p` (les positions ne dependent pas des faits), mais son
   artefact n'est pas comparable au precedent sur les autres calques.
3. **Le cout du champ est plein.** Si le volume devenait un sujet, le levier est l'ARRONDI
   (0,1 -> 0,5 deg), pas le seuil d'omission — mais c'est une reouverture de D7, donc une
   decision utilisateur.
