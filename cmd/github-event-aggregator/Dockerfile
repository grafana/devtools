FROM debian:buster-slim

RUN apt-get update && apt-get install -y ca-certificates

COPY github-event-aggregator /bin/github-event-aggregator
ENTRYPOINT [ "/bin/github-event-aggregator" ]
