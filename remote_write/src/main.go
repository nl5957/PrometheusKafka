package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/metrics/PrometheusToKafka/kafka"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/prometheus/prompb"
)

type config struct {
	kafkaUrl             		string
	remoteTimeout           time.Duration
	listenAddr              string
	promlogConfig           promlog.Config
}

func main() {
	cfg := parseFlags()

	logger := promlog.New(&cfg.promlogConfig)

	level.Info(logger).Log("msg", "kafkaUrl", cfg.kafkaUrl)
	level.Info(logger).Log("msg", "remoteTimeout", cfg.remoteTimeout)
	level.Info(logger).Log("msg", "listenAddr", cfg.listenAddr)

	writers := buildClients(logger, cfg)
	if err := serve(logger, cfg.listenAddr, writers); err != nil {
		level.Error(logger).Log("msg", "Failed to listen", "addr", cfg.listenAddr, "err", err)
		os.Exit(1)
	}
}

func parseFlags() *config {
	a := kingpin.New(filepath.Base(os.Args[0]), "Remote Kafka adapter")
	a.HelpFlag.Short('h')

	cfg := &config{
		kafkaUrl: os.Getenv("KAFKA_URL"),
		listenAddr: os.Getenv("LISTEN_ADDRESS"),
		promlogConfig:    promlog.Config{},
	}

	a.Flag("kafka-url", "The URL of the remote OpenTSDB server to send samples to. None, if empty.").
		Default("localhost:9092").StringVar(&cfg.kafkaUrl)
	a.Flag("send-timeout", "The timeout to use when sending samples to the remote storage.").
		Default("30s").DurationVar(&cfg.remoteTimeout)
	a.Flag("web.listen-address", "Address to listen on for web endpoints.").
		Default(":9201").StringVar(&cfg.listenAddr)

	flag.AddFlags(a, &cfg.promlogConfig)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	return cfg
}

type writer interface {
 	Write(samples model.Samples) error
 	Name() string
} 

func buildClients(logger log.Logger, cfg *config) ([]writer) {
	var writers []writer
	c := kafka.NewClient(
		log.With(logger, "storage", "Kafka"),
		cfg.kafkaUrl,
		cfg.remoteTimeout,
	)
	writers = append(writers, c)

	level.Info(logger).Log("msg", "Starting up...")
	return writers
}
 
func serve(logger log.Logger, addr string, writers []writer) error {
 	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
 		compressed, err := ioutil.ReadAll(r.Body)
 		if err != nil {
 			level.Error(logger).Log("msg", "Read error", "err", err.Error())
 			http.Error(w, err.Error(), http.StatusInternalServerError)
 			return
 		}
 
 		reqBuf, err := snappy.Decode(nil, compressed)
 		if err != nil {
 			level.Error(logger).Log("msg", "Decode error", "err", err.Error())
 			http.Error(w, err.Error(), http.StatusBadRequest)
 			return
 		}
 
 		var req prompb.WriteRequest
 		if err := proto.Unmarshal(reqBuf, &req); err != nil {
 			level.Error(logger).Log("msg", "Unmarshal error", "err", err.Error())
 			http.Error(w, err.Error(), http.StatusBadRequest)
 			return
		 }
		 
		level.Info(logger).Log("msg", "remote_write", req)
 
 		samples := protoToSamples(&req)
 
 		var wg sync.WaitGroup
 		for _, w := range writers {
 			wg.Add(1)
 			go func(rw writer) {
 				sendSamples(logger, rw, samples)
 				wg.Done()
 			}(w)
 		}
 		wg.Wait()
 	})
 
 	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
 		compressed, err := ioutil.ReadAll(r.Body)
 		if err != nil {
 			level.Error(logger).Log("msg", "Read error", "err", err.Error())
 			http.Error(w, err.Error(), http.StatusInternalServerError)
 			return
 		}
 
 		reqBuf, err := snappy.Decode(nil, compressed)
 		if err != nil {
 			level.Error(logger).Log("msg", "Decode error", "err", err.Error())
 			http.Error(w, err.Error(), http.StatusBadRequest)
 			return
 		}
 
 		var req prompb.ReadRequest
 		if err := proto.Unmarshal(reqBuf, &req); err != nil {
 			level.Error(logger).Log("msg", "Unmarshal error", "err", err.Error())
 			http.Error(w, err.Error(), http.StatusBadRequest)
 			return
		 }
		 
		level.Info(logger).Log("msg", "remote_read", req)
 
 	//	//// TODO: Support reading from more than one reader and merging the results.
 	//	//if len(readers) != 1 {
 	//	//	http.Error(w, fmt.Sprintf("expected exactly one reader, found %d readers", len(readers)), http.StatusInternalServerError)
 	//	//	return
 	//	//}
 	//	//reader := readers[0]
  //  //
 	//	//var resp *prompb.ReadResponse
 	//	//resp, err = reader.Read(&req)
 	//	//if err != nil {
 	//	//	level.Warn(logger).Log("msg", "Error executing query", "query", req, "storage", reader.Name(), "err", err)
 	//	//	http.Error(w, err.Error(), http.StatusInternalServerError)
 	//	//	return
 	//	//}
  //  //
 	//	//data, err := proto.Marshal(resp)
 	//	//if err != nil {
 	//	//	http.Error(w, err.Error(), http.StatusInternalServerError)
 	//	//	return
 	//	//}
  //  //
 	//	//w.Header().Set("Content-Type", "application/x-protobuf")
 	//	//w.Header().Set("Content-Encoding", "snappy")
  //  //
 	//	//compressed = snappy.Encode(nil, data)
 	//	//if _, err := w.Write(compressed); err != nil {
 	//	//	level.Warn(logger).Log("msg", "Error writing response", "storage", reader.Name(), "err", err)
 	//	//}
 	})
 
 	return http.ListenAndServe(addr, nil)
}
 
func protoToSamples(req *prompb.WriteRequest) model.Samples {
 	var samples model.Samples
 	for _, ts := range req.Timeseries {
 		metric := make(model.Metric, len(ts.Labels))
 		for _, l := range ts.Labels {
 			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
 		}
 
 		for _, s := range ts.Samples {
 			samples = append(samples, &model.Sample{
 				Metric:    metric,
 				Value:     model.SampleValue(s.Value),
 				Timestamp: model.Time(s.Timestamp),
 			})
 		}
 	}
 	return samples
}
 
func sendSamples(logger log.Logger, w writer, samples model.Samples) {
	err := w.Write(samples)
	if err != nil {
		level.Warn(logger).Log("msg", "Error sending samples to remote storage", "err", err, "storage", w.Name(), "num_samples", len(samples))
	}
}
