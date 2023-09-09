package mapstore

import (
	"fmt"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
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
	Maps map[string]*mapsv1alpha1.Map `json:"maps"`
}

func (m *MapStore) New() (*mapsv1alpha1.SwapMapList, error) {
	return &mapsv1alpha1.SwapMapList{}, nil
}

func (m *MapStore) Get(name string) (*mapsv1alpha1.Map, bool) {
	mapSpec, ok := m.Maps[name]
	return mapSpec, ok
}

func (m *MapStore) AddOrUpdate(mapKey string, mapSpec *mapsv1alpha1.Map) error {
	m.Maps[mapKey] = mapSpec
	return nil
}

func (m *MapStore) Delete(mapName string) error {
	delete(m.Maps, mapName)
	return nil
}

func NewMapStore() *MapStore {
	var once sync.Once
	var ms *MapStore

	once.Do(func() {
		ms = &MapStore{
			Maps: make(map[string]*mapsv1alpha1.Map),
		}
	})
	return ms
}

// GetMapKey is a method that reads a MapSpec and returns a string representing the image to be used as the key in the MapStore
// NOTE: We're using the "github.com/google/go-containerregistry/pkg/name" package to parse image strings into a consistent format.
// This will automatically change "docker.io" to "index.docker.io", insert "/library", etc.
// This will also validate the image string to ensure it is a valid reference.
// We should strive to use this for all internal image references.
// This may cause unintended side-effects, so we should be careful.
func GetMapKey(mapSpec mapsv1alpha1.Map) (string, error) {
	mapKey := ""

	fmt.Printf("\nRegistry: %v, Project: %v, Image: %v\n", mapSpec.SwapFrom.Registry, mapSpec.SwapFrom.Project, mapSpec.SwapFrom.Image)

	if mapSpec.Name == "default" && mapSpec.Type == "default" {
		mapKey += "default"
		return mapKey, nil
	}

	// Add Map Type to key if applicable
	if mapSpec.Type == "default" {
		if mapSpec.Name == "default" {
			mapKey += "default"
		}
	}

	mapKey = mapSpec.SwapFrom.ToString()

	if mapKey == "" {
		return "", fmt.Errorf("unable to generate map key")
	} else {

		// Verify the final mapKey is a valid reference
		parsedMapKey, err := name.ParseReference(mapKey)
		if err != nil {
			return "", fmt.Errorf("unable to parse map key: %v", err)
		}

		fmt.Printf("\nMap Key: %v\n", parsedMapKey.String())
		return parsedMapKey.String(), nil
	}
}
