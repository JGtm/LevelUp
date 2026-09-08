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

// AUCUN IMPORT : les METHODES de ce type vivent desormais dans identity_registry.go, seul
// fichier autorise a lire `SlotXUID` et `Owner` (garde-rail archlint, lot P2 2026-09-08). Ce
// fichier ne porte plus que la DECLARATION du rapport et sa documentation.

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
	// FromDeaths compte les slots nommés par la LECTURE SEULE, figé avant les fermetures.
	//
	// IL NE VAUT PLUS len(Owner) DEPUIS LES FERMETURES (2026-08-08), et le commentaire qui
	// l'affirmait a survécu au changement qui le rendait faux. L'écart entre les deux est
	// exactement ce que les déductions ont ajouté ; la règle de provenance qui remplace
	// l'égalité est vérifiée par `verdictOfBridge` :
	//
	//	FromDeaths + Closures.byShot + Closures.byRespawn == len(Owner)
	//
	// Un écart à CETTE somme signale une troisième source, non comptée.
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
	// DeathOffsetMS / DeathOffsetMatches portent le calage du fil des morts sur l'horloge du
	// film (`bestDeathOffset` : horlogeFilm = horlogeFil + DeathOffsetMS) et le nombre de
	// morts qu'il apparie.
	//
	// POURQUOI ILS SORTENT D'ICI. Ce calage est calculé pour NOMMER les vies ; il se trouve
	// être une SECONDE expression de l'origine du document (cf. origin.go), indépendante de
	// la lecture des en-têtes de paquet. Le republier coûte deux entiers et donne au témoin
	// une pièce qu'il ne partage pas avec ce qu'il contrôle. Zéro quand le pont n'a pas été
	// construit : il n'y a alors aucun témoin, ce qui n'est pas un désaccord.
	DeathOffsetMS      int64
	DeathOffsetMatches int
	// DeathOffsetRunnerUp : ce que le MEILLEUR DES AUTRES calages candidats aurait apparié.
	//
	// POURQUOI IL SORT D'ICI. Depuis le 2026-09-07 le calage n'est plus cherché par un balayage
	// exhaustif mais par un vote qui localise quelques candidats, puis par un affinage qui les
	// mesure (cf. bestDeathOffset). Une heuristique de localisation peut se tromper de panier ;
	// ce qui l'empêche de le faire en silence, c'est de publier À CÔTÉ du compte retenu celui de
	// son suivant. Un calage vrai écrase ses concurrents — mesuré x8,9 et x10,5 sur les deux
	// témoins du parc ; une marge qui se resserre est le signal qu'il faut aller regarder, et la
	// cuisson la journalise sous `deathOffsetMargeMin`.
	DeathOffsetRunnerUp int
	// SlotCollisions compte les slots dont les vies nommées désignent des joueurs différents.
	// Mesuré à 0 sur 000d5950 ; un film non nul invaliderait la table slot -> joueur.
	SlotCollisions int
	// Closures porte ce que les FERMETURES ont ajouté et refusé (cf. closures.go). Elles ne
	// sont pas des lectures : ce sont des déductions par élimination, et elles se comptent donc
	// à part de FromDeaths — sans quoi le pont dirait « tout vient de la lecture » alors que
	// non.
	Closures closureReport
	// SlotAmbiguous : les slots dont les vies nommées désignent des joueurs DIFFÉRENTS. Le pont
	// y garde le premier occupant nommé (`ownersFromLives`), ce qui est un choix par l'ORDRE DES
	// VIES et non par le temps : tout lecteur qui interroge `SlotXUID` sur l'un de ces slots
	// hérite d'un nom arbitraire. Le marqueur existe pour qu'il puisse s'abstenir — `xuidAt` le
	// fait déjà, et `SlotCollisions` en est le simple cardinal.
	SlotAmbiguous map[uint32]bool
	// lives : les vies découpées et nommées, telles que le nommage les a laissées. Interne au
	// paquet : c'est la source du nommage PAR VIE des tracks (nameTracksByLives, lot identité
	// des vies 2026-09-02) — un slot recyclé y porte une identité PAR OCCUPANT, là où SlotXUID
	// n'en retient qu'une par slot (première nommée, collisions comptées).
	lives []lifeSpan
}
