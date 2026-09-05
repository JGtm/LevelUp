package coordination

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// FenetreEchangeMs est la duree pendant laquelle une mort peut encore etre vengee : 5 s.
//
// Valeur arretee par l'utilisateur le 2026-09-05 (autorite mecanique de jeu). Elle n'est pas
// un parametre d'appel : c'est une regle du jeu, pas un reglage d'appelant — deux surfaces
// qui la choisiraient separement publieraient deux taux d'echange differents sous le meme
// nom.
const FenetreEchangeMs int64 = 5000

// Echanges suit chaque mort et dit si un coequipier a abattu le tueur dans la fenetre.
//
// DEFINITION. Un echange est un evenement dont la victime est le TUEUR de la mort initiale
// et dont l'auteur est un COEQUIPIER de la victime initiale, survenu dans les
// FenetreEchangeMs qui suivent (bornes comprises).
//
// LES DEUX CAS LIMITES, tranches par l'utilisateur le 2026-09-05 :
//
//   - un tueur qui abat DEUX coequipiers puis tombe dans la fenetre venge LES DEUX morts.
//     Chaque mort cherche son vengeur pour son propre compte ; il n'y a pas d'appariement
//     un-pour-un a faire, et un seul kill peut donc solder plusieurs morts ;
//   - un tueur mort de l'environnement, d'une grenade perdue ou de lui-meme n'echange RIEN.
//     Seul un kill de coequipier compte : un evenement sans auteur, ou dont l'auteur n'est
//     pas de l'equipe de la victime initiale, ne venge personne.
//
// Une mort dont le tueur est inconnu, ou dont les equipes ne sont pas connues, n'est pas
// VENGEABLE : elle sort du denominateur au lieu d'y compter comme un echec (personne ne
// pouvait la venger).
func Echanges(kills []domain.KillEvent, equipes domain.EquipesParMatch) domain.BilanEchanges {
	bilan := domain.BilanEchanges{Morts: make([]domain.MortSuivie, 0, len(kills))}
	for matchID, evenements := range grouperParMatch(kills) {
		parVictime := indexerParVictime(evenements)
		equipesDuMatch := equipes[matchID]
		for _, d := range evenements {
			m := domain.MortSuivie{
				MatchID:     matchID,
				VictimeXUID: d.VictimXUID,
				TueurXUID:   d.KillerXUID,
				TimeMs:      d.TimeMs,
				Vengeable:   estVengeable(d, equipesDuMatch),
			}
			if m.Vengeable {
				bilan.NbVengeables++
				if r, trouve := chercheVengeur(d, parVictime[d.KillerXUID], equipesDuMatch); trouve {
					m.Vengee = true
					m.VengeurXUID = r.KillerXUID
					m.DelaiMs = r.TimeMs - d.TimeMs
					bilan.NbVengees++
				}
			}
			bilan.Morts = append(bilan.Morts, m)
		}
	}
	trierMorts(bilan.Morts)
	bilan.Paires = agregerPaires(bilan.Morts)
	return bilan
}

// estVengeable : il faut un tueur identifie, distinct de la victime (un suicide ne se venge
// pas), et deux equipes CONNUES et differentes. Une equipe inconnue ne se devine pas.
func estVengeable(d domain.KillEvent, equipes map[string]int) bool {
	if d.KillerXUID == "" || d.KillerXUID == d.VictimXUID {
		return false
	}
	equipeVictime, connueV := equipes[d.VictimXUID]
	equipeTueur, connueT := equipes[d.KillerXUID]
	return connueV && connueT && equipeVictime != equipeTueur
}

// chercheVengeur rend le PREMIER kill du tueur, dans la fenetre, porte par un coequipier de
// la victime. `candidats` est trie par instant croissant, d'ou l'arret des que la fenetre
// est depassee.
func chercheVengeur(d domain.KillEvent, candidats []domain.KillEvent, equipes map[string]int) (domain.KillEvent, bool) {
	equipeVictime := equipes[d.VictimXUID]
	for _, r := range candidats {
		delai := r.TimeMs - d.TimeMs
		if delai < 0 {
			continue
		}
		if delai > FenetreEchangeMs {
			break
		}
		// Sans auteur (environnement, chute) ou par la victime elle-meme (deja morte) :
		// aucun echange. Le second cas ne devrait pas exister dans les donnees ; s'il
		// apparait, c'est une anomalie, pas une vengeance.
		if r.KillerXUID == "" || r.KillerXUID == d.VictimXUID {
			continue
		}
		equipeVengeur, connue := equipes[r.KillerXUID]
		if !connue || equipeVengeur != equipeVictime {
			continue
		}
		return r, true
	}
	return domain.KillEvent{}, false
}

// grouperParMatch decoupe les evenements par match et les trie par instant. Rien ne traverse
// la frontiere d'un match : deux matchs partagent les memes joueurs et la meme horloge, les
// melanger fabriquerait des echanges qui n'ont jamais eu lieu.
func grouperParMatch(kills []domain.KillEvent) map[string][]domain.KillEvent {
	parMatch := make(map[string][]domain.KillEvent)
	for _, k := range kills {
		parMatch[k.MatchID] = append(parMatch[k.MatchID], k)
	}
	for _, evenements := range parMatch {
		trierEvenements(evenements)
	}
	return parMatch
}

// indexerParVictime range les evenements d'un match par victime, en conservant l'ordre
// chronologique : chercher le vengeur d'une mort revient a lire les morts de son tueur.
func indexerParVictime(evenements []domain.KillEvent) map[string][]domain.KillEvent {
	index := make(map[string][]domain.KillEvent)
	for _, e := range evenements {
		index[e.VictimXUID] = append(index[e.VictimXUID], e)
	}
	return index
}

// agregerPaires compte les echanges par couple (vengeur, venge) et leur delai moyen.
func agregerPaires(morts []domain.MortSuivie) []domain.PaireEchange {
	type cumul struct {
		nombre int
		delais int64
	}
	parPaire := make(map[[2]string]*cumul)
	for _, m := range morts {
		if !m.Vengee {
			continue
		}
		cle := [2]string{m.VengeurXUID, m.VictimeXUID}
		c := parPaire[cle]
		if c == nil {
			c = &cumul{}
			parPaire[cle] = c
		}
		c.nombre++
		c.delais += m.DelaiMs
	}
	paires := make([]domain.PaireEchange, 0, len(parPaire))
	for cle, c := range parPaire {
		paires = append(paires, domain.PaireEchange{
			VengeurXUID:  cle[0],
			VengeXUID:    cle[1],
			Nombre:       c.nombre,
			DelaiMoyenMs: float64(c.delais) / float64(c.nombre),
		})
	}
	sort.Slice(paires, func(i, j int) bool {
		if paires[i].VengeurXUID != paires[j].VengeurXUID {
			return paires[i].VengeurXUID < paires[j].VengeurXUID
		}
		return paires[i].VengeXUID < paires[j].VengeXUID
	})
	return paires
}

// trierEvenements ordonne par instant, puis par victime et par tueur : deux morts au meme
// instant doivent se lire dans le meme ordre a chaque execution.
func trierEvenements(evenements []domain.KillEvent) {
	sort.SliceStable(evenements, func(i, j int) bool {
		a, b := evenements[i], evenements[j]
		if a.TimeMs != b.TimeMs {
			return a.TimeMs < b.TimeMs
		}
		if a.VictimXUID != b.VictimXUID {
			return a.VictimXUID < b.VictimXUID
		}
		return a.KillerXUID < b.KillerXUID
	})
}

// trierMorts ordonne la sortie : le parcours des matchs est un parcours de map.
func trierMorts(morts []domain.MortSuivie) {
	sort.SliceStable(morts, func(i, j int) bool {
		a, b := morts[i], morts[j]
		if a.MatchID != b.MatchID {
			return a.MatchID < b.MatchID
		}
		if a.TimeMs != b.TimeMs {
			return a.TimeMs < b.TimeMs
		}
		return a.VictimeXUID < b.VictimeXUID
	})
}
