//go:build research

package grammar

// p3_armes_naissance_recherche_research_test.go — LA RECHERCHE D EN-TETE EXACT de la sonde P3 (voir
// l en-tete de `p3_armes_naissance_research_test.go`).
//
// POURQUOI. La marche de la vue B ne rend presque aucun record NEW ti=35 aux naissances (mesure
// P3.0) : elle cale avant. La question de la sonde (le record NEW porte-t-il les armes ?) se pose
// donc AU RECORD, pas a la marche : pour chaque vie, dans les paquets delta de [debut - 1 s,
// debut + 3 s], on essaie CHAQUE bit comme debut de record — type NEW (prefixe `0` puis `01`), eid
// dont le slot est celui de la vie, R(6) = 35 — puis `TraverseEntity` du corps (la grammaire de
// production, observateur pose). L en-tete (~25 bits fixes) est le filtre ; la traversee propre du
// corps (etat par defaut bit-exact, masque, composants jusqu au bout, sans sortir du paquet) est
// la confirmation.
//
// TEMOIN DE LA RECHERCHE : la meme recherche pour le slot d une vie dans la fenetre d une AUTRE vie
// (un autre instant) — attendu : aucun record propre.

import (
	"fmt"
	"testing"
)

// p3FenetreAvant / p3FenetreApres bornent la recherche autour du debut de vie (pas de 100 ms).
const (
	p3FenetreAvant = 10
	p3FenetreApres = 50 // 5 s : couvre la vie ouverte 4,65 s AVANT la creation du corps (M1, slot 523)
)

// p3Paquet est un paquet delta du film, date sur l axe du document.
type p3Paquet struct {
	chunk, trame int
	pk           FilmPacket
	pay          []byte
}

// p3Recherche porte les compteurs de la recherche.
type p3Recherche struct {
	candidats, propres, desync, surnombre int
	temoinCandidats, temoinPropres        int
	temoinLus                             int
	dansMarche, apresMarche               int
	avantDebut, avantDebutListe           int // record AVANT le debut de la vue B localise (dont paquets a liste)
	listeFinie, finEgale                  int // liste marchee jusqu a son terminateur ; fin == bit du NEW
	finAvant, finApres                    int // fin de liste avant / apres le bit du NEW
	typesListe                            map[int]int
}

// p3Paquets charge les paquets delta du film et les place sur l axe du document.
func p3Paquets(tc t516Temoin, origine int64) []p3Paquet {
	var out []p3Paquet
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			tr := int((int64(pk.TimestampUS) - origine) / tcgPasUS) //nolint:gosec // horodatage de film
			out = append(out, p3Paquet{chunk: c, trame: tr, pk: pk, pay: pk.Payload(data)})
		}
	}
	return out
}

// p3Rechercher cherche le record NEW ti=35 de chaque vie, puis le temoin.
func p3Rechercher(t *testing.T, tc t516Temoin, vies []*p3Vie, origine int64,
	cat map[uint32]bool, ctx r7Ctx) p3Recherche {
	t.Helper()
	if tc.cfg.HasExtraFields {
		t.Fatalf("HasExtraFields : la forme d en-tete cherchee ne vaut pas")
	}
	paquets := p3Paquets(tc, origine)
	armeIdx := p3IndexArmes(tc.reg)
	r := p3Recherche{typesListe: map[int]int{}}
	for i, v := range vies {
		trouves := p3ChercherSlot(tc, paquets, armeIdx, v.slot, v.start)
		v.candidats = trouves
		for k := range trouves {
			n := &trouves[k]
			r.candidats++
			if n.desync != -1 {
				r.desync++
			} else {
				r.propres++
			}
			if v.trouvee == nil {
				v.trouvee = n
			} else {
				r.surnombre++
			}
		}
		autre := vies[(i+len(vies)/2)%len(vies)]
		if autre.slot == v.slot || p3Abs(autre.start-v.start) < p3FenetreAvant+p3FenetreApres {
			continue
		}
		for _, n := range p3ChercherSlot(tc, paquets, armeIdx, v.slot, autre.start) {
			r.temoinCandidats++
			if n.desync == -1 {
				r.temoinPropres++
			}
			armes, pos := p3ScanCatalogue(p3PayDe(paquets, n), &n, tc.cfg, cat)
			if pos >= 0 {
				r.temoinLus++
			}
			t.Logf("   TEMOIN slot %d cherche a t %d (vie du slot %d) : en-tete a t=%d bit %d desync %d ; "+
				"lecture par catalogue %d emplacements a %d", v.slot, autre.start, autre.slot, n.trame, n.bit,
				n.desync, len(armes), pos)
		}
	}
	p3SituerDansMarche(tc, vies, &r, ctx)
	return r
}

// p3PayDe rend le payload du paquet d une naissance.
func p3PayDe(paquets []p3Paquet, n p3Naissance) []byte {
	for _, q := range paquets {
		if q.chunk == n.chunk && q.pk.TimestampUS == n.ts {
			return q.pay
		}
	}
	return nil
}

// p3IndexArmes rend les index des emplacements `weapon-state-type-info` du bipede.
func p3IndexArmes(reg *Registry) map[int]bool {
	arch, _ := reg.Archetype(BipedTypeIndex)
	out := map[int]bool{}
	for i, nom := range arch.Components {
		if nom == compWeaponStateTypeInfo {
			out[i] = true
		}
	}
	return out
}

// p3ChercherSlot essaie chaque bit des paquets de la fenetre comme en-tete NEW ti=35 du slot.
func p3ChercherSlot(tc t516Temoin, paquets []p3Paquet, armeIdx map[int]bool, slot uint32,
	debut int) []p3Naissance {
	var out []p3Naissance
	cfg := tc.cfg
	var vues [][2]uint32
	cfg.Obs = NouvelleObservation()
	cfg.Obs.HeldWeaponHook = func(h, l uint32) { vues = append(vues, [2]uint32{h, l}) }
	for _, q := range paquets {
		if q.trame < debut-p3FenetreAvant || q.trame > debut+p3FenetreApres {
			continue
		}
		n := len(q.pay) * 8
		br := LecteurSur(q.pay)
		br.poserCadre(cfg)
		for o := 0; o+3+cfg.IDLowBits+2+6 <= n; o++ {
			br.SetBitPos(o)
			if readRecordType(br) != recNew {
				continue
			}
			id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
			if id&0x3fffffff != slot {
				continue
			}
			corps := br.BitPos()
			if br.ReadBits(6) != BipedTypeIndex {
				continue
			}
			br.SetBitPos(corps)
			vues = vues[:0]
			tr := TraverseEntity(br, tc.reg, cfg.NewDefaultStateBits)
			if br.BitPos() > n && tr.DesyncAt == -1 {
				tr.DesyncAt = 99 // la traversee sort du paquet : pas un record
			}
			out = append(out, p3Record(q, slot, id, o, tr, armeIdx, vues))
		}
	}
	return out
}

// p3Record rend la naissance lue a un bit donne.
func p3Record(q p3Paquet, slot, id uint32, o int, tr EntityTrace, armeIdx map[int]bool,
	vues [][2]uint32) p3Naissance {
	x := p3Naissance{ts: q.pk.TimestampUS, chunk: q.chunk, slot: slot, gen: id >> 30,
		desync: tr.DesyncAt, trame: q.trame, bit: o}
	for _, i := range tcgIndices(tr.Mask) {
		if armeIdx[i] {
			x.annoncees++
		}
	}
	for _, cr := range tr.Comps {
		if !armeIdx[cr.Index] {
			continue
		}
		a := p3Relire(q.pay, cr)
		for _, v := range vues {
			if v[0] == a.hi && v[1] == a.lo {
				a.crochet = true
			}
		}
		x.armes = append(x.armes, a)
	}
	return x
}

// p3SituerDansMarche rejoue la marche de la vue B sur le paquet de chaque record retenu et dit si
// le record est AVANT la fin de vue B (la marche l a traverse ou l a rate) ou APRES (elle a cale
// ou clos avant lui).
func p3SituerDansMarche(tc t516Temoin, vies []*p3Vie, r *p3Recherche, ctx r7Ctx) {
	type cle struct {
		chunk int
		ts    uint64
	}
	cible := map[cle][]*p3Naissance{}
	for _, v := range vies {
		if n := v.trouvee; n != nil {
			cible[cle{n.chunk, n.ts}] = append(cible[cle{n.chunk, n.ts}], n)
		}
	}
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				debut = marchLocate(pay, w, tc.cfg)
			}
			ns := cible[cle{c, pk.TimestampUS}]
			if debut < 0 {
				for _, n := range ns {
					n.finMarche = -1
					r.apresMarche++
				}
				continue
			}
			mar := t519Marcher(pay, w, tc.cfg, debut)
			for _, n := range ns {
				n.finMarche, n.hitEndB = mar.m.FinVueB, mar.m.HitEndB
				_, liste := PacketHeadEventType(pay)
				switch {
				case n.bit < debut:
					r.avantDebut++
					if liste {
						r.avantDebutListe++
						p3FinDeListe(pay, n.bit, ctx, r)
					}
				case n.bit < mar.m.FinVueB:
					r.dansMarche++
				default:
					r.apresMarche++
				}
			}
		}
	}
}

// p3FinDeListe marche la liste d evenements du paquet (marcheur R7 de la sonde P1) et situe sa fin
// par rapport a l en-tete du record NEW.
func p3FinDeListe(pay []byte, bit int, ctx r7Ctx, r *p3Recherche) {
	evs, stop, fin := tcgMarcherListe(pay, ctx)
	for _, e := range evs {
		r.typesListe[e.typ]++
	}
	if stop != r7StopFin {
		return
	}
	r.listeFinie++
	switch {
	case fin == bit:
		r.finEgale++
	case fin < bit:
		r.finAvant++
	default:
		r.finApres++
	}
}

// p3RapportRecherche publie la recherche.
func p3RapportRecherche(t *testing.T, r p3Recherche) {
	t.Helper()
	t.Logf("== P3.5 RECHERCHE D EN-TETE : en-tetes NEW/slot/ti=35 %d ; traversees PROPRES %d ; desync %d ; "+
		"en-tetes en surnombre dans une meme fenetre %d (le PREMIER est retenu)", r.candidats, r.propres, r.desync, r.surnombre)
	t.Logf("   TEMOIN (slot d une vie cherche dans la fenetre d une autre) : en-tetes %d, propres %d, lus "+
		"par catalogue %d", r.temoinCandidats, r.temoinPropres, r.temoinLus)
	t.Logf("   records retenus (premier en-tete de la fenetre) : AVANT le debut de vue B localise %d (dont "+
		"paquets a liste d evenements %d) ; entre le debut et la fin de vue B %d ; APRES la fin de vue B ou "+
		"paquet non localise %d", r.avantDebut, r.avantDebutListe, r.dansMarche, r.apresMarche)
	t.Logf("   liste d evenements de ces paquets (marcheur R7) : marchee jusqu au terminateur %d ; sa FIN tombe "+
		"SUR l en-tete du NEW %d, avant %d, apres %d ; types d evenements %s", r.listeFinie, r.finEgale,
		r.finAvant, r.finApres, tcgCompteTri(r.typesListe, tcgNomType, 0))
}

// p3PorteeLocalisation borne la localisation : bits parcourus apres l en-tete du record NEW.
const p3PorteeLocalisation = 6000

// p3Localiser cherche, dans le paquet du record NEW retenu de chaque vie, les familles de ses
// oracles (O1 puis O2) comme mot de 32 bits a TOUT decalage de [en-tete, en-tete + portee] ; rend
// les decalages relatifs a l en-tete et au debut du premier emplacement d arme lu. TEMOIN : les
// memes familles cherchees dans le record d une AUTRE vie de familles differentes.
func p3Localiser(t *testing.T, tc t516Temoin, vies []*p3Vie, origine int64) {
	t.Helper()
	paquets := p3Paquets(tc, origine)
	pay := map[[2]uint64][]byte{}
	for _, q := range paquets {
		pay[[2]uint64{uint64(q.chunk), q.pk.TimestampUS}] = q.pay //nolint:gosec // chunk positif
	}
	relHdr, relArme := map[int]int{}, map[int]int{}
	var cherches, trouves int
	for _, v := range vies {
		n := v.trouvee
		if n == nil || (!v.aO1 && !v.aO2) {
			continue
		}
		b := pay[[2]uint64{uint64(n.chunk), n.ts}] //nolint:gosec // chunk positif
		fams := append([]uint32(nil), v.o1Ordre...)
		if v.aO2 && !p3Contient(fams, v.o2) {
			fams = append(fams, v.o2)
		}
		arme0 := -1
		if len(n.armes) > 0 {
			arme0 = n.armes[0].startBit
		}
		var lignes []string
		for _, f := range fams {
			cherches++
			for _, o := range p3Occurrences(b, f, n.bit, n.bit+p3PorteeLocalisation) {
				trouves++
				relHdr[o-n.bit]++
				if arme0 >= 0 {
					relArme[o-arme0]++
				}
				lignes = append(lignes, fmt.Sprintf("%08X@+%d(arme0%+d)", f, o-n.bit, o-arme0))
			}
		}
		t.Logf("   LOC slot %d t=%d bit %d arme0 %d : %v", v.slot, n.trame, n.bit, arme0, lignes)
	}
	t.Logf("== P3.6 LOCALISATION DES FAMILLES DES ORACLES dans le paquet du NEW (portee %d bits) : %d "+
		"familles cherchees, %d occurrences ; decalages / en-tete %v ; / premier emplacement lu %v",
		p3PorteeLocalisation, cherches, trouves, relHdr, relArme)
}

// p3Occurrences rend les bits de [de, a) ou un mot de 32 bits (lu comme `ReadBits(32)`) vaut f.
func p3Occurrences(b []byte, f uint32, de, a int) []int {
	var out []int
	n := len(b) * 8
	br := LecteurSur(b)
	for o := de; o < a && o+32 <= n; o++ {
		br.SetBitPos(o)
		if uint32(br.ReadBits(32)) == f {
			out = append(out, o)
		}
	}
	return out
}

// p3Chainer verifie PAR LA GRAMMAIRE la position mesuree des familles : depuis le bit qui precede
// la premiere famille (la porte de l emplacement), `consumeWeaponStateTypeInfoVariant` doit lire
// cette famille puis s arreter EXACTEMENT sur la porte de l emplacement suivant annonce au masque —
// et ainsi de suite. Publie aussi, pour quelques records, les composants que la traversee de
// production croit lire (index, nom, debut relatif a l en-tete).
func p3Chainer(t *testing.T, tc t516Temoin, vies []*p3Vie, origine int64) {
	t.Helper()
	paquets := p3Paquets(tc, origine)
	pay := map[[2]uint64][]byte{}
	for _, q := range paquets {
		pay[[2]uint64{uint64(q.chunk), q.pk.TimestampUS}] = q.pay //nolint:gosec // chunk positif
	}
	arch, _ := tc.reg.Archetype(BipedTypeIndex)
	armeIdx := p3IndexArmes(tc.reg)
	var essais, chaines, dumps int
	for _, v := range vies {
		n := v.trouvee
		if n == nil || !v.aO1 || len(v.o1Ordre) == 0 {
			continue
		}
		b := pay[[2]uint64{uint64(n.chunk), n.ts}] //nolint:gosec // chunk positif
		occ := p3Occurrences(b, v.o1Ordre[0], n.bit, n.bit+p3PorteeLocalisation)
		if len(occ) == 0 {
			continue
		}
		essais++
		br := LecteurSur(b)
		br.poserCadre(tc.cfg)
		br.SetBitPos(occ[0] - 1)
		var lus []string
		for k := 0; k < n.annoncees; k++ {
			debut := br.BitPos()
			var hi uint32 = noVariant
			br2 := LecteurSur(b)
			br2.SetBitPos(debut)
			if br2.ReadBit() {
				hi = uint32(br2.ReadBits(32))
			}
			consumeWeaponStateTypeInfoVariant(br)
			lus = append(lus, fmt.Sprintf("+%d:%08X(%db)", debut-n.bit, hi, br.BitPos()-debut))
		}
		ok, lues := true, p3FamillesLues(b, occ[0]-1, n.annoncees, tc.cfg)
		for _, f := range v.o1Ordre {
			if !p3Contient(lues, f) {
				ok = false
			}
		}
		if ok {
			chaines++
		}
		t.Logf("   CHAINE slot %d : %v -> familles de l oracle toutes lues en chaine %v", v.slot, lus, ok)
		if dumps < 4 {
			dumps++
			p3DumpTraversee(t, b, n, arch.Components, armeIdx, tc)
		}
	}
	t.Logf("== P3.7 CHAINAGE GRAMMATICAL depuis la position mesuree : %d/%d records ou les familles de "+
		"l oracle se lisent toutes en enchainant les emplacements annonces", chaines, essais)
}

// p3FamillesLues enchaine `annoncees` emplacements d arme depuis `debut` et rend les familles lues.
func p3FamillesLues(b []byte, debut, annoncees int, cfg FrameConfig) []uint32 {
	br := LecteurSur(b)
	br.poserCadre(cfg)
	br.SetBitPos(debut)
	var out []uint32
	for k := 0; k < annoncees; k++ {
		br2 := LecteurSur(b)
		br2.SetBitPos(br.BitPos())
		if br2.ReadBit() {
			out = append(out, uint32(br2.ReadBits(32)))
		}
		consumeWeaponStateTypeInfoVariant(br)
	}
	return out
}

// p3DumpTraversee publie les composants que la traversee de production lit dans un record NEW.
func p3DumpTraversee(t *testing.T, b []byte, n *p3Naissance, noms []string, armeIdx map[int]bool,
	tc t516Temoin) {
	t.Helper()
	br := LecteurSur(b)
	br.poserCadre(tc.cfg)
	br.SetBitPos(n.bit)
	readRecordType(br)
	readRecordID(br, tc.cfg.IDLowBits, tc.cfg.IDBase)
	corps := br.BitPos()
	tr := TraverseEntity(br, tc.reg, tc.cfg.NewDefaultStateBits)
	var s []string
	for _, cr := range tr.Comps {
		nom := ""
		if cr.Index < len(noms) {
			nom = noms[cr.Index]
		}
		s = append(s, fmt.Sprintf("i%d %s @+%d", cr.Index, nom, cr.StartBit-n.bit))
	}
	t.Logf("   TRAVERSEE slot %d : corps @+%d, etat par defaut %d bits, masque %v, fin @+%d, desync %d : %v",
		n.slot, corps-n.bit, tr.DefaultBits, tcgIndices(tr.Mask), tr.EndBit-n.bit, tr.DesyncAt, s)
}
