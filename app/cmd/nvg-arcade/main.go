// Command nvg-arcade is the New Vision Gaming Classic Games Suite server.
//
// Stack:
//   - Go 1.24 (native FIPS 140-3 SC-13, zero third-party Go modules)
//   - html/template (.tmpl) + embed.FS + W3C DTCG Tailwind v4 + Vendored HTMX 2.0.4
//   - Firebase Hosting + Cloud Run ($PORT aware) + AdSense/GPT Zero-CLS Nonce CSP
//   - A2A v1.0 (/.well-known/agent-card.json + POST /a2a) & A2UI v1.0 (/api/a2ui/surface)
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/fips140"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"new-vision-gaming/app/pkg/tictactoe"
	"new-vision-gaming/app/web"
)

const (
	signatureDomain = "aegis-verifiable-software-factory-v1:"
	maxRequestBytes = 64 * 1024
)

type keyRecord struct {
	Role      string `json:"role"`
	KeyID     string `json:"keyId"`
	PublicKey string `json:"publicKey"`
}

type signedEnvelope struct {
	PayloadType string `json:"payloadType"`
	Role        string `json:"role"`
	KeyID       string `json:"keyId"`
	PublicKey   string `json:"publicKey"`
	Payload     []byte `json:"payload"`
	Signature   string `json:"signature"`
}

type bundleView struct {
	BinarySHA256 string `json:"binarySha256"`
}

type auditEvent struct {
	TimestampUTC string `json:"timestampUtc"`
	EventType    string `json:"eventType"`
	Protocol     string `json:"protocol,omitempty"`
	Actor        string `json:"actor,omitempty"`
	Move         string `json:"move,omitempty"`
	Valid        *bool  `json:"valid,omitempty"`
	Reason       string `json:"reason,omitempty"`
	FIPS140      bool   `json:"fips140Enabled"`
	BinaryDigest string `json:"binaryDigest"`
	RemoteAddr   string `json:"remoteAddr,omitempty"`
}

type cellVM struct {
	Index int
	Mark  string
}

type pageViewModel struct {
	PageTitle      string
	View           string // "arcade", "tictactoe", "privacy"
	Board          string
	Turn           string
	Status         string
	Winner         string
	GameOver       bool
	Notice         string
	Cells          []cellVM
	FIPS140Enabled bool
	SelfVerified   bool
	ShortDigest    string
	AdsEnabled     bool
	WebAdsEnabled  bool
	ClientShell    string // "web", "pwa", "native-ios", "native-android"
	IsNativeShell  bool
	AdSenseClient  string
	CSPNonce       string
	MT             *mtBoardVM
}

func detectClientShell(r *http.Request) (shell string, isNative bool) {
	raw := strings.ToLower(strings.TrimSpace(r.Header.Get("X-NVG-Client")))
	switch raw {
	case "native-ios", "capacitor-ios":
		return "native-ios", true
	case "native-android", "capacitor-android":
		return "native-android", true
	case "pwa":
		return "pwa", false
	default:
		return "web", false
	}
}

func emitAuditLog(ev auditEvent) {
	ev.TimestampUTC = time.Now().UTC().Format(time.RFC3339Nano)
	raw, _ := json.Marshal(ev)
	fmt.Fprintln(os.Stderr, string(raw))
}

type nonceCtxKey struct{}

func main() {
	defaultAddr := "127.0.0.1:8080"
	if p := os.Getenv("PORT"); p != "" {
		defaultAddr = "0.0.0.0:" + p
	}
	evalCLI := flag.String("eval-cli", "", "Evaluate one Tic-Tac-Toe MoveRequest JSON (used by Clean-Room verifier)")
	addr := flag.String("addr", defaultAddr, "HTTP listen address (defaults to 0.0.0.0:$PORT on Cloud Run)")
	bundlePath := flag.String("bundle", "", "Optional release_authority signed bundle for SI-7 startup self-attestation")
	trustRootsPath := flag.String("trust-roots", "", "Optional public trust-roots.json for SI-7 startup self-attestation")
	requireFIPS := flag.Bool("require-fips", false, "Fail closed unless Go FIPS 140-3 mode (GODEBUG=fips140=on) is active")
	flag.Parse()

	// Clean-Room deterministic CLI evaluation mode
	if *evalCLI != "" {
		var req tictactoe.MoveRequest
		dec := json.NewDecoder(strings.NewReader(*evalCLI))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			fmt.Fprintf(os.Stderr, "invalid move json: %v\n", err)
			os.Exit(2)
		}
		res := tictactoe.ApplyMove(req)
		v := "INVALID"
		if res.Valid {
			v = "VALID"
		}
		w := res.Winner
		if w == "" {
			w = "-"
		}
		nt := res.NextTurn
		if nt == "" {
			nt = "-"
		}
		fmt.Printf("%s %s %s %s %s %s\n", v, res.Status, w, nt, res.Reason, res.Board)
		return
	}

	fipsActive := fips140.Enabled()
	if *requireFIPS && !fipsActive {
		log.Fatalf("FATAL (SC-13): FIPS 140-3 required (--require-fips) but inactive. Start with GODEBUG=fips140=on")
	}

	adsEnabled := strings.EqualFold(os.Getenv("ADS_ENABLED"), "true")
	adsClient := os.Getenv("ADSENSE_CLIENT_ID")
	if adsClient == "" {
		adsClient = "ca-pub-0000000000000000"
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("locate self executable: %v", err)
	}
	selfSHA, err := sha256File(exePath)
	if err != nil {
		log.Fatalf("hash self executable: %v", err)
	}

	selfVerified := false
	if *bundlePath != "" || *trustRootsPath != "" {
		if err := verifySelfBundle(selfSHA, *bundlePath, *trustRootsPath); err != nil {
			log.Fatalf("FATAL (SI-7): Runtime self-attestation FAILED: %v", err)
		}
		selfVerified = true
	}

	shortDigest := selfSHA
	if len(shortDigest) > 16 {
		shortDigest = shortDigest[:16]
	}

	tmpl, err := template.ParseFS(web.Assets, "templates/*.tmpl")
	if err != nil {
		log.Fatalf("parse embedded html/templates: %v", err)
	}
	staticFS, err := fs.Sub(web.Assets, "static")
	if err != nil {
		log.Fatalf("mount embedded static assets: %v", err)
	}

	emitAuditLog(auditEvent{
		EventType:    "APP_STARTUP",
		FIPS140:      fipsActive,
		BinaryDigest: shortDigest,
		Reason:       fmt.Sprintf("selfVerified=%v adsEnabled=%v", selfVerified, adsEnabled),
	})

	newVM := func(r *http.Request, title, view, board, turn, status, winner, notice string) pageViewModel {
		cells := make([]cellVM, 9)
		for i := 0; i < 9; i++ {
			mark := ""
			if i < len(board) && (board[i] == 'X' || board[i] == 'O') {
				mark = string(board[i])
			}
			cells[i] = cellVM{Index: i, Mark: mark}
		}
		nonce, _ := r.Context().Value(nonceCtxKey{}).(string)
		shell, isNative := detectClientShell(r)
		return pageViewModel{
			PageTitle:      title,
			View:           view,
			Board:          board,
			Turn:           turn,
			Status:         status,
			Winner:         winner,
			GameOver:       status == "WIN" || status == "DRAW",
			Notice:         notice,
			Cells:          cells,
			FIPS140Enabled: fipsActive,
			SelfVerified:   selfVerified,
			ShortDigest:    shortDigest,
			AdsEnabled:     adsEnabled,
			WebAdsEnabled:  adsEnabled && !isNative,
			ClientShell:    shell,
			IsNativeShell:  isNative,
			AdSenseClient:  adsClient,
			CSPNonce:       nonce,
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Classic Games Suite Lobby
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "layout.tmpl", newVM(r, "Classic Games Arcade", "arcade", ".........", "X", "IN_PROGRESS", "", ""))
	})

	// Game #1: Tic-Tac-Toe Page
	mux.HandleFunc("/games/tictactoe", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "layout.tmpl", newVM(r, "Tic-Tac-Toe", "tictactoe", ".........", "X", "IN_PROGRESS", "", ""))
	})

	// AdSense / AdMob / GDPR / CCPA / App Store Privacy Policy
	mux.HandleFunc("/privacy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "layout.tmpl", newVM(r, "Privacy Policy & Ads", "privacy", ".........", "X", "IN_PROGRESS", "", ""))
	})

	// AdSense / AdMob / PWA / Deep-Link / SEO standard root documents (also served by Firebase Hosting)
	mux.HandleFunc("/ads.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		pubID := strings.TrimPrefix(adsClient, "ca-")
		fmt.Fprintf(w, "google.com, %s, DIRECT, f08c47fec0942fa0\n", pubID)
	})
	mux.HandleFunc("/app-ads.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		pubID := strings.TrimPrefix(adsClient, "ca-")
		fmt.Fprintf(w, "google.com, %s, DIRECT, f08c47fec0942fa0\n", pubID)
	})
	mux.HandleFunc("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":             "New Vision Gaming — Classic Arcade",
			"short_name":       "NVG Arcade",
			"description":      "Instant-load, zero-bloat classic strategy games including Tic-Tac-Toe and Mexican Train Dominoes.",
			"start_url":        "/?source=pwa",
			"scope":            "/",
			"display":          "standalone",
			"orientation":      "any",
			"background_color": "#090f1c",
			"theme_color":      "#090f1c",
			"categories":       []string{"games", "board", "strategy"},
			"icons": []map[string]any{
				{"src": "/static/icon.svg", "sizes": "any", "type": "image/svg+xml", "purpose": "any maskable"},
			},
		})
	})
	mux.HandleFunc("/.well-known/apple-app-site-association", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"applinks": map[string]any{
				"details": []map[string]any{
					{
						"appIDs": []string{"TEAMID.com.newvisiongaming.arcade"},
						"components": []map[string]string{
							{"/": "/play/*"},
							{"/": "/games/*"},
						},
					},
				},
			},
		})
	})
	mux.HandleFunc("/.well-known/assetlinks.json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []map[string]any{
			{
				"relation": []string{"delegate_permission/common.handle_all_urls"},
				"target": map[string]any{
					"namespace":                "android_app",
					"package_name":             "com.newvisiongaming.arcade",
					"sha256_cert_fingerprints": []string{"REPLACE_WITH_GOOGLE_PLAY_APP_SIGNING_SHA256_FINGERPRINT"},
				},
			},
		})
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "User-agent: *\nAllow: /\nSitemap: https://newvisiongaming.web.app/sitemap.xml")
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://newvisiongaming.web.app/</loc></url><url><loc>https://newvisiongaming.web.app/games/mexican-train</loc></url><url><loc>https://newvisiongaming.web.app/games/tictactoe</loc></url><url><loc>https://newvisiongaming.web.app/privacy</loc></url></urlset>`)
	})

	// Flagship Game #1: Mexican Train Dominoes (Solo vs Bot, 2-Player /play/MEX-XXXX rooms, REST API v1)
	registerMexicanTrainRoutes(mux, tmpl, newVM, fipsActive, shortDigest)

	// HTMX fragment endpoints
	mux.HandleFunc("/hx/move", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form payload", http.StatusBadRequest)
			return
		}
		pos, err := strconv.Atoi(r.FormValue("position"))
		if err != nil {
			http.Error(w, "invalid position", http.StatusBadRequest)
			return
		}
		req := tictactoe.MoveRequest{
			Board:    r.FormValue("board"),
			Player:   r.FormValue("player"),
			Position: pos,
			Actor:    r.FormValue("actor"),
		}
		res := tictactoe.ApplyMove(req)
		vCopy := res.Valid
		emitAuditLog(auditEvent{
			EventType:    "GAME_MOVE",
			Protocol:     "HTMX",
			Actor:        req.Actor,
			Move:         fmt.Sprintf("%s@%d", req.Player, req.Position),
			Valid:        &vCopy,
			Reason:       res.Reason,
			FIPS140:      fipsActive,
			BinaryDigest: shortDigest,
			RemoteAddr:   r.RemoteAddr,
		})

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var vm pageViewModel
		if !res.Valid {
			turn := res.NextTurn
			if turn == "" {
				turn = req.Player
			}
			vm = newVM(r, "Tic-Tac-Toe", "tictactoe", res.Board, turn, "IN_PROGRESS", res.Winner, "Rejected: "+res.Reason)
			if res.Winner != "" {
				vm.Status = "WIN"
				vm.GameOver = true
			}
		} else {
			vm = newVM(r, "Tic-Tac-Toe", "tictactoe", res.Board, res.NextTurn, res.Status, res.Winner, "")
		}
		_ = tmpl.ExecuteTemplate(w, "board.tmpl", vm)
	})

	mux.HandleFunc("/hx/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "board.tmpl", newVM(r, "Tic-Tac-Toe", "tictactoe", ".........", "X", "IN_PROGRESS", "", ""))
	})

	// Health probe
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"brand":          "New Vision Gaming",
			"app":            "nvg-arcade",
			"liveGames":      []string{"tictactoe", "mexican-train"},
			"uiStack":        "go-html-template-tailwind-htmx",
			"protocols":      []string{"REST", "HTMX", "A2A/1.0", "A2UI/1.0"},
			"adsEnabled":     adsEnabled,
			"selfVerified":   selfVerified,
			"fips140Enabled": fipsActive,
			"binarySha256":   selfSHA,
		})
	})

	// JSON REST API: POST /api/move (and stable versioned /api/v1/move for Phase 2 mobile clients)
	moveAPIHandler := func(w http.ResponseWriter, r *http.Request) {
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
		var req tictactoe.MoveRequest
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or non-conforming move JSON"})
			return
		}
		if dec.More() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unexpected trailing JSON data"})
			return
		}
		res := tictactoe.ApplyMove(req)
		vCopy := res.Valid
		emitAuditLog(auditEvent{
			EventType:    "GAME_MOVE",
			Protocol:     "REST",
			Actor:        req.Actor,
			Move:         fmt.Sprintf("%s@%d", req.Player, req.Position),
			Valid:        &vCopy,
			Reason:       res.Reason,
			FIPS140:      fipsActive,
			BinaryDigest: shortDigest,
			RemoteAddr:   r.RemoteAddr,
		})
		writeJSON(w, http.StatusOK, res)
	}
	mux.HandleFunc("/api/move", moveAPIHandler)
	mux.HandleFunc("/api/v1/move", moveAPIHandler)

	// A2A v1.0 Agent Card & JSON-RPC
	agentCardHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		writeJSON(w, http.StatusOK, buildA2AAgentCard("http://"+r.Host, selfSHA, fipsActive, selfVerified))
	}
	mux.HandleFunc("/.well-known/agent-card.json", agentCardHandler)
	mux.HandleFunc("/.well-known/agent.json", agentCardHandler)

	mux.HandleFunc("/a2a", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
		var rpcReq struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      any             `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&rpcReq); err != nil || rpcReq.JSONRPC != "2.0" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"error":   map[string]any{"code": -32600, "message": "Invalid A2A JSON-RPC 2.0 request"},
			})
			return
		}
		if rpcReq.Method != "message/send" && rpcReq.Method != "tasks/send" {
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      rpcReq.ID,
				"error":   map[string]any{"code": -32601, "message": "Method not found (supported: message/send, tasks/send)"},
			})
			return
		}
		var params struct {
			TaskID  string                 `json:"id"`
			Move    *tictactoe.MoveRequest `json:"move,omitempty"`
			Message *struct {
				Role  string `json:"role"`
				Parts []struct {
					Type string                 `json:"type"`
					Data *tictactoe.MoveRequest `json:"data,omitempty"`
				} `json:"parts"`
			} `json:"message,omitempty"`
		}
		_ = json.Unmarshal(rpcReq.Params, &params)
		moveReq := tictactoe.MoveRequest{Board: ".........", Player: "X", Position: 4, Actor: "a2a-peer-agent"}
		if params.Move != nil {
			moveReq = *params.Move
		} else if params.Message != nil {
			for _, p := range params.Message.Parts {
				if p.Data != nil {
					moveReq = *p.Data
					break
				}
			}
		}
		res := tictactoe.ApplyMove(moveReq)
		vCopy := res.Valid
		emitAuditLog(auditEvent{
			EventType:    "GAME_MOVE",
			Protocol:     "A2A/1.0",
			Actor:        moveReq.Actor,
			Move:         fmt.Sprintf("%s@%d", moveReq.Player, moveReq.Position),
			Valid:        &vCopy,
			Reason:       res.Reason,
			FIPS140:      fipsActive,
			BinaryDigest: shortDigest,
			RemoteAddr:   r.RemoteAddr,
		})
		taskID := params.TaskID
		if taskID == "" {
			taskID = fmt.Sprintf("task-%d", time.Now().UTC().UnixNano())
		}
		surface := buildA2UISurface(res.Board, res.NextTurn, res.Status, res.Winner, res.Reason)
		writeJSON(w, http.StatusOK, map[string]any{
			"jsonrpc": "2.0",
			"id":      rpcReq.ID,
			"result": map[string]any{
				"id": taskID,
				"status": map[string]any{
					"state":     "completed",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
				"artifacts": []map[string]any{
					{"artifactId": "game-decision", "name": "tictactoe-move-result", "mimeType": "application/json", "parts": []map[string]any{{"type": "data", "data": res}}},
					{"artifactId": "a2ui-surface", "name": "a2ui-v1-surface", "mimeType": "application/vnd.a2ui+json", "parts": []map[string]any{{"type": "data", "data": surface}}},
				},
			},
		})
	})

	// A2UI v1.0 Surface endpoint
	mux.HandleFunc("/api/a2ui/surface", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		board := r.URL.Query().Get("board")
		if len(board) != 9 {
			board = "........."
		}
		turn := r.URL.Query().Get("turn")
		if turn != "X" && turn != "O" {
			turn = "X"
		}
		writeJSON(w, http.StatusOK, buildA2UISurface(board, turn, "IN_PROGRESS", "", "READY"))
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           securityHeadersMiddleware(mux, adsEnabled),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	log.Printf("New Vision Gaming Arcade listening on http://%s (fips140=%v selfVerified=%v adsEnabled=%v sha256=%s)",
		*addr, fipsActive, selfVerified, adsEnabled, shortDigest)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func buildA2AAgentCard(baseURL, binarySHA string, fipsActive, selfVerified bool) map[string]any {
	return map[string]any{
		"name":            "New Vision Gaming — Classic Arcade Agent",
		"description":     "Firebase-hosted, spec-driven, clean-room verified classic games suite (Tic-Tac-Toe & Mexican Train Dominoes live) exposing A2A v1.0 tasks and A2UI v1.0 declarative surfaces.",
		"version":         "1.1.0",
		"protocolVersion": "1.0",
		"url":             baseURL + "/a2a",
		"supportedInterfaces": []map[string]string{
			{"protocolBinding": "JSONRPC", "url": baseURL + "/a2a"},
			{"protocolBinding": "HTTP+JSON", "url": baseURL + "/api/v1/move"},
			{"protocolBinding": "HTTP+JSON", "url": baseURL + "/api/v1/mexican-train/move"},
		},
		"capabilities": map[string]any{
			"streaming":              false,
			"pushNotifications":      false,
			"stateTransitionHistory": false,
			"extensions": []map[string]string{
				{"uri": "https://a2ui.org/specification/v1.0-a2ui/", "description": "A2UI v1.0 declarative UI surfaces bound to spec/design-tokens.json"},
			},
		},
		"defaultInputModes":  []string{"application/json"},
		"defaultOutputModes": []string{"application/json", "application/vnd.a2ui+json"},
		"skills": []map[string]any{
			{"id": "play_move", "name": "Apply Verified Tic-Tac-Toe Move", "description": "Validates and applies a Tic-Tac-Toe move against the formally verified Go/GCC/Clang-proven game kernel.", "tags": []string{"new-vision-gaming", "tictactoe", "a2ui"}},
			{"id": "play_mexican_train", "name": "Evaluate Verified Mexican Train Dominoes Move", "description": "Validates Double-9/Double-12 Mexican Train Dominoes rules (open doubles, public train markers, anti-backdoor invariants) against the 3-way Go/GCC/Clang-verified kernel.", "tags": []string{"new-vision-gaming", "mexican-train", "dominoes", "2-player"}},
			{"id": "render_a2ui_surface", "name": "Emit A2UI v1.0 Declarative Surface", "description": "Returns safe declarative A2UI v1.0 component JSON bound to spec/design-tokens.json.", "tags": []string{"a2ui", "design-tokens"}},
		},
		"assuranceMetadata": map[string]any{
			"binarySha256":   binarySHA,
			"fips140Enabled": fipsActive,
			"selfVerified":   selfVerified,
			"baselines":      []string{"FedRAMP-Moderate", "FedRAMP-20x", "DoD-IL4", "NIST-SSDF", "SLSA-L4"},
		},
	}
}

func buildA2UISurface(board, turn, status, winner, reason string) map[string]any {
	cellIDs := make([]string, 9)
	components := []map[string]any{
		{"id": "root", "component": "Card", "tokens": map[string]string{"bg": "color.surface.card", "border": "color.border.subtle", "radius": "radius.card"}, "children": []string{"header", "status-banner", "board-grid", "reset-btn"}},
		{"id": "header", "component": "Heading", "text": "New Vision Gaming — Tic-Tac-Toe", "tokens": map[string]string{"color": "color.ink.primary"}},
		{"id": "status-banner", "component": "Badge", "bind": "/game/status", "tokens": map[string]string{"color": "color.status.verified"}},
		{"id": "board-grid", "component": "Grid", "columns": 3, "children": cellIDs},
		{"id": "reset-btn", "component": "Button", "label": "New Match", "action": map[string]any{"name": "reset_match", "endpoint": "/hx/reset"}, "tokens": map[string]string{"bg": "color.accent.primary", "color": "color.ink.inverse"}},
	}
	cellsData := make([]map[string]any, 9)
	for i := 0; i < 9; i++ {
		cid := fmt.Sprintf("cell-%d", i)
		cellIDs[i] = cid
		mark := ""
		if i < len(board) && (board[i] == 'X' || board[i] == 'O') {
			mark = string(board[i])
		}
		tok := "color.ink.muted"
		if mark == "X" {
			tok = "color.accent.playerX"
		} else if mark == "O" {
			tok = "color.accent.playerO"
		}
		cellsData[i] = map[string]any{"index": i, "mark": mark}
		components = append(components, map[string]any{
			"id": cid, "component": "Button", "bind": fmt.Sprintf("/game/cells/%d/mark", i),
			"disabled": mark != "" || status == "WIN" || status == "DRAW",
			"tokens":   map[string]string{"bg": "color.surface.cell", "color": tok, "radius": "radius.cell"},
			"action":   map[string]any{"name": "play_move", "endpoint": "/a2a", "payload": map[string]any{"board": board, "player": turn, "position": i}},
		})
	}
	return map[string]any{
		"a2uiVersion":  "1.0",
		"designTokens": "spec/design-tokens.json",
		"surfaceId":    "nvg-tictactoe-main",
		"messages": []map[string]any{
			{"createSurface": map[string]any{"surfaceId": "nvg-tictactoe-main", "catalogId": "https://a2ui.org/catalogs/standard/v1.0"}},
			{"updateComponents": map[string]any{"surfaceId": "nvg-tictactoe-main", "components": components}},
			{"updateDataModel": map[string]any{"surfaceId": "nvg-tictactoe-main", "path": "/game", "value": map[string]any{"board": board, "turn": turn, "status": status, "winner": winner, "reason": reason, "cells": cellsData}}},
		},
	}
}

func securityHeadersMiddleware(next http.Handler, adsEnabled bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")
		h.Set("Cache-Control", "no-store")
		h.Set("Pragma", "no-cache")

		_, isNative := detectClientShell(r)
		if adsEnabled && !isNative {
			var nonceBuf [16]byte
			_, _ = rand.Read(nonceBuf[:])
			nonce := base64.StdEncoding.EncodeToString(nonceBuf[:])
			// Google AdSense / GPT Strict Nonce-based CSP (zero unsafe-inline!)
			h.Set("Content-Security-Policy", fmt.Sprintf(
				"default-src 'self'; script-src 'self' 'nonce-%s' 'strict-dynamic' https://pagead2.googlesyndication.com; style-src 'self'; img-src 'self' https://*.googlesyndication.com https://*.doubleclick.net data:; frame-src https://*.googlesyndication.com https://*.doubleclick.net; connect-src 'self' https://pagead2.googlesyndication.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
				nonce,
			))
			r = r.WithContext(context.WithValue(r.Context(), nonceCtxKey{}, nonce))
		} else {
			h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		}
		next.ServeHTTP(w, r)
	})
}

func verifySelfBundle(selfSHA, bundlePath, trustRootsPath string) error {
	rootsRaw, err := os.ReadFile(trustRootsPath)
	if err != nil {
		return fmt.Errorf("read trust-roots: %w", err)
	}
	var roots map[string]keyRecord
	if err := json.Unmarshal(rootsRaw, &roots); err != nil {
		return fmt.Errorf("parse trust-roots: %w", err)
	}
	raKey, ok := roots["release_authority"]
	if !ok {
		return fmt.Errorf("missing release_authority trust root")
	}
	bundleRaw, err := os.ReadFile(bundlePath)
	if err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}
	var env signedEnvelope
	if err := json.Unmarshal(bundleRaw, &env); err != nil {
		return fmt.Errorf("parse bundle: %w", err)
	}
	if env.Role != "release_authority" || env.PublicKey != raKey.PublicKey {
		return fmt.Errorf("bundle signed by untrusted key")
	}
	pubBytes, err := base64.StdEncoding.DecodeString(raKey.PublicKey)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key")
	}
	sigBytes, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature")
	}
	msg := append([]byte(signatureDomain+env.PayloadType+":"), env.Payload...)
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), msg, sigBytes) {
		return fmt.Errorf("Ed25519 signature verification failed")
	}
	var bv bundleView
	if err := json.Unmarshal(env.Payload, &bv); err != nil {
		return fmt.Errorf("parse bundle view: %w", err)
	}
	if bv.BinarySHA256 != selfSHA {
		return fmt.Errorf("binary tampered: running=%s != signed=%s", selfSHA, bv.BinarySHA256)
	}
	return nil
}

func sha256File(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
