// Package tictactoe implements the pure, AST-confined, formally verifiable
// Tic-Tac-Toe game kernel derived directly from spec/app.spec.json.
//
// SECURITY INVARIANT (Enforced by Software Factory AST Gate):
//   - Zero I/O, zero network, zero os/exec, zero unsafe, zero reflect, zero cgo
//   - Zero init() functions and zero package-level mutable vars
//   - Pure deterministic state transition function verified exhaustively and
//     cross-checked against the independent Clean-Room C reference kernel
//     compiled with both GCC and Clang.
package tictactoe

// MoveRequest represents a deterministic state transition request.
type MoveRequest struct {
	Board    string `json:"board"`
	Player   string `json:"player"`
	Position int    `json:"position"`
	Actor    string `json:"actor,omitempty"`
}

// MoveResult represents the deterministic outcome of ApplyMove.
type MoveResult struct {
	Valid    bool   `json:"valid"`
	Board    string `json:"board"`
	Status   string `json:"status"` // IN_PROGRESS, WIN, DRAW, INVALID
	Winner   string `json:"winner"`
	NextTurn string `json:"nextTurn"`
	Reason   string `json:"reason"`
}

// ApplyMove validates the board, turn balance, terminal immutability, bounds,
// and occupancy, then returns the next immutable board state and status.
func ApplyMove(req MoveRequest) MoveResult {
	// TTT-INV-1: Board integrity (9 chars of '.', 'X', 'O')
	if len(req.Board) != 9 {
		return invalidResult(req.Board, "", "", "ERR_INVALID_BOARD")
	}
	xCount, oCount := 0, 0
	for i := 0; i < 9; i++ {
		switch req.Board[i] {
		case '.':
		case 'X':
			xCount++
		case 'O':
			oCount++
		default:
			return invalidResult(req.Board, "", "", "ERR_INVALID_BOARD")
		}
	}

	// TTT-INV-2: Turn balance
	var expectedTurn string
	if xCount == oCount {
		expectedTurn = "X"
	} else if xCount == oCount+1 {
		expectedTurn = "O"
	} else {
		return invalidResult(req.Board, "", "", "ERR_INVALID_BOARD_BALANCE")
	}

	// TTT-INV-4: Terminal immutability (already won or full)
	if w := detectWinner(req.Board); w != "" {
		return invalidResult(req.Board, w, "", "ERR_GAME_ALREADY_OVER")
	}
	if xCount+oCount == 9 {
		return invalidResult(req.Board, "", "", "ERR_GAME_ALREADY_OVER")
	}

	// Player & turn check
	if req.Player != "X" && req.Player != "O" {
		return invalidResult(req.Board, "", expectedTurn, "ERR_INVALID_PLAYER")
	}
	if req.Player != expectedTurn {
		return invalidResult(req.Board, "", expectedTurn, "ERR_OUT_OF_TURN")
	}

	// TTT-INV-3: Bounds & no-overwrite
	if req.Position < 0 || req.Position > 8 {
		return invalidResult(req.Board, "", expectedTurn, "ERR_OUT_OF_BOUNDS")
	}
	if req.Board[req.Position] != '.' {
		return invalidResult(req.Board, "", expectedTurn, "ERR_CELL_OCCUPIED")
	}

	// Apply immutable transition
	buf := []byte(req.Board)
	buf[req.Position] = req.Player[0]
	nextBoard := string(buf)

	if w := detectWinner(nextBoard); w != "" {
		return MoveResult{
			Valid:    true,
			Board:    nextBoard,
			Status:   "WIN",
			Winner:   w,
			NextTurn: "",
			Reason:   "GAME_WON",
		}
	}
	if xCount+oCount+1 == 9 {
		return MoveResult{
			Valid:    true,
			Board:    nextBoard,
			Status:   "DRAW",
			Winner:   "",
			NextTurn: "",
			Reason:   "GAME_DRAW",
		}
	}

	nextTurn := "O"
	if req.Player == "O" {
		nextTurn = "X"
	}
	return MoveResult{
		Valid:    true,
		Board:    nextBoard,
		Status:   "IN_PROGRESS",
		Winner:   "",
		NextTurn: nextTurn,
		Reason:   "MOVE_ACCEPTED",
	}
}

func invalidResult(board, winner, nextTurn, reason string) MoveResult {
	return MoveResult{
		Valid:    false,
		Board:    board,
		Status:   "INVALID",
		Winner:   winner,
		NextTurn: nextTurn,
		Reason:   reason,
	}
}

func detectWinner(b string) string {
	lines := [8][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
		{0, 4, 8}, {2, 4, 6},
	}
	for _, ln := range lines {
		c := b[ln[0]]
		if c != '.' && c == b[ln[1]] && c == b[ln[2]] {
			return string(c)
		}
	}
	return ""
}
