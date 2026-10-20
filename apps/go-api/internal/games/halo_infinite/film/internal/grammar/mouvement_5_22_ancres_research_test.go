//go:build research

package grammar

// mouvement_5_22_ancres_research_test.go — LES ANCRES DU TEMOIN Madina97294 (lot 5.22).
//
// Le lot 5.11 a mesure le saut sur `dad793c7`, un film ou le bipede etait MUET avant le saut :
// sa fenetre ne separait pas « le saut commence » de « la replication commence » (D6 du lot
// 5.13). `bfecd02b` porte un temoin que l utilisateur a VERIFIE dans Theater le 2026-09-22 :
// Madina97294 (xuid 2533274858283686, slot 523) court en continu entre 1:19 et 1:41 de barre et
// y saute SEPT fois.
//
// Cet instrument pose les ancres de ce temoin SUR L HORLOGE DU FILM (`TimestampUS` des paquets),
// et JAMAIS sur la grille de frames du document — le lot 5.22 a mesure que cette grille est en
// retard de 1 a 2,5 s par endroits (§4). Le temps de barre affiche a cote est INDICATIF :
// `originMs + (ts - premierDelta)`, la conversion nominale, celle-la meme dont l ecart se mesure.
//
// Variables : `MOUV511_FILM`, `MOUV511_CARTE`, `MOUV511_BORNES` (les memes que le 5.11),
// `MOUV522_SLOT` (defaut 523), `MOUV522_ORIGINE_MS` (defaut 12547, l `originMs` de `bfecd02b`).

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"
)

// m522SlotTemoin est le slot du temoin par defaut : Madina97294 sur `bfecd02b`.
const m522SlotTemoin = 523

// m522OrigineMSDefaut est le decalage entre l horloge relative du film et la barre Theater.
//
// ZERO, ET C EST UNE MESURE, PAS UNE COMMODITE (lot 5.22.1) : sur `bfecd02b`, les six sauts
// derives que l utilisateur a dates dans Theater tombent aux instants relatifs 84,050 · 85,234 ·
// 86,602 · 89,322 · 93,126 · 96,962 s, c est-a-dire exactement aux temps de barre 1:24,0 · 1:25,1
// · 1:26,5 · 1:29,2 · 1:33,0 · 1:36,9 qu il cite ; les trois sprints lus (1:28,0 · 1:30,3 ·
// 1:35,7) tombent a 88,121 · 90,356 · 95,844 s et la queue de `mobility` (1:37,0-1:37,6) a
// 97,096-97,63 s. LA BARRE EST L HORLOGE RELATIVE DU FILM, au pas de frame pres ; `originMs`
// (12 547 ms sur ce film) est le recalage des evenements de MATCH, pas l origine de la barre.
const m522OrigineMSDefaut = 0

// m522Slot rend le slot du temoin.
func m522Slot() uint32 {
	if v := os.Getenv("MOUV522_SLOT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return uint32(n) //nolint:gosec // borne par la garde
		}
	}
	return m522SlotTemoin
}

// m522OrigineMS rend l origine de la barre Theater, en millisecondes.
func m522OrigineMS() float64 {
	if v := os.Getenv("MOUV522_ORIGINE_MS"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return m522OrigineMSDefaut
}

// m522Barre rend le temps de barre NOMINAL d un instant relatif, en `m:ss.d`.
func m522Barre(relS float64) string {
	ms := m522OrigineMS() + relS*1000
	s := ms / 1000
	return fmt.Sprintf("%d:%04.1f", int(s)/60, s-float64(int(s)/60*60))
}

// m522Echantillon est une observation datee du temoin : vitesse, ou transition d etat.
type m522Echantillon struct {
	ts  uint64
	rel float64
	vz  float64
	vxy float64
	// etat est non vide pour une transition lue (`crouch`, `slide`, `mobility`, `sprint`).
	etat string
	on   bool
}

// m522Marche deroule le film ENTIER par la marche de production (`DecodeFrameViews`, trois vues,
// paquets a evenements localises par `marchLocateStrict`) et rend, pour le slot demande, toutes
// les lectures de vitesse et toutes les transitions d etat, datees sur l horloge du film.
//
// Elle rend aussi l instant du premier paquet delta — l origine de l horloge relative.
func m522Marche(t *testing.T, slot uint32) ([]m522Echantillon, uint64) {
	t.Helper()
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	var out []m522Echantillon
	var origine, ts uint64
	obs := NouvelleObservation()
	obs.EtatMouvementHook = func(c EtatMouvementComposant, s uint32, v []uint64) {
		if s != slot {
			return
		}
		rel := float64(ts-origine) / 1e6
		if c == EtatVitesse {
			if len(v) < 4 || v[0] != 0 || v[1] != 0 {
				return
			}
			vec := DecodeVelocity(v[2], v[3])
			out = append(out, m522Echantillon{ts: ts, rel: rel, vz: float64(vec[2]),
				vxy: math.Hypot(float64(vec[0]), float64(vec[1]))})
			return
		}
		if len(v) < 1 {
			return
		}
		nom, on := c.String(), v[0] != 0
		if c == EtatCapaciteActive {
			nom, on = "sprint", v[0] == sprintAbilitySlotRaw
		}
		out = append(out, m522Echantillon{ts: ts, rel: rel, etat: nom, on: on,
			vz: float64(v[0])})
	}
	cfg.Obs = obs
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if origine == 0 {
				origine = pk.TimestampUS
			}
			ts = pk.TimestampUS
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			DecodeFrameViews(pay, w, cfg, MovementStateViews, debut)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out, origine
}

// TestMouvement522Ancres publie les ancres du temoin : TOUS les episodes de montee (retenus par
// la hauteur du Spartan OU NON — le saut qui manque au derive est celui qui ne l est pas), et
// toutes les transitions d etat lues, sur l horloge du film.
func TestMouvement522Ancres(t *testing.T) {
	slot := m522Slot()
	ech, origine := m522Marche(t, slot)
	if len(ech) == 0 {
		t.Fatalf("slot %d : aucune observation", slot)
	}
	t.Logf("SLOT %d — %d observations · premier paquet delta a %d us · horloge relative "+
		"de %.3f s a %.3f s (barre %s a %s)", slot, len(ech), origine, ech[0].rel,
		ech[len(ech)-1].rel, m522Barre(ech[0].rel), m522Barre(ech[len(ech)-1].rel))

	var vs []jumpVelSample
	for _, e := range ech {
		if e.etat == "" {
			vs = append(vs, jumpVelSample{ts: e.ts, vz: e.vz})
		}
	}
	t.Logf("VITESSES : %d lectures", len(vs))
	eps := episodesDeMontee(slot, vs)
	t.Logf("EPISODES DE MONTEE : %d fermes", len(eps))
	for _, e := range eps {
		r0 := float64(e.t0-origine) / 1e6
		r1 := float64(e.t1-origine) / 1e6
		verdict := "REJETE"
		if hauteurDeSaut(e.haut) {
			verdict = "RETENU"
		}
		t.Logf("  %s  t0 %8.3f s (barre %s) -> t1 %8.3f s (barre %s) · duree %5.3f s · "+
			"hauteur %6.4f m", verdict, r0, m522Barre(r0), r1, m522Barre(r1), r1-r0, e.haut)
	}
	t.Logf("TRANSITIONS D ETAT LUES :")
	for _, e := range ech {
		if e.etat == "" {
			continue
		}
		t.Logf("  t %8.3f s (barre %s) · %-10s on=%v (brut %.0f)", e.rel, m522Barre(e.rel),
			e.etat, e.on, e.vz)
	}
}
