# Kill feed / Death-recap — inventaire RE Ghidra (2026-06-10)

> Source : RE Ghidra (HaloInfinite.exe, projet HI) — strings + xrefs + décompiles.
> But : recenser TOUTES les données tueur/victime/assistant/arme attachées à un kill,
> et distinguer ce qui est **FILM-NATIF** (sérialisé, lisible offline) de ce qui est
> **LIVE-DÉRIVÉ** (calculé au replay par le moteur, plante offline).

---

## 0. Principe directeur (correction utilisateur 2026-06-10)

Un format de replay est **100% déterministe** : il n'y a **AUCUN aléatoire**. Quand notre
décodeur sort des valeurs incohérentes, ce sont des valeurs **déterministes lues au mauvais
offset bit** (désalignement), pas du bruit. Le « standard » existe forcément ; le travail =
caler l'alignement. Corollaire : ne jamais conclure « c'est aléatoire / pas exploitable » —
conclure « notre framing bit est faux à tel endroit ».

---

## 1. Le death-report télémétrie (`FUN_140a24f6c`) — property bag complet

`FUN_140a24f6c` construit le sac de propriétés CELL/télémétrie d'un kill (death-recap).
Champs (ordre du code), avec provenance :

| Champ | Source dans le code | Film-natif ? |
|---|---|---|
| `Disposition` | dérivé `*(report+0xc)` (1/3 → outcome) | dérivé |
| `VictimWorldLocation` / `WorldLocation` | `FUN_140a2546c(victim)` → position | **LIVE** (résolveur) |
| `VictimObjectDescriptionSessionIndex` | `FUN_140a2546c(victim)` | **LIVE** |
| `VictimObjectIndex` | idem | **LIVE** |
| `VictimSessionMatchIndex` | idem (`local_240`) | **LIVE** |
| `VictimAgent` | `local_286` (participant byte) | **film** (index participant) |
| `KillerWorldLocation` / `WorldLocation` | `FUN_140a2546c(killer)` (`local_210`) | **LIVE** |
| `KillerObjectDescriptionSessionIndex` | `FUN_140a2546c(killer)` | **LIVE** |
| `KillerObjectIndex` | idem | **LIVE** |
| `KillerSessionMatchIndex` | idem (`local_270`) | **LIVE** |
| `KillerAgent` | `local_284` (participant byte) | **film** (index participant) |
| `AssistingPlayerSessionMatchIndices` | boucle `param_5` → `local_a0[]` (jusqu'à 12) | **LIVE** (résolus) ; mais les **handles** assistants sont film-natifs (cf §3) |
| `DamageSourceObjectDescriptionSessionIndex` | `*(report+4)` | dérivé d'un index live ; **mais** le global-id source est film-natif (§2) |
| `DamageEffectDefinitionGlobalTagId` | `*(DAT_14494a908+0x78) + (handle>>0xd)*0x34 + 4` | **LIVE** (table runtime) |
| `DamageReportingModifier` | `uVar18` (disposition→0/1/2) | dérivé |

`FUN_140a2546c` (résolveur Killer/Victim → object/session-index) appelle **`FUN_140495abc`** =
résolveur **LIVE** qui plante offline (tables runtime nulles au repos). Donc **toute la colonne
« LIVE » n'est PAS reconstructible offline par ce chemin**.

## 1bis. Struct death-recap UI / stats (régions strings 143c72xxx / 143c73xxx / 143ca2xxx)

Champs nommés du recap (post-game / UI), calculés côté moteur (donc **LIVE-dérivés**, pas des
champs film bruts) :

| String | Adr | | String | Adr |
|---|---|---|---|---|
| `KillerWeapon` | 143c72830 | | `KillerEmblem` | 143c73d40 |
| `KillerGamerTag` | 143c72840 | | `NormalizedKillerShields` | 143c73d88 |
| `NormalizedKillerLife` | 143c73af8 | | `KillerPlayerSeqId` | 143ca2a18 |
| `KillerServiceTag` | 143c73b40 | | `KillerLatencyCompensation` | 143ca2a30 |
| `KillerTeam` | 143c73b68 | | `KillerWorldLocation` | 143697090 |
| `KillerPercentageDamageDone` | 143c73ce0 | | `KillerObjectDescriptionSessionIndex` | 1436970a8 |
| `KillerTeamColor` | 143c73d10 | | `KillerObjectIndex` | 1436970d0 |
| `KillerTeamIndex` | 143c73d20 | | `KillerSessionMatchIndex` | 1436970e8 |
| | | | `KillerAgent` / `DamageReportingModifier` | 143697100 / 143697120 |
| `prop_killer_player_configuration` | 14381ad40 | | `KillEffectGroup` | 143c2bc98 |

**Note `KillerWeapon`** : c'est un champ **dérivé** (calculé du damage-source via le catalogue
d'assets live), pas un champ sérialisé tel quel dans le film. La donnée brute sous-jacente
(global-id de la source de dégât) EST film-native (§2) ; `KillerWeapon` = sa résolution.

---

## 2. Composant `object-dead-state` — FILM-NATIF (sur le biped)

- Répliqué dans le film sur le biped : `FUN_142f07064` → `if (typeIndex==0x23||0x28) FUN_143203460(entity+0x74, stream)`. **0x23 = 35 = BIPED**.
- **Deser** : `FUN_140c1dd44` · **Ser** : `FUN_143203460` (confirme les noms de champs).
- Grammaire bit-exacte (vérifiée, `FUN_1406d310c(N)` = `ceil(log2(N))`, donc `(10)`→**4 bits**) :

```
present(1) [wrapper FUN_14080d69c] -> anim-handle: si bit, R(32)
R(8) tag        [FUN_140c1e3f0, NON stocké]
+0x04 victim-absolute-participant-index : R(1) present ; si 0 -> R(5) ; sinon -1   [FUN_1407f2058]
+0x08 KILLER-absolute-participant-index : R(1) present ; si 0 -> R(5) ; sinon -1   <-- LE TUEUR
+0x0c : R(4)
+0x0e : R(3)
+0x10 bloc damage-source :
        presentA R(1) ; si set :
          gidPresent R(1) ; si set : GlobalID = R(32) BRUT  <-- LA SOURCE DE DÉGÂT (arme)
          +0x14 R(3) ; +0x18 R(1)+[R(6)]
... (suite : +0x38 R(4), position +0x20, courbe +0x2c, handle final +0x4c)
Mort (+0x70) = R(1) lu AVANT ce bloc (consumeObjectDeadStateBiped)
```

- **Le `+0x10` est le global-id de la DÉFINITION (tag/object-description) de la source de dégât**
  (threads 2/3/5 convergents). Lu BRUT par `FUN_14080d6f0` **avant** la résolution.
  ⚠ Distinction clé : en **RAM** le struct `+0x10` stocke le **handle résolu**
  (`FUN_14080d61c` GetLocalHandleFromGlobalId) — c'est CETTE valeur RAM qu'une vieille capture CE
  avait mesurée **constante** (116963283). Dans le **FILM**, la valeur sérialisée est le global-id
  **brut** (≠ le handle résolu). Donc « +0x10 constant » mesuré en RAM **ne prouve pas** que le
  global-id film est constant. **Variance du global-id FILM par arme = NON ENCORE testée** (maillon
  make-or-break, cf §5).
- **Résolution offline** : NE PAS rejouer `FUN_14080d61c`/`FUN_140821f44`/`FUN_140712e10` ('obje')
  — leurs tables (`DAT_144b404f0`, `DAT_144eae7b8`, `DAT_14494a908`) sont **nulles au repos** =
  live-only. À la place : mapper le global-id brut → famille via un **catalogue d'assets statique**
  (type `metadata.duckdb:weapon_labels` / `analysis.WeaponIDToName`), identique pour tous.

Implémentation Go existante : `internal/analysis/filmdec/components_object.go`
(`DeadState{EnumA=+4 victime, EnumB=+8 tueur, GlobalID=+0x10}`, `consumeObjectDeadStateBiped`).
Grammaire = bit-exacte (vérifiée). ⚠ Étiquetage à corriger : `EnumA/EnumB` = victim/KILLER
participant-index (PAS « world datum-handles »).

### ⚠️ RÉCONCILIATION avec la vérité-terrain CE (mémoire `reference_killfeed_deadstate_fields`)

Une capture **Cheat Engine** antérieure (hook après `FUN_140c1dd44`, struct RDI+0x74, 25 morts) a
mesuré **+0x10 = CONSTANT (116963283)** et conclu **+0x10 ≠ l'arme**. Le dead-state encode
tueur·victime·**catégorie** (+0x0c : mêlée=0x40000, lancé=0x10001) + modificateur, **PAS le modèle
d'arme** (marteau == marteau antigrav sur toute la struct). ⇒ **l'hypothèse de ce tour « +0x10 =
global-id d'arme exploitable » est PROBABLEMENT FAUSSE** : la mesure CE (RAM, handle résolu) prime
sur l'inférence RE statique. La seule échappatoire théorique non testée = le global-id **brut du
film** (≠ handle RAM) varierait là où le handle résolu est constant — peu probable vu que le
dead-state ne distingue déjà pas les modèles. **Ne pas investir le dead-state pour l'arme.**

## 2bis. La VRAIE source d'arme (vérité-terrain CE) = kill-feed-event `event+0x538`

Per mémoire `reference_killfeed_deadstate_fields` (12 kills validés CE) :
- L'arme = **handle du composant kill-feed-event** à `event+0x538`, **désérialisé du flux répliqué
  du film** (`FUN_1406cfb28`/`FUN_140cec0a0`), lu au replay par `FUN_1406730c4`.
- `FUN_1406730c4` assemble l'id64 : `high = *(int*)(killEvent+0x1f30)` (famille) ;
  `def = FUN_140477618({high},0x1003)` ; `low = *(def+0x478)` ; `id64 = high|low` = clé
  `analysis.WeaponIDToName`.
- **Famille = high-32 = `variant_name` R(32) du composant `'obje'` de l'entité-arme** (deser
  `FUN_14080c1f8`) — **déjà décodé** par le film-decoder comme `rec.VariantName`.
- Validé CE : 3964796932=BR75, 3964993546=sniper, 3964928008=marteau, 3965059084=plasma.

**Blocage OFFLINE (sans CE)** : relier le **handle d'arme du kill-event** → **entité-arme** dont
on lit `'obje'.variant_name`. Ce lien handle→entité passe par les tables runtime live-only
(`FUN_140495abc` → entité 0x358, `DAT_144eae7b8`/`DAT_14494a908` nulles au repos, cf §2). C'est
LE mur offline pour l'arme : la famille est décodable (`rec.VariantName`), mais l'appariement
kill→entité-arme exige la résolution de handle live. Voir [[project_kill_feed_frame_decoder]].

---

## 3. Composant `kill-event` (victim/killer/assistant handles) — FILM-NATIF

- **Deser** : `FUN_14104bd08` · **Ser** : `FUN_142f18fd0` (écrit les noms via `FUN_142b549c0`).
- Grammaire (entièrement offline, vérifiée sur le deser) :

```
param_3[0] victim-participant-handle    : R(1)+[R(5)]  (FUN_1407f2058 -> FUN_14049746c -> FUN_140e958c4)
param_3[1] killer-participant-handle    : R(1)+[R(5)]
param_3[2] R(32) brut                   (scalaire ; PAS une arme)
param_3[3] bool R(1)                    (flag ; ex melee/headshot ?)
param_3[4] ASSISTANT-participant-handle : R(1)+[R(5)]   <-- assistant par kill, demandé
param_3[5] R(32) brut                   (scalaire)
tail (param_3+6) conditionnel (flags DAT_1451789b8 / DAT_145178a48) -> FUN_1431eb378
                 (2-3×R(32)+R(4) = séquence/timestamp/contexte ; PAS une arme)
```

- Strings ser : `victim-participant-handle`=143c988a8, `killer-participant-handle`=143c98888,
  `assistant-participant-handle`=143c98868.
- **VERDICT arme** : ce composant **ne porte AUCUNE référence d'arme** (param_3[2]/[5] = R(32)
  bruts non résolus). Pour l'arme → rester sur le dead-state §2.
- **VERDICT assistant** : victim + killer + **assistant** sont lisibles offline (3 index
  participant). C'est la source pour « assistant-participant-handle par kill ».
- **OUVERT** : sur quelle entité/archétype ce composant est répliqué (descripteur accédé par
  offset calculé, tables-pointeurs `145481740`/`1440a112c`/`1440a1138`, pas de xref nom direct).
  À localiser empiriquement dans le flux (scan de records dont [victim,killer] décodent vers un
  kill connu de chunk_27) OU via le descripteur. Faux ami écarté : `FUN_142f50e60` = RPC
  `server-acknowledge-kill-playback-started` (registrar `FUN_140f1ba60`).

---

## 4. Le mur réel : ATTEINDRE le composant bit-exact (pas la grammaire)

La grammaire des deux composants est **résolue et bit-exacte**. Le blocage est en **AMONT** :
pour lire le dead-state (composant i11 du biped) il faut consommer bit-exact TOUS les composants
present avant lui dans le record delta (mask `FUN_1406d7610` + i0 position-precision à largeurs
runtime `DAT_144632be0`/`DAT_1445cc9e0`, vitality, etc.).

Preuve empirique du désalignement (sonde `cmd/tmp_killeridx`, slot 519) : le flag `Mort` se
déclenche sur **201 onsets répartis ~toutes les 2 s** alors que le joueur du slot 519 ne meurt
que **~9×**. `EnumB` ne bijecte pas le tueur. ⇒ le composant dead-state est lu à des positions
bit fausses (déterministes, pas aléatoires). `DesyncAt==-1` (« traversée propre ») est un **FAUX
signal** : le décodeur termine le record à une frontière plausible tout en étant désaligné au
milieu (vraisemblablement mask ou i0 runtime-widths).

---

## 5. Expériences décisives (ordre, révisé après réconciliation CE)

1. **Arme — le vrai chemin (§2bis), pas le dead-state.** Le maillon offline = relier le handle
   d'arme du kill-event → l'entité-arme dont `'obje'.variant_name` (= famille high-32) est déjà
   décodé (`rec.VariantName`). Deux sous-pistes : (a) trouver le handle d'arme parmi les champs
   sérialisés du kill-event (param_3[2]/[5] de `FUN_14104bd08` ? — Thread 4 les disait scalaires,
   à re-vérifier vs `event+0x538`/`FUN_140cec0a0`) ; (b) reconstruire la table handle→entité
   depuis les spawns NEW du film (chaque entité-arme porte son global-id + son `'obje'`).
2. **Localiser le composant kill-event §3** dans le flux (scan + validation vs chunk_27 94/94) →
   victim/killer/**assistant** par kill, offline. C'est la donnée la plus proche d'être livrable.
3. **NE PAS** investir le dead-state pour l'arme (CE : +0x10 constant, ne distingue pas le modèle).
   Le dead-state reste utile pour tueur·victime·**catégorie** (mêlée/lancé) SI on cale le reach §4.
4. **Caler le reach §4** (mask / i0 runtime-widths, capture CE comme le `world_dump`) si on veut le
   dead-state fiable (catégorie + cross-check tueur). Secondaire.

## 6. Récap : ce qu'on PEUT déjà sortir offline aujourd'hui

| Donnée | Source | Statut |
|---|---|---|
| tueur, victime, time_ms par kill | chunk_27 highlight events | **94/94 prouvé** |
| medalType (proxy méthode) | chunk_27 | prouvé |
| killer/victim/**assistant** index par kill | composant kill-event §3 | grammaire OK ; **reste à localiser** dans le flux |
| catégorie de mort (mêlée/lancé) | dead-state +0x0c §2 | grammaire OK ; bloqué par reach §4 (faux `Mort`) |
| arme par kill (famille) | kill-event weapon-handle → `'obje'.variant_name` §2bis | famille décodable (`rec.VariantName`) ; **bloqué par lien handle→entité (live)** |
| ~~arme via dead-state +0x10~~ | ~~dead-state~~ | **ABANDONNÉ** (CE : +0x10 constant ≠ arme) |

## 7. Localisation empirique du composant kill-event (tests « œil neuf » 2026-06-10) — TOUS NÉGATIFS

Objectif : trouver les records kill-event (victim/killer/**assistant**) dans le film SANS décoder
le flux ECS bit-exact, en validant contre la vérité-terrain (93 kills chunk_27, roster DB). Sondes :

| Sonde | Hypothèse | Résultat |
|---|---|---|
| `cmd/tmp_killevtscan` | liste contiguë de records (grammaire R(1)+R(5)×3 + R(32)×2) | run max **7** = bruit (chance) sur les 28 chunks → **pas de liste contiguë** |
| `cmd/tmp_killevttime` | records distribués par tranche de temps (frames au moment du kill) | **201 candidats/kill**, idx dominant identique (idx0/idx4) pour TOUS, 0 bijection idx→joueur → **noyé dans le flux per-frame** (grammaire 5-bit trop permissive : 29257 frames) |
| `cmd/tmp_assistprobe` | assists = events medal de chunk_27 | total assists DB **17** ≠ tout `medalType` ; `raw_json` sans champ assist → **assists absents de chunk_27** |
| `cmd/tmp_xuidcluster` | records riches multi-xuid (64b) dans le paquet type-9 | occ xuid = nb highlight events **exact** par joueur, **0 cluster** → type-9 = SEULEMENT les highlight events (1 xuid/event), **aucun record killer+victime+assistant en xuid** |
| `cmd/tmp_dbcheck` | arme/assist déjà en DB | `killer_victim_pairs` SANS weapon_id (schéma skill périmé) ; assists = totaux only → **rien de plus en DB** |

**Structure du film confirmée** : chunk_00 = registre (1.97 Mo inflaté), chunk_01 = keyframe,
chunks 02-25 = frames gameplay (tranches ~20 s), **chunk_26 = mini-segment auto-contenu**
(keyframe type-2 + full-state type-1 + roster type-8 + 481 frames, ts ≈ 480 s = fin/re-sync),
**chunk_27 = 1 paquet type-9** (714 Ko : highlight events bit-packés, xuids LE). 28 chunks = film
complet de ce match.

**CONCLUSION (boundary nette)** : l'assistant-par-kill et l'arme-par-kill existent dans le film
UNIQUEMENT comme **petits composants ECS répliqués** (index participant 5-bit, handle d'arme)
enfouis dans le flux per-frame. Ils **ne sont PAS extractibles par scan/pattern-matching** (le
xuid n'y est pas ; la grammaire 5-bit est trop permissive face au flux). Seul le **décodeur ECS
per-frame bit-exact** (mur du reach §4) les rend lisibles — ce qui exige les largeurs de
quantification runtime (capture CE type `world_dump`). **Extractible par scan aujourd'hui = les
highlight events** (killer/victime/temps 93/93 + médailles). Tout le reste = derrière le décodeur ECS.
