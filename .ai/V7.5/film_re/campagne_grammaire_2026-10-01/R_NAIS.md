# R_NAIS — chantier « nais » : R-L1 (a), (b), (d) et R-P6 (2026-10-02)

> Campagne de recherche sur la grammaire du jeu, phase 2, recherches préalables au lot L1
> (`.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` §6.1 tableau « Recherches préalables », §6.2 « R-L1 »,
> ligne R-P6). Worktree temporaire `LevelUp-wt-cg-nais`, HEAD détachée sur `fe18bf67c`. Rien n'est
> commité. Aucun fichier suivi n'est modifié : seuls des fichiers NEUFS (préfixe `r_nais`), six sondes
> `*_research_test.go` sous `//go:build research`, cette note, des TSV et des décompilations Ghidra.
> `grammar.Rev` inchangé (`grammar-2026-09-27.3`). Aucune base, aucune cuisson, aucun backfill, aucune
> surcouche `-overlay`. Ghidra en lecture seule stricte (`decompile_function` seulement).
>
> Conventions (celles de la campagne) : **lu** = lu dans Ghidra (HaloInfinite.exe HI_1_13_0) ;
> **mesuré** = compté par une sonde sur les 20 films (19 témoins de `config/replay_corpus.toml` +
> `1c4c63c2`) ; **estimé** = dérivé d'une mesure par une hypothèse écrite ; **supposé** = ni lu ni
> mesuré. **Établi** = lu ET mesuré concordants, ou mesuré avec témoin. « Sain » = paquet fermé qui
> ne contredit aucun des trois invariants de l'écrivain du juge de la campagne (`cmContredit` :
> ordre de la vue B, masques, vue C) ; « factice » = fermé et contredit (décision D2 : exclu). Sain
> n'est pas juste (§2 le montre une fois de plus).

## Corrections du 2026-10-02 (ajoutées après coup ; le texte d'origine ci-dessous n'est pas réécrit)

Sources : verdicts adverses du chantier nais (`VERIFICATIONS_ADVERSES_R.md`, « Chantier nais »),
`CRITIQUE_COMPLETUDE_R.md` (points 14, 20, 23) et `R_COMB_2.md` §5.2.

1. **« Pool unique ≤ 2,3 % »** (§3.1) : faux ; maximum **2,80 %** (HI_1_4_1, 6 / 214), corpus 0,34 %
   (112 / 33 414). La conclusion ne change pas.
2. **« Un paquet sur seize »** (§2.2) : le témoin prend `Index%16 == 0`, soit 34 980 paquets, environ
   **1 / 8** des quelque 280 106 fermés après terminateur (indices des paquets delta espacés).
3. **R-L1 (b), témoin** : non symétrique (le vrai terminateur et un bit de présence 0 y arrêtent la
   vue C, barrière absente sous l'hypothèse nulle) : il sous-estime le taux nul sans pouvoir inverser
   l'écart. Les 95,7 % (sains et contredits) se comparent au témoin COMPLET, 4 129 / 34 980 (11,8 %),
   pas au témoin des seuls sains (11,0 %). « Les suivantes identiques » vaut pour 3 644 des 3 848
   paquets de la forme dominante. « Le dernier record a débordé » et « l'en-tête rejeté est une entrée
   de vue C » sont des DÉDUCTIONS (supposé). T3-C2 n'est sans perte saine que si le 4e invariant
   entre au juge.
4. **R-L1 (b), portée de la conclusion (critique point 23)** : « ces 4 598 fermetures sont factices »
   est établi pour les 4 402 qu'un début antérieur ferme ; 196 restent inexpliqués (32 sains au juge,
   164 contredits), et le sous-groupe à reste ≤ 8 bits (648 paquets) n'est pas analysé à part. Ces
   deux sous-groupes sont à mesurer avant d'adopter l'invariant L0.6 (PLAN D-113).
5. **R-L1 (a)** : les 100 « réalloués sous une autre génération » reposent sur D-43 (non vérifié
   liaison par liaison) ; sans eux, 994 / 1 289 (77,1 %). Aucun témoin d'attestation fortuite d'un X
   lu à un mauvais bit. Les tables du pont (0 / 33) et du décalage (64,5 % contre 84,2 %) ne sont que
   dans le scratchpad du chantier, pas dans `r_nais_tsv/` (PLAN D-101).
6. **R-L1 (d)** : les pourcentages « variables » (80,57 → 84,70 %, corpus 43,42 → 45,57 %) sont des
   utiles FERMÉS bruts ; en sains, HI_1_13_0 80,30 → 84,69 %, corpus 43,17 → 45,43 %. L'en-tête de
   `r_nais_scores.tsv` a 8 colonnes pour 10 ; les lignes « -hors-evenements » viennent d'un code
   retiré (non reproductibles). `bit_cfg_chunks` « 0 » = 20 chunks sans paquet delta, pas un bit à 0.
   « Lève le blocage » veut dire désactiver L1a sur HI_1_8_0 à HI_1_11_0 (gain nul) ; seuil choisi
   dans l'échantillon, sans validation hors échantillon.
7. **R-L1 (d) en combinaison (R-COMB-2, mesuré)** : sous LM, la condition causale EN LIGNE s'allume sur
   `084a804d`, `111fa685`, `1c4c63c2`, `60ae07c4` et `e5adf7b2` (52, 26, 19, 42 et 21 chunks) ; « la
   condition désactive L1a sur HI_1_8_0 à HI_1_11_0 » ne vaut donc plus dans la vague. La forme
   `|cumul` (score de la référence) ne s'allumerait pas. L1a causal en ligne : +13 374 sains seul,
   +29 594 en marginal dans C11. Réserve : la copie choisit son début avec le juge (`cmTeteInv`).
8. **R-P6** : le filtre `propre+pont` de BIS_3 §4 (68 / 72 contre 6 / 72, +295 / −2) est omis et
   discrimine mieux que `ferme+suivant` ; les taux de faux positifs sont sous-estimés (le témoin ne
   compte que les cibles qui en ont un) ; `ferme+alloc` mêle propriété d'eid et d'occurrence ;
   l'extension à (iii') au-delà de 3 paquets est extrapolée.
9. **Gate 5 (critique point 20)** : `r_nais_marche_research_test.go` (`rnMarcher`) est une seconde
   recopie de la marche (`cmMarcher` + crochet) : elle suit tout lot qui change le pilotage.
10. Les sondes de cette note sont taguées `research` seul, sans surcouche ; elles n'ont pas été
    rejouées après J12 (chiffres supposés identiques, la carte v2 de référence l'étant). Gate de la
    note : `archlint` rouge à `fe18bf67c` (`TestNoExpiredTODO`, hors chantier), soldé depuis.

## 0. Synthèse

| Item | Statut | Réponse en une phrase | Chiffre clé (mesuré sauf mention) |
|---|---|---|---|
| R-L1 (a) région (iii') | **établi** | L'en-tête rejeté est un vrai DELTA dans au moins 84,9 % des 1 289 cas ; l'écrivain n'écrit JAMAIS un NEW après un DELTA (lu) : les « NEW » trouvés derrière ne sont pas des records. | 1 094 / 1 289 liaisons derrière un DELTA d'entité attestée ; le paquet relu depuis l'occurrence ferme sain 5 / 2 143 fois (témoin 2 / 177) ; `R(6)` = archétype du pont 0 / 33 |
| R-L1 (b) 4 598 fermés après rejet | **établi** | Ni « pied de trame d'une vue vide », ni la piste « DEL ti=0 + en-tête nul » : la vue B a débordé son terminateur, l'en-tête rejeté est une entrée de vue C, et la vue C se referme par auto-synchronisation. Fermetures factices (D2), y compris les 1 008 que le juge actuel dit saines. | juge : 3 590 contredits (78,1 %) ; un début de vue C antérieur, précédé de `000`, ferme le paquet pour 4 402 / 4 598 (95,7 %) contre 11,0 % au témoin ; eid nul 0 / 4 598 |
| R-L1 (d) condition par film | **établi sur le corpus (seuil choisi sur ce même corpus)** | Activer le localisateur élargi (`tete-bloc+inv`) seulement si l'allocateur à cinq pools prédit au moins 50 % des NEW que la marche de référence lit proprement dans le film. Aucune baisse saine sur aucun film. | corpus +15 070 paquets sains, +159 864 utiles sains ; HI_1_13_0 75,93 % d'utiles sains sur dénominateur fixe (référence 70,54 %) ; 0 film en baisse (variante causale : 0 aussi) |
| R-P6 naissances à plus de 3 paquets | **partiel** | Un filtre d'OCCURRENCE validé existe (« le paquet relu depuis l'occurrence se ferme, sain »), mais il ne discrimine que faiblement du témoin et ne couvre que 40 à 111 eid : borne ≤ +485 paquets. Aucune règle générale ; la part (iii') de P6 tombe avec (a). | `P6-ferme+suivant` 40 liaisons, +474 / −4, témoin 5 contre 39 (> 10 paquets) ; `P6-ferme` 111 liaisons, +344 / −26, témoin 47 contre 111 |

Effet sur les lots (proposé, non appliqué : ce document ne modifie pas le plan) :
- **L1a** : la condition par film du §3 lève le blocage « pertes saines sur HI_1_8_0 à HI_1_11_0 »
  (D-46) ; elle exige de lire le bloc de type 1 en production (déjà au périmètre de L1) et deux passes
  (ou la variante causale, §3.4).
- **L1c / R-P6** : ne pas ouvrir de lot. La région (iii') est à retirer de toute recherche de
  naissance (§1) ; ce qui reste de P6 en régions (i) et (ii) suit L1a / L1b sans limite de distance.
- **L0** : nouvel invariant à instrumenter, « sortie de vue B par rejet ⇒ paquet non fermé »
  (T3-C1, et §2 ici) ; il requalifie 4 598 fermetures, dont 1 008 jugées saines aujourd'hui.

## 1. R-L1 (a) — la région (iii') : l'en-tête rejeté était-il un DELTA ?

### 1.1 Ce que l'écrivain fait (lu, Ghidra, décompilations dans `r_nais_ghidra/`)

| Fonction | Ce qu'elle fait (lu) |
|---|---|
| `FUN_142f2cc78` (écrivain de la vue B, tick) | Si la session de film est inactive (`*(DAT_144e61d78+0xd0) == 0` ou `*(DAT_144e61d80+0x28bec) == 0`) : seulement `FUN_14076c75c` (le terminateur). Sinon : liste des requêtes par `vtable[0x10]` ; budget `iVar5 = 0x48000 − vtable[0x30]()` ; si `DAT_145121140 == 1` et plus de `0x200` requêtes, la liste est coupée à `0x200` ; `vtable[0x28]` ; puis, pour chaque requête dans l'ordre de la liste : `iVar6 = vtable[0x18](req)` (bits écrits), `iVar5 −= iVar6`, et **`if (iVar5 < 0x1000) break` APRÈS l'écriture** (le dernier record écrit est gardé). Ensuite `vtable[0x20]` (concaténation), `vtable[0x58](seq, 1, 0)` sur chaque requête écrite (acquittement immédiat), `vtable[0x38]` (terminateur). |
| `FUN_142f24a78` (`vtable[0x18]`) | Sous-écrivain choisi par le genre : `param_1 + 0x1afb8 + genre * 0xd8` ; marge passée à `FUN_142f2cee0` = `capacité*8 − budget restant − bits déjà écrits` ; rend les bits écrits. |
| `FUN_142f2cee0` (un record) | Empile l'état du sous-écrivain (`+0x28..+0x44` vers `+0x50 + profondeur*0x20`, profondeur `+0x48` incrémentée) ; écrit par genre (`FUN_142f303bc` NEW, `FUN_142f304a8` DEL, `FUN_142f30610` DELTA) ; si l'écriture réussit ET tient dans le budget restant : dépile sans restaurer (engagement) puis effets de bord (NEW : `FUN_142f2f8f0`, état 3 ; DEL : liste + `FUN_1408f1358(.., 4)`) ; sinon **`FUN_14076a148(w, 1)`**. |
| `FUN_14076a148(w, 1)` (retour arrière) | Dépile et RESTAURE `+0x28..+0x44` depuis l'entrée empilée : le record disparaît en entier du sous-écrivain. Aucun effet de bord : l'entité garde son état (1 pour un NEW non engagé), donc son NEW est re-proposé au paquet suivant. Avec `param_2 == 0` : dépile sans restaurer. |
| `FUN_14076b9c8` (`vtable[0x20]`) | Vide l'accumulateur 64 bits des trois sous-écrivains (`FUN_1406d6d94`, aucun bit ajouté au compte logique `+0x28`), puis `FUN_1406d5d14(paquet, +0x1b090)` (genre 1, NEW), `(+0x1b240)` (genre 3, DELTA), `(+0x1b168)` (genre 2, DEL). |
| `FUN_1406d5d14` | Copie exactement `source+0x28` bits, sans préfixe ni bourrage. |
| `FUN_14076c75c` (terminateur) | `[0xf0c3a57e sur 32 bits si FUN_14076cea8()]`, puis 1 bit 0, puis 2 bits 0 : `000`. |

Conséquences (**établies par lecture**) :
1. Dans un paquet, TOUS les NEW précèdent TOUS les DELTA, qui précèdent TOUS les DEL : la
   concaténation se fait après coup, par genre. Aucun retour arrière, aucune coupure de budget, aucun
   plafond ne peut faire apparaître un NEW après un DELTA.
2. Un groupe peut avoir des TROUS (un record retiré par retour arrière, la suite de la liste écrite si
   le budget restant dépasse `0x1000`), jamais d'entrelacement.
3. Une entité dont le NEW est retiré n'a ni DELTA ni DEL dans ce paquet, ni dans les suivants tant
   que son NEW n'est pas engagé.

Donc : si l'en-tête rejeté X d'un paquet Q est un VRAI en-tête de DELTA (lu au bon bit), un en-tête
NEW trouvé derrière lui dans Q n'est pas un record de l'écrivain. Il ne peut l'être que si X est une
lecture à une position fausse alors que le groupe NEW n'était pas terminé.

### 1.2 Mesures (sondes `r_nais_occ_research_test.go`, `r_nais_pied_research_test.go`)

La passe 1 refait la sonde M1 à l'identique (mêmes premiers rejets, témoins, recherche, régions,
meilleure occurrence) ; contrôle : les **1 289** liaisons (iii') après rejet et leur oracle
(+363 / −885, 758 perdus sains) sont retrouvés à l'unité (MESURES_BIS_1 §3-4).

**Ce que la vue B de Q avait lu avant X** (`a_liaisons_lu`) : `NEW* DELTA+` **1 238 (96,0 %)**, rien
lu 20, NEW seulement 16, DEL lu 15.

**Classe de l'eid de X au bloc de type 1** (`a_liaisons_classe_x`) : naissance non lue 938, réalloué
(naissances de génération 0, D-43) 100, vivant au bloc 53, NEW lu dans le chunk 3 → **1 094 entités
attestées (84,9 %)** ; aucune allocation 162, non mesurable 31, libéré 2.

Liaisons dont X est une entité attestée ET dont la vue B avait déjà quitté le groupe NEW (DELTA lu
avant X, ou X premier DELTA) : **1 094 / 1 289 (84,9 %)**. Pour elles, la grammaire de l'écrivain
interdit le NEW trouvé (1.1). Les 195 autres (X sans allocation ou non mesurable) sont compatibles
avec un X lu à une position fausse.

**Test d'alignement au niveau de l'occurrence, avec témoin** (passe 2 : Q relu depuis l'occurrence,
sur le monde d'AVANT Q, puis restauré) — occurrences de la région (iii') à ≤ 3 paquets :

| | ferme, sain | ferme, contredit | ne ferme pas |
|---|---|---|---|
| eid rejeté (2 143 occurrences) | 5 (0,23 %) | 28 | 2 110 |
| témoin (177 occurrences) | 2 (1,1 %) | 0 | 175 |

Le taux de fermeture saine des occurrences de l'eid rejeté n'est pas au-dessus du hasard : ces
positions ne sont pas des débuts de record. Sur les 1 289 liaisons : 4 ferment saines, 21 contredites,
1 264 ne ferment pas.

**Archétype** (`a_liaisons_pont`) : là où le pont du bloc suivant résout l'archétype de l'eid
(33 liaisons), le `R(6)` lu derrière X le contredit **33 fois sur 33**.

**Position** (`a_occurrences_iii_rejet_decalage`) : les occurrences de l'eid rejeté sont à 16-255 bits
derrière la fin de l'en-tête X pour 1 382 / 2 143 (64,5 %) ; celles du témoin à ≥ 256 bits pour
149 / 177 (84,2 %). L'excès sur le hasard (1 289 liaisons contre 90 témoins, MESURES_BIS_1 §6) est donc
concentré dans le corps de X et des records qui le suivent. **Supposé** (non vérifié) : un champ de
composant qui porte une référence d'entité (poignée slot + tête) vers l'eid nouvellement né, précédé
par hasard de `0 01`.

**Oracles** (passe 3, juge) :

| Marche | Liaisons | Gagnés / perdus (dont sains) | Δ paquets sains | Δ utiles sains |
|---|---|---|---|---|
| `a-(iii')-rejet` (contrôle) | 1 289 | +363 / −885 (758) | −530 | −14 494 |
| `a-(iii')-rejet-Q-ferme-sain` | 4 | +16 / 0 | +15 | +116 |
| `a-(iii')-rejet-Q-non-ferme-sain` | 1 285 | +347 / −885 (758) | −545 | −14 610 |

**Réponse (a)** : oui, l'en-tête rejeté est un DELTA réel dans au moins 84,9 % des cas (attesté par le
bloc de type 1, lu après un DELTA propre) ; derrière lui, l'écrivain n'écrit plus de NEW. Les
1 289 « NEW trouvés derrière un rejet antérieur » (D-3) ne sont pas des naissances : leur position
n'ouvre aucune lecture qui ferme (taux du témoin), leur archétype contredit le pont 33 fois sur 33.
D-3 est tranché ; la borne négative de (iii') (−522, D-45) est expliquée. Les 4 liaisons dont
l'occurrence ferme le paquet (+15 paquets sains) sont trop peu pour une règle.

## 2. R-L1 (b) — les 4 598 paquets fermés après un rejet (D-2, D-56)

Sonde `r_nais_occ_research_test.go` (une ligne par paquet : `r_nais_pied.tsv`, queue de bits
comprise) et `r_nais_pied2_research_test.go` (`r_nais_pied_resync.tsv`).

### 2.1 Ce qu'ils sont (mesuré)

- **4 598** paquets (contrôle : CARTE v2 §4) ; HI_1_10_0 3 756, HI_1_13_0 590, HI_1_8_0 98,
  HI_1_11_0 61, HI_1_9_0 53, version-33 23, version-31 8, HI_1_4_1 7, HI_1_12_0 2.
- **Juge** : 3 590 contredits (**78,1 %**, factices au sens D2) ; 1 008 sains (HI_1_10_0 790,
  HI_1_13_0 177).
- **Piste D-56 « DEL ti=0 suivi d'un en-tête nul »** : eid rejeté nul **0 / 4 598** ; dernier record
  lu un DEL dans 441 cas (9,6 %), et le mot de 32 bits de ces DEL est non nul **441 / 441**, alors que
  l'écrivain l'écrit à 0 hors archétype `0x10` (T1 §1.3, lu) : ces DEL sont eux-mêmes douteux.
  Dernier record NEW 3 925 (85,4 %), DELTA 232. La piste ne décrit pas cette population. (Les 658
  paquets HORS CADRE « `DEL ti=0` » de BIS_3 §3.4 sont une autre population, non remesurée ici.)
- Classe de l'eid rejeté : aucune allocation 4 306 (93,7 %) ; tête 0 dans 4 237 (92,2 %).
- Toute la vue B n'est jamais vide : chacun des 4 598 a lu au moins un record avant le rejet.
- Reste du payload derrière l'en-tête rejeté : > 64 bits 3 369, 9-64 bits 581, ≤ 8 bits 648. Sur
  TOUS les paquets à rejet hors datum : fermés 648 / 660 quand le reste est ≤ 8 bits (fin de payload,
  zéros de bourrage), 3 369 / 257 939 (1,3 %) quand il dépasse 64 bits.

Les queues de bits montrent, AVANT l'en-tête rejeté, des motifs d'entrées de vue C
(`1 00 0 iiiii 1 …`, index croissants), par exemple `1c4c63c2` :
`…0010001010111011|1111001111100000|1000101101101111…`.

### 2.2 Le test du début antérieur, avec témoin (mesuré)

Pour chaque paquet fermé après rejet : le bit t le plus proche AVANT l'en-tête rejeté, précédé de
`000` (le terminateur de la vue B), d'où la vue C ferme aussi le paquet. Témoin : la même recherche,
en amont du VRAI terminateur, sur un paquet sur seize des paquets fermés après terminateur.

| Population | Début antérieur qui ferme | dont entrées de la marche en suffixe |
|---|---|---|
| fermés après rejet, sains (1 008) | **976 (96,8 %)** | 924 |
| fermés après rejet, contredits (3 590) | **3 426 (95,4 %)** | 3 179 |
| témoin : fermés après terminateur, sains (34 370) | 3 767 (11,0 %) | 3 338 |
| témoin : fermés après terminateur, contredits (610) | 362 (59,3 %) | 352 |

Forme dominante (3 848 paquets) : t est 0 à 15 bits avant l'en-tête rejeté, et la vue C lue depuis t
porte UNE entrée de plus que celle de la marche, les suivantes identiques.

Auto-synchronisation de la vue C (témoin du critère « fermé au bit près ») : sur les paquets fermés
après terminateur et sains, la vue C relue depuis un bit décalé de k ∈ [1, 24] ferme encore le paquet
pour au moins un k dans **5 343 / 34 370 (15,5 %)**.

### 2.3 Réponse (b)

Ce ne sont pas des « pieds de trame » d'une vue vide : la vue B a lu au moins un record, et dans
95,7 % des cas (contre 11,0 % au témoin) le vrai terminateur `000` est AVANT l'en-tête rejeté. Le
dernier record lu (un NEW dans 85 % des cas) a donc débordé sur le terminateur et la première entrée
de la vue C ; l'« en-tête rejeté » est une entrée de vue C ; la vue C lue derrière lui retombe sur les
entrées suivantes (auto-synchronisation, 15,5 % au témoin) et ferme le paquet. C'est cohérent avec
T3-C1 (établi : un rejet n'est jamais légitime dans un film). **Ces 4 598 fermetures sont factices**,
y compris les 1 008 que le juge des trois invariants laisse passer : il faut un quatrième invariant
« sortie de vue B par rejet ⇒ non fermé » (lot L0). Conséquence pour la mesure de la campagne : le
paquet fermé « sain » de la référence surestime de 1 008 paquets (0,36 % des 276 316).

## 3. R-L1 (d) — la condition par film qui sépare les chaînes de tête justes des fausses

### 3.1 Le bit de configuration et D-8 (lu, mesuré)

Lu : quand `DAT_144706104 == 0`, `FUN_142f2fc08` rend un pool unique `[0, DAT_144706100)` (base 0,
pool −1 → curseur du pool 0 dans `FUN_142f2f0cc`) ; `FUN_142f2f634` le met à 0 quand un pool borné
est plein. Le paquet delta écrit ce bit en tête. Mesuré (`r_nais_cadre.tsv`) : bit à 1 dans
**629 142 / 629 142** paquets delta des 20 films. Le modèle « pool unique » prédit ≤ 2,3 % des NEW
lus (rang < 64, tête prédite) sur tous les builds. **D-8 « `DAT_144706104` à 0 sur ces films » est
réfuté** ; la faible prédiction sur les vieux builds vient d'ailleurs (non tranché).

### 3.2 La condition proposée

**Score d'allocateur d'un film** = part des NEW que la marche de référence lit proprement (traversée
sans désynchronisation, non refusés), tous paquets, dont le (slot, tête) est prédit par l'allocateur
à cinq pools (rang < 64 dans la suite next-fit depuis les curseurs du bloc de type 1, tête
`(gen+1)&3`). Le localisateur élargi `tete-bloc+inv` (MESURES_BIS_1 §2) n'est activé que si le score
du film est **≥ 50 %** (et au moins 30 NEW). Aucune branche sur le build : le score est une propriété
lue du film.

Scores mesurés (`r_nais_scores.tsv`) : HI_1_13_0 54,7 % (`0797ce72`) à 93,4 % ; HI_1_12_0 86,1 % ;
version-31 57,3 % ; HI_1_11_0 31,9 % ; HI_1_10_0 5,1 % à 30,1 % ; HI_1_9_0 25,6 % ; HI_1_8_0 14,1 % ;
HI_1_4_1 36,9 % ; version-33 37,7 %. Tout seuil dans ]37,7 % ; 54,7 %] donne le même partage.
**Réserve** : le seuil est choisi sur ces 20 films (dans l'échantillon) ; un 21e film peut tomber
entre les deux.

Mesuré et ÉCARTÉ : le même score restreint aux paquets SANS liste d'événements (pour l'isoler du
localisateur) ne sépare pas les films (`1c4c63c2` 65,5 %, `0797ce72` 36,9 %) ; ses trois variantes
(`…-hors-evenements` dans `r_nais_variantes.tsv`) perdent sur `1c4c63c2`, `084a804d`, `e5adf7b2`,
`111fa685`, `60ae07c4`.

### 3.3 Résultat (mesuré, juge, 20 films)

Par build (Δ contre la référence ; utiles sains sur dénominateur FIXE) :

| Build | `tete-bloc+inv` Δ sains ; Δ utiles sains | `|film` (≥ 50 %, deux passes) | `|cumul` (causal) | `|chunk` |
|---|---|---|---|---|
| HI_1_13_0 | +14 380 ; +153 699 | +14 380 ; +153 699 | +12 844 ; +136 776 | +12 947 ; +137 078 |
| HI_1_12_0 | +684 ; +6 165 | +684 ; +6 165 | +530 ; +4 518 | +684 ; +6 165 |
| HI_1_11_0 | +17 ; **−1 904** | 0 ; 0 | 0 ; 0 | +14 ; +22 |
| HI_1_10_0 | +753 ; **−21 094** | 0 ; 0 | 0 ; 0 | +38 ; +477 |
| HI_1_9_0 | **−13 ; −1 404** | 0 ; 0 | +2 ; 0 | +5 ; +8 |
| HI_1_8_0 | **−129 ; −1 953** | 0 ; 0 | 0 ; 0 | **−59 ; −609** |
| HI_1_4_1, v31, v33 | +22 ; +2 | +6 ; 0 | +5 ; 0 | +8 ; 0 |
| **corpus** | +15 714 ; +133 511 | **+15 070 ; +159 864** | +13 381 ; +141 294 | +13 637 ; +143 141 |

**Gate par film** (`r_nais_variantes.tsv`) : `|film` — aucun film en baisse, ni en paquets sains ni en
utiles sains (minimum 0 ; onze films en hausse) ; `|cumul` — aucun film en baisse ; `|chunk` —
`60ae07c4` en baisse (−59 ; −609) : **écarté**. `tete-bloc+inv` sans condition : six films en baisse.

Pourcentages d'utiles sains, `|film` contre référence : HI_1_13_0 **75,93 %** contre 70,54 % sur le
dénominateur fixe (2 850 786 = maximum par film des records utiles lus sur toutes les marches de la
campagne, surcouche `ti=3` comprise, `r_nais_denominateur_fixe.tsv`), 84,70 % contre 80,57 % sur le
dénominateur variable (utiles fermés / utiles lus) ; corpus 41,99 % contre 39,54 % (fixe), 45,57 %
contre 43,42 % (variable). Brut (paquets fermés) : corpus 298 315 contre 284 704 (dont sains 291 386 contre 276 316).

Paquets perdus un à un : `|film` perd 1 706 paquets fermés, dont 467 sains en référence (tous sur
HI_1_13_0 : `4f77afc1` 307, `396cfc92` 114, `51ebbc0f` 46), chacun compensé dans son film (net
positif). Gains contredits : 1 sur 15 317.

### 3.4 Réponse (d)

La condition par film « score d'allocateur ≥ 50 % » tient le gate par film sur les 20 films. Deux
formes : deux passes (score du film entier, gain maximal) ou causale (score cumulé des chunks
précédents, au moins 30 NEW, −1 689 paquets sains de gain sur le corpus, aucune seconde passe). Coût
de production : le bloc de type 1 lu en production (déjà au périmètre de L1, gate 4 du §6.0) ; la
forme à deux passes double la marche des paquets à événements. Statut : **établi sur le corpus**, avec
la réserve du seuil choisi dans l'échantillon.

## 4. R-P6 — localiser une naissance à plus de 3 paquets

### 4.1 Le filtre d'occurrence

« **ferme** » : l'occurrence est en région non lue ET le paquet Q relu depuis elle (sur le monde
d'avant Q, copie restaurée) se ferme sans invariant contredit. C'est une propriété de l'OCCURRENCE
(une position), pas de l'eid, et elle a un sens de décodeur : c'est la règle de `debutParFermeture`
étendue à toute région non lue. Contrôle de la machinerie : `P6-propre+alloc+suivant` rend 761
liaisons, +3 227 / −578, témoin 94 / 1 et 705 / 57 : MESURES_BIS_3 §4 à l'unité.

### 4.2 Témoin (eid rejetés NON liés et leurs témoins ayant ≥ 1 occurrence qui passe ; 11 801 cibles avec témoin)

| Filtre | 4-10 paquets : rejetés / témoins | > 10 paquets : rejetés / témoins |
|---|---|---|
| `propre+alloc+suivant` (contrôle, filtre d'EID) | 94 / 1 | 705 / 57 |
| `ferme` | 5 / 2 | 106 / 45 |
| `ferme+suivant` | 1 / 0 | 39 / 5 |
| `ferme+alloc` | 5 / 0 | 62 / 6 |
| `ferme`, régions (i)(ii) | 2 / 2 | 30 / 18 |
| `ferme`, région (iii') | 4 / 1 | 82 / 35 |

### 4.3 Bornes (oracles, juge)

| Oracle | Liaisons | Gagnés / perdus (sains) | Δ sains | Δ utiles sains |
|---|---|---|---|---|
| `P6-propre+alloc+suivant` (contrôle) | 761 | +3 227 / −578 (559) | +2 589 | +38 253 |
| `P6-ferme` | 111 | +344 / −26 (16) | +313 | +941 |
| `P6-ferme+suivant` | 40 | +474 / −4 (2) | +466 | +1 280 |
| `P6-ferme+alloc` | 67 | +343 / −14 (14) | +314 | +941 |
| `P6-ferme-(i)(ii)` | 32 | +485 / −3 (2) | +481 | +1 365 |
| `P6-ferme-(iii')` | 86 | +108 / −23 (14) | +80 | +634 |
| `oracle-(i)+(ii)` (rappel BIS_1) | 3 281 | +29 755 / −1 587 | +28 027 | +357 028 |
| `oracle-(i)+(ii)` + `P6-ferme` | 3 392 | +30 119 / −1 599 | +28 369 | +358 115 |

### 4.4 Réponse R-P6

Le seul filtre d'occurrence mesuré qui dépasse nettement le hasard est `ferme+suivant` (39 contre 5
au-delà de 10 paquets, soit ≈ 13 % de faux positifs estimés par le témoin) ; il couvre 40 eid et
borne le gain à +466 paquets sains. `ferme` seul (111 contre 47) n'est pas valide. Aucune règle ne
localise la population P6 en général : **partiel**. Deux faits la réduisent : (1) sa part en région
(iii') (BIS_3 §4.1 : 3 584 eid, 40 893 paquets) relève du §1 — derrière un vrai DELTA, ce ne sont pas
des NEW ; (2) sa part en régions (i) et (ii) (2 818 eid, 18 805 paquets) suivra L1a / L1b, qui lisent
une région, pas une distance. Proposition : pas de lot P6 ; la borne `propre+alloc+suivant`
(+3 227 / −578) repose sur un filtre d'EID et ne doit pas servir de borne de correctif.

## 5. Protocole, sondes, contrôles

Sondes (toutes `//go:build research` en ligne 1, < 500 lignes, sous
`apps/go-api/internal/games/halo_infinite/film/internal/grammar/`) :

| Fichier | Lignes | Rôle |
|---|---|---|
| `r_nais_marche_research_test.go` | 146 | marche de la campagne avec crochet d'avant paquet ; juge compact `rnJuge` |
| `r_nais_occ_research_test.go` | 299 | passe 1 : M1 refaite en gardant chaque occurrence, paquets à rejet, score d'allocateur |
| `r_nais_pied_research_test.go` | 185 | (b) lignes par paquet ; (a) descripteurs |
| `r_nais_research_test.go` | 321 | pilote `TestRNaisMesures` : passes 1-2, oracles (a), R-P6, (d) |
| `r_nais_pied2_research_test.go` | 165 | (b) `TestRNaisPiedResync` : début antérieur, auto-synchronisation |
| `r_nais_cadre_research_test.go` | 175 | (d) `TestRNaisCadre` : bit de configuration, deux modèles d'allocateur |

Commandes (depuis `apps/go-api`, `GOCACHE` dédié `gocache-r_nais`, une commande `go` à la fois,
`-count=1`, un film à la fois, sentinelle `filmproc` 4 Gio, sorties dans le scratchpad puis copiées
dans `r_nais_tsv/`) :

```
CAMPAGNE_RACINE=<LevelUp>/data/cache/film_chunks CAMPAGNE_FILMS=<20 ids> CAMPAGNE_SORTIE=<scratchpad> \
  go test -tags=research -count=1 -timeout 240m -run '^TestRNaisMesures$' ./internal/games/halo_infinite/film/internal/grammar/
# (d) seul, score « tous NEW » : RNAIS_SEULEMENT=d (même commande)
go test -tags=research -count=1 -timeout 120m -run '^TestRNaisPiedResync$' ./internal/.../grammar/
go test -tags=research -count=1 -timeout 120m -run '^TestRNaisCadre$' ./internal/.../grammar/
```

Durées : `TestRNaisMesures` 1 595 s (17 marches par film, pics 91 à 400 Mio, 321 894 occurrences
relevées, celles en région non lue jugées en passe 2) ; mode `d` 481 s ; `TestRNaisPiedResync` 112 s ; `TestRNaisCadre` 119 s.

Contrôles (tous tenus) : référence = carte v2 (284 704 fermés, 2 588 167 / 5 961 028 utiles,
264 757 hors cadre) ; passe 2 = référence (vérifié par le test) ; `a-(iii')-rejet` = BIS_1 ;
`oracle-(i)+(ii)` = BIS_1 (3 281 liaisons, +29 755 / −1 587) ; `tete-bloc+inv` = BIS_1 (+16 407 /
−8 925, 1 921 sains perdus) ; `P6-propre+alloc+suivant` = BIS_3 ; `reference` et `tete-bloc+inv`
identiques à l'octet entre les deux exécutions de `TestRNaisMesures` (40 lignes), ainsi que les
tables (b).

TSV (`r_nais_tsv/`) : `r_nais_variantes.tsv` (une ligne par film et par marche ; les variantes
« `-hors-evenements` » viennent de la première exécution), `r_nais_agg_variantes.tsv` (par build et
corpus, deux dénominateurs), `r_nais_denominateur_fixe.tsv`, `r_nais_tables.tsv` (tables `a_*`,
`b_*`, `p6_temoin` ; les tables `a_liaisons_decalage` et `a_liaisons_pont` sont dans l'exécution
`RNAIS_SEULEMENT=d`, reprises ci-dessus), `r_nais_pied.tsv` (4 598 lignes), `r_nais_pied_resync.tsv`,
`r_nais_scores.tsv`, `r_nais_cadre.tsv`. Ghidra : `r_nais_ghidra/r_nais_dec_<adresse>.txt`
(12 décompilations).

## 6. Limites

- « Sain » ne prouve pas une lecture juste ; le §2 en donne 1 008 contre-exemples.
- (a) : l'hypothèse « référence d'entité dans un corps » n'est pas vérifiée (le corps de X ne se
  décode pas sans son archétype).
- (d) : seuil choisi sur le corpus même ; les pertes saines individuelles (467) sont compensées dans
  leur film, pas absentes ; la variante causale exige 30 NEW avant d'activer.
- R-P6 : le filtre `ferme` est jugé sur le monde d'avant Q de la marche de RÉFÉRENCE, pas sur le monde
  d'une marche corrigée.
- La population D-56 hors cadre (`DEL ti=0`, 658 paquets) n'est pas remesurée.

## 7. Découvertes (consignées, non traitées)

- **N-1** `go test ./internal/archlint/` est ROUGE sur la tête `fe18bf67c` par le calendrier :
  `TestNoExpiredTODO`, `internal/api/handlers/json_huma_coverage_test.go:34`, `TODO(expiry:2026-10-01)`
  échu (fichier inchangé depuis le 2026-07-07, hors campagne). Tous les autres tests d'`archlint`
  passent (`-skip '^TestNoExpiredTODO$'` : ok). À re-dater ou traiter par l'utilisateur.
- **N-2** La vue C s'auto-synchronise : 15,5 % des paquets sains fermés après terminateur se ferment
  encore depuis un départ décalé de 1 à 24 bits. « Fermé au bit près » est un témoin plus faible que
  supposé (D2, L0).
- **N-3** Le non-retenu T3-C2 (« ne pas lire la vue C après un rejet », −4 598) ne perd que des
  fermetures factices (§2) ; sous D2 son bilan change (0 gain, 0 perte saine). À revoir avec L0.
- **N-4** Sur `1c4c63c2`, 65,5 % des NEW propres des paquets SANS liste d'événements sont prédits par
  l'allocateur, contre 5,1 % de tous ses NEW propres : la plupart des NEW lus dans ses paquets à
  événements ne sont pas prédits (lectures fausses probables). Non étudié.
- **N-5** Le mot de 32 bits des DEL lus en dernier avant un rejet fermé est non nul 441 / 441 (écrivain :
  0 hors archétype `0x10`) : un DEL non nul hors archétype `0x10` serait un invariant de l'écrivain
  de plus pour le juge (non mesuré sur les autres DEL).
- **N-6** D-8 réfuté (bit de configuration à 1 partout) : la cause de la faible prédiction de
  l'allocateur avant HI_1_12_0 reste ouverte (table de pools d'un autre build, D6).
