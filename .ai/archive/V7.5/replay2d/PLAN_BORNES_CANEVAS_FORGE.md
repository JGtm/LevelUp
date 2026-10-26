# Plan — Les bornes des canevas Forge : choisir le BON sbsp, et le prouver par le film

> Ecrit le 2026-08-16, apres la decouverte du lot alertes (`317175a4c`) : sur trois canevas
> Forge (`fo05_desert`, `fo11_blank`, `fo13_frost`), le catalogue de bornes retient le tag
> sbsp le plus GROS (`mapquant-build`, `bsps[0]`, ~3 870 unites) alors que le film quantifie
> contre un BSP de ~550 unites — 18 cartes cataloguees, **276 matchs (15,1 % du registre)
> servis avec de fausses bornes**, et 14 cartes prouvees mais refusees (41 matchs). C'est la
> cause du report « ecart vertical ~1 270 m » du 14/08 (containment). **Prealable au re-build
> de masse** — sinon la masse re-cuit 276 artefacts faux. Branche `feat/v75` (principal, exige
> `data/` et l'installation du jeu), contrat `plan-execution`.

## Ce qui est acquis (ne pas re-mesurer)

- Le controle : `DetectI0Layout` (decoupage lu dans le film) DOIT egaler les largeurs deduites
  des bornes par la loi du moteur (`AxisWidths = min(26, ceilLog2(ceil(60*extent)))`).
  ACCORD sur `fo08_wetland` 4/4 et `fo09_academy` 3/3 ; DESACCORD systematique `[18 18 18]`
  contre `[15 15 17]` sur les trois canevas fautifs, 12 films sur 12.
- `himap.ReadModuleBSPBounds` rend TOUS les tags sbsp d'un module ; `mapquant-build` prend le
  premier (le plus gros).
- Registre lu SANS base (snapshot `match_registry.parquet`, DuckDB en memoire).
- Instruments : `TestWorldObjectPrecisionLayout` (`WORLDPREC_*`), `TestPreuveLevelIDCartes`
  (66 sous-tests, `himap`), `cmd/mapquant-build`.

## Objectif et critere de succes

Que le catalogue porte, pour chaque carte Forge cataloguee, les bornes du BSP contre lequel
le jeu quantifie — prouve par le controle film, carte par carte — et que le critere de choix
soit STRUCTUREL (lisible dans les modules), pas un ajustement a la main. Si aucun critere
structurel ne tient, un critere valide PAR LE FILM est acceptable a condition d'etre ecrit
comme tel (provenance dans le sidecar), jamais silencieux.

## Decisions tranchees avant execution

1. **La valeur cataloguee est TOUJOURS l'AABB d'un sbsp du module** — jamais des bornes
   derivees du film. Le film ne sert qu'a CHOISIR entre candidats et a CONTROLER.
2. **Aucune carte n'entre ni ne reste au catalogue sans passer le controle** — les 18 entrees
   fautives sont RE-ECRITES (pas retirees : retirer ferait perdre 276 rejeux servis) une fois
   la bonne borne trouvee ; tant qu'elle ne l'est pas, elles restent telles quelles ET la
   ligne de registre reste ouverte.
3. **Le diff du catalogue est explique entree par entree** (le lot alertes l'a fait en ajout
   pur ; ici ce sont des MODIFICATIONS — chaque changement de bornes est liste).
4. Un seul gros module ouvert a la fois (memoire), commande bloquante, jamais d'attente
   passive.

## Phases

### Phase 0 — INVENTORIER les candidats

- [x] 0.1 Pour les CINQ canevas (2 sains, 3 fautifs) : lister TOUS les tags sbsp du module
      avec leur AABB, leur etendue par axe, les largeurs deduites, et tout identifiant
      structurel disponible (nom de tag si resolu, ordre dans le module, references
      entrantes/sortantes que `himap` sait lire — `levl`, `stse`, liens du canevas).
- [x] 0.2 Marquer, par canevas, le ou les sbsp dont les largeurs deduites EGALENT le
      decoupage film (`[15 15 17]`). Publier : combien de candidats par canevas (0, 1, ou
      plusieurs — les trois cas se traitent differemment).

**Gate 0 PASSE (2026-08-16)** — instrument : `himap.TestBSPQuantificationCanevasForge`
(module `ds`, 29 Ko a 447 Ko, ouverture gratuite). Les CINQ canevas portent EXACTEMENT DEUX
tags sbsp, et les DEUX AABB sont IDENTIQUES d'un canevas a l'autre, au millieme pres :

| candidat | X | Y | Z | etendue | W deduit |
|---|---|---|---|---|---|
| arene | [-231,001 ; 231,635] | [-227,082 ; 226,349] | [-946,215 ; 242,327] | 462,64 / 453,43 / 1188,54 | **[15 15 17]** |
| decor lointain | [-1929,558 ; 1937,230] | [-1838,300 ; 1823,785] | [-946,215 ; 1717,299] | 3866,79 / 3662,09 / 2663,51 | [18 18 18] |

**Un seul candidat compatible par canevas** (5/5) — le cas « zero candidat » du gate ne s'est
pas presente, aucun canevas ne sort du perimetre. Taille de tag du candidat juste vs du decor :

| canevas | arene (file#, octets) | decor (file#, octets) | `bsps[0]` tombait |
|---|---|---|---|
| `fo08_wetland` | #4, 122 936 | #2, 14 068 | JUSTE (par hasard) |
| `fo09_academy` | #3, 950 168 | #2, 413 504 | JUSTE (par hasard) |
| `fo05_desert` | #7, 22 512 | #2, 1 372 104 | FAUX |
| `fo11_blank` | #4, 12 612 | #2, 13 776 | FAUX (a 1 164 octets pres) |
| `fo13_frost` | #15, 537 824 | #2, 1 602 164 | FAUX |

**DECOUVERTE HORS PERIMETRE ANNONCE, meme defaut** : le balayage de TOUTE l'installation
(`TestBSPQuantificationTousModules`, 29 modules porteurs de sbsp) rend SIX modules ou le plus
gros tag n'est pas l'arene, pas trois : `fo03_space` (**Starboard**, 24 matchs) et
`fo06_deepsea` (**Dredge**, 16 matchs) portent la meme signature et n'avaient JAMAIS ete
controles. `fo10_deadland` est le sixieme, aucune carte cataloguee dessus. Portee reelle :
**21 entrees fautives / 316 matchs**, pas 18 / 276.

Piste (a) du plan ECARTEE sur pieces : la table de DEPENDANCES du tag `levl` ne reference
AUCUN sbsp (groupes vus : `atcg bloc dcpt dwsg effe flck gptd hlds hscn locs lsnd pfnd rtrn
scen sddt snde sred sssc stse`). La reference vit dans les BLOCS DE DONNEES du `levl`, pas
dans ses dependances — c'est ce que la phase 1 exploite.

### Phase 1 — ETABLIR le critere de choix

- [x] 1.1 Chercher un critere STRUCTUREL qui, applique aux CINQ canevas, retient exactement le
      sbsp compatible : candidats a tester dans l'ordre — (a) le sbsp reference par le
      `levl`/la structure de niveau du canevas ; (b) le sbsp dont l'etendue contient TOUS les
      objets du `.mvar` d'une carte du canevas avec la plus petite marge ; (c) l'index 0 dans
      un ORDRE autre que la taille (ordre du module, ordre des references).
- [x] 1.2 Le critere doit rendre `fo08_wetland` et `fo09_academy` INCHANGES (temoins) et
      corriger les trois autres. Publier le tableau critere x canevas.
- [~] 1.3 Repli PAR LE FILM — **sans objet** : le critere structurel tient 5/5 (et 29/29 sur
      l'installation). Aucune provenance « choisi par controle film » n'est ecrite au
      catalogue, le film reste ce qu'il doit etre — un CONTROLE, jamais une entree
      (decision tranchee n°1).

**Gate 1 PASSE (2026-08-16)** — critere retenu : **STRUCTUREL**, c'est celui du moteur.

> Le sbsp de dequantification est la **REGION 0** du bloc structure-BSP du tag de niveau
> (`levl`), c'est-a-dire le PREMIER sbsp que le scenario declare. `FUN_140be9a14` precalcule
> `W[r][L][axe]` en parcourant ce bloc dans l'ordre, et le composant i0 designe la region 0
> par defaut (mesure film deja acquise : 3 enregistrements sur 291 288 portent un index 1).

Lecture sans plugin `levl` : le bloc se reconnait a sa FORME — un bloc de donnees du `levl`
dont les GlobalID sbsp distincts couvrent EXACTEMENT les tags sbsp du module. Deux blocs le
satisfont sur chaque canevas (440 o et 136 o, GlobalID a +0x8 et +0xc de chaque
enregistrement) et ils donnent le MEME ordre ; un desaccord entre blocs est une erreur, pas
un repli. Implementation : `himap.BSPQuantification` (`internal/himap/sbsp_region.go`).

**Tableau critere x canevas** (piece : `TestBSPQuantificationCanevasForge`) :

| canevas | plus gros tag | plus petit AABB | **region 0 (`levl`)** | film | verdict |
|---|---|---|---|---|---|
| `fo08_wetland` | [15 15 17] | [15 15 17] | **[15 15 17]** | [15 15 17] | temoin INCHANGE |
| `fo09_academy` | [15 15 17] | [15 15 17] | **[15 15 17]** | [15 15 17] | temoin INCHANGE |
| `fo05_desert` | [18 18 18] | [15 15 17] | **[15 15 17]** | [15 15 17] | CORRIGE |
| `fo11_blank` | [18 18 18] | [15 15 17] | **[15 15 17]** | [15 15 17] | CORRIGE |
| `fo13_frost` | [18 18 18] | [15 15 17] | **[15 15 17]** | [15 15 17] | CORRIGE |
| `fo03_space` | [18 18 18] | [15 15 17] | **[15 15 17]** | (phase 2.3) | CORRIGE (hors perimetre initial) |
| `fo06_deepsea` | [18 18 18] | [15 15 17] | **[15 15 17]** | (phase 2.3) | CORRIGE (hors perimetre initial) |

**5 canevas sur 5 justes** (2 temoins inchanges + 3 corriges), et 7/7 avec les deux canevas
decouverts. Robustesse : l'ordre des regions est LISIBLE sur les **29** modules porteurs de
sbsp de l'installation — jamais `ErrRegionBSPIndecidable` — et il coincide avec un controle
croise INDEPENDANT (le plus petit AABB, lu sur les bornes et non sur le tag de niveau) sur les
29. Garde-rail : `himap.TestBSPQuantificationTousModules`, qui nomme les six canevas corriges
et rougit si l'un redevient « le plus gros tag ».

### Phase 2 — REBATIR le catalogue

- [x] 2.1 `cmd/mapquant-build` applique le critere (plus jamais `bsps[0]` nu) ; un log
      `slog.WarnContext` quand plusieurs candidats subsistent ou aucun.
- [x] 2.2 Regenerer les entrees des 18 cartes fautives + ajouter les 14 refusees (elles sont
      deja dans `preuvesLevelID`, seul `mapModule` les refuse). Diff explique entree par
      entree (anciennes bornes -> nouvelles).
- [x] 2.3 CONTROLE `DetectI0Layout` sur TOUS les films disponibles de ces cartes (le lot
      alertes en avait 19 pour 22 cartes ; en prendre autant que possible). Une carte qui
      echoue le controle N'ENTRE PAS et va au registre avec l'ecart.

**2.1** — `himap.BSPQuantification(module)` rend le BSP de la region 0 ET la liste des
candidats ; `mapquant-build` prend le premier et AVERTIT (`slog.Warn`) chaque fois que le plus
gros tag a ete ecarte, avec les deux jeux de largeurs et d'etendues. Le cas « plusieurs
candidats » du plan n'existe pas avec ce critere : la region 0 est unique, et l'indecidabilite
(pas de `levl`, aucun bloc conforme, deux blocs en desaccord) est une ERREUR
(`ErrRegionBSPIndecidable`) qui fait echouer le catalogue entier — pas un repli sur la taille.

**2.2 — diff du catalogue, entree par entree** : **14 ajoutees · 21 modifiees · 0 retiree ·
43 inchangees** (64 -> 78 entrees). Les 21 modifications sont TOUTES le meme changement, du
decor lointain vers l'arene :

| bornes | X | Y | Z | W |
|---|---|---|---|---|
| AVANT (decor) | [-1929,5576 ; 1937,2299] | [-1838,3002 ; 1823,7852] | [-946,215 ; 1717,2988] | [18 18 18] |
| APRES (arene) | [-231,00145 ; 231,63503] | [-227,08232 ; 226,34886] | [-946,215 ; 242,32658] | [15 15 17] |

Entrees MODIFIEES (21) : `fo03_space` **starboard** · `fo05_desert` banished narrows,
cliffside, domicile, fortitude, kaiketsu, shiro, sylvanus · `fo06_deepsea` **dredge** ·
`fo11_blank` corpo, critical dewpoint, curfew, elevation, empyrean, goliath, opulence,
salvation, shogun, solitude, takamanohara · `fo13_frost` snowbound.

Entrees AJOUTEES (14, toutes aux bornes d'arene) : `fo05_desert` dawnbreaker, flood gulch,
solution, vallaheim firefight · `fo11_blank` credence, disciple, ecotone, nadair, pharaoh,
threshold, warehouse · `fo13_frost` lattice, outlook, 944396dd-5661-4a16-b1d8-a6053f762c55.

Les 43 entrees inchangees ne sont pas seulement « non touchees » : elles sont RE-LUES par le
nouveau critere et rendent la MEME valeur (le catalogue est regenere en entier).

**2.3 — controle film** : instrument neuf et durable,
`filmdec.TestControleBornesFilms` (`internal/analysis/filmdec/map_quant_control_test.go`), qui
prend la liste des couples (film, carte) en DONNEE — produite depuis `match_registry.parquet`
par `read_parquet` en DuckDB en memoire, aucune base ouverte. Il remplace le controle a la
main un film a la fois, qui est precisement ce qui avait laisse Starboard et Dredge dehors.

> **342 couples · 66 cartes · 66 vertes · accord 342 · desaccord 0 · illisible 0** (505 s).

Couverture : TOUS les films en cache des 35 cartes touchees (266 couples) + 3 films par carte
des deux canevas TEMOINS et des cartes non-Forge de reference (76 couples). Detail des cartes
corrigees : starboard 14/14, dredge 13/13, snowbound 20/20, curfew 17/17, shiro 15/15,
empyrean 15/15, domicile 14/14, goliath 13/13, cliffside 12/12, elevation 11/11, banished
narrows 10/10, solitude 9/9 (+5/5 en variante classee), sylvanus 9/9, takamanohara 9/9,
kaiketsu 8/8, ecotone 8/8, critical dewpoint 6/6, fortitude 6/6 (+4/4 Heavies), flood gulch
6/6, opulence 6/6, salvation 6/6, solution 6/6, shogun 5/5, threshold 5/5, pharaoh 3/3,
corpo 2/2, credence 2/2, disciple 2/2, dawnbreaker 1/1, lattice 1/1, nadair 1/1, outlook 1/1,
warehouse 1/1. Temoins inchanges verifies : cliffhanger [13 13 14] 3/3, catalyst [15 15 15]
3/3, forest [13 13 13] 3/3, streets [12 12 12] 3/3, prism [14 13 15] 3/3, recharge
[18 18 15] 3/3, plus les 13 cartes de `fo08_wetland` et les 8 de `fo09_academy`.

DEUX cartes ajoutees N'ONT AUCUN FILM en cache et n'ont donc pas de controle propre :
**Vallaheim Firefight** et **944396dd-5661-4a16-b1d8-a6053f762c55**. Elles entrent sur le
controle de leur CANEVAS — `fo05_desert` 66 films et `fo13_frost` 22 films, tous ACCORD —
exactement comme Cole Protocol au lot precedent : les bornes sont celles du MODULE, pas de la
carte, et le canevas est prouve par `level_id`. C'est ecrit, ce n'est pas silencieux.

**Gate 2 PASSE** : catalogue regenere (78 entrees), controle 342/342 vert,
`TestMapQuantCatalogShipped` (invariant « largeur deduite = largeur stockee » + quantum dans
]1/120 ; 1/60]) vert et NON saute, `TestPreuveLevelIDCartes` vert (66 sous-tests),
`golangci-lint run --new-from-merge-base=origin/main` : 0 issue.

### Phase 3 — VERIFIER LES CONSEQUENCES et fermer les reports

- [x] 3.1 Re-cuire au moins DEUX artefacts temoins des cartes corrigees (une `fo11_blank`,
      une `fo05_desert`) en commande bloquante ; publier la couverture et un controle
      d'emprise (nuage des bipedes dans l'AABB, en coordonnees normalisees — l'outil existe).
- [x] 3.2 Rejouer le diagnostic « ecart vertical » du containment sur UN film touche
      (`go run ./cmd/zone-attribution -match 1b1e380f`, ou un autre des 11) : l'ecart de
      ~1 270 m doit tomber a ~0. C'est le controle CROISE qui prouve que la cause etait la
      bonne.
- [x] 3.3 Threshold (`06dfe6d9`) : re-cuire et publier — c'est le temoin manquant du lot
      equipement.
- [x] 3.4 Registre : fermer « bornes de canevas Forge REFUTEES », « 14 cartes prouvees non
      cataloguees », et mettre a jour « ecart vertical ~1 270 m » (cause corrigee, containment
      re-ouvrable). Journal date.

**3.1 — deux temoins re-cuits** (`cmd/replay-build`, commandes bloquantes, un film par
process). L'emprise des JOUEURS de l'artefact est le controle qui parle le plus clair :

| artefact | carte / canevas | AVANT (bornes du decor) | APRES (bornes de l'arene) |
|---|---|---|---|
| `a32ee8d2` | Cliffside / `fo05_desert` | X[-108,21 148,92] Y[-114,94 162,45] **Z[1344,54 1368,15]** — 257 x 277 m | X[-13,09 17,68] Y[-13,70 20,65] **Z[75,99 86,53]** — 31 x 34 m |
| `c0a82e88` | Corpo / `fo11_blank` | X[-44,60 46,73] Y[-20,84 231,18] **Z[1489,77 1494,55]** — 91 x 252 m | X[-5,48 5,45] Y[-2,05 29,16] **Z[140,80 142,93]** — 11 x 31 m |

Le rapport des emprises vaut 8,36 sur Cliffside — exactement 3866,79 / 462,64, le rapport des
deux AABB. Ce n'etait pas « un peu decale » : c'etait une arene de 31 m dessinee sur 257 m,
1 270 m au-dessus du sol.

COUVERTURE INCHANGEE, et c'est la preuve que le correctif change le REPERE et rien d'autre :
`a32ee8d2` 43 pistes / 2 028 frames, tirs 505/581, grenades 40/40, verdicts nominaux, AVANT
comme APRES ; `c0a82e88` 21 pistes / 781 frames, tirs 265/300, grenades 23/23, idem. Aucune
trajectoire n'apparait ni ne disparait — seules leurs coordonnees changent.

CONTROLE D'EMPRISE par le nuage des bipedes (`TestWorldObjectPrecisionImpact`, film
`c0a82e88`, coordonnees normalisees [0,1] de l'AABB) : controle de coherence **ACCORD**
(catalogue [15 15 17] contre film [15 15 17]) ; nuage des bipedes 19 670 positions, emprise
X[0,485 0,511] Y[0,059 0,565] Z[0,406 0,916]. Objets du monde confrontes a ce nuage :
**28,89 % -> 99,40 %** sur les projectiles (ti=41, 3 011 echantillons) et
**27,76 % -> 99,95 %** sur l'equipement (ti=37, 12 887 echantillons) entre les largeurs par
defaut du paquet et celles du catalogue corrige.

**3.2 — le controle CROISE, sur le film meme du report** (`zone-attribution -match 1b1e380f`,
Solitude / `fo11_blank`) :

| | AVANT (registre, 2026-08-14) | APRES (2026-08-16) |
|---|---|---|
| ecart vertical median | **+1 240 a +1 300 m** | **+0,1 m** (p25 +0,0 · p75 +1,2) |
| distance mediane au volume | 1 279 a 1 438 m | **1,0 m** (p25 0,0 · p75 4,5 · max 15,1) |
| appartenance | **0,0 %** | 39,0 % a 0 m · 59,3 % a 2 m · 76,3 % a 5 m · 100 % a 20 m |
| par statistique | — | `zone_captures` 21/70 (30,0 %) · `zone_secures` 2/14 (14,3 %) |
| concordance inter-joueurs | — | 19/19 groupes, 100,0 % (temoin decale : 66,7 %) |

La cause etait bien celle-la. Les volumes d'objectifs viennent de `map_objectives.json` (le
`.mvar`), source INDEPENDANTE des bornes : ce n'est pas un controle circulaire.

**3.3 — Threshold `06dfe6d9` construit** (fo11_blank) : 254 pistes, 7 532 frames, 5,87 Mo,
emprise X[-50,16 50,52] Y[-51,70 51,67] Z[163,84 186,82] — une arene de 101 x 103 m a une
altitude plausible. Le temoin manquant du lot equipement EXISTE desormais : camouflage
2 vies / 7 episodes, surbouclier 3 vies / 3 episodes, grappin 100 tirs / 71 accroches /
71 tractions sur 36 vies, palette `famille_a` (304 lectures, 10 rangs nommes).

MAIS SA COUVERTURE DE CALQUES EST PARTIELLE, et ce n'est PAS un fait de bornes : tirs
245 rattaches sur 2 223 (1 954 sans slot), verdict « partiel : moins des deux tiers
rattaches », pont « non publiable : un slot change de porteur ». Le controle de bornes de ce
film est vert (5/5 films Threshold ACCORD) — la cause est le pont slot -> joueur sur un match
de 254 vies. Consigne au registre comme item PROPRE, pas melange aux bornes.

**Gate 3 PASSE** : ecart vertical retombe de ~1 270 m a +0,1 m, deux temoins re-cuits avec
couverture inchangee et emprise redressee, Threshold construit et publie, registre a jour.

## Decouvertes (NON traitees — regle « zero fix hors perimetre »)

1. **`0f9550e5` (Snowbound) porte encore les fausses bornes** : emprise X[-40,71 388,24]
   Y[-987,43 -640,65] Z[1292,99 1310,08]. Les artefacts deja cuits ne se corrigent pas tout
   seuls — c'est exactement ce que le re-build de MASSE doit reprendre (lot ops a fenetre
   utilisateur, hors de ce plan). Sur les 28 artefacts en cache, SIX vivent sur les canevas
   corriges : `a32ee8d2` et `83ee3f9f` et `084a804d` (`fo05_desert`), `c0a82e88` et
   `008e1bba` (`fo11_blank`), `0f9550e5` (`fo13_frost`) — trois re-cuits ici, trois restants.
2. **`geometryBounds` des artefacts est GENERIQUE, pas par carte** : les 382 objets de
   `data/titles/halo_infinite/reference/map_geometry/map_objects.csv` sont servis a TOUTES les
   cartes, y compris Cliffhanger qui n'est pas Forge — emprise identique X[-10,56 43,67]
   Y[-24,65 39,02] sur les quatre artefacts inspectes. `geometryBounds` ne peut donc pas
   servir de controle d'emprise par carte (c'est pourquoi le controle du 3.1 passe par le
   nuage des bipedes et par le containment).
3. **`rootBlockIndex` existe en DOUBLE dans `himap`** : fonction libre `sbsp.go:371` et
   methode `instances.go:405`, corps equivalents. Dette de lisibilite, aucun effet mesure.
4. **`cmd/mapstruct-build/pickBSP` retient deja le plus PETIT AABB** (a instances non vides) :
   le catalogue de structure designait donc l'arene pendant que le catalogue de bornes
   designait le decor — les deux se contredisaient depuis le debut. Aucun changement requis
   la-bas, mais les deux binaires gagneraient a partager `BSPQuantification`.

## Gates techniques

`go build ./...`, `go vet ./...`, `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ ./internal/himap/ -run 'TestPreuveLevelID|TestMapQuant|TestWorldObjectPrecision'` verts (la suite himap complete depasse le timeout local — connu) ; `golangci-lint run --new-from-merge-base=origin/main` 0 issue ; commits sur `feat/v75`, pas de push. Le re-build de MASSE reste un lot ops a fenetre utilisateur.

## Regles dures

AUCUNE base DuckDB en ecriture ; un seul gros module en memoire ; jamais de borne inventee ;
diff du catalogue explique ; JAMAIS `git add -A` (fichiers d'une autre session dans l'arbre :
`.ai/V7.5/README.md`, `PLAN_HABILLAGE_REJEU_2D.md`, `AUDIT_V7.2.0_MAIN_2026-08-06.md`,
`sonde_bouillie_gamefiles_test.go` — ne pas toucher, ne pas committer).
