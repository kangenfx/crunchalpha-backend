//+------------------------------------------------------------------+
//| CrunchAlpha_Analyst_PriceFeed_v2.mq5                             |
//| Configurable pairs via input settings                            |
//+------------------------------------------------------------------+
#property copyright "CrunchAlpha"
#property version   "2.00"
#property strict

input string InpApiUrl     = "https://crunchalpha.com";
input string InpEAKey      = "ea-crunchalpha-2026-pricefeed";
input int    InpPollSec    = 5;
input bool   InpDebugLog   = true;

// ── Pairs by category — edit from Inputs tab, no recompile needed ──
input string InpPairsForex    = "EURUSD,GBPUSD,USDJPY,AUDUSD,USDCAD,USDCHF,NZDUSD,GBPJPY,EURJPY,EURGBP,AUDCAD,AUDCHF,AUDJPY,AUDNZD,CADCHF,CADJPY,CHFJPY,EURNZD,GBPAUD,GBPCAD,GBPCHF,GBPNZD,NZDCAD,NZDCHF,NZDJPY";
input string InpPairsCommodity= "XAUUSD,XAGUSD,XPTUSD,XPDUSD,USOIL,UKOIL,NGAS";
input string InpPairsIndex    = "US30,NAS100,SPX500,GER40,UK100,FRA40,AUS200,JPN225,HK50";
input string InpPairsCrypto   = "BTCUSD,ETHUSD,LTCUSD,XRPUSD,BNBUSD,SOLUSD,ADAUSD,DOTUSD";
input string InpPairsCustom   = "";  // Add any extra pairs here, comma separated

datetime g_lastPoll   = 0;
int      g_totalUpdates = 0;
string   g_allPairs[];
int      g_pairCount  = 0;

//+------------------------------------------------------------------+
int OnInit() {
   Print("=== CrunchAlpha Analyst Price Feed v2.00 ===");
   Print("API: ", InpApiUrl);
   
   // Build master pair list from all input groups
   string combined = "";
   if(StringLen(InpPairsForex)     > 0) combined += InpPairsForex     + ",";
   if(StringLen(InpPairsCommodity) > 0) combined += InpPairsCommodity + ",";
   if(StringLen(InpPairsIndex)     > 0) combined += InpPairsIndex     + ",";
   if(StringLen(InpPairsCrypto)    > 0) combined += InpPairsCrypto    + ",";
   if(StringLen(InpPairsCustom)    > 0) combined += InpPairsCustom    + ",";
   
   // Remove trailing comma
   if(StringLen(combined) > 0 && StringGetCharacter(combined, StringLen(combined)-1) == ',')
      combined = StringSubstr(combined, 0, StringLen(combined)-1);
   
   // Split into array
   g_pairCount = StringSplit(combined, ',', g_allPairs);
   
   // Trim spaces and validate each pair
   int valid = 0;
   for(int i = 0; i < g_pairCount; i++) {
      StringTrimLeft(g_allPairs[i]);
      StringTrimRight(g_allPairs[i]);
      StringToUpper(g_allPairs[i]);
      if(StringLen(g_allPairs[i]) > 0) valid++;
   }
   
   Print("Total pairs configured: ", g_pairCount, " | Valid: ", valid);
   EventSetTimer(InpPollSec);
   return(INIT_SUCCEEDED);
}

void OnDeinit(const int reason) {
   EventKillTimer();
   Print("Price Feed stopped. Total signal updates: ", g_totalUpdates);
}

void OnTick() {
   datetime now = TimeCurrent();
   if(now - g_lastPoll >= InpPollSec) {
      SendBatchPrices();
      g_lastPoll = now;
   }
}

void OnTimer() {
   SendBatchPrices();
}

//+------------------------------------------------------------------+
void SendBatchPrices() {
   string pricesJson = "";
   int count = 0;

   for(int i = 0; i < g_pairCount; i++) {
      string sym = g_allPairs[i];
      if(StringLen(sym) == 0) continue;

      // Try to select symbol — some brokers use suffix e.g. XAUUSD.raw
      if(!SymbolSelect(sym, true)) {
         // Try common suffixes
         string suffixes[] = {".raw",".pro",".ecn","m","_i",".i"};
         bool found = false;
         for(int s = 0; s < ArraySize(suffixes); s++) {
            if(SymbolSelect(sym + suffixes[s], true)) {
               sym = sym + suffixes[s];
               found = true;
               break;
            }
         }
         if(!found) continue;
      }

      double bid = SymbolInfoDouble(sym, SYMBOL_BID);
      double ask = SymbolInfoDouble(sym, SYMBOL_ASK);
      if(bid <= 0 || ask <= 0) continue;

      // Determine decimal places based on price magnitude
      int digits = (int)SymbolInfoInteger(sym, SYMBOL_DIGITS);

      if(count > 0) pricesJson += ",";
      pricesJson += "{";
      pricesJson += "\"pair\":\"" + g_allPairs[i] + "\","; // Always send original pair name
      pricesJson += "\"bid\":" + DoubleToString(bid, digits) + ",";
      pricesJson += "\"ask\":" + DoubleToString(ask, digits);
      pricesJson += "}";
      count++;
   }

   if(count == 0) {
      if(InpDebugLog) Print("[WARN] No valid prices — check symbol names for your broker");
      return;
   }

   string body = "{\"prices\":[" + pricesJson + "]}";
   if(InpDebugLog) Print("[SEND] ", count, " pairs");

   string result = HttpPost(InpApiUrl + "/api/ea/analyst/batch-update", body);

   if(result != "") {
      int updPos = StringFind(result, "\"updated\":");
      if(updPos >= 0) {
         string updStr = StringSubstr(result, updPos + 10, 3);
         int updated = (int)StringToInteger(updStr);
         if(updated > 0) {
            g_totalUpdates += updated;
            Print("[✅ SIGNAL UPDATE] ", updated, " signal(s) changed status! Total: ", g_totalUpdates);
            Print("[DETAIL] ", result);
         } else {
            if(InpDebugLog) Print("[OK] ", count, " prices sent, no status changes");
         }
      }
   } else {
      Print("[ERROR] No response — check URL and MT5 allowed URLs list");
   }
}

//+------------------------------------------------------------------+
string HttpPost(string url, string body) {
   char   post[];
   char   result[];
   string headers      = "Content-Type: application/json\r\nX-EA-Key: " + InpEAKey;
   string resultHeaders;

   int bodyLen = StringLen(body);
   ArrayResize(post, bodyLen);
   StringToCharArray(body, post, 0, bodyLen);

   int res = WebRequest("POST", url, headers, 5000, post, result, resultHeaders);

   if(res == -1) {
      int err = GetLastError();
      if(err == 4014)
         Print("[ERROR] URL not allowed! Go to MT5 → Tools → Options → Expert Advisors → Add URL: ", InpApiUrl);
      else
         Print("[HTTP ERROR] Code: ", err);
      return "";
   }

   if(res != 200) {
      if(InpDebugLog) Print("[HTTP] Status ", res, ": ", CharArrayToString(result));
      return "";
   }

   return CharArrayToString(result);
}
