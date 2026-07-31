# PLAN — Session CE breakpoints : valider les largeurs bit-exactes du record NEW biped

> Objectif : capturer la position bit (`reader+0x2c`) à chaque étape d'un record NEW biped réel,
> pour obtenir les largeurs EXACTES (representation, gate/masque, i0…) et valider le décodeur offline.
> Une seule session. Le film importe peu (grammaire = constante de build) ; il faut juste que ça DÉCODE
> (film qui tourne en Théâtre → records NEW biped aux respawns / au seek).

## Préconditions (à faire par l'utilisateur)
1. Lancer Halo Infinite, **Théâtre**, ouvrir un film (le plus récent JGTm), **le mettre en LECTURE**.
2. Cheat Engine ouvert, **attaché à HaloInfinite.exe**, bridge MCP rechargé (`ce_mcp_bridge.lua`).
3. Me dire « prêt ». (L'ASLR aura changé les adresses → je recalcule tout depuis la base.)

## Rappels d'adresses (statiques Ghidra ; je reloc au runtime = `static - 0x140000000 + base`)
- `FUN_1408f1aa4` (0x1408f1aa4) : deser record NEW. reader = **param_4 = R9**. Lit R(6) archétype en tête.
- `FUN_140f44c38` (0x140f44c38) : default-state biped (representation). reader = **param_4 = R9**. Ne fire QUE pour biped.
- `FUN_1406cfe44` (0x1406cfe44) : deser i0 position. reader = **param_2 = RDX**. Fire pour toute position.
- BitReader : `reader+0x2c` = compteur de bits consommés (croissant). `reader+0x28` = compteur octets.

## Ce qu'on capture (séquence d'un record NEW biped)
| Point | Breakpoint | reader | Donne |
|---|---|---|---|
| A | entrée `FUN_140f44c38` | R9 | bitpos APRÈS R(6), AVANT representation |
| B | entrée `FUN_1406cfe44` (1re après A) | RDX | bitpos du DÉBUT de i0 position |
| C | (optionnel) desers i1-i4 | RDX | bitpos de chaque composant suivant |

Déductions :
- **longueur representation = B − A − (bits gate+masque)**. Comme le masque de présence est lu ENTRE la
  representation et le 1er composant, B−A = representation + gate(1) + masque. On isole en posant aussi un BP
  sur `FUN_1406d7610` (lecteur de masque) : bitpos masque − A = longueur representation exacte.
- Confirme si **i0 est lu inconditionnellement** (default-mask {i0}) : i0 doit apparaître même quand son bit
  n'est pas dans le masque sparse.

## Script CE Lua (brouillon — à coller dans CE « Lua Engine », on affinera live)
```lua
local base = getAddress("HaloInfinite.exe")
local function R(s) return s - 0x140000000 + base end
CAP = {}                       -- log global
local function pos(reader) return reader and readInteger(reader + 0x2c) or -1 end

-- BP1 : representation biped (reader=R9)
debug_setBreakpoint(R(0x140f44c38))
-- BP2 : i0 position (reader=RDX)
debug_setBreakpoint(R(0x1406cfe44))
-- BP3 : lecteur de masque de présence (reader = 2e arg de FUN_1406d7610 = RDX)
debug_setBreakpoint(R(0x1406d7610))

function debugger_onBreakpoint()
  local rip = RIP
  if rip == R(0x140f44c38) then
    CAP[#CAP+1] = string.format("A REP  bitpos=%d reader=%X", pos(R9), R9)
  elseif rip == R(0x1406d7610) then
    CAP[#CAP+1] = string.format("M MASK bitpos=%d reader=%X", pos(RDX), RDX)
  elseif rip == R(0x1406cfe44) then
    CAP[#CAP+1] = string.format("B POS0 bitpos=%d reader=%X", pos(RDX), RDX)
  end
  if #CAP >= 30 then debug_removeBreakpoint(R(0x140f44c38)); debug_removeBreakpoint(R(0x1406cfe44)); debug_removeBreakpoint(R(0x1406d7610)) end
  return 1 -- continue
end
-- Après quelques secondes de lecture : print(table.concat(CAP, "\n"))
```
Note : `debugger_onBreakpoint` global + `RIP/R9/RDX` sont l'API de contexte de breakpoint de CE. Si le
symbole `R9`/`RDX` n'est pas exposé, on lit via `getThreadContext`/`debug_getContext`. On ajustera au 1er run.

## Ce que je fais de la sortie
- Séquence `A(rep) → M(mask) → B(pos0)` sur un même record : `M.bitpos − A.bitpos` = **longueur representation
  exacte** → je corrige `consumeBipedDefaultState`. `B.bitpos − M.bitpos` = gate+masque → je valide.
- Puis on étend aux desers ambigus (i1-i4, angular i3 : poser un BP sur `FUN_140d70998`, lire bitpos avant/après
  → largeur exacte, résout le 8-vs-10 magnitude).
- Avec la representation + le default-mask calés : le keyframe biped traverse → binding → 8 trajectoires.

## Largeurs déjà résolues statiquement (pas besoin de CE)
- i0 position = FUN_1406cfe44 (déjà porté ✓). i4 vitalité = R(8)+3×R(1). Table des 64 desers = recette §6.
- Beaucoup de largeurs sont des **constantes littérales dans l'appelant** (ex i3 : FUN_140d70998 passe (8, 0x13)) —
  résolubles en lisant l'assembleur de l'appelant. CE sert surtout à VALIDER la representation + l'alignement global.
