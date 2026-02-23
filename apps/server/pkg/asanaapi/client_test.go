package asanaapi_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"mcpist/server/pkg/asanaapi"
	gen "mcpist/server/pkg/asanaapi/gen"
)

// These tests call the real Asana API.
// Set ASANA_ACCESS_TOKEN to run. Optionally set ASANA_TEST_WORKSPACE_GID
// and ASANA_TEST_PROJECT_GID for tests that require them.
//
// Usage:
//   ASANA_ACCESS_TOKEN=xxx go test ./pkg/asanaapi/ -v -count=1

func skipIfNoToken(t *testing.T) string {
	t.Helper()
	token := os.Getenv("ASANA_ACCESS_TOKEN")
	if token == "" {
		t.Skip("ASANA_ACCESS_TOKEN not set")
	}
	return token
}

func skipIfNoWorkspace(t *testing.T) string {
	t.Helper()
	gid := os.Getenv("ASANA_TEST_WORKSPACE_GID")
	if gid == "" {
		t.Skip("ASANA_TEST_WORKSPACE_GID not set")
	}
	return gid
}

func skipIfNoProject(t *testing.T) string {
	t.Helper()
	gid := os.Getenv("ASANA_TEST_PROJECT_GID")
	if gid == "" {
		t.Skip("ASANA_TEST_PROJECT_GID not set")
	}
	return gid
}

func newClient(t *testing.T) *gen.Client {
	t.Helper()
	token := skipIfNoToken(t)
	c, err := asanaapi.NewClient(token)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetMe(t *testing.T) {
	c := newClient(t)
	res, err := c.GetMe(context.Background())
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	fmt.Printf("Me: %s (gid=%s, email=%s)\n", res.Data.Value.Name.Value, res.Data.Value.Gid.Value, res.Data.Value.Email.Value)
}

func TestListWorkspaces(t *testing.T) {
	c := newClient(t)
	res, err := c.ListWorkspaces(context.Background())
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	fmt.Printf("Workspaces: %d\n", len(res.Data))
	for _, w := range res.Data {
		fmt.Printf("  - %s (gid=%s, org=%v)\n", w.Name.Value, w.Gid.Value, w.IsOrganization.Value)
	}
}

func TestGetWorkspace(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoWorkspace(t)
	res, err := c.GetWorkspace(context.Background(), gen.GetWorkspaceParams{WorkspaceGid: gid})
	if err != nil {
		t.Fatalf("GetWorkspace: %v", err)
	}
	fmt.Printf("Workspace: %s (gid=%s)\n", res.Data.Value.Name.Value, res.Data.Value.Gid.Value)
}

func TestListProjectsByWorkspace(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoWorkspace(t)
	res, err := c.ListProjectsByWorkspace(context.Background(), gen.ListProjectsByWorkspaceParams{WorkspaceGid: gid})
	if err != nil {
		t.Fatalf("ListProjectsByWorkspace: %v", err)
	}
	fmt.Printf("Projects: %d\n", len(res.Data))
	for _, p := range res.Data {
		fmt.Printf("  - %s (gid=%s)\n", p.Name.Value, p.Gid.Value)
	}
}

func TestGetProject(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoProject(t)
	res, err := c.GetProject(context.Background(), gen.GetProjectParams{ProjectGid: gid})
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	d := res.Data.Value
	fmt.Printf("Project: %s (gid=%s, archived=%v, view=%s)\n", d.Name.Value, d.Gid.Value, d.Archived.Value, d.DefaultView.Value)
}

func TestListSections(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoProject(t)
	res, err := c.ListSections(context.Background(), gen.ListSectionsParams{ProjectGid: gid})
	if err != nil {
		t.Fatalf("ListSections: %v", err)
	}
	fmt.Printf("Sections: %d\n", len(res.Data))
	for _, s := range res.Data {
		fmt.Printf("  - %s (gid=%s)\n", s.Name.Value, s.Gid.Value)
	}
}

func TestListTasksByProject(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoProject(t)
	res, err := c.ListTasksByProject(context.Background(), gen.ListTasksByProjectParams{
		ProjectGid: gid,
		OptFields:  gen.NewOptString("name,completed,due_on,assignee.name"),
	})
	if err != nil {
		t.Fatalf("ListTasksByProject: %v", err)
	}
	fmt.Printf("Tasks: %d\n", len(res.Data))
	for _, task := range res.Data {
		fmt.Printf("  - %s (gid=%s, completed=%v)\n", task.Name.Value, task.Gid.Value, task.Completed.Value)
	}
}

func TestGetTask(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoProject(t)

	// List tasks first to get a task GID
	tasks, err := c.ListTasksByProject(context.Background(), gen.ListTasksByProjectParams{ProjectGid: gid})
	if err != nil {
		t.Fatalf("ListTasksByProject: %v", err)
	}
	if len(tasks.Data) == 0 {
		t.Skip("No tasks in project to test")
	}

	taskGID := tasks.Data[0].Gid.Value
	res, err := c.GetTask(context.Background(), gen.GetTaskParams{TaskGid: taskGID})
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	d := res.Data.Value
	fmt.Printf("Task: %s (gid=%s, completed=%v, due_on=%s)\n", d.Name.Value, d.Gid.Value, d.Completed.Value, d.DueOn.Value)
}

func TestListSubtasks(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoProject(t)

	// Find a task with subtasks
	tasks, err := c.ListTasksByProject(context.Background(), gen.ListTasksByProjectParams{ProjectGid: gid})
	if err != nil {
		t.Fatalf("ListTasksByProject: %v", err)
	}
	if len(tasks.Data) == 0 {
		t.Skip("No tasks in project")
	}

	taskGID := tasks.Data[0].Gid.Value
	res, err := c.ListSubtasks(context.Background(), gen.ListSubtasksParams{
		TaskGid:   taskGID,
		OptFields: gen.NewOptString("name,completed,due_on,assignee.name"),
	})
	if err != nil {
		t.Fatalf("ListSubtasks: %v", err)
	}
	fmt.Printf("Subtasks of %s: %d\n", taskGID, len(res.Data))
	for _, s := range res.Data {
		fmt.Printf("  - %s (gid=%s)\n", s.Name.Value, s.Gid.Value)
	}
}

func TestListStories(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoProject(t)

	tasks, err := c.ListTasksByProject(context.Background(), gen.ListTasksByProjectParams{ProjectGid: gid})
	if err != nil {
		t.Fatalf("ListTasksByProject: %v", err)
	}
	if len(tasks.Data) == 0 {
		t.Skip("No tasks in project")
	}

	taskGID := tasks.Data[0].Gid.Value
	res, err := c.ListStories(context.Background(), gen.ListStoriesParams{TaskGid: taskGID})
	if err != nil {
		t.Fatalf("ListStories: %v", err)
	}
	fmt.Printf("Stories of %s: %d\n", taskGID, len(res.Data))
	for _, s := range res.Data {
		fmt.Printf("  - [%s] %s (by=%s)\n", s.Type.Value, truncate(s.Text.Value, 60), s.CreatedBy.Value.Name.Value)
	}
}

func TestListTags(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoWorkspace(t)
	res, err := c.ListTags(context.Background(), gen.ListTagsParams{WorkspaceGid: gid})
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	fmt.Printf("Tags: %d\n", len(res.Data))
	for _, tag := range res.Data {
		fmt.Printf("  - %s (gid=%s, color=%s)\n", tag.Name.Value, tag.Gid.Value, tag.Color.Value)
	}
}

func TestSearchTasks(t *testing.T) {
	c := newClient(t)
	gid := skipIfNoWorkspace(t)
	res, err := c.SearchTasks(context.Background(), gen.SearchTasksParams{
		WorkspaceGid: gid,
		Completed:    gen.NewOptBool(false),
		OptFields:    gen.NewOptString("name,completed,due_on,assignee.name"),
	})
	if err != nil {
		t.Fatalf("SearchTasks: %v", err)
	}
	fmt.Printf("Search results (incomplete tasks): %d\n", len(res.Data))
	for i, task := range res.Data {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(res.Data)-5)
			break
		}
		fmt.Printf("  - %s (gid=%s)\n", task.Name.Value, task.Gid.Value)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
