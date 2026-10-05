package grpc

import (
	"context"
	"errors"
	pb "steam-cfg-portet/api/pb"
	"steam-cfg-portet/internal/config_service/domain"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConfigGRPCServer struct {
	pb.UnimplementedConfigServiceServer
	uc domain.UseCase
	wg *sync.WaitGroup
}

func NewConfigGRPCServer(uc domain.UseCase, wg *sync.WaitGroup) *ConfigGRPCServer {
	return &ConfigGRPCServer{uc: uc, wg: wg}
}

func (g *ConfigGRPCServer) GetConfig(ctx context.Context, req *pb.GetConfigRequest) (*pb.GetConfigResponse, error) {
	g.wg.Add(1)
	defer g.wg.Done()

	cfg, err := g.uc.GetConfig(ctx, req.GetConfigId(), req.GetGame(), req.GetAuthorNickname())
	if err != nil {
		if errors.Is(err, domain.ErrConfigNotFound) {
			return nil, status.Error(codes.NotFound, "config not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get config: %v", err)
	}

	return &pb.GetConfigResponse{
		Config: mapDomainToPB(cfg),
	}, nil
}

func (g *ConfigGRPCServer) SaveConfig(ctx context.Context, req *pb.SaveConfigRequest) (*pb.SaveConfigResponse, error) {
	g.wg.Add(1)
	defer g.wg.Done()

	domainCfg := &domain.Config{
		Game:           req.GetGame(),
		Name:           req.GetName(),
		AuthorNickName: req.GetAuthorNickname(),
		UserID:         req.GetUserId(),
	}

	configID, err := g.uc.SaveMyConfig(ctx, domainCfg)
	if err != nil {
		if errors.Is(err, domain.ErrCantFindPath) {
			return nil, status.Error(codes.NotFound, "config path not found on local disk")
		}
		if errors.Is(err, domain.ErrConfigAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "config with this ID already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to save config: %v", err)
	}

	return &pb.SaveConfigResponse{
		ConfigId: configID,
		Success:  true,
	}, nil
}

func (g *ConfigGRPCServer) DeleteConfig(ctx context.Context, req *pb.DeleteConfigRequest) (*pb.DeleteConfigResponse, error) {
	g.wg.Add(1)
	defer g.wg.Done()

	success, err := g.uc.DeleteConfig(ctx, req.GetConfigId(), req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete config: %v", err)
	}

	return &pb.DeleteConfigResponse{
		Success: success,
	}, nil
}

func mapDomainToPB(d *domain.Config) *pb.ConfigModel {
	if d == nil {
		return nil
	}

	return &pb.ConfigModel{
		Id:             d.ID,
		Game:           d.Game,
		Name:           d.Name,
		AuthorNickname: d.AuthorNickName,
		IsGlobal:       d.IsGlobal,
		UserId:         d.UserID,
		ConfigBody:     d.ConfigBody,
	}
}
