//go:build research

// Package mouvement — INSTRUMENT DU LOT 5.3 (RECHERCHE SEULE) : ou le film ecrit les ETATS DE
// MOUVEMENT du Spartan — accroupi, glissade, sprint, saut.
//
// # POURQUOI IL EXISTE
//
// La question posee est « a-t-on les evenements de joueur comme slide, crouch, sprint et
// saut ? ». Y repondre depuis `ecs_table.tsv` ne prouve rien : la table ne connait que les
// archetypes rencontres dans les films DEJA decodes, et elle melange l'ecrivain `+0x40` et son
// compagnon `+0x28` (decouverte de bord du lot 3.7, § 7.3). La reponse se cherche CHEZ
// L'ECRIVAIN — la fonction du jeu qui serialise le composant — et le NEGATIF se mesure sur
// l'univers des chaines de l'image, pas sur une impression.
//
// # CE QU'IL FAIT
//
// Deux passes, toutes deux en LECTURE SEULE sur `HaloInfinite.exe`, sans Ghidra et sans
// ouvrir un seul film :
//
//	CIBLES       la chaine du descripteur du lot 3.7 (`reapparition`) appliquee aux quatre
//	             composants de mouvement : nom -> accesseur -> descripteur -> ecrivain `+0x40`,
//	             puis releve des largeurs du corps de l'ecrivain. La publication reste gardee
//	             par [reapparition.Calibrer] : six temoins, six concordances, sinon rien.
//	VOCABULAIRE  le pool COMPLET des chaines de l'image filtre par les mots du mouvement.
//	             Un mot absent de ce balayage est absent de l'image, pas seulement du corpus —
//	             c'est la forme forte du negatif, celle qu'exige la doctrine « la grammaire
//	             prime, un negatif se MESURE ».
//
// # CE QU'IL NE FAIT PAS
//
// Il ne lit aucun film, n'ouvre aucune base et n'ecrit aucun artefact. La preuve SUR FILM est
// un pas distinct du lot, joue par un test cible sous le meme tag `research`, un film a la
// fois, et seulement sur voie libre du pilote.
package mouvement
