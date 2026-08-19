# Plan — le son de pose de la balise : reconstruire l'EVENEMENT Wwise

> Ecrit le 2026-08-19. Demande utilisateur : il a ecoute les 32 `.wem` de la banque
> `dcfaa487` et n'en reconnait AUCUN. Son hypothese : le son du jeu est un PAQUET Wwise —
> un evenement joue PLUSIEURS couches ensemble, avec gains et hauteurs — donc un `.wem`
> brut isole ne peut pas ressembler au son en jeu.
>
> Worktree FRERE `C:\Users\Guillaume\Projects\LevelUp-wt-balise-mix`, branche
> `wt/balise-mix`, base `feat/v75`. Jeu et scratchpad par chemins ABSOLUS, lecture seule
> sur le jeu. Contrat `plan-execution` + `CLAUDE.md`. Un commit par phase.
> Ni journal ni registre : les textes partent au compte rendu.

## Acquis — lus sur pieces avant d'ecrire ce plan

1. **L'outil existe et il sait deja lire la structure.** `apps/go-api/cmd/weapon-sounds`,
   mode `arbre` (`arbre.go`) : il distingue `RandomSequence` (UNE variante tiree) de
   `Blend`/`ActorMixer` (TOUS les enfants, donc des COUCHES SIMULTANEES), accumule le
   GAIN DE CHEMIN et la fourchette de variation, resout les `Switch` a leur etat par
   defaut. Il accepte `-sbnk <gid>` : il n'a pas besoin d'un `.pck` d'arme. **Rien de ce
   parseur n'est a reecrire** — ce lot le BRANCHE sur la chaine `eqip`.
2. **Ce que la passe 2 `eqip-banks` perd, et c'est exactement le defaut denonce.**
   `gestesDeBank` (`eqip_banks.go`) appelle `bk.wemsDeEvent(mot)` : un SAC PLAT de `.wem`.
   Un geste « 3 wem » peut donc etre 1 couche de 3 variantes OU 3 couches simultanees —
   le rapport ne le dit pas, et l'ecoute des 3 fichiers isoles ne tranche pas.
   `couchesDeEvent` (deja ecrit, deja utilise par le mode `arbre`) rend la difference.
3. **LES DEUX `eqip` DU TRANSLOCATEUR NE PARTAGENT AUCUNE BANQUE** (mesure relue ce jour
   dans `eqip_sons.json`, sortie de la passe 1 du lot du 18/08) :

        eqip a1344fc2  (l'APPAREIL qui vole)   -> banque dcfaa487 SEULE, 9 snd!, 11 gestes
        eqip 730dc70f  (la BALISE posee)       -> banques 15c5b355, b29ac6de, de65048f

   **Les 32 fichiers ecoutes viennent tous de `dcfaa487`, la banque de l'APPAREIL.**
   Les trois banques de l'objet du monde — celui que le film cree, celui que le rejeu
   dessine — **n'ont jamais ete extraites ni ecoutees** (le scratchpad `wav2` porte
   6 banques : `dcfaa487`, `7bd0883c`, `7acb11cc`, `92c830f5`, `60b0f79c`, `5724312f`).
   L'hypothese « le son de pose vit sur l'AUTRE eqip » n'est donc pas seulement plausible :
   la moitie de la chaine n'a pas ete regardee.
4. **La couverture de `dcfaa487` est partielle** : 70 `.wem` embarques, 11 gestes qui n'en
   designent que 32. **38 `.wem` de cette banque ne sont atteints par aucun evenement
   resolu** — soit ils appartiennent a des evenements qu'aucun `snd!` d'`eqip` ne designe,
   soit la resolution par intersection de mots les rate. Le mode `arbre` enumere TOUS les
   Events d'une banque : c'est la mesure qui tranche.
5. **Les banques de la balise posee sont PARTAGEES** : `15c5b355` (21 `eqip`, 20 wem,
   1 geste), `b29ac6de` (17 `eqip`, 8 wem, 2 gestes de 4 wem), `de65048f` (33 `eqip`,
   8 wem, 1 geste d'1 wem — le plan du 18/08 la lit comme « objet d'equipement cree dans
   le monde »). Partagee ne veut pas dire muette : pour NOMMER, le partage disqualifie ;
   pour SONNER, un son generique de deploiement est precisement ce qu'on cherche.
6. **La recette des armes tranche l'assemblage** (`.ai/V7.5/RECETTE_SONS_ARMES.md`, section
   4) : gain = somme du chemin, `RandomSequence` = une variante uniforme, `Blend` sans
   courbe et `ActorMixer` = tous les enfants, **zero delai mesure sur les armes**. Le delai
   d'objet (`AkPropID_InitialDelay`, propriete 17) EST lu par `proprietes.go` mais n'est
   PAS propage le long du chemin (`etatChemin` ne porte que gain et variation) : pour un
   equipement, un decalage de couche est credible, donc ce lot le propage et le MESURE
   avant de s'en servir.
7. **Le delai d'ACTION reste le trou de preuve connu** (recette section 4 : « offset non
   valide »). Ce lot ne l'invente pas : s'il n'est pas lisible avec un controle de
   plausibilite, il est declare non lu et les couches s'empilent a t=0, comme pour les
   armes.

## Regles dures de ce lot

Un son vient du jeu ou n'existe pas. Une structure vient du parseur ou n'est pas affirmee.
Chaque candidat reconstruit publie SA RECETTE (couches, gains, decalages) — un mix sans
recette n'est pas soumis a l'ecoute. Aucun seuil ne bouge apres la mesure. Zero fix hors
perimetre. Lecture seule sur le jeu ; jamais `git add -A`, jamais `git stash`, jamais de
push. `GOCACHE` dedie au scratchpad, un seul `go` a la fois.

## Phase 1 — STRUCTURE : la banque rend ses couches, pas un sac de `.wem`

- [x] 1.1 Mode Go **`eqip-arbre`** dans `cmd/weapon-sounds` : pour une liste de banques
      (`-banks`), enumerer **TOUS** les Events avec, pour chacun, ses COUCHES
      (`couchesDeEvent` : type de noeud, `.wem` candidats, gain de chemin, variation), et
      marquer ceux qu'un `snd!` de la passe 1 DESIGNE (lecture de `-json`, l'echange
      existant). Sortie JSON + tableau lisible. Reutilise `bankParIdentifiant`,
      `parserBank`, `couchesDeEvent` : aucun parseur nouveau.
- [x] 1.2 **Couverture** : par banque, les `.wem` embarques qu'aucun Event n'atteint —
      le chiffre qui dit si l'ecoute du 18/08 portait sur toute la banque ou sur un tiers.
- [x] 1.3 **Delai de couche** : propager `DelaiS` dans `etatChemin` et le rendre par
      couche ; MESURER combien de noeuds en portent un non nul sur les banques cibles.
      Zero mesure = les couches s'empilent a t=0 et le plan le dit.
- [x] 1.4 Lancer sur les **cinq** banques du translocateur (`dcfaa487`, `15c5b355`,
      `b29ac6de`, `de65048f`, et `92c830f5` en temoin negatif : elle appartient a un autre
      `eqip`) ; module `pc/globals` (7,24 Go) charge SEUL.
- [x] 1.5 **Temoin de methode** : rejouer le cas simple reussi `repair_field`
      (`5724312f`, evenement -> 3 wem) et verifier que la structure rendue explique le
      resultat livre (3 variantes d'un point de choix, pas 3 couches). Une mesure qui ne
      retrouve pas un cas connu ne vaut rien.

**Gate 1 : PASSE** (2026-08-19). `go build ./cmd/weapon-sounds/` OK, `go vet` exit 0,
`golangci-lint run ./cmd/weapon-sounds/...` **0 issues**. Sortie :
`scratchpad/balise_structure.json` + `.log`, 175 `.wem` ecrits (toutes les banques,
orphelins compris).

**CE QUE LA MESURE DIT, ET ELLE CONTREDIT EN PARTIE L'HYPOTHESE DE DEPART :**

        banque     eqip  events  wem  orphelins  events multi-couches
        dcfaa487     2      23    70      2              0
        15c5b355    21      15    20      0              2   <- balise POSEE
        b29ac6de    17       2     8      0              0   <- balise POSEE
        de65048f    33       7     8      0              0   <- balise POSEE
        5724312f     1      14    31      0              1   (temoin repair_field)
        92c830f5     1      11    38      0              2   (temoin negatif)

1. **DANS `dcfaa487` — LA BANQUE ECOUTEE — AUCUN EVENEMENT N'EST MULTI-COUCHES.** Les
   23 evenements sont tous « une couche, un son tire parmi N » (N = 1 a 6). L'hypothese
   « le son de cette banque est un empilement que le `.wem` isole ne rend pas » est donc
   **REFUTEE POUR CETTE BANQUE** : ce que le jeu y joue EST un fichier unique.
2. **CE QUI EXPLIQUE L'ECHEC D'ECOUTE EST AILLEURS, ET C'EST UN COMPTE.** Les 11 gestes du
   18/08 (ceux qu'un `snd!` designe) ne couvrent que **11 evenements sur 23** et 32 `.wem`
   sur 70. **12 evenements de cette meme banque n'ont jamais ete entendus**, et 2 `.wem`
   ne sont atteints par aucun evenement.
3. **LES EVENEMENTS MULTI-COUCHES EXISTENT, ET DEUX SONT DANS LA BANQUE DE LA BALISE
   POSEE** (`15c5b355`, jamais extraite avant ce lot) :
   `044005ec` = `303633458` (0 dB) + `1062674912` (-23 dB) ;
   `92206f7d` = `405210764` (-3 dB) + `636465689` (+3 dB).
4. **ZERO DELAI SUR LES SIX BANQUES** (`DelaiNoeud` vide partout) : l'empilement a t=0
   n'est pas une simplification ici non plus, c'est la mesure. Aucun `adelay` au mixage.
5. **Temoin 1.5 PASSE** : les deux evenements designes de `repair_field` (`15b73ee0` et
   `8ed46d21`, `snd! 22c2323a`) rendent tous deux « 1 couche, un son parmi 3 » avec les
   MEMES trois `.wem` (`894865279`, `899552962`, `1001730562`) — exactement les trois
   fichiers livres. La structure explique le resultat connu.
6. **Un negatif utile en passant** : le geste generique `de65048f` / `92491129`
   (`snd! 7ff6244a`, 33 `eqip`) porte un gain de chemin de **-96 dB**. Le « son generique
   d'objet cree dans le monde » que le plan du 18/08 envisageait de brancher est donc
   **muet par construction**.

## Phase 2 — LES DEUX CHAINES, gestes nommes par leur role

- [ ] 2.1 Extraire les `.wem` **de toutes les banques cibles** (mode `embarques`/`-emb`),
      y compris les 38 non couverts de `dcfaa487` et les 36 des trois banques de la balise
      posee ; convertir en `.wav` par `vgmstream-cli` dans le scratchpad (jamais le depot).
- [ ] 2.2 Pour CHAQUE evenement des deux `eqip` : dire **quels wem, joues comment**
      (simultanes / sequence / un parmi N) et **avec quels gains**. Tableau au compte rendu.
- [ ] 2.3 Attribuer un ROLE plausible a chaque geste (pose, retour/rappel, activation,
      boucle, fin) — chaque attribution avec l'element qui la porte (nombre de variantes,
      duree, structure, banque d'origine), pas une intuition. Ce qui n'a pas d'element
      reste « indetermine » et le dit.

**Gate 2** : chaque `.wav` cite existe sur le disque avec sa duree mesuree ; aucun role
affirme sans son element.

## Phase 3 — RECONSTRUCTION : 2 a 5 candidats de POSE, chacun avec sa recette

- [ ] 3.1 Choisir les candidats par la STRUCTURE (evenements multi-couches en tete), pas
      par le nom de fichier. Un candidat = un evenement reconstruit.
- [ ] 3.2 Mixer par `ffmpeg` : `amix` des couches, gain de chemin applique en `volume`,
      decalage par `adelay` quand la phase 1 en a mesure un, boucles tronquees a UNE
      occurrence, normalisation identique au lot R2-S (crete vraie <= -1,0 dBTP).
- [ ] 3.3 **Publier la recette de chaque candidat** : couches, gains, decalages, et la
      commande exacte. Un candidat sans recette lisible ne part pas a l'ecoute.
- [ ] 3.4 Joindre les **variantes** quand un point de choix en offre plusieurs : le tirage
      change le son, et l'oreille doit pouvoir refuser le tirage plutot que le geste.

**Gate 3** : chaque `.wav` de candidat existe, sa duree et sa crete sont mesurees, sa
recette est ecrite.

## Phase 4 — PAGE D'ECOUTE v2

- [ ] 4.1 `ecoute_balise_v2.html` dans le scratchpad, patron de `ecoute_balise.html` :
      temoin capteur en tete, candidats RECONSTRUITS avec leur recette affichee sous
      chaque ligne, radio de selection, texte copiable. Audio embarque en base64.
- [ ] 4.2 Section « gestes bruts » conservee pour les banques JAMAIS ecoutees (celles de
      la balise posee) : ce sont des candidats a part entiere.
- [ ] 4.3 Verifier la page ouverte (structure, lecture d'au moins un candidat) avant de
      la rendre.

**Gate 4** : la page s'ouvre, chaque bouton a sa source, le texte de selection se copie.

## Phase 5 — VERDICT ecrit

- [ ] 5.1 Si la structure montre que le son de pose n'est dans AUCUNE de ces banques :
      le dire, avec la piste suivante CHIFFREE (quelle banque commune, combien d'`eqip`
      l'atteignent, combien de `.wem`, quel cout de balayage).

## Journal d'execution

- 2026-08-19 — plan ecrit.
- 2026-08-19 — **phase 1 CLOSE**. Mode `eqip-arbre` ecrit, delai propage et mesure a zero,
  structure des six banques rendue. L'hypothese « paquet multi-couches » tombe sur
  `dcfaa487` et se deplace sur `15c5b355` (deux evenements a deux couches, jamais extraits).

## Decouvertes — notees, NON traitees
