/**
 * Feature flags frontend — activation manuelle de fonctionnalités en développement.
 *
 * Pour activer une fonctionnalité : passer la constante à `true`.
 * Ces flags sont intentionnellement statiques (pas de fetch réseau) pour les
 * fonctionnalités dont l'activation est liée à un déploiement externe.
 */

/**
 * Rejouer le match en 2D — visualisation spatiale d'un match en vue du dessus.
 * Projet externe en alpha ; activer lorsque l'intégration sera stable.
 * @see https://github.com/TODO — repo du projet replay à renseigner ici
 */
export const REJEU_2D_ENABLED = false
