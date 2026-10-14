//go:build research

package grammar

// aj14f_poussee_research_test.go — LOT F1/F2 DU PLAN D AJUSTEMENTS DU 2026-10-14 : LES MESSAGES
// D EQUIPEMENT DE LA VUE A, LUS PAR LA GRAMMAIRE DE PRODUCTION.
//
// La vue A se lit message par message jusqu a son terminateur ([lireLaVueA]) ; la production n en
// garde que les genres, les kills et le fil. Cet instrument relit CHAQUE trame delta avec la meme
// grammaire (memes domaines de reference, memes charges) en gardant, pour les genres suivis, les
// trois references et les champs de la charge, puis VERIFIE que sa lecture finit au meme bit que la
// lecture de production et rend les memes genres : une divergence arrete l instrument.
//
// Genres suivis : 103 EquipmentSpawnedObject, 104 EquipmentKnockbackPlayer, 105
// EquipmentObjectKnockedBack, 117 EquipmentTranslocatorTeleportEffects, 119
// EquipmentKnockbackRequest, 31 equipment_teleport_request, 30 biped_equipment_activation.
//
// GARDES : `AJ14F_FILMS` (racine des chunks), `AJ14F_IDS` (liste `id8=map_id`, `;`), catalogue de
// bornes du depot. Lecture seule, aucune ecriture, `CGO_ENABLED=0`.
//
//	CGO_ENABLED=0 AJ14F_FILMS=<repo>/data/cache/film_chunks AJ14F_IDS='0d9a9af9=e8d56863-...' \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestAj14fMessagesDEquipement$' -count=1 -timeout 60m -v

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

var aj14fGenresSuivis = map[int]string{
	30: "biped_equipment_activation", 31: "equipment_teleport_request",
	103: "EquipmentSpawnedObject", 104: "EquipmentKnockbackPlayer",
	105: "EquipmentObjectKnockedBack", 117: "EquipmentTranslocatorTeleportEffects",
	119: "EquipmentKnockbackRequest", 82: "PlayerGameEventSmall", 15: "Script",
}

type aj14fRef struct {
	P      bool
	V, Gen uint64
}

type aj14fMsg struct {
	Genre, Rang int
	Refs        [3]aj14fRef
	Champs      []uint64
}

func TestAj14fMessagesDEquipement(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	if racine == "" || liste == "" {
		t.Skip("AJ14F_FILMS ou AJ14F_IDS absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		origine, err := ScanClockOrigin(film)
		if err != nil {
			t.Fatalf("%s : origine : %v", id, err)
		}
		aj14fEntree = &entree
		aj14fUnFilm(t, id, NewFilmContextForMap(film, &entree, nil), origine)
	}
}

func aj14fUnFilm(t *testing.T, id string, fc *FilmContext, origine uint64) {
	t.Helper()
	bal, g := fc.profilDeLaVueA(), fc.grammaireDeLaVueA()
	var paquets, listes, portees, arretees int
	compte := map[int]int{}
	for _, num := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			paquets++
			prod := lireLaVueA(pay, 1, bal, g)
			msgs, fin, genres := aj14fLire(pay, 1, bal, g)
			if fin != prod.Fin || fmt.Sprint(genres) != fmt.Sprint(prod.Genres) {
				t.Fatalf("%s chunk %d paquet %d : lecture divergente (fin %d contre %d, genres %v contre %v)",
					id, num, pk.Index, fin, prod.Fin, genres, prod.Genres)
			}
			if len(prod.Genres) > 0 {
				listes++
				if prod.Porte {
					portees++
				} else {
					arretees++
					if os.Getenv("AJ14F_ARRETS") != "" {
						t.Logf("%s %9.1f ms (%s) ARRET liste %v (fin %d / %d bits)", id,
							float64(pk.TimestampUS-origine)/1000, aj14fMMSS(float64(pk.TimestampUS-origine)/1000),
							prod.Genres, prod.Fin, len(pay)*8)
					}
				}
			}
			ms := float64(pk.TimestampUS-origine) / 1000
			if lo, hi, ok := aj14fFenetre(); ok && ms >= lo && ms <= hi && len(prod.Genres) > 0 {
				t.Logf("%s %9.1f ms FENETRE liste %v porte=%v", id, ms, prod.Genres, prod.Porte)
			}
			for _, m := range msgs {
				compte[m.Genre]++
				t.Logf("%s %9.1f ms (%s) genre %3d %-38s rang %d/%d porte=%v refs %s champs %v liste %v", id, ms,
					aj14fMMSS(ms), m.Genre, aj14fGenresSuivis[m.Genre], m.Rang, len(prod.Genres), prod.Porte,
					aj14fRefs(m.Refs), m.Champs, prod.Genres)
				if m.Genre == 104 && len(m.Champs) == 3 {
					v := aj14fImpulsion(m.Champs[1], m.Champs[2])
					t.Logf("%s %9.1f ms IMPULSION %.2f %.2f %.2f (%.2f m/s)", id, ms, v[0], v[1], v[2], math.Hypot(math.Hypot(v[0], v[1]), v[2]))
				}
			}
		}
	}
	t.Logf("%s : %d paquets delta, %d listes (%d portees jusqu au terminateur, %d arretees) ; suivis %v",
		id, paquets, listes, portees, arretees, compte)
}

func aj14fMMSS(ms float64) string {
	s := ms / 1000
	return fmt.Sprintf("%d:%04.1f", int(s)/60, s-float64(int(s)/60*60))
}

func aj14fRefs(r [3]aj14fRef) string {
	var b strings.Builder
	for i, x := range r {
		if i > 0 {
			b.WriteString(" ")
		}
		if !x.P {
			b.WriteString("-")
			continue
		}
		fmt.Fprintf(&b, "%d/%d", x.V, x.Gen)
	}
	return b.String()
}

// aj14fLire rejoue [lireLaVueA] en gardant les references et les champs des genres suivis. La
// charge de production est rejouee a chaque message : la lecture des champs ne decide de rien.
func aj14fLire(pay []byte, debut int, bal ProfilDeBalayage, g grammaireDeLaVueA) ([]aj14fMsg, int, []int) {
	br := LecteurSur(pay)
	br.PoserProfil(bal)
	br.vueA = g
	frameLen := len(pay) * 8
	lisible := false
	if debut >= 1 && debut <= frameLen {
		br.SetBitPos(debut - 1)
		lisible = br.ReadBit() && g.classe != vueAIllisible
	}
	var out []aj14fMsg
	var genres []int
	br.SetBitPos(debut)
	for placeDisponible(br, frameLen, 1) {
		if !br.ReadBit() {
			break
		}
		if !placeDisponible(br, frameLen, LargeurGenreVueA) {
			break
		}
		genre := int(br.ReadBits(LargeurGenreVueA))
		genres = append(genres, genre)
		finDuGenre := br.BitPos()
		if !lisible || genre >= br.vueA.genres {
			br.SetBitPos(finDuGenre)
			break
		}
		m := aj14fMsg{Genre: genre, Rang: len(genres)}
		domaines, vide := descripteurDuGenre(genre)
		for i, dom := range domaines {
			if br.ReadBit() {
				v, gen := readVarWidthInt(br, dom)
				m.Refs[i] = aj14fRef{P: true, V: v, Gen: gen}
			}
		}
		ok := true
		if !vide {
			avant := br.BitPos()
			m.Champs = aj14fChamps(br, genre)
			br.SetBitPos(avant)
			charge := chargeDuGenre(genre)
			ok = charge != nil && charge(br)
			if ok && br.p.Grammaire.ControleDeCorruption && br.ReadBit() {
				br.Skip(32)
			}
		}
		if !ok || br.Deborde() {
			br.SetBitPos(finDuGenre)
			break
		}
		if _, suivi := aj14fGenresSuivis[genre]; suivi {
			out = append(out, m)
		}
	}
	return out, br.BitPos(), genres
}

// aj14fChamps lit les champs de charge des genres suivis, dans l ordre des lecteurs du jeu
// (cf. `vue_a_charges_armes.go`).
func aj14fChamps(br *Lecteur, genre int) []uint64 {
	switch genre {
	case 104: // FUN_14116c344 -> FUN_14076d528 : R(1) ; si 0 : R(19) direction, R(10) magnitude
		if br.ReadBit() {
			return []uint64{1}
		}
		return []uint64{0, br.ReadBits(largeurDirection), br.ReadBits(largeurMagnitudePoussee)}
	case 105: // R(1) ; si 1 : R(32)
		if br.ReadBit() {
			return []uint64{1, br.ReadBits(32)}
		}
		return []uint64{0}
	case 119: // R(19) direction, R(4)+1 cibles, chacune : reference de domaine 0, R(19)
		out := []uint64{br.ReadBits(largeurDirection)}
		n := br.ReadBits(4) + 1
		out = append(out, n)
		for range n {
			v, gen := readVarWidthInt(br, 0)
			out = append(out, v, gen, br.ReadBits(largeurDirection))
		}
		return out
	case 31:
		return []uint64{br.ReadBits(4)}
	case 82:
		return aj14fEvenementCourt(br)
	case 15:
		var out []uint64
		if br.vueA.script == scriptAvecPrefixe {
			out = append(out, br.ReadBits(largeurPrefixeScript))
		}
		out = append(out, 5555, br.ReadBits(largeurTeteScript))
		n := br.ReadBits(largeurLongueurScript)
		out = append(out, n)
		for i := uint64(0); i+32 <= n; i += 32 {
			out = append(out, br.ReadBits(32))
		}
		return out
	}
	return nil
}

// aj14fImpulsion rend la vitesse que le 104 impose (FUN_14076d528 : direction R(19) par
// FUN_1406d8288, norme par FUN_14076d6dc sous min 0,05 et max 20 sur 10 bits).
func aj14fImpulsion(dir, q uint64) [3]float64 {
	x, y, z := DecodeAimVector(uint32(dir), largeurDirection)
	const mn, mx, n = 0.05, 20.0, 1024
	c := 1 - mn
	var m float64
	switch {
	case q == 0:
		m = mn
	case q >= n-1:
		m = mx
	default:
		pas := math.Log(c+mx) / n
		m = math.Exp(float64(q)*pas+pas*0.5) - c
	}
	return [3]float64{float64(x) * m, float64(y) * m, float64(z) * m}
}

// aj14fFenetre lit `AJ14F_FENETRE=debut,fin` (ms de film) : les listes de la fenetre sont journalisees.
func aj14fFenetre() (float64, float64, bool) {
	a, b, ok := strings.Cut(os.Getenv("AJ14F_FENETRE"), ",")
	if !ok {
		return 0, 0, false
	}
	lo, e1 := strconv.ParseFloat(a, 64)
	hi, e2 := strconv.ParseFloat(b, 64)
	return lo, hi, e1 == nil && e2 == nil
}

// aj14fEvenementCourt lit les champs de `PlayerGameEventSmall` (chargeEvenementJoueurCourt) : type
// R(32), R(8), puis pour chaque propriete du sac (nom R(32), etiquette R(3), valeur a 32 bits ou
// a un bit), le sac texte (nom, emplacements) et le masque des destinataires. Une etiquette de
// chaine ou de position rend la liste lue jusque-la.
// aj14fEntree est l entree de catalogue du film en cours (instrument mono-film par processus).
var aj14fEntree *profile.MapQuantEntry

func aj14fEvenementCourt(br *Lecteur) []uint64 {
	out := []uint64{br.ReadBits(32), br.ReadBits(8)}
	n := br.ReadBits(largeurCompteSacCourt)
	out = append(out, n)
	for range n {
		nom, et := br.ReadBits(32), br.ReadBits(largeurEtiquetteDePropriete)
		out = append(out, nom, et)
		switch et {
		case 0:
		case 1, 2, 3, 6:
			out = append(out, br.ReadBits(32))
		case 4:
			out = append(out, br.ReadBits(1))
		case 5:
			return out
		default:
			v, ok := readTranslocVec(br, aj14fEntree)
			if !ok {
				return append(out, 9999)
			}
			out = append(out, uint64(math.Round(float64(v[0])*100)+1e6), uint64(math.Round(float64(v[1])*100)+1e6),
				uint64(math.Round(float64(v[2])*100)+1e6))
		}
	}
	sac := consumeSacTexte(br)
	out = append(out, 7777, sac.Nom)
	for _, e := range sac.Entrees {
		out = append(out, uint64(e.SousType), uint64(e.Valeur))
	}
	return append(out, 8888, br.ReadBits(ParticipantsDuFil))
}

// TestAj14fScanKnockbacks rend les poussees du balayage de production.
func TestAj14fScanKnockbacks(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	if racine == "" || liste == "" {
		t.Skip("instrument saute")
	}
	cat, _ := profile.LoadMapQuantCatalog(e191bCatalogue())
	id, carte, _ := strings.Cut(liste, "=")
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte : %v", err)
	}
	film, ok, err := filmcache.LoadFilm(racine, id)
	if err != nil || !ok {
		t.Fatalf("film : %v", err)
	}
	origine, _ := ScanClockOrigin(film)
	ks, st := ScanKnockbacks(NewFilmContextForMap(film, &entree, nil))
	t.Logf("%s : %+v", id, st)
	for _, k := range ks {
		t.Logf("%s %9.1f ms pousse %+v auteur %+v (%v) impulsion %.2f (%v)", id, float64(k.TimestampUS-origine)/1000,
			k.Pushed, k.Author, k.HasAuthor, k.Impulse, k.HasImpulse)
	}
}
