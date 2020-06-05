FROM golang

RUN go get github.com/ChimeraCoder/anaconda

ADD . /app
WORKDIR /app

RUN go build --buildmode=exe -o bot .

CMD ./bot

