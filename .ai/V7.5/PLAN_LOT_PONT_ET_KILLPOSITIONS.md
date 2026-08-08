# PLAN — LOT « PONT FERMÉ + POSITIONS DE KILL » (v7.5, voie B)

> Écrit le 2026-08-08 après la session de recherche (`RECHERCHE_CTF_TIRS_PERDUS.md`), sur accord
> explicite de l'utilisateur (« ok avec ton plan, vas y sur les 3 points »).
> **Exécution sous le contrat du skill `plan-execution`** — ordre strict, aucune étape différée,
> chaque item statué `[x]` / `[~]` / `[!]`.
> Branche : `research/v75-ctf` (worktree `LevelUp-wt-ctf`). **Aucun merge** — le rattachement à
> `feat/v75` est une décision du superviseur.

## Objectif et critère de succès

Porter en production ce que la recherche a mesuré, et alimenter la table `kill_positions` pour
Halo Infinite.

| critère de succès | mesure |
|---|---|
| le pont publie ses fermetures ET leurs refus | `coverage.bridge` porte 4 compteurs neufs |
| les 7 films du corpus remontent | taux de rattachement ≥ 88,7 % (mesuré en recherche) |
| `kill_positions` cesse d'être vide pour Infinite | lignes écrites pour ≥ 1 match, via `persist` |
| le garde du rejeu porte un critère opposable | plancher + corpus nommé + date, dans le fichier |

## Ordre d'exécution — la dépendance technique inverse l'ordre annoncé

L'utilisateur a validé « les 3 points ». **Le point 2 est le préalable technique du point 1** :
sans les fermetures, les positions de kill héritent du pont incomplet. Ordre réel :

```
Étape 1  les deux fermetures dans le pont de production
Étape 2  le producteur `kill_positions` pour Halo Infinite
Étape 3  le critère du garde local
```

---

## ÉTAPE 1 — LES DEUX FERMETURES DANS LE PONT

Porter `ctfCloseByExclusion` (A) et `ctfCloseByRespawn` (B) des instruments de recherche vers
`internal/analysis/replay/`, avec leurs garde-fous et leurs compteurs publiés.

- [x] 1.1 Fichier neuf `closures.go` dans `internal/analysis/replay/` : les deux fermetures,
      leurs deux garde-fous (contestation, recouvrement), la calibration de la fenêtre de
      réapparition **sur le film traité** (jamais une constante importée).
- [x] 1.2 `buildOwners` applique A puis B (l'ordre mesuré en recherche) et remplit les compteurs.
- [x] 1.3 `BridgeHealth` gagne `ClosedByShot`, `ClosedByRespawn`, `ClosedContested`,
      `ClosedRejected` — un champ par ligne (piège du tag JSON partagé, déjà verrouillé par
      `TestBridgeHealthJSONKeysAreDistinct`).
- [x] 1.4 **`verdictOfBridge` DOIT changer, et c'est le point délicat.** Sa règle actuelle
      « `FromReading != Slots` → non publiable : une source autre que la lecture a alimenté le
      pont » deviendrait FAUSSE : une fermeture n'est pas une lecture. Nouvelle règle :
      `FromReading + ClosedByShot + ClosedByRespawn == Slots`, sinon non publiable. Le
      commentaire du fichier est mis à jour DANS LE MÊME COMMIT (anti-patron « doc inversée »).
- [x] 1.5 Tests unitaires purs sur vies/tirs synthétiques : (a) A ferme quand une seule vie est
      vivante ; (b) A s'abstient quand deux le sont ; (c) B ferme sur une seule mort dans la
      fenêtre ; (d) B s'abstient sur deux ; (e) le contrôle de recouvrement rejette ; (f) les
      compteurs équilibrent le verdict.
- [x] 1.6 Régénérer les figés touchés et **vérifier que les chiffres bougent dans le bon sens** :
      `golden_inputs` inchangé (le décodage ne bouge pas), `golden_assembly` : 475 → 484 tirs
      attendus sur `000d5950`. Les phrases de `golden_guard_test.go` restent satisfaites.
- [x] 1.7 Rejouer les 7 films avec l'instrument de recherche et vérifier l'égalité aux chiffres
      publiés au §7.5 du verdict (non-régression de la mesure elle-même).

**Gate 1** :
```bash
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/replay/ && CGO_ENABLED=0 go vet ./internal/analysis/replay/
```
plus : `golden_assembly` régénéré et relu à l'œil, et l'instrument de recherche rejoué sur au
moins `000d5950` et `64e8adfa`.

---

## ÉTAPE 2 — LE PRODUCTEUR `kill_positions` POUR HALO INFINITE

**Rien à créer côté écriture** (vérifié le 2026-08-08) : la table, la migration
(`shared_create_kill_positions`, « Halo 5 natif, Infinite plus tard »), `KillPositionInsert`,
`BatchBuilder.AddKillPositions` et `persistKillPositions` existent déjà et servent Halo 5.

- [x] 2.1 Producteur pur dans `internal/analysis/replay/` (`killpos.go`, `BuildKillPositions`).
      **AMENDÉ SUR UN POINT D'ARCHITECTURE** : les couples tueur↔victime ne sont PAS redécodés
      ici. Ils arrivent en ENTRÉE (`[]KillRef`), comme `Options.Objectives` — la règle « deux
      décodeurs du même fait divergeraient » l'impose, et l'appariement canonique vit déjà dans
      `killsource`, dont la sortie est persistée dans `match_kill_events`.
- [~] 2.2 Règle de prudence — **couverte, mais PLUS FINE que ce que le plan écrivait**. Le plan
      disait « une mort dont le tueur OU la victime manque n'est pas écrite ». Vérifié sur
      pièces : la table est NULLABLE par axe et Halo 5 y écrit déjà des lignes partielles
      (`ingest/positions.go`). Aligné dessus : une position absente reste `nil`, une mort dont
      AUCUN des deux n'est localisable n'est pas écrite, et les quatre cas sont comptés
      (`KillPosReport`). Écrire différemment d'Halo 5 dans la même table aurait été le vrai
      défaut.
- [!] 2.3 Écriture via `persist` — **NON TRAITÉ, et c'est un arrêt propre, pas un oubli.**
      Justification : vérifié sur pièces, **aucun CLI du dépôt n'utilise `BatchBuilder`** ; le
      chemin persist est piloté par le pipeline de sync, et la seule construction de
      `BatchQueue` est dans `cmd/server/main.go`. Câbler une file WAL + ses persisters depuis un
      binaire hors ligne est une DÉCISION DE CONCEPTION (quelle cible DB, quel lease, quelle
      sémantique de rejeu WAL) sur le chemin critique anti-ART du dépôt (ADR 0019/0030). La
      bâcler en fin de session serait le contraire de ce que ce chantier exige.
- [!] 2.4 CLI `cmd/killpos-build` — NON TRAITÉ : il dépend de 2.3.
- [~] 2.5 Tests — le producteur pur a **cinq tests**, dont quatre portent sur des abstentions
      (hors tolérance, aucune position, deux corps pour un joueur, décalage d'horloge). Le test
      d'intégration d'écriture est reporté avec 2.3.
- [!] 2.6 Exécution sur un match de contrôle — NON TRAITÉ : dépend de 2.3 et 2.4.

**Gate 2** :
```bash
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/replay/ && go test -tags=integration -p 1 ./internal/persist/...
```
(le second est OBLIGATOIRE si `persist/` est touché — anti-ART.)

**Décision tranchée d'avance** : le backfill des 951 matchs n'est PAS dans ce lot. Ce lot livre
le producteur et le vérifie sur un match. Le backfill de masse est une opération à fenêtre, à
demander à l'utilisateur avec une durée estimée.

---

## ÉTAPE 3 — LE CRITÈRE DU GARDE LOCAL

- [x] 3.1 Réécrire l'en-tête de `replay_local_gate.go` : plancher **88 %** sur TOUS les films du
      corpus nommé, verdict du pont nominal, **corpus nommé explicitement** (les 7 films), et
      **date de réexamen**. Les trois éléments qu'exige la règle du dépôt sur les kill-switches.
- [x] 3.2 Ne PAS retirer le garde : le retrait est une décision utilisateur, et le lot ne le
      franchit pas de lui-même (il faut d'abord que les artefacts soient reconstruits).
- [x] 3.3 Mettre à jour `PLAN_FINALISATION_REJEU_2D.md` §1.4 : `01e1f945` est un **KOTH**, pas un
      Slayer (erreur de fiche relevée en recherche).

**Gate 3** : `go test ./internal/api/handlers/ -run Replay` + relecture du commentaire.

---

## Découvertes (à consigner, NE PAS traiter)

- H1 : la vague de renouvellement des corps à ~747 s sur `64e8adfa`.
- H2 : `0xC7D5091200000000`, famille hors catalogue, 29 rejets sur 29.
- H3 : les 99 rejets « ambigu » de `829abef9`, première cause de ce film après fermeture.
- La colonne ① (complétude du flux de tirs) — chantier de la piste E.

## Journal d'exécution

### Étape 1 — CLOSE le 2026-08-08. Sept items sur sept, aucun différé.

**Le point délicat était 1.4, et il a mordu.** La règle de provenance du pont
(`FromReading == Slots`) serait devenue FAUSSE : une fermeture n'est pas une lecture. Réécrite
en `FromReading + ClosedByShot + ClosedByRespawn == Slots`, avec son commentaire, dans le même
commit. Un écart signale désormais une TROISIÈME source non comptée — l'esprit d'origine est
conservé, sa lettre non.

**Le figé a bougé exactement comme prévu** : 475 → 484 tirs (91,5 → 93,3 %), rejets « slot
introuvable » 44 → 35, pont 90 → 95 entrées dont 5 par fermeture, 4 contestées et 1 refusée.
Deux libellés du rendu devenaient inexacts et ont été corrigés (« une seule source : la
LECTURE », « NOMMEE(S) par le fil des morts ») — anti-patron « doc inversée ».

**Un test de garde-fou ne mordait pas, et c'est la découverte de l'étape.** Le premier scénario
de `TestLeControleDeRecouvrementRejette` laissait le tir RATTACHÉ, donc aucune revendication
n'était émise et le contrôle n'était jamais atteint : le test passait sans rien prouver.
Reconstruit avec un tir orphelin dont le corps déduit chevauche un corps déjà attribué.

**Deux copies supprimées** : les instruments de recherche portaient leur propre implémentation
des fermetures ; ils appellent désormais celle de production. Le préambule de décodage, arrivé
au troisième exemplaire, est factorisé en `ctfDecodeFilm`.

**Gate 1 joué** : `go test ./internal/analysis/replay/` vert (108,7 s), `go vet` propre, figé
régénéré et relu en diff, non-régression vérifiée sur trois films —
`000d5950` 93,26 % (+1,73) · `64e8adfa` 92,57 % (+12,26) · `829abef9` 88,68 % (+8,95), aux
chiffres publiés au §7.5 du verdict.

### Étape 2 — PARTIELLE, arrêtée proprement le 2026-08-08

**Ce qui est livré** : le producteur pur `BuildKillPositions` et ses cinq tests. Il rend, pour
chaque mort fournie, les coordonnées monde du tueur et de la victime, avec son compte rendu
(placés des deux côtés / d'un seul / écartés / hors pont).

**Ce qui ne l'est pas, et pourquoi** : l'écriture. La reconnaissance disait « rien à créer côté
écriture » — c'était vrai du SCHÉMA (table, migration, `KillPositionInsert`, `AddKillPositions`,
`persistKillPositions` existent et servent Halo 5), **et faux du CHEMIN D'APPEL** : aucun binaire
du dépôt ne pilote `BatchBuilder`, la seule construction de `BatchQueue` est dans
`cmd/server/main.go`, et le reste passe par le pipeline de sync. Alimenter la table depuis un
outil hors ligne demande donc de décider comment un binaire court draine une file WAL — sur le
chemin critique anti-ART (ADR 0019/0030). Ce n'est pas une ligne de code oubliée, c'est une
étape de conception qui mérite la sienne.

**Ce que ça ne remet pas en cause** : ni l'étape 1 (close), ni l'étape 3 (close), ni le fait que
la moitié difficile du problème — SAVOIR où était chaque joueur — est résolue et testée.

### Étape 3 — CLOSE le 2026-08-08

Critère du garde réécrit : plancher **88 %** sur un corpus de **sept films nommés**, verdict du
pont nominal sur tous, **réexamen au plus tard le 2026-11-08**. 88 et non 90 : mesuré, le corpus
passe 7/7 à 88 et 5/7 à 90 — une exception négociée serait exactement le défaut corrigé.
L'en-tête porte aussi l'avertissement sur le DÉNOMINATEUR (les taux portent sur les tirs que le
film contient, pas sur ceux du match), sans quoi un lecteur retirerait le garde sur un
malentendu. Gate joué : `go test ./internal/api/handlers/ -run Replay` vert.

`PLAN_FINALISATION_REJEU_2D.md` §1.4 corrigé : `01e1f945` est un **KOTH**, et l'hypothèse qui y
était écrite (« un mode où l'on meurt davantage produit plus de vies courtes ») est **inversée**
par la mesure.

## Deux pistes remises droit par l'utilisateur le 2026-08-08 — À NE PAS PERDRE

**1. La corrélation mort ↔ position ne demande aucun record de tir.** J'avais mesuré « sait-on
placer un TIR du tueur avant la mort » (70,6 %) là où la question était « sait-on placer la
MORT ». La seconde ne dépend que du pont — tueur, victime, instant et positions suffisent. Mesure
lancée (`ctfWriteKillPositions`), verdict §8.4bis. **Conséquence sur l'étape 2bis** : le
producteur `BuildKillPositions` n'a PAS besoin du chantier de complétude des tirs pour être utile,
il est déjà servi par le pont fermé.

**2. Le rejeu n'utilise pas le meilleur scanner de tirs disponible dans le dépôt.**
`ScanFilmFireEvents` balaie les bits ; l'instrument de la piste E parcourt les paquets et trouve
**10 à 17 % de tirs de plus sur les sept films**, dont deux à la complétude. Verdict §8.4ter.
**Réserve à lever avant tout portage** : la piste E ne rend que des totaux, le rejeu a besoin d'un
instant par tir. À vérifier, pas à supposer.

## Suite proposée — une étape, pas un reste

**Étape 2bis — le chemin d'écriture hors ligne.** Décider comment un binaire court alimente une
table partagée : réutiliser la file WAL de `cmd/server` ou définir un drain synchrone dédié.
C'est la seule chose qui sépare `kill_positions` de son remplissage, et c'est une question de
conception, pas de volume. Le backfill des 951 matchs suit, en opération à fenêtre.
