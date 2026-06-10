FROM python:3.11-slim AS builder
RUN pip install --no-cache-dir --upgrade pip==23.0.1 && \
pip install --no-cache-dir uv==0.11.7
WORKDIR /app
COPY pyproject.toml uv.lock /app/
RUN uv sync --no-cache

FROM python:3.11-slim
RUN apt-get update && apt-get -y install curl=8.14.1-2+deb13u3 --no-install-recommends && \
rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/.venv /app/.venv
COPY worker.py /app
RUN useradd -m app && chown -R app:app /app
USER app
ENV PATH="/app/.venv/bin:$PATH"
ENV PYTHONPATH=/app
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD [ "pgrep", "-x", "python3" ]
ENTRYPOINT [ "python3", "worker.py"]