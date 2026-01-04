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

# Pre-download models to prevent runtime race conditions in workers
# We set strict threading limits globally to be safe
# Bump to 2 threads for better performance on high-core machines
# Total load: 8 workers * 2 threads = 16 threads (leaving room on 24-thread host)
ENV OMP_NUM_THREADS=2
ENV MKL_NUM_THREADS=2
ENV OPENBLAS_NUM_THREADS=2
ENV VECLIB_MAXIMUM_THREADS=2
ENV NUMEXPR_NUM_THREADS=2
ENV ONNX_NUM_THREADS=1

# Trigger model download during build
RUN python -c "from docling.utils.model_downloader import download_models; download_models()"

CMD ["python", "main.py"]

