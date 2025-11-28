# CrunchAlpha Backend API

Backend sederhana untuk menerima data akun + trades dari EA Publisher MT4
dan menyediakan endpoint untuk Receiver.

## Run lokal

```bash
pip install -r requirements.txt
uvicorn app.main:app --reload
