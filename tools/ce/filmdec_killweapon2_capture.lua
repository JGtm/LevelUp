--[==[==========================================================================
  filmdec_killweapon2_capture.lua  (v4 : resout le variant_name = NOM d'arme)
  LevelUp / Halo Infinite -- kill feed AUTO-NOMME (zero narration)

  Dans FUN_1406730c4, apres lVar12 = FUN_140495abc(weapon_handle) (RAX = entite-arme
  0x358), on resout NOUS-MEMES le composant 'obje' de l'entite-arme et on lit son
  variant_name (= high-32 de l'id64 catalogue analysis.WeaponIDToName = la FAMILLE
  d'arme). Chaine (RE statique, workflow wumaiev2d / TACHE C) :
     record = FUN_1405839d0(entiteArme + 0x2c, 'obje'=0x6f626a65)
     variant_name = *(int*)(record + 0x14)
  Hook @ 0x67318f (MOV RDX,RDI ; MOV RCX,RAX) ou RAX = lVar12 (entite-arme non-nulle).
  Capture par kill : variant_name (famille), handle, et la fenetre event (tueur/victime).

  USAGE : Execute Script ; captureKW2() ; JOUE le film en Theater (lecture qui avance).
  Dump auto -> Downloads/filmdec_killweapon2.csv. repairKW2() si patch residuel.
  stopAllKW2() pour tout couper.
==========================================================================]==]

local MODULE     = "HaloInfinite.exe"
-- Hook AVANT la garde def-id (a 0x673161 = MOV EBX,[R13+0x538], qui a donne totalHits=12).
-- On resout NOUS-MEMES handle -> entite-arme (FUN_140495abc) -> 'obje' (FUN_1405839d0) ->
-- variant_name, sans la garde du jeu. AOB sans wildcard (CE bute sur ??).
local AOB        = "41 8B 9D 38 05 00 00 8B CB E8"   -- MOV EBX,[R13+538] ; MOV ECX,EBX ; CALL
local AOB_REPAIR = "8B CB E8 29 A0 E2 FF"            -- MOV ECX,EBX ; CALL FUN_14049d198 (apres les octets voles)
local INJ_OFF    = 0x00        -- injection au debut (le MOV EBX)
local STOLEN_LEN = 7           -- 41 8B 9D 38 05 00 00
local RESOLVE_RVA = 0x495abc   -- FUN_140495abc (handle -> entite-arme 0x358)
local OBJE_RVA    = 0x5839d0   -- FUN_1405839d0 (composant par group-tag : 'obje')
local EVT_LEN    = 0x20        -- fenetre event (tueur/victime)
local REC_SIZE   = 0x30        -- variant(4)+handle(4)+EVT_LEN
local MAX_RECORDS = 0x8000

kw2_inj = kw2_inj or nil
kw2_orig = kw2_orig or nil
kw2_buf = kw2_buf or nil
kw2_cnt = kw2_cnt or nil
kw2_tot = kw2_tot or nil
kw2_cave = kw2_cave or nil
kw2_scratch = kw2_scratch or nil
kw2_timer = kw2_timer or nil

local function moduleRange()
  local base = getAddress(MODULE); local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

local function findUnique(pattern)
  local base, size = moduleRange()
  if not base then print("[KW2] module introuvable (re-attacher CE)"); return nil end
  local ms = AOBScan(pattern)
  if ms == nil or ms.Count == 0 then print("[KW2] AOB introuvable. pattern=" .. pattern); if ms then ms.destroy() end; return nil end
  local hit, n = nil, 0
  for i = 0, ms.Count - 1 do
    local a = tonumber(ms[i], 16)
    if a and a >= base and a < base + size then hit = a; n = n + 1 end
  end
  ms.destroy()
  if n ~= 1 then print(string.format("[KW2] attendu 1, trouve %d", n)); return nil end
  return hit
end

local function defaultDumpPath()
  local home = os.getenv("USERPROFILE") or "C:/Users/Guillaume"
  return (home:gsub("\\", "/")) .. "/Downloads/filmdec_killweapon2.csv"
end

local function killTimer() if kw2_timer then pcall(function() kw2_timer.destroy() end); kw2_timer = nil end end

function repairKW2()
  local m = findUnique(AOB_REPAIR)  -- ancrage apres les octets voles (survit au patch)
  if not m then print("[KW2] ancrage introuvable. Sinon redemarre Halo."); return end
  local inj = m - STOLEN_LEN
  writeBytes(inj, { 0x41, 0x8B, 0x9D, 0x38, 0x05, 0x00, 0x00 })  -- MOV EBX,[R13+0x538]
  kw2_inj = nil
  print(string.format("[KW2] patch restaure @ %X.", inj))
end

function startKW2()
  killTimer()
  if kw2_inj then stopKW2() end
  local start = findUnique(AOB); if not start then return end
  local inj = start + INJ_OFF
  local base = getAddress(MODULE)
  kw2_inj = inj
  kw2_orig = readBytes(inj, STOLEN_LEN, true)
  kw2_cnt = allocateMemory(0x40, inj)
  kw2_tot = allocateMemory(0x40, inj)
  kw2_scratch = allocateMemory(0x40, inj)
  kw2_cave = allocateMemory(0x400, inj)
  kw2_buf = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (kw2_cnt and kw2_tot and kw2_scratch and kw2_cave and kw2_buf) then print("[KW2] echec alloc"); kw2_inj = nil; return end
  writeQword(kw2_cnt, 0); writeQword(kw2_tot, 0)
  for _, s in ipairs({ "kw2Buf", "kw2Cnt", "kw2Tot", "kw2Cave", "kw2Inj", "kw2Resolve", "kw2Obje", "kw2Scratch", "kw2Variant" }) do unregisterSymbol(s) end
  registerSymbol("kw2Buf", kw2_buf, true)
  registerSymbol("kw2Cnt", kw2_cnt, true)
  registerSymbol("kw2Tot", kw2_tot, true)
  registerSymbol("kw2Cave", kw2_cave, true)
  registerSymbol("kw2Inj", kw2_inj, true)
  registerSymbol("kw2Resolve", base + RESOLVE_RVA, true)
  registerSymbol("kw2Obje", base + OBJE_RVA, true)
  registerSymbol("kw2Scratch", kw2_scratch, true)
  registerSymbol("kw2Variant", kw2_scratch + 8, true)

  -- hook @0x673161 (AVANT la garde def-id). R13=event comp, RSI=param_2 ; EBX pas encore charge.
  -- On resout NOUS-MEMES : handle=[R13+0x538] -> FUN_140495abc(&handle)=entite-arme ->
  -- FUN_1405839d0(entite+0x2c,'obje')=record -> variant_name @ record+0x14. Les callees Win64
  -- preservent les non-volatils (R13/RSI/RBP/RDI). RSP=0 mod 16 ici -> sub 0x20 garde l'alignement.
  local asm = string.format([[
kw2Cave:
  inc qword ptr [kw2Tot]
  mov eax,[r13+538]             // handle d'arme
  mov [kw2Scratch],eax
  sub rsp,20
  lea rcx,[kw2Scratch]
  call kw2Resolve              // FUN_140495abc(&handle) -> RAX = entite-arme (0x358) ou 0
  test rax,rax
  jz kw2_novar
  lea rcx,[rax+2C]
  mov edx,6F626A65            // 'obje'
  call kw2Obje                // FUN_1405839d0(entite+0x2c,'obje') -> RAX = record ou 0
  test rax,rax
  jz kw2_novar
  mov ecx,[rax+14]          // variant_name (high-32 famille)
  mov [kw2Variant],ecx
  jmp kw2_have
kw2_novar:
  mov dword ptr [kw2Variant],0FFFFFFFF
kw2_have:
  add rsp,20
  mov rax,[kw2Cnt]
  cmp rax,%X
  jae kw2_done
  imul rdx,rax,%X
  mov r8,kw2Buf
  add r8,rdx
  mov ecx,[kw2Variant]        // [0] variant_name
  mov [r8+00],ecx
  mov ecx,[r13+538]           // [4] handle d'arme
  mov [r8+04],ecx
  xor rdx,rdx
kw2_cp:
  mov ecx,[rsi+rdx]          // [8..] event (tueur/victime) 0..EVT_LEN
  mov [r8+rdx+08],ecx
  add rdx,4
  cmp rdx,%X
  jb kw2_cp
  inc qword ptr [kw2Cnt]
kw2_done:
  mov ebx,[r13+538]           // octet vole : MOV EBX,[R13+0x538]
  jmp kw2Inj+07

kw2Inj:
  jmp kw2Cave
  nop
  nop
]], MAX_RECORDS, REC_SIZE, EVT_LEN)

  if autoAssemble(asm) then print(string.format("[KW2] capture ON @ %X (resolve=%X obje=%X)", inj, base + RESOLVE_RVA, base + OBJE_RVA))
  else print("[KW2] echec autoAssemble"); writeBytes(inj, kw2_orig); kw2_inj = nil end
end

function stopKW2()
  killTimer()
  if not kw2_inj then print("[KW2] pas actif"); return end
  writeBytes(kw2_inj, kw2_orig)
  print(string.format("[KW2] OFF -- kills=%s totalHits=%s", kw2_cnt and tostring(readQword(kw2_cnt)) or "?", kw2_tot and tostring(readQword(kw2_tot)) or "?"))
  kw2_inj = nil
end

function stopAllKW2() killTimer(); stopKW2() end

function kw2Status()
  print(string.format("[KW2] inj=%s kills=%s totalHits=%s",
    kw2_inj and string.format("%X", kw2_inj) or "nil",
    kw2_cnt and tostring(readQword(kw2_cnt)) or "nil",
    kw2_tot and tostring(readQword(kw2_tot)) or "nil"))
end

function dumpKW2(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\", "/") end
  local cnt = (kw2_buf and kw2_cnt) and readQword(kw2_cnt) or 0
  local tot = kw2_tot and readQword(kw2_tot) or 0
  if cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  local hdr = { "variantName", "weaponHandle" }
  for o = 0, EVT_LEN - 1, 4 do hdr[#hdr + 1] = string.format("e%02X", o) end
  local out = { string.format("# killweapon2 kills=%s totalHits=%s (variantName=high-32 famille WeaponIDToName)", tostring(cnt), tostring(tot)), table.concat(hdr, ",") }
  local nevt = EVT_LEN / 4
  for i = 0, cnt - 1 do
    local b = kw2_buf + i * REC_SIZE
    local row = { tostring(readInteger(b + 0, false)), tostring(readInteger(b + 4, false)) }
    for k = 0, nevt - 1 do row[#row + 1] = tostring(readInteger(b + 8 + k * 4, false)) end
    out[#out + 1] = table.concat(row, ",")
  end
  local f, err = io.open(path, "w")
  if not f then print("[KW2] ouverture IMPOSSIBLE: " .. tostring(err)); return end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[KW2] %d kills ecrits (totalHits=%d) -> %s", cnt, tot, path))
end

function captureKW2(target)
  target = target or 12
  startKW2()
  if not kw2_inj then print("[KW2] demarrage echoue -> repairKW2() puis reessaie"); return end
  print(string.format("[KW2] >>> JOUE le film (lecture qui AVANCE) -- arret a %d kills ou 120s <<<", target))
  local ticks = 0
  kw2_timer = createTimer(nil)
  kw2_timer.Interval = 1000
  kw2_timer.OnTimer = function()
    ticks = ticks + 1
    local cnt = kw2_cnt and readQword(kw2_cnt) or 0
    print(string.format("[KW2] ... kills=%d (t=%ds)", cnt, ticks))
    if cnt >= target or ticks >= 120 then killTimer(); stopKW2(); dumpKW2(); print("[KW2] termine. Donne le CSV.") end
  end
end

print("[KW2] charge. captureKW2() puis JOUE le film. stopAllKW2() pour couper.")
