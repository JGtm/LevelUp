package objectiveevents

import (
	"context"
	"log/slog"
	"sort"
)

// statborg.go — le decodage des ENREGISTREMENTS D'ENTITE des paquets FRAME, d'ou sortent
// le score de mode, le score personnel et les compteurs de combat, horodates a la ms.
//
// # Ce qu'est un enregistrement
//
// Chaque entite du match (2 equipes + 8 joueurs) est serialisee dans les paquets FRAME par
// un enregistrement portant une liste CREUSE de composants : l'enregistrement DIT quels
// composants il transporte, on n'a donc pas a parcourir les 58 de l'archetype. Un composant
// n'est reemis QUE lorsqu'il change — c'est ce qui rend la resolution temporelle bien plus
// fine que le tick de 5 s du footer.
//
// # La grammaire, MESUREE (jamais derivee du binaire seul)
//
//	[1 bit = 1 : record DELTA][13 bits identifiant][2 bits generation][1 bit selecteur d'etat
//	de base ; si 1 : 7 bits][liste de composants]
//
//	liste, gate = 0 (creuse) : [1 bit = 0][3 bits N][N x 6 bits index]
//	liste, gate = 1 (DENSE)  : [1 bit = 1][64 bits masque]
//
//	puis, par index : [5 bits MANCHE][5 bits MANCHE][valeur A][valeur B][2 drapeaux][conditionnelles]
//
// Chaque constante a ete lue sur 1 078 en-tetes et 2 708 lectures de composant issus d'une
// capture Cheat Engine, pas supposee (.ai/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §15) :
//   - le bit qui precede l'identifiant vaut 1 dans 1 077/1 078 (c'est le code de record DELTA) ;
//   - les slots valent 6 et 8 (equipes) et 10..24 pairs (les 8 joueurs), soit
//     2 x (identifiant runtime - 0x40000000) ;
//   - la generation vaut 1 et le selecteur d'etat de base vaut 0 dans 1 077/1 078.
//
// # Ce que la description precedente disait de FAUX, et ce que cela coutait (2026-08-18)
//
// Elle decrivait « 14 bits de slot » puis « 2 bits constants a 10 » : ce sont en realite
// 13 bits d'identifiant, 2 bits de GENERATION et 1 bit de SELECTEUR D'ETAT DE BASE. La
// contrainte cachait donc deux hypotheses (generation = 1, selecteur = 0) — d'ou le slot
// toujours pair, qui vaut 2 x identifiant.
//
// Elle exigeait surtout les deux en-tetes de 5 bits du composant NULS. Or ces en-tetes portent
// le NUMERO DE MANCHE (0 = premiere manche), comme le getter natif le dit deja :
//
//	value = *(int32*)(world + slot*0x88 + equipe*0x1DF0 + 0x38 + manche*4)
//
// L'assertion « == 0 » etait donc un FILTRE DE MANCHE : tout match a plusieurs manches n'etait
// lu que jusqu'a la fin de la premiere. Mesure du 2026-08-18 (phase 0-ter du lot A) : en Oddball,
// le score d'equipe passe de 100/78 (une manche) a 200/121 (somme des manches) = l'oracle, sur
// 4 films sur 4 ; les frags passent de 48/88 a 385/391 (98,5 %). Un match peut avoir TROIS
// manches (`c88ec007`) : ne jamais cabler un nombre de manches.
//
// La forme DENSE de liste (gate = 1, masque de 64 bits) etait ignoree de meme : 33 records
// perdus sur `24dbb67d`. Le moteur y bascule quand un enregistrement change plus de sept
// composants — typiquement a la fin d'une manche.
//
// Controle de non-regression de ce relachement : sur 9 films sans manches multiples, les
// en-tetes non nuls representent 22 emissions sur 869 (2,5 %), toujours isolees et jamais
// groupees ; `530820e5` n'en a aucune.
//
// Piege consigne : les 2 bits qui PRECEDENT le bit de presence ne sont pas un champ de
// type. Ils sont statistiquement independants (22 co-occurrences pour 21,6 attendues sous
// independance) : c'est la queue de l'enregistrement precedent. Ne pas les contraindre.
//
// # L'horloge
//
// L'horodatage vient du `us` du paquet, recale par chunk sur le `start_ms` du manifeste.
// Les deux concordent a moins de 4 ms sur 573 s de film, et le footer (donc les
// ObjectiveEvent) est sur la meme base : tout est superposable sans recalage. Prendre pour
// origine le premier paquet OU L'ON TROUVE QUELQUE CHOSE au lieu du manifeste decale toute
// la courbe (140 s mesures sur un CTF).

const (
	// statIDBits est la largeur du champ de slot d'entite dans l'en-tete d'enregistrement.
	statIDBits = 14
	// statGenBits / statGenValue : les 2 bits qui suivent le slot, constants. Ils valent en
	// realite le second bit de la GENERATION et le SELECTEUR D'ETAT DE BASE (cf. en-tete) ;
	// la forme conservee ici est celle qui a ete calibree, et elle equivaut a exiger
	// generation = 1 et selecteur = 0.
	statGenBits  = 2
	statGenValue = 0b10
	// statHdrBits est la largeur de chacun des deux en-tetes d'un composant. Ils portent le
	// NUMERO DE MANCHE (0-based).
	statHdrBits = 5
	// statMaxRound borne le numero de manche accepte. Huit manches est au-dela de tout format
	// observe (le maximum mesure est 3, sur `c88ec007`) et la borne conserve 2 bits de
	// contrainte sur chacun des deux en-tetes, soit 4 des 10 bits de filtre anti-faux-positifs
	// de la forme d'origine. Sans borne, l'ancrage laisserait passer 151 faux positifs par film.
	statMaxRound = 7
	// statDenseMaskBits est la largeur du masque de la forme dense (gate = 1).
	statDenseMaskBits = 64
	// statMaxComp borne les index de composant de l'archetype (58 composants).
	statMaxComp = 58
	// statCompIndexBits est la largeur d'un index de composant dans la liste creuse.
	statCompIndexBits = 6
	// statMaxCompPerRecord : un enregistrement porte de 1 a 7 composants (compte sur
	// 3 bits ; mediane mesuree 2, jamais les 58 de l'archetype).
	statMaxCompPerRecord = 7
	// statSlotMin / statSlotMax / statTeamSlotMax delimitent les slots d'entite :
	// 6 et 8 pour les deux equipes, 10 a 24 (pairs) pour les huit joueurs.
	statSlotMin     = 6
	statSlotMax     = 24
	statTeamSlotMax = 8
	// statTailBits : marge de fin de paquet sous laquelle on n'ancre plus.
	statTailBits = 64
	// statMaxRecordsPerFilm plafonne ce qu'un film peut rendre.
	//
	// POURQUOI CE PLAFOND EXISTE : ce balayage a rendu la machine de l'utilisateur inutilisable
	// deux fois en aout 2026, et un film du corpus (`1b1e380f`, Strongholds) a atteint 3,3 Go
	// avant d'etre tue par une surveillance externe le 2026-08-18. Un decodeur qui vit dans un
	// service ne peut pas dependre d'une surveillance externe.
	//
	// La valeur est QUATRE FOIS le maximum mesure sur le corpus de 22 films de la phase 0
	// (8 269 enregistrements sur `c88ec007`, un Oddball a trois manches — le film le plus
	// bavard). Un film qui la depasse est anormal : on rend ce qui a ete lu, on le journalise
	// et on marque le resultat TRONQUE plutot que de gonfler jusqu'au plafond memoire.
	statMaxRecordsPerFilm = 4 * 8269
	// statMaxModeScore borne le score de MODE d une manche. Ce n est pas un reglage mais une
	// contrainte de DOMAINE : aucun mode Halo Infinite ne fait marquer plus de 250 points dans
	// une manche (Strongholds et Oddball plafonnent a 200, Slayer a 50, KOTH compte des manches).
	// Une valeur au-dela est un ancrage fortuit, et il faut l ecarter AVANT de qualifier les
	// manches : sur le CTF `53ce4390`, un point isole a 2 104 suffisait a faire passer une manche
	// fantome pour reelle et portait le score d equipe de 1 a 2 104.
	statMaxModeScore = 250
)

// StatValue est la paire de valeurs d'un composant. Le sens de A et de B depend du
// composant : le score de mode est en A, le score personnel en B (cf. score.go).
type StatValue struct{ A, B int64 }

// StatRecord est un enregistrement d'entite decode d'un paquet FRAME.
type StatRecord struct {
	// TimeMS est l'instant de l'emission sur l'horloge du film (meme base que le
	// TimeMS des ObjectiveEvent).
	TimeMS int
	// Slot identifie l'entite : 6 et 8 sont les deux equipes, 10..24 les huit joueurs.
	Slot int
	// Round est la MANCHE que decrit cet enregistrement, 0-based (0 = premiere manche). Elle
	// est lue dans les en-tetes de 5 bits du premier composant. Un compteur repart de zero a
	// chaque manche : le total d'un match est la SOMME des manches, jamais la derniere valeur.
	Round int
	// Comps porte les composants effectivement transportes par cet enregistrement.
	Comps map[int]StatValue
}

// IsTeamSlot dit si un slot designe une entite d'equipe (par opposition a un joueur).
func IsTeamSlot(slot int) bool { return slot <= statTeamSlotMax }

// StatRecords decode tous les enregistrements d'entite d'un film, tries par temps puis
// par slot. L'ancrage est DIRECT : les contraintes de l'en-tete suffisent a localiser un
// enregistrement, aucune traversee de la chaine n'est necessaire.
//
// Variante sans contexte, conservee pour les appelants existants : elle delegue a
// [StatRecordsCtx] et JETTE le drapeau de troncature. Tout appelant qui publie ce qu'il lit
// doit utiliser [StatRecordsCtx] et propager `truncated` — publier un score tronque sans le
// dire serait un mensonge silencieux.
func StatRecords(src FilmSource) []StatRecord {
	recs, _ := StatRecordsCtx(context.Background(), src, "")
	return recs
}

// StatRecordsCtx decode les enregistrements d'entite sous PLAFOND (cf. statMaxRecordsPerFilm).
// Il rend les enregistrements lus et `truncated` = true si le plafond a ete atteint : dans ce
// cas la lecture s'arrete la, elle est journalisee, et l'appelant doit le publier.
//
// matchID n'est utilise que pour le journal ; il peut etre vide.
func StatRecordsCtx(ctx context.Context, src FilmSource, matchID string) (recs []StatRecord, truncated bool) {
	var out []StatRecord
	for _, meta := range src.Chunks() {
		raw, ok := src.ChunkData(meta.Index)
		if !ok {
			continue
		}
		data := decompressChunk(raw)
		frames := walkFrames(data)
		if len(frames) == 0 {
			continue
		}
		base := frames[0].us
		for _, f := range frames {
			tMS := meta.StartMS + int((f.us-base)/1000)
			out = append(out, scanFrameForRecords(data[f.off:f.off+f.size], tMS)...)
			if len(out) >= statMaxRecordsPerFilm {
				slog.WarnContext(ctx,
					"statborg: plafond d'enregistrements atteint, lecture tronquee",
					"match_id", matchID, "records", len(out),
					"limite", statMaxRecordsPerFilm, "chunk", meta.Index)
				return sortRecords(out), true
			}
		}
	}
	return sortRecords(out), false
}

// sortRecords ordonne les enregistrements par temps puis par slot.
func sortRecords(out []StatRecord) []StatRecord {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Round < out[j].Round
	})
	return out
}

// scanFrameForRecords balaie un paquet FRAME et rend les enregistrements qu'il porte.
func scanFrameForRecords(pay []byte, tMS int) []StatRecord {
	var out []StatRecord
	lim := len(pay)*8 - statTailBits
	for b := 1; b < lim; b++ {
		slot, idx, at, ok := matchRecordHeader(pay, b)
		if !ok {
			continue
		}
		comps, round := decodeComponents(pay, at, idx)
		if len(comps) == 0 {
			continue
		}
		out = append(out, StatRecord{TimeMS: tMS, Slot: slot, Round: round, Comps: comps})
	}
	return out
}

// matchRecordHeader teste l'en-tete d'enregistrement d'entite a la position b. Il rend le
// slot, la liste creuse d'index de composants et la position du premier composant.
func matchRecordHeader(pay []byte, b int) (slot int, idx []int, compAt int, ok bool) {
	if readBitsBE(pay, b-1, 1) != 1 {
		return 0, nil, 0, false
	}
	slot = int(readBitsBE(pay, b, statIDBits))
	if slot < statSlotMin || slot > statSlotMax || slot%2 != 0 {
		return 0, nil, 0, false
	}
	if readBitsBE(pay, b+statIDBits, statGenBits) != statGenValue {
		return 0, nil, 0, false
	}
	m := b + statIDBits + statGenBits
	// Le moteur a DEUX formes de liste de composants (FUN_1406d7610) : creuse quand un
	// enregistrement change au plus sept composants, DENSE au-dela — typiquement a la fin
	// d'une manche, quand les 28 compteurs se figent et que les 28 suivants repartent.
	if readBitsBE(pay, m, 1) != 0 {
		idx, ok = denseComponentList(pay, m+1)
		return slot, idx, m + 1 + statDenseMaskBits, ok
	}
	n := int(readBitsBE(pay, m+1, 3))
	if n < 1 || n > statMaxCompPerRecord {
		return 0, nil, 0, false
	}
	// La liste creuse doit etre strictement croissante et bornee : c'est elle qui porte
	// l'essentiel de la contrainte dure.
	idx = make([]int, n)
	prev := -1
	for i := 0; i < n; i++ {
		idx[i] = int(readBitsBE(pay, m+4+statCompIndexBits*i, statCompIndexBits))
		if idx[i] >= statMaxComp || idx[i] <= prev {
			return 0, nil, 0, false
		}
		prev = idx[i]
	}
	return slot, idx, m + 4 + statCompIndexBits*n, true
}

// denseComponentList lit la forme DENSE : un masque de 64 bits dont le bit i designe le
// composant i. L'archetype n'ayant que [statMaxComp] composants, les bits de poids fort
// doivent etre nuls — c'est ce qui remplace ici la contrainte de la liste croissante.
func denseComponentList(pay []byte, p int) ([]int, bool) {
	if p+statDenseMaskBits > len(pay)*8 {
		return nil, false
	}
	mask := readBitsBE(pay, p, statDenseMaskBits)
	if mask == 0 || mask>>statMaxComp != 0 {
		return nil, false
	}
	idx := make([]int, 0, statMaxComp)
	for i := 0; i < statMaxComp; i++ {
		if mask>>uint(i)&1 == 1 {
			idx = append(idx, i)
		}
	}
	return idx, true
}

// decodeComponents lit les composants listes, dans l'ordre, et rend la MANCHE que porte
// l'enregistrement.
//
// Le PREMIER composant porte la contrainte qui separe un vrai enregistrement d'un ancrage
// fortuit : ses deux en-tetes de 5 bits doivent designer la MEME manche, bornee par
// [statMaxRound]. Exiger la manche ZERO — ce que faisait la version d'avant le 2026-08-18 —
// revenait a jeter tout ce qui suit la premiere manche. Exiger l'egalite des deux en-tetes est
// ce qui recupere la contrainte perdue : sur les emissions reelles ils portent toujours la meme
// valeur, un couple depareille est un faux positif.
//
// Les composants suivants ne sont pas re-contraints — leurs largeurs sont chainees, une lecture
// qui derape s'arrete d'elle-meme.
func decodeComponents(pay []byte, at int, idx []int) (map[int]StatValue, int) {
	h1 := int(readBitsBE(pay, at, statHdrBits))
	h2 := int(readBitsBE(pay, at+statHdrBits, statHdrBits))
	if h1 != h2 || h1 > statMaxRound {
		return nil, 0
	}
	out := make(map[int]StatValue, len(idx))
	q := at
	for _, i := range idx {
		v, w, ok := decodeStatComponent(pay, q)
		if !ok {
			break
		}
		out[i] = v
		q += w
	}
	return out, h1
}

// decodeStatComponent lit un composant et rend sa largeur consommee. Reproduit
// FUN_140C18794 : deux en-tetes de 5 bits, deux valeurs a longueur variable, deux drapeaux
// commandant chacun une valeur conditionnelle.
func decodeStatComponent(pay []byte, p int) (StatValue, int, bool) {
	q := p + 2*statHdrBits
	a, n1, ok := readStatVarWidth(pay, q)
	if !ok {
		return StatValue{}, 0, false
	}
	b, n2, ok := readStatVarWidth(pay, q+n1)
	if !ok {
		return StatValue{}, 0, false
	}
	q += n1 + n2
	if q+2 > len(pay)*8 {
		return StatValue{}, 0, false
	}
	flags := [2]uint64{readBitsBE(pay, q, 1), readBitsBE(pay, q+1, 1)}
	q += 2
	for _, f := range flags {
		if f != 1 {
			continue
		}
		_, n, ok := readStatVarWidth(pay, q)
		if !ok {
			return StatValue{}, 0, false
		}
		q += n
	}
	return StatValue{A: a, B: b}, q - p, true
}

// readStatVarWidth reproduit le lecteur a longueur variable FUN_140C18A1C : un selecteur
// de 2 bits donne la largeur (8 << selecteur), la valeur est signee.
func readStatVarWidth(pay []byte, p int) (int64, int, bool) {
	if p < 0 || p+2 > len(pay)*8 {
		return 0, 0, false
	}
	w := 8 << uint(readBitsBE(pay, p, 2))
	if w > 32 || p+2+w > len(pay)*8 {
		return 0, 0, false
	}
	v := readBitsBE(pay, p+2, w)
	iv := int64(v)
	if w < 32 && v&(1<<uint(w-1)) != 0 {
		iv = int64(v) - (1 << uint(w))
	}
	return iv, 2 + w, true
}

// statMinRoundRun separe une MANCHE REELLE d une manche fantome.
//
// Le relachement de l'assertion d'en-tete (cf. en-tete de fichier) laisse passer un residu de
// faux positifs : ils portent un numero de manche quelconque et arrivent ISOLES. Les cumuler
// comme s'il s'agissait de manches ferait exploser les compteurs — mesure du 2026-08-18 :
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
func RealRounds(recs []StatRecord) map[int]bool {
	type key struct{ slot, round int }
	series := map[key][]ScorePoint{}
	for _, r := range recs {
		v, ok := r.Comps[modeScoreComp]
		if !ok || !modeScoreInDomain(v) {
			continue
		}
		k := key{r.Slot, r.Round}
		series[k] = append(series[k], ScorePoint{TimeMS: r.TimeMS, Slot: r.Slot, Value: v.A})
	}
	runs := map[int]int{}
	for k, pts := range series {
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].TimeMS < pts[j].TimeMS })
		if n := len(longestRun(pts, true)); n > runs[k.round] {
			runs[k.round] = n
		}
	}
	return contiguousRounds(runs, materialRounds(recs))
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
// `analysis/replay/assaut_manches_research_test.go`, qui REFUSE toute manche du corpus libre
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
func materialRounds(recs []StatRecord) map[int]bool {
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
func contiguousRounds(runs map[int]int, material map[int]bool) map[int]bool {
	out := map[int]bool{}
	gap := 0
	for round := 0; round <= statMaxRound; round++ {
		if runs[round] >= statMinRoundRun || material[round] {
			gap = 0
			out[round] = true
			continue
		}
		gap++
		if gap > statMaxEmptyRoundRun || !hasRoundAfter(runs, material, round) {
			break
		}
		out[round] = true // manche courte, mais une manche coherente la suit encore
	}
	// La premiere manche existe toujours : un film tres court, ou tronque par le plafond,
	// reste lisible.
	if len(out) == 0 {
		out[0] = true
	}
	return out
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
