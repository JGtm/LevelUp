package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
)

// WithEvent crée un identifiant d'événement court (8 octets hex = 16 chars)
// et le place dans le contexte. Tous les logs émis via slog.*Context depuis
// ce ctx auront `event_id=...` dans leurs attributs (cf. ContextHandler), ce
// qui permet de grep une opération multi-module dans les fichiers logs/.
//
// prefix : libellé court pour humain (ex: "sync.RunDelta", "swap.RoToRw",
// "backfill.events"). Apparait sous forme `event=<prefix>:<short_id>` —
// l'id reste assez court pour rester scannable en console.
//
// Usage typique :
//
//	ctx, eventID := logging.WithEvent(ctx, "sync.RunDelta")
//	slog.InfoContext(ctx, "sync: démarrage", "gamertag", gt) // log: event=sync.RunDelta:a1b2c3d4e5f60718
//	... // tous les sous-logs hériteront de event_id automatiquement
//	slog.InfoContext(ctx, "sync: terminé", "matches", n)     // même event_id
//
// Si ctx contient déjà un event_id (sous-événement), un nouvel id est créé
// quand même (un fils peut avoir son propre id pour granularité), mais le
// parent reste accessible via la chaîne d'appel. Pour propager le même id
// dans une sous-fonction, passer le ctx tel quel sans rappeler WithEvent.
func WithEvent(ctx context.Context, prefix string) (context.Context, string) {
	id := newShortEventID()
	full := id
	if prefix != "" {
		full = prefix + ":" + id
	}
	return ctxkeys.WithEventID(ctx, full), full
}

// CurrentEvent retourne l'event_id stocké dans le ctx (vide si absent).
// Helper de lecture pour les callers qui veulent embarquer l'id dans une
// réponse HTTP ou un message d'erreur.
func CurrentEvent(ctx context.Context) string {
	return ctxkeys.EventID(ctx)
}

// newShortEventID génère un id hexadécimal court (16 caractères = 8 octets
// crypto/rand). Collision pratique nulle pour le volume de logs LevelUp.
func newShortEventID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Crypto/rand échoue très rarement (panic POSIX-style en dernier
		// recours). Fallback non-secure mais déterministe pour rester
		// non-bloquant en cas de pénurie d'entropie au boot.
		return fmt.Sprintf("noent-%x", buf)
	}
	return hex.EncodeToString(buf[:])
}
