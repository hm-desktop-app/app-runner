package launcher

import "testing"

// Preload 让窗口启动期间先完成一次扫描，首个 Scan 直接复用结果；
// 结果只能消费一次，否则刷新会看到过期的应用库。
func TestPreloadFeedsFirstScanOnce(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "first", "title = 'First'")
	s := NewStore(root)
	s.Preload()
	first := scan(t, s)
	if len(first.Apps) != 1 || first.Apps[0].Title != "First" {
		t.Fatalf("preloaded snapshot not reused: %+v", first)
	}
	fixture(t, root, "second", "title = 'Second'")
	second := scan(t, s)
	if len(second.Apps) != 2 {
		t.Fatalf("preloaded snapshot reused twice: %+v", second)
	}
}
