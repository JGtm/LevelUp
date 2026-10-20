//go:build research

package grammar

// mouvement_5_9_research_test.go — LA MESURE DE VERIFICATION DU LOT 5.9.
//
// # DEUX QUESTIONS, ET ELLES NE SE MELANGENT PAS
//
//	(A) LE SAUT DERIVE, TEL QUE LA PRODUCTION LE PUBLIE. On appelle `ScanMovementStates` — la
//	    fonction de production, pas une copie — et on confronte ce qu elle derive a l ORACLE
//	    PHYSIQUE du lot 5.7.5, qui integre la meme vitesse sur la passe de recherche. Les deux
//	    marchent le film par des chemins differents (la production sous la porte de speculation
//	    du 5.7.4, l oracle sous la passe partagee `m57Passe`) : leur ACCORD est ce qui dit que le
//	    port est fidele, et leur ECART dit ou il ne l est pas.
//
//	(B) `i57`, LE CHAMP DU SPRINT. La chaine du 5.9.1 etablit que l etiquette d `i57` est
//	    l INDEX DE LA FENTE DE CAPACITE ACTIVE (-1 = aucune, 0..2 = la fente). Reste a savoir si
//	    ce champ est assez DENSE, sur la population retenue, pour qu un etat de sprint par
//	    instant soit publiable. C est une mesure de DENOMINATEUR, pas un score : tant que la
//	    fente de `'sasp'` n est pas nommee, aucun etat ne se publie.
//
// Rejouable (memes variables que `TestMouvement57Posture`) :
//
//	MOUV57_FILM=<dir> MOUV57_CARTE=<carte> MOUV57_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement59' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// m59FenetreUS est la tolerance d appariement entre un saut derive par la PRODUCTION et un
// episode de l ORACLE, en microsecondes. Un delta de 60 Hz dure 16 667 us ; deux ticks valent
// 33 333 us. C est la fenetre du protocole de preuve du lot 5.7, reprise telle quelle.
const m59FenetreUS = 40000

// TestMouvement59SautDerive confronte le saut publie par la production a l oracle physique.
func TestMouvement59SautDerive(t *testing.T) {
	lus, st := m59Production(t)
	t.Logf("PRODUCTION (`ScanMovementStates`) : records ti=35 %d (desyncs %d) · "+
		"lectures de vitesse RETENUES %d · montees FERMEES %d · sauts RETENUS %d (%.1f %% des "+
		"montees) · lectures publiees %d · largeurs de carte %v",
		st.Records, st.Desyncs, st.VelocityReads, st.JumpEpisodes, st.JumpsDerived,
		m533bPart(st.JumpsDerived, st.JumpEpisodes), st.Read, st.MapWidths)
	m59ParGenre(t, lus)

	rec, ok := m57PasseRetenue(t)
	if !ok {
		return
	}
	oracle := m59OracleRetenus(rec)
	prod := m59AmorcesDerivees(lus)
	t.Logf("ORACLE (passe de recherche, memes constantes) : %d episodes aeriens fermes ou "+
		"ouverts, dont %d dans la fenetre de H", len(m575Episodes(rec)), len(oracle))
	m59Accord(t, prod, oracle)
}

// TestMouvement59DensiteI57 publie la densite d `i57` sur la population RETENUE — le
// denominateur du sprint.
func TestMouvement59DensiteI57(t *testing.T) {
	rec, cands, ok := m575PasseCandidats(t)
	if !ok {
		return
	}
	t.Logf("POPULATION RETENUE : %d records ti=35, %d desynchronises", rec.ti35, rec.desync)
	var cles []string
	for nom := range cands {
		if strings.HasPrefix(nom, "i57.") {
			cles = append(cles, nom)
		}
	}
	sort.Strings(cles)
	if len(cles) == 0 {
		t.Logf("i57 : AUCUNE observation sur la population retenue — le composant ne voyage pas "+
			"assez pour porter un etat par instant (records %d)", rec.ti35)
		return
	}
	for _, nom := range cles {
		t.Logf("  %-22s : %5d instants (%.3f %% des records ti=35)",
			nom, len(cands[nom]), m533bPart(len(cands[nom]), rec.ti35))
	}
	t.Logf("LECTURE : l etiquette d `i57` est l INDEX DE LA FENTE DE CAPACITE ACTIVE (lot 5.9.1, " +
		"`FUN_14319db80`). Tant que la fente qui porte `'sasp'` n est pas nommee, aucun etat de " +
		"sprint ne se publie — et ces densites disent ce qu il y aurait a publier.")
}

// m59Production ouvre le film sous sa carte et appelle la fonction de PRODUCTION.
func m59Production(t *testing.T) ([]types.MovementStateRead, types.MovementStateStats) {
	t.Helper()
	dir := os.Getenv("MOUV57_FILM")
	if dir == "" {
		t.Skip("MOUV57_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m57Contexte(t, film)
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	lus, st, errS := ScanMovementStates(fc)
	if errS != nil {
		t.Fatalf("ScanMovementStates : %v", errS)
	}
	return lus, st
}

// m59ParGenre publie le compte des lectures et des vies par genre publie.
func m59ParGenre(t *testing.T, lus []types.MovementStateRead) {
	t.Helper()
	parGenre := map[string]int{}
	vies := map[string]map[uint32]struct{}{}
	for _, r := range lus {
		parGenre[r.Kind]++
		if vies[r.Kind] == nil {
			vies[r.Kind] = map[uint32]struct{}{}
		}
		vies[r.Kind][r.Slot] = struct{}{}
	}
	var genres []string
	for k := range parGenre {
		genres = append(genres, k)
	}
	sort.Strings(genres)
	for _, k := range genres {
		t.Logf("  %-12s : %5d lectures · %3d vies", k, parGenre[k], len(vies[k]))
	}
}

// m59AmorcesDerivees rend les instants d AMORCE des sauts publies par la production.
func m59AmorcesDerivees(lus []types.MovementStateRead) []m59Instant {
	var out []m59Instant
	for _, r := range lus {
		if r.Kind == types.MovementJumpDerived && r.On {
			out = append(out, m59Instant{slot: r.Slot, ts: r.TimestampUS})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out
}

// m59OracleRetenus rend les amorces des episodes de l oracle qui tombent dans la fenetre de H.
func m59OracleRetenus(rec *m57Rec) []m59Instant {
	var out []m59Instant
	for _, e := range m575Episodes(rec) {
		if !hauteurDeSaut(e.hauteur) {
			continue
		}
		out = append(out, m59Instant{slot: e.slot, ts: e.t0})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out
}

// m59Instant est UN instant attribue a une vie.
type m59Instant struct {
	slot uint32
	ts   uint64
}

// m59Accord apparie les deux listes a `m59FenetreUS` pres et publie precision et rappel.
//
// UN APPARIEMENT, PAS UNE EGALITE : les deux marches ne datent pas un paquet exactement pareil
// (la production tire ses horodatages sous la porte de speculation, l oracle sous la passe
// partagee). Exiger l egalite a la microseconde mesurerait la gigue, pas l accord.
func m59Accord(t *testing.T, prod, oracle []m59Instant) {
	t.Helper()
	if len(prod) == 0 && len(oracle) == 0 {
		t.Logf("ACCORD : les deux listes sont VIDES — aucun saut a la hauteur du Spartan sur ce film")
		return
	}
	pris := make([]bool, len(oracle))
	var bons int
	for _, p := range prod {
		for i, o := range oracle {
			if pris[i] || o.slot != p.slot || m575Proximite(o.ts, p.ts) > m59FenetreUS {
				continue
			}
			pris[i], bons = true, bons+1
			break
		}
	}
	var couverts int
	for _, v := range pris {
		if v {
			couverts++
		}
	}
	t.Logf("ACCORD PRODUCTION <-> ORACLE (fenetre %d us) : %d sauts publies, %d apparies "+
		"(precision %.1f %%) · %d episodes de l oracle, %d couverts (rappel %.1f %%)",
		m59FenetreUS, len(prod), bons, m533bPart(bons, len(prod)),
		len(oracle), couverts, m533bPart(couverts, len(oracle)))
	if len(prod) != bons || len(oracle) != couverts {
		t.Logf("  ECART : %s", m59Ecart(prod, oracle, pris))
	}
}

// m59Ecart decrit le premier desaccord, pour que le chiffre ait une cause nommee.
func m59Ecart(prod, oracle []m59Instant, pris []bool) string {
	var manques []string
	for i, o := range oracle {
		if !pris[i] {
			manques = append(manques, fmt.Sprintf("oracle slot %d a %d us", o.slot, o.ts))
			if len(manques) == 3 {
				break
			}
		}
	}
	if len(manques) == 0 {
		return fmt.Sprintf("%d saut(s) publie(s) sans episode d oracle en face", len(prod))
	}
	return strings.Join(manques, " · ")
}
