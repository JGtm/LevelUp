package grammar

// marche_trames.go — LA PHASE DELTA DE LA REPRESENTATION INTERMEDIAIRE : LA MARCHE DES TRAMES
// (ADR 0037 IR-2, IR-3).
//
// # C EST LA MARCHE DE PRODUCTION, PAS UN SECOND MARCHEUR
//
// Le pilotage qui suit — la table anticipee posee une fois, puis chunk par chunk la liaison des
// images-cles au monde ([lierLeChunkAuMonde]), puis paquet par paquet la localisation des listes
// d evenements ([localiserLaListe]) et la marche par rangs ([lireTrameParRangs]) — est celui des
// etats de mouvement et du tir continu ([ScanMarcheDesTrames]) et de la carte de fermeture
// ([FrameClosure], [FrameClosureDetaillee]) : ils le CONSOMMENT, aucun ne le recopie. Garde-rail :
// `marche_trames_unique_test.go`.
//
// # CE QU ELLE RANGE, ET CE QU ELLE NE CHANGE PAS
//
// Chaque trame delta est rangee dans la structure de lecture ([lecture.Paquet]) : les vues et
// leurs etendues, la sortie typee de la vue B, les records avec leur etendue, leur liaison et leur
// preuve, les composants avec leur etat et la provenance de leur largeur, le verdict de fermeture.
// L INTERPRETATION RESTE AUX CROCHETS de l observation que le consommateur passe : ils publient
// pendant la marche, exactement comme avant, et aucune valeur publiee ne bouge (ADR 0037 IR-8).
// La structure ne se persiste pas.
//
// # DUREE DE VIE
//
// Le paquet rendu est l arene de la marche : il n est valide que pendant le tour qui le rend, et
// son en-tete (chunk, rang, horodatage) est pose AVANT la marche du paquet — les crochets qui
// publient pendant la marche le lisent.

import (
	"iter"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// marcheurDesTrames porte l etat d UNE marche des trames d un film : le monde, la marche des
// images-cles, l arene du paquet en cours et ce que la marche a lu de lui.
type marcheurDesTrames struct {
	fc     *FilmContext
	cfg    FrameConfig
	chunks []int
	monde  *World
	images MarcheDImageCle
	// paquet : l arene, reutilisee d un paquet a l autre ; entites : la table d entites du monde,
	// en lecture seule.
	paquet  lecture.Paquet
	entites lecture.Entites
	// trame : ce que la marche a lu du paquet en cours.
	trame trameLue
	// liaisons : ce que la liaison des images-cles a fait au monde, sommee sur les chunks.
	liaisons LiaisonDUnChunk
	// interets : les occurrences que les canaux de la marche interpretent ([Distribuer]) ; vide hors
	// du distributeur.
	interets interetsResolus
}

// trameLue est ce que la marche rend pour UNE trame delta : la structure, et ce que la marche par
// rangs a lu pour la remplir — que les consommateurs de la grammaire lisent sans relire un bit.
type trameLue struct {
	paquet *lecture.Paquet
	// debut : le bit de depart de la marche, -1 pour une liste d evenements non localisee.
	debut int
	// parRangs : la trame a ete marchee par classes de vue ; faux sous un profil de recherche qui
	// les retire, et alors seuls les records sont ranges.
	parRangs bool
	lecture  lectureDeTrame
}

// Trames rend la PHASE DELTA du film : chaque trame delta (type 0, payload non vide), dans
// l ordre du flux, rangee dans la structure de lecture (ADR 0037). `obs` porte les crochets qui
// interpretent pendant la marche (nil : aucun) ; ce sont eux, et eux seuls, qui disent ce que la
// marche publie.
//
// Le paquet rendu n est valide que pendant le tour qui le rend.
func (c *FilmContext) Trames(obs *Observation) iter.Seq2[*lecture.Paquet, error] {
	return func(rendre func(*lecture.Paquet, error) bool) {
		m, err := c.nouveauMarcheurDesTrames(obs)
		if err != nil {
			rendre(nil, err)
			return
		}
		m.parcourir(func(t *trameLue) bool { return rendre(t.paquet, nil) })
	}
}

// nouveauMarcheurDesTrames prepare la marche des trames du film sous le cadre de balayage du
// contexte, construit depuis l en-tete de la marche ([FilmContext.EnTete]), et l observation
// `obs` : le monde, et la table anticipee posee une fois, avant la premiere trame (les
// preliminaires bornes de l ADR 0037 IR-3).
func (c *FilmContext) nouveauMarcheurDesTrames(obs *Observation) (*marcheurDesTrames, error) {
	chunks := c.ChunkNumbers()
	if len(chunks) == 0 {
		return nil, ErrNoFilmChunk
	}
	reg, err := c.Registry()
	if err != nil {
		return nil, err
	}
	cfg := c.CadreDeBalayage()
	cfg.IDLowBits = c.EnTete().IDLowBits.Valeur
	cfg.Obs = obs
	m := &marcheurDesTrames{fc: c, cfg: cfg, chunks: chunks, monde: NewWorld(reg),
		images: c.MarcheDImageCle()}
	m.entites = entitesDuMonde{w: m.monde}
	m.trame.paquet = &m.paquet
	// LE REPLI DU LOT 5.23 ENTRE EN PRODUCTION ICI. La table anticipee est construite en UNE
	// passe sur les images-cles de tous les chunks, sans decodage de trame ; au point de rejet, la
	// marche y lit l archetype qu une image-cle ULTERIEURE donne a un eid que le monde ne connait
	// pas encore. Cf. `keyframe_anticipe.go` et [World.LierParRepliDAnticipation].
	m.monde.PoserTableAnticipee(ConstruireTableAnticipee(c))
	return m, nil
}

// parcourir deroule la marche sur tous les chunks et rend chaque trame delta a `rendre`, apres
// sa marche ; faux arrete la marche.
func (m *marcheurDesTrames) parcourir(rendre func(*trameLue) bool) {
	for _, c := range m.chunks {
		data, pks, ok := m.fc.ChunkAt(c)
		if !ok {
			continue
		}
		// La table anticipee ne rend qu une declaration STRICTEMENT POSTERIEURE a ce chunk.
		m.monde.PoserChunkCourant(c)
		m.liaisons.ajouter(lierLeChunkAuMonde(m.monde, m.images, data, pks, m.cfg.Obs))
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			m.marcherLePaquet(c, pk, data)
			if !rendre(&m.trame) {
				return
			}
		}
	}
}

// marcherLePaquet marche UNE trame delta et la range dans l arene. Les paquets a liste
// d evenements partent du debut que [localiserLaListe] leur trouve ; une liste non localisee
// n est pas lue.
func (m *marcheurDesTrames) marcherLePaquet(c int, pk FilmPacket, data []byte) {
	t, p := &m.trame, &m.paquet
	pay := pk.Payload(data)
	viderLePaquet(p)
	p.Chunk, p.Index, p.Type, p.TS, p.Payload, p.Entites = c, pk.Index, pk.Type, pk.TimestampUS, pay, m.entites
	t.debut, p.Debut = movementStateSkipLeadBits, lecture.DebutEnTete
	t.parRangs = m.cfg.Profil.Grammaire.ClassesDeVue
	t.lecture = lectureDeTrame{debutVueB: -1, finVueB: -1}
	if _, present := PacketHeadEventType(pay); present {
		t.debut, p.Debut = localiserLaListe(pay, m.monde, m.cfg)
		if t.debut < 0 {
			rangerUneListeNonLocalisee(p)
			return
		}
	}
	if t.parRangs {
		br := LecteurSur(pay)
		br.poserCadre(m.cfg)
		lireTrameParRangs(br, pay, m.monde, m.cfg, t.debut, &t.lecture)
	} else {
		t.lecture.recs, t.lecture.rangs, t.lecture.curseur = DecodeFrameViewsCurseur(pay, m.monde, m.cfg,
			MovementStateViews, t.debut)
	}
	rangerLaTrame(p, &t.lecture, t.parRangs, m.interets)
}

// viderLePaquet remet l arene a zero en gardant la capacite de ses tranches.
func viderLePaquet(p *lecture.Paquet) {
	genres, entrees, records, comps := p.VueA.Genres[:0], p.VueC.Entrees[:0], p.Records[:0], p.Comps[:0]
	*p = lecture.Paquet{}
	p.VueA.Genres, p.VueC.Entrees, p.Records, p.Comps = genres, entrees, records, comps
}

// ajouter cumule `o` dans `l`.
func (l *LiaisonDUnChunk) ajouter(o LiaisonDUnChunk) {
	l.Datums += o.Datums
	l.Ambigus += o.Ambigus
	l.Oubliees += o.Oubliees
}
