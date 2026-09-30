# Banc de vérité du décodeur de film — conception (phase 1, 2026-09-30)

> Contexte : `.ai/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md` (ADR 0034). Branche
> `feat/suite-audit-decodeur-verite`, base `8cd560673`. Phase 1 = inventaire et conception, AUCUN code de
> production. Chiffres d'essai calculés en lecture seule sur des artefacts déjà cuits (aucun décodage,
> aucune base ouverte) : `C:/Users/GUILLA~1/AppData/Local/Temp/j11/art/<révision>/<short8>.json` et
> `.../j11/facts/<short8>.facts.json`, par un script jetable hors dépôt (non versionné ; la phase 2
> réimplémente en Go).

## 0. En une page

**But.** Juger une évolution du décodeur par des SCORES contre des ORACLES et des COMPTES DE VIOLATIONS,
calculés sur l'artefact publié de chaque témoin, avant et après, au lieu d'attribuer à la main les
505 feuilles de couverture de `replaydiff`.

**Trois constats d'inventaire qui changent la conception.**

1. **Plusieurs comparaisons « évidentes » sont CIRCULAIRES.** Les faits du match sont une ENTRÉE de la
   cuisson, pas seulement un oracle :
   - un slot statborg nommé par `triplet_feuille` a été nommé PARCE QUE son K/D/A final égale l'API
     (`objectives.SlotIdentityFrom`, `film/internal/facts/objectives/slotidentity.go:77`) ;
   - un camp résolu par `teamIdentity = a` ou `a0` a été rattaché PARCE QUE son score final égale
     `teamScores` (`film/replay/score_team_identity.go:72,103`) ;
   - le banc doit donc lire la MÉTHODE publiée (`identity.statborgSlots[].link.method`,
     `coverage.score.teamIdentity`) et exclure du calcul d'exactitude ce qui a été apparié sur
     l'oracle lui-même.

   Sur les 19 témoins, les slots statborg sont nommés à 97 % par `instants_de_mort` : ce sont les
   instants du fil des morts du film, donc c'est non circulaire. Un seul slot est nommé par
   `triplet_feuille` (`bcb6d393`). En revanche 15 témoins sur 19 ont leurs camps résolus par `a`/`a0`.
   Le score final d'équipe n'y est donc PAS un oracle.
2. **L'artefact ne publie aucun kill individuel.** La liste des kills (tueur, victime, assistant,
   origine `credit-concordant`…) vit dans la section 5 du fichier de faits du film
   (`replay.FilmFactsFile.Kills *killsource.Result`, `film/replay/filmfacts_fichier.go:216-241`). La
   cuisson du gate l'écrit dans sa racine de travail (`replaybuild/filmfacts_rangement.go`,
   `PathResolver(workRoot).FilmFactsPath`). C'est la seule source non circulaire des kills par joueur,
   et elle ne coûte aucun décodage de plus.
3. **La fermeture au bit près est DÉJÀ publiée au niveau du paquet.** `coverage.continuousFire.{packets,
   reached, closed}` porte le même verdict de vue C que `grammar.FrameClosure`. Vérifié à l'identique
   contre la carte J4.0.5 : `11de8353` 5 738/17 629 des deux côtés. Les « records utiles fermés » ne
   sont PAS publiés.

**Règle de verdict proposée.**
- Pour chaque oracle non circulaire, ni les faux positifs (FP) ni les faux négatifs (FN) ne montent.
  Les totaux officiels sont fixes, donc VP + FN est constant par joueur et « FN ne monte pas » équivaut
  à « VP ne baisse pas ».
- Aucune classe de violation ne monte.
- Tout le reste (les différences `replaydiff`) est affiché à titre d'information.
- Deux filets restent bloquants, voir §6.3.

**Démonstration sur une vraie fusion** (§7.3). Rejoué sur la fusion J5+J6+J9 (`8f89aeedc` → `d6701c058`),
le banc voit sur `084a804d` :
- des gains nets de vérité : morts par les vies FN 125 → 4, kills FN 240 → 230, score du camp 0 : 2 → 3 (= API) ;
- des corrections de faux : ramassages hors vie 167 → 0, grenades hors vie 62 → 20 ;
- ET trois faux nouveaux : 3 vies en trop, 3 doubles corps sur 866 images, 4 échantillons de trajet à plus
  de 3 m du véhicule.

Le banc aurait nommé ces 10 objets en quelques lignes, là où il a fallu attribuer à la main des
centaines de feuilles.

---

## 1. Inventaire de l'existant (ne pas réinventer)

`R` = `apps/go-api`, `F` = `R/internal/games/halo_infinite/film`.

### 1.1 Oracles et confrontations

| Élément | Ce qu'il mesure | Où | Réutilisable hors test ? | Rôle au banc |
|---|---|---|---|---|
| `killsource` origines `credit-concordant`, `source-victime`, `bot`, `tueur-bot`, `sans-revendication` | Concordance INTERNE au film entre dead-state et kill-feed (chunk HIGHLIGHT). Ce n'est pas une confrontation à l'API. | `F/internal/facts/killsource/kill.go:149-175`, `match.go:31-41`, `hybrid.go:142-166` | Oui : `decfilm.Kill`, `decfilm.Origin`, `Result`, `Coverage`, `PathStats.Ratio()`, `Result.LineByLinePublishable()` (`F/decfilm/decfilm.go:259-306`) | Source des kills par joueur (O-K4..6) et preuve interne (P-3) |
| `match_kill_events.read_origin` | Même vocabulaire, persisté par passe de production | `R/internal/migration/steps_shared_kill_events.go:256-340`, écrit par `sync/killcollector/collector_batch.go:73` | Base partagée : hors banc (le banc ne lit pas de base, et il doit lire la révision SOUS TEST, pas celle de production) | Aucun |
| `killsource_test.go` (`references`, `apiDeathsFilm`, `checkBotKillerDeaths`), `assist_test.go` (`multisetAssistances`, `TestAssistMultisetParJoueur`), `golden_test.go:66` | K/D/A du film contre l'API, par multiset | `F/internal/facts/killsource/*_test.go` | Non (`_test.go`, relevés figés à la main) | Modèle de la comparaison multiset (O-K6) |
| « A0.3 bombe » `a5Explosions` | 9 films, 28 explosions datées | `F/replay/assaut_a5_explosions_test.go:54-64` (+ 4 réutilisateurs), variable `ASSAUT_CACHE` | Non | Hors corpus du gate (films différents) ; méthode reprise par O-S3 |
| « Oracle 8 » `oracle8` / `checkAgainstOracle8` | Actions d'objectif nommées contre `match_objective_stats_latest` (2 films) | `F/internal/facts/objectives/named_test.go:37-118` | Non | Modèle d'O-S4 |
| `score_measure_oracle_test.go` (`loadOracle`, `readTSV`) | Courbe de score contre `match_registry` / `match_participants` (TSV) | `F/internal/facts/objectives/` | Non | Aucun (O-S1 le couvre sur l'artefact) |
| ~327 `_test.go` citant `oracle` sous `F/` (grammar 191, replay 105, objectives 20, killsource 4…) | Oracles de largeur de bits, de relevés Theater, de variables d'environnement (`SKULL_ORACLE`, `ODDBALL_ORACLE`, `VIP_ORACLE`…) | `F/**` | Non : logique en test, données souvent hors dépôt | Aucun ; rien à factoriser |
| `F/replay/testdata/equivalence/*.facts.json` (20) | `MatchFacts` figés | testdata | Oui (données) | Données des tests du banc (§8) |
| `SlotIdentityFrom` et variantes (`slotidentity_deaths.go:89`, `_rounds.go:289`, `_residue.go:57`, `_elimination.go:85`) | Ponts d'identité statborg → xuid | `F/internal/facts/objectives/` | Interne à `F` | **Source de circularité** : lire la méthode publiée |
| `resolveTeamIdentity`, `identityByFinalScore` (a), `identityByOneSidedScore` (a0), `identityByFrags` (b) | Rattachement des slots d'équipe aux camps | `F/replay/score_team_identity.go:40-138` | Non exporté ; publié en `coverage.score.teamIdentity` | **Source de circularité** pour O-S1 |
| `teamPublication.controler` → `coverage.teams.{accord, contradiction, silence}` | Équipe lue dans le film contre `MatchFacts.Players[].TeamID` | `F/replay/player_teams.go:280-293` | Résultat publié | Oracle non circulaire déjà publié (O-T1) |
| `replaydiff.mesurerJoueursScore`, `derniereValeur`, final d'équipe | Extraction K/D/A/score de fin de match par xuid et par camp | `R/internal/replaydiff/empreinte_axes.go:219,245,269` | Non exportées (`Empreindre` l'est) | À exporter ou à réutiliser, pour ne pas écrire une 2e extraction (règle n° 6) |
| `internal/analysis` | KPI produit (KDA, perf, citations) | `R/internal/analysis/` | Oui | Aucun oracle de film ; rien à réutiliser |

### 1.2 Vraisemblance et preuves internes

| Élément | Ce qu'il mesure | Où, seuil | Réutilisable sur le JSON publié ? | Rôle au banc |
|---|---|---|---|---|
| Catalogue AABB `map_quant_bounds.json` | Bornes de quantification par carte | `F/internal/profile/map_bounds.go`, via `decfilm` (l.430-439) | Oui, mais **inutile** : tautologique. Essai : 0 point hors AABB sur les 15 cartes trouvées (z de -946 à +242 sur Frost) | Rejeté comme oracle d'emprise |
| Emprise jouée (p1..p99 ± 12 étendues) + continuité | Positions de décodage aberrantes, ÉCARTÉES avant publication | `F/replay/geometry.go:140-176`, `emprise_jouee.go:47-94` (`continuiteVitesseMaxMPS = 60`), garde « une seule écriture » `emprise_jouee_test.go:19` | Non (non exporté, sur positions brutes) ; seuls les COMPTEURS sont publiés (`tracks.horsEmprise`, `vehicles.echantillonsHorsEmprise`, `spawnsHorsEmprise`) | Compteurs lus tels quels (V-2) ; interdiction de recopier les constantes |
| `doc.bounds` | Enveloppe publiée des données | `F/replay/geometry.go:155` | Oui | Référence d'emprise pour les OBJETS (V-2) |
| Filtre de vitesse et de téléportation | `DefaultMaxSpeedMPS = 100`, `maxRejectStreak = 3`, exemption ±200 ms d'une translocation | `F/internal/grammar/offline_biped.go:166-174`, `offline_filters.go` | Non (interne à `F`), et rien n'est compté dans l'artefact | Seuil de V-1 justifié par ce plafond ; `doc.translocations` pour l'exemption |
| Porte de présence du porteur | Portage de crâne ou de bombe hors de toute vie nommée | `F/replay/carrier_presence.go:60-164` | Non exporté mais réimplémentable, car il ne lit que des champs publiés ; résultat publié (`skullCarries.carrierAbsent`, `bombCarries.carrierAbsent`) | V-5, étendu aux drapeaux et aux ramassages (non couverts aujourd'hui) |
| `coverage.verdict.{shots, grenades, objectives, bridge}` | Rattachement ≥ 0,66, comptage équilibré, pont d'identité | `F/replay/coverage_bridge.go:202-279` | Résultat publié ; `Balanced()` exportés (`coverage_layer.go:58` et 5 autres) | P-4 (verdicts non nominaux) |
| `bridge.{concordant, discordant}` | Lecture directe contre pont par morts : deux méthodes d'identité, un désaccord = une des deux fausse | `F/replay/coverage_bridge.go` | Publié | V-7 |
| `keyframes.{refutations, contradictoryProofs}` | Preuves contradictoires de la marche d'image-clé | `F/replay/coverage_keyframes.go:21-62` | Publié | P-2 |
| `continuousFire.{packets, reached, closed}` | Paquets dont la vue C finit sur son terminateur (fermeture au bit près) | `F/internal/grammar/tir_continu.go:108-120`, `F/replay/fire_bursts.go:392` | Publié | P-1 (oracle interne de lecture) |
| `grammar.FrameClosure` / `cmd_fermeture` | Fermeture par vue, par archétype, records utiles | `F/internal/grammar/frame_closure.go:165`, `F/research/cmd_fermeture` (tag `research`) | Instrument ; décode le film | Option P-1bis (décision D-4) |
| `replaydiff.PolariteDe` | Polarité des 505 feuilles (176 échec, 170 succès, 157 neutres, 2 télémétrie) | `R/internal/replaydiff/polarite.go:143`, `polarite_table.go` | Oui, exportée | Reste la base de l'affichage informatif |
| `coverage.fallbacks[{name,hits}]` | Replis déclenchés ; chaque repli DÉCIDE un fait (champ `Fait` du registre) | `F/internal/facts/fallback/repli.go:206`, 116 noms | Publié ; registre via `decfilm.Table()` | R-1 (replis qui décident) |

**Aucun contrôle ne vérifie aujourd'hui** :
- que deux pistes d'un même xuid ne se recouvrent pas ;
- qu'un ramassage vient d'un joueur présent ;
- qu'un trajet en véhicule garde le joueur près du véhicule ;
- qu'un kill ou un portage vient d'un joueur vivant.

Ce sont les apports propres du banc.

---

## 2. Ce que le gate exporte, et ce qui manque

`levelup replay-facts-export` (`R/cmd/levelup/cmd_replay_facts_export.go`) écrit `replaybuild.FactsFile`
(`R/internal/replaybuild/facts_file.go:36-46`) = `domain.MatchFacts` à plat + `matchId` + `mapNames`.
Source : `match_registry` (`team_0_score`, `team_1_score`, `game_variant_name`, `map_id`) et
`match_participants` (`R/internal/platform/duckdb/replay_facts_repo.go:82,109`).

| Champ exporté | Usage au banc |
|---|---|
| `players[].{xuid, kills, deaths, assists, teamId}` | O-K1..6, O-V1, O-T1 |
| `players[].{joinedInProgress, leftInProgress}` | Contexte de O-V1 (vies attendues) |
| `players[].{joinMatchMs, leaveMatchMs}` | **INTERDIT comme oracle daté** : pas sur l'horloge du film, écart jusqu'à 25 s (plan §8.30). De plus, c'est une entrée de la cuisson (fermetures par relais), donc circulaire. |
| `teamScores` | O-S1 (si `teamIdentity` ∉ {a, a0}), O-S2 |
| `gameVariantName`, `mapId`, `mapNames` | Famille d'objectif (`objectives.ObjectiveTypeOf`), clé de carte |

**Manques (tous présents en base, aucun dans `facts.json`)** :

| Manque | Table | Oracle visé | Remarque |
|---|---|---|---|
| Score personnel | `match_participants.personal_score` et `.score` | O-S3 | Vérifier en phase 2 laquelle des deux colonnes égale le score statborg « affiché » |
| Stats d'objectif par joueur | `match_objective_stats_latest` (CTF, Zones, Oddball, Stockpile, Extraction, VIP : `persist/rows.go:339-389`) | O-S4, O-S5 | Pas de bloc Assaut : la bombe n'a pas d'oracle par joueur |
| Tirs | `match_participants.shots_fired`, `.shots_hit` | O-X1 | Vérifier si l'API compte les armes de véhicule |
| Kills par catégorie | `headshot_kills`, `melee_kills`, `grenade_kills`, `power_weapon_kills` | O-K7 (plus tard) | Rapprochement avec la classe de dégât de killsource ; `killscope.IsHeadshotCategory` cite déjà 99,3 % |
| Présence booléenne | `present_at_beginning`, `present_at_completion` | O-P1 (informatif) | Non daté, donc insensible au décalage d'horloge |
| Médailles | `medals_earned` | aucun | Le film ne les publie pas dans l'artefact |
| Score par manche | aucune table | aucun | Pas d'oracle officiel par manche : **non retenu** |

**Recommandation (D-2)** : ne PAS étendre `FactsFile`, qui est une entrée de cuisson (clé de cuisson J3.5,
harnais d'équivalence). Ajouter un second fichier `<short8>.oracle.json`, écrit par la même
sous-commande (`replay-facts-export --oracle`) en une seule ouverture RO, et lu par le seul banc. Les
entrées de cuisson restent ainsi identiques à l'octet.

---

## 3. Où vit le banc (couches ADR 0034)

**Paquet proposé : `R/internal/replayverite/`**, au même étage que `internal/replaydiff`. Hors de `film/`,
parce qu'il ne lit ni octets ni faits internes. Il lit :
- (a) le JSON publié ;
- (b) `domain.MatchFacts` + `oracle.json` ;
- (c) en option, un registre de kills que le gate lui passe (§4.1, O-K4..6).

Pur : aucune E/S, aucune base, aucun `film/internal`.

- **Il ne désérialise PAS en `replay.ReplayDocument`.** Il utilise ses propres structures minimales
  (étiquettes JSON des seuls champs lus), sur le modèle de `replaydiff`, qui lit un `map[string]any`.
  Deux raisons :
  - un oracle ne partage pas les types du code qu'il juge ;
  - le ratchet `archlint/film_facade_surface_test.go` compte 257 identifiants `replay.X` hors de `film/` :
    importer le document le ferait monter.
- **Garde contre la dérive de forme** : un test lit `F/replay/testdata/document_shape.golden` (le fichier
  texte, sans import) et échoue si une clé lue par le banc n'y figure plus. Un renommage côté producteur
  casse donc le banc bruyamment, jamais en silence.
- **Extraction K/D/A de fin de match** : exporter `replaydiff.mesurerJoueursScore` / `derniereValeur`
  (ou les déplacer dans un sous-paquet commun), plutôt qu'en écrire une deuxième (règle n° 6).
- **Title-agnostic** : le banc ne compare aucun slug. La famille d'objectif vient de la variante, par la
  fonction existante. Les stats d'objectif sont une table `stat publiée → colonne oracle` déclarée en
  donnée : TOML de mappings du titre, à trancher en phase 2.

---

## 4. Les scores (oracles)

Notation par score et par témoin : **VP / FP / FN**, cumulés sur les joueurs. Par joueur,
`VP = min(pub, off)`, `FP = max(0, pub − off)`, `FN = max(0, off − pub)`. Un joueur de l'API sans valeur
publiée compte tout en FN. Un xuid publié absent de l'API compte tout en FP.

Unité : l'événement (kill, mort, assistance, action), le point, ou la seconde.

Sens : FP et FN ne doivent jamais monter. On affiche aussi `précision = VP/(VP+FP)` et
`rappel = VP/(VP+FN)`.

### 4.1 Kills, morts, assistances

| Id | Définition | Oracle | Circularité, limites |
|---|---|---|---|
| **O-K1/2/3** | K, D et A finaux de `scoreTimeline.players[]` (dernier point de `total`), par xuid | `players[].{kills, deaths, assists}` | Exclure les xuid dont TOUS les slots statborg sont nommés `triplet_feuille` (circulaires) ; les compter à part en « nommés par l'oracle ». Limites : le statborg n'est lu que pour 8 slots en BTB (FN massif et stable, c'est un manque et non un faux) ; les courbes multi-manches sont liées à une manche (§8.30), d'où des FN sur `c75f33b8` |
| **O-K4** | Kills par tueur, depuis la section 5 des faits du film (`Kill.Feed.Killer`, nom → xuid par `identity.players[]` de l'artefact) | `players[].kills` | Non circulaire : le kill-feed vient du film, et ses noms passent par les tables du film. Les kills `bot` et `tueur-bot` suivent les règles du paquet (`Feed.Present`). Exige D-3 |
| **O-K5** | Morts par victime (`Kill.Victim` + `UnclaimedDeaths`) | `players[].deaths` | Idem |
| **O-K6** | Assistances (`Kill.Assist`, avec `assist_known`) | `players[].assists` | Une assistance inconnue n'est pas « pas d'assistant » : elle compte FN, jamais FP |
| **O-V1** | Morts DÉDUITES DES VIES : par xuid, `vies publiées − 1` (pistes `tracks[]` du xuid) | `players[].deaths` | Non circulaire : les vies sont nommées par `creation_bipede` / `film_table` / `PlayerIndexTable`. FP = coupure de vie sans mort ; FN = vies manquantes. Limites : un départ ou une arrivée en cours ajoute ou retire une vie ; `joined/leftInProgress` sert de tolérance de ±1 |
| **O-V2** | Même mesure, SANS identité : Σ fins de vie avant la fin du film, contre Σ morts de l'API | Σ `deaths` | Insensible aux erreurs de nommage. Isole les vies coupées ou manquantes |

### 4.2 Scores

| Id | Définition | Oracle | Circularité, limites |
|---|---|---|---|
| **O-S1** | Score final par camp (`scoreTimeline.teams[].total`) | `teamScores` | Circulaire si `coverage.score.teamIdentity ∈ {a, a0}` : alors NON NOTÉ, compté « rattaché par l'oracle ». Testable si `b`. `unresolved` = FN du camp. `coverage.score.oracle = displayed` : en Strongholds et KOTH, l'API ne compte pas la même chose (`document_score.go:26-38`), donc on applique la table par mode de l'ADR 0032 |
| **O-S2** | Actions de marque par camp contre le score officiel, par famille : CTF Σ `flag_captures` ; Assaut Σ `bombStats.detonations` (et les explosions datées) | `teamScores` | Non circulaire (actions nommées par les morts, camp par le roster). Table famille → action en donnée |
| **O-S3** | Score personnel final (`scoreTimeline.players[].score`) | `personal_score` ou `score` (manque, D-2) | Non circulaire (le triplet n'utilise pas le score) |
| **O-S4** | Actions d'objectif par (xuid, stat) (`objectives[]`, hors `kills`/`assists` déjà vus en O-K) | `match_objective_stats_latest` (manque, D-2) | Non circulaire à identité donnée. Modèle : `oracle8` |
| **O-S5** | Durées de portage par xuid (`flagCarries`, `skullCarries`, `bombCarries`) | `time_as_flag_carrier_seconds`, `time_as_skull_carrier_seconds` | Tolérance à mesurer en phase 2 (ex. 1 s + 2 %) : les portages se bornent à l'image. Pas d'oracle pour la bombe |
| **O-T1** | Équipe du roster contre `teamId` | `coverage.teams.{accord, contradiction, silence}` (déjà publié) | FP = `contradiction`, FN = `silence` |
| **O-X1** | Tirs par xuid (`shots[]` via les vies) | `shots_fired` (manque) | FP = tirs au-delà du compte officiel. À valider (armes de véhicule, rafales) |
| O-P1 (info) | Présent au début ou à la fin (`roster[].presence`) | `present_at_beginning` / `present_at_completion` | Informatif seulement : la présence est aussi alimentée par les faits (relais) |

### 4.3 Preuves internes de lecture (pas d'oracle externe)

| Id | Mesure | Sens |
|---|---|---|
| **P-1** | `continuousFire.closed / packets` : paquets fermés au bit près (vue C) | Ne baisse pas |
| P-1bis (option D-4) | Records utiles fermés (`FrameClosure`) | Ne baisse pas |
| **P-2** | `keyframes.contradictoryProofs`, `keyframes.refutations` | Ne monte pas |
| **P-3** | Part `credit-concordant` des kills (O-K4..6 disponibles) | Ne baisse pas |
| **P-4** | `coverage.verdict.*` différent de `nominal` | Aucun verdict ne se dégrade (ordre : nominal > partiel > non publiable) |

---

## 5. Les classes de violations (vraisemblance : elles détectent des FAUX)

Unité : l'instance (identifiée pour l'affichage : slot, image, xuid). Règle : aucun compte ne monte.

| Id | Définition | Seuil et justification |
|---|---|---|
| **V-1 Saut** | Deux points consécutifs d'une piste à plus de **100 m/s** (distance 3D / Δt), hors translocation publiée (±2 images) et hors **porte de carte** : destination (cellule de 2 m) partagée par au moins 3 vies distinctes (ascenseur, canon, largage). Les portes sont affichées à part | 100 m/s = plafond du filtre de la grammaire (`DefaultMaxSpeedMPS`). Un Spartan reste sous 35 m/s, un véhicule sous 26,1 m/s : au-delà, c'est un réancrage (`maxRejectStreak`) ou un faux. Essai : `11de8353` 47 sauts → 6 portes, **5 résiduels** ; `d9781168` 38 → 2 portes, 0 |
| **V-2 Objet hors emprise** | Arme au sol, pose d'équipement, présentoir, échantillon de véhicule, tir, grenade ou projectile publié hors de `doc.bounds` + marge | Marge à calibrer en phase 2 (proposé : 5 m). L'AABB du catalogue est rejeté (tautologique) et les pistes sont déjà filtrées. S'y ajoutent les compteurs publiés de rejet (`tracks.horsEmprise`, `vehicles.echantillonsHorsEmprise/spawnsHorsEmprise`). Exemple : arme au sol à (116, −175) sur Curfew (enveloppe x ∈ [−35, 1,5]) |
| **V-3 Action hors vie** | Tir, grenade, ramassage, changement d'arme, capacité ou changement d'équipement dont le slot n'a aucune vie qui couvre `t` (±5 images) | 5 images = 500 ms : datation au pas d'image et latence de publication. Essai : la tolérance 0 → 5 fait passer les tirs de 279 à 203 sur 42 735 ; les grenades restent à 285/2 784 (dont 55/55 sur `50247b26` et 151/282 sur `a349fea8`, builds anciens) |
| **V-4 Deux corps** | Deux pistes d'un même xuid qui se recouvrent dans le temps | Zéro tolérance : un joueur n'a qu'un corps. Essai : `084a804d` **3** (866 images), apparus à la fusion J5 |
| **V-5 Acteur absent ou mort** | Ramassage, action d'objectif ou portage (drapeau, crâne, bombe) par un xuid hors de sa présence au roster (`from` à `toMax`), ou hors de toute vie nommée (±5 images) | Même porte que `carrier_presence.gate`, étendue aux drapeaux, aux ramassages et aux objectifs. Essai : 1 action d'objectif hors présence (`bcb6d393`), 0 portage |
| **V-6 Trajet loin du véhicule** | Échantillon de véhicule pendant un trajet dont le passager, à la même image, est à plus de **3 m** du véhicule (plus de 10 m compté à part) | 3 m = `vehicleEventAnchorRadiusM`, dernier rayon « gratuit » mesuré. Essai : `084a804d` 4 (apparus à J5), `4f77afc1` 1, `a349fea8` 1 ; 0 au-delà de 10 m |
| **V-7 Identité discordante** | `bridge.discordant` (lecture directe contre pont par morts) + `bridge.slotCollisions` + `seats.chevauchements` | Publiés, 0 attendu pour les deux derniers. Essai : `discordant` de 0 à 32 |
| **V-8 Mort sans fin de vie** (conditionnelle) | Mort de O-K5 sans fin de vie de la victime à ±1 s | Exige d'aligner le temps du fil sur les images. Si l'offset du pont n'est pas publié exploitable, reporté (décision D-3) |
| **R-1 Replis qui décident** | Σ `hits` de `coverage.fallbacks[]`, par nom | Chaque repli décide un fait (champ `Fait`). Voir D-6 pour la règle : un nom nouveau bloque toujours ; une hausse de hits est bloquante ou informative |

---

## 6. Sortie, verdict, avant/après

### 6.1 Format

- **Bibliothèque** : `replayverite.Noter(doc []byte, faits domain.MatchFacts, oracle *Oracle, kills *Registre) (Bulletin, error)`.
  - `Bulletin{Scores map[Id]VPFPFN, Preuves map[Id]int, Violations map[Id][]Instance, Exclus map[Id]int}`.
  - `Exclus` = unités non notées pour cause de circularité, affichées.
  - `replayverite.Comparer(avant, apres Bulletin) Comparaison` : pour chaque score le Δ de FP/FN et les
    joueurs qui bougent ; pour chaque classe les instances NOUVELLES et DISPARUES, appariées par clé
    stable (xuid, slot, image arrondie).
- **Gate** : le banc est appelé dans `traiterTemoin` (`orchestrate.go:77-105`), après `bakeTemoin`,
  sur `refPath` et `cuissonHead.ArtifactPath`, avec les mêmes faits.
  - Il ajoute `ligneRapport.Verite *replayverite.Comparaison`.
  - Il imprime une section « BANC DE VÉRITÉ » dans `finaliser` après `imprimerTelemetrie` (`main.go:368`) :
    une ligne par témoin et par score, `FP a→b  FN a→b`, puis la liste nominative des instances nouvelles.
  - Il ajoute au JSON le champ `verite` (omitempty) à `ligneJSON`, sans casser les fixtures
    `rapport_fixture_test.go` ni `testdata/*.json`.
- **Colonne au tableau** : `vérité` = `ok`, `FAUX(n)` (FP ou violation en hausse) ou `MANQUE(n)` (FN en
  hausse).

### 6.2 Avant/après

Les deux artefacts (base et HEAD) sont notés avec les MÊMES faits. Seul le delta décide ; les valeurs
absolues s'affichent. Une ligne de base légitime (portes de carte, statborg à 8 slots en BTB, builds
anciens) ne gêne donc pas le verdict tant qu'elle ne bouge pas.

### 6.3 Règle de verdict (proposée)

Statuts, par priorité :
1. `ABSENT` / `ERREUR` ;
2. **`FAUX`** : un FP d'oracle ou une classe de violation en hausse ;
3. **`MANQUE`** : un FN d'oracle en hausse ou une preuve interne en baisse ;
4. `ok`.

`FAUX` et `MANQUE` sont bloquants. Les pertes et changements `replaydiff` deviennent informatifs, sauf
deux filets qui restent bloquants, parce qu'aucun oracle ne les voit :
- **(a)** un calque qui DISPARAÎT (révision de `layers` absente, ou tableau de premier niveau qui passe à
  vide) ;
- **(b)** une perte `replaydiff` dans un bloc qu'AUCUN score ni aucune classe ne couvre, à lister en
  phase 2 par une table de couverture des blocs.

Exception nominative : une hausse justifiée (ex. un FP causé par un oracle faux) passe par une liste
datée et justifiée dans le manifeste, jamais par une allowlist nue.

---

## 7. Chiffres d'essai (sans décodage)

Artefacts de référence `b452391f7` (J6-ter, schéma 76, 19 témoins) et faits exportés du 2026-09-28.
Oracles O-K4..6, O-S3..5 et O-X1 : **non chiffrables en phase 1** (faits du film binaires, lecture Go
interdite pendant le passage en cours ; colonnes absentes de `facts.json`).

### 7.1 Trois témoins en détail

| Score | `c75f33b8` Assaut, 3 manches | `60ae07c4` Oddball classé, 8 j. | `084a804d` BTB CTF, 26 j. |
|---|---|---|---|
| Nommage statborg | 7 `instants_de_mort`, 1 `residu_de_manche`, 16 non résolus | 16 `instants_de_mort` | 8 `instants_de_mort` |
| O-K1 kills VP/FP/FN | 40/0/22 (exacts 3/14) | 162/0/0 (8/8) | 124/0/230 (8/26) |
| O-K2 morts | 48/0/14 | 163/0/0 | 117/0/239 |
| O-K3 assistances | 15/0/7 | 63/0/0 | 50/0/72 |
| O-V1 morts par les vies | 59/**9**/3 | 160/**1**/3 | 352/**3**/4 |
| O-V2 fins de vie / Σ morts API | 78 / 62 | 167 / 163 | 375 / 356 |
| O-S1 camps | `a0` : circulaire (0:–, 3:3) | `a` : circulaire | `a` : circulaire |
| O-S2 actions de marque | détonations 0 contre score 3 : **FN 3** | — | captures (non chiffrées) |
| O-T1 équipes | accord 10, contradiction 0 | 8 / 0 | 26 / 0 |
| P-1 paquets fermés | 22 943 / 27 800 | 13 967 / 49 696 | 5 239 / 32 457 |
| V-1 / V-3 tirs, grenades / V-4 / V-6 | 0 / 1, 3 / 0 / 0 | 1 (translocation exemptée) / 9, 6 / 0 / 0 | 0 / 89, 20 / **3** / **4** |
| V-7 `bridge.discordant` | 10 | 12 | 32 |

Les K/D/A statborg n'ont **aucun FP sur 19 témoins**. Leur FN est un manque de nommage ou de manche,
jamais un faux. Les FP apparaissent sur les VIES (O-V1 : 27 au total), et c'est là que se trouvent les
faux de ce corpus.

### 7.2 Totaux sur les 19 témoins (`b452391f7`)

- O-K1 kills 1 439 / 0 / 1 337 ; O-K2 morts 1 396 / 0 / 1 409 ; O-K3 assistances 549 / 0 / 516.
- O-V1 morts par les vies 2 289 / **27** / 516.
- O-V2 : fins de vie > Σ morts sur tous les témoins. Écart extrême sur `50247b26` (243 contre 154) et
  `a349fea8` (409 contre 293) : builds anciens sans section d'identification, pistes sans xuid, pont nul.
- O-S1 : 15/19 circulaires ; 4 `unresolved` (`4f77afc1`, `50247b26`, `a349fea8`, `a521164d`) = FN.
- O-S2 : CTF `bcb6d393` captures par camp [3, 0] contre [3, 0] ✓, `fb1a1a72` [0, 1] contre [0, 1] ✓ ;
  Assaut `c75f33b8` 0 détonation contre 3 points = FN 3.
- V-3 hors vie (tolérance 0 / 5 images, sur l'ensemble) :
  - tirs 279 / 203 sur 42 735 ;
  - grenades 305 / 285 sur 2 784 ;
  - ramassages 3 / 3 sur 4 723 ;
  - changements d'arme 13 / 13 sur 1 537 ;
  - capacités 29 / 29 sur 2 050.
- V-4 : 3 (`084a804d`). V-5 : 1. V-6 (> 3 m) : 6.
- P-1 fermeture : à la révision J4 (`8f89aeedc`), `continuousFire.closed / packets` égale la carte J4.0.5
  (ex. `11de8353` 5 738 / 17 629, HI_1_9_0 ; `50247b26` 153 / 17 919, version-31). À `b452391f7` :
  5 741 / 17 629.

### 7.3 Avant/après sur des fusions réelles (validation du principe)

- **`8f89aeedc` (J4) → `d6701c058` (J5+J6+J9), `084a804d`** :
  - O-K1 FN 240 → 230 ; O-K2 exacts 0 → 8/26 ; O-S1 camp 0 : 2 → 3 (= API) ;
  - O-V1 FN 125 → 4 **mais FP 0 → 3** ;
  - V-3 : ramassages 167 → 0, grenades 62 → 20, tirs 169 → 89 ;
  - V-5 ramassages hors présence 48 → 0 ;
  - **V-4 0 → 3, V-6 0 → 4**.
  - Verdict banc : `FAUX(10)`, à attribuer instance par instance, avec un gain net massif affiché à côté.
  - Sur `4f77afc1` : O-V1 FN 81 → 7 ; V-3 ramassages 55 → 0 ; V-6 (> 10 m) 1 → 0.
- **`d6701c058` → `40484c0e3` (J8) → `2a66eefa4` (J10) → `c5e8e7b3e` (J7) → `876312171` (J6-bis) →
  `b452391f7` (J6-ter), 19 témoins.** Sur cette chaîne, aucun score O-K, O-V ou O-S ni aucune classe
  V-1 à V-6 ne bouge. Seuls varient le compte de tirs publiés et d'échantillons de trajet de `4f77afc1`,
  qui ne sont pas des violations. Le banc aurait néanmoins signalé deux choses, et les deux sont des
  signaux réels :
  - **P-1 sur J6-bis → `MANQUE`** :
    - paquets fermés en baisse sur 6 témoins : `d9781168` −9, `51ebbc0f` −4, `60ae07c4` −4, `a349fea8` −4,
      `4f77afc1` −3, `111fa685` −1 ;
    - en hausse sur 6 autres (`11de8353` +3, `084a804d` +2, `fb1a1a72` +2, `bf15f7ab` +1, `c75f33b8` +1,
      `f75e7053` +1) ;
    - ce sont exactement les « fermetures gagnées par J6 et rendues » du journal du plan.
  - **R-1 sur J8** : les hits de replis passent de centaines à des dizaines de milliers, parce que J8
    CÂBLE des compteurs qui n'existaient pas. Ce n'est pas une décision nouvelle. D'où une règle
    nécessaire (D-6) : un repli dont le compteur vient d'être branché (`CompteurBranche` faux → vrai au
    registre) n'est pas une hausse. Sur J7, R-1 monte de 17 hits et d'un nom sur `4f77afc1`, à attribuer.

  Sur tout le reste, le banc rend `ok` là où chaque passage a demandé une attribution manuelle.

---

## 8. Stratégie de tests (phase 2)

1. **Tests purs sur artefacts synthétiques** (JSON littéraux minimaux), un par score et par classe, chacun
   ROUGE d'abord. Chaque classe a son test de mutation : décaler un point de 20 m sur une image (V-1),
   ajouter un tir hors vie (V-3), dupliquer une piste (V-4), éloigner un passager (V-6).
2. **Circularité** : un slot `triplet_feuille` est exclu et compté dans `Exclus` ; `teamIdentity = a`
   rend O-S1 « non noté ».
3. **Garde de forme** : chaque clé lue existe dans `document_shape.golden`.
4. **Artefact réel en testdata** : `c75f33b8` (1,5 Mo) et `60ae07c4`, gzippés (environ 200 Ko), avec leurs
   faits de `F/replay/testdata/equivalence/`. Golden du bulletin, régénéré par la même porte que les
   autres goldens.
5. **Gate** : sur le modèle de `rapport_fixture_test.go`, avant/après synthétique (un FP qui monte →
   `FAUX`, code de sortie 1 ; un FN qui baisse → `ok`).
6. Archlint : aucun import de `film/internal` ni de `film/replay` par `replayverite` (ratchet nommé).

---

## 9. Décisions recommandées au superviseur

| Id | Décision | Recommandation |
|---|---|---|
| D-1 | Paquet et types | `internal/replayverite`, structures JSON propres, pas d'import de `film/replay` ; exporter l'extracteur de fin de match de `replaydiff` |
| D-2 | Oracles manquants | Fichier séparé `<short8>.oracle.json` (`replay-facts-export --oracle`) : `personal_score`, `score`, `shots_fired/hit`, stats d'objectif, présence booléenne. `FactsFile` inchangé |
| D-3 | Kills individuels | Le gate lit la section 5 des faits du film de CHAQUE cuisson (base et HEAD) et passe un registre minimal au banc. Coût : ratchet `replay.X` hors `film/` de +1 ou +2 (`DecodeFilmFactsFile`), date et justification dans le ratchet. Attention : le chantier « cache de la base » (`-gatecache`) doit mettre en cache le fichier de faits AVEC l'artefact de base, sinon O-K4..6 sera absent côté base. Découpage proposé : phase 2a sans O-K4..6 et sans V-8, phase 2b avec |
| D-4 | Fermeture | P-1 depuis `continuousFire` suffit au paquet. Les records utiles (P-1bis) coûtent une passe `FrameClosure` par film et par révision (15 à 43 s par film, 2 × 19 films). Recommandation : hors gate, à l'entrée des jalons de grammaire seulement |
| D-5 | Verdict | Le banc devient LE verdict (`FAUX` / `MANQUE`) ; `replaydiff` passe en information, avec les deux filets du §6.3. Pas de drapeau de bascule (règle 11) : livré actif |
| D-6 | Replis (R-1) | Nom nouveau = bloquant, sauf si le registre montre un compteur nouvellement branché (cas J8, §7.3) ; hausse de hits = informative tant qu'aucun dénominateur par repli n'existe. Alternative stricte : toute hausse bloque |
| D-7 | Seuils | V-1 100 m/s et porte de carte à 3 vies ; V-3 et V-5 ±5 images ; V-6 3 m ; V-2 marge 5 m (à calibrer). Chacun en constante nommée avec sa mesure en commentaire |
| D-8 | Ligne de base connue | O-S1 circulaire sur 15/19 : pour rendre l'oracle d'équipe testable, préférer la méthode `b` quand elle est possible, ou publier les deux. **Hors banc** : à consigner comme découverte, pas à traiter ici |

## 10. Découvertes (consignées, non traitées)

1. `c75f33b8` (Assaut) : `bombStats.detonations = 0` alors que le camp 1 marque 3 points (4 armements
   publiés). Soit les détonations ne sont pas lues, soit le score d'Assaut ne vient pas des détonations.
2. `084a804d` : 3 recouvrements de pistes d'un même xuid (866 images au total) et 4 échantillons de trajet
   à plus de 3 m du véhicule, apparus à la fusion J5+J6+J9 (`d6701c058`).
3. Oddball (`51ebbc0f`, `d9781168`) : plus de vies que de morts + 1 pour plusieurs joueurs (O-V1 FP 6 et
   8 à `b452391f7`), sans fin de vie simultanée. Cause à instruire (manche, porteur de crâne ?).
4. `396cfc92` : 6 085 points de piste sans `z`. Forme légale du document, mais tout consommateur doit
   traiter `z` comme optionnel.
5. Les courbes statborg ne couvrent que 8 joueurs en BTB (tous les BTB du corpus). C'est le premier
   gisement de FN sur O-K1..3.

### 10.1 Faux et manques du corpus actuel, mesurés par le banc livré (phase 2a)

Mesure du 2026-09-30 : `cmd/replay-verite` sur les 19 artefacts `b452391f7` (J6-ter, schéma 76), chacun
contre lui-même, valeurs absolues. Consignés sur instruction du superviseur, **aucun n'est corrigé**. Le
verdict du banc ne les juge pas : il ne juge que leur évolution.

6. **O-V1, vies en trop (28 FP)** : `c75f33b8` 9, `d9781168` 8, `51ebbc0f` 6, `084a804d` 3, `4f77afc1` 1,
   `60ae07c4` 1. Le détail par joueur (publié / officiel) est dans le rapport de l'outil. Le chiffre
   d'essai de la phase 1 (27) venait du script jetable ; celui-ci fait foi.
7. **O-V2, fins de vie au-delà des morts officielles** (FP sans identité) : 18 sur `084a804d`, 27 sur
   `4f77afc1`, 89 sur `50247b26`, 114 sur `a349fea8`, 17 sur `a521164d`, 14 sur `c75f33b8` et
   `d9781168`, de 1 à 11 ailleurs. Ce sont des vies coupées sans mort, ou des départs et fins de manche :
   la cause n'est pas instruite.
8. **V-4, deux corps** : `084a804d`, 3 recouvrements (xuid `2533274851740446` 1 image, `2535442829120831`
   9 images, `2535464635796745` 208 images), apparus à la fusion J5+J6+J9.
9. **V-3, actions hors vie** : 255 sur 19 témoins, dont 114 sur `4f77afc1` et 93 sur `084a804d`.
   **CORRECTION de la phase 1** : les « grenades hors vie des vieux builds » (285) étaient presque toutes
   au slot 0, qui veut dire « pont muet », le lanceur est inconnu (`film/replay/grenades.go`). Ce ne sont
   pas des faux et le banc livré ne les juge pas. Restent 17 actions sur `a349fea8` et 4 sur `50247b26`.
10. **V-6, passagers à plus de 3 m du véhicule** : 10 (`084a804d` 6, `4f77afc1` 3, `a349fea8` 1). Le
    chiffre de phase 1 (6) appariait à l'image exacte ; le banc apparie à ±1 image.
11. **V-8, morts du statborg sans fin de vie à ±2 images** : 62, dont 20 sur `4f77afc1` et 13 sur
    `084a804d`.
12. **V-2, objets hors emprise à plus de 10 m** : 899 sur `50247b26`, 430 sur `a521164d`, 535 sur
    `e5adf7b2` (surtout des échantillons de véhicule sur BTB et les builds anciens), 10 sur `084a804d` et
    `4f77afc1`. Sur les cartes Arena, ce sont des armes `spawned` jamais ramassées, à plus de 100 m.
    Est-ce un rangement hors carte réel ou un faux ? Non instruit.
13. **V-5** : 1 action d'objectif `kills` hors présence (`bcb6d393`, xuid `2535468064146356`, image 578).
14. **O-S2** : sur `084a804d` (BTB CTF), aucune capture publiée contre un score de 3 à 2 (FN 5) : le
    calque `objectives` n'est pas publié sur ce témoin. Sur `c75f33b8`, 0 détonation contre 3 (FN 3,
    découverte 1).
15. **V-7** : `084a804d` passe de 22 à 146 désaccords d'identité publiés à la fusion J5+J6+J9.
16. **J5 sur `4f77afc1`** : O-T1 (contradiction d'équipe) monte, et le verdict `bridge` de P-4 se dégrade.

## 11. Phase 2a — livré, et écarts à la conception

Livré sur `feat/suite-audit-decodeur-verite` :
- `internal/replayverite` : scores O-K1..3, O-V1, O-V2, O-S1, O-S2, O-T1 ; preuves P-1, P-2, P-4 ;
  violations V-1 à V-8 ; R-1 ; verdict `FAUX` / `MANQUE` / `ok` ; rendu avant/après ;
- l'outil `cmd/replay-verite`, qui prend deux artefacts, des faits et en option le registre des replis
  d'avant, et sait écrire ce registre ;
- l'oracle officiel : `domain.MatchOracle`, `port.ReplayOracleRepo`, `duckdb.ReplayOracleRepo`, et
  `levelup replay-facts-export --oracle` qui écrit `<short8>.oracle.json` en lecture seule
  (`OpenReadForQuery`).

Décisions superviseur appliquées : D-1 à D-8 ; les absolus sont informatifs ; chaque seuil est une
constante nommée avec sa mesure (`seuils.go`) ; les faux du corpus sont consignés (§10.1).

Écarts, chacun motivé :
- **P-3 n'est pas livré.** Il exige les kills individuels (section 5 des faits du film), qui relèvent de la
  phase 2b (D-3).
- **V-8 n'attend pas la 2b.** Au lieu des morts de killsource, il lit les morts du statborg (paliers de la
  courbe `deaths`). Mesure de la tolérance : sur 1 344 morts statborg, 982 tombent à l'image même de la
  fin de vie, 304 à +1 et 2 à +2, d'où ±2 images.
- **Exemption de V-1.** Aucune donnée de carte ne positionne ascenseurs, canons ou largages :
  `map_objectives.json` n'a que des noms de script sans coordonnées, `map_callouts.json` des zones
  d'appel, `map_positions_jouees.json` une seule carte. L'exemption est donc une **liste nommée et
  datée** (`portes_de_carte.go`) : 5 portes mesurées sur Thunderhead et Dredge, rayon 3 m au départ ou à
  l'arrivée du saut, avec un critère de retrait. Sa mesure : `11de8353` 47 sauts, tous exemptés ;
  `d9781168` 38 sauts, tous exemptés.
- **V-2 restreint.** Seuls les objets posés et les échantillons de véhicule sont jugés, avec une marge de
  10 m et non 5. Les tirs, grenades et projectiles volent au-delà de l'enveloppe des joueurs : des
  centaines par témoin BTB, même à 20 m.
- **V-3 ne juge pas le slot 0** (pont muet, voir §10.1 n° 9).
- **O-V1 sans tolérance de ±1** pour les arrivées et départs : seul le delta décide, et une tolérance
  aurait masqué une vie coupée.
- **Extracteur de fin de match.** `replaydiff` n'a pas été exporté, contrairement à D-1 : il travaille sur
  un `map[string]any`, le banc sur des types. `Serie.Finale` en est la **deuxième** copie (plafond de la
  règle n° 6), aucune troisième n'est admise.
- **Oracle exporté sans consommateur dans le banc.** Les scores qui le lisent (O-S3, O-S4, O-S5, O-X1)
  n'étaient pas dans la liste de la phase 2a : l'export existe, son lecteur reste à décider.

Preuves :
- tests purs sur artefacts synthétiques, et un artefact réel élagué (`bcb6d393`, 169 Ko compressés),
  bulletin figé ;
- **30 mutations par copie, toutes rouges**, au moins une par score, par preuve, par classe, par
  exemption et par règle de verdict ;
- gates : `go build ./...`, `go vet ./...`, tests du paquet et de l'outil, test d'intégration du dépôt
  d'oracle (`-tags=integration`), `internal/archlint`, golangci-lint des paquets touchés (0 constat
  nouveau).

Verdicts du banc sur la chaîne réelle (outil livré, artefacts `j11`, sans registre d'avant) :
- **J4 → J5 (`8f89aeedc` → `d6701c058`)** : `FAUX` sur `084a804d` (O-V1, O-V2, V-4, V-6, V-7), sur
  `4f77afc1` (O-T1, O-V1, O-V2, P-4 `bridge`, V-7) et sur `a349fea8` (O-V2) ; `MANQUE` (P-1) sur
  `0797ce72`, `51ebbc0f` et `c75f33b8`.
- **J8** : `FAUX` partout, de 16 à 26 replis « nouveaux » par témoin. C'est le cas du compteur
  nouvellement branché, que seul le registre d'avant excuse. Aucune révision de J8 ne porte l'outil qui
  l'écrit : ce cas n'est prouvé que par le test synthétique.
- **J10** : `ok` partout.
- **J7** : `FAUX` sur 4 témoins, un ou deux replis nouveaux (à attribuer : branchés ou nouveaux ?).
- **J6-bis** : `MANQUE` (P-1) sur 6 témoins, et un repli nouveau (`repli_emission_hors_domaine_jetee`)
  sur 9.
- **J6-ter** : `ok` partout.

## 12. Intégration au gate de corpus (2026-09-30, après fusion du cache de la base `3f644931a`)

- **Verdict.** `cmd/replay-corpus-gate/verite.go` note chaque témoin comparé : la référence et le
  HEAD, avec les mêmes faits et le même oracle. Le statut suit cette priorité : `ABSENT`/`ERREUR`,
  puis `FAUX`, `MANQUE`, `PERTE` (filets), `ok`. `CHANGEMENT` n'est plus un statut. Si le banc ne
  peut pas lire un artefact, le témoin est en `ERREUR` (code 3).
- **Filets.** Deux cas bloquent encore :
  - une perte `replaydiff` hors des blocs couverts. La table `blocsCouverts` rattache chaque bloc à
    la mesure qui le couvre, et un bloc n'est couvert que si cette mesure est NOTÉE des deux côtés.
    Un O-S1 circulaire ne couvre rien ; sans oracle, O-S3 ne couvre pas le score personnel ;
  - un calque de premier niveau qui passe à zéro ou disparaît, même dans un bloc couvert.

  Sans banc, aucun bloc n'est couvert.
- **Registre R-1** : produit par `cmd/replay-verite -registre`, compilé dans le worktree de base
  avec le GOCACHE de la base. Il est lu **une fois par passage et n'entre pas dans le cache**, parce
  qu'il ne dépend que du SHA de base, que la clé du cache porte déjà. Une base sans l'outil donne un
  registre inconnu, et le rapport le dit.
- **Oracle** : l'export des faits passe `--oracle`, et `<short8>.oracle.json` est lu à côté des
  faits. S'il est absent ou vide, O-S3 manque des deux côtés et le verdict n'est pas touché ;
  l'absence est journalisée. **O-S3 (score personnel) est branché**, contre `personal_score`, ou
  `score` s'il manque : la synchro remplit les deux colonnes depuis `CoreStats.PersonalScore`. Sa
  ligne de base connue : les décréments ne sont pas publiés, donc un joueur qui s'est suicidé ou a
  trahi porte un FP stable. **O-S4, O-S5 et O-X1 restent pour la 2b** : il faut une table stat
  publiée → colonne d'oracle, et mesurer ce que l'API compte en tirs.
- **Rapport** : une section `BANC DE VERITE` par témoin, placée avant le détail `replaydiff` ; un
  objet `verite` dans le JSON (statut, constats, filets). Les compteurs gains / pertes / changements
  restent au tableau, pour information.
- **Tests** : verdict du gate contre verdict du banc, filets, O-S1 circulaire, artefact réel
  (`bcb6d393` abîmé donne `FAUX`, section avant le détail, JSON), registre sans outil, rendu du
  registre inconnu. Les tests existants ont été mis à jour (changements informatifs, fixture de
  paire réelle : sans banc, les 13 pertes sont 13 filets). 15 mutations par copie sur le gate et
  O-S3, toutes rouges.
- **Non fait** : aucun essai réel (aucun décodage, aucune exécution du gate, sur consigne).
