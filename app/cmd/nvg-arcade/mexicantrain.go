package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"new-vision-gaming/app/pkg/mexicantrain"
)

const (
	mtCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	mtMaxRooms     = 500
	mtRoomTTL      = 2 * time.Hour
)

type mtRoom struct {
	Code       string
	Mode       string // "bot" or "2p"
	SeatTokens [2]string
	SeatJoined [2]bool
	State      mexicantrain.State
	Log        []string
	UpdatedAt  time.Time
}

type mtStore struct {
	mu    sync.Mutex
	rooms map[string]*mtRoom
}

func newMTStore() *mtStore {
	return &mtStore{rooms: make(map[string]*mtRoom)}
}

func cryptoSeedUint64() uint64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	v := binary.LittleEndian.Uint64(b[:])
	if v == 0 {
		return 0x9e3779b97f4a7c15
	}
	return v
}

func cryptoTokenHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func cryptoRoomCode() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	var sb strings.Builder
	sb.WriteString("MEX-")
	for _, v := range b {
		sb.WriteByte(mtCodeAlphabet[int(v)%len(mtCodeAlphabet)])
	}
	return sb.String()
}

func (s *mtStore) pruneLocked(now time.Time) {
	for code, rm := range s.rooms {
		if now.Sub(rm.UpdatedAt) > mtRoomTTL {
			delete(s.rooms, code)
		}
	}
	if len(s.rooms) >= mtMaxRooms {
		var oldestCode string
		var oldestTime time.Time
		for code, rm := range s.rooms {
			if oldestCode == "" || rm.UpdatedAt.Before(oldestTime) {
				oldestCode = code
				oldestTime = rm.UpdatedAt
			}
		}
		if oldestCode != "" {
			delete(s.rooms, oldestCode)
		}
	}
}

func (s *mtStore) createRoom(maxPip int, mode string) (*mtRoom, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.pruneLocked(now)

	code := cryptoRoomCode()
	for _, exists := s.rooms[code]; exists; _, exists = s.rooms[code] {
		code = cryptoRoomCode()
	}

	hostTok := cryptoTokenHex(16)
	guestTok := cryptoTokenHex(16)
	rm := &mtRoom{
		Code:       code,
		Mode:       mode,
		SeatTokens: [2]string{hostTok, guestTok},
		SeatJoined: [2]bool{true, mode == "bot"},
		State:      mexicantrain.NewMatch(maxPip, 2, cryptoSeedUint64()),
		Log:        []string{fmt.Sprintf("Started Double-%d match (Engine [%d|%d])", maxPip, maxPip, maxPip)},
		UpdatedAt:  now,
	}
	s.rooms[code] = rm
	return rm, hostTok
}

func (s *mtStore) get(code string) *mtRoom {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rooms[strings.ToUpper(strings.TrimSpace(code))]
}

type mtPlayOptionVM struct {
	TargetTrain int
	Label       string
}

type mtHandTileVM struct {
	Index   int
	Left    int
	Right   int
	Double  bool
	Options []mtPlayOptionVM
}

type mtTrainVM struct {
	ID          int
	Title       string
	OpenEnd     int
	Public      bool
	NeedsDouble bool
	Tiles       []mexicantrain.Tile
	TileCount   int
}

type mtBoardVM struct {
	RoomCode          string
	Mode              string
	SharePath         string
	MaxPip            int
	Engine            int
	HandNumber        int
	TotalHands        int
	ViewerSeat        int // 0, 1, or -1 (spectator)
	WaitingForPlayer2 bool
	IsMyTurn          bool
	TurnLabel         string
	OpenDoubleLabel   string
	BoneyardCount     int
	DrawnThisTurn     bool
	CanDraw           bool
	CanPass           bool
	HandOver          bool
	MatchOver         bool
	WinnerBanner      string
	MyScore           int
	OppScore          int
	MyName            string
	OppName           string
	OppTileCount      int
	MyHand            []mtHandTileVM
	Trains            []mtTrainVM
	RecentLog         []string
	Notice            string
	WebAdsEnabled     bool
	IsNativeShell     bool
	AdSenseClient     string
}

func mtSeatCookieName(code string) string {
	return "nvg_mt_" + strings.ReplaceAll(strings.ToUpper(code), "-", "_")
}

func setSeatCookie(w http.ResponseWriter, r *http.Request, code, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     mtSeatCookieName(code),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
		MaxAge:   7200,
	})
}

func (rm *mtRoom) viewerSeatForRequest(r *http.Request) int {
	if rm.Mode == "bot" {
		return 0
	}
	if c, err := r.Cookie(mtSeatCookieName(rm.Code)); err == nil {
		if c.Value == rm.SeatTokens[0] {
			return 0
		}
		if c.Value == rm.SeatTokens[1] {
			return 1
		}
	}
	return -1
}

func (rm *mtRoom) appendLog(entry string) {
	rm.Log = append(rm.Log, entry)
	if len(rm.Log) > 6 {
		rm.Log = rm.Log[len(rm.Log)-6:]
	}
}

func (rm *mtRoom) runBotTurnsIfNeeded() {
	if rm.Mode != "bot" {
		return
	}
	for !rm.State.HandOver && rm.State.Turn == 1 {
		mv := mexicantrain.ChooseBotMove(rm.State)
		var desc string
		if mv.Action == "PLAY" && mv.TileIndex >= 0 && mv.TileIndex < len(rm.State.Hands[1]) {
			t := rm.State.Hands[1][mv.TileIndex]
			tName := "Bot Train"
			if mv.TargetTrain == -1 {
				tName = "Mexican Train"
			} else if mv.TargetTrain == 0 {
				tName = "Your Train"
			}
			desc = fmt.Sprintf("Bot played [%d|%d] on %s", t.Left, t.Right, tName)
		} else if mv.Action == "DRAW" {
			desc = "Bot drew from boneyard"
		} else {
			desc = "Bot passed (Bot Train is now OPEN)"
		}
		res := mexicantrain.ApplyAction(rm.State, mv)
		if !res.Valid {
			break
		}
		rm.State = res.State
		rm.appendLog(desc)
		if res.Reason == "HAND_WON" {
			rm.appendLog("Bot went out and won the hand!")
		} else if res.Reason == "HAND_BLOCKED" {
			rm.appendLog("Hand blocked — scored by remaining pips.")
		}
	}
}

func buildMTBoardVM(rm *mtRoom, viewer int, notice string, webAds, isNative bool, adsClient string) mtBoardVM {
	st := rm.State
	mySeat := viewer
	if mySeat < 0 {
		mySeat = 0
	}
	oppSeat := 1 - mySeat

	myName := fmt.Sprintf("Player %d (You)", mySeat+1)
	oppName := fmt.Sprintf("Player %d", oppSeat+1)
	if rm.Mode == "bot" {
		myName = "You"
		oppName = "NVG Bot"
	}

	trainLabel := func(id int) string {
		if id == -1 {
			return "Mexican"
		}
		if id == mySeat {
			return "Your Train"
		}
		return oppName
	}

	var openDoubleLabel string
	if st.OpenDoubleTrain != -2 {
		openDoubleLabel = trainLabel(st.OpenDoubleTrain)
	}

	isMyTurn := viewer >= 0 && !st.HandOver && st.Turn == viewer && rm.SeatJoined[1]
	turnLabel := oppName + "'s Turn"
	if st.HandOver {
		turnLabel = "Hand Complete"
	} else if !rm.SeatJoined[1] {
		turnLabel = "Waiting for Player 2 to join..."
	} else if isMyTurn {
		turnLabel = "Your Turn"
	}

	// Build legal play options for viewer's own hidden hand ONLY (never exposes opponent hand!)
	var handVM []mtHandTileVM
	if viewer >= 0 && viewer < len(st.Hands) {
		legal := mexicantrain.LegalMoves(st)
		byTile := make(map[int][]mtPlayOptionVM)
		if isMyTurn {
			for _, m := range legal {
				byTile[m.TileIndex] = append(byTile[m.TileIndex], mtPlayOptionVM{
					TargetTrain: m.TargetTrain,
					Label:       trainLabel(m.TargetTrain),
				})
			}
		}
		for i, t := range st.Hands[viewer] {
			handVM = append(handVM, mtHandTileVM{
				Index:   i,
				Left:    t.Left,
				Right:   t.Right,
				Double:  t.IsDouble(),
				Options: byTile[i],
			})
		}
	}

	tailTiles := func(tiles []mexicantrain.Tile) []mexicantrain.Tile {
		if len(tiles) <= 6 {
			return tiles
		}
		return tiles[len(tiles)-6:]
	}

	trains := []mtTrainVM{
		{
			ID:          mySeat,
			Title:       myName + " Train",
			OpenEnd:     st.PlayerTrains[mySeat].OpenEnd,
			Public:      st.PlayerTrains[mySeat].Public,
			NeedsDouble: st.OpenDoubleTrain == mySeat,
			Tiles:       tailTiles(st.PlayerTrains[mySeat].Tiles),
			TileCount:   len(st.PlayerTrains[mySeat].Tiles),
		},
		{
			ID:          -1,
			Title:       "Mexican Train (Always Public)",
			OpenEnd:     st.MexicanTrain.OpenEnd,
			Public:      true,
			NeedsDouble: st.OpenDoubleTrain == -1,
			Tiles:       tailTiles(st.MexicanTrain.Tiles),
			TileCount:   len(st.MexicanTrain.Tiles),
		},
		{
			ID:          oppSeat,
			Title:       oppName + " Train",
			OpenEnd:     st.PlayerTrains[oppSeat].OpenEnd,
			Public:      st.PlayerTrains[oppSeat].Public,
			NeedsDouble: st.OpenDoubleTrain == oppSeat,
			Tiles:       tailTiles(st.PlayerTrains[oppSeat].Tiles),
			TileCount:   len(st.PlayerTrains[oppSeat].Tiles),
		},
	}

	var winnerBanner string
	if st.MatchOver {
		if st.MatchWinner == mySeat {
			winnerBanner = fmt.Sprintf("Match Complete — %s Won! (Lowest Score: %d vs %d)", myName, st.Scores[mySeat], st.Scores[oppSeat])
		} else {
			winnerBanner = fmt.Sprintf("Match Complete — %s Won! (Score: %d vs %d)", oppName, st.Scores[oppSeat], st.Scores[mySeat])
		}
	} else if st.HandOver {
		if st.HandWinner == mySeat {
			winnerBanner = fmt.Sprintf("Hand %d Complete — %s Won the Hand!", st.MaxPip-st.Engine+1, myName)
		} else {
			winnerBanner = fmt.Sprintf("Hand %d Complete — %s Won the Hand!", st.MaxPip-st.Engine+1, oppName)
		}
	}

	canDraw := isMyTurn && !st.DrawnThisTurn && len(st.Boneyard) > 0
	canPass := isMyTurn && (st.DrawnThisTurn || len(st.Boneyard) == 0)

	return mtBoardVM{
		RoomCode:          rm.Code,
		Mode:              rm.Mode,
		SharePath:         "/play/" + rm.Code,
		MaxPip:            st.MaxPip,
		Engine:            st.Engine,
		HandNumber:        st.MaxPip - st.Engine + 1,
		TotalHands:        st.MaxPip + 1,
		ViewerSeat:        viewer,
		WaitingForPlayer2: rm.Mode == "2p" && !rm.SeatJoined[1],
		IsMyTurn:          isMyTurn,
		TurnLabel:         turnLabel,
		OpenDoubleLabel:   openDoubleLabel,
		BoneyardCount:     len(st.Boneyard),
		DrawnThisTurn:     st.DrawnThisTurn,
		CanDraw:           canDraw,
		CanPass:           canPass,
		HandOver:          st.HandOver,
		MatchOver:         st.MatchOver,
		WinnerBanner:      winnerBanner,
		MyScore:           st.Scores[mySeat],
		OppScore:          st.Scores[oppSeat],
		MyName:            myName,
		OppName:           oppName,
		OppTileCount:      len(st.Hands[oppSeat]),
		MyHand:            handVM,
		Trains:            trains,
		RecentLog:         append([]string(nil), rm.Log...),
		Notice:            notice,
		WebAdsEnabled:     webAds,
		IsNativeShell:     isNative,
		AdSenseClient:     adsClient,
	}
}

func registerMexicanTrainRoutes(
	mux *http.ServeMux,
	tmpl *template.Template,
	newVM func(r *http.Request, title, view, board, turn, status, winner, notice string) pageViewModel,
	fipsActive bool,
	shortDigest string,
) {
	store := newMTStore()

	renderPage := func(w http.ResponseWriter, r *http.Request, rm *mtRoom, viewer int, notice string) {
		vm := newVM(r, "Mexican Train Dominoes", "mexican-train", ".........", "X", "IN_PROGRESS", "", notice)
		if rm != nil {
			bvm := buildMTBoardVM(rm, viewer, notice, vm.WebAdsEnabled, vm.IsNativeShell, vm.AdSenseClient)
			vm.MT = &bvm
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "layout.tmpl", vm)
	}

	renderBoardFragment := func(w http.ResponseWriter, r *http.Request, rm *mtRoom, viewer int, notice string) {
		vm := newVM(r, "Mexican Train Dominoes", "mexican-train", ".........", "X", "IN_PROGRESS", "", notice)
		bvm := buildMTBoardVM(rm, viewer, notice, vm.WebAdsEnabled, vm.IsNativeShell, vm.AdSenseClient)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "mexicantrain_board.tmpl", bvm)
	}

	// Flagship Game #1 Landing Page: /games/mexican-train
	mux.HandleFunc("/games/mexican-train", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		code := strings.TrimSpace(r.URL.Query().Get("room"))
		if code != "" {
			if rm := store.get(code); rm != nil {
				store.mu.Lock()
				viewer := rm.viewerSeatForRequest(r)
				store.mu.Unlock()
				renderPage(w, r, rm, viewer, "")
				return
			}
		}
		renderPage(w, r, nil, -1, "")
	})

	// Canonical 2-Player 6-Letter Invite Deep Link: /play/MEX-XXXX
	mux.HandleFunc("/play/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		code := strings.ToUpper(strings.TrimPrefix(r.URL.Path, "/play/"))
		code = strings.Trim(code, "/")
		rm := store.get(code)
		if rm == nil {
			renderPage(w, r, nil, -1, "Room "+code+" was not found or expired. Start a fresh room below!")
			return
		}
		store.mu.Lock()
		viewer := rm.viewerSeatForRequest(r)
		notice := ""
		if viewer == -1 && rm.Mode == "2p" && !rm.SeatJoined[1] {
			rm.SeatJoined[1] = true
			rm.UpdatedAt = time.Now().UTC()
			rm.appendLog("Player 2 joined via invite link!")
			viewer = 1
			setSeatCookie(w, r, rm.Code, rm.SeatTokens[1])
			notice = "Joined room " + rm.Code + " as Player 2!"
		}
		store.mu.Unlock()
		renderPage(w, r, rm, viewer, notice)
	})

	// Create New Match (Bot or 2-Player Room)
	mux.HandleFunc("/hx/mt/new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		_ = r.ParseForm()
		maxPip := 9
		if r.FormValue("variant") == "12" {
			maxPip = 12
		}
		mode := "bot"
		if r.FormValue("mode") == "2p" {
			mode = "2p"
		}
		rm, hostTok := store.createRoom(maxPip, mode)
		setSeatCookie(w, r, rm.Code, hostTok)
		if r.Header.Get("HX-Request") == "true" {
			store.mu.Lock()
			renderBoardFragment(w, r, rm, 0, "")
			store.mu.Unlock()
			return
		}
		http.Redirect(w, r, "/play/"+rm.Code, http.StatusSeeOther)
	})

	// HTMX Live Room Polling State Fragment
	mux.HandleFunc("/hx/mt/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		rm := store.get(r.URL.Query().Get("room"))
		if rm == nil {
			http.Error(w, "room expired", http.StatusNotFound)
			return
		}
		store.mu.Lock()
		viewer := rm.viewerSeatForRequest(r)
		renderBoardFragment(w, r, rm, viewer, "")
		store.mu.Unlock()
	})

	// HTMX Action Handler (PLAY / DRAW / PASS)
	mux.HandleFunc("/hx/mt/action", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		rm := store.get(r.FormValue("room"))
		if rm == nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		action := strings.ToUpper(strings.TrimSpace(r.FormValue("action")))
		tileIdx, _ := strconv.Atoi(r.FormValue("tileIndex"))
		targetTrain, _ := strconv.Atoi(r.FormValue("targetTrain"))

		store.mu.Lock()
		defer store.mu.Unlock()
		viewer := rm.viewerSeatForRequest(r)
		notice := ""
		if viewer < 0 {
			notice = "Spectators cannot play moves."
		} else {
			req := mexicantrain.ActionRequest{
				Player:      viewer,
				Action:      action,
				TargetTrain: targetTrain,
				TileIndex:   tileIdx,
			}
			var desc string
			if action == "PLAY" && tileIdx >= 0 && tileIdx < len(rm.State.Hands[viewer]) {
				t := rm.State.Hands[viewer][tileIdx]
				desc = fmt.Sprintf("P%d played [%d|%d]", viewer+1, t.Left, t.Right)
			} else if action == "DRAW" {
				desc = fmt.Sprintf("P%d drew from boneyard", viewer+1)
			} else if action == "PASS" {
				desc = fmt.Sprintf("P%d passed (train OPEN)", viewer+1)
			}
			res := mexicantrain.ApplyAction(rm.State, req)
			vCopy := res.Valid
			emitAuditLog(auditEvent{
				EventType:    "MEXICAN_TRAIN_ACTION",
				Protocol:     "HTMX",
				Move:         fmt.Sprintf("%s:%s@%d", rm.Code, action, targetTrain),
				Valid:        &vCopy,
				Reason:       res.Reason,
				FIPS140:      fipsActive,
				BinaryDigest: shortDigest,
				RemoteAddr:   r.RemoteAddr,
			})
			if !res.Valid {
				notice = "Move rejected: " + res.Reason
			} else {
				rm.State = res.State
				rm.UpdatedAt = time.Now().UTC()
				if desc != "" {
					rm.appendLog(desc)
				}
				rm.runBotTurnsIfNeeded()
			}
		}
		renderBoardFragment(w, r, rm, viewer, notice)
	})

	// Deal Next Hand (Engine N-1) Between Natural Round Breaks
	mux.HandleFunc("/hx/mt/next-hand", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		_ = r.ParseForm()
		rm := store.get(r.FormValue("room"))
		if rm == nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		viewer := rm.viewerSeatForRequest(r)
		if rm.State.HandOver && !rm.State.MatchOver {
			nextEngine := rm.State.Engine - 1
			starter := (rm.State.MaxPip - nextEngine) % rm.State.NumPlayers
			rm.State = mexicantrain.NewHand(rm.State.MaxPip, nextEngine, rm.State.NumPlayers, cryptoSeedUint64(), rm.State.Scores, starter)
			rm.UpdatedAt = time.Now().UTC()
			rm.appendLog(fmt.Sprintf("Dealt Hand %d — Engine [%d|%d]", rm.State.MaxPip-nextEngine+1, nextEngine, nextEngine))
			rm.runBotTurnsIfNeeded()
		}
		renderBoardFragment(w, r, rm, viewer, "")
	})

	// Stateless Versioned REST API v1: POST /api/v1/mexican-train/move
	mux.HandleFunc("/api/v1/mexican-train/move", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body exceeds 64 KiB limit"})
			return
		}
		dec := json.NewDecoder(bytes.NewReader(bodyBytes))
		dec.DisallowUnknownFields()
		var req mexicantrain.SpecMoveInput
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid mexican-train move JSON"})
			return
		}
		if dec.More() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unexpected trailing JSON"})
			return
		}
		res := mexicantrain.EvalSpecMove(req)
		vCopy := res.Valid
		emitAuditLog(auditEvent{
			EventType:    "MEXICAN_TRAIN_SPEC_EVAL",
			Protocol:     "REST",
			Actor:        req.Actor,
			Move:         fmt.Sprintf("%s:%d|%d->%d", req.Action, req.TileLeft, req.TileRight, req.TargetTrain),
			Valid:        &vCopy,
			Reason:       res.Reason,
			FIPS140:      fipsActive,
			BinaryDigest: shortDigest,
			RemoteAddr:   r.RemoteAddr,
		})
		writeJSON(w, http.StatusOK, res)
	})
}
