# PORTING_REFERENCE.md — Référence de portage et de parité

> Ce document regroupe ce qui alourdit le plan sans relever du pilotage :
> surfaces produit, algorithmes critiques, familles de requêtes, dépendances et stratégie de tests.

## Rôle du document

Il sert de mémo technique de haut niveau pour éviter deux écueils :

1. replanifier le chantier à partir d'intuitions vagues ;
2. enfouir les vrais invariants de portage au milieu d'un plan stratégique trop long.

Le détail exhaustif local est dans [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md).
La cible runtime finale du produit est détaillée dans [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md).
Le modèle canonique et la capability map initiale sont figés dans [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) et [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md).
Le contrat bootstrap et le blueprint types Go sont détaillés dans [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) et [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md).
La discipline de mapping et d'adaptation produit est désormais détaillée dans [HALO_INFINITE_CANONICAL_MAPPING.md](HALO_INFINITE_CANONICAL_MAPPING.md) et [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md).
La taxonomie d'erreurs et le freeze OpenAPI MVP des parcours P0/P1 sont maintenant détaillés dans [HALO_PROVIDER_ERROR_TAXONOMY.md](HALO_PROVIDER_ERROR_TAXONOMY.md) et [OPENAPI_MVP_P0_P1.md](OPENAPI_MVP_P0_P1.md).

## Surfaces produit prioritaires

| Surface | Priorité | Pourquoi | Oracle minimum |
|---------|:--------:|----------|----------------|
| Bootstrap / Players / Filters | P0 | socle contractuel du frontend | payload JSON et résolution cascade identiques |
| Career | P1 | parcours visible, stable, peu risqué en écriture | golden values carrière + historique rangs |
| Match History | P1 | parcours central et fortement consommé | pagination, tri, filtres, compteurs |
| Explorer + Match View | P1 | combine recherche, détails et killer/victim | payloads page + timeline + scoreboard |
| Stats / Series / Sessions | P1 | forte densité analytique | golden values chiffrées sur séries et sessions |
| Settings / Setup | P2 | faible valeur sans socle auth/jobs | contrats PATCH/GET et smoke test |
| Citations / Médias / Home | P2 | valeur produit mais moins critique au démarrage | golden values ciblées |
| Sync / Backfill / CLI | P4 | zone la plus risquée, à porter en dernier | cycle complet équivalent Python |

## Couverture métier complémentaire obligatoire

Ces sujets ne doivent pas être perdus parce qu'ils ne sont pas tous visibles dans les premiers écrans portés.

| Sujet | Pourquoi | Preuve minimale |
|-------|----------|-----------------|
| Bitmask de backfill | la détection des données manquantes dépend d'une identité exacte, pas d'une simple équivalence logique | identité numérique stricte des `BACKFILL_FLAGS` historiques et des `MatchBits` associés |
| i18n dynamique | les labels Halo et les filtres traduits font partie du contrat utile | traductions et fallback identiques via DuckDB et comportement cohérent selon la langue |
| PvE / Firefight | le programme ne couvre pas seulement le PvP | lecture et écriture de `shared_pve.duckdb` et requêtes PvE validées |
| Notifications Discord | le runbook réel inclut les notifications post-sync et post-backfill | embeds, anti-spam, thumbnails et langue correctement reproduits |
| Media indexing | le produit dépend aussi du pipeline média, pas seulement des pages de stats | hash, ffprobe, association match, indexation et miniatures vérifiés |
| Multi-joueurs | l'architecture réelle est organisée par joueur et par DB dédiée | pools par gamertag, isolation des lectures et write leases indépendants |
| Archive Parquet | le cold storage reste une surface technique utile | lecture et écriture archive couvertes ou stratégie de report explicitement tracée |

## Architecture API multi-titre à préserver

Le remplacement de Halo Infinite par un futur titre ne doit pas forcer une réécriture de l'API produit.

1. Orientation produit : les routes internes restent organisées par parcours utilisateur (bootstrap, history, explorer, match view, settings, sync), jamais par endpoints Waypoint.
2. Provider de titre : l'intégration externe est en deux niveaux, avec un socle Halo générique pour transport/auth/rate limit/erreurs/registre d'endpoints, puis un provider par titre.
3. Mapping canonique : chaque provider mappe son payload natif vers les modèles canoniques LevelUp avant toute logique métier ou exposition HTTP.
4. Zones à isoler : auth Xbox/XSTS/Waypoint, refdata, assets discovery/economy, formules skill, films/chunks, événements, labels et URLs Waypoint spécifiques au jeu.
5. Dégradation maîtrisée : si un futur titre n'expose pas exactement la même surface, l'API produit doit pouvoir désactiver ou dégrader une capability sans casser tout le contrat restant.

## Préparation immédiate avant implémentation multi-titre

Ces livrables doivent être figés pendant le cadrage, avant tout vrai code Go sur la couche Halo.

Ils sont maintenant matérialisés dans :

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md)
2. [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md)
3. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md)
4. [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md)
5. [HALO_INFINITE_CANONICAL_MAPPING.md](HALO_INFINITE_CANONICAL_MAPPING.md)
6. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md)
 7. [HALO_PROVIDER_ERROR_TAXONOMY.md](HALO_PROVIDER_ERROR_TAXONOMY.md)
 8. [OPENAPI_MVP_P0_P1.md](OPENAPI_MVP_P0_P1.md)

| Livrable | Contenu attendu | Rôle |
|----------|-----------------|------|
| Modèle canonique Halo | types produit stables pour identité, history, match detail, career, assets, films, erreurs | éviter qu'un provider dicte la forme du produit |
| Matrice de capabilities | table titre × surfaces produit × niveau de support | décider ce qui est supporté, dégradé ou masqué |
| Registre d'isolation | auth, refdata, endpoints, assets, skill, films, PvE, economy, URLs externes | empêcher la diffusion de détails Halo Infinite dans l'API produit |
| Politique de dégradation | règles pour retourner une capability absente, partielle ou différée | garder l'API stable si un titre expose moins de données |

### Contours du modèle canonique à préparer

Le modèle canonique du futur portage Go doit au minimum couvrir :

1. identité joueur et résolutions gamertag/xuid ;
2. historique de matchs et pagination ;
3. match détaillé avec skill, events et assets utiles ;
4. progression carrière et compteurs agrégés ;
5. accès films et chunks ;
6. erreurs et limitations de provider exploitables côté produit.

Le détail de ce cadrage est désormais figé dans [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md).

### Capability map à figer

La capability map doit être versionnée et lisible avant la Phase 1. Elle doit au minimum décrire :

1. historique de matchs ;
2. détails match ;
3. skill/MMR ;
4. assets discovery ;
5. customization/economy ;
6. career rank ;
7. films et extraction d'armes ;
8. PvE / Firefight ;
9. lookups d'identité bulk.

La première version mono-titre de cette map est désormais figée dans [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md).

## Déclinaison documentaire du bootstrap et des types Go

Deux documents complémentaires prolongent maintenant ce cadrage :

1. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) pour la projection produit de la capability map dans le bootstrap ;
2. [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md) pour la forme cible des structs et interfaces Go canoniques.

Deux documents supplémentaires verrouillent la chaîne complète de transformation :

1. [HALO_INFINITE_CANONICAL_MAPPING.md](HALO_INFINITE_CANONICAL_MAPPING.md) pour la projection `payloads Halo Infinite -> canonique` ;
2. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md) pour la projection `canonique -> bootstrap/OpenAPI`.

Le dernier lot documentaire utile avant le code est maintenant matérialisé par :

1. [HALO_PROVIDER_ERROR_TAXONOMY.md](HALO_PROVIDER_ERROR_TAXONOMY.md) pour la traduction `provider -> erreur/limitation produit` ;
2. [OPENAPI_MVP_P0_P1.md](OPENAPI_MVP_P0_P1.md) pour le gel contractuel des premières routes HTTP à préserver.

## Algorithmes critiques à porter avec oracle

| Algorithme | Risque | Oracle attendu | Tolérance cible |
|------------|:------:|----------------|-----------------|
| Performance score | Haute | historique de matchs + score final attendu | à figer dans les golden values, cible stricte `< 0.01` |
| LUSR / TrueSkill adapté | Très haute | historique complet joueur, mu/sigma séquentiels | cible `< 0.1` sur mu/sigma |
| Sessions | Moyenne | IDs de session + labels + cas limites temporels | identité exacte des groupes |
| Killer / Victim | Haute | paires confirmées/estimées sur corpus de matchs | identité exacte des comptages |
| Citations custom | Haute | sorties par match sur corpus représentatif | identité exacte des règles déclenchées |
| Weapon parser | Très haute | corpus binaire figé + sorties reconciliées | identité exacte (portage `encoding/binary`) ; fallback subprocess Python uniquement si échec (D6) |
| Spawn detection | Haute | corpus multi-modes avec sorties attendues | identité exacte des zones/événements capturés |

## Familles de requêtes SQL à traiter en premier

### Read-only prioritaires

1. Bootstrap joueur.
2. Résolution gamertag via `v_gamertag_lookup`.
3. Résolution cascade des filtres.
4. Match history paginée.
5. Career rank history.
6. Explorer : matchs communs, rencontres, top adversaires.
7. Match view : events, médailles, weapons, scoreboard.

### Read-write à porter tardivement

1. insert `match_registry` ;
2. insert `match_participants` ;
3. insert `highlight_events` et `medals_earned` ;
4. insert `weapon_kills` et réconciliation ;
5. upsert `xuid_aliases` ;
6. écritures player DB : enrichments, awards, citations, career, skill ;
7. refresh `mv_*` ;
8. mise à jour du bitmask de backfill ;
9. persistance cache MSAL.

## Dépendances et validations spécifiques

| Sujet | Validation demandée |
|-------|---------------------|
| DuckDB Go | compat format avec les fichiers Python 1.4.4, types critiques, ATTACH, locks |
| Toolchain Windows | build CGo reproductible et documenté |
| MSAL Go | device code flow + stratégie de compat cache et refresh tokens |
| Parquet | lecture/écriture archive au moins sur le chemin utile |
| ffprobe | stratégie d'exécution externe claire et portable |

## Invariants de robustesse

1. Le portage Go doit conserver la dégradation gracieuse du Python sur données partielles ou manquantes.
2. Une métrique absente ne doit pas faire tomber tout un endpoint ou toute une page si le reste du payload peut encore être servi proprement.
3. Les cas limites de pagination, zéro match, joueur sans médailles, match PvE ou cache auth invalide doivent faire partie du corpus de vérification.

## Stratégie de tests

### Principe

La parité doit être démontrée avant le remplacement, pas discutée après coup.

### Tests minimaux à conserver

1. golden values JSON pour les surfaces de page ;
2. tests de parité SQL sur le même corpus DuckDB ;
3. tests de parité algorithmiques sur les modules analytiques ;
4. tests de contrats API ;
5. tests E2E Playwright contre le backend Go une fois les slices read-only prêtes ;
6. benchmarks simples sur les requêtes critiques.

### Artefacts qui doivent faire foi

1. le schéma OpenAPI versionné ;
2. les fixtures Golden Values ;
3. le corpus ref player / corpus de référence utilisé côté Python ;
4. les rapports d'écart explicitement acceptés.

## Rappels DuckDB / concurrence

1. Reproduire d'abord la sémantique Python de write lease avant tout durcissement.
2. Un writer logique par DB path.
3. Pool read-only borné ; pas d'ATTACH à chaque requête.
4. Les lectures peuvent coexister avec un sync, mais le comportement intermédiaire doit rester explicable.
5. Aucun handler ou helper ne doit ouvrir un writer ad hoc hors composant central.

## Zones à haut risque

1. compatibilité CGo sur Windows ;
2. gestion ATTACH avec `database/sql` ;
3. portage LUSR séquentiel ;
4. bitmask de backfill ;
5. cache MSAL et compat refresh tokens ;
6. parser binaire d'armes ;
7. drift entre produit courant et corpus de référence.
8. oubli d'une surface secondaire mais réelle comme PvE, Discord, médias, archives ou multi-joueurs.

## Quand relire le plan complet d'origine

Relire [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) si tu as besoin :

1. de la liste exhaustive des surfaces produit ;
2. du détail complet des algorithmes et tables ;
3. du catalogue complet Q1-Q16 et W1-W15 ;
4. des estimations détaillées et des sprints originaux ;
5. des formulations exactes des décisions D1-D10.
