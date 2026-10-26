# Lot 3.7 — OU LE FILM ECRIT LA REAPPARITION DES OBJECTIFS ET DES VEHICULES (2026-09-17)

> RECHERCHE SEULE. Aucun fichier de production Go modifie, aucun film du cache ouvert.
> Worktree `LevelUp-wt-decfilm-37r`, branche `feat/decfilm-37r`, base `2fac0a4cf`.
> Instruments : `film/research/reapparition/` (executable, LECTURE SEULE) et
> `film/internal/grammar/reapparition_37_bassin_research_test.go` (les sept mini-bobines par
> build, aucun film du cache).
> Ghidra N'ETAIT PAS DISPONIBLE (§ 7) : la chaine du descripteur a ete rejouee en Go sur le PE,
> calibree sur six temoins, et les desassemblages sont d'`objdump` (binutils ucrt64).

---

## 0. LA REPONSE, APRES LA VOIE LIBRE DU 2026-09-17

> Cette section a ete REECRITE apres la lecture de cinq films entiers. La version du matin
> affirmait que le bassin du moteur « porte les minuteurs d'objectif, et c'est lui que designent
> les index de `ti=11 i0` ». **La mesure l'a REFUTEE** (§ 6 bis.3) et elle affirmait aussi que le
> cycle des vehicules etait « aujourd'hui applicable » : **refute aussi** (§ 5). Les deux
> corrections sont ecrites la ou elles portent ; ce qui suit est l'etat MESURE.

**OBJECTIFS — CE QUI EST ACQUIS.** Le film ecrit bien des comptes a rebours, et le lot en a lu
un sur film entier : le BASSIN du moteur de jeu, `managed-engine-timers-component`
(`ti=0`/`ti=2` `i15`, ecrivain `FUN_1407ee7b8`) — masque `R(64)` de 64 fentes, puis par fente
presente `R(2)` d'etiquette et le MEME enregistrement de minuteur que l'horloge de manche
(`R(16)+R(16)+R(5)`, en SECONDES). **La lecture est prouvee par la COHERENCE TEMPORELLE** : sur
`bcb6d393`, une fente porte `a = 29,939 s` de duree totale et un reste qui tombe de 24,995 s a
5,219 s entre deux images-cles espacees de 20,0 s — 19,776 s consommees, soit la resolution du
quantum (§ 6 bis.2). Les trois grammaires qui barraient la route (`i11` = `R(128)`, `i13` =
`R(13)` + `n x R(1)`, `i14` = quatre champs sous le niveau 2) sont relevees et suffisent (§ 2 bis).

**OBJECTIFS — CE QUI EST REFUTE, ET C'EST LE RESULTAT LE PLUS IMPORTANT DU LOT.** Le bassin **ne
porte PAS le minuteur du drapeau**. La jointure que tout le lot cherchait — l'index de minuteur
d'un objectif designe-t-il une fente du bassin ? — est un **NEGATIF MESURE** : sur
`bcb6d393` (CTF:Arena, Cliffhanger), `fb1a1a72` (CTF:Arena, Banished Narrows) et `d9781168`
(Oddball:Arena, Dredge), **`ti=11 i0` vaut `(-1, -1)` — « aucun minuteur » — sur 446 records
d'objectif sur 446**. Ni le drapeau de deux CTF ni le crane d'un Oddball ne branche de fente. Et
les memes deux durees (27,741 s et 29,939 s) apparaissent en CTF **et** en Oddball, ou il n'y a
pas de drapeau, pendant que le nombre de fentes allouees (10, 13, 7) suit l'ordre de grandeur du
nombre de JOUEURS : les minuteurs du bassin sont GENERIQUES, pas modaux. Piste nommee et non
prouvee, a coller a une mort datee (§ 9) : `Engine_SetSpawnTimerAndTotalRespawnDurationForPlayer`
— la chaine du jeu decrit exactement la forme mesuree (duree totale + restant, par joueur).

**OBJECTIFS — LA VOIE QUI RESTE.** `managed-navpoint-manual-timer-initial-duration` /
`-current-duration` (`ti=12 i11` / `i12`), `R(17)` chacun, pas de 50 ms : une duree INITIALE et
une duree COURANTE, sans indirection ni index a resoudre. Grammaire complete, decidable hors
ligne, relevee au lot 3.6. **Non mesuree** : le bloquant de `ti=12` est `i1`, dix composants
avant. C'est desormais le chemin le plus court vers un compte a rebours d'objectif (§ 8.2), et le
second est `ti=13` (la propriete reseau nommee par le script Lua, § 9 item 5).

**VEHICULES — LE NEGATIF EST DOUBLE, ET LES DEUX MOITIES SONT MESUREES.** (1) Le film n'ecrit
AUCUN minuteur de reapparition de vehicule : sur les **294 noms de composant que l'executable
embarque**, aucun des **15** `vehicle-*` ne porte de temps hors `vehicle-emp-timer-component`,
les **10** « respawn » sont tous ceux d'un JOUEUR ou d'une nuee IA, et « return » comme « reset »
rendent **ZERO** (§ 4) ; le delai vient des surcharges de VARIANTE
(`ManagedGameVariant_*VehicleRespawnTimeOverride*`) et des proprietes de placement Forge, deux
regles connues avant le match. (2) **La deduction, elle, n'est pas faisable aujourd'hui** :
`VehicleTrack.TEnd` — la fin datee par le dead-state `ti=40 i11` — est renseignee **1 fois sur
109** sur `a349fea8` et **1 fois sur 42** sur `a521164d` (cuisson de production). Sur 31
emplacements agglomeres, **zero ecart mesurable**. Le prealable n'est plus la resolution du
recensement (levee en 2026-09-05) mais l'ATTRIBUTION de la fin de vie (§ 5).

**LE CANDIDAT QUI RESTE OUVERT, NOMME POUR NE PAS REFAIRE V22 BIS** :
`device-object-dispenser-timer-component` (`ti=43 i37`) est un VRAI compte a rebours de
generateur — deux fentes, porte `R(1)`, `R(10)+R(10)+R(5)+R(10)`, `[0, 600] s` (§ 4.3). **Rien ne
relie une entite `device` a un vehicule** : la confrontation des positions `ti=43` aux
emplacements de naissance mesures n'a pas eu lieu (§ 6 bis.4), et conclure sans elle referait
exactement l'erreur que l'utilisateur a signalee.

---

## 1. CE QUE LE LOT A REPRIS, ET CE QU'IL AJOUTE

| Deja etabli avant ce lot | Ou |
|---|---|
| `ti=11 i0` = `2 x R(7)`, `valeur - 1` ; couple d'INDEX de minuteur, FIGE sur toute la vie d'un objectif (112 slots d'Assaut, 0 variable) ; legalite 100,0 % sur 1 149 lectures d'image-cle ; voie delta = bruit | `objectif_ti11_minuteurs_verdict_test.go` (2026-09-01) |
| domaine legal d'un index : `{-1} U [0,63] U {65,66,67}` ; 65/66/67 = manche, mort subite, delai de grace (`FUN_140fe957c`) | idem |
| « la VALEUR du compte a rebours, si elle existe, est derriere l'index — dans `ti=0 i15 managed-engine-timers-component` » | idem, conclusion |
| `ti=12 i10` = meme `2 x R(7)` ; `i11`/`i12` = `R(17)` quantifie, pas de 50 ms | `NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17.md` § 14-16 |
| le retour automatique d'un drapeau ne se DEDUIT pas des evenements : 1,3 s a 35,8 s entre p10 et p90 sur le MEME film, max 111,6 s | `document_objectives_live.go`, « CE QUE LA MESURE A REFUSE DE PUBLIER » |
| cooldown de vehicule non mesurable a la resolution du recensement (mediane ~35 s, IQR/mediane 0,87-0,98) | `V2_SPAWNS_COOLDOWNS_2026-09-01.md` § 3 |
| la destruction d'un vehicule EST datee depuis le dead-state `ti=40 i11` | `NOTE_V13_DEADSTATE_VEHICULE_2026-09-05.md` ; `VehicleTrack.TEnd` |
| les minuteurs d'ARMES et d'EQUIPEMENT sont deja lus et affiches (`PadCycle`, `ti=42` / `ti=37`) | V22 bis du plan |

**Ce lot ajoute** : la grammaire complete de `managed-engine-timers-component` (jamais relevee),
celle de `device-object-dispenser-timer-component` (jamais relevee), le negatif MESURE sur
l'univers des noms de composant de l'executable, le chemin de port chiffre sur les sept
mini-bobines, et une chaine descripteur -> ecrivain rejouable SANS Ghidra.

---

## 2. LE BASSIN DE MINUTEURS — `managed-engine-timers-component`

`ti=0 i15` et `ti=2 i15`. Chaine : chaine `0x143c94990` -> accesseur `0x141175700` -> descripteur
`0x143d08ca0` -> **ecrivain `FUN_1407ee7b8`** (bornes `.pdata` `0x1407ee7b8..0x1407ee87a`,
194 octets) ; compagnon `+0x28` = `0x142edad74`.

### 2.1 La grammaire

```
R(64)                                       masque des 64 fentes du bassin
pour k = 0..63 croissant, si bit k = 1 :
    R(2)                                    etiquette de la fente
      0      -> 0 bit de plus               fente ETEINTE (octet d'etat = 0)
      1      -> R(16) + R(16) + R(5) + R(16)   quatre champs, bornes [0, 3600] s
      2 ou 3 -> R(16) + R(16) + R(5)           trois champs, bornes [0, 36000] s
```

Largeur totale : `64 + somme_k (2 | 39 | 53)`.

### 2.2 D'ou vient chaque ligne (desassemblage, `objdump`)

- `1407ee7e4 call 0x1406d6bac` avec `edx = 0x40` -> le masque `R(64)` dans `rbx`.
- Boucle `1407ee7f5..1407ee819` : `rbp = 1 << edi`, `edi` de 0 a `0x40` exclu, pas de fente
  `r14 += 0x14` (20 octets). L'adresse de la fente se calcule en `1407ee843` :
  `rcx = edi + 0x59` ; `rcx = edi + rcx*4` ; `rax = rsi + rcx*4` = `rsi + 20*edi + 0x590`.
  **Une fente = 20 octets, le bassin commence a `etat + 0x590`.**
- `1407ee850 movl $0x10,0x28(%rsp)` : le contexte passe au lecteur de fente porte la largeur
  **16**. `1407ee869 call 0x1407ee87c` : le lecteur de fente.
- Lecteur de fente `FUN_1407ee87c` : `addl $0x2,0x2c(%r10)` puis `shr $0x3e,%r9` -> **`R(2)`**.
  - `r9d == 0` -> `142325d2c` : `movb $0x0,0x10(%rax)` — RIEN n'est lu.
  - `r9d == 1` -> `142325d08` : `movss xmm3,[0x143cd86b4]` (= **3600,0f**), `movups (%rcx)` remis
    a zero sur 16 octets, `movb $0x1,0x10(%rcx)`, puis `call 0x142ba78dc`.
  - sinon -> `1407ee8d4` : `(%rcx)` et `0x8(%rcx)` remis a zero (12 octets), `movb $0x2,0x10(%rcx)`,
    puis `call 0x1424cd048` = `movss xmm3,[0x143cd8a84]` (= **36000,0f**) ; `jmp 0x140d580d0`.
- `FUN_140d580d0(dest, flux, n, max)` : `xorps xmm2` (min = 0), deux `FUN_1406d84b4` ecrivant
  `(%rdi)` et `0x4(%rdi)`, puis `jmp 0x1407f0354` = `addl $0x5,0x2c(%rcx)` -> **`R(n)+R(n)+R(5)`**.
- `FUN_142ba78dc(dest, flux, n, max)` : appelle `FUN_140d580d0` PUIS un troisieme
  `FUN_1406d84b4` ecrivant `0xc(%rsi)` -> **`R(n)+R(n)+R(5)+R(n)`**.

**Controle de coherence interne** : l'etiquette 1 remet 16 octets a zero et ecrit quatre champs
(`+0`, `+4`, `+0x0b`, `+0x0c`) ; l'etiquette 2 n'en remet que 12 et n'ecrit pas `+0x0c`. La forme
du nettoyage CONFIRME la difference de largeur ; les deux lectures ne sont pas independantes de
cette verification.

### 2.3 Les unites, et la preuve qu'elles sont des SECONDES

`DAT_143cd8a84 = 0x470CA000 = 36000,0f` est **deja dans le depot** sous le nom `RoundTimerMax`
(`grammar/quantize_endpoint.go:53-56`), avec `roundTimerBits = 16` et
`decodeGameEngineRoundTimer` qui lit `R(16)+R(16)+R(5)` (`vitality.go:175`). L'horloge de MANCHE
— affichee a l'ecran, exploitee en production — utilise donc **exactement le meme enregistrement**
que l'etiquette 2/3 d'une fente du bassin, et son ecrivain (`ti=0 i5 FUN_1407ee790`) appelle le
MEME `FUN_140d580d0` avec le MEME `r8d = 0x10` et la MEME borne. Les fentes du bassin sont des
secondes, quantifiees sur 16 bits, point-milieu avec extremites exactes
(`DequantEndpoint(q, 0, max, 16, false, true)`).

| etiquette | borne haute | pas | plage |
|---|---|---|---|
| 1 | `DAT_143cd86b4 = 0x45610000 = 3600,0f` | 3600 / 65534 ~ **54,9 ms** | 1 heure |
| 2 / 3 | `DAT_143cd8a84 = 36000,0f` | 36000 / 65534 ~ **549 ms** | 10 heures |

### 2.4 Ce que ce composant est, et ce qu'il n'est pas

C'est le BASSIN GLOBAL des minuteurs de la partie, porte par l'entite du moteur de jeu — une seule
entite, une fois par image-cle. `ti=11 i0` (objectif) et `ti=12 i10` (navpoint) n'en portent que
**deux INDEX chacun**, et le verdict du 2026-09-01 a mesure que ces index sont FIGES : ils ne
datent rien par eux-memes, ils DESIGNENT. Les index 65/66/67 sont les trois minuteurs reserves
(manche, mort subite, delai de grace), 0..63 les fentes de ce bassin, -1 « aucun minuteur ».

**Ce n'est donc pas « le minuteur du drapeau ».** C'etait l'endroit ou le minuteur du drapeau se
trouverait SI le drapeau en avait un — et **LA MESURE DU 2026-09-17 DIT QU'IL N'EN A PAS**. La
jointure index -> fente a ete jouee sur trois films entiers : `ti=11 i0` vaut `(-1, -1)`, « aucun
minuteur », sur 446 records d'objectif sur 446 (§ 6 bis.3). Ce paragraphe garde sa grammaire, qui
est juste et desormais verifiee sur film ; son hypothese de DESTINATION, elle, est REFUTEE.

---

## 2 bis. LA ROUTE VERS LE BASSIN, OUVERTE : LES TROIS GRAMMAIRES QUI MANQUAIENT (2026-09-17, voie libre)

Le § 6 mesure que 113 records sur 113 butent sur `i11`. Les trois composants non portes qui
precedent `i15` ont ete resolus par la meme chaine (calibration inchangee, 6/6) puis lus a
`objdump`. **Les trois grammaires sont decidables hors ligne, et elles suffisent : la marche
franchit `i15` et ne s'arrete qu'a `i16`** (§ 6 bis).

### 2 bis.1 `game-engine-soft-ceilings-component` (i11) — `R(128)` plat

Descripteur `0x143d0f560`, ecrivain `FUN_14116d1ac` (36 octets). Un seul appel :
`FUN_1406d676c(flux, _, dest = etat + 0x148, n = 0x80)`. `FUN_1406d676c` est le lecteur de BLOC
DE BITS du moteur : `cmp $0x40,%ebp ; jae` puis, dans la branche large (`1406d68a6`),
`add %ebp,0x2c(%r10)` par mot de 64 bits avec `ebp = 0x40`, `sub %ebp,%r11d` et rebouclage tant
que le reste est >= 64. Il lit donc EXACTEMENT `n` bits. **128 bits, inconditionnel, sans porte.**

### 2 bis.2 `game-engine-disabled-kill-volume-flags-component` (i13) — `R(13)` puis `n x R(1)`

Descripteur `0x143d0f510`, ecrivain `FUN_142f03498` (325 octets).
`addl $0xd,0x2c(%rdx)` puis `shr $0x33` : **`R(13)`** = le COMPTE. Puis la boucle
`142f03579..142f035be` : `call FUN_1406cf008` (= `R(1)`) par volume, `inc %esi`,
`cmp %ebx,%esi ; jl` — **un bit par volume**, ecrit en champ de bits a `etat + 0x164`.
Largeur : `13 + n`.

### 2 bis.3 `GameEngineComposerLetterboxComponent` (i14) — DEUX branches, et le NIVEAU decide

Descripteur `0x143d0f740`, ecrivain `FUN_142f0328c` (521 octets). Premiere instruction utile :
`cmp $0x2,%r9d ; jb 0x142f03405` — `r9d` est le NIVEAU du composant, celui que le registre porte
en `entree + 0x100`. `ecs_table.tsv` le donne : **`level = 2`** pour `ti=0 i14` comme pour
`ti=2 i14`. La branche prise est donc la HAUTE (`142f032b9`) :

```
R(1)                                                    -> etat + 0x564
R(16) quantifie sur [0, DAT_143cd8374]                  -> etat + 0x568
4 x FUN_142efd284 : R(1) ; si le bit vaut 0 -> R(7)     -> etat + 0x56c..0x57b  (PORTE INVERSEE)
4 x [ R(1) ; si 1 -> R(16) ]                            -> etat + 0x57c..       (mots de 16 bits)
```

Largeur : **25 bits au minimum, 117 au maximum.** `FUN_142efd284` est bien une porte INVERSEE
(`call FUN_1406cf008 ; test %al,%al ; je` -> la lecture de `R(7)` n'a lieu que si le bit vaut
ZERO ; a un, le champ prend `-1`). La branche BASSE (`level < 2`) differe : elle finit par un
`FUN_1406d676c(..., n = 0x40)` = `R(64)` que la branche haute n'a pas. **Un port qui ignorerait
le niveau se desynchroniserait de 64 bits.**

### 2 bis.4 Le quatrieme champ du lecteur de minuteur, et il compte

`FUN_142ba78dc(dest, flux, n, max)` n'est pas un alias de `FUN_140d580d0` : il l'appelle PUIS lit
un TROISIEME reel quantifie (`142ba790d call FUN_1406d84b4`, `movss %xmm0,0xc(%rsi)`). Donc :

| lecteur | champs | largeur |
|---|---|---|
| `FUN_140d580d0(dest, flux, n, max)` | `R(n) + R(n) + R(5)` | `2n + 5` |
| `FUN_142ba78dc(dest, flux, n, max)` | `R(n) + R(n) + R(5) + R(n)` | `3n + 5` |

Les trois composants portes du moteur (`ti=0 i5/i6/i7`) utilisent la forme COURTE — c'est ce que
le depot lit deja (`R(16)+R(16)+R(5)`, `vitality.go:175`). La forme LONGUE n'apparait qu'a
l'etiquette 1 du bassin et chez le distributeur de `ti=43 i37`.

---

## 3. LE COMPTE A REBOURS DU NAVPOINT — `ti=12 i11` / `i12`

Releve integralement au lot 3.6 (`NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17.md` § 15-16), RE-CALIBRE
par cet instrument : la chaine du descripteur retombe sur `FUN_142ed5194` (`i11`) et `FUN_142ed512c`
(`i12`), les deux adresses de la note (§ 7.2 — ce sont deux des six temoins de calibration).

```
i11 managed-navpoint-manual-timer-initial-duration-component
    R(17) quantifie sur [-0,025 ; 6553,5752] -> v = q * 0,05   (pas de 50 ms)
i12 managed-navpoint-manual-timer-current-duration-component
    R(17) meme quantification, puis zone morte |v| <= 1e-4 -> 0
```

**C'est un compte a rebours DIRECTEMENT LISIBLE, sans resoudre d'index** : duree INITIALE et duree
COURANTE du meme minuteur, donc la fraction ecoulee se lit sans etat. C'est la forme que prend a
l'ecran « le drapeau revient dans 0:12 ».

**RESERVE, ET ELLE EST DANS LE BRIEF** : un navpoint est un AFFICHAGE. Ce minuteur est ce que le
HUD montre, pas forcement la source du moteur. Mais les deux sont ecrits par le film et rien
n'oblige a choisir : le navpoint est le canal LE MOINS CHER (17 bits, aucune indirection, pas de
50 ms) et le bassin est le canal AUTORITATIF. Si la mesure les fait coincider, le navpoint suffit.

---

## 4. LES VEHICULES — LE NEGATIF, MESURE

### 4.1 L'univers des noms de composant de l'executable

`cmd_reapparition -inventaire` balaye les chaines C isolees des sections de donnees et retient
celles terminees par `-component` ou `-component-<n>` : **294 noms**. Le vocabulaire de la
reapparition s'y repartit ainsi (liste integrale, aucune troncature) :

| mot | frappes | ce que la liste contient |
|---|---|---|
| `timer` | **13** | biped-emp · **device-object-dispenser-timer** · game-engine-campaign-timer · game-engine-round-timer · **managed-engine-timers** · **managed-navpoint-manual-timer-current-duration** · **managed-navpoint-manual-timer-initial-duration** · managed-navpoint-timers · managed-objective-timers · player-respawn-timer · player-soft-kill-timer · player-unsafe-respawn-timer · **vehicle-emp-timer** |
| `respawn` | **10** | flock-forced-respawn · managed-object-participant-respawn-block · player-desired-respawn-{location,player,seat} · player-early-respawn-requested · player-primary-respawn-object · player-respawn-safety · player-respawn-timer · player-unsafe-respawn-timer |
| `spawn` | **15** | les 10 ci-dessus + participant-spawn-availability-mask-data · player-pending-join-in-progress-spawn · spawn-filter-{filters,type,weight} |
| `dispenser` | **5** | device-dispenser-{monitors-changed,require-los,state,state-flags} · device-object-dispenser-timer |
| `duration` | **3** | managed-navpoint-manual-timer-{current,initial}-duration · managed-objective-outro-phase-duration |
| `delay` | **1** | equipment-energy-delay-ticks-left |
| **`return`** | **0** | — |
| **`reset`** | **0** | — |
| `vehicle` | **15** | auto-turret x4 · emp-timer · equipment-turret-parent · low-frequency · seats-override-{pitch,yaw} · sentry-state · transformed-or-desired-open-state-changed · type-physics · type-state · weapon-set · player-vehicle-entrance-ban |

**Trois lectures, toutes fermes :**

1. **AUCUN composant de vehicule ne porte un temps de reapparition.** Le seul `vehicle-*` avec un
   temps est l'EMP. Ce n'est pas « absent du corpus » : c'est absent du MOTEUR.
2. **Tous les composants « respawn » sont ceux d'un JOUEUR** (ou d'une nuee IA pour
   `flock-forced-respawn`). Le moteur ne replique pas la reapparition d'un OBJET par un composant
   dedie.
3. **`return` et `reset` rendent ZERO.** Le retour d'un drapeau n'a donc pas de composant a lui :
   il passe par les canaux generiques (le bassin, le navpoint, la jauge `ti=11 i12`), ce qui
   recoupe la mecanique Lua de `parcel_deliver_object.lua` (`flagReturnTimer`, `flagResetSeconds`,
   etats `Resetting` / `Returning` — memoire `reference_ctf_flag_lua_mechanics`) : c'est un SCRIPT
   de mode qui la tient, pas un composant du moteur.

### 4.2 Le vocabulaire de l'API du jeu confirme la nature du delai vehicule

Chaines relevees dans l'image (pool complet, hors composants) :
`ManagedGameVariant_{Get,Set}VehicleRespawnTimeOverrideFor{All,Channel,Class}`,
`ManagedGameVariant_{Get,Set}VehicleInitialSpawnDelayOverrideFor{All,Channel,Class}`,
`ManagedGameVariant_{Get,Set}VehicleSpawnLogicOverrideFor...`,
`AI_GetVehicleSpawnedCountFromSpawner`, `AI_GetFailedVehicleSpawnCountFromSpawner`,
`m_forgeRespawnTime`, `forge_object_properties_does_respawn`, `DoesRespawn`, `InheritRespawnTime`.

Le delai de reapparition d'un vehicule est donc (a) une propriete de PLACEMENT de la carte
(Forge / scenario) et (b) une SURCHARGE DE VARIANTE. Les deux sont des REGLES connues avant le
match, pas un etat replique image par image. Un film n'a aucune raison de les ecrire, et la mesure
dit qu'il ne les ecrit pas.

### 4.3 Le seul minuteur de GENERATEUR de l'image — et pourquoi il n'est pas conclu

`device-object-dispenser-timer-component` (`ti=43 i37`). Chaine `0x143c98a48` -> accesseur
`0x1411773c0` -> descripteur `0x143d0c340` -> **ecrivain `FUN_142f02c94`**
(`0x142f02c94..0x142f02d01`, 109 octets).

```
pour k = 0 et 1  (boucle rbx de etat+0x700 a etat+0x720, pas 0x10 : DEUX fentes) :
    R(1)  porte
      1 -> R(10) + R(10) + R(5) + R(10)   bornes [0, 600] s   (FUN_142ba78dc, n = 0xa)
      0 -> rien de plus (la fente est videe : `andl $0x0,0x4(%rbx)`, `movb $0x1,0xb(%rbx)`)
```

Largeur : 2 a 72 bits. Borne `DAT_143d13304 = 0x44160000 = **600,0f**` (10 minutes) ; pas
600 / 1022 ~ **587 ms**.

**C'est un vrai compte a rebours de generateur, et c'est le SEUL de l'image.** Mais rien, dans ce
lot, ne relie une entite `device` a un vehicule :

- V22 bis a etabli que les socles d'ARME et d'EQUIPEMENT ne passent PAS par les devices
  (`ti=42` / `ti=37`, `PadCycle`) — le pilote s'etait deja trompe une fois en les y rattachant ;
- `V2_SPAWNS_COOLDOWNS_2026-09-01.md` § 1 a mesure les emplacements de naissance des vehicules
  (rayon d'amas **0,00 m** ; 6 pads sur Behemoth en grille 2x3, 4-5 sur Launch Site) et
  **n'a jamais confronte ces positions a celles des devices** ;
- les composants `device-*` deja portes decrivent des portes et des ascenseurs
  (`device-position-component`).

**Conclure ici serait refaire l'erreur de V22 bis.** Le test qui tranche est en § 8.

---

## 5. CE QUE LE REJEU PEUT DEJA FAIRE DES VEHICULES — ET LA MESURE DU 2026-09-17 QUI CORRIGE CE PARAGRAPHE

> **CORRIGE LE 2026-09-17 SUR PIECES, APRES LA VOIE LIBRE.** La version initiale de cette
> section affirmait que le cycle de reapparition d'un vehicule etait « mesurable AUJOURD'HUI ».
> **C'EST FAUX, et la mesure sur deux films entiers le dit** (§ 6 ter). Le paragraphe est
> reecrit ci-dessous ; l'affirmation retiree est nommee pour qu'elle ne revienne pas.

`V2_SPAWNS_COOLDOWNS` a conclu « cooldown NON mesurable » pour une raison precise et datee : la
fin de vie d'un vehicule etait BORNEE par le recensement des images-cles, soit +/-20 s, et un
cooldown de 20-35 s n'en sort pas. Cette raison-la EST tombee le 2026-09-05 : le dead-state de
`ti=40 i11` est lu et le rejeu publie `VehicleTrack.TEnd`, date a la milliseconde.

**MAIS UNE SECONDE CAUSE, JAMAIS MESUREE, PREND SA PLACE : `TEnd` EST QUASI TOUJOURS ABSENT.**
Mesure du 2026-09-17, par la cuisson de PRODUCTION (`BuildFromFilm`) sur les deux BTB Heavies :

| film | carte | vies de vehicule | naissance situee | **fin DATEE (`TEnd`)** |
|---|---|---|---:|---:|
| `a349fea8` | Fragmentation Heavies | 109 | 90 | **1** |
| `a521164d` | Fragmentation Heavies | 42 | 37 | **1** |

**Une fin datee sur 109, une sur 42.** Le cycle exige, par vie, la fin datee de la vie
PRECEDENTE au meme emplacement : sur 31 emplacements agglomeres a 2 m pour `a349fea8` (dont 15
credibles, 2 a 8 vies chacun), le compte des ecarts mesurables est **ZERO**, et les 56 occasions
sont toutes comptees comme des MANQUES. Aucun cycle n'est etabli, sur aucun emplacement, sur
aucun des deux films.

CE QUI RESTE VRAI, ET CE QUI NE L'EST PLUS :

- VRAI : le film n'ecrit AUCUN minuteur de reapparition de vehicule (§ 4, negatif mesure sur
  l'univers des 294 noms de composant). La reapparition ne peut que se DEDUIRE.
- VRAI : la FORME de la deduction est celle de `PadCycle` (`document_ground_weapons.go:111`) —
  mediane, deciles, ecarts mesures, manques comptes, cle ABSENTE quand le cycle n'est pas etabli.
- **PLUS VRAI** : que la deduction soit faisable aujourd'hui. Elle ne l'est pas. Le prealable
  n'est plus la resolution du recensement, c'est **l'ATTRIBUTION de la fin de vie** : pourquoi
  `TEnd` ne se renseigne-t-il qu'une fois sur cent ? Le journal de cuisson de `a349fea8` dit
  `vehicules : balaye=true` et publie 109 vies, donc le calque marche ; c'est le rattachement du
  dead-state a la vie qui ne se fait pas. La reserve 2 de `NOTE_V13_DEADSTATE_VEHICULE` § 5
  (« sur-comptage certain : il faut grouper les dead-states consecutifs d'un meme slot en UN
  episode de mort. Non fait. ») designe le meme endroit.

C'est donc cela, la reponse produite pour les vehicules, et elle est en deux temps : **le film
n'ecrit pas le minuteur, et la deduction attend que la fin de vie soit attribuee.** Le lot de
port devra ouvrir CE prealable avant de publier un `VehicleCycle` — sinon il publierait un calque
vide sur 99 % des vies.

---

## 6. CE QUI SEPARE LE DECODEUR DU BASSIN — mesure sur les sept mini-bobines

`TestReapparition37CheminVersLeBassin`, sept mini-bobines par build, AUCUN film du cache.
Pour chaque record d'image-cle BORNE : ferme-t-il, et sinon, quel est le premier composant present
non porte ?

| bobine | mode | entite moteur | records | fermes | bloquant |
|---|---|---|---|---|---|
| `a521164d` | Total Control Heavies | `ti=2` | 11 | 0 | `i11 game-engine-soft-ceilings-component` (11/11) |
| `60ae07c4` | Oddball | `ti=2` | 30 | 0 | idem (30/30) |
| `11de8353` | BTB Fiesta | `ti=2` | 16 | 0 | idem (16/16) |
| `111fa685` | BTB Fiesta | `ti=2` | 14 | 0 | idem (14/14) |
| `e5adf7b2` | BTB Fiesta | `ti=2` | 11 | 0 | idem (11/11) |
| `bcb6d393` | **CTF:Arena** | `ti=2` | 19 | 0 | idem (19/19) |
| `fb1a1a72` | **CTF:Arena** | `ti=2` | 12 | 0 | idem (12/12) |

**113 records, 113 fois le MEME bloquant.** L'entite du moteur de jeu est presente a CHAQUE
image-cle de CHAQUE bobine — une image-cle est un ETAT COMPLET, donc le bassin de minuteurs est
ECRIT 19 fois dans le film CTF `bcb6d393`, 12 fois dans `fb1a1a72`. Il n'est pas lu, et une seule
chose l'empeche : `i11 game-engine-soft-ceilings-component`, non porte, sur la route de `i15`.

Les deux autres archetypes de la question, pour memoire (meme passe) :

| archetype | bloquant | distance au champ voulu |
|---|---|---|
| `ti=11` managed-objective | `i4 managed-objective-interaction-filter-component` (100 %) | `i0` est AVANT le bloquant : deja lu |
| `ti=12` managed-navpoint | `i1 managed-navpoint-flags-component` (~98 %) | `i11`/`i12` sont 10 composants plus loin |
| `ti=43` device | `i19 device-position-animation-name-component` | `i37` est 18 composants plus loin |
| `ti=29` respawn-block | aucun — **ferme a 100 %** sur 5 bobines sur 7 | sans objet (c'est le joueur) |
| `ti=40` vehicule | aucun ; marche complete, fin au mauvais bit | sans objet (§ 4.1) |

**NEGATIF D'INSTRUMENT, CONSIGNE** : une premiere version de cette passe comptait les bits de
`EntityTrace.Mask` pour recenser ce qu'une image-cle DECLARE — la recette qui avait debloque le
dead-state des vehicules (`NOTE_V13_DEADSTATE_VEHICULE` § 3). **Elle ne vaut pas sur les
images-cles** : l'image-cle est un ETAT COMPLET, il n'y a pas de masque de presence, et le champ
rendait 64 bits a UN sur `ti=2`, `ti=11` et `ti=12` — au-dela meme du nombre de composants
declares au registre (18, 34, 28). La recette V13 vaut pour les records DELTA. La difference est
de CADRE, pas de corpus ; ne pas la re-essayer sur une image-cle.

---

## 6 bis. LE BASSIN LU SUR CINQ FILMS ENTIERS — ET IL NE PORTE PAS LE MINUTEUR DU DRAPEAU

> Voie libre du pilote, 2026-09-17. **CINQ FILMS, UN PAR UN**, par deux instruments cibles ;
> jamais de balayage du corpus, ni `replay-equiv`, ni corpus gate. Films choisis dans
> `config/replay_corpus.toml` (mode et carte lus la, jamais en base) :
>
> | court | mode | carte | pourquoi |
> |---|---|---|---|
> | `bcb6d393` | CTF:Arena | Cliffhanger | CTF mono-manche, temoin canonique du corpus |
> | `fb1a1a72` | CTF:Arena | Banished Narrows | CTF multi-manche |
> | `d9781168` | Oddball:Arena | Dredge | le crane |
> | `a349fea8` | BTB Heavies:Total Control | Fragmentation Heavies | vehicules, Heavies |
> | `a521164d` | BTB Heavies:Total Control | Fragmentation Heavies | vehicules, second temoin |

### 6 bis.1 LA LECTURE EST ACQUISE SUR LES TROIS FILMS D'ARENE, ET ELLE ECHOUE SUR LES DEUX BTB

| film | records de moteur marches | fentes allouees (union des masques) | verdict |
|---|---:|---|---|
| `bcb6d393` | 19 | `0x00000000000003ff` = **10, contigues** | LECTURE ACQUISE |
| `fb1a1a72` | 41 | `0x0000000000001fff` = **13, contigues** | LECTURE ACQUISE |
| `d9781168` | 37 | `0x000000000000007f` = **7, contigues** | LECTURE ACQUISE |
| `a349fea8` | 53 | `0xffffffffffffffcf` = 62, **tout-a-un** | LECTURE REFUSEE (bruit) |
| `a521164d` | 20 | `0xb36fffffffffffc6` = 55, **tout-a-un** | LECTURE REFUSEE (bruit) |

**Un masque de 64 fentes presque tout a un n'est pas une mesure, c'est une signature de
desalignement** — le meme raisonnement que l'oracle `n2` du golden 0.A.3. Les trois films
d'arene rendent au contraire des PREFIXES CONTIGUS (0..9, 0..12, 0..6) : un pool alloue
sequentiellement, exactement ce qu'un bassin doit ressembler. La grammaire de ce lot vaut donc
sur les builds d'arene mesures et PAS sur les deux BTB Heavies ; c'est la doctrine du profil par
build, et le port devra la re-mesurer par build (§ 8).

Sur les cinq films, la marche franchit `i15` et s'arrete a **`i16 scenario-intro-component`** —
non porte, SITUE APRES le bassin. La regle de `NOTE_V13_DEADSTATE_VEHICULE` § 3 s'applique :
`DesyncAt` est l index du PREMIER composant non porte, donc tout ce qui precede a ete consomme
dans l ordre. La fermeture n'est donc pas atteignable sans porter `i16`/`i17`, et l'oracle de
justesse est ailleurs — § 6 bis.2.

### 6 bis.2 L'ORACLE QUI REMPLACE LA FERMETURE : LA COHERENCE TEMPORELLE DU DECOMPTE

La lecture rend, par fente, `a` (le premier reel) et `b` (le second). Sur `bcb6d393`, fentes 7, 8
et 9, aux images-cles espacees de 20,0 s :

| image-cle | `a` | `b` | reste consomme depuis la precedente |
|---|---|---|---|
| t = 40,0 s | 29,939 s | 24,995 s | — |
| t = 60,0 s | 29,939 s | 5,219 s | **19,776 s** pour 20,0 s ecoulees |
| t = 80,0 s | 29,939 s | 0,000 s | epuise |

**`a` est la DUREE TOTALE, `b` le TEMPS RESTANT, et le reste decroit au rythme REEL a 1,1 % pres
— sur une lecture de 16 bits quantifiee au pas de 0,549 s, c'est-a-dire a la resolution du
quantum.** Aucune fenetre mal posee ne produit cela : un decompte qui suit l'horloge du film sur
trois echantillons consecutifs est une preuve de justesse plus forte qu'une fermeture, parce
qu'elle porte sur la VALEUR et non sur la position.

Quanta de duree totale observes, IDENTIQUES sur les trois films d'arene (aucune autre valeur) :

| quantum `qa` | duree | `bcb6d393` | `fb1a1a72` | `d9781168` |
|---|---|---:|---:|---:|
| 0 | 0,000 s | 71 | 163 | 150 |
| 1 | 0,275 s | 31 | 64 | 57 |
| 51 | **27,741 s** | 22 | 136 | 3 |
| 55 | **29,939 s** | 48 | 157 | 42 |

Toutes les fentes vivantes portent l'etiquette **2** (echelle `[0, 36000]` s) ; l'etiquette 1 et
sa forme longue n'apparaissent sur aucun des cinq films.

### 6 bis.3 LA JOINTURE NE SE FAIT PAS : `ti=11 i0` VAUT `(-1, -1)` SUR 446 RECORDS SUR 446

C'etait LA question du lot : l'index de minuteur d'un objectif designe-t-il une fente du bassin ?
`ti=11 i0` est lu PAR LA MEME MARCHE que le bassin (meme cadre d'image-cle, meme etat par defaut,
meme frontiere — deux cadres differents ne se joignent pas), et il est le PREMIER composant apres
l'en-tete, donc acquis sans rien porter.

| film | records `ti=11` lus | couples de quanta distincts | index sous la lecture (a) `q-1` | sous la lecture (b) `q/2-1` |
|---|---:|---|---|---|
| `bcb6d393` | 85 | `(0, 0)` seulement | `(-1, -1)` | `(-1, -1)` |
| `fb1a1a72` | 195 | `(0, 0)` seulement | `(-1, -1)` | `(-1, -1)` |
| `d9781168` | 166 | `(0, 0)` seulement | `(-1, -1)` | `(-1, -1)` |

**446 records, un seul couple, et c'est « AUCUN MINUTEUR ».** Sur ces trois films, aucun objectif
du HUD — ni le drapeau de deux CTF, ni le crane d'un Oddball — ne branche de fente du bassin. La
jointure index -> fente n'existe pas sur ce corpus, et ce n'est pas un defaut de lecture : c'est
la valeur que le film ecrit.

DEUX CONSEQUENCES, TOUTES DEUX FERMES :

1. **LE BASSIN N'EST PAS LA VOIE DU RETOUR DU DRAPEAU.** L'hypothese de la version initiale de
   cette note (« c'est lui que designent les index de `ti=11 i0` ») est REFUTEE sur les trois
   films d'arene mesures. Le bassin porte de vrais comptes a rebours — mais pas ceux-la.
2. **LES MINUTEURS DU BASSIN SONT GENERIQUES, PAS MODAUX.** Les memes deux durees (27,741 s et
   29,939 s) apparaissent en CTF **et** en Oddball, ou il n'y a pas de drapeau ; et le nombre de
   fentes allouees (10, 13, 7) suit l'ordre de grandeur du nombre de JOUEURS, pas celui des
   objectifs (2 drapeaux, 1 crane). Trois fentes qui portent la MEME valeur au MEME instant
   (7, 8, 9 sur `bcb6d393`) se lisent alors comme trois minuteurs demarres ensemble.

**PISTE NOMMEE, NON PROUVEE — et elle ne doit pas etre presentee autrement.** L'executable porte
la chaine `Engine_SetSpawnTimerAndTotalRespawnDurationForPlayer` : « minuteur de reapparition ET
DUREE TOTALE de reapparition POUR UN JOUEUR », c'est-a-dire exactement la forme mesuree
(`a` = duree totale, `b` = restant) et exactement la granularite mesuree (par joueur).
`Player_GetRespawnTimerCountdown`, `NormalizedTimeRemainingUntilRespawn` et
`RespawnGrowthInSeconds` / `MaximumRespawnTimeInSeconds` sont du meme vocabulaire. **Ce qui
manque pour conclure : coller un demarrage de fente a une MORT datee du fil des morts.** Ce
n'est pas fait, c'est cheap, et c'est le premier item du § 9.

### 6 bis.4 CE QUE LA VOIE LIBRE N'A PAS PU MESURER, ET POURQUOI — DIT SANS ENJOLIVER

- **`flagCarries` n'est pas cuit par cet instrument.** `BuildFromFilm` avec le seul
  `Options.MapQuant` publie 0 intervalle de drapeau sur les deux CTF (`DRAPEAUX : aucun`) : le
  calque exige le catalogue versionne d'objectifs de carte, joint par `map_id`
  (`document_objectives_live.go`, « le DRAPEAU le socle `flag_spawn` de la carte »), que
  l'instrument ne fournit pas. La collation demandee « bassin contre `flagCarries.dropped` » n'a
  donc PAS eu lieu. Elle n'aurait de toute facon rien nomme : la jointure par `ti=11 i0` est un
  negatif mesure (§ 6 bis.3), donc il n'y a pas de fente de drapeau a coller.
- **`ti=12 i11`/`i12` (le minuteur manuel du navpoint) n'est pas mesure.** Le bloquant de `ti=12`
  est `i1`, dix composants avant : les franchir demande les cinq blocs de filtres et leur
  `param_4`, releves mais non portes. C'est desormais LA voie la plus prometteuse pour les
  objectifs, et le § 9 la chiffre.
- **`ti=13 i0` par la voie delta n'est pas mesure.** Meme raison qu'au premier commit : `i0` est
  marche mais pas recolte, et il faudrait hacher les noms Lua candidats.
- **Les positions des entites `ti=43` ne sont pas confrontees aux emplacements de vehicule.** Le
  document de rejeu ne publie pas les devices ; les extraire demande un troisieme instrument.

---

## 7. LA METHODE — LA CHAINE DU DESCRIPTEUR SANS GHIDRA

### 7.1 Pourquoi

Ghidra n'etait pas lance pendant ce lot : aucun processus Java, `127.0.0.1:8089` refuse la
connexion, aucun socket de decouverte. Relancer une analyse complete de `HaloInfinite.exe` coute
des heures. La chaine de `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` § 1 etant purement
MECANIQUE, elle a ete rejouee en Go sur le PE.

### 7.2 La chaine, et sa calibration

`film/research/reapparition/` : charge le PE (`debug/pe`), indexe les sections et le repertoire
d'exceptions (`.pdata`, `RUNTIME_FUNCTION` — bornes de fonction EXACTES), puis pour un nom de
composant : chaine C isolee en donnees -> accesseur `REX.W LEA reg,[rip+chaine] ; RET` (8 octets)
-> unique slot de `.rdata` pointant cet accesseur = `descripteur + 0x18` -> ecrivain a
`descripteur + 0x40`. Chaque pas exige l'UNICITE ; sinon la chaine s'arrete et le dit.

**Six temoins, six concordances** (la passe REFUSE de publier une adresse neuve si un seul rate) :

| temoin | attendu | source | lu |
|---|---|---|---|
| `ti=9 i3 managed-player-back-button-scoreboard-flair` | `142ed5af4` | NOTE_3_6_METHODE § 3 | OK |
| `ti=12 i0 managed-navpoint-sub-type` | `1410e0cac` | idem | OK |
| `ti=12 i14 managed-navpoint-radial-progress` | `140fc8d14` | idem | OK |
| `ti=43 i19 device-position-animation-name` | `1410156e4` | idem | OK |
| `ti=12 i11 manual-timer-initial-duration` | `142ed5194` | NOTE_3_6_TI12_GRAMMAIRES_A § 15 | OK |
| `ti=12 i12 manual-timer-current-duration` | `142ed512c` | idem § 16 | OK |

La SIGNATURE de famille est MESUREE sur ces temoins, pas recopiee : `+0x00 = 0x141191ab0`,
`+0x08 = 0x14076ced0`, `+0x10 = 0x14117b4a0`, `+0x20 = 0x1404ab600`, `+0x30 = 0x1411c8f80`,
`+0x38 = 0x14076ce9c`, `+0x48 = 0x1404ab600` — ce qui **confirme la correction du § 17 pt 7 de
`NOTE_3_6_TI12_GRAMMAIRES_A`** (la note de methode ecrivait `0x141c8f880` et `0x14049b600`, qui
sont des transpositions de chiffres).

### 7.3 Les douze cibles resolues (toutes, aucune en echec)

| archetype | composant | descripteur | ecrivain `+0x40` | compagnon `+0x28` |
|---|---|---|---|---|
| ti=0/2 i15 | managed-engine-timers | `143d08ca0` | **`1407ee7b8`** | `142edad74` |
| ti=11 i0 | managed-objective-timers | `143d08d90` | **`142ed5a6c`** | `142edbac8` |
| ti=12 i10 | managed-navpoint-timers | `143d084d0` | **`1410d9040`** | `142edb0f4` |
| ti=11 i32 | managed-objective-outro-phase-duration | `143d08f70` | `142ed5634` | `142edb740` |
| ti=43 i37 | device-object-dispenser-timer | `143d0c340` | **`142f02c94`** | `142f05b88` |
| ti=43 i36 | device-dispenser-state | `143d0c390` | `142f02bb0` | `142f0591c` |
| ti=43 i32 | device-dispenser-state-flags | `143d0c4d8` | `142f02bcc` | `142f05934` |
| ti=43 i31 | device-dispenser-monitors-changed | `143d0c100` | `142f02a48` | `142f057f4` |
| ti=20 i2 | spawn-filter-filters | `143d05d98` | `142ed7068` | `142edb5cc` |
| ti=40 i37 | vehicle-emp-timer | `143d0b480` | `142f049dc` | `142f08c94` |
| ti=29 i0 | managed-object-participant-respawn-block | `143c96f40` | `142ed6a20` | `142edcc8c` |
| ti=0 i5 | game-engine-round-timer (temoin positif) | `143d0f5b0` | `1407ee790` | `142f06700` |

**DECOUVERTE DE BORD, non traitee (§ 9)** : `ecs_table.tsv` porte `FUN_142edbac8` en `deser_addr`
de `ti=11 i0`, et `FUN_142ed6a20` pour `ti=29 i0`. La premiere est le COMPAGNON `+0x28` (le
serialiseur), pas l'ecrivain `+0x40` — la colonne melange les deux conventions selon la vague qui
l'a remplie. Les deux fonctions lisent les memes largeurs (symetrie du moteur), donc aucune
grammaire n'est fausse ; c'est une incoherence de TABLE.

---

## 8. LE CHEMIN DE PORT (lot post-M4, a instruire par le pilote)

### 8.1 Objectifs par le bassin — CE CHEMIN EST DESORMAIS SANS OBJET POUR LE DRAPEAU

> **STATUE LE 2026-09-17 (voie libre).** Les trois etapes ci-dessous ont ete JOUEES dans un
> instrument de recherche : les grammaires de `i11`, `i13` et `i14` sont relevees (§ 2 bis), le
> bassin est lu sur trois films entiers (§ 6 bis.1), et l'etape 3 — la jointure a l'index de
> `ti=11 i0` — est un NEGATIF MESURE (§ 6 bis.3). **Porter le bassin ne donnera donc PAS le
> minuteur du drapeau.** Le chemin reste ecrit parce qu'il vaut pour lui-meme (le bassin porte de
> vrais comptes a rebours ; le § 9.5 item 1 dit comment les nommer) et parce que les trois
> grammaires sont acquises. La voie des objectifs est le § 8.2.

### 8.1 bis Le chemin, tel qu'il a ete parcouru en recherche

1. **Porter `ti=0/2 i11 game-engine-soft-ceilings-component`**, puis re-mesurer avec
   `TestReapparition37CheminVersLeBassin` : le bloquant doit avancer (candidats suivants au
   registre : `i13 game-engine-disabled-kill-volume-flags`, `i14 GameEngineComposerLetterboxComponent`).
   Le ratchet 0.A.3 doit voir le bloquant du golden avancer d'un index a chaque port — c'est le
   gate, il existe deja.
2. **Porter `i15 managed-engine-timers-component`** avec la grammaire du § 2.1 :
   `consumeEngineTimers` — `R(64)` de masque, puis par fente `R(2)` et, selon l'etiquette, le
   lecteur de minuteur DEJA ECRIT (`decodeGameEngineRoundTimer`, `vitality.go:175`, qui lit
   exactement `R(16)+R(16)+R(5)` et dequantifie par `DequantEndpoint(q, 0, max, 16, false, true)`)
   avec `max = 3600` ou `36000`, plus un quatrieme `R(16)` pour l'etiquette 1. **REUTILISER ce
   lecteur, ne pas en ecrire un second** : regle du depot, et la borne 36000 y est deja nommee
   `RoundTimerMax`.
3. **Joindre l'index a la fente** : `ti=11 i0` publie deja son couple d'index
   (`ObjectiveFieldTimers`, `ObjectiveTimerValue(q) = q - 1`). Le rejeu lirait, par objectif,
   `bassin[index]` a chaque image-cle.
   **PREALABLE NON NEGOCIABLE** : la RESERVE du 2026-09-01 — le quantum de `ti=11 i0` est TOUJOURS
   PAIR sur 1 149 lectures d'image-cle, ce qui est compatible avec une boucle de composants
   demarrant UN BIT TROP LOIN. L'epreuve a faire AVANT d'exploiter un index est la CONTIGUITE
   (relire `i0` la fenetre reculee d'un bit et regarder si le couple devient contigu), pas la
   legalite. Un index faux d'un rang designerait la mauvaise fente, donc le mauvais minuteur.

### 8.2 Objectifs — la voie courte, en parallele

Porter `ti=12 i1`..`i12` (grammaires COMPLETES en `NOTE_3_6_TI12_GRAMMAIRES_A`, ordre le moins
cher donne en § 17 pt 4 de cette note) donne `i11`/`i12` : duree initiale et duree courante, pas de
50 ms, sans aucune indirection. C'est **le chemin le moins cher vers un compte a rebours affichable**
et il ne depend d'aucune des reserves ci-dessus. Il exige `paramByComponent` pour les cinq
`*-filter(s)-component` (§ 17 pt 1 de la meme note) — sans quoi la marche se desynchronise des le
premier filtre.

### 8.3 Ce que le rejeu publierait

Sur le modele de `PadCycle` (une donnee MESUREE, cle absente quand elle n'est pas etablie) :

- **objectifs** : par objet d'objectif et par intervalle, `returnS` (duree restante avant retour
  automatique) et `returnTotalS` (duree initiale) — ecrits SEULEMENT quand le film les porte.
  Champ optionnel, `SchemaVersion` +1 (le calque du drapeau existe deja : `flagCarries`).
- **vehicules** : un `VehicleCycle` par EMPLACEMENT de naissance (les pads mesures a 0,00 m de
  rayon), meme forme que `PadCycle` — mediane, p10, p90, `gaps`, `missing` —, l'ecart etant mesure
  de `TEnd` (dead-state, date a la ms) a la naissance suivante au meme emplacement. **`SchemaVersion`
  +1**, et `document_vehicles.go` porte deja `T0`, `TEnd` et `Spawn` : le calcul est une couche
  d'analyse, pas un decodage neuf.

---

## 9. CE QUI RESTE, APRES LA VOIE LIBRE — ET CE QUE CHAQUE ITEM COUTE

> Cette section remplace la liste « ce que je veux lire sur film entier » du matin : les quatre
> mesures qu'elle demandait ont ete jouees ou explicitement rendues impossibles, et le § 6 bis
> dit laquelle est laquelle. Ce qui suit est ce qui reste, avec son cout et son gate.

### 9.1 Les quatre mesures de la voie libre, statuees

| mesure demandee | statut | ou |
|---|---|---|
| (1a) le bassin porte-t-il le minuteur du drapeau (index `ti=11 i0` -> fente) | `[x]` **REPONDU : NON**, 446 records sur 446 a `(-1, -1)` | § 6 bis.3 |
| (1b) le navpoint du drapeau lache porte-t-il un minuteur manuel (`ti=12 i11`/`i12`) | `[!]` NON MESURE — bloquant `i1`, dix composants a porter | § 9.2 |
| (1c) `ti=13 i0` par la voie delta | `[!]` NON MESURE — `i0` marche mais pas recolte | § 9.3 |
| (2) Oddball : minuteur de remise a zero du crane | `[x]` **REPONDU : le bassin ne le porte pas** — `ti=11 i0` a `(-1, -1)` sur 166 records, memes durees generiques qu'en CTF | § 6 bis.3 |
| (3a) un generateur de vehicule est-il un `device` | `[!]` NON MESURE — le document ne publie pas les devices | § 9.4 |
| (3b) le cycle de reapparition depuis `T0`/`TEnd` | `[x]` **REPONDU : NON MESURABLE** — 1 fin datee sur 109 et sur 42, zero ecart sur 31 emplacements | § 5 |

### 9.2 `ti=12 i11`/`i12` — LE CHEMIN LE PLUS COURT VERS UN COMPTE A REBOURS D'OBJECTIF

Ce qu'il faut porter, dans l'ordre le moins cher donne par `NOTE_3_6_TI12_GRAMMAIRES_A` § 17 pt 4 :
`i1` (`R(8)`), `i7` (`R(8)`), `i8` (`R(32)`), `i10` (`2 x R(7)`, reutiliser
`consumeObjectiveTimers`), puis `i11`/`i12` (`R(17)`). **Prealable non negociable** :
`paramByComponent` doit poser `param_4 = 3` pour `i2` et `= 2` pour `i3`..`i6`, sinon la marche
se desynchronise des le premier bloc de filtres (meme note, § 17 pt 1). Gate : le ratchet 0.A.3
doit voir le bloquant de `ti=12` avancer de `i1` vers `i13`.

Ce que la mesure dira ensuite, et le critere s'ecrit MAINTENANT : `i12` (duree courante) doit
etre non nul PENDANT les intervalles `dropped` d'un CTF et nul en dehors. Le calque `flagCarries`
est l'oracle, et il faut le CUIRE AVEC le catalogue d'objectifs de carte — ce que l'instrument de
collation de ce lot ne fait pas (§ 6 bis.4).

### 9.3 `ti=13` — la propriete reseau nommee par le script de mode

Inchange depuis le matin : `Engine_CreateFloatNetworkedProperty` /
`NetworkedProperty_SetFloatProperty` laissent le NOM au script, donc il n'apparait pas dans
l'univers des 294 noms de composant du § 4 — **le negatif de ce lot ne couvre pas ce canal**. Tag
3 = `R(24)` quantifie sur `[-100, +100]`, l'echelle de la jauge de retour vaut 100 dans
`parcel_deliver_object.lua`, et le meme canal porte deja la jauge de capture des zones en
production. Bloquants : `i0` (le `StringId` du nom) est marche mais pas recolte, et l'etat par
defaut de `ti=13` est desaligne en image-cle (oracle `n2`) — seule la voie DELTA parle. Preuve
demandee : recolter `i0` en delta sur un CTF, hacher `flagReturnTimer` / `flagResetSeconds` /
`onReturnProgress` par le hacheur de `StringId` du jeu, et chercher un slot dont le tag 3 monte
vers 100 pendant un `dropped`.

### 9.4 Les devices contre les emplacements de vehicule

`device-object-dispenser-timer-component` reste le seul minuteur de generateur de l'image. Le
test qui tranche : extraire les positions des entites `ti=43` (`object-position` i0 et
`device-position` i18, tous deux PORTES — le bloquant de `ti=43` est `i19`, donc la regle de V13
s'applique) et les confronter aux emplacements de naissance de vehicule que ce lot a deja
agglomeres (31 sur `a349fea8`, dont 15 credibles a 2-8 vies). Seuil a ecrire AVANT la mesure ;
`V2_SPAWNS_COOLDOWNS` § 2 avait pose « >= 80 % des amas a < 1 m d'un emplacement declare » pour
la confrontation `.mvar`, et il est reutilisable tel quel.

### 9.5 Ce que la lecture du bassin doit encore prouver, et le cout est faible

1. **NOMMER les minuteurs du bassin.** Coller un DEMARRAGE de fente (le premier echantillon ou
   `b` devient non nul) a une MORT datee du fil des morts, deja lu par le rejeu. Si les
   demarrages tombent sur des morts, la piste
   `Engine_SetSpawnTimerAndTotalRespawnDurationForPlayer` est confirmee et le bassin est classe
   « minuteurs de reapparition des joueurs » — donc SANS INTERET pour ce chantier, ce qui est
   une reponse utile. Instrument : les deux de ce lot, plus une jointure ; aucun port.
2. **LE PROFIL PAR BUILD.** La lecture du bassin echoue sur les deux BTB Heavies (masques
   tout-a-un, § 6 bis.1) et reussit sur les trois films d'arene. Avant tout port, mesurer sur
   quel ensemble de builds la grammaire tient — la cause probable est le NIVEAU de `i14` (la
   branche `level < 2` lit `R(64)` de plus) ou l'etat par defaut de `ti=2`.
3. **`i16 scenario-intro-component` et `i17 matchflow-isplaying-flags-component`** : les porter
   rendrait la FERMETURE, donc l'oracle de justesse gratuit du ratchet 0.A.3 sur l'archetype du
   moteur. Sans eux la lecture du bassin restera toujours justifiee par la seule coherence
   temporelle — ce qui est solide mais ne se met pas sous ratchet.

---

## 10. INSTRUMENTS, REJOUABLES

```bash
export PATH=/c/msys64/ucrt64/bin:$PATH
export GOCACHE=<worktree>/.gocache
export CGO_ENABLED=1

# 1. L executable : calibration (6 temoins) + les 12 cibles, avec bornes et grammaire partielle
REAP_EXE="D:/SteamLibrary/steamapps/common/Halo Infinite/HaloInfinite.exe" \
  go run -tags=research ./internal/games/halo_infinite/film/research/cmd_reapparition

# 2. L univers des noms de composant, par mot du vocabulaire de la reapparition (le NEGATIF)
REAP_EXE="..." \
  go run -tags=research ./internal/games/halo_infinite/film/research/cmd_reapparition -inventaire

# 3. Le chemin vers le bassin, sur les 7 mini-bobines (AUCUN film du cache)
go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
  -run Reapparition37 -v -timeout 30m

# 4. La lecture COMPLETE d un ecrivain (l instrument ne desassemble pas) — bornes publiees par (1)
objdump -d --no-show-raw-insn --start-address=0x1407ee7b8 --stop-address=0x1407ee87a \
  "D:/SteamLibrary/steamapps/common/Halo Infinite/HaloInfinite.exe"
```

## 11. DECOUVERTES HORS PERIMETRE (consignees, NON traitees)

1. **`ecs_table.tsv` melange deux conventions dans `deser_addr`** : `ti=11 i0` y porte
   `FUN_142edbac8`, qui est le compagnon `+0x28` (serialiseur), quand la vague 3.6 y a mis des
   `+0x40` (deserialiseurs). Aucune grammaire n'est fausse (symetrie du moteur) ; la colonne est
   ambigue. Un lot de table pourrait ajouter une colonne `ser_addr` ou normaliser.
2. **La signature de famille de `NOTE_3_6_METHODE_DESCRIPTEURS` § 2 porte deux transpositions de
   chiffres** (`0x141c8f880` pour `0x1411c8f80`, `0x14049b600` pour `0x1404ab600`) ; le § 17 pt 7
   de `NOTE_3_6_TI12_GRAMMAIRES_A` les avait deja relevees, la note de methode n'a pas ete
   corrigee. Mesure independante ici (§ 7.2).
3. **`ti=29 respawn-block` FERME A 100 %** sur cinq mini-bobines sur sept (11/11, 16/16, 14/14,
   18/18, 10/10). C'est un archetype entierement lisible aujourd'hui, et personne ne le lit.
4. **Le lecteur de minuteur du moteur a un quatrieme champ** : `FUN_142ba78dc` =
   `FUN_140d580d0` + un `R(n)` de plus vers `dest+0x0c`. Les trois composants `ti=0 i5/i6/i7`
   portes par le depot utilisent la forme COURTE (`R(16)+R(16)+R(5)`) ; la forme LONGUE n'apparait
   qu'a l'etiquette 1 du bassin et chez le distributeur. Rien a corriger.
5. **`ti=40 vehicule` : 777 records d'image-cle bornes sur le corpus, marche COMPLETE, fin au
   mauvais bit, zero ferme.** Le decodeur consomme tous les composants declares et atterrit a cote
   : c'est une largeur fausse, pas une couverture manquante. Hors perimetre 3.7.
6. **`VehicleTrack.TEnd` EST RENSEIGNE UNE FOIS SUR CENT** (1/109 et 1/42 sur deux BTB Heavies,
   cuisson de production). Le calque des vehicules publie ses 109 vies, donc le balayage marche ;
   c'est le RATTACHEMENT du dead-state a la vie qui ne se fait pas. La reserve 2 de
   `NOTE_V13_DEADSTATE_VEHICULE` § 5 (« grouper les dead-states consecutifs d'un meme slot en UN
   episode de mort. Non fait. ») designe le meme endroit. C'est le prealable de tout cycle de
   reapparition de vehicule, et il n'appartient pas a un lot de recherche.
7. **LA LECTURE DU BASSIN ECHOUE SUR LES BUILDS BTB HEAVIES** (masques de fente tout-a-un :
   `0xffffffffffffffcf`, `0xb36fffffffffffc6`) et reussit sur les trois films d'arene (prefixes
   contigus `0x3ff`, `0x1fff`, `0x7f`). Cause probable : le NIVEAU de `i14` — sa branche
   `level < 2` lit `R(64)` de plus — ou l'etat par defaut de `ti=2`. A mesurer par build avant
   tout port (§ 9.5 item 2).
8. **`BuildFromFilm` avec le seul `Options.MapQuant` NE PUBLIE AUCUN `flagCarries`**, pas meme sur
   un CTF:Arena du corpus temoin (`DRAPEAUX : aucun` sur `bcb6d393` ET `fb1a1a72`). Le calque
   exige le catalogue versionne d'objectifs de carte. Aucun commentaire d'`Options` ne le dit :
   un instrument de recherche qui cuit un film croit legitimement obtenir le document complet.
9. **L'entite du moteur de jeu est declaree DEUX FOIS au registre** — `ti=0` (27 composants) et
   `ti=2` (18) declarent tous deux le bassin en `i15`, et c'est `ti=2` qui vit (19 a 53 records
   par film) pendant que `ti=0` n'a qu'un record a `n2 = 0`. Choisir le premier archetype qui
   declare un composant est donc un piege : la premiere passe de ce lot a rendu « 0 record
   marche » et aurait laisse croire que le bassin n'est pas dans le film.
