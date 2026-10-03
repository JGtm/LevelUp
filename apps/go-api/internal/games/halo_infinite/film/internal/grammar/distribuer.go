package grammar

// distribuer.go — UNE MARCHE, N CANAUX (ADR 0037 ; decision DT2-1 du plan de l etape 2).
//
// [Distribuer] marche UNE fois les deux phases du film — les images-cles, puis les trames delta —
// et donne chaque paquet range a chaque canal, dans l ordre des canaux. Un canal ([Canal]) declare
// ses interets et pose ses crochets sur l observation de la marche des trames : l interpretation
// reste PENDANT la marche (ADR 0037 IR-8), la ou elle a lieu aujourd hui — la phase des images-cles
// se lit sous l observation du contexte, comme [FilmContext.ImagesCles]. Un canal ne lit jamais un
// octet ; il lit la structure ([lecture.Paquet]) et ce que ses crochets recoivent.
//
// # LES INTERETS SONT DES PAIRES (ARCHETYPE, COMPOSANT)
//
// Resolues dans le registre du film, jamais un nom de composant seul : deux tables de composant
// homonymes existent (« high-frequency », ti=3 et ti=4, deux grammaires — regle « routage par
// archetype ou par table de composant » de la campagne de grammaire). Une occurrence est
// INTERPRETEE quand son archetype et son index sont dans l union des interets et qu elle est
// traversee (decision DT2-2) ; la structure la marque [lecture.EtatInterprete].
//
// # DEUX CANAUX NE POSENT PAS LE MEME CROCHET
//
// Chaque canal pose les siens sur une observation a lui, que le distributeur fond dans celle de la
// marche ([fondreLesCrochets]) : un crochet pose par deux canaux est une erreur, parce que le second
// ecraserait le premier sans que rien ne le dise.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// Interet est une paire (archetype, composant) qu un canal interprete : l index de type de
// l archetype dans le registre du film, et le nom du composant tel que le registre l ecrit (un
// canal qui connait deux orthographes declare les deux).
type Interet struct {
	TI        int
	Composant string
}

// Canal est un consommateur de la marche du film.
type Canal interface {
	// Interets rend les paires (archetype, composant) que le canal interprete.
	Interets() []Interet
	// Brancher pose les crochets du canal sur `obs`, une observation a lui que la marche des trames
	// portera ; `m` est ce que la marche expose pendant ses tours.
	Brancher(obs *Observation, m *MarcheDistribuee)
	// ImageCle recoit chaque paquet d image-cle, dans l ordre du flux.
	ImageCle(p *lecture.Paquet)
	// Trame recoit chaque trame delta, dans l ordre du flux, apres sa marche.
	Trame(p *lecture.Paquet)
	// Clore recoit le bilan de la marche, une fois les deux phases finies.
	Clore(b BilanDeMarche)
}

// MarcheDistribuee est ce qu un canal voit de la marche. Le paquet et la table d entites sont ceux
// de l arene : valides le temps du tour qui les rend et des crochets qu il declenche.
type MarcheDistribuee struct {
	// EnTete porte les parametres hors flux de la marche ([FilmContext.EnTete]).
	EnTete EnTete
	// Paquet est le paquet en cours, image-cle ou trame ; son en-tete est pose AVANT sa marche, et
	// les crochets le lisent.
	Paquet *lecture.Paquet
	// Entites est la table d entites de la phase delta ; nil pendant les images-cles, qui ne tiennent
	// pas de monde.
	Entites lecture.Entites
}

// BilanDeMarche est ce que la marche rend d elle-meme a la cloture.
type BilanDeMarche struct {
	// Obs est l observation de la marche, crochets fondus, NEW refuses soldes.
	Obs *Observation
	// Liaisons est ce que la liaison des images-cles a fait au monde, sommee sur les chunks.
	Liaisons LiaisonDUnChunk
}

// Distribuer marche les deux phases du film une fois et donne chaque paquet a chaque canal. Le
// decoupage MPP du format est pose sur le contexte pendant la phase des images-cles seulement,
// comme [FilmContext.ImagesCles] ; la phase delta se lit sous le cadre du contexte
// ([FilmContext.Trames]).
func Distribuer(fc *FilmContext, canaux ...Canal) error {
	reg, err := fc.Registry()
	if err != nil {
		return err
	}
	interets := resoudreLesInterets(reg, canaux)
	m := &MarcheDistribuee{EnTete: fc.EnTete()}
	obs, err := brancherLesCanaux(m, canaux)
	if err != nil {
		return err
	}
	if err := distribuerLesImagesCles(fc, interets, m, canaux); err != nil {
		return err
	}
	mt, err := fc.nouveauMarcheurDesTrames(obs)
	if err != nil {
		return err
	}
	mt.interets = interets
	m.Paquet, m.Entites = &mt.paquet, mt.entites
	mt.parcourir(func(t *trameLue) bool {
		for _, c := range canaux {
			c.Trame(t.paquet)
		}
		return true
	})
	obs.solderLesNeufsRefuses()
	b := BilanDeMarche{Obs: obs, Liaisons: mt.liaisons}
	for _, c := range canaux {
		c.Clore(b)
	}
	return nil
}

// distribuerLesImagesCles marche la phase des images-cles et en donne chaque paquet aux canaux.
func distribuerLesImagesCles(fc *FilmContext, interets interetsResolus, m *MarcheDistribuee, canaux []Canal) error {
	k, restaurer, err := fc.nouvelleMarcheDesImagesCles(interets)
	if err != nil {
		return err
	}
	defer restaurer()
	m.Paquet = &k.paquet
	k.parcourir(func(p *lecture.Paquet) bool {
		for _, c := range canaux {
			c.ImageCle(p)
		}
		return true
	})
	return nil
}

// interetsResolus est l union des interets des canaux d une marche, resolue dans le registre du
// film : par archetype, le masque des index d iteration (moins de 64) que les canaux interpretent.
type interetsResolus map[int]uint64

// contient dit si l occurrence d index `index` de l archetype `ti` est interpretee.
func (r interetsResolus) contient(ti, index int) bool {
	return index >= 0 && index < 64 && r[ti]>>uint(index)&1 == 1
}

// resoudreLesInterets resout les interets des canaux dans le registre du film. Un composant que
// l archetype ne declare pas n est l interet de rien : un canal ne lit pas ce que le build ne porte
// pas.
func resoudreLesInterets(reg *Registry, canaux []Canal) interetsResolus {
	out := interetsResolus{}
	for _, c := range canaux {
		for _, it := range c.Interets() {
			arch, ok := reg.Archetype(it.TI)
			if !ok {
				continue
			}
			for _, i := range arch.indicesOf(it.Composant) {
				if i < 64 {
					out[it.TI] |= 1 << uint(i)
				}
			}
		}
	}
	return out
}
