# Plan — Records de distance de frag par arme sur la Synthèse (rendu A, « la règle »)

Date : 2026-09-20. Branche `wt/records-distance` (worktree `LevelUp-wt-records-distance`,
base `feat/v75`). Maquette validée : `.ai/V7.5/MAQUETTE_REGLE_RECORDS_2026-09-20.html`
(rendu A) et le document Claude « Synthèse — records de distance de frag par arme : cinq
rendus ». Contrat d'exécution : skill `plan-execution`.

## Décisions fermes (utilisateur, 2026-09-20)

- Rendu A : une ligne graduée, un losange par arme à son record, couleur par classe d'arme,
  libellés étagés au-dessus. Légende EN DESSOUS, CENTRÉE. Même chrome de bloc que les autres
  blocs de la page (carte à titre, corps, pied de légende).
- Grain PAR ARME. TOUS les frags mesurés comptent, aucun seuil d'effectif.
- Halo 5 non couvert : la section s'omet (repo → `ErrCapabilityNotSupported`), le web gate
  sur `useCapability('weapon_range')`.
- Décisions techniques prises ici (conventions du dépôt, pas des choix produit) :
  - côté tueur seulement (« mes frags ») ;
  - classes écartées ET nommées en pied : `melee`, `environmental`, `equipment` (une
    distance n'y a pas de sens : coup de crosse, chute, répulseur, bobine) ; une clé sans
    dimension connue est GARDÉE avec une classe vide (couleur neutre) ;
  - le record ouvre le REJEU à l'instant du frag : `?t=<time_ms>&clock=match`
    (`match_kill_events.time_ms` est l'horloge du match, cf. `domain.TacticalClockMatch`) ;
  - carte et date du match source lues dans le scope canonique déjà chargé (aucune
    requête neuve) ;
  - ordre imposé côté Go (record croissant), jamais rejoué côté web.

## Étapes

### Étape 1 — Go, calcul pur
- [x] 1.1 `internal/analysis/weapon_distance_records.go` : `WeaponDistanceRecord` +
      `WeaponDistanceRecords(kills, side)` (record = max, médiane via `percentileLinear`,
      effectif ; tri record croissant ; départage déterministe).
- [x] 1.2 Tests `weapon_distance_records_test.go` (vide, un frag, plusieurs armes, tri,
      côté ignoré, départage, entrée non mutée).
- Gate : `go test ./internal/analysis/`.

### Étape 2 — Go, section de Synthèse
- [x] 2.1 `domain/synthesis_weapon_records.go` : `SynthesisWeaponRecords`,
      `WeaponDistanceRecordRow`, `WeaponRecordFrag`, `WeaponExcludedFromRecords`.
- [x] 2.2 Champ `WeaponRecords *SynthesisWeaponRecords json:"weapon_records,omitempty"`
      sur `SynthesisPageV2Response`.
- [x] 2.3 `service/synthesis_weapon_records.go` : `buildWeaponRecordsSection` (fonction
      libre, best-effort, régime de journalisation identique à la portée), exclusion par
      classe via `ResolveWeaponDimensions`, libellés via `ResolveWeaponLabels`, carte/date
      depuis les lignes canoniques.
- [x] 2.4 `SynthesisService.WithWeaponRangeRepo` + appel dans `GetSynthesisPage` ;
      câblage INCONDITIONNEL dans `SynthesisCtx` (registry_pages_home.go).
- [x] 2.5 Tests service (`synthesis_weapon_records_test.go`, mock existant) : nominal,
      exclusions nommées, dimensions muettes, libellés en échec, dégradations sans section.
- [x] 2.6 `make openapi-gen` puis golden test ; `make generate-types`.
- Gate : `go test ./internal/analysis/ ./internal/service/ ./internal/domain/... ./internal/api/... ./internal/api/wire/`
  + `go vet ./...`.

### Étape 3 — Web
- [x] 3.1 `lib/api/types.ts` : `weapon_records?` + re-exports des schémas.
- [x] 3.2 i18n : clés `synthesis.weapon_records.*` FR + EN dans `synthesis.toml`, manifestes
      régénérés.
- [x] 3.3 `features/synthesis/weaponRecords_logic.ts` : projection + étagement glouton des
      libellés (pur) + tests.
- [x] 3.4 `features/synthesis/WeaponRecordsRuler.tsx` : SVG dans une `SectionCard` (titre
      + compte), infobulle au survol, `<Link>` vers le rejeu, `ChartLegend` centrée en pied,
      note « Écartés » nommée.
- [x] 3.5 Montage dans `SynthesisPage.tsx` sous la rangée frags, gate
      `useCapability('weapon_range')` + donnée présente.
- [x] 3.6 Tests composant (libellés, légende, écartés, lien avec `t`/`clock`).
- Gate : purge `node_modules/.tmp`, `npm run typecheck`, `npm run lint`,
  `npm run lint:colors`, `npx vitest run src/features/synthesis`.

### Étape 4 — Vérification visuelle et clôture
- [x] 4.1 Serveur de dev sur le worktree, capture de la Synthèse (clair + sombre),
      survol, clic vers le rejeu.
- [x] 4.2 Maquette HTML copiée dans ce worktree (`.ai/V7.5/`), entrée thought_log,
      plan statué.
- [x] 4.3 Commit `6a49543ce` sur `wt/records-distance` (accord utilisateur du
      2026-09-20). Décision utilisateur du même jour : épée et marteau RESTENT sur la
      règle. Fusion dans `feat/v75` : à la main de l'utilisateur.

## Découvertes (hors périmètre, non traitées)
- `WEAPON_KEYS_WITHOUT_RANGE` côté web (`weaponRangeChart.ts`) reste un pis-aller : la
  section des records fait l'exclusion côté Go par classe ; la portée pourrait suivre.
- Épée à énergie et marteau antigravité sont de classe `heavy` (décision utilisateur du
  2026-09-01, `registry.go:455-465`) : l'exclusion par classe les GARDE sur la règle (1,3 m
  et 4,5 m sur données réelles). Les écarter demanderait un critère de registre autre que
  la classe (famille ? type de dégât ?) — décision utilisateur, pas prise ici.
- Le routeur sérialise `t` en JSON dans l'URL (`?t=%22482624%22`), comme pour la carte
  tactique : la route le relit correctement, le rejeu s'ouvre au bon instant. Cosmétique.

## Journal
- 2026-09-20 : plan écrit, étape 1 ouverte.
- 2026-09-20 : étapes 1 à 3 closes (gates Go et web verts, cf. thought_log). Étape 4 :
  vérification sur données réelles (API du worktree sur la base locale, 30 armes) —
  écart axe/libellés relevé de 14 à 24 px après capture ; clic → rejeu du bon match à
  l'instant du frag ; thèmes clair et sombre. Commit en attente de l'utilisateur.
