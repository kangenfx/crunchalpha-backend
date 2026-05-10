# CrunchAlpha Production State
# Last Updated: 2026-05-10

## 🐳 CURRENT PRODUCTION
### Backend:
- **Container:** `crunchalpha-backend`
- **Image:** `crunchalpha-v3:production-202605100900`
- **Port:** 8090 (internal), via nginx https
- **Network:** crunchalpha-net

### Frontend:
- **Container:** `crunchalpha-frontend-v3`
- **Image:** `crunchalpha-frontend-v3:prod-202605101459`
- **Port:** 5176 (internal), via nginx https
- **URL:** https://crunchalpha.com

## ✅ EA STATUS
- EA Publisher MT5 v3.1 (dengan ?apikey= di URL): HTTPS crunchalpha.com ✅
- EA Investor MT5 v3.9.1: HTTPS crunchalpha.com ✅
- EA Investor MT4 v2.1: HTTPS crunchalpha.com ✅

## 🔑 EA KEY RULES
- EA Publisher: 1 key per USER (bukan per account), max 10 keys
- EA Investor: 1 key per ACCOUNT investor, max 10 keys
- Key baru dari dashboard BERFUNGSI dengan EA Publisher v3.1 (ada ?apikey= di URL)
- Key lama (crunch_0a416...) masih aktif dan berfungsi
- nginx: X-API-Key dan X-EA-Key di-pass ke backend
- Backend: support X-API-Key header, X-EA-Key header, dan ?apikey= query param

## ✅ SINGLE SOURCE OF TRUTH
- Semua metric dari `alpha_ranks` — zero on-the-fly
- net_pnl global = closed + floating
- risk_level dari alpha_ranks, recalculate setiap EA push
- pillar reason dari alpha_ranks.pillars JSON

## 🔑 RISK LEVEL FORMULA (LOCKED)
- EXTREME: Any critical flag OR AlphaScore < 30
- HIGH: 3+ flags OR AlphaScore 30-50
- MEDIUM: 2 flags OR AlphaScore 50-70
- LOW: 0-1 flags + AlphaScore >= 70
- VERIFIED_SAFE: No flags + AlphaScore >= 85

## 🔑 COPY TRADING RULES (LOCKED)
- CONSERVATIVE: Max 0.5% AUM/trade, Max DD 5%
- BALANCED: Max 1.5% AUM/trade, Max DD 10%
- AGGRESSIVE: Max 3.0% AUM/trade, Max DD 20%
- Lot formula: traderLot × (AUM/traderEquity) × layer3 → capped by risk level
- propLot < 0.01 → SKIP (tidak copy)
- MT5 investor copy: TESTED ✅
- MT4 investor copy: PENDING TEST (market tutup)

## 🔑 PROJECT STRUCTURE
- **Backend source:** `/root/crunchalpha-v3.OLD` (Go)
- **Frontend source:** `/var/www/crunchalpha-frontend-v3-SRC` (React/JSX)
- **Network:** crunchalpha-net (bukan root_crunchalpha-net)
- **DB:** crunchalpha-postgres container

## 🔑 DEPLOY COMMAND BACKEND
```bash
docker rm -f crunchalpha-backend && \
docker run -d --name crunchalpha-backend --network crunchalpha-net -p 8090:8090 \
  --env-file /root/.env-crunchalpha \
  --restart unless-stopped \
  crunchalpha-v3:production-YYYYMMDDHHMM
```

## 🔑 DEPLOY COMMAND FRONTEND
```bash
cd /var/www/crunchalpha-frontend-v3-SRC && npm run build && \
docker build -t crunchalpha-frontend-v3:prod-YYYYMMDDHHMM . && \
docker rm -f crunchalpha-frontend-v3 && \
docker run -d --name crunchalpha-frontend-v3 --network crunchalpha-net -p 5176:80 \
  --restart unless-stopped \
  crunchalpha-frontend-v3:prod-YYYYMMDDHHMM
```

## ⚠️ PENDING
1. MT4 investor copy trade — test saat market buka (Senin)
2. Open positions display di trader & investor dashboard
3. Delete button EA key di UI trader dashboard
4. Remove debug log [APIKeyAuth] dari middleware setelah verified

## 🔑 SERVER INFO
- **Provider:** Contabo Cloud VPS
- **IP:** 62.146.239.174
- **Specs:** 4 vCPU, 8GB RAM, 150GB SSD, Singapore
- **Wine/MT5:** running untuk demo testing (akun real blocked by broker)
