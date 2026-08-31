package replay

// visee_etiquettes_research_test.go — LOT G : LES ETIQUETTES VIENNENT DESORMAIS DU FILM.
//
// CE QUI A CHANGE DEPUIS LE LOT F, ET POURQUOI CE LOT EXISTE.
//
// Le lot F cherchait un bit d'etat de lunette dans la charge utile des composants du bipede, en
// correlant avec la CHRONOLOGIE RELEVEE A LA MAIN par l'utilisateur (six episodes, un joueur,
// soixante secondes de film). Il a rendu un negatif adosse a une puissance mesuree — mais avec
// 36 echantillons seulement dans la classe « zoome » apres les bandes de garde. Sur le domaine
// complet (138 couples), la puissance s'effondrait a 4,75 % : un canal PARFAIT n'y aurait pas
// ete distingue du hasard, et l'instrument le declarait lui-meme NON CONCLUANT.
//
// Depuis, les evenements `unit_zoom` du film sont decodes et valides (6/6 de la chronologie
// utilisateur, controle par translation a 0,00 %, ~400 000 evenements sur le corpus, cf.
// `filmdec/zoom_events.go` et `replay/zoom_state.go`). CES EVENEMENTS SONT DES ETIQUETTES : on
// peut labelliser des instants « a la lunette » / « pas a la lunette » pour TOUS LES SLOTS et
// TOUT LE FILM, au lieu de six episodes d'un seul joueur. La meme recherche dispose donc d'une
// puissance sans commune mesure — c'est le seul facteur limitant que la verification pilote du
// lot F avait identifie.
//
// L'ENJEU EST ARCHITECTURAL, et c'est la question posee par l'utilisateur. Reconstruire l'etat
// en appariant entrees et sorties est FRAGILE : on dezoome en mourant, en subissant des degats,
// en changeant d'arme, et certaines sorties voyagent en deuxieme position d'une liste
// d'evenements, invisibles pour le scanner. Si l'etat de lunette existe AUSSI comme CHAMP dans
// les records d'etat — delta ou image-cle — il suffirait de le LIRE a chaque echantillon :
// plus de reconstruction, plus de plafond de maintien, plus de dependance a la capture
// exhaustive des sorties. C'est ce qu'un decodeur robuste ferait, et c'est ce que ce lot teste.
//
// CE QUE L'INSTRUMENT REPREND SANS LE REECRIRE (regle du dossier : deux verdicts ne se comparent
// que s'ils sortent du meme moteur) :
//
//	`vfMarche` / `vfLitBits` / `vfCompteCouverture` / `vfPublieCouverture` (lot F) : la marche
//	ANCREE des composants (`ScanBipedRecords` + `SetRecordMaskHook` + `ConsumeComponentAt`) et
//	ses compteurs de couverture ;
//	`vfTranspose` / `vfMeilleur` / `vfClassement` (lot F) et `ondeCol.evalue` / `ondeMasque`
//	(lot C) : la transposition en colonnes de bits et l'exactitude equilibree ;
//	`vfVerdict` (lot F) : l'echelle de verdict, appliquee telle quelle — donc les MEMES seuils
//	`vfSeuilCand`, `vfSeuilSuivre`, `vfSeuilP`, `vfCompEchMin`, `vfCtrlEchMin`, `vfEchMinClasse`.
//	Ils ne sont PAS redefinis ici : les redefinir aurait laisse deriver les deux lots.
//
// CE QUE L'INSTRUMENT CHANGE, ET RIEN D'AUTRE : la SOURCE DES CLASSES. La ou le lot F lisait une
// `ondeCarree` globale valable pour un joueur sur une fenetre de 60 s, ce lot lit une grille
// PAR SLOT couvrant tout le film.
//
// # G1 — LES ETIQUETTES, ET LEURS MARGES DE SECURITE
//
// L'etat vient de `buildScopedLookup` (la reconstruction de production, a plusieurs causes de
// fermeture). Deux marges obligatoires, ecrites avant la mesure :
//
//	« ZOOME »     retenu seulement a >= 300 ms APRES l'entree et >= 300 ms AVANT la fermeture.
//	              Ce que cela protege : l'instant exact d'une bascule est celui du PAQUET
//	              porteur, pas celui de la frame ou l'etat change dans le record ; et une
//	              fermeture par plafond ou par mort est datee a la seconde pres, pas mieux.
//	« PAS ZOOME » retenu seulement a >= 1 s de TOUTE periode. Asymetrique a dessein : une
//	              periode manquee (sortie non lue, entree orpheline) contaminerait la classe 0,
//	              qui est la classe majoritaire — donc celle dont la purete compte le plus.
//
// Tout le reste est EXCLU du score. La grille est echantillonnee au pas de 50 ms, et les marges
// s'appliquent par erosion (classe 1) et par dilatation (classe 0) sur cette grille : deux
// operations exactes a la resolution de la grille, pas des approximations par sondage.
//
// # LE CONTROLE PAR TRANSLATION EST CIRCULAIRE, ET C'EST UNE DECISION DECLAREE
//
// Le lot C et le lot F translataient une fenetre de 60 s a l'interieur d'un film de plusieurs
// minutes : la translation lineaire y gardait du materiau. Ici les etiquettes couvrent TOUT le
// film, donc une translation lineaire de 400 s ne laisserait presque plus de recouvrement et
// viderait la classe « zoome ». La translation est donc CIRCULAIRE sur la plage temporelle du
// film : elle preserve EXACTEMENT le nombre de periodes, leur duree, leurs marges et la
// repartition par slot — seul le contenu du flux en face change. C'est la propriete que le
// controle exige, et la seule.
//
// # SEUILS — ECRITS AVANT LA PREMIERE MESURE (repris a l'identique du lot F)
//
//	S1. CANDIDAT      : exactitude equilibree >= `vfSeuilCand` (0,95) avec >= `vfEchMinClasse`
//	                    (200) echantillons de CHAQUE classe.
//	S2. A SUIVRE      : exactitude equilibree >= `vfSeuilSuivre` (0,85), meme exigence.
//	S3. SOUS-DIMENSIONNE : moins de 200 echantillons d'une classe -> pas publiable comme
//	                    candidat, le verdict repose sur le seul controle.
//	S4. CONTROLE      : p(max GLOBAL) — part des decalages temoins dont le MEILLEUR score, tous
//	                    composants et tous offsets confondus, atteint le score observe. VERDICT
//	                    POSITIF EXIGE p(max global) < 1 % (`vfSeuilP`). p(max composant) et
//	                    p(position) sont publies pour comparaison SEULEMENT : le lot C a rattrape
//	                    un faux positif a p(position) = 0,19 % que seul p(max) a refuse.
//	S5. PUISSANCE     : part des decalages temoins dont le meilleur score atteint 1,0000. Si elle
//	                    depasse 1 %, le domaine est declare NON CONCLUANT — un negatif sans
//	                    puissance ne vaut rien (lecon des lots C et F).
//	S6. RECEVABILITE  : un composant n'entre dans la mesure que s'il est atteint sur >= 200
//	                    records ; un decalage temoin n'est retenu que s'il porte >= 30
//	                    echantillons de chaque classe.
//
// # LES DOMAINES
//
// Comme au lot F : le domaine COMPLET (le mandat) et UN DOMAINE PAR COMPOSANT. La puissance se
// paie au nombre d'hypotheses, et le lot F a mesure que 138 couples suffisaient a la detruire ;
// restreint a un composant, le meme instrument retrouvait 0,00 %. Les deux sont publies, chacun
// avec son p(max) ET sa puissance, et le lecteur voit lesquels concluent.
//
// # LE PERIMETRE DES SLOTS, ET LE CONFONDANT QU'IL RETIRE
//
// Seuls les slots portant AU MOINS UNE periode de lunette entrent dans la mesure. Sans cette
// restriction, la classe « zoome » viendrait de certains slots et la classe « pas zoome »
// surtout des autres : n'importe quel bit identifiant le porteur, son arme ou son equipe
// separerait les deux classes sans rien dire d'un etat. En n'admettant que des slots qui
// fournissent LES DEUX classes, ce confondant n'existe plus par construction.
//
// SOUS GARDE D'ENVIRONNEMENT (`ZOOMLBL_FILM`). Lecture seule ; AUCUN code de production modifie.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ZOOMLBL_FILM=<repo>/data/cache/film_chunks/00162144 \
//	  go test ./internal/analysis/replay/ -run TestViseeEtiquettes -v -timeout 90m

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	vgFilmEnv = "ZOOMLBL_FILM"
	// vgPasMS : resolution de la grille d'etiquettes. 50 ms est plus fin que l'intervalle des
	// paquets delta du film, donc la grille ne perd aucun record par agregation.
	vgPasMS = 50
	// vgMargeDedansMS / vgMargeDehorsMS : les deux marges de securite de G1.
	vgMargeDedansMS = 300
	vgMargeDehorsMS = 1000
	// vgCtrlDecalages : nombre VISE de decalages temoins ; le pas s'en deduit de la duree du
	// film, avec un plancher a vgCtrlPasMinMS pour ne pas mesurer deux fois le meme instant.
	vgCtrlDecalages = 400
	vgCtrlPasMinMS  = 250
	// vgCtrlGardeMS : voisinage du decalage nul exclu du controle (et, la translation etant
	// circulaire, voisinage de la duree totale — c'est le meme point).
	vgCtrlGardeMS = 10_000
)

// vgGrille porte les etiquettes : pour chaque slot, une suite de cellules de vgPasMS
// millisecondes valant 1 (zoome), 0 (pas zoome) ou -1 (exclu : trop pres d'une bascule).
type vgGrille struct {
	t0  int64 // instant de la premiere cellule, en ms d'horloge du film
	pas int64
	n   int
	lab map[uint32][]int8
	// slots : les slots retenus (au moins une periode de lunette), tries.
	slots []uint32
	// Diagnostics publies avant toute mesure.
	evts, entrees, sorties int
	periodes               int
	cell1, cell0, cellX    int
}

// classe rend l'etiquette du slot a l'instant tMS, l'etiquette etant translatee de delta.
//
// LA TRANSLATION EST CIRCULAIRE (cf. l'en-tete) : l'index est ramene modulo la longueur de la
// grille, ce qui preserve exactement la structure des periodes au lieu de la tronquer.
//
// LA DIVISION EST UN PLANCHER, PAS UNE TRONCATURE. Go tronque vers zero : un ecart negatif —
// ce qui arrive des qu'un decalage temoin depasse l'age du record — serait arrondi du mauvais
// cote et decalerait l'etiquette d'une cellule. Un biais d'une cellule ne changerait pas un
// verdict, mais il rendrait le controle legerement different du reel, et un controle doit etre
// exact avant d'etre commode.
func (g *vgGrille) classe(slot uint32, tMS, delta int64) int {
	l := g.lab[slot]
	if len(l) == 0 {
		return -1
	}
	d := tMS - delta - g.t0
	i := d / g.pas
	if d < 0 && d%g.pas != 0 {
		i--
	}
	n := int64(g.n)
	return int(l[(i%n+n)%n])
}

// cibles rend l'ensemble des slots retenus, sous la forme attendue par `ScanBipedRecords`.
func (g *vgGrille) cibles() map[uint32]bool {
	out := make(map[uint32]bool, len(g.slots))
	for _, s := range g.slots {
		out[s] = true
	}
	return out
}

// dureeMS rend la duree couverte par la grille.
func (g *vgGrille) dureeMS() int64 { return int64(g.n) * g.pas }

// vgBatGrille construit les etiquettes du film : evenements de lunette, vies, reconstruction de
// production, puis erosion / dilatation aux marges declarees.
//
// L'appelant detient `LockProcessDecode` : le balayage des positions est un decodage filmdec.
func vgBatGrille(dir string) (*vgGrille, error) {
	evts := filmdec.ScanFilmZoomEvents(dir)
	if len(evts) == 0 {
		return nil, fmt.Errorf("aucun evenement unit_zoom dans %s : pas d'etiquettes", dir)
	}
	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return nil, fmt.Errorf("balayage des positions : %w", err)
	}
	if len(pos) == 0 {
		return nil, fmt.Errorf("aucune position bipede : la grille n'a pas de bornes")
	}
	look := buildScopedLookup(evts, buildLifeSpans(indexBySlot(pos)), zoomHoldUS)
	g := vgBornes(pos)
	g.evts = len(evts)
	for _, e := range evts {
		if e.Scoped() {
			g.entrees++
		} else {
			g.sorties++
		}
	}
	for _, s := range vgSlotsZoomeurs(evts) {
		vgPoseSlot(g, s, look)
	}
	sort.Slice(g.slots, func(i, j int) bool { return g.slots[i] < g.slots[j] })
	return g, nil
}

// vgBornes cadre la grille sur la plage temporelle REPLIQUEE du film (premiere et derniere
// position bipede) : hors de cette plage il n'y a aucun record a etiqueter.
func vgBornes(pos []filmdec.BipedPosition) *vgGrille {
	t0, t1 := int64(pos[0].TimestampUS/1000), int64(pos[0].TimestampUS/1000)
	for _, p := range pos {
		t := int64(p.TimestampUS / 1000)
		if t < t0 {
			t0 = t
		}
		if t > t1 {
			t1 = t
		}
	}
	g := &vgGrille{t0: t0, pas: vgPasMS, lab: map[uint32][]int8{}}
	g.n = int((t1-t0)/vgPasMS) + 1
	return g
}

// vgSlotsZoomeurs rend les slots qui portent au moins une ENTREE en lunette.
func vgSlotsZoomeurs(evts []filmdec.ZoomEvent) []uint32 {
	vus := map[uint32]bool{}
	var out []uint32
	for _, e := range evts {
		if !e.Scoped() || vus[e.Slot] {
			continue
		}
		vus[e.Slot] = true
		out = append(out, e.Slot)
	}
	return out
}

// vgPoseSlot etiquette un slot et le retient s'il fournit les DEUX classes. Un slot qui ne
// donnerait qu'une classe n'apporterait rien a la mesure et gonflerait le denominateur.
func vgPoseSlot(g *vgGrille, slot uint32, look func(uint32, uint64) int) {
	brut := make([]bool, g.n)
	for i := range brut {
		brut[i] = look(slot, uint64(g.t0+int64(i)*g.pas)*1000) > 0
	}
	lab, n1, n0, nx := vgEtiquette(brut)
	if n1 == 0 || n0 == 0 {
		return
	}
	g.lab[slot] = lab
	g.slots = append(g.slots, slot)
	g.periodes += vgComptePeriodes(brut)
	g.cell1 += n1
	g.cell0 += n0
	g.cellX += nx
}

// vgComptePeriodes compte les blocs contigus de cellules zoomees d'un slot.
func vgComptePeriodes(brut []bool) int {
	n := 0
	for i, z := range brut {
		if z && (i == 0 || !brut[i-1]) {
			n++
		}
	}
	return n
}

// vgEtiquette applique les deux marges : erosion pour la classe 1, dilatation pour la classe 0.
//
// EXACT A LA RESOLUTION DE LA GRILLE, par sommes prefixees : une cellule est « zoome » si les
// vgMargeDedansMS/vgPasMS cellules de part et d'autre le sont TOUTES ; elle est « pas zoome » si
// aucune cellule ne l'est dans les vgMargeDehorsMS/vgPasMS de part et d'autre. Aux deux bords de
// la grille, la classe 1 est refusee (la marge ne peut pas etre verifiee) tandis que la classe 0
// accepte le hors-grille comme non zoome — c'est le cote prudent dans les deux cas.
func vgEtiquette(brut []bool) (lab []int8, n1, n0, nx int) {
	n := len(brut)
	cum := make([]int, n+1)
	for i, z := range brut {
		cum[i+1] = cum[i]
		if z {
			cum[i+1]++
		}
	}
	dd, dh := vgMargeDedansMS/vgPasMS, vgMargeDehorsMS/vgPasMS
	lab = make([]int8, n)
	for i := range lab {
		lab[i] = -1
		if i-dd >= 0 && i+dd < n && cum[i+dd+1]-cum[i-dd] == 2*dd+1 {
			lab[i], n1 = 1, n1+1
			continue
		}
		a, b := i-dh, i+dh+1
		if a < 0 {
			a = 0
		}
		if b > n {
			b = n
		}
		if cum[b]-cum[a] == 0 {
			lab[i], n0 = 0, n0+1
			continue
		}
		nx++
	}
	return lab, n1, n0, nx
}
