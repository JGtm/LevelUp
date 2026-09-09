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

- [x] E0.1 Sur au moins 20 artefacts locaux : compter les `equipmentChanges` par `Kind`,
      et le nombre de `Slot` distincts. Publier le taux d'émissions manquées lu dans la
      couverture
- [x] E0.2 Établir la jointure **rang de palette → famille** : pour chaque `EquipmentChange.R`,
      retrouver le nom via `abilityLabels`. Mesurer le taux de rangs NON nommés. Une famille
      sans nom ne compte dans aucune ligne
- [x] E0.3 Établir la jointure **`Slot` → joueur** avec le pont du rejeu (`buildPlayers`,
      `indexBySlot`) — un slot est une VIE. Mesurer le taux de slots non rattachés
- [x] E0.4 Sur les mêmes artefacts, vérifier l'identité `taken ≈ utilisé + lâché + gardé`
      par joueur et par famille. Publier l'écart médian et le pire cas
- [x] E0.5 Écrire les résultats dans `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` (§2,
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

- [x] E1.1 `components/charts/valueGridModel.ts` — `ValueGridCell.segments?: Array<{ key, value, fraction, color, label }>`
      **OPTIONNEL**. La borne de colonne se calcule sur le TOTAL de la pile.
      Champ absent = comportement strictement inchangé
- [x] E1.2 `components/charts/ValueGrid.tsx` — rendu des segments, dans l'ordre du tableau,
      chacun avec son `aria-label` (même contrat d'accessibilité que les cellules simples)
- [x] E1.3 Tests `components/charts/valueGridModel.test.ts` : une cellule à trois segments
      s'empile sur la borne du total ; une cellule SANS segment rend exactement comme avant
      (test de non-régression sur la grille des objectifs de match-view)

**Gate** :
```bash
cd apps/web && npx vitest run src/components/charts && npx tsc -b --force
```

---

### E2 — Vue match : une colonne d'issue par famille

**Périmètre fermé :**

- [x] E2.1 `features/match-replay/model/equipmentUsageColumns.ts` — `UsageGroupKey` :
      `'deployed' | 'dropped'` → `'equipment'`. Une colonne par famille, deux ou trois
      segments par cellule. Le `Record` exhaustif force toutes les tables à suivre
- [x] E2.2 Même fichier — les familles d'ACTIVATION entrent dans la colonne avec le canal
      des épisodes comme côté « utilisé » (décision P2). **Ne PAS les exclure** : la
      rédaction initiale de la décision D9 du plan de la vague C le demandait, elle a été
      corrigée le 2026-09-09
- [x] E2.3 Même fichier — **exclure le répulseur** (décision P4) et les grenades (décision D5
      de la vague C, toujours en vigueur)
- [x] E2.4 `features/match-replay/model/equipmentUsageChart.ts` — `USAGE_GROUP_TOKENS` perd
      `dropped` ; les couleurs de segment viennent du §3.1
- [x] E2.5 `features/match-replay/i18n/i18n.ts` — libellés FR **et** EN. Le total de cellule
      s'écrit « N objets pris », **jamais « ramassés »** (décision P12)
- [x] E2.6 Afficher la réserve de couverture sous le tableau (décision P13)
- [x] E2.7 Tests : un power-up entre bien dans la colonne fusionnée ; le répulseur n'y est
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

- [x] E3.1 `internal/analysis/replay/usage_summary.go` — ajouter à `UsagePlayerSummary` :
      `TakenByFamily`, `SpentByFamily`, `KeptByFamily` (dérivé), et **persister**
      `DroppedByFamily` qui existe déjà en mémoire mais n'est pas dans la DDL
- [x] E3.2 Même fichier — **incrémenter `UsageSummaryRev`** (`us3` → `us4`). C'est la clé de
      reprise du backfill : sans ça, les matchs déjà résumés ne seront jamais re-résumés
- [x] E3.3 `internal/migration/` — migration de schéma pour les nouvelles colonnes, dans le
      style des migrations existantes du résumé d'usage. **Écriture INSERT-only via
      `persist.BatchBuilder` / le persister d'usage** — jamais d'UPSERT concurrent
      (règle anti-ART, ADR 0019/0026/0030)
- [x] E3.4 `internal/domain/session_usage.go` — étendre `SessionUsageMetric` avec les trois
      issues par famille. Champs `omitempty` : un titre sans le canal ne publie rien, il ne
      publie pas des zéros
- [x] E3.5 `internal/analysis/sessionusage/` — agrégation des trois issues, mêmes règles de
      scope que `computeMetric` (numérateurs ET dénominateurs sur le sous-ensemble à camp
      connu ; sous-ensemble vide = nil, jamais un 0 inventé)
- [x] E3.6 Les **taux de référence** qui excluent le joueur (décision P7) : « reste de mon
      équipe » = équipe moins moi ; « eux » = lobby moins mon équipe. Calculés côté Go, pas
      au client
- [~] E3.7 Capability : brancher sur `film.usage_summary` comme le bloc existant, **jamais
      sur le slug** (`no_slug_comparison_test.go` est un ratchet)
- [x] E3.8 `slog.WarnContext` sur toute dégradation (rang non nommé, slot non rattaché),
      avec un compteur — jamais d'erreur avalée
- [x] E3.9 `make openapi-gen` puis `make generate-types`
- [x] E3.10 Tests : `analysis` purs (les trois issues, le scope, nil ≠ 0) ; `duckdb` en
      `:memory:` pour la migration ; `service` avec mock repo
- [~] E3.11 **Recuisson** : lancer le backfill du résumé sur le parc local et vérifier que
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
  - **Chiffres E0 du 2026-09-09, reportés SANS enquête** (64 artefacts, recensement brut des
    poses). Les deux chiffres hérités sont CONFIRMÉS À L'IDENTIQUE : capteur **49 / 302**,
    mur **295 / 251**. Le rapport 1:6 du capteur n'est pas isolé — c'est le mur qui est
    l'exception : grappin 52/604, propulseur 36/463, répulseur 61/303, écran occultant 9/60,
    traqueur 3/16, champ de réparation 3/13, translocateur 0/8, surbouclier 0/9. Autrement
    dit **toutes les familles sauf le mur sont autour de 1:6 à 1:12**, et l'anomalie à
    expliquer est le 1:1 du mur — probablement la double publication de ses poses (appareil
    + panneaux) côté `deployed`, qui double son numérateur. Rien de plus n'a été cherché.
  - Vu au même endroit : à l'issue de la classification par fenêtre, le capteur ne compte que
    **4 objets « utilisés » sur 36 pris** et le traqueur **3 sur 29** — cohérent avec le
    rapport de poses ci-dessus, et cohérent avec un défaut d'attribution du poseur plutôt
    qu'avec un joueur qui garderait son capteur. Non instruit.

- **Nouvelle, E0 du 2026-09-09** : **8 artefacts sur 64 (12,5 %) ne portent AUCUNE table
  `abilityLabels`** — leur palette de capacités n'est pas classée, donc aucun de leurs rangs
  ne reçoit de nom. À eux seuls ils pèsent 148 des 453 rangs muets (32,7 %). Ce n'est pas un
  défaut du canal `equipmentChanges` : c'est le classement de palette (`replay/abilities.go`
  + `markers` de `replay_labels.toml`) qui ne signe pas ces films. Non instruit.

- **Nouvelle, E0 du 2026-09-09** : un rang **32** apparaît une fois dans le corpus, hors de
  tous les `markers` déclarés (famille A : 1..12, 23 ; famille B : 19..22). Une lecture
  isolée, aucune conclusion. Non instruit.

- **Nouvelle, E0 du 2026-09-09** : **1 422 `taken` mais seulement 1 223 fenêtres classables** —
  les 199 écarts sont des prises dont le rang n'a pas de famille connue ou dont la vie n'est
  pas nommée. Un dénominateur affiché « objets pris » devra dire lequel des deux il compte.
  Non instruit.

- **Nouvelle, E3 du 2026-09-09** : le stem `réparation` de la reconnaissance rang -> famille
  côté web (`EQUIPMENT_CHANGE_FAMILY_STEMS`, `equipmentUsageLogic.ts`) ne s'apparie à RIEN :
  le manifeste écrit « champ de reparation » SANS accent (`replay_labels.toml`, rang 23), et
  c'est son jumeau anglais `repair` qui classe le rang. Sans conséquence aujourd'hui (le
  libellé EN est toujours publié quand le FR l'est), mais le web perdrait ce rang si un film
  ne publiait que son français. Le Go écrit la forme réellement publiée. Non traité côté web
  (hors périmètre E3).

- **Nouvelle, E3 du 2026-09-09** : le document de rejeu ne publie PAS la table rang -> famille
  du manifeste (`AbilityPalette.Families`), seulement rang -> libellé bilingue
  (`abilityLabels`). Les deux consommateurs qui ont besoin de la famille depuis un document
  DÉJÀ CUIT (le web en E2, le résumé d'usage en E3) reconstruisent donc chacun la jointure
  par racine de libellé — deux copies, le plafond de la règle CLAUDE.md n°6. Publier
  `abilityFamilies` dans l'artefact fermerait le sujet, mais c'est une montée de schéma et
  une RE-CUISSON du parc, pas une re-projection : hors périmètre de ce chantier. Non instruit.

- **Nouvelle, E3 du 2026-09-09** : `deployed_<famille>` et `equipment_<famille>` coexistent
  désormais dans `metrics`. La première est un compte de GESTES (poses), la seconde un compte
  d'OBJETS (les trois issues) — deux grandeurs différentes, aucune n'est morte : la page
  Sessions rend aujourd'hui la première, et E4 décidera si la seconde la remplace à l'écran.
  Si E4 la remplace, `deployed_*` devra être RETIRÉ du contrat dans le même lot (règle « 0
  code mort »). Noté, non traité.

- **Nouvelle, E2 du 2026-09-09** : `npx eslint src/features/match-replay src/components/charts
  --max-warnings=0` échoue sur 9 avertissements dans cinq fichiers (`ReplayExportDialog.tsx`,
  `useReplayVehicles.ts`, `useReplaySound.ts`, `ReplayCanvas.tsx`, `ReplayFeedName.tsx`) —
  `react-hooks/exhaustive-deps`, `react-refresh/only-export-components`,
  `react-hooks/preserve-manual-memoization`. Vérifié PRÉEXISTANT : `git diff
  feat/v75...HEAD --stat` est vide sur les cinq fichiers, aucun n'est touché par ce chantier.
  Dette de lint antérieure — un `--max-warnings=0` sur ces dossiers ne peut donc pas passer
  tel quel tant qu'elle n'est pas résorbée. Non traité (hors périmètre E1/E2).

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

- **2026-09-09 — E0, sortie BRUTE de l'instrument** (`internal/analysis/replay/e0_gachis_research_test.go`,
  supprimé au commit suivant conformément à E0.5). Corpus : les 64 artefacts du parc local
  (`data/cache/replays/halo_infinite/*.json`, schéma 50, recuits le 2026-09-09, `*derived*`
  exclus) — le plan en exige 20. Gate : `go test ./internal/analysis/replay/ -run Research -v`,
  vert. Les xuids sont remplacés par `<joueur>`.

```
CORPUS : 64 artefacts lus dans <depot>/data/cache/replays/halo_infinite
== E0.1 VOLUMES DU CANAL `equipmentChanges` ==
  artefacts portant des changements 64/64 (100.00 %) · sans section `identity` 0 · sans table `abilityLabels` 8
  changements publies 1880 · `taken` 1422/1880 (75.64 %) · `spent` 458/1880 (24.36 %) · AUTRE Kind 0
  slots distincts porteurs (somme sur les artefacts, un slot = une VIE) 1372
== E0.1 TEMOIN DE COMPLETUDE (couverture du canal — le compteur de rotation) ==
  emissions MANQUEES 71 sur 1954 publiees + manquees = TAUX D'EMISSIONS MANQUEES 71/1954 (3.63 %)
  sauts de compteur 58 · reapparitions ecartees 1096 · recuperees (schema 38) 95 · premiere emission hors norme 96 · repetitions 1
== E0.2 JOINTURE RANG DE PALETTE -> FAMILLE (`abilityLabels` du document) ==
  rangs lus 1797 · NOMMES 1344/1797 (74.79 %) · NON NOMMES 453/1797 (25.21 %)
  causes du silence : film SANS palette classee 148/453 (32.67 %) · rang connu du manifeste mais absent de la table du film 0/453 (0.00 %) · rang NON ETABLI nulle part 305/453 (67.33 %)
  rang muet  19 : 167 lectures
  rang muet  10 : 95 lectures
  rang muet  22 : 90 lectures
  rang muet   6 : 26 lectures
  rang muet  20 : 22 lectures
  rang muet   8 : 17 lectures
  rang muet   9 : 14 lectures
  rang muet   5 : 9 lectures
  rang muet  12 : 4 lectures
  rang muet   4 : 4 lectures
  rang muet  11 : 4 lectures
  rang muet  32 : 1 lectures
== E0.3 JOINTURE `Slot` -> JOUEUR (registre d'identite publie, bornes par VIE) ==
  evenements rattaches 1859/1880 (98.88 %) · NON RATTACHES 21/1880 (1.12 %)
  slots porteurs rattaches 1355/1372 (98.76 %) · NON RATTACHES 17/1372 (1.24 %)
== E0.4 IDENTITE `taken ~ utilise + lache + garde`, PAR FAMILLE ==
  repulsor             pris  375 · utilise    0 · lache  210 · garde  161 · NON EXPLIQUE 4/375 (1.07 %)  <- AUCUN CANAL D'USAGE (negatif mesure) : « utilise » y est impossible
  grapple              pris  306 · utilise  145 · lache   96 · garde   65 · NON EXPLIQUE 0/306 (0.00 %)
  wall                 pris  128 · utilise   50 · lache   54 · garde   24 · NON EXPLIQUE 0/128 (0.00 %)
  thruster             pris  111 · utilise   43 · lache   35 · garde   28 · NON EXPLIQUE 5/111 (4.50 %)
  powerup_camo         pris   96 · utilise   80 · lache    6 · garde    5 · NON EXPLIQUE 5/96 (5.21 %)
  powerup_overshield   pris   82 · utilise   73 · lache    6 · garde    3 · NON EXPLIQUE 0/82 (0.00 %)
  translocator_beacon  pris   38 · utilise   16 · lache    6 · garde   13 · NON EXPLIQUE 3/38 (7.89 %)
  sensor               pris   36 · utilise    4 · lache   11 · garde   16 · NON EXPLIQUE 5/36 (13.89 %)
  threat_seeker        pris   29 · utilise    3 · lache   12 · garde    8 · NON EXPLIQUE 6/29 (20.69 %)
  repair_field         pris   22 · utilise    2 · lache    4 · garde   14 · NON EXPLIQUE 2/22 (9.09 %)
  TOUTES FAMILLES      pris 1223 · utilise  416 · lache  440 · garde  337 · NON EXPLIQUE 30/1223 (2.45 %)
  hors de toute fenetre de `taken` : usages 2024 · lachers 9495 — equipement de REAPPARITION (jamais `taken`) et manques du canal, comptes a part, JAMAIS dans l'identite
  ECART sur 488 couples (joueur, famille) · MEDIANE 0.00 % · PIRE CAS 100.00 % (<joueur>|thruster, 1 pris dont 1 non expliques)
  PIRE CAS a >= 5 prises : 20.00 % (<joueur>|repulsor, 5 pris dont 1 non expliques)
== RECENSEMENT BRUT DES POSES (rapporte au §6 du plan, non instruit) ==
  grenade_frag         deployees   219 · lachees  5019 · origine inconnue   388
  grenade_plasma       deployees    30 · lachees  1240 · origine inconnue    73
  grenade_spike        deployees    41 · lachees   941 · origine inconnue    69
  grenade_dynamo       deployees    19 · lachees   700 · origine inconnue    44
  grapple              deployees    52 · lachees   604 · origine inconnue    29
  wall                 deployees   295 · lachees   251 · origine inconnue    29
  thruster             deployees    36 · lachees   463 · origine inconnue    16
  sensor               deployees    49 · lachees   302 · origine inconnue    23
  repulsor             deployees    61 · lachees   303 · origine inconnue     6
  shroud_screen        deployees     9 · lachees    60 · origine inconnue     4
  threat_seeker        deployees     3 · lachees    16 · origine inconnue     0
  repair_field         deployees     3 · lachees    13 · origine inconnue     0
  powerup_camo         deployees     3 · lachees     8 · origine inconnue     0
  powerup_overshield   deployees     0 · lachees     9 · origine inconnue     0
  translocator_beacon  deployees     0 · lachees     8 · origine inconnue     0
  TOUTES FAMILLES      deployees   820 · lachees  9937 · origine inconnue   681 — RESERVE DE COUVERTURE des poses 681/11438 (5.95 %)
```

- **2026-09-09 — E0 CLOSE. DÉCISION DE SORTIE : ARRÊT PROPRE.** Les cinq items sont `[x]`, le
  gate (`go test ./internal/analysis/replay/ -run Research -v`) est vert, les cinq mesures
  sont au §2 du document de référence, l'instrument est supprimé.

  **Le critère d'arrêt du plan est franchi sur UN de ses deux seuils :**

  | Critère du plan | Seuil | Mesuré | Verdict |
  |---|---|---|---|
  | Taux de rangs de palette NON nommés | > 15 % → arrêt | **25,21 %** (453/1 797) | **FRANCHI** |
  | Écart médian de E0.4 | > 10 % → arrêt | **0,00 %** (médiane sur 488 couples) | tenu |

  Application littérale de la règle écrite : **la troisième issue (« gardé sans l'utiliser »)
  devient `[!]` et les étapes E2 à E6 livrent DEUX segments** (utilisé / lâché) au lieu de
  trois, jusqu'à décision contraire de l'utilisateur.

  **Ce que la mesure dit vraiment, pour que l'arbitrage soit possible.** Le seuil franchi
  porte sur le NOMMAGE, pas sur la fiabilité de la troisième issue — dont l'écart propre est
  de 2,45 % toutes familles (30 fenêtres sur 1 223 fermées par un `spent` qu'aucun canal
  d'usage ne voit), médiane 0,00 %. Les 25,21 % se décomposent en deux causes réparables et
  disjointes : **8,2 % des lectures** viennent de 8 artefacts sur 64 dont la palette n'est pas
  classée (réparation : classer la palette de ces films), et **17,0 %** de rangs non établis,
  concentrés sur trois d'entre eux — 19 (mur, famille B, 167 lectures), 10 (95) et 22
  (capteur, famille B, 90). Les rangs 19 et 22 sont délibérément non nommés faute d'un
  SECOND relevé Theater (`replay_labels.toml`, famille B) ; le rang 10 est signalé depuis le
  2026-08-14 comme « le premier trou à combler ». **Trois noms de rangs feraient tomber le
  taux sous le seuil** — c'est un chantier de manifeste, pas de canal.

  **Ce qui reste vrai quoi qu'il arrive** : le canal `equipmentChanges` tient (3,63 %
  d'émissions manquées, 1,12 % d'événements non rattachés à un joueur, aucun `Kind`
  inattendu), et le répulseur confirme son négatif mesuré (375 objets pris, **zéro** classé
  utilisé). La décision P4 — pas de ligne pour lui — est validée par la mesure.

  **Prochaine étape : décision utilisateur.** Ouvrir E1 avec deux segments, ou combler
  d'abord les trois rangs de palette et remesurer.

- **2026-09-09 — DÉCISION UTILISATEUR (questionnaire), qui AMENDE la décision de sortie
  ci-dessus.** On garde les TROIS issues (utilisé / lâché en mourant / gardé sans l'utiliser).
  Le seuil franchi (25,21 % de rangs non nommés) ne disqualifie pas la troisième issue : il
  dit seulement que certains objets pris ne peuvent pas être VENTILÉS PAR FAMILLE. La réponse
  produit n'est donc pas de réduire à deux segments, mais de rendre la réserve VISIBLE : les
  objets pris dont le rang n'a pas de famille connue ne sont ni cachés ni forcés dans une
  famille — ils forment une ligne de réserve sous le tableau, « N objets pris sans famille
  connue » (FR) / « N objects taken without a known family » (EN), au sens de la décision P13
  (la réserve de couverture s'affiche, elle ne se cache pas). Un lot de manifeste séparé
  (classer la palette des 8 films sans table, nommer les rangs 10/19/22) réduira cette réserve
  plus tard ; il n'est pas traité ici. E1 et E2 s'exécutent donc avec les TROIS segments.

- **2026-09-09 — E1 CLOSE.** `components/charts/valueGridModel.ts` porte `ValueGridCell.segments?`
  et `ValueGridInput.segments?` (E1.1), `ValueGrid.tsx` les rend en pile ordonnée avec un
  `aria-label` par segment via un sous-composant `ValueGridStack` (E1.2), tests ajoutés à
  `valueGridModel.test.ts` : une pile à trois segments s'empile sur la borne du TOTAL de la
  cellule (`value`, pas une somme recalculée des segments) ; une cellule sans callback
  `segments`, ou dont le callback rend `undefined` pour cette cellule précise (cas d'un
  appelant qui n'empile qu'une partie de ses colonnes), rend EXACTEMENT comme avant (E1.3).
  Gate : `npx vitest run src/components/charts` (34 fichiers, 307 tests, vert) et
  `npx tsc -b --force` (silencieux, vert). Aucune régression sur la grille des objectifs de
  match-view (`objectivesChart.ts`, qui ne fournit jamais `segments`).

- **2026-09-09 — E2 ouverte.** Périmètre déclaré (columns.ts/chart.ts/i18n.ts) étendu à
  `equipmentUsageLogic.ts` (calcul de `kept` par famille depuis `doc.equipmentChanges` — la
  plomberie que E2.1-E2.7 présupposent sans la nommer, cf. mandat de la tâche : « le gardé se
  dérive côté web depuis le document ») et à `gameChangers.ts` (pont inverse épisode->socle,
  déjà propriétaire du pont direct — CLAUDE.md n°6). Décision de conception : `EquipmentUsageColumns.deployed`/`.dropped`/`.episodes`
  restent INCHANGÉS (évite de casser ~10 assertions de `equipmentUsageLogic.test.ts` hors
  périmètre déclaré) ; un champ ADDITIF `equipment: string[]` porte la liste fusionnée. Le
  groupe d'affichage `episodes` (3 sous-colonnes compte/durée/frags) reste lui aussi RENDU
  tel quel dans `equipmentUsageColumns.ts` — camouflage/surbouclier gagnent EN PLUS une colonne
  dans la pile fusionnée (leur compte d'épisodes y sert de côté « utilisé », P2/D9 amendée) :
  légère redite du compte d'épisodes entre les deux blocs, jugée préférable à la suppression
  d'une fonctionnalité existante (durée, frags sous effet) hors du périmètre écrit de E2.

- **2026-09-09 — E2 CLOSE.** Tous les items `[x]`. Résumé technique :
  - `equipmentUsageLogic.ts` : nouveau champ `EquipmentUsageTally.kept` (dérivé, jamais lu
    d'un canal — `max(0, taken - utilisé - lâché)`), nouvelle fonction exportée
    `equipmentChangeFamilyOf` (reconnaissance rang -> famille par racine de libellé, même
    patron que `abilityChargeLogic.ts`/`placementTeleport.ts`), nouveau champ
    `EquipmentUsage.unnamedTaken` (réserve match, jamais une ligne de joueur), nouveau champ
    additif `EquipmentUsageColumns.equipment` (liste fusionnée, `deployed`/`dropped`/`episodes`
    restent INCHANGÉS pour ne pas rouvrir leurs ~10 assertions hors périmètre déclaré).
  - `equipmentUsageColumns.ts` : `UsageGroupKey` perd `deployed`/`dropped`, gagne `equipment` ;
    nouvelle fonction `equipmentGroup` construisant une colonne par famille avec `segments`
    (used/kept/dropped, ordre du §3.1) ; `equipmentFamilyLabel` pontée vers
    `padEquipmentFamily` pour camo/overshield via `droppedFamilyOf` (nouveau, `gameChangers.ts`,
    pont INVERSE de `EPISODE_FAMILY_OF_POWERUP`, CLAUDE.md n°6).
  - `equipmentUsageChart.ts` : `USAGE_GROUP_TOKENS` perd `dropped`, `deployed` renommé
    `equipment` (même jeton `frag-shoulder`) ; nouveaux `USAGE_OUTCOME_TOKENS`/
    `usageOutcomeColor` (§3.1 : `divergent-pos`/`divergent-neutral`/`divergent-neg`) ;
    `usageGestureCount('equipment', ...)` somme déployés + lâchés + activations des deux
    power-ups (la duplication du compte d'épisode avec la ligne `episodes` est documentée,
    pas un bug).
  - `i18nContract.ts` + `i18n.ts` : `groupDeployed`/`groupDropped` remplacés par
    `groupEquipment`, quatre nouveaux formats d'issue (`outcomeUsedFmt`/`outcomeKeptFmt`/
    `outcomeDroppedFmt`/`outcomeTotalTakenFmt` — total = « N objets pris », jamais
    « ramassés », P12) et deux formats de réserve (`coverageUnknownOriginFmt`/
    `coverageUnnamedTakenFmt`), FR et EN.
  - `MatchEquipmentUsageSection.tsx` : `UsageFootnotes` affiche les DEUX réserves de
    couverture (P13) — poses d'origine inconnue (`coverage.placements.byFamilyOrigin`,
    clés `*/unknown`) et objets pris sans famille connue (`usage.unnamedTaken`). Ce fichier
    n'était pas nommé dans le périmètre déclaré de E2, mais l'item E2.6 (« sous le tableau »)
    l'exige structurellement — seul fichier qui rend quelque chose sous la carte.
  - Tests neufs : `equipmentUsageLogic.kept.test.ts` (extrait de `equipmentUsageLogic.test.ts`,
    seuil de 500 lignes du dépôt franchi par les ajouts E2 — fixtures partagées déplacées vers
    `test/equipmentUsageFixtures.ts`, CLAUDE.md n°6) ; ajouts dans `equipmentUsageColumns.test.ts`
    (pile à un et trois segments) et `MatchEquipmentUsageSection.test.tsx` (les deux réserves,
    et leur silence quand rien ne les justifie).
  - **Comment « gardé » est dérivé** : au niveau du web, PAR JOUEUR ET PAR FAMILLE, depuis
    `doc.equipmentChanges` — `taken` compté par famille canonique (résolue via
    `equipmentChangeFamilyOf`), moins le côté « utilisé » déjà connu (poses déployées pour les
    déployables, compte d'épisodes pour les deux power-ups), moins `dropped` (ponté vers son
    vocabulaire de pose pour les power-ups). Clampé à zéro : l'écart résiduel mesuré par E0.4
    (2,45 % toutes familles, médiane 0,00 %) ne peut jamais produire un gardé négatif.
  - **Gates** : `npx vitest run src/components/charts src/features/match-replay/model` (999
    tests après les ajouts E2, vert), `npx vitest run` complet (673 fichiers, 7166 tests, 1
    fichier / 17 tests skippés — préexistants, non touchés — vert), `npx tsc -b --force`
    (silencieux), les deux greps couleur (IDENTIQUES à la baseline `feat/v75`, aucune ligne
    nouvelle). `npx eslint src/features/match-replay src/components/charts --max-warnings=0`
    échoue sur **9 avertissements PRÉEXISTANTS**, dans des fichiers hors diff de toute la
    branche (`ReplayExportDialog.tsx`, `useReplayVehicles.ts`, `useReplaySound.ts`,
    `ReplayCanvas.tsx`, `ReplayFeedName.tsx` — vérifié `git diff feat/v75...HEAD --stat`
    vide sur les cinq) : dette de lint antérieure à ce chantier, non traitée (règle « zéro
    fix opportuniste hors périmètre »). Zéro nouveau problème lint dans les fichiers touchés
    par E1/E2.
- **2026-09-09 (superviseur, apres E2)** — Amendement utilisateur : les objets pris dont le rang n'a
  pas de famille connue NE S'AFFICHENT PAS dans l'interface (ligne de reserve et cles i18n
  retirees ; `unnamedTaken` reste calcule par la logique pour l'outillage). Leur identification
  passera par un artefact interactif d'investigation Theater (match, date/heure, carte, timestamp,
  joueurs) produit en fin de chantier — cf. registre et plan master 2.6. Test inverse : aucune
  mention « sans famille connue » ne doit etre rendue.

- **2026-09-09 — E3 CLOSE.** Neuf items `[x]`, deux `[~]`. Trois commits :
  `58297a995` (types + analyse), `73be18490` (migration + persist), `3859b256c`
  (domaine + service + contrat).

  **Ce que la projection publie maintenant** (`us3` -> **`us4`**, E3.2 — sans cette montée
  aucun match déjà résumé ne serait re-projeté) : `UsagePlayerSummary` porte
  `TakenByFamily`, `SpentByFamily`, `KeptByFamily` (dérivé) et **persiste** enfin
  `DroppedByFamily`, qui n'existait qu'en mémoire. La règle du gardé est celle de la vue
  match à l'octet près — `max(0, taken - utilisé - lâché)`, « utilisé » étant les ÉPISODES
  pour les deux bonus et les POSES pour tout le reste (P2).

  **Trois décisions de conception, prises à l'exécution :**

  1. **La jointure rang -> famille se fait sur la RACINE DU LIBELLÉ, pas sur le manifeste.**
     Le manifeste porte pourtant la table exacte (`AbilityPalette.Families`), et deux calques
     du rejeu s'en servent — mais eux tournent À LA CONSTRUCTION du document.
     `BuildUsageSummary` est une fonction PURE du document DÉJÀ CUIT, et c'est ce qui permet
     au backfill de re-résumer le parc SANS re-décoder un seul film. Deuxième et dernière
     copie de la table (la première est côté web, E2). De surcroît la table du manifeste ne
     suffirait pas : les rangs 8 et 9 (les deux bonus) sont NOMMÉS sans porter de `family`.
  2. **Les quatre ventilations parlent le vocabulaire des POSES** (`powerup_camo`, jamais
     `camo`), là où le web nomme un bonus par son épisode dans `kept` et par sa pose dans
     `dropped` puis ponte les deux. L'agrégat de session joint donc les quatre sur UNE clé.
  3. **Nouvelle famille de clés `equipment_<famille>`**, à côté de `deployed_<famille>` et
     non à sa place. Deux raisons mesurées : une famille PRISE mais JAMAIS POSÉE n'aurait
     aucune ligne — c'est le capteur du parc, 4 objets utilisés sur 36 pris (E0.4), soit
     exactement l'histoire que le bloc doit raconter ; et la valeur d'une barre empilée doit
     être la SOMME DE SES SEGMENTS, sinon la pile déborde ou laisse un trou. Conséquence pour
     **E4** : c'est `equipment_<famille>` qui porte `outcomes`, pas `deployed_*` — et si E4
     retire `deployed_*` de l'écran, il doit le retirer du contrat dans le même lot (§6).

  **Contrat** (E3.4) : `SessionUsageMetric.outcomes` (`omitempty`) — utilisé / gardé / lâché,
  les prises comme dénominateur d'honnêteté, et TROIS taux : le mien, celui du **reste de mon
  équipe** et celui d'**eux** (E3.6, décision P7 — les deux références M'EXCLUENT). Règle de
  scope de `computeMetric` appliquée sans exception : les références sont des grandeurs
  d'équipe, donc numérateurs ET dénominateurs sur les seuls matchs à camp connu ;
  sous-ensemble vide = **nil**, jamais un 0 % qui se lirait « ils n'utilisent rien ». Une
  barre vide rend un taux nil : 0/0 n'est pas 0 %.

  **Migration** (E3.3) : `shared_match_usage_players_outcomes_v1` — quatre colonnes JSON
  **plus la RECRÉATION de `match_usage_players_latest`**. La vue est un `SELECT p.*` et DuckDB
  fige cette étoile à la création : sans la recréation, les colonnes existeraient en table et
  seraient invisibles au seul chemin de lecture autorisé (ADR 0026). Écriture INSERT-only,
  **zéro entrée ajoutée à l'allowlist anti-ART**. `DEFAULT '{}'` sans `NOT NULL` : DuckDB
  refuse toute contrainte sur un `ADD COLUMN` (échec TDD n°2 ci-dessous).

  **E3.7 statué `[~]`** — couvert par le câblage existant, vérifié sur pièces :
  `registry_pages.go:343` n'injecte `SessionUsageRepo` que si
  `capabilitiesForPDB(pdb).Has(games.CapFilmUsageSummary)`, et `replayartifacts/usage.go`
  gate la production sur la même capability. Les nouvelles grandeurs voyagent dans ce bloc :
  aucun branchement neuf à écrire, et surtout aucun `slug ==` introduit.

  **E3.8** : `slog.WarnContext` avec compteurs chez les DEUX producteurs de passes
  (`replayartifacts/usage.go` pour le fil de l'eau, `cmd_backfill_usage_summary.go` pour le
  corpus) — prises dont le rang n'a aucun libellé, changements dont le slot n'ouvre aucune
  ligne, consommations dont la chaîne du compteur est trouée (`gap > 0`, le document le dit
  lui-même). Comptés, jamais rangés dans une famille inventée, **jamais publiés dans une
  métrique ni dans une réserve UI** (amendement utilisateur du 09-09). Rien à zéro ne se dit.

  **Échecs TDD observés, dans l'ordre.** (1) Les six tests de projection ne compilaient pas
  (champs inexistants). (2) La migration a échoué : `Parser Error: Adding columns with
  constraints not yet supported`. (3) Un test de contrat partait d'une prémisse fausse
  (« aucune ligne d'équipement sans prise mesurée ») : le code avait raison — une POSE est une
  issue même sans prise — c'est le test qui a été corrigé. (4) Le ratchet
  `no_french_label_literal_test.go` a rejeté le stem accentué `réparation` ; vérification
  faite, le manifeste écrit « champ de reparation » SANS accent, ce stem ne s'appariait donc à
  rien, même côté web (reporté au §6) — remplacé par la forme réellement publiée, **aucune
  allowlist agrandie**. (5) `golangci-lint` : trois `goconst`, résorbés par deux constantes de
  famille.

  **Gates, tous passés sur l'arbre final** : `gofmt -l` silencieux · `go build ./...` ·
  `go vet ./...` · `go test ./...` vert · `go test -tags=integration -p 1 -count=1 ./...`
  vert · `go test ./internal/sync/ -run NoART` vert · `golangci-lint
  --new-from-merge-base=origin/main` **0 issue** · `make openapi-gen` + `make generate-types`
  (+32 lignes openapi, +17 generated.ts, commitées, aucun diff résiduel) ·
  `npx tsc -b --force` silencieux.

- **2026-09-09 — E3.11 statué `[~] superviseur` : LA RECUISSON RESTE À FAIRE.**
  Elle exige d'ouvrir `shared_matches_v2.duckdb` en **RW**, donc **serveur de dev arrêté**
  (`OpenReadWrite` échoue si le lock est tenu — précondition écrite de la commande, valable
  y compris pour `--dry-run`, qui joue les migrations). La consigne de cette tâche interdisant
  toute ouverture d'une base sous `data/`, elle n'a pas été lancée ici.

  ```bash
  # 1) Serveur arrete. Controle a blanc : ce qui SERA repris, aucune ecriture.
  cd apps/go-api && go run ./cmd/levelup backfill-usage-summary --dry-run
  # 2) La passe reelle.
  cd apps/go-api && go run ./cmd/levelup backfill-usage-summary
  ```

  **`--force` n'est PAS nécessaire, et il ne faut pas l'employer.** La clé de reprise est
  `(summary_rev, artifact_schema)` lue sur `match_usage_films_latest` : le passage `us3` ->
  `us4` suffit à ce que CHAQUE match déjà résumé soit repris. `--force` ne servirait qu'au cas
  — qui ne doit jamais se produire — d'une règle changée sans montée de révision.

  **Attendu** : `resume usage : N ecrits, 0 deja a jour, ...` (le `0 deja a jour` EST la
  preuve que la montée de révision a mordu ; un compte non nul dirait que `us4` n'a pas été
  pris en compte). Le `--dry-run` imprime en plus la ligne neuve `couverture des ramassages :
  N prises sans famille connue, N slots non rattaches, N consommations a chaine trouee` —
  à rapprocher des mesures E0.2 (25,21 % de rangs non nommés) et E0.3 (1,12 %). **Consigner
  le compte d'écrits au journal une fois la passe faite.**
