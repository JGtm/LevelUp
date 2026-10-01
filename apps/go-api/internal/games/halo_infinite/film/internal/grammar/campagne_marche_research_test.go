//go:build research

package grammar

// campagne_marche_research_test.go — LES MESURES CIBLEES DE LA CAMPAGNE DE GRAMMAIRE (phase 1,
// etape 4, 2026-10-01) : la MARCHE. Un instrument de recherche : aucun fichier de production
// n est touche, aucune sortie ne change.
//
// La marche est celle de la carte v2 ([FrameClosureDetaillee]) : meme monde (liaison des
// images-cles chunk par chunk, table anticipee), meme localisation des listes, meme pilotage
// ([marcheDetaillee.marcherParRangs]), meme classement. Elle garde EN PLUS les records de la vue B
// de chaque paquet, ses NEW refuses, et accepte deux VARIANTES qui ne sont que des mesures :
//
//	oracle  : des liaisons posees APRES un paquet donne (la naissance retrouvee par la sonde M1,
//	          posee comme si son NEW avait ete lu dans ce paquet) ;
//	tete    : un localisateur de debut de liste dont les candidats NEW sont filtres par la bande
//	          de production OU par l allocation du bloc de type 1 (A/B du filtre de T1-3).
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestCampagneMesuresCiblees$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// cmFilm est un film ouvert pour la campagne.
type cmFilm struct {
	id, build string
	fc        *FilmContext
	reg       *Registry
	cfg       FrameConfig
	utiles    UsagesProduit
}

// cmPaquet est ce que la marche a rendu d UN paquet delta.
type cmPaquet struct {
	d       PaquetDeCarte
	pk      FilmPacket
	pay     []byte
	recs    []FrameRecord
	enTete  bool
	debut   int // -1 : liste non localisee
	strict  int // localisateur strict (paquets a evenements), -2 hors paquet a evenements
	refuses []neufRefuse
	chaine  []cmEnTete // la chaine de tete acceptee par debutParChaine, s il y en a une
	// utilesFermes : les records utiles que ce paquet a fermes (mesures bis 1).
	utilesFermes int
}

// cmEnTete : un en-tete de record lu (type, slot).
type cmEnTete struct {
	typ  int
	slot uint32
}

// cmLiaison : une liaison d oracle.
type cmLiaison struct{ eid, ti uint32 }

// cmVariante porte les deux variantes de mesure ; la valeur zero est la marche de production.
type cmVariante struct {
	oracle map[[2]int][]cmLiaison
	tete   func(c int) func(pay []byte, w *World, cfg FrameConfig) (int, bool)
}

// cmEcouteur recoit la marche.
type cmEcouteur interface {
	debutDeChunk(c int, data []byte, pks []FilmPacket, w *World)
	paquet(c int, p *cmPaquet, w *World)
	finDeFilm()
}

// cmMarcher marche un film sous une variante.
func cmMarcher(f *cmFilm, v cmVariante, e cmEcouteur) (FrameClosureReport, *Observation, int) {
	md := marcheDetaillee{mesureDesTrames: nouvelleMesureDesTrames(f.reg, f.utiles, f.cfg)}
	obs := md.cfg.Obs
	monde := NewWorld(f.reg)
	monde.PoserTableAnticipee(ConstruireTableAnticipee(f.fc))
	marche := f.fc.MarcheDImageCle()
	occupes := 0
	poser := func(cle [2]int) {
		for _, l := range v.oracle[cle] {
			if _, lie := monde.ArchetypeForSlot(l.eid & 0x3fffffff); lie {
				occupes++
			}
			monde.BindFull(l.eid, l.ti)
		}
	}
	for _, c := range f.fc.ChunkNumbers() {
		data, pks, ok := f.fc.ChunkAt(c)
		if !ok {
			continue
		}
		monde.PoserChunkCourant(c)
		lierLeChunkAuMonde(monde, marche, data, pks, obs)
		if e != nil {
			e.debutDeChunk(c, data, pks, monde)
		}
		poser([2]int{c, -1})
		var tete func([]byte, *World, FrameConfig) (int, bool)
		if v.tete != nil {
			tete = v.tete(c)
		}
		for _, pk := range pks {
			p := cmPaquetDe(&md, tete, c, pk, data, monde)
			if p == nil {
				continue
			}
			if e != nil {
				e.paquet(c, p, monde)
			}
			poser([2]int{c, pk.Index})
		}
	}
	obs.solderLesNeufsRefuses()
	if e != nil {
		e.finDeFilm()
	}
	return md.rapport(), obs, occupes
}

// cmPaquetDe est [marcheDetaillee.paquet] + [marcheDetaillee.marcherPaquetDetaille], qui garde
// les records, les NEW refuses et la chaine de tete.
func cmPaquetDe(md *marcheDetaillee, tete func([]byte, *World, FrameConfig) (int, bool), c int,
	pk FilmPacket, data []byte, w *World) *cmPaquet {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return nil
	}
	pay := pk.Payload(data)
	p := &cmPaquet{pk: pk, pay: pay, strict: -2, d: PaquetDeCarte{Chunk: c, Index: pk.Index,
		TimestampUS: pk.TimestampUS, Bits: len(pay) * 8, DebutVueB: -1, FinVueB: -1}}
	debut := movementStateSkipLeadBits
	if _, present := PacketHeadEventType(pay); present {
		p.strict = marchLocateStrict(pay, w, md.cfg)
		loc := debutDeLaListe
		if tete != nil {
			loc = tete
		}
		debut, _ = loc(pay, w, md.cfg)
		if debut < 0 {
			md.listeNonLocalisee()
			p.d.ListeNonLocalisee, p.d.Cause, p.debut = true, causeListeNonLocalisee, -1
			return p
		}
		if p.strict >= 0 && debut < p.strict {
			p.chaine = cmChaine(pay, debut, p.strict, w, md.cfg)
		}
	}
	p.debut = debut
	obs := md.cfg.Obs
	n0 := len(obs.neufsRefuses)
	md.vueC = LectureVueC{}
	recs, rangs, l := md.marcherParRangs(pay, w, debut, &p.d)
	enTete := debut == md.cfg.PacketPreambleBits && md.cfg.PacketPreambleBits >= 1
	pm := paquetMarche{enTete: enTete, recs: recs, rangs: rangs, vueC: l}
	avantLus, avantFermes := md.rep.Utiles.Records, md.rep.Utiles.RecordsFermes
	md.classer(pm)
	p.d.UtilesEnJeu = (md.rep.Utiles.Records - avantLus) - (md.rep.Utiles.RecordsFermes - avantFermes)
	p.utilesFermes = md.rep.Utiles.RecordsFermes - avantFermes
	p.d.Fermee = l.Fermee
	if !l.Fermee {
		p.d.Cause = md.bloquantDuPaquet(pm).nom
	}
	decrireLesRecords(md.reg, recs, len(pay)*8, &p.d)
	p.recs, p.enTete = recs, enTete
	if len(obs.neufsRefuses) > n0 {
		p.refuses = append([]neufRefuse(nil), obs.neufsRefuses[n0:]...)
	}
	return p
}

// cmChaine relit les en-tetes de la chaine de tete que [debutParChaine] a acceptee (de `pos` a
// `debut`), par les memes pas ([pasDEssai]).
func cmChaine(pay []byte, pos, debut int, w *World, cfg FrameConfig) []cmEnTete {
	essai := cfg
	essai.Obs = nil
	extra := motFacultatifDEnTete(cfg)
	var out []cmEnTete
	for n := 0; n < plafondChaineDeTete && pos < debut; n++ {
		br := LecteurSur(pay)
		br.poserCadre(essai)
		br.SetBitPos(pos + extra)
		typ := readRecordType(br)
		id := readRecordID(br, essai.IDLowBits, essai.IDBase)
		out = append(out, cmEnTete{typ: typ, slot: id & 0x3fffffff})
		fin, ok := pasDEssai(pay, pos, extra, w, essai)
		if !ok || fin <= pos {
			break
		}
		pos = fin
	}
	return out
}

// cmOuvrir ouvre un film comme `cmd_fermeture` (contexte des instruments).
func cmOuvrir(t *testing.T, racine, id string, utiles UsagesProduit) (*cmFilm, bool) {
	t.Helper()
	fc, _, _ := ContexteDeFilm(racine + "/" + id)
	if fc == nil {
		t.Errorf("%s : film illisible", id)
		return nil, false
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Errorf("%s : registre : %v", id, err)
		return nil, false
	}
	cfg := fc.CadreDeBalayage()
	if !cfg.Profil.Grammaire.ClassesDeVue {
		t.Errorf("%s : cadre sans classes de vue", id)
		return nil, false
	}
	return &cmFilm{id: id, build: cmBuild(fc), fc: fc, reg: reg, cfg: cfg, utiles: utiles}, true
}

// cmBuild : le build en clair (regle de `cmd_fermeture`).
func cmBuild(fc *FilmContext) string {
	film := fc.Film()
	if raw, ok := FilmRegistryChunk(film); ok {
		if ident, err := ReadFilmIdentity(raw); err == nil && ident.Build != "" {
			return ident.Build
		}
	}
	if v, ok := FilmMajorVersion(film); ok {
		return "version-" + strconv.Itoa(int(v))
	}
	return "inconnu"
}

// cmUtiles relit la colonne `product_use` de la table ECS (regle de `cmd_fermeture/table.go`).
func cmUtiles(t *testing.T) UsagesProduit {
	t.Helper()
	brut, err := os.ReadFile("testdata/ecs_table.tsv")
	if err != nil {
		t.Fatal(err)
	}
	out := UsagesProduit{}
	for i, l := range strings.Split(strings.TrimRight(string(brut), "\n"), "\n") {
		c := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if i == 0 || len(c) < 12 {
			continue
		}
		ti, err := strconv.Atoi(c[0])
		if err != nil || ti < 0 {
			continue
		}
		v := strings.TrimSpace(c[11])
		if v != "" && !strings.HasPrefix(v, "aucun") {
			out[CleComposant(ti, c[3])] = true
		}
	}
	return out
}
