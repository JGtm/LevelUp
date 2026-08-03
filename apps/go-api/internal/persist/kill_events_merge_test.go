package persist

// kill_events_merge_test.go — CE QUE LA FUSION DOIT TENIR, ET CE QU ELLE NE DOIT JAMAIS FABRIQUER.
//
// Chaque test correspond a une propriete du document de conception
// (`.ai/CONCEPTION_INVERSION_PRESEANCE.md`, §2 a §4) et a une MUTATION qui doit le faire rougir :
//
//	la base ne perd jamais une mort            retirer l ajout de la mort de base
//	l enrichissement est recopie               retirer la recopie de `source_tag`
//	les trois etats de l assistant survivent   defauter `assist_known` a TRUE
//	les orphelins sont conserves               retirer l ajout des orphelins
//	la clef est EXACTE                         elargir l appariement a une tolerance
//	l identite est un CONTROLE                 retirer la verification de concordance
//	l instant ambigu ne choisit pas            relacher `n == 1` en `n >= 1`
//
// Le fusionneur est PUR : ces tests n ouvrent aucune base et ne touchent a aucun compteur.

import (
	"testing"

	"levelup/go-api/internal/domain/killscope"
)

// mortCredit : une mort de la base credit — identites resolues, assistant INCONNU, aucune mesure
// de source. C est exactement ce que produisent les trois producteurs credit.
func mortCredit(t int) KillEventInsert {
	return KillEventInsert{
		TimeMS:             t,
		VictimGamertag:     "Victime",
		VictimXUID:         "xuid(2)",
		FeedKillerGamertag: "Tueur",
		FeedKillerXUID:     "xuid(1)",
		FeedPresent:        true,
		AssistKnown:        false,
		ReadPath:           killscope.ReadPathLiveFeed,
		ReadOrigin:         killscope.OriginCreditOnly,
	}
}

// mortFilm : une ligne de film — l arme mesuree, l assistant NOMME, et AUCUN xuid (le film ne
// porte que des noms cote replication).
func mortFilm(t int) KillEventInsert {
	pct := uint8(80)
	return KillEventInsert{
		TimeMS:             t,
		VictimGamertag:     "Victime",
		FeedKillerGamertag: "Tueur",
		FeedPresent:        true,
		AssistGamertag:     "Assistant",
		AssistXUID:         "xuid(3)",
		AssistKnown:        true,
		SourceTag:          0x6a707421,
		SourceCategory:     "Headshot",
		KillerDamagePct:    &pct,
		Diverges:           true,
		ReadPath:           killscope.ReadPathFilmWalk,
		ReadOrigin:         "credit-concordant",
	}
}

func batchCredit(deaths ...KillEventInsert) KillSourceBatch {
	return KillSourceBatch{
		MatchID: "m", DecoderRev: "credit-rev", Publishable: true,
		CreditBaseCount: len(deaths), Deaths: deaths,
	}
}

func batchFilm(deaths ...KillEventInsert) KillSourceBatch {
	return KillSourceBatch{
		MatchID: "m", DecoderRev: "film-rev", Publishable: true, Deaths: deaths,
	}
}

// TestFusionEnrichitSansRienPerdre — LE CAS NOMINAL, et les deux moities du contrat.
//
// La mort appariee garde son IDENTITE du credit (le film ne resout pas de xuid) et recoit
// l ENRICHISSEMENT du film. La mort de credit non couverte par le film reste intacte : c est
// elle, multipliee par 25 073, qui manquait a la bascule de la session 3.
func TestFusionEnrichitSansRienPerdre(t *testing.T) {
	base := batchCredit(mortCredit(1000), mortCredit(2000))
	out, st, err := MergeCreditAndFilm(base, batchFilm(mortFilm(1000)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}

	if len(out.Deaths) != 2 {
		t.Fatalf("%d morts publiees pour une base de 2 — la fusion ne doit JAMAIS en retirer",
			len(out.Deaths))
	}
	if st.Enriched != 1 || st.Orphans != 0 {
		t.Errorf("stats = %d enrichies / %d orphelins, attendu 1 / 0", st.Enriched, st.Orphans)
	}

	e := out.Deaths[0]
	if e.SourceTag != 0x6a707421 || e.SourceCategory != "Headshot" || !e.Diverges {
		t.Errorf("l enrichissement n a pas ete recopie (tag=%#x cat=%q div=%v) — la mort perd "+
			"l arme que le film avait mesuree", e.SourceTag, e.SourceCategory, e.Diverges)
	}
	if e.VictimXUID != "xuid(2)" || e.FeedKillerXUID != "xuid(1)" {
		t.Errorf("identites = %q / %q, attendu celles du CREDIT — le film ne les resout pas, et "+
			"recopier son absence rendrait la ligne injoignable par un agregat carriere",
			e.VictimXUID, e.FeedKillerXUID)
	}
	if e.ReadPath != killscope.ReadPathFilmWalk {
		t.Errorf("read_path = %q, attendu celui du film — la portee reste PAR LIGNE, c est elle "+
			"qui dit mort par mort si l arme a ete mesuree", e.ReadPath)
	}

	nonEnrichie := out.Deaths[1]
	if nonEnrichie.SourceTag != 0 || nonEnrichie.SourceCategory != "" || nonEnrichie.Diverges {
		t.Errorf("la mort NON couverte par le film porte une source (tag=%#x cat=%q div=%v) — "+
			"une absence de mesure ne doit pas devenir une mesure",
			nonEnrichie.SourceTag, nonEnrichie.SourceCategory, nonEnrichie.Diverges)
	}
	if out.CreditBaseCount != 2 {
		t.Errorf("CreditBaseCount = %d, attendu 2 — le plancher de la passe se perd, donc le "+
			"persister ne peut plus refuser une passe appauvrissante", out.CreditBaseCount)
	}
}

// TestFusionPreserveLesTroisEtatsDeLAssistant — LA COMBINAISON INTERDITE.
//
// `assist_known = FALSE` veut dire ON NE SAIT PAS ; `TRUE` + gamertag NULL veut dire « mesure :
// PAS d assistant ». Les confondre fabriquerait 60 297 « pas d assistant » jamais observes.
//
// Les trois etats sont testes ensemble parce que c est leur COEXISTENCE qui est la propriete :
// une mort non enrichie reste en etat 1, une mort enrichie par un film MUET reste en etat 1 (le
// film est l unique autorite sur ce qu il a observe), une mort enrichie par un film qui a mesure
// « pas d assistant » passe en etat 2.
func TestFusionPreserveLesTroisEtatsDeLAssistant(t *testing.T) {
	filmMuet := mortFilm(2000)
	filmMuet.AssistKnown = false
	filmMuet.AssistGamertag, filmMuet.AssistXUID = "", ""

	filmSansAssistant := mortFilm(3000)
	filmSansAssistant.AssistKnown = true
	filmSansAssistant.AssistGamertag, filmSansAssistant.AssistXUID = "", ""

	base := batchCredit(mortCredit(1000), mortCredit(2000), mortCredit(3000), mortCredit(4000))
	out, _, err := MergeCreditAndFilm(base, batchFilm(mortFilm(1000), filmMuet, filmSansAssistant))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}

	cas := []struct {
		quoi        string
		i           int
		known       bool
		nom         string
		pourquoiPas string
	}{
		{"assistant NOMME", 0, true, "Assistant", "le nom mesure par le film est perdu"},
		{"film MUET", 1, false, "", "un film muet devient « pas d assistant » : fait fabrique"},
		{"film : PAS d assistant", 2, true, "", "la mesure « pas d assistant » devient « on ne sait pas »"},
		{"non enrichie", 3, false, "", "une mort sans film devient « pas d assistant » : fait fabrique"},
	}
	for _, c := range cas {
		got := out.Deaths[c.i]
		if got.AssistKnown != c.known || got.AssistGamertag != c.nom {
			t.Errorf("%s : assist_known=%v gamertag=%q, attendu %v/%q — %s",
				c.quoi, got.AssistKnown, got.AssistGamertag, c.known, c.nom, c.pourquoiPas)
		}
	}
}

// TestFusionConserveLesOrphelins — 968 MORTS DE BOT SUR 980.
//
// Le kill-feed de l API est HUMAIN SEUL : un bot n a pas de xuid, sa mort ne produit aucun
// evenement, donc aucun couple, donc aucune ligne de credit. Rejeter les orphelins reviendrait a
// traiter une absence de mesure comme une mesure d absence.
//
// Le compteur DEDIE ne compte que la population humain-contre-humain (13 lignes) : c est la seule
// des trois dont le mecanisme n est pas demontre.
func TestFusionConserveLesOrphelins(t *testing.T) {
	bot := mortFilm(5000) // ni xuid de victime ni xuid de tueur : une mort de bot
	humain := mortFilm(6000)
	humain.VictimXUID, humain.FeedKillerXUID = "xuid(9)", "xuid(8)"

	out, st, err := MergeCreditAndFilm(batchCredit(mortCredit(1000)), batchFilm(bot, humain))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}

	if len(out.Deaths) != 3 {
		t.Fatalf("%d morts publiees, attendu 3 (1 de credit + 2 orphelines) — un orphelin rejete "+
			"est une mort de bot MESUREE que l on jette", len(out.Deaths))
	}
	if st.Orphans != 2 {
		t.Errorf("orphelins = %d, attendu 2", st.Orphans)
	}
	if st.OrphansHumanVsHuman != 1 {
		t.Errorf("orphelins humain-contre-humain = %d, attendu 1 — le compteur DEDIE ne doit "+
			"compter que la population dont le mecanisme n est pas demontre",
			st.OrphansHumanVsHuman)
	}
}

// TestFusionNAppariePasHorsDeLInstantExact — LA TOLERANCE EST ZERO, ET C EST MESURE.
//
// Les deux cotes sont le MEME flux lu par le MEME parseur : il n existe pas de population de
// morts « decalees de quelques millisecondes ». Toute tolerance non nulle achete 8 appariements
// sur 74 569 au prix de l unicite — et l unicite est ce qui rend l enrichissement sur.
func TestFusionNAppariePasHorsDeLInstantExact(t *testing.T) {
	out, st, err := MergeCreditAndFilm(batchCredit(mortCredit(1000)), batchFilm(mortFilm(1001)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if st.Enriched != 0 {
		t.Errorf("%d enrichissement(s) a t+1 ms — l appariement est devenu tolerant, donc il peut "+
			"attribuer l arme d une mort a une autre", st.Enriched)
	}
	if out.Deaths[0].SourceTag != 0 {
		t.Error("la mort de credit a recu une source venue d un AUTRE instant")
	}
	if len(out.Deaths) != 2 || st.Orphans != 1 {
		t.Errorf("%d morts / %d orphelins, attendu 2 / 1 — la ligne de film non appariee est une "+
			"mesure, elle se conserve", len(out.Deaths), st.Orphans)
	}
}

// TestFusionRejetteUneIdentiteDivergente — L IDENTITE EST UN CONTROLE.
//
// 0 divergence sur 73 589 lignes appariees. Une divergence signifierait que la clef
// `(match_id, time_ms)` a apparie DEUX MORTS DIFFERENTES — c est-a-dire que la propriete sur
// laquelle repose toute la fusion est fausse. Elle doit echouer bruyamment, pas s ecarter.
func TestFusionRejetteUneIdentiteDivergente(t *testing.T) {
	film := mortFilm(1000)
	film.VictimXUID = "xuid(999)"

	if _, _, err := MergeCreditAndFilm(batchCredit(mortCredit(1000)), batchFilm(film)); err == nil {
		t.Fatal("aucune erreur sur une victime divergente — la fusion aurait attribue a une mort " +
			"l arme mesuree sur la mort d un AUTRE joueur, sans rien signaler")
	}

	// L ABSENCE n est pas une divergence : c est le cas normal des 631 victimes et 754 tueurs
	// que le film ne resout pas, et c est la population que la clef courte existe pour garder.
	if _, st, err := MergeCreditAndFilm(batchCredit(mortCredit(1000)),
		batchFilm(mortFilm(1000))); err != nil || st.Enriched != 1 {
		t.Errorf("un xuid ABSENT cote film a ete traite comme une divergence (err=%v, enrichies=%d)",
			err, st.Enriched)
	}
}

// TestFusionRefuseDeChoisirSurUnInstantAmbigu — REJETER, PAS TIRER AU SORT.
//
// L unicite de `(match_id, time_ms)` est une propriete MESUREE du corpus Halo Infinite, pas une
// garantie de schema : Halo 5 porte 7 collisions REELLES (deux victimes distinctes a la meme
// milliseconde). Sur un instant ambigu, aucune mort n est enrichie — et la ligne de film n est
// PAS ajoutee en orpheline : la mort est deja dans la base, l ajouter la compterait deux fois.
func TestFusionRefuseDeChoisirSurUnInstantAmbigu(t *testing.T) {
	autreVictime := mortCredit(1000)
	autreVictime.VictimXUID, autreVictime.VictimGamertag = "xuid(7)", "Autre"

	out, st, err := MergeCreditAndFilm(
		batchCredit(mortCredit(1000), autreVictime), batchFilm(mortFilm(1000)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}

	if st.Enriched != 0 {
		t.Errorf("%d enrichissement(s) sur un instant portant DEUX morts — l arme a ete attribuee "+
			"au hasard a l une des deux", st.Enriched)
	}
	if st.AmbiguousInstants != 1 {
		t.Errorf("instants ambigus = %d, attendu 1 — le refus doit se COMPTER, sans quoi il est "+
			"indistinguable d une absence de film", st.AmbiguousInstants)
	}
	if len(out.Deaths) != 2 {
		t.Errorf("%d morts publiees, attendu 2 — la ligne de film ne doit ni enrichir ni s ajouter "+
			"(la mort est deja dans la base)", len(out.Deaths))
	}
}

// TestFusionSansFilmRendLaBaseIntacte — le cas des ~70 % de matchs sans film, et le plancher.
func TestFusionSansFilmRendLaBaseIntacte(t *testing.T) {
	base := batchCredit(mortCredit(1000), mortCredit(2000))
	base.CreditBaseCount = 0 // le producteur ne l a pas pose : la fusion doit le faire

	out, st, err := MergeCreditAndFilm(base, KillSourceBatch{MatchID: "m"})
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if len(out.Deaths) != 2 || out.DecoderRev != "credit-rev" {
		t.Errorf("%d morts / decoder_rev %q, attendu 2 / credit-rev", len(out.Deaths), out.DecoderRev)
	}
	if out.CreditBaseCount != 2 {
		t.Errorf("CreditBaseCount = %d, attendu 2 — sans plancher, le persister ne peut plus "+
			"refuser une passe appauvrissante", out.CreditBaseCount)
	}
	if st.Enriched != 0 || st.Orphans != 0 {
		t.Errorf("stats non nulles sans film : %+v", st)
	}
}

// TestFusionGardeLaRevisionDuDecodeur — celle du FILM quand il y en a un.
//
// C est la revision du decodeur qui commande le REDECODAGE (la passe chere) et sur elle que
// `levelup backfill-killsource` decide qu un match est a jour. Ecraser cette valeur par celle du
// producteur credit ferait redecoder tout le corpus — 3 a 11 heures — a chaque cycle de sync.
func TestFusionGardeLaRevisionDuDecodeur(t *testing.T) {
	out, _, err := MergeCreditAndFilm(batchCredit(mortCredit(1000)), batchFilm(mortFilm(1000)))
	if err != nil {
		t.Fatalf("MergeCreditAndFilm: %v", err)
	}
	if out.DecoderRev != "film-rev" {
		t.Errorf("decoder_rev = %q, attendu film-rev — le backfill ne reconnaitrait plus les "+
			"matchs deja decodes et redecoderait tout le corpus", out.DecoderRev)
	}
}
