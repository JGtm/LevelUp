--[==[==========================================================================
  filmdec_dmgsource_capture.lua  (v3 : SOURCE DE DEGAT, pas l'arme tenue)
  LevelUp / Halo Infinite -- ARME du kill = SOURCE DE DEGAT de l'event de mort.

  Pourquoi v3 : la v2 (filmdec_killweapon_capture.lua) hookait event+0x538 qui s'est
  revele etre l'ENTITE DU TUEUR (8 valeurs joueur, == tueur 98/98), PAS l'arme.
  Ici on hooke le VRAI handler de kill FUN_140b478d8 (dispatche par table d'events,
  rejoue au replay : il pilote kill-feed + effets de mort). Ses parametres portent la
  source de degat :
     param_5 ([RSP+0x28]) = report struct  -> +0x04 = DamageSourceObjectDescriptionSessionIndex
     param_6 ([RSP+0x30]) = damage-report handle -> DamageEffectDefinitionGlobalTagId
                            = *(u32)((p6>>0xd)*0x34 + 4 + *(0x14494A908+0x78))   [table LIVE]
     param_2 ([RSP+0x10] apres home, =EDX) = participant (victime)
     param_7 ([RSP+0x38]) = participant (tueur)
  On capture TOUT (handle brut + tete du report struct + tag calcule) : on decide apres
  quelle colonne discrimine l'arme. totalHits dit si le handler est atteint AU REPLAY.

  Hook = ENTREE de FUN_140b478d8 (@140b478d8). Home-block de 18 octets vole, re-execute
  dans la cave, retour a entry+0x12 (PUSH RBP).

  USAGE : Execute Script ; captureDmgSource() ; JOUE le film (lecture qui AVANCE).
  Dump auto -> Downloads/filmdec_dmgsource.csv. Diag : dmgSourceStatus().
==========================================================================]==]

local MODULE      = "HaloInfinite.exe"
-- AOB = prologue de FUN_140b478d8 (home-block + 6 push), tres specifique :
--   48 8B C4            mov rax,rsp
--   4C 89 48 20         mov [rax+20],r9
--   44 89 40 18         mov [rax+18],r8d
--   89 50 10            mov [rax+10],edx
--   48 89 48 08         mov [rax+8],rcx
--   55 53 56 57 41 54..  push rbp/rbx/rsi/rdi/r12/r13/r14/r15 ; lea rbp,[rsp-78] ; sub rsp,178 ; mov r15,rcx ; mov ebx,1003
-- (etendu jusqu'a MOV EBX,0x1003 pour etre UNIQUE -- le prologue seul matche 5 sites)
local AOB         = "48 8B C4 4C 89 48 20 44 89 40 18 89 50 10 48 89 48 08 55 53 56 57 41 54 41 55 41 56 41 57 48 8D 6C 24 88 48 81 EC 78 01 00 00 4C 8B F9 BB 03 10 00 00"
local STOLEN_LEN  = 0x12        -- 18 octets = home-block (re-execute dans la cave)
local RET_OFF     = 0x12        -- retour a entry+0x12 (PUSH RBP)
local REC_SIZE    = 0x30        -- 12 u32 par record
local NFIELDS     = REC_SIZE / 4
local MAX_RECORDS = 0x8000
-- *(DAT+0x78) = base table des damage-effect-def (LIVE). RVA (base preferee Ghidra 0x140000000) ;
-- resolu en absolu via getAddress(MODULE)+RVA a l'injection (robuste a l'ASLR).
local DAT_TAGTBL_RVA = 0x494A908 -- = 0x14494A908 - 0x140000000

fds_inj  = fds_inj  or nil
fds_orig = fds_orig or nil
fds_buf  = fds_buf  or nil
fds_cnt  = fds_cnt  or nil
fds_tot  = fds_tot  or nil
fds_cave = fds_cave or nil
fds_timer = fds_timer or nil

local function moduleRange()
  local base = getAddress(MODULE); local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

local function findUnique(pattern)
  local base, size = moduleRange()
  if not base then print("[FDS] module introuvable (re-attacher CE au process Halo)"); return nil end
  local ms = AOBScan(pattern)
  if ms == nil or ms.Count == 0 then
    print("[FDS] AOB introuvable. pattern=" .. pattern); if ms then ms.destroy() end; return nil
  end
  local hit, n = nil, 0
  for i = 0, ms.Count - 1 do
    local a = tonumber(ms[i], 16)
    if a and a >= base and a < base + size then hit = a; n = n + 1 end
  end
  ms.destroy()
  if n ~= 1 then print(string.format("[FDS] attendu 1, trouve %d", n)); return nil end
  return hit
end

local function defaultDumpPath()
  local home = os.getenv("USERPROFILE") or "C:/Users/Guillaume"
  return (home:gsub("\\", "/")) .. "/Downloads/filmdec_dmgsource.csv"
end

local function killTimer()
  if fds_timer then pcall(function() fds_timer.destroy() end); fds_timer = nil end
end

function repairDmgSourcePatch()
  local m = findUnique(AOB)
  if not m then print("[FDS] introuvable / deja restaure. Sinon redemarre Halo."); return end
  print(string.format("[FDS] entree @ %X. Si patch residuel, redemarre Halo (home-block ecrase).", m))
end

function startDmgSourceCapture()
  killTimer()
  if fds_inj then stopDmgSourceCapture() end
  local start = findUnique(AOB); if not start then return end
  local inj = start
  local datTbl = getAddress(MODULE) + DAT_TAGTBL_RVA -- table tags absolue (robuste ASLR)
  fds_inj  = inj
  fds_orig = readBytes(inj, STOLEN_LEN, true)
  fds_cnt  = allocateMemory(0x40, inj)
  fds_tot  = allocateMemory(0x40, inj)
  fds_cave = allocateMemory(0x800, inj)
  fds_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (fds_cnt and fds_tot and fds_cave and fds_buf) then print("[FDS] echec alloc"); fds_inj = nil; return end
  writeQword(fds_cnt, 0); writeQword(fds_tot, 0)
  for _, s in ipairs({ "fdsBuf", "fdsCnt", "fdsTot", "fdsCave", "fdsInj" }) do unregisterSymbol(s) end
  registerSymbol("fdsBuf", fds_buf, true)
  registerSymbol("fdsCnt", fds_cnt, true)
  registerSymbol("fdsTot", fds_tot, true)
  registerSymbol("fdsCave", fds_cave, true)
  registerSymbol("fdsInj", fds_inj, true)

  local asm = string.format([[
fdsCave:
  // --- home-block vole (rsp = entry rsp ici) ---
  mov rax,rsp
  mov [rax+20],r9
  mov [rax+18],r8d
  mov [rax+10],edx
  mov [rax+08],rcx
  inc qword ptr [fdsTot]
  push rax
  push rbx
  push rcx
  push rdx
  push r8
  push r9
  push r10
  push r11
  mov r10,[fdsCnt]
  cmp r10,%X
  jae fds_done
  imul r10,r10,%X
  mov r11,fdsBuf
  add r11,r10
  // participants
  mov ecx,[rax+10]
  mov [r11+00],ecx
  mov ecx,[rax+38]
  mov [r11+04],ecx
  mov ecx,[rax+18]
  mov [r11+08],ecx
  // p6 = damage-report handle brut
  mov r8d,[rax+30]
  mov [r11+0C],r8d
  // p5 = report struct ptr (low32 pour debug/null)
  mov rdx,[rax+28]
  mov [r11+14],edx
  // tete du report struct [0..0x14], garde null
  xor ecx,ecx
  test rdx,rdx
  jz fds_r0
  mov ecx,[rdx+00]
fds_r0:
  mov [r11+18],ecx
  xor ecx,ecx
  test rdx,rdx
  jz fds_r4
  mov ecx,[rdx+04]
fds_r4:
  mov [r11+1C],ecx
  xor ecx,ecx
  test rdx,rdx
  jz fds_r8
  mov ecx,[rdx+08]
fds_r8:
  mov [r11+20],ecx
  xor ecx,ecx
  test rdx,rdx
  jz fds_rC
  mov ecx,[rdx+0C]
fds_rC:
  mov [r11+24],ecx
  xor ecx,ecx
  test rdx,rdx
  jz fds_r10
  mov ecx,[rdx+10]
fds_r10:
  mov [r11+28],ecx
  xor ecx,ecx
  test rdx,rdx
  jz fds_r14
  mov ecx,[rdx+14]
fds_r14:
  mov [r11+2C],ecx
  // DamageEffectDefinitionGlobalTagId depuis p6 (r8d)
  mov ecx,0FFFFFFFF
  mov r9d,r8d
  add r9d,1
  cmp r9d,2
  jb fds_tag
  mov r9d,r8d
  shr r9d,0D
  imul r9,r9,34
  mov rdx,%X
  mov rdx,[rdx+78]
  test rdx,rdx
  jz fds_tag
  mov ecx,[r9+rdx+04]
fds_tag:
  mov [r11+10],ecx
  inc qword ptr [fdsCnt]
fds_done:
  pop r11
  pop r10
  pop r9
  pop r8
  pop rdx
  pop rcx
  pop rbx
  pop rax
  jmp fdsInj+%X

fdsInj:
  jmp fdsCave
]], MAX_RECORDS, REC_SIZE, datTbl, RET_OFF)

  if autoAssemble(asm) then
    -- NOP-fill du reste du home-block vole (inj+5 .. inj+0x11)
    local nops = {}
    for i = 1, STOLEN_LEN - 5 do nops[i] = 0x90 end
    writeBytes(inj + 5, nops)
    print(string.format("[FDS] capture ON @ %X (cave=%X)", inj, fds_cave))
  else
    print("[FDS] echec autoAssemble"); writeBytes(inj, fds_orig); fds_inj = nil
  end
end

function stopDmgSourceCapture()
  killTimer()
  if not fds_inj then print("[FDS] pas actif"); return end
  writeBytes(fds_inj, fds_orig)
  print(string.format("[FDS] capture OFF -- kills=%s totalHits=%s",
    fds_cnt and tostring(readQword(fds_cnt)) or "?", fds_tot and tostring(readQword(fds_tot)) or "?"))
  fds_inj = nil
end

function dmgSourceStatus()
  print(string.format("[FDS] inj=%s kills=%s totalHits=%s",
    fds_inj and string.format("%X", fds_inj) or "nil",
    fds_cnt and tostring(readQword(fds_cnt)) or "nil",
    fds_tot and tostring(readQword(fds_tot)) or "nil"))
end

function dumpDmgSource(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\", "/") end
  local cnt = (fds_buf and fds_cnt) and readQword(fds_cnt) or 0
  local tot = fds_tot and readQword(fds_tot) or 0
  if cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  local hdr = { "victim", "killer", "p3", "dmgReportRaw", "dmgEffectTagId", "param5ptr",
                "rep00", "rep04_dmgSrcSessionIdx", "rep08", "rep0C_disp", "rep10", "rep14" }
  local out = {
    string.format("# dmgsource kills=%s totalHits=%s (hook FUN_140b478d8 entry ; arme=source de degat)", tostring(cnt), tostring(tot)),
    table.concat(hdr, ",") }
  for i = 0, cnt - 1 do
    local b = fds_buf + i * REC_SIZE
    local row = {}
    for k = 0, NFIELDS - 1 do row[#row + 1] = tostring(readInteger(b + k * 4, false)) end
    out[#out + 1] = table.concat(row, ",")
  end
  local f, err = io.open(path, "w")
  if not f then print("[FDS] ouverture IMPOSSIBLE: " .. tostring(err)); return end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[FDS] %d kills ecrits (totalHits=%d) -> %s", cnt, tot, path))
end

function captureDmgSource(target)
  target = target or 100
  startDmgSourceCapture()
  if not fds_inj then print("[FDS] demarrage echoue -> repairDmgSourcePatch() puis reessaie"); return end
  print(string.format("[FDS] >>> JOUE le film ENTIER (lecture qui AVANCE) -- arret a %d kills ou 600s <<<", target))
  local ticks = 0
  fds_timer = createTimer(nil)
  fds_timer.Interval = 1000
  fds_timer.OnTimer = function()
    ticks = ticks + 1
    local cnt = fds_cnt and readQword(fds_cnt) or 0
    local tot = fds_tot and readQword(fds_tot) or 0
    print(string.format("[FDS] ... kills=%d totalHits=%d (t=%ds)", cnt, tot, ticks))
    if cnt >= target or ticks >= 600 then
      killTimer(); stopDmgSourceCapture(); dumpDmgSource()
      print("[FDS] termine. Donne le CSV a l'agent (+ dis-moi totalHits).")
    end
  end
end

function stopAllDmgSource() killTimer(); stopDmgSourceCapture() end

print("[FDS] v3 charge. captureDmgSource() puis JOUE le film.")
print("[FDS] IMPORTANT : si totalHits=0 apres lecture, FUN_140b478d8 ne tourne PAS au replay (-> me le dire).")
print("[FDS] stopAllDmgSource() pour tout arreter. dmgSourceStatus() pour voir totalHits.")
