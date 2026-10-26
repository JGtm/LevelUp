# PLAN — Explorer / recherche joueur : 3e rangée « résultats + assistances + portée » (2026-09-17)

> Date : 2026-09-17. Cadrage validé par l'utilisateur en conversation le 2026-09-17
> (disposition relue et amendée : bloc de droite à **1/3**, assistances **câblées pour de
> vrai**, portée en **placeholder** jusqu'à la livraison de l'autre agent).
> Branche d'exécution : `wt/explorer-rangee3`, worktree dédié
> `../LevelUp-wt-explorer-rangee3` créé depuis `feat/v75` (le worktree principal est
> PARTAGÉ entre agents, ne jamais y coder).
> Clôture : commits sur `wt/explorer-rangee3`, fusion dans `feat/v75` sur signal de
> l'utilisateur.
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report, tout item
> statué, zéro fix hors périmètre). Ce fichier est la source de vérité de l'avancement.

## Objectif et critère de succès

Réorganiser la section « Sur N matchs joués ensemble » de l'encart cible Explorer
(`features/explorer/ExplorerTargetProfileCard.tsx`) en **trois rangées** et y câbler un
nouveau bloc de données.

```
Rangée 1   ┌──────────────────────────────────┬──────────┐
           │ Répartition des frags      (2/3) │ Cadence  │
           │ seule dans sa colonne            │  (1/3)   │
           └──────────────────────────────────┴──────────┘
Rangée 2   ┌──────────┬──────────────────────────────────┐
           │ WR ens.  │ Écart de frags cumulé      (2/3) │
           │ WR face  │                                  │
           └──────────┴──────────────────────────────────┘
Rangée 3   ┌──────────────────────────────────┬──────────┐
           │ Répartition des résultats  (2/3) │ Portée   │
           ├──────────────────────────────────┤ des frags│
           │ Part des assistances       (2/3) │  (1/3)   │
           └──────────────────────────────────┴──────────┘
```

**Critère de succès** :
1. « Répartition des résultats » n'est plus dans la rangée 1 ; « Répartition des frags »
   occupe seule la colonne 2/3 et prend la hauteur de la rangée.
2. La rangée 3 existe sous « Écart de frags cumulé », colonne gauche 2/3 (résultats
   au-dessus, assistances en dessous), colonne droite 1/3 (portée).
3. Les hauteurs s'harmonisent : la colonne gauche impose la hauteur, le bloc de droite
   s'étire (`h-full`), comme la Cadence de la rangée 1.
4. « Part des assistances » rend la MÊME figure que la carte Binôme de Relations (barre
   papillon + « total · part » de chaque côté), alimentée par une donnée réelle ; « — »
   quand aucun match ensemble n'a de film analysé.
5. « Portée des frags » est un placeholder titré et explicite (aucune donnée inventée).
6. Gates Go et web verts, CI de branche verte au niveau job, gate visuel de l'utilisateur
   passé.

## Ce qui existe et se réutilise (vérifié sur pièces le 2026-09-17)

| Besoin | Existant | Fichier |
|---|---|---|
| Assistances échangées avec chaque coéquipier (map par xuid) | `port.RelationsRepository.GetRelationAssists(ctx, scope)` → `CareerRepo` | `port/repository.go:200`, `platform/duckdb/relation_assists_repo.go:152` |
| Agrégat + doctrine (tranches, couverture, « — » si non mesuré) | `domain.RelationAssists`, `AssistTiers` | `domain/relation_assists.go` |
| Provider relationnel déjà injecté dans l'Explorer | `ExplorerRelationsProvider` (satisfait par `CareerRepo`) | `service/explorer_service.go:146`, wire `api/wire/registry_pages_explorer.go:87` |
| Point d'enrichissement best-effort de la rangée « ensemble » | `enrichEncounterRelations` | `service/explorer_service.go:536` |
| Barre papillon + tons par tranche + infobulles | `AssistButterflyBar` (variants `card`/`row`) | `features/_shared/assists/AssistButterflyBar.tsx` |
| Parts, segments, borne d'échelle log | `receivedShare`, `givenShare`, `assistVolumeMax`, `assistSegments` | `features/_shared/assists/assistExchange.ts` |
| Libellés FR/EN des assistances (partagés palmares + match-view) | `ASSISTS_TEXT` | `features/_shared/assists/assistsI18n.ts` |
| Figure « têtes + papillon + total · part » (carte Binôme) | `BinomeAssistsBlock` | `features/palmares/RelationAssistsCards.tsx:36` |
| Mécanique d'harmonisation des hauteurs d'une rangée | `h-full` + enfants `flex-1` | `features/explorer/ExplorerTargetCadence.tsx:72` |
| Bloc « Répartition des résultats » à déplacer | `ExplorerTargetOutcome` | `features/explorer/ExplorerTargetSampleStats.tsx:244` |

## Décisions tranchées AVANT exécution (fermes, ne pas re-décider)

**D1 — Largeurs.** Rangée 3 = `grid lg:grid-cols-3` ; colonne gauche `lg:col-span-2`
(résultats + assistances empilés), colonne droite `lg:col-span-1` (portée). Amendement
utilisateur du 2026-09-17 (« le bloc de droite ne prendrait qu'un tiers de largeur, on
verra pour ajuster ensuite au besoin »).

**D2 — Hauteurs.** La colonne gauche impose la hauteur de la rangée ; le bloc de droite
est en `h-full` et étire son contenu. Même mécanique que la Cadence (rangée 1), pas de
hauteur en dur.

**D3 — Source des assistances.** `GetRelationAssists(ctx, nil)` (map de TOUS les
coéquipiers) appelé UNE fois dans `enrichEncounterRelations` : on y lit l'entrée de la
cible ET on en déduit la borne d'échelle. Aucun nouveau SQL, aucune nouvelle méthode de
repo, aucun risque de divergence avec la page Relations.

**D4 — Borne de l'échelle log (`volumeMax`).** Servie par le backend :
`AssistVolumeMax` = plus gros volume d'un sens parmi TOUTES les relations mesurées du
joueur — la même borne que la page Relations. Se borner à la paire seule remplirait
toujours la demi-barre : c'est l'échec explicitement documenté dans
`assistExchange.ts:9-17`, ne pas le reproduire.

**D5 — Absence de donnée = « — », jamais « 0 assistance ».** Doctrine
`domain/relation_assists.go` : pas d'objet du tout si zéro match mesuré. Le bloc reste
rendu (titre + emplacement) avec un « — », comme les autres blocs de l'encart.

**D6 — Factorisation de la figure d'échange.** La figure « têtes + papillon + total ·
part » passe de 1 à 2 copies (Binôme + Explorer) : elle est extraite dans
`features/_shared/assists/AssistExchangeSummary.tsx` et `BinomeAssistsBlock` la consomme.
Pas de 2e copie littérale (règle CLAUDE.md n°6 / anti-pattern « factorisation
abandonnée »). Le rendu de Relations doit rester octet pour octet identique.

**D7 — Portée = placeholder honnête.** Bloc titré « Portée des frags » avec un état
d'attente explicite, aucune donnée fabriquée, aucun flag. Il sera remplacé par la
livraison de l'autre agent (chantier `wt/compare-armes`,
`.ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md`, qui construit déjà
`domain.SynthesisWeaponRange` via `port.WeaponRangeRepository`).

**D8 — i18n.** Toute string neuve en FR **et** EN dans
`apps/web/src/lib/i18n/manifests/explorer.toml` puis `node scripts/build_i18n_manifests.mjs`
(le fichier `generated/explorer.ts` ne s'édite jamais à la main). FR sans anglicismes.

**D9 — Contrat OpenAPI.** `openapi.yaml` est GÉNÉRÉ : après modification du DTO Go,
`make openapi-gen` puis `make generate-types`. Ne jamais éditer `openapi.yaml` ni
`generated.ts` à la main.

**D10 — Aucun branchement par slug.** Un titre sans décodeur de film ne produit aucune
ligne mesurée : la map revient vide, le bloc affiche « — ». Pas de test de capability
ad hoc, pas de `slug ==` (ratchet ADR 0025).

## Étapes

### Étape 0 — Vérifications sur pièces

- [x] 0.1 Rouvrir `service/explorer_service.go:536` (`enrichEncounterRelations`) et
      `explorer_service.go:146` (`ExplorerRelationsProvider`) — confirmer les signatures.
- [x] 0.2 Rouvrir `domain/explorer.go:49` (`ExplorerEncounterStats`) — confirmer la forme.
- [x] 0.3 Lister les implémentations de `ExplorerRelationsProvider` (prod + tests) à
      étendre : `duckdb.CareerRepo`, `mockExplorerRelations`
      (`service/explorer_frag_gap_test.go:47`).
- [x] 0.4 Relire `features/palmares/RelationAssistsCards.tsx:36-66` (figure à extraire) et
      `ExplorerTargetProfileCard.tsx:159-184` (disposition à refondre).

**Gate 0** : aucune commande ; les 4 lectures faites, écarts éventuels notés en
« Découvertes ».

### Étape 1 — Backend : assistances de la paire + borne d'échelle

- [x] 1.1 `domain/explorer.go` : ajouter à `ExplorerEncounterStats` les champs
      `Assists *RelationAssists` (`json:"assists,omitempty"`) et `AssistVolumeMax int`
      (`json:"assist_volume_max,omitempty"`), documentés (doctrine + rôle de la borne).
- [x] 1.2 `service/explorer_service.go` : ajouter `GetRelationAssists` à
      `ExplorerRelationsProvider`.
- [x] 1.3 `enrichEncounterRelations` : 3e source best-effort — un appel, entrée de la
      cible + `assistVolumeMax` sur toute la map. Erreur → `slog.WarnContext` puis
      dégradation (jamais d'erreur avalée en silence).
- [x] 1.4 Helper pur `explorerAssistVolumeMax(map[string]domain.RelationAssists) int`
      (miroir Go de `assistVolumeMax` côté web), testable sans DB.
- [x] 1.5 Étendre `mockExplorerRelations` (`explorer_frag_gap_test.go`) + test de
      `enrichEncounterRelations` : cible présente dans la map → `Assists` rempli et borne
      = max global ; cible absente → `Assists` nil ; erreur repo → champs nil, pas de
      panique.
- [x] 1.6 `make openapi-gen` puis `make generate-types`.

**Gate 1** :
```
cd apps/go-api && go build ./... && go test ./internal/service/... ./internal/domain/...
make openapi-check
make check-types
```

### Étape 2 — Web : figure d'échange partagée + carte « Part des assistances »

- [x] 2.1 Créer `features/_shared/assists/AssistExchangeSummary.tsx` : têtes
      (reçues/données), `AssistButterflyBar` variant `card`, ligne « total · part ».
      Tokens `assist-received` / `assist-given` uniquement (skill `color-tokens`).
- [x] 2.2 `BinomeAssistsBlock` consomme le composant partagé ; rendu inchangé (le test
      `PalmaresRelationsPage.test.tsx` doit passer sans modification).
- [x] 2.3 Créer `features/explorer/ExplorerTargetAssists.tsx` : carte titrée « Part des
      assistances » (même habillage `rounded-lg border border-border bg-card` + en-tête
      que `ExplorerTargetOutcome`), `h-full`, figure partagée, « — » si `assists` absent.
- [x] 2.4 Clés i18n dans `explorer.toml` (titre du bloc, état « — ») FR + EN, puis
      régénération du manifeste.
- [x] 2.5 Test `ExplorerTargetAssists.test.tsx` : avec `assists` → papillon + parts
      rendus ; sans `assists` → « — » et pas de papillon.

**Gate 2** : `make check-types` puis `npx vitest run src/features/explorer src/features/palmares`
(hors sandbox, cf. mémoire `reference_vitest_outside_sandbox`).

### Étape 3 — Web : placeholder « Portée des frags »

- [x] 3.1 Créer `features/explorer/ExplorerTargetFragRangePlaceholder.tsx` : carte titrée,
      `h-full`, message d'attente centré (style bordure pointillée déjà utilisé dans
      `ExplorerTargetProfileCard`), `data-testid` dédié.
- [x] 3.2 Clés i18n FR + EN (titre + message) dans `explorer.toml` + régénération.

**Gate 3** : `make check-types`.

### Étape 4 — Web : la disposition en trois rangées

- [x] 4.1 `ExplorerTargetProfileCard.tsx` : retirer `ExplorerTargetOutcome` de la rangée 1
      et donner la hauteur pleine à `ExplorerTargetSampleStats` (colonne 2/3).
- [x] 4.2 `ExplorerTargetSampleStats` : le bloc « Répartition des frags » s'étire
      (`h-full` / `flex-1`) et le sunburst occupe la place libérée (`maxWidthPx` relevé).
- [x] 4.3 Ajouter la rangée 3 sous `ExplorerTargetVersusDonuts` : `grid lg:grid-cols-3`,
      colonne gauche `lg:col-span-2` (`ExplorerTargetOutcome` + `ExplorerTargetAssists`
      empilés, `flex flex-col gap-4`), colonne droite `lg:col-span-1` (placeholder,
      `h-full`).
- [x] 4.4 `ExplorerTargetOutcome` : `h-full` pour remplir sa part de colonne.
- [x] 4.5 Compléter `ExplorerTargetProfileCard.test.tsx` : ordre des trois rangées, et
      « Répartition des résultats » n'est plus dans la rangée des frags.
- [x] 4.6 Mettre à jour les en-têtes de fichier (`ExplorerTargetProfileCard.tsx`,
      `ExplorerTargetSampleStats.tsx`, `ExplorerTargetVersusDonuts.tsx`) : la doc décrit
      la disposition RÉELLE (anti-pattern « doc inversée »).

**Gate 4** : `make check-types` + `npx vitest run src/features/explorer`.

### Étape 5 — Livraison

- [x] 5.1 Skill `delivery-checklist`.
- [~] 5.2 Gates : `go vet ./...` module entier EXIT=0, `make go-api-test` EXIT=0 (36 paquets), golangci-lint `--new-from-merge-base=origin/main` 0 issue, suite web complète 731/7845 verte, `tsc -b` + `eslint` + `lint:colors` + `lint:fields` OK, `openapi-check` à jour. Le `go test ./...` global est resté 27 min sans une ligne de sortie ni un binaire de test vivant (il suivait un run interrompu : verrou de cache de build probable, cause NON établie sur pièces) ; tué, `make go-api-test` a ensuite tourné normalement. Couverture : paquets réellement touchés (`service`, `domain` : 0 échec) + `go vet` + CI de branche (gate d autorité).
      `make check-types`, `make openapi-check`.
- [x] 5.3 Entrée `.ai/thought_log.md`.
- [x] 5.4 Commits sur `wt/explorer-rangee3` (1 par étape minimum), plan à jour dans le
      commit.
- [x] 5.5 Point d'étape + demande de gate visuel à l'utilisateur (écran Explorer >
      recherche joueur sur une cible avec films analysés ET une cible sans).

## Découvertes (hors périmètre — noter, ne pas traiter)

| Date | Découverte | Suite |
|---|---|---|
| 2026-09-17 | `service/explorer_service.go` (573 L) et `domain/explorer.go` (599 L) dépassaient DÉJÀ le seuil de 500 L ; ce lot y ajoute 8 et 11 lignes (méthode de port + 2 champs de DTO). | Non traité ici (rule 7 : zéro fix hors périmètre). Un découpage de ces deux fichiers touche de nombreux appelants — chantier à part. |

## Journal

| Date | Étape | État | Note |
|---|---|---|---|
| 2026-09-17 | — | Plan écrit | Worktree `../LevelUp-wt-explorer-rangee3` créé depuis `feat/v75` (05723dce2). |
| 2026-09-17 | 1 | Close | Gate vert : `go build ./...`, `go test ./internal/service/... ./internal/domain/...` (0 échec), `openapi-gen -check` à jour, `tsc --noEmit` OK. `npm ci` a dû être lancé dans le worktree (node_modules absent d une copie de travail neuve). |
| 2026-09-17 | 2 | Close | Figure extraite dans `_shared/assists/AssistExchangeSummary` (Binôme la consomme, test Relations INCHANGÉ et vert) ; carte `ExplorerTargetAssists` + 4 tests. Gate : vitest explorer+palmares 19/19, `tsc --noEmit` OK. Aucune clé i18n « état vide » créée : `ASSISTS_TEXT.notMeasured` réutilisé. |
| 2026-09-17 | 3 | Close | Placeholder `ExplorerTargetFragRange` (titre + « En préparation », aucune donnée fabriquée). Écart au plan assumé : fichier nommé `ExplorerTargetFragRange.tsx` et non `...Placeholder.tsx` — il gardera son nom quand la mesure arrivera, pas de renommage à prévoir. Gate : `tsc --noEmit` OK. |
| 2026-09-17 | 4 | Close | Trois rangées en place ; `data-testid="explorer-target-outcome"` ajouté (ancre de l ordre des rangées) ; anneau des frags porté de 480 à 560 px pour occuper la place libérée ; en-têtes de fichiers remis en phase avec la disposition réelle. Gate : vitest `src/features/explorer` 27 fichiers / 229 tests verts, `tsc --noEmit` OK. |
| 2026-09-17 | 5 | Close | Gates : `make go-api-test` EXIT=0 (36 paquets, 0 échec), `go vet ./...` EXIT=0, golangci-lint `--new-from-merge-base=origin/main` sur service+domain 0 issue, suite web complète 731/7845 verte sur l arbre final, `tsc -b` cache purgé, eslint 0 erreur, lint:colors et lint:fields 0 violation, openapi-check à jour. Le `go test ./...` global a dû être abandonné (voir 5.2). 4 commits sur `wt/explorer-rangee3`. |
