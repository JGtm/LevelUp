package replay

// visee_lunette_research_test.go — INSTRUMENT DE MESURE : LE FILM DIT-IL QU'UN JOUEUR EST A LA
// LUNETTE ?
//
// LA QUESTION. Dans Theater on voit l'epaulement : un joueur distant qui vise a la lunette prend
// une pose differente. La question posee est de savoir si cette pose est TRANSMISE par le film ou
// RECONSTRUITE par le client au moment du rendu.
//
// CE QUI EST DEJA CLOS, ET QUE CET INSTRUMENT NE REFAIT PAS :
//
//	- le composant `unit-zoom` EXISTE dans le moteur (chaine "unit-zoom : relevance=%5.3f",
//	  fonction de pertinence FUN_142f1e808) mais il n'est PAS dans le registre de replication du
//	  film : les 118 archetypes, 1067 couples (archetype, composant) et 325 noms distincts ont ete
//	  dumpes integralement et sont figes dans `filmdec/testdata/ecs_table.tsv`. Aucun composant de
//	  zoom, sur aucun archetype.
//	- `IsZoomed` / `GetZoomState` sont des champs de data-binding d'interface, pas de la
//	  replication — meme motif que `KillerWeapon`, deja refute dans ce depot.
//	- le descope part en TELEMETRIE Xbox (`DescopedUnitPosition`, `DescopedUnitAimVector`, ...),
//	  pas dans le film — meme schema que l'arme-de-kill.
//
// CE QUI RESTAIT OUVERT, ET CE QUE CET INSTRUMENT TRANCHE. Le composant i21
// `unit-desired-aiming-vector` porte, apres le couple (cap, elevation) publie depuis longtemps,
// TROIS DRAPEAUX et un SECOND VECTEUR que le port Go lisait puis jetait. Le zoom pouvait s'y
// cacher sous un autre nom. La grammaire est :
//
//	R(1) flag0 ; R(12) cap + R(11) elevation ; R(1) flag1 ;
//	si flag0 == 0 : R(12) cap + R(11) elevation (SECOND vecteur) ; R(1) flag2.
//
// LA MESURE. Un oracle ou l'epaulement est quasi certain : les kills au fusil de precision
// (S7 Sniper, Skewer, Stalker Rifle). Si l'un des drapeaux portait la lunette, il serait allume
// nettement plus souvent dans la seconde qui precede ces kills que dans le temoin.
//
// DEUX TEMOINS, parce qu'un seul ne separe pas les deux explications concurrentes :
//
//	TEMOIN FOND     tous les echantillons i21 hors de toute fenetre de kill. Il dit ce que vaut
//	                le drapeau « en general ».
//	TEMOIN TIR      les memes fenetres d'une seconde, mais avant les kills a une arme SANS
//	                lunette. C'est LUI qui discrimine : il partage tout avec la population de
//	                mesure (le joueur vise, il va tuer, il est en combat) SAUF l'arme. Un
//	                drapeau qui monte sur les deux ne dit rien du zoom, il dit « je tire ».
//
// SEUIL ET VERDICT, ECRITS AVANT LA MESURE (regle du depot, cf.
// `.ai/V7.5/film_re/METHODE_RETRO_INGENIERIE_FILM.md`) : voir le bloc de constantes
// `adsFacteurSeuil` / `adsEcartSeuil` / `adsMinFenetres` / `adsMinEchantillons` ci-dessous.
//
// CE QUE LA RETRO-INGENIERIE PREDIT — et c'est une prediction, pas un resultat : chez le
// PRODUCTEUR (FUN_142ee09a8, bloc de masque 0x200000) flag1 et flag2 sont les SIGNES d'un produit
// vectoriel 2D entre le vecteur transmis et un axe compagnon de l'unite, et flag0 est un drapeau
// de compression (« les deux directions coincident »). Si la mesure confirme, la reponse a
// l'utilisateur est un NON MESURE, et l'epaulement est reconstruit cote client.
//
// IL NE MODIFIE RIEN : lecture seule du film, aucune base, aucun artefact. SOUS GARDE
// D'ENVIRONNEMENT (ADS_FILM), donc saute partout ailleurs, CI comprise. UN SEUL FILM PAR
// PROCESSUS (le verrou de decodage du paquet filmdec est repris deux fois).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ADS_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run TestViseeLunette -v -timeout 60m

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

const (
	adsFilmEnv = "ADS_FILM" // repertoire des chunks du film
	adsTSVEnv  = "ADS_TSV"  // repertoire de sortie du releve (facultatif)
)

// LES SEUILS, ECRITS AVANT LA MESURE.
const (
	// adsFenetreMS : la fenetre AMONT retenue avant un kill. Une seconde, parce que c'est la
	// duree que l'utilisateur decrit (« a la lunette au moment du tir ») et parce qu'a ~60 Hz de
	// replication elle contient plusieurs dizaines de records i21.
	adsFenetreMS = 1000
	// adsMinFenetres : sans au moins 30 fenetres de precision, aucun taux n'est publiable.
	adsMinFenetres = 30
	// adsMinEchantillons : idem pour le nombre d'echantillons i21 de chaque population.
	adsMinEchantillons = 200
	// adsFacteurSeuil / adsEcartSeuil : un drapeau n'est retenu comme CANDIDAT LUNETTE que si son
	// taux dans les fenetres de precision vaut au moins DEUX FOIS celui du temoin TIR *et* le
	// depasse d'au moins 20 points. Le facteur seul se laisse tromper par les petits taux, l'ecart
	// seul par les gros : il faut les deux.
	adsFacteurSeuil = 2.0
	adsEcartSeuil   = 0.20
)

// adsArmesLunette : les etiquettes de source de degat retenues comme fusils de precision. Elles
// viennent du catalogue `damagetag` (data/labels.tsv) et sont ecrites en toutes lettres pour
// qu'un changement de catalogue casse la mesure au lieu de la fausser en silence.
//
// `Mangler / Ravager / Shock Rifle / Skewer` est VOLONTAIREMENT absent : cette etiquette est
// ambigue, elle melangerait des armes sans lunette dans la population de mesure.
var adsArmesLunette = map[string]bool{
	"S7 Sniper":     true,
	"Skewer":        true,
	"Stalker Rifle": true,
}

// adsPop est UNE population d'echantillons i21 et ses compteurs de drapeaux.
type adsPop struct {
	nom string
	// fenetres : nombre de fenetres de kill ayant fourni au moins un echantillon (0 pour le fond).
	fenetres int
	// n : echantillons i21 retenus (HasYaw). f0/f1 : drapeaux allumes sur ces n.
	n, f0, f1 int
	// nB : echantillons portant le SECOND vecteur (donc flag0 == 0) ; f2 : flag2 allume sur nB.
	nB, f2 int
	// ecartAB : somme des ecarts de cap |A - B| en degres, sur les nB echantillons.
	ecartAB float64
}

func (p *adsPop) ajoute(s filmdec.BipedPosition) {
	p.n++
	if s.AimFlag0 {
		p.f0++
	}
	if s.AimFlag1 {
		p.f1++
	}
	if !s.HasAimB {
		return
	}
	p.nB++
	if s.AimFlag2 {
		p.f2++
	}
	p.ecartAB += adsEcartCapDeg(s.YawRaw, s.YawRawB)
}

// adsEcartCapDeg rend l'ecart angulaire absolu entre deux caps quantifies R(12), en degres,
// ramene dans [0,180].
func adsEcartCapDeg(a, b uint32) float64 {
	const pas = 360.0 / 4096.0
	d := math.Abs(float64(a)-float64(b)) * pas
	if d > 180 {
		d = 360 - d
	}
	return d
}

// adsTaux rend un taux et le libelle « n/d » qui va avec.
func adsTaux(num, den int) (float64, string) {
	if den == 0 {
		return 0, "0/0"
	}
	return float64(num) / float64(den), fmt.Sprintf("%d/%d", num, den)
}

// TestViseeLunette execute la mesure sur UN film.
func TestViseeLunette(t *testing.T) {
	dir := os.Getenv(adsFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", adsFilmEnv)
	}

	// killsource prend LUI-MEME le verrou de decodage du paquet : il doit donc tourner AVANT
	// notre propre section verrouillee, jamais dedans (le verrou n'est pas reentrant).
	armes := adsArmesParInstant(t, dir)

	release := filmdec.LockProcessDecode()
	defer release()
	pos, tracks, own := adsBalayage(t, dir)
	couples, nKills, ambigus := aimCouples(t, dir)
	t.Logf("FIL — %d instants de kill, %d couples retenus, %d ambigus ecartes", nKills, len(couples), ambigus)

	parXUID := map[uint64][]uint32{}
	for slot, x := range own.SlotXUID {
		parXUID[x] = append(parXUID[x], slot)
	}
	if len(parXUID) == 0 {
		t.Fatalf("pont slot->xuid vide : aucun kill ne peut etre rattache a une visee")
	}

	mesure, temoinTir, fenetres, joints := adsPopulations(couples, armes, tracks, parXUID, own.DeathOffsetMS)
	fond := adsFond(pos, fenetres)
	total := adsTotal(pos)
	t.Logf("JOINTURE — %d couples du fil apparies a une source de degat killsource (%d sans arme connue)",
		joints, len(couples)-joints)

	adsJournalise(t, total, mesure, temoinTir, fond)
	clos := adsConstance(t, total)
	adsVerdict(t, mesure, temoinTir, fond, clos)
	adsEcrisTSV(t, dir, total, mesure, temoinTir, fond)
}

// adsTotal totalise TOUS les records i21 du film, sans decoupage : c'est le denominateur du
// controle de constance.
func adsTotal(pos []filmdec.BipedPosition) adsPop {
	p := adsPop{nom: "FILM ENTIER (tous les records i21)"}
	for _, e := range pos {
		if e.HasYaw {
			p.ajoute(e)
		}
	}
	return p
}

// adsConstance est le CONTROLE DE CONSTANCE, et il domine l'oracle : un bit qui ne prend qu'une
// seule valeur sur tout le film ne peut coder aucun etat, et aucune population, si grande soit-elle,
// n'y changera rien. Il rend `true` quand les trois drapeaux sont constants — la question est alors
// close sans avoir besoin du seuil de l'oracle.
//
// CE CONTROLE A ETE AJOUTE APRES LA PREMIERE PASSE, et c'est dit ici plutot que masque : la
// premiere passe a montre des taux a 0 % et 100 % exacts, ce que le seuil de l'oracle — ecrit,
// lui, avant la mesure — ne savait pas nommer. Ce n'est pas un seuil deplace : c'est une
// OBSERVATION BRUTE, sans parametre, et elle est plus forte que le seuil qu'elle remplace.
//
// CONTROLE CROISE INDEPENDANT de flag0 : la table ECS (`filmdec/testdata/ecs_table.tsv`, ti=35
// i21) donne une largeur MESUREE de 25 bits, relevee par une tout autre voie (l'observation de
// largeur du decodeur de trame). Or 25 = 1 + 23 + 1 est exactement la branche flag0 == 1. Les deux
// chaines n'ont aucune etape commune et disent la meme chose.
func adsConstance(t *testing.T, p adsPop) bool {
	t.Helper()
	if p.n == 0 {
		t.Log("CONSTANCE — aucun record i21 lu : controle impossible")
		return false
	}
	c0 := p.f0 == 0 || p.f0 == p.n
	c1 := p.f1 == 0 || p.f1 == p.n
	c2 := p.nB == 0 || p.f2 == 0 || p.f2 == p.nB
	t.Logf("CONSTANCE — sur %d records i21 du film : flag0 %s · flag1 %s · second vecteur %s"+
		" · flag2 %s", p.n, adsConstat(c0, p.f0, p.n), adsConstat(c1, p.f1, p.n),
		adsConstat(p.nB == 0 || p.nB == p.n, p.nB, p.n), adsConstat(c2, p.f2, p.nB))
	if c0 && c1 && c2 {
		t.Log("CONSTANCE — LES TROIS DRAPEAUX SONT CONSTANTS sur tout le film. Un bit constant ne" +
			" code aucun etat : la question « le film porte-t-il la lunette dans i21 » est CLOSE" +
			" par la negative, independamment de la taille des populations de l'oracle.")
		return true
	}
	return false
}

func adsConstat(constant bool, num, den int) string {
	if den == 0 {
		return "absent (0 record)"
	}
	if constant {
		if num == 0 {
			return fmt.Sprintf("CONSTANT a 0 (0/%d)", den)
		}
		return fmt.Sprintf("CONSTANT a 1 (%d/%d)", num, den)
	}
	return fmt.Sprintf("variable (%d/%d)", num, den)
}

// adsBalayage lit les positions du film et construit le pont slot -> joueur. Les bornes de carte
// ne sont PAS demandees : cette mesure ne porte que sur des drapeaux, aucun metre n'y intervient.
func adsBalayage(t *testing.T, dir string) ([]filmdec.BipedPosition, map[uint32]slotTrack, OwnerReport) {
	t.Helper()
	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.QuantaOnly = true // aucun metre n'intervient ici : les bornes de carte seraient du poids mort
	debut := time.Now()
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	t.Logf("COUT — ScanFilmBipedPositions : %d positions en %s", len(pos), time.Since(debut).Round(time.Millisecond))

	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("index de joueur : %v", err)
	}
	table, collisions := injectiveOrEmpty(idx)
	tracks := indexBySlot(pos)
	own := buildOwners(tracks, deaths, table, nil)
	t.Logf("PONT — %d slots nommes sur %d vies · decalage d'horloge %d ms (%d fins de vie appariees)"+
		" · collisions d'index %d", len(own.SlotXUID), own.LivesTotal, own.DeathOffsetMS,
		own.DeathOffsetMatches, collisions)
	return pos, tracks, own
}

// adsArmesParInstant rend, pour chaque instant de kill du film, l'etiquette de la SOURCE DU DEGAT
// fatal lue par killsource. Les instants qui portent plusieurs morts sont ECARTES : leur associer
// une arme unique serait une invention.
func adsArmesParInstant(t *testing.T, dir string) map[int]killsource.SourceTruth {
	t.Helper()
	src, err := killsource.DirChunks(dir)
	if err != nil {
		t.Fatalf("chunks du film : %v", err)
	}
	debut := time.Now()
	res, err := killsource.Decode(context.Background(), filepath.Base(dir), src, nil)
	if err != nil {
		t.Fatalf("killsource : %v", err)
	}
	t.Logf("COUT — killsource.Decode : %d kills en %s", len(res.Kills), time.Since(debut).Round(time.Millisecond))

	compte := map[int]int{}
	out := map[int]killsource.SourceTruth{}
	for _, k := range res.Kills {
		compte[k.TimeMS]++
		out[k.TimeMS] = k.Source
	}
	multiples := 0
	for ms, n := range compte {
		if n > 1 {
			delete(out, ms)
			multiples++
		}
	}
	t.Logf("SOURCES — %d instants a source unique, %d instants multiples ecartes", len(out), multiples)
	return out
}

// adsFenetre borne une fenetre amont sur l'horloge du film, en millisecondes.
type adsFenetre struct{ debut, fin int64 }

// adsPopulations construit les deux populations de fenetres. Elle rend aussi la liste de TOUTES
// les fenetres retenues (les deux confondues), dont le temoin FOND est le complementaire.
func adsPopulations(couples []aimCouple, armes map[int]killsource.SourceTruth,
	tracks map[uint32]slotTrack, parXUID map[uint64][]uint32, offMS int64,
) (adsPop, adsPop, []adsFenetre, int) {
	mesure := adsPop{nom: "PRECISION (fusil a lunette)"}
	temoin := adsPop{nom: "TEMOIN TIR (arme sans lunette)"}
	var fenetres []adsFenetre
	joints := 0
	for _, c := range couples {
		src, ok := armes[int(c.tMS)]
		if !ok {
			continue
		}
		joints++
		lunette := adsArmesLunette[src.Display]
		if !lunette && src.Class != damagetag.ClassArme {
			continue // ni fusil a lunette, ni arme tenue : hors des deux populations
		}
		fin := c.tMS + offMS
		f := adsFenetre{debut: fin - adsFenetreMS, fin: fin}
		n := adsCollecte(tracks, parXUID[c.tueur], f, adsCible(&mesure, &temoin, lunette))
		if n == 0 {
			continue
		}
		fenetres = append(fenetres, f)
		if lunette {
			mesure.fenetres++
		} else {
			temoin.fenetres++
		}
	}
	return mesure, temoin, fenetres, joints
}

func adsCible(mesure, temoin *adsPop, lunette bool) *adsPop {
	if lunette {
		return mesure
	}
	return temoin
}

// adsCollecte verse dans `p` tous les echantillons i21 du tueur situes dans la fenetre, et rend
// leur nombre.
func adsCollecte(tracks map[uint32]slotTrack, slots []uint32, f adsFenetre, p *adsPop) int {
	n := 0
	for _, s := range slots {
		for _, e := range tracks[s].pts {
			ms := int64(e.TimestampUS) / 1000
			if ms < f.debut || ms > f.fin || !e.HasYaw {
				continue
			}
			p.ajoute(e)
			n++
		}
	}
	return n
}

// adsFond construit le temoin FOND : tous les echantillons i21 hors de TOUTE fenetre de kill.
func adsFond(pos []filmdec.BipedPosition, fenetres []adsFenetre) adsPop {
	p := adsPop{nom: "TEMOIN FOND (hors fenetre de kill)"}
	sort.Slice(fenetres, func(i, j int) bool { return fenetres[i].debut < fenetres[j].debut })
	for _, e := range pos {
		if !e.HasYaw {
			continue
		}
		if adsDansUneFenetre(fenetres, int64(e.TimestampUS)/1000) {
			continue
		}
		p.ajoute(e)
	}
	return p
}

func adsDansUneFenetre(fenetres []adsFenetre, ms int64) bool {
	i := sort.Search(len(fenetres), func(i int) bool { return fenetres[i].debut > ms })
	// Les fenetres ont toutes la meme duree : il suffit de remonter tant que `debut` peut encore
	// couvrir `ms`.
	for j := i - 1; j >= 0 && fenetres[j].debut >= ms-adsFenetreMS; j-- {
		if ms >= fenetres[j].debut && ms <= fenetres[j].fin {
			return true
		}
	}
	return false
}
