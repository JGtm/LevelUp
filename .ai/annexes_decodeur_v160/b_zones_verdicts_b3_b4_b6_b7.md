# Piste bl-zones : B.3, B.7, B.4, B.6 (2026-11-23)

Worktree `dec-v160-bl-zones` (branche `piste/bl-zones`, tete `46be3cf`). Aucun commit. Fichiers ajoutes au depot (non suivis) :
`replay/zone_lettres_controle.go` et `replay/zone_lettres_controle_test.go`. Aucune insertion dans un fichier reserve ni dans un fichier existant.
Sondes jetables (`research/blzones/`, test de recherche dans `replay/`) retirees ; `git status` : ces deux fichiers seulement.
Pieces hors depot : `rapport.md`, `fragments.md`, `sorties/` (TSV), `mesure.py`, `marqueurs.js`, `run.sh`, `bin/`.

## B.3 Equipe du porteur sur 6 socles : verdict [!] (aucun code)
- verifie (film 390b1de5 decode a la tete) : `carrierTeamUnknown` = 32 sur 32 portages (25 avant l'ouverture de la bande). Le defaut du compteur existe encore.
- verifie : l'equipe est lue pour 25 joueurs sur 25 (roster) ; `TeamOf` est rempli. Ce que `flagOfCarrier` (`replay/flag_carries_handoff.go`) ne sait pas rendre est le DRAPEAU : 3 socles par camp, donc 3 drapeaux adverses, « rien ne les departage ». Le compteur mesure l'abstention de la regle, pas une equipe inconnue.
- verifie : consequence mesurable nulle. Portages « au socle » a plus de 8 m : 0 ; chevauchements : 0 ; temps de porteur a 2 s de l'API : 24 joueurs sur 25 (le 25e : 15,6 s contre 13,6 s, 16 portages courts du meme joueur, un ecart d'une image par portage, sans lien avec le camp) ; les 6 drapeaux terminent au socle, 9 retours et 3 captures comme l'API.
- Nommer « le drapeau pris » par le film demanderait l'identite de l'objet pris a la prise (non lue) ou la geometrie (deja faite par `assignFlags`, apres le bornage qui en depend). Rien ne se prouve meilleur : pas de code.
- deduit : les trois regles qui nomment un drapeau par l'equipe (passage de main, retour credite, chute creditee) se taisent sur une carte a 3 drapeaux par camp ; sans effet observable sur le seul film connu. Verifiable sur un second film a 6 socles, non trouve.

## B.7 Zone au proprietaire de nom inconnu (Recharge) : verdict [!] (non reproduit)
- Le film est c4f6ce05 (Bases, Recharge). verifie a la tete (decode, carte donnee) : `ownerNamed` 3 sur 3 zones, `ownerVoteDisagreed` 0, `unknownOwner` 0, `ownerChecked` = `ownerAgreed` = 80, aucun repli `repli_zone_proprietaire_par_vote` ni `repli_zone_pousseur_par_election`. Second film de Bases sur Recharge (81eda4a6) : identique (3/3, 91/91).
- verifie (instrument `TestZoneProprietaireNoms`) : les trois noms de proprietaire (trois hachages de noms) sont releves aux images-cles de c4f6ce05, donc au vocabulaire `zoneBlocsNommes`.
- Cause de l'entree du backlog, verifie sur le TSV de la recherche G.1 : la sonde gardait le dernier nom lu par slot, lectures delta comprises ; le slot 1440 (proprietaire du bloc 2) porte trois lectures delta de nom different, au milieu d'une marche delta qui y lit aussi des types 5. Le rejeu ne nomme les slots que par les lectures d'image-cle (`replay/zone_states.go:181`). Defaut de la sonde, pas du rejeu.
- Parc v1.4.1, 77 films de Bases : `ownerNamed` = `paired` = 3 partout ; zero repli par vote.

## B.4 Compteurs de zones, anciens builds : verdict [!] (aucun code)
- verifie (a521164d, a349fea8 ; Total Control, version 33, sans section d'identification) : le composant 21 cote A (nomme `zone_secures` par `namedStatSlots`) n'est pas un compteur : 43 valeurs sur 44 (a521164d) et 55 sur 76 (a349fea8) sont congrues a 8 modulo 32, de la forme 8 + 32k dans un octet signe (40, 72, 104, -120, -88, -56, -24) ; le reste est hors domaine (-30707, 26638, 2062). Le cote B (`zone_offensive_kills`) y porte 6 a 123 puis 17930 : un autre champ. Le nom 21 A / 21 B vaut pour les builds recents, pas pour ces films.
- verifie (`dec-v150-recup/c5/defs_brut.tsv` joint a `builds.txt`) : la definition des emplacements du tableau (nom, type) n'existe dans le chunk 0 que pour HI_1_12_0 et HI_1_13_0 (45 films sur 45) ; elle est absente de HI_1_8_0 a HI_1_11_0 (10 sur 10) et des films sans identification. Le film ne donne aucun fait pour borner ces builds ; une table par version ecrite a la main est exclue par la regle. Aucun lecteur Go de ces definitions n'existe dans le depot (seul le temoin Rust de Den).
- verifie : sans effet aujourd'hui, `objectives[]` vide sur les deux films (`refusedByRoster` 76 et 83). Pas de gate des zones a defendre : aucun code.
- Piste si le pilote veut un garde : ne servir la table des noms d'emplacements que si la definition du film la confirme ; demande un lecteur Go des definitions (travail propre, Rust de Den en temoin, avec credit), hors de cet item.

## B.6 Lettre des zones comme controle : calcul FAIT et mesure, lecture NON FAITE
- Fait : `zoneLettersControlOf(markers []ZoneLetterMarker, zones []ZoneState, frameMS int) zoneLettersControl{Read, Agreed, Unmatched}`. Decalage d'horloge unique par film (mode des ecarts, pas de 1,5 s, ordre total), concordance de changements de proprietaire a equipe egale, bijection exigee (meilleur score strictement unique, zone reclamee par un seul marqueur) ; ne modifie jamais `LetterRank`. Tests sur donnees construites verts : accord 3/3, lettres fausses lues non accordees, bijection, egalite, silence, independance de l'ordre, division par le bas.
- Mesure (verifie) : marqueurs releves par la recherche G.1 (`dec-v150-recup/g/tsv`, 77 films de Bases) confrontes aux `zoneStates` des documents du parc : `Read` 3, `Agreed` 3, `Unmatched` 0 sur 77 films sur 77 (231 zones sur 231) ; temoin negatif (lettres decalees d'un cran) : `Agreed` 0 sur 77. Les deux films « incomplets » de la recherche (c4f6ce05, 6bc544a1) sont a 3/3 ici.
- NON fait : la SOURCE. Le calcul est pur et n'est appele par rien (code non branche, a signaler). Il manque (1) un balayage grammar qui publie, pour les marqueurs `ti=12`, le nom du groupe d'etats `control_point`, l'etat (empreinte de la lettre) et l'equipe de la premiere clause du groupe 0 (`consumeNavpointVisualStateGroup` traverse deja ces valeurs sans les publier ; modele : `objective_scan.go`) ; (2) son versement aux faits persistes (fichiers reserves) ; (3) les champs de couverture.
- NON fait : lecture du composant `ti=12 i19` qui arrete les 11 530 records de marqueur d'image-cle. Sa largeur n'est prouvable ni par le depot ni par le temoin sans lecteur du jeu (Ghidra indisponible) ; seule une mesure de fermeture des records (0 bit supposee par le temoin, ligne de version 2 et plus) la prouverait : recherche non commencee. Elle n'est pas necessaire au controle (les changements de proprietaire se lisent en delta).

## Gate et preuves
- verifie : aucune sortie de production ne change : documents du decodeur avant/apres (binaire `replay-build` bati sur le worktree) octet pour octet identiques sur 390b1de5, c4f6ce05, a521164d. Le gate du drapeau (aucun juste devenu faux ; 24 joueurs sur 25 a 2 s sur 390b1de5) et les compteurs de zones sont donc tenus par construction ; non rejoues sur six films ni a un et quatre fils (aucun code de production touche).
- Tests neufs verts ; `go vet ./...` vert, `go vet -tags=research ./...` vert (CGO actif, gcc msys64) ; `gofmt -l .` vide.
- Ce qui rougit : rien (aucun fichier existant, fixture ni golden modifie).
