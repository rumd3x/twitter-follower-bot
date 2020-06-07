FROM golang

RUN go get github.com/ChimeraCoder/anaconda
RUN go get go.mongodb.org/mongo-driver/mongo
RUN go get github.com/joho/godotenv


ADD . /app
WORKDIR /app

RUN go build --buildmode=exe -o bot .

CMD ./bot

