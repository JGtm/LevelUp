# Revue 6.R de la vague 6 — ronde de corrections 1

> 2026-09-11 · worktree `LevelUp-wt-couverture-objectifs`, branche `wt/couverture-objectifs`
> (partie de `feat/v75` = `bcee320a1`) · relecteur indépendant : 5 constats, 15 conditions tenues.
> Cette ronde corrige EXACTEMENT les 5 constats. Toute autre découverte est consignée au §8 et
> NON traitée.

## 1. Bilan en une page

| # | gravité | sujet | statut | commit |
|---|---|---|---|---|
| C1 | **P0** | `homed` n'est jamais démenti — le drapeau publié à sa base pendant qu'il gît au sol | **corrigé** | `3b2cca257` |
| C2 | P1 | `named_series.go` : doc inversée (filtre d'enregistrement dit absent, il existe) | **corrigé** | `a72190f8a` |
| C3 | P2 | branchement du filtre de domaine non couvert par un test | **corrigé** | `c30b6661d` |
| C4 | P2 | `useReplayGroundWeapons.ts` : doc inversée sur `ammoLine` | **corrigé** | `425a43f2c` |
| C5 | P2 | `AmmoRead` sans `omitempty`, et `required` au contrat | **corrigé** | `e96c1e69f` |
| réserve | — | dénominateur du verdict DRAPEAU NEUTRE après le resserrement de 6.13 | **instruite** | `bd0113849` |

**Gate C1, sur les 12 films CTF cuits hors ligne** : ratio parc **1,028 → 1,028**,
`b8a44fe8` **1,037 → 1,037**, **aucun joueur ne bouge d'un centième de seconde**, actions et
toutes les autres sorties de l'instrument **identiques à l'octet**.
**Effectif du défaut : 3 portages** comptés à la fois dans `closedByHome` et `closedByObject`
avant, **0 après** — et **46,3 s de drapeau dessiné à sa base alors qu'il gisait au sol** rendues
au sol, à la bonne position.

**Films à recuire : 9 des 12** (`16ea3668` `4ecdf3e7` `58864b3c` `7fce3219` `8bc6074f`
`a0c36016` `b8a44fe8` `f8efc5ca` `fb1a1a72`). `bc60b4d9`, `bf5ced1b` et `cde26226` sont
identiques à l'octet. Aucune montée de `SchemaVersion`.

---

## 2. Protocole de mesure, et son calibrage

Deux parcs de **12 films CTF** cuits HORS LIGNE (`cmd/replay-build`, `LEVELUP_REPO_ROOT` vers un
scratchpad, chunks du worktree partagé lus en LECTURE SEULE). **Aucune base DuckDB ouverte,
aucune commande `levelup`, aucun artefact du parc de production touché.**

| parc | binaire | rôle |
|---|---|---|
| `parcAvant` | HEAD `bcee320a1` | la référence AVANT |
| `parcApres` | HEAD + les correctifs de cette ronde | la référence APRÈS |

**Calibrage, trois fois :**

1. **`parcAvant` reproduit la référence 6.13 À L'IDENTIQUE** — les 12 lignes CTF de
   `replay2d/registre_film/vague6_613_bilan_portages.tsv` (secondes publiées et ratio) ressortent
   à la décimale : `16ea3668` 135,0 / 1,031 … `fb1a1a72` 272,7 / 1,047, TOTAL 2 447,3 / **1,028**.
2. **La cuisson est reproductible** : deux films recuits avec le même binaire sont identiques à
   l'octet à ceux du parc (`16ea3668`, `b8a44fe8`).
3. **L'instrument de recherche n'a rien changé à la sortie** : le binaire instrumenté et le
   binaire final rendent les **12 artefacts identiques à l'octet**.

**Instrument de recherche, SUPPRIMÉ avant livraison** (recette pour le rejouer) :
`internal/analysis/replay/zz6r_instrument.go`, ~70 lignes, sous garde `REV6R_DUMP=<fichier>`,
appelé depuis `flagCloseAt` (à chaque reprise d'un portage par un fermoir plus précoce, il note
le couple `ancien -> neuf`) et depuis `buildFlagCarries` (ventilation des fermoirs AVANT le filtre
de position). C'est lui qui rend les chiffres du §3.3.

**Instrument de mesure** : `.ai/V7.5/outillage/couverture_objectifs`, rejoué sur les deux parcs
contre l'oracle `replay2d/registre_film` (méthode : `RAPPORT_FILM_B8A44FE8_2026-09-11.md` §9).
Sorties versionnées : `replay2d/registre_film/vague6_6R_*`.

---

## 3. C1 — P0 — le plus petit fermoir gagne, et il EFFACE celui qu'il remplace

### 3.1 Le défaut, sur pièces

`homed` n'avait qu'**une seule écriture** dans tout le paquet : celle de `closeByHomecoming`
(`flag_carries_home.go`). Les fermoirs qui tournent APRÈS lui — `closeByCarrierKills`
(`flag_carries.go:350`) et `closeByFreeLives` (`flag_objects.go:337`) — écrivaient `t1`, `closed`
et `captured` **sans jamais remettre `homed` à faux**. Or `endsHome()` (`flag_carries.go:189`)
vaut `captured || homed`.

Séquence ordinaire, et elle n'a rien d'exotique : prise en t0, **lâcher daté en D** par la vie
libre de l'objet née aux pieds du porteur (hors socle), le drapeau **reste au sol**, puis il
**rentre seul en H > D**. `closeByHomecoming` passe d'abord (`t1 = H`, `homed = true`),
`closeByFreeLives` passe ensuite (`t1 = D`, `captured = false`) et `homed` **reste vrai**.

Quatre conséquences, toutes fausses :

| ce qui s'en sert | ce qu'il faisait |
|---|---|
| `applyFlagLifeEvent` (`flag_carries_lives.go:241-248`) | publiait `FlagStateHome` **à la position du socle** dès `frame(D)+1`, alors que l'objet gisait au sol jusqu'à H |
| `flagGround.poser` (`flag_assign.go:180`) | retirait le drapeau **du sol ET du jeu**, faussant l'attribution des prises suivantes |
| `repositionFlagDrops` (`flag_objects.go`) | sautait le repositionnement du lâcher |
| la couverture | comptait le MÊME portage dans `closedByHome` **ET** `closedByObject` |

### 3.2 Le correctif : un seul principe, un seul endroit

`internal/analysis/replay/flag_carries_close.go` (neuf) porte tout le principe.

- **`flagCloseAt(r, at, by)`** est le SEUL endroit qui ferme un portage après le bornage. Il
  n'accepte qu'un instant **strictement intérieur à `]t0, t1[`** — donc toujours plus petit que
  la borne en place : « le plus petit gagne » devient **structurel** plutôt que comparé — et il
  **RÉÉCRIT TOUT** l'état de fin : `captured` et `homed` valent ce que dit LE fermoir en vigueur,
  jamais la mémoire d'un fermoir périmé.
- **Les compteurs se DÉRIVENT, ils ne s'incrémentent plus.** Chaque portage porte son fermoir
  (`flagCarryRaw.closedBy`, type `flagCloser`) ; `tallyFlagCarries` compte les portages **par
  fermoir en vigueur**. Un portage ne PEUT plus peupler deux `closedBy*` : c'est une propriété de
  la représentation, pas une précaution de comptage.
- **L'invariant est posé dans `Balanced()`** (`document_objectives_live.go`) :
  `closedByHandoff + closedByReturn + closedByHome + closedByObject <= closed`. L'inégalité est
  LARGE et non une égalité : un portage fermé par la capture, la mort, la reprise du même slot ou
  la chute créditée ne peuple aucun de ces quatre compteurs.

Les quatre chaînes (`closeByHandoff`, `closeByHomecoming`, `closeByCarrierKills`,
`closeByFreeLives`) passent désormais toutes par `flagCloseAt`, et `closeByHandoff` /
`closeByHomecoming` / `closeByFreeLives` ne rendent plus de compte de fermeture.

**Une exception, MESURÉE et non supposée** : `flag_carriers_killed` reste une **BORNE** et non un
fermoir (`flagCloser.ferme()`). Il n'avait jamais posé `closed`, et le lui faire poser a été
mesuré : sur `b8a44fe8`, le seul span `carried_open` du parc tomberait de **25,0 s à 7,3 s** pour
un oracle de **17,2 s** — l'écart passerait de **+7,8 s à −9,9 s**. L'événement ne date donc PAS
la chute de ce porteur-là. Le désaccord qui subsiste entre `t1` et l'état publié est **consigné
au §8, non traité** : le trancher demande de qualifier ce que `flag_carriers_killed` date vraiment
quand plusieurs portages se succèdent — un sujet de mesure, pas de refactorisation.

### 3.3 Le test rouge, et les mutations jouées

`TestUnLacherPLUSTOTDEMENTLaRentree` (`flag_carries_home_test.go`), fixture exactement celle du
constat : `flagHomeScan` + **une vie libre à (50,5 ; 50,5) à l'image 15** (aux pieds du porteur,
hors socle) **en plus de celle du socle à l'image 20**.

Sortie AVANT correctif — **rouge**, et le message nomme le défaut :

```
flag_carries_home_test.go:166: etats [home carried home], attendu [home carried dropped home]
```

Le test exige : `t1 = D` (image 15), état publié **`dropped` à la position du lâcher** entre D et
H puis **`home` à partir de H**, la position du lâcher à **(50,5 ; 50,5)** et non au socle,
`closedByObject = 1`, `closedByHome = 0`, `closedByReturn = 0`, et `Balanced()`.

**Mutations jouées (deux, réellement exécutées) :**

| mutation | effet observé |
|---|---|
| `r.homed = by == flagCloserObject \|\| …` dans `flagCloseAt` | rouge : `etats [home carried home], attendu [home carried dropped home]` |
| compter un portage `object` AUSSI dans `ClosedByHome` (`tallyFlagCarries`) | rouge **deux fois** : sur `closedByHome = 1` et sur **l'invariant de `Balanced()`** (« la somme des fermoirs dépasse les portages fermés ») |

La seconde prouve que l'invariant ajouté à `Balanced()` porte : il attrape seul le double compte.

### 3.4 Le gate, AVANT → APRÈS

**Ratios (instrument, 12 films CTF) :**

| film | AVANT | APRÈS | film | AVANT | APRÈS |
|---|---|---|---|---|---|
| `16ea3668` | 1,031 | 1,031 | `b8a44fe8` | **1,037** | **1,037** |
| `4ecdf3e7` | 0,893 | 0,893 | `bc60b4d9` | 1,030 | 1,030 |
| `58864b3c` | 1,022 | 1,022 | `bf5ced1b` | 1,024 | 1,024 |
| `7fce3219` | 1,055 | 1,055 | `cde26226` | 1,040 | 1,040 |
| `8bc6074f` | 1,020 | 1,020 | `f8efc5ca` | 1,013 | 1,013 |
| `a0c36016` | 1,026 | 1,026 | `fb1a1a72` | 1,047 | 1,047 |
| | | | **TOTAL** | **1,028** | **1,028** |

- ratio parc **1,028 ≤ 1,028** : aucune régression ;
- `b8a44fe8` **1,037 ≤ 1,037** ;
- **aucun joueur ne perd ni ne gagne 0,1 s** — la comparaison joueur par joueur des deux
  `vague6_couverture_portage.tsv` (80 lignes) rend **zéro écart**. C'est exactement ce que le
  contrat prévoit : un portage qui était `home` à tort devient `dropped`, **la durée du portage ne
  change pas** ;
- **actions inchangées** — `vague6_couverture_actions.tsv` (372 lignes) identique à l'octet, et
  avec lui `stats_publiees`, `identite`, `bornage`, `couverture_parc`, `objet_sans_position`,
  `deroulage`, `deroulage_bilan`.

**Effectif du défaut, mesuré par l'instrument de recherche** (couples `ancien -> neuf` relevés à
chaque reprise, sur les 12 films) :

| recoupement AVANT | portages | compteurs concernés |
|---|---|---|
| `home -> object` | **3** | `closedByHome` **ET** `closedByObject` — **c'est l'effectif du défaut nommé par C1** |
| `handoff -> object` | **18** | `closedByHandoff` **ET** `closedByObject` |
| `carrierKill -> object` | 17 | aucun (la chute créditée ne peuple aucun compteur) |
| **total des doubles comptes de compteurs publiés** | **21** | sur **523** portages fermés |

**APRÈS : 0**, et structurellement — un portage ne porte qu'un `closedBy`.

Totaux des compteurs, les 12 films :

| compteur | AVANT | APRÈS | lecture |
|---|---|---|---|
| `closedByObject` | 425 | 424 | −1 : un portage sans piste, écarté par le filtre de position |
| `closedByHandoff` | 18 | **0** | les 18 passages étaient tous précédés d'un lâcher daté, qui les reprend |
| `closedByHome` | 8 | **5** | −3 : l'effectif du défaut |
| `closedByReturn` | 0 | 0 | |
| somme | **451** | **429** | 451 > 523 ? non — mais 451 comptait 21 portages deux fois |
| `closed` | 523 | 523 | |

Les trois portages qui changent d'état publié sont ceux du défaut, et l'écart n'est pas
cosmétique :

| film | AVANT | APRÈS | drapeau rendu au sol |
|---|---|---|---|
| `16ea3668` | `home` au socle (12,50 ; −0,00) sur [2324 ; 2863] | `dropped` à (8,84 ; −1,50) sur [2324 ; 2456] puis `home` | **13,3 s** |
| `a0c36016` | `home` au socle (−24,30 ; 22,82) sur [1784 ; 3055] | `dropped` à (−18,65 ; 16,41) sur [1784 ; 1849] puis `home` | **6,6 s** |
| `fb1a1a72` | `home` au socle (−25,49 ; 6,75) sur [6312 ; 7146] | `dropped` à (19,37 ; 7,85) sur [6312 ; 6575] puis `home` | **26,4 s** |

**46,3 s** de drapeau dessiné à sa base alors qu'il gisait au sol — sur `fb1a1a72` à **44,9 m** de
l'endroit où il était rendu. `homeByObject` et `dropsRepositioned` montent de 1 sur chacun de ces
trois films : la rentrée qui n'avait plus rien à renvoyer (le portage la « consommait ») redevient
une vraie rentrée, et le lâcher redevient repositionnable.

**Les 9 autres artefacts qui changent ne changent QUE leur couverture** — vérifié clé par clé :
sur `4ecdf3e7`, `58864b3c`, `7fce3219`, `8bc6074f`, `b8a44fe8`, `f8efc5ca`, seule la clé
`coverage` diffère, `flagCarries` est identique. Les 3 autres (`bc60b4d9`, `bf5ced1b`,
`cde26226`) sont identiques à l'octet.

---

## 4. C2 — P1 — la doc des bornes de déroulage décrivait une absence comblée

`objectiveevents/named_series.go` affirmait, en en-tête : « Le filtre exact serait au niveau de
l'ENREGISTREMENT … Il n'est toujours pas là. » Il y est depuis le lot 6.11 : `statMaxCounter`
(`statborg.go:268`) + `statCountersInDomain` (`statborg.go:130-150`), branché dans
`scanFrameForRecords`. Anti-patron n° 9 (doc inversée).

L'en-tête réécrit dit maintenant que **les deux bornes sont calibrées l'une par rapport à
l'autre**, et qu'aucune ne couvre seule le phénomène :

- **le PAS** (`maxUnrollPerStep` = 16) coupe les gros déroulages et laisse passer les petits : le
  record fortuit de `fb1a1a72` (slot 24, t = 764 967, canaux à 2 415 919 104 et −30 456) portait
  un pas de **10** sur `comp 22 A` — il passe SOUS 16, et cette borne-ci ne peut rien en dire ;
- **l'ENREGISTREMENT** (`statMaxCounter` = 2^20) ne regarde pas le pas mais l'ORDRE DE GRANDEUR
  des canaux : pire valeur **saine** mesurée **102 934**, plus petite **aberrante**
  **2 415 919 104**, et le seuil se pose dans le vide qui les sépare. C'est lui, et lui seul, qui
  attrape le cas.

---

## 5. C3 — P2 — le BRANCHEMENT du filtre de domaine rougit désormais

`statborg_domaine_test.go` l'écrivait lui-même à sa ligne 49 : retirer
`!statCountersInDomain(comps)` de `scanFrameForRecords` **ne rougissait aucun test**. Le prédicat
était prouvé sous toutes ses faces ; son **branchement** ne l'était pas.

`statborg_domaine_branche_test.go` (neuf) monte un paquet FRAME **synthétique**, à la grammaire de
la production (bit marqueur, en-tête d'enregistrement slot 24, liste creuse d'un composant
`comp 22`, deux canaux à longueur variable, deux drapeaux conditionnels nuls), et ne fait varier
que la VALEUR du canal A :

- **témoin positif** — `A = 10` : l'enregistrement **ressort** du balayage ;
- **le point du constat** — `A = 2 000 000` (au-delà de `statMaxCounter`) : l'enregistrement
  **est absent** de la sortie.

**Mutation jouée** : la clause retirée de `scanFrameForRecords`, seul
`TestLeBalayageJETTELEnregistrementHorsDomaine` rougit — le reste du paquet reste vert :

```
--- FAIL: TestLeBalayageJETTELEnregistrementHorsDomaine (0.00s)
    le balayage rend un enregistrement dont le canal A vaut 2000000 : le filtre de
    domaine n'est plus branche sur scanFrameForRecords
```

L'aveu d'absence est retiré du commentaire de mutation de l'ancien fichier, qui renvoie au
nouveau.

---

## 6. C4 et C5 — les armes au sol

### 6.1 C4 — `ammoLine` niait la lecture EXACTE

Le commentaire du champ (`layers/useReplayGroundWeapons.ts`) disait que « le chiffre n'est PAS
celui du lâcher », alors que la branche `kind: 'exact'` (`model/groundWeaponAmmo.ts:79`, lot 6.10)
est exactement cela : le chargeur et la réserve lus **sur l'objet**, dans son record de création,
à l'instant du lâcher — publiés **sans âge et sans « ≈ »**. Anti-patron n° 9.

Le champ décrit maintenant les **deux natures** et la frontière qui les sépare : une lecture
**DATÉE** ne s'affiche **jamais** sans son âge ni sans « ≈ » (dernière lecture d'inventaire du
lâcheur, en retard de 9,0 s en médiane) ; une lecture **EXACTE** ne s'affiche **jamais** avec. La
frontière est portée par les trois strings de `i18n/i18nContract.ts` :
`groundWeaponAmmoExactFmt` d'un côté, `groundWeaponAmmoFmt` / `groundWeaponAmmoResFmt` de
l'autre — et `groundWeaponAmmoLine` choisit sur la NATURE de la lecture, jamais sur le goût de
l'appelant.

### 6.2 C5 — `AmmoRead` prend `omitempty` et sort de `required`

Seul champ neuf de la vague écrit sans `omitempty`. Sans lui, un artefact cuit **avant** le lot
6.10 — qui n'a pas la clé — est indistinguable d'un film où le décodeur n'a rien lu, qui l'écrit
à zéro. La clé absente et la clé à zéro disent la même chose.

- `omitempty` **des deux côtés** : `analysis/replay/document_ground_weapon_items.go:160` **et**
  `domain/replaydoc/coverage_world.go:64`. La parité l'exige : le troisième test de
  `service/replayview/parity_test.go` compare les deux documents **à l'octet** après
  sérialisation — deux formes divergentes le rougiraient. Il reste vert.
- **contrat régénéré** (`go run ./cmd/openapi-gen`) : diff d'**une seule ligne**, `ammoRead` quitte
  `required`. Additif / optionnel, aucune clé créée ni supprimée.
- **types web régénérés** (`make generate-types`) : `ammoRead: number` → `ammoRead?: number`, une
  seule ligne.
- **témoin** : `TestAmmoReadAZeroNEcritPasLaCle` prouve que la clé est **absente** à zéro et
  **présente** dès qu'il y a de la matière. **Mutation jouée** : `,omitempty` retiré, le test
  rougit en affichant `…,"endOpen":0,"ammoRead":0`.

---

## 7. La réserve du relecteur, instruite

**La réserve.** Le lot 6.13 a resserré `flagBirthsNear` du rayon du LÂCHER (1,5 m) à
`flagHomeExactDist` (0,10 m). Ce seuil sert **aussi** de dénominateur au verdict de variante
DRAPEAU NEUTRE (`flagChooseSpawns`, `flagNeutralMinBirths = 3`). Or `flag_neutral_test.go` pose
ses naissances **exactement** sur les socles : il passe avec l'ancien seuil comme avec le nouveau,
et ne dit donc rien du resserrement.

**Mesure sur le seul film neutre du parc, `4ecdf3e7` (High Ground, drapeau neutre), après 6.13 :**

| compteur | valeur |
|---|---|
| `neutralFlag` | **true** — le film est toujours classé neutre |
| `neutralBirths` | **5** naissances à **≤ 0,10 m** du socle neutre |
| `teamBirths` | **0** |
| `spawns` | 1 |

Le verdict tient avec **deux de marge** au-dessus du seuil de trois, et un dénominateur adverse
**vide** : le resserrement ne l'a pas approché.

**Le test ajouté** (`TestFlagNaissancesLaDISTANCEDecideDuDenominateur`) fait varier la SEULE
distance, six naissances à chaque fois :

- à **0,05 m** du socle → le drapeau rentre, le verdict reste **neutre** ;
- à **0,50 m** du socle → c'est un lâcher à portée du support, il **ne compte pas** et le verdict
  reste ordinaire.

---

## 8. Découvertes, consignées et NON traitées

1. **`flag_carriers_killed` ramène `t1` sans que la publication suive.** Quand il est le seul
   fermoir d'un portage, `t1` recule mais `closed` reste faux : le span est publié
   `carried_open` **jusqu'à la fin de l'axe**, en désaccord avec sa propre borne. Le faire clore a
   été mesuré et refusé (§3.2) : sur `b8a44fe8` le seul span concerné du parc passerait de +7,8 s
   à −9,9 s de son oracle. Trancher demande de qualifier ce que cet événement date vraiment —
   mesure, pas refactorisation.
2. **`closedByHandoff` retombe à 0 sur les 12 films.** Ce n'est pas une perte de signal mais la
   conséquence directe de la règle du plus petit fermoir : un passage de main suppose que le
   drapeau a quitté la main, donc qu'un lâcher — que l'objet date depuis 6.13 — le précède. Le
   compteur mesure désormais ce qu'il annonce (« portages dont le passage est le PREMIER fait qui
   les ferme ») et il est vide sur ce parc. Le bornage apporté par 6.11, lui, reste en place.
3. **Le sous-critère `[!]` du lot 6.13 est inchangé** : `2533274823110022` sur `b8a44fe8` reste à
   +7,8 s, seul span `carried_open` du parc, pour la cause déjà nommée (`flagMatchEnd` borne à la
   fin de l'AXE et non à `playable_duration_seconds`).

---

## 9. Les 15 conditions du relecteur

Les quinze conditions que le relecteur a constatées comme **tenant** sur la vague 6, recopiées
telles quelles. Cette ronde n'en touche aucune ; celles marquées « re-vérifiée » l'ont été à
nouveau ici parce que les correctifs passent dessus.

| # | condition | état |
|---|---|---|
| 1 | aucune écriture DuckDB | tient — **re-vérifiée** : toute la ronde cuit hors ligne, aucune base ouverte |
| 2 | garde-rail resserré | tient |
| 3 | fermoirs strictement intérieurs | tient — **re-vérifiée** : c'est désormais `flagCloseAt` seul qui le fait respecter |
| 4 | tri des instants | tient |
| 5 | demi-fenêtre sans chevauchement | tient |
| 6 | `flagHomeExactDist` en mètres | tient — **re-vérifiée** au §7 (dénominateur du verdict neutre) |
| 7 | garde d'effectif avec remplaçants | tient |
| 8 | `RefusedByRoster` équilibré | tient |
| 9 | offsets i9/i20 | tient |
| 10 | repli daté typé | tient |
| 11 | un glyphe par objet | tient |
| 12 | goldens expliqués | tient — aucun golden ne change dans cette ronde |
| 13 | mutation KOTH jouée | tient |
| 14 | i18n FR+EN | tient — aucune string neuve dans cette ronde |
| 15 | gates verts | tient — **re-vérifiée** au §10 |

---

## 10. Gates techniques, joués réellement

| commande | résultat |
|---|---|
| `CGO_ENABLED=0 go test ./internal/analysis/replay/ ./internal/analysis/objectiveevents/ ./internal/replaybuild/ ./internal/domain/replaydoc/...` | **ok**, 3 paquets + 1 sans test |
| `CGO_ENABLED=1 go test ./internal/service/...` | **ok**, 4 paquets (dont `replayview` — parité du convertisseur) |
| `CGO_ENABLED=0 go vet` (4 paquets) | propre |
| `golangci-lint run ./internal/analysis/replay/... ./internal/analysis/objectiveevents/... ./internal/domain/replaydoc/...` | **0 issues** |
| `go run ./cmd/openapi-gen -check` | **à jour** (712 162 octets) |
| `make generate-types` | régénéré, diff d'une ligne |
| `npx tsc -b --noEmit` (après purge de `node_modules/.tmp`) | **0 erreur** |
| `npx vitest run src/features/match-replay` | **185 fichiers, 2 711 tests verts**, 1 fichier / 3 tests skippés |
| `npx eslint src/features/match-replay/layers/useReplayGroundWeapons.ts` | **0 avertissement** |

`node_modules` était absent du worktree : installé par `npm ci` dans `apps/web` de CE worktree
(jamais celui du worktree partagé).

---

## 11. Films à recuire — la liste exacte

**9 des 12 films CTF changent :**

```
16ea3668  4ecdf3e7  58864b3c  7fce3219  8bc6074f
a0c36016  b8a44fe8  f8efc5ca  fb1a1a72
```

`bc60b4d9`, `bf5ced1b` et `cde26226` sont **identiques à l'octet**, ainsi que les 57 films non-CTF
du parc (aucune de leurs chaînes n'est touchée). **Aucune montée de `SchemaVersion`** : le seul
changement de FORME est le retrait de `coverage.groundWeaponItems.ammoRead` quand il vaut zéro,
et c'est précisément ce que `omitempty` doit produire. Un artefact non recuit n'est pas illisible ;
il dessine seulement un drapeau à sa base là où il gît au sol, et il compte deux fois certains
portages dans sa couverture.

## 12. Reproduction

```bash
# 1. cuisson HORS LIGNE des 12 films CTF (aucune base ouverte, chunks en lecture seule)
LEVELUP_REPO_ROOT=<racine de travail> \
  go run ./apps/go-api/cmd/replay-build --map <carte> --facts <short8>.facts.json <matchId> \
  <racine partagee>/data/cache/film_chunks/<short8>

# 2. confrontation a l'oracle API
cd .ai/V7.5/outillage/couverture_objectifs
CGO_ENABLED=0 go run . -parc <parc cuit> \
  -oracle <racine>/.ai/V7.5/replay2d/registre_film -out <sortie>
```

## Ronde 2 (2026-09-11, 17:00) — relecture des corrections seules, contexte frais

**P0 = 0, P1 = 0, P2 = 3.** La boucle de revue est close (skill `adversarial-review`, §8 : deux rondes, P0+P1 strictement décroissant, 5 → 0).

| # | constat | gravité | suite |
|---|---|---|---|
| R2-1 | `document_objectives_live.go:375` — le 3e terme de `Balanced()` (somme des `closedBy*` ≤ `closed`) est vrai par construction depuis que le compteur se dérive d'un unique `closedBy` par portage ; la moitié « `Balanced` rougit » de la mutation annoncée dans `flag_carries_home_test.go:156-157` n'est pas jouable | P2 | consigné : la protection réelle contre le double compte est la ligne `ClosedByHome != 0` du test et la représentation elle-même ; commentaire à requalifier au prochain lot qui touche `Balanced()` |
| R2-2 | `flag_carries_marker.go:74-76` — un portage fermé par une chaîne puis rejeté par `attachFlagCarryPositions` (`noTrack`, `outOfWindow`) ne peuple plus `closedBy*` (sonde : `openings=1 carries=0 noTrack=1 closed=0 closedByReturn=0`, contre `closedByReturn=1 / closed=0` avant) — le nouveau chiffre est le bon, mais c'est un second déplacement de chiffre publié, distinct des 21 doubles comptes | P2 | consigné : sur tout film où `noTrack`/`outOfWindow` ≠ 0, les `closedBy*` peuvent baisser pour cette raison ; « aucun chiffre ne bouge » vaut pour les portages et leurs durées (80 lignes joueur au centième), pas pour ces compteurs de couverture |
| R2-3 | `flag_carries_close.go:77` — quatre copies centralisées sans le garde-rail exigé par la règle n° 6 du dépôt | P2 | **corrigé par le superviseur** : `internal/archlint/no_flag_carry_end_outside_close_test.go` interdit toute affectation par sélecteur de `t1`/`closed`/`homed`/`captured`/`closedBy` dans `replay/flag_*.go` hors `flag_carries_close.go` ; joué vert, puis rouge sur une écriture ajoutée dans `flag_carries_home.go` (mutation), lint 0 |

**13 conditions vérifiées qui tiennent** (relecteur ronde 2) : (a) tous les fermoirs post-bornage passent par `flagCloseAt` (grep : seules écritures en `flag_carries_close.go:81-84` ; le bornage construit par littéral) ; `closeByCarrierKills` y passe avec `ferme()` faux = borne et non fermoir, conforme à la décision ; (a bis) `nextOpeningOfSlot` est un candidat du bornage en une passe, pas un fermoir tardif ; (b) un fermoir plus tardif est ignoré sans effacement (garde `at <= t0 || at >= t1`, `t1` monotone décroissant, ex æquo au plus précoce) ; (c) dérivation des `closedBy*` complète (capture/mort/reprise = `flagCloserBound` sans compteur, handoff/retour/rentrée/objet un compteur chacun, chute créditée aucun, fin de match = `carried_open`) ; effacement de `captured`/`homed` à chaque passage, `homed ⇒ closed` ; aucun fermoir n'allonge ; ripple aval vérifié sur `endsHome()` et ses trois consommateurs ; (d) mutations réellement jouées par `go test -overlay` : `homed` collant → seul `TestUnLacherPLUSTOTDEMENTLaRentree` rougit ; clause de domaine retirée → seul `TestLeBalayageJETTELEnregistrementHorsDomaine` rougit ; `omitempty` retiré → `TestAmmoReadAZeroNEcritPasLaCle` rougit ; `flagHomeExactDist` remis à 1,5 m → le test de la réserve rougit ; (e) `FlagCarriesCoverage.Balanced()` n'est appelé que par les tests (état antérieur au lot, pas une régression) ; C2 exact sur pièces (16 / 2^20 / rejet du record entier) ; C3 ferme un vrai trou ; C5 parité complète (`omitempty` des deux côtés, hors `required`, `ammoRead?: number`, golden OpenAPI vert) ; C4 conforme au code et aux trois strings FR/EN.

---

## Ronde 1 bis (2026-09-11, 17:20-18:10) — défaut trouvé par le gate local de fin de vague

> Périmètre fermé : ce défaut, ses tests, son rapport. Zéro fix opportuniste.
> Commits `1690aec04` (mesures figées du superviseur), `cea109f4d` (correctif + tests),
> `261e86c98` (coupe de fichier).

### 1 bis.1 Ce que le gate a montré

`cd apps/go-api && go test -tags=integration ./internal/api/wire/ -run TestOuvrierReel_ConstruitEtLivre`
échouait sur UNE assertion, `assertCompteursJoueurs` point 2 — la cohérence INTERNE entre les deux
dérivations du même compteur :

```
SÉRIE PUBLIÉE ≠ CLÉ D'APPARIEMENT pour 2535458702376288 : la série finit à 5 frags / 0 morts /
60 assistances, alors que le pont l'a apparié sur 5 / 0 / 0 incréments
```

Sur `c0a82e88` (CTF, 8 joueurs), ce joueur occupe le slot 12 dont le compteur d'assistances est
DÉROULÉ à 60 par un déroulage aberrant — sa feuille de match en porte ZÉRO.

### 1 bis.2 Cause, sur pièces

Le même compteur avait **deux dérivations et une seule borne**. Lignes relues au commit
`1690aec04` :

| chemin | lignes | borne par pas ? |
|---|---|---|
| **CLÉ D'APPARIEMENT** | `objectiveevents/slotidentity.go:86-88` vers `named.go:377-383` (`countsOf`) vers `named_series.go:410-436` (`incrementTimes`), test `n > maxUnrollPerStep` en **`named_series.go:421`** | **OUI** |
| **SÉRIE PUBLIÉE** | `replay/score_timeline.go:327-330` (`buildPlayerScoresFlat`) et `:364-367` (`buildPlayerScoresByRound`) vers `score_timeline.go:80-89` (`loadScoreSeries` / `at`) vers `objectiveevents/score.go:110-130` (`SeriesByRound`) et **`score.go:138-147` (`SeriesTotal`)** vers `score_timeline.go:122-143` (`scoreTicksOf`) | **NON** |

`SeriesTotal` au commit `1690aec04` :

```go
func SeriesTotal(recs []StatRecord, c StatComponent, teams bool) map[int][]ScorePoint {
	out := cumulateRounds(rawSeriesByRound(recs, c.key(), teams), RealRounds(recs))
	if !c.Strict { return out }   // rend la suite BRUTE : la borne par pas n'est jamais consultée
	...
}
```

Les deux chemins partagent bien le filtre de DOMAINE au niveau de l'enregistrement
(`statborg.go:260`, `statCountersInDomain`, appliqué au décodage donc en amont des deux) ; c'est la
borne PAR PAS, et elle seule, qui manquait à la série. Sur le slot 12 de `c0a82e88`, le pas 0 vers 60
sur `comp 3 A` est refusé côté clé (journal `passe=slot_identity comp=3 cote=A slot=12
time_ms=39694 deroulage=60 borne=16`) et retenu côté série.

**Pourquoi le gate le voit MAINTENANT.** Avant la vague 6 ce slot n'était pas pontable (0 mort,
triplet agrégé introuvable à la feuille) : rien de lui n'était publié, et l'écart n'était visible
nulle part. La vague 6 le rend nommable — 8 pontés sur 8 — et la courbe de score s'est mise à
servir **60 assistances pour 0** à l'interface. Le progrès a révélé le défaut ; il ne l'a pas créé.
Le parc le confirme : **2 joueurs étaient DÉJÀ dans cet état** (§1 bis.6).

### 1 bis.3 Le correctif — une seule dérivation, un seul lecteur de la borne

`named_bounds.go` (extrait de `named_series.go`, cf. §1 bis.7) porte **`boundSteps`, SEULE lecture
de `maxUnrollPerStep` du dépôt**, et ses deux formes :

| forme | fonction | consommateur |
|---|---|---|
| ÉVÉNEMENTS | `incrementTimes` | clé d'appariement, nommage des actions d'objectif |
| SÉRIE | `boundedSeries` | `SeriesTotal` (`score.go:175`), `SeriesByRound` (`score.go:147`) |

`boundedSeries` recalcule les VALEURS (cumul des pas acceptés) sans toucher aux instants ni à la
cardinalité — un compteur resté à zéro garde donc sa série de points à zéro, et aucun joueur ne
disparaît du document par `PlayerScore.empty()`.

**La borne n'est PAS étendue à tout — elle reste dans son domaine de calibration.** Un champ
`StatComponent.Unitary` (`score.go:73`) la demande, et seuls les trois compteurs d'ACTION le
portent (`score.go:91-93`) : exactement les trois que lit la clé. Les autres compteurs publiés
sont des compteurs de CADENCE dont un pas légitime dépasse largement 16 :

| emplacement | pas légitime |
|---|---|
| score PERSONNEL (`comp 1 B`) | +100 et plus par frag |
| tics de GARDE (`comp 23 A`) | 35 tics valent UN point de colline (`hill_hold_ticks.go`) |
| score de MODE (`comp 0 A`) | Strongholds compte des tics, Oddball des secondes |

Les borner effacerait des courbes vraies. **Ni la valeur (`16`) ni la sémantique des bornes ne
changent** : seul leur domaine d'APPLICATION s'étend de la clé à la série du même compteur.

### 1 bis.4 Tests, et les mutations jouées

`apps/go-api/internal/analysis/objectiveevents/named_derivation_unique_test.go` (neuf, 260 L) :

| test | ce qu'il tient |
|---|---|
| `TestSeriePublieeEtCleAppariementSAccordent` | sur la forme du défaut (slot à 5/0/0 dont `comp 3 A` déroule 60), `SeriesTotal` ET `SeriesByRound` finissent au compte de la clé ; et le pont nomme bien le joueur |
| `TestSerieSaineTraverseLaDerivationIntacte` | contre-épreuve : un pas de 3 (pire pas sain du parc) passe entier dans les deux |
| `TestBoundedSeriesGardeLaFormeDeLaSuite` | instants et cardinalité intacts, `prev` avance malgré le rejet |
| `TestBoundedSeriesEtIncrementTimesNeDiventJamaisDeuxChoses` | l'invariant général sur 8 formes de suite |
| `TestUneSeuleLectureDeLaBorneParPas` | **garde-rail de la factorisation** (règle n° 6) : balayage de TOUT le module, `maxUnrollPerStep` n'est lisible que dans `boundSteps` ; allowlist à 2 entrées, la seconde (`eventBudget.rejeter`) ne fait qu'IMPRIMER la borne dans son avertissement |

**MUTATION JOUÉE** — retirer les deux appels à `boundedSeries` de `score.go` :

```
--- FAIL: TestSeriePublieeEtCleAppariementSAccordent
    SERIE PUBLIEE != CLE D'APPARIEMENT pour le slot 12 : la serie finit a 60 assistances,
    la cle en compte 0 — la borne par pas ne descend pas dans la serie
    serie PAR MANCHE du slot 12 : finit a 60, attendu 0
```

**UN GARDE-RAIL VOISIN RÉPARÉ, ET C'EST LE CORRECTIF QUI L'EXIGEAIT.** L'échantillon égaré de
`manches_compteurs_test.go` portait 60 assistances — la forme réelle de `51ebbc0f`. À 60, la borne
par pas l'écarte AUSSI, si bien que les trois mutations du filtre de manche (correctif du
2026-09-06) auraient continué de PASSER : un garde-rail éteint par un autre filtre, sans que rien
ne le dise. La fixture passe donc à **15** assistances (`manchesEgareAssists`, pas de 12 <= 16, donc
SAIN pour la borne) : chaque filtre reste devant son propre défaut. Les trois mutations rejouées
après le changement rougissent bien — `TestEchantillonEgareNAlimentePasSaMancheDeclaree`
(« l'egare ... a ete range en manche 0 : elle finit a {14000, 15} au lieu de {7700, 3} »),
`TestEchantillonEgareNeGonflePasLeTotalDuJoueur` (« assistances totales = 18, attendu 6 »),
`TestEgareSeulResteEcarteMalgreLaGardeParSlot` (« la manche 0 du slot 12 finit a 15 au lieu de 3 »).

### 1 bis.5 Gate, AVANT vers APRÈS

| gate | AVANT | APRÈS |
|---|---|---|
| `go test -tags=integration ./internal/api/wire/ -run TestOuvrierReel_ConstruitEtLivre` | **FAIL** (1 assertion) | **ok** 6,7 s |
| `go test -tags=integration ./internal/api/wire/` (suite entière) | — | **ok** 16,2 s |
| `go test ./internal/analysis/replay/ ./internal/analysis/objectiveevents/ ./internal/replaybuild/ ./internal/service/...` | — | **ok** |
| `go test ./...` (module entier, hors tag `gamefiles`) | — | **0 échec** |
| `go test -tags=integration ./internal/analysis/... ./internal/replaybuild/...` | — | **0 échec** |
| `golangci-lint run --new-from-merge-base=origin/main ./...` | — | **0 issues** |
| `go vet` (paquets touchés + hook `go-vet` à chaque commit) | — | vert |

Les mesures figées par le superviseur (8 pontés, 20 frags / 6 assistances / 3 captures / 3 vols,
6 joueurs publiés, 3 portages) **tiennent sans être retouchées**.

### 1 bis.6 Effet sur le parc — cuisson hors ligne AVANT/APRÈS, 78 films

Protocole : `<short8>.facts.json` **dérivés des exports commités** (`oracle_vague6_registry.tsv` +
`oracle_vague6_participants.tsv`), dérivation validée à l'octet contre le fichier de référence
`testdata/equivalence/01e1f945.facts.json`. Deux binaires : **AVANT** construit depuis
`git archive 1690aec04`, **APRÈS** depuis `261e86c98`. Aucune base ouverte, chunks lus en lecture
seule, artefacts écrits dans un faux `LEVELUP_REPO_ROOT` de scratchpad. 78 réussites, 0 échec de
chaque côté.

| mesure | AVANT | APRÈS |
|---|---|---|
| films comparés | 78 | 78 |
| joueurs publiés (`scoreTimeline.players[*]`) | 604 | 604 |
| **joueurs dont la série finale BOUGE** | — | **2** |
| écart total (série − feuille, en valeur absolue) | **15 669** | **0** |
| **écart MOYEN par joueur** | **25,9421** | **0,0000** |
| joueurs dont la série ÉGALE sa ligne de feuille | 602 / 604 | **604 / 604** |
| artefacts identiques à l'octet | — | **76 / 78** |
| films dont le calque `objectives` bouge | — | **0** |

Les deux joueurs, avec leur ligne de feuille (relevé complet :
`.ai/V7.5/replay2d/registre_film/vague6_6R_joueurs_avant_apres.tsv`, 604 lignes) :

| film | xuid | AVANT (k/d/a) | APRÈS (k/d/a) | feuille (k/d/a) |
|---|---|---|---|---|
| `1b2d9e08` (Dynasty) | 2533274806449978 | 16 / 7 / **15 616** | 16 / 7 / **2** | 16 / 7 / 2 |
| `bf2a9f05` (Bazaar) | 2533274903615304 | 19 / 7 / **56** | 19 / 7 / **1** | 19 / 7 / 1 |

**Le calque `objectives` ne bouge sur AUCUN film**, et c'est attendu : il passe par
`incrementTimes`, donc il était déjà borné. Les 9 513 / 15 648 / 15 645 assistances fantômes citées
pour `16ea3668`, `8bc6074f` et `f8efc5ca` étaient des mesures de CE calque-là, prises avant que le
lot 6.11 ne pose le filtre de domaine au niveau de l'enregistrement : sur le parc recuit au HEAD de
la vague, ces trois films **ne portent plus aucun déroulage aberrant** sur les compteurs de la clé
(journal `passe=slot_identity` vide pour eux). `a0c36016`, `cde26226`, `fb1a1a72` et `4f77afc1` non
plus. Seuls `1b2d9e08` et `bf2a9f05` ont un résidu, et c'est lui que 6.R rend au sol.

**Le détecteur est EXHAUSTIF et il est dans le journal** : la passe `slot_identity` lit exactement
les trois compteurs dont la série est désormais bornée, et elle tourne à chaque cuisson AVANT comme
APRÈS. Ses deux lignes de rejet sur les 78 films nomment les deux mêmes films — la cuisson des 78
et la comparaison à l'octet confirment qu'il n'y en a pas d'autre.

### 1 bis.7 Films à recuire — la liste exacte

**2 des 78 films du parc** :

```
1b2d9e08   bf2a9f05
```

Les **76 autres sont identiques à l'octet**. Aucune montée de `SchemaVersion` : la forme du
document ne change pas, seules deux valeurs de série changent. Un artefact non recuit n'est pas
illisible ; il affiche seulement un compteur d'assistances faux pour un joueur.

À recuire AUSSI, hors parc : la fixture `c0a82e88` (absente de `oracle_vague6_registry.tsv`, donc
hors des 78) — c'est elle que porte `TestOuvrierReel_ConstruitEtLivre`, et le test la recuit à
chaque exécution.

Un item d'hygiène, TRAITÉ DANS LE PÉRIMÈTRE parce que le correctif l'a créé : `named_series.go`
repassait à 508 lignes, au-dessus du seuil de 500. La coupe (`named_bounds.go`, commit
`261e86c98`) suit la RESPONSABILITÉ — bornes, budget d'événements et dérivation unique d'un côté
(278 L), passage des enregistrements aux suites de valeurs de l'autre (244 L) — et ne change aucun
comportement.

### 1 bis.8 Reproduction

```bash
# 1. facts derives des exports commites (validation : identiques au fichier de reference
#    apps/go-api/internal/analysis/replay/testdata/equivalence/01e1f945.facts.json)
# 2. cuisson HORS LIGNE, deux binaires (AVANT = `git archive 1690aec04`, APRES = worktree)
LEVELUP_REPO_ROOT=<faux repo de travail> <binaire> --map <carte> \
  --facts <short8>.facts.json <matchId> <racine partagee>/data/cache/film_chunks/<short8>
# 3. confrontation scoreTimeline.players[*] contre oracle_vague6_participants.tsv
#    sortie : .ai/V7.5/replay2d/registre_film/vague6_6R_joueurs_avant_apres.tsv
# 4. detecteur exhaustif, lu dans le journal de cuisson :
grep "deroulage aberrant rejete.*passe=slot_identity" <journal de cuisson>
```

**Le corpus d'équivalence (`make replay-corpus-gate`) n'est PAS joué ici** : il compare au parc
RÉEL du dépôt partagé, que ce correctif déplace légitimement sur deux films. Il appartient au
superviseur, après recuisson.
