--[==[==========================================================================
  filmdec_pos_capture.lua
  LevelUp / Halo Infinite -- ORACLE de POSITION (validation decodeur offline)

  BUT
    Capturer la VERITE-TERRAIN des positions i0 des bipeds pendant le replay
    Theater : pour chaque record biped ou le composant i0 (object-position) est
    deserialise, on lit la position ABSOLUE que le jeu (re)construit dans l'objet
    entite. C'est l'oracle qui manque pour valider (falsifier) le decodeur offline
    des trajectoires : sans XYZ de reference, tout decode est infalsifiable.

  CE QU'ON A PROUVE (decompile FUN_1406cfe44 = deser i0)
    i0 = position, avec 2 bits de controle :
      (0,0) ABSOLU  -> consumeAbsoluteWithGate (FUN_14076e524)
      (0,1) DELTA   -> position precedente (*param_4) + delta
      (1,.) REUSE
    La position reconstruite ABSOLUE est ecrite dans l'objet entite :
      lVar3 = *(param_3 + 0x10)          (param_3 = 3e arg du deser = r8 au dispatch)
      [lVar3 + 0x04] = x (float)
      [lVar3 + 0x08] = y (float)
      [lVar3 + 0x0C] = z (float)
    On lit ces 3 floats au SITE DE DISPATCH (avant le call), donc la valeur lue =
    resultat de la PRECEDENTE deser i0 de cette entite (lag d'1 record par entite,
    trivialement recale cote Go par tri (slot, curseur)).

  HOOK (identique au site prouve par filmdec_delta_capture, AOB inchange)
      14076cd11  mov [rsp+20], r13d   \ 8 octets voles
      14076cd16  mov rcx, rbx         /
      14076cd19  call [rax+28]        <- dispatch du deser du composant
    Registres vivants au dispatch :
      rsi = record     -> [rsi+30]=typeIndex (35=biped), [rsi+34]=eid (= full id)
      rdi = bitreader  -> [rdi+2C]=curseur (bits consommes)
      r15d = compIndex (0 = i0 = position)
      r8  = param_3 du deser -> obj = [r8+0x10], position a obj+4/+8/+C
    Le cave filtre EN ASM typeIndex==35 ET compIndex==0 (volume minimal), garde
    null sur obj (anti-crash), et empile 24 o : eid, curseur, x, y, z, pad.

  eid -> slot : slot = eid & 0x3FFFFFFF (full id = (gen<<30)|slot). Bipeds = 512..519.

  USAGE (compte OFFLINE sans anti-cheat, comme convenu)
    1. Theater -> film 000d5950 (le seul avec oracle world_dump slot:ti).
    2. CE attache a HaloInfinite.exe ; charger ce script (dofile ou coller+Execute).
    3. captureFilmdecPos(20000)  puis JOUE / SCRUBE le film (deltas, pas la pause).
       -> dump auto -> .ai/re_dump/ce_pos_oracle.csv
    Manuel : startFilmdecPos / filmdecPosStatus / stopFilmdecPos / dumpFilmdecPos
    Patch residuel d'une session perdue : repairFilmdecPos()
==========================================================================]==]

local AOB         = "44 89 6C 24 20 48 8B CB FF 50 28"
local MODULE      = "HaloInfinite.exe"
local STOLEN_LEN  = 8
local REC_SIZE    = 24
local MAX_RECORDS = 0x200000  -- 2097152 records (x24 = 48 Mo)

fdp_inj  = fdp_inj  or nil
fdp_orig = fdp_orig or nil
fdp_buf  = fdp_buf  or nil
fdp_cnt  = fdp_cnt  or nil
fdp_tot  = fdp_tot  or nil
fdp_cave = fdp_cave or nil

local function moduleRange()
  local base = getAddress(MODULE)
  local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

local function findUniqueInModule(pattern)
  pattern = pattern or AOB
  local base, size = moduleRange()
  if not base then print("[FDP] module "..MODULE.." introuvable (process attache ?)"); return nil end
  local ms = AOBScan(pattern)
  if ms == nil or ms.Count == 0 then
    print("[FDP] AOB introuvable. Si une capture precedente a ete perdue, le code est")
    print("[FDP] peut-etre encore PATCHE -> tape repairFilmdecPos()  (puis recommence).")
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
    print(string.format("[FDP] attendu 1 occurrence dans le module, trouve %d -- abandon", n))
    return nil
  end
  return hit
end

function startFilmdecPos()
  if fdp_inj then stopFilmdecPos() end
  local inj = findUniqueInModule()
  if not inj then return end
  fdp_inj  = inj
  fdp_orig = readBytes(inj, STOLEN_LEN, true)

  fdp_cnt  = allocateMemory(0x40, inj)
  fdp_tot  = allocateMemory(0x40, inj)
  fdp_cave = allocateMemory(0x400, inj)
  fdp_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (fdp_cnt and fdp_tot and fdp_cave and fdp_buf) then
    print("[FDP] echec allocation memoire"); fdp_inj = nil; return
  end
  writeQword(fdp_cnt, 0)
  writeQword(fdp_tot, 0)

  for _, s in ipairs({"fdpBuf","fdpCnt","fdpTot","fdpCave","fdpInj"}) do unregisterSymbol(s) end
  registerSymbol("fdpBuf",  fdp_buf,  true)
  registerSymbol("fdpCnt",  fdp_cnt,  true)
  registerSymbol("fdpTot",  fdp_tot,  true)
  registerSymbol("fdpCave", fdp_cave, true)
  registerSymbol("fdpInj",  fdp_inj,  true)

  local asm = string.format([[
fdpCave:
  inc qword ptr [fdpTot]
  push rax
  push rbx
  push rcx
  push rdx
  mov eax,[rsi+30]              // typeIndex
  cmp eax,23                    // 0x23 = 35 (biped)
  jne fdp_pop
  test r15d,r15d               // compIndex == 0 (i0 position) ?
  jnz fdp_pop
  mov rax,[fdpCnt]
  cmp rax,%X                    // MAX_RECORDS
  jae fdp_pop
  mov rcx,[r8+10]              // obj = *(param_3 + 0x10)
  test rcx,rcx
  jz fdp_pop                    // garde null anti-crash
  imul rdx,rax,%X              // * REC_SIZE
  mov rbx,fdpBuf
  add rbx,rdx                   // rbx = dest record
  mov edx,[rsi+34]            // [+00] eid (= full id, slot = eid & 0x3FFFFFFF)
  mov [rbx+00],edx
  mov edx,[rdi+2C]           // [+04] curseur (bits consommes)
  mov [rbx+04],edx
  mov edx,[rcx+04]           // [+08] x (float bits)
  mov [rbx+08],edx
  mov edx,[rcx+08]           // [+0C] y
  mov [rbx+0C],edx
  mov edx,[rcx+0C]           // [+10] z
  mov [rbx+10],edx
  mov dword ptr [rbx+14],0    // [+14] pad
  inc qword ptr [fdpCnt]
fdp_pop:
  pop rdx
  pop rcx
  pop rbx
  pop rax
fdp_re:
  mov [rsp+20],r13d            // octet vole 1
  mov rcx,rbx                  // octet vole 2 (rbx restaure par pop ci-dessus)
  jmp fdpInj+%X

fdpInj:
  jmp fdpCave
  nop
  nop
  nop
]], MAX_RECORDS, REC_SIZE, STOLEN_LEN)

  if autoAssemble(asm) then
    print(string.format("[FDP] capture ON  @ %X  (buffer %d records)", inj, MAX_RECORDS))
    print("[FDP] Theater: avance / LIS quelques secondes (deltas, pas en pause).")
  else
    print("[FDP] echec autoAssemble -- restauration")
    if fdp_orig then writeBytes(inj, fdp_orig) end
    fdp_inj = nil
  end
end

function stopFilmdecPos()
  if not fdp_inj then print("[FDP] pas actif"); return end
  if fdp_orig then writeBytes(fdp_inj, fdp_orig) end
  local n = fdp_cnt and readQword(fdp_cnt) or 0
  print(string.format("[FDP] capture OFF -- %d records (dumpFilmdecPos pour exporter)", n))
  fdp_inj = nil
end

function repairFilmdecPos()
  local inj = findUniqueInModule("E9 ?? ?? ?? ?? 90 90 90 FF 50 28")
  if not inj then
    print("[FDP] aucun patch residuel unique trouve. Si l'AOB original est AUSSI introuvable,")
    print("[FDP] redemarre Halo Infinite (code propre).")
    return
  end
  writeBytes(inj, { 0x44, 0x89, 0x6C, 0x24, 0x20, 0x48, 0x8B, 0xCB })
  fdp_inj = nil
  print(string.format("[FDP] patch residuel restaure @ %X. Relance: captureFilmdecPos(20000)", inj))
end

local function defaultDumpPath()
  return "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv"
end

function filmdecPosStatus()
  local cnt = fdp_cnt and readQword(fdp_cnt) or nil
  local tot = fdp_tot and readQword(fdp_tot) or nil
  print(string.format("[FDP] inj=%s buf=%s  totalHits=%s  recordsEcrits=%s",
    fdp_inj and string.format("%X", fdp_inj) or "nil",
    fdp_buf and string.format("%X", fdp_buf) or "nil",
    tot and tostring(tot) or "nil",
    cnt and tostring(cnt) or "nil"))
  print("[FDP] totalHits=0 -> hook non execute (film en pause / mauvais chemin).")
  print("[FDP] recordsEcrits=0 mais totalHits>0 -> aucun biped i0 decode (avance dans le match).")
end

function dumpFilmdecPos(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\", "/") end
  local cnt = (fdp_buf and fdp_cnt) and readQword(fdp_cnt) or 0
  local tot = fdp_tot and readQword(fdp_tot) or 0
  if cnt and cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  local out = {
    string.format("# filmdec_pos totalHits=%s recordsEcrits=%s", tostring(tot), tostring(cnt)),
    "eid,slot,bitCursor,x,y,z",
  }
  for i = 0, (cnt or 0) - 1 do
    local b = fdp_buf + i * REC_SIZE
    local eid = readInteger(b + 0) & 0xFFFFFFFF
    local slot = eid & 0x3FFFFFFF
    out[#out + 1] = string.format("%d,%d,%d,%.4f,%.4f,%.4f",
      eid, slot, readInteger(b + 4),
      readFloat(b + 8), readFloat(b + 12), readFloat(b + 16))
  end
  local f, err = io.open(path, "w")
  if not f then
    print("[FDP] ouverture IMPOSSIBLE: " .. tostring(err) .. "  (chemin=" .. path .. ")")
    return
  end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[FDP] %d lignes de donnees ecrites -> %s", cnt or 0, path))
  if (cnt or 0) == 0 then
    print("[FDP] 0 record : le CHEMIN marche (fichier cree) mais rien capture ->")
    print("[FDP] verifie filmdecPosStatus() apres avoir LU des deltas (pas en pause).")
  end
end

function captureFilmdecPos(target, path)
  target = target or 20000
  path = path or defaultDumpPath()
  startFilmdecPos()
  if not fdp_inj then print("[FDP] demarrage echoue -> repairFilmdecPos() puis reessaie"); return end
  print(string.format("[FDP] >>> JOUE / SCRUBe le film MAINTENANT (j'attends %d records) <<<", target))
  local ticks = 0
  local t = createTimer(nil)
  t.Interval = 1000
  t.OnTimer = function(timer)
    ticks = ticks + 1
    local cnt = fdp_cnt and readQword(fdp_cnt) or 0
    print(string.format("[FDP] ... %d records (t=%ds)", cnt, ticks))
    if cnt >= target or ticks >= 90 then
      timer.destroy()
      stopFilmdecPos()
      dumpFilmdecPos(path)
      if cnt == 0 then
        print("[FDP] 0 record en 90s -> film non joue, ou hook au mauvais endroit.")
      else
        print("[FDP] termine.")
      end
    end
  end
end

print("[FDP] charge. Commande (tout-en-un) : captureFilmdecPos(20000)  puis JOUE le film.")
print("[FDP] Manuel : startFilmdecPos / filmdecPosStatus / stopFilmdecPos / dumpFilmdecPos")
print("[FDP] Patch residuel : repairFilmdecPos()")
