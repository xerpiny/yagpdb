package featureflag

import (
	"emperror.dev/errors"
	"sync"

	"github.com/jonas747/yagpdb/common"
)

// FeatureFlag represents a feature that's enabled or disabled on a server
// A featureflag is designed to be lightweight and easily cacheable
//
// The goal of feature flags is to primarily reduce database load
//
// An example could be the logging configs
// instead of keeping all the logging settings in memory, you could have a flags for the most important ones
// (username or nickname logging enabled)
// which will be cached in memory forever
type FeatureFlag struct {
	Name              string
	Key               interface{}
	Plugin            common.Plugin
	fetchGuildEnabled func(guildID int64) (bool, error)
	guildStatuses     map[int64]bool
}

// LoadedFlags represents a loaded or partially loaded feature flag
type LoadedFlags int16

const (
	// LoadedFlagEnabled is the enabled status itself
	LoadedFlagEnabled LoadedFlags = 1 << iota

	// LoadedFlagFetching represents a flag being fetched from the database
	LoadedFlagFetching
)

var (
	registeredFeatureFlags map[interface{}]*FeatureFlag

	loadedGuildFlags       = make(map[int64]map[interface{}]LoadedFlags)
	loadedFeatureFlagsCond = sync.NewCond(&sync.Mutex{})
)

// GetFeatureFlagEnabled returns wether the feature flag, key, is enabled
func GetFeatureFlagEnabled(guildID int64, key interface{}) (bool, error) {
	loadedFeatureFlagsCond.L.Lock()
	defer loadedFeatureFlagsCond.L.Unlock()

	// init the guild store
	guildStore, ok := loadedGuildFlags[guildID]
	if !ok {
		guildStore = make(map[interface{}]LoadedFlags)
		loadedGuildFlags[guildID] = guildStore
	}

	for {

		flags, ok := guildStore[key]
		if ok {
			if flags&LoadedFlagFetching != 0 {
				// the flag is still being fetched from the database
				loadedFeatureFlagsCond.Wait()
				continue
			}

			// the flag is fully loaded
			return flags&LoadedFlagEnabled != 0, nil
		}

		// flag is not yet fetched from the database
		guildStore[key] = LoadedFlagFetching

		// fetch from underlying db
		return fetchFlag(guildID, key, guildStore)
	}

	return false, nil
}

func fetchFlag(guildID int64, key interface{}, guildStore map[interface{}]LoadedFlags) (bool, error) {

	// unlock while we fetch from the underlying database to allow other work while doing database stuff
	loadedFeatureFlagsCond.L.Unlock()

	// in case we panic during the fetching process, have the flag be deleted
	deleteFlag := true
	defer func() {
		if !deleteFlag {
			return
		}

		loadedFeatureFlagsCond.L.Lock()
		delete(guildStore, key)
		loadedFeatureFlagsCond.L.Unlock()

		// wake up all goroutines waiting
		loadedFeatureFlagsCond.Broadcast()
	}()

	flagDef := registeredFeatureFlags[key]
	enabled, err := flagDef.fetchGuildEnabled(guildID)
	if err != nil {
		return false, errors.WithStackIf(err)
	}

	deleteFlag = false
	f := LoadedFlags(0)
	if enabled {
		f = LoadedFlagEnabled
	}

	loadedFeatureFlagsCond.L.Lock()
	guildStore[key] = f
	loadedFeatureFlagsCond.L.Unlock()
	loadedFeatureFlagsCond.Broadcast()

	return enabled, nil
}
