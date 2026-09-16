package persist

// kill_events_merge.go — L INVERSION DE PRESEANCE : LE CREDIT EST LA BASE, LE FILM ENRICHIT.
//
// ─── LE DEFAUT QUE CE FICHIER FERME ────────────────────────────────────────────────────────
//
// Jusqu ici la preseance etait FILM > CREDIT : la vue `_latest` ne retenant qu UNE passe par
// match, une passe de film REMPLACAIT entierement la passe credit. Or la passe de film ne publie
// que 74,4 % des morts que le credit porte a 98,4 % : la bascule des lecteurs aurait efface
// 25 697 morts sur 949 matchs — sans erreur, sans compteur, sans qu un seul nom change. Seulement
// moins de lignes. C est la mesure qui a arrete la bascule en J4 session 3.
//
// La relation s inverse ici : la LISTE DES MORTS vient du credit, et le film y AJOUTE ce que lui
// seul sait (l arme via `source_tag`, l assistant nomme, les parts de degats, la divergence). Il
// n en RETIRE jamais aucune.
//
// ─── POURQUOI L APPARIEMENT EST SUR, ET POURQUOI LA TOLERANCE EST ZERO ─────────────────────
//
// LES DEUX SOURCES SONT LE MEME FLUX, LU A DEUX ENDROITS. Le kill-feed du film est decode par
// `analysis.ParseHighlightEvents` — exactement le meme parseur que celui qui alimente
// `highlight_events` depuis l API. Ce ne sont pas deux mesures independantes d un meme evenement :
// c est une seule mesure, livree par deux canaux. Il n y a donc AUCUN ecart d horloge a absorber,
// et c est mesure : passer de 0 ms a 1 SECONDE ne gagne que 8 lignes sur 74 569 (0,01 %).
//
// LA TOLERANCE DETRUIRAIT LA BIJECTION. A tolerance 0 sur la clef `(match_id, time_ms)`, le
// nombre de lignes de film appariees egale au chiffre pres le nombre de morts de credit appariees
// (73 589 = 73 589) : l appariement est une bijection STRICTE. Des 50 ms les deux nombres
// divergent (73 618 contre 73 848) — signature d une ligne de film qui capture plusieurs morts.
// Toute tolerance non nulle achete quelques appariements de plus au prix de l unicite, et
// l unicite est ce qui rend l enrichissement sur.
//
// L IDENTITE ETAIT UN CONTROLE ; DEPUIS LE LOT 2.9 ELLE EST AUSSI LE CRITERE. Sur les 73 589
// lignes appariees de 2026-08 : 0 divergence de xuid de victime, 0 divergence de xuid de tueur, le
// seul ecart etant une ABSENCE (631 victimes et 754 tueurs pour lesquels le film n a pas resolu de
// xuid) — c est exactement la population qu une clef a quatre colonnes perdrait. D ou la regle :
// la VICTIME apparie quand les deux cotes la portent, l INSTANT apparie en repli quand elle
// manque d un cote, et le controle d identite garde les paires ainsi formees
// (`kill_events_merge_pairing.go`, [verifierConcordance]).
//
// ─── CE QUE LA MESURE DU 2026-09-16 A CONTREDIT (lot 2.9, D1 de la cloture M1) ──────────────
//
// LE TEMOIN : `9f9b19e5-5df4-4268-aa32-900a4fc6725a@63757`. A cet instant le credit porte la mort
// de 2535413577167650 (Artemlv2774, tue par Ritio3987) et le film celle de 2535427572079378
// (DANIELBOIMEXICO, dont les morts de credit de ce match sont a 106 661 ms, 162 355 ms, ...).
// DEUX MORTS DISTINCTES A LA MEME MILLISECONDE, chaque cote n en voyant qu une. La clef
// `(match_id, time_ms)` les a appariees, le controle d identite a rendu l erreur — et la passe a
// refuse LE FILM ENTIER (249 films ecrits sur 250, tranche 1 du backlog du 2026-09-17).
//
// LES CHIFFRES, en lecture seule sur les vues `_latest` (oracle `pre-chaine-2026-09-09` pour
// Infinite, base du titre pour Halo 5) :
//
//	credit d Infinite, couples bruts     137 286 instants sur 1 384 matchs — 0 a deux victimes
//	passes servies d Infinite            138 807 instants — 0 a deux morts, maximum 1 par instant
//	credit de Halo 5, couples bruts      268 330 instants sur 2 754 matchs — 7 a DEUX VICTIMES
//	                                     DISTINCTES a la meme milliseconde
//	entre les deux cotes d un match      le temoin ci-dessus. INVISIBLE EN BASE : la fusion
//	                                     echouait avant d ecrire quoi que ce soit
//
// L UNICITE TIENT A L INTERIEUR DE CHAQUE COTE, PAS ENTRE LES DEUX. Le film voit des morts que le
// kill-feed humain-seul ne porte pas (bots), et il lui manque 25,6 % de celles du credit : deux
// morts a la meme milliseconde suffisent pour que chaque cote n en voie qu une, et pas la meme.
//
// ─── CE QUE LA FUSION NE FABRIQUE JAMAIS ───────────────────────────────────────────────────
//
// Les TROIS etats de l assistant survivent, et aucune combinaison nouvelle n apparait :
//
//	mort de credit NON enrichie   assist_known = FALSE, champs d assistant NULL. Etat 1, inchange.
//	mort de credit ENRICHIE       les champs d assistant sont recopies VERBATIM de la ligne de
//	                              film — Y COMPRIS quand celle-ci est elle-meme en etat 1 (le film
//	                              est muet sur 2,1 % des lignes appariees). Le film reste l unique
//	                              autorite sur ce qu il a ou n a pas observe.
//
// INTERDIT, ET RENDU IMPOSSIBLE ICI : deriver `assist_known` d autre chose que de la ligne de
// film. Un defaut a TRUE sur une mort non enrichie fabriquerait 60 297 « mesures : pas
// d assistant » jamais observees. Meme regle pour `source_tag`/`source_category`/`diverges`/parts
// de degats : NULL hors enrichissement (« non mesure »), valeur du film sinon.
//
// ─── PURE, ET C EST CE QUI LA REND TESTABLE ────────────────────────────────────────────────
//
// Aucune base, aucun contexte, aucun compteur publie : [MergeCreditAndFilm] rend ses observations
// dans [MergeStats] et ce sont ses APPELANTS qui les publient (ADR 0009). Une fonction qui
// ecrirait dans expvar ne serait plus rejouable dans un test sans polluer le process.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// Compteurs de sante de la fusion (ADR 0009 : entiers, snake_case, aucun ratio).
//
// Ils sont publies par [PublishMergeStats] et NON par [MergeCreditAndFilm] : la fusion est pure,
// et une fonction pure ne touche pas a expvar.
const (
	metricFusionEnrichies = "killsource_fusion_morts_enrichies"
	metricFusionOrphelins = "killsource_fusion_orphelins_film"
	// metricFusionOrphelinsHvH — LE COMPTEUR DEDIE des orphelins humain-contre-humain.
	//
	// Les orphelins de bot s expliquent structurellement (le kill-feed de l API est humain seul) ;
	// ceux-la, non. 13 lignes sur 74 569 au 2026-08-02, chacune portant deux xuids et coincidant
	// avec un evenement REEL de l API. Ils sont le symetrique attendu de la bijection greedy du
	// credit — une mort deja consommee par un autre kill — mais ce mecanisme n est PAS demontre.
	// C est la seule des trois populations d orphelins dans ce cas : elle se surveille.
	metricFusionOrphelinsHvH = "killsource_orphelins_film_humain_contre_humain"
	metricFusionAmbigus      = "killsource_fusion_instants_ambigus"
)

// PublishMergeStats publie les observations d une fusion et journalise les trois populations qui
// ne vont pas de soi.
//
// LE COMPTEUR SEUL NE SUFFIT PAS pour les instants ambigus : il dit COMBIEN, jamais OU. Le log
// porte le `match_id`, qui est la seule quantite avec laquelle on peut aller voir.
func PublishMergeStats(ctx context.Context, matchID string, st MergeStats) {
	observability.AddInt(metricFusionEnrichies, int64(st.Enriched))
	observability.AddInt(metricFusionOrphelins, int64(st.Orphans))
	observability.AddInt(metricFusionOrphelinsHvH, int64(st.OrphansHumanVsHuman))
	observability.AddInt(metricFusionAmbigus, int64(st.AmbiguousInstants))
	if st.AmbiguousInstants > 0 {
		slog.WarnContext(ctx, "killsource: instants portant plusieurs morts — enrichissement "+
			"refuse sur ces instants (aucune mort perdue, aucune arme attribuee au hasard)",
			"match_id", matchID, "instants_ambigus", st.AmbiguousInstants)
	}
	if st.OrphansSharedInstant > 0 {
		slog.WarnContext(ctx, "killsource: deux morts a la meme milliseconde — la ligne de film "+
			"porte une AUTRE victime que la mort de credit du meme instant, elle est conservee "+
			"en orpheline (lot 2.9 ; avant lui la passe entiere tombait)",
			"match_id", matchID, "orphelins_instant_partage", st.OrphansSharedInstant)
	}
	if st.OrphansHumanVsHuman > 0 {
		slog.InfoContext(ctx, "killsource: orphelins de film humain contre humain conserves — "+
			"population sous surveillance (mecanisme non demontre, cf. kill_events_merge.go)",
			"match_id", matchID, "orphelins_hvh", st.OrphansHumanVsHuman,
			"orphelins_total", st.Orphans)
	}
}

// MergeStats : ce que la fusion a observe. Rendu a l appelant, jamais publie ici.
type MergeStats struct {
	// Enriched : morts de credit qui ont recu l enrichissement d une ligne de film.
	Enriched int
	// Orphans : lignes de film CONSERVEES telles quelles (cf. [MergeCreditAndFilm], section
	// « le sort des orphelins ») — celles dont l instant ne porte aucune mort de credit, plus
	// celles que la victime prouve etre une AUTRE mort que celles de leur instant.
	Orphans int
	// OrphansSharedInstant : la sous-population des orphelins qui PARTAGENT LEUR INSTANT avec une
	// mort de credit portant une autre victime — deux morts a la meme milliseconde, une de chaque
	// cote (temoin `9f9b19e5@63757`). C est la population que le lot 2.9 rend visible : avant lui
	// elle faisait tomber la passe entiere.
	//
	// ELLE EST JOURNALISEE AVEC LE `match_id` ET NON PUBLIEE EN `expvar`, faute d une restitution
	// CLI (`cmd/levelup/cmd_backfill_killsource_sante.go`, hors frontiere du lot 2.9) : un
	// compteur expvar que la commande n affiche pas mourrait avec le process. Report consigne.
	OrphansSharedInstant int
	// OrphansHumanVsHuman : la sous-population des orphelins qui porte DEUX xuids — un humain
	// tue un humain. C est la seule des trois populations d orphelins dont le mecanisme ne soit
	// pas demontre (les deux autres sont des morts de bot, structurellement absentes du kill-feed
	// de l API qui est HUMAIN SEUL). 13 lignes sur 74 569 au 2026-08-02 : elle merite un
	// compteur, pas un rejet.
	OrphansHumanVsHuman int
	// AmbiguousInstants : instants ou l appariement a REFUSE DE CHOISIR — au moins une ligne de
	// film que ni la victime ni le repli sur l instant n ont su rattacher. Les morts de credit y
	// gardent leur etat credit (rien n est perdu), et les lignes de film refusees n y sont PAS
	// ajoutees en orphelines (la mort est peut-etre deja dans la base : ce serait la compter deux
	// fois). Un instant dont TOUTES les lignes de film sont tranchees — appariees par la victime,
	// ou prouvees autres morts — n est plus compte ici : la question y a une reponse.
	//
	// L unicite de `(match_id, time_ms)` est une propriete MESUREE de chaque COTE PRIS SEUL
	// (74 569 clefs pour 74 569 lignes de film, 98 662 pour 98 662 morts de credit ; 137 286
	// instants de couples credit sur Infinite au 2026-09-16, 0 a deux victimes). Elle n est ni une
	// garantie de schema — Halo 5 porte 7 collisions reelles, deux victimes distinctes a la meme
	// milliseconde — NI UNE PROPRIETE ENTRE LES DEUX COTES : le temoin `9f9b19e5@63757` porte une
	// mort de credit et une mort de film a la meme milliseconde, victimes differentes (en-tete du
	// fichier). Ce compteur n a jamais vu ce cas-la : il tombait sur le controle d identite avant.
	AmbiguousInstants int
}

// MergeCreditAndFilm : la base credit ENRICHIE de ce que le film a mesure.
//
// CONTRAT, dans l ordre :
//
//  1. la sortie porte TOUTES les morts de `base`, dans leur ordre, sans exception ;
//  2. une mort de `base` est enrichie quand [apparier] lui a trouve une ligne de film — par la
//     VICTIME quand les deux cotes la portent, par l instant en repli quand elle manque d un cote
//     et qu il ne reste qu une mort de chaque cote ; sinon elle reste telle quelle ;
//  3. sur une paire FORMEE, les xuids presents des deux cotes doivent concorder : une divergence
//     est une ERREUR rendue, pas une ligne ecartee en silence (cf. [verifierConcordance]) ;
//  4. une ligne de film est AJOUTEE telle quelle (orpheline) quand son instant ne porte aucune
//     mort de credit, ou quand la victime prouve qu elle est une AUTRE mort que celles de son
//     instant ; elle est REFUSEE — ni enrichissante ni ajoutee — quand l appariement n a pas su
//     trancher, sans quoi une mort deja en base serait comptee deux fois.
//
// ─── LE SORT DES ORPHELINS : ILS SONT CONSERVES, ET C EST UNE MESURE QUI LE DECIDE ─────────
//
// 980 lignes de film (1,3 %) n ont aucune mort de credit en face. Aucune n est une anomalie du
// decodeur : 819 sont « un humain tue un BOT », 149 « un BOT tue un humain », 13 humain contre
// humain — et sur les 980, ZERO ne tombe sur un instant sans evenement de l API (831 coincident
// avec un `kill`, 158 avec un `death`).
//
// La raison est structurelle : LE KILL-FEED DE L API EST HUMAIN SEUL. Un bot n a pas de xuid, sa
// mort ne produit aucun evenement — quand un humain tue un bot, l API porte le KILL et pas la
// MORT ; quand un bot tue un humain, l inverse. Dans les deux cas, aucun couple, donc aucune ligne
// de credit. Le credit fait foi sur l existence des morts D HUMAINS ; il ne fait pas foi sur les
// morts que sa source ne peut pas representer. Les rejeter serait traiter une absence de mesure
// comme une mesure d absence.
//
// ─── LE `decoder_rev` DE LA PASSE FUSIONNEE, ET POURQUOI C EST CELUI DU FILM ────────────────
//
// La colonne est unique par passe et la passe a desormais deux producteurs. On garde celui du
// FILM des qu il y a du film : c est lui qui commande le REDECODAGE (la passe chere), et c est sur
// lui que `levelup backfill-killsource` decide qu un match est a jour. Garder celui du credit
// ferait redecoder tout le corpus a chaque changement du producteur credit — 3 a 11 heures pour
// rien.
func MergeCreditAndFilm(base, film KillSourceBatch) (KillSourceBatch, MergeStats, error) {
	var st MergeStats
	if len(film.Deaths) == 0 {
		base.CreditBaseCount = len(base.Deaths)
		return base, st, nil
	}
	if base.MatchID != "" && film.MatchID != "" && base.MatchID != film.MatchID {
		return KillSourceBatch{}, st, fmt.Errorf(
			"persist: fusion credit<->film sur deux matchs differents (%q et %q)",
			base.MatchID, film.MatchID)
	}

	ap := apparier(base.Deaths, film.Deaths)

	out := KillSourceBatch{
		MatchID:         choisirNonVide(base.MatchID, film.MatchID),
		DecoderRev:      choisirNonVide(film.DecoderRev, base.DecoderRev),
		Publishable:     film.Publishable,
		CreditBaseCount: len(base.Deaths),
		Deaths:          make([]KillEventInsert, 0, len(base.Deaths)+len(film.Deaths)),
	}

	for i := range base.Deaths {
		mort := base.Deaths[i]
		idx, apparie := ap.filmPourCredit[i]
		if !apparie {
			out.Deaths = append(out.Deaths, mort)
			continue
		}
		if err := verifierConcordance(out.MatchID, &mort, &film.Deaths[idx]); err != nil {
			return KillSourceBatch{}, st, err
		}
		enrichir(&mort, &film.Deaths[idx])
		st.Enriched++
		out.Deaths = append(out.Deaths, mort)
	}
	st.AmbiguousInstants = ap.instantsAmbigus

	// LES ORPHELINS, dans l ordre du film. Une ligne de film REFUSEE (instant ambigu) n en est pas
	// une : la mort est peut-etre deja dans la base, l ajouter la compterait deux fois.
	for i := range film.Deaths {
		v := ap.verdict[i]
		if v == filmApparie || v == filmRefuse {
			// APPARIEE : la mort est deja dans la sortie, enrichie — l ajouter la compterait
			// deux fois. REFUSEE : l appariement n a pas su trancher, et la mort est peut-etre
			// dans la base sous l autre ligne de l instant — meme risque, meme abstention.
			continue
		}
		out.Deaths = append(out.Deaths, film.Deaths[i])
		st.Orphans++
		if film.Deaths[i].VictimXUID != "" && film.Deaths[i].FeedKillerXUID != "" {
			st.OrphansHumanVsHuman++
		}
		if v == filmOrphelinInstantPartage {
			st.OrphansSharedInstant++
		}
	}
	return out, st, nil
}

// verifierConcordance : LE CONTROLE D IDENTITE D UNE PAIRE DEJA FORMEE. Quand les deux cotes
// portent un xuid, il doit etre le meme.
//
// Une divergence est une ERREUR RENDUE et pas une ligne ecartee : elle signifie que la clef
// `(match_id, time_ms)` a apparie deux morts differentes, c est-a-dire que la propriete sur
// laquelle repose l appariement est fausse a cet instant.
//
// LES DEUX SEULES FACONS D ARRIVER ICI (lot 2.9, corrige a la revue adversariale du 2026-09-16).
// La fonction ne voit QUE des paires deja formees par [apparier], et il n en existe que deux
// sortes :
//
//	passe 1, par la victime          les deux cotes portent la MEME victime resolue.
//	passe 3, repli sur l instant     l instant porte UNE mort de credit et UNE ligne de film EN
//	                                 TOUT, et l une des deux victimes au moins est absente.
//
// LA VICTIME DIVERGENTE EST DONC IMPOSSIBLE PAR CONSTRUCTION DE CES DEUX PASSES : en passe 1 les
// victimes sont egales, en passe 3 l une est vide — et une absence n est pas une divergence. Le
// test reste, comme INVARIANT : le jour ou une paire serait formee sur autre chose que l identite,
// il vaut mieux perdre une passe que recopier l arme d une mort sur une autre. (Il n est PAS
// garde au nom d une unicite mesuree cote film : cette unicite-la n est pas mesurable en base,
// cf. §4 D2 (2.9) du plan — c est la construction de l appariement qui le fonde, pas un chiffre.)
//
// LE TUEUR DIVERGENT, LUI, EST ATTEIGNABLE, et c est le garde-fou vivant : la paire porte la MEME
// victime au MEME instant — donc la meme mort, une victime ne mourant pas deux fois dans la meme
// milliseconde — et deux tueurs resolus differents. Les deux cotes se contredisent sur QUI a tue :
// la passe tombe, bruyamment.
//
// L ABSENCE N EST PAS UNE DIVERGENCE : le film ne resout pas toujours un xuid (631 victimes,
// 754 tueurs). C est le cas normal d un nom que le roster n a pas su rattacher, et c est la
// population que le repli sur l instant existe pour garder.
func verifierConcordance(matchID string, credit, film *KillEventInsert) error {
	if credit.VictimXUID != "" && film.VictimXUID != "" && credit.VictimXUID != film.VictimXUID {
		return fmt.Errorf("persist: fusion %s@%d: victime divergente entre credit (%s) et film (%s) — "+
			"la clef (match_id, time_ms) a apparie deux morts differentes",
			matchID, credit.TimeMS, credit.VictimXUID, film.VictimXUID)
	}
	if credit.FeedKillerXUID != "" && film.FeedKillerXUID != "" &&
		credit.FeedKillerXUID != film.FeedKillerXUID {
		return fmt.Errorf("persist: fusion %s@%d: tueur divergent entre credit (%s) et film (%s) — "+
			"la clef (match_id, time_ms) a apparie deux morts differentes",
			matchID, credit.TimeMS, credit.FeedKillerXUID, film.FeedKillerXUID)
	}
	return nil
}

// enrichir : les champs que SEUL le film mesure passent du film a la mort de credit.
//
// CE QUI NE BOUGE PAS, ET C EST LE POINT : l instant, la victime, le tueur du feed et
// `FeedPresent` restent CEUX DU CREDIT. Le credit est la base — il porte les identites resolues
// (le film n en resout pas toujours), et c est lui qui fait foi sur l existence de la mort.
//
// CE QUI EST RECOPIE VERBATIM : tout le bloc assistant (les trois etats compris — un film muet
// reste muet), la source du degat, les parts, la divergence, ET la portee (`read_path` /
// `read_origin`). La portee reste PAR LIGNE : c est ce qui permet a un lecteur de savoir, mort par
// mort, si l arme a ete mesuree — et c est ce qui garde `FilmReadPaths` operant.
func enrichir(credit *KillEventInsert, film *KillEventInsert) {
	credit.AssistGamertag = film.AssistGamertag
	credit.AssistXUID = film.AssistXUID
	credit.AssistKnown = film.AssistKnown
	credit.AssistIndex = film.AssistIndex
	credit.AssistRejected = film.AssistRejected
	credit.AssistExtra = film.AssistExtra
	credit.SourceTag = film.SourceTag
	credit.SourceCategory = film.SourceCategory
	credit.KillerDamagePct = film.KillerDamagePct
	credit.AssistDamagePct = film.AssistDamagePct
	credit.Diverges = film.Diverges
	credit.ReadPath = film.ReadPath
	credit.ReadOrigin = film.ReadOrigin
}

// choisirNonVide : la premiere valeur non vide. Sert le `match_id` et le `decoder_rev` de la passe
// fusionnee quand l un des deux cotes ne les porte pas.
func choisirNonVide(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
