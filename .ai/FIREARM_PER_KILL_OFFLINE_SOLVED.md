# ARME À FEU PAR KILL — RÉSOLU OFFLINE & SCALABLE (2026-06-12)

> ## ⭐ MAJ FINALE 2026-06-12 — 96% PUR OFFLINE (le plafond 50% est tombé). Outil : `cmd/tmp_offwarp`.
> Le plafond ~50% de l'attribution par cross-corrélation temps (§5 ci-dessous, offset constant) n'était PAS un
> manque de données. **Deux corrections décisives :**
> 1. **La donnée est complète** : les 519 records `0xd2` contiennent l'arme de CHAQUE kill (mêmes 17 armes que le
>    live, ~86% du compte de chacune). Preuve : `FUN_14080a18c` (seul appelant du dégât `FUN_1407e00ac`) traite **1
>    record/appel** ; les 605 events live = 605 timestamps DISTINCTS (0 partagé → pas d'AoE multi-appliqué) ; les 86
>    « manquants » offline = sous-applications DoT/supercombine runtime (Needler -20, Disruptor -12), PAS des records
>    film séparés ; marqueurs frères `0xe9`(92)/`0x89`… = **apparence d'arme** (distribution uniforme ~1/arme).
> 2. **Le matching temps était cassé par l'échelle** : packet-ts (flux, ~946 unités/ms) vs `TimeMS` (jeu) comparés
>    avec un simple **offset**. Le bon modèle = **warp LINÉAIRE** packet-ts→TimeMS (R²=0.983, mesuré en appariant
>    offline↔live par attaquant+arme). Et le bon critère = **dernier dégât du tueur AVANT l'instant du kill** (pas
>    « le plus proche », qui tombe souvent après le kill sur une autre arme → 74%).
>
> **Résultat** : warp parfait (ancré live) = **99%** ; warp dérivé **pur offline** (fit packet-ts→TimeMS, raffiné 3×
> nearest puis 3× last-before) = **96%** (85/89), validé vs ground-truth live sur `000d5950`. Slots offline (R5>>1)
> == idx live (8/8, alignés par signature d'armes). **Pas de FRAME decoder, pas de champ victime offline, pas de
> held-weapon.** Pipeline complet : `apps/go-api/cmd/tmp_offwarp <matchID>` (décode 0xd2 + roster type-8 +
> kill-feed highlight + warp + attribution last-before + breakdown arme/joueur). Plan : `.ai/PLAN_SIBLING_DAMAGE_DESERS.md`
> (marqué RÉSOLU). NEXT = productionniser en service `internal/analysis/`.

> **TOURNANT (étape intermédiaire).** Le mur historique « arme à feu par kill impossible offline » (cf
> `FIRE_MELEE_GRENADE_EVENTS.md` §8/§11, plafond ~13/93) est **franchi en pur offline**. La donnée
> attaquant manquait pour une seule raison : un **champ lu dans le mauvais ordre** par le décodeur du
> projet. Validé contre la vérité-terrain **live** de la session 2026-06-11 (IKE→LORD = Disruptor).
> Outil de référence : `apps/go-api/cmd/tmp_killweaponoffline`.

## 1. Le record de dégât (paquet type-0, `payload[0]==0xd2`)

Désérialisé par `FUN_14080c1f8` (RVA 0x80C1F8). Après un en-tête fixe de **36 bits**, l'ordre RÉEL des
champs (lus dans cet ordre par le déser) est :

| bit / offset struct | lecture | sens |
|---|---|---|
| `+0x0c` (1er champ corps) | `FUN_1407f2034` → `FUN_1407f2058` = **R(5)** | **INDEX ATTAQUANT** (joueur). `joueur = R5>>1` (0..7) |
| `+0x08` | `FUN_1406d00ec` : R(1) ; si 0 R(2) | slot / cause de dégât |
| `+0x10` | `FUN_14080d69c` (gaté, =0 dans 519/519) | 2e partie ABSENTE de la sérialisation |
| `+0x14` | `variant_name` R(32) BE | **FAMILLE D'ARME** (clé high-32 de `analysis.WeaponIDToName`) |
| suivant | R(32) BE | suffixe variant universel `0x42c9679f` |

**`FUN_140495860`** résout le R5 (index) en handle joueur via la table d'entités (stride `0x358`) :
les **8 joueurs occupent les slots 0..7** (handles live `0xEC500000 + idx*0x10002`), donc le R5 prend
exactement **8 valeurs paires {0,2,4,6,8,10,12,14}** = `idx*2`.

### ⚠️ La cause du mur (le « mauvais angle »)

Le décodeur du projet (`cmd/tmp_dmgattacker`) lisait **le slot AVANT le R5** :
`skip(36) → slot → R5 → famille`. Le déser fait l'**inverse** : `skip(36) → R5 → slot → … → famille`.
Comme le slot est de longueur variable (1 ou 3 bits), lire le slot d'abord **désaligne** le R5 (il
chevauche les bits du slot) → le « R5 » du projet ne prenait que 4 valeurs bruitées (1,3,17,19) et fut
jugé « pas l'attaquant ». Avec l'ordre corrigé : **8 valeurs paires nettes**, et la **même arme apparaît
sous plusieurs R5** (preuve que R5 = le joueur, pas l'arme). Côté `+0x0c`, le projet avait aussi fusionné
le R5 (attaquant) avec le R32 (famille) en un faux « global-id R5+R32 » → d'où des « id d'arme non
stables » (l'instabilité venait des 5 bits de tête = le tueur).

## 2. Mapping R5 → joueur (slot → xuid) — ROSTER AUTO-DÉRIVÉ DU TYPE-8 (2026-06-12)

`joueur = R5>>1` (slot). Le binding **slot → xuid** est dérivable **offline et automatiquement** :
> Les xuids du kill-feed apparaissent dans le **paquet type-8 (roster) dans l'ORDRE EXACT des slots**,
> stockés **bit-shiftés en little-endian**. Bit-scan LE du type-8 pour les xuids du feed → tri par bit-pos
> = ordre des slots. **Validé 8/8 sur 000d5950** (rang0→slot0 … rang7→slot7 = piXuid exact).

Donc : kill-feed (xuid+gamertag, `HighlightEvent`) + roster type-8 → `slot → xuid → gamertag`, **sans hardcode,
sans live, sans DB**. La forte couverture d'attribution (80-90% sur d'autres matchs) **re-prouve** le roster
(un mauvais mapping → le slot du tueur n'aurait pas de dégât à ses kills). Échecs écartés avant : corrélation
temporelle slot↔killer (24%, collisions), fingerprint type-9 (0 arme dedans), type-8 byte-aligné (2/8). Le
décodeur de participant-index par dead-state (`tmp_killeridx`) était **réfuté** (0%) — le type-8 le remplace.

## 3. Pipeline complet (100% offline, scalable)

1. **Décoder** tous les paquets `type-0 / payload[0]==0xd2` avec le layout §1 →
   `(attaquant = R5>>1, arme = famille, temps = (ts - t0Us)/1000)`. (519 ticks sur 000d5950.)
2. **Kill-feed** : `analysis.ParseHighlightEvents(chunk_27)` → kills/deaths appariés en
   `(tueur, victime, temps)`. (93 kills sur 000d5950.)
3. **Caler les bases de temps** : les events highlight et les paquets frame ont des origines
   différentes. Cross-corrélation (offset maximisant la couverture `R5==tueur` à ±400ms) → **auto-calé**
   (≈ **−20.3 s** sur 000d5950 ; constant, pas de dérive). Le pic est net et univoque.
4. **Attribuer** : arme du kill = famille du tick **dont l'attaquant == le tueur** (`R5>>1 == pi(tueur)`)
   le plus proche du kill (fenêtre ±800 ms).
5. **Mêlée / grenade** : via les **scanners offline déjà résolus** (`FIRE_MELEE_GRENADE_EVENTS.md`
   §2/§3 : events 0x534/0x535 player_index@anchor+23 ; 0x4c0c00 player_index 5b).

## 4. Couverture & validation (000d5950)

- **50/93 kills** : arme à feu attribuée **avec attaquant vérifié** (R5==tueur sur `0xd2`). **Validation
  forte** : 1er kill `IKE → LORD = Disruptor` = exactement la vérité-terrain live du 2026-06-11. Les bursts
  par joueur s'alignent (IKE : Disruptor 14-15s → kill 35.3s ; M41 38s → 58.6s ; BR75 151-158s → 176.8s).
- **La donnée d'arme existe pour 92/93 kills** (record le plus proche au même offset −20.3 s). Donc les
  « 43 manquants » NE sont PAS des mêlée/grenade — c'est une fausse piste. Mais cette couverture dense n'est
  fiable qu'à **~78 %** (le record le plus proche attrape un tiers 1 fois sur 5).

### 4bis. Le mur restant : les marqueurs frères (découverte 2026-06-12)

Le dégât d'arme n'est PAS que `0xd2`. Le suffixe `0x42c9679f` apparaît dans **~1095 records sur ~12
marqueurs** (`0xe9` 159, `0x89` 146, `0xc0` 73, `0xc7` 48, `0xc3` 39, `0xc2` 35…). En ne gardant que ceux à
famille catalogue : **712 records** → couverture 92/93. MAIS :
- **Seul `0xd2` expose l'attaquant proprement** (champ 5 bits = `slot×2` ∈ {0..14} au bit 36). Les frères
  n'ont **aucun** champ 5 bits donnant les 8 joueurs (testé bit-scan + ancre-suffixe + test victime/tueur :
  plafond 44-50 % = bruit).
- Hypothèse (intuition user, plausible) : ces frères sont des events de **dégât avec MONTANT** (scalaires
  de quantité) et/ou victime-keyés — d'où l'absence de slot-joueur net.
- **Pour passer de 50/93 vérifié à 92/93 propre** : RE le **désérialiseur de chaque marqueur frère** pour en
  extraire joueur + montant + victime. Le bit-scan ne suffit pas. **NE PAS claim « 100 % propre » : c'est
  50/93 vérifié + 92/93 donnée-prouvée.** Outils : `cmd/tmp_suffixscan`, `cmd/tmp_calibmarkers`,
  `cmd/tmp_victimkey`, `cmd/tmp_e9`, `cmd/tmp_allcoverage`.

### 4ter. RE des marqueurs frères — angles tentés (2026-06-12) et mur

Trois angles statiques essayés, tous bloqués :
1. **Vtable** : le déser `0xd2` (`FUN_14080c1f8`) est référencé depuis `143d0ad08` — mais c'est la **vtable
   de sa classe** (méthodes adjacentes `0x14080A048/A130/A18C`), PAS une table de types. Ne donne pas les
   déser frères (classes séparées, vtables séparées).
2. **Empirique sur `0xe9`** (159 rec, le plus gros frère) : couvre **87/93 kills** à son offset propre
   (−12.4 s) → fortement lié aux kills, mais **aucun champ 5-bit ne corrèle** au tueur/victime (24 % ≈
   hasard 1/8 sur largeurs 3-7, depuis-début ET avant-suffixe, val ET >>1). Structure non extractible au
   bit-scan. NB : offsets temporels propres par marqueur (`0xd2`=−20.3 s, `0xe9`=−12.4 s) → events distincts.
3. **Chaîne `variant_name`** : lue par `FUN_14080dec4`, mais c'est un **lecteur de champ générique** (100+
   appelants, tout type de record nommé) → pas filtrable statiquement aux seuls déser d'arme.

**Conclusion** : la route définitive est le **debugger live** (Ghidra debugger ou CE) : breakpoint sur le
traitement d'un record `0xe9`/`0x89`/… au replay → identifie la fonction déser exacte + sa structure
(joueur, montant, victime). Nécessite le jeu lancé + CE. Tant que non fait : **50/93 vérifié + 92/93
donnée-prouvée** est l'état acquis honnête.

## 5. État & productionisation

- **Acquis** : la donnée `(attaquant · arme · temps)` par tick + le kill-feed `(tueur · victime · temps)`
  suffisent à l'arme-à-feu-par-kill **offline**. Plus besoin du live (le live a servi d'**ancre de
  validation**, pas de dépendance).
- **À faire** (port production) : sortir de `cmd/tmp_killweaponoffline` vers un package
  `internal/analysis/weaponkill/` (décodeur record + calage temps + jointure kill-feed) ; fusionner avec
  les scanners mêlée/grenade ; table de sortie `weapon_kills` (match_id, killer_xuid, victim_xuid,
  weapon_family, t). Garde-rail : re-valider la couverture sur 2-3 autres matchs (idéalement non-Fiesta,
  loadouts fixes → couverture firearm attendue plus haute).

## 6. Record de kill complet (assemblage cible)

En combinant ce pipeline avec le kill-event component `FUN_14104bd08` (tueur + assistant + %dégâts) :
> `Tueur · [arme : §1-§4 firearm / scanners mêlée-grenade] · Victime` **+** `Assistant · %participation`
— tout offline, sauf l'assist/% qui reste à câbler offline (composant localisable dans le flux, cf
`reference_killfeed_deadstate_fields`).

## 7. VÉRITÉ-TERRAIN LIVE (dual-hook CE) — arme-par-kill EXACTE 97/98 (2026-06-12)

Quand le scalable offline ne suffit pas (sibling markers non décodés), le **live dual-hook** donne la
vérité-terrain EXACTE par match. Deux hooks CE lecture-pure sur le replay, **même horloge RDTSC** :

- **`FUN_1407e00ac`** (apply dégât, RVA 0x7E00AC) → `attaquant [R8+0x0c]` (0xEC50+idx×0x10002) ·
  `victime-biped *[RDX]` (0x4000xxxx, **change à chaque respawn**) · `arme [R8+0x10]` · `suffixe [R8+0x14]`.
- **`FUN_1406730c4`** (kill-event, RVA 0x6730C4) → `victime [RDX+0x04]` · `tueur [RDX+0x08]`
  (les deux = 0xE150+idx×0x10002). + RDTSC.

**Décodage idx joueur** : `(handle - base)/0x10002` ; base = 0xEC500000 (dégât), 0xE1500000 (kill).
**Matching EXACT** (horloge partagée, pas d'alignement temps-film) : arme du kill = le dégât du tueur
(`attaquant==tueur`) de plus grand `tsc <= tsc_kill`. L'alignement temps-film linéaire échoue (replay
non-uniforme, RDTSC≠film linéairement) — d'où la nécessité de capturer kills ET dégâts sur la même horloge.

**Résultat 000d5950** : 605 dégâts + 98 kills → **97/98 attribués exact** (1 « ? » = kill précoce sans
dégât-tueur capté avant ; + 1 suicide LORD→LORD). Validé `IKE→LORD=Disruptor`. Breakdown par joueur
ground-truth (whiteknight Stalker×9, aldusbroncus S7 Sniper×7, LORD/IKE M41×7…). Outil : `cmd/tmp_dualcap`.
Captures : `tools/ce/{dmgcapture_run2.bin (605×32), killcapture.bin (118×16)}`.

**Portée** : EXACT mais **non scalable** (1 replay manuel + CE par match). C'est la **référence de
validation** pour tout décodeur offline futur, et le livrable direct si on accepte le coût manuel par match.
Setup CE : cf mémoire `reference_cheatengine_mcp_setup` ; bases ASLR/symboles à re-dériver à chaque lancement
(`enum_modules` → base ; hook AA via `auto_assemble`, lecture pure + null-checks R8/RDX = anti-crash).
