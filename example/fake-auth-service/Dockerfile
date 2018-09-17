FROM golang:1.11.0 as builder
WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o app


FROM alpine:latest as tz-certs
RUN apk --no-cache add tzdata zip ca-certificates
WORKDIR /usr/share/zoneinfo
RUN zip -r -0 -q /zoneinfo.zip .


FROM alpine
COPY --from=builder "/src/app" "/bin/app"
ENV ZONEINFO /zoneinfo.zip
COPY --from=tz-certs /zoneinfo.zip /
COPY --from=tz-certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 5000
ENTRYPOINT ["/bin/app"]
CMD ["-listen-addr", ":5000"]
