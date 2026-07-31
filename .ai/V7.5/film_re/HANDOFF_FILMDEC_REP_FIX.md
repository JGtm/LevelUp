# HANDOFF — Décodage film Halo : rep bit-exact + grind composants restant

> Post-compact. Branche : **feat/filmdec-continuation** (worktree `.claude/worktrees/filmdec-continuation/`).
> Objectif final : **trajectoires 2D des 8 joueurs** depuis le film `data/cache/film_chunks/000d5950/` (Cliffhanger).
> Contrainte user : décodage **offline-pur + universel** (CE seulement pour lire des constantes de build / valider, jamais arbitrer).

## MISE À JOUR 2026-07-03 — 8/8 bipèdes ATTEINTS + mur prouvé (structure≠position)
- **Inférence de chaînes récursive** (`frame_chain_infer.go`) + **resync validé** (`validatedResync`, activé via
  `SetInferChain(true)` + `SetInferResyncTargets(bipedSlots)`) : desync 27%→**18.5%**, **les 8 bipèdes sont maintenant
  atteints** (vs 2). Zéro faux-positif (soft-binding + alignement confirmé unique).
- **MUR PROUVÉ empiriquement** : validité STRUCTURELLE ≠ correction de POSITION. Le scan exhaustif (`ScanFrameTargets`) atteint
  la queue mais décode des positions FAUSSES (bruit) même avec confirmation forte → seul le vrai alignement séquentiel donne
  la vraie position. **Diagnostic `cmd/tmp_bipedreach`** : la donnée des affamés EST présente (~680-920 deltas-position/slot),
  non atteinte car coupée par desync amont.
- **⇒ Le livrable « 8 trajectoires propres » = réduire le desync séquentiel à ~0 en PORTANT les composants restants**
  (liste finie : ti=37 i29/i30, ti=35 i55/i57/i60/i63, ti=14 i1, ti=5 i7/i22/i24). C'est le §3 ci-dessous + le grind portage.
- Outils : `cmd/tmp_traj INFER=2 INFERRESYNC=1`, `cmd/tmp_bipedreach`, `cmd/tmp_bipedharvest`, `cmd/tmp_deadreckon CHAIN=1`.
  Tests : `frame_chain_infer_test.go`. Détail : thought_log 2026-07-03.

## 0. TL;DR — où on en est (état 2026-07-02, complété par la MAJ ci-dessus)
- **Mur architectural TOMBÉ** : (1) table ECS runtime universelle extraite (mapping typeIndex→deser), (2) bug racine du rep biped corrigé → **rep bit-exact (166 bits = jeu)**.
- **Record NEW biped décode à +62 bits** (1986 vs 1924) : bloqué par 3-4 composants « action » (i60-i63) alignement-sensibles.
- **Livrable : 2/8 trajectoires** (POV slot519 = 181 pts OK, slot515 = 111). Les 6 autres bipèdes = deltas sparse/désync.
- **Point clé stratégie** : les trajectoires viennent des records **DELTA** (`decodeDelta`), PAS du keyframe. Le fix rep (records NEW) est correct mais n'avance pas `tmp_traj`. Le vrai chemin livrable = fiabiliser le décodage DELTA des 6 bipèdes.

## 1. Ce qui est FAIT (commité, ne pas refaire)
- **Table ECS universelle** : `DAT_144e61d88` (runtime) → descripteur par typeIndex → deser par composant (`vtable[0x28]`, ou `[0x30]` si thunk `FUN_14076ce9c`). Constante de build. Recette + procédure d'extraction CE : **`.ai/V7.5/film_re/RECETTE_DECODAGE_FILM_CHUNKS.md`** + mémoire `reference_ecs_runtime_deser_table`. Table des 64 desers du biped dans la recette §6.
- **Fix rep (commit 4d912cedb)** : `consumeBipedDefaultState` (default_state.go) sur-lisait 32 bits. Cause = le `consumeOpt32` final (ligne ~131, gated `uVar10>=12`) : `FUN_14080d69c` lit R(1) du flux, mais son `R(32)` (`FUN_14080d6f0`) opère sur un reader NON-film → 0 bit film. **Fix : opt32b lit R(1) SEUL.** Validé : rep = 166 bits = exactement le jeu (repEntry@6, repExit@172 capturés live).
- **i15 = FUN_1407ef088** (pas la mauvaise fonction de la note 2026-06-14) : le port matche déjà, commentaire corrigé.
- **i60/i61 desers corrigés (commit 39f5b4854)** : vérif STATIQUE des thunks en mémoire vive (disassemble) : i60 thunk `0x142f02434` → **FUN_142ED6D88** ; i61 thunk `0x142f02454` → **FUN_142ED6D20** (= `consumeSimulationStatePlayback`, correct).

## 2. L'oracle offline (pas besoin de CE pour itérer)
- **`cmd/tmp_reccheck/main.go`** contient les **bits bruts d'un vrai record NEW biped** (capturé live, `acc`+`used`+`bytes`) et sa **longueur exacte = 1924 bits** (const `target`). Reconstruit le flux, lance `TraverseEntity`, dump composants + largeurs + compare endBit à 1924.
- Actuellement : `endBit=1986` (desyncAt=-1, +62). Les composants i0-i59 décodent alignés ; i60-i63 sur/sous-lisent.
- Hook de trace du rep : `filmdec.SetRepTraceHook(func(label,pos))` (position bit après chaque champ). `LastRepVersion()`.

## 3. Le blocage restant : composants i60-i63 (« action »)
Réconciliation validée : **i60(~175)+i61(69)+i62(55)+i63(~119) = 418 = 1924−1506** (i60 démarre @1506 record-relatif).
Ces composants ne sont présents QUE sur des bipèdes EN ACTION (rare : i60=0 hit en 55s/374 records). Le film au repos ne les a pas.

- **i60 `simulation-state`** = `FUN_142ED6D88` — **STRUCTURE RÉSOLUE bit-près (2026-07-02, disasm-vérifiée)** :
  `R(1)` flag ; si 0 → `FUN_14058c250` (0 bit) ; si 1 → `2×FUN_1407f2058` (=R(1)[gate0→R5]) + `4×FUN_142ee2194`
  (=`FUN_1406d84b4` w=0x10 = **R(16)**) + `R(2)` + `R(2)` + `4×FUN_142ee2194` + `FUN_140c1e79c` (=R(1) ; si top-bit 0 →
  R(19) packed-dir ; **PUIS TOUJOURS R(8) magnitude** — la R(8) était OUBLIÉE par l'ancien port = la vraie cause du chaos)
  + **tail conditionnel**. Port complet = `consumeSimulationState` + `consume140c1e79c` (traverse.go).
  - **Tail = SEUL inconnu restant** : `FUN_140501798` = predicate FLOAT (0 bit) sur 2 vecteurs décodés ; si vrai →
    `FUN_14076e494` → soit `FUN_1411b259c` = **R(96)** (ÉCARTÉE par sweep), soit `FUN_14076e524` = **R(1)[+R(DAT_144632be0)]**
    où `DAT_144632be0` = largeur d'index **runtime** (0 en statique). ⇒ **CE-dépendant** (lire DAT_144632be0), localisé au bit.
  - Avec tail=0 : record décode TOUS composants (nComps 34, plus AUCUN desync), endBit=2022 (+98), i63 misaligné.
  - **i60 rendu honnête** : `simStateComplete=false` par défaut → décode la structure connue puis desync propre (pas de
    false-clean). L'oracle met `SetSimStateComplete(true)` + sweepe `simStateExtra` (largeur tail) pour explorer.
- **i61 `simulation-state-playback`** = `consumeSimulationStatePlayback` (FUN_142ED6D20) — CORRECT (1 ou 69 bits).
- **i62 `biped-slide`** = `consumeBipedSlide` (FUN_142f02978 → FUN_142f26ce8) — CORRECT (1/26/55 selon gates).
- **i63 `biped-action`** = `consumeBipedAction` (FUN_142f027f4 → FUN_142f26a20) — **structure CONFIRMÉE** (subblock
  FUN_142f21b10 = 3×R32 = 96 ; count1=R(4) ; loop1 [R7+R5 tag+body 0..5] ; count2=RAM popcount ; +96). Commun (count1=0)
  = 196. Le count1/tags dépendent de l'alignement amont (donc du tail d'i60).

**Diagnostic (affiné 2026-07-02)** : i60/i61/i62/i63 sont TOUS portés bit-près SAUF le handle-tail d'i60 (largeur d'index
runtime `DAT_144632be0` + predicate float). Aucune valeur de tail flat (sweep 0..130) ne donne endBit=1924 AVEC i63 propre
(count1=0) → le tail n'est pas flat, c'est la branche runtime `FUN_14076e524`. **Le mur est réduit à UNE valeur runtime.**

## 4. Prochaines étapes (ordre recommandé)
1. **Capturer les largeurs RÉELLES de i60,i61,i62,i63** sur un record en action, via CE (jeu lancé, film scrubé à un **moment d'action** : kill, glissade, capacité). Breakpoint sur les vtable[0x28] (thunks : i60=`0x142f02434`, i61=`0x142f02454`, i62=`0x142f02978`, i63=`0x142f027f4`), reader=RDX, lire `reader+0x2c` à l'entrée + retour → largeur exacte de chacun. Filtrer biped (`FUN_1408f1aa4` R9, R(6)@acc>>58==35). NB : i60 rare, patience.
   - Astuce : le deser i60 (`FUN_142ED6D88`) est le MÊME quel que soit l'archétype → capturer sa largeur+bits sur N'IMPORTE quel record avec i60 suffit (pas forcément biped), MAIS il faut R(1)=1 (bloc présent).
2. Avec les largeurs : compléter `consumeSimulationState` (i60) + corriger i63 (`consumeBipedAction`) bit-exact → `tmp_reccheck` doit atteindre **1924**.
3. Puis décoder le **keyframe entier** (binder tous les slots) — mais AUSSI/D'ABORD viser le vrai livrable :
4. **Trajectoires (le but)** : fiabiliser `decodeDelta` (records DELTA) pour les 6 bipèdes non-POV. `cmd/tmp_traj` = 2/8 aujourd'hui. Les deltas désync sur slots non-liés + desers de composants faux. C'est le grind qui donne les 8 trajectoires (pas le keyframe directement).
5. PISTE 2 bancarisée : aim vector cubemap (`FUN_1406d8288`) pour la direction de regard.

## 5. Outils / fichiers clés
- `cmd/tmp_reccheck` : oracle record NEW (RAW2 + target 1924). `cmd/tmp_traj` : trajectoires PNG (2/8). `cmd/tmp_bindtrace` : couverture frames (18065 propres). `cmd/tmp_repcheck` : valide le rep seul.
- `internal/analysis/filmdec/` : `default_state.go` (rep = FUN_140F44C38), `traverse.go` (TraverseEntity + consumeByName + consumeMask + i60-63 cases), `components_*.go` (desers), `frame_records.go` (decodeDelta + DecodeFrameRecords).
- Recette : `.ai/V7.5/film_re/RECETTE_DECODAGE_FILM_CHUNKS.md`. Plan CE : `.ai/V7.5/film_re/PLAN_CE_BREAKPOINT_VALIDATION.md`. Journal : `.ai/thought_log.md` (entrées 2026-07-01/02).
- Build : `CGO_ENABLED=0 go run ./cmd/tmp_reccheck` (le sandbox bloque CGO ; filmdec n'a pas besoin de CGO).

## 6. Commits (feat/filmdec-continuation)
- `4d912cedb` fix rep (opt32b R(32)) — LE gros fix.
- `39f5b4854` i60/i61 desers corrigés (vérif statique thunks).
- `a10e74840` wip port partiel i60 (non-convergent à l'aveugle) — dernier.

## 7. Pièges connus
- `desyncAt==-1` ≠ bit-exact : la boucle « finit » même en sur/sous-lisant (faux-propre). Toujours valider `endBit==longueur réelle`.
- Le clean-frame count (`tmp_bindtrace`) est dominé par les faux-propres → ne PAS valider un fix de deser dessus.
- Sweep d'un extra sur UN composant = chaotique si les composants suivants sont alignement-sensibles → besoin de la ground truth CE de TOUS les composants concernés.
- t.Gate (TraverseEntity, avant le masque) est un VRAI bit (le retirer casse). consumeMask = le lecteur de masque (1 gate + dense 64 / sparse 3+6N).
