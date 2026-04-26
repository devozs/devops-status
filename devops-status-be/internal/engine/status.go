package engine

import (
	"sync"

	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

// StatusEvaluator aggregates probe samples per target and drives incident transitions.
type StatusEvaluator struct {
	store           *store.Store
	incident        *IncidentManager
	targetEvalLocks sync.Map // string -> *sync.Mutex
}

func NewStatusEvaluator(s *store.Store, inc *IncidentManager) *StatusEvaluator {
	return &StatusEvaluator{store: s, incident: inc}
}

func (e *StatusEvaluator) withTargetLock(targetType string, targetID uuid.UUID, fn func()) {
	key := targetType + ":" + targetID.String()
	v, _ := e.targetEvalLocks.LoadOrStore(key, new(sync.Mutex))
	mu := v.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	fn()
}
