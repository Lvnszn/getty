/******************************************************
# DESC    : echo client app
# AUTHOR  : Alex Stocks
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-09-06 17:24
# FILE    : main.go
******************************************************/

package main

import (
	"context"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

import (
	"github.com/AlexStocks/getty-examples/micro/proto"
	"github.com/AlexStocks/getty/micro"
	"github.com/AlexStocks/getty/rpc"
	"github.com/AlexStocks/goext/database/filter"
	"github.com/AlexStocks/goext/database/registry"
	"github.com/AlexStocks/goext/log"
	"github.com/AlexStocks/goext/net"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

const (
	pprofPath = "/debug/pprof/"
)

var (
	client *micro.Client
	seq    int64
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

////////////////////////////////////////////////////////////////////
// main
////////////////////////////////////////////////////////////////////

func main() {
	initConf()

	initProfiling()

	initClient()
	log.Info("%s starts successfull! its version=%s\n", conf.AppName, Version)

	go test()

	initSignal()
}

func initProfiling() {
	var (
		addr string
	)

	addr = gxnet.HostAddress(conf.Host, conf.ProfilePort)
	log.Info("App Profiling startup on address{%v}", addr+pprofPath)
	go func() {
		log.Info(http.ListenAndServe(addr, nil))
	}()
}

func LoadBalance(ctx context.Context, arr *gxfilter.ServiceArray) (*gxregistry.Service, error) {
	var (
		ok  bool
		seq int64
	)
	if ctx == nil {
		seq = int64(rand.Int())
	} else if seq, ok = ctx.Value("seq").(int64); !ok {
		return nil, jerrors.Errorf("illegal seq %#v", ctx.Value("seq"))
	}

	arrLen := len(arr.Arr)
	if arrLen == 0 {
		return nil, jerrors.Errorf("@arr length is 0")
	}

	service := arr.Arr[int(seq)%arrLen]
	gxlog.CInfo("seq %d, service %#v", seq, service)
	return service, nil
	// return arr.Arr[seq%arrLen], nil
	// context.WithValue(context.Background(), key, value).Value(key)
}

func initClient() {
	var err error
	client, err = micro.NewClient(&conf.ClientConfig, &conf.Registry, micro.WithServiceHash(LoadBalance))
	if err != nil {
		panic(jerrors.ErrorStack(err))
	}
}

func uninitClient() {
	client.Close()
}

func initSignal() {
	timeout, _ := time.ParseDuration(conf.FailFastTimeout)

	// signal.Notify的ch信道是阻塞的(signal.Notify不会阻塞发送信号), 需要设置缓冲
	signals := make(chan os.Signal, 1)
	// It is not possible to block SIGKILL or syscall.SIGSTOP
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-signals
		log.Info("get signal %s", sig.String())
		switch sig {
		case syscall.SIGHUP:
		// reload()
		default:
			go time.AfterFunc(timeout, func() {
				// log.Warn("app exit now by force...")
				// os.Exit(1)
				log.Exit("app exit now by force...")
				log.Close()
			})

			// 要么survialTimeout时间内执行完毕下面的逻辑然后程序退出，要么执行上面的超时函数程序强行退出
			uninitClient()
			// fmt.Println("app exit now...")
			log.Exit("app exit now...")
			log.Close()
			return
		}
	}
}

func testJSON() {
	ts := micro_examples.TestService{}
	ctx := context.WithValue(context.Background(), "seq", atomic.AddInt64(&seq, 1))
	testReq := micro_examples.TestReq{"aaa", "bbb", "ccc"}
	testRsp := micro_examples.TestRsp{}
	err := client.Call(ctx, rpc.CodecJson, ts.Service(), ts.Version(), "Test", &testReq,
		&testRsp)
	if err != nil {
		log.Error("client.Call(Json, TestService::Test) = error:%s", jerrors.ErrorStack(err))
		return
	}
	log.Info("TestService::Test(Json, param:%#v) = res:%s", testReq, testRsp)

	ctx = context.WithValue(context.Background(), "seq", atomic.AddInt64(&seq, 1))
	addReq := micro_examples.AddReq{1, 10}
	addRsp := micro_examples.AddRsp{}
	err = client.Call(ctx, rpc.CodecJson, ts.Service(), ts.Version(), "Add", &addReq, &addRsp)
	if err != nil {
		log.Error("client.Call(Json, TestService::Add) = error:%s", jerrors.ErrorStack(err))
		return
	}
	log.Info("TestService::Add(Json, req:%#v) = res:%#v", addReq, addRsp)

	ctx = context.WithValue(context.Background(), "seq", atomic.AddInt64(&seq, 1))
	errReq := micro_examples.ErrReq{1}
	errRsp := micro_examples.ErrRsp{}
	err = client.Call(ctx, rpc.CodecJson, ts.Service(), ts.Version(), "Err", &errReq, &errRsp)
	if err != nil {
		// error test case, this invocation should step into this branch.
		log.Error("client.Call(Json, TestService::Err) = error:%s", jerrors.ErrorStack(err))
		return
	}
	log.Info("TestService::Err(Json, req:%#v) = res:%s", errReq, errRsp)
}

func testProtobuf() {
	ts := micro_examples.TestService{}
	ctx := context.WithValue(context.Background(), "seq", atomic.AddInt64(&seq, 1))
	testReq := micro_examples.TestReq{"aaa", "bbb", "ccc"}
	testRsp := micro_examples.TestRsp{}
	err := client.Call(ctx, rpc.CodecProtobuf, ts.Service(), ts.Version(), "Test", &testReq,
		&testRsp)
	if err != nil {
		log.Error("client.Call(protobuf, TestService::Test) = error:%s", jerrors.ErrorStack(err))
		return
	}
	log.Info("TestService::Test(protobuf, param:%#v) = res:%s", testReq, testRsp)

	ctx = context.WithValue(context.Background(), "seq", atomic.AddInt64(&seq, 1))
	addReq := micro_examples.AddReq{1, 10}
	addRsp := micro_examples.AddRsp{}
	err = client.Call(ctx, rpc.CodecProtobuf, ts.Service(), ts.Version(), "Add", &addReq, &addRsp)
	if err != nil {
		log.Error("client.Call(protobuf, TestService::Add) = error:%s", jerrors.ErrorStack(err))
		return
	}
	log.Info("TestService::Add(protobuf, req:%#v) = res:%#v", addReq, addRsp)

	ctx = context.WithValue(context.Background(), "seq", atomic.AddInt64(&seq, 1))
	errReq := micro_examples.ErrReq{1}
	errRsp := micro_examples.ErrRsp{}
	err = client.Call(ctx, rpc.CodecProtobuf, ts.Service(), ts.Version(), "Err", &errReq, &errRsp)
	if err != nil {
		// error test case, this invocation should step into this branch.
		log.Error("client.Call(protobuf, TestService::Err) = error:%s", jerrors.ErrorStack(err))
		return
	}
	log.Info("TestService::Err(protobuf, req:%#v) = res:%#v", errReq, errRsp)
}

func test() {
	for i := 0; i < 1; i++ {
		log.Debug("start to test json:")
		testJSON()
		log.Debug("start to test pb:")
		testProtobuf()
	}
}
