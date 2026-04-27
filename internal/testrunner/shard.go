package testrunner

// ShardFiles returns the subset of files for the given shard.
// shardIndex is 1-based; shardTotal is the total number of shards.
func ShardFiles(files []string, shardIndex, shardTotal int) []string {
	if shardTotal <= 0 || shardIndex <= 0 || shardIndex > shardTotal {
		return files
	}
	var subset []string
	for i, f := range files {
		if (i % shardTotal) == (shardIndex - 1) {
			subset = append(subset, f)
		}
	}
	return subset
}
