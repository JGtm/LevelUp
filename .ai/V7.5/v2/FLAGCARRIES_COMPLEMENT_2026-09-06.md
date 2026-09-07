# `flagCarries` — l'identite des ports de drapeau completee (2026-09-06)

Branche `feat/v2-flagcarries`, worktree `LevelUp-wt-v2-flagcarries`, base `5e7704428` (= `feat/v75`,
SchemaVersion 41). Decision utilisateur du 2026-09-06 ; entree 543 du registre des reports fermee.

**VERDICT : le calque du drapeau publie enfin les portages de ceux qui meurent peu.** Sur
`c0a82e88` (le fixture E2E) il passe de 0 a 1 portage ; sur `e94163af` de 16 a 33, `noBridge` de
17 a 0 ; sur `51101d1d` de 10 a 11. Un film MULTI-MANCHE est rendu identique, au numero de schema
pres. Aucun portage deja publie n'a change de joueur, aucun n'a disparu.

---

## 1. Ce qui etait plafonne, et pourquoi

Le calque nommait son porteur par le pont d'identite PAR MORTS
(`objectiveevents.ResolveRoundIdentity`), qui exige `deathInstantMin` = **3** instants de mort
coincidents pour attribuer un slot d'entite statborg a un xuid. **Un joueur qui meurt moins de
trois fois lui echappe par construction** — et ce sont, par definition, ceux qui portent le
drapeau. Leurs prises etaient comptees `coverage.flagCarries.noBridge` et **aucun intervalle
n'etait publie** : le drapeau restait dessine a sa base pendant qu'un joueur le portait.

Le pont par TRIPLET (totaux frags/morts/assistances apparies aux lignes de match) les nomme, mais
il exige la base — que `analysis/replay` n'ouvre pas. C'est la frontiere qui rend le calque
« publiable hors ligne », et le §9.8 de `INSTRUCTION_CTF_DRAPEAUX.md` l'avait portee au registre
comme une decision produit.

## 2. La mesure AVANT (HEAD `5e7704428`, schema 41)

Cuisson de production (`cmd/replay-build --map ... --facts ...`), un film par processus, racine de
travail a jonctions, `data/cache/replays` reel. Faits exportes par `levelup replay-facts-export`.

| match | variante | manches | prises | portages | `noBridge` | `noTrack` |
|---|---|---|---|---|---|---|
| `c0a82e88` | Husky Raid:CTF | 1 | 3 | **0** | **3** | 0 |
| `e94163af` | CTF:Arena Neutral Flag | 1 | 33 | 16 | **17** | 0 |
| `51101d1d` | CTF:Arena Neutral Flag | 1 | 12 | 10 | **1** | 1 |
| `fb1a1a72` | CTF:Arena | **3** | 46 | **0** | **46** | 0 |

Slots des prises sans pont (journal ajoute par ce lot, cf. §4) — `c0a82e88` : `[12, 12]` apres
correctif, donc les trois prises se repartissent en **2 sur le slot 12 agrege** (compteurs de film
5/0/60 assistances : ils ne ressemblent a aucune ligne de match, il n'est nommable par aucun des
deux ponts) et **1 sur le slot 22** (SweatyYeti75, 7 frags / 2 morts / 1 assistance). C'est
exactement le releve du §9.8 de l'instruction CTF, reproduit au HEAD.

`51101d1d` est bien un CTF (Neutral Flag) ; son artefact au parc date d'AVANT l'existence de
`flagCarries` (schema anterieur au 14), d'ou son absence du recensement CTF du parc.

## 3. La conception

**Ce qui a ete demande, et ce qui a ete fait.** Le mandat disait « completer l'identite des ports
DANS `replaybuild`, apres la cuisson, exactement comme les actions », et prevoyait au besoin de
faire porter leur SLOT aux intervalles `noBridge`. **La completion se fait bien dans `replaybuild`
et `analysis/replay` ne voit toujours aucun fait de match** ; en revanche, un post-traitement du
document APRES `BuildFromFilm` est impossible, et la mesure le dit :

1. **Un intervalle `noBridge` n'existe pas dans le document.** `buildFlagCarries`
   (`flag_carries.go`) ecarte les prises sans xuid AVANT le bornage : elles ne deviennent jamais
   un `flagCarryRaw`. Il n'y a donc aucun intervalle a renommer.
2. **Les faire survivre sans xuid ne donnerait rien de publiable.** La FERMETURE d'un portage lit
   `deaths[xuid]`, et sa GEOMETRIE (`attachFlagCarryPositions`) vient de la piste PUBLIEE du
   porteur : sans xuid, pas de position, et le portage serait rejete sous `NoTrack`.
3. **`flagCarries` n'est pas une liste de portages, c'est la vie de chaque DRAPEAU.** Un
   `FlagCarry` porte des `Spans` CONTIGUS (`home` / `dropped` / `carried` / `carried_open`) :
   ajouter un portage apres coup ne renomme pas un intervalle, cela redecoupe la frise entiere du
   drapeau (`assembleFlagLives`). Un renommage post-cuisson laisserait un `dropped` la ou il faut
   un `carried`.

**La forme retenue** — celle que l'entree 543 du registre proposait, en plus stricte :

- `replay.FlagInput` gagne un champ **`Identity objectiveevents.RoundIdentity`** : un pont **DEJA
  RESOLU**, fourni par l'appelant. Ce n'est pas une `PlayerLine` : `analysis/replay` recoit une
  table `slot -> xuid`, jamais un fait de match. La regle qui la complete, et les faits qu'elle
  consomme, restent chez `replaybuild`.
- `attachFlagCarries` choisit par `flagIdentityOf(in, opt)` : le pont de l'appelant s'il est
  **`Resolved()`**, sinon la resolution locale par les seuls instants de mort. Champ a zero =
  comportement d'avant, a l'octet pres — le CLI hors ligne et l'ouvrier sans faits sont intacts.
- `objectiveevents.RoundIdentity.Resolved()` distingue « personne n'a resolu » (valeur zero) de
  « la resolution n'a nomme personne » — un compte de noms confondrait les deux.
- `replaybuild` resout le pont **une seule fois par cuisson**, dans un memo paresseux
  `pontParManche`, et le donne AUX DEUX calques qui en vivent (actions d'objectif ET drapeau). Il
  etait resolu deux fois auparavant, sur les memes enregistrements et le meme fil des morts.
- Le pont n'est demande **que sur un film de CTF**, et la garde est celle du calque : les trois
  signaux du FILM (`FlagFilmSignalsFrom(...).IsFlagFilm()`), jamais le nom de variante. C'est la
  protection du 2026-08-18 (deroulage a 19-22 Go sur un film d'une autre grammaire), conservee.

Les trois gardes de `CompletedByLines` sont inchangees et prouvees a leur source : mono-manche
seulement, completer sans jamais contredire, aucun xuid deux fois.

## 4. La mesure APRES

| match | manches | prises | portages | `noBridge` | marqueur confirme / observe |
|---|---|---|---|---|---|
| `c0a82e88` | 1 | 3 | **1** (0) | **2** (3) | **1 / 1** (0 / 0) |
| `e94163af` | 1 | 33 | **33** (16) | **0** (17) | **6 / 7** (5 / 5) |
| `51101d1d` | 1 | 12 | **11** (10) | **0** (1) | 4 / 4 (4 / 4) |
| `fb1a1a72` | 3 | 46 | 0 (0) | 46 (46) | 0 / 0 |

(entre parentheses : la valeur AVANT)

**L'ORACLE, ET IL EST INDEPENDANT.** Le porteur publie sur `c0a82e88` est `2535463878425995`, sur
l'intervalle de frames **[640, 706]**. Le calque des ACTIONS — autre chaine, autre grammaire —
date sur le MEME match un `flag_steals` a **641** et un `flag_captures` a **706**, au MEME xuid.
Le portage s'ouvre au vol et se ferme a la capture. Le **marqueur de portage des images-cles**,
troisieme chaine (une suite de bits du record de bipede), le confirme 1/1.

Sur `e94163af`, le joueur nouvellement ponte (`2535461839170315`, **1 mort**) recoit **17
portages** — et le calque des actions lui credite exactement **17 `flag_grabs`**. Deux chaines
disjointes, meme compte.

**Aucun portage deplace, aucun perdu.** Comptes par joueur, avant contre apres :

```
c0a82e88   2535463878425995  0 -> 1
e94163af   2533274820399233  3 -> 3   2533274823110022  2 -> 2   2533274858283686  8 -> 8
           2533274894252091  1 -> 1   2535469190789936  2 -> 2   2535461839170315  0 -> 17
51101d1d   2535429923417275  3 -> 3   2535450220111457  3 -> 3   2535462227353352  1 -> 1
           2537966040008427  3 -> 3   2535415792596376  0 -> 1
```

**Une baisse, expliquee et voulue** : `coverage.flagCarries.homeByObject` tombe (4 -> 0 sur
`e94163af`, 1 -> 0 sur `51101d1d`) et un etat `home` disparait (8 -> 7). La RENTREE PAR L'OBJET
n'agit que sur un drapeau **AU SOL** (`applyFlagHomecoming` : `st.state[f] != FlagStateDropped`
sort) : ces drapeaux-la etaient « au sol » uniquement parce que le portage qui les tenait n'etait
pas publie. Le lot corrige la cause, la consequence suit.

**Un gain de bord** : `flagReturnZone` APPARAIT sur `c0a82e88`. La regle de retour du mode n'est
publiee que s'il y a au moins un drapeau a entourer (`attachFlagReturnZone`) ; un match sans
aucun portage publie n'en avait pas.

**Cuisson au HEAD contre HEAD^, `cmd/replay-diff`** (559 a 633 mesures par paire) :
`changements = 0` partout — aucune mesure n'a change de VALEUR, seules des mesures apparaissent ou
montent. `fb1a1a72` (3 manches) : **1 ecart sur 573 mesures**, `schemaVersion 41 -> 42`.

## 5. Schema 42

Le CONTENU de `flagCarries` change, aucune cle ne bouge — la meme situation qu'aux montees **v14**
et **v15** du meme calque, et qu'a la **v40** (le pont complete des actions). La regle du depot
s'applique : un artefact 41 est ampute de portages sans que sa forme le dise, et
`backfill-replay` saute un artefact qui porte la version courante. Chronique ecrite dans
`document.go` et dans `structure_test.go`.

**Le contrat servi ne bouge pas** : aucun champ n'est ajoute au document (le champ ajoute est une
ENTREE de construction, `FlagInput.Identity`). `openapi.yaml` inchange ;
`apps/web/src/lib/api/generated.ts` regenere depuis l'`openapi.yaml` du worktree et **identique
octet pour octet** ; `TestReplayDocumentFieldCountIsFrozen` et la parite `replayview` inchanges.

**Golden d'assemblage** : regenere, **une seule ligne** de diff (`schema 41` -> `schema 42`),
verifiee avant regeneration (606 lignes des deux cotes, premier ecart ligne 9).

## 6. Journal ajoute

`logFlagOpeningsWithoutBridge` NOMME desormais les slots des prises sans pont
(`rejeu : prises de drapeau sans pont d'identite slots=[12 12] sansPont=2 prises=3`). Le compte
seul ne se diagnostique pas : il ne dit pas si les prises perdues appartiennent a un slot ou a
cinq, ni auxquels — la question exacte qu'il faut pouvoir poser a un artefact du parc.

## 7. Tests

**Unitaires, `internal/replaybuild/flagidentity_test.go`** (fixture mono-manche synthetique : slot
10 nommable par les morts, slot 12 nommable par le seul triplet, slot 14 agrege) :

- `TestWithFlagIdentityPoseLePontCompleteSurUnFilmCTF` — le slot 12 est nomme, le slot 10 garde son
  nom, le slot 14 reste muet ; **mutation dans le test** : sans lignes de match, le slot 12
  redevient anonyme.
- `TestWithFlagIdentityNeResoutRienHorsCTF` — sans burst ou sans evenement nomme, aucun pont n'est
  resolu (la garde de cout du 2026-08-18).
- `TestPontParMancheNeResoutQuUneFois` — la source est VIDEE apres le premier appel ; le second
  rend la meme table.

**Unitaires, `internal/analysis/replay/flag_carries_identity_test.go`** :

- `TestFlagIdentityOfResoutLocalementSansPontFourni` — l'artefact hors ligne reste entier.
- `TestFlagIdentityOfPrefereLePontDeLAppelant` — le pont fourni l'emporte.
- `TestFlagIdentityOfRespecteUnPontMuet` — un pont fourni VIDE est une reponse, pas une invitation
  a resoudre soi-meme (c'est ce qui justifie `Resolved()` plutot qu'un compte de noms).
- `TestFlagCarriesPontFourniPublieLePortage` — le meme film, deux ponts : celui par morts laisse la
  prise `noBridge` (l'etat du schema 41), celui de l'appelant publie l'intervalle au bon joueur.

**E2E, `internal/api/wire/build_queue_worker_objectifs_integration_test.go`** (`assertPortsDeDrapeau`,
cuisson reelle du fixture par l'ouvrier) : couverture 1 portage / 2 sans pont / 3 prises,
invariant de couverture tenu, **exactement un** intervalle porte, son xuid = le porteur de la
feuille, et sa fenetre s'ouvre au `flag_steals` et se ferme AU `flag_captures` du calque des
actions.

**MUTATIONS JOUEES, ROUGE PUIS VERT** (le complement debranche : `in.Identity = pont.identite()`
neutralise) :

```
--- FAIL: TestWithFlagIdentityPoseLePontCompleteSurUnFilmCTF
    aucun pont pose sur un film de CTF : le calque retomberait sur les seules morts
--- FAIL: TestOuvrierReel_ConstruitEtLivre
    couverture du drapeau : 0 portage(s), 3 sans pont, 3 prises — attendu 1 / 2 / 3
    0 intervalle(s) porte(s) publie(s), attendu 1
```

et, cote calque (`flagIdentityOf` force a resoudre localement) :

```
--- FAIL: TestFlagIdentityOfPrefereLePontDeLAppelant
--- FAIL: TestFlagIdentityOfRespecteUnPontMuet
```

## 8. Temoins et gates

| gate | resultat |
|---|---|
| `go test -count=1 ./internal/replaybuild/... ./internal/analysis/replay/... ./internal/analysis/objectiveevents/... ./internal/archlint/... ./contracttest/...` | **ok** (6 paquets) |
| `go test -tags=integration -p 1 -count=1 ./internal/api/wire/...` | **ok** 23,7 s |
| `go build ./...` | **ok** |
| `golangci-lint run --new-from-merge-base=origin/main ./...` | **0 issues** |
| `TestGoldenMiniBobine` (filmdec) | **ok** |
| `TestEquivalenceMiniFilm` + tous les `Golden*` (replay) | **ok** |
| `generated.ts` regenere puis compare | **identique** |
| parc principal (`data/cache/replays`, `data/titles`, `data/cache/film_chunks`) | **intact** (`git status data/` vide des deux cotes, 111 artefacts, plus recent du 4 septembre) |

## 9. Impact sur le parc

Releve des 27 artefacts CTF du parc local (lecture seule de `coverage.flagCarries` et
`coverage.score.rounds`) :

- **13 matchs** portent au moins une prise sans pont, **156 prises** au total ;
- **11 d'entre eux sont MONO-MANCHE** et portent **98** de ces prises : ce sont les candidats du
  correctif (`008e1bba` 15, `e94163af` 17, `3372e7eb` 13, `4ecdf3e7` 11, `a17e61a2` 11,
  `cf040013` 10, `846044ba` 8, `bf5ced1b` 5, `b8d1fe0c` 4, `c0a82e88` 3, `0f9550e5` 1) ;
- **2 sont MULTI-MANCHE** (`fb1a1a72` 3 manches / 46 prises, `7fce3219` 2 manches / 12 prises) :
  la garde s'abstient, a dessein — le slot y est reattribue d'une manche a l'autre.

Le taux de recuperation par match depend du nombre de slots AGREGES parmi les non-pontes : mesure,
il va de **1/3** (`c0a82e88`, deux prises sur un slot agrege) a **17/17** (`e94163af`). Les trois
films mono-manche mesures gagnent tous des portages nommes. `51101d1d` s'y ajoute, hors
recensement (son artefact du parc precede l'existence du calque).

Propagation : passe `backfill-replay` de la release v7.5.0, qui re-cuit desormais tout artefact
**< 42**.

## 10. Decouvertes, notees et NON traitees

1. **Les CTF multi-manche restent muets, et le trou y est bien plus grand.** `fb1a1a72`
   (3 manches) : 46 prises, 0 portage, et le calque des ACTIONS n'identifie que **3 actions sur
   936 nommees**. Le pont par manche y nomme quasiment personne, et le triplet ne peut pas venir
   a la rescousse (ses compteurs sont des totaux de match). C'est une question distincte de
   celle-ci : elle demande un pont par manche qui tienne, pas une completion.
2. **Le slot AGREGE** (`c0a82e88` slot 12 : 5 frags / 0 mort / 60 assistances au film contre
   5/0/0 a la feuille) reste hors de portee des deux ponts. Son compteur d'assistances lu a 60
   est la cause du refus du triplet ; personne n'a instruit pourquoi ce compteur derive.
3. Les calques **VIP** et **CRANE** resolvent leur identite par le meme pont par morts et
   subissent donc le meme plafond. Ils n'ont pas ete touches (hors perimetre).
