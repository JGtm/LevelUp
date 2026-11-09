package killsource

// kill_publication.go — LA PORTE DE PUBLICATION LIGNE PAR LIGNE D UN [Result], sortie de kill.go par
// deplacement pur pour tenir le seuil de 500 lignes.

// LineByLinePublishable : les attributions ligne par ligne sont-elles publiables ?
//
// DEUX CONDITIONS, et la seconde est celle qui a interdit le BTB : la bijection ne doit pas etre
// AMBIGUE, et la sante ne doit pas etre en ALERTE. Un `false` ne dit pas que le decodage est faux :
// il dit que seul l agregat l est.
//
// CE QUE « NON AMBIGUE » VEUT DIRE A CHANGE AU LOT 1.8, ET LA RAISON EST STRUCTURELLE.
// La marge de bijection mesure l ecart entre la meilleure INFERENCE et la deuxieme meilleure : une
// marge nulle dit que deux joueurs sont interchangeables AUX YEUX DE L INFERENCE (RE_LOG 7ter.53).
// Depuis que la table du film epingle les indices qu elle LIT (film_table.go), cette question ne
// se pose plus que sur ce qui RESTE a l inference : quand il ne lui reste qu une seule affectation
// a rendre, il n y a rien d interchangeable — la marge est alors SANS OBJET, pas nulle. Confondre
// les deux ferait refuser toutes les lignes d un film entierement lu, ce qui serait exactement
// l inverse de ce que la lecture apporte.
//
// « CE QUI RESTE » A DEUX COTES, ET LE LOT 1.8 N EN COMPTAIT QU UN (revue de jalon M1, lentille
// L4). Le critere etait « au plus un INDICE libre » ; il lui manquait « au plus un NOM libre ».
// Un indice libre pour deux noms libres n a rien de determine : le hongrois le tranche par les
// votes du kill-feed, nuls pour un joueur qui n a ni tue ni ete tue, donc arbitrairement — et
// c est TOUT ce que la ligne porte qui partait alors sur un occupant tire au sort. Le critere est
// desormais [FilmTablePinning.AffectationUnique].
//
// [Roster.FilmTable] dit toujours laquelle des deux voies a decide, indice par indice
// (`Pinned` / `Inferred`), et [Result.BijectionMargin] garde son sens d origine : le verdict de
// l INFERENCE sur sa propre part.
//
// CETTE PORTE VAUT POUR TOUT CE QUE LA LIGNE PORTE — la source du degat, LE CREDIT, L ASSISTANT et
// LES DEUX PARTS DE DEGATS. C est une DECISION, et elle refuse deliberement une porte separee par
// donnee : l assistant est nomme par la MEME bijection indice -> joueur que le reste, il tombe
// donc exactement au meme endroit. La mesure BTB le confirme sans ambiguite — 62 assistants nommes
// pour 122 a l API (51 %), contre 17/17 et 29/29 en Arena.
func (r *Result) LineByLinePublishable() bool {
	if len(r.Health.Alerts()) != 0 {
		return false
	}
	return r.BijectionMargin > 0 || r.BijectionDetermined
}
