--[==[==========================================================================
  filmdec_killweapon_capture.lua  (v2 : hook plus tot + totalHits + timer unique)
  LevelUp / Halo Infinite -- ARME du kill feed (FUN_1406730c4, replay Theater)

  FUN_1406730c4 :
     140673120  CALL FUN_1404969f0    ; R13 = composant d'event (deserialise du FILM)
     140673161  MOV EBX,[R13+0x538]   ; <-- ON HOOKE ICI (avant la resolution/saut)
     14067316a  CALL FUN_14049d198    ; resout handle -> def-id ; saute si -1
  On capture, a 0x673161 : [R13+0x538] (handle d'arme BRUT), [R13+0x1f30]
  (attaquant), et param_2=RSI (l'event : tueur/victime/flags). Un compteur
  totalHits dit si la fonction est atteinte au replay.

  USAGE : Execute Script ; captureKillWeapon() ; JOUE le film (lecture qui AVANCE,
  traverse les kills). Dump auto -> Downloads/filmdec_killweapon.csv.
  Diagnostic : killWeaponStatus() (voir totalHits). repairKillWeaponPatch() si patch residuel.
==========================================================================]==]

local MODULE      = "HaloInfinite.exe"
-- ancrage = MOV EBX,[R13+538] ; MOV ECX,EBX ; CALL FUN_14049d198(E8). On hooke le CALL
-- pour le re-executer NOUS-MEMES et capturer EAX = def-id d'arme STABLE (+ handle EBX).
local AOB         = "41 8B 9D 38 05 00 00 8B CB E8"
local INJ_OFF     = 0x09        -- offset du CALL (E8) depuis le debut de l'AOB
local STOLEN_LEN  = 5           -- E8 xx xx xx xx (CALL FUN_14049d198)
local EVT_LEN     = 0x50
local REC_SIZE    = 0x60        -- defId(4)+handle(4)+attacker(4)+EVT_LEN
local MAX_RECORDS = 0x8000

fkw_inj  = fkw_inj  or nil
fkw_orig = fkw_orig or nil
fkw_buf  = fkw_buf  or nil
fkw_cnt  = fkw_cnt  or nil
fkw_tot  = fkw_tot  or nil
fkw_cave = fkw_cave or nil
fkw_timer = fkw_timer or nil

local function moduleRange()
  local base = getAddress(MODULE); local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

local function findUnique(pattern)
  local base, size = moduleRange()
  if not base then print("[FKW] module introuvable (re-attacher CE au process Halo)"); return nil end
  local ms = AOBScan(pattern)
  if ms == nil or ms.Count == 0 then
    print("[FKW] AOB introuvable. pattern=" .. pattern); if ms then ms.destroy() end; return nil
  end
  local hit, n = nil, 0
  for i = 0, ms.Count - 1 do
    local a = tonumber(ms[i], 16)
    if a and a >= base and a < base + size then hit = a; n = n + 1 end
  end
  ms.destroy()
  if n ~= 1 then print(string.format("[FKW] attendu 1, trouve %d", n)); return nil end
  return hit
end

local function defaultDumpPath()
  local home = os.getenv("USERPROFILE") or "C:/Users/Guillaume"
  return (home:gsub("\\", "/")) .. "/Downloads/filmdec_killweapon.csv"
end

local function killTimer()
  if fkw_timer then local ok = pcall(function() fkw_timer.destroy() end); fkw_timer = nil end
end

function repairKillWeaponPatch()
  -- anchor non-vole : MOV EBX,[R13+538] ; MOV ECX,EBX (juste avant le CALL patche)
  local m = findUnique("41 8B 9D 38 05 00 00 8B CB")
  if not m then print("[FKW] aucun patch residuel. Sinon redemarre Halo."); return end
  local inj = m + INJ_OFF
  writeBytes(inj, { 0xE8, 0x29, 0xA0, 0xE2, 0xFF })  -- restaure CALL FUN_14049d198
  fkw_inj = nil
  print(string.format("[FKW] patch restaure @ %X. Relance captureKillWeapon()", inj))
end

function startKillWeaponCapture()
  killTimer()
  if fkw_inj then stopKillWeaponCapture() end
  local start = findUnique(AOB); if not start then return end
  local inj = start + INJ_OFF
  fkw_inj  = inj
  fkw_orig = readBytes(inj, STOLEN_LEN, true)
  -- cible du CALL = FUN_14049d198 = inj + 5 + rel32(octets voles)
  local rel = fkw_orig[2] + fkw_orig[3] * 0x100 + fkw_orig[4] * 0x10000 + fkw_orig[5] * 0x1000000
  if rel >= 0x80000000 then rel = rel - 0x100000000 end
  local defFn = inj + 5 + rel
  fkw_cnt  = allocateMemory(0x40, inj)
  fkw_tot  = allocateMemory(0x40, inj)
  fkw_cave = allocateMemory(0x400, inj)
  fkw_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (fkw_cnt and fkw_tot and fkw_cave and fkw_buf) then print("[FKW] echec alloc"); fkw_inj = nil; return end
  writeQword(fkw_cnt, 0); writeQword(fkw_tot, 0)
  for _, s in ipairs({ "fkwBuf", "fkwCnt", "fkwTot", "fkwCave", "fkwInj", "fkwDefFn" }) do unregisterSymbol(s) end
  registerSymbol("fkwBuf", fkw_buf, true)
  registerSymbol("fkwCnt", fkw_cnt, true)
  registerSymbol("fkwTot", fkw_tot, true)
  registerSymbol("fkwCave", fkw_cave, true)
  registerSymbol("fkwInj", fkw_inj, true)
  registerSymbol("fkwDefFn", defFn, true)

  -- a l'injection (le CALL) : ECX=handle (MOV ECX,EBX juste avant), EBX=handle,
  -- R13=event comp, RSI=param_2. On re-execute le CALL pour obtenir EAX=def-id.
  -- preserver R13/RSI/RBP/RDI ; EAX doit valoir def-id en sortie (push/pop rax).
  local asm = string.format([[
fkwCave:
  call fkwDefFn                  // EAX = def-id d'arme (FUN_14049d198(handle))
  inc qword ptr [fkwTot]
  push rax
  push rcx
  push rdx
  push r8
  push r9
  mov r9d,eax                   // sauve def-id
  mov rax,[fkwCnt]
  cmp rax,%X
  jae fkw_pop
  imul rdx,rax,%X
  mov r8,fkwBuf
  add r8,rdx
  mov [r8+00],r9d              // [0] def-id arme (STABLE)
  mov [r8+04],ebx             // [4] handle d'arme
  mov ecx,[r13+1F30]          // [8] attaquant
  mov [r8+08],ecx
  xor rdx,rdx
fkw_cp:
  mov ecx,[rsi+rdx]          // [0xC..] param_2 (event) 0..EVT_LEN
  mov [r8+rdx+0C],ecx
  add rdx,4
  cmp rdx,%X
  jb fkw_cp
  inc qword ptr [fkwCnt]
fkw_pop:
  pop r9
  pop r8
  pop rdx
  pop rcx
  pop rax                      // restaure EAX = def-id (resultat du CALL)
fkw_re:
  jmp fkwInj+05                 // retour apres le CALL (0x67316f : CMP EAX,-1)

fkwInj:
  jmp fkwCave
]], MAX_RECORDS, REC_SIZE, EVT_LEN)

  if autoAssemble(asm) then print(string.format("[FKW] capture ON @ %X (defFn=%X)", inj, defFn))
  else print("[FKW] echec autoAssemble"); writeBytes(inj, fkw_orig); fkw_inj = nil end
end

function stopKillWeaponCapture()
  killTimer()
  if not fkw_inj then print("[FKW] pas actif"); return end
  writeBytes(fkw_inj, fkw_orig)
  print(string.format("[FKW] capture OFF -- kills=%s totalHits=%s",
    fkw_cnt and tostring(readQword(fkw_cnt)) or "?", fkw_tot and tostring(readQword(fkw_tot)) or "?"))
  fkw_inj = nil
end

function killWeaponStatus()
  print(string.format("[FKW] inj=%s kills=%s totalHits=%s",
    fkw_inj and string.format("%X", fkw_inj) or "nil",
    fkw_cnt and tostring(readQword(fkw_cnt)) or "nil",
    fkw_tot and tostring(readQword(fkw_tot)) or "nil"))
end

function dumpKillWeapon(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\", "/") end
  local cnt = (fkw_buf and fkw_cnt) and readQword(fkw_cnt) or 0
  local tot = fkw_tot and readQword(fkw_tot) or 0
  if cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  local hdr = { "weaponDefId", "weaponHandle", "attacker" }
  for o = 0, EVT_LEN - 1, 4 do hdr[#hdr + 1] = string.format("p%02X", o) end
  local out = { string.format("# killweapon kills=%s totalHits=%s (weaponDefId=FUN_14049d198([event+0x538]))", tostring(cnt), tostring(tot)), table.concat(hdr, ",") }
  local nevt = EVT_LEN / 4
  for i = 0, cnt - 1 do
    local b = fkw_buf + i * REC_SIZE
    local row = { tostring(readInteger(b + 0)), tostring(readInteger(b + 4, false)), tostring(readInteger(b + 8, false)) }
    for k = 0, nevt - 1 do row[#row + 1] = tostring(readInteger(b + 12 + k * 4, false)) end
    out[#out + 1] = table.concat(row, ",")
  end
  local f, err = io.open(path, "w")
  if not f then print("[FKW] ouverture IMPOSSIBLE: " .. tostring(err)); return end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[FKW] %d kills ecrits (totalHits=%d) -> %s", cnt, tot, path))
end

function captureKillWeapon(target)
  target = target or 100  -- capturer TOUS les kills (le match en a 93) -- avant : 12
  startKillWeaponCapture()
  if not fkw_inj then print("[FKW] demarrage echoue -> repairKillWeaponPatch() puis reessaie"); return end
  print(string.format("[FKW] >>> JOUE le film ENTIER (lecture qui AVANCE, du debut a la fin) -- arret a %d kills ou 600s <<<", target))
  local ticks = 0
  fkw_timer = createTimer(nil)
  fkw_timer.Interval = 1000
  fkw_timer.OnTimer = function()
    ticks = ticks + 1
    local cnt = fkw_cnt and readQword(fkw_cnt) or 0
    local tot = fkw_tot and readQword(fkw_tot) or 0
    print(string.format("[FKW] ... kills=%d totalHits=%d (t=%ds)", cnt, tot, ticks))
    if cnt >= target or ticks >= 600 then
      killTimer()
      stopKillWeaponCapture()
      dumpKillWeapon()
      print("[FKW] termine. Donne le CSV a l'agent (+ dis-moi totalHits).")
    end
  end
end

-- arret manuel d'urgence si un timer traine
function stopAllKillWeapon() killTimer(); stopKillWeaponCapture() end

function diagCompare()
  local function scan(p) local r = AOBScan(p); local c = (r and r.Count) or 0; if r then r.destroy() end; return c end
  print(string.format("[FKW] base=%X", getAddress(MODULE)))
  print(string.format("[FKW] deadstate(connu)  -> %d", scan("48 8D 4F 74 48 8B D3 E8 2E 00 00 00")))
  print(string.format("[FKW] killfeed(connu)   -> %d", scan("8B 98 38 05 00 00 83 FB FF 0F 84")))
  print(string.format("[FKW] killweapon        -> %d", scan(AOB)))
end

print("[FKW] v2 charge. captureKillWeapon() puis JOUE le film (lecture qui avance).")
print("[FKW] stopAllKillWeapon() pour tout arreter. killWeaponStatus() pour voir totalHits.")
