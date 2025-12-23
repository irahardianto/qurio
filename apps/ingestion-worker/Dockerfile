FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1
ENV PLAYWRIGHT_BROWSERS_PATH="/ms-playwright"

# System dependencies for Playwright & NSQ (snappy)
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libsnappy-dev \
    git \
    # Playwright deps
    libwoff-dev libharfbuzz-dev libicu-dev libgirepository1.0-dev \
    libcairo2-dev libjpeg-dev libpng-dev libtool libnss3 libxss1 \
    libasound2 libatk-bridge2.0-0 libgtk-3-0 libgbm-dev libxkbcommon-x11-0 \
    # lxml deps
    libxml2-dev libxslt-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Install Playwright browsers
RUN playwright install --with-deps chromium

COPY . .

CMD ["python", "main.py"]

