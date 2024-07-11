package main

import (
	"community-robot-lib/framework"
	"community-robot-lib/logrusutil"
	liboptions "community-robot-lib/options"
	"community-robot-lib/secret"
	"flag"
	sdk "git-platform-sdk"
	sig "github.com/opensourceways/robot-sig-info-cache"
	"github.com/sirupsen/logrus"
	"net/url"
	"time"
)

type options struct {
	service liboptions.ServiceOptions
	client  liboptions.ClientOptions
}

func (o *options) Validate() error {
	if _, err := url.ParseRequestURI(o.client.CacheEndpoint); err != nil {
		return err
	}

	if err := o.service.Validate(); err != nil {
		return err
	}

	return o.client.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options

	o.client.AddFlags(fs)
	o.service.AddFlags(fs)
	fs.StringVar(&o.client.CacheEndpoint, "cache-endpoint", "", "The endpoint of repo file cache")
	fs.IntVar(&o.client.CacheMaxRetries, "max-retries", 3, "The number of failed retry attempts to call the cache api")

	_ = fs.Parse(args)
	return o
}

func main() {
	logrusutil.ComponentInit(botName)

	//o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	o := options{
		service: liboptions.ServiceOptions{
			Port:        8833,
			ConfigFile:  "D:\\Project\\github\\ibfru\\atomgit-bot\\robot-atomgit-openeuler-welcome\\local\\config.yaml",
			GracePeriod: 300 * time.Second,
		},
		client: liboptions.ClientOptions{
			TokenPath:       "D:\\Project\\github\\ibfru\\atomgit-bot\\robot-atomgit-openeuler-welcome\\local\\token",
			RepoCacheDir:    "",
			CacheRepoOnPV:   true,
			CacheEndpoint:   "http://localhost:8888/v1/file",
			CacheMaxRetries: 1,
		},
	}
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	secretAgent := new(secret.Agent)
	if err := secretAgent.Start([]string{o.client.TokenPath}); err != nil {
		logrus.WithError(err).Fatal("Error starting secret agent.")
	}

	defer secretAgent.Stop()

	p := newRobot(sdk.GetClientInstance(secretAgent.GetSecret(o.client.TokenPath)), sig.NewSDK(o.client.CacheEndpoint, o.client.CacheMaxRetries))

	framework.Run(p, o.service, o.client)
}
