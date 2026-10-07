package killsource

// assist_doublons_test.go — UN ENREGISTREMENT LU DEUX FOIS NE FABRIQUE RIEN (lot J7.6 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat FK-6).
//
// # LE DEFAUT QUE CE TEMOIN FERME
//
// Le generateur de candidats du rattrapage des kills (`grammar/kills_rattrapes.go`, la recherche
// bit a bit que killsource faisait sur toute la trame) retient parfois DEUX FOIS le meme kill-event 85 dans
// un paquet, a deux positions de bit voisines, champs identiques (RE_LOG 7ter.77 : les deux morts a
// multi-attachement du corpus, a 15 bits d ecart). Le champ `bit` de [killEventRec] etait prevu
// « pour dedoublonner » et n etait jamais lu. Le premier exemplaire etait consomme par le couple du
// meme instant ; le SECOND restait libre, et un kill orphelin du meme tueur dans la fenetre le
// lisait : il publiait la victime d une AUTRE mort — un couple FAUX, compte en `Contradiction`.
// Cote assistant, les deux exemplaires s attachaient a la meme mort (`Multi`).
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : vider [assistScan.dedoublonner].

import "testing"

func TestDoublonsDEnregistrement_AucunCoupleFabrique(t *testing.T) {
	kf := &killFeed{
		events: []feedEvent{
			{timeMS: 1000, killer: "K", victim: "V", victimXUID: 22}, // couple ecrit au meme instant
			{timeMS: 2000, killer: "K"},                              // kill orphelin de K, 1 s plus tard
		},
		names:  []string{"K", "V"},
		xuidDe: map[string]uint64{"V": 22},
	}
	r := buildRoster(kf, botMeta{}, true, FilmTable{Build: "b", Seats: map[int]string{0: "K", 1: "V"}},
		indexParMotif{})
	r.perm = []int{0, 1}
	champs := killEventFields{victim: 1, killer: 0, assist: -1, killerPct: 100, assistPct: 149}
	premier, second := champs, champs
	s := &assistScan{recs: []killEventRec{
		{ms: 1000, chunk: 1, pidx: 5, bit: 100, fields: premier},
		{ms: 1000, chunk: 1, pidx: 5, bit: 115, fields: second},
	}}
	s.dedoublonner()
	st := kf.resoudreCouples(s.recs, r)

	if st.Contradiction != 0 || st.Lus != 0 || len(kf.pairs) != 1 {
		t.Fatalf("couples %+v, compteurs %+v : le second exemplaire du meme enregistrement a fabrique "+
			"un couple pour le kill orphelin", kf.pairs, st)
	}
	if len(s.recs) != 1 || s.recs[0].bit != 100 {
		t.Errorf("enregistrements gardes = %+v, attendu le seul PREMIER exemplaire (bit 100)", s.recs)
	}

	c := &decodeCtx{roster: r, opts: DefaultOptions()}
	kills := []Kill{{TimeMS: 1000, Victim: "V", Feed: FeedTruth{Killer: "K", Present: true},
		paquet: paquetID{chunk: 1, pidx: 5, ok: true}}}
	a := c.attachAssists(kills, s)
	if a.Multi != 0 || a.Attached != 1 || a.Doublons != 1 {
		t.Errorf("assistants : multi %d, attaches %d, doublons %d — attendu 0, 1, 1", a.Multi, a.Attached, a.Doublons)
	}
}
