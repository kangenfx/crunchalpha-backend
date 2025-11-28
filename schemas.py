from pydantic import BaseModel
from datetime import datetime
from typing import List, Optional


# ---------- Account ----------

class AccountRegisterRequest(BaseModel):
    account_number: str
    broker: Optional[str] = None
    label: Optional[str] = None


class AccountRegisterResponse(BaseModel):
    account_id: int
    account_number: str
    api_key: str


# ---------- Trades ----------

class TradeIn(BaseModel):
    ticket: str
    symbol: str
    direction: str  # BUY / SELL
    volume: float
    open_price: float
    open_time: datetime
    close_price: Optional[float] = None
    close_time: Optional[datetime] = None
    profit: Optional[float] = None
    status: str = "OPEN"  # OPEN / CLOSED


class PublisherPayload(BaseModel):
    equity: Optional[float] = None
    balance: Optional[float] = None
    margin: Optional[float] = None
    trades: List[TradeIn]


class TradeOut(BaseModel):
    id: int
    ticket: str
    symbol: str
    direction: str
    volume: float
    open_price: float
    open_time: datetime

    class Config:
        orm_mode = True


class UpdateTradeStatusRequest(BaseModel):
    ticket: str
    status: str  # OPEN / CLOSED
    close_price: Optional[float] = None
    close_time: Optional[datetime] = None
    profit: Optional[float] = None
