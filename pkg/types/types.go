package types

type TaskBody struct {
	TaskType  TaskType    `json:"taskType"`
	ChaosKind string      `json:"chaosKind"`
	ChaosBody interface{} `json:"chaosBody"`
}

type ChaosMeshPluginBody struct {
	TaskBody *TaskBody `json:"chaosMesh"`
}

type TaskType string

const (
	TaskTypeInject  TaskType = "inject"
	TaskTypeRecover TaskType = "recover"
)
