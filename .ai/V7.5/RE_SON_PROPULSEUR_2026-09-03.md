# RE — LE SON DU PROPULSEUR : la banque `sb_007_abl_evade`, ses deux evenements, et le stem livre

> Ecrit le 2026-09-03. Suite de `RE_BANQUES_SONORES_NOMMEES_2026-08-26.md` (qui nommait deja
> la banque), `RE_GESTES_SONORES_2026-08-27.md` (la grammaire des evenements et la chaine de
> rendu) et `RE_SONS_RATELIER_ET_KOTH_2026-08-30.md` (l'outillage reecrit).
> Demande : extraire le son d'ACTIVATION du propulseur (le dash) et le livrer au format de la
> banque sonore du rejeu 2D.
>
> Ce lot ne touche AUCUN code web. Le cablage est un lot separe ; la section 7 dit ce qu'il
> doit declarer.

## 0. Le resultat, en une page

	fichier livre                 media       duree     source
	thruster_activate.wav         362715765   0,996 s   play_007_abl_evade_blast_player
	thruster_activate_v2.wav      496436576   0,895 s   idem, 2e variante
	thruster_activate_v3.wav      788441513   0,830 s   idem, 3e variante

Trois variantes parce que la banque en porte trois : l'evenement est un `RandomSequence`
« 1 couche, 1 parmi 3 » et le jeu en tire une A CHAQUE dash. Meme forme que `grapple_fire`,
`repulsor_kill` et `repair_field_activate`, qui livrent deja leurs trois variantes.

## 1. UNE CORRECTION DE LECTURE, ET ELLE CHANGE LA QUESTION

La commande de ce lot citait la banque comme portant « 6 evenements ». La table du
2026-08-26 (section 2) a deux colonnes, `wem` puis `evts` :

	sb_007_abl_evade    916d040a    globals    6    2    propulseur

**Six MEDIAS, DEUX evenements.** Le choix du stem d'activation ne se fait donc pas parmi six
gestes : il se fait entre DEUX evenements, et chacun porte trois variantes du meme geste.
Verifie sur piece dans `_donnees/arbre_all_globals.json` (balayage `eqip-arbre -banks all` du
2026-08-30) : la banque `916d040a` declare `wem_embarques: 6`, `noeuds_avec_delai: 0`, deux
evenements, zero orphelin.

## 2. LES DEUX EVENEMENTS SONT NOMMES — esperance 0,00091

Gabarits `play_007_abl_evade_<jeton>` et treize formes voisines (`_player`, `_npc`, `_enemy`,
`_loop`, `_start`, `_stop`, `_end`, `_tail`, `_1p`, `_3p`, `_other`, `blast_<jeton>`,
`blast_<jeton>_player`), dictionnaire de **138 886 jetons** tires du binaire du jeu
(`HaloInfinite.exe`, chaines ASCII decoupees sur les separateurs ET les frontieres camelCase),
**2 cibles** :

	ESPERANCE DE COLLISION = 1 944 404 x 2 / 2^32 = 0,00091   (seuil de publication 0,10)

	307114b6   play_007_abl_evade_blast_player      3 medias, gain de chemin -1 dB
	c4c81ff9   play_007_abl_evade_blast_nonplayer   3 medias, aucun gain declare (0 dB)

Les DEUX evenements de la banque sont nommes : il ne reste rien d'anonyme ici. Une premiere
passe par composition d'un vocabulaire curie de 189 mots (6 787 180 candidats, esperance
0,0032) avait deja sorti `blast_player` seule — les deux voies concordent, et la seconde,
plus large, n'ajoute aucun candidat concurrent.

**LE CONTROLE QUI VAUT MIEUX QUE L'ESPERANCE — une paire, et elle se ferme.**
`_player` / `_nonplayer` sur le meme radical `blast`, dans une banque qui n'a que ces deux
evenements : c'est la meme signature que les paires `_team` / `_enemy` du drapeau. Une
collision fortuite ne produit pas une paire complementaire sur un radical partage.

## 3. LA PREUVE INDEPENDANTE : LE NOMBRE DE CANAUX SEPARE LES DEUX PERSPECTIVES

Elle n'est pas dans les noms, elle est dans les medias — et elle a ete lue APRES le cassage,
donc elle n'a pas pu l'orienter. Metadonnees vgmstream des six `.wem` :

	evenement     media        canaux            debit      duree source
	307114b6      362715765    2  STEREO FL FR   118 kbps   0,996 s
	307114b6      496436576    2  STEREO FL FR   125 kbps   0,895 s
	307114b6      788441513    2  STEREO FL FR   128 kbps   0,830 s
	c4c81ff9      465652150    1  MONO   FC       79 kbps   0,742 s
	c4c81ff9      496554026    1  MONO   FC       73 kbps   0,888 s
	c4c81ff9      853373807    1  MONO   FC       73 kbps   0,977 s

Trois stereo d'un cote, trois mono de l'autre, sans exception : **la perspective du porteur
est stereo (relative a la tete, non spatialisee), celle des autres est mono (positionnee en
3D par le moteur)**. C'est exactement ce que `_player` / `_nonplayer` annonce, et c'est une
mesure de format, pas un argument. Le debit suit la meme frontiere (118-128 contre 73-79).

## 4. POURQUOI `blast_player` EST LE STEM D'ACTIVATION

Trois raisons, dans cet ordre.

1. **Il n'y a qu'UN geste dans cette banque.** Pas de boucle, pas de queue, pas de recharge :
   `nature = « 1 couche, 1 parmi 3 »` pour les deux evenements, aucun delai d'action
   (`noeuds_avec_delai: 0`), aucune repetition declaree, aucun conteneur `Blend` ni `Switch`,
   aucun orphelin. Le propulseur est une impulsion unique et sa banque le dit. La question
   « activation contre boucle contre queue », qui se posait pour le champ de reparation
   (0,38 s de pose contre 3,26 s d'activation, `RE_GESTES_SONORES` section 8.4), **ne se pose
   pas ici** : il n'y a rien d'autre a confondre.
2. **La modulation `_player` est celle que le rejeu sert deja, pour TOUS les joueurs.** Les
   sons de capacite livres viennent tous de la perspective du porteur :
   `camo_activate` <- `activecamo_activate_player`, `overshield_activate` <-
   `overshield_start_player`, `shroud_deploy` <- `shroud_deploy_player`, `grapple_fire` <-
   `grapplinghook_deploy_player`, `repulsor_kill` <- `knockback_pulse_player`. Livrer le mono
   `nonplayer` ferait de ce son le seul de la banque a sonner autrement que ses voisins.
3. **Le stereo est ce qu'il faut a un mixage non spatialise.** Le lecteur du rejeu ne place
   rien en 3D ; un media mono destine a l'etre y sonnerait plat et plus sourd (73 kbps contre
   128) a cote des autres capacites.

**CE QUE SONT LES TROIS AUTRES MEDIAS**, puisque la question est posee : les trois variantes
mono de `blast_nonplayer`, c'est-a-dire LE MEME DASH entendu depuis un autre joueur. Elles ne
sont pas un second geste. Elles sont rendues et archivees
(`916d040a_c4c81ff9{,_v2,_v3}.wav`, 0,742 / 0,888 / 0,977 s) pour qu'un lot futur qui voudrait
distinguer « mon propulseur » de « celui d'en face » n'ait rien a re-mesurer — mais elles ne
sont PAS livrees dans `static/sounds/`, parce qu'aucune autre capacite ne fait cette
distinction aujourd'hui et qu'un asset que rien ne joue est un asset mort (garde-rail
section 7).

## 5. LE RENDU — parametres, et ils sont ceux de la chaine existante

Chaine inchangee (`RE_GESTES_SONORES` section 5, `RE_SONS_RATELIER_ET_KOTH` section 1) :
decodage vgmstream r2117 (les `.wem` sont en Wwise Vorbis, `fmt = 0xFFFF` — **ffmpeg ne les
decode pas**), somme des couches a leur gain de chemin, crete ramenee a -1,0 dBTP mesuree en
DEUX passes, sortie PCM 16 bits.

	une couche, un `RandomSequence`, aucun delai, une lecture, mode d'enchainement 0
	gain de chemin -1 dB sur les trois medias
	crete du media source : 0,0 dBFS -> apres gain -1,00 dB -> correction +0,00 dB

**Format de sortie, identique aux voisins** (controle sur `grapple_fire.wav`,
`camo_activate.wav`, `repulsor_kill.wav`, `repair_field_activate.wav`, `shroud_deploy.wav`) :
`pcm_s16le`, **48 000 Hz, 2 canaux**, en-tete RIFF `fmt ` + `LIST/INFO` + `data` au meme
layout.

**MESURES DE CONTROLE SUR CE QUI EST LIVRE** — le fichier, pas l'intention :

	fichier                     duree      crete       RMS       plancher de bruit
	thruster_activate.wav       0,996 s   -1,00 dB   -19,2 dB   -50,7 / -52,0 dB
	thruster_activate_v2.wav    0,895 s   -1,00 dB   -18,1 dB   -43,4 / -43,2 dB
	thruster_activate_v3.wav    0,830 s   -1,00 dB   -18,9 dB   -48,6 / -48,0 dB

- **Non silencieux** : RMS -18 a -19 dB, dans la fourchette des voisins (`grapple_fire`
  -15,7 dB, `camo_activate` -19,7 dB, `shroud_deploy` -25,6 dB). Aucun risque de detoner.
- **Non ecrete** : crete a -1,00 dB, sous 0 dBFS, mesuree en deux passes.
- **Non tronque** : la duree livree vaut EXACTEMENT le nombre d'echantillons du media source
  (47 825 / 42 961 / 39 817 a 48 kHz = 0,99635 / 0,89502 / 0,82952 s).
- **Enveloppe d'une impulsion**, RMS par dixieme du fichier :
  `-7,3 -16,6 -21,4 -23,1 -24,2 -31,9 -36,6 -44,4 -46,8 -57,1` — attaque franche, decroissance
  monotone jusqu'au plancher. Ce n'est ni du bruit blanc (qui serait plat) ni une coupe (qui
  s'arreterait haut).

## 6. LES COMMANDES EXACTES, REJOUABLES

1. **Extraire les six medias de la banque** (ouvre `pc/globals`, 7,2 Go — jamais en parallele
   d'un autre chargement ; environ 8,6 Go alloues) :

		cd apps/go-api
		go run ./cmd/weapon-sounds -mode embarques \
		  -module "pc/globals/globals-rtx-new.module" -sbnk 0x916d040a -out <dossier_wem>

2. **Casser les noms des deux evenements** (aucun chargement de module : la cible est le JSON
   de balayage du 2026-08-30) :

		cd "Halo Infinite - Sons v75/_outils/cracker"
		go run . -action gabarits -arbres "../../_donnees/arbre_all_globals.json" \
		  -cibles 916d040a -exe "<jeu>/HaloInfinite.exe" \
		  -gabarits "play_007_abl_evade_%s,play_007_abl_evade_%s_player,..."

   La liste complete des quatorze gabarits est celle de la section 2.

3. **Rendre les trois variantes** (v = 0, 1, 2 -> suffixe vide, `_v2`, `_v3`) :

		cd "Halo Infinite - Sons v75/_outils/rendu"
		go run . -arbre "../../_donnees/arbre_all_globals.json" \
		  -bank 916d040a -event 307114b6 -wem <dossier_wem> -tmp <tmp> \
		  -variante <v> -out static/sounds/halo_infinite/thruster_activate[_vN].wav

**UN AJOUT A L'OUTILLAGE, ET IL EST ADDITIF.** `_outils/rendu` ne savait servir qu'UNE
variante par couche — « toujours la meme, le plus petit identifiant », choix assume pour que
deux rendus soient identiques. Un geste qui se livre en trois fichiers ne pouvait donc pas se
rendre avec l'outil canonique. Le drapeau **`-variante <rang>`** a ete ajoute (defaut 0 =
comportement d'origine, hors bornes = erreur explicite, nom du fichier intermediaire suffixe
pour ne pas se recouvrir). Cet outil vit hors du depot
(`C:\Users\Guillaume\Downloads\Halo Infinite - Sons v75\_outils\rendu\`) ; la sauvegarde
`main.go.bak` porte l'etat d'avant.

## 7. CE QUE LE LOT DE CABLAGE DEVRA DECLARER — et l'avertissement qui va avec

Trois stems, exactement :

	thruster_activate
	thruster_activate_v2
	thruster_activate_v3

Le premier est le stem ; les deux autres entrent par `SOUND_VARIANTS`
(`replaySoundVariants.ts`), sur le modele exact de `grapple_fire` et `repulsor_kill` :

	thruster_activate: ['thruster_activate', 'thruster_activate_v2', 'thruster_activate_v3']

**LE NOM SUIT LA CONVENTION DU CATALOGUE, PAS LE MOT DU MOTEUR.** Le moteur dit `blast` ; le
catalogue nomme par le geste declenche cote rejeu, et il l'a deja fait deux fois —
`overshield_start_player` est livre en `overshield_activate`, `grapplinghook_deploy_player` en
`grapple_fire`. Un geste d'ACTIVATION de capacite se nomme `<famille>_activate` dans ce
dossier (`camo_`, `overshield_`, `wall_`, `sensor_`, `repair_field_`), et la famille du
propulseur s'ecrit `thruster` cote web (`equipmentPlacementsLayer.ts`,
`equipmentUsageLogic.ts`) comme cote Go.

**AVERTISSEMENT — LA LIVRAISON EST ATOMIQUE AVEC LE CABLAGE.** Le garde-rail
`replaySoundAssets.guard.test.ts` porte DEUX assertions symetriques : « chaque stem a son
fichier » ET « chaque fichier est joue par un stem (0 asset mort) ». Les trois `.wav` de ce
lot sont aujourd'hui des orphelins : **la seconde assertion est ROUGE tant que le lot de
cablage n'a pas declare les trois stems.** Ce n'est pas un defaut de ce lot, c'est le
garde-rail qui fait son travail — mais les deux lots doivent atterrir dans le meme commit ou
la meme PR. La regle de duree ne pose, elle, aucun probleme : 0,83 a 1,00 s, sous toutes les
bornes du fichier (`COURT_S` 1,2 s comme `LONG_MAX_S` 6,0 s).

## 8. CE QUI RESTE OUVERT

1. **Le declencheur n'est pas dans ce lot.** L'usage du propulseur est publie depuis le
   schema 38 (commit `5a7ec4208`, valide 5/5 contre un releve Theater) : la source de datation
   existe, seul le branchement du son manque.
2. **`blast_nonplayer` n'est pas livre**, par coherence avec les autres capacites. Si le
   produit voulait un jour distinguer le propulseur du joueur suivi de celui des autres, les
   trois rendus mono sont deja dans `Halo Infinite - Sons v75/rendus/916d040a_c4c81ff9*.wav`.
3. **Aucune ecoute humaine n'a valide ce son.** Il est designe par le format (nom casse a
   0,00091, perspective confirmee par le nombre de canaux, geste unique dans la banque), pas a
   l'oreille. La banque, elle, etait deja nommee le 2026-08-26 ; c'est le seul espace de
   nommage `evade` du jeu.
