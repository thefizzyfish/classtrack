FROM debian:12-slim
WORKDIR /app
COPY classtrack-linux ./classtrack
RUN chmod +x ./classtrack
EXPOSE 4000
CMD ["./classtrack", "--db", "/data/classtrack.db"]
