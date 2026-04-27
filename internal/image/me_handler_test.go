package image

import "testing"

func TestToViewBuildsFreshProxyURLs(t *testing.T) {
	task := &Task{
		TaskID:     "img_test_123",
		ResultURLs: []byte(`["https://chatgpt.com/backend-api/estuary/content?id=old"]`),
		FileIDs:    []byte(`["sed:file_abc"]`),
	}

	view := toView(task, func(taskID string, idx int) string {
		return "/p/img/" + taskID + "/fresh-" + string(rune('0'+idx))
	})

	if len(view.ImageURLs) != 1 {
		t.Fatalf("expected 1 image url, got %d", len(view.ImageURLs))
	}
	if got := view.ImageURLs[0]; got != "/p/img/img_test_123/fresh-0" {
		t.Fatalf("expected rebuilt proxy url, got %q", got)
	}
	if len(view.FileIDs) != 1 || view.FileIDs[0] != "file_abc" {
		t.Fatalf("expected trimmed file id, got %#v", view.FileIDs)
	}
}
