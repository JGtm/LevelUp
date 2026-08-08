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

- [ ] 2.1 Producteur pur dans `internal/analysis/replay/` : à partir des positions décodées, du
      pont fermé et du fil des morts apparié (frag ⋈ mort), rendre pour chaque mort les
      coordonnées monde du tueur ET de la victime à l'instant de la mort.
- [ ] 2.2 **Règle de prudence identique au reste du chantier** : une mort dont le tueur OU la
      victime n'a pas de position à moins de la tolérance n'est PAS écrite. Se taire vaut mieux
      que poser un kill au mauvais endroit. Le nombre d'écartés est journalisé.
- [ ] 2.3 Écriture via `persist.BatchBuilder.AddKillPositions` → `Submit()`. **Aucun SQL direct,
      aucun UPSERT** (ADR 0019/0030).
- [ ] 2.4 CLI hors ligne `cmd/killpos-build` sur le modèle de `cmd/replay-build` (chemins par
      `PathResolver`, `--map` obligatoire, film en lecture seule).
- [ ] 2.5 Tests : producteur pur testé sans I/O ; un test d'intégration si l'écriture est touchée.
- [ ] 2.6 Exécution sur **un** match de contrôle, et relecture en base (`SELECT count(*)`).

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

- [ ] 3.1 Réécrire l'en-tête de `replay_local_gate.go` : plancher **88 %** sur TOUS les films du
      corpus nommé, verdict du pont nominal, **corpus nommé explicitement** (les 7 films), et
      **date de réexamen**. Les trois éléments qu'exige la règle du dépôt sur les kill-switches.
- [ ] 3.2 Ne PAS retirer le garde : le retrait est une décision utilisateur, et le lot ne le
      franchit pas de lui-même (il faut d'abord que les artefacts soient reconstruits).
- [ ] 3.3 Mettre à jour `PLAN_FINALISATION_REJEU_2D.md` §1.4 : `01e1f945` est un **KOTH**, pas un
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
