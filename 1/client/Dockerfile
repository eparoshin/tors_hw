FROM ubuntu:24.04

RUN apt-get update && apt-get install -y build-essential iproute2

WORKDIR /app

COPY . /app

RUN make

RUN chmod +x run.sh

ENTRYPOINT ["./run.sh"]
