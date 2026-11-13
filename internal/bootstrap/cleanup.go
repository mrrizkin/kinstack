package bootstrap

import "kinstack/internal/pkg/assert"

func (d *appDependencies) close() {
	for _, closer := range d.closers {
		assert.NoError(closer.Close())
	}
}
