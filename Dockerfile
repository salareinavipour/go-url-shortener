FROM hub.hamdocker.ir/golang:1.15-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go build -o /url-shortener

EXPOSE 9808
CMD [ "/url-shortener" ]