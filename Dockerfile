FROM ubuntu:16.04

RUN apt-get update -y && \
    apt-get install -y librsvg2-bin

WORKDIR /app/

COPY ./build/dp-map-renderer .

CMD ./dp-map-renderer
