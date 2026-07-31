--[==[==========================================================================
  filmdec_full_capture.lua
  LevelUp / Halo Infinite -- CAPTURE LARGE + PIERRE DE ROSETTE

  CE QUE CE SCRIPT AJOUTE AUX DEUX EXISTANTS
    filmdec_delta_capture  : journalise TOUS les composants de TOUS les archetypes,
                             mais SANS ancrage dans le film -> on ne peut pas confronter
                             la capture au decodage hors ligne du MEME flux.
    filmdec_pos_rosetta    : ancre chaque hit par une signature d'octets bruts, mais ne
                             garde QUE i0 du bipede.
    CE SCRIPT              : les deux a la fois. Tous les composants, tous les archetypes,
                             chacun ancre par sa signature.

  POURQUOI C'EST DECISIF (mesure du 2026-07-26)
    La capture actuelle (807 855 lignes) vient d'un AUTRE film que ceux qu'on decode : on
    ne peut comparer que des DISTRIBUTIONS de largeurs, jamais un record en face d'un
    record. Or le probleme n'est plus une largeur : la verite dit i22 present dans 0,19 %
    des records de bipede, nous le lisons dans 12 % -- un exces de 63 fois, qu'aucune
    correction de largeur ne peut produire. C'est le MASQUE qui est faux, c'est-a-dire la
    LISTE des composants qu'on croit presents.
    Le vrai masque EST dans cette capture : c'est l'ensemble des compIndex reellement
    dispatches pour un eid donne. Avec la signature pour aligner, la comparaison devient
    directe et le premier composant invente saute aux yeux.

  CE QUE LA CAPTURE COUVRE (tout le Tier 3 de RECAP_STATS_EXPLOITABLES)
    munitions i30/33/36/39 · chargeur i31/34/37/40 · surchauffe i32/35/38/41
    grenades i22 (comptes) et i47 (jeu de types)
    capacite i48 (jeu) · i56 (energie) · i57 · i59 (etat non predit)
    arme desiree i42 · arme en main i43..i46
    sante i04 · bouclier i05 · maxima i13 · velocite i01/i03 · visee i02/i21
    mobilite i54 · glissade i62 · mort i11 · position i0
    Aucun filtre : on capture tout et on trie hors ligne.

  HOOK (site prouve, AOB inchange depuis les captures precedentes)
    14076cd11, dispatch des composants.
    rsi = record   : [rsi+30] typeIndex, [rsi+34] eid
    rdi = bitreader: [rdi+2C] curseur en bits, [rdi+40] pointeur d'octet dans le film
    r15d = compIndex · r13d = param_4 · r14d = compteur de skips

  RECORD 40 octets
    +00 eid(4) +04 typeIndex(4) +08 compIndex(4) +0C param4(4)
    +10 bitCursor(4) +14 skipCount(4) +18 signature 16 octets

  MODE D'EMPLOI
    1. Lancer Halo Infinite SUR UN COMPTE HORS LIGNE, sans anti-triche.
    2. Attacher Cheat Engine au process.
    3. Table -> Show Cheat Table Lua Script -> coller ce fichier -> Execute.
    4. Taper :  startFilmdecFull()
    5. REJOUER EN THEATER un film QUI EST DEJA DANS NOTRE CACHE. C'est la condition
       essentielle : sans le meme film des deux cotes, l'ancrage ne sert a rien.
       Candidats mesures presents dans data/cache/film_chunks : 4f77afc1, fccc61cd,
       78919882, de7c1986, 44e14331, b955bf2a.
       Laisser tourner le film EN ENTIER, vitesse normale.
    6. Taper :  dumpFilmdecFull()
       -> ecrit ~/Downloads/filmdec_full.csv
    7. Taper :  stopFilmdecFull()      (restaure le code du jeu)

  VERIFICATION IMMEDIATE, avant de quitter le jeu
    statusFilmdecFull() doit montrer un total > 0. S'il vaut 0, le hook ne s'execute pas
    (mauvais chemin de code, ou le film n'est pas en train d'etre decode) et il ne sert a
    rien de dumper.
==========================================================================]==]

local AOB         = "44 89 6C 24 20 48 8B CB FF 50 28"
local MODULE      = "HaloInfinite.exe"
local STOLEN_LEN  = 8
local REC_SIZE    = 40
-- 1,14 M hits mesures sur un film entier ; on double la marge. 2 M x 40 o = 80 Mo.
local MAX_RECORDS = 0x200000

ffc_inj  = ffc_inj  or nil
ffc_orig = ffc_orig or nil
ffc_buf  = ffc_buf  or nil
ffc_cnt  = ffc_cnt  or nil
ffc_tot  = ffc_tot  or nil
ffc_cave = ffc_cave or nil

local function moduleRange()
  local base = getAddress(MODULE)
  local size = getModuleSize(MODULE)
  if not base or base == 0 then return nil end
  return base, (size and size > 0) and size or 0x8000000
end

-- findUnique : l'AOB DOIT etre unique dans le module. Plusieurs occurrences = on ne sait
-- pas ou on injecte, et une injection au mauvais endroit corrompt le jeu.
local function findUnique()
  local base, size = moduleRange()
  if not base then
    print("[FFC] module introuvable -- le process est-il attache ?")
    return nil
  end
  local ms = AOBScan(AOB)
  if ms == nil or ms.Count == 0 then
    print("[FFC] AOB introuvable. Si une capture precedente a mal fini, lancer")
    print("[FFC] repairFilmdecFull() puis reessayer ; sinon redemarrer le jeu.")
    if ms then ms.destroy() end
    return nil
  end
  local hits = {}
  for i = 0, ms.Count - 1 do
    local a = getAddress(ms[i])
    if a >= base and a < base + size then hits[#hits + 1] = a end
  end
  ms.destroy()
  if #hits ~= 1 then
    print(string.format("[FFC] %d occurrences dans le module -- ambigu, on n'injecte pas.", #hits))
    return nil
  end
  return hits[1]
end

function startFilmdecFull()
  if ffc_inj then print("[FFC] capture deja active.") return end
  local inj = findUnique()
  if not inj then return end

  ffc_inj  = inj
  ffc_orig = readBytes(inj, STOLEN_LEN, true)
  ffc_cnt  = allocateMemory(0x40, inj)
  ffc_tot  = allocateMemory(0x40, inj)
  ffc_cave = allocateMemory(0x400, inj)
  ffc_buf  = allocateMemory(REC_SIZE * MAX_RECORDS)
  if not (ffc_cnt and ffc_tot and ffc_cave and ffc_buf) then
    print("[FFC] allocation impossible -- pas assez de memoire contigue.")
    ffc_inj = nil
    return
  end
  writeQword(ffc_cnt, 0)
  writeQword(ffc_tot, 0)

  for _, s in ipairs({ "ffcBuf", "ffcCnt", "ffcTot", "ffcCave", "ffcInj" }) do unregisterSymbol(s) end
  registerSymbol("ffcBuf",  ffc_buf,  true)
  registerSymbol("ffcCnt",  ffc_cnt,  true)
  registerSymbol("ffcTot",  ffc_tot,  true)
  registerSymbol("ffcCave", ffc_cave, true)
  registerSymbol("ffcInj",  ffc_inj,  true)

  -- AUCUN FILTRE d'archetype : on capture tout, on trie hors ligne. Le compteur total est
  -- incremente AVANT toute condition, pour distinguer "le hook ne tourne pas" de "le hook
  -- tourne mais le filtre rejette tout".
  local asm = string.format([[
ffcCave:
  inc qword ptr [ffcTot]
  push rax
  push rbx
  push rcx
  push rdx
  mov rax,[ffcCnt]
  cmp rax,%X
  jae ffc_pop
  imul rdx,rax,%X
  mov rbx,ffcBuf
  add rbx,rdx
  mov edx,[rsi+34]
  mov [rbx+00],edx            // eid
  mov edx,[rsi+30]
  mov [rbx+04],edx            // typeIndex
  mov [rbx+08],r15d           // compIndex
  mov [rbx+0C],r13d           // param_4
  mov edx,[rdi+2C]
  mov [rbx+10],edx            // curseur en bits
  mov [rbx+14],r14d           // skips
  mov rcx,[rdi+40]            // pointeur d'octet dans le film inflate
  test rcx,rcx
  jz ffc_nosig
  mov rax,[rcx]
  mov [rbx+18],rax            // signature octets 0..7
  mov rax,[rcx+8]
  mov [rbx+20],rax            // signature octets 8..15
  jmp ffc_done
ffc_nosig:
  mov qword ptr [rbx+18],0
  mov qword ptr [rbx+20],0
ffc_done:
  inc qword ptr [ffcCnt]
ffc_pop:
  pop rdx
  pop rcx
  pop rbx
  pop rax
ffc_re:
  mov [rsp+20],r13d
  mov rcx,rbx
  jmp ffcInj+%X

ffcInj:
  jmp ffcCave
  nop
  nop
  nop
]], MAX_RECORDS, REC_SIZE, STOLEN_LEN)

  if autoAssemble(asm) then
    print(string.format("[FFC] capture LARGE ON @ %X (buffer %d records de %d o)",
      inj, MAX_RECORDS, REC_SIZE))
    print("[FFC] REJOUE MAINTENANT un film DEJA EN CACHE, en entier, vitesse normale.")
    print("[FFC] puis : statusFilmdecFull() -> dumpFilmdecFull() -> stopFilmdecFull()")
  else
    print("[FFC] echec autoAssemble -- restauration du code original.")
    if ffc_orig then writeBytes(inj, ffc_orig) end
    ffc_inj = nil
  end
end

function statusFilmdecFull()
  if not ffc_cnt then print("[FFC] capture inactive.") return end
  local n, t = readQword(ffc_cnt), readQword(ffc_tot)
  print(string.format("[FFC] %d records ecrits / %d hits totaux", n or 0, t or 0))
  if (t or 0) == 0 then
    print("[FFC] ZERO hit : le hook ne s'execute pas. Le film est-il en cours de lecture ?")
  elseif n and n >= MAX_RECORDS then
    print("[FFC] BUFFER PLEIN : la fin du film n'est PAS capturee. Augmenter MAX_RECORDS.")
  end
end

function stopFilmdecFull()
  if not ffc_inj then print("[FFC] capture inactive.") return end
  if ffc_orig then writeBytes(ffc_inj, ffc_orig) end
  print("[FFC] code du jeu restaure. Le buffer reste lisible jusqu'a la fermeture de CE.")
  ffc_inj = nil
end

-- repairFilmdecFull : a lancer si startFilmdecFull dit "AOB introuvable" alors qu'aucune
-- capture n'est active -- signe qu'un patch residuel d'une session precedente traine.
function repairFilmdecFull()
  local base, size = moduleRange()
  if not base then print("[FFC] module introuvable.") return end
  local ms = AOBScan("E9 ?? ?? ?? ?? 90 90 90 48 8B CB FF 50 28")
  if ms == nil or ms.Count == 0 then
    print("[FFC] aucun patch residuel trouve.")
    if ms then ms.destroy() end
    return
  end
  for i = 0, ms.Count - 1 do
    local a = getAddress(ms[i])
    writeBytes(a, 0x44, 0x89, 0x6C, 0x24, 0x20, 0x48, 0x8B, 0xCB)
    print(string.format("[FFC] patch restaure @ %X", a))
  end
  ms.destroy()
end

local function defaultPath()
  local home = os.getenv("USERPROFILE") or os.getenv("HOME") or "."
  return home .. "/Downloads/filmdec_full.csv"
end

function dumpFilmdecFull(path)
  if not ffc_buf then print("[FFC] rien a dumper.") return end
  path = path or defaultPath()
  local n = readQword(ffc_cnt) or 0
  if n > MAX_RECORDS then n = MAX_RECORDS end
  if n == 0 then print("[FFC] 0 record -- rien n'a ete capture.") return end
  local f, err = io.open(path, "w")
  if not f then print("[FFC] ecriture impossible : " .. tostring(err)) return end
  f:write("# filmdec full capture -- hits=", tostring(readQword(ffc_tot) or 0),
          " ecrits=", tostring(n), "\n")
  f:write("eid,typeIndex,compIndex,param4,bitCursor,skipCount,sighex\n")
  local out = {}
  for i = 0, n - 1 do
    local b = ffc_buf + i * REC_SIZE
    local sig = readBytes(b + 0x18, 16, true) or {}
    local hex = {}
    for k = 1, 16 do hex[k] = string.format("%02x", sig[k] or 0) end
    out[#out + 1] = string.format("%d,%d,%d,%d,%d,%d,%s",
      readInteger(b) or 0, readInteger(b + 4) or 0, readInteger(b + 8) or 0,
      readInteger(b + 0xC) or 0, readInteger(b + 0x10) or 0, readInteger(b + 0x14) or 0,
      table.concat(hex))
    if #out >= 4096 then f:write(table.concat(out, "\n"), "\n"); out = {} end
  end
  if #out > 0 then f:write(table.concat(out, "\n"), "\n") end
  f:close()
  print(string.format("[FFC] %d records ecrits dans %s", n, path))
  print("[FFC] a deposer dans .ai/V7.5/dumps/ du worktree filmdec-continuation.")
end

print("[FFC] charge. Sequence : startFilmdecFull() -> rejouer un film EN CACHE ->")
print("[FFC] statusFilmdecFull() -> dumpFilmdecFull() -> stopFilmdecFull()")
