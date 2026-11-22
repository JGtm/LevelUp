# Témoins des antagonistes

Ce dossier capture ce que le rejeu produit aujourd'hui, pour le comparer après un changement : la
toile, la frise, et les règles de Seul et Masquer d'un joueur.

## Capturer

Depuis `apps/web`, avec Git Bash :

```bash
TEMOINS_ANT_DIR="C:/Users/Guillaume/Downloads/Scripts/LevelUp-temoins-ant/avant" \
  npx vitest run src/features/match-replay/test/temoinsAntagonistes --maxWorkers=2
```

Le dossier désigné reçoit les témoins et `empreintes.txt` (une ligne `<sha256>  <fichier>` par
témoin). Sans la variable, la même commande vérifie seulement que chaque sortie se produit ; rien
n'est écrit.

## Comparer deux captures

Capturer l'état d'après dans un autre dossier (`.../apres`), puis :

```bash
diff <dossier avant>/empreintes.txt <dossier après>/empreintes.txt
diff <dossier avant>/toile-avec-2.json <dossier après>/toile-avec-2.json | head
```

Un témoin de toile porte un appel de dessin par ligne : le premier écart désigne l'appel. Un
témoin de frise porte une balise par ligne. Un témoin de filtre porte une chaîne de `1` (dessiné)
et de `0` (retiré) par genre d'élément, dans l'ordre du document.

## Ce que chaque témoin protège

| Témoin | Contenu | Doit rester identique |
|---|---|---|
| `toile-sans-1..3.json` | toile à trois images où le point de vue n'a encore ni éliminé un adversaire ni été éliminé par un adversaire | toujours |
| `toile-avec-1..3.json` | toile à trois images où le point de vue a ses deux titres | hors les appels ajoutés par un dessin d'antagoniste |
| `toile-seul.json`, `toile-masquer.json` | toile sous Seul, puis Masquer, du premier adversaire à éliminer le point de vue | hors l'écart qu'une règle nouvelle annonce |
| `frise.html` et les deux `frise-piste-*.html` | DOM de la frise et de ses deux pistes de marques | hors le placement vertical des marques |
| `filtre-seul.json`, `filtre-masquer.json` | réponses de `shownSlotPredicate`, `ownershipShown` et `buildFocusFilter` sur trois films | hors l'écart qu'une règle nouvelle annonce |
| `tests-existants.txt` | tests de la feature qui portent les règles de Seul et Masquer, par nom | la liste ne perd aucun nom |
| `scene.json` | film, point de vue, images retenues | toujours |

## Ce que la scène contient

Les films viennent de `test/fixtures/go/`. Ils ne portent pas le fil de la base : les éliminations
sont celles que le film écrit lui-même (`scoreActions`, action `killPersonalScore`), passées par
`buildFeedEntries`. La toile reçoit ce fil, le point de vue et l'allégeance : un calque qui lit les
éliminations à l'instant les trouve là.

## Le seul point à suivre

`focusDe` (dans `temoinsKit.tsx`) écrit Seul et Masquer d'un joueur dans ce que la page donne à la toile : la fiche ouverte
(`PlayerFocus`) et l'ensemble des joueurs masqués (`HiddenPlayers`). Masquer X est l'ensemble {X} ; Seul X est l'ensemble de tous
les autres joueurs du film (plan des antagonistes, piste E). Si ces types changent, c'est la seule fonction du dossier à
adapter, avec l'appel à la toile de `peindre` ; les sorties gardent leurs noms.
