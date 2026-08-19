package replay

// colline_propriete_mesure_test.go — LOT C-ter VOLET 1, CT.1.2 : LA MESURE D'UNE LECTURE, et son
// verdict aux seuils de `colline_propriete_oracle_test.go`.
//
// Pour une lecture (mode, tag, F ou D) : les series chainees par slot, les bascules par slot, le
// slot CANDIDAT (celui qui bascule le plus), l'exclusivite (i), la synchronie (ii) sur tous les
// increments puis hors increment terminal, les temoins (iii) — decale de +20 s, slots permutes
// (les bascules des AUTRES slots du meme tag), niveau du hasard d'une bascule et de N bascules —,
// le DELAI de la premiere bascule (film, rejeu), et la comparaison au second oracle (les periodes
// publiees aujourd'hui). Une ligne TSV par lecture. Rien n'est publie dans le document.

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// ctEtatSlot est un slot d'une lecture : ses bascules, et l'instant a partir duquel il « porte »
// (F : ses transitions d'etat ; D : sa premiere emission, puis il designe jusqu'a la fin).
type ctEtatSlot struct {
	slot     uint32
	bascules []int64
	porte    []int64
}

// ctMesureLecture mesure UNE lecture et publie son verdict.
func ctMesureLecture(t *testing.T, sb *strings.Builder, e ctEntree, incs []ctIncrement, r ctLecture2) {
	t.Helper()
	nom := fmt.Sprintf("mode %s tag %-2d lecture %s (%s)", ctMode(r.modeA), r.tag, ctNomLecture(r.flag), r.role)
	slots := ctEtatsDeLecture(e, r)
	t.Logf("")
	if len(slots) == 0 {
		t.Logf("=== %s : SANS OBJET — aucune lecture chainee valuee sur ce film", nom)
		fmt.Fprintf(sb, "lecture\t%s\t%s\t%d\t%s\t%s\tsans_objet\n", e.short, ctMode(r.modeA), r.tag,
			ctNomLecture(r.flag), r.role)
		return
	}
	cand := slots[0]
	t.Logf("=== %s : %d slot(s), candidat = slot %d (%d bascules)", nom, len(slots), cand.slot,
		len(cand.bascules))
	for _, s := range slots {
		taux, ok, tot, _ := ctSynchronie(incs, s.bascules, 0, false)
		t.Logf("    slot %-5d : %3d bascules · premiere %7d ms · synchrone %d/%d = %.0f %% · %s",
			s.slot, len(s.bascules), ctPremiere(s.bascules), ok, tot, 100*taux, ctExtrait(s.bascules))
	}
	m := ctMesure(e, incs, slots)
	m.log(t, e, incs)
	fmt.Fprintf(sb, "lecture\t%s\t%s\t%d\t%s\t%s\t%d\t%d\t%.4f\t%d\t%d\t%.4f\t%d\t%d\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%d\t%d\t%d\t%d\t%d\t%s\n",
		e.short, ctMode(r.modeA), r.tag, ctNomLecture(r.flag), r.role, cand.slot, len(cand.bascules),
		m.exclusivite, m.okTous, m.totTous, m.tauxTous, m.okChg, m.totChg, m.tauxChg, m.decale,
		m.permute, m.hasard1, m.hasardN, m.delaiFilm, m.delaiRejeu, m.presence.premierePresente,
		m.presence.derniereAbsente, m.premierContact, m.verdict())
}

// ctNomLecture rend F ou D.
func ctNomLecture(flag bool) string {
	if flag {
		return "F"
	}
	return "D"
}

// ctPremiere rend le premier instant d'une liste, -1 si vide.
func ctPremiere(ts []int64) int64 {
	if len(ts) == 0 {
		return -1
	}
	return ts[0]
}

// ctExtrait rend jusqu'a 12 instants, en secondes.
func ctExtrait(ts []int64) string {
	parts := make([]string, 0, 12)
	for i, t := range ts {
		if i == 12 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%.1f", float64(t)/1000))
	}
	return "[" + strings.Join(parts, " ") + "] s"
}

// ctEtatsDeLecture rend les slots d'une lecture, tries par nombre de bascules DECROISSANT (le
// premier est le candidat), puis par slot.
func ctEtatsDeLecture(e ctEntree, r ctLecture2) []ctEtatSlot {
	series := ctSeriesDuTag(e, r.modeA, r.tag)
	out := make([]ctEtatSlot, 0, len(series))
	for slot, ser := range series {
		s := ctEtatSlot{slot: slot, bascules: ctBasculesSlot(ser, r)}
		if len(s.bascules) == 0 {
			continue
		}
		if r.flag {
			s.porte = s.bascules
		} else {
			s.porte = []int64{s.bascules[0]}
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].bascules) != len(out[j].bascules) {
			return len(out[i].bascules) > len(out[j].bascules)
		}
		return out[i].slot < out[j].slot
	})
	return out
}

// ctMesureLect porte les chiffres d'une lecture.
type ctMesureLect struct {
	exclusivite            float64
	framesJugees           int
	tauxTous, tauxChg      float64
	okTous, totTous        int
	okChg, totChg          int
	ecarts                 []int64
	decale, permute        float64
	permuteObjet           bool
	hasard1, hasardN       float64
	delaiFilm, delaiRejeu  int64
	premiereFrame          int
	periodesAPlus1, aPlus5 int
	periodes               int
	// presence : la creation du slot candidat, bornee par les images-cles.
	presence ctPresence
	// premierContact : premiere emission chainee de la jauge (tag 3) ou du proprietaire (tag 4)
	// sur le film — l'instant ou quelqu'un touche la colline pour la premiere fois, borne haute
	// de l'activation de la premiere colline.
	premierContact int64
}

// ctMesure calcule (i), (ii), (iii) et le delai pour un jeu de slots (le premier = candidat).
func ctMesure(e ctEntree, incs []ctIncrement, slots []ctEtatSlot) ctMesureLect {
	var m ctMesureLect
	cand := slots[0]
	t0, t1 := ctBornes(e)
	m.exclusivite, m.framesJugees = ctExclusivite(e, slots)
	m.tauxTous, m.okTous, m.totTous, m.ecarts = ctSynchronie(incs, cand.bascules, 0, false)
	m.tauxChg, m.okChg, m.totChg, _ = ctSynchronie(incs, cand.bascules, 0, true)
	m.decale, _, _, _ = ctSynchronie(incs, cand.bascules, ctDecalageMS, false)
	if len(slots) > 1 {
		m.permuteObjet = true
		sum := 0.0
		for _, s := range slots[1:] {
			r, _, _, _ := ctSynchronie(incs, s.bascules, 0, false)
			sum += r
		}
		m.permute = sum / float64(len(slots)-1)
	}
	m.hasard1 = ctPartFenetres(incs, t0, t1)
	m.hasardN = ctHasardN(incs, len(cand.bascules), t0, t1, false)
	m.delaiFilm = cand.bascules[0]
	m.delaiRejeu = cand.bascules[0] - t0
	m.premiereFrame, _ = p2aFrameOf(e.doc, int(cand.bascules[0]))
	m.periodes, m.periodesAPlus1, m.aPlus5 = ctContrePeriodes(e, cand.bascules)
	m.presence = ctPresenceImagesCles(e, cand.slot)
	m.premierContact = ctPremierContact(e)
	return m
}

// ctPremierContact rend la premiere emission chainee de la jauge ou du proprietaire (-1 si aucune).
func ctPremierContact(e ctEntree) int64 {
	first := int64(-1)
	for _, l := range e.lectures {
		if !l.modeA || !l.chained || !l.has {
			continue
		}
		if l.tag != filmdec.ManagedPropertyTagQuant && l.tag != filmdec.ManagedPropertyTagU32 {
			continue
		}
		if first < 0 || l.tMS < first {
			first = l.tMS
		}
	}
	return first
}

// ctExclusivite rend la part des frames, a partir de la premiere bascule de la lecture, ou
// EXACTEMENT UN slot porte — et le nombre de frames jugees.
func ctExclusivite(e ctEntree, slots []ctEtatSlot) (float64, int) {
	first := int64(-1)
	for _, s := range slots {
		if p := ctPremiere(s.porte); p >= 0 && (first < 0 || p < first) {
			first = p
		}
	}
	f0, ok := p2aFrameOf(e.doc, int(first))
	if !ok {
		f0 = 0
	}
	juges, uns := 0, 0
	for f := f0; f < e.doc.FrameCount; f++ {
		ms := ctMSDeFrame(e, f)
		n := 0
		for _, s := range slots {
			if ctPorteA(s.porte, ms) {
				n++
			}
		}
		juges++
		if n == 1 {
			uns++
		}
	}
	return p2aRate(uns, juges), juges
}

// ctPorteA rejoue un etat a transitions (initial faux) a l'instant ms.
func ctPorteA(trans []int64, ms int64) bool {
	n := sort.Search(len(trans), func(i int) bool { return trans[i] > ms })
	return n%2 == 1
}

// ctContrePeriodes confronte les bascules du candidat aux DEBUTS des periodes actives publiees
// aujourd'hui : combien de debuts de periode ont une bascule a +/- 1 s, a +/- 5 s.
func ctContrePeriodes(e ctEntree, bascules []int64) (int, int, int) {
	n, a1, a5 := 0, 0, 0
	for _, st := range e.doc.ZoneStates {
		for _, sp := range st.Spans {
			if !sp.Active {
				continue
			}
			n++
			ms := ctMSDeFrame(e, sp.T0)
			best := int64(-1)
			for _, b := range bascules {
				if d := ctAbs(b - ms); best < 0 || d < best {
					best = d
				}
			}
			if best >= 0 && best <= 1000 {
				a1++
			}
			if best >= 0 && best <= 5000 {
				a5++
			}
		}
	}
	return n, a1, a5
}

// verdict rend le verdict aux seuils ecrits — la lettre du plan d'abord ((ii) sur TOUS les
// increments), puis la lecture « changements de colline » (hors terminal), entre crochets.
func (m ctMesureLect) verdict() string {
	i := m.exclusivite >= ctSeuilExclusivite
	iiTous := m.tauxTous >= ctSeuilSynchronie
	iiChg := m.tauxChg >= ctSeuilSynchronie
	iii := m.decale < ctFacteurTemoin*m.tauxTous && (!m.permuteObjet || m.permute < ctFacteurTemoin*m.tauxTous)
	iiiChg := m.decale < ctFacteurTemoin*m.tauxChg && (!m.permuteObjet || m.permute < ctFacteurTemoin*m.tauxChg)
	lettre := "NON TENU"
	if i && iiTous && iii {
		lettre = "TENU"
	}
	chg := "NON TENU"
	if i && iiChg && iiiChg {
		chg = "TENU"
	}
	return fmt.Sprintf("%s [hors terminal : %s]", lettre, chg)
}

// log publie la mesure d'une lecture, chiffre par chiffre, avec ses denominateurs.
func (m ctMesureLect) log(t *testing.T, e ctEntree, incs []ctIncrement) {
	t.Helper()
	t.Logf("  (i)   EXCLUSIVITE : exactement un slot porteur sur %.1f %% des %d frames jugees"+
		" (seuil %.0f %%) : %s", 100*m.exclusivite, m.framesJugees, 100*ctSeuilExclusivite,
		ctTenu(m.exclusivite >= ctSeuilExclusivite))
	t.Logf("  (ii)  SYNCHRONIE : increments a +/- %d ms d'une bascule : TOUS %d/%d = %.1f %% ·"+
		" HORS TERMINAL %d/%d = %.1f %% (seuil %.0f %%) : %s / %s · ecarts bascule-increment %v ms",
		ctFenetreMS, m.okTous, m.totTous, 100*m.tauxTous, m.okChg, m.totChg, 100*m.tauxChg,
		100*ctSeuilSynchronie, ctTenu(m.tauxTous >= ctSeuilSynchronie),
		ctTenu(m.tauxChg >= ctSeuilSynchronie), m.ecarts)
	perm := "sans objet (un seul slot)"
	if m.permuteObjet {
		perm = fmt.Sprintf("%.1f %%", 100*m.permute)
	}
	t.Logf("  (iii) TEMOINS : decale +%d s = %.1f %% · slots permutes = %s · HASARD : une bascule"+
		" %.1f %%, N bascules %.1f %% (moitie du reel = %.1f %%)", ctDecalageMS/1000, 100*m.decale,
		perm, 100*m.hasard1, 100*m.hasardN, 100*ctFacteurTemoin*m.tauxTous)
	t.Logf("  DELAI : premiere bascule a %d ms du debut du film, %d ms de l'origine du rejeu"+
		" (frame %d) ; %d increments · CREATION du slot candidat : presente des l'image-cle a"+
		" %d ms, derniere absente %d ms (%d images-cles) · PREMIER CONTACT (jauge/proprietaire)"+
		" a %d ms du film, %d ms de l'origine du rejeu", m.delaiFilm, m.delaiRejeu, m.premiereFrame,
		len(incs), m.presence.premierePresente, m.presence.derniereAbsente, m.presence.imagesCles,
		m.premierContact, m.premierContact-(m.delaiFilm-m.delaiRejeu))
	t.Logf("  SECOND ORACLE : %d periodes actives publiees aujourd'hui, dont %d debutent a +/- 1 s"+
		" d'une bascule et %d a +/- 5 s", m.periodes, m.periodesAPlus1, m.aPlus5)
	t.Logf("  VERDICT : %s", m.verdict())
}

// ctPresence borne la CREATION d'un slot par les images-cles : la premiere image-cle qui le
// porte et la derniere, avant elle, qui ne le porte pas (horloge du film, ms ; -1 = aucune).
//
// POURQUOI. Le delta ne porte que les CHANGEMENTS : l'etat initial du designateur (la premiere
// colline) vit dans l'image-cle, jamais dans le delta. Si l'objet de mode n'apparait aux
// images-cles qu'apres un delai, sa creation borne l'activation de la premiere colline ; s'il est
// la des la premiere image-cle, le film ne date pas cette activation par ce canal.
type ctPresence struct {
	premierePresente, derniereAbsente int64
	imagesCles                        int
}

// ctPresenceImagesCles releve la presence d'un slot a chaque image-cle du film.
func ctPresenceImagesCles(e ctEntree, slot uint32) ctPresence {
	p := ctPresence{premierePresente: -1, derniereAbsente: -1}
	for ch := 1; ch <= filmdec.CountFilmChunks(e.dir); ch++ {
		data, err := filmdec.ReadFilmChunk(e.dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			p.imagesCles++
			ms := (int64(pk.TimestampUS) - int64(e.clockUS)) / 1000
			present := false
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				if uint32(r.Slot) == slot {
					present = true
					break
				}
			}
			switch {
			case present && p.premierePresente < 0:
				p.premierePresente = ms
			case !present && p.premierePresente < 0:
				p.derniereAbsente = ms
			}
		}
	}
	return p
}

// ctTenu rend TENU / NON TENU.
func ctTenu(b bool) string {
	if b {
		return "TENU"
	}
	return "NON TENU"
}
