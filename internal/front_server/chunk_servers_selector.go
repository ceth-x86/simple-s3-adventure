package front_server

// selectFirstNChunkServers selects the first n chunk servers
func (f *FrontServer) selectFirstNChunkServers(n int) []string {
	if len(f.chunkServers) < n {
		return nil
	}

	f.muServers.RLock()
	defer f.muServers.RUnlock()

	chunkServers := make([]string, 0, n)
	for server := range f.chunkServers {
		chunkServers = append(chunkServers, server)
		if len(chunkServers) == n {
			break
		}
	}

	return chunkServers
}
