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
 * LE BLOC NE PORTE QUE LE STATUT (retour utilisateur du 2026-08-28, 2e passe) : « je ne veux que
 * le statut de la partie dans un bloc de couleur ; le nom de l'équipe et le score restent juste
 * du texte affiché librement ». Le bloc a donc absorbé la TYPOGRAPHIE du verdict — un bloc et un
 * titre séparés inviteraient à y remettre une deuxième ligne, ce qui est précisément ce que le
 * retour retire. Ce qui l'accompagne (nom, score) vit désormais HORS de lui.
 *
 * SANS ACCENT LATÉRAL GAUCHE, et c'est un invariant du lot précédent : l'écran de fin portait
 * une barre verticale à gauche (`borderLeft`) que l'utilisateur ne veut plus (« faut le virer de
 * ce style »). Elle n'est NULLE PART. Les classes couleur restent interdites (color-tokens) — la
 * couleur d'équipe passe par un style inline résolu depuis un token, le neutre par les tokens du
 * thème.
 */

/**
 * LE BLOC DU STATUT : sa forme, ses marges et LA POLICE DU VERDICT — sans bord ni fond, qui
 * appartiennent à l'appelant (couleur d'équipe résolue, ou tokens du thème).
 */
export const OVERLAY_STATUS_BLOCK =
  'rounded-lg px-8 py-4 text-center text-2xl font-bold uppercase tracking-wide text-foreground shadow-lg backdrop-blur-sm'

/** Le bloc NEUTRE (égalité, message inter-manche) : bord et fond par tokens du thème. */
export const OVERLAY_STATUS_NEUTRAL = `border-2 border-border bg-card ${OVERLAY_STATUS_BLOCK}`
