// Package server implements the entry point for running a robot web server.
package server

import (
	"context"
	"net"
	"os"
	"runtime/pprof"
	"time"

	"github.com/edaniels/golog"
	"github.com/edaniels/gostream"
	"github.com/edaniels/gostream/codec/opus"
	"github.com/edaniels/gostream/codec/x264"
	"go.uber.org/multierr"
	"go.viam.com/utils"
	"go.viam.com/utils/perf"
	"go.viam.com/utils/rpc"

	"go.viam.com/rdk/config"
	robotimpl "go.viam.com/rdk/robot/impl"
	"go.viam.com/rdk/robot/web"
	weboptions "go.viam.com/rdk/robot/web/options"
)

// Arguments for the command.
type Arguments struct {
	AllowInsecureCreds         bool   `flag:"allow-insecure-creds,usage=allow connections to send credentials over plaintext"`
	ConfigFile                 string `flag:"config,usage=robot config file"`
	CPUProfile                 string `flag:"cpuprofile,usage=write cpu profile to file"`
	Debug                      bool   `flag:"debug"`
	SharedDir                  string `flag:"shareddir,usage=web resource directory"`
	Version                    bool   `flag:"version,usage=print version"`
	WebProfile                 bool   `flag:"webprofile,usage=include profiler in http server"`
	WebRTC                     bool   `flag:"webrtc,usage=force webrtc connections instead of direct"`
	revealSensitiveConfigDiffs bool   `flag:"reveal-sensitive-config-diffs,usage=show config diffs"`
}

// RunServer is an entry point to starting the web server that can be called by main in a code
// sample or otherwise be used to initialize the server.
func RunServer(ctx context.Context, args []string, logger golog.Logger) (err error) {
	var argsParsed Arguments
	if err := utils.ParseFlags(args, &argsParsed); err != nil {
		return err
	}

	// Always log the version, return early if the '-version' flag was provided
	// fmt.Println would be better but fails linting. Good enough.
	logger.Infof("Viam RDK Version: %s, Hash: %s", config.Version, config.GitRevision)
	if argsParsed.Version {
		return
	}

	if argsParsed.ConfigFile == "" {
		logger.Error("please specify a config file through the -config parameter.")
		return
	}

	if argsParsed.CPUProfile != "" {
		f, err := os.Create(argsParsed.CPUProfile)
		if err != nil {
			return err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}

	// Read the config from disk and use it to initialize the remote logger.
	initialReadCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	cfgFromDisk, err := config.ReadLocalConfig(initialReadCtx, argsParsed.ConfigFile, logger)
	if err != nil {
		cancel()
		return err
	}
	cancel()

	// Start remote logging with config from disk.
	// This is to ensure we make our best effort to write logs for failures loading the remote config.
	if cfgFromDisk.Cloud != nil && (cfgFromDisk.Cloud.LogPath != "" || cfgFromDisk.Cloud.AppAddress != "") {
		var closer func()
		logger, closer, err = addCloudLogger(logger, cfgFromDisk.Cloud)
		if err != nil {
			return err
		}
		defer closer()
	}

	// Run the server with remote logging enabled.
	err = runServerWithLogging(ctx, argsParsed, logger)
	if err != nil {
		logger.Error("Fatal error running server, existing now: ", err)
	}

	return err
}

// RunServer is an entry point to starting the web server after the local config is read. Once the local config
// is read the logger may be initialized to remote log. This ensure we capture errors starting up the server and report to the cloud.
func runServerWithLogging(ctx context.Context, argsParsed Arguments, logger golog.Logger) (err error) {
	exporter := perf.NewDevelopmentExporter()
	if err := exporter.Start(); err != nil {
		return err
	}
	defer exporter.Stop()

	initialReadCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	cfg, err := config.Read(initialReadCtx, argsParsed.ConfigFile, logger)
	if err != nil {
		cancel()
		return err
	}
	cancel()

	err = serveWeb(ctx, cfg, argsParsed, logger)
	if err != nil {
		logger.Errorw("error serving web", "error", err)
	}

	return err
}

func createWebOptions(cfg *config.Config, argsParsed Arguments, logger golog.Logger) (weboptions.Options, error) {
	options, err := weboptions.FromConfig(cfg)
	if err != nil {
		return weboptions.Options{}, err
	}
	options.Pprof = argsParsed.WebProfile
	options.SharedDir = argsParsed.SharedDir
	options.Debug = argsParsed.Debug || cfg.Debug
	options.WebRTC = argsParsed.WebRTC
	if cfg.Cloud != nil && argsParsed.AllowInsecureCreds {
		options.SignalingDialOpts = append(options.SignalingDialOpts, rpc.WithAllowInsecureWithCredentialsDowngrade())
	}

	if len(options.Auth.Handlers) == 0 {
		host, _, err := net.SplitHostPort(cfg.Network.BindAddress)
		if err != nil {
			return weboptions.Options{}, err
		}
		if host == "" || host == "0.0.0.0" || host == "::" {
			logger.Warn("binding to all interfaces without authentication")
		}
	}
	return options, nil
}

func serveWeb(ctx context.Context, cfg *config.Config, argsParsed Arguments, logger golog.Logger) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	rpcDialer := rpc.NewCachedDialer()
	defer func() {
		err = multierr.Combine(err, rpcDialer.Close())
	}()
	ctx = rpc.ContextWithDialer(ctx, rpcDialer)

	processConfig := func(in *config.Config) (*config.Config, error) {
		tlsCfg := config.NewTLSConfig(cfg)
		out, err := config.ProcessConfig(in, tlsCfg)
		if err != nil {
			return nil, err
		}
		out.Debug = argsParsed.Debug || cfg.Debug
		out.FromCommand = true
		out.AllowInsecureCreds = argsParsed.AllowInsecureCreds
		return out, nil
	}

	processedConfig, err := processConfig(cfg)
	if err != nil {
		return err
	}

	if processedConfig.Cloud != nil {
		utils.PanicCapturingGo(func() {
			restartCheck, err := newRestartChecker(ctx, cfg.Cloud, logger)
			if err != nil {
				logger.Panicw("error creating restart checker", "error", err)
				cancel()
				return
			}
			defer restartCheck.close()
			restartInterval := defaultNeedsRestartCheckInterval

			for {
				if !utils.SelectContextOrWait(ctx, restartInterval) {
					return
				}

				mustRestart, newRestartInterval, err := restartCheck.needsRestart(ctx)
				if err != nil {
					logger.Infow("failed to check restart", "error", err)
					continue
				}

				restartInterval = newRestartInterval

				if mustRestart {
					cancel()
					return
				}
			}
		})
	}

	var streamConfig gostream.StreamConfig
	streamConfig.AudioEncoderFactory = opus.NewEncoderFactory()
	streamConfig.VideoEncoderFactory = x264.NewEncoderFactory()

	robotOptions := []robotimpl.Option{robotimpl.WithWebOptions(web.WithStreamConfig(streamConfig))}
	if argsParsed.revealSensitiveConfigDiffs {
		robotOptions = append(robotOptions, robotimpl.WithRevealSensitiveConfigDiffs())
	}

	myRobot, err := robotimpl.New(ctx, processedConfig, logger, robotOptions...)
	if err != nil {
		return err
	}
	defer func() {
		err = multierr.Combine(err, myRobot.Close(context.Background()))
	}()

	// watch for and deliver changes to the robot
	watcher, err := config.NewWatcher(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer func() {
		err = multierr.Combine(err, utils.TryClose(ctx, watcher))
	}()
	onWatchDone := make(chan struct{})
	oldCfg := processedConfig
	utils.ManagedGo(func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			select {
			case <-ctx.Done():
				return
			case cfg := <-watcher.Config():
				processedConfig, err := processConfig(cfg)
				if err != nil {
					logger.Errorw("error processing config", "error", err)
					continue
				}
				myRobot.Reconfigure(ctx, processedConfig)

				// restart web service if necessary
				diff, err := config.DiffConfigs(*oldCfg, *processedConfig, argsParsed.revealSensitiveConfigDiffs)
				if err != nil {
					logger.Errorw("error diffing config", "error", err)
					continue
				}
				if !diff.NetworkEqual {
					if err := myRobot.StopWeb(); err != nil {
						logger.Errorw("error stopping web service while reconfiguring", "error", err)
						continue
					}
					options, err := createWebOptions(processedConfig, argsParsed, logger)
					if err != nil {
						logger.Errorw("error creating weboptions", "error", err)
						continue
					}
					if err := myRobot.StartWeb(ctx, options); err != nil {
						logger.Errorw("error starting web service while reconfiguring", "error", err)
					}
				}
				oldCfg = processedConfig
			}
		}
	}, func() {
		close(onWatchDone)
	})
	defer func() {
		<-onWatchDone
	}()
	defer cancel()

	options, err := createWebOptions(processedConfig, argsParsed, logger)
	return web.RunWeb(ctx, myRobot, options, logger)
}
