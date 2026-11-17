package tactical

// reapparitions_adverses.go — LA LECTURE « REAPPARITIONS ADVERSES » : ou les adversaires reapparaissent,
// en general et quand l'equipe tient une zone choisie. Pur : aucune base, aucun fichier.
//
// # « ZONE TENUE » (ADR du plan, D-5)
//
// A l'instant ou un adversaire reapparait, la zone choisie est TENUE par l'equipe du joueur de la page
// quand au moins domain.ReapparitionsAdversesJoueursPourTenir joueurs VIVANTS de cette equipe se
// trouvent dans le volume de la zone : leur point est dans sa forme au sol ET sa hauteur dans sa tranche
// (marge de MargeTrancheZM). L'appartenance est le CONTENU de la zone choisie, pas son nom en jeu :
// une zone qui en contient une autre, plus fine, tient pour les joueurs de l'une comme de l'autre.
//
// La matiere est l'instantane que le sidecar porte pour chaque reapparition : les AUTRES joueurs
// vivants a la frame de la reapparition, ceux qui apparaissent dans la meme seconde exclus (leur premier
// point est un spawn, pas une position de jeu). Le joueur de la page y figure quand il est vivant. Un
// equipier qui apparait dans la meme seconde que l'adversaire n'est donc pas compte : la regle sous-compte
// ces cas, jamais l'inverse.
//
// # OU RANGER UNE REAPPARITION
//
// Dans la zone nommee qui nomme son point (NommerZone, avec sa hauteur) : la meme regle que la
// lecture de placement, donc le meme nom que celui que l'utilisateur voit sur le plan. Une
// reapparition qu'aucune zone ne nomme est comptee dans le total sans zone (TacticalRepartitionZones.HorsZone),
// jamais rangee dans un nom de repli.

import (
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// ObservationAdverse est la reapparition d'un joueur de l'equipe ADVERSE, avec ce qu'on savait de
// l'equipe du joueur de la page a cet instant.
type ObservationAdverse struct {
	MatchID string
	// X, Y, Z : le point de la reapparition (metres monde).
	X, Y, Z float64
	// EquipeActive : l'equipe du joueur de la page dans CE match (designateur du film).
	EquipeActive int
	// Instantane : les autres joueurs vivants a la frame de la reapparition.
	Instantane []Vivant
}

// ZoneTenue dit si la zone est tenue par l'equipe `equipe` dans l'instantane : au moins `seuil`
// vivants de cette equipe dans son volume. Une zone sans forme ne contient personne.
func ZoneTenue(zone ZoneNommee, equipe int, instantane []Vivant) bool {
	if !aUneForme(zone) {
		return false
	}
	dedans := 0
	for _, v := range instantane {
		if v.Equipe == equipe && dansLeVolume(zone, v.X, v.Y, v.Z) {
			dedans++
		}
	}
	return dedans >= domain.ReapparitionsAdversesJoueursPourTenir
}

// dansLeVolume : le point (x, y, z) est dans la forme au sol de la zone et dans sa tranche.
func dansLeVolume(z ZoneNommee, x, y, h float64) bool {
	return dansForme(x, y, z) && dansTranche(z, h)
}

// ZoneAUneForme dit si la zone porte une forme au sol exploitable (condition pour savoir qui s'y trouve).
func ZoneAUneForme(z ZoneNommee) bool { return aUneForme(z) }

// LireReapparitionsAdverses range les observations dans les deux repartitions : la reference (toutes)
// et la zone tenue (celles dont la zone choisie est tenue par l'equipe du joueur). `loc` choisit la
// langue des noms de zone.
func LireReapparitionsAdverses(loc i18n.Locale, obs []ObservationAdverse, choisie ZoneNommee,
	zones []ZoneNommee) (reference, tenue domain.TacticalRepartitionZones) {
	toutes := newCompteur()
	tenues := newCompteur()
	for _, o := range obs {
		retenue, nommee := NommerZone(o.X, o.Y, []float64{o.Z}, zones)
		toutes.ajouter(o.MatchID, retenue, nommee)
		if ZoneTenue(choisie, o.EquipeActive, o.Instantane) {
			tenues.ajouter(o.MatchID, retenue, nommee)
		}
	}
	return toutes.repartition(loc), tenues.repartition(loc)
}

// compteurDeZones accumule une repartition.
type compteurDeZones struct {
	matchs  map[string]bool
	parZone map[int]int
	zones   map[int]ZoneNommee
	total   int
}

func newCompteur() *compteurDeZones {
	return &compteurDeZones{matchs: map[string]bool{}, parZone: map[int]int{}, zones: map[int]ZoneNommee{}}
}

func (c *compteurDeZones) ajouter(matchID string, r ZoneRetenue, nommee bool) {
	c.matchs[matchID] = true
	c.total++
	if !nommee {
		return
	}
	c.parZone[r.Zone.VolumeIndex]++
	c.zones[r.Zone.VolumeIndex] = r.Zone
}

// repartition fige le compteur. Sous un plancher (RaisonSousPlancher), les zones ne sont pas servies.
func (c *compteurDeZones) repartition(loc i18n.Locale) domain.TacticalRepartitionZones {
	out := domain.TacticalRepartitionZones{
		Matchs: len(c.matchs), Reapparitions: c.total, Zones: []domain.TacticalPartDeZone{},
	}
	if out.RaisonSousPlancher() != "" {
		return out
	}
	out.Suffisante = true
	for idx, n := range c.parZone {
		out.Zones = append(out.Zones, domain.TacticalPartDeZone{
			VolumeIndex: idx, Nom: c.zones[idx].Nom.PickOptional(loc), Reapparitions: n,
			Part: float64(n) / float64(c.total),
		})
	}
	sort.Slice(out.Zones, func(i, j int) bool {
		if out.Zones[i].Reapparitions != out.Zones[j].Reapparitions {
			return out.Zones[i].Reapparitions > out.Zones[j].Reapparitions
		}
		return out.Zones[i].VolumeIndex < out.Zones[j].VolumeIndex
	})
	return out
}
