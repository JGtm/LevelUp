package profile

// rev.go — LA REVISION DE LA COUCHE PROFIL, ET SA CHRONIQUE.
//
// # CE QUE CETTE REVISION DIT, ET CE QU ELLE COMMANDE
//
// La couche `profile` ne lit AUCUN octet de film — c est meme sa definition (ADR 0034 D-1) —
// mais elle porte les LARGEURS, les BORNES et les PLAGES que les lecteurs appliquent : le
// decoupage d i0, le decoupage du bloc `object-multiplayer-properties`, la transposition par
// build et par version de format, les bornes de dequantification par carte. Une ligne de cette
// table qui change fait lire au decodeur d autres bits aux memes offsets : c est un changement
// de DECODAGE, exactement comme une largeur corrigee dans `grammar`.
//
// Elle est donc la SECONDE des quatre (decision V15 (11) du PLAN_DECODEUR_FILM_2026-09-13),
// entre `source` — la porte aux octets — et `grammar`, qui hache sa VALEUR (V15 (12)). Le sens
// unique est tenu par le mecanisme et non par la vigilance : une montee de `profile.Rev` fait
// monter `grammar.Rev`, et les revisions des faits avec elle (D6, signal utilisateur).
//
// ELLE NE HACHE PLUS LA VALEUR DE `source.Rev` DEPUIS LE LOT J3.2 (2026-09-26, DU-2 (b)) : le
// perimetre d une couche est la fermeture de ses imports, et `profile` n importe pas `source` —
// la table du decodeur ne depend pas de la facon dont les octets sont atteints. Les couches qui
// lisent des octets (`grammar`, `facts/...`) importent `source` et hachent sa valeur.
//
// # POURQUOI ELLE NAIT MAINTENANT, ET PAS AU LOT 2.5.b QUI A CREE LA COUCHE
//
// Tant que la couche n avait pas de revision propre, ses octets etaient haches dans l empreinte
// de la GRAMMAIRE, et dans elle seule (`racinesGrammaire` portait cinq racines). Le gate mordait
// donc — rien ne passait — mais il ne disait pas CE QUI avait bouge : une borne de carte et un
// ordre de composants rendaient le meme diagnostic. Les quatre revisions du lot 2.6 rendent la
// reponse lisible sans rien relacher : chaque couche hache ses propres octets, et les valeurs
// de celles du dessous.
//
// # CE QUI EST HACHE, ET CE QUI NE L EST PAS
//
// Les sources `.go` NON-TEST de ce paquet, sous le mecanisme central (`film/revision`, lot
// 2.6.0) : chemin relatif A LA RACINE hache a cote du contenu, fins de ligne normalisees en LF,
// `testdata/` et `_test.go` ecartes. CE fichier est EXCLU : il DECRIT la couche, il n en fait pas
// partie — sans quoi faire monter la revision changerait aussi l empreinte, et la branche « la
// revision a change sans que la couche bouge » serait du code mort (correctif R1 P2-3).
//
// LE CATALOGUE DES CARTES N EST PAS HACHE, et c est une limite ASSUMEE : les bornes de
// dequantification par carte sont chargees depuis `data/titles/halo_infinite/reference/`
// (`map_bounds.go`), donc depuis des fichiers de DONNEES que cette empreinte ne voit pas. Hacher
// un repertoire de donnees de production ferait dependre un gate de source de l etat du poste.
// Ce que l empreinte garde est le CODE qui les lit et les compose ; ce que le catalogue change,
// le corpus gate le voit.
//
// VALEUR AMONT : AUCUNE depuis le lot J3.2 (cf. plus haut) — `testdata/profile_perimetre.golden`
// fige ce que la fermeture rencontre.
//
// # LA FORME, ET POURQUOI LE PREMIER RANG DU JOUR N A PAS DE SUFFIXE
//
// `profile-AAAA-MM-JJ[.N]`, `N >= 2` : deux LOTS du meme jour se separent par leur rang, et le
// premier lot du jour s ecrit SANS suffixe (`revision.ParserRevision` refuse `.1` — une meme
// position ne s ecrit pas de deux facons).
//
// # LA CHRONIQUE — UNE ENTREE PAR RANG, ET RIEN QU UNE
//
// ENTREE `profile-2026-09-17` (2026-09-17, lot 2.6.1) : NAISSANCE DE LA REVISION DE PROFIL.
// AUCUNE LIGNE DE LA TABLE NE CHANGE.
//
// La couche existe depuis le lot 2.5.b, extraite de la grammaire par inversion de dependance.
// Ce rang pose la constante, son golden a historique et son gate ; il ne change AUCUNE largeur,
// AUCUNE borne, AUCUNE provenance, et aucun match deja decode n est candidat a quoi que ce soit.
// Ce qui change est la LISIBILITE du gate : jusqu ici la couche n etait hachee que dans
// l empreinte de la grammaire, qui rendait le meme diagnostic pour une borne de carte et pour un
// ordre de composants.
//
// ENTREE `profile-2026-09-17.2` (2026-09-17, lot 3.3.1) : L AMORCE DU RECORD DE CREATION DE
// PROJECTILE ENTRE AU PROFIL, PAR CLE ECRITE DANS LE FILM. LE DECODAGE CHANGE.
//
// `grenade.go` pose NEUF lignes — les sept builds du cache et les deux versions majeures des
// films sans section d identification, c est-a-dire exactement les clefs de la table des
// empreintes de registre du lot 3.2.1. Chaque ligne porte trois nombres MESURES : la largeur du
// motif d amorce (24 bits a partir de `HI_1_12_0`, 23 avant), la VALEUR de l amorce d etat par
// defaut (`0x40C00`, `0x20600`, et `0x20400` sur la majeure 31), et la position du champ d index
// de l auteur (+103, +100 ou +99).
//
// C EST UN CHANGEMENT DE DECODAGE, ET LE PLUS DIRECT QUI SOIT : jusqu ici `grammar` comparait
// 24 bits sur TOUS les films, donc lisait le bit de poids fort de l identifiant comme un bit
// d amorce sur les builds anciens. Les cinq temoins anciens du corpus publiaient ZERO lancer ;
// la mesure du volet recherche en compte 1 282, et celle de ce lot ajoute `a521164d` (105),
// `11de8353` (145) et `50247b26` (95).
//
// LA VALEUR DE L AMORCE EST UNE DONNEE, PAS UNE TRONCATURE. La majeure 31 porte `0x20400` la ou
// toutes les autres clefs anciennes portent `0x20600`, A LARGEUR EGALE : deriver l amorce
// ancienne de la recente par un decalage d un bit aurait marche sur huit clefs sur neuf.
//
// `grammar.Rev` et `facts.Rev` montent MECANIQUEMENT derriere cette ligne (elles hachent sa
// valeur, puis celle de `grammar.Rev`), et chacune porte son entree.
//
// ENTREE `profile-2026-09-17.3` (2026-09-17, lot 3.4.1-a) : LA LOI DES LARGEURS D AXE ENTRE
// DANS LA COUCHE, ET LA LARGEUR UNIFORME DEVINEE EN SORT. LE DECODAGE CHANGE.
//
//	`loi_largeurs.go`   NEUF : la transcription de `FUN_140be9b88` / `FUN_140be9c78` — le pas
//	                    du niveau, les deux gardes (2^22, epsilon 1e-4), le plafond 26, la
//	                    largeur d index de plage, et les bornes `+/-20000` du BUILD dont se
//	                    deduit la table DEFAUT (`22/22/22` au niveau du composant de position).
//	`map_bounds.go`     `MapQuantEntry.PrecisionAbsolue` — la projection de l entree de
//	                    catalogue vers le descripteur du chemin absolu ; `Layout()` en DERIVE,
//	                    au lieu de recomposer les memes largeurs a cote.
//	`profil.go`         `MovementProfile.AbsoluteAxisW` DISPARAIT. C etait une largeur UNIFORME
//	                    de 14 bits qui ecrasait les trois largeurs de la carte sur le chemin
//	                    absolu du bipede. `14` est bien une entree REELLE des tables du jeu,
//	                    mais au NIVEAU 8 de la table DEFAUT, quand le composant de position lit
//	                    au niveau 16 — ou la table defaut vaut `22/22/22` et la table par index
//	                    les largeurs de la carte (D3 (3.4)).
//	`profile_table.go`  la ligne PRESUMEE `Movement.AbsoluteAxisW` cede la place a la ligne
//	                    RELUE `Movement.LoiLargeursAxe` (« L=16 C=1/120 cap=26 garde=2^22
//	                    eps=1e-4 »), avec les trois fonctions et les quatre adresses de `.rdata`.
//
// LE DECODAGE CHANGE, ET LE COMPTE SE FERME OU IL NE SE FERMAIT PAS : sur Cliffhanger le chemin
// absolu du bipede lisait 49 bits (5 de porte + 3x14 + 2), il en lit 47 — exactement la mesure
// Cheat Engine du dispatch (une seule valeur distincte, 100 % de 154 158 releves).
const Rev = "profile-2026-09-17.3"

// L EMPREINTE DES SOURCES DE LA COUCHE VIT DANS UN GOLDEN, A COTE DE CETTE REVISION :
// `testdata/profile_rev.golden` porte le couple (revision, empreinte) avec son historique, et
// `rev_test.go` le compare aux sources non-test de ce paquet et a la valeur de `source.Rev`.
//
// POURQUOI LE GOLDEN PORTE LES DEUX VALEURS (lecon du gate `killsource`, revue adversariale du
// 2026-09-12, constat P1-4) : tant qu un gate ne compare que l EMPREINTE, remettre la revision a
// sa valeur d avant — en gardant la nouvelle empreinte — reste VERT. Le couple rend les deux
// derives visibles, avec deux messages distincts.
