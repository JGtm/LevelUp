# Plan — Rejeu : aucune section sans équipe (WEB) — 2026-10-06

Branche `feat/rejeu-equipes-web` (worktree `LevelUp-wt-rejeu-equipes-web`, partie de
`origin/feat/v75` 1518e6f10). Brief du superviseur (session levelup-dc) :
`BRIEF_REJEU_EQUIPES_WEB.md` (scratchpad de la session). Contrat d'exécution : skill
`plan-execution`. Lot Go parallèle (source) : `feat/rejeu-equipes-source`, hors de ce plan.

## Décision (ferme, venue du brief)

Sur la page Rejeu, l'équipe d'un joueur est le DÉSIGNATEUR DU FILM (`roster[].team`, schéma 57+),
et rien d'autre (ADR 0034 D-9). Une section « Sans équipe » n'existe pas ; un joueur dont le film
tait l'équipe ne rend aucune tuile et n'entre dans aucune section (défaut de source, compté par
`coverage.seats.sansEquipe`, corrigé par le lot Go). La feuille de match ne fait que NOMMER (et
l'encre allié / adverse, hors périmètre). Les tuiles d'attente ont la boîte d'une fiche.

## Étape E1 — L'équipe d'un joueur : un seul foyer

- [x] E1.1 `ReplayPlayer.team` posé par `buildPlayers` depuis l'entrée de roster, jointe par
  `rosterEntryKey` (la copie inline de la dérivation de clé dans `buildPlayers` disparaît).
- [x] E1.2 Module `lib/replay/replayCamps.ts` : type `ReplayCamp` (désignateur + côté de feuille
  qui nomme), `groupByCamp` (regroupe par désignateur, écarte les sans-équipe, ordre du
  désignateur), `campSideOf` (côté de feuille majoritaire des membres qui ont une ligne),
  `campLabel` (cascade de nommage, plancher numéroté depuis le désignateur). `groupByTeam`
  (rosterLogic) devient son adaptateur « joueurs ».
- [x] E1.3 `lib/halo/teamLabel.ts` : `resolveKnownTeamLabel` (même cascade, plancher « Équipe N »
  au lieu d'« Équipe inconnue ») ; `resolveTeamLabel` inchangé pour la Vue match (étapes 1 à 4
  factorisées dans `nameFromSheet`, sortie identique — tests existants verts).
- Gate E1 : tests unitaires de `replayCamps`, de `teamLabel`, de `buildPlayers` ; tsc.
  PASSÉ le 2026-10-06 : 4 fichiers / 83 tests verts, `tsc -b` sans erreur.

## Étape E2 — Tous les regroupements consomment l'équipe du film

- [ ] E2.1 Colonnes : `seatLogic` (`ReplaySeat.team`, retrait de `campsParCote`, `cleDeCamp`,
  clés `s:<team_side>` et `''`, rang « sans camp en dernier », place `joueur:<xuid>` devenue
  morte) ; `groupSeatsByTeam` par `groupByCamp` ; `ReplayTeams` / `ReplayTeamHeader` (libellé par
  `campLabel`, calculé une fois par document, clé de rendu `camp:<désignateur>`).
- [ ] E2.2 Menu de point de vue (frise) : `viewpointOptions` sur les camps du film, libellé par
  `campLabel` ; retrait du groupe `side == null` et de `viewpointNoTeam`.
- [ ] E2.3 Tables de l'onglet Arsenal : `equipmentUsageLogic` / `equipmentUsageChart` /
  `MatchEquipmentUsageSection`, `padControlLogic` / `padControlChart` / `MatchPadControlSection`
  sur les camps du film ; les gestes d'un joueur sans équipe rejoignent la réserve
  (`unattributed`), ses prises de socle les non rattachées (`unjoined`) — la somme ne ment pas ;
  retrait des clés `'sans-equipe'` et du rang 2 « camp inconnu ».
- [ ] E2.4 i18n : retrait de `teamUnknown` et `viewpointNoTeam` (FR, EN, contrat) ; les autres
  appels de la cascade dans le rejeu (`useTeamCascades`, `ReplayVictoryOverlay`,
  `exportOverlayPanels`) passent à `resolveKnownTeamLabel`.
- [ ] E2.5 Tests : réécriture des tests qui figeaient le repli de feuille ou la section sans
  équipe (code mort supprimé avec ses tests) ; nouveaux tests : entrée sans équipe = aucune tuile,
  aucune troisième colonne ; libellé plancher ; somme des tables.
- Gate E2 : vitest des dossiers touchés, tsc, eslint.

## Étape E3 — Tuiles d'attente au gabarit

- [ ] E3.1 Squelette de tuile partagé (boîte `TILE_LAYOUT[...].tile`, ligne de nom, corps fixe
  `BODY_CLASS[...]`) consommé par `ReplayPlayerCard` ET par `ReplaySeatVacant` /
  `ReplaySeatNotSpawned` ; chrome des tuiles d'attente en jetons (aucun littéral de couleur).
- [ ] E3.2 Test DOM : pour chaque gabarit (normal, compact BTB), les trois sortes de tuile ont la
  même classe de boîte et la même classe de corps fixe.
- [ ] E3.3 Fixations `replayTeams.4v4.html` / `6v6.html` inchangées (elles ne contiennent pas de
  tuile d'attente).
- Gate E3 : vitest `ui/`, garde `cardGabarit.guard.test.ts`.

## Étape E4 — Garde-rail

- [ ] E4.1 Test grep sous `features/match-replay` + `lib/replay` : (a) `.team_side` lu hors du
  helper de libellé = échec, sauf allowlist explicite, justifiée, datée, comptée ; (b) retour de
  « Sans équipe » / « No team » / `teamUnknown` / `viewpointNoTeam` / `sans-equipe` = échec.
  Contre-épreuves.

## Étape E5 — Vérification et livraison

- [ ] E5.1 Mesure sur pièces du parc (126 artefacts, lecture seule, script jetable non commité) :
  camps par artefact, entrées sans équipe rendues, tuiles d'attente.
- [ ] E5.2 Gates : `npm run typecheck`, `npm run lint`, knip, linters ratchet du pre-push, vitest
  COMPLET (`--pool=forks`), Playwright si l'environnement le permet.
- [ ] E5.3 `delivery-checklist`, `adversarial-review` sur le diff, corrections.
- [ ] E5.4 Commits (`fix(rejeu):`), push, CI suivie et verte, entrée `.ai/thought_log.md`.

## Découvertes (notées, non traitées — règle 7 du contrat)

(à remplir pendant l'exécution)

## Journal

- 2026-10-06 — plan rédigé après lecture du brief, du code et des tests ; base vitest verte
  (53 fichiers / 918 tests sur les dossiers touchés). Mesure préalable du parc (lecture seule,
  126 artefacts) : 0 place dont les occupants ont des équipes différentes, 18 entrées sans
  équipe (toutes avec une présence publiée, 2 sur une place partagée), désignateurs {0: 595,
  1: 588}. Les 8 fixtures Go du worktree (schéma 79) : 0 entrée sans équipe.
- 2026-10-06 — E1 close (gate vert). Choix : le côté de feuille d'un camp est celui de la
  MAJORITÉ de ses membres (un joueur que la feuille contredit ne renomme pas son camp).
