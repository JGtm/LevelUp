package notifications

// routes.go — construction canonique des TargetRoute joueur title-scopées.
//
// Point unique de vérité du format d'URL des notifications joueur : tout émetteur
// (post-sync, media, livesync) passe par PlayerTargetRoute — jamais de fmt.Sprintf
// ad hoc. Le garde-rail routes_guard_test.go interdit toute re-divergence (leçon
// CLAUDE.md n°6 : un littéral de route recopié re-diverge en N copies).

// PlayerTargetRoute construit la route title-scopée d'une notification joueur au
// format canonique post-D7 (« titre dans l'URL », .ai/PLAN_TITLE_SLUG_URL_2026-07.md) :
//
//	/t/{titleSlug}/players/{playerSlug}/{suffix}
//
// Pourquoi ce format. Depuis le chantier D7 les pages joueur du front vivent sous
// /t/{titleSlug}/players/{playerSlug}/… . Émettre ce format DIRECTEMENT donne au
// clic un zéro hop : le splat de redirection legacy (/players/**) ne couvre QUE le
// stock de notifications déjà persistées en ancien format (/players/…), pas les
// nouvelles émissions.
//
// Contrat des arguments.
//   - titleSlug : titre RÉEL du contexte d'émission — jamais un slug en dur, jamais
//     un défaut halo_infinite (title-agnostic). Provenance : pdb.TitleSlug côté
//     post-sync, ctxkeys.TitleSlug(ctx) côté handlers HTTP, le titre annoncé côté
//     livesync.
//   - suffix : chemin sous la racine joueur, SANS « / » de tête (ex. "home",
//     "stats/synthesis", "career/citations", "matches/"+matchID). Le suffixe doit
//     correspondre à une route front RÉELLE (pas une redirection interne) pour rester
//     zéro hop.
//
// Cas suffix vide : produit la racine joueur "/t/{titleSlug}/players/{playerSlug}/".
// AUCUN émetteur n'utilise ce cas (vérifié sur pièces le 2026-07-23) — documenté et
// couvert par TestPlayerTargetRoute, sans branche spéciale ici.
func PlayerTargetRoute(titleSlug, playerSlug, suffix string) string {
	return "/t/" + titleSlug + "/players/" + playerSlug + "/" + suffix
}
