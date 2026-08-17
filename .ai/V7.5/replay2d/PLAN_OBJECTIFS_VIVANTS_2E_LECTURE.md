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

- [ ] 0.1 **La famille du drapeau.** Sur les 3 films CTF : pour chaque `FlagGrabs`/`FlagSteals`
      d'un slot S a t, fenetre de portage = [t, min(capture de S, mort de S, `FlagCarriersKilled`
      dont la victime est S)] ; familles des loadouts d'images-cles de S DANS la fenetre moins
      familles de S HORS fenetre => famille candidate. Seuil : UN identifiant 32 bits, le MEME sur
      les 3 films, present dans >= 90 % des fenetres qui contiennent au moins une image-cle ; temoin :
      un slot non porteur tire au hasard sur les memes fenetres le porte dans <= 5 %. Denominateurs.
- [ ] 0.2 **Le canal delta `HeldWeapon`.** Instrument sous garde (`OBJ_FILM`), lecture seule, dans
      le paquet `filmdec` (accesseur de test ou hook — AUCUN changement de production avant
      validation) : latence entre le grab et la premiere image ou `HeldWeapon(S)` = famille du
      drapeau (mediane, p90) et entre la fin de portage et le retour a une autre famille. Seuil :
      >= 90 % des grabs apparies a <= 2 s ; sinon canal REFUTE (ecrit), le canal image-cle seul est
      retenu s'il a tenu 0.1.
- [ ] 0.3 **Nommer et generaliser.** Croiser la famille du drapeau avec le dictionnaire des tags
      `weap` (index killweapon, `weapon_labels`, sprite index) — attendu : le tag du drapeau ; le
      crane d'Oddball sur `24dbb67d` par signature structurelle (une famille portee par UN seul
      bipede a la fois, qui change de main) — seuil : <= 1 porteur sur >= 90 % des images ou la
      famille est portee ; les evenements approx `th=10` comme controle (accord grossier publie).
- [ ] 0.4 Publier au journal du plan : famille(s), canal retenu, latences, denominateurs.

**Gate 0** : famille du drapeau identifiee (0.1) ET un canal valide (0.2 ou 0.1) => phase 1 ;
sinon NEGATIF ecrit, phase 1 `[!]`, passage direct a la phase 2.

### Phase 1 — PUBLIER les objets portes (schema +1, chronique)

- [ ] 1.1 Document : `objectiveObjects` [{ kind `flag`|`ball`, team (drapeau : equipe du socle),
      spans [{ state `carried`|`dropped`|`home`, t0, t1, slot|-1, x, y }] }] + couverture (nombre
      d'evenements oracle vs spans) ; retour automatique mesure (decision 3) ; `SchemaVersion`
      chronique ; contrat (`wantReplayDocumentFields` 31->32 avec la ligne de chronique),
      OpenAPI, `generated.ts`, `NULLABLE_ARRAYS`, goldens, temoins re-cuits.
- [ ] 1.2 Controle : chaque `FlagGrabs`/`FlagSteals` de l'oracle tombe dans un span `carried` du
      bon slot (± 1 image en delta, ± 1 image-cle sinon) — taux publie, seuil >= 90 %.
- [ ] 1.3 Registre : la ligne R4 « objectifs vivants » passe TRAITEE (porteur) ; l'objet lui-meme
      reste NON decode (ecrit).

**Gate 1** : contrat/OpenAPI/goldens verts, controle 1.2 tenu.

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

- [ ] 3.1 `objectivesLayer.ts` : icone drapeau/crane collee au marqueur du porteur (sprite existant
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

## Journal du plan (avancement, source de verite pour la reprise)

- 2026-08-17 — plan ecrit (digest R4/R7 verifie sur pieces par un agent de lecture). EN ATTENTE :
  fin de l'item 6, fusion utilisateur de `wt/fusion-finale` + `wt/poses-revue-fix`. Non lance.
