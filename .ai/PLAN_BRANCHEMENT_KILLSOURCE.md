# PLAN DE BRANCHEMENT — killsource en production

> Ecrit le 2026-07-31. Le decodeur, les deux tables et leurs persisters sont ECRITS, TESTES et
> POUSSES, **sans aucun appelant**. Ce document dit comment on les branche.
>
> Contrat d execution : skill `plan-execution`. Ordre strict, une etape a la fois, gate verifie
> avant de passer a la suivante, statut ecrit pour chaque item a la cloture.

## CE QUI EXISTE DEJA, ET QU IL NE FAUT PAS REECRIRE

| Brique | Ou | Etat |
|---|---|---|
| Decodeur | `internal/games/halo_infinite/film/killsource` | livre, 30 ancres Theater, goldens |
| Pont de telechargement | `internal/sync/killsource_bridge.go` | ecrit, **aucun appelant** |
| Table des morts | `internal/migration/steps_shared_kill_events.go` | append-only + vue `_latest` |
| Table des tirs | `internal/migration/steps_shared_weapon_shots.go` | idem |
| Persisters | `internal/persist/{kill_events,weapon_shots}_persister.go` | INSERT-only, 0 allowlist ART |
| CLI de controle | `cmd/killsource` | `kills` / `json` / `sante` / `comparer` |

**Rien de tout cela n est appele.** Le branchement consiste a ecrire LE COLLECTEUR, et lui seul.

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

- [ ] `git fetch --unshallow` (le depot est en clone superficiel : aucun rebase n est possible avant)
- [ ] Creer `feat/killsource-prod` depuis un `main` A JOUR
- [ ] **Y porter le PRODUIT UNIQUEMENT** : le decodeur, les migrations, les persisters, le pont, le
      CLI. **PAS les 67 outils `cmd/tmp_*`** — ils restent sur la branche de recherche, ou le
      journal les cite comme commandes de reproduction.
- [ ] Rejouer la suite complete sur la branche neuve

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

- [ ] Ajouter une cle fine dans `config/titles/halo_infinite/mappings/capabilities.toml`
      (proposition : `film_kill_source`), absente de `halo_5`
- [ ] Le collecteur teste `HasCapability(...)`, **jamais `slug == "halo_infinite"`** — le ratchet
      `no_slug_comparison_test.go` l interdit
- [ ] Degradation : `ErrCapabilityNotSupported` -> le cycle continue sans decoder, aucun panic

### 1.3 Quand il tourne

**PAS au sync primaire** : le film n existe pas encore quand le match arrive. Le collecteur est une
passe SEPAREE, sur des matchs deja presents dans `match_registry`.

**LE COUT EST UNE CONTRAINTE DE CONCEPTION, PAS UN DETAIL** :

| Regime | Cout mesure |
|---|---|
| 4v4 (8-10 joueurs) | 8 a 30 s par film |
| **BTB (36 participants)** | **11 minutes** (mesure du 2026-07-31 sur `4f77afc1`) |

Consequences a respecter :
- [ ] **Un seul decodage a la fois dans un process** : les parametres de replication de `filmdec`
      sont des globaux de paquet ; `Decode` serialise deja par un verrou, ne pas contourner
- [ ] Le collecteur travaille en TACHE DE FOND, jamais dans le chemin d une requete HTTP
- [ ] Une limite de temps par match, et un compteur des abandons
- [ ] Journaliser `slog.InfoContext` a l entree et a la sortie avec `match_id` et la duree

### 1.4 Ce qu il ecrit

- [ ] Traduire `killsource.Kill` -> `persist.KillEventInsert` (l exemple est dans l en-tete de
      `killsource_bridge.go` ; **le suivre, il porte les pieges**)
- [ ] Ecriture via `persist.BatchBuilder.SetKillSource(...)` puis `Submit()` — **jamais un INSERT
      direct** (regle anti-ART, ADR 0019/0030)
- [ ] **Les trois etats de l assistant survivent jusqu en base** : `assist_known` FAUX = on ne sait
      pas, distinct de `assist_gamertag` NULL = pas d assistant. Les confondre publierait des faits
      jamais observes.
- [ ] **`null` n est jamais zero** sur les parts de degats ; aucun plafond a 100 (le decodeur sort
      des valeurs jusqu a 228, ce sont des donnees)
- [ ] Renseigner `decoder_rev` — il permet de RE-DECODER de facon ciblee au lieu de tout reprendre
- [ ] Publier les compteurs de sante en `expvar` (ADR 0009), dont **`assist_extra_count`** : c est
      le declencheur de migration vers une table fille si l hypothese << un seul assistant >> tombe

**GATE 1** : un test d integration qui decode un film de fixture, ecrit en base de test, et relit
par la vue `_latest` ; `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` ;
les trois garde-fous anti-ART verts ; `assist_extra_count` interrogeable en SQL.

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

- [ ] **949 films en cache**, 1 908 matchs au registre, 1 325 porteurs de paires
- [ ] Estimation : ~900 films 4v4 a 8-30 s = **2 h 30 a 7 h 30** ; les BTB a 11 min chacun sont a
      compter separement et a passer en dernier
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

## CE QUI RESTE OUVERT, ET QUI N EMPECHE PAS DE BRANCHER

- **Les noms de vehicules** : 8 tags sur 14 nommables, et la racine de banque sonore ne suffit pas
  (demonstration : elle disait << Pelican >> pour un Falcon). Ils sortent << Autres >>, ce qui est
  honnete.
- **Le tir a la tete sur les kills au vehicule** : le jeu ne l affiche pas. Plafond de statut = la
  coherence interne. **Ne jamais le mettre dans une liste de verification a soumettre a l utilisateur.**
- **Les 206 tags sans nom** sur 468.
