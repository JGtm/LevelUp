package halo5

import (
	"time"

	"github.com/JGtm/LevelUp/libs/haloapi"
)

// Paramètres de [RetryHalo5].
const (
	essaisHalo5    = 4
	delaiBaseHalo5 = 800 * time.Millisecond
	plafondHalo5   = 10 * time.Second
)

// RetryHalo5 : la politique des services Halo 5 que LevelUp emploie. 4 essais ; TOUT statut non-200
// hors 401/403/404/410 est retenté, 429 compris ; attente 800 ms·2^n, au moins la fenêtre
// Retry-After, bornée à 10 s, y compris après le dernier essai ; une fin de contexte pendant
// l'attente ne coupe pas la boucle ; limiteur avant chaque essai, son erreur rendue telle quelle.
func RetryHalo5() haloapi.Retry {
	return haloapi.Retry{
		Essais:           essaisHalo5,
		Retente:          func(int) bool { return true },
		HonoreRetryAfter: true,
		DelaiBase:        delaiBaseHalo5,
		Plafond:          plafondHalo5,
		AttenteFinale:    true,
	}
}
