# FIRE / MELEE / GRENADE EVENTS — L'ATTAQUANT EST DANS LE FILM (recette de décode)

> **DOC INOUBLIABLE — 2026-06-07.** À lire AVANT toute action « source par kill ». Ce fichier renverse
> la conclusion historique « pas d'attaquant offline ». Source de vérité pour la VOIE ROUVERTE
> (attribution par kill via le PLAYER INDEX porté par les events de tir/mêlée/grenade).
> Branche : `feat/weapon-attribution-v3` (worktree `.claude/worktrees/weapon-attribution-v3`).
> Compléments : `KILLFEED_STATE.md` §0 (vérité validée records de dégât) + `WALK_PORT_NOTES.md`
> (historique walk + position-linking réfuté + section TOURNANT/CALIBRATION).

> ⭐⭐⭐ **2026-06-12 — ARME À FEU PAR KILL RÉSOLUE *OFFLINE* & SCALABLE → voir `.ai/V7.5/killweapon/FIREARM_PER_KILL_OFFLINE_SOLVED.md`.**
> Le record de dégât (`0xd2`) porte l'**attaquant** en clair = un **R5 (5 bits) au bit 36, lu AVANT le slot**
> (le décodeur du projet lisait le slot d'abord → champ désaligné → fausse conclusion « pas d'attaquant »).
> `joueur = R5>>1` (8 valeurs paires {0..14}), mapping identité (ancre live IKE=slot4=pi4). Pipeline : décoder
> 0xd2 (attaquant·arme·temps) + kill-feed chunk_27 + cross-corrélation des bases de temps → arme du kill = tick de
> l'attaquant=tueur le plus proche. **50/93 kills firearm attribués offline, validé IKE→LORD=Disruptor (= vérité
> live).** Le live n'est plus qu'une **ancre de validation**, PLUS une dépendance. Outil : `cmd/tmp_killweaponoffline`.
> La bannière live ci-dessous reste l'historique du déblocage (utile, mais supplantée par l'offline).

> ⭐⭐ **DÉBLOCAGE 2026-06-11/12 — L'OPTION (3) DEBUGGER GROUND-TRUTH A MARCHÉ : ARME-À-FEU-PAR-KILL OBTENUE LIVE.**
> Le mur « arme à feu par kill » (§8/§11, plafond offline ~13/93) est franchi par la **voie (3) du §12** (capture
> live au replay), via le bridge **Cheat Engine-MCP** (cf mémoire `reference_cheatengine_mcp_setup` + le détail RE
> dans `reference_killfeed_deadstate_fields` § « MÉCANISME GROUND-TRUTH QUI MARCHE »). Récap :
> - **Le BON handler = `FUN_1407e00ac`** (apply du dégât, REPLAY-ACTIF — `FUN_1406730c4` résolvait `high` en
>   owner/équipe, fausse piste). Hook LECTURE PURE à l'entrée (RVA 0x7E00AC) : **attaquant = `*(param_3+0x0c)`**
>   (`0xEC50_0000+idx*0x10002`), **victime = `*param_2`** (biped), **arme id64 = `param_3+0x10` (high-32 FAMILLE) |
>   `param_3+0x14` (low-32 0x42c9679f)**. VALIDÉ : IKE 1er kill = 0x84BD29ED|42c9679f = Disruptor (= catalogue + narration).
> - **Marqueur finishing-blow = `FUN_1406730c4`** (kill-event, RVA 0x6730C4, replay-actif) : tueur `[event+0x08]` /
>   victime `[event+0x04]` (`0xE15x+idx*0x10002`). Entrelacé avec les ticks → arme du kill = dernier tick du tueur avant.
> - **RÉSULTAT 000d5950** : 98 kills (93+5 outro), **totaux/tueur EXACTS**, **17 familles = celles du décode offline**
>   (Disruptor/AR/Sniper/Needler/BR75/SPNKr/Stalker/Shock/Mangler/Ravager/Bulldog/Skewer/Heatwave/Hydra/Sidekick/
>   Cindershot/PulseCarbine). Catalogue `.ai/AUDIT_WEAPONS_2026-04-25.md`.
> - **PORTÉE** : ground-truth **non scalable** (1 replay/match, CE attaché). Pour le SCALABLE offline le mur biped→joueur
>   reste (§12). Le live sert d'**ancre de validation** pour l'arme à feu (mêlée/grenade restent offline §2/§3).
> - **MÊLÉE/GRENADE = déjà offline (§2/§3), NE PAS re-capturer en live** (le live les rate : 0 tick de dégât).
> - **Le QUI/assist/% par kill = le kill-event component `FUN_14104bd08`** (deser, replay-actif) : `+4`=tueur(finishing
>   blow), `+8`=%dmg tueur, `+0xC`=bool, `+0x10`=assistant, `+0x14`=%dmg assistant. L'arme N'Y est PAS (les 2 scalaires
>   = %dégâts) → se croise depuis le damage-handler/scanners. RECORD COMPLET = `Tueur·arme·Victime + Assistant·arme·%`.
> - ⚠️ Hooks live = lecture pure OK mais l'accumulation/charge a **crashé le jeu** plusieurs fois → capturer en UNE
>   passe puis lire VITE (le résultat agrégé survit, le flux brut est perdu si crash après).

---

## 0. LE RENVERSEMENT (à ne JAMAIS reperdre)

Pendant des semaines la conclusion était : **« le film ne relie aucun dégât à son attaquant ni à sa
victime ; l'association kill↔source est résolue en RAM au replay, jamais dans le flux »**. C'est vrai
**UNIQUEMENT pour les RECORDS DE DÉGÂT** (`FUN_14080c1f8`, paquets type-0 `payload[0]==0xd2`, 519 records,
famille bit-exacte mais NI tueur NI victime).

**C'est FAUX pour les EVENTS de tir / mêlée / grenade.** La recherche communautaire (@acurtis166 / @JGtm,
scripts Lua de lecture de film) montre que ces events portent le **PLAYER INDEX de l'auteur**
(tireur / frappeur / lanceur) + l'arme + (tir) un vecteur de visée + un bit hit/miss.

⇒ **L'ATTAQUANT EST DANS LE FILM** — pas dans le record de dégât, mais dans l'event d'action.
⇒ **La voie « source par kill » est ROUVERTE** : par kill (tueur connu via chunk_27, temps T) →
   chercher l'event fire/melee/grenade dont le `player index` == slot tueur, près de T → arme.

Ce qui était mort : lier le RECORD DE DÉGÂT à un kill (pas d'identité dedans).
Ce qui revit : lier l'EVENT D'ACTION à un kill (player index dedans = l'auteur).

---

## 1. HYPOTHÈSE DIRECTRICE USER (heuristique de décode — 2026-06-07)

> Citation : « je trouverai ça étrange qu'on encode d'une manière les types melee et grenades et d'une
> autre les "distance". Moi en tant que développeur j'aurais mis les infos plus ou moins au même endroit.
> Comme ça le moteur ne cherche pas à 1000 endroits. Mais je peux me tromper. »

**À appliquer comme heuristique forte pour décoder les FIRE events** : ne PAS présumer un schéma
totalement différent. Chercher la même structure (marqueur court → id d'arme → champs → player index 5b)
**près de la même région / même encodage** que melee & grenade. Les 3 types sont vraisemblablement des
**variantes d'un même record d'« action de combat »** (un event-type partagé ou 3 event-types voisins dans
la table de dispatch), pas 3 formats étrangers. Si le décode fire diverge, revérifier d'abord cette piste
de co-localisation avant d'inventer un format ad hoc.

---

## 2. GRENADE — ✅ DÉCODÉ ET VALIDÉ (70 events, player index 0-7)

Recette bit-packée (MSB-first, `bitsAt(d, bp, n)`), scan sur tous les chunks 00..27 :

```
marqueur     = 0x4c0c00          (24 bits)   ← ancre de scan
grenade_id   = bits[bp+24 .. +56]  (32 bits)  ← type de grenade (table ci-dessous)
(skip 47 bits)
player_index = bits[bp+24+32+47 .. +5] (5 bits) ← AUTEUR du lancer (slot 0-7)
```

Table grenade_id (32 bits) :

| id (hex)     | grenade |
|--------------|---------|
| `0xB0171062` | Frag    |
| `0xC0E34C44` | Plasma  |
| `0x3B2567D4` | Shock   |
| `0x9212E428` | Spike   |

**Résultat probe `cmd/tmp_meleegrenade`** : **70 grenades** décodées, **player index 0-7 tous valides**
(distribution cohérente, IKE/pi4 ~x20, etc.). Le temps vient du ts du paquet conteneur (`tsAtBit`).
⇒ La recette grenade est **fiable** ; sert de patron de référence pour fire/melee.

---

## 3. MELEE — ⚠️ CALIBRÉ À 90% (marqueur + type confirmés ; player-index offset TBD)

Calibré par debug ancré sur la famille marteau `high32 = 0x841ac5e5` (Gravity 0x841AC5E542C9679F,
Rushdown 0x841AC5E5D8D07CA1).

```
marqueur  = 0x534 (HIT) / 0x535 (MISS)   (11 bits)   ← CORRECTION : PAS 0x532 (source comm. erronée)
                                                        0b1010011010x, x = bit hit/miss
                                                        cohérent : 0x534>>3=0x34, 0x535>>3=0x35 (= "anchor 0x34/0x35")
anchor    = bp + 3
type      = bits[anchor+76 .. +8]   (8 bits)  ← 0x47 GRAVITY HAMMER (CONFIRMÉ, ~40 occ) ;
                                                 0x42 unpowered/non-melee-miss ; 0x60 energy sword/unpowered-hit
weapon    = bits[anchor+86 .. +32]  (32 bits high32 = famille)  pour type 0x47
            (type 0x42 → anchor+88 ; type 0x60 → anchor+101 ou +103)
player_index = bits[anchor+20 .. +5]  ← ❌ FAUX (lit toujours 0). OFFSET À RETROUVER.
```

**Statut** : marqueur + anchor + type 0x47 + weapon offset = **confirmés** (events marteau présents près
du kill narré 355.7s : markers 0x535 type 0x47 à 356.9/357.9/358.6s). **Le seul résidu = l'offset du
player index** (anchor+20 lit 0, donc faux). À retrouver en scannant les offsets autour du weapon pour un
champ 5b cohérent avec IKE/pi4 sur les events proches des kills marteau IKE→JGtm (115.5/292.5/355.7/375.1s).

---

## 4. FIRE — ❌ NON DÉCODÉ (recette cible + heuristique de co-localisation)

Champs attendus (recherche communautaire) dans un event de tir :

```
weapon_id    (32 bits, high32 = FAMILLE = notre clé analysis.WeaponIDToName)
shot_counter (0..127)        ← compteur de tir dans la rafale
burst_bit    (1 bit)         ← tir final de rafale
hitmiss_bit  (1 bit)         ← 0 = touché, 1 = manqué
player_index (5 bits)        ← TIREUR (slot 0-7)
aim_vector   (cubemap 30 bits) ← face = valeur ÷ 0xAAA8000 (_FACE_SIZE) ; u,v sur la face
```

Décode aim vector (référence communautaire, à porter Go si besoin) : 30 bits → `face = v / 0xAAA8000`
(0..5), puis (u,v) sur la face → direction monde. **Optionnel** pour l'attribution (le player index +
hitmiss suffisent) ; utile seulement pour des features avancées (heatmap de visée).

**HEURISTIQUE DE DÉCODE (cf §1)** : chercher la structure fire **par co-localisation** avec melee/grenade.
Pistes concrètes :
- Tester un marqueur court analogue (comme 0x4c0c00 pour grenade / 0x534 pour melee) en tête d'event fire.
- Le `player_index` est probablement à un offset fixe **relatif au weapon_id** (comme melee : champ 5b
  proche du weapon), pas à une position absolue exotique.
- high32 du weapon_id = la même clé `WeaponIDToName` que partout ailleurs → valider le décode en exigeant
  un high32 ∈ catalogue (Disruptor 0x84BD29ED == valeur communautaire == notre décode record = ancrage).

---

## 5. CROISEMENT EVENT → KILL (le plan d'attribution)

Une fois fire+melee+grenade décodés (player index + temps + arme), l'attribution par kill :

1. **chunk_27** (kill feed, 93/93 sur 000d5950) → par kill : `tueur slot` (via `b36` duo + `b37` team =
   bijection per-match) + `victime` + `temps T`.
2. **mapper player_index → slot tueur** : le player index des events est l'index roster (0-7) ; chunk_27
   donne le slot tueur. Établir la bijection player_index ↔ slot (probablement identité ou permutation
   roster stable ; à caler sur les grenades déjà décodées dont on connaît l'auteur).
3. Par kill : event (fire/melee/grenade) dont `player_index == tueur` ET `temps ≈ T` (fenêtre courte,
   préférer le hit_bit=touché et le plus proche ≤ T) → **arme = source du kill**.
4. **Cross-check** par les 519 records de dégât (famille présente près de T) pour la confiance.

⇒ Différence cruciale vs l'appariement temporel réfuté (record de dégât) : ici l'event PORTE l'auteur
(player index), donc on n'attribue PAS au mauvais joueur en Fiesta. C'est ce qui lève l'ambiguïté qui
avait tué la voie record-de-dégât (faux positif Stalker sur frag marteau, narration 0/6).

---

## 6. VALIDATION NARRATION (critère de succès)

Cibles connues (pi → xuid) :
- `pi2 = 2533274823110022` JGtm
- `pi4 = 2533274815845110` IKE ILYA
- `pi5 = 2535444178793711` Akatsuki fire17

Kills à retrouver :
- **Marteau IKE ILYA → JGtm** (mêlée) : kills narrés 115.5 / 292.5 / 355.7 / 375.1 s → events melee
  type 0x47 (Gravity Hammer) avec player_index = pi4 (IKE) près de ces temps.
- **BR75 JGtm → Akatsuki fire17** (tir) : kills narrés 112.9 / 329.8 s → events fire high32=BR75 avec
  player_index = pi2 (JGtm) près de ces temps.

Succès = ≥ ces narrés retrouvés avec le BON auteur, ≥1 mêlée + ≥1 tir + ≥1 grenade attribués correctement.

---

## 7. PROBES (worktree, untracked — ne PAS committer sans accord user)

- `cmd/tmp_meleegrenade/main.go` — ✅ grenade (70 events OK) + melee scaffold (marqueur 0x534/0x535,
  type 0x47, weapon offset OK ; player-index offset à corriger).
- `cmd/tmp_fireevents/main.go` — weapon ids byte-alignés (probablement weapon-defs, pas fire events) ;
  marqueur fire non byte-aligné (events bit-packés) → à refaire en scan bit-packé façon meleegrenade.
- Patron de scan : `bitsAt(d, bp, n)` MSB-first ; ts via `tsAtBit` (header paquet 16 o
  `[Type u16 LE][b2][b3][Size u32 LE][ts u64 LE]`, t0Us=4537898226).

---

## 7bis. RÉSULTATS WORKFLOW wl0melxzh (2026-06-07) — offsets trouvés, MAIS attribution par kill ÉCHOUE

> 3 agents (decode mêlée ‖ decode fire → cross). Probes worktree untracked : `cmd/tmp_meleepidx`,
> `cmd/tmp_firefinal` (+ tmp_fire* intermédiaires), `cmd/tmp_eventcross`.

**ACQUIS SOLIDES :**
- **Bijection player_index ↔ slot tueur = ÉTABLIE et VALIDE** (via chunk_27 `b36` duo + `b37` team,
  8 combos distincts). xuid↔pi = identité roster stable. **Réutilisable** : pi0 whiteknight=(duo3,team0),
  pi1 JAVIER=(3,1), pi2 JGtm=(1,0), pi3 LORD PEINX13=(2,0), pi4 IKE=(1,1), pi5 Akatsuki=(2,1),
  pi6 aldusbroncus=(0,1), pi7 VitaminA1688=(0,0).
- **MÊLÉE player_index offset = anchor+23** (5 bits), PAS anchor+20 (l'ancien était hérité du patron
  grenade, décalé de 3 bits). SEUL offset de [anchor-16..+120] donnant une distribution canonique 0-7
  (8 valeurs, bien réparties, stable hammer/autres). IKE(pi4) = porteur de marteau 7× dans le match.
- **FIRE : player_index = bitsAt(bp-4, 5)** — PRÉCÈDE le weapon_id (≠ grenade où il suit). Ancre =
  weapon_id 64b (high32 famille ∈ catalogue + low32 0x42c9679f, **même suffixe que melee/grenade →
  co-localisation CONFIRMÉE**). Validation forte : la **rafale BR75 (0x2b1824d5) 151.6→157.0s = 25 records
  consécutifs TOUS pidx=2 (JGtm)** → arme+auteur cohérents. hit/miss = nibble haut 0x3↔0x7 du post-id 32b.

**⛔ BLOCAGE — attribution PAR KILL = NON fiable (verdict honnête) :**
1. **DÉSYNC TEMPORELLE event↔kill-feed.** Le player_index est bon, mais les ts des events ne tombent pas
   sur les instants kill. Pire : certains "fire records" ont un `tsAtBit` qui déborde (~1.8e13) ⇒ le scan
   du motif 0x42c9679f matche AUSSI des données HORS paquets type-0 (records d'entités-armes + bruit).
   Les 1061 "fire records" sont donc un MÉLANGE (events de tir réels + snapshots d'état d'entité-arme).
2. **Narration 0/2.** Marteau IKE→JGtm (115.5/292.5/355.7/375.1s) : les 7 events mêlée de pi4 existent
   (75.0/271.9/274.9/307.3/324.5/335.1/462.6s) mais à 14-40s des kills narrés, et leur **famille @anchor+86
   décode "Diminisher of Hope" et non "Gravity Hammer"** (offset weapon mêlée FAUX pour type 0x47, à recaler).
   BR75 JGtm→Akatsuki (112.9/329.8s) : les 28 fire BR75 de pi2 sont TOUS dans la rafale 151-158s, jamais
   près des kills. ⇒ "event d'action à T" ≠ "kill à T".
3. **Couverture 21/93** (event auteur dans [-400ms,+2.5s]) dont la majorité à dt 1-2s = coïncidences.
   Attribution arme+auteur CORRECTS par kill ≈ **nulle / non démontrable**.

**RÉINTERPRÉTATION CLÉ (la vraie piste rouverte) :** les "fire records" qui débordent en ts et clusterisent
en rafales de pidx constant ressemblent fortement aux **NEW records d'entités-armes** (déjà connus, cf
KILLFEED_STATE §0 : "885 littéraux = entités-armes pickup/spawn, PAS un record de dégât"). Le champ bp-4
serait alors le **HOLDER (slot porteur)** de l'entité-arme, pas un tireur. **Si c'est le cas, on tient une
timeline ARME-TENUE-PAR-SLOT** (= exactement la VOIE A, atteinte SANS le walk biped i63) : weapon_id +
holder slot + temps. ⇒ Le bon modèle d'attribution n'est PAS "event à T" mais **"dernière arme tenue par le
tueur ≤ T" (carry-forward)** — modèle que le cross n'a PAS testé (il testait event-à-T). C'EST LE PROCHAIN
TEST DÉCISIF.

## 7ter. TEST DÉCISIF (probe `cmd/tmp_heldweapon`, 2026-06-07) — scan propre type-0 + modèle held-weapon

> Re-scan PAQUET PAR PAQUET (header 16 o validé, ts fiables, garde-fou tms∈[0,600s]). Probe untracked.

**LA CONTAMINATION ÉTAIT RÉELLE — et corrigée :** seulement 2 paquets ts-hors-borne ignorés. Les "fire
records" se répartissent : **792 en paquets type-0 (events réels) + 255 en type-2 (keyframe = snapshots
d'arme-tenue)**. ⇒ confirme que le scan brut mélangeait events de tir et snapshots d'état d'entité-arme.

**Distribution player_index propre (ts-valides) :** grenade = tous joueurs ; melee = tous ; **fire = seulement
pi2(105)/pi3(174)/pi6(311)/pi7(310)** (pi0/pi1/pi4/pi5 = 0 fire). pi4 IKE = 0 fire / 7 melee / 20 grenade
(marteau+grenade, cohérent). Réserve : fire ne couvre que 4 joueurs → soit ce sont les 4 gunners actifs,
soit le champ bp-4 n'est pas un player_index propre pour le fire (à surveiller).

**ATTRIBUTION — 2 modèles comparés sur 93 kills :**
- **MODÈLE A (event-à-T, [T-2.5s,T+0.4s]) = 21/93.** (le modèle d'eventcross, confirmé faible.)
- **MODÈLE B (held-weapon carry-forward = dernier fire/melee du tueur ≤ T) = 80/93.** ← Voie A directe.

**NARRATION (modèle B) :**
- ✅ **Marteau IKE(pi4)→JGtm, 4/4 FAMILLE CORRECTE.** Aux 4 kills (115.5/292.5/355.7/375.1s, tous appariés
  à un vrai kill chunk_27 à ±37ms) l'arme tenue d'IKE = **Rushdown Hammer** (famille marteau 0x841ac5e5).
  Le dt arme→kill est grand (-17 à -40s = le dernier coup de marteau capté avant le kill) mais la FAMILLE
  est bonne. ⇒ **le décode mêlée + held-weapon FONCTIONNE pour la mêlée.** (Bonus : sur le scan propre,
  l'arme mêlée type 0x47 lit bien un marteau — le "Diminisher of Hope" rapporté par le workflow était de
  la contamination.)
- ❌ **BR75 JGtm(pi2)→Akatsuki, 2/2 ÉCHEC.** Arme tenue = M41 SPNKr@51.9s (T=112.9s) / Energy Sword@272.5s
  (T=329.8s). La rafale BR75 de JGtm est à 151-157s, APRÈS le 1er kill et loin du 2e. ⇒ **capture fire trop
  ÉPARSE** (792 events type-0 pour TOUT le match = pas du per-shot ; probablement des events draw/equip ou
  state-périodique, pas chaque balle) → l'arme-tenue-gun n'est pas densément suivie → held-weapon non fiable
  pour les armes à feu. (Ou : le temps de narration BR75 est approximatif — narration ≠ ground-truth décodeur.)

**BILAN TEST DÉCISIF :** le modèle **held-weapon carry-forward** (= Voie A, sans walk biped) **marche pour la
MÊLÉE** (hammer 4/4) et donne **80/93 de couverture**, MAIS la branche **armes à feu reste non fiable** car la
capture fire type-0 est éparse (pas per-shot). Acquis : mêlée attribuable ; grenade attribuable (lanceur connu) ;
fire = couverture mais fiabilité gun non démontrée. Le verrou s'est déplacé de « pas d'attaquant » à
« densité/sémantique exacte des events fire type-0 ».

## 7quater. CROSS-CHECK A.0 vs 519 records de dégât (probe `cmd/tmp_heldvsdmg`, 2026-06-07)

> Valide le décode fire/held contre une source INDÉPENDANTE : les 519 records de dégât (famille EXACTE +
> temps, sans auteur). Le décode fire est-il CORRECT, et la capture est-elle DENSE ? Probe untracked.

**Données :** 519 records de dégât (0xd2, famille connue), 848 held-events (fire+melee type-0), 93 kills.

**(A) Modèle held-weapon carry-forward (dernier fire/melee du tueur ≤ T) :**
- 39 kills held=MÊLÉE (non corroborable par 519 — la mêlée ne produit PAS de record de dégât), 41 held=gun,
  13 sans arme tenue.
- **Gun corroboré = 8/41 (20%)** (held-fam présente dans un record de dégât même famille [T-1.5s,T+0.2s]).
  ⇒ le carry-forward gun est FAIBLE : il ne matche que les kills où le tueur a un fire event ~AU kill (dt≈0) ;
  sinon il sert une arme PÉRIMÉE (dt -17s..-60s). Pattern net : tous les ✓ ont dt≈0, tous les ✗ ont dt grand.

**(B) Modèle event-à-T (event auteur le plus proche, le modèle PRÉCIS) :**
| fenêtre | kills avec event auteur | dont gun | gun corroborés |
|---------|------------------------:|---------:|---------------:|
| ±500ms  | **13/93** | 12 | **9 (75%)** |
| ±1500ms | 22/93 | 15 | 9 (60%) |
| ±3000ms | 34/93 | 22 | 9 (41%) |

**⇒ CONCLUSION A.0 (capitale) :** le **décode fire est CORRECT** — quand un fire event du tueur tombe à ±500ms
du kill, il est corroboré par un record de dégât même famille **75%** du temps (player_index + arme justes).
MAIS la **capture fire est ÉPARSE** : seulement **~13/93 kills** ont un fire event serré (le nombre de gun
corroborés plafonne à **9** quelle que soit la fenêtre — élargir n'ajoute que du périmé). Le verrou n'est donc
ni l'identité (bonne) ni la corrélation (bonne quand serrée) mais **LA DENSITÉ DE CAPTURE des fire events**.
Les 792 fire events type-0 ne couvrent pas chaque tir → l'arme-à-feu-par-kill reste partielle (~9-13/93 fiables).

**État par type :** MÊLÉE = attribuable (held=marteau, IKE→JGtm 4/4 famille hammer ; 39 kills) ; GRENADE =
attribuable (lanceur connu) ; ARME À FEU = **CORRECTE mais éparse** (~9-13/93 fiables, plafond capture).

**Narration (confirme) :** Marteau IKE→JGtm = held=Gravity/Rushdown Hammer aux 4 kills ✓ (records vides =
normal, mêlée). BR75 JGtm→Akatsuki = held=SPNKr/Energy Sword aux 2 kills, **0 record BR75 à ±1.5s** : ni le
held ni les records ne voient un BR75 à ces kills (la rafale BR75 de JGtm est à 151-157s, ailleurs). ⇒ soit le
temps de narration est faux (narration ≠ ground-truth), soit ces 2 kills ne sont pas au BR75. Ne PAS forcer.

## 7quinquies. A.1 — DENSIFIER LA CAPTURE FIRE : MUR (probe `cmd/tmp_firemarker`, 2026-06-07)

> Objectif : capter plus de fire events (chaque tir, tous joueurs). 3 sondes : (1) low32 des ancres, (2)
> marqueur fire court, (3) bp-4 = player_index ? Probe untracked.

**(1) Pas de variant "manquant" exploitable :** sur 29257 paquets type-0, 1159 occurrences high32∈catalogue ;
**792 (68%) ont low32==0x42c9679f**, le reste = 91 autres low32 (variants d'arme : Rushdown Hammer 0xd8d07ca1,
etc. + NEW records d'entités-armes). Élargir le low32 ajoute surtout des records d'ENTITÉ (spawn/pickup), pas
des tirs → ne densifie pas les fire events.

**(2) AUCUN marqueur de fire event court** (≠ grenade 0x4c0c00 / melee 0x534) : zéro run de bits constants
autour de l'ancre weapon_id (hors la zone weapon_id elle-même). ⇒ l'hypothèse de co-localisation tient pour
le *suffixe partagé* mais **il n'y a pas de marqueur fire dédié** à scanner pour capter tous les tirs. Le
post-id 32b = `0x760x_xxxx` (tag de classe d'arme : Disruptor→03, Ravager→04), pas un hit/miss propre.

**(3) bp-4 N'EST PAS un player_index propre :** distribution sur ancre stricte = seulement **{2,3,6,7}** ;
sur ancre élargie = **{2,3,4,5,6,7}** mais **JAMAIS 0 ni 1**. Or pi0 (whiteknight) et pi1 (JAVIER) = les 2
joueurs PASSIFS (victimes fréquentes, ~0 kill). Donc bp-4 porte un **signal tireur réel mais PARTIEL** (cohérent
avec la corrob 75% à ±500ms de A.0 pour les joueurs actifs) — soit ces 2 joueurs n'ont quasi pas tiré d'arme
trackée, soit bp-4 n'est pas exactement le player_index. **Non concluant comme index 0-7 fiable.**

**⇒ VERDICT A.1 (MUR pour l'arme à feu) :** la capture fire **ne peut pas être densifiée** par scan offline —
pas de marqueur dédié, élargir le low32 ramène des records d'entité, et bp-4 n'est pas un player_index complet.
Le film **n'enregistre pas chaque tir** comme event indépendant trackable ; les ~792 fire events type-0 sont le
plafond. ⇒ **l'arme-à-feu-PAR-KILL fiable plafonne à ~9-13/93** (kills avec un fire event serré + corroboré).

## 7sexies. 3 ANGLES SUPPLÉMENTAIRES (push user "il y a forcément le reste") — tous négatifs pour l'arme à feu

> Probes untracked : `cmd/tmp_fireprobe2`, `cmd/tmp_hitevent`, `cmd/tmp_keyframeloadout`. Push user :
> « pourquoi une méthode pour la mêlée et une autre pour les armes à feu ; cherche le player_index autour
> des armes ; Acurtis traque spawns/pickups/refills donc le reste doit y être ». Trois pistes testées à fond :

**(α) Recherche EXHAUSTIVE d'un player_index 0-7 autour du weapon-id (tmp_fireprobe2).** Champ w=5 balayé
sur [-80,+200]. Sur les occurrences confirmées fire-adjacentes (588, à ±150ms d'un record de dégât même
famille), **bp-9 couvre les 8 joueurs** (dist [67 77 93 63 91 34 59 103]) — MAIS validation kill feed :
bp-9 = 50% corrob (8/16), **PIRE que bp-4** (59%, 10/17). ⇒ l'uniformité 0-7 de bp-9 est **coïncidente**
(champ 5b quelconque), pas le tireur. Les DEUX plafonnent à ~17/93. Le 0-7 "complet" n'est PAS prédictif.
NB : 574/792 occurrences-suffixe sont fire-adjacentes ⇒ ces records SONT corrélés au tir, mais sans index
tireur fiable dessus.

**(β) Hypothèse UNIFIÉE : 0x534/0x535 = event de dégât général (mêlée+arme) (tmp_hitevent).** REFUTÉE.
36139 marqueurs 0x534/0x535 (le marqueur 11b est trop court = bruit) ; seulement 90 portent une famille,
**dominées par Rushdown Hammer/Elite Bloodblade (mêlée)** ; validation = **1/93** kills gun. Les armes à feu
NE sont PAS encodées comme events-hit type-codés. La mêlée est un event distinct.

**(γ) Loadout aux KEYFRAMES (type-2) = possession dense (tmp_keyframeloadout).** 26 keyframes, **504**
weapon-id (≈19/keyframe = TOUTES les entités-armes du monde : tenues + lâchées + spawns, pas 8 loadouts).
Aucun champ slot w=4/w=5 uniforme 0-7 ; held-weapon@kill corroboré **0-4%** sur tous les offsets candidats.
⇒ les records keyframe ne portent pas un slot-porteur propre lisible ; c'est le **registre d'entités-armes**,
dont le lien entité→porteur est résolu en RAM (le mur de fond).

**⇒ CONVERGENCE (3 angles + A.0/A.1) :** la donnée d'arme à feu EST dans le film (entités-armes : weapon-id
+ handle + temps, fire-adjacentes), MAIS **aucun champ sérialisé ne porte le tireur/porteur de façon fiable**
(le lien entité-arme→joueur passe par la résolution de handle en RAM). C'est exactement la différence avec
mêlée/grenade, qui sont des **events d'ACTION** portant explicitement l'acteur (player_index propre, 8 joueurs).
Les armes à feu n'ont pas d'event d'action per-tir équivalent dans le flux.

**PIÈCE MANQUANTE PRÉCISE (= méthode non partagée d'Acurtis) :** l'**event de PICKUP/équipement** qui appaire
explicitement (joueur, entité-arme) au moment du ramassage. Il donnerait la possession (entité→joueur) ; en le
croisant avec le handle d'entité des records de dégât on attribuerait l'arme à feu par kill. Non localisé (event
rare, pas de vérité-terrain pour le caler par scan aveugle). Acurtis le décode mais n'a pas partagé la méthode.

## 7septies. RE GHIDRA — l'arme du kill = l'arme ÉQUIPÉE du tueur (RAM au replay), pas un index sérialisé

> Mission (push user "weapon index ?") : RE le mécanisme du kill feed pour trouver d'où vient l'arme du kill.
> Ghidra-mcp opérationnel (HaloInfinite.exe, projet HI, port 8089). Décompile `FUN_1406730c4` (apply kill feed).

**RÉSULTAT (décompile `FUN_1406730c4`) :** l'arme affichée au kill feed est lue sur **l'ENTITÉ DU TUEUR**, pas
sur l'event sérialisé :
- `lVar11 = FUN_1404969f0()` = entité du tueur ; `iVar18 = *(int*)(lVar11 + 0x538)` = **handle de l'arme
  équipée** ; `*(int*)(lVar11 + 0x1f30)` = **famille de l'arme équipée**. Résolus en RAM au replay.
- L'event sérialisé `param_2` porte : id tueur (offset 8), id victime (offset 4), positions (param_2[4..9]),
  flags (0x1c). **AUCUN champ arme / weapon-index.** ⇒ confirme la sonde data (pas de weapon-index dans le
  bloc kill de chunk_27) : l'arme du kill N'EST PAS sérialisée dans l'event — c'est l'arme ÉQUIPÉE du tueur
  à l'instant du kill, pull de l'état RAM de son entité.

**⇒ La donnée existe (l'arme équipée), mais elle est un ÉTAT d'entité, pas un event.** L'arme équipée est
sérialisée dans la **WST i43 du biped** (`weapon-state-type-info`, famille = id64 catalogue). `cmd/tmp_loadout`
la décode déjà : au keyframe (type-2 chunk_02), les 8 records biped #35 donnent l'arme primaire par record
(Hydra/SPNKr/Mangler/MA40/Cindershot…), **lue AVANT la désync i51+** (pas besoin du walk complet).

**LE CRUX UNIQUE (où TOUTES les voies arme-à-feu convergent) = mapping biped-record → joueur.** Au keyframe,
le `obje` LocalID est **absent** (`present=false`) ⇒ le record ne porte pas d'id joueur lisible ; les 8 records
sont en ordre roster mais `record_i ↔ player_i` n'est **pas confirmé**. C'est exactement ce qui DIFFÉRENCIE
l'arme à feu de la mêlée/grenade : ces dernières sont des **events d'ACTION portant le player_index directement**
(anchor+23 / 5b), tandis que l'arme à feu est un **ÉTAT** (arme équipée du biped) nécessitant la résolution
biped→joueur — le mur historique du projet (cf KILLFEED_STATE : walk biped, count2 RAM).

**LEVIER RESTANT (tractable, non épuisé) :** CONFIRMER l'ordre `record_i ↔ player_i` aux keyframes via vérité-
terrain (nos events mêlée/grenade FIABLES + 519 records de dégât). Si record#k = slot tueur (connu via kill
feed b36/b37), alors arme-à-feu-par-kill = WST i43 du record du tueur au keyframe ≤ T. C'est la SEULE voie
offline restante pour l'arme à feu, et elle réutilise du décode déjà écrit (tmp_loadout + filmdec WST).

## 7octies. LEVIER record↔slot via vérité-terrain mêlée — ÉCHEC (probe `cmd/tmp_recordslot`)

> Tentative de confirmer la permutation record-keyframe↔slot : vote (event mêlée arme-tenue + player_index)
> ⋈ (record du keyframe contemporain portant cette arme). 25 keyframes, 49 events mêlée, 47/49 appariés.

**RÉSULTAT : pas de permutation stable.** Pureté record→slot = **29-50%** (aucun slot ne domine un record).
Causes : (1) les "records" (groupes de littéraux d'armes par proximité) ne sont PAS alignés sur les bipeds —
ce sont les **entités-armes du monde** (tenues + lâchées + spawns) ; nb records/keyframe varie (8/7/6/7…),
pas 8 bipeds propres. (2) une même arme (ex marteau) apparaît dans plusieurs groupes → match ambigu.
⇒ Le groupement léger par proximité ne récupère pas le loadout par joueur. Il faudrait la **traversée biped
complète** (tmp_loadout calibration, coûteuse) pour attribuer les WST au bon biped — ET le biped→joueur reste
non résolu (obje LocalID absent). ⇒ **le crux biped→joueur n'est pas contournable par cette voie.**

**BILAN DES LEVIERS ARME-À-FEU (tous épuisés cette session) :** player_index exhaustif (bp-4 partiel, bp-9
coïncident) ; hit-event unifié 0x534/0x535 (réfuté) ; keyframe loadout (entités monde, pas 8 loadouts) ;
weapon-index dans l'event kill (absent, confirmé RE) ; record↔slot par vérité-terrain mêlée (pas de permutation
stable). **TOUS convergent sur le même mur : biped→joueur**, résolu en RAM au replay (count2 RAM + obje LocalID
absent). Seules voies pour le franchir : **debugger** (ground-truth replay, non scalable) ou la **méthode
biped→joueur non partagée d'Acurtis**.

## 8. SYNTHÈSE FINALE (2026-06-07) — ce que le film donne offline, par type

| Type de kill | Attribution par AUTEUR offline | Fiabilité | Mécanisme |
|---|---|---|---|
| **Mêlée** | ✅ OUI | **Fiable** | event melee 0x534/0x535, player_index @anchor+23 (couvre les 8 joueurs), arme @anchor+86. IKE→JGtm hammer **4/4**. |
| **Grenade** | ✅ OUI | **Fiable** | event 0x4c0c00, grenade_id 32b, player_index 5b. 70 events, 0-7 tous présents. |
| **Arme à feu** | ⚠️ PARTIEL | **~9-13/93** | fire event (weapon_id 64b, bp-4 signal tireur partiel) : correct à 75% quand serré (±500ms) mais capture éparse + bp-4 incomplet. |
| (cross-check) | — | — | 519 records de dégât = **famille + temps EXACTS, SANS auteur** : corrobore les fire serrés, mais ne désambiguïse pas l'auteur en Fiesta. |

**Le mur de fond reste le même que tout le projet** (cf KILLFEED_STATE §0) : le film résout l'association
kill↔source-d'arme-à-feu **en RAM au replay**, il ne sérialise PAS chaque tir attribuable. La **nouveauté
acquise cette session** : **mêlée + grenade SONT attribuables par auteur** (events à player_index propre),
ce qui était nié avant. L'arme à feu par kill reste partielle (plafond capture).

## 9. OPTIONS (décision user)
- **P-prod du fiable** : productioniser l'attribution **mêlée + grenade par auteur** (+ timeline famille via 519
  records comme contexte d'arme), en assumant l'arme-à-feu-par-kill comme "non couverte de façon fiable".
- **Pousser l'arme à feu** (rendements décroissants) : seule voie restante = **debugger** (breakpoint replay →
  ground-truth par kill, non scalable, validation seulement) ou accepter le plafond ~13/93.
- **Arrêter le décode** ici (mur de capture atteint et chiffré) et capitaliser sur l'acquis.

## 10. PROCHAINES ÉTAPES (si P-prod du fiable — mêlée + grenade par auteur)

1. **Décodeur events → `internal/analysis`** (algo pur) : parse film → liste d'events
   `{tms, kind(melee/grenade), weaponFamily, playerIndex}` (recettes §2/§3, scan type-0 propre §7ter).
2. **Attribution par kill** : kill feed chunk_27 (tueur slot via bijection b36/b37) ⋈ event melee/grenade du
   tueur ≤ T → `tueur · source · victime`. Gun = best-effort (fire serré ±500ms) marqué "indicatif", non garanti.
3. **Service + handler** : `internal/service` orchestre (PathResolver multi-titre, capability-gated — ne pas
   coder en dur `halo_infinite`) → handler HTTP. Table append-only ART-safe si persistance.
4. **Tests** : analysis (events bit-exacts sur 000d5950), service (mock repo), handler (httptest).
5. **Garde-fous honnêteté** : exposer la source mêlée/grenade comme fiable ; l'arme à feu par kill comme
   partielle (ne pas la présenter comme résolue). Doc + thought_log.

> Probes throwaway de cette session (worktree, untracked, NE PAS committer telles quelles) :
> `cmd/tmp_{meleegrenade,meleepidx,firefinal,eventcross,heldweapon,heldvsdmg,firemarker,fireprobe2,hitevent,`
> `keyframeloadout,weaponindex,kf27weapon,recordslot}`.
> Le code prod réécrit proprement les recettes validées dans `internal/analysis/filmdec`.

## 11. STATUT FINAL (2026-06-07, MIS À JOUR 2026-06-12) — arme-à-feu SUSPENDUE en OFFLINE, DÉBLOQUÉE en LIVE
> **MAJ 2026-06-12** : la voie (3) du §12 (debugger/CE ground-truth) a été exécutée et **MARCHE** — arme-à-feu-par-kill
> obtenue live via `FUN_1407e00ac` (cf bannière ⭐⭐ en tête de doc). Ce §11 ci-dessous = l'état OFFLINE (le mur
> biped→joueur reste pour le scalable). Le live n'est PAS scalable mais donne la vérité-terrain par match.

**Décision user (2026-06-07) : ON ABANDONNE l'arme-à-feu-par-kill OFFLINE pour le moment.** Acquis figé, pistes
documentées ci-dessous pour reprise éventuelle. Récap des stats exploitables : `.ai/V7.5/film_re/RECAP_STATS_EXPLOITABLES.md`.

**Ce qui est PROUVÉ et clos :**
- Arme du kill = arme ÉQUIPÉE du tueur (`killer_entity+0x1f30` famille / `+0x538` handle), résolue en RAM au
  replay (RE `FUN_1406730c4`, §7septies). PAS sérialisée dans l'event de kill.
- 0 weapon-family ni weapon-index dans chunk_27 (testé tous offsets bit, §7septies/sonde tmp_kf27weapon).
- La family EST dans le flux type-0 (records d'arme/dégât) mais liée à l'**entité-arme/biped**, jamais au kill.
- TOUS les leviers offline convergent sur le mur **biped→joueur** (count2 RAM + obje LocalID absent).

## 12. PISTES ENVISAGEABLES (pour une reprise future — par ordre de ROI décroissant)

1. **Méthode biped→joueur d'Acurtis (pickup/possession)** — la pièce manquante exacte. Il décode déjà
   spawns/pickups/refills (events lifecycle joueur↔entité-arme). Si partagée → débloque l'arme à feu par kill
   directement (possession → arme tenue du tueur au kill). **ROI le plus élevé, dépend d'un tiers.**
2. **RE Ghidra du composant joueur/owner du biped** — décompiler la chaîne qui écrit `biped+0x1f30`/`+0x538`
   (équip d'arme) et le composant qui porte le player-index/owner du biped (le `count2` RAM = `FUN_1409fe718`,
   i63). But : trouver SI un champ sérialisé (WST ou autre composant) porte l'owner du biped, contournant la
   résolution RAM. Frontal, lourd, incertain (le projet a déjà buté ici).
3. **Debugger ground-truth** — breakpoint `FUN_1406730c4` (ou `FUN_1407e00ac`) au replay Theater → lire
   `killer_entity+0x1f30` par kill. **Non scalable** (replay manuel par film) → validation seulement, pas prod.
4. **Walk biped bit-exact complet** (port i54/i59/i63, cf `WALK_PORT_NOTES.md`) → lire WST i43 (arme équipée)
   du biped tueur. Bute sur la **limite dure count2** (popcount masque RAM 73 bits, 0 bit dans le flux) ET le
   mapping biped→joueur. Effort élevé, plafond théorique.
5. **Décode offline de l'event pickup/equip** (sans Acurtis) — localiser l'event sérialisé qui change l'arme
   équipée. Pas de vérité-terrain pour le caler par scan aveugle → nécessite (2). Faible ROI seul.

**Recommandation de reprise** : tenter (1) d'abord (coût ~nul si Acurtis partage) ; sinon (2) en RE ciblé.
Ne PAS réinvestir : scan de fire events denses, hit-event unifié, keyframe loadout, weapon-index, record↔slot
par proximité — tous épuisés et documentés (§7bis→§7octies).
