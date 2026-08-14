# Plan — L'equipement actif a l'ecran : nommer juste, puis montrer l'etat

> Ecrit le 2026-08-14, APRES relecture de `RECETTE_LOADOUT_2026-07-27.md` (§2, §9, §13) et
> du decodeur. Il REMPLACE l'approche du lot du 14/08, qui a cherche ce que la recette
> portait deja. Execution sous `plan-execution`, branche `feat/v75`, commits par etape,
> PAS de push. Le reverse est FAIT : rien ici ne demande Ghidra.

## Correction de vocabulaire (question utilisateur du 14/08) — « saut », pas « traversee »

L'utilisateur demande : « pourquoi une traversee ? si ca fonctionne comme le reste on
parlerait plus de saut, non ? ». **Il a raison, et le code le dit** :

- `consumeMask` (traverse.go:1240) lit DEJA les deux encodages de masque : bit de garde 0 ->
  ensemble creux (R(3) compte + R(6) par index, donc au plus 7 composants), bit de garde 1 ->
  **masque dense R(64)**. Le masque dense n'est pas un cas non gere : il est implemente.
- Ce qui coince n'est donc pas la LECTURE du masque mais la LARGEUR des composants qui
  precedent celui qu'on veut. Un composant non porte = largeur inconnue = tout ce qui suit
  est perdu.
- Et le mecanisme de SAUT EXISTE : `unportedStubWidth` + `repairUnportedComponent`
  (frame_chain_infer.go:68) balaie la largeur du composant fautif (jusqu'a 640 bits), garde
  celles dont le re-decodage complet est propre, et retient l'alignement confirme EN AVAL.
  Son commentaire precise meme que « sur le chemin biped, position/velocite (i0/i1)
  precedent toujours les composants reparables (i55+) » — c'est exactement la zone de
  `i56`/`i57`.

**Donc la bonne formulation** : il faut SAUTER un ou plusieurs composants non portes situes
avant `i56`, avec une largeur determinee par balayage et validee par la coherence en aval.
Le present plan cherche POURQUOI ce saut n'aboutit pas aujourd'hui — pas comment ecrire une
traversee complete.

## Ce qui est DEJA acquis (ne pas re-chercher)

| acquis | ou | consequence |
|---|---|---|
| **La palette des capacites** : 1 detecteur, 2 mur, 4 grappin, 5 propulseur, 6 repulseur, **8 camouflage, 9 surbouclier, 11 translocateur**, 12 traqueur, 23 champ de reparation | RECETTE §13 (murmur3 des chaines + enumeration de l'executable) | les TROIS capacites voulues a l'ecran sont IDENTIFIEES |
| **L'etat actif est localise** : `i57` a `0x12E4`, `R(2)` | RECETTE §9 | plus rien a chercher dans le binaire |
| **Le compteur d'utilisations** : masque `R(3)`, puis 7 bits par emplacement arme, `0x7F` = plein ; deux encodages (continu `v/127`, ou discret quartet haut = charges) | RECETTE §9 | l'energie se lit, une fois le bon bit atteint |
| Le mecanisme de saut d'un composant non porte | `unportedStubWidth`, `repairUnportedComponent`, `resolveAlignment` | l'outil existe, il faut comprendre son echec |

## Etape 1 — CORRIGER LA TABLE FAUSSE (gain immediat, zero dependance)

> La table de production CONTREDIT la recette. Le §2 le dit lui-meme : « cette table est a
> moitie fausse, le `sofd` confirme 4 et 5, il CONTREDIT 3 et 6 ».

| `replay_labels.toml` (production) | RECETTE §13 (fait foi) |
|---|---|
| `3 = mur portatif` | rang 2 = mur de protection ; le rang 3 **n'est pas une capacite** |
| `6 = capteur de menace` | rang 6 = **repulseur** ; le detecteur est rang 1 |
| `4 = grappin` ✔ | rang 4 = grappin |
| `5 = propulseur` ✔ | rang 5 = propulseur |

- [x] 1.1 MESURER l'impact avant de corriger : sur le corpus de films, combien de lectures
      portent 3 et 6 ? → **605 lectures sur 1702, soit 35,5 %**, sur **12 films** des 19 qui
      rendent des lectures (balayage de **40 films** du cache, un processus par film,
      instrument versionne `i48_ability_width_test.go`, garde `ABILITY_FILM_ROOT`).
- [x] 1.2 Corriger la table selon le §13, en citant la recette dans le commentaire.
      → **3 et 6 RETIRES** (pas renumerotes : la clause de prudence de cet item s'applique,
      voir le journal) ; **4 et 5 conserves** — seuls a porter DEUX confirmations
      independantes (controle de groupe des triplets + palette `sofd`). Le commentaire du
      TOML porte la mesure, la regle qui tranche, et la condition de restauration.
- [x] 1.3 Verifier sur un match temoin : sur `000d5950`, **6 des 8 slots** de la verite
      terrain gardent un nom, et **ces 6 sont exactement ceux que le releve Theater
      confirme** (3 grappins, 3 propulseurs) ; les 2 slots contestes affichent desormais
      leur numero (« capacite inconnue (3) » / « (6) »). Verifie sur pieces via le golden
      d'assemblage du film de reference. Le controle A L'ECRAN reste la main de
      l'utilisateur (gate visuel).

Gate 1 : **PASSE**. Nombre de lectures corrigees publie (605/1702) ; aucune fiche ne montre
un nom que la recette contredit ; `go test ./internal/analysis/replay/ ./internal/games/...`
verts, `go vet` propre, `golangci-lint --new-from-merge-base=origin/main` 0 issue. Cote web :
aucun fichier touche (le rendu d'un index non nomme existait deja,
`ReplayInventoryRow.abilityText`).

### Journal de l'etape 1 (2026-08-14)

**Balayage** : 40 films, 4 s a 58 s par film, pic memoire **≤ 54 Mo** (un processus a la
fois, borne d'arret a 2 Go jamais approchee). **19 films rendent des lectures, 21 aucune**
(l'ancrage R1 n'y est jamais unique — decouverte deja notee le 14/08, non traitee).

    index      lectures   films
      3            270      12
      4            452      11
      5            348      11
      6            335      11
      7            297       8      jamais nomme
    TOTAL         1702      19      nommes apres correction : 800 (47,0 %)

**POURQUOI LA CLAUSE DE PRUDENCE A JOUE.** L'item 1.2 prevoyait deux issues ; la mesure en
designe une troisieme, plus forte, et elle est publiee ici parce qu'elle deplace l'etape 2 :
**le champ de 3 bits ne peut pas etre le rang de palette `sofd`**, pour quatre raisons
convergentes et toutes mesurees :

1. **Largeur** : 3 bits bornent a 7. Or la famille A place le camouflage au rang 8, le
   surbouclier au 9, le translocateur au 11, le champ de reparation au 23. Un champ qui ne
   sait pas ecrire ces rangs n'est pas ce rang.
2. **Rangs jamais vus** : sur 1702 lectures et 19 films, les valeurs **0, 1 et 2
   n'apparaissent JAMAIS**. Si le champ etait le rang famille A, le detecteur (1) et le mur
   (2) — deux equipements courants — seraient absents de tout le corpus.
3. **Valeur impossible** : la famille A dit que le rang 3 « n'est pas une capacite ». Le
   corpus le lit **270 fois sur 12 films**, porte par des joueurs. Une categorie nulle ne se
   porte pas.
4. **Elargissement deja refute** : le 14/08, `large6` / `suite6` / `aval6` rendent 0/8 contre
   la verite terrain la ou `prod3` rend 6/8, et `large6 = 16 + prod3` par construction du
   motif d'ancrage. Il n'y a ni porte ni champ de 6 bits a cette position.

**CE QUE CELA NE FAIT PAS.** Cela retire la CONTRADICTION, cela ne fournit pas la PREUVE.
Retirer le contradicteur de « 3 = mur » laisse ce nom adosse a **une seule observation**, et
la regle du depot dit qu'un temoignage isole ne vaut pas (RECETTE_LOADOUT §2). Le nom est
donc retire jusqu'a un second releve. **Le cout de l'erreur est asymetrique et c'est ce qui
tranche** : un numero prive d'un nom juste coute une information ; un nom faux affirme une
fausseté. L'etape 2 statue sur la regle rang -> nom avec ces chiffres en main.

## Etape 2 — QUELLE PALETTE POUR CE FILM ? (le vrai verrou de l'identite)

> RECETTE §13 : « la palette n'est PAS globale » — sur 46 equipements presents dans au moins
> deux palettes, **20 changent de rang** (grappin rang 4 ici, rang 8 ailleurs ; surbouclier
> rang 9 ici, rang 15 ailleurs). Et : « determiner quel `sofd` s'applique a un film donne est
> la question ouverte qui reste ». Sans cette reponse, un rang ne se traduit pas en nom.

- [~] 2.1 Recenser les `sofd` connus et leurs tables de rangs. **COUVERT PAR 2.2, ET DEPASSE** :
      il n'a pas fallu recenser les palettes du binaire, le FILM porte le rang. Le recensement
      utile est desormais celui des rangs REELLEMENT observes par film (ci-dessous).
- [x] 2.2 Chercher dans le FILM de quoi choisir. **TROUVE, EN LECTURE DE CODE, ET C'ETAIT DEJA
      DANS LE DEPOT** : `consumeBipedDesiredAbilitySet` (`components_biped_ability.go`, i48)
      reproduit `FUN_1406d0ff0` — `R(3)` compteur de rotation, `R(1)` porte, et **si la porte
      vaut 0, `R(6)` = le RANG DE PALETTE**. Le deserialiseur de production consomme ces
      6 bits pour rester aligne et les JETTE ; la sonde `SetAbilitySetHook` ne publiait que le
      R(3) et la largeur. **L'identite passait sous le nez du chantier depuis le debut.**
      → instrument versionne `internal/analysis/filmdec/i48_rank_test.go`, garde `I48_FILM`.
- [x] 2.3 REPLI MESURE — devenu MESURE PRINCIPALE : rangs observes par film (8 films).
      **La famille A n'est PAS universelle et le repli aurait ete FAUX** : trois films
      n'exposent QUE les rangs 19 a 22.
- [x] 2.4 Trancher et ECRIRE. → **LA PALETTE SE DETERMINE PAR LE FILM** (voie 2.2). La regle
      est ecrite au journal ci-dessous. La table de production reste celle de l'etape 1 : le
      nommage par rang exige un lecteur `i48` de production, qui n'existe pas encore — et
      nommer sans lui serait deviner.

Gate 2 : **PASSE**. La regle rang -> nom est ecrite et chiffree ; les films hors famille A
sont IDENTIFIES par la mesure au lieu d'etre presumes ; aucune capacite n'est devinee.

### Journal de l'etape 2 (2026-08-14) — le canal que personne n'avait lu

**LA MESURE** (un film par processus, 3 a 57 s, pic memoire ≤ 54 Mo) :

    film        records delta   masque ∋ i48   lus   rangs i48 transmis
    000d5950       171 851        92 (0,05 %)   92   19:18 20:22 21:26 22:16
    00502e52       182 876        82 (0,04 %)   82   19:22 20:17 21:8  22:18
    07aa428d       165 198        56 (0,03 %)   56   19:11 20:10 21:13 22:8
    00ba2e1c       240 645       206 (0,09 %)  206   1:31 2:25 4:34 5:20 6:36 10:21 23:35
    06dfe6d9       336 212       230 (0,07 %)  230   1:19 2:25 4:35 5:22 6:38 8:2 9:2
                                                     10:34 11:3 12:4 23:35
    084a804d       330 981       143 (0,04 %)  143   1:13 4:48 5:11 6:18 8:10 9:8 10:2
                                                     19:4 23:15 44:1
    00162144       141 051        39 (0,03 %)   39   2:14 4:9 9:2 10:10
    0014603f       118 054         0            —    (i48 jamais au masque)

**`i48` est lu a 100 % quand le masque l'annonce** — 0 illisible sur 748 lectures cumulees.
Ce n'est PAS le probleme de traversee d'`i56` : le composant est simplement rare (il est
transmis a peu pres une fois par vie, ce qui est exactement ce qu'il faut pour une fiche).

**CE QUE CELA ETABLIT — quatre resultats, dont deux renversent une conclusion ecrite.**

1. **Le champ d'image-cle est un rang de palette TRONQUE, et le 14/08 s'est trompe en
   classant `large6` « arithmetiquement trivial ».** Il l'etait comme calcul, il ne l'etait
   pas comme resultat : `large6 = 16 + prod3` rend {19,20,21,22} sur 000d5950, et `i48` —
   canal totalement independant, position derivee du decompile, lu par le deserialiseur de
   production — rend **exactement {19,20,21,22} sur le meme film**. Deux chaines
   independantes sur les memes valeurs : le rang est bien 19-22, pas 3-6.
2. **Le lecteur d'image-cle est STRUCTURELLEMENT AVEUGLE a 3/4 de la palette.** Les trois
   derniers bits du « motif 20 bits » d'ancrage ne sont pas une signature de structure : ce
   sont les **bits de poids fort du rang**, valeur fixe `010`. L'ancre ne peut donc matcher
   que les rangs **16 a 23**, et elle rend `rang - 16`. Trois consequences, toutes verifiees :
   les valeurs observees ne sortent jamais de 3-7 (rangs 19-23) ; les films « qui ne rendent
   AUCUNE lecture » (8 sur 14 le 14/08, 21 sur 40 ce jour) sont ceux dont aucun joueur ne
   porte un rang 16-23 — **PREDICTION VERIFIEE sur `00162144`, dont les rangs `i48` sont
   2, 4, 9, 10, tous hors plage** ; et les films « ou les 8 joueurs portent le meme
   equipement » sont un ARTEFACT — seuls les porteurs du rang 23 y sont visibles.
3. **La palette n'est pas la meme d'un film a l'autre, et c'est mesure, plus suppose.**
   `00ba2e1c`, `06dfe6d9`, `00162144`, `084a804d` exposent la signature de la **famille A**
   (1, 2, 4, 5, 6, 8, 9, 10, 11, 12, 23 — la table du §13 de la recette, au rang pres) ;
   `000d5950`, `00502e52`, `07aa428d` n'exposent **que 19 a 22** et aucun rang < 13. Le repli
   « on presume la famille A » de l'item 2.3 aurait donc produit des noms faux sur ces
   trois films.
4. **Les trois capacites que l'utilisateur veut a l'ecran EXISTENT dans le corpus, et sont
   mesurees** : camouflage (rang 8) 10 lectures sur `084a804d` et 2 sur `06dfe6d9` ;
   surbouclier (rang 9) 8 et 2 ; translocateur quantique (rang 11) 3 sur `06dfe6d9`. Le
   canal d'image-cle ne les verra JAMAIS (rangs hors 16-23) — le canal `i48`, lui, les voit.

**LA REGLE RANG -> NOM, ecrite comme le demande le gate 2** :

> Un nom ne se pose que si DEUX conditions tiennent ensemble : (a) la palette du film est
> identifiee — par la signature de ses rangs `i48`, pas par presomption ; (b) le rang porte
> un nom issu d'une source a controle de groupe (relevé terrain sur au moins deux porteurs)
> ou a double chaine (murmur3 + banque sonore, cf. §13). A defaut, le rang s'affiche comme
> rang. La table `replay_labels.toml` reste donc a deux entrees : elle est indexee par le
> champ d'image-cle TRONQUE, qui ne peut pas porter une palette a lui seul.

**CE QUE CELA COUTE DE NE PAS ALLER PLUS LOIN MAINTENANT, et c'est chiffre** : nommer 3, 6
et 7 demande un LECTEUR `i48` DE PRODUCTION (rang complet par vie, palette determinee par
film), pas une ligne de TOML. Hors perimetre de cette etape ; propose au registre.

## Etape 3 — ATTEINDRE `i57` : comprendre l'echec du saut

> Mesure du 14/08 : `i56` n'est lu que sur **176 records / 171 851** (0,10 %), 78 slots. Le
> mecanisme de saut existe pourtant. La question n'est pas « comment sauter » mais « pourquoi
> le saut echoue ici ».

- [ ] 3.1 INSTRUMENTER, ne pas supposer : sur un film temoin, pour les records dont le masque
      contient `i56`/`i57`, journaliser la CHAINE de composants decodee jusqu'a l'echec —
      quel composant fait desynchroniser, est-il porte, `repairUnportedComponent` est-il
      appele, et si oui pourquoi ne conclut-il pas (aucune largeur propre ? plusieurs
      largeurs candidates, donc alignement non unique ?).
- [ ] 3.2 Selon le verdict de 3.1, UNE de ces voies (pas les trois) :
      (a) le composant fautif est identifiable et sa largeur est FIXE -> la declarer
          (`SetUnportedStubWidth`) et mesurer le gain ;
      (b) plusieurs largeurs passent -> le depart est ambigu : elargir la confirmation aval
          (`resolveAlignment`) plutot que choisir au hasard ;
      (c) le composant a une largeur VARIABLE dependant du contenu -> alors il faut le porter
          vraiment, et la table des deserialiseurs ECS existante dit ou chercher.
- [ ] 3.3 MESURE DE SORTIE, la seule qui compte : le taux de records ou `i57` est lu, AVANT
      (0,10 %) et APRES. Un gain qui ne depasse pas quelques pourcents ne suffira pas a
      dater un usage — le dire alors, et s'arreter la.

Gate 3 : le taux publie ; l'instrument versionne et rejouable (garde par variable
d'environnement, saute en CI, patron des instruments `i54`/`i56` deja livres).

## Etape 4 — L'EFFET PLEIN FICHE (ne se code QUE si 2 et 3 passent)

> Demande utilisateur, precisee le 14/08 : l'effet porte sur TOUTE LA FICHE, pas un lisere.

- [ ] 4.1 Surbouclier (rang 9) -> fiche encadree DOREE. Camouflage actif (rang 8) -> effet de
      VERRE sur la fiche. Translocateur quantique (rang 11) -> BORDURE ANIMEE (bleu
      electrique vers jaune orange). Les autres capacites : aucun effet dedie (elles sont
      actives trop peu de temps — cahier des charges Notion).
- [ ] 4.2 Duree : celle que la donnee dit, jamais une duree inventee. Si l'etat actif n'est
      qu'un instant, l'effet dure une remanence FIXE et courte, ecrite en constante commentee.
- [ ] 4.3 Tokens semantiques uniquement, respect de `prefers-reduced-motion`, i18n FR+EN.

Gate 4 : verification a l'ecran par l'utilisateur, sur un match ou la verite terrain Theater
dit qu'une de ces trois capacites a ete utilisee.

## Regles dures (rappel, elles ont deja tranche ce chantier)

- **Aucun effet sans donnee mesuree.** Le lot du 14/08 a eu raison de ne rien afficher.
- **Aucun nom devine** : un rang non resolu s'affiche comme rang, pas comme capacite.
- Offline-pur et universel : pas de Cheat Engine, pas de capture runtime.
- Si l'etape 3 echoue, les etapes 1 et 2 valent quand meme — elles corrigent un FAUX affiche
  aujourd'hui, ce qui est plus important qu'un effet en plus.
