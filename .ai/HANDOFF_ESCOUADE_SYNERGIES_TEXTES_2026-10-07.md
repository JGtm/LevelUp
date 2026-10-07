# HANDOFF — Escouade › Synergies : retrait de la riposte et de la hauteur, textes sans personne

> Rédigé le 2026-10-07 par la session superviseur du chantier « pages solo aux formes de
> l'Emprise » (Séries temporelles, Sessions, Vue match — livrées). Ce document ouvre le lot
> qui reste : il est écrit pour la session qui l'exécutera. Doctrine RE-VÉRIFIER : les
> lignes citées ont été lues le 2026-10-07 sur `feat/v75` à `94ac8fd68` ; rouvrir chaque
> fichier avant de coder.
>
> **Statut (2026-10-07) : EXÉCUTÉ par l'exécutant** sur `feat/escouade-synergies-textes`
> (worktree `LevelUp-wt-escouade`, partie de `879f31bbf`), plan
> `.ai/PLAN_ESCOUADE_SYNERGIES_TEXTES_2026-10-07.md` : E1 `4cf68289b` (web), E2 `c23114e18`
> (Go et contrat), E3 `82020eacf` (textes et garde), E4 (clôture). Écart au §3 : `coordination.Mesurer`
> RESTE (l'appui et l'isolement le lisent, plan D1). Découvertes : plan §8 (dont §8.5,
> `TacticalKillEvents.Events` sans lecteur, à trancher). Reste au superviseur : push, revue
> adversariale (§5.4), fusion (§5.5), mémoire du projet (§5.6).

## 1. Ce que l'utilisateur a décidé (messages datés)

- **2026-10-05** — « Sur la page Escouade je ne voulais plus de "Rôles de hauteur", "Riposte",
  "Temps de riposte" et "Morts ripostées". » « Appui » et « Rôles de portée » sont GARDÉS.
  La notion de RIPOSTE et celle de HAUTEUR sortent de toutes les pages ; l'APPUI reste.
- **2026-10-06** — sémantique des textes : titres courts, factuels, en noms de domaine ;
  AUCUN mot de personne (ma, mes, mon, moi, notre, nous, ta, tes, ton, me, je, vous, votre,
  vos, « chez nous ») ; « équipe » et JAMAIS « camp » ; le joueur = son gamertag ; groupes
  « Équipe » / « Adversaire » / « Reste de l'équipe ». Aides (i) : une ou deux phrases, la
  mesure et son périmètre, sans phrase de lecture ni conseil. Termes retenus : Contrôle des
  ressources, Contribution aux prises, Rapport de force, Usage d'équipements, Isolement,
  Rendement des ressources, Frags par ressource, Rendement par ressource, Part du joueur à
  l'objectif, Prises par joueur, Portée.
- **2026-10-07** — « éliminé » plutôt que « fragué » dans une phrase (fiches némésis de la
  Vue match, `94ac8fd68`). « Frags » reste le nom de la mesure.

Ce lot n'ajoute aucune statistique : il supprime et il reformule.

## 2. État au 2026-10-07

- Les trois pages solo sont fusionnées dans `feat/v75` : Séries temporelles (`b033d30f0`,
  correctifs `be078e29d`), Sessions (`2668848b1`), Vue match (`19ec2c8ba`). Plans clos :
  `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, `.ai/PLAN_SESSIONS_EMPRISE_2026-10-06.md`,
  `.ai/PLAN_MATCHVIEW_EMPRISE_2026-10-06.md`. Riposte et hauteur n'existent plus sur ces trois
  pages ; Sessions garde « Appui reçu » (décision : Halo 5 le reçoit aussi).
- La garde des textes `apps/web/src/features/timeseries/usages/textesSansPersonne.test.ts`
  couvre les jeux de textes de Séries temporelles, Sessions, Vue match, rejeu, et de l'Escouade
  SEULEMENT `empriseStrings`, `placementStrings`, `objectifStrings` (plus, pour le jeu
  `squad/i18n.ts`, les seules parties que Sessions lit : `performanceCharts`, `weaponKills`,
  `empty`). Manifestes lus : `timeseries.toml`, `session.toml`, `match_view.toml`, bloc
  `synthesis.weapon_range.*`. Le reste de `squad/i18n.ts` et les `*Strings.ts` de Synergies ne
  sont PAS gardés.
- **Lot vivant à ne pas croiser** : Tactique v2 (session levelup-3f, worktree
  `LevelUp-wt-tactique`, branche `feat/tactique-v2`) travaille `features/tactical/` et
  `service/tactical_service*.go`. Sur `feat/v75` à `94ac8fd68`, la page Tactique affiche
  encore un taux de « morts vengées » (`tactical_service.go:358-382`, `mesurerEchange` via
  `coordination.Echanges`) ; `feat/tactique-v2` (`0641aceb6`, vérifié le 07/10 : plus de
  `mesurerEchange`, plus de `TacticalCoordinationCard`) l'a DÉJÀ retiré et sa fusion dans
  `feat/v75` est imminente. Ce lot ne touche pas Tactique ; il part de la tête de `feat/v75`
  APRÈS cette fusion (vérifier : `git grep mesurerEchange` vide), et à ce moment le SEUL
  lecteur de production de `coordination.Echanges` / `Mesurer` est
  `service/teammates/teammates_squad_echange.go` — donc ces helpers partent avec lui (§3).
- Registre `.ai/REGISTRE_REPORTS.md`, section « Pages solo aux formes de l'Emprise » et
  lignes du 2026-10-07 du lot Vue match : les entrées « Match pas encore synchronisé tutoie »
  et « Textes de l'Escouade à la personne et avec camp » sont à CLORE par ce lot.

## 3. Périmètre A — Escouade › Synergies, section « Coordination »

Fichier pivot : `apps/web/src/features/squad/SquadSynergiesPage.tsx`, section montée par
`(echange || assistPairs || rangeProfiles)` (L151), trois rangées (L170-186) :

| Rangée | Aujourd'hui | Après |
|---|---|---|
| 1 | « Appui » (`SquadAppuiCard`, bloc `assist_pairs`) | GARDÉE telle quelle |
| 2 | « Morts ripostées » + « Temps de riposte », puis « Riposte » (frise et repli) — `SquadRiposteCard`, bloc `echange` | SUPPRIMÉE |
| 3 | « Rôles de portée » et « Rôles de hauteur » côte à côte — `SquadRangeRolesCard`, la seconde avec `grandeur="hauteur"`, bloc `range_profiles` | « Rôles de portée » seule, dans sa colonne de gauche |

À supprimer côté web (règle 7 de CLAUDE.md : ce qui perd son dernier lecteur part avec ses
tests, types, fixtures, clés i18n et entrées de contrat) :

- `SquadRiposteCard.tsx` (+ `.test.tsx`), `SquadRiposteMatricePanel.tsx` (+ test),
  `SquadRiposteSessionsChart.tsx`, `squadRiposte.logic.ts` (+ test), `squadRiposte.fixtures.ts`,
  `squadRiposte.i18n.test.ts`, et dans `squadRiposteStrings.ts` tout ce qui n'est pas le
  titre et l'aide de la section (voir ci-dessous) ;
- la variante `grandeur="hauteur"` de `SquadRangeRolesCard.tsx` (prop `grandeur`, branche
  `hauteur` L88, `seriesPortee(..., grandeur)`), `profilHauteur` et la lecture
  `elevation_median_m` / `elevation_lobby_delta_m` dans `squadRangeRoles.logic.ts` (L65 et
  suivantes), le préfixe « hauteur » de `squadRangeRolesStrings.ts`, leurs tests
  (`squadRangeRoles.logic.test.ts` L193-243, `SquadRangeRolesCard.test.tsx`) ; vérifier
  `SquadRangeRolesTape.tsx` ;
- le titre et l'aide de la section : `tRiposte.coordinationTitle`,
  `coordinationHelpRiposte(...)`, `coordinationHelpAppui` (L152-162). La section devient
  « Appui et portée » (ou un nom aussi court, en nom de domaine), son aide = deux phrases,
  l'appui et la portée, sans personne ; la condition de montage devient
  `(assistPairs || rangeProfiles)`. Déplacer ces deux chaînes hors de `squadRiposteStrings.ts`
  (qui disparaît) vers `squadRangeRolesStrings.ts` ou le jeu de l'Appui ;
- le commentaire de section (L128-150) : le réécrire au présent (deux cartes, deux rangées),
  sans l'histoire des huit cartes — l'histoire va au journal.

À supprimer côté Go et contrat :

- le bloc `echange` du pageData de la page Escouade : `domain/squad_echange.go`
  (`SquadEchange`, L110-115 et la suite), son producteur
  `service/teammates/teammates_squad_echange.go` (lectures `coordination.Echanges` L133/142/411
  et `coordination.Ripostes` L317/352), son câblage dans `teammates_service_sections.go` /
  `teammates_service.go`, ses tests (`teammates_squad_echange_contrat_test.go`,
  `teammates_service_loads_test.go` pour la part échange), le champ du contrat
  (`make openapi-gen && make generate-types`, snapshot `contract-surface.snapshot.json`,
  `tools/lint-contract-ratchet.mjs` si une entrée le nomme) ;
- `coordination.Ripostes`, `coordination.Echanges` et `coordination.Mesurer`
  (`analysis/coordination/riposte.go` et voisins) : après la fusion de Tactique v2 (§2),
  leur seul lecteur de production est `teammates_squad_echange.go` ; ils partent donc avec
  lui, tests compris (`riposte_test.go`, la part « échange » de `no_naked_rate_test.go`,
  `bloc_appui_golden_test.go` à relire : l'APPUI reste). Vérifier par
  `git grep -n 'coordination\.\(Echanges\|Ripostes\|Mesurer\)'` qu'il ne reste aucun lecteur
  hors tests avant de supprimer. `domain/coordination.go` : `PaireEchange` (L85-100) et les
  types d'échange partent avec leur dernier lecteur ; ce qui sert à l'Appui reste ;
- le dénivelé des profils de portée : `domain/match_range_profile.go` champs
  `ElevationMedianM` (L48), `ElevationLobbyDeltaM` (L53), `LobbyElevationMedianM` (L79), et
  le calcul qui les remplit (le 07/10, aucun fichier de `platform/duckdb/` ne porte le mot
  `elevation` : le calcul est côté `analysis/` ou `service/`, le trouver par
  `grep -rn ElevationMedianM apps/go-api/internal` ; le lecteur `kill_measured.go` a déjà
  perdu ses colonnes de gamertag en M6 de la Vue match) — SEULEMENT si le grep web ne
  montre plus de lecteur hors
  `squadRangeRoles.logic.ts` (le 07/10 : aucun autre). Contrat régénéré ;
- garde-rails à relire après coup : `archlint` (ratchets de surface et de taille : re-mesurer
  à l'entrée, cf. mémoire « ratchet gelé »), `analysis/coordination/no_naked_rate_test.go`
  (il nomme des taux ; retirer ceux de la riposte), baseline
  `.ai/baselines/tests_pre_migration.jsonl` : tout test SUPPRIMÉ ou RENOMMÉ s'en retire avec
  un paragraphe daté dans `scripts/check_test_baseline.sh` (comparer la baseline aux
  fonctions `Test*` de l'arbre AVANT et APRÈS, pas seulement la liste qu'on croit avoir
  touchée — leçon Sessions S8).

Ce qui ne change pas : `SquadAppuiCard`, le bloc `assist_pairs`, le nuage de portée et son
Go, l'onglet Emprise, l'onglet Objectif, la page Tactique.

## 4. Périmètre B — textes de l'Escouade et un message de la Vue match

Reformuler selon la sémantique du 06/10, FR et EN, puis ÉTENDRE la garde pour qu'elle
refuse toute réintroduction :

- `apps/web/src/features/squad/i18n.ts` — au moins : L651 et L653 (« vous rapporte »,
  « vous survivez » : aides Rendement et Résistance — reprendre les aides de la Vue match
  déjà réécrites en mesure et portée), L764 (`prompt: 'Pick up to 3 teammates to analyze
  your synergies.'`), l'aide du score à manches (« au camp qui a perdu » — chercher
  `camp` dans le fichier) ; le grep du 07/10 compte 7 chaînes portant camp / vous / votre /
  vos / your dans `squad/i18n.ts` et les `*Strings.ts` de Synergies : les traiter toutes, FR
  et EN ;
- `apps/web/src/features/match-view/i18n.ts` — `notSyncedDescription` FR (L385 et
  suivantes : « reviens dans quelques minutes », « Vérifie aussi ») et EN (L718) : une
  phrase factuelle sans impératif ni personne (« Match pas encore synchronisé ; nouvelle
  tentative dans quelques minutes. Si le match n'apparaît pas, la synchronisation du
  joueur n'est peut-être pas active. » — à ajuster) ;
- la garde `textesSansPersonne.test.ts` : ajouter le jeu `squad/i18n.ts` EN ENTIER, les
  `*Strings.ts` de `features/squad/` (dont `squadRangeRolesStrings.ts`, le jeu de l'Appui),
  le manifeste `squad.toml` (`apps/web/src/lib/i18n/manifests/`) en entier ; compléter
  la liste FR avec « vous », « votre », « vos », « tu », « toi », « reviens », « vérifie »
  si elle ne les porte pas déjà, et la liste EN avec « you », « your » (bornes de mot : la
  garde utilise `\p{L}\p{N}` des deux côtés, « vouloir » ne doit pas tomber). La voir ROUGE
  sur l'état actuel, VERTE après la reformulation — c'est le gate du périmètre B.

Hors périmètre : les textes de `features/tactical/` (lot Tactique v2), les notes de
version (rédigées en « tu », décision antérieure).

## 5. Protocole d'exécution

1. Worktree dédié `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-escouade`, branche
   `feat/escouade-synergies-textes` depuis `origin/feat/v75` (jamais `wt/**` : pas de CI).
   `npm install` local dans `apps/web` du worktree plutôt qu'une jonction `node_modules`
   (les retraits de worktree suivent les jonctions — incident du 16/09).
2. Plan court `.ai/PLAN_ESCOUADE_SYNERGIES_TEXTES_2026-10-07.md` sous le contrat du skill
   `plan-execution` : E1 web Synergies, E2 Go + contrat, E3 textes + garde, E4 clôture
   (CHANGELOG EN/FR, journal, registre). Un commit par étape. Décisions tranchées en §1 et
   §3 : ne rien rediscuter, consigner les découvertes sans les corriger.
3. Gate par étape : Go `go build ./...`, `go vet`, tests des paquets touchés puis
   `go test ./internal/...` à la clôture, `-tags=integration -p 1` si `platform/duckdb` est
   touché, `make openapi-check` ; web tsc purgé (`node_modules/.tmp`), `npm run lint`,
   `lint:fields`, vitest, knip (aveugle en local : la CI Linux juge), `lint-no-hardcoded-colors`.
   CGO : `PATH=/c/msys64/ucrt64/bin:$PATH CGO_ENABLED=1 CC=gcc.exe` ; un seul `go` à la
   fois sur le poste.
4. Revue adversariale (skill `adversarial-review`), un relecteur Opus en contexte frais,
   lentilles L3 (code mort après suppression, commentaires périmés), L6 (la garde détecte-t-elle
   une chaîne fautive réintroduite ?), L5 ; deux rondes au plus.
5. Push, CI de branche verte (`gh run watch`), puis demande de fusion à l'utilisateur ;
   fusion `--no-ff` depuis une tête détachée sur `origin/feat/v75` du moment, gate rejoué sur
   le résultat, push, avance rapide du dossier principal (journal non commité à mettre de
   côté et réappliquer), prévenir les sessions qui fusionnent derrière.
6. Clôture : entrée `.ai/thought_log.md`, registre (deux lignes closes), mémoire du projet.

## 6. Prompt de lancement de l'exécutant (à coller tel quel, agent Opus)

```
Tu exécutes le lot « Escouade › Synergies : retrait de la riposte et de la hauteur, textes
sans personne » dans le dépôt LevelUp. Lis d'abord, dans cet ordre :
`CLAUDE.md`, `.ai/HANDOFF_ESCOUADE_SYNERGIES_TEXTES_2026-10-07.md` (ce document fait foi
pour le périmètre et les décisions), `.claude/skills/plan-execution/SKILL.md`,
`.claude/skills/frontend-patterns/SKILL.md`, `.claude/skills/arch-rules/SKILL.md`, et les
entrées du 2026-10-05 au 2026-10-07 de `.ai/thought_log.md`.
Crée le worktree et la branche du §5, écris le plan court du §5.2, soumets-le-moi, puis
exécute-le étape par étape sur mon « go » : ordre strict, gate par étape, chaque item statué,
zéro correction hors périmètre (les découvertes vont au §8 du plan). Ne touche ni
`features/tactical/` ni `service/tactical_service*.go`. Compte rendu à chaque étape : sha,
fichiers, ce que l'utilisateur verra à l'écran (titres et aides FR avant → après), gate,
découvertes. Ni merge ni rebase sans mon signal.
```
