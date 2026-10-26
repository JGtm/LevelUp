# PLAN — Objectifs PAR MANCHE + nettoyage porteur (lot wt/obj-parmanche)

> Protocole COMMITE AVANT tout refactor (contrat plan-execution). Base f2d2a87a8.
> Perimetre FERME : correctness du calque Objectives multi-manche + 3 P2 de revue porteur.

## Constat d'architecture (verifie sur pieces)

- `replaybuild/matchfacts.go:identifiedEvents` resout l'identite slot->xuid par
  `objectiveevents.SlotIdentityFrom(recs, lines)` (pont par TOTAUX, exige les lignes de
  match). En MULTI-MANCHE le slot d'entite est reattribue -> attribution fausse apres la
  bascule (meme defaut que couronne/drapeau, deja corriges).
- `ResolveRoundIdentity(recs, deaths)` (pont par MANCHE) exige le FIL DES MORTS
  (`[]DeathInstant`). `port.MatchFacts` ne le porte PAS ; il n'existe que dans
  `BuildFromFilm` (`opt.Deaths`). Donc la resolution DOIT descendre dans le paquet
  `analysis/replay`, EXACTEMENT comme `vip_crown.go` / `build_objectives_live.go` /
  `skull_carries.go` (qui font `ResolveRoundIdentity(in.Records, deathInstantsOf(opt.Deaths))`).
- `doc.Objectives` (actions posees) est aussi consomme par le calque des zones
  (`build_zones.go`) : il reste produit par `attachObjectiveActions`, forme inchangee.
- Forme du document (`ObjectiveAction` t/xuid/stat/timeMs) INCHANGEE -> AUCUN bump de schema.

## Etape 1 — MIGRATION Objectives par manche (touche du code LIVRE) — CLOSE

DECISION D'ARCHITECTURE (consignee) : `ResolveRoundIdentity` exige le fil des morts. Le
changer de type `Options.Objectives` (approche « resoudre dans replay ») reshaperait
DEUX consommateurs de RECHERCHE hors perimetre (`cmd/zone-attribution`, temoin zones p2b)
qui INJECTENT des `[]IdentifiedEvent` par un pont d'identite DELIBERE (SlotIdentityResolved).
Choix retenu : garder le type public `Options.Objectives []IdentifiedEvent` STABLE et
resoudre par manche EN PLACE dans `matchfacts.go` (le fichier du perimetre), en relisant le
fil des morts (`replay.ScanFilmDeaths`, un chunk highlight, borne). Zero outil de recherche
touche ; forme du document et contrat INCHANGES.

- [x] 1a. `objectiveevents` : `IdentifyNamedEventsByRound(evs, RoundIdentity)` ajoute (tri
  factorise `sortIdentifiedEvents`). Tests : mono-manche == pont plat ; multi-manche manche 1
  au bon joueur, DIFFERE du pont plat (`twoRoundReassignedFixture`). VERTS.
- [x] 1b. `replaybuild/matchfacts.go` : `identifiedEvents` resout par manche
  (`NamedEventsFrom` + `ScanFilmDeaths` + `ResolveRoundIdentity` + `IdentifyNamedEventsByRound`) ;
  coeur pur `identifyRoundEvents` + `deathInstantsOf`. Garde de mode court-circuite l'I/O.
- [~] 1c. (fusionne dans 1b — voir decision : pas de changement de type Options.)
- [x] 1d. Tests `matchfacts_test.go` reecrits : garde de mode sans I/O, multi-manche
  (capture manche 1 -> joueur B, contre-epreuve pont plat -> A), conversion. VERTS.
- [x] 1e. REGRESSION mono-manche : neutralite PAR CONSTRUCTION prouvee (RoundIdentity.At une
  manche = pont plat par morts ; `TestSlotIdentityByRoundMonoRoundNeutral`,
  `TestIdentifyNamedEventsByRoundMonoNeutral` VERTS). La bascule totaux->morts repose sur
  l'accord phase-0 (8/8, 0 desaccord) — MEME base que drapeau/VIP/crane deja livres. Golden
  film reel = corpus-gated, PAS de cache local : non joue ici, statue au CR.

## Etape 2 — P2 #1 : `skullCarrySecondsByXUID` test-only — CLOSE

- [x] `skullCarrySecondsByXUID` deplacee de `skull_carries.go` (PROD) vers
  `skull_carries_witness_test.go` (seul appelant, meme paquet `replay`). Retiree du binaire
  de prod, zero changement de comportement. Build/vet/tests VERTS.

## Etape 3 — P2 #2 : doc `SkullTicksComponent` — CLOSE

- [x] `objectiveevents/score.go` : reference morte `[incrementInstants]` (identifiant de
  `cmd/oddball-terrain` hors perimetre publie) remplacee par une mention en prose de la vraie
  fonction de dedup portee en prod : `skullTickInstants` (paquet replay). Doc seulement.

## Etape 4 — P2 #3 : test synthetique porteur — CLOSE

- [x] Peu couteux (reutilise `skullFixture`) : `TestSkullCarrySecondsByXUID` ajoute a
  `skull_carries_test.go` — gate en CI la grandeur du gate oracle (somme durees/xuid,
  porteur principal = A) que le temoin sur film reel skip. VERT.

## Gates (chaque etape)

`go build ./...`, `go vet ...`, `go test` des paquets touches (replaybuild, replay,
objectiveevents) + contracttest (le contrat NE bouge PAS). Pas de web (contrat stable). TOUS VERTS.

## Decouvertes (non traitees — perimetre)

1. `deathInstantsOf` existe desormais en 2 copies quasi identiques (`replaybuild/matchfacts.go`
   et `replay/build_objectives_live.go` : `[]Death` -> `[]DeathInstant`). Dans la regle des
   <= 2 copies ; candidat a centralisation a la 3e occurrence.
2. Chemin production : le chunk highlight des morts est desormais relu 2 fois par artefact
   (matchfacts `ScanFilmDeaths` + le scan interne de `BuildFromFilm`). Cout borne (un chunk),
   mais duplication. `BuildFromFilm` ecrase tout `opt.Deaths` fourni -> pas de partage trivial.
3. `cmd/zone-attribution` et le temoin zones p2b INJECTENT encore des objectifs identifies par
   l'ANCIEN pont (`SlotIdentityResolved`/`SlotIdentity`, par totaux) via `Options.Objectives` —
   desormais incoherent avec le pont PAR MANCHE de la prod. Non touches (type public stable,
   outils de recherche hors perimetre) ; a aligner sur le chemin prod dans un lot dedie.
