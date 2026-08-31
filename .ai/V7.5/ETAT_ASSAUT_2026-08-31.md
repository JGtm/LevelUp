# ETAT DE L'ART — MODE ASSAUT (2026-08-31)

> Cloture du lot A ouvert le 2026-08-27 (`PLAN_ASSAUT_LOT_A_2026-08-27.md`), qui s'etait arrete
> au DIAGNOSTIC. Ce document dit ce qui est PUBLIE, ce qui est REFUTE, et ce qui reste — avec les
> chiffres et les pieces. Branche : `wt/assaut-bombe`.

## 0. En une phrase

L'Assaut publie desormais ses explosions de bombe, datees a la milliseconde et attribuees a un
joueur : **26 explosions sur les 28 datees du corpus de 9 films (92,9 %), 0 publiee hors releve**.
Les deux manquantes n'ont AUCUN slot de joueur porteur dans le film — le point n'y existe que sur
le slot d'equipe.

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

### 1.d L'API ne donne RIEN pour l'Assaut — negatif DEFINITIF

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

1. **Le SON de l'explosion** — l'evenement existe desormais, les sons sont extraits et rendus ; il
   ne manque que la DESIGNATION du stem par l'utilisateur. Une ligne dans
   `objectiveSound.OBJECTIVE_SOUND_STEMS`. C'est le meilleur rapport effort/rendu du reste.
2. **Les 2 explosions sans porteur** (`df8fcbef` manche 3, `c75f33b8` manche 1). Le point n'existe
   que sur le slot d'EQUIPE — verifie en imprimant TOUS les enregistrements `comp 0` de ces
   manches. Ce n'est pas un filtre : le film ne replique pas le compteur par joueur. Rien a
   corriger sans une autre source.
3. **`BombPlants` separe de `BombDetonations`** — le releve A0.3 le dit : « l'ARMEMENT n'a aucun
   increment propre ». Distinguer l'armement de l'explosion demanderait une chaine OBJET (la bombe
   posee), et A1 a montre que `ti=42` ne la porte pas. Chantier ouvert, sans piste chiffree.
4. **Les sites d'amorcage comme ZONES** — A3 a echoue par l'ancrage `ti=13` ; le catalogue
   d'objectifs ne porte aucune forme de site sur les 5 cartes d'Assaut. La reprise passe par
   `assault_site` / `assault_site_plate` / `assault_bomb_spawn` (hachages craques le 2026-08-30),
   qui n'existent que sur **Isolation, Snowbound, The Pit, High Ground** — les 4 cartes SANS film
   au corpus. Corpus d'abord, code ensuite.
5. **Les 56 emplacements jamais balayes** — `decodeStatComponent` lit puis JETTE jusqu'a 2 valeurs
   conditionnelles par composant ; `StatValue{A, B}` n'en porte que 2 sur 4. Piste transverse, pas
   propre a l'Assaut, et la plus grosse du registre.

## 4. Pieces

- `internal/analysis/replay/assaut_a5_explosions_test.go` — la confrontation de publication.
- `internal/analysis/replay/assaut_manches_research_test.go` — les manches, le controle hors
  echantillon, et les trois sondes de diagnostic.
- `internal/analysis/objectiveevents/statborg_guards_test.go` — les 3 gardes du second critere.
- `replay2d/registre_film/A5_publication_explosions.log` et `A5_controle_hors_echantillon.log`.
- Le corpus : 9 films d'Assaut (`35b75a31` `69b16f5d` `3d58eb37` `34bb3bc8` `1c01e34f` `ce083875`
  `df8fcbef` `c75f33b8` `9f57c612`) + 53 films de controle echantillonnes par mode.

## 5. Avant le gate visuel

**Aucun match d'Assaut n'a d'artefact de rejeu en cache** (0/9 au 2026-08-31). Rien n'apparaitra a
l'ecran tant que les artefacts ne sont pas recuits : `cmd/replay-build` a l'unite, ou
`levelup backfill-replay` pour le parc. Un artefact a ete cuit a la main pour la verification
(`9f57c612`, carte Curfew) — il porte bien ses explosions.
