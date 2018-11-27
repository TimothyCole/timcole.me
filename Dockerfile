# Builder
FROM golang:1.11.1-alpine AS builder

RUN apk update && apk add git

WORKDIR /go/src/app
ADD . /go/src/app

# Build Backend
RUN go get ./...
RUN go build -o website

# Build Frontend
RUN apk add nodejs npm
RUN npm install
RUN npm run build

# -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

# Runner
FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /go/src/app/website /app/website
COPY --from=builder /go/src/app/build/ /app/build/

EXPOSE 80 443
ENTRYPOINT [ "./website" ]