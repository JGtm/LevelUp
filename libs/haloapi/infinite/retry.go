package infinite

import (
	"net/http"
	"time"

	haloapi "github.com/JGtm/halo-api"
)

// Retry : la politique de reprise d'un [Client] ([haloapi.Retry], partagée par les clients de la
// brique). Les trois politiques nommées ([RetryProvider], [RetryDiscovery], [RetryHaloClient]) sont
// celles que LevelUp emploie pour les services Halo Infinite.
type Retry = haloapi.Retry

// Limiter borne le débit des requêtes ([haloapi.Limiter]).
type Limiter = haloapi.Limiter

// Délais communs aux trois politiques.
const (
	delaiBaseReprise = 800 * time.Millisecond
	plafondReprise   = 10 * time.Second
	essaisProvider   = 3
	essaisDiscovery  = 3
	essaisHaloClient = 4
	premierStatut5xx = http.StatusInternalServerError
)

// RetryProvider : la politique des services du fournisseur (passe de combat, défis, service
// record, confidentialité, médailles, saisons, `FetchMatchStats`). 3 essais ; TOUT statut
// non-200 hors 401/403/404/410 est retenté, 429 compris ; Retry-After ignoré ; attente
// 800 ms·2^n bornée à 10 s, y compris après le dernier essai ; limiteur avant chaque essai, son
// erreur arrête.
func RetryProvider() Retry {
	return Retry{
		Essais:        essaisProvider,
		Retente:       func(int) bool { return true },
		DelaiBase:     delaiBaseReprise,
		Plafond:       plafondReprise,
		AttenteFinale: true,
	}
}

// RetryDiscovery : la politique des actifs Discovery UGC. 3 essais ;
// seuls les 5xx sont retentés (429 et tout autre 4xx sont rendus) ; Retry-After ignoré ; attente
// 800 ms puis 1,6 s entre les essais, aucune après le dernier ; une fin de contexte pendant
// l'attente rend `ctx.Err()` ; le limiteur n'est attendu qu'une fois, son erreur arrête.
func RetryDiscovery() Retry {
	return Retry{
		Essais:           essaisDiscovery,
		Retente:          func(s int) bool { return s >= premierStatut5xx },
		DelaiBase:        delaiBaseReprise,
		ArretSurContexte: true,
		LimiteurUneFois:  true,
	}
}

// RetryHaloClient : la politique du client de synchronisation (historique, stats, skill, CSR,
// carrière, apparence, playlists, manifestes de film ; ses essais et délais servent aussi aux blobs
// du CDN).
// 4 essais ; 429 est rendu IMMÉDIATEMENT (avec sa fenêtre Retry-After : le pool de l'appelant gère
// le refroidissement) ; tout autre statut non-200 est retenté ; attente 800 ms·2^n, au moins la
// fenêtre Retry-After, bornée à 10 s, y compris après le dernier essai ; limiteur avant chaque
// essai, son erreur ignorée.
func RetryHaloClient() Retry {
	return Retry{
		Essais:               essaisHaloClient,
		Retente:              func(s int) bool { return s != http.StatusTooManyRequests },
		HonoreRetryAfter:     true,
		DelaiBase:            delaiBaseReprise,
		Plafond:              plafondReprise,
		AttenteFinale:        true,
		IgnoreErreurLimiteur: true,
	}
}

// RetryAucune : un seul essai, pour les services au GET unique (aucune reprise, aucun limiteur) ;
// [New] exige une politique, le GET unique ne la consulte pas.
func RetryAucune() Retry { return Retry{Essais: 1} }
