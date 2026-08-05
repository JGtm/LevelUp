# Kill feed Halo — RE du mécanisme (findings consolidés)

> Doc dédié (le HANDOFF_FRAME_DECODER_L3 devenait trop gros). Couvre **comment le jeu
> produit le kill feed** `tueur · arme · headshot/parfait · victime`, établi par RE Ghidra
> + capture Cheat Engine en direct (film 000d5950, replay Theater offline).
> Outils CE : `tools/ce/filmdec_{deadstate,killfeed}_capture.lua`. Map pi→gamertag + narration
> utilisateur = vérité-terrain.

## Ce que le kill feed affiche (confirmé par l'utilisateur)
`tueur — arme — [modificateur] — victime`. Modificateur = **seulement** `headshot` ou `parfait`
(perfect) ; rien d'autre n'est affiché (ni distance, ni outre-tombe/posthume).

## Map joueurs (film 000d5950)
pi0 whiteknight2519 · pi1 JAVIERLOLITO540 · pi2 JGtm · pi3 LORD PEINX13 · pi4 IKE ILYA ·
pi5 Akatsuki fire17 · pi6 aldusbroncus · pi7 VitaminA1688.
Encodage d'un player-index dans les structs : **`0xE1500000 + idx*0x10002`** (idx 0-7), résolu
par `FUN_140e958c4` (tag de joueur). NB : l'**entity-id** (eid) d'un biped est en `0x4000xxxx`
et **change à chaque respawn** ; le **player-index** (idx) est stable par joueur.

## 1) object-dead-state de la victime (FUN_140c1dd44) — DÉCODÉ
Le wrapper `FUN_140c1dce0` (i11) lit `mort=R(1)` puis appelle la heavy form `FUN_140c1dd44`
(struct = composant `RDI+0x74`). Champs capturés en direct (hook après le CALL @140c1dd11) :
- **`+0x04` = player-index VICTIME** ; **`+0x08` = player-index TUEUR**. **Confirmé** contre la
  narration (JGtm←IKE 4:51 ; JGtm←aldus 5:36 ; LORD PEINX←aldus 5:44 plasma ; Akatsuki←JGtm 5:29
  BR75 ; aldus←LORD PEINX 5:43). ⇒ **tueur↔victime sont dans le dead-state, directement** (plus
  besoin de la jointure temporelle chunk_27).
- `+0x0c` = catégorie mêlée(0x40000)/distance-lancé(0x10001) ; `+0x14`/`+0x18`/`+0x38` =
  sous-type/modificateur (perfect=1 observé). **Données internes** : le kill feed n'affiche que
  headshot/parfait, pas la catégorie.
- `+0x10` = référence **résolue** par `GetLocalHandleFromGlobalId` (table `DAT_144b404f0`) →
  **CONSTANTE offline** (116963283 sur 25 morts) ⇒ inutilisable telle quelle. L'hypothèse
  historique "+0x10 = arme" est donc **fausse en l'état**.

### ⚠️ L'ARME N'EST PAS dans le dead-state stocké
Test décisif : **marteau** vs **marteau antigrav** (variante) = struct **identique**
(0x40000/2/5/7, même `+0x00`). **BR75** vs **sniper** = identiques sauf le bit modificateur.
⇒ aucun champ du dead-state ne distingue le modèle d'arme. (Réserve utilisateur : les variantes
"peuvent partager un bout de code" — donc ne pas conclure trop vite que l'arme est totalement
absente ; mais elle n'est pas un champ propre du dead-state.)

## 2) Constructeur d'entrée kill feed (FUN_14066b5e8)
```c
lVar5 = FUN_1404969f0(param_2);   // le DamageReport
iVar2 = *(int*)(lVar5 + 0x538);   // hook capture EBX ici
if (iVar2 == -1) return;
uVar4 = FUN_14049d198(iVar2);     // résolution affichage
FUN_1414ce150(*param_1) -> categorie (KillFeed=4)
... copie param_1[6] (tableau icône 0x100o) ...
FUN_1414ce4f0(iVar2, entry)       // push de l'entrée
```
Capture CE (hook @14066b62b) : **`[report+0x538]` = presque toujours 0** (17/18) → c'est un
**type d'événement**, PAS l'arme. La fonction est appelée **chaque frame** pour re-render les
entrées visibles (≈5 s à l'écran), d'où beaucoup de captures redondantes/vides.
La **seule ligne utile** révèle la structure d'une entrée : `[handle][victimeIdx][1][killerIdx][1]`
(ex. `[2231525701][idx5 Akatsuki][1][idx3 LORD PEINX][1]` puis `[1795313718][idx4 IKE][1][idx2 JGtm][1]`).
⇒ **le `handle` par entrée = candidat arme/icône** — NON validé (1 ligne ; paires hors narration).

## 3) Où est l'arme ? (pistes ouvertes — à approfondir EN PARALLÈLE)
- **A. `handle` par entrée du kill feed** (2231525701 / 1795313718 ci-dessus) : affiner la capture
  pour ne garder que les entrées non-vides (victime≠0) → beaucoup de lignes → croiser le `handle`
  avec les armes de la narration.
- **B. Global-id BRUT du dead-state** : `+0x10` capturé APRÈS résolution = constante ; le **R(32)
  brut** lu par `FUN_14080d6f0` (avant `GetLocalHandleFromGlobalId`) n'a jamais été capturé. C'est
  le champ que la RE pointait comme l'arme.
- **C. Le DamageReport (`FUN_1404969f0(param_2)`)** : `+0x538` = type d'événement ; l'arme/source
  de dégât est probablement à un AUTRE offset du report. RE de la structure du report.
- **D. `FUN_14049d198(iVar2)`** : ce que résout iVar2 (medal/icône ?).

## 4) BREAKTHROUGH — l'arme EST dans le film (event+0x538), lue au replay par FUN_1406730c4

Découvert par RE (3 agents parallèles) + capture CE validée. **`FUN_1406730c4` est le consommateur
kill feed actif AU REPLAY** (contrairement à `FUN_1406a6290`/`FUN_14066b5e8` qui ne capturent rien
en Theater) :
```c
R13 = FUN_1404969f0(event_id)        // composant d'event kill-feed (deserialise du FILM)
EBX = *(int*)(R13 + 0x538)           // HANDLE D'ARME (entite)  <-- l'arme
EAX = FUN_14049d198(EBX)             // resout (event-def) ; sautе si -1
... lVar12 = FUN_140495abc(EBX)      // resout handle -> COMPOSANT D'ARME
... FUN_1415cfa90(lVar12, ...)       // -> icone
R13+0x1f30 = entite attaquant
```
Le champ `+0x538` est **désérialisé du flux répliqué du film** (`FUN_1406cfb28`/`FUN_140cec0a0`,
lecteur big-endian réseau) → **présent au replay**. C'est exactement le modèle utilisateur :
l'événement porte l'arme (pas du held-weapon).

**Capture CE validée** (`tools/ce/filmdec_killweapon_capture.lua`, hook @0x673161/0x67316a, film
000d5950, 12 kills) — `event+0x04`=victime, `event+0x08`=tueur (idx `0xE1500000+idx*0x10002`),
`event+0x538`=**handle d'arme** :

| weaponHandle | tueur | arme (narration) |
|---|---|---|
| 3964796932 | JGtm | **BR75** (5:29) |
| 3964993546 | Akatsuki | **sniper** (5:08) |
| 3964928008 | IKE | **marteau** |
| 3965059084 | aldus | **plasma** (5:44) |
| 3964665856 / 3964731394 / 3965124622 | whiteknight / JAVIER / VitaminA | (à nommer) |

⇒ **tueur · arme · modificateur · victime reconstitué pour les 12 kills, depuis le film.**

## 5) PONT RÉSOLU (workflow RE statique wumaiev2d, 4 agents) — l'id64 d'arme complet

Chaîne tracée intégralement dans `FUN_1406730c4` (le jeu assemble lui-même l'id64) :
```c
iVar18 = *(int*)(lVar11 + 0x1f30);          // HIGH-32 (datum/def-handle de l'arme), lVar11=kill-event 0x1F40
def    = FUN_140477618({iVar18}, 0x1003);   // resout l'objet-DEFINITION typee (pool TLS+0x4acc0)
low    = *(int*)(def + 0x478);              // LOW-32 (def-id), fallback def+0x480 si -1 (FUN_1406713c4)
disc   = def+0x232 / def+0x233;             // 2 octets sous-discriminants (candidat variante/skin)
_local_53c = CONCAT44(iVar18, low);         // = ID64 catalogue (high|low) = cle analysis.WeaponIDToName
```
- **`FUN_14049d198(handle)` = résolveur d'EVENT-def** (≠ arme) → **-1 offline**, dead-end (réfuté).
- **`FUN_140495abc(handle)` = entité-arme générique (0x358)** ; `FUN_1415cfa90` = **faux ami** (poste
  juste l'event télémétrie `event_unit_killed_by_player`, n'lit pas l'arme). L'arme N'est PAS lisible
  sur cette entité 0x358.
- **L'id d'arme = `event+0x1f30` (high) + `def+0x478` (low)**, assemblé en `local_53c`.

### Variante (réserve du collègue) — TRANCHÉ
Catalogue `weaponv3/canon.go` + `canon_test.go` : **les variantes partagent le high-32** (famille) :
Gravity Hammer `0x841ac5e5` = {Gravity Hammer `…42c9679f`, Diminisher of Hope, Rushdown Hammer
`…d8d07ca1`} ; Energy Sword `0x4ff3937e` = {Duelist, Bloodblade}. ⇒ **gravity vs antigrav hammer =
même high-32, low-32 différent → il FAUT l'id64 COMPLET** (le high-32 seul = famille). Le collègue
avait raison. (`def+0x232/+0x233` = sous-discriminant runtime alternatif.)

### Offline (app) — le modèle utilisateur confirmé
Le kill-feed event porte une **RÉFÉRENCE explicite** vers l'**entité-arme** (`event+0x538` handle), dont
le composant **`'obje'`** (FUN_1405839d0(arme+0x2c,'obje'), deser FUN_14080c1f8) porte **`variant_name`
R(32) @ record+0x14 = high-32 (famille)** — exactement la donnée que le film-decoder décode déjà
(`rec.VariantName`, clé `analysis.WeaponIDToName`). C'est le **modèle que tu décrivais** (« une
référence vers l'arme rattachée à la mort »), pas une reconstruction.
- **Famille d'arme** (high-32) : récupérable offline du film via le `variant_name` du `'obje'`/WST.
- **id64 complet** (high|low, pour la variante) : le low-32 vient de la def runtime (pas offline via
  `FUN_140477618`) ; le film sérialise aussi l'id64 64-bit complet ailleurs dans le flux (voie déjà
  connue mais hors-périmètre kill-feed). Pour le kill feed, le high-32 (famille) suffit pour la
  plupart des cas ; la variante exacte nécessite l'id64 complet.

## Statut
- ✅ tueur + victime + modificateur (headshot/parfait) : décodés.
- ✅ **mécanisme arme entièrement cracké** : id64 = `event+0x1f30`(high) | `def+0x478`(low), assemblé
  par le jeu (local_53c). Famille = high-32 = `variant_name` du `'obje'` de l'entité-arme référencée.
- ✅ **variante tranchée** : variantes partagent high-32 ; id64 complet requis pour les distinguer.
- ⏳ **productionisation offline (app)** : décoder le `variant_name` (famille) de l'arme depuis le film
  (via le composant `'obje'`/WST) — dépend du spine ECS du film-decoder. CE non requis pour la suite.

## Réfs
Mémoires [[reference_killfeed_deadstate_fields]] (champs dead-state), [[project_kill_feed_frame_decoder]]
(⛔ interdits + but). Fonctions Ghidra : FUN_140c1dce0/FUN_140c1dd44 (dead-state), FUN_14066b5e8
(builder), FUN_1404969f0 (report), FUN_14049d198 (résolution), FUN_1414ce4f0 (push), FUN_14080d6f0
(R32 brut), FUN_14080d61c (GetLocalHandleFromGlobalId, table DAT_144b404f0).
