# PLAN — « Servi ou gâché » : équipement et armes spéciales sur quatre pages (2026-09-09)

> **Statut au 2026-09-10 (superviseur du plan maître)** : E0 à E6 exécutés via
> `.ai/PLAN_MASTER_2026-09-09.md` (lots 2.1 à 2.5, E6.1bis, revue 2.R, gate-push et CI de
> vague 2 verts) ; la liste de clôture §11 est couverte au niveau de la vague (revue unique,
> gate-push, thought_log, référence des canaux mise à jour par E0 puis par les lots 4.3, 5.5 et
> la ronde de corrections de la vague 5). Reste ouvert ici : les découvertes §6 (mesures E0
> instruites par `.ai/V7.5/RAPPORT_E0_2026-09-10.md`) et P5 (armes spéciales au grain session,
> reporté par décision utilisateur du 10-09).

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
| `.ai/V7.5/PLAN_RETOURS_VAGUE_C_FORMES_2026-09-08.md` lot **C6** (fusion déployé/lâché) | **REMPLACÉ** par les étapes E2 et E3 de ce plan. La forme validée a changé : trois issues au lieu de deux, et les bonus entrent dans la barre (correction de la décision D9, déjà écrite dans ce plan-là le 2026-09-09) |
| Même plan, lot **C5** (blocs sur Escouade et Synthèse) | **ABSORBÉ** par les étapes E5 et E6 |
| `.ai/V7.5/PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md` | **CLOS**, livré le même jour. Ce plan reprend son vocabulaire et ses formes. Ne pas défaire son travail |

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

- [x] E4.1 `features/session-detail/SessionUsageForms.tsx` — `UsageGauge` : le remplissage
      devient une pile à trois segments (§3.1), la longueur reste la part. Le trait de
      parité ne bouge pas
- [x] E4.2 Même fichier — les deux repères de taux DANS la tranche (décision P7), aux jetons
      du §3.2. Ce sont des marques, jamais des chiffres affichés
- [x] E4.3 `features/session-detail/usageLogic.ts` — projeter les trois issues et les deux
      taux de référence depuis le contrat étendu en E3
- [x] E4.4 `features/session-detail/usageI18n.ts` — libellés des trois issues et des deux
      repères, FR **et** EN, parité par typage
- [x] E4.5 La ligne « Objets lâchés » **disparaît de la liste des grandeurs** : une mort
      n'est pas un geste, elle est devenue un segment
- [x] E4.6 Tests : la pile respecte l'ordre utilisé → gardé → lâché (ordre de la table §3.1, décision S10 du master plan 2026-09-09 : cet item disait « utilisé → lâché → gardé », contradiction relevée par la revue de vague) ; une famille sans
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

- [x] E5.1 Déplacer le bloc dans `apps/web/src/features/_shared/usage/` : formes,
      projections, dictionnaire. `features/session-detail` l'importe désormais au lieu de le
      porter. `_shared/` n'entre pas dans la regex de nom de feature du ratchet
      `lint-cross-feature-imports` (`[a-z0-9-]+` ne matche pas `_shared`, vérifié sur pièces :
      ni comme consommateur, ni comme cible importée) — aucune entrée `ALLOWED_CROSS_IMPORTS`
      n'était donc nécessaire, précédent déjà établi par `firstBlood.ts` / `EncounterSplitBars.tsx`
  - [x] E5.1bis Scission `usageLogic.ts` (680 L, seuil 500) EN MÊME TEMPS que le déplacement —
        on ne peut pas déplacer un fichier hors-seuil puis le scinder ensuite sans double
        churn sur les imports/tests ; découpe par responsabilité (pas au hasard) en 8 fichiers
        du dossier `_shared/usage/` : `usageFormat.ts` (formatage, 56 L), `usageMetricKinds.ts`
        (classification des grandeurs, 195 L), `usageParity.ts` (parité camp/lobby, 20 L),
        `usageGaugeModel.ts` (forme jauge + pile des 3 issues, 261 L),
        `usageLobbyTrackModel.ts` (forme piste du lobby, 86 L),
        `usageRegularityBandModel.ts` (forme bande de régularité, 59 L),
        `usageObjectives.ts` (ordre des rôles, 23 L), `usageAvailability.ts` (états du bloc,
        43 L). Exports publics stables (mêmes noms). `usageI18n.ts` (425→430 L) et
        `usageGrids.ts` (274→275 L) étaient déjà ≤500 : déplacés tels quels, imports internes
        mis à jour. `SessionUsageForms.tsx` renommé `UsageForms.tsx` (retrait du préfixe
        « Session », convention déjà en vigueur dans `_shared/` — `EncounterSplitBars.tsx`,
        `ExperienceDropdown.tsx`). Le test `usageLogic.test.ts` (518 L, déjà hors seuil avant
        ce lot) est scindé à l'identique en 7 fichiers de test miroir + `usageGrids.test.ts` ;
        `SessionUsageForms.test.tsx` renommé `UsageForms.test.tsx`. Tous les fichiers créés ou
        touchés ≤ 500 L (`wc -l` vérifié, cf. gate)
  - [x] E5.1ter Scission `match-replay/model/equipmentUsageLogic.ts` (582 L, seuil 500) —
        sans rapport avec le déménagement ci-dessus (le fichier reste dans
        `match-replay/model/`, aucune réutilisation cross-feature en jeu ici). Nouveau
        fichier voisin `equipmentKeptLogic.ts` (142 L) : `KEPT_FAMILIES`,
        `isEpisodeMeasuredFamily`, `equipmentChangeFamilyOf` et une fonction neuve
        `deriveKeptFromTaken` (encapsule la collecte des prises + la dérivation du gardé,
        avant inline dans `buildEquipmentUsage`). `equipmentUsageLogic.ts` : 582 → 488 L
- [x] E5.2 Vérifié : `npx vitest run src/features/session-detail src/features/match-replay
      src/features/_shared src/components/charts` → 244 fichiers / 3083 tests verts (1 skip
      préexistant), `npx tsc -b --force` propre. Déménagement pur : aucune assertion de test
      n'a changé, seuls les chemins d'import
- [x] E5.3 Garde-rail posé : `apps/web/src/features/_shared/usage/noLocalUsageCopies.guard.test.ts`
      (grep de définitions — `function`/`interface`/`type` — des symboles canoniques du bloc,
      hors du dossier `_shared/usage/`, sur le modèle de `squad/singleCountSource.guard.test.ts`).
      Mordant prouvé par mutation : fichier `features/squad/__mutationTest.ts` créé avec une
      redéfinition de `usageAvailability` → test ROUGE (`AssertionError` listant le fichier) ;
      fichier retiré → test VERT. Mutation non committée

**Périmètre fermé — Go :**

- [x] E5.4 `internal/domain/` — publier le bloc d'usage pour la Synthèse, en types canoniques
- [x] E5.5 `internal/service/` — l'orchestration ; **aucun SQL inline**
- [x] E5.6 `internal/platform/duckdb/` — réutiliser la lecture de la page Sessions ; n'écrire
      une requête neuve qu'après avoir vérifié qu'aucune ne convient
- [x] E5.7 `make openapi-gen` + `make generate-types`

**Périmètre fermé — Web :**

- [x] E5.8 `features/synthesis/` — le bloc, en variante **comptes** (décision P9) : axe en
      objets pris, aucun pourcentage dans les barres, aucun trait de parité, lignes triées
      du plus pris au moins pris. Monté dans `SynthesisOverviewSection`
      (`features/synthesis/SynthesisPage.tsx`), juste après `SynthesisWeaponRangeSection`
- [x] E5.9 Les deux donuts (décision P10, P11) via `components/charts/DonutChart`, avec
      `sliceColors` du §3.3, `centerValue` = le compte du lobby, `centerLabel` = l'unité.
      Primitive étendue d'une prop `arcLabelKind='value'` (justifiée : P11 veut le COMPTE
      brut sur l'arc, pas un %) — jamais un donut écrit à la main
- [x] E5.10 Les deux sous-totaux emboîtés sous la légende, séparés par un filet
- [~] E5.11 Aucune query key neuve : zéro nouvelle requête réseau (le bloc arrive avec la
      réponse existante) — couvert par `queryKeys.synthesis(...)` / `queryKeys.teammates(...)`
      déjà en place
- [x] E5.12 Strings FR **et** EN (15 clés neuves dans `usageI18n.ts`, parité par typage)
- [x] E5.13 Tests front sur la projection (`usageCountsModel.test.ts`,
      `usageEquipmentPartiesModel.test.ts`, `UsageCountsGrid.test.tsx`,
      `UsageEquipmentDonutCard.test.tsx`, `EquipmentUsageSection.test.tsx`) + smoke bout en
      bout (`SynthesisPage.test.tsx`, fixture `equipment_usage`). Pas de couche `analysis`/
      `service` ici : Go déjà livré et testé par le lot E5-Go (E5.4-E5.7)

**Gate** :
```bash
make go-api-test && cd apps/go-api && go test -tags=integration -p 1 ./...
cd apps/web && npx vitest run src/features/synthesis src/features/session-detail src/features/_shared
cd apps/web && npx tsc -b --force && npx eslint src/features --max-warnings=0
```

---

### E6 — Escouade

- [x] E6.1 Go — même publication que E5, agrégée **par joueur suivi** (réutiliser
      `ResolveTrackedSquad`, déjà en place pour le contexte escouade de Sessions)
- [x] E6.2 Web — les deux graphes, **une ligne par coéquipier** au lieu d'une par famille.
      Variante comptes (décision P9). `EquipmentUsageSection` (bloc partagé) prend un
      `mode: 'solo' | 'squad'` — en mode `'squad'` la barre équipement lit `usage.players`
      (au lieu de `usage.families`) ; la barre armes spéciales lit TOUJOURS `usage.players`
      sur les deux pages (aucune ventilation par famille d'arme à ce grain, cf. Découvertes)
- [x] E6.3 Les deux donuts à **quatre parts** : moi, mes amis (une part PAR AMI SUIVI,
      colorée individuellement — §3.3, pas une part « amis » fusionnée), reste de l'équipe,
      eux — avec les deux sous-totaux « mon escouade » et « mon équipe ». Data-driven : les
      parts « amis » et le sous-total « mon escouade » apparaissent dès que
      `tracked_players` est non vide, sur LES DEUX pages (Synthèse peut aussi avoir des amis
      globalement configurés, cf. Go `ResolveScopeFriends` — vérifié sur pièces)
- [x] E6.4 Les couleurs de joueur viennent de `features/squad/colors.ts`
      (`SQUAD_MAIN_PLAYER_TOKEN`, `SQUAD_TEAMMATE_COLOR_TOKENS`), **source unique** : un
      joueur garde sa couleur d'une page à l'autre. Import direct depuis
      `_shared/usage/usageEquipmentPartiesModel.ts` — précédent déjà établi par
      `usageGrids.ts`/`usagePlayerInk` dans ce même dossier
- [~] E6.5 Tests front (`EquipmentUsageSection.test.tsx` mode squad,
      `SquadSynergiesPage.test.tsx` nouveaux tests bloc équipement). **Mounté mais NON
      VÉRIFIABLE en conditions réelles** — écart de contrat consigné aux Découvertes :
      `TeammatesPageResponse` (`/pages/teammates`, le SEUL endpoint que `SquadLayout` fetch
      en production) ne porte pas encore `equipment_usage` ; seul `SquadPageV2Response`
      (`/pages/squad/v2`, non fetché par la page Escouade) le porte côté Go. Le test « bloc
      présent » simule le futur contrat ; le test « bloc absent » documente l'état réel
      actuel (section auto-masquée, honnête). Pas de couche `service` ici : Go déjà livré
      par E6.1

**Gate** :
```bash
make go-api-test && cd apps/go-api && go test -tags=integration -p 1 ./...
cd apps/web && npx vitest run src/features/squad && npx tsc -b --force
cd apps/web && npx eslint src/features/squad --max-warnings=0
```

---

### E6.1bis — Corriger la publication : `equipment_usage` sur Teammates, pas SquadV2

> Lot correctif, périmètre fermé (défaut vérifié sur pièces par le superviseur, cf.
> Découverte E6.2-E6.5 ci-dessus) : E6.1 avait posé le bloc sur `SquadPageV2Response`
> (`GET /pages/squad/v2`), une réponse que la page Escouade réelle ne fetch jamais
> (seul `POST /pages/teammates` l'est). Corrige le tir sans toucher au rendu web.

- [x] E6.1bis.1 **Publication Teammates** : `domain.TeammatesPageResponse.EquipmentUsage`
      (`internal/domain/teammates.go`) + `TeammatesService.WithEquipmentUsage(repo
      port.SessionUsageRepository)` / `loadEquipmentUsage` dans un fichier voisin neuf
      `internal/service/teammates/teammates_service_usage.go` (`TeammatesService` vit
      dans `service/teammates`, `buildEquipmentUsageBlock` vivait dans `service` — cycle
      d'import évité en déplaçant l'orchestration vers la FEUILLE
      `internal/service/squadagg` déjà importée des deux côtés, cf. E6.1bis.3). Scope =
      `filteredMatches` de `GetPage` (période + cascade + sessions déjà appliquées, LA
      MÊME population qu'Options/MatchHistory — jamais l'intersection escouade) ; amis =
      `req.SelectedGamertags` (les coéquipiers SÉLECTIONNÉS, comme le faisait E6.1) ;
      sujet = `playerXUID` (le joueur de la route). Câblé dans
      `internal/api/wire/registry_pages_home.go` (`TeammatesCtx`), gaté par la
      capability `film.usage_summary`, jamais `slug==`. TDD : rouge observé (bloc `nil`
      sur les deux tests) en retirant temporairement l'appel de `GetPage`, puis vert
      après câblage — `internal/service/teammates/teammates_service_usage_test.go`
- [x] E6.1bis.2 **Retrait SquadV2** : champ `EquipmentUsage` de
      `internal/domain/squad_v2.go`, fichiers `squad_service_v2_usage.go` et
      `squad_service_v2_usage_test.go` supprimés (avec `squadUsageScope`,
      `WithEquipmentUsage`, `loadEquipmentUsage` et le champ `sessionUsageRepo` de
      `SquadServiceV2`), câblage retiré de `SquadV2Ctx`
      (`registry_pages_home.go`). Vérifié : `grep -rn "EquipmentUsage"
      internal/domain/squad_v2.go internal/service/squad_service_v2*.go` → 0 résultat
- [x] E6.1bis.3 **Contrat régénéré + alignement web** : `buildEquipmentUsageBlock`/
      `equipmentUsageQuery` déplacés de `internal/service` (package `service`, qui
      importe déjà `internal/service/teammates` depuis `synthesis_service_usage.go` —
      un appel dans l'autre sens aurait été un cycle) vers
      `internal/service/squadagg` (exportés `BuildEquipmentUsageBlock`/
      `EquipmentUsageQuery`), avec alias de compatibilité dans
      `squadagg_reexport.go` (même patron que `BuildSquadHeader`/`IntersectByMatchID` —
      zéro site d'appel existant modifié, `equipment_usage_block_test.go` et
      `synthesis_service_usage.go` inchangés). `make openapi-gen` : `equipment_usage`
      quitte `SquadPageV2Response` et apparaît sur `TeammatesPageResponse` dans
      `api/openapi.yaml` (diff vérifié, 2 lignes déplacées) ; `make generate-types` :
      même mouvement dans `generated.ts`. Web (`apps/web/src/lib/api/types.ts`,
      ~ligne 1439) : commentaire de `TeammatesPageResponse.equipment_usage` mis à jour
      (l'écart de contrat est corrigé, la déclaration manuelle du champ est conservée
      À L'IDENTIQUE — c'est la convention déjà en vigueur pour
      `SynthesisPageResponse.equipment_usage`, une interface à la main plutôt qu'un
      alias direct sur `components['schemas']`). Aucun composant web touché (hors
      périmètre du lot) : `EquipmentUsageSection`/`SquadSynergiesPage.tsx` lisaient
      déjà `pageData.equipment_usage`, le câblage s'active sans eux

**Gate** :
```bash
cd apps/go-api && go build ./... && go vet ./... && go test ./...
cd apps/go-api && go test -tags=integration -p 1 -count=1 ./internal/service/... ./internal/api/...
cd apps/go-api && golangci-lint run --new-from-merge-base=feat/v75 ./...
make openapi-gen && make generate-types && git status --short apps/go-api/api/openapi.yaml apps/web/src/lib/api/generated.ts
cd apps/web && rm -rf node_modules/.tmp && npx tsc -b --force && npx vitest run src/features/squad src/features/_shared
grep -rn "EquipmentUsage" apps/go-api/internal/domain/squad_v2.go apps/go-api/internal/service/squad_service_v2*.go
```
Tous verts / 0 résultat au grep (résultats détaillés au Journal).

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

- **Nouvelle, E3 du 2026-09-09 — TEST D'INTÉGRATION ROUGE, PRÉEXISTANT** :
  `internal/api/wire` / `TestOuvrierReel_ConstruitEtLivre` (tag `integration`) échoue sur
  trois mesures figées d'identité — « 0 vies anonymes, attendu 1 » et le xuid
  `2535458702376288` nommé par le film mais absent des mesures figées. **Prouvé préexistant** :
  le même test, rejoué au point de branche `32821ba86` dans un worktree détaché jetable,
  échoue à l'identique et produit le même artefact à l'octet près. L'attente figée a été
  périmée par le lot de nommage des vies (elle réclame une vie anonyme que le décodeur ne
  produit plus) ; à rapprocher de l'entrée « recuisson du parc au schéma 50 » du
  `REGISTRE_REPORTS`. Personne ne l'avait vu parce que le gate de la vague 2 n'a pas rejoué
  la suite d'intégration après ce lot. **Non instruit** (hors périmètre E3) — mais il faut
  le traiter avant `make gate-push` du §7, sinon la clôture du chantier ne peut pas être
  verte.

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
  - **Suite, E4 du 2026-09-09** : E4 REMPLACE `deployed_<famille>` / `camo_episodes` /
    `overshield_episodes` par `equipment_<famille>` à l'écran, MAIS la consigne de cette
    tâche interdit explicitement de retirer `deployed_*` du contrat ou du code Go dans ce
    lot (E5 tranchera « quand tous les consommateurs seront connus »). La clause « 0 code
    mort » de cette découverte est donc VOLONTAIREMENT non honorée ICI, sur instruction
    explicite reçue — pas un oubli. Web : `equipmentMetrics()` (`usageLogic.ts`) SUPPLANTE
    `deployed_<famille>`/`camo_episodes`/`overshield_episodes` par leur `equipment_<famille>`
    homonyme quand les deux coexistent (dédoublonnage par identité de famille,
    `equipmentBilanFamilyOf`) ; sans équivalent (grappin — `equipmentOutcomeStems` côté Go
    ne le nomme pas), la grandeur GESTE reste seule, rendu inchangé. Sur les données
    mesurées au parc (E0/E3), `deployed[fam] ⇒ bilan[fam]` pour les 6 familles déployables :
    en pratique `deployed_<famille>` ne s'affichera donc JAMAIS sur une session réelle — seul
    reste un chemin de repli structurel (théorique) pour une famille future du manifeste que
    `equipmentOutcomeStems` ne nommerait pas encore. **Consommateurs web restants de
    `deployed_*` après ce lot** : `metricKind`/`METRIC_RANK`/`metricLabel`/
    `USAGE_METRIC_TOKENS` (classification et repli, `usageLogic.ts`) et `deployedFamilyLabel`
    (`usageI18n.ts`) — tous conservés comme chemin de repli, plus aucun consommateur qui
    l'affiche en pratique sur le parc actuel. La cadence (`ValueGrid`) et la bande de
    régularité de l'Équipement basculent EN MÊME TEMPS que la jauge (même liste `metrics`
    unifiée, P3 « un seul graphe ») : la cadence d'un mur, par exemple, devient un débit
    d'OBJETS (utilisé+gardé+lâché) et non plus de POSES — décision d'exécution assumée, dans
    le droit fil de ce que ce chantier mesure, à signaler si elle surprend en revue.

- **Nouvelle, E2 du 2026-09-09** : `npx eslint src/features/match-replay src/components/charts
  --max-warnings=0` échoue sur 9 avertissements dans cinq fichiers (`ReplayExportDialog.tsx`,
  `useReplayVehicles.ts`, `useReplaySound.ts`, `ReplayCanvas.tsx`, `ReplayFeedName.tsx`) —
  `react-hooks/exhaustive-deps`, `react-refresh/only-export-components`,
  `react-hooks/preserve-manual-memoization`. Vérifié PRÉEXISTANT : `git diff
  feat/v75...HEAD --stat` est vide sur les cinq fichiers, aucun n'est touché par ce chantier.
  Dette de lint antérieure — un `--max-warnings=0` sur ces dossiers ne peut donc pas passer
  tel quel tant qu'elle n'est pas résorbée. Non traité (hors périmètre E1/E2).

- **Nouvelle, E4 du 2026-09-09** : `deployedFamilyLabel` (`usageI18n.ts`, préexistant, écrit
  avant ce chantier) classe les familles `deployed_<famille>` sur des ALIAS DE RENDU
  (`'rift'`, `'shroud'`, `'seeker'`, `'field'` — vocabulaire de `PLACEMENT_RENDER` côté vue
  match) au lieu des VRAIES clés que `deployed_<famille>` porte réellement (celles du
  manifeste : `translocator_beacon`, `shroud_screen`, `threat_seeker`, `repair_field` —
  vérifié sur pièces, `replay_labels.toml` et `sessionusage.PlayerRow.DeployedByFamily`,
  clé = `p.Family` du document, jamais l'alias de rendu). Ces quatre `case` ne matchent donc
  JAMAIS en pratique : une session avec un translocateur/écran occultant/traqueur/champ de
  réparation en `deployed_*` retombe toujours sur le repli `metricDeployedFmt(family)`
  (« Équipement translocator_beacon » au lieu de « Translocateur »). Bug préexistant,
  découvert en lisant le code pour bâtir `equipmentFamilyLabel` (E4, qui lui utilise les
  VRAIES clés). Sans conséquence pour E4 (les familles concernées basculent sur
  `equipment_<famille>` dès qu'elles ont une activité mesurée — cf. entrée ci-dessus) ; reste
  un défaut visible sur le SEUL chemin de repli résiduel (famille avec geste mais sans
  bilan). Non traité (hors périmètre E4, `deployedFamilyLabel` n'est pas un fichier que E4
  modifie pour cette raison).

- **Nouvelle, E5.1-E5.3 du 2026-09-09** : `cd apps/web && npx eslint src/features
  --max-warnings=0` (gate de clôture demandé par le lot) échoue sur 28 avertissements dans
  13 fichiers (`useReplayVehicles.ts`, `useReplaySound.ts`, `ReplayCanvas.tsx`,
  `ReplayFeedName.tsx`, `MatchEncountersTable.tsx`, `MatchPositionsHeatmap.tsx`,
  `MatchScoreboard.tsx`, `MediaAudioConfigButton.tsx`, `RelationsTable.tsx`,
  `SquadAssistPairsTable.tsx`, `SquadEchangeDelaiCard.tsx`, `SquadImpactScoreboard.tsx`,
  `SquadSynergyHistoryTable.tsx`) — `react-hooks/exhaustive-deps`,
  `react-hooks/incompatible-library` (React Compiler × TanStack Table),
  `react-refresh/only-export-components`, `react-hooks/set-state-in-effect`. Vérifié
  PRÉEXISTANT : `git status --short apps/web/src/features` ne montre aucun de ces 13
  fichiers parmi les fichiers touchés par ce lot (même méthode que l'entrée E2 du
  2026-09-08 ci-dessus, ratchet `react-hooks/exhaustive-deps` déjà connu). Vérifié à zéro
  sur le périmètre réel du lot : `npx eslint src/features/_shared/usage
  src/features/session-detail/SessionUsageSection.tsx
  src/features/session-detail/SessionUsageSection.gate.test.tsx --max-warnings=0` → 0
  problème. `npm run lint` (sans `--max-warnings=0`, le script réel du dépôt) passe
  (exit 0, 30 warnings dont les 28 ci-dessus, 0 erreur). Dette de lint antérieure au lot,
  non accrue. Non traité (hors périmètre E5.1-E5.3).

- **Nouvelle, scission n°2 du 2026-09-09** : `equipmentUsageColumns.ts:290` commente
  « une famille de `KEPT_FAMILIES` ... cf. `equipmentUsageLogic.ts` » — référence devenue
  inexacte, `KEPT_FAMILIES` vit maintenant dans `equipmentKeptLogic.ts`. Cosmétique
  (commentaire seul, aucun import cassé) ; `equipmentUsageColumns.ts` est explicitement hors
  périmètre de la scission n°2 (« Ne touche à rien d'autre dans match-replay »). Non traité.

- **Nouvelle, scission n°2 du 2026-09-09** : `src/features/match-view/xuidMeta.guard.test.ts`
  a timeout (5000 ms) une fois sur `npx vitest run src/features/match-view` en lot (42
  fichiers), test qui parcourt tout `features/match-view` sur disque. Rejoué seul et rejoué
  en lot une seconde fois : vert les deux fois (906 ms puis suite complète verte). Flake
  temporel de contention disque sous charge parallèle, pas une régression du déplacement
  (`equipmentKeptLogic.ts` n'est pas dans l'arbre scanné par ce garde-rail). Non traité,
  catégorie déjà connue du dépôt (cf. thought_log, vague 2 : « un flake temporel consigne »).

---

- **Nouvelle, E5-Go du 2026-09-09** : la clause `IN (...)` des trois lectures du résumé
  d'usage porte UN PARAMÈTRE PAR MATCH DU SCOPE (`Placeholders(len(matchIDs))`,
  `platform/duckdb/shared_query_helpers.go:49` — aucun découpage par lots dans le paquet).
  C'est déjà le patron de trois lecteurs de la Synthèse (`loadObjectiveStats`,
  `loadWeaponKillRows`, `loadWeaponAccuracy`), mais le scope Synthèse en période « all » est
  le PLUS LARGE du produit : le bloc d'usage y hérite donc d'une requête à plusieurs
  milliers de paramètres. Rien n'a été mesuré, rien n'a été changé — le lot réutilise le
  patron en vigueur plutôt que d'en inventer un second. Non instruit.

- **Nouvelle, E5-Go du 2026-09-09** : `synthesisMatchIDs` centralise une boucle qui existait
  en TROIS exemplaires dans `synthesis_service.go` (plafond de la règle CLAUDE.md n°6 —
  un quatrième l'aurait franchi) ; les trois copies sont migrées, mais AUCUN garde-rail grep
  n'a été posé, contrairement à ce que la règle demande. Justification : le « littéral
  ancien » est un accès de champ légitime (`r.Summary.MatchID`) employé ailleurs pour de
  bonnes raisons, et un ratchet dessus produirait des faux positifs. À trancher si le motif
  réapparaît. Non instruit au-delà de la migration des trois copies.

- **Observation, E5-Go du 2026-09-09** : le test d'intégration `internal/api/wire` /
  `TestOuvrierReel_ConstruitEtLivre`, consigné ROUGE PRÉEXISTANT par E3 ci-dessus, passe
  VERT sur ce worktree (`go test -tags=integration -p 1 -count=1 ./internal/api/...`,
  2026-09-09). Rien n'a été fait pour cela — l'écart tient probablement à l'artefact local
  que le test consomme. L'entrée E3 n'est PAS retirée : elle reste à vérifier sur la machine
  qui l'a vue rouge. Non instruit.

- **BLOQUANTE, E6.2-E6.5 du 2026-09-09 — écart de contrat entre le bloc Go et la page
  Escouade réellement servie en production.** Vérifié sur pièces AVANT de coder (skill
  plan-execution, règle 4).

  Le Go (E6.1) publie `equipment_usage` sur `domain.SquadPageV2Response`
  (`internal/domain/squad_v2.go:48`), la réponse de `GET /pages/squad/v2`
  (`internal/api/handlers/squad_v2.go`). Mais la page Escouade réellement montée
  (`features/squad/SquadLayout.tsx:26,357` → `useTeammates`, `features/squad/queries.ts`)
  appelle `POST /pages/teammates`, dont la réponse est `domain.TeammatesPageResponse`
  (`internal/domain/teammates.go:496`) — un type SÉPARÉ, qui NE PORTE PAS `equipment_usage`
  (`teammates_service.go` n'a aucun `WithEquipmentUsage`, contrairement à
  `synthesis_service_usage.go`). Grep exhaustif fait : AUCUN fichier de `apps/web/src`
  n'appelle `GET /pages/squad/v2` (seul son sous-chemin `/pages/squad/v2/engagement`,
  `features/engagement/queries.ts`, est fetché — par `SquadEngagementSection`, montée sur
  `SquadDynamiquePage`, avec sa PROPRE requête scopée par `match_ids`). Le commentaire du
  handler squad_v2.go le confirme : « vit en parallèle de l'endpoint legacy /pages/squad
  jusqu'à migration complète du frontend » — cette migration n'a jamais eu lieu ;
  `lib/api/types.ts` documente même explicitement que les types du payload V2 riche « ont
  été retirés : plus aucun consommateur côté web ».

  **Conséquence** : sans changement Go, `equipment_usage` ne peut PAS atteindre la page
  Escouade réelle sans soit (a) une nouvelle requête réseau — exclue par le périmètre de ce
  lot (« Aucune nouvelle requête réseau ») et par la logique produit (une requête de plus
  pour un seul bloc, quand `SessionUsageRepo` sert déjà les trois lectures sans filtre
  neuf) —, soit (b) le Go ajoutant `EquipmentUsage` à `TeammatesPageResponse` (hors
  périmètre de ce lot : `apps/go-api` exclu).

  **Décision d'exécution prise, signalée immédiatement (règle plan-execution §3), pas
  contournée en silence** : construire et monter `EquipmentUsageSection` (mode `'squad'`)
  contre `pageData.equipment_usage`, en ajoutant ce champ **optionnel** à
  `TeammatesPageResponse` côté web (`lib/api/types.ts`, commentaire daté expliquant l'écart
  et pointant `service.buildEquipmentUsageBlock`, déjà écrit et prêt à être branché). Tant
  que le Go ne peuple pas ce champ, il vaut `undefined` et `usageAvailability` rend `hidden`
  — la section s'auto-masque, exactement le même contrat que `available:false` pour un
  titre sans capability : **zéro donnée fabriquée, zéro graphe fantôme**, mais aussi
  **zéro régression visible** aujourd'hui sur la page réelle. Le câblage s'activera sans
  toucher au web dès qu'un lot Go ajoutera `TeammatesService.WithEquipmentUsage` (le
  helper `buildEquipmentUsageBlock` et son scope `matchIDsOf(resp.SharedMatches)` existent
  déjà, réutilisés tels quels par E6.1 pour l'assemblage Squad V2 — le brancher sur
  `TeammatesService` est une passe d'orchestration, pas une réécriture).

  **Non traité ici** (hors périmètre déclaré `apps/go-api`) : ajouter
  `WithEquipmentUsage` à `TeammatesService`. C'est le seul geste qui manque pour que ce lot
  s'affiche réellement sur l'Escouade — à planifier en premier dans le prochain lot Go.

- **Nouvelle, E6.3 du 2026-09-09** : la Synthèse (Solo) peut ELLE AUSSI publier une part
  « mes amis » au donut — pas seulement l'Escouade. Vérifié sur pièces :
  `synthesis_service_usage.go:48` passe `FriendGamertags: s.friendGamertags(ctx)`, qui
  résout les amis GLOBALEMENT CONFIGURÉS (`app_settings.json`), pas une sélection
  d'escouade. `buildPartiesDonutModel` est donc purement DATA-DRIVEN
  (`tracked_players.length > 0`), jamais posé sur `mode==='solo'` en dur — sinon la
  Synthèse d'un joueur avec des amis configurés aurait perdu leur part. Non instruit
  au-delà : aucune fixture locale ne l'a exercé, seul le test unitaire
  `usageEquipmentPartiesModel.test.ts` le couvre.

- **Nouvelle, E6.2 du 2026-09-09** : la barre « armes spéciales » n'a JAMAIS d'issues
  (pile utilisé/gardé/lâché) sur Solo ni sur Escouade — ni le mockup validé ni une lecture
  rapide du plan ne le disent, mais le contrat Go (E6.1, `EquipmentUsagePlayerLine`,
  commentaire de `PadPickups`) est explicite : le canal `shots` (« a tiré ») n'est publié
  qu'au grain DOCUMENT, jamais au grain session/période. La barre y rend donc un compte
  simple (aplat, pas de pile), sur les DEUX pages — écart assumé vis-à-vis du mockup (qui
  montrait des lignes PAR ARME avec pile utilisé/jamais-tirée), déjà anticipé et consigné
  par le lot E5-Go (« Ce que le lot ne publie PAS »). Non instruit davantage.

- **Nouvelle, E5.9 du 2026-09-09** : l'axe des graduations de fin de barre
  (`usageCountsModel.ts`, `niceAxisMax`) suit un algorithme standard « nice number »
  (1/2/5/10 × 10ⁿ), PAS les valeurs précises vues sur la maquette artefact (qui semblent
  choisies à l'œil par le mockup, pas produites par une formule déterministe — ex. 138 pris
  → axe à 140 sur la maquette, 200 avec cet algorithme). Aucune décision P1-P14 ne fixe de
  formule d'arrondi d'axe ; ce choix reste un axe rond et lisible, jamais un pixel-perfect
  de la maquette. Non instruit, à ajuster si un retour utilisateur le juge trop lâche.

- **Nouvelle, E6.1bis du 2026-09-09** : le commentaire de
  `apps/web/src/features/squad/SquadSynergiesPage.tsx` (au-dessus du montage de
  `<EquipmentUsageSection ... mode="squad" />`) et celui de
  `SquadSynergiesPage.test.tsx` (près de la fixture `equipment_usage`) décrivent
  encore l'écart de contrat E6.2-E6.5 (« le Go ne l'y publie pas encore ») — devenu
  FAUX après ce lot, `TeammatesPageResponse` porte désormais le champ. Périmètre de
  ce lot EXCLUT explicitement tout composant web (y compris un commentaire seul,
  aucune ligne de rendu) : non corrigé ici. Un lot web pourra le faire en passant
  (aucun garde-rail ni gate ne dépend de ce commentaire).

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
  `go vet ./...` · `go test ./...` **vert (suite complète)** · `go test ./internal/sync/
  -run NoART` vert · `golangci-lint --new-from-merge-base=origin/main` **0 issue** ·
  `make openapi-gen` + `make generate-types` (+32 lignes openapi, +17 generated.ts,
  commitées, aucun diff résiduel) · `npx tsc -b --force` silencieux.

  **`go test -tags=integration -p 1 -count=1 ./...` : UN ÉCHEC, PRÉEXISTANT — prouvé, pas
  supposé.** `internal/api/wire` / `TestOuvrierReel_ConstruitEtLivre` échoue sur trois
  mesures figées d'IDENTITÉ : « 0 vies anonymes, attendu 1 », et le xuid
  `2535458702376288` « nommé par le film mais absent des mesures figées ». Vérification faite
  en rejouant le MÊME test au point de branche `32821ba86`, dans un worktree détaché jetable
  (supprimé depuis — ni ma branche ni mon worktree touchés) : **échec identique, et l'artefact
  produit est le même à l'octet près (291 655 octets, 22 trajectoires, 781 frames)**. Le diff
  de cette étape ne touche AUCUN fichier du chemin de décodage ou de nommage d'identité —
  c'est une attente figée que le lot de nommage par record de création a périmée (elle attend
  une vie anonyme qui n'existe plus). Rapproché de l'entrée « recuisson du parc au schéma 50 »
  du `REGISTRE_REPORTS`. **Non traité : hors périmètre E3** (règle « zéro fix opportuniste »),
  reporté au §6. Tous les autres paquets d'intégration, dont `migration/` et `persist/` — les
  deux que ce lot touche — sont verts.

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

- **2026-09-09 — E4 CLOSE.** Six items `[x]`. Périmètre : `apps/web/src/features/session-detail/`
  (`SessionUsageForms.tsx`, `usageLogic.ts`, `usageI18n.ts` et leurs tests), plus deux fichiers
  hors liste déclarée mais structurellement exigés (même précédent que E2/`MatchEquipmentUsageSection.tsx`) :
  `apps/web/src/lib/api/types.ts` (alias manquant `SessionUsageOutcomes`, sans lequel le contrat
  étendu en E3 n'est pas typable côté web) et `SessionUsageSection.tsx` (une ligne : câbler
  `outcomes: m.outcomes` dans `metricGaugeRows`, sans quoi la projection E4.3 n'atteint jamais
  l'écran).

  **TDD, échecs observés AVANT le code** (dans l'ordre) :
  1. `usageLogic.test.ts` : 10 tests rouges (`equipmentMetrics` ne filtrait pas `dropped_objects`,
     `equipmentBilanFamilyOf`/`equipment` inexistants, `buildGaugeRow` ne portait ni `segments`
     ni `teammatesRatePct`/`opponentsRatePct`, `metricLabel('equipment_*')` non géré).
  2. `SessionUsageForms.test.tsx` : 4 tests rouges (`[data-outcome-key]`/`[data-outcome-ref]`
     absents du DOM, `[data-outcome-fill]` absent du cas sans issues).
  Les deux suites sont passées au vert par l'implémentation qui suit, sans modifier les
  assertions pour les faire passer artificiellement.

  **Décisions de conception prises à l'exécution :**

  1. **`equipmentMetrics()` SUPPLANTE `deployed_<famille>` / `camo_episodes` /
     `overshield_episodes` par leur `equipment_<famille>` homonyme** quand les deux coexistent
     (dédoublonnage par identité de famille via `equipmentBilanFamilyOf`, nouvelle fonction).
     Sans équivalent (grappin — la table Go `equipmentOutcomeStems` ne le nomme pas), la
     grandeur GESTE reste seule, rendu STRICTEMENT INCHANGÉ. Conséquence assumée et consignée
     au §6 (entrée E3 « deployed_<famille> et equipment_<famille> coexistent ») : la CADENCE
     (ValueGrid) et la BANDE DE RÉGULARITÉ de la famille basculent EN MÊME TEMPS que la jauge,
     puisque les trois formes partagent la MÊME liste `metrics` (P3 — un seul graphe). Sur les
     données mesurées (E0/E3), toute famille `deployed_*` avec activité a TOUJOURS son
     `equipment_*` homonyme (`deployed[fam] ⇒ bilan[fam]`) : en pratique `deployed_*` ne
     s'affiche donc plus jamais sur une session réelle du parc — reste un repli théorique pour
     une famille future non encore nommée par `equipmentOutcomeStems`. `deployed_*` N'A PAS été
     retiré du contrat Go ni du code Go, sur instruction explicite de cette tâche (le plan, à
     l'entrée §6 correspondante, disait le contraire — « si E4 la remplace, retirer `deployed_*`
     du contrat » — la consigne reçue prévaut, E5 tranchera avec la vue d'ensemble des
     consommateurs).
  2. **`SessionUsageOutcomes` ne s'attache qu'aux DEUX jauges de PART DU JOUEUR** (`player-of-team`,
     `player-of-lobby`), jamais à `team-of-lobby` : les trois issues sont une grandeur du JOUEUR
     (domaine Go : « les trois issues du JOUEUR »), quand la première jauge mesure la part de
     MON ÉQUIPE dans le lobby — une population sans porteur individuel. `buildGaugeRow` prend un
     paramètre `withOutcomes` par sous-jauge plutôt qu'un seul indicateur au niveau de la ligne.
  3. **Les deux repères de taux sont positionnés en POURCENTAGE DE LA TRANCHE**, pas du rail
     entier : `teammatesRatePct`/`opponentsRatePct` sont posés tels quels par `buildGaugeRow`
     (aucun calcul composé `valuePct × ratePct`), et `SessionUsageForms.tsx` les rend `left: X%`
     À L'INTÉRIEUR du conteneur déjà large de `valuePct` — le même dénominateur que les segments
     de la pile, sans arithmétique dans le composant.
  4. **Aucun aria-label individuel sur les segments/repères** (`aria-hidden="true"` sur tout le
     conteneur `UsageOutcomeStack`) : le texte qui compte (comptes bruts + taux) est fondu dans
     le `tooltip` COMBINÉ du rail entier (`gaugeOutcomeTipFmt`/`gaugeReferenceTipFmt`, nouveaux
     formatteurs FR/EN) — même contrat d'accessibilité que le trait de parité existant, qui n'a
     jamais eu sa propre aria-label.
  5. **`equipmentFamilyLabel` (nouvelle fonction, `usageI18n.ts`) utilise les VRAIES clés de
     famille du bilan** (`translocator_beacon`, `shroud_screen`, `threat_seeker`, `repair_field`,
     vocabulaire Go de `equipmentOutcomeStems`), PAS les alias de rendu (`rift`/`shroud`/
     `seeker`/`field`) que `deployedFamilyLabel` utilise déjà et qui ne matchent JAMAIS les
     vraies clés de `deployed_<famille>` — bug préexistant découvert en écrivant cette fonction,
     consigné au §6, non corrigé (hors périmètre E4, fichier/fonction différents). Nouvelle clé
     i18n `equipTranslocator` (« Translocateur » / « Translocator ») : le seul label qui
     n'existait sous AUCUNE forme correcte côté dictionnaire (les trois autres réutilisent
     `equipShroud`/`equipSeeker`/`equipField`, dont le TEXTE était déjà juste).
  6. **`metricLabel`/`t.metricDropped` ne sont PAS supprimés** malgré que `dropped_objects` ne
     soit plus jamais affiché (E4.5) : le contrat Go publie toujours cette clé inconditionnellement
     (une des « cinq grandeurs fixes », `sessionusage.metricKeys`), donc `metricKind`/`metricLabel`
     doivent rester capables de la classer sans planter — c'est le FILTRE d'affichage
     (`equipmentMetrics`) qui l'exclut de la liste rendue, pas la classification elle-même. Ce
     n'est pas du code mort au sens CLAUDE.md n°7 (rien n'est débranché du routing : la fonction
     reste un classifieur exhaustif d'un contrat encore émis tel quel).

  **Gates, tous exécutés sur l'arbre final** :
  - `cd apps/web && rm -rf node_modules/.tmp && npx vitest run src/features/session-detail` —
    **19 fichiers, 146 tests, vert** (dont les 14 tests neufs E4 + 2 tests d'intégration
    ajoutés au gate `SessionUsageSection.gate.test.tsx`, hors liste déclarée mais nécessaires
    pour prouver le câblage bout en bout de E4.3/E4.5).
  - `npx tsc -b --force` — silencieux, code de sortie 0.
  - `npx eslint src/features/session-detail --max-warnings=0` — silencieux, code de sortie 0.
  - Les deux greps couleur du §3.4 (`grep hex` / `grep tailwind couleur`) sur
    `apps/web/src/features/` et `apps/web/src/components/` : les fichiers touchés par E4
    (session-detail, types.ts) ne portent AUCUNE ligne nouvelle — les 29/25 correspondances
    trouvées globalement sont toutes hors périmètre E4, vérifiées PRÉEXISTANTES (aucune dans
    les fichiers du diff de cette étape).

  **Consommateurs web restants de `deployed_*` après ce lot** (question posée par la tâche) :
  `metricKind`, `METRIC_RANK`, `metricLabel`, `USAGE_METRIC_TOKENS['deployed_other'|'wall']`
  (classification/repli, `usageLogic.ts`) et `deployedFamilyLabel` (`usageI18n.ts`) — tous
  conservés comme chemin de repli pour une famille sans `equipment_<famille>` homonyme. Plus
  aucun consommateur ne l'AFFICHE en pratique sur le parc mesuré (E0/E3) : chaque famille
  déployable avec activité a désormais son homonyme `equipment_<famille>`, qui la supplante.

  **Prochaine étape** : E5 (bloc partagé, Solo/Synthèse) — décidera, avec la vue d'ensemble
  des consommateurs ci-dessus, si `deployed_*` sort du contrat Go.

- **2026-09-09 — E5.1/E5.2/E5.3, extraction du bloc partagé (worktree
  `LevelUp-wt-equipement-gachis`, branche `feat/equipement-gachis`)**. Périmètre : web
  uniquement (le Go de E5.4-E5.13 est mené en parallèle par un autre exécutant, worktree
  `LevelUp-wt-equipement-e5-go` — non touché ici, ni `apps/go-api/`, ni `openapi.yaml`, ni
  `generated.ts`).

  **Décision technique — ordre des deux scissions.** Le contrat de la tâche liste l'ordre
  « E5.1 → E5.2 → E5.3 → scission 1 → scission 2 » tout en demandant de statuer la scission
  de `usageLogic.ts` comme sous-item **de E5.1** (« E5.1bis »). Les deux ne sont conciliables
  qu'en faisant la scission de `usageLogic.ts` **PENDANT** E5.1 : un fichier de 680 lignes ne
  peut pas être déplacé tel quel dans `_shared/usage/` (seuil 500 L) puis scindé après coup
  sans changer une deuxième fois tous les chemins d'import des consommateurs déjà mis à jour
  — double churn, deux passes de risque au lieu d'une. La scission n°2
  (`equipmentUsageLogic.ts`, `match-replay/model/`) n'a, elle, AUCUN rapport avec le
  déménagement : elle reste dans son dossier d'origine et n'est réutilisée par personne de
  neuf. Elle est donc traitée en dernier, conformément à l'ordre littéral du contrat.

  **E5.1 — déplacement, avec scission intégrée (E5.1bis).** Le bloc « usages d'équipement,
  armes spéciales et objectifs » quitte `features/session-detail/` pour
  `apps/web/src/features/_shared/usage/`. Vérifié sur pièces AVANT de coder : la regex de
  nom de feature du ratchet `lint-cross-feature-imports`
  (`tools/lint-cross-feature-imports.mjs`, `FEATURE_IMPORT_RE = /@\/features\/([a-z0-9-]+).../`)
  ne matche PAS `_shared` (le `_` n'est pas dans la classe de caractères) — ni comme feature
  consommatrice (`getFeatureNameFromPath` rend `null`, la boucle passe), ni comme feature
  importée (le `match` de l'import échoue). `_shared/` est donc structurellement HORS SCAN du
  ratchet, précédent déjà exploité par `firstBlood.ts` / `EncounterSplitBars.tsx` (consommés
  par 9 features sans une seule entrée `ALLOWED_CROSS_IMPORTS`). Aucune entrée n'a donc été
  ajoutée au ratchet.

  `usageLogic.ts` (680 L, 507 avant E4, seuil 500) est scindé par responsabilité (pas au
  hasard, suivant ses propres sections déjà commentées dans le fichier source) en 8 fichiers :
  `usageFormat.ts` (56 L, formatage), `usageMetricKinds.ts` (195 L, classification des
  grandeurs), `usageParity.ts` (20 L), `usageGaugeModel.ts` (261 L, forme jauge + pile des
  3 issues — le plus gros morceau), `usageLobbyTrackModel.ts` (86 L),
  `usageRegularityBandModel.ts` (59 L), `usageObjectives.ts` (23 L),
  `usageAvailability.ts` (43 L). Tous les exports publics (noms de fonctions/types) sont
  restés IDENTIQUES — seuls les chemins d'import changent. `usageI18n.ts` (425→430 L après
  ajout d'un en-tête de déménagement) et `usageGrids.ts` (274→275 L) étaient déjà ≤ 500 L :
  déplacés sans réécriture, seuls leurs imports internes pointent maintenant vers les
  nouveaux fichiers scindés. `SessionUsageForms.tsx` (400 L) est renommé `UsageForms.tsx` en
  y arrivant (retrait du préfixe « Session », le bloc n'étant plus propre à cette page —
  convention déjà en vigueur pour les autres fichiers de `_shared/`, aucun composant partagé
  n'y porte de préfixe de feature).

  `features/session-detail/SessionUsageSection.tsx` (page-orchestrateur, reste dans
  `session-detail/` — il n'est PAS générique, il porte la mise en page des trois cartes
  propres à la page Sessions) importe désormais tout depuis `@/features/_shared/usage/...`
  au lieu de porter le bloc. `SessionUsageSection.gate.test.tsx` idem pour `USAGE_TEXT`.

  Le test `usageLogic.test.ts` (518 L, déjà hors seuil AVANT ce lot — dette gelée non
  accrue) est scindé à l'identique en miroir des 8 fichiers de logique + un
  `usageGrids.test.ts` neuf (tests de grilles, jusqu'ici mêlés dans le même fichier) :
  `usageFormat.test.ts`, `usageParity.test.ts`, `usageMetricKinds.test.ts`,
  `usageGaugeModel.test.ts`, `usageLobbyTrackModel.test.ts`,
  `usageRegularityBandModel.test.ts`, `usageAvailability.test.ts`, `usageGrids.test.ts`.
  Chaque `it(...)` a été déplacé tel quel, AUCUNE assertion n'a changé. `SessionUsageForms.test.tsx`
  renommé `UsageForms.test.tsx` (mêmes assertions).

  **E5.2 — preuve que rien ne casse.** `cd apps/web && rm -rf node_modules/.tmp && npx vitest
  run src/features/session-detail src/features/match-replay src/features/_shared
  src/components/charts` → **244 fichiers de test, 3083 tests, verts (1 skip préexistant,
  inchangé)**. `npx tsc -b --force` → silencieux, code 0.

  **E5.3 — garde-rail.** `apps/web/src/features/_shared/usage/noLocalUsageCopies.guard.test.ts`,
  sur le modèle de `features/squad/singleCountSource.guard.test.ts` : parcourt
  `apps/web/src/features/**/*.{ts,tsx}` (dossier canonique `_shared/usage/` exclu du scan) et
  fait échouer le test si une DÉFINITION (`function`/`interface`/`type`, pas un import) d'un
  échantillon de 9 identifiants canoniques du bloc (`buildGaugeRow`, `buildOutcomeSegments`,
  `buildLobbyTrack`, `buildRegularityBand`, `usageAvailability`, `equipmentMetrics`,
  `UsageGaugeModel`, `UsageGaugeRowModel`, `UsageOutcomeKind`) apparaît ailleurs. **Mordant
  prouvé par mutation** : création de `features/squad/__mutationTest.ts` avec une redéfinition
  de `usageAvailability` → `npx vitest run
  src/features/_shared/usage/noLocalUsageCopies.guard.test.ts` **ROUGE**
  (`AssertionError: ... src/features\squad\__mutationTest.ts`) ; fichier supprimé → **VERT**.
  La mutation n'a jamais été committée.

  **Gates de clôture exécutés** :
  - `npx vitest run src/features/session-detail src/features/match-replay src/features/_shared
    src/components/charts` → vert (cf. E5.2) ;
  - `npx tsc -b --force` → silencieux, code 0 ;
  - `npx eslint src/features --max-warnings=0` → 28 avertissements PRÉEXISTANTS dans 13
    fichiers hors périmètre (aucun touché par ce lot — cf. §6 Découvertes) ; scope réel du
    lot (`_shared/usage` + les 2 fichiers modifiés de `session-detail`) → **0 problème** ;
  - `npm run lint` (script réel du dépôt, sans `--max-warnings=0`) → exit 0, 30 warnings
    (les 28 ci-dessus + 2 fixables), 0 erreur ;
  - `node tools/lint-cross-feature-imports.mjs` → **7 ≤ 7** (plafond ratchet inchangé, aucune
    violation neuve — cf. E5.1) ;
  - `wc -l` de tous les fichiers créés/touchés → tous ≤ 500 L (max observé : 463 L,
    `SessionUsageSection.tsx`, inchangé par rapport à avant ce lot) ;
  - `grep hex` / `grep tailwind couleur` sur les fichiers créés/touchés → 0 résultat.

  **Statut des items** : E5.1 `[x]`, E5.1bis `[x]`, E5.2 `[x]`, E5.3 `[x]`. E5.1ter
  (scission `equipmentUsageLogic.ts`) reste `[ ]` à ce point du journal — traitée dans
  l'entrée suivante. E5.4-E5.13 (Go, Synthèse/Escouade) : hors périmètre de cette session,
  menés par un autre exécutant.

  **Prochaine étape** : scission n°2 (`equipmentUsageLogic.ts`, match-replay), puis clôture
  de ce lot (aucun push, aucune fusion — décision superviseur).

- **2026-09-09 — scission obligatoire n°2, `equipmentUsageLogic.ts` (match-replay)**. Sans
  rapport avec le déménagement E5.1 (ce fichier reste dans `match-replay/model/`, aucune
  réutilisation cross-feature). Nouveau fichier voisin `equipmentKeptLogic.ts` (142 L) : la
  TROISIÈME ISSUE et sa reconnaissance rang -> famille — `KEPT_FAMILIES`,
  `isEpisodeMeasuredFamily`, `equipmentChangeFamilyOf` (déplacés tels quels, exports
  stables) et une fonction NEUVE `deriveKeptFromTaken(doc, tallyOfSlotAt)` qui encapsule les
  deux boucles autrefois inline dans `buildEquipmentUsage` (collecte des prises par famille
  canonique, puis dérivation `taken - utilisé - lâché`). `equipmentUsageLogic.ts` :
  582 → 488 L. Import type-only de `EquipmentUsageTally` depuis `equipmentUsageLogic.ts`
  vers `equipmentKeptLogic.ts` (érasé à la compilation, aucun cycle runtime).
  `equipmentUsageLogic.kept.test.ts` : `equipmentChangeFamilyOf` importé depuis
  `./equipmentKeptLogic`, `buildEquipmentUsage` reste depuis `./equipmentUsageLogic` — même
  fichier de test, aucune assertion changée. Aucun autre consommateur externe de
  `equipmentChangeFamilyOf`/`KEPT_FAMILIES` (vérifié par grep global avant de coder) ;
  `EPISODE_FAMILIES`/`buildEquipmentUsage`/`tallyTotal` (consommés par `match-view` via
  l'entrée `ALLOWED_CROSS_IMPORTS` nommée `match-view=>match-replay/model/equipmentUsageLogic`)
  restent dans le fichier principal, donc cette entrée n'a pas bougé.

  **Résultats observés** : `npx vitest run src/features/match-replay` → 179 fichiers / 2596
  tests verts (1 skip préexistant). `npx vitest run src/features/match-view` → 1 flake
  temporel non reproductible (`xuidMeta.guard.test.ts`, timeout 5000 ms sous charge
  parallèle), vert au rejeu isolé ET au rejeu du lot complet (410/410) — consigné en
  Découvertes, non traité. `npx tsc -b --force` silencieux. `npx eslint
  src/features/match-replay/model/equipmentUsageLogic.ts
  src/features/match-replay/model/equipmentKeptLogic.ts
  src/features/match-replay/model/equipmentUsageLogic.kept.test.ts --max-warnings=0` → exit
  0. `node tools/lint-cross-feature-imports.mjs` → 7 ≤ 7, inchangé. `wc -l` : 488 / 142 / 137,
  tous ≤ 500. Greps couleur (hex/tailwind) sur les deux fichiers de logique : 0 résultat.

  **Statut** : E5.1ter `[x]`. Toutes les cases du périmètre de cette session (E5.1, E5.1bis,
  E5.1ter, E5.2, E5.3) sont maintenant `[x]`.

  **Conclusion** : chantier E5.1-E5.3 + les deux scissions obligatoires terminé et vérifié
  sur pièces. Pas de push, pas de fusion (décision superviseur — un autre exécutant mène en
  parallèle E5.4-E5.13, Go + Synthèse/Escouade, dans un worktree différent).
- **2026-09-09 — E5 (partie Go : E5.4 à E5.7) et E6.1, CLOS**. Branche
  `feat/equipement-e5-go` (worktree dédié `LevelUp-wt-equipement-e5-go`), quatre commits.

  **Ce qui est publié.** Un type à part, `domain.EquipmentUsageBlock`
  (`internal/domain/equipment_usage.go`), attaché aux DEUX réponses de page existantes —
  `SynthesisPageV2Response.equipment_usage` et `SquadPageV2Response.equipment_usage`, jamais
  un endpoint dédié (même patron que le bloc de la page Sessions). Il porte : `families[]`
  (une ligne par famille du bilan que le JOUEUR a touchée, triée du plus pris au moins pris,
  chacune embarquant `SessionUsageOutcomes` — trois issues, prises, taux d'utilisation, et
  les deux taux de référence qui excluent le sujet) ; `players[]` (une ligne par sujet,
  TOUTES familles confondues, le joueur de la route en tête puis les coéquipiers suivis, avec
  en plus `pad_pickups`) ; `tracked_players[]` (les identités, ordre d'affichage) ; et
  `equipment_parties` / `weapon_pad_parties`, les CINQ comptes exclusifs de chacun des deux
  donuts (`lobby_total`, `player`, `friends`, `rest_of_team`, `opponents`, plus
  `by_friend[]`). **Aucun pourcentage de part n'est calculé côté Go** (décisions P10/P11) :
  les quatre parts font exactement le lobby, le front fera les arcs et les deux sous-totaux.

  **Pourquoi un type à part et non `SessionUsageBlock`** : le point par match, les cadences
  par dix minutes et les deux parités du bloc de session n'ont aucun lecteur sur un scope de
  période, et sur une Synthèse « all » le seul `per_match` pèserait plus que tout le reste de
  la réponse.

  **Deux scopes, assumé et documenté** : les lignes (familles, joueurs) portent sur TOUT le
  scope mesuré, comme `PlayerTotal` côté session ; les deux donuts portent sur les seuls
  matchs à camp CONNU, numérateurs ET dénominateurs (règle de scope de `computeMetric`).
  C'est la seule façon d'avoir des parts qui ferment : mêler les deux ferait un donut dont la
  somme des parts ne vaudrait pas son centre. Scope sans aucun camp connu (FFA intégral) :
  les deux donuts sont ABSENTS, jamais servis à zéro.

  **Réutilisations, et ce qui a été factorisé plutôt que recopié.**
  1. `computeOutcomes(sujet, familles, mesurés)` extrait de `attachOutcomes`
     (`sessionusage/usage_outcomes.go`) : SOURCE UNIQUE du remplissage de barre, appelée par
     la page Sessions (une famille, le joueur de la route) et par le bloc de période (une
     famille pour la Synthèse, toutes familles pour l'Escouade). Le sujet devient un
     paramètre — sur une ligne de coéquipier, « le reste de mon équipe » est mon camp moins
     CE coéquipier (décision P7 appliquée à chaque ligne, pas seulement à la mienne).
  2. `sessionusage.BuildMatchInputs` extrait de `buildSessionUsageInput` : l'assemblage
     commun des `MatchInput` ; le builder de session n'ajoute plus que l'échelle de temps des
     cadences, qui lui est propre.
  3. `measuredMatches` extrait de `ComputeUsage`.
  4. `synthesisMatchIDs` : la boucle « rows -> match_ids » existait en TROIS exemplaires dans
     `synthesis_service.go` — plafond de la règle CLAUDE.md n°6 ; les trois sont migrées, un
     quatrième exemplaire l'aurait franchi (garde-rail non posé — justification au §6).
  5. `buildEquipmentUsageBlock` (`service/equipment_usage_block.go`) : UN seul assemblage
     pour les deux pages — un helper de package, jamais un service qui en appelle un autre.

  **E5.6 — aucune requête neuve, et c'est prouvé sur pièces.** Les trois lectures de la page
  Sessions (`duckdb.SessionUsageRepo` : `LoadUsageFilms`, `LoadUsagePlayers`,
  `LoadParticipants`) prennent DÉJÀ un scope FERMÉ de `match_id` — leurs trois requêtes sont
  un `SELECT ... WHERE match_id IN (...)` sans aucun filtre de session, temporel ou autre
  (l'en-tête du fichier le dit explicitement : « TROIS LECTURES, TOUTES SUR UN SCOPE FERMÉ DE
  match_id (aucun filtre temporel) »). Seule la liste d'identifiants change d'une page à
  l'autre. `internal/platform/duckdb/` n'a donc reçu AUCUNE ligne dans ce lot. Vues `_latest`
  uniquement, lecture seule, aucune migration.

  **Résolution des amis — une décision prise à l'exécution, à signaler.**
  `ResolveTrackedSquad` (grain session) exige d'être allié dans TOUS les matchs du scope. Sur
  un scope de période (Synthèse : des mois ; Escouade : tous les matchs partagés), cette
  intersection rend toujours vide, et la part « mes amis » du donut n'existerait jamais. J'ai
  donc ajouté son jumeau `ResolveScopeFriends` (`sessionusage/squad.go`, bâti sur la MÊME
  machine `alliesOf`, documenté juste sous lui) : allié dans AU MOINS UN match, classé par
  matchs partagés décroissants, plafonné à `MaxTrackedSquadPlayers`. L'attribution du bloc de
  période se faisant ligne à ligne (match, joueur), l'union garde des parts exclusives et
  exhaustives sans exiger la présence continue. Deuxième écart assumé : **sans ami configuré,
  aucun ami** — l'inverse de la convention de `ResolveTrackedSquad` (« liste vide = aucune
  restriction »), parce que retenir les trois alliés les plus fréquents d'une file d'attente
  nommerait « mes amis » des inconnus.

  **E6.1 — l'Escouade.** Même bloc, même assemblage, même repo ; le scope est
  `matchIDsOf(resp.SharedMatches)` et les « amis » sont les coéquipiers SÉLECTIONNÉS dans
  l'UI (`teammateGTs`), passés au résolveur qui n'invente donc personne : il fait la jointure
  gamertag -> xuid contre `match_participants`. Effet de bord voulu : un coéquipier
  sélectionné qui se trouve en FACE sur les matchs partagés compte du côté « eux » (le camp
  prime sur l'amitié dans le classement d'une ligne) — l'intersection de la page porte sur
  les `match_id`, jamais sur le camp, et sans cette priorité « eux » cesserait d'être « le
  lobby moins mon camp ».

  **Ce que le lot ne publie PAS, et pourquoi.** Aucune issue pour les armes spéciales : au
  grain session le canal `shots` (« a tiré », décision P5) n'est pas persisté — seul
  `pad_pickups` l'est. Le contrat sert donc un COMPTE de prises de socle par joueur, sans
  remplissage, et le dit dans le type. Aucune ventilation des socles PAR FAMILLE D'ARME non
  plus : le donut des armes n'en a pas besoin (ses parts sont des joueurs) et la page
  Escouade lit une ligne par coéquipier, pas par famille — publier des clés hexadécimales
  sans le catalogue d'armes du titre n'aurait servi personne. Les clés `deployed_*` sont
  restées au contrat, conformément à la décision superviseur.

  **Gates, tous exécutés sur l'arbre final, codes de sortie vérifiés** :
  - `go build ./...` + `go vet ./...` + `go test ./...` — VERT (aucun `--- FAIL:`).
  - `go test -tags=integration -p 1 -count=1 ./internal/platform/duckdb/ ./internal/service/ ./internal/api/...`
    — VERT (duckdb 160 s, service 19 s, api 24 s, handlers 18 s, wire 16 s).
  - `golangci-lint run --new-from-merge-base=feat/v75 ./...` — **0 issues**.
  - `make openapi-gen && make generate-types` rejoués sur l'arbre final : `git status --short`
    sur `openapi.yaml` et `generated.ts` VIDE (aucun diff résiduel).
  - `cd apps/web && npx tsc -b --force` — silencieux, sortie 0.
  - `grep -rn 'slug == '` sur les fichiers du lot : **0**.

  **Prochaine étape** : la partie web de E5 (E5.1-E5.3 extraction du bloc partagé,
  E5.8-E5.13 Synthèse) et E6.2-E6.5 (Escouade), qui consomment `equipment_usage` tel que
  `generated.ts` le décrit désormais.

- **2026-09-09 — E5.8-E5.13 (Synthèse) et E6.2-E6.5 (Escouade), CLOS côté web** (worktree
  `LevelUp-wt-equipement-gachis`, branche `feat/equipement-gachis`, HEAD au commit
  `4e1f9b5f0` de `feat/v75` — E5.1-E5.7/E6.1 déjà mergés).

  **Le bloc partagé ajouté à `features/_shared/usage/`** (variante COMPTES, P9) :
  - `usageCountsModel.ts` — `buildCountsGrid` : une ligne par grandeur, `valuePct` relatif
    au MAXIMUM DE L'AXE (jamais une part d'équipe), `parityPct` toujours `null`, tri
    descendant par défaut. RÉUTILISE `buildOutcomeSegments` (exporté depuis
    `usageGaugeModel.ts` pour l'occasion) — même pile utilisé/lâché/gardé que Sessions,
    zéro seconde définition.
  - `usageEquipmentPartiesModel.ts` — `buildPartiesDonutModel` : les cinq comptes Go
    (`EquipmentUsageParties`) en parts EXCLUSIVES contiguës (moi → mes amis → reste de mon
    équipe → eux), légende COULEUR SEULE (P11), deux sous-totaux en pourcentage. `null`
    quand `parties` est absent ou `lobby_total<=0` (donut masqué, jamais un anneau à zéro).
    Couleurs de joueur importées DIRECTEMENT depuis `features/squad/colors.ts` — précédent
    déjà établi dans ce dossier par `usageGrids.ts`/`usagePlayerInk` (vérifié sur pièces
    AVANT de coder : `_shared/` est hors scan du ratchet `lint-cross-feature-imports`,
    aucune entrée `ALLOWED_CROSS_IMPORTS` nécessaire).
  - `UsageCountsGrid.tsx` — le rendu de la barre, RÉUTILISE `UsageGauge` (exportée depuis
    `UsageForms.tsx` pour l'occasion, avec `LABEL_WIDTH`/`GAUGE_MIN`/`COLUMN_GAP`) : seule
    différence avec Sessions, le sens de `valuePct` et l'absence de trait de parité —
    aucune seconde cellule.
  - `UsageEquipmentDonutCard.tsx` — le donut (`components/charts/DonutChart`, jamais écrit
    à la main) + légende + sous-totaux.
  - `EquipmentUsageSection.tsx` — l'orchestrateur : deux `SectionCard` (« Usages
    d'équipement », « Contrôle des armes spéciales »), un `mode: 'solo' | 'squad'` qui ne
    change QUE la base de la barre équipement (`families[]` vs `players[]`) — la barre
    armes spéciales est TOUJOURS `players[]` sur les deux pages (aucune ventilation par
    famille d'arme à ce grain, cf. Découvertes E6.2). `usageAvailability` réutilisé tel
    quel (élargi en `UsageAvailabilityLike`, structurel, pour accepter
    `EquipmentUsageBlock` en plus de `SessionUsageBlock` — CLAUDE.md n°6).
  - `usageI18n.ts` : 15 clés neuves (FR+EN, parité par typage) — vues, aides de carte,
    formats de compte/axe, libellés du donut. Aucune string en dur dans les composants.
  - `components/charts/DonutChart.tsx` : nouvelle prop `arcLabelKind: 'percent' | 'value'`
    (défaut `'percent'`, comportement historique inchangé) + `ChartPointDonut.valueLabel?` —
    justifiée par P11 (« les valeurs sont sur les arcs », en COMPTE brut, jamais un %).
    Primitive étendue, pas de donut réécrit à la main.

  **Où le bloc est monté** :
  - Synthèse : `features/synthesis/SynthesisPage.tsx`, dans `SynthesisOverviewSection`,
    juste après `SynthesisWeaponRangeSection` (`mode="solo"`, prop `equipmentUsage`
    ajoutée à `SynthesisOverviewSectionProps`, câblée depuis `data.equipment_usage`).
    Type `equipment_usage?: EquipmentUsageBlock` ajouté à `SynthesisPageResponse`
    (`lib/api/types.ts`) — champ déjà publié par `SynthesisPageV2Response` (Go, E5.5).
  - Escouade : `features/squad/SquadSynergiesPage.tsx`, juste après `<MedalDigest>`
    (`mode="squad"`, lit `pageData.equipment_usage`).

  **BLOQUANT SIGNALÉ IMMÉDIATEMENT, pas contourné en silence** (détail complet aux
  Découvertes ci-dessus) : la page Escouade réelle (`SquadLayout` → `useTeammates` →
  `POST /pages/teammates` → `TeammatesPageResponse`) n'est PAS l'endpoint sur lequel le Go
  a publié `equipment_usage` (`SquadPageV2Response`, `/pages/squad/v2`, que le web ne
  fetch nulle part sauf son sous-chemin `/engagement`). Décision prise et signalée :
  déclarer `equipment_usage?: EquipmentUsageBlock` en OPTIONNEL sur `TeammatesPageResponse`
  (web seulement, commentaire daté), monter la section normalement — elle s'auto-masque
  aujourd'hui (champ `undefined`, même contrat que `available:false`), et s'activera sans
  toucher au web dès qu'un lot Go ajoutera `TeammatesService.WithEquipmentUsage`. E6.5
  statué `[~]` pour cette raison : le test « bloc présent » simule le contrat futur, le
  test « bloc absent » documente l'état réel actuel.

  **TDD, rouge puis vert, dans l'ordre** : `usageCountsModel.test.ts` (9 tests),
  `DonutChart.test.ts` (3 tests neufs sur `arcLabelKind`), `usageEquipmentPartiesModel.test.ts`
  (6 tests), `UsageCountsGrid.test.tsx` (3 tests), `UsageEquipmentDonutCard.test.tsx`
  (2 tests), `EquipmentUsageSection.test.tsx` (6 tests, modes solo ET squad) — chaque
  fichier a échoué à l'import (module inexistant) avant l'implémentation. Puis smoke bout
  en bout : `SynthesisPage.test.tsx` (fixture `equipment_usage` ajoutée à
  `test/handlers.ts`), `SquadSynergiesPage.test.tsx` (2 tests neufs, bloc présent/absent).

  **Gates, tous exécutés sur l'arbre final** :
  - `rm -rf node_modules/.tmp && npx vitest run src/features/synthesis src/features/squad
    src/features/_shared src/components/charts && npx tsc -b --force` — **119 fichiers,
    1009 tests, 14 skips préexistants, vert** ; tsc silencieux (code 0).
  - `npx vitest run` (suite complète) — **686 fichiers (1 skip), 7214 tests (17 skips),
    vert** — aucune régression du fixture `equipment_usage` ajouté à `test/handlers.ts`
    (partagé par toutes les pages Synthèse).
  - `npx eslint src/features/synthesis src/features/squad src/features/_shared
    --max-warnings=0` — **5 avertissements PRÉEXISTANTS**, tous dans 4 fichiers de
    `squad/` (`SquadAssistPairsTable.tsx`, `SquadEchangeDelaiCard.tsx`,
    `SquadImpactScoreboard.tsx`, `SquadSynergyHistoryTable.tsx`) — vérifié `git status
    --short` : AUCUN des quatre n'est dans le diff de ce lot. Sous-ensemble exact des 28
    warnings déjà consignés par le lot E5.1-E5.3 (même liste de fichiers).
  - `npx eslint src/features --max-warnings=0` — **28 warnings, identique à la baseline**
    (AVANT et APRÈS ce lot), 0 dans les fichiers touchés/créés par ce lot.
  - `npm run lint` (script réel du dépôt) — **exit 0**, 30 warnings (28 ci-dessus + 2
    préexistants ailleurs, déjà fixables).
  - `node tools/lint-cross-feature-imports.mjs` — **7 ≤ 7**, plafond inchangé, aucune
    entrée `ALLOWED_CROSS_IMPORTS` ajoutée.
  - `wc -l` de tous les fichiers créés/touchés — tous ≤ 500 L, à l'exception de
    `SynthesisPage.tsx` (865 L) et `lib/api/types.ts` (3177 L), DEUX fichiers déjà exemptés
    par un `eslint-disable max-lines` daté et justifié AVANT ce lot (2026-09-06) — cette
    session n'y a ajouté que quelques lignes de câblage, sans agrandir la dette.
  - Greps couleur (hex / Tailwind) sur tous les fichiers créés/touchés — **0 résultat**.

  **Statut des dix items** : E5.8 `[x]`, E5.9 `[x]`, E5.10 `[x]`, E5.11 `[~]` (zéro query
  key neuve — zéro requête neuve), E5.12 `[x]`, E5.13 `[x]`, E6.2 `[x]`, E6.3 `[x]`,
  E6.4 `[x]`, E6.5 `[~]` (front testé et monté, blocage de contrat signalé — cf.
  Découvertes).

  **Non traité ici** (hors périmètre déclaré `apps/go-api`, `openapi.yaml`,
  `generated.ts`) : brancher `TeammatesService.WithEquipmentUsage` pour que le bloc
  Escouade s'affiche réellement en production — seul geste manquant, détaillé aux
  Découvertes.

  **Prochaine étape** : §7 (clôture de chantier — `make gate-push`, revue adversariale
  unique, `delivery-checklist`, annotation du plan vague C) — hors périmètre de cette
  session (décision superviseur : un autre exécutant ou une session dédiée mène la
  clôture). Pas de push, pas de fusion.

- **2026-09-09 — E6.1bis** (worktree `LevelUp-wt-equipement-e5-go`, branche
  `feat/equipement-e5-go`) : correctif du blocage signalé par E6.5 ci-dessus — le bloc
  équipement passe de `SquadPageV2Response` (jamais fetché par la page Escouade réelle)
  à `TeammatesPageResponse` (`POST /pages/teammates`, le SEUL endpoint que `SquadLayout`
  appelle).

  **TDD** : deux tests neufs dans `internal/service/teammates/teammates_service_usage_test.go`
  (bloc présent avec le bon scope/amis ; bloc indisponible sans repo câblé). Rouge observé
  en retirant temporairement l'appel `s.loadEquipmentUsage(...)` + l'assignation
  `EquipmentUsage:` du `return` de `GetPage` (`teammates_service.go`) : les deux tests
  échouent (`bloc équipement = <nil>, attendu disponible` / `bloc = <nil>, attendu
  indisponible/unsupported`). Remis en place → vert.

  **Décision d'exécution — déplacement de package pour éviter un cycle** : le service-root
  (`package service`) importe déjà `internal/service/teammates`
  (`synthesis_service_usage.go`, pour `teammates.FriendGamertagsResolver`). `TeammatesService`
  vit dans `service/teammates` ; `buildEquipmentUsageBlock` vivait dans `service` — un appel
  direct aurait fermé un cycle `service → teammates → service`. Déplacé vers
  `internal/service/squadagg` (déjà une FEUILLE importée des deux côtés, même patron que
  `BuildSquadHeader`/`IntersectByMatchID`, K3b) : `EquipmentUsageQuery`/
  `BuildEquipmentUsageBlock` exportés, alias `equipmentUsageQuery`/`buildEquipmentUsageBlock`
  ajoutés à `squadagg_reexport.go` pour que `synthesis_service_usage.go` et
  `equipment_usage_block_test.go` restent inchangés (zéro site d'appel requalifié).

  **Contrat final** : `equipment_usage` publié UNIQUEMENT sur `TeammatesPageResponse`
  (`internal/domain/teammates.go`), via `TeammatesService.WithEquipmentUsage`
  (`internal/service/teammates/teammates_service_usage.go`, fichier neuf), câblé dans
  `registry_pages_home.go` (`TeammatesCtx`) gated `film.usage_summary`. Scope = matchs
  FILTRÉS de la page (`filteredMatches`, période+cascade+sessions déjà appliqués — la
  même population qu'Options/MatchHistory, PAS l'intersection escouade) ; amis =
  `req.SelectedGamertags` (coéquipiers SÉLECTIONNÉS, identique à E6.1) ; sujet =
  `playerXUID` de la route. Retiré de `SquadPageV2Response` (`internal/domain/squad_v2.go`),
  `SquadServiceV2` (`sessionUsageRepo`, `WithEquipmentUsage`, `loadEquipmentUsage`,
  `squadUsageScope` — fichiers `squad_service_v2_usage.go`/`_test.go` supprimés), câblage
  retiré de `SquadV2Ctx`.

  **Gates exécutés sur l'arbre final** :
  - `go build ./... && go vet ./... && go test ./...` — vert, aucun `--- FAIL:`.
  - `go test -tags=integration -p 1 -count=1 ./internal/service/... ./internal/api/...` —
    vert, y compris `internal/api/wire` (le test d'intégration consigné rouge préexistant
    par E3 passe sur ce worktree, cohérent avec l'observation déjà notée E5-Go).
  - `golangci-lint run --new-from-merge-base=feat/v75 ./...` — 0 issue.
  - `make openapi-gen` : diff exact attendu (`equipment_usage` quitte
    `SquadPageV2Response`, apparaît sur `TeammatesPageResponse`, 2 lignes déplacées).
    `make generate-types` : même mouvement dans `generated.ts`.
  - `apps/web` : `rm -rf node_modules/.tmp && npx tsc -b --force` silencieux ;
    `npx vitest run src/features/squad src/features/_shared` — **75 fichiers, 594 tests,
    vert**.
  - `grep -rn "EquipmentUsage" internal/domain/squad_v2.go internal/service/squad_service_v2*.go`
    — 0 résultat.

  **Alignement web** : `apps/web/src/lib/api/types.ts` (`TeammatesPageResponse.equipment_usage`,
  ~ligne 1439) — commentaire mis à jour (écart corrigé), déclaration manuelle du champ
  conservée à l'identique (convention déjà en vigueur pour
  `SynthesisPageResponse.equipment_usage`). Aucun composant web modifié : `EquipmentUsageSection`
  et `SquadSynergiesPage.tsx` lisaient déjà `pageData.equipment_usage`, le câblage
  Go suffit à activer la section — zéro régression, zéro changement de rendu.

  **Découverte consignée, non traitée** (hors périmètre `apps/go-api`/`types.ts`) : le
  commentaire de `SquadSynergiesPage.tsx`/`SquadSynergiesPage.test.tsx` décrivant l'écart
  E6.2-E6.5 est maintenant stale (§6).

  **Statut** : E6.1bis.1 `[x]`, E6.1bis.2 `[x]`, E6.1bis.3 `[x]`. Pas de push, pas de
  fusion.
