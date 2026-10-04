package objectives

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/signaux"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// statborg.go — ce que les ENREGISTREMENTS D'ENTITE du statborg veulent dire : les manches
// reelles d'un film, la frontiere joueur / equipe, et ce que la lecture a constate.
//
// LA LECTURE EST CELLE DE LA GRAMMAIRE ([signaux.LireLeStatborg], `grammar/signaux/statborg.go` : la
// grammaire mesuree de l'en-tete, des listes de composants et de l'horloge). Ce paquet ne lit aucun
// octet (ADR 0037, D-2 amende) : il recoit les enregistrements, porte les comptes des deux replis
// de la lecture dans [ComptesDesReplis] jusqu'au versement, et signale ses constats a
// l'orchestrateur.

// statMaxModeScore borne le score de MODE d une manche. Ce n est pas un reglage mais une
// contrainte de DOMAINE : aucun mode Halo Infinite ne fait marquer plus de 250 points dans
// une manche (Strongholds et Oddball plafonnent a 200, Slayer a 50, KOTH compte des manches).
// Une valeur au-dela est un ancrage fortuit, et il faut l ecarter AVANT de qualifier les
// manches : sur le CTF `53ce4390`, un point isole a 2 104 suffisait a faire passer une manche
// fantome pour reelle et portait le score d equipe de 1 a 2 104.
const statMaxModeScore = 250

// statMaxRound borne le numero de manche que le format peut porter ([signaux.StatborgMancheMax]) :
// la contiguite des manches ([contiguousRounds]) ne cherche pas au-dela.
const statMaxRound = signaux.StatborgMancheMax

// IsTeamSlot dit si un slot designe une entite d'equipe (par opposition a un joueur).
func IsTeamSlot(slot int) bool { return slot <= signaux.StatborgSlotEquipeMax }

// StatRecords decode tous les enregistrements d'entite d'un film, tries par temps puis
// par slot. L'ancrage est DIRECT : les contraintes de l'en-tete suffisent a localiser un
// enregistrement, aucune traversee de la chaine n'est necessaire.
//
// Variante des outils, conservee pour les appelants existants : elle delegue a
// [StatRecordsBornes] SANS recueillir de diagnostics et JETTE le drapeau de troncature. Tout
// appelant qui publie ce qu'il lit doit utiliser [StatRecordsBornes] et propager `truncated` —
// publier un score tronque sans le dire serait un mensonge silencieux.
func StatRecords(film *source.Film) []types.StatRecord {
	recs, _, _ := StatRecordsBornes(film, "")
	return recs
}

// StatRecordsBornes decode les enregistrements d'entite sous PLAFOND
// ([signaux.StatborgEnregistrementsMax]). Il rend les enregistrements lus et `truncated` = true si
// le plafond a ete atteint : la lecture s'arrete la, le dit dans `diags`, et l'appelant doit le
// publier. Ce paquet ne journalise pas (ADR 0034 D-4) : l orchestrateur journalise `diags` avec
// SON contexte. matchID n'est utilise que pour les diagnostics ; il peut etre vide.
func StatRecordsBornes(film *source.Film, matchID string) (
	recs []types.StatRecord, truncated bool, diags []constat.Diagnostic,
) {
	recs, truncated, _, diags = StatRecordsAvecReplis(film, matchID)
	return recs, truncated, diags
}

// StatRecordsAvecReplis est [StatRecordsBornes], plus les comptes des deux replis de la lecture —
// enregistrements abandonnes et composants arretes (lot J8.7). La cuisson les porte avec la section
// statborg des faits persistes, et les verse au compteur a l assemblage.
func StatRecordsAvecReplis(film *source.Film, matchID string) (
	recs []types.StatRecord, truncated bool, replis ComptesDesReplis, diags []constat.Diagnostic,
) {
	l := signaux.LireLeStatborg(film)
	return l.Records, l.Tronque, comptesDuStatborg(l), diagnosticsDuStatborg(l, film, matchID)
}

// comptesDuStatborg porte les deux replis de la lecture dans les comptes de ce paquet, champ pour
// champ : c est sous ces deux noms que la cuisson les persiste et les verse.
func comptesDuStatborg(l signaux.LectureDuStatborg) ComptesDesReplis {
	return ComptesDesReplis{
		EnregistrementsAbandonnes: l.EnregistrementsAbandonnes,
		ComposantsArretes:         l.ComposantsArretes,
	}
}

// diagnosticsDuStatborg rend ce que la lecture constate, dans l ordre ou elle le rencontre : un
// film dont aucun chunk n est decrit par le manifeste (rien n y est datable : se taire ferait lire
// « ce film ne porte rien » la ou il faut lire « on ne sait pas dater ce film »), puis une lecture
// tronquee par le plafond.
func diagnosticsDuStatborg(l signaux.LectureDuStatborg, film *source.Film, matchID string) []constat.Diagnostic {
	var diag constat.Diagnostics
	if l.ChunksDatables == 0 {
		diag.Signaler(constat.Diagnostic{Code: DiagFilmSansManifeste, Niveau: constat.NiveauInfo,
			Message: "objectives: film sans chunk decrit par le manifeste — rien a dater",
			Attrs:   []any{"match_id", matchID, "chunks_du_film", filmChunkCount(film)}})
	}
	if l.Tronque {
		diag.Signaler(constat.Diagnostic{Code: DiagStatborgTronque, Niveau: constat.NiveauWarn,
			Message: "statborg: plafond d'enregistrements atteint, lecture tronquee",
			Attrs: []any{"match_id", matchID, "records", len(l.Records),
				"limite", signaux.StatborgEnregistrementsMax, "chunk", l.ChunkTronque}})
	}
	return diag.Relever()
}

// filmChunkCount rend le nombre de chunks du film, ou 0 pour un film absent — de quoi dire au
// journal si « aucun chunk du manifeste » veut dire « pas de film » ou « pas de manifeste ».
func filmChunkCount(film *source.Film) int {
	if film == nil {
		return 0
	}
	return film.NumChunks()
}

// statMinRoundRun separe une MANCHE REELLE d une manche fantome.
//
// Le relachement de l'assertion d'en-tete (cf. l'en-tete de `grammar/signaux/statborg.go`) laisse passer
// un residu de faux positifs : ils portent un numero de manche quelconque et arrivent ISOLES. Les
// cumuler comme s'il s'agissait de manches ferait exploser les compteurs — mesure du 2026-08-18 :
// `flag_capture_assists` passait de 1 a 1 569 sur `1bc77d2e` avant l'introduction de ce seuil.
//
// La valeur separe deux populations mesurees sur le corpus de 22 films : un ancrage fortuit
// arrive ISOLE, et la plus petite manche REELLE observee tire 33 emissions coherentes
// (`c88ec007`, slot 6, manche 1). Trois est le plus petit seuil qui ecarte les paires fortuites
// sans jeter une manche courte — c est celui qui a ete valide en mesure (phase 0-ter).
const statMinRoundRun = 3

// RealRounds rend les manches d'un film qui sont vraiment des manches.
//
// Deux conditions, et il faut les DEUX — la premiere seule ne suffisait pas (mesure du
// 2026-08-18 : sur le CTF `53ce4390`, une manche fantome franchissait le seuil de comptage et le
// cumul portait le score d'equipe de 1 a 2 104) :
//
//	coherence   la manche tire, pour au moins un slot, une suite croissante d au moins
//	            [statMinRoundRun] emissions du score de mode ;
//	contiguite  les manches se jouent dans l'ordre, donc seules 0, 1, 2 ... sans trou sont
//	            retenues. Une « manche 5 » sans manche 1 a 4 est un ancrage fortuit, quel que
//	            soit son comptage.
//
// Le comptage porte sur une SUITE COHERENTE, pas sur des enregistrements bruts : pour chaque
// couple (slot, manche), la suite du score de mode est filtree par la meme plus longue
// sous-suite croissante que la production, et la manche doit en tirer au moins
// [statMinRoundRun] emissions pour au moins un slot. C'est le critere qui a ete valide en
// mesure : 4 films Oddball sur 4 exacts, et aucun faux positif sur les 9 films a une manche.
// Compter les enregistrements bruts ne suffisait pas — sur le CTF `53ce4390`, une manche
// fantome franchissait ce comptage et portait le score d'equipe de 1 a 2 104.
//
// ELLE DELEGUE A [ResolveRounds] DEPUIS LE LOT 1.9.11 et n'en garde que l'ensemble : le verdict
// COMPLET — ce que le film a ecrit, ce que l'ordre a CONTREDIT, et le decret de la manche 0 —
// se lit la, et c'est lui que l'artefact publie.
func RealRounds(recs []types.StatRecord) map[int]bool {
	return ResolveRounds(recs).RealSet()
}

// modeScoreRunsByRound rend, par manche, la plus longue suite STRICTEMENT croissante du score de
// mode, tous slots confondus — le premier des deux criteres d'admission.
func modeScoreRunsByRound(recs []types.StatRecord) map[int]int {
	type key struct{ slot, round int }
	series := map[key][]types.ScorePoint{}
	for _, r := range recs {
		v, ok := r.Comps[modeScoreComp]
		if !ok || !modeScoreInDomain(v) {
			continue
		}
		k := key{r.Slot, r.Round}
		series[k] = append(series[k], types.ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
	}
	runs := map[int]int{}
	for k, pts := range series {
		slices.SortStableFunc(pts, func(a, b types.ScorePoint) int { return cmp.Compare(a.TimeMS, b.TimeMS) })
		if n := len(longestRun(pts, true)); n > runs[k.round] {
			runs[k.round] = n
		}
	}
	return runs
}

// presentRounds rend les manches qui EXISTENT dans le film : au moins UN enregistrement, de
// joueur ou d'equipe.
//
// Les slots d'equipe comptent ici, contrairement a [materialRounds] : la question n'est pas
// « cette manche a-t-elle assez de matiere pour etre une manche » mais « cette manche
// a-t-elle laisse la moindre trace ». Une manche dont pas un seul enregistrement ne porte le
// numero n'a pas ete jouee — elle n'est meme pas courte, elle est absente.
func presentRounds(recs []types.StatRecord) map[int]bool {
	out := make(map[int]bool, 4)
	for _, r := range recs {
		out[r.Round] = true
	}
	return out
}

// statMinRoundRecordShare : la part MINIMALE, en pour cent, des enregistrements de slot
// JOUEUR de la manche la plus fournie du film qu'une manche doit porter pour etre MATERIELLE.
//
// # Pourquoi un SECOND critere d'admission, et pourquoi celui-ci
//
// Le critere de suite coherente ne peut PAS etre tenu par une manche d'Assaut One Bomb : une
// manche y porte au plus UNE emission de score (un point de mode = une explosion, releve A0.3
// fige au protocole du lot A), donc sa plus longue suite strictement croissante vaut 2 — sous
// [statMinRoundRun]. Mesure du 2026-08-31 : sur `df8fcbef`, `c75f33b8` et `9f57c612`, seule la
// manche 0 survivait et 8 explosions sur 11 etaient perdues, alors que le releve BRUT somme au
// score de l'API sur 9 films sur 9.
//
// Ce qui separe VRAIMENT une manche jouee d'un ancrage fortuit, c'est la MATIERE : une manche
// jouee fait emettre tous les slots de joueur pendant toute sa duree ; un faux positif de
// l'assertion d'en-tete arrive isole. La part est prise RELATIVEMENT a la manche la plus
// fournie du meme film — ce denominateur est toujours une manche reelle, donc la mesure ne
// depend d'aucune constante de duree, de cadence, ni de nombre de joueurs (un FFA a 6 joueurs
// passe comme un 4v4).
//
// # Les deux populations, mesurees sur 65 films et 227 manches brutes
//
// Corpus de recherche (12 films : 3 One Bomb a verite connue, 6 Assaut mono-manche, les DEUX
// contre-exemples documentes `53ce4390` et `1bc77d2e`, un Oddball a manches courtes) et corpus
// de CONTROLE HORS ECHANTILLON (53 films echantillonnes PAR MODE : Slayer, Fiesta, BTB, Husky
// Raid, Firefight, Oddball, CTF, KOTH, Strongholds, variantes communautaires) :
//
//	ancrage fortuit   part <= 5,84 %   (`bfcd1175` manche 6 : 18/308 — un film de Slayer,
//	                                   mode qui n'a pas de manche)
//	manche reelle     part >= 21 %     (`df8fcbef` manche 1 : 45/212)
//
// DIX pour cent se pose au milieu de ce vide : 1,7 fois au-dessus du plus gros ancrage observe,
// 2,1 fois sous la plus maigre manche reelle. Instrument et controle :
// `games/halo_infinite/film/replay/assaut_manches_research_test.go`, qui REFUSE toute manche du corpus libre
// dans la bande 7 %..15 %.
const statMinRoundRecordShare = 10

// statMinRoundRecords : le PLANCHER ABSOLU du second critere, en enregistrements de slot joueur.
//
// La part seule serait piegeuse sur un film TRES pauvre : trois enregistrements en manche 0 et
// un en manche 1 font 33 %, et l'ancrage passerait. Le plancher ferme cette porte sans rien
// coûter aux vrais films — le plus maigre du corpus de mesure (`69b16f5d`) porte deja 306
// enregistrements. Les deux extremes mesures l'encadrent : plus gros ancrage observe 18
// (`bfcd1175` manche 6), plus maigre manche reelle 45 (`df8fcbef` manche 1).
//
// LES DEUX CONDITIONS SONT EXIGEES ENSEMBLE. Le second critere ne peut donc qu'AJOUTER des
// manches a ce que le premier retenait deja, jamais en retirer.
const statMinRoundRecords = 25

// materialRounds rend les manches MATERIELLES : celles qui portent au moins
// [statMinRoundRecords] enregistrements de slot JOUEUR ET au moins [statMinRoundRecordShare] %
// de ceux de la manche la plus fournie.
//
// Les slots d'EQUIPE sont exclus du comptage : ils emettent sur un rythme propre, independant
// du nombre de joueurs, et deux slots suffiraient a faire passer un ancrage.
func materialRounds(recs []types.StatRecord) map[int]bool {
	parRound := map[int]int{}
	for _, r := range recs {
		if IsTeamSlot(r.Slot) {
			continue
		}
		parRound[r.Round]++
	}
	ref := 0
	for _, n := range parRound {
		if n > ref {
			ref = n
		}
	}
	out := map[int]bool{}
	if ref == 0 {
		return out
	}
	for round, n := range parRound {
		if n >= statMinRoundRecords && n*100 >= ref*statMinRoundRecordShare {
			out[round] = true
		}
	}
	return out
}

// statMaxEmptyRoundRun : combien de manches SANS suite coherente la contiguite tolere d'affilee.
//
// LE ZERO ETAIT UN BUG (revue R1, 2026-08-18). La contiguite sortait au PREMIER trou : une
// premiere manche trop courte pour tirer [statMinRoundRun] emissions coherentes — un camp qui
// s'effondre en quelques secondes — faisait perdre TOUTES les manches suivantes, y compris
// completes, et le match retombait sur la seule manche 1 de repli.
//
// UN est le plus petit reglage qui repare cela sans rouvrir ce que la contiguite ferme : une
// manche courte n'a pas de suite coherente, cinq d'affilee n'existent pas. Le controle negatif
// tient toujours — une « manche 5 » sans les manches 1 a 4 est ecartee, elle laisse quatre
// manches vides d'affilee.
const statMaxEmptyRoundRun = 1

// contiguousRounds applique la contiguite : les manches se jouent DANS L'ORDRE, donc la suite
// retenue part de zero et s'arrete des qu'il n'y a plus rien de coherent apres.
//
// Une manche sans suite coherente est CONSERVEE quand une manche coherente la suit encore (elle
// a bien ete jouee, elle a seulement ete courte) et que le trou ne depasse pas
// [statMaxEmptyRoundRun]. Sinon on s'arrete : ce qui suit est du bruit.
//
// DEUX CRITERES D'ADMISSION, en OU : la suite coherente du score de mode ([statMinRoundRun])
// OU la matiere de la manche ([statMinRoundRecordShare]). Le second existe pour l'Assaut One
// Bomb, ou une manche ne porte qu'UNE emission de score et ne peut donc jamais tenir le
// premier — voir l'en-tete de [statMinRoundRecordShare] pour les deux populations mesurees.
//
// UNE MANCHE TOLEREE APRES UNE MANCHE ADMISE DOIT EXISTER (lot 6.7-B1, item 4, 2026-09-11). La
// tolerance ci-dessus est ecrite pour une manche JOUEE mais trop COURTE ; elle ne dit rien
// d'une manche qui n'a laisse AUCUN enregistrement. `e60aaf06` (Strongholds, une seule manche a
// son fil de score) declare les manches 0 et 2 et RIEN en manche 1 : la chaine sautait
// par-dessus le vide et atteignait la manche 2, un ancrage de 44 enregistrements (14 % de la
// manche 0, donc MATERIEL au sens de [statMinRoundRecordShare]) dont l'intervalle tombe
// ENTIEREMENT dans celui de la manche 0. Le film publiait alors trois manches, l'identite se
// resolvait PAR MANCHE, et 130 de ses 154 actions partaient en `noSlot` faute d'identite dans
// une manche qui n'existe pas.
//
// EN TETE DE CHAINE, EN REVANCHE, UNE MANCHE ABSENTE RESTE TOLEREE : un film qui numerote ses
// manches a partir de 1 ne declare rien en manche 0, et ce n'est pas un trou mais un DECALAGE
// de numerotation. La distinction se fait sur `vue` : tant qu'aucune manche n'a ete ADMISE par
// l'un des deux criteres, l'absence ne prouve rien.
//
// # CE QUE LE LOT 1.9.11 A MESURE, ET POURQUOI CETTE REGLE RESTE (2026-09-16)
//
// L'item 1.9.11 du PLAN_DECODEUR_FILM devait la RETIRER : « le film porte le compteur de
// manche » (decision utilisateur du 2026-09-14). La mesure sur les 1 351 films du cache
// (instrument `e1911_manches_*_research_test.go`) a REFUTE l'hypothese, et le detail est au
// §5 du plan. En resume : retirer cette regle ajoute une manche a 24 films, TOUS du motif
// « designateur 2, manche 1 absente », et 23 des 24 ont fini de 38 a 442 s DANS leur temps
// reglementaire sur des modes SANS manche (Team Slayer, Slayer, Strongholds, Husky Raid) —
// une seconde manche y est impossible. Les VRAIES prolongations, elles, le film les ecrit en
// designateur 1 CONTIGU et cette chaine les publie deja (14 films CTF:Arena au-dela du temps
// reglementaire, 13 a egalite au debut de la manche, ZERO lecture `finalized-rounds-values`
// quand la population fantome en porte jusqu'a la moitie).
//
// LA REGLE N'EST DONC PLUS UN DECRET, C'EST LE SEUL CRITERE QUI SEPARE LES DEUX POPULATIONS
// SUR LE CORPUS MESURE (24 fantomes refuses, 20 manches reelles admises). Les deux autres
// candidats ont ete mesures et ECARTES : le CONSENSUS DE SLOTS ne separe rien (les 41
// designateurs materiels du corpus de verdict sont declares par les DIX slots, part 100 %), et
// la part de `finalized-rounds-values` n'attrape que 19 des 24 fantomes.
//
// CE QU'ELLE DOIT EN REVANCHE, ET QUE LE LOT AJOUTE : un designateur ECRIT, MATERIEL et refuse
// par l'ordre est une CONTRADICTION au sens de D14 (c) — elle se COMPTE et se PUBLIE
// (`coverage.score.roundsContradicted`), elle ne se tait pas. Le seul REPLI de cette chaine est
// le decret de la manche 0 quand aucune n'est admise (`repli_manche_zero_decretee`).
func contiguousRounds(runs map[int]int, material, present map[int]bool) (map[int]bool, bool) {
	out := map[int]bool{}
	gap, vue := 0, false
	for round := 0; round <= statMaxRound; round++ {
		if runs[round] >= statMinRoundRun || material[round] {
			gap, vue = 0, true
			out[round] = true
			continue
		}
		gap++
		if gap > statMaxEmptyRoundRun || (vue && !present[round]) ||
			!hasRoundAfter(runs, material, round) {
			break
		}
		out[round] = true // manche courte, mais une manche coherente la suit encore
	}
	// La premiere manche existe toujours : un film tres court, ou tronque par le plafond,
	// reste lisible. C'est le REPLI `repli_manche_zero_decretee`, et il se compte.
	if len(out) == 0 {
		out[0] = true
		return out, true
	}
	return out, false
}

// hasRoundAfter dit s'il reste une manche ADMISE STRICTEMENT apres celle-ci — par l'un ou
// l'autre des deux criteres, comme la boucle de [contiguousRounds].
func hasRoundAfter(runs map[int]int, material map[int]bool, round int) bool {
	for r := round + 1; r <= statMaxRound; r++ {
		if runs[r] >= statMinRoundRun || material[r] {
			return true
		}
	}
	return false
}
