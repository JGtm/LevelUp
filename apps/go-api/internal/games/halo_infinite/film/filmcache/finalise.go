package filmcache

// finalise.go — UN FILM N'EST ARCHIVE, DECODE NI CUIT QUE FINALISE (lot L3 de
// PLAN_RETOURS_REJEU_2026-09-23, 2026-09-23).
//
// # LA REGLE, LUE DANS LA GRAMMAIRE DU FILM
//
// Un film Theater se publie en deux temps cote serveur Halo : les morceaux de replication
// arrivent pendant la partie, puis, environ une minute apres la fin, le serveur FINALISE le
// film en ajoutant son morceau de TEMPS FORTS (`chunk_type 3` : fil des morts et kill-feed),
// par definition le dernier. Un manifeste qui ne le porte pas decrit donc un film EN COURS DE
// PUBLICATION, pas un film complet. Mesure du 2026-09-23 sur le cache : 1 624 manifestes sur
// 1 625 portaient ce morceau ; l'exception (`ab526724`, match detecte 37 s apres sa fin) avait
// ete archivee 50 s apres la fin, sur 34 morceaux au lieu de 37, et TOUTE sa chaine d'identite
// etait tombee (fil des morts illisible, pont statborg a 3/8, drapeau et actions perdus).
//
// # UN SEUL PREDICAT, ET UN RATCHET
//
// [Finalise] est LE predicat : le writer ([Write]), le telechargement
// (`sync/haloclient.fetchFilmChunks`), le killsource (`sync/killcollector.LocalCacheFilms`), la
// cuisson (`replaybuild`) et le fil des morts (`replay.ScanDeaths`) le partagent. Une comparaison
// au type 3 recopiee ailleurs re-divergerait : `archlint/film_finalise_predicate_test.go` interdit
// toute comparaison au type des temps forts hors de ce fichier (allowlist datee des sites
// anterieurs au lot, chacun avec son critere de retrait).
//
// # CE QUE LA REGLE COUTE SI ELLE SE TROMPE
//
// La regle est LUE dans la grammaire du film (le serveur ecrit ce morceau en dernier), et elle
// est MESUREE : 1 624 manifestes sur 1 625 la verifiaient le 2026-09-23. Elle n'est donc pas une
// loi garantie pour les builds futurs, et son cout si elle se trompait est ecrit ici plutot que
// tu (constat L3-R1 de la revue adverse du lot) : un film qui ne serait JAMAIS finalise n'est ni
// archive ni cuit, et il EXPIRE cote serveur si personne ne s'en apercoit. Il n'est jamais marque
// (aucun marqueur terminal) : il est retente a chaque cycle dans la borne de l'horizon de
// rattrapage, chaque refus se compte chez l'appelant, et au-dela du delai de finalisation
// (`sync/replayartifacts.DelaiDeFinalisation`) il sort du journal INFO pour un WARN sous un
// compteur distinct, qui doit rester a zero. Un film vit des semaines cote serveur et
// `levelup archive-films` rattrape tout match sans film au cache : le signal laisse le temps
// d'instruire la derive et d'archiver apres correction.

import (
	"errors"
)

// ChunkTypeTempsForts : le type de manifeste du morceau des TEMPS FORTS (fil des morts,
// kill-feed). C'est LE litteral du depot ; `haloclient.FilmChunkTypeHighlightEvents` le reprend.
const ChunkTypeTempsForts = 3

// ErrFilmNonFinalise : le film n'est pas (encore) finalise — son manifeste ne porte pas le
// morceau des temps forts, ou des morceaux du film ne sont pas decrits par son manifeste.
//
// CE N'EST NI UN FILM ABSENT NI UNE PANNE. Absent (404/410) poserait le marqueur TERMINAL du
// registre et retirerait le film de tout rattrapage ; une panne ferait chercher un incident. Le
// film existe, il sera complet au cycle suivant : l'appelant REPORTE, compte et journalise.
//
// SON TEXTE TRAVERSE UNE FRONTIERE DE PROCESSUS : l'enfant de cuisson rend un `stderr`, et
// `sync/replayartifacts` classe sur ce texte (meme contrat que `replaybuild.ErrUnknownFilmKey`).
var ErrFilmNonFinalise = errors.New("film non finalise : le manifeste ne porte pas le morceau " +
	"des temps forts (chunk_type 3)")

// EstTempsForts dit si un type de manifeste est celui du morceau des temps forts. C'est la
// seule comparaison au type 3 permise hors de [Finalise] : elle SELECTIONNE ce morceau (fil des
// morts, kill-feed), la ou [Finalise] juge le film entier.
func EstTempsForts(chunkType int) bool { return chunkType == ChunkTypeTempsForts }

// Finalise dit si une liste de morceaux decrit un film FINALISE : elle porte le morceau des
// temps forts. `typeDe` rend le type de manifeste d'un element — le predicat sert quatre formes
// de morceau (manifeste de l'API, manifeste du cache, morceau a ecrire, metadonnees d'un film
// charge) et ne doit en connaitre aucune.
//
// Une liste VIDE n'est pas finalisee. Les appelants qui traitent « zero morceau » comme un film
// ABSENT (manifeste API vide d'un film expire) le testent AVANT, et c'est voulu : les deux cas
// ne se soignent pas pareil.
func Finalise[C any](chunks []C, typeDe func(C) int) bool {
	for _, c := range chunks {
		if EstTempsForts(typeDe(c)) {
			return true
		}
	}
	return false
}
