# Handoff — Emprise et cartes déplacées : clôture (2026-09-27)

Plan : `.ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md` (section L6). Maquettes de référence :
`.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html` (onglet) et
`.ai/V7.5/MAQUETTE_TRI_CARTES_DEPLACEES_2026-09-26.html` (cartes déplacées).

## 1. État

| Étape | État |
|---|---|
| L0 à L5 (code) | fait, sur `feat/v75` |
| L6.1 revue adversariale | close (ronde 1 : R1-R13 ; ronde 2 : 0 constat ; R14-R15) |
| L6.2 rattrapage local | fait (voir §3) |
| L6.3 fusion + CI | fait : `feat/v75` = `0cae929af`, CI verte au niveau job (runs 36335594113, 36335588369) |
| L6.4 gate visuel | **à faire par l'utilisateur** |
| L6.5 rattrapage prod | **à faire par l'utilisateur, à la main** (§3) |
| L6.6 référence équipement §4 | à faire |
| L7 véhicules | plan détaillé à écrire puis soumettre |

Commit local non poussé : `d7c40151e` sur `wt/emprise` (L6.3 statué au plan, journal) — et ce
handoff. Les pousser avec L6.6 (le pre-push complet prend environ 12 minutes).

Worktree : `C:\Users\Guillaume\Projects\LevelUp-wt-emprise`, branche `wt/emprise`, alignée sur
`feat/v75` plus les commits ci-dessus. Dossier principal : avancé à `0cae929af` ; les
modifications non commitées d'autres sessions y sont intactes (`CLAUDE.md`, les deux
`CONTRIBUTING`, entrées de journal réappliquées en fin de fichier). Sauvegarde de l'état
d'avant : scratchpad de la session, dossier `principal_backup`.

## 2. Gate visuel (L6.4)

`make dev` depuis le dossier principal, Escouade, soirée du 22/09, maquettes à côté :
- onglet **Emprise** (l'ancienne adresse Usages y redirige) : piste camp contre camp, contrôle
  des ressources, au fil de la session, répartition des prises (fiches), match par match,
  production, rendement, habitude ;
- **Contributions** : Répartition des frags, Outils de destruction, les quatre cartes d'objectif ;
- **Dynamique** : Écart cumulé au FDA attendu.

L'utilisateur nomme les cartes validées et celles à reprendre. Toute reprise = un commit sur
`wt/emprise`, gate web (`lefthook run pre-push` compris), puis fusion dans `feat/v75`.

## 3. Rattrapage des bases (L6.5)

### Ce qui a été écrit en local

Une seule base, une seule table :
`data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`, table append-only
`match_pad_pickups_by_tier` (lue par la vue `_latest`). `levelup backfill-pad-tiers` : 12 matchs
écrits, 99 déjà présents, 0 échec. `backfill-usage-summary` n'a rien écrit (0 écrit, 17 à jour) :
rien à reporter.

Pourquoi il en faut un en prod : le correctif des niveaux de socle (L4) ne répare que les passes
futures. Les matchs dérivés pendant la panne portent leur marque et ne seront jamais repris
d'eux-mêmes ; sans rattrapage, l'Emprise les affiche « non classé ».

### ATTENTION avant d'écraser la base de prod par la copie locale

- **La base locale s'arrête au 22/09** : dernier match 2026-09-22 22:29 (+02), 9 170 matchs dans
  `match_registry`. La prod synchronise en continu : tout match postérieur présent en prod
  (et toute autre écriture faite en prod depuis) serait **perdu** par l'écrasement.
- La base locale a été compactée le 27/09 par une autre session (1 264 → 350 Mio, vues `_latest`
  identiques) et contient aussi le rattrapage killsource du 27/09 (autre chantier) : la copie les
  emporterait.
- Le code du chantier n'est en prod qu'après la fusion `feat/v75` → `main` (sortie v7.5) : un
  rattrapage fait avant ce déploiement serait refait sur une base que la prod réécrit.

Vérification à faire avant la copie, sur la prod (serveur arrêté ou en lecture) :

```sql
SELECT max(COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')), count(*) FROM match_registry;
```

Si la prod a plus récent que le 22/09 22:29 ou plus de 9 170 matchs, **ne pas écraser**.

### Voie recommandée (sans copie)

Après le déploiement de v7.5, sur le VPS : arrêter le serveur (un seul écrivain par base), puis
`levelup backfill-pad-tiers` (reprenable, aucun décodage de film, un artefact à la fois ; lit les
artefacts de rejeu déjà rangés de la prod), puis relancer le serveur. `--dry-run` d'abord pour
voir la liste. `backfill-usage-summary` n'est pas nécessaire.

### Si copie malgré tout

Serveur de prod arrêté ; copier `shared_matches_v2.duckdb` **et** supprimer / ne pas laisser
traîner un `shared_matches_v2.duckdb.wal` de l'ancienne base à côté ; relancer ; vérifier que
`match_pad_pickups_by_tier_latest` compte au moins 111 matchs.

## 4. Autres coordinations

- levelup-ac lira `shared_matches_v2.duckdb` en local (gate `replay-corpus-gate`, lecture seule,
  serveur arrêté) dans l'heure qui suit 20:00 le 27/09 ; il a l'accord de l'utilisateur pour
  arrêter/relancer le serveur local.
- Découvertes non traitées : section « Découvertes » du plan (L6.1 : ligne « frags obtenus avec »
  d'un match au camp inconnu ; `backfill-usage-summary` : 111 artefacts au schéma périmé, recuisson
  hors chantier).
- Leçon : les gates des lots web doivent jouer `lefthook run pre-push` ; deux garde-rails
  (`lint-no-hardcoded-fields`, `lint-contract-ratchet`) n'ont échoué qu'au push.
