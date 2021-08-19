package scheduler

import "github.com/da-moon/soil/manifest"

type RegistryConsumer interface {
	ConsumeRegistry(payload manifest.PodSlice)
}
