FROM ubuntu:16.04

WORKDIR /app/

COPY ./build/dp-map-renderer .

CMD ./dp-map-renderer
