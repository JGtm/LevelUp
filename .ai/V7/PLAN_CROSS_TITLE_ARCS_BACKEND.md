# Plan — Cross-titre arcs : préparation backend (backend-ready, sans UX)

> **Créé le** : 2026-06-09
> **Statut** : ✅ **LIVRÉ 2026-06-18** (backend-ready, sans UX, comme cadré). Phases 1+2+3 implémentées sur `feat/multititre-peripherie`. Comportement mono-titre observable strictement inchangé. cf. thought_log 2026-06-18.
> **Priorité** : 🟢 Basse aujourd'hui, mais **à anticiper** : un nouveau titre arrive bientôt (intégration non encore planifiée).
> **Origine backlog** : `[coach/prestige] V3 — Cross-titre arcs`

> **Bilan livraison** : table `arc_titles(arc_id, title_slug)` + migration `create_arc_titles_join` (schéma + backfill 1-ligne/arc idempotent) ; `prestige.ArcTitlesRepo` (`ArcTitles`/`ArcsByTitle`, fallback `arc.title_slug`) + `Create` maintient l'invariant ; point d'extension unique `creditTitlesFor(challenge)` (= titre primaire aujourd'hui, câblé dans `creditCompletion`). Tests : backfill+idempotence, repo+invariant+fallback, extension point. **Reste hors périmètre (garde-fou)** : créer un arc réellement multi-titres + trancher la répartition PP (décision produit+UX, pending 2e titre réel).

## Objectif & cadrage

Aujourd'hui chaque `Arc` (et `Challenge`) est lié à **un seul** `title_slug`
(`internal/prestige/types.go:70`). On veut **préparer le backend** pour qu'un arc puisse couvrir
plusieurs titres, **sans** définir l'UX/UI (impossible à ce stade : le 2e titre n'est pas encore
en pipeline produit, son périmètre est inconnu).

**Principe directeur** : tout est **additif et réversible**. Aucune fonctionnalité cross-titre
n'est activée ni exposée ; on pose seulement le socle de données + les accès, derrière un état
neutre qui se comporte exactement comme aujourd'hui en mono-titre.

## Décision de modélisation : table de jointure (validée)

**Retenu : table `arc_titles(arc_id, title_slug)`** plutôt que `Arc.TitleSlugs []string`.

Justification (relation **many-to-many** réelle : un arc peut couvrir N titres ; un titre porte N arcs) :
- ✅ **Additif** : `Arc.TitleSlug` reste comme **titre primaire** (rétro-compat totale) ; les lectures
  mono-titre existantes ne changent pas.
- ✅ **Normalisé / pérenne** : pas de liste sérialisée dans une colonne (anti-pattern requêtes), index
  natif sur `(title_slug)` et `(arc_id)`.
- ✅ **Réversible** : si abandonné, on `DROP TABLE` sans toucher au schéma `arcs`.
- ❌ La variante `[]string` casserait **chaque** lecture/écriture `Arc` existante et dénormaliserait.

**Invariant de transition** : pour tout arc existant, `arc_titles` contient exactement une ligne
`(arc.id, arc.title_slug)`. Un arc reste donc « mono-titre » par défaut — comportement identique à
l'actuel tant que personne n'ajoute de 2e ligne.

## Phases (backend uniquement)

### Phase 1 — Schéma + backfill (migration additive)
1. `CREATE TABLE arc_titles (arc_id TEXT, title_slug TEXT, PRIMARY KEY (arc_id, title_slug))`
   (+ index sur `title_slug`). Migration dans le pipeline `internal/migration/`.
2. Backfill : `INSERT INTO arc_titles SELECT id, title_slug FROM arcs` (1 ligne/arc).
3. **Aucune suppression** de `arcs.title_slug` (reste source du titre primaire).

### Phase 2 — Accès données (repo), sans changer la sémantique
1. Helpers repo : `ArcTitles(arcID) []string`, `ArcsByTitle(slug)` lit via `arc_titles` (fallback
   `arcs.title_slug` si la table est vide — garde-fou pré-backfill).
2. **Garder** les lectures actuelles `WHERE title_slug = ?` fonctionnelles ; la nouvelle voie est un
   sur-ensemble. Pas de bascule des consommateurs maintenant.
3. Écriture : à la création d'un arc, insérer la ligne `arc_titles` correspondante en plus du champ
   primaire (cohérence de l'invariant).

### Phase 3 — Point d'extension PP (spécifié, non implémenté)
Le **crédit des PP par titre** pour un challenge cross-titre est une décision **produit + UX** (un
challenge cross-titre crédite-t-il chaque titre, le primaire seulement, au prorata ?). À ce stade :
- **Ne pas trancher.** Documenter la question ici et laisser le calcul PP actuel inchangé (titre
  primaire).
- Poser un seul point d'extension testable : une fonction `creditTitlesFor(challenge) []string` qui
  retourne aujourd'hui `[challenge.TitleSlug]` (comportement actuel), surchargeable plus tard.

## Garde-fous & non-objectifs

- ⛔ **Ne pas** exposer d'endpoint ni de champ API cross-titre (pas d'UX définie).
- ⛔ **Ne pas** créer d'arc réellement multi-titres en prod tant que le 2e titre n'est pas validé.
- ✅ Tests : invariant 1-ligne/arc après backfill ; `ArcsByTitle` ≡ ancienne requête en mono-titre ;
  `creditTitlesFor` ≡ comportement actuel.
- ✅ Comportement observable **strictement identique** à aujourd'hui en fin de plan.

## Estimation
~moyen côté backend (migration + repo + tests). 1 branche `feat/arc-titles-backend`, 3 commits
(schéma+backfill / repo / point d'extension PP + tests).

## Références
- `internal/prestige/types.go:70` (`Arc.TitleSlug`), composition `TryCompose`
- `internal/migration/` (modèle de migration additive)
- Lié : `[Multi-titre] weapon_family canonical` (`.ai/PLAN_WEAPON_FAMILY_CANONICAL.md`) — même
  déclencheur « 2e titre réel ».
