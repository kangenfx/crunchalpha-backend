from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    DATABASE_URL: str
    SECRET_KEY: str = "super-secret-key-change-this"

    class Config:
        env_file = ".env"


settings = Settings()
