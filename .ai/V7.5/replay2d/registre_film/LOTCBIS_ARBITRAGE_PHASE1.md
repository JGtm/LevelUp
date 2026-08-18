# Lot C-bis — arbitrage superviseur apres la phase 1 (2026-08-18) : gate 1 non atteint par la lettre, PHASE 2 (replay) ouverte sur la substance

> Lu sur pieces : `LOTCBIS_PHASE1.md` (284 L), commits `4ad72a4a1` (port) et `b7501f3d0` (mesure),
> gates (`EXIT_*=0`), `traverse.go` +6 lignes, 33 lignes de table ECS `porte`, `DesyncAt` en
> progression sur les DOUZE films. Fusionne dans `feat/v75` = `9b959de28`.

## Ce que la phase 1 etablit (acquis)

1. **ti=13 est PORTE en production** (`components_managed_property.go`, deux composants, 16 tags, 2 modes),
   vecteurs reels verts, garde « memes bits sans hook », `DesyncAt` +8 a +128 records par film — le
   portage repare aussi les records qui SUIVENT ti=13 dans le paquet.
2. **Tag 3 = jauge de capture de zone** : 97,2 % / 94,8 % des captures Strongholds precedees d'un sommet
   de rampe a ± 2 s, contre 56,7 % / 60,8 % de hasard (le temoin ecrit — sous la moitie du reel — etait
   inatteignable : le hasard seul le depasse ; defaut de redaction, deuxieme occurrence apres le lot C).
3. **Tag 4 = le premier canal ENUMERABLE du corpus** : 100 % des slots <= 8 valeurs, 5 films, 4 modes ;
   sa semantique (proprietaire ? conteste ?) n'est PAS etablie — la clause temporelle etait vide par
   construction (fenetre couvrant 93-97 % du match).
4. **Tag 5 = cle de nommage des zones** : une emission par slot (par construction, une constante), et la
   carte slot -> identifiant est IDENTIQUE sur les deux Strongholds (1525 `0x67F43AC3`, 1526 `0xD690D6B4`,
   1527 `0xF2F9EB27` : trois slots, trois zones). Le seuil de volume (>= 10 emissions) supposait un canal
   repetitif : mal calibre, pas un canal muet.
5. **KOTH : une seule zone active a la fois, 100 % du temps** (60 rampes, `01e1f945`) — la clause fausse au
   niveau navpoint (lot C) est vraie au niveau de l'objet ti=13. Strongholds : trois zones capturables
   simultanement, la non-unicite est le MODE.
6. Par joueur (mode B) : KOTH tag 7 (flottant) 50-68 % ; Strongholds domine par le tag 0 (vide) — contamination
   deja chiffree en phase 0. CTF : socles non ressortis (negatif ecrit).

## Arbitrage

- (a) est TENU sur son intention (cle stable), le seuil de volume est requalifie (canal constant) ;
- (b) est TENU sur un film et manque de 5,8 points sur l'autre AVEC un temoin impossible : requalifie ;
- (c) reste OUVERT (semantique du tag 4 non etablie, faute de metrique adaptee) ;
- (d) TENU en KOTH.
=> **PHASE 2 ouverte, dans le paquet `replay`** (la ou vivent `AttributeZones`, les faits du match — roster
`team_id` par xuid — et l'assemblage), en DEUX temps : d'abord la MESURE qui manque, ensuite seulement la
publication (`zoneStates`, schema SERIALISE apres l'item 4 de l'utilisateur = 14, donc 15).

Correction de la fausse piste du CR : l'equipe du capteur ne viendra PAS d'un hook sur
`game-engine-team-mapping` (ti=0 n'est PAS replique dans le film — lots B/P) mais du ROSTER fourni par
`port.MatchFacts` (xuid -> `team_id`), deja branche dans `replaybuild` par le lot A.

## Phase 2a — MESURE dans `replay` (instrument sous garde, seuils ecrits ci-dessous, temoins vs HASARD)

- CB.2a.1 **Appariement slot ti=13 -> zone du catalogue** : pour chaque `ZoneCapture` (joueur P a t, `NamedEvents` +
  `SlotIdentity` -> xuid), la zone du catalogue ou P se trouve a t (`AttributeZones` sur la position de P,
  `Attributed` seulement) vs le slot ti=13 dont le tag 3 culmine dans [t-2 s ; t+2 s] : la carte slot -> zone
  doit etre coherente sur tout le match a >= 90 % (denominateur = captures appariees des deux cotes) et
  STABLE entre les deux Strongholds via la cle tag 5 (memes identifiants -> memes zones du catalogue,
  a la symetrie de la carte pres — ecrire la table).
- CB.2a.2 **Semantique du tag 4** (metrique adaptee a un canal bavard) : PRECISION = part des changements de
  tag 4 d'un slot qui tombent dans [t-2 s ; t+2 s] d'une capture/securisation DE CETTE ZONE ; RAPPEL = part
  des captures de la zone suivies d'un changement de tag 4 dans la fenetre ; HASARD = fenetre x densite des
  changements ; seuil : rappel >= 80 % ET precision >= 2x le hasard. Puis la VALEUR : apres une capture par
  un joueur d'equipe T (roster), la valeur du tag 4 est-elle la meme pour toutes les captures de T (et
  differente pour l'autre equipe) sur >= 90 % ? Si oui : tag 4 = PROPRIETAIRE ; sinon chercher « conteste »
  (valeur pendant les rampes du tag 3 non abouties).
- CB.2a.3 **KOTH** : la zone active (unicite 100 %) appariee au catalogue par la grappe des positions des
  marqueurs de score (`th=10`) ou des joueurs dans la rampe ; periodes couvrant >= 80 % du temps de match sur
  >= 2 films (l'objectif du plan item 4 phase 2 — s'il est atteint ici, la phase 2 de l'item 4 devient `[~]`).
- Gate 2a : CB.2a.1 >= 90 % ET (CB.2a.2 proprietaire etabli OU CB.2a.3 tenu) => phase 2b ; sinon negatif ecrit
  par volet et lot clos `[!]` (la jauge de capture reste publiable seule).

## Phase 2b — PUBLICATION (`zoneStates`, schema +1 SERIALISE apres l'item 4)

`zoneStates: [{zoneRef (index de mapObjectives.zones), key (tag 5), spans: [{t0, t1, owner: teamId|null,
state, progress?}]}]`, `coverage.zones` ; contrat/OpenAPI/`generated.ts`/goldens/temoins ; web
`objectivesLayer.ts` : zone teintee par proprietaire, colline active en surbrillance, arc de progression ;
les pulses substituts retires (0 code mort). Gate visuel utilisateur.

Regles : instrument sous garde d'environnement, un film par processus, avant-plan, plafond 3 Go, jamais le
principal, seuils jamais rebaisses, temoins juges contre le hasard PUBLIE.

## Statut de la phase 2a (renseigne par l'executeur, 2026-08-18 — detail : `LOTCBIS_PHASE2A.md`)

- [x] **CB.2a.1** — TENU. Coherence de la carte slot -> zone **93,1 %** (`7344d24f`, 54/58) et
  **98,4 %** (`696a9d7c`, 62/63) pour un seuil de 90 %, temoins a 41,4/47,6 % (permutation) et
  57,1/51,4 % (decalage). **Stabilite inter-films 3/3 = 100 %** (1532 -> zone 1, 1537 -> zone 2,
  1542 -> zone 0 sur les DEUX matchs), verifiee par `TestZoneEtatPhase2aStabilite`. Reserve : la
  cle porteuse est le NUMERO DE SLOT, pas le tag 5 (absent ou instable sur les slots de jauge).
- [x] **CB.2a.2** — clause temporelle **NON TENUE** (precision 59,0 / 61,6 % pour un hasard de
  10,5 / 11,1 %, soit 5,6x le facteur 2 exige, mais rappel **74,8 / 77,5 %** sous le seuil de
  80 %) ; volet VALEUR **TENU : le tag 4 EST le PROPRIETAIRE** — valeurs `0xFFFFFFFF` / `0x0` /
  `0x1`, concordance avec l'index d'equipe du capteur **100,0 %** (48/48) et **91,1 %** (51/56)
  hors emissions neutres, un slot de proprietaire par zone (1530, 1535, 1540). « Conteste » NON
  MESURABLE : les slots de rampe ne portent pas de tag 4.
- [x] **CB.2a.3** — TENU au seuil ecrit : couverture **91,1 %** (`01e1f945`), **83,9 %**
  (`8076f97f`), **81,9 %** (`606d9844`) pour 80 % exiges sur >= 2 films. L'hypothese « une colline
  = un slot » est REFUTEE (un seul slot porte la jauge de tout un match KOTH) : la segmentation se
  fait par rampe. Clause faible, nettete inegale (2/3 films), `0a247154` non mesurable (Solitude
  absente du catalogue), temoin Slayer muet (0 rampe).
- [x] **Gate 2a : TENU**, par les deux branches (CB.2a.1 >= 90 % ET proprietaire etabli ET KOTH
  tenu). Forme PROPOSEE de `zoneStates` en section 7 du journal — **aucune publication faite**.
