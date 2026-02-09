package migration

// Migrator runs database migrations
type Migrator interface {
	Run(workerCount int, noFresh bool) error
}

