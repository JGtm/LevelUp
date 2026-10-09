//go:build research && campagne_overlay

package grammar

// r_comp_l3_research_test.go — CHANTIER « comp », R-L3 (plan de la campagne §6.2, 2026-10-02) :
// le bassin `i15 managed-engine-timers-component` « tout a un » sur HI_1_4_1, version-31 et
// version-33, et l effet par film d un portage du moteur `ti=0/1/2`.
//
// CE FICHIER EXIGE LA SURCOUCHE DE RECHERCHE (tag `campagne_overlay`, copie de `capture.go` de
// `mesures_bis2_overlay/`, crochet [bis2Intercepteur]) : les composants non portes sont lus par le
// crochet, jamais par un fichier de production.
//
// GRAMMAIRES (lues chez l ecrivain, T7 §2.6 et §2.7, note 3.7 §2 bis ; reprises de
// `reapparition_37_bassin_film_*`) :
//
//	i11 game-engine-soft-ceilings-component          FUN_14116d1ac : R(128)
//	i13 game-engine-disabled-kill-volume-flags       FUN_142f03498 : R(13) = n, n x R(1)
//	i14 GameEngineComposerLetterboxComponent (niv 2) FUN_142f0328c : R(1) R(16) 4x[!R(1) -> R(7)] 4x[R(1) -> R(16)]
//	i15 managed-engine-timers-component              FUN_1407ee7b8 + FUN_1407ee87c :
//	    R(64) ; par fente : R(2) ; 1 -> R(16)R(16)R(5)R(16) ; 2,3 -> R(16)R(16)R(5)
//	i16 scenario-intro-component                     FUN_1410d9004 : R(7) + R(1)
//	i17 matchflow-isplaying-flags-component          FUN_141101038 : R(8)
//
// VARIANTES : `i15` (le bassin seul), `moteur` (les six), et deux TEMOINS decales d un bit
// (`moteur, i15 +1` lit un bit de plus apres le bassin ; `moteur, i15 -1` en rend un).
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 120m \
//	  -run '^TestRCompL3' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"math/bits"
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// rl3Variante decrit un portage mesure.
type rl3Variante struct {
	nom      string
	bassin   bool // i15
	moteur   bool // i11, i13, i14, i16, i17
	decalage int  // temoin : bits ajoutes (+) ou rendus (-) apres le bassin
	// Hypotheses de format des vieux builds (mesure, D6) : `ancien` nomme UNE lecture alternative
	// d un composant de tete du moteur (cf. rl3Ancien).
	ancien string
}

// rl3Ancien lit un composant de tete du moteur sous une hypothese de format ; rend faux si
// l hypothese ne vise pas ce composant.
func rl3Ancien(br *Lecteur, h, name string) bool {
	switch {
	case h == "i2 R(2)" && name == compGameEngineCurrentState:
		br.ReadBits(2)
	case h == "i3 absent" && name == "game-engine-game-finished-component":
	case h == "i4 R(5) sans porte" && name == compGameEngineCurrentRound:
		br.ReadBits(5)
	case h == "i4 porte + R(4)" && name == compGameEngineCurrentRound:
		if !br.ReadBit() {
			br.ReadBits(4)
		}
	case h == "i4 -1 bit" && name == compGameEngineCurrentRound:
		consumeGameEngineCurrentRound(br)
		br.SetBitPos(br.BitPos() - 1)
	case h == "i0 -1 bit" && name == compGameEngineTeamMapping:
		br.SetBitPos(br.BitPos() - 1)
		consumeGameEngineTeamMapping(br)
	default:
		return false
	}
	return true
}

// rl3Hypotheses : les hypotheses de format mesurees sur les images-cles (et sur les deltas).
var rl3Hypotheses = []string{"i0 -1 bit", "i2 R(2)", "i3 absent", "i4 R(5) sans porte", "i4 porte + R(4)", "i4 -1 bit"}

var rl3Variantes = []rl3Variante{
	{nom: "reference"},
	{nom: "i15", bassin: true},
	{nom: "moteur", bassin: true, moteur: true},
	{nom: "moteur, i15 +1 (temoin)", bassin: true, moteur: true, decalage: 1},
	{nom: "moteur, i15 -1 (temoin)", bassin: true, moteur: true, decalage: -1},
}

// rl3Masques : les masques de bassin lus depuis le dernier releve (classe -> compte).
var rl3Masques []string

// rl3Classe range un masque de 64 fentes.
func rl3Classe(m uint64) string {
	n := bits.OnesCount64(m)
	switch {
	case m == 0:
		return "vide"
	case m&(m+1) == 0:
		return "prefixe contigu"
	case n >= 48:
		return "tout a un (>= 48 fentes)"
	}
	return "autre"
}

// rl3Crochet rend le crochet d une variante.
func rl3Crochet(v rl3Variante) func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
	return func(br *Lecteur, name string, _, level uint32) (bool, bool) {
		if v.ancien != "" && rl3Ancien(br, v.ancien, name) {
			return true, true
		}
		switch {
		case v.bassin && name == "managed-engine-timers-component":
			m, _ := reap37LireBassin(br)
			rl3Masques = append(rl3Masques, rl3Classe(m))
			if v.decalage != 0 {
				br.SetBitPos(br.BitPos() + v.decalage)
			}
			return true, true
		case !v.moteur:
			return false, false
		case name == "game-engine-soft-ceilings-component":
			br.Skip(reap37SoftCeilingsBits)
		case name == "game-engine-disabled-kill-volume-flags-component":
			reap37LireVolumes(br)
		case name == "GameEngineComposerLetterboxComponent":
			if level < 2 {
				return true, false // branche basse jamais declaree (T7 §3) : non portee
			}
			reap37LireLetterbox(br)
		case name == "scenario-intro-component":
			br.ReadBits(7)
			br.ReadBits(1)
		case name == "matchflow-isplaying-flags-component":
			br.ReadBits(8)
		default:
			return false, false
		}
		return true, true
	}
}

// rl3Drainer vide les masques releves dans une table.
func rl3Drainer(t cmTables, table, etat string) {
	for _, c := range rl3Masques {
		t.add(table, cmJoindre(etat, c), cmCompte{n: 1})
	}
	rl3Masques = rl3Masques[:0]
}

// rl3Paquets releve, par paquet, l etat des masques de bassin lus dans sa vue B.
type rl3Paquets struct{ t cmTables }

func (e *rl3Paquets) debutDeChunk(int, []byte, []FilmPacket, *World) {
	rl3Drainer(e.t, "bassin_delta", "liaison d image-cle")
}
func (e *rl3Paquets) finDeFilm() { rl3Drainer(e.t, "bassin_delta", "fin") }
func (e *rl3Paquets) paquet(_ int, p *cmPaquet, _ *World) {
	etat := "paquet non ferme"
	if p.d.Fermee {
		etat = "paquet ferme"
	}
	rl3Drainer(e.t, "bassin_delta", etat)
}

// rl3Sains somme, dans la table du juge, les paquets fermes sains et leurs records utiles.
func rl3Sains(j *cmJuge) (paquets, utiles, perdusSains int) {
	for cle, x := range j.t["juge"] {
		if strings.HasPrefix(cle, "perdu · ref sain") {
			perdusSains += x.n
		}
		if !strings.Contains(cle, "aucun invariant contredit") {
			continue
		}
		paquets += x.n
		utiles += x.enJeu
	}
	return paquets, utiles, perdusSains
}

// TestRCompL3Delta : la marche des paquets delta (carte v2 + juge) sous chaque variante.
func TestRCompL3Delta(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tvariante\tpaquets\tfermes\tfermes_sains\tutiles_fermes\tutiles_fermes_sains\t" +
		"utiles_lus\thors_cadre\tgagnes\tperdus\tgagnes_contredits\tperdus_sains\tbloquant_i15\tbloquants_moteur"}
	tabs := []string{"film\tbuild\tvariante\ttable\tcle\tn"}
	defer func() { bis2Intercepteur = nil; rl3Masques = nil }()
	for _, id := range films {
		garde := filmproc.Arm("campagne/r_comp-l3", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		b := cmLireBlocs(f)
		var jref *cmJuge
		var ref map[[2]int]bool
		for _, v := range rl3VariantesDelta() {
			bis2Intercepteur, rl3Masques = nil, rl3Masques[:0]
			if v.bassin || v.moteur || v.ancien != "" {
				bis2Intercepteur = rl3Crochet(v)
			}
			var j *cmJuge
			if jref == nil {
				j = cmNouveauJuge(f, b, nil)
			} else {
				j = cmNouveauJuge(f, b, ref)
				j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			}
			e := &rl3Paquets{t: cmTables{}}
			st := &cmComparateur{ref: map[[2]int]bool{}}
			r, _, _ := cmMarcher(f, cmVariante{}, cmMux{e, j, b3Statut{st}})
			bis2Intercepteur = nil
			if jref == nil {
				jref, ref = j, st.ref
			}
			ps, us, perdusSains := rl3Sains(j)
			i15, moteur := rl3Bloquants(r)
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build,
				v.nom, r.Paquets, r.PaquetsFermes, ps, r.Utiles.RecordsFermes, us, r.Utiles.Records,
				r.Bloquants[CauseHorsCadre].Paquets, j.gagnes, j.perdus, j.gagnesContredits, perdusSains, i15, moteur))
			for _, tm := range []cmTables{e.t, j.t} {
				for nom, m := range tm {
					for _, cle := range cmCles(m) {
						tabs = append(tabs, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d", id, f.build, v.nom, nom, cle, m[cle].n))
					}
				}
			}
		}
		t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "r_comp_l3_delta.tsv", lignes)
	b2Ecrire(t, sortie, "r_comp_l3_delta_tables.tsv", tabs)
}

// rl3Bloquants rend les paquets arretes par `i15` et par l un des six composants du moteur.
func rl3Bloquants(r FrameClosureReport) (i15, moteur int) {
	for cause, x := range r.Bloquants {
		nom := cause
		if !strings.HasPrefix(nom, "ti=0 ") && !strings.HasPrefix(nom, "ti=1 ") && !strings.HasPrefix(nom, "ti=2 ") {
			continue
		}
		for _, c := range []string{"i11 ", "i13 ", "i14 ", "i15 ", "i16 ", "i17 "} {
			if strings.Contains(nom, " "+c) {
				moteur += x.Paquets
				if c == "i15 " {
					i15 += x.Paquets
				}
			}
		}
	}
	return i15, moteur
}

// rl3VariantesDelta : les variantes de la marche delta, plus les hypotheses de format de `i4`
// (mesurees sur les images-cles des trois vieux builds) jouees sur TOUS les films.
func rl3VariantesDelta() []rl3Variante {
	return append(append([]rl3Variante(nil), rl3Variantes...),
		rl3Variante{nom: "i4 -1 bit seul", ancien: "i4 -1 bit"},
		rl3Variante{nom: "moteur + i4 R(5) sans porte", bassin: true, moteur: true, ancien: "i4 R(5) sans porte"},
		rl3Variante{nom: "moteur + i4 porte + R(4)", bassin: true, moteur: true, ancien: "i4 porte + R(4)"})
}
