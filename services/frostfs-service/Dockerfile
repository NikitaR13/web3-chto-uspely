FROM golang:1.20-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . /app
RUN go build -o frostfs-service main.go

FROM alpine:3.17
WORKDIR /app
COPY --from=build /app/frostfs-service /app/
EXPOSE 8081
CMD ["/app/frostfs-service"]
