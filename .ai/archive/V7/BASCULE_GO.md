# Procédure de bascule Go — LevelUp (Sprint 36)

> **Objectif** : documenter la procédure de mise en production du backend Go,
> les critères de validation, et le plan de rollback en cas d'incident.

---

## Critères de bascule (Gate Phase 7)

Tous les critères suivants **doivent** être verts avant d'activer Go en production :

| # | Critère | Vérification | Statut |
|---|---------|:---:|:-----:|
| 1 | `parity_check.py` = 0 diff sur 24 endpoints | `python scripts/parity_check.py` | ⬜ CI requis |
| 2 | 15 specs Playwright = vert | `npm run e2e` | ⬜ CI requis |
| 3 | Onboarding E2E (auth → home) = vert | `npx playwright test slice-9` | ⬜ CI requis |
| 4 | Sécurité (CSRF, pool, errors) = OK | `go test ./internal/api/middleware/...` | ✅ |
| 5 | Infra Docker + healthcheck + Makefile OK | `docker compose up --build` | ✅ |
| 6 | Couverture Go ≥ 50% | `make go-api-coverage` | ⬜ CI requis |

> **Note Sprint 49** : les critères 1–3 et 6 nécessitent un environnement CI avec CGO (DuckDB),
> Node.js (Playwright) et données de test. Le script `apps/go-api/scripts/parity_check.py`
> couvre 24 endpoints (8 golden values, 16 status-only). Le contrat OpenAPI ↔ chi est vérifié
> par `TestContractRoutesRegistered` (0 exemption depuis Sprint 49).

---

## Procédure de bascule

### Étape 0 — Prérequis

```bash
# Vérifier que la branche sprint est à jour
git log --oneline -5

# Healthcheck infra
docker compose ps
curl -sf http://localhost:8000/health | python -m json.tool
```

### Étape 1 — Validation finale avant bascule

```bash
# 1. Parity check complet (24 endpoints)
python apps/go-api/scripts/parity_check.py \
  --go-url http://production-host:8000 \
  --player VOTRE_GAMERTAG

# 2. Tests E2E Playwright
cd apps/web && npm run e2e

# 3. Tests sécurité Go
cd apps/go-api && go test ./internal/api/middleware/... -v

# 4. Couverture (≥ 50%)
make go-api-coverage
```

### Étape 2 — Activation du feature flag Go

Le feature flag se configure via la variable d'environnement `LEVELUP_BACKEND`.

Dans `.env.local` (ou `docker-compose.yml`), ajouter ou modifier :

```env
# Backend actif : "go" | "python" (défaut "go" depuis Sprint 36)
LEVELUP_BACKEND=go
```

Dans `docker-compose.yml`, le service `levelup` utilise déjà le binaire Go.
Aucune modification de `docker-compose.yml` n'est nécessaire — la bascule
est actée par la suppression du service `fastapi` si encore présent :

```yaml
# Si un service FastAPI existe encore dans docker-compose.yml :
# Commenter ou supprimer le bloc 'fastapi:' pour désactiver Python.
# Le garder en commentaire pendant 2 semaines (garde-fou rollback).
```

### Étape 3 — Déploiement

```bash
# Build et démarrage
docker compose up -d --build levelup

# Vérifier le démarrage (logs 30 dernières secondes)
docker compose logs --since 30s levelup

# Healthcheck post-déploiement
sleep 10 && curl -sf http://localhost:8000/health | python -m json.tool
```

### Étape 4 — Monitoring 48h

Surveiller pendant 48h après la bascule :

```bash
# Logs d'erreur en temps réel
docker compose logs -f levelup | grep -E "ERROR|WARN|shadow"

# Métriques shadow mode (si activé)
docker compose logs levelup | grep "shadow: divergence" | wc -l

# Healthcheck automatisé (toutes les 5 min)
watch -n 300 'curl -sf http://localhost:8000/health'
```

**Critères de revert** : revert immédiat si l'un des suivants survient :
- Taux d'erreur HTTP 5xx > 1% en 10 min
- Temps de réponse médian > 2× la baseline Python
- Divergences shadow mode sur > 5% des requêtes
- Crash ou `OOMKilled` du container

---

## Plan de rollback

### Rollback immédiat (< 5 min)

En cas d'incident critique, le rollback se fait en réactivant le service Python
dans `docker-compose.yml` :

```bash
# 1. Inverser la modification docker-compose.yml (décommenter service fastapi)
git stash  # si non commité, ou :
git revert HEAD --no-commit

# 2. Redémarrer avec l'ancien backend
docker compose up -d levelup

# 3. Vérifier healthcheck
curl -sf http://localhost:8000/health
```

### Si docker-compose.yml a été modifié (service Python supprimé)

```bash
# Revenir au commit avant la bascule
git log --oneline | grep "sprint/30-31" | head -5
git checkout <COMMIT_AVANT_BASCULE> -- docker-compose.yml

# Ou restaurer depuis le backup
cp docker-compose.yml.bak docker-compose.yml

docker compose up -d --build
```

### Rollback complet (retour branche main pre-migration)

```bash
# Identifier le dernier commit stable pre-migration
git log --oneline main | head -10

# Créer une branche hotfix depuis main
git checkout main
git checkout -b hotfix/rollback-to-python

# Déployer
docker compose down
docker compose up -d --build
```

---

## Conservation du backend Python (2 semaines)

Le service FastAPI doit rester **en commentaire** dans `docker-compose.yml`
pendant **2 semaines** après la bascule Go, pour permettre un rollback rapide
sans rebuild.

Après 2 semaines sans incident : supprimer le service Python du compose.

---

## Contacts & escalade

| Situation | Action |
|-----------|--------|
| 5xx > 1% | Rollback immédiat, créer issue `bug/critical` |
| Divergence shadow > 5% | Analyser les logs, créer issue `bug/parity` |
| Healthcheck KO | Vérifier DuckDB path + `LEVELUP_REPO_ROOT` |
| OOM | Réduire `LEVELUP_POOL_SIZE` + redémarrer |

---

## Historique

| Date | Action | Auteur |
|------|--------|--------|
| 2026-07-20 | Document créé (Sprint 36) | Copilot |
| 2026-07-25 | Sprint 49 — Contrat OpenAPI aligné (0 exemption), `POST /session/context` enrichi, `JobMeta` structuré, ADR routage confirmée définitive. Critères 1-3/6 en attente CI. | Copilot |
| 2025-12-01 | Sprint 51 — Critère 4 (sécurité) ✅ branche `feat/sprint-55-career-synthesis-ux` ; critère 5 (docker-compose healthcheck `levelup-server` Go natif) ✅ vérifié. Items 1-3/6 en attente CI. | Copilot |
