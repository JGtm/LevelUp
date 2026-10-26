# Table des default-states d'archétype (keyframe decoder) — vtable[0x60] par typeIndex

> Résolu statiquement (agents Ghidra, 2026-07-03) via `DAT_144e61d88[ti] → descripteur → vtable → +0x60`.
> Adresses STATIQUES (image_base 0x140000000). Le default-state d'un record NEW est lu par ce slot :
> `STUB` = `0x1408d8220` (`MOV AL,1 ; RET`, **0 bit** — le `Skip(0)` du port est déjà correct) ;
> `REAL` = une vraie fonction (largeur à décompiler, souvent courte/fixe).
>
> Contexte : cf HANDOFF_KEYFRAME_LIVE_CAPTURE « KEYFRAME DÉCODE ». Le keyframe = record-loop plat tout-NEW
> à slots croissants ; chaque record NEW = `header(type+id) + R(6) ti + default-state(vtable[0x60]) + gate R(1)
> + masque + composants + TAIL R(1)`. Pour décoder un archétype, il faut sa largeur de default-state (ci-dessous)
> PUIS ses composants (consumeByName + largeurs quantAxisWidth(L)).

## Classification ti 4..49 (REAL=29, STUB=17)

| ti | vtable[0x60] | classe | ti | vtable[0x60] | classe |
|----|--------------|--------|----|--------------|--------|
| 4  | 0x1408d8220  | STUB | 27 | 0x1408d8220  | STUB |
| 5  | 0x140FED600  | REAL | 28 | 0x142EEA4D8  | REAL |
| 6  | 0x140FF7F44  | REAL | 29 | 0x14116F514  | REAL |
| 7  | 0x1408d8220  | STUB | 30 | 0x1408d8220  | STUB |
| 8  | 0x142F14688  | REAL | 31 | 0x1408d8220  | STUB |
| 9  | 0x1410D7540  | REAL | 32 | 0x1408d8220  | STUB |
| 10 | 0x141020244  | REAL | 33 | 0x1408d8220  | STUB |
| 11 | 0x14110D4D8  | REAL | 34 | 0x1408d8220  | STUB |
| 12 | 0x1410ED0E8  | REAL | 35 | 0x140F44C38  | REAL (biped, `consumeBipedDefaultState`) |
| 13 | 0x140CE55E8  | REAL | 36 | 0x1407F2224  | REAL |
| 14 | 0x140FED6F4  | REAL | 37 | 0x1407F105C  | REAL |
| 15 | 0x1408d8220  | STUB | 38 | 0x1408F0B48  | REAL |
| 16 | 0x1408d8220  | STUB | 39 | 0x1408F0B48  | REAL (partage ti=38) |
| 17 | 0x14101A0A4  | REAL | 40 | 0x1410A5A74  | REAL |
| 18 | 0x1408d8220  | STUB | 41 | 0x1408EFB58  | REAL |
| 19 | 0x1408d8220  | STUB | 42 | 0x1407F0C68  | REAL |
| 20 | 0x142EEA600  | REAL | 43 | 0x140FE7630  | REAL |
| 21 | 0x141133C24  | REAL | 44 | 0x142EEA020  | REAL |
| 22 | 0x1408d8220  | STUB (record0 oracle, confirmé) | 45 | 0x1408d8220  | STUB |
| 23 | 0x142EEA440  | REAL | 46 | 0x1408d8220  | STUB |
| 24 | 0x142EEA5B4  | REAL | 47 | 0x1410F44F8  | REAL |
| 25 | 0x1408d8220  | STUB | 48 | 0x142F14668  | REAL |
| 26 | 0x1408d8220  | STUB | 49 | 0x141FD39C0  | REAL |

**STUB (0 bit)** : ti = 4, 7, 15, 16, 18, 19, 22, 25, 26, 27, 30, 31, 32, 33, 34, 45, 46.
**Cluster 0x142EEA###** (REALs proches, sûrement même famille de deser) : ti = 20, 23, 24, 28, 44 (+ ti=3 = 0x142eea28c).
**Partagés** : ti=38 & 39 → 0x1408F0B48.

## RESTE À FAIRE

1. **Largeurs des REAL** : décompiler chaque vtable[0x60] REAL pour sa largeur de bits (beaucoup sont courts/fixes,
   ex ti=5→R(6), ti=14→R(5)). À peupler dans `defaultStateBitsByTI` (traverse.go) via `SetDefaultStateBitsForTI`.
2. **ti ≥ 50** (dont **ti=63** = record1 de l'oracle, bloquant actuel) : étendre la table (agents n'ont couvert que 4..49).
3. **Composants** : après la largeur de default-state, la boucle de composants (consumeByName + quantAxisWidth(L))
   doit être bit-exacte par archétype pour extraire i0 (position).

Validation : `cmd/tmp_kfdecode` (dérouleur record-par-record) — chaîne de NEW à slots croissants sans désync.
