# Bilan — Scores d'équipe & Top matchs
_28 mars 2026_

---

## 1. Ce qui est en place (branches)

| Branche | Statut | Contenu |
|---------|--------|---------|
| `fix/top-matches-btb-score-normalization` | ✅ prête, non mergée sur main | Tri normalisé `ABS(diff)/GREATEST(scores)` — Arena et BTB comparables |
| `feat/top-matches-exclude-btb` | ✅ prête, non mergée sur main | Paramètre `career_top_exclude_btb` dans `app_settings.json` |

Les deux branches sont à merger sur `main` quand validé.

---

## 2. Problème ouvert — Inversion team_0/team_1 dans match_registry

### Constat
31.6 % des **victoires** de Madina97294 (149/471) ont `my_team_score < enemy_team_score` dans la vue.  
Ce n'est **pas** un score corrompu (valeur > 100) — c'est une **inversion pure** des colonnes `team_0_score` / `team_1_score` dans l'API Halo.

Exemple vérifié — match `27a69918` (Ranked:Slayer, 31/03/2023) :
- `match_registry` : `team_0_score=24`, `team_1_score=12`
- Madina sur `team_id=1` → `my_team_score=12` alors que son équipe a fait 24 kills
- Les `ps_scores` sont corrects (`team_1_ps_score=3100` = somme des scores perso de team_1 ✓)
- Conclusion : l'API retourne les scores de kills **attribués à l'équipe adverse** dans certains cas

Répartition par mode :
```
BTB         81 matchs affectés
Assassin    39
Other       13
Fiesta      10
Ranked       6
```

### Ce qu'il faudrait faire
Détecter l'inversion en croisant `team_0_score` avec `team_0_ps_score` :
- Si `team_0_ps_score` est cohérent avec les participants de `team_0` mais `team_0_score ≠ somme kills team_0`, les scores sont inversés.
- Alternative plus simple : utiliser `outcome` des participants — l'équipe avec le plus de gagnants devrait avoir le score le plus élevé.

**⚠️ À investiguer avant de coder quoi que ce soit** :
1. Est-ce systématique (toujours inversé pour certains modes/playlists) ou aléatoire ?
2. Le fix doit être dans `match_registry` (backfill UPDATE) ou dans la vue `mv_player_matches` ?
3. La branche `fix/team-scores-ctf-corruption` traitait les scores > 100 — bug différent.

---

## 3. Scores > 100 hors modes objectifs (à vérifier)

Ces modes ont des scores légitimement > 100 (Oddball = secondes de possession, Strongholds = ticks) :
- **Oddball / Ranked:Oddball** : max ~200 — **normal**
- **Strongholds** : max ~200 — **normal**
- **Attrition** : max ~2000 — **normal**
- **KOTH** : score `3075` — **suspect** (3× max habituel ~200)
- **Sentry Defense** : score `7865` — **normal pour ce mode ?** À vérifier
- **Assault / Neutral Bomb** : score `15485`, `9525` — **clairement personal score** (somme de scores perso)
- **CASTLE WARS** : score `8750` — mode custom, règles inconnues

### Ce qu'il faudrait faire
Étendre le filtre de `_score_sql.py` aux modes Assault/Bomb pour les mettre à NULL quand > seuil.
Les modes Oddball/Strongholds/Attrition sont légitimes et ne doivent pas être touchés.

---

## 4. Header "BTB exclus" dans la page Carrière

Actuellement : texte ajouté dans le `st.subheader()` — peu visible.  
**Question en suspens** : format souhaité ? Badge coloré, `st.info()`, ou autre chose ?

---

## 5. Bug CONTRE_REMONTADA — dead code + valeurs stales

**Constat (28 mars 2026)** : match `1561d357` (score 50-13) affichait « domination totale » dans match_view mais « contre remontada » dans le tableau top performance de Madina97294.

**Cause racine** : deux bugs successifs dans l'historique :
- Commit `4c8472c` avait `max_deficit >= 1` → faux positifs massifs (toute victoire dominante avec 1 frag ennemi recevait flag=5)
- La correction ultérieure en `max_deficit >= threshold` rendait le bloc CONTRE_REMONTADA **inaccessible** (dead code) : REMONTADA était testé en premier avec la même condition

**Fix appliqué** (commit `e76f86f`, branche `feat/top-matches-exclude-btb`) : CONTRE_REMONTADA vérifié **en premier** dans `src/analysis/comeback_analysis.py` car c'est le cas le plus spécifique (les deux équipes ont eu une avance de `threshold+` frags).

**Valeurs stales** : les flags=5 incorrects persistent en DB (non corrigés par le backfill normal car `force=False` ne retouche pas les flags 3-5).  
**Action requise** après arrêt de l'app :
```bash
python scripts/backfill_data.py --all --comeback-badges --force-comeback-badges
```

---

## 6. Backfill `comeback_flag` / `domination_flag`

**Contexte** : Depuis v6.2, les badges Remontada/Débandade/Contre-Remontada sont calculés à la volée via `comeback_analysis.py`. Les colonnes `comeback_flag` / `domination_flag` ne sont pas encore persistées dans `player_match_enrichment`, ce qui force un recalcul à chaque affichage.

**Action requise** :
1. Ajouter les colonnes dans `player_match_enrichment` via migration idempotente (`_add_column_if_missing`)
2. Créer l'entrée migration dans `src/data/migration/steps/`
3. Ajouter `--comeback-flags` dans `backfill_data.py` pour écrire les flags sur l'historique
4. Brancher l'écriture dans le flux sync normal après `comeback_analysis`

**Dépendance** : `comeback_backfill.py` (v6.2) — vérifier ce qui est déjà implémenté avant de démarrer.

---

## 6. Adaptation comeback/domination au BTB — seuils en pourcentage

**Contexte** : Les seuils sont actuellement en valeur absolue. En BTB (12v12) les scores sont naturellement plus élevés → faux négatifs/positifs sur les remontadas/débandades.

**Action requise** :
1. Vérifier dans `comeback_analysis.py` si les seuils sont absolus ou relatifs
2. Si absolus : passer en pourcentage du score max de fin de match (ex : `écart > 30% du score cible`)
3. Segmenter par mode si nécessaire (`is_btb` ou via `team_size`)
4. Valider sur des matchs BTB réels

**À vérifier** : si les seuils sont déjà paramétrables, la tâche se réduit à un ajustement de constantes.
