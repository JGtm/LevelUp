//go:build research && campagne_overlay

package grammar

// campagne_bis2_vehicules_research_test.go — MESURES BIS 2 DE LA CAMPAGNE DE GRAMMAIRE
// (2026-10-01) : T6-C2 (images-cles `ti=40` sans masque, porte `+0x818` par chassis) et point 26
// (le temoin utilisateur `81c02726` et la grammaire `ti=43` de la note T7).
//
// CE FICHIER EXIGE LA SURCOUCHE DE RECHERCHE (tag `campagne_overlay`, cf.
// `campagne_bis2_positions_research_test.go`) : les composants non portes sont lus par le crochet
// d interception de la copie de recherche de `capture.go` ([bis2Intercepteur]), jamais par un
// fichier de production. Les grammaires viennent des notes T6 §4 et T7 §5 (lecteur ET ecrivain
// lus dans Ghidra par ces notes).
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 120m \
//	  -run '^TestCampagneBis2ImagesClesTi40$|^TestCampagneBis2Dispositifs$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// --- T6-C2 : images-cles ti=40 ------------------------------------------------------------------

// b2vPhysique lit CAMPAGNE_PHYSIQUE : `chassis:type,...` (type 13 = aucun bloc).
func b2vPhysique() map[uint32]int {
	out := map[uint32]int{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_PHYSIQUE"), ",") {
		g, ty, ok := strings.Cut(strings.TrimSpace(x), ":")
		if !ok {
			continue
		}
		gid, err1 := strconv.ParseUint(g, 16, 32)
		v, err2 := strconv.Atoi(ty)
		if err1 == nil && err2 == nil {
			out[uint32(gid)] = v
		}
	}
	return out
}

// b2vIssue : ce qu une marche d image-cle a rendu.
type b2vIssue struct {
	fermee bool
	arret  string
}

// b2vMarcherImageCle marche un record d image-cle et rend son issue et le MPPWord32 lu.
func b2vMarcherImageCle(pay []byte, b keyframeBorne, reg *Registry, ctx ContexteDeLecture) (b2vIssue, uint32, bool) {
	var mpp uint32
	vu := false
	obs := &Observation{MppHook: func(f MPPField, v uint64, present bool) {
		if f == MPPWord32 && present && !vu {
			mpp, vu = uint32(v), true
		}
	}}
	ctx.Obs = obs
	tr := WalkKeyframeFullState(pay, b.Bit, reg, ctx)
	is := b2vIssue{fermee: tr.DesyncAt < 0 && tr.EndBit == b.Want}
	switch {
	case tr.DesyncAt >= 0:
		is.arret = nomComposantBloquant(reg, b.TI, tr.DesyncAt)
	case tr.EndBit < b.Want:
		is.arret = "sous"
	case tr.EndBit > b.Want:
		is.arret = "sur"
	}
	return is, mpp, vu
}

// TestCampagneBis2ImagesClesTi40 mesure la fermeture des records d image-cle `ti=40` : reference,
// composants portes porte POSEE, porte LEVEE, porte par CHASSIS (table CAMPAGNE_PHYSIQUE).
func TestCampagneBis2ImagesClesTi40(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	physique := b2vPhysique()
	lignes := []string{"film\tbuild\trecords_ti40\tref_fermes\tposee_fermes\tlevee_fermes\tchassis_connus\t" +
		"chassis_fermes_inconnu_posee\tchassis_fermes_inconnu_levee\toracle_l_une_ou_l_autre\tposee_et_levee"}
	parChassis := []string{"film\tbuild\tchassis\ttype_physique\trecords\tferme_posee_seule\tferme_levee_seule\t" +
		"ferme_les_deux\tferme_aucune\tarret_posee\tarret_levee"}
	arrets := []string{"film\tbuild\tvariante\tarret\trecords"}
	defer func() { bis2Intercepteur = nil }()
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis2-ti40", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		restore, errMPP := InstallFilmFormatMPP(f.fc)
		ctx := f.fc.ContexteDeLecture()
		marche := f.fc.MarcheDImageCle()
		type cumul struct {
			n, posee, levee, lesDeux, aucune int
			arretP, arretL                   map[string]int
		}
		parCh := map[uint32]*cumul{}
		var n, ref, posee, levee, cc, cp, cl, orac, deux int
		parArret := map[string]map[string]int{"reference": {}, "posee": {}, "levee": {}}
		for _, num := range f.fc.ChunkNumbers() {
			data, pks, ok := f.fc.ChunkAt(num)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				pay := pk.Payload(data)
				for _, b := range seulesBornees(keyframeBornesDe(marche.Records(pay))) {
					if b.TI != 40 {
						continue
					}
					n++
					bis2Intercepteur = nil
					r, mpp, vu := b2vMarcherImageCle(pay, b, f.reg, ctx)
					bis2Intercepteur = b2vCrochet(true, false)
					b2vPorte = true
					p, _, _ := b2vMarcherImageCle(pay, b, f.reg, ctx)
					b2vPorte = false
					l, _, _ := b2vMarcherImageCle(pay, b, f.reg, ctx)
					bis2Intercepteur = nil
					parArret["reference"][r.arret]++
					parArret["posee"][p.arret]++
					parArret["levee"][l.arret]++
					if r.fermee {
						ref++
					}
					if p.fermee {
						posee++
					}
					if l.fermee {
						levee++
					}
					if p.fermee || l.fermee {
						orac++
					}
					if p.fermee && l.fermee {
						deux++
					}
					cle := uint32(0xffffffff)
					if vu {
						cle = mpp
					}
					ty, connu := physique[cle]
					switch {
					case connu:
						cc++
						if (ty == 6 && p.fermee) || (ty != 6 && l.fermee) {
							cp++
							cl++
						}
					default:
						if p.fermee {
							cp++
						}
						if l.fermee {
							cl++
						}
					}
					x := parCh[cle]
					if x == nil {
						x = &cumul{arretP: map[string]int{}, arretL: map[string]int{}}
						parCh[cle] = x
					}
					x.n++
					x.arretP[p.arret]++
					x.arretL[l.arret]++
					switch {
					case p.fermee && l.fermee:
						x.lesDeux++
					case p.fermee:
						x.posee++
					case l.fermee:
						x.levee++
					default:
						x.aucune++
					}
				}
			}
		}
		if errMPP == nil {
			restore()
		}
		lignes = append(lignes, fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, n, ref,
			posee, levee, cc, cp, cl, orac, deux))
		for cle, x := range parCh {
			ty, connu := physique[cle]
			tys := "inconnu"
			if connu {
				tys = strconv.Itoa(ty)
			}
			parChassis = append(parChassis, fmt.Sprintf("%s\t%s\t%08x\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%s", id, f.build,
				cle, tys, x.n, x.posee, x.levee, x.lesDeux, x.aucune, b2vTop(x.arretP), b2vTop(x.arretL)))
		}
		for v, m := range parArret {
			for a, k := range m {
				if a == "" {
					a = "(fermee)"
				}
				arrets = append(arrets, fmt.Sprintf("%s\t%s\t%s\t%s\t%d", id, f.build, v, a, k))
			}
		}
		t.Logf("%s %s : %d records ti=40 bornes ; fermes ref %d, posee %d, levee %d ; pic %d Mio, %s", id,
			f.build, n, ref, posee, levee, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "mb2_ti40_images_cles.tsv", lignes)
	b2Ecrire(t, sortie, "mb2_ti40_par_chassis.tsv", parChassis)
	b2Ecrire(t, sortie, "mb2_ti40_arrets.tsv", arrets)
}

// b2vTop rend les trois arrets les plus frequents.
func b2vTop(m map[string]int) string {
	type kv struct {
		k string
		v int
	}
	var xs []kv
	for k, v := range m {
		if k == "" {
			k = "(fermee)"
		}
		xs = append(xs, kv{k, v})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].v > xs[j].v || (xs[i].v == xs[j].v && xs[i].k < xs[j].k) })
	var parts []string
	for i, x := range xs {
		if i == 3 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s=%d", x.k, x.v))
	}
	return strings.Join(parts, ";")
}

// --- point 26 : la grammaire ti=43 (T7) et le temoin 81c02726 -------------------------------------

// b2vFenetre : l ecouteur qui decrit les paquets d une fenetre de temps.
type b2vFenetre struct {
	debutUS, finUS uint64
	lignes         []string
	variante, film string
	statut         map[[2]int]bool
	causes         map[string]int
	pertes         *b2vPertes
}

// b2vPertes : les pertes d une marche par rapport a la reference (point 26 : ou sont les paquets
// perdus du temoin, et a quel composant ils s arretent).
type b2vPertes struct {
	ref            map[[2]int]bool
	gagnes, perdus int
	utilesGagnes   int
	causesGagnees  map[string]int
}

func (e *b2vFenetre) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *b2vFenetre) finDeFilm()                                     {}

func (e *b2vFenetre) paquet(c int, p *cmPaquet, _ *World) {
	cle := [2]int{p.d.Chunk, p.d.Index}
	e.statut[cle] = p.d.Fermee
	if !p.d.Fermee {
		e.causes[p.d.Cause]++
	}
	if e.pertes != nil && e.pertes.ref != nil {
		avant := e.pertes.ref[cle]
		switch {
		case p.d.Fermee && !avant:
			e.pertes.gagnes++
			e.pertes.utilesGagnes += p.utilesFermes
		case !p.d.Fermee && avant:
			e.pertes.perdus++
		}
	}
	if e.finUS == 0 || p.d.TimestampUS < e.debutUS || p.d.TimestampUS > e.finUS {
		return
	}
	var ti43, montees []string
	for _, r := range p.recs {
		if r.TypeIndex == 43 && r.Type != 2 {
			etat := "lu"
			if r.Trace.DesyncAt >= 0 {
				etat = "desync@" + strconv.Itoa(r.Trace.DesyncAt)
			}
			var dev []string
			for _, cr := range r.Trace.Comps {
				if cr.Index >= 19 {
					dev = append(dev, "i"+strconv.Itoa(cr.Index))
				}
			}
			ti43 = append(ti43, fmt.Sprintf("%s:%d:%s[%s]", recordKind(r.Type), r.Slot, etat, strings.Join(dev, "+")))
		}
		for _, cr := range r.Trace.Comps {
			if ps, ok := cr.ParentOf(); ok && ps.Attached {
				montees = append(montees, fmt.Sprintf("slot%d(ti%d)->%d", r.Slot, r.TypeIndex, ps.Quant16))
			}
		}
	}
	e.lignes = append(e.lignes, fmt.Sprintf("%s\t%s\t%d\t%d\t%.2f\t%t\t%s\t%s\t%08x\t%d\t%s\t%s\t%s", e.film,
		e.variante, c, p.d.Index, float64(p.d.TimestampUS)/1e6, p.d.Fermee, p.d.Cause, p.d.Sortie, p.d.EIDRejete,
		len(p.recs), p.d.DernierLu, strings.Join(ti43, ","), strings.Join(montees, ",")))
}

// recordKind nomme le type de record.
func recordKind(t int) string {
	switch t {
	case 1:
		return "NEW"
	case 2:
		return "DEL"
	case 3:
		return "DELTA"
	}
	return "T" + strconv.Itoa(t)
}

// TestCampagneBis2Dispositifs marche chaque film sans puis avec la grammaire `ti=43` de T7, et
// decrit paquet par paquet la fenetre CAMPAGNE_FENETRE (`film:debut_s-fin_s`) du temoin.
func TestCampagneBis2Dispositifs(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	fenetres := map[string][2]uint64{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_FENETRE"), ";") {
		id, plage, ok := strings.Cut(x, ":")
		if !ok {
			continue
		}
		a, b, _ := strings.Cut(plage, "-")
		da, _ := strconv.ParseFloat(a, 64)
		db, _ := strconv.ParseFloat(b, 64)
		fenetres[id] = [2]uint64{uint64(da * 1e6), uint64(db * 1e6)}
	}
	lignes := []string{"film\tbuild\tvariante\tpaquets\tfermes\tutiles_fermes\tutiles_lus\thors_cadre\t" +
		"paquets_gagnes\tpaquets_perdus\tutiles_gagnes\tcauses_ti43\tcause_ti43_principale\t" +
		"fermes_contredits\tgagnes_contredits"}
	detail := []string{"film\tvariante\tchunk\tpaquet\tt_s\tferme\tcause\tsortie_vue_b\teid_rejete\trecords\t" +
		"dernier_lu\trecords_ti43\tparents_attaches"}
	causes := []string{"film\tbuild\tvariante\tcause\tpaquets"}
	defer func() { bis2Intercepteur = nil }()
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis2-ti43", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		fen := fenetres[id]
		blocs := cmLireBlocs(f)
		var ref map[[2]int]bool
		for _, v := range []string{"reference", "grammaire-ti43-T7"} {
			bis2Intercepteur = nil
			if v != "reference" {
				bis2Intercepteur = b2vCrochet(false, true)
			}
			e := &b2vFenetre{debutUS: fen[0], finUS: fen[1], variante: v, film: id, statut: map[[2]int]bool{},
				causes: map[string]int{}, pertes: &b2vPertes{ref: ref}}
			juge := cmNouveauJuge(f, blocs, ref)
			rep, _, _ := cmMarcher(f, cmVariante{}, cmMux{e, juge})
			bis2Intercepteur = nil
			if ref == nil {
				ref = e.statut
			}
			n43, top, topN := 0, "", 0
			for cause, k := range e.causes {
				causes = append(causes, fmt.Sprintf("%s\t%s\t%s\t%s\t%d", id, f.build, v, cause, k))
				if strings.Contains(cause, "ti=43") {
					n43 += k
					if k > topN {
						top, topN = cause, k
					}
				}
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%d\t%d", id, f.build, v,
				rep.Paquets, rep.PaquetsFermes, rep.Utiles.RecordsFermes, rep.Utiles.Records,
				rep.Bloquants[CauseHorsCadre].Paquets, e.pertes.gagnes, e.pertes.perdus, e.pertes.utilesGagnes, n43, top,
				juge.fermesContre, juge.gagnesContredits))
			detail = append(detail, e.lignes...)
		}
		t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "mb2_ti43.tsv", lignes)
	b2Ecrire(t, sortie, "mb2_ti43_fenetre.tsv", detail)
	b2Ecrire(t, sortie, "mb2_ti43_causes.tsv", causes)
}
