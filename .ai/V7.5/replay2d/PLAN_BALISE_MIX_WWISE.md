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

- [x] 2.1 Extraire les `.wem` **de toutes les banques cibles** (mode `embarques`/`-emb`),
      y compris les 38 non couverts de `dcfaa487` et les 36 des trois banques de la balise
      posee ; convertir en `.wav` par `vgmstream-cli` dans le scratchpad (jamais le depot).
      **106 `.wav`** dans `scratchpad/wav_balise/` (70 + 20 + 8 + 8), durees dans
      `durees_balise.txt`. Le temoin `5724312f` et le temoin negatif `92c830f5` restent en
      `.wem` : leur role s'arrete a la structure.
- [x] 2.2 Pour CHAQUE evenement des deux `eqip` : dire **quels wem, joues comment**
      (simultanes / sequence / un parmi N) et **avec quels gains**. Tableau au compte rendu.
      Fait : 47 evenements tabules (23 + 15 + 2 + 7), avec gain de chemin et fourchette de
      duree. **Aucun evenement de `dcfaa487` n'a plus d'une couche** ; les seuls multi-couches
      du perimetre balise sont les deux de `15c5b355`.
- [x] 2.3 **ROLES — ce que les elements portent, et ou ils s'arretent.**

      **`b29ac6de` — le couple le plus proche d'une POSE.** Deux evenements, PAS UN DE PLUS,
      tous deux designes par `7b5cbe75` — le `snd!` accroche DIRECTEMENT au tag `eqip`
      (sans passer par `effe`), celui que 21 objets d'equipement partagent. Quatre variantes
      chacun, toutes entre 0,41 et 0,48 s, gains +6 dB et +3 dB. Elements : la paire, le
      `snd!` direct, la brievete, l'uniformite des durees. **Role : la paire d'un objet
      d'equipement — apparition et disparition.** Lequel des deux est la pose n'est PAS
      determine par la structure : les deux ont la meme forme.

      **`15c5b355` — l'autre `snd!` direct, et les deux seuls empilements.** `c73036e4`
      (designe par `725186aa`, le second `snd!` direct, 2 variantes de 0,59 et 0,80 s) ;
      `044005ec` et `92206f7d`, **les deux seuls evenements a deux couches de tout le
      perimetre balise**, non designes, 1,34 s et 1,94 s. Element pour les seconds : une
      couche pleine plus une couche attenuee (-23 dB) ou renforcee (+3 dB) — le profil d'un
      impact suivi d'une resonance. **Role : indetermine entre pose et activation** ; la
      structure ne nomme pas, elle classe.

      **`de65048f` — ecartee par la mesure.** Son unique evenement designe (`92491129`,
      `snd! 7ff6244a`, 33 `eqip`) porte **-96 dB** de gain de chemin : le « son generique
      d'objet cree dans le monde » est muet. Les six autres evenements ne sont designes par
      aucun `snd!` d'equipement.

      **`dcfaa487` — la banque de l'APPAREIL, et ses durees le disent.** Gestes de 0,32 s a
      **6,77 s** (les deux orphelins), sept evenements au-dela de 3 s. Un objet pose au sol
      n'a pas besoin de six secondes ; un voyage, une charge, une boucle de vol, si.
      **Role : le trajet et l'appareil, pas la pose** — ce qui est coherent avec le refus
      d'ecoute du 19/08 sur ses 32 fichiers.

**Gate 2 : PASSE.** Les 106 `.wav` existent, chacun avec sa duree mesuree par `ffprobe` ;
chaque role ci-dessus cite l'element qui le porte, et les deux indeterminations sont
ecrites comme telles.

## Phase 3 — RECONSTRUCTION : 2 a 5 candidats de POSE, chacun avec sa recette

- [x] 3.1 Choisir les candidats par la STRUCTURE (evenements multi-couches en tete), pas
      par le nom de fichier. Un candidat = un evenement reconstruit.
      **CINQ candidats, douze fichiers** (un par tirage possible), dans
      `scratchpad/candidats_balise/` ; recettes dans `candidats_recettes.txt`.
- [x] 3.2 Mixer par `ffmpeg` : `amix` des couches, gain de chemin applique en `volume`,
      decalage par `adelay` quand la phase 1 en a mesure un, boucles tronquees a UNE
      occurrence, normalisation identique au lot R2-S (crete vraie <= -1,0 dBTP).
      **AUCUN `adelay` : la phase 1 mesure zero delai.** Aucune boucle a tronquer non plus —
      pas un evenement du perimetre n'en porte. Outil : `scratchpad/mix_candidat.sh`,
      reutilisable (sortie puis couches `fichier:gain_dB`), qui imprime sa propre recette.
- [x] 3.3 **Publier la recette de chaque candidat** : couches, gains, decalages, et la
      commande exacte. Faite par le script, une recette par fichier produit :

          candidat                            couches (gain de chemin)                duree
          A  b29ac6de / 0b2a938e  4 tirages   1 couche, +6 dB                    0,41-0,48 s
          B  b29ac6de / fb25cbdd  4 tirages   1 couche, +3 dB                    0,41-0,48 s
          C  15c5b355 / c73036e4  2 tirages   1 couche, 0 dB                     0,59-0,80 s
          D  15c5b355 / 044005ec  1 tirage    303633458 (0 dB) + 1062674912 (-23 dB)  1,34 s
          E  15c5b355 / 92206f7d  1 tirage    405210764 (-3 dB) + 636465689 (+3 dB)   1,94 s

      **CE QUE LA RECETTE DOIT AVOUER** : sur A, B et C — une seule couche — appliquer le
      gain de chemin puis normaliser a -1,0 dBTP revient au fichier normalise. Le gain n'y
      change que le niveau RELATIF a d'autres couches, et il n'y en a pas. Ces trois
      candidats sont donc des gestes bruts remis au bon niveau, pas des melanges ; seuls
      D et E sont des reconstructions au sens strict.
- [x] 3.4 Joindre les **variantes** quand un point de choix en offre plusieurs : fait,
      A et B en portent quatre chacun, C deux. Le tirage est uniforme (regle prouvee du
      chantier armes), aucune variante n'est privilegiee.

**Gate 3 : PASSE.** Les 12 `.wav` existent, chacun avec sa duree et sa crete avant/apres
correction consignees dans `candidats_recettes.txt`.

## Phase 4 — PAGE D'ECOUTE v2

- [x] 4.1 `ecoute_balise_v2.html` dans le scratchpad, patron de `ecoute_balise.html` :
      temoin capteur en tete, candidats RECONSTRUITS avec leur recette affichee sous
      chaque ligne, radio de selection, texte copiable. Audio embarque en base64.
      **692 Ko, 45 sons** (1 temoin + 12 candidats + 32 gestes), opus 64 kbps mono.
- [x] 4.2 Section « gestes bruts » conservee pour les banques JAMAIS ecoutees (celles de
      la balise posee) : ce sont des candidats a part entiere. **32 gestes** : les 12 de
      `15c5b355` hors candidats, les 6 de `de65048f`, et **les 14 de `dcfaa487` que la
      selection du 19/08 laissait de cote** (12 evenements sans `snd!` designant + 2 sons
      hors de tout evenement). Les 11 evenements deja ecoutes et refuses n'y sont PAS.
- [x] 4.3 Verifier la page ouverte (structure, lecture d'au moins un candidat) avant de
      la rendre. **Verifie sans navigateur** (la verification visuelle revient a
      l'utilisateur, regle du chantier) : 45 balises `audio`, 45 boutons tous apparies a
      un identifiant existant, 0 source vide, 44 radios, 5 recettes affichees, champ
      copiable et bouton de copie presents. Deux sources embarquees ont ete redecodees
      depuis leur base64 et relues par `ffprobe` (0,44 s et 0,61 s).

**Gate 4 : PASSE.** La page s'ouvre sur un document complet, chaque bouton a sa source,
le texte de selection se copie.

## Phase 5 — VERDICT ecrit

- [x] 5.1 **LE SON DE POSE N'EST PAS OU ON LE CHERCHAIT, ET L'HYPOTHESE DU MELANGE N'EST
      VRAIE QU'A MOITIE.** Deux resultats, chacun avec son chiffre.

      **(a) L'hypothese « paquet Wwise » est REFUTEE sur la banque ecoutee.** Les
      23 evenements de `dcfaa487` sont tous mono-couche : ce que le jeu y joue est un
      `.wem` unique, tire parmi 1 a 6 variantes. Aucun melange n'aurait rapproche ces
      fichiers du son du jeu. **Elle est VRAIE ailleurs** : `15c5b355` porte deux
      evenements a deux couches, et `92c830f5` (temoin negatif) en porte un a trois — le
      mecanisme existe bien dans le jeu, il n'est simplement pas a l'oeuvre sur l'appareil
      du translocateur.

      **(b) Ce qui a ete ecoute ne couvrait ni la bonne banque ni toute la banque.** Les
      32 fichiers du 19/08 sont 11 evenements sur les 23 de `dcfaa487`, et `dcfaa487` est
      la banque de l'`eqip` `a1344fc2` — **l'appareil qui vole**. L'objet du monde, celui
      que le film cree et que le rejeu dessine, est l'`eqip` `730dc70f` : il atteint
      `b29ac6de`, `15c5b355` et `de65048f`, et **aucune des trois n'avait ete extraite**.
      La page v2 les rend toutes les trois, plus les 14 gestes non entendus de la
      quatrieme.

      **SI L'OREILLE REFUSE AUSSI CES 44 SONS, VOICI LA PISTE SUIVANTE ET SON COUT.** Le
      son ne serait alors dans aucune banque atteignable par la chaine
      `eqip -> effe -> snd! -> sbnk`, donc dans une banque d'INTERFACE ou de gameplay
      generique qu'aucun tag `eqip` ne reference. Denominateur mesure : le module
      `pc/globals` porte **1 305 tags `sbnk`** ; la chaine d'equipement n'en atteint que
      **17** (1,3 %). Le mode `eqip-arbre` traite une banque nommee en quelques secondes
      une fois le module charge (~2 min de chargement, ~7,2 Go de RAM) : un balayage des
      1 305 banques avec enumeration des evenements est de l'ordre de **20 a 30 minutes en
      un seul processus**, et rend un tri par duree et par structure. C'est le seul
      elargissement qui ne postule rien — et il n'a PAS ete fait ici, faute d'un critere
      de selection qui vaille mieux que « ecouter 1 305 banques ».

## Journal d'execution

- 2026-08-19 — plan ecrit.
- 2026-08-19 — **phase 1 CLOSE**, commit `8386b1e94`. Mode `eqip-arbre` ecrit, delai propage
  et mesure a zero, structure des six banques rendue. L'hypothese « paquet multi-couches »
  tombe sur `dcfaa487` et se deplace sur `15c5b355` (deux evenements a deux couches, jamais
  extraits).
- 2026-08-20 — **phase 2 CLOSE**, commit `70a5b9c8e`. 106 `.wav` extraits des deux chaines,
  47 evenements tabules, roles portes par leurs elements, deux indeterminations ecrites.
- 2026-08-20 — **phase 3 CLOSE**, commits `7046eff33` et `8242ecfe4`. Cinq candidats,
  douze fichiers, recette publiee par fichier, aveu sur les trois candidats mono-couche.
- 2026-08-20 — **phases 4 et 5 CLOSES**. Page `ecoute_balise_v2.html` (692 Ko, 45 sons)
  verifiee sans navigateur ; verdict ecrit avec le cout de la piste suivante.

## Decouvertes — notees, NON traitees

1. **`dcfaa487` est atteinte par DEUX `eqip`, pas un** : `a1344fc2` et **`10bea582`**. Le
   plan du 18/08 ne nommait que le premier. Le second n'a pas ete cartographie ici.
2. **Deux `.wem` de `dcfaa487` sont hors de tout evenement** (`532684898` a 6,22 s et
   `708804123` a 6,77 s) : les plus longs de la banque, et aucun Event de la banque ne les
   atteint. Soit ils sont joues depuis une autre banque, soit la lecture des Events en rate
   une forme. Non tranche.
3. **Le « son generique d'objet cree dans le monde » est muet** : `de65048f` / `92491129`
   porte -96 dB de gain de chemin. Le plan `PLAN_EQUIPEMENTS_MANQUANTS_SONS` envisageait de
   le brancher (decouverte 3 de ce plan) : la mesure le disqualifie.
4. **Le rang 10 anonyme (`92c830f5`) porte le seul evenement a TROIS couches du perimetre**
   (`a239ae5f`, trois points de choix de 3 variantes, gains +9, +8 et +3 dB). Le jour ou cet
   objet se nommera, son son ne sera pas un `.wem` isole non plus.
5. **`15c5b355` porte quinze evenements pour vingt `.wem` et vingt-et-un `eqip`** : c'est la
   banque generique des objets d'equipement poses. Une identification y vaudrait pour TOUS
   les objets poses, pas seulement la balise — ce qui change la valeur du vote a venir.
