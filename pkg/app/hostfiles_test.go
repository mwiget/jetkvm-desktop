package app

import "testing"

func TestHostURLTarget(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{raw: "jetkvm://192.168.68.57", want: "192.168.68.57"},
		{raw: "jetkvm://jetkvm.local:8080/", want: "jetkvm.local:8080"},
		{raw: " JETKVM://kvm.example.com ", want: "kvm.example.com"},
		{raw: "http://192.168.68.57", wantErr: true},
		{raw: "jetkvm:///", wantErr: true},
	}
	for _, tt := range tests {
		got, err := hostURLTarget(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Fatalf("hostURLTarget(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
		}
		if got != tt.want {
			t.Fatalf("hostURLTarget(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestHostRequestQueueTakeDrains(t *testing.T) {
	var q hostRequestQueue
	q.addFile("/tmp/a.iso")
	q.addURL("jetkvm://kvm")
	files, urls := q.take()
	if len(files) != 1 || len(urls) != 1 {
		t.Fatalf("take = %v, %v", files, urls)
	}
	if files, urls = q.take(); len(files) != 0 || len(urls) != 0 {
		t.Fatalf("second take should be empty, got %v, %v", files, urls)
	}
}
