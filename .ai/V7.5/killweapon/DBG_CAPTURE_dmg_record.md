# CAPTURE DEBUGGER — record de degat FUN_14080c1f8 (2026-06-07, film 000d5950 en Theater)
Capture live (Ghidra dbgeng via ghidra-mcp, go_blocking). RIP=0x14080C1F8 (entree fonction).

## Registres (x64 fastcall)
- RCX/param_1 = 0x7FF617C94DD8
- RDX/param_2 = 0x328 (= taille du record)
- R8 /param_3 = 0x23591C838A0  -> record SORTIE (tout a 0 = memset, on est a l'entree)
- R9 /param_4 = 0x236D08F3560  -> etat BITREADER film

## Pile d'appel (= la chaine de lecture a repliquer)
- L0 FUN_14080C1F8 (deser record degat)
- L1 caller = FUN_14080AADE  <== LE LECTEUR GENERIQUE (extrait le message + bitreader + appelle le deser)
- L2 FUN_14076A24E
- L3+ ManagedGameVariant_* (labels d'export approx, profond)

## Bitreader (R9 @0x236D08F3560), parse (LE)
- +0x00 = 0x236D0ACCA20  (objet contexte exterieur, region differente)
- +0x08 = 0x237D0340000  (BUFFER start)
- +0x10 = 0x237D03400D6  (BUFFER end)  => longueur 0xD6 = 214 octets
- +0x18 = 0xD6 (214)     (limite/longueur)
- +0x2c = 0x18 (24)      (bitpos)
- +0x30 = 0x0004384BD2000000 (registre 64-bit MSB-first)
- +0x38 = 0x18 (24)      (count bits)
- +0x40 = 0x237D0340008  (byteptr courant = start+8)

## BUFFER message (214 o) @0x237D0340000  (= 1 message d'event de degat)
d260440004384bd29ed42c9679f76036028265840c0023483e495a66340f6840c088012000aac828e93e89d2c2ed901d7a22776e0220146002c81cff12d99680097922dc0440690005564138c97c4d161062c0ebc553e27011022500155984108292acc38022988d784841270006c40a900055640dfa9d1cc3801e0ccbc8114267011032400155903f463313f2808ec139600af8dc0440e900055641374924500011d0310ece43d2701283a24a36111fffffffff4f7c0100...
NB: contient 2c9679f7 @offset ~0x0a = suffixe variant arme 0x42c9679f bit-packe (PREUVE: la source/arme est dans ce message).

## A FAIRE offline
1. Matcher ce buffer (ou un substring ex 9ed42c9679f7 / 04384bd2) contre les chunks INFLATES de 000d5950 -> packet-type + offset.
2. Decompiler FUN_14080AADE (lecteur generique) -> framing du flux d'events (longueur/type prefixe, init bitreader).
3. Repliquer en Go -> decoder tous les records event-type 11 -> source par kill.

## MATCH OFFLINE (tmp_bufmatch) — FRAMING CRACKÉ
Le buffer capturé (214o) se retrouve VERBATIM dans les chunks INFLATÉS, en **paquets type-0** :
- chunk_02 inflate: full32 (32o) @abs 0x9cbb2, **packet-type=0, payload_off=0** (le message = début du payload type-0).
- suffixe variant (9ed42c9679f7) @payload_off=8 dans de NOMBREUX paquets type-0 (chunk_02/04/06...) = nombreux records de dégât.
⇒ Les messages de dégât sont des **paquets type-0** (ou sous-blocs) — décodables offline. Contredit l'analyse statique "pas type-0".
PROCHAIN: décompiler FUN_14080AADE (lecteur) + mapper offsets champ; structure message[0:8]=header (d2604400 04384bd2), variant @~offset8.

---
# RUNBOOK — REFAIRE UNE CAPTURE DEBUGGER (secours, reproductible) — 2026-06-07
Setup ghidra-mcp debugger (pybag/dbgeng) qui a MARCHE. A refaire pour capter un autre event (ex: FUN_1407e00ac l'apply, qui a attaquant+victime en RAM).

## Pre-requis (une fois)
- Deps debugger dans le venv ISOLE ghidra-mcp : `C:\Users\Guillaume\Downloads\ghidra-mcp\.venv\Scripts\python.exe -m pip install -r requirements-debugger.txt` (pybag, comtypes, protobuf).
- Pas besoin d'installer WinDbg : System32 a dbgeng/dbghelp/dbgmodel/dbgcore. On pointe pybag dessus via `WINDBG_DIR=C:\Windows\System32`.
- PATCHS ADDITIFS au bridge (deja appliques, inoffensifs) :
  - `debugger/server.py` : endpoint POST `/debugger/go_blocking` -> `engine.go()` (BLOQUANT, WaitForEvent propre). INDISPENSABLE : le `/debugger/go` standard est non-bloquant (go_nowait, PAS de WaitForEvent) -> l'exception du bp n'est jamais traitee -> GEL + wedge.
  - `debugger/tracing.py` : `_on_entry` patche x64 (capture R9/registres + dump memoire -> logs/dmg_capture.txt). Optionnel (la capture bp manuelle suffit).

## Demarrer le serveur debugger (port 8099)
```
cd C:/Users/Guillaume/Downloads/ghidra-mcp && export WINDBG_DIR='C:\Windows\System32' && .venv/Scripts/python.exe -m debugger
```
(lancer en background ; verifier "Debugger server starting on 127.0.0.1:8099").

## Sequence de capture (curl direct sur 8099 — fiable)
1. Jeu : lancer Halo, aller dans Theater, selectionner le film, **NE PAS cliquer "Regarder le film"** encore. Recuperer le PID :
   `powershell "(Get-Process HaloInfinite).Id"`  (le PID CHANGE a chaque relance).
2. **Attach par PID** (PAS par nom — l'enum de process plante sur un nom non-UTF8) :
   `curl -s -X POST http://127.0.0.1:8099/debugger/attach -d '{"target":"<PID>"}'`  -> state:"stopped".
3. **Sync map** (HaloInfinite image_base Ghidra = 0x140000000) :
   `curl -s -X POST http://127.0.0.1:8099/debugger/sync_modules -d '{"ghidra_bases":{"HaloInfinite":"0x140000000"}}'` -> mapped:1.
4. **Poser le bp** (adresse Ghidra ; runtime = base 0x7FF6_xxxx + (addr-0x140000000)) :
   `curl -s -X POST http://127.0.0.1:8099/debugger/breakpoint -d '{"ghidra_address":"0x<ADDR>","module":"HaloInfinite"}'`.
5. **go_blocking en BACKGROUND** (relance le jeu + attend l'arret proprement ; le curl HANG jusqu'au hit) :
   `curl -s -X POST http://127.0.0.1:8099/debugger/go_blocking -d '{}'`  (run_in_background).
6. Dire a l'user : **clique "Regarder le film"** (la deserialization des events se fait au CHARGEMENT, pas en lecture).
7. Quand le background go_blocking RETOURNE (state:"stopped") = bp frappe. Lire :
   `curl -s http://127.0.0.1:8099/debugger/registers` ; `curl -s "http://127.0.0.1:8099/debugger/stack?depth=12"` ;
   `curl -s "http://127.0.0.1:8099/debugger/memory?address=0x<R9>&size=80&address_type=runtime"`.
   x64 fastcall : RCX=param_1, RDX=param_2, R8=param_3, R9=param_4. Pile niveau 1 = le caller (lecteur).
8. **Liberer le jeu** : `curl -s -X POST http://127.0.0.1:8099/debugger/detach`.

## Pieges (vecus)
- `interrupt` sur une cible EN COURS injecte un thread `DbgUiRemoteBreakin` (mauvais contexte) -> ne PAS l'utiliser pour capter un bp ; utiliser `go_blocking`.
- "never became queryable" a l'attach = le process a ferme (l'user a clique fermer ; la commande s'execute au detach). Relancer le jeu, re-attach.
- Le `status` ne repond pas pendant un go_blocking (worker bloque dans WaitForEvent) = NORMAL.
