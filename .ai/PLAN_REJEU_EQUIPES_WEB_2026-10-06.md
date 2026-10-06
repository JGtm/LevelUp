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

- [x] E2.1 Colonnes : `seatLogic` (`ReplaySeat.team`, retrait de `campsParCote`, `cleDeCamp`,
  `ordreDesCamps`, `rangDePlace`, clés `s:<team_side>` et `''`, rang « sans camp en dernier »,
  place `joueur:<xuid>` devenue morte) ; clé de place `siege:<équipe>:<place>` (places finies PAR
  ÉQUIPE : deux équipes ne partagent jamais une tuile) ; `groupSeatsByTeam` par `groupByCamp` ;
  `ReplayTeams` / `ReplayTeamHeader` (libellé par `campLabel` sur la feuille de TOUS les occupants,
  calculé une fois par document ; clé de rendu `camp:<désignateur>` ; l'en-tête ne garde que
  l'encre).
- [x] E2.2 Menu de point de vue (frise) : `viewpointOptions` sur les camps du film
  (`groupByTeam`), libellé `campLabelOf` = `campLabel` (même cascade que les colonnes) ; retrait
  du groupe `side == null`, de `labelDuGroupe` et de `viewpointNoTeam`.
- [x] E2.3 Tables de l'onglet Arsenal : `equipmentUsageLogic` / `equipmentUsageChart` /
  `MatchEquipmentUsageSection`, `padControlLogic` / `padControlChart` / `padControlColumns` /
  `MatchPadControlSection` sur les camps du film (lignes et camps `extends ReplayCamp`) ; les
  gestes d'un joueur sans équipe rejoignent `unattributed` (réserve du titre), ses prises de socle
  `unjoined` — la somme ne ment pas ; retrait des clés `'sans-equipe'` (React et `data-testid` :
  `camp:<désignateur>`) et du rang 2 « camp inconnu » ; tri des camps du contrôle des socles :
  total puis désignateur.
- [x] E2.4 i18n : retrait de `teamUnknown` et `viewpointNoTeam` (FR, EN, contrat) ; les autres
  appels de la cascade dans le rejeu (`useTeamCascades`, `ReplayVictoryOverlay`,
  `exportOverlayPanels`) passent à `campLabel` (donc `resolveKnownTeamLabel`) : plus aucun appel
  de `resolveTeamLabel` dans `features/match-replay`.
- [x] E2.5 Tests : réécriture des tests qui figeaient le repli de feuille ou la section sans
  équipe (traduction mesurée, côté contradictoire, repli `s:`, voie `joueur:<xuid>`, « Sans
  équipe » du menu, de la frise et de l'en-tête de colonne) ; équipes du film posées dans les
  rosters des tests de rendu (fixations 4v4 / 6v6 inchangées à l'octet) ; nouveaux tests :
  témoin 43716616 (logique et DOM, FR + EN : deux colonnes à chaque image, jamais « Sans équipe »
  ni « No team »), places d'équipes différentes, plancher « Équipe N », gestes et prises d'un
  joueur sans équipe hors camp, contrat Go/web « toute entrée porte son équipe ». Tests de
  l'équipe de `rosterLogic` déplacés dans `rosterLogic.equipes.test.ts` (seuil de 500 lignes).
- Gate E2 : vitest des dossiers touchés, tsc, eslint. PASSÉ le 2026-10-06 : `tsc -b` 0 erreur,
  eslint des 45 fichiers touchés 0 problème, vitest `features/match-replay` + `lib/replay` +
  `lib/halo` + `features/match-view` : 296 fichiers, 4 061 tests verts (2 garde-rails à balayage
  de `src/` ont dépassé leur délai de 5 s sous charge machine ; rejoués seuls deux fois : 9/9).

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

- D1 — ENCRE DES BOTS SUR LA CARTE : `useSlotIdentity.isAlly(p.xuid)` interroge `xuidMeta` (clés
  de la feuille, `bid(N.0)` pour un bot) avec la clé du film `bot:<nom>` : un bot n'y est jamais
  trouvé, donc `isAlly` rend `false` et son pion prend l'encre ADVERSE quel que soit son camp —
  y compris un bot allié. Famille « encre allié / adverse » (point 6 du brief, hors périmètre).
- D2 — PISTE COÉQUIPIERS ET CAPTEUR DE MENACES : la population de la piste « Coéquipiers » (frise)
  vient de `identity` (`xuidMeta`, feuille) et l'opposition du capteur de menaces de
  `sideResolver` (`team_side` de la feuille) : un coéquipier que la feuille ignore n'y figure pas.
  Même famille que D1, laissée sur la feuille (allowlist du garde-rail E4).
- D3 — FFA : le désignateur `-1` (« aucune équipe », mode sans camps) regroupe tous les joueurs en
  UNE colonne (comportement inchangé depuis le lot 1.9.14), alors que la décision D3 du plan des
  fiches compactes (2026-09-06) disait « chaque joueur est sa propre équipe, N colonnes d'un
  siège » ; et si aucun membre n'a de ligne de feuille, son nom plancher serait « Équipe -1 ».
  Aucun FFA au parc (126 artefacts à deux camps) : question produit à porter à l'utilisateur.
- D4 — FILM SANS ROSTER (sans identification, artefact antérieur au schéma 57) : il n'écrit
  aucune équipe, la colonne affiche donc le constat `rosterEmpty` (« Aucune vie du film n'a pu
  être rattachée à un joueur »), dont le libellé ne dit pas la vraie cause. 0 artefact du parc
  concerné.
- D5 — ADR 0034 D-9 écrit encore « the web colours players by `team_side` from the match sheet »
  (faux depuis le lot 1.9.14, et pour le regroupement depuis ce lot) : document hors périmètre.
- D6 — Deux garde-rails à balayage complet de `src/` (`xuidMeta.guard`, `teamLabel.guard`)
  frôlent le délai de 5 s de vitest sous charge machine (7,4 s et 8,1 s observés en parallèle,
  5,4 s seul, puis verts).

## Journal

- 2026-10-06 — plan rédigé après lecture du brief, du code et des tests ; base vitest verte
  (53 fichiers / 918 tests sur les dossiers touchés). Mesure préalable du parc (lecture seule,
  126 artefacts) : 0 place dont les occupants ont des équipes différentes, 18 entrées sans
  équipe (toutes avec une présence publiée, 2 sur une place partagée), désignateurs {0: 595,
  1: 588}. Les 8 fixtures Go du worktree (schéma 79) : 0 entrée sans équipe.
- 2026-10-06 — E1 close (gate vert). Choix : le côté de feuille d'un camp est celui de la
  MAJORITÉ de ses membres (un joueur que la feuille contredit ne renomme pas son camp).
- 2026-10-06 — E2 close (gate vert). Choix : la clé d'une place porte son équipe (places finies
  par équipe) ; les gestes / prises d'un joueur sans équipe vont aux compteurs « hors camp »
  existants (`unattributed`, `unjoined`) plutôt que de disparaître. Découvertes D1 à D6 notées.
