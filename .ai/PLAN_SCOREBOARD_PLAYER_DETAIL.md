# Plan — Panel détaillé joueur sur le Scoreboard

> Statut : **À implémenter**
> Branche cible : `feature/scoreboard-player-detail`
> Fichiers impactés : `match_view_scoreboard.py`, `match_view_scoreboard_detail.py` (nouveau), `styles.css`

---

## Contexte

Le scoreboard de la page Last Match est rendu en **HTML pur** via `st.markdown(unsafe_allow_html=True)`.
Il n'est pas possible de rendre une `<tr>` HTML cliquable avec un callback Python sans JavaScript
personnalisé. La sélection native de `st.dataframe` est donc écartée (régression visuelle totale).

**Solution retenue : Sélecteur pill stylisé + panel expansible.**

---

## Maquette attendue

```
┌──────────────────────────────────────────────────────────────┐
│  [Équipe Bleue]  › tableau HTML existant inchangé            │
└──────────────────────────────────────────────────────────────┘
  [ ⭐ Acme_GT ]  [ JohnDoe ]  [ Player3 ]  [ Player4 ]    ← pills
┌──────────────────────────────────────────────────────────────┐
│  ▼  JohnDoe                                             [✕]  │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐                        │
│  │  K   │ │  D   │ │  A   │ │ Score│  ← KPIs (scoreboard)    │
│  │  14  │ │  5   │ │  3   │ │ 4200 │                        │
│  └──────┘ └──────┘ └──────┘ └──────┘                        │
│  🏅 Médailles : Killing Spree ×3 · Headmaster ×2            │
│  🔫 Armes : Battle Rifle (8k) · Sniper (4k) · BR (2k)        │
│  📊 Performance Score : 78/100   [si DB disponible]          │
│  📜 Citations : "Relentless" · ...   [si DB disponible]      │
└──────────────────────────────────────────────────────────────┘
```

---

## Principes UX

- Un seul panel ouvert à la fois (par match, pas par équipe).
- Cliquer sur le pill du joueur déjà sélectionné **ferme** le panel (toggle).
- La sélection est stockée dans `session_state` avec une clé dérivée du `match_id`
  pour éviter les collisions lors de la navigation précédent/suivant.
- Le panel est rendu dans un `@fragment_if_available` pour éviter le rerun global.

---

## Données affichées

### Disponibles sans DB joueur (shared_matches, toujours présentes)

| Section | Source DuckDB |
|---------|---------------|
| KPIs (K/D/A, accuracy, damage, avg life, etc.) | `shared.match_participants` (déjà dans le scoreboard) |
| Médailles du match | `shared.medals_earned WHERE xuid = ? AND match_id = ?` |
| Kills par arme (top 5) | `shared.v_weapon_kills WHERE xuid = ? AND match_id = ?` |
| Duels clés (nemesis / victime fav) | `shared.v_killer_victim_full` |

### Disponibles uniquement si la DB joueur existe

| Section | Source DuckDB |
|---------|---------------|
| Performance Score | `player.player_match_enrichment WHERE match_id = ?` |
| Session (avec ou sans amis) | `player.player_match_enrichment.is_with_friends` |
| Citations du match | `player.match_citations WHERE match_id = ?` |
| Rang Halo au moment du match | `player.career_progression` (nearest timestamp) |

---

## Architecture des fichiers

### Nouveau fichier : `src/ui/pages/match_view_scoreboard_detail.py`

Responsabilités :
- `load_player_detail_data(db_path, match_id, xuid, has_player_db) -> PlayerDetailData`
  — Charge toutes les données du panel (shared + optionnel player DB)
- `render_player_detail_panel(data: PlayerDetailData, player: dict, is_me: bool) -> None`
  — Rendu HTML/Streamlit du panel (KPIs, médailles, armes, extras si dispo)

### Modifications : `src/ui/pages/match_view_scoreboard.py`

- `_render_team_pills(team_players, team_css_class, match_id, me_xuid, mvp_xuid, lvp_xuid) -> None`
  — Rend la rangée de pills sous chaque tableau d'équipe
- `render_match_scoreboard()` — appel de `_render_team_pills` après chaque `_render_team_table`
  + récupération de la sélection depuis `session_state` + appel `render_player_detail_panel`

### Modifications : `static/styles.css`

Nouvelles classes à ajouter (section scoreboard) :

| Classe | Rôle |
|--------|------|
| `.os-sb-pills` | Conteneur flex des pills |
| `.os-sb-pill` | Pill joueur de base (dark, coins coupés style Waypoint) |
| `.os-sb-pill--mine` | Pill du joueur principal (cyan) |
| `.os-sb-pill--mvp` | Pill MVP (vert Okabe-Ito) |
| `.os-sb-pill--lvp` | Pill LVP (rouge Okabe-Ito) |
| `.os-sb-pill--active` | Pill actuellement sélectionné (surbrillance) |
| `.os-sb-detail` | Container du panel détaillé (fond légèrement surélevé) |
| `.os-sb-detail-header` | En-tête du panel (nom + bouton fermer) |
| `.os-sb-detail-kpis` | Grille des KPI cards dans le panel |
| `.os-sb-detail-section` | Section médailles / armes / extras |
| `.os-sb-detail-badge` | Badge inline médaille ou arme |

---

## Clés session_state

```python
# Joueur sélectionné dans le panel détaillé pour un match donné
f"sb_selected_xuid_{match_id}"   # str | None

# Équipe du joueur sélectionné (pour le placement du panel)
f"sb_selected_team_{match_id}"   # Any | None
```

---

## Étapes d'implémentation

- [ ] 1. Ajout des classes CSS dans `static/styles.css`
- [ ] 2. Créer `match_view_scoreboard_detail.py` avec `load_player_detail_data` + `render_player_detail_panel`
- [ ] 3. Ajouter `_render_team_pills` dans `match_view_scoreboard.py`
- [ ] 4. Modifier `render_match_scoreboard` pour orchestrer pills + panel
- [ ] 5. Tests manuels : joueur avec DB / joueur sans DB / bot / MVP/LVP
- [ ] 6. Vérifier taille fichiers (≤ 500 L) et fonctions (≤ 80 L)
- [ ] 7. Commit `feat(scoreboard): panel détaillé joueur via pills sélecteur`

---

## Gestion des formats de match

### FFA (chacun pour soi)

**Détection — structurelle, sans requête supplémentaire :**
```python
is_ffa = (
    all(len(players_in_team) == 1 for players_in_team in teams.values())
    and len(teams) > 2
)
```
Couverture : FFA 8/12/16 joueurs → détecté. 1v1 (2 équipes de 1) et tous les modes multi-équipes normaux → non affecté.

**Layout FFA :**
- Un **tableau unique** sans en-tête d'équipe (pas de bandeau "Eagle" / "Cobra").
- Joueurs triés par `score` décroissant (classement naturel FFA).
- **Une seule rangée de pills** sous le tableau (tous les joueurs).
- Si > 8 joueurs : pills en `overflow-x: auto` (scroll horizontal) ou grille 2 colonnes — seuil `MAX_PILLS_INLINE = 8` à définir dans `match_view_scoreboard.py`.

**Implémentation :**
- Ajouter `is_ffa` dans `render_match_scoreboard` avant la boucle d'équipes.
- Si FFA : appeler `_render_team_table` avec `tid=None` sur tous les joueurs triés + pills global unique.
- Si non-FFA : comportement actuel inchangé (N tableaux + N rangées de pills).
- Note de classement (`mv_scoreboard_rank_note`) : afficher aussi en FFA.

### Autres formats (BTB 16 joueurs, 4 équipes de 2…)

Aucune adaptation nécessaire : la boucle `teams` gère N équipes de taille quelconque.
Pills > 8 par équipe : même règle `MAX_PILLS_INLINE` que FFA.

---

## Points de vigilance

- Les pills utilisent `st.button` → chaque clic déclenche un rerun partiel (fragment).
- `key` des boutons : `f"sb_pill_{match_id}_{xuid}"` pour éviter tout conflit multi-match.
- Les bots (`bid(...)`) doivent avoir un pill désactivé (pas de données en DB).
- `has_player_db` est déterminé par l'existence du fichier `data/players/{gamertag}/stats.duckdb`
  **et** la présence de `player_match_enrichment` — utiliser `has_table_duckdb` existant.
- Ne pas faire de requête DB si le panel est fermé (lazy loading).
- **FFA + 1v1 ambiguïté** : le seuil `> 2` équipes évite de traiter le 1v1 comme FFA.
  Si un jour Halo ajoute un mode 3 équipes de 1, il sera traité en FFA — acceptable.
