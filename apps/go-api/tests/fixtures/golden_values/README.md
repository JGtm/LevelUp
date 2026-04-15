# Golden Values — Corpus de référence LevelUp Go

> **Sprint 2** — Corpus d'oracle pour la parité des endpoints Go vs Python.
> Date de création : 2026-04-15

---

## Rôle

Ces fichiers JSON sont les **golden values** : les réponses de référence attendues
de chaque endpoint. Ils servent d'oracle pendant tout le portage Go pour vérifier
que le backend Go retourne exactement les mêmes données que le Python.

---

## Fichiers présents

| Fichier | Endpoint | Sprint cible |
|---------|----------|:---:|
| `health_ok.json` | `GET /health` | Sprint 4 |
| `bootstrap_no_auth.json` | `GET /api/v1/bootstrap` | Sprint 6 |
| `players_list.json` | `GET /api/v1/players` | Sprint 6 |
| `filters_resolve_all.json` | `POST /api/v1/players/{slug}/filters/resolve` (sans filtre) | Sprint 6 |
| `filters_resolve_zero_matches.json` | POST filters/resolve — cas 0 match | Sprint 6 |
| `career_page_chocoboflor.json` | `GET /api/v1/players/Chocoboflor/pages/career` | Sprint 6 |
| `match_history_page1_nofilter.json` | `POST /pages/match-history/query` (page 1, sans filtre) | Sprint 6 |
| `gamertag_search_cho.json` | `GET /directory/gamertags/search?q=cho` | Sprint 8 |
| `gamertag_search_empty.json` | GET gamertags/search — cas 0 résultat | Sprint 8 |
| `match_view_slayer.json` | `GET /players/{slug}/matches/{match_id}` — Slayer | Sprint 8 |

---

## Statut des fixtures

| Statut | Description |
|--------|-------------|
| `schema-conformant` | Construit manuellement, conforme au schéma, **pas capturé depuis l'API réelle** |
| `captured_live` | Capturé depuis l'API Python en fonctionnement — valeurs réelles |

Les fixtures actuelles sont `schema-conformant`. Elles deviennent la référence de forme.
Avant le Sprint 6, remplacer par des fixtures `captured_live` via `apps/go-api/scripts/capture_golden_values.py`.

---

## Générer des golden values réelles

```bash
# 1. Démarrer l'API Python
cd apps/api
pip install -e ".[dev]"
uvicorn app.main:app --port 8000

# 2. Dans un autre terminal, lancer la capture
cd apps/go-api
python apps/go-api/scripts/capture_golden_values.py --player Chocoboflor

# 3. Vérifier les fichiers générés
ls tests/fixtures/golden_values/*.json
```

La capture remplace les fixtures existantes avec des valeurs réelles (source = `captured_live`).

---

## Utilisation dans les tests Go

Les tests de parité (Sprint 6+) chargeront ces fixtures et compareront avec la réponse Go :

```go
// Exemple d'usage attendu en Sprint 6
func TestBootstrapParite(t *testing.T) {
    golden := loadGolden(t, "bootstrap_no_auth.json")
    resp := callBootstrap(t)
    assertParite(t, golden, resp, golden.Meta.Tolerances)
}
```

---

## Tolérances

Chaque fixture contient un bloc `_meta.tolerances` qui liste les champs
dont la valeur exacte n'est pas exigée (ex. : compteurs variables, dates).

Le comparateur Go doit respecter ces tolérances :
- Champ absent de `tolerances` → comparaison exacte
- Champ présent → comparaison lâche (ordre, ±delta numérique, etc.)

---

## Cas limite couverts

| Cas | Fichier |
|-----|---------|
| 0 matchs après filtre | `filters_resolve_zero_matches.json` |
| Recherche gamertag sans résultat | `gamertag_search_empty.json` |
| Carrière sans charts (null) | `career_page_chocoboflor.json` |
| Bootstrap sans auth | `bootstrap_no_auth.json` |

**Cas à ajouter avant Sprint 6 (capture live) :**
- Joueur sans médailles
- Match PvE (Firefight)
- Match avec bot coéquipier
- Historique avec filtre de sessions

---

## Règles de maintenance

1. **Ne jamais committer de clés/tokens** dans les fixtures.
2. **Ne jamais hardcoder les XUIDs réels** dans les schema-conformant — utiliser des valeurs synthétiques.
3. Après chaque sync ou refactoring qui change la forme d'une réponse → recapturer les golden values.
4. Les futures fixtures `captured_live` priment sur les `schema-conformant` pour la validation de parité réelle.
