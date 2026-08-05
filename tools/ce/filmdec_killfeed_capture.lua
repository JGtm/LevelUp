--[==[==========================================================================
  filmdec_killfeed_capture.lua
  LevelUp / Halo Infinite -- LECTURE de l'ARME du kill feed (a la source)

  L'arme affichee par le kill feed est lue par le constructeur d'entree
  FUN_14066b5e8 :
     14066b60c  CALL FUN_1404969f0      ; RAX = DamageReport
     14066b61c  MOV EBX,[RAX+0x538]     ; EBX = ID arme/icone du kill feed
     14066b62b  LEA RCX,[RSP+0x40]      ; <-- on hooke ICI
  A ce point : EBX = arme, RAX = DamageReport, RSI = entree kill feed (param_1).
  On capture EBX + une fenetre du report (autour de +0x538) + l'entree, pour
  retrouver tueur/victime par recoupement avec la narration de l'utilisateur.

  USAGE : Execute Script, captureKillfeed(), JOUE le film (autour des kills),
  dump auto -> Downloads/filmdec_killfeed.csv. repairKillfeedPatch() si besoin.
==========================================================================]==]

local MODULE      = "HaloInfinite.exe"
-- MOV EBX,[RAX+538] ; CMP EBX,-1 ; JZ rel32 ; LEA RCX,[RSP+40]
local AOB         = "8B 98 38 05 00 00 83 FB FF 0F 84 ?? ?? ?? ?? 48 8D 4C 24 40"
local CALL_OFF    = 0x0F       -- offset du LEA (point d'injection) dans l'AOB
local REP_OFF     = 0x500      -- debut de la fenetre report copiee
local REP_LEN     = 0x60       -- 0x60 octets de report (couvre +0x538 arme)
local ENT_LEN     = 0x40       -- 0x40 octets d'entree (param_1)
local REC_SIZE    = 0xB0       -- 4(arme)+REP_LEN+ENT_LEN, arrondi
local MAX_RECORDS = 0x8000

fkf_inj  = fkf_inj  or nil
fkf_orig = fkf_orig or nil
fkf_buf  = fkf_buf  or nil
fkf_cnt  = fkf_cnt  or nil
fkf_cave = fkf_cave or nil

local function moduleRange()
  local base = getAddress(MODULE); local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

local function findUnique(pattern)
  local base, size = moduleRange()
  if not base then print("[FKF] module introuvable"); return nil end
  local ms = AOBScan(pattern)
  if ms == nil or ms.Count == 0 then
    print("[FKF] AOB introuvable. Patch residuel ? -> repairKillfeedPatch(). pattern=" .. pattern)
    if ms then ms.destroy() end; return nil
  end
  local hit, n = nil, 0
  for i = 0, ms.Count - 1 do
    local a = tonumber(ms[i], 16)
    if a and a >= base and a < base + size then hit = a; n = n + 1 end
  end
  ms.destroy()
  if n ~= 1 then print(string.format("[FKF] attendu 1, trouve %d", n)); return nil end
  return hit
end

local function defaultDumpPath()
  local home = os.getenv("USERPROFILE") or "C:/Users/Guillaume"
  return (home:gsub("\\", "/")) .. "/Downloads/filmdec_killfeed.csv"
end

function repairKillfeedPatch()
  local start = findUnique("8B 98 38 05 00 00 83 FB FF 0F 84 ?? ?? ?? ?? E9 ?? ?? ?? ??")
  if not start then print("[FKF] aucun patch residuel. Sinon redemarre Halo."); return end
  local inj = start + CALL_OFF
  writeBytes(inj, { 0x48, 0x8D, 0x4C, 0x24, 0x40 })  -- restaure LEA RCX,[RSP+40]
  fkf_inj = nil
  print(string.format("[FKF] patch restaure @ %X. Relance captureKillfeed()", inj))
end

function startKillfeedCapture()
  if fkf_inj then stopKillfeedCapture() end
  local start = findUnique(AOB); if not start then return end
  local inj = start + CALL_OFF
  fkf_inj = inj
  fkf_orig = readBytes(inj, 5, true)
  fkf_cnt  = allocateMemory(0x40, inj)
  fkf_cave = allocateMemory(0x400, inj)
  fkf_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (fkf_cnt and fkf_cave and fkf_buf) then print("[FKF] echec alloc"); fkf_inj = nil; return end
  writeQword(fkf_cnt, 0)
  for _, s in ipairs({ "fkfBuf", "fkfCnt", "fkfCave", "fkfInj" }) do unregisterSymbol(s) end
  registerSymbol("fkfBuf", fkf_buf, true)
  registerSymbol("fkfCnt", fkf_cnt, true)
  registerSymbol("fkfCave", fkf_cave, true)
  registerSymbol("fkfInj", fkf_inj, true)

  -- EBX/RSI/RBP non-volatils ou requis -> NE PAS toucher. RAX=report. rcx/rdx/r8/r9 = scratch.
  local asm = string.format([[
fkfCave:
  mov r9,[fkfCnt]
  cmp r9,%X
  jae fkf_done
  imul rdx,r9,%X
  mov r8,fkfBuf
  add r8,rdx
  mov [r8+00],ebx                // [0] ID arme (=[report+0x538])
  xor rdx,rdx
fkf_cp1:
  mov ecx,[rax+rdx+%X]          // report[REP_OFF + rdx]
  mov [r8+rdx+04],ecx
  add rdx,4
  cmp rdx,%X                     // REP_LEN
  jb fkf_cp1
  xor rdx,rdx
fkf_cp2:
  mov ecx,[rsi+rdx]            // entree param_1[rdx]
  mov [r8+rdx+%X],ecx          // dst = 4 + REP_LEN
  add rdx,4
  cmp rdx,%X                     // ENT_LEN
  jb fkf_cp2
  inc qword ptr [fkfCnt]
fkf_done:
  lea rcx,[rsp+40]              // octet vole (5 o)
  jmp fkfInj+05                 // retour apres le LEA

fkfInj:
  jmp fkfCave
]], MAX_RECORDS, REC_SIZE, REP_OFF, REP_LEN, 4 + REP_LEN, ENT_LEN)

  if autoAssemble(asm) then print(string.format("[FKF] capture ON @ %X", inj))
  else print("[FKF] echec autoAssemble"); writeBytes(inj, fkf_orig); fkf_inj = nil end
end

function stopKillfeedCapture()
  if not fkf_inj then print("[FKF] pas actif"); return end
  writeBytes(fkf_inj, fkf_orig)
  print(string.format("[FKF] capture OFF -- %d entrees", fkf_cnt and readQword(fkf_cnt) or 0))
  fkf_inj = nil
end

function killfeedStatus()
  print(string.format("[FKF] inj=%s entrees=%s",
    fkf_inj and string.format("%X", fkf_inj) or "nil",
    fkf_cnt and tostring(readQword(fkf_cnt)) or "nil"))
end

function dumpKillfeed(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\", "/") end
  local cnt = (fkf_buf and fkf_cnt) and readQword(fkf_cnt) or 0
  if cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  local hdr = { "weapon" }
  for o = 0, REP_LEN - 1, 4 do hdr[#hdr + 1] = string.format("r%03X", REP_OFF + o) end
  for o = 0, ENT_LEN - 1, 4 do hdr[#hdr + 1] = string.format("e%02X", o) end
  local out = { string.format("# killfeed entrees=%s (weapon=[report+0x538])", tostring(cnt)), table.concat(hdr, ",") }
  local nrep, nent = REP_LEN / 4, ENT_LEN / 4
  for i = 0, cnt - 1 do
    local b = fkf_buf + i * REC_SIZE
    local row = { tostring(readInteger(b + 0, false)) }
    for k = 0, nrep - 1 do row[#row + 1] = tostring(readInteger(b + 4 + k * 4, false)) end
    for k = 0, nent - 1 do row[#row + 1] = tostring(readInteger(b + 4 + REP_LEN + k * 4, false)) end
    out[#out + 1] = table.concat(row, ",")
  end
  local f, err = io.open(path, "w")
  if not f then print("[FKF] ouverture IMPOSSIBLE: " .. tostring(err)); return end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[FKF] %d entrees ecrites -> %s", cnt, path))
end

function captureKillfeed(target, path)
  target = target or 30
  path = path or defaultDumpPath()
  startKillfeedCapture()
  if not fkf_inj then print("[FKF] demarrage echoue -> repairKillfeedPatch() puis reessaie"); return end
  print(string.format("[FKF] >>> JOUE le film MAINTENANT (j'attends %d entrees ou 90s) <<<", target))
  local ticks = 0
  local t = createTimer(nil)
  t.Interval = 1000
  t.OnTimer = function(timer)
    ticks = ticks + 1
    local cnt = fkf_cnt and readQword(fkf_cnt) or 0
    print(string.format("[FKF] ... %d entrees (t=%ds)", cnt, ticks))
    if cnt >= target or ticks >= 90 then
      timer.destroy(); stopKillfeedCapture(); dumpKillfeed(path)
      print("[FKF] termine. Donne le CSV a l'agent.")
    end
  end
end

print("[FKF] charge. Commande : captureKillfeed()  puis JOUE le film (dump auto -> Downloads).")
print("[FKF] Si 'AOB introuvable' : repairKillfeedPatch()  puis recommence.")
