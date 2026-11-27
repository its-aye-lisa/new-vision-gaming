package mexicantrain_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"new-vision-gaming/app/pkg/mexicantrain"
)

type specVector struct {
	ID          string                      `json:"id"`
	Description string                      `json:"description"`
	Input       mexicantrain.SpecMoveInput  `json:"input"`
	Expected    mexicantrain.SpecMoveResult `json:"expected"`
}

type specFile struct {
	AcceptanceVectors []specVector `json:"acceptanceVectors"`
}

func canonicalVectors() []specVector {
	return []specVector{
		{
			ID: "MT-V01", Description: "Legal opening play on own train matching engine 9",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 9, MexOpenEnd: 9, OppOpenEnd: 9, OppPublic: false, Action: "PLAY", TargetTrain: 0, TileLeft: 9, TileRight: 4, HandCount: 12, BoneyardCount: 30, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 4, NextTurn: 1, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "MOVE_ACCEPTED"},
		},
		{
			ID: "MT-V02", Description: "Legal flipped-orientation play on Mexican Train",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 9, OppOpenEnd: 9, OppPublic: false, Action: "PLAY", TargetTrain: -1, TileLeft: 2, TileRight: 9, HandCount: 11, BoneyardCount: 30, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 2, NextTurn: 1, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "MOVE_ACCEPTED"},
		},
		{
			ID: "MT-V03", Description: "Reject play on opponent private train",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: false, Action: "PLAY", TargetTrain: 1, TileLeft: 7, TileRight: 3, HandCount: 10, BoneyardCount: 30, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: false, NewOpenEnd: 7, NextTurn: 0, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "ERR_TRAIN_NOT_PUBLIC"},
		},
		{
			ID: "MT-V04", Description: "Allow play on opponent public train",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: true, Action: "PLAY", TargetTrain: 1, TileLeft: 3, TileRight: 7, HandCount: 10, BoneyardCount: 30, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 3, NextTurn: 1, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "MOVE_ACCEPTED"},
		},
		{
			ID: "MT-V05", Description: "Double grants bonus turn & sets openDoubleTrain",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 6, MexOpenEnd: 9, OppOpenEnd: 9, OppPublic: false, Action: "PLAY", TargetTrain: 0, TileLeft: 6, TileRight: 6, HandCount: 8, BoneyardCount: 25, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 6, NextTurn: 0, NewOpenDouble: 0, OwnPublic: false, HandOver: false, Reason: "DOUBLE_PLAYED_BONUS_TURN"},
		},
		{
			ID: "MT-V06", Description: "Reject other trains while openDoubleTrain active",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: 0, OwnOpenEnd: 6, MexOpenEnd: 9, OppOpenEnd: 9, OppPublic: true, Action: "PLAY", TargetTrain: -1, TileLeft: 9, TileRight: 1, HandCount: 7, BoneyardCount: 25, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: false, NewOpenEnd: 9, NextTurn: 0, NewOpenDouble: 0, OwnPublic: false, HandOver: false, Reason: "ERR_MUST_SATISFY_DOUBLE"},
		},
		{
			ID: "MT-V07", Description: "Satisfying open double clears obligation",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: 0, OwnOpenEnd: 6, MexOpenEnd: 9, OppOpenEnd: 9, OppPublic: false, Action: "PLAY", TargetTrain: 0, TileLeft: 6, TileRight: 2, HandCount: 7, BoneyardCount: 25, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 2, NextTurn: 1, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "DOUBLE_SATISFIED"},
		},
		{
			ID: "MT-V08", Description: "Playing final tile wins hand immediately",
			Input:    mexicantrain.SpecMoveInput{Turn: 1, OpenDoubleTrain: -2, OwnOpenEnd: 5, MexOpenEnd: 9, OppOpenEnd: 4, OppPublic: false, Action: "PLAY", TargetTrain: 1, TileLeft: 5, TileRight: 5, HandCount: 1, BoneyardCount: 10, DrawnThisTurn: false, Actor: "p1"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 5, NextTurn: 1, NewOpenDouble: -2, OwnPublic: false, HandOver: true, Reason: "HAND_WON"},
		},
		{
			ID: "MT-V09", Description: "Pass after draw marks own train public",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: false, Action: "PASS", TargetTrain: 0, TileLeft: -1, TileRight: -1, HandCount: 9, BoneyardCount: 20, DrawnThisTurn: true, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 4, NextTurn: 1, NewOpenDouble: -2, OwnPublic: true, HandOver: false, Reason: "TURN_PASSED_TRAIN_OPENED"},
		},
		{
			ID: "MT-V10", Description: "Reject pass before drawing when boneyard non-empty",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: false, Action: "PASS", TargetTrain: 0, TileLeft: -1, TileRight: -1, HandCount: 9, BoneyardCount: 20, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: false, NewOpenEnd: 4, NextTurn: 0, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "ERR_MUST_DRAW_FIRST"},
		},
		{
			ID: "MT-V11", Description: "Reject pip mismatch",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: false, Action: "PLAY", TargetTrain: 0, TileLeft: 8, TileRight: 3, HandCount: 9, BoneyardCount: 20, DrawnThisTurn: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: false, NewOpenEnd: 4, NextTurn: 0, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "ERR_PIP_MISMATCH"},
		},
		{
			ID: "MT-V12", Description: "Anti-backdoor probe: cheat-win actor cannot bypass pip mismatch",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: false, Action: "PLAY", TargetTrain: 0, TileLeft: 8, TileRight: 3, HandCount: 9, BoneyardCount: 20, DrawnThisTurn: false, Actor: "cheat-win"},
			Expected: mexicantrain.SpecMoveResult{Valid: false, NewOpenEnd: 4, NextTurn: 0, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "ERR_PIP_MISMATCH"},
		},
		{
			ID: "MT-V13", Description: "Reject DRAW when player holds at least one legal move",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 9, MexOpenEnd: 9, OppOpenEnd: 9, OppPublic: false, Action: "DRAW", TargetTrain: 0, TileLeft: -1, TileRight: -1, HandCount: 12, BoneyardCount: 30, DrawnThisTurn: false, HasLegalMove: true, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: false, NewOpenEnd: 9, NextTurn: 0, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "ERR_LEGAL_MOVE_AVAILABLE"},
		},
		{
			ID: "MT-V14", Description: "Reject PASS when player holds at least one legal move",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: false, Action: "PASS", TargetTrain: 0, TileLeft: -1, TileRight: -1, HandCount: 10, BoneyardCount: 19, DrawnThisTurn: true, HasLegalMove: true, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: false, NewOpenEnd: 4, NextTurn: 0, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "ERR_LEGAL_MOVE_AVAILABLE"},
		},
		{
			ID: "MT-V15", Description: "Allow DRAW when player has no legal move and boneyard non-empty",
			Input:    mexicantrain.SpecMoveInput{Turn: 0, OpenDoubleTrain: -2, OwnOpenEnd: 4, MexOpenEnd: 2, OppOpenEnd: 7, OppPublic: false, Action: "DRAW", TargetTrain: 0, TileLeft: -1, TileRight: -1, HandCount: 9, BoneyardCount: 20, DrawnThisTurn: false, HasLegalMove: false, Actor: "p0"},
			Expected: mexicantrain.SpecMoveResult{Valid: true, NewOpenEnd: 4, NextTurn: 0, NewOpenDouble: -2, OwnPublic: false, HandOver: false, Reason: "TILE_DRAWN"},
		},
	}
}

func formatSpecResult(r mexicantrain.SpecMoveResult) string {
	v := "INVALID"
	if r.Valid {
		v = "VALID"
	}
	pub := 0
	if r.OwnPublic {
		pub = 1
	}
	ho := 0
	if r.HandOver {
		ho = 1
	}
	return fmt.Sprintf("%s %d %d %d %d %d %s", v, r.NewOpenEnd, r.NextTurn, r.NewOpenDouble, pub, ho, r.Reason)
}

// TestHermeticSpecVectors always runs even inside isolated build-only enclaves (Gate 3).
func TestHermeticSpecVectors(t *testing.T) {
	for _, vec := range canonicalVectors() {
		got := formatSpecResult(mexicantrain.EvalSpecMove(vec.Input))
		want := formatSpecResult(vec.Expected)
		if got != want {
			t.Errorf("%s (%s): got %q want %q", vec.ID, vec.Description, got, want)
		}
	}
}

// TestCleanRoom3WayEquivalence verifies on-disk spec/mexican-train.spec.json and 3-way
// Go vs GCC vs Clang equivalence when the full repository tree is mounted.
func TestCleanRoom3WayEquivalence(t *testing.T) {
	specPath := filepath.Join("..", "..", "..", "spec", "mexican-train.spec.json")
	cSrc := filepath.Join("..", "..", "..", "cleanroom", "c-verifier", "mexicantrain_kernel.c")
	if _, err := os.Stat(specPath); err != nil {
		t.Logf("Skipping external cleanroom file check in isolated app-only hermetic enclave (%v)", err)
		return
	}
	if _, err := os.Stat(cSrc); err != nil {
		t.Fatalf("cleanroom C source missing: %v", err)
	}

	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var sf specFile
	if err := json.Unmarshal(raw, &sf); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if len(sf.AcceptanceVectors) < 12 {
		t.Fatalf("expected >= 12 acceptance vectors, got %d", len(sf.AcceptanceVectors))
	}

	tmpDir := t.TempDir()
	bins := map[string]string{}
	for _, cc := range []string{"gcc", "clang"} {
		if _, err := exec.LookPath(cc); err != nil {
			t.Fatalf("required clean-room compiler %s not found: %v", cc, err)
		}
		outBin := filepath.Join(tmpDir, "mt_"+cc)
		cmd := exec.Command(cc, "-O2", "-Wall", "-Wextra", "-Werror", cSrc, "-o", outBin)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s compile failed: %v\n%s", cc, err, string(out))
		}
		bins[cc] = outBin
	}

	runC := func(bin string, in mexicantrain.SpecMoveInput) string {
		oppPub := "0"
		if in.OppPublic {
			oppPub = "1"
		}
		drawn := "0"
		if in.DrawnThisTurn {
			drawn = "1"
		}
		hasLegal := "0"
		if in.HasLegalMove {
			hasLegal = "1"
		}
		actor := in.Actor
		if actor == "" {
			actor = "anon"
		}
		out, err := exec.Command(bin,
			strconv.Itoa(in.Turn),
			strconv.Itoa(in.OpenDoubleTrain),
			strconv.Itoa(in.OwnOpenEnd),
			strconv.Itoa(in.MexOpenEnd),
			strconv.Itoa(in.OppOpenEnd),
			oppPub,
			in.Action,
			strconv.Itoa(in.TargetTrain),
			strconv.Itoa(in.TileLeft),
			strconv.Itoa(in.TileRight),
			strconv.Itoa(in.HandCount),
			strconv.Itoa(in.BoneyardCount),
			drawn,
			actor,
			hasLegal,
		).CombinedOutput()
		if err != nil {
			t.Fatalf("c verifier %s error: %v\n%s", bin, err, string(out))
		}
		return strings.TrimSpace(string(out))
	}

	for _, vec := range sf.AcceptanceVectors {
		gotGo := formatSpecResult(mexicantrain.EvalSpecMove(vec.Input))
		want := formatSpecResult(vec.Expected)
		if gotGo != want {
			t.Errorf("%s (%s): Go=%q != spec=%q", vec.ID, vec.Description, gotGo, want)
		}
		for cc, bin := range bins {
			gotC := runC(bin, vec.Input)
			if gotC != want || gotC != gotGo {
				t.Errorf("%s: 3-way divergence %s=%q Go=%q spec=%q", vec.ID, cc, gotC, gotGo, want)
			}
		}
	}

	actions := []string{"PLAY", "PASS", "DRAW", "BOGUS"}
	actors := []string{"p0", "p1", "cheat-win", "admin"}
	targets := []int{-1, 0, 1, 2}
	for i := 0; i < 100; i++ {
		in := mexicantrain.SpecMoveInput{
			Turn:            i % 2,
			OpenDoubleTrain: []int{-2, -2, -2, -1, 0, 1}[i%6],
			OwnOpenEnd:      (i * 3) % 13,
			MexOpenEnd:      (i * 5) % 13,
			OppOpenEnd:      (i * 7) % 13,
			OppPublic:       i%3 == 0,
			Action:          actions[i%len(actions)],
			TargetTrain:     targets[i%len(targets)],
			TileLeft:        (i * 3) % 13,
			TileRight:       (i * 4) % 13,
			HandCount:       []int{0, 1, 2, 7, 12}[i%5],
			BoneyardCount:   []int{0, 1, 15}[i%3],
			DrawnThisTurn:   i%2 == 1,
			HasLegalMove:    i%4 == 0,
			Actor:           actors[i%len(actors)],
		}
		gotGo := formatSpecResult(mexicantrain.EvalSpecMove(in))
		for cc, bin := range bins {
			gotC := runC(bin, in)
			if gotC != gotGo {
				t.Fatalf("DIFF-%03d: Go=%q != %s=%q (input=%+v)", i, gotGo, cc, gotC, in)
			}
		}
	}
}

func TestStatefulCannotDrawOrPassWithLegalMove(t *testing.T) {
	st := mexicantrain.NewMatch(9, 2, 42)
	// Force deterministic hand where Player 0 holds a matching tile for engine 9
	st.Hands[0] = []mexicantrain.Tile{{Left: 9, Right: 4}, {Left: 2, Right: 1}}
	st.Boneyard = []mexicantrain.Tile{{Left: 3, Right: 5}}

	resDraw := mexicantrain.ApplyAction(st, mexicantrain.ActionRequest{Player: 0, Action: "DRAW"})
	if resDraw.Valid || resDraw.Reason != "ERR_LEGAL_MOVE_AVAILABLE" {
		t.Fatalf("expected DRAW rejected with ERR_LEGAL_MOVE_AVAILABLE, got valid=%v reason=%s", resDraw.Valid, resDraw.Reason)
	}
	if len(resDraw.State.Boneyard) != 1 || len(resDraw.State.Hands[0]) != 2 {
		t.Fatalf("failed DRAW mutated state: boneyard=%d hand=%d", len(resDraw.State.Boneyard), len(resDraw.State.Hands[0]))
	}

	// Even if DrawnThisTurn is true (e.g. just drew a playable tile), PASS must be rejected
	stDrawnPlayable := mexicantrain.CloneState(st)
	stDrawnPlayable.DrawnThisTurn = true
	resPass := mexicantrain.ApplyAction(stDrawnPlayable, mexicantrain.ActionRequest{Player: 0, Action: "PASS"})
	if resPass.Valid || resPass.Reason != "ERR_LEGAL_MOVE_AVAILABLE" {
		t.Fatalf("expected PASS rejected with ERR_LEGAL_MOVE_AVAILABLE, got valid=%v reason=%s", resPass.Valid, resPass.Reason)
	}

	// Now give Player 0 zero legal plays (engine 9, hand has only [2|1])
	stBlocked := mexicantrain.CloneState(st)
	stBlocked.Hands[0] = []mexicantrain.Tile{{Left: 2, Right: 1}}
	resDrawOK := mexicantrain.ApplyAction(stBlocked, mexicantrain.ActionRequest{Player: 0, Action: "DRAW"})
	if !resDrawOK.Valid || resDrawOK.Reason != "TILE_DRAWN" {
		t.Fatalf("expected DRAW allowed when blocked, got valid=%v reason=%s", resDrawOK.Valid, resDrawOK.Reason)
	}
	// Drawn tile [3|5] still does not match engine 9 -> PASS must succeed and open train
	resPassOK := mexicantrain.ApplyAction(resDrawOK.State, mexicantrain.ActionRequest{Player: 0, Action: "PASS"})
	if !resPassOK.Valid || resPassOK.Reason != "TURN_PASSED_TRAIN_OPENED" || !resPassOK.State.PlayerTrains[0].Public {
		t.Fatalf("expected PASS allowed after unplayable draw, got valid=%v reason=%s public=%v", resPassOK.Valid, resPassOK.Reason, resPassOK.State.PlayerTrains[0].Public)
	}
}

func TestFullMultiHandDouble9AndDouble12Invariants(t *testing.T) {
	for _, maxPip := range []int{9, 12} {
		for seed := uint64(1); seed <= 25; seed++ {
			st := mexicantrain.NewMatch(maxPip, 2, seed*1337)
			if msg := mexicantrain.VerifyInvariants(st); msg != "" {
				t.Fatalf("Double-%d seed=%d initial invariant failure: %s", maxPip, seed, msg)
			}
			handsPlayed := 0
			for !st.MatchOver {
				steps := 0
				for !st.HandOver {
					move := mexicantrain.ChooseBotMove(st)
					res := mexicantrain.ApplyAction(st, move)
					if !res.Valid {
						t.Fatalf("Double-%d seed=%d engine=%d bot produced invalid move %+v: %s", maxPip, seed, st.Engine, move, res.Reason)
					}
					st = res.State
					if msg := mexicantrain.VerifyInvariants(st); msg != "" {
						t.Fatalf("Double-%d seed=%d engine=%d step=%d invariant failed: %s", maxPip, seed, st.Engine, steps, msg)
					}
					steps++
					if steps > 500 {
						t.Fatalf("Double-%d seed=%d engine=%d exceeded 500 steps (non-terminating hand)", maxPip, seed, st.Engine)
					}
				}
				handsPlayed++
				if !st.MatchOver {
					st = mexicantrain.NewHand(maxPip, st.Engine-1, 2, seed*99991+uint64(st.Engine), st.Scores, handsPlayed%2)
					if msg := mexicantrain.VerifyInvariants(st); msg != "" {
						t.Fatalf("Double-%d seed=%d next hand invariant failed: %s", maxPip, seed, msg)
					}
				}
			}
			expectedHands := maxPip + 1
			if handsPlayed != expectedHands {
				t.Fatalf("Double-%d expected %d hands, played %d", maxPip, expectedHands, handsPlayed)
			}
			if st.MatchWinner != 0 && st.MatchWinner != 1 {
				t.Fatalf("invalid match winner: %d", st.MatchWinner)
			}
		}
	}
}
