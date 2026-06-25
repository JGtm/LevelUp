# ADR 0025 — Title-agnostic : fenêtre minimale viable + cible Phase 2 canonical-typée

**Date** : 2026-06-02
**Statut** : Accepté
**Branche** : `refactor/title-agnostic-services` (à créer ; travaux préparatoires sur la branche active en attendant)
**Complète** : [ADR 0011](0011-canonical-vs-semantic-adapter-separation.md) (frontière canonical / semantic / asset-URL)
**Plans liés** : [.ai/V7/PLAN_TITLE_AGNOSTIC_REFACTORING.md](../../.ai/V7/PLAN_TITLE_AGNOSTIC_REFACTORING.md) (master v2.5), [.ai/V7/PLAN_TITLE_AGNOSTIC_TRACKER.md](../../.ai/V7/PLAN_TITLE_AGNOSTIC_TRACKER.md) (suivi)

## Contexte

Un 2e titre est confirmé au backlog produit. Le ROI du refactor title-agnostic passe donc de « marginal » à « décisif » (chaque titre ajouté coûte alors ~1-2 j au lieu de plusieurs semaines). Le master plan v2.5 décrit l'épopée complète (Phases 0→5, 62-82 j). Un audit du code réel (2026-06-02, agents Explore) montre que le code a **avancé** depuis la rédaction du master (2026-05-18) et a **convergé vers un mécanisme différent** de celui que le master prescrivait pour la Phase 2.

Cet ADR acte les **décisions cadrantes** prises avec Guillaume pour démarrer, sans ré-écrire le master.

## Décisions

### D-MV1 — Fenêtre minimale viable = Phases 0→3a

On vise l'état « services title-agnostic + DTO propres + feature-matrix opérationnelle », **Phase 3b (Huma) différée**. C'est la fenêtre mergeable indépendamment (master §11). On ne s'engage pas sur l'épopée complète d'un bloc ; chaque phase reste fermable via son Exit Gate (master §8).

### D-MV2 — Phase 2 : repos canonical-typés, **abandon du FieldKey-map** (supersède D1/D2 du master)

Le master prescrivait un `port.MatchFieldRepository` renvoyant `map[FieldKey]*canonical.Value` (D1 `Value{Kind,...}`, D2 présent/nil/absent). **Le code a convergé vers une autre cible, plus simple et déjà en place** : des repos **canonical-typés** — `PlayerMatchesRepository.LoadPlayerMatches → canonical.PlayerMatchRow` (alias « P4.3 ») — consommés par `synthesis`, `home`, `match_history`, `timeseries` ; `career` via `dataAdapter.LoadCareerSnapshot/LoadEncounters`. **Aucun de ces services n'importe `internal/platform/duckdb` pour la data** (seule exception : `home_service` importe le type `PersistSink`, non-data — à isoler).

**Décision** : la cible Phase 2 devient le **canonical-typé**. On **ne construit pas** `MatchFieldRepository`, son SELECT dynamique, ni le bench D1/D2.

**Conséquences** :
- Le but de découplage (aucun nom de colonne DB ni type Halo dans `service/`/`api/`/`web/`) est tenu par les types canonical, pas par une map de FieldKeys.
- Item « mapping **physique** table→colonne » de la Phase 1 devient **caduc** : `fields.toml` reste **sémantique** (labels/units/format), ce qui suffit.
- Phase 2 résiduelle = finir `explorer`/`career` sur le chemin canonical + isoler le `PersistSink` de `home_service` + (à trancher au cas par cas) les `Load*` encore stubs de l'adapter qu'un service consomme réellement. Pas de réécriture des 5 services déjà canonical.

### D-MV3 — OpenAPI : `PLAN_WEB_API_TYPES_MIGRATION` absorbé dans la Phase 3b (Huma)

La Phase 3b (Huma) **auto-génère** `openapi.yaml` depuis les types Go, d'où se régénère le client TS front. Ça rend caduque la réconciliation **manuelle** schéma-par-schéma de [.ai/V7/PLAN_WEB_API_TYPES_MIGRATION.md](../../.ai/V7/PLAN_WEB_API_TYPES_MIGRATION.md). **Décision** : on ne grind pas la réconciliation manuelle ; la fiabilisation du contrat est un livrable de la Phase 3b. La « fondation posée » de WEB_API_TYPES (pipeline réparé, batch 1) reste acquise ; le plan ne redevient actif que si la Phase 3b glisse > 2 trimestres.

### D-MV4 — Renumérotation des ADR du master

Le master cite des ADR 0014-0017 pour ce chantier ; ces numéros sont **déjà pris** (progression-v2, player-profile, shared-db-provider, rebuild-art) et la numérotation a même des doublons (0020, 0021, 0024). Les ADR de ce chantier prennent les **prochains numéros libres** :
- **0025** (cet ADR) — title-agnostic minimal-viable + cible Phase 2 canonical-typée.
- **0026** — Huma + gel des nouveaux handlers (D13). *À rédiger au démarrage de la Phase 3b.*
- **0027** — Feature-Matrix 3 états + cascade (D7/D8/D12). *À rédiger au démarrage de la Phase 1.7b.*
- **0028** — Title-Diagnostic Lab read-only (D9/D10). *À rédiger au démarrage de la Phase 1.8.*

## Statut des prérequis « 2e titre » (rappel, cf. tracker)

Bloquants réels pour onboarder un 2e titre, **non commencés** : Phase 1.5 (sortir les ~53 `steps_*.go` Halo de `internal/migration/` vers une DDL par-titre — attention à l'ordre `init()` = ordre d'exécution, cf. `registry.go:38`) et Phase 1.6 (clé pool `(titleSlug, gamertag)`). Ces deux phases sont sur le chemin critique du 2e titre et à mener en passe supervisée.

## Anti-décisions (rappel master §10)

- Ne pas pousser `canonical.PlayerMatchRow` directement dans le DTO HTTP (le DTO reste un view-model service composé — frontière ADR 0011).
- Ne pas réintroduire de `if titleSlug == "halo_infinite"` pour gater une feature (→ capability / FeatureChecker).
