from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session
from typing import List

from ..database import get_db
from .. import models, schemas

router = APIRouter(prefix="/trades", tags=["receiver"])


@router.get("/pull-open", response_model=List[schemas.TradeOut])
def pull_open_trades(db: Session = Depends(get_db)):
    """
    Receiver: ambil semua posisi OPEN (versi simple)
    Nanti bisa di-filter per follower, per master, dll.
    """
    trades = db.query(models.Trade).filter(models.Trade.status == "OPEN").all()
    return trades


@router.post("/update-status")
def update_trade_status(payload: schemas.UpdateTradeStatusRequest, db: Session = Depends(get_db)):
    trade = db.query(models.Trade).filter(models.Trade.ticket == payload.ticket).first()
    if not trade:
        return {"updated": False, "reason": "Trade not found"}

    trade.status = payload.status
    trade.close_price = payload.close_price
    trade.close_time = payload.close_time
    trade.profit = payload.profit

    db.commit()
    return {"updated": True}
