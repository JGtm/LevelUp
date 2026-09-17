# PLAN — Explorer : le bloc « Portée des frags » remplace son placeholder (2026-09-17)

> Date : 2026-09-17. Suite de `.ai/PLAN_EXPLORER_RANGEE3_2026-09-17.md` (rangée 3 livrée,
> placeholder posé). Débloqué par la fusion de `wt/compare-armes` dans `feat/v75`
> (`5b2298f06`). Branche d'exécution : `wt/explorer-portee-frags`, worktree dédié
> `../LevelUp-wt-explorer-rangee3` (le worktree principal est PARTAGÉ, ne jamais y coder).
> Clôture : commits sur la branche, fusion dans `feat/v75` sur signal de l'utilisateur.
> Contrat : skill `plan-execution`. Ce fichier est la source de vérité de l'avancement.

## Objectif et critère de succès

Remplacer le placeholder « Portée des frags » de la 3e rangée de l'encart cible Explorer par
la MÊME figure que le bloc « Où ils fraguent » du Face-à-face : une ligne par rôle d'arme,
bâton du 10e au 90e centile, losange sur la médiane, deux joueurs superposés.

**Critère de succès** :
1. Le bloc rend le graphe pour le joueur courant (bande du haut) et la cible (bande du bas),
   sur les matchs joués ensemble, côté frags UNIQUEMENT.
2. Rien d'autre dans le bloc : pas de note de couverture, pas de liste des rôles sous le
   seuil (demande utilisateur du 2026-09-17 : « n'affiche que les bandes des joueurs »).
3. La colonne droite de la rangée 3 passe à **45 %** de largeur (55/45).
4. Aucune dérogation `explorer=>synthesis` n'est créée ; celle de `compare` disparaît.
5. Dégradation propre : titre sans positions par kill, aucun film décodé sur les matchs
   communs, cible sans mesure → bloc en état vide, jamais de donnée fabriquée.
6. Gates Go et web verts, CI de branche verte au niveau job, gate visuel utilisateur.

## Ce qui existe et se réutilise (vérifié sur pièces le 2026-09-17, après la fusion)

| Besoin | Existant | Fichier |
|---|---|---|
| Frags mesurés (arme, côté, distance) bornés par matchs + xuid | `port.WeaponRangeRepository.LoadWeaponRange` | `internal/port/weapon_range.go:117` |
| Clé d'arme → dimensions (classe/rôle/famille) | `ResolveWeaponDimensions` | `internal/port/weapon_range.go:141` |
| Regroupement des frags mesurés par rôle | `analysis.RegroupMeasuredKills` | `internal/analysis/weapon_range.go:274` |
| Agrégat P10/médiane/P90 + seuil + bloc de réponse | `buildWeaponRangeBlock`, `weaponRangeScopeInfo` | `internal/service/weapon_range_section_build.go:29` |
| Chaîne complète « portée par rôle d'un joueur sur un scope » | `CompareService.compareWeaponRange` + `resolveWeaponRoles` | `internal/service/compare_weapons.go:258` |
| Grammaire graphique (bâton P10-P90, losange médiane, `top`/`bottom`) | `buildWeaponRangeOption`, `weaponRangeChartHeight`, `WeaponRangeLine` | `features/synthesis/_weaponRangeChart.ts` |
| Libellé d'une clé d'arme | `resolveWeaponLabel` | `features/synthesis/weaponRange_logic.ts:39` |
| Lignes par rôle depuis deux côtés + axe commun | `roleRangeLines`, `roleLabel` | `features/compare/compareWeapons_logic.ts` |
| Le bloc à imiter | `CompareWeaponsRange` | `features/compare/CompareWeaponsRange.tsx` |
| Le placeholder à remplacer | `ExplorerTargetFragRange` | `features/explorer/ExplorerTargetFragRange.tsx` |

## Décisions tranchées AVANT exécution (fermes)

**D1 — Ce que montre le bloc.** Côté FRAGS seulement, deux bandes : le joueur courant en
haut, la cible en bas, sur les matchs joués ensemble (le scope de la section). Le côté morts
(« Où ils meurent ») n'est PAS repris : la rangée a une seule colonne pour ce bloc.

**D2 — Rien que les bandes.** Demande utilisateur : le graphe et sa légende, sans la note de
couverture « N frags mesurés sur M » ni la liste des rôles écartés par le seuil que le
Face-à-face affiche sous le sien.

**D3 — Largeur 45 %.** La rangée 3 quitte `lg:grid-cols-3` pour un ratio explicite 55/45.

**D4 — Le déplacement d'abord (validé par l'utilisateur le 2026-09-17).**
`features/synthesis/_weaponRangeChart.ts` et `resolveWeaponLabel` partent dans
`components/charts/` AVANT d'être consommés par l'Explorer. Motif : le module ne dessine rien
de propre à la Synthèse — laquelle ne l'affiche même plus depuis le 2026-09-13 — et un
troisième consommateur demanderait une troisième dérogation à l'anti-import inter-features.
Les deux modules n'importent que `@/lib/**` et `@/components/charts/_utils` : la frontière
inversée du lint est respectée sans retouche.
Effet : la dérogation `compare=>synthesis/_weaponRangeChart` disparaît, celle d'`explorer`
n'est jamais créée. `timeseries=>synthesis` RESTE (elle porte le montage de
`SynthesisWeaponRangeSection`, qui ne bouge pas).

**D5 — Deuxième copie côté Go = on factorise (règle CLAUDE.md n°6).** La chaîne « frags
mesurés → rôles → regroupement → bloc » existe une fois (`compareWeaponRange`). L'Explorer
serait la deuxième : elle est extraite dans un helper partagé et les DEUX appelants
l'utilisent, avec un garde-rail qui interdit la réécriture de la chaîne ailleurs.

**D6 — Idem côté web pour `roleRangeLines`.** Généralisé pour prendre deux
`SynthesisWeaponRange` (et non deux `CompareWeaponSide`) et déplacé auprès du module de
rendu ; sinon l'Explorer devrait déroger vers `features/compare`.

**D7 — Dégradation.** Le repo dit lui-même l'absence de capability
(`games.ErrCapabilityNotSupported`) : bloc absent → état vide titré, jamais de donnée
fabriquée, jamais de branchement par slug (ADR 0025).

**D8 — Aucun libellé FR/EN côté Go.** Les rôles voyagent en CLÉS, le front résout via son
manifeste (`frags.role.*` puis `frags.class.*`), comme le Face-à-face.

**D9 — La section de la Synthèse ne bouge pas.** L'utilisateur a tranché le 2026-09-17 : le
graphe reste sur l'onglet Résumé. `SynthesisWeaponRangeSection` garde son nom et son dossier ;
seul le module de DESSIN est déplacé. L'anomalie de nommage restante est notée, pas traitée.

## Étapes

### Étape 0 — Vérifications sur pièces

- [x] 0.1 Relire `compare_weapons.go:258-325` (chaîne portée par rôle) et
      `weapon_range_section_build.go:29` (`buildWeaponRangeBlock`, `weaponRangeScopeInfo`).
- [x] 0.2 Relire `CompareWeaponsRange.tsx` et `compareWeapons_logic.ts` (`roleRangeLines`,
      `roleLabel`, axe commun) dans leur version FUSIONNÉE.
- [x] 0.3 Relever les consommateurs de `_weaponRangeChart` et de `weaponRange_logic`.
- [x] 0.4 Repérer où l'Explorer tient les match_id des matchs communs.

### Étape 1 — Le déplacement du module de rendu

- [x] 1.1 `features/synthesis/_weaponRangeChart.ts` → `components/charts/weaponRangeChart.ts`.
- [x] 1.2 `resolveWeaponLabel` part avec lui ; `WEAPON_RANGE_MIN_MEASURED` et
      `hasWeaponRangeRows` restent dans `features/synthesis/weaponRange_logic.ts`.
- [x] 1.3 Mettre à jour les imports : synthesis (section, table, `_weaponElevationChart`),
      compare (`CompareWeaponsRange`), et les tests des deux.
- [x] 1.4 Retirer `compare=>synthesis/_weaponRangeChart` de
      `tools/lint-cross-feature-imports.mjs`.
- [x] 1.5 En-têtes de fichiers remis en phase (le module ne se réclame plus de la Synthèse).

**Gate 1** : `node tools/lint-cross-feature-imports.mjs`, `npm run typecheck`,
`npx vitest run src/features/synthesis src/features/compare`.

### Étape 2 — Backend : la portée par rôle devient partagée, et l'Explorer la sert

- [x] 2.1 Extraire la chaîne « mesures → rôles → regroupement → bloc » dans un helper de
      paquet `service` réutilisable (entrées : slug, match_id, xuid, totaux du scope).
- [x] 2.2 `compareWeaponRange` appelle le helper (comportement inchangé, tests existants
      verts sans retouche).
- [x] 2.3 Garde-rail : test qui interdit une seconde réécriture de la chaîne hors du helper.
- [x] 2.4 DTO : la portée par rôle des DEUX joueurs sur les matchs communs rejoint la
      réponse de l'encart cible.
- [x] 2.5 Service Explorer : remplir ce bloc, best-effort strict (log puis dégradation).
- [x] 2.6 Tests : nominal, cible sans mesure, capability absente, repo en erreur.
- [x] 2.7 `make openapi-gen` puis `make generate-types`.

**Gate 2** : `go build ./...`, `go test ./internal/service/... ./internal/domain/...
./internal/analysis/...`, `openapi-gen -check`, `npm run typecheck`.

### Étape 3 — Web : le bloc remplace le placeholder

- [x] 3.1 Généraliser `roleRangeLines` (deux `SynthesisWeaponRange` + axe) et le déplacer
      auprès du module de rendu ; le Face-à-face consomme la version déplacée.
- [x] 3.2 `ExplorerTargetFragRange` rend le graphe : bandes joueur courant / cible, côté
      frags, légende des deux noms, rien d'autre (D2).
- [x] 3.3 État vide titré quand aucune mesure (D7).
- [x] 3.4 Rangée 3 en 55/45 (D3).
- [x] 3.5 i18n : la clé d'attente du placeholder disparaît, les libellés neufs sont FR + EN.
- [x] 3.6 Tests : rendu nominal (deux bandes), état vide, et la rangée garde sa disposition.

**Gate 3** : `npm run typecheck`, `npx vitest run src/features/explorer`,
`node tools/lint-cross-feature-imports.mjs`.

### Étape 4 — Livraison

- [x] 4.1 Skill `delivery-checklist`.
- [ ] 4.2 Gates complets Go + web + lints + openapi-check.
- [x] 4.3 Entrée `.ai/thought_log.md`.
- [ ] 4.4 Commits (1 par étape minimum), plan à jour dans le commit.
- [ ] 4.5 Point d'étape + demande de gate visuel.

## Découvertes (hors périmètre — noter, ne pas traiter)

| Date | Découverte | Suite |
|---|---|---|
| 2026-09-17 | `SynthesisWeaponRangeSection` porte le nom et le dossier d'une page qui ne l'affiche plus (montée par `features/timeseries`). | Non traité : D9, l'utilisateur garde le graphe sur l'onglet Résumé. Renommage/déplacement de la section = lot à part. |

## Journal

| Date | Étape | État | Note |
|---|---|---|---|
| 2026-09-17 | — | Plan écrit | Branche `wt/explorer-portee-frags` depuis `feat/v75` (`5b2298f06`, compare-armes fusionné). |
| 2026-09-17 | 1 | Close | `_weaponRangeChart.ts` → `components/charts/weaponRangeChart.ts` (+ son test), `resolveWeaponLabel` et ses tests partis avec lui. Dérogation `compare=>synthesis/_weaponRangeChart` retirée. Gate : lint inter-features 7/7 (un cran libéré), `tsc` OK, vitest synthesis+compare+charts 14 fichiers / 148 tests. |
| 2026-09-17 | 2 | Close | Chaîne « mesures → rôles → regroupement → bloc » extraite dans `service/weapon_range_by_role.go` ; `compareWeaponRange` n'en est plus qu'un appel (tests compare inchangés, verts). Garde-rail : `analysis.RegroupMeasuredKills` interdit hors du helper. DTO `frag_range_self`/`frag_range_target`, repo câblé sans condition, 4 tests de dégradation. Gate : `go test service+domain+analysis` 28 paquets / 0 échec, `openapi-gen -check` à jour. |
| 2026-09-17 | 3 | Close | `roleAxis`/`roleRangeLines`/`roleLabel` généralisés (entrée = bloc de portée, plus un côté de page) et déplacés dans `components/charts/weaponRangeRoles.ts` avec leurs tests. `ExplorerTargetFragRange` rend le graphe (2 bandes, côté frags, légende) ; rangée 3 en 55/45. Gate : vitest `src/features/explorer` 28 fichiers / 234 tests, `tsc` OK. |
