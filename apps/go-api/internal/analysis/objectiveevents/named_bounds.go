package objectiveevents

// named_bounds.go — LA DERIVATION UNIQUE DES INCREMENTS FILTRES, ET LES BORNES QUI LA REGLENT.
//
// Extrait de `named_series.go` le 2026-09-11 (correctif 6.R) : le fichier y depassait le seuil
// de 500 lignes, et la coupe suit la RESPONSABILITE. Ici vivent les deux bornes de prudence du
// deroulage, le budget d'evenements d'une passe, et les DEUX FORMES d'une meme lecture d'un
// compteur — [incrementTimes] pour les evenements, [boundedSeries] pour la serie.
// `named_series.go` garde ce qui va des enregistrements a une suite de valeurs (groupement,
// rejet des ancrages parasites, cumul des manches).

import (
	"log/slog"
)

// # LES BORNES DE PRUDENCE DU DEROULAGE (lot 4b, 2026-09-03 — decision utilisateur du meme jour)
//
// POURQUOI ELLES EXISTENT. [incrementTimes] emet UN entier par UNITE gagnee par le compteur.
// Un enregistrement mal aligne — decode a une position ou il n'y en a pas, donc dont TOUS les
// canaux sont faux a la fois — porte une valeur arbitraire et POSITIVE, que ni le rejet des
// emissions negatives ni le domaine du score de mode n'attrapent. Mesure du 2026-09-03
// (item 4b.1) : `51101d1d` deroule 2 163 333 677 evenements sur le seul `comp 20 B`
// (slot 24, t = 136 636 ms) ; le `[]int` pese alors 17,31 Go et son dernier `growslice`
// (8,65 Go copies vers 17,31 Go, les deux vivants) demande 25,96 Go — les « ~26 Go, crash go
// runtime » du registre des reports (2026-08-24), retrouves a 0,2 % pres. Trois autres films
// du cache ont la meme structure (`a349fea8` 1,1 Md sur 21 B ; `1c4c63c2` 537 M sur 22 A ;
// `60ae07c4` 2,1 Md sur 21 A).
//
// CE QUE CES BORNES SONT. Un rempart MEMOIRE d'abord, et — depuis le lot 6.7-B1 — un filtre
// d'ANOMALIE calibre sur l'oracle API. Elles ne travaillent PAS seules, et c'est le lot 6.11 qui
// leur a donne leur second etage : le filtre au niveau de l'ENREGISTREMENT existe depuis lors
// (`statMaxCounter` + `statCountersInDomain`, `statborg.go`), et il rejette le record ENTIER des
// qu'un de ses canaux A ou B sort du domaine — exactement ce que [modeScoreInDomain] fait pour le
// score de mode.
//
// LES DEUX BORNES SONT CALIBREES L'UNE PAR RAPPORT A L'AUTRE, et aucune ne couvre seule le
// phenomene :
//
//	le PAS (ici)             coupe les gros deroulages et laisse passer les petits. Le record
//	                         fortuit de `fb1a1a72` (slot 24, t = 764 967, canaux a
//	                         2 415 919 104 et -30 456) portait un pas de 10 sur `comp 22 A` :
//	                         il passe SOUS [maxUnrollPerStep] = 16, et cette borne-ci ne peut
//	                         rien en dire — dix prises de drapeau publiees pour ZERO a l'oracle ;
//	l'ENREGISTREMENT         ne regarde pas le pas mais l'ORDRE DE GRANDEUR des canaux : la pire
//	  (`statborg.go`)        valeur SAINE mesuree vaut 102 934, la plus petite ABERRANTE
//	                         2 415 919 104, et [statMaxCounter] = 2^20 se pose dans le vide qui
//	                         les separe. C'est lui, et lui seul, qui attrape le cas ci-dessus.
const (
	// maxUnrollPerStep borne le deroulage d'UN point, PREMIER TERME COMPRIS (`prev` part de
	// zero et ne redescend jamais : la grandeur qui explose est `p.Value - prev`, pas l'ecart
	// entre deux echantillons consecutifs).
	//
	// # LA VALEUR EST MESUREE SUR L'ORACLE API (lot 6.7-B1, item 6, 2026-09-11)
	//
	// ELLE VALAIT 100 000, ET C'ETAIT TROIS ORDRES DE GRANDEUR AU-DESSUS DU PHENOMENE. Sa
	// justification d'origine (item 4b.1) qualifiait de « pire deroulage d'un film SAIN » les
	// 17 306 unites du comp 20 B de `d9781168`. L'oracle refute cette qualification :
	// `match_objective_stats_latest` etablit que ce MEME emplacement produit, sur `a0c36016`,
	// 84 assistances de capture pour ZERO reelle. La population dite saine ne l'etait pas, et
	// la borne se calait donc sur du bruit (audit du 2026-09-10, decouverte 12.3).
	//
	// RECALIBRAGE, 68 artefacts du parc confrontes joueur par joueur a l'oracle et a la feuille
	// de match. La grandeur mesuree est le plus gros deroulage d'UN pas, lu dans l'artefact
	// lui-meme : `incrementTimes` DATE toutes les unites d'un pas au meme instant, donc n
	// actions publiees au meme `timeMs` pour un meme (joueur, statistique) sont un pas de n.
	//
	//	population SAINE      1 sur 376 triples (film, action, joueur) d'action d'objectif ;
	//	                      1 a 3 sur 147 triples de `kills` (3 une seule fois) ;
	//	                      1 sur 130 triples de `assists`. PIRE PAS SAIN : 3.
	//	population ABERRANTE  64 (`4f77afc1` flag_grabs), 64 (`cde26226` flag_steals),
	//	                      84 (`a0c36016` flag_capture_assists), puis 9 482 / 15 608 /
	//	                      15 610 (`assists` de `16ea3668`, `f8efc5ca`, `8bc6074f`).
	//	                      PLUS PETIT PAS ABERRANT : 64.
	//
	// Les deux populations sont separees d'un facteur 21 et RIEN n'occupe l'intervalle. 16 s'y
	// pose : 5,3 fois au-dessus du pire pas sain — la marge couvre la PREMIERE emission d'un
	// slot vu en retard, qui date d'un coup les unites deja acquises (cf. [incrementTimes]) —
	// et 4 fois sous le plus petit pas aberrant.
	//
	// L'ORACLE VALIDE LE RESULTAT A L'UNITE sur les trois films a `assists` explosives : le pas
	// aberrant retire, le total publie tombe a 6, 6 et 8 — exactement les valeurs de la feuille
	// de match. Les quatre bombes memoire connues (537 698 416 a 2 163 333 677) restent
	// neutralisees avec une marge de 33 millions.
	maxUnrollPerStep = 16
	// maxNamedEventsPerFilm borne le TOTAL emis par une passe sur un film. Le pire total sain
	// mesure vaut 21 160 (`d9781168`) — marge 47x ; les quatre bombes sont 500 a 3 900 fois
	// au-dessus. Une passe qui l'atteint s'arrete : le rejeu vaut mieux tronque qu'absent.
	maxNamedEventsPerFilm = 1_000_000
	// maxRejectLogs borne le DETAIL journalise. Un film pathologique peut porter un deroulage
	// aberrant par point (jusqu'a `statMaxRecordsPerFilm` = 33 076) : sans cette borne, le
	// dernier rempart memoire deviendrait un rempart a inonder le journal. Le COMPTE, lui,
	// n'est jamais tronque — il est publie par [eventBudget.resume].
	maxRejectLogs = 8
)

// eventBudget porte le solde d'evenements d'une passe sur un film, et la trace de ce que les
// bornes ont refuse.
//
// LE SOLDE DESCEND JUSQU'A [incrementTimes], ET CE N'EST PAS UN DETAIL D'IMPLEMENTATION.
// `statMaxRecordsPerFilm` (33 076) borne les POINTS, pas les evenements : sous la seule borne
// par pas, UNE SEULE serie peut encore emettre 33 076 x 100 000 entrees, soit 26 Gio DANS UN
// SEUL APPEL — avant que l'appelant ait la main pour verifier un total. Un plafond verifie
// seulement ENTRE les series ne protegerait donc de rien. Ce qui voyage ici est le SOLDE ; la
// VALEUR du plafond, elle, reste detenue par les trois passes qui ouvrent un budget
// ([NamedEventsFrom], [SlotIdentityFrom], [crossCheckFrom]).
type eventBudget struct {
	// origine nomme la passe dans le journal — c'est ce qui distingue « le nommage a rejete »
	// de « le pont d'identite a rejete » quand les deux lisent le meme film.
	origine string
	// reste est le nombre d'evenements encore emissibles par cette passe.
	reste int
	// rejetes compte les deroulages refuses par la borne par pas (compte EXACT, jamais tronque).
	rejetes int
	// tronque dit que le solde a ete epuise : la passe n'emet plus rien du tout ensuite.
	tronque bool
	// journalises compte les avertissements DETAILLES deja emis (cf. maxRejectLogs).
	journalises int
}

// newEventBudget ouvre le budget d'une passe sur un film.
func newEventBudget(origine string) *eventBudget {
	return &eventBudget{origine: origine, reste: maxNamedEventsPerFilm}
}

// rejeter enregistre un deroulage hors borne et le journalise (detail borne, compte exact).
func (b *eventBudget) rejeter(key statSlotKey, p ScorePoint, n int64) {
	b.rejetes++
	if b.journalises >= maxRejectLogs {
		return
	}
	b.journalises++
	slog.Warn("objectiveevents: deroulage aberrant rejete (dernier rempart memoire)",
		"passe", b.origine, "comp", key.Comp, "cote", key.Side, "slot", p.Slot,
		"time_ms", p.TimeMS, "deroulage", n, "borne", maxUnrollPerStep)
}

// epuiser marque le solde consomme. Une seule ligne de journal : les appels suivants sortent
// immediatement, il n'y a rien de nouveau a dire a chacun d'eux.
func (b *eventBudget) epuiser(key statSlotKey, p ScorePoint, n int64) {
	b.tronque = true
	slog.Warn("objectiveevents: plafond d'evenements du film atteint, deroulage interrompu",
		"passe", b.origine, "comp", key.Comp, "cote", key.Side, "slot", p.Slot,
		"time_ms", p.TimeMS, "deroulage", n, "reste", b.reste, "plafond", maxNamedEventsPerFilm)
}

// resume publie, en fin de passe, ce que les bornes ont coute. SILENCIEUX quand elles n'ont
// rien refuse — c'est le cas des neuf films sains du corpus d'equivalence, et c'est la seule
// facon pour qu'une ligne de journal signifie encore quelque chose quand elle apparait.
func (b *eventBudget) resume() {
	if b.rejetes == 0 && !b.tronque {
		return
	}
	slog.Warn("objectiveevents: bornes de deroulage appliquees sur ce film",
		"passe", b.origine, "deroulages_rejetes", b.rejetes, "tronque", b.tronque,
		"evenements_emis", maxNamedEventsPerFilm-b.reste, "plafond", maxNamedEventsPerFilm)
}

// boundedStep est UN point d'une suite cumulee, avec le deroulage qu'il demande et le verdict
// de la borne par pas. C'est l'unite de la DERIVATION UNIQUE des increments filtres.
type boundedStep struct {
	// Point est l'emission telle qu'elle sort de [cumulateRounds] / [longestRun].
	Point ScorePoint
	// Unroll est le deroulage BRUT demande par ce point (`p.Value - prev`). ZERO quand le
	// point est un palier : il ne fait pas avancer le compteur.
	Unroll int64
	// Kept est ce que la borne RETIENT : `Unroll`, ou zero quand elle le refuse.
	Kept int64
}

// rejected dit que ce point demandait des unites et que la borne les a toutes refusees.
func (s boundedStep) rejected() bool { return s.Unroll > 0 && s.Kept == 0 }

// boundSteps est LA SEULE LECTURE DE [maxUnrollPerStep] DU DEPOT — garde-rail :
// `named_derivation_unique_test.go`. C'est le correctif 6.R (2026-09-11).
//
// # POURQUOI UNE SEULE
//
// Le meme compteur etait derive DEUX fois : la CLE d'appariement le lisait par
// [incrementTimes] (borne appliquee), la SERIE publiee par [SeriesTotal] / [SeriesByRound]
// (borne absente). Sur `c0a82e88` le slot 12 deroulait 60 assistances d'un coup la ou sa
// feuille en porte zero : la cle voyait 0 et nommait le joueur, la courbe de score servait 60
// a l'ecran. Une seule derivation supprime la classe entiere de ces ecarts — les deux lecteurs
// ne peuvent plus diverger puisqu'ils lisent le meme verdict.
//
// # CE QU'ELLE FAIT, MOT POUR MOT COMME AVANT
//
// `prev` ne redescend JAMAIS, sinon la meme unite se compte deux fois apres un creux. Sans
// cela, une seule emission aberrante a -115 faisait remonter le compteur de 0 a 1 en **116**
// evenements (mesure sur `1bc77d2e`, slot 24, comp 0 A). Les emissions negatives elles-memes
// sont ecartees plus tot, par [seriesBySlot].
//
// Un deroulage au-dela de [maxUnrollPerStep] est REJETE : le point ne rend rien et `prev`
// avance quand meme a sa valeur — sinon le point SUIVANT rejouerait le meme ecart geant, et
// la borne n'aurait fait que deplacer l'explosion d'un cran.
//
// TOUS les points d'entree ressortent, paliers compris : c'est ce qui permet a
// [boundedSeries] de rendre une suite de MEME cardinalite que celle qu'elle assainit, donc de
// ne jamais faire disparaitre un compteur reste a zero.
func boundSteps(pts []ScorePoint) []boundedStep {
	out := make([]boundedStep, 0, len(pts))
	prev := int64(0)
	for _, p := range pts {
		s := boundedStep{Point: p}
		if n := p.Value - prev; n > 0 { // >= 0 : les negatives sont ecartees par [seriesBySlot]
			prev = p.Value
			s.Unroll = n
			if n <= maxUnrollPerStep {
				s.Kept = n
			}
		}
		out = append(out, s)
	}
	return out
}

// boundedSeries rend la suite cumulee des unites RETENUES : memes instants, meme cardinalite,
// valeurs recalculees comme le cumul des pas acceptes par [boundSteps].
//
// C'est la forme SERIE de la derivation unique — [incrementTimes] en est la forme EVENEMENTS.
// Un pas rejete ne laisse aucune trace dans la valeur publiee : la serie finit donc exactement
// sur le compte que la cle d'appariement a lu.
//
// LE REJET N'EST PAS JOURNALISE ICI, et ce n'est pas une erreur avalee : la meme table
// d'enregistrements passe par [SlotIdentityFrom] a chaque construction d'artefact, et sa passe
// `slot_identity` journalise deja chaque deroulage refuse sur ces trois memes emplacements
// (cf. [eventBudget.rejeter]). Une seconde ligne par serie ne dirait rien de neuf et noierait
// la premiere.
func boundedSeries(pts []ScorePoint) []ScorePoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]ScorePoint, 0, len(pts))
	var total int64
	for _, s := range boundSteps(pts) {
		total += s.Kept
		out = append(out, ScorePoint{TimeMS: s.Point.TimeMS, Slot: s.Point.Slot, Value: total})
	}
	return out
}

// incrementTimes rend un instant par UNITE gagnee par le compteur : c'est la conversion
// d'un compteur en evenements, et la forme EVENEMENTS de la derivation unique de
// [boundSteps]. `key` ne sert qu'au journal des bornes ; `b` porte le solde d'evenements de
// la passe et n'est jamais nil (cf. [eventBudget]).
//
// La premiere valeur observee est comptee depuis zero — un compteur de recompense part de
// zero au coup d'envoi. Si le film ne montre le slot qu'apres coup, les unites deja
// acquises sont datees de cette premiere emission, ce qui MAJORE leur instant.
//
// # Les deux bornes (lot 4b) — cf. l'en-tete des constantes de ce fichier
//
// La borne par PAS vit dans [boundSteps] ; celle du TOTAL vit ici, parce qu'elle ne borne que
// ce qui est materialise en memoire : le solde de la passe est consomme au fur et a mesure, et
// quand un deroulage n'y tient plus la passe est TRONQUEE et n'emet plus rien, ici comme dans
// ses appels suivants.
func incrementTimes(pts []ScorePoint, key statSlotKey, b *eventBudget) []int {
	if b.tronque {
		return nil
	}
	var out []int
	for _, s := range boundSteps(pts) {
		switch {
		case s.rejected():
			b.rejeter(key, s.Point, s.Unroll)
		case s.Kept == 0:
			continue // palier : le compteur n'avance pas
		case s.Kept > int64(b.reste):
			b.epuiser(key, s.Point, s.Kept)
			return out
		default:
			b.reste -= int(s.Kept)
			for i := int64(0); i < s.Kept; i++ {
				out = append(out, s.Point.TimeMS)
			}
		}
	}
	return out
}
