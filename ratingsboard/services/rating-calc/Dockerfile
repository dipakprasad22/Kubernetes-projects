FROM python:3.12-slim AS runtime
WORKDIR /app
ENV PYTHONUNBUFFERED=1
RUN useradd -r -u 1001 calc
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY calc.py .
USER calc
ENTRYPOINT ["python", "calc.py"]
