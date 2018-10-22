FROM debian:stretch-slim

RUN apt-get update && apt-get install -y ca-certificates

ENV DB="mysql"
ENV CONNSTRING="grafana:password@tcp(localhost:3306)/grafana?collation=utf8mb4_unicode_ci"

COPY aggregate /bin/aggregate
ENTRYPOINT [ "/bin/aggregate", "-database=${DB}", "-connectionString=${CONNSTRING}", "-maxDurationMin=10" ]
