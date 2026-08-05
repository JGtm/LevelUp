# PLAN DE RECHERCHE — les icones d armes pour l interface

> Ecrit le 2026-07-31. **Chantier SEPARE du branchement**, et volontairement : il ne bloque rien,
> il n a aucune dependance sur le decodeur, et il peut echouer sans consequence.
>
> Besoin, dans les mots de l utilisateur : *<< ce n est pas pour la resolution mais purement pour
> l UI. Aujourd hui j utilise des images que j ai dessinees moi-meme, et il m en manque. Si je
> pouvais combler ce gap proprement avec des assets du jeu ce serait parfait. >>*

## LE BESOIN EXACT, ET CE QU IL N EST PAS

**CE N EST PAS un probleme de decodage.** L arme est deja identifiee par son tag `jpt!`, et le tag
est plus precis qu une icone : plusieurs armes partagent la meme image. Le decodeur n a rien a
apprendre ici.

**C EST un probleme d ASSETS** : produire une image par arme, et l associer au tag. Deux moities,
et **la seconde est la difficile**.

## LE PIEGE, DEJA DEMONTRE — a lire avant de commencer

Le chantier a voulu nommer les vehicules par la **racine de banque sonore** du champ `Detail`.
Resultat : sur 14 tags observes, 8 seulement recoivent un nom, et l un d eux etait **faux** — la
banque disait << Pelican >> pour ce que l utilisateur a identifie au Theater comme un **Falcon**.
La banque Wwise est REUTILISEE entre chassis : dix entrees citent de 2 a 6 chassis differents, dont
une qui cite Scorpion, Chopper, Banshee, Wasp et tourelle Shade dans le meme champ.

**Consequence pour ce chantier : la chaine sonore ne peut pas servir a associer une icone.** Il faut
un autre chemin, et c est l objet de la phase 1.

## PHASE 1 — TROUVER LE CHEMIN TAG -> ASSET

Trois voies, par ordre de promesse. **La regle : celle qui donne une correspondance VERIFIABLE gagne,
pas celle qui donne le plus de resultats.**

### Voie A — le graphe de dependances `.module`

C est le chemin qui a DEJA fonctionne pour NOMMER les armes : `weap -> proj -> jpt!` en standard,
et quatre variantes documentees. Le depot sait deja lire ces archives (`internal/himodule/`,
`internal/ooz/`).

- [ ] Un `weap` reference-t-il un asset d icone (bitmap, `bitm`, ou un tag d interface) ?
- [ ] Si oui, remonter du `jpt!` au `weap` puis a l icone — le graphe est deja parcouru dans l autre
      sens par le generateur de catalogue
- [ ] **Controle** : les 39 armes de `metadata.weapon_labels` doivent toutes rendre une icone, et
      deux armes distinctes ne doivent pas rendre la meme image sans raison

### Voie B — l executable, la ou le kill feed choisit son icone

Historique a connaitre : le rendu du kill feed **resout bien l arme pour choisir l icone**, et il
tourne au replay. Cette piste a ete exploree en juin 2026 et abandonnee — la chaine du handle vers
le type d arme s est revelee accessible **en live seulement**. C est cette impasse qui a pousse le
chantier vers le dead-state.

- [ ] Mais la question ici est DIFFERENTE : on ne cherche pas a resoudre l arme (on l a), on cherche
      **le nom de l asset d icone associe a une definition d arme**. C est une table statique, pas
      un chemin runtime.
- [ ] Chercher les chaines d assets d interface autour du widget de kill feed

### Voie C — les assets deja extraits par le depot

- [ ] `static/abilities-assets/halo_infinite/` existe deja et contient onze capacites : quelqu un a
      donc deja su extraire des images du jeu. **Retrouver COMMENT, avant de reinventer.**
- [ ] Reserve mesuree, et elle est serieuse : le dossier voisin des armes est **mal nomme** —
      `Cremator.png` y est en realite le Cindershot. **Un nom de fichier n est pas une source
      fiable**, il faut verifier chaque association visuellement.

**GATE 1** : une correspondance tag -> fichier image, verifiee sur au moins 10 armes en regardant
les images, dont deux armes visuellement proches (BR75 / Bandit Evo) et une arme a variantes
(marteau antigrav).

## PHASE 2 — L INTEGRATION

- [ ] Les assets vivent la ou le depot les met deja, avec la convention par titre
- [ ] La correspondance passe par `TitleAssetURLAdapter` (ADR 0011) — **jamais une URL en dur dans
      un composant**
- [ ] Une arme sans icone tombe sur un placeholder explicite, **jamais sur l icone d une autre arme**
- [ ] Les icones dessinees a la main restent la ou l extraction echoue : ce chantier COMPLETE, il ne
      remplace pas

**GATE 2** : `make check-types`, aucune URL en dur, un test qui verifie qu une arme inconnue rend le
placeholder et non une image arbitraire.

## CE QUI FERAIT ECHOUER CE CHANTIER, ET COMMENT LE VOIR TOT

| Risque | Signal precoce | Reponse |
|---|---|---|
| aucune icone dans les `.module` | voie A rend zero sur les 5 premieres armes | passer a la voie B |
| licence / redistribution des assets | — | **question a poser a l utilisateur AVANT d integrer** |
| noms de fichiers trompeurs | une image ne correspond pas a son nom | verification visuelle obligatoire, pas de confiance au nom |
| icone partagee entre armes | deux tags rendent le meme fichier | c est peut-etre CORRECT (le jeu le fait), le verifier avant de corriger |

## POURQUOI CE CHANTIER EST BON MARCHE

Il n a **aucune dependance** sur le reste : le decodeur n en a pas besoin, le branchement non plus,
et son echec ne coute rien d autre que le temps passe. Il peut donc etre lance quand il y a de la
place, et interrompu sans dette.
