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

- [x] A0.1 Lire : le plan chantier (§3 LOT A), le registre (entrees lot O du 27/08), les
      en-tetes de `flag_objects.go`, `build_objectives_live.go`, `ground_weapon_rules.go`
      (recette ti=42), `cmd/statnames-sweep/*.go`, l'entree `assault_bomb` de
      `objective_roles.toml` et ce que `service/replay_map_objectives.go` sert pour les
      cartes du corpus.
      — Fait 2026-08-27. Constat cle de la lecture sur pieces : les 4 sites
      `assault_bomb` du catalogue vivent sur Isolation/Snowbound/The Pit/High Ground —
      AUCUNE carte du corpus ; l'entree `[[modes]]` Assaut de `objective_roles.toml` ne
      sert donc RIEN sur ces matchs aujourd'hui.
- [x] A0.2 Qualifier les 9 films : bornes de quantification presentes ? catalogue
      d'objectifs present (sites `assault_bomb` par carte) ? pont bipede >= 50 % de slots
      nommes (meme instrument que le lot O) ? Publier le tableau film -> verdict
      admis/exclu + raison DANS le protocole. Husky Raid : exclu attendu (Forge) — le
      constater, pas le presumer.
      — Fait 2026-08-27 (`TestAssautA0Qualification`, un film/processus, log
      `A_P0_qualification.log`) : 8 films ADMIS (pont 75,0-95,1 %), `ce083875` EXCLU
      (pont 19/180 = 10,6 %). Bornes presentes 9/9 (y compris Urban Raid — le Forge
      etait annonce exclu, le CONSTAT le garde : sa carte est absente du catalogue
      d'OBJECTIFS, comme Rat's Nest). Sites `assault_bomb` : 0 sur 9/9 films — le
      corpus d'ANCRAGE AU SITE de A1/A3 est VIDE (decouverte au §5, pas de reparation).
- [x] A0.3 Relever pour chaque film admis les manches et le score de mode
      (`scoreTimeline`, schema 12) : les increments d'armement/explosion existent-ils et
      se datent-ils ? C'est la corroboration d'A3 — figer ces releves au protocole.
      — Fait 2026-08-27 (meme passe, releves figes au protocole §2) : chaque EXPLOSION
      se date (1 increment = 1 point), score film = score API sur 9/9 ; l'ARMEMENT n'a
      aucun increment propre ; `RealRounds` refuse les manches de One Bomb (1 emission
      par manche) — releve BRUT publie, decouverte au §5.
- [x] A0.4 Ecrire et COMMITTER `.ai/V7.5/replay2d/registre_film/A_PROTOCOLE.md` :
      corpus admis fige, seuils recopies du §3 SANS modification, temoins, repartition
      moities disjointes pour A4. Citer le hash au CR.
      — Fait 2026-08-27 : protocole + log de qualification + oracle participants fige
      (`A_oracle_participants.tsv`, 104 lignes) commites ensemble (hash au CR). Moities
      A4 : recherche `1c01e34f`,`35b75a31`,`69b16f5d`,`c75f33b8` / verification
      `34bb3bc8`,`3d58eb37`,`9f57c612`,`df8fcbef`.

### A1 — Identite de l'objet bombe (gate ECRIT, herite du crane/drapeau)

- [x] A1.1 Instrument sous garde d'environnement (patron des campagnes D) : mots MPP des
      creations `ti=42` ecartees du catalogue d'armes, nes a <= 3 m d'un site
      `assault_bomb` du catalogue statique.
      — Fait 2026-08-27 (`assaut_a1_identite_test.go`, recette D4, classes temporelles du
      protocole §3). 7 films mesures ; `34bb3bc8` NON EXPLOITABLE (0/617 creations
      resolues au catalogue d'armes = bloc MPP lu aux mauvaises largeurs — la lecon D4,
      ni pour ni contre). Denominateurs publies : 57-350 ecartees, 32-272 mots distincts
      par film.
- [x] A1.2 Temoin de selectivite : aucun AUTRE mot ecarte ne reunit la naissance au site
      ET la coincidence avec le debut de manche / la remise en jeu.
      — Fait 2026-08-27 : les deux jambes publiees separement par mot. La jambe du SITE
      est VIDE par construction (0 site au catalogue, protocole §1) ; la jambe
      temporelle vit (mots recurrents a coincidences non nulles : `0x3FEE4FCF` sur 7/7
      films mesures, `0xE9E7FF79` sur Absolution + les 3 Curfew) — aucun mot ne peut
      reunir les deux, AUCUN candidat n'est elu.
- [!] A1.3 GATE (ecrit ici, ne bouge pas) : **UN SEUL mot candidat, LE MEME sur >= 2
      films admis, temoin = 0 autre candidat.** Log fige `A1_identite_bombe.log`.
      Si rate : bombe `[!]` avec chiffres, le lot continue en A3/A4 (A2 tombe).
      — GATE RATE 2026-08-27, chiffre : 0 candidat sur 7/7 films mesures (1 exige sur
      >= 2), cause nommee = ancrage au site INDISPONIBLE (0 site `assault_bomb` au
      catalogue pour les 5 cartes du corpus — §1 du protocole, decouverte §5). Ce zero
      chiffre le catalogue manquant, il ne refute PAS l'objet bombe : condition de
      reprise = sites `assault_bomb` des cartes du corpus au catalogue (re-extraction
      mvar / chasse au hash), puis rejouer CET instrument tel quel. **BOMBE `[!]` — A2
      TOMBE, le lot continue en A3/A4.**

### A2 — Publication des vies libres de la bombe (SEULEMENT si A1 tient)

> A2 TOMBE EN BLOC le 2026-08-27 : le gate A1.3 est rate (0 candidat, ancrage au site
> indisponible — voir A1.3). La condition d'ouverture du plan (« SEULEMENT si A1 tient »)
> n'est pas remplie ; aucun item ci-dessous n'est executable sans identite d'objet.

- [!] A2.1 Entree `[[objective_objects]]` famille `bomb` au manifeste (EN+FR), exclusions
      de socles verifiees comme pour `ball` (`ground_weapon_flag_exclusion_test.go` :
      etendre le garde si necessaire).
      — Sans mot MPP etabli, il n'y a pas d'`id` a ecrire au manifeste.
- [!] A2.2 Verifier si un bump de schema est requis (a priori non : famille = donnee).
      S'il l'est : STOP, arbitrage superviseur au CR (ne pas bumper seul).
      — Tombe avec A2.1.
- [!] A2.3 Rendu web : la famille `bomb` dans le calque `objectiveObjects` (glyphe
      distinct, encre neutre, patron du crane). Strings i18n FR+EN si un libelle surface.
      — Tombe avec A2.1.
- [!] A2.4 Re-cuisson des TEMOINS seulement (>= 1 film admis par variante Neutral/One
      Bomb), avec verification du CONTENU (recette du registre : bonne version != bonne
      configuration ; racine temporaire a jonctions — config du worktree, data du
      principal). Publier le compte de vies libres par temoin.
      — Rien a re-cuire : aucune donnee nouvelle ne surface.

### A3 — Etat des sites d'amorcage (diagnostic puis publication conditionnelle)

- [x] A3.1 Chercher le canal `ti=13` aux sites d'amorcage des films admis (structure de
      rampe/jauge comme la capture de Bastion ; tag 4 de propriete ; designateur type
      colline). Diagnostic d'abord : quels slots, quels tags, quelle correlation aux
      armements dates par le score de mode (releves A0.3).
      — Fait 2026-08-27 (`assaut_a3_ti13_test.go` : balayage p2a + `findZoneRamps` de
      production + explosions datees, log `A3_sites_amorcage.log`). VERDICT NEGATIF NET
      sur 8/8 films : (1) l'ANCRAGE `ti=13` est au niveau du HASARD STRUCTUREL — chainage
      1,9-16,4 % contre 87-99 % en KOTH arene, et surtout indistinguable du temoin decale
      de 3 bits (2,8-7,1 %) sur 7/8 films (seule exception : `69b16f5d`, 16,4 vs 2,9 %) ;
      (2) ZERO rampe de jauge (definition Bastion intacte : >= 3 echantillons, >= 4096
      quanta) sur les 8 films ; (3) donc 0 confrontation aux explosions possible. La
      contamination d'ancrage documentee sur BTB (registre 27/08) se retrouve ici SUR
      CARTES ARENE : en Assaut, la bande ti=13 lue est fantome ou le mode n'emet pas de
      propriete geree de type jauge/designateur.
- [!] A3.2 GATE de publication (ecrit ici) : accord canal <-> armements dates >= **90 %**
      des confrontations possibles sur >= **2** films admis ; temoin spatial (formes
      decalees de 12 m) <= **20 %** ; sinon `[!]` diagnostic consigne, rien ne se publie.
      Log fige `A3_sites_amorcage.log`.
      — GATE NON TENU 2026-08-27, doublement : 0 confrontation possible (aucune rampe,
      denominateur nul — 90 % de rien n'existe pas) ET temoin spatial inmesurable (aucune
      forme de site au catalogue, protocole §1/§4). RIEN ne se publie. Conditions de
      reprise : (a) sites `assault_bomb` au catalogue (la meme condition que A1) ET
      (b) un ancrage ti=13 fiabilise sur ces films (le chantier de decodage d'adressage
      des slots deja consigne au registre pour BTB — il couvre desormais l'Assaut arene).
- [!] A3.3 Si gate tenu : publication au patron `zoneStates` (roles/table du titre,
      degradation par absence de donnee), rendu jauge sur la forme (patron Bastion),
      re-cuisson temoins avec verification de contenu.
      — Tombe avec A3.2 (gate non tenu).

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

- (A0, 2026-08-27) **Le catalogue d'objectifs n'a de sites `assault_bomb` que sur 4
  cartes sans film (Isolation, Snowbound, The Pit, High Ground)** ; Origin, Curfew et
  Absolution sont au catalogue SANS objet de ce role, Rat's Nest et Urban Raid en sont
  absentes. Deux hypotheses non departagees ici : la variante `.mvar` extraite le 25/08
  ne porte pas les objets d'Assaut, OU les cartes recentes les portent sous un label de
  role NON RESOLU (patron KOTH : hash sans nom, cf. `mapvar/objectives.go`). Toute
  reprise de A1/A2/A3 passe par la re-extraction/chasse au hash — chantier catalogue,
  PAS ce lot.
- (A0, 2026-08-27) **`RealRounds` refuse structurellement les manches de One Bomb** :
  une manche s'y termine sur UN point de mode, sous le critere de suite coherente
  (`statMinRoundRun`). Consequence : `SeriesByRound`/`SeriesTotal` (et tout ce qui les
  consomme — courbe de score du rejeu, TSV de `statnames-sweep`) ne retiennent que la
  manche 0 sur les films One Bomb ; le bandeau de score du rejeu est donc PARTIEL sur
  ces matchs (meme classe que l'entree KOTH du registre du 26/08). Le releve BRUT du log
  A0 date pourtant les 4 manches et leurs explosions, somme = score API 9/9.
- (A0, 2026-08-27) `ce083875` (Origin, 949 s) : une emission de score de mode PARASITE
  (valeur 127, slot 8, t=273547 ms) ecartee par la plus longue sous-suite croissante —
  noter que le canal du score n'est pas exempt de parasites sur Assaut.
- (A0, 2026-08-27) Urban Raid (carte Forge) A ses bornes de quantification au catalogue
  (`map_quant_bounds.json`) et son film decode proprement (pont 93,5 %) — le piege
  « carte Forge = canevas+rack » vaut pour la GEOMETRIE/objectifs, pas pour les bornes.
