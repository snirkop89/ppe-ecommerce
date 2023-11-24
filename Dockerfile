FROM golang:1.21
ARG APP_NAME

WORKDIR /source
COPY go.mod go.sum ./ 
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o bin/inventory-consumer ./app/services/$APP_NAME

FROM alpine:latest
WORKDIR /app
COPY --from=0 /source/bin/inventory-consumer .
CMD [ "/app/inventory-consumer" ]