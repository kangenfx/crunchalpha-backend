from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from uuid import uuid4

from ..database import get_db
from .. import models, schemas

router = APIRouter(prefix="/auth", tags=["auth"])


@router.post("/register-account", response_model=schemas.AccountRegisterResponse)
def register_account(payload: schemas.AccountRegisterRequest, db: Session = Depends(get_db)):
    existing = db.query(models.Account).filter(
        models.Account.account_number == payload.account_number
    ).first()
    if existing:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Account already registered",
        )

    api_key = uuid4().hex

    account = models.Account(
        account_number=payload.account_number,
        broker=payload.broker,
        label=payload.label,
        api_key=api_key,
    )
    db.add(account)
    db.commit()
    db.refresh(account)

    return schemas.AccountRegisterResponse(
        account_id=account.id,
        account_number=account.account_number,
        api_key=account.api_key,
    )
