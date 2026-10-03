# Le rattrapage des cartes absentes au fetch de films

## Le trou

`Pickup.origin: spawner` (schema 32) se lit sur la CARTE : le catalogue `map_weapon_pads.json`
declare les points d'apparition. Une carte absente du catalogue rend `spawner` impossible, et
l'artefact le dit (`spawnPointsState: "map_absent"`). Le catalogue couvrait 72 cartes sur la
centaine jouee, et rien ne comblait jamais ce trou.

## Ou le rattrapage vit, et pourquoi pas ailleurs

| emplacement | verdict |
|---|---|
| sync rapide | EXCLU — il reste intact, c'est le chemin que l'utilisateur attend |
| cuisson d'artefact | EXCLU — offline-pure, une generation ne telecharge RIEN |
| **fetch de film** | **ICI** — dernier maillon EN LIGNE, il connait deja le match |

Carte ABSENTE du catalogue : un appel UGC, depot du `.mvar`, ajout d'une CLE NEUVE.
Carte PRESENTE : rien, pas meme un appel. Verifier la derive d'une carte connue couterait un
appel API PAR MATCH, ce qui est exactement la lourdeur que ce chemin doit eviter.

## Ce qui rend l'ecriture sure

- **Ajout-seul STRUCTUREL** : `mapcatalog.AddEntry` relit le catalogue, REFUSE si la cle existe,
  et n'ecrit que sinon. Ce n'est pas une consigne d'appelant, c'est la seule chose que la
  fonction sache faire.
- **Ecriture atomique** (fichier temporaire puis `rename`) : le serveur LIT le meme fichier
  pendant ce temps.
- **Best-effort strict** : auth, reseau, parse, ecriture — tout echec est journalise, compte, et
  le fetch de film CONTINUE. Un film ne se perd jamais a cause d'un `.mvar`.

## LE PIEGE DU FICHIER DE VARIANTE — et il a failli empoisonner le catalogue

Une passe de re-validation (`--refresh-drifted`) a rendu des socles **deplaces de 22 a 80 metres
sur neuf cartes**. Aucune mise a jour du jeu ne deplace un socle de 80 m : le suspect etait la
resolution de fichier, et c'etait bien elle.

**Un asset UGC de carte sert souvent DEUX `.mvar`** : la carte de BASE, nommee d'apres le niveau
(`btb_highpower.mvar`, `ctf_aquarius.mvar`), et la VARIANTE jouee, nommee `map.mvar`. Les deux
parsent, les deux rendent des socles plausibles — et leurs socles n'ont rien a voir.

**Preuve par les comptes d'objets que le catalogue enregistre lui-meme :**

| carte | catalogue | `map.mvar` | fichier nomme |
|---|---|---|---|
| Highpower Sentry Defense | 421 objets | **421** | 524 |
| Aquarius - Ranked | 236 objets | **236** | 349 |

L'aplatissement fautif prenait « le plus gros fichier ». Avec `map.mvar` :

| | fichier fautif | fichier correct |
|---|---|---|
| cartes regenerees | 16 | **8** |
| dont AUCUN changement de socle | — | **6** |
| deplacements significatifs | 9 (jusqu'a 79,87 m) | **2** |

**Sept des neuf ecarts spectaculaires etaient un artefact de resolution.**

**Ce que cela change RETROACTIVEMENT** : les seize « cartes a source derivee » du lot precedent
n'avaient pas derive — le dump local portait le mauvais fichier pour elles. Le verrou a trois
termes (`objects_n`, `level_id`, socles) a fait exactement son travail : il a refuse d'ecrire
des points issus d'un fichier qui ne decrivait pas la carte.

**Ce que la verification a evite** : sans elle, la passe aurait ecrit dans le catalogue les
socles de la carte de BASE sur neuf cartes tres jouees (Deadlock, Fragmentation, Highpower,
Oasis, Breaker, Scarr...). Tous les rejeux FUTURS de ces cartes auraient affiche des socles a
des dizaines de metres de leur place, et la datation des occupations aurait cesse de trouver
quoi que ce soit — sans qu'aucun test ne rougisse, puisque le fichier reste parfaitement valide.

**Correctif** : `FetchMvarForMap` prefere `map.mvar`, puis le fichier declare par le catalogue
d'objectifs, puis le premier de la liste. L'ordre est fige par
`TestPreferenceDuFichierDeVariante` (5 cas, dont le canevas Forge et la carte native sans
`map.mvar`).

## Etat livre

- Le catalogue n'est **PAS modifie** par ce lot : il reste a 56 cartes a points etablis / 16 non
  etablies, byte-identique a la reference du depot.
- `--refresh-drifted` existe et est teste, mais **sa PASSE n'est pas livree** : les deux cartes
  qui bougent encore avec le bon fichier (`live_fire_-_ranked`, `recharge_-_ranked`) changeraient
  ce que l'application sert. C'est une decision produit, pas une decision d'outillage.
- Les six cartes qui retombent socle pour socle pourraient etre re-validees sans risque : leur
  `.mvar` a change, leurs socles non.

## Reproduire

```
go run ./cmd/mapopads-build --refresh-drifted --dry-run --from <dossier de .mvar>
```

Le dossier doit contenir, pour chaque carte, le fichier de la VARIANTE (`map.mvar` quand l'asset
en porte un), sous le nom que le resolveur attend. Prendre le plus gros fichier est une erreur :
c'est ce qui a produit les 80 metres.
