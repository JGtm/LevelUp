package filmdec

// film_context.go — CE QUE TOUS LES BALAYAGES D'UN MEME FILM SE PARTAGENT.
//
// # CE QUE CE FICHIER FERME
//
// Lot 2 de PLAN_CUISSON_PERF (2026-09-03). Le lot 1 avait supprime les ~36 relectures du film :
// les balayages recoivent un `*filmsource.Film` decompresse UNE fois. Restait le second etage du
// meme defaut — chaque balayage RECALCULAIT, sur ce film deja charge, les trois memes
// derivations :
//
//	la BANDE DE SLOTS BIPEDE   `bipedSlotBand` — une marche de l'image-cle de tete de CHAQUE
//	                            chunk. Huit balayages la relevaient (positions, ramassages
//	                            natifs, et les six canaux delta).
//	le DECOUPAGE D'i0          `DetectI0LayoutOf` — six chunks marches bit a bit, plus SA
//	                            PROPRE bande. Six balayages delta le redetectaient.
//	le REGISTRE chunk_00       `ParseRegistryChunk` — une douzaine de re-analyses par cuisson,
//	                            une par accesseur d'archetype.
//
// Ces trois valeurs ne dependent QUE du film. Les recalculer par balayage etait du temps pur.
// [FilmContext] les porte, construit UNE fois par `replay.BuildFromFilm`, et les balayages le
// recoivent a la place du film.
//
// # LA REGLE DU CATALOGUE (lot 3, 2026-09-03)
//
// Le decoupage d'i0 n'est plus l'auto-detecte : c'est celui du CATALOGUE quand l'entree de carte
// est valide, l'auto-detection en repli — la regle EXACTE que les positions appliquaient deja
// seules (`replay/build_from_film.go`, doctrine heritee de `build.go:249-259`). Elle est ecrite
// UNE fois, ici, et les positions la lisent au meme endroit que les six canaux delta
// ([FilmContext.ImposedLayout]).
//
// CE QUE CELA CORRIGE, ET OU. Sur une carte a plus de deux regions de compression, l'index de
// region occupe PLUS d'un bit dans i0, et l'auto-detection ne sait pas le voir : elle rend
// toujours `GateBits = DefaultI0GateBits` (5) et `Region = 0` (cf. i0_layout.go). Sur Live Fire
// — 4 regions declarees, arene en region 1, catalogue `gate=6 region=1 12/12/11` — elle rend
// `gate=5 region=0 13/12/11` : MEME longueur totale d'i0 (41 bits), donc les balayages delta
// marchaient bel et bien, mais la porte de region ne testait qu'UN bit contre zero. Elle
// acceptait donc AUSSI les enregistrements de la region 00, dont les quanta sont exprimes dans
// une AUTRE AABB : 27 enregistrements sur 267 400 (mesure du 2026-09-03 sur `60ae07c4`), qui
// n'appartiennent pas a l'arene jouee. Le catalogue teste les DEUX bits contre 1 et les ecarte.
// Sur les 12 autres films du corpus d'equivalence, catalogue et auto-detection donnent le MEME
// decoupage, au bit pres — c'est pourquoi la correction ne mord que sur Live Fire.
//
// # POURQUOI LA MEMOISATION EST PARESSEUSE, ET NON CALCULEE AU CONSTRUCTEUR
//
// Le lot 2 est un REFACTO PUR : les sorties doivent etre identiques a l'octet, et l'ORDRE des
// etapes observees ne doit pas bouger (garde `replay/observe_test.go`, harnais `cmd/replay-equiv`).
// Un constructeur qui calculerait tout d'avance deplacerait le premier calcul AVANT le premier
// balayage — donc avant l'installation des largeurs d'axe et avant le demarrage de l'horloge des
// etapes — et ferait travailler un film qui echoue des les positions. Paresseux, le premier
// calcul a lieu EXACTEMENT la ou il avait lieu avant (le premier balayage qui en a besoin) ;
// les suivants le lisent. C'est aussi ce qui garantit « jamais pire qu'avant » aux enveloppes D2,
// qui construisent leur propre contexte a chaque appel.
//
// # POURQUOI LE CONSTRUCTEUR NE REND PAS D'ERREUR
//
// Les trois derivations ECHOUENT sur des films legitimes — une bobine partielle n'a pas de
// `chunk_00`, un film trop court ne donne pas trois frontieres nettes dans i0 — et chaque
// balayage rend AUJOURD'HUI son propre message a son propre moment (`ErrNoRegistryChunk`,
// « decoupage i0 illisible », « aucun slot biped (ti=35) ... »). Refuser au constructeur
// changerait ces messages ET l'etape a laquelle la cuisson s'arrete : la fixture
// `replay/testdata/minifilm_000d5950` n'a ni registre ni slot bipede, et
// `replay/zero_disque_test.go` exige l'erreur EXACTE des positions. Le contexte MEMORISE donc
// l'echec au lieu de le lever, et chaque accesseur le rejoue a l'identique, autant de fois qu'on
// le lui demande.
//
// # CE QU'IL N'EST PAS
//
// Ni un cache global (aucun `var` de paquet — le ratchet `archlint/filmdec_package_vars_test.go`
// gele leur compte), ni un objet partageable entre goroutines : il n'est ni verrouille ni
// atomique, et il vit sous le meme `LockProcessDecode` que le decodage qu'il sert.

import "levelup/go-api/internal/analysis/filmsource"

// FilmContext porte les derivations d'un film qui ne dependent que de lui : les numeros de ses
// chunks de donnees, la bande de slots bipede, le decoupage d'i0 et le registre chunk_00. Il se
// construit par [NewFilmContextForMap] (production, avec le catalogue de la carte) ou par
// [NewFilmContext] (auto-detection seule) et se passe aux balayages.
//
// Les champs sont PRIVES : un layout ou un registre se lit avec l'erreur qui va avec (cf.
// l'en-tete), et une bande de slots exposee en clair serait modifiable par son lecteur.
type FilmContext struct {
	film *filmsource.Film

	chunks    []int
	chunksLus bool

	slots    SlotBand
	slotsLus bool

	// impose est le decoupage d'i0 que la REGLE DU CATALOGUE tranche a la construction (cf.
	// resolveI0Layout). Non nil = les trois champs `lay*` ci-dessous ne servent pas : aucune
	// auto-detection n'a lieu, et c'est aussi ce qui la retire du chemin de cuisson.
	impose *I0Layout

	lay    I0Layout
	layErr error
	layLu  bool

	reg    *Registry
	regErr error
	regLu  bool
}

// NewFilmContext ouvre le contexte d'un film DEJA CHARGE, SANS catalogue : le decoupage d'i0 est
// l'AUTO-DETECTE. C'est la forme des enveloppes D2 `ScanFilm*(dir)` et des usages hors production
// (recherche, instruments), qui n'ont pas d'entree de carte a fournir ; la cuisson, elle, passe
// par [NewFilmContextForMap].
//
// Il ne lit rien : chaque derivation est calculee au premier accesseur qui la demande, puis
// memorisee (cf. l'en-tete du fichier).
//
// `film` nil est ACCEPTE et n'est pas une erreur : la cuisson passe un film nil quand les chunks
// sont illisibles (`replaybuild.chargerFilm`), et chaque balayage rend alors son
// [ErrNoFilmChunk] a sa place — exactement comme un repertoire vide avant le lot 1.
func NewFilmContext(film *filmsource.Film) *FilmContext {
	return &FilmContext{film: film}
}

// NewFilmContextForMap ouvre le contexte d'un film DEJA CHARGE sous LA REGLE DU CATALOGUE (cf.
// l'en-tete du fichier) : le decoupage d'i0 est celui de `entry` quand l'entree est valide,
// l'auto-detection en repli ; un decoupage deja FORCE par l'appelant (`ScanFilmOptions.Layout`,
// `replay.Options.Scan`) reste maitre. Les deux parametres acceptent nil.
//
// C'est le constructeur de la CUISSON : `replay.BuildFromFilm` le construit une fois, lit le
// decoupage tranche par [FilmContext.ImposedLayout] pour en armer les positions, et passe le
// contexte aux six canaux delta et aux ramassages natifs — un seul decoupage pour tout le film.
func NewFilmContextForMap(film *filmsource.Film, entry *MapQuantEntry, forced *I0Layout) *FilmContext {
	return &FilmContext{film: film, impose: resolveI0Layout(forced, entry)}
}

// resolveI0Layout EST LA REGLE, ecrite une fois : le decoupage FORCE s'il y en a un, sinon celui
// du CATALOGUE quand il est valide, sinon nil — l'auto-detection.
//
// Le repli sur nil n'est pas une tolerance : une entree de catalogue anterieure au champ des
// largeurs (`axisWidths` absent, donc `Valid()` faux) doit laisser lire le film plutot
// qu'imposer des largeurs nulles, exactement comme le chemin world-object garde son defaut.
func resolveI0Layout(forced *I0Layout, entry *MapQuantEntry) *I0Layout {
	if forced != nil {
		lay := *forced
		return &lay
	}
	if entry != nil {
		if lay := entry.Layout(); lay.Valid() {
			return &lay
		}
	}
	return nil
}

// ImposedLayout rend le decoupage d'i0 que la regle du catalogue a tranche a la construction, ou
// nil quand rien ne s'impose (auto-detection). La valeur est COPIEE : un appelant qui la range
// dans ses propres options ne peut pas modifier celle du contexte.
func (c *FilmContext) ImposedLayout() *I0Layout {
	if c == nil || c.impose == nil {
		return nil
	}
	lay := *c.impose
	return &lay
}

// Film rend le film sous-jacent, pour les balayages qui lisent des chunks sans rien deriver.
func (c *FilmContext) Film() *filmsource.Film {
	if c == nil {
		return nil
	}
	return c.film
}

// ChunkNumbers rend les numeros des chunks de DONNEES du film ([FilmChunkNumbers]), releves une
// fois. La tranche est celle du contexte : ses lecteurs la parcourent, ils ne la modifient pas.
func (c *FilmContext) ChunkNumbers() []int {
	if c == nil {
		return nil
	}
	if !c.chunksLus {
		c.chunks, c.chunksLus = FilmChunkNumbers(c.film), true
	}
	return c.chunks
}

// ChunkAt rend les octets decompresses du chunk de NUMERO `num` et ses paquets ([FilmChunkAt]).
// Rien n'est memorise : la conversion des en-tetes est deja le prix plancher (cf. film_chunks.go).
func (c *FilmContext) ChunkAt(num int) ([]byte, []FilmPacket, bool) {
	if c == nil {
		return nil, nil, false
	}
	return FilmChunkAt(c.film, num)
}

// BipedSlots rend la bande de slots bipede du film ([bipedSlotBand] sur TOUS les chunks de
// donnees), relevee une fois.
//
// Bande VIDE quand le film n'a pas de chunk de donnees : les balayages testent `len(...) == 0`
// avant d'appeler, et rendent [ErrNoFilmChunk] — la garde ci-dessous ne fait que leur epargner
// l'indexation d'une tranche vide dans `bipedSlotBand`.
//
// CE N'EST PAS LA BANDE DE `DetectI0LayoutOf`, qui releve la SIENNE sur les six premiers chunks
// seulement : deux valeurs differentes, deux calculs, et c'est pourquoi la detection garde le
// sien (cf. i0_layout.go).
func (c *FilmContext) BipedSlots() SlotBand {
	if c == nil {
		return SlotBand{}
	}
	if !c.slotsLus {
		if nums := c.ChunkNumbers(); len(nums) > 0 {
			c.slots = bipedSlotBand(c.film, nums)
		}
		c.slotsLus = true
	}
	return c.slots
}

// I0Layout rend le decoupage d'i0 du film sous la REGLE DU CATALOGUE : celui que le catalogue
// impose quand l'entree de carte est valide (jamais d'erreur — il ne se lit pas dans le film),
// sinon l'AUTO-DETECTE ([DetectI0LayoutOf]), detecte une fois. L'erreur rendue est alors celle
// de la detection, BRUTE : c'est l'appelant qui l'habille (« decoupage i0 illisible : %w »),
// comme il le faisait de l'appel direct.
func (c *FilmContext) I0Layout() (I0Layout, error) {
	if c == nil {
		return I0Layout{}, ErrNoFilmChunk
	}
	if c.impose != nil {
		return *c.impose, nil
	}
	if !c.layLu {
		c.lay, _, c.layErr = DetectI0LayoutOf(c.film)
		c.layLu = true
	}
	return c.lay, c.layErr
}

// Registry rend le registre ECS du film (le chunk NUMERO 0), analyse une fois.
//
// [ErrNoRegistryChunk] quand le film ne porte pas son registre — bobine partielle, fixture : les
// lecteurs d'archetype le disent plutot que de rendre un registre vide, qui se lirait comme
// « archetype absent du build ».
//
// C'EST LE SEUL LECTEUR DE REGISTRE DU PAQUET : les six accesseurs d'archetype
// (`bipedArchetype`, `EquipmentArchetype`, `groundWeaponArchetype`, `managedPropertyArchetype`,
// `filmArchetype`, `objectiveArchetype`) en derivent tous, chacun ne gardant que son message
// d'archetype manquant. Un garde-rail ferme la porte a un second site d'analyse
// (`archlint/no_recomputed_film_context_test.go`).
func (c *FilmContext) Registry() (*Registry, error) {
	if c == nil {
		return nil, ErrNoRegistryChunk
	}
	if !c.regLu {
		c.regLu = true
		raw, ok := FilmRegistryChunk(c.film)
		if !ok {
			c.regErr = ErrNoRegistryChunk
		} else {
			c.reg, c.regErr = ParseRegistryChunk(raw)
		}
	}
	return c.reg, c.regErr
}

// archetype rend l'archetype `ti` du registre memorise. `ok` faux = le registre a ete lu mais ne
// porte pas cet archetype ; c'est a l'accesseur nomme de dire lequel manque, avec SON message —
// les six libelles historiques sont conserves mot pour mot.
func (c *FilmContext) archetype(ti int) (Archetype, *Registry, bool, error) {
	reg, err := c.Registry()
	if err != nil {
		return Archetype{}, nil, false, err
	}
	arch, ok := reg.Archetype(ti)
	return arch, reg, ok, nil
}
