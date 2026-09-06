# Gate de non-régression du rejeu sur corpus témoin — 2026-09-06

> Construction de `cmd/replay-corpus-gate` : industrialise la méthode du balayage ponctuel
> (`.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`) sur un corpus TÉMOIN restreint et versionné
> (`config/replay_corpus.toml`), pour qu'un différentiel sur films réels tourne AVANT tout
> merge qui touche au décodeur ou au constructeur de rejeu — les trois régressions du 28/08,
> 30/08 et 02/09 ont traversé des goldens synthétiques pendant dix-neuf schémas faute de ce
> différentiel. Worktree `LevelUp-wt-v2-corpus`, branche `feat/v2-corpus`.

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

## 5. Gates joués

```
cd apps/go-api
go test -count=1 ./cmd/replay-diff/... ./cmd/replay-corpus-gate/... ./internal/replaydiff/... ./internal/archlint/...
go build ./...                         # CGO_ENABLED=0 et CGO_ENABLED=1
golangci-lint run --new-from-merge-base=origin/main ./...
go vet ./...
```

Résultats détaillés : cf. rapport final de la session (thought_log + réponse à l'utilisateur).
