package bot

import (
	"sync"

	"github.com/jonas747/yagpdb/common"
)

// FeatureFlag represents a feature that's enabled or disabled on a server
// Instead of caching whole configs in memory and re-fetching them at intervals
type FeatureFlag struct {
	Name              string
	Key               interface{}
	Plugin            common.Plugin
	fetchGuildEnabled func(guildID int64) (bool, error)
	guildStatuses     map[int64]bool
}

var (
	registeredFeatureFlags map[interface{}]*FeatureFlag

	guildFeatureFlags   = make(map[int64]map[interface{}]FeatureFlag, nil)
	guildFeatureFlagsmu sync.Mutex
)

// GetFeatureFlagEnabled returns wether the feature flag, key, is enabled
func GetFeatureFlagEnabled(guildID int64, key interface{}) (bool, error) {

}
