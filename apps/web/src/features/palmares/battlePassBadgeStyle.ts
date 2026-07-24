/**
 * battlePassBadgeStyle — tokens des pastilles pleines du pass saisonnier.
 *
 * Les pastilles de statut/premium du pass (accueil + page Pass saisonnier +
 * lightbox) sont rendues PLEINES (fond saturé + texte blanc), comme les pills
 * de joueur du hub Communaute > Relations (reference de l'app, cf.
 * RelationBadges -> NarrativeBadge solid).
 *
 * Contrainte d'accessibilite (memoire user + _encounterColors.ts) : un badge
 * plein a texte blanc exige un fond sombre garantissant l'AA sur blanc DANS
 * TOUTES les palettes. Seul le set `narrative-encounter-*` est a la fois sombre
 * ET palette-invariant (verifie par wcagContrast.test.ts) : les tokens
 * chart-series / outcome / success virent clairs en Okabe-Ito / Tol-Bright et
 * le texte blanc y devient illisible. On reutilise donc ce set (le meme que les
 * pills Relations) ; les teintes sont choisies par semantique et le LIBELLE
 * desambigue toujours la pastille (WCAG 1.4.1, jamais la couleur seule).
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

export type SeasonPassBadgeRole =
  | 'completed' // termine / obtenu
  | 'active' // actif / palier courant / en cours d'affichage
  | 'in-progress' // progression en cours
  | 'premium' // premium / possede
  | 'free' // gratuit
  | 'neutral' // statut inconnu / defaut

const ROLE_TOKEN: Record<SeasonPassBadgeRole, SemanticToken> = {
  completed: 'narrative-encounter-ally-plus', // vert     — termine / obtenu
  active: 'narrative-encounter-ordinal', // cobalt   — actif / courant
  'in-progress': 'narrative-encounter-duo-gagnant', // sarcelle — en cours
  premium: 'narrative-encounter-proie-favorite', // or       — premium / possede
  free: 'narrative-encounter-cross-game', // ardoise  — gratuit
  neutral: 'narrative-encounter-cross-game', // ardoise  — defaut
}

export function seasonPassBadgeToken(role: SeasonPassBadgeRole): SemanticToken {
  return ROLE_TOKEN[role]
}

/**
 * Statut de pass -> role de pastille. Le contrat OpenAPI expose `status` en
 * `string` : toute valeur hors des statuts connus retombe sur `neutral`.
 */
export function seasonPassStatusRole(status: string): SeasonPassBadgeRole {
  switch (status) {
    case 'active':
      return 'active'
    case 'completed':
      return 'completed'
    case 'in_progress':
      return 'in-progress'
    default:
      return 'neutral'
  }
}
