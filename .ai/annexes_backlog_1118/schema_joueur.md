# Schéma d'une base joueur selon l'ordre de création (mesure du 2026-11-18)

Mesure faite par `TestPlayerSchemaForme_NaissanceParLaSynchroOuParLesMigrations`
(`apps/go-api/internal/sync/schema_forme_test.go`, étiquette `integration`) : une base temporaire
née de `EnsurePlayerSchema` (synchronisation) puis de la chaîne de migrations, contre une base née
de la chaîne seule. Comparaison dans les deux sens, après fermeture et réouverture du fichier, sur
le type, la valeur par défaut et le NOT NULL de chaque colonne, plus les contraintes de table, les
vues, les index et les séquences (491 clés de chaque côté).

## État avant l'alignement : 16 lignes d'écart

Aucune table, vue, index ni séquence n'est présent d'un seul côté ; les 16 écarts sont des colonnes.
Le « 37 lignes » mesuré plus tôt n'est pas reproduit par ce relevé (non archivé : méthode inconnue).
Pour mémoire, la base née de la seule synchronisation, avant toute migration, diffère de la chaîne
sur 328 clés (304 objets absents d'un côté, 24 colonnes présentes des deux côtés) ; la chaîne
reconstruit ensuite `career_progression` et corrige ces colonnes-là.

| Colonne | Synchronisation puis migrations | Migrations seules | Base existante relevée | Référence |
|---|---|---|---|---|
| `match_skill_rank.match_id` | VARCHAR NOT NULL | VARCHAR nullable | VARCHAR nullable | migrations |
| `match_skill_rank.rating_type` | VARCHAR NOT NULL | VARCHAR nullable | VARCHAR nullable | migrations |
| `match_skill_rank.sub_tier` | SMALLINT défaut 0 | SMALLINT sans défaut | SMALLINT sans défaut | migrations |
| `match_skill_rank.measurement_matches_remaining` | INTEGER défaut 0 | INTEGER sans défaut | INTEGER sans défaut | migrations |
| `match_skill_rank.written_at` | TIMESTAMP défaut UTC NOT NULL | TIMESTAMP défaut UTC nullable | TIMESTAMPTZ défaut UTC nullable | migrations (le type TIMESTAMPTZ est un héritage, voir plus bas) |
| `match_skill_rank.created_at` | TIMESTAMP défaut UTC | TIMESTAMP sans défaut | sans défaut | migrations |
| `match_skill_rank.updated_at` | TIMESTAMP défaut UTC | TIMESTAMP sans défaut | sans défaut | migrations |
| `player_match_enrichment.match_id` | VARCHAR NOT NULL | VARCHAR nullable | VARCHAR nullable | migrations |
| `player_match_enrichment.is_with_friends` | BOOLEAN défaut FALSE | BOOLEAN sans défaut | sans défaut | migrations |
| `player_match_enrichment.is_excluded` | BOOLEAN défaut FALSE | BOOLEAN sans défaut | sans défaut | migrations |
| `player_match_enrichment.written_at` | TIMESTAMP défaut UTC NOT NULL | TIMESTAMP défaut UTC nullable | TIMESTAMPTZ défaut `now()` nullable | migrations (héritage, voir plus bas) |
| `player_match_enrichment.created_at` | TIMESTAMP défaut UTC | TIMESTAMP sans défaut | sans défaut | migrations |
| `player_match_enrichment.updated_at` | TIMESTAMP défaut UTC | TIMESTAMP sans défaut | sans défaut | migrations |
| `player_match_enrichment.performance_score` | FLOAT | DOUBLE | FLOAT | synchronisation |
| `player_match_enrichment_latest.performance_score` (vue) | FLOAT | DOUBLE | FLOAT | synchronisation |
| `sync_meta.updated_at` | TIMESTAMP défaut UTC | TIMESTAMP sans défaut | TIMESTAMP défaut UTC | synchronisation |

Cause des écarts « migrations » : la chaîne crée les tables au format ancien (clé `match_id`, défauts
posés), puis `player_append_only_*_v1` les reconstruit par `CREATE TABLE AS`, qui perd les défauts et
les NOT NULL. La synchronisation, elle, crée directement la forme append-only avec ses défauts. Pour
`measurement_matches_remaining`, la synchronisation ne portait pas la colonne : l'ajout additif de la
baseline la posait avec son défaut 0.

## Forme des bases existantes

Une seule base relevée : la plus grosse des bases joueur du poste, copiée vers un dossier à part et
ouverte en lecture seule. Les autres bases du poste et la production n'ont pas été relevées ; la
forme de référence repose donc sur cet échantillon unique.

Cette base porte la forme « migrations » partout où les deux définitions s'écartent, sauf trois
colonnes où elle porte la forme « synchronisation » (`performance_score` FLOAT, vue comprise ;
`sync_meta.updated_at` avec son défaut).

Écarts de cette base avec les DEUX formes neuves, hors de la comparaison synchronisation / migrations :

- `match_skill_rank.written_at` et `player_match_enrichment.written_at` : TIMESTAMPTZ (le défaut de
  `player_match_enrichment.written_at` est `now()`), contre TIMESTAMP dans les deux définitions neuves.
- `player_match_enrichment.known_teammates_count` et `friends_xuids` : colonnes mortes que la base porte
  et que ni l'une ni l'autre forme neuve ne crée.
- Tables et colonnes héritées absentes de la chaîne actuelle : `mv_session_stats`, `mv_player_matches`,
  `mv_global_stats`, `mv_mode_category_stats`, `spartan_identity`, la séquence `highlight_events_id_seq`,
  la clé primaire `skill_history(playlist_id, recorded_at)`, et les colonnes `sessions.total_kills`,
  `total_deaths`, `total_assists`, `avg_kda`, `avg_accuracy`, `performance_score`.
- Colonnes que la chaîne crée et que cette base n'a pas : `match_citations.created_at` (et la vue
  `match_citations_latest`), `sessions.created_at`, `sessions.label`.

## État après l'alignement

Alignés sur la forme de référence, côté définition de la synchronisation (`playerCoreSchemaSQL`),
pour les bases neuves seulement : 11 colonnes de `match_skill_rank` et `player_match_enrichment`
dont la référence est « migrations » (NOT NULL retirés, défauts retirés, TIMESTAMP nullable pour
`written_at`) ; `measurement_matches_remaining` entre dans la définition de la synchronisation sans
défaut.

Écarts résiduels (cinq lignes, nommées et datées dans `formeEcartsResiduels`) :

- `performance_score` (table et vue) et `sync_meta.updated_at` : la référence est la forme
  « synchronisation » ; l'aligner exige de modifier `create_baseline_player_v1`, étape déjà jouée en
  production (et son instantané de référence `testdata/squash/player_block_golden.snapshot`), ce que la
  piste interdit.
- `player_match_enrichment.is_with_friends` et `is_excluded` : la référence (bases existantes et chaîne)
  est « sans défaut », mais `TestPlayerMatchEnrichment_DefaultFalse` impose DEFAULT FALSE à la définition
  de la synchronisation (garde de la régression « Solo qui revient » : un NULL). Retirer le défaut
  contredit cette garde ; l'ajouter à la chaîne exige de modifier une étape déjà jouée. La définition de
  la synchronisation les garde donc.

Options, au choix du user :

1. Modifier la définition de `create_baseline_player_v1` (`DOUBLE` vers `FLOAT`, défaut UTC sur
   `sync_meta.updated_at`) et régénérer l'instantané de référence : n'affecte aucune base existante
   (la baseline n'est pas rejouée, DM-5), mais change une étape déjà livrée. Ne règle pas les défauts
   FALSE, que la reconstruction append-only de `player_match_enrichment` perd : il faudrait en plus la
   modifier.
2. Écrire une étape neuve qui pose le défaut de `sync_meta.updated_at` (`ALTER ... SET DEFAULT`, sans
   reconstruction) : touche les bases existantes (inscription au registre, effet nul sur celle
   relevée). Rien d'équivalent sans reconstruction de table pour `performance_score` (une vue lit la
   colonne).
3. Laisser les cinq résiduels, nommés au test : l'effet d'un FLOAT contre un DOUBLE sur les lecteurs de
   `performance_score` n'a pas été mesuré ; les lecteurs de `is_with_friends` passent par la vue
   `player_match_enrichment_latest`, qui coalesce à FALSE.
