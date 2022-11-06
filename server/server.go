package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jackc/pgx/v5"
	pb "github.com/sornick01/distributed_config/protos"
)

type GRPCserver struct {
	Conn *pgx.Conn
	pb.UnimplementedConfigManagerServer
}

func (s *GRPCserver) AddService(ctx context.Context, req *pb.AddServiceRequest) (*pb.AddServiceReply, error) {
	var serviceId int
	configsJson, err := json.Marshal(req.GetConfig())
	if err != nil {
		return &pb.AddServiceReply{}, errors.New("Unable to marshall configs in json: " + err.Error())
	}

	rows, err := s.Conn.Query(context.Background(), "select * from services where name=$1", req.AppName)
	if err != nil {
		return &pb.AddServiceReply{}, err
	}
	if rows.Next() {
		return &pb.AddServiceReply{}, errors.New("unable to add service: service have been already added, use UpdateConfig")
	}
	rows.Close()
	_, err = s.Conn.Exec(context.Background(), "insert into services(name) values ($1)", req.GetAppName())
	rows, err = s.Conn.Query(context.Background(), "select id from services where name=$1", req.GetAppName())
	if err != nil {
		return &pb.AddServiceReply{}, err
	}
	if rows.Next() {
		rows.Scan(&serviceId)
	}
	rows.Close()
	_, err = s.Conn.Exec(context.Background(), "insert into configs (service_id, config, version) values ($1, $2, (select count(*) from configs where service_id=$1) + 1)", serviceId, configsJson)
	if err != nil {
		return nil, err
	}
	//s.Conn.
	return &pb.AddServiceReply{
		ServiceId: int32(serviceId),
		ConfigId:  1,
	}, nil
}

func (s *GRPCserver) DeleteConfig(ctx context.Context, req *pb.DeleteConfigRequest) (*pb.DeleteConfigReply, error) {
	var inUse bool
	rows, err := s.Conn.Query(context.Background(), "select in_use from services s join configs c on s.id = c.service_id where c.version=$1", req.Version)
	if err != nil {
		return &pb.DeleteConfigReply{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return &pb.DeleteConfigReply{DeletedConfigId: -1}, nil
	}
	rows.Scan(&inUse)
	rows.Close()
	if inUse {
		return &pb.DeleteConfigReply{DeletedConfigId: -1}, errors.New("config is in use")
	}
	_, err = s.Conn.Exec(context.Background(), "delete from configs where service_id="+
		"(select id from services where name=$1) and version=$2", req.GetAppName(), req.GetVersion())
	if err != nil {
		return &pb.DeleteConfigReply{DeletedConfigId: -1}, err
	}
	return &pb.DeleteConfigReply{DeletedConfigId: -1}, nil //TODO
}

func (s *GRPCserver) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*empty.Empty, error) {
	var serviceId int
	configsJson, err := json.Marshal(req.GetConfig())
	if err != nil {
		return &empty.Empty{}, err
	}
	rows, err := s.Conn.Query(context.Background(), "select id from services where name=$1", req.GetAppName())
	if err != nil {
		return &empty.Empty{}, nil
	}
	defer rows.Close()
	if !rows.Next() {
		return &empty.Empty{}, errors.New("there is no such service")
	}
	rows.Scan(&serviceId)
	rows.Close()
	_, err = s.Conn.Exec(context.Background(), "insert into configs (service_id, config, version) values ($1, $2, (select count(*) from configs where service_id=$1) + 1)",
		serviceId, configsJson)
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

func (s *GRPCserver) GetLatestConfig(ctx context.Context, req *pb.GetLatestConfigRequest) (*pb.GetConfigReply, error) {
	config := make(map[string]string)
	var configId, serviceId, prevId int
	rows, err := s.Conn.Query(context.Background(), "select c.config, c.id, s.id from services s join configs c on s.id = c.service_id "+
		"where version=(select max(version) from configs where c.service_id = (select id from services where name=$1))", req.GetAppName())
	if err != nil {
		return &pb.GetConfigReply{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return &pb.GetConfigReply{}, errors.New("no such config")
	}
	rows.Scan(&config, &configId, &serviceId)
	rows.Close()
	rows, err = s.Conn.Query(context.Background(), "select id from configs where service_id=$1 and in_use=true", serviceId)
	if err != nil {
		return &pb.GetConfigReply{}, err
	}
	rows.Next()
	rows.Scan(&prevId)
	rows.Close()
	_, err = s.Conn.Exec(context.Background(), "update configs set in_use=false where id=$1", prevId)
	if err != nil {
		return &pb.GetConfigReply{}, err
	}
	_, err = s.Conn.Exec(context.Background(), "update configs set in_use=true where id=$1", configId)
	return &pb.GetConfigReply{Config: config}, nil
}

func (s *GRPCserver) GetConfigByVersion(ctx context.Context, req *pb.GetConfigByVersionRequest) (*pb.GetConfigReply, error) {
	config := make(map[string]string)
	var serviceId, configId, prevId int
	rows, err := s.Conn.Query(context.Background(), "select c.config, c.id, s.id from services s join configs c on "+
		"s.id = c.service_id where s.name=$1 and c.version=$2", req.GetAppName(), req.GetVersion())
	if err != nil {
		return &pb.GetConfigReply{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return &pb.GetConfigReply{}, errors.New("no such config")
	}
	rows.Scan(&config, &configId, &serviceId)
	rows.Close()
	rows, err = s.Conn.Query(context.Background(), "select id from configs where service_id=$1 and in_use=true", serviceId)
	if err != nil {
		return &pb.GetConfigReply{}, err
	}
	rows.Next()
	rows.Scan(&prevId)
	rows.Close()
	_, err = s.Conn.Exec(context.Background(), "update configs set in_use=false where id=$1", prevId)
	if err != nil {
		return &pb.GetConfigReply{}, err
	}
	_, err = s.Conn.Exec(context.Background(), "update configs set in_use=true where id=$1", configId)
	return &pb.GetConfigReply{Config: config}, nil
}
