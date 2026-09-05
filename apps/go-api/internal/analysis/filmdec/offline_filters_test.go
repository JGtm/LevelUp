package filmdec

import "testing"

func bp(slot uint32, ms int, x, y float32) BipedPosition {
	return BipedPosition{Slot: slot, TimestampUS: uint64(ms) * 1000, X: x, Y: y}
}

// TestDropIsolated reproduit la signature réelle des faux positifs de 000d5950 : un point
// unique séparé de plus de 60 s du reste de son slot, en tête comme en queue de séquence.
func TestDropIsolated(t *testing.T) {
	in := []BipedPosition{
		bp(513, 0, 1, 1),
		bp(513, 16, 1, 1),
		bp(513, 32, 1, 1),
		bp(513, 200_000, -37, -55), // isolé de 200 s en fin de slot -> écarté
		bp(577, 0, -12, -54),       // isolé en tête de slot (170 s avant la vie) -> écarté
		bp(577, 170_000, 5, 12),
		bp(577, 170_016, 5, 12),
	}
	got := DropIsolated(in, DefaultIsolationGapMS)
	if len(got) != 5 {
		t.Fatalf("attendu 5 positions conservées, obtenu %d (%+v)", len(got), got)
	}
	for _, p := range got {
		if p.X < -10 {
			t.Errorf("aberration conservée: %+v", p)
		}
	}
	// slot réduit à un seul échantillon : aucun voisin -> écarté
	if got := DropIsolated([]BipedPosition{bp(600, 5000, 1, 1)}, DefaultIsolationGapMS); len(got) != 0 {
		t.Errorf("un slot à un seul échantillon devrait disparaître, obtenu %+v", got)
	}
	// filtre désactivé : rien n'est touché
	if got := DropIsolated(in, 0); len(got) != len(in) {
		t.Errorf("gapMS=0 devrait désactiver le filtre, obtenu %d/%d", len(got), len(in))
	}
}

// TestDropTeleports : un saut instantané est écarté, la trajectoire continue conservée, et
// une série longue de rejets réancre le filtre (pas de slot condamné par une mauvaise ancre).
func TestDropTeleports(t *testing.T) {
	in := []BipedPosition{
		bp(513, 0, 0, 0),
		bp(513, 16, 0.1, 0), // 6,25 m/s : normal
		bp(513, 32, 60, 0),  // ~3700 m/s : téléportation
		bp(513, 48, 0.2, 0), // retour sur la trajectoire
	}
	got := DropTeleports(in, DefaultMaxSpeedMPS)
	if len(got) != 3 {
		t.Fatalf("attendu 3 positions, obtenu %d (%+v)", len(got), got)
	}
	for _, p := range got {
		if p.X > 1 {
			t.Errorf("téléportation conservée: %+v", p)
		}
	}

	// 4 rejets d'affilée : le 4e est accepté (réancrage) — sinon une ancre fausse
	// supprimerait tout le reste du slot.
	seq := []BipedPosition{bp(600, 0, 0, 0)}
	for i := 1; i <= 4; i++ {
		seq = append(seq, bp(600, i*16, float32(100*i), 0))
	}
	if got := DropTeleports(seq, DefaultMaxSpeedMPS); len(got) != 2 {
		t.Errorf("attendu 2 positions (ancre + réancrage après %d rejets), obtenu %d", maxRejectStreak, len(got))
	}

	if got := DropTeleports(in, 0); len(got) != len(in) {
		t.Errorf("maxSpeed=0 devrait désactiver le filtre, obtenu %d/%d", len(got), len(in))
	}
}

// TestDropTeleportsExcept : l'exemption D2 — le filtre est levé à ±200 ms d'un événement 117
// du MÊME slot, et JAMAIS ailleurs. Le cas d'essai reproduit la forme mesurée (R3 §2) : une
// arrivée de téléportation à ~1400 m/s, suivie de la trajectoire réelle.
func TestDropTeleportsExcept(t *testing.T) {
	in := []BipedPosition{
		bp(535, 0, 0, 0),
		bp(535, 17, 0.1, 0),  // trajectoire normale
		bp(535, 34, 22, 0),   // arrivée de téléportation : ~1290 m/s
		bp(535, 51, 22.1, 0), // la trajectoire continue au nouvel endroit
	}
	evt := []TranslocatorTeleport{{Slot: 535, TimestampUS: 30_000}} // 4 ms avant l'arrivée

	t.Run("dans la fenetre : accepte, et l ancre suit", func(t *testing.T) {
		got := DropTeleportsExcept(in, DefaultMaxSpeedMPS, TeleportExemptionsOf(evt))
		if len(got) != 4 {
			t.Fatalf("attendu 4 positions (l'arrivée est réelle), obtenu %d (%+v)", len(got), got)
		}
	})
	// Sans exemption applicable, le filtre actuel rejette l'arrivée PUIS le point suivant
	// (la vitesse se mesure depuis la dernière position ACCEPTÉE) : 2 positions restent.
	t.Run("autre slot : jamais exempte", func(t *testing.T) {
		autre := []TranslocatorTeleport{{Slot: 600, TimestampUS: 30_000}}
		got := DropTeleportsExcept(in, DefaultMaxSpeedMPS, TeleportExemptionsOf(autre))
		if len(got) != len(DropTeleports(in, DefaultMaxSpeedMPS)) {
			t.Fatalf("un événement d'un AUTRE slot a levé le filtre : %d positions (%+v)", len(got), got)
		}
	})
	t.Run("hors fenetre : jamais exempte", func(t *testing.T) {
		loin := []TranslocatorTeleport{{Slot: 535, TimestampUS: 300_000}} // à +266 ms du saut
		got := DropTeleportsExcept(in, DefaultMaxSpeedMPS, TeleportExemptionsOf(loin))
		if len(got) != len(DropTeleports(in, DefaultMaxSpeedMPS)) {
			t.Fatalf("un événement hors ±200 ms a levé le filtre : %d positions (%+v)", len(got), got)
		}
	})
	t.Run("bord de fenetre : ±200 ms inclus, pas au-dela", func(t *testing.T) {
		x := TeleportExemptionsOf([]TranslocatorTeleport{{Slot: 5, TimestampUS: 1_000_000}})
		if !x.covers(5, 1_000_000-translocExemptToleranceUS) ||
			!x.covers(5, 1_000_000+translocExemptToleranceUS) {
			t.Error("la borne ±200 ms doit être couverte (51/51 rejets mesurés y tombent)")
		}
		if x.covers(5, 1_000_000-translocExemptToleranceUS-1) ||
			x.covers(5, 1_000_000+translocExemptToleranceUS+1) {
			t.Error("au-delà de ±200 ms, le filtre reste entier")
		}
	})
	// DEUX ÉVÉNEMENTS ÉLOIGNÉS DU MÊME SLOT (revue ronde 1, F5) : les fenêtres sont deux
	// ÎLOTS, jamais un intervalle qui les joint — une recherche qui comparerait toujours au
	// PREMIER événement du slot couvrirait le milieu, et ces assertions la tuent.
	t.Run("deux evenements du meme slot : rien entre les ilots", func(t *testing.T) {
		e1, e2 := uint64(1_000_000), uint64(10_000_000)
		x := TeleportExemptionsOf([]TranslocatorTeleport{
			{Slot: 5, TimestampUS: e2}, // désordre volontaire : la construction re-trie
			{Slot: 5, TimestampUS: e1},
		})
		for _, ts := range []uint64{e1, e1 + translocExemptToleranceUS, e2 - translocExemptToleranceUS, e2} {
			if !x.covers(5, ts) {
				t.Errorf("instant %d non couvert alors qu'il tombe dans un îlot", ts)
			}
		}
		for _, ts := range []uint64{
			e1 + translocExemptToleranceUS + 1, // juste après le premier îlot
			(e1 + e2) / 2,                      // au milieu des deux
			e2 - translocExemptToleranceUS - 1, // juste avant le second
			e2 + translocExemptToleranceUS + 1, // après le second
		} {
			if x.covers(5, ts) {
				t.Errorf("instant %d couvert ENTRE ou APRÈS les îlots : l'exemption a débordé", ts)
			}
		}
	})
	t.Run("aberration entre deux evenements du meme slot : filtree", func(t *testing.T) {
		serie := []BipedPosition{
			bp(535, 2_483, 0.2, 0),
			bp(535, 2_500, 60, 0), // aberration à ~3500 m/s, LOIN des deux événements
			bp(535, 2_517, 0.3, 0),
		}
		x := TeleportExemptionsOf([]TranslocatorTeleport{
			{Slot: 535, TimestampUS: 30_000}, {Slot: 535, TimestampUS: 5_000_000},
		})
		got := DropTeleportsExcept(serie, DefaultMaxSpeedMPS, x)
		if len(got) != 2 {
			t.Fatalf("%d positions conservées, attendu 2 : l'aberration entre deux fenêtres du "+
				"même slot doit rester filtrée (%+v)", len(got), got)
		}
	})
}

// dropTeleportsReference est LA SÉMANTIQUE D'AVANT L'EXEMPTION D2, copiée VERBATIM depuis
// le DropTeleports du schéma 37 (état d'offline_filters.go avant le lot P1) et FIGÉE ici :
// c'est l'ORACLE de l'invariance. Comparer DropTeleportsExcept(nil) à DropTeleports ne
// prouverait RIEN — depuis le lot P1, le second DÉLÈGUE au premier (constat de revue
// ronde 1, F6 : la fonction se comparait à elle-même). Ne jamais « moderniser » cette
// copie : sa valeur est précisément de ne pas suivre la production.
func dropTeleportsReference(pos []BipedPosition, maxSpeed float64) []BipedPosition {
	if maxSpeed <= 0 || len(pos) == 0 {
		return pos
	}
	type anchor struct {
		p      BipedPosition
		ok     bool
		streak int
	}
	anchors := map[uint32]*anchor{}
	out := pos[:0:0]
	for _, p := range pos {
		a := anchors[p.Slot]
		if a == nil {
			a = &anchor{}
			anchors[p.Slot] = a
		}
		if a.ok && a.streak < maxRejectStreak && speedFrom(a.p, p) > maxSpeed {
			a.streak++
			continue
		}
		a.p, a.ok, a.streak = p, true, 0
		out = append(out, p)
	}
	return out
}

// TestDropTeleportsInvarianceSansEvenement : SANS événement 117, le filtre exempté rend
// BIT À BIT ce que rendait le filtre du schéma 37 — comparé à l'implémentation de RÉFÉRENCE
// figée ci-dessus, jamais à la production (qui délègue). Le même oracle sert au test gaté
// sur film témoin (TestP1InvarianceSansTete117, 188 979 échantillons réels).
func TestDropTeleportsInvarianceSansEvenement(t *testing.T) {
	var in []BipedPosition
	for i := 0; i < 200; i++ {
		in = append(in, bp(513, i*16, float32(i)*0.1, 0))
		if i%17 == 0 {
			in = append(in, bp(513, i*16+8, float32(100+i), 0)) // aberrations à écarter
		}
	}
	avant := dropTeleportsReference(in, DefaultMaxSpeedMPS)
	if len(avant) == len(in) {
		t.Fatal("la référence n'a rien filtré : le jeu d'essai ne prouve rien")
	}
	for nom, x := range map[string]TeleportExemptions{
		"nil": nil, "vide": {}, "construit de rien": TeleportExemptionsOf(nil),
	} {
		apres := DropTeleportsExcept(in, DefaultMaxSpeedMPS, x)
		if len(avant) != len(apres) {
			t.Fatalf("exemption %s : %d contre %d positions — l'invariance contre la référence "+
				"figée est rompue", nom, len(apres), len(avant))
		}
		for i := range avant {
			if avant[i] != apres[i] {
				t.Fatalf("exemption %s : divergence à l'échantillon %d", nom, i)
			}
		}
	}
	// L'enveloppe DropTeleports reste la sémantique de référence, elle aussi.
	prod := DropTeleports(in, DefaultMaxSpeedMPS)
	if len(prod) != len(avant) {
		t.Fatalf("DropTeleports rend %d positions, la référence figée %d", len(prod), len(avant))
	}
	for i := range avant {
		if avant[i] != prod[i] {
			t.Fatalf("DropTeleports diverge de la référence figée à l'échantillon %d", i)
		}
	}
}
