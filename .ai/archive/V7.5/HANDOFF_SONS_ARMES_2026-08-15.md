# HANDOFF — extraction des sons d'armes Halo Infinite (2026-08-15)

> Point d'entree pour reprendre ce chantier. Le plan
> (`.ai/V7.5/PLAN_EXTRACTION_SONS_ARMES.md`) fait autorite sur le detail et le journal ;
> ce fichier dit l'etat, ce qui est PROUVE, ce qui ne l'est pas, et par ou continuer.
> Branche `feat/extraction-sons-armes`, HEAD `c0b6da40f`. Rien n'est merge.

## 0. ETAT EN UNE PHRASE

La chaine « pack audio -> bank Wwise -> evenement -> couches -> `.wem` » est etablie et
outillee (`apps/go-api/cmd/weapon-sounds`, 15 modes). 58 armes exposees, 55 avec une
structure d'evenements prouvee, 37 avec des sons de tir designes par un champ NOMME du tag
`weap`, 31 nommees avec icone. Un outil de tri hors depot permet a l'utilisateur d'ecouter
et de voter. Il reste des defauts CONNUS et non corriges, listes en section 4.

## 1. CE QUI EST PROUVE (et par quelle mesure)

- Les banks Wwise ne sont pas sur disque : elles sont converties en tags `sbnk` dans les
  modules. 1305 dans `pc/globals/globals-rtx-new.module`, format Wwise verbatim
  (`BKHD` 1299, `HIRC` 1296).
- Aucun nom n'existe dans le jeu : `stringsSize` = 0 sur les 132 modules, 0 chunk `STID`
  reel sur 1305 banks. **Toute identification passe donc par les TAGS, jamais par un nom.**
  Corollaire paye cher : le nom interne d'un pack audio ne dit PAS quelle arme c'est
  (`fr_heatwave` mene a `hinf_cindershot`, `fr_hotrod` a `hinf_heatwave`).
- Le son de tir est designe par le champ NOMME « Weapon Fire Sound » de `weap.xml`. Le
  plugin derive du build de +64 octets — verifie sur deux champs independants
  (« Weapon Fire Sounds » +4288 -> +4352, « barrels » +3220 -> +3284).
- La cadence est lisible (`barrels` -> `rounds per second`) : MA40 720 c/min, S7 67,
  MA5K 1200, Vestige 285. Attention, ~30 coups/s est un PLAFOND MOTEUR, pas une cadence.
- Un coup est un EMPILEMENT de couches paralleles, chacune tirant sa variante. Les gains
  sont declares par objet `Sound` (5 312 sur 62 753, jusqu'a -96 dB) et sont appliques.
- Les evenements vont par paires **mono = 3e personne / stereo = 1re personne**. Pour le
  rejeu 2D, la 3P est la perspective pertinente (decision utilisateur).

## 2. OU SONT LES DONNEES

	apps/go-api/cmd/weapon-sounds/      l'outil (15 modes, cf. en-tete de main.go)
	scratchpad/lot1.json                par bank : evenements, `.wem`, couches, gains
	scratchpad/lot2.json                par arme : weap, tags de tir, events, modes, cadence
	scratchpad/coups.json               rendus par (mode, perspective)
	Desktop/Halo Infinite - Sons armes/ 10 754 `.wav` + `TRIER.html` (outil de vote)

ATTENTION MEMOIRE : `pc/globals` fait 7,24 Go et `himodule.Open` lit tout. Ne JAMAIS
l'ouvrir en parallele d'un autre gros chargement. `any/globals` fait 0,62 Go et porte les
`weap`/`snd!`/`lsnd`/`stai`. Les modes s'echangent leurs resultats par JSON exactement pour
ne jamais avoir les deux en memoire.

## 3. LA DECISION EN ATTENTE — generaliser le lien direct

L'utilisateur penche pour cette generalisation, elle n'est PAS faite.

CONSTAT. L'appariement `.pck` <-> bank se fait par intersection d'identifiants `.wem`. Il
est solide, mais il designe la bank dont les sons sont DANS LE PACK — pas la bank du TIR.
Or une arme peut avoir plusieurs banks, et celle du tir peut etre entierement embarquee,
donc invisible a cet appariement.

CAS D'ECOLE, detecte A L'OREILLE par l'utilisateur avant tout controle automatique : le
Mutilator (`weap d7915565`) porte deux entrees dans le rapport, toutes deux rattachees au
meme `weap` :

	Banished_enforcer         bank ff09acbd   78 pck + 81 embarques   0 mode   (repli relais)
	Banished_bank8827aa7e     bank 8827aa7e   0 pck + 129 embarques   2 modes  (lien DIRECT)

C'est la seconde qui sonne comme le Mutilator. Normal : ses deux modes de tir pointent
DIRECTEMENT vers `8827aa7e`, tandis que `ff09acbd` n'est atteinte que par un relais `stai`
a quatre niveaux — le maillon faible de la chaine.

PRINCIPE A GENERALISER : **quand un `weap` pointe directement vers une bank par son champ
de tir, c'est CELLE-LA qui porte le tir**, et elle prime sur celle appariee au pack. A
faire : appliquer ce critere partout, fusionner les entrees d'un meme `weap` en distinguant
« bank de tir » et « bank secondaire », et MESURER combien d'armes etaient silencieusement
mal servies. Le nombre est inconnu — c'est la premiere chose a etablir.

## 4. DEFAUTS CONNUS ET NON CORRIGES

1. **989 banks portent des chunks au nom non imprimable** — le decoupeur en chunks derape.
   Aucun effet mesure sur les armes traitees, mais c'est un trou non explique.
2. **Le paquet `RANGED` n'est pas exploite.** Ce sont les aleas de volume et de hauteur
   appliques a CHAQUE lecture : c'est pourquoi un meme son ne sonne jamais deux fois pareil
   en jeu. Les rendus actuels sont donc plus figes que la realite.
3. **La melee n'est pas un « mode de tir ».** Le Mutilator a TROIS modes du point de vue
   joueur : deux tirs plus la frappe. Les deux premiers sont dans le tableau
   « Weapon Fire Sounds », la frappe est dans le champ `melee sound` — un champ DIFFERENT,
   deja lu par le mode `melee` mais jamais integre au rendu. A unifier.
4. **Coup touche contre coup manque : distinction jamais faite.** Verifie sur pieces —
   l'impact ne vient PAS de la bank de l'arme mais du champ `sound material effects`, qui
   depend de la surface touchee. Un coup qui rate est donc le balayage SEUL. Les deux
   familles sont deja dans les donnees extraites (l'analyse de l'epee mesure des evenements
   « impact » a attaque <= 0,11 s et 5-6 kHz, nettement distincts des « balayage »), elles
   n'ont simplement jamais ete separees. **Ce ne sont pas des pistes manquantes, c'est une
   distinction manquante** — donc corrigeable sans nouvelle extraction.
5. ~~**La detection de perspective n'a pas de garde-fou de duree.**~~ **CORRIGE** (etape 10).
   Le garde-fou est pose dans `scratchpad/coups_lot.py` (`RAPPORT_MAX`), avec un seuil qui
   n'est PAS celui annonce ici. Le seuil de 3 estime plus haut refusait 26 appariements sur
   12 armes, dont des appariements DEJA VALIDES a l'oreille (skewer 6,3 ; spiker 6,2 ;
   sniper 5,9). En 1re personne la duree est legitimement plus grande — on entend la
   mecanique et la queue. Seuil recalibre sur les 44 votes : **10**. Le Needler (rapport 26)
   reste le seul refus sur les armes votees ; son evenement `7474d8d8` porte une couche
   `Switch` de 40 sons a 1,54-3,88 s, c'est la SUPERCOMBINAISON, pas le tir.
   Les evenements refuses restent ecoutables (`_ECARTE_*.wav` + rafale) : un refus calcule
   doit pouvoir etre dementi a l'oreille.
6. **Le marteau d'Escharum** n'a ni nom ni icone (arme de boss, aucune entree produit).
7. **`weapon_names.toml` ignore le Mutilator** ; `jeu/index.json` lui met `arme: None`.
   Rattachement pose a la main dans le generateur de manifeste, avec sa provenance.

8. **Le critere de choix `max(couches, wem)` est degenere et fait des degats visibles.**
   Il choisit pour l'epee infectee une reference de 3e personne a 25,71 s de mediane
   (`dabf5bc3`, retenu parce qu'il a 2 couches quand tous les autres en ont 1) : ses 10
   refus de perspective sont mesures contre une reference absurde. Le defaut etait connu
   comme lecon de methode ; il est desormais un defaut ACTIF, a corriger avec le garde-fou.

9. ~~**LES CONTENEURS `Switch` SONT TRAITES COMME DES CONTENEURS ALEATOIRES.**~~ **CORRIGE**
   (etape 12). Le decodeur est dans `conteneurs.go`, branche par `bank.go` -> `resoudreSwitch` :
   seuls les enfants de l'ETAT PAR DEFAUT sont retenus, et les trois issues sont comptees
   (etat porteur / etat vide / table non decodee). Mesure : 440 tables sur 445 decodees et
   validees. 28 coups de 1re personne changent, le lot de candidats se reduisant d'un facteur
   3 a 5 (sniper 30 -> 6, Needler 40 -> 8, MA40 133 -> 45 wem). **Le Needler retrouve sa 1re
   personne** : son evenement etait refuse par le garde-fou de duree UNIQUEMENT parce que la
   couche `Switch` melangeait tous les etats, dont la supercombinaison. Le defaut 5 et le
   defaut 9 n'en faisaient qu'un.
   HORS PERIMETRE, decision utilisateur du 2026-08-15 : **l'etat par defaut suffit**. Le
   rejeu 2D n'a besoin ni de la hauteur, ni de la distance, ni de l'environnement. Le
   `Blend` a automation (42 conteneurs sur 303) et les etats non par defaut sont donc
   CLOS, pas en attente — ne pas les rouvrir sans une demande explicite.

9bis. **[HISTORIQUE, pour comprendre le correctif ci-dessus]** C'est le
   defaut le plus lourd du chantier, trouve en instruisant les votes de l'utilisateur sur
   des sons ISOLES. Un `RandomSequence` tire une variante au hasard : ses candidats sont
   interchangeables. Un `Switch` choisit selon un ETAT DE JEU (distance, materiau, nombre
   d'aiguilles plantees) : ses candidats ne le sont PAS, et il peut ne rien jouer du tout.
   Le parseur ne lit que le NOM du type (`arbre.go:27`) ; il ne lit jamais le groupe de
   commutation ni la table etat -> enfants, et resout les enfants par l'heuristique
   generique. Les etats sont donc melanges dans un seul lot ou le rendu pioche au hasard.

   Mesure : **31 coups reconstitues sur 107 portent une couche `Switch`, dont 28 en 1re
   personne.** Elle y est presque toujours la couche DOMINANTE — mediane de 30 candidats,
   jusqu'a 40 — et pese 38 a 71 % du melange (sniper 71 %, shotgun 38 %, skewer 40 %).

	  UNSC_sniperrifle  M1 1p  ev 78c09986  c0 Switch 30 wem 2,2-4,7 s  -> 71 % du melange
	  Covenant_needler  M1 1p  ev 7474d8d8  c2 Switch 40 wem 1,5-3,9 s  -> la supercombinaison

   Ce defaut explique en cascade : les evenements 1P « trop longs » (defaut 5, dont le
   garde-fou ne soigne que le symptome), le motif « 18 wem en 3P contre 30 en 1P » vu sur
   cinq armes sans lien, et le fait que 12 votes portent sur un son isole. **A CORRIGER
   AVANT TOUT NOUVEAU VOTE SUR LES COUPS DE 1re PERSONNE** : 18 des 44 votes portent sur un
   `_coup_m*_1p` d'une arme concernee, et ces rendus changeront.
   Travail : lire `AkSwitchPackage` (groupe de commutation, etat par defaut, enfants par
   etat), exposer l'etat, et rendre l'etat par defaut au lieu d'un tirage dans tout le lot.

10. **Un son unique partage par 20 armes de 4 factions entre dans les coups a -2 dB.**
   Le `.wem` `195277626` (0,92 s) forme a lui seul une couche `Sound` dans **21 coups de 20
   armes** — Banished, Covenant, UNSC, Forerunner confondus. Il y pese 11 a 36 % du melange
   (pistolet plasma 36 %, sniper 27 %, fusil a pompe 26 %). Un son unique partage par des
   armes sans rapport n'est le tir d'aucune d'elles. Deux autres cas du meme genre :
   `87187708` (0,03 s, 5 armes), `5270936` (0,06 s, 4 armes, -8 dB). A identifier avant de
   decider s'il faut les rendre : ce sont peut-etre des envois de bus ou une foley generique
   que le moteur attenue autrement.


## 5. LECONS DE METHODE (les plus couteuses)

- **Ne jamais deduire l'identite d'une arme du nom de son pack audio.** Le pipeline joint
  par tag `weap` et c'est ce qui l'a sauve du croisement heatwave/cindershot.
- **Parser le minimum necessaire produit des resultats plausibles et faux.** Trois
  specificites Wwise ont ete manquees successivement : les medias embarques `DIDX` (plus de
  la moitie des sons), le tableau des modes de tir, le type d'Action (`Stop` empile comme
  une couche a jouer). D'ou le mode `audit`, qui ENUMERE ce que le format contient en regard
  de ce que le parseur lit. **A relancer apres toute evolution du parseur.**
- **Un critere de tri peut etre degenere sans qu'on le voie.** `max(couches, wem)` sur
  l'epee : 34 evenements a une couche, six a egalite, `max()` rendait le premier du JSON.
- **L'oreille de l'utilisateur a trouve trois defauts avant les controles automatiques**
  (modes de tir, melange SPNKr, bank du Mutilator). Ses retours valent une mesure.
- Deux validations independantes obtenues sans les viser confirment le correctif
  perspectives : l'epee en 1P donne `110deea3` (l'evenement recommande par analyse
  acoustique separee) et le Ravageur mode 2 en 1P donne `be684013` (celui vote a l'oreille).

## 6. ORDRE SUGGERE POUR LA SUITE

0. **LIRE LES CONTENEURS `Switch` (defaut 9).** Passe devant tout le reste : il touche
   28 coups de 1re personne, y pese jusqu'a 71 % du melange, et il est la cause commune de
   plusieurs symptomes deja traites en surface. Tant qu'il tient, voter sur un coup de 1re
   personne revient a juger un tirage au sort entre des etats de jeu differents.
1. ~~Garde-fou de duree~~ FAIT (etape 10, seuil calibre sur les votes).
2. Remplacer le critere de choix degenere `max(couches, wem)` (defaut 8) — il produit
   aujourd'hui une reference a 25 s sur l'epee infectee, donc des refus non interpretables.
3. Generaliser le lien direct (section 3) et MESURER l'ampleur du probleme. Le critere
   « un vrai mode a son propre son de 1re personne » (etape 10) est un acquis a reutiliser :
   il a retrouve seul les 2 modes du pistolet plasma et les 2 du Mutilator.
4. Integrer la melee comme un mode a part entiere (defaut 3).
5. Separer coup touche / coup manque (defaut 4) — sans nouvelle extraction.
6. Exploiter le paquet `RANGED` pour que deux rendus d'un meme coup different (defaut 2).
7. Instruire les 989 banks a chunks illisibles (defaut 1).

## 7. VOTES DE L'UTILISATEUR — SELECTION FINALE VALIDEE

**46 votes dans `Downloads/votes-sons-armes(4).json` (2026-08-16), TOUT VALIDE par
l'utilisateur** apres reecoute complete de la generation « semantique prouvee » : « certains
votes n'ont pas bouges mais j'ai tout reecoute donc pour moi c'est tout valide ». 46/46 se
rattachent au manifeste, 0 orphelin, 0 coup a revoter.

FAIT MARQUANT : la reecoute finale a largement deplace les choix des coups de 3e personne
vers ceux de PREMIERE personne (8 votes `_coup_m1_3p` retires, autant de `_coup_m1_1p`
ajoutes). **La convention « la 3P est la perspective du rejeu » ne gouverne plus la
livraison : LE VOTE PRIME.** On livre ce qui est vote, quelle que soit la perspective.

ROLES MULTIPLES, donnes par l'utilisateur et apparies PAR MESURE (a confirmer par lui) :

	Ravageur (Covenant_provoker) — 3 sons conserves, roles CONFIRMES par l'utilisateur :
	  bb31841b  10 wem, elements des 0,06 s        -> TIR 3 COUPS   = LE SON DU REJEU
	  coup reconstitue (_coup_m1_1p, ev be684013)  -> COUP UNIQUE, conserver
	  c15c9e77  2,65-4,00 s  -> CHARGEMENT (montee en charge de l'arme, PAS un
	                            rechargement), conserver, pas pour le rejeu

	Rayon de sentinelle (Forerunner_sentinelbeam) — CONFIRME par l'utilisateur :
	  coup reconstitue (_coup_m1_1p, ev a220122d) -> TIR CONTINU (vidage de chargeur),
	                                                 conserver, PAS pour le rejeu
	  503433748  1,98 s  -> le court            = LE SON DU REJEU
	  770988828  1,95 s  -> retenu aussi, sans role rejeu declare

## 8bis. CE QUI EST LIVRE, ET SOUS QUELLE FORME

**L'objet livre pour une arme automatique est la RAFALE, pas le coup isole** (decision
produit, mesuree — cf. etape 16 du plan). Le rejeu 2D etant a la 3e personne, ce sont les
rendus `_RAFALE_M<n>_3p.wav` de `Desktop/Halo Infinite - Sons armes/`. Le coup unitaire
(`_M<n>_3p_<k>.wav`) reste utile pour construire et pour juger, pas pour jouer.

Votes finaux : `Downloads/votes-sons-armes(3).json`, 47 votes, 28 des 31 armes nommees
couvertes. Le rendu relit automatiquement le dernier export : toute nouvelle decision est
prise en compte a la regeneration suivante, sans table a maintenir.

## 8. ETAT DE L'ARTEFACT

`scratchpad/tri.html`, publie ; copie dans `Desktop/Halo Infinite - Sons armes/TRIER.html`.
Le compteur de la liste porte sur les COUPS (32/115 tranches), pas sur tous les groupes.
Groupes marques : « choisi » (deja vote), « variante de distance » (element de tir sans son
de 1re personne propre), « garde-fou de duree » (appariement refuse, ecoutable quand meme).
Chaque coup porte SA rafale pre-rendue, a la cadence lue dans le tag.
