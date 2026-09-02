package replay

// t0_mouvement_research_test.go — LE COUP D'ENVOI SE LIT-IL DANS LE PREMIER MOUVEMENT ?
//
// ## LA QUESTION
//
// Le T0 d'un match (fin du decompte pre-match, coup d'envoi du gameplay) est aujourd'hui
// ESTIME cote sync depuis les `first_joined_time` de l'API (`analysis/timeline/compute_t0.go`).
// Sur une part des matchs, l'API rend des `first_joined_time` colles au `start_time` — a
// quelques millisecondes — et le T0 calcule vaut alors ~0 alors que le vrai decompte dure une
// vingtaine de secondes. Le rejeu 2D demarre donc sur des joueurs statufies, et tout ce qui se
// date sur l'axe du match (premier frag...) est decale d'autant.
//
// HYPOTHESE MESUREE ICI : la grille se leve d'un coup, tout le monde part quasi simultanement,
// donc le PREMIER MOUVEMENT lu dans le film DATE le coup d'envoi. Si c'est vrai, l'ecart
// `t0_film - t0_api` doit etre STABLE (faible dispersion) sur les matchs ou le T0-API est sain,
// et le detecteur doit rendre un decompte plausible (15-45 s) la ou le T0-API a degenere.
//
// ## MESURE PURE
//
// Aucun code produit n'est touche, aucune base n'est ouverte, rien n'est ecrit hors `t.Logf`.
// Le document est relu par une structure MINIMALE (`t0mDoc`) et non par `ReplayDocument` : le
// corpus pese 240 Mo, et desserialiser les 40 champs de l'artefact pour n'en lire que les
// positions serait une bombe RAM gratuite. `encoding/json` ignore les champs inconnus.
//
// ## GARDES
//
//	REPLAY_CORPUS   dossier des artefacts de rejeu (data/cache/replays/halo_infinite)
//	T0_API_TSV      etalon T0-API : TSV `match_id \t t0_api_ms \t start_time_utc \t pair_name`
//
// Sans l'un des deux le test SAUTE : ni le corpus ni l'etalon ne sont versionnes.
//
// ## LE DETECTEUR, ET LES DEUX PIEGES QU'IL EVITE
//
// Par piste (une piste = UNE VIE, cf. `Track.XUID`), sur la serie de points :
//
//  1. le premier point d'une piste n'ouvre aucun pas (rien a soustraire) ;
//  2. un pas dont les deux points sont separes de plus de `t0mFenetreMS` est une RUPTURE, pas un
//     deplacement — le film ne replique la position que lorsqu'elle change, donc un joueur
//     immobile pendant le decompte n'a AUCUN point entre la frame 0 et son depart (temoin sur
//     1b2d9e08 : le slot 512 a un point a t=0 puis plus rien jusqu'a t=227). Compter ce pas
//     comme un deplacement daterait le coup d'envoi a la frame 0 de tous les films ;
//  3. un pas de plus de `t0mSautM` est une TELEPORTATION (apparition, arrivee tardive), pas une
//     locomotion : rupture aussi.
//
// MOUVEMENT = le cumul des pas contigus depasse `t0mCumulM` dans une fenetre glissante de
// `t0mFenetreMS`. Un jitter d'une seule image ne suffit donc pas.
//
// `t0_film_ms = originMs + frameDuPremierMouvement * frameIntervalMs` — `originMs` etant
// l'instant de la frame 0 sur l'axe du match (schema >= 4, cf. document.go / origin.go), la
// comparaison au T0-API est une soustraction, pas un recalage.
//
// ## TROIS VARIANTES PUBLIEES, PARCE QU'UN SEUL JOUEUR N'EST PAS UNE MESURE
//
// Le premier joueur a bouger est, par construction, le plus extreme des tirages : un seul
// artefact (un point aberrant, une piste de spectateur) le decale. Le test publie donc aussi le
// TROISIEME joueur a bouger et la MEDIANE des premiers mouvements par joueur, sur les memes
// matchs et avec les memes statistiques. Le choix se fait sur la dispersion mesuree, pas sur le
// nom de la variante.
//
// ## PLANCHER DE BRUIT
//
// Sur les matchs au T0-API sain, le test mesure ce qu'un joueur deplace AVANT le T0-API — la
// fenetre de decompte, ou il est cense etre immobile. C'est ce chiffre qui valide (ou invalide)
// le seuil de 0,5 m : si le decompte porte deja des cumuls d'un metre, le seuil est trop bas.
//
// ## CE QUE LA MESURE A TROUVE (2026-09-02, 106 artefacts, 101 retenus)
//
// L'HYPOTHESE TIENT, ET ELLE TIENT MIEUX QUE PREVU. Le detecteur est stable au dixieme de
// seconde ; c'est l'ETALON qui est bruite, pas lui. Trois temoins internes au film, aucun
// etalon en jeu :
//
//	RAFALE      66,7 % de l'effectif (mediane 6 joueurs sur 8-9) part dans la SECONDE du
//	            premier — la grille se leve bien d'un coup.
//	ACCORD      l'ecart entre le 1er et le 3e partant vaut 100 ms de mediane, 500 ms au p95.
//	            Le choix de la variante ne change donc rien : 1er et 3e donnent le meme
//	            instant a 250 ms pres sur les 49 matchs sains.
//	MARGE       le premier mouvement tombe a 22 700 ms de la frame 0 du film, ECART-TYPE
//	            299 ms, CV 0,013, etendue complete 21 500 - 23 000 ms sur 83 matchs.
//
// LA MARGE CONSTANTE EST LE RESULTAT, et il faut le lire pour ce qu'il est : le decompte
// pre-match ne commence pas au `start_time` de l'API, il commence quand les positions se
// repliquent — c'est-a-dire une fois tout le monde charge. `originMs` porte ce chargement (3,4 s
// a 38,5 s selon le match), et le decompte qui suit est une CONSTANTE DU JEU. T0 vaut donc
// `originMs + 22 700 ms`, et le mouvement ne fait que retrouver cette constante.
//
// DEUX REGIMES DE DEMARRAGE DE FILM, et le second n'est pas une mesure perdue. 18 films sur 101
// ont une marge de 200-400 ms au lieu de 22,7 s : leur frame 0 tombe SUR le coup d'envoi. Le
// controle ecarte la lecture « film tronque » — les deux populations rendent le meme t0_film
// (mediane 33 725 contre 33 802 ms, p5 26 185 contre 26 789), ce qu'un demarrage a un instant
// quelconque du match ne ferait pas.
//
// L'ETALON, LUI, EST CASSE SUR 41 MATCHS SUR 101 : `t0_api_ms` y vaut -3 600 000 ou
// -7 200 000 ms, soit exactement une ou deux heures — un decalage de fuseau, pas un decompte.
// Sur les 49 matchs sains restants, l'ecart `t0_film - t0_api` a une mediane de +4 523 ms mais
// un ecart-type de 5 949 ms, et |ecart| <= 1 s dans 8,2 % des cas seulement. Retirer la censure
// n'y change rien (n=39, ecart-type 6 351 ms) : la dispersion vient de l'etalon. Le controle
// direct est la comparaison des DEUX horloges sur ces memes 49 matchs — t0_film a un ecart-type
// de 9 752 ms, t0_api de 12 764 ms, et t0_api descend a 17 804 ms la ou t0_film ne descend
// jamais sous 25 907 ms.
//
// LA VARIANTE « MEDIANE DES JOUEURS » EST REFUTEE par sa propre mesure (+241 s d'ecart median) :
// une piste est une VIE, pas un joueur. Elle reste publiee comme negatif.
//
// SUR LES 11 MATCHS DEGENERES (t0_api < 5 s, dont `1b2d9e08`, `72b0a25e` et `28c9b538`), le
// detecteur rend 10 decomptes sur 11 dans 15-45 s. `1b2d9e08` sort a 31 862 ms, ce qui recoupe
// l'observation utilisateur (« le vrai gameplay demarre vers ~20 s » — a l'ecran, le decompte
// visible commence apres le chargement, ici 9 062 ms).

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// --- Seuils du detecteur, ecrits avant la mesure (cf. en-tete pour le raisonnement) ---
const (
	// t0mSautM : au-dela, le pas est une teleportation, pas une locomotion.
	t0mSautM = 5.0
	// t0mCumulM : le deplacement cumule qui fait un mouvement.
	t0mCumulM = 0.5
	// t0mFenetreMS : largeur de la fenetre glissante de cumul, ET duree maximale d'un pas
	// exploitable (au-dela, le film n'a rien replique : c'est une rupture).
	t0mFenetreMS = 1000
)

// --- Bornes du corpus SAIN ---
//
// LA BORNE HAUTE EST UN AJUSTEMENT ASSUME, et il faut le dire : la consigne definissait le
// corpus sain par `t0_api_ms >= 5000` seul. L'etalon porte des valeurs de 3 600 000 ms (une
// heure pile — un decalage de fuseau) et des valeurs negatives de l'ordre de -3 500 000 ms. Les
// premieres passent `>= 5000` sans etre saines. La borne haute reprend `t0MaxPlausibleMs` de la
// production (`analysis/timeline/compute_t0.go`), qui rejette exactement ces valeurs sous la
// qualite `suspicious_high` ; elles sont comptees a part, jamais melangees au corpus sain.
const (
	t0mSainMinMS = 5000
	t0mSainMaxMS = 120000
)

// t0mMargeCensureMS : en deca, le premier mouvement colle a la frame 0 du film, et le detecteur
// n'a RIEN mesure — il rend `originMs`, une borne superieure de T0. Deux secondes, parce que la
// rafale de depart (mesuree ci-dessous) tient dans quelques centaines de millisecondes : un
// premier mouvement a moins de 2 s de la frame 0 est un film qui commence apres le coup d'envoi.
const t0mMargeCensureMS = 2000

// t0mRafaleMS : la fenetre dans laquelle on compte les joueurs qui partent AVEC le premier.
const t0mRafaleMS = 1000

// --- Lecture minimale de l'artefact ---

type t0mPoint struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type t0mTrack struct {
	Slot   uint32     `json:"slot"`
	XUID   string     `json:"xuid"`
	Points []t0mPoint `json:"points"`
}

type t0mDoc struct {
	SchemaVersion   int        `json:"schemaVersion"`
	MatchID         string     `json:"matchId"`
	FrameCount      int        `json:"frameCount"`
	FrameIntervalMS int        `json:"frameIntervalMs"`
	OriginMs        *int64     `json:"originMs"`
	Tracks          []t0mTrack `json:"tracks"`
}

// t0mPas est un deplacement entre deux points CONSECUTIFS d'une meme piste.
type t0mPas struct {
	tDebut, tFin int
	d            float64
	// rupture : le pas ne peut pas etre lu comme une locomotion (trou de replication ou
	// teleportation). Il remet l'accumulateur a zero au lieu de l'alimenter.
	rupture bool
}

// t0mPasDeLaPiste ramene une piste a la suite de ses pas. Le premier point n'ouvre aucun pas.
func t0mPasDeLaPiste(pts []t0mPoint, pasMS int) []t0mPas {
	if len(pts) < 2 {
		return nil
	}
	tri := append([]t0mPoint(nil), pts...)
	sort.Slice(tri, func(i, j int) bool { return tri[i].T < tri[j].T })
	out := make([]t0mPas, 0, len(tri)-1)
	for i := 1; i < len(tri); i++ {
		a, b := tri[i-1], tri[i]
		d := dist3([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z})
		dtMS := (b.T - a.T) * pasMS
		out = append(out, t0mPas{
			tDebut:  a.T,
			tFin:    b.T,
			d:       d,
			rupture: dtMS > t0mFenetreMS || d > t0mSautM,
		})
	}
	return out
}

// t0mAnalysePiste parcourt les pas d'une piste et rend :
//   - la frame du PREMIER MOUVEMENT (-1 quand la piste n'en porte aucun) ;
//   - le pas unitaire MAXIMAL et le cumul de fenetre MAXIMAL observes strictement AVANT
//     `frameCoupe` — le plancher de bruit. `frameCoupe < 0` desactive cette mesure.
func t0mAnalysePiste(pas []t0mPas, pasMS, frameCoupe int) (mouv int, maxPas, maxCumul float64) {
	mouv = -1
	// La fenetre glissante, en FRAMES : les pas contigus retenus et leur somme.
	largeur := t0mFenetreMS / pasMS
	var fTDebut []int
	var fD []float64
	somme := 0.0
	for _, p := range pas {
		if p.rupture {
			fTDebut, fD, somme = fTDebut[:0], fD[:0], 0
			continue
		}
		fTDebut = append(fTDebut, p.tDebut)
		fD = append(fD, p.d)
		somme += p.d
		// La fenetre se mesure du DEBUT du pas le plus ancien a la FIN du pas courant.
		for len(fD) > 1 && (p.tFin-fTDebut[0]) > largeur {
			somme -= fD[0]
			fTDebut, fD = fTDebut[1:], fD[1:]
		}
		if frameCoupe >= 0 && p.tFin < frameCoupe {
			if p.d > maxPas {
				maxPas = p.d
			}
			if somme > maxCumul {
				maxCumul = somme
			}
		}
		if mouv < 0 && somme > t0mCumulM {
			mouv = p.tFin
		}
	}
	return mouv, maxPas, maxCumul
}

// t0mDepart est le premier mouvement d'une piste : sa frame et l'identite de son porteur.
type t0mDepart struct {
	frame int
	xuid  string
}

// t0mMesureMatch est le resultat du detecteur sur un artefact.
type t0mMesureMatch struct {
	prefixe  string
	matchID  string
	originMs int64
	pasMS    int
	// premiers : le premier mouvement de CHAQUE piste qui en porte un, trie par frame.
	premiers []t0mDepart
	// pistes : nombre de pistes lues ; sansMouvement : celles qui n'en portent aucun.
	pistes, sansMouvement int
	// slots : nombre de XUID distincts, c'est-a-dire l'EFFECTIF du match. Le premier jet
	// comptait les SLOTS et c'etait faux : le slot est reattribue a chaque vie (mediane de
	// 6,4 % « d'effectif » dans la rafale, sur un denominateur de ~94 « joueurs »). Le xuid est
	// la seule identite stable d'une piste (cf. `Track.XUID`).
	slots int
	// plancher : le maximum, tous joueurs confondus, du pas unitaire et du cumul de fenetre
	// observes avant la frame de coupe (le T0-API).
	planchePas, plancheCumul float64
	// pasAvantCoupe : nombre de pas non-rupture situes avant la coupe (0 = plancher non mesure).
	pasAvantCoupe int
}

// t0mMS convertit une frame en instant sur l'axe du match.
func (m t0mMesureMatch) t0mMS(frame int) int64 {
	return m.originMs + int64(frame)*int64(m.pasMS)
}

// premier / troisieme / mediane : les trois variantes de lecture du coup d'envoi.
func (m t0mMesureMatch) premier() (int64, bool) {
	if len(m.premiers) == 0 {
		return 0, false
	}
	return m.t0mMS(m.premiers[0].frame), true
}

func (m t0mMesureMatch) troisieme() (int64, bool) {
	if len(m.premiers) < 3 {
		return 0, false
	}
	return m.t0mMS(m.premiers[2].frame), true
}

// medianeJoueurs est REFUTEE par sa propre mesure, et elle reste publiee pour que le negatif
// soit lisible : une PISTE est UNE VIE (cf. `Track.XUID`), pas un joueur. La mediane des
// premiers mouvements par piste tombe donc au MILIEU DU MATCH (+241 s de delta median a la
// premiere passe), ou l'on renait et l'on repart en continu. Seules les toutes premieres pistes
// portent le coup d'envoi ; c'est pourquoi les variantes utiles sont le 1er et le 3e.
func (m t0mMesureMatch) medianeJoueurs() (int64, bool) {
	if len(m.premiers) == 0 {
		return 0, false
	}
	return m.t0mMS(m.premiers[len(m.premiers)/2].frame), true
}

// marge est l'ecart entre le premier mouvement et la FRAME 0 du film. C'est le temoin de
// CENSURE : quand elle est quasi nulle, le film commence APRES le coup d'envoi et le detecteur
// ne mesure plus rien — il rend `originMs`, c'est-a-dire une BORNE SUPERIEURE de T0, pas T0.
func (m t0mMesureMatch) marge() int64 {
	if len(m.premiers) == 0 {
		return -1
	}
	return int64(m.premiers[0].frame) * int64(m.pasMS)
}

// rafale compte les pistes dont le premier mouvement tombe a moins de `fenetreMS` du tout
// premier. C'EST LE TEMOIN DIRECT DE L'HYPOTHESE : si la grille se leve d'un coup, l'essentiel
// de l'effectif part ensemble. Un « premier mouvement » isole serait, lui, un artefact.
func (m t0mMesureMatch) rafale(fenetreMS int64) int {
	if len(m.premiers) == 0 {
		return 0
	}
	base := int64(m.premiers[0].frame) * int64(m.pasMS)
	// PAR XUID, pas par piste : deux vies du meme joueur ne font pas deux partants.
	vus := map[string]bool{}
	n := 0
	for _, d := range m.premiers {
		if int64(d.frame)*int64(m.pasMS)-base > fenetreMS {
			break
		}
		if d.xuid == "" {
			n++
			continue
		}
		if !vus[d.xuid] {
			vus[d.xuid] = true
			n++
		}
	}
	return n
}

// t0mAnalyseDoc applique le detecteur a tout un document. `frameCoupe` borne la mesure du
// plancher de bruit (-1 = pas de plancher).
func t0mAnalyseDoc(doc *t0mDoc, frameCoupe int) t0mMesureMatch {
	m := t0mMesureMatch{
		prefixe:  t0mPrefixe(doc.MatchID),
		matchID:  doc.MatchID,
		originMs: *doc.OriginMs,
		pasMS:    doc.FrameIntervalMS,
	}
	vus := map[string]bool{}
	for i := range doc.Tracks {
		if x := doc.Tracks[i].XUID; x != "" {
			vus[x] = true
		}
		pts := doc.Tracks[i].Points
		if len(pts) < 2 {
			continue
		}
		m.pistes++
		pas := t0mPasDeLaPiste(pts, doc.FrameIntervalMS)
		mouv, maxPas, maxCumul := t0mAnalysePiste(pas, doc.FrameIntervalMS, frameCoupe)
		if mouv < 0 {
			m.sansMouvement++
		} else {
			m.premiers = append(m.premiers, t0mDepart{frame: mouv, xuid: doc.Tracks[i].XUID})
		}
		if frameCoupe >= 0 {
			for _, p := range pas {
				if !p.rupture && p.tFin < frameCoupe {
					m.pasAvantCoupe++
				}
			}
			if maxPas > m.planchePas {
				m.planchePas = maxPas
			}
			if maxCumul > m.plancheCumul {
				m.plancheCumul = maxCumul
			}
		}
	}
	sort.Slice(m.premiers, func(a, b int) bool { return m.premiers[a].frame < m.premiers[b].frame })
	m.slots = len(vus)
	return m
}

// --- Etalon T0-API ---

type t0mLigneEtalon struct {
	matchID  string
	t0APIms  int64
	present  bool // false = colonne vide (NULL en base)
	pairName string
}

// t0mPrefixe rend les 8 premiers caracteres hexadecimaux d'un match_id — la cle de jointure,
// et le nom de fichier du corpus.
func t0mPrefixe(matchID string) string {
	s := strings.TrimSpace(matchID)
	if len(s) < 8 {
		return s
	}
	return strings.ToLower(s[:8])
}

// t0mChargerEtalon lit le TSV et l'indexe par prefixe 8 hex. Les lignes de queue d'un client
// SQL (« (N rows) ») et l'en-tete sont ecartees et COMPTEES.
func t0mChargerEtalon(t *testing.T) (map[string]t0mLigneEtalon, int) {
	t.Helper()
	path := strings.TrimSpace(os.Getenv("T0_API_TSV"))
	if path == "" {
		t.Skip("T0_API_TSV non defini : etalon T0-API absent (non versionne)")
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		t.Fatalf("lecture de l'etalon %s : %v", path, err)
	}
	out := map[string]t0mLigneEtalon{}
	ecartees := 0
	for i, ligne := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(ligne) == "" {
			continue
		}
		champs := strings.Split(ligne, "\t")
		if i == 0 && champs[0] == "match_id" {
			continue
		}
		if len(champs) < 2 || len(strings.TrimSpace(champs[0])) < 8 {
			ecartees++
			continue
		}
		l := t0mLigneEtalon{matchID: strings.TrimSpace(champs[0])}
		if len(champs) >= 4 {
			l.pairName = strings.TrimSpace(champs[3])
		}
		brut := strings.TrimSpace(champs[1])
		if brut != "" {
			v, err := strconv.ParseInt(brut, 10, 64)
			if err != nil {
				ecartees++
				continue
			}
			l.t0APIms, l.present = v, true
		}
		out[t0mPrefixe(l.matchID)] = l
	}
	return out, ecartees
}

// --- Statistiques ---

type t0mStats struct {
	n                   int
	mediane, moyenne    float64
	ecartType, cv       float64
	p5, p95, mini, maxi float64
}

func t0mCalcStats(v []float64) t0mStats {
	s := t0mStats{n: len(v)}
	if s.n == 0 {
		return s
	}
	tri := append([]float64(nil), v...)
	sort.Float64s(tri)
	somme := 0.0
	for _, x := range tri {
		somme += x
	}
	s.moyenne = somme / float64(s.n)
	s.mediane = t0mQuantile(tri, 0.5)
	s.p5, s.p95 = t0mQuantile(tri, 0.05), t0mQuantile(tri, 0.95)
	s.mini, s.maxi = tri[0], tri[s.n-1]
	varSum := 0.0
	for _, x := range tri {
		varSum += (x - s.moyenne) * (x - s.moyenne)
	}
	s.ecartType = math.Sqrt(varSum / float64(s.n))
	if s.moyenne != 0 {
		s.cv = s.ecartType / math.Abs(s.moyenne)
	}
	return s
}

func t0mQuantile(tri []float64, q float64) float64 {
	if len(tri) == 0 {
		return 0
	}
	idx := int(math.Round(q * float64(len(tri)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(tri) {
		idx = len(tri) - 1
	}
	return tri[idx]
}

func (s t0mStats) ligne() string {
	return fmt.Sprintf("n=%d · mediane %+.0f ms · moyenne %+.0f ms · ecart-type %.0f ms · "+
		"CV %.3f · p5 %+.0f · p95 %+.0f · min %+.0f · max %+.0f",
		s.n, s.mediane, s.moyenne, s.ecartType, s.cv, s.p5, s.p95, s.mini, s.maxi)
}

// --- Le test ---

// t0mEcart nomme un ecart pour la liste des pires cas.
type t0mEcart struct {
	prefixe, pair string
	delta         float64
	t0API, t0Film int64
}

// t0mComptes ventile tout ce qui est ECARTE, et pourquoi.
type t0mComptes struct {
	fichiers, decodes           int
	sansOrigine, sansIntervalle int
	sansPiste, sansMouvement    int
	horsEtalon, etalonNul       int
	aberrants, degeneres, sains int
	schemaTropVieux, illisibles int
	lignesEtalonEcartees        int
	degeneresSansDetection      int
}

func TestT0MouvementContreEtalonAPI(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv("REPLAY_CORPUS"))
	if dir == "" {
		t.Skip("REPLAY_CORPUS non defini : corpus d'artefacts absent (non versionne)")
	}
	etalon, ecartees := t0mChargerEtalon(t)
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture du corpus %s : %v", dir, err)
	}

	var c t0mComptes
	c.lignesEtalonEcartees = ecartees
	var deltasPremier, deltasTroisieme, deltasMediane []float64
	// deltasNonCensures : les seuls matchs ou le detecteur MESURE quelque chose — ceux dont le
	// premier mouvement est franchement posterieur a la frame 0 du film.
	var deltasNonCensures []float64
	var pires []t0mEcart
	var planchesPas, planchesCumul []float64
	var pasAvantCoupeTotal int
	// Temoins internes au film, mesures sur TOUS les matchs retenus (sains ou non) : ils ne
	// dependent pas de l'etalon.
	var marges, rafales, partsRafale, ecartsInternes []float64
	// Les deux regimes de demarrage de film, pour savoir si la « censure » est une mesure
	// perdue ou simplement un film qui commence AU coup d'envoi.
	var t0FilmsLibres, t0FilmsCensures []float64
	// Les deux populations d'instants, pour comparer leur DISPERSION sur le corpus sain.
	var t0FilmsSains, t0APIsSains []float64
	censures := 0
	type degenere struct {
		prefixe, pair             string
		t0API                     int64
		premier, troisieme, med   int64
		ok                        bool
		pistes, joueursQuiBougent int
		originMs, marge           int64
		rafale, effectif          int
	}
	var degeneres []degenere

	noms := make([]string, 0, len(entrees))
	for _, e := range entrees {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		noms = append(noms, e.Name())
	}
	sort.Strings(noms)
	c.fichiers = len(noms)

	for _, nom := range noms {
		raw, err := os.ReadFile(filepath.Join(dir, nom)) //nolint:gosec // corpus de l'operateur
		if err != nil {
			c.illisibles++
			t.Logf("  ECARTE %s : lecture impossible (%v)", nom, err)
			continue
		}
		var doc t0mDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			c.illisibles++
			t.Logf("  ECARTE %s : decodage impossible (%v)", nom, err)
			continue
		}
		raw = nil
		c.decodes++
		if doc.SchemaVersion < 4 {
			c.schemaTropVieux++
			t.Logf("  ECARTE %s : schema %d < 4, aucune origine possible", nom, doc.SchemaVersion)
			continue
		}
		if doc.OriginMs == nil {
			c.sansOrigine++
			t.Logf("  ECARTE %s : aucun originMs publie (schema %d)", nom, doc.SchemaVersion)
			continue
		}
		if doc.FrameIntervalMS <= 0 {
			c.sansIntervalle++
			t.Logf("  ECARTE %s : frameIntervalMs absent ou nul", nom)
			continue
		}
		pref := t0mPrefixe(doc.MatchID)
		ref, connu := etalon[pref]
		if !connu {
			c.horsEtalon++
			t.Logf("  ECARTE %s : match absent de l'etalon TSV", pref)
			continue
		}
		if !ref.present {
			c.etalonNul++
			t.Logf("  ECARTE %s : t0_api_ms NULL a l'etalon (%s)", pref, ref.pairName)
			continue
		}

		sain := ref.t0APIms >= t0mSainMinMS && ref.t0APIms <= t0mSainMaxMS
		aberrant := ref.t0APIms < 0 || ref.t0APIms > t0mSainMaxMS
		frameCoupe := -1
		if sain {
			// La coupe du plancher de bruit : la frame du film qui precede le T0-API.
			frameCoupe = int((ref.t0APIms - *doc.OriginMs) / int64(doc.FrameIntervalMS))
			if frameCoupe < 0 {
				frameCoupe = 0
			}
		}
		m := t0mAnalyseDoc(&doc, frameCoupe)
		doc.Tracks = nil

		if m.pistes == 0 {
			c.sansPiste++
			t.Logf("  ECARTE %s : aucune piste a deux points", pref)
			continue
		}
		if len(m.premiers) == 0 {
			c.sansMouvement++
			t.Logf("  ECARTE %s : aucun mouvement detecte sur %d piste(s)", pref, m.pistes)
			continue
		}

		// Temoins internes au film — ils valent pour les trois populations.
		if p, ok := m.premier(); ok {
			marges = append(marges, float64(m.marge()))
			if m.marge() < t0mMargeCensureMS {
				censures++
				t0FilmsCensures = append(t0FilmsCensures, float64(p))
			} else {
				t0FilmsLibres = append(t0FilmsLibres, float64(p))
			}
			r := m.rafale(t0mRafaleMS)
			rafales = append(rafales, float64(r))
			if m.slots > 0 {
				partsRafale = append(partsRafale, 100*float64(r)/float64(m.slots))
			}
			if tr, ok3 := m.troisieme(); ok3 {
				ecartsInternes = append(ecartsInternes, float64(tr-p))
			}
		}

		switch {
		case aberrant:
			c.aberrants++
			p, _ := m.premier()
			t.Logf("  ABERRANT %s (%s) : t0_api = %d ms hors [0, %d] — t0_film = %d ms",
				pref, ref.pairName, ref.t0APIms, t0mSainMaxMS, p)
		case sain:
			c.sains++
			p, _ := m.premier()
			tr, okT := m.troisieme()
			md, okM := m.medianeJoueurs()
			d := float64(p - ref.t0APIms)
			deltasPremier = append(deltasPremier, d)
			if okT {
				deltasTroisieme = append(deltasTroisieme, float64(tr-ref.t0APIms))
			}
			if okM {
				deltasMediane = append(deltasMediane, float64(md-ref.t0APIms))
			}
			if m.marge() >= t0mMargeCensureMS {
				deltasNonCensures = append(deltasNonCensures, d)
			}
			t0FilmsSains = append(t0FilmsSains, float64(p))
			t0APIsSains = append(t0APIsSains, float64(ref.t0APIms))
			pires = append(pires, t0mEcart{pref, ref.pairName, d, ref.t0APIms, p})
			if m.pasAvantCoupe > 0 {
				planchesPas = append(planchesPas, m.planchePas)
				planchesCumul = append(planchesCumul, m.plancheCumul)
				pasAvantCoupeTotal += m.pasAvantCoupe
			}
		default:
			c.degeneres++
			p, ok := m.premier()
			tr, _ := m.troisieme()
			md, _ := m.medianeJoueurs()
			degeneres = append(degeneres, degenere{
				prefixe: pref, pair: ref.pairName, t0API: ref.t0APIms,
				premier: p, troisieme: tr, med: md, ok: ok,
				pistes: m.pistes, joueursQuiBougent: len(m.premiers), originMs: m.originMs,
				marge: m.marge(), rafale: m.rafale(t0mRafaleMS), effectif: m.slots,
			})
			if !ok {
				c.degeneresSansDetection++
			}
		}
	}

	t0mPublierPopulation(t, c)
	t0mPublierTemoinsFilm(t, marges, rafales, partsRafale, ecartsInternes, censures)
	t0mPublierPlancher(t, planchesPas, planchesCumul, pasAvantCoupeTotal)
	t0mPublierSains(t, deltasPremier, deltasTroisieme, deltasMediane, pires)
	t0mPublierCensure(t, deltasNonCensures, t0FilmsSains, t0APIsSains)
	// LA CENSURE EST-ELLE UNE MESURE PERDUE, OU UN AUTRE REGIME DE DEMARRAGE ? Si le film
	// « censure » commencait a un instant quelconque du match, son t0_film serait quelconque
	// lui aussi. S'il tombe dans la MEME plage que celui des films qui filment le decompte,
	// c'est que sa frame 0 est POSEE sur le coup d'envoi — et la mesure vaut.
	sLibre, sCens := t0mCalcStats(t0FilmsLibres), t0mCalcStats(t0FilmsCensures)
	t.Logf("")
	t.Logf("== LES DEUX REGIMES DE DEMARRAGE DU FILM (tous matchs retenus) ==")
	t.Logf("films qui FILMENT le decompte (marge >= %d ms) — t0_film : n=%d · mediane %.0f ms · "+
		"p5 %.0f · p95 %.0f", t0mMargeCensureMS, sLibre.n, sLibre.mediane, sLibre.p5, sLibre.p95)
	t.Logf("films dont la frame 0 colle au depart (marge < %d ms) — t0_film : n=%d · mediane "+
		"%.0f ms · p5 %.0f · p95 %.0f", t0mMargeCensureMS, sCens.n, sCens.mediane, sCens.p5,
		sCens.p95)
	t.Logf("LECTURE : deux plages qui se recouvrent => la frame 0 du second regime EST le coup " +
		"d'envoi, la mesure n'est pas perdue. Deux plages disjointes => t0_film n'y est qu'une borne.")

	t.Logf("")
	t.Logf("== CORPUS DEGENERE (0 <= t0_api < %d ms) — %d match(s) ==", t0mSainMinMS, len(degeneres))
	t.Logf("%-9s %-38s %7s %8s %8s %8s %7s %8s", "prefixe", "variante", "t0_api", "t0_film",
		"3e", "origine", "marge", "rafale")
	sort.Slice(degeneres, func(i, j int) bool { return degeneres[i].premier < degeneres[j].premier })
	plausibles, censuresDeg := 0, 0
	for _, d := range degeneres {
		if !d.ok {
			t.Logf("%-9s %-38s %7d %s", d.prefixe, t0mCoupe(d.pair, 38), d.t0API,
				"AUCUN MOUVEMENT DETECTE")
			continue
		}
		if d.premier >= 15000 && d.premier <= 45000 {
			plausibles++
		}
		marque := ""
		if d.marge < t0mMargeCensureMS {
			censuresDeg++
			marque = "  CENSURE (film demarre apres le coup d'envoi : borne SUPERIEURE)"
		}
		t.Logf("%-9s %-38s %7d %8d %8d %8d %7d %5d/%-2d%s", d.prefixe, t0mCoupe(d.pair, 38),
			d.t0API, d.premier, d.troisieme, d.originMs, d.marge, d.rafale, d.effectif, marque)
	}
	t.Logf("decomptes PLAUSIBLES (15-45 s) chez les degeneres : %d / %d = %.1f %% · dont "+
		"CENSURES (t0_film = borne superieure, pas une mesure) : %d",
		plausibles, len(degeneres), pct100(plausibles, len(degeneres)), censuresDeg)
}

// t0mPublierPopulation ventile tout ce qui entre et tout ce qui sort.
func t0mPublierPopulation(t *testing.T, c t0mComptes) {
	t.Helper()
	t.Logf("== POPULATION ==")
	t.Logf("artefacts .json au corpus : %d · decodes : %d · illisibles : %d",
		c.fichiers, c.decodes, c.illisibles)
	t.Logf("ECARTES — schema < 4 : %d · sans originMs : %d · sans frameIntervalMs : %d · "+
		"absents de l'etalon : %d · t0_api NULL : %d · sans piste : %d · sans mouvement : %d",
		c.schemaTropVieux, c.sansOrigine, c.sansIntervalle, c.horsEtalon, c.etalonNul,
		c.sansPiste, c.sansMouvement)
	t.Logf("RETENUS — sains (t0_api dans [%d, %d]) : %d · degeneres (< %d) : %d · "+
		"aberrants (< 0 ou > %d) : %d",
		t0mSainMinMS, t0mSainMaxMS, c.sains, t0mSainMinMS, c.degeneres, t0mSainMaxMS, c.aberrants)
	t.Logf("lignes de l'etalon ecartees a la lecture (queue de client SQL, champs manquants) : %d",
		c.lignesEtalonEcartees)
}

// t0mPublierPlancher publie le plancher de bruit : ce qu'un joueur deplace PENDANT le decompte.
func t0mPublierPlancher(t *testing.T, pas, cumuls []float64, nPas int) {
	t.Helper()
	t.Logf("")
	t.Logf("== PLANCHER DE BRUIT (fenetre de decompte des matchs SAINS) ==")
	if len(pas) == 0 {
		t.Logf("aucun pas exploitable avant le T0-API : le plancher n'est pas mesure "+
			"(l'origine du film tombe apres le T0-API sur les %d match(s) sains)", len(pas))
		return
	}
	sp, sc := t0mCalcStats(pas), t0mCalcStats(cumuls)
	t.Logf("matchs avec au moins un pas avant le T0-API : %d · pas non-rupture totaux : %d",
		len(pas), nPas)
	t.Logf("PAS UNITAIRE max par match (m)  : mediane %.3f · moyenne %.3f · p95 %.3f · max %.3f",
		sp.mediane, sp.moyenne, sp.p95, sp.maxi)
	t.Logf("CUMUL DE FENETRE max par match (m) : mediane %.3f · moyenne %.3f · p95 %.3f · max %.3f",
		sc.mediane, sc.moyenne, sc.p95, sc.maxi)
	depassent := 0
	for _, v := range cumuls {
		if v > t0mCumulM {
			depassent++
		}
	}
	t.Logf("matchs dont le decompte depasse deja le seuil de %.2f m : %d / %d = %.1f %%",
		t0mCumulM, depassent, len(cumuls), pct100(depassent, len(cumuls)))
	t.Logf("LECTURE : la MEDIANE est le chiffre qui juge le seuil — a %.3f m de cumul maximal, la "+
		"moitie des fenetres de decompte sont muettes, et %.2f m est huit fois au-dessus de ce "+
		"bruit. Les depassements ne sont PAS du jitter (p95 a %.3f m, c'est de la course) : ce "+
		"sont les matchs ou le T0-API tombe APRES le coup d'envoi, donc ou la fenetre contient "+
		"du jeu reel.", sc.mediane, t0mCumulM, sc.p95)
}

// t0mPublierSains publie la confrontation sur le corpus sain, variante par variante.
func t0mPublierSains(t *testing.T, premier, troisieme, mediane []float64, pires []t0mEcart) {
	t.Helper()
	t.Logf("")
	t.Logf("== CORPUS SAIN — delta = t0_film - t0_api ==")
	t.Logf("VARIANTE 1er joueur    : %s", t0mCalcStats(premier).ligne())
	t.Logf("VARIANTE 3e joueur     : %s", t0mCalcStats(troisieme).ligne())
	t.Logf("VARIANTE mediane joueur: %s", t0mCalcStats(mediane).ligne())
	dansUneSeconde, dansDeux := 0, 0
	for _, d := range premier {
		if math.Abs(d) <= 1000 {
			dansUneSeconde++
		}
		if math.Abs(d) <= 2000 {
			dansDeux++
		}
	}
	t.Logf("VARIANTE 1er joueur — |delta| <= 1 s : %d / %d = %.1f %% · <= 2 s : %d / %d = %.1f %%",
		dansUneSeconde, len(premier), pct100(dansUneSeconde, len(premier)),
		dansDeux, len(premier), pct100(dansDeux, len(premier)))
	sort.Slice(pires, func(i, j int) bool {
		return math.Abs(pires[i].delta) > math.Abs(pires[j].delta)
	})
	t.Logf("LES 5 PIRES ECARTS (variante 1er joueur) :")
	for i, p := range pires {
		if i >= 5 {
			break
		}
		t.Logf("  %-9s %-38s delta %+8.0f ms (t0_api %6d · t0_film %6d)",
			p.prefixe, t0mCoupe(p.pair, 38), p.delta, p.t0API, p.t0Film)
	}
}

// t0mPublierTemoinsFilm publie ce que le FILM dit tout seul, sans l'etalon : la rafale de
// depart (la grille se leve-t-elle d'un coup ?), l'accord interne du detecteur, et la CENSURE.
func t0mPublierTemoinsFilm(t *testing.T, marges, rafales, parts, ecarts []float64, censures int) {
	t.Helper()
	t.Logf("")
	t.Logf("== TEMOINS INTERNES AU FILM (aucun etalon en jeu) ==")
	sr, sp := t0mCalcStats(rafales), t0mCalcStats(parts)
	t.Logf("RAFALE DE DEPART — pistes qui bougent dans les %d ms du premier : mediane %.0f · "+
		"moyenne %.1f · min %.0f · max %.0f", t0mRafaleMS, sr.mediane, sr.moyenne, sr.mini, sr.maxi)
	t.Logf("  soit, en part de l'effectif (slots distincts) : mediane %.1f %% · moyenne %.1f %% · "+
		"p5 %.1f %%", sp.mediane, sp.moyenne, sp.p5)
	se := t0mCalcStats(ecarts)
	t.Logf("ACCORD INTERNE — ecart 3e mouvement - 1er mouvement : mediane %.0f ms · p95 %.0f ms · "+
		"max %.0f ms", se.mediane, se.p95, se.maxi)
	sm := t0mCalcStats(marges)
	t.Logf("MARGE — premier mouvement - frame 0 du film : mediane %.0f ms · p5 %.0f ms · "+
		"min %.0f ms", sm.mediane, sm.p5, sm.mini)
	t.Logf("CENSURE — matchs dont la marge est < %d ms (le film commence APRES le coup d'envoi, "+
		"t0_film n'est alors qu'une borne superieure) : %d / %d = %.1f %%",
		t0mMargeCensureMS, censures, len(marges), pct100(censures, len(marges)))
	// LA MARGE NON CENSUREE EST LE RESULTAT LE PLUS INATTENDU DE CETTE MESURE, et elle merite
	// sa propre ventilation : si elle est CONSTANTE, alors le coup d'envoi ne se lit pas dans le
	// mouvement mais dans `originMs` plus un decalage fixe — et le detecteur ne fait que
	// retrouver cette constante.
	var libres []float64
	for _, v := range marges {
		if v >= t0mMargeCensureMS {
			libres = append(libres, v)
		}
	}
	sl := t0mCalcStats(libres)
	t.Logf("MARGE, CENSURE RETIREE : %s", sl.ligne())
	dansLaFenetre := 0
	for _, v := range libres {
		if v >= 22000 && v <= 23500 {
			dansLaFenetre++
		}
	}
	t.Logf("  marges dans [22000, 23500] ms : %d / %d = %.1f %% — au-dela de ~90 %%, le coup "+
		"d'envoi est `originMs + constante` et le mouvement n'apporte rien",
		dansLaFenetre, len(libres), pct100(dansLaFenetre, len(libres)))
	t0mHistogramme(t, "MARGE (censure retiree)", libres, 2000, 12000, 40000)
}

// t0mHistogramme publie une ventilation par tranches de `pas` ms entre `bas` et `haut`.
func t0mHistogramme(t *testing.T, titre string, v []float64, pas, bas, haut float64) {
	t.Helper()
	if len(v) == 0 {
		return
	}
	seaux := map[int]int{}
	for _, x := range v {
		switch {
		case x < bas:
			seaux[-1]++
		case x >= haut:
			seaux[-2]++
		default:
			seaux[int((x-bas)/pas)]++
		}
	}
	t.Logf("  histogramme %s (tranches de %.0f ms) :", titre, pas)
	if n := seaux[-1]; n > 0 {
		t.Logf("    < %6.0f ms : %3d", bas, n)
	}
	for i := 0; float64(i)*pas+bas < haut; i++ {
		if n := seaux[i]; n > 0 {
			t.Logf("    %6.0f-%6.0f ms : %3d %s", float64(i)*pas+bas, float64(i+1)*pas+bas,
				n, strings.Repeat("#", n))
		}
	}
	if n := seaux[-2]; n > 0 {
		t.Logf("    >= %5.0f ms : %3d", haut, n)
	}
}

// t0mPublierCensure reprend la confrontation du corpus sain une fois la censure retiree, et
// compare la DISPERSION des deux horloges.
func t0mPublierCensure(t *testing.T, deltas, films, apis []float64) {
	t.Helper()
	t.Logf("")
	t.Logf("== CORPUS SAIN, CENSURE RETIREE (marge >= %d ms) ==", t0mMargeCensureMS)
	t.Logf("delta 1er joueur : %s", t0mCalcStats(deltas).ligne())
	sf, sa := t0mCalcStats(films), t0mCalcStats(apis)
	t.Logf("")
	t.Logf("== DISPERSION DES DEUX HORLOGES SUR LE CORPUS SAIN ==")
	t.Logf("t0_FILM  : mediane %.0f ms · ecart-type %.0f ms · p5 %.0f · p95 %.0f · min %.0f · max %.0f",
		sf.mediane, sf.ecartType, sf.p5, sf.p95, sf.mini, sf.maxi)
	t.Logf("t0_API   : mediane %.0f ms · ecart-type %.0f ms · p5 %.0f · p95 %.0f · min %.0f · max %.0f",
		sa.mediane, sa.ecartType, sa.p5, sa.p95, sa.mini, sa.maxi)
	t.Logf("LECTURE : un decompte pre-match est une constante de playlist ; l'horloge la MOINS " +
		"dispersee est la plus proche de cette constante.")
}

// t0mCoupe tronque un libelle pour la mise en colonnes.
func t0mCoupe(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "~"
}
