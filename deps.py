from fastapi import Header, HTTPException, status, Depends
from sqlalchemy.orm import Session
from .database import get_db
from . import models


def get_account_by_api_key(
    x_api_key: str = Header(..., alias="x-api-key"),
    db: Session = Depends(get_db),
) -> models.Account:
    account = db.query(models.Account).filter(models.Account.api_key == x_api_key).first()
    if not account:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid API key",
        )
    return account
