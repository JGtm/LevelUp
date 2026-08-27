# PLAN — LOT C : Catalogues de cartes (bornes + objectifs), deblocages Oddball/Assaut/Land Grab

> Ecrit le 2026-08-27 par la session superviseur. Ce lot leve le verrou COMMUN nomme par les
> lots O et A : des catalogues de cartes incomplets excluent des films (Oddball : Live Fire
> x2 sans bornes de quantification ; Lattice sans catalogue d'objectifs) et vident des gates
> par construction (Assaut : 0 site `assault_bomb` sur les 5 cartes du corpus). Il absorbe le
> lot L (cablage `landgrab_zone`), meme famille de travail. Branche : `wt/catalogues`
> (worktree `../LevelUp-wt-catalogues`), base `b14140106`.
> Contrat : skill `plan-execution`. Regles du chantier : filmproc pour tout decodage, gates
> ecrits ici et jamais abaisses, decouvertes au §5 non traitees, aucun fix hors perimetre.
> IMPORTANT : ce lot repare des CATALOGUES (donnees derivees du jeu installe, pratique
> etablie des chantiers cartes) — l'extracteur de rejeu livre reste offline-pur : il ne lit
> que le film et les catalogues.

## 0. CE QUI EST SU (verifie les 27/08, ne pas re-etablir)

- `map_quant_bounds.json` : Live Fire ABSENTE — les 2 films Oddball Live Fire (`60ae07c4`,
  `c88ec007`) sont INDECODABLES (lot O, O0.2). Lattice y est (le film `92f18088` decode).
- `map_objectives.json` : Lattice ABSENTE (0 socle `oddball_spawn` — reserve consignee au
  protocole D10) ; AUCUN site `assault_bomb` pour Origin, Curfew, Absolution, Rat's Nest,
  Urban Raid (les 4 seuls du catalogue : Isolation, Snowbound, The Pit, High Ground).
  Deux hypotheses NON departagees (CR lot A) : (H1) la variante `.mvar` extraite le 25/08 ne
  portait pas les objets d'Assaut ; (H2) hash de role NON RESOLU sur les cartes recentes
  (patron KOTH, `mapvar/objectives.go`).
- Land Grab : `ObjectiveTypeOf` le classe `zone` (`extract.go:135`) mais AUCUNE entree de
  `objective_roles.toml` ne le sert, hashs `landgrab_zone` presents dans les fichiers de
  carte sans role associe (§2.8 du plan obj-etat, incoherence nommee). Aucun film Land Grab
  (expire) : le cablage sert les matchs FUTURS. Decision produit : OUI, cabler (plan
  chantier §1.5).
- Outillage d'extraction existant (chantiers cartes) : `cmd/mapquant-build`,
  `cmd/mapobj-build`, `cmd/mapstruct-build`, decodeur `mapvar` ; point d'entree documente :
  `.ai/V7.5/cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md` et les en-tetes des CLIs. Le jeu
  est installe sur ce poste (chemins : voir les defauts/docs des CLIs — les LIRE, ne pas
  deviner).
- Gate A1 (identite bombe), a rejouer TEL QUEL apres reparation :
  `assaut_a1_identite_test.go` — UN SEUL mot, LE MEME sur >= 2 films, temoin = 0 autre
  candidat. Mots recurrents deja publies (premiers a confronter, sans en faire le gate) :
  `0x3FEE4FCF` (7/7 films), `0xE9E7FF79` (4). Publication A2 (si gate tenu) : patron du
  crane libre — famille `bomb` au manifeste `replay_labels.toml` (EN+FR), garde d'exclusion
  des socles etendue, calque `objectiveObjects` web, re-cuisson des TEMOINS seulement avec
  verification du CONTENU (recette du registre : racine temporaire a jonctions — config du
  worktree, data du principal).
- Schema 21 / contrat 39 intacts. L'ajout d'une famille au manifeste est de la DONNEE
  (verifier quand meme ; si un bump s'impose : STOP et arbitrage au CR).

## 1. PHASES

> Ordre strict C1 -> C2 -> C3 -> C4 -> C5. Items [x]/[~]/[!], commits `catalogues-c(<n>):`,
> jamais `git add -A`, pas de push, thought_log/REGISTRE jamais touches (textes au CR).
> Donnees via `LEVELUP_REPO_ROOT` (depot principal), DuckDB en lecture seule. Un seul
> build/test Go a la fois. AUCUNE re-mesure des campagnes D4-D10 (interdit — le lot O est
> clos) : ici on repare des catalogues et on rejoue les seuls gates A1 (+A2 si tenu) et la
> QUALIFICATION des films Live Fire.

### C1 — Bornes de quantification : Live Fire

- [ ] C1.1 Etablir POURQUOI Live Fire manque a `map_quant_bounds.json` (lire l'outil qui
      l'a produit et sa source ; Live Fire est une carte du jeu de base — l'absence a une
      cause, la nommer).
- [ ] C1.2 Produire les bornes de Live Fire avec l'outil existant, MEME METHODE que les
      cartes presentes (aucune borne bricolee a la main).
- [ ] C1.3 GATE C1 : les 2 films Oddball Live Fire DECODENT (un film par processus) et leur
      pont bipede se mesure (instrument de qualification du lot O rejoue TEL QUEL) ; publier
      film -> taux -> admis/exclu au critere 50 % inchange. Log fige
      `C1_livefire_qualification.log`. (Qualification SEULEMENT — le corpus Oddball passe de
      4 a N films consigne pour une reprise future ; aucune remesure D9/D10.)

### C2 — Catalogue d'objectifs : sites d'Assaut + oddball_spawn de Lattice

- [ ] C2.1 Departager H1/H2 sur pieces : ce que la variante `.mvar` des cartes du corpus
      contient reellement (Origin, Curfew, Absolution en priorite ; Rat's Nest et Urban
      Raid si le meme chemin les sert), et ce que le decodeur `mapvar` resout ou laisse en
      hash inconnu. Verdict ecrit avec fichiers/hashs a l'appui.
- [ ] C2.2 Ajouter les sites `assault_bomb` des cartes du corpus au catalogue par la MEME
      chaine que les autres roles (donnee, pas de code special-case) ; idem
      `oddball_spawn` de Lattice.
- [ ] C2.3 GATE C2 (ecrit ici) : pour chaque carte reparee, les sites sont DANS les bornes
      de la carte, leur COMPTE est publie (attendu : 1-2 sites d'armement par carte
      d'Assaut, 1-2 socles de crane pour Lattice), et un controle de coherence spatiale
      passe : en One Bomb/Neutral Bomb, chaque EXPLOSION datee par le score de mode (releves
      A0.3, commits du lot A) doit avoir eu de l'activite de joueurs a proximite du site
      dans les secondes qui precedent — accord >= 75 % des explosions a <= 10 m d'un site
      ajoute, temoin sites decales de 12 m <= 25 %. Log fige `C2_sites_controle.log`.
      Si le controle spatial rate : les entrees ne rentrent PAS au catalogue, `[!]` chiffre.

### C3 — Rejouer le gate A1 TEL QUEL ; publier A2 si et seulement s'il tient

- [ ] C3.1 `assaut_a1_identite_test.go` rejoue sans AUCUNE modification de seuil ni de
      logique (seul le catalogue a change). Log fige `C3_identite_bombe_rejeu.log`.
- [ ] C3.2 Si gate tenu (UN mot, >= 2 films, temoin 0) : publication des vies libres de la
      bombe au patron du crane (les 4 items A2.1-A2.4 du plan lot A, repris tels quels :
      manifeste famille `bomb` EN+FR, garde d'exclusion, calque web, re-cuisson temoins
      avec verification de CONTENU). Si un bump de schema s'avere requis : STOP, arbitrage.
- [ ] C3.3 Si gate rate : `[!]` avec chiffres, rien ne se publie (pas de 2e essai).

### C4 — Land Grab : cablage du role (ex-lot L)

- [ ] C4.1 Role `landgrab_zone` au decodeur `mapvar` + entree `objective_roles.toml`
      (match Land Grab), degradation par absence de donnee — patron des roles existants,
      zero special-case dans le code.
- [ ] C4.2 GATE C4 : tests unitaires du decodeur/roles verts ; sur une carte porteuse de
      hashs `landgrab_zone`, les formes sortent du catalogue (verification statique par
      test, pas de film requis) ; le commentaire d'incoherence du §2.8 (dans
      `service/replay_map_objectives.go`) est mis a jour dans le MEME commit.

### C5 — Cloture

- [ ] C5.1 Gates du lot : `go build ./...` exit 0, `go vet` propre sur packages touches,
      `go test` des packages touches ; si C3.2 a publie : contracttest + `tsc -b` (cache
      purge) + vitest `match-replay` + lint web 0 erreur.
- [ ] C5.2 Plan statue (0 case vide), logs figes commites, CR avec textes journal/registre.

## 2. COMPTE-RENDU ATTENDU

Cause de l'absence Live Fire ; verdict H1/H2 ; comptes de sites par carte + controle spatial
chiffre ; qualification des 2 films Live Fire ; verdict du rejeu A1 (LE mot ou echec
chiffre) ; si publication : familles, comptes de vies libres par temoin re-cuit, gates web ;
etat Land Grab ; commits ; textes thought_log + registre ; decouvertes.

## 5. DECOUVERTES (a consigner, ne pas traiter)

- (vide a l'ouverture)
