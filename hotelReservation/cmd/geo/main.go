package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/harlow/go-micro-services/services/geo"
	"github.com/harlow/go-micro-services/tracing"
	"github.com/harlow/go-micro-services/tune"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	tune.Init()
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()

	log.Info().Msg("Reading config...")
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Error().Msgf("Got error while reading config: %v", err)
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var result map[string]string
	json.Unmarshal([]byte(byteValue), &result)

	log.Info().Msgf("Read database URL: %v", result["GeoMongoAddress"])
	log.Info().Msg("Initializing DB connection...")
	ctx := context.Background()
	mongo_client := initializeDatabase(ctx, result["GeoMongoAddress"])
	defer mongo_client.Disconnect(ctx)
	log.Info().Msg("Successfull")

	serv_port, _ := strconv.Atoi(result["GeoPort"])
	serv_ip := result["GeoIP"]

	log.Info().Msgf("Read target port: %v", serv_port)
	log.Info().Msgf("Read consul address: %v", result["consulAddress"])
	log.Info().Msgf("Read jaeger address: %v", result["jaegerAddress"])
	var (
		// port       = flag.Int("port", 8083, "Server port")
		jaegeraddr = flag.String("jaegeraddr", result["jaegerAddress"], "Jaeger address")
	)
	flag.Parse()

	log.Info().Msgf("Initializing jaeger agent [service name: %v | host: %v]...", "geo", *jaegeraddr)

	tracer, err := tracing.Init("geo", *jaegeraddr)
	if err != nil {
		log.Panic().Msgf("Got error while initializing jaeger agent: %v", err)
	}
	log.Info().Msg("Jaeger agent initialized")

	srv := &geo.Server{
		// Port:     *port,
		Port:        serv_port,
		IpAddr:      serv_ip,
		Tracer:      tracer,
		MongoClient: mongo_client,
	}

	log.Info().Msg("Starting server...")
	log.Fatal().Msg(srv.Run().Error())
}
