package main

// backfill_memlimit.go — LE PLAFOND MEMOIRE des processus de cuisson de films.
//
// # DEUX PLAFONDS, PARCE QU'UN SEUL NE SUFFIT PAS
//
// `debug.SetMemoryLimit` est un plafond SOUPLE : il ne tue rien, il fait travailler le GC
// de plus en plus fort a mesure qu'on s'en approche. Sur un tas qui depasse VRAIMENT la
// cible, il produit exactement le sinistre du 2026-08-20 : une spirale ou le programme ne
// fait plus que collecter, six heures durant, jusqu'a ce que l'OS rende les armes
// (`errno=1450`, ERROR_NO_SYSTEM_RESOURCES). Un plafond souple SEUL aurait donc deplace le
// probleme dans l'enfant, pas resolu.
//
// D'ou le second plafond, DUR : une sentinelle echantillonne l'empreinte et TUE le
// processus au-dela. Le film est alors perdu — proprement, en quelques secondes, avec sa
// raison — au lieu de prendre la machine en otage pour la nuit.
//
// # CE QUE CE PLAFOND NE PEUT PAS FAIRE (et il faut le savoir)
//
// La sentinelle ECHANTILLONNE : elle ne peut pas interrompre une allocation DEJA EN VOL. Un
// `make([]T, n)` de plusieurs gibioctets d'un seul tenant passe sous les deux plafonds — le
// souple ne refuse rien, la sentinelle ne voit le resultat qu'au tick suivant. Mesure du
// 2026-08-24 sur `51101d1d` : 7,9 Go atteints en 2,6 s, soit un depassement reel du plafond
// dur avant la coupure. Le processus meurt donc en QUELQUES SECONDES au lieu de quelques
// heures, et la machine survit — mais le pic transitoire, lui, a bien eu lieu.
//
// Le cran suivant, si un jour il le faut, n'est pas un reglage : c'est un Job Object Windows
// (`ProcessMemoryLimit`) pose sur l'enfant, qui fait ECHOUER l'allocation au lieu de la
// laisser aboutir. Non fait ici : le blindage par processus suffit a ce que la passe et la
// machine survivent, ce qui etait l'objet du lot.
//
// # LA SENTINELLE NE VIT QUE DANS UN PROCESSUS QUI MEURT APRES UN SEUL FILM
//
// Elle appelle `os.Exit`. C'est acceptable dans l'ENFANT, qui ne tient aucun handle
// d'ecriture : la passe `backfill-replay` relache ses handles DuckDB AVANT tout decodage
// (cf. cmd_backfill_replay.go), et l'artefact s'ecrit atomiquement. Ne JAMAIS la demarrer
// dans un processus qui tient une base en ecriture : une mort brutale au milieu d'une
// transaction est precisement ce que la doctrine anti-corruption interdit.

import (
	"log/slog"
	"os"
	"runtime/debug"
	"runtime/metrics"
	"sync/atomic"
	"time"
)

const (
	// plafondMemoireDefautGiB : le defaut, en gibioctets.
	//
	// Le corpus connait un film a 3,3 Go (`1b1e380f`, tue par une surveillance externe le
	// 2026-08-18) et un plafond de records pose depuis pour le borner. 3 GiB laisse donc
	// passer tout ce qui est sain sur ce corpus, et arrete ce qui retombe dans ce travers.
	plafondMemoireDefautGiB = 3

	// octetsParGiB : la conversion, nommee pour ne pas semer des 1<<30 dans le code.
	octetsParGiB = 1 << 30

	// periodeSentinelle : l'echantillonnage.
	//
	// MESURE DU 2026-08-24 : le film `51101d1d` monte a 7,9 Go en 2,6 s — environ 3 Go/s.
	// A 2 s d'echantillonnage il depassait le plafond dur de pres de 4 Go avant d'etre vu ;
	// a 250 ms le depassement retombe sous le gibioctet. Le cout est deux compteurs du
	// runtime par tick, c'est-a-dire rien — c'est la latence de coupure qui commande ici.
	periodeSentinelle = 250 * time.Millisecond
)

// margeDure : le plafond DUR se pose au-dessus du souple (ici +25 %). Sous le souple, le GC
// doit avoir le droit de travailler dur sans etre abattu — c'est son role. La sentinelle ne
// tranche que ce qui a franchi la zone ou le GC aurait deja du reprendre la main.
func margeDure(souple uint64) uint64 { return souple + souple/4 }

// Les deux compteurs qui composent l'empreinte gouvernee par `debug.SetMemoryLimit` : tout
// ce que le runtime a cartographie, moins ce qu'il a deja rendu a l'OS.
const (
	metriqueTotal = "/memory/classes/total:bytes"
	metriqueRendu = "/memory/classes/heap/released:bytes"
)

// empreinteMemoire rend l'empreinte courante, dans la MEME unite de compte que le plafond
// souple — sans quoi les deux plafonds parleraient de deux choses differentes.
func empreinteMemoire() uint64 {
	ech := []metrics.Sample{{Name: metriqueTotal}, {Name: metriqueRendu}}
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

// sentinelleMemoire : le plafond dur, et le pic observe.
type sentinelleMemoire struct {
	plafondDur uint64
	pic        atomic.Uint64
}

// armerPlafondMemoire pose les deux plafonds et rend la sentinelle, qui porte le pic.
//
// `gib <= 0` desarme les DEUX plafonds : c'est l'echappatoire de l'operateur qui sait ce
// qu'il fait (mesurer un film-bombe, par exemple). Le pic reste mesure — on veut le chiffre
// meme quand on ne veut pas la coupure.
func armerPlafondMemoire(gib int) *sentinelleMemoire {
	s := &sentinelleMemoire{}
	if gib > 0 {
		souple := uint64(gib) * octetsParGiB
		debug.SetMemoryLimit(int64(souple))
		s.plafondDur = margeDure(souple)
		slog.Info("plafond memoire arme",
			"souple_gib", gib, "dur_octets", s.plafondDur)
	}
	go s.veiller(periodeSentinelle)
	return s
}

// veiller echantillonne l'empreinte, tient le pic, et TUE au-dela du plafond dur.
func (s *sentinelleMemoire) veiller(periode time.Duration) {
	t := time.NewTicker(periode)
	defer t.Stop()
	for range t.C {
		v := empreinteMemoire()
		s.noterPic(v)
		if s.plafondDur > 0 && v > s.plafondDur {
			slog.Error("PLAFOND MEMOIRE DEPASSE — arret du processus",
				"empreinte_octets", v, "plafond_dur_octets", s.plafondDur)
			// Le pic part AVANT la mort : c'est la seule trace que le parent aura.
			emettrePicMemoire(v)
			os.Exit(codeEnfantMemoire)
		}
	}
}

// noterPic retient le maximum observe.
func (s *sentinelleMemoire) noterPic(v uint64) {
	for {
		ancien := s.pic.Load()
		if v <= ancien || s.pic.CompareAndSwap(ancien, v) {
			return
		}
	}
}

// picObserve rend le pic, en incluant l'instant present : un processus qui meurt vite peut
// n'avoir jamais ete echantillonne.
func (s *sentinelleMemoire) picObserve() uint64 {
	s.noterPic(empreinteMemoire())
	return s.pic.Load()
}
