--[==[==========================================================================
  filmdec_delta_capture.lua
  LevelUp / Halo Infinite -- calibration du decodeur FILM (kill feed RE, L3)

  BUT
    Capturer, pendant le replay Theater du film, les largeurs reelles (en bits)
    de chaque composant ECS d'un record DELTA biped. C'est la verite-terrain qui
    manque au .exe statique : la largeur de i0 (object-position predicted-delta)
    et la valeur runtime param_4 (recordStateParam) de i10/i19/i20/i23 sont
    populees au chargement de map et lues a 0 statiquement. On les lit ici en
    direct, puis on cale le decodeur Go -> spine delta bit-exact -> lecture du
    composant object-dead-state (i11) = ou le jeu enregistre la cause de mort.

  COMMENT CA MARCHE (aucune adresse en dur -> NON hardware-specific)
    On pose un code-cave sur le SEUL site de dispatch des desers de composants,
    dans l'iterateur FUN_14076cb60 :
        14076cd11  mov [rsp+20], r13d   \ 8 octets voles
        14076cd16  mov rcx, rbx         /
        14076cd19  call [rax+28]        <- dispatch du deser du composant
    A ce point, tous les registres utiles sont vivants :
        rdi = bitreader   -> [rdi+2C] = curseur (bits consommes depuis le debut)
        rsi = record      -> [rsi+30] = typeIndex (35 = biped) ; [rsi+34] = entity id
        r15d = index du composant dans l'archetype (ordre = registre)
        r13d = param_4 (recordStateParam calcule pour CE composant)
        r14d = compteur de skips (bit de mask = r15 - r14)
    Le cave filtre typeIndex==35 (biped) EN ASM (volume reduit) et empile par
    record 24 octets : eid, typeIndex, compIndex, param4, bitCursor, skipCount.
    On differencie ensuite (cote Go) les curseurs consecutifs d'un meme record
    pour obtenir la largeur exacte de chaque composant (i0..i11).

  PRE-REQUIS
    - Compte/instance OFFLINE sans anti-cheat (comme convenu).
    - Le build HaloInfinite.exe doit correspondre a l'AOB ci-dessous (meme build
      que l'import Ghidra). Si l'AOB n'est pas trouve / multiple -> le script
      l'indique et s'arrete (pas d'injection hasardeuse).

  USAGE
    1. Lancer Halo, aller dans Theater, ouvrir le film du match :
         match 000d5950  (Cliffhanger, Super Fiesta, 8 mars 2026).
       -> c'est imperatif : on cale sur les octets de CE film precisement.
    2. Cheat Engine -> attacher au process "HaloInfinite.exe".
    3. CE -> menu Table -> "Show Cheat Table Lua Script" (ou Ctrl+Alt+L),
       coller TOUT ce fichier, cliquer "Execute Script".
    4. Taper dans la console Lua :   startFilmdecCapture()
    5. Dans Theater : AVANCER au milieu du match puis LIRE quelques secondes
       (objectif : decoder des DELTAS, pas seulement la 1ere keyframe). Idealement
       lire autour d'un kill (pour capturer un dead-state i11 actif).
    6. Taper :   filmdecStatus()    (verifie cnt > 0 ; sinon relire des deltas)
    7. Taper :   stopFilmdecCapture()
    8. Taper :   dumpFilmdecCapture()    -> ecrit ~/Downloads/filmdec_delta.csv
       (ou un chemin explicite : dumpFilmdecCapture([[D:\un\dossier\sortie.csv]]))
    9. Donner le CSV a l'agent (LevelUp). Fin.

  Re-jouable : startFilmdecCapture() remet le compteur a 0 a chaque appel.
==========================================================================]==]

local AOB         = "44 89 6C 24 20 48 8B CB FF 50 28"  -- mov [rsp+20],r13d ; mov rcx,rbx ; call [rax+28]
local MODULE      = "HaloInfinite.exe"
local STOLEN_LEN  = 8        -- octets voles (les 2 mov), le call reste a +8
local REC_SIZE    = 24       -- octets par enregistrement
local MAX_RECORDS = 0x300000  -- 3145728 records (x24 = 72 Mo) : tient un film entier (1.14M hits mesurés)

-- etat global (persiste dans la console Lua de CE)
fdc_inj   = fdc_inj   or nil   -- adresse d'injection
fdc_orig  = fdc_orig  or nil   -- octets originaux (pour restaurer)
fdc_buf   = fdc_buf   or nil   -- buffer de capture
fdc_cnt   = fdc_cnt   or nil   -- compteur de records ecrits (qword)
fdc_tot   = fdc_tot   or nil   -- compteur de TOUS les hits du hook (diagnostic)
fdc_cave  = fdc_cave  or nil   -- code cave

local function moduleRange()
  local base = getAddress(MODULE)
  local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

-- Trouve l'unique occurrence d'un pattern AOB DANS le module (pas ailleurs en RAM).
local function findUniqueInModule(pattern)
  pattern = pattern or AOB
  local base, size = moduleRange()
  if not base then print("[FDC] module "..MODULE.." introuvable (process attache ?)"); return nil end
  local ms = AOBScan(pattern)  -- toutes regions ; on filtre ensuite par plage module
  if ms == nil or ms.Count == 0 then
    print("[FDC] AOB introuvable. Si une capture precedente a ete perdue, le code est")
    print("[FDC] peut-etre encore PATCHE -> tape repairFilmdecPatch()  (puis recommence).")
    print("[FDC] pattern="..pattern)
    if ms then ms.destroy() end
    return nil
  end
  local hit, n = nil, 0
  for i = 0, ms.Count - 1 do
    local a = tonumber(ms[i], 16)
    if a and a >= base and a < base + size then hit = a; n = n + 1 end
  end
  ms.destroy()
  if n ~= 1 then
    print(string.format("[FDC] attendu 1 occurrence dans le module, trouve %d -- abandon", n))
    return nil
  end
  return hit
end

function startFilmdecCapture()
  -- si deja actif, on stoppe proprement avant de relancer
  if fdc_inj then stopFilmdecCapture() end

  local inj = findUniqueInModule()
  if not inj then return end
  fdc_inj  = inj
  fdc_orig = readBytes(inj, STOLEN_LEN, true)  -- table d'octets originaux

  -- allocations : cnt + cave PRES de l'injection (portee rel32 des jmp) ;
  -- le buffer peut etre loin (accede par adresse absolue mov rbx,imm64).
  fdc_cnt  = allocateMemory(0x40, inj)
  fdc_tot  = allocateMemory(0x40, inj)
  fdc_cave = allocateMemory(0x400, inj)
  fdc_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (fdc_cnt and fdc_tot and fdc_cave and fdc_buf) then
    print("[FDC] echec allocation memoire"); fdc_inj = nil; return
  end
  writeQword(fdc_cnt, 0)
  writeQword(fdc_tot, 0)

  -- symboles pour l'auto-assembleur
  for _, s in ipairs({"fdcBuf","fdcCnt","fdcTot","fdcCave","fdcInj"}) do unregisterSymbol(s) end
  registerSymbol("fdcBuf",  fdc_buf,  true)
  registerSymbol("fdcCnt",  fdc_cnt,  true)
  registerSymbol("fdcTot",  fdc_tot,  true)
  registerSymbol("fdcCave", fdc_cave, true)
  registerSymbol("fdcInj",  fdc_inj,  true)

  -- DIAGNOSTIC : pas de filtre typeIndex (on capture TOUT, plafonne au buffer) +
  -- compteur total fdcTot incremente a CHAQUE hit (avant tout). Si fdcTot==0 apres
  -- lecture -> le hook ne s'execute pas (mauvais chemin de code / film non decode).
  -- Si fdcTot>0 -> la colonne typeIndex du CSV revele quelles archetypes existent.
  local asm = string.format([[
fdcCave:
  inc qword ptr [fdcTot]         // TOTAL hits (inconditionnel, avant tout)
  push rax
  push rbx
  push rdx
  mov rax,[fdcCnt]
  cmp rax,%X                     // MAX_RECORDS en HEX (CE lit les nombres en hex)
  jae fdc_pop
  imul rdx,rax,%X                // * REC_SIZE en HEX (0x18 = 24)
  mov rbx,fdcBuf
  add rbx,rdx
  mov edx,[rsi+34]              // [+00] entity id
  mov [rbx+00],edx
  mov edx,[rsi+30]             // [+04] typeIndex (toutes valeurs)
  mov [rbx+04],edx
  mov [rbx+08],r15d            // [+08] index du composant (ordre archetype)
  mov [rbx+0C],r13d            // [+0C] param_4 (recordStateParam de ce composant)
  mov edx,[rdi+2C]            // [+10] curseur (bits consommes)
  mov [rbx+10],edx
  mov [rbx+14],r14d            // [+14] compteur de skips
  inc qword ptr [fdcCnt]
fdc_pop:
  pop rdx
  pop rbx
  pop rax
fdc_re:
  mov [rsp+20],r13d            // octet vole 1 : mov [rsp+20],r13d
  mov rcx,rbx                  // octet vole 2 : mov rcx,rbx
  jmp fdcInj+%X               // retour sur : call [rax+28] (offset HEX)

fdcInj:
  jmp fdcCave
  nop
  nop
  nop
]], MAX_RECORDS, REC_SIZE, STOLEN_LEN)

  if autoAssemble(asm) then
    print(string.format("[FDC] capture ON  @ %X  (buffer %d records)", inj, MAX_RECORDS))
    print("[FDC] Theater: avance au milieu du match et LIS quelques secondes (deltas).")
  else
    print("[FDC] echec autoAssemble -- restauration")
    if fdc_orig then writeBytes(inj, fdc_orig) end
    fdc_inj = nil
  end
end

function stopFilmdecCapture()
  if not fdc_inj then print("[FDC] pas actif"); return end
  if fdc_orig then writeBytes(fdc_inj, fdc_orig) end  -- restaure les 8 octets
  local n = fdc_cnt and readQword(fdc_cnt) or 0
  print(string.format("[FDC] capture OFF -- %d records captures (dumpFilmdecCapture pour exporter)", n))
  fdc_inj = nil  -- buffer/compteur restent lisibles pour le dump
end

-- repairFilmdecPatch restaure le code si une injection precedente a ete perdue
-- (etat Lua reset avant stopFilmdecCapture). Le site reste alors patche :
--   E9 rel32 90 90 90 FF 50 28   (notre jmp + 3 nops + le call original intact).
-- On le retrouve et on remet les 8 octets voles d'origine. Pas besoin de redemarrer
-- le jeu. A lancer si startFilmdecCapture dit "AOB introuvable".
function repairFilmdecPatch()
  local inj = findUniqueInModule("E9 ?? ?? ?? ?? 90 90 90 FF 50 28")
  if not inj then
    print("[FDC] aucun patch residuel unique trouve.")
    print("[FDC] -> si l'AOB original est introuvable AUSSI, redemarre Halo Infinite (code propre).")
    return
  end
  writeBytes(inj, { 0x44, 0x89, 0x6C, 0x24, 0x20, 0x48, 0x8B, 0xCB })
  fdc_inj = nil
  print(string.format("[FDC] patch residuel restaure @ %X. Relance: captureFilmdec(10)", inj))
end

-- defaultDumpPath construit un chemin ABSOLU dans Downloads via USERPROFILE, en
-- slashes-avant (Windows + CE io.open les acceptent, et AUCUN echappement \ requis).
local function defaultDumpPath()
  local home = os.getenv("USERPROFILE") or os.getenv("HOMEPATH") or "C:/Users/Guillaume"
  home = home:gsub("\\", "/")
  return home .. "/Downloads/filmdec_delta.csv"
end

-- filmdecStatus imprime le compteur courant (a appeler AVANT le dump pour verifier
-- que la capture a bien tourne).
function filmdecStatus()
  local cnt = fdc_cnt and readQword(fdc_cnt) or nil
  local tot = fdc_tot and readQword(fdc_tot) or nil
  print(string.format("[FDC] inj=%s buf=%s  totalHits=%s  recordsEcrits=%s",
    fdc_inj and string.format("%X", fdc_inj) or "nil",
    fdc_buf and string.format("%X", fdc_buf) or "nil",
    tot and tostring(tot) or "nil",
    cnt and tostring(cnt) or "nil"))
  print("[FDC] totalHits=0 -> le hook ne s'execute pas (film non decode / mauvais chemin).")
  print("[FDC] totalHits>0 -> OK, le CSV contient toutes les archetypes (filtre biped cote Go).")
end

-- dumpFilmdecCapture ecrit le CSV. path optionnel -> defaut = Downloads/filmdec_delta.csv.
-- Ecrit TOUJOURS un fichier (au moins l'en-tete) pour prouver que le chemin fonctionne :
-- si le fichier sort avec 0 ligne de donnees, c'est la CAPTURE qui n'a rien pris
-- (injection/AOB), pas le chemin.
function dumpFilmdecCapture(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\", "/") end
  local cnt = (fdc_buf and fdc_cnt) and readQword(fdc_cnt) or 0
  local tot = fdc_tot and readQword(fdc_tot) or 0
  if cnt and cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  -- ligne de diagnostic (prefixe # -> ignoree par l'ingesteur Go) : visible meme si
  -- seul le fichier est transmis.
  local out = {
    string.format("# filmdec totalHits=%s recordsEcrits=%s", tostring(tot), tostring(cnt)),
    "eid,typeIndex,compIndex,param4,bitCursor,skipCount",
  }
  for i = 0, (cnt or 0) - 1 do
    local b = fdc_buf + i * REC_SIZE
    out[#out + 1] = string.format("%d,%d,%d,%d,%d,%d",
      readInteger(b + 0), readInteger(b + 4), readInteger(b + 8),
      readInteger(b + 12), readInteger(b + 16), readInteger(b + 20))
  end
  local f, err = io.open(path, "w")
  if not f then
    print("[FDC] ouverture IMPOSSIBLE: " .. tostring(err) .. "  (chemin=" .. path .. ")")
    print("[FDC] essaie un dossier ou tu peux ecrire, ex: dumpFilmdecCapture([[" .. defaultDumpPath() .. "]])")
    return
  end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[FDC] %d lignes de donnees ecrites -> %s", cnt or 0, path))
  if (cnt or 0) == 0 then
    print("[FDC] ATTENTION: 0 record capture. Le CHEMIN marche (fichier cree) mais la capture")
    print("[FDC] n'a rien pris -> verifie filmdecStatus() apres avoir LU des deltas (pas en pause).")
  end
end

-- captureFilmdec = TOUT-EN-UN (recommande). Injecte, te laisse jouer le film pendant
-- `seconds` s, puis arrete + dumpe AUTOMATIQUEMENT. Aucune etape manuelle entre la
-- capture et l'ecriture -> impossible de perdre l'etat Lua entre-temps.
--   Usage : captureFilmdec(10)   puis JOUE/SCRUBe le film immediatement.
function captureFilmdec(target, path)
  target = target or 5000        -- nb de records vises avant dump auto
  path = path or defaultDumpPath()
  startFilmdecCapture()
  if not fdc_inj then print("[FDC] demarrage echoue -> repairFilmdecPatch() puis reessaie"); return end
  print(string.format("[FDC] >>> JOUE / SCRUBe le film MAINTENANT (j'attends %d records) <<<", target))
  local ticks = 0
  local t = createTimer(nil)
  t.Interval = 1000
  t.OnTimer = function(timer)
    ticks = ticks + 1
    local cnt = fdc_cnt and readQword(fdc_cnt) or 0
    print(string.format("[FDC] ... %d records captures (t=%ds)", cnt, ticks))
    if cnt >= target or ticks >= 60 then
      timer.destroy()
      stopFilmdecCapture()
      dumpFilmdecCapture(path)
      if cnt == 0 then
        print("[FDC] 0 record en 60s -> le film ne jouait pas, ou hook au mauvais endroit.")
      else
        print("[FDC] termine. Donne le CSV a l'agent.")
      end
    end
  end
end

print("[FDC] charge. Commande RECOMMANDEE (tout-en-un) :")
print("[FDC]   captureFilmdec(10)   -- injecte, tu joues le film 10s, dump auto -> Downloads")
print("[FDC]   (ou chemin explicite : captureFilmdec(10, [[C:\\Users\\Guillaume\\Desktop\\filmdec_delta.csv]]) )")
print("[FDC] Manuel (si besoin) : startFilmdecCapture / filmdecStatus / stopFilmdecCapture / dumpFilmdecCapture")
print("[FDC] Si 'AOB introuvable' (patch residuel d'une session perdue) : repairFilmdecPatch()")
