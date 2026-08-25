package replay

// inventory_mort_recouvrement_test.go — LE RECORD SANS ARME EST-IL UN MORT ?
//
// CE QU'ON MESURE, ET POURQUOI. La mesure du 2026-08-24
// (`.ai/V7.5/replay2d/MESURE_TROUS_INVENTAIRE_2026-08-24.md`) etablit que 17,4 % des lectures
// d'inventaire ne rendent RIEN, et que 96,7 % de ces lectures sont des records SANS AUCUNE
// ARME. Elle avance une interpretation — « bipede mort ou en reapparition » — en la declarant
// explicitement NON PROUVEE. Encoder une etiquette « mort » dans le contrat publie sans la
// prouver serait affirmer a l'ecran ce que personne n'a mesure.
//
// LA PREUVE DEMANDEE. Croiser chaque record VIDE avec le FIL DES MORTS (deaths_source.go) :
// l'instant du record tombe-t-il dans la fenetre [mort, reapparition] du porteur du slot ?
//
// LE TEMOIN EST LA PIECE MAITRESSE. La meme fenetre est appliquee aux records PLEINS. Sans lui,
// un taux de recouvrement eleve ne dirait rien : si un joueur meurt toutes les 20 s et que la
// fenetre en dure 10, la moitie de TOUS les records y tomberait par construction. Le rapport
// entre les deux taux est le seul chiffre qui porte une information.
//
// ZERO OCTET DE FILM. La mesure tourne sur le fixture d'entrees deja versionne
// (`golden_inputs_test.go`), qui porte pour le film de verite terrain 000d5950 les inventaires
// d'image-cle ET le fil des morts ET les positions. C'est la meme source que la production.
// L'extension a un corpus de films est gatee par INV_MORT_FILMS (ci-dessous), non jouee en CI.

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// invRespawnWindowsMS est l'echelle de fenetres balayee. La reapparition mesuree a une mediane
// de 8,0 s (cf. lives.go, lifeGapUS) ; on balaie de part et d'autre pour voir OU le signal se
// separe de son temoin, plutot que de postuler une bonne largeur.
var invRespawnWindowsMS = []int64{2_000, 4_000, 6_000, 8_000, 10_000, 12_000, 15_000, 20_000}

// invMortStat porte le decompte d'une categorie de records face au fil des morts.
type invMortStat struct {
	total     int // records de la categorie
	attribues int // records dont le slot a un porteur nomme (denominateur du taux)
	dans      map[int64]int
}

func newInvMortStat() *invMortStat { return &invMortStat{dans: map[int64]int{}} }

func (s *invMortStat) taux(w int64) float64 {
	if s.attribues == 0 {
		return 0
	}
	return 100 * float64(s.dans[w]) / float64(s.attribues)
}

// invMortMeasure croise les inventaires d'un film avec son fil des morts.
//
// LES DEUX HORLOGES. Le fil des morts est date sur l'horloge du MATCH, les records sur celle du
// FILM ; le decalage entre les deux est RESOLU par bestDeathOffset et publie par buildOwners
// (OwnerReport.DeathOffsetMS). On l'applique dans le meme sens que nameLivesByDeaths :
// instant_film = instant_match + offset.
func invMortMeasure(
	pos []filmdec.BipedPosition, fire []filmdec.FireEvent,
	inv []KeyframeInventory, deaths []Death, idx PlayerIndexTable,
) (vide, plein *invMortStat, own OwnerReport) {
	own = buildOwners(indexBySlot(pos), deaths, idx, fireRefs(fire))
	// LES MEMES PIECES QUE LA PRODUCTION : le regroupement des morts par victime et la recherche
	// de la mort qui precede viennent de inventory_dead_readings.go. Une seconde implementation
	// mesurerait autre chose que ce que le code publie.
	byXUID := deathTimesByVictimMS(deaths, own.DeathOffsetMS)
	vide, plein = newInvMortStat(), newInvMortStat()
	for _, r := range inv {
		st := plein
		if invReadingIsEmpty(r) {
			st = vide
		}
		st.total++
		x, ok := own.SlotXUID[r.Slot]
		if !ok {
			continue
		}
		st.attribues++
		delta, has := invSinceLastDeathMS(byXUID[x], int64(r.TimestampUS/1000))
		if !has {
			continue
		}
		for _, w := range invRespawnWindowsMS {
			if delta <= w {
				st.dans[w]++
			}
		}
	}
	return vide, plein, own
}

// TestInventaireRecordVideRecouvrementMorts mesure, sur le film de verite terrain, la part des
// lectures VIDES qui tombent dans une fenetre de reapparition — avec son temoin.
//
// CE QU'IL VERROUILLE. La valeur d'invDeadWindowMS et l'etiquette `dead` reposent sur ce
// recouvrement. Si un refactor du pont slot->joueur, du calage d'horloge ou du decodeur le fait
// s'effondrer, l'etiquette devient un mensonge a l'ecran : ce test tombe AVANT.
func TestInventaireRecordVideRecouvrementMorts(t *testing.T) {
	g := loadGoldenInputs(t)
	vide, plein, own := invMortMeasure(g.Positions, g.Fire, g.Inventory, g.Deaths, g.Indices)
	t.Log(invMortReport(g.Film, vide, plein, own))
	if vide.total == 0 {
		t.Fatal("aucune lecture vide dans le fixture : la mesure n'a plus de sujet")
	}
	if vide.attribues == 0 {
		t.Fatal("aucune lecture vide attribuee a un porteur : le pont slot->joueur est vide")
	}
	// Mesure du 2026-08-25 sur ce fixture : 93,8 % contre 0,7 % (137x). Les seuils sont pris
	// LARGES sous la mesure — ils gardent l'ETIQUETTE, ils ne figent pas un centieme.
	signal, temoin := vide.taux(invDeadWindowMS), plein.taux(invDeadWindowMS)
	if signal < 80 {
		t.Errorf("recouvrement des lectures vides par le fil des morts : %.1f %% a %d ms — "+
			"sous 80 %%, l'etiquette « mort » n'est plus justifiee", signal, invDeadWindowMS)
	}
	if temoin >= 10 {
		t.Errorf("temoin : %.1f %% des lectures PLEINES tombent dans la meme fenetre — "+
			"la fenetre attrape des joueurs vivants, le recouvrement ne prouve plus rien", temoin)
	}
}

// invMortReport rend le tableau de la mesure, categorie par fenetre.
func invMortReport(film string, vide, plein *invMortStat, own OwnerReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nfilm %s — decalage du fil des morts %d ms (%d appariements)\n",
		film, own.DeathOffsetMS, own.DeathOffsetMatches)
	fmt.Fprintf(&b, "lectures VIDES %d (attribuees %d) · lectures PLEINES %d (attribuees %d)\n",
		vide.total, vide.attribues, plein.total, plein.attribues)
	fmt.Fprintf(&b, "%-10s %14s %14s %8s\n", "fenetre", "vide", "plein (temoin)", "rapport")
	for _, w := range invRespawnWindowsMS {
		tv, tp := vide.taux(w), plein.taux(w)
		ratio := "n/a"
		if tp > 0 {
			ratio = fmt.Sprintf("%.1fx", tv/tp)
		}
		fmt.Fprintf(&b, "%7d ms %9d %4.1f%% %9d %4.1f%% %8s\n",
			w, vide.dans[w], tv, plein.dans[w], tp, ratio)
	}
	return b.String()
}

// TestInventaireRecordVideCorpus etend la mesure a un corpus de films du cache. GATE PAR
// INV_MORT_FILMS (liste de repertoires separes par des virgules) : il fait des I/O disque et
// decode des films entiers, ce qui n'a pas sa place dans la suite ordinaire.
//
//	INV_MORT_FILMS=<repo>/data/cache/film_chunks/000d5950,<...> \
//	  go test ./internal/analysis/replay/ -run RecordVideCorpus -v
func TestInventaireRecordVideCorpus(t *testing.T) {
	raw := os.Getenv("INV_MORT_FILMS")
	if raw == "" {
		t.Skip("corpus : INV_MORT_FILMS non defini")
	}
	totalVide, totalPlein := newInvMortStat(), newInvMortStat()
	for _, dir := range strings.Split(raw, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		vide, plein, own := invMortFilm(t, dir)
		if vide == nil {
			continue
		}
		t.Log(invMortReport(dir, vide, plein, own))
		invMortAccumulate(totalVide, vide)
		invMortAccumulate(totalPlein, plein)
	}
	t.Log(invMortReport("TOTAL corpus", totalVide, totalPlein, OwnerReport{}))
	// LE CORPUS CONCLUT, IL NE SE CONTENTE PAS DE PUBLIER. Sans assertion, une degradation du
	// pont d'identite ou du calage d'horloge passait en silence : le tableau s'affichait, personne
	// ne le lisait, et l'etiquette « mort » restait posee a l'ecran sur une mesure qui ne la
	// portait plus.
	//
	// LES SEUILS SONT CEUX DU TEST DE VERITE TERRAIN, DESSERRES D'UN CRAN. Le corpus du lot 1
	// (8 films, 1 419 records) donne 88,3 % de signal pour 1,1 % de temoin ; la mesure sur le seul
	// fixture donne 93,8 % / 0,7 %. Un corpus melange des cartes et des modes, donc il varie plus
	// qu'un film : signal >= 75 (au lieu de 80) laisse ~13 points sous la mesure, temoin <= 10
	// reste la meme borne — c'est le RAPPORT (82x mesure) que ces deux bornes protegent, pas un
	// centieme.
	if totalVide.total == 0 {
		t.Fatal("corpus sans aucune lecture vide : la mesure n'a plus de sujet")
	}
	if totalVide.attribues == 0 {
		t.Fatal("corpus sans aucune lecture vide attribuee : le pont slot->joueur est vide")
	}
	signal, temoin := totalVide.taux(invDeadWindowMS), totalPlein.taux(invDeadWindowMS)
	if signal < corpusSignalMin {
		t.Errorf("corpus : recouvrement des lectures vides %.1f %% a %d ms — sous %.0f %%, "+
			"l'etiquette « mort » n'est plus justifiee", signal, invDeadWindowMS, corpusSignalMin)
	}
	if temoin > corpusTemoinMax {
		t.Errorf("corpus : temoin %.1f %% des lectures PLEINES dans la meme fenetre — au-dela de "+
			"%.0f %%, la fenetre attrape des vivants et le recouvrement ne prouve plus rien",
			temoin, corpusTemoinMax)
	}
}

// Les seuils du corpus, ecrits ici pour qu'ils se lisent d'un coup et se comparent a ceux du test
// de verite terrain (80 / 10, cf. TestInventaireRecordVideRecouvrementMorts).
const (
	corpusSignalMin = 75.0
	corpusTemoinMax = 10.0
)

func invMortAccumulate(dst, src *invMortStat) {
	dst.total += src.total
	dst.attribues += src.attribues
	for w, n := range src.dans {
		dst.dans[w] += n
	}
}

// invMortFilm rejoue sur un film du cache la MEME sequence de decodage que la production, pour
// les seules entrees que la mesure consomme.
func invMortFilm(t *testing.T, dir string) (*invMortStat, *invMortStat, OwnerReport) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	scan := filmdec.DefaultScanFilmOptions()
	// QUANTA SEULS, et c'est suffisant : la mesure ne lit d'une position que son SLOT et son
	// HORODATAGE (indexBySlot, buildLifeSpans). Exiger les bornes de carte n'ajouterait aucune
	// information et bornerait le corpus aux seules cartes du catalogue.
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Logf("%s : positions illisibles : %v", dir, err)
		return nil, nil, OwnerReport{}
	}
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		// LA SEULE BRANCHE QUI CONTINUE SANS SES PIECES, et elle doit le DIRE. Les tirs
		// alimentent `fireRefs` -> `buildOwners` : sans eux le pont slot->joueur est plus
		// pauvre, `attribues` baisse, et le DENOMINATEUR du taux change sans qu'aucune ligne
		// du rapport ne l'explique. La mesure reste jouable, elle n'est plus comparable.
		t.Logf("%s : events de tir illisibles : %v — pont slot->joueur reduit aux morts, "+
			"denominateur du taux affaibli", dir, err)
		fire = nil
	}
	inv, _, err := ScanFilmKeyframeInventory(dir, loadoutFamilies(), 0)
	if err != nil {
		t.Logf("%s : inventaire illisible : %v", dir, err)
		return nil, nil, OwnerReport{}
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Logf("%s : fil des morts illisible : %v", dir, err)
		return nil, nil, OwnerReport{}
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Logf("%s : index de joueur illisible : %v", dir, err)
		return nil, nil, OwnerReport{}
	}
	table, _ := injectiveOrEmpty(idx)
	vide, plein, own := invMortMeasure(pos, fire, inv, deaths, table)
	return vide, plein, own
}
