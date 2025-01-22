FROM golang:1.20-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . /app
RUN go build -o content-api main.go

FROM alpine:3.17
WORKDIR /app
COPY --from=build /app/content-api /app/
EXPOSE 8082
CMD ["/app/content-api"]
