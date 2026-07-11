// Package temporal expose les algorithmes purs de mesure temporelle pour
// l'analyse Halo : moyennes mobiles, lissage LOWESS, bucketing, et le score
// d'engagement (engagement_score.go).
//
// Reference conceptuelle pour engagement : .ai/REFLEXION_ENGAGEMENT_SCORE_INTRA_MATCH.md
// Plan d'implementation : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md
package temporal

import (
	"errors"
	"math"
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// Erreurs sentinelles retournees par ComputeEngagementScore.
var (
	// ErrInsufficientData : le match n'a pas d'events exploitables.
	ErrInsufficientData = errors.New("engagement: insufficient event data")

	// ErrMatchTooShort : duree match < seuil (default 3 min).
	ErrMatchTooShort = errors.New("engagement: match too short")

	// ErrInvalidBoundaries : matchStartMS >= matchEndMS ou negatifs.
	ErrInvalidBoundaries = errors.New("engagement: invalid match boundaries")
)

// Constantes de calibration. Documentees dans la reflexion §6 et §16.
const (
	// DefaultWindowSeconds est la largeur de la fenetre glissante pour la
	// courbe d'engagement. 90s = bon compromis lecture vs reactivite (cf §16).
	DefaultWindowSeconds = 90

	// DefaultSamplingSeconds est l'echantillonnage de la courbe (10s par
	// defaut, soit 72 points pour un match de 12 min).
	DefaultSamplingSeconds = 10

	// PassiveDeathThresholdMS : duree minimale d'inaction avant une mort
	// pour qualifier celle-ci de "passive" (caught off-guard). 30s par defaut.
	PassiveDeathThresholdMS = 30_000

	// MinMatchDurationMS : duree minimale d'un match pour calculer un score.
	// En dessous, le signal est trop bruite pour etre exploitable.
	MinMatchDurationMS = 180_000 // 3 min

	// HistoryMinFull : seuil de matchs d'historique pour confiance "full".
	HistoryMinFull = 30

	// HistoryMinPartial : seuil minimal pour calculer un score (confidence
	// "partial" si entre ce seuil et HistoryMinFull).
	HistoryMinPartial = 10

	// ObjectivePointsPerCapture : poids unitaire utilise pour deriver les
	// "events objectifs estimes" depuis le personal_score (modes asymetriques).
	// (personal_score - 100*kills - 50*assists) / ObjectivePointsPerCapture
	// = nombre estime d'events objectif. A calibrer en post-validation H7.
	ObjectivePointsPerCapture = 25.0
)

// EngagementScoreInput regroupe les inputs explicites du calcul. Pas de
// couplage DB ou HTTP : la fonction est strictement pure.
//
// Tous les events doivent etre filtres en amont (par xuid pour PlayerEvents,
// xuid des coequipiers humains pour TeamEvents, xuid des humains du lobby
// pour LobbyEvents). Les bots ne doivent jamais figurer dans ces slices.
type EngagementScoreInput struct {
	PlayerEvents []canonical.HighlightEvent
	TeamEvents   []canonical.HighlightEvent
	LobbyEvents  []canonical.HighlightEvent

	NTeam        int // taille equipe alliee humains (joueur cible inclus)
	NHumansLobby int // taille lobby humains
	XUID         string

	MatchStartMS int64
	MatchEndMS   int64

	History []domain.HistoricalEngagementBrut

	// CoefLobbyShare est le coef lobby global du joueur (mediane historique de
	// pace_joueur/pace_lobby, toutes intensites confondues). Sert de fallback
	// (ExpectedBasis "global") quand aucun bin d'intensite n'est exploitable.
	// N'est pris en compte que si HasGlobalLobbyCoef est vrai.
	CoefLobbyShare float64

	// HasGlobalLobbyCoef indique que CoefLobbyShare provient d'un calcul reel
	// (>= MinMatchesForCoef echantillons), pas du defaut neutre cold-start.
	HasGlobalLobbyCoef bool

	// ResponseBins porte les coefs de reponse par bin d'intensite (terciles de
	// pace_lobby). Nil = pas de bins persistes. C'est la source prioritaire de
	// l'attendu (ExpectedBasis "bin"). cf engagement_response_bins.go.
	ResponseBins *domain.EngagementResponseBins

	// PersonalScore et Kills/Assists permettent de calculer les events
	// objectif estimes pour les modes asymetriques (CTF, Strongholds, Oddball).
	// = (PersonalScore - 100*Kills - 50*Assists) / ObjectivePointsPerCapture
	PersonalScore int
	Kills         int
	Assists       int

	// Mode est la categorie de mode (PvP_ranked, PvP_unranked, ...).
	Mode string

	// Weights porte les poids d'events PAR TITRE (chantier F7, DE-4). Zero-value →
	// DefaultEventWeights (byte-identique Infinite). Renseigne par le point de
	// collecte via games.EngagementWeightsFor(slug). Le moteur reste agnostic.
	Weights EventWeights

	// Signals est le vecteur de signaux d'engagement du match (masque de presence
	// + signaux riches optionnels), construit par le point de collecte title-owned
	// (cf. SignalsFromEvents). Gouverne la porte de suffisance (1re porte F7). Si
	// laisse a zero par l'appelant (tests legacy), ComputeEngagementScore le derive
	// de ses propres inputs. Les signaux riches ne modifient PAS le score en l'etat
	// (poids nul tant que non calibres, DE-5) : ils qualifient la confiance par match.
	Signals EngagementSignals

	// IsTeamMode indique si le mode a des equipes (vs FFA / 1v1). N'influe PLUS
	// sur l'attendu (ancre lobby unifiee, cf. D1) ; ne sert qu'a l'affichage de
	// la courbe « Equipe reelle » cote front.
	IsTeamMode bool

	// WindowSeconds et SamplingSeconds sont optionnels. Si 0, defauts utilises.
	WindowSeconds   int
	SamplingSeconds int
}

// ComputeEngagementScore est la fonction publique de calcul. Pure : pas
// d'acces DB, pas de log, pas de side-effect.
//
// Les exigences de regle arch-rules sont respectees : 0 import platform,
// 0 import port, 0 service.
//
// Erreurs sentinelles (toutes wrappables avec errors.Is) :
//   - ErrInvalidBoundaries  : matchStart/End incoherents
//   - ErrMatchTooShort      : match < 3 min
//   - ErrInsufficientData   : 0 event exploitable
//
// La confiance "insufficient_history" est portee par le champ Confidence
// de l'output (pas une erreur), pour permettre a l'UI d'afficher la courbe
// sans score numerique.
func ComputeEngagementScore(input EngagementScoreInput) (domain.EngagementScoreResult, error) {
	if err := validateBoundaries(input); err != nil {
		return domain.EngagementScoreResult{}, err
	}

	matchDurationMS := input.MatchEndMS - input.MatchStartMS
	if matchDurationMS < MinMatchDurationMS {
		return domain.EngagementScoreResult{}, ErrMatchTooShort
	}

	if len(input.PlayerEvents) == 0 && len(input.TeamEvents) == 0 {
		return domain.EngagementScoreResult{}, ErrInsufficientData
	}

	windowMS, samplingMS := resolveWindow(input)

	// Poids d'events par titre (F7) : ceux fournis, ou defaut byte-identique.
	weights := input.Weights
	if weights.IsZero() {
		weights = DefaultEventWeights()
	}

	// "Equipe reelle" = coequipiers + joueur cible. On inclut le joueur au
	// numerateur pour rester coherent avec NTeam (qui le compte au denominateur) :
	// pace_team = events de TOUTE l'equipe / NTeam. Sans ca, num = N-1 joueurs et
	// denom = N -> ligne equipe sous-evaluee + coef gonfle (cf. thought_log
	// 2026-06-18). En FFA/1v1 le denominateur reste le lobby (deja joueur inclus).
	teamInclPlayer := make([]canonical.HighlightEvent, 0, len(input.TeamEvents)+len(input.PlayerEvents))
	teamInclPlayer = append(teamInclPlayer, input.TeamEvents...)
	teamInclPlayer = append(teamInclPlayer, input.PlayerEvents...)

	// Construction de la courbe des paces (joueur / team / lobby). L'attendu
	// n'est PAS calcule ici : il depend de l'intensite moyenne du match (mean
	// pace_lobby), elle-meme derivee de la courbe. Il se pose en 2e passe.
	curve := buildEngagementCurve(buildCurveParams{
		PlayerEvents: input.PlayerEvents,
		TeamEvents:   teamInclPlayer,
		LobbyEvents:  input.LobbyEvents,
		NTeam:        input.NTeam,
		NHumansLobby: input.NHumansLobby,
		MatchStartMS: input.MatchStartMS,
		MatchEndMS:   input.MatchEndMS,
		WindowMS:     windowMS,
		SamplingMS:   samplingMS,
		Weights:      weights,
	})

	curve = annotateDeaths(curve, input.PlayerEvents, samplingMS)

	// Means agreges de la courbe — persistes dans player_match_enrichment pour
	// alimenter le recompute des coefficients/bins (cf. engagement_coefficients.go
	// + engagement_response_bins.go). meanLobby = intensite du match.
	meanJoueur, meanTeam, meanLobby := meansFromCurve(curve)

	// Attendu ancre lobby (2e passe) : coef (bin d'intensite / global / cold-start)
	// x pace_lobby(t). L'ancre est le lobby partout (D1) ; le bin est choisi via
	// meanLobby (intensite du match).
	coef, expectedBasis, intensityBin := resolveExpectedCoef(meanLobby, input)
	applyExpectedToCurve(curve, coef)

	residualBrut := computeResidualBrut(curve)
	matchIntensity := computeMatchIntensity(input.LobbyEvents, matchDurationMS, input.NHumansLobby)

	score, confidence, nHist := scoreFromHistory(residualBrut, input.History)

	// Porte de suffisance (1re porte F7). On prend le vecteur fourni par le point
	// de collecte title-owned ; a defaut (tests legacy), on le derive des inputs
	// (meme algorithme title-agnostic). Un resultat produit ici a toujours passe
	// les gardes ErrMatchTooShort/ErrInsufficientData -> au moins Partial.
	signals := input.Signals
	if signals.IsZero() {
		signals = SignalsFromEvents(input.PlayerEvents, input.LobbyEvents, matchDurationMS)
	}
	signalBasis := signals.Sufficiency().String()

	result := domain.EngagementScoreResult{
		EngagementScore: score,
		ResidualBrut:    residualBrut,
		EngagementCurve: curve,
		MatchIntensity:  matchIntensity,
		Confidence:      confidence,
		NHistoryMatches: nHist,
		MeanPaceJoueur:  meanJoueur,
		MeanPaceTeam:    meanTeam,
		MeanPaceLobby:   meanLobby,
		PlayerActivity:  input.Kills + input.Assists + countDeaths(input.PlayerEvents),
		ExpectedBasis:   expectedBasis,
		IntensityBin:    intensityBin,
		SignalBasis:     signalBasis,
	}
	return result, nil
}

// resolveExpectedCoef determine le coefficient de l'attendu ancre lobby selon la
// chaine de fallback (cf. plan §3) : bin d'intensite (>= MinMatchesForBin
// echantillons) → coef lobby global → cold-start (1.0). Retourne le coef, la
// base (ExpectedBasis) et le libelle du bin (vide hors basis "bin").
func resolveExpectedCoef(meanLobby float64, input EngagementScoreInput) (coef float64, basis, bin string) {
	if input.ResponseBins != nil {
		if b, ok := input.ResponseBins.ResolveBin(meanLobby); ok && b.NMatches >= MinMatchesForBin {
			return b.CoefLobby, domain.ExpectedBasisBin, b.Bin
		}
	}
	if input.HasGlobalLobbyCoef {
		return input.CoefLobbyShare, domain.ExpectedBasisGlobal, ""
	}
	return 1.0, domain.ExpectedBasisColdStart, ""
}

// applyExpectedToCurve pose PaceAttendu = coef x PaceLobby sur chaque point (2e
// passe de l'attendu ancre lobby). PaceLobby est deja le pace lobby per-player
// de la fenetre — invariant a la taille du lobby et coherent avec l'univers des
// coefficients.
func applyExpectedToCurve(curve []domain.EngagementPoint, coef float64) {
	for i := range curve {
		curve[i].PaceAttendu = coef * curve[i].PaceLobby
	}
}

// meansFromCurve calcule en un seul passage les means des 3 paces de la courbe.
// Renvoie (0,0,0) si la courbe est vide.
func meansFromCurve(curve []domain.EngagementPoint) (joueur, team, lobby float64) {
	n := len(curve)
	if n == 0 {
		return 0, 0, 0
	}
	var sJoueur, sTeam, sLobby float64
	for _, p := range curve {
		sJoueur += p.PaceJoueur
		sTeam += p.PaceTeam
		sLobby += p.PaceLobby
	}
	return sJoueur / float64(n), sTeam / float64(n), sLobby / float64(n)
}

// countDeaths compte les events death dans une liste d'events deja filtree
// pour le joueur cible. Sert a calculer PlayerActivity (= K+A+D).
func countDeaths(playerEvents []canonical.HighlightEvent) int {
	n := 0
	for _, e := range playerEvents {
		if e.EventType == string(canonical.EventDeath) {
			n++
		}
	}
	return n
}

// EventsObjectifEstimes derive un nombre estime d'events "objectif" depuis le
// personal_score, pour compenser les modes asymetriques ou un porteur d'objectif
// peut etre engage sans tuer.
//
// Formule : (personal_score - 100*kills - 50*assists) / ObjectivePointsPerCapture
//
// Negatif clampe a 0 (peut arriver si personal_score < 100*kills, par exemple
// trade defavorable + 0 objectif).
func EventsObjectifEstimes(personalScore, kills, assists int) float64 {
	residue := float64(personalScore) - 100.0*float64(kills) - 50.0*float64(assists)
	if residue <= 0 {
		return 0
	}
	return residue / ObjectivePointsPerCapture
}

// computeMatchIntensity calcule la caracteristique objective du match :
// events/min/joueur du lobby sur toute sa duree.
//
// Pas de fenetre glissante : c'est un agregat global du match. Permet de
// classer les matchs entre eux (calme / standard / chaotique).
//
// Retourne 0 si LobbyEvents vide ou NHumansLobby <= 0 (cas degenere).
func computeMatchIntensity(lobbyEvents []canonical.HighlightEvent, matchDurationMS int64, nHumans int) float64 {
	if len(lobbyEvents) == 0 || nHumans <= 0 || matchDurationMS <= 0 {
		return 0
	}
	durationMin := float64(matchDurationMS) / 60_000.0
	return float64(len(lobbyEvents)) / float64(nHumans) / durationMin
}

// computeResidualBrut calcule la moyenne du residu (joueur - attendu) sur
// toute la courbe. Cas degenere (curve vide) : retourne 0.
//
// Les points avec PostDeathFlag=true ne sont PAS exclus : la mort fait partie
// integrante du signal d'engagement (un joueur en bonne forme rebondit
// vite, un joueur en sous-forme reste planque -> sa courbe joueur reste basse
// et son residu est negatif sur la zone post-mort).
func computeResidualBrut(curve []domain.EngagementPoint) float64 {
	if len(curve) == 0 {
		return 0
	}
	var sum float64
	for _, p := range curve {
		sum += p.PaceJoueur - p.PaceAttendu
	}
	return sum / float64(len(curve))
}

// scoreFromHistory normalise le residu brut en percentile (0-100) sur la
// distribution historique du joueur.
//
// Retourne :
//   - score      : *float64 (nil si insufficient_history)
//   - confidence : "full" / "partial" / "insufficient_history"
//   - nHist      : nombre de matchs utilises (taille de l'historique)
//
// Convention : 50 = ecart moyen comme d'habitude (mediane historique).
// > 50 = ecart plus positif que d'habitude (en forme).
// < 50 = ecart plus negatif que d'habitude (sous-forme).
func scoreFromHistory(brut float64, history []domain.HistoricalEngagementBrut) (*float64, string, int) {
	n := len(history)
	if n < HistoryMinPartial {
		return nil, "insufficient_history", n
	}

	// Extraction des valeurs brutes en slice tries pour calcul de percentile.
	sorted := make([]float64, n)
	for i, h := range history {
		sorted[i] = h.Brut
	}
	sort.Float64s(sorted)

	percentile := percentileRank(brut, sorted)

	confidence := "full"
	if n < HistoryMinFull {
		confidence = "partial"
	}
	return &percentile, confidence, n
}

// percentileRank calcule le percentile d'une valeur dans un slice trie.
// Convention : le rang est [0, 100], la mediane d'une distribution donne 50.
//
// Si value est inferieur a min(sorted), retourne 0.
// Si value est superieur a max(sorted), retourne 100.
// Sinon interpolation lineaire.
//
// Cas degenere : sorted vide -> retourne 50 (neutre par convention).
func percentileRank(value float64, sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 50
	}
	if value <= sorted[0] {
		return 0
	}
	if value >= sorted[n-1] {
		return 100
	}
	// Recherche par dichotomie de la position de value dans sorted.
	idx := sort.SearchFloat64s(sorted, value)
	// idx = position d'insertion. Le percentile est idx/(n-1) * 100.
	return float64(idx) / float64(n-1) * 100
}

// validateBoundaries verifie la coherence des timestamps de match.
func validateBoundaries(input EngagementScoreInput) error {
	if input.MatchStartMS < 0 || input.MatchEndMS < 0 {
		return ErrInvalidBoundaries
	}
	if input.MatchEndMS <= input.MatchStartMS {
		return ErrInvalidBoundaries
	}
	return nil
}

// resolveWindow applique les defauts si les valeurs sont laissees a 0.
func resolveWindow(input EngagementScoreInput) (windowMS, samplingMS int64) {
	w := input.WindowSeconds
	if w <= 0 {
		w = DefaultWindowSeconds
	}
	s := input.SamplingSeconds
	if s <= 0 {
		s = DefaultSamplingSeconds
	}
	return int64(w) * 1000, int64(s) * 1000
}

// annotateDeaths positionne les flags PostDeathFlag et IsPassiveDeath sur les
// points de la courbe. Une mort est "passive" si elle est precedee par un
// creux d'inactivite > PassiveDeathThresholdMS sans event d'engagement du joueur.
func annotateDeaths(curve []domain.EngagementPoint, playerEvents []canonical.HighlightEvent, samplingMS int64) []domain.EngagementPoint {
	if len(curve) == 0 || len(playerEvents) == 0 {
		return curve
	}
	// Index par TimeMS croissant des events du joueur.
	deathTimes := make([]int64, 0, len(playerEvents))
	engagementTimes := make([]int64, 0, len(playerEvents))
	for _, e := range playerEvents {
		// Event d'engagement = kill / assist / medal (pas la death elle-meme).
		switch canonical.HighlightEventType(e.EventType) {
		case canonical.EventKill, canonical.EventAssist, canonical.EventMedal,
			canonical.EventFinisher, canonical.EventClutch, canonical.EventFirstKill:
			engagementTimes = append(engagementTimes, e.TimeMS)
		case canonical.EventDeath, canonical.EventFirstDeath:
			deathTimes = append(deathTimes, e.TimeMS)
		}
	}
	sort.Slice(deathTimes, func(i, j int) bool { return deathTimes[i] < deathTimes[j] })
	sort.Slice(engagementTimes, func(i, j int) bool { return engagementTimes[i] < engagementTimes[j] })

	if len(deathTimes) == 0 {
		return curve
	}

	// Pour chaque mort, determiner :
	//  1. Sa nature passive (creux > seuil avant la mort)
	//  2. Le prochain event d'engagement post-mort (pour la zone post-death)
	for _, dt := range deathTimes {
		isPassive := isPassiveDeath(dt, engagementTimes)
		nextEng := nextEngagementAfter(dt, engagementTimes)

		// Trouver les points de courbe correspondants.
		for i := range curve {
			pointTime := curve[i].TimeMS
			// Marquage du point exact de la mort (samplingMS-window match).
			if math.Abs(float64(pointTime-dt)) <= float64(samplingMS)/2 {
				curve[i].IsPassiveDeath = isPassive || curve[i].IsPassiveDeath
			}
			// Zone post-mort : entre dt et nextEng (ou fin de match).
			if pointTime >= dt && (nextEng == 0 || pointTime <= nextEng) {
				curve[i].PostDeathFlag = true
			}
		}
	}
	return curve
}

// isPassiveDeath retourne true si la mort dt est precedee par un creux
// > PassiveDeathThresholdMS sans event d'engagement.
func isPassiveDeath(deathMS int64, engagementTimes []int64) bool {
	// Trouver le dernier engagement avant la mort.
	var lastEng int64 = -1
	for _, e := range engagementTimes {
		if e < deathMS {
			lastEng = e
		} else {
			break
		}
	}
	if lastEng < 0 {
		// Pas d'engagement avant : passif si la mort arrive apres le seuil
		// depuis le debut du match.
		return deathMS >= PassiveDeathThresholdMS
	}
	return (deathMS - lastEng) > PassiveDeathThresholdMS
}

// nextEngagementAfter retourne le timestamp du prochain event d'engagement
// strictement apres deathMS. 0 si aucun.
func nextEngagementAfter(deathMS int64, engagementTimes []int64) int64 {
	for _, e := range engagementTimes {
		if e > deathMS {
			return e
		}
	}
	return 0
}
