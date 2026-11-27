# DESIGN.md — New Vision Gaming Tabletop Design System

> Human + AI Design Constitution for the **New Vision Gaming** Classic Games Suite.
> Machine-readable values live in [`spec/design-tokens.json`](./design-tokens.json) (W3C DTCG v2.0.0) and compile into `app/web/styles/input.css` → `app/web/static/tailwind.css`.

---

## 1. Visual Direction — Warm Light Tabletop Suite-Wide

New Vision Gaming uses a **clean, warm, high-legibility light tabletop aesthetic** across every page and game (Arcade Lobby, Mexican Train Dominoes, Tic-Tac-Toe, Privacy & Ads, and all future Phase 2/3 titles):

- **Canvas (`#f1f2f4`):** Soft cool-gray tabletop surface (`--color-surface-canvas`).
- **Cards & Panels (`#ffffff`):** Crisp white surfaces (`--color-surface-card`) with subtle borders (`#c2ccd6`) and soft tabletop elevation (`shadow-card` — never heavy dark drop shadows).
- **Recessed Cells & Train Tracks (`#e9eef3`):** Calm recessed lanes (`--color-surface-cell`) warming to soft light-gold (`#fdf4e3`, `--color-surface-cell-hover`) when highlighted/playable.
- **Primary Accent (`#e99225`):** Warm amber-gold (`--color-accent-primary`) for primary CTAs, active turn indicators, and playable focus rings.
- **Typography:** Zero-CDN system sans (`ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif`) for instant LCP + strict CSP; system monospace for scores, room codes, engines, and SHA-256 build digests.
- **Accessibility:** Primary ink (`#2e3138`) achieves `13.0:1` on white; muted ink (`#52677d`) achieves `5.3:1` on canvas and `5.9:1` on white; status tokens (`#15803d`, `#b45309`, `#dc2626`) meet WCAG AA on light cards. For small text placed directly on `#e99225`, prefer high-contrast dark ink (`#1f1607`) or bold inverse ink (`#fefcf5`) at large/semibold sizes.

---

## 2. Pure-CSS Domino Tile Anatomy (Strict CSP: Zero Inline `style=`)

Every domino tile is rendered server-side via the reusable Go `html/template` partial `app/web/templates/domino_tile.tmpl` using **only static compiled CSS classes** (zero inline `style=` attributes, zero client-side canvas/React rendering, 100% compliant with `style-src 'self'`):

### 2.1 Pip Colors (`--color-domino-pip-0` .. `--color-domino-pip-12`)

| Value | Color Token | Hex | Rendering |
| :--- | :--- | :--- | :--- |
| `0` | `--color-domino-pip-0` | `#9ca3af` | Empty 3×3 grid (blank half) |
| `1` | `--color-domino-pip-1` | `#06b6d4` | 3×3 grid: `[4]` |
| `2` | `--color-domino-pip-2` | `#84cc16` | 3×3 grid: `[0, 8]` |
| `3` | `--color-domino-pip-3` | `#ef4444` | 3×3 grid: `[0, 4, 8]` |
| `4` | `--color-domino-pip-4` | `#22c55e` | 3×3 grid: `[0, 2, 6, 8]` |
| `5` | `--color-domino-pip-5` | `#3b82f6` | 3×3 grid: `[0, 2, 4, 6, 8]` |
| `6` | `--color-domino-pip-6` | `#eab308` | 3×3 grid: `[0, 2, 3, 5, 6, 8]` |
| `7` | `--color-domino-pip-7` | `#f472b6` | 3×3 grid: `[0, 2, 3, 4, 5, 6, 8]` |
| `8` | `--color-domino-pip-8` | `#14b8a6` | 3×3 grid: `[0, 1, 2, 3, 5, 6, 7, 8]` |
| `9` | `--color-domino-pip-9` | `#a855f7` | 3×3 grid: `[0, 1, 2, 3, 4, 5, 6, 7, 8]` |
| `10` | `--color-domino-pip-10` | `#f97316` | Centered bold numeral `10` |
| `11` | `--color-domino-pip-11` | `#000000` | Centered bold numeral `11` |
| `12` | `--color-domino-pip-12` | `#6b7280` | Centered bold numeral `12` |

### 2.2 Orientation & Framing
- **Surface:** `bg-domino-face` (`#ffffff`), `rounded-tile` (`6px`), `border border-black/20`, `shadow-tile`.
- **Played Non-Double Tiles on Trains:** Horizontal (`w-16 h-8 md:w-20 md:h-10 flex-row`) with vertical center bar (`w-px h-3/4 bg-domino-divider/60 self-center`).
- **Center Engine, Played Doubles on Trains, and Hand Tiles:** Vertical (`w-10 h-20 flex-col` in hand; `w-8 h-16 md:w-10 md:h-20 flex-col` on train lanes) with horizontal center bar (`w-3/4 h-px bg-domino-divider/60 self-center`).
- **Dot Sizing:** `w-1.5 h-1.5 rounded-full` (`md:w-2 md:h-2` where board geometry permits); numerals `10–12` use `font-bold text-base md:text-lg leading-none`.

### 2.3 Hand Interaction States
- **Playable Tiles:** Full opacity, `ring-2 ring-accent-primary/80 hover:-translate-y-0.5 hover:shadow-lg transition-transform`, with compact 1-click destination train buttons directly beneath the vertical domino.
- **Unplayable Tiles:** Softly dimmed (`opacity-45`) with **no repeated `"No match"` caption** so players scan playable options instantaneously without vertical bloat.

---

## 3. Above-the-Fold `1280×800` Layout & Zero-CLS Ad Constraints

### 3.1 Immutable Zero-CLS SafeFrame Slots (`CLS <= 0.01`)
Never shrink or collapse reserved heights on web/PWA shells (all web ad slots remain omitted on `X-NVG-Client: native-ios` / `native-android`):
- `#nvg-top-leaderboard`: `min-h-[90px]` (`728×90` desktop / `320×50` mobile)
- `#nvg-sidebar-rect`: `300px` desktop right rail, `min-h-[250px]`
- `#nvg-post-match-banner`: `min-h-[100px]`, rendered only during natural between-hand breaks (`HandOver == true`)

### 3.2 Active Match Density (`1280×800`)
- When a Mexican Train match is active (`{{if .MT}}`), collapse the landing hero into a single compact title bar so scoreboard + 3 train lanes + full player hand stay visible together above the fold.
- Scoreboard uses a single tight 4-cell strip (`Engine`, `You`, `Opponent`, `Boneyard`); omit unused room codes in Solo mode.
- Train lanes use dashed tabletop tracks (`border-2 border-dashed border-border-subtle bg-surface-cell`), warming to `border-accent-primary bg-surface-cell-hover` when open/playable and `border-status-danger bg-status-danger/10` when `NeedsDouble == true`, prefixed by the visual vertical Engine tile `[E|E]`.
- Hand renders as a compact wrapping flex tray (`flex flex-wrap gap-2 items-start`) rather than a tall sparse grid so 11–15 vertical tiles sit cleanly in 1–2 rows directly under the trains.

---

## 4. Strict Security & Anti-Pattern Guardrails

- **Zero Inline CSS/JS:** Never emit `style="..."`, `onclick="..."`, or inline `<script>` blocks (enforced by `style-src 'self'`, `script-src 'self'`, and DAST-02). Interactive helpers (e.g., Copy Invite Link) live in small, audited static files under `app/web/static/` with optional CSP nonce attributes for `'strict-dynamic'` AdSense mode.
- **Zero External CDNs:** No Google Fonts, CDN CSS, or runtime icon scripts. Everything embeds via `//go:embed`.
- **A2UI v1.0 Alignment:** All semantic keys in `spec/design-tokens.json` (`color.surface.*`, `color.border.*`, `color.ink.*`, `color.accent.*`, `color.status.*`, `color.train.*`, `color.domino.*`) remain stable so future A2UI v1.0 declarative surfaces bind directly to the exact same visual contract.
