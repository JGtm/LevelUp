# Suspects film_match_start — JGtm

> Matchs dont la corrélation filmshell ↔ highlight_events est suspecte
> (**gap_min > +60s** : l'estimation est trop précoce, capturée pendant le countdown).

**Nb total** : 48 matchs suspects (47 PvP + 1 Firefight ignoré)

---

## Commande de retraitement (autre PC avec chunks locaux)

Copier le répertoire `data/cache/film_chunks/` de l'autre PC vers ce PC (ou lancer
directement sur l'autre PC), puis exécuter :

```bash
# --max-chunks 10 = couvre jusqu'a 200s, détecte les countdowns longs
for mid in 1ea71def-8c63-46e4-8bd9-95bd9cabc080 9fe88ec4-6f34-44e9-998c-8663ce905f2c 6802effb-e0b4-4338-a08c-58aa6669ba43 0674b220-57e3-4444-b700-bc9e66561d2f 69f7fd6a-cfb3-485e-858f-dd3ea08c1c9e edf7983b-9310-4e5f-8106-68de97a9ba41 ad824bcc-f6cd-44fa-98cd-2d1131098aeb 04851350-9cf3-48f3-a034-d6883b1e77a8 c32b79dc-2678-430c-9921-3c69e333b333 95b2e456-b81b-402b-b6e4-d447953a1242 630c78f2-e182-45ee-b6e2-fc339078b624 c5c9db26-cd33-4e02-837b-3b6ba8e74059 bd133927-a509-47ff-9f2d-44b2c5b58f31 658c9e26-2fef-4874-b4ff-53a126e46420 5b47657d-5ad1-486e-892c-70518d04f1b9 14ba5518-a5ce-4bc9-bbcc-0fee49a74ee0 ca9a2500-11b2-4b7c-ae8a-38c824fd7ce4 3b289bd8-f4bd-410c-aab4-e03f1344a430 28109472-2d47-4242-bdd7-0a5256e65e26 8a48a47f-3f8a-4e04-b085-590708364084 e271dbfc-2fa2-4e9f-95f8-b1c431a918be d40afcfb-2b18-4de7-875e-7cc2ff631433 f060a151-1813-4081-99ac-3060f5be2568 aa027af8-8540-4f2c-adbd-9b594546d17e 58d09c44-decc-4946-a630-e7916c5cd68c 5faa6b74-0026-4e60-aaca-34522d75050c 81cc9952-3339-4681-aec7-893202fde627 55df2a12-ccbf-4f52-bf8c-6772632096f0 69b16f5d-c9af-4ec1-a9ef-4ec788c49441 21ece4d8-5715-4e2c-8c9b-937440ee564f f82203b7-5711-4e82-ae75-35365cfe492f 9123a92b-f2be-4a0b-9dd6-ecdbbfc86bc0 3824e8eb-0e0e-4584-bdd5-f0aa983bcf28 f43acfd2-186b-46fe-bc76-3fa875980dcb ace5c586-d3b7-4460-ab8f-024144ff6097 d99e5dbd-500d-407e-bc3b-86be0053624e f37771d1-73cc-49a4-a9d2-2749562976b4 04f7d9d5-7a59-4922-887e-8d7be56ac55d d9329229-693f-429c-a84c-3fd2784bcf4a 47149c49-ae40-43b9-91a1-dbd8cbf9400c 419457ca-b8cb-4dae-90ff-aee0eaaf75c9 cd802d23-e6e8-40b8-944e-cf0d8e45e52a 5fe3f9a2-0235-4988-8427-07231c8c9330 4c9e4014-0667-46da-a9ce-dd6654f96328 2acc306f-da41-41e3-8dfd-9e12ab1c9926 a974fdeb-56ba-47fc-9a7b-7ff009958fd1 8171b141-4797-4592-82d9-929592b9ec9b; do
  python scripts/_exp_spawn_download.py \
    --match-id "$mid" \
    --cached-only \
    --write-db \
    --max-chunks 10
done
```

> `--match-id` ignore `--skip-done` et **écrase** la valeur déjà en DB.

---

## Matchs à vérifier en mode cinéma

| # | Date | Heure | Map | Mode | Chunks locaux | dbstart | gap_min | Lien |
|---|------|-------|-----|------|:---:|---:|---:|:---:|
| 1 | 2025-11-13 | 19:27 | Deadlock | Big Team Battle | 1 | 4.1s | +63s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/1ea71def-8c63-46e4-8bd9-95bd9cabc080) |
| 2 | 2025-11-13 | 19:44 | Insolence | Big Team Battle | 6 | 53.6s | +85s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/9fe88ec4-6f34-44e9-998c-8663ce905f2c) |
| 3 | 2025-11-29 | 20:27 | Streets | Quick Play | 1 | 2.9s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/6802effb-e0b4-4338-a08c-58aa6669ba43) |
| 4 | 2025-12-01 | 21:14 | Banished Narrows | Quick Play | 1 | 4.1s | +62s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/0674b220-57e3-4444-b700-bc9e66561d2f) |
| 5 | 2025-12-13 | 21:30 | High Ground | Quick Play | 1 | 3.2s | +65s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/69f7fd6a-cfb3-485e-858f-dd3ea08c1c9e) |
| 6 | 2025-12-15 | 21:19 | Forbidden | Quick Play | 6 | 3.6s | +64s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/edf7983b-9310-4e5f-8106-68de97a9ba41) |
| 7 | 2025-12-18 | 12:07 | Critical Dewpoint | Quick Play | 6 | 3.9s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/ad824bcc-f6cd-44fa-98cd-2d1131098aeb) |
| 8 | 2025-12-18 | 15:56 | High Ground | Quick Play | 1 | 4.1s | +73s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/04851350-9cf3-48f3-a034-d6883b1e77a8) |
| 9 | 2025-12-20 | 14:22 | The Pit | Quick Play | 1 | 4.9s | +77s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/c32b79dc-2678-430c-9921-3c69e333b333) |
| 10 | 2025-12-20 | 19:24 | Streets | Quick Play | 1 | 4.2s | +63s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/95b2e456-b81b-402b-b6e4-d447953a1242) |
| 11 | 2025-12-25 | 16:51 | Argyle | Quick Play | 1 | 3.1s | +62s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/630c78f2-e182-45ee-b6e2-fc339078b624) |
| 12 | 2025-12-25 | 17:19 | Nemesis | Quick Play | 1 | 4.1s | +65s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/c5c9db26-cd33-4e02-837b-3b6ba8e74059) |
| 13 | 2025-12-27 | 18:01 | Chasm | Quick Play | 6 | 2.6s | +65s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/bd133927-a509-47ff-9f2d-44b2c5b58f31) |
| 14 | 2025-12-27 | 18:16 | Detachment | Quick Play | 6 | 3.1s | +64s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/658c9e26-2fef-4874-b4ff-53a126e46420) |
| 15 | 2025-12-29 | 22:24 | Live Fire | Quick Play | 1 | 2.4s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/5b47657d-5ad1-486e-892c-70518d04f1b9) |
| 16 | 2025-12-30 | 21:36 | Detachment | Quick Play | 6 | 3.0s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/14ba5518-a5ce-4bc9-bbcc-0fee49a74ee0) |
| 17 | 2026-01-02 | 20:18 | Domicile | Quick Play | 3 | 48.6s | +70s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/ca9a2500-11b2-4b7c-ae8a-38c824fd7ce4) |
| 18 | 2026-01-17 | 15:06 | Takamanohara | Quick Play | 6 | 3.1s | +62s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/3b289bd8-f4bd-410c-aab4-e03f1344a430) |
| 19 | 2026-01-18 | 16:03 | Cliffside | Quick Play | 6 | 3.8s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/28109472-2d47-4242-bdd7-0a5256e65e26) |
| 20 | 2026-01-19 | 17:20 | Cliffhanger | Quick Play | 1 | 7.3s | +64s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/8a48a47f-3f8a-4e04-b085-590708364084) |
| 21 | 2026-01-21 | 21:49 | Isolation | Quick Play | 6 | 2.5s | +66s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/e271dbfc-2fa2-4e9f-95f8-b1c431a918be) |
| 22 | 2026-01-21 | 21:58 | Origin | Quick Play | 6 | 2.3s | +73s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/d40afcfb-2b18-4de7-875e-7cc2ff631433) |
| 23 | 2026-01-27 | 19:13 | Detachment | Quick Play | 6 | 3.3s | +63s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/f060a151-1813-4081-99ac-3060f5be2568) |
| 24 | 2026-01-27 | 23:23 | Sylvanus | Quick Play | 6 | 4.7s | +75s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/aa027af8-8540-4f2c-adbd-9b594546d17e) |
| 25 | 2026-01-28 | 18:54 | Refuge | Big Team Battle | 6 | 4.2s | +66s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/58d09c44-decc-4946-a630-e7916c5cd68c) |
| 26 | 2026-01-28 | 19:20 | Fragmentation Heavies | Big Team Battle | 1 | 6.0s | +74s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/5faa6b74-0026-4e60-aaca-34522d75050c) |
| 27 | 2026-01-29 | 17:52 | Cliffside | Quick Play | 6 | 3.4s | +66s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/81cc9952-3339-4681-aec7-893202fde627) |
| 28 | 2026-02-01 | 18:21 | Solution | Quick Play | 1 | 3.8s | +71s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/55df2a12-ccbf-4f52-bf8c-6772632096f0) |
| 29 | 2026-02-03 | 17:06 | Origin | Quick Play | 1 | 2.9s | +76s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/69b16f5d-c9af-4ec1-a9ef-4ec788c49441) |
| 30 | 2026-02-03 | 17:26 | Live Fire | Quick Play | 1 | 3.0s | +64s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/21ece4d8-5715-4e2c-8c9b-937440ee564f) |
| 31 | 2026-02-09 | 18:11 | Launch Site | Quick Play | 1 | 3.6s | +80s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/f82203b7-5711-4e82-ae75-35365cfe492f) |
| 32 | 2026-02-09 | 21:46 | Behemoth | Quick Play | 6 | 2.1s | +60s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/9123a92b-f2be-4a0b-9dd6-ecdbbfc86bc0) |
| 33 | 2026-02-10 | 20:33 | Cliffhanger | Quick Play | 1 | 4.5s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/3824e8eb-0e0e-4584-bdd5-f0aa983bcf28) |
| 34 | 2026-02-10 | 22:19 | Detachment | Quick Play | 6 | 2.5s | +67s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/f43acfd2-186b-46fe-bc76-3fa875980dcb) |
| 35 | 2026-02-11 | 22:08 | Snowbound | Quick Play | 1 | 3.3s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/ace5c586-d3b7-4460-ab8f-024144ff6097) |
| 36 | 2026-02-15 | 18:26 | Launch Site | Quick Play | 1 | 2.8s | +65s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/d99e5dbd-500d-407e-bc3b-86be0053624e) |
| 37 | 2026-02-18 | 21:54 | Detachment | Quick Play | 6 | 3.1s | +78s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/f37771d1-73cc-49a4-a9d2-2749562976b4) |
| 38 | 2026-02-18 | 22:51 | High Ground | Quick Play | 1 | 3.9s | +68s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/04f7d9d5-7a59-4922-887e-8d7be56ac55d) |
| 39 | 2026-03-01 | 22:47 | Streets | Quick Play | 1 | 1.5s | +65s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/d9329229-693f-429c-a84c-3fd2784bcf4a) |
| 40 | 2026-03-03 | 22:31 | High Ground | Quick Play | 1 | 4.6s | +68s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/47149c49-ae40-43b9-91a1-dbd8cbf9400c) |
| 41 | 2026-03-03 | 23:23 | Cliffhanger | Quick Play | 1 | 2.1s | +62s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/419457ca-b8cb-4dae-90ff-aee0eaaf75c9) |
| 42 | 2026-03-10 | 20:00 | Behemoth | Quick Play | 6 | 2.3s | +60s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/cd802d23-e6e8-40b8-944e-cf0d8e45e52a) |
| 43 | 2026-03-14 | 11:25 | Recharge | Quick Play | 6 | 5.6s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/5fe3f9a2-0235-4988-8427-07231c8c9330) |
| 44 | 2026-03-14 | 11:56 | Prism | Quick Play | 1 | 2.7s | +66s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/4c9e4014-0667-46da-a9ce-dd6654f96328) |
| 45 | 2026-03-18 | 21:58 | Illusion | Quick Play | 6 | 2.7s | +77s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/2acc306f-da41-41e3-8dfd-9e12ab1c9926) |
| 46 | 2026-03-18 | 22:55 | Ecotone | Quick Play | 6 | 2.0s | +75s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/a974fdeb-56ba-47fc-9a7b-7ff009958fd1) |
| 47 | 2026-03-27 | 23:13 | High Ground | Quick Play | 1 | 3.6s | +61s | [Waypoint](https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/8171b141-4797-4592-82d9-929592b9ec9b) |

---

## Ignorés (Firefight — pas de film POV)

- `edc5daf6…` — 2025-11-09 18:55 — Cole Protocol — gap=+361s (normal : Firefight sans events kill/death)

---

## Interprétation du gap

- **gap_min ~+3s** : estimation correcte (premier frag 3s après le début du match)
- **gap_min ~+60-80s** : estimation trop précoce — le script a détecté un mouvement
  pendant le **countdown** (~3-5s dans l'enregistrement) alors que le match a
  vraiment commencé ~63-85s plus tard.
- **Correction attendue** : avec `--max-chunks 10`, la détection remontera les
  chunks suivants et trouvera la vraie rupture de mouvement post-countdown.