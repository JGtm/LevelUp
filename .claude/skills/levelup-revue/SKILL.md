---
name: levelup-revue
description: Complément LevelUp des skills adversarial-review et adversarial-audit (calibrage par dossier, lentilles et axes maison anti-ART, multi-titre, couches, données, front, doctrine de référence, dette assumée, emplacement du registre). À lire avec adversarial-review avant de relire un diff LevelUp, et avec adversarial-audit avant d'auditer du code LevelUp.
---

# LevelUp : complément des revues et des audits

Ce skill ne remplace pas `adversarial-review` ni `adversarial-audit`. Il fournit ce qu'un relecteur générique ne connaît pas : les pièges de LevelUp.

Chaque lentille ci-dessous se colle **telle quelle** dans le prompt d'un relecteur ou d'un auditeur, une par sous-agent.

## 1. Calibrage LevelUp

| Le diff touche… | Revue |
|---|---|
| `persist/`, `sync/`, `migration/`, dblease, tables append-only | Obligatoire, lentille L1, relecteur Sonnet ciblé |
| `platform/auth`, jetons, mise en commun (`Pool`), ownership, `PlayerDirectory` | Obligatoire, relecteur ciblé |
| Nouvelle capability, nouvel adapter, frontière multi-titre | Obligatoire, lentille L2 |
| Indicateur, LUSR, note de performance, agrégat | Obligatoire, lentille L4 |
| Page, graphe, hook web | Lentille L5, en fin de jalon |

## 2. Lentilles et axes

Les lentilles L1 à L5 servent à la revue d'un diff. Elles servent aussi d'axes d'audit, sous les noms A-ART, A-MULTI, A-COUCHES, A-DONNEES et A-FRONT.

**L1 / A-ART : écritures DuckDB (bug ART #23645).**
- Toute écriture per-match sur une base partagée passe par `persist.BatchBuilder.Submit()` puis un `Persister`, en INSERT seulement.
- Pas d'UPSERT ni de `ON CONFLICT DO UPDATE` concurrent sur les tables critiques.
- Toute lecture d'une table append-only passe par la vue `<table>_latest` : une lecture brute sert des lignes périmées.
- Un seul process écrivain par base : jamais une ouverture RO et une RW sur le même fichier.
- Pour lire une base qu'un autre peut tenir en RW : `OpenReadForQuery`, jamais `OpenReadOnly` forcé.
- `duckdb.OpenReadWrite` n'est appelé que depuis la liste datée de l'ADR 0030.
- `shared_social` : `SharedSocialPersister` suivi d'un `CHECKPOINT`.
- Résumés du journal des morts : écrits par `insertKillSummaryRows` dans la transaction de leur passe (ADR 0039).
- Un garde-fou élargi (liste d'exceptions, motif assoupli) sans justification datée est un P0.

**L2 / A-MULTI : multi-titre.**
- Jamais `slug == "..."` : brancher sur `HasCapability` / `CapabilityMap` (cliquet `no_slug_comparison_test.go`).
- Jamais `filepath.Join(..., "data", ...)` : tout chemin passe par `PathResolver`.
- Aucun libellé FR ou EN en dur côté Go : `TitleSemanticAdapter` + TOML, ou catalogue `i18n.Msg`.
- Une capability absente produit `ErrCapabilityNotSupported` géré, jamais une panique ni les données d'un autre titre.
- Aucun import d'un paquet `internal/games/<titre>/` dans `analysis/`, `service/` ni `api/handlers`.

**L3 / A-COUCHES : frontières de couches.**
- Les gardes de `internal/archlint/` (table « Frontières de couches » de `CLAUDE.md`) : pas de SQL dans `api/`, `service/`, `analysis/` ; handlers sans `sync`, `persist` ni `platform/duckdb`.
- `os.Getenv` seulement dans `internal/config` et `cmd/`.
- Absence de ligne : `errors.Is(err, port.ErrNotFound)`, jamais `sql.ErrNoRows`.
- Web : imports entre features (`tools/lint-cross-feature-imports.mjs`), `echarts` seulement sous `components/charts/`, `fetch` seulement dans `lib/api/` et `lib/desktop/`, aucun `useQuery` dans un `.tsx` de `features/`.
- Les 10 anti-patterns de `CLAUDE.md` : code mort, compatibilité sans date, fichier de plus de 500 lignes ou fonction de plus de 80, littéral répété, ouverture DuckDB directe, nombre magique, logique métier dans un handler ou un composant, factorisation abandonnée, doc inversée, erreur ignorée.

**L4 / A-DONNEES : justesse des données.**
- Le KDA n'est jamais le quotient. Par match : la valeur native de l'API. Agrégat : `((frags + assists/3) - morts) / nb_matchs`. Le KDR (frags/morts) est distinct (ADR 0006). À l'écran : « FDA ».
- Tout filtre ou tri temporel passe par `COALESCE(x.start_time_utc, x.start_time AT TIME ZONE 'UTC')`, via `sqlfrag`.
- Les lecteurs de notes lisent les vues `_latest`. LUSR v2 seulement.
- Formule de LUSR ou de note de performance changée : la constante de révision (`PerformanceScoreRev` ou `LUSRRev`, ADR 0047) change dans le même commit.
- Modes à manches : le score affiché vient des manches (`analysis.ReadTeamScore`, ADR 0032).
- Les agrégats réutilisent les KPI dérivés existants. Unités 0..1 cohérentes de la base à l'UI.
- Les bots comptent partout.

**L5 / A-FRONT : web.**
- Aucune valeur hex ni classe Tailwind de couleur dans `features/` ou `components/` : tokens sémantiques seulement.
- Chaque texte existe dans chaque langue de `config/locales.toml`. Jamais `=== 'fr'` ni de liste de langues en dur (ADR 0040).
- Query keys dans `lib/query/keys.ts`. Tableaux par TanStack Table avec `useColumnSort`.
- `routeTree.gen.ts` n'est jamais édité à la main.
- FR sans anglicismes. Pas de bordure colorée à gauche d'une fiche.

## 3. Doctrine de référence pour un audit

Extraits à coller selon l'axe :
- `CLAUDE.md` : « Règles » (1 à 19), « Règles critiques — écritures DuckDB », « Frontières de couches », « Diagnostic de revue de code ».
- ADR (`docs/adr/`) les plus souvent concernées : 0006 (indicateurs), 0008 (chemins), 0013 et 0016 (écrivain unique), 0019, 0026 et 0030 (anti-ART), 0023 et 0038 (jetons), 0025 (title-agnostic), 0035 (annuaire des joueurs), 0036 (lectures par périmètre), 0040 (i18n), 0047 (révision des notes).
- Le skill du domaine : `arch-rules`, `db-schema`, `canonical-types`, `frontend-patterns`, `color-tokens`.

## 4. Dette connue et assumée : ne pas remonter

À coller dans chaque prompt d'auditeur. À relire avant chaque audit, car ce bloc vieillit.

```
DETTE CONNUE ET ASSUMÉE — NE PAS REMONTER
- Baseline golangci-lint : la dette existante est gelée ; ce n'est pas un constat.
- Les exemptions de seuil qui portent un commentaire de justification sont valides.
- Player DBs legacy sans contraintes : le motif SELECT-then-UPDATE-or-INSERT est la
  solution retenue, pas un défaut.
- Les entrées d'allowlist datées et justifiées des gardes (archlint, sync, platform/duckdb)
  sont valides.
- Décodeur de film (brique publique) : un champ que LevelUp ne lit pas y est normal.
```

## 5. Registre d'audit

Emplacement : `.ai/AUDIT_<PERIMETRE>_<AAAA-MM-JJ>.md`. Entrée au journal `.ai/thought_log.md`. Les P2 retenus vont dans `.ai/BACKLOG.md`.
