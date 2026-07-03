# Package `progression`

Package parent des sous-systèmes de progression joueur post-match. **Aucun `.go`
au niveau racine** (d'où un README plutôt qu'un `doc.go`) — uniquement des
sous-packages :

- `milestones/` — paliers atteints (`milestone_earned`).
- `streaks/` — séries + boucliers (`streak`).
- `records/` — records personnels (`player_records`).
- `profile/` — profil de combat (lissage LOWESS).
- `coach/` + `coach_advisor/` — génération narrative / alertes (ADR 0020 / 0028).

Les tables de progression V2 (Ascension) sont écrites par le pipeline post-sync ;
la lecture des tables append-only passe par les vues `_latest` (ADR 0026).
