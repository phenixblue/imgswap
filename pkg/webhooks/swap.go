package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"twr.dev/imgswap/api/v1alpha1"
	"twr.dev/imgswap/pkg/mapstore"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/go-containerregistry/pkg/name"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	DefaultTypeString = "default"
	ExactTypeString   = "exact"
	ReplaceTypeString = "replace"
)

var (
	// swapmaplog is for logging in this package.
	swapmaplog = logf.Log.WithName("pod-imgswap-webhook").V(1)
)

// ImageSwapConfig is a struct that holds the configuration for the ImageSwap webhook
type ImageSwapConfig struct {
	namespace    string
	pod          string
	disableLabel string
	mode         string
}

// PodImageSwapHandler is a struct that holds the configuration for the ImageSwap webhook Handler
type PodImageSwapHandler struct {
	Client   client.Client
	Decoder  *admission.Decoder
	MapStore *mapstore.MapStore
}

// Check if our PodImageSwapper implements necessary interface
var _ admission.Handler = &PodImageSwapHandler{}

// +kubebuilder:webhook:path="/pod-imgswap",mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=swap.imgswap.io
func (piswh *PodImageSwapHandler) Handle(ctx context.Context, req admission.Request) admission.Response {

	// Validate Handler components are configured properly
	if piswh.Client == nil {
		return admission.Errored(http.StatusInternalServerError, errors.New("misconfigured Admission client"))
	}

	if piswh.Decoder == nil {
		return admission.Errored(http.StatusInternalServerError, errors.New("misconfigured Admission decoder"))
	}

	iswConfig := ImageSwapConfig{
		namespace:    "imgswap-system",
		pod:          "imgswap-pod",
		disableLabel: "k8s.twr.io/imageswap",
		mode:         "crd",
	}

	podOrig := &corev1.Pod{}
	err := piswh.Decoder.Decode(req, podOrig)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	podPatched := podOrig.DeepCopy()

	// Begin ImageSwap logic ----------------------------------------------------------------------------------------//

	workloadName := ""
	workloadType := req.Kind.Kind
	needsPatch := false

	// Detect if "name" in ObjectMeta is set
	//  this was added because the request object for pods doesn't
	// include a "name" field in the object metadata. This is because generateName
	// occurs Server Side post-admission

	if podOrig.ObjectMeta.Name != "" {
		workloadName = podOrig.ObjectMeta.Name
	} else if podOrig.ObjectMeta.Name == "" {
		workloadName = podOrig.ObjectMeta.GenerateName

	} else {
		workloadName = string(string(podOrig.ObjectMeta.UID))
	}

	// Skip swapping if Pod has disable label and value is "disabled"
	val, ok := podOrig.ObjectMeta.Labels[iswConfig.disableLabel]
	if ok && val == "disabled" {
		swapmaplog.Info("Skipping ImageSwap for Pod", "name", podOrig.ObjectMeta.Name)
	} else {
		if workloadType == "Pod" {
			for _, container := range podPatched.Spec.Containers {
				swapmaplog.Info("Processing Container", "pod", workloadName, "container", container.Name, "image", container.Image)
				needsPatch = swapImage(&container, *piswh.MapStore) || needsPatch
			}

			for _, container := range podPatched.Spec.InitContainers {
				swapmaplog.Info("Processing Init Container", "pod", workloadName, "container", container.Name, "image", container.Image)
				needsPatch = swapImage(&container, *piswh.MapStore) || needsPatch
			}
		} else {
			swapmaplog.Info("Invalid workload type", "type", workloadType)
			return admission.Allowed("Invalid workload type")
		}

	}

	if needsPatch {
		swapmaplog.V(5).Info("Needs patch", "pod", workloadName)

		marshaledPatchedPod, err := json.Marshal(podPatched)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

		admissionResponse := admission.PatchResponseFromRaw(req.Object.Raw, marshaledPatchedPod)
		swapmaplog.V(5).Info("Admission Response", "response", admissionResponse)
		return admissionResponse
	} else {
		swapmaplog.V(5).Info("Doesn't need patch", "pod", workloadName)
		admission.Allowed("ImageSwap - Doesn't need patch")
	}

	// End ImageSwap logic ------------------------------------------------------------------------------------------//

	marshaledPod, err := json.Marshal(podOrig)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (pisw *PodImageSwapHandler) InjectDecoder(d *admission.Decoder) error {
	pisw.Decoder = d

	fmt.Printf("\nInside Decoder: %v\n", pisw.Decoder)
	return nil
}

// swapImage is a function that performs an imageswap for a container spec
func swapImage(container *corev1.Container, swapMaps mapstore.MapStore) bool {

	var newImage string
	oldImage := container.Image

	parsedImage, err := name.ParseReference(container.Image)
	if err != nil {
		err = fmt.Errorf("error parsing image: %v", err)
		swapmaplog.Error(err, "Error parsing image")
		return false
	}

	// Check for match in swapMaps
	matchedMap, found := findMap(parsedImage, swapMaps)

	if found {
		// Swap image
		newImage, err = matchedMap.GetSwap(swapmaplog, parsedImage)
		if err != nil {
			err = fmt.Errorf("error swapping image: %v", err)
			swapmaplog.Error(err, "Error swapping image")
			return false
		}

		// Update container image
		if matchedMap != nil {
			fmt.Printf("/nBefore container image update: %v\n", container.Image)
			container.Image = newImage
			fmt.Printf("After container image update: %v\n", container.Image)
		}
	} else {
		swapmaplog.Info("No Map found")
		return false
	}

	swapmaplog.Info("Swapped image", "old", oldImage, "new", newImage)

	return true
}

// splitImage is a function that splits an image string into a SwapRef
func splitImage(image string) (v1alpha1.SwapRef, error) {
	//newImage := ""
	//isLibraryImage := false
	newSwapRef := v1alpha1.SwapRef{}
	imageRegistry := ""
	imagePath := ""
	imageTag := ""

	swapmaplog.Info("Got to image split")

	parsedImage, err := name.ParseReference(image)
	if err != nil {
		err := fmt.Errorf("error parsing image: %v", err)
		return v1alpha1.SwapRef{}, err
	}
	imageRegistry = parsedImage.Context().RegistryStr()
	imagePath = parsedImage.Context().RepositoryStr()
	imageTag = parsedImage.Identifier()
	imageProjectSections := strings.Split(imagePath, "/")

	newSwapRef.Registry = imageRegistry
	newSwapRef.Project = strings.Join(imageProjectSections[0:len(imageProjectSections)-1], "/")
	newSwapRef.Image = imageProjectSections[len(imageProjectSections)-1]
	newSwapRef.Tag = imageTag

	swapmaplog.Info("Image Registry", "registry", newSwapRef.Registry, "project", newSwapRef.Project, "image", newSwapRef.Image, "tag", newSwapRef.Tag)

	return newSwapRef, nil
}

// findMap is a function that finds a Map in a MapStore based on a specified image string
func findMap(image name.Reference, swapMaps mapstore.MapStore) (*v1alpha1.Map, bool) {
	matchedMap := &v1alpha1.Map{}
	found := false

	swapmaplog.Info("Finding Map", "image-context-name", image.Context().Name())

	// Split image string into SwapRef
	imageRef, err := splitImage(image.String())
	if err != nil {
		swapmaplog.Error(err, "Error splitting image")
		return matchedMap, found
	}

	// Check for exact image match in swapMaps
	if tmpMap, ok := swapMaps.Get(image.String()); ok {
		found = true
		matchedMap = tmpMap
		swapmaplog.Info("Found exact mapping", "image", image.String())
		return matchedMap, found
		// Check for Image match in swapMaps
	} else if tmpMap, ok := swapMaps.Get(imageRef.Registry + "/" + imageRef.Project + "/" + imageRef.Image); ok {
		found = true
		matchedMap = tmpMap
		swapmaplog.Info("Found Image mapping", "image", image.String())
		return matchedMap, found
		// Check for Project match in swapMaps
	} else if tmpMap, ok := swapMaps.Get(imageRef.Registry + "/" + imageRef.Project); ok {
		found = true
		matchedMap = tmpMap
		swapmaplog.Info("Found Project mapping", "image", image.String())
		return matchedMap, found
		// Check for Registry only match in swapMaps
	} else if tmpMap, ok := swapMaps.Get(imageRef.Registry); ok {
		found = true
		matchedMap = tmpMap
		swapmaplog.Info("Found Registry mapping", "image", image.String())
		return matchedMap, found
		// Check if port in Registry and check for No port in swapMaps
	} else if strings.Contains(imageRef.Image, ":") {
		registryNoPort := strings.Split(imageRef.Registry, ":")[0]
		if tmpMap, ok := swapMaps.Get(registryNoPort); ok {
			found = true
			matchedMap = tmpMap
			swapmaplog.Info("Found Registry mapping", "image", image.String())
			return matchedMap, found
		}
	}

	return matchedMap, found

}
