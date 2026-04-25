//+------------------------------------------------------------------+
//| CrunchAlpha Investor EA v2.1 MT4                                 |
//| AUM-Based: Copy Signal (Analyst) + Copy Trader in one EA        |
//| Supports MT4 — configure via web dashboard                       |
//| Lot calculated by backend — EA only executes                     |
//+------------------------------------------------------------------+
#property copyright "CrunchAlpha"
#property version   "2.50"
#property strict

//── Inputs ──────────────────────────────────────────────────────────
extern string EAKey      = "";                          // EA Key (from Dashboard > Copy Settings)
extern string BackendURL = "https://crunchalpha.com";   // Backend URL

//── Globals ─────────────────────────────────────────────────────────
bool     copySignalEnabled = false;
bool     copyTraderEnabled = false;
double   signalMaxLot      = 0.10;
double   maxDailyLossPct   = 5.0;
int      maxOpenTrades     = 10;
datetime lastEquityPush    = 0;
datetime lastSettingsLoad  = 0;
datetime lastSignalPoll    = 0;
datetime lastCopyPoll      = 0;
datetime lastTradeSync     = 0;
int      tradeSyncInterval = 300;
int      equityInterval    = 30;
int      settingsInterval  = 300;
int      signalInterval    = 10;
int      copyInterval      = 2;

//+------------------------------------------------------------------+
int OnInit()
{
   if(EAKey == "")
   {
      Alert("[CA] ERROR: EA Key is empty! Generate from Dashboard > Copy Settings > EA Key");
      return INIT_PARAMETERS_INCORRECT;
   }
   Print("[CA] CrunchAlpha Investor EA v2.1 MT4 started");
   EventSetTimer(5);
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
   if(now - lastEquityPush >= equityInterval)   { PushEquity();    lastEquityPush   = now; }
   if(now - lastSettingsLoad >= settingsInterval){ LoadSettings();  lastSettingsLoad = now; }
   if(IsDailyLossBreached())
   {
      static datetime lastWarn = 0;
      if(now - lastWarn > 300) { Print("[CA] STOP: Daily loss limit reached"); lastWarn = now; }
      return;
   }
   if(copySignalEnabled && now - lastSignalPoll >= signalInterval) { PollSignals();    lastSignalPoll = now; }
   if(copyTraderEnabled && now - lastCopyPoll  >= copyInterval)    { PollCopyTrades(); lastCopyPoll   = now; }
   if(now - lastTradeSync >= tradeSyncInterval) { SyncTrades(); lastTradeSync = now; }
}

//+------------------------------------------------------------------+
void PushEquity()
{
   double equity  = AccountEquity();
   double balance = AccountBalance();
   string body    = "{\"equity\":"  + DoubleToStr(equity,2) +
                    ",\"balance\":" + DoubleToStr(balance,2) + "}";
   string headers = "X-EA-Key: " + EAKey + "\r\nContent-Type: application/json\r\n";
   char post[], result[]; string rh;
   StringToCharArray(body, post, 0, StringLen(body));
   int res = WebRequest("POST", BackendURL+"/api/ea/investor/push-equity", headers, 5000, post, result, rh);
   if(res==200) Print("[CA] Equity pushed $",DoubleToStr(equity,2));
   else         Print("[CA] Equity push failed HTTP:",res);
}

//+------------------------------------------------------------------+
void LoadSettings()
{
   string headers = "X-EA-Key: " + EAKey + "\r\n";
   char post[], result[]; string rh;
   int res = WebRequest("GET", BackendURL+"/api/ea/investor/settings", headers, 5000, post, result, rh);
   if(res!=200) { Print("[CA] Settings failed HTTP:",res); return; }
   string j = CharArrayToString(result);
   copySignalEnabled = ExtractBool(j,"copySignalEnabled");
   copyTraderEnabled = ExtractBool(j,"copyTraderEnabled");
   signalMaxLot      = ExtractDbl(j, "signalMaxLot");
   maxDailyLossPct   = ExtractDbl(j, "maxDailyLossPct");
   maxOpenTrades     = (int)ExtractDbl(j,"maxOpenTrades");
   Print("[CA] Settings — Signal:",copySignalEnabled," Trader:",copyTraderEnabled,
         " MaxLot:",signalMaxLot," MaxDD:",maxDailyLossPct,"%");
}

//+------------------------------------------------------------------+
void PollSignals()
{
   string headers = "X-EA-Key: " + EAKey + "\r\n";
   char post[], result[]; string rh;
   int res = WebRequest("GET", BackendURL+"/api/ea/investor/pending-signals", headers, 5000, post, result, rh);
   if(res!=200) { Print("[CA] Signal poll failed HTTP:",res); return; }
   string j = CharArrayToString(result);
   int cp = StringFind(j,"\"count\":");
   if(cp<0) return;
   int count = (int)StringToInteger(StringSubstr(j,cp+8,5));
   if(count==0) return;
   Print("[CA] ",count," signal(s) pending");
   int pos=0;
   while(true)
   {
      int idPos = StringFind(j,"\"id\":",pos);
      if(idPos<0) break;
      int idEnd = StringFind(j,",",idPos+5);
      if(idEnd<0) break;
      long sigID = StringToInteger(StringSubstr(j,idPos+5,idEnd-idPos-5));
      string pair      = ExtractStrFrom(j,"\"pair\":",         idPos);
      string dir       = ExtractStrFrom(j,"\"direction\":",    idPos);
      double sl        = StringToDouble(ExtractStrFrom(j,"\"sl\":",           idPos));
      double tp        = StringToDouble(ExtractStrFrom(j,"\"tp\":",           idPos));
      double calcLot   = StringToDouble(ExtractStrFrom(j,"\"calculatedLot\":",idPos));
      string status    = ExtractStrFrom(j,"\"status\":",       idPos);
      string ordStatus = ExtractStrFrom(j,"\"orderStatus\":",  idPos);
      int tkPos=StringFind(j,"\"ticket\":",idPos);
      int tkEnd=StringFind(j,",",tkPos+9);
      long ticket=0;
      if(tkPos>=0&&tkEnd>=0) ticket=StringToInteger(StringSubstr(j,tkPos+9,tkEnd-tkPos-9));
      ProcessSignal(sigID,pair,dir,sl,tp,calcLot,status,ordStatus,ticket);
      pos=idEnd;
      if(pos>=StringLen(j)-5) break;
   }
}

void ProcessSignal(long sigID, string pair, string dir,
                   double sl, double tp, double calcLot,
                   string status, string ordStatus, long ticket)
{
   if((status=="CLOSED_TP"||status=="CLOSED_SL") && ordStatus=="OPENED")
   {
      if(ticket>0 && OrderSelect((int)ticket,SELECT_BY_TICKET) && OrderCloseTime()==0)
      {
         double cp2 = (OrderType()==OP_BUY)?Bid:Ask;
         if(OrderClose((int)ticket,OrderLots(),cp2,3))
            SendSignalUpdate(sigID,ticket,status,0,cp2,0);
      }
      return;
   }
   if(ordStatus=="OPENED"||ordStatus=="CLOSED_TP"||
      ordStatus=="CLOSED_SL"||ordStatus=="CLOSED_MANUAL") return;
   if(status!="RUNNING") return;
   if(OrdersTotal()>=maxOpenTrades) { Print("[CA] Signal skip — max trades"); return; }
   string sym=NormalizePair(pair);
   if(sym=="") { Print("[CA] Symbol not found: ",pair); return; }
   double lot=(calcLot>0)?MathMin(calcLot,signalMaxLot):0.01;
   lot=NormalizeLot(sym,lot);
   int cmd=(dir=="BUY")?OP_BUY:OP_SELL;
   double price=(cmd==OP_BUY)?MarketInfo(sym,MODE_ASK):MarketInfo(sym,MODE_BID);
   string comment="CA-SIG:"+IntegerToString(sigID);
   int tkt=OrderSend(sym,cmd,lot,price,3,sl,tp,comment,20260307);
   if(tkt>0) { Print("[CA] Signal opened ticket:",tkt," lot:",lot); SendSignalUpdate(sigID,tkt,"OPENED",price,0,lot); }
   else        Print("[CA] Signal failed error:",GetLastError());
}

//+------------------------------------------------------------------+
void PollCopyTrades()
{
   string headers = "X-EA-Key: " + EAKey + "\r\n";
   char post[], result[]; string rh;
   int res = WebRequest("GET", BackendURL+"/api/ea/investor/pending-copy-trades", headers, 5000, post, result, rh);
   if(res!=200) { Print("[CA] CopyTrade poll failed HTTP:",res); return; }
   string j = CharArrayToString(result);
   int cp = StringFind(j,"\"count\":");
   if(cp<0) return;
   int count=(int)StringToInteger(StringSubstr(j,cp+8,5));
   if(count==0) return;
   Print("[CA] ",count," copy trade event(s) pending");
   int pos=0;
   while(true)
   {
      int idPos=StringFind(j,"\"id\":",pos);
      if(idPos<0) break;
      int idStart=StringFind(j,"\"",idPos+5)+1;
      int idEnd=StringFind(j,"\"",idStart);
      if(idStart<0||idEnd<0) break;
      string eventID=StringSubstr(j,idStart,idEnd-idStart);
      string action  = ExtractStrFrom(j,"\"action\":",         idPos);
      string symbol  = ExtractStrFrom(j,"\"symbol\":",         idPos);
      int    dir     = (int)StringToDouble(ExtractStrFrom(j,"\"direction\":",    idPos));
      double calcLot = StringToDouble(ExtractStrFrom(j,"\"calculatedLot\":",    idPos));
      double sl      = StringToDouble(ExtractStrFrom(j,"\"sl\":",               idPos));
      double tp      = StringToDouble(ExtractStrFrom(j,"\"tp\":",               idPos));
      long   provTkt = (long)StringToDouble(ExtractStrFrom(j,"\"providerTicket\":",idPos));
      double aumUsed = StringToDouble(ExtractStrFrom(j,"\"aumUsed\":",          idPos));
      ProcessCopyTrade(eventID,action,symbol,dir,calcLot,sl,tp,provTkt,aumUsed);
      pos=idEnd;
      if(pos>=StringLen(j)-5) break;
   }
}

void ProcessCopyTrade(string eventID, string action, string symbol,
                      int dir, double calcLot, double sl, double tp,
                      long provTicket, double aumUsed)
{
   if(action=="CLOSE")
   {
      string sc="CA-CT:"+IntegerToString(provTicket);
      for(int i=OrdersTotal()-1;i>=0;i--)
      {
         if(OrderSelect(i,SELECT_BY_POS)&&OrderCloseTime()==0&&StringFind(OrderComment(),sc)>=0)
         {
            double cp2=(OrderType()==OP_BUY)?Bid:Ask;
            if(OrderClose(OrderTicket(),OrderLots(),cp2,3))
               SendCopyTradeUpdate(eventID,"EXECUTED","",OrderTicket(),OrderLots(),cp2);
            return;
         }
      }
      SendCopyTradeUpdate(eventID,"EXECUTED","Already closed",0,0,0);
      return;
   }
   if(action!="OPEN") return;
   if(OrdersTotal()>=maxOpenTrades) { SendCopyTradeUpdate(eventID,"REJECTED","Max open trades",0,0,0); return; }
   string sym=NormalizePair(symbol);
   if(sym=="") { SendCopyTradeUpdate(eventID,"REJECTED","Symbol not found: "+symbol,0,0,0); return; }
   double lot=NormalizeLot(sym,calcLot);
   if(lot<MarketInfo(sym,MODE_MINLOT)) { SendCopyTradeUpdate(eventID,"REJECTED","Lot below min: "+DoubleToStr(lot,4),0,0,0); return; }
   int cmd=(dir==0)?OP_BUY:OP_SELL;
   double price=(cmd==OP_BUY)?MarketInfo(sym,MODE_ASK):MarketInfo(sym,MODE_BID);
   string comment="CA-CT:"+IntegerToString(provTicket);
   int tkt=OrderSend(sym,cmd,lot,price,3,sl,tp,comment,20260307);
   if(tkt>0) { Print("[CA] CopyTrade opened:",tkt," lot:",lot," AUM:$",DoubleToStr(aumUsed,2)); SendCopyTradeUpdate(eventID,"EXECUTED","",tkt,lot,price); }
   else { string r="Order failed:"+IntegerToString(GetLastError()); Print("[CA] ",r); SendCopyTradeUpdate(eventID,"REJECTED",r,0,0,0); }
}

//+------------------------------------------------------------------+
void SendSignalUpdate(long sigID, long ticket, string status,
                      double openP, double closeP, double lot)
{
   string body="{\"signalId\":"+IntegerToString(sigID)+
               ",\"ticket\":"+IntegerToString(ticket)+
               ",\"status\":\""+status+"\""+
               ",\"openPrice\":"+DoubleToStr(openP,5)+
               ",\"closePrice\":"+DoubleToStr(closeP,5)+
               ",\"lotSize\":"+DoubleToStr(lot,4)+"}";
   string headers="X-EA-Key: "+EAKey+"\r\nContent-Type: application/json\r\n";
   char post[],result[]; string rh;
   StringToCharArray(body,post,0,StringLen(body));
   WebRequest("POST",BackendURL+"/api/ea/investor/order-update",headers,5000,post,result,rh);
}

void SendCopyTradeUpdate(string eventID, string status, string reason,
                         long ticket, double lot, double price)
{
   string body="{\"eventId\":\""+eventID+"\""+
               ",\"status\":\""+status+"\""+
               ",\"rejectionReason\":\""+reason+"\""+
               ",\"followerTicket\":"+IntegerToString(ticket)+
               ",\"executedLot\":"+DoubleToStr(lot,4)+
               ",\"executedPrice\":"+DoubleToStr(price,5)+"}";
   string headers="X-EA-Key: "+EAKey+"\r\nContent-Type: application/json\r\n";
   char post[],result[]; string rh;
   StringToCharArray(body,post,0,StringLen(body));
   WebRequest("POST",BackendURL+"/api/ea/investor/copy-trade-update",headers,5000,post,result,rh);
}

//+------------------------------------------------------------------+
bool IsDailyLossBreached()
{
   if(maxDailyLossPct<=0) return false;
   double balance=AccountBalance();
   return (balance-AccountEquity())>=(balance*maxDailyLossPct/100.0);
}

double NormalizeLot(string sym, double lot)
{
   double mn=MarketInfo(sym,MODE_MINLOT);
   double mx=MarketInfo(sym,MODE_MAXLOT);
   double st=MarketInfo(sym,MODE_LOTSTEP);
   lot=MathFloor(lot/st)*st;
   return MathMax(mn,MathMin(lot,mx));
}

string NormalizePair(string pair)
{
   if(MarketInfo(pair,MODE_DIGITS)>0) return pair;
   string sfx[]=  {".raw",".pro",".ecn",".std",".stp","+"};
   for(int i=0;i<6;i++) { string t=pair+sfx[i]; if(MarketInfo(t,MODE_DIGITS)>0) return t; }
   return "";
}

string ExtractStrFrom(string j, string key, int from)
{
   int p=StringFind(j,key,from);
   if(p<0) return "";
   int s=p+StringLen(key);
   while(s<StringLen(j)&&StringGetCharacter(j,s)==' ') s++;
   bool quoted=(StringGetCharacter(j,s)=='"');
   if(quoted) s++;
   int e=s;
   while(e<StringLen(j))
   {
      ushort c=StringGetCharacter(j,e);
      if(quoted&&c=='"') break;
      if(!quoted&&(c==','||c=='}'||c==']')) break;
      e++;
   }
   return StringSubstr(j,s,e-s);
}

double ExtractDbl(string j,string key)  { return StringToDouble(ExtractStrFrom(j,key,0)); }
bool   ExtractBool(string j,string key) { return ExtractStrFrom(j,key,0)=="true"; }

//+------------------------------------------------------------------+
// SYNC TRADES — kirim history trades ke backend setiap 5 menit
//+------------------------------------------------------------------+
void SyncTrades()
{
   string trades = "";
   int count = 0;
   int total = OrdersHistoryTotal();
   int start = MathMax(0, total - 200);

   for(int i = start; i < total; i++)
   {
      if(!OrderSelect(i, SELECT_BY_POS, MODE_HISTORY)) continue;

      string comment = OrderComment();
      if(StringFind(comment, "CA-CT:") < 0) continue;

      string sym    = OrderSymbol();
      double lots   = OrderLots();
      double openP  = OrderOpenPrice();
      double closeP = OrderClosePrice();
      double profit = OrderProfit();
      double swap   = OrderSwap();
      double comm   = OrderCommission();
      long   openT  = (long)OrderOpenTime();
      long   closeT = (long)OrderCloseTime();
      int    type   = OrderType();
      long   ticket = OrderTicket();

      string typeStr = (type == OP_BUY) ? "buy" : "sell";
      string status  = (closeT > 0) ? "closed" : "open";

      if(trades != "") trades += ",";
      trades += "{\"ticket\":"    + IntegerToString(ticket) +
                ",\"symbol\":\""  + sym + "\"" +
                ",\"type\":\""    + typeStr + "\"" +
                ",\"lots\":"      + DoubleToStr(lots, 2) +
                ",\"openPrice\":" + DoubleToStr(openP, 5) +
                ",\"closePrice\":"+ DoubleToStr(closeP, 5) +
                ",\"openTime\":"  + IntegerToString(openT) +
                ",\"closeTime\":" + IntegerToString(closeT) +
                ",\"profit\":"    + DoubleToStr(profit, 2) +
                ",\"swap\":"      + DoubleToStr(swap, 2) +
                ",\"commission\":"+ DoubleToStr(comm, 2) +
                ",\"status\":\""  + status + "\"" +
                ",\"comment\":\"" + comment + "\"}";
      count++;
   }

   if(count == 0) return;

   string body = "{\"trades\":[" + trades + "]}";
   string headers = "X-EA-Key: " + EAKey + "\r\nContent-Type: application/json\r\n";
   char post[], result[]; string rh;
   StringToCharArray(body, post, 0, StringLen(body));
   int res = WebRequest("POST", BackendURL+"/api/ea/investor/sync-trades", headers, 5000, post, result, rh);
   if(res == 200) Print("[CA] SyncTrades: ", count, " trades synced");
   else           Print("[CA] SyncTrades failed HTTP:", res);
}
