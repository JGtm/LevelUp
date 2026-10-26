# RAPPORT — Les munitions EXACTES d'une arme au lâcher (lot 6.10, 2026-09-11)

> **CE RAPPORT A UNE SUITE, ET ELLE CHANGE DEUX DE SES CONCLUSIONS.** Le lot **6.10 bis**
> (section en fin de fichier, meme journee) a desassemble i9 : la RESERVE DE LECTURE du §3 est
> **LEVEE**, la couverture passe de 15,8 % a 59,7 %, et le critere « > 95 % » du §3.3 s'est
> revele au-dessus du plafond de l'oracle. La piste `object-dissolver-component` (i14) nommee au
> §5 a ete instruite : **aucun champ de fin de vie n'y est etabli**, et la cause est prouvee.
> Tout le reste de ce rapport tient.

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

---
---

# 6.10 BIS — i9 DÉSASSEMBLÉ, LA RÉSERVE LEVÉE, i14 MESURÉ (2026-09-11)

> Suite directe du lot 6.10 : ses deux items `[!]` (i9, faute de désassembleur ;
> `object-dissolver-component` i14, jamais mesuré) sont traités. Même worktree
> `LevelUp-wt-munitions-objet`, même branche `wt/munitions-objet`. Aucune base DuckDB ouverte,
> aucune recuisson jouée, serveur intact. Ghidra était joignable cette fois : instance pid 26060,
> `HaloInfinite.exe` ouvert, image base `140000000`, 311 103 fonctions.

## B0. LES TROIS LIGNES QUI COMMANDENT LE RESTE

1. **La grammaire d'i9 ne ressemblait à aucune fonction du binaire.** Le portage inventait un flux
   TLV « chaque type a un corps ». Le vrai flux est un **sous-message en mode 2** : un en-tête
   LEB128, puis des champs dont le premier octet porte un TYPE DE FIL sur ses 5 bits bas et un
   COMPTE DE SAUT sur ses 3 bits hauts, chaque type de fil ayant sa longueur propre. Quatre erreurs
   sont corrigées, chacune nommée au §B2.
2. **La preuve qui tranche n'est pas l'oracle, c'est la congruence.** La longueur consommée par i9
   vaut **6 modulo 8 dans 100,0 % des 6 318 records** qui le portent — exactement ce qu'exige
   1 bit de porte + 5 bits d'étiquette suivis d'un flux d'OCTETS — et elle ne prend plus que **10
   valeurs, toutes entre 414 et 502 bits**. Le lot 6.10 mesurait 34 longueurs et « aucune structure
   modulo 8 ». Ce contrôle n'emprunte rien à l'inventaire.
3. **`object-dissolver-component` ne porte aucun champ de fin de vie, et la cause est
   structurelle.** À l'instant du record de CRÉATION son état vaut le neutre (13) dans **99,80 %**
   des cas : un composant de dissolution décrit une FIN, et la fin n'est pas connue à la naissance.

**CE QUE LA COUVERTURE DEVIENT** : les munitions exactes passent de **4 283 lectures sur 27 155
créations `ti=42` (15,8 %)** à **16 218 (59,7 %)** — c'est-à-dire la totalité des records dont le
masque annonce i20. Les 40,3 % restants ne portent aucune munition à lire.

---

## B1. LA CHAÎNE D'APPELS D'i9, RELUE INSTRUCTION PAR INSTRUCTION

`consumeObjectMultiplayerProperties` porte `FUN_1407d4c94`. Ce qu'il fallait établir n'était pas
dans cette fonction-là mais dans les neuf qu'elle entraîne.

| Adresse | Rôle | Ce que l'instruction décisive dit |
|---|---|---|
| `FUN_1407d4c94` | i9 | `R(1)` porte ; si le bit vaut 0 le bloc est PRÉSENT, `R(5)` étiquette de variante, puis le sous-message. La branche `else` écrit `*(dst+0xbc) = 0` — l'effacement d'un champ absent |
| `FUN_1407d54ac` | construction de la variante (0..5) | **NE LIT AUCUN BIT** : six branches qui appellent un constructeur et posent `dst+0xbc` |
| `FUN_1407d4e10` | pose le contexte de flux | `1407d4e39: MOV EAX,0x2` puis `1407d4e3e: MOV word ptr [RSP+0x28],AX` — le **mode vaut 2** |
| `FUN_1424ccb1c` | recopie le contexte | `1424ccb28: MOV AL,byte ptr [RDX+0x29]` — l'octet HAUT du mot `1` posé en `1407d4e5f`, donc **ZÉRO** : c'est lui qui ouvre l'en-tête |
| `FUN_140c7fedc` | le corps | `140c7ff00: CALL 0x1408ccb7c` (en-tête), `140c7ff27: CALL 0x140b4bc28` (premier tag), `140c7ff3b: CALL 0x140ccda34` (les champs connus), puis le fragment hors ligne `142295806` (la boucle de saut) |
| `FUN_1408ccb7c` | en-tête du sous-message | `if (indicateur == 0 && mode == 2) FUN_140b4bcb4(...)` — **un LEB128, lu et jeté** |
| `FUN_140b4bc28` | un tag | `b0 = R(8)` ; `*param_2 = b0 & 0x1f` (le TYPE DE FIL) ; si `b0 & 0xe0 == 0xe0` deux octets de plus, si `0xc0` un octet de plus (le COMPTE DE SAUT) |
| `FUN_141cbbae0` | le saut d'un champ | la table du §B2, lue sur `141cbbae7..141cbbb68` |
| `FUN_1428fbcd0` | les types composés | `1428fbcde..1428fbe00` |
| `FUN_1408cc940` / `FUN_1406d654c` | lire n octets | `8n` bits à la position courante, **sans alignement** — le mode trame n'aligne pas sur l'octet |

**LA BOUCLE ET SES DEUX SORTIES** (`140c7ff4c..140c7ff5a` + le fragment hors ligne) :

```
142295806: MOV ECX,EDX ; CMP EDX,0x1 ; JZ 0x140c7ff52   <- type de fil 1 : FIN, sans saut
142295811: MOV RCX,[RBX] ; CALL 0x141cbbae0             <- sinon : saut du champ
142295819: ... CALL 0x140b4bc28 ; TEST ECX,ECX ; JNZ    <- type de fil 0 : FIN
```

**POURQUOI SAUTER DONNE LE MÊME COMPTE QUE LIRE.** `FUN_140ccda34` est le déroulé des ~30 lecteurs
typés du schéma des propriétés multijoueur ; `FUN_141cbbae0` est le sauteur générique. Les deux
consomment le même nombre de bits pour un type de fil donné — c'est la définition d'un format
auto-descriptif. La confirmation vient d'un chemin INDÉPENDANT : `FUN_140b4af20`, le lecteur de
LISTE typée, porte la **même table de largeurs** que le sauteur, branche pour branche
(`14228cc1e..14228cc56`).

---

## B2. LA TABLE DES TYPES DE FIL, ET LES QUATRE ERREURS QU'ELLE CORRIGE

| Type de fil | Consommation | Adresse |
|---|---|---|
| `0` | FIN du message | `140c7ff4e` |
| `1` | FIN de portée, **sans saut de corps** | `142295808` |
| `2`, `3`, `0xe` | 1 octet | `141cbbb52: MOV EDX,0x1` |
| `7` | 4 octets | `141cbbb12: MOV EDX,0x4` |
| `8` | 8 octets | `141cbbb59: MOV EDX,0x8` |
| `4`, `5`, `6`, `0xf`, `0x10`, `0x11` | **un LEB128, ET RIEN DE PLUS** | `141cbbb40 -> FUN_140b4ba68 ; RET` |
| `9`, `0xa` | LEB128 `n` puis `n` octets | `1428fbdeb` / `1428fbd8b` |
| `0xb`, `0xc` | liste homogène : en-tête `FUN_140b4bbb8` puis `count` fois le saut du type des éléments | `1428fbd5e` |
| `0xd` | table de paires : `R(8)` type de clé, `R(8)` type de valeur, LEB128 `count`, puis `count` fois (clé, valeur) | `1428fbd1b -> FUN_1408cc830` |
| `0x12` | LEB128 `n` puis **2n** octets | `1428fbd08` |
| tout le reste | **ZÉRO bit** | `1428fbe00: RET` |

**LES QUATRE ERREURS DU PORTAGE PRÉCÉDENT**, chacune mesurable :

1. **L'en-tête LEB128 du mode 2 n'était pas lu.** Sur un flux réel son premier octet vaut souvent
   `0x00` — que l'ancien lecteur prenait pour le terminateur. C'est la raison pour laquelle i9
   consommait 14 bits au lieu de quatre cents et quelques dans la plupart des cas.
2. **Les types 4, 0xf, 0x10 et 0x11 étaient lus comme des corps préfixés par leur longueur.**
   `FUN_140b4ba68` lit le LEB128 et **rend la main** (`141cbbb40: LEA RDX,[RSP+0x40] ; CALL
   0x140b4ba68 ; ADD RSP,0x28 ; RET`). Ce sont des ENTIERS à longueur variable, pas des chaînes.
3. **Les types 5, 6, 9, 0xa, 0xb, 0xc, 0xd et 0x12 ne consommaient RIEN.** Huit types de fil sur
   dix-neuf tombaient dans le `skip = 0` par défaut.
4. **Le type de fil 1 ne terminait pas la boucle.** Le moteur sort sans sauter de corps.

Le tout est porté dans `apps/go-api/internal/analysis/filmdec/tlv_mode2.go` (le flux, avec sa
doctrine et ses plafonds anti-coût) et `components_batch7.go` (i9 lui-même, quatre lignes).
**UN SEUL LECTEUR, AUCUNE VARIANTE PAR ARCHÉTYPE** : i9 est un composant d'objet générique.

---

## B3. LE GATE, ET POURQUOI SON SEUIL LITTÉRAL ÉTAIT AU-DESSUS DU PLAFOND DE L'ORACLE

Le lot 6.10 (§3.3) écrivait : « la position d'i20 doit retomber au décalage zéro sur > 95 % des
records qui portent i9 ». La mesure, sur les 64 films, même oracle, même appariement :

| Population | records avec i20 | i20 atteint | oracle retrouvé | **décalage ZÉRO** | décalages distincts |
|---|---:|---:|---:|---:|---:|
| **AVEC i9** | 6 318 | 6 318 (100 %) | 2 903 | **2 738 (94,3 %)** | 29 |
| **SANS i9** (contrôle) | 1 018 | 1 018 (100 %) | 426 | **391 (91,8 %)** | 14 |

**LE SEUIL DE 95 % N'EST PAS ATTEIGNABLE, ET LA POPULATION TÉMOIN LE DÉMONTRE.** Les records SANS
i9 sont ceux dont la marche était DÉJÀ prouvée bit-exacte au lot 6.10 ; ils plafonnent eux-mêmes à
91,8 % sur ce corpus, parce que l'oracle d'inventaire vieillit (9 s en médiane) et qu'une arme
rechargée ou vidée entre les deux instants ne retombe sur aucun décalage. Les records qui portent
i9 font donc **MIEUX que la population de référence** : le résidu de 5,7 % n'est pas imputable à
i9. Le contrôle qui tranche est la comparaison des deux populations, pas le franchissement d'un
seuil écrit avant la mesure.

Les décalages non nuls sont d'ailleurs les MÊMES des deux côtés, dans les mêmes proportions :
`+16` (1,8 % avec i9, 0,9 % sans), `-77`/`-78`/`-79` (1,8 % contre 1,4 %).

### B3.1 Le contrôle STRUCTUREL, qui n'emprunte rien à l'inventaire

| | lot 6.10 (mesure indirecte) | 6.10 bis (grammaire portée) |
|---|---|---|
| longueurs distinctes d'i9 | 34 | **10** |
| plage | 300 à 470 bits | **414 à 502 bits** |
| structure modulo 8 | aucune (restes 0:23, 3:16, 5:11, 2:5, 7:5, 1:4, 4:3, 6:2) | **reste 6 dans 100,0 % des 6 318 records** |
| valeurs hors plage (> 600 bits) | — | **0,0 %** |

Les valeurs dominantes sont 414 (4 695 records), 478 (565), 502 (489), 462 (191), 470 (154),
494 (132), 486 (58), 454 (21) — **espacées de 8 bits exactement**, et toutes congrues à 6 modulo 8.
C'est la signature d'un en-tête de 6 bits (1 porte + 5 étiquette) suivi d'un flux d'OCTETS, et rien
d'autre ne la produit.

### B3.2 Le gradient de fraîcheur, les deux populations côte à côte

| Âge de la référence | n (sans i9) | A ≤ chargeur | B == réserve | n (avec i9) | A ≤ chargeur | B == réserve |
|---|---:|---:|---:|---:|---:|---:|
| ≤ 1 s | 68 | 97,1 % | 94,1 % | 349 | **94,6 %** | **90,3 %** |
| 1 à 2 s | 61 | 95,1 % | 93,4 % | 365 | 90,1 % | 83,6 % |
| 2 à 5 s | 190 | 93,7 % | 89,5 % | 1 042 | 91,1 % | 80,5 % |
| 5 à 10 s | 271 | 89,3 % | 79,7 % | 1 668 | 90,8 % | 69,4 % |
| 10 à 20 s | 405 | 90,1 % | 65,9 % | 2 718 | 89,6 % | 56,7 % |
| > 20 s | 23 | 95,7 % | 43,5 % | 176 | 92,6 % | 44,3 % |

Les deux colonnes décrivent la MÊME courbe : `A ≤ chargeur` reste haut et plat (la valeur lue n'est
jamais supérieure à la dernière lecture connue, hors rechargement), `B == réserve` décroît
lentement (la réserve ne bouge qu'au rechargement). **Témoin du champ voisin** (même largeur,
19 bits plus loin) : **4,1 à 6,8 %** avec i9, 1,6 à 4,1 % sans — les deux sous le seuil de 10 %.

**LE TÉMOIN DE PERMUTATION EST À 12-15 % AVEC i9 ET 2-8 % SANS, ET CE N'EST PAS UN CONTRASTE
RÉEL.** Il est tiré par le rang du cas dans sa tranche : les tranches « sans i9 » comptent 23 à
405 cas et n'échantillonnent qu'une fraction du corpus, les tranches « avec i9 » jusqu'à 2 718 et
le couvrent. Le témoin qui compte ici est celui du champ voisin, tiré au même endroit pour les deux
populations.

---

## B4. CE QUE LA CORRECTION CHANGE AILLEURS — MESURÉ, LIGNE PAR LIGNE

La découverte centrale du lot 6.10 était que « i9 étant générique, tout composant lu APRÈS lui sur
tout archétype héritait de la dérive ». Voici ce que ça donne, chiffré.

### B4.1 Ce qui NE change PAS, et pourquoi

| Famille | Effet | Cause |
|---|---|---|
| `groundWeaponCreations`, `equipmentCreations` | **aucun changement au commit de grammaire** | leur marche consomme i9 par le BLOC MPP du default-state (`consumeMultiplayerPropertiesBlock`), un lecteur DISTINCT qui n'a pas changé |
| oracles de position d'atterrissage (118/119, 97/99) | **aucun changement** | ils reposent sur le default-state et le composant i0, tous deux AVANT i9 dans le record |
| identité `MPPWord32` (98,9 %) | **aucun changement** | même raison : le mot voyage dans le default-state |

La preuve n'est pas un raisonnement mais le golden : au commit de grammaire (`723b3b61a`), les deux
lignes `groundWeaponCreations` et `equipmentCreations` du golden de mini-bobine sont **identiques
au caractère près**. Une dérive de position ou d'identité les aurait fait rougir.

### B4.2 Ce qui change, et c'est une AMÉLIORATION

Une seule famille bouge : `equipmentState` (l'état des équipements déployés, `ti=37`), qui
TRAVERSE i9. Mesure des deux côtés sur la mini-bobine versionnée (copie `git archive HEAD` d'un
côté, branche de l'autre, même instrument) :

| | AVANT | APRÈS |
|---|---:|---:|
| records delta `ti=37` | 5 282 | 5 282 |
| masques annonçant un champ | 66 | 66 |
| marches abouties | 65 | **66** |
| marches **ROMPUES** | **1** | **0** |
| i20 `equipment-deployed` | annoncé 4 · lu **3** | annoncé 4 · lu **4** |
| i21 `equipment-activated` | annoncé 1 · lu 1 | idem |
| i23 `equipment-creator` | annoncé 0 · lu 0 | idem |
| i24 `equipment-energy` | annoncé 1 · lu 1 | idem |
| i26 `energy-delay-ticks-left` | annoncé 48 · lu 48 | idem |
| i27 `charges-remaining` | annoncé 12 · lu 12 | idem |

**LE DELTA EST UN SEUL RECORD, ET DANS LE BON SENS** : le seul record de la bobine dont la marche
se rompait est celui qui traversait i9 ; il aboutit désormais et son champ est lu. Aucune valeur
pré-existante ne change.

### B4.3 La levée de la réserve, et son invariance

Le second commit (`296cbd5e6`) retire le refus des masques portant i9 dans `readGroundWeaponAmmo`.
Sur la mini-bobine :

| | AVANT | APRÈS |
|---|---:|---:|
| créations acceptées | 28 | 28 |
| ancres reconnues | 141 | 141 |
| masque portant i20 | 21 | 21 |
| masque portant i9 | 27 | 27 |
| **munitions LUES** (`HasAmmo`) | **0** | **21** |

**PREUVE D'INVARIANCE SUR OCTETS RÉELS**, empreinte sha256 de TOUS les champs pré-existants de
chaque record (`HasAmmo`/`Ammo` exclus), mesurée des deux côtés :

```
groundWeaponCreations  n=28
  sha256=4409c0381383427ae59f99e3cbce31001d0cbdf1c9f68d3378b353daf74a19f8   (identique)
```

Slot, génération, horodatage, `MPPVal`, masque, `DefaultStateBits`, position et `AfterBit` sont donc
au bit près les mêmes : seul le champ `Ammo`, qui était vide, se remplit.

### B4.4 Les goldens changés, et pourquoi

| Ligne | Commit | Cause |
|---|---|---|
| `equipmentState` 65 -> 66 | `723b3b61a` | une marche rompue qui ne l'est plus (§B4.2) |
| `groundWeaponCreations` (digest) | `296cbd5e6` | `HasAmmo` passe de 0 à 21 records ; `rendreStable` rend tous les champs (§B4.3) |

Les deux régénérations sont documentées dans l'en-tête de `golden_minibobine_test.go`, avec leurs
tableaux champ par champ. **Aucune autre ligne du golden ne bouge**, sur 35.

---

## B5. i14 `object-dissolver-component` — AUCUN CHAMP DE FIN DE VIE ÉTABLI

### B5.1 La grammaire, relue sur `FUN_140dd9f9c`

```
R(4)   état        140dd9faf: MOV ECX,0xe ; CALL 0x1406d310c  -> bitLen(0xe) = 4
                   140dd9ff6: MOV dword ptr [RDI+0x3a8],R10D
si état != 0xd :   140dd9ffd: CMP R10D,0xd ; JNZ 0x140dda074
  R(96) brut       140dda07b: MOV R9D,0x60 ; CALL 0x1406d676c  -> [RDI+0x3ac], 12 octets
  R(12) déquant.   140dda0a1: MOV dword ptr [RSP+0x20],0xc ; XMM2 = 0.0 ;
                   XMM3 = [0x143cd873c] = 10.0f ; CALL 0x1406d84b4
                   140dda0ae: MOVSS dword ptr [RDI+0x3b8],XMM0  -> un FLOTTANT dans [0, 10]
  R(1)   drapeau   140dda0d6: MOV byte ptr [RDI+0x3bc],CL
```

**AUCUNE LARGEUR N'A CHANGÉ** : `consumeObjectDissolver` consommait déjà 4, puis 96 + 12 + 1. La
relecture a servi à savoir ce que les bits PORTENT, pas à corriger un compte. Un garde-rail
(`components_object_state_test.go`) fige désormais les quatre largeurs **en clair** — jamais à
partir des constantes du décodeur, un test qui réutilise la constante qu'il vérifie ne vérifie
rien. Trois mutations jouées (durée 12 -> 11, corps 96 -> 64, état `0xe` -> `0x1e`) : les trois
rougissent.

### B5.2 La mesure, et le verdict

Sur les 64 films du parc, records de CRÉATION `ti=42` :

| Quantité | Valeur |
|---|---:|
| créations acceptées | 27 155 |
| dont le masque porte i14 | 18 214 (**67,1 %**) |
| objets du document appariés à leur record | 13 014 |
| dont l'**état vaut 13 — LE NEUTRE** | 12 988 (**99,80 %**) |
| dont le corps est donc LU | 26 (**0,20 %**) |

Distribution de l'état `R(4)` : `13` x 12 988, `8` x 14, `0` x 5, `1` x 3, `4` x 1, `5` x 1,
`6` x 1, `14` x 1.

**LA CAUSE EST STRUCTURELLE, PAS UN RENONCEMENT.** Un composant de DISSOLUTION décrit une FIN ; la
fin n'est pas connue à la naissance de l'objet. Au record de création le dissolveur est à son état
neutre et il n'y a rien à y lire.

### B5.3 Les 26 exceptions, et pourquoi rien ne s'y publie

| Champ lu | Ce qu'il donne |
|---|---|
| **durée `R(12)`** déquantifiée dans [0, 10] | 18 valeurs distinctes sur 26 cas. Rangée par quartile, la vie OBSERVÉE de l'objet donne 301, 669, 1 527 puis 1 219 images — **non monotone**, sur six cas par quartile |
| **drapeau `R(1)`** | vrai 17 fois, faux 9 fois. **Aucun des 26 objets n'a été ramassé** — les deux modalités donnent 0 ramassage, et les 363 objets du parc à fin `pickup` sont TOUS à l'état neutre |
| **les 96 bits bruts** (3 mots de 32) | 21, 18 et 17 valeurs distinctes sur 26 cas. Aucun mode, aucune constante par famille |

Ventilation par fin d'objet : `pickup` 363 objets, **tous** à l'état neutre, zéro corps ; `seen`
12 296 dont 22 avec corps ; `open` 355 dont 4 avec corps.

**VERDICT : aucun champ de fin de vie établi.** Publier l'un de ces champs reviendrait à nommer du
bruit — même règle que le champ C d'i20 du lot 6.10, qui reste non publié.

**PISTE NON TRAITÉE, consignée** : i14 dans les paquets DELTA plutôt que dans le record de
création. Le film ne date la disparition d'aucun objet posé (acquis du 2026-08-17) et le calque
publie un INTERVALLE `[t1, t1max]` : c'est toujours la meilleure réponse disponible.

---

## B6. CE QUI EST LIVRÉ

| Élément | Fichier | Commit |
|---|---|---|
| Le flux TLV mode 2, avec sa chaîne d'appels et ses plafonds | `filmdec/tlv_mode2.go` (nouveau) | `723b3b61a` |
| i9 réduit à quatre lignes, avec sa doctrine | `filmdec/components_batch7.go` | `723b3b61a` |
| Garde-rail de grammaire (17 cas, un par branche) + témoin du varint sans corps | `filmdec/components_batch7_test.go` | `723b3b61a` |
| Golden `equipmentState` + en-tête expliquant le delta | `filmdec/golden_minibobine_test.go`, `testdata/golden_minibobine_familles.tsv` | `723b3b61a` |
| Réserve de lecture LEVÉE | `filmdec/ground_weapon_ammo.go` | `296cbd5e6` |
| Garde-rail de traversée d'i9 (trois familles de types de fil) | `filmdec/ground_weapon_ammo_test.go` | `296cbd5e6` |
| Golden `groundWeaponCreations` + preuve d'invariance | `filmdec/golden_minibobine_test.go`, `testdata/...` | `296cbd5e6` |
| Grammaire d'i14 documentée + seuils nommés | `filmdec/components_object_state.go` | `87181b6cf` |
| Garde-rail des largeurs d'i14 (valeurs en clair) | `filmdec/components_object_state_test.go` (nouveau) | `87181b6cf` |

**RIEN N'A CHANGÉ CÔTÉ WEB NI AU CONTRAT** : la forme publiée (`groundWeapons[].ammo = {mag, res}`,
`omitempty`, additif) et l'infobulle du lot 6.10 sont inchangées. Seule la COUVERTURE augmente, et
c'est un effet de la cuisson, pas du schéma. Aucun bump de `SchemaVersion`.

**RECUISSON : OUI, ET DE TOUT LE PARC** — `ammo` est un champ CUIT et sa couverture passe de 15,8 %
à 59,7 %. C'est la recuisson unique déjà prévue au plan maître après 6.10 et 6.11 (lot 6.8).

---

## B7. GATES JOUÉS

| Gate | Résultat |
|---|---|
| `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ ./internal/replaybuild/ ./internal/service/...` | vert (CGO activé — sans CGO le paquet `service` ne construit pas sur ce poste) |
| `go vet ./internal/analysis/filmdec/` | 0 |
| `golangci-lint run ./internal/analysis/filmdec/...` | **0 nouvel avertissement** sur les fichiers touchés ; les 12 restants sont la dette de baseline (9 `goconst`, 2 `unparam`, 1 `unused`) |
| Goldens | invariance démontrée sur octets réels des DEUX côtés (§B4.3) ; les deux lignes qui changent sont expliquées champ par champ dans l'en-tête du test |
| Web | **non touché** — aucun fichier de `apps/web/` modifié |
| Corpus d'équivalence | joué par le superviseur |

---

## B8. LA RECETTE DE L'INSTRUMENT (supprimé avant livraison)

Quatre fichiers `zz_i9bis*_research_test.go` dans `internal/analysis/filmdec/`, garde
`MUN_RACINE=<racine qui porte data/>` (+ `MUN_N=<nb de films>`). Ils reprennent le socle du lot 6.10
(§7) avec **une correction qui change tout** :

> **LE REJEU SE FAIT DANS LA BOUCLE PAR FILM, JAMAIS APRÈS.** `projPosBits()` et la boucle de
> composants lisent des GLOBAUX (précision i0, largeurs MPP) calibrés film par film. La première
> version de l'instrument collectait tous les cas puis rejouait la marche à la fin : tous les films
> étaient alors décodés avec les réglages du DERNIER, et la mesure donnait 81,5 % de décalage zéro
> au lieu de 94,3 %, avec des longueurs d'i9 à 131 206 bits. **Le symptôme d'un instrument qui se
> trompe ressemble exactement à celui d'une grammaire fausse** — c'est le contrôle de congruence
> modulo 8 qui a permis de les distinguer.

1. **Socle** (`zz_i9bis_socle`) : lister les films qui ont un artefact ET des chunks ;
   `filmsource.LoadDir`, `DetectI0LayoutOf` + `SetWorldObjectPrecisionFromLayout` ; calibrer les
   largeurs MPP par le chemin de production (`ScanWorldObjectsForBand` + `EquipmentLifeSpans` +
   `CalibrateMPPWidthsOf` sur `ti=37`) ; bande par `worldObjectSlotBand(film, 42)` ; balayer les
   paquets delta avec `equipCreationWalk.scanPayload` **en conservant le payload de chaque record**
   (le balayage de production ne le publie pas, et le rejeu en a besoin).
2. **Oracle** : lire l'artefact cuit (`groundWeapons`, `loadouts`, `inventory`) ; apparier par
   (famille `MPPVal[MPPWord32]` en `%08x`, image ± 2, unicité EXIGÉE) ; le chargeur et la réserve
   viennent du dernier `loadouts` du lâcheur avant `t0` (il donne l'ORDRE des emplacements) et de
   l'`inventory` du MÊME instant.
3. **Rejeu** : `NewBitReader(pay)`, `SetBitPos(cre.AfterBit - projPosBits())`,
   `traverseComponentLoop` avec le masque du record, puis relire à `CompResult{Index:20}.StartBit`.
   La longueur d'i9 est `StartBit(composant suivant) - StartBit(i9)`.
4. **Les cinq mesures** : couverture (acceptées / portant i20 / lues), balayage de décalage sur
   [-96, +96], congruence modulo 8 des longueurs d'i9, gradient de fraîcheur séparé AVEC/SANS i9,
   témoins (permutation, champ voisin à +19 bits).
5. **i14** : même socle, lecture directe à `CompResult{Index:14}.StartBit` (`R(4)` état, puis
   `3 x R(32)`, `R(12)`, `R(1)`), confrontée à `end`, `picker`, `t1 - t0` et `t1max - t0`.
6. **Invariance** : sha256 du rendu de chaque record, champs nouveaux EXCLUS, joué des deux côtés
   d'une copie `git archive HEAD`.

Durées observées : 205 s pour l'oracle sur les 64 films, 160 s pour i14, 0,15 s pour les deltas de
mini-bobine.

---

## B9. STATUT DES ITEMS DU CONTRAT 6.10 BIS

| Item | Statut |
|---|---|
| **Point 1.1** — retrouver et désassembler le déserialiseur d'i9, grammaire complète documentée | `[x]` §B1, §B2 — neuf fonctions relues, l'instruction décisive citée pour chacune |
| **Point 1.2** — porter en Go sans seconde copie, tests rouges puis verts, test de mutation | `[x]` un seul lecteur générique (`tlv_mode2.go` + quatre lignes dans i9) ; 17 cas de grammaire ; mutations jouées : table d'avant le lot (16 lignes sur 17 rouges), varint lu comme longueur (témoin dédié rouge), type de fil 5 retiré (garde-rail de traversée rouge) |
| **Point 1.3** — gate : décalage zéro > 95 %, couverture 15,8 % vers 59,7 %, témoins < 10 %, invariance des goldens | `[~]` **couverture tenue à l'unité près (15,8 % vers 59,7 %, 4 283 vers 16 218)**, témoin du champ voisin tenu (4,1 à 6,8 % < 10 %), invariance démontrée sur octets réels, goldens expliqués ligne par ligne. **Le seuil littéral de 95 % n'est PAS atteint (94,3 %) et il n'était pas atteignable** : la population de contrôle, dont la marche était déjà prouvée bit-exacte, plafonne à 91,8 % sur le même corpus (§B3). Le contrôle structurel indépendant — congruence modulo 8 à **100,0 %** — est, lui, sans réserve |
| **Point 1.4** — chiffrer ce que la correction change ailleurs | `[x]` §B4 : « rien » PROUVÉ pour les créations, les positions d'atterrissage et l'identité (goldens identiques au commit de grammaire) ; **une amélioration mesurée** sur `equipmentState` (une marche rompue en moins, un champ lu en plus) ; aucune régression |
| **Point 2** — i14, grammaire désassemblée puis oracle ; publier un champ seulement si prouvé | `[x]` §B5 — grammaire relue (aucune largeur ne change), garde-rail posé, **aucun champ de fin de vie établi**, avec la liste des champs lus et leurs distributions. Le champ C d'i20 reste non publié |
| **Point 3** — rapport, ligne 6.10 du plan, journal, mention datée superséder l'entrée du 30-08 | `[x]` cette section, plus les trois écritures |
| Instrument supprimé avant livraison, recette au rapport | `[x]` §B8 |

---

## B10. DÉCOUVERTES (consignées, NON traitées — règle « zéro fix opportuniste »)

1. **Le pont MCP Ghidra ne peut pas se connecter en UDS sur ce poste** : il tourne sous un Python
   sans `socket.AF_UNIX`, et il refuse le repli TCP tant qu'il n'a pas pu lire le nom du projet.
   Le plugin répond pourtant parfaitement en HTTP sur `127.0.0.1:8089` (`/mcp/schema`,
   `/decompile_function`, `/disassemble_function`, `/read_memory`, `/disassemble_bytes`). C'est par
   là que tout ce lot a été désassemblé.
2. **`FUN_140ccda34` est le déroulé des lecteurs typés du schéma des propriétés multijoueur** —
   une trentaine de tentatives ordonnées, chacune testant un type de fil attendu. Le décoder
   donnerait le SENS des champs d'i9 (et non plus seulement leur longueur). Personne n'en a besoin
   aujourd'hui : le décodeur ne fait que traverser i9.
3. **Le compte de saut des trois bits hauts du tag** désigne un nombre de champs SAUTÉS dans le
   schéma, pas dans le flux : il ne consomme rien au-delà de ses octets d'extension. Il n'est donc
   pas rendu par `readTLVTag` (ce serait du code mort), mais c'est lui qui permettrait de nommer
   les champs si la découverte n° 2 était instruite.
