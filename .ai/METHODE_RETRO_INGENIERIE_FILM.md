# MÉTHODE — rétro-ingénierie du format de film

> Écrit le 2026-07-27, au terme d'un chantier qui a produit quatre tables et commis six erreurs
> de méthode identifiables. Ce document ne raconte pas ce qui a été trouvé — c'est le rôle de
> `RECETTE_LOADOUT_2026-07-27.md`. Il dit **comment chercher**, et surtout **comment se tromper**,
> parce que chaque règle ci-dessous est adossée à une erreur réellement commise, pas à un principe
> général.

---

# PREMIÈRE PARTIE — CE QUI PRODUIT DES RÉSULTATS

## 1. Suivre le CONSOMMATEUR, pas le producteur

**La règle.** Savoir lire une valeur et savoir ce qu'elle désigne sont deux problèmes distincts.
La grammaire est chez celui qui écrit le champ ; **l'identité est chez celui qui le relit.**

**Ce qui l'a établie.** Deux jours ont été passés à corréler des valeurs entre elles pour deviner
quel compteur d'`i22` était quelle grenade. Résultat : 53 % de pureté contre 25 % de hasard, du
bruit. La réponse est tombée en une passe dès qu'on a cherché **ce qui relit le champ** : une table
`grenade_types` indexée par le rang, avec les noms internes dedans.

**Comment l'appliquer.** Localiser le désérialiseur donne le décalage du champ. Chercher ensuite
les références à ce décalage : celui qui compare la valeur à des constantes, ou s'en sert comme
indice dans un tableau, porte la sémantique. Un `switch`, une table de saut, une indexation de
palette — c'est là que sont les noms.

## 2. Deux chaînes indépendantes valent une preuve ; une chaîne vaut une hypothèse

**La règle.** Une conclusion obtenue deux fois par des voies qui n'ont **aucune étape commune**
est close. Obtenue une seule fois, elle reste ouverte quel que soit le nombre de contrôles
internes.

**Ce qui l'a établie.** La table des grenades a été obtenue d'abord par appariement de 35 lancers
observés aux décréments de compteur, puis par lecture de la table `grenade_types` dans
l'exécutable. Aucune étape partagée. Les quatre rangs concordent. **La question est close** — et
elle ne l'était pas avec la seule première chaîne, malgré ses 35/35 et ses nuls à zéro.

Même mécanique sur les armes : 16 identifiants nommés par une table bâtie par balayage des
images-clés, confrontés aux identifiants lus au curseur journalisé par la capture. Deux chemins,
8/8.

**Le test qui dit si les chaînes sont vraiment indépendantes** : si l'une échoue, l'autre
tombe-t-elle aussi ? Si oui, elles partagent une étape et ne comptent que pour une.

## 3. Un oracle interne bat une vérité terrain

**La règle.** Quand une propriété peut être établie **à l'intérieur du film**, elle ne dépend ni
d'une capture, ni d'un relevé humain, ni de l'appariement à un match. Elle survit à toutes les
confusions de source.

**Ce qui l'a établie.** Le sens du sélecteur d'arme a été trouvé sans aucun relevé : l'emplacement
dégainé est celui dont les munitions bougent. 94,7 % et 95,4 % contre une référence à 66/33. Cette
conclusion a **survécu intacte** à la découverte que le POC mélangeait deux films, alors que toutes
les conclusions adossées à la vérité terrain ont dû être réexaminées.

**Corollaire.** Avant de demander un relevé à un humain — qui coûte du temps et introduit un risque
d'appariement — chercher si le film porte lui-même un oracle. Le fil des morts, les décréments de
compteur, les mouvements de munitions en sont.

## 4. Mesurer les faux positifs sur le flux réel, jamais les calculer

**La règle.** Le flux est saturé d'octets nuls et de motifs répétés. **Aucun calcul de probabilité
uniforme n'y est valide.**

**Ce qui l'a établie.** Un taux de faux positifs calculé s'est révélé faux d'un facteur 20 000.
À l'inverse, les résultats solides du chantier portent tous un compte réel : 0 faux positif sur
650 641 positions alignées, contrôle négatif à 0 sur 192, 0 sur 200 000 permutations aléatoires.

**Comment l'appliquer.** Faire tourner le détecteur sur des positions où la réponse est connue
comme négative, et compter. Si le compte n'est pas mesurable, la conclusion n'est pas publiable.

## 5. La fermeture arithmétique comme vérification gratuite

**La règle.** Quand deux mesures indépendantes doivent coïncider par construction, leur
coïncidence est une vérification qui ne coûte rien.

**Ce qui l'a établie.** La base des emplacements d'arme (`0x7F0`) et le pas (`0x90`) ont été lus
dans une fonction ; le décalage du jeu de grenades (`0xA30`) dans une autre, sans rapport. Or
`0x7F0 + 4 × 0x90 = 0xA30`. Trois mesures se ferment **sans ajustement** : la structure est
comprise, pas devinée.

**Comment l'appliquer.** Systématiquement chercher la contrainte qui relie deux mesures. Un pas et
un nombre d'entrées donnent une borne ; une borne qui tombe sur un décalage connu vaut
confirmation.

## 6. Le désassemblage fait foi, pas le décompilé

**La règle.** Ghidra **supprime silencieusement des blocs** qu'il juge inatteignables, avec la
mention « Removing unreachable block » qu'on ne voit pas si on ne la cherche pas.

**Ce qui l'a établie.** Une conclusion de ce chantier — « ce composant n'écrit que trois octets de
sentinelle, l'écriture principale est ailleurs » — venait d'un décompilé tronqué. Le bloc supprimé
contenait littéralement les instructions qui font l'écriture. Le piège avait **déjà été consigné
la veille** dans le dépôt ; le chantier est retombé dedans.

**Comment l'appliquer.** Sur tout composant dont le décompilé paraît trop court ou incohérent avec
la largeur mesurée, désassembler. La largeur mesurée sur la capture est le juge : un désérialiseur
dont le code visible ne consomme pas la bonne largeur est un code tronqué.

## 7. Attaquer par des lentilles distinctes, pas par répétition

**La règle.** Trois vérificateurs qui posent la même question ne valent qu'un. Trois qui attaquent
par des angles différents — le contrôle interne, l'instant de lecture, le hasard — trouvent des
failles différentes.

**Ce qui l'a établie.** La conclusion sur les grenades a été confirmée par la lentille « hasard »
(qui n'a rien cassé, mesures à l'appui) **et démolie** par la lentille « contrôle interne », qui a
montré qu'une hypothèse d'échappement n'avait jamais été testée. Une seule des trois lentilles a
trouvé la faille.

---

# DEUXIÈME PARTIE — LES SIX FAÇONS DE SE TROMPER, OBSERVÉES

Chacune est datée du 2026-07-26/27 et porte le contrôle bon marché qui l'aurait évitée.

## A. Bâtir sur une prémisse jamais retestée

**Ce qui s'est passé.** Tout le chantier reposait sur « les libellés lisibles n'existent pas en
version publiée, l'identité viendra d'un ordre ». **Faux deux fois** : les noms sont dans
l'exécutable (`vtable + 0x08`) et dans le film lui-même (registre du `chunk_00`) — ce dernier
**déjà analysé par le dépôt depuis le 2026-07-01**. Deux jours de travail ont partiellement
re-dérivé une table existante.

**Le contrôle qui l'aurait évité, coût : deux minutes.** Avant d'attaquer un problème
« impossible », chercher dans le dépôt ce qui est déjà décodé. Un `grep` sur le nom du composant,
une lecture de l'index des recettes. Une prémisse négative — « ça n'existe pas » — se re-teste à
chaque reprise de session, parce qu'elle est le plus souvent héritée sans preuve.

## B. Annoncer un contrôle sans l'exécuter

**Ce qui s'est passé.** Le contrôle interne « les trois grappins doivent partager la même valeur »
a été présenté comme passé. Il **n'a tourné dans aucune des quatre passes**. Pire : il ne
*pouvait* pas tourner, les deux côtés de la comparaison venant de deux films différents.

**Le contrôle qui l'aurait évité.** Exiger, pour tout contrôle annoncé, la **sortie chiffrée** :
« 3/3 sur les slots 512, 513, 516, valeur 4 ». Une formulation qualitative — « le contrôle passe »,
« les triplets sortent groupés » — sans chiffres par ligne est le symptôme d'un contrôle non
exécuté.

## C. Généraliser depuis un seul échantillon

**Ce qui s'est passé.** « La palette des capacités compte **exactement** 4 valeurs. » La mesure
disait seulement « 4 valeurs **dans ce match** ». Le jeu en compte au moins onze. La même erreur a
été commise le même jour sur les munitions : « ces armes n'ont aucun composant de munitions »,
conclu sur 2 lectures là où il y en avait 597 dans une autre branche.

**Le contrôle qui l'aurait évité.** Une règle d'écriture, appliquée sans exception : **« N valeurs
observées dans ce match »**, jamais « la palette compte N ». La différence n'est pas rhétorique :
la première formulation invite à chercher les autres, la seconde clôt la recherche.

## D. Ne pas vérifier que deux sources parlent du même sujet

**Ce qui s'est passé.** La capture décrivait un film, le document de rejeu un autre, la vérité
terrain le second. Toutes les confrontations comparaient deux matchs différents — rosters
disjoints à 7 joueurs sur 8. Trois explications successives ont été inventées pour justifier les
contradictions : horloge fausse, dotation de pré-match, grammaire erronée. **Aucune n'était la
bonne.** Le document de rejeu déclarait pourtant son identifiant de film et sa date en clair, dans
les données relues à chaque session.

**Le contrôle qui l'aurait évité, coût : une lecture de champ.** Avant toute confrontation,
afficher l'identifiant de la source de **chaque** côté. Et faire porter à tout document composite
l'identifiant de la source de **chaque bloc** — un document de rejeu qui agrège plusieurs origines
sans les tracer est un piège à contradictions.

**Le signal d'alerte à reconnaître** : quand trois explications successives échouent sur la même
contradiction, ce n'est pas l'explication qui manque, c'est une prémisse partagée par les trois
qui est fausse. Arrêter de chercher des explications et remettre en cause les données d'entrée.

## E. Résoudre une contradiction par une échappatoire non testée

**Ce qui s'est passé.** Face à « aucune table ne réconcilie l'état mesuré et la vérité terrain »,
la conclusion tirée fut « donc cet état est une dotation de pré-match ». L'autre branche de
l'alternative — « donc le modèle du composant est faux » — n'a jamais été testée. La mesure qui la
départageait était disponible dans les fichiers déjà produits, et elle détruisait l'échappatoire.

**Le contrôle qui l'aurait évité.** Une contradiction ouvre toujours **au moins deux** branches.
Les énumérer explicitement avant d'en choisir une, et exiger une mesure pour éliminer celles qu'on
écarte. Une branche écartée sans mesure est une hypothèse déguisée en conclusion.

## F. Confondre incertitude et risque

**Ce qui s'est passé.** « Je n'ai pas vérifié que l'énumération est stable dans le temps » a été
publié comme « l'énumération peut dériver, il faut indexer la table par version du jeu ». La
seconde formulation appelle une parade coûteuse que la première ne justifiait pas. Une objection
de l'utilisateur l'a corrigée : une énumération sérialisée grandit par ajout, et le rejeu du jeu
lui-même impose cette stabilité.

**Le contrôle qui l'aurait évité.** Avant de traiter une incertitude comme un danger, se demander
ce qui la rendrait vraie. Ici : quel mécanisme renumérotrerait une énumération ? Aucun de plausible
— au contraire, la compatibilité du rejeu l'interdit. Une incertitude sans mécanisme de défaillance
identifié n'est pas un risque, c'est une vérification à planifier.

---

# TROISIÈME PARTIE — RECETTES TECHNIQUES RÉUTILISABLES

## Nommer un composant

Deux voies, la seconde étant la moins chère :

1. **Hors ligne, dans l'exécutable** : le nom est à `vtable + 0x08`, thunk
   `LEA RAX,[chaîne] ; RET`. Piège : `vtable + 0x58` est aussi un thunk de chaîne mais rend le nom
   d'un **autre** composant de la même famille.
2. **Depuis le film lui-même** : le registre ECS du `chunk_00` porte les 64 noms. Déjà implémenté
   dans `internal/analysis/filmdec/registry.go`, table consignée dans
   `RECETTE_DECODAGE_FILM_CHUNKS.md` §6. **À consulter avant toute recherche de nommage.**

## Localiser une lecture dans le flux

La capture journalise 16 octets lus au pointeur d'octet du lecteur, qui avance par mots de 64 bits.

    offset de la signature = paquet.Start + 8 * floor(curseur / 64) + 8

Rendement : 98,4 % de rappel, multiplicité de 0 ou 1 paquet par lecture, **0 faux positif mesuré
sur 650 641 positions alignées**. Le filtre d'entropie utilisé auparavant est inutile et coûteux —
il rejetait des lectures valides.

Le décalage de `+8` n'est pas une convention : il a été trouvé par balayage de −32 à +64 par pas
de 8, `+8` donnant 623 lectures et toutes les autres valeurs zéro ou une.

## Trouver la position exacte d'un composant

    position exacte = paquet.Start * 8 + curseur_moteur       (offset NUL)
    largeur consommee = curseur(composant suivant) - curseur(courant)

Établi par balayage de l'amorce sur 0..8 : seul `+0` produit un parse valide, 249 fois sur 249.
Conséquence pratique : **aucune largeur n'a besoin d'être portée depuis Ghidra**, elles se mesurent
sur la capture.

## Lire une structure depuis un enregistrement

    tampon = *(void**)(enregistrement + 0x10)

Tous les décalages de champ sont relatifs à ce tampon. Les tableaux d'emplacements se reconnaissent
à des composants **jumeaux** : même largeur, même nombre de valeurs, décalages séparés d'un pas
constant. Cinq paires ont été identifiées ainsi avant même d'ouvrir le binaire, simplement en
lisant la table des largeurs.

## Reconnaître une union dans une grammaire

Un champ dont le premier bit conditionne la suite est une **union**, pas un entier avec un drapeau.
Symptôme observable : un invariant du type « bit 0 à 0 implique bit 9 à 1 » vérifié sur toutes les
lectures, et une population qui se scinde nettement par catégorie de porteur.

Exemple mesuré : `i30` porte le chargeur (branche entière) ou la jauge de charge (branche
fraction), selon l'arme. Quatre armes à 99,8-100 % dans une branche, toutes les autres dans
l'autre.

---

# QUATRIÈME PARTIE — LE PROTOCOLE, EN SEPT LIGNES

1. **Chercher d'abord dans le dépôt** ce qui est déjà décodé. Re-tester les prémisses négatives.
2. **Vérifier que toutes les sources parlent du même sujet**, en affichant leur identifiant.
3. **Mesurer la grammaire** sur la capture (positions et largeurs), pas dans le décompilé.
4. **Suivre le consommateur** pour la sémantique. Désassembler, ne pas se fier au décompilé.
5. **Chercher un oracle interne** avant de demander un relevé humain.
6. **Compter les faux positifs** sur le flux réel. Publier le chiffre, pas l'adjectif.
7. **Attaquer par lentilles distinctes**, et énumérer les branches d'une contradiction avant d'en
   choisir une.

Et une règle d'écriture qui vaut les six autres : **ne jamais écrire comme propriété du jeu ce qui
n'a été mesuré que sur un match.**
