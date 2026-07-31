# HANDOFF — Arme à feu par kill (point d'entrée post-compact)

> **Branche / worktree** : `feat/weapon-attribution-v3` au chemin
> `.claude/worktrees/weapon-attribution-v3/`. ⛔ NE JAMAIS écrire dans le repo principal (`main`) — toujours dans CE
> worktree. (Leak nettoyé 2026-06-12 : des fichiers avaient atterri sur `main`, retirés.)
> **But** : extraire l'arme à feu de CHAQUE kill, 100% offline, scalable. Répondre en français. Demander avant tout commit.

## 1. LIVRÉ — pipeline warp (exploitable maintenant)
**`apps/go-api/cmd/tmp_offwarp <matchID>`** : arme par kill, pur offline.
- Décode `0xd2` (dégât) : attaquant = `bitsAt(pl,36,5)>>1` ; arme = famille `variant_name` (bit ~41) + suffixe 0x42c9679f.
- Roster type-8 (bit-scan LE, `u64LE` MSB-first-par-byte) → slot↔xuid. Kill-feed = `analysis.ParseHighlightEvents`.
- **Warp LINÉAIRE** packet-ts→TimeMS (fit nearest×3 puis last-before×3) + **arme = DERNIER dégât du tueur AVANT le kill**.
- **Résultats** : 96% par-kill sur 000d5950 (Fiesta), 90-97% attribution sur 8 matchs ; validé vs ground-truth live.
- **Limite** : Team Slayer 9b191a7f = **58% par-kill** (erreur dominante **BR↔MA40**). Cause = imprécision du warp
  (2 horloges) quand un joueur alterne BR/MA40 sur ~1-2s. **PAS du held-weapon** (méthode = source de dégât). Mapping
  joueur = identité (Hungarian confirmé). Métrique fiable = pont-tsc par-kill (`tmp_offwarp` : 94% sur l'oracle).

## 2. VOIE EXACTE (en cours) — corrélation MÊME-HORLOGE
Plan : **`.ai/V7.5/killweapon/PLAN_SAMECLOCK_ATTRIBUTION.md`** (lire en entier). Idée : corréler dégât↔MORT dans la MÊME horloge
(flux), pas un warp. Le **dead-state** (`filmdec` `DeadState{EnumA=victime, EnumB=tueur, GID}`) vit dans le même flux
que les `0xd2` → corrélation par packet-ts = exacte, zéro warp. Preuve que ça marche : live dual-hook = 97/98.

**BLOCAGE (diagnostic complet fait 2026-06-12)** : décodeur FRAME `filmdec` désync.
- #1 détecté = `i63 biped-action`, mais la cause est **EN AMONT** (composants à partie variable).
- **Fausses pistes éliminées empiriquement** : `defaultReplRange` (sweep plat), `recordStateParam` (sweep plat).
- **COUPABLES priorisés** (`tmp_maskcorr`, corrélation présence-désync) : **i23 unit-malleable-property** (+0.19),
  **i0/i1 object-*-dynamic-precision** (largeurs quantif position), **i5 shield-vitality**. Plusieurs composants.
- **PROCHAIN PAS** : porter bit-exact, dans l'ordre **i23 → i0/i1 → i5** (décompiler chaque FUN, trouver la partie
  variable, calibrer — possiblement mesure CE pour la vérité-terrain des largeurs). Valider que le compte de records
  propres MONTE (`tmp_framedesync`/`tmp_replsweep`) et que `tmp_killeridx` sort EnumA/EnumB dans 0-7 matchant le kill-feed.
- **Outils diag en place** : `tmp_framedesync`, `tmp_i63debug` (+biDebug), `tmp_maskcorr`, `tmp_replsweep`, `tmp_killeridx`.

## 3. PRODUCTIONISATION (si on garde le warp)
Plan : **`.ai/V7.5/killweapon/PLAN_WEAPON_PER_KILL_PRODUCTION.md`**. Point d'insertion TRACÉ :
`internal/sync/backfill_weapons.go::BackfillWeaponKillsForMatchAll` (télécharge déjà le film complet via `GetMatchFilm`,
écrit dans table `weapon_kills`, déclenché en PostSync). **Remplacer l'attribution interne** (ancien fire-scanner
`ScanFireEventsAll`+`CorrelateKillsGlobal`, rejeté) par le pipeline `tmp_offwarp`. Piège : 3 numérotations player_index
à réconcilier (getXuidToPI vs roster type-8 vs R5).

## 4. Capture live exacte (97/98) — réutilisable
**`tools/ce/filmdec_dualcap_capture.lua`** : 2 hooks pure-read (dégât FUN_1407e00ac RVA 0x7E00AC, kill FUN_1406730c4
RVA 0x6730C4 ; base+RVA + vérif octets ; RDTSC ; null-checks). Usage : `captureDual(150,"prefix")` dans CE (déjà
attaché à Halo via le bridge MCP `cheatengine`), JOUER le film, dump `tools/ce/<prefix>_{dmg,kill}.bin` dans le REPO.
~12 min/match. Setup CE/Ghidra MCP opérationnels (mémoires reference_cheatengine_mcp_setup / reference_ghidra_mcp_setup).

## 5. Ground-truth & build
- Captures oracle : `tools/ce/{dmgcapture_run2,killcapture}.bin` (000d5950, 97/98) + `9b191a7f_{dmg,kill}.bin` (Team Slayer).
- Décodage tmp_dualcap : dmg 32o `[0]atk [4]vic [8]fam [12]sfx [16/20]rdtsc` ; kill 16o `[0]vic [4]kil [8/12]rdtsc`.
  idx = `(handle-base)/0x10002`, base 0xEC500000 (dmg) / 0xE1500000 (kill). slot R5>>1 == idx (validé).
- **Build** : outils `filmdec` (pas de DuckDB) → `CGO_ENABLED=0 go run ./cmd/...`. Outils DuckDB → `CGO_ENABLED=1` +
  `export PATH=/c/msys64/ucrt64/bin:$PATH`. Vitest/front hors sandbox. gofmt requis (pre-commit hook).

## 6. Faits RE clés (ne pas re-chercher)
- Le film ne stocke AUCUN lien direct kill↔arme (kill-event component `FUN_14104bd08` = victime+tueur+%dmg+final-blow
  +assistant, PAS d'arme ; `0xd2` = arme sans victime exploitable). Le replay recalcule au runtime (= le live dual-hook).
- ⛔ **JAMAIS held-weapon** (rejeté fermement par l'user, multiples fois). La méthode = source de dégât.
- Devinage marqueur type-0 pour le kill-event (0xe6=98/0xc7=97) = échoue (en-tête réplication variable).

## 7. État commits (worktree, propre)
Du plus récent : `3fb002dff` (coupables FRAME) · `dc15f443e` (diag i63) · `ce008d790` (voie same-clock) ·
`3c881e0d4` (2e ground-truth + dualcap script) · `f73a2b4a5` (warp 96%). Tout poussé sur origin.
