# Lot 3.7 — OU LE FILM ECRIT LA REAPPARITION DES OBJECTIFS ET DES VEHICULES (2026-09-17)

> RECHERCHE SEULE. Aucun fichier de production Go modifie, aucun film du cache ouvert.
> Worktree `LevelUp-wt-decfilm-37r`, branche `feat/decfilm-37r`, base `2fac0a4cf`.
> Instruments : `film/research/reapparition/` (executable, LECTURE SEULE) et
> `film/internal/grammar/reapparition_37_bassin_research_test.go` (les sept mini-bobines par
> build, aucun film du cache).
> Ghidra N'ETAIT PAS DISPONIBLE (§ 7) : la chaine du descripteur a ete rejouee en Go sur le PE,
> calibree sur six temoins, et les desassemblages sont d'`objdump` (binutils ucrt64).

---

## 0. LA REPONSE, EN DEUX PHRASES

**OBJECTIFS.** Le film ECRIT les minuteurs d'objectif, et il les ecrit a DEUX endroits distincts,
tous deux identifies sur pieces : (a) le BASSIN du moteur de jeu,
`managed-engine-timers-component` (`ti=0 i15`, en pratique `ti=2 i15` sur les builds du corpus) —
un masque `R(64)` de 64 fentes, puis par fente presente `R(2)` d'etiquette et un enregistrement de
minuteur en SECONDES, le MEME que l'horloge de manche ; c'est lui que designent les index figes de
`ti=11 i0` et de `ti=12 i10` ; (b) le COMPTE A REBOURS MANUEL du navpoint,
`managed-navpoint-manual-timer-initial-duration` / `-current-duration` (`ti=12 i11` / `i12`),
`R(17)` chacun, pas de 50 ms, borne 6 553,55 s. Ni l'un ni l'autre n'est porte ; les deux sont
DECIDABLES HORS LIGNE (aucune dependance de configuration runtime) et leur grammaire complete est
en § 2 et § 3.

**VEHICULES.** Le film N'ECRIT PAS de minuteur de reapparition de vehicule, et le negatif est
MESURE, pas suppose : sur les **294 noms de composant que l'executable embarque**, aucun des **15**
composants `vehicle-*` ne porte de temps sauf `vehicle-emp-timer-component` (la neutralisation
EMP), les **10** composants « respawn » sont TOUS ceux d'un JOUEUR ou d'une nuee IA, et « return »
comme « reset » rendent **ZERO** composant (§ 4). Le delai de reapparition d'un vehicule est une
REGLE DE VARIANTE (`ManagedGameVariant_Get/SetVehicleRespawnTimeOverrideForAll|Channel|Class`,
`...VehicleInitialSpawnDelayOverride...`), pas un etat replique. **La reapparition d'un vehicule se
DEDUIT de l'apparition et de la destruction, toutes deux DEJA DATEES a la milliseconde par le
rejeu** (`VehicleTrack.T0` du record de creation, `VehicleTrack.TEnd` du dead-state `ti=40 i11`) :
c'est le modele `PadCycle`, et il est aujourd'hui applicable alors qu'il ne l'etait pas en
2026-09-01 (§ 5).

Un candidat reste OUVERT et il est nomme : `device-object-dispenser-timer-component` (`ti=43 i37`)
est un VRAI compte a rebours de generateur (deux fentes, porte `R(1)`, `[0, 600] s`, § 4.3). Rien
ne prouve qu'un generateur de vehicule soit une entite `device` — et c'est exactement le piege de
V22 bis. Le test qui tranche est ecrit en § 8, et il demande des films entiers.

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

**Ce n'est donc pas « le minuteur du drapeau ».** C'est l'endroit ou le minuteur du drapeau se
trouve SI le drapeau en a un, et l'index de `ti=11 i0` dit lequel. La jointure index -> fente est
la preuve qui manque, et elle n'est pas faisable aujourd'hui : le bassin n'est pas lisible (§ 6).

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

## 5. CE QUE LE REJEU PEUT DEJA FAIRE DES VEHICULES, ET QUI A CHANGE DEPUIS 2026-09-01

`V2_SPAWNS_COOLDOWNS` a conclu « cooldown NON mesurable » pour une raison precise et datee : la
fin de vie d'un vehicule etait BORNEE par le recensement des images-cles, soit +/-20 s, et un
cooldown de 20-35 s n'en sort pas. **Cette raison est tombee le 2026-09-05** : le dead-state de
`ti=40 i11` est lu, la destruction est DATEE A LA MILLISECONDE, et le rejeu la publie deja
(`VehicleTrack.TEnd`, present pour le seul `End == "destroyed"` — `document_vehicles.go`).

Le cycle de reapparition d'un vehicule est donc mesurable AUJOURD'HUI, par le modele deja ecrit et
deja publie pour les armes : `PadCycle` (`document_ground_weapons.go:111`) — mediane, deciles,
nombre d'ecarts MESURES, nombre de reapparitions dont la disparition precedente n'est pas datee.
La regle de `PadCycle` s'applique telle quelle, y compris sa discipline : **cle ABSENTE quand le
cycle n'est pas ETABLI** (au moins deux ecarts), jamais un chiffre instable publie comme stable.

C'est cela, la reponse produite pour les vehicules : **la reapparition se DEDUIT de l'apparition
et de la destruction, le minuteur n'est pas ecrit** — et la deduction est maintenant assez fine
pour etre honnete.

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

### 8.1 Objectifs — trois etapes, dans cet ordre

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

## 9. CE QUI RESTE INCERTAIN, ET CE QUE JE VEUX LIRE SUR FILM ENTIER

Rien de ce qui suit n'est faisable sur les mini-bobines : elles ne portent que des images-cles
(sauf `000d5950`, un Fiesta sans objectif ni vehicule) et le bassin n'est pas lisible avant le
port de `i11`.

1. **Le bassin porte-t-il le minuteur du drapeau ?** Non prouve. Preuve demandee : un film **CTF**
   entier, lire `ti=11 i0` (deja porte) pour chaque objectif, et les fentes du bassin aux memes
   images-cles apres port de `i11`/`i15`. Attendu : la fente designee par l'index d'un objectif de
   drapeau porte une valeur qui DESCEND entre deux images-cles d'un lacher sans reprise.
2. **Le navpoint du drapeau lache porte-t-il un minuteur manuel ?** Non prouve. Preuve demandee :
   un film **CTF** entier, `ti=12 i11`/`i12` apres port — non nuls seulement pendant les
   intervalles `dropped` que `flagCarries` publie deja. C'est le controle croise le moins cher,
   et il valide ou invalide la voie courte (§ 8.2).
3. **Un generateur de vehicule est-il une entite `device` ?** Non prouve, et c'est le piege a ne
   pas refaire. Preuve demandee : un film **BTB a vehicules** entier, confronter les positions des
   entites `ti=43` (composant `object-position` / `device-position`, deja portes) aux emplacements
   de naissance des vehicules mesures par `V2_SPAWNS_COOLDOWNS` § 1 (rayon 0,00 m). Si un device
   se tient a chaque pad de vehicule, `i37` est le compte a rebours cherche ; sinon le negatif du
   § 4 est complet et definitif.
4. **Le cycle de reapparition des vehicules, maintenant que la destruction est datee.** Mesure
   demandee : 2 a 3 films **BTB / Heavies** entiers, ecart `TEnd -> naissance suivante au meme
   emplacement`, avec les regles de stabilite de `PadCycle` (>= 2 ecarts, `missing` compte). C'est
   la mesure qui remplace le `IQR/mediane 0,87-0,98` de 2026-09-01.

5. **LE CANAL QUE CE LOT N'A PAS ECARTE, ET QU'IL FAUT NOMMER : `ti=13`,
   `managed-object-property`.** Ce n'est pas un composant DE minuteur — c'est un sac de
   proprietes NOMMEES qu'un script de mode attache a un objet (`Engine_CreateFloatNetworkedProperty`,
   `NetworkedProperty_SetFloatProperty` : le NOM est un `StringId` choisi par le script, donc
   il n'apparait PAS dans l'univers des noms de composant du § 4 — le negatif de ce paragraphe
   ne le couvre pas). Trois faits le rendent candidat au retour de drapeau, et aucun ne suffit :
   (a) le tag 3, DOMINANT a 85,7-95,4 % des slots mesures, est un flottant quantifie `R(24)` sur
   **`[-100, +100]`** (`components_managed_property.go`) ; (b) l'echelle de la jauge de retour de
   drapeau vaut **100** dans `parcel_deliver_object.lua` (memoire `reference_ctf_flag_lua_mechanics`,
   avec `flagReturnTimer`, `flagReturnTimerRate`, `onReturnProgress`, `flagResetSeconds`) ;
   (c) le meme canal porte deja la JAUGE DE CAPTURE des zones en production
   (`zone_state_scan.go`, `document_zones.go`). Ce que ce lot NE peut pas faire : `ti=13 i0`
   (`managed-object-property-name-component`, le `StringId` du nom) est MARCHE mais **pas
   recolte** — le balayage l'a retire comme sortie morte a la revue R1 de la phase 2b — et le
   golden de fermeture 0.A.3 mesure que l'ETAT PAR DEFAUT de `ti=13` est DESALIGNE (oracle `n2` :
   `n1` constant a 136, `n2` du bruit), donc sa voie image-cle ne vaut rien ; seule la voie DELTA
   parle, et elle n'existe pas dans les mini-bobines par build. Preuve demandee, sur un **CTF**
   entier : recolter `i0` par la voie delta, hacher les noms Lua candidats (`flagReturnTimer`,
   `flagResetSeconds`, `onReturnProgress`) par le hacheur de `StringId` du jeu, et regarder si un
   slot porte un tag 3 qui monte vers 100 pendant un intervalle `dropped` de `flagCarries`.
   **Ne pas conclure sans ce test** : c'est un canal generique, et un tag 3 qui varie ne dit pas
   de lui-meme qu'il parle du drapeau.

**Films souhaites, cinq au plus, un par un** : 2 CTF (dont un recent), 1 Oddball (le crane a-t-il
un minuteur de remise a zero ?), 2 BTB a vehicules (dont un Heavies). Aucun KOTH ni Strongholds :
les zones ne reapparaissent pas, et le lot 3.6 a mesure qu'un Strongholds ne porte aucun slot
`ti=11` dans ses images-cles.

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
