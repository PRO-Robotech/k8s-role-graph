package indexer

// SetSnapshotForTest replaces the current snapshot. Intended for use in tests only.
func (i *Indexer) SetSnapshotForTest(s *Snapshot) {
	i.snapshot.Store(s)
}
