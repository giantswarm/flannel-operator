package main

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/microkit/command"
	microserver "github.com/giantswarm/microkit/server"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"

	"github.com/giantswarm/flannel-operator/flag"
	"github.com/giantswarm/flannel-operator/pkg/project"
	"github.com/giantswarm/flannel-operator/server"
	"github.com/giantswarm/flannel-operator/service"
)

var (
	f = flag.New()
)

func main() {
	err := mainE(context.Background())
	if err != nil {
		panic(microerror.JSON(err))
	}
}

func mainE(ctx context.Context) error {
	var err error

	// Create a new logger which is used by all packages.
	var newLogger micrologger.Logger
	{
		newLogger, err = micrologger.New(micrologger.Config{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// We define a server factory to create the custom server once all command
	// line flags are parsed and all microservice configuration is storted out.
	newServerFactory := func(v *viper.Viper) microserver.Server {
		// Create a new custom service which implements business logic.
		var newService *service.Service
		{
			serviceConfig := service.Config{
				Logger: newLogger,

				Flag:  f,
				Viper: v,
			}

			newService, err = service.New(serviceConfig)
			if err != nil {
				panic(microerror.JSON(err))
			}
			go newService.Boot()
		}

		// Create a new custom server which bundles our endpoints.
		var newServer microserver.Server
		{
			c := server.Config{
				Logger:  newLogger,
				Service: newService,
				Viper:   v,

				ProjectName: project.Name(),
			}

			newServer, err = server.New(c)
			if err != nil {
				panic(microerror.JSON(err))
			}
		}

		return newServer
	}

	// Create a new microkit command which manages our custom microservice.
	var newCommand command.Command
	{
		c := command.Config{
			Logger:        newLogger,
			ServerFactory: newServerFactory,

			Description:    project.Description(),
			GitCommit:      project.GitSHA(),
			Name:           project.Name(),
			Source:         project.Source(),
			Version:        project.Version(),
			VersionBundles: service.NewVersionBundles(),
		}

		newCommand, err = command.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	daemonCommand := newCommand.DaemonCommand().CobraCommand()

	daemonCommand.PersistentFlags().String(f.Service.CRD.LabelSelector, "", "Label selector for CRD informer ListOptions.")

	daemonCommand.PersistentFlags().StringSlice(f.Service.Etcd.Endpoints, []string{"http://127.0.0.1:2379"}, "Endpoints used to connect to host's etcd.")
	daemonCommand.PersistentFlags().String(f.Service.Etcd.TLS.CAFile, "", "Certificate authority file path to use to authenticate with etcd.")
	daemonCommand.PersistentFlags().String(f.Service.Etcd.TLS.CrtFile, "", "Certificate file path to use to authenticate with etcd.")
	daemonCommand.PersistentFlags().String(f.Service.Etcd.TLS.KeyFile, "", "Key file path to use to authenticate with etcd.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.Address, "http://127.0.0.1:6443", "Address used to connect to Kubernetes. When empty in-cluster config is created.")
	daemonCommand.PersistentFlags().Bool(f.Service.Kubernetes.InCluster, false, "Whether to use the in-cluster config to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.KubeConfig, "", "KubeConfig used to connect to Kubernetes. When empty other settings are used.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CAFile, "", "Certificate authority file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CrtFile, "", "Certificate file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.KeyFile, "", "Key file path to use to authenticate with Kubernetes.")

	err = newCommand.CobraCommand().Execute()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
