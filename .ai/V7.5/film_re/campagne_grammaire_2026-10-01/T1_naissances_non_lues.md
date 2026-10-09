# T1 — Naissances non lues (campagne grammaire, phase 1, 2026-10-01)

> Piste T1 du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (étape 2). Lecture seule : Ghidra
> (`HaloInfinite.exe`, serveur HTTP 127.0.0.1:8089, uniquement `decompile_function`,
> `disassemble_function`, `get_xrefs_to`, `read_memory`, `search_strings`, `search_instructions`),
> code Go à la tête du worktree `feat/campagne-grammaire` (base `69564ef7d`). Aucune commande `go`,
> aucune base, aucun film décodé. Image base `0x140000000`. Les numéros de ligne Go sont ceux de la
> tête du worktree ; les symboles font foi.
>
> La piste T3 (`T3_fin_de_vue_b.md`, même dossier) a lu indépendamment la même chaîne d'écriture
> (ordre `NEW*` `DELTA*` `DEL*`, état 3, rejet jamais légitime). Cette note la recoupe et ajoute ce
> qui est propre à T1 : l'ALLOCATEUR (où et sous quel identifiant une entité naît), la
> réutilisation d'un slot SANS `DEL`, l'endroit où vivent les `NEW` que la marche ne lit pas, et
> les écarts Go correspondants.

## 0. Réponse courte

| question | réponse | confiance |
|---|---|---|
| comment l'écrivain émet-il la naissance d'une entité entre deux images-clés ? | par un record `NEW` de la vue B (rang 1) d'un paquet de type 0, et par RIEN d'autre : `FUN_142f303bc` (seul écrivain d'en-tête de genre 1) n'est appelé que par `FUN_142f2cee0` (écriture d'une requête de vue B) et `FUN_142f2c658` (aller-retour Theater `FUN_1428e24bc`, hors film) | établi |
| où est-il dans le paquet ? | en TÊTE de la vue B : la vue B écrite est `NEW*` `DELTA*` `DEL*` `000`, chaque groupe par slot croissant ; le `NEW` est dans un paquet STRICTEMENT antérieur au premier `DELTA` de l'entité | établi |
| sous quel identifiant ? | slot choisi par un allocateur next-fit DÉTERMINISTE à cinq pools (le pool dépend de l'archétype) dont les cinq curseurs sont les cinq mots de queue du bloc de type 1 ; tag = (génération du slot + 1) & 3 | établi |
| pourquoi la marche ne le lit pas ? | quatre chemins nommés : tête de vue B sautée dans un paquet à événements (le localisateur part du delta du slot 123, et le filtre de candidats `NEW` est une bande d'image-clé qui exclut justement les slots nouvellement alloués) ; paquet à événements non localisé (paquet entier perdu) ; `NEW` désynchronisé (arrête tout le groupe `NEW`, slots croissants) ; `NEW` lu mais REFUSÉ parce qu'il réutilise un slot dont le `DEL` n'a jamais été écrit | chemins établis (jeu + code) ; leur poids est À MESURER (§7) |

Corollaire (recoupé avec T3) : dans un flux écrit par le jeu, une sortie de vue B par rejet hors
datum n'est JAMAIS légitime. Chaque sortie par rejet d'une lecture bien cadrée est un `NEW`
antérieur non lu ou non lié ; la carte de fermeture la range aujourd'hui sous « vue C :
terminateur hors cadre », c'est-à-dire sous l'effet et non sous la cause.

---

## 1. L'écrivain du film et la vue d'enregistrement `0x20`

### 1.1 La vue B du film est une vue d'entités d'index `0x20`, sans acquittement distant

`FUN_142e2eb2c` (démarrage de l'enregistrement, garde `FUN_1409a621c(&DAT_144c23178)` = session de
film active) :

```
FUN_1408f14fc(&DAT_145178bf0, 0x20, *(DAT_144e61d78 + 8))   // la vue d'enregistrement, index 0x20
FUN_1408f1730(decodeur, 0x20, &DAT_145178bf0)               // toute entité vivante -> état 1 (NEW à écrire)
```

`DAT_145178bf0` est construit par `FUN_140373bf0` -> `FUN_140b87eec`, qui pose
`*obj = 0x1436a87e0` (`140b87ef9: LEA RAX,[0x1436a87e0]`) : la MÊME classe que la vue B que le
lecteur `FUN_142987460` emprunte (vtable `0x1436a87e0`, `[0x40] = FUN_1406cd128`). La Kill-cam
fabrique une seconde vue de la même classe et du même index (`FUN_142f28768`, chaîne
`KillPlayback.cpp`, `FUN_1408f14fc(v, 0x20, ...)`) : l'index `0x20` est celui des vues
d'enregistrement locales.

### 1.2 La chaîne d'écriture d'un paquet delta

| maillon | adresse | rôle |
|---|---|---|
| enregistreur par tick | `FUN_142f2c3b0` | trois écrivains de bits ; section 0 = `FUN_142f2c050` (messages, vue A) ; section 1 = `FUN_142f2cc78` (vue B) ; section 2 = tampons de contrôle par joueur (vue C) ; remis à `FUN_1428e339c(&DAT_144c23178, ...)` |
| vue B | `FUN_142f2cc78` | `vtable[0x10]` = `FUN_142f2e174` (collecte des requêtes) ; pour chaque requête dans l'ordre de collecte, `vtable[0x18]` = `FUN_142f24a78` (écriture), budget `0x48000 - vtable[0x30]()` bits, arrêt sous `0x1000` ; `vtable[0x20]` = `FUN_14076b9c8` (concaténation) ; `vtable[0x58]` = `FUN_140862664` -> `FUN_14086268c(vue, seq, 1, 0)` (acquittement immédiat de chaque requête) ; `vtable[0x38]` = `FUN_14076b010` -> `FUN_14076c75c` (terminateur) |
| collecte | `FUN_142f2e174` | parcourt le bitmap « à mettre à jour » `vue+0x58` par slot CROISSANT ; état de vue 3 -> `0x1000000` (DEL) si le bit de suppression de la vue est levé dans `datum+8` ou si l'entité n'existe plus, sinon `0x1800000` (DELTA) ; autre état, entité existante -> `0x800000` (NEW) ; mot de requête = `rang<<30 | genre<<23 | priorité | slot & 0x1fff` |
| écriture d'une requête | `FUN_142f24a78` -> `FUN_142f2cee0` | sous-écrivain choisi par le GENRE : `142f24a87: LEA RBX,[RCX+0x1afb8]` puis `142f24aae: IMUL R9,RAX,0xd8` -> `vue + 0x1afb8 + genre*0xd8` (NEW `+0x1b090`, DEL `+0x1b168`, DELTA `+0x1b240`) ; position sauvée (`writer+0x48`), puis `FUN_142f303bc` / `FUN_142f304a8` / `FUN_142f30610` ; si le record dépasse le budget : `FUN_14076a148(w, 1)` = RETOUR ARRIÈRE, rien n'est engagé |
| engagement d'un NEW | `FUN_142f2f8f0` | `entrée de vue |= 1` ; `FUN_1408f1358(vue, eid, 3 - (vue+0x14 != 0x20))` : état **3** pour la vue `0x20` (2 = « en vol » pour une vue distante) ; `datum+8 |= 2` |
| concaténation | `FUN_14076b9c8` | `14076ba15: LEA RDX,[RBX+0x1b090]` / `14076ba24: LEA RDX,[RBX+0x1b240]` / `14076ba33: LEA RDX,[RBX+0x1b168]`, trois `CALL 0x1406d5d14` (copie de bits sans préfixe) : NEW, puis DELTA, puis DEL |

### 1.3 L'en-tête de record écrit (`FUN_142f2c754(w, genre, eid, archetype)`)

```
[si champs supplémentaires : W(32) = 0xf0c3a57e]
W(1) = (genre == 3)                       142f2c81d: CMP EDI,0x3 ; SETZ R8B ; CALL 0x1406d49c4
si genre != 3 : W(2) = genre              1 = NEW, 2 = DEL
FUN_1406d5110(_, w, 7, eid, -1) :         142f2c8d5: MOV R8D,0x7
    W(largeur(compte7)) = (eid & 0x3fffffff) - base7
    W(2) = eid >> 30                      le tag de génération
[si champs supplémentaires et genre != 3 : W(1) ; si 1 : W(8) = archétype]
```

C'est exactement ce que lit `FUN_1406cd128` (branche vive, `DAT_14474cd78 != 0`) et ce que porte
le Go (`readRecordType`, `readRecordID`, `frame_records.go:164-181`). Corps : `NEW` ->
`FUN_142e35a58` (lu par `FUN_1408f1aa4`, conforme au 5.19 §4.4) ; `DELTA` -> `FUN_140769e08`
(baseline) puis `FUN_142e35e60` ; `DEL` -> `W(32)` (valeur de `vtable+0xd8` si archétype `0x10`,
sinon 0), ce que lit `recDel: br.Skip(32)`.

### 1.4 La grammaire de la vue B écrite

```
vueB  := NEW*  DELTA*  DEL*  FIN
         chaque groupe par slot strictement croissant (collecte FUN_142f2e174, slot 0 -> N)
FIN   := 0 00
```

## 2. Quand l'écrivain écrit un `NEW`, et pourquoi un `DELTA` le suppose

### 2.1 Les états d'une entité dans une vue (`FUN_1408f1358`, `vue+0x38 + slot*0xa0 + 2`)

| état | sens | qui le pose |
|---:|---|---|
| 0 | inconnue de la vue | `FUN_142f2f320` (créée puis détruite avant toute écriture), `FUN_142f2e710` / `FUN_142f2e684` (suppression soldée), `FUN_1408f18d0` (lecteur, occupant écrasé) |
| 1 | `NEW` à écrire | `FUN_142f2e534`, appelée par l'allocation `FUN_142f2e598` (pour les 33 vues), par l'attache d'une vue `FUN_1408f1730`, par `FUN_142f301cc` / `FUN_142f258a0` (pertinence par vue) ; `FUN_14086268c` en cas de perte (vue distante) |
| 2 | `NEW` en vol (vue distante) | `FUN_142f2f8f0` quand `vue+0x14 != 0x20` |
| 3 | créée dans la vue : `DELTA` / `DEL` possibles | `FUN_142f2f8f0` (vue `0x20` : IMMÉDIATEMENT après l'engagement du `NEW`), `FUN_14086268c` (acquittement d'un `NEW`), `FUN_142f2f320` (état 1 et `NEW` déjà engagé une fois -> 3 AVEC le bit de suppression : la requête suivante est un `DEL`, pas un `DELTA`) ; côté lecteur `FUN_1408f1314`, `FUN_142f2f73c` |

La collecte ne produit un `DELTA` qu'à l'état 3 ; la collecte d'un paquet précède ses écritures
(`FUN_142f2cc78` : `vtable[0x10]` puis la boucle des `vtable[0x18]`). D'où l'invariant :

> **Dans la vue `0x20`, un `DELTA` de l'eid E dans le paquet P implique un `NEW` de E ENGAGÉ dans
> un paquet P' < P (strictement), sans réallocation du slot entre les deux, ou la présence de E
> dans l'état de départ du chunk (image-clé / bloc de type 1).**

Et une entité créée puis détruite entre deux collectes ne laisse AUCUNE trace dans le film
(`FUN_142f2f320` : état 1, jamais engagée -> état 0, compteur `vue+0x88`).

### 2.2 L'allocateur : où une entité naît (`FUN_142e31ef8` -> `FUN_142f2f634` -> `FUN_142f2f0cc`, puis `FUN_142f2e598`)

`FUN_142f2fc08(archetype, &base, &compte, &pool)` donne le pool de l'archétype ; la table est
statique en `0x143cefd78` (`read_memory`, paires `(base, compte)` en `int32`) :

| pool | base | compte | archétypes (lecture de l'aiguillage de `FUN_142f2fc08`) |
|---:|---:|---:|---|
| 0 | 0 | 512 | 0, 1, 2, 3, 6, 33, 34, 45, 46, 48, ... (moteur, joueurs, globaux) |
| 1 | 512 | 256 | **35** (bipèdes : la bande 521-601 du 5.19 §5.1) |
| 2 | 768 | 256 | **40** (véhicules) |
| 3 | 1024 | 256 | **41** |
| 4 | 1280 | `DAT_144706100 - 1280` | 12, 36, 37, 38, 39, 42, 43, 44, 47, 49, ... (objets) |

Si `DAT_144706104 == 0` (le bit de configuration du paquet), un seul pool `[0, DAT_144706100)`.

`FUN_142f2f0cc` est un next-fit par pool : curseur `*(table + 0x160 + pool*4)` ; recherche d'un
slot libre (`(drapeaux & 1) == 0` ou `(drapeaux & 2) != 0`) du curseur à la fin du pool, puis du
début du pool au curseur ; succès -> curseur = slot + 1 (remis à la base au-delà de la fin) ;
échec dans un pool borné -> `DAT_144706104 = 0` et nouvel essai (`FUN_142f2f634`) ; pool 4 plein
-> la table GRANDIT d'un slot et les globaux de largeur suivent (`DAT_144706100`, `DAT_1451f990c`,
`DAT_1451f98d4`, ... = slot + 1).

`FUN_142f2e598(table, slot, param)` pose le slot : `drapeaux |= 4`, `gen = (gen + 1) & 3`,
`compteur = compteur + 1` (repasse à 1 au-delà de 255), `eid = gen << 30 | slot`, puis
`FUN_142f2e534` pour chacune des 33 vues (état 1), sauf entité marquée `0x20` (`param != -1` et
`FUN_140be99fc()`), rendue pertinente vue par vue plus tard.

**Les cinq mots de queue du bloc de type 1 (5.21 §3.3, `type1_datums.go` `Queue`) sont ces cinq
curseurs** (`table + 0x160`, que 5.21 nommait « cinq compteurs du magasin ») : `bfecd02b`
chunk 1 `[123 0 777 0 1581]` (pool 0 curseur 123, pool 2 curseur 777 dans `[768,1024)`, pool 4
curseur 1581 = slot max de l'image-clé 1580 + 1) ; chunk 27 `[124 603 780 1048 2645]` (pool 1
curseur 603 dans la bande des bipèdes, pool 3 1048, pool 4 2645 = 2644 + 1).

Ce que l'allocateur explique, sur des mesures DÉJÀ publiées :

- « le premier slot rejeté d'un chunk est le slot max de son image-clé, plus un » (5.19 §5.5,
  13 chunks sur 26) : c'est le curseur du pool 4 ;
- la tête des en-têtes rejetés (5.25 §5) : tête 1 = 13 874 / 15 474 (**89,7 %**), tête 2 = 1 232,
  tête 3 = 237, tête 0 = 131. Un slot jamais alloué porte `gen = 0` dans le bloc de type 1
  (190 043 entrées, 5.21 §3.1) : sa première allocation donne le tag 1 ; un slot libéré une fois
  (`gen = 1`) donne 2 ; etc. Des en-têtes lus à une position FAUSSE auraient un tag uniforme
  (25 % chacun). La distribution mesurée est celle de l'allocateur, pas celle d'un décadrage.

### 2.3 Réutilisation d'un slot SANS `DEL` : le `NEW` vaut suppression implicite

`FUN_142f2f634`, quand le slot choisi porte `drapeaux & 1` et `drapeaux & 2` (alloué, détruit, `DEL`
pas encore soldé) :

```
drapeaux |= 0x10
pour chaque vue attachée (masque table[0x22], 33 vues) : FUN_142f2e710(vue, ancien eid)
    -> si le bit de suppression de la vue est levé : état 0, bit effacé, FUN_142f2df60(...)
drapeaux &= ~0x10 ; drapeaux |= 1
```

Le `DEL` en attente dans la vue `0x20` est ANNULÉ : le film reçoit le `NEW` du nouvel occupant
sans jamais recevoir le `DEL` de l'ancien. Le lecteur du jeu le sait : `FUN_1406cbaa0` (`NEW`) ->
`FUN_1408f1314` -> `FUN_1408f18d0`, qui, si l'entrée de vue du slot est occupée (état != 0, eid
!= -1), appelle `FUN_1408f1358(vue, ancien, 0)` et `FUN_1408f12c4(datums, ancien, 1)` — puis rend
TOUJOURS 1 (le code 2 du `NEW` est inatteignable) ; `FUN_1408f1618` alloue ensuite l'entrée de
datum (et fait grandir la table et les largeurs si le slot dépasse le cardinal).

## 3. Le lecteur du jeu correspondant (rappel, lu)

`FUN_1406cd128`, branche vive : `R(1)` (1 -> DELTA) sinon `R(2)` (0 = fin) ; id `base7 +
R(FUN_1406d310c(compte7))` (base 0 et `DAT_144706100` si `DAT_144706104 == 0`) ; tag `R(2)` ;
puis `FUN_1406cbaa0(genre, eid, ...)`. Pour un DELTA : garde
`*(uint *)(slot*200 + *(*(vue+0x20)+0x20)) != eid` -> code `3 - (*(DAT_144c1cfa8+4) != 2)`, zéro
bit lu, la boucle sort (la suite, côté `FUN_142987460`, est l'objet de T3). La garde compare
l'eid COMPLET (slot ET tag).

## 4. Pourquoi notre marche ne lit pas la naissance

Sous l'invariant §2.1, le `NEW` d'un eid rejeté hors datum est dans un paquet antérieur du même
chunk (5.21 : 23 306 des 23 325 slots rejetés sont VIDES dans le bloc de type 1 du chunk, donc nés
après son début). Il n'a pas été lu (ou pas été lié) par l'un de ces chemins :

**(a) Tête de vue B d'un paquet à événements.** Quand la vue A n'est pas vide, `consumeVueA`
(`frame_vue_messages.go:75-89`) ne lit que le premier genre `R(7)` et la marche part de
`debutDeLaListe` (`frame_closure.go:230-245`, `debut_de_liste.go:137-143`) : le localisateur
`marchLocateStrict` (`object_deaths_march.go:123`) rend la position du delta du slot 123. Par
la grammaire écrite, tout le groupe `NEW` et les `DELTA` des slots < 123 sont AVANT ce point. Le
lot M4b les récupère si une CHAÎNE part d'un candidat `NEW` et tombe exactement sur ce début ;
mais un candidat n'est retenu que si son slot est dans la BANDE de son archétype
(`candidatsDeTete`, `debut_de_liste.go:147-156`, `TableAnticipee.SlotDeLArchetype`,
`keyframe_anticipe.go:198-218`) : plage [min, max] des slots où les images-clés du film ont vu
l'archétype, moins tout slot vu porter un autre archétype (`slotBandExcluding`,
`slot_band_filled.go:63-69`). Or l'allocateur donne au nouveau-né du pool 4 le slot max + 1 (hors
de toute plage vue jusque-là, sauf si une image-clé ULTÉRIEURE y a vu le même archétype), et un
slot de pool 4 très réutilisé (projectiles, armes, équipement) a porté plusieurs archétypes au fil
du film, donc est retiré de toutes les bandes. Le filtre écarte précisément les naissances.

**(b) Paquet à événements non localisé.** `debutDeLaListe` rend -1 -> `listeNonLocalisee()` : le
paquet entier est sauté, groupe `NEW` compris (48 720 paquets sur 20 films, `MESURES_CLOTURE_J11`).

**(c) `NEW` désynchronisé.** `corpsDeRecordNeuf` (`frame_infer.go:178-206`) : une traversée qui
désynchronise arrête la vue. Les `NEW` étant en tête et par slot croissant, un seul `NEW`
d'archétype mal porté perd tous les `NEW` suivants du paquet (et toute la suite).

**(d) `NEW` lu mais refusé.** `contreditUneEntiteVivante` (`frame_infer.go:223-226`) refuse un
`NEW` traversé proprement dont le slot est lié en dur à un autre archétype ; son commentaire pose
« le jeu ne crée pas une entité sur une entrée occupée ... une entrée ne se libère que par la
suppression de son occupant ». L'écrivain (§2.3) fait l'inverse : il réutilise un slot détruit
dont le `DEL` n'est pas écrit. Un tel `NEW` est une vraie création ; refusé, le slot garde
l'ancien archétype et les deltas du nouvel occupant se décodent sous ce mauvais archétype
jusqu'à l'image-clé suivante (le verdict DFIX-R6, `keyframe_liaison.go:33-66`, les range en
« créations perdues » APRÈS coup).

**(e) Garde par slot, et non par eid.** `rejetDeVue` (`frame_infer.go:152-170`) teste
`w.ArchetypeForSlot(slot)` (`world.go:345`) : un slot lié à un occupant antérieur passe la garde
même quand le delta porte un AUTRE tag. Le jeu compare l'eid complet. Une naissance non lue sur
un slot RÉUTILISÉ ne produit donc pas un rejet mais un décodage sous l'archétype de l'ancien
occupant : sortie par désynchronisation, ou par un faux terminateur suivi d'une vue C hors cadre.
(L'A/B de `GenerationMatches` du 5.11.6 a coûté des records parce que les liaisons d'image-clé
sont `GenAny` — `world.go:205-207` ; l'écrivain, lui, écrit l'eid complet dans l'en-tête
d'image-clé : `FUN_142f30610` -> `FUN_142f2c754(w, 3, eid, ...)`.)

Ce que dit une mesure déjà publiée : 5.19 §5.5, chunk 2 de `bfecd02b`, **1 144 paquets
consécutifs FERMÉS avant le premier rejet, ZÉRO `NEW` lu, et le slot rejeté 1 673 pour une
image-clé à 1 663** — au moins dix naissances. Sous l'invariant, ces dix `NEW` sont dans les
1 144 paquets fermés ; un paquet SANS liste d'événements, lu depuis sa tête, ne peut pas fermer en
sautant un `NEW` ; ils étaient donc dans des paquets À événements fermés depuis le localisateur
(à l'époque, sans le lot M4b). C'est une déduction sur une mesure ancienne, pas une mesure à la
tête : d'où le §7.

## 5. Écarts Go / jeu, nommés

| # | le jeu (écrivain / lecteur) | le Go | conséquence |
|---|---|---|---|
| E1 | vue B = `NEW*` `DELTA*` `DEL*` `FIN`, slots croissants par groupe | ordre ignoré : ni oracle de cadrage, ni contrainte sur les candidats de tête | un décadrage n'est vu qu'au rejet ou à la vue C ; la chaîne de tête accepte des chaînes impossibles |
| E2 | un rejet hors datum n'existe pas dans un flux écrit par le jeu | `rejetDeVue` rend `hitEnd = true` et la vue C est lue (`frame_harvest.go:333-353`) ; la carte classe le paquet « vue C : terminateur hors cadre » (`frame_closure_classement.go:94-109`) | la cause n° 1 de la carte confond la cause (naissance non lue) et l'effet (vue C mal placée) |
| E3 | naissance = (slot, tag) prédits par l'allocateur (pool de l'archétype, curseur, drapeaux, génération du bloc de type 1) | candidats `NEW` filtrés par une bande d'image-clé (`SlotDeLArchetype`) ; bloc de type 1 non lu en production (`LireBlocDeDatums`, research seulement, `Queue` non interprétée) | les naissances de pool 4 et des slots multi-archétypes sont écartées par construction |
| E4 | `NEW` = suppression implicite de l'occupant ; réutilisation sans `DEL` | `contreditUneEntiteVivante` refuse | créations vraies perdues jusqu'à l'image-clé suivante |
| E5 | garde de delta sur l'eid complet | garde sur le slot | naissance non lue sur slot réutilisé = mauvais archétype au lieu d'un rejet |

## 6. Correctifs proposés (phase 2 ; aucun n'est écrit ici)

1. **Instrument d'abord (étape 1.2 du plan)** : pour toute sortie de vue B par rejet hors datum,
   classer le paquet « vue B : naissance non lue » (et non « vue C : terminateur hors cadre ») et
   chercher le `NEW` de l'eid (§7, M1). Aucune sortie de production ne change.
2. **Prédicteur d'allocation (production, phase 2)** : lire le bloc de type 1 de chaque chunk
   (`LireBlocDeDatums` existe) et en tirer, par pool, le curseur, les drapeaux et la génération ;
   avancer l'état à chaque `NEW` lu. Remplacer le filtre de bande de `candidatsDeTete` par :
   « (slot, tag) = une des k prochaines allocations prédites pour le pool de l'archétype annoncé
   par le `R(6)` du `NEW` » (k petit et nommé : des naissances du même pool peuvent tomber dans
   des paquets non lus). Ajouter les contraintes d'ordre de E1 à `chaineJusqua` (slots `NEW`
   strictement croissants, puis slots `DELTA` strictement croissants, aucun `NEW` après un
   `DELTA`). Étendre `debutParFermeture` (paquets non localisés) au même filtre.
3. **Suppression implicite** : quand un `NEW` traversé proprement a le (slot, tag) prédit par
   l'allocateur, le lier (et délier l'ancien occupant) au lieu de le refuser ; garder le refus
   pour un `NEW` dont le (slot, tag) n'est pas prédit (c'est alors bien une lecture suspecte).
4. **Oracle d'ordre** : dans la boucle de records de la vue B, un record qui viole l'ordre écrit
   (NEW après DELTA, DEL suivi d'autre chose que DEL ou FIN, slot non croissant dans un groupe)
   est une désynchronisation nommée, pas un record.
5. **Garde par eid** (dépend de 2 et de liaisons d'image-clé portant le tag) : à instruire après
   la mesure M5 ; ne pas la poser tant que les liaisons d'image-clé sont `GenAny`.

Tous ces correctifs changent une sortie (records liés, vues C lues) : preuve et fusion contre la
tête post-J12, `grammar.Rev` à monter, gate de corpus.

### Vecteurs de test construits d'après l'écrivain

- **V1 (ordre)** — vue B synthétique écrite avec l'en-tête §1.3 (`HasExtraFields` faux, id 13 bits) :
  `NEW(1<<30|1581, ti=T)` `NEW(1<<30|1582, ti=T)` `DELTA(3)` `DELTA(123)` `DELTA(540)` `DEL(700)`
  `000` -> accepté ; les permutations `DELTA(123) NEW(...)`, `NEW(1582) NEW(1581)`,
  `DEL(700) DELTA(800)` -> désynchronisation nommée « ordre de vue B ».
- **V2 (naissance en tête d'un paquet à événements)** — chunk dont le bloc de type 1 porte
  `Queue = [123 0 777 0 1581]`, slot 1581 drapeaux 0 gen 0, et une image-clé qui ne déclare pas
  1581. Paquet P1 : `cfg` · vue A `1 g(7) <corps opaque> 0` · vue B `NEW(1<<30|1581, ti=T)`
  `DELTA(123, signature)` `000` · vue C fermante. Paquet P2 : `cfg` · `0` · vue B
  `DELTA(1<<30|1581)` `000` · vue C fermante. Attendu jeu : P2 ferme. Tête actuelle : si 1581
  n'est pas dans la bande de T, P1 part du slot 123, P2 rejette 1581 et sa vue C est hors cadre.
  Après le correctif 2 : P1 lit le `NEW` (prédiction pool 4 = 1581, tag 1), P2 ferme.
- **V3 (réutilisation sans DEL)** — slot 1590 lié en dur (NEW propre) à `ti=A`, détruit sans `DEL`
  écrit ; type 1 : drapeaux `0x7`, gen 1 ; curseur de pool = 1590. Paquet : `NEW(2<<30|1590, ti=B)`
  puis, paquet suivant, `DELTA(2<<30|1590)` au corps de B. Attendu jeu : lié à B, delta décodé
  sous B. Tête actuelle : `NEW` refusé, delta décodé sous A.
- **V4 (garde par eid)** — slot 1600 lié à `1<<30|1600` ; delta portant `2<<30|1600` sans `NEW`
  lu : attendu jeu = rejet (code 2/3) ; tête actuelle = décodage sous l'archétype de l'ancien.

## 7. Mesures proposées (ce qui confirmerait ou réfuterait, sur la carte v2)

- **M1 — où est le `NEW` d'un eid rejeté.** Pour chaque sortie de vue B par rejet hors datum
  (eid E, chunk k, paquet P), balayer bit à bit les paquets de type 0 du chunk k avant P pour un
  en-tête `0 01 idx(13) tag(2)` égal à E, suivi d'un `R(6)` d'archétype d'objet et d'une
  traversée `NEW` propre. Classer la position : (i) paquet à événements, avant le début rendu par
  `debutDeLaListe` ; (ii) paquet à événements non localisé ; (iii) paquet lu dont la vue B s'est
  arrêtée avant ; (iv) lu mais refusé (`contreditUneEntiteVivante`) ; (v) introuvable. Témoin :
  même balayage pour des eid (slot jamais vivant du chunk, tag quelconque) -> taux de faux
  positifs. Prédiction de l'écrivain : (v) au niveau du témoin, (i)+(ii)+(iii)+(iv) = le reste.
  Une part (v) nettement au-dessus du témoin réfute l'invariant §2.1 tel que je l'ai lu.
- **M2 — le prédicteur d'allocation.** Pour chaque `NEW` lu proprement, comparer (slot, tag) à la
  prédiction tirée du bloc de type 1 du chunk et des `NEW` déjà lus : taux d'égalité exacte pour
  le premier `NEW` de chaque pool dans chaque chunk (attendu proche de 100 % sur HI_1_13_0 ; les
  vieux builds peuvent avoir une autre table de pools).
- **M3 — refus.** `NeufsContreUnVivant` et ses verdicts DFIX-R6 par film, croisés avec « (slot,
  tag) = prédiction » : un refus prédit par l'allocateur est une création perdue.
- **M4 — gain sur la fermeture.** A/B de recherche « oracle de naissance » : lier E à la position
  trouvée par M1 (archétype = son `R(6)`), rejouer la carte, compter les paquets qui passent de
  « hors cadre » à fermé, par build et par chemin (i)-(iv). C'est la borne du correctif 2+3.
- **M5 — garde par eid.** Compter les deltas dont le slot est lié et le tag diffère du tag lié
  (liaisons non `GenAny`), et leur sortie de vue B.

## 8. Questions ouvertes

- Le poids de chaque chemin (a)-(e) dans les 264 757 paquets « hors cadre » : seule M1/M4 le dira.
- L'aiguillage complet de `FUN_142f2fc08` (archétype -> pool) est lu en gros ; une table exhaustive
  par archétype demande un relevé instruction par instruction (les vieux builds peuvent différer).
- `FUN_142f2df60` (solde d'une suppression, libération du datum) n'est pas lu : le moment exact où
  un slot détruit redevient allouable (et donc la fenêtre de réutilisation sans `DEL`) en dépend.
- Le champ `vue+0x1b320` (file de records différés, `FUN_142f2b5c4` / `FUN_142f29538` /
  `FUN_142f2913c`, `NEW` lu avec `param_7 = 1` sans allocation de datum) : aucune écriture trouvée
  par `search_instructions` ; s'il est actif à la lecture d'un film, un `NEW` différé suivi d'un
  `DELTA` rejetterait CHEZ LE JEU. À trancher par T3 ou une lecture dynamique.
- Quel `R(6)` / quel pool pour les 801 eid de `bfecd02b` : la tête 1 dominante dit « première
  allocation », pas « quel pool ».

## 9. Découvertes hors périmètre (consignées, non traitées)

- La largeur de l'identifiant de record n'est pas une constante du film : `FUN_1408f1618`
  (lecteur) et `FUN_142f2f0cc` (écrivain) posent `DAT_144706100` / `DAT_1451f990c` = slot + 1
  quand la table grandit au-delà de son cardinal ; le Go fige `IDLowBits = 13`
  (`frame_records.go:144`). Sans effet tant que la table reste sous 8 192 slots.
- Le bit de configuration du paquet (`DAT_144706104`) bascule à 0 quand un pool borné est plein
  (`FUN_142f2f634`) : la base et la largeur des identifiants de domaine changent alors pour tous
  les records suivants.
- `FUN_142f2fc08` range les bipèdes (35) dans `[512, 768)`, les véhicules (40) dans `[768, 1024)`,
  `ti=41` dans `[1024, 1280)` : la « bande des bipèdes » du dépôt est une constante de l'écrivain,
  lisible statiquement, pas une mesure.
