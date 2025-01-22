FROM python:3.9-slim

WORKDIR /app
COPY requirements.txt /app/
RUN pip install --no-cache-dir -r requirements.txt

COPY server.py /app/
ENV NEO_RPC=http://neo-node:10332
ENV CONTRACT_HASH=0x1234abcd...
EXPOSE 5001
CMD ["python", "server.py"]
