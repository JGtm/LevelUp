# ETAT — ASSAUT : LES EVENEMENTS DE MODE VUS DU BINAIRE (2026-09-04)

> Mission STATIQUE. Zero decodage de film, zero base, zero reseau externe. Aucun fichier de
> production touche. Question posee : existe-t-il, pour l'AMORCAGE et le DESAMORCAGE, un canal
> moteur plus precis que ce qu'on lit aujourd'hui ?
>
> Reponse courte : **OUI pour le vocabulaire, PAS ENCORE pour le canal**. Le moteur NOMME
> l'amorcage et le desamorcage comme des quantites de premiere classe, et il pousse l'etat de la
> bombe ET sa jauge jusqu'au HUD du client. La recherche precedente les a cherches sous un nom
> que le moteur n'utilise jamais. Detail en §3 et §6.

## 0. Sources, et leur rang

| source | rang | acces |
|---|---|---|
| `HaloInfinite.exe` sous Ghidra | **PREMIERE** | le pont MCP est CASSE (voir §0.1) ; le plugin repond en direct sur `127.0.0.1:8089` |
| Pool Lua des tags `hsc*` (dumps hors depot) | PREMIERE (chaines en clair) | `Downloads\Halo Infinite - Sons v75\_donnees\lua\` |
| Depot (`.ai/`, tests, logs de registre) | SECONDE (retro-ingenierie deja faite) | lecture directe |

### 0.1 Le pont MCP `ghidra` est inutilisable — le plugin, lui, repond

`mcp__ghidra__list_instances` voit bien l'instance (`pid 34652`, socket UDS) mais elle
n'annonce **aucun nom de projet** (`"available": ["unknown"]`), et `connect_instance` refuse de
s'y attacher pour ne pas viser le mauvais projet. Les deux tentatives ont echoue.

**Contournement, deja documente au depot** (`.ai/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` §19.1,
meme panne cote client Windows) : le plugin GhidraMCP expose une API HTTP locale. Elle repond :

    {"name":"HaloInfinite.exe","image_base":"140000000","function_count":311103,
     "executable_path":"/D:/SteamLibrary/steamapps/common/Halo Infinite/HaloInfinite.exe"}

**Toutes les adresses de cette note sont donc des MESURES de premiere main**, relevees ce jour
sur ce programme, pas des citations du depot.

---

## 1. La famille `Bomb`, exhaustive et telle que le binaire l'ecrit — MESURE

`search_strings("BombStats_")` rend **exactement 9 resultats, total 9**. Le prefixe est
`BombStats_`, comme `CtfStats_` / `StrongholdsStats_`.

| # | nom exact | adresse |
|---:|---|---|
| 0 | `BombStats_BombDetonations` | `14381e790` |
| 1 | `BombStats_BombDefusals` | `14381e8d0` |
| 2 | `BombStats_BombPlants` | `14381ea10` |
| 3 | `BombStats_BombCarriersKilled` | `14381eb58` |
| 4 | `BombStats_BombDefusersKilled` | `14381ec90` |
| 5 | `BombStats_BombPickUps` | `14381edc0` |
| 6 | `BombStats_BombReturns` | `14381ef28` |
| 7 | `BombStats_KillsAsBombCarrier` | `14381f078` |
| 8 | `BombStats_TimeAsBombCarrier` | `14381f1b8` |

L'affirmation de `objectiveevents/named.go` (« les huit autres statistiques de la famille
`Bomb` ») est **VERIFIEE et exacte** : 9 stats, 1 exploitee, 8 non repliquees. La table figee du
depot (`.ai/refs/TABLE_STATS_BINAIRE.tsv`) est conforme au binaire, au prefixe pres qu'elle
n'ecrit pas.

Un second jeu de chaines porte les memes noms SANS prefixe (`143ba8758`..`143ba8908` :
`BombCarriersKilled`, `BombDefusals`, ...) — les noms de champ, distincts des cles de schema.

## 2. Ou vivent ces 9 compteurs — MESURE, et c'est une reponse fermee

**Ils ne vivent dans AUCUN conteneur replique. Ce sont des champs de TELEMETRIE BOND.**

La preuve tient en une chaine, a `14373a2a0` :

    Microsoft.Halo.HaloStats.Bond.HaloInfinite.Match.Stats.BombStats

et en une seconde, `BombStats` a `14373a290` — le nom court du schema.

Le chemin d'ecriture, releve bout en bout :

| etape | adresse | ce qui s'y passe |
|---|---|---|
| accesseurs de chaine | `FUN_14004ed10` / `ed90` / `ee10` (un par nom) | rendent le pointeur de chaine |
| table de pointeurs `.rdata` | `1443d0de0` / `0e38` / `0e88` | `PTR_s_BombStats_*` |
| **fabrique d'identifiants** | **`FUN_1403a7370`** | hache les 3 noms et pose les ids dans `_DAT_1451a2890/94/98` |
| **consommateur unique** | **`FUN_1434771b8`** | serialiseur d'evenement de telemetrie : longue cascade « si ce sous-bloc est non nul, emets ces champs sous ces ids » |

`FUN_1403a7370` ne hache que **trois** des neuf (`BombDetonations`, `BombDefusals`,
`BombPlants`) ; les six autres n'ont pas d'identifiant runtime dans cette fonction.

**Consequence, et elle est structurelle** : un champ Bond est un payload envoye au service de
stats de Microsoft. Ce n'est pas un composant d'entite, pas une propriete d'objet gere, pas une
ligne de pied de film. **Le film ne peut pas les repliquer, par construction** — et l'API 343
choisit de ne pas exposer le bundle `BombStats` (negatif deja mesure le 2026-08-31 sur les
3 variantes). Les deux negatifs ont donc la meme cause unique.

Ce que le film porte aujourd'hui (`bomb_detonations`, statborg `comp 0` canal A des slots
joueurs) n'est PAS `BombStats_BombDetonations` : c'est le compteur GENERIQUE de points de mode,
qui en Assaut vaut numeriquement les explosions. Coincidence de valeur, pas de nature.

**Statut : MESURE.** Sources : `14373a2a0`, `FUN_1403a7370`, `FUN_1434771b8`.

## 3. `BombObjectState` — qui l'ecrit, qui le lit, et OU IL ARRIVE

### 3.a L'enumeration, confirmee au chiffre — MESURE

`FUN_14034a0d0` enregistre les cinq membres avec leurs valeurs, en clair :

    _DAT_144d6f5f4 = hash("BombObjectState_Unarmed")    ; valeur 1 ; nom court "Unarmed"
    _DAT_144d6f604 = hash("BombObjectState_Armed")      ; valeur 2 ; nom court "Armed"
    _DAT_144d6f614 = hash("BombObjectState_Disarming")  ; valeur 3 ; nom court "Disarming"
    _DAT_144d6f624 = hash("BombObjectState_Contested")  ; valeur 4 ; nom court "Contested"
    _DAT_144d6f634 = hash("BombObjectState_None")       ; valeur 0

`FUN_14034a090` enregistre le TYPE lui-meme (`"BombObjectState"`, `143c882e8`, 5 membres). Les
cinq membres ont chacun leur propre initialiseur d'identifiant paresseux
(`FUN_1403199b0` Armed, `FUN_1403199e0` Contested, `FUN_140319a10` Disarming,
`FUN_140319a40` None, `FUN_140319a70` Unarmed), tous appeles depuis une table d'initialiseurs
statiques (`143669678`..`143669698`, pas de 8 octets).

**Il n'y a pas d'etat « Arming » : `Contested` (4) EST l'armement en cours.** Confirme.

### 3.b Est-il REPLIQUE ? OUI — il arrive jusqu'au HUD du client — MESURE

C'est le resultat neuf de cette passe. Le binaire declare une famille de proprietes de
contexte de donnees du HUD, `CenterScoreboard_*` (36 membres releves), et **deux d'entre elles
sont les quantites cherchees** :

| chaine | adresse | accesseur |
|---|---|---|
| **`CenterScoreboard_BombState`** | `14382d300` | `FUN_14004f290` -> `_DAT_14487466c` |
| **`CenterScoreboard_BombNormalizedCaptureProgress`** | `14382d160` | `FUN_14004f250` |

Le HUD d'un client affiche l'etat de la bombe et sa jauge normalisee. **Un client ne peut
afficher que ce qu'on lui replique.** L'etat de la bombe et sa jauge sont donc dans le flux
client — ce que la reserve du negatif `bombstate` disait deja en creux (« il l'est forcement
quelque part : le HUD l'affiche »), et qui est maintenant NOMME.

`CenterScoreboard_*` est une famille **globale au match**, pas par joueur : ces deux proprietes
ne portent donc PAS l'acteur.

Il existe en outre un composant d'entite explicitement nomme :
**`bomb_icon_reader_component`** (`143803848`, accesseur `FUN_14003e390`), l'un des trois seuls
`*_reader_component` du binaire (avec `pip_data_reader_component` et
`target_designator_reader_component`).

### 3.c Dans quel archetype, a quel index ? NON ETABLI

Le binaire donne le NOM du composant, pas sa place dans le registre ECS d'un film. Cette
question se tranche cote film, et **elle se tranche sans decoder quoi que ce soit** : voir §6.

### 3.d Pourquoi le negatif `bombstate` du 2026-09-01 ne ferme PAS cette porte

`filmdec/objectif_bombstate_test.go` a cherche, par EGALITE, `murmur3("bombobjectstate")` =
`0x19813E20` dans `ti=13 i0`, et ne l'a trouve dans aucun des 13 films. Le negatif est net et
il reste vrai — **mais il porte sur une chaine que le moteur ne hache jamais.**

Verifie sur pieces ce jour : les seules chaines que le moteur hache autour de la bombe sont les
CINQ NOMS DE MEMBRE (`BombObjectState_Contested`...), la propriete de HUD
(`CenterScoreboard_BombState`), les trois stats de mode (§4) et `bomb_icon_reader_component`.
Le nom nu du TYPE (`"BombObjectState"`, `143c882e8`) n'est reference qu'une seule fois, par
`FUN_14034a090`, pour enregistrer le type par reflexion — jamais comme cle de propriete.

**Correction methodologique** : j'avais suppose que `FUN_140748a74` (2 arguments) et
`FUN_140748d64` (3 arguments, litteral `4`) etaient deux fabriques distinctes, avec un espace de
noms. **C'est faux, et je l'ai verifie plutot que de le publier** : les deux decompilations ont
le MEME corps — meme normalisation (minuscules, `-` et espace -> `_`, saut de ligne -> `#`),
meme noyau murmur3 (`0x1b873593`, rotation 13), meme repli `FUN_140748fb4` au-dela de
64 octets, ou le `4` reapparait en dur. Ghidra a scinde une fonction unique. **Il n'y a qu'UNE
fabrique de hachage, et l'implementation murmur3 du depot** (`mapvar/hash.go` + la
normalisation, controlee 5/5 sur des libelles connus) **s'y applique telle quelle.**

Le negatif se relit donc ainsi : *bonne fonction de hachage, bon instrument, MAUVAISE CHAINE* —
en plus du canal `ti=13` lui-meme, mesure muet en Assaut.

## 4. Le script Lua : ce qu'il emet a l'amorcage et au desamorcage — MESURE

Pool Lua `common-rtx-new.module`, tag `hsc*` **`25af9c45` = `primitive_carriable_arming_base`**
(12 559 octets), lignes 27267-27460 du vidage
`_donnees\lua\hsc_all_any_globals_common-rtx-new.module.txt`.

**La machine a etats et ses SIX EVENEMENTS, tels que le script les enregistre :**

    modeParcel:AddOnInitializationStarted     -> OnInitializationStarted
    modeParcel:AddOnInitializationInterrupted -> OnInitializationReleased
    modeParcel:AddOnInitializationCompleted   -> OnInitializationCompleted
    modeParcel:AddOnDeactivationStarted       -> OnDeactivationStarted
    modeParcel:AddOnDeactivationInterrupted   -> OnDeactivationReleased
    modeParcel:AddOnDeactivationCompleted     -> OnDeactivationCompleted

    etats : GotoIdle  GotoArming  GotoArmed  GotoDisarming  GotoComplete
    boucle : UpdateThread -> UpdateArming / UpdateArmed / UpdateDisarming
    champs : armDisarmProgress  armDisarmTime  conversionCount
    moteur : Device_GetInteractionHoldTime  deactivationBaseInteractTimeSec
             deactivationConvertsPlantedObject

**INITIALIZATION = amorcage, DEACTIVATION = desamorcage**, et chacun a ses trois temps :
**COMMENCE / INTERROMPU / TERMINE**. C'est exactement le triplet demande, et il est nomme.

**L'evenement porte-t-il l'ACTEUR ? NON — il porte l'EQUIPE.** Dans tout le corps du parcel, les
seuls appels touchant a un acteur sont :

    Item_GetInventoryUnit  ->  Player_GetMultiplayerTeam  ->  Object_SetMultiplayerTeam
    modeParcel.activatingTeam
    modeParcel.normalizedActivationProgress

Le script lit l'unite qui porte l'objet uniquement pour en tirer une EQUIPE, qu'il repose sur
l'objet. Il n'y a ni `activatingPlayer`, ni identifiant de joueur, nulle part dans ce parcel.
**Statut : MESURE pour ce que le script nomme ; DEDUIT pour l'absence** — le pool Lua deduplique
ses chaines et ne donne jamais un negatif (regle etablie au depot le 2026-08-30). Ce qui est sur :
le script n'a besoin que de l'equipe, et c'est l'equipe qu'il propage.

**La jauge est CALCULEE, pas transmise telle quelle** : `armProgressFunction` /
`disarmProgressFunction` sont poses par `Object_SetFunctionValue` — des valeurs de fonction
d'objet, le canal de rendu. C'est coherent avec l'anneau `ti=12 i14` deja exploite.

**La meche n'est pas une constante du script** : `armDisarmTime` est un CHAMP LU, pas un
litteral. Confirme — le pool ne porte aucun nombre a cet endroit.

Le script du MODE, tag **`a35c6ce9`**, declare de son cote **DEUX appareils distincts** :

    Armzone  ArmingTag  PlantedTag
    BombTag  ArmingDeviceTag  DefuseDeviceTag
    AssaultLoopArmTeam  AssaultLoopArmEnemy  AssaultLoopDisarm  AssaultLoopPlanted
    AssaultLoopResetting   (+ les cinq variantes BTB)
    armzoneArgs  defender_bombsite  attacker_bombsite  bombTag  goalPlate

**`ArmingDeviceTag` et `DefuseDeviceTag` sont deux appareils separes** : amorcer et desamorcer
ne sont pas le meme geste sur le meme objet, ce sont deux tenues sur deux appareils. Et
`AssaultLoopDisarm` est une boucle sonore propre au desamorcage.

## 5. Le DESAMORCAGE : le moteur distingue-t-il abouti et interrompu ? — OUI

**Trois signaux independants le disent, et ils concordent :**

1. **Le script** (§4) : `OnDeactivationCompleted` et `OnDeactivationInterrupted` sont deux
   evenements distincts, enregistres separement. **MESURE.**
2. **L'enumeration** (§3.a) : `Disarming` (3) est un etat a part entiere. Une tenue interrompue
   repasse de `Disarming` a `Armed` ; une tenue aboutie sort de `Disarming` vers `Unarmed` ou
   `None` (`deactivationConvertsPlantedObject` decide laquelle). **MESURE pour les etats ;
   DEDUIT pour les transitions** — je n'ai pas lu la machine de transition cote moteur.
3. **Les appareils** (§4) : `DefuseDeviceTag` est distinct de `ArmingDeviceTag`.

**Cote FILM, l'etat de l'art tient et il ne faut pas le maquiller.** Le corpus de 9 films n'a
**AUCUNE occurrence confirmee de desamorcage** :

- `wt/bombe-portee` : « zero occurrence oracle sur Neutral/Husky (poses completes =
  explosions, 17/17) » — sur ces variantes, toute pose aboutie a explose.
- `B4_bombe_desamorcage.log` (`.ai/V7.5/replay2d/registre_film/`) : les 32 « CANDIDATE
  DESAMORCAGE » viennent **toutes** des deux films One Bomb (`9f57c612`, `c75f33b8`), et
  l'arbitrage de merge les a **rejetees** — fenetre de 10 s contre une meche One Bomb de 16,2 s
  pausable. Ce ne sont pas des desamorcages, ce sont des artefacts de fenetre.
- La seule lecture qui date reellement des tenues de desarmement est indirecte : sur One Bomb,
  `wt/onebomb` lit les PAUSES de la meche (descente < 60 quanta/s, mesuree 14-26 contre 138 pour
  une chute d'explosion) et une descente COMPLETE (fin a 127) comme un desamorcage. **Signal
  derive de la meche, pas un evenement de desamorcage** — et sur One Bomb seulement.

**Le corpus est le facteur limitant autant que le canal** : sur Neutral et Husky, personne n'a
desamorce.

## 6. Ce qui reste ouvert, et ce qui est ferme

| canal | verdict | temoin |
|---|---|---|
| API 343 (`GetMatchStats`) | **FERME** | payloads 3 variantes, 2026-08-31 : `CoreStats`+`PvpStats` seuls, « bomb » absent |
| Statborg, composants 0-27 (112 canaux) | **FERME** | balayage A6, 9 films / 28 explosions : aucun candidat, dispersion 116 % au mieux |
| Pied de film `th=10` | **FERME** | 6 blocs sur 9 films ; `th=20/50` couvrent trivialement (334 % / 768 % de dispersion) |
| Pied — recompenses de score | **FERME** | `assaut_pied_classement_test.go`, negatif net : l'armement n'est pas recompense |
| `ti=13` proprietes nommees | **FERME EN ASSAUT** | canal muet : 8 slots pour 12 a 51 noms distincts = du bruit |
| `ti=11 i12/i13/i14` | **FERME** (voie delta) | `objective_scan.go` : legalite 45,7 % SOUS le hasard ; densite x0,97/x0,94/x1,14, p = 0,61/0,63/0,09 ; « aucun des huit etats de i14 ne ressort ». Consigne au depot : NE PAS re-interroger sans canal nouveau |
| `ti=12 i14` anneau radial | **EXPLOITE** | armement date, Neutral 13/13 et Husky 4/4 a CV 0,016, 0/1000 ; ne donne ni acteur ni lieu |
| Statborg, composants **28-57** | **JAMAIS BALAYE** | l'archetype en compte 58 ; le balayage s'est arrete a 27 |
| Registre ECS du film, par NOM | **JAMAIS INTERROGE avec ces noms** | voir ci-dessous |

**LA PISTE QUI N'A JAMAIS ETE TENTEE, et elle ne coute presque rien.**

Le film porte **son propre dictionnaire de composants, en clair**, dans `chunk_00` : le registre
ECS nomme chaque archetype et chacun de ses composants. C'est lui qui a donne
`managed-objective-progress-component` et `managed-navpoint-radial-progress`. L'instrument
existe et il est ecrit : `replay/assaut_a9_interaction_test.go` (`TestAssautA9Dictionnaire`).

Il a cherche avec la liste de motifs `a9Motifs` — `"interact", "device", "arm", "plant",
"bomb", "hold", "progress", ...`. **`"bomb"` y etait.** Le resultat de ce balayage n'est
journalise nulle part (aucun log `A9_*` dans `.ai/V7.5/replay2d/registre_film/`, aucune entree
au journal), et l'entree du 2026-08-31 ne cite que les composants retenus par `"timer"`,
`"interact"` et `"progress"`. **On ne sait donc pas si `bomb_icon_reader_component` figure au
registre d'un film d'Assaut.** C'est une question a une seule lecture, et la reponse tranche.

Note de methode : `ti=11 i14 state` (R(3), `FUN_142edba10` -> `FUN_1424d121c`) est un enumere de
trois bits, donc capable de porter les cinq valeurs de `BombObjectState`. J'ai cru tenir la un
candidat neuf ; **verification faite, il est deja porte, deja publie et deja refute** par la
mesure de densite ci-dessus. Je l'ecris pour que personne ne le represente comme une piste.

## 7. Recommandation — un seul chemin, et par quoi commencer

**OUI, il existe un canal plus precis, et le moteur le nomme.** L'amorcage et le desamorcage ne
sont pas des quantites que le jeu ignore : il les compte (`prop_stats_mode_assault_bomb_arms`,
`prop_stats_mode_assault_bomb_disarms`), il les distingue a trois temps chacun
(Started / Interrupted / Completed), et il pousse l'etat de la bombe et sa jauge jusqu'au HUD
(`CenterScoreboard_BombState`, `CenterScoreboard_BombNormalizedCaptureProgress`).

Ce qui manque n'est pas le vocabulaire, c'est **l'adresse dans le film**.

**Par quoi attaquer, dans cet ordre — le premier pas coute UNE lecture de `chunk_00` :**

1. **Interroger le registre ECS du film par NOM.** Relancer `TestAssautA9Dictionnaire` sur un
   film d'Assaut en imprimant le dictionnaire ENTIER (pas seulement les composants filtres), et
   y chercher `bomb_icon_reader_component`. Aucun corps de record n'est decode : c'est la lecture
   de `chunk_00` et rien d'autre. Trois issues, toutes utiles :
   - **le nom y est** -> l'archetype et l'index tombent immediatement, et §3.c est resolu ;
   - **le nom n'y est pas, mais un autre composant `bomb*` / `assault*` y est** -> nouvelle cible ;
   - **rien** -> le film ne porte aucun composant de bombe dedie, et il faut passer au 2.
2. **Balayer les composants 28-57 du statborg** avec, comme aiguilles, les trois noms
   `prop_stats_mode_assault_bomb_{arms,detonations,disarms}` haches par l'implementation murmur3
   du depot. Le rendement attendu est reel : `detonations` est le frere des deux autres dans la
   MEME famille de 20, et c'est la seule stat d'Assaut qu'on lit deja. Trente emplacements
   jamais regardes, et le balayage de production s'arrete a 27.
3. **Ne PAS rouvrir** `ti=11` (delta), `ti=13`, le pied de film ni les composants 0-27 : quatre
   negatifs mesures, bornes, avec temoins. Les rejouer avec de nouvelles aiguilles ne changera
   pas un canal muet en canal parlant.

**Et une reserve qui vaut pour tout ce qui precede** : meme si le canal tombe, il donnera
l'INSTANT et l'EQUIPE, pas le JOUEUR. Ni `CenterScoreboard_*` (global au match) ni le parcel Lua
(qui ne manipule que `activatingTeam`) ne portent d'acteur. **L'acteur restera a fermer par le
canal des armes tenues** (`replay/held_object_carry.go`, la bombe = `weap 0x3fee4fcf`), comme il
l'est deja pour la pose — 13/17 en accord direct, 27/28 pour « bombe posee portee par personne ».

**Cote corpus, enfin** : sur Neutral et Husky, aucune pose n'a ete desamorcee (17/17 des poses
completes ont explose). Un canal de desamorcage parfait ne rendrait rien sur ces films. **Toute
validation du desamorcage passe par des films One Bomb, ou par un corpus elargi** — c'est un
prealable de CORPUS, pas de code, et il se demande avant de se faire (regle absolue : aucune
cuisson d'artefacts en lot sans accord).
