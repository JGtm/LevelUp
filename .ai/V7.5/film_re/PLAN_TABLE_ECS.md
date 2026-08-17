# PLAN — TABLE ECS CONSOLIDEE (grammaire du film Theater, archetype x composant)

Lot pilote, branche `wt/table-ecs`, base `feat/v75` = `90bc83c12`.
Contrat d'execution : `.claude/skills/plan-execution/SKILL.md`.

## Objectif

UNE table versionnee, lisible par machine, de la grammaire ECS du film : une ligne par
(archetype, composant) tel que le FILM le declare, portant le statut de portage du decodeur,
sa source `fichier:ligne`, l'adresse du deserialiseur quand elle est connue, et ce que le
composant alimente (ou non) dans le document de rejeu. Trois garde-rails l'empechent de
diverger du code.

La table N'EST PAS la grammaire bit-exacte : celle-ci reste dans le code (`consumeByName`,
`components_*.go`, `keyframe_*.go`). La table dit QUI EXISTE, QUEL EST SON STATUT, OU LE LIRE.

## Etat des lieux (mesure sur pieces, 2026-08-18)

| fait | valeur mesuree | comment |
|---|---|---|
| blocs d'archetype rendus par `ParseRegistryChunk` | **118** | dump du chunk_00, 3 films temoins |
| blocs PORTEURS d'au moins un composant | **49** (0..7, 9..49 ; le bloc 8 est vide) | idem |
| lignes (archetype, composant) declarees | **1 067** | idem |
| noms de composants distincts | 325 | idem |
| registre identique d'un film a l'autre | OUI, octet pour octet (`000d5950`, `00502e52`, `07aa428d`) | `diff` des 3 dumps |
| etiquettes `case` de `consumeByName` | 189 (183 litteraux + 6 constantes) | AST |
| etiquettes `case` absentes du registre (alias d'ecriture) | 14 | comm |
| lignes ou `Flags[k+1] != Flags[k]` (decalage R7-e) | 178 sur 1 067 | dump |

**Le chiffre « 174 archetypes » du brief et de l'inventaire est FAUX pour ce que le decodeur
lit.** Il ne correspond a aucune grandeur du registre tel que `registry.go` le parse : 118
blocs, 49 non vides. Le critere de couverture est re-enonce en consequence ci-dessous.

## Critere de fin

1. La table couvre les **1 067** lignes (archetype, composant, niveau) du registre des films
   temoins — soit les **49** archetypes porteurs sur les 118 blocs — plus les 14 alias
   d'ecriture reconnus par le code.
2. Pour chaque composant DECODE (present comme `case` de `consumeByName`) la ligne porte un
   statut, une source `fichier:ligne` et l'adresse du deser quand elle est connue.
3. Trois garde-rails verts :
   - **G1 code <-> table** (sans film) : toute etiquette `case` de `consumeByName` est un
     composant de la table ; reciproquement toute ligne `porte`/`partiel` correspond a un
     `case`, et toute ligne `non_porte` n'en est pas un. Le statut derive de l'AST :
     `porte` = tous les retours du cas rendent le litteral `true` ; `partiel` = au moins un
     retour non-`true` (desync data-dependant ou drapeau).
   - **G2 film <-> table** (garde `ECS_TABLE_FILM`, SKIP propre sans film) : archetypes,
     composants et NIVEAUX de la table = ceux lus dans le registre du film.
   - **G3 table <-> document** : chaque champ cite en `doc_field` existe dans `document.go`.

## Format

`apps/go-api/internal/analysis/filmdec/testdata/ecs_table.tsv` — UTF-8, tabulations, en-tete,
tri par `ti` puis `i`. Colonnes :

`ti	archetype	i	component	level	status	deser_addr	grammar	bits_typ	code_source	doc_field	product_use	meaning_fr	exploitable_fr	confidence	notes`

Statuts : `porte` · `partiel` · `non_porte` · `deser_non_cable` (grammaire ecrite, aucun
appelant) · `alias` (autre orthographe acceptee par le code, `ti = -1`).

`level` = `Flags[k]`, ce que `registry.go` sert et ce que le traverseur passe aux desers. Quand
la lecture du JEU (`Flags[k+1]`, lot R7-e) differe, `notes` porte `niveau_jeu=N`.

## Phases

- [ ] **Phase 0** — ce plan.
- [ ] **Phase 1** — verifier l'inventaire `ecs_inventaire_2026-08-18.json` contre le code de la
      base : statut et source de chacune des 216 lignes, corrections consignees (avant -> apres),
      completion par les composants traites par le code mais absents, et par les archetypes du
      registre absents.
- [ ] **Phase 2** — la table `ecs_table.tsv` + son `README` (20 L). Lecteur Go minuscule
      seulement s'il sert aux garde-rails.
- [ ] **Phase 3** — `ecs_table_guard_test.go` : G1, G2, G3. Les trois verts, G2 joue sur les
      3 films temoins.
- [ ] **Phase 4** — docs : note de tete dans `WALK_PORT_NOTES.md` et
      `RECETTE_DECODAGE_FILM_CHUNKS.md`, index `.ai/V7.5/README.md`, ligne au registre des
      reports, suppression du JSON brut (la table le remplace), entree thought_log.

## Journal

- 2026-08-18 — Phase 0 ouverte. Registre dumpe sur les 3 films temoins via un instrument
  temporaire, resultats ci-dessus. Le chiffre du brief (174) est refute sur pieces.

## Decouvertes (non traitees — regle 7)

