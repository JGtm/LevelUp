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

- [ ] 1.1 MESURER l'impact avant de corriger : sur le corpus de films, combien de lectures
      portent 3 et 6 ? (chiffre du 14/08 : les index observes sont 3,4,5,6 et un 7 sur deux
      films). C'est le nombre de fiches qui affichent aujourd'hui un nom FAUX.
- [ ] 1.2 Corriger la table selon le §13, en citant la recette dans le commentaire.
      **Attention** : ne pas se contenter de renumeroter — si l'observation dit 3 et que la
      palette dit « 3 n'est pas une capacite », alors soit le champ lu n'est pas le rang de
      palette, soit la palette du match n'est pas la famille A. C'est l'etape 2 qui tranche ;
      tant qu'elle n'a pas parle, une lecture douteuse s'affiche SANS NOM plutot qu'avec un
      faux.
- [ ] 1.3 Verifier a l'ecran sur un match temoin : la capacite affichee correspond-elle a ce
      que la verite terrain Theater du 2026-07-27 a releve (8/8 sur `000d5950`) ?

Gate 1 : le nombre de lectures corrigees est publie ; aucune fiche ne montre un nom que la
recette contredit ; tests Go + web verts.

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
