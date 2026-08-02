# PLAN DE BRANCHEMENT — killsource en production

> Ecrit le 2026-07-31. Le decodeur, les deux tables et leurs persisters sont ECRITS, TESTES et
> POUSSES, **sans aucun appelant**. Ce document dit comment on les branche.
>
> Contrat d execution : skill `plan-execution`. Ordre strict, une etape a la fois, gate verifie
> avant de passer a la suivante, statut ecrit pour chaque item a la cloture.

## CE QUI EXISTE — ETAT REEL AU 2026-08-01 (branche `feat/replay2d-prod`)

> ⚠ **Le tableau d origine de cette section decrivait la branche d ARCHIVE**
> (`feat/filmdec-killweapon`), pas la branche vivante. Corrige par le master plan §1/§2, puis
> par l execution : les briques ont ete **REECRITES contre le code actuel** en J4 session 1 —
> pas portees. Ce que la reecriture a corrige au passage est note en derniere colonne : la
> version d archive portait des enonces PERIMES sur le code d aujourd hui.

| Brique | Ou | Etat | Corrige a la reecriture |
|---|---|---|---|
| Decodeur | `internal/games/halo_infinite/film/killsource` | livre, 30 ancres Theater, goldens | intact — **non touche** |
| Table des morts | `internal/migration/steps_shared_kill_events.go` | append-only + vue `_latest` | la note « pas de `;` dans les commentaires, splitSQL couperait » : **fausse aujourd hui**, `splitSQL` est conscient des commentaires `--` |
| Table des tirs | `internal/migration/steps_shared_weapon_shots.go` | idem | — |
| Persisters | `internal/persist/{kill_events,weapon_shots}_persister.go` | INSERT-only, 0 allowlist ART | — |
| Garde-fous ART | `no_art_patterns_test.go` **et** `append_only_state_guard_test.go` | les 2 tables dans les DEUX listes | l archive n en peuplait qu UNE et affirmait « un DELETE nu passerait » : **faux**, `TestNoRawDeleteOnAppendOnlyTables` le couvre, scope au nom exact |
| Pont de telechargement | `internal/sync/killcollector/bridge.go` | + `haloclient.GetFilmChunks` (types 1/2/3, UNE lecture de manifeste) | l archive assemblait `GetMatchFilm` + `GetHighlightEventsChunk` : **l en-tete n etait telecharge par personne** (decouverte J0.1) |
| Collecteur | `internal/sync/killcollector/collector.go` + `roster.go` | capability `film.kill_source`, serie, limite de temps, expvar | — |
| CLI de controle | `cmd/killsource` | `kills` / `json` / `sante` / `comparer` | intact |

**Le collecteur EST branche** (`killcollector.CollectMatch` / `CollectMatches`) mais **aucun
appelant de production ne l invoque encore** : son declencheur est la sous-commande de backfill
(phase 3, session suivante). C est voulu — voir la correction d ordre du master plan (§J4).

⚠ **Le paquet s appelle `killcollector`, pas `sync`** : le ratchet `TestSyncRootPackageFrozen`
gele la racine de `internal/sync` a 80 fichiers et impose un sous-paquet cohesif. Les appels
ecrits `sync.ChunkSourceForMatch(...)` dans ce document et dans `GUIDE_KILLSOURCE.md` §9/§12
s ecrivent donc `killcollector.ChunkSourceForMatch(...)`.

---

## PHASE 0 — REMETTRE LA BRANCHE SOUS `main`

**Pourquoi en premier** : la branche n a JAMAIS ete rebasee et `main` a beaucoup diverge. Tout
travail fait avant ce recalage sera a refaire.

### LES CHIFFRES REELS, mesures le 2026-07-31 APRES `git fetch --unshallow`

**AVERTISSEMENT DE METHODE** : avant le depliage, le depot etait un clone SUPERFICIEL de profondeur
**1**. `git merge-base` et `git rev-list` y repondaient n importe quoi — ils donnaient une base
commune qui contredisait le compte de divergence dans la meme seconde. **Aucune strategie de merge
ne peut etre etablie sur un depot superficiel.**

| | valeur |
|---|---|
| base commune des DEUX branches et de `main` | `811be64ec`, **2026-06-02** |
| commits de `main` depuis cette base | **1 855** |
| commits propres a `feat/filmdec-killweapon` | **321** |
| commits propres a `feat/filmdec-continuation` | **313** |

Les deux branches sont des SOEURS parties du meme commit, et **aucune n a jamais ete rebasee**.

### LE RECOUVREMENT DE FICHIERS, ET C EST LUI QUI DECIDE

| | fichiers |
|---|---|
| touches par `killweapon` | 1 148 |
| touches par `continuation` | 835 |
| **`killweapon` ∩ `continuation`** | **554 — quasi integralement de la documentation `.ai/`** |
| **`killweapon` ∩ `main`** | **41 seulement**, dont l essentiel est de l infrastructure (workflows CI, `.gitignore`, `.gitleaks.toml`, `openapi.yaml`, deux handlers) |

### CE QUE CES CHIFFRES IMPOSENT

**NE PAS REBASER LES 321 COMMITS SUR 1 855.** Le cout serait enorme pour une surface de conflit
reelle de 41 fichiers.

**NE PAS FUSIONNER LES DEUX BRANCHES ENTRE ELLES.** Leur recouvrement est fait de JOURNAUX
APPEND-ONLY : la fusion produirait des centaines de conflits purement mecaniques (les deux ont
ajoute a la fin des memes fichiers), sans aucune valeur. Piege connu du depot : les conflits sur
`thought_log.md` se resolvent **en BASH sur des octets bruts** — PowerShell 5.1 double-encode
l UTF-8 sans BOM.

**LA VOIE : UNE BRANCHE NEUVE DEPUIS UN `main` A JOUR**, et on y PORTE le produit — qui est une
surface bornee et connue (le paquet `killsource`, les 2 migrations, les 2 persisters, le pont, le
CLI, le correctif de `weapon_scanner`, `weaponv3`, `backfill_weapons`). Les documents `.ai/` se
RECOPIENT, ils ne se fusionnent pas. Les 67 outils `cmd/tmp_*` NE PARTENT PAS.

**Corollaire pour le prochain worktree** : le creer depuis `main`, jamais depuis l une des deux
branches — repartir de l une d elles importerait ses deux mois de derive dans le chantier suivant.

> **PHASE 0 : CLOSE — AUTREMENT.** La reconciliation du 2026-07-31 a produit
> `feat/replay2d-prod` (main `6c2b00402` + killsource + rejeu 2D, fusion bit-exacte). La branche
> neuve `feat/killsource-prod` decrite ci-dessous n a pas ete creee : elle est ABSORBEE.
> Les items restent pour memoire du raisonnement, statues `[~]`.

- [~] `git fetch --unshallow` — fait pendant la reconciliation
- [~] Creer `feat/killsource-prod` depuis un `main` A JOUR — remplace par `feat/replay2d-prod`
- [~] **Y porter le PRODUIT UNIQUEMENT** : le decodeur, les migrations, les persisters, le pont, le
      CLI. **PAS les 67 outils `cmd/tmp_*`** — ils restent sur la branche de recherche, ou le
      journal les cite comme commandes de reproduction.
- [~] Rejouer la suite complete sur la branche neuve — fait a chaque jalon (J1, J2, J3, J4)

**GATE 0** : `go build ./internal/... ./cmd/killsource`, `go vet`, suite `killsource` avec fixtures
verte, **30 ancres et goldens inchanges**, et `git log --oneline` montrant que la base est le `main`
courant.

---

## PHASE 1 — LE COLLECTEUR

### 1.1 Ou il vit, et pourquoi

**`internal/sync/`**, parce que c est la couche d ORCHESTRATION : elle combine un acces reseau (le
telechargement des chunks), un algorithme (le decodage) et une ecriture (le persister). Aucune de
ces trois responsabilites n a le droit de migrer ailleurs :

| Responsabilite | Couche | Ne doit PAS |
|---|---|---|
| decoder un film | `internal/games/halo_infinite/film/killsource` | toucher la base ni le reseau |
| telecharger les chunks | `internal/sync/killsource_bridge.go` | decoder |
| ecrire les lignes | `internal/persist/*Persister` | decider QUOI ecrire |
| enchainer les trois | `internal/sync/` (le collecteur) | contenir de la logique de decodage |

**Le decodeur reste title-specific** (`games/halo_infinite/`) et c est correct : le format de film
est propre a Halo Infinite. Le collecteur, lui, est **title-agnostic** et se branche sur une
CAPABILITY.

### 1.2 La capability

- [x] Ajouter une cle fine dans `config/titles/halo_infinite/mappings/capabilities.toml`
      (proposition : `film_kill_source`), absente de `halo_5`
      → **`film.kill_source`** (le vocabulaire du depot ecrit avec des points). TROIS endroits,
      pas un : la constante `games.CapFilmKillSource`, `AllCapabilityKeys()`, et **la
      CapabilityMap codee en dur de l adapter** — un test de PARITE exige que le TOML et le
      hardcode coincident, et il a mordu.
- [x] Le collecteur teste `HasCapability(...)`, **jamais `slug == "halo_infinite"`** — le ratchet
      `no_slug_comparison_test.go` l interdit
      → `c.caps.Has(games.CapFilmKillSource)`, et la porte passe **AVANT tout appel reseau**
      (un test le verifie en comptant les appels au client).
- [x] Degradation : `ErrCapabilityNotSupported` -> le cycle continue sans decoder, aucun panic
      → outcome `capability-absente`, **aucune erreur remontee** : une erreur ferait s arreter
      une passe entiere sur un titre qui ne supporte simplement pas la feature.

### 1.3 Quand il tourne

**PAS au sync primaire** : le film n existe pas encore quand le match arrive. Le collecteur est une
passe SEPAREE, sur des matchs deja presents dans `match_registry`.

**LE COUT EST UNE CONTRAINTE DE CONCEPTION, PAS UN DETAIL** :

| Regime | Cout mesure |
|---|---|
| 4v4 (8-10 joueurs) | 8 a 30 s par film |
| **BTB (36 participants)** | **11 minutes** (mesure du 2026-07-31 sur `4f77afc1`) |

Consequences a respecter :
- [x] **Un seul decodage a la fois dans un process** : les parametres de replication de `filmdec`
      sont des globaux de paquet ; `Decode` serialise deja par un verrou, ne pas contourner
      → `CollectMatches` boucle EN SERIE, et le commentaire dit pourquoi paralleliser ne
      donnerait rien (N goroutines empilees sur le meme verrou).
- [x] Le collecteur travaille en TACHE DE FOND, jamais dans le chemin d une requete HTTP
      → aucun handler, aucun appelant dans `api/`.
- [x] Une limite de temps par match, et un compteur des abandons
      → `defaultKillSourceTimeout = 22 min` (2x le pire cas connu) + compteur expvar
      `killsource_abandons_delai` + `KillSourceSummary.Timeouts`. ⚠ L abandon se reconnait au
      ctx du MATCH, pas a celui de l appelant : un arret demande (shutdown) n est PAS un abandon.
- [x] Journaliser `slog.InfoContext` a l entree et a la sortie avec `match_id` et la duree

### 1.4 Ce qu il ecrit

- [x] Traduire `killsource.Kill` -> `persist.KillEventInsert` (l exemple est dans l en-tete de
      `killsource_bridge.go` ; **le suivre, il porte les pieges**)
      → `killcollector.BuildKillSourceBatch`, **exportee a dessein** : le backfill de la session
      suivante doit emprunter EXACTEMENT ce chemin. Une seconde traduction serait une seconde
      chance de confondre les trois etats.
- [~] Ecriture via `persist.BatchBuilder.SetKillSource(...)` puis `Submit()` — **jamais un INSERT
      direct** (regle anti-ART, ADR 0019/0030)
      → **le chemin builder EXISTE et est cable** (`SetKillSource` / `SetWeaponShots`, appeles
      par `CombinedPersister` dans la meme fenetre de lease). Mais le collecteur emprunte le
      chemin DIRECT `PersistPass`, et c est la conception : une passe de decodage arrive sur un
      match DEJA insere, donc sans `Shared.Match` — `SharedPersister` y serait un no-op, meme
      raison que pour `EventsCompletionPersister`. Aucun INSERT direct nulle part : les deux
      chemins passent par le persister.
- [x] **Les trois etats de l assistant survivent jusqu en base** : `assist_known` FAUX = on ne sait
      pas, distinct de `assist_gamertag` NULL = pas d assistant. Les confondre publierait des faits
      jamais observes.
      → verifie SUR FILM REEL : 3 « on ne sait pas », 63 « pas d assistant MESURE » sur les
      90 morts de `9b191a7f`. Le test echoue si la population « pas d assistant mesure » est vide.
- [x] **`null` n est jamais zero** sur les parts de degats ; aucun plafond a 100 (le decodeur sort
      des valeurs jusqu a 228, ce sont des donnees)
      → un test ecrit 228 et le relit a 228. Le seul plafond est celui du TYPE (255) et il
      **journalise en WARN** en disant d elargir le type, pas d ecreter la valeur.
- [x] Renseigner `decoder_rev` — il permet de RE-DECODER de facon ciblee au lieu de tout reprendre
      → `KillSourceDecoderRev`, et le persister REFUSE une passe sans lui.
- [x] Publier les compteurs de sante en `expvar` (ADR 0009), dont **`assist_extra_count`** : c est
      le declencheur de migration vers une table fille si l hypothese << un seul assistant >> tombe
      → 9 compteurs `killsource_*` + les `Health.ExpvarPairs()` du decodeur. `assist_extra_count`
      est publie **en plus** d etre stocke sur les lignes : un compteur qu il faut interroger en
      SQL pour voir bouger n alerte personne.

**GATE 1** : un test d integration qui decode un film de fixture, ecrit en base de test, et relit
par la vue `_latest` ; `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` ;
les trois garde-fous anti-ART verts ; `assist_extra_count` interrogeable en SQL.

**GATE 1 — ATTEINT le 2026-08-01.**

| controle | resultat |
|---|---|
| film reel decode -> ecrit -> relu PAR LA VUE | `9b191a7f`, **90 morts**, 2 passes (48 s) |
| `-tags=integration -p 1` sync + persist | vert, **une** exception : `TestWorker_Run_PersistsAndACKs` (flake de timing WAL connu sous charge) — **repasse 3x isole** |
| garde-fous anti-ART | `no_art_patterns_test.go` (4 tests) + `append_only_state_guard_test.go` verts, **0 entree d allowlist ajoutee** |
| `SUM(assist_extra_count)` | interrogeable, vaut 0 — et un test FABRIQUE le cas non nul pour prouver que le compteur peut bouger |
| `go test ./...` | vert apres correction de 3 ratchets (gel du god-package `sync`, compte de capabilities, parite TOML/hardcode) |
| 3 artefacts de rejeu | inchanges (le decodeur n a pas ete touche) |

⚠ **Le test du film reel se SKIPPE sans `KILLSOURCE_FIXTURES`** (les films pesent 107 Mo et ne
sont pas versionnes — meme regime que les goldens du decodeur). Le skip porte la commande exacte
pour le rejouer. **En CI il ne tourne donc pas** : c est une limite assumee, pas un oubli.

---

## PHASE 2 — LA BASCULE DES LECTEURS

### 2.1 La strategie : une VUE de compatibilite, pas huit reecritures

`killer_victim_pairs` devient une **VUE** sur `match_kill_events_latest`. Les 8 lecteurs continuent
sans une ligne modifiee, et la deduplication arrive le jour de la bascule.

**Le gain immediat, et c est le vrai motif de tout ce chantier** : la table actuelle porte
**46,8 % de doublons** (248 566 lignes pour 132 313 morts) et gonfle les agregats carriere d un
facteur **1,879** — sur un duo reel, l ecran affiche **101 frags la ou il y en a 29**.

### 2.2 Les sites, tous recenses

> ⚠ **L INVENTAIRE CI-DESSOUS ETAIT INCOMPLET DE MOITIE.** Re-fait par grep le 2026-08-02
> (session 3) : **20 sites de LECTURE** en production, pas 8, plus 3 ecrivains, 3 sites de DDL
> et 5 sites speciaux. Manquaient : `queries_career_encounters.go:191` et `:294`,
> `queries_relations_moments.go:104`, `queries_squad.go:201`,
> `engagement_score_repo_queries.go:199`, `highlight_events_repo.go:135`,
> `halo5/halo5_match_events_source.go:46`, `sync/engagement.go:468`,
> `sync/enrichment_discipline.go:39` et `:47`, `analysis/identity.go` (v_gamertag_lookup),
> plus `ops/seed_demo_corpus.go:343` et `ops/snapshot_export.go:26` cote sites speciaux.
> *Lecon : une liste de sites qu on recopie d une session a l autre vieillit sans le dire.*

| # | Site | Ce qu il attend |
|---|---|---|
| 1 | `queries_career_encounters.go:48` (Q26) | cumul — rencontres carriere |
| 2 | `queries_career_encounters.go:98` (Q27) | cumul — nemesis / souffre-douleur |
| 3 | `queries_match_detail.go:133` (Q23b) | cumul — kills croises |
| 4 | `queries_match.go:339` (Q19b) | cumul — comparaison 2 joueurs |
| 5 | `compare_repo.go:290` (`qKV`) | **copie textuelle de Q19b** — traiter ENSEMBLE |
| 6 | `queries_match.go:350` (Q20) | journal horodate — tug-of-war, timeline K/D, nemesis |
| 7 | `skill_v2_quit_penalty.go:226` | journal horodate — penalite de depart LUSR |
| 8 | `events_replay.go:71` | presence (`NOT EXISTS`) |
| S1 | `internal/ops/seed_demo.go:108` | la table est COPIEE dans la base de demo |
| S2 | `cmd/rebuild_mp/main.go:24` | `v_killer_victim_full` dans les vues dependantes |

- [!] **Traiter 4 et 5 dans le meme commit** (regle des <= 2 copies) et poser le garde-rail —
      NON TRAITE : la bascule des lecteurs est reportee (cf. journal session 3), les deux copies
      restent identiques et lisent la meme table.
- [x] `v_killer_victim_full` NE SURVIT PAS — **FAIT le 2026-08-02**. Supprimee du DDL
      (`ApplyResolutionViews`), de la migration, des vues dependantes de `rebuild_mp` et du
      contrat de snapshot. Son unique lecteur, Q20, lit `killer_victim_pairs` directement et
      rend les SIX MEMES colonnes : les deux `LEFT JOIN` re-joignaient `v_gamertag_lookup` pour
      produire deux colonnes qui portaient deja ces noms-la, et DuckDB renommant silencieusement
      les homonymes, `kvf.killer_gamertag` designait la colonne de la table — jamais la jointure.

### 2.3 Deux arbitrages DEJA TRANCHES, a ecrire dans le DDL

- [~] **Les bots restent EXCLUS des agregats carriere** — sans objet tant que les lecteurs
      carriere lisent l ancienne table, qui ne porte aucune ligne de bot. A re-armer le jour de
      la bascule des lecteurs (le filtre `NOT LIKE 'bid(%'` de Q27 est un NO-OP aujourd hui,
      Q26 n en a pas : c est la que le piege se refermera).
- [~] **Le BTB reste INCLUS dans les cumuls, interdit ligne par ligne** — idem : `publishable`
      est ecrit dans `match_kill_events` et n a pas encore de lecteur.

**GATE 2** : pour chaque lecteur bascule, une requete AVANT/APRES sur le meme match rend le meme
resultat a la deduplication pres ; aucun lecteur ne casse en silence ; `go test ./internal/...`.

**GATE 2 — RESULTAT DU 2026-08-02, sur les deux bases reelles.** Onze mesures (lignes, matchs,
cles distinctes, couples carriere, top duo Q26 et Q27, lignes nommees, xuids du resolveur, xuids
NOMMES par le resolveur, sonde de presence, instants non nuls) prises AVANT sur la sauvegarde et
APRES sur la base migree : **les onze sont identiques sur les deux titres**. Halo Infinite
250 139 / 1 343 / 133 886 / 49 708 / 101 / 101 / 18 219 / **18 183** ; Halo 5 268 337 / 2 754 /
268 337 / 86 484 / 274 / 274 / 14 652 / **14 652**. Aucun lecteur ne bouge — c est le resultat
attendu d une session qui ALIMENTE la table canonique sans encore y brancher personne.

---

## PHASE 3 — LE BACKFILL

> **MESURE DU 2026-08-01 QUI CHANGE LE PLAN DE CETTE PHASE : LE BACKFILL EST 100 % HORS LIGNE.**
> Le guide du decodeur affirme (§9, piege 3) que « le cache disque local ne stocke QUE les chunks
> de replication » et que le HIGHLIGHT est re-telecharge a chaque appel. **C EST FAUX sur le
> cache d aujourd hui** : sur les **949** films utilisables (951 repertoires, 2 vides), les 949
> portent sur disque leur en-tete (type 1), TOUTES leurs replications (type 2) ET leur kill-feed
> (type 3). Croisement manifeste x fichiers presents, 949/949 sur les trois criteres.
>
> Consequence : **le backfill local n a besoin ni de reseau, ni de tokens, ni du CDN** — donc ni
> de la fenetre d auth, ni du risque d expiration pendant qu il tourne. Verifie sur pieces sur
> trois films (`000d5950` type-3 = index 27 present · `4f77afc1` index 62 present · `9b191a7f`
> index 32 present).

> **LE PERIMETRE, RE-MESURE LE 2026-08-01 (les chiffres du plan ont vieilli, la base a grossi)** :
>
> | quantite | valeur |
> |---|---|
> | matchs au registre | **1 926** |
> | porteurs de `killer_victim_pairs` | **1 343** |
> | films en cache utilisables | **949** (951 repertoires, 2 vides) |
> | films en cache qui correspondent a un match du registre | **949 / 949 — aucun orphelin** |
> | porteurs de paires **avec** film | **949** |
> | porteurs de paires **sans** film — perdus DEFINITIVEMENT | **394, soit 29,3 %** |
> | `killer_victim_pairs` | **250 139 lignes pour 133 886 cles distinctes — 46,5 % de doublons** |
>
> Les deux derniers confirment les deux enonces qui motivent tout le chantier : « au moins 28 % »
> etait juste (29,3 %), et le taux de doublon n a pas bouge (46,8 % -> 46,5 % sur une table qui a
> grossi de 1 573 lignes).

> **COUT REEL, MESURE (machine locale, binaire `cmd/killsource`)** :
>
> | film | chunks | duree | par chunk |
> |---|---|---|---|
> | `e157a672` | 8 | 1,6 s | 0,20 s |
> | `cf040013` | 10 | 1,9 s | 0,19 s |
> | `000d5950` | 28 | 10,4 s | 0,37 s |
> | `fccc61cd` | 31 | 14,1 s | 0,46 s |
> | `9b191a7f` | 33 | 14,4 s | 0,44 s |
> | `4f77afc1` | 63 | **575 s** (9 min 35) | 9,1 s |
> | `1c4c63c2` | 69 | **1 145 s** (19 min 05) | 16,6 s |
>
> ⚠ **PASSER DE 63 A 69 CHUNKS (+9,5 %) DOUBLE LE TEMPS (+99 %).** Le cout par chunk va de 0,20 s
> a 16,6 s — **un facteur 83**. Ce n est pas une pente, c est un mur. **A PROFILER, PAS A
> OPTIMISER** (consigne utilisateur du 2026-07-31) : une non-linearite de cet ordre suggere une
> faille logique (cout quadratique en entites ou en enregistrements — le BTB a 36 participants
> contre 8), pas un manque de puissance. **Non traite dans cette session.**
>
> Repartition du corpus par nombre de chunks : **120** films ≤ 20 · **554** de 21 a 30 · **216**
> de 31 a 40 · **35** de 41 a 50 · **24** au-dela de 50.
>
> **Estimation du backfill complet, avec ces mesures** : les 925 films ≤ 50 chunks a ~0,45 s/chunk
> ≈ **2 h 30 a 3 h** ; les 24 films au-dela de 50 chunks sont a compter SEPAREMENT — entre 10 min
> et 20 min chacun, soit **4 h a 8 h a eux seuls**. Les passer EN DERNIER reste le bon ordre.

> **⚠ LE MUR DE COUT CI-DESSUS EST TOMBE LE 2026-08-01 (J4 session 2). Les chiffres de cout de
> cette phase sont donc PERIMES — ils decrivent le decodeur d avant le correctif.** Cause
> mesuree au profil CPU (film `34bb3bc8` : 102 s sur 131 s dans `filmdec.ReadBits`, appele
> depuis UNE fonction) : `consumeObjectMultiplayerProperties` sautait le corps d un TLV par
> `ReadBits(8)` **une fois par octet**, plafonne a 1 048 576 iterations — sur une traversee mal
> alignee (la quasi-totalite des tentatives d un calibrage), la longueur LEB128 lue est du bruit
> et declenche ce million de lectures. Le corps n est jamais interprete : `Skip(8N)` a le meme
> etat final. **Gain mesure : 1 145 s -> 46,7 s sur le plus gros film (x24,5), 331 s -> 33,7 s,
> 575 s -> 51 s.** Le facteur 83 sur le cout par chunk tombe a ~5. Equivalence prouvee
> DEUX FOIS : 3 artefacts de rejeu reconstruits bit-identiques, et JSON `killsource`
> bit-identique sur 8 films dont les 4 de reference.

- [x] ~~**949 films en cache**, 1 908 matchs au registre, 1 325 porteurs de paires~~ → re-mesure ci-dessus
- [x] ~~Estimation : ~900 films 4v4 a 8-30 s = **2 h 30 a 7 h 30** ; les BTB a 11 min chacun~~ → **re-mesure
      apres correctif : 951 films, 28 769 chunks, passe complete en ~1 h 15** (voir journal J4-2)
- [x] Sous-commande dediee du CLI principal (`levelup`), pas un script jetable
      → `levelup backfill-killsource` (`cmd/levelup/cmd_backfill_killsource.go`) : deux passes
      (films puis credit), `--dry-run`, `--limit`, `--films-only`, `--credit-only`, `--force`,
      restitution des compteurs de sante en fin de passe (un process qui s arrete emporte ses
      `expvar` avec lui).
- [x] Reprenable : `decoder_rev` + la vue par PASSE permettent de rejouer sans tout refaire
      → la selection lit `match_kill_events_latest` et saute les matchs deja a jour ; **verifie
      sur pieces** : une seconde passe `--limit 25` a traite 25 films NEUFS, pas les 20 deja faits.
- [x] **Les films Theater EXPIRENT cote serveur** : ce qui n est pas en cache est perdu
      definitivement. Au moins **28 % des matchs** n auront jamais de ligne issue d un film — d ou
      `source_tag` NULLABLE et le chemin `highlight_events` comme SECOND producteur.
      → le second producteur EXISTE depuis J4-2 (`killcollector/credit.go`), avec preseance
      film > credit testee.
- [x] Prevenir avant toute execution sur le VPS de prod
      → **aucune execution VPS dans cette session** (interdit de l ordre de mission). La passe
      prod reste a planifier en J5, avec l utilisateur.
- [x] **HORS LIGNE, et c est mesure** : la source de chunks du backfill est
      `killcollector.LocalCacheFilms` — elle lit le manifeste et les chunks du cache disque et
      **n a aucun client HTTP derriere elle**. Il est donc structurellement impossible que la
      passe parte sur le reseau ou expire une authentification en cours de route.

**GATE 3** : le backfill tourne sur 20 films, les compteurs de sante sont dans le domaine, et un
`SELECT` de controle montre que les agregats carriere ont DEGONFLE du facteur attendu.

**GATE 3 — ATTEINT le 2026-08-01 (J4 session 2).**

| controle | resultat |
|---|---|
| backfill sur 20 films puis EN ENTIER | **949 films ecrits, 74 569 morts, 2 h 52 min 57 s** ; 2 films absents (repertoires vides), **0 erreur, 0 abandon sur delai, 0 sans kill-feed** |
| producteur credit-seul | **394 matchs, 50 125 morts, 33 s** ; **949 refuses par la PRESEANCE** (film prioritaire) ; 583 sans evenement |
| **couverture** | **1 343 matchs** — la cible exacte du plan (949 par le film + 394 par le credit), et **100 % de ce qui est recuperable** : les 583 restants portent `events_loaded = TRUE` sans evenement, c est-a-dire un 404 deja constate par le sync (527 datent de 2023) |
| lignes joignables | 124 694 lignes ; **123 244 xuid de victime (98,8 %)**, 123 786 xuid de tueur (99,3 %) |
| compteurs de sante | dans le domaine — cf. detail ci-dessous ; **une exception : `assist_extra_count = 5`** |
| `go test -tags=integration -p 1 ./...` | **vert** (parite CI) |
| `go test ./...` | vert |
| ratchet lint (`--new-from-merge-base=origin/main`, cache purge) | **0 issue** |
| 3 artefacts de rejeu | **bit-identiques**, revérifies APRES toutes les modifications de la session |

**LE `SELECT` DE CONTROLE — LE DEGONFLEMENT EST LA, ET IL TOMBE SUR LE CHIFFRE ANNONCE.**

```
population : les 1 343 matchs couverts
  killer_victim_pairs        250 139 lignes
  match_kill_events_latest   124 694 lignes
  facteur de degonflement    2,006

le duel de controle, joint par XUID (Luigi Numba 01 -> JGtm) :
  avant   101 frags affiches
  apres    29 frags
```

**101 -> 29 : c est EXACTEMENT le chiffre que le plan annoncait** (« sur un duo reel, l ecran
affiche 101 frags la ou il y en a 29 »), retrouve sans avoir ete cherche. Les quatre duels
suivants degonflent pareil : 73 -> 24, 72 -> 18, 71 -> 24, 64 -> 19.

**PROVENANCE DES LIGNES** — la portee voyage avec la donnee :

```
read_path          lignes   matchs   publiables   avec source de degat
marche             57 278      947       39 242   57 278
scan               17 291      923       13 104   17 291
highlight-events   50 125      394       50 125        0   <- credit seul, par construction
```

**LES TROIS ETATS DE L ASSISTANT, EN BASE** : 52 466 « on ne sait pas » · 49 147 « pas
d assistant MESURE » · 23 081 assistants nommes. Les trois populations sont non vides : la
distinction survit jusqu au bout de la chaine.

⚠ **`assist_extra_count = 5` — LE DECLENCHEUR DE L ADR 0026 A BOUGE POUR LA PREMIERE FOIS.**
Il valait 0 sur tout ce qui avait ete decode jusqu ici ; sur 74 569 morts, **5** portent plus
d un assistant distinct. L hypothese de schema « un seul assistant » n est donc pas fausse a
l echelle du corpus, mais elle n est plus universelle. Le compteur a fait exactement ce pour
quoi il a ete pose. **Arbitrage superviseur** : migrer vers une table fille, ou documenter le
plafond a 1 assistant comme une perte connue de 5 lignes sur 74 569 (0,007 %).

**LA VENTILATION DES TIRS** : 49 392 lignes sur **941 matchs**, 6 040 joueurs, **16 397
publiables** (33,2 % — la porte des +-10 % fait son travail, Fiesta et grand format refuses).
`MAX(player_index) = 26` : la preuve que la lecture 5 bits etait necessaire, une lecture 4 bits
aurait fusionne ces joueurs avec ceux d indice 26-16 = 10.
`killsource_tirs_indices_non_resolus = 1 593` : les joueurs dont le motif de xuid n apparait pas
dans le flux — ils n ont AUCUNE ligne, et c est ce compteur qui empeche de confondre leur
absence avec « ce joueur n a pas tire ».

---

## PHASE 4 — CE QUE LE PRODUIT EXPOSE

### 4.1 Ce qui est fiable, et a quel grain

| Donnee | Grain | Statut | Reserve |
|---|---|---|---|
| tueur / victime / instant | par mort | solide | — |
| **source du degat fatal** | par mort | 30 ancres Theater, 0 etiquette fausse | 206/468 tags sans nom -> << Autres >> |
| **assistant** | par mort | multiset = API a l unite sur 2 films | 3 etats a preserver |
| **parts de degats** | par mort | 4 jambes + 1 confrontation terrain (7ter.104) | valeur exacte inconfrontable |
| divergence credit/source | par mort | 8/8 confirmees au Theater | — |
| **tirs par arme** | joueur x arme x match | k = 1, **84 %** des joueurs en mode standard | Fiesta et BTB NON livrables |
| **precision par joueur** | joueur x match | taux de remplissage 0,4267 vs 0,4462 API, r = +0,82 | sans reference externe |
| **precision par arme** | joueur x arme | **4 armes seulement** a +-0,03 | voir le piege ci-dessous |

### 4.2 LE PIEGE A NE JAMAIS OUBLIER

**Ne JAMAIS publier le taux par arme calcule sur le corpus entier.** Il **INVERSE l ordre**
MA40/Sidekick : brut, le MA40 sort 8 points devant ; a la reference API, c est le Sidekick qui est
devant de 3 points. Ce n est pas une imprecision, c est une reponse fausse.

Armes publiables, sur les joueurs dont l arme domine >= 80 % des tirs decodes :
`MA40 AR +0,0249` · `BR75 -0,0108` · `Mk51 Sidekick -0,0050` · `Bandit Evo +0,0132`.
Les armes a projectile sont fausses d un facteur 30 a 60 — **non publiables**.

> **ETAT AU 2026-08-01 (J4 session 2) : LA TABLE EST REMPLIE, ET ELLE NE PUBLIE RIEN.** Le
> producteur (`killcollector/shots.go`) ecrit `match_weapon_shots` dans la MEME passe de
> decodage que les morts — un film decode une fois alimente les deux tables. Il ne calcule
> AUCUN taux par arme : la porte de publication (+-10 % du `shots_fired` de l API) est
> evaluee par le persister, jamais fournie, et **aucun lecteur n existe**. L interdit du §4.2
> est donc tenu par construction, pas par vigilance.
>
> Deux points a savoir avant de lire la table :
> - **l indice de joueur y est lu sur 5 bits** (`analysis.FireEvent.PlayerIndex5`), pas 4. La
>   lecture 4 bits SATURE a 15 : au-dela de 16 joueurs elle FUSIONNERAIT les tirs de deux
>   joueurs sur une meme ligne. Constate sur les donnees ecrites : `MAX(player_index) = 23`.
>   ⚠ `FireEvent.PlayerIndex` (4 bits) est CONSERVE tel quel pour la correlation d armes de
>   production : la corriger ne repare rien tant que `getXuidToPI` derive l indice de l ordre
>   de la base (indiscernable d une permutation au hasard). Remplacement hors perimetre J4-2.
> - **l indice -> xuid est resolu DANS LE FILM** (motif du xuid au bit pres, 5 bits devant),
>   jamais par l ordre de la base.

### 4.3 Capabilities

- [x] Une cle par famille de donnee, pas une seule fourre-tout : le kill enrichi, les tirs par arme,
      la precision. Un titre peut avoir l un sans les autres.
      *FAIT 2026-08-02 (J4 session 4).* **TROIS familles, trois cles, et aucune n en couvre deux** :
      `film.kill_source` (le kill enrichi, deja posee en session 1), **`film.weapon_shots`
      (NOUVELLE** — la ventilation des tirs, `shared.match_weapon_shots`) et
      `match.weapon.accuracy` (la precision, deja au vocabulaire). Les TROIS endroits exiges par le
      test de parite sont faits : constante `games.CapFilmWeaponShots`, `AllCapabilityKeys()`
      (21 -> 22, compteur mis a jour) et la `CapabilityMap` codee en dur de l adapter.
      **La cle a un CONSOMMATEUR, ce n est pas un drapeau decoratif** : `collectShots` la teste et
      degrade en silence (aucune erreur — la passe des morts reste acquise). Prouve sur film reel
      dans les DEUX sens : `film.kill_source` seule -> 90 morts et **0** ligne de tirs ; les deux
      -> 90 morts et **30** lignes (`TestTirsParArmeSuiventLeurPropreCapability`, `-tags=integration`).
- [x] La RAISON du `not_exposed` de `match.weapon.accuracy` corrigee pour Infinite — elle etait
      PERIMEE. Elle disait « pas d events weapon_drop, table non peuplee » ; la table des tirs est
      desormais remplie. La vraie raison est le §4.2 : le taux calcule sur le corpus entier
      INVERSE l ordre MA40/Sidekick. Ecrit dans `adapter.go` ET dans `capabilities.toml`, avec le
      critere de bascule (une population de publication declaree DANS LE CODE + l ecart a l API
      publie a cote du taux). *Doc inversee sur un flag = incident en attente (anti-pattern n°9).*
- [~] Cote web : `useFieldLabel()` / `useOutcomeLabel()`, jamais de libelle en dur ; strings FR ET EN
      — **aucune surface produit n expose ces donnees**, donc rien a localiser ici. La regle a en
      revanche ete appliquee la ou une surface EXISTE (rejeu 2D, lot 3.2) : les libelles d armes,
      de grenades et de capacites sont passes du code Go/TS aux mappings du titre, bilingues.
- [~] Couleurs par tokens semantiques uniquement — idem : rien de nouveau a l ecran pour ces
      familles. Verifie sur le diff de la session : 0 hex, 0 classe Tailwind couleur.

**GATE 4** : `make check-types`, `make test-web`, aucun hex ni classe Tailwind couleur dans
`features/`, i18n FR+EN complet.

---

## JOURNAL D EXECUTION

### 2026-08-01 — J4 session 1 : les briques d ecriture + le collecteur

**Perimetre traite** : J4.0 (garde de CI), les 2 migrations, les 2 persisters, le pont, le
collecteur, GATE 1. **Phase 0 `[~]`** (close autrement par la reconciliation), **phase 1 close**.
Phases 2/3/4 : non entamees, conformement a l ordre corrige du master plan (P1 -> P3 -> P2 -> P4).

**Commits** : `e1b46d9d8` (J4.0) · `32589dad4` (migrations + persisters) · `d06df0fe6` (pont +
collecteur) · `36fc76835` (sous-paquet + ratchets).

**LES PREMISSES DU PLAN QUI ETAIENT FAUSSES, ET CE QUE LEUR CORRECTION A COUTE OU RAPPORTE.**
C est le resultat le plus utile de la session, parce qu il porte sur des enonces qui servaient
d argument :

1. **« Le cache ne stocke que les chunks de replication »** (guide §9) — **FAUX**. Les 949 films
   utilisables portent en-tete + replication + kill-feed sur disque. Le backfill de la session
   suivante est donc **integralement hors ligne** : ni reseau, ni tokens, ni CDN. C est le
   changement de plan le plus important de la session, et il RETIRE un risque (l expiration
   serveur pendant que le backfill tourne).
2. **« splitSQL couperait sur un `;` dans un commentaire »** — **FAUX aujourd hui** : le decoupeur
   est conscient des commentaires `--`. L avertissement recopie de l archive decrivait une
   version anterieure.
3. **« Ce scan ne detecte pas un DELETE nu »** (commentaire d archive sur `no_art_patterns`) —
   **FAUX** : `TestNoRawDeleteOnAppendOnlyTables` le couvre, scope au nom exact de la table.
   L archive n inscrivait les tables que dans UNE des deux listes de garde ; elles sont
   desormais dans les DEUX.
4. **« Le collecteur va dans `internal/sync/` »** — vrai, mais pas a la racine : le ratchet
   `TestSyncRootPackageFrozen` gele celle-ci a 80 fichiers. D ou `internal/sync/killcollector`.
   **Le plan n avait pas tort, il etait imprecis** — et c est le ratchet qui l a dit, pas une
   relecture.
5. **La question laissee ouverte par le guide §12.5** (« le persister accepte-t-il un tueur
   nomme sans XUID ? », cas `tueur-bot` de RE_LOG 7ter.79) — **tranchee : OUI**, et elle a
   maintenant un test. C etait le seul point du guide marque « a verifier avant branchement ».

**MESURES DE COUT ET DE PERIMETRE** : tableaux complets en phase 3. Les deux chiffres a retenir
ici — **le backfill est integralement hors ligne** (949/949 films complets sur disque), et le
cout par chunk va de **0,20 s** (8 chunks) a **16,6 s** (69 chunks), un facteur **83**. Passer
de 63 a 69 chunks (+9,5 %) DOUBLE le temps. **A profiler, pas a optimiser** — non traite ici.

**Consequence sur le collecteur, appliquee** : la limite de temps par match a ete portee de
22 a **45 min** apres la mesure. 22 min etait calibre sur « 11 min doublees » ; le pire cas
mesure est 19 min, et avec une pente pareille le prochain film un peu plus gros peut couter
beaucoup plus. Une limite trop juste ne protege de rien — elle transforme un film lent mais
VALIDE en perte de donnee.

**DECOUVERTES CONSIGNEES, NON TRAITEES** :
- **(a) Deux gardes de CI `--if-present`** (`npm run lint`, `npm run test:coverage` dans
  `ci.yml`) : meme motif que le garde repare en J4.0 — un script renomme les rendrait
  silencieusement inoperants. **Verifie : les deux scripts existent aujourd hui**, donc les
  steps tournent reellement. Trou LATENT, pas actif. Non traite (l item J4.0 demandait un
  AUDIT, pas une correction generalisee) — arbitrage superviseur.
- **(b) `match_weapon_shots` n a AUCUN PRODUCTEUR.** La table, son persister et sa porte de
  publication sont ecrits et testes, mais rien ne les remplit : la ventilation des tirs vient du
  scanner de fire-events (`analysis`), pas de `killsource`, et son cablage n etait pas au
  perimetre de cette session. **C est l etat voulu**, il est juste bon de le savoir avant de
  lire la table.
- **(c) Le second producteur de `match_kill_events` (chemin `highlight_events`, credit seul)
  n existe pas non plus.** Sans lui, la table couvrira 949 matchs sur 1 325 porteurs de paires,
  et le retrait de `killer_victim_pairs` (phase 2, etape 7) reste impossible. Deja ecrit dans le
  DDL ; rappele ici parce que c est la dependance qui gouverne la phase 2.
- **(d) `killer_victim_pairs` figure dans `internal/ops/seed_demo.go`** : le jour de la bascule,
  la table de remplacement doit s y ajouter, sinon la demo affiche des duels vides — panne
  silencieuse visible seulement a l ecran.

---

### 2026-08-01 — J4 session 2 : le mur de cout, les deux producteurs, le backfill

**Perimetre traite** : (A) profilage du mur de cout, (B) producteur de `match_weapon_shots`,
(C) second producteur de `match_kill_events`, (D) sous-commande de backfill, (E) controle
avant/apres. **Phase 3 CLOSE.** Phase 2 (bascule des lecteurs) : non entamee, c est la session 3.

**(A) LE MUR DE COUT ETAIT UN BUG, ET LE PROFIL L A DIT EN UNE MESURE.** 78 % du temps d un
decodage dans `filmdec.ReadBits`, appele depuis UNE fonction :
`consumeObjectMultiplayerProperties` sautait le corps d un TLV en lisant **un octet a la fois**,
plafonne a 1 048 576 iterations. Le corps n est jamais interprete — les lectures jetaient leur
resultat. Sur une traversee mal alignee (la quasi-totalite des tentatives d un calibrage ou
d une recherche de signature), la longueur LEB128 lue est du BRUIT : d ou le million de lectures.
`Skip(8N)` a exactement le meme etat final, `ReadBits` n ayant d autre effet que d avancer la
position. **L equivalence est structurelle, pas empirique** — et elle a ete verifiee DEUX FOIS
quand meme (3 artefacts de rejeu bit-identiques, JSON `killsource` bit-identique sur 8 films).

| film | chunks | avant | apres |
|---|---|---|---|
| `e157a672` | 8 | 1,6 s | **1,0 s** |
| `9b191a7f` | 33 | 14,2 s | **7,3 s** |
| `34bb3bc8` | 42 | 131 s | **24,8 s** |
| `37f8b21c` | 40 | **331 s** | **33,7 s** |
| `4f77afc1` | 64 | 575 s | **51 s** |
| `1c4c63c2` | 69 | **1 145 s** | **46,7 s** |

*Ce que la mesure a appris et qui vaut plus que le gain* : la non-linearite n etait pas dans le
DECODAGE mais dans le chemin d ERREUR. Un cout qui explose avec la taille du film n implique pas
un algorithme quadratique en taille de film — ici c est la FREQUENCE des traversees ratees qui
croit, et chaque ratee coutait un million de lectures. Chercher la complexite dans le chemin
nominal aurait ete chercher au mauvais endroit.

**(B) LE PRODUCTEUR DE TIRS ECRIT DANS LA MEME PASSE QUE LES MORTS.** Un film telecharge une
fois, decompresse une fois, deux tables ecrites. Deux points ont demande une decision :
1. **L indice de joueur se lit sur 5 bits** (`PlayerIndex5`), pas 4. Ce n est pas un raffinement :
   la lecture 4 bits SATURE a 15, donc au-dela de 16 joueurs elle FUSIONNERAIT les tirs de deux
   joueurs distincts sur une meme ligne — une ligne fabriquee, qu aucune colonne ne signalerait.
   `MAX(player_index) = 23` sur les donnees ecrites : le cas est reel, pas theorique. Le champ
   4 bits reste EN PLACE pour la correlation d armes de production, avec la raison ecrite : le
   corriger ne repare rien tant que `getXuidToPI` derive l indice de l ordre de la base.
2. **La resolution indice -> xuid a du etre reecrite.** La version naive (`weaponv3.ResolveBest`)
   balaie le film UNE FOIS PAR XUID, chaque position coutant une relecture de 64 bits : mesure a
   **24 s par film** sur des films de 8 a 16 chunks, la ou le decodage des morts en coute 1 a 3.
   La resolution coutait DIX FOIS le decodage qu elle accompagne. Fenetre glissante + prefiltre
   16 bits + arret des que tout est resolu : **3,8 s/film**, et l equivalence exacte (memes
   chunks, memes positions, premiere occurrence gagnante) est TESTEE contre la version naive.

**(C) LE SECOND PRODUCTEUR EST UNE TRANSFORMATION SQL -> SQL.** Il lit `highlight_events` et
apparie avec `analysis.ComputeKillerVictimPairs` — LA fonction qui produit `killer_victim_pairs`
aujourd hui, meme tolerance de 5 ms. Trois exigences, trois tests sur base :
- **preseance film > credit** : un match porteur des deux sources ne rend QU UN jeu de lignes, et
  c est celui du film. Testee dans les DEUX ordres d arrivee ;
- **trois etats de l assistant** : `assist_known = FALSE`, et source/categorie/divergence/parts
  ABSENTES (NULL), jamais nulles-comme-zero. Le test verifie colonne par colonne ;
- **couverture** : un match sans film produit quand meme ses lignes.

**(D) LE BACKFILL EST UNE SOUS-COMMANDE, ET IL EST HORS LIGNE PAR CONSTRUCTION.**
`levelup backfill-killsource` : films (les moins chers d abord, gros en dernier) puis credit.
La source de chunks n a AUCUN client HTTP derriere elle — il n y a pas de reseau a atteindre,
donc pas d authentification a expirer. Reprenable par `decoder_rev`, verifie sur pieces.

**LE DEFAUT LE PLUS COUTEUX DE LA SESSION — L IDENTITE DES JOUEURS, ET IL ETAIT SILENCIEUX.**
Repere en cours de backfill (200 films deja ecrits), sur une remarque de l utilisateur : *« a mon
avis tu regardes la mauvaise table pour les xuid-gamertags »*. Trois faits qui s enchainent :

1. **`match_participants.gamertag` est VIDE en production** — 27 607 lignes sans nom sur 27 989,
   **4 xuids nommes sur 16 996**. Le roster du collecteur lisait cette colonne : il rendait une
   table quasi vide.
2. **La source canonique est la VUE `v_gamertag_lookup`** (`analysis.GamertagLookupViewSQL`) et
   sa cascade : bot connu -> `xuid_aliases` -> `match_participants` -> **`killer_victim_pairs`**
   -> libelle masque. `xuid_aliases` a cesse d etre alimente en avril 2026 : ce sont les
   gamertags du KILL-FEED qui couvrent les adversaires depuis. 18 219 xuids, 36 masques.
3. **Le film ne donne pas toujours un pseudo.** Quand il n en a pas, le decodeur ecrit
   `xuid:<decimal>` (`feed.go`). Cette forme EST l identite — la chercher dans une table de
   gamertags ne rend evidemment rien.

**Constat chiffre du defaut** : 16 908 morts ecrites, **10** portaient un xuid de victime. Des
lignes qu AUCUN agregat carriere ne peut joindre — c est-a-dire exactement ce que ce chantier
existe pour produire. Apres correctif, sur les memes films : **77 morts, 75 xuids victime,
77 xuids tueur**. La regle a ete centralisee en UNE copie (`MatchIdentities.Resoudre`), le
prefixe `xuid:` exporte par le decodeur (`killsource.XUIDNamePrefix`) pour qu il n existe qu un
seul litteral, et un test de non-regression couvre les deux formes de nom.

*Lecon* : **une passe qui « reussit » sans erreur peut n avoir rien produit d utilisable.** Les
compteurs disaient 20 matchs ecrits, 0 erreur ; c est une requete sur le CONTENU — pas sur le
succes — qui a montre le trou. Un backfill doit se controler sur ce qu il rend JOIGNABLE, pas
sur son taux d echec.

⚠ **DEPENDANCE A CONSIGNER POUR LA PHASE 2** : `v_gamertag_lookup` LIT `killer_victim_pairs`.
Supprimer cette table (etape 7 du chemin de retrait) sans avoir d abord rebranche la vue sur
`match_kill_events_latest` ferait retomber tous les adversaires sur « Joueur #### » — une panne
d affichage que rien ne signalerait, sur le chemin le plus visible du produit.

**DEUX AUTRES DEFAUTS TROUVES EN EXECUTANT, PAS EN RELISANT** :
- `KillSourceSummary.Deaths` n etait **jamais incremente** : chaque match journalisait ses
  dizaines de morts pendant que la synthese de passe affichait `0`. Corrige (le compte remonte
  desormais de `CollectMatch`) — c est le genre de zero qu on croit longtemps ;
- le paquet `filmdec` definit son propre `min(int, int)` dans un fichier de TEST, qui masque le
  builtin a la compilation des tests. Un `min(uint64, uint64)` legitime dans le code de
  production ne compile donc pas. Contourne, non traite (hors perimetre).

**DECOUVERTE GRAVE, NON TRAITEE — LES GOLDENS DU DECODEUR SONT DEJA ROUGES SUR LA BRANCHE.**
`TestGoldenFilms` echoue sur les 4 films de reference, **avant tout changement de cette
session** : prouve en rejouant le test sur l arbre restaure, diffs identiques au bit pres avec et
sans le correctif (A). Les goldens datent du 2026-07-31 17:49 ; **J3 (lots B/C/D) a modifie le
decodage sans que personne le voie**, parce que ce test SE SKIPPE sans `KILLSOURCE_FIXTURES` et
que le gate J3 portait sur les artefacts de REJEU, pas sur la sortie `killsource`. Les ancres
Theater, elles, tiennent (`TestGoldenAncres...`, `TestLigneDiscriminante...` verts). Ecarts :
`78919882` 91 -> 92 apparies, `fccc61cd` 84 -> 85, `000d5950` 86 -> 84, `9b191a7f` calibration
17/2 -> 16/3 a score egal. **Arbitrage superviseur requis** : re-congeler deliberement apres
revue du diff, ou instruire l ecart. *Lecon : un gate qui se skippe sur une variable
d environnement n est pas un gate, c est une option.*

> ⚠ **CORRECTION DU 2026-08-02 (session 2bis).** L attribution ci-dessus est FAUSSE : **J3 n y
> est pour rien**, et c est mesure — la signature de l ecart est identique, au bit pres, des
> `96bc56175` (J2 lot 2.5), soit AVANT le premier lot de J3. Le lot D de J3 (`bijection.go`)
> ne change rien d observable. La cause est `47c9e72ac`. Voir l entree ci-dessous.

### 2026-08-02 — J4 session 2bis : instruire l ecart des goldens

**LA CAUSE, ET CE QU ELLE N EST PAS.** Bissection sur worktree detache, 4 films, `go test`
avec `KILLSOURCE_FIXTURES` : `5b5f97c69` VERT, `47c9e72ac` ROUGE. Entre les deux, `2044b7139`
**NE COMPILE PAS** (`capture.go` reference `CompResult.Payload`, absent) — commit non
evaluable, la paire port+fusion se juge sur `47c9e72ac`. Piege de methode releve en cours de
route : `RC=1` confond echec de BUILD et echec de TEST ; deux conclusions intermediaires ont du
etre reprises apres lecture des sorties.

**Deux hypotheses refutees, chacune deux fois :**

- *Le correctif de performance de J4-2* (`Skip(8N)` au lieu de la boucle plafonnee). Refute par
  la mesure (l ecart existe des J2) ET par la lecture : `ReadBits` n a d autre effet que
  d avancer `pos`, et le plafond est CONSERVE (`tlvBodyMaxBytes = 1<<20`, exactement le
  `i < 1<<20` d origine). Le saut n est pas plus loin, il est au meme endroit. **Le correctif
  reste bon, et il n y a rien a borner.**
- *J3 lots B/C/D* — exonere par la mesure (cf. correction ci-dessus).

**LA CAUSE : `47c9e72ac`** (« reunir les deux decodeurs filmdec — fusion a trois voies »). Son
message annonce « killsource (golden) vert » : le test avait SKIPPE. Faux vert de bonne foi.

**Sous-cause A, PROUVEE** — le `br.ReadBits(2)` ajoute dans `consumeAbsoluteWithGate`
(`components_movement.go`) : champ « fini », FUN_14076e304, lu sous un predicat qui ne consomme
aucun bit ; le chemin world-object le lisait DEJA (`[2 finite]` de ses 45 bits) et la capture CE
mesure 47 bits sur 100 % de 154 158 dispatches, sans variance. La calibration l absorbe en
retrecissant l axe, et **`3*axisW + indexW` perd exactement 2 sur les QUATRE films** :
000d5950 43->41, 9b191a7f 53->51, 78919882 50->48, fccc61cd 52->50. **Controle** : neutraliser
cette seule ligne a HEAD restaure exactement `axisW=17 indexW=2` sur 9b191a7f. Le critere de
calibration ne voit que la SOMME des largeurs (le record se referme au meme bit) : le parametre
est sous-determine, le balayage re-alloue, la sortie ne bouge pas.

**Sous-cause B, NON ISOLEE** : la neutralisation des 2 bits ne restaure ni la ventilation
marche/scan ni la mediane (58 -> 57). Une seconde difference vit dans les 892 insertions de
`47c9e72ac`. Non poursuivie — son effet est borne et mesure (cf. ci-dessous) ; l isoler exige
des reverts fichier par fichier dans un commit de fusion. Consigne en Decouvertes.

**L ORACLE NE DEPARTAGE PAS, ET ON SAIT EXACTEMENT DE COMBIEN.** Comparaison **ligne a ligne**
des sorties JSON completes de `cmd/killsource json` sur les 4 films, etat fige (`5b5f97c69`)
contre HEAD (`deba72742`) — le paquet `killsource` et son CLI n ont change qu en UN fichier
entre les deux, le rendu est donc le meme et seul le decodeur differe. **Les seules differences
sont** : la chaine `calibration`, les compteurs `stats` par voie, et le champ `voie` /
`voie_identifiant` de **4 lignes au total** (2 sur 000d5950, 1 sur 78919882, 1 sur fccc61cd,
0 sur 9b191a7f). **AUCUNE ligne publiee ne change de victime, de tueur, de tag, d etiquette, de
categorie, d instant, de divergence ni d assistant : 371 sur 371 identiques.** Donc **380/380
morts de l API inchange** des deux cotes (`couverts`, `mortsBot`, `mortsParBot` identiques sur
les 4 films) et **30/30 ancres Theater inchangees**, etiquettes ET voie — aucune des 4 lignes
deplacees n est une ancre. `DESACCORD = 0` des deux cotes : quand une mort change de voie, les
deux voies disaient deja la meme chose. Ce qui bouge est la PROVENANCE INTERNE de 1,1 % des
lignes, jamais leur contenu.

**VERDICT : issue (c) au sens de la mesure — equivalence sur l oracle.** Mais la consigne
attachee a (c) (« le plafond de l ancienne boucle fait foi, restaurer le comportement »)
reposait sur une premisse tombee : il n y a pas de departage arbitraire qui bouge, il y a un
decodeur qui lit un champ de 2 bits qu il ne lisait pas, avec sa propre trace de mesure.
Restaurer reviendrait a SUPPRIMER cette correction pour conserver des nombres etablis contre un
decodeur a qui il manquait un champ. **Recommandation : re-congeler**, cause et preuve
d equivalence dans le meme commit. **Arbitrage rendu au superviseur** — la premisse etant
tombee, le choix lui revient. **TRANCHE LE MEME JOUR : re-congeler.**

**RE-CONGELEMENT FAIT**, et son diff est instructif : **13 lignes sur les 4 goldens de film, et
`cumul.golden` INCHANGE**. Les sommes agregees sont identiques au numerateur pres — marche
340/340 proposes et 334 apparies, scan 37/37 et 29 apparies — donc **seul le PARTAGE par film a
bouge, pas la ventilation du chantier**. C est le controle le plus fort qu on pouvait esperer :
si la re-allocation avait degrade quoi que ce soit, le cumul l aurait montre.

**LE BACKFILL.** `match_kill_events` persiste `read_path VARCHAR NOT NULL` : la voie de lecture
EST en base, et 4 lignes sur 371 (1,1 %) y porteraient une valeur differente ; tous les autres
champs sont identiques. **Si re-congelement : RIEN A REJOUER** — les 124 694 lignes / 1 343
matchs ont ete produites par le decodeur ACTUEL, celui dont les lignes sont identiques a celles
du decodeur fige. **Si restauration : a rejouer** (2 h 53, reprenable par `decoder_rev`), pour
~1 % de lignes dont seule la colonne de provenance differe. *Reserve honnete : l equivalence est
mesuree sur les 4 films de reference, pas sur les 1 343 matchs ; le mecanisme (budget de bits
total conserve) est general, la mesure ne l est pas.*

**LE DEFAUT STRUCTUREL — 3e occurrence.** `TestGoldenFilms` fait `t.Skip` sans
`KILLSOURCE_FIXTURES`, ce qui a laisse `47c9e72ac` annoncer un golden vert en toute bonne foi.
Apres le garde feedback-drawer (J4.0) et le ratchet knip (J3.3), c est la troisieme fois qu un
garde ne peut pas echouer. *Lecon, la meme qu en J4.0 : un garde qui ne peut pas echouer ne
garde rien — et un gate optionnel est un garde qui ne peut pas echouer.*

**CORRECTIF POSE LE MEME JOUR** : `TestGoldenMiniBobine` tourne **sans fixture et sans variable
d environnement**, donc dans le `go test ./... -p 1` du job de couverture, en 0,63 s.

- **La bobine** (`testdata/minibobine_000d5950`, 3,8 Mo) est un **PREFIXE CONTIGU** des 6
  premiers chunks de `000d5950`, en-tete compris, suivi de son chunk HIGHLIGHT. Contigu et
  depuis le debut **parce que le decodeur l exige** — et c est une premisse du plan qui tombe :
  le patron de la mini-bobine de J2 (paquets choisis, concatenes hors continuite) y decode
  **ZERO** mort. Mesure : en-tete + chunk 01 + highlight (644 Ko) -> 0 candidat ; prefixe 00-03
  -> 6 morts ; prefixe 00-05 -> 10 morts, et c est ce dernier qui atteint la ligne `01:12`.
- **Ce qu il fige** : les **10 lignes publiees EN ENTIER** (tag, etiquette, categorie, credit,
  assistant, voie, origine, divergence), dont **3 ancres Theater** — 00:35 Disruptor,
  00:44 Skewer, 01:12 M41 SPNKr — et **la ligne de divergence `01:12`**, la seule du prefixe ou
  la source appartient a la victime.
- **Trois gardes, pas un** — la lecon de J4.0 appliquee : bobine absente ou tronquee = echec
  bruyant (comptage de chunks) ; **plancher de 10 lignes publiees**, qui interdit de figer une
  sortie vide (un golden fige une sortie, il ne dit pas qu elle est NON TRIVIALE) ; recette de
  fabrication **executable**, qui retrouve le chunk HIGHLIGHT **par son contenu** (n27 ici, n62
  en BTB — l outillage de RE a deja perdu un kill-feed entier en le bornant, 7ter.52).
  Regeneration verifiee **byte-identique** (sha256 des 7 chunks).
- **TEMOIN DE DETECTION JOUE**, parce qu un garde dont on ignore PAR OU il detecte est un garde
  qu on croit avoir : en neutralisant le `br.ReadBits(2)`, la sortie de la bobine CHANGE
  (`recordStateParam` 0 -> 2, croissance 1.000 -> 1.004). Les 10 lignes publiees, elles, ne
  bougent pas — coherent avec le 371/371 du jour. **Le canal de detection est la ligne de
  calibration**, et c est ecrit dans le fichier de test.
- **Portee declaree** : la calibration tombe en PROFIL PLAT sur un prefixe (l echantillon ne
  rend aucun record de biped), donc ce garde ne verrouille PAS le balayage lui-meme — seul
  `TestGoldenFilms`, sur les films entiers, le fait.

---

### 2026-08-02 — J4 session 3 : le producteur live, et la mesure qui a arrete la bascule

**Perimetre traite** : item zero (CI), re-inventaire des sites, producteur live des kill-events
pour les DEUX titres, reprise dedupliquee de l ancienne table, suppression de
`v_killer_victim_full`, GATE 2. **Phase 2 : PARTIELLE** — la table est alimentee et les deux
titres y ecrivent, mais AUCUN lecteur n a bascule et `killer_victim_pairs` n est pas retiree.
**Phase 4 : NON ENTAMEE.**

**LA MESURE QUI A TOUT DECIDE — LA PASSE DE FILM PUBLIE UN QUART DE MORTS EN MOINS.**
La bascule etait ecrite comme « `killer_victim_pairs` devient une vue sur
`match_kill_events_latest` ». Elle a ete IMPLEMENTEE, appliquee sur les bases reelles, puis
RETIREE — parce que le controle avant/apres a montre ce que personne n avait mesure. Oracle :
les evenements `death` de `highlight_events`, sur les 949 matchs a film de la base Infinite.

| source | morts | part de l oracle |
|---|---|---|
| API (`highlight_events`, `death`) | **100 266** | — |
| `killer_victim_pairs` (dedupliquee) | 98 662 | **98,4 %** |
| passe de film (`marche` + `scan`) | 74 569 | **74,4 %** |

La vue `_latest` retenant UNE passe par match, la passe de film — plus riche par LIGNE (source du
degat, assistant, parts) mais plus pauvre en COUVERTURE — serait devenue la generation servie et
aurait efface **25 697 morts** de la lecture. Sans erreur, sans compteur, sans qu un seul nom ni
un seul instant ne change : seulement moins de lignes. Le controle chiffre : 405 matchs sur 949
en perte, 133 886 couples distincts tombant a 107 533.

*Ce que ca apprend, et qui vaut plus que le correctif* : **le corollaire etait ECRIT dans le DDL
depuis la session 1** (« si les deux producteurs ecrivent le meme match, LE DERNIER GAGNE
ENTIEREMENT »), et il etait meme signale comme une contrainte d ordonnancement. Ce qui manquait
n etait pas l avertissement, c etait sa MAGNITUDE. Un risque documente sans chiffre se lit comme
un risque theorique.

**CE QUI A ETE LIVRE, ET QUI TIENT** :

1. **Le producteur live** (`persist/kill_events_credit.go`) — les deux producteurs de
   `match_kill_events` etaient HORS LIGNE, appeles par la seule sous-commande
   `backfill-killsource` : verifie par grep sur `internal/api/`, `internal/sync/v2/`,
   `cmd/server/`, `internal/service/` → **zero appel**. Tout match synchronise apres la derniere
   passe de backfill n avait donc AUCUNE ligne. Les couples que le pipeline rend deja
   (`Shared.KillerVictim`) sont desormais traduits en morts credit-seul et ecrits dans la MEME
   transaction que le match. UNE seule traduction dans le depot, empruntee par les deux
   ecrivains — flux primaire et completion combat.
2. **Title-agnostic par construction** : Halo Infinite y arrive par sa completion combat, Halo 5
   par son carnage natif. Ce n est pas cosmetique — mesure : `highlight_events` de Halo 5 ne
   porte AUCUN evenement kill/death (medal, assist, mode seulement), donc le producteur
   credit-seul de `killcollector`, qui lit cette table, n aurait JAMAIS couvert ce titre. Et
   Halo 5 pese **268 337 couples sur 2 754 matchs, dont ZERO doublon**.
3. **La reprise dedupliquee** (migration `shared_kill_events_from_pairs_v1`) : tout ce que porte
   l ancienne table entre dans la canonique, `SELECT DISTINCT`, une passe par match, en sautant
   les matchs deja couverts. Effet mesure : Infinite **0 ligne** (les 1 343 matchs etaient deja
   couverts), Halo 5 **268 337 lignes** — le titre passe de « la table n existe meme pas » a
   couverture complete.
4. **La preseance testee EN POSITIF** sur les voies de film (`persist.FilmReadPaths`), au lieu
   de la negation de sa propre voie. L ancienne forme (`read_path <> 'highlight-events'`) faisait
   passer TOUTE voie autre que la sienne pour un film : le jour ou une seconde voie credit est
   apparue — ce jour meme — elle aurait bloque le producteur live en silence.
5. **`v_killer_victim_full` supprimee** (cf. §2.2), et **`feature-flags.ts`** cote web, avec le
   plafond knip abaisse **de 29/90/86 a 0/0/0** : le compte reel etait deja a zero sur les trois
   axes, donc les plafonds laissaient rentrer 115 findings sans rien dire.
6. **Le resolveur d identite lit les DEUX sources.** `v_gamertag_lookup` a ete rebranche sur
   `match_kill_events_latest` — le 11e site, celui dont l oubli aurait fait retomber tous les
   adversaires sur « Joueur #### ». Mesure : la canonique SEULE coutait **4 gamertags sur
   18 219**. La jambe historique est conservee en UNION jusqu au retrait de la table : le
   resolveur rend exactement les memes 18 183 noms qu avant.

**LE PIEGE DE METHODE DE LA SESSION** : le decoupeur SQL de `internal/sync/schema.go` (`splitSQL`)
n est PAS conscient des commentaires — il coupe sur CHAQUE `;`, y compris dans un `--`. Un
point-virgule ajoute dans un commentaire du schema de base a fait echouer 13 tests avec
« empty query », a des kilometres de la cause. ⚠ A ne pas confondre avec le decoupeur du paquet
`migration`, lui conscient des commentaires (constat de la session 1) : **ce sont deux fonctions
differentes, et une seule des deux est sure**.

**CE QUE LE RETRAIT DE `killer_victim_pairs` ATTEND** — un critere, pas une date : que la passe
de film porte l INTEGRALITE du kill-feed. Le collecteur doit partir de la liste officielle des
morts et l ENRICHIR de ce qu il sait decoder, au lieu de publier la seule liste qu il sait
decoder. Mesurable : `lignes_passe_film / morts_api` >= le ratio de l ancienne table (98,4 %) sur
le meme perimetre. Tant que ce n est pas vrai, remplacer reviendrait a echanger 46,5 % de
doublons contre 25 % de morts manquantes — *et le doublon se voit, la mort manquante non*.

---

### 2026-08-02 — J4 session 4 : l exposition, et la porte qui n existait pas

**Perimetre traite** : phase 4 (capabilities par famille), plus les lots 1.2/1.3 et 3.1/3.2 du plan
de finalisation du rejeu. **Phase 4 : CLOSE au sens ou elle etait armable** — les trois familles ont
leur cle, la nouvelle a un consommateur teste, et la raison perimee du `not_exposed` de la precision
est corrigee. **Aucune surface produit n a ete ouverte**, et c est conforme : exposer une lecture
alimentee par une table dont la couverture est en arbitrage (74,4 % contre 98,4 %) ferait des pages
qui se vident. La bascule des lecteurs n a pas ete rouverte.

**CE QUI A ETE TROUVE EN POSANT LE GATE VISUEL, ET QUI VAUT LA SESSION.** Le lien vers le rejeu
n aurait **jamais** pu apparaitre. L outil hors ligne ecrit l artefact sous la forme COURTE du
`match_id` (`000d5950.json` — celle du film qu il vient de decoder) ; l application ne manipule que
la forme COMPLETE. Le service cherchait donc un fichier que personne n ecrit : `GetReplay` rendait
un 404 **sur un artefact present**, et `replay_available` aurait ete faux partout. Les 3 artefacts
du depot sont tous en forme courte — le defaut etait total, pas marginal.

*Ce que ca apprend* : le champ `replay_available` ne prouve rien tant qu on n a pas ouvert la page.
Le gate visuel n est pas un rituel de fin — c est le seul controle qui aurait pu voir ce defaut, et
il l a vu **avant** d etre joue, simplement en cherchant l URL a donner a l utilisateur.

**Correctif** : la troncature devient `title.FilmShortMatchID`, **une seule fois dans le depot** —
la copie de `sync/haloclient/local_film_cache.go` l appelle desormais. `ReplayArtifactPath`
normalise : les deux formes donnent le meme chemin. Garde-rail `TestUneSeuleTroncatureDeMatchID`
(scan d `internal/`, echoue a la 3e copie).

**LE RESTE, EN BREF** :

1. **Les grenades etaient nommees deux fois et differemment**, et c est mesure sur `000d5950` : le
   lancer publiait « Shock » quand le compteur porte de la meme fiche disait « Dynamo ». Le
   decodeur ne nomme plus rien (liste ORDONNEE de tags, sans libelle) ; le lancer publie son RANG,
   qui est un index dans la table du titre. Un index ne peut pas diverger de sa table.
2. **Les catalogues Halo sortent du code** vers `config/titles/halo_infinite/mappings/` : les noms
   d armes viennent de `weapon_names.toml` (bilingue, deja la), les rangs de grenade / capacites /
   effets de tir d un nouveau `replay_labels.toml`. Cote web, `shotEffects.ts` ne connait plus une
   seule arme Halo. **Trois libelles CHANGENT et ce sont des corrections** : l enumeration du
   decodeur nommait `0x9D6AAED2` « M41 SPNKr » (c est le SPNKr a combustible) et `0x3E070217`
   « Pulse Carbine » (c est la carabine Vestige).
3. **Les 3 artefacts reconstruits et compares champ par champ** : traces, tirs, inventaire,
   couverture, roster, geometrie et structure **identiques** ; les lancers identiques hors leur
   type ; la correspondance ancien nom -> nouveau rang exacte ; 39/39, 17/17 et 12/12 armes
   toujours nommees. Le seul identifiant sans effet de tir est la grenade Dynamo PORTEE — une
   grenade ne se tire pas.
4. **4 copies de `keep*OfPublishedTracks` factorisees** avec garde-rail et temoin de detection joue.

**CE QUI ATTEND L UTILISATEUR** : la revue visuelle du lot 1 (le lien, son absence, les familles
d effet de tir). Elle n est PAS declaree passee.

---

## CE QUI RESTE OUVERT, ET QUI N EMPECHE PAS DE BRANCHER

- **Les noms de vehicules** : 8 tags sur 14 nommables, et la racine de banque sonore ne suffit pas
  (demonstration : elle disait << Pelican >> pour un Falcon). Ils sortent << Autres >>, ce qui est
  honnete.
- **Le tir a la tete sur les kills au vehicule** : le jeu ne l affiche pas. Plafond de statut = la
  coherence interne. **Ne jamais le mettre dans une liste de verification a soumettre a l utilisateur.**
- **Les 206 tags sans nom** sur 468.
