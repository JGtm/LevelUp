--[==[==========================================================================
  filmdec_dualcap_capture.lua — VÉRITÉ-TERRAIN arme-par-kill (dual-hook pure-read)
  LevelUp / Halo Infinite

  Reconstruit le mécanisme validé 2026-06-11 (97/98 sur 000d5950). Deux hooks
  CE LECTURE-PURE (zéro appel moteur = anti-crash), même horloge RDTSC :

   HOOK DÉGÂT  FUN_1407e00ac (RVA 0x7E00AC, vol 5o `mov [rsp+18],rbx`) :
     R8 = descripteur dégât, RDX = ptr biped victime.
     rec 32o : [00]=attaquant [R8+0x0c] · [04]=victime *[RDX] · [08]=famille [R8+0x10]
               · [0C]=suffixe [R8+0x14] · [10]=rdtsc_low · [14]=rdtsc_high
   HOOK KILL   FUN_1406730c4 (RVA 0x6730C4, vol 7o `mov rax,rsp ; mov [rax+08],rbx`) :
     RDX = event.
     rec 16o : [00]=victime [RDX+0x04] · [04]=tueur [RDX+0x08] · [08]=rdtsc_low · [0C]=rdtsc_high

  Décodé par cmd/tmp_dualcap (idx = (handle-base)/0x10002 ; base 0xEC500000 dégât,
  0xE1500000 kill). Attribution = arme du dernier dégât du tueur (atk==kill) à tsc<=tsc_kill.

  USAGE :
    1) Execute Script (charge les fonctions)
    2) captureDual(150, "9b191a7f")   -- arme les hooks puis arme un timer
    3) JOUE le film en Theater jusqu'au bout (lecture qui AVANCE)
    4) dump auto -> tools/ce/<prefix>_dmg.bin + <prefix>_kill.bin à la fin (ou stopDualCap()+dumpDual())
    repairDual() si un patch reste collé ; stopAllDual() pour tout couper.
==========================================================================]==]

local MODULE = "HaloInfinite.exe"
-- Localisation par RVA + vérification d'octets (les prologues ne sont PAS uniques
-- en AOB : 13/671 matches). La base ASLR est fixe par lancement -> base+RVA exact.
local DMG_RVA  = 0x7E00AC   -- FUN_1407e00ac (apply dégât)
local KILL_RVA = 0x6730C4   -- FUN_1406730c4 (kill-event)
local DMG_BYTES  = { 0x48, 0x89, 0x5C, 0x24, 0x18 }              -- mov [rsp+18],rbx
local KILL_BYTES = { 0x48, 0x8B, 0xC4, 0x48, 0x89, 0x58, 0x08 }  -- mov rax,rsp ; mov [rax+08],rbx
local DMG_STOLEN  = 5
local KILL_STOLEN = 7
local DREC = 0x20
local KREC = 0x10
local MAXREC = 0x4000

dc_dmgInj = dc_dmgInj or nil
dc_killInj = dc_killInj or nil
dc_dmgOrig = dc_dmgOrig or nil
dc_killOrig = dc_killOrig or nil
dc_timer = dc_timer or nil

local function moduleRange()
  local base = getAddress(MODULE); local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

-- Localise un hook via base+RVA et vérifie que les octets d'entrée correspondent
-- (garde-fou anti-mise-à-jour du jeu). Retourne l'adresse absolue ou nil.
local function locateHook(rva, expected, label)
  local base = getAddress(MODULE)
  if not base or base == 0 then print("[DC] module introuvable (re-attacher CE)"); return nil end
  local addr = base + rva
  local got = readBytes(addr, #expected, true)
  if not got then print("[DC] lecture impossible @ " .. string.format("%X", addr)); return nil end
  for i = 1, #expected do
    if got[i] ~= expected[i] then
      print(string.format("[DC] %s : octets inattendus @ %X (got %02X exp %02X) — RVA périmée (maj jeu) ?", label, addr, got[i], expected[i]))
      return nil
    end
  end
  return addr
end

local function killTimer() if dc_timer then pcall(function() dc_timer.destroy() end); dc_timer = nil end end

function startDualCap()
  killTimer()
  if dc_dmgInj or dc_killInj then stopDualCap() end
  local dmg = locateHook(DMG_RVA, DMG_BYTES, "dégât"); if not dmg then return false end
  local kill = locateHook(KILL_RVA, KILL_BYTES, "kill"); if not kill then return false end

  -- Allocations (caves proches du module pour jmp rel32 ; buffers n'importe où).
  local dmgCave = allocateMemory(0x400, dmg)
  local killCave = allocateMemory(0x400, kill)
  local dmgBuf = allocateMemory(DREC * MAXREC)
  local killBuf = allocateMemory(KREC * MAXREC)
  local dmgCnt = allocateMemory(0x40, dmg)
  local killCnt = allocateMemory(0x40, kill)
  if not (dmgCave and killCave and dmgBuf and killBuf and dmgCnt and killCnt) then
    print("[DC] échec alloc"); return false
  end
  writeQword(dmgCnt, 0); writeQword(killCnt, 0)

  for _, s in ipairs({ "dcDmgCave", "dcKillCave", "dcDmgBuf", "dcKillBuf", "dcDmgCnt", "dcKillCnt", "dcDmgInj", "dcKillInj" }) do unregisterSymbol(s) end
  registerSymbol("dcDmgCave", dmgCave, true)
  registerSymbol("dcKillCave", killCave, true)
  registerSymbol("dcDmgBuf", dmgBuf, true)
  registerSymbol("dcKillBuf", killBuf, true)
  registerSymbol("dcDmgCnt", dmgCnt, true)
  registerSymbol("dcKillCnt", killCnt, true)
  registerSymbol("dcDmgInj", dmg, true)
  registerSymbol("dcKillInj", kill, true)

  dc_dmgOrig = readBytes(dmg, DMG_STOLEN, true)
  dc_killOrig = readBytes(kill, KILL_STOLEN, true)

  -- HOOK DÉGÂT. Push rax,rbx,rcx,rdx + flags (rdtsc clobbe rax/rdx ; rdx=param à préserver).
  -- R8 lu seulement (jamais modifié) -> préservé naturellement. Null-checks R8 et RDX.
  local asmDmg = string.format([[
dcDmgCave:
  push rax
  push rbx
  push rcx
  push rdx
  pushfq
  test r8,r8
  jz dc_dmg_end
  mov rbx,[dcDmgCnt]
  cmp rbx,%X
  jae dc_dmg_end
  mov rax,dcDmgBuf
  shl rbx,5
  add rbx,rax
  mov ecx,[r8+0c]
  mov [rbx+00],ecx
  xor ecx,ecx
  test rdx,rdx
  jz dc_dmg_nv
  mov ecx,[rdx]
dc_dmg_nv:
  mov [rbx+04],ecx
  mov ecx,[r8+10]
  mov [rbx+08],ecx
  mov ecx,[r8+14]
  mov [rbx+0c],ecx
  rdtsc
  mov [rbx+10],eax
  mov [rbx+14],edx
  inc qword ptr [dcDmgCnt]
dc_dmg_end:
  popfq
  pop rdx
  pop rcx
  pop rbx
  pop rax
  mov [rsp+18],rbx
  jmp dcDmgInj+05
dcDmgInj:
  jmp dcDmgCave
]], MAXREC)

  -- HOOK KILL. Push rax,rbx,rcx,rdx + flags. RDX=event (lu seulement). Null-check RDX.
  local asmKill = string.format([[
dcKillCave:
  push rax
  push rbx
  push rcx
  push rdx
  pushfq
  test rdx,rdx
  jz dc_kill_end
  mov rbx,[dcKillCnt]
  cmp rbx,%X
  jae dc_kill_end
  mov rax,dcKillBuf
  shl rbx,4
  add rbx,rax
  mov ecx,[rdx+04]
  mov [rbx+00],ecx
  mov ecx,[rdx+08]
  mov [rbx+04],ecx
  rdtsc
  mov [rbx+08],eax
  mov [rbx+0c],edx
  inc qword ptr [dcKillCnt]
dc_kill_end:
  popfq
  pop rdx
  pop rcx
  pop rbx
  pop rax
  mov rax,rsp
  mov [rax+08],rbx
  jmp dcKillInj+07
dcKillInj:
  jmp dcKillCave
  nop
  nop
]], MAXREC)

  if not autoAssemble(asmDmg) then print("[DC] échec autoAssemble DÉGÂT"); return false end
  dc_dmgInj = dmg
  if not autoAssemble(asmKill) then
    print("[DC] échec autoAssemble KILL — rollback dégât")
    writeBytes(dmg, dc_dmgOrig); dc_dmgInj = nil; return false
  end
  dc_killInj = kill
  print(string.format("[DC] hooks ON — dégât @ %X, kill @ %X", dmg, kill))
  return true
end

function stopDualCap()
  killTimer()
  local d = getAddress("dcDmgCnt"); local k = getAddress("dcKillCnt")
  local dn = d and readQword(d) or "?"
  local kn = k and readQword(k) or "?"
  if dc_dmgInj then writeBytes(dc_dmgInj, dc_dmgOrig); dc_dmgInj = nil end
  if dc_killInj then writeBytes(dc_killInj, dc_killOrig); dc_killInj = nil end
  print(string.format("[DC] hooks OFF — dégâts=%s kills=%s", tostring(dn), tostring(kn)))
end

function stopAllDual() killTimer(); stopDualCap() end

function statusDual()
  local d = getAddress("dcDmgCnt"); local k = getAddress("dcKillCnt")
  print(string.format("[DC] dmgInj=%s killInj=%s dégâts=%s kills=%s",
    dc_dmgInj and string.format("%X", dc_dmgInj) or "nil",
    dc_killInj and string.format("%X", dc_killInj) or "nil",
    d and tostring(readQword(d)) or "nil",
    k and tostring(readQword(k)) or "nil"))
end

local function dumpBuf(symBuf, symCnt, recSize, path)
  local cnt = readQword(getAddress(symCnt))
  if cnt > MAXREC then cnt = MAXREC end
  local total = cnt * recSize
  local f, err = io.open(path, "wb")
  if not f then print("[DC] ouverture impossible: " .. tostring(err)); return 0 end
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

function dumpDual(prefix)
  prefix = prefix or "dualcap"
  local home = (os.getenv("USERPROFILE") or "C:/Users/Guillaume"):gsub("\\", "/")
  local dir = home .. "/Downloads/Scripts/LevelUp-go-migration/tools/ce"
  local dpath = dir .. "/" .. prefix .. "_dmg.bin"
  local kpath = dir .. "/" .. prefix .. "_kill.bin"
  local dn = dumpBuf("dcDmgBuf", "dcDmgCnt", DREC, dpath)
  local kn = dumpBuf("dcKillBuf", "dcKillCnt", KREC, kpath)
  print(string.format("[DC] %d dégâts -> %s ; %d kills -> %s", dn, dpath, kn, kpath))
end

function repairDual()
  local base = getAddress(MODULE)
  if base and base ~= 0 then
    writeBytes(base + DMG_RVA, DMG_BYTES); print("[DC] patch dégât restauré @ " .. string.format("%X", base + DMG_RVA))
    writeBytes(base + KILL_RVA, KILL_BYTES); print("[DC] patch kill restauré @ " .. string.format("%X", base + KILL_RVA))
  end
  dc_dmgInj = nil; dc_killInj = nil
end

function captureDual(seconds, prefix)
  seconds = seconds or 150
  prefix = prefix or "dualcap"
  if not startDualCap() then print("[DC] démarrage échoué -> repairDual() puis réessaie"); return end
  print(string.format("[DC] >>> JOUE le film maintenant (lecture qui AVANCE) — dump auto dans %ds <<<", seconds))
  local ticks = 0
  dc_timer = createTimer(nil)
  dc_timer.Interval = 1000
  dc_timer.OnTimer = function()
    ticks = ticks + 1
    local d = readQword(getAddress("dcDmgCnt")); local k = readQword(getAddress("dcKillCnt"))
    if ticks % 5 == 0 then print(string.format("[DC] ... dégâts=%d kills=%d (t=%ds)", d, k, ticks)) end
    if ticks >= seconds then
      killTimer(); stopDualCap(); dumpDual(prefix)
      print("[DC] terminé. Donne les .bin.")
    end
  end
end

print("[DC] chargé. captureDual(150, \"9b191a7f\") puis JOUE le film. stopAllDual() pour couper.")
