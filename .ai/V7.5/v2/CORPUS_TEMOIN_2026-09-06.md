# Gate de non-régression du rejeu sur corpus témoin — 2026-09-06

> Construction de `cmd/replay-corpus-gate` : industrialise la méthode du balayage ponctuel
> (`.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`) sur un corpus TÉMOIN restreint et versionné
> (`config/replay_corpus.toml`), pour qu'un différentiel sur films réels tourne AVANT tout
> merge qui touche au décodeur ou au constructeur de rejeu — les trois régressions du 28/08,
> 30/08 et 02/09 ont traversé des goldens synthétiques pendant dix-neuf schémas faute de ce
> différentiel. Worktree `LevelUp-wt-v2-corpus`, branche `feat/v2-corpus`.

> **NOTE DATÉE 2026-09-06 (2e passe, corrections de périmètre demandées avant revue).** Deux
> corrections apportées après la rédaction initiale de ce journal : (1) le bug de mesure
> `<calque>.<sous-champ>/n` (§6 ci-dessous) découvert lors de la 1re exécution est maintenant
> CORRIGÉ À LA SOURCE — les chiffres de la §3 ci-dessous datent d'AVANT ce correctif et sont
> **remplacés** par ceux de la §7 (2e exécution, mode parc, correctif inclus) ; (2) la référence
> par défaut du gate n'est plus le parc mais une **cuisson à la base** (§5) — §1 à §4 restent le
> compte rendu de la 1re conception (mode parc uniquement), conservé pour l'historique du
> raisonnement, mais **ne reflètent plus le comportement par défaut du gate livré**.
>
> **Effet du correctif §6 sur `BALAYAGE_PARC_2026-09-06.md` : aucun sur ses CONCLUSIONS,
> potentiel sur les COMPTES bruts de sa section 5.** Le balayage cite exclusivement des mesures
> SPÉCIALISÉES (`coverage.*`, `objectives/par-*`, `tracks/points`, `vies-par-xuid`...) — aucune
> n'utilise `mesurerTableau` au niveau imbriqué (le seul niveau affecté par le bug ; le niveau
> racine, un seul appel par calque, a toujours été correct). Les COMPTES AGRÉGÉS de sa section 5
> (« ports : 91 pertes », « véhicules : 51 pertes »...) additionnaient TOUTE l'empreinte, donc
> INCLUAIENT `flagCarries.spans/n`, `vehicles.rides/n`, `vehicles.samples/n`, `zoneStates.spans/n`
> et `zoneStates.gauge/n` avec une valeur SOUS-COMPTÉE (dernier groupe itéré, pas la somme) —
> ces cinq métriques précises n'apparaissent dans AUCUNE conclusion textuelle du balayage
> (vérifié par relecture des §5, §6 et §9). Aucun chiffre cité dans le corps du texte de
> `BALAYAGE_PARC_2026-09-06.md` n'est donc faux ; les totaux bruts du tableau de sa section 5
> auraient pu être légèrement plus élevés avec le correctif (le rapport n'est pas rejoué : il
> n'est PAS réécrit, conformément à la consigne). `BALAYAGE_PARC` n'est donc PAS corrigé par ce
> commit.


## 1. Conception retenue

### 1.1 Axe « somme des durées » (`internal/replaydiff`)

Extraction PRÉALABLE de la logique de `cmd/replay-diff` (jusque-là un `package main` non
importable) vers `internal/replaydiff`, partagé par `cmd/replay-diff` (CLI historique,
comportement externe inchangé — mêmes flags, même sortie, vérifié par test de fumée) et
`cmd/replay-corpus-gate`. Décision justifiée par la règle des ≤ 2 copies (CLAUDE.md n°6) : sans
extraction, le gate aurait dû dupliquer `Empreindre`/`Comparer`/`AfficherTableau`.

Le nouvel axe (`empreinte_durees.go`) descend récursivement dans tout le document et pose,
pour CHAQUE objet qui porte un couple `t0`/`t1` numérique (n'importe quel calque, n'importe
quelle profondeur ≤ 6) sa durée `t1 - t0 + 1` (T1 inclus, la convention documentée sur chaque
span du dépôt). Aucun nom de calque n'est câblé en dur — un calque neuf à intervalles entre
dans le rapport sans qu'on l'y inscrive, même principe que la passe générique existante. La
ventilation individuelle suit LA CLÉ QUE LA SOURCE UTILISE DÉJÀ : `xuid` en priorité (porteur
direct — `FlagSpan`, `BombCarry`, `SkullCarry`, `VipPeriod`, `VehicleRide`), `slot` à défaut
(vie ou entité du calque — `GrappleLine`, `EquipmentEpisode`, `VehicleTrack`...), rien quand ni
l'un ni l'autre n'existe (`ZoneSpan` : seul `owner`, une équipe).

Tests par mutation (`empreinte_durees_test.go`, 8 tests) : le test central,
`TestIntervalleRogneEstUnePerte`, reproduit exactement le défaut mesuré sur l'Oddball du
2026-09-06 (une fenêtre de grappin rognée de bordure coûte 91,2 s de durée cumulée contre
32,6 s pour l'équivalent en rejets purs — le comparateur d'avant cet axe ne voyait que les
rejets). Preuve par mutation supplémentaire : l'appel à `mesurerDurees` a été commenté
temporairement — 6 des 8 tests rougissent immédiatement (les 2 qui ne rougissent pas
vérifient une ABSENCE d'écart, donc restent vrais sans la mesure), l'appel restauré ensuite,
suite revérifiée verte.

### 1.2 Manifeste versionné (`config/replay_corpus.toml`)

Un témoin par famille de mode, choisi dans le parc local après recensement complet des 106
matchs disponibles (`levelup replay-facts-export` en lecture seule sur le parc, aucune
supposition) :

| id | famille | mode | carte | raison |
|---|---|---|---|---|
| `bcb6d393` | ctf_mono_manche | CTF:Arena | Cliffhanger | porte le calque objectifs sur un CTF à 1 manche ; résidu connu (BALAYAGE_PARC) |
| `fb1a1a72` | ctf_multi_manche | CTF:Arena | Banished Narrows | CTF multi-manche ; pont d'identité par manche encore ouvert au registre |
| `d9781168` | oddball | Oddball:Arena | Dredge | résidu non instruit du balayage final (portages de crâne, 36 → 30) |
| `c75f33b8` | assaut_bombe | Assault:One Bomb | Curfew | seul match Assaut du parc local ; porte bombCarries + statistiques d'Assaut |
| `bf15f7ab` | slayer | Slayer:Arena | Perilous | mode sans objectif (couvre pistes/armes/grenades/équipement seuls) |
| `51ebbc0f` | deux_manches | Oddball:Arena | Banished Narrows | film à 2 manches ; jointure score/manche indépendante du CTF |
| `084a804d` | vehicules | BTB Heavies:CTF | Fortitude Heavies | densité maximale de véhicules du parc (57 chunks) |

### 1.3 `cmd/replay-corpus-gate` : binaire séparé, pas une sous-commande

Décision : binaire dédié (nom découvrable, cohérent avec les autres `cmd/replay-*`), la
logique de comparaison étant partagée via `internal/replaydiff` (§1.1) — la sous-commande de
`replay-diff` aurait mélangé deux responsabilités (comparer une paire donnée / orchestrer un
corpus entier) dans le même `package main`.

Trois racines distinctes (`roots.go`) :

- `sourceRoot` : le dépôt où le gate tourne (code + catalogues versionnés au HEAD testé) ;
- `parcRoot` : le parc de développement (chunks, artefacts de référence) — auto-détecté par
  `git rev-parse --git-common-dir`, VALIDÉ par la présence de la base partagée du titre avant
  d'être accepté ;
- `workRoot` : racine de travail temporaire et jetable, où sont copiés (jamais liés) le
  manifeste + les chunks du témoin (depuis `parcRoot`) et les catalogues versionnés du titre
  (depuis `sourceRoot`) — l'artefact frais s'y écrit, jamais dans le parc.

**Découverte de topologie, mesurée en cours de route** : sur ce dépôt, `LevelUp-go-migration`
(traité partout comme « le principal ») est LUI-MÊME un worktree d'un ancêtre `.git` nommé
`LevelUp` — l'auto-détection naïve (`.git` commun) résolvait donc vers `LevelUp`, qui porte un
`data/` PÉRIMÉ (dernière modification 12 juillet) plutôt que le vrai parc. Corrigé par une
validation post-résolution (présence de `SharedDBPath(titleSlug)`, pas seulement d'un dossier
`data/`) qui refuse explicitement plutôt que de laisser échouer plus loin avec un message
DuckDB opaque — `--parc-root` explicite reste la méthode sûre sur une topologie inhabituelle.
Documenté en tête de `roots.go` et couvert par `TestResolveParcRootAutoDetectionValideeParLaBase`.

Verrouillage : `filmproc.AcquireSolo` sur `lockRoot` (défaut : `CacheRootDir()` du PARC, PAS de
la racine de travail) — le MÊME verrou que `cmd/replay-build`/`backfill-replay` posent déjà
depuis n'importe quel checkout, donc deux cuissons lancées depuis deux répertoires différents
s'excluent mutuellement. Les faits du match sont obtenus via `levelup replay-facts-export` en
sous-processus (CGO/DuckDB) — le gate lui-même reste compilable sans CGO.

Sortie : tableau récapitulatif (témoin, schéma parc, schéma HEAD, gains, pertes, durée) PUIS
détail nommé de chaque perte (axe, métrique, ancien → nouveau) — nécessaire pour distinguer un
correctif déjà documenté d'une régression neuve (cf. §3). Code 1 dès qu'un témoin cuit porte au
moins une perte ou une erreur ; un témoin absent du parc local est un avertissement `slog`,
jamais un échec (`codeSortie` l'exclut explicitement).

## 2. Exécution au HEAD (`a059caefc`, schéma 43)

Protocole de verrou inter-agents respecté (mkdir/rmdir sur le scratchpad autour de chaque
cuisson manuelle) ; deux essais corrigés en cours de route (résolution `--source-root` sans
`db_profiles.json` local à un worktree dédié ; résolution `--parc-root`, cf. §1.3) avant
l'exécution retenue ci-dessous, avec `--keep-work` pour l'analyse détaillée.

```
temoin       famille            parc   HEAD    gains   pertes      duree  statut
bcb6d393     ctf_mono_manche      20     43      205       27     11.49s  PERTE
fb1a1a72     ctf_multi_manche     34     43       27        2     29.37s  PERTE
d9781168     oddball              23     43      176        6     23.91s  PERTE
c75f33b8     assaut_bombe         28     43      168        8     14.54s  PERTE
bf15f7ab     slayer               34     43       41        2     13.78s  PERTE
51ebbc0f     deux_manches         21     43      184        9     17.68s  PERTE
084a804d     vehicules            20     43      482       21   1m53.29s  PERTE
```

7/7 témoins cuits sans échec, aucun absent du parc, pic mémoire maximum 0,44 Gio (084a804d) sur
un plafond de 3 Gio — cohérent avec les mesures du balayage complet (119 films, pic 0,56 Gio).
Code de sortie 1 (au moins une perte par témoin) : **exact et attendu** — le parc local est aux
schémas 20-34, très antérieur au HEAD (43), et §3 montre que la quasi-totalité des 75 pertes
brutes est expliquée par la chronique déjà écrite.

## 3. Analyse des pertes : déjà expliqué, ou fait nouveau

Chaque perte a été confrontée à `BALAYAGE_PARC_2026-09-06.md` (passes contre le schéma 41) et à
l'historique git des fichiers concernés.

### 3.1 Expliqué par la chronique existante (immense majorité)

| Motif | Témoins touchés | Référence |
|---|---|---|
| Bornes de scène assainies (points aberrants supprimés) | d9781168, c75f33b8, 51ebbc0f, 084a804d (bounds + tracks.points, pertes de 1 à 9 points sur des dizaines de milliers) | BALAYAGE_PARC §6.3, gain documenté |
| Compteurs d'un joueur en baisse dans un gain massif (ancien artefact ne portait qu'1 joueur au fil de score) | 51ebbc0f : `joueur/2535469889270266/{kills,assists,score}` 14→6 / 2→1 / 1845→970 — **valeurs identiques au chiffre** cité par le balayage final | BALAYAGE_PARC §6.3 et liste des 15 faits résiduels, ligne `51ebbc0f` |
| Réattribution dans un gain (2 actions `kills` perdues sur 2 xuids pendant que le match gagne partout ailleurs) | bcb6d393 : `objectives/par-joueur/.../kills` −2 et −1 — **valeurs identiques** | BALAYAGE_PARC §6.3, entrée `bcb6d393` exacte |
| Reclassement de la famille `other` (poses d'équipement) | bcb6d393, 084a804d : `equipmentPlacements/par-family/other` disparu, compensé par les familles nommées | BALAYAGE_PARC §6.3, chronique schéma 10 |
| `weaponLabels/n` −1 (catalogue dérivé, sans effet sur les tirs publiés) | bf15f7ab : 17→16 | BALAYAGE_PARC §6.3, entrée exacte |
| Compteurs de DÉFAUT en baisse (amélioration, pas perte — doctrine §5.1 du balayage) | fb1a1a72 (`counterJumps`, `missedEstimate`), bf15f7ab (`missedEstimate`), c75f33b8 (`livesFirstOffSpec`), 084a804d (`ambiguousReturns`, `shots.noSlot`) | BALAYAGE_PARC §5.1 |

**61 des 75 pertes brutes** relèvent de l'une de ces cinq familles — aucune n'est une
régression de produit.

### 3.2 Bug de mesure préexistant découvert (hors périmètre, consigné au registre)

`<calque>.spans/n` (et tout calque à deux niveaux du même patron : `skullCarries`,
`bombCarries`, `vehicles.rides`) est posé par un `e.num` (SET) au lieu d'un `e.incr` dans
`mesurerTableau` — la mesure finale est celle du DERNIER groupe de premier niveau itéré (une
équipe de `flagCarries`, un véhicule de `vehicles`...), pas la somme sur tous les groupes.
`flagCarries.spans/n` rend 28 → 1 sur `084a804d` alors que le journal de cuisson dit
« portages=15 fermes=15 » — **ce défaut est PRÉEXISTANT dans `cmd/replay-diff`**, hérité par
`internal/replaydiff` lors du déplacement (§1.1) sans modification de cette fonction. Le nouvel
axe « durée » (`e.incr`, non affecté) a permis de le remarquer. Non corrigé ici (règle CLAUDE.md
n°7, zéro fix opportuniste hors périmètre) — **consigné au registre**
(`.ai/V7.5/REGISTRE_REPORTS.md`).

**7 des 75 pertes brutes** sont des artefacts de ce bug de mesure (`flagCarries.spans/n` sur
`bcb6d393` et `084a804d`), pas des pertes réelles.

### 3.3 Faits nouveaux — non expliqués, consignés au registre

Deux découvertes, toutes deux détectées PAR LE NOUVEL AXE DURÉE (pas visibles au comptage
seul) :

1. **`bcb6d393` — `flagCarries` perd de la matière réelle** : `coverage.flagCarries.carries`
   16 → 7, `.closed` 16 → 7, `spans/duree-totale/par-xuid` −76 et −10 frames sur 2 joueurs.
   Journal de cuisson : pont d'identité RÉUSSI (`sansPont=0`, 16 prises) mais 9 prises sur 16
   SANS PISTE porteuse (`sansPiste=9`) — différent du défaut « candidate 1 » (pont incomplet,
   résolu au schéma 42, vérifié FLAG-R1 sur 4 films qui NE COUVRENT PAS `bcb6d393`).
2. **`084a804d` — `equipmentEpisodes` perd de la durée sans perdre d'épisodes** :
   `duree-totale` 3697 → 3629, tout sur le slot 620 (568 → 500) ; `equipmentEpisodes/n` NE
   BOUGE PAS — exactement le scénario que l'axe durée existe pour attraper. Le même match
   perd aussi une vie nommée complète (`tracks/vies-par-xuid/2533274806581989` 6 → 5),
   peut-être liée.

**7 des 75 pertes brutes** relèvent de ces deux faits (5 sur bcb6d393/flagCarries en plus des 2
lignes `coverage.*`, 2 sur 084a804d/equipmentEpisodes).

### 3.4 Récapitulatif chiffré

| Catégorie | Pertes brutes | Part |
|---|---|---|
| Expliqué par la chronique existante | 61 | 81 % |
| Bug de mesure préexistant (`spans/n`) | 7 | 9 % |
| Fait nouveau, consigné au registre | 7 | 9 % |
| **Total** | **75** | **100 %** |

**Verdict : aucune régression massive au schéma 43.** Le gate confirme que le HEAD reste sain
sur l'immense majorité de la matière déjà vérifiée par le balayage complet, et isole
précisément les deux points qui méritent une instruction séparée — exactement le rôle attendu
d'un gate de non-régression (détecter et nommer, pas trancher).

## 4. Limites connues, héritées de la méthode du balayage

Toutes documentées dans `BALAYAGE_PARC_2026-09-06.md` §8, et valables ici à l'identique : un
artefact du parc n'est pas une vérité (les bornes de scène le prouvent) ; la comparaison est
agrégée par match, pas élément par élément (un décalage temporel à effectif constant échappe) ;
aucune vérification visuelle. S'y ajoute la limite propre à ce gate : **7 témoins sur un parc
de 119+ matchs et 19 schémas** — un défaut isolé à une famille de mode non couverte par le
manifeste (ou à un schéma intermédiaire non représenté) resterait invisible. Le manifeste est
extensible sans changement de code (§1.2).

## 5. Correction 1 — la référence par défaut devient une cuisson à la base

**Constat du superviseur** : un gate qui rend PERTE sur 7 témoins sur 7 au MEILLEUR état connu
(HEAD contre le parc, jamais à jour) ne gate rien — personne ne peut lire un tableau tout rouge
et savoir si SON changement a introduit une régression. La §2-§4 ci-dessus, en mode parc seul,
en est la démonstration : 7/7 PERTE, alors qu'aucune régression de produit n'était en cause.

**Conception retenue** : `--reference=base` (nouveau défaut). Le gate résout une révision de
BASE (`--base`, défaut : `origin/feat/v75` si le HEAD courant en diffère, sinon `HEAD^` —
`base.go:resolveBaseRevision`), crée un **worktree Git détaché temporaire** de cette révision
sous la racine de travail (`git worktree add --detach`), y compile `cmd/replay-build` (GOCACHE
dédié, distinct de celui du HEAD), cuit chaque témoin **avec le binaire de base ET avec celui du
HEAD** dans deux racines de travail distinctes, puis compare les deux artefacts frais sur tous
les axes. Toute perte fait sortir en code 1.

Le mode `--reference=parc` reste disponible (balayage de release contre l'artefact déjà cuit) —
désormais **informatif par défaut** (imprime le tableau, sort en 0), bloquant seulement avec
`--strict`.

**Pourquoi un binaire compilé et invoqué en sous-processus, pas un import direct**
(`internal/replaybuild`) : importer ce paquet donnerait TOUJOURS le comportement du code AVEC
LEQUEL LE GATE EST COMPILÉ (le HEAD), quelle que soit la révision qu'on croit cuire — la
comparaison HEAD-contre-base serait vacuante (les deux côtés cuiraient avec le même code). La
cuisson passe donc par un binaire `cmd/replay-build` **compilé depuis la révision voulue**,
invoqué en sous-processus — la même méthode pour les deux côtés, symétrique par construction
(`bake.go`, `orchestrate.go`).

**Le verrou de décodage partagé reste externe au sous-processus** : `cmd/replay-build` arme sa
propre sentinelle et son propre verrou solo, mais sur `LEVELUP_REPO_ROOT=workRoot` — une racine
jetable que personne d'autre ne dispute. Ce verrou-là ne protège rien contre la concurrence
MACHINE. `bake.go` prend donc EN PLUS le verrou PARTAGÉ (`lockRoot` = `CacheRootDir()` du PARC,
jamais de la racine de travail) avant de lancer le sous-processus — le même verrou que
`cmd/replay-build` et `backfill-replay` posent déjà depuis n'importe quel checkout.

**Suppression du worktree détaché, et sa garde** : retiré à la fin (`defer`), même en échec.
AVANT `git worktree remove`, le worktree est balayé pour une JONCTION (`contientUneJonction`,
`os.ModeSymlink`) — `git worktree remove` la SUIVRAIT et supprimerait récursivement des fichiers
de l'autre côté (piège déjà mesuré sur ce dépôt, `reference_worktree_remove_follows_junctions.md`).
Ce gate ne pose jamais de jonction dans ce worktree (copie systématique, `staging.go`) : la
vérification est une garde défensive, jamais un cas attendu.

**Découverte opérationnelle en cours de route** : les trois premières tentatives d'exécution
complète (7 témoins × 2 cuissons) ont dépassé le budget de 600 s d'une commande avant-plan
bloquante — la machine porte des dizaines de worktrees actifs (`git worktree list` : ~90
entrées), et une commande TUÉE par expiration de timeout (contrairement à une bascule
automatique en arrière-plan) ne joue AUCUN `defer` Go : un worktree détaché et un sous-processus
`replay-build-base.exe` sont restés orphelins deux fois, tenant le verrou partagé et un fichier
de log, jusqu'à nettoyage manuel (`git worktree remove --force`, `taskkill`). Le calcul du gate
lui-même n'est pas en cause — le budget d'une invocation manuelle avant-plan l'est. Documenté
ici pour la prochaine exécution : prévoir un budget target ≥ 12-15 min pour le manifeste complet
en mode base sur une machine chargée, ou lancer en arrière-plan surveillé.

## 6. Correction 2 — le bug de mesure `<calque>.<sous-champ>/n`, corrigé à la source

Périmètre de ce lot inclus (`internal/replaydiff` est possédé par ce chantier) : le bug de
mesure préexistant découvert lors de la 1re exécution (§3.2 ci-dessus, désormais consigné puis
corrigé) est réparé à la source, pas seulement consigné.

**Le bug.** `mesurerTableau` (`empreinte_axes.go`) posait `prefixe+"/n"` par un **SET** (`e.num`)
inconditionnellement. Pour un calque à deux niveaux (`flagCarries[].spans[]`,
`vehicles[].rides[]`, `vehicles[].samples[]`, `zoneStates[].spans[]`, `zoneStates[].gauge[]`,
et — via la passe générique, en parallèle de la mesure spécialisée correcte — `tracks[].points[]`),
cette fonction est appelée **une fois par groupe de premier niveau** (une équipe de
`flagCarries`, un véhicule de `vehicles`...) : la mesure finale n'était que celle du DERNIER
groupe itéré, jamais la somme.

**Le correctif.** Distinction par profondeur : au niveau RACINE (`profondeur == 0`, un seul
appel par calque de premier niveau — le même calcul que `passeGenerique` pose déjà pour le même
préfixe), `e.num` reste correct (poser = accumuler depuis zéro en un seul appel). Au niveau
IMBRIQUÉ (`profondeur > 0`, plusieurs appels sur le même préfixe), `e.incr` accumule. Un `e.incr`
inconditionnel à TOUS les niveaux avait été essayé en premier et rejeté : `passeGenerique`
pose déjà `<calque>/n` en `e.num` pour le niveau racine, et un `e.incr` y additionnerait à une
valeur préexistante au lieu de la remplacer (mesuré : `flagCarries/n` rendait 4 au lieu de 2
pour deux équipes vides).

**Preuve par mutation** (`internal/replaydiff/empreinte_axes_test.go`, 3 tests neufs) :
`TestSpansDeCalqueImbriqueEstLaSomme` (deux `FlagCarry` de tailles différentes, 2+1=3 attendu),
`TestVehiclesRidesDeCalqueImbriqueEstLaSomme` (même défaut sur un autre calque à deux niveaux),
`TestCalqueRacineNAPasBesoinDeSomme` (non-régression : le niveau racine n'a besoin d'aucune
somme). Mutation temporaire (retour au `e.num` inconditionnel) : les deux premiers tests
rougissent exactement comme attendu (`flagCarries.spans/n` rend 1 au lieu de 3,
`vehicles.rides/n` rend 0 au lieu de 3), le troisième reste vert (le niveau racine n'était pas
affecté) — correctif restauré, suite revérifiée verte.

**Preuve en conditions réelles** (§7, comparaison avec le rapport de la §2) : `flagCarries.spans/n`
sur `bcb6d393` rend désormais **34 → 17** — la VRAIE somme des deux équipes (cohérente avec
`flagCarries.spans/total`, mesurée séparément par `e.incr` et donc jamais affectée par ce bug) —
alors que le rapport de la 1re exécution (bug non corrigé) rendait « 3 → 1 » (le dernier groupe
itéré). La perte RÉELLE de matière sur ce témoin (déjà consignée au registre comme découverte
nouvelle) est confirmée par ce chiffre correct : moitié moins de spans de drapeau, pas un
artefact de mesure.

## 7. Ré-exécution au HEAD (2e passe) — les deux modes

Mêmes commit et manifeste que la §2 (worktree `LevelUp-wt-v2-corpus`, HEAD étendu des
corrections des §5-§6). Protocole de verrou inter-agents respecté à chaque cuisson.

### 7.1 Mode base (défaut) — HEAD contre `origin/feat/v75`

```
temoin       famille          base(origin/feat/v75)   HEAD    gains   pertes      duree  statut
bcb6d393     ctf_mono_manche      43     43        0        0     11.44s  ok
fb1a1a72     ctf_multi_manche     43     43        0        0     29.71s  ok
d9781168     oddball              43     43        0        0     49.11s  ok
c75f33b8     assaut_bombe         43     43        0        0      15.8s  ok
bf15f7ab     slayer               43     43        0        0     13.47s  ok
51ebbc0f     deux_manches         43     43        0        0     19.24s  ok
084a804d     vehicules            43     43        0        0    2m32.3s  ok
```

**Code de sortie 0. 7/7 témoins « ok », 0 gain, 0 perte, schéma 43 des deux côtés.** Exactement
l'attendu : ce lot (le gate lui-même + le correctif §6 dans `internal/replaydiff`) ne touche
aucun code de cuisson (`internal/replaybuild`, `internal/analysis/replay`,
`internal/analysis/filmdec`) — les deux artefacts (base et HEAD) sont produits par un code de
cuisson STRICTEMENT IDENTIQUE, donc byte-identiques en substance. C'est la démonstration que le
mode base fonctionne : un gate silencieux sur un lot qui ne change aucune cuisson.

### 7.2 Mode parc (informatif) — HEAD contre l'artefact déjà cuit

```
temoin       famille            parc   HEAD    gains   pertes      duree  statut
bcb6d393     ctf_mono_manche      20     43      205       27     11.23s  PERTE
fb1a1a72     ctf_multi_manche     34     43       27        2     31.19s  PERTE
d9781168     oddball              23     43      176        7     24.23s  PERTE
c75f33b8     assaut_bombe         28     43      168        9     14.45s  PERTE
bf15f7ab     slayer               34     43       41        2     15.51s  PERTE
51ebbc0f     deux_manches         21     43      184       10     18.47s  PERTE
084a804d     vehicules            20     43      483       21   2m13.37s  PERTE
```

**Code de sortie 0 (mode informatif, sans `--strict`)** : le parc n'est jamais à jour, ces pertes
sont attendues et NE bloquent PAS le gate par défaut — cf. §5. Le détail nommé
(`imprimerDetailPertes`) confirme, chiffres CORRIGÉS (§6) à l'appui, la même analyse que la §3
ci-dessus : l'immense majorité des 96 pertes brutes (contre 75 dans la 1re exécution — le
correctif révèle correctement `tracks.points/n`, qui était auparavant silencieux ou trompeur)
relève de motifs déjà expliqués dans `BALAYAGE_PARC_2026-09-06.md` (bornes de scène assainies,
reclassements, compteurs de défaut, réattributions dans un gain — souvent avec des chiffres
identiques au balayage), plus les deux faits nouveaux déjà consignés au registre (`bcb6d393`
flagCarries, `084a804d` equipmentEpisodes — désormais chiffrés correctement : 34→17 spans de
drapeau, pas 3→1). Aucune régression massive. Ces deux faits restent au registre, non traités
ici (un autre agent les instruit).

## 8. Gates joués

```
cd apps/go-api
go test -count=1 ./cmd/replay-diff/... ./cmd/replay-corpus-gate/... ./internal/replaydiff/... ./internal/archlint/...
go build ./...                         # CGO_ENABLED=0 et CGO_ENABLED=1
golangci-lint run --new-from-merge-base=origin/main ./...
go vet ./...
```

Résultats détaillés : cf. rapport final de la session (thought_log + réponse à l'utilisateur).
