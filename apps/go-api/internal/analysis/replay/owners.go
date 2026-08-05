package replay

// owners.go — LE PONT SLOT -> JOUEUR. Une seule source : la LECTURE.
//
// CE QUI A ÉTÉ SUPPRIMÉ ICI, ET POURQUOI. Ce fichier portait deux méthodes qui faisaient ÉLIRE
// le propriétaire d'un slot : l'égalité des caps de visée, et la naissance du projectile d'un
// lancer de grenade. Elles ont été retirées le 2026-07-28 sur décision de l'utilisateur, dans
// des termes qui valent d'être conservés : « je préfère rien afficher que quelque chose de
// complètement faux ».
//
// LA MESURE QUI A FONDÉ CE RETRAIT, faite AVANT de supprimer quoi que ce soit :
//
//	                          avec le repli voté   sans lui
//	tirs publiés              496 / 519            475 / 519
//	lancers publiés           68 / 70              63 / 70
//	slots du pont             96                   90
//	DÉSACCORDS entre sources  4                    0
//
// On perd 21 tirs et 5 lancers — 4 % du calque. On gagne la propriété qui compte : **tout ce
// qui est affiché vient d'une lecture**, et plus rien ne peut être contredit par une autre
// méthode. Les 24 tirs qui changeaient de propriétaire selon la méthode employée disparaissent
// avec le problème : il n'y a plus de seconde méthode pour les revendiquer.
//
// CE QUE LE PONT FAIT, ET POURQUOI IL EXISTE ENCORE. L'événement de tir porte DÉJÀ son auteur —
// c'est le champ FilmIndex, écrit dans le film, jamais deviné. Le pont ne sert donc pas à
// trouver QUI a tiré. Il sert à trouver OÙ il était : les positions sont indexées par SLOT de
// biped, un slot change à chaque réapparition, et rien dans l'événement de tir ne le donne.
// Sans ce pont on connaîtrait le tireur sans pouvoir dessiner son tir.
//
// LE PONT EST DONC ENTIÈREMENT FAIT DE LECTURES, depuis le 2026-07-28 :
//
//	le fil des morts     nomme chaque vie (slot) par le XUID de sa victime
//	les 5 bits d'index   donnent, pour chaque XUID, son index de joueur dans le film
//	                     (cf. player_index.go — 26 chunks concordants sur 000d5950)
//
// La composition des deux donne slot -> index. L'affectation de coût minimal sur les 8!
// permutations qui tenait lieu de second maillon a été SUPPRIMÉE : elle donnait la bonne
// table, mais par un choix à marge étroite (32 contre 39) là où le film écrit la réponse.

// OwnerReport porte le pont et de quoi juger sa qualité. Publier un pont sans dire sur quoi il
// repose reviendrait à masquer la faiblesse de sa source.
type OwnerReport struct {
	// Owner est le pont : slot -> index de joueur du film.
	Owner map[uint32]int
	// SlotXUID est le MÊME pont exprimé en identités : slot -> xuid. Il sort du même parcours
	// et de la même règle de collision qu'Owner.
	//
	// POURQUOI LES DEUX. `Owner` sert au rattachement des ÉVÉNEMENTS, qui portent un index de
	// film. `SlotXUID` sert à publier QUI porte une trace, et un index publié serait
	// inexploitable par un client : il n'a de sens qu'à l'intérieur de ce film.
	SlotXUID map[uint32]uint64
	// FromDeaths compte les slots nommés par la lecture. C'est désormais la seule source, donc
	// ce compteur vaut toujours len(Owner) — il est conservé parce qu'un écart entre les deux
	// signalerait qu'une seconde source est réapparue sans qu'on l'ait voulu.
	FromDeaths int
	// DeathsNamed / LivesTotal disent combien de vies le fil des morts a nommées. Un rapport
	// publié sans son dénominateur ne se juge pas.
	DeathsNamed, LivesTotal int
	// IndexReadings est le nombre de chunks de réplication qui ont livré la MÊME table
	// identité -> index. Il remplace la « marge » de l'ancienne résolution par choix : là où
	// celle-ci disait « la bonne réponse gagne de 7 points », celui-ci dit « le film l'écrit
	// 26 fois de suite ».
	IndexReadings int
	// IndexDisagreements compte les identités que deux chunks ont lues différemment. Non nul,
	// la lecture est fausse et rien n'est publié pour ces joueurs.
	IndexDisagreements int
	// SlotCollisions compte les slots dont les vies nommées désignent des joueurs différents.
	// Mesuré à 0 sur 000d5950 ; un film non nul invaliderait la table slot -> joueur.
	SlotCollisions int
}

// buildOwners construit le pont à partir du seul fil des morts.
//
// PAS DE REPLI. Si le film ne porte pas son fil des morts, le pont est VIDE et aucun tir n'est
// publié — c'est le comportement voulu. Un rejeu muet se voit ; un rejeu qui pose des tirs sur
// le mauvais joueur ne se voit pas, et c'est bien pire.
func buildOwners(tracks map[uint32]slotTrack, deaths []Death, idx PlayerIndexTable) OwnerReport {
	rep := OwnerReport{Owner: map[uint32]int{}, SlotXUID: map[uint32]uint64{}}
	if len(deaths) == 0 || len(tracks) == 0 || len(idx.ByXUID) == 0 {
		return rep
	}
	lives := buildLifeSpans(tracks)
	rep.LivesTotal = len(lives)
	off, _ := bestDeathOffset(lives, deaths)
	rep.DeathsNamed = nameLivesByDeaths(lives, deaths, off)
	if rep.DeathsNamed == 0 {
		return rep
	}
	rep.IndexReadings = idx.Readings
	rep.IndexDisagreements = idx.Disagreements
	owners, byXUID, collisions := ownersFromLives(lives, idx.ByXUID)
	rep.SlotCollisions = collisions
	rep.Owner = owners
	rep.SlotXUID = byXUID
	rep.FromDeaths = len(owners)
	return rep
}
