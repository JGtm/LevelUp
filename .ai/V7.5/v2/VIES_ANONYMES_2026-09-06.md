# Correction des lecteurs qui supposaient « un slot = une piste nommée » — 2026-09-06/07

Branche `feat/v2-vies-anonymes`, worktree `LevelUp-wt-v2-vies`, base `7e5c454bc`
(= `feat/v2-durees`, schéma 45). Source : `.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md`
(cherry-pick `370955a35`), 14 constats retenus dont 2 P0 et 7 P1. Périmètre Go seulement — les
deux constats web (le report de lecture des fiches, le rendu d'une vie sans nom) sont traités
ailleurs.

---

## Décision : aucune vie anonyme (utilisateur, 2026-09-07)

**« Les vies anonymes n'existent pas ; une vie est un humain ou un bot, point. »**

Cette décision est arrivée EN COURS de lot et elle en change la doctrine. Jusque-là, l'invariant
de travail était « un nom manquant = inconnu, jamais absent » : une piste sans identité restait
une catégorie de donnée légitime, et le travail consistait à empêcher les lecteurs de la jeter.
Désormais :

- une piste publiée sans identité est un **DÉFAUT DE NOMMAGE du pont**, pas une donnée ;
- elle ne se publie pas comme un état « inconnu », elle ne se dessine pas en encre neutre ;
- elle se **répare à la source**, et ce qui résiste se **compte** et s'**alarme**.

Ce que la décision NE change pas : les correctifs de lecteurs restent tous nécessaires, comme
**défense**. Un lecteur ne doit jamais jeter une lecture VRAIE parce qu'un nom manque — parce que
le nommage peut échouer, parce qu'un artefact ancien en porte encore, et parce qu'un lecteur qui
dépend d'un nom pour publier une position est fragile par construction. Leur FORMULATION, elle,
a changé : ils ne parlent plus de « vie anonyme acceptée » mais de **lecture de la piste sous
l'identité résolue de son slot**.

Conséquence sur le lot : un item **P0-0** a été ajouté EN TÊTE (« nommer toutes les vies à la
source »), traité après les constats de l'audit parce qu'il en dépend — sans les lecteurs
corrigés, le nommage n'aurait rien à alimenter.

---

## Verdict en une page

| # | Constat | Verdict | Correctif |
|---|---|---|---|
| **P0-0** | Des vies publiées restent sans identité | **CONFIRMÉ** (décision produit) | Passe de nommage final par l'OCCUPATION DU SLOT DANS LE TEMPS + résidu publié et alarmé |
| **P0-1** | `samplesByXUID` n'indexe que les pistes nommées | **CONFIRMÉ** | `AttributeZones` exige le pont canonique |
| **P1-3** | `dropUnpublishedActions` cadencé sur le XUID nommé | **CONFIRMÉ** | `publishedXUIDs` + garde-rail étendu ; idem `neutral_deaths.go` |
| **P1-4** | Couverture des objectifs aveugle aux deux bouts | **CONFIRMÉ** | Le pont rend son compte d'écartés ; `warnIfLossy` voit `Unpublished` |
| **P1-5** | `ZoneCoverage` jetée à la ligne d'appel | **CONFIRMÉ** | Trois causes publiées + deux lignes de journal |
| **P1-6** | Garde-fous des fermetures sur le slot entier | **CONFIRMÉ** | Bornés à la vie DÉSIGNÉE |
| **P1-7** | La ride prend le premier occupant du slot | **CONFIRMÉ** | `OwnerReport.xuidAt(slot, instant)` |
| **P1-8** | Deux bots d'un même siège s'écrasent | **CONFIRMÉ** | `botNamesBySeat` s'abstient sur un siège ambigu |
| **Résidu A** | Rognage de `carrierPresence.gate` | **CONFIRMÉ — l'exemption ne le couvrait pas** | `unionOverlap` |
| **Résidu B** | Repli `dernier[slot]` d'`usage_summary.go` | **CONFIRMÉ — l'exemption ne le couvrait pas** | La vie sans nom occupe son slot avec un xuid VIDE |

**Aucun constat n'a été réfuté.** Les cinq P2 restent au registre des reports, non traités
(règle du zéro fix hors périmètre).

Schéma **45 → 47**. 44, 45 et 46 sont pris par les lots des manches, des durées et des drapeaux,
en cours sur d'autres branches.

---

## Méthode

Cuisson par `cmd/replay-build`, le chemin de production (`replaybuild.NewBuilder` + `BuildMatch`),
un film par processus borné : verrou `filmproc.AcquireSolo`, plafond mémoire 3 Gio, priorité
basse, `timeout 300`. Verrou inter-agents (`mkdir`/`rmdir` sur le scratchpad) posé autour de
CHAQUE cuisson et libéré quoi qu'il arrive.

**Le parc n'a reçu aucune écriture.** La racine de travail est dédiée : `film_chunks`,
`film_manifests`, `mvar`, `data/titles` et `config` y sont des JONCTIONS en lecture,
`data/cache/replays` un vrai dossier du scratchpad. Faits de match repris de l'export en lecture
seule du balayage (`facts/<short8>.facts.json`).

**Les deux côtés sont cuits par le MÊME outil**, seul le code diffère : `replay-build-avant.exe`
compilé sur la base (`7e5c454bc`, schéma 45), `replay-build-apres.exe` sur le HEAD corrigé
(schéma 47). Comparer au parc aurait mélangé les correctifs du lot avec quarante bumps de dérive.
Comparaison par `cmd/replay-diff`, axe des durées inclus.

---

## P0-0 — Aucune vie publiée ne reste sans nom

### Le diagnostic, chiffré

Neuf témoins cuits des deux côtés. « Sans nom » = piste publiée sans `xuid` ET sans `bot`.

| témoin | pistes | sans nom AVANT | sans nom APRÈS | par vie préc. | par vie suiv. | par le pont | frontières indécidables |
|---|---|---|---|---|---|---|---|
| `1b2d9e08` (bots) | 94 | 3 | **0** | 0 | 3 | 0 | 0 |
| `bf15f7ab` (slayer) | 91 | 9 | **1** | 0 | 8 | 0 | 0 |
| `af13e2b2` (zones) | 53 | 14 | **6** | 0 | 8 | 0 | 0 |
| `d9781168` (oddball) | 174 | 32 | **19** | 0 | 13 | 0 | 0 |
| `084a804d` (véhicules) | 344 | 142 | **80** | 9 | 52 | 0 | **1** |
| `3372e7eb` (objectifs) | 42 | 10 | **8** | 0 | 2 | 0 | 0 |
| `7344d24f` (zones) | 123 | 6 | **5** | 0 | 1 | 0 | 0 |
| `696a9d7c` (zones) | 110 | 3 | **3** | 0 | 0 | 0 | 0 |
| `51ebbc0f` (2 manches) | 86 | 75 | **73** | 0 | 2 | 0 | 0 |
| `000d5950` (golden) | 104 | 11 | **6** | — | — | — | — |

**Total sur les témoins : 305 vies sans nom → 201.** 104 réparées, **aucune inventée** — et la
seule frontière indécidable du corpus (`084a804d` slot 734) est refusée ET comptée plutôt que
tranchée au hasard, cf. le complément ci-dessous.

### Les causes, sur pièces

1. **Une vie qu'aucune mort ne termine.** `nameLivesByDeaths` nomme une vie par la mort qui la
   TERMINE. La dernière vie d'un joueur (de sa dernière mort au coup de sifflet) et les vies
   antérieures au début réel du match n'en ont pas. C'est la population MAJORITAIRE, et c'est
   celle que la passe finale répare — la prédominance de `parVieSuivante` sur `parViePrécédente`
   le dit : ces vies tombent surtout AVANT les vies nommées de leur slot, pas après.
2. **Un slot dont AUCUNE vie n'est nommée et que le pont ne nomme pas.** Ni (a), ni (b), ni (c)
   ne s'appliquent : rien dans le film ne le rattache. `696a9d7c` (3 vies sur 110) est
   entièrement dans ce cas — d'où son absence de progrès.
3. **Un pont d'identité globalement muet.** `51ebbc0f` porte 75 vies sans nom sur 86 AVANT : le
   film a DEUX MANCHES et sa reconstruction par manche est douteuse de bout en bout (déjà
   consigné : aucune de ses 8 courbes de score ne colle à la feuille de match). La passe finale
   n'a presque rien à quoi s'accrocher — 73 restent. **Ce n'est pas un défaut de la passe, c'est
   un défaut du pont EN AMONT**, et il est porté au registre des reports comme fait à instruire
   pour lui-même.
4. **`084a804d`** garde 80 vies sans nom sur 344 : c'est un film à 344 pistes pour 201 slots au
   pont, dont beaucoup de vies très courtes de slots jamais nommés. Même famille que (2).

### Le correctif

`internal/analysis/replay/unnamed_lives.go`, appelé dans `build.go` APRÈS les quatre passes
existantes (fil des morts, fermetures, sièges de bot, relais). Pour chaque piste encore sans
identité — les pistes de BOT sont intouchées, `Track.Bot` EST une identité :

1. la vie nommée du **MÊME slot** la plus proche **AVANT** elle ;
2. à défaut, la vie nommée du **MÊME slot** la plus proche **APRÈS** elle ;
3. à défaut, le **pont canonique** par slot (`SlotXUID`) — il couvre les slots qu'aucune vie
   nommée ne touche mais qu'une FERMETURE a attribués.

**La règle de collision est respectée par construction** : quand deux joueurs nommés se partagent
un slot, (1) et (2) rendent celui qui l'occupait à cet instant-là — jamais « le premier », qui est
ce que `SlotXUID` retient et ce que P1-7 a fait corriger ailleurs.

**Ce qui résiste n'est pas deviné** : `Coverage.Bridge.UnnamedLives` (publié au contrat) plus un
`slog.Error` portant le match, le nombre, et le slot et les bornes de la première. Trois
compteurs disent par quelle voie le reste a été comblé — les voies n'ont pas la même force de
preuve, un total unique les mélangerait.

### L'interaction attrapée par la cuisson : une déduction n'est pas une réfutation

La première cuisson après P0-0 a fait **perdre un portage et 101 frames sur `d9781168`**. Cause,
sur pièces : `carrierPresence.gate` s'abstient quand une vie SANS NOM recouvre l'intervalle
(c'est la moitié la plus coûteuse du correctif du schéma 43). Le nommage final vidant cette
population, l'abstention mourait avec elle, et le gate REJETAIT de nouveau.

**L'oracle tranche** : en Oddball le score EST le temps de portage. Feuille de match `d9781168` =
**387 s**. Publié : 331,3 s avec l'abstention, **321,2 s sans**. La perte éloignait l'artefact de
la vérité.

Correctif : une identité **DÉDUITE** (celle que la passe finale pose) entre dans les DEUX tables
de présence — sous son xuid, car c'est bien sa présence à lui, ET parmi les vies qui ne prouvent
l'absence de personne. **Une déduction ajoute une présence, elle n'en retire jamais une.** Les
indices des pistes déduites voyagent dans `unnamedLivesReport.deduced`.

### Le complément : une frontière entre deux occupants ne se tranche pas au hasard

Signalé par la revue du lot des durées, **vérifié sur pièces et corrigé** : `ownersFromLives`
(`lives.go`) publiait un slot en COLLISION sous le nom de son **PREMIER occupant nommé**, par
ordre des vies, et `SlotCollisions` n'en était qu'un TOTAL de match — aucun consommateur ne
pouvait savoir QUEL slot était concerné. Les marques de portage, les ramassages, les frags sous
équipement actif et les calques d'objectif héritaient donc d'un nom arbitraire sur ces slots.

**Le témoin, sur pièces** : `084a804d` slot 734 — A `2535430265968559` `[5872..6981]`, une vie non
résolue `[7123..7158]`, B `2535456423427614` `[7457..7591]`. La première version de la passe
nommait la vie du milieu à **A**, par la règle « l'occupant précédent ». C'est un choix par
l'ORDRE, pas par le temps : rien dans le film ne dit de quel côté de la relève elle tombe.

Trois corrections, à la source :

1. **`ownersFromLives` MARQUE le slot** (`OwnerReport.SlotAmbiguous`) au lieu de seulement
   compter ; `SlotCollisions` en devient le cardinal.
2. **Le repli par le pont s'abstient sur un slot ambigu**, dans les deux lecteurs qui l'emploient
   (`OwnerReport.xuidAt` et la passe de nommage) : servir `SlotXUID` à un instant que sa vie ne
   couvre pas publierait le nom arbitraire.
3. **La frontière est REFUSÉE et COMPTÉE** : quand la vie tombe entre deux vies nommées
   d'occupants DIFFÉRENTS, `occupantContested` — publié sous `bridge.unnamedLivesContested`.
   Ce qui pourrait la dater serait une SUCCESSION, mais `attributeSuccessions` a déjà couru et ne
   date que les relèves de BOT ; une relève entre deux humains n'est datée par rien de disponible
   ici. C'est écrit au code plutôt que deviné.

**Mesure après correction** : slot 734 laisse sa vie du milieu **sans nom**, et `084a804d` publie
`unnamedLives: 80` dont `unnamedLivesContested: 1`, `slotCollisions: 3` — les 3 slots à
plusieurs occupants sont désormais MARQUÉS, là où seul leur nombre était publié.

**Ce que cela déplace, et c'est une correction, pas une perte** : la ride du slot 734
`[7158..7384]` (227 frames, elle commence exactement à la fin de la vie non résolue) était
créditée à **A** par le pont ; elle sort désormais **sans occupant**. Elle reste PUBLIÉE — 74
rides des deux côtés, et le contrat le prévoit explicitement (« l'épisode reste publié — le
véhicule EST occupé, c'est son occupant qui est inconnu ») —, et ses 227 frames réapparaissent au
grain du slot (`rides/duree-totale/par-slot/734`). `ridesNamed` fait le bilan : 51 → **52** (+2
par la résolution dans le temps de P1-7, −1 par cette abstention honnête).

### Mutations

- `TestUneFrontiereEntreDeuxOccupantsNeSeTranchePasAuHASARD` — rouge sans le refus : la vie prend
  l'identité de A sans preuve. **Contre-épreuves** :
  `TestUneFrontiereEntreDEUXVIESDuMemeJoueurSeTrancheBien` (deux vies du MÊME joueur ne créent
  aucune ambiguïté) et `TestLePontNeSertPasDeRepliSurUnSlotAMBIGU` (le repli joue toujours sur un
  slot non ambigu).
- `TestUneVieNommableParLeTempsNeResteJamaisSansNom` — rouge sans la voie par le temps.
- `TestLeTempsTRANCHEEntreDeuxOccupantsDunMemeSlot` — rouge : `SlotXUID` dirait 111, le temps dit 222.
- `TestUneVieAnterieureAuPremierDecesPrendLOccupantSUIVANT` — rouge (voie b).
- `TestUnSlotQueSeuleUneFERMETURENommePasseParLePont` — voie c.
- `TestCeQuiResisteEstCOMPTE_JamaisDevine` — **la contre-épreuve** : deux slots que rien ne nomme
  restent sans nom et entrent au résidu ; aucune identité inventée.
- `TestUneVieDeBotNEstPasUnDefautDeNommage`, `TestUneVieDejaNommeeNEstJamaisReecrite`.
- `TestUneIdentiteDEDUITENeProuveLAbsenceDePersonne` + sa contre-épreuve
  `TestUneIdentiteLUEProuveToujoursUnePresence`.

---

## P0-1 — Les zones lisent la piste sous l'identité résolue du slot

- **Verdict : CONFIRMÉ.** `samplesByXUID` (`zone_attribution.go`) était la MÊME construction que
  le `tracksByXUID` du drapeau avant le schéma 45, dans le fichier voisin et sans le pont.
- **Conséquence** : une capture pendant une vie non résolue → pas d'échantillon à ≤ 2 frames →
  `NoPosition` → la capture ne vote plus à l'appariement jauge ↔ propriétaire → et quand plus
  aucune n'est attribuée, `buildZoneStates` rend `nil` hors mode à colline : le calque
  `zoneStates` **ENTIER** (propriétaire, jauge en direct, lettres A/B/C) disparaît de l'artefact.
- **Correctif** : `AttributeZones` prend le pont canonique (5 paramètres, dans le seuil) et lit
  chaque piste par `xuidOfPublishedTrack`. Le pont descend de `build.go` par `zoneCtx`.
  `cmd/zone-attribution` reconstruit le pont depuis les pistes publiées (`slotBridgeOf`, même
  règle de collision qu'`ownersFromLives`) : sans lui l'outil de mesure croiserait sur MOINS de
  pistes que la production.
- **Mutations** : `TestCaptureSurVieNonNommeeEstAttribueeParLePont` (0 attribuée / NoPosition 1)
  et `TestZonesAttribueLesCapturesDesViesNonNommees` (0 zone publiée — le calque entier
  disparaît). **Contre-épreuve** : `TestCaptureSansPontResteSansPosition`, trois sous-cas (pont
  muet, autre slot, autre nom).
- **Témoins — CORRIGÉ par la revue VIES-R1 (C3)** : la première rédaction annonçait
  « `af13e2b2` +19 gains ». **C'est faux, et les gains appartiennent à P0-0.** Mesuré des deux
  côtés : `zones.captures` **19 → 19**, `zones.attributed` **14 → 14** — *aucune capture
  récupérée* sur ce témoin ; les 19 gains sont tous des pistes nommées. Ce que P0-1 apporte
  vraiment ici est la CAUSE, désormais lisible : `zones.noPosition = 5` au HEAD, champ qui
  n'existait pas. P0-1 est donc un **correctif de défense** — sa mutation rougit (`M9`), le
  chemin est fermé — mais **aucune mesure de parc ne le chiffre** : le corpus local ne porte pas
  de film où une capture tombe sur une vie que seul le pont nomme.

## P1-3 — Le filtre « piste publiée » passe par le pont

- **Verdict : CONFIRMÉ.** `dropUnpublishedActions` (`objectives.go`) et
  `keepNeutralDeathsOfPublishedTracks` (`neutral_deaths.go`) étaient les **deux SEULS** des treize
  filtres du paquet à cadencer sur le nom LU ; les onze autres cadencent sur le SLOT via
  `keepOfPublishedTracks`.
- **Conséquence** : un joueur dont aucune vie n'est nommée perdait TOUTES ses actions d'objectif —
  35 sur 76 (46 %) sur `3372e7eb`, 7 artefacts du parc sur 111. Trois consommateurs, dont **deux
  qui n'ont jamais eu besoin d'une trajectoire** : le SON d'objectif (ne lit que l'instant) et la
  garde tout-ou-rien de l'armement de bombe.
- **Correctif** : `xuidOfPublishedTrack` / `publishedXUIDs` dans `published_tracks.go`, une seule
  écriture de la règle, `tracksByXUID` y passe aussi.
- **Garde-rail** : il ne filtrait que le motif keyé par `.Slot`, qu'une map keyée par chaîne ne
  déclenche jamais — **c'est ce qui a laissé ces deux sites diverger sans être vus**. Motif par
  XUID ajouté, ancré sur `range tracks` (sans quoi il attrape `rosterFromDeaths`, un autre espace
  de clés — un garde-rail qui crie sur du code juste finit désactivé), failabilité prouvée sur un
  extrait synthétique.
- **Mutations** : `TestActionDunJoueurSansVieNommeeEstPubliee` et
  `TestMortNeutreDunJoueurSansVieNommeeEstPubliee` (0 publiée au lieu de 1). **Contre-épreuve** :
  `TestActionSansPontResteEcartee`, quatre sous-cas.
- **Témoin — CORRIGÉ par la revue VIES-R1 (C3)** : la première rédaction annonçait
  « `3372e7eb` +7 gains » et attribuait ses 35 actions perdues à « un joueur dont aucune vie
  n'est nommée ». **Les deux sont faux.** Mesuré des deux côtés : `objectives.unpublished`
  **35 → 35**, `attached` **41 → 41** — *aucune action récupérée* ; les 7 gains sont la version
  de schéma et des pistes nommées (P0-0). Et la cause réelle est autre : les 35 actions
  appartiennent à `2535429116401024` et `2535433300797518`, **absents du roster ET sans aucune
  piste dans le film** (roster 6, feuille 8) — le repli par le pont ne peut pas les atteindre,
  puisqu'il n'y a aucune piste à nommer. P1-3 reste un **correctif de défense** portant
  (mutation `M10` rouge) ; le fait « joueur de la feuille sans aucune piste » est porté au
  registre des reports, et il relève du lot du pont muet.

## P1-4 — La couverture voit l'amont, et l'alarme voit la bonne catégorie

- **Verdict : CONFIRMÉ, aux deux bouts.**
  - *Dénominateur* : `IdentifyNamedEventsByRound` n'annonçait que les rescapés. `Available`
    comptait les actions DÉJÀ identifiées ; le rapport se lisait ~100 % sur un calque partiel, et
    `noSlot` — le seul champ du contrat public prévu pour « le pont ne couvre pas ce joueur » —
    valait **0 sur les 111 artefacts du parc, sans une seule exception**.
  - *Alarme* : `warnIfLossy` ne parcourait que `{NoSlot, Ambiguous, OutOfWindow}`. `Unpublished`,
    la seule catégorie qui BOUGE, n'avait aucun seuil : 46 % des actions de `3372e7eb`
    disparaissaient **sans une ligne de journal**, pour un seuil de 10 %.
- **Correctif** : le pont rend son compte d'écartés, il descend par `Options.ObjectivesUnnamed`
  jusqu'au dénominateur ET sous `noSlot` ; `warnIfLossy` surveille les quatre catégories.
- **Branche `e.XUID == ""` conservée**, et c'est écrit dans le code : sa population de production
  est nulle (les deux ponts écartent déjà le xuid vide), elle garde l'invariant du champ publié
  `ObjectiveAction.XUID`. Le vrai peuplement de `NoSlot` vient désormais de l'amont — la branche
  n'est plus seule à tenir la catégorie verte.
- **Mutation** : `TestCouvertureCompteCeQueLePontNaPasNomme` (disponibles 2 au lieu de 5, sansSlot
  0 au lieu de 3) ; `TestWarnIfLossyAlerteAussiSurLesNonPubliees`.
- **Témoin** : `3372e7eb` publie désormais `unnamedLives` et les compteurs de nommage.

## P1-5 — La cause d'une capture perdue est publiée

- **Verdict : CONFIRMÉ.** `att, _ := AttributeZones(...)` jetait `ZoneCoverage`, et `ZonesCoverage`
  n'avait aucun champ pour l'accueillir.
- **Correctif** : `noPosition`, `outside`, `ambiguousZone` publiés (contrat OpenAPI + types web
  regénérés), invariant `attribuées + sansPosition + dehors + ambiguës == captures`, un WARN dès
  qu'une capture se perd et les quatre compteurs dans la ligne de couverture.
- **Mutation** : `TestZonesCouverturePublieLaCauseDesCapturesPerdues` (les trois causes à zéro,
  invariant rompu 4 ≠ 6).

## P1-6 — Les fermetures bornent sur la vie DÉSIGNÉE

- **Verdict : CONFIRMÉ, aux deux fermetures.** `overlapsNamedLife` testait le recouvrement sur le
  nuage COMPLET du slot ; `bodyExtendsShooter` datait la terminalité sur le début de la PREMIÈRE
  vie du slot.
- **L'autre côté reste le slot entier, et c'est voulu** : `owner[s] = pi` attribue TOUT le slot à
  ce joueur, et un trou de réplication n'est pas une absence (un porteur invisible et immobile
  cesse d'être répliqué). Le resserrer rendrait le garde-fou plus permissif sans rien prouver.
- **Mutations** : `TestFermetureBTesteLaVieDesigneeEtPasLeSlotEntier` et
  `TestFermetureADateLaTerminaliteSurLaVieDesignee` (refused = 1, slot non attribué).
  **Contre-épreuves** : un VRAI recouvrement et un corps POSTÉRIEUR restent refusés.
- **Témoin** : `084a804d`, `closedRefused` 6 → 5 et `bridge.slots` 200 → 201 — une déduction
  rendue au pont, et **11 tirs de plus publiés** (`shots/n` 3193 → 3204).

## P1-7 — L'occupant d'un véhicule suit le temps

- **Verdict : CONFIRMÉ sur le mécanisme.** `SlotXUID` est une identité UNIQUE par slot pour tout le
  match : `ownersFromLives` garde la PREMIÈRE vie nommée. Les deux voies d'assemblage lisaient le
  pont sans instant.
- **Enjeu** : `VehicleRide.XUID` donne sa COULEUR au véhicule — le sprite prenait l'équipe du
  mauvais joueur, et le document se contredisait (la piste du même slot au même instant portait
  l'autre nom).
- **Correctif** : `OwnerReport.xuidAt(slot, instant)` — par vie d'abord, pont en repli.
- **Mutation** : `TestOwnerReportXuidAtSuitLOccupantDansLeTemps` (occupant à 30 s = 111 au lieu de
  222).
- **Témoin — l'oracle est le calque lui-même** : `084a804d` passe de 51 à **53 rides nommées**, et
  la ride du slot 590 (282 frames) **quitte le seau non attribué `par-slot/590` pour
  `par-xuid/2535467063146739`** (171 → 453 frames, +282 exactement). C'est la seule ligne
  « disparue » du lot, et c'est une attribution gagnée.

## P1-8 — Deux bots d'un même siège ne s'écrasent plus

- **Verdict : CONFIRMÉ.** `nameByIndex[b.FilmIndex] = b.Name` : le dernier balayé gagnait, sans
  ordre garanti. Le cas est MESURÉ (RE_LOG 7ter.62, « 343 Aloysius » puis « 343 PardonMy », les
  deux déclarant slot=8) et la doctrine est écrite quatre fonctions plus haut, dans `buildRoster`,
  qui la respecte.
- **Correctif** : `botNamesBySeat` s'abstient sur un siège ambigu ; ses vies restent libres pour
  `attributeSuccessions`, qui seul les départage par l'instant de bascule.
- **Pourquoi pas un simple échange d'ordre** avec `attributeSuccessions`, comme envisagé :
  **vérifié sur pièces**, `candidateIn` ne restreint PAS ses candidates au slot du remplaçant — il
  balaie toutes les pistes anonymes de la fenêtre. Faire courir les relais en premier leur
  laisserait prendre des vies que le nommage par siège attribue correctement. L'abstention ne
  libère que les sièges réellement ambigus, ce qui est exactement le but.
- **Mutation** : `TestDeuxBotsSurUnMemeSiegeNeSecrasentPas` (siège 8 nommé « 343 PardonMy », piste
  non libérée). **Contre-épreuve** : `TestUnMemeBotDeclareDeuxFoisResteNomme`.
- **Témoin** : `1b2d9e08`, le seul film à bots du parc — **+9 gains, 0 perte**, et ses 3 vies sans
  nom tombent à **0**.

## Résidu A — le rognage de `carrierPresence.gate`

- **Verdict : CONFIRMÉ — l'exemption « lecteur déjà rattrapé » ne le couvrait PAS.** Le correctif
  du schéma 43 a traité le REJET (« l'ignorance passe avant le rognage ») ; le ROGNAGE est resté
  sur `bestOverlap`, la vie de recouvrement MAXIMAL. C'est exactement le défaut que `windowFor`
  portait avant le schéma 45 et que `spanFor` a fermé.
- **Correctif** : `unionOverlap`. `bestOverlap`, dont plus aucun appelant n'exploitait le
  « best », est supprimée (règle n°7).
- **Mutation** : portage [80..100] au lieu de [20..100], 60 frames amputées. **Contre-épreuves** :
  l'union reste bornée par les vies, un portage hors de toute vie nommée reste écarté.

## Résidu B — le repli `dernier[slot]` d'`usage_summary.go`

- **Verdict : CONFIRMÉ — l'exemption ne le couvrait PAS.** Le correctif du 2026-09-06 a remplacé
  l'agrégat « dernier gagnant » par une résolution PAR VIE, mais `usageSlotOwners` continuait
  d'écarter de `parVie` les vies non résolues : `at()` retombait sur le DERNIER occupant du match,
  c'est-à-dire précisément la règle que le correctif déclare avoir supprimée.
- **Correctif** : la vie entre avec un xuid VIDE — elle n'ouvre aucune ligne mais elle OCCUPE son
  slot. Même traitement que les vies de BOT, pour la même raison. `UsageSummaryRev` us2 → **us3**.
- **Mutation** : « propriétaire à la frame 150 = B » au lieu de vide.

---

## Témoins : avant / après, élément par élément

Neuf films, les deux côtés cuits par le même outil, seul le code diffère.

| témoin | famille | gains | **pertes** | apparus | identiques |
|---|---|---|---|---|---|
| `696a9d7c` | zones (Vagabond) | 1 | **0** | 3 | 694 |
| `7344d24f` | zones (Vagabond) | 5 | **0** | 4 | 669 |
| `af13e2b2` | zones (Origin) | 19 | **0** | 3 | 578 |
| `3372e7eb` | objectifs (Isolation) | 7 | **0** | 2 | 624 |
| `084a804d` | véhicules (Fortitude) | 68 | **6** (instruites ci-dessous) | 6 | 987 |
| `1b2d9e08` | bots (Dynasty) | 9 | **0** | 1 | 581 |
| `bf15f7ab` | slayer (Perilous) — non concerné | 13 | **0** | 2 | 578 |
| `d9781168` | oddball (Dredge) | 15 | **0** | 2 | 589 |
| `51ebbc0f` | deux manches | 5 | **0** | 2 | 588 |
| **total** | | **142** | **6** | 25 | |

### Les six lignes de `084a804d`, instruites une par une — aucune n'est une perte

| ligne | lecture | contre-partie |
|---|---|---|
| `coverage.bridge.closedRefused` 6 → 5 | une déduction de moins REFUSÉE | `closedByRespawn` 4 → 5, `bridge.slots` 200 → **201** (P1-6) |
| `coverage.shots.noSlot` 2785 → 2768 | 17 tirs de moins sans slot | `shots.attached` 3193 → **3204** (+11) et `shots.ambiguous` 36 → 42 (+6) — somme exacte |
| `coverage.vehicles.shotsNoRide` 2785 → 2768 | miroir du précédent | `vehicles.shots` 290 → **295**, `shotsVehicleWeapon` 27 → **32** |
| `coverage.t0Film.burst` 21 → 20 | un partant de moins dans la rafale | **déduplication** : `t0FilmMs` = 44910 et `marginMs` = 22600 **identiques des deux côtés** — deux pistes comptées séparément partagent désormais une identité. Le verdict du coup d'envoi ne bouge pas. |
| `vehicles.rides/duree-totale/par-slot/590` 282 → *(disparue)* | la ride quitte le seau NON ATTRIBUÉ | `par-xuid/2535467063146739` 171 → **453** (+282 exactement) (P1-7) |
| `vehicles.rides/par-xuid/2535430265968559` 5 → 4 et sa durée 794 → 567 | la ride du slot 734 n'est plus créditée à A par un nom ARBITRAIRE du pont | elle reste PUBLIÉE (74 rides des deux côtés) et ses 227 frames réapparaissent en `par-slot/734` — **correction**, cf. le complément de P0-0 ci-dessus. Bilan : `ridesNamed` 51 → **52** |

**Aucune perte réelle sur les neuf témoins** : quatre lignes sont des gains lus à l'envers, et
les deux dernières sont une attribution ARBITRAIRE retirée. Le témoin non concerné (`bf15f7ab`, slayer) ne perd
rien et gagne 13 mesures, toutes de nommage.

### Oracles

- **`d9781168` — la feuille de match.** En Oddball le score EST le temps de portage : 387 s réelles.
  Publié 331,3 s avec l'abstention par identité déduite, 321,2 s sans. L'oracle a tranché contre la
  version qui perdait un portage, et c'est lui qui a dicté le correctif.
- **`084a804d` — le calque véhicules lui-même.** Une ride ne peut pas être à la fois « sans
  occupant » et « à ce joueur » : les 282 frames quittent le seau par-slot pour le seau par-xuid,
  au chiffre près. Aucune durée n'est créée.
- **`3372e7eb` — la feuille d'actions.** 35 actions sur 76 étaient supprimées ; elles reviennent
  sans qu'aucune action nouvelle n'apparaisse.
- **`000d5950` — le golden d'assemblage.** 93 pistes nommées → **98**, sur 104. Le fichier figé le
  montre ligne à ligne.

---

## Schéma et contrat

`SchemaVersion` **45 → 47**, chronique écrite dans `document.go`, raison dans le ratchet
`structure_test.go`. **44, 45 et 46 sont pris** par les lots des manches, des durées et des
drapeaux, en cours sur d'autres branches.

Sept champs s'ajoutent au contrat, tous publiés et typés côté web (`openapi.yaml` +
`generated.ts` regénéré par `openapi-typescript`, diff limité aux champs ajoutés) :
`bridge.namedByPreviousLife`, `bridge.namedByNextLife`, `bridge.namedBySlotBridge`,
`bridge.unnamedLives`, `zones.noPosition`, `zones.outside`, `zones.ambiguousZone`.

Golden d'assemblage régénéré. Deux écarts, tous deux voulus : la ligne de version, et la ligne des
pistes nommées (93/11 → 98/6) — dont la formulation, qui décrivait une piste sans nom comme « une
LIMITE, pas une erreur », a été corrigée pour ne plus contredire la décision du 2026-09-07
(anti-patron n°9, doc inversée). Le garde-rail de phrases du golden suit.

`UsageSummaryRev` **us2 → us3** (résidu B change une règle de projection).

---

## Découvertes, notées et NON traitées

1. **`51ebbc0f` garde 73 vies sans nom sur 86.** Le pont d'identité y est muet de bout en bout
   (film à deux manches ; aucune de ses 8 courbes de score ne colle à la feuille). Ce n'est pas un
   défaut de la passe de nommage mais du pont EN AMONT. **Au registre des reports.**
2. **`696a9d7c` ne progresse pas** (3 vies sans nom, avant comme après) : ses trois slots ne
   portent aucune vie nommée et le pont ne les nomme pas. Même famille que (1), volume négligeable.
3. **`084a804d` garde 80 vies sans nom sur 344** — 344 pistes pour 201 slots au pont, beaucoup de
   vies très courtes sur des slots jamais nommés. Même famille que (1).
4. **Les cinq P2 de l'audit** (`birthOfLives`, `coverage.equipmentChanges.lives`, la borne de
   manche, `indexBySlot` web, les jointures roster↔joueur qui perdent les bots) restent au
   registre, non traités — règle du zéro fix hors périmètre.

---

## Corrections R1 (revue adversariale VIES-R1, 2026-09-07)

Revue close : **cœur exact** (rien perdu, rien inventé, 14 mutations rouges dont une cuisson
mutée, oracle Oddball tenu sur l'artefact réel), **aucun P0**, huit constats corrigés ici.

### C1 (P1) — le compte des actions écartées ne quittait jamais `replaybuild`

**Le défaut** : `filmStats.objectivesUnnamed` était écrit (`matchfacts.go:112`) et **jamais lu** —
le littéral `replay.Options{...}` de `BuildMatch` ne le passait pas. Champ mort (anti-patron n°1),
qu'aucun gate n'attrape : `go vet` et `golangci-lint` ne signalent pas un champ de structure
inutilisé. **P1-4 n'était donc livré qu'à moitié** : `coverage.objectives.available` restait un
compte de RESCAPÉS et `noSlot` restait structurellement à 0 — le défaut même que l'item déclarait
corriger. (L'autre moitié, `warnIfLossy` sur `Unpublished`, était bien branchée.)

**Le correctif** : `ObjectivesUnnamed: stats.objectivesUnnamed` dans `replaybuild.go`.

**La preuve, par cuisson** — `c0a82e88`, les deux côtés au même outil, seul C1 diffère :

| | `available` | `attached` | `noSlot` |
|---|---|---|---|
| sans C1 | 23 | 23 | **0** |
| **avec C1** | **92** | 23 | **69** |

Le dénominateur compte enfin les événements que le FILM porte (92) et non les rescapés du pont
(23) ; l'invariant tient au chiffre près (23 + 69 = 92). `3372e7eb` ne bouge pas (76/41/0/35) —
son pont nomme tout ce qu'il peut, et ses 35 pertes sont d'une autre nature (cf. C3).

**Le garde-rail** : `film_stats_cables_guard_test.go` lit LA SOURCE de l'assemblage et exige que
chaque champ de `filmStats` apparaisse dans le littéral `replay.Options{...}` — un champ
délibérément non câblé s'inscrit dans `champsNonCables` avec sa raison. Un test unitaire qui
reconstruirait le littéral serait resté vert quoi qu'il arrive ; c'est la même logique que
`published_tracks_guard_test.go`. **Mutation** : le champ débranché, le garde rougit
(« champ(s) de filmStats ÉCRIT(S) mais jamais passé(s) à replay.Options : [objectivesUnnamed] »).

### C2 (P1) — le helper partagé servait le nom arbitraire là où la passe s'abstient

**Le défaut** : `xuidOfPublishedTrack` repliait sur `slotXUID[t.Slot]` **sans la garde
`SlotAmbiguous`** que le lot venait d'ajouter à ses deux jumeaux (`bridgeOfSlot`, `xuidAt`) pour
cette raison précise. Sur la configuration réelle du slot 734 de `084a804d`, la passe de nommage
REFUSE (`contested = 1`) et le helper servait quand même `2535430265968559` : `samplesByXUID`
(zones, chemin NEUF de ce lot) indexait les positions de la piste contestée sous le PREMIER
occupant — **une capture de zone pouvait être géolocalisée sur la trajectoire d'un autre joueur**.
Même exposition pour `tracksByXUID` (drapeau).

**Le correctif, à la source** : `OwnerReport.NamingBridge()` rend le pont **débarrassé des slots
ambigus**, et les quatre lecteurs qui NOMMENT une piste le reçoivent (zones, pistes de porteur de
drapeau, actions d'objectif, morts neutres). Plutôt que de faire descendre `SlotAmbiguous` dans
quatre chaînes d'appel, on retire les slots ambigus en amont : **le lecteur ne peut plus oublier
la garde, puisqu'il n'a plus de quoi l'enfreindre.** `SlotXUID` reste inchangé pour ses autres
consommateurs (ramassages, marques de portage, frags sous équipement actif) — leur exemption est
explicite au cadrage de l'audit, et la modifier sortirait du périmètre.

**Mutation** : `NamingBridge` rendant `SlotXUID` tel quel,
`TestUnePisteCONTESTEEnEstJamaisIndexeeSousUnXUID` rougit sur les deux assertions (piste indexée
sous `2535430265968559`, échantillons de zone servis). **Contre-épreuve** : sur un slot non
ambigu, le repli par le pont joue toujours.

### C3 (P2) — deux témoins chiffrés ne tenaient pas

La revue a recuit `af13e2b2` et `3372e7eb` des deux côtés. Verdict, sur pièces :

| témoin | ce que le journal annonçait | ce qui est MESURÉ |
|---|---|---|
| `af13e2b2` (P0-1) | « +19 gains » | `zones.captures` **19 → 19**, `attributed` **14 → 14** — *aucune capture récupérée*. Les 19 gains sont des pistes nommées, donc **P0-0**. Apport réel de P0-1 : `zones.noPosition = 5`, la CAUSE, champ qui n'existait pas |
| `3372e7eb` (P1-3) | « +7 gains » et « un joueur dont aucune vie n'est nommée » | `objectives.unpublished` **35 → 35**, `attached` **41 → 41** — *aucune action récupérée*. Et la cause écrite est **fausse** : les 35 actions appartiennent à `2535429116401024` et `2535433300797518`, **absents du roster ET sans aucune piste** (roster 6, feuille 8) |

**P0-1 et P1-3 restent des correctifs portants** — leurs mutations rougissent (`M9`, `M10`), le
chemin est fermé — mais ce sont des **correctifs de DÉFENSE, non chiffrés sur le parc** : aucun
film local ne porte la configuration déclenchante. Le journal, la chronique du schéma, le ratchet
et les commentaires de `objectives.go`, `published_tracks.go` et de leurs tests sont corrigés en
conséquence : **le seul gain MESURÉ du lot est celui du nommage** (305 → 201 vies sans nom).

Le fait « joueur de la feuille de match sans aucune piste dans le film » (`3372e7eb` : 2 sur 8)
entre au registre des reports — il relève du lot du pont muet.

### C4 à C8 (P3)

- **C4** — la chronique (`document.go`) et le ratchet (`structure_test.go`) énuméraient **7**
  champs quand l'artefact en porte **8** : `bridge.unnamedLivesContested`, ajouté par le
  complément, n'avait pas d'entrée. Corrigé, avec le paragraphe « frontières » qui manquait.
- **C5** — `coverage.go` était passé de 497 à **522 lignes** sans exemption (règle n°5, qu'aucun
  linter n'attrape). La santé du PONT sort dans `coverage_bridge.go` — déplacement PUR, frontière
  naturelle du sujet : `coverage.go` porte ce que chaque CALQUE a rattaché, l'autre ce que le PONT
  a su nommer. **385 + 146 lignes.**
- **C6** — `UsageSummaryRev = "us3"` sans entrée de chronique (doc inversée : un lecteur concluait
  que les résumés `us2` sont à jour). Entrée `us3` datée, avec ce qu'elle change et ce qu'elle
  oblige à refaire.
- **C7** — `t0_film.go` affirmait « le film n'offre aucun moyen de replier une piste anonyme sur
  un joueur » : depuis `unnamed_lives.go` il en offre un, et `burst` passe 21 → 20 sur
  `084a804d`. Le commentaire est corrigé ET **le couplage est écrit** : le nommage ne peut que
  FAIRE BAISSER ce compteur, or `burst` est une GARDE (`t0FilmMinBurst = 2`, en deçà le coup
  d'envoi n'est plus daté). Risque non observé (les deux artefacts du parc à `t0Film` ont un burst
  de 5 et 6, et le verdict de `084a804d` ne bouge pas), mais désormais connu.
- **C8** — « `084a804d` garde 79 vies sans nom » → **80**, la valeur publiée
  (`bridge.unnamedLives = 80`, et 344 − 264 pistes nommées = 80).
