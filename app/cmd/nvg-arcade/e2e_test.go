package main

import (
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func startTestArcade(t *testing.T, adsEnabled bool) *httptest.Server {
	t.Helper()
	h, err := newArcadeHandler(true, false, adsEnabled, "ca-pub-1234567890123456", "testdigest123456", "testdigest1234567890")
	if err != nil {
		t.Fatalf("newArcadeHandler failed: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func newBrowserClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return &http.Client{Jar: jar}
}

func mustReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func assertStrictZeroInlineHTML(t *testing.T, route, body string) {
	t.Helper()
	lower := strings.ToLower(body)
	if strings.Contains(lower, " style=") {
		t.Errorf("%s rendered forbidden inline style= attribute (violates style-src 'self')", route)
	}
	if strings.Contains(lower, " onclick=") {
		t.Errorf("%s rendered forbidden inline onclick= attribute (violates script-src 'self')", route)
	}
}

// TestE2ESuitePagesThemeCSPAndAds verifies light tabletop tokens, active nav state,
// strict CSP, Zero-CLS slots, native shell ad suppression, and static asset serving.
func TestE2ESuitePagesThemeCSPAndAds(t *testing.T) {
	srv := startTestArcade(t, false)
	client := newBrowserClient(t)

	pages := []struct {
		path       string
		activeHref string
	}{
		{"/", `href="/"`},
		{"/games/mexican-train", `href="/games/mexican-train"`},
		{"/games/tictactoe", `href="/games/tictactoe"`},
		{"/privacy", `href="/privacy"`},
	}

	for _, tc := range pages {
		resp, err := client.Get(srv.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status=%d want 200", tc.path, resp.StatusCode)
		}
		csp := resp.Header.Get("Content-Security-Policy")
		if !strings.Contains(csp, "style-src 'self'") || strings.Contains(csp, "unsafe-inline") {
			t.Errorf("GET %s: unexpected CSP %q", tc.path, csp)
		}
		body := mustReadBody(t, resp)
		assertStrictZeroInlineHTML(t, tc.path, body)
		if !strings.Contains(body, `content="#f1f2f4"`) {
			t.Errorf("GET %s: missing light tabletop theme-color #f1f2f4", tc.path)
		}
		if !strings.Contains(body, "/static/nvg-ui.js") {
			t.Errorf("GET %s: missing /static/nvg-ui.js script include", tc.path)
		}
		if !strings.Contains(body, `id="nvg-top-leaderboard"`) || !strings.Contains(body, `id="nvg-sidebar-rect"`) {
			t.Errorf("GET %s: missing Zero-CLS web ad slots", tc.path)
		}
		// Only the active route nav link should carry the active underline highlight
		if count := strings.Count(body, "decoration-accent-primary"); count != 1 {
			t.Errorf("GET %s: expected exactly 1 active nav highlight, got %d", tc.path, count)
		}
		if tc.path == "/" && !strings.Contains(body, "whitespace-nowrap shrink-0") {
			t.Errorf("GET /: expected single-line whitespace-nowrap on LIVE • FLAGSHIP pill")
		}
	}

	// Verify manifest.webmanifest light tabletop colors
	resp, err := client.Get(srv.URL + "/manifest.webmanifest")
	if err != nil {
		t.Fatalf("GET /manifest.webmanifest: %v", err)
	}
	manifest := mustReadBody(t, resp)
	if !strings.Contains(manifest, `"background_color":"#f1f2f4"`) || !strings.Contains(manifest, `"theme_color":"#f1f2f4"`) {
		t.Errorf("manifest.webmanifest does not reflect #f1f2f4 tabletop theme: %s", manifest)
	}

	// Verify static assets load with HTTP 200
	for _, asset := range []string{"/static/tailwind.css", "/static/htmx.min.js", "/static/nvg-ui.js"} {
		r, err := client.Get(srv.URL + asset)
		if err != nil || r.StatusCode != http.StatusOK {
			t.Fatalf("GET %s failed: err=%v status=%v", asset, err, r)
		}
		_ = mustReadBody(t, r)
	}

	// Verify Native iOS shell suppresses web AdSense slots
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/games/mexican-train", nil)
	req.Header.Set("X-NVG-Client", "native-ios")
	nativeResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("native-ios request: %v", err)
	}
	nativeBody := mustReadBody(t, nativeResp)
	if strings.Contains(nativeBody, "nvg-top-leaderboard") || strings.Contains(nativeBody, "nvg-sidebar-rect") {
		t.Errorf("native-ios shell must suppress web AdSense slots")
	}

	// Verify ADS_ENABLED=true injects CSP nonce onto static scripts for strict-dynamic compatibility
	adsSrv := startTestArcade(t, true)
	adsResp, err := client.Get(adsSrv.URL + "/")
	if err != nil {
		t.Fatalf("GET ads-enabled /: %v", err)
	}
	adsCSP := adsResp.Header.Get("Content-Security-Policy")
	adsBody := mustReadBody(t, adsResp)
	if !strings.Contains(adsCSP, "'strict-dynamic'") || !strings.Contains(adsBody, `nonce="`) {
		t.Errorf("ADS_ENABLED=true missing strict-dynamic CSP nonce: csp=%q", adsCSP)
	}
}

// TestE2ETicTacToeFullUserMatch simulates a human playing an entire Tic-Tac-Toe match
// over HTMX, verifying win detection, terminal immutability, and reset.
func TestE2ETicTacToeFullUserMatch(t *testing.T) {
	srv := startTestArcade(t, false)
	client := newBrowserClient(t)

	board := "........."
	moves := []struct {
		player string
		pos    string
		want   string
	}{
		{"X", "0", "X........"},
		{"O", "3", "X..O....."},
		{"X", "1", "XX.O....."},
		{"O", "4", "XX.OO...."},
		{"X", "2", "XXXOO...."},
	}

	var lastBody string
	for _, mv := range moves {
		form := url.Values{"board": {board}, "player": {mv.player}, "position": {mv.pos}, "actor": {"e2e-user"}}
		resp, err := client.PostForm(srv.URL+"/hx/move", form)
		if err != nil {
			t.Fatalf("POST /hx/move: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("POST /hx/move status=%d", resp.StatusCode)
		}
		lastBody = mustReadBody(t, resp)
		assertStrictZeroInlineHTML(t, "/hx/move", lastBody)
		if !strings.Contains(lastBody, `value="`+mv.want+`"`) {
			t.Fatalf("after %s@%s expected board %s in HTML", mv.player, mv.pos, mv.want)
		}
		board = mv.want
	}
	if !strings.Contains(lastBody, "Winner: Player X") {
		t.Fatalf("expected Winner: Player X banner")
	}

	// Terminal immutability: post-win move must be rejected and leave board unchanged
	resp, err := client.PostForm(srv.URL+"/hx/move", url.Values{"board": {board}, "player": {"O"}, "position": {"5"}, "actor": {"e2e-user"}})
	if err != nil {
		t.Fatalf("post-win /hx/move: %v", err)
	}
	postWin := mustReadBody(t, resp)
	if !strings.Contains(postWin, `value="XXXOO...."`) {
		t.Fatalf("terminal board was mutated after win")
	}

	// Reset restores clean playable board
	resetResp, err := client.Post(srv.URL+"/hx/reset", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("POST /hx/reset: %v", err)
	}
	resetBody := mustReadBody(t, resetResp)
	if !strings.Contains(resetBody, `value="........."`) {
		t.Fatalf("reset did not restore empty board")
	}
}

type parsedMTAction struct {
	Room        string `json:"room"`
	Action      string `json:"action"`
	TileIndex   string `json:"tileIndex"`
	TargetTrain string `json:"targetTrain"`
}

var hxValsRe = regexp.MustCompile(`hx-vals='([^']+)'`)

func parseMTActions(t *testing.T, body string) []parsedMTAction {
	t.Helper()
	matches := hxValsRe.FindAllStringSubmatch(body, -1)
	var out []parsedMTAction
	for _, m := range matches {
		raw := html.UnescapeString(m[1])
		var act parsedMTAction
		if err := json.Unmarshal([]byte(raw), &act); err == nil && act.Room != "" {
			out = append(out, act)
		}
	}
	return out
}

// TestE2EMexicanTrainSoloFullHandGameplaySimulation simulates a real user playing both
// Double-9 and Double-12 Mexican Train Solo matches end-to-end via real HTMX endpoints,
// asserting ERR_LEGAL_MOVE_AVAILABLE enforcement, visual domino tiles, polish fixes, and Next Hand.
func TestE2EMexicanTrainSoloFullHandGameplaySimulation(t *testing.T) {
	srv := startTestArcade(t, false)

	for _, tc := range []struct {
		variant       string
		wantNextLabel string
	}{
		{"9", "Deal Next Hand (Engine [8|8])"},
		{"12", "Deal Next Hand (Engine [11|11])"},
	} {
		t.Run("Double-"+tc.variant, func(t *testing.T) {
			client := newBrowserClient(t)

			// Start Solo match via HTMX POST /hx/mt/new
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/hx/mt/new",
				strings.NewReader(url.Values{"mode": {"bot"}, "variant": {tc.variant}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("POST /hx/mt/new: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("POST /hx/mt/new status=%d", resp.StatusCode)
			}
			body := mustReadBody(t, resp)
			assertStrictZeroInlineHTML(t, "/hx/mt/new", body)

			// Verify visual domino & polish contracts on initial board
			if !strings.Contains(body, "Your Train") {
				t.Errorf("expected 'Your Train' label in rendered board")
			}
			if strings.Contains(body, "You Train") {
				t.Errorf("found legacy typo 'You Train' in rendered board")
			}
			if strings.Contains(body, "No match") {
				t.Errorf("found legacy 'No match' spam label in rendered hand")
			}
			if !strings.Contains(body, "bg-domino-face") || !strings.Contains(body, "rounded-tile") {
				t.Errorf("missing visual domino tile markup")
			}
			if !strings.Contains(body, "Double-"+tc.variant) {
				t.Errorf("expected Solo boneyard sub-label Double-%s", tc.variant)
			}

			// Extract room code from any hx-vals action (Solo hides room code from scoreboard text)
			actions := parseMTActions(t, body)
			if len(actions) == 0 {
				t.Fatalf("no HTMX action buttons rendered on opening turn")
			}
			roomCode := actions[0].Room
			if strings.Contains(body, "&middot; "+roomCode) {
				t.Errorf("Solo mode should hide unused room code %s under Boneyard", roomCode)
			}

			checkedDrawReject := false
			checkedHumanPlayLog := false
			handCompleted := false

			// Play an entire real hand to completion via HTMX POST /hx/mt/action
			for step := 0; step < 250; step++ {
				if strings.Contains(body, "Deal Next Hand") {
					handCompleted = true
					break
				}
				actions = parseMTActions(t, body)
				var playAct, drawAct, passAct *parsedMTAction
				for i := range actions {
					switch actions[i].Action {
					case "PLAY":
						if playAct == nil {
							playAct = &actions[i]
						}
					case "DRAW":
						drawAct = &actions[i]
					case "PASS":
						passAct = &actions[i]
					}
				}

				// Crucial Rule Check (MT-INV-7): whenever a PLAY exists, Draw & Pass buttons
				// MUST NOT appear AND forged DRAW/PASS POSTs must be rejected with ERR_LEGAL_MOVE_AVAILABLE.
				if playAct != nil {
					if drawAct != nil || passAct != nil {
						t.Fatalf("step %d: Draw/Pass button rendered while legal PLAY exists", step)
					}
					if !checkedDrawReject {
						for _, illegalAct := range []string{"DRAW", "PASS"} {
							forgedResp, err := client.PostForm(srv.URL+"/hx/mt/action", url.Values{
								"room":   {roomCode},
								"action": {illegalAct},
							})
							if err != nil {
								t.Fatalf("forged %s request: %v", illegalAct, err)
							}
							forgedBody := mustReadBody(t, forgedResp)
							if !strings.Contains(forgedBody, "ERR_LEGAL_MOVE_AVAILABLE") {
								t.Fatalf("forged %s while holding legal moves was NOT rejected with ERR_LEGAL_MOVE_AVAILABLE", illegalAct)
							}
						}
						checkedDrawReject = true
					}
				}

				var chosen *parsedMTAction
				if playAct != nil {
					chosen = playAct
				} else if drawAct != nil {
					chosen = drawAct
				} else if passAct != nil {
					chosen = passAct
				} else {
					t.Fatalf("step %d: no PLAY/DRAW/PASS action available and hand not over", step)
				}

				form := url.Values{
					"room":        {chosen.Room},
					"action":      {chosen.Action},
					"tileIndex":   {chosen.TileIndex},
					"targetTrain": {chosen.TargetTrain},
				}
				resp, err := client.PostForm(srv.URL+"/hx/mt/action", form)
				if err != nil {
					t.Fatalf("step %d POST /hx/mt/action: %v", step, err)
				}
				body = mustReadBody(t, resp)
				assertStrictZeroInlineHTML(t, "/hx/mt/action", body)

				if chosen.Action == "PLAY" && !checkedHumanPlayLog {
					if !strings.Contains(body, "You played [") || !strings.Contains(body, " on ") {
						t.Errorf("expected descriptive human Table Log ('You played [...] on ...'), got:\n%s", body)
					}
					checkedHumanPlayLog = true
				}
			}

			if !handCompleted {
				t.Fatalf("Double-%s hand did not complete within 250 user steps", tc.variant)
			}
			if !checkedDrawReject {
				t.Errorf("never exercised legal-move DRAW/PASS rejection during hand")
			}
			if !strings.Contains(body, tc.wantNextLabel) {
				t.Errorf("expected next-hand button %q, got body missing it", tc.wantNextLabel)
			}
			if strings.Contains(body, "&rarr; Next") {
				t.Errorf("found legacy literal '-> Next' in Deal Next Hand button")
			}

			// Click Deal Next Hand and verify Hand 2 starts cleanly
			nextResp, err := client.PostForm(srv.URL+"/hx/mt/next-hand", url.Values{"room": {roomCode}})
			if err != nil {
				t.Fatalf("POST /hx/mt/next-hand: %v", err)
			}
			nextBody := mustReadBody(t, nextResp)
			if !strings.Contains(nextBody, "2/") {
				t.Errorf("expected Hand 2/N indicator after Deal Next Hand")
			}
		})
	}
}

var dominoAriaRe = regexp.MustCompile(`aria-label="Domino (\d+-\d+)"`)

// TestE2EMexicanTrainTwoPlayerInviteAndSeatIsolation simulates two separate players
// joining via /play/MEX-XXXX (plus a same-tab revisit and a 3rd-party spectator),
// verifying Invite Link UX, seat cookie isolation, turn enforcement, and live polling.
func TestE2EMexicanTrainTwoPlayerInviteAndSeatIsolation(t *testing.T) {
	srv := startTestArcade(t, false)
	clientP1 := newBrowserClient(t)
	clientP2 := newBrowserClient(t)
	clientSpec := newBrowserClient(t)

	// 1. Player 1 creates a 2-player Double-9 room (follows 303 redirect to /play/MEX-XXXX)
	resp1, err := clientP1.PostForm(srv.URL+"/hx/mt/new", url.Values{"mode": {"2p"}, "variant": {"9"}})
	if err != nil {
		t.Fatalf("P1 create 2p room: %v", err)
	}
	inviteURL := resp1.Request.URL.String()
	if !strings.Contains(inviteURL, "/play/MEX-") {
		t.Fatalf("expected redirect to /play/MEX-XXXX, got %s", inviteURL)
	}
	roomCode := strings.TrimPrefix(resp1.Request.URL.Path, "/play/")
	p1Body := mustReadBody(t, resp1)
	assertStrictZeroInlineHTML(t, "/play/"+roomCode, p1Body)

	// Verify upgraded 2-player invite strip: full ShareURL, Copy Link button, Incognito tip
	if !strings.Contains(p1Body, inviteURL) {
		t.Errorf("invite strip missing full ShareURL %q", inviteURL)
	}
	if !strings.Contains(p1Body, `data-copy-text="`+inviteURL+`"`) {
		t.Errorf("invite strip missing strict-CSP Copy Link button")
	}
	if !strings.Contains(p1Body, "Incognito window") {
		t.Errorf("invite strip missing local testing Incognito tip")
	}
	if !strings.Contains(p1Body, "Waiting for Player 2") {
		t.Errorf("expected Waiting for Player 2 before seat 2 joins")
	}

	// 2. Reopening invite URL in P1's same browser keeps P1 as Seat 1 (does not claim Seat 2)
	revisitResp, err := clientP1.Get(inviteURL)
	if err != nil {
		t.Fatalf("P1 revisit: %v", err)
	}
	revisitBody := mustReadBody(t, revisitResp)
	if !strings.Contains(revisitBody, "Waiting for Player 2") {
		t.Errorf("same-browser P1 revisit should NOT claim Player 2 seat")
	}

	// 3. Player 2 opens invite URL in a fresh cookie jar (Incognito / second device)
	resp2, err := clientP2.Get(inviteURL)
	if err != nil {
		t.Fatalf("P2 join: %v", err)
	}
	p2Body := mustReadBody(t, resp2)
	if !strings.Contains(p2Body, "Joined room "+roomCode+" as Player 2!") {
		t.Errorf("P2 did not join seat 2 cleanly")
	}

	// 4. Third browser (spectator) cannot play moves
	specResp, err := clientSpec.PostForm(srv.URL+"/hx/mt/action", url.Values{"room": {roomCode}, "action": {"DRAW"}})
	if err != nil {
		t.Fatalf("spectator action: %v", err)
	}
	specBody := mustReadBody(t, specResp)
	if !strings.Contains(specBody, "Spectators cannot play moves") {
		t.Errorf("expected spectator rejection notice")
	}

	// 5. P2 cannot play while it is P1's turn
	p2EarlyResp, err := clientP2.PostForm(srv.URL+"/hx/mt/action", url.Values{"room": {roomCode}, "action": {"DRAW"}})
	if err != nil {
		t.Fatalf("P2 early action: %v", err)
	}
	if !strings.Contains(mustReadBody(t, p2EarlyResp), "ERR_OUT_OF_TURN") {
		t.Errorf("expected P2 out-of-turn action rejected with ERR_OUT_OF_TURN")
	}

	// 6. P1 polls /hx/mt/state, sees Your Turn, and plays a real legal move (or draws/passes if blocked)
	p1PollResp, err := clientP1.Get(srv.URL + "/hx/mt/state?room=" + roomCode)
	if err != nil {
		t.Fatalf("P1 poll: %v", err)
	}
	p1TurnBody := mustReadBody(t, p1PollResp)
	if !strings.Contains(p1TurnBody, "Your Turn") {
		t.Fatalf("expected P1 to have Your Turn after P2 joined")
	}

	// Verify strict hidden-hand isolation: P1's hand tiles and P2's hand tiles are disjoint
	p1HandSection := p1TurnBody[strings.Index(p1TurnBody, "Your Hand"):]
	p2HandSection := p2Body[strings.Index(p2Body, "Your Hand"):]
	p1Tiles := make(map[string]bool)
	for _, m := range dominoAriaRe.FindAllStringSubmatch(p1HandSection, -1) {
		p1Tiles[m[1]] = true
	}
	if len(p1Tiles) == 0 {
		t.Fatalf("failed to extract P1 hand dominoes from rendered HTML")
	}
	for _, m := range dominoAriaRe.FindAllStringSubmatch(p2HandSection, -1) {
		if p1Tiles[m[1]] {
			t.Fatalf("hidden-hand isolation breach: tile %s appeared in both P1 and P2 hands", m[1])
		}
	}

	// Execute P1's turn until turn advances to P2
	for guard := 0; guard < 10 && strings.Contains(p1TurnBody, "Your Turn"); guard++ {
		acts := parseMTActions(t, p1TurnBody)
		if len(acts) == 0 {
			t.Fatalf("P1 has Your Turn but zero HTMX actions")
		}
		a := acts[0]
		r, err := clientP1.PostForm(srv.URL+"/hx/mt/action", url.Values{
			"room": {a.Room}, "action": {a.Action}, "tileIndex": {a.TileIndex}, "targetTrain": {a.TargetTrain},
		})
		if err != nil {
			t.Fatalf("P1 move: %v", err)
		}
		p1TurnBody = mustReadBody(t, r)
	}

	// 7. P2 polls /hx/mt/state and now holds Your Turn!
	p2PollResp, err := clientP2.Get(srv.URL + "/hx/mt/state?room=" + roomCode)
	if err != nil {
		t.Fatalf("P2 poll: %v", err)
	}
	p2TurnBody := mustReadBody(t, p2PollResp)
	if !strings.Contains(p2TurnBody, "Your Turn") {
		t.Fatalf("expected P2 to receive Your Turn after P1 finished turn")
	}

	// 8. Concurrent GET /play/ + GET /hx/mt/state + POST /hx/mt/action (verified under go test -race)
	done := make(chan struct{}, 2)
	go func() {
		defer func() { done <- struct{}{} }()
		for i := 0; i < 10; i++ {
			if r, err := clientP1.Get(inviteURL); err == nil {
				_ = mustReadBody(t, r)
			}
		}
	}()
	go func() {
		defer func() { done <- struct{}{} }()
		for i := 0; i < 10; i++ {
			if r, err := clientP2.Get(srv.URL + "/hx/mt/state?room=" + roomCode); err == nil {
				_ = mustReadBody(t, r)
			}
		}
	}()
	<-done
	<-done
}

func TestBuildShareURLRejectsPoisonedHost(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"127.0.0.1:8080", "http://127.0.0.1:8080/play/MEX-ABCD"},
		{"newvisiongaming.web.app", "http://newvisiongaming.web.app/play/MEX-ABCD"},
		{"[::1]:8080", "http://[::1]:8080/play/MEX-ABCD"},
		{"legit.example@evil.example", "/play/MEX-ABCD"},
		{"evil.example/path", "/play/MEX-ABCD"},
		{"bad!host$name", "/play/MEX-ABCD"},
		{"", "/play/MEX-ABCD"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/play/MEX-ABCD", nil)
		req.Host = tc.host
		if got := buildShareURL(req, "MEX-ABCD"); got != tc.want {
			t.Errorf("buildShareURL(Host=%q) = %q, want %q", tc.host, got, tc.want)
		}
	}
}

