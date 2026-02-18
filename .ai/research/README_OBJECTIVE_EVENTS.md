# Index : Tracking des Événements d'Objectifs

Ce dossier contient la recherche complète sur le tracking des événements d'objectifs Halo Infinite via l'API SPNKr.

---

## 📌 Points d'Entrée

### Pour une réponse rapide
👉 **Lire** : [`REPONSE_UTILISATEUR_OBJECTIFS.md`](./REPONSE_UTILISATEUR_OBJECTIFS.md)

### Pour les détails techniques
👉 **Lire** : [`OBJECTIVE_EVENTS_TRACKING.md`](./OBJECTIVE_EVENTS_TRACKING.md)

### Pour une référence rapide
👉 **Lire** : [`OBJECTIVE_EVENTS_SUMMARY_FR.md`](./OBJECTIVE_EVENTS_SUMMARY_FR.md)

---

## 📊 Réponse Courte

**Question** : "Est-ce qu'on peut déterminer quand notre équipe ou l'équipe adverse a capturé le drapeau ?"

**Réponse** :
- ✅ **OUI** pour savoir **QUI** et **QUELLE ÉQUIPE** a capturé
- ✅ **OUI** pour savoir **COMBIEN DE FOIS**
- ❌ **NON** pour savoir **EXACTEMENT QUAND** (pas de timestamp précis)
- ⚠️ **ESTIMATION POSSIBLE** via corrélation avec kills/deaths (±30s)

---

## 🎯 Modes Supportés

| Mode | Événements | Status |
|------|-----------|--------|
| **CTF** | 6 types (Flag Captured, etc.) | ✅ Complet |
| **Strongholds** | 4 types (Zone Captured, etc.) | ✅ Complet |
| **Oddball** | 3 types (Ball Control, etc.) | ✅ Complet |
| **Assault** | 0 type | ❌ Non documenté |

---

## 💻 Code Implémenté

### Service
- **Fichier** : `src/data/services/objective_events_service.py`
- **Fonctions** : 9 fonctions d'extraction
  - `get_flag_captures_by_team()`
  - `get_flag_captures_by_player()`
  - `get_base_captures_by_team()`
  - `get_base_captures_by_player()`
  - `get_oddball_control_by_team()`
  - `get_oddball_control_by_player()`
  - `get_objective_events_by_team()`
  - `get_objective_mvp()`

### Tests
- **Fichier** : `tests/test_objective_events_service.py`
- **Couverture** : 9 tests unitaires avec mocks

---

## 📚 Structure de la Recherche

```
.ai/research/
├── OBJECTIVE_EVENTS_TRACKING.md        # Documentation complète (18 KB)
│   ├── Analyse des capacités API
│   ├── Événements disponibles par mode
│   ├── Exemples de requêtes SQL
│   ├── Cas d'usage et visualisations
│   └── Roadmap d'implémentation
│
├── OBJECTIVE_EVENTS_SUMMARY_FR.md      # Résumé exécutif (6 KB)
│   ├── Réponse rapide
│   ├── Exemples de requêtes
│   ├── Plan d'action
│   └── FAQ
│
├── REPONSE_UTILISATEUR_OBJECTIFS.md    # Réponse finale (9 KB)
│   ├── Réponse directe à la question
│   ├── Fichiers livrés
│   ├── Exemples d'utilisation Python
│   ├── Visualisations proposées
│   └── Prochaines étapes
│
└── README_OBJECTIVE_EVENTS.md          # Ce fichier
```

---

## 🚀 Comment Utiliser le Service

### Exemple 1 : CTF - Qui a capturé des drapeaux ?

```python
from src.data.repositories.duckdb_repo import DuckDBRepository
from src.data.services.objective_events_service import get_flag_captures_by_player

repo = DuckDBRepository("data/players/JGtm/stats.duckdb", "xuid123")
df = get_flag_captures_by_player(repo, "match_abc123")

print(df)
```

### Exemple 2 : Strongholds - Captures par équipe

```python
from src.data.services.objective_events_service import get_base_captures_by_team

df = get_base_captures_by_team(repo, "match_sh_123")
print(df)
```

### Exemple 3 : MVP Objectifs

```python
from src.data.services.objective_events_service import get_objective_mvp

mvp = get_objective_mvp(repo, "match_abc123")
print(mvp)
```

---

## 📖 Lire les Documents dans l'Ordre

1. **Débutant** : Lire [`REPONSE_UTILISATEUR_OBJECTIFS.md`](./REPONSE_UTILISATEUR_OBJECTIFS.md) en premier
2. **Développeur** : Consulter [`OBJECTIVE_EVENTS_SUMMARY_FR.md`](./OBJECTIVE_EVENTS_SUMMARY_FR.md) pour les références
3. **Expert** : Approfondir avec [`OBJECTIVE_EVENTS_TRACKING.md`](./OBJECTIVE_EVENTS_TRACKING.md) pour tous les détails

---

## 🛠️ Prochaines Étapes

### Phase 1 : Visualisations (À faire)
- [ ] Créer `src/visualization/objective_timeline.py`
- [ ] Graphiques Plotly pour CTF, Strongholds, Oddball

### Phase 2 : UI Streamlit (À faire)
- [ ] Créer `src/ui/pages/objective_events_details.py`
- [ ] Intégration au menu principal

### Phase 3 : Estimation Timestamps (Optionnel)
- [ ] Corrélation avec `highlight_events` pour timeline approximative

---

## ❓ Questions Fréquentes

**Q : Peut-on avoir le timestamp exact d'une capture ?**  
R : Non, les PersonalScores n'ont pas de champ `time_ms`. Estimation possible via corrélation.

**Q : Pourquoi pas d'événements Assault ?**  
R : Non documentés dans l'enum `PersonalScoreNameId`. Investigation nécessaire (API Grunt).

**Q : Les données sont-elles fiables ?**  
R : Oui, elles proviennent directement de l'API officielle Halo Waypoint.

---

*Analyse réalisée le 2026-02-18 suite à la question utilisateur sur le tracking des captures*
