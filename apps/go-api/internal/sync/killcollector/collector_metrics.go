package killcollector

// collector_metrics.go — LES COMPTEURS DE SANTE DE LA PASSE (ADR 0009) : leurs noms, et les
// trois fonctions qui les publient.
//
// Sorti de `collector.go` par deplacement pur au lot 2.7 (2026-09-16, scission des fichiers de
// plus de 500 lignes) : aucune ligne de logique n'a change. Un compteur n'est pas une etape de
// la passe ; les melanger faisait un fichier ou l'on ne savait plus ce qui ORCHESTRE et ce qui
// OBSERVE.

import (
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// Compteurs de sante du collecteur (ADR 0009 : entiers, snake_case, aucun ratio).
const (
	metricCollected   = "killsource_matchs_collectes"
	metricNoFilm      = "killsource_films_absents"
	metricNoKillFeed  = "killsource_sans_killfeed"
	metricDecodeError = "killsource_erreurs_decodage"
	metricTimeout     = "killsource_abandons_delai"
	// metricBudget : passes arretees par leur budget. UN ARRET NOMINAL, pas une erreur —
	// il se compte a part pour ne pas polluer `killsource_erreurs_decodage`.
	metricBudget     = "killsource_budgets_epuises"
	metricWriteError = "killsource_erreurs_ecriture"
	// metricUnknownKey : films ECARTES parce que leur cle ecrite est absente de la table de
	// profil (lot 3.1.1, D-4 d ADR 0034). IL NE REMPLACE PAS les compteurs PAR CLE que
	// `grammar` nomme (`filmdec_unknown_build_<build>`, `filmdec_unknown_format_<n>`) : ceux-la
	// disent QUELLE cle manque, celui-ci dit combien de PASSES la politique a arretees. Les
	// deux se lisent ensemble, et leur somme par cle doit se recouper.
	metricUnknownKey  = "killsource_ecartes_cle_inconnue"
	metricDeaths      = "killsource_morts_ecrites"
	metricNotPublish  = "killsource_passes_non_publiables"
	metricAssistExtra = "killsource_assist_extra_count"
	// LES QUATRE COMPTEURS DE PROVENANCE DU LIEN `indice -> joueur` (lot 1.8). Ils disent, en
	// exploitation et pas seulement dans le journal du jour, quelle part de chaque passe vient
	// d une LECTURE et quelle part d un REPLI (D14 c du chantier, D-10 d ADR 0034) :
	//
	//	table_film     indices lus dans la table des joueurs de `chunk_00`
	//	inference      indices laisses a la bijection inferee des votes du kill-feed
	//	silence        indices lus que le kill-feed ne confirme ni n infirme (le joueur n a ni
	//	               tue ni n est mort dans la fenetre d appariement) — un silence n est PAS
	//	               un desaccord
	//	contradiction  indices lus que les votes du kill-feed designent autrement. La valeur
	//	               publiee NE BOUGE PAS : la lecture prime, la contradiction se compte.
	//
	// UN CINQUIEME COMPTEUR NOMME LE REFUS DE LA TABLE ENTIERE, par cause : sans lui un film
	// tombe au repli complet sans que rien ne le dise en dehors d une ligne de WARN.
	metricBijTableFilm    = "killsource_bijection_table_film"
	metricBijInference    = "killsource_bijection_inference"
	metricBijSilence      = "killsource_bijection_silence"
	metricBijContradict   = "killsource_bijection_contradiction"
	metricBijTableRefusee = "killsource_bijection_table_refusee_"
	// metricBijAmbigue : LES FILMS QUI BASCULENT, ET EUX SEULS.
	//
	// Incremente d UN PAR FILM sur `Inferred == 1 && FreeNames >= 2` : UN indice a inferer pour
	// AU MOINS DEUX noms libres. C est exactement la population que la porte corrigee refuse et
	// que l ancienne (`Inferred <= 1`) publiait — un choix arbitraire entre deux noms, tranche
	// par des votes nuls des deux cotes.
	//
	// IL A SURCOMPTE JUSQU AU 2026-09-16 (revue de jalon M1, RONDE 2, constat F3). La condition
	// etait `Inferred > 0 && FreeNames > Inferred`, donc elle mordait AUSSI sur `Inferred >= 2`
	// — un regime ou l ancienne porte refusait DEJA (`Inferred <= 1` faux) et ou rien ne
	// bascule. Le compteur gonflait ainsi d une population qui ne perd aucune publication, et
	// la phrase qu il porte (« tant qu il est au-dessus de zero, des films sont refuses par le
	// critere corrige ») etait fausse d autant.
	//
	// CE QU IL MESURE, DONC : tant qu il est au-dessus de zero, des films sont refuses ligne
	// par ligne parce que le film ne nomme pas assez d indices — pas parce que le decodage a
	// echoue. `Inferred >= 2` reste hors de ce compte : il est couvert par
	// `killsource_bijection_inference`, qui dit combien d indices sont devines en tout.
	metricBijAmbigue = "killsource_bijection_noms_libres_en_trop"
	// LES SIX COMPTEURS DE PROVENANCE DU COUPLE `(tueur, victime)` (lot 1.9.3). Le kill-feed
	// ecrit le couple lui-meme la plupart du temps ; quand il ne porte que le kill, c est le
	// KILL-EVENT 85 qui le decide, et le recollage sur un instant voisin n est plus qu un repli :
	//
	//	meme_instant   le feed porte le kill ET la mort : aucun arbitrage
	//	lu             le film ECRIT le couple (kill-event 85) et c est lui qui decide
	//	recolle        LE REPLI (`repli_couple_recolle_sur_le_voisin`) — tant qu il monte, des
	//	               couples sont encore DEVINES, et c est lui qui dira quand le retirer (D14 d)
	//	muet           la lecture s est tue : le diagnostic typé qui OUVRE le repli
	//	ambigu         deux enregistrements nomment le meme tueur et des victimes differentes
	//	contradiction  le film nomme une victime dont aucun instant voisin ne porte la mort. La
	//	               valeur publiee ne bouge pas — la lecture prime —, l ecart se compte.
	//
	// Une SEPTIEME quantite se compte a part parce qu elle n est pas un couple : les kills dont
	// le film NOMME un bot en victime. Avant ce lot, ils etaient RECOLLES sur la mort d un
	// humain et fabriquaient un couple qui n a jamais eu lieu.
	metricCoupleMemeInstant = "killsource_couple_meme_instant"
	metricCoupleLu          = "killsource_couple_lu"
	metricCoupleRecolle     = "killsource_couple_recolle"
	metricCoupleMuet        = "killsource_couple_muet"
	metricCoupleAmbigu      = "killsource_couple_ambigu"
	metricCoupleContradict  = "killsource_couple_contradiction"
	metricVictimeBotLue     = "killsource_victime_bot_lue"

	// D OU VIENT L APPARIEMENT `dead-state <-> kill-feed` de chaque ligne publiee (lot 1.9.7).
	// L IDENTITE DE PAQUET decide ; la fenetre de 2,5 s n entre que sur son silence, et elle est
	// alors un REPLI NOMME au registre. Les trois compteurs de repli sont SEPARES parce que leurs
	// criteres de retrait le sont : `_fenetre` doit tomber a zero (le film ecrit le lien), les
	// deux autres mesurent un negatif (le kill feed est humain-seul, il ne nomme ni la mort d un
	// bot ni une mort que personne ne revendique).
	metricApparIdentite    = "killsource_appariement_identite"
	metricApparFenetre     = "killsource_appariement_fenetre"
	metricApparBotFenetre  = "killsource_appariement_bot_fenetre"
	metricApparNonRevFen   = "killsource_appariement_non_revendiquee_fenetre"
	metricCoupleSansPaquet = "killsource_couple_sans_identite_de_paquet"
	metricAssistFenetre    = "killsource_assistant_fenetre"
)

// publishKillSourceMetrics publie les compteurs de sante (ADR 0009).
//
// `killsource_assist_extra_count` est LE declencheur de migration vers une table fille : le jour
// ou il bouge, l hypothese de schema « un seul assistant » est en defaut. Il est publie ICI en
// plus d etre stocke sur les lignes, parce qu un compteur qu il faut interroger en SQL pour voir
// bouger n alerte personne.
func publishKillSourceMetrics(res *decfilm.Result, batch persist.KillSourceBatch) {
	observability.AddInt(metricCollected, 1)
	observability.AddInt(metricDeaths, int64(len(batch.Deaths)))
	if !batch.Publishable {
		observability.AddInt(metricNotPublish, 1)
	}
	extra := 0
	for i := range batch.Deaths {
		extra += batch.Deaths[i].AssistExtra
	}
	if extra > 0 {
		observability.AddInt(metricAssistExtra, int64(extra))
	}
	for _, p := range res.Health.ExpvarPairs() {
		observability.AddInt(p.Name, p.Value)
	}
	publishBijectionProvenance(res.Roster.FilmTable)
	publishCoupleProvenance(res.Stats.Couples)
	publishApparProvenance(res.Stats.Appariement)
	if n := res.Stats.Assist.ParLaFenetre; n > 0 {
		observability.AddInt(metricAssistFenetre, int64(n))
	}
}

// publishCoupleProvenance : D OU VIENT LE COUPLE `(tueur, victime)`, en exploitation (lot 1.9.3).
//
// LE COMPTEUR QUI INFORME EST `killsource_couple_recolle` : c est le REPLI, et son critere de
// retrait est ecrit au registre (`repli_couple_recolle_sur_le_voisin`). Les trois compteurs de
// diagnostic (`_muet`, `_ambigu`, `_contradiction`) disent POURQUOI il a fallu se replier — sans
// eux, un compte de replis ne designe aucune correction.
func publishCoupleProvenance(c decfilm.CoupleStats) {
	for _, p := range []struct {
		nom string
		val int
	}{
		{metricCoupleMemeInstant, c.MemeInstant},
		{metricCoupleLu, c.Lus},
		{metricCoupleRecolle, c.Recolles},
		{metricCoupleMuet, c.Muet},
		{metricCoupleAmbigu, c.Ambigu},
		{metricCoupleContradict, c.Contradiction},
		{metricVictimeBotLue, c.VictimesBotLues},
	} {
		if p.val > 0 {
			observability.AddInt(p.nom, int64(p.val))
		}
	}
}

// publishBijectionProvenance : D OU VIENT LE LIEN `indice -> joueur`, en exploitation (lot 1.8).
//
// LE COMPTEUR QUI INFORME EST `killsource_bijection_inference` : tant qu il monte, des indices
// sont encore DEVINES au lieu d etre lus, et c est ce qui dira quand le repli pourra etre retire
// (D14 d : un repli dont le compte est a zero sur un jalon se supprime au suivant). Les cinq
// films du cache sans section d identification (`03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`,
// `a349fea8`) le tiennent au-dessus de zero, et c est la raison ECRITE pour laquelle l inference
// reste.
func publishBijectionProvenance(t decfilm.FilmTablePinning) {
	observability.AddInt(metricBijTableFilm, int64(t.Pinned))
	observability.AddInt(metricBijInference, int64(t.Inferred))
	observability.AddInt(metricBijSilence, int64(t.Silent))
	observability.AddInt(metricBijContradict, int64(t.Contradict))
	if t.Inferred == 1 && t.FreeNames >= 2 {
		observability.AddInt(metricBijAmbigue, 1)
	}
	if t.Refusal == decfilm.FilmTableRead {
		return
	}
	// La cause entre dans le NOM du compteur : « la table a ete refusee » sans dire pourquoi
	// n oriente aucun diagnostic. Meme forme que `filmdec_unknown_build_<build>` (ADR 0009).
	observability.AddInt(metricBijTableRefusee+string(t.Refusal), 1)
	if t.Refusal == decfilm.FilmTableUnknownBuild {
		// D-4 d ADR 0034 : un build hors profil est mis de cote AVEC son compteur nomme, pour
		// que le refus se voie en production et pas seulement au journal.
		for _, p := range decfilm.UnknownBuildExpvarPairs(t.Build) {
			observability.AddInt(p.Name, p.Value)
		}
	}
}

// publishApparProvenance : D OU VIENT L APPARIEMENT `dead-state <-> kill-feed`, en exploitation
// (lot 1.9.7).
//
// LE COMPTEUR QUI INFORME EST `killsource_appariement_fenetre` : c est le REPLI
// `repli_appariement_par_fenetre_temporelle`, et son critere de retrait est ecrit au registre.
// `killsource_couple_sans_identite_de_paquet` dit POURQUOI il a fallu se replier — sans lui, un
// compte de replis ne designe aucune correction.
func publishApparProvenance(a decfilm.ApparStats) {
	for _, p := range []struct {
		nom string
		val int
	}{
		{metricApparIdentite, a.Identite},
		{metricApparFenetre, a.Fenetre},
		{metricApparBotFenetre, a.BotFenetre},
		{metricApparNonRevFen, a.NonRevendiqueeFenetre},
		{metricCoupleSansPaquet, a.CouplesSansIdentite},
	} {
		if p.val > 0 {
			observability.AddInt(p.nom, int64(p.val))
		}
	}
}
