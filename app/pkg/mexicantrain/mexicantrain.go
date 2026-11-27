// Package mexicantrain implements the pure, zero-import, AST-confined,
// WASM-portable Mexican Train Dominoes (Double-9 & Double-12) game kernel
// strictly conforming to spec/mexican-train.spec.json.
//
// Security & Portability Invariants:
//   - 0 imports (verified by AST confinement gate)
//   - 0 init() functions, 0 mutable package globals, 0 unsafe/cgo
//   - Compiles cleanly to GOOS=js GOARCH=wasm for Phase 2 offline mobile apps
package mexicantrain

// Tile represents a single domino with pip values Left and Right (0..MaxPip).
type Tile struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}

// PipValue returns the total pip count of the domino.
func (t Tile) PipValue() int {
	return t.Left + t.Right
}

// IsDouble reports whether both halves have identical pip values.
func (t Tile) IsDouble() bool {
	return t.Left == t.Right
}

// Matches reports whether the tile can legally connect to openEnd.
func (t Tile) Matches(openEnd int) bool {
	return t.Left == openEnd || t.Right == openEnd
}

// Orient returns the tile oriented so Left == openEnd and Right is the new open end.
func (t Tile) Orient(openEnd int) (Tile, bool) {
	if t.Left == openEnd {
		return t, true
	}
	if t.Right == openEnd {
		return Tile{Left: t.Right, Right: t.Left}, true
	}
	return t, false
}

// Train represents either the public Mexican Train (Owner == -1) or a player's train (Owner >= 0).
type Train struct {
	Owner   int    `json:"owner"`
	OpenEnd int    `json:"openEnd"`
	Public  bool   `json:"public"`
	Tiles   []Tile `json:"tiles"`
}

// State holds the complete deterministic state of a Mexican Train Dominoes hand & match.
type State struct {
	MaxPip            int      `json:"maxPip"`
	Engine            int      `json:"engine"`
	NumPlayers        int      `json:"numPlayers"`
	Turn              int      `json:"turn"`
	OpenDoubleTrain   int      `json:"openDoubleTrain"` // -2 = none; -1 = Mexican Train; 0..N-1 = player train
	MexicanTrain      Train    `json:"mexicanTrain"`
	PlayerTrains      []Train  `json:"playerTrains"`
	Hands             [][]Tile `json:"hands"`
	Boneyard          []Tile   `json:"boneyard"`
	Scores            []int    `json:"scores"`
	ConsecutivePasses int      `json:"consecutivePasses"`
	DrawnThisTurn     bool     `json:"drawnThisTurn"`
	HandOver          bool     `json:"handOver"`
	MatchOver         bool     `json:"matchOver"`
	HandWinner        int      `json:"handWinner"`
	MatchWinner       int      `json:"matchWinner"`
	LastEvent         string   `json:"lastEvent"`
}

// ActionRequest represents a player's move in a live hand.
type ActionRequest struct {
	Player      int    `json:"player"`
	Action      string `json:"action"`      // "PLAY", "DRAW", "PASS"
	TargetTrain int    `json:"targetTrain"` // -1 = Mexican Train; 0..N-1 = player train
	TileIndex   int    `json:"tileIndex"`   // index in Hands[Player] for Action == "PLAY"
	Actor       string `json:"actor,omitempty"`
}

// ActionResult is the deterministic outcome of applying ActionRequest to State.
type ActionResult struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
	State  State  `json:"state"`
}

// SpecMoveInput matches the canonical stateless transition schema in spec/mexican-train.spec.json
// and cleanroom/c-verifier/mexicantrain_kernel.c for 3-way Go/GCC/Clang equivalence testing.
type SpecMoveInput struct {
	Turn            int    `json:"turn"`
	OpenDoubleTrain int    `json:"openDoubleTrain"`
	OwnOpenEnd      int    `json:"ownOpenEnd"`
	MexOpenEnd      int    `json:"mexOpenEnd"`
	OppOpenEnd      int    `json:"oppOpenEnd"`
	OppPublic       bool   `json:"oppPublic"`
	Action          string `json:"action"`
	TargetTrain     int    `json:"targetTrain"`
	TileLeft        int    `json:"tileLeft"`
	TileRight       int    `json:"tileRight"`
	HandCount       int    `json:"handCount"`
	BoneyardCount   int    `json:"boneyardCount"`
	DrawnThisTurn   bool   `json:"drawnThisTurn"`
	HasLegalMove    bool   `json:"hasLegalMove,omitempty"`
	Actor           string `json:"actor"`
}

// SpecMoveResult matches the expected output schema in spec/mexican-train.spec.json.
type SpecMoveResult struct {
	Valid         bool   `json:"valid"`
	NewOpenEnd    int    `json:"newOpenEnd"`
	NextTurn      int    `json:"nextTurn"`
	NewOpenDouble int    `json:"newOpenDouble"`
	OwnPublic     bool   `json:"ownPublic"`
	HandOver      bool   `json:"handOver"`
	Reason        string `json:"reason"`
}

// EvalSpecMove evaluates a single stateless transition per spec/mexican-train.spec.json.
// Actor is intentionally ignored to satisfy MT-INV-6 (anti-backdoor determinism).
func EvalSpecMove(in SpecMoveInput) SpecMoveResult {
	_ = in.Actor

	if in.Turn != 0 && in.Turn != 1 {
		return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_INVALID_TURN"}
	}
	opp := 1 - in.Turn

	if in.Action == "PASS" {
		if in.HasLegalMove {
			return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_LEGAL_MOVE_AVAILABLE"}
		}
		if in.BoneyardCount > 0 && !in.DrawnThisTurn {
			return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_MUST_DRAW_FIRST"}
		}
		return SpecMoveResult{Valid: true, NewOpenEnd: in.OwnOpenEnd, NextTurn: opp, NewOpenDouble: in.OpenDoubleTrain, OwnPublic: true, Reason: "TURN_PASSED_TRAIN_OPENED"}
	}

	if in.Action == "DRAW" {
		if in.HasLegalMove {
			return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_LEGAL_MOVE_AVAILABLE"}
		}
		if in.DrawnThisTurn {
			return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_ALREADY_DRAWN"}
		}
		if in.BoneyardCount <= 0 {
			return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_BONEYARD_EMPTY"}
		}
		return SpecMoveResult{Valid: true, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "TILE_DRAWN"}
	}

	if in.Action != "PLAY" {
		return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_INVALID_ACTION"}
	}

	if in.HandCount <= 0 {
		return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_EMPTY_HAND"}
	}

	if in.TargetTrain != -1 && in.TargetTrain != 0 && in.TargetTrain != 1 {
		return SpecMoveResult{Valid: false, NewOpenEnd: in.OwnOpenEnd, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_INVALID_TRAIN"}
	}

	targetOpen := in.MexOpenEnd
	if in.TargetTrain == in.Turn {
		targetOpen = in.OwnOpenEnd
	} else if in.TargetTrain == opp {
		targetOpen = in.OppOpenEnd
	}

	if in.OpenDoubleTrain != -2 && in.TargetTrain != in.OpenDoubleTrain {
		return SpecMoveResult{Valid: false, NewOpenEnd: targetOpen, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_MUST_SATISFY_DOUBLE"}
	}

	if in.OpenDoubleTrain == -2 && in.TargetTrain == opp && !in.OppPublic {
		return SpecMoveResult{Valid: false, NewOpenEnd: targetOpen, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_TRAIN_NOT_PUBLIC"}
	}

	if in.TileLeft < 0 || in.TileRight < 0 || in.TileLeft > 12 || in.TileRight > 12 {
		return SpecMoveResult{Valid: false, NewOpenEnd: targetOpen, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_INVALID_TILE"}
	}

	var newOpen int
	if in.TileLeft == targetOpen {
		newOpen = in.TileRight
	} else if in.TileRight == targetOpen {
		newOpen = in.TileLeft
	} else {
		return SpecMoveResult{Valid: false, NewOpenEnd: targetOpen, NextTurn: in.Turn, NewOpenDouble: in.OpenDoubleTrain, Reason: "ERR_PIP_MISMATCH"}
	}

	if in.HandCount == 1 {
		return SpecMoveResult{Valid: true, NewOpenEnd: newOpen, NextTurn: in.Turn, NewOpenDouble: -2, HandOver: true, Reason: "HAND_WON"}
	}

	if in.TileLeft == in.TileRight {
		return SpecMoveResult{Valid: true, NewOpenEnd: newOpen, NextTurn: in.Turn, NewOpenDouble: in.TargetTrain, Reason: "DOUBLE_PLAYED_BONUS_TURN"}
	}

	if in.OpenDoubleTrain != -2 {
		return SpecMoveResult{Valid: true, NewOpenEnd: newOpen, NextTurn: opp, NewOpenDouble: -2, Reason: "DOUBLE_SATISFIED"}
	}

	return SpecMoveResult{Valid: true, NewOpenEnd: newOpen, NextTurn: opp, NewOpenDouble: -2, Reason: "MOVE_ACCEPTED"}
}

// TotalTilesForVariant returns 55 for Double-9 or 91 for Double-12.
func TotalTilesForVariant(maxPip int) int {
	return (maxPip + 1) * (maxPip + 2) / 2
}

// HandSizeForVariant returns the standard starting hand size per player.
func HandSizeForVariant(maxPip, numPlayers int) int {
	if maxPip == 12 {
		if numPlayers <= 4 {
			return 15
		}
		return 12
	}
	// Double-9 standard: 12 tiles in 2-player, 10 tiles in 3-4 player
	if numPlayers <= 2 {
		return 12
	}
	return 10
}

// splitmix64 is a pure 0-import deterministic PRNG seeded externally by CSPRNG or test seeds.
func splitmix64(state *uint64) uint64 {
	*state += 0x9e3779b97f4a7c15
	z := *state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// NewMatch initializes a brand new Double-9 (maxPip=9) or Double-12 (maxPip=12) match.
func NewMatch(maxPip, numPlayers int, seed uint64) State {
	if maxPip != 9 && maxPip != 12 {
		maxPip = 9
	}
	if numPlayers < 2 || numPlayers > 4 {
		numPlayers = 2
	}
	scores := make([]int, numPlayers)
	return NewHand(maxPip, maxPip, numPlayers, seed, scores, 0)
}

// NewHand deals a deterministic hand for the specified engine (maxPip down to 0).
// Preserves MT-INV-1 (exact tile conservation with zero duplicates).
func NewHand(maxPip, engine, numPlayers int, seed uint64, prevScores []int, startingPlayer int) State {
	if maxPip != 9 && maxPip != 12 {
		maxPip = 9
	}
	if engine < 0 {
		engine = 0
	}
	if engine > maxPip {
		engine = maxPip
	}
	if numPlayers < 2 || numPlayers > 4 {
		numPlayers = 2
	}
	if startingPlayer < 0 || startingPlayer >= numPlayers {
		startingPlayer = 0
	}

	// Build complete deck excluding the center engine double (engine, engine)
	deck := make([]Tile, 0, TotalTilesForVariant(maxPip)-1)
	for a := 0; a <= maxPip; a++ {
		for b := a; b <= maxPip; b++ {
			if a == engine && b == engine {
				continue
			}
			deck = append(deck, Tile{Left: a, Right: b})
		}
	}

	// Deterministic Fisher-Yates shuffle
	rng := seed
	if rng == 0 {
		rng = 0x517cc1b727220a95
	}
	for i := len(deck) - 1; i > 0; i-- {
		j := int(splitmix64(&rng) % uint64(i+1))
		deck[i], deck[j] = deck[j], deck[i]
	}

	handSize := HandSizeForVariant(maxPip, numPlayers)
	hands := make([][]Tile, numPlayers)
	idx := 0
	for p := 0; p < numPlayers; p++ {
		hands[p] = make([]Tile, handSize)
		copy(hands[p], deck[idx:idx+handSize])
		sortTilesDesc(hands[p])
		idx += handSize
	}

	boneyard := make([]Tile, len(deck)-idx)
	copy(boneyard, deck[idx:])

	playerTrains := make([]Train, numPlayers)
	for p := 0; p < numPlayers; p++ {
		playerTrains[p] = Train{
			Owner:   p,
			OpenEnd: engine,
			Public:  false,
			Tiles:   []Tile{},
		}
	}

	scores := make([]int, numPlayers)
	for p := 0; p < numPlayers && p < len(prevScores); p++ {
		scores[p] = prevScores[p]
	}

	return State{
		MaxPip:          maxPip,
		Engine:          engine,
		NumPlayers:      numPlayers,
		Turn:            startingPlayer,
		OpenDoubleTrain: -2,
		MexicanTrain: Train{
			Owner:   -1,
			OpenEnd: engine,
			Public:  true,
			Tiles:   []Tile{},
		},
		PlayerTrains: playerTrains,
		Hands:        hands,
		Boneyard:     boneyard,
		Scores:       scores,
		HandWinner:   -1,
		MatchWinner:  -1,
		LastEvent:    "Hand dealt",
	}
}

// sortTilesDesc sorts tiles highest-pip first using simple insertion sort (0 imports).
func sortTilesDesc(tiles []Tile) {
	for i := 1; i < len(tiles); i++ {
		cur := tiles[i]
		j := i - 1
		for j >= 0 && (tiles[j].PipValue() < cur.PipValue() || (tiles[j].PipValue() == cur.PipValue() && tiles[j].Right < cur.Right)) {
			tiles[j+1] = tiles[j]
			j--
		}
		tiles[j+1] = cur
	}
}

// CloneState returns a deep copy of State so pure evaluation never mutates caller slices.
func CloneState(s State) State {
	out := s
	out.MexicanTrain.Tiles = append([]Tile(nil), s.MexicanTrain.Tiles...)
	out.PlayerTrains = make([]Train, len(s.PlayerTrains))
	for i, tr := range s.PlayerTrains {
		out.PlayerTrains[i] = tr
		out.PlayerTrains[i].Tiles = append([]Tile(nil), tr.Tiles...)
	}
	out.Hands = make([][]Tile, len(s.Hands))
	for i, h := range s.Hands {
		out.Hands[i] = append([]Tile(nil), h...)
	}
	out.Boneyard = append([]Tile(nil), s.Boneyard...)
	out.Scores = append([]int(nil), s.Scores...)
	return out
}

// PipSum returns the sum of pips in a hand.
func PipSum(hand []Tile) int {
	total := 0
	for _, t := range hand {
		total += t.PipValue()
	}
	return total
}

func trainPtr(s *State, target int) *Train {
	if target == -1 {
		return &s.MexicanTrain
	}
	if target >= 0 && target < len(s.PlayerTrains) {
		return &s.PlayerTrains[target]
	}
	return nil
}

// CanPlayOnTrain reports whether player p may legally target train target right now.
func CanPlayOnTrain(s State, p, target int) bool {
	if s.HandOver || p != s.Turn {
		return false
	}
	tr := trainPtr(&s, target)
	if tr == nil {
		return false
	}
	if s.OpenDoubleTrain != -2 {
		return target == s.OpenDoubleTrain
	}
	if target == -1 || target == p {
		return true
	}
	return tr.Public
}

// LegalMove represents one valid (TileIndex, TargetTrain) play for the active player.
type LegalMove struct {
	TileIndex   int `json:"tileIndex"`
	TargetTrain int `json:"targetTrain"`
}

// LegalMoves returns all legal PLAY moves available to the active player in state s.
func LegalMoves(s State) []LegalMove {
	if s.HandOver || s.Turn < 0 || s.Turn >= len(s.Hands) {
		return nil
	}
	p := s.Turn
	hand := s.Hands[p]
	targets := make([]int, 0, s.NumPlayers+1)
	targets = append(targets, p, -1)
	for other := 0; other < s.NumPlayers; other++ {
		if other != p {
			targets = append(targets, other)
		}
	}

	var moves []LegalMove
	for ti, tile := range hand {
		for _, target := range targets {
			if !CanPlayOnTrain(s, p, target) {
				continue
			}
			tr := trainPtr(&s, target)
			if tr != nil && tile.Matches(tr.OpenEnd) {
				moves = append(moves, LegalMove{TileIndex: ti, TargetTrain: target})
			}
		}
	}
	return moves
}

// ApplyAction validates and applies ActionRequest to State purely (returns a cloned next State).
func ApplyAction(orig State, req ActionRequest) ActionResult {
	_ = req.Actor // MT-INV-6: strictly actor-independent

	s := CloneState(orig)
	if s.HandOver {
		return ActionResult{Valid: false, Reason: "ERR_HAND_ALREADY_OVER", State: orig}
	}
	if req.Player != s.Turn || req.Player < 0 || req.Player >= s.NumPlayers {
		return ActionResult{Valid: false, Reason: "ERR_OUT_OF_TURN", State: orig}
	}
	p := req.Player
	nextP := (p + 1) % s.NumPlayers

	switch req.Action {
	case "DRAW":
		if len(LegalMoves(s)) > 0 {
			return ActionResult{Valid: false, Reason: "ERR_LEGAL_MOVE_AVAILABLE", State: orig}
		}
		if s.DrawnThisTurn {
			return ActionResult{Valid: false, Reason: "ERR_ALREADY_DRAWN", State: orig}
		}
		if len(s.Boneyard) == 0 {
			return ActionResult{Valid: false, Reason: "ERR_BONEYARD_EMPTY", State: orig}
		}
		drawnTile := s.Boneyard[0]
		s.Boneyard = s.Boneyard[1:]
		s.Hands[p] = append(s.Hands[p], drawnTile)
		sortTilesDesc(s.Hands[p])
		s.DrawnThisTurn = true
		s.LastEvent = "Drew a tile from the boneyard"
		return ActionResult{Valid: true, Reason: "TILE_DRAWN", State: s}

	case "PASS":
		if len(LegalMoves(s)) > 0 {
			return ActionResult{Valid: false, Reason: "ERR_LEGAL_MOVE_AVAILABLE", State: orig}
		}
		if len(s.Boneyard) > 0 && !s.DrawnThisTurn {
			return ActionResult{Valid: false, Reason: "ERR_MUST_DRAW_FIRST", State: orig}
		}
		s.PlayerTrains[p].Public = true
		s.DrawnThisTurn = false
		s.Turn = nextP
		if len(s.Boneyard) == 0 {
			s.ConsecutivePasses++
		} else {
			s.ConsecutivePasses = 0
		}
		s.LastEvent = "Passed turn — personal train is now OPEN"
		if len(s.Boneyard) == 0 && s.ConsecutivePasses >= s.NumPlayers {
			finalizeHand(&s, -1)
			return ActionResult{Valid: true, Reason: "HAND_BLOCKED", State: s}
		}
		return ActionResult{Valid: true, Reason: "TURN_PASSED_TRAIN_OPENED", State: s}

	case "PLAY":
		if req.TileIndex < 0 || req.TileIndex >= len(s.Hands[p]) {
			return ActionResult{Valid: false, Reason: "ERR_INVALID_TILE_INDEX", State: orig}
		}
		tr := trainPtr(&s, req.TargetTrain)
		if tr == nil {
			return ActionResult{Valid: false, Reason: "ERR_INVALID_TRAIN", State: orig}
		}
		if s.OpenDoubleTrain != -2 && req.TargetTrain != s.OpenDoubleTrain {
			return ActionResult{Valid: false, Reason: "ERR_MUST_SATISFY_DOUBLE", State: orig}
		}
		if s.OpenDoubleTrain == -2 && req.TargetTrain >= 0 && req.TargetTrain != p && !tr.Public {
			return ActionResult{Valid: false, Reason: "ERR_TRAIN_NOT_PUBLIC", State: orig}
		}
		tile := s.Hands[p][req.TileIndex]
		oriented, ok := tile.Orient(tr.OpenEnd)
		if !ok {
			return ActionResult{Valid: false, Reason: "ERR_PIP_MISMATCH", State: orig}
		}

		// Place tile onto target train & remove from player hand
		tr.Tiles = append(tr.Tiles, oriented)
		tr.OpenEnd = oriented.Right
		s.Hands[p] = append(s.Hands[p][:req.TileIndex], s.Hands[p][req.TileIndex+1:]...)

		// MT-INV-3: Playing on own train removes the public marker
		if req.TargetTrain == p {
			s.PlayerTrains[p].Public = false
		}
		s.ConsecutivePasses = 0
		s.DrawnThisTurn = false

		// MT-INV-5: Emptying hand wins immediately (even if final tile is a double)
		if len(s.Hands[p]) == 0 {
			s.OpenDoubleTrain = -2
			finalizeHand(&s, p)
			return ActionResult{Valid: true, Reason: "HAND_WON", State: s}
		}

		// MT-INV-4: Playing a double grants an immediate bonus turn to cover it
		if oriented.IsDouble() {
			s.OpenDoubleTrain = req.TargetTrain
			s.Turn = p
			s.LastEvent = "Double played — must be satisfied!"
			return ActionResult{Valid: true, Reason: "DOUBLE_PLAYED_BONUS_TURN", State: s}
		}

		wasOpenDouble := s.OpenDoubleTrain != -2
		s.OpenDoubleTrain = -2
		s.Turn = nextP
		if wasOpenDouble {
			s.LastEvent = "Open double satisfied"
			return ActionResult{Valid: true, Reason: "DOUBLE_SATISFIED", State: s}
		}
		s.LastEvent = "Tile played"
		return ActionResult{Valid: true, Reason: "MOVE_ACCEPTED", State: s}

	default:
		return ActionResult{Valid: false, Reason: "ERR_INVALID_ACTION", State: orig}
	}
}

func finalizeHand(s *State, emptiedPlayer int) {
	s.HandOver = true
	bestPips := -1
	winner := emptiedPlayer
	for p := 0; p < s.NumPlayers; p++ {
		pips := PipSum(s.Hands[p])
		s.Scores[p] += pips
		if emptiedPlayer == -1 {
			if bestPips == -1 || pips < bestPips {
				bestPips = pips
				winner = p
			}
		}
	}
	s.HandWinner = winner
	if s.Engine == 0 {
		s.MatchOver = true
		bestScore := s.Scores[0]
		matchWin := 0
		for p := 1; p < s.NumPlayers; p++ {
			if s.Scores[p] < bestScore {
				bestScore = s.Scores[p]
				matchWin = p
			}
		}
		s.MatchWinner = matchWin
		s.LastEvent = "Match complete!"
	} else if emptiedPlayer >= 0 {
		s.LastEvent = "Hand complete — player went out!"
	} else {
		s.LastEvent = "Hand blocked — scored by remaining pips"
	}
}

// ChooseBotMove picks a strong, deterministic pure-Go move for the active player.
// Works identically on Cloud Run and offline Phase 2 WASM mobile builds.
func ChooseBotMove(s State) ActionRequest {
	p := s.Turn
	moves := LegalMoves(s)
	if len(moves) == 0 {
		if !s.DrawnThisTurn && len(s.Boneyard) > 0 {
			return ActionRequest{Player: p, Action: "DRAW", Actor: "nvg-bot"}
		}
		return ActionRequest{Player: p, Action: "PASS", Actor: "nvg-bot"}
	}

	best := moves[0]
	bestScore := -100000
	for _, m := range moves {
		tile := s.Hands[p][m.TileIndex]
		score := tile.PipValue() * 10
		// Closing own public train is high priority
		if m.TargetTrain == p && s.PlayerTrains[p].Public {
			score += 60
		}
		// Prefer playing on own train slightly over opening Mexican Train
		if m.TargetTrain == p {
			score += 15
		}
		// If tile is a double, prefer it if we also have a cover tile or it's our last tile
		if tile.IsDouble() {
			if len(s.Hands[p]) == 1 {
				score += 500
			} else {
				hasCover := false
				for idx, other := range s.Hands[p] {
					if idx != m.TileIndex && other.Matches(tile.Left) {
						hasCover = true
						break
					}
				}
				if hasCover {
					score += 45
				} else {
					score -= 25
				}
			}
		}
		if score > bestScore {
			bestScore = score
			best = m
		}
	}
	return ActionRequest{
		Player:      p,
		Action:      "PLAY",
		TargetTrain: best.TargetTrain,
		TileIndex:   best.TileIndex,
		Actor:       "nvg-bot",
	}
}

// VerifyInvariants checks MT-INV-1 (exact tile conservation & zero duplicates) and
// train chain continuity (MT-INV-2) on any State. Returns empty string if valid.
func VerifyInvariants(s State) string {
	if s.MaxPip != 9 && s.MaxPip != 12 {
		return "invalid maxPip"
	}
	size := s.MaxPip + 1
	seen := make([]bool, size*size)
	mark := func(t Tile) bool {
		a, b := t.Left, t.Right
		if a < 0 || b < 0 || a > s.MaxPip || b > s.MaxPip {
			return false
		}
		if a > b {
			a, b = b, a
		}
		k := a*size + b
		if seen[k] {
			return false
		}
		seen[k] = true
		return true
	}

	// Mark center engine double
	if !mark(Tile{Left: s.Engine, Right: s.Engine}) {
		return "invalid engine tile"
	}
	count := 1

	checkTrain := func(tr Train) string {
		prev := s.Engine
		for _, t := range tr.Tiles {
			if t.Left != prev {
				return "broken train chain"
			}
			if !mark(t) {
				return "duplicate or out-of-range train tile"
			}
			prev = t.Right
			count++
		}
		if tr.OpenEnd != prev {
			return "train openEnd mismatch"
		}
		return ""
	}

	if err := checkTrain(s.MexicanTrain); err != "" {
		return "mexican train: " + err
	}
	for _, tr := range s.PlayerTrains {
		if err := checkTrain(tr); err != "" {
			return "player train: " + err
		}
	}
	for _, h := range s.Hands {
		for _, t := range h {
			if !mark(t) {
				return "duplicate or out-of-range hand tile"
			}
			count++
		}
	}
	for _, t := range s.Boneyard {
		if !mark(t) {
			return "duplicate or out-of-range boneyard tile"
		}
		count++
	}

	expected := TotalTilesForVariant(s.MaxPip)
	if count != expected {
		return "tile conservation violated"
	}
	return ""
}
