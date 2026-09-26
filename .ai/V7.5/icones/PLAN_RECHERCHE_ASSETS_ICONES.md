# PLAN DE RECHERCHE — les icones d armes pour l interface

> **PHASE 1 CLOSE le 2026-08-08** (branche `feat/v75-icones`). Resultats, chaine complete,
> pistes refutees et reserves : **`V7.5/icones/ETAT_DE_L_ART_ICONES.md`** — c'est lui qui fait
> foi desormais, ce plan reste comme trace du cadrage.
>
> **GATE 1 : PASSE, et au-dela de ce qui etait demande.** La correspondance n'est pas
> « verifiee sur 10 armes en regardant les images » : elle est **LUE dans le jeu** (champ
> `sprite index` du bloc `UI display info` du tag `weap`) et auto-validee arme par arme —
> **29 sur 29**. 168 PNG extraits, trois atlas : armes en contour, armes en silhouette, et
> l'atlas du **kill feed** (88 icones : vehicules, grenades lancees, pictogrammes).
>
> **PHASE 2 : NON COMMENCEE**, et c'est volontaire — elle attend le gate visuel de
> l'utilisateur (decision #4 du master plan). Rien n'est branche : ni `apps/web/`, ni
> `adapter_asset_urls.go`.
>
> Ce que le cadrage ci-dessous avait bien vu : la voie A etait la bonne, la chaine sonore
> etait bien inutilisable, et un nom de fichier n'est pas une source fiable. Ce qu'il n'avait
> pas prevu : le jeu porte lui-meme le lien arme -> icone, et il n'y avait donc pas a le
> deviner.

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

- [x] Un `weap` reference-t-il un asset d icone ? **OUI** — deux `bitm` communs aux 29 armes
      (`bc17adf1` contour, `e39747c8` silhouette), plus un troisieme qui varie : le reticule.
- [x] Remonter jusqu a l icone : **inutile de remonter**, le `weap` porte lui-meme l index
      (`sprite index` du bloc `UI display info`). 29 armes sur 29, auto-validees.
- [~] **Controle** : couvert autrement et mieux — chaque ligne est validee par le fait que le
      champ `sprite` du meme bloc porte bien l atlas attendu. Deux armes PEUVENT partager une
      icone, et c est correct : les deux Bandit, les deux epees et les deux Shock Rifle le font.

### Voie B — l executable, la ou le kill feed choisit son icone

Historique a connaitre : le rendu du kill feed **resout bien l arme pour choisir l icone**, et il
tourne au replay. Cette piste a ete exploree en juin 2026 et abandonnee — la chaine du handle vers
le type d arme s est revelee accessible **en live seulement**. C est cette impasse qui a pousse le
chantier vers le dead-state.

- [!] NON TRAITEE, et sans regret : la voie A a rendu la reponse. Le kill feed a bien sa table
      d icones — mais elle est dans un TAG (`bitd 8646f61a`), pas dans l executable, et elle a
      ete lue sans desassembler quoi que ce soit.
- [!] Chercher les chaines d assets d interface autour du widget de kill feed — sans objet :
      les chaines sont STRIPPEES en release, seuls des murmur3 subsistent. Ce sont eux qui ont
      ete craques.

### Voie C — les assets deja extraits par le depot

- [x] `static/abilities-assets/` : la question « comment ont-ils ete extraits » est restee sans
      reponse (aucune trace en depot ni au journal). Sans objet desormais — la voie A donne une
      chaine reproductible, ce que les capacites n avaient pas.
- [x] Reserve confirmee et ELARGIE : `Cremator.png` est bien le Cindershot, et le jeu ajoute sa
      propre divergence — l index 20 se nomme `heatwave` alors que le registre y lit
      `hinf_cindershot`, depuis le MEME tag. Un nom de fichier ne fait pas foi, un nom interne
      non plus sans confrontation.

**GATE 1 — PASSE** (voir l en-tete). Le critere initial etait « verifiee sur au moins 10 armes en regardant
les images, dont deux armes visuellement proches (BR75 / Bandit Evo) et une arme a variantes
(marteau antigrav) » ; le resultat obtenu est plus fort — la correspondance est LUE dans le
jeu pour les 29 armes. Et la reserve « BR75 / Bandit visuellement proches » etait justifiee :
je les avais **interverties** a l oeil, le champ du jeu a tranche.

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
