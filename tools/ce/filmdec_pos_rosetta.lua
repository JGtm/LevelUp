--[==[==========================================================================
  filmdec_pos_rosetta.lua
  LevelUp / Halo Infinite -- PIERRE DE ROSETTE position (offset film <-> valeur)

  BUT
    Comme filmdec_pos_capture, mais capture EN PLUS une SIGNATURE d'octets bruts
    (16 o) lus par le jeu depuis le buffer du film ([bitreader+0x40] = pointeur
    d'octet courant dans le chunk INFLATE en memoire). Offline, on retrouve cette
    signature dans les chunks telecharges (apres inflate) -> on obtient l'OFFSET
    FILM EXACT de chaque position biped i0. C'est la pierre de Rosette :
        (offset film) <-> (slot biped) <-> (x,y,z)
    -> resout l'ambiguite d'ancrage / de seed du decodeur offline.

  HOOK (identique au site prouve, AOB inchange) : 14076cd11 dispatch composants.
    rsi=record ([rsi+30]=typeIndex, [rsi+34]=eid) ; rdi=bitreader ([rdi+2C]=curseur,
    [rdi+40]=pointeur d'octet dans le film) ; r15d=compIndex ; r8=param_3 (obj=[r8+10]).

  RECORD 40 o : eid(4) curseur(4) x(4) y(4) z(4) sig[16] pad(4).

  USAGE : dofile ce fichier ; captureFilmdecRosetta(20000) ; JOUE le film.
    -> dump .ai/V7.5/dumps/ce_pos_rosetta.csv (eid,slot,bitCursor,x,y,z,sighex)
==========================================================================]==]

local AOB         = "44 89 6C 24 20 48 8B CB FF 50 28"
local MODULE      = "HaloInfinite.exe"
local STOLEN_LEN  = 8
local REC_SIZE    = 40
local MAX_RECORDS = 0x100000  -- 1048576 records (x40 = 40 Mo)

fdr_inj  = fdr_inj  or nil
fdr_orig = fdr_orig or nil
fdr_buf  = fdr_buf  or nil
fdr_cnt  = fdr_cnt  or nil
fdr_tot  = fdr_tot  or nil
fdr_cave = fdr_cave or nil

local function moduleRange()
  local base = getAddress(MODULE)
  local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

local function findUniqueInModule(pattern)
  pattern = pattern or AOB
  local base, size = moduleRange()
  if not base then print("[FDR] module introuvable (process attache ?)"); return nil end
  local ms = AOBScan(pattern)
  if ms == nil or ms.Count == 0 then
    print("[FDR] AOB introuvable -> repairFilmdecRosetta() si patch residuel.")
    if ms then ms.destroy() end
    return nil
  end
  local hit, n = nil, 0
  for i = 0, ms.Count - 1 do
    local a = tonumber(ms[i], 16)
    if a and a >= base and a < base + size then hit = a; n = n + 1 end
  end
  ms.destroy()
  if n ~= 1 then print(string.format("[FDR] attendu 1 occurrence, trouve %d -- abandon", n)); return nil end
  return hit
end

function startFilmdecRosetta()
  if fdr_inj then stopFilmdecRosetta() end
  local inj = findUniqueInModule()
  if not inj then return end
  fdr_inj  = inj
  fdr_orig = readBytes(inj, STOLEN_LEN, true)
  fdr_cnt  = allocateMemory(0x40, inj)
  fdr_tot  = allocateMemory(0x40, inj)
  fdr_cave = allocateMemory(0x400, inj)
  fdr_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (fdr_cnt and fdr_tot and fdr_cave and fdr_buf) then
    print("[FDR] echec allocation"); fdr_inj = nil; return
  end
  writeQword(fdr_cnt, 0); writeQword(fdr_tot, 0)
  for _, s in ipairs({"fdrBuf","fdrCnt","fdrTot","fdrCave","fdrInj"}) do unregisterSymbol(s) end
  registerSymbol("fdrBuf",  fdr_buf,  true)
  registerSymbol("fdrCnt",  fdr_cnt,  true)
  registerSymbol("fdrTot",  fdr_tot,  true)
  registerSymbol("fdrCave", fdr_cave, true)
  registerSymbol("fdrInj",  fdr_inj,  true)

  local asm = string.format([[
fdrCave:
  inc qword ptr [fdrTot]
  push rax
  push rbx
  push rcx
  push rdx
  mov eax,[rsi+30]              // typeIndex
  cmp eax,23                    // 35 (biped)
  jne fdr_pop
  test r15d,r15d               // compIndex == 0 (i0) ?
  jnz fdr_pop
  mov rax,[fdrCnt]
  cmp rax,%X                    // MAX_RECORDS
  jae fdr_pop
  mov rcx,[r8+10]              // obj = *(param_3+0x10)
  test rcx,rcx
  jz fdr_pop
  imul rdx,rax,%X              // * REC_SIZE (40)
  mov rbx,fdrBuf
  add rbx,rdx
  mov edx,[rsi+34]            // [+00] eid
  mov [rbx+00],edx
  mov edx,[rdi+2C]           // [+04] curseur bit
  mov [rbx+04],edx
  mov edx,[rcx+04]           // [+08] x
  mov [rbx+08],edx
  mov edx,[rcx+08]           // [+0C] y
  mov [rbx+0C],edx
  mov edx,[rcx+0C]           // [+10] z
  mov [rbx+10],edx
  mov rcx,[rdi+40]           // pointeur d'octet dans le buffer film
  test rcx,rcx
  jz fdr_nosig
  mov rax,[rcx]              // [+14] signature octets 0..7
  mov [rbx+14],rax
  mov rax,[rcx+8]            // [+1C] signature octets 8..15
  mov [rbx+1C],rax
fdr_nosig:
  mov dword ptr [rbx+24],0    // [+24] pad
  inc qword ptr [fdrCnt]
fdr_pop:
  pop rdx
  pop rcx
  pop rbx
  pop rax
fdr_re:
  mov [rsp+20],r13d
  mov rcx,rbx
  jmp fdrInj+%X

fdrInj:
  jmp fdrCave
  nop
  nop
  nop
]], MAX_RECORDS, REC_SIZE, STOLEN_LEN)

  if autoAssemble(asm) then
    print(string.format("[FDR] capture ROSETTE ON @ %X (buffer %d records)", inj, MAX_RECORDS))
  else
    print("[FDR] echec autoAssemble -- restauration")
    if fdr_orig then writeBytes(inj, fdr_orig) end
    fdr_inj = nil
  end
end

function stopFilmdecRosetta()
  if not fdr_inj then print("[FDR] pas actif"); return end
  if fdr_orig then writeBytes(fdr_inj, fdr_orig) end
  local n = fdr_cnt and readQword(fdr_cnt) or 0
  print(string.format("[FDR] capture OFF -- %d records", n))
  fdr_inj = nil
end

function repairFilmdecRosetta()
  local inj = findUniqueInModule("E9 ?? ?? ?? ?? 90 90 90 FF 50 28")
  if not inj then print("[FDR] aucun patch residuel ; sinon redemarre Halo."); return end
  writeBytes(inj, { 0x44, 0x89, 0x6C, 0x24, 0x20, 0x48, 0x8B, 0xCB })
  fdr_inj = nil
  print(string.format("[FDR] patch restaure @ %X", inj))
end

local function defaultDumpPath()
  return "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_rosetta.csv"
end

function filmdecRosettaStatus()
  print(string.format("[FDR] inj=%s tot=%s cnt=%s",
    fdr_inj and string.format("%X",fdr_inj) or "nil",
    fdr_tot and tostring(readQword(fdr_tot)) or "nil",
    fdr_cnt and tostring(readQword(fdr_cnt)) or "nil"))
end

function dumpFilmdecRosetta(path)
  path = path or defaultDumpPath()
  if type(path) == "string" then path = path:gsub("\\","/") end
  local cnt = (fdr_buf and fdr_cnt) and readQword(fdr_cnt) or 0
  if cnt and cnt > MAX_RECORDS then cnt = MAX_RECORDS end
  local out = { "# filmdec_pos_rosetta records="..tostring(cnt), "eid,slot,bitCursor,x,y,z,sighex" }
  for i = 0, (cnt or 0) - 1 do
    local b = fdr_buf + i * REC_SIZE
    local eid = readInteger(b+0) & 0xFFFFFFFF
    local slot = eid & 0x3FFFFFFF
    local sig = readBytes(b+0x14, 16, true)
    local hx = {}
    for j=1,16 do hx[j] = string.format("%02x", sig[j] or 0) end
    out[#out+1] = string.format("%d,%d,%d,%.4f,%.4f,%.4f,%s",
      eid, slot, readInteger(b+4), readFloat(b+8), readFloat(b+12), readFloat(b+16), table.concat(hx))
  end
  local f, err = io.open(path, "w")
  if not f then print("[FDR] ouverture IMPOSSIBLE: "..tostring(err)); return end
  f:write(table.concat(out, "\n")); f:write("\n"); f:close()
  print(string.format("[FDR] %d lignes -> %s", cnt or 0, path))
end

function captureFilmdecRosetta(target, path)
  target = target or 20000
  path = path or defaultDumpPath()
  startFilmdecRosetta()
  if not fdr_inj then print("[FDR] demarrage echoue -> repairFilmdecRosetta()"); return end
  print(string.format("[FDR] >>> JOUE le film (j'attends %d records) <<<", target))
  local ticks = 0
  local t = createTimer(nil); t.Interval = 1000
  t.OnTimer = function(timer)
    ticks = ticks + 1
    local cnt = fdr_cnt and readQword(fdr_cnt) or 0
    print(string.format("[FDR] ... %d records (t=%ds)", cnt, ticks))
    if cnt >= target or ticks >= 90 then
      timer.destroy(); stopFilmdecRosetta(); dumpFilmdecRosetta(path)
      print("[FDR] termine.")
    end
  end
end

print("[FDR] charge. captureFilmdecRosetta(20000) puis JOUE le film.")
print("[FDR] Manuel : startFilmdecRosetta / filmdecRosettaStatus / stopFilmdecRosetta / dumpFilmdecRosetta")
