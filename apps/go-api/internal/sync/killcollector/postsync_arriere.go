package killcollector

// postsync_arriere.go — L ARRIERE DE FILMS QUE LA PASSE LAISSE DERRIERE ELLE.
//
// Trois lecteurs, une seule publication par passe ([publierArriere]) :
//
//	les jauges expvar         `killsource_postsync_backlog_restant` (tout ce qui reste, sans carte
//	                          compris) et `killsource_postsync_backlog_sans_carte` (la part que le
//	                          decodeur ne peut pas resorber) ;
//	l ordonnanceur et l ecran d etat de l historique, par `filmbacklog` : ce que la passe peut encore
//	                          traiter — le restant moins les matchs sans carte.
//
// LE RESTANT EST CELUI D APRES LA PASSE. Il compte ce que la passe n a pas atteint (budget de temps,
// arret) en plus de ce qu elle n avait pas retenu : une jauge calculee avant le decodage disait
// « 395 » alors que le budget venait d arreter la passe, et ne comptait pas les films retenus mais
// non traites.
//
// UN MATCH ESSAYE ET NON RESOLU N Y FIGURE PAS (echec, delai, film refuse) : la cadence courte qui
// les reprendrait sans fin ne les ferait pas avancer, et chaque reprise coute un telechargement.

import (
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/filmbacklog"
)

// Traites : les matchs que la passe a essayes, quel qu en soit le resultat. La difference avec le
// nombre demande est ce que la passe n a pas atteint (budget de temps, arret de l appelant).
func (s KillSourceSummary) Traites() int {
	return s.Written + s.NoFilm + s.NoKillFeed + s.Timeouts + s.Errors + s.NotSupport + s.UnknownKey + s.CarteNonResolue
}

// publierArriere publie l arriere de la passe : `restant` matchs au backlog, dont `sansCarte` que
// le decodeur ne peut pas resorber (carte non resolue).
func publierArriere(titre string, restant, sansCarte int) {
	observability.SetInt(CompteurPostSyncRetard, int64(restant))
	observability.SetInt(CompteurPostSyncSansCarte, int64(sansCarte))
	filmbacklog.Publier(titre, filmbacklog.Frags, restant-sansCarte)
}
