# PLAN — Lisibilité des cartes d'usage de la page Sessions (2026-09-09)

Chantier issu d'une revue utilisateur des blocs « Usages d'équipement » / « Contrôle des
armes spéciales » / « Parts et parités » (page Sessions). Maquettes avant/après validées
par l'utilisateur le 2026-09-09.

Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report d'action
exécutable, statut sur chaque item, gate avant clôture).

## Décisions validées (fermes — ne pas re-débattre)

- **D1** — Libellés de grandeur au NOM NU : « Camouflage », « Surbouclier »,
  « Mur de protection », « Capteur de menaces », « Grappin », « Objets lâchés ». La clé
  machine `deployed_<famille>` ne doit plus atteindre l'écran.
- **D2** — Le compte brut quitte la cellule de jauge. « 49,3 % (105 sur 213) » → « 49,3 % » ;
  le dénominateur reste dans l'infobulle (déjà présent, `gaugeTipFmt`).
- **D3** — Les barres d'ÉTENDUE (min/max de la part joueur/équipe) sont SUPPRIMÉES du rendu :
  la bande « Régularité match par match » de la même carte porte déjà cette information.
- **D4** — Une seule colonne de jauges visible par défaut (« Ma part dans mon équipe » =
  `player-of-team`) ; les deux autres derrière `CollapsedItemsToggle`. Un seul axe gradué
  par colonne rendue.
- **D5** — Les textes de méthode (corollaire du bloc, légende de parité, légende de la
  bande) quittent le corps et le pied pour une infobulle d'en-tête de carte.
- **D6** — Le vocabulaire « socle » disparaît de l'écran. La grandeur `pad_pickups` n'est
  plus une ligne de jauge PAIRE de ses propres composantes : elle devient un TOTAL mis à
  part sous les familles d'armes.
- **D7** — Les familles d'armes s'affichent avec leur NOM (« Sniper »), plus leur clé
  hexadécimale. Résolution côté Go via `replaylabels.Load` (patron
  `service/replay_weapon_labels.go`). Une famille sans nom au catalogue GARDE sa clé —
  règle du chantier rejeu : « un nom approchant se lit comme une certitude ».
- **D8** — Le bloc usage est servi ET rendu pour la session COMPARÉE ; drawer ouvert =
  version compacte des DEUX côtés.

## Découvertes

- **2026-09-09 (étape 1)** — `player_share_of_team_min_pct` / `_max_pct` du contrat
  (`SessionUsageMetric`) n'ont PLUS AUCUN LECTEUR depuis le retrait de l'étendue (D3).
  Champs calculés côté Go, publiés, jamais lus. Traité en étape 2 (item 2.5), qui touche
  déjà le contrat et l'openapi — pas différé, ordonnancé.

---

## Étape 1 — Front : allègement des deux cartes (D1..D6)

Périmètre : `apps/web/src/features/session-detail/` uniquement, aucun changement de contrat.

- [x] 1.1 `usageI18n.ts` — libellés courts (D1), `deployedFamilyLabel`
      (sensor/rift/shroud/seeker/field), `cardHint*` d'en-tête (D5), vocabulaire sans
      « socle » (D6). Parité FR/EN par typage.
- [x] 1.2 `usageLogic.ts` — `metricLabel` branché sur `deployedFamilyLabel` ; champs
      d'étendue RETIRÉS de `UsageGaugeModel` / `UsageGaugeRowInput` (D3) ; `isTotal`
      ajouté au modèle de ligne (D6).
- [x] 1.3 `SessionUsageForms.tsx` — `UsageGauge` sans texte d'honnêteté ni étendue
      (D2/D3) ; `UsageGaugeGrid` à une colonne + `CollapsedItemsToggle` (D4), un axe par
      colonne rendue, plus de légende de parité ; filet pleine largeur avant un total.
- [x] 1.4 `SessionUsageSection.tsx` — `cardTitleAdornment` porte l'aide sur les trois
      cartes (D5) ; pied équipement et pied objectifs supprimés ; pied armes spéciales
      réduit aux comptes ; familles avant le total `pad_pickups` marqué `isTotal` (D6).
- [x] 1.5 Tests : `usageLogic.test.ts` mis à jour (+ contrat d'ordre des jauges) et
      `SessionUsageForms.test.tsx` créé (repli D4, compte brut en infobulle D2).

Gate : `npx tsc -b --force` exit 0 · `npx eslint src/features/session-detail
--max-warnings=0` exit 0 · `npx vitest run` 663 fichiers / 7061 tests verts (le nouveau
`SessionUsageForms.test.tsx` a été lancé à part après ce run : 3/3 verts, typecheck
rejoué exit 0 après ajout).

## Étape 2 — Noms d'armes des familles de socle (D7)

- [x] 2.1 Go — `SessionUsagePadFamily.FamilyLabel` (optionnel) au contrat ; résolution
      hex → `replay.LabelCatalog.Weapons` localisée dans un fichier dédié
      `service/session_page_usage_labels.go` (et non dans `session_page_usage.go`, déjà
      long) ; appelée en fin de `attachSessionUsage`, `locale` propagée depuis `req`.
- [x] 2.2 Go — `repoRoot` porté par `WithSessionUsage` (et non un `With*` de plus) : il
      n'a d'utilité que pour ce bloc. Wiring `registry_pages.go` → `r.cfg.RepoRoot`.
- [x] 2.3 `make openapi-gen` + `make generate-types` ; front : `fam.family_label ||
      t.padFamilyFmt(fam.family_key)` — le repli reste, il ne sert plus qu'aux familles
      hors catalogue.
- [x] 2.4 `session_page_usage_labels_test.go` : famille connue nommée FR **et** EN, clé
      jamais remplacée ; famille hexa hors registre et clé non-hexa (`powerup_camo`) →
      aucun nom ; montage sans `repoRoot` → aucun nom, aucune erreur ; table de
      `parseWeaponFamilyKey`. Le test lit le VRAI catalogue du titre (pas de fixture qui
      recopierait la jointure).
- [x] 2.5 Champs `player_share_of_team_min_pct` / `_max_pct` retirés : `domain`,
      `computeMetric` (variables `minShare`/`maxShare` comprises), openapi et
      `generated.ts` régénérés, deux tests Go recadrés (unitaire + intégration DuckDB).

Gate : `cd apps/go-api && go test ./... && go vet ./...` + `make check-types` + `make test-web`.

## Étape 3 — Bloc usage dans le drawer de comparaison (D8)

- [x] 3.1 Go — `SessionPageResponse.CompareUsage` ; `attachSessionUsage` scindé en
      `buildSessionUsage` (calcul d'UNE session) + un attacheur qui sert les deux,
      MIROIR de `attachSessionEventBlocks`. Réutilise `compareMatchesForEvents` : mêmes
      matchs que les blocs event-based, donc les deux colonnes parlent des mêmes matchs.
- [x] 3.2 `make openapi-gen` + `make generate-types`.
- [x] 3.3 Front — `SessionDetailPage` passe `data.compare_usage` au drawer ;
      `SessionColumnBody` propage `compact` ; `SessionUsageSection` prend `compact` et
      retire les trois formes larges (bande de régularité, piste du lobby, grilles
      famille × rôle et joueur × rôle). Parts et cadences restent des deux côtés.
- [x] 3.4 Go : bloc comparé servi quand des matchs comparés arrivent, jamais sinon, et
      jamais le même pointeur que le courant. Front : `SessionUsageSection.gate.test.tsx`
      compare pleine largeur / compact.

En cours d'exécution, nettoyage de la DOC INVERSÉE laissée par les étapes 1-2 (anti-pattern
n°9 du diagnostic CLAUDE.md) : en-têtes de `SessionUsageSection`, `SessionUsageForms`,
`usageLogic`, `usageGrids`, `usageI18n` et des deux fichiers de test — ils décrivaient
encore l'étendue, les « socles » et un corollaire en pied de carte qui n'existent plus.

Gate : suite Go complète + `make check-types` + `make test-web`.

---

## Journal

- **2026-09-09** — Plan créé après validation des maquettes par l'utilisateur (artefact
  « Refonte des cartes d'usage »). Décisions D1..D8 fermes.
- **2026-09-09** — ÉTAPE 1 CLOSE. Front seul, contrat inchangé. Cinq items `[x]`, aucun
  `[!]`. Deux arbitrages pris en cours d'exécution, tous deux dans l'esprit de D5 :
  (a) le titre de vue « Parts et parités » est retiré au-dessus des grilles de jauges — la
  grille porte désormais son propre en-tête qui NOMME le dénominateur rendu, un titre de
  plus aurait répété la même chose en moins précis ; l'`aria-label` de la section garde le
  nom de la vue, donc aucun test de sélection ne bouge ;
  (b) la cadence par 10 min des armes spéciales est CONSERVÉE au pied de la carte 2. La
  règle de tri du pied est « chiffre ou méthode » : un chiffre reste (il ne se survole
  pas), la méthode part dans l'aide. Le compte « matchs au-dessus de la parité LOBBY » de
  la bande de régularité, lui, est retiré — il était orphelin, aucune case de la bande ne
  le représentait (les cases se teintent contre la parité d'ÉQUIPE).
