# Plan : Import depuis une BDD OpenSpartan

> **Statut** : Plan d'implémentation futur, indépendant.
> **Dépendance** : profite du SSO Xbox (`SPRINT_XBOX_SSO.md`) pour valider l'identité, mais peut techniquement vivre sans.
> **Branche cible** : à créer (`feat/openspartan-import`), depuis `main`.
> **Auteur du plan** : Claude (session du 2026-05-16).

---

## 0. Contexte

**OpenSpartan** (`github.com/OpenSpartan/openspartan` et `OpenSpartan/grunt`) est un client Halo Infinite tiers qui maintient sa propre BDD SQLite locale par utilisateur. Beaucoup de joueurs Halo l'ont utilisé avant LevelUp et possèdent un historique de matchs important — souvent **plus ancien** que ce que l'API Halo de Microsoft expose actuellement (l'API tronque l'historique au-delà d'une fenêtre récente, le nombre exact dépendant du mode de jeu).

**Use case** :
1. User s'inscrit sur LevelUp via SSO Xbox.
2. Au lieu d'attendre 30 min de sync initial (et de perdre les matchs vieux que l'API ne renvoie plus), il importe son `.osdb` OpenSpartan.
3. LevelUp mappe les matchs OpenSpartan vers le schéma DuckDB v6 et les insère dans `shared_matches_v2`.

---

## 1. Décision design

### Quoi importer depuis OpenSpartan ?

| Donnée OpenSpartan | Cible LevelUp | Priorité |
|---|---|---|
| Match registry (id, mode, map, dates) | `shared.match_registry` | P0 |
| Match participants (stats par joueur) | `shared.match_participants` | P0 |
| Medals earned | `shared.medals_earned` | P1 |
| Highlight events | `shared.highlight_events` | P1 |
| Killer/victim pairs | `shared.killer_victim_pairs` | P1 |
| Service record cumulatifs | **Skip** (recalculé depuis matchs) | — |
| Career progression historique | `player.career_progression` | P2 |
| Médias (screenshots) | **Skip** (pas dans OpenSpartan en standard) | — |

### Stratégie de conflit

- **Match déjà présent dans `shared.match_registry`** → skip silencieux (l'API est source de vérité pour les matchs récents).
- **Match présent uniquement dans OpenSpartan** → insert.
- **Stats joueur incomplètes côté OpenSpartan** (ex: shots_fired manquant) → insert quand même, les colonnes manquantes restent NULL, le sync API peut les compléter plus tard.

---

## 2. PR 1 — Reader OpenSpartan (SQLite isolé)

**Périmètre** : module dédié qui lit le SQLite OpenSpartan et expose des structures Go canoniques.

- Nouveau package `apps/go-api/internal/import/openspartan/`
  - `reader.go` : ouvre le `.osdb` (SQLite read-only) — **seule exception à la règle "SQLite interdit"** de CLAUDE.md, à documenter en tête de fichier
  - `schema.go` : définit les tables OpenSpartan attendues + détection de version
  - `models.go` : structs Go (`OSMatch`, `OSParticipant`, `OSMedal`) qui mappent les rows OpenSpartan
- Tests : fixture `.osdb` minimal dans `testdata/` (5 matchs, 2 joueurs, quelques médailles)
- Couvrir au moins 2 versions de schéma OpenSpartan (le projet a évolué, prévoir un `if schema_version < X`)

---

## 3. PR 2 — Mapper OpenSpartan → LevelUp v6

**Périmètre** : transformation `OSMatch → domain.SharedMatch`, etc.

- Nouveau package `apps/go-api/internal/import/openspartan/mapper/`
  - `match.go` : `func MapMatch(os OSMatch) (domain.SharedMatch, []domain.MatchParticipant, error)`
  - `medal.go` : mapping des `medal_id` OpenSpartan (qui sont stables, fournis par Microsoft) vers `medals_earned`
  - `mode.go` : utiliser la skill `halo-modes` pour normaliser les modes
- Edge cases :
  - Matchs avec mode inconnu (DLC futur ?) → log warning + skip
  - Stats manquantes (champs NULL côté OpenSpartan) → insert avec NULL
  - Timestamp dans le futur (corruption) → skip
- Tests : pour chaque type de match (matchmaking 4v4, btb, ranked, custom), vérifier le mapping.

---

## 4. PR 3 — Validation XUID + service d'import

**Périmètre** : sécurité + orchestration.

- Service `apps/go-api/internal/service/openspartan_import_service.go`
  - `func ImportFromOpenSpartan(ctx, xuid, osdbPath) (ImportResult, error)`
  - Vérifie que le `.osdb` contient bien le XUID de l'utilisateur authentifié (sinon refuse — empêche d'importer la BDD d'un autre)
  - Stream-insertion vers DuckDB en batch (1000 rows) — pas tout charger en RAM
  - Dry-run mode : compte ce qui *serait* importé sans écrire
- Endpoint `POST /import/openspartan` :
  - Upload multipart (le `.osdb` peut faire 50-500 MB)
  - Renvoie `ImportResult { added_matches, skipped_existing, errors }`
  - Long-running : utiliser le système de jobs existant (cf. `useJobToasts` côté frontend)
- Sécurité :
  - Quota taille upload (1 GB max ?)
  - Le fichier est traité puis supprimé (pas stocké)
  - Le service tourne avec les perms de l'user connecté uniquement — pas de bypass admin pour importer la BDD d'un autre

---

## 5. PR 4 — UI d'import (mode avancé)

**Périmètre** : exposer l'option dans le flow d'onboarding + settings.

- Étape optionnelle après inscription Xbox : "Tu as déjà utilisé OpenSpartan ? Importe ton historique."
- Composant `OpenSpartanImportCard` dans `SettingsPage` (tab Sync ou nouvelle tab Import)
- Drag & drop d'un fichier `.osdb`
- Affiche le résultat : "X matchs importés, Y déjà connus, Z erreurs (cliquer pour détails)"
- Tests : composant avec MSW + fixture `.osdb` minimal

---

## 6. Pièges à éviter

1. **Schema drift OpenSpartan** : le projet évolue, prévoir une détection de version et des messages clairs si le schéma n'est pas reconnu ("votre `.osdb` vient d'une version d'OpenSpartan trop récente/ancienne").
2. **Validation XUID stricte** : sinon n'importe qui peut importer la BDD de n'importe qui d'autre et polluer son historique.
3. **Pas de fusion automatique** des médailles si l'utilisateur réimporte deux fois — `medal_id + match_id` doit être une clé unique pour éviter les doublons.
4. **Performance** : 10 000+ matchs importés = quelques minutes. Le faire en job background avec progress visible, pas dans le request handler.
5. **Backup avant import** : proposer (ou forcer) une sauvegarde du `stats.duckdb` avant l'opération, pour pouvoir rollback si l'import corrompt quelque chose. Tu as déjà l'infra backup ([cf. README sur backup_player.py](README.md)).
6. **`shared` ou `player` ?** : les matchs OpenSpartan vont dans `shared.match_*` (centralisé). Bien faire attention à ne PAS écrire dans `data/players/{gamertag}/stats.duckdb` (qui ne contient plus de tables matchs depuis v5.1).

---

## 7. Estimation

| PR | Effort | Bloque la suite ? |
|---|---|---|
| PR 1 (reader SQLite) | 1j | Oui |
| PR 2 (mapper) | 1.5j | Oui (dépend PR 1) |
| PR 3 (service + endpoint) | 1j | Oui (dépend PR 1+2) |
| PR 4 (UI) | 0.5j | Non (peut être stub `curl` au début) |

**Total** : ~4 jours de dev focused.

---

## 8. Avant de démarrer

- [ ] Récupérer 2-3 `.osdb` réels (le tien + un autre joueur) pour tester sur des données vraies.
- [ ] Vérifier la version actuelle du schéma OpenSpartan (`PRAGMA user_version` ou table de migrations dans le SQLite).
- [ ] Confirmer que les `match_id` OpenSpartan correspondent bien aux `match_id` Halo Infinite (UUIDs Microsoft) — sinon il faut un mapping.
- [ ] Décider du quota taille upload max.
