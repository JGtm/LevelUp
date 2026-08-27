# PLAN — LOT A : Assaut (bombe), donnee de rejeu 2D

> Ecrit le 2026-08-27 par la session superviseur. Lot A du chantier
> `.ai/V7.5/PLAN_MODES_PORTEURS_2026-08-27.md` (§3), ouvert apres la cloture du lot O
> (fusion `42d553fc0` : P1 infirmee, th=10 heartbeat, statborg sans compteur de crane).
> Branche : `wt/assaut` (worktree `../LevelUp-wt-assaut`), base `7dfb05ca6`.
> Contrat d'execution : skill `plan-execution`. Regles NON NEGOCIABLES du chantier :
> filmproc pour tout decodage, protocole COMMITE avant toute mesure, seuils jamais
> abaisses, temoins negatifs, arret propre au seuil rate, decouvertes consignees
> au §5 et jamais traitees.

## 0. CE QUI EST SU (verifie le 2026-08-27, ne pas re-etablir)

- **Corpus : 9 matchs, 9 films TOUS en cache** (`data/cache/film_manifests` + `film_chunks`
  du depot principal) :
  - Neutral Bomb : `35b75a31` (Origin), `ce083875` (Origin), `69b16f5d` (Origin),
    `3d58eb37` (Absolution), `34bb3bc8` (Neutral Bomb Squad, Rat's Nest, 2025-04)
  - One Bomb : `df8fcbef`, `c75f33b8`, `9f57c612` (Curfew x3)
  - Husky Raid : `1c01e34f` (Urban Raid — carte FORGE : piege connu « .mvar = canevas+rack »,
    catalogues probablement absents)
- **AUCUN oracle API par joueur** : les 9 payloads bruts ont ete dumpes le 27/08 —
  CoreStats+PvpStats seuls, pas de bloc AssaultStats. Les gates de ce lot sont donc
  INTERNES au film : temoins spatiaux et de selectivite, corroboration par le score de
  mode (armement/explosion = increments dates) et les manches.
- **Le role statique `assault_bomb` existe** (4/4 objets en `team_index = -1`, servi par
  `objective_roles.toml`) ; `ObjectiveTypeOf` ne connait PAS l'Assaut (aucun evenement
  nomme au statborg attendu — a confirmer en A4, pas a supposer).
- **La recette d'identite d'un objet porte a fait ses preuves deux fois** : drapeau
  `0x2A392328`, crane `0x0017592C` — mot MPP 32 bits des creations `ti=42`, naissance au
  socle, temoin de selectivite. Instruments de reference :
  `attachement_phase0_drapeau_test.go`, chaine `flag_objects.go` / `build_objectives_live.go`.
- **Lecon Live Fire du lot O** : un film sans bornes de quantification
  (`map_quant_bounds.json`) ou sans catalogue d'objectifs (`map_objectives.json`) est
  INDECODABLE ou infirmable — l'exclure avec sa raison, ne pas reparer les catalogues
  (decouverte au §5).
- **CLI durable `cmd/statnames-sweep`** livre par le lot O (balayage statborg + mode
  `-confront`). Le pont statborg peut etre etroit (asymetrie consignee au registre).
- Schema d'artefact courant : 21 ; contrat : 39. Le calque `objectiveObjects` publie des
  vies libres par famille depuis le MANIFESTE (`config/titles/halo_infinite/mappings/
  replay_labels.toml`, table `[[objective_objects]]`) — l'ajout d'une famille est de la
  DONNEE ; verifier en A2 si un bump de schema est necessaire (a priori non).

## 1. OBJECTIF ET LIVRABLES

Livrable vise (decision produit du plan chantier, §1.2) : **les vies libres de la BOMBE**
au patron du crane libre, et — si son gate tient — **l'etat des SITES d'amorcage**
(jauge/armement). Le PORTEUR de bombe n'est PAS promis et ne se publie pas dans ce lot
(sans oracle API, et la mecanique porteur est restee fermee au lot O).

## 2. PHASES

> Une phase a la fois ; items statues `[x]`/`[~]`/`[!]` ; commits `assaut-a(<phase>): ...` ;
> jamais `git add -A` ; pas de push ; thought_log et REGISTRE_REPORTS jamais touches
> (textes au CR). Donnees du depot principal via `LEVELUP_REPO_ROOT`, lectures DuckDB par
> `OpenReadForQuery` ou `cmd/diag_q` (read-only). Un seul build/test Go a la fois.

### A0 — Recensement sur pieces + protocole COMMITE (aucune mesure avant le commit)

- [ ] A0.1 Lire : le plan chantier (§3 LOT A), le registre (entrees lot O du 27/08), les
      en-tetes de `flag_objects.go`, `build_objectives_live.go`, `ground_weapon_rules.go`
      (recette ti=42), `cmd/statnames-sweep/*.go`, l'entree `assault_bomb` de
      `objective_roles.toml` et ce que `service/replay_map_objectives.go` sert pour les
      cartes du corpus.
- [ ] A0.2 Qualifier les 9 films : bornes de quantification presentes ? catalogue
      d'objectifs present (sites `assault_bomb` par carte) ? pont bipede >= 50 % de slots
      nommes (meme instrument que le lot O) ? Publier le tableau film -> verdict
      admis/exclu + raison DANS le protocole. Husky Raid : exclu attendu (Forge) — le
      constater, pas le presumer.
- [ ] A0.3 Relever pour chaque film admis les manches et le score de mode
      (`scoreTimeline`, schema 12) : les increments d'armement/explosion existent-ils et
      se datent-ils ? C'est la corroboration d'A3 — figer ces releves au protocole.
- [ ] A0.4 Ecrire et COMMITTER `.ai/V7.5/replay2d/registre_film/A_PROTOCOLE.md` :
      corpus admis fige, seuils recopies du §3 SANS modification, temoins, repartition
      moities disjointes pour A4. Citer le hash au CR.

### A1 — Identite de l'objet bombe (gate ECRIT, herite du crane/drapeau)

- [ ] A1.1 Instrument sous garde d'environnement (patron des campagnes D) : mots MPP des
      creations `ti=42` ecartees du catalogue d'armes, nes a <= 3 m d'un site
      `assault_bomb` du catalogue statique.
- [ ] A1.2 Temoin de selectivite : aucun AUTRE mot ecarte ne reunit la naissance au site
      ET la coincidence avec le debut de manche / la remise en jeu.
- [ ] A1.3 GATE (ecrit ici, ne bouge pas) : **UN SEUL mot candidat, LE MEME sur >= 2
      films admis, temoin = 0 autre candidat.** Log fige `A1_identite_bombe.log`.
      Si rate : bombe `[!]` avec chiffres, le lot continue en A3/A4 (A2 tombe).

### A2 — Publication des vies libres de la bombe (SEULEMENT si A1 tient)

- [ ] A2.1 Entree `[[objective_objects]]` famille `bomb` au manifeste (EN+FR), exclusions
      de socles verifiees comme pour `ball` (`ground_weapon_flag_exclusion_test.go` :
      etendre le garde si necessaire).
- [ ] A2.2 Verifier si un bump de schema est requis (a priori non : famille = donnee).
      S'il l'est : STOP, arbitrage superviseur au CR (ne pas bumper seul).
- [ ] A2.3 Rendu web : la famille `bomb` dans le calque `objectiveObjects` (glyphe
      distinct, encre neutre, patron du crane). Strings i18n FR+EN si un libelle surface.
- [ ] A2.4 Re-cuisson des TEMOINS seulement (>= 1 film admis par variante Neutral/One
      Bomb), avec verification du CONTENU (recette du registre : bonne version != bonne
      configuration ; racine temporaire a jonctions — config du worktree, data du
      principal). Publier le compte de vies libres par temoin.

### A3 — Etat des sites d'amorcage (diagnostic puis publication conditionnelle)

- [ ] A3.1 Chercher le canal `ti=13` aux sites d'amorcage des films admis (structure de
      rampe/jauge comme la capture de Bastion ; tag 4 de propriete ; designateur type
      colline). Diagnostic d'abord : quels slots, quels tags, quelle correlation aux
      armements dates par le score de mode (releves A0.3).
- [ ] A3.2 GATE de publication (ecrit ici) : accord canal <-> armements dates >= **90 %**
      des confrontations possibles sur >= **2** films admis ; temoin spatial (formes
      decalees de 12 m) <= **20 %** ; sinon `[!]` diagnostic consigne, rien ne se publie.
      Log fige `A3_sites_amorcage.log`.
- [ ] A3.3 Si gate tenu : publication au patron `zoneStates` (roles/table du titre,
      degradation par absence de donnee), rendu jauge sur la forme (patron Bastion),
      re-cuisson temoins avec verification de contenu.

### A4 — Statborg Assaut (inventaire borne avec l'outil du lot O)

- [ ] A4.1 `cmd/statnames-sweep` sur les films admis ; confrontation aux compteurs
      derivables SANS API : morts du porteur presume (kill feed), armements/explosions
      dates par le score de mode, sur les moities disjointes ecrites en A0.4.
- [ ] A4.2 Verdict nomme (replique / ne replique pas quoi), log fige
      `A4_statborg_assaut.log`. Diagnostic seulement — aucune publication depuis A4.

## 3. GATES DU LOT (a rejouer dans le worktree, codes de sortie au CR)

- Protocole commite AVANT mesures (historique git en temoigne, un seul commit).
- Logs figes commites avec leur phase.
- Si A2/A3 publient : `go test` des packages touches + `go test ./internal/contracttest/...`,
  `tsc -b` (cache purge), vitest `src/features/match-replay`, lint web 0 erreur,
  garde-rails de parite schema web/Go verts.
- `go vet ./...` propre sur les packages touches ; `go build ./...` exit 0.
- Aucun push ; arbre propre a la cloture.

## 4. COMPTE-RENDU ATTENDU

Tableau de qualification (film -> admis/exclu + raison) ; verdict A1 avec le mot et son
temoin ; comptes de vies libres par temoin re-cuit (si A2) ; verdict A3 chiffre (accords,
temoin) ; verdict A4 ; liste des commits ; textes prets pour thought_log et
REGISTRE_REPORTS ; decouvertes du §5.

## 5. DECOUVERTES (a consigner, ne pas traiter)

- (vide a l'ouverture)
