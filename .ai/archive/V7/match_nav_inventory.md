# Inventaire des points d'entrée vers `/players/$playerSlug/matches/$matchId`

Produit pour Phase 2a (router state + sessionStorage) et Phase 2b (URL params + Q25 paramétrable) du rework header MatchView.

Méthode : `grep -rn "to: '/players/\$playerSlug/matches/\$matchId'" apps/web/src` + variantes string-template.

## Légende

- **Liste ordonnée disponible** : la page source dispose-t-elle d'une liste de `match_id` déjà triée DESC qu'on peut passer en `MatchNavContext.matchIds` ?
- **Filtres pertinents** : ce qui sera mis dans `MatchFilterSpec` en Phase 2b (`filterSpec`).
- **Source de la liste** : query/hook qui détient la liste.

## Tableau

| # | Page source | Composant + chemin | Filtres actifs | Liste ordonnée disponible ? | Source de la liste | Phase 2a `source` | Phase 2b `filterSpec` |
|---|---|---|---|---|---|---|---|
| 1 | Home — RecentMatches grid | [`HomePage.tsx:61`](apps/web/src/features/home/HomePage.tsx#L61) `goToMatch` | aucun | oui | `useHomePage().recent_matches[]` | `home_recent` | (vide — chronologie globale) |
| 2 | Home — FavoriteMatches grid | idem `HomePage.tsx` `goToMatch` (même fonction) | aucun (favoris seulement) | oui | `useHomePage().favorite_matches[]` | `home_favorites` | (vide ; `is_favorite=true` n'est pas un filtre temporel) |
| 3 | Match History — table principale | [`MatchHistoryTable.tsx:97`](apps/web/src/features/match-history/MatchHistoryTable.tsx#L97) `navigateToMatch` | playlist + mode + outcome + dates (Zustand `match-history`) | oui (page courante 50/100) | `useMatchHistory(playerSlug, filters)` | `history` | `playlist_name`, `mode_category`, `date_from/to`, `outcome` |
| 4 | Career — Top matches | [`CareerTopMatchesTable.tsx:45`](apps/web/src/features/career/CareerTopMatchesTable.tsx#L45) `goToMatch` | top score (pas de filtres user) | oui (top 10/20) | `useCareerPage().top_matches` | `history` (générique) | (vide) |
| 5 | Explorer | [`ExplorerPage.tsx:115`](apps/web/src/features/explorer/ExplorerPage.tsx#L115) `goToMatch` | filtres riches Explorer (mode/playlist/dates/outcome/badges) | oui (page courante) | `useExplorer(filters)` | `history` | `playlist_name`, `mode_category`, `date_from/to`, `outcome` |
| 6 | Squad — Match history table | [`SquadMatchHistoryTable.tsx:191`](apps/web/src/features/squad/SquadMatchHistoryTable.tsx#L191) row click + Enter | session squad, cascade, période (Zustand `squad`) | oui (page courante 20/page) | `useSquadMatches()` ou similaire | `session` | `session_id` (squad), `date_from/to` |
| 7 | Squad — Synergies history | [`SquadSynergyHistoryTable.tsx:80`](apps/web/src/features/squad/SquadSynergyHistoryTable.tsx#L80) row click + button | sélection coéquipiers (synergies) | oui | `useSquadSynergies()` | `session` (taggé synergies) | (squad-specific — pas de mapping naturel vers MatchFilterSpec ; passer `matchIds` seul) |
| 8 | Squad v2 — History table | [`squad/v2/components/HistoryTable.tsx:61`](apps/web/src/features/squad/v2/components/HistoryTable.tsx#L61) row click | filtres v2 (à clarifier — équiv squad ?) | oui | `useSquadV2()` | `session` | `session_id` |
| 9 | Synthesis — Highlights section | [`SynthesisHighlightsSection.tsx:28`](apps/web/src/features/synthesis/SynthesisHighlightsSection.tsx#L28) `<Link>` | aucun | oui (highlights de la synthèse) | `useSynthesis().highlights` | `home_recent` (générique narrative) | (vide) |
| 10 | MatchHeader.tsx — nav prev/next | [`MatchHeader.tsx:108`](apps/web/src/features/match-view/MatchHeader.tsx#L108) `goTo` | hérite du contexte courant | déjà depuis API/state | `useMatchNeighbors()` | (chaîné — propage le ctx courant si présent) | (idem) |

## Points sans liste ordonnée disponible

Ces points doivent rester en fallback global (sans `MatchNavContext`) :

- **Notifications** : un match isolé reçu par notification — pas de "liste". (À chercher si présent dans `apps/web/src/features/notifications/`).
- **Media — détail** : un match lié à un média isolé. (À chercher dans `apps/web/src/features/media/`).
- **Search** : si recherche par ID match. (Pas trouvé de UI dédiée à ce stade).

## Couches de propagation à ajouter (Phase 2a)

1. Tous les `navigate({ to: '/players/$playerSlug/matches/$matchId', ... })` → remplacer par `useNavigateToMatch(playerSlug)(matchId, ctx)` avec `ctx` rempli depuis le state local de la source.
2. Le composant `MatchHeader.tsx` (la nav prev/next interne) lit le ctx courant via `useRouterState({ select: s => s.location.state.matchNavContext })` — quand on clique prev/next, on passe `next/prev → matchId` au helper en re-passant le **même ctx** (chain).
3. La sortie de contexte (lien dans la barre nav) purge `sessionStorage:matchNav:${matchId}` et navigue avec `state: undefined` → fallback global Q25.

## Notes Phase 2b

- Pour `history` et `explorer`, l'état Zustand existant doit exposer un sélecteur dédié `getActiveFilterSpec()` qui produit `MatchFilterSpec` canonique. À implémenter dans la migration consommateurs (Étape 2b.9).
- Pour `session`, `session_id` est l'identifiant exposé par les hooks `useSquad*`. Pas de transformation nécessaire.
- Les sources `home_recent` / `home_favorites` / `home` ne portent pas de filtre temporel — `filterSpec` reste `undefined`, seul `matchIds` est utilisé.

## Validation finale

- [x] 10 points d'entrée identifiés dans 8 features (`home`, `match-history`, `career`, `explorer`, `squad` ×3, `synthesis`).
- [x] 9 sur 10 ont une liste ordonnée disponible (Phase 2a applicable).
- [x] 1 point (MatchHeader interne) chaîne le contexte hérité.
- [ ] Notifications + Media — à explorer en Phase 2b si on veut les compléter.

---

*Dernière mise à jour : 2026-05-05 — généré pendant Phase 2 du rework header MatchView.*
