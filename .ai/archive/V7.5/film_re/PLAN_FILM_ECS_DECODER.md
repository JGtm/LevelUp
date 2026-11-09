# PLAN — Décodeur ECS du film Halo (la « v3 réelle »)

> **Décision user 2026-06-05 (Option B)** : abandonner `weapon_kills_v3` (inférence fire events, gain jugé mineur vs v2).
> La vraie v3 = **reconstruire l'état ECS depuis le flux de réplication du film** → arme **tous-kills** + stats riches
> (health/shield/ammo/position/score) par joueur dans le temps.
> Outillage : **ghidra-mcp** (Ghidra en direct, cf. [[reference_ghidra_mcp_setup]]). Méthode : RE **statique** + **vérité-terrain**
> (match connu, known-plaintext). 100 % statique sur le `.exe` (pas de live-attach anti-triche).

## Architecture en couches

- **L1 — BitReader bit-exact** : primitif read-N-bits **MSB-first** + refill 8 octets / byte-swap big-endian.
  État du BitReader (params observés) : fin buffer `+0x10`, compteurs `+0x28/+0x2c`, accumulateur 64-bit `+0x30`,
  position `+0x38`, curseur `+0x40`. Réf lecteur : `FUN_140c18a1c`.
- **L2 — Codecs de valeur** : (a) FRAME/full-state = **int signé largeur-variable, préfixe 2-bit** (`FUN_140c18a1c` :
  `sel=2 bits ; w=8<<sel ∈ {8,16,32,64} ; v=read w bits MSB-first ; sign-extend si w<32`) ; (b) **1-bit** (`FUN_1406cf008`) ;
  (c) keyframe TYPE_2 = **varint continuation**.
- **L3 — Framing** : boucle de records `[type-id][handle][champs bit-packés]`. Dispatch = **tableau de triplets**
  `{descripteur_RVA, deser_RVA, fn2_RVA}` en **pointeurs image-base 32-bit (ibo32), stride 12 octets**.
  Ex. statborg : descripteur `0x143ea1a8c`, deser `FUN_140c18794`. **Pas d'interpréteur de schéma unique** : deser **bespoke** par composant (~264).
- **L4 — Désérialiseurs par composant** : RE un par un ceux qu'on veut. statborg/score = `FUN_140c18794`
  (record 2-équipes `[5b A][5b B][val A][val B][1b A][1b B]…`) ; 'obje'/arme ; player-state/health-shield.
- **L5 — Application** : recopie vers l'état monde (`FUN_140807ebc` → `world+team*0x1DF0+0x38`) → reconstruction → time-series.

## Priorité produit (user, 2026-06-05)

**1. Arme + kill feed** (objectif premier) → **2. Stats supplémentaires** (health/shield/…) → **3. Positions/mouvements** (dernier).
Le **score (statborg)** n'est PAS un objectif produit : c'est l'**oracle de validation** le moins cher (déjà implémenté + scores connus). Méthode : valider le **framing entité** avec le score, puis l'appliquer **immédiatement à l'arme** (composant kill-feed `'obje'`). Les positions (déquantizer livré) passent en dernier.

## Milestones

- **M1 — L1+L2+L3 : spec bit-exacte + impl Go + sanity** : RE complet BitReader + codecs + framing ; impl Go ;
  valider sur le **score gagnant** (déjà connu : TYPE_2 `token+24` ×3.86).
- **M2 — score perdant** : décoder la 2ᵉ équipe via le framing 2-équipes (`FUN_140c18794`). C'est **LE test** qui prouve
  le framing (l'échec offline antérieur = ne savait pas reconstruire les 2 équipes).
- **M3 — arme tous-kills** : deser du composant 'obje' / du record de kill → tag d'arme par kill, **tous joueurs**.
- **M4 — stats riches** : player-state (health/shield/ammo/position) → time-series par joueur.

## Validation (vérité-terrain)

**Anchor « Frag Parfait » (user, 2026-06-05) — l'oracle premium pour arme + santé** : sur un **match Slayer classique**
(pool d'armes réduit → parser prod = cross-check fiable), chercher un kill ayant rapporté la médaille **Perfect Kill /
Frag Parfait**. On en déduit : **arme = Sidekick** (la Perfect au Sidekick = 3 tirs corps + 1 tête), et un **profil de
santé connu : 225 → 0** (220+5 boucliers). Donc au timestamp de ce kill, le décodeur doit montrer (a) l'arme-de-kill =
Sidekick, (b) la santé de la victime chutant 225 → 0 selon le pattern. Valide **l'arme (prio 1) ET health/shield (prio 2)**
en un seul événement vérifiable. Données déjà dispo : timings, killer/victim (DB + parser prod). Match cible : `000d5950`
(Slayer, 28 chunks cachés).

Score (statborg) = oracle secondaire (`7344d24f` gagnant connu).

## Statut

- **M1 — L1/L2 LIVRÉS + VALIDÉS** : package Go `apps/go-api/internal/analysis/filmdec/` — `BitReader` MSB-first big-endian + `ReadSignedVarWidth` (sélecteur 2-bit, w=8<<sel) + `ParseStatborgRecord` + `ParseOptionalValue`. **8 vecteurs de test bit-exacts PASS** + `go vet` OK. Insight : la machine à états du moteur = **bitstream big-endian MSB-first simple** (impl robuste, pas besoin de répliquer accumulateur/refill). Pièges verrouillés : masquage `& ((1<<n)-1)` (bug toujours-refill du moteur), sign-extend **seulement** w=8/16, w=32 = int32 brut, w=64 = tronqué low-32. (Le vecteur V6 du workflow avait une erreur de packing octet3 `0xFC`→`0xC0`, corrigée ; le code décode fidèlement.) RE source : `.ai/V7.5/dumps/m1_funcs.c` ; spec : workflow `film-decoder-m1-spec` (6 agents).
- **M1 — RESTE (= framing, OUVERT)** : (1) **délimitage des records + énumération des bindings** (idxA,idxB) → RE le **caller de `FUN_140c18794`** (`get_function_callers` MCP, chercher la boucle sur une table de bindings) ; (2) **routage type-id** entre désérialiseurs ; (3) **codec keyframe TYPE_2** + l'échelle `×3.86`/offset `+24` du score (= l'oracle de validation bout-en-bout sur le gagnant connu).
- **Acquis RE** : dispatch cracké (triplets ibo32 {descripteur, deser, fn2}, stride 12). Le « schéma embarqué » était une sur-interprétation (`0x1404aae98` = itérateur générique). Deser bespoke par composant confirmé. BitReader struct mappé (+0x10 fin, **+0x18 longueur**, **+0x24 overrun**, +0x28/+0x2c compteurs, +0x30 acc, +0x38 used, +0x40 cursor).

### M2 — progrès (2026-06-05, cache empirique + agent statique)

- **Packet framing RÉSOLU** : chunk → paquets `[Type u16][b2][b3][Size u32][Timestamp u64]`. type-1 full state (343KB) → type-2 keyframe → metadata (6/8/12) → boucle `[type-10][type-0 FRAME]`. Sonde `cmd/tmp_filmprobe` (throwaway).
- **FRAME type-0 36o = positions** (marker `a0 7b 42` + 2 blocs quasi-constants). Score = keyframe type-2 (varint) ou FRAME score-change.
- **Record d'entité framing = `FUN_14080c1f8`** (param_4 = BitReader) : spawn bit → `variant_name` → header interne `FUN_14080cc68` (compteurs nb-stats `+0x34`, nb-bindings `+0xf8`) → boucle stat-entries `[2b][1b][32b]` → boucle bindings `[4b idx][1b présence][vecteur quantifié via `FUN_140c1e924`]` → orientation/vélocité. **C'est le décodeur de POSITION** (gros, ~25 sous-fns). Le statborg (score) reste sur le dispatch séparé `FUN_140c18794`.
- **Dispatch tables** : Table A `0x145348000` (statborg @0x145435cd4), Table B `0x1453f0000` (apply/entity builder @0x1453fd1d8/0x1453fd530). Routage `entry = base + id*12 ; deser = imagebase + entry[+4]`.
- **Boucle top-level + constructeur BitReader = derrière la vtable du monde** `PTR_FUN_1436b0fb0`. NON épinglée en statique (dispatch indirect calculé, zéro caller statique ; les 3 1ères entrées `0x142ba43b0`/`0x1410a5918`/`0x1405f0ac0` = destructeurs/stub Wwise). Debugger live décliné (anti-triche).
- **CONTOURNEMENT (clé, 2026-06-05)** : la boucle top-level n'est **pas nécessaire** pour décoder UN record si on sait où il commence. On réplique son rôle : `filmdec.BitReader` (validé) + grammaire `FUN_14080c1f8`.

### Session AUTONOME (2026-06-05) — décodeur d'entité construit + validé ; mur framing confirmé au niveau DATA

- **LIVRÉ** : `filmdec/entity.go` = `DecodeEntityRecord(br, fullState)` (grammaire complète `FUN_14080c1f8`, 26 étapes, par workflow `entity-record-decoder-synth` + audit bit-exactness) + `BitReader.Skip`. Compile, 10 tests filmdec PASS, go vet OK. Sous-décodeurs profonds stubés (`FUN_1431a0cbc`/`FUN_140c9e738`, à câbler).
- **VALIDATION** (sonde `cmd/tmp_entityprobe` sur chunk_02 type-1) : le décodeur **se synchronise sur 4000 records sans crash** (grammaire cohérente). MAIS variants **ronds/périodiques**, pas de string-ids aléatoires.
- **CAUSE (importante)** : le payload type-1 (343KB) est **sparse** = table structurée régulière en tête (`14 04 00 00 00 08…`, valeurs en doublement = quantif/index ?) + zone d'indices + **majoritairement ZÉROS**. **Ce n'est PAS une séquence plate de records.** ⇒ **le mur du framing/dispatch est confirmé au niveau DATA** : les records sont indexés/dispatchés, pas concaténés. Le décodeur est un **outil validé** mais le **framing du type-1** reste à cracker.
- **Pistes pour la suite** : (1) RE la table d'en-tête `14 04…` (index → offsets des records ?) ; (2) tester le **keyframe type-2** (dense, 142KB ; mais codec varint ≠ fixed-width — vérifier param_5) ; (3) reprendre la boucle top-level / le processeur de paquet type-1. **Hypothèse** : type-1 = registre/allocation (sparse), entités denses dans type-2 ; OU records dans les premiers ~60KB du type-1 après l'index.
- **Déquantizer position LIVRÉ + testé** (`filmdec/quantize.go`) : `ReadQuantizedVec3(bits, range)` = lit 3 composantes × N bits puis déquantifie `valeur = min + step·(q+0.5)` (RE de `FUN_140c1e9d4` lecture + `FUN_140c1e978` math + table `DAT_143b8c6f0` + centrage `DAT_143cd84b0=0.5`). Ranges : ±3 (dir/vél), ±0.7 (normalisé), **±100 (position monde)**. 10 vecteurs de test PASS.
- **Réalité scope** : décodeur d'entité complet = chantier multi-sessions (~25 sous-fns). **`filmdec` = BitReader + statborg + déquantizer** (les briques mathématiques sont là et testées). **Reste pour positions end-to-end** : le **framing entité** (`FUN_14080c1f8`) pour localiser le vec3 position dans les bits du FRAME + une vérité-terrain positions pour valider. Fondation posée, briques cartographiées.
- **Réf** : [[project-weapon-attribution-v3-status]], `RE_EXE_GHIDRA_FINDINGS.md` (§2 wire, §3 framing).

### Session (2026-06-05) — décodeur QUANTIFIÉ param_5=1 LIVRÉ + VALIDÉ sur vraie entité keyframe

- **LIVRÉ** : `filmdec/entity_quant.go` = `DecodeEntityRecordQ(br, statWidth)` (param_5=1 : stat loop
  quantifié `FUN_1406d3140` = probe+W+2 bits ; position `FUN_140c9e4d8` magW=14/scaleW=20 ; step19
  R(30) skip si cVar2≠0 ; step25 skip) + `readQuantStat`/`decodeC9E4D8`/`decodeC9E990`/`decodeC9E738`.
  Compile, vet, 10 tests filmdec PASS.
- **VALIDÉ (clé)** : sonde `cmd/tmp_keyframe_sweep` sur chunk_02 type-2 (Slayer 000d5950). Keyframe
  **DENSE** (18 % zéros). Ancre `0x67abd42a` au **bit 168 pile** (prédiction workflow). Record-ancre
  décodé bit-exact (148→268, Valid, PosValid, compteurs sains, 0 stub touché). **Grammaire param_5=1
  prouvée sur une VRAIE entité.**
- **MUR = framing keyframe** : pas un flux plat (tuilages bit0/bit148 convergent au bit600 = auto-sync
  stateless). Records délimités par per-packet header + component loop `FUN_14076cb60` + presence-bitmask
  `FUN_1406d7610` ; `componentCount` = registre runtime `*(ctx+0x4320)`. **Prochain verrou** (Ghidra
  statique) : init registre composant/entity-type → énumérer les entités → composant `'obje'` = ARME.
