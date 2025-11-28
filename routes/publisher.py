from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session
from datetime import datetime

from ..database import get_db
from .. import models, schemas
from ..deps import get_account_by_api_key

router = APIRouter(prefix="/publisher", tags=["publisher"])


@router.post("/send-data")
def send_data(
    payload: schemas.PublisherPayload,
    db: Session = Depends(get_db),
    account: models.Account = Depends(get_account_by_api_key),
):
    """
    EA Publisher kirim snapshot data:
    - equity, balance, margin (optional disimpan nanti di tabel lain)
    - list trade (OPEN/CLOSED)
    """

    for t in payload.trades:
        existing = (
            db.query(models.Trade)
            .filter(models.Trade.account_id == account.id, models.Trade.ticket == t.ticket)
            .first()
        )

        if existing:
            # update existing trade
            existing.symbol = t.symbol
            existing.direction = t.direction
            existing.volume = t.volume
            existing.open_price = t.open_price
            existing.open_time = t.open_time
            existing.close_price = t.close_price
            existing.close_time = t.close_time
            existing.profit = t.profit
            existing.status = t.status
        else:
            # new trade
            db_trade = models.Trade(
                account_id=account.id,
                ticket=t.ticket,
                symbol=t.symbol,
                direction=t.direction,
                volume=t.volume,
                open_price=t.open_price,
                open_time=t.open_time,
                close_price=t.close_price,
                close_time=t.close_time,
                profit=t.profit,
                status=t.status,
            )
            db.add(db_trade)

    db.commit()
    return {"message": "Data received", "time": datetime.utcnow().isoformat()}
