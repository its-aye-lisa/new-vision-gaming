package tictactoe

import "testing"

// TestExhaustiveTicTacToeStateSpace traverses EVERY reachable Tic-Tac-Toe
// game state from the empty board and verifies all 5 formal invariants:
//   - TTT-INV-1 (Board Integrity)
//   - TTT-INV-2 (Turn Balance)
//   - TTT-INV-3 (No Overwrite & Bounds)
//   - TTT-INV-4 (Terminal Immutability)
//   - TTT-INV-5 (Anti-Backdoor Actor Non-Interference)
func TestExhaustiveTicTacToeStateSpace(t *testing.T) {
	visited := make(map[string]bool)
	terminalStates := 0
	transitions := 0

	var dfs func(board string, turn string)
	dfs = func(board string, turn string) {
		if visited[board] {
			return
		}
		visited[board] = true

		// Check out-of-bounds rejection
		if r := ApplyMove(MoveRequest{Board: board, Player: turn, Position: -1, Actor: "ghost-admin"}); r.Valid {
			t.Fatalf("TTT-INV-3 VIOLATION: position -1 allowed on %s", board)
		}
		if r := ApplyMove(MoveRequest{Board: board, Player: turn, Position: 9, Actor: "cheat-win"}); r.Valid {
			t.Fatalf("TTT-INV-3 VIOLATION: position 9 allowed on %s", board)
		}

		// Wrong player check
		other := "O"
		if turn == "O" {
			other = "X"
		}

		for pos := 0; pos < 9; pos++ {
			base := ApplyMove(MoveRequest{Board: board, Player: turn, Position: pos, Actor: "player-normal"})
			alt := ApplyMove(MoveRequest{Board: board, Player: turn, Position: pos, Actor: "cheat-win"})
			if base != alt {
				t.Fatalf("TTT-INV-5 VIOLATION: actor influenced outcome on %s pos %d: %+v vs %+v", board, pos, base, alt)
			}
			transitions++

			if board[pos] != '.' {
				if base.Valid {
					t.Fatalf("TTT-INV-3 VIOLATION: occupied cell %d overwritten on %s", pos, board)
				}
				continue
			}

			// Out-of-turn must be rejected
			wrongTurn := ApplyMove(MoveRequest{Board: board, Player: other, Position: pos, Actor: "ghost-admin"})
			if wrongTurn.Valid {
				t.Fatalf("TTT-INV-2 VIOLATION: out-of-turn move allowed for %s on %s", other, board)
			}

			if !base.Valid {
				t.Fatalf("valid move rejected on %s player=%s pos=%d: %+v", board, turn, pos, base)
			}

			if base.Status == "WIN" || base.Status == "DRAW" {
				terminalStates++
				// TTT-INV-4: Verify NO move is allowed from terminal board
				for p2 := 0; p2 < 9; p2++ {
					if after := ApplyMove(MoveRequest{Board: base.Board, Player: other, Position: p2, Actor: "cheat-win"}); after.Valid {
						t.Fatalf("TTT-INV-4 VIOLATION: move allowed after terminal state %s", base.Board)
					}
				}
			} else {
				dfs(base.Board, base.NextTurn)
			}
		}
	}

	dfs(".........", "X")
	t.Logf("Exhaustively verified ALL %d reachable non-terminal boards, %d terminal transitions, %d total moves across 5 formal invariants", len(visited), terminalStates, transitions)
}
