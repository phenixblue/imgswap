package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var swapmaplog = logf.Log.WithName("pod-imgswap-webhook")

type PodImageSwapper struct {
	Client  client.Client
	decoder *admission.Decoder
}

// +kubebuilder:webhook:path="/pod-imgswap",mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=swap.imgswap.io
func (pisw *PodImageSwapper) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := pisw.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// mutate the fields in pod
	swapmaplog.Info("Mutating pod", "name", pod.Name)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (pisw *PodImageSwapper) InjectDecoder(d *admission.Decoder) error {
	pisw.decoder = d
	return nil
}
