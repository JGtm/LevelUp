package filmproc

// memguard.go — LA SENTINELLE MEMOIRE, canonique.
//
// DEUX PLAFONDS, PARCE QU'UN SEUL NE SUFFIT PAS. `debug.SetMemoryLimit` est un plafond SOUPLE :
// il ne tue rien, il fait travailler le GC de plus en plus fort a mesure qu'on s'en approche.
// Sur un tas qui depasse VRAIMENT la cible, il produit le sinistre du 2026-08-20 : une spirale
// ou le programme ne fait plus que collecter, des heures durant, jusqu'a ce que l'OS rende les
// armes. D'ou le second plafond, DUR : la sentinelle echantillonne l'empreinte et rend la main
// a l'appelant au-dela, pour qu'il arrete le processus. Le film est alors perdu — proprement,
// en quelques secondes, avec sa raison — au lieu de prendre la machine en otage.
//
// L'ARRET N'EST PAS DECIDE ICI, ET C'EST CE QUI REND LE PAQUET PARTAGEABLE. La sentinelle
// APPELLE un callback ; c'est l'appelant qui choisit sa doctrine d'arret — l'enfant d'une passe
// hors ligne rend un code de sortie a son parent, l'ouvrier rapporte d'abord au serveur puis
// s'arrete. Les deux doctrines divergent legitimement ; la mesure de tas, elle, est la meme.

import (
	"log/slog"
	"runtime/debug"
	"runtime/metrics"
	"sync/atomic"
	"time"
)

const (
	// DefaultLimitGiB : le plafond souple par defaut, en gibioctets.
	//
	// LE CORPUS L'A FIXE, PAS UN ARBITRAGE : il connait un film a 3,3 Go (`1b1e380f`, tue par
	// une surveillance externe le 2026-08-18) et un plafond de records pose depuis pour le
	// borner. 3 GiB laisse donc passer tout ce qui est sain sur ce corpus, et arrete ce qui
	// retombe dans ce travers.
	//
	// UNE MESURE QUI TOURNE SUR LE POSTE DE L'UTILISATEUR DOIT DESCENDRE PLUS BAS. La leçon du
	// 2026-08-26 : 3 GiB par enfant suffit a faire suffoquer une machine de travail quand la
	// mesure enchaine les films. Les outils de MESURE arment donc un plafond plus serré (cf.
	// MeasureLimitGiB) ; les passes de PRODUCTION gardent celui-ci.
	DefaultLimitGiB = 3

	// MeasureLimitGiB : le plafond des outils de MESURE, qui tournent sur la machine de
	// travail de l'utilisateur pendant qu'il s'en sert. Deux gibioctets couvrent les films du
	// corpus sain et laissent la machine respirer ; un film qui le depasse est precisement
	// celui qu'on ne veut pas laisser courir en tache de fond.
	MeasureLimitGiB = 2

	// octetsParGiB : la conversion, nommee pour ne pas semer des 1<<30 dans le code.
	octetsParGiB = 1 << 30

	// samplePeriod : l'echantillonnage. Mesure du 2026-08-24 : 250 ms garde le depassement au
	// moment de la coupure sous le gibioctet sur `51101d1d`.
	samplePeriod = 250 * time.Millisecond
)

// hardMargin : le plafond DUR se pose au-dessus du souple (+25 %). Sous le souple, le GC a le
// droit de travailler dur sans qu'on abatte le processus — c'est son role ; la sentinelle ne
// tranche que ce qui a franchi la zone ou le GC aurait deja du reprendre la main.
func hardMargin(soft uint64) uint64 { return soft + soft/4 }

// Les deux compteurs qui composent l'empreinte gouvernee par `debug.SetMemoryLimit` : tout ce
// que le runtime a cartographie, moins ce qu'il a deja rendu a l'OS.
const (
	metricTotal    = "/memory/classes/total:bytes"
	metricReleased = "/memory/classes/heap/released:bytes"
)

// Footprint rend l'empreinte courante, dans la meme unite de compte que le plafond souple.
func Footprint() uint64 {
	s := []metrics.Sample{{Name: metricTotal}, {Name: metricReleased}}
	metrics.Read(s)
	if s[0].Value.Kind() != metrics.KindUint64 || s[1].Value.Kind() != metrics.KindUint64 {
		return 0
	}
	total, released := s[0].Value.Uint64(), s[1].Value.Uint64()
	if released > total {
		return 0
	}
	return total - released
}

// Guard : le plafond dur d'UN decodage, et le pic observe pendant qu'il dure.
type Guard struct {
	hardLimit uint64
	peak      atomic.Uint64
	stop      chan struct{}
	stopped   atomic.Bool
	// probe remplace la mesure reelle dans les tests ; nil en production.
	//
	// IL EST DANS LA STRUCTURE ET NON EN VARIABLE DE PAQUET : deux tests paralleles qui
	// remplaceraient une variable globale se marcheraient dessus, et le defaut serait un test
	// vert par hasard.
	probe func() uint64
}

// HardLimitFor rend le plafond DUR qu'[Arm] posera pour ce nombre de gibioctets (0 quand la
// sentinelle est desarmee).
//
// IL EXISTE POUR QUE L'APPELANT PUISSE NOMMER CE PLAFOND DANS SON JOURNAL sans re-derouler le
// calcul chez lui : la ligne fatale de l'enfant de passe dit « voici l'empreinte atteinte ET le
// plafond franchi », et sans cet acces elle devrait recopier la marge de 25 % — exactement la
// duplication que le lot v2 G.1 a supprimee.
func HardLimitFor(giB int) uint64 {
	if giB <= 0 {
		return 0
	}
	return hardMargin(uint64(giB) * octetsParGiB)
}

// messageArmementParDefaut : la ligne d'armement des binaires qui n'en imposent pas d'autre.
const messageArmementParDefaut = "plafond memoire arme pour ce decodage"

// armConfig : ce qu'une option d'armement peut ajuster.
type armConfig struct{ message string }

// ArmOption ajuste l'armement de la sentinelle.
type ArmOption func(*armConfig)

// WithArmMessage impose le TEXTE de la ligne d'armement journalisee.
//
// POURQUOI UN APPELANT VOUDRAIT LE CHOISIR. Avant la centralisation (lot v2 G.1), chacun des
// deux `main` qui portaient leur propre copie de la sentinelle journalisait SON texte —
// « plafond memoire arme » cote passe de backfill, « replay-worker: plafond memoire arme pour
// ce job » cote ouvrier. Les unifier a change une sortie observable : un filtre de journal ou
// un oeil d'operateur cale sur ces libelles ne matchait plus (constat C5 de la revue R1). Le
// texte redevient donc le choix de l'appelant ; le defaut sert les six autres binaires, qui
// n'en avaient jamais eu d'autre.
func WithArmMessage(message string) ArmOption {
	return func(c *armConfig) { c.message = message }
}

// Arm pose les deux plafonds pour la duree d'UN decodage et rend la sentinelle.
//
// onExceeded est appele AU PLUS UNE FOIS, depuis la goroutine de la sentinelle, si l'empreinte
// franchit le plafond dur. L'appelant y applique SA doctrine d'arret (rendre un code de sortie,
// rapporter puis s'arreter...). `giB <= 0` desarme les deux plafonds : c'est l'echappatoire de
// l'operateur qui sait ce qu'il fait.
//
// LE NOM DE L'OUTIL VOYAGE DANS LE JOURNAL parce que huit binaires arment desormais cette
// sentinelle : sans lui, une ligne « plafond memoire arme » ne dirait pas qui l'a armee.
func Arm(tool string, giB int, onExceeded func(peakBytes uint64), opts ...ArmOption) *Guard {
	cfg := armConfig{message: messageArmementParDefaut}
	for _, o := range opts {
		o(&cfg)
	}
	hard := HardLimitFor(giB)
	if giB > 0 {
		debug.SetMemoryLimit(int64(uint64(giB) * octetsParGiB))
		slog.Info(cfg.message, "outil", tool, "souple_gib", giB, "dur_octets", hard)
	}
	return newGuard(hard, samplePeriod, onExceeded)
}

// newGuard : le constructeur bas niveau, separe d'Arm pour rester testable SANS dependre de la
// memoire reelle du processus — les tests posent un plafond et une periode minuscules pour
// observer un declenchement deterministe en quelques millisecondes.
func newGuard(hardLimit uint64, period time.Duration, onExceeded func(peakBytes uint64)) *Guard {
	g := &Guard{hardLimit: hardLimit, stop: make(chan struct{})}
	go g.watch(period, onExceeded)
	return g
}

// watch echantillonne l'empreinte, tient le pic, et declenche onExceeded (une seule fois)
// au-dela du plafond dur.
func (g *Guard) watch(period time.Duration, onExceeded func(peakBytes uint64)) {
	t := time.NewTicker(period)
	defer t.Stop()
	for {
		select {
		case <-g.stop:
			return
		case <-t.C:
			v := g.sample()
			g.notePeak(v)
			if g.hardLimit > 0 && v > g.hardLimit {
				if onExceeded != nil {
					onExceeded(v)
				}
				return // l'appelant applique sa doctrine d'arret : rien a revoir ici
			}
		}
	}
}

// sample rend l'empreinte a mesurer. Champ de test injectable : voir newGuardForTest.
func (g *Guard) sample() uint64 {
	if g.probe != nil {
		return g.probe()
	}
	return Footprint()
}

// Peak rend le maximum observe, en octets, EN INCLUANT L'INSTANT PRESENT.
//
// LA RE-MESURE N'EST PAS UN LUXE. L'echantillonnage periodique est a 250 ms : un processus qui
// meurt avant le premier tick — verrou de decodage refuse, constructeur indisponible, toute
// sortie tres precoce — n'a JAMAIS ete echantillonne. Sans elle, il emet un pic de ZERO et le
// recap de la passe imprime « (pic inconnu) » ; l'ancienne sentinelle copiee dans les deux
// `main` (picObserve, supprimee au lot v2 G.1) faisait deja exactement ceci.
func (g *Guard) Peak() uint64 {
	g.notePeak(g.sample())
	return g.peak.Load()
}

// notePeak retient le maximum observe.
func (g *Guard) notePeak(v uint64) {
	for {
		old := g.peak.Load()
		if v <= old || g.peak.CompareAndSwap(old, v) {
			return
		}
	}
}

// Disarm arrete la sentinelle SANS declencher onExceeded : appelee quand le decodage se termine
// normalement (succes ou echec ordinaire).
//
// IDEMPOTENTE, et ce n'est pas du confort : un appelant qui la differe ET l'appelle sur le
// chemin nominal fermerait deux fois le meme canal, ce qui panique.
func (g *Guard) Disarm() {
	if g.stopped.CompareAndSwap(false, true) {
		close(g.stop)
	}
}
