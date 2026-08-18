package replay

// zone_state_p2a_test.go — LOT C-bis PHASE 2a : LES SEUILS, ET LE POINT D'ENTREE.
//
// PERIMETRE (arbitrage `registre_film/LOTCBIS_ARBITRAGE_PHASE1.md` §« Phase 2a ») :
//
//	CB.2a.1  appariement slot ti=13 -> zone du catalogue, par la POSITION du capteur ;
//	CB.2a.2  semantique du tag 4 (precision / rappel / hasard, puis la valeur vs l'EQUIPE) ;
//	CB.2a.3  KOTH : la zone active appariee par la GRAPPE des positions.
//
// MESURE SEULEMENT. Aucun champ de document, aucun schema, aucune publication : la forme de
// `zoneStates` est PROPOSEE dans le journal si le gate tient, elle n'est pas ecrite en code.
//
// LES SEUILS SONT ECRITS ICI, AVANT LA MESURE, et ne sont pas ajustables apres. Chaque taux est
// publie avec SON DENOMINATEUR et avec son NIVEAU DE HASARD — la lecon du lot C et de la phase 1,
// ou deux clauses de gate se sont averees vides par construction (un temoin sous la moitie du
// reel quand le hasard seul valait 57-61 % ; une fenetre couvrant 93-97 % du match).
//
// USAGE (depuis apps/go-api, UN film par processus, avant-plan) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestZoneEtatPhase2a -v -timeout 30m ./internal/analysis/replay/

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// SEUILS DE LA PHASE 2a — ecrits avant la mesure (arbitrage §Phase 2a).
const (
	// p2aFenetreMS : demi-fenetre autour d'une capture (l'arbitrage la fixe a 2 s).
	p2aFenetreMS = 2000
	// p2aDecalageMS : le TEMOIN TEMPOREL. Les memes captures, lues 20 s plus loin. Meme
	// valeur qu'en phase 1, pour que les deux mesures se comparent.
	p2aDecalageMS = 20000
	// CB.2a.1 : la carte slot -> zone doit etre coherente a ce taux sur tout le match.
	p2aSeuilCoherence = 0.90
	// CB.2a.2 : rappel >= 80 % ET precision >= 2x le hasard.
	p2aSeuilRappel      = 0.80
	p2aFacteurPrecision = 2.0
	// CB.2a.2, second volet : la valeur du tag 4 designe l'EQUIPE du capteur a ce taux.
	p2aSeuilProprietaire = 0.90
	// CB.2a.3 : les periodes KOTH attribuees doivent couvrir cette part du temps de match.
	p2aSeuilCouvertureKOTH = 0.80
	// Ce qui compte comme une RAMPE du tag 3 : trois echantillons croissants et une amplitude
	// d'au moins 4 096 quanta sur 24 bits. REPRIS TELS QUELS de la phase 1 (`ti13Ramp*`) : une
	// definition qui bougerait entre deux phases rendrait les chiffres incomparables.
	p2aRampMinSamples   = 3
	p2aRampMinAmplitude = 4096
	// Volume minimal pour qu'un slot soit juge. Sous ce seuil, un taux n'a pas de sens.
	p2aMinParSlot = 5
)

// p2aDistancesM est la COURBE de tolerance publiee par CB.2a.1, en metres. Le contrat de
// `AttributeOptions.MaxDistanceM` l'exige : « l'appelant doit publier la COURBE taux(seuil) avec
// son temoin, jamais un seuil seul ».
var p2aDistancesM = []float64{0, 2, 5, 10}

// p2aVerdictDistanceM est LE seuil auquel le verdict de CB.2a.1 se prononce, ecrit avant la
// mesure. Cinq metres, et c'est motive : mesure du chantier, a l'instant d'une action de zone,
// 100 % des joueurs sont a moins de 20 m d'une zone, mediane 6,6 m, mais seuls ~10 % sont
// DEDANS — parce que la statistique du statborg est repliquee APRES l'action, que la recompense
// d'equipe n'exige pas d'etre dans la zone, et que la forme du fichier n'est pas le volume de
// capture du jeu. Juger a 0 m rendrait un denominateur d'une poignee de captures.
//
// CE QUE CE CHOIX NE CONCEDE PAS. CB.2a.1 ne mesure pas « le joueur etait dans la zone » mais la
// COHERENCE de la carte slot -> zone : relacher la tolerance ne rend pas la carte coherente, et
// les deux temoins (permutation des slots, decalage de 20 s) restent juges au meme seuil.
const p2aVerdictDistanceM = 5.0

// p2aTemoinTranslationM : le decalage du temoin de zones translatees, en x et en y (valeur du
// releve terrain d'origine, reprise de `cmd/zone-attribution`).
const p2aTemoinTranslationM = 12.0

// TestZoneEtatPhase2a joue la mesure de la phase 2a sur UN film.
func TestZoneEtatPhase2a(t *testing.T) {
	dir := p2aRequireFilm(t)
	short, film := p2aFilmOf(t, dir)
	out := p2aOutDir(t)

	p2aCheckRegistre(t, dir)
	src := p2aSource(t, dir)
	sc := p2aScanFilm(t, dir, p2aStartMS(src))
	dureeMS := sc.t1MS - sc.t0MS
	var sbAncrage strings.Builder
	t.Logf("FILM %s (%s, %s) — %d slots dans la bande ti=13 · %d records ancres, %d rejoues"+
		" entierement, %d CHAINES (%.1f %%)", short, film.Mode, film.Carte, sc.bandeSlots,
		sc.records, sc.walked, sc.chained, 100*p2aRate(sc.chained, sc.walked))
	t.Logf("  VALEURS : i1 scalaire %d · i2..i33 par joueur %d · film de %d ms (%d -> %d)",
		len(sc.scal), len(sc.joue), dureeMS, sc.t0MS, sc.t1MS)
	t.Logf("  CHAINAGE par population : records SCALAIRES (i0/i1 seuls) %d/%d = %.1f %% ·"+
		" records PAR JOUEUR %d/%d = %.1f %% · temoin decale de 3 bits %d/%d = %.1f %%",
		sc.chainedScal, sc.walkedScal, 100*p2aRate(sc.chainedScal, sc.walkedScal),
		sc.chainedJoue, sc.walkedJoue, 100*p2aRate(sc.chainedJoue, sc.walkedJoue),
		sc.decale, sc.walked, 100*p2aRate(sc.decale, sc.walked))
	fmt.Fprintf(&sbAncrage, "# chainage\t%s\tscal\t%d\t%d\tjoueur\t%d\t%d\tdecale\t%d\t%d\n",
		short, sc.chainedScal, sc.walkedScal, sc.chainedJoue, sc.walkedJoue, sc.decale, sc.walked)

	quant := p2aQuant(t, film.Carte)
	zones := p2aZones(t, film.MapID, p2aRolesDuMode(film)...)
	doc := p2aDoc(t, dir, short, quant)
	t.Logf("  REJEU : %d trajectoires, %d frames de %d ms, origine %s · CATALOGUE : %d zones",
		len(doc.Tracks), doc.FrameCount, doc.FrameIntervalMS, p2aOrigine(doc), len(zones))

	var sb strings.Builder
	fmt.Fprintf(&sb, "# lot C-bis phase 2a — film %s (%s / %s)\n", short, film.Mode, film.Carte)
	fmt.Fprintf(&sb, "# ancrage\t%s\trecords\t%d\trejoues\t%d\tchaines\t%d\tslots_bande\t%d\n",
		short, sc.records, sc.walked, sc.chained, sc.bandeSlots)
	sb.WriteString(sbAncrage.String())

	p2aStructure(t, &sb, p2aEntree{short: short, film: film, sc: sc, doc: doc, zones: zones, src: src})
	app := p2aVoletAppariement(t, &sb, p2aEntree{short: short, film: film, sc: sc, doc: doc,
		zones: zones, src: src})
	p2aVoletTag4(t, &sb, p2aEntree{short: short, film: film, sc: sc, doc: doc, zones: zones,
		src: src}, app)
	p2aVoletKOTH(t, &sb, p2aEntree{short: short, film: film, sc: sc, doc: doc, zones: zones,
		src: src})

	p2aWrite(t, out, short+"_p2a.tsv", sb.String())
}

// p2aEntree regroupe les entrees d'un volet : sans ce regroupement, chaque volet depasserait les
// cinq parametres du seuil projet.
type p2aEntree struct {
	short string
	film  p2aFilm
	sc    *p2aScan
	doc   ReplayDocument
	zones []Zone
	src   *filmcache.Source
}

// p2aRolesDuMode rend les roles de zone a retenir pour le film. En Strongholds, le role du mode
// et lui seul ; en KOTH, l'union (le catalogue ne connait aucun role de colline).
func p2aRolesDuMode(f p2aFilm) []mapvar.Role {
	if f.Mode == "Strongholds" {
		return []mapvar.Role{mapvar.RoleStrongholdZone}
	}
	return nil
}

// p2aOrigine rend l'origine publiee du document, ou « absente ».
func p2aOrigine(doc ReplayDocument) string {
	if doc.OriginMs == nil {
		return "ABSENTE (les instants ne sont pas recalables)"
	}
	return fmt.Sprintf("%d ms", *doc.OriginMs)
}

// p2aRate rend un taux, 0 quand le denominateur est nul.
func p2aRate(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

// p2aWrite ecrit un TSV de mesure.
func p2aWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	p2aMkdir(t, dir)
	if err := os.WriteFile(dir+string(os.PathSeparator)+name, []byte(content), 0o644); err != nil {
		t.Fatalf("ecriture de %s : %v", name, err)
	}
}
