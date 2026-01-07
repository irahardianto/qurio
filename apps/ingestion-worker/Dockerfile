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

# Create a non-root user with explicit UID
RUN useradd -m -u 1000 appuser

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Install Playwright browsers (as root)
RUN playwright install --with-deps chromium && \
    chmod -R 777 /ms-playwright

COPY . .

# Set ownership and create upload dir
RUN mkdir -p /var/lib/qurio/uploads && \
    chown -R appuser:appuser /var/lib/qurio && \
    chown -R appuser:appuser /app

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

# Set cache directories to a shared location within /app
ENV HF_HOME=/app/.cache/huggingface
ENV DOCLING_HOME=/app/.cache/docling
ENV RAPIDOCR_CACHE_DIR=/app/.cache/rapidocr

# Create cache directory and set permissions
RUN mkdir -p /app/.cache && \
    chown -R appuser:appuser /app/.cache

# Trigger model download during build (as root to write to site-packages AND shared cache)
# We run as root but ensure the cache dir is writable by appuser later
RUN python -c "from docling.utils.model_downloader import download_models; download_models()" && \
    chown -R appuser:appuser /app/.cache && \
    # Fix RapidOCR permission issue by granting ownership of the package directory to appuser
    # It tries to download models to its own package dir despite cache settings
    chown -R appuser:appuser /usr/local/lib/python3.12/site-packages/rapidocr

# Switch to non-root user
USER appuser

CMD ["python", "main.py"]