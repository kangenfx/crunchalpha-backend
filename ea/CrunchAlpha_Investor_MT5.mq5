//+------------------------------------------------------------------+
//| CrunchAlpha Investor EA v2.1                                     |
//| AUM-Based: Copy Signal (Analyst) + Copy Trader in one EA        |
//| Supports MT5 — configure via web dashboard                       |
//| Lot calculated by backend — EA only executes                     |
//+------------------------------------------------------------------+
#property copyright "CrunchAlpha"
#property version   "2.10"
#property strict

#include <Trade\Trade.mqh>

//── Inputs ──────────────────────────────────────────────────────────
input string EAKey      = "";                          // EA Key (from Dashboard > Copy Settings)
input string BackendURL = "https://crunchalpha.com";   // Backend URL

//── Globals ─────────────────────────────────────────────────────────
CTrade   trade;
string   baseURL;
bool     copySignalEnabled = false;
bool     copyTraderEnabled = false;
double   signalMaxLot      = 0.10;
double   maxDailyLossPct   = 5.0;
int      maxOpenTrades     = 10;
datetime lastEquityPush    = 0;
datetime lastSettingsLoad  = 0;
datetime lastSignalPoll    = 0;
datetime lastCopyPoll      = 0;
int      equityInterval    = 30;
int      settingsInterval  = 300;
int      signalInterval    = 10;
int      copyInterval      = 10;

//+------------------------------------------------------------------+
int OnInit()
{
   if(EAKey == "")
   {
      Alert("[CA] ERROR: EA Key is empty! Generate from Dashboard > Copy Settings > EA Key");
      return INIT_PARAMETERS_INCORRECT;
   }
   baseURL = BackendURL;
   trade.SetDeviationInPoints(30);
   trade.SetExpertMagicNumber(20260307);
   EventSetTimer(5);
   Print("[CA] CrunchAlpha Investor EA v2.1 started");
   Print("[CA] AUM-Based proportional lot active");
   LoadSettings();
   PushEquity();
   return INIT_SUCCEEDED;
}

void OnDeinit(const int reason)
{
   EventKillTimer();
   Print("[CA] EA stopped. Reason: ", reason);
}

void OnTick() {}

//+------------------------------------------------------------------+
void OnTimer()
{
   datetime now = TimeCurrent();

   if(now - lastEquityPush >= equityInterval)
   {
      PushEquity();
      lastEquityPush = now;
   }
   if(now - lastSettingsLoad >= settingsInterval)
   {
      LoadSettings();
      lastSettingsLoad = now;
   }
   if(IsDailyLossBreached())
   {
      static datetime lastWarn = 0;
      if(now - lastWarn > 300)
      {
         Print("[CA] STOP: Daily loss limit reached — copying paused");
         lastWarn = now;
      }
      return;
   }
   if(copySignalEnabled && now - lastSignalPoll >= signalInterval)
   {
      PollSignals();
      lastSignalPoll = now;
   }
   if(copyTraderEnabled && now - lastCopyPoll >= copyInterval)
   {
      PollCopyTrades();
      lastCopyPoll = now;
   }
}

//+------------------------------------------------------------------+
// PUSH EQUITY
//+------------------------------------------------------------------+
void PushEquity()
{
   double equity  = AccountInfoDouble(ACCOUNT_EQUITY);
   double balance = AccountInfoDouble(ACCOUNT_BALANCE);
   string body    = "{\"equity\":" + DoubleToString(equity,2) +
                    ",\"balance\":" + DoubleToString(balance,2) + "}";
   string headers = "X-EA-Key: " + EAKey + "\r\nContent-Type: application/json\r\n";
   char   post[], result[];
   string rh;
   StringToCharArray(body, post, 0, StringLen(body));
   int res = WebRequest("POST", baseURL + "/api/ea/investor/push-equity", headers, 5000, post, result, rh);
   if(res == 200) Print("[CA] Equity pushed $", DoubleToString(equity,2));
   else           Print("[CA] Equity push failed HTTP:", res);
}

//+------------------------------------------------------------------+
// LOAD SETTINGS
//+------------------------------------------------------------------+
void LoadSettings()
{
   string headers = "X-EA-Key: " + EAKey + "\r\n";
   char   post[], result[];
   string rh;
   int res = WebRequest("GET", baseURL + "/api/ea/investor/settings", headers, 5000, post, result, rh);
   if(res != 200) { Print("[CA] Settings failed HTTP:", res); return; }
   string j = CharArrayToString(result);
   copySignalEnabled = ExtractBool(j, "copySignalEnabled");
   copyTraderEnabled = ExtractBool(j, "copyTraderEnabled");
   signalMaxLot      = ExtractDbl(j,  "signalMaxLot");
   maxDailyLossPct   = ExtractDbl(j,  "maxDailyLossPct");
   maxOpenTrades     = (int)ExtractDbl(j, "maxOpenTrades");
   Print("[CA] Settings loaded — Signal:", copySignalEnabled, " Trader:", copyTraderEnabled,
         " MaxLot:", signalMaxLot, " MaxDD:", maxDailyLossPct, "%");
}

//+------------------------------------------------------------------+
// POLL SIGNALS (Copy Analyst)
//+------------------------------------------------------------------+
void PollSignals()
{
   string headers = "X-EA-Key: " + EAKey + "\r\n";
   char   post[], result[];
   string rh;
   int res = WebRequest("GET", baseURL + "/api/ea/investor/pending-signals", headers, 5000, post, result, rh);
   if(res != 200) { Print("[CA] Signal poll failed HTTP:", res); return; }
   string j = CharArrayToString(result);

   int cp = StringFind(j, "\"count\":");
   if(cp < 0) return;
   int count = (int)StringToInteger(StringSubstr(j, cp + 8, 5));
   if(count == 0) return;
   Print("[CA] ", count, " signal(s) pending");

   int pos = 0;
   while(true)
   {
      int idPos = StringFind(j, "\"id\":", pos);
      if(idPos < 0) break;
      int idEnd = StringFind(j, ",", idPos + 5);
      if(idEnd < 0) break;
      long sigID = StringToInteger(StringSubstr(j, idPos + 5, idEnd - idPos - 5));

      string pair      = ExtractStrFrom(j, "\"pair\":",           idPos);
      string dir       = ExtractStrFrom(j, "\"direction\":",      idPos);
      double sl        = StringToDouble(ExtractStrFrom(j, "\"sl\":",             idPos));
      double tp        = StringToDouble(ExtractStrFrom(j, "\"tp\":",             idPos));
      double calcLot   = StringToDouble(ExtractStrFrom(j, "\"calculatedLot\":",  idPos));
      string status    = ExtractStrFrom(j, "\"status\":",         idPos);
      string ordStatus = ExtractStrFrom(j, "\"orderStatus\":",    idPos);

      int  tkPos  = StringFind(j, "\"ticket\":", idPos);
      int  tkEnd  = StringFind(j, ",", tkPos + 9);
      long ticket = 0;
      if(tkPos >= 0 && tkEnd >= 0)
         ticket = StringToInteger(StringSubstr(j, tkPos + 9, tkEnd - tkPos - 9));

      ProcessSignal(sigID, pair, dir, sl, tp, calcLot, status, ordStatus, ticket);

      pos = idEnd;
      if(pos >= StringLen(j) - 5) break;
   }
}

void ProcessSignal(long sigID, string pair, string dir,
                   double sl, double tp, double calcLot,
                   string status, string ordStatus, long ticket)
{
   // Close signal
   if((status == "CLOSED_TP" || status == "CLOSED_SL") && ordStatus == "OPENED")
   {
      if(ticket > 0 && PositionSelectByTicket(ticket))
      {
         if(trade.PositionClose(ticket))
            SendSignalUpdate(sigID, ticket, status, 0, trade.ResultPrice(), 0);
      }
      return;
   }
   if(ordStatus == "OPENED" || ordStatus == "CLOSED_TP" ||
      ordStatus == "CLOSED_SL" || ordStatus == "CLOSED_MANUAL") return;
   if(status != "RUNNING") return;
   if(PositionsTotal() >= maxOpenTrades) { Print("[CA] Signal skip — max trades"); return; }

   string sym = NormalizePair(pair);
   if(sym == "") { Print("[CA] Symbol not found: ", pair); return; }

   double lot = (calcLot > 0) ? MathMin(calcLot, signalMaxLot) : 0.01;
   lot = NormalizeLot(sym, lot);

   double margin = 0;
   ENUM_ORDER_TYPE otype = (dir == "BUY") ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;
   double price = (dir == "BUY") ? SymbolInfoDouble(sym, SYMBOL_ASK) : SymbolInfoDouble(sym, SYMBOL_BID);
   if(OrderCalcMargin(otype, sym, lot, price, margin))
   {
      if(AccountInfoDouble(ACCOUNT_MARGIN_FREE) < margin * 1.2)
      {
         Print("[CA] Signal rejected — insufficient margin");
         SendSignalUpdate(sigID, 0, "REJECTED", 0, 0, lot);
         return;
      }
   }

   bool ok = (dir == "BUY") ? trade.Buy(lot,  sym, 0, sl, tp, "CA-SIG:" + IntegerToString(sigID))
                             : trade.Sell(lot, sym, 0, sl, tp, "CA-SIG:" + IntegerToString(sigID));
   if(ok)
   {
      Print("[CA] Signal opened ticket:", trade.ResultOrder(), " lot:", lot);
      SendSignalUpdate(sigID, trade.ResultOrder(), "OPENED", trade.ResultPrice(), 0, lot);
   }
   else
      Print("[CA] Signal failed:", trade.ResultRetcode(), " ", trade.ResultRetcodeDescription());
}

//+------------------------------------------------------------------+
// POLL COPY TRADES
//+------------------------------------------------------------------+
void PollCopyTrades()
{
   string headers = "X-EA-Key: " + EAKey + "\r\n";
   char   post[], result[];
   string rh;
   int res = WebRequest("GET", baseURL + "/api/ea/investor/pending-copy-trades", headers, 5000, post, result, rh);
   if(res != 200) { Print("[CA] CopyTrade poll failed HTTP:", res); return; }
   string j = CharArrayToString(result);

   int cp = StringFind(j, "\"count\":");
   if(cp < 0) return;
   int count = (int)StringToInteger(StringSubstr(j, cp + 8, 5));
   if(count == 0) return;
   Print("[CA] ", count, " copy trade event(s) pending");

   int pos = 0;
   while(true)
   {
      int idPos = StringFind(j, "\"id\":", pos);
      if(idPos < 0) break;
      int idStart = StringFind(j, "\"", idPos + 5) + 1;
      int idEnd   = StringFind(j, "\"", idStart);
      if(idStart < 0 || idEnd < 0) break;
      string eventID = StringSubstr(j, idStart, idEnd - idStart);

      string action  = ExtractStrFrom(j, "\"action\":",          idPos);
      string symbol  = ExtractStrFrom(j, "\"symbol\":",          idPos);
      int    dir     = (int)StringToDouble(ExtractStrFrom(j, "\"direction\":",   idPos));
      double calcLot = StringToDouble(ExtractStrFrom(j, "\"calculatedLot\":",   idPos));
      double sl      = StringToDouble(ExtractStrFrom(j, "\"sl\":",              idPos));
      double tp      = StringToDouble(ExtractStrFrom(j, "\"tp\":",              idPos));
      long   provTkt = (long)StringToDouble(ExtractStrFrom(j, "\"providerTicket\":", idPos));
      double aumUsed = StringToDouble(ExtractStrFrom(j, "\"aumUsed\":",         idPos));

      ProcessCopyTrade(eventID, action, symbol, dir, calcLot, sl, tp, provTkt, aumUsed);

      pos = idEnd;
      if(pos >= StringLen(j) - 5) break;
   }
}

void ProcessCopyTrade(string eventID, string action, string symbol,
                      int dir, double calcLot, double sl, double tp,
                      long provTicket, double aumUsed)
{
   if(action == "CLOSE")
   {
      string searchComment = "CA-CT:" + IntegerToString(provTicket);
      for(int i = PositionsTotal() - 1; i >= 0; i--)
      {
         ulong tkt = PositionGetTicket(i);
         if(PositionSelectByTicket(tkt))
         {
            if(StringFind(PositionGetString(POSITION_COMMENT), searchComment) >= 0)
            {
               if(trade.PositionClose(tkt))
                  SendCopyTradeUpdate(eventID, "EXECUTED", "", tkt, calcLot, trade.ResultPrice());
               return;
            }
         }
      }
      SendCopyTradeUpdate(eventID, "EXECUTED", "Already closed", 0, 0, 0);
      return;
   }
   if(action != "OPEN") return;
   if(PositionsTotal() >= maxOpenTrades)
   {
      SendCopyTradeUpdate(eventID, "REJECTED", "Max open trades", 0, 0, 0);
      return;
   }
   string sym = NormalizePair(symbol);
   if(sym == "") { SendCopyTradeUpdate(eventID, "REJECTED", "Symbol not found: "+symbol, 0, 0, 0); return; }

   double lot = NormalizeLot(sym, calcLot);
   if(lot < SymbolInfoDouble(sym, SYMBOL_VOLUME_MIN))
   {
      SendCopyTradeUpdate(eventID, "REJECTED", "Lot below minimum: "+DoubleToString(lot,4), 0, 0, 0);
      return;
   }
   double margin = 0;
   ENUM_ORDER_TYPE otype = (dir == 0) ? ORDER_TYPE_BUY : ORDER_TYPE_SELL;
   double price = (dir == 0) ? SymbolInfoDouble(sym, SYMBOL_ASK) : SymbolInfoDouble(sym, SYMBOL_BID);
   if(OrderCalcMargin(otype, sym, lot, price, margin))
   {
      if(AccountInfoDouble(ACCOUNT_MARGIN_FREE) < margin * 1.2)
      {
         SendCopyTradeUpdate(eventID, "REJECTED",
            "Insufficient margin. Need:$"+DoubleToString(margin*1.2,2), 0, 0, 0);
         return;
      }
   }
   string comment = "CA-CT:" + IntegerToString(provTicket);
   bool ok = (dir == 0) ? trade.Buy(lot,  sym, 0, sl, tp, comment)
                        : trade.Sell(lot, sym, 0, sl, tp, comment);
   if(ok)
   {
      Print("[CA] CopyTrade opened ticket:", trade.ResultOrder(), " lot:", lot, " AUM:$", DoubleToString(aumUsed,2));
      SendCopyTradeUpdate(eventID, "EXECUTED", "", trade.ResultOrder(), lot, trade.ResultPrice());
   }
   else
   {
      string reason = "Order failed: "+IntegerToString(trade.ResultRetcode())+" "+trade.ResultRetcodeDescription();
      Print("[CA] CopyTrade failed: ", reason);
      SendCopyTradeUpdate(eventID, "REJECTED", reason, 0, 0, 0);
   }
}

//+------------------------------------------------------------------+
// SEND UPDATES
//+------------------------------------------------------------------+
void SendSignalUpdate(long sigID, long ticket, string status,
                      double openP, double closeP, double lot)
{
   string body = "{\"signalId\":" + IntegerToString(sigID) +
                 ",\"ticket\":"   + IntegerToString(ticket) +
                 ",\"status\":\"" + status + "\"" +
                 ",\"openPrice\":" + DoubleToString(openP,5) +
                 ",\"closePrice\":" + DoubleToString(closeP,5) +
                 ",\"lotSize\":"  + DoubleToString(lot,4) + "}";
   string headers = "X-EA-Key: " + EAKey + "\r\nContent-Type: application/json\r\n";
   char post[], result[]; string rh;
   StringToCharArray(body, post, 0, StringLen(body));
   WebRequest("POST", baseURL + "/api/ea/investor/order-update", headers, 5000, post, result, rh);
}

void SendCopyTradeUpdate(string eventID, string status, string reason,
                         long ticket, double lot, double price)
{
   string body = "{\"eventId\":\"" + eventID + "\"" +
                 ",\"status\":\"" + status + "\"" +
                 ",\"rejectionReason\":\"" + reason + "\"" +
                 ",\"followerTicket\":" + IntegerToString(ticket) +
                 ",\"executedLot\":"   + DoubleToString(lot,4) +
                 ",\"executedPrice\":" + DoubleToString(price,5) + "}";
   string headers = "X-EA-Key: " + EAKey + "\r\nContent-Type: application/json\r\n";
   char post[], result[]; string rh;
   StringToCharArray(body, post, 0, StringLen(body));
   WebRequest("POST", baseURL + "/api/ea/investor/copy-trade-update", headers, 5000, post, result, rh);
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
   string sfx[] = {".raw",".pro",".ecn",".std",".stp","+"};
   for(int i = 0; i < 6; i++)
   {
      string t = pair + sfx[i];
      if(SymbolInfoInteger(t, SYMBOL_DIGITS) > 0) return t;
   }
   return "";
}

string ExtractStrFrom(string j, string key, int from)
{
   int p = StringFind(j, key, from);
   if(p < 0) return "";
   int s = p + StringLen(key);
   while(s < StringLen(j) && StringGetCharacter(j, s) == ' ') s++;
   bool quoted = (StringGetCharacter(j, s) == '"');
   if(quoted) s++;
   int e = s;
   while(e < StringLen(j))
   {
      ushort c = StringGetCharacter(j, e);
      if(quoted  && c == '"') break;
      if(!quoted && (c == ',' || c == '}' || c == ']')) break;
      e++;
   }
   return StringSubstr(j, s, e - s);
}

double ExtractDbl(string j, string key)  { return StringToDouble(ExtractStrFrom(j, key, 0)); }
bool   ExtractBool(string j, string key) { return ExtractStrFrom(j, key, 0) == "true"; }
