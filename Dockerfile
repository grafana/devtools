FROM debian:stretch-slim

RUN apt-get update && apt-get install -y ca-certificates

ENV DB "sqlite3"
ENV CONNSTRING "./test.db"
ENV ARCHIVE_URL "https://data.githubarchive.org/%d-%02d-%02d-%d.json.gz"
ENV BIN "archive"

ADD cmd/archive/archive /
ADD cmd/aggregate/aggregate /
ADD run.sh /
WORKDIR /
CMD ["sh", "-c", "/run.sh" ]
