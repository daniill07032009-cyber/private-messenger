FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY backend ./backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./backend/cmd/api
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
