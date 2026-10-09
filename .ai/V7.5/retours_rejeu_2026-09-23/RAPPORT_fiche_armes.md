# Lot `fiche_armes` — « à 2:08, XxDaemonGamerxX n'a pas d'armes dans sa fiche » (81c02726)

Enquête en lecture seule, base `43a01721e`. Horloge : `t = affiché_s × 10 + 227`.
Légende : **MESURÉ** (fichier:ligne, sortie chiffrée), **DÉDUIT**, **HYPOTHÈSE**.

Pièces produites (scratchpad `SP/sondes/fiche_armes/`) : `sweep.mjs`, `sweep2.mjs` (gate parc),
`holes.mjs`, `dist.mjs`, `instant.mjs`, `absorb.mjs`, `modeload.mjs`, `spawncov.mjs`, `d1..d7.mjs`,
sortie de la sonde Go `go_out1.txt`. Sonde Go (faits persistés de 81c02726, a0c36016, b1f01a33,
aucun film ouvert) : `LevelUp-wt-inv-fiche_armes/apps/go-api/internal/games/halo_infinite/film/replay/fiche_armes_research_test.go`.

---

## 1. Constat reproduit sur pièces

### 1.1 La vie concernée (MESURÉ, document 81c02726)

Slot 534, xuid 2533274833178266, filmIndex 1, équipe 0, vie `startFrame 1422 → endFrame 1623`
(1:59.5 → 2:19.6). Création de bipède lue à t 1422 (gen 1, index de participant 1). Pour cette vie :
**0 `loadouts`, 0 `inventory`, 0 `abilities`, 0 `equipmentChanges`**. Ce qui existe :
`weaponChanges {t 1520, taken, fd98554c VK78, sans from}`, `pickups` VK78 à 1520 et grenade à pointes
à 1576 (origine spawner), `grenadeReads` delta à 1576/1577, trois tirs VK78 (1583, 1584, 1606).

### 1.2 Ce que la fiche affiche entre 1:59.5 et 2:19.6 (MESURÉ dans le code web)

| instant affiché (t) | armes | grenades | munitions / capacité |
|---|---|---|---|
| 1:59.5 → 2:19.6 (1422 → 1623) | **deux cellules vides en pointillés**, infobulle « armes non lues sur cette vie » | lecture **à venir** de 1576 : frag ×2 + pointes ×1, « dans X s » (jusqu'à 2:14.9) | rien |
| 2:09.3 (1520) : prise VK78 écrite dans le film | **inchangé** | — | — |
| 2:14.9 (1576) : ramassage de pointes | inchangé | lecture delta fraîche [2,0,0,1] puis [2,0,0,2] | — |
| 2:15.6 / 2:17.9 : tirs VK78 | inchangé | — | — |
| 2:19.6 (1623) : mort | encadré « Éliminé » | | |

Chaîne : `ReplayPlayerCard.tsx:138` → `playerCardReadings.ts:101` → `equippedLogic.ts:65-66`
(`loadoutAt` nul → `null`) → `rosterLogic.ts:574-578` (`nearestReading` borné à la vie
`[1422, 1623]`, `rosterLogic.ts:645-668` ; aucune lecture ni passée ni « à venir ») →
`ReplayWeaponsRow.tsx:119-131` (cellules vides + `loadoutUnread`, `i18n.ts:324`).

Pourquoi la prise de 2:09.3 n'est pas exploitée : `refineWeaponsReading` n'est appelé que s'il
existe une lecture de base (`rosterLogic.ts:578-588`), et il n'applique de toute façon que les
substitutions avec `from` connu (`changeRefine.ts:87-91`) — or la première émission d'un
emplacement n'a jamais de `from` (l'état de naissance n'est pas vu dans le delta, §2.3).
L'héritage de la vie précédente est interdit à juste titre (correctif P0-2 du 2026-09-06 ; chaque
vie a son propre slot ici, 534 n'est utilisé qu'une fois).

### 1.3 Pourquoi aucune lecture : l'image-clé de 2:06.7 (t 1494) est trouée (MESURÉ)

Images-clés du film (sonde Go, recensements des faits) :

| frame | loadouts | inventaires | ti42 | ti37 | ti40 | slots vus [min..max] |
|---:|---:|---:|---:|---:|---:|---|
| 1294 | 8 | 9 | 27 | 23 | 2 | [517..1913] |
| **1494** | **1** | **1** | 30 | 23 | 2 | **[536..1963]** |
| 1694 | 4 | 8 | 30 | 21 | 2 | [519..2009] |

À 1494, **rien sous le slot 536 n'est reconnu**, tous archétypes confondus ; les bipèdes 519, 528,
530, 533 (vus à 1294 ET à 1694, même vie) et 534, 535 (nés à 1422 et 1435) sont absents ; les
objets ti=42/37/40 (slots ≥ 768) sont présents en nombre normal. **C'est une perte de PRÉFIXE de la
table d'image-clé, pas un film vide.** C'était l'unique image-clé de la vie 534.

Correction au superviseur : l'image-clé de t 1494 est celle du **chunk 9** (début 160 012 ms :
`(160 012 − 10 559)/100 = 1494,5`), pas du chunk 8 (140 007 ms → t 1294). `chunk_09.bin` fait
883 783 o, taille normale.

Effet de bord à 2:08 : les fiches de 519/528/530/533 affichent une lecture vieille de 21,3 s (1294)
au lieu de 1,3 s.

### 1.4 Le parc (111 documents, grille d'images-clés complétée) — MESURÉ

| grandeur | valeur |
|---|---|
| images-clés de la grille avec des vivants | 2 868 |
| trouées (≥ 1 bipède vivant absent) | **133** (4,6 %) |
| sans AUCUN bipède alors que des joueurs vivent | **80** (2,8 %) |
| total touchées | **213 / 2 868 = 7,4 %**, dans **69 / 111** documents |
| instants-vie absents | 1 079 / 19 159 = 5,63 % |
| forme de la perte (133 trouées) | **préfixe 106**, suffixe 26, milieu 1, entrelacée 0 |
| vies sans AUCUNE arme publiée | **2 162 / 11 443 = 18,9 %**, soit **25 663 s = 6,6 % du temps de vie** |
| — dont une image-clé tombait PENDANT la vie (trou) | 290 vies, 7 787 s (durée médiane 21,1 s, p90 51,8 s) |
| — dont vie finie avant toute image-clé (structurel) | 1 872 vies, 17 875 s (médiane 9,7 s, max 19,8 s) |
| vies dont le début affiche une lecture « à venir » | 9 237 vies, 89 035 s = 23,1 % du temps de vie |
| attente naissance → 1re lecture (vies lues) | médiane 9,0 s, p90 17,5 s, max 135,3 s |

## 2. Cause racine

### 2.1 Cause du cas rapporté : le balayeur d'image-clé saute des records (MESURÉ + DÉDUIT)

`loadouts` et `inventory` sortent tous deux de `grammar.WalkKeyframeWorld`
(`keyframe_loadout.go:124`, `inventory_decode.go:288-303`), le balayeur HEURISTIQUE
(`keyframe_world.go:193-269`) : il n'enchaîne pas les records par leur grammaire, il cherche
l'ancre suivante (`kfScanNext`, `keyframe_world.go:155-188`) et, faute d'un voisin « slot+1, gen 1 »,
**élit sur une fenêtre de 120 000 bits** (`:213`) le candidat de plus basse génération puis de plus
bas slot. Tout ce qui est entre l'ancre quittée et l'ancre élue est perdu, et comme la table est à
slots croissants, rien ne se rattrape. Son filtre fort (`archétype < 50`, `:114`) le rend en outre
aveugle aux entrées « sans archétype » (`0xFFFFFFFF`, 108 bits, `keyframe_record_walk.go:40-44`).

Preuves que les records SONT dans le film et que c'est la marche qui les saute :
- **MESURÉ** (document) — quand le saut laisse un record trouvé devant la zone sautée, les familles
  d'arme de la zone lui sont attribuées (`familiesByRecord` attribue au record qui « contient » le
  bit, `keyframe_loadout.go:123-158`). 81c02726 à 4:46.8 (t 3095) : le slot 541 (MORT depuis 2942)
  porte **cinq armes** — les siennes (Needler + Sidekick), l'AR + Bulldog de 551 et le BR75 que 554
  a pris à 3042 — pendant que 551 et 554, vivants, sont absents. Sur le parc, **6 images-clés**
  montrent ce motif avec les armes des absents lues aux images-clés voisines (`absorb.mjs`, ex.
  0a44c6cc t 2665 : slot 545 à 9 armes, absents 548/550/551/552).
- **DÉDUIT** pour la forme préfixe (1494) : l'écrivain `FUN_142e2bfd0` écrit UNE entrée par entité
  vivante (NOTE_5_20 §1.2), et les quatre bipèdes absents sont vus, même vie, aux images-clés
  d'avant et d'après ; aucun record n'étant trouvé devant la zone perdue, le témoin d'absorption
  ne peut pas s'y lire.

Sous-mécanisme exact de la perte de préfixe (départ refusé au bit 1 puis élection lointaine, ou
fausse ancre élue, ou entrée sans archétype) : **NON TRANCHÉ** sans le payload. Sonde S1 (§4).
La NOTE_5_20 §1.5 avait déjà mesuré que « `betterThan` élit un candidat lointain qui déraille la
chaîne » (dad793c7) et laissé le balayeur en place jusqu'à fermeture complète des images-clés
(30,8 %, bipède 6 records).

### 2.2 Cause structurelle : les armes de naissance ne sont lues nulle part (MESURÉ)

Même sans trou, 1 872 vies (16,4 %) meurent avant toute image-clé, et 9 237 autres affichent au
début une lecture future (qui peut montrer une arme ramassée plus tard : JGtm slot 529, lecture de
1094 = Hydra affichée dès 1:09.9 alors qu'il la prend à 1:16.0).

### 2.3 Le film porte-t-il les armes à l'instant de l'apparition ?

Canaux existants, mesurés sur les naissances des trois films de faits (sonde Go) :

| canal | 81c02726 (Arena) | a0c36016 (CTF) | b1f01a33 (Super Fiesta) |
|---|---|---|---|
| naissances (records de création de bipède) | 45 | 142 | 106 |
| **i48 équipement émis ≤ 1 s après la naissance** (étalon) | 0 | 1 | **94** |
| **i43..i46 identité d'arme émise ≤ 1 s** | **0** | **0** | **0** |
| création ti=42 dans [−1 s, +2 s] | 32 (hasard attendu ~81 %) | 115 (~92 %) | 51 (~81 %) |

- **Négatif ÉTALONNÉ sur le canal delta** : le marcheur des records de mise à jour
  (`delta_biped_walk.go`, ancré par `matchBipedHeader`, `offline_biped.go:268-314`) lit i48 sur
  94 naissances sur 106 de b1f01a33 (même marcheur, même record, composant d'index supérieur,
  `ability_rank.go:169-180`, 0 illisible) et n'y voit JAMAIS i43..i46 (0/106). Les armes de
  réapparition ne sont donc pas transmises comme mise à jour delta à la naissance. Super Fiesta tire
  les armes au sort : elles ne peuvent pas venir d'une table de mode, le film DOIT les porter.
- **Pas de création ti=42 au-dessus du hasard** : les armes tenues ne sont pas des objets du monde
  (le lâcher en crée un, `held_weapon_changes.go:14-18`).
- **DÉDUIT, fort** : le seul porteur répliqué de l'arme tenue est `weapon-state-type-info` du
  bipède ; absent des mises à jour, il est dans le **record NEW du bipède**, que le marcheur delta
  n'ancre jamais (il n'accepte que des records à masque épars ouvrant sur i0). Or la MARCHE du dépôt
  (`frame_infer.go:178-200` → `TraverseEntity`, `traverse.go:94-137`, état par défaut du bipède
  « VALIDÉ BIT-EXACT », 4 désync sur 129 572 records ti=35 de bfecd02b) traverse ces records NEW et
  lit l'arme via `consumeWeaponStateTypeInfoVariant` (`components_object.go:300-320`, qui publie
  famille + variante à l'observateur). **Personne n'a jamais branché cette lecture.** Sonde S2 (§4).
  (Le commentaire de `biped_creation.go` « porté à ~120 bits sur ~380 » est périmé face à
  `traverse.go:103-110`.)

**Donc : pas « image-clé seulement ».** Le film porte très probablement les armes de naissance
dans le record NEW du bipède ; aucun canal du dépôt ne les lit.

## 3. Historique

- 2026-08-24 `MESURE_TROUS_INVENTAIRE` : « 67 bipèdes absents sur 698 images-clés » et « 58
  images-clés sans aucun bipède (8,3 %) — à instruire séparément » ; 2026-08-25 `LOT3_GRENADES` :
  « non instruits ». Jamais repris. Le lot visait les lectures VIDES, pas les records manquants.
- 2026-08-12 (décision utilisateur) : lecture « à venir » en début de vie — ne couvre que les vies
  qui ont une image-clé ; inopérante quand l'unique image-clé est trouée.
- Schémas 24-25 (`weaponChanges`, `changeRefine.ts`) : la datation fine suppose une base
  d'image-clé et ignore les prises sans `from` ; sans base, rien.
- Série 5 (5.16 D2, 5.20) : la marche d'image-clé heuristique connue pour dérailler, gardée
  « jusqu'à fermeture à 100 % » ; lien avec la fiche jamais fait.

## 4. Solution proposée

### Deux sondes d'abord (lecture de film : décision superviseur/utilisateur ; ~minutes, RAM faible, jamais un BTB)

- **S1 — la perte de préfixe.** Test `research` (paquet `grammar`) sur le SEUL `chunk_09.bin` de
  81c02726 (et chunks 1-2 de a0c36016, pertes jusqu'au slot 1438) : tracer `WalkKeyframeWorld`
  (départ, élections, arrêt), chercher dans le payload les en-têtes EXACTS de 64 bits
  `[(1<<30)|slot][35]` des bipèdes vivants (519…536), compter les entrées sans archétype
  traversées (`readKeyframeHeader`). Tranche : présence prouvée + sous-mécanisme.
- **S2 — les armes de naissance.** Test `research` : la marche (`DecodeFrameRecords`) avec un
  `HeldWeaponHook`, relever pour chaque record NEW ti=35 (slot, gen, t) les familles lues. Oracles :
  1re lecture d'image-clé de la vie sans prise intermédiaire, arme des tirs ; témoin : lecture d'une
  autre vie. Films : 81c02726, b1f01a33 (départs aléatoires, le test décisif), a0c36016. Seuil pour
  produire : ≥ 95 % des naissances lues, accord ≥ 98 %, témoin au niveau du hasard.

### Option 1 (RECOMMANDÉE) — lire à l'instant : armes de naissance + marche d'image-clé réparée, un seul lot décodeur

1. Grammaire (si S1 confirme) : `keyframe_world.go` — enchaîner les entrées sans archétype à
   108 bits comme l'écrivain ; se resynchroniser sur l'en-tête EXACT d'un eid CONNU (vu à l'image-clé
   précédente, ou né par un NEW depuis) au lieu d'élire par (gen, slot) ; l'élection ne reste qu'en
   repli NOMMÉ et compté (`coverage.fallbacks`). Publier le compteur des bipèdes absents encadrés
   (`coverage.inventory`), aujourd'hui invisible.
2. Grammaire (si S2 confirme) : nouveau canal « armes de naissance » tiré de la marche → `FilmInputs`
   → faits (`SchemaDesFaits` +1) → document : entrée `loadouts` à la frame de naissance avec
   provenance optionnelle, et `weaponChanges[].k` (emplacement 0/1, lu dans `SlotIndex`) pour
   appliquer une prise comme substitution. Reclasser les premières émissions contre la dotation de
   naissance (supprime le repli futur de `spawnSetFrom`, §6-1).
3. Web : `loadoutAt` prend la dotation de naissance comme base ; `refineWeaponsReading` applique les
   prises avec `k` ; infobulle FR+EN de provenance ; plus de lecture « à venir » là où la naissance
   est lue.

- Fichiers : `grammar/keyframe_world.go`, nouveau `grammar/birth_loadouts.go` (+ marche),
  `replay/film_scan.go`, `film_inputs.go`, `filmfacts_*` (section), `loadouts.go`,
  `document_weapon_changes.go`, `coverage.go`, `document_chronicle.go` ; web `rosterLogic.ts`,
  `changeRefine.ts`, `ReplayWeaponsRow.tsx`, `i18n.ts` ; OpenAPI + types générés.
- **Montée de schéma : oui (68 → 69). Republication : REDÉCODAGE de tout le parc** (`GrammarRev`
  monte, les faits sont périmés) puis séquence prod (recuisson → usage-summary → pad-tiers --force →
  killsource), serveur arrêté — **décision utilisateur**. La marche d'image-clé alimente aussi le
  monde de la marche et `killsource` : révision des faits et re-figeage de l'équivalence probables.
- Tests ROUGE→VERT : payload synthétique (entrée sans archétype + record gen 2 + record gen 1
  lointain : saute aujourd'hui) ; première émission avant toute image-clé classée `restated` à tort
  (rouge aujourd'hui) ; NEW ti=35 synthétique portant i43/i44 ; web `loadoutAt` avec base de
  naissance. Garde-rails : ratchet « bipèdes absents encadrés » sur les mini-films par build
  (plancher 0), test interdisant une dotation de spawn prise dans le futur.
- **Gate sur documents réels** (scripts `sweep2.mjs`, `absorb.mjs`, `dist.mjs`) : images-clés
  touchées 213/2 868 → 0 ; instants-vie absents 1 079 → 0 ; absorptions 6 → 0 ; vies sans arme
  2 162 (18,9 %) → ≤ 1 % hors BTB ; fiche sans arme 25 663 s (6,6 %) → < 0,5 % ; lecture « à venir »
  89 035 s (23,1 %) → ≈ 0 ; 81c02726 : slot 534 lu dès 1:59.5, sept bipèdes présents à 1494.
- Risques : films denses — la marche ne ferme que 13 % des trames sur bfecd02b (NOTE_5_26), donc
  naissances manquées en BTB (couverture à publier par film) ; rayon d'action large (recensements
  d'objets, datation des socles, monde de la marche) ; la série « résidu film dense » est close par
  l'utilisateur — ce lot la touche par la marche d'image-clé.
- Taille : **L** (sondes S, grammaire M, naissance M, web S, recuisson L).

### Option 2 — sans redécodage : la fiche dit ce que le document porte déjà à l'instant

Sans lecture d'image-clé dans la vie, composer une rangée PARTIELLE depuis `weaponChanges`
(taken/swapped), `pickups` (kind weapon) et `shots[].w` (arme en main au tir), avec provenance
(« Prise à 2:09,3 », « Tirée à 2:15,6 ») et le reste en lacune ; côté Go, reclasser les premières
émissions À L'ASSEMBLAGE (depuis les faits) pour ne plus jeter de vraies prises en `restated`.
Couvre **10 899 s sur 25 663 (42,5 %)**, 1 659 vies sur 2 162, première preuve à 5,7 s (médiane).
Pas de montée de schéma pour le web ; republication DEPUIS LES FAITS (pas de décodage) pour le
reclassement. Ne répare ni le trou ni 2:08 (aucune preuve avant 2:09.3). Taille **S-M**.

**Recommandation : Option 1, S1 et S2 d'abord** (elles décident tout pour quelques minutes de
machine). L'option 2 est un palliatif compatible, pas une réponse au cas rapporté.

## 5. Questions pour l'utilisateur

1. Arme inconnue sur une fiche vivante : (a) cellules vides « armes non lues » (aujourd'hui),
   (b) dotation de départ du mode « présumée » avec marqueur — juste dans **98,4 %** des vies en mode
   à départ fixe, **4 %** en Super Fiesta, donc jamais en départ aléatoire —, ou (c) seulement ce que
   le film a dit à l'instant (dernière prise, dernier tir) ? Réponds a, b ou c.
2. J'autorise S1 (un chunk de 81c02726, deux de a0c36016) et S2 (marche sur 3 films arène/Fiesta) :
   oui / non ?
3. Réparer la marche d'image-clé heuristique (lot décodeur, redécodage de tout le parc) malgré la
   clôture de la série « film dense » : oui / non / après S1 ?
4. Une fois les armes de naissance lues, supprime-t-on la lecture « à venir » des armes : oui / non ?

## 6. Hors périmètre découvert (noté, non traité)

1. **`spawnSetFrom` prend une image-clé FUTURE** quand la vie n'en a pas encore
   (`document_weapon_changes.go:173-180`, `pick := list[0]`, appelé au balayage `film_scan.go:135`) :
   de vraies prises sont classées `restated` et retirées du document. 81c02726 : Hydra de JGtm à
   1:16.0 (slot 529, t 987) ; b1f01a33 : 3 `restated` sur 5 tombent sur un ramassage natif de la même
   arme à la même frame.
2. **Armes absorbées** : un record trouvé devant une zone sautée hérite des armes (et sans doute des
   munitions et grenades, même bornage `invRecordSpans`) des bipèdes sautés — 6 images-clés au parc,
   plus 81c02726 t 3095.
3. `coverage.inventory` ne dit rien des bipèdes absents : la perte est invisible dans le document.
4. Images-clés antérieures à l'origine (chunk 1) : 0 record dans tous les recensements sur les trois
   films de faits — à vérifier (légitime ou même défaut).
5. Commentaire périmé `biped_creation.go` (~120/380 bits) contredit `traverse.go:103-110`.
