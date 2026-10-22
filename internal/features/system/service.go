package system

import "kinstack/internal/pkg/assert"

type ServiceDependency struct {
	Repository Repository
}

type Service struct {
	repository Repository
}

func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "system repository is required")

	return &Service{repository: dependency.Repository}
}

func (s *Service) NotFound() NotFoundPageProps {
	result := s.repository.NotFound()
	return NotFoundPageProps(result)
}

// Page returns the public system page payload for the requested surface.
func (s *Service) Page(surface string) PageProps {
	result := s.repository.Page(surface)
	shortcuts := make([]Shortcut, 0, len(result.Shortcuts))
	for _, shortcut := range result.Shortcuts {
		shortcuts = append(shortcuts, Shortcut(shortcut))
	}
	return PageProps{
		Surface:   result.Surface,
		Topics:    append([]string(nil), result.Topics...),
		Shortcuts: shortcuts,
	}
}
