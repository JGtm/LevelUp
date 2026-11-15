# Témoin avant / après du lot B : le mode `depart` ne change pas (2026-11-15)

Plan : `PLAN_REAPPARITIONS_2026-11-15.md`, item B.3. Le critère : en mode `depart` (défaut), la sortie du service
Tactique est identique octet pour octet avant et après le lot.

## Conditions

- Aucune écriture dans le dossier de données principal, aucune jonction. Les artefacts de trois cartes (les trois
  plus jouées du joueur parmi celles qui ont un artefact : 58, 56 et 53 matchs, soit 166 sidecars écrits) et une
  COPIE de la base partagée (lue seulement pour retrouver la carte de chaque match) ont été copiés dans un dossier
  temporaire ; `LEVELUP_DATA_DIR` pointe dessus. Le dossier principal garde ses 1 460 sidecars en schéma 6.
- Sidecars générés par `levelup tactical-rasters --backfill` sur cette copie : 166 lus, 166 écrits, 0 échec, 166 au
  schéma 7 (la voie de repli, le parc étant au schéma 116).
- Témoin : un test jetable du paquet `service` (non versionné) monte le service Tactique avec le VRAI lecteur de
  sidecars (`NewTacticalRasterStore`) sur ce dossier et un double de dépôt dont l'univers est la liste des matchs de
  la carte. Par carte : lecture `temps` et `routes` sans filtre, puis, pour chacune des (au plus quatre) premières
  grappes de départ, `temps` et `routes` sous ce filtre, et le détail de la cellule la plus chaude des routes avec et
  sans le filtre de grappe. Tout est sérialisé en JSON (23 655 579 octets).
- AVANT : le test est lancé sur `HEAD` (`c6bcbf0dd8`, extrait par `git archive` dans un dossier à part, compilé à part).
  APRÈS : le même test, même fichier, sur l'arbre du lot B, sans aucun paramètre `vies` (`Scope.Vies` vide).

## Résultat

`cmp avant.json apres.json` : identiques.

```
92e8efb20bc44dbd9be77ac2723da7ea69f23a204beee3bbb9cd8c3e32d44d56  avant.json
92e8efb20bc44dbd9be77ac2723da7ea69f23a204beee3bbb9cd8c3e32d44d56  apres.json
```

| Carte (début de l'identifiant) | Matchs | Grappes de départ | Matchs retenus (`temps`) | Cellules de `routes` |
|---|---|---|---|---|
| `3e1e4cec` | 56 | 4 | 56 | 2 738 |
| `5324364b` | 53 | 3 | 52 | 3 507 |
| `9e821f5e` | 58 | 3 | 58 | 2 786 |

Ce que le test d'égalité ne couvre pas, `depart` explicite contre vide, est tenu par `TestVies_DepartEstLeDefaut`
(`service`) et `TestVies_DepartSousFiltre_RoutesInchangees`.

## Ce que `toutes` donne sur la même carte (information, hors témoin)

Carte `9e821f5e` (58 matchs), lecture `routes` : 3 grappes sous `depart`, 56 sous `toutes`. Sous `toutes` et le
filtre d'une grappe, les routes comptées passent de 2 786 cellules (sans filtre) à 73 à 105 cellules selon la grappe
(matchs retenus : 16 à 18), c'est-à-dire seulement les routes des vies parties de la grappe.
