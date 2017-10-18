package node

import (
	log "github.com/noxiouz/zapctx/ctxlog"
	pb "github.com/sonm-io/core/proto"
	"golang.org/x/net/context"
)

type hubAPI struct {
	// endpoint is cached mostly for debug purposes
	endpoint string
	cc       pb.HubClient
	ctx      context.Context
}

func (h *hubAPI) Status(ctx context.Context, req *pb.Empty) (*pb.HubStatusReply, error) {
	log.G(h.ctx).Debug("handling Status request")
	return h.cc.Status(ctx, req)
}

func (h *hubAPI) WorkersList(ctx context.Context, req *pb.Empty) (*pb.ListReply, error) {
	log.G(h.ctx).Debug("handling WorkersList request")
	return h.cc.List(ctx, req)
}

func (h *hubAPI) WorkerStatus(ctx context.Context, req *pb.ID) (*pb.InfoReply, error) {
	log.G(h.ctx).Debug("handling WorkersStatus request")
	return h.cc.Info(ctx, req)
}

func (h *hubAPI) GetRegistredWorkers(ctx context.Context, req *pb.Empty) (*pb.GetRegistredWorkersReply, error) {
	log.G(h.ctx).Debug("handling GetRegistredWorkers request")
	// TODO(sshaman1101): implement stub on Hub
	return nil, nil
}

func (h *hubAPI) RegisterWorker(ctx context.Context, req *pb.ID) (*pb.Empty, error) {
	log.G(h.ctx).Debug("handling RegisterWorkers request")
	// TODO(sshaman1101): implement stub on Hub
	return nil, nil
}

func (h *hubAPI) UnregisterWorker(ctx context.Context, req *pb.ID) (*pb.Empty, error) {
	log.G(h.ctx).Debug("handling UnregisterWorkers request")
	// TODO(sshaman1101): implement stub on Hub
	return nil, nil
}

func (h *hubAPI) GetWorkerProperties(ctx context.Context, req *pb.ID) (*pb.GetMinerPropertiesReply, error) {
	log.G(h.ctx).Debug("handling GetWorkerProperties request")
	return h.cc.GetMinerProperties(ctx, req)
}

func (h *hubAPI) SetWorkerProperties(ctx context.Context, req *pb.SetMinerPropertiesRequest) (*pb.Empty, error) {
	log.G(h.ctx).Debug("handling SetWorkerProperties request")
	return h.cc.SetMinerProperties(ctx, req)
}

func (h *hubAPI) GetAskPlan(ctx context.Context, req *pb.ID) (*pb.GetSlotsReply, error) {
	log.G(h.ctx).Debug("GetAskPlan")
	return h.cc.GetSlots(ctx, req)
}

func (h *hubAPI) CreateAskPlan(ctx context.Context, req *pb.AddSlotRequest) (*pb.Empty, error) {
	log.G(h.ctx).Debug("CreateAskPlan")
	return h.cc.AddSlot(ctx, req)
}

func (h *hubAPI) RemoveAskPlan(ctx context.Context, req *pb.ID) (*pb.Empty, error) {
	log.G(h.ctx).Debug("RemoveAskPlan")
	// TODO(sshaman1101): wait for 3Hren changes and fix this
	request := &pb.RemoveSlotRequest{ID: req.GetId()}
	return h.cc.RemoveSlot(ctx, request)
}

func (h *hubAPI) TaskList(ctx context.Context, req *pb.Empty) (*pb.TaskListReply, error) {
	log.G(h.ctx).Debug("handling TaskList request")
	workers, err := h.cc.List(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}

	reply := &pb.TaskListReply{
		Tasks: []*pb.InfoReply{},
	}

	for id := range workers.GetInfo() {
		info, err := h.cc.Info(ctx, &pb.ID{Id: id})
		if err != nil {
			return nil, err
		}
		reply.Tasks = append(reply.Tasks, info)
	}

	return reply, nil
}

func (h *hubAPI) TaskStatus(ctx context.Context, req *pb.ID) (*pb.TaskStatusReply, error) {
	log.G(h.ctx).Debug("handling TaskStatus request")
	return h.cc.TaskStatus(ctx, req)
}

func newHubAPI(ctx context.Context, endpoint string) (pb.HubManagementServer, error) {
	cc, err := initGrpcClient(endpoint, nil)
	if err != nil {
		return nil, err
	}

	return &hubAPI{
		endpoint: endpoint,
		cc:       pb.NewHubClient(cc),
		ctx:      ctx,
	}, nil
}