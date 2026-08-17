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

- [x] **Phase 0** — ce plan.
- [x] **Phase 1** — verifier l'inventaire `ecs_inventaire_2026-08-18.json` contre le code de la
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


---

# PHASE 1 — VERIFICATION DE L'INVENTAIRE CONTRE LE CODE DE LA BASE

Methode : le REGISTRE DU FILM fait foi pour `(ti, i, nom, niveau)` ; l'AST de `consumeByName`
fait foi pour le STATUT et la SOURCE. Statut derive mecaniquement : `porte` = tous les retours
du cas rendent le litteral `true` ; `partiel` = au moins un retour non-`true` ; `non_porte` =
aucun `case` (branche `default`, 0 bit consomme) ; `deser_non_cable` = grammaire ecrite sans
appelant.

Bilan : **73 corrections** et **318 ajouts** sur les 216 lignes de l'inventaire.

## Corrections de STATUT (2) — le fond

| ligne | avant | apres | piece |
|---|---|---|---|
| `ti=35 i=60 simulation-state-component` | `non porte` | **`partiel`** | `traverse.go:861-868` — la grammaire est COMPLETE depuis R7-b ; ce qui reste est le drapeau `simStateComplete` (`traverse.go:1130`, defaut `false`), pas la grammaire |
| `ti=23 i=0..31 selectable-zone-data-component` (x32) | `non porte` | **`deser_non_cable`** | `components_world.go:106` — le deser EXISTE (`consumeSelectableZoneData`, `FUN_142ed6cec`, `//nolint:unused`), il n'a pas d'appelant. Ce n'est pas la meme chose que « pas de grammaire » |

## Corrections d'AVERTISSEMENT GLOBAL (3) — l'entete du JSON etait perime

L'entete de l'inventaire annoncait trois defauts « vivant seulement dans `LevelUp-wt-kfloop` ».
Verification sur pieces dans la base `90bc83c12` : **les trois sont dans le code**.

| affirmation de l'inventaire | realite de la base |
|---|---|
| « `components_batch7.go:27` lit encore le bloc TLV quand le bit vaut 1 » | FAUX. `components_batch7.go:42` : `if br.ReadBit() { return }` — la polarite est CORRIGEE, commentaire date du 2026-08-17 |
| « `traverse.go:644` rend `ported=false` sur i57 `tag==3` » | Le cas i57 est en `traverse.go:850` ; il rend la valeur de `consumeBipedSpartanAbility` — statut `partiel`, inchange sur le fond mais la ligne citee est perimee |
| « la queue d'i60 vit dans un autre worktree » | FAUX. `traverse.go:861-868` porte la grammaire complete ; `simStateComplete` est la seule porte restante |

## Corrections de NOM (5 de fond + 35 cosmetiques)

De fond — l'inventaire nommait un composant qui n'est PAS a cet index dans le registre :

| ligne | inventaire | registre du film |
|---|---|---|
| `ti=2 i=0` | `game-engine-campaign-timer-component` | **`game-engine-team-mapping-component`** (le campaign-timer est a `i12` de ti 0, 1 ET 2) |
| `ti=43 i=0` | `device-position-component` | **`object-position-component`** (device-position est a `ti=43 i=18`) |
| `ti=35 i=59` | `biped-spartan-ability-non-predicted-state-component` | **`biped-spartan-ability-non-predicted-state`** (sans suffixe ; l'autre orthographe est un ALIAS du dispatch) |
| `ti=5 i=22` | `(non identifie)` | **`player-aim-assist-component`** |
| `ti=5 i=24` | `(non identifie)` | **`player-desired-frame-configuration-component`** |

Cosmetiques (35) : l'inventaire agregait des repetitions dans le nom
(`weapon-state-ammo (emplacement 1)`, `selectable-zone-data-component (x32, i0..i31)`,
`(bloc objet partage) ... (presume)`). La table les developpe, une ligne par index reel.

## Corrections de SOURCE `fichier:ligne` (30)

Toutes les references `traverse.go:NNN` de l'inventaire ont derive (fusion R7-a..e + lot poses).
Exemples : `game-engine-sudden-death` 733 -> **633** · `object-position-component` (ti=43 i0)
880 -> **259** · `unit-desired-aiming-vector` 916 -> **918** · `asset-transform` « 919-930 » ->
**921** · `spawn-filter-weight` « 931-940 » -> **933**. La table porte desormais la ligne
recalculee par l'AST, et G1 la recalcule a chaque execution : elle ne peut plus deriver en
silence.

## Correction de PERIMETRE (1)

`ti=11 i=-1` : l'inventaire portait une ligne fourre-tout (« 8 composants restants, ordre non
releve »). Le registre donne les 34 composants de `ti=11` nommes et indexes : la ligne
fourre-tout disparait, remplacee par les vrais index.

## AJOUTS (318)

**16 archetypes** du registre etaient absents de l'inventaire (33 documentes sur 49 porteurs) :

| ti | nom prudent (prefixe commun) | composants |
|---|---|---|
| 1 | inconnu (game-engine-*) | 15 |
| 3 | inconnu | 2 |
| 7 | `forge-sim-generic-*` | 64 |
| 19 | `sound-placement-state-*` | 32 |
| 24 | inconnu | 27 |
| 25 | `powerframe-player-selection-*` | 1 |
| 26 | `supply-lines-*` | 33 |
| 27 | `supply-lines-item-*` | 64 |
| 28 | `narrative-moment-beat-*` | 1 |
| 31 | `tacmap-*` | 11 |
| 36 | inconnu | 20 |
| 39 | inconnu | 20 |
| 45 | `matchflow-*` | 2 |
| 46 | `state-broker-state-*` | 64 |
| 48 | `forge-player-data-*` | 2 |
| 49 | `managed-radialmenu-*` | 35 |

**302 composants** traites par le code (`porte` ou `partiel`) mais absents de l'inventaire.
L'essentiel vient de la reutilisation d'un meme composant par plusieurs archetypes, que
l'inventaire ne comptait qu'une fois : `game-engine-*` sur ti 0/1/2, `effect-state-data` sur
les 32 index de ti=18, `nav-cutscene-flag` sur ti=15, `branch-script-results` sur ti=16, le
bloc objet partage (`object-*` i0..i17) sur ti 36/37/38/39/40/41/42/43.

**14 alias d'ecriture** acceptes par `consumeByName` et absents du registre des films temoins :
`biped-action`, `biped-control-context`, `biped-desired-ability-set`, `biped-desired-grenade-set`,
`biped-low-frequency-data`, `biped-malleable-property`, `biped-map-editor-flag`,
`biped-mobility-action`, `biped-slide`, `biped-spartan-ability-energy`,
`biped-spartan-ability-non-predicted-state-component`, `game-engine-team-mapping`,
`simulation-state`, `simulation-state-playback`. Ils entrent a la table en `ti = -1`,
statut `alias`.

## Ce que la verification NE change pas

`i57`, `i59`, `i63` restent `partiel` (retour data-dependant), conforme a l'inventaire.
Les 191 `porte` de l'inventaire sont tous confirmes `porte` par l'AST : aucun statut de
portage n'a ete infirme a la baisse.
