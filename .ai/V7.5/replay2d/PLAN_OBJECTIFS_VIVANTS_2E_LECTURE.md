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

- [ ] 1.0 Prealables : (a) `objectiveevents` — repli du pont par totaux vers le pont par INSTANTS de
      mort (l'instrument de la phase 0 le porte : 8/8 sur les 4 films, 8 accords / 0 desaccord),
      code de production + test (film tronque -> identites completes ; film complet -> identique a
      l'existant) ; (b) `filmdec` — retirer `World.SetHeldWeapon`/`HeldWeapon` et le champ de trace
      qui ne sert qu'a eux (verifier : aucun appelant hors tests ; les instruments de la phase 0.2
      qui les lisaient sont ajustes ou retires — pas de code mort ni de test qui teste du mort).
- [ ] 1.1 Portages CTF (films `64e8adfa`, `530820e5`, `53ce4390`) : un portage = [`FlagGrabs` ou
      `FlagSteals` du slot S a t0] -> fin = min(`FlagCaptures` de S, mort de S dans les pistes,
      `FlagCarriersKilled` dont la victime se resout a S) ; porteur = S -> xuid par le pont ; equipe du
      drapeau = socle `flag_spawn` le plus proche du point de grab (`FlagSteals` = adverse) ; le
      marqueur d'image-cle sert de CONTROLE (part des portages confirmes par >= 1 image-cle portant
      `0x00010005` sur S ; seuil ecrit >= 90 % des portages ayant une image-cle) ; simultaneite : au
      plus 2 portages ouverts a la fois — les depassements (`64e8adfa` : 6, dus aux 53 records sans
      pont) sont RESOLUS par le pont 1.0(a) ou publies comme incoherences comptees (jamais tus).
- [ ] 1.2 Drapeau lache / a la maison : fin de portage sans capture => `dropped` a la derniere
      position du porteur jusqu'au prochain grab, a un `FlagReturns`, ou au retour automatique
      (duree MESUREE : ecart median entre fin de portage sans reprise et prochain grab au socle) ;
      sinon `home` (marqueur statique existant, etat present/absent).
- [ ] 1.3 Document : `flagCarries` [{ team, spans [{ state `carried`|`dropped`|`home`, t0, t1, xuid|null,
      x, y }] }] + couverture (portages, confirmes par marqueur, incoherences, sans pont) ;
      `SchemaVersion` chronique (11 -> 12) ; contrat (`wantReplayDocumentFields` 33 -> 34, ligne de
      chronique), OpenAPI (`cmd/openapi-gen`), `generated.ts`, `NULLABLE_ARRAYS`/`normalizeReplayDocument`,
      goldens en connaissance de cause, temoins CTF re-cuits ; films non-CTF : tableau vide.
- [ ] 1.4 Registre : ligne R4 « objectifs vivants » -> porteur TRAITE (CTF) ; crane `[!]` (condition
      de reprise : discriminant externe, score par seconde de portage) ; canal delta CLOS (retire).

**Gate 1** : controle 1.1 >= 90 % ; chaque `FlagGrabs`/`FlagSteals` de l'oracle ouvre un span
`carried` du bon xuid ; contrat/OpenAPI/goldens verts ; incoherences publiees, pas masquees.

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

- `World.HeldWeapon(slot)` n'a aucun appelant (digest 2026-08-17) : soit la phase 0 le branche,
  soit il devient du code mort a supprimer (regle 0 code mort) — a trancher a la cloture de la
  phase 0, pas avant.
- La lignee R7 n'a ni ligne de registre ni entree de journal partage (vit dans 5 plans + commits
  de `wt/fusion-finale`) — a consigner par la session utilisateur a sa fusion.
- `HANDOFF_KEYFRAME_LIVE_CAPTURE.md` (04/07, « 205 entites keyframe decodees ») contredit R5-R7 sans
  retractation explicite — une note croisee est due au registre.
- (2026-08-18, phase 0) **`objectiveevents.SlotIdentity` tombe a 0/8 sur un film TRONQUE.** Son
  appariement compare les TOTAUX du film a ceux de `match_participants` ; un film dont le Theater
  ne rend pas la fin ne les atteint jamais. Mesure : 0/8 sur `64e8adfa` et `24dbb67d`, 8/8 sur
  `530820e5` et `53ce4390`. Le remplacant construit ici (appariement des INSTANTS de mort au fil
  des morts) rend 8/8 sur les quatre et concorde 8/8 avec l'existant la ou celui-ci repond. NON
  TRAITE dans le paquet de production : `SlotIdentity` reste tel quel.
- (2026-08-18, phase 0) **Le pont bipede laisse 53 records d'image-cle sans porteur sur
  `64e8adfa`** (16 % des records), et c'est le seul film ou le plafond structurel de deux porteurs
  au plus est depasse. Les deux faits vont probablement ensemble ; non instruit ici.
- (2026-08-18, phase 0) **La marche stateful rend 96 a 100 % de variantes NULLES sur les slots de
  joueur.** Sur 68 284 lectures d'arme tenue, moins de 1 600 tombent sur un bipede nomme, et la
  quasi-totalite valent `0x00000000`. C'est une mesure de la sante du decodeur delta, hors
  perimetre de ce lot mais utile a qui reprendra `World.HeldWeapon` ou le chemin i43-i46.

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
