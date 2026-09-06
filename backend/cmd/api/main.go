package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/config"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/httpserver"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/postgres"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/ratelimit"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/recovery"
)

func main(){
	cfg,err:=config.Load(); if err!=nil { slog.Error("invalid configuration","error",err); os.Exit(1) }
	startupCtx,cancel:=context.WithTimeout(context.Background(),10*time.Second); defer cancel()
	pool,err:=postgres.Open(startupCtx,cfg.DatabaseURL); if err!=nil { slog.Error("database unavailable","error",err); os.Exit(1) }; defer pool.Close()
	accountService,err:=account.NewService(postgres.NewAccountStore(pool),password.Params{MemoryKiB:cfg.ArgonMemoryKiB,Iterations:cfg.ArgonIterations,Parallel:cfg.ArgonParallel,SaltBytes:16,KeyBytes:32},cfg.SessionTTL); if err!=nil { slog.Error("account service init failed","error",err); os.Exit(1) }
	if cfg.RecoverySMTPAddress!="" {
		notifier,err:=recovery.NewSMTP(recovery.SMTPConfig{Address:cfg.RecoverySMTPAddress,Host:cfg.RecoverySMTPHost,Username:cfg.RecoverySMTPUsername,Password:cfg.RecoverySMTPPassword,From:cfg.RecoveryFrom,ResetURL:cfg.RecoveryResetURL,ImplicitTLS:cfg.RecoveryImplicitTLS,Timeout:10*time.Second}); if err!=nil { slog.Error("password recovery init failed","error",err); os.Exit(1) }
		if err:=accountService.ConfigureRecovery(notifier,cfg.PasswordResetTTL); err!=nil { slog.Error("password recovery service init failed","error",err); os.Exit(1) }
		slog.Info("password recovery enabled","smtp_host",cfg.RecoverySMTPHost)
	}
	authLimiter,err:=ratelimit.New(ratelimit.Config{Limit:cfg.AuthRateLimit,Window:cfg.AuthRateWindow,IdleTTL:cfg.AuthRateIdleTTL,MaxEntries:cfg.AuthRateMaxEntries}); if err!=nil { slog.Error("auth rate limiter init failed","error",err); os.Exit(1) }
	app:=httpserver.New(httpserver.Dependencies{Accounts:accountService,Ready:pool.Ping,AuthLimiter:authLimiter})
	srv:=&http.Server{Addr:cfg.HTTPAddr,Handler:app.Handler(),ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:15*time.Second,IdleTimeout:60*time.Second}
	ctx,stop:=signal.NotifyContext(context.Background(),syscall.SIGINT,syscall.SIGTERM); defer stop()
	errCh:=make(chan error,1); go func(){ slog.Info("linkup api listening","addr",cfg.HTTPAddr); if err:=srv.ListenAndServe(); err!=nil && !errors.Is(err,http.ErrServerClosed){ errCh<-err } }()
	select{ case <-ctx.Done(): case err:=<-errCh: slog.Error("http server failed","error",err); os.Exit(1) }
	shutdownCtx,shutdownCancel:=context.WithTimeout(context.Background(),10*time.Second); defer shutdownCancel(); if err:=srv.Shutdown(shutdownCtx); err!=nil { slog.Error("graceful shutdown failed","error",err); os.Exit(1) }
}
