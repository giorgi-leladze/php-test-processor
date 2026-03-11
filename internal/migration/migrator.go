package migration

// Migrator runs database migrations
type Migrator interface {
	Run(workerCount int, fresh bool) error
}

