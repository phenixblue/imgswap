package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"twr.dev/imgswap/api/v1alpha1"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	imgref "github.com/containers/image/docker/reference"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// swapmaplog is for logging in this package.
var swapmaplog = logf.Log.WithName("pod-imgswap-webhook")

// ImageSwapConfig is a struct that holds the configuration for the ImageSwap webhook
type ImageSwapConfig struct {
	namespace    string
	pod          string
	disableLabel string
	mode         string
}

// PodImageSwapHandler is a struct that holds the configuration for the ImageSwap webhook Handler
type PodImageSwapHandler struct {
	Client  client.Client
	Decoder *admission.Decoder
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
				swapmaplog.Info("Processing Container", "pod", workloadName, "container", container.Name)
				needsPatch = swapImage(&container) || needsPatch
			}

			for _, container := range podPatched.Spec.InitContainers {
				swapmaplog.Info("Processing Init Container", "pod", workloadName, "container", container.Name)
				needsPatch = swapImage(&container) || needsPatch
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
func swapImage(container *corev1.Container) bool {
	splitImage, err := splitImage(container.Image)
	if err != nil {
		swapmaplog.Error(err, "Error splitting image")
		return false
	}
	swapmaplog.Info("Split Image", "image", splitImage)
	return false
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

	parsedImage, err := imgref.ParseNormalizedNamed(image)
	if err != nil {
		err := fmt.Errorf("error parsing image: %v", err)
		return v1alpha1.SwapRef{}, err
	}

	//imageRef, err := imgref.Parse(image)
	if err != nil {
		err := fmt.Errorf("error parsing image: %v", err)
		return v1alpha1.SwapRef{}, err
	}

	imageRegistry = imgref.Domain(parsedImage)
	imagePath = imgref.Path(parsedImage)
	tagged, ok := parsedImage.(imgref.Tagged)
	if ok {
		imageTag = tagged.Tag()

	} else {
		imageTag = ""
	}
	swapmaplog.Info("Image Registry", "registry", imageRegistry, "path", imagePath, "tag", imageTag)

	newSwapRef.Registry = imageRegistry
	newSwapRef.Project = imagePath

	return newSwapRef, nil
}
