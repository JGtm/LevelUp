# PLAN — Settings : Règles d'analyse configurables

**Date** : 2026-04-24  
**Branche cible** : à créer depuis `feat/v7-assets-abstraction` (ou `main` selon timing)  
**Statut** : Réflexion / À valider

---

## 1. Contexte et état actuel

### Règles de sessions (groupement des matchs)

Le code actuel (`apps/go-api/internal/analysis/sessions.go`) implémente :

| Constante / Paramètre | Valeur actuelle | Localisation |
|---|---|---|
| `DefaultSessionGapMinutes` | **120 min (2h)** | `analysis/sessions.go:16` |
| `SplitOnRankedChange` | **false (hardcodé)** | `ComputeSessionsWithContext` |
| `isTeammatesBreak` | actif (compare `teammates_signature`) | idem |
| `CutoffHour` | 08h00 (découpe de nuit) | `analysis/sessions.go:18` |

En frontend, `gap_minutes=120` est une constante invariante dans `apps/web/src/stores/globalFilterStore.ts` (commentaire explicite : _"Invariant : toujours 120"_).

`ComputeSessionsWithContext` supporte déjà trois critères de rupture :
1. **Gap temporel** : `startTime[i] - startTime[i-1] > gapMinutes`
2. **Changement de coéquipiers** : `isTeammatesBreak(prevSig, curSig, friendSet)` — coupe si la composition change significativement par rapport aux amis connus
3. **Passage ranked/non-ranked** : `SplitOnRankedChange && (prev.IsRanked != cur.IsRanked)`

> **Problème actuel** : ces paramètres ne sont pas configurables. L'utilisateur ne peut pas choisir si un départ de coéquipier ou un changement ranked → sociale constitue une nouvelle session.

### Flags narratifs / Outcome

Stockés dans `player_match_enrichment.dominance_flag` (TINYINT) :

| Valeur | Label | Signification |
|---|---|---|
| 0 | — | Aucun badge |
| 1 | DOMINATION | Victoire écrasante |
| 2 | HUMILIATION | Défaite écrasante subie |
| 3 | REMONTADA | Victoire après avoir été mené |
| 4 | DÉBÂCLE | Défaite après avoir mené |
| 5 | CONTRE-REMONTADA | Victoire malgré un 2ème retard adverse |

La colonne `had_bot_teammate` (BOOLEAN, `player_match_enrichment`) indique si un bot était présent côté joueur lors du match.

> **Problème actuel** : la logique de calcul (`comeback_analysis.py`) n'est **pas encore portée en Go**. Les seuils (ex : proportion du score d'équipe, amplitude du lead retourné) sont codés en dur côté Python et ne sont pas configurables.

### Onboarding (SetupPage)

Machine d'état actuelle (`setup_state`) :
```
no_halo_link → StepDeviceCode
halo_linked_no_profile → StepPlayer
profile_ready_no_sync → StepInitialSync
ready → redirect /
```

Aucune étape de personnalisation des règles d'analyse n'existe avant la synchro initiale.

---

## 2. Questions de fond

### 2a. Délai de session — quel impact si changé ?

Changer `gap_minutes` ou les critères de rupture **invalide tous les `session_id` / `session_label`** déjà calculés dans `player_match_enrichment`. Un recalcul complet (backfill `--sessions --force-sessions`) est nécessaire.

→ **Décision recommandée** : exposer le paramètre mais afficher un avertissement "Ce changement nécessite un recalcul des sessions (peut prendre quelques secondes)". Déclencher automatiquement un job async de recalcul au PATCH.

### 2b. Coéquipiers qui partent/rejoignent : nouvelle session ?

`isTeammatesBreak` compare les `teammates_signature` (hash SHA du set de coéquipiers) sur deux matchs consécutifs. Actuellement : si la compagnie change, **nouvelle session**. C'est la règle la plus subjective.

Options exposées :
- **Mode "par temps uniquement"** : ignorer la composition (rebrancher uniquement le gap temporel)
- **Mode "composition stable"** : nouvelle session si au moins 1 coéquipier quitte ou rejoint (comportement actuel)
- **Mode "groupe core"** : nouvelle session seulement si les amis définis dans `friend_gamertags` changent (pas les inconnus)

→ **Décision recommandée** : liste déroulante à 3 choix (plus lisible qu'un toggle quand il y a 3 options sémantiquement distinctes) :
- `"Ignorer la composition"` — sessions découpées par le temps uniquement
- `"Changement de groupe"` *(défaut)* — nouvelle session dès qu'un coéquipier quitte ou rejoint
- `"Changement d'amis seulement"` — nouvelle session uniquement si un joueur de `friend_gamertags` change

### 2c. Ranked → Social = nouvelle session ?

`SplitOnRankedChange` est faux par défaut. Si activé, passer d'une partie rankée à une sociale découpe la session.

→ Exposer comme toggle optionnel.

### 2d. Flags d'outcome : configurer les seuils, oui ou non ?

Les seuils de domination/remontada sont complexes à expliquer à un utilisateur non-expert. Les valeurs brutes (ex : `score_delta > 40% de l'amplitude totale`) sont difficilement interprétables.

Options :
- **Option A — Seuils en %** : exposer un slider "Amplitude de domination" (ex : 30–70 %)
- **Option B — Préréglages** : "Relaxé / Standard / Strict" qui mappe sur des triplets de seuils prédéfinis
- **Option C — Ne pas exposer** : les seuils restent internes, seul le flag bots est configurable

→ **Décision recommandée** : **Option B** — 3 boutons radio "Souple / Standard / Strict". Le choix radio est préférable au dropdown car les 3 options sont toujours visibles et la comparaison est immédiate. Chaque option est accompagnée **non pas d'un simple tooltip mais d'un texte d'aide permanent** sous le groupe radio, expliquant :
- ce que l'algo mesure (évolution du score d'équipe au fil du match)
- ce que "seuil" signifie concrètement (ex : "Un écart de 40 % = si une équipe mène 10-6 dans un match à 20 points")
- l'impact attendu sur le nombre de badges attribués dans l'historique

### 2e. Exclusion des matchs avec bots : où l'appliquer ?

`had_bot_teammate = true` peut être utilisé pour :
1. **Exclure du calcul des flags narratifs** (une domination avec des bots dans l'équipe adverse n'a pas de sens)
2. **Exclure du classement carrière / records** (déjà partiellement géré par `career_top_exclude_btb`)
3. **Les deux**

→ **Décision recommandée** : exposer deux toggles distincts, chacun accompagné d'un paragraphe d'explication permanent (pas un tooltip) :

**Toggle 1 — Badges** : _"Quand Halo Infinite ne trouve pas assez de joueurs, il remplace les absents par des bots. Ces bots jouent mal et font exploser les scores adverses, ce qui peut déclencher artificiellement un badge Domination ou Humiliation. Ce toggle retire ces matchs du calcul."_

**Toggle 2 — Records** : _"Les matchs avec bots peuvent produire des stats de combat atypiques (K/D très élevé, précision gonflée) qui fausseraient vos records personnels. Ce toggle les exclut du calcul de vos meilleures performances."_

### 2f. Faut-il présenter ces choix à l'onboarding ?

**Argument pour** : l'utilisateur configurerait ses préférences avant d'ingérer 500 matchs, évitant de devoir recalculer tout après coup.

**Argument contre** :
- L'onboarding est déjà un point de friction (auth Xbox, création profil, attente sync)
- Les valeurs par défaut sont raisonnables pour 95% des utilisateurs
- Le recalcul post-hoc est automatisable (job async, quelques secondes)
- Les flags narratifs se calculent lors du backfill *après* le sync, pas pendant

→ **Décision recommandée** : **Pas d'étape onboarding supplémentaire**. À la place :
- Le `StepInitialSync` affiche une note grisée : _"Les règles de regroupement des sessions et les badges de performance sont configurables dans Paramètres après la synchronisation."_
- Un lien direct `→ Paramètres` est affiché après la première synchro (dans la page d'accueil ou en notification).

---

## 3. Plan d'implémentation

### Phase 1 — Backend : nouveaux champs dans Settings

**Fichiers concernés** :
- `internal/domain/settings.go` — ajouter les champs dans `SettingsResponse` et `UpdateSettingsRequest`
- `internal/platform/settings/store.go` + `defaults.go` — valeurs par défaut
- `internal/platform/settings/apply.go` — application des PATCH
- `api/openapi.yaml` — documentation API

**Champs à ajouter** :

```go
// --- Règles de sessions ---
SessionGapMinutes            int  `json:"session_gap_minutes"`          // défaut: 120
SessionSplitOnRankedChange   bool `json:"session_split_on_ranked_change"` // défaut: false
SessionSplitOnTeamChange     bool `json:"session_split_on_team_change"`  // défaut: true (comportement actuel)

// --- Règles de badges narratifs ---
OutcomeExcludeBotMatchesFromBadges bool   `json:"outcome_exclude_bot_matches_from_badges"` // défaut: true
OutcomeExcludeBotMatchesFromRecords bool  `json:"outcome_exclude_bot_matches_from_records"` // défaut: false
OutcomeBadgeSensitivity            string `json:"outcome_badge_sensitivity"` // "relaxed"|"standard"|"strict", défaut: "standard"
```

**Valeurs par défaut** à documenter dans `defaults.go` :

| Champ | Défaut | Raison |
|---|---|---|
| `session_gap_minutes` | 120 | Historique Python + comportement attendu |
| `session_split_on_ranked_change` | false | Moins de coupes inattendues |
| `session_split_on_team_change` | true | Comportement actuel `isTeammatesBreak` |
| `outcome_exclude_bot_matches_from_badges` | true | Bots faussent les scores adverses |
| `outcome_exclude_bot_matches_from_records` | false | Pas de changement de comportement par défaut |
| `outcome_badge_sensitivity` | "standard" | Seuils historiques Python |

### Phase 2 — Backend : propagation vers les algorithmes

**`analysis/sessions.go`** : `ComputeSessionsWithContext` doit recevoir `SessionComputeOptions` depuis les settings (actuellement `SplitOnRankedChange=false` hardcodé).

**`internal/sync/engine.go`** (ou équivalent) : lors du PATCH settings sur les champs sessions, déclencher automatiquement un job async de recalcul des sessions pour tous les joueurs actifs.

**Portage `comeback_analysis.py` → Go** : créer `internal/analysis/comeback.go` avec :
- `ComputeDominanceFlag(kills/score events, sensitivity string) int`
- Mappings des seuils par sensibilité

> Note : ce portage est un chantier indépendant. À prioriser avant d'exposer la config `outcome_badge_sensitivity`.

### Phase 3 — Frontend : section Settings

**Fichiers concernés** :
- `apps/web/src/features/settings/SettingsPage.tsx` — ajouter sections/cards
- `apps/web/src/features/settings/i18n.ts` — traductions FR/EN
- `apps/web/src/lib/api/types.ts` — champs TS

**Structure UI recommandée** — nouvel onglet **"Analyse"** dans `SettingsPage` (à côté de Général / Synchronisation) :

```
Onglet : Analyse
│
├── Card "Regroupement de sessions"
│   ├── Délai entre deux matchs : [slider 15–480 min] — défaut 120 min
│   │   ↳ Aide permanente : "Deux matchs séparés de plus de X minutes
│   │     appartiennent à des sessions différentes. 120 min (2 h) est le
│   │     réglage recommandé pour une soirée de jeu classique."
│   ├── Composition de l'équipe : [Dropdown]
│   │     • Ignorer la composition
│   │     • Changement de groupe (défaut)
│   │     • Changement d'amis seulement
│   │   ↳ Aide permanente : "Détermine si un changement dans votre
│   │     groupe de coéquipiers déclenche une nouvelle session, en
│   │     complément du délai. 'Changement de groupe' correspond au
│   │     comportement par défaut. 'Amis seulement' utilise la liste
│   │     de Paramètres → Général → Amis."
│   ├── Toggle : Couper si passage ranked ↔ social
│   │   ↳ Aide permanente : "Active une nouvelle session dès que vous
│   │     basculez entre une playlist classée et une playlist sociale.
│   │     Utile si vous séparez vos stats ranked des sessions casual."
│   └── [⚠ Bouton] "Recalculer les sessions" — grisé si un job tourne déjà
│         ↳ Alerte si sync/backfill en cours : voir §3 gestion concurrence
│
└── Card "Badges de performance"
    │
    ├── Sensibilité des badges : [Radio : Souple | Standard | Strict]
    │   ↳ Aide permanente (toujours visible sous le groupe radio) :
    │     "LevelUp analyse l'évolution du score d'équipe au fil du match
    │     pour détecter les moments décisifs. Un 'écart' correspond à
    │     l'avance d'une équipe par rapport à l'autre, exprimée en % du
    │     score final total.
    │     Exemple : un match terminé 50-30 a un écart de 40 %
    │     (soit le seuil Standard).
    │     • Souple : écart ≥ 25 % → plus de badges, y compris les matchs
    │       assez déséquilibrés.
    │     • Standard : écart ≥ 40 % → seuils utilisés historiquement
    │       (recommandé).
    │     • Strict : écart ≥ 60 % → uniquement les matchs très marquants.
    │     Ce réglage nécessite un recalcul des badges (job automatique)."
    │
    ├── Toggle : Exclure les matchs avec bots des badges
    │   ↳ Aide permanente : "Quand Halo Infinite manque de joueurs, il
    │     remplace les absents par des bots. Ces bots jouent mal et gonflent
    │     les scores adverses — un badge Domination ou Humiliation obtenu
    │     contre des bots ne reflète pas votre niveau réel. Ce toggle
    │     désactive le calcul de badges pour ces matchs."
    │
    └── Toggle : Exclure les matchs avec bots des records carrière
        ↳ Aide permanente : "Les matchs avec bots peuvent produire des
          stats atypiques (K/D très élevé, précision gonflée) qui
          fausseraient vos records personnels. Ce toggle les retire du
          calcul de vos meilleures performances, indépendamment des badges."
```

**Comportement auto-save** : aligné sur le pattern existant (mutation immédiate au changement, feedback "✓ Enregistré").

**Alerte de recalcul** : si `session_gap_minutes`, `session_split_on_ranked_change` ou `session_split_on_team_change` changent → afficher un bandeau `"Vos sessions vont être recalculées (quelques secondes)"` + lancer un job async.

### Phase 4 — Frontend : note onboarding

**Fichier** : `apps/web/src/features/setup/SetupPage.tsx`

Dans `StepInitialSync`, en dessous du titre et avant le bouton "Lancer la synchronisation", ajouter un bloc discret :

```tsx
<p className="text-xs text-muted-foreground mt-2">
  Les règles de regroupement des sessions et les badges de performance
  sont configurables dans{' '}
  <Link to="/settings">Paramètres → Analyse</Link>{' '}
  après la synchronisation.
</p>
```

---

## 4. Dépendances et risques

| Risque | Probabilité | Mitigation |
|---|---|---|
| Portage `comeback_analysis.py` → Go non trivial (algo basé sur les events kill) | Haute | Implémenter Phase 3 UI avec contrôle grisé "À venir" si le portage n'est pas fini |
| Recalcul sessions sur 2000+ matchs → lenteur | Faible | Opération O(n) en DuckDB, < 1s pour 2000 matchs |
| **Conflit d'écriture DB** : recalcul sessions déclenché pendant un sync ou backfill en cours | Moyenne | Voir §4a — gestion de concurrence |
| `gap_minutes` invariant dans `globalFilterStore.ts` à déverrouiller | Certaine | Lire la valeur depuis `settings` au chargement du store |

### 4a. Gestion de la concurrence DB lors du recalcul

Le recalcul des sessions écrit dans `player_match_enrichment` (colonnes `session_id`, `session_label`). Ce sont exactement les mêmes colonnes qu'un sync delta ou un backfill `--sessions` en cours.

**Risques sans garde** :
- Recalcul écrase des lignes en cours d'écriture par le sync → `session_id` incohérents
- Sync termine après le recalcul → session_id remis à NULL pour les nouveaux matchs

**Stratégie recommandée** :

1. **Avant de lancer le job de recalcul**, vérifier via `jobStore` si un job `initial_sync`, `backfill` ou `sessions` est en cours (`status = running`) pour le même joueur.
   - Si oui : **mettre en file d'attente** (état `pending`) et replanifier automatiquement à la fin du job en cours.
   - Si non : lancer immédiatement.

2. **Côté UI** : le bouton "Recalculer les sessions" est **grisé** (disabled) si un job sync/backfill est actif sur n'importe quel joueur. Un bandeau explique : _"Une synchronisation est en cours. Le recalcul démarrera automatiquement à sa fin."_

3. **DuckDB** : la DB player est en mode `read_write` exclusif par joueur. Le job de recalcul doit acquérir la même connexion exclusive plutôt qu'en ouvrir une parallèle — aligner avec le pattern `duckdb_read_write()` existant.

4. **Timeout** : si le job en attente ne peut pas démarrer dans les 10 minutes (ex : sync bloqué), l'annuler et notifier l'utilisateur.

---

## 5. Ordre de priorité recommandé

1. **[P1] Portage `comeback_analysis.py` → Go** — prérequis bloquant pour la config des badges ; voir détail §5a
2. **[P1] Section Settings — Règles de session** (gap, ranked, team change) — impact immédiat, logique Go déjà prête
3. **[P1] Section Settings — Badges narratifs** (sensibilité + exclusion bots) — déblocable après portage Go
4. **[P2] Note onboarding** `StepInitialSync` — trivial, 5 lignes
5. **[P3] Déverrouiller `gap_minutes` dans `globalFilterStore`** — mineur, l'UI filtre toujours côté backend

### 5a. Détail du portage `comeback_analysis.py` → Go

**Fichier source** : `src/analysis/comeback_analysis.py` (Python legacy)  
**Fichier cible** : `apps/go-api/internal/analysis/comeback.go`

**Ce que fait l'algo** (d'après le changelog + la structure dominance_flag) :
- Charge les `highlight_events` (type `kill`, champ `time_ms`) pour reconstruire une courbe de score par équipe au fil du match
- Calcule le lead maximal de chaque équipe à différents instants (`build_score_snapshot`)
- Détecte le badge selon le pattern du score : domination stable, humiliation, retournement de situation (remontada / contre-remontada), débâcle
- Prend en compte `had_bot_teammate` pour invalider le calcul si des bots étaient présents

**Signatures Go cibles** :

```go
// internal/analysis/comeback.go

// ScoreSnapshot représente l'état du score à un instant t du match.
type ScoreSnapshot struct {
    TimeMS      int64
    Team0Score  int
    Team1Score  int
}

// BuildScoreSnapshots reconstruit la courbe de score depuis les events kill.
// events doit être trié par time_ms ASC.
func BuildScoreSnapshots(events []domain.KillEvent, playerTeamID int) []ScoreSnapshot

// ComputeDominanceFlag calcule le dominance_flag (0–5) à partir des snapshots.
// sensitivity : "relaxed" | "standard" | "strict"
// hadBotTeammate : si true et outcome_exclude_bot_matches_from_badges → retourne 0
func ComputeDominanceFlag(
    snapshots []ScoreSnapshot,
    playerOutcome int,
    sensitivity string,
    excludeBots bool,
    hadBotTeammate bool,
) int
```

**Seuils par sensibilité** (à valider / rétro-ingénierer depuis Python) :

| Sensibilité | Lead min pour Domination | Amplitude min de retournement |
|---|---|---|
| `relaxed` | 25 % | 20 % |
| `standard` | 40 % | 35 % (seuils historiques Python) |
| `strict` | 60 % | 55 % |

**Tests** : `internal/analysis/comeback_test.go` — couvrir les 5 cas (domination, humiliation, remontada, débâcle, contre-remontada) + cas bots + cas score plat.

**Backfill** : le scope `ComebackBadges` existe déjà (`sync/scope.go`). Une fois l'algo Go implémenté, brancher dans l'engine de backfill (qui appelait l'algo Python via subprocess ou le laissait au backfill_data.py).

---

## 6. Tests et logging

### 6a. Tests unitaires — `comeback.go`

Fichier : `internal/analysis/comeback_test.go`

Cas obligatoires (non-régression) :

| Cas | Setup | Flag attendu |
|---|---|---|
| Victoire stable, lead constant ≥ seuil | score 6-0 à mi-match | `1` (Domination) |
| Défaite stable, retard constant ≥ seuil | score 0-6 à mi-match | `2` (Humiliation) |
| Victoire après retard | score -4 à t=30%, puis remontée | `3` (Remontada) |
| Défaite après lead | score +4 à t=30%, puis effondrement | `4` (Débâcle) |
| Victoire malgré 2ème retard | lead → retard → nouveau retard → victoire | `5` (Contre-Remontada) |
| Score plat / match serré | écart jamais ≥ seuil | `0` (aucun badge) |
| Bots présents + `excludeBots=true` | n'importe quel score | `0` (neutralisé) |
| Bots présents + `excludeBots=false` | score Domination | `1` (calcul normal) |
| Events vides | slice nil | `0` (pas de panic) |
| Sensibilité `relaxed` vs `strict` | même score intermédiaire | badge présent vs absent |

Les valeurs de seuil **doivent être rétro-validées** contre les données réelles de la DB (`--comeback-badges` lancé sur JGtm) avant merge.

### 6b. Tests unitaires — `sessions.go` (non-régression)

Fichier existant : `internal/analysis/sessions_test.go` — **à étendre** avec :

| Cas | Vérification |
|---|---|
| `gap_minutes=120` → comportement identique à avant | résultats identiques aux golden values existantes |
| `gap_minutes=30` → sessions plus fréquentes | moins de matchs par session |
| `gap_minutes=480` → sessions plus larges | plus de matchs regroupés |
| `SplitOnTeamChange=false` → ignore la signature | même session même si composition change |
| `SplitOnRankedChange=true` → coupe ranked↔social | deux sessions distinctes |
| Combinaison `SplitOnTeamChange=true` + `SplitOnRankedChange=true` | chaque critère coupe indépendamment |

**Golden value critique** : le résultat avec `gap=120, teamChange=true, rankedChange=false` doit être **byte-for-byte identique** aux sessions calculées aujourd'hui (test de non-régression strict).

### 6c. Tests d'intégration — endpoint PATCH /settings

Fichier : `internal/api/handlers/settings_test.go` — étendre avec :

- PATCH `session_gap_minutes=30` → réponse contient la nouvelle valeur
- PATCH `session_gap_minutes=-1` → 400 (validation)
- PATCH `outcome_badge_sensitivity="invalid"` → 400
- PATCH `outcome_badge_sensitivity="strict"` → 200, valeur persistée
- PATCH `session_split_on_team_change=false` → 200, valeur persistée
- Round-trip GET→PATCH→GET : les valeurs modifiées sont relues correctement

### 6d. Tests frontend — `SettingsPage`

Fichier : `apps/web/src/features/settings/SettingsPage.test.tsx` — étendre avec :

- L'onglet "Analyse" est visible
- Le slider `session_gap_minutes` affiche la valeur courante
- Modifier le slider déclenche `mutation.mutate` avec la bonne valeur
- Le bandeau de recalcul s'affiche si `session_gap_minutes` change
- Les 3 boutons radio de sensibilité sélectionnent la bonne valeur
- Mode démo : les champs sont en lecture seule

### 6e. Logging

**`comeback.go`** : utiliser `slog` (aligné sur le reste du projet) :

```go
slog.Debug("comeback: match analysé",
    "match_id", matchID,
    "flag", flag,
    "sensitivity", sensitivity,
    "had_bot", hadBotTeammate,
    "excluded", excluded,
)
slog.Warn("comeback: events vides, flag=0 par défaut", "match_id", matchID)
```

**Backfill sessions** (lors du recalcul déclenché par PATCH settings) :

```go
slog.Info("sessions: recalcul déclenché",
    "player", playerSlug,
    "gap_minutes", opts.GapMinutes,
    "split_ranked", opts.SplitOnRankedChange,
    "split_team", opts.SplitOnTeamChange,
    "matches_count", len(rows),
)
slog.Info("sessions: recalcul terminé",
    "player", playerSlug,
    "sessions_count", len(groups),
    "duration_ms", elapsed.Milliseconds(),
)
```

**Settings PATCH** : logger chaque modification de champ sensible à niveau `Info` (pas de valeurs secrètes — ces champs ne sont pas secrets) :

```go
slog.Info("settings: règles d'analyse modifiées",
    "field", fieldName,
    "old", oldValue,
    "new", newValue,
)
```

---

## 7. Scope hors plan

- La **logique BTB** (`career_top_exclude_btb`) existante n'est pas modifiée — elle reste dans l'onglet Général
- Pas de migration de schéma DB requise (tous les champs utilisent les tables et colonnes existantes)
- Le recalcul des sessions ne touche que `player_match_enrichment` (pas les tables shared)
