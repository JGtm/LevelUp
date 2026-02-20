# 🚀 Quick Start — Résoudre les Lenteurs v5

> **TL;DR** : La v5 est lente en UI car on a optimisé le backend sans optimiser le frontend. 3 sprints = problème résolu.

## ⚡ Diagnostic en 5 Minutes

### 1. Comprendre le Problème

```
┌────────────────────────────────────┐
│  v5 = Backend ✅ + Frontend ❌     │
├────────────────────────────────────┤
│  Sync : -72% API (SUCCÈS)          │
│  UI   : +200% temps (RÉGRESSION)   │
└────────────────────────────────────┘
```

**Lire** : `.ai/diagnostics/PARADOXE_V5.md` (5 min de lecture)

---

### 2. Mesurer la Baseline

```bash
python scripts/diagnose_performance.py --gamertag JGtm --runs 10
```

**Résultat** : Rapport JSON + résumé console

---

### 3. Choisir Son Niveau d'Action

#### 🟢 Niveau 1 : Quick Win (30 min)

**Action** : Cache repository uniquement

```python
# Dans src/app/data_loader.py
@st.cache_resource
def get_cached_repository(gamertag: str, xuid: str):
    return RepositoryFactory.create(gamertag, xuid)
```

**Gain** : -80% temps connexion (80ms → 15ms)

---

#### 🟡 Niveau 2 : Recommandé (2h)

**Actions** : Cache repository + Vue matérialisée

1. **Cache repository** (30 min)
2. **Vue matérialisée** (1h30)
   ```sql
   CREATE VIEW mv_player_matches AS ...
   ```

**Gain** : -70% temps total

---

#### 🔴 Niveau 3 : Complet (2-3 jours)

**Actions** : 3 sprints complets

1. **Sprint 1** : Vue matérialisée
2. **Sprint 2** : Cache repository
3. **Sprint 3** : Index + Schema cache

**Gain** : -60% à -81% sur toutes les métriques

---

## 📖 Documentation

### Pour Décider

| Fichier | Description | Temps |
|---------|-------------|-------|
| `PARADOXE_V5.md` | Visualisation problème/solution | 5 min |
| `RESUME_EXECUTIF.md` | Synthèse décisionnelle | 10 min |

### Pour Implémenter

| Fichier | Description | Temps |
|---------|-------------|-------|
| `PLAN_OPTIMISATION_V5.md` | Roadmap détaillée | 20 min |
| `DIAGNOSTIC_LENTEURS_V5.md` | Analyse technique | 30 min |

---

## 🎯 Objectifs

```
Métrique             Actuel    Objectif    Après Sprint 2
───────────────────────────────────────────────────────────
Connexion             80ms      <20ms         15ms ✅
load_matches(100)    200ms      <80ms         60ms ✅
Première page       1500ms     <800ms        600ms ✅
```

---

## ✅ Checklist

### Diagnostic
- [ ] Lire `PARADOXE_V5.md` (comprendre)
- [ ] Lancer `diagnose_performance.py` (mesurer)
- [ ] Lire `PLAN_OPTIMISATION_V5.md` (implémenter)

### Implémentation (Niveau 2 Recommandé)
- [ ] Sprint 1 : Vue matérialisée (1h30)
- [ ] Sprint 2 : Cache repository (30 min)
- [ ] Benchmark post-implémentation
- [ ] Valider gains

### Optionnel (Niveau 3)
- [ ] Sprint 3 : Index + Schema cache (1h)
- [ ] Benchmark final
- [ ] Documentation mise à jour

---

## 🆘 Aide Rapide

**Q : Par où commencer ?**  
R : Lire `PARADOXE_V5.md` (5 min) puis lancer le diagnostic.

**Q : Quel niveau choisir ?**  
R : Niveau 2 (cache + vue) = meilleur ROI (2h, -70% gain).

**Q : Combien de temps total ?**  
R : Niveau 2 = 2h. Niveau 3 = 2-3 jours.

**Q : Risques ?**  
R : Faibles. Tests existants + rollback possible.

---

## 🔗 Ressources

- Script : `scripts/diagnose_performance.py`
- Docs : `.ai/diagnostics/`
- Architecture : `docs/ARCHITECTURE_V5.md`

---

**Let's make v5 UI great again! 🚀**
