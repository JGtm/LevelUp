// Package skill_v2 implémente le modèle TrueSkill / TrueSkill 2 pour le LUSR v2.
//
// Conçu pour cohabiter avec le LUSR v1 historique (cf. internal/sync/skill_rating.go)
// sans dépendance croisée. Aucun import de internal/sync. Toutes les fonctions sont
// stateless et pures — pas d'accès DB ; la persistance vit dans internal/service/skill_v2_service.go.
//
// Phase 1 (cette implémentation) — TrueSkill classique :
//   - skill latent gaussien par joueur (mu, sigma²)
//   - mise à jour online par match (1 forward pass, pas de batch)
//   - équipes de tailles quelconques (closed-form 2-team)
//   - draws supportés
//
// Phases ultérieures (non implémentées ici, cf. docs/adr/0022-lusr-v2.md à venir) :
//   - Phase 2 : squadOffset par taille de squad (TS2 §6)
//   - Phase 3 : kills/deaths comme observations (TS2 §8) + quit penalty (TS2 §9)
//     ⇒ nécessitera un factor graph + Expectation Propagation, refactor du closed-form
//   - Phase 4 : base + offset par mode (TS2 §11)
//   - Phase 5 : TrueSkill Through Time (batch)
//
// Références :
//   - Herbrich, Minka, Graepel — TrueSkill: A Bayesian Skill Rating System (NIPS 2006)
//   - Minka, Cleven, Zaykov — TrueSkill 2: An improved Bayesian skill rating system (MSR 2018)
//   - Menke (GDC) — Significantly Improving your Skill System with TrueSkill Through Time
package skill_v2
