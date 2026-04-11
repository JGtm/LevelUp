# SPNKr PR — feat(economy+gamecms): challenges, reward tracks, item metadata

Fichiers prêts à être déposés dans un fork de SPNKr avant ouverture de la PR.

## Structure

```
spnkr_pr/
├── models/
│   ├── economy_additions.py      # ← diff à fusionner dans spnkr/models/economy.py
│   └── gamecms_hacs_additions.py # ← diff à fusionner dans spnkr/models/gamecms_hacs.py
├── services/
│   ├── economy_additions.py      # ← diff à fusionner dans spnkr/services/economy.py
│   └── gamecms_hacs_additions.py # ← diff à fusionner dans spnkr/services/gamecms_hacs.py
└── tests/
    ├── fixtures/                  # réponses JSON mockées (anonymisées)
    └── test_economy_ext.py        # tests unitaires pytest/aioresponses
```

## Endpoints couverts

| Endpoint | Service | Méthode |
|---|---|---|
| `GET /hi/players/xuid({xuid})/challenges` | `EconomyService` | `get_player_challenges` |
| `GET /hi/players/xuid({xuid})/rewardtracks` | `EconomyService` | `get_player_reward_tracks` |
| `GET /hi/players/xuid({xuid})/rewardtracks/{type}/{id}` | `EconomyService` | `get_player_reward_track` |
| `GET /hi/players/xuid({xuid})/currency/spartan-points` | `EconomyService` | `get_player_spartan_points` |
| `GET /hi/Waypoint/file/{inventory_item_path}` | `GameCmsHacsService` | `get_item_metadata` |

## Incertitudes connues (à valider avec captures réseau)

- Noms exacts des champs JSON de `/challenges` (PascalCase attendu d'après le
  pattern des autres endpoints economy)
- `current_progress` sur `/rewardtracks` : XP brut pour les opérations, rang
  entier pour `career_rank`
- Les champs optionnels de `InventoryItem` varient selon le type d'item
- Ces endpoints exigent que le xuid soit celui du joueur authentifié (pas de vue publique)
