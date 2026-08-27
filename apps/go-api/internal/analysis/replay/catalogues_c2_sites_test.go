package replay

// catalogues_c2_sites_test.go — LOT C (catalogues), C2.3 : LE CONTROLE SPATIAL DES SITES
// CANDIDATS D'ASSAUT, joue AVANT toute entree au catalogue.
//
// CE QUI EST CONTROLE. Les cartes du corpus Assaut ne portent AUCUN objet au role
// historique `assault_bomb` (C2.1, dumps du 2026-08-27) ; le motif candidat est porte par
// deux hashs de label NON RESOLUS (chasse murmur3 a 2173 candidats sans nom, patron KOTH) :
// -1537427652 = position CENTRALE neutre, -1843278509 = positions de BASE par equipe —
// motif stable sur les 5 cartes. Ces positions sont FIGEES dans
// `registre_film/C2_sites_candidats.json` AVANT cette mesure. Le gate du plan (ecrit avant
// mesure, ne bouge pas) : chaque EXPLOSION datee par le score de mode doit avoir eu de
// l'activite de joueurs a proximite d'un site candidat dans les secondes qui precedent —
// accord >= 75 % des explosions a <= 10 m d'un site, temoin (sites decales de 12 m)
// <= 25 %. Si le controle rate : RIEN n'entre au catalogue.
//
// PARAMETRES OPERATIONNELS — AMENDES UNE FOIS, ET VOICI POURQUOI. La V1 (fenetre
// [t-5000 ms, t], « au moins UNE position de n'importe quel joueur a <= 10 m ») a ete jouee
// sur les 8 films le 2026-08-27 : signal 100 % partout (dmin mediane 0,91-3,54 m) mais
// TEMOIN a 75-100 % — le temoin a fait son travail : avec un rayon de 10 m contre un
// decalage de 12 m, les deux disques se recouvrent, et une fenetre de 5 s x 8-16 joueurs
// couvre toute l'arene. L'instrument V1 ne discriminait RIEN (la V1 reste au log, au-dessus
// de la re-mesure). Les chiffres du PLAN (75/25, 10 m, 12 m) ne bougent pas ; ce qui est
// amende est la definition d'« activite », que le plan ne chiffre pas : l'activite devient
// la PRESENCE SOUTENUE D'UN MEME JOUEUR AU MEME SITE — la signature du POSEUR qui arme —
// soit, pour un couple (slot, site) : etendue temporelle >= 2000 ms et >= 3 echantillons a
// <= 10 m du site, dans la fenetre [t-15000 ms, t] (l'armement + le fusible tiennent
// dedans). Temoin STRICTEMENT identique aux sites decales de +12 m sur X. UNE re-mesure —
// pas de deuxieme amendement : si le temoin sature encore, le gate est RATE et rien n'entre.
// Distance = 2D horizontale (x, y), la meme que la primitive de proximite des instruments
// D6/D8.
//
// Les explosions se datent par LA MEME regle que le gate A1 (`a1ClassesTemporelles` :
// montee du comp 0 A d'un slot d'equipe au statborg — releve A0.3, score film = score API
// 9/9). Les positions de bipede sortent de la chaine d6 (bornes + layout du catalogue).
//
// REGIME : gardes `ATT_FILM` + `C2_FILM` + `C2_SITES`, UN FILM PAR PROCESSUS, lecture
// seule, AUCUNE base.
//
//	$env:ATT_FILM="<repo>/data/cache"; $env:ATT_REF="<worktree>/data/titles/halo_infinite/reference"
//	$env:C2_FILM="35b75a31"; $env:C2_SITES="<worktree>/.ai/V7.5/replay2d/registre_film/C2_sites_candidats.json"
//	go test ./internal/analysis/replay/ -run CataloguesC2Sites -v

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/filmproc"
)

const (
	// c2FenetreMS / c2RayonM / c2TemoinDecalageM : parametres du controle, en-tete de fichier
	// (fenetre amendee 5000 -> 15000 ms avec la definition de presence soutenue, cf. en-tete).
	c2FenetreMS       = 15000
	c2RayonM          = 10.0
	c2TemoinDecalageM = 12.0
	// c2PresenceMinMS / c2PresenceMinEch : la presence SOUTENUE d'un meme (slot, site) —
	// etendue temporelle et compte minimal d'echantillons a <= c2RayonM (en-tete de fichier).
	c2PresenceMinMS  = 2000
	c2PresenceMinEch = 3
	// c2SeuilSignal / c2SeuilTemoin : le gate C2.3 du plan, recopie sans modification.
	c2SeuilSignal = 0.75
	c2SeuilTemoin = 0.25
)

// c2Site est un site candidat fige (positions du JSON du registre).
type c2Site struct {
	Kind string  `json:"kind"`
	Hash int32   `json:"hash"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Z    float64 `json:"z"`
	Team int     `json:"team"`
}

// c2Candidats est le fichier fige du registre.
type c2Candidats struct {
	Maps map[string]struct {
		Name  string   `json:"name"`
		Sites []c2Site `json:"sites"`
	} `json:"maps"`
}

func TestCataloguesC2SitesControle(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv("C2_FILM")
	if id == "" {
		t.Skipf("mesure non demandee : C2_FILM vide")
	}
	sitesPath := os.Getenv("C2_SITES")
	if sitesPath == "" {
		t.Skipf("mesure non demandee : C2_SITES vide (sites candidats figes)")
	}
	g := filmproc.Arm("c2-sites", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio — ce film ne compte NI POUR NI CONTRE",
			id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	sites := c2ChargeSites(t, sitesPath, id)
	src, ok := objOpenFilm(t, root, id)
	if !ok {
		t.Fatalf("%s : film absent du cache", id)
	}
	_, explosions := a1ClassesTemporelles(t, id, src)
	if len(explosions) == 0 {
		t.Logf("%s : AUCUNE explosion datee — ce film ne controle rien, et cela se dit", id)
		return
	}
	clockUS, err := ScanFilmClockOrigin(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : origine d'horloge illisible : %v", id, err)
	}
	wr, lay, ok := d6Bornes(t, root, id)
	if !ok {
		t.Fatalf("%s : bornes de quantification indisponibles", id)
	}
	for _, s := range sites {
		dedans := s.X >= float64(wr[0].Min) && s.X <= float64(wr[0].Max) &&
			s.Y >= float64(wr[1].Min) && s.Y <= float64(wr[1].Max) &&
			s.Z >= float64(wr[2].Min) && s.Z <= float64(wr[2].Max)
		t.Logf("%s : site %s (hash %d, team %d) @ %.2f %.2f %.2f — dans les bornes : %v",
			id, s.Kind, s.Hash, s.Team, s.X, s.Y, s.Z, dedans)
		if !dedans {
			t.Errorf("%s : site %s HORS des bornes de la carte", id, s.Kind)
		}
	}
	pos, err := d6Positions(objChunkDir(root, id), wr, lay)
	if err != nil {
		t.Fatalf("%s : positions de bipede illisibles : %v", id, err)
	}
	c2Mesure(t, id, sites, explosions, clockUS, pos)
}

// c2ChargeSites lit les sites candidats figes de la carte du film (via attCartes).
func c2ChargeSites(t *testing.T, path, id string) []c2Site {
	t.Helper()
	c, ok := attCartes[id]
	if !ok {
		t.Fatalf("%s : film inconnu du fixture attCartes", id)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin d'instrument, lecture seule
	if err != nil {
		t.Fatalf("sites candidats illisibles (%s) : %v", path, err)
	}
	var cand c2Candidats
	if err := json.Unmarshal(raw, &cand); err != nil {
		t.Fatalf("sites candidats invalides (%s) : %v", path, err)
	}
	e, ok := cand.Maps[c.MapID]
	if !ok || len(e.Sites) == 0 {
		t.Fatalf("%s : aucun site candidat fige pour la carte %s (%s)", id, c.Nom, c.MapID)
	}
	t.Logf("%s : carte %s (%s) — %d site(s) candidat(s) figes", id, c.Nom, c.MapID, len(e.Sites))
	return e.Sites
}

// c2Mesure confronte chaque explosion aux sites (signal) et aux sites decales (temoin) —
// avec la definition de PRESENCE SOUTENUE de l'en-tete (V2, amendee une fois).
func c2Mesure(t *testing.T, id string, sites []c2Site, explosions []int64, clockUS uint64, pos []filmdec.BipedPosition) {
	t.Helper()
	var okSignal, okTemoin int
	for _, x := range explosions {
		sig := c2PresenceSoutenue(pos, sites, clockUS, x, 0)
		tem := c2PresenceSoutenue(pos, sites, clockUS, x, c2TemoinDecalageM)
		if sig {
			okSignal++
		}
		if tem {
			okTemoin++
		}
		t.Logf("%s : explosion t=%d ms — presence soutenue au site : signal %v, temoin %v", id, x, sig, tem)
	}
	n := len(explosions)
	t.Logf("C2.3 %s : %d explosion(s) — signal %d/%d = %.1f %% (seuil >= %.0f %%), temoin %d/%d = %.1f %% "+
		"(seuil <= %.0f %%)",
		id, n, okSignal, n, 100*float64(okSignal)/float64(n), 100*c2SeuilSignal,
		okTemoin, n, 100*float64(okTemoin)/float64(n), 100*c2SeuilTemoin)
}

// c2PresenceSoutenue dit si, dans la fenetre [x-c2FenetreMS, x], UN MEME slot a tenu UN MEME
// site (decale de dx) : >= c2PresenceMinEch echantillons a <= c2RayonM, etendue temporelle
// >= c2PresenceMinMS.
func c2PresenceSoutenue(pos []filmdec.BipedPosition, sites []c2Site, clockUS uint64, x int64, dx float64) bool {
	type cle struct {
		slot uint32
		site int
	}
	type etendue struct {
		min, max int64
		n        int
	}
	acc := map[cle]*etendue{}
	for _, p := range pos {
		if p.TimestampUS < clockUS {
			continue
		}
		pMS := int64(p.TimestampUS-clockUS) / 1000
		if pMS < x-c2FenetreMS || pMS > x {
			continue
		}
		for i, s := range sites {
			if math.Hypot(float64(p.X)-(s.X+dx), float64(p.Y)-s.Y) > c2RayonM {
				continue
			}
			k := cle{p.Slot, i}
			e := acc[k]
			if e == nil {
				e = &etendue{min: pMS, max: pMS}
				acc[k] = e
			}
			if pMS < e.min {
				e.min = pMS
			}
			if pMS > e.max {
				e.max = pMS
			}
			e.n++
		}
	}
	for _, e := range acc {
		if e.n >= c2PresenceMinEch && e.max-e.min >= c2PresenceMinMS {
			return true
		}
	}
	return false
}
