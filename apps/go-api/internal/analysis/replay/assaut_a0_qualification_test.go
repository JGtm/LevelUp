package replay

// assaut_a0_qualification_test.go — LOT A (Assaut), PHASE A0.2/A0.3 : LA QUALIFICATION DU
// CORPUS ET LES RELEVES DE SCORE DE MODE.
//
// LE PLAN EST COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_ASSAUT_LOT_A_2026-08-27.md`) ; le
// protocole `A_PROTOCOLE.md` fige ce que cet instrument aura publie. Trois questions par
// film, aucune n'est supposee :
//
//	BORNES     la carte est-elle au catalogue de quantification (`map_quant_bounds.json`) ?
//	           Sans elles le film est INDECODABLE en coordonnees monde (lecon Live Fire,
//	           lot O) — il sort avec sa raison, on ne repare pas le catalogue.
//	SITES      combien de sites `assault_bomb` le catalogue d'objectifs
//	           (`map_objectives.json`) porte-t-il pour cette carte ? C'est l'ANCRAGE de
//	           A1/A3 : zero site = ancrage au site indisponible, et cela se PUBLIE.
//	PONT       part des slots de bipede nommes par le pont d'identite — la precondition
//	           >= 50 % heritee du lot O (`d8PontMinimum`), meme instrument.
//
// ET LES RELEVES A0.3 : les manches reelles (`RealRounds`) et le score de MODE par equipe
// (composant 0 A, slots 6 et 8) — chaque increment date, par manche et en cumule. C'est la
// corroboration d'A3 (armement/explosion = increments dates ?) et elle se fige au protocole.
//
// REGIME : gardes `ATT_FILM` (racine du cache) + `ASSAUT_FILM` (identifiant court), UN FILM
// PAR PROCESSUS, lecture seule, AUCUNE base.
//
//	$env:ATT_FILM="<repo>/data/cache"; $env:ASSAUT_FILM="35b75a31"
//	go test ./internal/analysis/replay/ -run AssautA0Qualification -v

import (
	"context"
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/filmproc"
)

const (
	// a0FilmEnv designe LE film qualifie par ce processus — une boucle sur neuf films dans
	// un seul processus est exactement ce que la doctrine de l'executeur borne interdit.
	a0FilmEnv = "ASSAUT_FILM"
	// a0RoleSite est le role du catalogue de carte qui ancre l'objet bombe (A1) et les
	// sites d'amorcage (A3). C'est le seul role Assaut du decodeur mapvar.
	a0RoleSite = "assault_bomb"
)

// TestAssautA0Qualification — la qualification d'UN film du corpus Assaut.
func TestAssautA0Qualification(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(a0FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide (identifiant court du film Assaut)", a0FilmEnv)
	}
	g := filmproc.Arm("a0-assaut", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — qualification interrompue, ce film "+
			"sort NON QUALIFIE", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio (plafond souple %d Gio)",
			id, float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()

	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache (%s=%q)", id, attFilmEnv, root)
	}

	// (1) BORNES de quantification — sans elles, le verdict tombe ici.
	wr, lay, okB := d6Bornes(t, root, id)
	if !okB {
		t.Logf("VERDICT %s : EXCLU — bornes de quantification absentes du catalogue "+
			"(film indecodable en coordonnees monde, lecon Live Fire du lot O)", id)
		return
	}

	// (2) SITES `assault_bomb` du catalogue d'objectifs — l'ancrage de A1/A3.
	sites := attMarqueurs(t, root, id, a0RoleSite)
	t.Logf("%s : %d site(s) `%s` au catalogue d'objectifs pour cette carte", id, len(sites), a0RoleSite)
	for _, s := range sites {
		t.Logf("%s :   site team_index=%d (%.2f, %.2f, %.2f)", id, s.TeamIndex,
			s.Center.X, s.Center.Y, s.Center.Z)
	}

	// (3) PONT bipede — la precondition du lot O, meme calcul que `d8Charge`.
	pos, err := d6Positions(objChunkDir(root, id), wr, lay)
	if err != nil {
		t.Fatalf("%s : positions de bipede illisibles : %v", id, err)
	}
	tracks := indexBySlot(pos)
	pont := objBridgeOf(t, root, id)
	nommes := 0
	for slot := range tracks {
		if _, ok := pont.SlotXUID[slot]; ok {
			nommes++
		}
	}
	part := float64(nommes) / math.Max(float64(len(tracks)), 1)
	t.Logf("%s : pont %d/%d slot(s) nomme(s) = %.1f %% (plancher %.0f %%)",
		id, nommes, len(tracks), 100*part, 100*d8PontMinimum)

	// (4) RELEVES A0.3 — manches reelles et increments du score de mode, dates.
	a0RelevesScore(t, id, src)

	if part < d8PontMinimum {
		t.Logf("VERDICT %s : EXCLU — pont %.1f %% sous le plancher de %.0f %%", id, 100*part,
			100*d8PontMinimum)
		return
	}
	t.Logf("VERDICT %s : bornes OK, pont %.1f %% OK, %d site(s) %s", id, 100*part, len(sites),
		a0RoleSite)
}

// a0RelevesScore publie les manches reelles et chaque increment du score de MODE par equipe.
func a0RelevesScore(t *testing.T, id string, src *objDiskFilm) {
	t.Helper()
	recs, truncated := objectiveevents.StatRecordsCtx(context.Background(), src, id)
	if truncated {
		t.Logf("%s : lecture des enregistrements TRONQUEE — les releves de score sont partiels "+
			"et cela se reporte au protocole", id)
	}
	real := objectiveevents.RealRounds(recs)
	rounds := make([]int, 0, len(real))
	for r, ok := range real {
		if ok {
			rounds = append(rounds, r)
		}
	}
	sort.Ints(rounds)
	t.Logf("%s : %d manche(s) reelle(s) %v", id, len(rounds), rounds)

	// Debut de chaque manche reelle : le premier enregistrement d'entite qui la porte.
	debut := map[int]int{}
	for _, r := range recs {
		if !real[r.Round] {
			continue
		}
		if d, ok := debut[r.Round]; !ok || r.TimeMS < d {
			debut[r.Round] = r.TimeMS
		}
	}
	for _, r := range rounds {
		t.Logf("%s : manche %d — premier enregistrement a %d ms", id, r, debut[r])
	}

	// Increments du score de mode, PAR MANCHE (la forme que l'ecran affiche) puis en cumule.
	parManche := objectiveevents.SeriesByRound(recs, objectiveevents.ModeScoreComponent, true)
	slots := make([]int, 0, len(parManche))
	for s := range parManche {
		slots = append(slots, s)
	}
	sort.Ints(slots)
	for _, s := range slots {
		for _, r := range rounds {
			for _, p := range parManche[s][r] {
				t.Logf("%s : SCOREMODE slot=%d manche=%d t=%d ms valeur=%d", id, s, r, p.TimeMS, p.Value)
			}
		}
	}
	total := objectiveevents.SeriesTotal(recs, objectiveevents.ModeScoreComponent, true)
	for _, s := range slots {
		pts := total[s]
		if len(pts) == 0 {
			continue
		}
		t.Logf("%s : SCOREMODE slot=%d cumule : %d emission(s), final=%d", id, s, len(pts),
			pts[len(pts)-1].Value)
	}
	a0RelevesScoreBrut(t, id, recs, real)
}

// a0RelevesScoreBrut publie les emissions BRUTES du score de mode des slots d'equipe, TOUTES
// manches confondues — y compris celles que `RealRounds` refuse.
//
// POURQUOI CE RELEVE EXISTE A COTE DU PRECEDENT : en One Bomb une manche se termine sur UN
// point — elle ne peut jamais porter la suite d'emissions que le critere de `RealRounds`
// exige (critere valide sur Oddball/CTF, ou une manche fantome gonflait le score de 1 a
// 2 104). Les manches reelles de One Bomb sont donc structurellement REFUSEES, et le score
// de mode filtre est PARTIEL sur ces films. Le brut ci-dessous rend chaque emission datee
// avec sa manche, dedoublonnee par valeur (une reemission de la meme valeur n'est pas un
// increment) — les parasites eventuels restent visibles, c'est un RELEVE, pas un calque.
func a0RelevesScoreBrut(t *testing.T, id string, recs []objectiveevents.StatRecord,
	real map[int]bool) {
	t.Helper()
	type cle struct{ slot, round int }
	vu := map[cle]int64{}
	// Les enregistrements sont deja tries par temps puis par slot (contrat de StatRecords).
	for _, r := range recs {
		if !objectiveevents.IsTeamSlot(r.Slot) {
			continue
		}
		v, ok := r.Comps[0]
		if !ok || v.A < 0 {
			continue
		}
		k := cle{r.Slot, r.Round}
		if prev, seen := vu[k]; seen && prev == v.A {
			continue
		}
		vu[k] = v.A
		marque := ""
		if !real[r.Round] {
			marque = " (manche REFUSEE par RealRounds)"
		}
		t.Logf("%s : SCOREMODE-BRUT slot=%d manche=%d t=%d ms valeur=%d%s",
			id, r.Slot, r.Round, r.TimeMS, v.A, marque)
	}
}
