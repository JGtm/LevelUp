package main

// memlimit.go — LE PLAFOND MÉMOIRE du décodage d'UN job, côté ouvrier.
//
// MÊME DOCTRINE QUE cmd/levelup/backfill_memlimit.go (lot blindage, 2026-08-20/24),
// ADAPTÉE À UN PROCESSUS LONG-VIVANT. La passe hors ligne meurt d'un `os.Exit` par film : le
// code de sortie EST son protocole de retour vers un parent qui l'attend. L'ouvrier, lui,
// sert une FILE HTTP sans parent qui l'attend film par film — il ne peut pas se contenter de
// mourir en silence, il doit d'abord DIRE au serveur pourquoi avant de partir. D'où l'unique
// différence de fond : au lieu d'une ligne de protocole sur un tube stdout
// (`emettrePicMemoire`), la sentinelle appelle un CALLBACK qui rend compte au serveur
// (POST /build-queue/complete, error_code=memory_exceeded) puis arrête le processus — l'OS
// récupère la RAM par CONSTRUCTION, exactement comme pour l'enfant de la passe hors ligne.
//
// LES DEUX PLAFONDS ET LEUR RAISON D'ÊTRE (soupçon souple qui laisse le GC travailler, coupure
// dure au-delà) : voir le commentaire d'en-tête de cmd/levelup/backfill_memlimit.go, qui reste
// la référence. LES VALEURS SONT IDENTIQUES (3 GiB souple, +25 % dur, échantillonnage 250 ms) :
// mesurées sur le même film-bombe (`51101d1d`, 7,9 Go en 2,6 s) via le même
// internal/replaybuild.BuildMatch, il n'y a aucune raison de les faire diverger entre les
// deux binaires qui décodent le même film.
//
// POURQUOI CE FICHIER NE PARTAGE PAS DE CODE AVEC backfill_memlimit.go. Les deux fichiers
// vivent dans deux paquets `main` distincts (cmd/levelup, cmd/replay-worker) : Go ne permet
// pas d'importer un paquet main. Factoriser exigerait d'ouvrir un paquet interne partagé pour
// ~80 lignes de mesure de tas, entre deux outils dont la doctrine d'arrêt diverge légitimement
// (l'un meurt TOUJOURS après un film, l'autre sert une file et ne doit mourir QUE sur ce cas
// précis) — la duplication mesurée est le coût accepté plutôt qu'une dépendance croisée entre
// deux doctrines différentes.
//
// ARMÉ PAR JOB, PAS PAR PROCESSUS. Le pic doit être celui DE CE FILM, pas un maximum glissant
// depuis le démarrage de l'ouvrier qui confondrait plusieurs matchs. `processJob` arme une
// sentinelle fraîche avant chaque décodage et la désarme (`disarm`) dès qu'il se termine —
// succès ou échec ordinaire — avant de rearmer pour le job suivant.
//
// CE QUE CE PLAFOND NE PEUT PAS FAIRE : voir le même avertissement chez backfill_memlimit.go —
// la sentinelle ÉCHANTILLONNE, elle ne peut pas interrompre une allocation déjà en vol.

import (
	"log/slog"
	"runtime/debug"
	"runtime/metrics"
	"sync/atomic"
	"time"
)

const (
	// memGuardDefaultGiB : le défaut, en gibioctets — IDENTIQUE à plafondMemoireDefautGiB
	// de cmd/levelup (même corpus, même film de référence).
	memGuardDefaultGiB = 3

	// memGuardOctetsParGiB : la conversion, nommée pour ne pas semer des 1<<30 dans le code.
	memGuardOctetsParGiB = 1 << 30

	// memGuardPeriode : l'échantillonnage. Voir backfill_memlimit.go — mesuré à 250 ms pour
	// que le dépassement au moment de la coupure reste sous le gibioctet sur `51101d1d`.
	memGuardPeriode = 250 * time.Millisecond
)

// memGuardMargeDure : le plafond DUR se pose au-dessus du souple (+25 %). Sous le souple, le
// GC a le droit de travailler dur sans être abattu — c'est son rôle ; la sentinelle ne tranche
// que ce qui a franchi la zone où le GC aurait déjà dû reprendre la main.
func memGuardMargeDure(souple uint64) uint64 { return souple + souple/4 }

// Les deux compteurs qui composent l'empreinte gouvernée par `debug.SetMemoryLimit` : tout ce
// que le runtime a cartographié, moins ce qu'il a déjà rendu à l'OS. Mêmes noms de métriques
// que backfill_memlimit.go — c'est le même contrat runtime des deux côtés.
const (
	memGuardMetriqueTotal = "/memory/classes/total:bytes"
	memGuardMetriqueRendu = "/memory/classes/heap/released:bytes"
)

// memGuardEmpreinte rend l'empreinte courante, dans la même unité de compte que le plafond
// souple.
func memGuardEmpreinte() uint64 {
	ech := []metrics.Sample{{Name: memGuardMetriqueTotal}, {Name: memGuardMetriqueRendu}}
	metrics.Read(ech)
	if ech[0].Value.Kind() != metrics.KindUint64 || ech[1].Value.Kind() != metrics.KindUint64 {
		return 0
	}
	total, rendu := ech[0].Value.Uint64(), ech[1].Value.Uint64()
	if rendu > total {
		return 0
	}
	return total - rendu
}

// memoryGuard : le plafond dur d'UN job, et le pic observé pendant son décodage.
type memoryGuard struct {
	plafondDur uint64
	pic        atomic.Uint64
	stop       chan struct{}
}

// armMemoryGuard pose les deux plafonds pour la durée d'UN job et rend la sentinelle.
// onExceeded est appelé AU PLUS UNE FOIS, depuis la goroutine de la sentinelle, si
// l'empreinte franchit le plafond dur ; l'appelant y RAPPORTE l'échec au serveur puis arrête
// le processus (cf. job.go, reportMemoryExceeded). giB <= 0 désarme les deux plafonds — c'est
// l'échappatoire de l'opérateur qui sait ce qu'il fait, même sémantique que côté CLI.
func armMemoryGuard(giB int, onExceeded func(peakBytes uint64)) *memoryGuard {
	var plafondDur uint64
	if giB > 0 {
		souple := uint64(giB) * memGuardOctetsParGiB
		debug.SetMemoryLimit(int64(souple))
		plafondDur = memGuardMargeDure(souple)
		slog.Info("replay-worker: plafond memoire arme pour ce job",
			"souple_gib", giB, "dur_octets", plafondDur)
	}
	return newMemoryGuard(plafondDur, memGuardPeriode, onExceeded)
}

// newMemoryGuard : le constructeur bas niveau, séparé d'armMemoryGuard pour rester testable
// SANS dépendre de la mémoire réelle du processus — les tests posent un plafondDur et une
// période minuscules pour observer un déclenchement déterministe en quelques millisecondes
// (cf. memlimit_test.go).
func newMemoryGuard(plafondDur uint64, periode time.Duration, onExceeded func(peakBytes uint64)) *memoryGuard {
	g := &memoryGuard{plafondDur: plafondDur, stop: make(chan struct{})}
	go g.veiller(periode, onExceeded)
	return g
}

// veiller échantillonne l'empreinte, tient le pic, et déclenche onExceeded (une seule fois)
// au-delà du plafond dur.
func (g *memoryGuard) veiller(periode time.Duration, onExceeded func(peakBytes uint64)) {
	t := time.NewTicker(periode)
	defer t.Stop()
	for {
		select {
		case <-g.stop:
			return
		case <-t.C:
			v := memGuardEmpreinte()
			g.noterPic(v)
			if g.plafondDur > 0 && v > g.plafondDur {
				onExceeded(v)
				return // le callback rapporte puis arrête le processus : rien à revoir ici
			}
		}
	}
}

// noterPic retient le maximum observé.
func (g *memoryGuard) noterPic(v uint64) {
	for {
		ancien := g.pic.Load()
		if v <= ancien || g.pic.CompareAndSwap(ancien, v) {
			return
		}
	}
}

// disarm arrête la sentinelle SANS déclencher onExceeded : appelée quand le job se termine
// normalement (succès ou échec ordinaire), avant que le prochain job n'arme la sienne.
func (g *memoryGuard) disarm() {
	close(g.stop)
}
