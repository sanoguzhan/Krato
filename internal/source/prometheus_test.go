package source

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestPrometheusSourceSendsOneQueryPerResource(t *testing.T) {
	var mu sync.Mutex
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse request: %v", err)
		}
		query := r.Form.Get("query")
		mu.Lock()
		queries = append(queries, query)
		mu.Unlock()

		value := "250"
		if strings.Contains(query, "memory") {
			value = "1048576"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[
			{"metric":{"namespace":"a","pod":"p1"},"value":[1700000000,"%[1]s"]},
			{"metric":{"namespace":"b","pod":"p2"},"value":[1700000000,"%[1]s"]}]}}`, value)
	}))
	defer srv.Close()

	src, err := newPrometheusSource(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	pods := []corev1.Pod{testPod("a", "p1"), testPod("b", "p2"), testPod("a", "p3")}
	usage, err := src.PodUsage(context.Background(), pods)
	if err != nil {
		t.Fatal(err)
	}

	if len(queries) != 2 {
		t.Fatalf("got %d queries for %d pods, want 2: %q", len(queries), len(pods), queries)
	}
	for _, q := range queries {
		if !strings.Contains(q, `namespace=~"a|b"`) {
			t.Errorf("query %q is not scoped to namespaces a and b", q)
		}
	}

	want := map[types.NamespacedName]Usage{
		{Namespace: "a", Name: "p1"}: {CPUmilli: 250, MemoryBytes: 1048576},
		{Namespace: "b", Name: "p2"}: {CPUmilli: 250, MemoryBytes: 1048576},
	}
	if !reflect.DeepEqual(usage, want) {
		t.Errorf("got usage %v, want %v", usage, want)
	}
}

func TestPrometheusSourceSkipsQueriesWithoutPods(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected query %s", r.URL)
	}))
	defer srv.Close()

	src, err := newPrometheusSource(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	usage, err := src.PodUsage(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(usage) != 0 {
		t.Errorf("got usage %v, want none", usage)
	}
}

func testPod(namespace, name string) corev1.Pod {
	return corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
}
