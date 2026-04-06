package indexer

// SetSnapshotForTest replaces the current snapshot. Intended for use in tests only.
func (i *Indexer) SetSnapshotForTest(s *Snapshot) {
	i.snapshot.Store(s)
}

// SetDiscoveryCacheForTest replaces the current discovery cache. Intended for use in tests only.
func (i *Indexer) SetDiscoveryCacheForTest(c *APIDiscoveryCache) {
	i.discoveryCache.Store(c)
}
