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

| témoin | pistes | sans nom AVANT | sans nom APRÈS | par vie préc. | par vie suiv. | par le pont |
|---|---|---|---|---|---|---|
| `1b2d9e08` (bots) | 94 | 3 | **0** | 0 | 3 | 0 |
| `bf15f7ab` (slayer) | 91 | 9 | **1** | 0 | 8 | 0 |
| `af13e2b2` (zones) | 53 | 14 | **6** | 0 | 8 | 0 |
| `d9781168` (oddball) | 174 | 32 | **19** | 0 | 13 | 0 |
| `084a804d` (véhicules) | 344 | 142 | **79** | 10 | 52 | 0 |
| `3372e7eb` (objectifs) | 42 | 10 | **8** | 0 | 2 | 0 |
| `7344d24f` (zones) | 123 | 6 | **5** | 0 | 1 | 0 |
| `696a9d7c` (zones) | 110 | 3 | **3** | 0 | 0 | 0 |
| `51ebbc0f` (2 manches) | 86 | 75 | **73** | 0 | 2 | 0 |
| `000d5950` (golden) | 104 | 11 | **6** | — | — | — |

**Total sur les témoins : 305 vies sans nom → 200.** 105 réparées, aucune inventée.

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
4. **`084a804d`** garde 79 vies sans nom sur 344 : c'est un film à 344 pistes pour 201 slots au
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

### Mutations

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
- **Témoins** : `af13e2b2` **+19 gains, 0 perte** ; `7344d24f` +5 ; `696a9d7c` +1.

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
- **Témoin** : `3372e7eb` **+7 gains, 0 perte**.

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
| `084a804d` | véhicules (Fortitude) | 68 | **4** (instruites ci-dessous) | 4 | 987 |
| `1b2d9e08` | bots (Dynasty) | 9 | **0** | 1 | 581 |
| `bf15f7ab` | slayer (Perilous) — non concerné | 13 | **0** | 2 | 578 |
| `d9781168` | oddball (Dredge) | 15 | **0** | 2 | 589 |
| `51ebbc0f` | deux manches | 5 | **0** | 2 | 588 |
| **total** | | **142** | **4** | 23 | |

### Les quatre lignes de `084a804d`, instruites une par une — toutes sont des GAINS

| ligne | lecture | contre-partie |
|---|---|---|
| `coverage.bridge.closedRefused` 6 → 5 | une déduction de moins REFUSÉE | `closedByRespawn` 4 → 5, `bridge.slots` 200 → **201** (P1-6) |
| `coverage.shots.noSlot` 2785 → 2768 | 17 tirs de moins sans slot | `shots.attached` 3193 → **3204** (+11) et `shots.ambiguous` 36 → 42 (+6) — somme exacte |
| `coverage.vehicles.shotsNoRide` 2785 → 2768 | miroir du précédent | `vehicles.shots` 290 → **295**, `shotsVehicleWeapon` 27 → **32** |
| `coverage.t0Film.burst` 21 → 20 | un partant de moins dans la rafale | **déduplication** : `t0FilmMs` = 44910 et `marginMs` = 22600 **identiques des deux côtés** — deux pistes comptées séparément partagent désormais une identité. Le verdict du coup d'envoi ne bouge pas. |
| `vehicles.rides/duree-totale/par-slot/590` 282 → *(disparue)* | la ride quitte le seau NON ATTRIBUÉ | `par-xuid/2535467063146739` 171 → **453** (+282 exactement), `ridesNamed` 51 → **53** (P1-7) |

**Aucune perte réelle sur les neuf témoins.** Le témoin non concerné (`bf15f7ab`, slayer) ne perd
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
3. **`084a804d` garde 79 vies sans nom sur 344** — 344 pistes pour 201 slots au pont, beaucoup de
   vies très courtes sur des slots jamais nommés. Même famille que (1).
4. **Les cinq P2 de l'audit** (`birthOfLives`, `coverage.equipmentChanges.lives`, la borne de
   manche, `indexBySlot` web, les jointures roster↔joueur qui perdent les bots) restent au
   registre, non traités — règle du zéro fix hors périmètre.
