# Plan — Objectifs vivants du rejeu 2D, deuxieme lecture (drapeau, crane, colline)

> Ecrit le 2026-08-17 par la session de pilotage (item 4 de la file du handoff superviseur).
> Sujet utilisateur (bilan de la planche, item F1) : « les objectifs, le but c'est de les avoir
> aussi et de les symboliser, surtout le crane d'Oddball et le drapeau de CTF, ou le noyau de
> Stockpile » ; precision du 14/08 : « KOTH n'est PAS hors de portee — la colline BOUGE, mais elle
> est FIXE pendant une periode ». Deuxieme lecture APRES le lot R4 (`PLAN_R4_OBJECTIFS_VIVANTS_TI11.md`,
> clos : `ti=11` est le DESCRIPTEUR d'objectif du HUD, sans position) et la lignee R7 (session
> utilisateur : la bit-exactitude des deserialiseurs d'image-cle plafonne a 0,85 % meme sur le
> bipede). Ce plan NE repasse PAS par ces voies. Branche `feat/v75`, contrat `plan-execution`.
> **Lancement : apres l'item 6 (ordre du handoff) ET apres la fusion par l'utilisateur de
> `wt/fusion-finale` + `wt/poses-revue-fix`** (elles modifient `traverse.go`, ou vit le canal
> « arme en main » que la phase 0 mesure) — sinon la mesure se ferait sur un decodeur qui va bouger.

## Ce qui est FAIT (sur pieces, ne pas refaire) — digest du 2026-08-17

| acquis | ou | portee |
|---|---|---|
| `ti=11` = descripteur d'objectif du HUD, 34 composants `managed-objective-*`, AUCUNE position ; contenu jamais lu (grammaire seule) ; voie delta refutee par temoin fantome (0,73x) ; voie image-cle bloquee par la grammaire du corps | `PLAN_R4_...md` §4, `sonde_ti11_*_test.go`, registre R4 | NEGATIF acquis — on ne decode PAS l'objet |
| Bit-exactitude d'image-cle : plafond 3-5 / 591 records sur le bipede (R7 a-e), cadre resolu, derive DANS les deserialiseurs | `PLAN_R7A..E_*.md` (`wt/fusion-finale`) | la condition de reprise R4 (« etat par defaut bit-exact ») est un chantier hors v7.5 |
| « Arme en main » DELTA : `weapon-state-type-info` (i43-i46) lu par `consumeWeaponStateTypeInfoVariant` (`components_object.go:318`), capture dans `EntityTrace.HeldWeapon` (`traverse.go:36`, `:1224`), cache par slot `World.SetHeldWeapon/HeldWeapon` (`world.go:107-119`) — **aucun appelant, jamais publie, jamais mesure** | filmdec | canal candidat n°1 (image par image) |
| Loadouts d'images-cles : `KeyframeLoadout{TimestampUS, Slot, Families []uint32}` (balayage des 32 bits de famille, ancrage 1750x anti-hasard) ; publie `Loadout` champ `w` = INVENTAIRE, pas la main | `keyframe_loadout.go`, `replay/document.go:406-430` | canal candidat n°2 (toutes les ~20 s) |
| Evenements CTF NOMMES par slot + `TimeMS` : `FlagGrabs`, `FlagSteals`, `FlagCaptures`, `FlagReturns`, `FlagCarriersKilled`, `FlagCaptureAssists` ; identifies en xuid (`IdentifiedEvent`) ; zones : `ZoneCaptures`/`ZoneSecures` ; **KOTH et Oddball sans table** (« le corpus manque ») | `objectiveevents/named.go:83-105,158-160`, `slotidentity.go` | ORACLE de la phase 0 (CTF) |
| Chaine DB `match_objective_events` : CTF bursts ms-exacts, KOTH/Oddball/Zones `th=10` approx 5-20 s, `objective_id` toujours NULL | `objectiveevents/extract.go`, `migration/steps_shared_objective_events.go` | pas de position ni de colline |
| Objectifs STATIQUES depuis `.mvar` : roles `flag_spawn`, `flag_delivery`, `strongholds_zone`, `extraction_zone`, `oddball_spawn`, `stockpile_*`, `assault_bomb` ; **aucun role colline** ; construits a la requete | `replay/map_objectives.go`, `mapvar/objectives.go:66-75`, `config/.../objective_roles.toml`, `data/.../map_objectives.json` | socles/zones : fait |
| Web : `objectivesLayer.ts` (324 l.) dessine zones + marqueurs, et des « pulses » d'action poses sur l'ELEMENT STATIQUE LE PLUS PROCHE de l'auteur — un substitut, pas une lecture | `apps/web/src/features/match-replay/objectivesLayer.ts:253-290` | a remplacer par l'objet vivant |
| Contrat gele : `wantReplayDocumentFields = 31` ; `objectives` (22->23), `mapObjectives` (23->25) | `contracttest/replay_contract_test.go:172` | phase 1 = +1 champ, chronique |
| Corpus : CTF `64e8adfa`, `530820e5` (Catalyst), `53ce4390` ; KOTH `0a247154`, `01e1f945`, `606d9844`, `8076f97f` ; Strongholds `7344d24f`, `696a9d7c` ; Oddball `24dbb67d` ; **Stockpile : aucun film exploitable** (404 Theater) | R4, `corpus_objectifs/README.md`, etat de l'art mode/score | Stockpile `[!]` corpus des le depart |

## Decisions tranchees avant execution

1. **On ne decode PAS l'objet ; on lit son PORTEUR.** Le drapeau et le crane sont des armes
   (`weap`) tenues en main : la position de l'objet porte EST la position du porteur (piste
   bipede deja publiee). Le canal se MESURE en phase 0 contre l'oracle des evenements CTF
   nommes (grab/steal = debut de portage ; capture / porteur tue / mort du porteur = fin).
2. **Deux canaux, un seul gagnant publie** : delta `HeldWeapon` (image par image) s'il tient
   le seuil, sinon loadouts d'images-cles (granularite ~20 s, publiee comme telle) ; si aucun
   ne tient : NEGATIF ecrit, la phase 1 est `[!]`, la phase 2 (colline) reste ouverte.
3. **L'objet au repos** (drapeau lache) = derniere position du porteur a la fin du portage,
   jusqu'au prochain grab, a un `FlagReturns`, ou au retour automatique (duree MESUREE sur les
   films : ecart median entre fin de portage sans reprise et prochain grab au socle) ; l'objet
   « a la maison » = marqueur statique existant (`flag_spawn` / `oddball_spawn`) avec un etat
   present/absent. Quel drapeau (equipe) : celui dont le socle est le plus proche du point de grab
   (`FlagSteals` = drapeau adverse par definition).
4. **Colline KOTH = zone FIXE par periode.** Candidats de zones : les formes du `.mvar` de la
   carte non attribuees a un role (le decodeur `mapvar` les voit deja) ; la zone ACTIVE d'une
   periode = celle qui contient les positions des joueurs aux instants de score de zone (`th=10`
   approx, ou pistes) ; les bornes de periode = les sauts de cette grappe. Publie SEULEMENT si
   >= 80 % du temps de match est couvert par des periodes attribuees, sur >= 2 films.
5. **Oddball** : meme canal que le drapeau (phase 0.3), oracle structurel (une seule main a la
   fois) + evenements approx ; si le canal echoue sur le crane, `[!]` mesure.
6. **Stockpile** : `[!]` corpus (aucun film) — ecrit, pas cherche.
7. **Rendu (phase 3, worktree frere web)** : icone drapeau/crane collee au marqueur du porteur,
   objet lache dessine a sa position, socle avec etat present/absent, colline active en
   surbrillance et collines inactives estompees ; tokens uniquement, FR+EN ; les « pulses »
   substituts de `objectivesLayer.ts` sont RETIRES quand l'objet vivant est publie (0 code mort).
8. Regles inchangees : seuils avant mesure, denominateurs publies, un seul decodage filmdec par
   process, aucune base en ecriture, JAMAIS `git add -A`, jamais d'attente passive, decouvertes au
   registre.

## Phases

### Phase 0 — MESURER le canal du porteur (worktree principal, films)

- [x] 0.1 **La famille du drapeau.** Sur les 3 films CTF : pour chaque `FlagGrabs`/`FlagSteals`
      d'un slot S a t, fenetre de portage = [t, min(capture de S, mort de S, `FlagCarriersKilled`
      dont la victime est S)] ; familles des loadouts d'images-cles de S DANS la fenetre moins
      familles de S HORS fenetre => famille candidate. Seuil : UN identifiant 32 bits, le MEME sur
      les 3 films, present dans >= 90 % des fenetres qui contiennent au moins une image-cle ; temoin :
      un slot non porteur tire au hasard sur les memes fenetres le porte dans <= 5 %. Denominateurs.
- [x] 0.2 **Le canal delta `HeldWeapon`.** Instrument sous garde (`OBJ_FILM`), lecture seule, dans
      le paquet `filmdec` (accesseur de test ou hook — AUCUN changement de production avant
      validation) : latence entre le grab et la premiere image ou `HeldWeapon(S)` = famille du
      drapeau (mediane, p90) et entre la fin de portage et le retour a une autre famille. Seuil :
      >= 90 % des grabs apparies a <= 2 s ; sinon canal REFUTE (ecrit), le canal image-cle seul est
      retenu s'il a tenu 0.1.
- [x] 0.3 **Nommer et generaliser.** Croiser la famille du drapeau avec le dictionnaire des tags
      `weap` (index killweapon, `weapon_labels`, sprite index) — attendu : le tag du drapeau ; le
      crane d'Oddball sur `24dbb67d` par signature structurelle (une famille portee par UN seul
      bipede a la fois, qui change de main) — seuil : <= 1 porteur sur >= 90 % des images ou la
      famille est portee ; les evenements approx `th=10` comme controle (accord grossier publie).
- [x] 0.4 Publier au journal du plan : famille(s), canal retenu, latences, denominateurs.

**Gate 0** : famille du drapeau identifiee (0.1) ET un canal valide (0.2 ou 0.1) => phase 1 ;
sinon NEGATIF ecrit, phase 1 `[!]`, passage direct a la phase 2.

**Gate 0 : PASSE SUR SES CRITERES ECRITS, AVEC UNE RESERVE DE NATURE** (2026-08-18, mesures au
journal ci-dessous). Le critere (b) est tenu : le canal IMAGE-CLE porte les deux seuils. Le
critere (a) est tenu AU SENS DE SES PROPRES CHIFFRES — un identifiant de 32 bits, le meme sur
les trois films, 97,4 % des fenetres, temoin 2,6 % — mais l'item 0.3 etablit que cet
identifiant **n'est PAS une famille d'arme** : il ne porte pas le suffixe d'identifiant `weap`
(0x42C9679F, 0 occurrence sur 83), et le canal delta des armes tenues ne le voit jamais
(0 sur 68 284 lectures). **La decision 1 du plan — le drapeau et le crane seraient des `weap`
tenus en main — est donc REFUTEE** ; ce qui est trouve est un MARQUEUR DE PORTAGE, lisible et
mesure, mais anonyme et non generalise au crane. La phase 1 ne peut donc pas etre ouverte telle
qu'elle est ecrite (elle publie `kind` flag/ball et `team`, qu'aucune mesure ne donne) : elle
est a REDIGER A NOUVEAU sur ce que la phase 0 rend reellement. Arret apres la phase 0,
conformement au brief.

### Phase 1 — PUBLIER le portage du drapeau (CTF), schema +1 — REECRITE le 2026-08-18 apres la phase 0

> **Arbitrage superviseur (2026-08-18, CR de phase 0 verifie sur pieces, fusion `6b856ea3a`)** : la
> decision 1 est REFUTEE en partie — le marqueur de portage `0x00010005` n'est PAS une famille
> d'arme et ne nomme rien ; le crane d'Oddball n'a ni marqueur ni oracle. Ce qui EST mesure : en
> CTF, le porteur se lit a l'image-cle (97,4 %, temoin 2,6 %) et les evenements CTF nommes datent
> les bornes a la milliseconde. La phase 1 publie donc le PORTAGE DU DRAPEAU en CTF, ou `kind` vient
> du MODE (CTF => drapeau) et non du marqueur, et ne publie RIEN pour Oddball (item `[!]`) ni pour
> le canal delta (refute). Deux prealables mesures entrent en production : le pont statborg par
> INSTANTS de mort (repli quand le pont par totaux rend < 8 identites — films tronques), et le
> retrait de `World.SetHeldWeapon/HeldWeapon` (0 appelant, mesure close : regle 0 code mort).

- [x] 1.0 Prealables : (a) `objectiveevents` — repli du pont par totaux vers le pont par INSTANTS de
      mort (l'instrument de la phase 0 le porte : 8/8 sur les 4 films, 8 accords / 0 desaccord),
      code de production + test (film tronque -> identites completes ; film complet -> identique a
      l'existant) ; (b) `filmdec` — retirer `World.SetHeldWeapon`/`HeldWeapon` et le champ de trace
      qui ne sert qu'a eux (verifier : aucun appelant hors tests ; les instruments de la phase 0.2
      qui les lisaient sont ajustes ou retires — pas de code mort ni de test qui teste du mort).
- [!] 1.1 Portages CTF (films `64e8adfa`, `530820e5`, `53ce4390`) : un portage = [`FlagGrabs` ou
      `FlagSteals` du slot S a t0] -> fin = min(`FlagCaptures` de S, mort de S dans les pistes,
      `FlagCarriersKilled` dont la victime se resout a S) ; porteur = S -> xuid par le pont ; equipe du
      drapeau = socle `flag_spawn` le plus proche du point de grab (`FlagSteals` = adverse) ; le
      marqueur d'image-cle sert de CONTROLE (part des portages confirmes par >= 1 image-cle portant
      `0x00010005` sur S ; seuil ecrit >= 90 % des portages ayant une image-cle) ; simultaneite : au
      plus 2 portages ouverts a la fois — les depassements (`64e8adfa` : 6, dus aux 53 records sans
      pont) sont RESOLUS par le pont 1.0(a) ou publies comme incoherences comptees (jamais tus).
- [x] 1.2 Drapeau lache / a la maison : fin de portage sans capture => `dropped` a la derniere
      position du porteur jusqu'au prochain grab, a un `FlagReturns`, ou au retour automatique
      (duree MESUREE : ecart median entre fin de portage sans reprise et prochain grab au socle) ;
      sinon `home` (marqueur statique existant, etat present/absent).
- [x] 1.3 Document : `flagCarries` [{ team, spans [{ state `carried`|`carried_open`|`dropped`|`home`,
      t0, t1, xuid|null, x, y }] }] + `coverage.flagCarries` (portages, fermes, ouverts, confirmes
      par marqueur, sans pont, sans piste, incoherences) ; `SchemaVersion` 13 -> 14 avec chronique ;
      contrat (`wantReplayDocumentFields` 34 -> 35, ligne de chronique), OpenAPI (`cmd/openapi-gen`),
      `generated.ts`, `NULLABLE_ARRAYS`/`normalizeReplayDocument`, goldens re-congeles, temoins CTF
      re-cuits ; films non-CTF : tableau vide. **FAIT le 2026-08-18** (report de coordination leve :
      la fusion tierce a pris 12 et 13, le calque prend 14) — mesures et verdict au journal.
- [~] 1.4 Registre : ligne R4 « objectifs vivants » -> porteur TRAITE (CTF) ; crane `[!]` (condition
      de reprise : discriminant externe, score par seconde de portage) ; canal delta CLOS (retire).
      Couvert par le CR de lot : les TEXTES de journal et de registre y sont fournis, et c'est le
      superviseur qui les consigne a la fusion (`.ai/thought_log.md` et `REGISTRE_REPORTS.md` ne
      sont pas touches par cette branche).

**Gate 1** : controle 1.1 >= 90 % ; chaque `FlagGrabs`/`FlagSteals` de l'oracle ouvre un span
`carried` du bon xuid ; contrat/OpenAPI/goldens verts ; incoherences publiees, pas masquees.

**Gate 1 : NON ATTEINT SUR SA PREMIERE CLAUSE (2026-08-18), LES AUTRES SONT TENUES.** Le controle
du marqueur donne **37 / 42 = 88,1 %** contre 90 % exiges — seuil NON rebaisse, item 1.1 statue
`[!]` avec sa mesure. Clause 2 TENUE : les 149 prises nommees des trois films rendent 137 spans
`carried`, et les 12 manquantes sont TOUTES comptees sous une cause nommee (`NoTrack`). Clause 3
TENUE : les incoherences sont publiees et comptees (simultaneite > 2, porteurs tues ambigus,
retours ambigus). Clause 4 SANS OBJET : la publication au document est reportee (item 1.3).

**Gate 1.3 (arbitrage du 2026-08-18) : TENU, ET LA CLAUSE 4 EST LEVEE.** Le controle du marqueur
sur les portages FERMES donne **38 / 40 = 95,0 %** (seuil >= 90 %, jamais rebaisse), et **40 / 42
= 95,2 %** tous portages confondus — le 88,1 % de l item 1.1 ne tenait pas au seuil mais au pont :
la production nomme plus de slots que l instrument (fermetures). Clause 2 TENUE (chaque prise
nommee ouvre un span porte du bon xuid, rejets comptes). Clause 3 TENUE, avec une reserve NEUVE :
la simultaneite > 2 de `64e8adfa` n est PAS portee par les portages ouverts (ce film n en a aucun)
— elle oppose des portages FERMES, et se publie sous `overlaps` ET `closedOverlaps`. Clause 4
TENUE : contrat, OpenAPI, `generated.ts`, frontiere de nullabilite et goldens sont verts.

### Phase 2 — La COLLINE par periode (KOTH)

- [ ] 2.1 Sur les 4 films KOTH : formes du `.mvar` de la carte non attribuees a un role (liste,
      nombre) ; attendu 3-6 candidats. Si 0 candidat : chercher la colline dans les POSITIONS
      seules (grappe des joueurs qui marquent), l'ecrire.
- [ ] 2.2 Zone active par periode : positions des joueurs aux instants de score (`th=10`), grappe,
      attribution a une forme candidate (>= 80 % des positions d'une periode dans UNE forme),
      bornes de periode aux sauts ; duree des periodes (mediane, p10-p90) ; couverture du temps
      de match par des periodes attribuees.
- [ ] 2.3 Publier `hillPeriods` [{t0, t1, zoneIndex}] + role `hill` (`objective_roles.toml`
      + `mapvar` `Role`) pour que `mapObjectives` porte les collines ; contrat/OpenAPI/goldens.

**Gate 2** : couverture >= 80 % sur >= 2 films ; sinon `[!]` avec la mesure.

### Phase 3 — RENDU (worktree frere web `../LevelUp-wt-objvivants`, branche `wt/objvivants`)

- [ ] 3.1 `objectivesLayer.ts` : icone DRAPEAU (CTF seulement, le crane est `[!]`) collee au marqueur du porteur (sprite existant
      si le dictionnaire en a un, sinon glyphe SVG neutre), objet `dropped` a sa position avec
      respiration, socle `home` present/absent ; collines : active en surbrillance, inactives
      estompees ; les pulses substituts RETIRES (0 code mort) ; tokens (`color-tokens`), FR+EN.
- [ ] 3.2 Tests (formes, etats, aucune couleur en dur, parite i18n) ; typecheck/lint/vitest apres
      purge `node_modules/.tmp` ; note pour la planche (temoins : `64e8adfa`, `01e1f945`).
- [ ] 3.3 Fusion `--no-ff` par le superviseur, gates rejoues, journal, registre.

**Gate 3** : gates web verts ; gate VISUEL utilisateur (planche + en app).

## Regles dures

Porteur AVANT objet ; jamais de decodage de `ti=11` ni de corps d'image-cle dans ce plan ;
seuils ecrits ci-dessus, jamais rebaisses ; un canal refute s'ecrit ; aucun rendu avant la
donnee ; commits sur `feat/v75` (donnees) et `wt/objvivants` (web), pas de push.

## Contrat d'execution (rappel, `plan-execution` fait foi)

Une phase a la fois ; phase CLOSE = gate joue dans la session (sorties au journal du plan),
items statues (`[x]`/`[~]`/`[!]`), plan mis a jour et commite avec le lot, entree en tete de
`.ai/thought_log.md`, registre a jour, gates Go verts (`go build`, `go vet`, `go test
./internal/analysis/... ./internal/archlint/... ./contracttest/...`, `golangci-lint
--new-from-merge-base=origin/main` = 0). Aucun `git add -A` ; aucune attente passive ; aucun
fix hors perimetre (decouvertes ci-dessous) ; un seul `go` a la fois.

## Decouvertes (hors perimetre — notees, NON traitees)

- ~~`World.HeldWeapon(slot)` n'a aucun appelant (digest 2026-08-17)~~ **TRAITE (item 1.0(b),
  2026-08-18)** : la phase 0 ne l'a PAS branche et a REFUTE le canal qui aurait pu lui donner un
  appelant. `SetHeldWeapon` / `HeldWeapon`, le champ `slotState.HeldWeapon`, le champ
  `EntityTrace.HeldWeapon` et sa capture dans `traverse.go` sont supprimes, avec l'instrument
  0.2 qui les mesurait (0 test de code mort).
- La lignee R7 n'a ni ligne de registre ni entree de journal partage (vit dans 5 plans + commits
  de `wt/fusion-finale`) — a consigner par la session utilisateur a sa fusion.
- `HANDOFF_KEYFRAME_LIVE_CAPTURE.md` (04/07, « 205 entites keyframe decodees ») contredit R5-R7 sans
  retractation explicite — une note croisee est due au registre.
- ~~(2026-08-18, phase 0) **`objectiveevents.SlotIdentity` tombe a 0/8 sur un film TRONQUE.**~~
  **TRAITE (item 1.0(a), 2026-08-18)** : le pont par INSTANTS DE MORT est en production
  (`objectiveevents/slotidentity_deaths.go`), `SlotIdentityResolved` s'y replie quand il nomme
  STRICTEMENT plus de slots, et son seul appelant existant (`cmd/zone-attribution`) y est migre.
  Mesure d'origine conservee : 0/8 sur `64e8adfa` et `24dbb67d`, 8/8 sur `530820e5` et
  `53ce4390` ; le remplacant rend 8/8 sur les quatre et concorde 8/8 avec l'existant la ou
  celui-ci repond.
- (2026-08-18, phase 0) **Le pont bipede laisse 53 records d'image-cle sans porteur sur
  `64e8adfa`** (16 % des records), et c'est le seul film ou le plafond structurel de deux porteurs
  au plus est depasse. Les deux faits vont probablement ensemble ; non instruit ici.
- (2026-08-18, phase 0) **La marche stateful rend 96 a 100 % de variantes NULLES sur les slots de
  joueur.** Sur 68 284 lectures d'arme tenue, moins de 1 600 tombent sur un bipede nomme, et la
  quasi-totalite valent `0x00000000`. C'est une mesure de la sante du decodeur delta, hors
  perimetre de ce lot mais utile a qui reprendra le chemin i43-i46.
- (2026-08-18, phase 1) **`replay.Options.Objectives` n'est renseigne par AUCUN appelant de
  production.** Le seul site qui le remplit est `cmd/zone-attribution/measure.go`, un outil de
  mesure ; `replaybuild.BuildMatch` — la porte de TOUS les artefacts cuits (CLI, ouvrier, action
  admin, etape post-sync) — ne le fournit pas. Le champ `objectives` du document est donc VIDE
  dans tous les artefacts de production, alors que le contrat le decrit et que le web le lit. NON
  TRAITE : le reparer demande le pont d'identite ET la famille d'objectif du match, c'est-a-dire
  la meme plomberie que l'item 1.3.
- (2026-08-18, phase 1) **`public_name` est VIDE sur la quasi-totalite des entrees de
  `map_objectives.json`** (produit depuis les variantes UGC, qui ne le portent pas). Toute
  jointure carte -> objectifs doit passer par `map_id` ou par `module` ; joindre sur le nom public
  ne trouve rien, SILENCIEUSEMENT. Le service joint bien par `map_id` — c'est la mesure de la
  phase 1 qui s'est fait prendre.
- (2026-08-18, phase 1) **Les deux catalogues de carte ne nomment pas les modules pareil.**
  `map_quant_bounds.json` dit `va_behemoth` et `ridgeline` la ou `map_objectives.json` dit
  `behemoth_va_behemoth` et `cliffhanger_ridgeline` ; Catalyst y figure DEUX fois (`catalyst` et
  `catalyst_map`, memes socles au centimetre). Les deux sont produits par des chaines differentes
  et AUCUN test ne les rapproche : toute jointure entre eux PAR LE MODULE se fera de travers,
  sans rien dire. NON TRAITE — l'item 1.3 joindra par `map_id`, comme le service.
- (2026-08-18, phase 1) **Chaque carte de CTF declare TROIS `flag_spawn`** : un par equipe, plus un
  NEUTRE au centre (variantes « drapeau neutre »). Les publier tous sur une partie de CTF ordinaire
  ferait apparaitre un troisieme drapeau qui n'existe pas dans le match. La mesure ne retient que
  les socles d'equipe ; l'item 1.3 devra trancher le cas du drapeau neutre (`0f9550e5` est au
  corpus et le discriminant le reconnait comme CTF).
- (2026-08-18, item 1.3) **Le calque `objectives` reste vide sur un film TRONQUE, alors que le
  drapeau ne l'est plus.** `replaybuild.identifiedEvents` nomme ses actions avec le pont par
  TOTAUX (`objectiveevents.SlotIdentityFrom`), celui que la phase 0 a mesure a 0/8 sur un film
  tronque ; la cuisson du temoin `64e8adfa` le montre en clair (`nommees=266 identifiees=0
  slotsApparies=0`) au moment meme ou `flagCarries` publie ses 78 portages par le pont des
  INSTANTS DE MORT. Le remede est connu (`SlotIdentityResolved`, deja en production) et tient en
  une ligne, mais `objectives` appartient a un autre chantier et le present lot n'y touche pas.
- (2026-08-18, phase 1) **Deux espaces de coordonnees se ressemblent assez pour se confondre.** Les
  pistes construites en `QuantaOnly` (phase 0) portent des INDICES DE QUANTUM ; les socles du
  catalogue portent des METRES. Un rayon de ramassage ecrit en metres, applique a des quanta, ne
  signale rien — il attribue simplement le drapeau au hasard. La mesure de la phase 1 exige donc
  les bornes de carte la ou la phase 0 s'en passait.

## Journal du plan (avancement, source de verite pour la reprise)

- 2026-08-17 — plan ecrit (digest R4/R7 verifie sur pieces par un agent de lecture). EN ATTENTE :
  fin de l'item 6, fusion utilisateur de `wt/fusion-finale` + `wt/poses-revue-fix`. Non lance.

### 2026-08-18 — PHASE 0 CLOSE (worktree `LevelUp-wt-objvivants`, branche `wt/objvivants`)

Instruments : `internal/analysis/replay/objectifs_phase0_*_test.go` (6 fichiers, garde `OBJ_FILM`
= racine du cache film, lecture seule, aucune base ouverte, aucun changement de production).

**Prealable — LES DEUX PONTS, et un pont NEUF qu'il a fallu construire.** Le pont statborg par
TRIPLETS (`objectiveevents.SlotIdentity`, totaux frags/morts/assistances contre
`match_participants`) rend **0 slot sur 8** sur `64e8adfa` et `24dbb67d` : ces films sont
TRONQUES (compteurs du film inferieurs aux totaux de l'API — ex. `64e8adfa` slot 24 = 10/18/7
contre 10/21/7 a l'API). Voie de remplacement, sans aucun recours a la base : apparier les
INSTANTS de progression du compteur de morts (`comp 2 B`) au FIL DES MORTS du film (xuid +
instant), les deux etant sur l'horloge du match. Resultat **8/8 sur les quatre films**, et
**8 accords / 0 desaccord** avec le pont par triplets la ou celui-ci repond (`530820e5`,
`53ce4390`) — deux chaines totalement disjointes qui disent la meme chose. Pont bipede (fil des
morts) : 122/141, 92/98, 109/144, 87/105 vies nommees ; collisions 0, 0, 2, 0.

**0.1 — LE MARQUEUR DE PORTAGE : `0x00010005`.** Balayage SANS predicat des fenetres de 32 bits
de l'emprise des records de bipede d'image-cle (le catalogue d'armes exclurait le drapeau par
construction), confronte aux fenetres de portage de l'oracle CTF.

| film | images-cles | records bipede (sans pont) | fenetres de portage | records portage / hors | motifs candidats |
|---|---|---|---|---|---|
| `64e8adfa` | 43 | 337 (53) | 82 | 24 / 260 | 1 |
| `530820e5` | 25 | 178 (14) | 33 | 4 / 160 | 11 |
| `53ce4390` | 39 | 300 (39) | 34 | 20 / 241 | 8 |

Taux par record : 24/24 = 100 % en portage contre 6/260 = 2,31 % hors (`64e8adfa`) ; 3/4 = 75 %
contre 2/160 = 1,25 % (`530820e5`) ; 20/20 = 100 % contre 4/241 = 1,66 % (`53ce4390`).
**Un seul motif est commun aux trois films** : 4 valeurs (`0x00010005`, `0x0002000B`,
`0x00040017`, `0x0008002F`) qui sont **la meme suite de bits lue a quatre decalages** — repliees,
elles font UN motif de racine `0x00010005`.

Seuils du plan, appliques aux FENETRES : **37/38 = 97,4 %** (seuil >= 90 %) — par film 21/21,
3/4, 13/13. **Temoin 1/38 = 2,6 %** (seuil <= 5 %). **TENU.**
Correction d'instrument a signaler : un premier passage donnait un temoin a 10,5 % (4/38) parce
que le tirage pouvait designer le porteur ADVERSE (il y a deux drapeaux en CTF). Le tirage
exclut desormais tout joueur lui-meme en fenetre de portage — c'est ce que le plan demande
(un slot NON porteur) ; aucun seuil n'a bouge.

**Controle POSITIF de l'emprise** (sans lui, un ecart ne prouve rien) : 258/337, 132/178 et
233/300 records (74 a 78 %) portent au moins une famille d'arme CONNUE, 1,47 a 1,55 par record.
Le balayage lit donc bien la zone des armes. **Controle croise sur les familles connues** : le
MA40 et le Sidekick sont a 87,5 % / 75,0 % en portage contre 60,4 % / 66,9 % hors — aucune
famille connue ne separe le portage. Le marqueur n'est donc pas un effet de changement
d'armement du porteur.

**0.2 — CANAL DELTA `World.HeldWeapon` : REFUTE, et diagnostique.** Marche stateful sur tout le
film (monde reamorce a chaque image-cle), API exportee seulement, aucun changement de production.

| film | paquets delta | records propres | lectures d'arme | slots | familles |
|---|---|---|---|---|---|
| `64e8adfa` | 50 956 | 138 610 / 183 330 | 30 141 | 5 783 | 33 |
| `530820e5` | 29 148 | 68 464 / 94 564 | 15 298 | 3 970 | 23 |
| `53ce4390` | 45 856 | 120 112 / 161 058 | 22 845 | 5 514 | 26 |

Le canal VIT (68 284 lectures au total) mais **il ne porte pas le marqueur : 0 occurrence sur les
trois films**, et **0/149 prises appariees a <= 2 s** (seuil >= 90 %) — latence mediane et p90
sans objet. Cause nommee et mesuree : sur les slots que le pont bipede NOMME il ne reste que
462, 456 et 617 lectures, et **96 a 100 % d'entre elles valent `0x00000000`** (variante nulle) ;
aucune des familles lues n'est nommee par le catalogue d'armes. La quasi-totalite des 5 783
slots touches ne sont pas des joueurs. **Canal REFUTE ; le canal image-cle de 0.1 est le seul
retenu.** Consequence pour la decouverte du plan (`World.HeldWeapon` sans appelant) : la phase 0
ne le branche PAS — il reste sans appelant, et la question de son retrait est desormais
tranchable, avec ce chiffre.

**0.3 — NOMMER : NEGATIF ECRIT. GENERALISER AU CRANE : NEGATIF ECRIT.**
Nommage : le marqueur n'est suivi du suffixe d'identifiant d'arme `0x42C9679F` **0 fois sur
83 occurrences** (42 + 11 + 30). Ce n'est donc pas un `weap`, il n'a pas de global tag id, et
**aucun nom de tag ne peut lui etre donne** — ni par l'index killweapon, ni par `weapon_labels`,
ni par le sprite index, qui sont tous keyes sur ce meme espace de tags. Contexte binaire, stable
sur les trois films : les 32 bits qui le PRECEDENT valent `0x00000006` (42/42, 11/11, 25/30) ;
ceux qui le SUIVENT sont `0xF19A127D` (12, 6, 12 fois) puis des valeurs `0xE01340xx` a octet bas
variable. Position mediane dans le record : **1 100, 1 068 et 1 190 bits** depuis son debut —
loin des ~1 950 bits ou vivent les identifiants d'arme.
Simultaneite (controle structurel : deux drapeaux en CTF, donc 0, 1 ou 2 porteurs) : `53ce4390`
respecte exactement le plafond (0 -> 17 images, 1 -> 12, 2 -> 9) ; `530820e5` le depasse une fois
(3 porteurs sur 1 image sur 23) ; `64e8adfa` six fois (3 porteurs sur 5 images, 4 sur 1, sur 42)
— c'est le film dont 53 records n'ont pas de pont. **Le marqueur n'est donc pas exclusivement
l'objet drapeau**, ou l'attribution de slot derape sur ce film ; a trancher avant toute
publication.
Crane (`24dbb67d`, 27 images-cles, 209 records) : **le motif CTF y est totalement absent**
(0 porteur sur les 26 images) — le marqueur n'est PAS universel entre modes. La signature
structurelle seule ne discrimine pas : 1 981 valeurs (195 motifs apres repli) tiennent
« presente sur >= 20 % des images, <= 1 porteur sur >= 90 % d'entre elles, >= 2 porteurs
distincts ». Sans oracle nomme (le statborg ne replique aucun compteur de crane), **aucun
candidat unique ne se degage**. Controle grossier disponible et sain : 87 evenements `th=10` de
crane, 87 acteurs, **87/87 nommes par le pont bipede** — il ne manque que l'objet a confronter.

**0.4 — publie ci-dessus.** Verdict du gate 0 : voir la fin de la section Phase 0.

Commits : `26b245d2a` (socle), `bedc3b8d8` (pont par instants, controles, seuils),
`0e5b70c7a` (canal delta, nommage, crane), `5d733c8c6` (ce versement), `baa053ad8` (canal
delta confronte famille par famille). Gates rejoues dans la session, depuis `apps/go-api` du
worktree : `go build ./...` (CGO, msys64 ucrt64) = 0, `go vet ./...` = 0, `go test
./internal/analysis/... ./internal/archlint/...` = 0, garde `OBJ_FILM` absente = 5 SKIP,
`golangci-lint run --new-from-merge-base=origin/main` = 0 issue.
- 2026-08-18 — phase 0 CLOSE et fusionnee (`6b856ea3a`, docs `bfe4d5b10`) ; **phase 1 REECRITE**
  par le superviseur (portage CTF par evenements nommes + marqueur en controle ; pont par instants
  et retrait de HeldWeapon en prealables ; Oddball `[!]`) ; phase 1 LANCEE (agent Opus, worktree
  frere `../LevelUp-wt-objvivants1`).

### 2026-08-18 — PHASE 1, ITEMS 1.0 A 1.2 (worktree `LevelUp-wt-objvivants1`, branche `wt/objvivants1`)

**ARBITRAGE DE COORDINATION** : l'item 1.3 (publication au document) est REPORTE apres le rebasage
sur `feat/v75` — une autre session y fait entrer les schemas 12 et 13, et deux montees de version
concurrentes se marcheraient dessus. Les types du calque existent
(`replay/document_objectives_live.go`) mais SANS etiquette JSON et sans champ dans
`ReplayDocument` : contrat, OpenAPI, `generated.ts` et goldens sont INTACTS.

**1.0(a) — LE PONT PAR INSTANTS DE MORT EST EN PRODUCTION.**
`objectiveevents/slotidentity_deaths.go` : `SlotIdentityFromDeaths` (film seul, aucune base) et
`SlotIdentityResolved` (totaux d'abord, repli sur les instants quand ceux-ci nomment STRICTEMENT
plus de slots ; desaccords ECARTES et comptes). Cinq tests purs, sans film, tournent en CI : film
complet -> identique a `SlotIdentity` ; film tronque -> 8/8 ; abstention sans marge ; abstention
sous le minimum de morts communes ; sans fil des morts -> totaux seuls ; desaccord -> slot retire.
Le seul appelant existant de `SlotIdentity` (`cmd/zone-attribution`) est migre vers le pont
resolu — sans quoi la voie neuve serait du code mort.

Mesure sur films reels, l'instrument APPELANT la production (il n'a plus de pont a lui) :

| film | pont par INSTANTS | pont par TRIPLETS | recoupement | pont RESOLU |
|---|---|---|---|---|
| `64e8adfa` | 8 | **0** (film tronque) | — | 8, voie `deaths` |
| `530820e5` | 8 | 8 | 8 accords / 0 desaccord | 8, voie `totals` |
| `53ce4390` | 8 | 8 | 8 accords / 0 desaccord | 8, voie `totals` |
| `24dbb67d` | 8 | **0** (film tronque) | — | 8, voie `deaths` |

**1.0(b) — LE CACHE `World.HeldWeapon` EST RETIRE.** `World.SetHeldWeapon`, `World.HeldWeapon`,
le champ `slotState.HeldWeapon`, le champ `EntityTrace.HeldWeapon` et sa capture dans
`traverse.go` (i43-i46) disparaissent, avec les six initialisations `HeldWeapon: noVariant` qui
les accompagnaient. L'instrument 0.2 qui les mesurait est SUPPRIME (326 lignes) : il testait un
canal refute a travers du code qui n'existe plus. Diff : 7 fichiers, 11 insertions / 39
suppressions, plus la suppression de l'instrument. La chronique du retrait, avec le chiffre qui
l'a tranche, vit en tete de `filmdec/world.go`.

**1.1 — LE MODE CTF SE LIT DANS LE FILM (arbitrage que le plan ne posait pas).** L'artefact est
construit sur les SEULS chunks : il ne connait ni la carte ni le `game_variant_name`, donc pas le
mode. Or la table d'emplacements du DRAPEAU, appliquee a un film d'un autre mode, rend n'importe
quoi (film Oddball : 1 470 « prises » et 994 « vols »). Premier discriminant essaye — le BURST DE
CAPTURE seul — **REFUTE par la mesure** : quatre films non-CTF en portent (Oddball 2, une colline
4, un Slayer 2). Ce qui tient est l'ACCORD DE TROIS SIGNAUX, tous du film :

	bursts > 0      captures > 0      captures <= bursts      vols > 0

L'inegalite est dans ce sens parce qu'un film TRONQUE sous-compte ses captures et ne les
sur-compte jamais (`64e8adfa` : 4 captures pour 5 bursts). **15 films de mode connu, 15 verdicts
justes, 0 faux positif et 0 film CTF ecarte** (6 CTF, 1 Oddball, 2 zones, 4 collines, 2 Slayer).
Le corpus est GELE dans `objectiveevents/flagfilm_test.go`, qui rejoue les quinze lignes SANS
film, et un second test verifie que chacune des quatre clauses est NECESSAIRE.

**1.1 — LES PORTAGES.** Regle en production (`replay/flag_carries*.go`), instrument sous garde
`OBJ_FILM` + `OBJ_REPO` qui l'appelle. Coordonnees MONDE obligatoires (les socles du catalogue
sont en metres ; la phase 0 travaillait en quanta).

| film | CTF | prises | portages | sans pont | sans piste | marqueur | simult.>2 | porteurs tues ambigus | retours ambigus |
|---|---|---|---|---|---|---|---|---|---|
| `64e8adfa` | oui (5/4/17) | 82 | 78 | **0** | 4 | 21/22 | 12 | 5 | 7 |
| `530820e5` | oui (3/3/10) | 33 | 30 | **0** | 3 | 3/7 | 0 | 4 | 1 |
| `53ce4390` | oui (3/3/13) | 34 | 29 | **0** | 5 | 13/13 | 0 | 0 | 3 |
| `000d5950` | **non** (0/0/0) | 0 | 0 | 0 | 0 | 0/0 | 0 | 0 | 0 |

**ZERO prise sans pont sur les trois films, `64e8adfa` compris** — c'est le gain direct de
l'item 1.0(a) : le film tronque qui rendait 0 slot sur 8 nomme desormais les huit. L'invariant de
couverture (`Balanced`) tient sur les quatre films. Le temoin non-CTF publie un calque VIDE.

**CONTROLE DU MARQUEUR : 37 / 42 = 88,1 %, SOUS LE SEUIL DE 90 %.** Le seuil n'est pas rebaisse et
l'item est `[!]`. Le NUMERATEUR est exactement celui de la phase 0 (21 + 3 + 13 = 37) : le marqueur
confirme les MEMES portages. C'est le DENOMINATEUR qui grandit — 42 contre 38 — et la cause est
nommee : la phase 0 fermait ses fenetres au dernier fait date du match, la production les prolonge
jusqu'a la FIN DE L'AXE quand rien ne les ferme. Quatre portages de plus contiennent donc une
image-cle, et aucun n'est confirme — ce qui est attendu, puisque le drapeau avait ete lache
depuis longtemps (le lacher volontaire n'est date par rien, cf. `flag_carries.go`). Le biais joue
CONTRE ce qu'on affirme, jamais en sa faveur, et il est desormais chiffre.

**LA SIMULTANEITE NE SE RESOUT PAS PAR LE PONT, ET C'ETAIT L'HYPOTHESE DU PLAN.** Le plan prevoyait
que le pont 1.0(a) leve les 6 depassements de `64e8adfa` ; la mesure en compte **12** avec la regle
de production, sur un film ou plus AUCUNE prise n'est sans pont. La cause n'etait donc pas
l'identite mais la DUREE des portages que rien ne ferme. Publie en incoherence comptee, comme le
plan l'exige en second recours.

**1.2 — LACHE / A LA MAISON, ET LE RETOUR AUTOMATIQUE : NEGATIF ECRIT.** Les trois etats sont
produits (`carried` / `dropped` / `home`), les socles `flag_spawn` du catalogue donnent l'equipe
proprietaire, et les deux drapeaux de chaque carte sortent distinctement :

| film | drapeau 0 | drapeau 1 |
|---|---|---|
| `64e8adfa` | equipe 1 : 111 spans (54 portes, 52 au sol, 5 a la base) | equipe 0 : 50 spans (24, 22, 4) |
| `530820e5` | equipe 1 : 22 spans (11, 7, 4) | equipe 0 : 44 spans (19, 19, 6) |
| `53ce4390` | equipe 0 : 23 spans (9, 9, 5) | equipe 1 : 44 spans (20, 19, 5) |

Duree au sol des laches REPRIS (frames de 100 ms) : `64e8adfa` mediane 28, p10 13, p90 358,
max 1 116 ; `530820e5` mediane 44, p10 18, p90 86, max 122 ; `53ce4390` mediane 30, p10 19,
p90 287, max 399. Laches JAMAIS repris : 16, 13 et 15, medianes 66, 90 et 103.
**AUCUN retour automatique ne se deduit de cette distribution** : de 1,3 s a 35,8 s entre p10 et
p90 sur le meme film, et un maximum a 111,6 s. Une minuterie fixe posee la-dessus renverrait des
drapeaux qui sont encore au sol. Le `dropped` court donc jusqu'a sa reprise, un `flag_returns` ou
la fin du match — un etat trop long, jamais une position inventee.

**Commits** : `26379f180` (1.0a), `cecab6d37` (1.0b), `102152b83` (discriminant + marqueur au
decodeur), puis le lot de regles et de mesure. Aucun push.
- 2026-08-18 — phase 1 items 1.0-1.2 CLOS et FUSIONNES (`94e4e4142`, correctifs de fusion `016860616`) ;
  gate 1 NON atteint a 88,1 % (37/42) parce que les portages NON FERMES sont prolonges jusqu'a la fin de
  l'axe (37/37 des FERMES confirmes par le marqueur) ; retour automatique : NEGATIF (aucune minuterie).
  **Arbitrage superviseur pour 1.3** (journal du 18/08) : publier `carried` (fermes) et `carried_open`
  (non fermes, borne haute = fin de l'axe, etat incertain explicite) ; gate 1.3 juge sur les FERMES,
  seuil 90 % inchange ; simultaneite > 2 portee par les seuls `carried_open` sinon incoherence publiee ;
  `dropped` court jusqu'a reprise / `flag_returns` / fin ; **schema 14** (la fusion tierce a pris 12 et
  13). Item 1.3 LANCE (agent Opus, worktree frere `../LevelUp-wt-objvivants2`, base `016860616`).

### 2026-08-18 — PHASE 1 ITEM 1.3 : LE CALQUE DU DRAPEAU EST PUBLIE (schema 14)

Worktree frere `LevelUp-wt-objvivants2`, branche `wt/objvivants2`, base `e86c17d74`.

**CE QUI ENTRE AU DOCUMENT.** `flagCarries` : une entree PAR DRAPEAU (l'objet, pas le portage) —
`{team, spans[]}` — et pour chaque span `{state, t0, t1, xuid|null, x, y}` sur l'axe de frames du
rejeu. `coverage.flagCarries` porte les denominateurs : verdict de mode et ses trois signaux,
prises de l'oracle, portages publies partages en FERMES / OUVERTS, rejets par cause (sans pont,
sans piste, hors fenetre), controle du marqueur (fermes ET ouverts, comptes a part), incoherences
(simultaneite, dont celle des seuls fermes ; porteurs tues ambigus ; retours ambigus), socles
connus. Films non-CTF : aucune entree, et la couverture dit lequel des deux silences.

**LE QUATRIEME ETAT EST LE RESULTAT, PAS UN CONFORT.** `carried` = un fait DATE ferme le portage ;
`carried_open` = rien ne le ferme, l'intervalle court jusqu'a la fin de l'axe et c'est une BORNE
HAUTE. La mesure sur les trois films CTF le justifie chiffre en main : le marqueur confirme 38 des
40 portages FERMES observables (95,0 %), et les 2 portages ouverts observables le sont aussi — la
population est trop petite pour trancher a elle seule, mais le partage, lui, se voit.

**GATE 1.3 (arbitrage du 18/08) : TENU.** Controle du marqueur sur les portages FERMES :
38 / 40 = 95,0 % (seuil >= 90 %, jamais rebaisse). Pour memoire, tous portages confondus :
40 / 42 = 95,2 % — c'est le 88,1 % de l'item 1.1, et l'ecart entre les deux EST ce que l'etat
`carried_open` publie.

**CONTROLE D'ANCRAGE — les portages publies sont CEUX de la phase 1.**

| film | CTF (bursts/captures/vols) | prises | portages | fermes | ouverts | sans pont | sans piste | marqueur FERMES | marqueur OUVERTS | simult. > 2 | dont entre FERMES | drapeaux publies |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `64e8adfa` | oui (5/4/17) | 82 | **78** | 78 | 0 | **0** | 4 | **22/22** | — | 12 | **12** | eq. 1 : 111 spans · eq. 0 : 50 |
| `530820e5` | oui (3/3/10) | 33 | **30** | 28 | 2 | **0** | 3 | 3/5 | **2/2** | 0 | 0 | eq. 1 : 22 spans · eq. 0 : 44 |
| `53ce4390` | oui (3/3/13) | 34 | **29** | 29 | 0 | **0** | 5 | **13/13** | — | 0 | 0 | eq. 0 : 23 spans · eq. 1 : 44 |
| `000d5950` | **non** (0/0/0) | 0 | 0 | 0 | 0 | 0 | 0 | — | — | 0 | 0 | **aucun** (calque vide, couverture publiee) |

Les PORTAGES, les rejets et les spans par drapeau sont EXACTEMENT ceux de l item 1.1 (78/30/29,
0 sans pont, 4/3/5 sans piste, 111+50 / 22+44 / 23+44 spans) : la publication n a rien change a
la regle. Ce qui a bouge, et il faut le dire, c est le CONTROLE : 40 portages confirmes sur 42
contre 37/42 a l item 1.1. La cause est nommee — l instrument de l item 1.1 posait le marqueur
avec SON pont bipede (lecture seule), la production utilise le sien, qui inclut les FERMETURES
(closures.go) et nomme donc plus de slots. Trois marques de plus trouvent leur porteur.

**SIMULTANEITE : L HYPOTHESE DE L ARBITRAGE NE TIENT PAS, ET C EST PUBLIE.** `530820e5` et
`53ce4390` n ont aucun depassement. `64e8adfa` en a 12 — et ce film n a AUCUN portage ouvert :
`closedOverlaps` vaut 12 lui aussi. La simultaneite n y oppose donc pas des portages incertains
mais des portages FERMES, c est-a-dire des faits dates entre eux. C est le second recours que
l arbitrage prevoyait : l incoherence est COMPTEE et PUBLIEE sous ses deux formes (`overlaps` et
`closedOverlaps`), le calque avertit au journal, et rien n est tu. Ce que le compte ne dit pas
encore, c est POURQUOI : les 5 `flag_carriers_killed` ambigus de ce film et ses 7 retours ambigus
designent la meme zone d ombre (des fins de portage que rien ne date), et la trancher demande un
canal qui date le lacher — condition de reprise au registre.

**CE QUI DESCEND JUSQU'AU CONSTRUCTEUR.** Le calque se publie SANS AUCUNE BASE : le pont slot
statborg -> xuid passe par les INSTANTS DE MORT (film seul). Seuls les SOCLES viennent d'ailleurs
— du catalogue versionne d'objectifs, joint par `map_id`, que `port.MatchFacts` transporte
desormais (`match_registry.map_id`, une colonne de plus au SELECT du lecteur de faits) et que
`replaybuild` resout en socles d'EQUIPE (le socle NEUTRE de chaque carte de CTF est ecarte : le
publier ferait apparaitre un troisieme drapeau qui n'existe pas dans le match).

**CONTRAT ET CLIENT.** `SchemaVersion` 13 -> 14 (chronique en tete de `document.go`, complete dans
`document_objectives_live.go`, justification exigee par `structure_test.go`) ;
`wantReplayDocumentFields` 34 -> 35 avec sa ligne de chronique et trois schemas de plus
(`FlagCarry`, `FlagSpan`, `FlagCarriesCoverage`) ; `api/openapi.yaml` REGENEREE (+137 lignes,
jamais editee a la main) ; `generated.ts` regenere (+60 lignes) ; `NULLABLE_ARRAYS` et
`NULLABLE_ARRAY_PATHS` gagnent `flagCarries` et `flagCarries[].spans`, et
`normalizeReplayDocument` comble les DEUX (le tableau imbrique compris). AUCUN rendu : la phase 3
reste entiere.

**GOLDEN.** Une seule ligne change dans `testdata/assembly_000d5950.golden` : `schema 13` ->
`schema 14`. Le film de reference est un Super Fiesta et le fixture fige ne porte aucune entree de
drapeau : le calque ne publie rien, sa couverture est ABSENTE (« personne n'a lu »), et c'est
exactement ce que le golden doit montrer.

**TEMOINS RE-CUITS** sous `C:\Users\Guillaume\Projects\LevelUp\data\cache\replays\halo_infinite\`,
un film par processus, aucune base ouverte (les faits de match viennent de fichiers JSON, avec le
`mapId` qui donne les socles) : `64e8adfa` 2 879 687 o, `530820e5` 1 617 220 o, `53ce4390`
2 540 731 o (absent du cache jusqu'ici), `000d5950` 2 402 255 o. Les quatre portent `schema 14`.

**UN COUT DE PRODUCTION MESURE, ET CORRIGE DANS LE LOT.** Le balayage du marqueur est une marche
COMPLETE des images-cles avec une fenetre glissante de 32 bits : sur `530820e5` la phase finale du
decodage passe de 57 s (cuisson de reference du lot A) a 136 s, quand la phase precedente n a
grossi que du facteur de charge de la machine (1,62x) — soit environ 45 s pour ce seul balayage,
un quart du temps de construction. Or il ne produit qu un CONTROLE, et sur un film d un autre mode
le calque est vide de toute facon : il ne tourne desormais QUE sur les films reconnus CTF, le
verdict se lisant dans ce que l appelant a deja fourni. Les temoins ci-dessus ont ete cuits AVANT
ce correctif ; il ne change aucune sortie (un film non-CTF n utilisait deja pas ses marques).

**LES GATES.** `go build ./...` 0, `go vet ./...` 0, `go test ./internal/analysis/...
./internal/replaybuild/... ./contracttest/... ./internal/archlint/...` 0 (19 paquets),
`golangci-lint run --new-from-merge-base=origin/main` 0 issue, `tsc -b --force` 0,
`npm run lint` 0 erreur (19 avertissements PREEXISTANTS), `vitest run src/features/match-replay
src/lib/replay` 0 (49 fichiers, 716 tests). Codes de retour au journal de gates du lot.

**Commits** : `b3e39edac` (calque publie + golden), `b0fb3e10f` (socles par map_id jusqu au
constructeur), `e527d7be5` (contrat + OpenAPI), `7d1295774` (frontiere web), `57d908989` (le
balayage du marqueur ne tourne que sur les films CTF), plus ce versement. Aucun push.
- 2026-08-18 — item 1.3 CLOS `[x]` et item 1.4 `[~]` (textes de journal et de registre au CR du
  lot ; `.ai/thought_log.md` et `REGISTRE_REPORTS.md` ne sont pas touches par cette branche).
  Gate 1.3 TENU (38/40 = 95,0 % sur les FERMES). Reserve publiee : la simultaneite > 2 de
  `64e8adfa` oppose des portages FERMES, ce que l arbitrage n avait pas prevu — comptee sous
  `closedOverlaps`, condition de reprise (un canal qui date le lacher) au registre.
