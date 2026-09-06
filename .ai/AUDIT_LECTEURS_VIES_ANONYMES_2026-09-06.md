# Audit — « un slot = une piste nommée », les lecteurs restants — 2026-09-06

Audit adversarial de code EXISTANT (pas un diff). **Rien n'a été corrigé** : ce registre est la
seule sortie, conformément au skill `adversarial-audit` (« l'audit ne corrige pas »). Branche
`feat/v2-audit-vies`, worktree détaché sur `feat/v2-durees` (`7e5c454bc`, schéma 45).

## Cadrage

**Périmètre** — `apps/go-api/internal/analysis/replay/` (116 fichiers de production),
`apps/go-api/internal/analysis/objectiveevents/`, `apps/go-api/internal/replaybuild/`,
`apps/go-api/internal/service/replayview/`, `apps/go-api/internal/domain/replaydoc/`, et
`apps/web/src/features/match-replay/` — plus son foyer partagé `apps/web/src/lib/replay/`, dont
le périmètre dépend directement et où vivent deux des constats web.

**Axe UNIQUE** — l'hypothèse « un slot = une piste nommée », fausse depuis le schéma 36
(`48cf4905d`, 2026-09-02) : une `Track` publiée est UNE VIE, un slot recyclé en publie
plusieurs, et une vie que le fil des morts ne nomme pas reste ANONYME. Six formes cherchées :
(a) index bâti sur `XUID != ""` ; (b) sélection de « LA piste du slot » au singulier ;
(c) bornage d'une mesure à UNE vie ; (d) jointure joueur↔piste par xuid seul, sans repli sur le
pont canonique ; (e) dénominateur par SLOT en face d'un numérateur par VIE ; (f) compteur qui
classe en « absent » un cas où l'identité est INCONNUE.

**Méthode** — cinq auditeurs à contexte frais, un par tranche de fichiers, même axe unique,
mêmes règles de recevabilité, aveugles les uns aux autres ; plus une passe personnelle sur les
fichiers d'assemblage non attribués (`build.go`, `document.go`, `identity.go`, `lives.go`,
`owners.go`, `t0_film.go`, `published_tracks.go`, `coverage.go`). **Chaque constat P0/P1 a été
rouvert et vérifié sur pièces par le superviseur**, avec recherche active de ce qui le
réfuterait : un appelant qui pré-filtre, un test qui couvre, une garantie amont.

**Doctrine de référence** — CLAUDE.md règle 3 (« jamais d'erreur avalée : logger AVANT toute
dégradation best-effort »), règle 6 (« ≤ 2 copies, sinon helper ET garde-rail »), anti-patterns
n°1 « dead code museum », n°8 « factorisation abandonnée », n°10 « swallowed error » ; invariant
produit du chantier : **une lecture VRAIE du film ne doit jamais être jetée parce qu'un nom
manque ; un nom manquant est « inconnu », jamais « absent »**.

**Dette assumée, non remontée** — les six lecteurs déjà rattrapés (`skull_carries.go`
`carrierPresence`/`gate`, `bomb_carries.go`, `flag_carrier_tracks.go` `tracksByXUID`,
`equipment_episodes.go` `spanFor`, le pont `replaybuild` → `FlagInput.Identity`,
`usage_summary.go`) ; la baseline lint ; le plafond `deathInstantMin = 3` du pont par morts ;
les CTF multi-manche muets ; les cas où le pont est muet ET où le code s'abstient explicitement.

### La population déclenchante, mesurée sur le parc

Le parc local (111 artefacts, lecture seule, aucune cuisson) ne compte que **deux artefacts
cuits depuis le schéma 36** — et **tous les deux** portent des slots recyclés et des vies
anonymes :

| artefact | schéma | pistes | slots | vies anonymes |
|---|---|---|---|---|
| `1b2d9e08` | 38 | 94 | 91 | 9 |
| `1cd3848a` | 38 | 119 | 104 | 21 |

Les 109 autres sont antérieurs (schémas 6 à 34, une piste = un slot). Les mesures de perte
citées plus bas viennent donc d'artefacts PRÉ-36 : elles établissent que le chemin de code est
CHAUD, pas que le schéma 36 l'a créé. Le schéma 36 en ÉLARGIT le déclencheur — il multiplie les
pistes anonymes par slot.

---

## Constats retenus

**2 P0, 7 P1, 5 P2.**

### [P0-1] `samplesByXUID` n'indexe que les pistes NOMMÉES — le défaut corrigé au schéma 45 pour le drapeau, intact sur le chemin des zones

- **Où** : `apps/go-api/internal/analysis/replay/zone_attribution.go:207-214`, consommé en
  production par `AttributeZones` (`zone_attribution.go:140`) ← `buildZoneStates`
  (`zone_states.go:139`) ← `attachZoneStates` (`build_zones.go:66`) ← `build.go:249`.
- **Règle enfreinte** : forme (a) de l'axe, et l'invariant produit. Le dépôt a DÉJÀ statué le
  cas identique — `document.go:831-834` (chronique du schéma 45) : « `tracksByXUID` n'indexait
  que les pistes NOMMÉES : une prise que seule la vie ANONYME du porteur recouvre sortait
  `NoTrack`. `bcb6d393` : 9 prises sur 16 perdues, `carries` 16 → 7. L'identité vient du PONT
  canonique (`ResolveSlotXUID`) ». `samplesByXUID` est la MÊME construction, dans le fichier
  voisin, sans le pont. Son commentaire (`:204-206`) oppose l'argument que le correctif du
  drapeau a précisément écarté : lire le pont canonique n'est pas « rattacher une position
  anonyme à un joueur », c'est lire la table que `flag_carrier_tracks.go:25-29` désigne comme
  non-déductive, avec sa propre règle de collision.
- **Conséquence** : une capture de zone tombant pendant une vie ANONYME d'un slot par ailleurs
  nommé ne trouve plus d'échantillon à ≤ 2 frames (`DefaultMaxGapFrames = 2`,
  `zone_attribution.go:48`) → `cov.NoPosition++` → la capture ne vote plus (`zonePairsOf`,
  `zone_states.go:270-279`) → la zone perd sa voix d'appariement jauge↔propriétaire
  (`zone_states_owner.go:158-197`) et n'est **pas publiée**. Quand plus aucune capture n'est
  attribuée, `buildZoneStates` rend `nil` hors mode à colline (`zone_states.go:142-152`) : c'est
  le calque `zoneStates` ENTIER — propriétaire, jauge en direct, lettres A/B/C — qui disparaît
  de l'artefact servi. Films déclencheurs : modes à zones (Bastion, Colline) sur film à slots
  recyclés ou à forte anonymie.
- **Mesure sur le parc** (captures non attribuées ; la cause n'est enregistrée nulle part,
  cf. [P1-3]) : `696a9d7c` 77 captures → 66 attribuées (**11 perdues**), `7344d24f` 71 → 59
  (**12**), `af13e2b2` 19 → 14 (**5**). Ces trois films portent respectivement 3, 5 et 5 pistes
  anonymes.
- **Reproduction** :
  `grep -rn 'tracksByXUID(\|samplesByXUID(\|published\[tr\.XUID\]' apps/go-api/internal/analysis/replay/*.go | grep -v _test`
  — deux sites passent `slotXUID` (`flag_carries.go`, `flag_objects.go`), deux ne le font pas
  (`zone_attribution.go:140`, `objectives.go:117`).
- **Vérification adverse — TIENT.** Cherché un pré-filtre, un test, une garantie amont.
  (1) `c.actions` vient de `doc.Objectives`, dont chaque action a un auteur ayant au moins une
  piste nommée — ce qui garantit que le JOUEUR existe, jamais que la vie portant la capture soit
  nommée. (2) `TestUneVieAnonymeNeSertAPersonne` (`zone_attribution_test.go:138-157`) ne réfute
  pas : il ne fournit AUCUN pont, et ses deux assertions restent vraies avec un repli sur pont
  vide — c'est même la contre-épreuve qu'un correctif devra conserver, exactement comme
  `TestFlagCarriesVieAnonymeSansPontResteEcartee`. (3) Le pont est disponible au site d'appel :
  `own` est en portée depuis `build.go:63`, 186 lignes avant `attachZoneStates`, et le patron de
  descente existe déjà (`flagCarryCtx.slotXUID`, `flag_carries.go:127`).

### [P0-2] Le report de lecture des fiches n'est borné à AUCUNE vie : sur un slot recyclé, la fiche affiche les armes de la vie PRÉCÉDENTE

- **Où** : `apps/web/src/lib/replay/rosterLogic.ts:580-598` (`nearestReading`, boucle 587-596).
  Consommateurs : `rosterLogic.ts:550` (`loadoutAt`), `rosterLogic.ts:627` (`abilityAt`),
  `apps/web/src/features/match-replay/model/inventoryReading.ts:90` (`inventoryAt`).
  Site frère, même défaut : `apps/web/src/lib/replay/changeRefine.ts:131-149`
  (`refineAbilityReading` — dernier `equipmentChange` du SLOT, sans borne de vie).
- **Règle enfreinte** : formes (b) et (c) de l'axe. Le code écrit la fausse prémisse noir sur
  blanc (`rosterLogic.ts:573-575`) : « la recherche porte sur le SLOT (…) Un slot est réattribué
  à chaque réapparition : c'est ce qui rend le report SÛR, une lecture ne peut pas franchir une
  mort ». La réattribution est ce qui rend le report DANGEREUX, pas sûr. Le remède existe dans le
  dépôt et n'est pas appelé : `lifeOfSlotAt`
  (`apps/web/src/features/match-replay/model/livesPosition.ts:121`), déjà employé par
  `fireMark.ts:49`, `grappleLayer.ts:61`, `thrusterDashFx.ts:141`, `shotFx.ts:97`.
- **Conséquence à l'écran** : `nearestReading` préfère TOUJOURS `best` (la lecture passée la plus
  proche, quel que soit son âge) à `ahead` — le repli protecteur documenté est donc inerte dès
  que le slot porte une vie antérieure. Sur toute fiche joueur (ReplayTeams → ReplayWeaponsRow /
  ReplayInventoryRow / ReplayAbilityCell), après une réapparition sur un slot recyclé, la rangée
  d'armes montre les DEUX armes tenues à la mort précédente, mise en valeur « en main »
  comprise, et l'infobulle affirme « Lu il y a X s » — jamais « à venir ». Cas le plus grave,
  MULTI-MANCHE ou REMPLAÇANT : le slot est réattribué à un AUTRE joueur, et la fiche de B affiche
  les armes, les munitions et la capacité de A. Symétrique sur la capacité : un `spent` de la vie
  précédente fait DISPARAÎTRE la vignette d'une vie neuve (`changeRefine.ts:148` → `null`).
- **Reproduction** :
  `grep -n 's.slot !== slot\|c.slot !== slot' apps/web/src/lib/replay/rosterLogic.ts apps/web/src/lib/replay/changeRefine.ts apps/web/src/features/match-replay/model/inventoryReading.ts`
- **Vérification adverse — TIENT.** (1) `ReplayTeams.tsx:243/257/272` appelle bien avec
  `state.life.slot`, la vie couvrante : la fuite est dans la LECTURE, pas dans le choix de la
  vie. (2) `rosterLogic.test.ts` `describe('loadoutAt')` (l. 422-458) ne monte AUCUN cas
  multi-vies sur un même slot ; le test de la ligne 440 verrouille au contraire la fausse
  prémisse. (3) `rosterLogic.guard.test.ts` ne garde que l'unicité d'écriture du repli, pas sa
  justesse. (4) `refineWeaponsReading` ne rattrape rien : il refuse les prises sur emplacement
  vide (`changeRefine.ts:31`), c'est-à-dire exactement le cas du spawn.

### [P1-1] `dropUnpublishedActions` supprime une action d'objectif VRAIE et IDENTIFIÉE parce qu'aucune piste ne porte le nom de son auteur — et la classe « sans trajectoire publiée »

- **Où** : `apps/go-api/internal/analysis/replay/objectives.go:113-119` (index `published` bâti
  sur `tr.XUID != ""`) et `:120-128` (suppression + `cov.Unpublished++`), appelé en `:142` depuis
  `attachObjectiveActions` ← `build.go:105`.
- **Règle enfreinte** : formes (a) et (f), plus CLAUDE.md règle 6 / anti-pattern n°8. **Onze
  filtres « piste publiée » du paquet cadencent sur le SLOT** — `abilities.go:117`,
  `build.go:409`, `document_ability_charges.go:156`, `document_equipment_changes.go:159`,
  `document_translocations.go:95`, `grenades.go:230`, `grenade_reads.go:134`, `inventory.go:223`,
  `loadouts.go:103`, `vehicle_shots.go:85`, via le foyer canonique `keepOfPublishedTracks`.
  **Deux seulement cadencent sur le XUID NOMMÉ** : `objectives.go:117` et `neutral_deaths.go:29`.
  Ce sont aussi les deux seuls à ne pas passer par le helper — et le garde-rail censé l'interdire
  ne les voit pas : `published_tracks_guard_test.go:22` filtre sur
  `\w+\[\w+\.Slot\]\s*=\s*true`, un motif qu'une map keyée par chaîne ne déclenche jamais. La
  divergence « INVISIBLE » que le garde-rail existe pour empêcher (son en-tête, l. 14-16) est
  déjà là.
- **Conséquence** : un joueur dont AUCUNE vie n'est nommée perd TOUTES ses actions d'objectif.
  Le contrat `Track.XUID` (`document.go:1348-1352`) nomme la population : « Vide quand la vie n'a
  pas été nommée — 15 vies sur 105 sur le film de référence, dont 4 antérieures au début réel du
  match et **6 survivants de fin de partie**, que le film ne clôt par aucun événement ». Trois
  consommateurs perdent la donnée, dont deux n'ont JAMAIS eu besoin d'une trajectoire :
  1. **le SON** — `objectiveSoundEvents`
     (`apps/web/src/features/match-replay/sound/objectiveSound.ts:159-168`) ne lit que `a.t` et
     `a.stat`, aucune position : la capture de drapeau, la prise de zone ou l'explosion de ce
     joueur ne sonne JAMAIS ;
  2. **la garde de l'armement** — `bombDetonationTimes(doc.Objectives)` (`bomb_armings.go:186`)
     n'a besoin que de l'instant ; une explosion supprimée sort du dénominateur `cov.Detonations`
     et la confrontation tout-ou-rien, dont le principe est qu'« une seule explosion orpheline
     retient le calque ENTIER » (`bomb_armings.go:46-49`), ne peut plus être retenue par elle ;
  3. **les zones** — `zoneCapturesOf(doc.Objectives)` (`zone_states.go:137`), qui cumule avec
     [P0-1] : deux portes d'anonymat en série sur la même donnée.

  La couverture publiée MENT sur la cause : `LayerCoverage.Unpublished` est documenté « son slot
  n'a pas de trajectoire publiée (track trop courte) » (`coverage.go:48-50`), alors que la
  trajectoire EST publiée — elle n'est simplement pas nommée.
- **Mesure sur le parc** : 7 artefacts sur 111 portent `coverage.objectives.unpublished > 0` —
  `3372e7eb` **35 actions sur 76 (46 %)**, `06dfe6d9` 20/213, `cf040013` 14/44, `b8d1fe0c` 8/85,
  `66aa5f0b` 8/64, `82f29378` 4/9, `846044ba` 1/38.
- **Reproduction** :
  `grep -rn 'published\[.*\.Slot\]\|published\[.*XUID\]' apps/go-api/internal/analysis/replay/ --include=*.go | grep -v _test`
- **Vérification adverse — TIENT, avec une réserve écrite.** (1) Appelant unique non-test, aucun
  pré-filtre : `buildObjectiveActions` ne rejette en amont que l'action SANS identité
  (`e.XUID == ""` → `NoSlot`, `objectives.go:80-85`), cas différent — et d'ailleurs mort en
  production, cf. [P1-2]. (2) Aucun test ne couvre le comportement multi-vies ;
  `bomb_stats_wiring_test.go:44` et `:93` s'APPUIENT dessus sans le juger. (3) `own.SlotXUID` est
  en portée depuis `build.go:63` et n'est pas passé, alors que les quatre calques frères
  (`attachFlagCarries`, `attachVipCrown`, `attachSkullCarries`, `attachBombCarries`) le
  reçoivent.
  **RÉSERVE, écrite parce qu'elle borne le correctif** : `own.SlotXUID` ne récupère pas tous les
  cas. Il ne nomme un slot que si une vie nommée en sort (`ownersFromLives`, `lives.go:215-237`)
  OU si une FERMETURE l'a attribué (`extendSlotXUID`, `owners.go:171-188`) — cette seconde voie
  couvrant des slots dont AUCUNE vie n'est nommée (`closureReport.closedLife` peut valoir −1),
  ainsi que le cas où l'unique vie nommée n'est pas PUBLIÉE (mécanique `minPoints`, faits n°3-5
  du balayage du parc). Sur les 7 artefacts mesurés, les xuid écartés n'apparaissent dans aucune
  piste : leur pont est probablement muet, cas exempté. **Le défaut est que le code ne distingue
  pas les deux** — il applique le rejet du pont muet à des slots que le pont nomme.

### [P1-2] La couverture des actions d'objectif est aveugle AUX DEUX BOUTS : le dénominateur ne compte que les rescapés, et la seule catégorie qui bouge n'a pas d'alarme

Deux défauts distincts, remontés ensemble parce qu'ils se referment l'un sur l'autre.

- **Où, amont** : `apps/go-api/internal/analysis/objectiveevents/slotidentity.go:158-169`
  (`IdentifyNamedEventsByRound` : `xuid := identity.At(...); if xuid == "" { continue }`, sans
  compteur de retour) → `apps/go-api/internal/replaybuild/matchfacts.go:272` (seul le rescapé
  traverse) → `apps/go-api/internal/replaybuild/replaybuild.go:249`
  (`Objectives: stats.objectives`, **unique producteur de production** de
  `replay.Options.Objectives`) → `apps/go-api/internal/analysis/replay/objectives.go:73`
  (`cov := LayerCoverage{Available: len(evs)}`).
- **Où, aval** : `apps/go-api/internal/analysis/replay/coverage.go:86-101` — `warnIfLossy` ne
  parcourt que `{NoSlot, Ambiguous, OutOfWindow}` ; `Unpublished` (`coverage.go:50`) n'y est pas.
- **Règle enfreinte** : formes (e) et (f), CLAUDE.md règle 3, et anti-pattern n°1 (« dead code
  museum, le pire : avec des tests verts qui entretiennent l'illusion »). Le code énonce
  lui-même la règle qu'il n'applique pas — `coverage.go:36-38` : « Available est le nombre
  d'événements DISPONIBLES DANS LE FILM pour ce calque, le dénominateur sans lequel un compte de
  rattachés ne se juge pas » ; `coverage.go:105-107` : « ELLE EST PUBLIÉE DANS LE DOCUMENT, pas
  seulement journalisée : c'est la différence entre un décodeur qui SAIT ce qu'il perd et un
  écran qui le MONTRE » ; `slotidentity.go:130-135` : « publier des événements attribués sans
  dire combien ne l'ont pas été laisserait croire à l'exhaustivité ».
- **Conséquence, en trois effets qui se composent** :
  1. `coverage.objectives.available` compte les actions DÉJÀ identifiées, jamais celles que le
     statborg portait : le rapport attaché/disponible se lit ~100 % sur un calque partiel, et
     `Balanced()` (`coverage.go:55-56`, « une somme fausse signale une fuite ») est vrai à vide —
     la fuite est EN AMONT du point d'équilibre.
  2. `coverage.objectives.noSlot` — le seul champ du contrat public prévu pour dire « le pont ne
     couvre pas ce joueur » (`replaydoc/coverage.go:46`, servi par
     `replayview/convert_coverage.go`) — est structurellement inatteignable. **Vérifié sur les
     111 artefacts du parc : `noSlot` vaut 0 partout, sans une seule exception.**
  3. `warnIfLossy("objectifs")` ne teste que `NoSlot / Ambiguous / OutOfWindow` : l'alarme
     « slotIntrouvable » ne peut jamais se déclencher, et la catégorie qui BOUGE réellement —
     `Unpublished`, celle que [P1-1] alimente — n'a aucun seuil. Sur `3372e7eb`, 46 % des actions
     disparaissent **sans une seule ligne de journal**, alors que le seuil est de 10 %
     (`rejectSampleThreshold`, `coverage.go:31`).

  Le seul témoin de la perte amont est un `slog` non durable (`matchfacts.go:273-274`,
  « nommees » / « identifiees ») : il ne voyage ni dans l'artefact ni dans le contrat. Mesure du
  dépôt lui-même sur `c0a82e88` : 17 actions nommées, 12 identifiées — 5 perdues, avec un
  `coverage.objectives` qui aurait annoncé 12/12 et 0 `noSlot`.
- **Le test qui entretient l'illusion** : `objectives_test.go:72-84`
  (`TestBuildObjectiveActionsRefusesUnidentified`) fabrique un `IdentifiedEvent{XUID: ""}` que la
  production ne peut PAS produire, et garde ainsi verte une branche morte (`objectives.go:80-85`).
- **Reproduction** :
  `for f in data/cache/replays/halo_infinite/*.json; do jq -r 'select((.coverage.objectives.noSlot//0)>0)|input_filename' "$f"; done`
  (sortie vide sur 111 artefacts) puis
  `grep -n -A16 'func (c LayerCoverage) warnIfLossy' apps/go-api/internal/analysis/replay/coverage.go`
- **Vérification adverse — TIENT.** (1) Grep exhaustif de `Objectives:` sur tout le module :
  `replaybuild.go:249` est le seul producteur de production ; `cmd/zone-attribution` est un outil
  de mesure. (2) Grep exhaustif des écritures de `NoSlot` pour ce calque : aucune autre.
  (3) Cherché un journal de repli ailleurs : `build.go:382-389` journalise `Unpublished` pour les
  tirs seulement (`tirsNonPublies`), jamais pour les objectifs ni les grenades.

### [P1-3] La couverture d'attribution des zones est jetée à la ligne d'appel : la cause d'une capture perdue n'est enregistrée NULLE PART

- **Où** : `apps/go-api/internal/analysis/replay/zone_states.go:139` —
  `att, _ := AttributeZones(...)`.
- **Règle enfreinte** : anti-pattern n°10 (« `_ = f()` / `continue` sur erreur sans log ni
  compteur ») et règle 3. `ZoneCoverage` porte `NoPosition`, `Outside` et `Ambiguous`
  (`zone_attribution.go:83-99`) avec un invariant testé, et le commentaire de `NoPosition`
  (`:86-88`) nomme lui-même la cause d'axe : « **vie non nommée**, joueur sans track publiée,
  trou d'échantillonnage ». Ce compteur est jeté, et `ZonesCoverage` — la structure PUBLIÉE
  (`document_zones.go:234-318`) — n'a aucun champ pour l'accueillir.
- **Conséquence** : sur les trois films mesurés au [P0-1], 11, 12 et 5 captures disparaissent
  **sans qu'aucune ligne du journal ni aucun champ de l'artefact ne dise pourquoi**. Il est
  impossible, sur un artefact du parc, de distinguer « le joueur n'était pas dans la zone »
  (`Outside`, une mesure) de « sa vie n'est pas nommée » (`NoPosition`, une ignorance) — la
  distinction même sur laquelle `coverage.go` fonde toute sa doctrine.
- **Reproduction** :
  `grep -n 'AttributeZones' apps/go-api/internal/analysis/replay/zone_states.go; grep -n 'json:"' apps/go-api/internal/analysis/replay/document_zones.go`
- **Vérification adverse — TIENT.** Cherché un second consommateur qui la sauverait : le seul
  autre appelant de `AttributeZones` est `cmd/zone-attribution/report.go:200`, une CLI de
  recherche hors production.

### [P1-4] Les deux fermetures bornent leur garde-fou sur le SLOT ENTIER alors qu'elles ont DÉSIGNÉ une vie

- **Où** : `apps/go-api/internal/analysis/replay/closures_respawn.go:147-152`
  (`overlapsNamedLife` : `cand := tracks[slot].pts`, puis `from, to` = premier et dernier point
  du slot) et `apps/go-api/internal/analysis/replay/closures.go:291-296` (`bodyExtendsShooter` :
  `from := cand[0].TimestampUS`).
- **Règle enfreinte** : forme (c) prise à l'envers — la mesure est prise sur TOUT le slot là où
  le code vient de désigner UNE vie. `slotTrack.pts` (`indexBySlot`, `shots.go:120`) est le nuage
  COMPLET du slot, toutes vies confondues. L'indice de la vie désignée est en portée aux deux
  sites d'appel : `lives[vies[0]].slot` (`closures_respawn.go:83`, la ligne juste au-dessus de
  l'appel) et `c.life[slot]` (`closures.go:229`, cinq lignes sous l'appel `:224`). Le fichier
  pose lui-même la règle : « `closedLife` retient (…) l'INDICE de la vie que la fermeture a
  désignée — jamais le slot seul » (`closures.go:83-84`).
- **Conséquence** : sur un slot portant deux vies éloignées (30 s puis 600 s), l'intervalle testé
  couvre 570 s ; n'importe quelle vie nommée du joueur candidat y tombe, `rep.refused++`, et
  `owner[slot]` n'est pas posé. Côté fermeture A, `from` est le début de la PREMIÈRE vie du slot,
  donc « tous les corps connus du tireur s'achèvent avant » échoue dès qu'un corps nommé du
  tireur finit entre les deux vies. Résultat servi : la vie reste ANONYME, le pont n'a pas ce
  slot, et tous les tirs du slot sortent en `coverage.shots.noSlot` sans être dessinés — la perte
  exacte que `closures.go` existe pour réparer (son en-tête : « 63 à 92 % des tirs perdus tombent
  dans une vie non nommée »). Films déclencheurs : tout film à slot multi-vies — 174 vies pour
  160 slots sur `d9781168`, dont 18 slots sans aucune vie nommée.
- **Reproduction** :
  `grep -n 'lives\[vies\[0\]\]\|c\.life\[slot\]\|tracks\[slot\]\.pts\|tracks\[s\]\.pts' apps/go-api/internal/analysis/replay/closures.go apps/go-api/internal/analysis/replay/closures_respawn.go`
- **Vérification adverse — TIENT.** (1) `freeLives` (`closures.go:319`) n'exclut que les slots
  DÉJÀ au pont, jamais les slots à plusieurs vies libres : aucun pré-filtre. (2) Aucun test ne
  cite `bodyExtendsShooter` ni `overlapsNamedLife`. (3) Les deux tests multi-vies existants —
  `TestFermetureANeTranchePasEntreDeuxViesDuMemeSlot` (`closures_test.go:394`) et
  `TestFermetureBNommeLaVieDesigneeQuandLeSlotEnPorteDeux` (`:346`) — placent la vie désignée en
  PREMIÈRE position du slot, avec un `owner` sans corps intercalaire : les deux configurations où
  le bornage large donne par accident la bonne réponse.

### [P1-5] L'occupant d'un véhicule est nommé par le SEUL pont par slot, jamais par la vie à bord

- **Où** : `apps/go-api/internal/analysis/replay/vehicle_rides.go:280` (`vehicleRideOf`) et
  `apps/go-api/internal/analysis/replay/vehicle_rides_events.go:274` (`vehicleRideFromEpisode`) —
  tous deux `in.own.SlotXUID[...]`.
- **Règle enfreinte** : formes (b) et (d). `SlotXUID` est une identité UNIQUE par slot pour tout
  le match : `ownersFromLives` (`lives.go:229-234`) garde la PREMIÈRE vie nommée et jette les
  suivantes en collision (`continue`), et `buildLifeSpans` trie par slot puis chronologiquement —
  c'est donc le PREMIER occupant. Le motif corrigé du dépôt est « par vie d'abord, pont en
  repli » (`tracksByXUID`, `flag_carrier_tracks.go:30-44`) ; ici la première moitié manque
  entièrement. La table par vie est pourtant dans le MÊME objet : `OwnerReport.lives`
  (`owners.go:97`), passé à `attachVehicles` (`build.go:218`) et déjà utilisé pour nommer les
  pistes (`build.go:67`) et les charges/impulsions (`build.go:341`, `:363`).
- **Conséquence** : sur un slot bipède recyclé entre deux joueurs nommés, l'épisode d'occupation
  est publié avec le xuid du PREMIER, quel que soit l'instant. Or `VehicleRide.XUID` est ce qui
  donne sa COULEUR au véhicule — `document_vehicles.go:163-165` : « C EST LUI QUI DONNE SA
  COULEUR AU VEHICULE (…) le client joint xuid → equipe → couleur ». Le sprite prend l'équipe du
  mauvais joueur et la fiche crédite le mauvais conducteur, là où le contrat prévoit
  explicitement le silence (`:161-163` : « Vide quand le pont n'a pas nommé ce slot : l'épisode
  reste publié — le véhicule EST occupé, c'est son occupant qui est inconnu »). Le document se
  contredit alors lui-même : la `Track` du même slot au même instant porte `Bot` ou reste
  anonyme pendant que la ride affirme un humain.
- **Reproduction** :
  `grep -rn 'in\.own\.SlotXUID\[\|own\.lives' apps/go-api/internal/analysis/replay/vehicle_rides.go apps/go-api/internal/analysis/replay/vehicle_rides_events.go apps/go-api/internal/analysis/replay/build.go`
- **Vérification adverse — TIENT sur le mécanisme, PAS ENCORE OBSERVÉ dans le parc.**
  (1) Le déclencheur est exactement `coverage.bridge.slotCollisions > 0` : **9 artefacts du parc**
  le portent (`00ba2e1c`, `04023f8a`, `06dfe6d9`, `5dfdc63b`, `879a4dba`, `daaa17d6` à 1 ;
  `53ce4390` à 2 ; `084a804d`, `11de8353` à 3). (2) **Réserve honnête** : aucun de ces neuf ne
  porte de `vehicleRides` — le calque véhicules est postérieur à ces artefacts — donc la
  conjonction n'est pas encore observable ; seul le mécanisme est établi sur pièces. (3) Les
  tests `build_vehicles_test.go:161` et `vehicle_rides_events_test.go:262` verrouillent le
  nommage sur un slot MONO-identité : une résolution par vie avec repli sur le pont y rendrait la
  même valeur, ils ne réfutent rien.

### [P1-6] Deux bots successifs sur le MÊME siège s'écrasent dans une `map[siège]nom` — et le mécanisme de relais qui devait les départager est neutralisé

- **Où** : `apps/go-api/internal/analysis/replay/identity.go:209-213`
  (`nameByIndex[b.FilmIndex] = b.Name` : le dernier balayé gagne, sans ordre garanti).
  Donnée d'entrée : `apps/go-api/internal/replaybuild/replaybuild.go:462-478` (`botIdentities`,
  qui émet correctement DEUX entrées de même `FilmIndex`) et `:485-520` (`botSuccessions`, dont
  la sortie est ensuite rendue inopérante).
- **Règle enfreinte** : forme (b) mot pour mot. Contredit frontalement la doctrine écrite quatre
  fonctions plus haut, `identity.go:163-168` : « **DEUX BOTS PEUVENT PARTAGER UN INDEX, et c'est
  mesuré** (RE_LOG 7ter.62 : "343 Aloysius" puis "343 PardonMy", les deux déclarant slot=8) — des
  remplaçants SUCCESSIFS sur le même siège de réplication. Ils entrent tous les deux : le nom les
  différencie, l'index dit le siège. » `buildRoster` respecte la règle ; `nameBotTracks`, à
  40 lignes de là, la viole.
- **Conséquence** : (1) toutes les vies anonymes du slot que le pont attribue à ce siège
  reçoivent le nom du DERNIER bot dans l'ordre de balayage — arbitraire, sans rapport avec la
  chronologie du remplacement : le pion du rejeu affiche un gamertag faux sur des pans entiers du
  match. (2) `nameBotTracks` est appelé AVANT `attributeSuccessions` (`build.go:70` puis `:73`)
  et pose `Track.Bot`, or `candidateIn` écarte toute piste dont `Bot != ""`
  (`successions.go:96`) : la chaîne de relais alimentée par `botSuccessions` trouve alors zéro
  candidat et s'arrête (`chainesArretees`). Le mécanisme de relais entier — la correction PAR VIE,
  qui est juste — est écrasé par la table par siège. Films déclencheurs : parties avec
  remplissage de bots, siège recyclé (témoin du dépôt : slot=8, RE_LOG 7ter.62).
- **Reproduction** :
  `grep -n 'nameByIndex\[b.FilmIndex\]\|nameBotTracks(doc\|attributeSuccessions(doc\|tracks\[i\].Bot != ""\|DEUX BOTS PEUVENT PARTAGER UN INDEX' apps/go-api/internal/analysis/replay/identity.go apps/go-api/internal/analysis/replay/build.go apps/go-api/internal/analysis/replay/successions.go`
- **Vérification adverse — TIENT.** `replay.Options.Bots` n'a qu'un producteur de production
  (`replaybuild.go:350`, `botIdentities(ksRes)`), qui n'écarte que les bots sans nom et les
  `UnpinnedBots` : il transmet bien les deux entrées de même siège.
  `killsource.loadBotMeta` (`botmeta.go:74-83`) déduplique sur le couple (Slot, BotID), donc les
  deux entrées SURVIVENT jusque-là. Aucun pré-filtre ne rend le défaut inatteignable.

### [P1-7] Une vie ANONYME n'est dessinée nulle part — ESCALADE UTILISATEUR : la prémisse de la décision du 2026-08-20 a changé

**Ce constat n'appelle aucune action proposée : il touche une décision produit prise par
l'utilisateur, et c'est à lui de trancher (skill `adversarial-audit` §8).**

- **Où** : `apps/web/src/lib/replay/rosterLogic.ts:110-111` (`buildPlayers` :
  `const key = track.xuid || …; if (!key) continue`) →
  `apps/web/src/features/match-replay/layers/useSlotIdentity.ts:161-167` (`colorOfSlot` bâti sur
  les seuls joueurs nommés) →
  `apps/web/src/features/match-replay/layers/replayMarkers.ts:243-244`
  (`const color = style.colorOfSlot(...); if (!color) return`).
- **Le fait nouveau** : la porte est documentée et volontaire — `useSlotIdentity.ts:17-22` :
  « UNE VIE SANS PROPRIÉTAIRE (…) NE SE DESSINE PAS (…) Ce sont les caméras et les spectateurs de
  fin de partie ; les replier sur l'encre neutre semait des pions gris qui ne désignaient
  personne (**retour utilisateur du 2026-08-20**) ». Cette décision est **antérieure au schéma 36
  (2026-09-02)**. Avant lui, une piste anonyme était un slot que RIEN n'avait nommé de tout le
  match — une caméra, un spectateur. Depuis, c'est **UNE VIE d'un joueur qui peut être nommé sur
  ses autres vies** : 32 vies anonymes sur 174 (`d9781168`), 21 sur 119 (`1cd3848a`, l'un des
  deux seuls artefacts récents du parc), 75 sur 86 (`51ebbc0f`). La population que la décision
  visait n'est plus celle que le filtre attrape.
- **Conséquence à l'écran, si la décision est maintenue telle quelle** : un joueur dont la vie
  courante est anonyme n'a plus de pion, plus de traînée, plus de cône de visée, plus d'anneau
  d'apparition, et sa mort ne pose aucune croix — alors que le document publie ses positions
  image par image. Sur un film à forte anonymie, la carte n'affiche qu'une poignée de pions
  pendant que le film montre huit joueurs qui bougent.
- **Sous-point, celui-là sans décision produit derrière** : `fireMark.ts:100` emploie
  `colorOfSlot` comme simple TEST d'identité — la marque est peinte avec `style.ink`
  (`fireMark.ts:105-106`), pas avec la couleur retournée. Le « ! » du tireur est donc supprimé
  pour une raison qui ne touche même pas son encre. Idem `thrusterDashFx.ts:240-241`.
- **Reproduction** :
  `grep -rn 'if (!color) return\|const color = style.colorOfSlot\|?? ink.neutral\|?? style.fallback' apps/web/src/features/match-replay/layers/`
  — la même valeur `null` veut dire « encre neutre » dans `equipmentPlacementsLayer.ts:352/:482`
  et `replayDraw.ts:211`, et « ne rien dessiner » dans `replayMarkers.ts:244`.
- **Vérification adverse — TIENT comme fait ; la décision reste au user.**
  `ReplayCanvas.tsx:433` passe bien `doc.tracks` ENTIER (anonymes compris) à `drawTracksLayer` :
  la porte est dans le calque, pas en amont. Aucun test ne couvre le rendu d'une piste anonyme.
  L'encre de repli documentée en `useSlotIdentity.ts:99-103` n'est jamais atteinte,
  `buildPlayers` refusant toute clé vide.

### [P2-1] `birthOfLives` rend la naissance du SLOT, servie comme naissance de la VIE

- **Où** : `apps/go-api/internal/analysis/replay/document_equipment_changes.go:169-183`. Le
  contrat écrit juste au-dessus (`:162-163`) dit « le premier échantillon de position de **chaque
  vie** » ; le code indexe `first[p.Slot]` et garde le MINIMUM.
- **Conséquence** : le témoin est le seul juge de « réapparition équipée » contre « ramassage »
  (`filmdec/equipment_changes.go:422`, `bornAt(ch.Slot)`, fenêtre 1 s). Sur un slot recyclé dont
  la PREMIÈRE vie n'a émis aucun i48, l'annonce de réapparition de la vie suivante est comparée à
  la naissance de la vie PRÉCÉDENTE : l'écart dépasse la fenêtre et l'émission sort
  `EquipmentTaken`. Donnée fausse servie : `equipmentChanges[].kind == "taken"`, un ramassage qui
  n'a pas eu lieu — auquel le rejeu joue le son `objective_pad_pickup`
  (`apps/web/src/features/match-replay/sound/equipmentChangeSound.ts:37-43`). Le même témoin
  borne la fenêtre de récupération de tête de vie (`filmdec/equipment_recovery.go:86-94`), qui
  s'étend alors sur toute la vie précédente.
- **Reproduction** :
  `sed -n '162,183p' apps/go-api/internal/analysis/replay/document_equipment_changes.go`
- Appelant unique : `build_from_film.go:224`, avec le nuage NON décimé (toutes les vies).

### [P2-2] `coverage.equipmentChanges.lives` compte des SLOTS et se publie comme des VIES

- **Où** : `apps/go-api/internal/analysis/replay/document_equipment_changes.go:89-90` (contrat :
  « Lives est le nombre de vies ayant émis au moins une fois ») et `:117` (`Lives: st.Lives`).
  `st.Lives` est incrémenté une fois par entrée de `mergeEquipEmissions`, une map keyée par SLOT
  (`filmdec/equipment_changes.go:249` et `:208`).
- **Règle enfreinte** : forme (e).
- **Conséquence** : sur un film à slots recyclés, le dénominateur du seul calque du rejeu « qui
  sache s'auto-mesurer » SOUS-COMPTE (un slot à trois vies vaut 1), et les témoins qui se lisent
  contre lui (`counterJumps`, `missedEstimate`, `livesFirstOffSpec`) surestiment la perte : le
  redémarrage du compteur R(3) au début de chaque vie suivante est lu comme un SAUT.
- **Reproduction** :
  `grep -n 'Lives: st.Lives' apps/go-api/internal/analysis/replay/document_equipment_changes.go; grep -n 'merged := map\[uint32\]\|st.Lives++' apps/go-api/internal/analysis/filmdec/equipment_changes.go`

### [P2-3] La borne de manche est un MIN sur une population non filtrée

- **Où** : `apps/go-api/internal/analysis/objectiveevents/slotidentity_rounds.go:142-158`
  (`roundStartsOf`), et précisément la boucle `:146-154` — `min[r.Round] = r.TimeMS`, sans filtre
  de slot ni garde de domaine. Consommé par `:299` (`roundOfTime`) puis `:175-185` (`At`).
- **Conséquence** : le dépôt mesure lui-même la population de bruit — `statborg.go:60-63`, « sur
  9 films sans manches multiples, les en-têtes non nuls représentent 22 émissions sur 869
  (2,5 %), toujours isolées ». UN seul de ces enregistrements portant le numéro d'une manche
  RÉELLE, à un instant antérieur à son vrai début, ramène sa borne à cet instant ; `roundOfTime`
  rendant « la manche dont le début est le plus grand qui ne dépasse pas t », toute la plage des
  manches précédentes bascule sur une table d'identité qui nomme d'AUTRES joueurs sur les mêmes
  slots. Trois consommateurs de `At` sont touchés : actions d'objectif (`slotidentity.go:161`),
  ouvertures et porteurs tués du drapeau (`flag_carries.go:212` et `:291`), couronne VIP
  (`vip_crown.go:108`). `AtRound` (crâne, score par joueur) est immunisé.
- **Honnêteté** : la fréquence de déclenchement sur film multi-manche n'est **pas mesurée**
  (aucune cuisson). Ce qui est établi : la garde est absente, et la population de bruit est
  mesurée dans le dépôt. D'où P2 et non P1.
- **Reproduction** :
  `grep -n -A14 'func roundStartsOf' apps/go-api/internal/analysis/objectiveevents/slotidentity_rounds.go`

### [P2-4] `indexBySlot` : en multi-manche, les gestes d'équipement d'un slot recyclé sont crédités au DERNIER occupant

- **Où** : `apps/web/src/lib/replay/rosterLogic.ts:171-181` (`bySlot.set(life.slot, value)` en
  boucle sur TOUTES les vies), consommé par
  `apps/web/src/features/match-replay/model/equipmentUsageLogic.ts:258` puis `:284` (grappin),
  `:289` (épisodes camo/surbouclier), `:295` (poses et objets lâchés).
- **Conséquence** : dans « Utilisation de l'équipement », sur un film MULTI-MANCHE ou avec
  REMPLAÇANT, la ligne du joueur de la manche 1 affiche 0 traction / 0 mur / 0 épisode, et celle
  de qui hérite du slot affiche la somme des deux. Le total d'équipe reste juste, ce qui rend
  l'erreur invisible au contrôle : seule la répartition par joueur est fausse.
- **Note** : la dette est DÉCLARÉE dans le code (`rosterLogic.ts:165` : « dette connue, assumée
  ici parce que le comptage d'usage est au niveau du match, pas de l'image »). Elle est remontée
  parce que le niveau d'agrégation ne rend pas juste l'attribution d'un geste de A à la LIGNE de
  B, et que le remède est écrit deux fonctions plus bas (`buildSlotOwnership` / `ownerAtFrame`,
  `:184`) ; les trois canaux portent l'instant nécessaire (`grappleLines[].t0`,
  `equipmentEpisodes[].t0`, `equipmentPlacements[].t0`).
- **Reproduction** :
  `grep -n 'indexBySlot\|ownerOfSlot\|tallyOfSlot' apps/web/src/lib/replay/rosterLogic.ts apps/web/src/features/match-replay/model/equipmentUsageLogic.ts`

### [P2-5] Deux jointures roster↔joueur perdent les BOTS : elles comparent des clés de deux espaces différents

- **Où** : `apps/web/src/features/match-replay/model/equipmentUsageLogic.ts:280`
  (`doc.roster.map((entry) => [entry.filmIndex, players.find((p) => p.xuid === entry.xuid)])`) et
  `apps/web/src/features/match-replay/ui/ReplayTeams.tsx:245`
  (`doc.roster.find((r) => r.xuid === player.xuid)?.filmIndex ?? null`).
- **Règle enfreinte** : formes (a) et (d) — un bot est une vie IDENTIFIÉE **sans xuid**
  (`Track.Bot`, `document.go:1353-1359`). La clé canonique n'est pas `entry.xuid` mais
  `entry.xuid || botKey(entry.name)` (`rosterLogic.ts:100` et `:110`) : un bot a `xuid: ''` au
  roster et `ReplayPlayer.xuid === 'bot:<nom>'`. Aucune des deux comparaisons ne peut jamais être
  vraie pour un bot. La troisième jointure du dépôt (`seatLogic.ts:180`) fait la dérivation
  correcte.
- **Conséquence, sur tout film AVEC BOTS** : (1) la colonne Grenades du bot est vide alors qu'il
  en a lancé, et ses lancers rejoignent le pied de tableau « N gestes mesurés sans propriétaire
  (vie sans joueur, ou poseur non mesuré) » — phrase fausse, le film le nomme ; si SEULS des bots
  ont lancé une grenade d'un rang, `columnsOf` (`equipmentUsageLogic.ts:355`) supprime la
  colonne. (2) Sur la fiche, `filmIndex` vaut `null`, donc le badge de lancer de grenade ne
  s'allume JAMAIS pour un bot (`ReplayWeaponsRow.tsx:88`).
- **Reproduction** :
  `grep -rn 'const key = entry.xuid\|roster.map((entry)\|roster.find((r)' apps/web/src/lib/replay/rosterLogic.ts apps/web/src/features/match-replay/model/equipmentUsageLogic.ts apps/web/src/features/match-replay/ui/ReplayTeams.tsx`
- Aucun test de `equipmentUsageLogic.test.ts` ni de `ReplayTeams.test.tsx` ne monte de bot.

---

## Constats écartés

| Constat | Motif d'écart |
|---|---|
| `keepNeutralDeathsOfPublishedTracks` (`neutral_deaths.go:26-34`) — même index par XUID nommé, sans repli sur le pont | **Réfuté comme perte UI** : le client filtre symétriquement. Ses lignes de mort neutre viennent de `lifeEndsOf` (`killFeedLogic.ts:373`, `if (!tr.xuid) continue`) et `attachDeathKinds` (`:346`) joint par xuid : une entrée conservée pour un joueur sans piste nommée ne rencontrerait aucune ligne à décorer. Reste le second des deux seuls filtres keyés par XUID — à traiter AVEC [P1-1] pour la cohérence du helper, pas pour lui-même. |
| `IdentifyNamedEvents` (variante plate, `slotidentity.go:136-147`), `SlotIdentityResolved`, `slotIdentityFromDeaths` | Même filtre, mais **aucun appelant de production** (`cmd/zone-attribution`, `cmd/statnames-sweep`, tests). Défaut inatteignable. |
| `carrierPresence.gate` / `bestOverlap` (`skull_carries.go:202-246`) — borne à la vie de recouvrement MAXIMAL, comme `windowFor` avant le schéma 45 | Dette assumée (lecteur déjà rattrapé). **Signalé au superviseur** : un portage de bombe qu'un trou > `lifeGapUS` coupe en deux pistes est tronqué à la moitié la plus longue — même cause que celle qui a produit `spanFor`. |
| `usage_summary.go:300-312` — `default: continue` sur les vies anonymes, `at()` retombe sur `dernier[slot]` | Dette assumée (« usage de session » est l'un des six lecteurs exemptés). **Signalé** pour que le superviseur tranche si l'exemption couvrait ce résidu. |
| `markFlagCarries` (`flag_carries_marker.go:364-376`) — index par `slotXUID[m.Slot]`, premier occupant sur slot recyclé | La règle de collision du pont est exemptée ; les seules sorties touchées sont `MarkerObserved`/`MarkerConfirmed`, témoins de contrôle, pas un calque servi. |
| `document_pickups.go:217` + `pad_pickup_dating.go:171-175` — `Pickup.XUID` nommé par slot, contrat annoncé « par vie » | `SlotXUID` est le pont canonique légitime par cadrage ; rien dans le dépôt ne mesure la part des vies anonymes appartenant à un AUTRE joueur. |
| `equipment_episode_kills.go:116-130` — `K`/`A` à 0 avec `killsRead: true` sur slot non ponté | Latent, non servi : le web route ces épisodes vers `unattributed`, dont seul un total est rendu. Aucune cellule n'affiche ce faux zéro aujourd'hui. |
| `gamertagXUIDIndex` (`replaybuild/kills.go:163-174`) — un tueur qui ne meurt jamais, et tout tueur bot, n'a pas d'entrée | Aucun repli disponible dans le périmètre : `domain.MatchPlayerFact` ne porte aucun gamertag et `replaybuild` n'ouvre pas de base. Ce serait une refonte de contrat, pas un défaut local. |
| `shots.go:86` / `coverage.go:466-486` (`slotFor`), `grenades.go:149`, `vehicle_shots.go:129-131`, `inventory_dead_readings.go:42-44` | Pont muet + catégorie nommée et PUBLIÉE (`noSlot`, `shotsNoRide`, `empty:"unknown"`). Dette assumée. |
| `document_ability_impulses.go:282-291` — `break` après la première vie, fenêtre élargie de ±`lifeGapUS` | Écarté faute de conséquence démontrable : la bande de recouvrement est strictement à l'intérieur du trou de découpe. Affirmer le contraire relèverait du « pourrait ». |
| `document_weapon_changes.go:163-188` (`spawnSetFrom`) | Choix par le TEMPS, consulté à la première émission d'un couple (slot, emplacement) : la configuration fautive n'est établie par aucune mesure. |
| `botIdentities` (`replaybuild.go:466`) — `unpinned` indexé par `BotID` alors que l'anomalie porte sur le SLOT | `firstCopyOnly` (`botmeta.go:139`) déduplique par NOM : deux entrées de même `BotID` exigeraient deux noms distincts partageant un `bid(N.0)`, jamais observé. |
| `withoutContestedXUID` (`slotidentity_deaths.go:248-260`), seconde passe de `SlotIdentityFrom` | Refus de trancher documenté et mesuré : la présence n'est pas NIÉE, elle n'est pas NOMMÉE. |
| `spawnPoints` (`replaybuild/spawnpoints.go:165-180`) — premier match par `PublicName` en itérant une map | Hors axe : catalogue de cartes, pas identité de joueur. |
| `placementTeleport.ts:228` (`lastTeleportAge` non borné à la vie) | L'appelant plafonne l'éclat à 1 400 ms alors que le retour mesuré est à ~8 s : aucune conséquence à l'écran. |
| `abilityChargeLogic.ts:133` (`doc.tracks.find(tr => tr.slot === slot && isAliveAt(tr, frame))`) | La condition EST temporelle et les vies d'un slot sont disjointes : le `find` désigne la vie couvrante sans ambiguïté. |
| `objectiveMark.ts`, `livesPosition.ts`, `killFx.ts`, `shotFx.ts`, `grappleLayer.ts`, `riftStations.ts`, `placementWall.ts`, `threatSensor.ts`, `heatmapLayer.ts` | Multi-vies déjà correct : index en LISTE par slot/xuid puis vie couvrante, ou balayage complet. |
| `grapple_lines.go`, `zoom_state.go`, `successions.go`, `vehicle_relays.go`, découpage des vies de VÉHICULE (`vehicle_tracks.go`), `equipment_placements.go:251-304`, `ground_weapon_rules.go:361-380`, `document_ground_weapon_items.go:201-234`, `pickup_origin.go:178-198`, `zone_states_hill.go:262-270`, `t0_film.go:248-269` | Multi-vies correct, vérifié ligne à ligne. `t0FilmBurst` est le modèle du traitement juste : une piste anonyme y compte pour UN partant, avec la raison écrite. |
| `publishedSlots` / `keepOfPublishedTracks` (`published_tracks.go`) et ses 11 appelants par slot | Filtre par SLOT booléen, explicitement légitime ; aucun appelant ne s'en sert pour borner une mesure à une vie. |
| `score_timeline.go`, `score_team_identity.go`, `vip_crown.go`, `hill_hold_ticks.go`, `killpos.go:162-192`, `document_vehicles.go:333/:359` | Abstention explicite sur pont muet, ou aucun défaut de grain. |
| `nameTracksByLives` (`identity.go:36`), règle de collision d'`ownersFromLives`, `deathInstantMin = 3`, CTF multi-manche muets, baseline lint, couleurs / i18n | Dette assumée par le cadrage. |

## Zones sans constat

- **`internal/service/replayview/` (10 fichiers) et `internal/domain/replaydoc/` (10 fichiers) :
  rien.** Aucune agrégation, aucun index : `sliceOf`/`mapOf`/`ptrOf` sont strictement 1:1 et
  préservent la nullité ; `replaydoc` ne contient que des déclarations de types. La forme (g) de
  l'axe n'a aucun site ici, et `parity_test.go` (listes `champsNonServis` et `typesNonServis`
  toutes deux vides) l'interdit structurellement.
- **Assemblage** (`build.go`, `document.go`, `structure.go`, `catalog.go`, `geometry.go`,
  `origin.go`, `observe.go`, `options.go`, `match_clock.go`, `map_background*.go`,
  `callouts_catalog.go`) : aucun index piste↔joueur ; l'ordre d'assemblage (nommage par vie →
  bots → relais → calques) est correct, à la réserve du [P1-6] sur la place de `nameBotTracks`.
- **Décodage binaire, géométrie, clés de vie d'OBJET `(slot, gen)`** : espace de slots distinct
  de celui des bipèdes, hors axe.

## Découvertes hors axe, notées et NON traitées

- **Code sans appelant de production dans `objectiveevents/`** : `awards.go` en entier
  (280 lignes : `LabelPersonalScore`, `SummarizeLabels`, `Describe`, `decompose`,
  `combinations`), plus `CountsBySlot`, `KnownStats`, `CrossCheckNamedEvents`, `SlotIdentity`,
  `LabelledEvent` — seulement des tests. Anti-pattern n°1, à trancher hors de cet audit.
- `replaybuild/kills.go:148` traite en `slog.Warn` le cas nominal du tueur bot, alors que le cas
  symétrique (victime bot) est en `Info` avec sa justification.

## Suite

1. **Lot Go, cohérent** — les six constats Go P0/P1 demandent tous la même chose : faire
   descendre le pont canonique (`own.SlotXUID`, `own.lives`) là où il manque, sur le patron déjà
   éprouvé de `tracksByXUID`, avec sa contre-épreuve (« une piste anonyme dont le pont ne nomme
   pas le slot reste écartée »). Ordre imposé par le couplage : **[P1-2] d'abord** (sans une
   couverture qui voit, aucune correction ne se mesurera), puis [P1-1] et [P0-1] qui sont deux
   portes EN SÉRIE sur la même donnée, puis [P1-3]. [P1-4], [P1-5] et [P1-6] sont indépendants.
   À cadrer sous `plan-review`, exécuter sous `plan-execution`, relire sous `adversarial-review`.
2. **Lot web, autonome** — [P0-2] n'a aucune dépendance Go : le helper correct (`lifeOfSlotAt`)
   existe et est employé par quatre calques. [P2-4] et [P2-5] s'y rattachent naturellement.
3. **Escalade utilisateur, sans proposition d'action** — [P1-7] : la décision du 2026-08-20
   (« pas de pions gris ») porte sur une population que le schéma 36 a changée. Décision produit.
4. **À trancher par le superviseur** — les deux résidus signalés en écartés : le rognage de
   `carrierPresence.gate` sur les portages de bombe, et le repli `dernier[slot]` d'
   `usage_summary.go` sur un instant couvert par une vie anonyme. Les deux tombent dans
   l'exemption « lecteur déjà rattrapé » ; reste à dire si l'exemption les couvrait vraiment.
5. **Angle mort de mesure, à lever avant de chiffrer un correctif** — aucun artefact du parc
   n'est cuit au schéma 41+, et les deux plus récents (schéma 38) ne portent ni objectifs ni
   zones. Une cuisson de contrôle sur un témoin CTF et un témoin Bastion au HEAD est le seul
   moyen de chiffrer [P0-1] et [P1-1] sur la géométrie actuelle des vies. **Non faite ici : la
   consigne interdit toute cuisson.**
