
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
