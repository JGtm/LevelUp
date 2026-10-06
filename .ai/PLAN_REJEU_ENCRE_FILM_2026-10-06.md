# Plan — Rejeu : l'encre allié / adverse vient de l'équipe du FILM (WEB, phase E6) — 2026-10-06

Branche `feat/rejeu-equipes-web` (worktree `LevelUp-wt-rejeu-equipes-web`), repartie de
`f8a14b3b9` (fusion de `origin/feat/v75` fed1efed2 par le superviseur ; `feat/v75` avancé dessus).
Commande : message du superviseur (session levelup-dc), phase accordée par l'utilisateur.
Contrat d'exécution : skill `plan-execution`. Lot Go parallèle (source) hors de ce plan.
Fusion dans `feat/v75` : le superviseur, pas ce lot.

## Décision (ferme, venue de la commande)

allié(p) = (`ReplayPlayer.team` de p) === (équipe du film du joueur de référence). Le joueur de
référence garde la sémantique d'aujourd'hui (rien n'est re-décidé ici) :

| Surface | Référence aujourd'hui (constatée sur pièces) |
|---|---|
| Page Rejeu (carte, colonnes, fil, frise, bandeau, écran de fin, export, calques d'objectif, sons d'objectif) | `model.viewpoint` = point de vue choisi, sinon le joueur de la page (`replayModel.ts:134`) |
| Fin de partie sonore | le joueur de la page (`endMatchSound.ts`, décision 3 du plan « frise, point de vue ») |
| Tables de l'onglet Arsenal (page Match) | le joueur de la page (ligne `is_me`) |

Calcul en UN seul endroit (`lib/replay/filmAllegiance.ts`), consommé par toutes les surfaces
d'encre et d'allégeance. `xuidMeta` ne sert plus que l'identité (noms, filtre des kills de la
base) ; les marques « moi / ami » restent `buildPlayerMarks`.

Règles de bord (choisies ici, documentées dans le module et testées) :

- Joueur dont le film tait l'équipe : AUCUNE encre de camp (`null`), sur toutes les surfaces —
  encre neutre de la surface (pion `floorStyle.edge`, nom du fil `teamTokenCssVar(null)`, en-tête
  neutre, glyphe de présence à l'encre courante), son à deux variantes muet. Aucun repli sur la
  feuille. C'est le défaut de source que le lot Go corrige.
- Joueur de référence sans équipe du film (ou absent du film) : personne n'a d'encre de camp —
  lui compris (même règle que n'importe quel joueur sans équipe) ; les surfaces dont le camp EST la
  structure (bandeau de score, écran de fin, sons à deux variantes) se taisent, comme aujourd'hui
  quand la ligne « moi » manque. Jamais d'encre devinée.
- Mode sans camps (désignateur `-1` de la référence) : la référence est alliée d'elle-même, tout
  autre joueur est adverse (comportement d'aujourd'hui en mêlée générale) ; un CAMP `-1` n'a pas
  d'encre (`ofTeam` → `null`).

## Inventaire des surfaces (lu sur pièces, 2026-10-06)

| # | Surface | Fichier(s) | Source d'allégeance avant |
|---|---|---|---|
| S1 | Pions, traînées, véhicules, effets de frontière (objet lâché, mort) | `rosterLogic` (`colorResolver`, `colorResolverOrLast`, `colorByXuidResolver`, `teamColorOfOwner`), `useSlotIdentity`, `ReplayCanvas` | `xuidMeta.ally` par clé film (bot `bot:<nom>` jamais trouvé : D1) |
| S2 | Opposition capteur de menaces / zones de fiche / poses | `rosterLogic.sideResolver`, `threatSensor`, `equipmentZones`, `equipmentPlacementsLayer`, `ReplayTeams` | `board.team_side` |
| S3 | En-têtes de colonnes | `ReplayTeamHeader`, `ReplayTeams` | `xuidMeta.ally` des occupants présents (bots inconnus) |
| S4 | Fil des éliminations (kill, médaille seule, mort neutre, entrée/sortie) | `ReplayKillFeed`, route (`colorOfTeam`) | `KillEvent.ally` + `xuidMeta.ally` (replis devinés `!k.ally`, `true`) |
| S5 | Frise : piste des coéquipiers, absences des coéquipiers, glyphe de présence | `useReplayTimeline`, `replayTimelineTracksLogic`, `ReplayTimelineTracks`, `ReplayPresenceShade` | `identity.ally` |
| S6 | Piste Dominance | `useTeamCascades` | `allyOfTeamId` (feuille + `xuidMeta`) |
| S7 | Bandeau de score (+ garde de colline, pastilles de manche) | `scoreBannerLogic`, `ReplayScoreBanner` | camps `team_side` + `allyOfTeamId` |
| S8 | Écran de fin, panneau d'export, score final, fin de partie sonore | `victoryLogic`, `ReplayVictoryOverlay`, `exportOverlayPanels`, `useReplayExport`, `useReplayCapture`, `endMatchSound`, `replayModel` | camps `team_side`, ligne `is_me` |
| S9 | Zones (encre, jauge) | `useZoneStates` | `matchSides.allyTeamFromScoreboard` |
| S10 | Drapeaux (glyphe, onde de capture, défenseurs de la zone de retour) | `useReplayFlagCarries` | `matchSides` (allié + table xuid -> équipe) |
| S11 | Déflagration d'Assaut | `useReplayBombBlast` | `matchSides` |
| S12 | Sons d'objectif (voix allié / adverse, tics et sécurisation de zone, marques de crâne) | `objectiveSound`, `useReplaySound` | `matchSides` |
| S13 | Tables de l'onglet Arsenal (grille, barres, contrôle des socles) | `MatchEquipmentUsageSection`, `equipmentUsageChart`, `MatchPadControlSection` | `team_side` de la ligne `is_me` |

Hors périmètre (constaté) : `ReplayMarkTrack` (kill = `team-ally`, mort = `team-enemy` : encre de
l'ÉVÉNEMENT du joueur regardé, pas une allégeance) ; les cinq graphes de la page Match
(`allyOfTeamId` / `resolveXuidMeta` à deux arguments : la Vue match garde la feuille).

## Étape E6.1 — Le foyer unique et la carte

- [x] E6.1.1 `lib/replay/filmAllegiance.ts` : `buildFilmAllegiance(players, reference)` →
  `referenceTeam`, `camps` (camps du film à désignateur >= 0, nommés par la feuille),
  `ofTeam`, `ofPlayer`, `ofXuid` (clé du film OU xuid de base ; un bot se relie par
  `board.xuid`), `teamOfXuid`, `membersOf` ; `filmCampOf` (camp d'un joueur pour l'opposition).
  Tests unitaires de toutes les règles de bord.
- [x] E6.1.2 `rosterLogic` : les trois résolveurs d'encre prennent l'allégeance d'un JOUEUR
  (tri-état, `null` → encre neutre) ; `sideResolver` (feuille) remplacé par `campResolver`
  (équipe du film) ; consommateurs `threatSensor`, `equipmentZones`, `equipmentPlacementsLayer`,
  `ReplayTeams`. `useSlotIdentity` reçoit l'allégeance au lieu de `xuidMeta`.
- [x] E6.1.3 `replayModel` construit l'allégeance UNE fois (`model.allegiance`, référence =
  `model.viewpoint`) ; la route et `ReplayCanvas` la relaient (prop requise).
- [x] E6.1.4 Tests : résolveurs (bot allié joint par `board.xuid`, sans équipe neutre, référence
  sans équipe), opposition par le film, `replayModel`.
- Gate E6.1 : vitest des fichiers touchés, `tsc -b`, eslint des fichiers touchés.
  PASSÉ le 2026-10-06 : vitest `src/lib/replay` + `src/features/match-replay` 244 fichiers /
  3 599 tests verts (4 fichiers sautés par conception), `tsc -b` 0 erreur, eslint du projet
  0 erreur. Tests neufs : `filmAllegiance.test.ts` (14), résolveurs d'encre et `campResolver`
  dans `rosterLogic.equipes.test.ts` (5 ; le bot allié D1, le sans-équipe neutre, la référence
  sans équipe), `replayModel.test.ts` (2). Allowlist du garde : l'entrée `rosterLogic.ts`
  (`sideResolver`) sort (0 lecture).

## Étape E6.2 — Colonnes, fil, frise

- [x] E6.2.1 `ReplayTeamHeader` reçoit l'allégeance de son CAMP (`ofTeam`) ; `ReplayTeams` reçoit
  `allegiance` au lieu de `xuidMeta`.
- [x] E6.2.2 `ReplayKillFeed` : encre de chaque nom par `ofXuid` + `teamTokenCssVar` (tokens
  `team-ally` / `team-enemy`, neutre pour `null`) ; retrait des replis devinés, du résolveur
  `colorOf` de la route et de la cascade d'identité `teamColorResolver` devenue sans appelant.
- [x] E6.2.3 Frise : coéquipiers, absences et glyphes de présence par l'allégeance.
- [x] E6.2.4 Tests DOM : en-tête d'un camp de bots allié, fil (bot allié, joueur sans équipe
  neutre), piste des coéquipiers avec un bot allié.
- Gate E6.2 : idem E6.1. PASSÉ le 2026-10-06 : `tsc -b` 0 erreur ; vitest `src/lib/replay` +
  `src/lib/halo` + `src/features/match-replay` + `src/features/match-view` 296 fichiers /
  4 103 tests verts ; eslint `src` 0 erreur (26 avertissements préexistants, compte inchangé).
  Tests neufs : `ReplayTeams.encre.test.tsx` (3 : camp de bots adverse inké, vu d'un bot, référence
  sans équipe), `ReplayKillFeed.encre.test.tsx` (5 : bot allié et son arme, sans-équipe neutre,
  victime hors film neutre, médaille / mort / présence, référence sans équipe),
  `useReplayTimeline.test.ts` (bot allié coéquipier). Fixations DOM 4v4 / 6v6 inchangées à
  l'octet avec l'allégeance du film. Mutations : en-tête forcé neutre → 4 rouges (dont les deux
  fixations) ; repli « adverse » sur allégeance inconnue au fil → 4 rouges. `ReplayTeams` exige
  `allegiance` (requise, comme le point de vue) : 49 montages de test reçoivent `NO_ALLEGIANCE`
  (en-têtes neutres, comme sans `xuidMeta` avant). Allowlist : `ReplayKillFeed.tsx` sort.

## Étape E6.3 — Les camps vus de la référence (bandeau, dominance, fin, objectifs, sons)

- [x] E6.3.1 Bandeau de score : camps du film, côté allié par `ofTeam`.
- [x] E6.3.2 Dominance : `allyOf` = `ofTeam`.
- [x] E6.3.3 Fin de match : `victoryLogic` sur les camps du film (équipe du joueur de la page,
  équipe du sujet) ; écran, export, score final, fin de partie sonore.
- [x] E6.3.4 Zones, drapeaux (encre et défenseurs), déflagration, sons d'objectif ; suppression
  de `matchSides` (module, tests, garde-rail) ; `allyOfTeamId` (Vue match seulement) quitte
  `lib/replay` pour `features/match-view` ; `teamIdOfSide` sans appelant supprimé.
- [x] E6.3.5 Tests de chaque surface migrée.
- Gate E6.3 : idem E6.1. PASSÉ le 2026-10-06 : `tsc -b` 0 erreur ; vitest `src/lib/replay` +
  `src/lib/halo` + `src/features/match-replay` + `src/features/match-view` 295 fichiers /
  4 081 tests verts ; eslint `src` 0 erreur (26 avertissements préexistants). Tests réécrits sur
  l'allégeance du film : `scoreBannerLogic` (32, dont « un camp de bots sans ligne de feuille est
  un camp », « le film ne situe pas le joueur regardé »), `victoryLogic` (40, dont « le film
  décide, pas la feuille »), `endMatchSound`, `objectiveSound`, `useZoneStates`,
  `useReplayFlagCarries`, `ReplayScoreBanner`, `ReplayVictoryOverlay`, `exportOverlayPanels`,
  `useReplayCapture`, `useReplayExport`, contextes sonores ; neufs : `useReplayBombBlast.test.tsx`
  (2), `allyOfTeamId` dans `xuidMeta.test.ts` (2). `matchSides` (module, test, garde-rail)
  supprimé ; `allyOfTeamId` déplacé dans `features/match-view/xuidMeta.ts` (seuls lecteurs : deux
  graphes de la page Match) ; `teamIdOfSide` et son test supprimés (plus d'appelant). Allowlist :
  `matchSides.ts`, `scoreBannerLogic.ts`, `victoryLogic.ts`, `scoreTimeline.ts` sortent ; garde
  `is_me` : exemptions `matchSides.ts` et `victoryLogic.ts` retirées (elles étaient devenues
  périmées, le garde l'a dit).

## Étape E6.4 — Tables de l'onglet Arsenal

- [x] E6.4.1 `MatchEquipmentUsageSection`, `MatchPadControlSection`, `equipmentUsageChart` :
  allégeance du film, référence = joueur de la page (`meXUIDOf`).
- [x] E6.4.2 Tests (la feuille qui contredit le film ne change ni l'ordre ni l'encre ; joueur
  de la page sans équipe du film : encres neutres, ordre du film). Le « camp d'un bot allié » ne
  se pose pas ici : l'encre des tables est celle d'un CAMP, déjà nommé par ses membres de feuille
  — l'écart D1 était celui des joueurs (carte, fil).
- Gate E6.4 : idem E6.1. PASSÉ le 2026-10-06 : `tsc -b` 0 erreur ; vitest des quatre dossiers
  295 fichiers / 4 084 tests verts ; eslint `src` 0 erreur. Tests neufs : 2 dans
  `MatchEquipmentUsageSection.test.tsx`, 1 dans `MatchPadControlSection.test.tsx` ; mutation
  (allégeance des socles forcée nulle) → 2 rouges. Allowlist : les deux sections sortent ; garde
  `is_me` : leurs exemptions sont retirées (`meXUIDOf`). Commentaires de
  `equipmentUsageLogic`, `padControlLogic`, `padControlChart`, `padControlColumns` remis au vrai
  (le côté de feuille ne fait plus que nommer).

## Étape E6.5 — Garde-rails et documentation

- [ ] E6.5.1 `replayCamps.guard.test.ts` : allowlist réduite aux lectures de `team_side` qui
  NOMMENT, justifiée ; règle nouvelle : aucune lecture d'allégeance dans la table d'identité
  (`.get(…)?.ally`) sur le périmètre du rejeu.
- [ ] E6.5.2 `noIsMeOutsideViewpoint.guard.test.ts` : exemptions périmées retirées.
- [ ] E6.5.3 `xuidMeta` : régime à trois arguments (allié relatif au point de vue, rejeu seul)
  si plus aucune lecture ne le consomme — suppression avec ses tests (0 code mort).
- [ ] E6.5.4 En-têtes et commentaires des modules touchés (la feuille ne fait que nommer).
- Gate E6.5 : garde-rails verts, mutation de contrôle (une lecture réintroduite fait rougir).

## Étape E6.6 — Mesure au parc (126 artefacts, lecture seule)

- [ ] E6.6.1 Vies de bots qui changent d'encre (adverse → alliée, alliée → adverse, → neutre).
- [ ] E6.6.2 En-têtes qui ne sont plus neutres.
- [ ] E6.6.3 Témoins pour l'utilisateur : bot ALLIÉ à équipe connue (c7f94693 Donos si sa vie a
  une équipe, sinon un autre), avec les images.

## Étape E6.7 — Livraison

- [ ] E6.7.1 Suite vitest complète `--pool=forks` hors sandbox, `tsc -b` à froid, eslint, knip.
- [ ] E6.7.2 `delivery-checklist`.
- [ ] E6.7.3 `adversarial-review` (contexte frais, ≤ 2 tours).
- [ ] E6.7.4 Commits `fix(rejeu):`, push de `feat/rejeu-equipes-web`, CI verte.
- [ ] E6.7.5 Entrée `.ai/thought_log.md`.

## Journal

- 2026-10-06 — Repris de `f8a14b3b9` (`git pull --ff-only` : à jour). Inventaire ci-dessus lu sur
  pièces (grep `team_side|ally|xuidMeta|identity|is_me` sur `features/match-replay` +
  `lib/replay` + route, puis lecture de chaque lecteur).

## Découvertes

- E6-D1 (2026-10-06) : `rosterLogic.test.ts` est à 500 lignes de code ; les tests neufs des
  résolveurs vont dans `rosterLogic.equipes.test.ts` (même règle qu'en E1).
- E6-D2 (2026-10-06, conséquence, pas un fix opportuniste) : le résolveur `colorOf` du fil et son
  repli `teamColorResolver(scoreboard)` (cascade d'IDENTITÉ : `team_color`, puis couleur
  officielle par `team_id`) disparaissent avec la migration du fil ; la page passait toujours les
  jetons, le repli ne vivait plus que par un test. `teamColorResolver` / `TeamColorResolver`
  (`match-view/teamColor.ts`), `resolveTeamColorFromID` et `TEAM_COLORS_HALO_INFINITE`
  (`lib/halo/teamNames.ts`) n'ont plus aucun appelant : supprimés (CLAUDE.md n° 7) ; en-têtes de
  `teamColor.ts` et `teamSeriesColor.ts` et doc de `team_color` (`lib/api/types.ts`) remis au
  vrai. `occupantsPresents` (`ReplayTeams`) et `NO_IDENTITY` (`replayCanvasConfig`) supprimés
  avec leur seul usage.
- E6-D3 (2026-10-06, NON traitée — hors périmètre de l'encre) : la piste Dominance attribue
  chaque frag au camp que l'API publie pour son auteur (`KillEvent.teamID`, `actor_team_id`),
  pas à l'équipe du film du tueur ; son ENCRE, elle, vient du film (`ofTeam`). Les deux
  coïncident tant que le contrôle d'accord Go tient (désignateur du film = identifiant d'équipe
  de l'API). À statuer par le superviseur : l'appartenance d'un frag à un camp relève-t-elle de
  la règle « équipe = film » ?
