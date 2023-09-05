package mapstore

import (
	"fmt"
	"sync"

	mapsv1alpha1 "twr.dev/imgswap/api/v1alpha1"
)

type mapStore interface {
	// New creates a new MapStore
	New() (*mapsv1alpha1.SwapMapList, error)
	// Get returns the SwapMap with the given name from the MapStore
	Get(name string) (bool, *mapsv1alpha1.SwapMap)
	// AddOrUpdate adds or updates an existing SwapMap in the MapStore
	AddOrUpdate(mapSpec *mapsv1alpha1.SwapMap) error
	// Delete deletes an existing SwapMap from the MapStore
	Delete(mapName string) error
}

type MapStore struct {
	maps map[string]*mapsv1alpha1.Map
}

func (m *MapStore) New() (*mapsv1alpha1.SwapMapList, error) {
	return &mapsv1alpha1.SwapMapList{}, nil
}

func (m *MapStore) Get(name string) (bool, *mapsv1alpha1.Map) {
	mapSpec, ok := m.maps[name]
	return ok, mapSpec
}

func (m *MapStore) AddOrUpdate(mapKey string, mapSpec *mapsv1alpha1.Map) error {
	m.maps[mapKey] = mapSpec
	return nil
}

func (m *MapStore) Delete(mapName string) error {
	delete(m.maps, mapName)
	return nil
}

func NewMapStore() *MapStore {
	var once sync.Once
	var ms *MapStore

	once.Do(func() {
		ms = &MapStore{
			maps: make(map[string]*mapsv1alpha1.Map),
		}
	})
	return ms
}

func GetMapKey(mapSpec mapsv1alpha1.Map) (string, error) {
	mapKey := ""

	fmt.Printf("\nRegistry: %v, Project: %v, Image: %v\n", mapSpec.SwapFrom.Registry, mapSpec.SwapFrom.Project, mapSpec.SwapFrom.Image)

	if mapSpec.Name == "default" && mapSpec.Type == "default" {
		mapKey += "default"
		return mapKey, nil
	}

	if mapSpec.SwapFrom.Registry != "" {
		mapKey += mapSpec.SwapFrom.Registry
	}

	if mapSpec.SwapFrom.Project != "" {
		mapKey += "/" + mapSpec.SwapFrom.Project
	}

	if mapSpec.SwapFrom.Image != "" {
		mapKey += "/" + mapSpec.SwapFrom.Image
	}

	if mapKey == "" {
		return "", fmt.Errorf("unable to generate map key")
	} else {
		return mapKey, nil
	}
}
