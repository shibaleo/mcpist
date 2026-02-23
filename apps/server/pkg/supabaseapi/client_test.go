package supabaseapi_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"mcpist/server/pkg/supabaseapi"
	gen "mcpist/server/pkg/supabaseapi/gen"
)

// These tests call the real Supabase Management API.
// Set SUPABASE_ACCESS_TOKEN and SUPABASE_TEST_PROJECT_REF to run.
//
// Usage:
//   SUPABASE_ACCESS_TOKEN=sbp_xxx SUPABASE_TEST_PROJECT_REF=xxx go test ./pkg/supabaseapi/ -v -count=1

func skipIfNoToken(t *testing.T) string {
	t.Helper()
	token := os.Getenv("SUPABASE_ACCESS_TOKEN")
	if token == "" {
		t.Skip("SUPABASE_ACCESS_TOKEN not set")
	}
	return token
}

func skipIfNoProject(t *testing.T) string {
	t.Helper()
	ref := os.Getenv("SUPABASE_TEST_PROJECT_REF")
	if ref == "" {
		t.Skip("SUPABASE_TEST_PROJECT_REF not set")
	}
	return ref
}

func newClient(t *testing.T) *gen.Client {
	t.Helper()
	token := skipIfNoToken(t)
	c, err := supabaseapi.NewClient(token)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestListOrganizations(t *testing.T) {
	c := newClient(t)
	res, err := c.ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("ListOrganizations: %v", err)
	}
	fmt.Printf("Organizations: %d\n", len(res))
	for _, org := range res {
		fmt.Printf("  - %s (id=%s)\n", org.Name, org.ID)
	}
}

func TestListProjects(t *testing.T) {
	c := newClient(t)
	res, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	fmt.Printf("Projects: %d\n", len(res))
	for _, p := range res {
		fmt.Printf("  - %s (ref=%s, region=%s, status=%s)\n", p.Name, p.ID, p.Region, p.Status)
	}
}

func TestGetProject(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.GetProject(context.Background(), gen.GetProjectParams{Ref: ref})
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	fmt.Printf("Project: %s (ref=%s, region=%s, status=%s)\n", res.Name, res.ID, res.Region, res.Status)
}

func TestRunDatabaseQuery(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.RunDatabaseQuery(context.Background(),
		&gen.RunQueryRequest{Query: "SELECT 1 as ok"},
		gen.RunDatabaseQueryParams{Ref: ref},
	)
	if err != nil {
		t.Fatalf("RunDatabaseQuery: %v", err)
	}
	fmt.Printf("Query result: %s\n", string(res))
}

func TestGetLogs(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)

	// Test without timestamps first (select * is not allowed by Analytics API)
	p := gen.GetLogsParams{Ref: ref}
	p.SQL.SetTo("select id, timestamp, event_message from postgres_logs limit 5")

	res, err := c.GetLogs(context.Background(), p)
	if err != nil {
		t.Fatalf("GetLogs (no timestamps): %v", err)
	}
	fmt.Printf("Logs result count: %d (null=%v)\n", len(res.Result.Value), res.Result.Null)
}

func TestGetLogsWithTimestamps(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)

	p := gen.GetLogsParams{Ref: ref}
	p.SQL.SetTo("select id, timestamp, event_message from postgres_logs limit 5")
	p.IsoTimestampStart.SetTo("2026-02-11T00:00:00Z")
	p.IsoTimestampEnd.SetTo("2026-02-12T00:00:00Z")

	res, err := c.GetLogs(context.Background(), p)
	if err != nil {
		t.Fatalf("GetLogs with timestamps: %v", err)
	}
	fmt.Printf("Logs with timestamps result count: %d (null=%v)\n", len(res.Result.Value), res.Result.Null)
}

func TestGetSecurityAdvisors(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.GetSecurityAdvisors(context.Background(), gen.GetSecurityAdvisorsParams{Ref: ref})
	if err != nil {
		t.Fatalf("GetSecurityAdvisors: %v", err)
	}
	fmt.Printf("Security advisors: %d lints\n", len(res.Lints))
}

func TestGetPerformanceAdvisors(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.GetPerformanceAdvisors(context.Background(), gen.GetPerformanceAdvisorsParams{Ref: ref})
	if err != nil {
		t.Fatalf("GetPerformanceAdvisors: %v", err)
	}
	fmt.Printf("Performance advisors: %d lints\n", len(res.Lints))
}

func TestGetApiKeys(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.GetApiKeys(context.Background(), gen.GetApiKeysParams{Ref: ref})
	if err != nil {
		t.Fatalf("GetApiKeys: %v", err)
	}
	fmt.Printf("API Keys: %d\n", len(res))
	for _, k := range res {
		fmt.Printf("  - %s (type=%s)\n", k.Name, k.Type.Value)
	}
}

func TestListEdgeFunctions(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.ListEdgeFunctions(context.Background(), gen.ListEdgeFunctionsParams{Ref: ref})
	if err != nil {
		t.Fatalf("ListEdgeFunctions: %v", err)
	}
	fmt.Printf("Edge Functions: %d\n", len(res))
	for _, f := range res {
		fmt.Printf("  - %s (slug=%s)\n", f.Name, f.Slug)
	}
}

func TestGetEdgeFunction(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)

	// First list to get a slug
	funcs, err := c.ListEdgeFunctions(context.Background(), gen.ListEdgeFunctionsParams{Ref: ref})
	if err != nil {
		t.Fatalf("ListEdgeFunctions: %v", err)
	}
	if len(funcs) == 0 {
		t.Skip("No edge functions to test")
	}

	slug := funcs[0].Slug
	res, err := c.GetEdgeFunction(context.Background(), gen.GetEdgeFunctionParams{Ref: ref, Slug: slug})
	if err != nil {
		t.Fatalf("GetEdgeFunction: %v", err)
	}
	fmt.Printf("Edge Function: %s (slug=%s, status=%s)\n", res.Name, res.Slug, res.Status)
}

func TestListStorageBuckets(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.ListStorageBuckets(context.Background(), gen.ListStorageBucketsParams{Ref: ref})
	if err != nil {
		t.Fatalf("ListStorageBuckets: %v", err)
	}
	fmt.Printf("Storage Buckets: %d\n", len(res))
	for _, b := range res {
		fmt.Printf("  - %s (public=%v)\n", b.Name, b.Public)
	}
}

func TestGetStorageConfig(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	res, err := c.GetStorageConfig(context.Background(), gen.GetStorageConfigParams{Ref: ref})
	if err != nil {
		t.Fatalf("GetStorageConfig: %v", err)
	}
	fmt.Printf("Storage Config: fileSizeLimit=%d\n", res.FileSizeLimit)
}

// TestDescribeProjectAPIs tests all APIs used by describe_project composite tool in parallel
func TestDescribeProjectAPIs(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	ctx := context.Background()

	// Run all calls in parallel (same as describe_project handler)
	type result struct {
		key string
		err error
	}
	ch := make(chan result, 6)

	go func() { _, err := c.GetProject(ctx, gen.GetProjectParams{Ref: ref}); ch <- result{"getProject", err} }()
	go func() {
		_, err := c.RunDatabaseQuery(ctx, &gen.RunQueryRequest{Query: "SELECT schemaname, tablename FROM pg_tables WHERE schemaname='public' LIMIT 5"}, gen.RunDatabaseQueryParams{Ref: ref})
		ch <- result{"listTables", err}
	}()
	go func() { _, err := c.GetApiKeys(ctx, gen.GetApiKeysParams{Ref: ref}); ch <- result{"getApiKeys", err} }()
	go func() {
		_, err := c.ListEdgeFunctions(ctx, gen.ListEdgeFunctionsParams{Ref: ref})
		ch <- result{"listEdgeFunctions", err}
	}()
	go func() {
		_, err := c.ListStorageBuckets(ctx, gen.ListStorageBucketsParams{Ref: ref})
		ch <- result{"listStorageBuckets", err}
	}()
	go func() {
		_, err := c.GetStorageConfig(ctx, gen.GetStorageConfigParams{Ref: ref})
		ch <- result{"getStorageConfig", err}
	}()

	for i := 0; i < 6; i++ {
		r := <-ch
		if r.err != nil {
			t.Errorf("%s: %v", r.key, r.err)
		} else {
			fmt.Printf("  %s: OK\n", r.key)
		}
	}
}

// TestInspectHealthAPIs tests all APIs used by inspect_health composite tool in parallel
func TestInspectHealthAPIs(t *testing.T) {
	c := newClient(t)
	ref := skipIfNoProject(t)
	ctx := context.Background()

	type result struct {
		key   string
		count int
		err   error
	}
	ch := make(chan result, 2)

	go func() {
		res, err := c.GetSecurityAdvisors(ctx, gen.GetSecurityAdvisorsParams{Ref: ref})
		if err != nil {
			ch <- result{"security", 0, err}
		} else {
			ch <- result{"security", len(res.Lints), nil}
		}
	}()
	go func() {
		res, err := c.GetPerformanceAdvisors(ctx, gen.GetPerformanceAdvisorsParams{Ref: ref})
		if err != nil {
			ch <- result{"performance", 0, err}
		} else {
			ch <- result{"performance", len(res.Lints), nil}
		}
	}()

	for i := 0; i < 2; i++ {
		r := <-ch
		if r.err != nil {
			t.Errorf("%s: %v", r.key, r.err)
		} else {
			fmt.Printf("  %s: %d lints\n", r.key, r.count)
		}
	}
}
