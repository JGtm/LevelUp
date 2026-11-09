package service

// tactical_service_groupes.go — LES CARTES DE LA TACTIQUE SONT DES GROUPES (plan cartes variantes,
// E3).
//
// Une carte de base et ses variantes déclarées (map_variants.toml : « - Ranked », « Heavies ») sont UNE
// vignette, UNE lecture de placement, UN « N sur M » : le client désigne la carte par l'identifiant de
// BASE du groupe (celui que porte la vignette), et toute lecture résout cet identifiant en la liste des
// identifiants du groupe ICI, au seul endroit (`cartesDe`) — puis la passe au lecteur par
// `domain.TacticalQuery.MapIDs`. Aucune lecture ne reçoit un identifiant seul.
//
// La table est lue UNE fois par service, donc par requête (la factory de l'onglet construit un service
// par appel). Lecteur absent ou lecture en échec : aucun groupe, chaque identifiant reste seul
// (mapgroups.Load journalise).

import (
	"context"
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/mapgroups"
)

// WithMapGroups injecte le lecteur des groupes de cartes. Sans injection, chaque carte reste seule.
func (s *TacticalService) WithMapGroups(r port.MapGroupReader) *TacticalService {
	s.mapGroups = r
	return s
}

// groupes rend la table des groupes de cartes, lue au premier appel.
func (s *TacticalService) groupes(ctx context.Context) domain.GroupesDeCartes {
	s.groupesMu.Lock()
	defer s.groupesMu.Unlock()
	if !s.groupesLus {
		s.groupesTable = mapgroups.Load(ctx, s.mapGroups)
		s.groupesLus = true
	}
	return s.groupesTable
}

// cartesDe rend les identifiants à lire pour la carte désignée par `carte` (l'identifiant de base du
// groupe, ou n'importe lequel de ses membres) : tout le groupe, ou `[carte]` hors groupe. Nil pour une
// carte vide — « toutes les cartes », ce que `TacticalQuery.MapIDs` vide veut dire.
func (s *TacticalService) cartesDe(ctx context.Context, carte string) []string {
	if carte == "" {
		return nil
	}
	return s.groupes(ctx).Membres(carte)
}

// fusionnerLesCartes replie les rangées par identifiant du lecteur en UNE carte par groupe : somme des
// parties, victoires et défaites, identifiant de base du groupe et libellé de la BASE (jamais celui d'une
// variante). Le plancher de matchs est recalculé après la somme. Ordre : parties décroissantes, puis
// identifiant. Sans groupe, les rangées passent telles quelles.
func fusionnerLesCartes(rows []domain.TacticalMapRow, groupes domain.GroupesDeCartes) []domain.TacticalMapRow {
	if groupes.Vide() {
		return rows
	}
	index := make(map[string]int, len(rows))
	out := make([]domain.TacticalMapRow, 0, len(rows))
	for _, r := range rows {
		cle := groupes.Groupe(r.MapID)
		i, vu := index[cle]
		if !vu {
			index[cle] = len(out)
			out = append(out, domain.TacticalMapRow{MapID: cle, MapName: r.MapName})
			i = len(out) - 1
		}
		out[i].Matchs += r.Matchs
		out[i].Victoires += r.Victoires
		out[i].Defaites += r.Defaites
		if base, ok := groupes.Noms(r.MapID); ok {
			out[i].MapName = base
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Matchs != out[j].Matchs {
			return out[i].Matchs > out[j].Matchs
		}
		return out[i].MapID < out[j].MapID
	})
	return out
}
