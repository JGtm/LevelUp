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

- [ ] **Traiter 4 et 5 dans le meme commit** (regle des <= 2 copies) et poser le garde-rail
- [ ] `v_killer_victim_full` NE SURVIT PAS : ses deux `LEFT JOIN` sont du travail mort a chaque
      chargement de vue match, et elle ne marche que par un renommage silencieux de DuckDB

### 2.3 Deux arbitrages DEJA TRANCHES, a ecrire dans le DDL

- [ ] **Les bots restent EXCLUS des agregats carriere.** La nouvelle table sait les representer,
      l ancienne non (zero ligne de bot en prod) : un compteur qui bouge a cause d une migration
      technique est une mauvaise surprise. Ils sont DANS la table, les lecteurs carriere filtrent.
      **Attention** : Q26 ne filtre PAS les bots aujourd hui, Q27 si — c est le piege de la bascule.
- [ ] **Le BTB reste INCLUS dans les cumuls, interdit ligne par ligne.** C est exactement ce que
      `publishable` veut dire : exact en agregat, faux individuellement.

**GATE 2** : pour chaque lecteur bascule, une requete AVANT/APRES sur le meme match rend le meme
resultat a la deduplication pres ; aucun lecteur ne casse en silence ; `go test ./internal/...`.

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

- [ ] ~~**949 films en cache**, 1 908 matchs au registre, 1 325 porteurs de paires~~ → re-mesure ci-dessus
- [ ] ~~Estimation : ~900 films 4v4 a 8-30 s = **2 h 30 a 7 h 30** ; les BTB a 11 min chacun~~ → re-mesure ci-dessus
- [ ] Sous-commande dediee du CLI principal (`levelup`), pas un script jetable
- [ ] Reprenable : `decoder_rev` + la vue par PASSE permettent de rejouer sans tout refaire
- [ ] **Les films Theater EXPIRENT cote serveur** : ce qui n est pas en cache est perdu
      definitivement. Au moins **28 % des matchs** n auront jamais de ligne issue d un film — d ou
      `source_tag` NULLABLE et le chemin `highlight_events` comme SECOND producteur.
- [ ] Prevenir avant toute execution sur le VPS de prod

**GATE 3** : le backfill tourne sur 20 films, les compteurs de sante sont dans le domaine, et un
`SELECT` de controle montre que les agregats carriere ont DEGONFLE du facteur attendu.

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

### 4.3 Capabilities

- [ ] Une cle par famille de donnee, pas une seule fourre-tout : le kill enrichi, les tirs par arme,
      la precision. Un titre peut avoir l un sans les autres.
- [ ] Cote web : `useFieldLabel()` / `useOutcomeLabel()`, jamais de libelle en dur ; strings FR ET EN
- [ ] Couleurs par tokens semantiques uniquement

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

## CE QUI RESTE OUVERT, ET QUI N EMPECHE PAS DE BRANCHER

- **Les noms de vehicules** : 8 tags sur 14 nommables, et la racine de banque sonore ne suffit pas
  (demonstration : elle disait << Pelican >> pour un Falcon). Ils sortent << Autres >>, ce qui est
  honnete.
- **Le tir a la tete sur les kills au vehicule** : le jeu ne l affiche pas. Plafond de statut = la
  coherence interne. **Ne jamais le mettre dans une liste de verification a soumettre a l utilisateur.**
- **Les 206 tags sans nom** sur 468.
