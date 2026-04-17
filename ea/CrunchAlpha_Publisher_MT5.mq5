//+------------------------------------------------------------------+
//|              CrunchAlpha_Publisher_MT5_v3.0.mq5                  |
//|         HTTP Direct + min_equity + SL/TP tracked from cache      |
//+------------------------------------------------------------------+
#property copyright "CrunchAlpha"
#property version   "3.00"
#property strict

input string InpApiKey           = "";
string InpBackendURL       = "https://crunchalpha.com";
int    InpCheckIntervalSec = 5;
bool   InpSyncAllHistory   = true;

struct PositionCache {
    ulong    ticket;
    string   symbol;
    double   lots;
    double   open_price;
    datetime open_time;
    double   min_equity;
    double   sl;
    double   tp;
};

PositionCache g_positions[];
int    g_posCount      = 0;
string g_accountNumber = "";

int OnInit() {
    Print("[CA v3.1] CrunchAlpha Publisher MT5 v3.0 starting...");
    if(StringLen(InpApiKey) < 60) { Alert("[CA] ERROR: API Key required!"); return(INIT_FAILED); }
    g_accountNumber = IntegerToString(AccountInfoInteger(ACCOUNT_LOGIN));
    Print("[CA] Account: ", g_accountNumber);
    Print("[CA] Balance: ", DoubleToString(AccountInfoDouble(ACCOUNT_BALANCE), 2));
    Print("[CA] Equity:  ", DoubleToString(AccountInfoDouble(ACCOUNT_EQUITY), 2));
    Print("[CA] Server:  ", InpBackendURL);
    LoadExistingPositions();
    Print("[CA] Loaded ", g_posCount, " open positions");
    if(InpSyncAllHistory) { Print("[CA] Starting full history sync..."); FullHistorySync(); }
    EventSetTimer(InpCheckIntervalSec);
    Print("[CA] Timer set to ", InpCheckIntervalSec, "s. EA ready.");
    return(INIT_SUCCEEDED);
}

void OnDeinit(const int reason) { Print("[CA] EA stopped. Reason: ", reason); EventKillTimer(); }

void OnTimer() {
    UpdateMinEquity();
    UpdateSLTP();
    CheckForNewPositions();
    CheckForClosedPositions();
    SendAccountUpdate();
}

void UpdateMinEquity() {
    double equity = AccountInfoDouble(ACCOUNT_EQUITY);
    for(int i = 0; i < g_posCount; i++) {
        if(equity < g_positions[i].min_equity) {
            g_positions[i].min_equity = equity;
            Print("[CA] New min_equity ticket=", g_positions[i].ticket, " $", DoubleToString(equity, 2));
        }
    }
}

void UpdateSLTP() {
    for(int i = 0; i < g_posCount; i++) {
        if(!PositionSelectByTicket(g_positions[i].ticket)) continue;
        double newSL = PositionGetDouble(POSITION_SL);
        double newTP = PositionGetDouble(POSITION_TP);
        if(newSL != g_positions[i].sl || newTP != g_positions[i].tp) {
            g_positions[i].sl = newSL;
            g_positions[i].tp = newTP;
            Print("[CA] SL/TP updated ticket=", g_positions[i].ticket,
                  " sl=", DoubleToString(newSL, 5), " tp=", DoubleToString(newTP, 5));
        }
    }
}

void FullHistorySync() {
    if(!HistorySelect(0, TimeCurrent())) { Print("[CA] ERROR: HistorySelect failed"); return; }
    int    totalDeals     = HistoryDealsTotal();
    double totalDeposits  = 0, totalWithdrawals = 0, initialDeposit = 0;
    bool   foundFirst     = false;
    string tradesJson     = "";
    int    tradeCount     = 0;
    Print("[CA] Processing ", totalDeals, " deals...");
    for(int i = 0; i < totalDeals; i++) {
        ulong deal = HistoryDealGetTicket(i);
        if(deal == 0) continue;
        long   dealType    = HistoryDealGetInteger(deal, DEAL_TYPE);
        double dealProfit  = HistoryDealGetDouble(deal, DEAL_PROFIT);
        string dealComment = HistoryDealGetString(deal, DEAL_COMMENT);
        if(dealType == DEAL_TYPE_BALANCE) {
            if(StringFind(dealComment,"bonus")>=0 || StringFind(dealComment,"credit")>=0 ||
               StringFind(dealComment,"Bonus")>=0 || StringFind(dealComment,"Credit")>=0) continue;
            if(!foundFirst && dealProfit > 0) { initialDeposit = dealProfit; foundFirst = true; Print("[CA] Initial deposit: $", DoubleToString(dealProfit,2)); }
            else if(foundFirst) { if(dealProfit > 0) totalDeposits += dealProfit; else totalWithdrawals += MathAbs(dealProfit); }
            continue;
        }
        if(HistoryDealGetInteger(deal, DEAL_ENTRY) != DEAL_ENTRY_OUT) continue;
        long posID = HistoryDealGetInteger(deal, DEAL_POSITION_ID);
        if(tradeCount > 0) tradesJson += ",";
        tradesJson += BuildTradeJson(deal, posID, 0, 0, 0, 0);
        tradeCount++;
    }
    Print("[CA] Sync: ", tradeCount, " trades | Initial=$", DoubleToString(initialDeposit,2),
          " Deposits=$", DoubleToString(totalDeposits,2), " Withdrawals=$", DoubleToString(totalWithdrawals,2));
    string json = "{";
    json += "\"account_number\":\"" + g_accountNumber + "\",";
    json += "\"balance\":"           + DoubleToString(AccountInfoDouble(ACCOUNT_BALANCE), 2) + ",";
    json += "\"equity\":"            + DoubleToString(AccountInfoDouble(ACCOUNT_EQUITY), 2) + ",";
    json += "\"initial_deposit\":"   + DoubleToString(initialDeposit, 2) + ",";
    json += "\"total_deposits\":"    + DoubleToString(totalDeposits, 2) + ",";
    json += "\"total_withdrawals\":" + DoubleToString(totalWithdrawals, 2) + ",";
    json += "\"closed_trades\":[" + tradesJson + "]";
    json += "}";
    int result = HTTPPost("/api/ea/sync", json);
    Print("[CA] Sync result: HTTP ", result);
}

string BuildTradeJson(ulong deal, long posID, double minEquity, double equityAtOpen, double sl, double tp) {
    string symbol     = HistoryDealGetString(deal, DEAL_SYMBOL);
    double closePrice = HistoryDealGetDouble(deal, DEAL_PRICE);
    double openPrice  = closePrice;
    long   closeTime  = HistoryDealGetInteger(deal, DEAL_TIME);
    long   openTime   = closeTime;
    for(int j = 0; j < HistoryDealsTotal(); j++) {
        ulong d = HistoryDealGetTicket(j);
        if(HistoryDealGetInteger(d, DEAL_POSITION_ID) == posID &&
           HistoryDealGetInteger(d, DEAL_ENTRY) == DEAL_ENTRY_IN) {
            openPrice = HistoryDealGetDouble(d, DEAL_PRICE);
            openTime  = HistoryDealGetInteger(d, DEAL_TIME);
            if(sl == 0) sl = HistoryDealGetDouble(d, DEAL_SL);
            if(tp == 0) tp = HistoryDealGetDouble(d, DEAL_TP);
            break;
        }
    }
    string tradeType = (HistoryDealGetInteger(deal, DEAL_TYPE) == DEAL_TYPE_SELL) ? "buy" : "sell";
    string res = "{";
    res += "\"ticket\":"         + IntegerToString(posID) + ",";
    res += "\"symbol\":\""       + symbol + "\",";
    res += "\"type\":\""         + tradeType + "\",";
    res += "\"lots\":"           + DoubleToString(HistoryDealGetDouble(deal, DEAL_VOLUME), 2) + ",";
    res += "\"open_price\":"     + DoubleToString(openPrice, 5) + ",";
    res += "\"close_price\":"    + DoubleToString(closePrice, 5) + ",";
    res += "\"sl\":"             + DoubleToString(sl, 5) + ",";
    res += "\"tp\":"             + DoubleToString(tp, 5) + ",";
    res += "\"profit\":"         + DoubleToString(HistoryDealGetDouble(deal, DEAL_PROFIT), 2) + ",";
    res += "\"swap\":"           + DoubleToString(HistoryDealGetDouble(deal, DEAL_SWAP), 2) + ",";
    res += "\"commission\":"     + DoubleToString(HistoryDealGetDouble(deal, DEAL_COMMISSION), 2) + ",";
    res += "\"open_time\":"      + IntegerToString(openTime) + ",";
    res += "\"close_time\":"     + IntegerToString(closeTime) + ",";
    res += "\"min_equity\":"     + DoubleToString(minEquity, 2) + ",";
    res += "\"equity_at_open\":" + DoubleToString(equityAtOpen, 2) + ",";
    res += "\"timestamp\":"      + IntegerToString(closeTime) + ",";
    res += "\"status\":\"closed\"";
    res += "}";
    return res;
}

void PublishOpen(ulong ticket) {
    if(!PositionSelectByTicket(ticket)) return;
    double equity = AccountInfoDouble(ACCOUNT_EQUITY);
    double sl = PositionGetDouble(POSITION_SL);
    double tp = PositionGetDouble(POSITION_TP);
    Print("[CA] Position opened: ticket=", ticket, " symbol=", PositionGetString(POSITION_SYMBOL),
          " sl=", DoubleToString(sl,5), " tp=", DoubleToString(tp,5));
    string json = "{";
    json += "\"account_number\":\"" + g_accountNumber + "\",";
    json += "\"ticket\":"           + IntegerToString((long)ticket) + ",";
    json += "\"symbol\":\""         + PositionGetString(POSITION_SYMBOL) + "\",";
    json += "\"type\":\""           + ((PositionGetInteger(POSITION_TYPE)==POSITION_TYPE_BUY)?"buy":"sell") + "\",";
    json += "\"lots\":"             + DoubleToString(PositionGetDouble(POSITION_VOLUME), 2) + ",";
    json += "\"open_price\":"       + DoubleToString(PositionGetDouble(POSITION_PRICE_OPEN), 5) + ",";
    json += "\"sl\":"               + DoubleToString(sl, 5) + ",";
    json += "\"tp\":"               + DoubleToString(tp, 5) + ",";
    json += "\"equity_at_open\":"   + DoubleToString(equity, 2) + ",";
    json += "\"timestamp\":"        + IntegerToString((long)TimeCurrent()) + ",";
    json += "\"status\":\"open\"";
    json += "}";
    int result = HTTPPost("/api/ea/trade", json);
    Print("[CA] Open trade result: HTTP ", result);
}

void PublishClose(int cacheIndex) {
    ulong  ticket = g_positions[cacheIndex].ticket;
    double minEq  = g_positions[cacheIndex].min_equity;
    double sl     = g_positions[cacheIndex].sl;
    double tp     = g_positions[cacheIndex].tp;
    if(!HistorySelect(0, TimeCurrent())) return;
    for(int i = HistoryDealsTotal()-1; i >= 0; i--) {
        ulong deal = HistoryDealGetTicket(i);
        if(deal == 0) continue;
        if(HistoryDealGetInteger(deal, DEAL_POSITION_ID) != (long)ticket) continue;
        if(HistoryDealGetInteger(deal, DEAL_ENTRY) != DEAL_ENTRY_OUT) continue;
        double openPrice = 0; long openTime = 0;
        string tradeType = (HistoryDealGetInteger(deal, DEAL_TYPE) == DEAL_TYPE_SELL) ? "buy" : "sell";
        for(int j = 0; j < HistoryDealsTotal(); j++) {
            ulong d = HistoryDealGetTicket(j);
            if(HistoryDealGetInteger(d,DEAL_POSITION_ID)==(long)ticket && HistoryDealGetInteger(d,DEAL_ENTRY)==DEAL_ENTRY_IN) {
                openPrice = HistoryDealGetDouble(d, DEAL_PRICE);
                openTime  = HistoryDealGetInteger(d, DEAL_TIME);
                break;
            }
        }
        long closeTime = HistoryDealGetInteger(deal, DEAL_TIME);
        string json = "{";
        json += "\"account_number\":\"" + g_accountNumber + "\",";
        json += "\"ticket\":"           + IntegerToString((long)ticket) + ",";
        json += "\"symbol\":\""         + HistoryDealGetString(deal, DEAL_SYMBOL) + "\",";
        json += "\"type\":\""           + tradeType + "\",";
        json += "\"lots\":"             + DoubleToString(HistoryDealGetDouble(deal, DEAL_VOLUME), 2) + ",";
        json += "\"open_price\":"       + DoubleToString(openPrice, 5) + ",";
        json += "\"close_price\":"      + DoubleToString(HistoryDealGetDouble(deal, DEAL_PRICE), 5) + ",";
        json += "\"sl\":"               + DoubleToString(sl, 5) + ",";
        json += "\"tp\":"               + DoubleToString(tp, 5) + ",";
        json += "\"profit\":"           + DoubleToString(HistoryDealGetDouble(deal, DEAL_PROFIT), 2) + ",";
        json += "\"swap\":"             + DoubleToString(HistoryDealGetDouble(deal, DEAL_SWAP), 2) + ",";
        json += "\"commission\":"       + DoubleToString(HistoryDealGetDouble(deal, DEAL_COMMISSION), 2) + ",";
        json += "\"open_time\":"        + IntegerToString(openTime) + ",";
        json += "\"close_time\":"       + IntegerToString(closeTime) + ",";
        json += "\"min_equity\":"       + DoubleToString(minEq, 2) + ",";
        json += "\"equity_at_open\":"   + DoubleToString(0, 2) + ",";
        json += "\"timestamp\":"        + IntegerToString(closeTime) + ",";
        json += "\"status\":\"closed\"";
        json += "}";
        int result = HTTPPost("/api/ea/trade", json);
        Print("[CA] Close: HTTP ", result, " ticket=", ticket,
              " profit=$", DoubleToString(HistoryDealGetDouble(deal, DEAL_PROFIT), 2),
              " sl=", DoubleToString(sl,5), " tp=", DoubleToString(tp,5),
              " min_equity=$", DoubleToString(minEq, 2));
        break;
    }
}

void SendAccountUpdate() {
    double balance=AccountInfoDouble(ACCOUNT_BALANCE), equity=AccountInfoDouble(ACCOUNT_EQUITY);
    double margin=AccountInfoDouble(ACCOUNT_MARGIN), freeMargin=AccountInfoDouble(ACCOUNT_MARGIN_FREE);
    double floating=equity-balance, openLots=0;
    int openCount=PositionsTotal();
    for(int i=0;i<openCount;i++){ulong t=PositionGetTicket(i);if(t>0){PositionSelectByTicket(t);openLots+=PositionGetDouble(POSITION_VOLUME);}}
    string json="{";
    json+="\"account_number\":\""+g_accountNumber+"\",";
    json+="\"balance\":"+DoubleToString(balance,2)+",";
    json+="\"equity\":"+DoubleToString(equity,2)+",";
    json+="\"margin\":"+DoubleToString(margin,2)+",";
    json+="\"free_margin\":"+DoubleToString(freeMargin,2)+",";
    json+="\"floating_profit\":"+DoubleToString(floating,2)+",";
    json+="\"open_lots\":"+DoubleToString(openLots,2)+",";
    json+="\"open_positions\":"+IntegerToString(openCount)+",";
    json+="\"timestamp\":"+IntegerToString((long)TimeCurrent());
    json+="}";
    int result=HTTPPost("/api/ea/account",json);
    if(result!=200) Print("[CA] Account update result: HTTP ",result);
}

void CheckForNewPositions() {
    int total=PositionsTotal();
    for(int i=0;i<total;i++){ulong ticket=PositionGetTicket(i);if(ticket>0&&!IsInCache(ticket)){AddToCache(ticket);PublishOpen(ticket);}}
}

void CheckForClosedPositions() {
    for(int i=g_posCount-1;i>=0;i--){if(!PositionSelectByTicket(g_positions[i].ticket)){PublishClose(i);RemoveFromCache(i);}}
}

bool IsInCache(ulong ticket) {
    for(int i=0;i<g_posCount;i++) if(g_positions[i].ticket==ticket) return true;
    return false;
}

void AddToCache(ulong ticket) {
    if(!PositionSelectByTicket(ticket)) return;
    double equity=AccountInfoDouble(ACCOUNT_EQUITY);
    ArrayResize(g_positions,g_posCount+1);
    g_positions[g_posCount].ticket     = ticket;
    g_positions[g_posCount].symbol     = PositionGetString(POSITION_SYMBOL);
    g_positions[g_posCount].lots       = PositionGetDouble(POSITION_VOLUME);
    g_positions[g_posCount].open_price = PositionGetDouble(POSITION_PRICE_OPEN);
    g_positions[g_posCount].open_time  = (datetime)PositionGetInteger(POSITION_TIME);
    g_positions[g_posCount].min_equity = equity;
    g_positions[g_posCount].sl         = PositionGetDouble(POSITION_SL);
    g_positions[g_posCount].tp         = PositionGetDouble(POSITION_TP);
    g_posCount++;
    Print("[CA] Cached: ticket=",ticket," sl=",DoubleToString(g_positions[g_posCount-1].sl,5),
          " tp=",DoubleToString(g_positions[g_posCount-1].tp,5));
}

void LoadExistingPositions() {
    int total=PositionsTotal();
    ArrayResize(g_positions,total);
    g_posCount=0;
    double equity=AccountInfoDouble(ACCOUNT_EQUITY);
    for(int i=0;i<total;i++){
        ulong ticket=PositionGetTicket(i);
        if(ticket>0){
            PositionSelectByTicket(ticket);
            g_positions[g_posCount].ticket     = ticket;
            g_positions[g_posCount].symbol     = PositionGetString(POSITION_SYMBOL);
            g_positions[g_posCount].lots       = PositionGetDouble(POSITION_VOLUME);
            g_positions[g_posCount].open_price = PositionGetDouble(POSITION_PRICE_OPEN);
            g_positions[g_posCount].open_time  = (datetime)PositionGetInteger(POSITION_TIME);
            g_positions[g_posCount].min_equity = equity;
            g_positions[g_posCount].sl         = PositionGetDouble(POSITION_SL);
            g_positions[g_posCount].tp         = PositionGetDouble(POSITION_TP);
            Print("[CA] Loaded: ticket=",ticket," symbol=",g_positions[g_posCount].symbol,
                  " sl=",DoubleToString(g_positions[g_posCount].sl,5),
                  " tp=",DoubleToString(g_positions[g_posCount].tp,5));
            g_posCount++;
        }
    }
}

void RemoveFromCache(int index) {
    for(int i=index;i<g_posCount-1;i++) g_positions[i]=g_positions[i+1];
    g_posCount--;
    ArrayResize(g_positions,g_posCount);
}

int HTTPPost(string endpoint, string jsonBody) {
    string url=InpBackendURL+endpoint;
    string headers="Content-Type: application/json\r\nX-API-Key: "+InpApiKey+"\r\n";
    char post[],result[];
    string resultHeaders;
    int len=StringToCharArray(jsonBody,post,0,WHOLE_ARRAY,CP_UTF8)-1;
    ArrayResize(post,len);
    int res=WebRequest("POST",url,headers,5000,post,result,resultHeaders);
    if(res==-1){
        int err=GetLastError();
        Print("[CA] WebRequest ERROR: ",err);
        if(err==4014) Print("[CA] Add ",InpBackendURL," to Allow WebRequest list");
    }
    return res;
}