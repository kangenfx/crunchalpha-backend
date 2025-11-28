from sqlalchemy import Column, Integer, String, DateTime, Float, ForeignKey, Boolean
from sqlalchemy.orm import relationship
from datetime import datetime
from .database import Base


class Account(Base):
    __tablename__ = "accounts"

    id = Column(Integer, primary_key=True, index=True)
    account_number = Column(String, unique=True, index=True, nullable=False)
    broker = Column(String, nullable=True)
    label = Column(String, nullable=True)
    api_key = Column(String, unique=True, index=True, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)

    trades = relationship("Trade", back_populates="account")


class Trade(Base):
    __tablename__ = "trades"

    id = Column(Integer, primary_key=True, index=True)
    account_id = Column(Integer, ForeignKey("accounts.id"), nullable=False)

    ticket = Column(String, index=True, nullable=False)
    symbol = Column(String, nullable=False)
    direction = Column(String, nullable=False)  # BUY / SELL
    volume = Column(Float, nullable=False)
    open_price = Column(Float, nullable=False)
    open_time = Column(DateTime, nullable=False)

    close_price = Column(Float, nullable=True)
    close_time = Column(DateTime, nullable=True)
    profit = Column(Float, nullable=True)

    status = Column(String, default="OPEN")  # OPEN / CLOSED
    is_copied = Column(Boolean, default=False)

    account = relationship("Account", back_populates="trades")
