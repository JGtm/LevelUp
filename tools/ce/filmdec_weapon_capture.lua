--[==[==========================================================================
  filmdec_weapon_capture.lua
  LevelUp / Halo Infinite -- LECTURE du NOM D'ARME du kill feed

  Le vrai builder d'entree kill feed FUN_1406a6290(entry, report, ...) lit l'arme
  depuis le DamageReport (objet 8000 o) :
     FUN_1404e979c(entry+0xf8, report+0x1200)  ; copie le NOM d'arme (wstring)
     [report+0x11f8] = handle de tag d'arme (-> icone via FUN_141b316b0(5,..))
  On hooke l'entree de FUN_1406a6290 (RDX = report) et on capture :
     - [report+0x11f8] (qword handle d'arme)
     - le NOM d'arme wstring [report+0x1200] (UTF-16, suit la SSO MSVC)
     - [report+0x53c], [report+0x1f30] (flags annexes)
  Le NOM distingue nativement les variantes (gravity hammer vs antigrav).

  USAGE : Execute Script, captureWeapon(), JOUE le film (autour des kills),
  dump auto -> Downloads/filmdec_weapon.csv. repairWeaponPatch() si besoin.
==========================================================================]==]

local MODULE      = "HaloInfinite.exe"
-- prologue FUN_1406a6290 : push rbp/rbx/rsi/rdi/r12/r14/r15 ; lea rbp,[rsp-0x330]
local AOB         = "40 55 53 56 57 41 54 41 56 41 57 48 8D AC 24 D0 FC FF FF"
local NAME_LEN    = 0x60       -- octets de nom copies (48 wchars)
local REC_SIZE    = 0x80
local MAX_RECORDS = 0x8000

fwp_inj  = fwp_inj  or nil
fwp_orig = fwp_orig or nil
fwp_buf  = fwp_buf  or nil
fwp_cnt  = fwp_cnt  or nil
fwp_cave = fwp_cave or nil

local function moduleRange()
  local base = getAddress(MODULE); local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

local function findUnique(pattern)
  local base, size = moduleRange()
  if not base then print("[FWP] module introuvable"); return nil end
  local ms = AOBScan(pattern)
  if ms == nil or ms.Count == 0 then
    print("[FWP] AOB introuvable. Patch residuel ? -> repairWeaponPatch(). pattern=" .. pattern)
    if ms then ms.destroy() end; return nil
  end
  local hit, n = nil, 0
  for i = 0, ms.Count - 1 do
    local a = tonumber(ms[i], 16)
    if a and a >= base and a < base + size then hit = a; n = n + 1 end
  end
  ms.destroy()
  if n ~= 1 then print(string.format("[FWP] attendu 1, trouve %d", n)); return nil end
  return hit
end

local function defaultDumpPath()
  local home = os.getenv("USERPROFILE") or "C:/Users/Guillaume"
  return (home:gsub("\\", "/")) .. "/Downloads/filmdec_weapon.csv"
end

function repairWeaponPatch()
  local inj = findUnique("E9 ?? ?? ?? ?? 41 54 41 56 41 57 48 8D AC 24 D0 FC FF FF")
  if not inj then print("[FWP] aucun patch residuel. Sinon redemarre Halo."); return end
  writeBytes(inj, { 0x40, 0x55, 0x53, 0x56, 0x57 })  -- restaure push rbp/rbx/rsi/rdi
  fwp_inj = nil
  print(string.format("[FWP] patch restaure @ %X. Relance captureWeapon()", inj))
end

function startWeaponCapture()
  if fwp_inj then stopWeaponCapture() end
  local inj = findUnique(AOB); if not inj then return end
  fwp_inj = inj
  fwp_orig = readBytes(inj, 5, true)
  fwp_cnt  = allocateMemory(0x40, inj)
  fwp_cave = allocateMemory(0x400, inj)
  fwp_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (fwp_cnt and fwp_cave and fwp_buf) then print("[FWP] echec alloc"); fwp_inj = nil; return end
  writeQword(fwp_cnt, 0)
  for _, s in ipairs({ "fwpBuf", "fwpCnt", "fwpCave", "fwpInj" }) do unregisterSymbol(s) end
  registerSymbol("fwpBuf", fwp_buf, true)
  registerSymbol("fwpCnt", fwp_cnt, true)
  registerSymbol("fwpCave", fwp_cave, true)
  registerSymbol("fwpInj", fwp_inj, true)

  -- RDX = report (param_2) a l'entree. Scratch volatils: rax, rcx, r10, r11 (sauves).
  -- rdx preserve (lu seulement). rbp/rbx/rsi/rdi re-pushes par les octets voles.
  local asm = string.format([[
fwpCave:
  push rax
  push rcx
  push r10
  push r11
  mov rax,[fwpCnt]
  cmp rax,%X
  jae fwp_pop
  imul r10,rax,%X
  mov r11,fwpBuf
  add r11,r10
  mov rax,[rdx+11F8]            // [0] handle d'arme (qword)
  mov [r11+00],rax
  mov eax,[rdx+53C]             // [8] report+0x53c
  mov [r11+08],eax
  mov eax,[rdx+1F30]            // [C] report+0x1f30
  mov [r11+0C],eax
  // nom d'arme wstring @ rdx+0x1200 : cap=[rdx+0x1218] ; >7 -> heap ptr [rdx+0x1200]
  lea rcx,[rdx+1200]
  mov rax,[rdx+1218]
  cmp rax,7
  jbe fwp_inline
  mov rcx,[rdx+1200]
fwp_inline:
  xor rax,rax
fwp_cp:
  mov r10w,[rcx+rax]
  mov [r11+rax+10],r10w
  add rax,2
  cmp rax,%X
  jb fwp_cp
  inc qword ptr [fwpCnt]
fwp_pop:
  pop r11
  pop r10
  pop rcx
  pop rax
fwp_re:
  push rbp
  push rbx
  push rsi
  push rdi
  jmp fwpInj+05

fwpInj:
  jmp fwpCave
]], MAX_RECORDS, REC_SIZE, NAME_LEN)

  if autoAssemble(asm) then print(string.format("[FWP] capture ON @ %X", inj))
  else print("[FWP] echec autoAssemble"); writeBytes(inj, fwp_orig); fwp_inj = nil end
end

function stopWeaponCapture()
  if not fwp_inj then print("[FWP] pas actif"); return end
  writeBytes(fwp_inj, fwp_orig)
  print(string.format("[FWP] capture OFF -- %d entrees", fwp_cnt and readQword(fwp_cnt) or 0))
  fwp_inj = nil
end

function weaponStatus()
  print(string.format("[FWP] inj=%s entrees=%s",
    fwp_inj and string.format("%X", fwp_inj) or "nil",
    fwp_cnt and tostring(readQword(fwp_cnt)) or "nil"))
end

-- decode 0x60 octets UTF-16LE -> ASCII (stop au premier wchar nul)
local function wstr(b)
  local t = {}
  for k = 0, NAME_LEN - 2, 2 do
    local lo = readBytes(b + k, 1, false)
    local hi = readBytes(b + k + 1, 1, false)
    if not lo then break end
    local c = lo + (hi or 0) * 256
    if c == 0 then break end
    if c >= 32 and c < 127 then t[#t + 1] = string.char(c) else t[#t + 1] = "." end
  end
  return table.concat(t)
end

function dumpWeapon(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\", "/") end
  local cnt = (fwp_buf and fwp_cnt) and readQword(fwp_cnt) or 0
  if cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  local out = { string.format("# weapon entrees=%s", tostring(cnt)), "handleLo,handleHi,r53c,r1f30,name" }
  for i = 0, cnt - 1 do
    local b = fwp_buf + i * REC_SIZE
    out[#out + 1] = string.format("%d,%d,%d,%d,%s",
      readInteger(b + 0, false), readInteger(b + 4, false),
      readInteger(b + 8), readInteger(b + 12), wstr(b + 16))
  end
  local f, err = io.open(path, "w")
  if not f then print("[FWP] ouverture IMPOSSIBLE: " .. tostring(err)); return end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[FWP] %d entrees ecrites -> %s", cnt, path))
end

function captureWeapon(target, path)
  target = target or 30
  path = path or defaultDumpPath()
  startWeaponCapture()
  if not fwp_inj then print("[FWP] demarrage echoue -> repairWeaponPatch() puis reessaie"); return end
  print(string.format("[FWP] >>> JOUE le film MAINTENANT (j'attends %d entrees ou 90s) <<<", target))
  local ticks = 0
  local t = createTimer(nil)
  t.Interval = 1000
  t.OnTimer = function(timer)
    ticks = ticks + 1
    local cnt = fwp_cnt and readQword(fwp_cnt) or 0
    print(string.format("[FWP] ... %d entrees (t=%ds)", cnt, ticks))
    if cnt >= target or ticks >= 90 then
      timer.destroy(); stopWeaponCapture(); dumpWeapon(path)
      print("[FWP] termine. Donne le CSV a l'agent.")
    end
  end
end

print("[FWP] charge. Commande : captureWeapon()  puis JOUE le film (dump auto -> Downloads).")
print("[FWP] Si 'AOB introuvable' : repairWeaponPatch()  puis recommence.")
