# Protocole — re-cuisson Oddball d9781168 (schema 23, porteur du crane)

> Ecrit et commite AVANT le fix. Branche `wt/oddball-recuisson`, base 53db169ce.
> Cible : re-cuire `data/cache/replays/halo_infinite/d9781168.json` du depot principal
> (via `LEVELUP_REPO_ROOT`) avec le PORTEUR DU CRANE visible, le score a equipes resolues
> et les dots de manche.

## Diagnostic (etabli sur pieces, avant tout fix)

- `match_registry` de d9781168 : `map_name = "Dredge"`, `map_id = e4bb06db-...`,
  `game_variant_name = "Oddball:Arena"`, 8 joueurs / 2 equipes, scores 191 / 196.
- `map_quant_bounds.json` (`data/titles/halo_infinite/reference/`) PORTE la cle `"dredge"`.
  `NormalizeMapName("Dredge") == "dredge"` => `Lookup` REUSSIT.
- CAUSE de « carte hors catalogue ([]) » : la forme DIRECTE `backfill-replay --one <id>`
  (documentee comme licite a la main) n'appelle PAS `registreParShort` — la resolution de
  carte vit UNIQUEMENT dans le PARENT, qui passe les candidats a l'enfant via `--map-name`.
  Tapee a la main sans `--map-name`, `o.mapNames == []` => `ResolveMapEntry([])` =>
  `ErrMapNotInCatalog (candidats: [])`. Le `[]` du message EST la liste vide, PAS une
  resolution DB qui rend vide. Ce n'est donc PAS un defaut de donnee (« Dredge » est deja au
  registre ET au catalogue) : c'est un TROU de code dans l'auto-suffisance de l'enfant.
- CONSEQUENCE sur l'artefact servi (schema 23, issu du repli `replay-build --map Dredge`
  SANS facts) : `objectiveObjects` present (crane libre — ne depend pas des facts),
  mais `skullCarries` ABSENT et `scoreTimeline.players` ABSENT. Le porteur exige
  `facts.GameVariantName` (garde de mode `isSkullVariant`) ET le roster ; le chemin
  `replay-build --map` ne charge aucun fait. D'ou : crane invisible.

## Fix (minimal, cible, title-agnostic)

Faire que la forme `--one` resolve elle-meme les identites de carte depuis le registre quand
`--map-name` est absent, par la MEME logique que le parent (`map_id` -> asset EN + `map_name`
brut), via PathResolver. La masse (parent -> enfant) passe toujours `--map-name` : la
resolution registre ne se declenche QUE pour le `--one` tape a la main (aucun surcout de masse).
Centraliser l'ordre des candidats dans un seul helper (`candidatsCarte`) partage parent/enfant
(regle ≤2 copies).

## Etapes

1. [x] Fix code : `candidatsCarte` partage ; `mapNamesForOne` (RO relache avant decodage) ;
       branchement dans `runBackfillReplayUn` (explicit `--map-name` gagne, sinon registre).
       Commit 4d1142243.
2. [x] Gates : `go build ./...` (0), `go vet ./cmd/levelup/... ./internal/replaybuild/...` (0),
       `go test ./cmd/levelup/` (ok, dont TestCandidatsCarte). Pre-commit hook vert.
3. [x] Re-cuit d9781168 par `backfill-replay --one <ID COMPLET> --force`, LEVELUP_REPO_ROOT =
       principal. Module fo06_deepsea, 160 tracks, resolu via candidat brut « Dredge »
       (metadata tenue RW par le serveur -> asset EN degrade, sans effet : « Dredge » suffit).
4. [x] Verdict sur pieces (extraction JSON) :
       - schemaVersion 23 [x]
       - skullCarries = 36 portages, plusieurs xuids (porteur resolu PAR MANCHE) [x]
       - objectiveObjects = 47 vies (crane libre) [x]
       - scoreTimeline : 2 EQUIPES resolues (teamId 0/1, methode « a » = score final, la plus
         forte ; totaux 191/196 == registre), 3 manches par equipe, cible non publiee
         (regulation Oddball absente) [x]
       - scoreTimeline.players = 1 [!] — voir Decouvertes : limitation PRE-EXISTANTE du pont
         par TOTAUX du calque de score, hors perimetre du fix carte.
5. [x] Investigation « 2 joueurs / respawn 20 s » : verdict (a) mecanique de fin de manche —
       voir section dediee. Aucun fix (comportement fidele + trou de nommage pre-existant).
6. [x] Impact masse : 26 matchs Oddball au registre, 9 cartes distinctes TOUTES au catalogue ->
       0 bloque par bornes manquantes. Le bug de carte ne touchait QUE `--one` a la main ; la
       masse (parent) resout deja. Propagation du crane a tous les Oddball = `--only-existing`.

## Verdict « 2 joueurs / respawn 20 s » (item 4, SANS fix)

Mesure sur l'artefact (frameCount 7051, 100 ms/frame = 705 s ; bornes de manche a 206 / 417 / 698 s) :
- Histogramme du nombre de tracks actives par frame : la grande majorite a 4-8 joueurs. Seuls
  DEUX segments tombent a <=2 actifs : 213-214 s (transition manche 0->1) et 700-705 s (fin de
  match). Le champ n'est JAMAIS durablement a 2 en cours de manche.
- Les « respawns 20-60 s » se CONCENTRENT aux bornes : les morts juste avant la fin d'une manche
  ne reapparaissent qu'au DEBUT de la manche suivante (respawns groupes a ~2144 et ~4256).
- VERDICT = (a) vraie mecanique de fin de manche (respawn au reset de manche), pas (c). Les
  bornes de manche sont JUSTES (totaux == registre, respawns alignes dessus). Le segment de fin
  (5,5 s) est une extinction de fin de partie normale (film a 705 s vs 716 s registre, ecart =
  decalage de chargement via originMs), pas une troncature de manche (pas (b)).
- NUANCE consignee : 18 tracks sans xuid (vies non nommees par le fil des morts) gonflent
  certains ecarts PAR JOUEUR (ex. 158 s) — trou de NOMMAGE, pas absence reelle ; l'histogramme
  toutes-tracks confirme le plateau a 4-8.

## Decouvertes (NON corrigees — hors perimetre)

- **`scoreTimeline.players` par TOTAUX en multi-manches.** `buildPlayerScores` ->
  `SlotIdentityFrom(recs, lines)` apparie les slots d'entite aux joueurs par le triplet
  (kills, morts, assistances) des TOTAUX de match. En Oddball 3 manches, le slot d'entite est
  reattribue et ses compteurs repartent de zero par manche : les totaux ne matchent plus -> 1
  seul joueur apparie. Le calque d'OBJECTIFS a deja migre vers le pont PAR MANCHE (instants de
  mort, `ResolveRoundIdentity`) ; le calque de SCORE ne l'a pas fait. C'est la meme dette que la
  suite P2 mentionnee au commit de base 53db169ce (« score encore par totaux »). Le PORTEUR DU
  CRANE, lui, utilise deja le pont par manche -> 36 portages nommes, plusieurs joueurs.
- **Trou de nommage des vies.** 18 tracks sur 160 sans xuid (fil des morts incomplet en fin de
  manche / survivants), ce qui gonfle les ecarts de respawn par joueur.
- **Caveat masse + serveur.** Un backfill de masse LANCE PENDANT que le serveur tient metadata en
  RW degrade la resolution asset EN : sans effet pour les cartes a map_name propre (tous les
  Oddball), mais un film dont le registre ne porte qu'un UUID brut echouerait « carte hors
  catalogue » faute d'asset EN. Pre-existant, non traite ici.

## Invariants

- DBs en LECTURE SEULE (serveur tient RW) : `OpenReadForQuery` / `diag_q`, jamais RW.
- Seul l'artefact d9781168 du principal est reecrit. Pas de push. thought_log/REGISTRE = CR.
