package grammar

// distribuer.go — UNE MARCHE, N CANAUX (ADR 0037 ; decision DT2-1 du plan de l etape 2).
//
// [Distribuer] marche UNE fois les phases du film que ses canaux lisent — les images-cles
// ([CanalDImageCle]), les trames delta ([CanalDesTrames]), ou leurs seules tetes ([CanalDesTetes])
// — et donne chaque paquet range a chaque canal, dans l ordre des canaux. Un canal ([Canal])
// declare ses interets ; un canal des trames pose en plus ses crochets sur l observation de la
// marche des trames : l interpretation reste PENDANT la marche (ADR 0037 IR-8), la ou elle a lieu
// aujourd hui — la phase des images-cles se lit sous l observation du contexte, comme
// [FilmContext.ImagesCles]. Un canal ne lit jamais un octet ; il lit la structure
// ([lecture.Paquet]), la marche d ancres du paquet ([MarcheDistribuee.Ancres]) et ce que ses crochets
// recoivent.
//
// # LES INTERETS SONT DES PAIRES (ARCHETYPE, COMPOSANT), DANS UNE PHASE
//
// Resolues dans le registre du film, jamais un nom de composant seul : deux tables de composant
// homonymes existent (« high-frequency », ti=3 et ti=4, deux grammaires — regle « routage par
// archetype ou par table de composant » de la campagne de grammaire). Une occurrence est
// INTERPRETEE quand son archetype et son index sont dans l union des interets de SA phase et qu elle
// est traversee (decision DT2-2) ; la structure la marque [lecture.EtatInterprete]. Dans la phase
// des images-cles, un archetype interprete est aussi un CORPS A PARCOURIR : les records des autres
// archetypes gardent leur identite et leur ancre, sans composant ([lecture.CorpsNonParcouru],
// decision 3 du lot 2.2).
//
// # DEUX CANAUX NE POSENT PAS LE MEME CROCHET
//
// Chaque canal des trames pose les siens sur une observation a lui, que le distributeur fond dans
// celle de la marche ([fondreLesCrochets]) : un crochet pose par deux canaux est une erreur, parce
// que le second ecraserait le premier sans que rien ne le dise.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// Phase est la phase de la marche ou un canal interprete un composant.
type Phase uint8

// Les phases d une marche.
const (
	// PhaseTrames : les trames delta. Le canal y interprete l occurrence par ses crochets.
	PhaseTrames Phase = iota
	// PhaseImagesCles : les images-cles. Le canal y lit l occurrence dans la structure du paquet ;
	// la phase parcourt l etat complet des records de l archetype.
	PhaseImagesCles
	// nombreDePhases borne l index des interets resolus.
	nombreDePhases
)

// Interet est une paire (archetype, composant) qu un canal interprete dans une phase de la marche
// (celle des trames par defaut) : l index de type de l archetype dans le registre du film, et le nom
// du composant tel que le registre l ecrit (un canal qui connait deux orthographes declare les deux).
type Interet struct {
	Phase     Phase
	TI        int
	Composant string
}

// Canal est un consommateur de la marche du film : il declare ses interets et recoit le bilan. Ce
// qu il lit dit quelles phases la marche joue : un canal est aussi un [CanalDImageCle], un
// [CanalDesTrames] ou un [CanalDesTetes], ou plusieurs d entre eux.
type Canal interface {
	// Interets rend ce que le canal interprete.
	Interets() []Interet
	// Clore recoit le bilan de la marche, une fois ses phases finies.
	Clore(b BilanDeMarche)
}

// CanalDImageCle est un canal qui lit la phase des images-cles.
type CanalDImageCle interface {
	Canal
	// ImageCle recoit chaque paquet d image-cle, dans l ordre du flux ; `m` expose la marche
	// d ancres du paquet ([MarcheDistribuee.Ancres]).
	ImageCle(p *lecture.Paquet, m *MarcheDistribuee)
}

// CanalDesTrames est un canal qui lit la phase des trames. Une distribution dont aucun canal ne lit
// les trames ne les marche pas.
type CanalDesTrames interface {
	Canal
	// Brancher pose les crochets du canal sur `obs`, une observation a lui que la marche des trames
	// portera ; `m` est ce que la marche expose pendant ses tours.
	Brancher(obs *Observation, m *MarcheDistribuee)
	// Trame recoit chaque trame delta, dans l ordre du flux, apres sa marche.
	Trame(p *lecture.Paquet)
}

// CanalDesTetes est un canal qui lit la TETE de chaque trame delta : la continuation de sa vue A
// et, quand elle annonce un message, son genre ([lecture.Paquet.VueA], [rangerLaTete], [teteDe]) ;
// la vue A rangee porte aussi les messages qui suivent quand le film la rend lisible. Une
// distribution sans canal des trames n en marche pas les records : elle n en lit que les vues A.
type CanalDesTetes interface {
	Canal
	// Tete recoit chaque trame delta (payload non vide), dans l ordre du flux, sa tete rangee.
	Tete(p *lecture.Paquet)
}

// MarcheDistribuee est ce qu un canal voit de la marche. Le paquet, la marche d ancres et la table
// d entites sont ceux de la marche : valides le temps du tour qui les rend et des crochets qu il
// declenche.
type MarcheDistribuee struct {
	// EnTete porte les parametres hors flux de la marche ([FilmContext.EnTete]).
	EnTete EnTete
	// Paquet est le paquet en cours, image-cle ou trame ; son en-tete est pose AVANT sa marche, et
	// les crochets le lisent.
	Paquet *lecture.Paquet
	// Ancres est la marche d ancres du paquet d image-cle en cours ([FilmContext.MarcheDImageCle]) :
	// ses ancres dans l ordre de la marche, les candidats que son repli a ecartes, ses decisions.
	// Vide pendant les trames.
	Ancres MarcheDePayload
	// Entites est la table d entites de la phase delta ; nil pendant les images-cles, qui ne tiennent
	// pas de monde.
	Entites lecture.Entites
}

// BilanDeMarche est ce que la marche rend d elle-meme a la cloture.
type BilanDeMarche struct {
	// Obs est l observation de la marche des trames, crochets fondus, NEW refuses soldes ; nil quand
	// aucun canal ne lit les trames.
	Obs *Observation
	// Liaisons est ce que la liaison des images-cles a fait au monde, sommee sur les chunks.
	Liaisons LiaisonDUnChunk
	// ChunksLus est le nombre de chunks de donnees que la marche a pu lire, porteurs d image-cle ou
	// de trame ou non ([FilmContext.ChunkAt]).
	ChunksLus int
}

// Distribuer marche les phases du film que ses canaux lisent, une fois chacune, et donne chaque
// paquet a chaque canal qui le lit : les images-cles a qui les lit — et a la marche des trames, qui
// y lit ses preliminaires —, les trames marchees quand un canal les lit, sinon leurs seules tetes
// quand un canal de tete les lit. Le decoupage MPP du format est pose sur le contexte pendant la
// phase des images-cles seulement, comme [FilmContext.ImagesCles] ; la phase delta se lit sous le
// cadre du contexte ([FilmContext.Trames]).
//
// La marche des trames exige le registre du film, et c est la seule erreur d une distribution avec
// [ErrCrochetDejaPose] : une distribution sans canal des trames ne peut pas echouer
// ([distribuerSansMarcherLesTrames]).
func Distribuer(fc *FilmContext, canaux ...Canal) error {
	l := lecteursDe(canaux)
	if len(l.trames) == 0 {
		distribuerSansMarcherLesTrames(fc, canaux)
		return nil
	}
	reg, err := fc.Registry()
	if err != nil {
		return err
	}
	return distribuerLesDeuxPhases(fc, reg, canaux, l)
}

// lecteursDesPhases : les canaux d une distribution, ranges par phase lue, dans l ordre des canaux.
type lecteursDesPhases struct {
	images []CanalDImageCle
	trames []CanalDesTrames
	tetes  []CanalDesTetes
}

// lecteursDe range les canaux par phase lue.
func lecteursDe(canaux []Canal) lecteursDesPhases {
	var l lecteursDesPhases
	for _, c := range canaux {
		if k, ok := c.(CanalDImageCle); ok {
			l.images = append(l.images, k)
		}
		if t, ok := c.(CanalDesTrames); ok {
			l.trames = append(l.trames, t)
		}
		if t, ok := c.(CanalDesTetes); ok {
			l.tetes = append(l.tetes, t)
		}
	}
	return l
}

// distribuerSansMarcherLesTrames joue, pour des canaux qui ne lisent pas les trames marchees, la
// phase des images-cles quand l un d eux la lit, puis la passe des tetes quand l un d eux les lit
// ([distribuerLesTetes]), et clot les canaux.
//
// LE REGISTRE N Y EST PAS EXIGE : la marche d ancres ne le lit pas, la passe des tetes non plus.
// Sans lui, aucun interet ne se resout, donc aucun corps n est parcouru, et les canaux recoivent
// les ancres — ce que les balayages d image-cle lisent sur un film sans `chunk_00` (decision 4 du
// lot 2.2). L erreur du registre reste celle du contexte ([FilmContext.Registry]) : un canal qui lit
// des corps la consulte.
func distribuerSansMarcherLesTrames(fc *FilmContext, canaux []Canal) {
	l := lecteursDe(canaux)
	lus := 0
	if len(l.images) > 0 {
		lus = distribuerLaPhaseDesImagesCles(fc, demandeDImagesCles{}, canaux, l.images)
	}
	if len(l.tetes) > 0 {
		lus = distribuerLesTetes(fc, l.tetes)
	}
	for _, c := range canaux {
		c.Clore(BilanDeMarche{ChunksLus: lus})
	}
}

// distribuerLesImagesClesSeules marche la seule phase des images-cles pour des canaux d image-cle,
// et les clot.
func distribuerLesImagesClesSeules(fc *FilmContext, canaux []Canal) {
	distribuerLaDemande(fc, demandeDImagesCles{}, canaux)
}

// distribuerLesImagesClesDesChunks est [distribuerLesImagesClesSeules] sur les seuls chunks
// `chunks`, dans leur ordre : un chunk que le film ne porte pas est saute, comme
// [FilmContext.ChunkAt] le dit.
func distribuerLesImagesClesDesChunks(fc *FilmContext, chunks []int, canaux []Canal) {
	distribuerLaDemande(fc, demandeDImagesCles{chunks: chunks}, canaux)
}

// distribuerLaDemande marche la seule phase des images-cles selon `d` (ses chunks, sa marche
// d ancres) pour les canaux d image-cle, et clot les canaux.
func distribuerLaDemande(fc *FilmContext, d demandeDImagesCles, canaux []Canal) {
	lus := distribuerLaPhaseDesImagesCles(fc, d, canaux, lecteursDe(canaux).images)
	for _, c := range canaux {
		c.Clore(BilanDeMarche{ChunksLus: lus})
	}
}

// distribuerLaPhaseDesImagesCles marche la phase des images-cles hors de la marche des trames selon
// `d`, le registre et les interets des canaux resolus ici (sans registre : les ancres seules), et
// rend le nombre de chunks lus.
func distribuerLaPhaseDesImagesCles(fc *FilmContext, d demandeDImagesCles, canaux []Canal,
	images []CanalDImageCle) int {
	reg, err := fc.Registry()
	if err != nil {
		reg = nil
	}
	d.reg, d.interets = reg, resoudreLesInterets(reg, canaux)[PhaseImagesCles]
	return distribuerLesImagesCles(fc, d, &MarcheDistribuee{EnTete: fc.EnTete()}, images, nil)
}

// distribuerLesDeuxPhases marche les images-cles puis les trames, les preliminaires de la marche
// des trames lus dans la meme phase des images-cles que les canaux ; chaque trame marchee va aux
// canaux des trames puis aux canaux de tete.
func distribuerLesDeuxPhases(fc *FilmContext, reg *Registry, canaux []Canal, l lecteursDesPhases) error {
	interets := resoudreLesInterets(reg, canaux)
	m := &MarcheDistribuee{EnTete: fc.EnTete()}
	obs, err := brancherLesCanaux(m, l.trames)
	if err != nil {
		return err
	}
	prel := nouveauxPreliminaires(fc)
	lus := distribuerLesImagesCles(fc, demandeDImagesCles{reg: reg, interets: interets[PhaseImagesCles]}, m,
		l.images, prel)
	mt, err := fc.marcheurDesTramesDepuis(reg, obs, prel)
	if err != nil {
		return err
	}
	mt.interets = interets[PhaseTrames]
	m.Paquet, m.Entites = &mt.paquet, mt.entites
	mt.parcourir(func(t *trameLue) bool {
		for _, c := range l.trames {
			c.Trame(t.paquet)
		}
		for _, c := range l.tetes {
			c.Tete(t.paquet)
		}
		return true
	})
	obs.solderLesNeufsRefuses()
	b := BilanDeMarche{Obs: obs, Liaisons: mt.liaisons, ChunksLus: lus}
	for _, c := range canaux {
		c.Clore(b)
	}
	return nil
}

// demandeDImagesCles est ce qu une distribution demande a la phase des images-cles : le registre
// (nil : aucun corps n est parcouru), les interets de la phase, les chunks marches (nil : ceux du
// film) et la marche d ancres (nil : celle du film, [FilmContext.MarcheDImageCle] ; le principe des
// doutes d absence se teste sous la marche sans preuve, `player_entities_test.go`).
type demandeDImagesCles struct {
	reg      *Registry
	interets interetsResolus
	chunks   []int
	marche   *MarcheDImageCle
}

// distribuerLesImagesCles marche la phase des images-cles, en donne chaque paquet aux canaux puis
// aux preliminaires de la marche des trames quand il y en a (qu elle clot), et rend le nombre de
// chunks lus.
func distribuerLesImagesCles(fc *FilmContext, d demandeDImagesCles, m *MarcheDistribuee, canaux []CanalDImageCle,
	prel *preliminairesDesTrames) int {
	k, restaurer := fc.nouvelleMarcheDesImagesCles(d, false)
	defer restaurer()
	m.Paquet = &k.paquet
	lus := k.parcourir(func(p *lecture.Paquet) bool {
		m.Ancres = k.ancres
		for _, c := range canaux {
			c.ImageCle(p, m)
		}
		prel.recevoir(p, m)
		return true
	})
	m.Ancres = MarcheDePayload{}
	prel.clore()
	return lus
}

// interetsResolus est l union des interets des canaux d une marche dans UNE phase, resolue dans le
// registre du film : par archetype, le masque des index d iteration (moins de 64) que les canaux
// interpretent. Dans la phase des images-cles, un archetype present est un corps a parcourir.
type interetsResolus map[int]uint64

// interetsDesPhases : les interets resolus de chaque phase, indexes par [Phase].
type interetsDesPhases [nombreDePhases]interetsResolus

// contient dit si l occurrence d index `index` de l archetype `ti` est interpretee.
func (r interetsResolus) contient(ti, index int) bool {
	return index >= 0 && index < 64 && r[ti]>>uint(index)&1 == 1
}

// parcourt dit si un canal lit les composants de l archetype `ti`.
func (r interetsResolus) parcourt(ti int) bool {
	_, ok := r[ti]
	return ok
}

// resoudreLesInterets resout les interets des canaux dans le registre du film, phase par phase. Un
// composant que l archetype ne declare pas n est l interet de rien : un canal ne lit pas ce que le
// build ne porte pas. Sans registre (nil), rien ne se resout.
func resoudreLesInterets(reg *Registry, canaux []Canal) interetsDesPhases {
	var out interetsDesPhases
	for i := range out {
		out[i] = interetsResolus{}
	}
	if reg == nil {
		return out
	}
	for _, c := range canaux {
		for _, it := range c.Interets() {
			arch, ok := reg.Archetype(it.TI)
			if !ok || it.Phase >= nombreDePhases {
				continue
			}
			for _, i := range arch.indicesOf(it.Composant) {
				if i < 64 {
					out[it.Phase][it.TI] |= 1 << uint(i)
				}
			}
		}
	}
	return out
}
