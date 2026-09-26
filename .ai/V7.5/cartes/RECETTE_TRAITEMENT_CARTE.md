# Recette — traiter UNE carte de fond de rejeu 2D

> Ecrite le 2026-08-26 a la demande de l'utilisateur : « note dans un doc ou un handoff ton
> process pour traiter les cartes, parce que quand on sera arrives au bout de ton contexte ca
> va etre la misere de continuer s'il n'y a pas de recette ».
>
> Elle suppose lus, dans cet ordre : `REGISTRE_CARTES.md` (etat de chaque carte, verdicts
> avec verbatim) puis `PLAN_REVUE_CARTE_PAR_CARTE.md` (pourquoi le chantier existe).
> Branche `wt/cartes-revue-par-carte`, **worktree PRINCIPAL** : la cuisson exige `data/` et
> l'installation du jeu.

## La regle qui gouverne tout le reste

**UNE carte a la fois, et on ne passe a la suivante que quand la precedente est validee et
publiee.** L'utilisateur l'a impose le 26/08 apres une planche de masse : « il faut
retravailler une a une celles qui sont non validees ».

Corollaire : ne JAMAIS presenter plusieurs cartes a valider ensemble, et ne jamais publier en
production une image que l'utilisateur n'a pas vue.

## Les cinq axes, et ou ils vivent

Aucun n'est une branche de code. Tous sont des ENTREES de `himap.OptionsCuisson`, choisies
dans `data/titles/halo_infinite/reference/map_fond_reglages.json`, une entree par carte.

| axe | champ JSON | defaut | ce qu'il corrige |
|---|---|---|---|
| habillage | `style` | `jeu` | `encre` = quasi monochrome ; la teinte ne bascule pas, seule la valeur varie |
| echelle | `echelle` | **auto**, 3 000 px de grille | une petite arene rend une petite image, pixelisee des qu'on l'agrandit |
| toits | `ecreteToits` | absent | vide les pixels dont aucune surface n'est a hauteur de jeu |
| plafond | `plafondArene` | 6 m | le cran de l'ecretage : 4 m si « encore trop de toits », 8 m si « trop vide » |
| zones | `rogneAuxZones` | absent | efface la matiere hors des callouts dilates de 4 m |

## Le taux de couverture decide de l'ecretage — ne pas tatonner

Ajoute le 2026-08-26 apres Launch Site. `mapfond-build` journalise le taux de couverture
de chaque carte. **Il faut le lire AVANT de choisir d'armer l'ecretage**, parce que les deux
regimes sont opposes et qu'aucune image ne les distingue avant cuisson :

| taux de couverture | ce qui se passe | ecretage |
|---|---|---|
| **sous 1/3** (`SeuilCarteCouverte`) | la voie de reference native ne se declenche JAMAIS | **seul chemin** — Catalyst 28,4 %, Streets |
| **au-dessus de 1/3** | la voie native se declenche et remet deja le sol | **le REFUSER** — Launch Site 53,5 % |

Cause : armer l'ecretage REMPLACE la voie native au lieu de s'y ajouter (memes tampons
liberes). Sur une carte deja couverte, substituer une coupe a hauteur fixe a la voie native
ne peut que retirer du sol : sur Launch Site, 162 139 cellules de matiere contre 245 307 sans
lui, et 16 ancres sur 28 contre 20 — l'arene devient un contour creux. Sur Illusion, meme
cause, autre effet : 108 810 cellules substituees contre 1 338 343 par la voie native, donc
une arene sans relief.

**Tant que ce defaut n'est pas corrige, une carte couverte ne s'ecrete pas.**

## Un vide dans l'arene : ce sont les ancres qui tranchent, pas l'oeil

Ajoute le 2026-08-26. Devant une zone blanche au milieu d'une arene, la question « trou reel
ou matiere manquante ? » se decide par les ancres d'objectif : **une ancre d'objectif est du
terrain joue par definition.** Des ancres sans sol dessous DANS le vide = matiere manquante,
et le comblement est justifie (Launch Site : 8 ancres sur 28 flottaient, comblement arme,
2 recuperees). Aucune ancre dans le vide = vide probablement reel, ne pas combler (Chasm :
le gouffre, comble a tort).

Le **cadre** n'est pas un reglage : il est toujours rogne a la matiere plus 6 m
(`himap/cadre_utile.go`). L'origine du calage suit le rognage, l'echelle NON.

**Toute entree porte `raison` (>= 80 caracteres) et `gateLe` (AAAA-MM-JJ).**
`TestReglagesFondJustifies` refuse une entree sans les deux, sans effet, ou avec un habillage
inconnu. Ces seuils ne se relevent pas pour faire passer une entree.

## La boucle, etape par etape

### 1. Choisir la carte

L'ordre est celui des matchs decroissants parmi les non closes — il est tenu a jour au
registre. Ne pas devier sans raison ecrite.

### 2. L'echelle est AUTOMATIQUE — ne plus la calculer a la main

**Change le 2026-08-26, apres douze entrees ecrites une par une.** `himap.EchellePourCadre`
vise `CibleCadrePx` = 3 000 px sur le plus grand cote de la GRILLE et en deduit le cote du
pixel, borne a [0,025 ; 0,0920 m/px]. Le rognage au cadre utile garde ensuite 42 a 55 pour
cent de la grille, soit environ 1 300 a 1 600 px publies.

Pourquoi la main etait une erreur : le cadre monde est connu AVANT le rendu — il vient des
ancres — donc la valeur n'exprimait rien de propre a la carte, seulement la taille de son
arene. Douze copies d'un meme calcul, c'est le « copy-paste config » que la revue interdit
des la troisieme.

**Une `echelle` explicite reste PRIORITAIRE** : les onze cartes deja gatees gardent la leur au
pixel pres, la regle ne repasse pas dessus. Garde-rail : `TestEchelleExpliciteGagneToujours`.

Pour mesurer la matiere utile d'un fond publie (diagnostic, plus pour regler) :

```
CGO_ENABLED=0 go -C apps/go-api run ./cmd/mapfond-cadrage --dir <.../map_backgrounds>
```

### 3. Ecrire l'entree de reglages

Style `encre` par defaut depuis le gate du 26/08 (« style encre me va »). Ecretage et rognage
aux zones : les poser en premiere intention sur une carte couverte, les retirer si la mesure
ou l'image le demandent. La raison doit contenir les CHIFFRES qui ont motive le choix, pas une
paraphrase.

### 4. Cuire vers un dossier SCRATCH, jamais en production

```
CGO_ENABLED=1 go run ./cmd/mapfond-build --maps <cle> --forge=false --out-dir <scratch>
```

Une cuisson dure une a deux minutes par carte. **Ne jamais lancer deux commandes Go en
parallele** : le cache se corrompt (lecon payee).

### 5. LIRE LES CHIFFRES du journal et du sidecar

C'est l'etape qu'on saute et qui coute le plus cher. A regarder systematiquement :

- `matiere hors des zones nommees ... part=X%` — au-dela de ~25 %, le prevenir a
  l'utilisateur AVANT qu'il regarde l'image, jamais apres.
- `cadre rogne sur la matiere avant=AxB apres=CxD` — le gain de cadrage.
- `anchorMedianGapM` — attendu **-0,29 m** (l'etalonnage). Un ecart fort signale soit des
  toits, soit le defaut de niveau de jeu (voir « ce qui reste ouvert »).
- `cellsClipped` — combien l'ecretage a VIDE. Zero = chaque toit avait un sol dessous.

### 6. OUVRIR LE PNG

**Non negociable.** Trois defauts de la journee du 26/08 n'ont ete attrapes que par la : le
cadre du rejeu suivi sur des props invisibles, le catalogue ampute de 59 modules, et la dalle
d'eau de Recharge. Aucun test ne les couvrait.

### 7. Publier la planche

```
go run ./cmd/mapfond-planche --manifeste M.tsv --sortie P.html --cote 560 --titre "..." --intro "..."
```

Manifeste TSV, une ligne par image : `cle · libelle · sous-titre · statut · colonne · chemin`.
Plusieurs lignes de meme cle = les colonnes d'une meme fiche. Republier le MEME fichier garde
la meme URL.

Le damier derriere chaque image rend le vide transparent VISIBLE — sans lui le defaut de
cadrage ne se voit pas. Mettre dans l'intro les chiffres et **ce qui ne va pas**, avant que
l'utilisateur regarde.

### 8. Verdict, puis publication

Au verdict positif : re-cuire SANS `--out-dir` (donc vers `map_backgrounds/`), passer la ligne
du registre a `VALIDEE` avec le **verbatim** de l'utilisateur, completer la `raison` de
l'entree de reglages avec ce que le gate a etabli, committer.

Au verdict negatif : corriger sur CETTE carte, republier la meme planche, recommencer.

## Ce qui reste ouvert, et qu'il ne faut pas croire regle

- **Le niveau de jeu est un SCALAIRE** : `MedianeZ(ancres) - 0,29` (`cuisson.go`), une
  constante etalonnee sur UNE carte. Sur une carte a plusieurs etages, la mediane n'est le sol
  d'aucun. 13 fonds sur 56 ont un ecart de -3 a -17,5 m, et au-dela de
  `PorteeNiveauDeJeu` = 10 m la teinte peint la surface JOUEE au recul maximal. **Correctif
  nomme, non fait** : la teinte doit lire la surface de reference PAR PIXEL, qui existe deja
  (`SurfaceReference`, armee par `ArmeReference`). Bazaar a prouve que l'ecretage ne le
  resorbe pas toujours (-3,20 m apres ecretage).
- **La dalle d'eau** : le fond de Recharge publie depuis le 10/08 porte un grand rectangle
  bleu. Hypothese NON confirmee : `PoseEau` peint la BOITE ENGLOBANTE d'un volume
  (`AABBMin`/`AABBMax`, sddt.go). Lot a part.
- **Le contour du masque des zones porte des marches rectilignes** : ce sont les carres de la
  dilatation (`dilate` est separable, donc carree). Une dilatation circulaire les supprimerait.
- **Les 37 fonds Forge n'ont AUCUN callout** : le rognage aux zones ne vaudra jamais pour eux,
  et ce sont les plus mal cadres (88,3 % de largeur occupee, la « bouillie » refusee en bloc
  le 13/08). Il leur faudra un autre levier — le prototype « arene par la reference » du
  13/08 n'a jamais ete applique.
- **`SeuilCarteCouverte` (1/3)** est plus en cause que le mecanisme de substitution : Streets,
  mesuree a 7,1 %, n'a jamais declenche la voie de reference qui lui allait.

## Cartes a NE PAS ecreter, et pourquoi

- **Cliffhanger** : ses rochers hauts sont a plus de 6 m de la reference ET hors de ses zones
  nommees. L'ecretage comme le masque les effaceraient, et l'utilisateur l'a validee AVEC eux.

## Pieges deja payes — ne pas les rejouer

1. **Comparer deux chiffres a deux echelles differentes.** 30 970 cellules d'eau a 0,0920 m/px
   et 325 353 a 0,029 m/px, c'est la MEME surface (rapport 3,17 au carre = 10). Trois
   diagnostics faux sont sortis de la.
2. **Conclure avant d'ouvrir le PNG.** Voir etape 6.
3. **`| head -N` sur un run de test** ferme le tube, envoie SIGPIPE et tue le run.
4. **`go test ./internal/himap/`** entier depasse la butee de 10 min sur un poste ou le jeu est
   installe (documente, pas une regression). Lancer les tests qui ne dependent pas du jeu :
   construire le `-run` a partir des fichiers `*_test.go` SANS `_gamefiles_` (50 tests, 5 s).
5. **Un garde-rail ne protege que le champ qu'il compte.** Le lot KOTH du 25/08 comptait les
   collines — vert — pendant qu'il ecrasait `module` sur 58 entrees du catalogue et rendait
   8 cartes incuisables.
