# Plan — Le rang de capacite par `i48` : nommer juste, et debloquer l'effet actif

> Ecrit le 2026-08-14 apres la mesure du lot « equipement actif » (RECETTE_LOADOUT §14).
> Decision utilisateur : « oui bonne idee, planifie bien ». Execution sous `plan-execution`,
> branche `feat/v75`, commits par etape, PAS de push. Le reverse est FAIT — rien ici ne
> demande Ghidra ni capture runtime.

## Le fait qui commande tout le plan

**Le rang de palette est dans le film depuis le debut, et le deserialiseur le jette.**
`consumeBipedDesiredAbilitySet` lit `R(3)` compteur, `R(1)` porte, **`R(6)` identite** — il
consomme les 6 bits pour rester aligne puis les abandonne ; le hook de production
(`abilitySetHook`) ne publie que le `R(3)` et la largeur. Mesure de l'instrument
(`filmdec/i48_rank_test.go`, garde `I48_FILM`) : **748 lectures sur 8 films, ZERO illisible**,
environ une transmission par vie — ce qui suffit largement a une fiche.

Et le canal utilise aujourd'hui (le champ de 3 bits des images-cles) est **tronque** : son
motif d'ancrage se termine par `010`, qui sont les bits de POIDS FORT du rang. Il ne peut donc
voir que les rangs **16 a 23**, et il rend `rang − 16`. Cela explique enfin trois anomalies :
les valeurs ne sortent jamais de 3-7 ; **21 films sur 40 ne rendent aucune lecture** ; et les
films « ou les huit joueurs portent le meme equipement » sont un artefact.

**Ce que le canal `i48` ouvre** : les rangs 8 (camouflage), 9 (surbouclier) et 11
(translocateur) — les TROIS capacites que l'utilisateur veut voir actives — sont OBSERVES dans
le corpus (`084a804d` 8:10 9:8 · `06dfe6d9` 8:2 9:2 11:3). Le canal actuel ne les verra
jamais ; celui-ci les voit, avec le rang complet.

## Etape 1 — Publier le rang (le decodeur cesse de le jeter)

- [ ] 1.1 Elargir le hook de production pour qu'il porte le RANG COMPLET (`R(6)`) en plus de
      ce qu'il publie deja. Ne pas casser ses appelants : la valeur `R(3)` et la largeur
      restent. L'instrument `i48_rank_test.go` devient le temoin de non-regression du hook.
- [ ] 1.2 Faire remonter le rang jusqu'a l'inventaire du rejeu (`Inventory.A` ou un champ
      voisin — DECIDER et ECRIRE lequel, en regardant ce que la fiche consomme aujourd'hui).
      **Attention** : le rang complet et l'index tronque ne sont PAS la meme grandeur —
      publier le rang complet dans un champ qui portait l'index tronque est un changement de
      SENS, pas seulement de valeur. Si le champ est servi par l'API : bump `SchemaVersion`,
      `openapi-gen` + `generate-types` dans le meme commit, `go test ./contracttest/`
      OBLIGATOIRE, et re-cuisson des artefacts existants (~23, CLI `backfill-replay
      --only-existing`, un a la fois).
- [ ] 1.3 Le canal d'image-cle reste-t-il ? Le trancher explicitement : soit il devient un
      REPLI documente (il ne voit que 16-23, donc il ne sert que si `i48` manque sur une vie),
      soit il est retire. Pas de troisieme voie, et pas de silence : deux canaux qui rendent
      des grandeurs differentes sous le meme nom, c'est le defaut qui a coute ce chantier.

Gate 1 : le rang complet apparait pour les 8 films de l'instrument ; les rangs 8, 9, 11 sont
VISIBLES la ou la mesure les annonce (`084a804d`, `06dfe6d9`) ; aucun test existant ne rougit.

## Etape 2 — La palette du film (le verrou de la traduction rang -> nom)

> Mesure §14 : la palette VARIE. Signature famille A (rangs 1,2,4,5,6,8,9,10,11,12,23) sur
> `00ba2e1c`, `06dfe6d9`, `00162144`, `084a804d` ; rangs **19-22 exclusivement**, aucun < 13,
> sur `000d5950`, `00502e52`, `07aa428d`. Presumer la famille A donnerait des noms FAUX sur
> ces trois films.

- [ ] 2.1 Chercher d'abord une DESIGNATION dans le film (lecture de code, pas de calcul) : le
      §13 dit que le `sofd` est choisi a l'execution par un composant a `unite+0x268`. Quelque
      chose de ce choix est-il serialise ? C'est la reponse propre, et elle se cherche avant
      toute heuristique.
- [ ] 2.2 A defaut, IDENTIFIER LA PALETTE PAR SA SIGNATURE : l'ensemble des rangs observes
      dans un film le classe (famille A contient 1,2,4,5,6,8-12,23 ; l'autre groupe n'expose
      que 19-22). Regle a ECRIRE avec ses chiffres, et surtout : **une signature ambigue ou
      inconnue ne nomme RIEN** — le rang s'affiche seul, comme aujourd'hui.
- [ ] 2.3 Table indexee par **(palette, rang)**, en DONNEE (TOML du titre, jamais en dur), et
      non plus par rang seul. La table actuelle (`replay_labels.toml [abilities]`) devient un
      cas particulier : la palette famille A. Les entrees `4`/`5` conservees a l'etape 1 du
      lot precedent doivent etre RELUES a cette lumiere — elles etaient indexees par l'index
      TRONQUE, pas par le rang.

Gate 2 : la regle de classement est ecrite et justifiee ; sur les 8 films de l'instrument, le
classement est publie film par film ; aucun film ambigu ne recoit de nom.

## Etape 3 — Nommer les rangs qui manquent (sans jamais deviner)

- [ ] 3.1 Rangs 19 a 22 : NON CASSES par les chaines murmur3 (le §13 le dit deja de la plage
      13-22). Mais le releve Theater du 2026-07-27 les nomme INDIRECTEMENT sur `000d5950` —
      19 mur, 20 grappin, 21 propulseur, 22 capteur — dont **20 et 21 avec un controle de
      groupe**. Statuer : ce qui a deux appuis independants est nomme ; ce qui n'en a qu'un
      reste un rang. La regle du depot (§2 : « un temoignage isole ne vaut pas ») s'applique
      telle quelle.
- [ ] 3.2 Rang 7 (vu 297 fois sur 8 films par le canal tronque, donc rang reel 23) : le §13
      donne 23 = champ de reparation, **confirme par DEUX chaines** (murmur3 + banque sonore
      `sb_007_abl_repairfield`). C'est donc nommable — le verifier et le nommer.
- [ ] 3.3 Publier la couverture APRES : combien de lectures portent un nom, contre les
      **47,0 %** mesures apres la correction de l'etape 1 du lot precedent.

Gate 3 : tableau des rangs (rang, palette, nom, nombre d'appuis, source) ; aucun nom a un
seul appui.

## Etape 4 — L'effet actif plein-fiche (ce que tout ceci debloque)

> Rappel de la demande utilisateur : l'effet porte sur TOUTE LA FICHE. Surbouclier = fiche
> doree, camouflage = effet de verre, translocateur = bordure animee.

- [ ] 4.1 PRE-REQUIS, a re-verifier avant de coder : l'identite est desormais connue (etapes
      1-3) ; l'ETAT ACTIF, lui, ne l'est toujours PAS de facon exploitable — `i57` est lu sur
      **0,82 %** des records et son association avec les episodes `i54` vaut **72,2 % contre
      34 % de temoin**, soit une erreur sur quatre. **Cette etape ne se code donc PAS tant que
      l'etat n'a pas une source fiable.**
- [ ] 4.2 Ce qui devient possible SANS l'etat actif, et qui vaut d'etre propose a
      l'utilisateur : afficher la capacite PORTEE correctement nommee sur les trois capacites
      qui l'interessent (aujourd'hui elles ne sont jamais nommees). C'est un gain reel et
      immediat, a ne pas confondre avec l'effet demande.
- [ ] 4.3 Si une source d'etat fiable apparait (voies au registre : rendre `i56` lisible sur
      les records denses ; verite Theater datee a la seconde) : l'effet se code alors sans
      rien inventer, la palette etant resolue.

Gate 4 : NON FRANCHI par ce plan — l'item 4.1 est un `[!]` attendu. Ce plan livre l'IDENTITE,
pas l'ETAT.

## Hors perimetre

- L'etat actif lui-meme (voies au registre) ; le lecteur `i56` sur records denses.
- Toute renumerotation « au jugé » de la table actuelle sans passer par l'etape 2.
- Le rendu plein-fiche tant que 4.1 n'est pas leve.

## Ce qui peut faire echouer ce plan

1. **`i48` est rare** (0,03 a 0,09 % des records delta, ~une fois par vie). Si une vie n'en
   porte aucun, la fiche n'a pas d'identite pour cette vie — prevoir le repli (derniere
   valeur connue du joueur ? rien ?) et l'ECRIRE, ne pas le laisser au hasard.
2. **La palette peut rester indeterminee** sur des films dont la signature est trop pauvre
   (peu de rangs observes). C'est prevu : on n'affiche pas de nom, on affiche le rang.
3. **Le champ d'inventaire change de sens** (rang complet au lieu d'index tronque) : tout
   consommateur non mis a jour lirait un nombre qui ne veut plus dire la meme chose. C'est
   l'item 1.2, et c'est le vrai risque de regression de ce plan.
