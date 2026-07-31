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
