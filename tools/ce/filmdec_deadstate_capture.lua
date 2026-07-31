--[==[==========================================================================
  filmdec_deadstate_capture.lua — VÉRITÉ-TERRAIN dead-state (hook pure-read)
  LevelUp / Halo Infinite

  Hooke le déser dead-state FUN_140c1dd44 (RVA 0xC1DD44) en LECTURE PURE pendant
  le replay Theater. But : (1) prouver que EnumA(+4)/EnumB(+8) résolus = victime/
  tueur ; (2) capturer l'état du bitreader à l'entrée du composant pour DIFFER
  contre le décodage offline (trouver le bit exact où ça diverge) ; (3) rdtsc =
  MÊME horloge que le dual-cap 0xd2 -> corrélation same-clock validable en live.

  À l'entrée : RCX = param_1 (state du composant), RDX = param_2 (bitreader).
  Les champs +4/+8/+0x10 du state contiennent le résultat de la frame PRÉCÉDENTE
  (stable sur un cadavre persistant -> victime/tueur réels, 1 frame de retard).

  Record 0x40 :
    [00] rdtsc_lo   [04] rdtsc_hi
    [08] state ptr low                  (identité du biped victime)
    [0c] victime résolue  [RCX+04]      (handle global -> slot via (h-base)/0x10002)
    [10] tueur  résolu    [RCX+08]
    [14] GID arme-source  [RCX+10]
    [18] bitreader+0x28   (compteur bits)   [1c] bitreader+0x2c (compteur bits)
    [20] bitreader+0x38   (bits dans le mot)
    [24] bitreader+0x40 low (ptr lecture)    [3c] bitreader+0x44 (ptr lecture high)
    [28] bitreader+0x00 low   [2c] bitreader+0x08 low   [30] bitreader+0x10 low (fin)
    [34] bitreader+0x30 lo (accumulateur)    [38] bitreader+0x34 hi

  USAGE :
    1) Execute Script
    2) captureDS(150, "000d5950")   -- arme le hook + timer ; JOUE le film
    3) dump auto -> tools/ce/000d5950_deadstate.bin  (ou stopDS()+dumpDS())
    repairDS() si patch collé ; stopAllDS() pour couper.
  Décodé par cmd/tmp_dscap (à écrire) : idx victime/tueur = (handle-base)/0x10002.
==========================================================================]==]

local MODULE = "HaloInfinite.exe"
local DS_RVA   = 0xC1DD44                          -- FUN_140c1dd44 (déser dead-state)
local DS_BYTES = { 0x40, 0x55, 0x53, 0x56, 0x57 }  -- push rbp/rbx/rsi/rdi
local DS_STOLEN = 5
-- DREC must match the asm record stride (imul rbx,rbx,60 = 0x60) AND the offline
-- reader cmd/tmp_oraclediff (REC = 0x60). The earlier 0x40 mis-strided the dump
-- (records overlap-read) -> the 2026-06-13 fix sets it to 0x60.
local DREC = 0x60
local MAXREC = 0x8000

ds_inj = ds_inj or nil
ds_orig = ds_orig or nil
ds_timer = ds_timer or nil

local function locateHook(rva, expected, label)
  local base = getAddress(MODULE)
  if not base or base == 0 then print("[DS] module introuvable (re-attacher CE)"); return nil end
  local addr = base + rva
  local got = readBytes(addr, #expected, true)
  if not got then print("[DS] lecture impossible @ " .. string.format("%X", addr)); return nil end
  for i = 1, #expected do
    if got[i] ~= expected[i] then
      print(string.format("[DS] %s : octets inattendus @ %X (got %02X exp %02X) — RVA périmée (maj jeu) ?", label, addr, got[i], expected[i]))
      return nil
    end
  end
  return addr
end

local function killTimer() if ds_timer then pcall(function() ds_timer.destroy() end); ds_timer = nil end end

function startDS()
  killTimer()
  if ds_inj then stopDS() end
  local ds = locateHook(DS_RVA, DS_BYTES, "dead-state"); if not ds then return false end

  local cave = allocateMemory(0x400, ds)
  local buf = allocateMemory(DREC * MAXREC)
  local cnt = allocateMemory(0x40, ds)
  if not (cave and buf and cnt) then print("[DS] échec alloc"); return false end
  writeQword(cnt, 0)

  for _, s in ipairs({ "dsCave", "dsBuf", "dsCnt", "dsInj" }) do unregisterSymbol(s) end
  registerSymbol("dsCave", cave, true)
  registerSymbol("dsBuf", buf, true)
  registerSymbol("dsCnt", cnt, true)
  registerSymbol("dsInj", ds, true)

  ds_orig = readBytes(ds, DS_STOLEN, true)

  -- Hook pure-read. À l'entrée RCX=state, RDX=bitreader. rdtsc clobbe rax/rdx ->
  -- on sauve bitreader dans r8, state dans r9 AVANT rdtsc. Null-checks RCX/RDX.
  local asm = string.format([[
dsCave:
  push rax
  push rbx
  push rcx
  push rdx
  push r8
  push r9
  pushfq
  test rcx,rcx
  jz ds_end
  test rdx,rdx
  jz ds_end
  mov r8,rdx
  mov r9,rcx
  mov rbx,[dsCnt]
  cmp rbx,%X
  jae ds_end
  mov rax,dsBuf
  imul rbx,rbx,60
  add rbx,rax
  rdtsc
  mov [rbx+00],eax
  mov [rbx+04],edx
  mov eax,r9d
  mov [rbx+08],eax
  mov eax,[r9+04]
  mov [rbx+0c],eax
  mov eax,[r9+08]
  mov [rbx+10],eax
  mov eax,[r9+10]
  mov [rbx+14],eax
  mov eax,[r8+28]
  mov [rbx+18],eax
  mov eax,[r8+2c]
  mov [rbx+1c],eax
  mov eax,[r8+38]
  mov [rbx+20],eax
  mov eax,[r8+40]
  mov [rbx+24],eax
  mov eax,[r8+00]
  mov [rbx+28],eax
  mov eax,[r8+08]
  mov [rbx+2c],eax
  mov eax,[r8+10]
  mov [rbx+30],eax
  mov eax,[r8+30]
  mov [rbx+34],eax
  mov eax,[r8+34]
  mov [rbx+38],eax
  mov eax,[r8+44]
  mov [rbx+3c],eax
  mov rcx,[r8+08]
  test rcx,rcx
  jz ds_fp_done
  mov eax,[rcx+00]
  mov [rbx+40],eax
  mov eax,[rcx+04]
  mov [rbx+44],eax
  mov eax,[rcx+08]
  mov [rbx+48],eax
  mov eax,[rcx+0c]
  mov [rbx+4c],eax
ds_fp_done:
  inc qword ptr [dsCnt]
ds_end:
  popfq
  pop r9
  pop r8
  pop rdx
  pop rcx
  pop rbx
  pop rax
  push rbp
  push rbx
  push rsi
  push rdi
  jmp dsInj+05
dsInj:
  jmp dsCave
]], MAXREC)

  if not autoAssemble(asm) then print("[DS] échec autoAssemble"); return false end
  ds_inj = ds
  print(string.format("[DS] hook ON — dead-state @ %X", ds))
  return true
end

function stopDS()
  killTimer()
  local c = getAddress("dsCnt")
  local n = c and readQword(c) or "?"
  if ds_inj then writeBytes(ds_inj, ds_orig); ds_inj = nil end
  print(string.format("[DS] hook OFF — dead-states=%s", tostring(n)))
end

function stopAllDS() killTimer(); stopDS() end

function statusDS()
  local c = getAddress("dsCnt")
  print(string.format("[DS] inj=%s dead-states=%s",
    ds_inj and string.format("%X", ds_inj) or "nil",
    c and tostring(readQword(c)) or "nil"))
end

local function dumpBuf(symBuf, symCnt, recSize, path)
  local cnt = readQword(getAddress(symCnt))
  if cnt > MAXREC then cnt = MAXREC end
  local total = cnt * recSize
  local f, err = io.open(path, "wb")
  if not f then print("[DS] ouverture impossible: " .. tostring(err)); return 0 end
  if total > 0 then
    local bytes = readBytes(getAddress(symBuf), total, true)
    local chunk = {}
    for i = 1, total do
      chunk[#chunk + 1] = string.char(bytes[i])
      if #chunk >= 4096 then f:write(table.concat(chunk)); chunk = {} end
    end
    if #chunk > 0 then f:write(table.concat(chunk)) end
  end
  f:close()
  return cnt
end

function dumpDS(prefix)
  prefix = prefix or "deadstate"
  local home = (os.getenv("USERPROFILE") or "C:/Users/Guillaume"):gsub("\\", "/")
  local dir = home .. "/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce"
  local path = dir .. "/" .. prefix .. "_deadstate.bin"
  local n = dumpBuf("dsBuf", "dsCnt", DREC, path)
  print(string.format("[DS] %d dead-states -> %s", n, path))
end

function repairDS()
  local base = getAddress(MODULE)
  if base and base ~= 0 then
    writeBytes(base + DS_RVA, DS_BYTES); print("[DS] patch restauré @ " .. string.format("%X", base + DS_RVA))
  end
  ds_inj = nil
end

function captureDS(seconds, prefix)
  seconds = seconds or 150
  prefix = prefix or "deadstate"
  if not startDS() then print("[DS] démarrage échoué -> repairDS() puis réessaie"); return end
  print(string.format("[DS] >>> JOUE le film maintenant (lecture qui AVANCE) — dump auto dans %ds <<<", seconds))
  local ticks = 0
  ds_timer = createTimer(nil)
  ds_timer.Interval = 1000
  ds_timer.OnTimer = function()
    ticks = ticks + 1
    local n = readQword(getAddress("dsCnt"))
    if ticks % 5 == 0 then print(string.format("[DS] ... dead-states=%d (t=%ds)", n, ticks)) end
    if ticks >= seconds then
      killTimer(); stopDS(); dumpDS(prefix)
      print("[DS] terminé. Donne le .bin.")
    end
  end
end

print("[DS] chargé. captureDS(150, \"000d5950\") puis JOUE le film. stopAllDS() pour couper.")
