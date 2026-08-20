# Plan — Profil Combat : câblage OC + DR + Engagement

**Date** : 2026-05-23  
**Statut** : Brouillon — non démarré  
**Branche cible** : `feat/combat-profile-wiring` (à créer)

**Objectif** : exposer le profil combat 3 axes (OC, DR, Engagement) dans toutes les surfaces analytiques pertinentes, en maximisant la réutilisation du code existant et en introduisant le minimum de nouveau calcul.

**Critère de succès** : un joueur peut voir ses tendances combat (OC + DR + descripteurs textuels) dans Session Compare, Synthesis et Escouade. La match view et le timeseries sont déjà couverts.

---

## 1. État des lieux — ce qui existe déjà

### Calcul backend (100% done, ne pas retoucher)

| Métrique | Fichier | Formule |
|----------|---------|---------|
| `offensive_conversion` | `internal/analysis/combat_yield.go` | `225 × (kills + assists/3) / damage_dealt` |
| `defensive_resistance` | `internal/analysis/combat_yield.go` | `damage_taken / (225 × deaths)` |
| `EngagementScore` | `internal/analysis/temporal/engagement_score.go` | percentile 0–100 vs historique perso |
| `ResidualBrut` | idem | `mean(PaceJoueur − PaceAttendu)`, cross-joueurs |

**Important sur l'Engagement** :
- `EngagementScore` (0–100 vs soi-même) → coaching individuel
- `ResidualBrut` → comparaison cross-joueurs, profiling. Déjà persisté dans `player_match_enrichment` pour le recompute des coefficients — colonne exploitable pour les agrégats.

**Important sur l'Engagement affiché dans le profil** :
La `StyleDisciplineSection` (ObjectifsPage) affiche déjà un "Engagement" — mais c'est la **fréquence de jeu** (matches_per_day, max_gap). Ce n'est pas `ResidualBrut`. Les deux s'appellent "engagement" mais mesurent des choses distinctes. Ne pas les confondre dans les labels UI.

### Surfaces déjà câblées (ne pas toucher)

| Surface | OC/DR | Engagement intra-match |
|---------|-------|------------------------|
| **Match View — Scoreboard** | ✅ via `MatchScoreboardRow` | ✅ conditionnel (Team Tab) |
| **Match Card (Home/MatchHistory)** | ✅ via `RecentMatchItem` | ❌ |
| **Timeseries/Stats** | ✅ `TimeseriesCombatYield` chart | ❌ endpoint séparé |
| **Engagement feature** | ❌ | ✅ endpoints dédiés + squad |

### Surfaces à câbler (cibles du plan)

| Surface | OC/DR | Descripteurs textuels | Difficulté |
|---------|-------|-----------------------|-----------|
| **Session Compare** | ❌ | ❌ | Moyenne |
| **Synthesis** | ❌ | ❌ | Moyenne — surface principale pour les descripteurs |
| **Escouade** | ❌ | ❌ | Faible (OC/DR manquants, engagement déjà wired) |
| **Career — KPI tiles** | ❌ | ❌ | Faible — mais hors scope (voir §6) |

### Composant frontend prêt, non déployé

`CombatYieldBar` (`apps/web/src/components/ui/combat-yield-bar.tsx`) — barre double sens OC + DR avec tooltip. Testé, zéro consumer hors MatchCard.

---

## 2. Décisions de design

1. **DD/(K+A) abandonné** — redondant avec `offensive_conversion`.
2. **Résistance = DT/Death** = `defensive_resistance` existante.
3. **ResidualBrut pour le profilage cross-joueurs**, `EngagementScore` pour le coaching individuel.
4. **Synthesis = surface principale** pour les descripteurs textuels solo. Escouade = surface inter-joueurs avec le même concept, pour cohérence.
5. **Descripteurs génériques, pas de noms d'archétypes** — données trop limitées pour des conclusions fiables. Trois descripteurs indépendants par axe, avec gate sur le nombre de matchs.
6. **Calcul des descripteurs côté backend** — règle métier, pas transformation d'affichage.

### Descripteurs textuels — vocabulaire retenu

Trois axes indépendants, un mot par axe. Pas de nom composite ("Soldat", "Fantôme", etc.) tant que l'historique est insuffisant.

| Axe | Métrique | Bas | Moyen | Haut |
|-----|----------|-----|-------|------|
| **Offensif** | `avg_oc` | "Généreux" (dégâts setup) | "Équilibré" | "Précis" (finisseur) |
| **Défensif** | `avg_dr` | "Fragile" | "Solide" | "Résistant" |
| **Activité** | `avg_residual_brut` | "Discret" | "Modéré" | "Actif" |

Seuils d'affichage :
- Minimum **15 matchs** pour afficher les descripteurs (en dessous : afficher uniquement les valeurs brutes sans label).
- Afficher systématiquement le N de matchs sur lequel les descripteurs sont calculés.

Noms internes backend : `combat_style_offensive`, `combat_style_defensive`, `combat_style_activity` (strings énumérées).

### Archétypes internes (pour le coaching uniquement, jamais affichés tels quels)

Ces combinaisons restent des références internes pour déclencher les messages coaching — elles ne sont pas exposées à l'utilisateur sous ces noms.

| ResidualBrut | OC | Référence interne | DR affine |
|---|---|---|---|
| Positif | Haut | Soldat | DR bas → fragile offensif |
| Positif | Bas | Fonçeur | DR haut → absorbe sans convertir |
| Négatif | Haut | Sniper | Profil légitime si KDA positif |
| Négatif | Bas | Fantôme | DR bas → proie facile ; DR ok → hider |

---

## 3. Phases

### Phase 1 — Synthesis : OC + DR + descripteurs textuels
**Valeur** : haute — surface profil solo naturelle. **Risque** : faible. **Effort** : ~1.5 j.

**Constat** : `SynthesisPageV2Response` a un bloc overview (KDA, kills, win rate). OC/DR et descripteurs absents. La Synthesis est déjà une "page profil" — c'est le bon endroit.

**Backend** :
- `internal/platform/duckdb/synthesis_repo.go` (ou équivalent) : ajouter `AVG(offensive_conversion)`, `AVG(defensive_resistance)`, `AVG(engagement_residual_brut)` + count matchs dans l'agrégat.
- Nouveau type `CombatProfileBlock` dans `internal/domain/synthesis.go` :
  ```go
  type CombatProfileBlock struct {
      AvgOC            float64
      AvgDR            float64
      AvgResidualBrut  float64
      MatchCount       int
      // nil si MatchCount < 15
      StyleOffensive   *string // "precis" | "equilibre" | "genereux"
      StyleDefensive   *string // "resistant" | "solide" | "fragile"
      StyleActivity    *string // "actif" | "modere" | "discret"
  }
  ```
- Fonction `classifyCombatProfile(oc, dr, residual float64) CombatProfileBlock` dans `internal/analysis/combat_yield.go`.

**Frontend** :
- Nouveau bloc `SynthesisCombatProfileSection` dans `apps/web/src/features/synthesis/`.
- Affiche `CombatYieldBar` (OC + DR visuels) + 3 badges textuels (Offensif / Défensif / Activité) avec le N matchs en sous-titre.
- Insérer après `SynthesisOverviewSection` (Bloc 1).

---

### Phase 2 — Escouade : OC + DR + descripteurs par joueur
**Valeur** : haute — comparaison composition d'équipe. **Risque** : faible. **Effort** : ~1 j.

**Principe de cohérence** : même concept que Synthesis, même descripteurs, même gate N≥15. Un joueur voit son profil combat seul en Synthesis, et celui de ses coéquipiers en Escouade avec le même vocabulaire.

**Backend** :
- `internal/platform/duckdb/squad_repo.go` : ajouter OC/DR/ResidualBrut moyens par joueur pour les matchs communs + `classifyCombatProfile` appliqué.
- Exposer `CombatProfileBlock` par joueur dans la réponse escouade.

**Frontend** :
- `apps/web/src/features/squad/` : nouveau composant `SquadCombatProfileRow` — `CombatYieldBar` + 3 badges par joueur, alignés en tableau pour comparaison directe.

---

### Phase 3 — Session Compare : colonnes OC + DR
**Valeur** : haute. **Risque** : faible. **Effort** : ~1 j.

**Constat** : `SessionCompareResponse` expose KDA, Performance Score, Win Rate — pas OC/DR. Pas de descripteurs textuels ici (la session est trop courte pour les gates N≥15 en général).

**Backend** :
- `session_repo.go` ou `queries_match.go` : ajouter `AVG(offensive_conversion)` et `AVG(defensive_resistance)` dans l'agrégat par session.
- Étendre `internal/domain/session.go`.

**Frontend** :
- Colonnes OC + DR dans `apps/web/src/features/session-compare/`.
- `CombatYieldBar` en miniature par ligne session.

---

### Phase 4 — Engagement dans les agrégats (Synthesis + Session Compare)
**Valeur** : haute pour le coaching. **Risque** : moyen. **Effort** : ~2 j.

> ⚠️ **Bloquer cette phase sur vérification préalable** : confirmer que `player_match_enrichment` contient bien une colonne pour `ResidualBrut` et qu'elle est peuplée. Si non peuplée → backfill ou scope réduit aux nouveaux matchs.

**Backend** :
- Vérifier `player_match_enrichment.engagement_residual_brut` (nom exact à confirmer dans le schéma).
- Si ok : join direct dans les queries synthesis/session → `avg_residual_brut` sans coût supplémentaire.

**Frontend** :
- Synthesis : le `CombatProfileBlock.StyleActivity` s'affiche dès que ResidualBrut est disponible.
- Session Compare : colonne activité optionnelle (affichée si données disponibles).

---

### Phase 5 — Coaching proactif (Ascension)
**Valeur** : haute pour la rétention. **Risque** : moyen. **Effort** : ~3 j.

Dépend des Phases 1–4. Voir section algo coaching dans le plan (post_sync_progression.go).

**Ce qui change dans l'algo** :
- `loadProgressionMatches` : ajouter OC/DR/ResidualBrut dans `MatchActivity.Stats`.
- `medianOC` + `medianDR` helpers (copie de `medianKDA`).
- `generateCoachAlerts` : passer OC/DR trends via `LOWESSTrends` existant.
- Nouveaux `AlertType` dans `coach/types.go` : `AlertTypeCombatPatternActif`, `AlertTypeCombatPatternDiscret`, `AlertTypeCombatPatternFragile`.
- `buildCombatPatternAlerts` dans `coach/generator.go` : détection sur N matchs consécutifs (nouveau pattern, seule vraie nouveauté algo).

**Messages coaching** (positifs uniquement, jamais sur régression) :

| Condition | Message |
|-----------|---------|
| OC haut confirmé (LOWESS positif) | "Ton efficacité au combat s'améliore — tu closes mieux tes duels" |
| DR haut confirmé | "Tu résistes mieux — tu encaisses avant de tomber" |
| OC bas + activité haute (N matchs) | "Tu te bats beaucoup mais perds tes duels à la fin — travaille le finish" |
| Activité très basse (N matchs) | "Tu t'impliques peu dans le match — cherche plus le contact" |
| DR bas systématique | "Tu meurs trop vite — positionnement défensif à revoir" |

**Milestones TOML** (ajouts au catalogue) :
- `combat.precision_1/2/3` : X matchs avec OC ≥ p75 + activité ≥ médiane perso
- `combat.endurance_1` : X matchs avec DR ≥ p80
- `combat.consistency` : 10 matchs avec OC + DR tous deux au-dessus de la médiane perso

---

## 4. Ordre recommandé

```
Phase 1 (Synthesis)  →  Phase 2 (Escouade)  →  Phase 3 (Session Compare)
                     →  Phase 4 (Engagement agrégats) — après vérif PME
                     →  Phase 5 (Coaching) — après Phase 4
```

Phases 1–3 sont indépendantes, livrables en un seul PR. Phase 4 dépend d'une vérification préalable. Phase 5 dépend de Phase 4.

---

## 5. Fichiers clés

| Fichier | Rôle | Action |
|---------|------|--------|
| `internal/analysis/combat_yield.go` | Calcul OC/DR | Ajouter `classifyCombatProfile` |
| `internal/analysis/temporal/engagement_score.go` | Calcul engagement | Ne pas modifier |
| `apps/web/src/components/ui/combat-yield-bar.tsx` | Visu OC/DR | Instancier dans Synthesis, Escouade, Session Compare |
| `apps/web/src/features/timeseries/TimeseriesCombatYield.tsx` | Modèle chart OC/DR | Référence de pattern |
| `apps/web/src/features/prestige/components/profile/StyleDisciplineSection.tsx` | Engagement fréquence (existant) | Ne pas modifier — concept différent |

---

## 6. Ce qui est hors scope de ce plan

- **Career page** — le profil narratif complet est dans Ascension/ObjectifsPage. Dupliquer les descripteurs combat dans Career créerait une redondance sans valeur claire. Synthesis couvre le besoin de "vue profil solo".
- Renommer `offensive_conversion` / `defensive_resistance` — noms internes stables.
- Créer DD/(K+A) — abandonné.
- Modifier le calcul d'engagement — algorithme stable.
- Leaderboard et Home — hors scope confirmé.
- Noms d'archétypes composites ("Soldat", "Fantôme") dans l'UI — réservés au coaching interne, jamais exposés à l'utilisateur avec les données actuelles.
