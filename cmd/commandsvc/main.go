package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/boltdb/bolt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/micromdm/command"
	"github.com/micromdm/command/service/simple"
	nsq "github.com/nsqio/go-nsq"
	"github.com/nsqio/nsq/nsqd"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

func main() {
	var (
		httpAddr    = flag.String("http.addr", "0.0.0.0:8080", "HTTP listen address")
		nsqdTCPAddr = flag.String("nsqd.tcp.addr", "0.0.0.0:4150", "NSQD tcp.listen address")
	)
	flag.Parse()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stdout)
		logger = log.NewContext(logger).With("ts", log.DefaultTimestampUTC)
		logger = log.NewContext(logger).With("caller", log.DefaultCaller)
	}

	logger.Log("msg", "server started")
	defer logger.Log("msg", "server stopped")

	ctx := context.Background()
	// setup BoltDB
	db, err := bolt.Open("mdm_commands.bolt", 0666, nil)
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	// setup nsq
	done := make(chan bool)
	go func() {
		opts := nsqd.NewOptions()
		opts.TCPAddress = *nsqdTCPAddr
		nsqd := nsqd.New(opts)
		nsqd.Main()

		// wait until we are told to continue and exit
		<-done
		nsqd.Exit()
	}()

	cfg := nsq.NewConfig()
	producer, err := nsq.NewProducer(*nsqdTCPAddr, cfg)
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}
	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "addsvc",
			Name:      "request_duration_ns",
			Help:      "Request duration in nanoseconds.",
		}, []string{"method", "success"})
	}
	var payloads metrics.Counter
	{
		// Business level metrics.
		payloads = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "commandsvc",
			Name:      "payloads_created",
			Help:      "Total count of payloads created by the NewCommand  method.",
		}, []string{})
	}

	var svc command.Service
	{
		svc, err = simple.NewService(db, producer)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
			svc = command.ServiceLoggingMiddleware(logger)(svc)
			svc = command.ServiceInstrumentingMiddleware(payloads)(svc)
		}
	}

	var commandEndpoint endpoint.Endpoint
	{
		newCommandDuration := duration.With("method", "NewCommand")
		newCommandLogger := log.NewContext(logger).With("method", "NewCommand")

		commandEndpoint = command.MakeNewCommandEndpoint(svc)
		commandEndpoint = command.EndpointInstrumentingMiddleware(
			newCommandDuration)(commandEndpoint)
		commandEndpoint = command.EndpointLoggingMiddleware(
			newCommandLogger)(commandEndpoint)
	}

	endpoints := command.Endpoints{
		NewCommandEndpoint: commandEndpoint,
	}

	r := mux.NewRouter()
	{
		httpLogger := log.NewContext(logger).With("transport", "http")
		opts := []httptransport.ServerOption{
			httptransport.ServerErrorLogger(httpLogger),
			httptransport.ServerErrorEncoder(command.EncodeError),
		}
		handlers := command.MakeHTTPHandlers(ctx, endpoints, opts...)
		r.Handle("/v1/commands", handlers.NewCommandHandler).Methods("POST")
		r.Handle("/metrics", stdprometheus.Handler())
	}

	errc := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()
	go func() {
		logger := log.NewContext(logger).With("transport", "HTTP")
		logger.Log("addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, r)
	}()

	logger.Log("exit", <-errc)
}
