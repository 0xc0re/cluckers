package config

import "testing"

func TestIsAppImage(t *testing.T) {
	t.Run("returns false when APPIMAGE not set", func(t *testing.T) {
		t.Setenv("APPIMAGE", "")
		if IsAppImage() {
			t.Error("expected IsAppImage() to return false when APPIMAGE is empty")
		}
	})

	t.Run("returns true when APPIMAGE is set", func(t *testing.T) {
		t.Setenv("APPIMAGE", "/path/to/Cluckers-x86_64.AppImage")
		if !IsAppImage() {
			t.Error("expected IsAppImage() to return true when APPIMAGE is set")
		}
	})
}

func TestAppImagePath(t *testing.T) {
	t.Run("returns empty when APPIMAGE not set", func(t *testing.T) {
		t.Setenv("APPIMAGE", "")
		if got := AppImagePath(); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns path when APPIMAGE is set", func(t *testing.T) {
		want := "/home/user/Cluckers-x86_64.AppImage"
		t.Setenv("APPIMAGE", want)
		if got := AppImagePath(); got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})
}

func TestAppDir(t *testing.T) {
	t.Run("returns empty when APPDIR not set", func(t *testing.T) {
		t.Setenv("APPDIR", "")
		if got := AppDir(); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns path when APPDIR is set", func(t *testing.T) {
		want := "/tmp/.mount_CluckXYZ123"
		t.Setenv("APPDIR", want)
		if got := AppDir(); got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})
}

func TestBundledProtonPath(t *testing.T) {
	t.Run("returns empty when CLUCKERS_BUNDLED_PROTON not set", func(t *testing.T) {
		t.Setenv("CLUCKERS_BUNDLED_PROTON", "")
		if got := BundledProtonPath(); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns path when CLUCKERS_BUNDLED_PROTON is set", func(t *testing.T) {
		want := "/tmp/.mount_CluckXYZ123/proton"
		t.Setenv("CLUCKERS_BUNDLED_PROTON", want)
		if got := BundledProtonPath(); got != want {
			t.Errorf("expected %q, got %q", want, got)
		}
	})
}
