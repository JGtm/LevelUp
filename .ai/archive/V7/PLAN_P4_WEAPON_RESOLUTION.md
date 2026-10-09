# PLAN_P4_WEAPON_RESOLUTION.md — Router la résolution d'arme par le registre

> Audit fan-out (6 agents, 2026-06-23). 55 sites recensés (beaucoup en doublon entre auditeurs).
> Branche : `feat/weapon-taxonomy-registry`.

## 0. Conclusions de l'audit

1. **Backend-only.** Le front reçoit des noms **déjà résolus** (`label`/`weapon_label`) et fallback sur le `weapon_id` décimal si vide/numérique (`/^\d+$/`). **Aucun changement front.** La parité se joue côté backend (le fallback décimal doit être préservé).
2. **Surface concentrée.** La résolution réelle vit dans ~5 fonctions ; tous les builders (synthesis/timeseries/squad/teammates) sont des **consommateurs pass-through** de `row.Label` → aucun changement si la couche de résolution devient registry-aware.
3. **Fallback `weapon_labels` obligatoire** : sentinels 0/1/2 (grenade/mêlée/véhicule), Mutilator (`0xd791556542c9679f`), Mythic Sandwich, Sandwich, variantes saisonnières — **absents du registre curé (59)**, doivent rester résolus par `weapon_labels` (sinon régression : noms manquants).
4. **Fusion variante→canonique** (`WeaponFusionMapID`) appliquée AVANT le lookup (Duelist Energy Sword → Energy Sword). À préserver.
5. **Locale-aware** : weapon_labels = `COALESCE(name_fr, name_en)`. Le registre a `Name` (modèle EN) + `NameFR`. Le caller doit appliquer la MÊME logique partout (sinon scoreboard FR vs timeseries EN divergent).

## 1. Sites de résolution (les vrais points à migrer)

| Site | Rôle | Risque |
|---|---|---|
| `platform/duckdb/weapon_kills_repo.go::attachWeaponLabels` (L336-386) | nom agrégé → synthesis/timeseries/squad/teammates | **high** |
| `platform/duckdb/match_view_repo_weapons.go::lookupWeaponMeta` (L24-55) + `lookupWeaponLabels` (L57-91) | match-view scoreboard + bulk (Q28) | **high** |
| `platform/duckdb/explorer_repo.go::resolveWeaponLabels` (L542-573) | explorer top weapons | medium |
| `platform/duckdb/home_repo_medals_citations.go::LoadFavoriteWeapon` (Q26k) | carte arme favorite home | low |
| `platform/duckdb/weapon_kills_repo.go::attachWeaponRoles` (L394-439) | rôle (DÉJÀ registre) | — fait |
| `analysis/weapon_data.go::WeaponIDToName` → `weapon_scanner.go:253` + `sync/backfill_weapons.go` + `cmd/diag_film` | film analysis HINF (nom filmshell) | medium |
| `sync/citations.go::loadWeaponNames` (name_en pour `weapon_kills:<name>`) | citations | high (cohérence name_en) |
| `cmd/diag_weapons*` (4) | CLI debug | low |
| `metadata_repo_assets_list.ListWeaponsByTitle` + `api/server.go` asset catalog | recherche/icônes | medium |

Consommateurs **inchangés** (pass-through `row.Label`) : `buildTopWeapons` (timeseries), `buildTopWeaponKills` + `buildKillsByRole` (synthesis), `BuildWeaponsTable` (squad), `buildSquadWeaponKills` (teammates).

## 2. DÉCISION BLOQUANTE — politique de nommage

Router les noms par le registre **change des noms affichés**, car `registry.name_fr` ≠ `weapon_labels.name_fr` :

| Affiché aujourd'hui (weapon_labels) | registry.name (modèle) | registry.name_fr (traduit) |
|---|---|---|
| BR75 | BR75 | Fusil de combat |
| MA40 AR | MA40 AR | Fusil d'assaut |
| Pistolet à plasma | Plasma Pistol | Pistolet à plasma |
| Crémateur | Cindershot | **Crémator** (⚠ typo vs « Crémateur ») |
| Rayon de Sentinelle | Sentinel Beam | Laser de Sentinelle |
| Épée à énergie | Energy Sword | Épée à énergie |

→ **Aucun champ du registre ne donne une parité exacte** avec l'affichage actuel (weapon_labels est un MIX modèle + quelques traductions). Donc P4 = décision de contenu, pas refacto mécanique.

### Options
- **A — Parité (noms inchangés)** : le registre devient le **point d'entrée unique** + apporte les **dimensions** (rôle/famille/faction), mais l'affichage des noms reste celui de weapon_labels. Zéro surprise. Le registre n'est pas encore l'autorité de nommage.
- **B — Registre = autorité de nommage** : bascule l'affichage sur `registry.name_fr`. Change l'affichage **partout**. Pré-requis : **passe de validation des `name_fr`** (corriger typos type « Crémator », trancher modèle-vs-type, vérifier Laser/Rayon de Sentinelle).

**Décision user : ___ (à remplir)**

## 3. Implémentation (une fois la politique choisie)
- Helper unique `resolveWeaponLabelsRegistryFirst(ctx, meta, titleSlug, ids)` (LEFT JOIN `weapon_ids`/`weapons` + fallback `weapon_labels` dans la même requête, COALESCE selon la politique choisie + locale). Tous les sites §1 l'appellent.
- Sentinels 0/1/2 + variantes hors registre → restent sur weapon_labels (le COALESCE final les couvre).
- Garde-fou : registre nil / tables absentes → fallback weapon_labels intégral (best-effort, jamais de panic).
- Data gap à traiter : Mutilator / Mythic Sandwich / Sandwich (3 filmshell ids dans `weapon_data.go`, absents du registre) → soit les ajouter au seed, soit les laisser sur le fallback weapon_labels (sans régression).
- Tests golden parity : pour un échantillon d'ids, le label résolu doit être IDENTIQUE à l'ancien (option A) ou conforme à la table validée (option B).
