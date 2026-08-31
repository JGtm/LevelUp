# ETAT DE L'ART — MODE ASSAUT (2026-08-31)

> Cloture du lot A ouvert le 2026-08-27 (`PLAN_ASSAUT_LOT_A_2026-08-27.md`), qui s'etait arrete
> au DIAGNOSTIC. Ce document dit ce qui est PUBLIE, ce qui est REFUTE, et ce qui reste — avec les
> chiffres et les pieces. Branche : `wt/assaut-bombe`.

## 0. En une phrase

L'Assaut publie desormais ses explosions de bombe, datees a la milliseconde et attribuees a un
joueur. DEUX CHIFFRES, et il faut les deux : **26 explosions sur les 28 datees sont attribuees a
un SLOT** (92,9 %, 0 publiee hors releve), et **21 sur 28 arrivent A L'ECRAN** (75 %) une fois le
pont d'identite passe. Avant ce lot il y en avait **3**.

Les ecarts sont mesures et expliques : 2 explosions n'ont AUCUN slot de joueur porteur dans le
film (le point n'existe que sur le slot d'equipe), et 5 de plus tombent au pont d'identite par
manche — toutes sur les 3 films One Bomb, les seuls multi-manches, ou une manche courte n'offre
pas assez de progressions du compteur de morts pour nommer son slot.

## 1. Ce qui a ete PUBLIE

### 1.a `ObjectiveTypeBomb` — le poseur, enfin sorti du diagnostic

`comp 0 A` des slots de JOUEUR porte le point de mode, et en Assaut **un point de mode vaut UNE
EXPLOSION** — rien d'autre ne fait bouger le score (releve A0.3 §2 constat 1, fige le 2026-08-27).
La phase A4 avait etabli la replication sur MOITIES DISJOINTES (4/4 recherche, 4/4 verification,
controle de lecture 37/37 sur les morts) mais n'avait rien publie.

Trois lignes suffisaient une fois la troncature levee (§1.b) :

| ou | quoi |
|---|---|
| `objectiveevents/extract.go` | `ObjectiveTypeBomb` + la branche `assault`/`bomb` de `classifyObjectiveMode` |
| `objectiveevents/named.go` | `StatBombDetonations` + `namedStatSlots[bomb] = {0 A}` |
| `.ai/refs/TABLE_STATS_STATBORG.tsv` | la ligne `bomb 0 A bomb_detonations` (le test de concordance l'exige) |

Le reste de la chaine EXISTAIT DEJA : `replaybuild/matchfacts.go` appelle
`NamedEventsFrom(recs, ObjectiveTypeOf(variant))`, le pont d'identite par manche nomme le slot, et
`buildObjectiveActions` pose l'action sur la grille de frames. Aucune plomberie neuve.

**Cote web** : un sixieme genre de marque de fiche, `bomb` — un INSTANT tenu 2,5 s, comme la prise
de base, parce que rien n'attribue le PORT de la bombe. Glyphe (corps rond + meche), libelles FR
« Vient de faire sauter la bombe » / EN « Just detonated the bomb », trois tests.

### 1.b `RealRounds` — la troncature de One Bomb, levee

**LE BLOCAGE ETAIT LA, et il fallait le lever AVANT de publier**, pas apres. Une manche d'Assaut
One Bomb porte au plus UNE emission de score, donc sa plus longue suite strictement croissante vaut
2 — sous le seuil de 3 de `statMinRoundRun`. Consequence mesuree : sur `df8fcbef`, `c75f33b8` et
`9f57c612`, seule la manche 0 survivait et **8 explosions sur 11 etaient perdues**. Publier dans cet
etat aurait montre 1 explosion sur 4.

Second critere d'admission, en OU avec le premier : une manche est **MATERIELLE** si elle porte au
moins **25 enregistrements de slot joueur** ET au moins **10 %** de ceux de la manche la plus
fournie du film. Les deux conditions ensemble, donc le critere ne peut qu'AJOUTER des manches.

| population | part des enregistrements de la manche la plus fournie |
|---|---|
| ancrage fortuit | **<= 5,84 %** (`bfcd1175` manche 6 : 18/308 — un film de Slayer, mode sans manche) |
| manche reelle | **>= 21 %** (`df8fcbef` manche 1 : 45/212) |

Dix pour cent se pose au milieu de ce vide. **Controle HORS ECHANTILLON** : 53 films echantillonnes
PAR MODE (Slayer, Fiesta, BTB, Husky Raid, Firefight, Oddball, CTF, KOTH, Strongholds,
communautaires), 180 manches brutes, **aucune dans la bande interdite 7 %..15 %**.

Les deux contre-exemples historiques TIENNENT : `53ce4390` (score d'equipe 1 -> 2 104) et
`1bc77d2e` (`flag_capture_assists` 1 -> 1 569) rendent toujours la seule manche 0.

### 1.c La garde de domaine, etendue au canal frere

Le premier essai de publication a sorti **66 explosions au meme instant** sur `ce083875` : un
enregistrement `comp 0` a `A=66, B=16635` que `incrementTimes` transformait en 66 evenements. Son A
passait la borne `statMaxModeScore` ; sa seule marque distinctive etait le B.

Mesure sur 65 films (3 986 enregistrements joueur porteurs du composant 0) : **le canal B vaut ZERO
dans 98,3 % des cas**, et les 4 enregistrements a B hors domaine du corpus d'Assaut sont EXACTEMENT
les 4 aberrations connues, les deux contre-exemples historiques compris. Les deux canaux d'un
composant sortent de la MEME emission : `modeScoreInDomain` les borne tous les deux. Un
enregistrement legitime a `B=1` (`ce083875` a 947537 ms, une vraie explosion) survit — exiger `B=0`
l'aurait jete.

### 1.d LA DEFLAGRATION SUR LA CARTE — l'effet voyant (retour du 2026-08-31)

Le filigrane de fiche (22 % d'opacite, derriere le contenu, a l'autre bout de l'ecran par
rapport a la carte) ne suffisait pas : « faudrait un truc bien voyant quand meme ». L'explosion
se peint desormais SUR LA CARTE, a l'endroit et a l'instant ou elle a eu lieu, en trois couches
(`bombBlastFx.ts`) :

| couche | ce qu'elle fait |
|---|---|
| ECLAT | disque plein, large d'emblee, eteint au premier QUART de la vie |
| ONDE | anneau epais qui s'ouvre a 44 px — presque DEUX FOIS l'onde d'une capture (24 px) |
| ECLATS | huit traits radiaux ; ils meurent au carre du temps, pour que la fin soit un anneau seul |

Tenue **12 images (1,2 s)**, deux fois celle d'une capture : une capture arrive plusieurs fois
par manche, une explosion d'Assaut une a quatre fois par MATCH — un evenement rare a le droit de
tenir l'ecran. Encre = celle du CAMP de l'auteur, comme l'onde de capture : sur la carte le
glyphe dit QUOI, la couleur dit QUI. Sous « mouvement reduit », une empreinte FIXE et pleine qui
ne fait que palir — l'information reste, seul le mouvement part.

Aucune garde de mode : la garde EST la donnee (seul un match d'Assaut publie la stat). Le
cablage (`useReplayBombBlast`) a ete PAYE par une extraction, comme le cliquet de taille du
canvas l'exige — la fin de vol des grenades part dans `useReplayGrenadeRest`, et le plafond
descend de 678 a 665.

### 1.e Le PONT D'IDENTITE, la derniere marche — 21 sur 28 a l'ecran

Le nommage attribue une explosion a un SLOT d'entite statborg ; le rejeu, lui, joint sur le XUID.
Une explosion dont le pont ne sait pas nommer le slot POUR SA MANCHE n'est pas publiable — la
rattacher au slot d'une autre manche serait exactement l'erreur que le pont par manche existe pour
eviter (le slot est REATTRIBUE d'une manche a l'autre).

| film | nommees | publiees |
|---|---:|---:|
| `1c01e34f` `34bb3bc8` `35b75a31` `3d58eb37` `69b16f5d` `ce083875` | 17 | **17** |
| `9f57c612` | 4 | 2 |
| `df8fcbef` | 3 | 2 |
| `c75f33b8` | 2 | 0 |

**Les 5 pertes sont TOUTES sur les 3 films One Bomb**, les seuls multi-manches : le pont y resout
chaque manche separement, sur les seules progressions du compteur de morts DE CETTE MANCHE, et une
manche courte n'en offre pas assez. Ce n'est PAS une regression de ce lot — avant lui, `RealRounds`
ne retenait qu'une manche par film One Bomb, donc une seule explosion publiable par film : **3 au
total contre 21 aujourd'hui**. Le chiffre est fige par `TestAssautA5PontIdentite` pour qu'une
amelioration du pont se voie, et une degradation aussi.

### 1.f L'API ne donne RIEN pour l'Assaut — negatif DEFINITIF

La reprise supposait qu'il fallait re-interroger l'API sous le bon nom de famille (`BombStats` et
non `AssaultStats`). **C'est refute par la mesure** : dump du payload `GetMatchStats` brut de
3 matchs couvrant les 3 variantes (`Assault:One Bomb`, `Assault:Neutral Bomb`,
`Husky Raid:Assault`) le 2026-08-31 —

- le bundle `Stats` ne porte que `CoreStats` et `PvpStats`, au niveau JOUEUR comme au niveau EQUIPE ;
- la chaine « bomb » et la chaine « assault » n'apparaissent **nulle part** dans les 3 payloads.

Le binaire, lui, DECLARE bien la famille : 9 statistiques `Bomb` aux adresses `14381e790`..
`14381f1b8` (`BombDetonations`, `BombDefusals`, `BombPlants`, `BombCarriersKilled`,
`BombDefusersKilled`, `BombPickUps`, `BombReturns`, `KillsAsBombCarrier`, `TimeAsBombCarrier`).
Le moteur les calcule ; le service ne les expose pas. **Le film est la seule source, et il n'en
replique qu'une.**

## 2. Ce qui est REFUTE, et qu'il ne faut pas rouvrir

| piste | verdict | piece |
|---|---|---|
| API `BombStats` / `AssaultStats` | **REFUTE** — le payload ne porte que `CoreStats`+`PvpStats` | dumps du 2026-08-31, 3 variantes |
| Sites d'amorcage par ancrage `ti=13` | **NON TENU** — chainage 1,9-16,4 % contre 87-99 % en KOTH, indistinguable du temoin decale de 3 bits sur 7/8 films | commit `c8a107339` (A3) |
| Identite de la bombe par `ti=42` | **RATE** — 0 candidat sur 7/7 films mesures | commit `cb3887215` (A1) |
| Les 27 objets « C2 » = sites d'Assaut | **REFUTE** — ce sont des emplacements de crane d'Oddball (`minigame_megalo_object_2/3`) | craquage de hachages du 2026-08-30 |
| Bande de mode (comps 20-27) en Assaut | **VIDE** — aucune des 8 autres stats `Bomb` n'y est repliquee | balayage A4, 43 emplacements |

## 3. Ce qui RESTE, par ordre de rendement

1. **Le SON de l'explosion** — LIVRE, voir §3.1.
2. **Le PONT D'IDENTITE sur les films multi-manches** — 5 explosions sur 28 y tombent (`c75f33b8` en perd ses 2). `SlotIdentityByRound` resout chaque manche sur ses seules progressions de morts ; une manche de One Bomb en offre peu. Piste : elargir la source d'appariement au-dela du compteur de morts, ou accepter un appariement inter-manches SOUS PREUVE. Chantier a chiffrer, transverse (il sert aussi Oddball multi-manches).
3. **Les 2 explosions sans porteur** (`df8fcbef` manche 3, `c75f33b8` manche 1). Le point n'existe
   que sur le slot d'EQUIPE — verifie en imprimant TOUS les enregistrements `comp 0` de ces
   manches. Ce n'est pas un filtre : le film ne replique pas le compteur par joueur. Rien a
   corriger sans une autre source.
4. **L'ARMEMENT de la bombe** — voir §3.3 et §3.4. Les TROIS maisons de la donnee d'objectif sont
   cartographiees : les deux LISIBLES sont fermees par la mesure, et la troisieme (, celle
   des JAUGES) est ILLISIBLE en Assaut — chainage 1,9-16,4 %. Reparer son ancrage est le chemin
   critique.
5. **Les sites d'amorcage comme ZONES** — A3 a echoue par l'ancrage `ti=13` ; le catalogue
   d'objectifs ne porte aucune forme de site sur les 5 cartes d'Assaut. La reprise passe par
   `assault_site` / `assault_site_plate` / `assault_bomb_spawn` (hachages craques le 2026-08-30),
   qui n'existent que sur **Isolation, Snowbound, The Pit, High Ground** — les 4 cartes SANS film
   au corpus. Corpus d'abord, code ensuite.
6. **Les 56 emplacements jamais balayes** — OUVERTS le 2026-08-31 : `StatValue` porte desormais
   `C`, `D`, `HasC`, `HasD`, et le vecteur dense fige en montre deux. Le balayage d'Assaut n'y a
   rien trouve (§3.3), mais ils restent a instruire pour les AUTRES modes — c'est la plus grosse
   piste transverse du registre.

### 3.1 Le SON de l'explosion — LIVRE (2026-08-31)

Designe a l'oreille par l'utilisateur sur la planche d'ecoute, et corrobore INDEPENDAMMENT par la
structure de la banque : l'evenement `984f65e5` (`play_004_mod_mp_assault_bomb_detonated`) declare
« 1 couche, 1 son » et pointe `538469998`. Deux chemins, une seule reponse.

    static/sounds/halo_infinite/objective_bomb_detonated.wav    4,41 s, stereo 48 kHz, -1,0 dBTP
    OBJECTIVE_SOUND_STEMS   bomb_detonations: { any: 'objective_bomb_detonated' }

`{ any }` et pas une paire : la banque ne porte qu'UN son de detonation, sans jumeau
`_team`/`_enemy` — comme le retour de drapeau et la contestation de zone.

**Piege de rendu paye** : `-ac 2` applique une loi de panoramique de -3 dB en silence (crete
mesuree a -4 dB au lieu de -1). Le passage mono -> stereo se fait par `pan=stereo|c0=c0|c1=c0`,
puis la crete se pose en DEUX passes sur un intermediaire flottant.

### 3.2 La banque, telle que sa structure la donne

42 evenements, **24 gestes distincts** (le jeu declare chacun deux fois, une par variante de
mode). Les 9 nommes par la RE du 26/08 sont confirmes media par media. Deux gestes COMPOSES —
`0c1f744d` et `db750736` — sont les seuls a empiler une boucle bout-a-bout et une queue tiree
parmi six : la forme d'un COMPTE A REBOURS. Les sept autres evenements nommes (`bomb_taken`,
`bomb_pickup`, `bomb_spawn`, `bomb_despawn`, `bomb_disarm_loop`) n'ont AUCUN evenement
correspondant cote rejeu — le film ne replique que la detonation.

### 3.3 L'ARMEMENT — negatif mesure sur la branche « compteur », indecis sur la branche « minuterie »

Hypothese de l'utilisateur (2026-08-31) : « pour l'armement a mon avis ca doit etre dans le
statborg ». Elle meritait la mesure, parce que le releve A0.3 n'avait regarde que DEUX canaux.

**Les deux canaux jamais lus sont ouverts.** Le composant porte quatre valeurs : A et B
inconditionnelles, puis deux drapeaux commandant chacun une valeur. `decodeStatComponent` lisait
ces deux dernieres pour avancer le curseur et les JETAIT — **56 emplacements que rien n'avait
jamais regardes**. `StatValue` porte desormais `C`, `D`, `HasC`, `HasD`.

**Balayage des 112 canaux** (28 composants x 4) sur 9 films contre 28 explosions. Critere ecrit
avant la mesure : une progression avant CHAQUE explosion, dispersion des delais <= 20 % de la
mediane. **Aucun candidat** — les meilleures couvertures sont les compteurs ordinaires, dispersion
116 % au mieux, six fois le seuil.

**La branche minuterie est INDECISE, et c'est le temoin qui le dit.** Une sonde de decroissance
lineaire rend des correlations elevees partout ; le temoin, rejoue sur des instants SANS explosion
(-180 s), rend **48,9 % contre 51,4 %, rapport 1,05**. La sonde ne distingue rien. Le negatif se
lit « la mesure ne sait pas repondre », jamais « ce n'est pas la ».

**Reste a instruire** : les slots d'EQUIPE en progression, les composants au-dela de 27
(l'archetype en compte 58), et la voie hors statborg — la MECHE comme constante moteur, a lire
dans le pool Lua de la ParcelLibrary (meme technique que les constantes du drapeau CTF), d'ou
`armement = explosion - meche`.

### 3.4 LES TROIS MAISONS de la donnee d'objectif — la carte qui manquait

L'intuition de l'utilisateur (« ce serait etrange d'avoir ca a un autre endroit ») designe le bon
voisinage. La carte exacte, etablie le 2026-08-31 :

| maison | ce qu'elle porte | lecteur |
|---|---|---|
| **1. Composants du statborg** | les COMPTEURS par joueur — `flag_captures`, `zone_captures`, `vip_selected`, `bomb_detonations` | `objectiveevents/named.go` |
| **2. Pied de film, `th=10`** | les INTERACTIONS d'objectif — prises de zone, prises de colline, possession du crane | `extractFromTh10` |
| **3. Archetype `ti=13`** | les PROPRIETES d'objet gere — **la JAUGE de capture**, le proprietaire d'une zone, la colline active | `filmdec.ScanFilmManagedProperties` |

**Les jauges et minuteurs vivent dans la TROISIEME**, pas dans les composants. C'est la source de
la confusion : la phase A6 cherchait dans la premiere.

**Maison 2 — jamais ouverte pour l'Assaut, et desormais balayee.** `classifyObjectiveMode`
rendait `""`, donc le pied de ces neuf films n'avait jamais ete lu. Sonde ecrite SANS le filtre
`th == 10` de `decodeTh10Block`. Resultat : **`th=10` est quasi absent en Assaut — 6 blocs sur
9 films**. Les indices dominants sont `th=20` (1 036), `th=50` (1 232), `th=100` (188),
`th=150` (25) ; aucun ne tient le critere (les deux gros couvrent 28/28 avec 334 % et 768 % de
dispersion, ils precedent trivialement n'importe quoi).

**Maison 3 — PAS un negatif, un canal ILLISIBLE.** Premiere passe en ne gardant que les lectures
CHAINEES : **zero progression**, parce que le chainage vaut **1,9 a 16,4 %** contre **87 a 99 %
sur un KOTH de reference** (la contamination d'ancrage de la phase A3). Passe relachee : 7 couples
(slot, tag) portent une progression, chacun couvrant 1 explosion sur 28. Les 8 slots `ti=13` sont
bien la a chaque film — c'est leur ANCRAGE qui est casse.

**Consequence, et c'est le chemin critique** : reparer l'ancrage de `ti=13` sur les cartes
d'Assaut est le PREALABLE de l'armement. C'est la maison ou vivent les jauges, donc celle ou
l'armement est le plus probablement — et la seule des trois qui ne soit pas fermee.

Reserve a garder : `1c01e34f` et `34bb3bc8` journalisent une **empreinte de registre ECS
INCONNUE** (build de jeu different du corpus de calibration).

### 3.5 `ti=13` : le diagnostic A3 est un ARTEFACT DE DENSITE — et le canal reste vide en Assaut

**LA CORRECTION D'ABORD, parce qu'elle vaut au-dela de l'Assaut.** La phase A3 concluait a une
« contamination d'ancrage » sur la foi d'un chainage de 1,9 a 16,4 % contre 87-99 % en KOTH.
C'EST UNE MAUVAISE LECTURE DE L'INSTRUMENT. `Chained` est FAUX PAR CONSTRUCTION pour le dernier
record d'un paquet — rien ne peut le suivre. Mesure du 2026-08-31, densite de records `ti=13`
par paquet delta contre chainage, TOUS MODES CONFONDUS :

| film | mode | densite (rec/paquet) | chainage |
|---|---|---:|---:|
| `c75f33b8` | Assaut | 0,010 | 1,9 % |
| `9f57c612` | Assaut | 0,011 | 2,4 % |
| `35b75a31` | Assaut | 0,020 | 3,0 % |
| `1c01e34f` | Assaut | 0,037 | 4,7 % |
| `0a247154` | Ranked:KOTH | 0,140 | 29,8 % |
| `2ce58582` | Ranked:Strongholds | 0,258 | 56,1 % |
| `21ece4d8` | KOTH:Arena | 0,408 | 46,5 % |
| `7f1bbf06` | KOTH:Arena | 0,567 | 47,6 % |
| `696a9d7c` | Strongholds:Arena | 1,070 | 44,1 % |

**Le chainage suit la densite, monotone, sur tous les modes — l'Assaut est SUR la courbe, pas a
cote.** Il mesure combien de records se suivent dans un paquet, pas si la lecture est juste. Le
taux de MARCHE, lui, est le vrai temoin de justesse : 23-49 % en Assaut, contre 23-81 % chez les
temoins. **La lecture de `ti=13` en Assaut n'a jamais ete cassee.**

(Cas a part : `cde26226` (CTF) a une densite de 7,3 pour 2,6 % de chainage — sa bande fait
**914 slots** contre 8 a 52 ailleurs. La, c'est une vraie sur-inclusion de bande.)

**MAIS LE CANAL EST VIDE DE JAUGE EN ASSAUT.** Le contenu, dumpe pour la premiere fois : 8 slots,
56 couples (slot, tag) porteurs de valeur sur `9f57c612`, la plupart avec **1 a 6 lectures**, des
valeurs a l'echelle du milliard, et presque exclusivement le canal PAR JOUEUR (i2..i33). Rien qui
ressemble a une jauge — ni rampe, ni plage bornee, ni cadence.

**Conclusion des trois maisons : le film ne replique PAS l'armement de la bombe.** Reste la voie
hors film — la MECHE comme constante moteur, a lire dans le pool Lua de la ParcelLibrary (meme
technique que les constantes du drapeau CTF), d'ou `armement = explosion - meche`.

## 4. Pieces

- `internal/analysis/replay/assaut_a5_explosions_test.go` — la confrontation de publication.
- `internal/analysis/replay/assaut_manches_research_test.go` — les manches, le controle hors
  echantillon, et les trois sondes de diagnostic.
- `internal/analysis/objectiveevents/statborg_guards_test.go` — les 3 gardes du second critere.
- `replay2d/registre_film/A5_publication_explosions.log` et `A5_controle_hors_echantillon.log`.
- Le corpus : 9 films d'Assaut (`35b75a31` `69b16f5d` `3d58eb37` `34bb3bc8` `1c01e34f` `ce083875`
  `df8fcbef` `c75f33b8` `9f57c612`) + 53 films de controle echantillonnes par mode.

## 5. Etat des artefacts, et la consigne qui l'accompagne

**4 artefacts sur 9 sont recuits** : `9f57c612` (Curfew), `69b16f5d` (Origin), `3d58eb37`
(Absolution), `c75f33b8` (Curfew). Les 5 autres ne le sont PAS — `df8fcbef` (Curfew),
`ce083875` (Origin), `1c01e34f` (Urban Raid), `35b75a31` (Origin), `34bb3bc8` (Rat's Nest).

**LA CUISSON EN LOT SE DEMANDE AVANT DE SE FAIRE** (consigne utilisateur du 2026-08-31, apres la
saturation de sa machine). `cmd/replay-build` porte desormais ses trois protections — verrou
d'exclusion `filmproc.AcquireSolo`, priorite CPU basse, sentinelle memoire a 2 Gio — mais elles
protegent la MACHINE, pas la DECISION : recuire un parc reste un acte qui se demande.

## 6. La bombe RAM du 2026-08-31, et ce qui a ete pose contre

Les 4 premiers artefacts ont ete cuits par une boucle de shell autour de `cmd/replay-build`,
puis une SECONDE boucle en arriere-plan pendant que la premiere tournait. Plusieurs decodages en
vol, aucun plafond : la machine de travail de l'utilisateur a suffoque. QUATRIEME sinistre du
meme genre (2026-08-20, 08-24, 08-26, 08-31).

Les trois protections du depot (`internal/filmproc`) et son ratchet
(`archlint.TestNoUnboundedFilmLoop`) gardent toutes l'INTERIEUR du processus.
`cmd/replay-build` etait declare « CLI unitaire : un film par invocation » — vrai, et sans effet
contre celui qui LANCE les invocations. **Une garantie qui s'arrete a la frontiere du processus
ne dit rien du nombre de processus.**

Pose en reparation, dans ce meme lot :

| quoi | ou |
|---|---|
| Verrou « un seul decodage a la fois sur la machine », battement de coeur + reprise auto d'un verrou perime | `internal/filmproc/solo.go` (+ 4 tests) |
| Priorite CPU basse du processus COURANT (il n'existait que pour un enfant a lancer) | `internal/filmproc/selfpriority_*.go` |
| Les trois protections armees dans le CLI, AVANT tout decodage | `cmd/replay-build/main.go` |
| Ratchet : tout point d'entree `cmd/` de l'allowlist DOIT armer une sentinelle | `archlint.TestPointsDEntreeDeDecodageArmentUneSentinelle` |
| Sentinelle armee dans les 8 instruments de recherche de ce lot (pic mesure : 0,01-0,02 Gio) | `assaut_manches_research_test.go`, `assaut_a5_explosions_test.go` |

Verification sur pieces : un verrou temoin pose a la main fait refuser le CLI en **0 decodage**,
code de sortie 12, message nommant le detenteur.
