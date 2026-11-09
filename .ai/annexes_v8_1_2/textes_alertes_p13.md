# Clés et textes exacts des alertes de comportement (annexe du plan v8.1.2, item 2.1)

Fichier cible : `apps/go-api/internal/i18n/messages/patterns.toml`, suivre le format des fichiers voisins. Les paramètres numériques sont pré-formatés par
`fmt.Sprintf` aux précisions d'aujourd'hui, indiquées ci-dessous. Le séparateur décimal reste le point.

| Clé (`patterns.behavior.`) | fr | en | Paramètres |
|---|---|---|---|
| `tilt.trigger` | `{n}+ défaites consécutives` | `{n}+ consecutive losses` | `n` entier |
| `tilt.evidence` | `FDA moyen {before} → {during}` | `Average KDA {before} → {during}` | `%.2f` |
| `tilt.evidence_engagement` | `FDA moyen {before} → {during} ; engagement −{pct}%` | `Average KDA {before} → {during}; engagement −{pct}%` | `%.2f`, `pct` `%.0f` |
| `fatigue.trigger` | `Sessions de {n}+ matchs` | `Sessions of {n}+ matches` | `n` entier |
| `fatigue.evidence` | `FDA en baisse sur {pct}% des sessions analysées` | `KDA dropping in {pct}% of analysed sessions` | `%.0f` |
| `engagement_drop.trigger` | `{n} matchs récents sous les seuils habituels` | `{n} recent matches below usual levels` | `n` entier |
| `engagement_drop.evidence` | `Engagement perso : {self}% vs habitude ; {lobby}% vs le lobby` | `Personal engagement: {self}% vs usual; {lobby}% vs the lobby` | `%.0f` |
| `accuracy_plateau.trigger` | `{n} matchs sans progression` | `{n} matches without progress` | `plateauWindow` |
| `accuracy_plateau.evidence` | `Précision stable à {acc}% (σ={sigma})` | `Accuracy steady at {acc}% (σ={sigma})` | `%.1f`, `%.3f` |
| `perf_ceiling.trigger` | `Progression LOWESS plate sur {n}+ matchs` | `Flat LOWESS trend over {n}+ matches` | `PerfCeilingWindow` |
| `perf_ceiling.evidence` | `Plafond de performance à {max} pts (top {topn} moyen : {mean_top})` | `Performance ceiling at {max} pts (top {topn} average: {mean_top})` | `%.0f`, entier, `%.0f` |

Le remplacement de `behavioral.go:69+71` par `tilt.evidence` ou `tilt.evidence_engagement` supprime la concaténation : la clé est choisie selon que l'engagement est présent.
