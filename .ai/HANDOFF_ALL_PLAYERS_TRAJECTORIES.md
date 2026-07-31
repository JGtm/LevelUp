# HANDOFF — Trajectoires de TOUS les joueurs décodées (Objectif 1)

> 2026-07-04. Débloque le replay 2D multi-joueurs. Fait suite à
> `HANDOFF_KEYFRAME_LIVE_CAPTURE.md` (framing keyframe résolu).

## Le verrou « seulement 2 joueurs » = chemin de réplication (RÉSOLU)

Le plafond historique (« on décode 2 joueurs, pas les autres ») n'était PAS une limite du
film : c'est un artefact de **chemin de dispatch**. Preuves runtime (CE, film courant,
9 joueurs actifs = 8 humains + 1 bot) :

| Hook (base + off) | Rôle | Observé |
|---|---|---|
| `FUN_1406cbaa0` (0x6CBAA0) | dispatch record des entités **possédées** | 2 slots seulement (POV + bot) |
| `FUN_1406cfe44` (0x6CFE44) | deser position i0 | ~9/frame = **TOUS** les joueurs |
| `FUN_14076cb60` (0x76CB60) | boucle de composants | appelée depuis `+6CAC80` pour tous |
| `FUN_1406caad8` (0x6CAAD8) | **processeur de delta COMMUN** | **12 bipeds** (ti=35) |

`FUN_1406cbaa0` est le chemin des 2 entités dont le client enregistreur a l'**autorité**
(ton pion POV, prédit ; le bot, IA simulée hôte). Les humains distants sont des proxies →
chemin principal. Mais **tous** convergent sur `FUN_1406caad8` (le processeur de delta) puis
`FUN_14076cb60` → `FUN_1406cfe44`. Mon ancien décodeur suivait `FUN_1406cbaa0` (2 possédés)
→ d'où le plafond. Cf. mémoire `project_filmdec_two_players_ownership_views`.

## Capture tous-joueurs (CE)

Hook non-bloquant sur `FUN_1406caad8`, filtré `ti=35` (biped) via la table d'entités
`*(RCX+0x20) + slot*200 + 4` :
- slot = `RDX & 0x3fffffff` ; reader = `param_5` (pile `RSP+0x28`) ; data `reader+0x08`,
  bit-pos `reader+0x2c`.
- 8000 records capturés, 11-12 slots (les slots MIGRENT au fil du match = respawns : un
  joueur mort réapparaît sur un nouveau slot).
- Fixture : `.ai/re_dump/allbipeds_capture_sample.txt` (2000 records ; format
  `id bp%8 byteoff hex96`).

## Décodage offline (cmd/tmp_kfmatch)

Chaque delta biped = `consumeMask` + boucle de composants (ti=35 lié dans le `World`), PAS
de header. Le i0 (composant index 0) émet la position AVANT tout desync éventuel :
- `World.BindFull(id, 35)` puis `DecodeDeltaRecordAt(bytes, bp%8, w, slot)`.
- `SetPositionCaptureHook` capture le i0 ; `SetFilmComponentCorruptionCheck(true)` (mode film).
- **7808/8000 records portent une position i0**, 42 desync (0.5 %), **11 slots**.
- Tous les i0 sont `PosKindAbsolute` (position absolue quantifiée), pas de delta.

### Calibration de la largeur d'axe (i0 absolu)

`TraversalPrecision.AxisW` : balayage 6→18 sur le pas 3D moyen entre positions
consécutives (un joueur bouge peu par frame). **Transition de phase nette : pas moyen 179
(W=14) → 0.07 (W=15)** = alignement des bits. W=15 :
- trajectoires lisses (pas max 0.23 u/frame ~14 u/s), **aucun wrap**.
- spans physiques : X~24-80, Y~30-125, Z~4 (mouvement horizontal dominant, Z quasi-plat).

⇒ **W=15 = largeur d'axe de cette map** (uniforme X/Y/Z). W=16 marche aussi (Z plus ample).
La largeur est map-spécifique (config de précision de réplication, tables dumpées
`prec_default_1445cc9e0.bin` / `prec_perindex_1445ccbe0.bin`).

## CARTE 2D (Obj1 finition) — ÉTAT ET PROCHAINE ACTION

Ce qu'on a : par slot, une suite `(X, Y, Z)` par frame, dans le repère monde quantifié du
composant i0. Dequant (`position_capture.go` `dequantWorldAxis`) : `world = min + step*(q +
0.5)`, `step = (max-min)/2^15`, `WorldPositionRange[axis] = {min,max}`. Les i0 sont tous
`PosKindAbsolute` → valeur prise telle quelle (pas d'accumulation sur cette capture).

Ce qui manque pour poser sur la carte 2D :

1. **Le RANGE monde exact (min/max par axe).** Aujourd'hui `WorldPositionRange =
   QuantRangeCliffhanger` (hypothèse) → la FORME des trajectoires est bonne (lisse, physique)
   mais l'échelle/offset absolus ne sont pas garantis. Le vrai range est index-sélectionné
   dans la table de plages `DAT_14462cbe0` (dumpée `scratchpad/prec_ranges_14462cbe0.bin`,
   18432 o) par le champ `index` du i0 (idx=0 = bornes réelles map ; idx!=0 = ±20000 off-map).
   Action : lire l'entrée idx=0 (format à confirmer : min/max f32 par axe ?), remplacer
   `WorldPositionRange`, re-décoder -> coords en unités monde réelles.
2. **Aligner sur la géométrie de la map.** La carte 2D vient de `.module` (mémoire
   `project_map_geometry_from_modules` : module->Kraken->walker->`runtime_geo`->carte,
   `cmd/tmp_walker`/`cmd/tmp_geores`), dans le MÊME repère monde. Range calé (point 1) -> les
   `(X,Y)` joueur tombent dans le repère carte (au plus une rotation/flip à vérifier). Overlay
   = superposer les polylignes par slot sur le SVG/geo de la map.
3. **Attribution slot->gamertag.** Les slots MIGRENT (respawn = nouveau slot ; la capture passe
   de 0x28E-0x29D à 0x299-0x2A6). Pour nommer : capturer aussi le record NEW/spawn (xuid dans
   le default-state, via `FUN_1408f1aa4`) OU croiser l'ordre de spawn avec le roster `type-8
   PLAYER_METADATA`. Non bloquant pour un replay anonyme (11 trajectoires déjà séparées par slot).

Chemin le plus court : point 1 (lire le range dans la table dumpée) puis point 2 (overlay).
`allbipeds_capture_sample.txt` + `cmd/tmp_kfmatch` suffisent pour itérer offline.

## RESTE À FAIRE (autres axes)

4. **Objectif 2 (kill feed) — DÉJÀ RÉSOLU AILLEURS + impasse dead-state re-confirmée.**
   Lire `README_KILLWEAPON_INDEX.md` AVANT toute reprise. L'arme par kill est DÉJÀ résolue
   offline via le **warp** (`cmd/tmp_offwarp`, **96% Fiesta / 58% Team Slayer** : marqueurs de
   dégât `0xd2` = source de dégât `FUN_14080c1f8` ⋈ kill feed `chunk_27` type-9), pas encore en
   prod (`internal/sync/backfill_weapons.go`, cf `PLAN_WEAPON_PER_KILL_PRODUCTION.md`). Ma
   piste dead-state de cette session = l'impasse DOCUMENTÉE (§2 piste 3) : le `+0x10` (mon
   `0x6AE4A160` constant, identique sur 2 sections) est le "GID CONSTANT = pas l'arme" déjà
   établi par capture CE. Le dead-state porte tueur/victime/**catégorie** (mêlée/lancé), PAS le
   modèle d'arme. **Frontière ouverte** : battre le 58% Team Slayer via attribution
   MÊME-HORLOGE (§3 Phase 3) = lire tueur/victime dans le flux FRAME (même horloge que le
   `0xd2`), ce que la campagne précédente ratait (dead-state offline = garbage, "binding World
   incomplet slot 1038" → cascade de desync).

   **DÉBLOCAGE 2026-07-04 (CE, film joué) — le kill-event same-clock est LOCALISÉ** (cf mémoire
   `project_killfeed_sameclock_localized`). Le kill-event `FUN_14104bd08` (RVA 0x104BD08) FIRE
   pendant le replay ; reader = R9 (data `R9+0x08`, bit-pos `R9+0x2c`) ; **son buffer
   `0x2B98D1F0000` = EXACTEMENT le même que le dégât `0xd2` (`FUN_14080c1f8`)** (double-hook) →
   kill-event et dégât dans le MÊME flux = MÊME HORLOGE. Le "mur 2 horloges" tombe : la campagne
   utilisait `chunk_27` (temps-jeu) faute d'avoir localisé le kill-event frame-clock. Dataset
   250 events `.ai/re_dump/killevents_sample.txt` : DMG portent tous `42c9679f` (arme, §169),
   KILL clusterisent (tueur/victime/type). **Build offline restant** : décoder le kill-event
   (grammaire §174) dans le MÊME flux frame que le `0xd2` (déjà décodé par `tmp_offwarp`),
   corréler par proximité même-frame → arme par kill, viser >90% Team Slayer.
5. **100 % offline (sans CE)** : décodage self-navigué des chunks téléchargés = porter les
   desers de composants manquants (desync résiduel 0.5 %) pour retrouver les frontières de
   records sans capture CE. Voir handoff keyframe.
