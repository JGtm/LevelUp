/**
 * replayOverlayStyles.ts — L'HABILLAGE PARTAGÉ des panneaux d'overlay du rejeu : l'écran de fin
 * de match (`ReplayVictoryOverlay`) et le message inter-manche (`ReplayRoundBreakOverlay`).
 *
 * POURQUOI CENTRALISÉ (2026-08-28). Le message inter-manche devait adopter « le même style que
 * le texte de défaite ou victoire » (retour utilisateur) : la carte et le titre étaient sur le
 * point d'exister en TROIS exemplaires (le panneau d'équipe et le panneau neutre de l'écran de
 * fin, plus le message). À la 3e copie, la règle CLAUDE.md n°6 impose de centraliser ET de poser
 * un garde-rail — c'est ce fichier, plus `replayOverlayStyles.guard.test.ts`.
 *
 * SANS ACCENT LATÉRAL GAUCHE, ET C'EST LE POINT DE CE LOT. L'écran de fin portait une barre
 * verticale à gauche (`borderLeft`, l'accent d'identité d'équipe) que l'utilisateur ne veut plus
 * (« faut le virer de ce style »). Elle n'est donc NULLE PART : ni l'écran de fin ni le message
 * inter-manche ne la portent. Les classes couleur restent interdites (color-tokens) — l'identité
 * d'équipe passe par un style inline résolu (fond + trait), le neutre par les tokens du thème.
 */

/** Le corps du panneau : coins, marges, centrage, ombre, flou — SANS bord ni fond (au panneau). */
export const OVERLAY_PANEL_BODY = 'rounded-lg px-8 py-5 text-center shadow-lg backdrop-blur-sm'

/** Le panneau NEUTRE (égalité, message inter-manche) : bord et fond par tokens du thème. */
export const OVERLAY_NEUTRAL_PANEL = `border-2 border-border bg-card ${OVERLAY_PANEL_BODY}`

/** Le TITRE du panneau : la police du verdict de fin, reprise TELLE QUELLE par le message
 *  inter-manche (la demande de l'utilisateur). */
export const OVERLAY_TITLE = 'text-2xl font-bold uppercase tracking-wide text-foreground'
