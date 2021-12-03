package dummy

import (
	"context"
	_ "embed"
	"encoding/json"
	"net"
	"runtime"
	"sort"
	"sync"

	"github.com/gradusp/crispy-dummy/pkg/dummy"
	"github.com/gradusp/go-platform/pkg/slice"
	"github.com/gradusp/go-platform/server"
	grpcRt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//GetSwaggerDocs get swagger spec docs
func GetSwaggerDocs() (*server.SwaggerSpec, error) {
	const api = "dummy/GetSwaggerDocs"
	ret := new(server.SwaggerSpec)
	err := json.Unmarshal(dummyRawSwagger, ret)
	return ret, errors.Wrap(err, api)
}

//NewAnnounceService creates AnnounceService as server.APIService
func NewAnnounceService(ctx context.Context, netInterfaceName string) (server.APIService, error) {
	const api = "dummy/NewAnnounceService"

	netIface, err := net.InterfaceByName(netInterfaceName)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: net-intarface('%s')", api, netInterfaceName)
	}
	appCtx, stop := context.WithCancel(ctx)

	ret := &announceService{
		appCtx:       appCtx,
		stop:         stop,
		netInterface: netIface,
		sema:         make(chan struct{}, 1),
	}
	runtime.SetFinalizer(ret, func(o *announceService) {
		if o.sema != nil {
			close(o.sema)
		}
	})
	return ret, nil
}

var (
	_ dummy.AnnounceServiceServer  = (*announceService)(nil)
	_ server.APIService            = (*announceService)(nil)
	_ server.APIGatewayProxy       = (*announceService)(nil)
	_ server.APIServiceOnStopEvent = (*announceService)(nil)

	//go:embed dummy.swagger.json
	dummyRawSwagger []byte
)

type announceService struct {
	dummy.UnimplementedAnnounceServiceServer
	appCtx       context.Context
	stop         func()
	netInterface *net.Interface

	sema chan struct{}
}

//Description impl server.APIService
func (srv *announceService) Description() grpc.ServiceDesc {
	return dummy.AnnounceService_ServiceDesc
}

//RegisterGRPC impl server.APIService
func (srv *announceService) RegisterGRPC(_ context.Context, s *grpc.Server) error {
	dummy.RegisterAnnounceServiceServer(s, srv)
	return nil
}

//RegisterProxyGW impl server.APIGatewayProxy
func (srv *announceService) RegisterProxyGW(ctx context.Context, mux *grpcRt.ServeMux, c *grpc.ClientConn) error {
	return dummy.RegisterAnnounceServiceHandler(ctx, mux, c)
}

//OnStop impl server.APIServiceOnStopEvent
func (srv *announceService) OnStop() {
	srv.stop()
}

//GetState ...
func (srv *announceService) GetState(ctx context.Context, _ *emptypb.Empty) (*dummy.GetStateResponse, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("net-interface", srv.netInterface.Name))

	leave, e := srv.enter(ctx)
	if e != nil {
		return nil, e
	}
	defer leave()
	var addrs []net.Addr
	if addrs, e = srv.netInterface.Addrs(); e != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "get addresses from '%s' interface",
			srv.netInterface.Name)
	}
	resp := new(dummy.GetStateResponse)
	resp.Services = make([]string, 0, len(addrs))
	for _, a := range addrs {
		resp.Services = append(resp.Services, a.String())
	}
	sort.Strings(resp.Services)
	_ = slice.DedupSlice(&resp.Services, func(i, j int) bool {
		return resp.Services[i] == resp.Services[j]
	})
	return resp, nil
}

//RemoveIP ...
func (srv *announceService) RemoveIP(ctx context.Context, req *dummy.RemoveIpRequest) (*emptypb.Empty, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("net-interface", srv.netInterface.Name),
		attribute.String("IP-to-REMOVE", req.GetIp()),
	)
	leave, e := srv.enter(ctx)
	if e != nil {
		return nil, e
	}
	defer leave()
	ip := req.GetIp()
	if len(ip) == 0 || net.ParseIP(ip) == nil {
		return nil, status.Errorf(codes.InvalidArgument, "provided invalid IP(%s)", ip)
	}

	addr2del := ip + "/32"
	var found bool
	if found, e = srv.isAddrIn(addr2del); e != nil {
		return nil, status.Errorf(codes.Internal, "isAddrIn(%s) -> %v", addr2del, e)
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "addr '%s' is not found from '%s' interface",
			addr2del, srv.netInterface.Name)
	}

	var lnk netlink.Link
	if lnk, e = netlink.LinkByName(srv.netInterface.Name); e != nil {
		return nil, status.Errorf(codes.Internal, "'netlink': LinkByName('%s') -> %v",
			srv.netInterface.Name, e)
	}
	var addr *netlink.Addr
	if addr, e = netlink.ParseAddr(addr2del); e != nil {
		return nil, status.Errorf(codes.Internal, "netlink: LinkByName('%s') ->  %v",
			addr2del, e)
	}
	if e = netlink.AddrDel(lnk, addr); e != nil {
		return nil, status.Errorf(codes.Internal, "netlink: AddrDel('%s') -> %v",
			addr, e)
	}

	return new(emptypb.Empty), nil
}

//AddIP ...
func (srv *announceService) AddIP(ctx context.Context, req *dummy.AddIpRequest) (*emptypb.Empty, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("net-interface", srv.netInterface.Name),
		attribute.String("IP-to-ADD", req.GetIp()),
	)

	leave, e := srv.enter(ctx)
	if e != nil {
		return nil, e
	}
	defer leave()

	ip := req.GetIp()
	if len(ip) == 0 || net.ParseIP(ip) == nil {
		return nil, status.Errorf(codes.InvalidArgument, "provided invalid IP(%s)", ip)
	}
	addr2Add := ip + "/32"
	var found bool
	if found, e = srv.isAddrIn(addr2Add); e != nil {
		return nil, status.Errorf(codes.Internal, "isAddrIn('%s') -> %v", addr2Add, e)
	}

	if found {
		return nil, status.Errorf(codes.AlreadyExists, "addr '%s' already exist in '%s' interface",
			addr2Add, srv.netInterface.Name)
	}

	var lnk netlink.Link
	if lnk, e = netlink.LinkByName(srv.netInterface.Name); e != nil {
		return nil, status.Errorf(codes.Internal, "netlink: LinkByName('%s') -> %v",
			srv.netInterface.Name, e)
	}
	var addr *netlink.Addr
	if addr, e = netlink.ParseAddr(addr2Add); e != nil {
		return nil, status.Errorf(codes.Internal, "netlink: ParseAddr('%s') -> %v",
			addr2Add, e)
	}
	if e = netlink.AddrAdd(lnk, addr); e != nil {
		return nil, status.Errorf(codes.Internal, "netlink: AddrAdd('%s') -> %v",
			addr, e)
	}
	return new(emptypb.Empty), nil
}

func (srv *announceService) enter(ctx context.Context) (leave func(), err error) {
	select {
	case <-srv.appCtx.Done():
		err = srv.appCtx.Err()
	case <-ctx.Done():
		err = ctx.Err()
	case srv.sema <- struct{}{}:
		var o sync.Once
		leave = func() {
			o.Do(func() {
				<-srv.sema
			})
		}
		return
	}
	err = status.FromContextError(err).Err()
	return
}

func (srv *announceService) isAddrIn(addr string) (bool, error) {
	addrs, err := srv.netInterface.Addrs()
	if err != nil {
		return false, err
	}
	for _, a := range addrs {
		if a.String() == addr {
			return true, nil
		}
	}
	return false, nil
}
