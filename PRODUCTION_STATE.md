
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
- **Image:** `crunchalpha-v3:production-202603270205`
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202603260931`

## 📋 CHANGES 2026-03-27 (Performance Chart)
### Frontend
- feat: Weekly & Monthly chart tambah $ symbol dan cumulative ROI %
- feat: Monthly ROI = cumulative dalam selected year / deposit
- feat: Weekly ROI = cumulative dalam selected month / deposit
- feat: TraderProfile marketplace chart juga updated

## 🐳 CURRENT PRODUCTION (Updated 2026-03-27)
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202603270345`

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
- **Image:** `crunchalpha-v3:production-202603281246`
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202603281254`

## 📋 CHANGES 2026-03-27
### Backend
- feat: pagination trade history — tambah offset/limit/total di endpoint trader & investor
- feat: add nickname/about edit, strategy field tampil di dashboard & marketplace
- feat: weekly/monthly chart tambah currency symbol dan cumulative ROI %
- Backend: crunchalpha-v3:production-202603270825

### Frontend
- fix: AddAccountModal — onClose sebelum alert, support onAccountAdded prop
- fix: double notification bug setelah add account resolved
- Frontend: crunchalpha-frontend-v3:prod-202603270345

## 📋 CHANGES 2026-03-28
### Backend
- feat: pagination trade history di trader dashboard dan marketplace trader profile
- Backend: crunchalpha-v3:production-202603281246

### Frontend
- fix: AddAccountModal — close before alert, support onAccountAdded prop
- Frontend: crunchalpha-frontend-v3:prod-202603281359

## 🐳 CURRENT PRODUCTION (Updated 2026-03-28)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202603281246`
- **Port:** 8090 (internal), via nginx https
- **Git:** master branch

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202603281359`
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
- Backend: crunchalpha-v3:production-202603281514

### Frontend
- feat: MarketplacePage Copy Traders — server-side filter+sort+pagination
- feat: filter bar: Sort, Risk Level, Platform, Search
- feat: pagination UI (muncul jika >12 traders)
- feat: empty state jika tidak ada hasil filter
- fix: card data fields — support camelCase dari backend baru
- Frontend: crunchalpha-frontend-v3:prod-202603281516

## 🐳 CURRENT PRODUCTION (Updated 2026-03-28)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202603281514`

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202603281516`

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
  crunchalpha-v3:production-YYYYMMDDHHMM
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
- **Image:** `crunchalpha-v3:production-202603281815`
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202603290958`

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
- **Image:** `crunchalpha-v3:production-202603291539`
### Frontend:
- **Image:** `crunchalpha-frontend-v3:prod-202603291544`

## 📋 CHANGES 2026-03-31
### Backend
- fix: copy-trader-subscribe — handle no_account error, fix enum 'active'→'ACTIVE'
- fix: upsert user_allocations saat subscribe copy trader
- Backend: crunchalpha-v3:production-202603310422

### Frontend  
- fix: copy trader modal — tampil warning "Link Account First" jika belum punya trader_account
- fix: hapus step "Install EA" dari modal (platform yang handle)
- Frontend: crunchalpha-frontend-v3:prod-202603310432

## 🐳 CURRENT PRODUCTION (Updated 2026-03-31)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202603310422`
- **Deploy command:** `docker run --env-file /root/.env-crunchalpha ...`

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202603310432`

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
- Backend: crunchalpha-v3:production-202603310849

## 🐳 CURRENT PRODUCTION (Updated 2026-03-31 08:49)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202603310849`

## 📋 CHANGES 2026-04-01 (Risk Normalization)
### Backend
- feat: Risk Normalization Engine — Conservative/Balanced/Aggressive
- feat: estimateSL dari trader history (avg_loss/avg_lots/pip_value)
- feat: calcFinalLot = MIN(prop_lot, risk_lot) 
- feat: simpan prop_lot, risk_lot, estimated_sl, final_lot ke copy_events
- feat: DB migration — risk_level di investor_settings
- Backend: crunchalpha-v3:production-202604010616

## 🐳 CURRENT PRODUCTION (Updated 2026-04-01)
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604010616`

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
- **Image:** `crunchalpha-frontend-v3:prod-202604090237`
- **Port:** 5176 (internal), via nginx https
- **Changes:** Mobile responsive, EA connection status display, design system seragam

### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202604090457`
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
- **Image:** `crunchalpha-v3:production-202604090601`
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
- **Image:** `crunchalpha-v3:production-202604100356`
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
- **Image:** `crunchalpha-v3:production-202604100432`
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
