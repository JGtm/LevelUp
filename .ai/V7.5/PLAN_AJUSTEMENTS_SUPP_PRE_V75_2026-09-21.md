# Plan — Ajustements supplementaires pre-v7.5 (deuxieme vague, 2026-09-21)

Source : `C:\Users\Guillaume\Downloads\Ajustements-supp-prev75.txt` (retours utilisateur) + arbitrages
du 2026-09-21 (conversation). Branche de chantier : `feat/ajustements-supp-v75` (depuis
`origin/feat/v75` = 28b93cca5 + prop `frameless` de ChartCard b8166c9bf). Contrat : skill
`plan-execution`. Chaque lot = un worktree, un executeur Opus, une branche `feat/ajsup-<lot>`,
fusion par le pilote dans la branche de chantier. Statuts : `[x]` fait, `[~]` couvert ailleurs,
`[!]` non traite (justifie).

## Decisions utilisateur (tranchees, ne pas rouvrir)

- D1 « Prises nettes de drapeau » (Sessions) : retirer EXACTEMENT le texte cite (de « Prises nettes
  de drapeau » a « attribuees a un joueur ») ; le pilote lit : les phrases explicatives du bloc.
  Le titre de vue et la jauge du role « prendre » restent.
- D2 Tiers d'armes, partout : puissance en tete, puis terrain, puis un depliable ferme par defaut
  avec base. Les socles de BONUS (camouflage, surbouclier) sont des EQUIPEMENTS (decision du
  2026-09-21) : ils quittent la section armes, deja comptes dans « Usages d'equipement ». La ligne
  « non identifie » disparait des grilles (son compte passe dans l'infobulle (i) du titre).
- D3 Empaleur : ERREUR de classement constatee sur b1ad85eb (depart a loadout classique et egal
  pour tous) — corriger la cause AVANT tout reclassement.
- D4 « geste » -> « usages » partout dans l'UI.
- D5 « Les formes retenues » (Timeseries + Escouade) : supprimer le titre de section ET la rangee
  de tuiles de couverture ; les blocs Equipement / Armes speciales / Objectifs restent.
- D6 Sessions « Controle des armes speciales » : scinde en cartes dediees comme Synthese/Escouade.
- D7 Legende raye/plein : sur TOUTES les grilles de jauges et les deux pistes du lobby.
- D8 Etats vides : bloc dans une rangee = reste affiche avec un message nommant la cause (aucun
  film decode / aucun socle ni objectif dans les modes selectionnes / erreur de chargement) ;
  section entiere sans objet = masquee ; intertitre orphelin « Objectifs » corrige.
- D9 Colonne « Objets laches au sol » : detail par famille si le film le sert, sinon suppression.
- D10 Explorer « Il s'est passe quoi sur le bloc » : ignore.
- D11/D13/D14/D15 : MAQUETTE de 3 propositions chacune (pas de rond, pas de plat/mince) pour :
  Part des assistances, Repartition des resultats, Taux d'echange par session, Controle des armes
  speciales (Match view), Usages d'equipement (Match view : « par joueur » + « part de chaque
  equipe » cote a cote). Aucun code avant le verdict.
- D12 Titres Escouade : « Assistances croisees », « Delai de vengeance », « Frags non venges »,
  « Assistances par coequipier », « Part de mon camp », « Detail des prises d'armes speciales ».
- D16 Alignement comparaison : rangees partagees ; placeholder « Sans equivalent dans cette session ».

## Lots

### Lot A1 — chrome des graphes, Escouade, formes retenues (worktree `LevelUp-wt-ajsup-a1`) — FUSIONNE 57362a0d5
- [x] Six cartes de l'echange + donuts + tout ChartCard monte dans une SectionCard : `frameless`
- [x] Titres D12 (FR + EN)
- [x] Taux d'echange par session : rendu inchange (attend D13)
- [x] Frises : retirer « Affiches : les XX derniers matchs… » et « Hors de cette barre… »
- [x] Formes retenues (solo + squad) : titre de section + HeaderStrip retires (D5)
- [x] Notes sous legende des FormesCard -> InfoTooltip (i) a droite du titre de carte
- [x] Footers/notes des cartes de l'echange -> InfoTooltip (i)
- [x] Intertitre « Objectifs » orphelin corrige ; politique D8 sur les blocs des formes
- [x] Etiquettes blanches sur rayures (Piste100Form) — « Ce que mon camp prend du lobby »
- [x] « geste » -> « usages » dans `formes/cardsI18n.ts` et `formes/i18n.ts` ; « Bastion » -> « Bases »
- [x] Colonne « Objets laches au sol » (D9)

### Lot A2 — usage partage + Sessions (worktree `LevelUp-wt-ajsup-a2`)
- [ ] Sessions : retirer « Matchs mesures X/X », pied du controle des armes, texte D1
- [ ] Sessions : Regularite a droite de Parts et parites en pleine page ; legende unique
- [ ] Legende raye/plein : composant unique pose sur UsageGaugeGrid et UsageLobbyTrack (D7)
- [ ] Etiquettes blanches sur UsageLobbyTrack (verifier sur capture) ; epaisseurs homogenes
- [ ] Controle des armes speciales (Sessions) scinde (D6)
- [ ] Ordre des tiers D2 dans `usagePadTiersModel.ts` + depliable base + « non identifie » retire
- [ ] Centrage vertical des grilles dans EquipmentUsageSection (Timeseries/Escouade/Synthese)
- [ ] Donuts d'usage sortis de leur cadre (`frameless`)
- [ ] Notes de EquipmentUsageSection -> InfoTooltip (i)
- [ ] Etats vides D8 sur les blocs d'usage
- [ ] « Bastion(s) » -> « Bases » dans `usageI18n.ts`

### Lot B — comparaison Sessions alignee (worktree `LevelUp-wt-ajsup-b`) — FUSIONNE 0fb899baa
- [x] Rangees partagees gauche/droite, placeholder D16

### Lot C — Explorer, Relations, Synthese, Timeseries intensite (worktree `LevelUp-wt-ajsup-c`) — FUSIONNE 95065fee3
- [x] Top medailles : dans Profil de combat, suit le switch En direct / Local ; retire de la carte cible
- [x] Noyau dur : plus de repli, tout affiche
- [x] Rythme des rencontres : cellules plus hautes (hauteur locale, liseret partage intact)
- [x] Synthese : sous-titre « Frag le plus lointain… » retire
- [x] Intensite : sous-titre et titre de panneau « Joueur » retires

### Lot D — Match view (worktree `LevelUp-wt-ajsup-d`) — FUSIONNE a9733a985
- [x] Occupation du terrain : +20 %, zoom et deplacement (hooks du rejeu 2D)
- [x] Notes sous legende -> InfoTooltip (i)
- [x] Usages d'equipement : deux cartes sur la meme rangee (rendu inchange, attend D15)
- [x] Ordre des tiers D2 dans `weaponTier.ts` / MatchPadControlSection ; « geste » -> « usages »

### Lot E — Empaleur (worktree `LevelUp-wt-ajsup-e`)
- [ ] Diagnostic sur b1ad85eb : pourquoi Skewer = base ; cause prouvee sur pieces
- [ ] Correctif Go + jumeau TS + tests ; aucun reclassement heuristique

### Lot F — maquette (fichier `.ai/V7.5/MAQUETTE_RENDUS_AJSUP_2026-09-21.html`) — FUSIONNE bed1a2600
- [x] 5 sujets x 3 propositions, jetons de l'app, clair/sombre

## Decouvertes (hors perimetre, ne pas traiter)
- D : `PAD_TIER_ORDER` sert aussi au departage de `tierOfWeaponOf` (`padControlLogic.ts:236`) : ordre d affichage et ordre de departage a decoupler (lot G).
- A1 : double cadre aussi dans `compare/CompareWeaponsRange.tsx`, `match-view/MatchKillDistanceSection.tsx`, `tactical/TacticalCoordinationCard.tsx`.
- A1 : motif titleAdornment + InfoTooltip recopie dans ~8 fichiers -> helper canonique + garde-rail (lot G du chantier).
- A1 : `DroppedByFamily` existe en amont (`film/replay/usage_summary.go:99`, `analysis/sessionusage`) mais s arrete a `analysis/squadformes/formes.go:229` (scalaire `Dropped`) : exposer la ventilation rendrait la colonne par famille (lot G).
- A1 : etiquettes sur rayures en `text-foreground` (pas blanc) car la hachure adverse laisse voir le fond de carte -> verdict visuel utilisateur.
