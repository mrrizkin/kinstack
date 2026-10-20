package bootstrap

import "kinstack/internal/pkg/assert"

// close releases every repository the application owns.
func (d *appDependencies) close() {
	for _, closer := range d.closers {
		assert.NoError(closer.Close())
	}
}
