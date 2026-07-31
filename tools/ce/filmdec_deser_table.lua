--[==[==========================================================================
  filmdec_deser_table.lua
  LevelUp / Halo Infinite -- LA TABLE UNIVERSELLE DES DESERIALISEURS

  POURQUOI CE SCRIPT EXISTE, ET POURQUOI IL EST URGENT

  Le mapping `typeIndex -> descripteur -> deserialiseur par composant` est la
  SEULE donnee du decodage de film qui ne soit ni dans le film, ni derivable de
  l'executable statique : elle est batie a l'initialisation du jeu dans
  DAT_144e61d88, et vaut 0 tant que le jeu n'a pas demarre. Elle se lit donc sur
  un jeu LANCE, et uniquement la.

  Jusqu'ici elle avait ete lue POINTEUR PAR POINTEUR via le MCP Cheat Engine,
  pour le seul archetype biped #35 (64 composants). Cette methode ne passe pas a
  l'echelle : il y a environ 1067 composants repartis sur une cinquantaine
  d'archetypes d'objets.

  CE QU'IL RESTE A COUVRIR, mesure en croisant le registre du film avec le
  dispatch du decodeur Go :

      ti=35  biped         62/64   quasi complet
      ti=37  objets au sol 31/31   complet
      ti=41  projectiles   22/22   complet
      ti=42  armes au sol  21/21   complet
      ti=40  vehicules     32/48   TROU
      ti=43  dispositifs   18/41   TROU
      ti=23  zones          0/33   TROU TOTAL
      ti=11  objectifs      0/34   TROU TOTAL  <- ce que le chantier veut lire

  Les deux derniers sont exactement les cibles annoncees : savoir qui porte le
  drapeau, ou en est une capture, qui tient le crane. Sans cette table, aucun de
  ces composants ne sera lisible — et sans Cheat Engine, la table est hors
  d'atteinte POUR TOUJOURS sur ce build.

  CE SCRIPT LES PREND TOUS D'UN COUP, en une passe, sans charger de film.

  ----------------------------------------------------------------------------
  STRUCTURE LUE (etablie section 5 de RECETTE_DECODAGE_FILM_CHUNKS.md)

      REG          = *(DAT_144e61d88)                  ; 0 si l'ECS n'est pas peuple
      descripteur  = *(REG + 8 + typeIndex*8)          ; 174 typeIndex valides
      composants   = descripteur + 8                   ; tableau de pointeurs
      count        = *(int*)(descripteur + 8 + 0x4320)
      compDesc[i]  = *(composants + i*8)
      deser[i]     = *( *compDesc[i] + 0x28 )
                     SI deser[i] == thunk FUN_14076ce9c  ->  prendre vtable[0x30]

  Le thunk est un `jmp vtable[0x30]` : quand [0x28] pointe dessus, le vrai
  deserialiseur est en [0x30]. Ne pas le suivre produirait 1067 fois la meme
  adresse inutile.

  ----------------------------------------------------------------------------
  MODE D'EMPLOI

  1. Lancer Halo Infinite et entrer dans N'IMPORTE QUEL film en mode Theatre.
     La carte, le mode et le film n'ont aucune importance : la table est une
     CONSTANTE DU BUILD, identique pour tous.
     ATTENTION : elle vaut 0 tant que l'ECS n'est pas peuple. Etre dans un film,
     pas seulement dans le menu.
  2. Cheat Engine : ouvrir le processus HaloInfinite.exe.
  3. Table -> Show Cheat Table Lua Script -> coller ce fichier -> Execute.
  4. La sortie va dans SORTIE ci-dessous, au format TSV. Elle se relit en Go
     sans parseur dedie.

  DUREE ATTENDUE : quelques secondes. La lecture est purement memoire, il n'y a
  ni hook, ni cave, ni interception de flux — donc aucun risque pour le jeu.

  ----------------------------------------------------------------------------
  CE QUE LA SORTIE CONTIENT, ET COMMENT LA RELIRE

  Une ligne par (typeIndex, composant), colonnes separees par des tabulations :

      typeIndex  compIndex  deserStatique  viaThunk  descripteur  compDesc

  `deserStatique` est deja reconverti en adresse GHIDRA (statique) par
      static = runtime - base + 0x140000000
  donc il se cherche tel quel dans le projet Ghidra, sans reconversion.

  `viaThunk` vaut 1 quand la valeur vient de vtable[0x30] apres detection du
  thunk. C'est une information de tracabilite : si un jour une adresse parait
  fausse, c'est le premier endroit ou regarder.

  UNE ADRESSE A ZERO N'EST PAS UNE ERREUR : elle veut dire que le composant
  n'a pas de deserialiseur a cet index. On la publie telle quelle plutot que de
  la masquer — un trou tu se confondrait avec une lecture ratee.
==========================================================================]==]

local MODULE      = "HaloInfinite.exe"
local DAT_STATIC  = 0x144e61d88   -- adresse Ghidra de la variable de registre
local IMAGE_BASE  = 0x140000000   -- base statique de l'image dans Ghidra
local THUNK_STATIC= 0x14076ce9c   -- FUN_14076ce9c : jmp vtable[0x30]
local COUNT_OFF   = 0x4320        -- offset du compteur de composants
local MAX_TYPEIDX = 200           -- 174 valides mesures ; on balaye large
local MAX_COMPS   = 512           -- garde-fou : aucun archetype connu n'en a autant
local SORTIE      = [[C:\Users\Guillaume\Downloads\deser_table.tsv]]

-- lisRuntime : adresse Ghidra -> adresse runtime (ASLR).
local base = getAddress(MODULE)
if base == nil or base == 0 then
  print("ECHEC : module " .. MODULE .. " introuvable. Le jeu est-il ouvert dans Cheat Engine ?")
  return
end
local function versRuntime(statique) return statique - IMAGE_BASE + base end
local function versStatique(runtime)  return runtime  - base + IMAGE_BASE end

local REG = readQword(versRuntime(DAT_STATIC))
if REG == nil or REG == 0 then
  print("ECHEC : le registre ECS vaut 0.")
  print("  La table n'est batie qu'a l'initialisation du jeu : il faut etre DANS un film")
  print("  en mode Theatre, pas seulement dans le menu.")
  return
end
print(string.format("registre ECS = 0x%X  (base module 0x%X)", REG, base))

local THUNK_RT = versRuntime(THUNK_STATIC)
local f = io.open(SORTIE, "w")
if f == nil then
  print("ECHEC : impossible d'ecrire " .. SORTIE)
  return
end
f:write("typeIndex\tcompIndex\tdeserStatique\tviaThunk\tdescripteur\tcompDesc\n")

local nbTypes, nbComps, nbThunks, nbZeros = 0, 0, 0, 0

for ti = 0, MAX_TYPEIDX - 1 do
  local desc = readQword(REG + 8 + ti * 8)
  if desc ~= nil and desc ~= 0 then
    local count = readInteger(desc + 8 + COUNT_OFF)
    -- Un compte absurde signale qu'on lit hors structure : on saute plutot que
    -- de deverser des milliers de lignes de bruit.
    if count ~= nil and count > 0 and count <= MAX_COMPS then
      nbTypes = nbTypes + 1
      local composants = desc + 8
      for ci = 0, count - 1 do
        local compDesc = readQword(composants + ci * 8)
        local deser, viaThunk = 0, 0
        if compDesc ~= nil and compDesc ~= 0 then
          local vtable = readQword(compDesc)
          if vtable ~= nil and vtable ~= 0 then
            deser = readQword(vtable + 0x28) or 0
            -- LE THUNK : [0x28] pointe un `jmp vtable[0x30]`. Sans cette
            -- detection, tous les composants concernes rendraient la meme
            -- adresse, celle du thunk, et la table serait inexploitable.
            if deser == THUNK_RT then
              deser = readQword(vtable + 0x30) or 0
              viaThunk = 1
              nbThunks = nbThunks + 1
            end
          end
        end
        if deser == 0 then nbZeros = nbZeros + 1 end
        f:write(string.format("%d\t%d\t0x%X\t%d\t0x%X\t0x%X\n",
          ti, ci,
          deser ~= 0 and versStatique(deser) or 0,
          viaThunk,
          versStatique(desc),
          compDesc ~= nil and compDesc ~= 0 and versStatique(compDesc) or 0))
        nbComps = nbComps + 1
      end
    end
  end
end

f:close()
print(string.format("ECRIT : %s", SORTIE))
print(string.format("  archetypes valides : %d", nbTypes))
print(string.format("  composants         : %d", nbComps))
print(string.format("  via thunk          : %d", nbThunks))
print(string.format("  sans deserialiseur : %d  (ce n'est pas une erreur)", nbZeros))
print("")
print("CONTROLE A FAIRE AVANT DE SE FIER A LA SORTIE :")
print("  l'archetype 35 doit rendre 64 composants, et son vtable[0x60] live vaut")
print("  0x140F44C38 — c'est le port deja valide. Si ce couple ne tombe pas,")
print("  la structure a bouge et la table entiere est suspecte.")
