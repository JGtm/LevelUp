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

- [ ] 2.1 Recenser les `sofd` connus et leurs tables de rangs (les 12 du `glpa` sont deja
      mesures) ; identifier la « famille A » (d91958af / 03137359 / 13c097ed), dont le §13
      dit que le prefixe des rangs 0 a 9 est RIGOUREUSEMENT IDENTIQUE et que c'est la seule
      famille compatible avec un jeu de capacites de joueur.
- [ ] 2.2 Chercher dans le FILM de quoi choisir : le `sofd` employe est designe a l'execution
      (composant a `unite+0x268` d'apres la recette). Est-ce que quelque chose de cette
      designation est SERIALISE dans le film (un identifiant, un index, un hash) ? C'est la
      question decisive, et elle se pose en lecture de code avant toute mesure.
- [ ] 2.3 REPLI MESURE, si 2.2 echoue : la famille A a un prefixe identique sur les rangs
      0 a 9. Si TOUS les films du corpus n'exposent que des rangs < 10, alors la table de la
      famille A suffit — et ce n'est plus une hypothese mais une mesure a publier (rangs
      observes, par film, avec leur frequence). Le rang 7 vu sur deux films BTB Fiesta le
      14/08 devient alors un cas a instruire, pas une anomalie.
- [ ] 2.4 Trancher et ECRIRE : soit la palette se determine (2.2), soit elle est presumee
      famille A avec sa preuve de couverture (2.3), soit on ne nomme pas et on l'assume.

Gate 2 : la regle de resolution rang -> nom est ecrite et justifiee par des chiffres ; les
capacites hors famille A (le cas 51e60c5a) sont soit exclues soit traitees, jamais devinees.

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
