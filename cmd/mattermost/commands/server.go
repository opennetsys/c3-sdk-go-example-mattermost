// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package commands

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c3systems/c3-sdk-go-example-mattermost/api4"
	"github.com/c3systems/c3-sdk-go-example-mattermost/app"
	"github.com/c3systems/c3-sdk-go-example-mattermost/manualtesting"
	"github.com/c3systems/c3-sdk-go-example-mattermost/mlog"
	"github.com/c3systems/c3-sdk-go-example-mattermost/model"
	"github.com/c3systems/c3-sdk-go-example-mattermost/utils"
	"github.com/c3systems/c3-sdk-go-example-mattermost/web"
	"github.com/c3systems/c3-sdk-go-example-mattermost/wsapi"
	"github.com/spf13/cobra"
)

const (
	SESSIONS_CLEANUP_BATCH_SIZE = 1000
)

var MaxNotificationsPerChannelDefault int64 = 1000000

var serverCmd = &cobra.Command{
	Use:          "server",
	Short:        "Run the Mattermost server",
	RunE:         serverCmdF,
	SilenceUsage: true,
}

var a *app.App

func init() {
	RootCmd.AddCommand(serverCmd)
	RootCmd.RunE = serverCmdF
}

func serverCmdF(command *cobra.Command, args []string) error {
	log.Printf("in serverCmdf")
	config, err := command.Flags().GetString("config")
	if err != nil {
		return err
	}

	disableConfigWatch, _ := command.Flags().GetBool("disableconfigwatch")
	usedPlatform, _ := command.Flags().GetBool("platform")
	// note: we want the default to be to listen, hence the negative "NotListen" here...
	shouldNotListen, _ := command.Flags().GetBool("shouldNotListen")
	mlog.Info(fmt.Sprintf("should not listen: %v", shouldNotListen))

	interruptChan := make(chan os.Signal, 1)
	return runServer(config, disableConfigWatch, usedPlatform, interruptChan, !shouldNotListen)
}

func Serve(w http.ResponseWriter, r *http.Request) {
	a.Srv.Server.Handler.ServeHTTP(w, r)
}

func runServer(configFileLocation string, disableConfigWatch bool, usedPlatform bool, interruptChan chan os.Signal, shouldListen bool) error {
	log.Printf("in run server")
	var err error
	options := []app.Option{app.ConfigFile(configFileLocation)}
	if disableConfigWatch {
		options = append(options, app.DisableConfigWatch)
	}

	log.Println("app.New")
	a, err = app.New(options...)
	if err != nil {
		log.Printf("err in app.New")
		mlog.Critical(err.Error())
		return err
	}
	defer a.Shutdown()

	log.Println("test mail service")
	// mailservice.TestConnection(a.Config())

	log.Println("os.Getwd")
	pwd, _ := os.Getwd()
	if usedPlatform {
		mlog.Error("The platform binary has been deprecated, please switch to using the mattermost binary.")
	}

	log.Println("url.ParseRequestURI")
	if _, err := url.ParseRequestURI(*a.Config().ServiceSettings.SiteURL); err != nil {
		mlog.Error("SiteURL must be set. Some features will operate incorrectly if the SiteURL is not set. See documentation for details: http://about.mattermost.com/default-site-url")
	}

	mlog.Info(fmt.Sprintf("Current version is %v (%v/%v/%v/%v)", model.CurrentVersion, model.BuildNumber, model.BuildDate, model.BuildHash, model.BuildHashEnterprise))
	mlog.Info(fmt.Sprintf("Enterprise Enabled: %v", model.BuildEnterpriseReady))
	mlog.Info(fmt.Sprintf("Current working directory is %v", pwd))
	mlog.Info(fmt.Sprintf("Loaded config file from %v", utils.FindConfigFile(configFileLocation)))

	log.Println("fileBackend")
	backend, appErr := a.FileBackend()
	if appErr == nil {
		appErr = backend.TestConnection()
	}
	if appErr != nil {
		mlog.Error("Problem with file storage settings: " + appErr.Error())
	}

	if model.BuildEnterpriseReady == "true" {
		a.LoadLicense()
	}

	log.Println("migrations")
	a.DoAdvancedPermissionsMigration()
	a.DoEmojisPermissionsMigration()

	log.Println("plugins")
	a.InitPlugins(*a.Config().PluginSettings.Directory, *a.Config().PluginSettings.ClientDirectory)
	a.AddConfigListener(func(prevCfg, cfg *model.Config) {
		if *cfg.PluginSettings.Enable {
			a.InitPlugins(*cfg.PluginSettings.Directory, *a.Config().PluginSettings.ClientDirectory)
		} else {
			a.ShutDownPlugins()
		}
	})

	mlog.Info(fmt.Sprintf("\n\n\nhere\n\n\nShould listen: %v", shouldListen))
	serverErr := a.StartServer(shouldListen)
	if serverErr != nil {
		mlog.Critical(serverErr.Error())
		return serverErr
	}

	log.Println("api4 init")
	api := api4.Init(a, a.Srv.Router)
	wsapi.Init(a, a.Srv.WebSocketRouter)
	web.NewWeb(a, a.Srv.Router)

	log.Println("license")
	license := a.License()

	if license == nil && len(a.Config().SqlSettings.DataSourceReplicas) > 1 {
		mlog.Warn("More than 1 read replica functionality disabled by current license. Please contact your system administrator about upgrading your enterprise license.")
		a.UpdateConfig(func(cfg *model.Config) {
			cfg.SqlSettings.DataSourceReplicas = cfg.SqlSettings.DataSourceReplicas[:1]
		})
	}

	if license == nil {
		a.UpdateConfig(func(cfg *model.Config) {
			cfg.TeamSettings.MaxNotificationsPerChannel = &MaxNotificationsPerChannelDefault
		})
	}

	log.Println("reload config")
	a.ReloadConfig()

	// Enable developer settings if this is a "dev" build
	if model.BuildNumber == "dev" {
		a.UpdateConfig(func(cfg *model.Config) { *cfg.ServiceSettings.EnableDeveloper = true })
	}

	log.Println("reset uses")
	resetStatuses(a)

	// If we allow testing then listen for manual testing URL hits
	if a.Config().ServiceSettings.EnableTesting {
		manualtesting.Init(api)
	}

	//a.Go(func() {
	//runSecurityJob(a)
	//})
	//a.Go(func() {
	//runDiagnosticsJob(a)
	//})
	//a.Go(func() {
	//runSessionCleanupJob(a)
	//})
	//a.Go(func() {
	//runTokenCleanupJob(a)
	//})
	//a.Go(func() {
	//runCommandWebhookCleanupJob(a)
	//})

	//if complianceI := a.Compliance; complianceI != nil {
	//complianceI.StartComplianceDailyJob()
	//}

	//if a.Cluster != nil {
	//a.RegisterAllClusterMessageHandlers()
	//a.Cluster.StartInterNodeCommunication()
	//}

	//if a.Metrics != nil {
	//a.Metrics.StartServer()
	//}

	//if a.Elasticsearch != nil {
	//a.StartElasticsearch()
	//}

	//if *a.Config().JobSettings.RunJobs {
	//a.Jobs.StartWorkers()
	//defer a.Jobs.StopWorkers()
	//}
	//if *a.Config().JobSettings.RunScheduler {
	//a.Jobs.StartSchedulers()
	//defer a.Jobs.StopSchedulers()
	//}

	log.Println("\n\n\nREADY...\n\n\n")
	notifyReady()

	// wait for kill signal before attempting to gracefully shutdown
	// the running service
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interruptChan

	//if a.Cluster != nil {
	//a.Cluster.StopInterNodeCommunication()
	//}

	//if a.Metrics != nil {
	//a.Metrics.StopServer()
	//}

	return nil
}

func runSecurityJob(a *app.App) {
	doSecurity(a)
	model.CreateRecurringTask("Security", func() {
		doSecurity(a)
	}, time.Hour*4)
}

func runDiagnosticsJob(a *app.App) {
	doDiagnostics(a)
	model.CreateRecurringTask("Diagnostics", func() {
		doDiagnostics(a)
	}, time.Hour*24)
}

func runTokenCleanupJob(a *app.App) {
	doTokenCleanup(a)
	model.CreateRecurringTask("Token Cleanup", func() {
		doTokenCleanup(a)
	}, time.Hour*1)
}

func runCommandWebhookCleanupJob(a *app.App) {
	doCommandWebhookCleanup(a)
	model.CreateRecurringTask("Command Hook Cleanup", func() {
		doCommandWebhookCleanup(a)
	}, time.Hour*1)
}

func runSessionCleanupJob(a *app.App) {
	doSessionCleanup(a)
	model.CreateRecurringTask("Session Cleanup", func() {
		doSessionCleanup(a)
	}, time.Hour*24)
}

func resetStatuses(a *app.App) {
	if result := <-a.Srv.Store.Status().ResetAll(); result.Err != nil {
		mlog.Error(fmt.Sprint("Error to reset the server status.", result.Err.Error()))
	}
}

func doSecurity(a *app.App) {
	a.DoSecurityUpdateCheck()
}

func doDiagnostics(a *app.App) {
	if *a.Config().LogSettings.EnableDiagnostics {
		a.SendDailyDiagnostics()
	}
}

func notifyReady() {
	// If the environment vars provide a systemd notification socket,
	// notify systemd that the server is ready.
	systemdSocket := os.Getenv("NOTIFY_SOCKET")
	if systemdSocket != "" {
		mlog.Info("Sending systemd READY notification.")

		err := sendSystemdReadyNotification(systemdSocket)
		if err != nil {
			mlog.Error(err.Error())
		}
	}
}

func sendSystemdReadyNotification(socketPath string) error {
	msg := "READY=1"
	addr := &net.UnixAddr{
		Name: socketPath,
		Net:  "unixgram",
	}
	conn, err := net.DialUnix(addr.Net, nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(msg))
	return err
}

func doTokenCleanup(a *app.App) {
	a.Srv.Store.Token().Cleanup()
}

func doCommandWebhookCleanup(a *app.App) {
	a.Srv.Store.CommandWebhook().Cleanup()
}

func doSessionCleanup(a *app.App) {
	a.Srv.Store.Session().Cleanup(model.GetMillis(), SESSIONS_CLEANUP_BATCH_SIZE)
}
