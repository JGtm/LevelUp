# Mesures bis 3 — les populations sans lot ni recherche (2026-10-01)

> Répond aux points **27** et **28** de `CRITIQUE_COMPLETUDE_1.md` et à **D-24** du plan (`IDLowBits`
> figé à 13). Worktree `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`, rien de
> commité. Aucun fichier de production modifié sur disque, `grammar.Rev` inchangé
> (`grammar-2026-09-27.3`, `TestGrammarRevSuitLaGrammaire` PASS). Aucune base, aucune cuisson, aucun
> backfill. Films en lecture seule : les 20 films du corpus (19 témoins de `config/replay_corpus.toml`
> + `1c4c63c2`), 9 builds. Ghidra en lecture seule (`search_strings`, `get_xrefs_to`,
> `get_assembly_context`, `read_memory`, `search_byte_patterns`, `decompile_function`,
> `disassemble_function`).
>
> Convention : **mesuré** = compté par une sonde sur les films ; **lu** = lu dans Ghidra ;
> **estimé** = dérivé d'une mesure par une hypothèse écrite ; **hypothèse** = ni mesuré ni lu.
> « Sain » = paquet fermé qui ne contredit aucun des trois invariants de l'écrivain (juge de
> `MESURES_BIS_1.md` §0). Sain n'est pas juste.
>
> **Corrections du 2026-10-02** (critique de complétude n° 2, points N11 et N17) :
> - `DEL ti=0` vaut **658** paquets hors cadre (`mb3_tables.tsv`, marche `reference`, classe
>   « aucune allocation »), comme au §0. Le « 622 » des §3.2 et §3.4 recopiait le compte de
>   `ti=21 i16 flock-position` ; corrigé en place.
> - P7 (§0, §6) : la cause est **établie pour `ti=3 i0`** (`FUN_142ed4aec`, lu). Pour `ti=3 i1`, la
>   grammaire de 26 bits est lue, mais l'appartenance de sa table à `ti=3` est déduite (§8) : elle
>   est **probable**.

## 0. Synthèse

| Population (critique) | Mesuré ici (eid / paquets hors cadre) | Cause établie | Proposition |
|---|---|---|---|
| P1 image-clé incomplète (D-5) | 87 / 13 643 | **La marche d'image-clé du Go perd des records que l'image-clé porte** : l'en-tête exact `[eid][field 0][ti]` est présent pour 87 eid sur 87. Deux gardes du Go : génération 0 traitée comme « handle nul » (57 eid, `kfAnchorFromID`), et voisin / recalage limités à la génération 1 (30 eid de génération 2, bipèdes surtout). Témoin : sur 377 379 entrées vivantes des blocs, les 191 de génération 0 ne sont **jamais** déclarées (0 / 191). | **Lot** (marche d'image-clé : accepter la génération 0 et les générations 2-3 au voisin et au recalage). Borne mesurée : **+1 784 / −1 paquets, +45 694 utiles**, 20 gains contredits. |
| P2 réalloué sous une autre génération | 705 / 12 906 | **Mauvais classement de l'instrument** : 589 eid (12 854 paquets, 99,6 %) sont des naissances de génération **0** (tête `(3+1)&3`) déjà libérées au bloc suivant ; l'entrée « génération 0, drapeaux 0 » y passe pour « vide ». Ce sont des naissances non lues. | **Couvert par L1** (mêmes régions (i), (ii), (iii')) ; correction d'instrument (classe de naissance pour la génération 0). Borne de leur part dans l'oracle-NEW : **+1 440 / −52, +26 979 utiles**. |
| P3 aucune allocation (D-6) | 9 782 / 6 455 (+ 4 306 fermés) | **Signature d'un curseur décalé, pas d'une entité** : têtes uniformes sur des slots vierges (0 : 3 739, 1 : 1 369, 2 : 1 738, 3 : 1 340), cascade d'un seul paquet pour 4 382 paquets. Le composant lu juste avant désigne le coupable ; deux coupables prouvés par A/B : **lecture par index de plage sur Live Fire** (`ti=40 i25` dernier lu : `0797ce72` 1 948 → 367, `60ae07c4` 1 556 → 133) et **`flock-position`** (`ti=21 i16` : 622 → 142 sur le corpus). | **Couvert** par les lots de position (T4-C3 lecture par index, site `flock-position`). D-6 (`0797ce72`) **établi** : T4-C3. Reste non attribué : `ti=20` (694), `DEL ti=0` (658), `ti=14 i1` (241), `ti=41 i2` (161) → **recherche** (§3.4). |
| P4 slot au-delà du plafond | 5 175 / 6 987 | Sous-population de P3 (5 057 eid, 3 990 paquets) plus 115 eid « non mesurables » (dernier chunk, 2 955 paquets). **Pas une question de largeur** : la largeur du jeu est 13 sur tout le corpus (§5). | **Couvert** comme P3. |
| P5 eid introuvables | 2 856 / 13 274 | **Pas une population** : 60 eid = P1 (10 992 paquets, 82,8 %, en-tête présent dans l'image-clé) ; 2 698 eid = P3 (2 062 paquets, garbage) ; 5 « naissances non lues » seulement (120 paquets). | **Couvert** par P1 et P3. 5 eid / 120 paquets non couverts (négligeable). |
| P6 trouvés à plus de 3 paquets | 6 724 / 61 858 | 1 750 eid sont des **naissances attestées** par le bloc (52 890 paquets, 85 %) ; leur NEW n'est pas localisé de façon sûre : au-delà de 3 paquets, la recherche ne discrimine pas (témoin ≈ rejeté). Seul filtre au niveau de l'occurrence qui discrimine : `R(6)` = archétype du pont (68 / 72 contre 6 / 72), mais il ne s'applique qu'à 72 eid. | **Recherche préalable** (la règle qui localise une naissance lointaine) ; la part en régions (i) / (ii) (2 818 eid, 18 805 paquets) suivra L1a / L1b sans limite de distance. Bornes mesurées : `propre+alloc+suivant` +3 227 / −578 (90 gains contredits) ; `propre+pont` +295 / −2. |
| P7 région (iii) : NEW `ti=3 i0` | 1 187 NEW désynchronisés (référence) | **Grammaire non portée, et homonyme mal routée** : `ti=3 i0 low-frequency` = `FUN_142ed4aec` (lu) ; `ti=3 i1 high-frequency` n'est **pas** le composant de `ti=4 i0` (deux tables de composant portent ce nom, lu) : 26 bits (`FUN_142ed4880`) au lieu de `R(8)`. | **Lot** (port de `ti=3 i0` + routage de `ti=3 i1` par descripteur). Gain mesuré : **+30 618 / −10 paquets, +234 454 utiles**, 34 gains contredits ; témoin `ti=4` à 26 bits : −256 370. |
| D-24 `IDLowBits` figé à 13 | — | **Réfuté comme cause** : le cardinal de **tous** les blocs de type 1 du corpus vaut 8 191 (largeur du jeu `ceil(log2(8191)) = 13`, lu), le calibrage de `ScanMarchFacts` retient 13 sur 20 films / 20, et toute autre largeur effondre la fermeture (≤ 30 040 paquets fermés contre 284 704) et multiplie P4. | **Non couvert, rien à couvrir** : garder 13 ; un garde-fou sur le cardinal du bloc (≠ 8 191) suffirait à détecter le cas D-24. |

## 1. Protocole et contrôles

### 1.1 Sondes (tag `research`, aucun fichier de production)

| Fichier (`apps/go-api/internal/games/halo_infinite/film/internal/grammar/`) | Rôle |
|---|---|
| `campagne_bis3_diag_research_test.go` | diagnostic des populations, branché sur la marche de référence (`cmMarcher`) : mêmes premiers rejets, même classe de naissance, même plausibilité, même recherche M1 et même témoin que `campagne_naissances_research_test.go` ; ventilations par population ; oracle P1 (liaison au début du chunk, archétype du pont du bloc du chunk) |
| `campagne_bis3_p6_research_test.go` | P6 : filtres d'occurrence, comptés sur l'eid rejeté ET sur son témoin ; oracle « étendu » par filtre |
| `campagne_bis3_research_test.go` | pilote : référence, oracles par population, largeurs 10 à 15, cardinal des blocs, calibrage `calibrateFrameConfig` ; test unitaire `TestCampagneBis3Log2Haut` (`FUN_1406d310c`) |
| `campagne_bis3_ti3_research_test.go` (tags `research,campagne_overlay`) | région (iii) : grammaire `ti=3` par le crochet de la surcouche des mesures bis 2 ; témoin `ti=4` ; P3 sous `flock-position` au jeu ; P3 sur Live Fire avec la lecture par index de plage ; branche de niveau de `spawn-filter-type` |

La surcouche est celle des mesures bis 2 (`mesures_bis2_overlay/overlay.json`, copies de recherche de
`capture.go`, `lecteur_position.go`, `lecteur_position_exceptions.go`) : aucun fichier de production
n'est modifié sur disque.

Commandes (depuis `apps/go-api`, `GOCACHE` dédié, une commande `go` à la fois, `-count=1`, un film à la
fois, sentinelle `filmproc` 4 Gio, sorties dans le scratchpad puis copiées dans `mesures_bis3_tsv/`) :

```
CAMPAGNE_RACINE=<LevelUp>/data/cache/film_chunks CAMPAGNE_FILMS=<20 ids> CAMPAGNE_SORTIE=<scratchpad> \
  go test -tags=research -count=1 -timeout 180m -run '^TestCampagneBis3Populations$' ./internal/games/halo_infinite/film/internal/grammar/
go test -tags=research,campagne_overlay -overlay=<overlay.json> -count=1 -timeout 120m \
  -run '^TestCampagneBis3Ti3$|^TestCampagneBis3Sites$|^TestCampagneBis3FiltreDeReapparition$' ./internal/.../grammar/
# Live Fire : CAMPAGNE_CATALOGUE=<LevelUp>/data/titles/halo_infinite/reference/map_quant_bounds.json
#   CAMPAGNE_CARTES="0797ce72=Live Fire;60ae07c4=Live Fire - Ranked"
#   CAMPAGNE_BORNES_sgh_interlock=<plages 0, 2, 3 de mb2_regions_live_fire.tsv>  -run '^TestCampagneBis3LiveFire$'
```

Durées : populations 2 573 s (23 marches par film, pics 95 à 382 Mio) ; `ti=3` 545 s (5 marches) ;
sites 277 s ; filtre 189 s ; Live Fire 6 s.

### 1.2 Contrôles (mesurés, tous tenus)

- Marche `reference` = carte v2 : 284 704 paquets fermés, 2 588 167 / 5 961 028 records utiles,
  264 757 hors cadre, 8 388 fermés contredits.
- `oracle-NEW` = MESURES_CIBLEES §2 : 5 170 liaisons, 312 315 fermés, +29 769 / −2 158.
- Populations retrouvées : P1 87 eid / 13 644 paquets (13 643 hors cadre) ; P2 12 906 ; P3 6 455
  hors cadre ; P4 6 987 ; P5 2 856 eid / 15 192 paquets ; P6 6 724 eid (= les 6 724 liaisons de
  `oracle-P6-region`).
- Écarts de libellé avec la critique : « 2 836 eid » (MESURES_CIBLEES §T1-1, RAPPORT) est une coquille,
  le TSV `mc_m1_regions.tsv` et la présente mesure donnent **2 856** ; « 6 451 » (CARTE §4, compté
  rejet par rejet) contre **6 455** (MESURES §T1-5 et ici, compté par eid sur ses paquets).
- Contexte de production Live Fire = MESURES_BIS_2 §3.3 : `0797ce72` +3 269 / −3, `60ae07c4`
  +1 806 / −49 sous la lecture par index.
- `TestCampagneBis3Ti3` joué deux fois (avec et sans le témoin `ti=4`) : variantes communes identiques.

Sorties : `mesures_bis3_tsv/` — `mb3_variantes.tsv` (23 marches par film), `mb3_tables.tsv` (toutes
les ventilations, par film et par marche), `mb3_cadre.tsv` (cardinal, plafond, calibrage),
`mb3_ti3.tsv` + `mb3_ti3_records.tsv`, `mb3_sites*.tsv`, `mb3_livefire*.tsv`, `mb3_filtre*.tsv` ;
`ghidra/` : décompilations et désassemblage cités.

## 2. P1 — image-clé incomplète (D-5)

**Définition** (celle de MESURES §T1-5) : eid rejeté, vivant sous sa génération au bloc de type 1 du
chunk, non lié au monde au début du chunk et absent des déclarations de la première image-clé.

**Mesures** (`mb3_tables.tsv`, tables `p1_detail`, `p1_pont`, `p1_controle`) :

| Génération | eid | Paquets hors cadre | Lié à la fin du chunk précédent | Bloc précédent | Archétype (pont) |
|---|---|---|---|---|---|
| 2 | 30 | 12 071 | 14 oui (4 827 paquets) | vivant, même génération | `ti=35` 27 eid (10 265), `ti=0` 3 |
| 0 | 57 | 1 572 | 11 oui (309) | absent (né pendant le chunk précédent) | `ti=41` 48 eid (1 302), `ti=0` 9 |

- **87 eid sur 87 : l'en-tête exact `[eid 32 bits][field 26 bits = 0][ti 6 bits]` est présent** dans
  le payload d'image-clé du chunk (recherche exhaustive bit à bit, `b3EnTeteDImageCle`) ; aucun n'est
  dans les écartés de la marche. Le hasard d'un motif de 58 bits est négligeable : **l'image-clé porte
  ces entités ; la marche du Go ne les lit pas.**
- Témoin (toutes les entrées VIVANTES des blocs, 377 379 entrées-chunks) : génération 1 déclarée
  373 612 / 373 612 ; génération 3 418 / 418 ; génération 2 3 106 / 3 158 ; **génération 0 : 0 / 191**.
- 14 des 30 eid de génération 2 étaient liés au monde à la fin du chunk précédent : la
  liaison du chunk (`OublierLesSlotsNonPortes`) **oublie** une entité que le monde tenait, parce que la
  marche n'a pas lu son record.
- Par film : `1c4c63c2` 19 eid / 8 474 paquets, `4f77afc1` 20 / 1 570, `a349fea8` 8 / 1 457,
  `084a804d` 13 / 1 104 ; les films à têtes déclarées ≠ 1 de T1-6.

**Cause (lue dans le code Go, mesurée sur le film)** : `kfAnchorFromID` (`keyframe_world.go:72-75`)
rejette toute ancre de génération 0 (« handle null / zone de données »), et le voisin immédiat comme
le recalage d'un bipède n'acceptent que la génération 1 (`keyframe_world.go:131-159`). Or la
génération 0 est une génération valide du jeu (tête `(gen+1)&3`, T1-3 ; 191 entrées vivantes de
génération 0 mesurées dans les blocs). Le masque par vue du bloc ne départage rien (`0x0` sur toutes
les entrées, contrôles compris).

**Gain borné (mesuré)** — `oracle-P1-pont` : 75 eid liés au début de leur chunk avec l'archétype du
pont du bloc du chunk (12 non résolus) : **+1 784 / −1 paquets, +45 694 utiles fermés**, 20 gains
contredits ; HI_1_13_0 +1 050, HI_1_10_0 +564, HI_1_8_0 +133. Hors cadre seulement −1 772 : le reste
des paquets de ces eid bute ensuite sur une autre cause. Additivité : `oracle-NEW+P1-pont` 313 483
fermés contre 312 315 (+1 168). Lier P1 par les NEW de M1 (`oracle-P1`) : +4 / −65 (ce ne sont pas
des naissances).

**Proposition : lot** « marche d'image-clé, toutes générations » — porte de génération 0 levée dans
`kfAnchorFromID`, voisin et recalage ouverts aux générations 0, 2 et 3. Prérequis du gate : la
recherche exhaustive d'en-tête de cette sonde comme témoin (87 / 87 retrouvés), aucune déclaration
nouvelle contraire à un bloc. Risque à mesurer dans le lot : une ancre de génération 0 est aussi
l'aspect d'une zone de données nulle (motivation de la garde) ; le filtre `field 26 = 0, ti < 50`
reste en place.

## 3. P2, P3, P4 et P5

### 3.1 P2 — « réalloué sous une autre génération » : des naissances de génération 0

`p2_detail` (mesuré) : **589 eid sur 705, 12 854 paquets hors cadre sur 12 906 (99,6 %)** ont la
forme « tête de l'eid = génération du bloc + 1 ; bloc du chunk : trace ; bloc suivant : génération 0,
drapeaux 0 ». La génération du bloc vaut 3, l'eid porte la génération 0 = `(3+1)&3` : c'est une
**naissance** sur un slot réutilisé, déjà libérée au bloc suivant, que la classe range en
« réalloué » parce que `cmAlloueSous` (et `v2_datums.go` de la carte) exige `Gen != 0 || Drapeaux != 0`.
Le reste (116 eid, 52 paquets) est dispersé.

Où M1 trouve leur NEW (`population_x_m1`) : région (ii) à ≤ 3 paquets 188 eid (4 108 paquets),
(iii) `ti=3` 71 (1 995), (i) 75 (1 868), (iii') 89 (1 320) ; à plus de 3 paquets 241 eid (3 513).

**Gain (mesuré)** — `oracle-P2` (les 427 liaisons de l'oracle-NEW qui portent ces eid) : **+1 440 /
−52, +26 979 utiles**, 15 gains contredits. **Proposition : couvert par L1** (ce sont des naissances
des régions (i), (ii), (iii')) ; **correction d'instrument** à porter avec L1 : la classe de naissance
de la carte v2 et des sondes doit reconnaître une naissance de génération 0. Aucune recherche.

### 3.2 P3 — « aucune allocation » : un curseur décalé

`p3_detail`, `p3_ferme` (mesuré) :

- Têtes des eid rejetés sur des slots **jamais alloués** : génération 0 3 739, 1 1 369, 2 1 738,
  3 1 340. La tête d'une première allocation serait 1 (`(0+1)&3`) : la répartition est celle du
  hasard. Les « slots » rejetés dépassent le plafond du film pour 5 057 eid sur 9 782.
- Cascade : 6 253 eid n'ont qu'**un** paquet rejeté (4 382 hors cadre) ; le paquet suivant se relit
  normalement. 4 306 paquets de la population sont **fermés** (2 071 à un paquet, dont la forme
  « génération 0, bloc vide » : c'est la population D-2 des « fermés après un rejet »).
- Lecture : l'en-tête rejeté n'est pas un en-tête ; le record précédent a été lu avec une mauvaise
  largeur. Le dernier composant lu avant le rejet (`dernier_composant_x_classe`, hors cadre) :
  `ti=40 i25 unit-command-tick` 2 340 (`0797ce72` 1 200, `60ae07c4` 1 139), `ti=20 i1
  spawn-filter-weight` 694, `DEL ti=0` **658** (corrigé le 2026-10-02 : le « 622 » recopiait le
  compte suivant), `ti=21 i16 flock-position` 622, `ti=14 i1
  crew-marked-objects` 241, `ti=41 i2 object-forward-and-up` 161, `NEW ti=48` 148,
  `ti=35 i59` 143.

**A/B de cause (mesurés, surcouche)** :

| Variante | P3 hors cadre | Coupable désigné | Paquets |
|---|---|---|---|
| Live Fire `0797ce72`, production | 1 948 | `ti=40 i25` 1 195 | 19 124 fermés |
| + lecture par index de plage | **367** | `ti=40 i25` **0** | 22 390 (+3 269 / −3) |
| Live Fire `60ae07c4`, production | 1 556 | `ti=40 i25` 1 139 | 13 965 |
| + lecture par index de plage | **133** | `ti=40 i25` **0** | 15 722 (+1 806 / −49) |
| corpus, référence | 6 455 | `ti=21 i16` 622 | 284 704 |
| corpus, `flock-position` au jeu | 5 892 | `ti=21 i16` **142** | 285 110 (+416 / −10) |
| corpus, les 13 sites au jeu | 5 541 | `ti=21 i16` 142 | 285 217 (+1 315 / −802) |
| corpus, `spawn-filter-type` branche de niveau (`R(9)`) | 6 455 | `ti=20 i1` 694 | **identique à l'octet** |

- **D-6 établi** : les 1 809 rejets « aucune allocation » de `0797ce72` sont des véhicules hors de
  l'arène (`ti=40 i0`, index de plage 3, MESURES_BIS_2 §3.3) lus aux largeurs de la plage 1 ; le
  curseur glisse jusqu'au dernier composant du record (`i25`). La lecture par index retire 81 % de P3
  sur ce film et 91 % sur `60ae07c4`. La « génération 0 née et morte » de la CARTE §4 n'y joue pas.
- `flock-position` au jeu retire 77 % des P3 qu'il désignait.

**Proposition** : P3 n'est pas une population d'entités ; c'est un **compteur de décalages**, utile
comme témoin de gate. **Couvert** par les lots de position (T4-C3 lecture par index de plage — sa
borne propre est celle de MESURES_BIS_2 : +3 269 et +1 806 paquets ; site `flock-position` :
+416 / −10). **Recherche** pour le reste (§3.4).

### 3.3 P4 — « slot au-delà du plafond » : une partie de P3

`population_x_classe` (mesuré) : 5 057 eid sur 5 175 sont « aucune allocation » (3 990 paquets hors
cadre) ; 115 eid « non mesurables » (dernier chunk, sans bloc suivant) portent 2 955 paquets ; 3 eid
« NEW lu ». Live Fire par index : 1 738 → 493 et 1 369 → 270 ; `flock-position` au jeu : 6 987 → 6 612.
**Couvert** comme P3. Rien n'y relève de la largeur d'identifiant (§5).

### 3.4 Coupables de P3 non attribués (recherche)

- `ti=20 i1 spawn-filter-weight` (694 paquets hors cadre, `111fa685` 320, `bf15f7ab` 172). Lu : le
  lecteur `FUN_142ed70b8` est un seul `FUN_1406d84b4(…, 0x10, 1, 0)` = `R(16)`, conforme au Go. Lu :
  `spawn-filter-type` (`FUN_142ed708c` → `FUN_142ecf744`) branche sur le niveau au cas 1 (niveau < 2 :
  `FUN_1407f2058` ; sinon `FUN_142b67e34` = `R(9)`), le Go lit toujours la première forme, et le niveau
  du registre vaut 2. **Mesuré : porter la branche ne change rien** (identique à l'octet) — le cas 1
  n'apparaît pas, ou `i0` n'est pas dans ces masques. Le cas 3 passe `p5 = 1` à `FUN_14076e494` et
  finit sur `FUN_1407f1ff4("associated-participant-handle")`, non relus. Cause non établie.
- `DEL ti=0` (**658**, corrigé le 2026-10-02 d'après `mb3_tables.tsv` ; dont la moitié des
  « fermés après rejet ») : un DEL suivi d'un en-tête nul ; lien
  avec D-2 (pied de trame), non tranché.
- `ti=14 i1 crew-marked-objects` (241, `11de8353`) : port `partiel` déclaré par la table (largeur
  runtime).
- `ti=41 i2 object-forward-and-up` (161, `0797ce72`, inchangé sous la lecture par index).

Proposition : une **recherche** groupée « coupables résiduels de P3 », gate = ces quatre comptes.

### 3.5 P5 — « introuvables » : P1 et P3

`population_x_classe`, `p5_detail` (mesuré) :

| Classe | eid | Paquets hors cadre | Ce que la recherche étendue trouve |
|---|---|---|---|
| vivant au bloc du chunk (= P1) | 60 | **10 992 (82,8 %)** | en-tête exact présent dans l'image-clé : 60 / 60 ; NEW dans le chunk précédent : 59 |
| aucune allocation (= P3) | 2 698 | 2 062 | en-tête exact absent de l’image-clé pour 2 596 eid (présent pour 102) ; aucune allocation aux blocs |
| naissance non lue | 5 | 120 | rien |
| autres | 93 | 100 | — |

**Proposition : couvert par P1 et P3.** Les 5 naissances introuvables (120 paquets) restent non
couvertes, sans enjeu.

## 4. P6 — naissances trouvées à plus de 3 paquets

### 4.1 Ce qu'est la population (mesuré)

6 724 eid, 61 858 paquets hors cadre, 757 461 utiles en jeu. Par classe : **naissance non lue
1 750 eid, 52 890 paquets (85,5 %)** ; « réalloué » (naissances de génération 0, §3.1) 233 / 3 485 ;
« aucune allocation » 4 485 eid mais 2 381 paquets seulement. Par région de l'occurrence retenue :
(iii') après rejet 3 584 eid / 40 893 paquets ; (ii) 1 785 / 10 289 ; (i) 1 033 / 8 516 ; (iii')
ouverte 284 / 1 557 ; (iii) 37 / 599.

### 4.2 Les filtres contre le témoin (mesuré, `p6_filtres`)

Eid NON liés par l'oracle-NEW : 11 927 rejetés, 11 801 témoins (72 et 72 avec une cible de pont).
Compte = eid qui ont au moins une occurrence passant le filtre dans la classe de distance.

| Filtre | 4-10 paquets : rejetés / témoins | > 10 paquets : rejetés / témoins |
|---|---|---|
| région non lue | 1 345 / 333 | 6 497 / 5 315 |
| + traversée propre | 1 126 / 229 | 5 729 / 4 392 |
| + en-tête suivant plausible | 140 / 42 | 1 738 / 1 213 |
| + allocateur (rang < 64, tête prédite) | 665 / 9 | 1 610 / 154 |
| + allocateur + suivant | 94 / 1 | 705 / 57 |
| + `R(6)` = archétype du pont (cible : 72 eid) | 10 / 0 | 58 / 6 |
| + pont + suivant | 1 / 0 | 18 / 2 |

Lecture : au-delà de 10 paquets, région, traversée et en-tête suivant ne séparent pas l'eid de son
témoin (rapport ≤ 1,5). Le filtre « allocateur » sépare (rapport ≈ 10), mais c'est une propriété de
l'**eid** (son slot et sa tête sont prédits), pas de l'occurrence : il ne prouve pas que l'occurrence
retenue est le NEW. Le seul filtre au niveau de l'occurrence qui sépare est le **pont** (`R(6)` =
archétype du bloc suivant : 68 / 72 contre 6 / 72, ≈ 8 % de faux positifs), et il ne s'applique qu'à
72 eid : 5 714 des 6 033 naissances non lues ont un masque vide au bloc suivant (MESURES §2).

### 4.3 Bornes (mesurées, oracle « étendu »)

| Oracle | Liaisons | Gagnés / perdus | Δ utiles fermés | Gains contredits |
|---|---|---|---|---|
| `P6-region` | 6 724 | +2 388 / −7 287 | −83 477 | 235 |
| `P6-propre` | 5 930 | +2 058 / −6 832 | −87 665 | 259 |
| `P6-propre+suivant` | 1 816 | +3 619 / −1 466 | +38 518 | 137 |
| `P6-propre+alloc` | 1 728 | +1 858 / −4 148 | −79 600 | 147 |
| `P6-propre+alloc+suivant` | 761 | **+3 227 / −578** | **+39 083** | 90 |
| `P6-propre+pont` | 68 | +295 / −2 | +6 033 | 8 |
| `P6-propre+pont+suivant` | 19 | +38 / 0 | +500 | 0 |
| `oracle-NEW` + `P6-propre+alloc+suivant` | 5 931 | +31 809 / −2 297 (contre la référence) | +370 823 | 540 |

**Proposition : recherche préalable**, pas de lot. La population est réelle (85 % de naissances
attestées), mais aucun lecteur de production ne peut aujourd'hui dire OÙ est leur NEW : la distance
n'est pas un critère (une entité statique n'a pas de DELTA pendant des secondes), et le témoin montre
que la recherche aveugle lie au hasard (`P6-region` : −4 899 paquets nets). Deux faits orientent la
recherche : (1) la part en régions (i) et (ii) (2 818 eid, 18 805 paquets) suivra L1a / L1b qui lisent
une région, pas une distance — elle est donc HORS de la borne de L1 telle que mesurée (≤ 3 paquets)
et l'élargit ; (2) la part en région (iii') (3 868 eid, 42 450 paquets) dépend de R-L1 (a) (D-3).
Gain borné, non garanti : +2 649 paquets nets (`propre+alloc+suivant`), +1 901 au-delà de l'oracle-NEW.

## 5. D-24 — `IDLowBits` figé à 13 (mesuré + lu)

**Lu (Ghidra)** : l'identifiant de record se lit par `FUN_1406d3140(_, br, 7, &id)` (appel en
`FUN_1406cd128`) ; domaine 7 : base 0 et compte `DAT_144706100` (`FUN_140d10bb0`, initialisation des
domaines) ; largeur = `FUN_1406d310c(DAT_144706100)` = `ceil(log2(n))` ; `FUN_1408f1618` ne fait
grandir `DAT_144706100` qu'à l'allocation d'un slot ≥ cardinal. Le bloc de type 1 écrit
`min(cardinal, DAT_144706100)` entrées (`type1_datums.go`) : **son cardinal EST la largeur du film**.

**Mesuré** (`mb3_cadre.tsv`) :

- Cardinal de **chaque** bloc de type 1 des 20 films : **8 191 entrées** (de 19 à 67 blocs par film),
  soit une largeur de **13 bits** partout. Plafond (plus grand slot alloué) : 2 312 à **8 190**
  (`1c4c63c2`, un slot sous la limite ; 8 191 l'aurait fait passer à 14 bits).
- `calibrateFrameConfig` (le calibrage de `ScanMarchFacts`, 10..15) retient **13 sur 20 films** :
  19 profils francs (dauphin 10, 11 ou 12 avec 2 à 60 paquets localisés contre 44 à 146), 1 profil
  plat (`a349fea8`, 124 contre 116 pour 11 : défaut conservé = 13).
- Marche de référence sous chaque largeur (corpus) :

| Largeur | Paquets fermés | Hors cadre | P3 hors cadre | P4 (eid / hors cadre) |
|---|---|---|---|---|
| 10 | 20 011 | 237 574 | 59 043 | 0 (slot ≤ 1 023) |
| 11 | 23 521 | 64 314 | 27 503 | 0 (slot ≤ 2 047) |
| 12 | 30 040 | 296 207 | 185 035 | 25 736 / 48 200 |
| **13** | **284 704** | **264 757** | **6 455** | **5 175 / 6 987** |
| 14 | 26 958 | 370 658 | 302 567 | 69 349 / 130 168 |
| 15 | 20 230 | 428 890 | 403 680 | 18 693 / 12 818 |

**Verdict** : une largeur calibrée par film ne change rien (elle vaut 13 partout) ; toute autre largeur
détruit la marche et gonfle P4 d'un facteur 7 à 19. La population « au-delà du plafond » n'a **aucun
lien** avec `IDLowBits`. D-24 reste vrai en principe (la largeur suit `DAT_144706100`) mais ne se
produit sur aucun film du corpus. **Non couvert, sans enjeu mesuré** ; le seul geste utile est un
garde-fou peu coûteux : refuser (ou dater) un film dont un bloc de type 1 n'a pas 8 191 entrées, et
alors lire la largeur sur le cardinal du bloc (pas par balayage).

## 6. P7 — région (iii) : les NEW `ti=3` désynchronisés (point 28)

### 6.1 Grammaire (lue, Ghidra)

- `low-frequency` : chaîne `0x143c957c8`, accesseur `0x141177bd0`, **une** table de composant
  `0x143d07b40` = [accesseur, `1404ab600`, écrivain `142eda938`, `1411c8f80`, `14076ce9c`, lecteur
  **`142ed4aec`**]. `FUN_142ed4aec` (décompilé et désassemblé) :
  `FUN_14076e494(.., 0x10, 0, 0, 0)` (position, `lireE494`) ; `FUN_140c5f938(.., 0)` (avant / haut) ;
  `R(16)` ; `R(8)` ; `R(2)` ; `n = R(6)` ; `n` fois { `f = R(3)` (`FUN_1424d9a30`, destination
  `RDI+0x27` lue au désassemblage) ; si `f & 1` position ; si `f & 2` avant / haut ; `R(16)` ;
  `R(5)` (`FUN_1424ccc74`) }. Toutes les briques sont déjà portées.
- `high-frequency` : chaîne `0x143c95710`, accesseur `0x14119d7f0`, **deux** tables : `0x143d06a60`
  (lecteur `FUN_14076d034` = `R(8)`, le port actuel de `ti=4 i0`) et `0x143d07af0` (lecteur
  `FUN_142ed4880` = `R(16) + R(8) + R(2)`, mêmes champs `+0x52c`, `+0x52e`, `+0x52f` que la
  basse fréquence), **voisine immédiate** de la table de `low-frequency`. Le Go route les composants
  par leur NOM : `ti=3 i1` lit `R(8)`.

### 6.2 Mesure (surcouche, `mb3_ti3.tsv`, `mb3_ti3_records.tsv`)

| Variante | Fermés | Gagnés / perdus | Δ utiles | Hors cadre | Gains contredits |
|---|---|---|---|---|---|
| référence | 284 704 | — | — | 264 757 | — |
| `ti=3 i0` porté seul | 285 130 | +836 / −410 | +1 498 | 255 830 | 64 |
| `ti=3 i1` à 26 bits seul | 286 155 | +1 462 / −11 | +11 071 | 263 680 | 9 |
| **`ti=3 i0` + `i1` à 26 bits** | **315 312** | **+30 618 / −10** | **+234 454** | **235 620** | **34** |
| témoin : `ti=4 i0` à 26 bits | 34 399 | +6 065 / −256 370 | −2 495 992 | 266 969 | 4 628 |

- Records `ti=3` : en référence, **1 187 NEW désynchronisés sur `low-frequency`** et 114 traversés,
  1 495 DELTA ; avec la grammaire complète, 0 désynchronisé, 2 941 NEW et **76 586 DELTA** traversés
  (64 743 dans des paquets fermés).
- Le gain est concentré : `fb1a1a72` (Banished Narrows, CTF) 22 318 → 42 081 fermés (+19 764),
  `51ebbc0f` (Banished Narrows, Oddball) 9 847 → 19 872 (+10 029), `c75f33b8` (Curfew) +817 ; les
  17 autres films à ±5 paquets.
- Le témoin `ti=4` effondre la marche : les deux tables `high-frequency` sont **deux composants
  distincts** ; `ti=4 i0` = `R(8)` (port actuel juste), `ti=3 i1` = 26 bits.

### 6.3 Ce que cela corrige dans les documents

- BIS_1 §4 : « lier (iii) est nuisible (−65 net) » reste vrai de l'**oracle** (le `R(6)` d'un NEW
  désynchronisé ne suffit pas) ; la cause est la grammaire, et la porter gagne +30 608 nets — plus
  que la borne entière de L1 (+27 611).
- RAPPORT §1 : « (iii) presque aucun utile (412) » sous-estime : la grammaire `ti=3` débloque aussi les
  DELTA `ti=3` lus faux en `R(8)` (+234 454 utiles fermés).

**Proposition : lot** « `ti=3` low-frequency + routage des homonymes par descripteur » : porter
`FUN_142ed4aec` pour `ti=3 i0` et lire `ti=3 i1` par `FUN_142ed4880`, en routant `high-frequency` par
archétype (ou par descripteur) et non par nom ; gate : `fb1a1a72`, `51ebbc0f`, `c75f33b8` en hausse,
aucun film en baisse, juge des invariants (34 gains contredits mesurés à surveiller), témoin `ti=4`
inchangé. Gain borné mesuré : +30 618 / −10. Ce lot ne dépend d'aucune recherche, et monte
`grammar.Rev` (cf. CRITIQUE point 4).

## 7. Tableau des propositions

| Population | Statut proposé | Gain mesuré (borne) | Dépendance |
|---|---|---|---|
| P7 région (iii) `ti=3` | **lot** | +30 618 / −10 paquets ; +234 454 utiles | aucune |
| P1 image-clé (générations 0 et 2) | **lot** | +1 784 / −1 ; +45 694 utiles (oracle) | aucune |
| P2 naissances de génération 0 | couvert par L1 + correction d'instrument | +1 440 / −52 ; +26 979 utiles (part de l'oracle-NEW) | L1 |
| P3 / P4 décalages | couvert par T4-C3 et `flock-position` ; reste = **recherche** | Live Fire −81 % / −91 % de P3 ; `flock-position` −77 % de son coupable | lots de position |
| P5 introuvables | couvert (= P1 + P3) | — | P1, P3 |
| P6 naissances lointaines | **recherche préalable** | ≤ +3 227 / −578 (non garanti, filtre non validé au niveau de l'occurrence) | R-L1 (a), L1a / L1b |
| D-24 `IDLowBits` | non couvert, sans enjeu ; garde-fou proposé | 0 (13 partout) | — |

## 8. Limites

- Les oracles lient au début d'un paquet ou d'un chunk : ce sont des bornes de fermeture, pas des
  lecteurs ; P1 lie avec l'archétype du pont (constant par archétype, NOTE 5.21 §3.2), non avec le
  record d'image-clé relu.
- Les A/B de position et de `ti=3` passent par le crochet de la surcouche, pas par un port ; un port
  doit refaire la mesure.
- La branche de niveau de `spawn-filter-type` est portée pour le cas 1 seulement ; le cas 3 (`p5 = 1`
  de `FUN_14076e494`, `FUN_1407f1ff4`) n'est pas relu.
- `ti=3` : l'appartenance de la table `0x143d07af0` à `ti=3` est **déduite** (voisinage, champs, et
  mesure : +30 608 nets ; témoin `ti=4` : −256 370), pas lue dans le registre d'archétypes du jeu.
- Contexte Live Fire : bornes des plages 0, 2, 3 lues dans les modules HI_1_13_0 (MESURES_BIS_2),
  appliquées aussi à `60ae07c4` (HI_1_8_0).
- Rien n'est publié : aucune mesure ne dit l'effet sur le document de rejeu.

## 9. Découvertes (consignées, non traitées)

1. **Deux composants de même nom, deux grammaires** (`high-frequency`) : le routage par nom du
   dispatch (`dispatch_item.go:122`) est faux pour `ti=3` ; d'autres homonymes peuvent exister —
   à recenser (nom → tables de composant dans le binaire).
2. **La génération 0 est valide** et le Go la traite comme nulle en au moins deux endroits
   (`kfAnchorFromID`, classe de naissance des instruments) ; les autres lecteurs d'eid sont à revoir.
3. `ecs_table.tsv` : `ti=20 i1` a pour `deser_addr` `FUN_14076ce9c`, qui est l'entrée commune `+0x20`
   de toutes les tables, pas le lecteur (`FUN_142ed70b8`, déjà cité par `dispatch_biped.go:250`).
4. `1c4c63c2` alloue le slot 8 190 : un slot de plus aurait porté la largeur à 14 bits (D-24 réel).
5. Les `DEL ti=0` suivis d'un en-tête nul portent la moitié des « fermés après rejet » de P3 (lien D-2).

## 10. Gate de la sonde (joué le 2026-10-01, depuis `apps/go-api`, GOCACHE dédié)

```
$ go vet -tags=research ./internal/games/halo_infinite/film/internal/grammar/
rc=0
$ go vet -tags=research,campagne_overlay -overlay=<overlay.json> ./internal/games/halo_infinite/film/internal/grammar/
rc=0
$ go test -tags=research -count=1 -run TestCampagneBis3 ./internal/games/halo_infinite/film/internal/grammar/
--- SKIP: TestCampagneBis3Populations (0.00s)
--- PASS: TestCampagneBis3Log2Haut (0.00s)
ok  	levelup/go-api/internal/games/halo_infinite/film/internal/grammar	0.059s
$ go test -tags=research,campagne_overlay -overlay=<json> -count=1 -run TestCampagneBis3 ./internal/.../grammar/
--- SKIP: TestCampagneBis3Populations / Ti3 / Sites / LiveFire / FiltreDeReapparition (sans variables)
--- PASS: TestCampagneBis3Log2Haut (0.00s)
ok  	levelup/go-api/internal/games/halo_infinite/film/internal/grammar	0.059s
$ go test -count=1 -run Rev ./internal/games/halo_infinite/film/internal/grammar/
--- PASS: TestGrammarRevSuitLaGrammaire (0.05s)
ok
```

Les sondes sur films ont toutes rendu `PASS` (sorties en §1.1). `git status` : aucun fichier suivi
modifié de plus qu'au début de la mission ; `grammar.Rev` = `grammar-2026-09-27.3`.
