package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	chaosmesh "argo-chaos-mesh-plugin/pkg/chaos-mesh"
	"argo-chaos-mesh-plugin/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	executorplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	chaosmeshapi "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type Controller struct {
	ChaosClient chaosmesh.Client
}

func (ct *Controller) ExecuteChaosMeshExperience(ctx *gin.Context) {
	c := &executorplugins.ExecuteTemplateArgs{}
	if err := ctx.BindJSON(&c); err != nil {
		klog.Error(err)
		return
	}

	// Get workflow First
	inputBody := &types.ChaosMeshPluginBody{}
	pluginJson, err := c.Template.Plugin.MarshalJSON()
	if err != nil {
		klog.Error(err)
		ct.Response404(ctx)
		return

	}
	klog.Info("Receive: ", string(pluginJson))
	err = json.Unmarshal(pluginJson, inputBody)
	if err != nil {
		klog.Error(err)
		ct.Response404(ctx)
		return

	}

	chaosObj, err := ConvertToChaosObject(inputBody.TaskBody)
	if err != nil {
		msg := fmt.Sprintf("failed to convert inputBody.TaskBody to chaos object, err %v", err)
		klog.Error(msg)
		ct.ResponseMsg(ctx, wfv1.NodeFailed, msg)
		return
	}

	switch inputBody.TaskBody.TaskType {
	case types.TaskTypeInject:
		err = ct.InjectExperiment(ctx, inputBody.TaskBody.ChaosKind, chaosObj)
		if err != nil {
			klog.Error(err)
			ct.ResponseMsg(ctx, wfv1.NodeFailed, err.Error())
			return
		}
	case types.TaskTypeRecover:
		err = ct.RecoverExperiment(ctx, inputBody.TaskBody.ChaosKind, chaosObj)
		if err != nil {
			klog.Error(err)
			ct.ResponseMsg(ctx, wfv1.NodeFailed, err.Error())
			return
		}
	}
}

func (ct *Controller) InjectExperiment(ctx *gin.Context, kind string, chaos chaosmeshapi.InnerObject) error {
	// 1. query experiment exists
	exists := false
	chaosName := chaos.GetName()
	chaosNamespace := chaos.GetNamespace()
	experiment, err := ct.ChaosClient.GetExperiment(ctx, chaosNamespace, chaosName, kind)
	if err != nil {
		if errors.IsNotFound(err) {
			exists = false
		} else {
			klog.Errorf("failed to get chaos mesh experiment %s/%s, err %v", chaosNamespace, chaosName, err)
			return err
		}
	} else {
		exists = true
	}

	// 2. found and return
	if exists {
		klog.Infof("# found exists chaos mesh experiment: %s/%s returning Status...", chaosNamespace, chaosName)
		ct.ResponseWaitInjection(ctx, experiment)
		return nil
	}

	// 3. create experiment if not exists
	_, err = ct.ChaosClient.CreateExperiment(ctx, chaos)
	if err != nil {
		klog.Error("### " + err.Error())
		ct.ResponseMsg(ctx, wfv1.NodeFailed, err.Error())
		return err
	}
	ct.ResponseRequeue(ctx, types.TaskTypeInject)
	return nil
}

// RecoverExperiment recover chaos experiment
// since all chaos mesh object has finalizer(which used to recover the experiment),
// so if we can not find the object,
// it means the object has been deleted, and the recover process is done.
func (ct *Controller) RecoverExperiment(ctx *gin.Context, kind string, chaos chaosmeshapi.InnerObject) error {
	chaosName := chaos.GetName()
	chaosNamespace := chaos.GetNamespace()
	experiment, err := ct.ChaosClient.GetExperiment(ctx, chaosNamespace, chaosName, kind)
	if err != nil {
		if errors.IsNotFound(err) {
			ct.ResponseMsg(ctx, wfv1.NodeSucceeded, "recover success")
			return nil
		} else {
			klog.Errorf("failed to get chaos mesh experiment %s/%s, err %v", chaosNamespace, chaosName, err)
			return err
		}
	}

	if experiment.GetDeletionTimestamp() == nil {
		if err = ct.ChaosClient.DeleteExperiment(ctx, chaosNamespace, chaosName, kind); err != nil {
			klog.Errorf("failed to delete chaos mesh experiment %s/%s, err %v", chaosNamespace, chaosName, err)
			return err
		}
		ct.ResponseRequeue(ctx, types.TaskTypeRecover)
		return nil
	} else {
		deleteAt := experiment.GetDeletionTimestamp()
		if time.Now().Sub(deleteAt.Time).Seconds() > 30 {
			ct.ResponseMsg(ctx, wfv1.NodeFailed, "recover timeout after 30 seconds")
			return nil
		}
	}

	return nil
}

func (ct *Controller) ResponseRequeue(ctx *gin.Context, action types.TaskType) {
	ctx.JSON(http.StatusOK, &executorplugins.ExecuteTemplateReply{
		Node: &wfv1.NodeResult{
			Phase:   wfv1.NodePending,
			Message: fmt.Sprintf("chaos mesh experiment %s done", action),
			Outputs: nil,
		},
		Requeue: &metav1.Duration{
			Duration: 5 * time.Second,
		},
	})
}

func (ct *Controller) ResponseMsg(ctx *gin.Context, status wfv1.NodePhase, msg string) {
	ctx.JSON(http.StatusOK, &executorplugins.ExecuteTemplateReply{
		Node: &wfv1.NodeResult{
			Phase:   status,
			Message: msg,
			Outputs: nil,
		},
	})
}

func (ct *Controller) ResponseWaitInjection(ctx *gin.Context, experiment chaosmeshapi.InnerObject) {
	var status wfv1.NodePhase
	// check if timeout
	createAt := experiment.GetCreationTimestamp()
	if time.Now().Sub(createAt.Time).Seconds() > 30 {
		ct.ResponseMsg(ctx, wfv1.NodeFailed, "drill timeout after 30 seconds")
		return
	}

	var running, succeed int
	for _, record := range experiment.GetStatus().Experiment.Records {
		if record.InjectedCount > 0 {
			succeed++
		} else {
			running++
		}
	}
	if running > 0 {
		ctx.JSON(http.StatusOK, &executorplugins.ExecuteTemplateReply{
			Node: &wfv1.NodeResult{
				Phase:   status,
				Message: "still wait",
				Outputs: nil,
			},
			Requeue: &metav1.Duration{
				Duration: 5 * time.Second,
			},
		})
	}
	if succeed > 0 {
		ct.ResponseMsg(ctx, wfv1.NodeSucceeded, "injection complete")
	}
}

func (ct *Controller) Response404(ctx *gin.Context) {
	ctx.AbortWithStatus(http.StatusNotFound)

}

// ConvertToChaosObject convert ChaosMeshPluginBody to chaos object
func ConvertToChaosObject(body *types.TaskBody) (chaosmeshapi.InnerObject, error) {
	chaosKind, exists := chaosmeshapi.AllKinds()[body.ChaosKind]
	if !exists {
		return nil, fmt.Errorf("unknwon chaos kind '%s'", body.ChaosKind)
	}
	v, err := json.Marshal(body.ChaosBody)
	if err != nil {
		return nil, fmt.Errorf("failed marshao chaos body, %s", err.Error())
	}

	chaos := chaosKind.SpawnObject()
	if err = json.Unmarshal(v, chaos); err != nil {
		return nil, fmt.Errorf("failed unmarshal chaos body to object, %s", err.Error())
	}
	return chaos.(chaosmeshapi.InnerObject), nil
}
