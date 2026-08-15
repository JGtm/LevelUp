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
5. **La detection de perspective n'a pas de garde-fou de duree.** Elle repose UNIQUEMENT
   sur mono/stereo. Elle peut donc apparier deux evenements qui n'ont rien a voir, du
   moment que l'un est mono et l'autre stereo. Cas mesure, signale a l'oreille par
   l'utilisateur sur le Needler :

		081fe06b   75 wem   0,08-0,46 s   mediane 0,08 s   75 mono    -> etiquete 3P
		7474d8d8   62 wem   0,03-3,88 s   mediane 2,08 s   61 stereo  -> etiquete 1P

   A 720 coups/min une aiguille dure 83 ms : un son de 2 s ne peut pas etre le tir. La
   couche `Switch` de `7474d8d8` (40 sons, 1,54-3,88 s) designe la SUPERCOMBINAISON — les
   aiguilles plantees qui explosent, le branchement dependant du nombre plante. Le critere
   mono/stereo est valide pour departager deux versions DU MEME son (il a bien marche sur
   l'epee, ou les durees s'appariaient) ; il ne dit RIEN sur le fait que deux evenements
   soient la meme chose. GARDE-FOU A POSER : deux perspectives d'un meme son ont des durees
   comparables — refuser l'appariement au-dela d'un rapport de l'ordre de 3. Ici le rapport
   est de 26.
6. **Le marteau d'Escharum** n'a ni nom ni icone (arme de boss, aucune entree produit).
7. **`weapon_names.toml` ignore le Mutilator** ; `jeu/index.json` lui met `arme: None`.
   Rattachement pose a la main dans le generateur de manifeste, avec sa provenance.

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

1. Poser le garde-fou de duree sur l'appariement de perspective (defaut 5) — c'est le moins
   couteux et il produit aujourd'hui des faux AUDIBLES.
2. Generaliser le lien direct (section 3) et MESURER l'ampleur du probleme.
3. Integrer la melee comme un mode a part entiere (defaut 3).
4. Separer coup touche / coup manque (defaut 4) — sans nouvelle extraction.
5. Exploiter le paquet `RANGED` pour que deux rendus d'un meme coup different (defaut 2).
6. Instruire les 989 banks a chunks illisibles (defaut 1).

## 7. VOTES DE L'UTILISATEUR

28 votes dans `Downloads/votes-sons-armes.json` (2026-08-15T17:59Z), tous « garder ».
Les cles `ev_*` restent valides. Les cles `_coup` et `_mode_*` sont ORPHELINES depuis le
passage aux rendus par (mode, perspective) : 18 votes sur 28 ne se rattachent plus, l'objet
vote ayant ete scinde. Aucune migration automatique n'est honnete ici.
