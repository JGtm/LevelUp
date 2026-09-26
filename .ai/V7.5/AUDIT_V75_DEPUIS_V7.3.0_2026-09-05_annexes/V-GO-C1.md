# Vérification adverse V-GO-C1

Dépôt `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, branche `feat/v75`.
HEAD au moment de la vérification : `081871f09` (le code des constats, `736ccf3c3`, y est intact —
aucun des fichiers cités n'a bougé depuis). Lecture seule, aucun `go test` / `go build` / lint exécuté.

## Constat 1 — `entity.go` + `entity_quant.go`, deux décodeurs sans appelant : TIENT

- **Ce que j'ai vérifié**
  - `Grep` ripgrep sur TOUT le dépôt, motif `DecodeEntityRecord|\bEntityRecord\b` : 10 lignes Go,
    **toutes** des déclarations ou des commentaires (`entity.go:27,29,65,66,67`,
    `entity_quant.go:5,20,21,22`, `components_batch7.go:7`). Les autres occurrences sont dans
    `.ai/` (thought_log, plans RE). **Zéro site d'appel**, tests compris.
  - Recherche d'appelants indirects : aucune variable de fonction, aucun `map[string]func`, aucun
    `cmd/`, aucune réflexion ne les atteint (le grep couvre `apps/go-api/cmd/` et les `_test.go`).
  - Clôture morte, un symbole à la fois, appelants HORS `entity*.go` :
    `decodeFullStatePosition`, `decodePositionBody`, `positionTag`, `readDequant`, `decodeC9E4D8`,
    `decodeC9E990`, `readOptional10`, `decodeInnerHeader`, `buildGate0x24`, `unpackDir30`,
    `decodeOptComponent`, `decodePair6`, `decodeC9E738` → **`[]` pour chacun**. Seul
    `decodeQuatBlock` sort dans un autre fichier, et c'est un **commentaire**
    (`unit_weaponstate.go:374` « identical to entity.go's decodeQuatBlock »).
  - `grep -n fullState entity.go` → **2 lignes seulement** : `:65` (commentaire) et `:66`
    (signature). Le paramètre n'est jamais lu. `wc -l` = 338 + 248 = **586 L**.
  - Allowlists et décisions : `internal/archlint/` (35 fichiers) ne mentionne `entity` que pour
    `keyframe_entity_queue.go` ; `.ai/V7.5/REGISTRE_REPORTS.md` ne porte **aucune** ligne
    entity.go/`DecodeEntityRecord` ; aucun `TODO(expiry:)` (`archlint/todo_expiry_test.go`).
- **Ce qui confirme, au-delà du rapport**
  - Le dépôt documente lui-même la **supersession** : `components_batch7.go:6-8` — « the REAL
    frame-deser of object-multiplayer-properties-component (**CORRECTED 2026-06-14: the old
    `DecodeEntityRecordQ` targeted FUN_14080c1f8, a DIFFERENT/larger entity record**) ». Ce n'est
    donc pas seulement du code sans appelant : c'est du code dont la cible RE a été déclarée
    fausse pour son unique usage envisagé, et remplacée.
  - La doctrine maison tranche déjà ce cas exact : `REGISTRE_REPORTS.md` l.252 — « le décodeur est
    COMPLET et MESURÉ, et il n'a AUCUN appelant […] `filmdec.ParseStatborgRecord` +
    `TestParseStatborgRecord_V8` sont SUPPRIMÉS par ce lot (règle "0 code mort") ».
  - `git log --diff-filter=A` : créés le **2026-07-31** (`3f0ec70b3`), dernier passage
    **2026-08-01** `1cb67b790` « J3 lot B — unparam » — la campagne de dette a bien corrigé une
    signature à l'intérieur de la clôture morte, comme le dit le rapport.
- **Nuance à la marge** : le rapport dit garder « `bitLen` et `readQuantStat` (seuls helpers à
  appelants externes vivants) ». Il en oublie un troisième, `quantStatDefaultWidth`
  (`entity_quant.go:18`), lu par `components_object_state.go:147`, `components_batch7.go:125` et
  `traverse.go:395`. Cela ne change pas l'ordre de grandeur (~20 L vivantes sur 586).
- **Conséquence réelle** : ~565 des 586 lignes de ces deux fichiers sont inatteignables, y compris
  deux décodeurs dont le dépôt écrit ailleurs qu'ils visaient la mauvaise fonction du jeu.

## Constat 2 — 22 `Set*` morts, deux drapeaux legacy sans date : TIENT (deux chiffres à corriger)

- **Ce que j'ai vérifié**
  - Recensement indépendant (boucle sur `^func Set[A-Z]` de `filmdec/*.go` non-test, comptage des
    sites d'appel hors déclaration, sur tout `apps/go-api` tests compris) : **66 setters déclarés**,
    dont **22 à zéro appelant** — et c'est **exactement la même liste de 22 noms** que le rapport.
    (Ma ventilation du reste diffère un peu : 20 test-only / 24 production contre 18 / 26 ; sans
    incidence sur le constat.)
  - `dynPrecHook` : **tous** les sites listés. Les seules affectations sont `= nil` (sauvegarde /
    restauration) plus le setter mort `components_movement.go:64`. **Aucune affectation non-nil
    dans le dépôt, tests compris** — y compris par affectation directe intra-paquet, qui est le
    contournement qu'un test Go pourrait employer. Idem `repTraceHook` (`default_state.go:85,94`).
  - `defaultStateBitsByTI` (`traverse.go:1160`) : peuplé uniquement par `SetDefaultStateBitsForTI`
    (mort) → les branches `traverse.go:1093` et `probe_export.go:101` sont inatteignables.
  - Règle 11 : `useLegacyAngularVel` (`traverse.go:1134-1140`) et `useBipedDefaultStateDeser`
    (`traverse.go:1049-1064`) ne portent **ni date de bascule, ni date de retrait, ni critère
    mesurable**, alors que `simStateComplete` (`traverse.go:1180-1196`), dans le même fichier,
    porte les trois. Le contre-modèle du rapport est exact.
- **Ce que l'auditeur n'a pas vu, et qui NE réfute pas mais corrige**
  1. **« 12 sites de sauvegarde/restauration » est faux : il y en a 8.** `grep -c savedDP` rend
     16 occurrences (`frame_chain_infer.go` 4, `frame_debug.go` 4, `frame_records.go` 6,
     `probe_export.go` 2) = **8 blocs save/restore** — ce qui est d'ailleurs exactement le nombre
     de plages que le rapport énumère lui-même (`:76-78`, `:349-351`, `:34-36`, `:127-129`,
     `:400-402`, `:458-470`, `:688-696`, `:68-70`). Le chiffre 12 contredit sa propre liste.
  2. **La « branche legacy fausse » ne meurt pas avec le drapeau.**
     `consumeObjectAngularVelocity` (le déser « OLD (wrong) ») est **du code vivant** : il est
     l'implémentation **correcte** d'i3 pour `ti=40`, via `consumeObjectAngularVelocityDynPrec`
     (`components_dynprec_orientation.go:194`), et son en-tête `:186-192` explique pourquoi
     (« la correction de 2026-07 était juste pour ti=35 et a cassé ti=40 »). Seule la branche
     `if useLegacyAngularVel` (`traverse.go:240-242`) est inatteignable. Le traitement proposé
     (« les deux drapeaux legacy disparaissent avec leur branche ») doit donc être reformulé.
  3. Le ratchet **porte bien** une date cible et un critère mesurable, mais pour l'**état global
     dans son ensemble** (D10), pas pour les drapeaux :
     `archlint/filmdec_package_vars_test.go` en-tête — « RETRAIT CIBLE : le jour où `filmdec` est
     dé-globalisé […] Critère mesurable : `LockProcessDecode` n'a plus de raison d'être ». Cela
     n'excuse pas les deux kill-switches, mais la formulation « le garde-rail gèle du code mort »
     doit se lire avec le fait que le ratchet, lui, est daté et sait redescendre.
- **Conséquence réelle** : un tiers de la surface publique de réglage de `filmdec` est
  inatteignable, deux bascules A/B sans date ni critère gèlent des branches inertes, et 8 blocs de
  sauvegarde/restauration promènent deux crochets prouvablement toujours nuls.

## Constat 3 — grammaire de la liste d'événements écrite six fois, `dom3` divergent : TIENT

- **Ce que j'ai vérifié**
  - Les 6 lectures du préambule 9 bits, lues une par une : canonique `event_list.go:56-66` ;
    inline `zoom_events.go:136-141`, `transloc_events.go:168-173`, `biped_pickups.go:211-216`
    (`Skip(1)` + `ReadBit()`), `fire_aim_modal.go:57-59`, `weapon_hits.go:208-210` (`Skip(2)`).
    **Deux conventions pour un même préambule** : confirmé au caractère près.
  - `PacketHeadEventType` : **1 seul appelant de production, lui-même** (`event_list.go:219`) ;
    14 appelants, tous dans des `_test.go` (`event_list_test.go` ×2, `vehicules_v6/v7_*_test.go` ×12).
  - Divergence `dom3` : `event_list.go:98` `dom3RefWidth = 7` contre `weapon_hits_decode.go:17`
    `lot1RefDomWidths{… 3: 8 …}` et `zoom_events.go:68` `zoomRefWidth{… 3: 8 …}` — les deux tables
    de production se réclamant de la MÊME source (« table 0x1451f98d0 » / « table des domaines de
    l'exécutable »).
  - « Rien de faux n'est servi aujourd'hui » : vérifié. `zoomRefDomains = [3]int{4, 8, 7}`
    (`zoom_events.go:65`) ; le seul appel de production de `lot1RefDom` est
    `weapon_hits.go:288` avec `dom=7`. Le domaine 3 n'est lu que par `boardRefs`
    (`event_list.go:292`), qui utilise la constante mesurée.
  - Dates : `git log --diff-filter=A` → `event_list.go` **2026-09-01** (`f32784673`),
    `transloc_events.go` **2026-09-03** (`fa09f4ee5`). Deux jours d'écart : exact.
  - Aucun garde-rail AST sur le préambule dans `internal/archlint/` (35 fichiers passés en revue).
- **Ce qui nuance, sans réfuter**
  - La divergence `dom3` **est documentée sur place** : `event_list.go:82-96` écrit que la largeur
    est runtime, illisible dans l'image statique, et que **7** est mesurée par un oracle
    indépendant (le siège : 5/6 d'accord à 7 bits, **0/6 à 8 bits**, 8 films / 22 embarquements,
    2026-09-02). Ce n'est donc pas une dérive silencieuse mais une valeur mesurée qui contredit
    deux tables recopiées — ce qui est le risque que le rapport décrit, mais la prose de
    `event_list.go:74` (« 2/3/5 = 8 ») décrit la table de l'exe, pas un défaut de mise à jour.
  - La « factorisation refusée par écrit » de `biped_pickups.go:207-209` est bien là, et sa
    justification est **aujourd'hui périmée** : elle dit « celle-ci n'avait aucun lecteur et la CI
    l'a relevée (`unused`) », alors que `eventPayloadStartBit` a désormais deux lecteurs
    (`event_list.go:247` et `:291`). Le rapport a raison sur le fond, et le motif invoqué est
    caduc — ce qui renforce le constat.
- **Conséquence réelle** : le modèle canonique existe, ne sert qu'à lui-même en production, et la
  sixième copie a été écrite deux jours après lui ; le domaine 3 porte déjà deux valeurs
  contradictoires, aujourd'hui sans effet parce que la production ne le lit pas.

## Constat 4 — neuf copies de la triple boucle du marcheur delta bipède : TIENT

- **Ce que j'ai vérifié**
  - `grep -n "bipedHeaderBits + bipedIndexBits\*bipedMinMaskCnt" *.go | grep -v _test` → **9**
    exactement : `ability_charges.go:162`, `ability_impulses.go:154`, `ability_rank.go:200`,
    `camo_state.go:138`, `grapple_state.go:131`, `held_weapon_changes.go:242`,
    `inventory_delta.go:223`, `offline_biped.go:220`, `unit_equipment_scan.go:86`.
  - Symétriquement **9** appels de production à `matchBipedHeader` (mêmes fichiers).
  - **Je ne me suis pas contenté de compter les littéraux** : `diff` du squelette entre
    `camo_state.go:130-170` et `grapple_state.go:123-163`. Les lignes
    `for _, c := range chunks` → `fc.ChunkAt(c)` → `if !ok { continue }` → `for _, pk := range pks`
    → `if pk.Type != PacketTypeDelta { continue }` → `pay := pk.Payload(data)` →
    `total := len(pay) * 8` → `for p := 0; p+minRecord <= total;` → `matchBipedHeader(...)` →
    `if !ok { p++; continue }` → `p = i0 + lay.TotalBits()` sont **identiques caractère pour
    caractère**. Seuls le crochet installé et le corps de visite diffèrent — c'est-à-dire
    exactement le découpage `walk(visit)` que le rapport propose. Il n'y a pas de « grammaire
    réellement différente par événement » qui justifierait la recopie du squelette.
  - Aucun garde-rail dans `internal/archlint/` sur `matchBipedHeader`.
- **Conséquence réelle** : toute règle de porte nouvelle (bande de slots, filtre de région, borne
  de déroulage) devra être posée neuf fois et prouvée neuf fois, sans filet pour dire qu'une copie
  a été oubliée.

## Constat 5 — `killcollector/positions.go` sans `LockProcessDecode` : TIENT (gravité → P2)

- **Ce que j'ai vérifié**
  - `grep -rn "LockProcessDecode()"` sur tout `apps/go-api`, hors déclaration : **3 preneurs de
    production** — `replay/build_from_film.go:50`, `killsource/decode.go:78`,
    `killcollector/hits.go:106` — et environ 240 sites de test. `positions.go` n'y est pas.
  - Le contrat est bien celui que cite le rapport : `decode_gate.go:16-18`, mot pour mot
    (« tout chemin qui enchaine les balayages de ce paquet (Scan*, walk killsource) acquiert ce
    verrou pour TOUTE la duree du decodage d'un film — jamais par sous-appel »).
  - L'asymétrie est visible dans la prose des deux frères : `hits.go:101-102` — « **Tient le
    verrou de decode du process** (les parametres de replication de filmdec sont des globaux de
    paquet) » ; le godoc de `buildPositionRows` (`positions.go:163-170`) parle du partage du film
    déjà chargé et **ne dit rien du verrou**. Aucune justification écrite de l'écart.
  - **Garantie amont — je l'ai cherchée et elle tient aujourd'hui** : `grep "go func\|errgroup\|
    sync.WaitGroup"` sur `internal/sync/killcollector/*.go` non-test → **aucun**. `CollectMatch`
    n'est appelé que depuis `credit.go:342` et `roster.go:210`, deux boucles séquentielles.
    `sync.Mutex` n'est pas réentrant, mais `killsource.Decode` relâche avant de rendre : prendre
    le verrou dans `buildPositionRows` ne provoquerait pas d'interblocage.
- **Ce que l'auditeur n'a pas vu**
  - **Un garde-rail du verrou existe déjà**, contrairement à « sans garde-rail » / « il en manque
    un » : `replay/world_object_precision_guard_test.go:107-112`
    (`TestBuildFromFilmWiresWorldObjectPrecision`) lit la source de `BuildFromFilm` et **échoue**
    si `filmdec.LockProcessDecode()` n'apparaît pas AVANT l'installation des largeurs
    (« l'installation des largeurs précède la prise du verrou de décodage : le descripteur est un
    global de paquet, il ne s'écrit que sous le verrou »). Il est simplement borné à une fonction,
    donc il ne couvre pas `positions.go` — le geste manquant est une **généralisation** d'un
    garde-rail existant, pas la création d'une catégorie absente.
- **Pourquoi j'abaisse la gravité** : le rapport établit lui-même qu'aucune donnée fausse n'est
  servie ; j'ai confirmé qu'aucune concurrence n'est structurellement possible dans ce paquet
  aujourd'hui (0 goroutine, appelants séquentiels) ; et le contrat dispose déjà d'un garde-rail
  partiel. C'est une violation de contrat latente et une dette de garde-rail — P2, pas P1.
- **Conséquence réelle** : le contrat écrit du paquet est violé sur un des quatre chemins qui
  décodent, sans effet observable aujourd'hui, et il faudra re-prouver l'absence de goroutine à
  chaque lot tant que le garde-rail reste borné à `BuildFromFilm`.

## Constat 6 — « le seul test qui confronte le décodeur au registre RÉEL est éteint » : RÉFUTÉ

- **Ce que j'ai vérifié**
  - Le fait de départ est exact : `registry_test.go:38` porte
    `const chunk00Dir = "c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950"`
    et `:43` `t.Skipf("chunk_00 absent (%v)", err)`. `TestParseRegistrySynthetic` fabrique bien ses
    blocs avec `archetypeBlockSize` / `registrySlotSize`, donc il est auto-validant sur ces
    constantes. Tout cela tient.
- **Ce que l'auditeur n'a pas vu — et qui casse le « seul » ET la conséquence**
  - `TestGoldenMiniBobine` **confronte le décodeur au registre RÉEL du MÊME film `000d5950`,
    inconditionnellement, en CI** :
    - `killsource/testdata/minibobine_000d5950/` contient **`chunk_00.bin`, 435 425 octets**
      (`ls -la`) — le rapport G6 le sait, il l'écrit lui-même (« mini-bobine `killsource` 3,8 Mo
      (**avec** chunk_00) »), mais n'en tire pas la conséquence ici ;
    - `killsource/world.go:58` : `reg, err := filmdec.ParseRegistryChunk(f.chunks[0])`, puis
      `world.go:71` `tl.w = filmdec.NewWorld(reg)` ;
    - le registre pilote ensuite toute la marche : `traverse.go:1106` `arch, ok :=
      reg.Archetype(int(t.TypeIndex))` → `traverse.go:1305` `consumeByNameCapturing(br,
      arch.Components[i], …)`, atteint depuis `frame_records.go:173` / `:626`
      (`traverseComponentLoop`), eux-mêmes atteints par `killsource/walk.go:65` (`DecodeFrameRecords`)
      et `:77,104,116` (`TryDeltaAt`) ;
    - `minibobine_test.go:106-124` : `t.Fatalf` si la bobine est absente, plancher
      `len(res.Kills) >= miniBobinePlancher` (10), puis `comparerGolden` sur
      `testdata/minibobine.golden` qui fige **10 lignes publiées en entier** (instant, victime,
      tag, étiquette, catégorie, crédit, voie, divergence), 3 ancres Theater, la ventilation par
      voie et un contrôle négatif (« 5484 paquets sans événement · 7 687 528 bits balayés ·
      **0 candidat accepté** »).
  - **Il gate réellement** : le job `go-coverage` lance `go test … -tags=integration … ./...`
    (`.github/workflows/ci.yml:340-350`) — donc `internal/games/halo_infinite/film/killsource` —
    et fait `exit $rc` (`ci.yml:356`), l'échec n'est pas toléré.
  - Donc la conséquence écrite — « Un `parseRegistry` qui décalerait le bloc d'archétype, ou un
    `archetypeBlockSlots` faux, **passe la CI** » — est **fausse** : soit `ParseRegistryChunk`
    échoue et `newTimeline` rend `errRegistry` (→ `t.Fatalf("Decode sur la mini-bobine")`), soit
    les noms de composants se désalignent, `consumeByName` désynchronise, les lignes tombent sous
    le plancher de 10 et les 10 lignes figées changent.
- **Ce qui reste vrai** (et mérite d'être reformulé, pas remonté) : aucun test de CI n'**assertionne**
  les indices nommés de l'archétype bipède (`i0`/`i4`/`i11`/`i43-46`), ni l'empreinte du registre
  réel — `TestRegistryFingerprintOnFilm` (`registry_fingerprint_test.go:117`) est gardé par
  `DELTA_WITNESS_FILM` **et ne porte aucune assertion sur l'empreinte** (seulement `t.Logf` sur la
  concordance). Le traitement proposé (repointer `chunk00Dir` sur la mini-bobine versionnée,
  `t.Fatalf` au lieu de `t.Skipf`) reste bon ; sa justification, non.
- **Conséquence réelle** : un test utile est éteint par un chemin absolu, mais le registre réel du
  même film est bel et bien décodé et figé en CI par un autre chemin — le trou porte sur des
  assertions nommées, pas sur l'absence de tout filet.

## Constat 7 — « l'oracle Cheat Engine versionné n'est jamais consommé » : RÉFUTÉ

- **Ce que j'ai vérifié**
  - Les faits matériels tiennent : `git ls-files` confirme les deux fichiers suivis ;
    `ls -la` donne **11 418 o** (`kf_capture_sample.txt`) + **7 286 o** (`kf_slot0_live.bin`) =
    18 704 o. `default_state_ti42_oracle_test.go:62-64` skippe si `TI42_CAPTURE`/`TI42_BUFFER`
    sont absents, et `grep -rn "TI42_CAPTURE\|TI42_BUFFER" .github/ Makefile scripts/` ne rend
    **rien** — aucune ressource CI ne les pose.
- **Ce que l'auditeur n'a pas vu — et qui casse la conséquence**
  - **`TestTI42WidthOracle` ne contient AUCUNE assertion de mesure.** Ses deux seuls `t.Fatalf`
    (`:65`, `:69`) sont des erreurs d'E/S (« tampon illisible », « capture illisible »). Tous les
    résultats mesurés sortent en `t.Logf` : `:71` (frontières), `:74` (en-têtes réconciliés),
    `:182` (largeurs par ti), **`:222` (`ATTERRISSAGE … — ti=42 : %d exacts sur %d records`)** et
    `:228` (contrôle croisé ti=37). Il n'existe ni seuil, ni comparaison au témoin faux, ni
    `t.Errorf`. **Dégardé, ce test ne pourrait toujours pas échouer** : il imprimerait des taux.
  - La phrase centrale du constat — « C'est le seul mécanisme du périmètre capable de faire
    échouer une **largeur** de désérialiseur ; il ne s'exécute nulle part » — est donc doublement
    fausse : il ne peut faire échouer aucune largeur, garde ou pas.
  - **Contradiction interne à l'audit** : ce fichier est, par construction, l'un des « 85 fichiers
    de test sans aucune assertion » que le même rapport classe en P2 n°1 (« ils ne peuvent pas
    échouer »). Il est ici promu en P1 comme l'unique filet de largeur.
  - **L'oracle a été consommé, et le dépôt le dit** : `.ai/V7.5/REGISTRE_REPORTS.md` l.227 —
    « L'oracle de LARGEUR live (`kf_capture_sample.txt`) est **ÉPUISÉ** et le négatif est publié :
    un seul record NEW `ti=42`, porte ouverte. » Pour cet archétype, les « 400 frontières » ne se
    traduisent pas en 400 largeurs testables : la population utile est d'**un** record. (Une
    seconde entrée, l.509, du 2026-08-27, le désigne comme le levier de reprise d'un lot futur —
    une décision datée, avec sa condition de reprise, exactement le motif d'écart que l'audit
    applique ailleurs.)
- **Conséquence réelle** : deux fichiers de 18 Ko sont versionnés et lus par un instrument gardé
  qui n'assertionne rien ; le dégarder ne créerait aucun gate, et le registre daté du dépôt note
  déjà que leur contenu exploitable pour `ti=42` est épuisé.

## Constat 8 — 26 des 30 familles de balayage sans test octets réels en CI : TIENT

- **Ce que j'ai vérifié**
  - Les 4 familles couvertes : `equivalence_minifilm_test.go:138,142,146,162`
    (`ScanFilmFireEvents`, `ScanFilmGrenadeThrows`, `ScanFilmKeyframeLoadouts`,
    `ScanFilmProjectiles`), redoublées sous forme `Film` par `zero_disque_test.go:171-195`.
    Les deux fichiers ne contiennent **aucun `t.Skip`**.
  - **J'ai ouvert les trois goldens que le mandat me demandait de vérifier, et aucun n'élargit
    le compte** :
    - `killsource/minibobine_test.go` : exerce `Decode`, `DecodeFrameRecords`, `TryDeltaAt`,
      `ParseRegistryChunk` — mais **aucune famille `filmdec.Scan*`** (`grep "filmdec\.[A-Z]"` sur
      `walk.go` : `NewBitReader`, `DecodeFrameRecords`, `TryDeltaAt`, `DefaultFrameConfig`).
    - `filmdec/film_shims_test.go` et `replay/film_shims_test.go` : de simples **enveloppes `dir`
      réservées aux tests** (`bipedSlotBandDir`, `vehicleArchetypeDir`, …), sans un seul test.
    - `replay/golden_inputs_test.go` : son propre en-tête l.10-13 le disqualifie —
      « C'est un ÉTAGE 1 : les entrées DÉJÀ DÉCODÉES […] Il verrouille l'ASSEMBLAGE et rien
      d'autre — **un changement du DÉCODAGE ne le fait pas bouger**, c'est le travail de
      l'étage 2 ». Il appelle bien `ScanFilmCamoStates`, mais uniquement dans
      `TestGoldenInputsRegenerate`, gardé par `REPLAY_FILM_DIR` (`:1203-1208`).
  - **Le constat est en réalité plus fort que ce qu'il annonce** : les points d'entrée de famille
    eux-mêmes (`ScanCamoStates`, `ScanAbilityCharges`, `ScanZoomEvents`,
    `ScanTranslocatorTeleports`, `ScanBipedPickups`, `ScanInventoryDeltas`,
    `ScanHeldWeaponChanges`, `ScanWeaponShots`) ne sont appelés par **aucun** test de `filmdec`,
    gardé ou non ; seules les enveloppes `ScanFilm*` le sont, depuis des instruments gardés.
  - Le dénominateur, en revanche, est **sous-estimé** : en dédupliquant les paires
    `ScanX`/`ScanFilmX` et en fusionnant les variantes `ForBand` avec leur base, je compte
    ~34 familles de production, pas 30. L'erreur va dans le sens qui minimise l'écart.
- **Conséquence réelle** : quatre familles de balayage sur une trentaine sont confrontées à des
  octets réels dans un job qui échoue ; les autres n'ont, en CI, que des désérialiseurs isolés sur
  octets synthétiques, jamais la marche de records qui les assemble.

## Constat 9 — garde-rail de `consumeByName` différentiel entre deux copies : TIENT (gravité → P2)

- **Ce que j'ai vérifié**
  - Les deux tests sont bien différentiels : `capture_test.go:19-45` compare `consumeByName` à
    `consumeByNameCapturing` (dont `capture.go:30-44` montre qu'il **délègue** à `consumeByName`
    hors de ses 4 `case`), et `components_hooks_test.go:68-90` compare avec et sans crochet. Une
    largeur fausse est invisible des deux côtés : le raisonnement du rapport est juste.
  - **Le trou de table est structurellement confirmé** : `ecs_table_guard_test.go:81` construit
    `ecsRow` à partir de `c[3], c[5], c[9], c[10], c[15]` et la structure (`:32-42`) n'a **aucun
    champ** pour `c[7]` (`grammar`) ni `c[8]` (`bits_typ`). Aucun contrôle ne peut donc confronter
    le code à ces deux colonnes. G2 est bien gardé par `ECS_TABLE_FILM`.
  - Comptages : 526 composants `porte` ; **263** portent `grammar` ET `bits_typ` (les deux colonnes
    sont renseignées ensemble) ; 193 `case` entre `traverse.go:216` et `:1020` ; 3 sites
    d'assertion `BitCost` dans 2 fichiers.
  - **Erreur de méthode sans conséquence** : l'`awk` de reproduction teste `$8`, qui est la colonne
    `grammar` (l'en-tête donne `… status(6) deser_addr(7) grammar(8) bits_typ(9) …`), pas
    `bits_typ`. Les deux colonnes étant co-renseignées, le résultat 263 est néanmoins correct.
    En revanche le traitement proposé (« pour chaque composant dont `bits_typ` est un entier »)
    ne porterait que sur **179** composants, pas 263 : les 84 autres portent une expression.
- **Ce que l'auditeur n'a pas vu, et pourquoi j'abaisse la gravité**
  - « **Aucun** contrôle ne confronte le code aux colonnes » est vrai ; « une largeur fausse
    passe » ne l'est pas universellement. `consumeByName` **est** exercé sur des octets réels en
    CI par `TestGoldenMiniBobine`, via `killsource/walk.go:65,77` →
    `frame_records.go:173/626 traverseComponentLoop` → `traverse.go:1305`. Une largeur fausse sur
    un composant effectivement présent dans les records de la mini-bobine désynchronise la marche,
    fait tomber les lignes sous le plancher de 10 et casse le golden figé.
  - La portée de ce filet est cependant **écrite et étroite**, et c'est ce qui empêche de réfuter :
    l'en-tête du golden dit lui-même « la calibration automatique tombe ici en PROFIL PLAT
    (l'échantillon du préfixe **ne rend aucun record de biped**) […] Seul `TestGoldenFilms`
    verrouille le balayage, sur les films entiers » — et `TestGoldenFilms` est gardé par
    `KILLSOURCE_FIXTURES`.
- **Conséquence réelle** : les deux garde-rails nommés ne peuvent pas détecter une largeur fausse
  et la table `ecs_table.tsv` n'est jamais confrontée au code ; le seul filet effectif est un
  golden dont la population, de son propre aveu, ne couvre pas l'archétype bipède.

## Bilan : 7 tiennent, 2 réfutés, 2 requalifiés

| # | Constat | Verdict |
|---|---|---|
| 1 | `entity.go`/`entity_quant.go` morts | **TIENT** (aggravé : supersession écrite en 2026-06-14) |
| 2 | 22 `Set*` morts + drapeaux sans date | **TIENT** (2 chiffres faux : 8 sites et non 12 ; la branche legacy ne meurt pas avec le drapeau) |
| 3 | 6 copies du préambule, `dom3` divergent | **TIENT** |
| 4 | 9 copies de la triple boucle | **TIENT** (squelette vérifié identique par `diff`) |
| 5 | `positions.go` sans verrou | **TIENT (gravité → P2)** — aucune concurrence possible, un garde-rail du verrou existe déjà |
| 6 | `registry_test.go` éteint = seul filet | **RÉFUTÉ** — `TestGoldenMiniBobine` décode le registre réel du même film, inconditionnellement, dans un job qui `exit $rc` |
| 7 | Oracle CE jamais consommé | **RÉFUTÉ** — le test n'a aucune assertion de mesure ; dégardé il ne peut pas échouer ; l'oracle est déclaré « ÉPUISÉ » au registre daté |
| 8 | 26/30 familles sans octets réels en CI | **TIENT** (dénominateur sous-estimé : ~34) |
| 9 | Garde-rail `consumeByName` différentiel | **TIENT (gravité → P2)** — le trou de table est réel, mais le golden inconditionnel n'est pas « aucun » filet |
