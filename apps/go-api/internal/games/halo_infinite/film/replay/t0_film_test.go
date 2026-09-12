package replay

// t0_film_test.go — LE DETECTEUR DE COUP D'ENVOI, SUR FIXTURES SYNTHETIQUES.
//
// Chaque cas isole UNE propriete du contrat de `DetectT0Film` (cf. t0_film.go) : ce qu'il
// date, et surtout ce qu'il REFUSE. Les fixtures sont ecrites a la main plutot que lues d'un
// artefact — un corpus n'est pas versionne, et surtout il ne permet pas de poser un cumul
// EXACTEMENT sur le seuil.
//
// CONVENTION DES FIXTURES : pas de temps de 100 ms (celui de la production), donc une fenetre
// glissante de 10 frames, et des deplacements sur le seul axe X pour que la distance de chaque
// pas soit LISIBLE dans la fixture (0,25 m ecrit vaut 0,25 m parcouru — les valeurs choisies
// sont exactes en float32, sans quoi « juste sous le seuil » ne voudrait rien dire).

import "testing"

const t0tInterval = 100

// t0tLigne fabrique une piste dont le point d'index i est a l'abscisse xs[i], sur des frames
// CONTIGUES a partir de `start`.
func t0tLigne(xuid string, start int, xs ...float32) T0FilmTrack {
	pts := make([]T0FilmPoint, len(xs))
	for i, x := range xs {
		pts[i] = T0FilmPoint{T: start + i, X: x}
	}
	return T0FilmTrack{XUID: xuid, Points: pts}
}

// t0tCourse est une piste qui part franchement : quatre pas de 0,25 m, cumul 1 m.
func t0tCourse(xuid string, start int) T0FilmTrack {
	return t0tLigne(xuid, start, 0, 0.25, 0.5, 0.75, 1)
}

func TestDetectT0FilmRafaleNominale(t *testing.T) {
	// Deux joueurs partent a la meme frame, un troisieme 300 ms plus tard : la grille se leve.
	tracks := []T0FilmTrack{
		t0tCourse("A", 200),
		t0tCourse("B", 200),
		t0tCourse("C", 203),
	}
	t0, cov := DetectT0Film(tracks, t0tInterval, 9000, "test-rafale")
	if t0 == nil {
		t.Fatalf("coup d'envoi refuse alors que trois joueurs partent ensemble : %+v", cov)
	}
	// Le cumul depasse 0,5 m au troisieme point (0,5 m au deuxieme pas, pas encore STRICTEMENT
	// au-dessus ; 0,75 m au troisieme), soit la frame 203.
	if got, want := *t0, int64(9000+203*t0tInterval); got != want {
		t.Errorf("coup d'envoi = %d ms, attendu %d ms (origine + frame du premier mouvement)", got, want)
	}
	if !cov.Detected || cov.Reason != "" {
		t.Errorf("verdict = %+v, attendu detecte sans raison de refus", cov)
	}
	if cov.Tracks != 3 || cov.Moving != 3 {
		t.Errorf("pistes = %d exploitables / %d en mouvement, attendu 3 / 3", cov.Tracks, cov.Moving)
	}
	if cov.Burst != 3 {
		t.Errorf("rafale = %d, attendu 3 (les trois partent dans la seconde du premier)", cov.Burst)
	}
	if cov.MarginMs != int64(203*t0tInterval) {
		t.Errorf("marge = %d ms, attendu %d ms", cov.MarginMs, 203*t0tInterval)
	}
}

func TestDetectT0FilmRefuseRafaleUnique(t *testing.T) {
	// Un seul joueur part au debut ; le second ne bouge que 30 s plus tard. Un partant isole
	// est un artefact, pas une levee de grille.
	tracks := []T0FilmTrack{
		t0tCourse("A", 200),
		t0tCourse("B", 500),
	}
	t0, cov := DetectT0Film(tracks, t0tInterval, 9000, "test-solo")
	if t0 != nil {
		t.Fatalf("coup d'envoi date (%d ms) alors qu'un seul joueur part : %+v", *t0, cov)
	}
	if cov.Reason != t0FilmReasonSmallBurst {
		t.Errorf("raison = %q, attendu %q", cov.Reason, t0FilmReasonSmallBurst)
	}
	if cov.Burst != 1 || cov.Moving != 2 {
		t.Errorf("verdict = %+v, attendu rafale 1 et 2 pistes en mouvement", cov)
	}
}

func TestDetectT0FilmRefuseAfkComplet(t *testing.T) {
	// AUCUN point apres la frame 0 : le film ne replique la position que lorsqu'elle change,
	// donc un match ou personne ne bouge n'a qu'un point par piste.
	tracks := []T0FilmTrack{
		{XUID: "A", Points: []T0FilmPoint{{T: 0, X: 1}}},
		{XUID: "B", Points: []T0FilmPoint{{T: 0, X: 2}}},
	}
	t0, cov := DetectT0Film(tracks, t0tInterval, 9000, "test-afk")
	if t0 != nil {
		t.Fatalf("coup d'envoi date (%d ms) sur un film ou personne ne bouge", *t0)
	}
	if cov.Reason != t0FilmReasonNoMovement {
		t.Errorf("raison = %q, attendu %q", cov.Reason, t0FilmReasonNoMovement)
	}
	if cov.Tracks != 0 {
		t.Errorf("pistes exploitables = %d, attendu 0 (une piste a un point n'ouvre aucun pas)",
			cov.Tracks)
	}
}

func TestDetectT0FilmTeleportationNEstPasUnMouvement(t *testing.T) {
	// Un saut de 40 m en une frame est une apparition, pas une locomotion : il ne date rien,
	// et il ne doit pas non plus alimenter le cumul du pas suivant.
	tracks := []T0FilmTrack{
		{XUID: "A", Points: []T0FilmPoint{{T: 10, X: 0}, {T: 11, X: 40}, {T: 12, X: 40.25}}},
		{XUID: "B", Points: []T0FilmPoint{{T: 10, X: 0}, {T: 11, X: 40}, {T: 12, X: 40.25}}},
	}
	t0, cov := DetectT0Film(tracks, t0tInterval, 9000, "test-teleport")
	if t0 != nil {
		t.Fatalf("coup d'envoi date (%d ms) sur une simple teleportation : %+v", *t0, cov)
	}
	if cov.Reason != t0FilmReasonNoMovement {
		t.Errorf("raison = %q, attendu %q", cov.Reason, t0FilmReasonNoMovement)
	}
	if cov.Tracks != 2 {
		t.Errorf("pistes exploitables = %d, attendu 2 (elles ont trois points chacune)", cov.Tracks)
	}
}

func TestDetectT0FilmTrouDeReplicationPuisLocomotion(t *testing.T) {
	// LE PIEGE PRINCIPAL. Un joueur immobile pendant le decompte a un point a la frame 0, puis
	// plus rien jusqu'a son depart. Le pas frame 0 -> frame 220 est une RUPTURE (2,2 s de trou,
	// bien au-dela de la fenetre) : le compter daterait le coup d'envoi a la frame 0.
	saut := func(xuid string) T0FilmTrack {
		return T0FilmTrack{XUID: xuid, Points: []T0FilmPoint{
			{T: 0, X: 0},
			{T: 220, X: 0.4}, // rupture : 22 s de trou de replication
			{T: 221, X: 0.65},
			{T: 222, X: 0.9},
			{T: 223, X: 1.15},
		}}
	}
	t0, cov := DetectT0Film([]T0FilmTrack{saut("A"), saut("B")}, t0tInterval, 9000, "test-trou")
	if t0 == nil {
		t.Fatalf("coup d'envoi refuse alors que deux joueurs courent apres le trou : %+v", cov)
	}
	// Cumul apres le trou : 0,25 m (frame 221) puis 0,50 m (222, pas encore strictement
	// au-dessus) puis 0,75 m (223).
	if got, want := *t0, int64(9000+223*t0tInterval); got != want {
		t.Errorf("coup d'envoi = %d ms, attendu %d ms — le trou de replication a ete compte "+
			"comme un deplacement", got, want)
	}
}

func TestDetectT0FilmRefusePlusDeDeuxMinutes(t *testing.T) {
	// Rien avant la 130e seconde : ce n'est plus un coup d'envoi, c'est du jeu deja en cours
	// (film qui demarre tard, ou positions manquantes).
	tracks := []T0FilmTrack{t0tCourse("A", 1300), t0tCourse("B", 1300)}
	t0, cov := DetectT0Film(tracks, t0tInterval, 9000, "test-tard")
	if t0 != nil {
		t.Fatalf("coup d'envoi date (%d ms) a plus de deux minutes de la frame 0", *t0)
	}
	if cov.Reason != t0FilmReasonTooLate {
		t.Errorf("raison = %q, attendu %q", cov.Reason, t0FilmReasonTooLate)
	}
	if cov.MarginMs <= t0FilmMaxDelayMS {
		t.Errorf("marge = %d ms, attendu strictement au-dessus de %d ms",
			cov.MarginMs, t0FilmMaxDelayMS)
	}
}

func TestDetectT0FilmPisteAUnPointNEstPasExploitable(t *testing.T) {
	// La piste a un point ne compte NI au denominateur NI au numerateur : elle n'ouvre aucun
	// pas. Deux coureurs a cote d'elle suffisent a dater le coup d'envoi.
	tracks := []T0FilmTrack{
		{XUID: "Z", Points: []T0FilmPoint{{T: 5, X: 12}}},
		t0tCourse("A", 40),
		t0tCourse("B", 40),
	}
	t0, cov := DetectT0Film(tracks, t0tInterval, 0, "test-un-point")
	if t0 == nil {
		t.Fatalf("coup d'envoi refuse : %+v", cov)
	}
	if cov.Tracks != 2 || cov.Moving != 2 {
		t.Errorf("pistes = %d exploitables / %d en mouvement, attendu 2 / 2 (la piste a un "+
			"point est ecartee du denominateur)", cov.Tracks, cov.Moving)
	}
	// Origine nulle : le coup d'envoi vaut la marge seule — et c'est le PIEGE omitempty que le
	// pointeur ferme, un zero mesure restant une mesure.
	if got, want := *t0, int64(43*t0tInterval); got != want {
		t.Errorf("coup d'envoi = %d ms, attendu %d ms", got, want)
	}
}

func TestDetectT0FilmCumulJusteSousLeSeuil(t *testing.T) {
	// 0,25 + 0,25 = 0,50 m EXACTEMENT : le seuil est STRICT, ce n'est pas un mouvement.
	tracks := []T0FilmTrack{
		t0tLigne("A", 30, 0, 0.25, 0.5),
		t0tLigne("B", 30, 0, 0.25, 0.5),
	}
	t0, cov := DetectT0Film(tracks, t0tInterval, 9000, "test-seuil-sous")
	if t0 != nil {
		t.Fatalf("coup d'envoi date (%d ms) pour un cumul de 0,50 m — le seuil doit etre "+
			"strict", *t0)
	}
	if cov.Reason != t0FilmReasonNoMovement {
		t.Errorf("raison = %q, attendu %q", cov.Reason, t0FilmReasonNoMovement)
	}
}

func TestDetectT0FilmCumulJusteAuDessusDuSeuil(t *testing.T) {
	// Le meme cumul plus 0,0625 m : 0,5625 m franchit le seuil, a la frame du dernier pas.
	tracks := []T0FilmTrack{
		t0tLigne("A", 30, 0, 0.25, 0.5, 0.5625),
		t0tLigne("B", 30, 0, 0.25, 0.5, 0.5625),
	}
	t0, cov := DetectT0Film(tracks, t0tInterval, 9000, "test-seuil-sur")
	if t0 == nil {
		t.Fatalf("coup d'envoi refuse pour un cumul de 0,5625 m : %+v", cov)
	}
	if got, want := *t0, int64(9000+33*t0tInterval); got != want {
		t.Errorf("coup d'envoi = %d ms, attendu %d ms", got, want)
	}
}

func TestDetectT0FilmFenetreGlissanteOublieLesPasAnciens(t *testing.T) {
	// Un pas de 0,25 m toutes les 6 frames : deux pas consecutifs couvrent 12 frames, plus que
	// la fenetre de 10, donc le plus ancien est evince a chaque fois et le cumul retenu
	// plafonne a 0,25 m. C'est ce qui distingue une derive lente d'un depart.
	derive := func(xuid string) T0FilmTrack {
		pts := make([]T0FilmPoint, 0, 8)
		for i := 0; i < 8; i++ {
			pts = append(pts, T0FilmPoint{T: 10 + i*6, X: float32(i) * 0.25})
		}
		return T0FilmTrack{XUID: xuid, Points: pts}
	}
	t0, cov := DetectT0Film([]T0FilmTrack{derive("A"), derive("B")}, t0tInterval, 9000, "test-derive")
	if t0 != nil {
		t.Fatalf("coup d'envoi date (%d ms) sur une derive lente : %+v", *t0, cov)
	}
	if cov.Reason != t0FilmReasonNoMovement {
		t.Errorf("raison = %q, attendu %q", cov.Reason, t0FilmReasonNoMovement)
	}
}

func TestDetectT0FilmRefuseSansIntervalleDeFrame(t *testing.T) {
	// Sans echelle de temps, ni la fenetre ni la marge n'ont de sens : le detecteur s'abstient
	// au lieu de diviser par zero.
	tracks := []T0FilmTrack{t0tCourse("A", 10), t0tCourse("B", 10)}
	t0, cov := DetectT0Film(tracks, 0, 9000, "test-sans-intervalle")
	if t0 != nil {
		t.Fatalf("coup d'envoi date (%d ms) sans intervalle de frame", *t0)
	}
	if cov.Reason != t0FilmReasonNoFrameStep {
		t.Errorf("raison = %q, attendu %q", cov.Reason, t0FilmReasonNoFrameStep)
	}
}

func TestT0FilmBurstCompteLesJoueursPasLesVies(t *testing.T) {
	// Deux vies du meme joueur ne font qu'un partant ; une vie que le fil des morts n'a pas
	// nommee en fait un a elle seule (le film n'offre aucun moyen de la replier).
	deps := []t0FilmDeparture{
		{frame: 100, xuid: "A"},
		{frame: 102, xuid: "A"},
		{frame: 104, xuid: ""},
		{frame: 106, xuid: "B"},
		{frame: 300, xuid: "C"}, // 20 s plus tard : hors rafale
	}
	if got := t0FilmBurst(deps, t0tInterval, t0FilmBurstMS); got != 3 {
		t.Errorf("rafale = %d, attendu 3 (A, une vie anonyme, B — C est hors fenetre)", got)
	}
}

func TestT0FilmTracksOfConserveIdentiteEtPoints(t *testing.T) {
	src := []Track{{XUID: "A", Points: []Point{{T: 3, X: 1, Y: 2, Z: 3}, {T: 4, X: 5}}}}
	got := t0FilmTracksOf(src)
	if len(got) != 1 || got[0].XUID != "A" || len(got[0].Points) != 2 {
		t.Fatalf("conversion = %+v, attendu une piste nommee A a deux points", got)
	}
	if p := got[0].Points[0]; p.T != 3 || p.X != 1 || p.Y != 2 || p.Z != 3 {
		t.Errorf("premier point = %+v, attendu {T:3 X:1 Y:2 Z:3}", p)
	}
}
