package grammar

// diagnostic.go — LES CODES DES DIAGNOSTICS DE `grammar` (lot J12.3, ADR 0034 D-4).
//
// La couche ne journalise pas : elle note des [constat.Diagnostic] dans les diagnostics du
// contexte du film ([FilmContext.Diagnostics]), que l orchestrateur releve et journalise avec son
// `ctx` (cf. l en-tete du paquet `constat`).

import "levelup/go-api/internal/games/halo_infinite/film/internal/constat"

// Les diagnostics de `grammar`.
const (
	// DiagProfilIncomplet : une cle ecrite dans le film manque a la table de profil.
	DiagProfilIncomplet constat.Code = "grammar.profil_incomplet"
	// DiagRegistreInconnu : l empreinte du registre ECS du film n est pas celle du binaire de
	// reference.
	DiagRegistreInconnu constat.Code = "grammar.registre_inconnu"
	// DiagAnticipationActive : le repli du lot 5.23 (liaison par anticipation) a servi sur ce film.
	DiagAnticipationActive constat.Code = "grammar.anticipation_active"
	// DiagGrenadeAmorceAbsente : la cle du film est absente de la table d amorce de grenade.
	DiagGrenadeAmorceAbsente constat.Code = "grammar.grenade_amorce_absente"
	// DiagGrenadeArchetypeNonResolu : l archetype projectile n a pas ete resolu par le nom.
	DiagGrenadeArchetypeNonResolu constat.Code = "grammar.grenade_archetype_non_resolu"
	// DiagGrenadeCouverture : la couverture du balayage des lancers de grenade.
	DiagGrenadeCouverture constat.Code = "grammar.grenade_couverture"
)

// cleDiagSlot : la cle d attribut du slot dans les diagnostics de la couche.
const cleDiagSlot = "slot"
