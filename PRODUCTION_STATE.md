
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
