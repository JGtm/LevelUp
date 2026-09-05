package replay

// vehicules_v0_cadrage_test.go — INSTRUMENT JETABLE du lot V0 (cadrage vehicules, 2026-08-31).
//
// CE QU'IL MESURE, ET POURQUOI IL EXISTE. Le cadrage doit dire, SUR PIECES, ce que le decodeur
// actuel sait deja des vehicules (`ti=40`) : combien de VIES d'entite le recensement des
// images-cles rend par film, sur quelle duree elles sont bornees, et si le chemin DELTA des
// objets du monde rend des trajectoires ou du bruit. Aucun de ces chiffres n'existe dans le
// depot : la sonde du 18/08 (`attachement_phase0_vehicules_test.go`) ne tournait que sur deux
// films BTB, jamais sur le corpus Behemoth / Launch Site vise par le chantier.
//
// IL N'AJOUTE AUCUN DECODEUR. Il n'appelle que des fonctions deja exportees et deja validees
// (`ScanFilmWorldObjectKeyframes`, `ScanFilmWorldObjects`) et reutilise les helpers de l'item
// 0.3 (`attEtalement`, `attBandesKeyframe`, `attBandeFantome`). Rien ici n'est destine a rester :
// c'est un instrument de reconnaissance, a supprimer a la cloture du lot V0.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 ATT_FILM=<depot>/data/cache \
//	  V0_FILMS=8a049c50:behemoth,fccc61cd:launch site \
//	  go test ./internal/analysis/replay/ -run TestV0Cadrage -v -timeout 60m
//
// V0_DELTA=1 ajoute le balayage DELTA (lourd : tout le film, bit a bit) et son temoin fantome.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	// v0FilmsEnv porte le corpus : « short8:nom de carte » separes par des virgules. Le nom de
	// carte est celui du catalogue de bornes (`map_quant_bounds.json`), en minuscules.
	v0FilmsEnv = "V0_FILMS"
	// v0DeltaEnv arme le balayage DELTA, qui lit tout le film bit a bit.
	v0DeltaEnv = "V0_DELTA"
)

// v0Film est une entree du corpus.
type v0Film struct{ ID, Carte string }

// v0Corpus lit le corpus de l'environnement.
func v0Corpus(t *testing.T) []v0Film {
	t.Helper()
	v := os.Getenv(v0FilmsEnv)
	if v == "" {
		t.Skipf("mesure non demandee : %s vide (« short8:carte, ... »)", v0FilmsEnv)
	}
	var out []v0Film
	for _, s := range strings.Split(v, ",") {
		id, carte, ok := strings.Cut(strings.TrimSpace(s), ":")
		if !ok {
			t.Fatalf("entree de corpus invalide %q : forme attendue « short8:carte »", s)
		}
		out = append(out, v0Film{ID: strings.TrimSpace(id), Carte: strings.TrimSpace(carte)})
	}
	return out
}

// v0Bornes rend les bornes monde d'une carte NOMMEE et installe ses largeurs d'axe pour le
// chemin objet du monde. Il double `attBornes` parce que celui-ci passe par un fixture
// film -> carte (`attCartes`) que le corpus de ce lot ne peuple pas : ici la carte est donnee.
//
// L'APPELANT DOIT DETENIR LockProcessDecode ET RESTAURER WorldObjectPrecision.
func v0Bornes(t *testing.T, root, carte string) (filmdec.Vec3Range, bool) {
	t.Helper()
	cat, err := filmdec.LoadMapQuantCatalog(filepath.Join(attRefDir(root), "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Logf("carte %q absente du catalogue de bornes (%v)", carte, err)
		return filmdec.Vec3Range{}, false
	}
	filmdec.SetWorldObjectPrecisionFromLayout(e.Layout())
	return e.Range(), true
}

// v0BalayageEnv arme le balayage du CACHE ENTIER, et v0SortieEnv dit ou ecrire son releve.
const (
	v0BalayageEnv = "V0_BALAYAGE"
	v0SortieEnv   = "V0_SORTIE"
)

// TestV0CadrageBalayageCache — COMBIEN DE FILMS DU CACHE PORTENT DES VEHICULES.
//
// La question du corpus ne se repond pas par le nom d'une carte : une carte a vehicules jouee
// en mode Tactical n'en porte aucun (mesure du recensement, `f0680b37`). Seule la presence de
// slots `ti=40` dans le flux tranche. Le balayage ne lit que les trois premiers chunks de
// chaque film : c'est un test de PRESENCE, pas un comptage.
//
// Le releve part dans un CSV (V0_SORTIE) pour etre croise avec le registre des matchs en SQL —
// la jointure carte/mode/date n'a rien a faire dans un test Go.
func TestV0CadrageBalayageCache(t *testing.T) {
	if os.Getenv(v0BalayageEnv) == "" {
		t.Skipf("balayage du cache non demande : %s vide", v0BalayageEnv)
	}
	root := attRequireRoot(t)
	base := filepath.Join(root, "film_chunks")
	ents, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("cache film illisible (%s) : %v", base, err)
	}
	var lignes []string
	lignes = append(lignes, "short8,slots40,slots_bipede,archetypes")
	avec, total := 0, 0
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		total++
		cens, _ := attCensusTI(filepath.Join(base, e.Name()), 3)
		n := len(cens[int(attVehiculeTI)])
		if n > 0 {
			avec++
		}
		lignes = append(lignes, fmtLigne(e.Name(), n, len(cens[objBipedTI]), len(cens)))
	}
	t.Logf("V0 BALAYAGE — %d films du cache, %d portent au moins un slot ti=%d dans leurs trois "+
		"premiers chunks (%.1f %%)", total, avec, attVehiculeTI, 100*attPart(avec, total))
	sortie := os.Getenv(v0SortieEnv)
	if sortie == "" {
		return
	}
	if err := os.WriteFile(sortie, []byte(strings.Join(lignes, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("ecriture du releve %s : %v", sortie, err)
	}
	t.Logf("V0 BALAYAGE — releve ecrit : %s (%d lignes)", sortie, len(lignes)-1)
}

// fmtLigne formate une ligne du releve.
func fmtLigne(id string, slots40, bipedes, tis int) string {
	return fmt.Sprintf("%s,%d,%d,%d", id, slots40, bipedes, tis)
}

// TestV0CadrageRecensement — LE RECENSEMENT DES VIES DE VEHICULE aux images-cles.
//
// C'est la mesure la moins chere et la plus informative : elle ne decode aucun composant, elle
// compte les records `ti=40` par vie (slot, generation) et les instants ou ils sont recenses.
// Une vie recensee du debut a la fin est un vehicule permanent ; une vie qui apparait puis
// disparait BORNE une naissance et une destruction — c'est la matiere du lot V2.
func TestV0CadrageRecensement(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		dir := objChunkDir(root, f.ID)
		if filmdec.CountFilmChunks(dir) == 0 {
			t.Logf("%s : film absent du cache — saute", f.ID)
			continue
		}
		k := filmdec.ScanFilmWorldObjectKeyframes(dir, int(attVehiculeTI))
		var complets, partiels, uniques int
		var duree []uint64
		for _, vus := range k.SeenUS {
			switch {
			case len(vus) == 1:
				uniques++
			case len(vus) == len(k.TimesUS):
				complets++
			default:
				partiels++
			}
			if len(vus) > 1 {
				duree = append(duree, (vus[len(vus)-1]-vus[0])/1_000_000)
			}
		}
		sort.Slice(duree, func(i, j int) bool { return duree[i] < duree[j] })
		med := uint64(0)
		if len(duree) > 0 {
			med = duree[len(duree)/2]
		}
		t.Logf("V0 %s (%s) — %d images-cles · bande %d slots · %d vies ti=%d recensees "+
			"(%d sur toute la duree, %d partielles, %d vues une seule fois) · duree mediane des "+
			"vies vues au moins deux fois : %d s",
			f.ID, f.Carte, len(k.TimesUS), len(k.Band), len(k.SeenUS), attVehiculeTI,
			complets, partiels, uniques, med)
	}
}

// TestV0CadrageNuageDelta — LE CHEMIN DELTA sur `ti=40`, avec son temoin fantome.
//
// POURQUOI LE TEMOIN EST OBLIGATOIRE. Le meme decodeur a deja ete REFUTE sur les armes au sol
// (`ti=42`, 2026-08-12) : un record delta ne dit pas son archetype, la bande comblee est
// contaminee, et 62,4 % des slots s'etalaient au-dela de 20 u. Publier un nombre de « vies de
// vehicule » sans la bande fantome de meme cardinalite serait refaire l'erreur en changeant de
// numero.
func TestV0CadrageNuageDelta(t *testing.T) {
	if os.Getenv(v0DeltaEnv) == "" {
		t.Skipf("balayage delta non demande : %s vide (lecture bit a bit de tout le film)", v0DeltaEnv)
	}
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v0NuageDeltaFilm(t, root, f)
	}
}

// v0NuageDeltaFilm mesure UN film sur le chemin delta.
func v0NuageDeltaFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	veh, err := filmdec.ScanFilmWorldObjects(dir, &wr, int(attVehiculeTI))
	if err != nil {
		t.Logf("V0 %s (%s) — balayage ti=%d : %v", f.ID, f.Carte, attVehiculeTI, err)
		return
	}
	serre, large, comptes := attEtalement(veh)
	t.Logf("V0 %s (%s) — DELTA : %d vies decodees, %d a >= %d echantillons dont %d tiennent "+
		"dans %.1f u (%.1f %%) et %d s'etalent au-dela de %.0f u (%.1f %%)",
		f.ID, f.Carte, len(veh), comptes, attMinEchantillons, serre, attEtalementSerre,
		100*attPart(serre, comptes), large, attEtalementLarge, 100*attPart(large, comptes))
	vus, autres := attBandesKeyframe(dir)
	fantome := attBandeFantome(vus, autres)
	if len(fantome) == 0 {
		t.Logf("V0 %s : aucun slot libre pour une bande fantome", f.ID)
		return
	}
	fveh, err := filmdec.ScanFilmWorldObjectsForBand(dir, &wr, fantome)
	if err != nil {
		t.Logf("V0 %s : bande fantome : %v", f.ID, err)
		return
	}
	fs, fl, fc := attEtalement(fveh)
	t.Logf("V0 %s (%s) — TEMOIN FANTOME (%d slots jamais vus porter ti=%d) : %d vies, %d a >= %d "+
		"echantillons dont %d serrees (%.1f %%) et %d etalees (%.1f %%)",
		f.ID, f.Carte, len(fantome), attVehiculeTI, len(fveh), fc, attMinEchantillons, fs,
		100*attPart(fs, fc), fl, 100*attPart(fl, fc))
}

// v0PlausibleMPS — LE CRITERE, ECRIT AVANT LA MESURE. Le vehicule le plus rapide de Halo
// Infinite reste largement sous 35 m/s (le depot retient deja 35 comme ordre de grandeur dans
// le commentaire de `DefaultMaxSpeedMPS`). Un pas de trajectoire au-dela de ce seuil n'est pas
// un deplacement : c'est une lecture desalignee.
const v0PlausibleMPS = 35.0

// v0PasMaxUS borne l'ecart temporel d'une paire d'echantillons consideree. Deux paquets delta
// valent ~1 s (ecart median mesure 0,5 s a l'item 0.3) : au-dela de 2 s, l'entite a pu faire
// tout autre chose entre les deux points et la vitesse ne mesure plus la continuite.
const v0PasMaxUS = uint64(2_000_000)

// TestV0CadrageGrammaireI0 — LA QUESTION QUI COMMANDE TOUT LE LOT V1.
//
// CE QUE LE REGISTRE DIT, ET QUE PERSONNE N'AVAIT LU. `ti=40` porte a i0
// `object-position-dynamic-precision-component` — la grammaire du BIPEDE (porte de 5 bits) —
// et NON `object-position-component`, la grammaire des objets du monde (porte de 3 bits) que
// portent `ti=36/37/38/39/41/42/43`. `ScanFilmWorldObjects`, seule voie employee jusqu'ici sur
// `ti=40` (sonde du 18/08 comprise), decode donc les vehicules avec la grammaire d'un autre
// archetype.
//
// LE DEPARTAGE EST LA CONTINUITE, ET LE SEUIL EST ECRIT AVANT LA MESURE. Toute lecture de
// 17 bits redonne une coordonnee DANS l'emprise de la carte : « etre dans la carte » ne
// discrimine rien. Ce qui discrimine, c'est qu'une trajectoire reelle est CONTINUE. On compare
// donc, sur la MEME bande de slots et le MEME film, la part des pas consecutifs qui restent
// sous `v0PlausibleMPS` — grammaire objet du monde contre grammaire bipede.
func TestV0CadrageGrammaireI0(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v0GrammaireUnFilm(t, root, f)
	}
}

// v0GrammaireUnFilm confronte les deux grammaires d'i0 sur UN film.
func v0GrammaireUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := filmdec.ScanFilmWorldObjectKeyframes(dir, int(attVehiculeTI)).Band
	if len(bande) == 0 {
		t.Logf("V0 %s (%s) — aucun slot ti=%d : rien a comparer", f.ID, f.Carte, attVehiculeTI)
		return
	}

	// (a) GRAMMAIRE OBJET DU MONDE — celle employee jusqu'ici sur ti=40.
	veh, err := filmdec.ScanFilmWorldObjectsForBand(dir, &wr, bande)
	if err != nil {
		t.Logf("V0 %s : grammaire objet du monde : %v", f.ID, err)
		return
	}
	nWO, pWO := v0ContinuiteTracks(veh)

	// (b) GRAMMAIRE BIPEDE (dynamic-precision) — celle que le registre attribue a ti=40.
	// RequireTag1 est DESARME : le tag de 2 bits est la generation du handle, et les objets du
	// monde en emploient les quatre (regle etablie par matchWorldObjectRecord).
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange, opt.RequireTag1 = &wr, false
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		t.Logf("V0 %s : decoupage i0 illisible : %v", f.ID, err)
		return
	}
	opt.Layout = &lay
	nBP, pBP := v0ContinuitePositions(v0ScanBipedeSurBande(dir, bande, lay, opt))

	// (b bis) ORIENTATION ET VITALITE, DANS LE MEME RECORD. `CaptureDirs` poursuit le curseur
	// apres i0 sur i1/i2/i3/i4/i5 — exactement les indices que le masque d'un vehicule porte.
	// Ce que la mesure dit ici, c'est si le curseur ARRIVE : une part elevee de directions lues
	// signifie que la grammaire d'i1/i2 tient aussi sur `ti=40`, une part nulle qu'elle ne tient
	// pas. Elle NE dit PAS que les valeurs sont justes — le cap d'un vehicule se confronte a sa
	// direction de deplacement, et c'est un gate du lot V1, pas de ce cadrage.
	optDirs := opt
	optDirs.CaptureDirs = true
	dirs := v0ScanBipedeSurBande(dir, bande, lay, optDirs)
	nAim, nVel, nBody := 0, 0, 0
	for _, p := range dirs {
		if p.HasAim {
			nAim++
		}
		if p.HasVel {
			nVel++
		}
		if p.HasBody {
			nBody++
		}
	}
	t.Logf("V0 %s (%s) — MEME RECORD, composants suivants : %d echantillons, %d portent i2 "+
		"(cap, %.1f %%), %d portent i1 (velocite, %.1f %%), %d portent i4 (vitalite, %.1f %%)",
		f.ID, f.Carte, len(dirs), nAim, 100*attPart(nAim, len(dirs)), nVel,
		100*attPart(nVel, len(dirs)), nBody, 100*attPart(nBody, len(dirs)))

	// (c) TEMOIN FANTOME sous la grammaire GAGNANTE. Sans lui, « 99 % de pas plausibles » ne se
	// distingue pas d'un critere qui accepte tout : il faut montrer que la MEME lecture sur des
	// slots qui ne portent AUCUN archetype ne le satisfait pas.
	vus, autres := attBandesKeyframe(dir)
	nFA, pFA := 0, 0.0
	if fantome := attBandeFantome(vus, autres); len(fantome) > 0 {
		nFA, pFA = v0ContinuitePositions(v0ScanBipedeSurBande(dir, fantome, lay, opt))
	}

	t.Logf("V0 %s (%s) — GRAMMAIRE i0 sur la MEME bande de %d slots ti=%d :\n"+
		"    objet du monde (porte 3 bits)  : %d pas mesures, %.1f %% sous %.0f m/s\n"+
		"    bipede dyn.-precision (5 bits) : %d pas mesures, %.1f %% sous %.0f m/s\n"+
		"    TEMOIN FANTOME, grammaire bipede : %d pas mesures, %.1f %% sous %.0f m/s",
		f.ID, f.Carte, len(bande), attVehiculeTI,
		nWO, 100*pWO, v0PlausibleMPS, nBP, 100*pBP, v0PlausibleMPS, nFA, 100*pFA, v0PlausibleMPS)
}

// v0ScanBipedeSurBande deroule le balayage BIPEDE (`ScanBipedRecords`, deja exporte et deja
// valide au quantum exact) sur une bande de slots QUELCONQUE. Il n'existe pas d'entree de
// paquet pour cela : `ScanFilmBipedPositions` releve lui-meme la bande `ti=35`. C'est
// exactement le morceau manquant que le lot V1 aura a exposer proprement ; ici il tient en une
// boucle de lecture de chunks, sans toucher au decodeur.
func v0ScanBipedeSurBande(dir string, bande map[uint32]bool, lay filmdec.I0Layout,
	opt filmdec.ScanFilmOptions) []filmdec.BipedPosition {
	var out []filmdec.BipedPosition
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta {
				continue
			}
			for _, r := range filmdec.ScanBipedRecords(pk.Payload(data), filmdec.NewSlotBand(bande), lay, opt) {
				r.Chunk, r.PacketIndex, r.TimestampUS = c, pk.Index, pk.TimestampUS
				out = append(out, r)
			}
		}
	}
	return out
}

// v0ContinuiteTracks mesure la continuite des trajectoires rendues par le chemin objet du monde.
func v0ContinuiteTracks(tracks []filmdec.ProjectileTrack) (int, float64) {
	pas, bons := 0, 0
	for _, tr := range tracks {
		for i := 1; i < len(tr.Pts); i++ {
			a, b := tr.Pts[i-1], tr.Pts[i]
			if !v0PasCompte(a.TimestampUS, b.TimestampUS) {
				continue
			}
			pas++
			if v0Vitesse([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z},
				b.TimestampUS-a.TimestampUS) <= v0PlausibleMPS {
				bons++
			}
		}
	}
	return pas, attPart(bons, pas)
}

// v0ContinuitePositions mesure la continuite des positions rendues par le chemin bipede.
func v0ContinuitePositions(pos []filmdec.BipedPosition) (int, float64) {
	parSlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		if p.HasWorld {
			parSlot[p.Slot] = append(parSlot[p.Slot], p)
		}
	}
	pas, bons := 0, 0
	for _, ech := range parSlot {
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		for i := 1; i < len(ech); i++ {
			a, b := ech[i-1], ech[i]
			if !v0PasCompte(a.TimestampUS, b.TimestampUS) {
				continue
			}
			pas++
			if v0Vitesse([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z},
				b.TimestampUS-a.TimestampUS) <= v0PlausibleMPS {
				bons++
			}
		}
	}
	return pas, attPart(bons, pas)
}

// v0PasCompte dit si une paire d'instants forme un pas mesurable.
func v0PasCompte(a, b uint64) bool { return b > a && b-a <= v0PasMaxUS }

// v0Vitesse rend la vitesse en m/s entre deux points separes de dtUS microsecondes.
//
// LA DISTANCE PASSE PAR `dist3`, l'UNIQUE ecriture de la formule euclidienne du paquet
// (`geometry.go`, garde-rail `TestUneSeuleFormuleDeDistance3D`). La premiere redaction de cet
// instrument la recopiait a six flottants — exactement la signature que le correctif de revue
// du 2026-08-17 avait supprimee, et le garde-rail l'a rattrapee.
func v0Vitesse(a, b [3]float32, dtUS uint64) float64 {
	return dist3(a, b) / (float64(dtUS) / 1_000_000)
}
