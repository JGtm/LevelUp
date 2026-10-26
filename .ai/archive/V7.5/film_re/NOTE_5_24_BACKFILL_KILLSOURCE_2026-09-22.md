# NOTE 5.24 — `backfill-killsource` : plus rapide, observable, reprenable (2026-09-22)

Branche `feat/decfilm-74`, base `f3d52514b` (schema 68). **Aucun changement de decodeur, aucun
changement de ce qui est ecrit en base** : `SchemaVersion`, `grammar.Rev`, `facts.Rev` et
`IsolationDecoderRev` sont INCHANGES, aucun octet de `film/internal/`. Le lot change COMMENT la
passe tourne, pas CE QU ELLE PRODUIT — et deux tests d egalite le prouvent ligne a ligne.

Detail complet, tableaux et journal des gates : `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, section
« Post-chantier — lot 5.24 », §4 et §5.

## 1. Ce que la mesure a dit, et ce qu elle a tranche

`internal/sync/killcollector/backfill_cout_integration_test.go` chronometre les sept etapes d un
match, chacune par UN appel a la fonction de production, sur un banc de 180 000 lignes de
`match_kill_events` + 50 000 couples et sur les films reels du cache.

| part du temps | echantillon large (4 films, 61,6 s) | 9 petits films (10,7 s) |
|---|---|---|
| DECODAGE | **69,3 %** | **46,6 %** |
| annexes (tirs, positions, touches — memes chunks rebalayes) | **29,8 %** | **46,7 %** |
| roster (`v_gamertag_lookup` par match) | 0,3 % | 4,6 % |
| ecriture | 0,6 % | 2,1 % |
| assemblage | 0,0 % | 0,1 % |
| chargement des chunks, resolution de carte | **0,0 %** | **0,0 %** |

**Verdict : 93 a 99 % du temps d un film est du CPU hors base.** Le parallelisme est la seule
reponse qui porte ; un lot d ecriture groupee n aurait rien rapporte. Pic memoire re-mesure sur
le pire film du corpus (`1c4c63c2`, 69 chunks, 92,2 Mio) : **422 Mio de HeapInuse** — l en-tete
de la commande (« largement sous le gibioctet », 2026-08-24) tient, et c est ce chiffre qui
autorise N ouvriers.

## 2. Le verrou qui interdisait le parallelisme n existait plus

`collector.go` et `CollectMatches` affirmaient que « les parametres de replication de `grammar`
sont des GLOBAUX DE PAQUET » et que « `killsource.Decode` serialise par un verrou de paquet ».
**Les deux affirmations etaient perimees** : la cloture M3 du chantier decodeur (ADR 0034,
2026-09-17) a DEPENSE le profil — `ProfilDeBalayage` voyage en argument jusqu a `calibrate` et
`runWalk` — et le dernier reglage global (`SetInferResyncTargets`) a ete supprime au lot E.2 du
2026-09-05. Balayage du 2026-09-22 : aucun `Set*` de paquet dans `grammar`, aucun `var` mutable
de paquet dans `grammar` ni `killsource` hors tables constantes, `registryWarned` est une
`sync.Map`, le compteur de replis porte son verrou. Les deux commentaires sont corriges avec
leur date et leur cause. Une troisieme copie du meme texte perime subsiste dans
`facts/killsource/doc.go` — **D2 (5.24)** au §4, hors perimetre.

**La preuve n est pas le raisonnement** : `TestOuvriers_MemesLignesQuUnSeulOuvrier` ecrit les
memes films avec 1 puis 3 ouvriers et compare les cinq vues `_latest` ligne a ligne.

## 3. Ce qui est livre

| point | piece | effet mesure |
|---|---|---|
| 5.24.1 | `backfill_cout_integration_test.go` | la decomposition ci-dessus |
| 5.24.2 | `collector_ouvriers.go`, `porte_de_la_base.go`, `--workers` | **x2,70 a x2,78** a 3 ouvriers, **939 lignes identiques** sur 5 vues |
| 5.24.2 | `SharedRoster.AvecAnnuaireDePasse` (D1 du 5.12 ferme) | roster **50,7 ms -> 4,0 ms** par match (x12,5) |
| 5.24.3 | `cmd_backfill_killsource_etat.go`, `..._status.go` | fichier d etat par film, ligne tous les 25 films OU 60 s, `--status` |
| 5.24.4 | `cmd_backfill_killsource_arret.go`, `AvecArretDoux` | coupure a 4/6, relance **4 sautes / 2 ecrits**, **249 lignes identiques** |

## 4. Les trois decisions qui meritent d etre retenues

**UN JETON, PAS UN GOROUTINE ECRIVAIN.** La forme canonique (un goroutine qui possede la base,
des messages) a ete ecartee SUR PIECE : le contrat d ecriture du collecteur est un LEASE
(`acquireShared` rend `(db, release, err)` et l appelant ecrit ENTRE les deux, sur **sept
sites**). Le convertir en messages exigerait de decouper `collect`, `collectPositions` et
`collectHits` en « phase qui calcule » / « phase qui ecrit » — c est-a-dire de reecrire ce que la
passe FAIT, ce que le perimetre interdisait. Le jeton unique (`PorteDeLaBase`, canal borne a 1)
donne le MEME invariant — au plus un goroutine parle a la base, lectures comprises — sans toucher
une ligne de ce qui est ecrit. Les sept leases ont ete verifies DISJOINTS, et l attente est
bornee a 5 min avec un message qui nomme l imbrication : un futur « programme qui ne finit
jamais » devient une erreur lisible.

**L ETA COMPTE DES CHUNKS, PAS DES FILMS.** Les gros films passent en dernier (c est ce qui fait
qu une interruption laisse un resultat presque complet). Un reste compte en NOMBRE de films
annoncerait une fin proche **juste avant la queue la plus chere** — il mentirait exactement au
moment ou on le consulte le plus. Le reste vaut (chunks restants x cout par chunk **mesure depuis
le debut de la passe**) / ouvriers.

**DEUX ARRETS, ET LES CONFONDRE PERD DU TRAVAIL.** L arret DUR annule le contexte de travail et
coupe le film en cours ; l arret DOUX cesse de DISTRIBUER et laisse aller au bout ce qui est en
vol, ecriture comprise. Avec N ouvriers, un Ctrl-C en arret dur jetterait N decodages entames
pour ne rien gagner — la reprise se decidant de toute facon en base. L arret doux **ne protege
pas la base** (une ecriture interrompue est rollback, jamais a moitie) : il protege le temps de
calcul. Le releveur de signaux est RENDU des le premier signal, donc un second Ctrl-C tue.

## 4 bis. Un defaut trouve en LANCANT la commande, pas en la relisant

`go run ./cmd/levelup backfill-killsource --status` — qui n ouvre aucune base et sort en 0 —
terminait par « **arret demande : plus aucun film n est distribue** ». Le message le plus
alarmant de la commande s affichait a la fin de TOUTE passe reussie. Cause :
`signal.NotifyContext` annule son contexte **par les deux chemins** (le signal ET l appel a
`stop`), donc le `defer` de fin nominale reveillait le goroutine d annonce. Corrige dans le lot
(canal de signal dedie, `stop` idempotente) avec son test de non-regression.

Aucune relecture de diff n aurait trouve ce defaut : il ne se voit qu en EXECUTANT.

## 4 ter. La revue adversariale — 1 P1 trouve DEUX FOIS

Deux relecteurs aveugles, contextes frais, lentilles L1 (anti-ART) et L6 (couverture des tests).

**L1 : aucun defaut recevable sur les ecritures**, et 14 conditions verifiees qui tiennent —
les 7 sites de lease exhaustifs et tous gardes, aucune imbrication de jeton, aucune fuite,
`decode_pass` unique par passe, les six vues `_latest` partitionnees par `match_id`, aucun etat
mutable de paquet sur le chemin de decodage (la justification centrale du lot, re-verifiee par
un tiers).

**Le meme P1 trouve par les DEUX** : la passe CREDIT recevait le contexte d ARRET, deja annule
apres un Ctrl-C pendant la passe des films. `database/sql` teste `ctx.Done()` avant de prendre
une connexion, donc `matchsDuRegistre` rendait `context canceled` sans rien executer, et la
commande sortait AVANT de fermer l etat. **Les trois promesses de l arret tombaient sur le
chemin nominal** : etat fige en phase « films » (`--status` annoncant « probablement morte » au
lieu de « INTERROMPUE »), code de sortie 1 au lieu de 130, message d erreur au lieu des
instructions de reprise. Corrige — contexte de travail par `context.WithoutCancel` et arret par
`CreditCollector.AvecArretDoux`, applique entre deux matchs —, avec un test dont la MUTATION a
ete verifiee rouge.

Cinq P2 dans le perimetre corriges avec leurs tests : `--credit-only` sans fichier d etat ;
`--online --workers N` accepte puis ignore ; l arret doux de la boucle en serie (`--workers 1`)
non couvert ; un test de porte sensible a l ordonnancement ; la marge du test de reprise.

Deux P2 consignes au paragraphe 5 : **D4** (aucun job CI ne tient les trois affirmations
centrales — la CI n a pas les films) et **D5** (quatre chemins de cablage CLI sans test de
mutation).

## 5. Ce qui reste ouvert

- **D1 (5.24)** — les dix films les moins chers du cache ne produisent rien neuf fois sur dix
  (5 sans identite au fil des morts, 3 sans chunk HIGHLIGHT). Cout borne, remede = un marqueur
  de registre terminal. Non traite.
- **D2 (5.24)** — l en-tete de `facts/killsource/doc.go` affirme encore le verrou de paquet
  disparu. Correction de 7 lignes, a prendre par le premier lot qui rouvrira ce paquet.
- **D4 (5.24)** — ⚠ **aucun job CI ne joue les trois tests qui tiennent les affirmations du
  lot** : `ci.yml` lance `go test -tags=integration ./...` sans `KILLSOURCE_FIXTURES`, donc
  l egalite 1-contre-N, l egalite annuaire-contre-jointure et la reprise se SAUTENT toutes les
  trois. Elles sont jouees LOCALEMENT a chaque cloture et leurs resultats sont colles au §5 du
  plan. Y remedier est une decision d infrastructure (la CI n a pas les films, 107 Mo non
  versionnes), pas un geste de ce lot.
- **D5 (5.24)** — quatre chemins de cablage CLI sans test de mutation : retirer
  `.AvecArretDoux(ctx)` ou `.AvecObservateur(suivi)` de la construction de la passe des films,
  le pont `AvecProgression` de la passe credit, ou le compteur `SansFilmEnCache`, ne fait rougir
  AUCUN test. L extraction de `jouerLesDeuxPasses` faite au tour de revue rend le remede
  possible ; c est un lot a part.
- **D3 (5.24)** — ⚠ **le seul rouge que le corpus reel declenche** :
  `TestKillSourceFaitsDIsolementFilmReel` assert `named_by NOT IN ('death','closure')` alors que
  `persist` en declare SEPT depuis le 2026-09-08 (registre d identite). 99 vies sur 99 « hors
  enum » sur `000d5950`. Le producteur est juste, c est l assertion qui est fausse ; le test se
  SAUTE sans `KILLSOURCE_FIXTURES`, donc aucun gate du depot ne l a jamais vu. Remede de trois
  lignes (employer les constantes `persist.NommePar*`). Hors perimetre du lot — **a prendre
  avant le prochain backfill du parc**.

## 6. Ce que le pilote peut lancer

```bash
# la passe, avec son etat lisible depuis un autre terminal
go run ./apps/go-api/cmd/levelup backfill-killsource --dry-run
go run ./apps/go-api/cmd/levelup backfill-killsource
go run ./apps/go-api/cmd/levelup backfill-killsource --status   # second terminal
```

⚠ **Serveur arrete** (un seul writer, ADR 0013). La mesure du parc entier — combien de temps la
campagne complete prend reellement a 3 ouvriers sur 1 612 films — n a **pas** ete faite : la
machine est partagee, et c est au pilote de la lancer.
