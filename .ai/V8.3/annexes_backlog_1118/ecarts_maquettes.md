# Écarts avec les maquettes — instruction Q.1 (2026-11-18)

Instruction en lecture seule des onze points de l'entrée « Écarts avec les maquettes à juger à l'écran »
de `.ai/BACKLOG.md`, contre les maquettes de `.ai/V8.3/annexes_refonte_ui_rejeu/maquettes/` (HTML lus ;
aucune image : le dossier ne contient que des fichiers `.dc.html` et `canvas.json`, rendus non regardés),
la grille `conformite_maquettes.md`, l'audit `audit_conception.md`, la revue finale
`revue_finale_2026-11-18.md` et le plan `.ai/V8.3/PLAN_REFONTE_UI_REJEU_2026-11-15.md` (décisions D-1 à D-24,
section « 1 quater », comptes rendus d'items). Code lu dans `C:/t/bl18-d`, dernier commit `14e69f8124`.
Dans ce fichier `MR/` désigne `apps/web/src/features/match-replay/`.

Règle appliquée : hors des écarts voulus V1 à V17 et des décisions datées du user, la maquette fait foi.
Un compte rendu d'agent ou du pilote qui écrit « écart accepté et dit au user » n'est PAS une décision du
user (mémoire « remarque n'est pas validation ») : aucun de ces écarts n'est traité ici comme décidé.

Les points 1, 2, 4 se découpent en sous-points parce que leurs éléments n'ont pas le même sort.

## 1. Tableau récapitulatif

| Point | Sujet | Classement | Effort |
|---|---|---|---|
| 1 | Pictogrammes et marquage actif des icônes de cadre | SANS DESSIN, AU USER (la forme du bouton suit 2a) | S si accord |
| 2a | Boutons ronds de la barre d'une ligne (et leur taille) | ÉCART À CORRIGER | M |
| 2b | Curseur de volume gardé dans la barre d'une ligne | SANS DESSIN, AU USER | S |
| 2c | Cartouche de capture et d'export | ÉCART À CORRIGER | S |
| 3 | Pas de croix de déplacement dans la barre d'une ligne | CONFORME À UNE DÉCISION DATÉE | rien |
| 4a | Fenêtre de fiche : 8 px sous la fiche au lieu de 8 px sous le haut du plan | ÉCART À CORRIGER | S |
| 4b | Fenêtre de fiche : seul le bas défile | ÉCART À CORRIGER | S |
| 5 | Bloc arrondi commun carte + colonne du Standard | ÉCART À CORRIGER | M |
| 6 | Infobulle de « Plan tourné » sans le gain en pour-cent | ÉCART À CORRIGER | M |
| 7 | Ligne choisie du fil marquée par un fond | CONFORME À UNE DÉCISION DATÉE (le liseré gauche) | rien |
| 8 | Tirs à la tête écrits « 3 · sur 5 mesurées » | ÉCART À CORRIGER | S |
| 9 | F / M / A de la tête de fiche sans couleur | CONFORME À UNE DÉCISION DATÉE | rien |
| 10 | Fond d'accent du bouton caméra quand un joueur est suivi | SANS DESSIN, AU USER | S |
| 11 | Plein cadre : la rangée est à 12 px du bord, la maquette à 16 px | ÉCART À CORRIGER | S |

Bilan : 8 écarts à corriger (2a, 2c, 4a, 4b, 5, 6, 8, 11 : 8 items de Q.2), 3 points sans dessin à poser au
user (1, 2b, 10), 3 points conformes à une décision (3, 7, 9).

## 2. Fiches

### Point 1 — icônes de cadre dessinées sans maquette

(a) Maquette. Aucune maquette ne dessine les icônes de cadre : K (`CinemaRangee.dc.html`), L, D
(`PleinCadre.dc.html`), H montrent les cadres en SEGMENT DE TEXTE dans l'en-tête (`.seg`, bouton actif
`.on{background:#2a2a2a;color:#fafafa}` ; dans D les capsules de touche « T » et « F » en clair, classe `.kbd`).
Ce segment est l'écart voulu V1. Ce que les maquettes dessinent à droite de la barre est le seul « Plein écran
du système » : `.ibtn`, 30 x 30 px, rayon 8 px, sans fond ni bordure, pictogramme de coins de 17 px.

(b) Code. `MR/ui/ReplayFrameButtons.tsx:27-31` (trois pictogrammes de cadre : fenêtre à bandeau, écran large, cadre
à zone teintée, dans `MR/ui/ReplayTransportIcons.tsx:153-207`), `:33-40` (bouton rond `h-8 w-8 rounded-full`,
état actif `bg-accent text-accent-foreground`, `aria-pressed`), infobulles « Cinéma (T) », « Plein cadre (F) »
(`MR/i18n/frames.i18n.ts:43-58`).

(c) Décision. D-9 (user, 2026-11-15) : « Les cadres s'affichent en icônes, pas en texte. » D-10 (validée par le
user le 2026-11-15, « réponse point par point ») : « À droite de la barre de lecture, dans les trois cadres :
Cinéma (bascule avec Standard), Plein cadre, plein écran système. » Grille, « Sans dessin » n° 1 : « Forme
proposée : trois boutons ronds fantômes au format du bouton plein écran actuel de la barre, 30 px dans la barre
compacte ; cadre actif sur fond d'accent, état porté par `aria-pressed` [...] Pas de choix à montrer. »
Plan E.2 : « icônes de cadre SANS maquette, formes de l'agent, à montrer au user ». Le user a décidé le principe
(icônes) et l'emplacement ; il n'a jamais vu ni validé les pictogrammes ni le marquage actif. La grille elle-même
proposait 30 px ; le code fait 32 px.

(d) Classement : SANS DESSIN, AU USER. Deux choses distinctes :
- la FORME du bouton (30 px, rayon 8) n'est pas une question : le « Plein écran du système » est dessiné en
  `.ibtn`, et les trois icônes de cadre rejoignent la même barre ; elle est traitée avec 2a ;
- les pictogrammes eux-mêmes et le marquage du cadre actif : question au user (section 4, question 1).

### Point 2 — boutons ronds, curseur de volume, cartouche de capture gardés dans la barre d'une ligne

Contexte commun. Plan E.3 : « ÉCARTS avec K à montrer au user : boutons ronds de la barre complète réutilisés,
curseur de volume gardé, capture et export dans leur cartouche, pas de croix de déplacement dans la barre d'une
ligne » : ce sont des écarts relevés par l'agent, jamais soumis au user. La décision G0-4 (« Tout reprendre »)
ne porte que sur trois commandes ajoutées (lecture automatique, « Recommencer », repères de médias), pas sur la
forme des boutons ni sur le volume. D-8 (le Standard garde sa barre) ne concerne pas la barre d'une ligne.

#### 2a — boutons ronds

(a) Maquette. K-10 (`CinemaRangee.dc.html`, classes `.ibtn`, `.play`, `.tp`), identique dans L, D, H : tout est
`.ibtn` (minimum 30 x 30, rayon 8 px, sans fond ni bordure, texte « −10 », « +10 », « 1× » en 12,5 px de poids
500) SAUF la lecture (`.play` : rond plein de 36 px). Caméra, son, −10, +10, vitesse, zoom − et +, capture,
export, plein écran : tous en `.ibtn`. Grille K-10 : « lecture (rond plein de 36 px), −10, +10, [...] vitesse ;
zoom − et + ; capture, export, plein écran système ; séparateurs verticaux entre groupes ».

(b) Code. Tout est `rounded-full` : caméra 34 px (`MR/ui/ReplayCameraMenu.tsx:196-200`,
`MR/model/cotesMenuCamera.ts:22`) ; son 34 px (`MR/sound/ReplaySoundControls.tsx:38`) ; −10 et +10 bordés
(`MR/ui/ReplayRoundButton.tsx:26-30`, `bordered`, appelés `MR/ui/ReplayTransportCompact.tsx:87-92`) ; zoom
rond fantôme (`:110-115`) ; vitesse en pilule bordée de 32 px (`MR/ui/ReplaySpeedMenu.tsx:45`) ; lecture
automatique en pilule (`MR/ui/ReplayAutoPlayToggle.tsx:25`) ; icônes de cadre 32 px
(`MR/ui/ReplayFrameButtons.tsx:39`). Seul le bouton de lecture (36 px rond plein) est conforme.

(c) Décision. Aucune. V1 à V17 ne couvrent pas la forme des boutons ; G0-4 ne l'évoque pas ; l'écart est
signalé « à montrer au user » (E.3) sans réponse.

(d) Classement : ÉCART À CORRIGER. Cote attendue : un format unique `.ibtn` de 30 x 30 px, rayon 8 px, sans
bordure ni fond au repos (fond d'accent au survol), glyphe de 17 à 18 px, textes −10 / +10 / vitesse en 12,5 px
poids 500, lecture inchangée (36 px rond plein). Les cotes entrent dans `MR/model/cotesBarreCompacte.ts`
(`boutonPx: 30`, `boutonRayonPx: 8`), le menu caméra passe de 34 à 30 px. La barre complète du Standard garde
sa forme (D-8) : chaque composant partagé reçoit une variante « compacte », pas une modification globale.
Fichiers : `MR/ui/ReplayRoundButton.tsx`, `MR/ui/ReplayCameraMenu.tsx`, `MR/model/cotesMenuCamera.ts`,
`MR/sound/ReplaySoundControls.tsx`, `MR/ui/ReplaySpeedMenu.tsx`, `MR/ui/ReplayAutoPlayToggle.tsx`,
`MR/ui/ReplayFrameButtons.tsx`, `MR/ui/ReplayTransportCompact.tsx`, `MR/model/cotesBarreCompacte.ts` et leurs
tests. Effort M (neuf composants, une variante chacun, aucun comportement touché).

#### 2b — curseur de volume

(a) Maquette. K-10 : « caméra, son » ; le son est UNE icône (`aria-label="Son"`, haut-parleur de 17 px) ; aucun
curseur dans K, L, D ni H.

(b) Code. `MR/sound/ReplaySoundControls.tsx:36-66` : bouton de 34 px puis curseur de 58 px
(`h-3 w-[58px]`), partagé avec la barre complète (`MR/ui/ReplayTransport.tsx:59`) ; une demande du user du
2026-08-25 (curseur qui ne s'escamote pas quand le son est coupé) vaut pour la barre du Standard.

(c) Décision. Aucune pour la barre d'une ligne. G0-4 (« mêmes commandes dans les trois cadres ») peut se lire
comme « le volume aussi », mais la liste de G0-4 ne le nomme pas et le volume n'est pas une commande que K
omet dans l'audit (question 7 de l'audit ne le cite pas). Ambigu.

(d) Classement : SANS DESSIN, AU USER (question 2). Retirer le curseur supprime le seul moyen de régler le
volume dans le Cinéma et le Plein cadre ; le garder coûte 66 px sur une barre que le plan sait déjà étroite
(piste M, planche « barre sur écran étroit »).

#### 2c — cartouche de capture et d'export

(a) Maquette. K-10 : « capture, export, plein écran système » ; deux `.ibtn` ordinaires (appareil photo,
flèche d'export), sans cartouche, sans fond plein, sans couleur propre.

(b) Code. `MR/ui/ReplayCaptureButtons.tsx:39` : cartouche `rounded-full border bg-muted/40 p-1` ; capture en
`bg-secondary` (`:45`) ; export en `bg-primary` plein (`:56`), qui s'élargit pour porter le pourcentage
pendant l'export (`:15-18`, `:60`) ; enregistrement rouge de repli (`:69-80`) ; panneau d'export ancré sur le
cartouche (`:36-37`, `:83-85`). Appelé par la barre d'une ligne (`MR/ui/ReplayTransportCompact.tsx:118`).

(c) Décision. Aucune pour la barre d'une ligne. L'habillage en cartouche vient d'une planche de la barre
complète (2026-08-28) ; D-8 le garde pour le Standard, pas pour K.

(d) Classement : ÉCART À CORRIGER. Dans la barre d'une ligne : deux boutons au format `.ibtn` de 2a, glyphes
inchangés, sans cartouche ni fond plein. À garder car la maquette est statique et ne dit rien du calcul :
le pourcentage affiché à côté de l'icône pendant l'export, le panneau d'export, la variante « enregistrement »
quand l'export manque. Fichiers : `MR/ui/ReplayCaptureButtons.tsx` (variante compacte),
`MR/ui/ReplayTransportCompact.tsx`. Effort S.

### Point 3 — pas de croix de déplacement dans la barre d'une ligne

(a) Maquette. K-10 (et L, D, H) : « zoom − et + » : deux `.ibtn` « Dézoomer » et « Zoomer », sans croix, sans
niveau, sans retour à 1x. Aucune maquette ne dessine de croix de déplacement dans aucun cadre ; en Standard la
commande de zoom actuelle est conservée par D-8 (V5), donc la croix y reste.

(b) Code. `MR/ui/ReplayTransportCompact.tsx:109-116` : zoom − et + seulement ;
`MR/ui/frames/replayFrameSlots.tsx:28-37` : la croix flottante (`ReplayZoomControl`, `MR/ui/ReplayZoomControl.tsx`)
n'est montée que dans la barre complète. Le déplacement reste possible au glisser (`MR/hooks/useReplayDrag.ts`,
branché `MR/ui/ReplayCanvas.tsx:342`), à la molette (`:341`) et par Maj + flèches
(`MR/hooks/useReplayShortcuts.ts:176-186`) ; le retour à la carte entière reste la touche 0.

(c) Décision datée. D-2 (user, 2026-11-15) : « Cinéma = tableau K, Plein cadre = tableau D (structure de K sans
l'app autour). Message du user : « K à la place de C pour le mode cinéma ». » Titre du canevas
(`canvas.json`) : « K · Cinéma (validé) ». Grille K-10 : « zoom − et + ». Le code reproduit la maquette validée.

(d) Classement : CONFORME À UNE DÉCISION DATÉE (D-2 et K validé). Rien à faire. Remarque pour le user, pas un
écart : le déplacement n'a plus de bouton dans le Cinéma et le Plein cadre, seulement le glisser, la molette et
le clavier.

### Point 4 — fenêtre de fiche accrochée 8 px sous la fiche, dont seul le bas défile

#### 4a — position verticale

(a) Maquette. L (`CinemaRangeeFiche.dc.html`, classe `.pop`) : `position:absolute; top:8px; left:140px;
width:330px; max-height:calc(100% - 16px)`, dans le bloc du plan (`.vp`, bordure d'1 px), lui-même à 8 px sous la
rangée (`.band{margin-bottom:8px}`). La fenêtre commence donc à 8 px sous le HAUT DU PLAN, soit environ 17 px
sous la fiche ; son bord gauche est aligné sur celui de la fiche (140 px dans le bloc = 164 px de page = bord de
la deuxième fiche). Grille L-1 : « bord GAUCHE aligné sur celui de la tuile, 8 px sous le haut du plan, hauteur
au plus celle du plan moins 16 px ».

(b) Code. `MR/settings/useAnchoredPanel.ts:41` (`GAP = 8`) et `:104` (`top: r.bottom + GAP`) : 8 px sous la FICHE ;
or la rangée est à 8 px du plan (`MR/ui/frames/ReplayFrameLayout.tsx:53`, `space-y-2`), donc la fenêtre commence
à la hauteur du bord haut du plan, environ 9 px plus haut que la maquette. Alignement gauche et borne basse
(`:105`, `boundBottom`) conformes.

(c) Décision. Aucune. Plan D.3 : « ÉCARTS avec L, acceptés et dits au user : fenêtre 8 px sous la fiche, tête
et actions fixes avec défilement du bas seulement, ombre et fond par jetons » : compte rendu du pilote, pas une
réponse du user. Les jetons pour l'ombre et le fond ne sont pas des écarts (règle des couleurs).

(d) Classement : ÉCART À CORRIGER. Cote : haut de la fenêtre = haut du bloc du plan + 8 px (bordure comprise),
bas inchangé (bas du plan - 8 px). Fichiers : `MR/settings/useAnchoredPanel.ts` (une option « haut du
plan »), `MR/ui/ReplayPlayerPopover.tsx`, `MR/model/cotesFenetreFiche.ts` (cote nommée), `MR/ui/ReplayCanvas.tsx:583`
(un attribut qui désigne le bloc du plan : fichier à un seul écrivain) et leurs tests. Effort S.

#### 4b — défilement

(a) Maquette. L : `.pop{display:flex;flex-direction:column;gap:12px;padding:12px 14px;overflow-y:auto;
scrollbar-width:thin}` : c'est la fenêtre ENTIÈRE qui défile, identité et actions (`.who`, `.acts`) comprises
dans la zone qui défile. Grille L-1 : « défilement interne ». Même principe en J pour le Standard (J-1 : « retour,
sélecteur, identité et actions sont DANS la zone qui défile »), que D.2 a réalisé (`scroll: 'outer'`). Le
contenu dessiné tient sans défiler : le comportement ne se lit que dans le CSS.

(b) Code. `MR/ui/ReplayPlayerPopover.tsx:126` (`overflow-hidden`) et `:137` (`scroll="internal"`) :
identité et trois actions restent en place, seule la région basse défile
(`MR/ui/ReplayPlayerPanelBody.tsx:91-94`, `:140`). L'option `scroll="outer"` existe déjà.

(c) Décision. Aucune du user (voir 4a ; D.1 : « à trancher en D.2 », tranché pour J seulement).

(d) Classement : ÉCART À CORRIGER. Cote : la fenêtre porte `overflow-y: auto` avec une barre fine, le corps passe
en `scroll="outer"`. Fichiers : `MR/ui/ReplayPlayerPopover.tsx`, `MR/model/cotesFenetreFiche.ts` (la marge de
barre `margeBarrePx` ne s'applique plus au corps seul) et le test de la fenêtre. Effort S. À savoir : c'est le
seul point où le code est ergonomiquement plus confortable que la maquette (les trois actions restent
atteignables sur une fiche longue) ; le pilote peut le dire au user au moment de la validation, sans changer le
classement.

### Point 5 — bloc arrondi commun à la carte et à la colonne du Standard non fait

(a) Maquette. A (`Main.dc.html`) : `.vp{height:600px;display:grid;grid-template-columns:minmax(0,1fr) 360px;
border-radius:12px;border:1px solid #1f1f1f;background:#0f0f0f;overflow:hidden}` contient `.stage` (la carte) ET
`.col` (la colonne, `border-left:1px solid #1c1c1c`) : UN seul bloc arrondi ; la barre de lecture `.tp` est
HORS du bloc, dessous. J (`FicheJoueur.dc.html`) et H (`RoninCinema.dc.html`, `.vp` identique) montrent le même
bloc. Grille A-5 : « bloc arrondi (rayon 12 px, bordure) ; grille carte puis colonne de 360 px ».

(b) Code. Deux objets distincts : la toile a son bloc (`MR/ui/ReplayCanvas.tsx:583`, `rounded-lg border
bg-card`, qui contient la toile ET la barre), à côté d'une grille à `gap-3`
(`MR/ui/frames/ReplayFrameLayout.tsx:70`) ; la colonne n'a qu'un filet gauche (`MR/ui/frames/ReplayStandardFrame.tsx:62`, `xl:border-l`), sans
bordure ni fond ni rayon ni liaison avec le bloc de la toile.

(c) Décision. Plan S.4 : « le bloc arrondi commun carte et colonne de A-5 n'est PAS fait, la toile gardant son
propre bloc (V5) : à juger à l'écran » : motif du pilote. V5 (D-8, D-18 ; user, 2026-11-15 : « pas besoin de
revoir le lecteur et les options de zoom qui sont bien positionnés » ; « les contrôles ne changeaient pas [...]
le reste c'est repris de la maquette ») ne garde que la zone de lecture et le zoom. Le bloc commun relève de
« le reste ». Grille A-5, statut : la barre hors du bloc « ne concerne donc que le Cinéma et le Plein cadre ».

(d) Classement : ÉCART À CORRIGER. Lecture qui respecte A-5 ET D-8 : un seul bloc `rounded-xl` (12 px) bordé,
fond de jeton, qui contient à gauche la toile et sa barre de lecture TELLES QU'ELLES SONT (le lecteur n'est pas
déplacé : D-8), à droite la colonne de 360 px avec son filet gauche, sur toute la hauteur du bloc ; la toile perd
son cadre propre en Standard. Fichiers : `MR/ui/frames/ReplayFrameLayout.tsx` (le bloc), `MR/ui/ReplayCanvas.tsx`
(le cadre propre devient une option : le Cinéma et le Plein cadre le gardent, K-8), `MR/ui/frames/ReplayStandardFrame.tsx`,
`MR/ui/rosterHeight.guard.test.ts` (la colonne reste dimensionnée par la carte) ; fichier partagé à un seul
écrivain. Effort M. Réserve à poser au user à la validation : la barre étant dans le bloc, la colonne descend
jusqu'à son bas (A dessine la barre dessous, hors du bloc ; la déplacer toucherait D-8).

### Point 6 — infobulle de « Plan tourné » sans le gain en pour-cent

(a) Maquette. H (`RoninCinema.dc.html:207`) : `title="Plan tourné automatiquement : +42 % de taille. Cliquer
pour revenir à l'orientation d'origine."`. Grille H-3 reprend le texte. D-7 (user, 2026-11-15) : « Le plan tourne
de 90° quand il y gagne au moins 25 % de surface dans le cadre ; bouton « Plan tourné » pour revenir ».

(b) Code. `MR/i18n/frames.i18n.ts:64` (fr) et `:80` (en) : « Plan tourné automatiquement pour mieux remplir la
zone. Cliquer pour revenir à l'orientation d'origine. », posé en `title` par `MR/ui/ReplayRotatedButton.tsx:46`.
Aucun pourcentage. La donnée existe : `autoTurns` calcule `gain = tourne / droit - 1`
(`apps/web/src/lib/replay/orientation.ts:236-245`) ; il n'est pas exposé.

(c) Décision. Aucune. Plan F.2 : « l'infobulle n'écrit pas le gain en pour-cent de la maquette H, la règle pure
ne le rendant pas » : motif d'implémentation, pas une décision du user. Le gain est calculable avec les mêmes
entrées que la règle.

(d) Classement : ÉCART À CORRIGER. Texte attendu : « Plan tourné automatiquement : +N % de taille. Cliquer pour
revenir à l'orientation d'origine. » (en : « Map rotated automatically: +N % in size. Click to return to the
original orientation. »). N = gain de SURFACE arrondi (la même grandeur que le seuil de D-7 ; la maquette écrit
« de taille » sans dire surface ou longueur : voir « non établi »). Fichiers : `apps/web/src/lib/replay/orientation.ts`
(exposer le gain), `MR/hooks/useResolvedTurns.ts` et `MR/hooks/useReplayView.ts` (le porter), `MR/ui/replayTransportCommands.ts:64,120`
(`autoRotation.gainPct`), `MR/ui/ReplayRotatedButton.tsx`, `MR/i18n/frames.i18n.ts` (texte à variable, fr et en)
et leurs tests. Effort M (cinq sauts de transport, deux langues, aucune règle neuve).

### Point 7 — ligne choisie du fil marquée par un fond

(a) Maquette. Aucune maquette ne dessine une ligne choisie (A, J, K montrent des lignes de fil sans état). Grille,
« Sans dessin » n° 15 : « Forme proposée : la marque actuelle (fond de jeton et deux filets aux couleurs des
deux camps), sur la ligne à colonnes fixes ».

(b) Code. `MR/ui/ReplayFeedLines.tsx:339-346` : `bg-muted` + `data-selected` + `aria-pressed`. L'ancienne marque
(`git show 31e2ac60b1^:.../ReplayFeedLines.tsx`) ajoutait `box-shadow: inset 3px 0 <couleur du tueur>, inset -3px 0
<couleur de la victime>` : un liseré coloré de 3 px sur le bord GAUCHE et un sur le bord droit.

(c) Décision datée. D-5 (user, 2026-11-15) : le fond teinté se met « à la place de l'ancien liseré gauche refusé
par le user ». Message du user du 2026-11-17 (mémoire `feedback_pas_de_bordure_coloree_a_gauche`) : « je t'ai dit
que j'aimais pas les bordure colorée sur la gauche ». Plan S.3 : « la ligne choisie se lit par un fond, ses
liserés de camp sont retirés ». Le liseré gauche est donc proscrit par le user ; la grille proposait de le
garder (item 15) et c'est la grille qui est périmée sur ce point.

(d) Classement : CONFORME À UNE DÉCISION DATÉE pour le retrait du liseré gauche et le fond. Rien à corriger.
Limite : le liseré DROIT a été retiré par la même occasion, sans décision ni dessin ; sa disparition ne retire
aucune information (la marque reste le fond et `aria-pressed`). Voir « Découvertes » n° 1 : la teinte D-5 des
lignes du joueur suivi n'existe pas, ce qui change le contraste attendu entre ligne choisie et ligne suivie.

### Point 8 — tirs à la tête écrits « 3 · sur 5 mesurées »

(a) Maquette. M (`FicheInstant.dc.html`, section « Depuis le début du match », liste libellé / valeur) :
« Tirs à la tête | 3 », valeur seule ; seul « Score » porte une note (`<small>` « 800 · 2e sur 8 »). Grille M-2
(validée par le user le 2026-11-17, « Forme 2, Gestes ok, armes de lourdes ») : « FDA, série en cours, meilleure
série, tirs à la tête, éliminations parfaites [...] score et rang (« 800 · 2e sur 8 ») » : seul le score porte une
mention. M-11 : « aucun texte de remplacement ».

(b) Code. `MR/ui/fiche/FicheDepuisDebut.tsx:28-37` : valeur puis `<Note>` « · sur 5 mesurées »
(`MR/i18n/playerInstant.i18n.ts:112`, en `:158`, type `:47`). Le nombre est celui des éliminations dont le coup
fatal est mesuré (`MR/model/playerInstantStats.ts:60-64`, `MR/model/playerInstantView.ts:138`).

(c) Décision. Aucune pour la mention. D-19 (user, 2026-11-16) : « tirs à la tête (quand le coup fatal est
mesuré) » : la condition d'existence de la ligne, pas une note. Plan T.5 : « ÉCARTS dits au user : tirs à la tête
écrits « 3 · sur 5 mesurées » » (pas une réponse). Règle du user (message du 2026-10-09, mémoire
`feedback_bots_comptent_jamais_non_mesures`) : « on n'indique pas "X non mesurés" quoi qu'il arrive, niveau UX
c'est trop troublant et induit un manque de confiance » ; « un « N sur M » prend pour base les frags officiels ».
« sur 5 mesurées » est un compteur de couverture de la mesure, de la même nature.

(d) Classement : ÉCART À CORRIGER. Cote : « Tirs à la tête | 3 », rien d'autre ; la ligne reste absente quand
aucune élimination n'est mesurée (condition actuelle `measured > 0`, M-4 / M-11). Fichiers :
`MR/ui/fiche/FicheDepuisDebut.tsx`, `MR/i18n/playerInstant.i18n.ts` (retirer `headshotsOfFmt` des deux langues et du
type : texte sans lecteur, règle 7), `MR/model/playerInstantView.ts` et `MR/model/playerInstantStats.ts` (garder
`measured` seulement s'il sert encore à décider de la présence de la ligne) et leurs tests. Effort S.

### Point 9 — F / M / A de la tête de fiche sans couleur

(a) Maquette. J (`FicheJoueur.dc.html:64-65`), L (`.who .kda`), M (`FicheInstant.dc.html:40-41`) :
`.who .kda{margin-left:auto;font-size:15px;font-weight:600}`, chiffres à la couleur de texte du thème, seuls les
séparateurs « / » sont estompés (`color:#5c5c5c`, ou `#7d7d7d` dans M). Aucune couleur par valeur. Dans K la
tuile `.kda` est aussi neutre. Grille J-4 : « pastille, nom (17 px, gras), camp, F / M / A (15 px) » ; L-2 :
« pastille, nom (16 px, gras), camp, F / M / A, capsule « Échap », croix ».

(b) Code. `MR/ui/fiche/FicheIdentite.tsx:48-62` : `font-semibold tabular-nums text-foreground`, 15 px
(`COTES_FICHE_INSTANT.identite.triplePx`), séparateurs `text-muted-foreground`.

(c) Décision datée. D-6 (user, 2026-11-15) : « Standard = tableau J (message du user du 2026-11-15 :
« j'avais validé la vue J hors mode cinéma et plein écran ») » et « Cinéma et Plein cadre = tableau L ».
Validation du tableau M (user, 2026-11-17, « Forme 2, Gestes ok, armes de lourdes ») : « Tableau M validé [...]
il fait foi pour le contenu de la fiche, en colonne (J) et en fenêtre (L) ». Titres du canevas : J « (validée) »,
L « (validée) ».

(d) Classement : CONFORME À UNE DÉCISION DATÉE (D-6 et validation de M). Rien à faire. Le code reproduit les
maquettes validées au pixel près (taille, graisse, séparateurs). Si le user veut une couleur par valeur, c'est
une demande nouvelle, pas un écart.

### Point 10 — fond d'accent du bouton caméra quand un joueur est suivi

(a) Maquette. K, L, D, H : bouton caméra en `.ibtn` ordinaire, `aria-label` et `title` « Caméra : suivre JGtm »
(JGtm étant le joueur suivi dans ces écrans). Aucun état visuel du bouton n'est dessiné (ni fond, ni bordure,
ni pastille) ; le suivi se lit sur la fiche du joueur suivi (K-5 : bordure à la couleur de suivi, nom en gras).
Le menu à deux listes qu'il ouvre n'est dessiné nulle part (grille, « Sans dessin » n° 4).

(b) Code. `MR/ui/ReplayCameraMenu.tsx:196-201` : `bg-accent text-accent-foreground` tant que `followedName !== null`
(décrit `:22-23`), nom du bouton « Caméra : suivre <nom> » (`:182-183`) conforme au texte de la maquette.

(c) Décision. Plan E.4 : « fond d'accent du bouton quand un joueur est suivi, non dessiné : à montrer au user ».
Menu caméra : « Tranché par le pilote [...] dit au user le 2026-11-17 » (pas une réponse). G0-2 (user, 2026-11-17) :
« « suivi » = le joueur regardé [...] « Suivre » reste l'action de caméra, à part, éteinte au départ comme
aujourd'hui » : la caméra peut donc suivre quelqu'un sans que ce soit le joueur de la page, et le bouton est le
seul témoin de cet état dans la barre.

(d) Classement : SANS DESSIN, AU USER (question 3). Rien n'est dessiné ; le choix (fond d'accent contre bouton
inchangé) est un choix de lisibilité de l'état, pas une conformité.

### Point 11 — au Plein cadre, la rangée va d'un bord à l'autre

(a) Maquette. D (`PleinCadre.dc.html`) : `<main style="padding: 0 16px 6px">`, `.head` en `padding: 0 16px` ; la
rangée (`.band`), le plan et la barre sont dans ce `main` : marge latérale de 16 px, marge basse de 6 px. Le
Cinéma (K) est à 24 px. Grille D-2 : « rangée, plan, fil et barre de K ; marges de 16 px ; marge basse relevée :
6 px ».

(b) Code. Plein cadre : `MR/ui/frames/ReplayFrameLayout.tsx:54-57`, corps `my-auto shrink-0 space-y-2` SANS marge
latérale (le Cinéma a `px-6`, `:53`) ; il hérite des seuls 12 px de `REPLAY_FULLSCREEN_FRAME`
(`MR/hooks/useReplayFullscreen.ts:71-72`, `p-3`, 12 px de chaque côté ET en bas). L'en-tête, lui, est à 16 px
(`p-3` + `px-1`, `MR/ui/frames/ReplayFrameHeader.tsx:216-217`) : la rangée est donc 4 px plus près du bord que le
titre. Elle n'est pas collée au bord (12 px, pas 0) : le constat du user est juste en substance (écart de 4 px
par côté et désalignement avec l'en-tête), pas en lettre. La règle de tenue de la rangée compte 24 px de marge
de chaque côté pour les deux cadres (`MR/model/cardGabarit.ts:84`, `margeCorps: 24` ; `MR/model/rowLayout.ts:49-52`).

(c) Décision. Aucune. Plan 2026-11-18, « Marge de la rangée de fiches : 24 px entre les deux sens » : c'est
l'hystérésis de la bascule rangée / colonne (item C.1), sans rapport.

(d) Classement : ÉCART À CORRIGER. Cote : marge latérale du corps du Plein cadre = 16 px (alignée sur l'en-tête),
marge basse = 6 px, et `margeCorps` de la règle de tenue devient une cote PAR CADRE (24 au Cinéma, 16 au Plein
cadre) pour que le seuil de largeur reste exact. Fichiers : `MR/ui/frames/ReplayFrameLayout.tsx`,
`MR/model/cardGabarit.ts`, `MR/model/rowLayout.ts` (+ `MR/hooks/useCardRowLayout.ts` si le cadre doit y entrer),
`MR/model/rowLayout.test.ts`, `MR/ReplayPage.row.test.tsx`. Effort S. La marge basse de 6 px se heurte au `p-3`
du cadre plein écran (12 px) : à traiter en même temps ou à écrire comme reste.

## 3. Liste fermée des écarts à corriger (candidats de Q.2, un commit par point)

| Item | Point | Correction | Fichiers (MR/ = `apps/web/src/features/match-replay/`) | Effort |
|---|---|---|---|---|
| Q.2-a | 2a (et forme du point 1) | Barre d'une ligne : boutons au format `.ibtn` 30 x 30, rayon 8, sans bordure ni fond ; lecture inchangée ; variantes « compactes », barre du Standard intacte | `MR/ui/ReplayRoundButton.tsx`, `MR/ui/ReplayCameraMenu.tsx`, `MR/model/cotesMenuCamera.ts`, `MR/sound/ReplaySoundControls.tsx`, `MR/ui/ReplaySpeedMenu.tsx`, `MR/ui/ReplayAutoPlayToggle.tsx`, `MR/ui/ReplayFrameButtons.tsx`, `MR/ui/ReplayTransportCompact.tsx`, `MR/model/cotesBarreCompacte.ts` | M |
| Q.2-b | 2c | Capture et export de la barre d'une ligne : deux `.ibtn`, sans cartouche ; progression, panneau d'export et repli d'enregistrement gardés | `MR/ui/ReplayCaptureButtons.tsx`, `MR/ui/ReplayTransportCompact.tsx` | S |
| Q.2-c | 4a | Fenêtre de fiche : haut = haut du plan + 8 px | `MR/settings/useAnchoredPanel.ts`, `MR/ui/ReplayPlayerPopover.tsx`, `MR/model/cotesFenetreFiche.ts`, `MR/ui/ReplayCanvas.tsx` | S |
| Q.2-d | 4b | Fenêtre de fiche : la fenêtre entière défile (`scroll="outer"`) | `MR/ui/ReplayPlayerPopover.tsx`, `MR/model/cotesFenetreFiche.ts` | S |
| Q.2-e | 5 | Un seul bloc arrondi (12 px, bordure) pour la carte et la colonne du Standard | `MR/ui/frames/ReplayFrameLayout.tsx`, `MR/ui/ReplayCanvas.tsx`, `MR/ui/frames/ReplayStandardFrame.tsx`, `MR/ui/rosterHeight.guard.test.ts` | M |
| Q.2-f | 6 | Infobulle de « Plan tourné » avec « +N % de taille » (gain de surface de la règle Auto), fr et en | `apps/web/src/lib/replay/orientation.ts`, `MR/hooks/useResolvedTurns.ts`, `MR/hooks/useReplayView.ts`, `MR/ui/replayTransportCommands.ts`, `MR/ui/ReplayRotatedButton.tsx`, `MR/i18n/frames.i18n.ts` | M |
| Q.2-g | 8 | « Tirs à la tête | 3 » sans « sur N mesurées » ; texte retiré des deux langues | `MR/ui/fiche/FicheDepuisDebut.tsx`, `MR/i18n/playerInstant.i18n.ts`, `MR/model/playerInstantView.ts`, `MR/model/playerInstantStats.ts` | S |
| Q.2-h | 11 | Plein cadre : marge latérale 16 px, marge basse 6 px, `margeCorps` par cadre | `MR/ui/frames/ReplayFrameLayout.tsx`, `MR/model/cardGabarit.ts`, `MR/model/rowLayout.ts` | S |

Dépendances : Q.2-a avant Q.2-b (les boutons de la barre partagent le format) ; Q.2-c et Q.2-d touchent le
même fichier (`ReplayPlayerPopover.tsx`) ; Q.2-e et Q.2-c touchent `ReplayCanvas.tsx` (un seul écrivain à la
fois). Les points 1 (forme), 2b et 10 s'ajoutent à Q.2-a selon les réponses de la section 4.

## 4. Questions au user (en termes de ce qu'il verra)

1. Pictogrammes des trois cadres (point 1). Dans la barre de lecture du Cinéma et du Plein cadre, trois petits
   dessins remplacent le texte « Standard / Cinéma / Plein cadre » : une fenêtre avec un bandeau en haut
   (Standard), un écran large (Cinéma), un cadre avec un rectangle plein à l'intérieur (Plein cadre), plus les quatre coins du plein écran
   système ; le cadre actif est sur fond gris relevé, et l'infobulle dit « Cinéma (T) » / « Plein cadre (F) ».
   Option A : on garde ces dessins et ce marquage. Option B : vous les voyez d'abord sur une planche, et on les
   redessine selon vos retours.
2. Volume dans le Cinéma et le Plein cadre (point 2b). Option A : la barre n'a que l'icône de haut-parleur
   (comme les maquettes) et le volume se règle par un curseur qui s'ouvre au clic sur l'icône. Option B : on garde le curseur
   toujours visible à côté de l'icône, qui prend environ 66 px sur la barre.
3. Bouton caméra pendant un suivi (point 10). Option A : le bouton caméra est sur fond gris relevé tant que la
   caméra suit un joueur (aujourd'hui) ; vous voyez d'un coup d'oeil que la caméra est liée. Option B : le bouton reste
   semblable aux autres, et seule l'infobulle dit « Caméra : suivre <nom> » (comme les maquettes).

## 5. Découvertes hors des onze points (notées, non traitées)

1. D-5 non réalisé. Décision du user du 2026-11-15 : « Une ligne du fil qui concerne le joueur suivi porte un
   fond légèrement teinté (token) » (valable pour S.3, C.2, D.2 ; écart voulu V8). Aucun code ne teinte une ligne :
   `MR/ui/ReplayFeedLines.tsx:67-70` ne met en gras que le joueur de la fiche ouverte (`strongXuids`, passé par
   `MR/ui/ReplayFeedExtract.tsx` seulement), ni la colonne (`MR/ui/ReplayKillFeed.tsx`) ni le fil du plan
   (`MR/ui/ReplayFeedOverlay.tsx`) ne lisent le joueur suivi. À faire entrer dans Q.2 si le pilote confirme ; elle
   décide aussi de la distinction avec la ligne choisie du point 7.
2. K-8 : la barre de lecture est encore DANS le bloc arrondi du plan au Cinéma et au Plein cadre
   (`MR/ui/ReplayCanvas.tsx:583-640`, la barre `:639`), alors que K, D et H la dessinent dessous, hors du bloc, en
   `padding: 0 2px` (le code : `margeXPx: 12`, `MR/model/cotesBarreCompacte.ts:65`). La grille K-8 renvoyait ce
   point à E.1 : fait sans le déplacement. Touche le cadrage et la mesure de hauteur du plan.
3. L-3 : Suivre / Seul / Masquer sont trois boutons séparés à écart de 4 px (`MR/ui/ReplayPlayerPanelBody.tsx:123-137`),
   la maquette (L `.acts`, J-5) un SEGMENT de trois colonnes égales dont l'actif est plein. La grille notait
   « aujourd'hui trois boutons séparés à largeur égale » comme un état, sans l'inscrire aux écarts voulus.
4. Séquençage : les fichiers `ReplayCanvas.tsx`, `ReplayPlayerPopover.tsx` et `ReplayTransportCompact.tsx`
   sont à un seul écrivain dans le plan de refonte ; les corrections de la liste sont à séquencer, pas à paralléliser.

## 6. Ce que je n'ai pas pu établir

- Aucune image de maquette : le dossier ne contient que du HTML de canevas (`.dc.html`) et `canvas.json`. Tout est
  tiré du CSS et du balisage de ces fichiers et de la grille, sans rendu ; aucun navigateur ouvert, aucun test lancé.
- Les remarques que le user a pu inscrire à « l'artefact des vérifications » ne sont pas accessibles d'ici : si
  le user a déjà tranché un des onze points à l'écran, cette instruction ne le sait pas.
- « +42 % de taille » (point 6) : la maquette ne dit pas si c'est la surface ou la longueur ; j'ai retenu la
  surface parce que D-7 et le seuil de la règle sont en surface. Le gain de H n'est pas recalculable (boîte de la
  maquette inconnue).
- Écart vertical du point 4a (environ 9 px) : calculé sur les classes (`space-y-2`, bordure d'1 px, `GAP = 8`), pas
  mesuré dans un navigateur.
- Point 11 : le chiffre 12 px est celui de `p-3` ; le décalage réel dépend du défilement et de la barre de
  défilement du cadre, que je n'ai pas vus.
