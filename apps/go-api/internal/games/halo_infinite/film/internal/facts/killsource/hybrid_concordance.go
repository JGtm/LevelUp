package killsource

// hybrid_concordance.go — LA CONCORDANCE DES DEUX VOIES : [pass.concordance], [atInstant],
// [pass.noteInstant].
//
// SORTI DE `hybrid.go` PAR DEPLACEMENT PUR au lot J7.1 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM) :
// le fichier atteignait le plafond de 500 lignes (`archlint/film_file_size_test.go`) au moment ou
// la garde du nom de remplissage s ajoutait aux temps 4 et 5. Aucun octet de code n est change.

// concordance : les deux voies se contredisent-elles ?
//
// PORTEE, et elle est essentielle : la comparaison se fait sur le SCAN ENTIER, redondances
// COMPRISES — pas sur la population amputee que l hybride consulte. Comparer apres avoir retire
// les redondants ne mesurerait rien, puisque le redondant est PRECISEMENT l enregistrement que
// les deux voies partagent.
//
// `disagree` doit valoir ZERO. S il bouge, l hybride n est plus une PREFERENCE mais un
// ARBITRAGE, et il faut le documenter comme tel.
func (p *pass) concordance(walkCands []candidate) {
	byT := map[int]*atInstant{}
	for _, cd := range walkCands {
		p.noteInstant(byT, cd, PathWalk)
	}
	for _, cd := range p.ctx.scanCands {
		p.noteInstant(byT, cd, PathScan)
	}
	for _, b := range byT {
		if b.n > 2 || (b.n == 2 && (!b.hasWalk || !b.hasScan)) {
			p.multiCand++ // plusieurs candidats DISTINCTS, pas un simple doublon de voie
		}
		if !b.hasWalk || !b.hasScan {
			continue
		}
		if b.walk.tag == b.scan.tag && b.walk.cat == b.scan.cat {
			p.agree++
		} else {
			p.disagree++
		}
	}
}

// atInstant : ce que les deux voies disent d un meme instant.
type atInstant struct {
	walk, scan       candidate
	hasWalk, hasScan bool
	n                int
}

// noteInstant : range un candidat apparie sous son instant, avec la voie qui l a produit.
func (p *pass) noteInstant(byT map[int]*atInstant, cd candidate, path Path) {
	if cd.victim == cd.killer {
		return
	}
	e, _ := p.ctx.matchExact(cd)
	if e == nil {
		return
	}
	b := byT[e.timeMS]
	if b == nil {
		b = &atInstant{}
		byT[e.timeMS] = b
	}
	b.n++
	if path == PathWalk {
		b.walk, b.hasWalk = cd, true
		return
	}
	b.scan, b.hasScan = cd, true
}
