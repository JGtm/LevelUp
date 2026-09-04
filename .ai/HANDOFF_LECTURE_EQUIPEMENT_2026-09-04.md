# HANDOFF — Lecture fiable des usages d'équipement (chantier du 2026-09-03)

> Pour la session qui reprend. Branche : `feat/v75`, tout est mergé (`146f1d92e` et suites).
> Worktree du chantier : `LevelUp-wt-lecture-equipement` (branche `wt/lecture-equipement`),
> conservé — il porte l'historique détaillé lot par lot.
> Plan de référence : `.ai/PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md` (items statués).
> Journal : `.ai/thought_log.md`, 44 entrées du 03/09.

## LA DOCTRINE DU CHANTIER — à ne pas perdre

Commande utilisateur, mot pour mot : « **je ne veux pas de choix ou de règle arbitraire, je
veux de la lecture fiable** ». Un lot « détecter la téléportation par un saut de plus de X
mètres » avait été proposé puis ABANDONNÉ sur son refus : *une téléportation peut faire
50 cm*. Tout ce chantier cherche donc les ENREGISTREMENTS du film, jamais un seuil qui
devine. Un négatif mesuré vaut mieux qu'une détection inventée.

Corollaire tenu tout du long : **une famille absente d'une table est écartée ET COMPTÉE**,
jamais silencieusement absente ; et une couverture ne se publie pas quand le balayage a
échoué — publier des zéros affirmerait « lu, rien trouvé » là où rien n'a été lu.

## CE QUI EST ACQUIS

### Le translocateur — complet

- L'usage est l'**événement film type 117** `EquipmentTranslocatorTeleportEffects` (nom lu
  dans l'exe). Précision 18/18 sur 5 films, rappel 8/8.
- Sa **charge utile est décodée** (Ghidra, `FUN_140f04fb8`) : elle porte les DEUX positions
  du saut, départ et arrivée. Publiées dans `translocations[]`, validées 18/18 à
  0,00-0,10 m contre les discontinuités de piste.
- Sémantique établie : **VA-ET-VIENT** — la balise échange sa position avec le joueur à
  chaque usage (l'arrivée d'un saut est le départ du précédent, à 0,09 m près). Toute UI
  « faille posée une fois, fixe » serait fausse.
- La POSE initiale est **illisible** : négatif mesuré sur trois canaux (créations d'entités
  tous archétypes, deltas d'objet du monde, recensement d'images-clés). Rien ne se dessine
  avant le premier échange.
- Fin de l'équipement : deux modes mesurés — épuisement à l'usage final (`spent` quasi
  simultané), ou expiration 9 à 16,5 s après le dernier échange ; la mort du porteur clôt
  aussi, sans `spent`.
- Web : éclat de fiche daté au geste, lien du va-et-vient tracé des positions lues, faille
  qui suit la balise et s'éteint à sa fin mesurée.

### Le propulseur — complet, et validé par vérité terrain

- L'usage est l'**impulsion `tag == 1`** des composants bipède i57 `biped-spartan-ability` et
  i59 `-non-predicted-state` — **le même composant dont le tag 3 porte déjà le grappin**. Les
  désérialiseurs étaient portés depuis le 2026-08-16 : rien n'était à décoder, seul le
  croisement avec l'identité manquait.
- **Vérité terrain utilisateur** (Theater, film `1cd3848a`) : 5 usages relevés à 1:51, 1:54,
  2:03, 2:05, 2:14 ; la chaîne rend 1:52, 1:55, 2:03, 2:05, 2:15. **Précision 5/5, rappel
  5/5, écart ≤ 1 s.** C'est le test d'acceptation du lot, env-gaté.
- Deux VARIANTES d'objet existent (`0x430dda48` classique, `0xeef5d48d` dite légendaire) : la
  découverte portait sur la première, la vérité terrain sur la seconde — **les deux émettent
  le même signal**. Aucune palette ne nomme jamais deux rangs « propulseur », donc la
  reconnaissance par le rang est naturellement insensible aux variantes.
- Web : **dash sur le pion** (sillage en coin plus deux chevrons, 460 ms de temps de match,
  direction lue dans la trajectoire), et le son `thruster_activate` avec ses deux variantes.

### La lecture de l'identité — réparée

- Le canal i48 perdait environ 5 % de ses émissions. **Les trois quarts sont récupérables** :
  les octets sont bien dans le film, rejetés par deux formes que le balayage strict refuse
  (record sans i0, masque dense R(64)).
- Récupération **gatée par le témoin de compteur** : un candidat n'est accepté que s'il
  comble exactement le saut. Le relâchement inconditionnel est RÉFUTÉ par contrôle négatif
  (+800 fausses acceptations sur 10 films). Émissions récupérées étiquetées `recovered`.
- Le résiduel incompressible est **publié** (`gap` par émission) : un `from` sous `gap > 0`
  est traité comme INCONNU par tous ses consommateurs — ni faux, ni sûr.

### Les négatifs mesurés — à ne PAS rechercher à nouveau

- **Le canal des événements ne porte ni le répulseur ni le propulseur.** La liste COMPLÈTE
  est désormais marchée (96,5 % des listes, 236 321 événements, 12 films). La preuve tient
  sur la position 1, où le cadrage est certain : tous les témoins positifs y sont à hauteur
  de leur part, les cibles à zéro pour 42, 30 et 16 attendues.
- **Le répulseur n'est dans AUCUN des neuf canaux jugés** : événements, impulsion i57/i59,
  branche tag 3, poses `deployed`, i48, i54, énergie i56, masque bipède entier, entité ti=37.
  Le cas décisif : un joueur porte le répulseur 68 s, le film annonce lui-même la
  consommation de sa dernière charge, et le compteur de charges reste muet — **pendant qu'il
  compte le grappin de trois autres joueurs de la même partie**.
- Le filtre `MaxSpeedMPS=100` ne coûte que 1 à 3 échantillons bruts par téléportation
  (réancrage aveugle après 3 rejets) : l'arrivée n'est jamais perdue. Ne pas le « corriger ».

## CE QUI RESTE — par ordre de valeur

### 1. PUBLIER LES CHARGES — le lot évident, tout est prêt

`i56 biped-spartan-ability-energy` transmet le **nombre de charges entières restantes** (le
quartet haut de sa valeur sur 7 bits). Sur le film témoin, la série descend **4, 3, 2, 1, 0**
exactement aux cinq usages relevés par l'utilisateur. Deux contrôles indépendants : 52
baisses sur 54 coïncident avec une impulsion, et **36 accroches de grappin sur 36** sont
appariées, contre 2 sur 36 pour un témoin décalé de 5 secondes.

**Rien n'est publié aujourd'hui** — ni dans l'artefact, ni à l'écran. La demande utilisateur
est explicite : la fiche doit montrer l'équipement **et ses charges**, du ramassage jusqu'à
l'épuisement ou la mort. Instruments : `filmdec/r11_*_research_test.go`. Rapport :
`RAPPORT_R11_REPULSEUR_CHARGES_2026-09-03.md`.

PIÈGE À CONNAÎTRE : ce canal avait été écarté par un lot antérieur comme « trop rare »
(0,083 % des records). Il est rare PAR RECORD mais émis À CHAQUE CHANGEMENT DE CHARGE — c'est
ce qui le rend utilisable. Un canal jugé sur sa fréquence brute, faute d'ancre pour le lire,
avait été classé inutilisable alors qu'il portait la réponse.

### 2. LE RÉPULSEUR — il manque UNE ancre, pas un canal

Neuf canaux ont été jugés **sans jamais disposer d'un seul instant d'usage certain**. Le
grappin en a 1 101 dans le corpus, et c'est exactement pour cela qu'on l'a trouvé du premier
coup. **Un relevé Theater d'un seul usage, avec son instant, débloquerait tout.**

Les plages où l'utilisateur porte lui-même le répulseur sont calculées et corrigées (bornées
par ses changements d'équipement, pas par la fin de vie) dans
`CRENEAUX_VERIFICATION_EQUIPEMENT_2026-09-03.md` et dans l'artefact publié pour lui. La plus
longue dure 291 s. Deux fenêtres où un AUTRE joueur consomme sa dernière charge :
`a6ae19fb` 6:21→7:29 et `0d265ab0` 2:46→4:19.

Dernier canal non porté : le type d'événement **14 `PlayEffectOnObject`**.

Ses trois sons d'activation existent déjà dans la bibliothèque de l'utilisateur
(`Downloads/Audio Library/EQUIPMENT/Repulser`) : le jour où le signal est trouvé, le câblage
est de quelques minutes.

### 3. DETTE OUVERTE — `damage_aftermath`, 872 000 événements

L'oracle de trame du lot R7 mesure la grammaire de PRODUCTION `lot1DecodeDamageAftermath`
(`filmdec/weapon_hits_decode.go`) comme **DOUTEUSE** : 1,161 contre une médiane de 1,793 pour
les grammaires validées, sur 2 297 listes à événement unique. Elle touche **872 000
événements de dégâts déjà exploités en production**. Rien n'a été corrigé (R7 était en
lecture seule). **À instruire avant toute exploitation nouvelle du canal dégâts** — et
l'oracle de R7 est précisément l'instrument qui sait la juger.

### 4. Points mineurs

- 18 gestes de propulseur sur 61 (film témoin) n'ont pas d'identité lisible : le geste est
  mesuré, l'équipement non — écartés et comptés (`noIdentity`), jamais devinés.
- Le fixture golden `inputs_000d5950.bin.gz` **n'est pas reproductible** : `lessTrack`
  (`filmdec/projectiles.go:194-202`) n'est pas un ordre total et `sort.Slice` n'est pas
  stable, donc 98 octets bougent d'une régénération à l'autre. Un futur diff de ce binaire ne
  distingue plus une régression de données du bruit d'ordonnancement.
- Le canal `zoom` de production ne lit que les TÊTES de liste : `unit_zoom` apparaît aussi en
  position 2 — il sous-compte.
- Le bloc manuel des schémas 25-27 de `apps/web/src/lib/api/types.ts` a atteint son critère
  de retrait.
- La bibliothèque audio porte une « Recharge » par capacité : non livrée, faute de signal (le
  film ne date aucune recharge, et un asset non déclaré casse le garde-rail).

### FERMÉ DEPUIS, par une session parallèle (`2e0b17cd7`)

La dette du grappin — jointure par slot écrasant les vies multiples — est **corrigée** :
`grappleLayer.ts` joint désormais par la vie (`isAliveAt`), avec sa fixture à deux vies.
Rapport : `RAPPORT_D1_GRAPPIN_VIES_2026-09-04.md`.

## PIÈGES QUI ONT COÛTÉ DU TEMPS — à relire avant de mesurer

1. **Horloge MOTEUR contre horloge FILM.** Les paquets sont datés sur l'horloge moteur, les
   artefacts sur celle du film : `US_moteur = US_film + horodatage du premier paquet du
   chunk 1`. Une première mesure a rendu zéro paquet fenêtré pour l'avoir ignoré.
2. **`bounds` d'artefact n'est PAS les bornes de déquantification.** C'est un cadrage
   d'affichage. Les mètres vrais passent par `map_quant_bounds.json` / `MapQuantEntry`. Une
   mesure entière a été fausse d'un facteur ~10 pour les avoir confondus.
3. **`WorldObjectPrecision` est un global de paquet** : un instrument qui l'oublie rend 13
   poses au lieu de 537, **sans erreur**.
4. **La porte de région du vecteur quantifié est INVERSÉE** : le bloc « lire l'index de région
   et prendre ses bornes » s'exécute quand le bit vaut ZÉRO.
5. **Un slot de bipède est réattribué à chaque réapparition.** Toute jointure `slot → piste`
   doit passer par la VIE (`Map<slot, tracks[]>` puis `.find(isAliveAt)`). Ce défaut a été
   trouvé DEUX fois dans ce chantier (dash, puis grappin) — le patron canonique est dans
   `fireMark.ts:50-59` et `shotFx.ts:62-73`.
6. **Le code et son manifeste voyagent ensemble.** Une cuisson faite avec le binaire à jour
   mais le manifeste absent produit un artefact valide, silencieux, et parfaitement trompeur
   si l'on ne lit pas les compteurs.
7. **Un tube masque le code de sortie.** `go build ... | tail` a rendu `$?` = 0 alors que le
   build avait échoué (mauvaise racine : `go.mod` est dans `apps/go-api/`).
8. **Ghidra headless sous JDK 25** : `Selector.open()` échoue ; contournement et commande
   complète au `RAPPORT_R6` §0 et annexe C.1. Aucune instance n'a besoin d'être ouverte à la
   main, et le serveur se referme en fin de lot.

## MÉTHODE QUI A MARCHÉ — à reproduire

- **L'ancre datée d'abord.** Chaque découverte est venue d'un instant certain — une
  discontinuité mesurée, un relevé utilisateur — contre lequel juger un canal. Sans ancre, on
  compare au bruit : c'est exactement pourquoi le répulseur résiste encore.
- **Le témoin positif est obligatoire.** Une méthode qui ne retrouve pas le grappin (1 101
  usages) ou le propulseur ne prouve rien sur le répulseur. Ce contrôle a fait tomber trois
  pistes, dont une du superviseur.
- **Pré-enregistrer le verdict** avant de regarder, et publier la prédiction même quand elle
  se révèle fausse — elle l'a été une fois, et c'est écrit au `RAPPORT_R7` §4.6.
- **Revue adversariale à contexte frais avant chaque commit.** Sept lots relus, vingt et un
  constats recevables, dont un P0 qui plantait la cuisson et un autre qui aurait dessiné un
  effet à la place d'un autre joueur — tous invisibles aux gates.

## RÉFÉRENCES

Rapports, dans `.ai/V7.5/replay2d/`, tous datés du 2026-09-03 :
`RAPPORT_R1_FAILLE_ACTIVATION`, `R2_I48_MANQUES`, `R3_FILTRE_VITESSE`,
`R5_EVENTS_EQUIPEMENTS`, `R6_GHIDRA_EVENTS`, `R7_TRAME_COMPLETE`,
`R8_USAGE_REPULSEUR_PROPULSEUR`, `R9_REPULSEUR`, `R11_REPULSEUR_CHARGES`,
`CRENEAUX_VERIFICATION_EQUIPEMENT`, `RE_SON_PROPULSEUR`.

48 instruments env-gatés dans `apps/go-api/internal/analysis/filmdec/` (préfixes `r6_` à
`r11_`, `faille_`, `i48_`, `vitesse_`, `equipements_`), sautés par défaut, avec les commandes
rejouables en annexe de chaque rapport.

Notion : « L'usage des équipements dans le film : le propulseur trouvé, le répulseur
introuvable » et « R7 — La liste COMPLÈTE d'événements se marche », toutes deux sous la page
« Percer la trame du film ».

Schéma d'artefact : **38**. Deux films cuits pour le gate visuel — `1b2d9e08` (translocation)
et `1cd3848a` (propulseur). Le reste du parc est antérieur : le repli `spentTranslocations`
tient jusqu'au 2026-12-01 (kill-switch daté, critère « plus aucun artefact sans
`coverage.translocations` »).
