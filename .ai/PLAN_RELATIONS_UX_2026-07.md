# PLAN — Améliorations UX page Relations (Communauté) — 2026-07

> Plan d'exécution pour agent (Opus). Contrat : skill `plan-execution` (ordre strict,
> une étape à la fois, gate par lot, statuts [x]/[~]/[!], zéro fix hors périmètre).
> Rédigé le 2026-07-07 après revue UX de la page avec l'utilisateur — toutes les
> décisions produit ci-dessous sont TRANCHÉES, ne pas les rouvrir.

## Contexte et objectif

La page Communauté > Relations (`apps/web/src/features/palmares/PalmaresRelationsPage.tsx`,
route canonique `/players/$playerSlug/community/relations`) est bien exécutée mais
descriptive : elle redonne la même donnée plusieurs fois et n'offre ni classement, ni
navigation vers les matchs, ni raison d'y revenir. Diagnostic validé avec l'utilisateur.

**Objectif** : transformer la page d'« almanach consulté deux fois » en page utile et
vivante : tableau ordonnançable, duels cliquables vers la match view, volet « Quoi de
neuf », notification « rival croisé » post-sync, suppression de la triple redondance
noyau dur.

**Critère de succès global** : les 7 lots obligatoires (A, B, C, H, F, D, E) livrés et
verts (le lot G est conditionnel), suites Go + web vertes, aucune régression sur les
consommateurs partagés (`OutcomeSequenceTape` est utilisé par 5 autres pages).

**Décisions produit actées (ne pas rediscuter)** :
- Pas de champ de recherche dans le tableau (redondant avec l'Explorer mode joueur,
  accessible en 1 clic depuis chaque gamertag). Le TRI, lui, est retenu.
- La heatmap « Rythme des rencontres » est conservée telle quelle (choix utilisateur).
- Le toggle « amis » est conservé mais relibellé selon sa vraie sémantique
  (« jamais affrontés ») et son défaut passe à MASQUÉ.
- La section détaillée CoreCards (bas de page) est SUPPRIMÉE (triple redondance).
- CSR du rival = lot G optionnel/conditionnel uniquement (données classées seulement).
- Multi-jeux = un CHIP de plus dans le segmented control (tranche de population, comme
  tous/noyau/alliés/rivaux/récents), PAS un toggle : le toggle est réservé aux
  préférences d'affichage orthogonales. Filtre 100 % client sur le badge cross-jeu
  déjà servi (Phase 3b) — aucun changement backend.

## Références de code (vérifier sur pièces avant chaque lot — le code a pu bouger)

| Élément | Emplacement |
|---|---|
| Page | `apps/web/src/features/palmares/PalmaresRelationsPage.tsx` |
| Tableau | `apps/web/src/features/palmares/RelationsTable.tsx` |
| Cartes rivaux | `apps/web/src/features/palmares/RelationsRivalryCards.tsx` |
| Section moments | `apps/web/src/features/palmares/RelationsMomentsSection.tsx` |
| Filtre client + prefs | `apps/web/src/features/palmares/relationsFilter.ts`, `apps/web/src/stores/relationsPrefsStore.ts` |
| Frise partagée | `apps/web/src/components/charts/OutcomeSequenceTape.tsx` (5 autres consommateurs : Home, session-detail, lab, timeseries, squad) |
| i18n page | `apps/web/src/lib/i18n/manifests/palmares.toml` → regen `node apps/web/scripts/build_i18n_manifests.mjs` → `lib/i18n/generated/palmares.ts` (adapter `features/palmares/i18n.ts`) |
| Algos purs | `apps/go-api/internal/analysis/relations/{relations,overview}.go` |
| Service | `apps/go-api/internal/service/relations_service.go`, `relations_moments_service.go` |
| Repo SQL | `apps/go-api/internal/platform/duckdb/relations_repo.go`, templates `Q28RelationsTpl` / `Q28RelationsScopedTpl` dans `queries_career_encounters.go` |
| DTO | `apps/go-api/internal/domain/relations.go` (`RelationInsight`) + `apps/go-api/api/openapi.yaml` (manuel — migration Huma terminée) → `make generate-types` |
| Notifications | `apps/go-api/internal/notifications/types.go` (catégories), `internal/api/wire/post_sync_deltas.go` (hook post-sync), `internal/migration/steps_player_notifications.go` (seed prefs), front `apps/web/src/features/notifications/{i18n,navigation,format}.ts` |
| Route match view | `/players/$playerSlug/matches/$matchId` |

## Branche et livraison

- Branche : **`feat/relations-ux-2026-07`**, créée depuis `main` À JOUR au moment de
  l'exécution. Attention : le chantier audits (`refactor/audits-2026-07`) est en attente
  de merge — si son merge dans `main` n'est pas encore fait, le signaler à l'utilisateur
  AVANT de commencer (risque de conflit sur les fichiers relations) et attendre sa
  décision. Ne JAMAIS travailler sur `main` (push main = deploy prod auto).
- 1 branche, N commits : un commit par lot (`feat(relations-A): ...` etc.), au fil de
  l'eau sans redemander (autonomie accordée sur ce chantier). Pas de `git stash`.
- Entrée `.ai/thought_log.md` à la clôture de chaque lot.
- Pas d'emojis dans les fichiers. Strings UI en FR **et** EN (parité typée). FR sans
  anglicismes. Aucune couleur hex/classe Tailwind couleur dans features/ (tokens only).

## Gates communs (à exécuter tels quels)

```
# GATE-GO (lots D, E) — depuis apps/go-api/
go test ./...          # code de sortie 0 exigé (filtrer avec '^--- FAIL:' seulement)
go vet ./...
# Lot E uniquement (touche internal/migration/) :
go test -tags=integration -p 1 ./...   # -p 1 NON NÉGOCIABLE (DuckDB mono-process)

# GATE-WEB (tous les lots) — depuis apps/web/
Remove-Item -Recurse -Force node_modules\.tmp   # purge tsBuildInfo (anti faux vert)
npm run typecheck
npm run lint
npx vitest run src/features/palmares src/features/notifications src/stores  # hors sandbox (dangerouslyDisableSandbox)

# GATE-I18N (lots C, D, E, F) — après édition de palmares.toml
node apps/web/scripts/build_i18n_manifests.mjs   # puis vérifier le diff de generated/palmares.ts

# GATE-TYPES (lot D) — après édition de openapi.yaml
make generate-types    # régénère apps/web/src/lib/api/generated.ts
```

« Lot clos » = tous ses items statués + son gate passé (code de sortie vérifié) +
commit + entrée thought_log. Ne pas commencer le lot suivant avant.

---

## Lot A — Tri des colonnes du tableau (front pur, effort : rapide)

Le tableau (`RelationsTable.tsx`) n'a ni `getSortedRowModel` ni accesseurs : ordre serveur
figé (matchs communs DESC). Objectif : en-têtes cliquables, tri client.

- [x] A1. Ajouter un `accessorFn` à chaque colonne triable + `getSortedRowModel()` :
      - `player` → `gamertag` (alpha, insensible à la casse)
      - `encounters` → `total_matches`
      - `wr_ally` → `teammate_win_rate` ; `wr_enemy` → `enemy_win_rate`
      - `frags_deaths` → net `kills_dealt - deaths_suffered`
      - `ratio` → `duel_ratio`
      - `last_seen` → timestamp epoch de `last_seen_at`
      - `link` (catégorie) → NON triable (`enableSorting: false`)
      FAIT : accessorFn + `sortDescFirst: true` sur les colonnes numériques/date
      (1er clic = décroissant, attendu par A5a) ; `player` alpha asc-first.
- [x] A2. Valeurs nulles toujours en fin de liste quel que soit le sens : accessor
      retourne `undefined` pour null/NaN + `sortUndefined: 'last'` sur ces colonnes.
      FAIT via helpers `numOrUndef` / `lastSeenEpoch`.
- [x] A3. En-têtes : bouton cliquable (cycle asc→desc→none), indicateur de sens en
      caractère texte (pas d'icône emoji), `aria-sort` sur le `<th>` actif. Pas d'état
      de tri initial (ordre serveur conservé tant qu'on ne clique pas).
      FAIT : helper `SortLabel` (bouton + indicateur `↑`/`↓`) ; pour « Ratio » le
      Tooltip enveloppe le bouton (`div > button` valide). `aria-sort` sur le `<th>`.
- [x] A4. Vérifier que le changement de tri réinitialise la pagination page 1
      (comportement TanStack `autoResetPageIndex` par défaut — tester, ne pas supposer).
      VÉRIFIÉ : `autoResetPageIndex` ne réinitialise PAS de façon fiable (ni sync ni
      async) dans notre harnais → reset explicite piloté via état contrôlé
      (`onSortingChange` remet `pageIndex: 0`). Test dédié (30 lignes, page 2 → tri →
      page 1). Découverte consignée en bas de plan.
- [x] A5. Tests vitest (nouveau `RelationsTable.test.tsx`) :
      (a) clic sur « Ratio » → ordre décroissant attendu sur un jeu de 4 lignes dont une
      à ratio null (null en dernier) ; (b) second clic → ordre inversé, null toujours
      en dernier ; (c) `aria-sort` présent. FAIT (+ tri alpha casse-insensible + « Lien »
      non triable).

**Gate A** : GATE-WEB.

## Lot B — Duels cliquables vers la match view (front, effort : moyen)

`OutcomeSequenceTape` (ECharts custom series, runs RLE) porte déjà `matchId` par point
mais n'a aucun clic. Les cartes rivaux deviennent le seul endroit cliquable (la
mini-frise `OutcomeSparkline` du hero bête noire reste décorative `aria-hidden`).

- [ ] B1. `OutcomeSequenceTape.tsx` : prop optionnelle `onMatchClick?: (matchId: string) => void`.
      Sans la prop, comportement STRICTEMENT identique (5 autres consommateurs — zéro
      régression : ne rien changer d'autre à l'option ECharts par défaut).
- [ ] B2. Résolution du match cliqué : extraire un helper pur exporté
      `matchIndexAtX(runs, xValue)` (xValue continu 0..xMax → index global borné →
      match). Câblage : `onEvents={{ click }}` + instance chart (`onChartReady`) +
      `convertFromPixel` pour retrouver xValue depuis `event.event.offsetX/offsetY`.
      Cas dégradé : si la conversion échoue, retomber sur le premier match du run
      cliqué (`params.dataIndex`).
- [ ] B3. Quand `onMatchClick` est fourni : `cursor: 'pointer'` sur les rects du
      renderItem (propriété zrender par élément), pas de changement visuel autre.
- [ ] B4. `RelationsRivalryCards.tsx` : accepter `onMatchClick` et le passer au tape ;
      `RelationsMomentsSection.tsx` : construire la navigation
      `navigate({ to: '/players/$playerSlug/matches/$matchId', params })` (playerSlug
      déjà en props) et la passer aux cartes.
- [ ] B5. Tests : (a) test unitaire pur de `matchIndexAtX` (bornes : x négatif, x ≥ xMax,
      runs vides, run unique) ; (b) test vitest de non-régression : tape SANS prop rend
      comme avant (mocker `echarts-for-react` — cf. mémoire jsdom, `vi.mock` sinon crash
      canvas).

**Gate B** : GATE-WEB + vérification manuelle rapide de 2 consommateurs non modifiés
(Home, squad synergies) en typecheck (aucune prop requise ajoutée = aucun changement).

## Lot C — Toggle « jamais affrontés » : libellé honnête + défaut masqué (rapide)

Le toggle actuel dit « amis » mais masque en réalité les relations jamais affrontées
(`enemy_matches === 0`). Décisions : renommer selon la sémantique réelle, défaut = masqué.

- [ ] C1. `relationsPrefsStore.ts` : renommer `includeFriends` → `includeNeverFaced`,
      défaut `false`. Bump `version: 2` + `migrate` v1→v2 : réinitialiser la valeur à
      `false` (changement de sémantique assumé, une seule fois), conserver `filter` et
      `heatmapMode`. Commenter la raison + date dans le migrate.
- [ ] C2. `PalmaresRelationsPage.tsx` : renommer les usages (setter, filtre
      `visibleRows`). La logique ne change pas : OFF → masquer `enemy_matches === 0`.
- [ ] C3. i18n `palmares.toml` : remplacer les clés du toggle par la nouvelle sémantique.
      Libellés actés — état ON : FR « Jamais affrontés inclus » / EN « Never-faced
      included » ; état OFF : FR « Inclure les jamais affrontés » / EN « Include
      never-faced players ». Supprimer les anciennes clés (0 code mort). Regen manifest.
- [ ] C4. Mettre à jour les tests existants qui référencent l'ancien libellé/état par
      défaut (`PalmaresRelationsPage.test.tsx` et tout test du store).
- [ ] C5. Ajouter un test du migrate v1→v2 (persisted v1 avec `includeFriends: true` →
      state v2 `includeNeverFaced: false`, `filter`/`heatmapMode` préservés).

**Gate C** : GATE-WEB + GATE-I18N.

## Lot H — Chip « Multi-jeux » dans le segmented control (front pur, rapide)

Voir les joueurs croisés sur plusieurs jeux. Le backend sert déjà l'information
(Phase 3b) : badge `label_key = "narrative.encounter.cross_game"` dans
`relations[].badges`, avec `detail.game` et `detail.matches_together` (seuil serveur
`CrossGameMinMatchesTogether = 3`, best-effort — si l'enrichissement cross-titre échoue,
le badge est simplement absent). Aucun changement Go.

- [ ] H1. Étendre l'union `RelationFilter` avec `'cross'` dans `relationsFilter.ts` ET
      dans son miroir `relationsPrefsStore.ts` (les deux unions doivent rester
      identiques — commentaire de miroir déjà en place). Pas de migration du store :
      valeur additive, les états persistés existants restent valides.
- [ ] H2. Prédicat `isCrossGame(r)` : au moins un badge dont `label_key` est le littéral
      cross-jeu. Le littéral `"narrative.encounter.cross_game"` doit exister en UNE
      constante exportée côté front (vérifier si `RelationBadges.tsx` ou un module badges
      le porte déjà ; sinon la définir dans `relationsFilter.ts` et l'importer partout —
      règle des ≤ 2 copies).
- [ ] H3. Chip « Multi-jeux » ajouté au segmented control (`FILTER_CHIPS` +
      `SegmentedFilter`), clés i18n FR « Multi-jeux » / EN « Multi-game » dans
      `palmares.toml` (section chips) + regen manifest.
- [ ] H4. Affichage conditionnel : le chip n'est RENDU que si au moins une relation
      porte le badge (pas de segment mort pour les profils mono-titre). Garde-fou : si
      le filtre persisté vaut `'cross'` alors que le chip est masqué (données devenues
      vides), traiter comme `'all'` sans écrire dans le store.
- [ ] H5. Le chip compose normalement avec le toggle « jamais affrontés » (même
      pipeline `visibleRows`) — aucun cas spécial.
- [ ] H6. Tests vitest : (a) `filterRelations('cross')` ne garde que les relations au
      badge cross-jeu ; (b) chip absent quand aucune relation multi-jeux ; (c) chip
      présent + clic → tableau réduit aux bonnes lignes ; (d) filtre persisté `'cross'`
      sans donnée → rendu identique à « tous ».

**Gate H** : GATE-WEB + GATE-I18N.

## Lot F — Dé-redondance noyau dur + pont vers Escouade (rapide)

Frontière actée : Escouade = « nous » (groupe déclaré), Relations = « le monde croisé ».
Le noyau dur apparaît 3 fois sur la page ; la section détaillée n'ajoute rien au tableau.

> Note d'ordre : F est exécuté AVANT D pour stabiliser la structure de page avant d'y
> ajouter le volet « Quoi de neuf ».

- [ ] F1. Supprimer la section « Noyau dur » détaillée : composant `CoreCards` + son
      rendu dans `RelationsContent` + le bloc titre/description de section. Supprimer
      les clés i18n devenues orphelines (`core.sectionTitle`, `core.sectionDescription`,
      `core.together`, `core.empty` — VÉRIFIER par grep qu'elles ne sont pas utilisées
      ailleurs avant suppression ; celles partagées avec la carte hero restent).
- [ ] F2. La carte hero « Noyau dur » et le chip « noyau » du tableau restent inchangés.
      Ajouter dans le pied de la carte hero noyau un lien discret vers la page Escouade :
      FR « Voir l'escouade » / EN « View squad » → route `/players/$playerSlug/squad`
      (nouvelles clés i18n FR+EN).
- [ ] F3. Purger les tests/snapshots qui référencent CoreCards ; ajouter l'assertion
      inverse (la section n'est plus rendue) dans le test de page.

**Gate F** : GATE-WEB + GATE-I18N + grep `CoreCards` = 0 occurrence hors historique git.

## Lot D — Volet « Quoi de neuf » (full-stack, effort : moyen-lourd)

Donner une raison de revenir : nouvelles têtes et retrouvailles. Bande compacte rendue
entre les cartes hero et les chips de filtre, UNIQUEMENT si non vide.

**Définitions actées** (constantes nommées dans `analysis/relations`, pas de magic number) :
- Nouvelle tête : `first_seen_at` dans les 30 derniers jours (`NewFaceWindowDays = 30`).
  Calculable CÔTÉ CLIENT (`first_seen_at` déjà dans le DTO) — pas de SQL.
- Retrouvailles : au moins 1 rencontre dans les 30 derniers jours ET la dernière
  rencontre AVANT cette fenêtre remonte à ≥ 90 jours (`RevivedWindowDays = 30`,
  `RevivedMinGapDays = 90`). Nécessite 2 agrégats SQL nouveaux.

- [ ] D1. SQL — étendre `Q28RelationsTpl` ET `Q28RelationsScopedTpl`
      (`queries_career_encounters.go`) avec 2 colonnes :
      `encounters_30d` = COUNT(*) FILTER (rencontres ≥ now − 30 j) et
      `prev_seen_before_window` = MAX(ts) FILTER (rencontres < now − 30 j).
      OBLIGATOIRE : réutiliser le fragment timezone canonique déjà employé dans ces
      templates (`COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`) — jamais
      `start_time` brut. Étendre `scanRelationRow` + `domain.RelationRawRow`.
- [ ] D2. `analysis/relations` : constantes + fonction pure
      `IsRevived(encounters30d int, prevSeen *time.Time, now time.Time) bool`.
      Tests unitaires purs (cas : jamais vu avant, gap 89 j, gap 90 j, 0 rencontre
      récente, prevSeen nil).
- [ ] D3. Service : mapper vers le DTO — `RelationInsight` += `is_revived bool`
      (source unique côté serveur, même modèle que `is_core`). `openapi.yaml` mis à
      jour (schéma RelationInsight) + GATE-TYPES.
- [ ] D4. Repo test (`relations_repo_test.go`, DuckDB en mémoire) : fixture avec une
      relation « ravivée » (rencontres à now−200 j et now−5 j) et une non ravivée
      (rencontres régulières) → colonnes SQL correctes sur les DEUX templates
      (scopé et non scopé).
- [ ] D5. Front — composant `RelationsWhatsNewStrip` (nouveau fichier dans
      `features/palmares/`) : deux groupes rendus seulement si non vides,
      « Nouvelles têtes (30 j) » (client : `first_seen_at` ≤ 30 j) et
      « Retrouvailles » (`is_revived`), max 5 gamertags par groupe (+ compteur « +N »),
      gamertags cliquables → Explorer (réutiliser `goToExplorer`). Aucun appel réseau
      nouveau : tout vient de `data.relations`. Clés i18n FR/EN :
      FR « Quoi de neuf » / « Nouvelles têtes » / « Retrouvailles » (+ tooltip
      expliquant les fenêtres 30 j / 90 j) ; EN « What's new » / « New faces » /
      « Reunions ». La logique de sélection (fenêtres, tri, plafond 5) dans un
      `whatsNew.ts` pur à côté de `relationsFilter.ts`, testé en vitest.
- [ ] D6. Tests front : strip absent quand aucune donnée ne matche ; présent avec les
      bons gamertags sinon ; `whatsNew.ts` testé unitairement (bornes de fenêtre).

**Gate D** : GATE-GO + GATE-WEB + GATE-I18N + GATE-TYPES.

## Lot E — Notification « rival croisé » post-sync (full-stack, effort : lourd)

Le moment « revanche » se joue juste après le match : émettre une notification in-app
quand une sync ramène un nouveau duel contre un des top rivaux.

**Design acté** :
- Détection par WATERMARK, pas par re-scan : `PlayerSnapshot` (post_sync_deltas.go)
  += `LastMatchStartTime` (MAX du start canonique des matchs du joueur dans le shared —
  une requête légère, timezone canonique). Capturée avant ET après la sync.
  Si `after <= before` (aucun nouveau match) → détection SKIPPÉE (coût nul sur les
  syncs à vide, qui sont la majorité).
- Sinon : réutiliser le chemin existant top rivaux (`GetRelations` + logique
  `selectTopRivals`, seuils existants) + `GetRivalTimeline` (limite existante) ; un
  duel est « nouveau » si `started_at > before.LastMatchStartTime`. Le watermark rend
  l'émission idempotente (une re-sync des mêmes matchs ne ré-émet pas).
- Garde-fous nommés : `rivalNotifMaxPerSync = 3` (anti-spam après longue absence) et
  `rivalNotifMaxAgeDays = 7` (pas de notification pour un duel plus vieux — cas
  backfill/import).
- Catégorie : `rival_encounter`. Severity : `SeveritySuccess` si duel gagné,
  `SeverityInfo` sinon. `TargetRoute` : la match view du duel
  (`/players/{slug}/matches/{match_id}`). `Source` : `post_sync`.
  Params : `gamertag`, `outcome` (win|loss|other), `kills`, `deaths`, `match_id`.

- [ ] E1. `notifications/types.go` : + `CategoryRivalEncounter Category = "rival_encounter"`.
      Synchroniser les 2 dépendants documentés dans l'en-tête du fichier :
      `migration/steps_player_notifications.go` (seed préférence, activée par défaut)
      et le front (E5). Étendre le test du seed.
- [ ] E2. Détection : méthode de service dédiée (ex.
      `RelationsService.DetectRivalEncounters(ctx, since time.Time) ([]RivalEncounter, error)`)
      — orchestration dans `internal/service`, AUCUN SQL nouveau (réutilise
      `port.RelationsRepository`). Type de retour dans `internal/domain`. Tests service
      avec mock repo : duel récent → détecté ; duel antérieur au watermark → non ;
      plafonds `maxPerSync`/`maxAgeDays` respectés.
- [ ] E3. `post_sync_deltas.go` : + `LastMatchStartTime` dans `PlayerSnapshot`
      (via une méthode repo légère — si une donnée équivalente existe déjà dans la
      snapshot, la réutiliser : VÉRIFIER sur pièces d'abord) ; câbler la détection +
      l'émission dans la closure post-sync, best-effort (`slog.WarnContext` et on
      continue, jamais d'échec de sync à cause d'une notif). L'accès au service
      relations depuis le `ServiceRegistry` suit le modèle des autres services du
      registry (vérifier l'existant `reg.*` avant d'inventer).
- [ ] E4. Test wire (`post_sync_deltas_test.go` ou fichier sœur) : sync avec nouveau
      duel rival → 1 notification `rival_encounter` émise avec les bons params ;
      sync sans nouveau match → 0 émission (et détection non appelée).
- [ ] E5. Front notifications : `features/notifications/i18n.ts` +
      `notif.rival_encounter.title` / `.body` FR+EN
      (FR titre « Rival croisé », corps « Tu as recroisé {gamertag} : {kills} frags /
      {deaths} morts » ; EN « Rival encountered » / « You crossed paths with {gamertag}
      again: {kills} frags / {deaths} deaths ») ; `navigation.ts` : vérifier que la
      TargetRoute match view est déjà gérée (elle l'est pour d'autres catégories —
      sinon l'ajouter) ; étendre `navigation.test.ts`.
- [ ] E6. Logging : émissions et skips significatifs en `slog.InfoContext/DebugContext`
      avec clés structurées (`"rival"`, `"match_id"`, `"duels_new"`).

**Gate E** : GATE-GO **avec** `go test -tags=integration -p 1 ./...` (le lot touche
`internal/migration/`) + GATE-WEB.

## Lot G (CONDITIONNEL) — Contexte CSR sur la bête noire (optionnel, moyen)

À n'exécuter QUE si les lots A–F sont clos et si la vérification de couverture passe.
Sinon statuer `[!]` avec la mesure observée — ce n'est pas un échec du plan.

- [ ] G0. Vérification de couverture (gate d'entrée) : sur la copie locale des données
      (`duckdb` CLI, lecture seule), mesurer la part des rivaux (enemy_matches >= 8)
      ayant au moins une ligne CSR exploitable dans le shared (vues `_latest`
      UNIQUEMENT — règle ART n° 2). Si < 30 % des rivaux couverts → statuer `[!]`
      et s'arrêter là.
- [ ] G1. Si couvert : exposer sur l'overview le CSR courant du top nemesis (meilleur
      snapshot le plus récent, vue `_latest`, best-effort nil si absent) — champ
      optionnel DTO + openapi + generate-types.
- [ ] G2. Carte hero bête noire : afficher le tier (libellé + couleur via les utilitaires
      CSR existants du front — chercher l'existant avant d'implémenter, skill
      `go-features` côté Go, grep `csr`/`tier` côté web). Rien affiché si absent
      (dégradation silencieuse, pas de « N/A »).
- [ ] G3. Tests : service (rival sans CSR → nil, avec CSR → valeur) + front (carte sans
      CSR inchangée).

**Gate G** : GATE-GO + GATE-WEB + GATE-TYPES.

---

## Hors périmètre (NE PAS FAIRE, même si tentant)

- Recherche/filtre texte dans le tableau (décision : redondant avec l'Explorer).
- Toucher à la heatmap (conservée telle quelle).
- Refonte de la page Escouade ou déplacement de contenus entre pages (seul le lien F2
  est au périmètre).
- Notifications push/Discord pour le rival (in-app seulement).
- Tendances par relation (WR glissant par binôme etc.) — non retenu à ce stade.
- Tout fix opportuniste hors des fichiers listés → section Découvertes.

## Découvertes en cours de route

(Consigner ici tout bug/dette rencontré hors périmètre, avec fichier:ligne — ne pas traiter.)

- Lot A / A4 — `autoResetPageIndex` (défaut TanStack Table v8) NE réinitialise PAS
  la pagination en page 1 au changement de tri dans notre harnais vitest+jsdom (test
  échoue même avec attente async `findByText`). Contournement adopté (dans le
  périmètre A4, comportement voulu) : état tri+pagination contrôlé,
  `onSortingChange` remet `pageIndex: 0`. `RelationsTable.tsx`. Pas d'action
  supplémentaire requise.

## Protocole de reprise de session

1. `git branch --show-current` (attendu : `feat/relations-ux-2026-07`) + `git log --oneline -10`.
2. Relire ce fichier : les cases cochées font foi de l'avancement ; le premier lot avec
   une case vide est le lot courant.
3. Relire les dernières entrées `.ai/thought_log.md` du chantier.
4. Re-vérifier sur pièces les fichiers du lot courant avant de coder (le code a pu bouger).

## Clôture du chantier

- [ ] Tous les lots statués (A, B, C, H, F, D, E obligatoires ; G statué même si `[!]`).
- [ ] Gate global final : suites complètes Go (`go test ./...` + `go vet`) et web
      (typecheck avec purge cache + lint + vitest run complet), CI de branche verte
      (`gh run list --branch feat/relations-ux-2026-07`) — un gate local ne couvre pas
      les jobs CI (baseline Go Linux + build Vite).
- [ ] Entrée thought_log de clôture (bilan, décisions, restes éventuels).
- [ ] Revue visuelle = l'utilisateur au merge (ne pas merger soi-même : prévenir —
      push `main` = déploiement prod automatique).
