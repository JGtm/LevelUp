# RE statique HaloInfinite.exe (Ghidra) — système de stats, format wire de réplication, lien kill→arme

> **Document de référence / handoff.** Toutes ces trouvailles viennent de la décompilation **statique** du
> fichier `HaloInfinite.exe` (Steam, ~83 Mo, **non packé**, strings en clair) — analyse du fichier, jamais
> du process. Adresses = **VA Ghidra** (ImageBase `0x140000000`). Build du jeu au moment de la RE : `6.10026.19225.0`.
> Complète le film-from-anchors de `RESEARCH_THEATER_RE.md` (§M = score, §M-quater = résumé wire).

---

## 1. Système de stats en mémoire (le « statborg »)

- **`DAT_144ebd098`** = pointeur monde ECS. Constructeur **`FUN_140c180a8`** :
  `DAT_144ebd098 = world ; memset(world+8, 0, 0x59d00) ; FUN_14047bdd0(world+8, 0x1df0, 0x30, ctor=FUN_140c185c4)`.
  → **48 statlines** (0x30) de **0x1DF0** octets chacune, à `world+8`. Reset de round : **`FUN_141dc3c50`**.
- **Layout d'une statline (0x1DF0)** : `+0x18` = 32 flags (1 octet/stat), `+0x38` = valeurs par round (int32,
  une par round ; **stride par stat = 0x88** = 0x22 dwords), `+0xbc` = champ dérivé (autre représentation).
- **Lecture** :
  - getter round-value : **`FUN_1406ada4c`** lit `*(int32*)(world + statSlot*0x88 + teamIdx*0x1DF0 + 0x38 + round*4)`.
  - dispatch 4-types : **`FUN_1406aee98`** (type0→`FUN_140b72c10`, type1→inline `+0xbc`, type2→`FUN_1406ada4c`, type3→`FUN_1406ac350`).
  - flag-readers (`+0x18`) : `FUN_142b74aa8`, `FUN_142b794e4`, `FUN_141e37fe9`.
  - **`displayed = raw * scale`**, `scale = DAT_143ce70a8[ *(byte*)(descripteur + 0xBC) ]` (table d'échelle par stat).
    → explique le `token+24 ×3.86` du film (§M) : le stockage est le **brut**, l'affichage est brut×échelle.
- Binding HavokScript `Team_GetCurrentRoundStatValue` = getter natif **`0x142C6B118`** → lookup `FUN_142b7974c` → `FUN_1406ada4c`.
- Descripteur de stat : table à `engine+0xDF77C` (entrées 0x30). Noms de composants en `.rdata` ~`0x143c95b80`,
  dont `statborg-current-round-value-stat-component`, `statborg-finalized-rounds-values...`, et **`stat-value-encoding-bits-header`**.

## 2. FORMAT WIRE DE RÉPLICATION (le cœur — ce que personne n'a publié)

Le film **EST** le flux de réplication enregistré ; le Theater le rejoue via **le même** désérialiseur. Donc
décoder le film = appliquer ces fonctions. Bit-packé **MSB-first** (byte-swap big-endian), modèle **baseline + deltas**.

**Lecteur d'entier** `FUN_140c18a1c(bitreader)` — l'encodage exact d'un int :
```
sel = read 2 bits                  // sélecteur de largeur
w   = 8 << sel                     // largeur = 8 / 16 / 32 / 64 bits  (1/2/4/8 octets)
v   = read w bits  (MSB-first)
if (w < 32 && bit_de_poids_fort(v))  v = sign_extend(v)   // ENTIER SIGNÉ
return (int32) v
```
→ **entier signé à longueur variable, préfixe de 2 bits de largeur.** Ex. score ≤127 → `00`+8b ; 128..32767 → `01`+16b
(allié 167/193 en 16b, ennemi 17..112 en 8b). C'est pourquoi toute recherche **largeur-fixe** échouait.

**Lecteur 1 bit** `FUN_1406cf008` : `bit = (acc>>63)&1 ; acc<<=1`.

**Désérialiseur d'un stat 2-équipes** `FUN_140c18794(handle, bitreader, target)` — ordre du record :
`[5 bits en-tête A][5 bits en-tête B][valeur A][valeur B][1 bit flag A][1 bit flag B][valeurs conditionnelles]`.
→ **les 2 équipes sont sérialisées CONSÉCUTIVEMENT** (l'adverse suit l'allié, sans ancre séparée). dirty-bits à `target+0x1c0`.

**Apply** `FUN_140807ebc(world, teamIdx, dirtymask, …, src)` (+ wrapper `FUN_140808274`) : recopie l'objet
dé-sérialisé `src` vers `world+teamIdx*0x1DF0`, boucle sur les stats (`+= 0x88`), gatée dirty-bits. Écrit le
round-value à `statline+0x38` (instruction `0x14080801E : mov [rbx+rax*4],ecx`).

**État du BitReader** (param `+0x10`=fin buffer, `+0x28`/`+0x2c`=compteurs, `+0x30`=accumulateur 64-bit,
`+0x38`=position, `+0x40`=curseur). Refill 8 octets + byte-swap big-endian, consommation depuis le MSB.

**DEUX encodages coexistent** : (a) keyframe TYPE_2 = **varint à continuation** (l'allié décodé à `token+24`) ;
(b) deltas FRAME + full-state = **`FUN_140c18a1c`** (2-bit sélecteur signé). Le score adverse absolu passe par (b).

## 3. FRAMING (comment localiser un composant dans le paquet)

Le réplicateur est **générique + data-driven** : aucune constante stat dans le désérialiseur ; il est trouvé par **type-id**.
- **Table de dispatch** `0x145435cd8` (.data) : triplets `{fn, fn, type-id-hash}` (ex. hash `0x03e902a0`…). Le paquet
  porte un type-id par composant → le dispatch trouve la **vtable** du composant → appelle le slot désérialiseur.
- **Vtable du composant statborg** `0x143c96b38` (.rdata, près du nom `0x143c95bb8`) : tableau de pointeurs
  `[u32 addr][01 00 00 00]`, et **le slot à +0x38 = `0x140c18794`** (notre désérialiseur). Autres slots = ctor/serialize/…
- ⇒ Décoder le film offline = parser le paquet FRAME, lire chaque record `[type-id][handle][champs bit-packés]`,
  router par type-id, appliquer le désérialiseur. **Le type-id du statborg reste à extraire** (un des hashs du dispatch).

## 4. Lien KILL → ARME (engine OUI ; film = À CONFIRMER)

- **`FUN_140a24f6c`** = builder d'event **télémétrie** Xbox (CELL) du kill. Champs : `KillerAgent`/`VictimAgent`,
  positions, `AssistingPlayerSessionMatchIndices`, **`DamageSourceObjectDescriptionSessionIndex`** (l'objet arme),
  **`DamageEffectDefinitionGlobalTagId`** (= **l'identité de l'arme**, le tag global de l'effet de dégât, type Murmur
  comme notre suffixe `0x42c9679f`), **`DamageReportingModifier`**. `param_7` (la source de dégât) → le GlobalTagId.
- **`FUN_142baee74`** = event télémétrie « descope » avec `OpposingDamageEffectTagId`, `OpposingWeaponObjDescSeqId`…
- **`FUN_140b478d8`** = **handler de kill maître** : reçoit `param_7` = l'arme, calcule killer/victim/assists, et
  passe l'arme à `FUN_140b48200`→**`FUN_140b48460`** et `FUN_140b480f0`→**`FUN_140b48148`** (packagers d'event),
  + `FUN_140b48558` (médailles). **Les écrivains réels = `FUN_140b48460` / `FUN_140b48148`** (non décompilés à l'arrêt).

**CONCLUSION kill-weapon — TRANCHÉE (2026-06-05)** : l'engine **connaît l'arme-de-kill** au moment du kill
(`DamageEffectDefinitionGlobalTagId`), mais elle part vers **(a) la télémétrie Xbox** (`FUN_140a24f6c`) et
**(b) l'event script `__OnKill`** (`FUN_140b48148` → callback HavokScript `FUN_1406f9280(...,"__OnKill")`, poussé
au gametype Lua). `FUN_140b48460` ne fait que copier le record. **AUCUN des deux n'écrit le FILM.** L'highlight
event type-3 qu'on parse ne porte que killer/victim + `DamageReportingModifier` (type de kill), **pas d'arme**.
⇒ **L'arme-de-kill n'est PAS dans le film** → elle reste **inférée** : `weapon_kills_v3` (attribution par fire
events) est la **bonne approche, et la seule**. Pas de gain Ghidra possible sur le kill-weapon. **RÉGLÉ.**

**Re-vérifié 2026-06-05 (à la demande user — confirme le négatif STRUCTURELLEMENT, pas circulairement)** : le champ
**`KillerWeapon`** existe bien, mais il vit dans un **ViewModel HUD**, pas un composant répliqué. Registreurs
`FUN_140400760` (kill-feed, 2 champs : `KillerGamerTag`+`KillerWeapon`) et `FUN_14040e290` (death-recap, 23 champs) ;
owners `FUN_1404006a0`/`FUN_14040e1d0` partageant la **vtable `DAT_144f4fab0`** (classe data-binding UI, fieldCount
2 et 0x17). `KillerWeapon` = **tag 64-bit** (getter `FUN_14348977c` = to-variant type-4 ; valeur à wrapper+8). Les
champs voisins (`IsEntering`/`IsLeaving`/`IsRecapOver`, `KillerTeamColor`, `NormalizedKillerLife/Shields`, emblèmes,
`KillerPercentageDamageDone`) sont **100% présentation** ⇒ le bandeau kill-feed du replay est **reconstruit côté
client**, l'arme **n'est PAS sérialisée dans le film**. Confirme §4 par une 2ᵉ ligne indépendante. `FUN_140748a74` =
**Murmur3_32(nom normalisé)** = hasher string_id (confirmé). Bonus produit : l'inventaire ViewModel = **spec officielle
recap/kill-feed** (cf. memory `reference_halo_hud_viewmodels`).

## 5. Taxonomies utiles

- **`DamageReportingModifier`** (type de kill, enum) : `Headshot`(1), `HeadshotMultiplier`(2), `SilentMelee`(3),
  `CollisionDamage`(4), `AttachedDamage`(5), `WeakSpot`(6), `ChainedProjectile`(7), `SweetHeat`(8), `VehicleTransferDamage`(9).
  → si présent dans l'event film, permettrait de désambiguïser melee/grenade-collée/etc. (gain incrémental).
- **Médailles** (`FUN_140b48558`) : déclencheurs `__OnPullMedal`, `__OnHoldThisMedal`, `__OnStyle360Medal`,
  `__OnNinjaMedal`, `__OnDeathRaceMedal`, `__OnSpecialDeliveryMedal`, `__OnFromTheVoidMedal`, `__OnReturnToSenderMedal`,
  `__OnBankShotMedal`, `__OnDeadlyCatchMedal`. (On a déjà les médailles via l'API ; pour info.)
- `TeamDesignator` : Defender(0), Attacker(1), ThirdParty(2)…Neutral(8), None(-1). `MultiplayerSquad` : Alpha..Hotel.

## 6. Carte des adresses (VA Ghidra)

| Fonction / data | Rôle |
|---|---|
| `DAT_144ebd098` | pointeur monde ECS (base du tableau de stats) |
| `0x140c180a8` | constructeur du système de stats (48×0x1DF0) |
| `0x141dc3c50` | reset des statlines (début round) |
| `0x1406ada4c` | getter round-value (`+0x38+round*4`) |
| `0x1406aee98` | dispatch lecture stat (4 types) |
| `0x142C6B118` | getter natif `Team_GetCurrentRoundStatValue` |
| **`0x140c18794`** | **désérialiseur stat 2-équipes** |
| **`0x140c18a1c`** | **lecteur int signé `[2-bit][largeur]`** |
| `0x1406cf008` | lecteur 1 bit |
| `0x140807ebc` / `0x140808274` | apply → statline |
| `0x145435cd8` | table de dispatch (type-id → fn) |
| `0x143c96b38` | vtable composant statborg (slot +0x38 = deser) |
| `0x143ce70a8` | table d'échelle (scale par stat) |
| `0x140a24f6c` | builder télémétrie kill (a l'arme) |
| `0x140b478d8` | handler de kill maître (reçoit l'arme `param_7`) |
| `0x140b48460` / `0x140b48148` | écrivains d'event de kill (à décompiler pour « arme dans film ») |
| `0x140b48558` | handler médailles |

## 7. Acquis vs ouvert

**ACQUIS** : structure mémoire des stats ; **format wire exact** (encodage valeur + structure 2-équipes + MSB-first +
delta/dirty-bits) ; framing (dispatch type-id + vtable) ; lien kill→arme au niveau engine ; taxonomie kill-type + médailles.

**OUVERT** : (1) BitReader **bit-exact** en Go (machine à états refill/byte-swap) ; (2) **type-id du statborg** +
parsing des records FRAME par type-id (le framing concret). Chantier d'implémentation, mais tout est cartographié.

**RÉGLÉ (négatif)** : **arme-de-kill dans le film = NON** (cf. §4 : va en télémétrie + event script `__OnKill`,
pas dans le film). `weapon_kills_v3` (inférence fire events) reste la bonne approche. Ne pas re-explorer.
**Re-confirmé 2026-06-05** : `KillerWeapon` existe mais = champ d'un **ViewModel HUD** (vtable `DAT_144f4fab0`),
reconstruit côté client au replay — cf. §4. L'inventaire des 2 ViewModels (death-recap 23 champs + kill-feed 2) est
une **spec officielle** réutilisable côté app.
**Nuance finale (cf. §4 RAFFINEMENT)** : « pas dans le film » = vrai pour le **type-3 event** seulement. L'arme EST
dans le film comme **valeur de composant ECS `'obje'`** (u32 asset-id), même mur que score/positions → décodeur FRAME.

**RAFFINEMENT 2026-06-05 (le « négatif » était trop fort — l'arme EST dans le film, comme composant ECS)** : en
traçant l'écrivain du kill-feed, la source de l'arme remonte à l'**ECS**, pas à une valeur inventée par l'UI :
`FUN_1420ca9a0` (peuple le kill-feed en itérant une liste) → `FUN_1407e941c(VM, handle, key)` →
`FUN_1407e8afc(handle, idx, 0)` → **`FUN_1405839d0(entity, 0x6f626a65='obje')`** = lookup du composant **object**
(group tag Halo `'obje'`), tableau de records 0x344 o, item-arme = sous-bloc 0x24 o. Le visuel sort de
`FUN_1407e9724` : **`*(u32*)(srcItem+0x18)` = asset-id de l'arme** (+ `+0x1c`/`+0x20`). ⇒ l'arme-de-kill est un
**u32 asset-id dans un composant ECS répliqué** — **même substrat que le score (gagnant) et les positions déjà
décodés**, pas un octet libre du type-3. **Conclusion corrigée** : l'arme N'est PAS dans le type-3 event (mon point
initial tient), mais elle EST dans le film comme **valeur de composant ECS** (le point de l'utilisateur tient),
verrouillée par le **même mur** : décodeur FRAME bit-exact + schéma `'obje'` (`.module` hors-film) + pertinence
réseau (cf. §2/§3 + §7). Donc le kill-weapon tombe **avec** le décodeur ECS générique, pas avant. `weapon_kills_v3`
(inférence fire events) reste la voie pragmatique d'ici là.

> ⚠️ Note méthode : certaines adresses (écrivain du score, handler de kill) ont été localisées par l'utilisateur via
> un write-watchpoint sur le jeu lancé. L'**analyse** ci-dessus est 100% **statique** sur le fichier .exe. L'agent n'a
> pas fourni de procédure de contournement anti-triche (refus assumé) ; lire le fichier est sans risque.

---

## 8. L'ÉCRIVAIN DE `chunk_00` (2026-09-12) — la grammaire complète de la « section 3 »

Ancre de départ : les chaînes d'identification de build que le film écrit en clair
(`HI_1_13_0` à `0x1436a37c0`, `release` à `0x143690658`). Leurs xrefs mènent à
`FUN_140b32390`, qui construit la structure d'informations de build, puis à
`FUN_140b32438`, qui la recopie dans des globales. **Deux de ces globales sont les deux u32
inconnus de l'en-tête du film** (`NOTE_CARTE_CHUNK00_2026-08-30`, question ouverte n°4) :
`DAT_144e4ef50` (identifiant de build pipeline, `0x41ba9` dans cet exe) et `DAT_144e4ef54`
(**changelist**, `0x86f6bf`, formaté par `" changelist: %d"`).

La fonction qui LIT ces trois globales ensemble et les recopie dans des champs de 32 octets est
l'initialiseur de l'écrivain de film. Tout en découle.

| Fonction / donnée | Rôle | Preuve |
|---|---|---|
| `0x140b32390` | construit la structure d'infos de build (version, branche, saveur, date, heure, projet, hash, changelist) | xrefs des chaînes `HI_1_13_0` / `release` |
| `0x140b32438` | recopie cette structure vers les globales `DAT_144e4ef30..90` | appelé par `FUN_140b32390` |
| `DAT_144e4ef38` / `ef88` / `ef78` | version · build · saveur (les trois champs de 32 o du film) | recopiés à `+0xcb524/0xcb544/0xcb564` |
| `DAT_144e4ef50` / `ef54` | **les deux u32 de `0x0CB454`** : id de build, **changelist** | même ordre, même magnitude que les valeurs du film |
| **`0x14299b674`** | **initialiseur de l'écrivain de film** : remplit le tampon de `chunk_00` | écrit les trois chaînes à `+0xcb524/44/64`, les deux u32 à `+0xcb584/88`, `_time64()` à `+0xcb790` |
| — boucle registre | `FUN_142998c7c(base+0x138 + i*0x4100)` tant que `< 0xcb200` → **50 blocs** de 64 slots de 260 o | `0xcb200 / 0x4100 = 50` exactement |
| — boucle table par type | `(**(vtable+0x30))()` sur les objets de `DAT_144e61d88+0x210`, écrits à `+0xcb338 + j*4` tant que `j*4 < 0x1ec` → **123 u32** | `0x1ec / 4 = 123` ; appuie la lecture « version de sérialisation PAR TYPE » : la valeur sort d'une méthode virtuelle du descripteur de type |
| `0x14299cb5c` | ouvre le tampon de sortie de **`0x1E1B80` = 1 973 120 octets** et lance la sérialisation | `FUN_1429907c8(buf, 0x1e1b80)` — égal à la taille inflatée mesurée le 30/08 |
| **`0x14299b198`** | **sérialise la TÊTE** : 2 u32, registre (`0x659000` bits), table par type (`0xF60` bits), 3 × `0x100` bits, 2 × `0x20` bits, **1 booléen d'UN bit** (`FUN_1406d49c4`) | désassemblage : `LEA R8,[RBX+off] ; MOV R9D,bits ; CALL FUN_1406d60f4`, offsets strictement contigus |
| **`0x14299b278`** | **sérialise la SUITE** : 2 × `0x800` bits (noms UTF-16 de 128 car.), 4 × `0x20` bits (dont **l'horodatage du match**), 3 × `0x8000` bits, 2 × `0x80` bits, puis le corps | idem |
| `0x1406d60f4` | **écrivain de bits générique** `(writer, _, src, nbits)` : MSB-first, byte-swap 64 bits, curseur `*(w+0x2c) += nbits` | décompilé |
| `0x1406d49c4` | écrivain d'UN bit (booléen) | appelé sans largeur |
| `0x1406d6498` | écrivain d'un scalaire `(writer, valeur, nbits)` | décompilé |
| `0x1407ebe7c` | écrivain de chaîne ASCII terminée par NUL, largeur max en 4ᵉ arg (8 bits/caractère) | décompilé |
| **`0x1407ec560`** | **SÉRIALISEUR DU CORPS** (= la « section 3 ») : champs de 3/3/2/7/64/32/3 bits, sous-structures, **2 chaînes ASCII de 256 car. max** (`+0xEA810`, `+0xEA910`), `0x6C0` bits, puis la boucle de slots | décompilé + désassemblé |
| — boucle de slots | `for (p = jeu+0xEAAF0 ; p != +0x113AF0 ; p += 0x1450)` → **32 enregistrements de 0x1450 octets** | `0x28A00 / 0x1450 = 32` ; et `0xEAAF0 + 0x28A00 = 0x113AF0` ferme sur la fin du tampon |
| **`0x1407ecb08`** | **SÉRIALISEUR D'UN SLOT** : `1+1+1` bits booléens, `32` bits, `2` bits, `48` bits, puis **`64` bits = LE XUID**, puis le sous-enregistrement `FUN_1407edea8`, puis `32` bits | désassemblage ; vérifié sur films : 44 XUID de `match_participants` retrouvés à 85 bits de l'en-tête, 0 faux positif sur 240 leurres |
| `0x1407edea8` | sous-enregistrement du slot (`slot+0x18` … `+0x1448`) : apparence, personnalisation, dotation | callees `FUN_1407ecd00/ece18/edaf4/edcc4/edd3c/eddb4/ede30` — **non décodé** |
| `0x140b857d8` · `0x1410bc140` · `0x140b85504` | sous-sérialiseurs du corps (`jeu+0xE951C`, `+0xEA72C`, objet optionnel à `jeu+0x28`) | **non décodés** |

**Conséquences immédiates.**

1. **Le registre ECS commence à l'octet 8 du tampon inflaté**, pas à 0 : `0x659000` bits écrits
   depuis `base+8`. `parseRegistry` lit depuis 0 ; son champ « kind » du slot *i* est en réalité le
   champ situé 8 octets avant le nom du slot *i*. Cela explique d'un coup « kind = 0 sur 1 066 des
   1 067 slots » et le « niveau lu un cran plus loin » de `registryBlockTail`.
   **Découverte hors périmètre, non traitée.**
2. **La table par type commence bien à `0x0CB208` et compte exactement 123 entrées** : la question
   ouverte n°2 de la note du 30/08 est fermée par l'écrivain, avec fermeture arithmétique
   (`0x0CB208 + 123 × 4 = 0x0CB3F4` = l'offset du champ version).
3. **Après le booléen d'un bit à `0x0CB45C`, tout le reste du fichier est décalé d'un bit.** C'est
   la raison mécanique pour laquelle aucune lecture alignée sur l'octet ne rendait rien dans la
   section 3.

Détail, offsets et commandes de rejeu : `.ai/V7.5/film_re/NOTE_SECTION3_CHUNK00_2026-09-12.md`.

### 8.1 Le sous-enregistrement de slot, ouvert d'un cran (2026-09-12)

`FUN_1407edea8(sub = slot+0x18, writer)` appelle d'abord `FUN_1407ecd00`, qui sérialise trois
listes préfixées par leur longueur — la source mécanique de la longueur variable d'un
enregistrement de slot :

| source (sub-relative) | forme | écrivain |
|---|---|---|
| `+0x000`..`+0x100` | **masque de présence de 2 048 bits** (64 u32), préfixé par le rang du bit le plus haut, puis écrit bit à bit | `FUN_1407ecd78` (`MOV EAX,0x3f` = balayage arrière des 64 mots, puis `BT`/`SETC` + `FUN_1406d49c4`) |
| `+0x100` / `+0x108` | u32 de longueur N, puis **N octets** (`N << 3` bits) | `FUN_1411b1a24` puis `FUN_1406d60f4` |
| `+0x908` / `+0x910` | u32 de longueur M, puis **M × 4 octets** (`M << 5` bits) | `FUN_1411b198c` puis `FUN_1406d60f4` |

Puis, toujours dans `FUN_1407edea8` : `+0xC48` sur `0x340` bits (104 o), `+0xC14` via
`FUN_1407ece18` (`0x10`), `+0xC38` sur `0x80` bits, `+0xCB0` via `FUN_1407edaf4(writer,
DAT_143686818, valeur)` — **le pointeur de données `0x143686818` est un candidat table de
correspondance** —, `+0xCB8` sur `0x40` bits, puis `FUN_1407edcc4`, `FUN_1407edd3c`,
`FUN_1407eddb4`, `FUN_1407ede30`.

**Correction du 2026-09-12 (phase 2)** : la phrase qui concluait cette section — « le gamertag et
l'équipe ne sont dans aucun des champs ci-dessus » — est FAUSSE pour le gamertag. `FUN_1407ece18`
n'est pas un écrivain de scalaire de 16 bits : c'est un **écrivain de chaîne UTF-16**, qui écrit des
unités de 16 bits MSB d'abord jusqu'à l'unité nulle incluse, au plus `R9D` unités. Le champ
`sub+0xc14` sur `0x10` unités EST donc le gamertag. Voir 8.2. Pour l'équipe, la phrase tient : elle
n'est dans aucun des champs décodés, et la phase 2 l'établit par la mesure.

### 8.2 L'enregistrement de slot, décodé au complet (2026-09-12, phase 2)

Le sous-enregistrement `FUN_1407edea8(sub = slot+0x18, writer)` est désassemblé **en entier**, et la
largeur de chaque champ est lue **dans le code de son écrivain** — pas déduite, pas supposée. Les
fonctions ci-dessous sont toutes des écrivains de bits du même moule : `MOV ECX,<largeur> ;
ADD dword ptr [RDX + 0x2c],ECX`, la convention de curseur déjà connue du dossier.

| Fonction | Rôle | Largeur | Preuve |
|---|---|---|---|
| **`0x1407edea8`** | **sous-enregistrement de slot, 16 champs** | — | désassemblage complet : `CALL 0x1407ecd00` ; `+0xc48`/`0x340` ; `+0xc14`/`0x10` via `1407ece18` ; `+0xc38`/`0x80` ; `+0xcb0` via `1407edaf4` ; `+0xcb8`/`0x40` ; `+0xc12` via `1407edcc4` ; `+0xc36` via `1407edd3c` ; `+0xc35` via `1407eddb4` ; `+0xc10` inline ; `+0xc34` via `1407ede30` ; `+0xc11 & 1` inline ; `+0xcc0`/`0x39e0` ; `+0x1400`/`0x160` (tail call) |
| `0x1407ecd00` | les trois listes préfixées : masque, N octets, M mots | — | `CALL 1407ecd78` ; `[RBX+0x100]` → `1411b1a24` puis `0x108` sur `N<<3` ; `[RBX+0x908]` → `1411b198c` puis `0x910` sur `M<<5` |
| `0x1407ecd78` | écrit le masque de présence : cherche le rang du bit le plus haut (`MOV EAX,0x3f`, balayage arrière des 64 mots), écrit ce rang, puis ce nombre de bits un par un | — | désassemblage |
| **`0x1424ccf94`** | préfixe du masque : écrit `R8 - 1` sur **11 bits** | `0xb` | `LEA R9D,[R8 + -0x1] ; MOV ECX,0xb` |
| **`0x1411b1a24`** | longueur N de la liste d'octets | **12 bits** | `MOV ECX,0xc` |
| **`0x1411b198c`** | longueur M de la liste de mots de 32 bits | **8 bits** | `MOV ECX,0x8` (`ADD [RDX+0x2c],0x8`) |
| **`0x1407ece18`** | **écrivain de CHAÎNE UTF-16** : unités de 16 bits MSB d'abord, boucle jusqu'à `R9D` unités, **s'arrête APRÈS l'unité nulle** | 16 bits par unité | `MOVZX EDI,word ptr [RSI + R9*0x2]` ; `SHL R8,0x10 ; OR R8,RDI` ; `TEST DI,DI ; JNZ` |
| **`0x1407edaf4`** | écrivain de u32 **ÉTIQUETÉ** : `(writer, nom, valeur)`, le nom n'est pas sérialisé | **32 bits** | `MOV ECX,0x20` ; `DAT_143686818` = `"desired-representation"` |
| **`0x1407edcc4`** | u16 | **10 bits** | `MOVZX R9D,word ptr [R8] ; MOV ECX,0xa` |
| **`0x1407edd3c`** | u16 | **14 bits** | `MOV ECX,0xe` |
| **`0x1407eddb4`** | char signé, écrit **VALEUR + 1** (donc -1 représentable) | **6 bits** | `MOVSX R9D,R8B ; INC R9D ; MOV ECX,0x6` |
| **`0x1407ede30`** | octet | **7 bits** | `MOVZX R9D,byte ptr [R8] ; MOV ECX,0x7` |

**Le gamertag est à `sub+0xc14`** (16 unités au plus). Mesuré : 616/636 noms égaux au `roster[].name`
des documents de rejeu sur 76 films ; 44/44 sur les films témoins.

**Le second champ de nom est le bloc brut `sub+0x1400`** (44 octets = 22 unités UTF-16). Comme il
passe par `FUN_1406d60f4` (recopie de l'image mémoire), il est en UTF-16 **petit-boutiste**, là où
`FUN_1407ece18` produit du **gros-boutiste** : 640/640 blocs rendent le gamertag de leur
enregistrement en petit-boutiste, **0/44 en gros-boutiste**. Les deux conventions d'écriture se
voient donc directement dans les octets du film.

#### La structure de personnalisation, et le sérialiseur qui la nomme

| Fonction / donnée | Rôle | Preuve |
|---|---|---|
| **`0x140969c54`** | **sérialiseur DELTA de la MÊME structure de slot** : mêmes offsets `+0xc10`, `+0xc12`, `+0xc14`, `+0xc34`, `+0xc36`, `+0xc38`, `+0xc48`, `+0xcb0`, `+0xcb8`, même `FUN_1407ecd00` | désassemblage |
| **`0x1407ec27c`** | **sérialiseur CHAMP PAR CHAMP de la personnalisation**, appelé sur `sub+0xcc0` | `LEA RDX,[RSI + 0xcc0] ; CALL 0x1407ec27c` dans `140969c54` |
| — champs nommés | `variantName` `[0x4f]`, `styleName` `[0x50]`, `themeName` `[0x1cc]`, `coatingName` `[0x1cd]`, `actionPose` `[0x1ce]`, `model_region`/`model_permutation` (paires, comptées par `[0]`) | noms passés en clair à `FUN_1407edaf4` |
| `0x1407ebf44` | 24 attaches d'armure de `0x24` octets, base `cust+0x14C` | `0x14C + 24 × 0x24 = 0x4AC` |
| `0x1407eda5c` | 7 objets de `0x58` octets, base `cust+0x4AC` (variant, style, theme/coating/marker, 8 × region+perm) | `0x4AC + 7 × 0x58 = 0x714` |
| `0x143686770`..`0x1436868a0` | pool de chaînes : `unarmed`, `variantName`, `styleName`, `regionOverrideName`, `themeName`, `coatingName`, `markerName`, `desired-representation`, `variant-name`, `queued-replay-mission`, `unknown`, `region`, `permutation`, `model_permutation`, `model_region` | lecture mémoire |

**Fermeture arithmétique** : le plus haut indice touché par `FUN_1407ec27c` est `param_2[0x1ce]`
(`actionPose`), soit l'octet `0x738`, et `0x738 + 4 = 0x73C = 1 852` — **exactement** la largeur du
bloc brut que l'écrivain de film recopie (`0x39e0` bits). Les deux sérialiseurs décrivent donc la
même structure : **la personnalisation est bien dans l'enregistrement de slot du film.**

**MAIS elle y est VIDE** : mesuré sur 44 enregistrements de 6 films et 2 builds, **0 octet non nul
sur 81 488**. Le format la réserve, le flux enregistré ne la porte pas. Détail et verdict chiffré :
`.ai/V7.5/film_re/NOTE_SECTION3_SLOTS_2026-09-12.md`.

#### Ce que la phase 2 corrige du relevé par build

`TestD1Builds` classe les 1 351 `chunk_00` du cache en **7 builds** (13 groupes build/version/table),
et non deux comme la phase 1 le concluait. La carte d'en-tête de la section 8 vaut pour `HI_1_12_0`
et `HI_1_13_0` **seulement** : sur les builds antérieurs, l'offset de la chaîne de build recule de
`16 644` (`HI_1_11_0`), `16 648` (`HI_1_10_0`/`HI_1_9_0`/`HI_1_8_0`) ou `16 668` (`HI_1_4_1`) octets,
soit **un bloc de registre ECS (`0x4100 = 16 640`) plus 4 octets par entrée manquante de la table
par type** — trois fermetures sans ajustement. La longueur d'un enregistrement de slot y change
aussi, d'une **constante par build** (-2 880, -4 320, +1 600 bits), le XUID et le gamertag restant
lisibles partout (11 550/11 728 enregistrements).

### 8.3 L'ÉQUIPE D'UN JOUEUR : elle n'est pas dans `chunk_00`, elle est dans la trame (2026-09-12, phase 3)

La phase 2 avait fermé la question **par la négative** : aucun des champs courts de
l'enregistrement de slot de `chunk_00` ne porte l'équipe (huit des neuf sont constants sur tout le
corpus). La phase 3 la ferme **par le positif**, en suivant le CONSOMMATEUR (méthode, règle 1) :
l'équipe est une donnée **répliquée**, portée par un composant ECS de la trame d'état (paquets de
type 2), et non par l'en-tête du film.

**Le chemin, depuis la chaîne de nom du composant.** Le tableau des descripteurs de composant a un
pas de `0x50` (dix pointeurs), le champ de nom en `+0x08`, le SÉRIALISEUR en `+0x18` et le
DÉSÉRIALISEUR en `+0x30` — forme vérifiée sur deux entrées distinctes (elle explique la mention
« deser thunk (vtable+0x28) » déjà portée par `filmdec/components_team_mapping.go`, qui compte
depuis le champ de nom).

| Fonction / donnée | Rôle | Preuve |
|---|---|---|
| `0x143c953c0` | chaîne `"managed-player-team-designator-component"` | `search_strings` |
| `0x141177eb0` | thunk de nom (`LEA RAX,[chaîne] ; RET`) | xref de la chaîne |
| **`0x143d08ad0`** | **descripteur du composant** : nom en `+0x08`, écrivain en `+0x18`, lecteur en `+0x30` | xref du thunk ; lecture mémoire |
| **`0x142edbd3c`** | **écrivain (sérialiseur)** du composant | `descripteur + 0x18` |
| **`0x140f581e8`** | **lecteur (désérialiseur)** : un seul appel, `FUN_1407ef804` | `descripteur + 0x30` ; décompilé — concorde avec `ecs_table.tsv:228` |
| **`0x1407ef804`** | **primitive : lit 4 BITS et STOCKE la valeur MOINS UN** | `ADD dword ptr [RCX+0x2c],0x4` ; `SHR R9,0x3c` ; `DEC R9B` |
| `0x1445c0c00` | descripteur de l'énumération de script `mp_team_designator` (« Enum for MP team designators »), **9 entrées** à `0x144723da0` | lecture mémoire |
| `0x144723da0` | les 9 noms, dans l'ordre : `First`, `Second`, `Third`, `Fourth`, `Fifth`, `Sixth`, `Seventh`, `Eighth`, `Neutral` | lecture mémoire des 9 pointeurs |

**Le codage, et c'est une prédiction, pas une lecture.** `DEC R9B` fixe la convention : le champ de
4 bits porte `désignateur + 1`, donc `0` code **-1 = aucune équipe**. Le lecteur de la composante
globale le confirme par son test de validité (`INC AL ; CMP AL,0x9 ; JA` → domaine `-1..8`, soit
les neuf désignateurs plus « aucun »). Prédiction écrite avant toute mesure sur les films :
**`brut = team_id + 1`**.

**Vérifié sur les films.** Archétype **ti=9** (« managed-player »), composant **i0**, champ à
**186 bits du début du record d'image-clé** (décalage MESURÉ, pas porté : les largeurs de l'en-tête
par entité et du bloc d'état par défaut de ti=9 ne sont pas tranchées par le dossier). Corpus de
22 films (14 d'arène, 6 de Grande bataille, 2 de FFA) :

- **16 films sur 18 en accord EXACT** avec `match_participants.team_id`, **160/176 slots**, dont
  **24/24 deux fois** en Grande bataille (`03af54c3`, `213a87dc`) ;
- **le seul décalage de tout le record** qui satisfasse l'oracle : **1 sur 456/457 par film**,
  soit 0 faux positif sur 7 292 positions ; **0 touche sur 576** décalages voisins ;
- **témoin négatif naturel** : les deux films de FFA lisent `0` sur les huit entités — le moteur
  ne donne aucun désignateur en FFA, là où l'API fabrique un `team_id` par joueur ;
- **8 entités ti=9 par image-clé en arène, 24 en Grande bataille**, slots consécutifs de pas 2,
  longueur de record constante (459/460 bits selon le build) ;
- le désignateur bouge **si et seulement si** la suite des slots bouge (22/22 films) : il est
  stable par ENTITÉ, et c'est la réattribution de slot qui déplace l'appariement.

**Ce que la section 8.1 disait, et ce qui tient.** « Le gamertag et l'équipe ne sont dans aucun des
champs ci-dessus » : faux pour le gamertag (corrigé en 8.2), **vrai pour l'équipe**, et la phase 3
dit désormais où elle est à la place.

#### Le composant qui porte `team` dans son nom, et qui n'est PAS l'équipe d'un joueur

| Fonction / donnée | Rôle | Preuve |
|---|---|---|
| `0x143c985c0` | chaîne `"game-engine-team-mapping-component"` | `search_strings` |
| `0x143d0f7a0` | son descripteur (nom `0x141173050` en `+0x08`) | xref du thunk |
| `0x142f068bc` · `0x140f58200` | écrivain · lecteur — déjà portés par `filmdec/components_team_mapping.go` | `descripteur + 0x18` / `+ 0x30` |
| **`0x142f1b44c`** | **vidangeur de debug du masque de champs sales : il NOMME les six champs** dans l'ordre des bits — `team-mapping` (0), `shared-team-lives` (1), `current-state` (2), `game-finished` (3), `current-round` (4), `round-timer` (5) | décompilé (`FUN_14064d734(dest, "team-mapping:", 0x400)`) |

Son état fait 20 octets : six champs de 2 octets puis un tableau de **HUIT** octets signés, gaté
par le masque de `+0x06`, chaque entrée lue par `FUN_1407ef804` (4 bits, valeur-1) et mise à `-1`
si le bit est absent. **Huit entrées, pas trente-deux : c'est une table par ÉQUIPE, pas par
joueur**, et le vocabulaire de ses six champs est celui d'un composant global du moteur de jeu.

#### Le chunk de type 8 « PLAYER_METADATA » n'existe pas dans ce corpus

Mesure sur les **1 351 manifestes** du cache : les seuls types déclarés sont **1 (x1 351),
2 (x37 661) et 3 (x1 351)**. Aucun type 8, aucun type 12. La piste « le roster et les équipes sont
dans un chunk de type 8 » est donc **réfutée pour ce corpus**.

Détail, contrôles chiffrés, chemin actuel de la production et ses pertes :
`.ai/V7.5/film_re/NOTE_EQUIPE_FILM_2026-09-12.md`.
