# Constats A2.T (type d'une entite lie a son eid complet)

Instruments : `C:\t\marche\A2T\mesure.sh` (fermeture sous carte, 38 films de `liste40.tsv` = 30 films de films_30.txt + 8 anciens ; divergence contre le temoin, 10 films de temoins_duree.txt), `compare.sh`, `pq_all.sh` (detail de chaque paquet). Avant = HEAD 37688edefd (exe `div_avant.test.exe`). Sorties : `avant/`, `essai_a1/` (regle appliquee a la garde et a l'anticipation).

## Lecture du jeu (note A2, point 3)
- Garde de delta : `*(uint*)(slot*200+base) != eid` -> code 3, ZERO bit lu : le slot ET la generation.
- Creation : `entree[0] = eid` complet, `entree[1] = type`. Reallocation sous une autre generation : les deux sont ecrases. Un delta de l'ancienne generation echoue la garde.
- Suppression : n'efface pas `entree[0]` ni `entree[1]` dans les fonctions lues (non verifie ailleurs).

## Code d'avant
- `rejetDeVue` (frame_infer.go) ne testait que « slot lie » (`ArchetypeForSlot`). Le champ `slotState.FullID` / `GenAny` existait mais seule la voie legacy `DecodeFrameRecords` (et le localisateur) le consultait (`GenerationMatches`, mode strict).
- Liaisons a generation connue (`GenAny` faux) : NEW lu (`BindFull`), inference (`BindSoft`). Liaisons de generation inconnue (`GenAny` vrai) : image-cle (`BindImageCle`), table de datums, anticipation, joker : le lot 5.13.1 a mesure que les deux bits de tete d'une image-cle et ceux d'un delta ne sont pas le meme champ.

## Mesure
- Regle appliquee (garde de la vue B + condition du repli d'anticipation sur l'eid) : fermeture IDENTIQUE sur les 38 films (0 paquet de plus ou de moins sur aucun film, aucun build ; voir `compare.sh avant essai_a1`). Table « films » de la divergence IDENTIQUE (memes `en desaccord`, `nous seuls`, `temoin seul`).
- La garde se declenche (1 090 evenements « slot lie sous une autre generation » sur les 38 films, deux contextes, marches d'essai comprises, toutes liaisons `LueNeuf` de generation connue) sans changer aucun verdict de paquet.
- Les 1 738 premiers desaccords « type de l'entite » des dix temoins : 1 578 paquets non fermes chez nous, tous sous une liaison d'IMAGE-CLE (liaison 2, generation inconnue) ; 154 fermes sous une liaison NEW lue (liaison 1, generation egale des deux cotes) ; 6 fermes sous image-cle. Exemples : 8b1f15cb slot 62 (nous ti=5 a CHAQUE chunk par la chaine d'ancres de l'image-cle, temoin type 19) ; 672641d0 slot 4 (ti=6 aux chunks 1 et 16, ti=2 au chunk 11, temoin 6) ; 221dd082 slot 1047 gen 1 (NEW lu ti=41 au chunk 4, temoin 36, memes generations). Ce n'est pas une confusion de generation : c'est le type que NOTRE lecture d'image-cle ou de NEW donne au slot.

## Etat final de A2.T (arbre : world.go, frame_infer.go, tests, revisions, goldens)
- Code : `World.ArchetypeForEID` (garde de `FUN_1406caad8` : eid entier, generation inconnue = repond a toute generation) ; `rejetDeVue` l'utilise ; `LierParRepliDAnticipation` traite un slot lie sous une autre generation comme un eid non lie (sa liaison est remplacee). DEL inchange (delie le slot : un DEL lu prouve que l'occupant precedent n'est plus la, meme si le jeu garde l'entree). Liaisons d'image-cle / datum / anticipation : generation inconnue, inchangees.
- Tests synthetiques `entite_par_eid_test.go` (5) : rouges sur l'ancienne garde (mutation `ArchetypeForSlot` dans rejetDeVue : 2 tests rouges).
- Fermeture, 38 films, avant/apres : IDENTIQUE par build et par film (compare.sh avant apres : +0 partout). Divergence 10 temoins : table des films identique, `en desaccord` somme 262 -> 262, `nous seuls`, `temoin seul` identiques.
- Detail par paquet (pq_all.sh, 38 films) : 12 paquets changent sur 3 films (8b1f15cb 10, 3f0a1fcb 1, c6238946 1), TOUS non fermes avant et apres : la vue B sort une etape plus tot par rejet hors datum au lieu de lire un delta sous un type perime (fin de paquet par terminateur hors cadre / invariant d'ecrivain avant).
- Mini-bobine ks_000d5950 (ratchet) : records lus sous un type perime ne se lisent plus (ti=30 0/3 disparu, ti=33 0/5 -> 0/4, ti=8 0/14 -> 0/12, ti=35 28 879 -> 28 878) ; paquets et au bit inchanges ; aucune ligne `fermes` ne baisse.
- Builds anciens : 0 paquet ne change sur les 8 films anciens, fermes identiques.
- grammar.Rev -> grammar-2026-11-10.2 ; killsource.Rev et objectives.Rev constants (golden regenere ; minibobine.golden de killsource identique).
