# SONDE — l'activation des socles par MODE vit-elle dans l'asset UgcGameVariants ?

> Lot de MESURE, 2026-08-19. Aucune production. Branche `wt/variant-sonde`, worktree frere
> `LevelUp-wt-variant-sonde`, base `feat/v75` = `b1f5f4188`.
> Contrat : `.claude/skills/plan-execution/SKILL.md`. Derogation ORDONNEE par la consigne :
> ni entree `.ai/thought_log.md` ni entree au registre — les textes partent au CR.

## 1. La question de l'utilisateur

« Concretement, pour le rejeu 100 % offline sur mon VPS, comment avoir les scenarios / tous
les socles ACTIVES du mode ? »

Le lot `PLAN_SOCLES_MVAR.md` a repondu a la moitie : **le fichier de carte POSE les socles,
au centimetre** (32/32 apparies, mediane 0,01 m). Il a laisse l'autre moitie ouverte :
**le mode ALLUME** — Cliffhanger porte 17 poses au fichier, 10 actives en CTF, **0 en Super
Fiesta**. Rien dans le `.mvar` des cartes DEV ne dit lesquelles.

Cette sonde teste UNE hypothese pour combler ce trou : l'activation vit-elle dans l'asset
`UgcGameVariants` du mode, recuperable au SYNCHRO (jamais au rejeu) ?

## 2. Architecture proposee (c'est ELLE que la sonde valide ou refute)

Le rejeu reste **100 % offline**. Le reseau ne sert qu'a la SYNCHRO, exactement comme
aujourd'hui pour les matchs, les stats et les noms d'assets :

1. chaque match du registre porte DEJA `game_variant_id` + `game_variant_version_id`
   (`internal/openspartan/mapper/mapper.go:99-101`, colonnes du `match_registry`) ;
2. l'asset `UgcGameVariants/{assetId}/versions/{versionId}` se recupere UNE fois par variant
   DISTINCT via le client discovery existant ;
3. il se versionne en reference locale (patron `asset_translations` / `map_objectives.json`) ;
4. le VPS lit la reference locale — **jamais le reseau au moment du rejeu**.

## 3. Acquis, verifies sur pieces le 2026-08-19

| Fait | Piece |
|---|---|
| `AssetTypeGameVariant -> "ugcGameVariants"` | `internal/platform/halo/discovery_types.go:21-25` |
| `FetchAsset` = `GET {host}/hi/{segment}/{id}/versions/{ver}`, Spartan + 343-clearance | `internal/platform/halo/discovery_client.go:50-100` |
| L'asset expose `Files.Prefix` + `Files.FileRelativePaths` (les blobs) | idem, struct `raw` |
| **Les blobs sont en lecture ANONYME** (`blobs-infiniteugc`), le 401 ne vaut que pour discovery | commentaire mesure 2026-07-25, `discovery_client.go:44-49` |
| Patron d'appel COMPLET deja ecrit (auth store -> asset -> blob) | `cmd/mapobj-build/auth.go`, `cmd/mapobj-build/fetch.go` |
| Auth = `RefreshHaloTokensViaStoreFirst` sur `MultiUserTokenStore`, zero re-capture | `cmd/mapobj-build/auth.go:47-56` |
| `versionID` vide -> endpoint sans `/versions/` (derniere version publiee) | `cmd/mapobj-build/fetch.go:66-70` |
| Les 4 matchs temoins sont dans l'instantane parquet du 2026-07-11 | requete ci-dessous |
| 3 `type_id` de socle | `1597478195`=`0x5F379533`, `1649659840`=`0x6253CFC0`, `1585893648`=`0x5E86D110` |
| Labels de mode vus sur Forge | `stockpile_include`, `stockpile_exclude`, `infection_exclude`, `ctf_multi_exclude` |
| 9 hashs de label non resolus sur Catalyst (le plus frequent `-886053664`, 18 fois) ; `-831896525` sur Smallhalla | `PLAN_SOCLES_MVAR.md` sections 7 et 9 |
| Un serveur tient `:8000` (server.exe 26984) | `netstat` — **lecture par le parquet UNIQUEMENT**, aucune ouverture de `shared_matches_v2.duckdb` |

### Les 4 temoins (parquet `match_registry_20260711_090652.parquet`)

| Match | Carte | Variant | `game_variant_id` | `version_id` | Attendu |
|---|---|---|---|---|---|
| `bcb6d393` | Cliffhanger | CTF:Arena | `8650f7e0-1f82-4d45-a127-32dd54df06e5` | **NULL** | 10 socles actifs |
| `000d5950` | Cliffhanger | Slayer:Arena Super Fiesta | `a5d824f0-c064-4b40-951f-8b6b3005ccf6` | **NULL** | **0 actif** |
| `01e1f945` | Catalyst | KOTH:Arena | `373f3d27-cb4c-4d7b-b6c9-7757de3c1133` | **NULL** | power-up = SURBOUCLIER |
| `64e8adfa` | Catalyst | CTF:Arena | `8650f7e0-1f82-4d45-a127-32dd54df06e5` | **NULL** | pas de power-up |

Deux paires discriminantes : **meme carte, deux variants**, des deux cotes.

### L'acquis qui contraint TOUT le reste (mesure de cadrage, avant toute requete reseau)

`CTF:Arena` (`8650f7e0`) sert **31 cartes distinctes** dans le registre ; `Team Slayer:Arena`
en sert 48. **Le variant de mode n'est donc PAS par carte.** Il ne peut pas porter les
positions des socles de Cliffhanger : il est partage avec 30 autres cartes.

Consequence sur ce qu'on peut esperer trouver — et c'est le coeur de la sonde :

- **H-FILTRE** (seule forme survivable) : le variant porte une REGLE — une liste de labels a
  inclure/exclure — appliquee aux objets du `.mvar`. C'est exactement la forme des labels
  Forge deja vus (`*_include` / `*_exclude`). Le croisement serait alors
  `.mvar` (OU) x variant (QUOI allumer).
- **H-POSITIONS** : REFUTEE AVANT MESURE par le partage 1 variant / 31 cartes. On ne la teste
  pas, on l'ecrit comme close.

## 4. Seuils, poses AVANT la mesure

**Q2 — le payload contient-il l'activation ?** POSITIF si et seulement si on trouve, dans le
JSON de l'asset OU dans un fichier qu'il reference, au moins un de :

- a) un des 3 `type_id` de socle, en clair (decimal) ou sur 4 octets (LE ou BE) ;
- b) un des labels de mode connus en clair, OU son hash murmur3_x86_32(seed 0) sur 4 octets ;
- c) un des 9 hashs non resolus du plan socles-mvar (dont `-886053664`, `-831896525`) ;
- d) une structure nommee (cle JSON ou chaine du blob) de la famille
  « spawner / pad / equipment palette / weapon set / object filter ».

**Aucun des quatre -> NEGATIF ECRIT.** On ne conclut pas « c'est peut-etre encode » : un
negatif est un resultat, et il oriente vers les alternatives (section 6).

**Q3 — le variant EXPLIQUE-t-il 17 -> 10 / 0 ?** POSITIF seulement si un CHAMP identifie
differe entre les deux variants d'une paire ET que la difference a la forme d'un allumage
(inclusion/exclusion de socles ou de labels). Le seuil anti-illusion est explicite :
**« les deux payloads different » ne vaut RIEN** — CTF et Super Fiesta different
trivialement (score a atteindre, armes de depart, duree). Il faut nommer le champ, et il
doit valoir des deux cotes des DEUX paires.

**Q1 — recuperation.** Le `version_id` est NULL pour les 4 temoins dans l'instantane (1544
lignes sur 1819 sans version). Le repli est le chemin sans `/versions/` de
`cmd/mapobj-build/fetch.go`. Si les DEUX chemins echouent : `[!]` et arret sur ce point.

**Auth (regle dure, ADR 0023)** : jeton par le store existant uniquement. 401/AADSTS ->
**arret**, erreur exacte rapportee, aucune reparation, aucune re-capture, aucune ecriture de
jeton. Appels : une poignee (4 variants + leurs blobs), jamais de boucle sur le registre.

## 5. Phases

### Phase 0 — plan (commit 1)
- [x] 0.1 Acquis verifies sur pieces (section 3), seuils poses (section 4), H-POSITIONS close
      avant mesure par la mesure de cadrage 1 variant / 31 cartes.

### Phase 1 — recuperer les 4 variants (Q1) (commit 2)
- [x] 1.1 `cmd/variant-probe` : outil de SONDE, args explicites obligatoires, aucun defaut qui
      tape le reseau. Reutilise `cmd/mapobj-build/auth.go` + `fetch.go` comme patron.
- [x] 1.2 Recuperer les 4 assets. Ecrire les JSON bruts dans `.ai/re_dump/gamevariant/`
      (gitignore : `.gitignore:254`). Chiffrer : taille, liste `FileRelativePaths`.
- [x] 1.3 Gate : `go vet ./cmd/variant-probe/...`, golangci 0 sur le perimetre.

### Phase 2 — les fichiers references (commit 3)
- [x] 2.1 Telecharger les fichiers references depuis `Files.Prefix`.
- [x] 2.2 Inventaire : nom, taille, type (texte/binaire), entete.
- [x] 2.3 AJOUTE EN COURS D'EXECUTION (decouverte de 1.2, elle deplace la cible) : interroger
      l'asset `EngineGameVariants/{assetId}/versions/{versionId}` pointe par chaque variant.
      C'est l'alternative n. 3 de la section 6, servie par le document lui-meme.

### Phase 3 — chercher l'activation (Q2) (commit 4)
- [x] 3.1 Les 4 sondes a/b/c/d du seuil, sur JSON ET blobs.
- [x] 3.2 Verdict Q2 : ou est l'activation, ou elle n'est pas. Negatif ecrit si negatif.

### Phase 4 — verdict discriminant (Q3) + cout (Q4) (commit 5)
- [ ] 4.1 Comparer les deux paires (Cliffhanger CTF/Fiesta ; Catalyst KOTH/CTF).
- [ ] 4.2 Verdict Q3 au seuil de la section 4, sans complaisance.
- [ ] 4.3 Q4 : 56 variants distincts / 1819 matchs (deja mesure) + taille des payloads +
      faisabilite d'une reference versionnee.
- [ ] 4.4 Architecture validee ou refutee ; si refutee, alternatives nommees.

## 6. Alternatives si NEGATIF (a instruire, pas a supposer)

1. **Le blob du variant est binaire et l'activation y est encodee** — parser (grammaire CB2
   deja ecrite pour `.mvar` : `internal/analysis/replay/mapvar/cb2.go`).
2. **`mapModePairs`** : la PAIRE carte x mode est le seul asset qui connait les deux. Le
   registre porte deja `pair_id` + `pair_version_id`.
3. **Engine game variant** : le variant UGC pointe peut-etre un variant de MOTEUR
   (`EngineGameVariantLink`), ou vivraient les regles.
4. **Le jeu installe** : les tags `himap` portent les type_id de socle (piste ouverte au
   plan socles-mvar section 9). Hors ligne pur, mais dependant d'une installation.
5. **Repli empirique deja disponible** : le FILM dit ce qui a spawn. La regle de croisement
   « ne montrer les socles statiques que si le film en confirme au moins un » est deja
   proposee au plan socles-mvar section 8bis — elle ne demande AUCUN reseau.

## 7. Journal

- **2026-08-19, phase 0 close** — Plan ouvert. Acquis verifies sur pieces. Serveur actif sur
  `:8000` constate : la lecture du registre passe par l'instantane parquet, aucune ouverture
  de `shared_matches_v2.duckdb`. Mesure de cadrage faite AVANT toute requete reseau : 56
  variants distincts sur 1819 matchs, et `CTF:Arena` sert 31 cartes — ce seul chiffre REFUTE
  H-POSITIONS avant la premiere mesure et reduit la sonde a H-FILTRE. Seuils poses, y compris
  le seuil anti-illusion de Q3 (« les payloads different » ne prouve rien).

- **2026-08-19, phase 1 close** — `cmd/variant-probe` ecrit (3 fichiers, 94 a 140 lignes,
  garde : sans `--variant` ET `--out` il sort en erreur sans emettre une requete). Auth par
  le store : `oauth_refresh: echange OK`, xuid 2533274823110022 (JGtm), clearance presente,
  **aucune re-capture**. Les 4 temoins ne font que **3 variants distincts** (CTF:Arena sert
  Cliffhanger ET Catalyst) : 3 appels, pas 4.
  **Le repli sans `/versions/` marche** et il resout la MEME version que le registre quand
  celui-ci en a une : CTF:Arena -> `7f104b0c` (identique au registre), Super Fiesta ->
  `c1e1f80d` (identique), KOTH:Arena -> `e76c9d61` (le registre n'en avait pas). Q1 = OUI.
  **Taille des payloads : 2 361 a 2 538 octets.** Un document de VITRINE, pas de regles :
  `CustomData.KeyValues` = **`{}` vide sur les trois**, `HasNodeGraph: false`, et
  `Files.FileRelativePaths` ne liste que des **images** (hero, screenshots, thumbnail — 3 a 8
  fichiers). Aucun `.bin`, aucun blob de donnees.
  **DECOUVERTE qui redirige la phase 2** : chaque variant porte un `EngineGameVariantLink`
  vers un asset SEPARE — CTF:Arena -> `71cca199` « Capture the Flag » (`ParentAssetCount`
  107 517), KOTH:Arena -> `62216cfe` « King of the Hill », Super Fiesta -> `a65a43f0`
  « Slayer-SlayerSuperFiesta ». C'est l'alternative n. 3 de la section 6, et elle est servie
  par le document lui-meme. Dans le lien imbriqué, `FileRelativePaths` est **vide** — il faut
  interroger l'asset engine directement. Item 2.3 ajoute en consequence.
  Gates : `go vet ./cmd/variant-probe/...` = 0, `golangci-lint run ./cmd/variant-probe/...` =
  **0 issues**.

- **2026-08-19, phase 2 close** — Fichiers references par les UgcGameVariants : **QUE des
  images** (hero, screenshots, thumbnail — 41 Mo pour CTF:Arena, la hero est en pleine
  resolution). Aucune donnee de regle de ce cote : Q2 est NEGATIF sur l'asset de vitrine.
  **L'asset `engineGameVariants` change tout** : il expose **38 fichiers**, dont un `.bin`
  de regles par mode — `MultiFlag.bin` (CTF, **529 643 o**), `Default.bin` (KOTH/Bastion,
  **516 332 o**), `SlayerSuperFiesta.bin` (**405 880 o**) — plus 18 `CustomGamesUIMarkup/
  *_{langue}.bin` (98 a 147 Ko) et 18 fichiers de libelles minuscules (`MultiFlag.fr` =
  153 o). Le `_guid.txt` donne un GUID stable par mode (CTF `27ee48da-...`, KOTH
  `23c9f7c0-...`, Fiesta `50ec7add-...`).
  **Les `.bin` sont du Bond**, comme les `.mvar` : entete `e8a9 202b 4a...` puis des chaines
  lisibles en clair — `CTF` / `MultiFlag`, `Bastion` / `Default`, `Slayer` /
  `SlayerSuperFiesta`. La grammaire `internal/analysis/replay/mapvar/cb2.go` est donc
  candidate a les lire.
  Le segment `engineGameVariants` n'existait dans aucune table du depot : il a ete trouve
  par le document lui-meme, pas devine.
  Gates : `go vet` = 0, `golangci-lint run ./cmd/variant-probe/...` = **0 issues**.

- **2026-08-19, phase 3 close** — Mode `--scan` ajoute (HORS LIGNE, aucune requete).
  **32 valeurs cherchees x 4 encodages** (4 octets LE, 4 octets BE, varint LEB128, varint
  zigzag) sur 123 fichiers. Le quadruple encodage n'est pas du zele : le seul resultat
  positif de tout le lot est en **varint zigzag**, un scan en 4 octets fixes aurait conclu
  « absent » a tort.

  | Sonde | `MultiFlag.bin` (CTF) | `Default.bin` (KOTH) | `SlayerSuperFiesta.bin` |
  |---|---|---|---|
  | a) 3 `type_id` de socle | **0** | **0** | **0** |
  | b) 27 labels connus | **3** : `ctf_include`, `ctf_exclude`, `ctf_multi_exclude`, x1 chacun | **0** | **0** |
  | c) hashs non resolus cibles | **0** | **0** | **0** |
  | d) famille spawner/pad/palette | **0** | **0** | **0** |

  Le motif est VERIFIE, pas suppose : a l'offset 526037 de `MultiFlag.bin`, la sequence
  `0a 07 10 9b 8c c9 c6 0f 00` decode en varint zigzag vers **-2087265038**, soit exactement
  `mapvar.LabelHash("ctf_include")`.
  **Ce qu'on ne trouve nulle part** : aucun des 3 `type_id` de socle, aucun nom d'arme de
  socle ni de power-up (`overshield`, `camo`, `sword`, `hammer`, `sniper`, `spnkr`... : zero
  sur les trois), aucune structure de palette. Les seules chaines d'armes sont une
  enumeration de vehicules identique entre CTF et KOTH.
  **Les .bin embarquent du bytecode Lua** (`LuaQ`, 107 a 124 chunks) : les regles sont bien
  la. Ses identifiants ne parlent que de drapeaux et de spawn joueur (`FlagSpawnEffect`,
  `HandlePlayerSpawnOnClient`, `PowerWeaponKills` = une statistique). Les gestionnaires
  d'armes sont des **chemins de tags vers le jeu installe**
  (`tags\scripts\parcellibrary\parcel_mp_weapon_manager.lua`,
  `MPWeaponManagerStartup.lua`) — **identiques dans les trois modes**, Super Fiesta compris,
  qui n'allume pourtant aucun socle. Le payload reference le code, il ne le contient pas.
  **Verdict Q2** : le seuil (b) est techniquement atteint sur CTF, et **il ne prouve pas ce
  qu'on cherchait**. Ces trois labels sont les filtres des objets d'OBJECTIF (les supports de
  drapeau, nommes en clair dans le meme fichier : `Blue Flag Stand`, `Red Flag Stand`,
  `Neutral Flag Stand`), pas des socles d'armes. Et le lot socles-mvar a mesure que **les
  socles des cartes DEV ne portent AUCUN label** : un filtre par label ne peut pas allumer un
  objet qui n'en porte pas.
  Gates : `go vet` = 0, `golangci-lint run ./cmd/variant-probe/...` = **0 issues**.

## 8. Decouvertes (notees, NON traitees)

- `resolveTokens` + `lookupPlayer` de `cmd/mapobj-build/auth.go` sont RECOPIES dans
  `cmd/variant-probe/auth.go` — 2e copie, encore sous le seuil de la regle 6 du CLAUDE.md.
  A la 3e, centraliser dans un helper partage + garde-rail. Non traite : hors perimetre d'un
  lot de mesure.
- Le registre a `game_variant_version_id` NULL sur 1544 lignes / 1819. L'API resout pourtant
  la bonne version sans le champ. Le backfill de cette colonne n'est donc pas un prealable.
- `CTF:Arena` apparait au registre sous deux libelles (`CTF:Arena`) et `Slayer:Arena` sous
  deux (`Slayer:Arena` et `Arena:Slayer`, meme `game_variant_id` `1e8cd10b`) : le libelle
  suit l'ordre de la paire, pas l'asset. Sans effet sur cette sonde.
