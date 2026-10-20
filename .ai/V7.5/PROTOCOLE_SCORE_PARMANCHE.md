# Protocole — SCORE PAR JOUEUR par manche (dernier calque faux en multi-manche)

> Commite AVANT le refactor (plan-execution). Fige l'approche, les invariants et les
> temoins avant toute mesure. Branche `wt/score-parmanche` (base aa1245fa0).

## Defaut a corriger (contexte etabli, non re-mesure)

`scoreTimeline.players` (`score_timeline.go` `buildPlayerScores`) apparie le slot d'entite
au joueur par le pont des TOTAUX (`objectiveevents.SlotIdentityFrom(recs, in.Lines)` : triplet
kills/morts/assists du match). En multi-manche le slot d'entite est REATTRIBUE d'une manche a
l'autre ET le compteur de morts repart de zero par manche : le pont plat ne voit que la manche
0, un seul joueur est apparie (`scoreTimeline.players=1` mesure sur d9781168, 3 manches). Les
calques freres (drapeau CTF, couronne VIP, porteur crane, objectifs) ont deja migre vers
l'identite PAR MANCHE (`ResolveRoundIdentity` / `AtRound`, par les instants de mort). Le score
par joueur est le seul reste.

## Approche retenue — FORK mono/multi (preserve la neutralite mono-manche a l'octet)

Contrairement aux objectifs (evenements PONCTUELS), le score personnel est une COURBE CONTINUE
par slot. La courbe existe DEJA sous deux formes dans `objectiveevents` :
`SeriesByRound` (map slot -> manche -> points, valeurs propres a la manche) et
`SeriesTotal` (map slot -> points cumules sur le match). La forme PAR MANCHE fournit
gratuitement le decoupage (a).

- **Mono-manche (`len(RealRounds) <= 1`)** : chemin PLAT INCHANGE, mot pour mot — pont des
  totaux `SlotIdentityFrom`, `SeriesByRound`/`SeriesTotal` par slot. Le slot n'est pas
  reattribue, l'ancien pont est exact. **Neutralite BYTE-EXACTE** : aucun octet de
  `scoreTimeline.players` ne bouge, aucun temoin/golden ne change. (Mieux que la migration
  objectifs, qui bascule totaux->morts meme en mono-manche.)
- **Multi-manche (`> 1` manche reelle)** : identite PAR MANCHE
  (`ResolveRoundIdentity(recs, deaths)` par les instants de mort, comme les freres).
  - (a) DECOUPAGE : `SeriesByRound(recs, comp, false)` donne deja les segments par slot/manche.
  - (b) RATTACHEMENT : chaque segment (slot, manche) va au joueur `AtRound(manche, slot)`.
  - (c) FUSION : les segments d'un meme xuid (occupant des slots differents selon les manches)
    sont regroupes -> UNE entree `PlayerScore` par xuid. `Rounds` = les segments par manche ;
    `Total` = cumul recompose dans l'ordre des manches (offset par le total des precedentes),
    reproduisant exactement `cumulateRounds` mais par xuid. Les 4 composants joueur
    (score perso, kills, morts, assists) sont NON stricts : le segment par manche de
    `SeriesByRound` est identique au `kept` de `cumulateRounds` -> le cumul par xuid coincide
    avec `SeriesTotal` quand un joueur garde son slot, et fusionne correctement sinon.

Cablage : `opt.Deaths` (deja scanne dans `BuildFromPositions`, l.346) est passe a
`attachScoreTimeline` -> `buildScoreTimeline`, exactement comme `attachVipCrown` /
`attachFlagCarries` lisent `opt.Deaths`. Aucun changement a `objectiveevents` (API exportee
suffit). Perimetre code = `score_timeline.go` + la ligne de cablage `build.go:448`.

## Invariants (a ne pas casser)

1. **`scoreTimeline.teams` INCHANGE** : `SlotIdentityFrom(recs, in.Lines)` reste la source de
   la preuve (b) d'identite des camps (`resolveTeamIdentity`). Non touche.
2. **Mono-manche `scoreTimeline.players` INCHANGE** (byte-exact, chemin plat conserve).
3. **Forme du document / contrat INCHANGE** : `PlayerScore` garde ses champs
   (`xuid`, `score`, `kills`, `deaths`, `assists`, chacun `ScoreSeries{rounds,total}`).
   PAS de bump `SchemaVersion` (reste 23). Si la forme devait changer -> STOP, signaler.

## Temoins ecrits AVANT la mesure (tests)

- REGRESSION mono-manche : les tests existants de `score_timeline_test.go` (chemin plat)
  passent inchanges (assertions identiques) -> preuve que le plat n'a pas bouge.
- MULTI-MANCHE slot reattribue : fixture a 2 manches, slot reattribue joueur A (manche 0) ->
  joueur B (manches 1+). Le score de B en manche 1 lui est attribue ;
  CONTRE-EPREUVE : le pont plat le donnait a A (ou ne publiait pas B).
- FUSION xuid multi-slots : un joueur occupant slot S0 en manche 0 puis slot S1 en manche 1
  rend UNE seule entree `PlayerScore`, courbe recomposee dans l'ordre du temps.

## Gates (verts exiges)

`go build ./...`, `go vet ./...`, `go test` sur `internal/analysis/replay` +
`internal/analysis/objectiveevents` + `internal/replaybuild` (contrat/forme inchanges).

## Honnetete

Le lot LIVRE un calque -> ADVERSARIAL-REVIEW requise avant merge (fichiers a risque :
`score_timeline.go`, `build.go`). Si le decoupage/fusion s'avere plus complexe que prevu
(ex. score personnel accumule sur tout le match et non par manche) -> STOP, documenter au CR.
