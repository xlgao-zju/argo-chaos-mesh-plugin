package main

import (
	"fmt"
	"net/http"

	"argo-chaos-mesh-plugin/controller"
	chaosmesh "argo-chaos-mesh-plugin/pkg/chaos-mesh"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

var port int

func runServer() *cobra.Command {
	rootCmd := cobra.Command{
		Use:   "server",
		Short: "argo chaos mesh plugin",
		Long:  `a argo step that can run a chaos mesh experiment`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("### Listen on: ", port)
			return runPlugin()
		},
	}
	rootCmd.Flags().IntVarP(&port, "port", "p", 8443,
		"the port used by argo chaosmesh plugin.")
	return &rootCmd
}

func runPlugin() error {
	client, err := chaosmesh.NewClient()
	if err != nil {
		klog.Error("### failed to create chaos mesh client" + err.Error())
		return fmt.Errorf("failed to create chaos mesh client, err %v", err)
	}
	ct := &controller.Controller{
		ChaosClient: client,
	}
	//gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.POST("/api/v1/template.execute", ct.ExecuteChaosMeshExperience)
	return router.Run(fmt.Sprintf(":%d", port))
}
