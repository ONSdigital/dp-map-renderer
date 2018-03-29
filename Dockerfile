FROM onsdigital/dp-concourse-tools-ubuntu

RUN apt-get update -y && \
    apt-get install -y librsvg2-bin

WORKDIR /app/

COPY ./build/dp-map-renderer .

CMD ./dp-map-renderer
