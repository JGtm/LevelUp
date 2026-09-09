# PLAN — « Servi ou gâché » : équipement et armes spéciales sur quatre pages (2026-09-09)

**Pour l'agent qui exécute.** Ce plan est autoportant. Avant la première ligne de code :

1. Lire `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` — **obligatoire**. C'est le seul
   document qui dit ce que le film mesure. Répondre de mémoire sur ce sujet a produit trois
   affirmations fausses dans la conversation qui a mené à ce plan.
2. Invoquer le skill `plan-execution` : c'est le contrat d'exécution (ordre strict, aucun
   report d'action faisable, statut sur chaque case, gate avant clôture).
3. Invoquer le skill `color-tokens` avant toute ligne de rendu. Le §3 de ce plan fixe la
   correspondance marque → jeton ; **aucun hex, aucune classe Tailwind de couleur** dans
   `features/` ni `components/`.
4. Reprise de session : l'avancement est dans ce fichier (cases + journal en fin). Reprendre
   à la première case non statuée de l'étape courante. Ne pas re-décider ce qui est au §2.

**Maquette validée par l'utilisateur le 2026-09-09** : artefact « Équipement, quatre
lectures », https://claude.ai/code/artifact/fd673170-e415-4c86-9e4c-6c9466043200

**Branche** : `wt/equipement-gachis` depuis `feat/v75`, dans un worktree dédié
(`LevelUp-wt-equipement-gachis`). Ne jamais travailler dans le worktree partagé de session.

**Critère de succès global** : sur les quatre pages, un joueur peut répondre à « est-ce que
je fais ma part » ET « est-ce que je gâche mon équipement », avec une référence qui ne le
contient pas.

---

## 1. Rapport avec les plans existants

| Plan | Rapport |
|---|---|
| `.ai/PLAN_RETOURS_VAGUE_C_FORMES_2026-09-08.md` lot **C6** (fusion déployé/lâché) | **REMPLACÉ** par les étapes E2 et E3 de ce plan. La forme validée a changé : trois issues au lieu de deux, et les bonus entrent dans la barre (correction de la décision D9, déjà écrite dans ce plan-là le 2026-09-09) |
| Même plan, lot **C5** (blocs sur Escouade et Synthèse) | **ABSORBÉ** par les étapes E5 et E6 |
| `.ai/PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md` | **CLOS**, livré le même jour. Ce plan reprend son vocabulaire et ses formes. Ne pas défaire son travail |

**Action à faire à l'ouverture de ce chantier** : annoter le plan de la vague C pour dire que
C5 et C6 sont repris ici, sinon deux agents travailleront sur le même sujet.

---

## 2. Décisions produit — TRANCHÉES, ne pas rouvrir

| # | Décision |
|---|---|
| **P1** | Tout objet d'équipement ramassé finit d'**une seule** de trois façons : **utilisé**, **lâché en mourant**, **gardé sans l'utiliser** (fin de match l'arme encore en poche). Les trois sont exclusives ; leur somme est le nombre d'objets ramassés sur la carte |
| **P2** | « Utilisé » a **deux définitions** selon la famille, et le lecteur ne voit pas la différence : un équipement d'ACTIVATION sert quand il est activé (camouflage, surbouclier, grappin, translocateur, propulseur) ; un DÉPLOYABLE sert quand il est posé (mur, capteur, écran occultant, traqueur, champ de réparation, balise) |
| **P3** | **UN SEUL graphe équipement**, toutes familles dans la même liste. Ne jamais séparer « activés » et « déployés » à l'écran — c'est un détail de calcul |
| **P4** | Le **répulseur n'a pas de ligne**. Son activation n'est mesurée par aucun canal (négatif mesuré, 9 canaux). Une ligne dirait « 0 utilisation » là où la vérité est « non mesuré » |
| **P5** | Les armes spéciales suivent la même grammaire, avec une **troisième définition de « utilisé » : avoir tiré**. Deux issues seulement (tirée / jamais tirée) — « gardée sans l'utiliser » n'a pas de sens pour une arme |
| **P6** | **Une seule barre** porte les deux lectures : sa longueur est ma part, son remplissage est l'issue |
| **P7** | Les **repères de taux sont des TAUX, pas des comptes**, et la référence **m'exclut** : « le reste de mon équipe » et « eux ». Se comparer à une moyenne qui vous contient amortit le signal |
| **P8** | **Sessions** : axe en pourcentage de l'équipe, trait de parité. Livré tel quel — **cette page est validée, ne pas la retoucher au-delà de ce que l'étape E4 décrit** |
| **P9** | **Solo et Escouade** : axe en **comptes d'objets**, aucun pourcentage dans les barres, **pas de trait de parité** (il n'a pas de position sur un axe de comptes) |
| **P10** | **Solo et Escouade** ajoutent **deux donuts** (un équipement, un armes spéciales) qui portent les parts. UN anneau, parts **exclusives** faisant 100 % du lobby. L'emboîtement se lit par **contiguïté** : arcs rangés moi → mes amis → reste de l'équipe → eux, sous-totaux « mon escouade » et « mon équipe » écrits sous la légende |
| **P11** | Les **valeurs sont sur les arcs** du donut ; la légende ne dit que la couleur. Le **centre porte le volume** (compte total du lobby) |
| **P12** | Le **total de la barre n'est JAMAIS étiqueté « ramassés »** au sens large : le canal écarte les annonces de réapparition. Libellé exact : « objets pris » / « prises » |
| **P13** | La **réserve de couverture s'affiche**, elle ne se cache pas : ~5 % des poses sont d'origine inconnue, et le canal des ramassages annonce ~5 % d'émissions manquées |

---

## 3. COULEURS — la table normative

**Aucune couleur n'est choisie par l'agent.** Toute marque de ce chantier prend son jeton
ici. Les jetons existent tous dans `apps/web/src/lib/accessibility/semantic-tokens.ts` ;
aucun jeton nouveau n'est à créer.

### 3.1 Les segments d'issue (le remplissage de la barre)

| Marque | Jeton | Pourquoi ce jeton |
|---|---|---|
| Utilisé | `divergent-pos` | Gamme ordinale bon / neutre / mauvais, déjà employée par la bande de régularité du même bloc. L'issue EST un jugement |
| Gardé sans l'utiliser | `divergent-neutral` | Ni servi ni perdu pour l'adversaire |
| Lâché en mourant | `divergent-neg` | La pire des trois : un adversaire a pu le reprendre |
| Arme tirée au moins une fois | `divergent-pos` | Même gamme, même sens |
| Arme prise et jamais tirée | `divergent-neg` | Idem |

### 3.2 Les repères et références

| Marque | Jeton / traitement | Règle |
|---|---|---|
| Trait de parité (Sessions seulement) | `warning` | Jeton DISTINCT, **jamais une teinte de donnée** — règle déjà en vigueur dans `SessionUsageForms.tsx`. Ne pas y toucher |
| Repère « taux du reste de mon équipe » | `team-ally` | La référence EST mon camp |
| Repère « taux de eux » | **Pas de jeton de donnée** : trait pointillé en `muted-foreground` | La règle du bloc usage veut que l'adversaire ne soit jamais coloré. Un pointillé neutre, comme la hachure de la piste du lobby |

### 3.3 Les parts du donut

| Part | Jeton |
|---|---|
| Moi | `squad-player-1` |
| Mes amis (Escouade) | `squad-player-2`, `squad-player-3`, `squad-player-4` — dans l'ordre de `SQUAD_TEAMMATE_COLOR_TOKENS` (`features/squad/colors.ts`), pour qu'un joueur garde sa couleur d'une page à l'autre |
| Reste de l'équipe | `team-ally` |
| Eux | `team-enemy` |

> **DÉCISION À CONSIGNER À L'EXÉCUTION (P14)** : employer `team-enemy` pour la part « Eux »
> **amende** la règle locale du bloc usage, qui veut que l'adversaire n'existe que comme
> dénominateur hachuré et anonyme. Dans un donut, une part doit être une part — un secteur
> hachuré serait illisible. L'adversaire reste **agrégé et non nommé** : la règle de fond
> tient, seule sa traduction visuelle change. Écrire cet amendement dans l'en-tête du
> composant donut ET dans le journal de ce plan.

### 3.4 Les APIs de couleur — obligatoire

| Contexte | Helper |
|---|---|
| `className` / `style` en JSX | `tokenCssVar(token)` |
| ECharts (le donut passe par `DonutChart`) | `sliceColors: Record<string, SemanticToken>` — **le composant résout lui-même**, ne pas résoudre à la main |
| SVG écrit à la main | `resolveToken(token, tokens)` |
| N coéquipiers | `getSeriesColors(n, tokens[])` |

**Interdit** : `#RRGGBB`, `color-mix(...)` sur un hex, `text-red-*` / `bg-*` de Tailwind.
Le texte posé SUR un aplat coloré (valeur dans un arc, libellé dans un segment) est la seule
exception tolérée et s'écrit `text-white` avec le commentaire d'usage — c'est le contraste
d'un texte dans un aplat, pas une couleur sémantique (précédent : `UsageLobbyTrack`,
`MatchNemesisCards`).

**Vérification de fin d'étape** :
```bash
grep -rn "#[0-9a-fA-F]\{6\}" apps/web/src/features/ apps/web/src/components/ | grep -v "\.test\."
grep -rn "text-\(red\|green\|blue\|yellow\|amber\|rose\|slate\|gray\)-" apps/web/src/features/ apps/web/src/components/
```
Les deux doivent ne rien rendre de nouveau par rapport à la baseline de la branche.

---

## 4. Ce que la mesure fournit — résumé opérationnel

Détail et références de code : `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`. Résumé :

| Besoin | Canal | Disponible où |
|---|---|---|
| Utilisé — déployables | `equipmentPlacements` `origin: deployed` | Document ; résumé session (`DeployedByFamily`) |
| Utilisé — camouflage, surbouclier | `equipmentEpisodes` | Document ; résumé session |
| Utilisé — grappin | `grappleLines` | Document ; résumé session (`GrapplePulls`) |
| Utilisé — translocateur | `translocations` | Document **seulement** |
| Utilisé — propulseur | `abilityImpulses` | Document **seulement** |
| Lâché en mourant | `equipmentPlacements` `origin: dropped` | Document (par famille) ; résumé session **en TOTAL seulement** |
| Ramassé / consommé | `equipmentChanges` (`taken` / `spent`) | Document **seulement**, et **lu par aucun écran d'usage** |
| Gardé jusqu'à la fin | `taken` sans `spent` ni `dropped` | **À dériver** |
| Arme : prise | `padPickups` + `pickups` + `weaponChanges` | Document ; résumé session (`PadPickups`) |
| Arme : a tiré | `shots` | Document **seulement** |

**Conséquence structurante** : la vue match peut tout servir sans recuisson. Sessions, Solo
et Escouade passent par `UsagePlayerSummary` et ne peuvent rien servir de neuf **avant
l'étape E3** (nouveaux champs + recuisson).

---

## 5. Étapes

Ordre strict. Une étape est CLOSE quand : gate passé, toutes ses cases statuées
(`[x]` fait / `[~]` couvert ailleurs avec référence / `[!]` non traité avec justification
écrite au journal), plan mis à jour, entrée `.ai/thought_log.md`, point d'étape à
l'utilisateur. **Zéro fix hors périmètre** : toute découverte va au §6, non traitée.

---

### E0 — Vérifier `equipmentChanges` sur le corpus (aucun rendu)

> Toute la valeur ajoutée de ce chantier repose sur ce canal. S'il ne tient pas, les étapes
> E3 à E6 se réduisent à deux issues au lieu de trois, et il faut le savoir AVANT.

**Périmètre fermé — instrument de mesure jetable, sous `apps/go-api/internal/analysis/replay/`
en `*_research_test.go` (patron `filmdec/r11_*_research_test.go`) :**

- [ ] E0.1 Sur au moins 20 artefacts locaux : compter les `equipmentChanges` par `Kind`,
      et le nombre de `Slot` distincts. Publier le taux d'émissions manquées lu dans la
      couverture
- [ ] E0.2 Établir la jointure **rang de palette → famille** : pour chaque `EquipmentChange.R`,
      retrouver le nom via `abilityLabels`. Mesurer le taux de rangs NON nommés. Une famille
      sans nom ne compte dans aucune ligne
- [ ] E0.3 Établir la jointure **`Slot` → joueur** avec le pont du rejeu (`buildPlayers`,
      `indexBySlot`) — un slot est une VIE. Mesurer le taux de slots non rattachés
- [ ] E0.4 Sur les mêmes artefacts, vérifier l'identité `taken ≈ utilisé + lâché + gardé`
      par joueur et par famille. Publier l'écart médian et le pire cas
- [ ] E0.5 Écrire les résultats dans `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` (§2,
      colonne « État »), **avec les chiffres**, et supprimer l'instrument

**Gate** :
```bash
cd apps/go-api && go test ./internal/analysis/replay/ -run Research -v
```
Sortie verte ET les cinq mesures écrites dans le document de référence.

**Décision de sortie** : si le taux de rangs non nommés dépasse 15 %, ou si l'écart médian
de E0.4 dépasse 10 %, **arrêter proprement** et remonter à l'utilisateur — la troisième
issue devient un `[!]` et les étapes suivantes livrent deux segments.

---

### E1 — La cellule empilée dans `ValueGrid` (front, aucune donnée nouvelle)

**Périmètre fermé :**

- [ ] E1.1 `components/charts/valueGridModel.ts` — `ValueGridCell.segments?: Array<{ key, value, fraction, color, label }>`
      **OPTIONNEL**. La borne de colonne se calcule sur le TOTAL de la pile.
      Champ absent = comportement strictement inchangé
- [ ] E1.2 `components/charts/ValueGrid.tsx` — rendu des segments, dans l'ordre du tableau,
      chacun avec son `aria-label` (même contrat d'accessibilité que les cellules simples)
- [ ] E1.3 Tests `components/charts/valueGridModel.test.ts` : une cellule à trois segments
      s'empile sur la borne du total ; une cellule SANS segment rend exactement comme avant
      (test de non-régression sur la grille des objectifs de match-view)

**Gate** :
```bash
cd apps/web && npx vitest run src/components/charts && npx tsc -b --force
```

---

### E2 — Vue match : une colonne d'issue par famille

**Périmètre fermé :**

- [ ] E2.1 `features/match-replay/model/equipmentUsageColumns.ts` — `UsageGroupKey` :
      `'deployed' | 'dropped'` → `'equipment'`. Une colonne par famille, deux ou trois
      segments par cellule. Le `Record` exhaustif force toutes les tables à suivre
- [ ] E2.2 Même fichier — les familles d'ACTIVATION entrent dans la colonne avec le canal
      des épisodes comme côté « utilisé » (décision P2). **Ne PAS les exclure** : la
      rédaction initiale de la décision D9 du plan de la vague C le demandait, elle a été
      corrigée le 2026-09-09
- [ ] E2.3 Même fichier — **exclure le répulseur** (décision P4) et les grenades (décision D5
      de la vague C, toujours en vigueur)
- [ ] E2.4 `features/match-replay/model/equipmentUsageChart.ts` — `USAGE_GROUP_TOKENS` perd
      `dropped` ; les couleurs de segment viennent du §3.1
- [ ] E2.5 `features/match-replay/i18n/i18n.ts` — libellés FR **et** EN. Le total de cellule
      s'écrit « N objets pris », **jamais « ramassés »** (décision P12)
- [ ] E2.6 Afficher la réserve de couverture sous le tableau (décision P13)
- [ ] E2.7 Tests : un power-up entre bien dans la colonne fusionnée ; le répulseur n'y est
      pas ; une cellule à un seul segment reste lisible

**Gate** :
```bash
cd apps/web && npx vitest run src/components/charts src/features/match-replay/model && npx tsc -b --force
cd apps/web && npx eslint src/features/match-replay --max-warnings=0
```
Plus les deux greps couleur du §3.4.

---

### E3 — Backend : les grandeurs manquantes au grain session (LE lot lourd)

> Sans cette étape, Sessions, Solo et Escouade ne peuvent servir ni la ventilation des
> lâchers par famille, ni le dénominateur des ramassages, ni la troisième issue.

**Périmètre fermé — Go :**

- [ ] E3.1 `internal/analysis/replay/usage_summary.go` — ajouter à `UsagePlayerSummary` :
      `TakenByFamily`, `SpentByFamily`, `KeptByFamily` (dérivé), et **persister**
      `DroppedByFamily` qui existe déjà en mémoire mais n'est pas dans la DDL
- [ ] E3.2 Même fichier — **incrémenter `UsageSummaryRev`** (`us3` → `us4`). C'est la clé de
      reprise du backfill : sans ça, les matchs déjà résumés ne seront jamais re-résumés
- [ ] E3.3 `internal/migration/` — migration de schéma pour les nouvelles colonnes, dans le
      style des migrations existantes du résumé d'usage. **Écriture INSERT-only via
      `persist.BatchBuilder` / le persister d'usage** — jamais d'UPSERT concurrent
      (règle anti-ART, ADR 0019/0026/0030)
- [ ] E3.4 `internal/domain/session_usage.go` — étendre `SessionUsageMetric` avec les trois
      issues par famille. Champs `omitempty` : un titre sans le canal ne publie rien, il ne
      publie pas des zéros
- [ ] E3.5 `internal/analysis/sessionusage/` — agrégation des trois issues, mêmes règles de
      scope que `computeMetric` (numérateurs ET dénominateurs sur le sous-ensemble à camp
      connu ; sous-ensemble vide = nil, jamais un 0 inventé)
- [ ] E3.6 Les **taux de référence** qui excluent le joueur (décision P7) : « reste de mon
      équipe » = équipe moins moi ; « eux » = lobby moins mon équipe. Calculés côté Go, pas
      au client
- [ ] E3.7 Capability : brancher sur `film.usage_summary` comme le bloc existant, **jamais
      sur le slug** (`no_slug_comparison_test.go` est un ratchet)
- [ ] E3.8 `slog.WarnContext` sur toute dégradation (rang non nommé, slot non rattaché),
      avec un compteur — jamais d'erreur avalée
- [ ] E3.9 `make openapi-gen` puis `make generate-types`
- [ ] E3.10 Tests : `analysis` purs (les trois issues, le scope, nil ≠ 0) ; `duckdb` en
      `:memory:` pour la migration ; `service` avec mock repo
- [ ] E3.11 **Recuisson** : lancer le backfill du résumé sur le parc local et vérifier que
      les matchs à l'ancienne révision sont bien repris. Consigner le compte au journal

**Gate** :
```bash
cd apps/go-api && go test ./... && go vet ./...
cd apps/go-api && go test -tags=integration -p 1 ./...   # OBLIGATOIRE : le diff touche migration/ et persist/
cd apps/web && npx tsc -b --force
```
`-p 1` non négociable : le driver DuckDB est mono-process, un run parallèle donne un faux vert.

---

### E4 — Sessions : la barre combinée (page déjà validée, périmètre étroit)

> Cette page a été livrée le 2026-09-09. **Ne toucher que ce qui suit.**

- [ ] E4.1 `features/session-detail/SessionUsageForms.tsx` — `UsageGauge` : le remplissage
      devient une pile à trois segments (§3.1), la longueur reste la part. Le trait de
      parité ne bouge pas
- [ ] E4.2 Même fichier — les deux repères de taux DANS la tranche (décision P7), aux jetons
      du §3.2. Ce sont des marques, jamais des chiffres affichés
- [ ] E4.3 `features/session-detail/usageLogic.ts` — projeter les trois issues et les deux
      taux de référence depuis le contrat étendu en E3
- [ ] E4.4 `features/session-detail/usageI18n.ts` — libellés des trois issues et des deux
      repères, FR **et** EN, parité par typage
- [ ] E4.5 La ligne « Objets lâchés » **disparaît de la liste des grandeurs** : une mort
      n'est pas un geste, elle est devenue un segment
- [ ] E4.6 Tests : la pile respecte l'ordre utilisé → lâché → gardé ; une famille sans
      troisième issue rend deux segments ; le compte brut reste en infobulle

**Gate** :
```bash
cd apps/web && npx vitest run src/features/session-detail && npx tsc -b --force
cd apps/web && npx eslint src/features/session-detail --max-warnings=0
```

---

### E5 — Le bloc partagé, puis Solo / Synthèse

> `features/squad` et `features/synthesis` **ne peuvent pas importer** `features/session-detail`
> (ratchet `lint-cross-feature-imports`). Le bloc doit déménager AVANT d'être réutilisé.

**Périmètre fermé — extraction :**

- [ ] E5.1 Déplacer le bloc dans `apps/web/src/features/_shared/usage/` (ou
      `components/usage/` si le ratchet l'exige) : formes, projections, dictionnaire.
      `features/session-detail` l'importe désormais au lieu de le porter
- [ ] E5.2 Vérifier qu'aucun test de `session-detail` ne casse — c'est un déménagement, pas
      une réécriture
- [ ] E5.3 Poser le garde-rail : un test grep interdit qu'une feature réintroduise une copie
      locale des formes d'usage (règle CLAUDE.md n°6 — une factorisation sans garde-rail
      re-diverge)

**Périmètre fermé — Go :**

- [ ] E5.4 `internal/domain/` — publier le bloc d'usage pour la Synthèse, en types canoniques
- [ ] E5.5 `internal/service/` — l'orchestration ; **aucun SQL inline**
- [ ] E5.6 `internal/platform/duckdb/` — réutiliser la lecture de la page Sessions ; n'écrire
      une requête neuve qu'après avoir vérifié qu'aucune ne convient
- [ ] E5.7 `make openapi-gen` + `make generate-types`

**Périmètre fermé — Web :**

- [ ] E5.8 `features/synthesis/` — le bloc, en variante **comptes** (décision P9) : axe en
      objets pris, aucun pourcentage dans les barres, aucun trait de parité, lignes triées
      du plus pris au moins pris
- [ ] E5.9 Les deux donuts (décision P10, P11) via `components/charts/DonutChart`, avec
      `sliceColors` du §3.3, `centerValue` = le compte du lobby, `centerLabel` = l'unité.
      **Ne pas écrire un donut à la main** — la primitive existe
- [ ] E5.10 Les deux sous-totaux emboîtés sous la légende, séparés par un filet
- [ ] E5.11 Query keys dans `lib/query/keys.ts`, jamais en ligne
- [ ] E5.12 Strings FR **et** EN
- [ ] E5.13 Tests : `analysis` purs, `service` avec mock, front sur la projection

**Gate** :
```bash
make go-api-test && cd apps/go-api && go test -tags=integration -p 1 ./...
cd apps/web && npx vitest run src/features/synthesis src/features/session-detail src/features/_shared
cd apps/web && npx tsc -b --force && npx eslint src/features --max-warnings=0
```

---

### E6 — Escouade

- [ ] E6.1 Go — même publication que E5, agrégée **par joueur suivi** (réutiliser
      `ResolveTrackedSquad`, déjà en place pour le contexte escouade de Sessions)
- [ ] E6.2 Web — les deux graphes, **une ligne par coéquipier** au lieu d'une par famille.
      Variante comptes (décision P9)
- [ ] E6.3 Les deux donuts à **quatre parts** : moi, mes amis, reste de l'équipe, eux —
      avec les deux sous-totaux « mon escouade » et « mon équipe »
- [ ] E6.4 Les couleurs de joueur viennent de `features/squad/colors.ts`
      (`SQUAD_MAIN_PLAYER_TOKEN`, `SQUAD_TEAMMATE_COLOR_TOKENS`), **source unique** : un
      joueur garde sa couleur d'une page à l'autre
- [ ] E6.5 Tests service + front

**Gate** :
```bash
make go-api-test && cd apps/go-api && go test -tags=integration -p 1 ./...
cd apps/web && npx vitest run src/features/squad && npx tsc -b --force
cd apps/web && npx eslint src/features/squad --max-warnings=0
```

---

## 6. Découvertes — à consigner, PAS à traiter

- **Ouverte, héritée du plan de la vague C** : le capteur affiche 49 déploiements pour 302
  lâchers sur le parc (1:6), quand le mur est à 1:1 (295/251). Vrai comportement de jeu ou
  défaut de classement d'origine — chantier à part. **L'étape E0 donnera des chiffres
  utiles : les reporter ici sans enquêter.**

_(à compléter en cours d'exécution)_

---

## 7. Clôture de chantier

- [ ] `make gate-push` vert
- [ ] **Une** revue adversariale sur le diff d'intégration complet — pas par étape
      (skill `adversarial-review`)
- [ ] `delivery-checklist` avant la demande de merge
- [ ] `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` mis à jour de tout ce que E0 a mesuré
- [ ] Le plan de la vague C annoté : C5 et C6 repris ici
- [ ] Entrée `.ai/thought_log.md`
- [ ] **Ne jamais merger vers `main` sans l'utilisateur** — push sur `main` = déploiement prod

---

## Journal

- **2026-09-09** — Plan créé. Formes validées par l'utilisateur sur maquette après une
  longue mise au point : un seul graphe équipement (pas de séparation activés/déployés),
  barre combinée part + issue, références qui excluent le joueur, donuts à parts exclusives
  sur Solo et Escouade. Décisions P1..P13 fermes.
