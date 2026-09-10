package objectiveevents

// slotidentity_elimination.go — L'IDENTITE D'UN SLOT D'ENTITE PAR ELIMINATION, MANCHE PAR MANCHE.
//
// # LE TROU QUE CE FICHIER FERME (diagnostic R4, `.ai/V7.5/v2/RESTES_R4_R5_2026-09-08.md` §2-§4)
//
// [bestDeathClaim] exige `deathInstantMin` = 3 instants coincidents pour nommer un couple
// (slot, manche). Un joueur qui meurt MOINS DE TROIS FOIS dans une manche lui echappe par
// construction — et zero fois, ce qui arrive, le met hors de portee absolue. Mesure :
// `51ebbc0f`, manche 0, un joueur a 0 mort : sa manche entiere n'est publiee NULLE PART, ecart
// cumule K/D/A de 9 contre la feuille. Generalite : 8 couples (xuid, manche) perdus sur 3 films,
// et dans les 8 cas le joueur meurt 0 ou 1 fois dans la manche perdue — aucun contre-exemple.
//
// [RoundIdentity.CompletedByLines] existait exactement pour ce trou, mais elle est gardee
// MONO-MANCHE : le triplet apparie des TOTAUX DE MATCH, et en multi-manche le slot est
// reattribue tandis que le compteur repart de zero. La lever serait reintroduire le defaut que
// `d173b1a8c` a corrige.
//
// # POURQUOI L'ELIMINATION, ET PAS LE TRIPLET
//
// L'elimination ne suppose RIEN du contenu : elle constate qu'il ne reste qu'UNE affectation
// possible dans la manche — exactement un slot emetteur non nomme, exactement un xuid de la
// feuille non attribue. Ce n'est pas une deduction par ressemblance, c'est un appariement force.
//
// # LE CONTROLE QUI REND LA DEDUCTION VERIFIABLE
//
// Quand la feuille est la, le RESIDU (total de la feuille moins la somme des manches deja
// nommees du meme xuid) doit egaler le segment du slot candidat dans la manche. Sur `51ebbc0f` :
// residu K = 15 − 7 = 8, D = 4 − 4 = 0, A = 2 − 1 = 1 — le slot candidat de la manche 0 doit
// porter exactement (8, 0, 1). Un desaccord = ON NE NOMME PAS.
//
// # CE QU'ELLE NE FERME PAS, ET IL FAUT LE DIRE
//
// `64e8adfa` garde 5 couples perdus sur 8 slots : l'elimination n'y a pas d'unicite. Le cas
// general appartient au lien DIRECT index de joueur <-> slot de statborg, que le film ne porte
// pas aujourd'hui (inventaire P1, E3 « exige decodeur »).

// Les trois provenances d'une identite de slot d'entite, dans l'ordre de la force de preuve.
// Elles alimentent la provenance publiee du registre (`canonical.LinkMethod`) ; ce paquet les
// garde en chaines nues pour rester une feuille du decodage.
const (
	// OriginDeathInstants : les instants de mort apparies au fil — la voie principale.
	OriginDeathInstants = "instants_de_mort"
	// OriginSheetTriplet : le triplet K/D/A confronte a la feuille (mono-manche seulement).
	OriginSheetTriplet = "triplet_feuille"
	// OriginElimination : l'unique affectation restante dans la manche, controlee par le residu.
	OriginElimination = "elimination_roster"
)

// Origin rend la voie par laquelle le couple (manche, slot) a ete nomme, ou une chaine vide.
func (ri RoundIdentity) Origin(round, slot int) string {
	if m := ri.origins[round]; m != nil {
		return m[slot]
	}
	return ""
}

// EmittingPlayerSlots rend, pour une manche, les slots de JOUEUR qui emettent au moins un
// enregistrement porteur des compteurs de base. C'est le denominateur de l'elimination : un
// slot qui ne parle pas n'a rien a nommer.
func EmittingPlayerSlots(recs []StatRecord, round int) []int {
	vus := map[int]bool{}
	for _, r := range recs {
		if r.Round != round || IsTeamSlot(r.Slot) {
			continue
		}
		if _, ok := r.Comps[coreKillsComp]; ok {
			vus[r.Slot] = true
		}
	}
	return sortedInts(vus)
}

// CompletedByElimination complete l'identite PAR MANCHE quand il ne reste qu'une affectation
// possible, et que le residu de la feuille la confirme.
//
// `lines` vide rend l'identite INCHANGEE : l'artefact reste publiable hors ligne, sans base —
// meme garde que [RoundIdentity.CompletedByLines].
//
// TROIS GARDES, AUCUNE NEGOCIABLE : completer sans jamais contredire (un slot deja nomme garde
// son nom) ; aucun xuid deux fois dans la meme manche ; le residu de la feuille doit egaler le
// segment du slot candidat.
func (ri RoundIdentity) CompletedByElimination(recs []StatRecord, lines []PlayerLine) RoundIdentity {
	if len(lines) == 0 || len(ri.byRound) == 0 {
		return ri
	}
	seg := segmentsParManche(recs)
	out := ri.copieProfonde()
	for _, round := range ri.Rounds() {
		slot, xuid, ok := candidatUniqueDeManche(out.byRound, recs, lines, round)
		if !ok {
			continue
		}
		if !residuConcorde(seg, out.byRound, lines, round, slot, xuid) {
			continue
		}
		out.byRound[round][slot] = xuid
		out.origins[round][slot] = OriginElimination
	}
	return out
}

// candidatUniqueDeManche rend le couple (slot, xuid) quand la manche n'a qu'UNE affectation
// possible : exactement un slot emetteur non nomme, exactement un xuid de la feuille libre.
func candidatUniqueDeManche(byRound map[int]map[int]string, recs []StatRecord,
	lines []PlayerLine, round int) (int, string, bool) {
	nommes := byRound[round]
	var muets []int
	for _, slot := range EmittingPlayerSlots(recs, round) {
		if nommes[slot] == "" {
			muets = append(muets, slot)
		}
	}
	pris := map[string]bool{}
	for _, x := range nommes {
		pris[x] = true
	}
	var libres []string
	for _, l := range lines {
		if !pris[l.XUID] {
			libres = append(libres, l.XUID)
		}
	}
	if len(muets) != 1 || len(libres) != 1 {
		return 0, "", false
	}
	return muets[0], libres[0], true
}

// segmentKDA porte les trois compteurs de base d'un slot DANS UNE MANCHE.
type segmentKDA struct{ kills, deaths, assists int }

// segmentsParManche rend, par manche et par slot de joueur, le segment (frags, morts,
// assistances) de la manche — la DERNIERE valeur emise, les compteurs repartant de zero a
// chaque manche.
func segmentsParManche(recs []StatRecord) map[int]map[int]segmentKDA {
	out := map[int]map[int]segmentKDA{}
	poser := func(c StatComponent, set func(*segmentKDA, int)) {
		for slot, byRound := range SeriesByRound(recs, c, false) {
			for round, pts := range byRound {
				if len(pts) == 0 {
					continue
				}
				if out[round] == nil {
					out[round] = map[int]segmentKDA{}
				}
				s := out[round][slot]
				set(&s, int(pts[len(pts)-1].Value))
				out[round][slot] = s
			}
		}
	}
	poser(KillsComponent, func(s *segmentKDA, v int) { s.kills = v })
	poser(DeathsComponent, func(s *segmentKDA, v int) { s.deaths = v })
	poser(AssistsComponent, func(s *segmentKDA, v int) { s.assists = v })
	return out
}

// residuConcorde verifie que le segment du slot candidat vaut EXACTEMENT le residu de la
// feuille — le total du joueur moins la somme de ses manches deja nommees.
//
// C'est ce controle qui distingue un appariement force d'une devinette : sans lui, une manche
// mal decoupee attribuerait des compteurs qui ne sont pas ceux du joueur, et l'ecran ne
// montrerait rien.
//
// LE CALCUL DU RESIDU N'EST ECRIT QU'UNE FOIS ([residuDeManche], slotidentity_residue.go) : la
// voie par residu de manche le PRODUIT, celle-ci le CONTROLE, et deux ecritures du meme calcul
// divergeraient (regle n° 6 du depot).
func residuConcorde(seg map[int]map[int]segmentKDA, byRound map[int]map[int]string,
	lines []PlayerLine, round, slot int, xuid string) bool {
	if !porteLeXUID(lines, xuid) {
		return false
	}
	return seg[round][slot] == residuDeManche(seg, byRound, lines, round, xuid)
}

// porteLeXUID dit que la feuille contient bien ce joueur — un xuid absent n'a pas de residu, et
// un residu nul n'est pas la meme chose qu'une absence de ligne.
func porteLeXUID(lines []PlayerLine, xuid string) bool {
	for _, l := range lines {
		if l.XUID == xuid {
			return true
		}
	}
	return false
}

// copieProfonde duplique les tables du resolveur : la completion ne doit jamais modifier
// l'identite que son appelant lui a passee (deux calques la partagent, memorisee).
func (ri RoundIdentity) copieProfonde() RoundIdentity {
	out := RoundIdentity{
		byRound: make(map[int]map[int]string, len(ri.byRound)),
		origins: make(map[int]map[int]string, len(ri.byRound)),
		starts:  ri.starts,
	}
	for round, m := range ri.byRound {
		copie := make(map[int]string, len(m))
		for slot, xuid := range m {
			copie[slot] = xuid
		}
		out.byRound[round] = copie
		orig := make(map[int]string, len(m))
		for slot, o := range ri.origins[round] {
			orig[slot] = o
		}
		out.origins[round] = orig
	}
	return out
}

// sortedInts rend les cles d'un ensemble d'entiers en ordre croissant.
func sortedInts(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
