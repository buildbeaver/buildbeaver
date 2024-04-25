package scm

import (
	"fmt"
	"sync"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
)

type SCMRegistry struct {
	scmByName map[models.SystemName]SCM
	mutex     sync.RWMutex
}

func NewSCMRegistry() *SCMRegistry {
	return &SCMRegistry{
		scmByName: make(map[models.SystemName]SCM),
	}
}

// Register an SCM. If an SCM with that name is already registered then this function will panic.
func (s *SCMRegistry) Register(scm SCM) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.scmByName[scm.Name()]; ok {
		panic(fmt.Sprintf("SCMRegistry: attempt to register SCM %q more than once", scm.Name()))
	}

	s.scmByName[scm.Name()] = scm
}

// Get the registered SCM by name. If an SCM with the specified name does not
// exist an error will be returned.
func (s *SCMRegistry) Get(name models.SystemName) (SCM, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	scm, ok := s.scmByName[name]
	if !ok {
		return nil, gerror.NewErrNotFound("Not Found").IDetail("SCM", name)
	}
	return scm, nil
}
