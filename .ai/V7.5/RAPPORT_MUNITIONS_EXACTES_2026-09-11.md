# RAPPORT — Les munitions EXACTES d'une arme au lâcher (lot 6.10, 2026-09-11)

> Branche `wt/munitions-objet`, worktree `LevelUp-wt-munitions-objet`.
> Aucune base DuckDB ouverte, aucune recuisson, aucun serveur touché. Les 64 artefacts cuits
> (schéma 51) et les chunks de film du cache sont lus comme des fichiers.
> Suite directe du lot 6.6 (`RAPPORT_MUNITIONS_OBJET_2026-09-10.md`), qui avait RÉFUTÉ les trois
> feuilles candidates de l'état par défaut de `ti=42`.

---

## 0. LES QUATRE LIGNES QUI COMMANDENT LE RESTE

1. **Les munitions SONT sur l'objet — mais pas là où 6.6 les a cherchées.** Elles ne sont pas
   dans l'ÉTAT PAR DÉFAUT de `ti=42` (points 3, 4, 5 : réfutés, et ils le restent). Elles sont
   dans un **COMPOSANT du même record de création**, `weapon-ammo-component` (i20, déser
   `FUN_140fc3028`, grammaire `R(8) + R(11) + R(12)`), qui vit **après le masque de présence**.
   Le balayage des créations s'arrêtait au composant i0 (la position) et n'allait jamais jusque-là.
2. **Deux de ses trois champs sont PROUVÉS** : le champ A `R(8)` est le **chargeur**, le champ B
   `R(11)` la **réserve**. Le record de création est daté à l'INSTANT DU LÂCHER : ce n'est pas
   une lecture d'inventaire en retard de neuf secondes, c'est l'état de l'arme quand elle touche
   le sol. Le troisième champ (C, `R(12)`) n'est PAS publié — sa sémantique n'est pas établie.
3. **La couverture est le point faible, et sa cause est ISOLÉE et CHIFFRÉE.** Pour atteindre i20
   il faut traverser les composants qui le précèdent ; tous sont portés sauf **i9
   `object-multiplayer-properties`**, dont le flux TLV ne consomme pas le bon nombre de bits sur
   cet archétype. On ne lit donc que lorsque la marche est PROUVÉE bit-exacte :
   **4 283 lectures sur 27 155 créations `ti=42` (15,8 %)** sur le parc.
4. **Aucun champ de fin de vie n'a été trouvé** (demande utilisateur). Le champ C est **nul dans
   97,7 %** des objets lâchés lus, et ses quartiles n'ont aucune relation avec la durée observée
   de l'objet. La liste des champs lus est au §5.

**VERDICT PAR VOIE.** Voie COURTE (étape 1) : **sémantique PROUVÉE, couverture 15,8 %** — le
gate de VALEUR est tenu (témoins 0,0 à 3,2 %, très en dessous des 10 % exigés), le gate de
POPULATION ne l'est pas, et sa cause est nommée. Voie LONGUE (étape 2) : **non ouverte**, et
c'est motivé au §6 — elle chercherait une grandeur moins bonne que celle qui est déjà prouvée.

---

## 1. MÉTHODE

### 1.1 Ce qui a été instrumenté

Rien de nouveau n'a été décodé : le composant i20 était **déjà porté** dans
`traverse.go` (`consumeWeaponAmmo`, posé le 2026-08-30) et **déjà jugé illisible** ce jour-là
(entrée du journal « Munitions des armes au sol : composant atteint, valeurs NON FIABLES »).
Ce lot-ci n'a pas changé sa grammaire : il a changé **la façon de l'atteindre**.

| Ce qui a changé | Détail |
|---|---|
| Point de lecture | Le record de **CRÉATION** (`ScanGroundWeaponCreationsForBand`), pas les paquets delta. Il porte `ti=42` explicitement, son default-state est validé par oracle de position depuis le 2026-08-17, et il est daté à l'instant du lâcher |
| Attribution | La boucle de composants de production (`traverseComponentLoop`) est rejouée **record par record** à partir du premier bit d'i0, avec son PROPRE curseur. La sonde de 2026-08-30 capturait un global de paquet et l'attribuait au premier record `ti=42` venu — l'attribution était fausse par construction |
| Garde | La lecture est REFUSÉE quand le masque porte i9 (§3), quand il ne porte pas i20, ou quand l'archétype ne porte pas ces composants à ces index |

L'instrument de mesure (`zz_mun*_research_test.go`, garde `MUN_RACINE`, treize fichiers) est
**supprimé après mesure** — précédent E0 et lots 6.3 / 6.6. Sa recette est au §7.

### 1.2 Le corpus et l'oracle

- **64 artefacts** du parc local, `data/cache/replays/halo_infinite/<8 hex>.json`, schéma 51,
  et leurs chunks `data/cache/film_chunks/<8 hex>/`. Aucun film écarté a priori ; deux films
  (`0797ce72`, `30724141`) ne rendent aucun cas parce que leur artefact ne publie aucun objet
  `dropped` à lâcheur nommé.
- **Câblage identique à la cuisson** : largeurs d'axe lues du film (`DetectI0LayoutOf`),
  largeurs du bloc MPP calibrées par le chemin de production (`ScanEquipmentPlacements`), bande
  de slots prise des images-clés (`ScanWorldObjectKeyframes`).
- **Bornes monde synthétiques** (le catalogue de cartes ne se résout pas hors ligne sans la
  base) : les positions décodées n'ont donc aucun sens ici et **ne sont pas utilisées**. Elles ne
  servent qu'au gate de sélectivité du balayage, exactement comme au lot 6.6.
- **Appariement objet ↔ record** : par (famille `%08x`, image de naissance ± 2), l'image étant
  reconstruite avec l'horloge du document (`(ts − premierPaquetDuChunk1)/1000 − originMs`, divisée
  par `frameIntervalMs`). Les appariements ambigus sont écartés.
- **L'ORACLE** : pour chaque objet `dropped` à lâcheur nommé, le chargeur et la réserve du
  lâcheur à sa **dernière lecture d'inventaire avant `t0`** (même recette que le repli livré au
  lot 6.6 : loadout du MÊME instant, emplacement indexé par l'ordre de `loadouts[].w`).
  **6 756 cas appariés**, dont **939 sur la marche prouvée bit-exacte**.

**CETTE RÉFÉRENCE EST EN RETARD, ET C'EST CE QUI LA REND PROBANTE.** Elle vieillit de 9,0 s en
médiane (mesure 6.3 §4.2) — précisément la fenêtre pendant laquelle le porteur tire. Un champ qui
serait le chargeur doit donc **s'en écarter de plus en plus quand elle vieillit**, et rester
**toujours inférieur ou égal** à elle hors rechargement. C'est cette FORME, et non un taux brut,
qui constitue la preuve.

---

## 2. RÉSULTAT — LA SÉMANTIQUE DES DEUX PREMIERS CHAMPS

### 2.1 Le gradient de fraîcheur (64 films, 939 cas bit-exacts)

| Âge de la référence | n | A == chargeur | A ≤ chargeur | B == réserve |
|---|---:|---:|---:|---:|
| ≤ 1 s | 62 | **74,2 %** | **100,0 %** | **96,8 %** |
| 1 à 2 s | 55 | 52,7 % | 94,5 % | 94,5 % |
| 2 à 5 s | 174 | 42,5 % | 96,0 % | 90,8 % |
| 5 à 10 s | 247 | 36,4 % | 92,7 % | 82,2 % |
| 10 à 20 s | 377 | 41,4 % | 92,0 % | 66,0 % |
| > 20 s | 24 | 41,7 % | 91,7 % | 37,5 % |

**Les deux courbes disent deux choses différentes, et c'est ça le résultat.** A s'effondre vite
(74 % → 36 %) : le chargeur change à chaque rafale. B décroît lentement (97 % → 66 % → 38 %) : la
réserve ne bouge qu'au rechargement ou au ramassage. Aucune paire de champs pris au hasard ne
produirait ces deux pentes-là dans ce sens-là.

**Les 25,8 % de désaccord à référence fraîche ne sont pas du bruit** : ils sont l'imprécision de
la RÉFÉRENCE, pas de la lecture. Une seconde suffit à vider un chargeur en mêlée. Le contrôle qui
le confirme est la colonne « A ≤ chargeur » : **100,0 % à ≤ 1 s** — la valeur lue n'est JAMAIS
supérieure à la dernière lecture connue. Les 8 % qui débordent aux âges élevés sont les
rechargements, qui remontent le chargeur entre les deux instants.

### 2.2 Les témoins

| Témoin | A | B |
|---|---:|---:|
| Référence prise sur un **AUTRE objet** (permutation) | **3,2 %** | **9,7 %** |
| Champ **VOISIN de même largeur**, 19 bits plus loin dans le même composant | **0,0 %** | — |

Les deux sont sous le seuil de 10 % du contrat, le second est à zéro.

### 2.3 Un second oracle, INDÉPENDANT de l'inventaire : les capacités par famille

Le mode du champ A, famille d'arme par famille d'arme, sur les 1 768 lectures de 24 films :

| Famille | n | valeur dominante de A |
|---|---:|---:|
| `2b1824d5` | 272 | **36** |
| `fd98554c` | 97 | **20** |
| `3e070217` | 101 | **20** |
| `84bd29ed` | 81 | **12** |
| `80977ba5` | 84 | **8** |
| `9d6aaed2` | 71 | **6** |
| `0a1992bc` | 243 | 0 |
| `d7915565` | 64 | 0 |

Ce sont des **capacités de chargeur** — 36, 20, 12, 8, 6 — une par famille, et le mode est faible
(11 à 30 %) exactement comme il doit l'être : une arme lâchée est le plus souvent entamée, le
chargeur plein n'est que le cas le plus fréquent. Deux familles ont 0 pour mode : une arme lâchée
vide est le cas dominant pour elles. **Cet oracle n'emprunte rien à l'inventaire** : il ne repose
que sur la cohérence interne du champ avec l'identité de l'arme.

### 2.4 Ce que l'on NE publie pas : le champ C

**Distribution** (4 283 lectures, 64 films) : 1 323 valeurs distinctes, 0 dans 42 % des cas,
puis des valeurs éparpillées (4, 8, 2 688, 1 461, 3 584…). Mode par famille : 46,3 %. Il n'a
donc ni la forme d'un compte, ni celle d'une constante d'arme. **Il est consommé et jeté**, comme
les trois feuilles réfutées du lot 6.6 : ne rien en tirer est le résultat, pas un oubli.

---

## 3. LA RÉSERVE DE LECTURE — i9, ISOLÉ ET CHIFFRÉ

### 3.1 Comment la cause a été trouvée

La première mesure, tous records confondus, ne donnait rien : à la position où la marche place
i20, A n'égalait le chargeur que dans 7,2 % des cas (témoin 0,7 %) et B la réserve dans 15,7 %
(témoin 3,1 %). Un signal réel, dix fois le témoin, mais noyé.

**Le tri par SIGNATURE DE MASQUE a tout séparé** — deux records de même masque subissent la même
dérive :

| Masque du record | n | A == chargeur | B == réserve |
|---|---:|---:|---:|
| `[0 1 2 12 14 15 17 20]` | 191 | **39 %** | **81 %** |
| `[0 1 2 12 14 15 20]` | 17 | **41 %** | **82 %** |
| `[0 1 2 6 12 14 15 17 20]` | 16 | **62 %** | **75 %** |
| `[0 1 2 **9** 12 14 15 17 20]` | 868 | 1 % | 1 % |
| `[0 1 2 **6 9** 12 14 15 17 20]` | 331 | 0 % | 1 % |

**Le partage est exactement i9.** Confirmation par la recherche, record par record, du décalage
qui fait retomber A ET B sur l'oracle : sur les records SANS i9, ce décalage vaut **ZÉRO dans 80
cas sur 89 (89,9 %)** ; sur ceux QUI portent i9, il s'éparpille sur **121 décalages distincts**.

### 3.2 Ce qu'on sait de la longueur vraie d'i9, et pourquoi ça ne suffit pas

Recherche de la position de FIN d'i9 par rejeu des composants suivants (392 records avec i9,
69 longueurs trouvées sans ambiguïté, soit 17,6 %) :

- **34 longueurs distinctes**, concentrées entre **300 et 470 bits** (modes : 331, 312, 360, 341) ;
- **aucune structure modulo 8** (restes 0:23, 3:16, 5:11, 2:5, 7:5, 1:4, 4:3, 6:2) alors que le
  flux TLV porté est décrit comme orienté OCTET ;
- **aucune corrélation** avec les six premiers bits d'i9 (porte + étiquette de variante).

Sept lectures candidates ont été rejouées sur 1 224 records et **toutes échouent** :

| Candidate | records alignés |
|---|---:|
| le portage actuel (`R(1)` ; si 0 → `R(5)` + TLV) | 0 (0,0 %) |
| 1 bit seul (bloc absent) | 0 (0,0 %) |
| polarité inverse | 1 (0,1 %) |
| `R(1) + R(5)` sans TLV | 2 (0,2 %) |
| le bloc du default-state `FUN_14080cfe8` | 0 (0,0 %) |
| `R(1)` + [ce bloc si 0] | 1 (0,1 %) |
| 0 bit | 0 (0,0 %) |

**La grammaire d'i9 ne se rétablit donc pas par la mesure : il faut désassembler
`FUN_1407d4c94`.** Ni Ghidra ni Cheat Engine n'étaient joignables dans cette session (aucune
instance Ghidra en cours, pont Cheat Engine injoignable), et aucun dump statique de cette
fonction n'existe au dépôt. C'est l'item `[!]` du lot.

### 3.3 Ce que ça coûte, en couverture

Sur les 64 films : **27 155 créations `ti=42`**, dont **16 218 (59,7 %)** portent i20 au masque,
dont **4 283 (26,4 % de celles-là, 15,8 % du total)** sont LISIBLES. Sur les objets `dropped`
appariés à un artefact : **939 sur 6 756 (13,9 %)**.

**Critère de levée, mesurable** : le portage d'i9 corrigé, la position d'i20 doit retomber au
décalage zéro sur > 95 % des records qui portent i9 — le même oracle que celui de ce lot. La
couverture passerait alors de 15,8 % à 59,7 % des créations.

---

## 4. CE QUI EST PUBLIÉ

| Élément | Fichier | Commit |
|---|---|---|
| Lecteur + preuve + réserve de lecture | `apps/go-api/internal/analysis/filmdec/ground_weapon_ammo.go` (+ son test sur octets fabriqués) | `be7d7a2fd` |
| Câblage au balayage des créations | `filmdec/equipment_creation.go` (`HasAmmo`/`Ammo`, stat `WithAmmo`), `filmdec/ground_weapon_creation.go` | `be7d7a2fd` |
| Golden des familles régénéré + preuve d'invariance | `filmdec/golden_minibobine_test.go`, `testdata/golden_minibobine_familles.tsv` | `be7d7a2fd` |
| Champ au document | `replay/document_ground_weapon_items.go` (`GroundWeapon.Ammo`, `GroundWeaponAmmo`, couverture `AmmoRead`), `replay/ground_weapon_objects.go` | `838e9c7bb` |
| Parité `replaydoc` + convertisseurs | `domain/replaydoc/{ground_weapons,coverage_world}.go`, `service/replayview/convert_{ground_weapons,coverage_world}.go` | `838e9c7bb` |
| Contrat | `api/openapi.yaml` (diff ADDITIF), `apps/web/src/lib/api/generated.ts` | `838e9c7bb` |
| Infobulle + i18n FR/EN + garde-rail de clés | `apps/web/src/features/match-replay/{model/groundWeaponAmmo.ts,i18n/*,layers/*}`, `lib/api/types.ts` | `c0fa82958` |

**Forme publiée** : `groundWeapons[].ammo = { mag, res }`, **optionnel** (`omitempty`) et
**additif** — **aucun bump de `SchemaVersion`** (la règle du document l'exige seulement pour un
changement cassant). `coverage.groundWeaponItems.ammoRead` compte les objets qui en portent.

**L'ABSENCE RESTE ABSENTE.** Un objet sans lecture ne publie pas un couple de zéros, qui se
lirait « arme vide » — le pointeur est nil et le convertisseur le garde nil. Un chargeur exact
à **zéro**, lui, se publie : une arme lâchée vide est une information.

**Infobulle** : « … · **12 au chargeur, 24 en réserve** » — sans « ≈ » et sans âge, parce qu'il
n'y a rien à dater. Le repli du lot 6.6 (dernière lecture d'inventaire du lâcheur AVEC son âge)
reste en place pour les objets sans lecture exacte. Le type `GroundWeaponAmmoReading` est devenu
une **union discriminée** (`exact` / `dated`) : une lecture datée ne PEUT PLUS s'afficher sans
son âge, parce que le type ne permet pas de l'oublier.

**RECUISSON : OUI, ET DE TOUT LE PARC.** `ammo` est un champ CUIT. Les 64 artefacts locaux et la
production doivent être recuits pour le porter. Le plan maître prévoit déjà une recuisson unique
après 6.10 et 6.11 : c'est celle-là.

**Preuve d'invariance des bits**, mesurée des DEUX côtés du changement sur la mini-bobine
versionnée (empreinte sha256 des champs pré-existants, `HasAmmo`/`Ammo` exclus ; copie
`git archive HEAD` d'un côté, branche de l'autre, même test) :

```
groundWeaponCreations  n=28  ancres=141  acceptees=28
  sha256=a71593143c298a8c5e2696bc8d5d9c3847b95f976dff269ae87feb15214f55e5   (identique)
equipmentCreations     n=38  ancres=170  acceptees=38
  sha256=71cd12247c95a07f11849fc8d43565674172cebfd140145de31ef6cefe086cee   (identique)
```

Le golden `golden_minibobine_familles.tsv` change sur ces deux lignes, et seulement parce que
`rendreStable` rend TOUS les champs : sur CETTE bobine, les 28 créations d'arme au sol portent
TOUTES i9, donc `HasAmmo` vaut `false` partout. L'explication est dans son en-tête.

---

## 5. LA FIN DE VIE — AUCUN CHAMP DE CE TYPE TROUVÉ

La demande utilisateur était : « en décodant plus largement, note tout champ qui ressemble à un
compteur de disparition ou à un instant de fin de vie ».

**Réponse : aucun champ de ce type n'a été trouvé.** Ce qui a été lu et ce qu'il donne :

| Champ lu | Verdict |
|---|---|
| i20 champ C `R(12)` | **Nul dans 97,7 % des objets lâchés lus (388 sur 397)**, et les 9 exceptions sont toutes des objets de fin `seen`. Ventilation par fin : `open` 100 % nul, `pickup` 100 % nul, `seen` 97,4 % nul. Les quartiles de C contre la durée OBSERVÉE de l'objet donnent des médianes de 55, 60, 75 et 73 images — aucune relation, et les trois premiers quartiles sont entièrement à zéro |
| i20 champs A et B | Chargeur et réserve (§2). Une arme vide n'a pas une durée de vie différente : le champ A vaut 0 dans 13,5 % des lectures, toutes fins confondues |
| Les 21 composants de l'archétype `ti=42` | Deux portent un nom évocateur et **n'ont pas été mesurés** : `object-dead-state-component` (i11, présent dans **0,7 %** des créations — trop rare pour dater quoi que ce soit) et `object-dissolver-component` (i14, présent dans **71,4 %**). Leurs déserialiseurs consomment leurs bits sans en rendre la valeur |

**Ce que le dépôt savait déjà et qui reste vrai** : le film ne date la disparition d'AUCUN objet
posé (acquis du correctif de revue des poses, 2026-08-17 — `t1` est une mise au repos, pas une
disparition ; le record DEL est noyé). Le calque des armes au sol publie donc un INTERVALLE
`[t1, t1max]` borné par le recensement des images-clés, et c'est toujours la meilleure réponse
disponible.

**Piste de reprise, non traitée** : `object-dissolver-component` (i14) est le seul candidat au
nom explicite présent sur 71 % des créations. Le mesurer demanderait de lui faire rendre ses
valeurs (même geste que pour i20 ici) et de les confronter à l'intervalle de disparition observé.
C'est un lot à part.

---

## 6. POURQUOI LA VOIE LONGUE N'A PAS ÉTÉ OUVERTE

Le contrat prévoyait, en cas d'échec de la voie courte, de rouvrir le canal munitions des paquets
DELTA de l'arme PORTÉE (`consumeWeaponStateTypeInfoVariant`, `components_object.go`). Elle n'a
pas été ouverte, pour trois raisons **mesurées**, pas préférées :

1. **La voie courte n'a pas échoué sur la sémantique** — c'est elle que le contrat désignait
   comme suffisante (« si tenu → étape 3 »). Les témoins sont à 0,0 et 3,2 %, la borne
   « A ≤ chargeur » est à 100,0 % à référence fraîche.
2. **La voie longue donnerait une grandeur MOINS bonne pour la question posée.** Le contrat
   demande « les munitions au LÂCHER ». Le record de création EST daté au lâcher ; les deltas de
   l'arme portée datent, eux, le dernier CHANGEMENT d'état d'arme avant le lâcher — il faudrait
   encore les propager jusqu'à l'instant du lâcher.
3. **Le bloc que la voie longue viserait a déjà été réfuté en partie.** `consumeWeaponMagazineList`
   y porte des HANDLES (lot 6.6, 0 égalité sur 1 524 comparaisons) ; les `R(12)` et `R(7)` qui le
   précèdent sont ceux-là mêmes que 6.6 a réfutés sur `ti=42`. Ce qui reste à y décoder
   (`consume1407f0550`) n'a aucun oracle propre.

**Ce qui ferait vraiment gagner de la couverture n'est pas la voie longue : c'est i9** (§3.2). La
même correction sert la voie courte ET toute autre lecture de composant sur cet archétype.

---

## 7. LA RECETTE DE L'INSTRUMENT (supprimé après mesure)

Treize fichiers `zz_mun*_research_test.go` dans `internal/analysis/filmdec/`, garde
`MUN_RACINE=<racine qui porte data/>` (+ `MUN_N=<nb de films>`), reproductibles en une centaine
de lignes :

1. **Socle** (`munCasesOf`) : charger le film, `DetectI0LayoutOf` +
   `SetWorldObjectPrecisionFromLayout`, calibrer les largeurs MPP par `ScanEquipmentPlacements`,
   bande par `ScanWorldObjectKeyframes`, créations par `ScanGroundWeaponCreationsForBand` ; lire
   l'artefact JSON avec des structures locales ; apparier par (famille, image ± 2, unicité
   exigée) ; calculer l'oracle (chargeur/réserve du lâcheur, loadout du même instant).
2. **Traversée** : `NewBitReader(payload)`, `SetBitPos(cre.AfterBit − projPosBits())`,
   `traverseComponentLoop` avec le masque de la création convertie en `uint64`, puis relire à
   `CompResult{Index:20}.StartBit`.
3. **Les cinq mesures** : masque (quels composants), balayage de décalage (où le champ tombe),
   tri par signature de masque (qui dérive), gradient de fraîcheur (la preuve), longueurs vraies
   d'i9 (ce qu'il faudrait pour lever la réserve).
4. **Empreinte d'invariance** : sha256 du rendu `rendreStable` de chaque record, champs nouveaux
   EXCLUS, joué des deux côtés d'une copie `git archive HEAD`.

Durées observées : 190 s pour le gradient sur les 64 films, 72 s pour la couverture sur 24.

---

## 8. STATUT DES ITEMS DU CONTRAT

| Item | Statut |
|---|---|
| **Étape 1.1** — décoder plus largement l'état de l'objet `ti=42`, grammaire sur pièces | `[x]` fait, mais **par le record de CRÉATION, pas par l'image-clé**, et la substitution est motivée : le corps d'un record d'image-clé n'est PAS bit-exact (lot R7-a/R7-e, 0,51 % d'exactitude, RE ARRÊTÉE par décision utilisateur) et il porte l'état COMPLET, donc i9 toujours — la voie image-clé aurait la même réserve PLUS celle du cadre. Le record de création, lui, est validé par oracle de position (97,6 %) et daté au lâcher. La « grammaire établie » est celle d'i20 (`FUN_140fc3028`, `R(8)+R(11)+R(12)`), déjà au dépôt ; ce lot l'a ATTEINTE, pas réécrite |
| **Étape 1.2** — oracle chiffré, constance, contre-exemples, témoin sur champ voisin | `[x]` §2.1 à §2.3. La « constance d'une image-clé à l'autre » est **sans objet** : la lecture retenue est unique par vie d'objet (record de création), il n'y a pas deux lectures à comparer. Elle est remplacée par un contrôle PLUS fort — la borne `A ≤ chargeur` à 100,0 % et le gradient de fraîcheur |
| **Étape 1.3** — gate de la voie courte | `[~]` partiellement : gate de VALEUR tenu (témoins 0,0 / 3,2 % < 10 % ; `A ≤ chargeur` 100,0 % ; `B == réserve` 96,8 % à référence fraîche ; 64 films ≫ 3), gate de POPULATION **non tenu** (15,8 % au lieu de 95 %), cause isolée et chiffrée au §3 |
| **Étape 2** — voie longue | `[!]` NON OUVERTE, motivé au §6 : elle viserait une grandeur moins bonne que celle qui est prouvée, et ce qui manque à la couverture est i9, pas un canal |
| **i9 — rétablir la grammaire** | `[!]` NON TRAITÉ, faute d'outil : aucune instance Ghidra, pont Cheat Engine injoignable. Ce qui est établi pour la reprise : longueur vraie 300 à 470 bits, 34 valeurs distinctes, aucune structure modulo 8, sept lectures candidates réfutées (§3.2), critère de levée mesurable (§3.3) |
| **Étape 3** — champ `ammo` cuit, omitempty, additif, sans bump, couverture publiée | `[x]` §4 |
| **Étape 3** — openapi + `generate-types`, parité `replaydoc` | `[x]` diff purement additif |
| **Étape 3** — infobulle exacte, i18n FR+EN, test de mutation | `[x]` deux mutations jouées côté web (préférence de la source ; « ≈ » ajouté à la forme exacte), deux côté Go (ordre des champs ; refus d'i9) — les quatre rouges |
| **Étape 3** — fin de vie / compteur de disparition | `[x]` **aucun champ de ce type trouvé**, §5, avec la liste des champs lus et une piste de reprise nommée |
| Découverte 6.6 n° 2 (propriétaire des `spawned`) | `[~]` hors périmètre, non traitée |
| Instrument supprimé avant livraison, recette au rapport | `[x]` §7 |
| Empreintes des goldens : invariance démontrée sur octets réels | `[x]` §4 |

---

## 9. DÉCOUVERTES (consignées, NON traitées — règle « zéro fix opportuniste »)

1. **`consumeObjectMultiplayerProperties` (i9) n'est pas bit-exact sur `ti=42`** —
   `apps/go-api/internal/analysis/filmdec/components_batch7.go:26`. C'est la découverte centrale
   du lot et elle dépasse ce calque : i9 est un composant d'objet GÉNÉRIQUE, présent sur la
   plupart des archétypes. Toute lecture de composant située APRÈS lui, sur n'importe quel
   archétype, hérite de la même dérive. Le lot R7-a l'avait déjà désigné comme « le composant le
   plus souvent responsable du décrochage de l'image-clé (25 % des franchissements de
   frontière) » ; sa polarité a été corrigée le 2026-08-17 (R7-b) mais **pas son flux TLV**.
2. **L'entrée de journal du 2026-08-30 attribuait la faute au mauvais endroit** —
   `.ai/thought_log.md`, « Munitions des armes au sol : composant atteint, valeurs NON FIABLES ».
   Elle conclut « l'état par défaut de `ti=42` n'a jamais été validé par un oracle », alors qu'il
   l'était depuis le 2026-08-17 (`6603eeaf8`). La vraie cause était double : une attribution de
   sonde fausse par construction (un global de paquet rendu au premier record venu) et i9. La
   conclusion pratique de cette entrée (« ne pas afficher de munitions d'arme au sol ») est
   maintenant PÉRIMÉE pour les records lisibles.
3. **`object-dissolver-component` (i14) est présent sur 71,4 % des créations `ti=42`** et n'a
   jamais été mesuré — seul candidat au nom explicite pour une fin de vie (§5).
4. **Le champ C d'i20 se comporte différemment selon l'origine de l'objet** : nul dans 97,7 % des
   objets `dropped`, mais seulement dans 42 % de la population lisible entière (qui inclut les
   armes de râtelier). Il porte donc quelque chose que seules les armes APPARUES ont — non
   identifié, non publié.

---

## 10. RÉFÉRENCES

- Contrat du lot : `.ai/PLAN_MASTER_2026-09-09.md`, ligne `| 6.10 |` et journal des
  2026-09-10 22:55 / 2026-09-11 10:05.
- Amont : `.ai/V7.5/RAPPORT_MUNITIONS_OBJET_2026-09-10.md` (lot 6.6, ce qui est RÉFUTÉ),
  `.ai/V7.5/RAPPORT_ARMES_AU_SOL_2026-09-10.md` §4 (le canal inventaire et sa cadence),
  `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`.
- Code : `filmdec/ground_weapon_ammo.go` (la preuve en en-tête),
  `filmdec/unit_weaponstate.go` (`consumeWeaponAmmo`), `filmdec/components_batch7.go` (i9),
  `filmdec/traverse.go`, `replay/document_ground_weapon_items.go` ; côté web
  `features/match-replay/model/groundWeaponAmmo.ts`.
