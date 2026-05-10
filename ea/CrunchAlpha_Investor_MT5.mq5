//+------------------------------------------------------------------+
//| CrunchAlpha Investor EA v3.4 MT5                                 |
//| AUM-Based: Copy Signal (Analyst) + Copy Trader                   |
//| Fix v3.4: ExtractBool pakai StringFind pattern "key":true        |
//|           lebih reliable dari string compare                     |
//+------------------------------------------------------------------+
#property copyright "CrunchAlpha"
#property version   "3.8"
#property strict

#include <Trade\Trade.mqh>


input string EAKey      = "";
string BackendURL = "https://crunchalpha.com";

CTrade   trade;
string   baseURL;
bool     copySignalEnabled = false;
bool     copyTraderEnabled = false;
double   signalMaxLot      = 0.10;
double   maxDailyLossPct   = 5.0;
int      maxOpenTrades     = 10;
datetime lastEquityPush    = 0;
datetime lastTradeSync     = 0;
int      tradeSyncInterval = 60;
datetime lastSettingsLoad  = 0;
int      equityInterval    = 30;
int      settingsInterval  = 60;

//+------------------------------------------------------------------+
int OnInit()
{
    if(EAKey == "") { Alert("[CA] ERROR: EA Key kosong! Generate dari Dashboard > Copy Settings > EA Key"); return INIT_PARAMETERS_INCORRECT; }
    baseURL = BackendURL;
    trade.SetDeviationInPoints(30);
    trade.SetExpertMagicNumber(20260307);
    EventSetTimer(2);
    Print("[CA v3.8] CrunchAlpha Investor EA MT5 v3.4 started");
    Print("[CA] Backend: ", BackendURL);
    LoadSettings();
    PushEquity();
    return INIT_SUCCEEDED;
}

void OnDeinit(const int reason) { EventKillTimer(); Print("[CA] EA stopped. Reason: ", reason); }
void OnTick() {}

//+------------------------------------------------------------------+
void OnTimer()
{
    datetime now = TimeCurrent();
    if(now - lastEquityPush > equityInterval)    { PushEquity();   lastEquityPush   = now; }
    if(now - lastSettingsLoad > settingsInterval){ LoadSettings(); lastSettingsLoad = now; }
    if(IsDailyLossBreached()) {
        static datetime lastWarn = 0;
        if(now - lastWarn > 300) { Print("[CA] STOP: Daily loss limit reached — copying paused"); lastWarn = now; }
        return;
    }
    if(copySignalEnabled) PollSignals();
    if(copyTraderEnabled) PollCopyTrades();
    SyncTrades();
}

//+------------------------------------------------------------------+
// PUSH EQUITY — 7 fields
//+------------------------------------------------------------------+
void PushEquity()
{
    double equity      = AccountInfoDouble(ACCOUNT_EQUITY);
    double balance     = AccountInfoDouble(ACCOUNT_BALANCE);
    double margin      = AccountInfoDouble(ACCOUNT_MARGIN);
    double freeMargin  = AccountInfoDouble(ACCOUNT_MARGIN_FREE);
    double floating    = equity - balance;
    double openLots    = 0;
    int    openCount   = PositionsTotal();
    for(int i = 0; i < openCount; i++) {
        ulong t = PositionGetTicket(i);
        if(t > 0) { PositionSelectByTicket(t); openLots += PositionGetDouble(POSITION_VOLUME); }
    }
    string body = "{\"equity\":"          + DoubleToString(equity, 2) +
                  ",\"balance\":"         + DoubleToString(balance, 2) +
                  ",\"margin\":"          + DoubleToString(margin, 2) +
                  ",\"free_margin\":"     + DoubleToString(freeMargin, 2) +
                  ",\"floating_profit\":" + DoubleToString(floating, 2) +
                  ",\"open_lots\":"       + DoubleToString(openLots, 2) +
                  ",\"open_positions\":"  + IntegerToString(openCount) + "}";
    int res = HTTPPost("/api/ea/investor/push-equity", body);
    if(res == 200) Print("[CA] Equity pushed $", DoubleToString(equity, 2), " floating:", DoubleToString(floating, 2));
    else           Print("[CA] Equity push failed HTTP:", res);
}

//+------------------------------------------------------------------+
// LOAD SETTINGS
//+------------------------------------------------------------------+
void LoadSettings()
{
    string j = HTTPGet("/api/ea/investor/settings");
    if(j == "") return;

    // ExtractBool v3.4: cari pattern "key":true langsung di raw JSON
    // Tidak perlu extract nested object
    copySignalEnabled = ExtractBool(j, "copySignalEnabled");
    copyTraderEnabled = ExtractBool(j, "copyTraderEnabled");
    signalMaxLot      = ExtractDbl(j,  "signalMaxLot");
    maxDailyLossPct   = ExtractDbl(j,  "maxDailyLossPct");
    maxOpenTrades     = (int)ExtractDbl(j, "maxOpenTrades");

    Print("[CA] Settings — Signal:", copySignalEnabled, " Trader:", copyTraderEnabled,
          " MaxLot:", signalMaxLot, " MaxDD:", maxDailyLossPct, "% MaxTrades:", maxOpenTrades);
}

//+------------------------------------------------------------------+
// POLL SIGNALS
//+------------------------------------------------------------------+
void PollSignals()
{
    string j = HTTPGet("/api/ea/investor/pending-signals");
    if(j == "") return;
    int cp = StringFind(j, "\"count\":");
    if(cp < 0) return;
    int count = (int)StringToInteger(StringSubstr(j, cp + 8, 5));
    if(count == 0) return;
    Print("[CA] ", count, " signal(s) pending");
    int pos = StringFind(j, "\"signals\":");
    while(pos >= 0) {
        int idPos = StringFind(j, "\"id\":", pos + 1);
        if(idPos < 0) break;
        int idEnd = StringFind(j, ",", idPos + 5);
        if(idEnd < 0) break;
        long sigID     = StringToInteger(StringSubstr(j, idPos + 5, idEnd - idPos - 5));
        string pair    = ExtractStrFrom(j, "\"pair\":",          idPos);
        string dir     = ExtractStrFrom(j, "\"direction\":",     idPos);
        double sl      = StringToDouble(ExtractStrFrom(j, "\"sl\":",            idPos));
        double tp      = StringToDouble(ExtractStrFrom(j, "\"tp\":",            idPos));
        double calcLot = StringToDouble(ExtractStrFrom(j, "\"calculatedLot\":", idPos));
        string status  = ExtractStrFrom(j, "\"status\":",        idPos);
        string ordStat = ExtractStrFrom(j, "\"orderStatus\":",   idPos);
        int tkPos = StringFind(j, "\"ticket\":", idPos);
        int tkEnd = StringFind(j, ",", tkPos + 9);
        long ticket = 0;
        if(tkPos >= 0 && tkEnd >= 0) ticket = StringToInteger(StringSubstr(j, tkPos + 9, tkEnd - tkPos - 9));
        ProcessSignal(sigID, pair, dir, sl, tp, calcLot, status, ordStat, ticket);
        pos = idEnd;
        if(pos >= StringLen(j) - 5) break;
    }
}

void ProcessSignal(long sigID, string pair, string dir,
                   double sl, double tp, double calcLot,
                   string status, string ordStatus, long ticket)
{
    if((status == "CLOSED_TP" || status == "CLOSED_SL") && ordStatus == "OPENED") {
        if(ticket > 0 && PositionSelectByTicket(ticket)) {
            if(trade.PositionClose(ticket))
                SendSignalUpdate(sigID, ticket, status, 0, trade.ResultPrice(), 0);
            else
                Print("[CA] Signal close failed ticket:", ticket, " err:", trade.ResultRetcode());
        }
        return;
    }
    if(ordStatus == "OPENED" || ordStatus == "CLOSED_TP" ||
       ordStatus == "CLOSED_SL" || ordStatus == "CLOSED_MANUAL") return;
    if(status != "RUNNING") return;
    if(PositionsTotal() >= maxOpenTrades) { Print("[CA] Signal skip — max trades ", maxOpenTrades); return; }

    string sym = NormalizePair(pair);
    if(sym == "") { Print("[CA] Symbol not found: ", pair); return; }

    double lot = (calcLot > 0) ? MathMin(calcLot, signalMaxLot) : 0.01;
    lot = NormalizeLot(sym, lot);
    if(lot < 0.01) { Print("[CA] Signal rejected — lot too small: ", lot); SendSignalUpdate(sigID, 0, "REJECTED", 0, 0, lot); return; }

    ENUM_ORDER_TYPE otype = (dir == "BUY") ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;
    double price = (dir == "BUY") ? SymbolInfoDouble(sym, SYMBOL_ASK) : SymbolInfoDouble(sym, SYMBOL_BID);

    double margin = 0;
    if(OrderCalcMargin(otype, sym, lot, price, margin)) {
        if(AccountInfoDouble(ACCOUNT_MARGIN_FREE) < margin * 1.2) {
            Print("[CA] Signal rejected — insufficient margin. Need:$", DoubleToString(margin * 1.2, 2));
            SendSignalUpdate(sigID, 0, "REJECTED", 0, 0, lot);
            return;
        }
    }

    string comment = "CA-SIG:" + IntegerToString(sigID);
    Print("[CA] Signal ", dir, " ", sym, " lot:", lot, " sl:", sl, " tp:", tp);
    bool ok = (dir == "BUY") ? trade.Buy(lot, sym, 0, sl, tp, comment)
                              : trade.Sell(lot, sym, 0, sl, tp, comment);
    if(ok) {
        Print("[CA] Signal opened ticket:", trade.ResultOrder(), " price:", trade.ResultPrice());
        SendSignalUpdate(sigID, trade.ResultOrder(), "OPENED", trade.ResultPrice(), 0, lot);
    } else {
        Print("[CA] Signal failed: ", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
        SendSignalUpdate(sigID, 0, "REJECTED", 0, 0, lot);
    }
}

//+------------------------------------------------------------------+
// POLL COPY TRADES
//+------------------------------------------------------------------+
void PollCopyTrades()
{
    string j = HTTPGet("/api/ea/investor/pending-copy-trades");
    if(j == "") return;
    int cp = StringFind(j, "\"count\":");
    if(cp < 0) return;
    int count = (int)StringToInteger(StringSubstr(j, cp + 8, 5));
    if(count == 0) return;
    Print("[CA] ", count, " copy trade event(s) pending");
    int pos = StringFind(j, "\"events\":");
    while(pos >= 0) {
        int idPos   = StringFind(j, "\"id\":", pos + 1);
        if(idPos < 0) break;
        int idStart = StringFind(j, "\"", idPos + 5) + 1;
        int idEnd   = StringFind(j, "\"", idStart);
        if(idStart < 0 || idEnd < 0) break;
        string eventID = StringSubstr(j, idStart, idEnd - idStart);
        string action  = ExtractStrFrom(j, "\"action\":",          idPos);
        string symbol  = ExtractStrFrom(j, "\"symbol\":",          idPos);
        int    dir     = (int)StringToDouble(ExtractStrFrom(j, "\"direction\":",     idPos));
        double calcLot = StringToDouble(ExtractStrFrom(j, "\"calculatedLot\":",     idPos));
        double sl      = StringToDouble(ExtractStrFrom(j, "\"sl\":",                idPos));
        double tp      = StringToDouble(ExtractStrFrom(j, "\"tp\":",                idPos));
        long   provTkt = (long)StringToDouble(ExtractStrFrom(j, "\"providerTicket\":", idPos));
        double aumUsed      = StringToDouble(ExtractStrFrom(j, "\"aumUsed\":",           idPos));
        double maxSlipPips  = StringToDouble(ExtractStrFrom(j, "\"maxSlippagePips\":",  idPos));
        double masterOpen   = StringToDouble(ExtractStrFrom(j, "\"openPrice\":",         idPos));
        ProcessCopyTrade(eventID, action, symbol, dir, calcLot, sl, tp, provTkt, aumUsed, maxSlipPips, masterOpen);
        pos = idEnd;
        if(pos >= StringLen(j) - 5) break;
    }
}

void ProcessCopyTrade(string eventID, string action, string symbol,
                      int dir, double calcLot, double sl, double tp,
                      long provTicket, double aumUsed,
                      double maxSlipPips=0, double masterOpenPrice=0)
{
    if(action == "CLOSE") {
        string sc = "CA-CT:" + IntegerToString(provTicket);
        for(int i = PositionsTotal() - 1; i >= 0; i--) {
            ulong tkt = PositionGetTicket(i);
            if(PositionSelectByTicket(tkt) && StringFind(PositionGetString(POSITION_COMMENT), sc) >= 0) {
                double savedLot    = PositionGetDouble(POSITION_VOLUME);
                double savedProfit = PositionGetDouble(POSITION_PROFIT);
                if(trade.PositionClose(tkt)) {
                    Print("[CA] CopyTrade closed ticket:", tkt);
                    SendCopyTradeUpdate(eventID, "EXECUTED", "", tkt, savedLot, trade.ResultPrice(), savedProfit);
                }
                return;
            }
        }
        SendCopyTradeUpdate(eventID, "EXECUTED", "Already closed", 0, 0, 0);
        return;
    }
    if(action != "OPEN") return;
    if(PositionsTotal() >= maxOpenTrades) { SendCopyTradeUpdate(eventID, "REJECTED", "Max open trades (" + IntegerToString(maxOpenTrades) + ")", 0, 0, 0); return; }

    string sym = NormalizePair(symbol);
    if(sym == "") { SendCopyTradeUpdate(eventID, "REJECTED", "Symbol not found: " + symbol, 0, 0, 0); return; }

    double lot = NormalizeLot(sym, calcLot);
    if(lot < 0.01) { SendCopyTradeUpdate(eventID, "REJECTED", "Lot below minimum: " + DoubleToString(lot, 4), 0, 0, 0); return; }

    ENUM_ORDER_TYPE otype = (dir == 0) ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;
    double price = (dir == 0) ? SymbolInfoDouble(sym, SYMBOL_ASK) : SymbolInfoDouble(sym, SYMBOL_BID);
    double margin = 0;
    if(OrderCalcMargin(otype, sym, lot, price, margin)) {
        if(AccountInfoDouble(ACCOUNT_MARGIN_FREE) < margin * 1.2) {
            string reason = "Insufficient margin. Need:$" + DoubleToString(margin * 1.2, 2) +
                            " Free:$" + DoubleToString(AccountInfoDouble(ACCOUNT_MARGIN_FREE), 2);
            SendCopyTradeUpdate(eventID, "REJECTED", reason, 0, 0, 0);
            return;
        }
    }

    // Slippage check — bandingkan harga master open vs harga sekarang
    if(maxSlipPips > 0 && masterOpenPrice > 0) {
        double currentPrice = (dir == 0) ? SymbolInfoDouble(sym, SYMBOL_ASK) : SymbolInfoDouble(sym, SYMBOL_BID);
        double point = SymbolInfoDouble(sym, SYMBOL_POINT);
        double digits = (double)SymbolInfoInteger(sym, SYMBOL_DIGITS);
        double pipSize = (digits == 3 || digits == 5) ? point * 10 : point;
        double slippagePips = MathAbs(currentPrice - masterOpenPrice) / pipSize;
        if(slippagePips > maxSlipPips) {
            string reason = "Slippage too high: " + DoubleToString(slippagePips, 1) +
                            " pips (max " + DoubleToString(maxSlipPips, 1) + ")";
            Print("[CA] CopyTrade SKIPPED — ", reason);
            SendCopyTradeUpdate(eventID, "REJECTED", reason, 0, 0, 0);
            return;
        }
        Print("[CA] Slippage OK: ", DoubleToString(slippagePips, 1), " pips (max ", DoubleToString(maxSlipPips, 1), ")");
    }
    string comment = "CA-CT:" + IntegerToString(provTicket);
    Print("[CA] CopyTrade ", (dir == 0 ? "BUY" : "SELL"), " ", sym,
          " lot:", lot, " sl:", sl, " tp:", tp, " AUM:$", DoubleToString(aumUsed, 2));
    bool ok = (dir == 0) ? trade.Buy(lot, sym, 0, sl, tp, comment)
                         : trade.Sell(lot, sym, 0, sl, tp, comment);
    if(ok) {
        Print("[CA] CopyTrade opened ticket:", trade.ResultOrder(), " price:", trade.ResultPrice());
        SendCopyTradeUpdate(eventID, "EXECUTED", "", trade.ResultOrder(), lot, trade.ResultPrice());
    } else {
        string reason = "Order failed: " + IntegerToString(trade.ResultRetcode()) + " " + trade.ResultRetcodeDescription();
        Print("[CA] CopyTrade failed: ", reason);
        SendCopyTradeUpdate(eventID, "REJECTED", reason, 0, 0, 0);
    }
}

//+------------------------------------------------------------------+
// SEND UPDATES
//+------------------------------------------------------------------+
void SendSignalUpdate(long sigID, long ticket, string status, double openP, double closeP, double lot)
{
    string body = "{\"signalId\":"   + IntegerToString(sigID) +
                  ",\"ticket\":"     + IntegerToString(ticket) +
                  ",\"status\":\"" + status + "\"" +
                  ",\"openPrice\":"  + DoubleToString(openP, 5) +
                  ",\"closePrice\":" + DoubleToString(closeP, 5) +
                  ",\"lotSize\":"    + DoubleToString(lot, 4) + "}";
    HTTPPost("/api/ea/investor/order-update", body);
}

void SendCopyTradeUpdate(string eventID, string status, string reason, long ticket, double lot, double price, double profit=0)
{
    string body = "{\"eventId\":\"" + eventID + "\"" +
                  ",\"status\":\"" + status + "\"" +
                  ",\"rejectionReason\":\"" + reason + "\"" +
                  ",\"followerTicket\":" + IntegerToString(ticket) +
                  ",\"executedLot\":"   + DoubleToString(lot, 4) +
                  ",\"executedPrice\":" + DoubleToString(price, 5) +
                  ",\"profit\":" + DoubleToString(profit, 2) + "}";
    HTTPPost("/api/ea/investor/copy-trade-update", body);
}

//+------------------------------------------------------------------+
// DAILY LOSS GUARD
//+------------------------------------------------------------------+
bool IsDailyLossBreached()
{
    if(maxDailyLossPct <= 0) return false;
    double balance = AccountInfoDouble(ACCOUNT_BALANCE);
    double equity  = AccountInfoDouble(ACCOUNT_EQUITY);
    return (balance - equity) >= (balance * maxDailyLossPct / 100.0);
}

//+------------------------------------------------------------------+
// HELPERS
//+------------------------------------------------------------------+
double NormalizeLot(string sym, double lot)
{
    double mn = SymbolInfoDouble(sym, SYMBOL_VOLUME_MIN);
    double mx = SymbolInfoDouble(sym, SYMBOL_VOLUME_MAX);
    double st = SymbolInfoDouble(sym, SYMBOL_VOLUME_STEP);
    lot = MathFloor(lot / st) * st;
    return MathMax(mn, MathMin(lot, mx));
}

string NormalizePair(string pair)
{
    if(SymbolInfoInteger(pair, SYMBOL_DIGITS) > 0) return pair;
    string sfx[] = {".raw", ".pro", ".ecn", ".std", ".stp", "+", ".m", ".r", "c", ".micro"};
    for(int i = 0; i < 10; i++) {
        string t = pair + sfx[i];
        if(SymbolInfoInteger(t, SYMBOL_DIGITS) > 0) return t;
    }
    return "";
}

string HTTPGet(string endpoint)
{
    string headers = "X-EA-Key: " + EAKey + "\r\n";
    char post[], result[];
    string rh;
    int res = WebRequest("GET", baseURL + endpoint, headers, 5000, post, result, rh);
    if(res != 200) { Print("[CA] GET ", endpoint, " failed HTTP:", res); return ""; }
    return CharArrayToString(result);
}

int HTTPPost(string endpoint, string body)
{
    string headers = "X-EA-Key: " + EAKey + "\r\nContent-Type: application/json\r\n";
    char post[], result[];
    string rh;
    int len = StringToCharArray(body, post, 0, WHOLE_ARRAY, CP_UTF8) - 1;
    ArrayResize(post, len);
    int res = WebRequest("POST", baseURL + endpoint, headers, 5000, post, result, rh);
    if(res == -1) {
        int err = GetLastError();
        Print("[CA] WebRequest ERROR:", err, " endpoint:", endpoint);
        if(err == 4014) Print("[CA] Add ", baseURL, " to Tools > Options > Expert Advisors > Allow WebRequest");
    }
    return res;
}

// ExtractStrFrom — key sudah include ":" misal "\"pair\":"
string ExtractStrFrom(string j, string key, int from)
{
    int p = StringFind(j, key, from);
    if(p < 0) return "";
    int s = p + StringLen(key);
    while(s < StringLen(j) && StringGetCharacter(j, s) == ' ') s++;
    bool quoted = (StringGetCharacter(j, s) == '"');
    if(quoted) s++;
    int e = s;
    while(e < StringLen(j)) {
        ushort c = StringGetCharacter(j, e);
        if(quoted  && c == '"') break;
        if(!quoted && (c == ',' || c == '}' || c == ']')) break;
        e++;
    }
    return StringSubstr(j, s, e - s);
}

// ExtractDbl — key tanpa colon, cari "key": lalu ambil nilai
double ExtractDbl(string j, string key)
{
    string pattern = "\"" + key + "\":";
    int p = StringFind(j, pattern);
    if(p < 0) return 0;
    int s = p + StringLen(pattern);
    while(s < StringLen(j) && StringGetCharacter(j, s) == ' ') s++;
    int e = s;
    while(e < StringLen(j)) {
        ushort c = StringGetCharacter(j, e);
        if(c == ',' || c == '}' || c == ']') break;
        e++;
    }
    return StringToDouble(StringSubstr(j, s, e - s));
}

// ExtractBool v3.4 — cari pattern "key":true langsung
bool ExtractBool(string j, string key)
{
    string patternTrue  = "\"" + key + "\":true";
    string patternFalse = "\"" + key + "\":false";
    if(StringFind(j, patternTrue)  >= 0) return true;
    if(StringFind(j, patternFalse) >= 0) return false;
    return false;
}
//+------------------------------------------------------------------+
// SYNC TRADES — kirim history trades ke backend setiap 5 menit
//+------------------------------------------------------------------+
void SyncTrades()
{
    datetime now = TimeCurrent();
    if(now - lastTradeSync < tradeSyncInterval) return;
    lastTradeSync = now;

    if(!HistorySelect(0, TimeCurrent())) return;
    int total = HistoryDealsTotal();
    int start = MathMax(0, total - 200);
    string trades = "";
    int count = 0;

    for(int i = start; i < total; i++)
    {
        ulong deal = HistoryDealGetTicket(i);
        if(deal == 0) continue;
        if(HistoryDealGetInteger(deal, DEAL_ENTRY) != DEAL_ENTRY_OUT) continue;
        string comment = HistoryDealGetString(deal, DEAL_COMMENT);
        if(StringFind(comment, "CA-CT:") < 0) continue;

        long   posID      = HistoryDealGetInteger(deal, DEAL_POSITION_ID);
        string sym        = HistoryDealGetString(deal, DEAL_SYMBOL);
        double closePrice = HistoryDealGetDouble(deal, DEAL_PRICE);
        double lots       = HistoryDealGetDouble(deal, DEAL_VOLUME);
        double profit     = HistoryDealGetDouble(deal, DEAL_PROFIT);
        double swap       = HistoryDealGetDouble(deal, DEAL_SWAP);
        double comm       = HistoryDealGetDouble(deal, DEAL_COMMISSION);
        long   closeTime  = (long)HistoryDealGetInteger(deal, DEAL_TIME);
        int    dealType   = (int)HistoryDealGetInteger(deal, DEAL_TYPE);
        string typeStr    = (dealType == DEAL_TYPE_SELL) ? "buy" : "sell";

        double openPrice = closePrice;
        long   openTime  = closeTime;
        for(int j = 0; j < total; j++) {
            ulong d = HistoryDealGetTicket(j);
            if(d == 0) continue;
            if(HistoryDealGetInteger(d, DEAL_POSITION_ID) == posID &&
               HistoryDealGetInteger(d, DEAL_ENTRY) == DEAL_ENTRY_IN) {
                openPrice = HistoryDealGetDouble(d, DEAL_PRICE);
                openTime  = (long)HistoryDealGetInteger(d, DEAL_TIME);
                break;
            }
        }

        if(trades != "") trades += ",";
        trades += "{\"ticket\":"     + IntegerToString(posID) +
                  ",\"symbol\":\""   + sym + "\"" +
                  ",\"type\":\""     + typeStr + "\"" +
                  ",\"lots\":"       + DoubleToString(lots, 2) +
                  ",\"openPrice\":"  + DoubleToString(openPrice, 5) +
                  ",\"closePrice\":" + DoubleToString(closePrice, 5) +
                  ",\"openTime\":"   + IntegerToString(openTime) +
                  ",\"closeTime\":"  + IntegerToString(closeTime) +
                  ",\"profit\":"     + DoubleToString(profit, 2) +
                  ",\"swap\":"       + DoubleToString(swap, 2) +
                  ",\"commission\":" + DoubleToString(comm, 2) +
                  ",\"status\":\"closed\"" +
                  ",\"comment\":\"" + comment + "\"}";
        count++;
    }

    if(count == 0) return;
    string body = "{\"trades\":[" + trades + "]}";
    int res = HTTPPost("/api/ea/investor/sync-trades", body);
    if(res == 200) Print("[CA] SyncTrades: ", count, " trades synced");
    else           Print("[CA] SyncTrades failed HTTP:", res);
}
