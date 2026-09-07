package replay

// duels_sonde_research_test.go — SONDE DE FAISABILITE d'une feature « duels » lue dans le film.
//
// LA QUESTION, POSEE PAR L'UTILISATEUR : peut-on dire combien de duels un joueur a livres et
// combien il en a gagnes ? Le critere retenu N'EST PAS geometrique (qui regardait qui) mais
// LA RECIPROCITE DU DEGAT : un duel est un echange ou le degat circule DANS LES DEUX SENS.
// Un tir dans le dos, ou la victime n'a jamais rien place, n'est pas un duel — c'est une
// elimination. Ce critere a l'avantage de ne rien supposer d'une intention : il se mesure.
//
// CE QUE CETTE SONDE MESURE, ET POURQUOI CHAQUE MESURE EXISTE :
//
//	M0 pont d'index — part des degats dont LES DEUX slots (blesse, responsable) tombent sur un
//	   bipede connu. Sans ce pont, aucune des mesures suivantes n'a de sens.
//	M1 reciprocite — part des morts dont la victime avait inflige un degat a son tueur dans la
//	   fenetre. C'EST LE CHIFFRE QUI DECIDE : s'il est absurdement bas, c'est la lacune connue du
//	   flux de degats (cf. la remise de la precision par arme du 2026-09-01) qui reparle, et la
//	   feature reclasserait en masse des duels en eliminations.
//	M2 temoin — le meme calcul sur un flux de degats DECALE dans le temps. La reciprocite doit
//	   s'effondrer ; si elle tient, on mesure une densite d'evenements et non un fait de jeu.
//	M3 distance — part des duels dont LES DEUX positions se resolvent a l'instant du premier
//	   degat, et l'histogramme des distances. Repond a « peut-on croiser duel x portee ».
//	M4 trio — part des duels ou un TIERS a inflige ou subi du degat dans la meme fenetre.
//
// CE QU'ELLE NE MESURE PAS, ET C'EST ASSUME : l'EQUIPE des participants. La sonde travaille sur
// des SLOTS (une vie), pas sur des joueurs — le pont slot -> xuid existe (ResolveSlotXUID) mais
// exige le roster, donc la base, et la sonde reste hors ligne. Consequence : M4 dit « un tiers
// est intervenu », jamais « dans mon camp » ou « contre moi ».
//
// SEUILS ECRITS AVANT LA MESURE (sinon on ajuste le seuil au resultat) :
//
//	M0 >= 80 % — en dessous, le pont d'index est le premier chantier, pas les duels.
//	M1 >= 40 % — un joueur d'arene perd la majorite de ses vies en combat, pas en execution.
//	M2 rapport M1/temoin >= 3 — en deca, la reciprocite est un artefact de densite.
//	M3 >= 70 % — en dessous, la distance est une donnee d'appoint, pas un axe d'analyse.
//
// USAGE (un film par process, verrou de decodage pris) :
//
//	CGO_ENABLED=0 DUELS_FILM=<repo>/data/cache/film_chunks/000d5950 DUELS_MAP=Cliffhanger \
//	  go test ./internal/analysis/replay -run TestSondeDuels -v -timeout 900s

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

const (
	duelsFilmEnv = "DUELS_FILM"
	duelsMapEnv  = "DUELS_MAP"
)

// duelsRiposteWindowsUS : les fenetres de riposte mesurees. TROIS valeurs et non une : le choix
// du seuil est justement ce que la sonde doit eclairer ; l'epingler d'avance preterait au film
// une cadence d'echange qu'on n'a pas mesuree.
var duelsRiposteWindowsUS = []uint64{2_000_000, 3_000_000, 5_000_000}

// duelsKillWindowUS : fenetre amont de recherche du degat FATAL d'une mort. Large a dessein — on
// cherche le DERNIER degat recu, pas tous ; l'elargir ne peut qu'aider a en trouver un.
const duelsKillWindowUS = 10_000_000

// duelsShiftUS : decalage du temoin M2. Grand devant toute fenetre de riposte, et sans rapport
// simple avec les cadences de match (respawn ~8 s), pour ne pas retomber en phase.
const duelsShiftUS = 37_000_000

// duelsPosTolUS : tolerance evenement <-> echantillon de position. MEME valeur que
// filmdec.WeaponHitPosToleranceUS et shots.go (120 ms).
const duelsPosTolUS = filmdec.WeaponHitPosToleranceUS

// duelDmg : un degat direct, ses deux slots deja resolus.
type duelDmg struct {
	ts               uint64
	victim, attacker uint32
}

// duelMort : une fin de vie appariee a une mort du fil — donc une VRAIE mort, datee sur
// l'horloge du film.
type duelMort struct {
	slot uint32
	ts   uint64
}

func TestSondeDuels(t *testing.T) {
	dir := os.Getenv(duelsFilmEnv)
	carte := os.Getenv(duelsMapEnv)
	if dir == "" || carte == "" {
		t.Skipf("sonde desactivee : %s et %s requis", duelsFilmEnv, duelsMapEnv)
	}

	release := filmdec.LockProcessDecode()
	defer release()

	rng := duelsBornes(t, carte)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s illisible : %v", dir, err)
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &rng
	// CaptureDirs poursuit le MEME record de deux composants de plus (i4 vie, i5 bouclier) :
	// c'est ce qui donne M5, et ca ne change aucune position emise (cf. ScanFilmOptions).
	opt.CaptureDirs = true
	positions, err := filmdec.ScanBipedPositions(film, opt)
	if err != nil {
		t.Fatalf("positions bipeds : %v", err)
	}
	tracks := indexBySlot(positions)

	lives := buildLifeSpans(tracks)
	morts := duelsMorts(t, film, lives)
	brut, baseScan := duelsScanDegats(t, dir)
	base, dmg := duelsResoudreBase(t, brut, duelsViesParSlot(lives))

	t.Logf("FILM %s (%s) : %d positions, %d slots, %d morts appariees, %d degats bruts",
		filepath.Base(dir), carte, len(positions), len(tracks), len(morts), len(brut))
	t.Logf("M0 pont d'index : base %d (argmax du scan : %d) — %d/%d degats resolus = %s",
		base, baseScan, len(dmg), len(brut), duelsPct(len(dmg), len(brut)))

	duelsRecensementC0(t, dir)
	duelsMesureReciprocite(t, morts, dmg)
	duelsMesureDistance(t, morts, dmg, tracks)
	duelsMesureTrio(t, morts, dmg)

	chutes, lectures := duelsChutesBouclier(positions, duelsViesParSlot(lives))
	duelsMesureBouclier(t, morts, chutes, lectures, duelsViesParSlot(lives))
}

// duelsBornes charge les bornes de dequantification de la carte depuis le catalogue VERSIONNE.
func duelsBornes(t *testing.T, carte string) filmdec.Vec3Range {
	t.Helper()
	chemin := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", chemin, err)
	}
	entry, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue : %v", carte, err)
	}
	return entry.Range()
}

// duelsMorts rend les fins de vie APPARIEES a une mort du fil : la population des vraies morts.
// Compose les fonctions eprouvees de lives.go — aucune seconde lecture du fil des morts (la
// regle « deux decodeurs du meme fait divergeraient », cf. l'en-tete de killpos_bridge.go).
func duelsMorts(t *testing.T, film *filmsource.Film, lives []lifeSpan) []duelMort {
	t.Helper()
	deaths, err := ScanDeaths(film)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	off, _, _ := bestDeathOffset(lives, deaths)
	nommees := nameLivesByDeaths(lives, deaths, off)
	if nommees == 0 {
		t.Fatalf("aucune vie nommee (%d vies, %d morts) : le pont mort <-> vie a echoue",
			len(lives), len(deaths))
	}
	out := make([]duelMort, 0, nommees)
	for _, l := range lives {
		if l.xuid != 0 {
			out = append(out, duelMort{slot: l.slot, ts: uint64(l.to)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out
}

// duelsScanDegats decode les damage_aftermath du film. Le decodage est PRODUCTIONISE
// (ScanFilmWeaponDamages) : cet adaptateur ne fait que fournir le registre et le nombre de chunks.
func duelsScanDegats(t *testing.T, dir string) ([]filmdec.WeaponDamage, int) {
	t.Helper()
	raw, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	dmg, base, err := filmdec.ScanFilmWeaponDamages(dir, reg, filmdec.CountFilmChunks(dir))
	if err != nil {
		t.Fatalf("collecte des degats : %v", err)
	}
	return dmg, base
}

// duelsViesParSlot indexe les vies par slot, pour repondre a « ce slot etait-il VIVANT a t ».
func duelsViesParSlot(lives []lifeSpan) map[uint32][]lifeSpan {
	out := map[uint32][]lifeSpan{}
	for _, l := range lives {
		out[l.slot] = append(out[l.slot], l)
	}
	return out
}

// duelsVivant dit si le slot avait une vie en cours a ts. La tolerance est lifeGapUS, la MEME
// que celle qui decoupe les vies : plus serre, on rejetterait un degat tombe dans un trou de
// replication ; plus large, deux vies successives se confondraient.
func duelsVivant(vies map[uint32][]lifeSpan, slot uint32, ts uint64) bool {
	for _, l := range vies[slot] {
		if int64(ts) >= l.from-lifeGapUS && int64(ts) <= l.to+lifeGapUS {
			return true
		}
	}
	return false
}

// duelsResoudreBase calibre la base d'atterrissage par BALAYAGE.
//
// LE CRITERE N'EST PAS « le slot existe » MAIS « le slot etait VIVANT a l'instant du degat ».
// La difference n'est pas cosmetique : un slot existe pendant tout le film, donc le critere
// d'existence est satisfait par presque n'importe quelle base — il note la DENSITE des slots,
// pas la justesse de la base. La vivacite, elle, date : une base fausse fait tomber les degats
// sur des slots morts ou pas encore nes, et son score s'effondre. C'est la meme logique que le
// plateau de bestDeathOffset — on cherche un maximum QUI SE DETACHE, pas un maximum.
func duelsResoudreBase(
	t *testing.T, brut []filmdec.WeaponDamage, vies map[uint32][]lifeSpan,
) (int, []duelDmg) {
	t.Helper()
	scores := make([]int, 2049)
	for b := range scores {
		for _, d := range brut {
			if d.VictimIdx < 0 || d.ResponsibleIdx < 0 {
				continue
			}
			if duelsVivant(vies, uint32(b+d.VictimIdx), d.TimestampUS) &&
				duelsVivant(vies, uint32(b+d.ResponsibleIdx), d.TimestampUS) {
				scores[b]++
			}
		}
	}
	ordre := make([]int, len(scores))
	for i := range ordre {
		ordre[i] = i
	}
	sort.Slice(ordre, func(i, j int) bool { return scores[ordre[i]] > scores[ordre[j]] })
	for i := 0; i < 5 && i < len(ordre); i++ {
		t.Logf("M0ter base candidate %d : %d couples vivants", ordre[i], scores[ordre[i]])
	}
	return ordre[0], duelsDegatsResolus(brut, vies, ordre[0])
}

// duelsDegatsResolus projette les degats sur les slots pour la base retenue. Ecarte le soin
// (Negative), le degat sur soi et tout couple dont un slot n'etait pas vivant a l'instant.
func duelsDegatsResolus(
	brut []filmdec.WeaponDamage, vies map[uint32][]lifeSpan, base int,
) []duelDmg {
	out := make([]duelDmg, 0, len(brut))
	for _, d := range brut {
		if d.VictimIdx < 0 || d.ResponsibleIdx < 0 || d.Negative {
			continue
		}
		v, a := uint32(base+d.VictimIdx), uint32(base+d.ResponsibleIdx)
		if v == a {
			continue // degat sur soi : jamais un duel
		}
		if !duelsVivant(vies, v, d.TimestampUS) || !duelsVivant(vies, a, d.TimestampUS) {
			continue
		}
		out = append(out, duelDmg{ts: d.TimestampUS, victim: v, attacker: a})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out
}
