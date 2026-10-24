package collector

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	attributionv1alpha1 "github.com/sanoguzhan/krato/api/v1alpha1"
)

func TestCollectRecordsAndDropsAttributions(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := attributionv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	custom := "custom"
	broken := testAttribution("broken", "payments")
	broken.Spec.MetricsBackend = &custom // not implemented, so its group fails

	search := testAttribution("search", "search")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testAttribution("payments", "payments"),
		search,
		broken,
		testPod("api-1", "payments", "250m", "128Mi", corev1.PodRunning),
		testPod("api-2", "payments", "250m", "128Mi", corev1.PodPending),
		testPod("job-1", "payments", "1", "1Gi", corev1.PodSucceeded), // finished, not counted
		testPod("idx-1", "search", "300m", "64Mi", corev1.PodRunning),
	).Build()

	l := &Loop{Client: c}
	l.collect(ctx)

	expectGauge(t, "krato_attribution_cpu_cores", "payments", 0.5)
	expectGauge(t, "krato_attribution_memory_bytes", "payments", 256*1024*1024)
	expectGauge(t, "krato_attribution_pods", "payments", 2)
	expectGauge(t, "krato_attribution_cpu_cores", "search", 0.3)
	expectGauge(t, "krato_attribution_memory_bytes", "search", 64*1024*1024)
	expectGauge(t, "krato_attribution_pods", "search", 1)
	expectNoGauge(t, "krato_attribution_cpu_cores", "broken")

	if err := c.Delete(ctx, search); err != nil {
		t.Fatal(err)
	}
	l.collect(ctx)

	expectGauge(t, "krato_attribution_pods", "payments", 2)
	expectNoGauge(t, "krato_attribution_cpu_cores", "search")
	expectNoGauge(t, "krato_attribution_memory_bytes", "search")
	expectNoGauge(t, "krato_attribution_pods", "search")
}

func testAttribution(name, team string) *attributionv1alpha1.ResourceAttribution {
	return &attributionv1alpha1.ResourceAttribution{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team"},
		Spec: attributionv1alpha1.ResourceAttributionSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": team}},
		},
	}
}

func testPod(name, team, cpu, memory string, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team", Labels: map[string]string{"team": team}},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "app",
			Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			}},
		}}},
		Status: corev1.PodStatus{Phase: phase},
	}
}

func expectGauge(t *testing.T, name, attribution string, want float64) {
	t.Helper()
	got, ok := gaugeValue(t, name, attribution)
	if !ok {
		t.Errorf("%s{attribution=%q} is missing", name, attribution)
		return
	}
	if got != want {
		t.Errorf("%s{attribution=%q} = %v, want %v", name, attribution, got, want)
	}
}

func expectNoGauge(t *testing.T, name, attribution string) {
	t.Helper()
	if got, ok := gaugeValue(t, name, attribution); ok {
		t.Errorf("%s{attribution=%q} = %v, want no series", name, attribution, got)
	}
}

func gaugeValue(t *testing.T, name, attribution string) (float64, bool) {
	t.Helper()
	families, err := crmetrics.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, m := range family.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			if labels["namespace"] == "team" && labels["attribution"] == attribution {
				return m.GetGauge().GetValue(), true
			}
		}
	}
	return 0, false
}
