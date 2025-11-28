from fastapi import FastAPI
from .database import Base, engine
from .routes import auth, publisher, receiver

# Buat semua tabel di database (sekali saat start pertama)
Base.metadata.create_all(bind=engine)

app = FastAPI(
    title="CrunchAlpha Backend API",
    version="1.0.0",
)


@app.get("/")
def root():
    return {"status": "OK", "service": "CrunchAlpha Backend API"}


app.include_router(auth.router)
app.include_router(publisher.router)
app.include_router(receiver.router)
