# RAPPORT — Les munitions sont-elles SUR L'OBJET au sol ? (lot 6.6, 2026-09-10)

> Branche `wt/armes-au-sol-calque`, worktree `LevelUp-wt-armes-au-sol-calque`.
> Aucune base DuckDB ouverte, aucune recuisson, aucun serveur redémarré. Les artefacts cuits
> (JSON, schéma 51) et les chunks de film sont lus comme des fichiers.
> Demande utilisateur (10-09 au soir) : « lire les munitions SUR L'OBJET » plutôt que de les
> joindre depuis l'inventaire daté.

---

## 0. LES TROIS LIGNES QUI COMMANDENT LE RESTE

1. **La « liste de chargeurs » du record de création (point 5) n'est PAS un compte de
   munitions.** 10 613 mots de 32 bits lus sur 20 990 créations, **6 609 valeurs DISTINCTES**,
   **minimum 98**, **2 valeurs sous 1024 (0,02 %)** — et **0 égalité avec le chargeur du lâcheur
   sur 1 524 comparaisons** (0 aussi avec la réserve, 0 en « inférieur ou égal »). Un même mot
   revient sur une même famille d'arme pendant que le chargeur, lui, varie : c'est un HANDLE.
2. **Les deux feuilles voisines du même record ne le sont pas davantage.** Le `R(7)` du point 4
   égale le chargeur **10 fois sur 6 477 (0,2 %)** et vaut 0 dans 74 % des records ; le `R(12)`
   du point 3, **0 fois sur 6 477** (3 égalités avec la réserve).
3. **En revanche le point 6 est identifié, et il ne dit pas les munitions : il dit le
   PROPRIÉTAIRE.** La référence d'entité de 5 bits vaut l'**index de joueur du lâcheur**
   (`roster[].filmIndex`) dans **6 551 cas sur 6 931 testés, soit 94,5 %** — et elle est
   transmise pour **92,5 % des objets `spawned`**, ceux que le calque laisse aujourd'hui
   anonymes.

**Décision d'étape 2 qui en découle : REPLI.** Pas de champ `ammo` cuit, pas de recuisson du
parc. L'infobulle affiche la dernière lecture d'inventaire du lâcheur, **datée**, jointe à la
lecture depuis ce que l'artefact sert déjà.

---

## 1. MÉTHODE

### 1.1 Ce qui a été instrumenté

Trois feuilles du default-state de l'archétype `ti=42` (`filmdec/default_state_ti42.go`) étaient
lues et jetées. Elles ont été rendues observables **sans changer un seul bit** :

| Point | Grammaire | Instrumentation |
|---|---|---|
| 3 | `R(12)` inline -> `dst+0x60` | valeur captée par une sonde de paquet temporaire |
| 4 | `R(7)` via `FUN_1406d84b4` (déquant. flottante) -> `dst+0x64` | idem |
| 5 | `consumeWeaponMagazineList` (`components_object.go:448`) : porte `R(1)` ; si 0 -> `[R(1) ; si 1 -> R(32)]` ; si 1 -> `n=R(4)` puis n × `[R(1) ; si 1 -> R(32)]` | réceptacle passé en paramètre au lecteur PARTAGÉ (aucune seconde copie ; `consumeWeaponStateTypeInfoVariant` passait `nil`) |
| 6 | `ECS_ReadEntityRefIndex5` = `consumeGate0R(br, 5)` | déroulé à plat, publié par `publishEquipmentCreation(EquipCreationRef, ...)` — le champ que `ti=37` utilise déjà |

**PREUVE D'INVARIANCE DE LA GRAMMAIRE, sur octets réels.** L'empreinte sha256 de TOUS les champs
préexistants (slot, génération, chunk, paquet, horodatage, position de bit, MPP, position,
masque, `DefaultStateBits`, `AfterBit`) de chaque record de création de la mini-bobine
versionnée, mesurée **des deux côtés du changement** (copie `git archive HEAD` d'un côté, branche
de l'autre, même test) :

```
groundWeaponCreations  n=28  ancres=141  acceptees=28
  sha256=9b2dbd3166d9cf62c2f0e449deaa2ff7ffb1b90685c87b052034a901b8d27a08   (identique)
equipmentCreations     n=38  ancres=170  acceptees=38
  sha256=0f8c60fa57e4286a5d851f149e0fbf3d0c2b248f1986e63ae4941b29c8690984   (identique)
```

Le golden `golden_minibobine_familles.tsv` change d'une ligne (`groundWeaponCreations`) et
seulement parce que `rendreStable` rend TOUS les champs, `HasRef`/`Ref` compris : l'explication
est écrite dans l'en-tête de `golden_minibobine_test.go`.

### 1.2 Le corpus et la jointure

- **61 films d'arène** (roster ≤ 16) sur les 64 artefacts du parc local ; les 3 films de Grand
  combat sont comptés à part et **jamais mêlés aux pourcentages**. 0 film écarté.
- Pour chaque film : chunks lus depuis `data/cache/film_chunks/<8hex>/`, largeurs d'axe prises
  du film (`DetectI0LayoutOf`), largeurs du bloc MPP calibrées par le chemin de production
  (`decodeFilmPlacements`), puis `decodeFilmPadScan(..., groundWeaponArchetype())` — le MÊME
  câblage que la cuisson.
- **Les positions décodées n'ont aucun sens ici** (bornes monde synthétiques : le catalogue de
  cartes ne se résout pas hors ligne sans la base) et ne sont **pas utilisées** : la mesure ne
  porte que sur des slots, des instants, des identités et des mots. La validité du décodage est
  contrôlée autrement — le nombre de créations acceptées est comparé à
  `coverage.groundWeapons.objects` de l'artefact, avec écart franc = film écarté (0 cas).
- **Appariement création ↔ objet publié** : par (famille `%08x`, image de `t0`), l'image étant
  reconstruite avec l'horloge du document (`origine = originMs × 1000 + ScanClockOrigin(film)`,
  pas `frameIntervalMs × 1000`). **9 219 objets lâchés appariés sur 9 794 (94,1 %)** ; 512
  ambigus (deux objets de même famille nés à la même image) écartés.
- **Inventaire du lâcheur** : dernière lecture NON VIDE du slot avant `t0` (`inventoryAt`
  reproduit : bornée à la vie, report sur la dernière lecture pleine), emplacement indexé par
  l'ordre de `loadouts[].w` du MÊME instant.

### 1.3 L'instrument

`internal/analysis/replay/munitions_objet_research_test.go`, jetable, gardé par
`MUNITIONS_OBJET=1` (+ `MUNITIONS_RACINE` pour désigner la racine qui porte `data/`), **supprimé
après mesure** — précédent E0 et rapport 6.3. Durée : 193 s pour les 64 artefacts.

---

## 2. RÉSULTATS — LE POINT 5 (LISTE DE CHARGEURS)

### 2.1 La forme

| Grandeur | Valeur |
|---|---|
| Créations `ti=42` acceptées (61 films) | 20 990 |
| dont au moins un mot transmis | **6 311 (30,1 %)** |
| Branche empruntée | mot unique 18 432 · liste 2 558 |
| `n` annoncé sur la branche liste | 0 : 313 · 2 : 383 · 3 : 157 · 4 : 133 · … · 15 : 133 |
| Mots présents par objet | 0 : 14 679 · 1 : 4 888 · 2 : 361 · 3 : 324 · 4 : 259 · … · 12 : 2 |

**Premier fait qui suffirait à lui seul :** 72,2 % des objets LÂCHÉS ne portent AUCUN mot. Un
champ « munitions au lâcher » absent pour sept armes sur dix n'est pas un champ de munitions.

### 2.2 L'espace des valeurs

| Grandeur | Valeur |
|---|---|
| Mots lus, toutes créations | 10 613 |
| Valeurs **distinctes** | **6 609** |
| Mots **< 1024** | **2 (0,02 %)** |
| Minimum observé | **98** |
| Un mot vu sur 1 seule famille d'arme | 6 458 / 6 609 |
| Mots distincts par famille | 1 : 1 258 · 2 : 321 · 3 : 302 · … · 54 : 1 |

Un compte de munitions vit entre 0 et ~60 avec des modes marqués (chargeur plein). Ici : des
valeurs quasi uniques, aucune sous 98, chacune attachée à une famille. La porte elle-même le
disait déjà — la table de primitives de `components_object.go` nomme `FUN_14080d69c`
« optional-**handle** gate ».

### 2.3 La confrontation à l'inventaire

**1 524 comparaisons** (objets lâchés, appariés, porteurs d'un mot, avec une lecture exploitable
du lâcheur portant un chargeur) :

| Hypothèse | Résultat |
|---|---|
| mot == chargeur | **0 (0,0 %)** |
| mot ≤ chargeur | **0 (0,0 %)** |
| mot == réserve | **0 (0,0 %)** |
| **N'IMPORTE LEQUEL** des mots présents == chargeur | **0 (0,0 %)** |
| idem == réserve | **0 (0,0 %)** |
| écart mot − chargeur | p10 = 320 323 476 · p50 = 1 353 951 962 · p90 = 4 038 578 764 |

**Contre-exemples — ils disent la CAUSE**, pas seulement l'échec (film, image, famille, lâcheur,
mot, chargeur, réserve, tirs entre la lecture et le lâcher) :

```
0a44c6cc frame=312  w=b533957e lacheur=514 mot=1353951992 mag=30 res=30 tirs=3
0a44c6cc frame=1110 w=b533957e lacheur=523 mot=1353951992 mag=30 res=30 tirs=8
0a44c6cc frame=327  w=48c19d2d lacheur=513 mot=1739183068 mag=25 res=75 tirs=6
0a44c6cc frame=518  w=48c19d2d lacheur=522 mot=1739183068 mag=25 res=75 tirs=0
0a44c6cc frame=327  w=84bd29ed lacheur=513 mot=320323477  mag=7  res=14 tirs=6
0a44c6cc frame=935  w=84bd29ed lacheur=524 mot=320323477  mag=4  res=7  tirs=0
0a44c6cc frame=1110 w=84bd29ed lacheur=523 mot=320323477  mag=7  res=5  tirs=8
```

La lecture est directe : **le mot est CONSTANT par famille d'arme** (`84bd29ed` -> 320 323 477
trois fois, `b533957e` -> 1 353 951 992 deux fois) **pendant que le chargeur varie** (7, 4, 7).
C'est l'identifiant du chargeur de cette arme, pas son contenu.

**VERDICT : RÉFUTÉ.** Hypothèses (a) « un mot = munitions en chargeur ≤ lecture précédente » et
(b) « un mot = réserve » : réfutées, 0 sur 1 524. Hypothèse (c) « cohérence avec les tirs » :
sans objet, il n'y a pas de grandeur à corréler.

---

## 3. RÉSULTATS — LES POINTS 3 ET 4 (les meilleurs candidats par leur LARGEUR)

Ils n'étaient pas au contrat, et c'est précisément pourquoi il fallait les mesurer : `R(7)` a la
forme d'un compte de munitions (0..127) et `R(12)` celle d'une réserve (0..4095). Ils sont lus
INCONDITIONNELLEMENT, donc leur échantillon est le plus large du lot : **6 477 comparaisons**.

| Hypothèse | Résultat |
|---|---|
| `R(7)` == chargeur | **10 (0,2 %)** |
| `R(7)` ≤ chargeur | 6 437 (99,4 %) — mais l'écart médian vaut **−25** |
| `R(7)` == réserve | 337 (5,2 %) |
| `R(12)` == chargeur | **0 (0,0 %)** |
| `R(12)` == réserve | 3 (0,0 %) |
| écart `R(7)` − chargeur | p10 = −36 · p50 = −25 · p90 = −3 · min = −250 · max = 115 |
| valeurs de `R(7)` | **0 : 15 539 (74 %)**, puis 36 : 171 · 64 : 122 · 13 : 109 · 8 : 106 … |
| valeurs de `R(12)` | 360 : 94 · 206 : 91 · 207 : 90 · 269 : 75 · 538 : 73 … |

Le « 99,4 % ≤ chargeur » est un piège de dénominateur, pas un résultat : `R(7)` vaut 0 dans trois
quarts des records, et zéro est inférieur à tout. Les 0,2 % d'égalité sont du hasard.
`FUN_1406d84b4` est le lecteur de flottant déquantifié — `R(7)` est une fraction, pas un compte.

**VERDICT : RÉFUTÉS tous les deux.**

---

## 4. RÉSULTAT — LE POINT 6 : PAS LES MUNITIONS, LE PROPRIÉTAIRE

Hypothèse (d) du contrat : « le point 6 vaut *aucun propriétaire* pour les objets lâchés et autre
chose pour les `spawned` ». **Réfutée dans sa forme, confirmée au-delà dans son fond.**

| Grandeur | Valeur |
|---|---|
| Créations `ti=42` avec référence transmise | 15 021 / 20 990 |
| Objets `dropped` appariés : avec référence | **8 717 / 9 219 (94,6 %)** |
| Objets `spawned` appariés : avec référence | **2 395 / 2 588 (92,5 %)** |
| Valeurs observées | 0..7 quasi uniformes (972 à 1 107 chacune), puis 8..13 en décroissance, plus la sentinelle « absente » |
| **Référence == `roster[].filmIndex` du lâcheur** | **6 551 / 6 931 testés = 94,5 %** |

Les 380 désaccords (5,5 %) ne sont pas ventilés : ils peuvent venir de l'appariement par
(famille, image), de la règle d'attribution du lâcheur elle-même (`gwPadsClass`, 200 ms / 1,5 m)
ou d'un vrai propriétaire différent du dernier porteur. La mesure n'a pas cherché à trancher.

**CE QUE ÇA VAUT, ET CE QUE ÇA NE FAIT PAS ICI.** Le rapport 6.3 avait mesuré et écarté l'option
E (nommer le lâcheur volontaire en joignant `weaponChanges`) : **12,7 % de rendement**. Le point
6 offre la même information sur **92,5 %** des objets `spawned` — et sans jointure. C'est un lot
à part : le publier au document exige un champ nouveau ET une recuisson du parc, l'un et l'autre
hors du périmètre de 6.6 (règle « zéro fix opportuniste »). **Le décodeur cesse simplement de
jeter la valeur** ; la publication au document attend une décision.

---

## 5. ÉTAPE 2 — CE QUI EST LIVRÉ

**Le repli, tel que le plan l'a écrit.** Aucun champ `ammo` cuit, aucun bump de `SchemaVersion`,
**aucune recuisson du parc n'est nécessaire** — la jointure se fait à la lecture, côté web,
depuis `inventory` / `loadouts` / `groundWeapons`, tous les trois déjà servis.

| Élément | Fichier |
|---|---|
| Jointure + phrase (PUR) | `apps/web/src/features/match-replay/model/groundWeaponAmmo.ts` |
| Tests (9) + trois mutations jouées | `.../model/groundWeaponAmmo.test.ts` |
| Câblage du survol | `.../layers/useReplayGroundWeapons.ts` (champ `ammoLine`) |
| Assemblage de l'infobulle | `.../ui/ReplayGroundWeaponTip.tsx` (+ son test, neuf) |
| Strings FR + EN | `.../i18n/i18n.ts`, contrat dans `.../i18n/i18nContract.ts` |

La ligne affichée : `≈ 12 au chargeur et 24 en réserve, lues 9.0 s avant le lâcher` (ou
`≈ 12 munitions au chargeur, lues 9.0 s avant le lâcher` quand la réserve n'est pas lue).
**Jamais un nombre nu** : `groundWeaponAmmoAt` rend la valeur AVEC son âge, et
`groundWeaponAmmoLine` les écrit ensemble — les séparer demanderait de modifier les deux.

**Le seuil est une constante nommée et testée** : `GROUND_WEAPON_AMMO_MAX_AGE_MS = 20 000`, soit
un intervalle d'image-clé. Au-delà, la lecture retenue n'est plus « l'image-clé précédente » mais
un TROU dans le recensement, et le fragment est omis.

**Les cinq refus**, tous couverts par un test : arme `spawned` (pas de lâcheur mesuré), arme
absente du relevé de loadout de cet instant (l'index servirait les munitions d'une AUTRE arme),
emplacement sans chargeur (arme à jauge — publier 0 dirait « chargeur vide »), lecture plus
vieille que le seuil, lecture À VENIR.

---

## 6. STATUT DES ITEMS DU CONTRAT

| Item | Statut |
|---|---|
| 1. `consumeWeaponMagazineList` rend ses valeurs, captées à la création `ti=42`, sans casser `consumeWeaponStateTypeInfoVariant` | `[x]` fait (réceptacle en paramètre, une seule copie du lecteur, test d'égalité de largeur avec/sans réceptacle) **puis RETIRÉ après mesure** — la sémantique étant réfutée, le garder aurait laissé un type et un champ exportés sans lecteur (« pas de au cas où », CLAUDE.md n° 7). La recette est au §1.1, reproductible en une vingtaine de lignes |
| 1 bis. Point 6 capté | `[x]` fait **et conservé** : il publie dans `EquipCreationRef`, champ qui existe déjà pour `ti=37` — aucun type ni champ nouveau — et sa sémantique est PROUVÉE (§4) |
| 2. Instrument hors ligne, hypothèses (a) (b) (c) (d) chiffrées, distributions, contre-exemples identifiés | `[x]` fait — §2, §3, §4. (c) sans objet, motivé au §2.3 |
| 3. Rapport court avec le verdict et la décision d'étape 2 | `[x]` ce fichier |
| Étape 2 — publication `ammo` si prouvé | `[!]` NON TRAITÉ, **et c'est le verdict** : la sémantique est réfutée (§2, §3). Publier un champ dont on ne connaît pas le sens est explicitement interdit par le contrat |
| Étape 2 — repli daté | `[x]` livré, §5 |
| TDD, test de mutation, golden expliqué | `[x]` — rouge d'abord des deux côtés ; mutation de polarité jouée côté Go (le balayage tombe de 28 records à 1), trois mutations jouées côté web ; golden expliqué dans son en-tête ET ici |
| Empreintes `filmdec` inchangées, démontré | `[x]` §1.1 — sha256 identiques des deux côtés du changement |

---

## 7. DÉCOUVERTES (consignées, NON traitées — règle « zéro fix opportuniste »)

1. **`GroundWeapon.W` : commentaire faux, et il coûte une jointure.**
   `apps/go-api/internal/analysis/replay/document_ground_weapon_items.go:78` annonce « MÊME
   convention et MÊME espace d'identifiants que `Loadout.W` ». **C'est faux** :
   `groundWeapons[].w` est écrit `%08x` (nu, minuscules), `Loadout.W` est écrit `0x%08X`.
   Mesuré sur `0891225f` : les deux ensembles portent les MÊMES 8 familles et leur intersection
   brute est **VIDE**. C'est la même divergence que celle réparée côté web au lot 6.5 (item 0),
   mais côté SERVEUR le commentaire, lui, n'a pas été corrigé — anti-pattern « doc inversée » du
   CLAUDE.md. Le repli livré ici la contourne par `weaponLabelKeyOf` ; le commentaire reste à
   corriger.
2. **Le point 6 nomme le propriétaire des armes `spawned` à 92,5 %** (§4) — de quoi remplacer
   l'option E du rapport 6.3 (12,7 %, écartée) et lever l'anonymat des 21,4 % d'objets que le
   calque affiche aujourd'hui sans lâcheur. Coût : un champ au document + une recuisson du parc.
3. **Le canal de l'ARME PORTÉE lit la même liste de chargeurs, dans les paquets DELTA.**
   `consumeWeaponStateTypeInfoVariant` (`components_object.go:390`) appelle
   `consumeWeaponMagazineList` sur l'arme en main, à chaque enregistrement delta — pas seulement
   aux images-clés. Le mot y étant un handle (§2), il n'y a pas de munitions à y prendre non
   plus ; mais ce canal reste, lui, le seul endroit du film où une grandeur d'arme portée est
   transmise AU CHANGEMENT. Non mesuré ici.
4. **512 objets lâchés sur 9 794 (5,2 %) sont ambigus à l'appariement** (même famille, même
   image). Ce n'est un problème que pour un instrument externe ; la cuisson, elle, tient
   l'objet par sa clé de vie `(slot, gen)`.

---

## 8. RÉFÉRENCES

- Contrat du lot : `.ai/PLAN_MASTER_2026-09-09.md`, ligne `| 6.6 |`.
- Amont : `.ai/V7.5/RAPPORT_ARMES_AU_SOL_2026-09-10.md` (§4 « les munitions au lâcher », option
  E du §5), `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` (§2 bis).
- Code : `apps/go-api/internal/analysis/filmdec/default_state_ti42.go`,
  `components_object.go:448`, `ground_weapon_creation.go`, `equipment_creation.go`,
  `golden_minibobine_test.go` ; côté web `apps/web/src/features/match-replay/model/groundWeaponAmmo.ts`.
