FROM python:3.11

WORKDIR /app

RUN apt-get update && apt-get install -y git

# Clone distobject-py
RUN git clone https://github.com/didip/distobject-py.git /app/distobject-py

# Install it
RUN pip install -e /app/distobject-py

# Copy script that will save object
COPY save_initial_data.py .

CMD ["python3", "save_initial_data.py"]
