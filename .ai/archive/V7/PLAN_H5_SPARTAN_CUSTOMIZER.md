# PLAN — Personnalisateur Spartan (nameplates + emblèmes) Halo 5

> Statut : Pré-P0 FAIT, P0 à venir. Date : 2026-06-28.
> Branche : `feat/h5-spartan-customizer`
> Worktree : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-h5-spartan-customizer`

## Objectif

Feature **Halo 5 uniquement** : depuis la bannière d'identité de la home, un clic ouvre une
**modale** (sans navigation) permettant de parcourir nameplates + emblèmes, choisir des couleurs
(rouge/bleu présélectionnés), voir le rendu **recolorisé en temps réel** (nameplate + emblème
côte à côte, non collés), puis Enregistrer / Annuler.

Critère de succès : modale opérationnelle, gated sur la capability `spartan_customizer`
(invisible sur Infinite), recolor live correct, persistance basique. Cible ~240px de hauteur,
proportions conservées.

## Spec utilisateur (condensée)

- Halo 5 only → capability / manifest.
- Clic bannière home → modale (pas de changement de page).
- Parcourir nameplates + emblèmes.
- Couleurs rouge/bleu présélectionnées (masques bruts illisibles sinon).
- Palette large mais pas pléthorique.
- Recolor temps réel, appliqué nameplate ET emblème.
- 2 images côte à côte, non collées.
- Enregistrer / Annuler (gestion basique).

## Système de couleurs (reverse-engineering)

- Chaque masque = canaux **R/G/B = zones coloriables** (Primary/Secondary/Tertiary), confirmé via
  le Lua compilé `nameplatemanager.lua` (moteur Anubis UI) + inspection des canaux.
- Modèle de recolor **additif** validé visuellement :
  `out = primary*R + secondary*G + tertiary*B`, `alpha = max(R,G,B)`.
  (Le mode « canal dominant » écrase le design → rejeté.)
- Emblème `detail` natif = 512×512 (assez pour 240px). Nameplate natif = 324×68 (→ upscale requis).
- Placeholders (ex : index ~200 = texture de texte debug) à exclure du catalogue.

## Ancrages codebase (exploration)

### Bannière + modale
- `apps/web/src/features/home/HomeSpartanIdentityBanner.tsx` (rendu dans `HomePage.tsx:231`), pas de onClick.
- Modales = maison (pas de Radix). Pattern : `useState` parent + rendu conditionnel + `role="dialog"`
  + Escape + backdrop. Réf : `features/palmares/BattlePassRewardLightbox.tsx`, `components/ui/alert-dialog.tsx`.
- Déclenchement : prop `onIdentityClick` sur la bannière → state dans `HomePage`.

### Gating par titre (miroir de `native_kill_mechanics`)
1. Go const : `apps/go-api/internal/domain/title/registry.go` → `CapSpartanCustomizer = "spartan_customizer"`.
2. Go validation : `config_loader.go` map `knownCapabilities`.
3. TOML : `config/titles/halo_5/title.toml` → ajout dans `capabilities` (PAS dans halo_infinite).
4. TS : `apps/web/src/lib/capabilities/capabilities.ts` → `TITLE_CAPABILITIES`.
5. UI : `useCapability('spartan_customizer')` ou `<FeatureGate capability="spartan_customizer">`.
6. Exposé via `/bootstrap` → `appShellStore.availableTitles` → hook.

### Persistance
- v1 = **localStorage** (précédent : `stores/settingsDraftStore.ts` LocalUiPrefs a déjà
  `allyTeamColor`/`enemyTeamColor`). Champ : `spartanAppearance { nameplateId, emblemId, primary, secondary, tertiary }`.
- Follow-up backend (cross-device) : table append-only `shared_social` (miroir `notification_preferences_history`
  + vue `_latest`), endpoints GET/PATCH `/api/v1/players/{slug}/spartan/preferences`.
  ATTENTION : écritures `shared_social` via SocialPersister + CHECKPOINT sous lease (ADR 0022).

### Assets
- v1 = **statiques same-origin** : `apps/web/public/titles/halo_5/spartan/{nameplates,emblems}/`
  (canvas lit les pixels sans CORS) + `spartan-catalog.json` (ids, placeholders exclus).
- Précédent : `apps/web/public/titles/halo_5/wallpaper_*.png`.

### Couleurs
- Palette curée (rouge/bleu + ~12-16 teintes) dans `apps/web/src/lib/accessibility/palettes/spartan.ts`
  (exception couleurs-données documentée, comme `rarity.ts`).
- Couleurs joueur = données arbitraires (hex) → exemptées. UI structurelle de la modale = tokens (`tokenCssVar`).

## Phases

### Pré-P0 — Upscale fiable par CLI (FAIT — 2026-06-28)

**Résultat** : 380 nameplates upscalés en `digital-art-4x` → 1296×272, 26 Mo, dans
`_upscayl_output/nameplates_digitalart_4x`. CLI `upscayl-bin.exe` fiable (RTX 4080), zéro plantage.
`digital-art-4x` < `standard-4x` pour les bavures inter-canaux. Commande :
`upscayl-bin.exe -i <in> -o <out> -n digital-art-4x -s 4 -m "C:\Program Files\Upscayl\resources\models" -f png`

- GUI Upscayl plante → piloter `C:\Program Files\Upscayl\resources\bin\upscayl-bin.exe` (realesrgan-ncnn-vulkan).
- Modèles : `C:\Program Files\Upscayl\resources\models` (digital-art-4x, high-fidelity-4x, remacri-4x,
  ultramix-balanced-4x, ultrasharp-4x, upscayl-lite-4x, upscayl-standard-4x).
- Batch : `upscayl-bin.exe -i <in> -o <out> -n digital-art-4x -s 4 -m <models> -f png`.
- `digital-art-4x` retenu pour limiter les bavures (graphismes plats). Si bavure aux frontières de canaux →
  fallback upscale **canal par canal** (R/G/B en grayscale séparés puis recombinaison).
- TIF inutile (on travaille en PNG). Cible : masques nameplate 324×68 → 1296×272. Emblèmes 512² = pas besoin.

### P0 — Assets + index
Masques PNG (nameplates upscalés + emblèmes natifs) → `apps/web/public/titles/halo_5/spartan/`
+ `spartan-catalog.json` (placeholders exclus).

### P1 — Capability `spartan_customizer` (multi-titre)
Les 5 étapes ci-dessus. Test : absente sur Infinite.

### P2 — Cœur recolor (pur, testé)
`recolorMask(imageData, {primary,secondary,tertiary})` (additif) + palette + presets rouge/bleu.
Tests vitest (hors sandbox).

### P3 — Modale UI
`onIdentityClick` (gated) → `SpartanCustomizerModal` : browser, color-pickers, recolor canvas live
nameplate + emblème côte à côte, Save/Cancel → store localStorage. i18n EN+FR. Tokens pour l'UI.

### P4 — Tests + clôture
vitest (recolor + smoke + gating masqué sur Infinite), typecheck/eslint, entrée `thought_log.md`,
delivery-checklist. Aucun commit sans feu vert utilisateur.

## Décisions

- Persistance v1 = localStorage (backend = follow-up).
- Assets statiques `public/` (same-origin canvas).
- Recolor = canvas 2D.
- 1 sélection = 1 emblème → nameplate + emblème côte à côte.
- Capability = `spartan_customizer`.
- Upscale via `upscayl-bin.exe`, modèle `digital-art-4x`.

## Risques / points ouverts

- Bavure de canaux à l'upscale → fallback canal-par-canal.
- Emplacement worktree : créé en sibling `LevelUp-h5-spartan-customizer` (convention repo =
  `.claude/worktrees/`) — relocalisable.
- Taille assets `public/` (~14 Mo) — acceptable ; sinon servir via static Go.
- Persistance localStorage = pas cross-device (assumé en v1).

## État des assets (Desktop, hors repo)

- `Halo5_Assets_Wiki/_app_assets/nameplates/` : 380 masques natifs (PNG, `{pc}` retiré).
- `Halo5_Assets_Wiki/_upscayl_input/nameplate_raw/` : 380 masques natifs.
- `Halo5_Assets_Wiki/_upscayl_output/...` : 182 nameplates upscalés (standard-4x, GUI planté).
- `Halo5_Assets_Wiki/_RESTE_A_UPSCALER/` : 197 masques restants à upscaler.
- Emblèmes : `Halo5_Assets_Wiki/__chore/pc__/ui/wpf/anubisui/images/emblem/{detail,full,nameplate}/`.
