package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	SessionTTL      time.Duration
	MigrationDir    string
	ArgonMemoryKiB  uint32
	ArgonIterations uint32
	ArgonParallel   uint8
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        envOr("LINKUP_HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		SessionTTL:      30 * 24 * time.Hour,
		MigrationDir:    envOr("LINKUP_MIGRATIONS_DIR", "../db/migrations"),
		ArgonMemoryKiB:  19 * 1024,
		ArgonIterations: 2,
		ArgonParallel:   1,
	}
	if cfg.DatabaseURL == "" { return Config{}, errors.New("DATABASE_URL is required") }
	var err error
	if cfg.SessionTTL, err = durationEnv("LINKUP_SESSION_TTL", cfg.SessionTTL); err != nil { return Config{}, err }
	if cfg.ArgonMemoryKiB, err = uint32Env("LINKUP_ARGON_MEMORY_KIB", cfg.ArgonMemoryKiB); err != nil { return Config{}, err }
	if cfg.ArgonIterations, err = uint32Env("LINKUP_ARGON_ITERATIONS", cfg.ArgonIterations); err != nil { return Config{}, err }
	parallel, err := uint32Env("LINKUP_ARGON_PARALLELISM", uint32(cfg.ArgonParallel)); if err != nil || parallel > 255 { if err==nil { err=errors.New("LINKUP_ARGON_PARALLELISM must be <=255") }; return Config{}, err }
	cfg.ArgonParallel=uint8(parallel)
	return cfg, nil
}

func envOr(key, fallback string) string { if v:=os.Getenv(key); v!="" { return v }; return fallback }

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v:=os.Getenv(key); if v=="" { return fallback,nil }
	if seconds,err:=strconv.ParseInt(v,10,64); err==nil && seconds>0 { return time.Duration(seconds)*time.Second,nil }
	d,err:=time.ParseDuration(v); if err!=nil || d<=0 { return 0,fmt.Errorf("%s must be a positive duration or integer seconds",key) }
	return d,nil
}

func uint32Env(key string, fallback uint32) (uint32,error) {
	v:=os.Getenv(key); if v=="" { return fallback,nil }
	n,err:=strconv.ParseUint(v,10,32); if err!=nil || n==0 { return 0,fmt.Errorf("%s must be a positive integer",key) }
	return uint32(n),nil
}
