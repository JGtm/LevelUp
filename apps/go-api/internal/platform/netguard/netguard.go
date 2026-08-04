// Package netguard — coupe-circuit des appels réseau SORTANTS en mode démo.
//
// POURQUOI. Le mode démo sert des données seedées et ne synchronise rien : aucun
// appel à une API tierce (Halo, Xbox, OAuth) n'y a de sens. Il n'était pourtant
// hermétique que par accident — « pas de tokens sur la machine ». Dès qu'une
// instance démo tourne sur un poste qui EN A (cas nominal : `LEVELUP_REPO_ROOT`
// pointe le checkout de dev, dont `data/auth/` contient les vrais tokens), le
// serveur s'authentifie et martèle l'API Halo pour les xuid FACTICES de la
// fixture (0000000000000000/1/2) : 400/404 en boucle, 4 tentatives, ~12 s par
// appel. Ces requêtes saturent le pool sortant et AFFAMENT le rendu des pages —
// symptôme observé : pages en timeout 30 s à un run, « données absentes » au
// suivant. Un harnais visuel adossé à ça n'est pas reproductible.
//
// CE QUE FAIT CE PACKAGE. Un drapeau process, posé UNE FOIS au boot depuis
// `cfg.DemoMode`, et vérifié à la FRONTIÈRE SORTANTE des clients HTTP tiers. En
// mode démo la requête n'est jamais émise : `Check` retourne `ErrOffline`, que
// chaque appelant traite par son chemin de dégradation DÉJÀ EN PLACE (fallback
// DB, catalogue statique, valeur vide). Aucune erreur avalée : le premier saut de
// chaque surface est loggé en INFO.
//
// POURQUOI un drapeau process et pas une injection par constructeur : les clients
// concernés sont construits à ~10 endroits (crons, wire, handlers, factories
// per-request), dont plusieurs sans accès à `*config.AppConfig`. Un drapeau posé
// au boot garantit la couverture par CONSTRUCTION — un client ajouté demain qui
// oublie le garde reste le seul risque, et c'est ce que verrouille le test
// `netguard_coverage_test.go` (scan AST des `client.Do(` des packages sortants).
//
// PORTÉE. Uniquement les appels vers des tiers. Le trafic interne (DuckDB, FS,
// serveur HTTP entrant) n'est pas concerné.
package netguard

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
)

// ErrOffline est retourné par Check quand le mode démo interdit la sortie.
// Les appelants doivent le traiter comme un échec de fetch BÉNIN (dégradation),
// jamais comme une erreur fatale : c'est un refus attendu, pas une panne.
var ErrOffline = errors.New("demo mode: external fetch skipped")

var (
	offline atomic.Bool
	// logged : une ligne INFO par surface, pas une par requête. Un cron qui
	// boucle ne doit pas noyer le journal, mais chaque surface coupée doit
	// apparaître AU MOINS une fois (sinon le saut serait invisible).
	logged sync.Map
)

// SetOffline arme (ou désarme) le coupe-circuit. Appelé une seule fois au boot
// depuis cmd/server/main.go avec cfg.DemoMode. Idempotent.
func SetOffline(on bool) {
	offline.Store(on)
	if !on {
		logged.Clear()
	}
}

// Offline indique si les sorties réseau tierces sont coupées.
func Offline() bool { return offline.Load() }

// Check est le garde à poser JUSTE AVANT l'émission d'une requête sortante.
// Retourne nil en fonctionnement normal ; ErrOffline en mode démo, après avoir
// loggé le saut (une fois par surface).
//
// `surface` nomme le point de sortie (ex. "halo_api", "gamecms_assets") : c'est
// ce qui apparaît dans le log et ce qui permet de savoir QUI aurait appelé.
func Check(ctx context.Context, surface string) error {
	if !offline.Load() {
		return nil
	}
	if _, seen := logged.LoadOrStore(surface, struct{}{}); !seen {
		slog.InfoContext(ctx, "demo mode: external fetch skipped",
			"surface", surface,
			"reason", "LEVELUP_DEMO_MODE=true — la démo sert des données seedées, aucun appel tiers")
	}
	return ErrOffline
}
