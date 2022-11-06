FROM golang:latest
WORKDIR /distributed config
COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .
RUN go build -o server_bin github.com/sornick01/distributed_config/server
CMD ["./server_bin"]