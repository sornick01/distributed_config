package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	pb "github.com/sornick01/distributed_config/protos"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	//host := os.Getenv("HOST")
	//fmt.Println(host)
	//conn, err := pgx.Connect(context.Background(), "postgres://postgres:1234@database:5432/postgres")
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:1234@database:5432/postgres")
	defer conn.Close(context.Background())
	if err != nil {
		log.Fatal("error connecting to database" + err.Error())
	}
	srv := GRPCserver{Conn: conn}
	s := grpc.NewServer()
	l, err := net.Listen("tcp", ":1001")
	pb.RegisterConfigManagerServer(s, &srv)
	if err := s.Serve(l); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK connecting to database", srv.Conn)
}
