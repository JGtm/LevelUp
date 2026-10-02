//go:build research

package killcollector

// emprise_v1_temoin_research_test.go — LOT V1.3 DU PLAN `.ai/PLAN_EMPRISE_VIES_2026-09-28.md` :
// LE TÉMOIN DU CALCUL PUR `replay.PlacementDesVies` SUR DE VRAIS FILMS.
//
// # NOURRI COMME LE COLLECTEUR LE NOURRIRA
//
// Le registre et les positions sont ceux de la passe de positions de production
// (`buildPositionRows`, `positions.go`), les morts celles que le journal ÉCRIT — la base crédit
// relue (`persist.CreditBaseForMatch`) puis fusionnée avec la passe du film
// (`persist.MergeCreditAndFilm`), exactement la liste `fusionnees` que `collect` passe aux faits
// d'isolement —, la publiabilité celle de la passe fusionnée, les camps ceux du roster
// (`equipesNumeriques`, `isolation_facts.go`). Rien n'est écrit : la base est une COPIE ouverte
// en lecture seule, aucun persister n'est appelé.
//
// DEUX ENTRÉES NE SONT PAS NOURRIES, ET AUCUN CRITÈRE NE LES LIT : les intervalles de port (les
// films témoins sont du Team Slayer — la garde de mode des porteurs, lot V1.4, n'y lit rien) et
// la portée du radar (injectée au collecteur en V2.5 ; elle ne touche que `HorsRadarMS`).
//
// # LES TROIS CRITÈRES DU PLAN
//
//	1. le nombre de vies égale celui de `match_lives_latest` pour le match ;
//	2. au moins 97 % des frags publiables sont rattachés à une vie — dénominateur : les frags
//	   d'une passe publiable, tueur résolu, tueur et victime de camps CONNUS et différents (les
//	   autres sont écartés par règle et comptés à part, jamais « non rattachés ») ;
//	3. pour les vies finies par une mort, la distance du DERNIER INSTANT MESURÉ est à moins de
//	   1 m de celle du contexte de mort (`match_death_context_latest`) sur au moins 90 % d'entre
//	   elles — dénominateur : les vies finies par une mort qui ont un instant mesuré ET un
//	   contexte de mort à distance, joint par (victime, instant à ±150 ms de la fin de vie, la
//	   fenêtre d'appariement des morts `deathMatchWindowMS`). Les vies sans l'un ou l'autre sont
//	   comptées et imprimées.
//
// SANS SES VARIABLES, IL SE SAUTE (films et base ne sont pas versionnés) :
//
//	EMPRISE_V1_FILMS=<match_id,...> EMPRISE_V1_DB=<copie shared> EMPRISE_V1_CACHE=<racine data/cache> \
//	  go test ./internal/sync/killcollector/ -run '^TestEmpriseV1Temoin$' -v -count=1

import (
	"context"
	"database/sql"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// Les seuils du plan (V1.3), et la fenêtre de jointure des contextes de mort.
const (
	v1SeuilFragsRattaches   = 0.97
	v1SeuilFinsConformes    = 0.90
	v1EcartMaxDistanceM     = 1.0
	v1FenetreDeJointureMs   = 150 // `replay.deathMatchWindowMS`, la fenêtre d'appariement des morts
	v1RequeteViesDuMatch    = `SELECT count(*) FROM match_lives_latest WHERE match_id = ?`
	v1RequeteContextesMorts = `SELECT victim_xuid, time_ms, nearest_teammate_m
		FROM match_death_context_latest WHERE match_id = ? AND nearest_teammate_m IS NOT NULL`
)

func TestEmpriseV1Temoin(t *testing.T) {
	var films []string
	for f := range strings.SplitSeq(os.Getenv("EMPRISE_V1_FILMS"), ",") {
		if f = strings.TrimSpace(f); f != "" {
			films = append(films, f)
		}
	}
	db, cache := os.Getenv("EMPRISE_V1_DB"), os.Getenv("EMPRISE_V1_CACHE")
	if len(films) == 0 || db == "" || cache == "" {
		t.Skip("EMPRISE_V1_FILMS, EMPRISE_V1_DB et EMPRISE_V1_CACHE requis : temoin saute")
	}
	lecture := v0OuvrirCopie(t, db)
	// L'ECRITURE N'EST JAMAIS APPELEE : le collecteur recoit la copie en lecture seule, qu'une
	// ecriture accidentelle ferait echouer au lieu de la laisser passer.
	col := v0Collecteur(t, v0Env{cac: cache}, lecture, lecture)
	ctx := context.Background()
	for _, id := range films {
		p := v1RejouerLaPasse(t, ctx, col, lecture, id)
		vies, bilan := replay.PlacementDesVies(replay.EntreePlacement{
			Positions: p.mat.positions,
			Registre:  p.mat.registre,
			Equipes:   equipesNumeriques(p.ids.Equipes),
			Journal:   journalDuPlacement(p.fusionne),
		})
		v1CritereDesVies(t, ctx, lecture, id, vies)
		v1CritereDesFrags(t, id, bilan)
		v1CritereDesFins(t, ctx, lecture, id, p.mat.registre.ViesNommees(), vies)
	}
}

// v1CritereDesVies : critère 1.
func v1CritereDesVies(t *testing.T, ctx context.Context, db *sql.DB, id string, vies []replay.PlacementVie) {
	t.Helper()
	var enBase int
	if err := db.QueryRowContext(ctx, v1RequeteViesDuMatch, id).Scan(&enBase); err != nil {
		t.Fatalf("%s : match_lives_latest : %v", id, err)
	}
	mesurees := 0
	var duree, mesure, porteur, aTerre, nonSitue, coequipier int64
	for _, v := range vies {
		if v.MedianeM != nil {
			mesurees++
		}
		duree += v.DureeMS
		mesure, porteur, aTerre = mesure+v.MesureMS, porteur+v.PorteurMS, aTerre+v.EquipeATerreMS
		nonSitue, coequipier = nonSitue+v.NonSitueMS, coequipier+v.CoequipierNonSitueMS
	}
	t.Logf("%s  critere 1 : %d vies calculees, %d en base (match_lives_latest) — %d avec mediane ; "+
		"duree %d ms, mesure %d, porteur %d, equipe a terre %d, non situe %d, coequipier non situe %d",
		id[:8], len(vies), enBase, mesurees, duree, mesure, porteur, aTerre, nonSitue, coequipier)
	if len(vies) != enBase {
		t.Errorf("%s : %d vies calculees contre %d en base", id[:8], len(vies), enBase)
	}
}

// v1CritereDesFrags : critère 2.
func v1CritereDesFrags(t *testing.T, id string, b replay.BilanPlacement) {
	t.Helper()
	eligibles := b.FragsRattaches + b.FragsAvantLaPremiereVie + b.FragsSansVie
	part := 1.0
	if eligibles > 0 {
		part = float64(b.FragsRattaches) / float64(eligibles)
	}
	t.Logf("%s  critere 2 : %d frags rattaches sur %d eligibles (%.2f %%) — avant la premiere vie %d, "+
		"sans vie %d ; ecartes par regle : non publiables %d, tueur inconnu %d, camp inconnu %d, "+
		"trahisons %d", id[:8], b.FragsRattaches, eligibles, 100*part, b.FragsAvantLaPremiereVie,
		b.FragsSansVie, b.FragsNonPubliables, b.FragsTueurInconnu, b.FragsCampInconnu, b.Trahisons)
	if part < v1SeuilFragsRattaches {
		t.Errorf("%s : %.2f %% des frags publiables rattaches, seuil %.0f %%", id[:8], 100*part,
			100*v1SeuilFragsRattaches)
	}
}

// v1Contexte : un contexte de mort de la base.
type v1Contexte struct {
	tMS int64
	d   float64
}

// v1CritereDesFins : critère 3. `nommees` et `vies` sont dans le même ordre (`ViesNommees`).
func v1CritereDesFins(t *testing.T, ctx context.Context, db *sql.DB, id string,
	nommees []replay.VieNommee, vies []replay.PlacementVie) {
	t.Helper()
	ctxs := v1Contextes(t, ctx, db, id)
	var parMort, sansMesure, sansContexte, comparees, conformes int
	ecartMax := 0.0
	for i := range vies {
		if i >= len(nommees) || nommees[i].Cause != replay.CauseVieMort {
			continue
		}
		parMort++
		if vies[i].DerniereMesure == nil {
			sansMesure++
			continue
		}
		c, ok := v1ContexteDeLaFin(ctxs[vies[i].XUID], vies[i].FinMS)
		if !ok {
			sansContexte++
			continue
		}
		comparees++
		ecart := math.Abs(vies[i].DerniereMesure.DistanceM - c.d)
		ecartMax = math.Max(ecartMax, ecart)
		if ecart < v1EcartMaxDistanceM {
			conformes++
		}
	}
	part := 1.0
	if comparees > 0 {
		part = float64(conformes) / float64(comparees)
	}
	t.Logf("%s  critere 3 : %d vies finies par une mort — %d comparees, %d a moins de 1 m (%.2f %%) ; "+
		"ecart max %.2f m ; sans instant mesure %d, sans contexte a distance %d", id[:8], parMort, comparees,
		conformes, 100*part, ecartMax, sansMesure, sansContexte)
	if part < v1SeuilFinsConformes {
		t.Errorf("%s : %.2f %% des fins de vie a moins de 1 m du contexte de mort, seuil %.0f %%",
			id[:8], 100*part, 100*v1SeuilFinsConformes)
	}
}

// v1Contextes lit les contextes de mort à distance du match, par victime.
func v1Contextes(t *testing.T, ctx context.Context, db *sql.DB, id string) map[uint64][]v1Contexte {
	t.Helper()
	rows, err := db.QueryContext(ctx, v1RequeteContextesMorts, id)
	if err != nil {
		t.Fatalf("%s : match_death_context_latest : %v", id, err)
	}
	defer func() { _ = rows.Close() }()
	out := map[uint64][]v1Contexte{}
	for rows.Next() {
		var x string
		var c v1Contexte
		if err := rows.Scan(&x, &c.tMS, &c.d); err != nil {
			t.Fatalf("%s : lecture d'un contexte : %v", id, err)
		}
		if v, err := strconv.ParseUint(x, 10, 64); err == nil {
			out[v] = append(out[v], c)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("%s : contextes : %v", id, err)
	}
	return out
}

// v1ContexteDeLaFin rend le contexte le plus proche de la fin de vie, dans la fenêtre.
func v1ContexteDeLaFin(ctxs []v1Contexte, finMS int64) (v1Contexte, bool) {
	best, trouve := v1Contexte{}, false
	for _, c := range ctxs {
		ecart := c.tMS - finMS
		if ecart < 0 {
			ecart = -ecart
		}
		if ecart > v1FenetreDeJointureMs {
			continue
		}
		if !trouve || math.Abs(float64(c.tMS-finMS)) < math.Abs(float64(best.tMS-finMS)) {
			best, trouve = c, true
		}
	}
	return best, trouve
}
