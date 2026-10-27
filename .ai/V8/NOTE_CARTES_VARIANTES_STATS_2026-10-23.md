# Cartes variantes (Heavies, - Ranked) : regrouper aussi les statistiques par carte

> NOTE CLOSE le 2026-10-26 : la version complète a été exécutée, voir `.ai/V8/PLAN_CARTES_VARIANTES_STATS_2026-10-26.md`
> (relecture sur pièces, mesures E0, décisions D1-D3, étapes E0-E5). Ce qui suit est l'état de la note à la date du 2026-10-23.

## Ou on en est

Livre dans le commit `0fff7a5f0` (branche `release/v8.0.4`) : les FILTRES de cartes regroupent une
variante sous sa carte de base. « Thunderhead Heavies » (affiche « Du lourd sur Thunderhead ») et
« Thunderhead » ne forment plus qu'une option ; la choisir ramene les parties des deux. Meme chose pour
les cartes en « - Ranked ».

- Regle de rattachement : `apps/go-api/internal/analysis/map_variant.go`
- Suffixes par titre : `config/titles/halo_infinite/mappings/map_variants.toml`
- Lecture du catalogue et reecriture des lignes : `apps/go-api/internal/platform/duckdb/map_variants.go`
- Filtre de l'Explorateur : `apps/go-api/internal/service/match_history_service_map_filter.go`

Chaque variante garde son identifiant de carte. Les STATISTIQUES par carte n'ont pas bouge : une variante
y est comptee a part de sa carte de base.

## Le sujet restant (« option 2 »)

Ce que le joueur voit aujourd'hui : deux vignettes dans l'onglet Tactique (« Thunderhead » et « Du lourd
sur Thunderhead »), deux taux de victoire par carte dans l'Explorateur. L'option 2 les regrouperait en une
seule ligne chacun.

### Pourquoi ce n'est pas fait

1. **Le melange statistique.** « Lourds » est la meme carte avec une autre disposition de vehicules. Un
   joueur a 70 % en partie normale et 30 % en « Lourds » verrait 60 % une fois fusionne : moins juste pour
   juger son niveau.
2. **L'effort.** Dix fichiers environ, avec tests, et un risque de regression sur la Tactique, dont les
   positions des morts, les zones et le fond du rejeu se cherchent par identifiant de carte.
3. **Un garde-rail** (`apps/go-api/internal/archlint/no_group_by_registry_name_test.go`) interdit de
   regrouper par NOM de carte : la fusion doit passer par une table de correspondance variante vers base,
   pas par un raccourci.

### Surfaces reperees (a re-verifier)

Elles regroupent aujourd'hui par identifiant de carte, ou par libelle :

- Tactique : `apps/go-api/internal/platform/duckdb/tactical_repo.go` (cartes jouees, une rangee par
  identifiant) et `apps/go-api/internal/platform/duckdb/tactical_repo_morts_par_carte.go`
- Decoupage par carte de l'analyse : `apps/go-api/internal/analysis/breakdown/by_map.go`
- Motifs « par carte » : `apps/go-api/internal/analysis/patterns/context.go`
- Taux de victoire par carte de l'Explorateur : `computeMapWinRates`, appele depuis
  `apps/go-api/internal/service/match_history_service_filters.go` (regroupe par libelle affiche)
- Hors perimetre probable : `apps/go-api/internal/sync/replayartifacts/zones_cartes.go` (artefacts de
  rejeu, cote sync)

Non releves et a chercher : citations de l'accueil, carriere, score d'engagement.

## Deux facons de reprendre

- **Version legere (conseillee si les doublons gênent).** Regrouper seulement l'AFFICHAGE : les vignettes
  de la Tactique et le taux de victoire par carte de l'Explorateur. Positions, zones et rejeu restent par
  identifiant, donc intacts. Deux ou trois endroits. Defaut : les parties « Lourds » entrent dans le taux
  de victoire de la carte. Variante possible : une seule vignette, mais un taux de victoire distinct pour
  la variante.
- **Version complete.** Tout regrouper, via une table de correspondance variante vers base chargee une
  fois, et des requetes de la Tactique qui cherchent avec tous les identifiants du groupe.

## Condition de reprise

Quand les deux vignettes ou les deux taux separes gênent a l'usage. Avant de coder : re-lister les
surfaces par un grep des regroupements par `map_id` et par libelle de carte, mesurer sur la base reelle
combien de parties sont en variante (relevé du 2026-10-23 : 11 cartes « Heavies », de 3 a 30 parties
chacune, et 6 cartes « - Ranked », dont Live Fire 445 parties), puis decider avec le user.

## Limites connues de ce qui est livre

- Une selection de carte enregistree sous l'ancien libelle de variante (« Du lourd sur X ») n'est plus une
  option des filtres du pipeline.
- Beaver Canyon et Kusini Bay (« - Ranked ») n'ont pas leur carte de base au catalogue des cartes : elles
  restent seules dans les filtres.
