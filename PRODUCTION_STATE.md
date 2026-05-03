
## 📋 CHANGES 2026-03-27
### Backend
- feat: PUT /api/trader/accounts/:id — edit nickname & about per account
- feat: about field di GetUserAccounts, GetPublicTraders, GetTraderProfile
- fix: duplicate route PUT /api/trader/accounts/:id di main.go
- fix: duplicate UpdateAccountMeta, UpdateAccount methods
### Frontend
- feat: Edit modal di Accounts tab (nickname + description)
- feat: Strategy tampil di Trader Dashboard bawah subtitle
- feat: Strategy tampil di Marketplace TraderProfile
- feat: field About/Description di Add Account modal
- fix: semua placeholder English

## 🐳 CURRENT PRODUCTION (Updated 2026-03-27)
### Backend:
- **Image:** `crunchalpha-v3:production-202604160220
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## 📋 CHANGES 2026-03-27 (Performance Chart)
### Frontend
- feat: Weekly & Monthly chart tambah $ symbol dan cumulative ROI %
- feat: Monthly ROI = cumulative dalam selected year / deposit
- feat: Weekly ROI = cumulative dalam selected month / deposit
- feat: TraderProfile marketplace chart juga updated

## 🐳 CURRENT PRODUCTION (Updated 2026-03-27)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## 📋 CHANGES 2026-03-28
### Backend
- feat: GetTrades pagination dengan offset & total (handler_trades.go)
- feat: GetTraderTrades investor endpoint pagination dengan offset & total
- fix: column trade_type → type di GetTradesByAccountPaginated
- feat: GetTradesByAccountPaginated di repository
### Frontend
- feat: Trade History pagination di Trader Dashboard (Showing X-Y of Z, Prev/Next)
- feat: Trade History pagination di Marketplace TraderProfile
- feat: dropdown "X per page" ganti "Last X"
- feat: hapus Position Size Calculator dari Tools

## 🐳 CURRENT PRODUCTION (Updated 2026-03-28)
### Backend:
- **Image:** `crunchalpha-v3:production-202604160220
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## 📋 CHANGES 2026-03-27
### Backend
- feat: pagination trade history — tambah offset/limit/total di endpoint trader & investor
- feat: add nickname/about edit, strategy field tampil di dashboard & marketplace
- feat: weekly/monthly chart tambah currency symbol dan cumulative ROI %
- Backend: crunchalpha-v3:production-202604160220

### Frontend
- fix: AddAccountModal — onClose sebelum alert, support onAccountAdded prop
- fix: double notification bug setelah add account resolved
- Frontend: crunchalpha-frontend-v3:prod-202604150205

## 📋 CHANGES 2026-03-28
### Backend
- feat: pagination trade history di trader dashboard dan marketplace trader profile
- Backend: crunchalpha-v3:production-202604160220

### Frontend
- fix: AddAccountModal — close before alert, support onAccountAdded prop
- Frontend: crunchalpha-frontend-v3:prod-202604150205

## 🐳 CURRENT PRODUCTION (Updated 2026-03-28)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Port:** 8090 (internal), via nginx https
- **Git:** master branch

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Port:** 5176 (internal), via nginx https
- **URL:** https://crunchalpha.com

## ⚠️ PENDING
1. EA MT4 distribute ke publisher external
2. Overleveraging flag formula review
3. EA Keys management di Copy Settings frontend
4. Trigger copy engine dari EA trader push
5. Trader Profile marketplace — trade history pagination

## 📋 CHANGES 2026-03-28 (Marketplace)
### Backend
- feat: marketplace GET /api/public/traders — server-side filter, sort, pagination
- feat: filter: min 10 trades, alpha_score > 0, status active
- feat: sort: alphaScore, roi, win_rate, profit_factor, net_pnl, drawdown, trades
- feat: filter params: risk, platform, search, page, limit
- Backend: crunchalpha-v3:production-202604160220

### Frontend
- feat: MarketplacePage Copy Traders — server-side filter+sort+pagination
- feat: filter bar: Sort, Risk Level, Platform, Search
- feat: pagination UI (muncul jika >12 traders)
- feat: empty state jika tidak ada hasil filter
- fix: card data fields — support camelCase dari backend baru
- Frontend: crunchalpha-frontend-v3:prod-202604150205

## 🐳 CURRENT PRODUCTION (Updated 2026-03-28)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## ⚠️ PENDING
1. EA MT4 distribute ke publisher external
2. Overleveraging flag formula review
3. EA Keys management di Copy Settings frontend
4. Trigger copy engine dari EA trader push
5. Trader Profile marketplace — trade history pagination

## 🔑 DEPLOY COMMAND (WAJIB PAKAI ENV-FILE)
```bash
docker rm -f crunchalpha-backend && \
docker run -d --name crunchalpha-backend \
  --network root_crunchalpha-net \
  -p 8090:8090 \
  --env-file /root/.env-crunchalpha \
  --restart unless-stopped \
  --health-cmd="wget -qO- http://localhost:8090/health || exit 1" \
  --health-interval=30s \
  crunchalpha-v3:production-202604160220
```
⚠️ JANGAN deploy tanpa --env-file, email akan pakai mock mode!

## 📋 CHANGES 2026-03-29
### Backend
- feat: /api/auth/impersonate — exchange impersonate token untuk JWT
- feat: impersonate_tokens table di DB
- feat: admin endpoints: create user, force verify, reset password, suspend, impersonate, delete trading account
- fix: login blocked kalau email belum verified
- fix: welcome email dikirim setelah verify, bukan saat register
- fix: SMTP env-file wajib dipakai saat deploy backend

### Frontend  
- feat: ImpersonatePage — auto login via URL token
- feat: ImpersonateBanner — banner kuning + Exit button
- feat: Suspend/Unsuspend button di admin Users tab
- fix: duplicate Create User button
- fix: impersonate redirect ke /impersonate?token= (bukan localStorage manual)

## 🐳 CURRENT PRODUCTION (Updated 2026-03-29)
### Backend:
- **Image:** `crunchalpha-v3:production-202604160220
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## 📋 CHANGES 2026-03-29 (Session 2)
### Backend
- feat: impersonate response tambah field role untuk redirect
- fix: email_verified field di GetUserByEmail query
### Frontend
- feat: impersonate redirect sesuai role (investor→/investor, analyst→/analyst, trader→/trader)
- feat: ImpersonateBanner pakai useState+useEffect agar reaktif
- fix: duplicate return di ImpersonateBanner
- fix: admin sidebar hapus Cashflow & User Growth (tidak ada route)

## 🐳 CURRENT PRODUCTION (Updated 2026-03-29)
### Backend:
- **Image:** `crunchalpha-v3:production-202604160220
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## 📋 CHANGES 2026-03-31
### Backend
- fix: copy-trader-subscribe — handle no_account error, fix enum 'active'→'ACTIVE'
- fix: upsert user_allocations saat subscribe copy trader
- Backend: crunchalpha-v3:production-202604160220

### Frontend  
- fix: copy trader modal — tampil warning "Link Account First" jika belum punya trader_account
- fix: hapus step "Install EA" dari modal (platform yang handle)
- Frontend: crunchalpha-frontend-v3:prod-202604150205

## 🐳 CURRENT PRODUCTION (Updated 2026-03-31)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Deploy command:** `docker run --env-file /root/.env-crunchalpha ...`

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## ⚠️ PENDING
1. EA MT4 distribute ke publisher external
2. Overleveraging flag formula review
3. EA Keys management di Copy Settings frontend
4. Trigger copy engine dari EA trader push ← NEXT PRIORITY
5. Back button marketplace → kembali ke tab yang benar

## 📋 CHANGES 2026-03-31 (Copy Engine)
### Backend
- feat: TriggerCopyEngine — dipanggil saat EA trader push status=open
- feat: TriggerCopyEngineClose — dipanggil saat EA trader push status=closed
- feat: AUM proportional lot calculation di engine
- feat: Rejection checks: max positions, total alloc >100%, daily loss
- fix: copy_subscriptions query — pakai follower_account_id JOIN trader_accounts
- fix: INSERT copy_events — subquery via trader_accounts bukan investor_id langsung
- Backend: crunchalpha-v3:production-202604160220

## 🐳 CURRENT PRODUCTION (Updated 2026-03-31 08:49)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220

## 📋 CHANGES 2026-04-01 (Risk Normalization)
### Backend
- feat: Risk Normalization Engine — Conservative/Balanced/Aggressive
- feat: estimateSL dari trader history (avg_loss/avg_lots/pip_value)
- feat: calcFinalLot = MIN(prop_lot, risk_lot) 
- feat: simpan prop_lot, risk_lot, estimated_sl, final_lot ke copy_events
- feat: DB migration — risk_level di investor_settings
- Backend: crunchalpha-v3:production-202604160220

## 🐳 CURRENT PRODUCTION (Updated 2026-04-01)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220

## 🎨 DESIGN SYSTEM (WAJIB DIIKUTI - FRONTEND)
- **File:** `src/index.css` — single source of truth untuk semua styling
- **Background:** `#0F172A` (base), `#1E2937` (surface), `#162033` (subtle)
- **Border:** `#334155`
- **Text:** `#F1F5F9` (primary), `#94A3B8` (muted), `#64748B` (faint)
- **Accent:** `#3B82F6` (blue), hover `#2563EB`
- **Success:** `#22C55E` | **Danger:** `#EF4444` | **Warning:** `#F59E0B`
- **Font:** Inter (Google Fonts)
- **Radius:** sm=4px, md=8px, lg=12px, xl=16px
- **NO emoji** di seluruh aplikasi — gunakan SVG icon
- **NO gradient** background — solid color only
- **Branding:** "CrunchAlpha" (bukan CRUNCHALPHA), tagline "Risk Controlled Copy Trading"
- **CSS variables prefix:** `--bg`, `--text-main`, `--accent`, `--border`, dll (lihat index.css)
- Semua halaman baru HARUS pakai class dari `index.css` — jangan inline style kecuali terpaksa

## 🎨 DESIGN SYSTEM (WAJIB DIIKUTI - FRONTEND)
- **File:** `src/index.css` — single source of truth untuk semua styling
- **Background:** `#0F172A` (base), `#1E2937` (surface), `#162033` (subtle)
- **Border:** `#334155`
- **Text:** `#F1F5F9` (primary), `#94A3B8` (muted), `#64748B` (faint)
- **Accent:** `#3B82F6` (blue), hover `#2563EB`
- **Success:** `#22C55E` | **Danger:** `#EF4444` | **Warning:** `#F59E0B`
- **Font:** Inter (Google Fonts)
- **Radius:** sm=4px, md=8px, lg=12px, xl=16px
- **NO emoji** di seluruh aplikasi — gunakan SVG icon
- **NO gradient** background — solid color only
- **Branding:** "CrunchAlpha" (bukan CRUNCHALPHA), tagline "Risk Controlled Copy Trading"
- **CSS variables prefix:** `--bg`, `--text-main`, `--accent`, `--border`, dll (lihat index.css)
- Semua halaman baru HARUS pakai class dari `index.css` — jangan inline style kecuali terpaksa

## 🐳 CURRENT PRODUCTION (Updated 2026-04-08)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-landing-fix2`
- **Port:** 5176 (internal), via nginx https

## 📋 CHANGES 2026-04-08
### Frontend - Full Redesign Complete
- feat: LandingPage — design system baru, no emoji, clean colors, semua teks non-data pakai text-muted
- feat: AboutUs page — /about, founder story, clean layout
- feat: ForgotPassword & ResetPassword — pakai auth-shell CSS classes, konsisten
- fix: slogan "Risk Controlled Copy Trading" warna accent (biru) di semua navbar
- fix: Hendri Saputro title — hapus CEO, jadi "Founder, CrunchAlpha"
- fix: landing page section labels tidak warna-warni — pakai text-faint

## 🐳 CURRENT PRODUCTION (Updated 2026-04-09)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-profile`
- **Port:** 5176 (internal), via nginx https

## 📋 CHANGES 2026-04-09
### Frontend - Dashboard & Profile Redesign
- feat: TraderDashboard — no emoji, design system vars, clean tabs, pagination
- feat: ProfilePage — clean form layout, readonly fields styled, design system
- fix: Sidebar — hapus text-transform uppercase dari .sidebar-logo CSS
- fix: index.css .sidebar-logo — letter-spacing 0.01em, no uppercase

## 🐳 CURRENT PRODUCTION (Updated 2026-04-09)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Port:** 5176 (internal), via nginx https
- **Changes:** Mobile responsive, EA connection status display, design system seragam

### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:** connection_status dari DB, cron 5min, scan order fix, ea_verified filter

## 📋 CHANGES 2026-04-09
### Backend
- feat: connection_status dari DB — cron update setiap 5 menit
- fix: scan order mismatch — connection_status before last_sync_at
- feat: ea_verified filter marketplace

### Frontend
- feat: EA connection status di Accounts tab (Connected/Disconnected/Pending EA)
- feat: mobile responsive landing page
- fix: design system seragam semua halaman

## 🐳 CURRENT PRODUCTION (Updated 2026-04-09 06:01)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Fix:** minimum trades AlphaRank naik dari 10 ke 20

## 📋 CHANGES 2026-04-09 (Backend)
- fix: minimum trades AlphaRank — 10 → 20
- fix: marketplace filter — ea_verified + alpha_ranks exist

## ⚠️ PENDING
- feat: currency label di dashboard & marketplace — tampilkan CNT/USD/EUR sesuai akun broker, bukan hardcode USD
- feat: marketplace filter — ea_verified=true AND alpha_ranks exist (20+ trades)

## 🐳 CURRENT PRODUCTION (Updated 2026-04-10)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Port:** 8090 (internal), via nginx https

## 📋 CHANGES 2026-04-10
### Backend - Affiliate Admin Module
- feat: affiliate_handler.go — admin affiliate management
- feat: GET /api/admin/affiliates — list semua affiliate + stats + config
- feat: PUT /api/admin/affiliates/:id/commission — set custom commission per affiliate
- feat: POST /api/admin/affiliates/:id/payout — record payout
- feat: PUT /api/admin/affiliates/payout/:payout_id/mark-paid — mark payout paid
- feat: PUT /api/admin/affiliate-config — update mode (flat/tier) + flat_pct
- db: ALTER affiliates ADD custom_commission_pct
- db: INSERT platform_fee_config affiliate_mode=1, affiliate_flat_pct=10
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-clean-colors`

## ⚠️ PENDING (Updated 2026-04-10)
1. Earnings page trader & analyst — tunggu keputusan bisnis alur payout non-custodial
2. Affiliate dashboard frontend redesign — baca commission dari API, sembunyikan tier kalau mode=flat
3. AffiliateAdmin page — frontend admin management affiliate
4. Tools page — hapus calculator, pindah API Keys ke tab Settings di TraderDashboard

## 🐳 CURRENT PRODUCTION (Updated 2026-04-10 04:45)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-affiliate-admin`
- **Changes:** Affiliate dashboard real data from DB, AffiliateAdmin tab in AdminDashboard
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:** GetAffiliateOverview — commissionPct, affiliateMode, isCustomCommission from DB

## 📋 CHANGES 2026-04-10
### Affiliate System
- feat: affiliate_handler.go — admin list, custom commission, payout, config endpoints
- feat: GetAffiliateOverview — return commissionPct + affiliateMode + isCustomCommission
- feat: AffiliateDashboard.jsx — commission from DB, tier hidden on flat mode
- feat: AdminDashboard — tab Affiliates: summary, config, per-affiliate commission override, payout recording
- db: ALTER affiliates ADD custom_commission_pct
- db: INSERT platform_fee_config affiliate_mode=1, affiliate_flat_pct=10

## ⚠️ PENDING (Updated 2026-04-10)
1. Earnings page trader & analyst — tunggu keputusan bisnis alur payout non-custodial
2. Filter admin dari affiliate list — admin tidak boleh jadi affiliate
3. Tools page — hapus calculator, pindah API Keys ke tab Settings di TraderDashboard

## 🐳 CURRENT PRODUCTION (Updated 2026-04-10 05:45)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:** fee_handler — GetDefaultFees dari DB, FeeOverride tambah rebate_share_pct, affiliate_commission_pct, subscription_fee_monthly
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-fee-override`
- **Changes:** TabFees — default values dari API, form tambah rebate/affiliate/subscription fields

## ⚠️ PENDING (Updated 2026-04-10 05:45)
1. Earnings page trader & analyst — tunggu keputusan bisnis
2. Filter admin dari affiliate list
3. Tools page cleanup

## 📋 CHANGES 2026-04-10
### Layer 3 Elite System Intelligence
- feat: layer3.go — 3 modul risk engine baru
- feat: Modul 1 Behavior Shift — lot spike, win rate drop, SL skip, erratic sizing
- feat: Modul 2 Market Regime — volatility proxy dari trade data, loss streak
- feat: Modul 3 Adaptive DD Scaling — DD tiers + active flags penalty
- feat: Final multiplier = M1 × M2 × M3, cap 0.30–1.00, zero on-the-fly
- feat: Auto-apply ke copy lot di copy_trader_engine (baca dari DB)
- db: alpha_ranks tambah layer3_multiplier, layer3_status, layer3_reason, layer3_detail, layer3_calculated_at
- note: investor tidak bisa override Layer 3 — sistem proteksi final

## 🐳 CURRENT PRODUCTION (Updated 2026-04-10)
### Backend:
- **Image:** crunchalpha-v3:production-202604160220
- **Changes:** Layer 3 Elite System Intelligence live

## 🐳 CURRENT PRODUCTION (Updated 2026-04-10 Layer3 Complete)
### Backend:
- **Image:** crunchalpha-v3:production-202604160220
- **Changes:**
  - Layer 3 system_mode: FULL_ACTIVE / MONITORING / DEFENSIVE / PROTECTED
  - Layer 3 soft_reasons: investor-friendly language
  - detailed_handler: zero on-the-fly, all from DB
  - API: layer3.multiplier, status, reason, detail, system_mode, soft_reasons

## 🐳 CURRENT PRODUCTION (Updated 2026-04-10 Layer3 Complete)
### Backend:
- **Image:** crunchalpha-v3:production-202604160220
- **Changes:**
  - Layer 3 system_mode: FULL_ACTIVE / MONITORING / DEFENSIVE / PROTECTED
  - Layer 3 soft_reasons: investor-friendly language
  - detailed_handler: zero on-the-fly, all from DB
  - API: layer3.multiplier, status, reason, detail, system_mode, soft_reasons

## 🐳 CURRENT PRODUCTION (Updated 2026-04-11)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:** fix SubRow Layer3 fields, earnings endpoint, duplicate account check

## 📋 CHANGES 2026-04-11
### Backend
- fix: SubRow struct — tambah Layer3 fields (RiskLevel, Layer3Multiplier, Layer3Status, Layer3SystemMode, Layer3Reason)
- feat: GET /api/trader/earnings — earnings summary + per-investor breakdown
- fix: duplicate account_number check — block register akun yang sudah terdaftar user lain

## 🐳 CURRENT PRODUCTION (Updated 2026-04-11)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:** audit log — logAudit helper, catat impersonate/suspend/reset_password/force_verify/delete_user
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`

## ⚠️ PENDING (Updated 2026-04-11)
1. Earnings page trader & analyst — tunggu keputusan bisnis
2. Filter admin dari affiliate list
3. Tools page cleanup
4. Audit log untuk fee config change + fee override add/delete

## 🐳 CURRENT PRODUCTION (Updated 2026-04-11)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:**
  - Layer 3 Elite System Intelligence — LIVE
  - allocation repository fix — layer3 fields di SELECT
  - copy-trader-subscriptions — layer3 fields exposed
  - detailed_handler — zero on-the-fly, all from DB

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Changes:**
  - Layer3Badge di CopyTradersTab — system mode, multiplier, reason
  - Sidebar fix — desktop always visible, mobile drawer
  - MainLayout simplified

## ⚠️ PENDING (Updated 2026-04-11)
1. Earnings page trader & analyst — tunggu keputusan bisnis
2. Affiliate dashboard redesign dark theme
3. Tools page — hapus calculator, pindah API Keys ke Settings
4. Input broker account form — cursor lose focus tiap ketik (re-render issue)
5. Layer3Badge di 5177 test — verify tampilan di production

## 🔑 FRONTEND DEPLOY PROCEDURE (WAJIB)
Setiap perubahan frontend HARUS ikuti urutan ini:
1. Edit source di `/var/www/crunchalpha-frontend-v3-SRC/src/`
2. `cd /var/www/crunchalpha-frontend-v3-SRC`
3. `npm run build` — compile React ke dist/
4. `docker build -t crunchalpha-frontend-v3:test-xxx .` — build image
5. Test di port 5177
6. Verify tampilan di browser
7. `docker build -t crunchalpha-frontend-v3:prod-202604150205YYYYMMDDHHMM .`
8. Deploy production
9. `git add -A && git commit`

⚠️ JANGAN skip `npm run build` — Docker COPY dist/, bukan src/

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:**
  - Layer 3 behavior guard: DD < 10% → behavior floor 0.75
  - Layer 3 threshold: min 40 trades untuk behavior & volatility check
  - Layer 3 false positive fixed: trader bagus tidak ter-reduce
  - Recalculate semua akun aktif — hasil valid

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Changes:** Layer3Badge live di CopyTradersTab

## ⚠️ PENDING (Updated 2026-04-12)
1. Earnings page trader & analyst — tunggu keputusan bisnis
2. Affiliate dashboard redesign dark theme
3. Tools page — hapus calculator, pindah API Keys ke Settings
4. Input broker account form — cursor lose focus tiap ketik (re-render issue)
5. Layer 3 — recalculate otomatis periodik (sekarang hanya saat EA push)

## 📋 CHANGES 2026-04-12 (Layer 3 Cron)
### Backend - Layer 3 Periodic Recalculate
- feat: cron goroutine setiap 6 jam — recalculate Layer 3 semua akun aktif
- Layer 3 tidak lagi hanya update saat EA push
- Log: [Layer3Cron] Recalculated N accounts
- Image: crunchalpha-v3:production-202604160220

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v9`
- **Port:** 5176 (internal), via nginx https
- **Changes:**
  - Mobile sidebar drawer — hamburger menu, slide dari kiri, overlay close
  - Sign out button di dalam sidebar nav
  - Landing page clean colors — problem/solution section no colored background
  - Topbar mobile — CrunchAlpha brand + hamburger
  - CSS mobile fix — app-sidebar drawer pattern

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12 late)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v10`
- **Changes:**
  - Dashboard grids responsive — auto-fit minmax, no horizontal overflow
  - TraderDashboard, InvestorDashboard, AnalystDashboard all fixed

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12 final)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v11`
- **Changes:**
  - TraderDashboard tab bar scrollable di mobile
  - Header flex-wrap — account selector tidak overflow
  - Dashboard grids semua responsive auto-fit

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12 v12)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v12`
- **Changes:**
  - InvestorDashboard tab bar scrollable, grid 1fr 1fr → auto-fit
  - AnalystDashboard tab bar scrollable

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12 v16)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v16`
- **Changes:**
  - Hapus semua emoji dari 24 files — no emoji policy enforced
  - Investor settings risk level buttons flex-wrap mobile
  - Investor & analyst tab bar scrollable
  - Copy traders card stats flex-wrap

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12 v17)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v17`
- **Changes:**
  - AnalystDashboard signal sets table scrollable mobile
  - Header buttons flex-wrap
  - Remove remaining emoji

## 🐳 CURRENT PRODUCTION (Updated 2026-04-12 16:30)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:**
  - fix: per-pair DD — peak-to-trough (Layer1) + equity vs peak (Layer2) + floating per symbol (Layer2b)
  - fix: per-pair peakBalance init dari initialDeposit bukan 0
  - fix: per-pair DD pakai peak global bukan per-symbol
  - debug log DD-DEBUG ditambah sementara
  - DB: alpha_ranks per-pair max_drawdown_pct direset (one-time fix formula lama)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v18`
- **Changes:** Mobile responsive fixes — analyst/investor dashboard, no emoji

## ⚠️ PENDING (Updated 2026-04-12)
1. Earnings page trader & analyst — tunggu keputusan bisnis
2. Filter admin dari affiliate list
3. API Keys management di tab Accounts trader dashboard
4. Hapus DD-DEBUG log setelah per-pair DD verified benar
5. EA MT4 verify — reset GlobalVariable LastTicket

## 🐳 CURRENT PRODUCTION (Updated 2026-04-13)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:**
  - fix: DD Layer2 normalized equity = equity + totalWithdrawals
  - fix: withdrawal reset peak, guard peakBalance < 0
  - fix: per-pair DD = global DD untuk single-pair account
  - verified: SarMt5 DD 3.8%, GoldCentrum per-pair 48.13%
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-mobile-v18`

## ⚠️ PENDING (Updated 2026-04-13)
1. Earnings page trader & analyst — tunggu keputusan bisnis
2. Filter admin dari affiliate list
3. API Keys management di tab Accounts trader dashboard
4. Hapus DD-DEBUG log dari service.go setelah verified
5. EA MT4 verify data masuk DB

## 🐳 CURRENT PRODUCTION (Updated 2026-04-13 05:05)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Changes:**
  - Analyst summary tab grid responsive — 1fr 320px → auto-fit minmax(280px)
  - AlphaScore banner flex-wrap mobile
  - Signal sets table scrollable

## 🐳 CURRENT PRODUCTION (Updated 2026-04-13 05:13)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Changes:**
  - Hamburger menu hidden di desktop, muncul hanya di mobile
  - X button sidebar hidden di desktop

## 🐳 CURRENT PRODUCTION (Updated 2026-04-13 05:23)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Changes:**
  - Landing page desktop nav — Sign out muncul saat sudah login

## 🐳 CURRENT PRODUCTION (Updated 2026-04-13 08:30)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:**
  - fix: SaveTrade hapus WHERE status constraint — sl/tp/min_equity bisa update di closed trades
  - fix: MT4 EA v3.0 always full sync on init (reset LastTicket=0)
  - DD-DEBUG log masih aktif (hapus setelah verified)

## ⚠️ PENDING (Updated 2026-04-13)
1. Earnings page — tunggu keputusan bisnis
2. Filter admin dari affiliate list
3. API Keys di tab Accounts trader dashboard
4. Hapus DD-DEBUG log setelah verified
5. Currency cent normalization (akun cent tampil dalam USD)

## 📋 EA STATUS FINAL (2026-04-13)
- EA MT5 Publisher v3.1: HTTP Direct, SL/TP from cache, min_equity tracking ✅
- EA MT4 Publisher v3.0: HTTP Direct, SL/TP, min_equity tracking ✅
- Verified: sl/tp/min_equity masuk DB saat trade close
- DD Layer3 (min_equity) aktif untuk trade baru ke depan

## 📋 EA STATUS FINAL (2026-04-13)
- EA MT5 Publisher v3.1: HTTP Direct, SL/TP from cache, min_equity tracking ✅
- EA MT4 Publisher v3.0: HTTP Direct, SL/TP, min_equity tracking ✅
- Verified: sl/tp/min_equity masuk DB saat trade close
- DD Layer3 (min_equity) aktif untuk trade baru ke depan

## 🔑 BACKEND ENV (WAJIB SAAT DOCKER RUN)
- DB_HOST=crunchalpha-postgres
- DB_USER=alpha_user
- DB_PASSWORD=alpha_password
- DB_NAME=crunchalpha
- RESEND_API_KEY=re_CXt3D9BE_47scGRGR2DD84WWdayHo6Ksb
- EMAIL_MODE=smtp
- SMTP_HOST=smtp.resend.com
- SMTP_PORT=465
- SMTP_USER=resend
- SMTP_FROM=noreply@crunchalpha.com
- SMTP_PASS=re_CXt3D9BE_47scGRGR2DD84WWdayHo6Ksb

## 🐳 CURRENT PRODUCTION (Updated 2026-04-14)
### Backend:
- **Image:** crunchalpha-v3:production-202604160220
- **Changes:**
  - feat: POST /api/investor/auto-allocate — AlphaScore-based proportional allocation
  - fix: route registered in main.go
  - fix: copy engine reject lot < 0.01 — prevents disproportionate position
  - fix: settings GET scan mismatch — risk_level now saved & read correctly
  - feat: trader name exposed in allocation API

### Frontend:
- **Image:** crunchalpha-frontend-v3:prod-202604150205
- **Changes:**
  - feat: Auto/Manual allocation toggle di CopyTradersTab
  - fix: allocation save validation > 100%
  - fix: progress bar hijau di bawah slider — dihapus
  - fix: analyst dashboard missing closing div

## 🐳 CURRENT PRODUCTION (Updated 2026-04-14)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:** Earnings dashboard — trader & analyst earnings endpoint, withdraw request, monthly chart, DB table earnings_withdrawals

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150205`
- **Changes:** TraderEarnings & AnalystEarnings — real API (no dummy), withdraw form, monthly chart, per-investor/subscriber table, withdrawal history

## ⚠️ PENDING (Updated 2026-04-14)
1. Earnings page — admin panel untuk approve/reject withdrawal request
2. Affiliate dashboard redesign dark theme
3. Tools page — pindah API Keys ke Settings tab

## 🐳 CURRENT PRODUCTION (Updated 2026-04-15)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-tools`
- **Port:** 5176 (internal), via nginx https
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Changes:** DD formula revision

## 📋 CHANGES 2026-04-15
### Frontend
- feat: Tools page redesign — tab layout, EA setup collapsible, Accounts tab
### Backend
- fix: DD formula revision — production-202604151350

## 🐳 CURRENT PRODUCTION (Updated 2026-04-15)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160220
- **Fix:** analyst-subscriptions API return alphaScore & alphaGrade
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150913`
- **Changes:** Copy Analyst parity — alphaScore, AUM, Smart Insights sama dengan Copy Traders tab

## 🐳 CURRENT PRODUCTION (Updated 2026-04-15 09:35)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604150935`
- **Fix:** rebuild dist — alphaScore & AUM visible di Copy Analyst cards

## 🔑 FRONTEND BUILD WORKFLOW (WAJIB)
- SELALU jalankan `npm run build` di `/var/www/crunchalpha-frontend-v3-SRC` SEBELUM docker build
- Tanpa npm run build, dist/ masih bundle lama dan patch tidak masuk ke image

## 📋 CHANGES 2026-04-16
### Backend - EA Investor Auth Fix
- fix: EAMiddleware (X-EA-Key) dipasang di semua /api/ea/investor/* routes
- fix: semua EA handlers ganti X-Investor-ID legacy → getEAInvestorID() (support keduanya)
- fix: EAPushEquity upgrade — simpan 7 fields: equity, balance, margin, free_margin, floating, open_lots, open_positions ke investor_ea_keys
- fix: investor_settings.investor_equity = SUM(equity) dari semua EA keys investor
- db: ALTER investor_ea_keys ADD balance, margin, free_margin, floating, open_lots, open_positions
- Backend: crunchalpha-v3:production-202604160XXX

## 🐳 CURRENT PRODUCTION (Updated 2026-04-16)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604160818`
- **Changes:** Overview portfolio allocation — analyst subs muncul, traderName fix, badge SIGNAL untuk analyst

## 🐳 CURRENT PRODUCTION (Updated 2026-04-16)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604160847`
- **Fix:** investor connection_status — cron sync last_sync_at dari investor_ea_keys untuk role=follower accounts

## 📋 CHANGES 2026-04-16
### Backend - Investor Connection Status Fix
- fix: cron 5min tambah sync last_sync_at & ea_verified untuk follower accounts dari investor_ea_keys
- fix: trader_accounts role=follower kini auto-update connection_status = connected saat EA investor push equity
- fix: akun 20686862 (InvAlpha2) role diubah provider → follower (business rule: 1 akun = 1 role)
- fix: investor_settings.mt5_account diisi untuk link EA key ke settings
- rule: account_number tidak boleh double role (provider + follower)

## 🐳 CURRENT PRODUCTION (Updated 2026-04-16 investor-fix)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604161100` (sesuai timestamp)
- **Fix:** generateKey otomatis isi mt5Account dari accounts[0] saat generate EA key

## 📋 CHANGES 2026-04-16 (Frontend)
- fix: generateKey — mt5Account tidak lagi kosong, ambil dari broker account pertama yang terdaftar

## 🐳 CURRENT PRODUCTION (Updated 2026-04-16 15:17)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604161517`
- **Changes:** EA key section di TraderDashboard — generate, list, delete EA keys per account
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604161455`
- **Changes:** trader EA keys CRUD API, DD metrics historical peak, rate limiter cleanup

## 🐳 CURRENT PRODUCTION (Updated 2026-04-16 23:34)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604162334`
- **Changes:** EAKeySection trader — UI disamakan dengan investor (simple: status + generate new key)

## 🐳 CURRENT PRODUCTION (Updated 2026-04-17)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604170033`
- **Port:** 8090 (internal), via nginx https
- **Changes:** feat: trader EA keys — generate/list/delete via api_keys table, max 3 keys per account

## 🐳 CURRENT PRODUCTION (Updated 2026-04-17 02:27)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604170227`
- **Port:** 8090 (internal), via nginx https
- **Git:** master branch (df20609)

## 📋 CHANGES 2026-04-17
### Backend - DD Calculation Fix
- fix: DD logic salah — rewrite di internal/alpharank/dd_metrics.go
- fix: rateLimiter.StartCleanup comment-out (build fix)
- Backend: crunchalpha-v3:production-202604170227

## 🐳 CURRENT PRODUCTION (Updated 2026-04-17 07:53)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604170753`

## 📋 CHANGES 2026-04-17
### Backend
- fix: DD calculation logic fix
- fix: investor routes — tambah copy-trader-unsubscribe, analyst-unsubscribe, analyst-subscriptions, analyst-feed, analyst-subscribe, ea-keys GET/POST/DELETE
- fix: investor connection_status — cron sync dari investor_ea_keys untuk role=follower
- fix: akun 20686862 role provider → follower (business rule: 1 akun = 1 role)
- fix: generateKey investor — mt5Account otomatis dari accounts[0]

## 🐳 CURRENT PRODUCTION (Updated 2026-04-17 08:24)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-$(date +%Y%m%d%H%M)`
- **Port:** 8090 (internal), via nginx https
- **Git:** master branch

## 📋 CHANGES 2026-04-17 (DD Fix Final)
### Backend - DD Zero On-The-Fly
- fix: UpdateDrawdownMetrics return (maxDD, currentDD) — tidak perlu baca DB lagi
- fix: CalculateForAccount override metrics.MaxDrawdownPct dari UpdateDrawdownMetrics
- fix: hapus GREATEST di upsert alpha_ranks — DD bisa turun
- result: account 6a725323 DD 49.18% → 17.90% ✅
- Backend: crunchalpha-v3:production-202604170824

## 🐳 CURRENT PRODUCTION (Updated 2026-04-17 08:46)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604170846`
- **Port:** 8090 (internal), via nginx https
- **Git:** master branch

## 📋 CHANGES 2026-04-17 (DD Fix — Withdrawal)
### Backend - DD Exclude Withdrawal
- fix: withdrawal exclude dari DD events — WD adalah profit diambil, bukan loss
- fix: DD hanya dari deposit + closed trades
- result: account 13038460 DD 100% → 5.09% ✅
- result: account 20661475 DD 17.90% ✅
- Backend: crunchalpha-v3:production-202604170846

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604180316`
- **Port:** 5176 (internal), via nginx https
- **Changes:** fix: EA download URL fix — /api/ea/download/mt5 & mt4 working

## ✅ EA STATUS (Updated 2026-04-18)
- EA Publisher MT5 v3.0: download OK ✅
- EA Publisher MT4 v3.0: download OK ✅
- EA Investor MT5: download OK ✅
- EA Investor MT4: download OK ✅

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604180430`
- **Changes:** feat: per-pair avg_rr — avg_win/avg_loss/risk_reward saved to DB & returned in API

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Backend:
- **Image:** `crunchalpha-v3:production-202604180506`
- **Changes:** floating_profit dari trader_accounts, excessive position size severity DD-based, per-pair filter min 20 trades, avgRR
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604180433`
- **Changes:** PerPairTable — hapus Grade/Score/MaxDD, tambah AvgRR dan Flags
### ⚠️ PENDING
- Per-pair floating profit — EA perlu update profit field di open trades setiap push
- BalanceAtOpen per trade — setelah EA running 24h realtime

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604180833`
- **Changes:** fix: EA key valid untuk semua akun milik user — hapus per-account restriction

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604180856`
- **Changes:** fix: TraderProfile per-pair — hapus Grade/Score/MaxDD/Risk, tambah AvgRR, filter min 20 trades, soft flags

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604180920`
- **Changes:** fix: GetTraderProfile per-pair — hapus grade/score/maxDD/risk, tambah avg_rr, filter min 20 trades

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604181000` (sesuai timestamp deploy)
- **Port:** 5176 (internal), via nginx https
- **Changes:** TraderProfile per-pair — samakan dengan trader dashboard, hapus Grade/Score/MaxDD/Risk, tambah AvgRR & Flags
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604180920`

## 📋 CHANGES 2026-04-18
### DD Fix — Final
- fix: DD exclude withdrawal dari events — WD adalah profit diambil bukan loss
- fix: hapus GREATEST di upsert — DD bisa turun
- fix: UpdateDrawdownMetrics return value — override buildMetrics (zero on-the-fly)
- fix: per-pair DD withdrawal juga di-exclude
### Frontend
- fix: TraderProfile per-pair — hapus Grade/Score/MaxDD/Risk, tambah AvgRR & soft flags
- fix: samakan dengan Performance Per Trading Pair di trader dashboard

## 📋 CHANGES 2026-04-18 (EA MT4 v3.2)
### EA MT4 Publisher v3.2
- feat: equity_at_open di PositionCache, PublishOpen, PublishClose, BuildTradeJsonHistory
- feat: sl/tp di PositionCache + UpdateSLTP() setiap timer
- feat: UpdateOpenProfit() — kirim floating profit tiap timer ke /api/ea/trade/profit
- feat: floating_by_symbol di SendAccountUpdate() — akumulasi per symbol
- parity: semua fitur MT5 v3.2 sekarang ada di MT4 v3.2

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18 10:19)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604181019`
- **Port:** 8090 (internal), via nginx https
- **Changes:** EA MT4 v3.2 compiled ex4 — equity_at_open, sl/tp, floating_by_symbol, UpdateOpenProfit

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604181100`
- **Changes:** AddAccountModal — tambah Account Type Standard/Cent, currency USC/USD auto-set

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18 11:30)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604181130`
- **Changes:** GET /api/public/traders return currency, PUT /api/admin/trading-accounts/:id/currency
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604181130`
- **Changes:** Marketplace badge CENT (USC), Admin tab Accounts button CCY + modal, AddAccountModal Standard/Cent

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604181537`
- **Changes:** fix: TraderProfile per-pair — title Performance Per Trading Pair, hapus grade/score/maxDD/risk, soft flags

## 🐳 CURRENT PRODUCTION (Updated 2026-04-18 17:00)
### Backend:
- **Image:** `crunchalpha-v3:production-202604181700`
- **Changes:**
  - No SL flag threshold 70% (sesuai TrueAlpha framework), severity berdasarkan DD
  - floating_by_symbol per symbol di SendAccountUpdate EA
  - UpdateOpenProfit endpoint /api/ea/trade/profit
  - floating_profit dari trader_accounts (bukan query open trades)
  - per-pair table: hapus grade/score/maxDD, tambah avgRR, filter min 20 trades

## 🐳 CURRENT PRODUCTION (Updated 2026-04-19)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604191522`

## 📋 CHANGES 2026-04-19
### Backend
- fix: CopyTraderUnsubscribe — followerAccountID dari trader_accounts (bukan uid user)
- fix: investor routes — tambah semua missing routes (unsubscribe, analyst, ea-keys)
### EA Investor
- fix: EA v3.4 — ExtractBool pakai StringFind pattern "key":true (tidak lagi false positif)
- fix: EA v3.4 — ExtractDbl pakai pattern "key": langsung
- Settings sekarang return Signal:true Trader:true ✅

## 🐳 CURRENT PRODUCTION (Updated 2026-04-19 16:02)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604191559`
- **Fix:** CopyTraderUnsubscribe — reset allocation_value=0, fix followerAccountID
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604191601`
- **Fix:** Overview RiskDashboard — fetch copy-trader-subscriptions, filter ACTIVE only, traderName dari subs

## 📋 CHANGES 2026-04-19 (continued)
### Backend
- fix: CopyTraderUnsubscribe — enum CANCELLED (bukan inactive), reset user_allocations ke 0
- fix: CopyTraderUnsubscribe — followerAccountID dari trader_accounts bukan uid
### Frontend
- fix: Copy Traders list — filter hanya ACTIVE (subs→activeSubs)
- fix: Overview RiskDashboard — fetch trader subs sendiri, filter ACTIVE, tampilkan traderName

## 🐳 CURRENT PRODUCTION (Updated 2026-04-20)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604200242`

## 📋 CHANGES 2026-04-20
### Backend - Risk Level Enforcement
- feat: riskLevelMaxRiskPct — Conservative 0.5%, Balanced 1.5%, Aggressive 3.0% AUM per trade
- feat: riskLevelMaxDD — Conservative 5%, Balanced 10%, Aggressive 20% max DD
- feat: DD guard real-time dari investor_ea_keys.floating — stop copy jika DD >= limit
- feat: lot cap per trade berdasarkan risk level × AUM × allocation%
- feat: equity investor ambil dari investor_ea_keys (real-time) bukan investor_settings
- fix: subscribe limit enforcement — max 3 traders + 3 analysts (pending)

## 🔑 RISK LEVEL RULES (LOCKED)
- CONSERVATIVE: Max 0.5% AUM/trade, Max DD 5%
- BALANCED:     Max 1.5% AUM/trade, Max DD 10%
- AGGRESSIVE:   Max 3.0% AUM/trade, Max DD 20%
- DD check: real-time dari floating profit EA push
- Lot formula: traderLot × (AUM/traderEquity) × layer3 → capped by risk level

## 🐳 CURRENT PRODUCTION (Updated 2026-04-20 03:02)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604200302`
- **Fix:** trader name priority — COALESCE(nickname, user.name) bukan sebaliknya

## 🐳 CURRENT PRODUCTION (Updated 2026-04-20)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604200419`
- **Changes:** feat: GET /api/public/market-price/:pair — live price dari ea_price_cache

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604200426`
- **Changes:**
  - AlphaScore™ & Grade hidden until 20 closed signals, progress bar X/20
  - Send Signal form — hapus Analyst Name & Notes field
  - Pair dropdown expanded — Commodity, Forex Major/Minor/Exotic, Index, Crypto, Energy, Agricultural
  - Dark theme fix untuk select dropdown

## 🐳 CURRENT PRODUCTION (Updated 2026-04-20 05:16)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604200516`
- **Changes:**
  - Admin Signal Sets tab — tombol Import History, modal upload CSV
  - Template CSV download (client-side, no auth needed)
  - Status label berwarna: CLOSED_TP/CLOSED_SL/CANCELLED_MANUAL
  - Contoh CANCELLED_MANUAL di template

## 🐳 CURRENT PRODUCTION (Updated 2026-04-20 07:30)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604200730` (sesuai timestamp)
- **Changes:**
  - feat: unified PortfolioTab — Copy Traders + Copy Signals dalam 1 tab "My Portfolio"
  - feat: Signal Feed dipindah ke sub-tab dalam My Portfolio
  - fix: trader name priority — nickname dulu baru user.name
  - fix: AUM display dari investorEquity settings

## 🐳 CURRENT PRODUCTION (Updated 2026-04-20 08:00)
### Backend:
- **Image:** latest crunchalpha-v3 production
- **Fix:** investorEquity DISTINCT ON mt5_account — no duplicate SUM
- **Fix:** GenerateEAKeyForAccount — delete old key sebelum insert baru
- **Fix:** GetSettings return real equity dari investor_ea_keys

## 📋 CHANGES 2026-04-20 (AUM Fix)
- fix: investor_ea_keys duplikat — 3 keys sama mt5_account → SUM salah ($1746 bukan $582)
- fix: GetSettings investorEquity — DISTINCT ON mt5_account ORDER BY last_equity_at DESC
- fix: GenerateEAKeyForAccount — hapus key lama untuk mt5_account yang sama
- fix: auto allocation unified — 1 pool 100% untuk traders + analysts berdasarkan AlphaScore

## 🐳 CURRENT PRODUCTION (Updated 2026-04-20)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604201224`
- **Changes:** Auto-allocate unified pool — traders + analysts share 100% proportional by AlphaScore; RiskDashboard fetch traderSubs + accounts

## 🐳 CURRENT PRODUCTION (Updated 2026-04-21)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604210020`
- **Changes:**
  - fix: DD formula — pure equity_snapshots running peak-to-trough
  - fix: normalized_equity = equity + withdrawals (WD bukan loss)
  - fix: hapus GREATEST di upsert global — dd_metrics.go sudah return max DD dari full history
  - fix: hapus deposit double-count di per-pair DD calculation


## 🐳 CURRENT PRODUCTION (Updated 2026-04-21 01:20)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604210120`
- **Changes:**
  - fix: DD formula — pure equity_snapshots running peak-to-trough, normalized equity+withdrawals
  - fix: hapus GREATEST di upsert global — tidak persist nilai salah
  - fix: per-pair net_pnl include floating dari floating_by_symbol
  - fix: hapus DD layer lama (Layer2, Layer2b) di buildMetricsForSymbol


## 🐳 CURRENT PRODUCTION (Updated 2026-04-21 01:30)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604210130`
- **Changes:** hapus debug logs DD-DEBUG dan DEBUG-FLOAT

## 🐳 CURRENT PRODUCTION (Updated 2026-04-21)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604210204`
- **Changes:**
  - fix: analyst marketplace filter min 20 closed_signals — tidak muncul di leaderboard/marketplace jika belum cukup data
  - feat: analyst subscriptions tambah closedSignals field untuk auto-allocate filter

## 🐳 CURRENT PRODUCTION (Updated 2026-04-21)
### Backend:
- **Image:** `crunchalpha-v3:production-202604210334`
- **Changes:**
  - fix: copy-trader-subscriptions query semua akun investor (bukan LIMIT 1)
  - fix: followerAccountId, followerAccountNumber, followerPlatform di response
  - fix: analyst subscriptions tambah closedSignals field
  - fix: subscribe copy trader terima followerAccountId dari frontend
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604210334`
- **Changes:**
  - feat: subscribe modal tampilkan dropdown pilih akun investor
  - fix: My Portfolio tampilkan "via [account]" per copy trader
  - fix: AnalystProfile hide AlphaScore until 20 closed signals

## 🐳 CURRENT PRODUCTION (Updated 2026-04-21)
### Backend:
- **Image:** `crunchalpha-v3:production-202604210425` (atau sesuai timestamp terakhir)
- **Changes:**
  - feat: follower_account_id di user_allocations — allocation per akun investor
  - feat: subscribe copy trader kirim + validasi followerAccountId
  - feat: unsubscribe copy trader kirim followerAccountId
  - feat: GetCopyTraderSubscriptions return semua akun user + follower info
  - db: ALTER TABLE investor_ea_keys tambah trader_account_id, risk_level, max_daily_loss_pct, max_open_trades
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604210425` (atau sesuai timestamp terakhir)
- **Changes:**
  - feat: My Portfolio group by follower account dengan header per akun
  - feat: subscribe modal dropdown pilih akun investor
  - feat: unsubscribe + save allocation kirim followerAccountId

## 🐳 CURRENT PRODUCTION (Updated 2026-04-21)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604211600`
- **Changes:** EA Key per account — tiap broker account punya EA key sendiri di Copy Settings
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604211420`
- **Changes:** feat: ea-keys per account routes — GET/POST/DELETE /api/investor/ea-keys

## 🐳 CURRENT PRODUCTION (Updated 2026-04-21 16:30)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604211630`
- **Changes:**
  - My Portfolio: allocation key per account (traderId+followerAccountId) — independent 100% per akun
  - Auto allocate: pass follower_account_id — fix silent fail
  - Warning min 5% AUM per trader
  - Warning over 100% per account

## 🐳 CURRENT PRODUCTION (Updated 2026-04-22)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604221820`
- **Changes:** Auto allocate per follower account — 100% independent per akun; min 5% per trader
### Backend:
- **Image:** `crunchalpha-v3:production-202604221800`
- **Changes:** fix scan order allocation — follower_account_id/number/platform sebelum status

## 🐳 CURRENT PRODUCTION (Updated 2026-04-22 19:00)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604221900`
- **Changes:**
  - My Portfolio: AUM per akun dari ea_keys equity (bukan global sum)
  - Overview: Portfolio Allocation per follower account (bukan breakdown trader/signal)
  - Auto allocate: 100% per follower account independent
  - Fix: account_role "follower" (bukan "investor") saat add account

## 🐳 CURRENT PRODUCTION (Updated 2026-04-22)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604222000`
- **Changes:**
  - fix: signal history tambah closedAt — struct, query, scan
  - fix: import signal upsert by (set_id, pair, direction, issued_at) — no duplicate
  - db: unique index uq_analyst_signals_import on analyst_signals
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604222100`
- **Changes:**
  - fix: history table tambah Open Time + Close Time columns
  - fix: format waktu bersih — hapus +00:00/Z suffix
  - fix: hapus set_name sub-label di pair cell history

## 🐳 CURRENT PRODUCTION (Updated 2026-04-22 22:00)
### Backend:
- **Image:** `crunchalpha-v3:production-202604222200`
- **Changes:**
  - feat: signal set limit 2→5 per analyst
  - feat: description field di analyst_signal_sets (DB + API)
  - fix: signal history closedAt — struct, query, scan
  - fix: import signal upsert — no duplicate by (set_id,pair,direction,issued_at)
  - fix: signal history order by issued_at DESC
  - db: ALTER TABLE analyst_signal_sets ADD COLUMN description text
  - db: unique index uq_analyst_signals_import on analyst_signals
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604222200`
- **Changes:**
  - feat: Create/Edit Signal Set modal tambah Strategy Description field
  - feat: stat card SIGNAL SETS 2/2 → 5/5
  - fix: history table Open Time + Close Time columns
  - fix: format waktu bersih — hapus +00:00/Z
  - fix: hapus set_name sub-label di pair cell

## 🐳 CURRENT PRODUCTION (Updated 2026-04-22 19:45)
### Backend:
- **Image:** `crunchalpha-v3:production-202604221945`
- **Changes:**
  - TraderAccount struct tambah Role field
  - GetUserAccounts SELECT & Scan include role
  - Fix: 17 scan columns match query

## 🐳 CURRENT PRODUCTION (Updated 2026-04-22 20:00)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604222000`
- **Changes:** DD Guard baca floating real dari eaKeys (bukan hardcode 0)

## 🐳 CURRENT PRODUCTION (Updated 2026-04-22 20:30)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604222030`
- **Changes:** 
  - Copy Settings: hapus Risk Level (sudah ada per-account di My Portfolio)
  - Max Daily Loss & Max Open Trades: tambah keterangan "per account"

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 03:10)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604230310`
- **Changes:**
  - Fix: loadAccounts di SettingsTab missing .then() handler
  - EA Key per Account sekarang tampil dengan benar di Copy Settings

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 03:30)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604230330`
- **Changes:**
  - Overview: Risk section per-account (risk level + AUM per akun)
  - Overview: DD Guard per-account dengan progress bar masing-masing
  - Fix: loadAccounts missing .then() di SettingsTab

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 05:45)
### Backend:
- **Image:** `crunchalpha-v3:production-202604230545`
- **Changes:**
  - fix: import upsert ON CONFLICT tanpa WHERE — issued_at tipe text bukan timestamp
  - db: recreate uq_analyst_signals_import tanpa WHERE clause

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 09:40)
### Backend:
- **Image:** `crunchalpha-v3:production-202604230940`
- **Changes:**
  - fix: copy engine query filter per follower_account_id
  - fix: allocation check per follower account (bukan total semua akun)
  - fix: acct_equity dari investor_ea_keys (bukan investor_settings global)
  - fix: checkCopyRejection tambah followerAccountID parameter
  - TESTED: copy trade EXECUTED di akun 20686862 ✅

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 10:05)
### Backend:
- **Image:** `crunchalpha-v3:production-202604231005`
- **Changes:**
  - fix: sanitizeTimestamp() helper — OpenTime < 2000-01-01 diganti time.Now()
  - fix: berlaku untuk SaveTrade & SyncTrade (INSERT path sebelumnya tidak ada guard)
  - Root cause: to_timestamp(0) langsung jadi 1970-01-01 saat INSERT pertama

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 11:15)
### Backend:
- **Image:** `crunchalpha-v3:production-202604231110`
- **Changes:**
  - fix: copy engine SL=0 ke follower (sebelumnya pakai OpenPrice → Invalid stops di broker)
  - fix: follower_account_id dari trader_accounts (bukan investor_ea_keys)
  - fix: totalAlloc check per follower_account_id (bukan total semua alokasi)
  - fix: sanitizeTimestamp() — epoch-0 open_time tidak masuk DB
  - TESTED: copy trade EXECUTED akun 20686862 ✅

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 16:55)
### Backend:
- **Image:** `crunchalpha-v3:production-202604231655`
- **Changes:**
  - fix: calcFinalLot — propLot < 0.01 SKIP (tidak copy, bukan fallback ke 0.01)
  - fix: layer3 multiplier inside calcFinalLot (defensive by sistem)
  - fix: order history query — IN multiple follower accounts (bukan LIMIT 1)
  - Logic: propLot gate → layer3 → min 0.01 jika lolos gate

## 🐳 CURRENT PRODUCTION (Updated 2026-04-23 17:20)
### Backend:
- **Image:** `crunchalpha-v3:production-202604231720`
- **Changes:**
  - fix: GET /api/investor/trade-copies — query via copy_events JOIN, support multiple follower accounts
  - fix: GET /api/investor/order-history — IN multiple follower accounts

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24 00:10)
### Backend:
- **Image:** `crunchalpha-v3:production-202604231740`
- **Changes:**
  - fix: copy_executions INSERT param mismatch — remove duplicate eventID
  - fix: trade-copies query via copy_events JOIN, multiple follower accounts
  - fix: propLot gate <0.01 SKIP — no fallback to minimum
  - fix: layer3 inside calcFinalLot
  - TESTED: copy_executions terisi ✅ OPEN+CLOSE EXECUTED ✅

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24 08:00)
### Backend:
- **Image:** `crunchalpha-v3:production-202604240800`
- **Changes:**
  - fix: GetTradeCopies — JOIN copy_events, IN multiple follower accounts
  - fix: EA Investor v3.41 — EventSetTimer 5s → 2s

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24 09:00)
### Backend:
- **Image:** `crunchalpha-v3:production-202604240900`
- **Changes:**
  - fix: GetTradeCopies — tambah symbol, direction, action, close_price, profit, close_time
  - fix: JOIN trades via follower_ticket untuk close price & profit
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202604240109`
- **Changes:**
  - feat: Trade Copies table — tambah Action, Type, Close Price, P&L kolom
  - fix: direction 0=BUY 1=SELL mapping

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24 09:30)
### Backend: crunchalpha-v3:production-202604240920
### Frontend: crunchalpha-frontend-v3:prod-202604240127
### Pending:
- Close Price & P&L follower — butuh storage follower trade data dari EA
- Type display untuk CLOSE event

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24)
### Backend:
- **Image:** `crunchalpha-v3:production-202604240340`
- **Changes:**
  - feat: GET /trade-copies route terdaftar
  - fix: copy-trade-history query semua akun investor (hapus LIMIT 1)
  - feat: risk level per akun di trader_accounts
  - feat: GET/POST account-risk-levels endpoints

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24)
### Backend:
- **Image:** `crunchalpha-v3:production-202604240510`
- **Changes:**
  - feat: trade-copies LATERAL JOIN executed_price, close_price, profit
  - fix: copy-trade-history query semua akun (IN instead of LIMIT 1)
  - feat: GET /trade-copies route registered

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24 11:15)
### Backend:
- **Image:** `crunchalpha-v3:production-202604241115`
- **Changes:**
  - fix: GetCopyTraderSubscriptions — hapus kolom duplikat, fix scan order
  - fix: GetTradeCopies — rewrite query pakai copy_events table
  - Both APIs working: subscriptions count=2, trade-copies EXECUTED ✅

## 🐳 CURRENT PRODUCTION (Updated 2026-04-24 11:30)
### Backend:
- **Image:** `crunchalpha-v3:production-202604241130`
- **Changes:** fix: GetTraderProfile — hapus orphan cev columns; Marketplace trader profile working

## 📋 CHANGES 2026-04-25
### EA MT4 Investor v2.1 Fix
- fix: ExtractBool — smart quotes → escaped straight quotes
- fix: ExtractDbl — cari "key": dengan colon agar nilai numerik ter-parse
- Result: Signal:true Trader:true MaxLot:0.1 MaxDD:5.0% ✅
### EA STATUS (Updated)
- EA MT5 Investor v3.50: HTTPS crunchalpha.com ✅
- EA MT4 Investor v2.1: HTTPS crunchalpha.com ✅ (fix ExtractBool & ExtractDbl)

## 🐳 CURRENT PRODUCTION (Updated 2026-04-25 09:10)
### Backend: crunchalpha-v3:production-202604250400
### Frontend: crunchalpha-frontend-v3:prod-202604240934
### EA Status:
- EA Investor MT5 v3.50 — SyncTrades(), timer 2s ✅
- EA Investor MT4 v2.51 — SyncTrades(), ExtractBool fix, copyInterval 2s ✅
- Download: /api/ea/download/investor-mt5 & investor-mt4 ✅
### Pending:
- Overview Copy Trade P&L — tunggu SyncTrades data masuk
- Managed VPS deploy — tunggu VPS Windows minggu depan

## 🐳 CURRENT PRODUCTION (Updated 2026-04-25)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604251600`
- **Changes:**
  - feat: POST /api/ea/investor/sync-trades — EA kirim trade history ke DB (investor_trades)
  - feat: GET /api/investor/trade-history — investor fetch trade history dari DB
  - fix: route injection via main.go (bukan routes.go)

## ⚠️ PENDING (Updated 2026-04-25)
1. Frontend OrderHistoryTab — sudah difix di source tapi belum deploy (MarketplacePage.jsx ada build error corruption [e.target] dari sesi sebelumnya, perlu fix dulu)
2. EA MT5 Investor — belum ada SyncTrades(), perlu tambahkan
3. Earnings page trader & analyst
4. Affiliate dashboard redesign dark theme

## 🐳 CURRENT PRODUCTION (Updated 2026-04-25 17:30)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604251730`
- **Changes:**
  - fix: OrderHistoryTab fetch dari /api/investor/trade-history (bukan signal-orders)
  - fix: stats cards — Total Trades, Closed, Open, Total P&L
  - fix: MarketplacePage restore dari git (corruption [S.bg] di working file)

## 🐳 CURRENT PRODUCTION (Updated 2026-04-25 18:20)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604251820`
- **Changes:**
  - feat: Order History — dropdown filter by account (All Accounts / per account)
  - feat: refetch trades saat ganti account
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604251800`
- **Changes:**
  - feat: GET /api/investor/trade-history?account_id= — optional filter by follower_account_id
  - feat: response tambah followerAccount field

## 🐳 CURRENT PRODUCTION (Updated 2026-04-25 18:35)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202604251835`
- **Changes:**
  - fix: accounts state di InvestorDashboard root — pass ke OrderHistoryTab
  - feat: dropdown account filter berfungsi di Order History

## 🐳 CURRENT PRODUCTION (Updated 2026-04-26 15:00)
### Backend:
- **Image:** `crunchalpha-v3:production-202604261500`
- **Changes:**
  - feat: GetInvestorTradeHistory baca dari copy_events (bukan investor_trades)
  - feat: copy-trade-update tambah profit field
  - feat: copy_executions.profit tersimpan saat EA kirim close

## ⚠️ PENDING
1. EA MT5 Investor v3.7 — compile & deploy (profit field di SendCopyTradeUpdate)
2. Data lama 4 trades tanpa open_price — tidak bisa direcovery
3. P&L akan terisi setelah EA v3.7 live dan trader close posisi

## 🐳 CURRENT PRODUCTION (Updated 2026-04-26)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-format`
- **Port:** 5176 (internal), via nginx https
- **Changes:**
  - feat: light/dark theme toggle — pojok kanan bawah, persist di localStorage
  - feat: CSS variables light theme di index.css [data-theme="light"]
  - fix: number formatting standard — fMoney (smart compact $52.6K), fPct (2 decimal), fRatio
  - fix: StatCard font auto-shrink untuk angka panjang
  - fix: semua hardcode hex colors → var(--xxx) di AffiliateDashboard, MarketplacePage, TraderProfile, AnalystProfile, InvestorDashboard, AdminDashboard
  - fix: soft colors — success #86efac, danger #f87171, warning #fcd34d
  - fix: layout konsisten semua dashboard pages — width:100%, hapus maxWidth/margin:auto
  - fix: Marketplace page layout — page-header standard
  - fix: AffiliateDashboard warna angka konsisten pakai CSS vars
  - fix: Analyst/Investor table text → var(--text-muted)
  - ⚠️ DEV NOTE: Selalu pakai var(--xxx) bukan hardcode hex agar otomatis ikut light/dark theme


## 🐳 CURRENT PRODUCTION (Updated 2026-04-27)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-format`
- **Changes:**
  - fix: admin dashboard C.text/C.muted → CSS vars (light theme compatible)
  - fix: card-meta + text-xs CSS class — sub-text grey bukan putih
  - fix: analyst dashboard TD color → var(--text-muted)
  - fix: about/broker/landing navbar hardcode dark → var(--bg-surface)
  - fix: broker page hero gradient → var(--bg-elevated)
  - fix: broker page emoji dihapus dari badges
  - feat: "Brokers" link di navbar landing page

## 🐳 CURRENT PRODUCTION (Updated 2026-04-28 08:50)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604280850`
- **Changes:**
  - fix: auto-close copy_events via SyncTrades — detect manual/SL/TP close di investor
  - fix: maxOpenTrades tidak stuck karena posisi lama tidak ter-close di DB

## 🐳 CURRENT PRODUCTION (Updated 2026-04-28)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:test-format`
- **Changes:**
  - fix: Order History table — GoldCentrum/symbol text-muted bukan bold putih
  - fix: leaderboard landing page — klik trader/analyst → marketplace profile (bukan dashboard)
  - fix: AUM explanation text — "per trader account and per signal set on My Portfolio tab"
  - fix: investor dashboard allocation mode buttons flexWrap mobile
  - fix: mobile scroll utilities di index.css

## 🐳 CURRENT PRODUCTION (Updated 2026-04-29 06:00)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604290600`
- **Changes:**
  - fix: copy_executions INSERT — ce.id langsung dari SELECT (bukan $1::uuid duplikat)
  - fix: WHERE ce.id = $1::uuid (hapus corrupt markdown link yang menyebabkan pq: could not determine data type of parameter $1)
  - Root cause: MT4 investor tidak bisa open posisi karena INSERT copy_executions selalu gagal

## 🐳 CURRENT PRODUCTION (Updated 2026-04-29 07:42)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604280955`
- **Note:** Rollback — image ini MT5 copy trading berjalan normal
- **Pending:** fix INSERT copy_events parameter $1 untuk MT4 investor (harus proper via UI flow)

## ⚠️ PENDING (Updated 2026-04-29)
1. MT4 investor copy tidak jalan — duplicate key di copy_events karena subquery subscription_id tidak match follower_account_id dengan benar
2. Root cause: binary production-202604280955 pakai $17 parameter (CAST), source code sekarang berbeda
3. Fix harus: sesuaikan INSERT copy_events source dengan binary lama — JANGAN ubah tanpa test menyeluruh

## 📋 CHANGES 2026-04-30
### Followers AUM Fix
- fix: followers table — AUM = investor_equity × allocation% (bukan follower_equity yg selalu 0)
- fix: HWM & start_equity reset setiap investor ubah allocation
- fix: trader_account_id auto-link saat investor register EA key
- fix: kolom "Method" dihapus dari followers table, "Equity" → "AUM"
### Production Images
- Backend: crunchalpha-v3:production-$(date +%Y%m%d%H%M)
- Frontend: crunchalpha-frontend-v3:prod-$(date +%Y%m%d%H%M)

## ⚠️ PENDING
1. Earnings page trader & analyst — tunggu keputusan bisnis alur payout non-custodial
2. Affiliate dashboard redesign dark theme  
3. Tools page — hapus calculator, pindah API Keys ke tab Settings di TraderDashboard
4. fix/total-deposit-alpharank — pending deploy (total_deposit tidak tersimpan di alpha_ranks)
   - Branch sudah ada, build clean, tunggu waktu yang tepat untuk deploy

## 🐳 CURRENT PRODUCTION (Updated 2026-05-02)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202605021500`
- **Changes:**
  - fix: pipMult per instrument — XAUUSD mult 100→10, add XAG/Oil/Index/Crypto
  - fix: alpharank response tambah daysActive dari set created_at (bukan first signal)
  - fix: avgHoldHours query dari running_at→closed_at
  - fix: signalSets response tambah daysActive, closedSignals, alphaScore, grade
  - fix: XAAUSD typo → XAUUSD di DB (4 signals)
  - fix: hapus signal test id 1,2,17 (entry <3000)
  - fix: pips DB recalc semua sets dengan pipMult yang benar

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202605021500`
- **Changes:**
  - fix: History tab — hapus kolom R:R & Status badge, pips pakai backend value
  - fix: History pagination 20/page dengan Prev/Next
  - fix: History nomor urut global (#1, #2... across pages)
  - fix: pipMult & pipUnit JS — label dinamis pips/pts per instrument
  - fix: Summary banner — SET AGE (dari daysActive DB) gantikan Closed Signals
  - fix: Hapus 4 top summary cards — info dipindah ke Your Signal Sets table
  - fix: Your Signal Sets table tambah kolom Age & AlphaScore
  - fix: Hapus kolom Subscribers dari table (tidak relevan)
  - fix: Hapus "+ New" duplikat, "+ Send Signal" warna biru
  - fix: Discipline Score bug dihapus dari Statistics tab
  - fix: SummaryStats Avg TP/SL/MaxTP/MaxSL kalikan pipMult

## ⚠️ PENDING (Updated 2026-05-02)
1. Analyst public profile redesign — investor-friendly, risk-focused, style summary
2. Analyst masuk leaderboard (sama seperti trader)
3. Tab Subscribers → ganti jadi Followers, tampilan sama seperti trader followers (AUM dll)
4. avgHoldHours tampil di Summary stats
5. InvestorDashboard.jsx syntax error fix (sudah fixed tapi perlu commit frontend)

## 📋 CHANGES 2026-05-03
### Marketplace Trader Card Redesign (DUB-style)
- feat: card redesign — Risk banner, Grade hero, storytelling investor type
- feat: totalAUM dari copy_subscriptions (equity × allocation%)
- feat: accountAge dari first trade date (bukan ea_first_push_at)
- feat: hapus Net P&L dari card & dropdown → ganti AUM Managed
- feat: hapus risk banner redundant — hanya rank & copying badge
- feat: "Trading for Xmo" label di footer card
- feat: storytelling "Suited for [investor type]" dengan risk tag
- fix: hapus Sort: Net P&L dari dropdown

## 🐳 CURRENT PRODUCTION (Updated 2026-05-03)
### Frontend:
- **Image:** crunchalpha-frontend-v3:prod-$(date +%Y%m%d%H%M)
### Backend:
- **Image:** crunchalpha-v3:production-$(date +%Y%m%d%H%M)

## 🐳 CURRENT PRODUCTION (Updated 2026-05-03)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202605031200`
- **Port:** 5176 (internal), via nginx https
- **Changes:**
  - feat: Subscription tier Basic & Premium untuk Trader & Analyst fees
  - feat: 16 key baru di platform_fee_config (basic/premium fee, trader/analyst/platform/affiliate share)
  - feat: RoleFeesPanel — Performance Fee + Subscription Fee sections terpisah
  - feat: Basic default $10/mo, Premium $30/mo, split 80/20/0
  - fix: AnalystProfile.jsx duplicate useState (setSubModal) compile error

## 🐳 CURRENT PRODUCTION (Updated 2026-05-03 12:15)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202605031215`
- **Changes:** Investor Fees tab — tambah reference section "Fees Paid by Investor" (read-only): Performance Fee trader/analyst + Subscription Basic/Premium trader/analyst, nilai otomatis ikut dari Trader/Analyst Fees config

## 🐳 CURRENT PRODUCTION (Updated 2026-05-03 13:06)
### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202605031306`
- **Changes:**
  - TraderProfile redesign — pillar bars biru netral, stats netral
  - Risk flags netral (hapus warna merah)
  - Survivability/Scalability netral
  - Header storytelling "Suited for X Investors"
  - Trade History — BUY/SELL warna netral
